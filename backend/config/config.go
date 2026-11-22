package config

import (
	"log"
	"os"
	"strings"
)

type Config struct {
	NatsURL        string
	ServerPort     string
	AllowedOrigins []string
	OperatorSeed   string
}

// parseOrigins splits a comma-separated origin list, trimming spaces (an
// origin like " https://x" never matches the browser's Origin header).
func parseOrigins(raw string) []string {
	var origins []string
	for _, o := range strings.Split(raw, ",") {
		if o = strings.TrimSpace(o); o != "" {
			origins = append(origins, o)
		}
	}
	return origins
}

func Load() *Config {
	natsURL := os.Getenv("NATS_URL")
	if natsURL == "" {
		natsURL = "nats://localhost:4222"
	}

	serverPort := os.Getenv("SERVER_PORT")
	if serverPort == "" {
		serverPort = "8080"
	}

	allowedOrigins := os.Getenv("ALLOWED_ORIGINS")
	if allowedOrigins == "" {
		allowedOrigins = "http://localhost:3000"
	}

	operatorSeed := os.Getenv("OPERATOR_NKEY_SEED")
	if operatorSeed == "" {
		log.Println("WARNING: OPERATOR_NKEY_SEED not set - account/user management will not work")
	}

	return &Config{
		NatsURL:        natsURL,
		ServerPort:     serverPort,
		AllowedOrigins: parseOrigins(allowedOrigins),
		OperatorSeed:   operatorSeed,
	}
}
