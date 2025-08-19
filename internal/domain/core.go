package domain

import "context"

// RancherClient handles all Rancher API operations.
type RancherClient interface {
	// Authenticate with a Rancher server and return an auth token.
	Authenticate(ctx context.Context, server ConfigServer, password string) (AuthToken, error)

	// ListClusters retrieves all clusters from a Rancher server.
	ListClusters(ctx context.Context, token AuthToken, server ConfigServer) ([]Cluster, error)

	// GetKubeconfig retrieves the kubeconfig for a specific cluster.
	GetKubeconfig(ctx context.Context, token AuthToken, server ConfigServer, clusterID string) ([]byte, error)
}

// KubeconfigHandler handles all kubeconfig file operations.
type KubeconfigHandler interface {
	// SaveKubeconfig saves a kubeconfig to a file after preprocessing to avoid conflicts.
	SaveKubeconfig(ctx context.Context, path string, content []byte, serverID string) error

	// MergeKubeconfigs merges multiple kubeconfig files into one, applying cluster filtering.
	// The filter is applied to context and cluster names within each kubeconfig before merging.
	MergeKubeconfigs(ctx context.Context, paths []string, outputPath string, filter ClusterFilter) error

	// CleanupTempFiles removes temporary kubeconfig files.
	CleanupTempFiles(ctx context.Context, paths []string) error
}

// SyncOrchestrator orchestrates the entire kubeconfig synchronization process.
type SyncOrchestrator interface {
	// SyncServers performs concurrent discovery and download of kubeconfigs from the provided servers.
	// Returns a SyncResult containing paths to downloaded kubeconfig files and statistics.
	// Filtering is now handled at the kubeconfig merge level, not during download.
	SyncServers(
		ctx context.Context,
		servers []ConfigServer,
		passwords map[string]string,
	) (*SyncResult, error)
}
