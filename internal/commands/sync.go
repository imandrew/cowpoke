package commands

import (
	"context"
	"errors"
	"fmt"
	"log/slog"

	"cowpoke/internal/domain"
	"cowpoke/internal/services/filter"
)

// SyncCommand handles syncing kubeconfigs from Rancher servers.
type SyncCommand struct {
	configRepo     domain.ConfigRepository
	configProvider domain.ConfigProvider
	passwordReader domain.PasswordReader
	logger         *slog.Logger
}

// NewSyncCommand creates a new sync command.
func NewSyncCommand(
	configRepo domain.ConfigRepository,
	configProvider domain.ConfigProvider,
	passwordReader domain.PasswordReader,
	logger *slog.Logger,
) *SyncCommand {
	return &SyncCommand{
		configRepo:     configRepo,
		configProvider: configProvider,
		passwordReader: passwordReader,
		logger:         logger,
	}
}

// SyncRequest contains the parameters for the sync command.
type SyncRequest struct {
	Output           string
	InsecureSkipTLS  bool
	CleanupTempFiles bool
	Verbose          bool
	ExcludePatterns  []string
}

// Execute runs the sync command using the SyncOrchestrator for concurrent processing.
func (c *SyncCommand) Execute(
	ctx context.Context,
	req SyncRequest,
	syncOrchestrator domain.SyncOrchestrator,
	kubeconfigHandler domain.KubeconfigHandler,
) error {
	servers, err := c.configRepo.GetServers(ctx)
	if err != nil {
		return fmt.Errorf("failed to get servers: %w", err)
	}

	if len(servers) == 0 {
		c.logger.InfoContext(ctx, "No servers configured")
		return nil
	}

	c.logger.InfoContext(ctx, "Starting concurrent sync for servers", "count", len(servers))

	// Collect passwords for all servers upfront
	passwords, err := c.collectPasswords(ctx, servers)
	if err != nil {
		return fmt.Errorf("failed to collect passwords: %w", err)
	}

	// Create cluster filter based on exclude patterns
	var clusterFilter domain.ClusterFilter
	if len(req.ExcludePatterns) > 0 {
		excludeFilter, filterErr := filter.NewExcludeFilter(req.ExcludePatterns, c.logger)
		if filterErr != nil {
			return fmt.Errorf("failed to create exclude filter: %w", filterErr)
		}
		clusterFilter = excludeFilter
	} else {
		clusterFilter = filter.NewNoOpFilter()
	}

	// Use SyncOrchestrator for concurrent processing
	allKubeconfigPaths, err := syncOrchestrator.SyncServers(ctx, servers, passwords, clusterFilter)
	if err != nil {
		return fmt.Errorf("concurrent sync failed: %w", err)
	}

	if len(allKubeconfigPaths) == 0 {
		return errors.New("no kubeconfigs downloaded successfully")
	}

	// Determine output path
	outputPath := req.Output
	if outputPath == "" {
		outputPath, err = c.configProvider.GetDefaultKubeconfigPath()
		if err != nil {
			return fmt.Errorf("failed to get default kubeconfig path: %w", err)
		}
	}

	// Merge all kubeconfigs
	c.logger.InfoContext(ctx, "Merging kubeconfigs",
		"count", len(allKubeconfigPaths),
		"output", outputPath)

	if mergeErr := kubeconfigHandler.MergeKubeconfigs(ctx, allKubeconfigPaths, outputPath); mergeErr != nil {
		return fmt.Errorf("failed to merge kubeconfigs: %w", mergeErr)
	}

	// Cleanup temporary files if requested
	if req.CleanupTempFiles {
		if cleanupErr := kubeconfigHandler.CleanupTempFiles(ctx, allKubeconfigPaths); cleanupErr != nil {
			c.logger.WarnContext(ctx, "Failed to cleanup some temporary files", "error", cleanupErr)
		}
	}

	c.logger.InfoContext(ctx, "Concurrent sync completed successfully",
		"servers", len(servers),
		"kubeconfigs", len(allKubeconfigPaths),
		"output", outputPath)

	return nil
}

// collectPasswords prompts for passwords for all servers upfront.
func (c *SyncCommand) collectPasswords(ctx context.Context, servers []domain.ConfigServer) (map[string]string, error) {
	passwords := make(map[string]string)

	c.logger.InfoContext(ctx, "Collecting passwords for all servers")

	for _, server := range servers {
		password, err := c.passwordReader.ReadPassword(ctx,
			fmt.Sprintf("Password for %s: ", server.URL))
		if err != nil {
			return nil, fmt.Errorf("failed to read password for %s: %w", server.URL, err)
		}
		passwords[server.ID()] = password
	}

	return passwords, nil
}
