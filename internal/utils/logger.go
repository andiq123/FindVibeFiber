package utils

import (
	"log/slog"
	"os"
)

var defaultLogger *slog.Logger

func init() {
	// Use JSON logger in production, text logger in development
	if os.Getenv("ENV") == "production" {
		defaultLogger = slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
			Level: slog.LevelInfo,
		}))
	} else {
		defaultLogger = slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
			Level: slog.LevelDebug,
		}))
	}
}

func GetLogger() *slog.Logger {
	return defaultLogger
}

func SetLogger(logger *slog.Logger) {
	defaultLogger = logger
}
