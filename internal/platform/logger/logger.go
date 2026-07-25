// Package logger настраивает структурное логирование (log/slog) для всего приложения.
package logger

import (
	"log/slog"
	"os"
	"strings"
)

// New создаёт JSON slog.Logger с уровнем логирования, заданным строкой (debug/info/warn/error).
func New(level string, env string) *slog.Logger {
	handler := slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: parseLevel(level),
	})

	logger := slog.New(handler).With(slog.String("env", env))
	slog.SetDefault(logger)

	return logger
}

func parseLevel(level string) slog.Level {
	switch strings.ToLower(level) {
	case "debug":
		return slog.LevelDebug
	case "warn", "warning":
		return slog.LevelWarn
	case "error":
		return slog.LevelError
	default:
		return slog.LevelInfo
	}
}
