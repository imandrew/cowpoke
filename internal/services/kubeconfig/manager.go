package kubeconfig

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"maps"
	"os"
	"path/filepath"

	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/tools/clientcmd/api"

	"cowpoke/internal/domain"
)

const (
	dirPermissions  = 0o700 // Owner-only access
	filePermissions = 0o600 // Read/write owner only
)

// Manager handles kubeconfig file operations.
type Manager struct {
	fs            domain.FileSystemAdapter
	kubeconfigDir string
	logger        *slog.Logger
}

// NewManager creates a new kubeconfig manager.
func NewManager(fs domain.FileSystemAdapter, kubeconfigDir string, logger *slog.Logger) (*Manager, error) {
	if err := fs.MkdirAll(kubeconfigDir, dirPermissions); err != nil {
		return nil, fmt.Errorf("failed to create kubeconfig directory: %w", err)
	}

	return &Manager{
		fs:            fs,
		kubeconfigDir: kubeconfigDir,
		logger:        logger,
	}, nil
}

// SaveKubeconfig saves a kubeconfig to a file after preprocessing to avoid conflicts.
func (m *Manager) SaveKubeconfig(ctx context.Context, path string, content []byte, serverID string) error {
	dir := filepath.Dir(path)
	if err := m.fs.MkdirAll(dir, dirPermissions); err != nil {
		return fmt.Errorf("failed to create directory for kubeconfig: %w", err)
	}

	// Preprocess the kubeconfig to append server ID to all resources
	processedContent, err := m.PreprocessKubeconfig(ctx, content, serverID)
	if err != nil {
		return fmt.Errorf("failed to preprocess kubeconfig: %w", err)
	}

	if writeErr := m.fs.WriteFile(path, processedContent, filePermissions); writeErr != nil {
		return fmt.Errorf("failed to write kubeconfig file: %w", writeErr)
	}

	m.logger.DebugContext(ctx, "Kubeconfig saved", "path", path)
	return nil
}

// PreprocessKubeconfig appends server ID to all kubeconfig resources to avoid naming conflicts.
func (m *Manager) PreprocessKubeconfig(ctx context.Context, content []byte, serverID string) ([]byte, error) {
	config, err := clientcmd.Load(content)
	if err != nil {
		return nil, fmt.Errorf("failed to parse kubeconfig: %w", err)
	}

	m.logger.DebugContext(ctx, "Preprocessing kubeconfig to append server ID",
		"server_id", serverID,
		"clusters", len(config.Clusters),
		"contexts", len(config.Contexts),
		"users", len(config.AuthInfos))

	// Rename resources and track mappings
	clusterNameMap := m.renameClusters(ctx, config, serverID)
	userNameMap := m.renameUsers(ctx, config, serverID)
	contextNameMap := m.renameContexts(ctx, config, serverID, clusterNameMap, userNameMap)

	m.logger.InfoContext(ctx, "Kubeconfig preprocessing completed",
		"server_id", serverID,
		"renamed_clusters", len(clusterNameMap),
		"renamed_users", len(userNameMap),
		"renamed_contexts", len(contextNameMap))

	return clientcmd.Write(*config)
}

// renameClusters renames all clusters by appending server ID and returns name mappings.
func (m *Manager) renameClusters(ctx context.Context, config *api.Config, serverID string) map[string]string {
	clusterNameMap := make(map[string]string)

	config.Clusters = maps.Collect(func(yield func(string, *api.Cluster) bool) {
		for oldName, cluster := range config.Clusters {
			newName := fmt.Sprintf("%s-%s", oldName, serverID)
			clusterNameMap[oldName] = newName
			m.logger.DebugContext(ctx, "Renamed cluster", "old", oldName, "new", newName)
			if !yield(newName, cluster) {
				return
			}
		}
	})

	return clusterNameMap
}

// renameUsers renames all users/auth-infos by appending server ID and returns name mappings.
func (m *Manager) renameUsers(ctx context.Context, config *api.Config, serverID string) map[string]string {
	userNameMap := make(map[string]string)

	config.AuthInfos = maps.Collect(func(yield func(string, *api.AuthInfo) bool) {
		for oldName, authInfo := range config.AuthInfos {
			newName := fmt.Sprintf("%s-%s", oldName, serverID)
			userNameMap[oldName] = newName
			m.logger.DebugContext(ctx, "Renamed user", "old", oldName, "new", newName)
			if !yield(newName, authInfo) {
				return
			}
		}
	})

	return userNameMap
}

// renameContexts renames all contexts, updates their references, and returns name mappings.
func (m *Manager) renameContexts(
	ctx context.Context,
	config *api.Config,
	serverID string,
	clusterNameMap, userNameMap map[string]string,
) map[string]string {
	contextNameMap := make(map[string]string)

	config.Contexts = maps.Collect(func(yield func(string, *api.Context) bool) {
		for oldName, context := range config.Contexts {
			newName := fmt.Sprintf("%s-%s", oldName, serverID)

			// Update cluster reference
			if newClusterName, exists := clusterNameMap[context.Cluster]; exists {
				context.Cluster = newClusterName
			}

			// Update user reference
			if newUserName, exists := userNameMap[context.AuthInfo]; exists {
				context.AuthInfo = newUserName
			}

			contextNameMap[oldName] = newName
			m.logger.DebugContext(ctx, "Renamed context", "old", oldName, "new", newName)
			if !yield(newName, context) {
				return
			}
		}
	})

	return contextNameMap
}

// MergeKubeconfigs merges multiple kubeconfig files into one.
// All resources should already be renamed with server IDs by preprocessing.
func (m *Manager) MergeKubeconfigs(ctx context.Context, paths []string, outputPath string) error {
	if len(paths) == 0 {
		return errors.New("no kubeconfig paths provided for merging")
	}

	m.logger.DebugContext(ctx, "Starting kubeconfig merge", "input_count", len(paths))

	// Use client-go's built-in merging
	loadingRules := &clientcmd.ClientConfigLoadingRules{
		Precedence: paths,
	}

	mergedConfig, err := loadingRules.Load()
	if err != nil {
		return fmt.Errorf("failed to merge kubeconfigs: %w", err)
	}

	if len(mergedConfig.Clusters) == 0 {
		return errors.New("no valid clusters found in merged kubeconfig")
	}

	// Ensure output directory exists with secure permissions
	outputDir := filepath.Dir(outputPath)
	if mkdirErr := m.fs.MkdirAll(outputDir, dirPermissions); mkdirErr != nil {
		return fmt.Errorf("failed to create output directory: %w", mkdirErr)
	}

	if writeErr := clientcmd.WriteToFile(*mergedConfig, outputPath); writeErr != nil {
		return fmt.Errorf("failed to write merged kubeconfig: %w", writeErr)
	}

	// Ensure the output file has secure permissions
	if chmodErr := m.fs.Chmod(outputPath, filePermissions); chmodErr != nil {
		return fmt.Errorf("failed to set secure permissions on output file: %w", chmodErr)
	}

	m.logger.InfoContext(ctx, "Kubeconfigs merged successfully",
		"input_count", len(paths),
		"output", outputPath,
		"total_clusters", len(mergedConfig.Clusters),
		"total_contexts", len(mergedConfig.Contexts))

	return nil
}

// CleanupTempFiles removes temporary kubeconfig files.
func (m *Manager) CleanupTempFiles(ctx context.Context, paths []string) error {
	var errs []error

	for _, path := range paths {
		if err := m.fs.Remove(path); err != nil {
			if !os.IsNotExist(err) {
				errs = append(errs, fmt.Errorf("failed to remove %s: %w", path, err))
			}
		} else {
			m.logger.DebugContext(ctx, "Cleaned up temporary kubeconfig", "path", path)
		}
	}

	if len(errs) > 0 {
		return fmt.Errorf("failed to cleanup some files: %w", errors.Join(errs...))
	}

	return nil
}
