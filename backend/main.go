// SPDX-License-Identifier: AGPL-3.0-or-later

package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"log/slog"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/imadMansour85/Jetyard/backend/auth"
	"github.com/imadMansour85/Jetyard/backend/buildinfo"
	"github.com/imadMansour85/Jetyard/backend/config"
	"github.com/imadMansour85/Jetyard/backend/handlers"
	"github.com/imadMansour85/Jetyard/backend/logging"
	"github.com/imadMansour85/Jetyard/backend/middleware"
	"github.com/imadMansour85/Jetyard/backend/nats"
	"github.com/imadMansour85/Jetyard/backend/nkeys"
	"github.com/imadMansour85/Jetyard/backend/store"
	"github.com/imadMansour85/Jetyard/backend/web"
	"github.com/rs/cors"
)

// shutdownTimeout is how long in-flight requests get to finish on SIGTERM.
const shutdownTimeout = 10 * time.Second

// connectNATSWithRetry retries the initial NATS connection with backoff so a
// transient outage (e.g. the NATS host still coming up, or a private-network
// DNS blip right after a cold start) doesn't crash the backend and burn
// through its restart budget.
func connectNATSWithRetry(settings *config.NATSSettings) *nats.Client {
	const maxAttempts = 8
	backoff := 1 * time.Second

	for attempt := 1; attempt <= maxAttempts; attempt++ {
		nc, err := nats.NewClientWithSettings(settings)
		if err == nil {
			return nc
		}

		if attempt == maxAttempts {
			log.Fatalf("Failed to connect to NATS after %d attempts: %v", maxAttempts, err)
		}

		log.Printf("Failed to connect to NATS (attempt %d/%d): %v — retrying in %s", attempt, maxAttempts, err, backoff)
		time.Sleep(backoff)
		if backoff < 30*time.Second {
			backoff *= 2
		}
	}

	return nil // unreachable
}

// runHealthcheck implements `jetyard healthcheck`, for Docker
// HEALTHCHECK in images that have no shell or curl. Exit 0 when /health
// answers 200 on the local port.
func runHealthcheck() {
	port := os.Getenv("SERVER_PORT")
	if port == "" {
		port = "8080"
	}
	client := &http.Client{Timeout: 3 * time.Second}
	resp, err := client.Get("http://127.0.0.1:" + port + "/health")
	if err != nil {
		fmt.Fprintln(os.Stderr, "healthcheck:", err)
		os.Exit(1)
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		fmt.Fprintln(os.Stderr, "healthcheck: status", resp.Status)
		os.Exit(1)
	}
}

func main() {
	if len(os.Args) > 1 && os.Args[1] == "healthcheck" {
		runHealthcheck()
		return
	}

	logging.Setup()
	bi := buildinfo.Get()
	slog.Info("starting jetyard backend", "version", bi.Version, "commit", bi.Commit, "build_time", bi.BuildTime)

	// Load configuration
	cfg := config.Load()

	// Load NATS settings (persisted file takes priority over env vars)
	natsSettings := config.LoadNATSSettings(cfg)

	// Connect to NATS, retrying with backoff before giving up
	nc := connectNATSWithRetry(natsSettings)

	// Create handler
	h := handlers.NewHandler(nc)

	// Create KV handler
	kvHandler := handlers.NewKVHandler(nc)

	// Create settings handler
	settingsHandler := handlers.NewSettingsHandler(natsSettings)

	// Dashboard authentication
	signer, persistent, err := auth.NewSigner()
	if err != nil {
		log.Fatalf("Auth setup failed: %v", err)
	}
	if !persistent {
		log.Println("WARNING: JWT_SECRET not set - using an ephemeral secret; all sessions end on restart")
	}

	authStore, err := store.NewAuthStore()
	if err != nil {
		log.Fatalf("Failed to load dashboard users: %v", err)
	}
	if authStore.Count() == 0 {
		log.Println("No dashboard users yet - POST /api/auth/register to create the first admin")
	}
	authHandler := handlers.NewAuthHandler(authStore, signer)

	// Initialize operator manager (optional, for account/user management)
	var accountHandler *handlers.AccountHandler
	var userHandler *handlers.UserHandler

	operatorMgr, err := nkeys.GetOperatorManager()
	if err != nil {
		log.Printf("Operator not configured: %v. Account/user management disabled", err)
	} else {
		// Create store
		st, err := store.NewStore()
		if err != nil {
			log.Fatalf("Failed to load account/user store: %v", err)
		}

		// Create handlers
		accountHandler = handlers.NewAccountHandler(st, operatorMgr)
		userHandler = handlers.NewUserHandler(st)
		log.Println("Account and user management enabled")
	}

	// Public routes: health and the auth endpoints needed to obtain a token.
	root := http.NewServeMux()

	root.HandleFunc("GET /health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	})

	root.HandleFunc("GET /version", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(bi)
	})

	root.HandleFunc("GET /api/auth/setup", authHandler.SetupState)
	root.HandleFunc("POST /api/auth/register", authHandler.Register)
	root.HandleFunc("POST /api/auth/login", authHandler.Login)

	// Everything below requires a valid bearer token.
	mux := http.NewServeMux()

	mux.HandleFunc("GET /api/auth/me", authHandler.Me)
	mux.HandleFunc("POST /api/auth/change-password", authHandler.ChangePassword)

	// Dashboard user administration
	mux.Handle("GET /api/auth/users", auth.RequireAdmin(http.HandlerFunc(authHandler.ListUsers)))
	mux.Handle("POST /api/auth/users", auth.RequireAdmin(http.HandlerFunc(authHandler.CreateUser)))
	mux.Handle("DELETE /api/auth/users/{username}", auth.RequireAdmin(http.HandlerFunc(authHandler.DeleteUser)))

	// Stats
	mux.HandleFunc("GET /api/stats", h.GetStats)
	mux.HandleFunc("GET /api/server", h.GetServerInfo)
	mux.HandleFunc("GET /api/connections", h.ListConnections)

	// Streams
	mux.HandleFunc("GET /api/streams", h.ListStreams)
	mux.HandleFunc("GET /api/streams/{name}", h.GetStream)
	mux.Handle("POST /api/streams", auth.RequireAdmin(http.HandlerFunc(h.CreateStream)))
	mux.Handle("DELETE /api/streams/{name}", auth.RequireAdmin(http.HandlerFunc(h.DeleteStream)))

	// Consumers
	mux.HandleFunc("GET /api/consumers", h.ListConsumers)
	mux.Handle("POST /api/consumers", auth.RequireAdmin(http.HandlerFunc(h.CreateConsumer)))
	mux.Handle("DELETE /api/consumers/{name}", auth.RequireAdmin(http.HandlerFunc(h.DeleteConsumer)))

	// Messages
	mux.Handle("POST /api/publish", auth.RequireAdmin(http.HandlerFunc(h.PublishMessage)))
	mux.HandleFunc("GET /api/messages/{stream}", h.GetMessages)

	// Message Browser
	mux.HandleFunc("GET /api/streams/{stream}/messages", h.ListStreamMessages)
	mux.HandleFunc("GET /api/streams/{stream}/messages/{seq}", h.GetMessageBySeq)

	// SSE
	mux.HandleFunc("GET /api/events", h.SSEHandler)

	// Accounts (if operator configured)
	if accountHandler != nil {
		mux.HandleFunc("GET /api/accounts", accountHandler.ListAccounts)
		mux.Handle("POST /api/accounts", auth.RequireAdmin(http.HandlerFunc(accountHandler.CreateAccount)))
		mux.HandleFunc("GET /api/accounts/{name}", accountHandler.GetAccount)
		mux.Handle("DELETE /api/accounts/{name}", auth.RequireAdmin(http.HandlerFunc(accountHandler.DeleteAccount)))
		mux.HandleFunc("GET /api/accounts/{name}/jwt", accountHandler.GetAccountJWT)
	}

	// Users (if operator configured)
	if userHandler != nil {
		mux.HandleFunc("GET /api/users", userHandler.ListUsers)
		mux.Handle("POST /api/users", auth.RequireAdmin(http.HandlerFunc(userHandler.CreateUser)))
		mux.HandleFunc("GET /api/users/{name}", userHandler.GetUser)
		mux.Handle("DELETE /api/users/{name}", auth.RequireAdmin(http.HandlerFunc(userHandler.DeleteUser)))
		mux.Handle("PUT /api/users/{name}/permissions", auth.RequireAdmin(http.HandlerFunc(userHandler.UpdatePermissions)))
		mux.HandleFunc("GET /api/users/{name}/credentials", userHandler.DownloadCredentials)
	}

	// NATS Settings
	mux.HandleFunc("GET /api/settings/nats", settingsHandler.GetNATSSettings)
	mux.Handle("POST /api/settings/nats/test", auth.RequireAdmin(http.HandlerFunc(settingsHandler.TestNATSSettings)))
	mux.Handle("POST /api/settings/nats/save", auth.RequireAdmin(http.HandlerFunc(settingsHandler.SaveNATSSettings)))

	// KV Store
	mux.HandleFunc("GET /api/kv/buckets", kvHandler.ListBuckets)
	mux.Handle("POST /api/kv/buckets", auth.RequireAdmin(http.HandlerFunc(kvHandler.CreateBucket)))
	mux.HandleFunc("GET /api/kv/buckets/{name}", kvHandler.GetBucket)
	mux.Handle("DELETE /api/kv/buckets/{name}", auth.RequireAdmin(http.HandlerFunc(kvHandler.DeleteBucket)))
	mux.HandleFunc("GET /api/kv/buckets/{bucket}/keys", kvHandler.ListKeys)
	mux.HandleFunc("GET /api/kv/buckets/{bucket}/keys/{key}", kvHandler.GetKey)
	mux.Handle("PUT /api/kv/buckets/{bucket}/keys/{key}", auth.RequireAdmin(http.HandlerFunc(kvHandler.PutKey)))
	mux.Handle("DELETE /api/kv/buckets/{bucket}/keys/{key}", auth.RequireAdmin(http.HandlerFunc(kvHandler.DeleteKey)))

	// Mount the authenticated mux under the public one.
	root.Handle("/api/", signer.Require(mux))

	// Single-image mode: serve the statically exported frontend from the same
	// origin, so no CORS or NEXT_PUBLIC_API_URL is needed.
	if dir := os.Getenv("STATIC_DIR"); dir != "" {
		if !web.Available(dir) {
			slog.Error("STATIC_DIR has no index.html, not serving the frontend", "dir", dir)
		} else {
			root.Handle("/", web.Handler(dir))
			slog.Info("serving frontend", "dir", dir)
		}
	}

	// Apply middleware. RequestID runs first so the access log and panic
	// recovery can include the ID; Recover is innermost so a panic is logged
	// as a 500 by Logger.
	handler := middleware.RequestID(middleware.Logger(middleware.Recover(root)))

	// Setup CORS
	c := cors.New(cors.Options{
		AllowedOrigins:   cfg.AllowedOrigins,
		AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"*"},
		ExposedHeaders:   []string{middleware.RequestIDHeader},
		AllowCredentials: true,
	})

	handlerWithCORS := c.Handler(handler)

	// Every request context derives from baseCtx. Cancelling it on shutdown
	// ends the long-lived SSE streams, which would otherwise keep Shutdown
	// waiting for the full timeout.
	baseCtx, cancelBase := context.WithCancel(context.Background())
	defer cancelBase()

	server := &http.Server{
		Addr:              ":" + cfg.ServerPort,
		Handler:           handlerWithCORS,
		ReadHeaderTimeout: 10 * time.Second,
		BaseContext:       func(net.Listener) context.Context { return baseCtx },
	}
	server.RegisterOnShutdown(cancelBase)

	sigCtx, stopSignals := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stopSignals()

	serveErr := make(chan error, 1)
	go func() {
		slog.Info("server listening", "port", cfg.ServerPort)
		serveErr <- server.ListenAndServe()
	}()

	select {
	case err := <-serveErr:
		if err != nil && !errors.Is(err, http.ErrServerClosed) {
			slog.Error("server failed", "error", err)
			nc.Close()
			os.Exit(1)
		}
	case <-sigCtx.Done():
		stopSignals() // a second Ctrl-C kills the process immediately
		slog.Info("shutdown signal received, draining requests", "timeout", shutdownTimeout.String())

		ctx, cancel := context.WithTimeout(context.Background(), shutdownTimeout)
		defer cancel()
		if err := server.Shutdown(ctx); err != nil {
			slog.Warn("graceful shutdown timed out, closing remaining connections", "error", err)
			server.Close()
		}
	}

	// HTTP is drained, so nothing uses the NATS connection any more.
	nc.Close()
	slog.Info("server stopped")
}
