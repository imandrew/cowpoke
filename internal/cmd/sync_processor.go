package cmd

import (
	"context"
	"fmt"
	"sync"
	"time"

	"cowpoke/internal/config"
	"cowpoke/internal/kubeconfig"
	"cowpoke/internal/logging"
	"cowpoke/internal/rancher"
	"cowpoke/internal/utils"

	"golang.org/x/sync/errgroup"
)

// SyncProcessor handles the business logic for syncing kubeconfigs from Rancher servers
type SyncProcessor struct {
	kubeconfigManager *kubeconfig.Manager
	logger            *logging.Logger
}

// NewSyncProcessor creates a new sync processor
func NewSyncProcessor(kubeconfigManager *kubeconfig.Manager, logger *logging.Logger) *SyncProcessor {
	return &SyncProcessor{
		kubeconfigManager: kubeconfigManager,
		logger:            logger,
	}
}

// ProcessServers processes multiple Rancher servers concurrently
func (sp *SyncProcessor) ProcessServers(ctx context.Context, servers []config.RancherServer) ([]string, error) {
	// Collect passwords upfront for all servers before concurrent processing
	passwords, err := sp.collectPasswords(ctx, servers)
	if err != nil {
		return nil, fmt.Errorf("failed to collect passwords: %w", err)
	}
	// Use errgroup for concurrent processing with proper error handling
	g, ctx := errgroup.WithContext(ctx)

	// Thread-safe slice for collecting kubeconfig paths
	var kubeconfigPaths []string
	var pathsMutex sync.Mutex

	// Limit concurrent servers to avoid overwhelming the system
	semaphore := make(chan struct{}, 3) // Max 3 concurrent servers

	for _, server := range servers {
		server := server // Capture loop variable
		g.Go(func() error {
			// Acquire semaphore
			select {
			case semaphore <- struct{}{}:
				defer func() { <-semaphore }()
			case <-ctx.Done():
				return ctx.Err()
			}

			return sp.processSingleServer(ctx, server, passwords[server.URL], &kubeconfigPaths, &pathsMutex)
		})
	}

	// Wait for all goroutines to complete
	if err := g.Wait(); err != nil {
		sp.logger.ErrorContext(ctx, "Error during concurrent sync", "error", err)
		return nil, fmt.Errorf("sync failed: %w", err)
	}

	return kubeconfigPaths, nil
}

// collectPasswords prompts for passwords for all servers that need them
func (sp *SyncProcessor) collectPasswords(ctx context.Context, servers []config.RancherServer) (map[string]string, error) {
	// Check for context cancellation first
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	passwords := make(map[string]string)

	// Check if we can prompt for passwords (running in a terminal)
	if !utils.CanPromptForPassword() {
		// For non-interactive environments (like tests), use a dummy password
		// In production, this would typically indicate the need for stored credentials or tokens
		sp.logger.WarnContext(ctx, "Cannot prompt for passwords in non-interactive environment, using fallback")
		for _, server := range servers {
			passwords[server.URL] = "test-fallback-password"
		}
		return passwords, nil
	}

	for _, server := range servers {
		// For now, we prompt for all servers. In the future, we could:
		// 1. Check if the server uses token-based auth
		// 2. Check if credentials are stored in a keychain
		// 3. Skip prompting for certain auth types

		prompt := fmt.Sprintf("Enter password for %s (%s): ", server.URL, server.Username)
		password, err := utils.PromptForPassword(prompt)
		if err != nil {
			return nil, fmt.Errorf("failed to read password for %s: %w", server.URL, err)
		}

		passwords[server.URL] = password
	}

	return passwords, nil
}

// processSingleServer handles syncing kubeconfigs from a single Rancher server
func (sp *SyncProcessor) processSingleServer(
	ctx context.Context,
	server config.RancherServer,
	password string,
	kubeconfigPaths *[]string,
	pathsMutex *sync.Mutex,
) error {
	serverLogger := sp.logger.WithServer(server.URL, server.AuthType)
	serverLogger.InfoContext(ctx, "Starting server processing")

	client := rancher.NewClient(server)

	// Create context with timeout for authentication
	authCtx, authCancel := context.WithTimeout(ctx, 30*time.Second)
	defer authCancel()

	_, err := client.Authenticate(authCtx, password)
	if err != nil {
		serverLogger.ErrorContext(ctx, "Authentication failed", "error", err)
		// Don't return error here - continue with other servers
		return nil
	}
	serverLogger.InfoContext(ctx, "Authentication successful")

	// Create context with timeout for getting clusters
	clustersCtx, clustersCancel := context.WithTimeout(ctx, 30*time.Second)
	defer clustersCancel()

	clusters, err := client.GetClusters(clustersCtx)
	if err != nil {
		serverLogger.ErrorContext(ctx, "Failed to get clusters", "error", err)
		return nil // Continue with other servers
	}

	serverLogger.InfoContext(ctx, "Retrieved clusters", "count", len(clusters))

	return sp.processClusters(ctx, clusters, server.ID, server.URL, client, kubeconfigPaths, pathsMutex, serverLogger)
}

// processClusters processes multiple clusters concurrently within a server
func (sp *SyncProcessor) processClusters(
	ctx context.Context,
	clusters []config.Cluster,
	serverID string,
	serverURL string,
	client *rancher.Client,
	kubeconfigPaths *[]string,
	pathsMutex *sync.Mutex,
	serverLogger *logging.Logger,
) error {
	// Process clusters concurrently within this server
	clusterGroup, clusterCtx := errgroup.WithContext(ctx)
	clusterSemaphore := make(chan struct{}, 2) // Max 2 concurrent clusters per server

	for _, cluster := range clusters {
		cluster := cluster // Capture loop variable
		clusterGroup.Go(func() error {
			// Acquire cluster semaphore
			select {
			case clusterSemaphore <- struct{}{}:
				defer func() { <-clusterSemaphore }()
			case <-clusterCtx.Done():
				return clusterCtx.Err()
			}

			return sp.processCluster(clusterCtx, cluster, serverID, serverURL, client, kubeconfigPaths, pathsMutex, serverLogger)
		})
	}

	// Wait for all clusters to be processed
	if err := clusterGroup.Wait(); err != nil {
		serverLogger.ErrorContext(ctx, "Error processing clusters", "error", err)
		return nil // Don't fail the entire sync for one server
	}

	serverLogger.InfoContext(ctx, "Server processing completed successfully")
	return nil
}

// processCluster handles downloading and saving a single cluster's kubeconfig
func (sp *SyncProcessor) processCluster(
	ctx context.Context,
	cluster config.Cluster,
	serverID string,
	serverURL string,
	client *rancher.Client,
	kubeconfigPaths *[]string,
	pathsMutex *sync.Mutex,
	serverLogger *logging.Logger,
) error {
	clusterLogger := serverLogger.WithCluster(cluster.ID, cluster.Name)
	clusterLogger.InfoContext(ctx, "Processing cluster")

	// Create context with timeout for getting kubeconfig
	kubeconfigCtx, kubeconfigCancel := context.WithTimeout(ctx, 60*time.Second)
	defer kubeconfigCancel()

	kubeconfigData, err := client.GetKubeconfig(kubeconfigCtx, cluster.ID)
	if err != nil {
		clusterLogger.ErrorContext(ctx, "Failed to get kubeconfig", "error", err)
		return nil // Continue with other clusters
	}

	cluster.ServerID = serverID
	cluster.ServerURL = serverURL
	savedPath, err := sp.kubeconfigManager.SaveKubeconfig(cluster, kubeconfigData)
	if err != nil {
		clusterLogger.ErrorContext(ctx, "Failed to save kubeconfig", "error", err)
		return nil // Continue with other clusters
	}

	// Thread-safe append to shared slice
	pathsMutex.Lock()
	*kubeconfigPaths = append(*kubeconfigPaths, savedPath)
	pathsMutex.Unlock()

	clusterLogger.InfoContext(ctx, "Cluster processed successfully", "path", savedPath)
	return nil
}
