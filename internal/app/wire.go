package app

import (
	"context"
	"os"
	"time"

	"cowpoke/internal/adapters/filesystem"
	"cowpoke/internal/adapters/http"
	"cowpoke/internal/adapters/terminal"
	"cowpoke/internal/logging"
	"cowpoke/internal/services/config"
	"cowpoke/internal/services/kubeconfig"
	"cowpoke/internal/services/rancher"
	"cowpoke/internal/services/sync"
)

const (
	// defaultHTTPTimeout is the default timeout for HTTP requests to Rancher.
	defaultHTTPTimeout = 30 * time.Second
)

// NewAppWithConfig creates a new App with the given configuration, wiring all dependencies.
func NewAppWithConfig(ctx context.Context, cfg *Config) (*App, error) {
	// Create logger.
	logger := logging.NewLogger(cfg.LogLevel)

	// Create filesystem adapter.
	fs := filesystem.New()

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

	// Create kubeconfig directory and handler.
	kubeconfigDir, err := configProvider.GetKubeconfigDir()
	if err != nil {
		return nil, err
	}
	kubeconfigHandler, err := kubeconfig.NewHandler(fs, kubeconfigDir, logger)
	if err != nil {
		return nil, err
	}

	// Log configuration details.
	logger.InfoContext(ctx, "Initializing cowpoke with configuration",
		"logLevel", cfg.LogLevel.String(),
		"verbose", cfg.Verbose,
		"configPath", configPath)

	// Note: RancherClient and SyncOrchestrator will be created on-demand with appropriate TLS settings.

	return &App{
		ConfigRepo:        configRepo,
		ConfigProvider:    configProvider,
		KubeconfigHandler: kubeconfigHandler,
		PasswordReader:    passwordReader,
		FileSystem:        fs,
		Logger:            logger,
		Config:            cfg,
	}, nil
}

// CreateRancherClient creates a rancher client with the specified TLS configuration.
func (app *App) CreateRancherClient(insecureSkipTLS bool) *rancher.Client {
	httpAdapter := http.NewAdapter(defaultHTTPTimeout, insecureSkipTLS, app.Logger)
	return rancher.NewClient(httpAdapter, app.Logger)
}

// CreateSyncOrchestrator creates a sync orchestrator with the given rancher client.
func (app *App) CreateSyncOrchestrator(rancherClient *rancher.Client) *sync.Orchestrator {
	return sync.NewOrchestrator(rancherClient, app.KubeconfigHandler, app.ConfigProvider, app.Logger)
}
