// Package logger configures an application-wide structured logger backed by
// the standard library's log/slog.
//
// Callers should use the slog.Logger returned from New() (or the global
// slog.Default() after calling Init()) so log output is machine-parseable in
// production (JSON) and human-readable in development (text).
package logger

import (
	"io"
	"log/slog"
	"os"
	"strings"
)

// Config controls logger construction.
type Config struct {
	Level  string // debug, info, warn, error
	Format string // json, text
	Output io.Writer
}

// New returns a slog.Logger configured per cfg. An unset Output defaults to
// os.Stdout. Unknown Level or Format values fall back to info/json.
func New(cfg Config) *slog.Logger {
	if cfg.Output == nil {
		cfg.Output = os.Stdout
	}

	opts := &slog.HandlerOptions{Level: parseLevel(cfg.Level)}

	var handler slog.Handler
	switch strings.ToLower(cfg.Format) {
	case "text":
		handler = slog.NewTextHandler(cfg.Output, opts)
	default: // json and any unknown value
		handler = slog.NewJSONHandler(cfg.Output, opts)
	}

	return slog.New(handler)
}

// Init constructs a logger from cfg and installs it as slog.Default().
// Returns the configured logger for convenience.
func Init(cfg Config) *slog.Logger {
	l := New(cfg)
	slog.SetDefault(l)
	return l
}

func parseLevel(s string) slog.Leveler {
	switch strings.ToLower(s) {
	case "debug":
		return slog.LevelDebug
	case "warn", "warning":
		return slog.LevelWarn
	case "error", "err":
		return slog.LevelError
	default:
		return slog.LevelInfo
	}
}
