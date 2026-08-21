package logger

import (
	"log/slog"
	"os"
	"strings"
)

// New returns a JSON slog logger. Production default (info) hides debug.
func New(level string) *slog.Logger {
	var lv slog.Level
	switch strings.ToLower(strings.TrimSpace(level)) {
	case "debug":
		lv = slog.LevelDebug
	case "warn", "warning":
		lv = slog.LevelWarn
	case "error":
		lv = slog.LevelError
	default:
		lv = slog.LevelInfo
	}
	h := slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: lv})
	return slog.New(h)
}

func EnvLevel() string {
	if v := os.Getenv("GORHINO_LOG_LEVEL"); v != "" {
		return v
	}
	return "info"
}
