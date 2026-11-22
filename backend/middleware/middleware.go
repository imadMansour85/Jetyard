package middleware

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"runtime/debug"
	"time"

	"github.com/google/uuid"
)

// RequestIDHeader carries the request ID in and out.
const RequestIDHeader = "X-Request-Id"

type requestIDKey struct{}

// validRequestID accepts caller-supplied IDs that are safe to echo and log:
// 1-128 characters of letters, digits and . _ : -
func validRequestID(id string) bool {
	if id == "" || len(id) > 128 {
		return false
	}
	for _, c := range id {
		switch {
		case c >= 'a' && c <= 'z', c >= 'A' && c <= 'Z', c >= '0' && c <= '9', c == '.', c == '_', c == ':', c == '-':
		default:
			return false
		}
	}
	return true
}

// RequestID reuses the caller's X-Request-Id (e.g. from a reverse proxy) or
// generates one, stores it in the request context and sets it on the
// response. Handlers read it with RequestIDFrom or from the response header.
func RequestID(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		id := r.Header.Get(RequestIDHeader)
		if !validRequestID(id) {
			id = uuid.NewString()
		}
		w.Header().Set(RequestIDHeader, id)
		next.ServeHTTP(w, r.WithContext(context.WithValue(r.Context(), requestIDKey{}, id)))
	})
}

// RequestIDFrom returns the request ID stored by RequestID, or "".
func RequestIDFrom(ctx context.Context) string {
	id, _ := ctx.Value(requestIDKey{}).(string)
	return id
}

// redactedURI strips sensitive query parameters (e.g. the SSE bearer token
// passed as ?access_token=...) before a URL is written to logs.
func redactedURI(r *http.Request) string {
	if r.URL.RawQuery == "" {
		return r.RequestURI
	}

	q := r.URL.Query()
	if q.Has("access_token") {
		q.Set("access_token", "REDACTED")
	}

	u := *r.URL
	u.RawQuery = q.Encode()
	return u.RequestURI()
}

// statusRecorder captures the status code and size for the access log. It
// keeps Flush working (SSE needs it) and exposes the wrapped writer to
// http.ResponseController.
type statusRecorder struct {
	http.ResponseWriter
	status int
	bytes  int
}

func (s *statusRecorder) WriteHeader(code int) {
	if s.status == 0 {
		s.status = code
	}
	s.ResponseWriter.WriteHeader(code)
}

func (s *statusRecorder) Write(b []byte) (int, error) {
	if s.status == 0 {
		s.status = http.StatusOK
	}
	n, err := s.ResponseWriter.Write(b)
	s.bytes += n
	return n, err
}

func (s *statusRecorder) Flush() {
	if f, ok := s.ResponseWriter.(http.Flusher); ok {
		f.Flush()
	}
}

func (s *statusRecorder) Unwrap() http.ResponseWriter { return s.ResponseWriter }

// Logger writes one structured access-log line per request. Health checks
// are logged at debug level so they don't drown the log.
func Logger(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		rec := &statusRecorder{ResponseWriter: w}

		next.ServeHTTP(rec, r)

		status := rec.status
		if status == 0 {
			status = http.StatusOK
		}
		level := slog.LevelInfo
		switch {
		case status >= 500:
			level = slog.LevelError
		case r.URL.Path == "/health" || r.URL.Path == "/ready":
			level = slog.LevelDebug
		}
		slog.LogAttrs(r.Context(), level, "http request",
			slog.String("method", r.Method),
			slog.String("path", redactedURI(r)),
			slog.Int("status", status),
			slog.Int("bytes", rec.bytes),
			slog.Float64("duration_ms", float64(time.Since(start).Microseconds())/1000),
			slog.String("request_id", RequestIDFrom(r.Context())),
		)
	})
}

// Recover turns a panic in a handler into a 500 with the request ID, and
// logs the panic value and stack trace.
func Recover(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			rec := recover()
			if rec == nil {
				return
			}
			// net/http uses this sentinel to abort a response on purpose.
			if err, ok := rec.(error); ok && errors.Is(err, http.ErrAbortHandler) {
				panic(rec)
			}
			id := RequestIDFrom(r.Context())
			slog.Error("panic in handler",
				"request_id", id,
				"method", r.Method,
				"path", redactedURI(r),
				"panic", fmt.Sprint(rec),
				"stack", string(debug.Stack()),
			)
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusInternalServerError)
			json.NewEncoder(w).Encode(map[string]string{
				"error":      "internal server error",
				"request_id": id,
			})
		}()

		next.ServeHTTP(w, r)
	})
}
