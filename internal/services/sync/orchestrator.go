package sync

import (
	"context"
	"fmt"
	"log/slog"
	"path/filepath"
	"sync"

	"cowpoke/internal/domain"
)

const (
	// maxConcurrentDownloads is the number of concurrent download workers.
	maxConcurrentDownloads = 5
)

// Orchestrator orchestrates concurrent kubeconfig synchronization from multiple Rancher servers.
type Orchestrator struct {
	rancherClient     domain.RancherClient
	kubeconfigHandler domain.KubeconfigHandler
	configProvider    domain.ConfigProvider
	logger            *slog.Logger
}

// NewOrchestrator creates a new sync orchestrator.
func NewOrchestrator(
	rancherClient domain.RancherClient,
	kubeconfigHandler domain.KubeconfigHandler,
	configProvider domain.ConfigProvider,
	logger *slog.Logger,
) *Orchestrator {
	return &Orchestrator{
		rancherClient:     rancherClient,
		kubeconfigHandler: kubeconfigHandler,
		configProvider:    configProvider,
		logger:            logger,
	}
}

// DiscoveryTask represents a server to authenticate and discover clusters from.
type DiscoveryTask struct {
	Server   domain.ConfigServer
	Password string
}

// DiscoveryResult contains the result of cluster discovery for a server.
type DiscoveryResult struct {
	Server   domain.ConfigServer
	Token    domain.AuthToken
	Clusters []domain.Cluster
	Error    error
}

// DownloadTask represents a cluster kubeconfig to download.
type DownloadTask struct {
	Server    domain.ConfigServer
	Cluster   domain.Cluster
	Token     domain.AuthToken
	OutputDir string
}

// DownloadResult contains the result of a kubeconfig download.
type DownloadResult struct {
	Task     DownloadTask
	FilePath string
	Error    error
}

// SyncServers performs concurrent discovery and download of kubeconfigs from the provided servers.
func (o *Orchestrator) SyncServers(
	ctx context.Context,
	servers []domain.ConfigServer,
	passwords map[string]string,
	filter domain.ClusterFilter,
) ([]string, error) {
	if len(servers) == 0 {
		return nil, nil
	}

	o.logger.InfoContext(ctx, "Starting concurrent sync",
		"servers", len(servers),
		"maxConcurrentDownloads", maxConcurrentDownloads)

	// Phase 1: Concurrent cluster discovery
	downloadTasks, err := o.discoverClustersAsync(ctx, servers, passwords, filter)
	if err != nil {
		return nil, fmt.Errorf("cluster discovery failed: %w", err)
	}

	if len(downloadTasks) == 0 {
		o.logger.WarnContext(ctx, "No clusters discovered from any server")
		return nil, nil
	}

	// Phase 2: Concurrent kubeconfig downloads
	kubeconfigPaths, err := o.downloadKubeconfigsAsync(ctx, downloadTasks)
	if err != nil {
		return nil, fmt.Errorf("kubeconfig downloads failed: %w", err)
	}

	o.logger.InfoContext(ctx, "Concurrent sync completed",
		"servers", len(servers),
		"clusters", len(downloadTasks),
		"kubeconfigs", len(kubeconfigPaths))

	return kubeconfigPaths, nil
}

// discoverClustersAsync performs concurrent authentication and cluster discovery.
func (o *Orchestrator) discoverClustersAsync(
	ctx context.Context,
	servers []domain.ConfigServer,
	passwords map[string]string,
	filter domain.ClusterFilter,
) ([]DownloadTask, error) {
	// Create discovery tasks
	discoveryTasks := make([]DiscoveryTask, 0, len(servers))
	for _, server := range servers {
		password, exists := passwords[server.ID()]
		if !exists {
			o.logger.WarnContext(ctx, "No password provided for server", "server", server.URL)
			continue
		}
		discoveryTasks = append(discoveryTasks, DiscoveryTask{
			Server:   server,
			Password: password,
		})
	}

	// Execute discovery tasks concurrently
	resultChan := make(chan DiscoveryResult, len(discoveryTasks))
	var wg sync.WaitGroup

	for _, task := range discoveryTasks {
		wg.Add(1)
		go func(task DiscoveryTask) {
			defer wg.Done()
			o.discoverClustersForServer(ctx, task, resultChan)
		}(task)
	}

	// Wait for all discovery tasks to complete
	wg.Wait()
	close(resultChan)

	// Get kubeconfig directory for download tasks
	kubeconfigDir, err := o.configProvider.GetKubeconfigDir()
	if err != nil {
		return nil, fmt.Errorf("failed to get kubeconfig directory: %w", err)
	}

	// Collect results and build download tasks
	var downloadTasks []DownloadTask
	for result := range resultChan {
		if result.Error != nil {
			o.logger.ErrorContext(ctx, "Failed to discover clusters for server",
				"server", result.Server.URL,
				"error", result.Error)
			continue
		}

		for _, cluster := range result.Clusters {
			// Apply filter to exclude clusters
			if filter.ShouldExclude(cluster.Name) {
				o.logger.DebugContext(ctx, "Excluding cluster",
					"cluster", cluster.Name,
					"server", result.Server.URL)
				continue
			}

			downloadTasks = append(downloadTasks, DownloadTask{
				Server:    result.Server,
				Cluster:   cluster,
				Token:     result.Token,
				OutputDir: kubeconfigDir,
			})
		}
	}

	return downloadTasks, nil
}

// discoverClustersForServer authenticates with a server and discovers its clusters.
func (o *Orchestrator) discoverClustersForServer(
	ctx context.Context,
	task DiscoveryTask,
	resultChan chan<- DiscoveryResult,
) {
	o.logger.InfoContext(ctx, "Discovering clusters for server", "server", task.Server.URL)

	// Authenticate with the server
	token, err := o.rancherClient.Authenticate(ctx, task.Server, task.Password)
	if err != nil {
		resultChan <- DiscoveryResult{
			Server: task.Server,
			Error:  fmt.Errorf("authentication failed: %w", err),
		}
		return
	}

	// Get list of clusters
	clusters, err := o.rancherClient.ListClusters(ctx, token, task.Server)
	if err != nil {
		resultChan <- DiscoveryResult{
			Server: task.Server,
			Error:  fmt.Errorf("failed to list clusters: %w", err),
		}
		return
	}

	o.logger.InfoContext(ctx, "Discovered clusters for server",
		"server", task.Server.URL,
		"clusters", len(clusters))

	resultChan <- DiscoveryResult{
		Server:   task.Server,
		Token:    token,
		Clusters: clusters,
		Error:    nil,
	}
}

// downloadKubeconfigsAsync performs concurrent kubeconfig downloads using a worker pool.
func (o *Orchestrator) downloadKubeconfigsAsync(
	ctx context.Context,
	downloadTasks []DownloadTask,
) ([]string, error) {
	if len(downloadTasks) == 0 {
		return nil, nil
	}

	o.logger.InfoContext(ctx, "Starting concurrent downloads",
		"tasks", len(downloadTasks),
		"workers", maxConcurrentDownloads)

	// Create channels for work distribution
	taskChan := make(chan DownloadTask, len(downloadTasks))
	resultChan := make(chan DownloadResult, len(downloadTasks))

	// Start worker pool
	var wg sync.WaitGroup
	for i := range maxConcurrentDownloads {
		wg.Add(1)
		go func(workerID int) {
			defer wg.Done()
			o.downloadWorker(ctx, workerID, taskChan, resultChan)
		}(i)
	}

	// Send tasks to workers
	for _, task := range downloadTasks {
		taskChan <- task
	}
	close(taskChan)

	// Wait for all workers to complete
	wg.Wait()
	close(resultChan)

	// Collect results
	var kubeconfigPaths []string
	var errorCount int
	for result := range resultChan {
		if result.Error != nil {
			o.logger.ErrorContext(ctx, "Failed to download kubeconfig",
				"server", result.Task.Server.URL,
				"cluster", result.Task.Cluster.Name,
				"error", result.Error)
			errorCount++
			continue
		}
		kubeconfigPaths = append(kubeconfigPaths, result.FilePath)
	}

	o.logger.InfoContext(ctx, "Concurrent downloads completed",
		"successful", len(kubeconfigPaths),
		"failed", errorCount,
		"total", len(downloadTasks))

	if errorCount > 0 {
		return kubeconfigPaths, fmt.Errorf(
			"failed to download %d out of %d kubeconfigs",
			errorCount,
			len(downloadTasks),
		)
	}
	return kubeconfigPaths, nil
}

// downloadWorker processes download tasks from the task channel.
func (o *Orchestrator) downloadWorker(
	ctx context.Context,
	workerID int,
	taskChan <-chan DownloadTask,
	resultChan chan<- DownloadResult,
) {
	for task := range taskChan {
		o.logger.DebugContext(ctx, "Worker processing download",
			"worker", workerID,
			"server", task.Server.URL,
			"cluster", task.Cluster.Name)

		result := o.downloadKubeconfig(ctx, task)
		resultChan <- result
	}
}

// downloadKubeconfig downloads and saves a kubeconfig for a specific cluster.
func (o *Orchestrator) downloadKubeconfig(ctx context.Context, task DownloadTask) DownloadResult {
	// Get kubeconfig for this cluster
	kubeconfig, err := o.rancherClient.GetKubeconfig(ctx, task.Token, task.Server, task.Cluster.ID)
	if err != nil {
		return DownloadResult{
			Task:  task,
			Error: fmt.Errorf("failed to get kubeconfig: %w", err),
		}
	}

	// Save to temporary file
	filename := fmt.Sprintf("%s-%s.yaml", task.Cluster.Name, task.Server.ID())
	path := filepath.Join(task.OutputDir, filename)

	if saveErr := o.kubeconfigHandler.SaveKubeconfig(ctx, path, kubeconfig, task.Server.ID()); saveErr != nil {
		return DownloadResult{
			Task:  task,
			Error: fmt.Errorf("failed to save kubeconfig: %w", saveErr),
		}
	}

	o.logger.DebugContext(ctx, "Successfully downloaded kubeconfig",
		"server", task.Server.URL,
		"cluster", task.Cluster.Name,
		"path", path)

	return DownloadResult{
		Task:     task,
		FilePath: path,
		Error:    nil,
	}
}
