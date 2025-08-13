package kubeconfig

import (
	"fmt"
	"os"
	"path/filepath"

	"cowpoke/internal/config"
	"cowpoke/internal/utils"

	"k8s.io/client-go/tools/clientcmd"
	clientcmdapi "k8s.io/client-go/tools/clientcmd/api"
	"sigs.k8s.io/yaml"
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
	if err := os.MkdirAll(m.baseDir, 0755); err != nil {
		return "", fmt.Errorf("failed to create kubeconfig directory: %w", err)
	}

	var filename string
	var processedContent []byte
	
	if cluster.ID == "local" || cluster.Name == "local" {
		// Special handling for Rancher local clusters (ID and Name are both "local")
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

	err := os.WriteFile(filePath, processedContent, 0644)
	if err != nil {
		return "", fmt.Errorf("failed to write kubeconfig file: %w", err)
	}

	return filePath, nil
}

// rewriteLocalClusterNames rewrites all "local" references in kubeconfig content
// to use unique names while preserving server URLs and credentials
func (m *Manager) rewriteLocalClusterNames(kubeconfigContent []byte, uniqueName string) ([]byte, error) {
	// Parse the kubeconfig
	var kubeconfigData map[string]interface{}
	if err := yaml.Unmarshal(kubeconfigContent, &kubeconfigData); err != nil {
		return nil, fmt.Errorf("failed to parse kubeconfig: %w", err)
	}

	// Update clusters section: clusters[].name (preserve server URLs)
	if clusters, ok := kubeconfigData["clusters"].([]interface{}); ok {
		for _, clusterInterface := range clusters {
			if cluster, ok := clusterInterface.(map[string]interface{}); ok {
				if name, exists := cluster["name"]; exists && name == "local" {
					cluster["name"] = uniqueName
				}
			}
		}
	}

	// Update users section: users[].name (preserve credentials)
	if users, ok := kubeconfigData["users"].([]interface{}); ok {
		for _, userInterface := range users {
			if user, ok := userInterface.(map[string]interface{}); ok {
				if name, exists := user["name"]; exists && name == "local" {
					user["name"] = uniqueName
				}
			}
		}
	}

	// Update contexts section: contexts[].name and nested references
	if contexts, ok := kubeconfigData["contexts"].([]interface{}); ok {
		for _, contextInterface := range contexts {
			if context, ok := contextInterface.(map[string]interface{}); ok {
				// Update context name: contexts[].name
				if name, exists := context["name"]; exists && name == "local" {
					context["name"] = uniqueName
				}
				// Update nested context references: contexts[].context.{cluster,user}
				if contextData, ok := context["context"].(map[string]interface{}); ok {
					if cluster, exists := contextData["cluster"]; exists && cluster == "local" {
						contextData["cluster"] = uniqueName
					}
					if user, exists := contextData["user"]; exists && user == "local" {
						contextData["user"] = uniqueName
					}
				}
			}
		}
	}

	// Update current-context reference
	if currentContext, exists := kubeconfigData["current-context"]; exists && currentContext == "local" {
		kubeconfigData["current-context"] = uniqueName
	}

	// Marshal back to YAML
	rewrittenContent, err := yaml.Marshal(kubeconfigData)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal rewritten kubeconfig: %w", err)
	}

	return rewrittenContent, nil
}

func (m *Manager) MergeKubeconfigs(kubeconfigPaths []string, outputPath string) error {
	if len(kubeconfigPaths) == 0 {
		return fmt.Errorf("no kubeconfig files to merge")
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

	if err := os.MkdirAll(filepath.Dir(outputPath), 0755); err != nil {
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
