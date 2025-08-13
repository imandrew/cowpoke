package logging

import (
	"log/slog"
	"os"
)

var logger *slog.Logger

func init() {
	// Default to info level, text output to stderr
	logger = slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}))
}

// Get returns the global logger
func Get() *slog.Logger {
	return logger
}

// SetVerbose enables or disables verbose (debug) logging
func SetVerbose(verbose bool) {
	level := slog.LevelInfo
	if verbose {
		level = slog.LevelDebug
	}

	logger = slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{
		Level: level,
	}))
}

// Default returns the global logger (backwards compatibility during migration)
func Default() *slog.Logger {
	return logger
}
