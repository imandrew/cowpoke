package domain

import "context"

// KubeconfigWriter handles saving kubeconfig files.
type KubeconfigWriter interface {
	SaveKubeconfig(ctx context.Context, path string, content []byte, serverID string) error
}

// KubeconfigMerger handles merging multiple kubeconfig files.
type KubeconfigMerger interface {
	MergeKubeconfigs(ctx context.Context, paths []string, outputPath string) error
}

// KubeconfigCleaner handles cleanup of temporary kubeconfig files.
type KubeconfigCleaner interface {
	CleanupTempFiles(ctx context.Context, paths []string) error
}
