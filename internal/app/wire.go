package app

import (
	"context"
	"log/slog"
	"os"

	"cowpoke/internal/adapters/filesystem"
	"cowpoke/internal/adapters/terminal"
	"cowpoke/internal/services/config"
)

// NewAppWithConfig creates a new App with the given configuration, wiring all dependencies.
func NewAppWithConfig(ctx context.Context, cfg *Config) (*App, error) {
	// Create logger.
	loggerOpts := &slog.HandlerOptions{
		Level: cfg.LogLevel,
	}
	logger := slog.New(slog.NewTextHandler(os.Stderr, loggerOpts))

	// Create filesystem adapter.
	fs := filesystem.New()

	// Create Rancher service factory for on-demand service creation.
	rancherServiceFactory := NewRancherServiceFactory(logger, fs)

	// Create password reader with environment variable support.
	passwordReader := terminal.NewAdapter(os.Stdin, os.Stderr)

	// Create config services.
	configProvider := config.NewProvider(fs)
	configPath, err := configProvider.GetConfigPath()
	if err != nil {
		return nil, err
	}
	configRepo, err := config.NewRepository(fs, configPath, logger)
	if err != nil {
		return nil, err
	}

	// Log configuration details.
	logger.InfoContext(ctx, "Initializing cowpoke with configuration",
		"logLevel", cfg.LogLevel.String(),
		"verbose", cfg.Verbose,
		"configPath", configPath)

	// Note: kubeconfig manager is created on-demand by RancherServiceFactory

	return &App{
		ConfigRepo:            configRepo,
		ConfigProvider:        configProvider,
		RancherServiceFactory: rancherServiceFactory,
		PasswordReader:        passwordReader,
		FileSystem:            fs,
		Logger:                logger,
		Config:                cfg,
	}, nil
}
