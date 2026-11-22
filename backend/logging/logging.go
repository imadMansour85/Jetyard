// Package logging configures the process-wide slog logger.
package logging

import (
	"fmt"
	"io"
	"log/slog"
	"os"
	"strings"
)

// New builds a logger. format is "text" (default) or "json"; level is
// "debug", "info" (default), "warn" or "error".
func New(w io.Writer, format, level string) (*slog.Logger, error) {
	var lvl slog.Level
	switch strings.ToLower(strings.TrimSpace(level)) {
	case "", "info":
		lvl = slog.LevelInfo
	case "debug":
		lvl = slog.LevelDebug
	case "warn", "warning":
		lvl = slog.LevelWarn
	case "error":
		lvl = slog.LevelError
	default:
		return nil, fmt.Errorf("LOG_LEVEL must be debug, info, warn or error, got %q", level)
	}

	opts := &slog.HandlerOptions{Level: lvl}
	switch strings.ToLower(strings.TrimSpace(format)) {
	case "", "text":
		return slog.New(slog.NewTextHandler(w, opts)), nil
	case "json":
		return slog.New(slog.NewJSONHandler(w, opts)), nil
	default:
		return nil, fmt.Errorf("LOG_FORMAT must be text or json, got %q", format)
	}
}

// Setup installs the logger described by LOG_FORMAT and LOG_LEVEL as the
// default. Output from the standard log package goes through it too, at info
// level. An invalid setting falls back to the defaults and is reported.
func Setup() {
	logger, err := New(os.Stderr, os.Getenv("LOG_FORMAT"), os.Getenv("LOG_LEVEL"))
	if err != nil {
		logger, _ = New(os.Stderr, "", "")
		slog.SetDefault(logger)
		slog.Warn("invalid logging config, using text/info", "error", err)
		return
	}
	slog.SetDefault(logger)
}
