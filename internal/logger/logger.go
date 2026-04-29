package logger

import (
	"log/slog"
	"os"
	"strings"
)

// Config — параметры логирования (можно вынести в config.Config позже)
type Config struct {
	Level  string // debug, info, warn, error
	Format string // text или json
}

// New создаёт настроенный *slog.Logger
func New(cfg Config) *slog.Logger {
	var level slog.Level
	switch strings.ToLower(cfg.Level) {
	case "debug":
		level = slog.LevelDebug
	case "info":
		level = slog.LevelInfo
	case "warn":
		level = slog.LevelWarn
	case "error":
		level = slog.LevelError
	default:
		level = slog.LevelInfo
	}

	opts := &slog.HandlerOptions{
		Level: level,
	}

	var handler slog.Handler
	if cfg.Format == "json" {
		handler = slog.NewJSONHandler(os.Stdout, opts)
	} else {
		handler = slog.NewTextHandler(os.Stdout, opts)
	}

	return slog.New(handler)
}
