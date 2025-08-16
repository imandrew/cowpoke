package app

import (
	"context"
	"log/slog"

	"cowpoke/internal/domain"
)

// App contains all application dependencies.
type App struct {
	// Core configuration dependencies (always needed)
	ConfigRepo     domain.ConfigRepository
	ConfigProvider domain.ConfigProvider

	// Factories for creating services on-demand
	RancherServiceFactory domain.RancherServiceFactory

	// File operations (needed by multiple commands)
	FileSystem domain.FileSystemAdapter

	// I/O dependencies
	PasswordReader domain.PasswordReader

	// Logging
	Logger *slog.Logger

	// Configuration
	Config *Config
}

// Config holds application configuration.
type Config struct {
	LogLevel slog.Level
	Verbose  bool
}

// Option is a functional option for configuring the App.
type Option func(*Config)

// WithLogLevel sets the logging level.
func WithLogLevel(level slog.Level) Option {
	return func(cfg *Config) {
		cfg.LogLevel = level
	}
}

// WithVerbose enables verbose logging.
func WithVerbose(verbose bool) Option {
	return func(cfg *Config) {
		cfg.Verbose = verbose
		if verbose {
			cfg.LogLevel = slog.LevelDebug
		}
	}
}

// NewApp creates a new App with the given options.
func NewApp(ctx context.Context, opts ...Option) (*App, error) {
	cfg := &Config{
		LogLevel: slog.LevelInfo,
		Verbose:  false,
	}

	// Apply options.
	for _, opt := range opts {
		opt(cfg)
	}

	return NewAppWithConfig(ctx, cfg)
}
