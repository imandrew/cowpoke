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

// Handler handles all kubeconfig file operations.
type Handler struct {
	fs            domain.FileSystemAdapter
	kubeconfigDir string
	logger        *slog.Logger
}

// NewHandler creates a new kubeconfig handler.
func NewHandler(fs domain.FileSystemAdapter, kubeconfigDir string, logger *slog.Logger) (*Handler, error) {
	if err := fs.MkdirAll(kubeconfigDir, dirPermissions); err != nil {
		return nil, fmt.Errorf("failed to create kubeconfig directory: %w", err)
	}

	return &Handler{
		fs:            fs,
		kubeconfigDir: kubeconfigDir,
		logger:        logger,
	}, nil
}

// SaveKubeconfig saves a kubeconfig to a file after preprocessing to avoid conflicts.
func (h *Handler) SaveKubeconfig(ctx context.Context, path string, content []byte, serverID string) error {
	dir := filepath.Dir(path)
	if err := h.fs.MkdirAll(dir, dirPermissions); err != nil {
		return fmt.Errorf("failed to create directory for kubeconfig: %w", err)
	}

	// Preprocess the kubeconfig to append server ID to all resources
	processedContent, err := h.PreprocessKubeconfig(ctx, content, serverID)
	if err != nil {
		return fmt.Errorf("failed to preprocess kubeconfig: %w", err)
	}

	if writeErr := h.fs.WriteFile(path, processedContent, filePermissions); writeErr != nil {
		return fmt.Errorf("failed to write kubeconfig file: %w", writeErr)
	}

	h.logger.DebugContext(ctx, "Kubeconfig saved", "path", path)
	return nil
}

// PreprocessKubeconfig appends server ID to all kubeconfig resources to avoid naming conflicts.
func (h *Handler) PreprocessKubeconfig(ctx context.Context, content []byte, serverID string) ([]byte, error) {
	config, err := clientcmd.Load(content)
	if err != nil {
		return nil, fmt.Errorf("failed to parse kubeconfig: %w", err)
	}

	h.logger.DebugContext(ctx, "Preprocessing kubeconfig to append server ID",
		"server_id", serverID,
		"clusters", len(config.Clusters),
		"contexts", len(config.Contexts),
		"users", len(config.AuthInfos))

	// Rename resources and track mappings
	clusterNameMap := h.renameClusters(ctx, config, serverID)
	userNameMap := h.renameUsers(ctx, config, serverID)
	contextNameMap := h.renameContexts(ctx, config, serverID, clusterNameMap, userNameMap)

	h.logger.DebugContext(ctx, "Kubeconfig preprocessing completed",
		"server_id", serverID,
		"renamed_clusters", len(clusterNameMap),
		"renamed_users", len(userNameMap),
		"renamed_contexts", len(contextNameMap))

	return clientcmd.Write(*config)
}

// renameClusters renames all clusters by appending server ID and returns name mappings.
func (h *Handler) renameClusters(ctx context.Context, config *api.Config, serverID string) map[string]string {
	clusterNameMap := make(map[string]string)

	config.Clusters = maps.Collect(func(yield func(string, *api.Cluster) bool) {
		for oldName, cluster := range config.Clusters {
			newName := fmt.Sprintf("%s-%s", oldName, serverID)
			clusterNameMap[oldName] = newName
			h.logger.DebugContext(ctx, "Renamed cluster", "old", oldName, "new", newName)
			if !yield(newName, cluster) {
				return
			}
		}
	})

	return clusterNameMap
}

// renameUsers renames all users/auth-infos by appending server ID and returns name mappings.
func (h *Handler) renameUsers(ctx context.Context, config *api.Config, serverID string) map[string]string {
	userNameMap := make(map[string]string)

	config.AuthInfos = maps.Collect(func(yield func(string, *api.AuthInfo) bool) {
		for oldName, authInfo := range config.AuthInfos {
			newName := fmt.Sprintf("%s-%s", oldName, serverID)
			userNameMap[oldName] = newName
			h.logger.DebugContext(ctx, "Renamed user", "old", oldName, "new", newName)
			if !yield(newName, authInfo) {
				return
			}
		}
	})

	return userNameMap
}

// renameContexts renames all contexts, updates their references, and returns name mappings.
func (h *Handler) renameContexts(
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
			h.logger.DebugContext(ctx, "Renamed context", "old", oldName, "new", newName)
			if !yield(newName, context) {
				return
			}
		}
	})

	return contextNameMap
}

// MergeKubeconfigs merges multiple kubeconfig files into one, applying cluster filtering.
// Resources are preprocessed with server IDs to avoid conflicts.
// Filtering is applied at kubeconfig level to handle multi-cluster Rancher files.
func (h *Handler) MergeKubeconfigs(
	ctx context.Context,
	paths []string,
	outputPath string,
	filter domain.ClusterFilter,
) error {
	if len(paths) == 0 {
		return errors.New("no kubeconfig paths provided for merging")
	}

	h.logger.DebugContext(ctx, "Starting kubeconfig merge with filtering",
		"input_count", len(paths),
		"filter_type", fmt.Sprintf("%T", filter))

	mergedConfig := &api.Config{
		Clusters:  make(map[string]*api.Cluster),
		Contexts:  make(map[string]*api.Context),
		AuthInfos: make(map[string]*api.AuthInfo),
	}

	var totalOriginalContexts int
	var totalFilteredContexts int
	var excludedContexts int

	for _, path := range paths {
		config, err := h.loadAndFilterKubeconfig(ctx, path, filter)
		if err != nil {
			h.logger.WarnContext(ctx, "Failed to load or filter kubeconfig, skipping",
				"path", path,
				"error", err)
			continue
		}

		if config == nil {
			// Completely filtered out
			continue
		}

		originalContextCount := len(config.Contexts)
		totalOriginalContexts += originalContextCount

		// Apply filtering to this kubeconfig
		filteredConfig := h.applyFilterToConfig(ctx, config, filter)
		if filteredConfig == nil {
			h.logger.DebugContext(ctx, "All contexts filtered out from kubeconfig",
				"path", path,
				"original_contexts", originalContextCount)
			excludedContexts += originalContextCount
			continue
		}

		filteredContextCount := len(filteredConfig.Contexts)
		totalFilteredContexts += filteredContextCount
		excludedContexts += originalContextCount - filteredContextCount

		h.logger.DebugContext(ctx, "Kubeconfig processed",
			"path", path,
			"original_contexts", originalContextCount,
			"filtered_contexts", filteredContextCount,
			"excluded_contexts", originalContextCount-filteredContextCount)

		// Merge filtered config into the accumulated result
		h.mergeConfigInto(mergedConfig, filteredConfig)
	}

	if len(mergedConfig.Clusters) == 0 {
		if excludedContexts > 0 {
			h.logger.InfoContext(ctx, "All contexts excluded by filters",
				"excluded_contexts", excludedContexts)
			return errors.New("all clusters were excluded by filters - no kubeconfig to write")
		}
		return errors.New("no valid clusters found after filtering")
	}

	// Ensure output directory exists with secure permissions
	outputDir := filepath.Dir(outputPath)
	if mkdirErr := h.fs.MkdirAll(outputDir, dirPermissions); mkdirErr != nil {
		return fmt.Errorf("failed to create output directory: %w", mkdirErr)
	}

	if writeErr := clientcmd.WriteToFile(*mergedConfig, outputPath); writeErr != nil {
		return fmt.Errorf("failed to write merged kubeconfig: %w", writeErr)
	}

	// Ensure the output file has secure permissions
	if chmodErr := h.fs.Chmod(outputPath, filePermissions); chmodErr != nil {
		return fmt.Errorf("failed to set secure permissions on output file: %w", chmodErr)
	}

	h.logger.InfoContext(ctx, "Merged kubeconfigs successfully",
		"contexts", len(mergedConfig.Contexts),
		"excluded", excludedContexts,
		"output", outputPath)

	return nil
}

// loadAndFilterKubeconfig loads a kubeconfig file from disk.
func (h *Handler) loadAndFilterKubeconfig(
	ctx context.Context,
	path string,
	_ domain.ClusterFilter,
) (*api.Config, error) {
	h.logger.DebugContext(ctx, "Loading kubeconfig for filtering", "path", path)

	// Read the kubeconfig file
	data, err := h.fs.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read kubeconfig file %s: %w", path, err)
	}

	// Parse the kubeconfig
	config, err := clientcmd.Load(data)
	if err != nil {
		return nil, fmt.Errorf("failed to parse kubeconfig %s: %w", path, err)
	}

	return config, nil
}

// applyFilterToConfig applies the cluster filter to a kubeconfig, removing excluded contexts and cleaning up unused resources.
func (h *Handler) applyFilterToConfig(
	ctx context.Context,
	config *api.Config,
	filter domain.ClusterFilter,
) *api.Config {
	if filter == nil {
		h.logger.DebugContext(ctx, "No filter provided, keeping all contexts")
		return config
	}

	filteredConfig := &api.Config{
		Clusters:  make(map[string]*api.Cluster),
		Contexts:  make(map[string]*api.Context),
		AuthInfos: make(map[string]*api.AuthInfo),
	}

	usedClusters := make(map[string]bool)
	usedAuthInfos := make(map[string]bool)

	for contextName, context := range config.Contexts {
		// Filter by both context name and cluster name
		shouldExcludeContext := filter.ShouldExclude(contextName)
		shouldExcludeCluster := filter.ShouldExclude(context.Cluster)

		if shouldExcludeContext || shouldExcludeCluster {
			h.logger.DebugContext(ctx, "Excluding context",
				"context", fmt.Sprintf("%q", contextName),
				"cluster", fmt.Sprintf("%q", context.Cluster),
				"by_context", shouldExcludeContext,
				"by_cluster", shouldExcludeCluster)
			continue
		}

		h.logger.DebugContext(ctx, "Including context",
			"context", fmt.Sprintf("%q", contextName),
			"cluster", fmt.Sprintf("%q", context.Cluster))

		filteredConfig.Contexts[contextName] = context
		usedClusters[context.Cluster] = true
		usedAuthInfos[context.AuthInfo] = true
	}

	// Only keep clusters that are referenced by remaining contexts
	for clusterName, cluster := range config.Clusters {
		if usedClusters[clusterName] {
			filteredConfig.Clusters[clusterName] = cluster
		}
	}

	// Only keep authInfos that are referenced by remaining contexts
	for authInfoName, authInfo := range config.AuthInfos {
		if usedAuthInfos[authInfoName] {
			filteredConfig.AuthInfos[authInfoName] = authInfo
		}
	}

	// If no contexts remain, return nil
	if len(filteredConfig.Contexts) == 0 {
		return nil
	}

	return filteredConfig
}

// mergeConfigInto merges the source config into the destination config.
func (h *Handler) mergeConfigInto(dest, src *api.Config) {
	// Merge clusters
	for name, cluster := range src.Clusters {
		dest.Clusters[name] = cluster
	}

	// Merge contexts
	for name, context := range src.Contexts {
		dest.Contexts[name] = context
	}

	// Merge authInfos
	for name, authInfo := range src.AuthInfos {
		dest.AuthInfos[name] = authInfo
	}
}

// CleanupTempFiles removes temporary kubeconfig files.
func (h *Handler) CleanupTempFiles(ctx context.Context, paths []string) error {
	var errs []error

	for _, path := range paths {
		if err := h.fs.Remove(path); err != nil {
			if !os.IsNotExist(err) {
				errs = append(errs, fmt.Errorf("failed to remove %s: %w", path, err))
			}
		} else {
			h.logger.DebugContext(ctx, "Cleaned up temporary kubeconfig", "path", path)
		}
	}

	if len(errs) > 0 {
		return fmt.Errorf("failed to cleanup some files: %w", errors.Join(errs...))
	}

	return nil
}
