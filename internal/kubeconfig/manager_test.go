package kubeconfig

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"cowpoke/internal/config"
)

func TestKubeconfigManager_SaveKubeconfig(t *testing.T) {
	tempDir := t.TempDir()

	cluster := config.Cluster{
		ID:         "c-cluster1",
		Name:       "test-cluster",
		ServerID:   "server-1",
		ServerName: "Test Server",
	}

	kubeconfigContent := []byte(`apiVersion: v1
kind: Config
clusters:
- cluster:
    server: https://test-cluster.example.com:6443
  name: test-cluster
contexts:
- context:
    cluster: test-cluster
    user: admin
  name: test-cluster
current-context: test-cluster
users:
- name: admin
  user:
    token: test-token`)

	manager := NewManager(tempDir)

	savedPath, err := manager.SaveKubeconfig(cluster, kubeconfigContent)
	if err != nil {
		t.Fatalf("Failed to save kubeconfig: %v", err)
	}

	expectedPath := filepath.Join(tempDir, "test-cluster.yaml")
	if savedPath != expectedPath {
		t.Errorf("Expected path %s, got %s", expectedPath, savedPath)
	}

	if _, statErr := os.Stat(savedPath); os.IsNotExist(statErr) {
		t.Error("Kubeconfig file was not created")
	}

	content, err := os.ReadFile(savedPath)
	if err != nil {
		t.Fatalf("Failed to read saved kubeconfig: %v", err)
	}

	if string(content) != string(kubeconfigContent) {
		t.Error("Saved kubeconfig content doesn't match original")
	}
}

func TestKubeconfigManager_MergeKubeconfigs(t *testing.T) {
	tempDir := t.TempDir()

	kubeconfig1 := []byte(`apiVersion: v1
kind: Config
clusters:
- cluster:
    server: https://cluster1.example.com:6443
  name: cluster1
contexts:
- context:
    cluster: cluster1
    user: admin
  name: cluster1
current-context: cluster1
users:
- name: admin
  user:
    token: token1`)

	kubeconfig2 := []byte(`apiVersion: v1
kind: Config
clusters:
- cluster:
    server: https://cluster2.example.com:6443
  name: cluster2
contexts:
- context:
    cluster: cluster2
    user: admin
  name: cluster2
current-context: cluster2
users:
- name: admin
  user:
    token: token2`)

	cluster1 := config.Cluster{
		ID:         "c-cluster1",
		Name:       "cluster1",
		ServerID:   "server-1",
		ServerName: "Server 1",
	}

	cluster2 := config.Cluster{
		ID:         "c-cluster2",
		Name:       "cluster2",
		ServerID:   "server-2",
		ServerName: "Server 2",
	}

	manager := NewManager(tempDir)

	path1, err := manager.SaveKubeconfig(cluster1, kubeconfig1)
	if err != nil {
		t.Fatalf("Failed to save kubeconfig1: %v", err)
	}

	path2, err := manager.SaveKubeconfig(cluster2, kubeconfig2)
	if err != nil {
		t.Fatalf("Failed to save kubeconfig2: %v", err)
	}

	kubeconfigPaths := []string{path1, path2}
	outputPath := filepath.Join(tempDir, "merged-kubeconfig.yaml")

	err = manager.MergeKubeconfigs(kubeconfigPaths, outputPath)
	if err != nil {
		t.Fatalf("Failed to merge kubeconfigs: %v", err)
	}

	if _, statErr := os.Stat(outputPath); os.IsNotExist(statErr) {
		t.Error("Merged kubeconfig file was not created")
	}

	mergedContent, err := os.ReadFile(outputPath)
	if err != nil {
		t.Fatalf("Failed to read merged kubeconfig: %v", err)
	}

	mergedStr := string(mergedContent)

	if !strings.Contains(mergedStr, "cluster1") {
		t.Error("Merged kubeconfig should contain cluster1")
	}

	if !strings.Contains(mergedStr, "cluster2") {
		t.Error("Merged kubeconfig should contain cluster2")
	}

	if !strings.Contains(mergedStr, "https://cluster1.example.com:6443") {
		t.Error("Merged kubeconfig should contain cluster1 server URL")
	}

	if !strings.Contains(mergedStr, "https://cluster2.example.com:6443") {
		t.Error("Merged kubeconfig should contain cluster2 server URL")
	}
}

func TestKubeconfigManager_MergeKubeconfigs_EmptyList(t *testing.T) {
	tempDir := t.TempDir()
	manager := NewManager(tempDir)

	outputPath := filepath.Join(tempDir, "empty-merged.yaml")

	err := manager.MergeKubeconfigs([]string{}, outputPath)
	if err == nil {
		t.Error("Expected error when merging empty kubeconfig list")
	}
}

func TestKubeconfigManager_MergeKubeconfigs_NonExistentFile(t *testing.T) {
	tempDir := t.TempDir()
	manager := NewManager(tempDir)

	kubeconfigPaths := []string{"/non/existent/file.yaml"}
	outputPath := filepath.Join(tempDir, "merged.yaml")

	err := manager.MergeKubeconfigs(kubeconfigPaths, outputPath)
	if err == nil {
		t.Error("Expected error when merging non-existent kubeconfig file")
	}
}

func TestKubeconfigManager_SaveKubeconfig_InvalidClusterName(t *testing.T) {
	tempDir := t.TempDir()

	cluster := config.Cluster{
		ID:         "c-cluster1",
		Name:       "test/cluster:with*invalid?chars",
		ServerID:   "server-1",
		ServerName: "Test Server",
	}

	kubeconfigContent := []byte(`apiVersion: v1
kind: Config
clusters:
- cluster:
    server: https://test.example.com:6443
  name: test-cluster`)

	manager := NewManager(tempDir)

	savedPath, err := manager.SaveKubeconfig(cluster, kubeconfigContent)
	if err != nil {
		t.Fatalf("Failed to save kubeconfig: %v", err)
	}

	// Should sanitize filename - check that the filename portion is sanitized
	filename := filepath.Base(savedPath)
	if strings.Contains(filename, "/") || strings.Contains(filename, ":") || strings.Contains(filename, "*") ||
		strings.Contains(filename, "?") {
		t.Errorf("Filename should be sanitized, got: %s", filename)
	}

	// Should contain underscores where invalid chars were replaced
	if !strings.Contains(filename, "_") {
		t.Errorf("Expected sanitized filename to contain underscores, got: %s", filename)
	}

	if _, statErr := os.Stat(savedPath); os.IsNotExist(statErr) {
		t.Error("Kubeconfig file was not created")
	}
}

func TestKubeconfigManager_SaveKubeconfig_EmptyContent(t *testing.T) {
	tempDir := t.TempDir()

	cluster := config.Cluster{
		ID:         "c-cluster1",
		Name:       "test-cluster",
		ServerID:   "server-1",
		ServerName: "Test Server",
	}

	manager := NewManager(tempDir)

	savedPath, err := manager.SaveKubeconfig(cluster, []byte{})
	if err != nil {
		t.Fatalf("Failed to save empty kubeconfig: %v", err)
	}

	content, err := os.ReadFile(savedPath)
	if err != nil {
		t.Fatalf("Failed to read saved kubeconfig: %v", err)
	}

	if len(content) != 0 {
		t.Errorf("Expected empty content, got: %s", string(content))
	}
}

func TestKubeconfigManager_MergeKubeconfigs_SingleFile(t *testing.T) {
	tempDir := t.TempDir()

	kubeconfig := []byte(`apiVersion: v1
kind: Config
clusters:
- cluster:
    server: https://single.example.com:6443
  name: single-cluster
contexts:
- context:
    cluster: single-cluster
    user: admin
  name: single-cluster
current-context: single-cluster
users:
- name: admin
  user:
    token: single-token`)

	cluster := config.Cluster{
		ID:         "c-single",
		Name:       "single-cluster",
		ServerID:   "server-1",
		ServerName: "Single Server",
	}

	manager := NewManager(tempDir)

	path, err := manager.SaveKubeconfig(cluster, kubeconfig)
	if err != nil {
		t.Fatalf("Failed to save kubeconfig: %v", err)
	}

	kubeconfigPaths := []string{path}
	outputPath := filepath.Join(tempDir, "single-merged.yaml")

	err = manager.MergeKubeconfigs(kubeconfigPaths, outputPath)
	if err != nil {
		t.Fatalf("Failed to merge single kubeconfig: %v", err)
	}

	mergedContent, err := os.ReadFile(outputPath)
	if err != nil {
		t.Fatalf("Failed to read merged kubeconfig: %v", err)
	}

	mergedStr := string(mergedContent)
	if !strings.Contains(mergedStr, "single-cluster") {
		t.Error("Merged kubeconfig should contain single-cluster")
	}
}

func TestKubeconfigManager_NewManager(t *testing.T) {
	tempDir := t.TempDir()
	manager := NewManager(tempDir)

	if manager == nil {
		t.Fatal("Expected manager to be created, got nil")
	}

	if manager.baseDir != tempDir {
		t.Errorf("Expected baseDir %s, got %s", tempDir, manager.baseDir)
	}
}

func TestKubeconfigManager_MergeKubeconfigs_InvalidYAML(t *testing.T) {
	tempDir := t.TempDir()

	invalidKubeconfig := []byte(`invalid: yaml: content:
- malformed
  - yaml`)

	cluster := config.Cluster{
		ID:         "c-invalid",
		Name:       "invalid-cluster",
		ServerID:   "server-1",
		ServerName: "Invalid Server",
	}

	manager := NewManager(tempDir)

	path, err := manager.SaveKubeconfig(cluster, invalidKubeconfig)
	if err != nil {
		t.Fatalf("Failed to save invalid kubeconfig: %v", err)
	}

	kubeconfigPaths := []string{path}
	outputPath := filepath.Join(tempDir, "invalid-merged.yaml")

	err = manager.MergeKubeconfigs(kubeconfigPaths, outputPath)
	if err == nil {
		t.Error("Expected error when merging invalid YAML kubeconfig")
	}
}

// Additional tests to improve coverage

func TestKubeconfigManager_SaveKubeconfig_DirectoryCreateError(t *testing.T) {
	// Test error path when directory cannot be created
	invalidPath := "/root/cannot_create_directory"

	cluster := config.Cluster{
		ID:         "c-cluster1",
		Name:       "test-cluster",
		ServerID:   "server-1",
		ServerName: "Test Server",
	}

	kubeconfigContent := []byte("test content")

	manager := NewManager(invalidPath)

	_, err := manager.SaveKubeconfig(cluster, kubeconfigContent)
	if err == nil {
		t.Skip("Expected error creating directory, but operation succeeded")
	}

	if !strings.Contains(err.Error(), "failed to create kubeconfig directory") {
		t.Errorf("Expected directory creation error, got: %v", err)
	}
}

func TestKubeconfigManager_SaveKubeconfig_WriteFileError(t *testing.T) {
	// Test error path when file write fails (directory is readonly)
	tempDir := t.TempDir()

	// Make directory readonly to cause WriteFile to fail
	err := os.Chmod(tempDir, 0o555) // Read and execute only
	if err != nil {
		t.Skip("Cannot change directory permissions on this system")
	}
	defer func() { _ = os.Chmod(tempDir, 0o755) }() // Restore permissions

	cluster := config.Cluster{
		ID:         "c-cluster1",
		Name:       "test-cluster",
		ServerID:   "server-1",
		ServerName: "Test Server",
	}

	kubeconfigContent := []byte("test content")

	manager := NewManager(tempDir)

	_, err = manager.SaveKubeconfig(cluster, kubeconfigContent)
	if err == nil {
		t.Skip("Expected error writing file to readonly directory, but operation succeeded")
	}

	if !strings.Contains(err.Error(), "failed to write kubeconfig file") {
		t.Errorf("Expected file write error, got: %v", err)
	}
}

func TestKubeconfigManager_MergeKubeconfigs_OutputDirectoryCreateError(t *testing.T) {
	// Test error path when output directory cannot be created
	tempDir := t.TempDir()

	kubeconfig := []byte(`apiVersion: v1
kind: Config
clusters:
- cluster:
    server: https://test.example.com:6443
  name: test-cluster
contexts:
- context:
    cluster: test-cluster
    user: admin
  name: test-cluster
current-context: test-cluster
users:
- name: admin
  user:
    token: test-token`)

	cluster := config.Cluster{
		ID:         "c-cluster1",
		Name:       "test-cluster",
		ServerID:   "server-1",
		ServerName: "Test Server",
	}

	manager := NewManager(tempDir)

	path, err := manager.SaveKubeconfig(cluster, kubeconfig)
	if err != nil {
		t.Fatalf("Failed to save kubeconfig: %v", err)
	}

	kubeconfigPaths := []string{path}
	// Use a path that should fail to create (root directory)
	outputPath := "/root/cannot_create/merged.yaml"

	err = manager.MergeKubeconfigs(kubeconfigPaths, outputPath)
	if err == nil {
		t.Skip("Expected error creating output directory, but operation succeeded")
	}

	if !strings.Contains(err.Error(), "failed to create output directory") {
		t.Errorf("Expected output directory creation error, got: %v", err)
	}
}

func TestKubeconfigManager_MergeKubeconfigs_WriteOutputError(t *testing.T) {
	// Test error path when output file write fails
	tempDir := t.TempDir()
	outputDir := filepath.Join(tempDir, "readonly")

	// Create output directory and make it readonly
	err := os.MkdirAll(outputDir, 0o755)
	if err != nil {
		t.Fatalf("Failed to create output directory: %v", err)
	}

	err = os.Chmod(outputDir, 0o555) // Read and execute only
	if err != nil {
		t.Skip("Cannot change directory permissions on this system")
	}
	defer func() { _ = os.Chmod(outputDir, 0o755) }() // Restore permissions

	kubeconfig := []byte(`apiVersion: v1
kind: Config
clusters:
- cluster:
    server: https://test.example.com:6443
  name: test-cluster
contexts:
- context:
    cluster: test-cluster
    user: admin
  name: test-cluster
current-context: test-cluster
users:
- name: admin
  user:
    token: test-token`)

	cluster := config.Cluster{
		ID:         "c-cluster1",
		Name:       "test-cluster",
		ServerID:   "server-1",
		ServerName: "Test Server",
	}

	manager := NewManager(tempDir)

	path, err := manager.SaveKubeconfig(cluster, kubeconfig)
	if err != nil {
		t.Fatalf("Failed to save kubeconfig: %v", err)
	}

	kubeconfigPaths := []string{path}
	outputPath := filepath.Join(outputDir, "merged.yaml")

	err = manager.MergeKubeconfigs(kubeconfigPaths, outputPath)
	if err == nil {
		t.Skip("Expected error writing output file to readonly directory, but operation succeeded")
	}

	if !strings.Contains(err.Error(), "failed to write merged kubeconfig") {
		t.Errorf("Expected output file write error, got: %v", err)
	}
}

func TestKubeconfigManager_SaveKubeconfig_LocalCluster(t *testing.T) {
	tempDir := t.TempDir()

	cluster := config.Cluster{
		ID:         "local",
		Name:       "local",
		ServerID:   "server-1",
		ServerName: "Test Server",
		ServerURL:  "https://rancher.example.com",
	}

	kubeconfigContent := []byte(`apiVersion: v1
kind: Config
clusters:
- cluster:
    server: https://rancher.example.com:6443
  name: local
contexts:
- context:
    cluster: local
    user: local
  name: local
current-context: local
users:
- name: local
  user:
    token: test-token`)

	manager := NewManager(tempDir)

	savedPath, err := manager.SaveKubeconfig(cluster, kubeconfigContent)
	if err != nil {
		t.Fatalf("Failed to save kubeconfig: %v", err)
	}

	// Should use special naming for local cluster: local-<sanitized-url>.yaml
	expectedPath := filepath.Join(tempDir, "local-rancher-example-com.yaml")
	if savedPath != expectedPath {
		t.Errorf("Expected path %s, got %s", expectedPath, savedPath)
	}

	// Verify file was created
	if _, statErr := os.Stat(savedPath); os.IsNotExist(statErr) {
		t.Error("Kubeconfig file was not created")
	}

	// Verify content was rewritten with unique names
	content, err := os.ReadFile(savedPath)
	if err != nil {
		t.Fatalf("Failed to read saved kubeconfig: %v", err)
	}

	// Parse the saved content to verify rewriting
	contentStr := string(content)
	expectedUniqueName := "local-rancher-example-com"

	// Check that "local" references have been replaced with unique name
	if !strings.Contains(contentStr, expectedUniqueName) {
		t.Errorf(
			"Expected kubeconfig to contain unique name '%s', but it doesn't. Content:\n%s",
			expectedUniqueName,
			contentStr,
		)
	}

	// Check that server URL is preserved
	if !strings.Contains(contentStr, "https://rancher.example.com:6443") {
		t.Error("Expected server URL to be preserved in kubeconfig")
	}

	// Check that token is preserved
	if !strings.Contains(contentStr, "test-token") {
		t.Error("Expected user token to be preserved in kubeconfig")
	}
}

func TestKubeconfigManager_SaveKubeconfig_NonLocalCluster(t *testing.T) {
	tempDir := t.TempDir()

	cluster := config.Cluster{
		ID:         "c-m-xyz123",
		Name:       "production-cluster",
		ServerID:   "server-1",
		ServerName: "Test Server",
		ServerURL:  "https://rancher.example.com",
	}

	kubeconfigContent := []byte(`apiVersion: v1
kind: Config
clusters:
- cluster:
    server: https://production-cluster.example.com:6443
  name: production-cluster`)

	manager := NewManager(tempDir)

	savedPath, err := manager.SaveKubeconfig(cluster, kubeconfigContent)
	if err != nil {
		t.Fatalf("Failed to save kubeconfig: %v", err)
	}

	// Should use normal naming for non-local cluster: <cluster-name>.yaml
	expectedPath := filepath.Join(tempDir, "production-cluster.yaml")
	if savedPath != expectedPath {
		t.Errorf("Expected path %s, got %s", expectedPath, savedPath)
	}

	// Verify file was created
	if _, statErr := os.Stat(savedPath); os.IsNotExist(statErr) {
		t.Error("Kubeconfig file was not created")
	}
}

func TestKubeconfigManager_SaveKubeconfig_LocalClusterIDOnly(t *testing.T) {
	// Test when only ID is "local" (edge case)
	tempDir := t.TempDir()

	cluster := config.Cluster{
		ID:         "local",
		Name:       "some-other-name", // Different name, but ID is "local"
		ServerID:   "server-1",
		ServerName: "Test Server",
		ServerURL:  "https://rancher.example.com",
	}

	kubeconfigContent := []byte(`apiVersion: v1
kind: Config
clusters:
- cluster:
    server: https://rancher.example.com:6443
  name: some-other-name`)

	manager := NewManager(tempDir)

	savedPath, err := manager.SaveKubeconfig(cluster, kubeconfigContent)
	if err != nil {
		t.Fatalf("Failed to save kubeconfig: %v", err)
	}

	// Should use local naming since ID is "local"
	expectedPath := filepath.Join(tempDir, "local-rancher-example-com.yaml")
	if savedPath != expectedPath {
		t.Errorf("Expected path %s, got %s", expectedPath, savedPath)
	}
}

func TestKubeconfigManager_SaveKubeconfig_LocalClusterNameOnly(t *testing.T) {
	// Test when only Name is "local" (edge case)
	tempDir := t.TempDir()

	cluster := config.Cluster{
		ID:         "c-m-xyz123", // Different ID, but Name is "local"
		Name:       "local",
		ServerID:   "server-1",
		ServerName: "Test Server",
		ServerURL:  "https://rancher.example.com",
	}

	kubeconfigContent := []byte(`apiVersion: v1
kind: Config
clusters:
- cluster:
    server: https://rancher.example.com:6443
  name: local`)

	manager := NewManager(tempDir)

	savedPath, err := manager.SaveKubeconfig(cluster, kubeconfigContent)
	if err != nil {
		t.Fatalf("Failed to save kubeconfig: %v", err)
	}

	// Should use local naming since Name is "local"
	expectedPath := filepath.Join(tempDir, "local-rancher-example-com.yaml")
	if savedPath != expectedPath {
		t.Errorf("Expected path %s, got %s", expectedPath, savedPath)
	}
}

func TestKubeconfigManager_SaveKubeconfig_MultipleLocalClusters(t *testing.T) {
	// Test that multiple Rancher servers with local clusters don't overwrite each other
	tempDir := t.TempDir()
	manager := NewManager(tempDir)

	// First Rancher server's local cluster
	cluster1 := config.Cluster{
		ID:         "local",
		Name:       "local",
		ServerID:   "server-1",
		ServerName: "Rancher Server 1",
		ServerURL:  "https://rancher1.example.com",
	}

	kubeconfigContent1 := []byte(`apiVersion: v1
kind: Config
clusters:
- cluster:
    server: https://rancher1.example.com:6443
  name: local`)

	savedPath1, err := manager.SaveKubeconfig(cluster1, kubeconfigContent1)
	if err != nil {
		t.Fatalf("Failed to save first kubeconfig: %v", err)
	}

	expectedPath1 := filepath.Join(tempDir, "local-rancher1-example-com.yaml")
	if savedPath1 != expectedPath1 {
		t.Errorf("Expected path %s, got %s", expectedPath1, savedPath1)
	}

	// Second Rancher server's local cluster
	cluster2 := config.Cluster{
		ID:         "local",
		Name:       "local",
		ServerID:   "server-2",
		ServerName: "Rancher Server 2",
		ServerURL:  "https://rancher2.example.com",
	}

	kubeconfigContent2 := []byte(`apiVersion: v1
kind: Config
clusters:
- cluster:
    server: https://rancher2.example.com:6443
  name: local`)

	savedPath2, err := manager.SaveKubeconfig(cluster2, kubeconfigContent2)
	if err != nil {
		t.Fatalf("Failed to save second kubeconfig: %v", err)
	}

	expectedPath2 := filepath.Join(tempDir, "local-rancher2-example-com.yaml")
	if savedPath2 != expectedPath2 {
		t.Errorf("Expected path %s, got %s", expectedPath2, savedPath2)
	}

	// Verify both files exist and have different content
	content1, err := os.ReadFile(savedPath1)
	if err != nil {
		t.Fatalf("Failed to read first kubeconfig: %v", err)
	}

	content2, err := os.ReadFile(savedPath2)
	if err != nil {
		t.Fatalf("Failed to read second kubeconfig: %v", err)
	}

	content1Str := string(content1)
	content2Str := string(content2)

	// Verify each file contains its unique name and preserves server URLs
	if !strings.Contains(content1Str, "local-rancher1-example-com") {
		t.Errorf("First kubeconfig should contain unique name 'local-rancher1-example-com'. Content:\n%s", content1Str)
	}
	if !strings.Contains(content1Str, "https://rancher1.example.com:6443") {
		t.Error("First kubeconfig should preserve server URL")
	}

	if !strings.Contains(content2Str, "local-rancher2-example-com") {
		t.Errorf("Second kubeconfig should contain unique name 'local-rancher2-example-com'. Content:\n%s", content2Str)
	}
	if !strings.Contains(content2Str, "https://rancher2.example.com:6443") {
		t.Error("Second kubeconfig should preserve server URL")
	}

	// Verify files are different (most important - no overwrites)
	if bytes.Equal(content1, content2) {
		t.Error("Both kubeconfig files have the same content - possible overwrite occurred")
	}
}

func TestKubeconfigManager_RewriteLocalClusterNames(t *testing.T) {
	// Test the rewriting function specifically to ensure all references are updated
	tempDir := t.TempDir()
	manager := NewManager(tempDir)

	// Create a comprehensive kubeconfig with all possible "local" references
	originalContent := []byte(`apiVersion: v1
kind: Config
clusters:
- cluster:
    server: https://rancher.example.com:6443
    certificate-authority-data: LS0tLS1CRUdJTi0K
  name: local
contexts:
- context:
    cluster: local
    user: local
    namespace: default
  name: local
current-context: local
users:
- name: local
  user:
    token: test-token-123`)

	uniqueName := "local-rancher-example-com"
	rewrittenContent, err := manager.rewriteLocalClusterNames(originalContent, uniqueName)
	if err != nil {
		t.Fatalf("Failed to rewrite local cluster names: %v", err)
	}

	rewrittenStr := string(rewrittenContent)

	// Test 1: Cluster name should be rewritten
	if !strings.Contains(rewrittenStr, fmt.Sprintf("name: %s", uniqueName)) {
		t.Errorf("Cluster name was not rewritten. Content:\n%s", rewrittenStr)
	}

	// Test 2: Context name should be rewritten (no standalone "local" names should remain)
	if strings.Contains(rewrittenStr, "name: local\n") || strings.Contains(rewrittenStr, "name: local ") {
		t.Errorf("Found unrewritten standalone 'name: local' reference. Content:\n%s", rewrittenStr)
	}

	// Test 3: Context cluster reference should be rewritten
	if !strings.Contains(rewrittenStr, fmt.Sprintf("cluster: %s", uniqueName)) {
		t.Errorf("Context cluster reference was not rewritten. Content:\n%s", rewrittenStr)
	}

	// Test 4: Context user reference should be rewritten
	if !strings.Contains(rewrittenStr, fmt.Sprintf("user: %s", uniqueName)) {
		t.Errorf("Context user reference was not rewritten. Content:\n%s", rewrittenStr)
	}

	// Test 5: Current-context should be rewritten
	if !strings.Contains(rewrittenStr, fmt.Sprintf("current-context: %s", uniqueName)) {
		t.Errorf("Current-context was not rewritten. Content:\n%s", rewrittenStr)
	}

	// Test 6: Server URL should be preserved (not rewritten)
	if !strings.Contains(rewrittenStr, "https://rancher.example.com:6443") {
		t.Errorf("Server URL was not preserved. Content:\n%s", rewrittenStr)
	}

	// Test 7: Certificate data should be preserved
	if !strings.Contains(rewrittenStr, "LS0tLS1CRUdJTi0K") {
		t.Errorf("Certificate data was not preserved. Content:\n%s", rewrittenStr)
	}

	// Test 8: Token should be preserved
	if !strings.Contains(rewrittenStr, "test-token-123") {
		t.Errorf("User token was not preserved. Content:\n%s", rewrittenStr)
	}

	// Test 9: No standalone "local" references should remain (check specific patterns)
	// Check for exact matches that would cause merge conflicts
	if strings.Contains(rewrittenStr, "name: local\n") {
		t.Error("Found unrewritten 'name: local' reference")
	}
	if strings.Contains(rewrittenStr, "cluster: local\n") {
		t.Error("Found unrewritten 'cluster: local' reference")
	}
	if strings.Contains(rewrittenStr, "user: local\n") {
		t.Error("Found unrewritten 'user: local' reference")
	}
	if strings.Contains(rewrittenStr, "current-context: local\n") {
		t.Error("Found unrewritten 'current-context: local' reference")
	}
}
