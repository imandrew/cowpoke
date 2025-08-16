package domain

import "context"

// ClusterLister handles listing clusters from Rancher servers.
type ClusterLister interface {
	ListClusters(ctx context.Context, token AuthToken, server ConfigServer) ([]Cluster, error)
}

// KubeconfigFetcher handles fetching kubeconfig for specific clusters.
type KubeconfigFetcher interface {
	GetKubeconfig(
		ctx context.Context,
		token AuthToken,
		server ConfigServer,
		clusterID string,
	) ([]byte, error)
}

// Cluster represents a Kubernetes cluster in Rancher.
type Cluster struct {
	ID   string
	Name string
	Type string
}
