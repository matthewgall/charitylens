package logger

import (
	"context"
	"log/slog"
	"os"
)

var defaultLogger *slog.Logger

func init() {
	// Initialize default logger
	defaultLogger = slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}))
}

// SetLogger sets the global logger
func SetLogger(l *slog.Logger) {
	defaultLogger = l
}

// WithDebug returns a logger configured for debug mode
func WithDebug(debug bool) *slog.Logger {
	level := slog.LevelInfo
	if debug {
		level = slog.LevelDebug
	}

	handler := slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		Level: level,
	})

	logger := slog.New(handler)
	SetLogger(logger)
	return logger
}

// Info logs an info message
func Info(msg string, args ...any) {
	defaultLogger.Info(msg, args...)
}

// Error logs an error message
func Error(msg string, args ...any) {
	defaultLogger.Error(msg, args...)
}

// Debug logs a debug message
func Debug(msg string, args ...any) {
	defaultLogger.Debug(msg, args...)
}

// Warn logs a warning message
func Warn(msg string, args ...any) {
	defaultLogger.Warn(msg, args...)
}

// InfoContext logs an info message with context
func InfoContext(ctx context.Context, msg string, args ...any) {
	defaultLogger.InfoContext(ctx, msg, args...)
}

// ErrorContext logs an error message with context
func ErrorContext(ctx context.Context, msg string, args ...any) {
	defaultLogger.ErrorContext(ctx, msg, args...)
}

// With returns a logger with additional attributes
func With(args ...any) *slog.Logger {
	return defaultLogger.With(args...)
}
