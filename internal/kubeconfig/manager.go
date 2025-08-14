package kubeconfig

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"cowpoke/internal/config"
	"cowpoke/internal/utils"

	"k8s.io/client-go/tools/clientcmd"
	clientcmdapi "k8s.io/client-go/tools/clientcmd/api"
	"sigs.k8s.io/yaml"
)

const (
	// LocalClusterName is the identifier used by Rancher for local clusters.
	LocalClusterName = "local"
)

type Manager struct {
	baseDir string
}

func NewManager(baseDir string) *Manager {
	return &Manager{
		baseDir: baseDir,
	}
}

func (m *Manager) SaveKubeconfig(cluster config.Cluster, kubeconfigContent []byte) (string, error) {
	if err := os.MkdirAll(m.baseDir, 0o750); err != nil {
		return "", fmt.Errorf("failed to create kubeconfig directory: %w", err)
	}

	var filename string
	var processedContent []byte

	if cluster.ID == LocalClusterName || cluster.Name == LocalClusterName {
		// Special handling for Rancher local clusters (ID and Name are both LocalClusterName)
		sanitizedURL := utils.SanitizeURL(cluster.ServerURL)
		uniqueName := fmt.Sprintf("local-%s", sanitizedURL)
		filename = fmt.Sprintf("%s.yaml", uniqueName)

		// Rewrite kubeconfig content to use unique names while preserving server URLs
		var err error
		processedContent, err = m.rewriteLocalClusterNames(kubeconfigContent, uniqueName)
		if err != nil {
			return "", fmt.Errorf("failed to rewrite local cluster names: %w", err)
		}
	} else {
		filename = fmt.Sprintf("%s.yaml", utils.SanitizeFilename(cluster.Name))
		processedContent = kubeconfigContent
	}

	filePath := filepath.Join(m.baseDir, filename)

	err := os.WriteFile(filePath, processedContent, 0o600)
	if err != nil {
		return "", fmt.Errorf("failed to write kubeconfig file: %w", err)
	}

	return filePath, nil
}

// rewriteLocalClusterNames rewrites all "local" references in kubeconfig content
// to use unique names while preserving server URLs and credentials.
func (m *Manager) rewriteLocalClusterNames(kubeconfigContent []byte, uniqueName string) ([]byte, error) {
	var kubeconfigData map[string]any
	if err := yaml.Unmarshal(kubeconfigContent, &kubeconfigData); err != nil {
		return nil, fmt.Errorf("failed to parse kubeconfig: %w", err)
	}

	m.updateClusters(kubeconfigData, uniqueName)
	m.updateUsers(kubeconfigData, uniqueName)
	m.updateContexts(kubeconfigData, uniqueName)
	m.updateCurrentContext(kubeconfigData, uniqueName)

	rewrittenContent, err := yaml.Marshal(kubeconfigData)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal rewritten kubeconfig: %w", err)
	}

	return rewrittenContent, nil
}

// updateClusters updates cluster names in the clusters section.
func (m *Manager) updateClusters(kubeconfigData map[string]any, uniqueName string) {
	if clusters, ok := kubeconfigData["clusters"].([]any); ok {
		for _, clusterInterface := range clusters {
			if cluster, clusterOk := clusterInterface.(map[string]any); clusterOk {
				if name, exists := cluster["name"]; exists && name == LocalClusterName {
					cluster["name"] = uniqueName
				}
			}
		}
	}
}

// updateUsers updates user names in the users section.
func (m *Manager) updateUsers(kubeconfigData map[string]any, uniqueName string) {
	if users, ok := kubeconfigData["users"].([]any); ok {
		for _, userInterface := range users {
			if user, userOk := userInterface.(map[string]any); userOk {
				if name, exists := user["name"]; exists && name == LocalClusterName {
					user["name"] = uniqueName
				}
			}
		}
	}
}

// updateContexts updates context names and nested references in the contexts section.
func (m *Manager) updateContexts(kubeconfigData map[string]any, uniqueName string) {
	if contexts, ok := kubeconfigData["contexts"].([]any); ok {
		for _, contextInterface := range contexts {
			if context, contextOk := contextInterface.(map[string]any); contextOk {
				if name, exists := context["name"]; exists && name == LocalClusterName {
					context["name"] = uniqueName
				}
				m.updateContextReferences(context, uniqueName)
			}
		}
	}
}

// updateContextReferences updates cluster and user references within a context.
func (m *Manager) updateContextReferences(context map[string]any, uniqueName string) {
	if contextData, contextDataOk := context["context"].(map[string]any); contextDataOk {
		if cluster, exists := contextData["cluster"]; exists && cluster == LocalClusterName {
			contextData["cluster"] = uniqueName
		}
		if user, exists := contextData["user"]; exists && user == LocalClusterName {
			contextData["user"] = uniqueName
		}
	}
}

// updateCurrentContext updates the current-context reference.
func (m *Manager) updateCurrentContext(kubeconfigData map[string]any, uniqueName string) {
	if currentContext, exists := kubeconfigData["current-context"]; exists && currentContext == LocalClusterName {
		kubeconfigData["current-context"] = uniqueName
	}
}

func (m *Manager) MergeKubeconfigs(kubeconfigPaths []string, outputPath string) error {
	if len(kubeconfigPaths) == 0 {
		return errors.New("no kubeconfig files to merge")
	}

	var mergedConfig *clientcmdapi.Config

	for i, path := range kubeconfigPaths {
		if _, err := os.Stat(path); os.IsNotExist(err) {
			return fmt.Errorf("kubeconfig file does not exist: %s", path)
		}

		config, err := clientcmd.LoadFromFile(path)
		if err != nil {
			return fmt.Errorf("failed to load kubeconfig from %s: %w", path, err)
		}

		if i == 0 {
			mergedConfig = config
		} else {
			mergedConfig = mergeConfigs(mergedConfig, config)
		}
	}

	if err := os.MkdirAll(filepath.Dir(outputPath), 0o750); err != nil {
		return fmt.Errorf("failed to create output directory: %w", err)
	}

	err := clientcmd.WriteToFile(*mergedConfig, outputPath)
	if err != nil {
		return fmt.Errorf("failed to write merged kubeconfig: %w", err)
	}

	return nil
}

func mergeConfigs(base, additional *clientcmdapi.Config) *clientcmdapi.Config {
	merged := base.DeepCopy()

	for name, cluster := range additional.Clusters {
		merged.Clusters[name] = cluster
	}

	for name, context := range additional.Contexts {
		merged.Contexts[name] = context
	}

	for name, user := range additional.AuthInfos {
		merged.AuthInfos[name] = user
	}

	return merged
}
