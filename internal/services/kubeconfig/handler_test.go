package kubeconfig

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"cowpoke/internal/adapters/filesystem"
	"cowpoke/internal/domain"
	"cowpoke/internal/services/filter"
	"cowpoke/internal/testutil"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"k8s.io/client-go/tools/clientcmd"
	clientcmdapi "k8s.io/client-go/tools/clientcmd/api"
)

func TestHandler_MergeKubeconfigs_WithFiltering(t *testing.T) {
	tests := []struct {
		name              string
		inputKubeconfigs  []string // YAML content for each input file
		excludePatterns   []string
		expectedContexts  []string // context names that should remain
		expectedClusters  []string // cluster names that should remain
		expectedAuthInfos []string // user names that should remain
	}{
		{
			name: "filter mgmt contexts by context name",
			inputKubeconfigs: []string{
				// First kubeconfig with mgmt context
				`apiVersion: v1
kind: Config
clusters:
- cluster:
    server: https://mgmt-cluster.example.com
  name: mgmt-cluster
- cluster:
    server: https://app-cluster.example.com  
  name: app-cluster
contexts:
- context:
    cluster: mgmt-cluster
    user: admin
  name: us-west-2-prod-mgmt-a1
- context:
    cluster: app-cluster
    user: admin
  name: us-west-2-prod-app-a1
users:
- name: admin
  user:
    token: fake-token
current-context: us-west-2-prod-app-a1`,
			},
			excludePatterns:   []string{"mgmt"},
			expectedContexts:  []string{"us-west-2-prod-app-a1"},
			expectedClusters:  []string{"app-cluster"},
			expectedAuthInfos: []string{"admin"},
		},
		{
			name: "filter mgmt contexts by cluster reference",
			inputKubeconfigs: []string{
				`apiVersion: v1
kind: Config
clusters:
- cluster:
    server: https://mgmt-server.example.com
  name: mgmt-cluster
- cluster:
    server: https://app-server.example.com  
  name: app-cluster
contexts:
- context:
    cluster: mgmt-cluster
    user: admin
  name: prod-context-1
- context:
    cluster: app-cluster
    user: admin
  name: prod-context-2
users:
- name: admin
  user:
    token: fake-token`,
			},
			excludePatterns:   []string{"mgmt"},
			expectedContexts:  []string{"prod-context-2"},
			expectedClusters:  []string{"app-cluster"},
			expectedAuthInfos: []string{"admin"},
		},
		{
			name: "multi-pattern filtering",
			inputKubeconfigs: []string{
				`apiVersion: v1
kind: Config
clusters:
- cluster:
    server: https://mgmt.example.com
  name: mgmt-cluster
- cluster:
    server: https://admin.example.com  
  name: admin-cluster
- cluster:
    server: https://app.example.com  
  name: app-cluster
contexts:
- context:
    cluster: mgmt-cluster
    user: user1
  name: prod-mgmt-context
- context:
    cluster: admin-cluster
    user: user2
  name: prod-admin-context
- context:
    cluster: app-cluster
    user: user3
  name: prod-app-context
users:
- name: user1
  user:
    token: token1
- name: user2
  user:
    token: token2
- name: user3
  user:
    token: token3`,
			},
			excludePatterns:   []string{"mgmt", "admin"},
			expectedContexts:  []string{"prod-app-context"},
			expectedClusters:  []string{"app-cluster"},
			expectedAuthInfos: []string{"user3"},
		},
		{
			name: "multiple kubeconfigs with filtering",
			inputKubeconfigs: []string{
				// First kubeconfig
				`apiVersion: v1
kind: Config
clusters:
- cluster:
    server: https://server1.example.com
  name: cluster1
contexts:
- context:
    cluster: cluster1
    user: user1
  name: us-west-2-prod-mgmt-a1
users:
- name: user1
  user:
    token: token1`,
				// Second kubeconfig
				`apiVersion: v1
kind: Config
clusters:
- cluster:
    server: https://server2.example.com
  name: cluster2
contexts:
- context:
    cluster: cluster2
    user: user2
  name: us-west-2-prod-app-a1
users:
- name: user2
  user:
    token: token2`,
			},
			excludePatterns:   []string{"mgmt"},
			expectedContexts:  []string{"us-west-2-prod-app-a1"},
			expectedClusters:  []string{"cluster2"},
			expectedAuthInfos: []string{"user2"},
		},
		{
			name: "no filtering when no patterns match",
			inputKubeconfigs: []string{
				`apiVersion: v1
kind: Config
clusters:
- cluster:
    server: https://prod.example.com
  name: prod-cluster
- cluster:
    server: https://staging.example.com  
  name: staging-cluster
contexts:
- context:
    cluster: prod-cluster
    user: admin
  name: prod-context
- context:
    cluster: staging-cluster
    user: admin
  name: staging-context
users:
- name: admin
  user:
    token: fake-token`,
			},
			excludePatterns:   []string{"mgmt"},
			expectedContexts:  []string{"prod-context", "staging-context"},
			expectedClusters:  []string{"prod-cluster", "staging-cluster"},
			expectedAuthInfos: []string{"admin"},
		},
		{
			name: "complex regex pattern filtering",
			inputKubeconfigs: []string{
				`apiVersion: v1
kind: Config
clusters:
- cluster:
    server: https://server1.example.com
  name: cluster1
- cluster:
    server: https://server2.example.com
  name: cluster2
- cluster:
    server: https://server3.example.com
  name: cluster3
contexts:
- context:
    cluster: cluster1
    user: user1
  name: gke-prod-us-central1-mgmt
- context:
    cluster: cluster2
    user: user2
  name: gke-prod-us-central1-app
- context:
    cluster: cluster3
    user: user3
  name: other-prod-mgmt
users:
- name: user1
  user:
    token: token1
- name: user2
  user:
    token: token2
- name: user3
  user:
    token: token3`,
			},
			excludePatterns:   []string{"gke.*mgmt"},
			expectedContexts:  []string{"gke-prod-us-central1-app", "other-prod-mgmt"},
			expectedClusters:  []string{"cluster2", "cluster3"},
			expectedAuthInfos: []string{"user2", "user3"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Setup temporary directory
			tempDir := t.TempDir()

			// Create input kubeconfig files
			var inputPaths []string
			for i, content := range tt.inputKubeconfigs {
				path := filepath.Join(tempDir, "input-"+string(rune('a'+i))+".yaml")
				writeErr := os.WriteFile(path, []byte(content), 0o600)
				require.NoError(t, writeErr)
				inputPaths = append(inputPaths, path)
			}

			outputPath := filepath.Join(tempDir, "merged-output.yaml")

			// Create handler and filter
			fs := filesystem.New()
			handler, err := NewHandler(fs, tempDir, testutil.Logger())
			require.NoError(t, err)

			var clusterFilter domain.ClusterFilter
			if len(tt.excludePatterns) > 0 {
				excludeFilter, filterErr := filter.NewExcludeFilter(tt.excludePatterns, testutil.Logger())
				require.NoError(t, filterErr)
				clusterFilter = excludeFilter
			} else {
				clusterFilter = filter.NewNoOpFilter()
			}

			// Execute merge with filtering
			ctx := context.Background()
			err = handler.MergeKubeconfigs(ctx, inputPaths, outputPath, clusterFilter)
			require.NoError(t, err)

			// Load and verify the merged result
			config, err := clientcmd.LoadFromFile(outputPath)
			require.NoError(t, err)

			// Verify contexts
			var actualContexts []string
			for name := range config.Contexts {
				actualContexts = append(actualContexts, name)
			}
			assert.ElementsMatch(t, tt.expectedContexts, actualContexts,
				"Contexts don't match expected after filtering")

			// Verify clusters
			var actualClusters []string
			for name := range config.Clusters {
				actualClusters = append(actualClusters, name)
			}
			assert.ElementsMatch(t, tt.expectedClusters, actualClusters,
				"Clusters don't match expected after filtering")

			// Verify auth infos
			var actualAuthInfos []string
			for name := range config.AuthInfos {
				actualAuthInfos = append(actualAuthInfos, name)
			}
			assert.ElementsMatch(t, tt.expectedAuthInfos, actualAuthInfos,
				"Auth infos don't match expected after filtering")

			// Verify all remaining contexts are valid (reference existing clusters and users)
			for contextName, contextInfo := range config.Contexts {
				assert.Contains(t, config.Clusters, contextInfo.Cluster,
					"Context %q references non-existent cluster %q", contextName, contextInfo.Cluster)
				assert.Contains(t, config.AuthInfos, contextInfo.AuthInfo,
					"Context %q references non-existent user %q", contextName, contextInfo.AuthInfo)
			}
		})
	}
}

func TestHandler_MergeKubeconfigs_UserReportedCase(t *testing.T) {
	// This tests the specific case reported by the user where 'mgmt' clusters
	// should be excluded but were still appearing in the merged kubeconfig

	tempDir := t.TempDir()

	// Simulate a Rancher kubeconfig that contains multiple clusters
	// This is the key insight: Rancher returns kubeconfigs with multiple clusters!
	rancherKubeconfig := `apiVersion: v1
kind: Config
clusters:
- cluster:
    server: https://rancher.example.com/k8s/clusters/c-mgmt123
  name: us-west-2-prod-mgmt-a1
- cluster:
    server: https://rancher.example.com/k8s/clusters/c-app456
  name: us-west-2-prod-app-a1
- cluster:
    server: https://rancher.example.com/k8s/clusters/c-web789
  name: us-west-2-prod-web-a1
contexts:
- context:
    cluster: us-west-2-prod-mgmt-a1
    user: user-mgmt
  name: us-west-2-prod-mgmt-a1
- context:
    cluster: us-west-2-prod-app-a1
    user: user-app
  name: us-west-2-prod-app-a1
- context:
    cluster: us-west-2-prod-web-a1
    user: user-web
  name: us-west-2-prod-web-a1
users:
- name: user-mgmt
  user:
    token: mgmt-token
- name: user-app
  user:
    token: app-token
- name: user-web
  user:
    token: web-token
current-context: us-west-2-prod-app-a1`

	// Write the kubeconfig file
	inputPath := filepath.Join(tempDir, "rancher-multi-cluster.yaml")
	writeErr := os.WriteFile(inputPath, []byte(rancherKubeconfig), 0o600)
	require.NoError(t, writeErr)

	outputPath := filepath.Join(tempDir, "merged.yaml")

	// Create handler and exclude filter for 'mgmt'
	fs := filesystem.New()
	handler, err := NewHandler(fs, tempDir, testutil.Logger())
	require.NoError(t, err)
	excludeFilter, filterErr := filter.NewExcludeFilter([]string{"mgmt"}, testutil.Logger())
	require.NoError(t, filterErr)

	// Execute merge with filtering
	ctx := context.Background()
	mergeErr := handler.MergeKubeconfigs(ctx, []string{inputPath}, outputPath, excludeFilter)
	require.NoError(t, mergeErr)

	// Load and verify the result
	config, loadErr := clientcmd.LoadFromFile(outputPath)
	require.NoError(t, loadErr)

	// Verify that mgmt context is excluded
	_, mgmtContextExists := config.Contexts["us-west-2-prod-mgmt-a1"]
	assert.False(t, mgmtContextExists, "Management context should be excluded")

	// Verify that app and web contexts remain
	_, appContextExists := config.Contexts["us-west-2-prod-app-a1"]
	assert.True(t, appContextExists, "App context should be included")

	_, webContextExists := config.Contexts["us-west-2-prod-web-a1"]
	assert.True(t, webContextExists, "Web context should be included")

	// Verify that mgmt cluster is cleaned up
	_, mgmtClusterExists := config.Clusters["us-west-2-prod-mgmt-a1"]
	assert.False(t, mgmtClusterExists, "Management cluster should be cleaned up")

	// Verify that mgmt user is cleaned up
	_, mgmtUserExists := config.AuthInfos["user-mgmt"]
	assert.False(t, mgmtUserExists, "Management user should be cleaned up")

	t.Logf("Verified that mgmt cluster filtering works correctly with multi-cluster kubeconfigs")
}

func TestHandler_applyFilterToConfig_CleanupUnusedResources(t *testing.T) {
	// Test that clusters and users are properly cleaned up when no contexts reference them

	tempDir := t.TempDir()

	fs := filesystem.New()
	handler, handlerErr := NewHandler(fs, tempDir, testutil.Logger())
	require.NoError(t, handlerErr)
	excludeFilter, filterErr := filter.NewExcludeFilter([]string{"mgmt"}, testutil.Logger())
	require.NoError(t, filterErr)

	// Create a config where filtering removes all contexts that reference certain clusters/users
	config := &clientcmdapi.Config{
		Clusters: map[string]*clientcmdapi.Cluster{
			"mgmt-cluster":   {Server: "https://mgmt.example.com"},
			"app-cluster":    {Server: "https://app.example.com"},
			"unused-cluster": {Server: "https://unused.example.com"}, // No contexts reference this
		},
		Contexts: map[string]*clientcmdapi.Context{
			"mgmt-context": {Cluster: "mgmt-cluster", AuthInfo: "mgmt-user"}, // Will be filtered out
			"app-context":  {Cluster: "app-cluster", AuthInfo: "app-user"},   // Will remain
		},
		AuthInfos: map[string]*clientcmdapi.AuthInfo{
			"mgmt-user":   {Token: "mgmt-token"},
			"app-user":    {Token: "app-token"},
			"unused-user": {Token: "unused-token"}, // No contexts reference this
		},
	}

	ctx := context.Background()
	filteredConfig := handler.applyFilterToConfig(ctx, config, excludeFilter)

	// Verify mgmt context was filtered out
	assert.NotContains(t, filteredConfig.Contexts, "mgmt-context")
	assert.Contains(t, filteredConfig.Contexts, "app-context")

	// Verify unused clusters are cleaned up
	assert.NotContains(t, filteredConfig.Clusters, "mgmt-cluster", "Referenced cluster should be cleaned up")
	assert.NotContains(t, filteredConfig.Clusters, "unused-cluster", "Unreferenced cluster should be cleaned up")
	assert.Contains(t, filteredConfig.Clusters, "app-cluster", "Referenced cluster should remain")

	// Verify unused users are cleaned up
	assert.NotContains(t, filteredConfig.AuthInfos, "mgmt-user", "Referenced user should be cleaned up")
	assert.NotContains(t, filteredConfig.AuthInfos, "unused-user", "Unreferenced user should be cleaned up")
	assert.Contains(t, filteredConfig.AuthInfos, "app-user", "Referenced user should remain")
}

func TestHandler_MergeKubeconfigs_NoFilter(t *testing.T) {
	// Test that merge works correctly with NoOpFilter (no filtering)

	tempDir := t.TempDir()

	kubeconfig := `apiVersion: v1
kind: Config
clusters:
- cluster:
    server: https://mgmt.example.com
  name: mgmt-cluster
- cluster:
    server: https://app.example.com
  name: app-cluster
contexts:
- context:
    cluster: mgmt-cluster
    user: user1
  name: mgmt-context
- context:
    cluster: app-cluster
    user: user2
  name: app-context
users:
- name: user1
  user:
    token: token1
- name: user2
  user:
    token: token2`

	inputPath := filepath.Join(tempDir, "input.yaml")
	writeErr := os.WriteFile(inputPath, []byte(kubeconfig), 0o600)
	require.NoError(t, writeErr)

	outputPath := filepath.Join(tempDir, "output.yaml")

	fs := filesystem.New()
	handler, err := NewHandler(fs, tempDir, testutil.Logger())
	require.NoError(t, err)
	noOpFilter := filter.NewNoOpFilter()

	ctx := context.Background()
	mergeErr := handler.MergeKubeconfigs(ctx, []string{inputPath}, outputPath, noOpFilter)
	require.NoError(t, mergeErr)

	// Verify all contexts remain when no filtering is applied
	config, loadErr := clientcmd.LoadFromFile(outputPath)
	require.NoError(t, loadErr)

	assert.Len(t, config.Contexts, 2)
	assert.Contains(t, config.Contexts, "mgmt-context")
	assert.Contains(t, config.Contexts, "app-context")
}
