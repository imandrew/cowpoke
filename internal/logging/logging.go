package logging

import (
	"context"
	"io"
	"log/slog"
	"os"
)

// Logger wraps slog.Logger with application-specific functionality
type Logger struct {
	*slog.Logger
}

// LogLevel represents the logging level
type LogLevel string

const (
	LevelDebug LogLevel = "debug"
	LevelInfo  LogLevel = "info"
	LevelWarn  LogLevel = "warn"
	LevelError LogLevel = "error"
)

// Config holds logger configuration
type Config struct {
	Level  LogLevel
	Format string // "json" or "text"
	Output io.Writer
}

// DefaultConfig returns a default logger configuration
func DefaultConfig() Config {
	return Config{
		Level:  LevelInfo,
		Format: "text",
		Output: os.Stderr,
	}
}

// NewLogger creates a new structured logger
func NewLogger(config Config) *Logger {
	var level slog.Level
	switch config.Level {
	case LevelDebug:
		level = slog.LevelDebug
	case LevelInfo:
		level = slog.LevelInfo
	case LevelWarn:
		level = slog.LevelWarn
	case LevelError:
		level = slog.LevelError
	default:
		level = slog.LevelInfo
	}

	opts := &slog.HandlerOptions{
		Level: level,
	}

	var handler slog.Handler
	if config.Format == "json" {
		handler = slog.NewJSONHandler(config.Output, opts)
	} else {
		handler = slog.NewTextHandler(config.Output, opts)
	}

	return &Logger{
		Logger: slog.New(handler),
	}
}

// WithContext adds context to the logger
func (l *Logger) WithContext(ctx context.Context) *Logger {
	return &Logger{
		Logger: l.With(),
	}
}

// WithFields adds structured fields to the logger
func (l *Logger) WithFields(fields map[string]any) *Logger {
	args := make([]any, 0, len(fields)*2)
	for k, v := range fields {
		args = append(args, k, v)
	}
	return &Logger{
		Logger: l.With(args...),
	}
}

// WithServer adds server-related fields to the logger
func (l *Logger) WithServer(serverURL, authType string) *Logger {
	return l.WithFields(map[string]any{
		"server_url": serverURL,
		"auth_type":  authType,
	})
}

// WithCluster adds cluster-related fields to the logger
func (l *Logger) WithCluster(clusterID, clusterName string) *Logger {
	return l.WithFields(map[string]any{
		"cluster_id":   clusterID,
		"cluster_name": clusterName,
	})
}

// WithOperation adds operation-related fields to the logger
func (l *Logger) WithOperation(operation string) *Logger {
	return l.WithFields(map[string]any{
		"operation": operation,
	})
}

// Global logger instance
var defaultLogger = NewLogger(DefaultConfig())

// SetDefault sets the default logger
func SetDefault(logger *Logger) {
	defaultLogger = logger
}

// Default returns the default logger
func Default() *Logger {
	return defaultLogger
}

// Convenience functions using the default logger
func Debug(msg string, args ...any) {
	defaultLogger.Debug(msg, args...)
}

func Info(msg string, args ...any) {
	defaultLogger.Info(msg, args...)
}

func Warn(msg string, args ...any) {
	defaultLogger.Warn(msg, args...)
}

func Error(msg string, args ...any) {
	defaultLogger.Error(msg, args...)
}

func DebugContext(ctx context.Context, msg string, args ...any) {
	defaultLogger.DebugContext(ctx, msg, args...)
}

func InfoContext(ctx context.Context, msg string, args ...any) {
	defaultLogger.InfoContext(ctx, msg, args...)
}

func WarnContext(ctx context.Context, msg string, args ...any) {
	defaultLogger.WarnContext(ctx, msg, args...)
}

func ErrorContext(ctx context.Context, msg string, args ...any) {
	defaultLogger.ErrorContext(ctx, msg, args...)
}