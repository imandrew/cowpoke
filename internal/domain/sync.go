package domain

import "context"

// KubeconfigSyncer orchestrates concurrent kubeconfig synchronization from multiple Rancher servers.
type KubeconfigSyncer interface {
	// SyncAllServers performs concurrent discovery and download of kubeconfigs from all servers.
	// Returns paths to downloaded kubeconfig files ready for merging.
	SyncAllServers(
		ctx context.Context,
		servers []ConfigServer,
		passwords map[string]string,
	) ([]string, error)
}
