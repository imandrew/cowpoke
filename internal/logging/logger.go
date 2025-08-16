// Package logging provides centralized logger creation for the cowpoke application.
package logging

import (
	"io"
	"log/slog"
	"os"
)

// NewLogger creates a standard text logger for CLI usage.
func NewLogger(level slog.Level) *slog.Logger {
	opts := &slog.HandlerOptions{
		Level: level,
	}
	return slog.New(slog.NewTextHandler(os.Stderr, opts))
}

// NewTestLogger creates a silent logger for tests.
func NewTestLogger() *slog.Logger {
	opts := &slog.HandlerOptions{
		Level: slog.LevelError + 1, // Higher than any real level = silent
	}
	return slog.New(slog.NewTextHandler(io.Discard, opts))
}
