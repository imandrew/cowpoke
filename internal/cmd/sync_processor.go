package cmd

import (
	"context"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"cowpoke/internal/config"
	"cowpoke/internal/kubeconfig"
	"cowpoke/internal/rancher"
	"cowpoke/internal/utils"

	"golang.org/x/sync/errgroup"
)

const (
	// maxConcurrentServers limits the number of servers processed simultaneously.
	maxConcurrentServers = 3
	// maxConcurrentClusters limits the number of clusters processed simultaneously per server.
	maxConcurrentClusters = 2
	// authTimeout is the timeout for authentication operations.
	authTimeout = 30 * time.Second
	// clustersTimeout is the timeout for retrieving cluster list.
	clustersTimeout = 30 * time.Second
	// kubeconfigTimeout is the timeout for retrieving kubeconfig.
	kubeconfigTimeout = 60 * time.Second
)

// SyncProcessor handles the business logic for syncing kubeconfigs from Rancher servers.
type SyncProcessor struct {
	kubeconfigManager *kubeconfig.Manager
	logger            *slog.Logger
}

// NewSyncProcessor creates a new sync processor.
func NewSyncProcessor(kubeconfigManager *kubeconfig.Manager, logger *slog.Logger) *SyncProcessor {
	return &SyncProcessor{
		kubeconfigManager: kubeconfigManager,
		logger:            logger,
	}
}

// ProcessServers processes multiple Rancher servers concurrently.
func (sp *SyncProcessor) ProcessServers(ctx context.Context, servers []config.RancherServer) ([]string, error) {
	passwords, err := sp.collectPasswords(ctx, servers)
	if err != nil {
		return nil, fmt.Errorf("failed to collect passwords: %w", err)
	}

	g, ctx := errgroup.WithContext(ctx)
	var kubeconfigPaths []string
	var pathsMutex sync.Mutex
	semaphore := make(chan struct{}, maxConcurrentServers)

	for _, server := range servers {
		g.Go(func() error {
			select {
			case semaphore <- struct{}{}:
				defer func() { <-semaphore }()
			case <-ctx.Done():
				return ctx.Err()
			}

			return sp.processSingleServer(ctx, server, passwords[server.URL], &kubeconfigPaths, &pathsMutex)
		})
	}

	if waitErr := g.Wait(); waitErr != nil {
		sp.logger.ErrorContext(ctx, "Error during concurrent sync", "error", waitErr)
		return nil, fmt.Errorf("sync failed: %w", waitErr)
	}

	return kubeconfigPaths, nil
}

// collectPasswords prompts for passwords for all servers that need them.
func (sp *SyncProcessor) collectPasswords(
	ctx context.Context,
	servers []config.RancherServer,
) (map[string]string, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	passwords := make(map[string]string)

	if !utils.CanPromptForPassword() {
		sp.logger.WarnContext(ctx, "Cannot prompt for passwords in non-interactive environment, using fallback")
		for _, server := range servers {
			passwords[server.URL] = "test-fallback-password"
		}
		return passwords, nil
	}

	for _, server := range servers {
		prompt := fmt.Sprintf("Enter password for %s (%s): ", server.URL, server.Username)
		password, err := utils.PromptForPassword(prompt)
		if err != nil {
			return nil, fmt.Errorf("failed to read password for %s: %w", server.URL, err)
		}

		passwords[server.URL] = password
	}

	return passwords, nil
}

// processSingleServer handles syncing kubeconfigs from a single Rancher server.
func (sp *SyncProcessor) processSingleServer(
	ctx context.Context,
	server config.RancherServer,
	password string,
	kubeconfigPaths *[]string,
	pathsMutex *sync.Mutex,
) error {
	serverLogger := sp.logger.With("server_url", server.URL, "auth_type", server.AuthType)
	serverLogger.InfoContext(ctx, "Starting server processing")

	client := rancher.NewClient(server)

	authCtx, authCancel := context.WithTimeout(ctx, authTimeout)
	defer authCancel()

	_, err := client.Authenticate(authCtx, password)
	if err != nil {
		serverLogger.ErrorContext(authCtx, "Authentication failed", "error", err)
		return nil
	}
	serverLogger.InfoContext(authCtx, "Authentication successful")

	clustersCtx, clustersCancel := context.WithTimeout(ctx, clustersTimeout)
	defer clustersCancel()

	clusters, err := client.GetClusters(clustersCtx)
	if err != nil {
		serverLogger.ErrorContext(clustersCtx, "Failed to get clusters", "error", err)
		return nil
	}

	serverLogger.InfoContext(clustersCtx, "Retrieved clusters", "count", len(clusters))

	sp.processClusters(ctx, clusters, server.ID, server.URL, client, kubeconfigPaths, pathsMutex, serverLogger)
	return nil
}

// processClusters processes multiple clusters concurrently within a server.
func (sp *SyncProcessor) processClusters(
	ctx context.Context,
	clusters []config.Cluster,
	serverID string,
	serverURL string,
	client *rancher.Client,
	kubeconfigPaths *[]string,
	pathsMutex *sync.Mutex,
	serverLogger *slog.Logger,
) {
	clusterGroup, clusterCtx := errgroup.WithContext(ctx)
	clusterSemaphore := make(chan struct{}, maxConcurrentClusters)

	for _, cluster := range clusters {
		clusterGroup.Go(func() error {
			select {
			case clusterSemaphore <- struct{}{}:
				defer func() { <-clusterSemaphore }()
			case <-clusterCtx.Done():
				return clusterCtx.Err()
			}

			return sp.processCluster(
				clusterCtx,
				cluster,
				serverID,
				serverURL,
				client,
				kubeconfigPaths,
				pathsMutex,
				serverLogger,
			)
		})
	}

	if err := clusterGroup.Wait(); err != nil {
		serverLogger.ErrorContext(ctx, "Error processing clusters", "error", err)
		return
	}

	serverLogger.InfoContext(ctx, "Server processing completed successfully")
}

// processCluster handles downloading and saving a single cluster's kubeconfig.
func (sp *SyncProcessor) processCluster(
	ctx context.Context,
	cluster config.Cluster,
	serverID string,
	serverURL string,
	client *rancher.Client,
	kubeconfigPaths *[]string,
	pathsMutex *sync.Mutex,
	serverLogger *slog.Logger,
) error {
	clusterLogger := serverLogger.With("cluster_id", cluster.ID, "cluster_name", cluster.Name)
	clusterLogger.InfoContext(ctx, "Processing cluster")

	kubeconfigCtx, kubeconfigCancel := context.WithTimeout(ctx, kubeconfigTimeout)
	defer kubeconfigCancel()

	kubeconfigData, err := client.GetKubeconfig(kubeconfigCtx, cluster.ID)
	if err != nil {
		clusterLogger.ErrorContext(kubeconfigCtx, "Failed to get kubeconfig", "error", err)
		return nil
	}

	cluster.ServerID = serverID
	cluster.ServerURL = serverURL
	savedPath, err := sp.kubeconfigManager.SaveKubeconfig(cluster, kubeconfigData)
	if err != nil {
		clusterLogger.ErrorContext(ctx, "Failed to save kubeconfig", "error", err)
		return nil
	}

	pathsMutex.Lock()
	*kubeconfigPaths = append(*kubeconfigPaths, savedPath)
	pathsMutex.Unlock()

	clusterLogger.InfoContext(ctx, "Cluster processed successfully", "path", savedPath)
	return nil
}
