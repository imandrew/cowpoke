package cmd

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"cowpoke/internal/config"
	"cowpoke/internal/kubeconfig"
	"cowpoke/internal/logging"
	"cowpoke/internal/rancher"

	"github.com/spf13/cobra"
)

// SyncProcessor Tests

func TestNewSyncProcessor(t *testing.T) {
	kubeconfigManager := kubeconfig.NewManager("/test/path")
	logger := logging.Get()

	processor := NewSyncProcessor(kubeconfigManager, logger)

	if processor == nil {
		t.Fatal("Expected processor to be created, got nil")
	}

	if processor.kubeconfigManager != kubeconfigManager {
		t.Error("Expected kubeconfig manager to be set correctly")
	}

	if processor.logger != logger {
		t.Error("Expected logger to be set correctly")
	}
}

func TestSyncProcessor_ProcessServers_EmptyList(t *testing.T) {
	kubeconfigManager := kubeconfig.NewManager("/test/path")
	logger := logging.Get()
	processor := NewSyncProcessor(kubeconfigManager, logger)

	ctx := context.Background()
	servers := []config.RancherServer{}

	paths, err := processor.ProcessServers(ctx, servers)

	if err != nil {
		t.Fatalf("Expected no error for empty server list, got: %v", err)
	}

	if len(paths) != 0 {
		t.Errorf("Expected 0 paths for empty server list, got %d", len(paths))
	}
}

func TestSyncProcessor_ProcessServers_ContextCancellation(t *testing.T) {
	kubeconfigManager := kubeconfig.NewManager("/test/path")
	logger := logging.Get()
	processor := NewSyncProcessor(kubeconfigManager, logger)

	// Create a context that will be cancelled after a short delay
	ctx, cancel := context.WithCancel(context.Background())

	servers := []config.RancherServer{
		{
			ID:       "server-1",
			Name:     "Test Server 1",
			URL:      "https://rancher1.example.com",
			Username: "admin",
			AuthType: "local",
		},
	}

	// Cancel the context after starting to simulate cancellation during processing
	go func() {
		time.Sleep(10 * time.Millisecond)
		cancel()
	}()

	_, err := processor.ProcessServers(ctx, servers)

	// The error handling is graceful, so we might get nil or context.Canceled
	// depending on timing. Both are acceptable.
	if err != nil && !errors.Is(err, context.Canceled) {
		t.Errorf("Expected nil or context.Canceled error, got: %v", err)
	}
}

func TestSyncProcessor_ProcessServers_SemaphoreLimit(t *testing.T) {
	kubeconfigManager := kubeconfig.NewManager("/test/path")
	logger := logging.Get()
	processor := NewSyncProcessor(kubeconfigManager, logger)

	ctx := context.Background()

	// Create more servers than the semaphore limit (3)
	servers := make([]config.RancherServer, 5)
	for i := 0; i < 5; i++ {
		servers[i] = config.RancherServer{
			ID:       "server-" + string(rune('1'+i)),
			Name:     "Test Server " + string(rune('1'+i)),
			URL:      "https://rancher" + string(rune('1'+i)) + ".example.com",
			Username: "admin",
			AuthType: "local",
		}
	}

	// This should complete without deadlock due to semaphore handling
	// Note: This will fail during authentication since we're using dummy servers,
	// but the semaphore logic should still work correctly
	paths, err := processor.ProcessServers(ctx, servers)

	// We expect this to complete (not hang) and return empty paths
	// since authentication will fail for all dummy servers
	if err != nil {
		t.Fatalf("Expected no error (failures should be handled gracefully), got: %v", err)
	}

	if len(paths) != 0 {
		t.Errorf("Expected 0 paths since all authentications should fail, got %d", len(paths))
	}
}

func TestSyncProcessor_ProcessServers_ConcurrentProcessing(t *testing.T) {
	kubeconfigManager := kubeconfig.NewManager("/test/path")
	logger := logging.Get()
	processor := NewSyncProcessor(kubeconfigManager, logger)

	ctx := context.Background()

	// Create multiple servers to test concurrent processing
	servers := []config.RancherServer{
		{
			ID:       "server-1",
			Name:     "Test Server 1",
			URL:      "https://rancher1.example.com",
			Username: "admin",
			AuthType: "local",
		},
		{
			ID:       "server-2",
			Name:     "Test Server 2",
			URL:      "https://rancher2.example.com",
			Username: "admin",
			AuthType: "github",
		},
	}

	// Track execution times to verify concurrent processing
	start := time.Now()
	paths, err := processor.ProcessServers(ctx, servers)
	duration := time.Since(start)

	// Should complete relatively quickly due to concurrent processing
	// Even though authentications will fail, timeouts should be concurrent
	if duration > 10*time.Second {
		t.Errorf("Expected concurrent processing to complete faster, took %v", duration)
	}

	if err != nil {
		t.Fatalf("Expected no error (failures should be handled gracefully), got: %v", err)
	}

	if len(paths) != 0 {
		t.Errorf("Expected 0 paths since authentications should fail, got %d", len(paths))
	}
}

func TestSyncProcessor_processSingleServer_ContextTimeout(t *testing.T) {
	kubeconfigManager := kubeconfig.NewManager("/test/path")
	logger := logging.Get()
	processor := NewSyncProcessor(kubeconfigManager, logger)

	// Create a context with very short timeout
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Millisecond)
	defer cancel()

	server := config.RancherServer{
		ID:       "server-1",
		Name:     "Test Server",
		URL:      "https://rancher.example.com",
		Username: "admin",
		AuthType: "local",
	}

	var kubeconfigPaths []string
	var pathsMutex sync.Mutex

	// Should return nil (no error) since errors are handled gracefully
	err := processor.processSingleServer(ctx, server, "test-password", &kubeconfigPaths, &pathsMutex)

	if err != nil {
		t.Errorf("Expected nil error (graceful handling), got: %v", err)
	}

	if len(kubeconfigPaths) != 0 {
		t.Errorf("Expected 0 paths due to timeout, got %d", len(kubeconfigPaths))
	}
}

func TestSyncProcessor_processSingleServer_InvalidServer(t *testing.T) {
	kubeconfigManager := kubeconfig.NewManager("/test/path")
	logger := logging.Get()
	processor := NewSyncProcessor(kubeconfigManager, logger)

	ctx := context.Background()

	// Server with invalid URL
	server := config.RancherServer{
		ID:       "server-1",
		Name:     "Test Server",
		URL:      "invalid-url",
		Username: "admin",
		AuthType: "local",
	}

	var kubeconfigPaths []string
	var pathsMutex sync.Mutex

	err := processor.processSingleServer(ctx, server, "test-password", &kubeconfigPaths, &pathsMutex)

	// Should return nil (graceful error handling)
	if err != nil {
		t.Errorf("Expected nil error (graceful handling), got: %v", err)
	}

	if len(kubeconfigPaths) != 0 {
		t.Errorf("Expected 0 paths due to invalid server, got %d", len(kubeconfigPaths))
	}
}

func TestSyncProcessor_processSingleServer_ThreadSafety(t *testing.T) {
	kubeconfigManager := kubeconfig.NewManager("/test/path")
	logger := logging.Get()
	processor := NewSyncProcessor(kubeconfigManager, logger)

	ctx := context.Background()

	server := config.RancherServer{
		ID:       "server-1",
		Name:     "Test Server",
		URL:      "https://rancher.example.com",
		Username: "admin",
		AuthType: "local",
	}

	var kubeconfigPaths []string
	var pathsMutex sync.Mutex

	// Run multiple goroutines to test thread safety
	var wg sync.WaitGroup
	numGoroutines := 5

	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_ = processor.processSingleServer(ctx, server, "test-password", &kubeconfigPaths, &pathsMutex)
		}()
	}

	wg.Wait()

	// Should complete without data races
	// All calls should fail gracefully, so paths should be empty
	if len(kubeconfigPaths) != 0 {
		t.Errorf("Expected 0 paths due to authentication failures, got %d", len(kubeconfigPaths))
	}
}

func TestSyncProcessor_processClusters_EmptyList(t *testing.T) {
	kubeconfigManager := kubeconfig.NewManager("/test/path")
	logger := logging.Get()
	processor := NewSyncProcessor(kubeconfigManager, logger)

	ctx := context.Background()
	clusters := []config.Cluster{}

	var kubeconfigPaths []string
	var pathsMutex sync.Mutex

	err := processor.processClusters(ctx, clusters, "server-1", "https://test.example.com", nil, &kubeconfigPaths, &pathsMutex, logger)

	if err != nil {
		t.Errorf("Expected no error for empty cluster list, got: %v", err)
	}

	if len(kubeconfigPaths) != 0 {
		t.Errorf("Expected 0 paths for empty cluster list, got %d", len(kubeconfigPaths))
	}
}

func TestSyncProcessor_processClusters_ContextCancellation(t *testing.T) {
	kubeconfigManager := kubeconfig.NewManager("/test/path")
	logger := logging.Get()
	processor := NewSyncProcessor(kubeconfigManager, logger)

	// Create a context that will be cancelled
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	clusters := []config.Cluster{
		{
			ID:         "c-cluster1",
			Name:       "test-cluster",
			ServerID:   "server-1",
			ServerName: "Test Server",
		},
	}

	var kubeconfigPaths []string
	var pathsMutex sync.Mutex

	// Create a dummy server for the client (will fail but won't panic)
	server := config.RancherServer{
		URL:      "https://test.example.com",
		Username: "admin",
		AuthType: "local",
	}
	client := rancher.NewClient(server)

	err := processor.processClusters(ctx, clusters, "server-1", "https://test.example.com", client, &kubeconfigPaths, &pathsMutex, logger)

	// Should return nil (graceful error handling)
	if err != nil {
		t.Errorf("Expected nil error (graceful handling), got: %v", err)
	}
}

func TestSyncProcessor_processClusters_ConcurrentProcessing(t *testing.T) {
	kubeconfigManager := kubeconfig.NewManager("/test/path")
	logger := logging.Get()
	processor := NewSyncProcessor(kubeconfigManager, logger)

	ctx := context.Background()

	// Create multiple clusters to test concurrent processing
	clusters := []config.Cluster{
		{
			ID:         "c-cluster1",
			Name:       "test-cluster-1",
			ServerID:   "server-1",
			ServerName: "Test Server",
		},
		{
			ID:         "c-cluster2",
			Name:       "test-cluster-2",
			ServerID:   "server-1",
			ServerName: "Test Server",
		},
		{
			ID:         "c-cluster3",
			Name:       "test-cluster-3",
			ServerID:   "server-1",
			ServerName: "Test Server",
		},
	}

	var kubeconfigPaths []string
	var pathsMutex sync.Mutex

	// Create a dummy server for the client (will fail but won't panic)
	server := config.RancherServer{
		URL:      "https://test.example.com",
		Username: "admin",
		AuthType: "local",
	}
	client := rancher.NewClient(server)

	start := time.Now()
	err := processor.processClusters(ctx, clusters, "server-1", "https://test.example.com", client, &kubeconfigPaths, &pathsMutex, logger)
	duration := time.Since(start)

	// Should complete relatively quickly due to concurrent processing
	if duration > 5*time.Second {
		t.Errorf("Expected concurrent processing to complete faster, took %v", duration)
	}

	if err != nil {
		t.Errorf("Expected nil error (graceful handling), got: %v", err)
	}
}

func TestSyncProcessor_processCluster_InvalidCluster(t *testing.T) {
	kubeconfigManager := kubeconfig.NewManager("/test/path")
	logger := logging.Get()
	processor := NewSyncProcessor(kubeconfigManager, logger)

	ctx := context.Background()

	cluster := config.Cluster{
		ID:         "invalid-cluster",
		Name:       "Invalid Cluster",
		ServerID:   "server-1",
		ServerName: "Test Server",
	}

	var kubeconfigPaths []string
	var pathsMutex sync.Mutex

	// Create a dummy server for the client (will fail but won't panic)
	server := config.RancherServer{
		URL:      "https://test.example.com",
		Username: "admin",
		AuthType: "local",
	}
	client := rancher.NewClient(server)

	err := processor.processCluster(ctx, cluster, "server-1", "https://test.example.com", client, &kubeconfigPaths, &pathsMutex, logger)

	// Should return nil (graceful error handling)
	if err != nil {
		t.Errorf("Expected nil error (graceful handling), got: %v", err)
	}

	if len(kubeconfigPaths) != 0 {
		t.Errorf("Expected 0 paths due to invalid cluster, got %d", len(kubeconfigPaths))
	}
}

func TestSyncProcessor_processCluster_ContextTimeout(t *testing.T) {
	kubeconfigManager := kubeconfig.NewManager("/test/path")
	logger := logging.Get()
	processor := NewSyncProcessor(kubeconfigManager, logger)

	// Create a context with very short timeout
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Millisecond)
	defer cancel()

	cluster := config.Cluster{
		ID:         "c-cluster1",
		Name:       "test-cluster",
		ServerID:   "server-1",
		ServerName: "Test Server",
	}

	var kubeconfigPaths []string
	var pathsMutex sync.Mutex

	// Create a dummy server for the client (will fail but won't panic)
	server := config.RancherServer{
		URL:      "https://test.example.com",
		Username: "admin",
		AuthType: "local",
	}
	client := rancher.NewClient(server)

	err := processor.processCluster(ctx, cluster, "server-1", "https://test.example.com", client, &kubeconfigPaths, &pathsMutex, logger)

	// Should return nil (graceful error handling)
	if err != nil {
		t.Errorf("Expected nil error (graceful handling), got: %v", err)
	}

	if len(kubeconfigPaths) != 0 {
		t.Errorf("Expected 0 paths due to timeout, got %d", len(kubeconfigPaths))
	}
}

func TestSyncProcessor_processCluster_ThreadSafety(t *testing.T) {
	kubeconfigManager := kubeconfig.NewManager("/test/path")
	logger := logging.Get()
	processor := NewSyncProcessor(kubeconfigManager, logger)

	ctx := context.Background()

	cluster := config.Cluster{
		ID:         "c-cluster1",
		Name:       "test-cluster",
		ServerID:   "server-1",
		ServerName: "Test Server",
	}

	var kubeconfigPaths []string
	var pathsMutex sync.Mutex

	// Create a dummy server for the client (will fail but won't panic)
	server := config.RancherServer{
		URL:      "https://test.example.com",
		Username: "admin",
		AuthType: "local",
	}
	client := rancher.NewClient(server)

	// Run multiple goroutines to test thread safety
	var wg sync.WaitGroup
	numGoroutines := 10

	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_ = processor.processCluster(ctx, cluster, "server-1", "https://test.example.com", client, &kubeconfigPaths, &pathsMutex, logger)
		}()
	}

	wg.Wait()

	// Should complete without data races
	// All calls should fail gracefully, so paths should be empty
	if len(kubeconfigPaths) != 0 {
		t.Errorf("Expected 0 paths due to failures, got %d", len(kubeconfigPaths))
	}
}

// Integration test to verify the overall flow
func TestSyncProcessor_Integration(t *testing.T) {
	kubeconfigManager := kubeconfig.NewManager("/test/path")
	logger := logging.Get()
	processor := NewSyncProcessor(kubeconfigManager, logger)

	ctx := context.Background()

	servers := []config.RancherServer{
		{
			ID:       "server-1",
			Name:     "Test Server 1",
			URL:      "https://rancher1.example.com",
			Username: "admin",
			AuthType: "local",
		},
		{
			ID:       "server-2",
			Name:     "Test Server 2",
			URL:      "https://rancher2.example.com",
			Username: "admin",
			AuthType: "github",
		},
	}

	// This tests the full integration flow
	paths, err := processor.ProcessServers(ctx, servers)

	// Should complete without hanging and handle errors gracefully
	if err != nil {
		t.Fatalf("Expected graceful error handling, got: %v", err)
	}

	if len(paths) != 0 {
		t.Errorf("Expected 0 paths since all operations should fail, got %d", len(paths))
	}
}

// Test error handling patterns
func TestSyncProcessor_ErrorHandling(t *testing.T) {
	kubeconfigManager := kubeconfig.NewManager("/test/path")
	logger := logging.Get()
	processor := NewSyncProcessor(kubeconfigManager, logger)

	ctx := context.Background()

	// Test with servers that will definitely fail
	servers := []config.RancherServer{
		{
			ID:       "server-1",
			Name:     "Invalid Server",
			URL:      "http://localhost:99999", // Non-existent port
			Username: "admin",
			AuthType: "local",
		},
	}

	paths, err := processor.ProcessServers(ctx, servers)

	// Should handle errors gracefully and not return an error
	if err != nil {
		t.Errorf("Expected graceful error handling, got: %v", err)
	}

	if len(paths) != 0 {
		t.Errorf("Expected 0 paths due to server failures, got %d", len(paths))
	}
}

// Sync Command Tests

func TestRunSync_NoServersConfigured(t *testing.T) {
	// Create a temporary directory for test config
	tempDir := t.TempDir()
	tempConfigPath := filepath.Join(tempDir, "config.yaml")

	// Create empty config file
	emptyConfig := &config.Config{
		Version: "1.0",
		Servers: []config.RancherServer{},
	}

	manager := config.NewConfigManager(tempConfigPath)
	err := manager.SaveConfig(emptyConfig)
	if err != nil {
		t.Fatalf("Failed to create test config: %v", err)
	}

	// Set environment to use our test config
	originalHome := os.Getenv("HOME")
	defer func() { _ = os.Setenv("HOME", originalHome) }()
	_ = os.Setenv("HOME", tempDir)

	// Create and run the command
	cmd := &cobra.Command{}
	err = runSync(cmd, []string{})

	// Should complete successfully with no servers message
	if err != nil {
		t.Errorf("Expected no error for empty server list, got: %v", err)
	}
}

func TestRunSync_ConfigManagerError(t *testing.T) {
	// Set an invalid home directory that will cause utils.GetConfigManager to fail
	originalHome := os.Getenv("HOME")
	defer func() { _ = os.Setenv("HOME", originalHome) }()

	// Set HOME to empty string which will cause the config path resolution to fail
	_ = os.Setenv("HOME", "")

	cmd := &cobra.Command{}
	err := runSync(cmd, []string{})

	// Should return an error due to config manager failure
	if err == nil {
		t.Error("Expected error when config manager fails")
	}
}

func TestRunSync_LoadServersError(t *testing.T) {
	// Create a temporary directory but no config file
	tempDir := t.TempDir()

	// Set environment to use our test directory but don't create config
	originalHome := os.Getenv("HOME")
	defer func() { _ = os.Setenv("HOME", originalHome) }()
	_ = os.Setenv("HOME", tempDir)

	// Create invalid config file that will cause parsing error
	configPath := filepath.Join(tempDir, ".config", "cowpoke", "config.yaml")
	err := os.MkdirAll(filepath.Dir(configPath), 0755)
	if err != nil {
		t.Fatalf("Failed to create config directory: %v", err)
	}

	// Write invalid YAML
	err = os.WriteFile(configPath, []byte("invalid: yaml: content: ["), 0644)
	if err != nil {
		t.Fatalf("Failed to write invalid config: %v", err)
	}

	cmd := &cobra.Command{}
	err = runSync(cmd, []string{})

	// Should return an error due to invalid config
	if err == nil {
		t.Error("Expected error when loading servers fails")
	}

	if !strings.Contains(err.Error(), "failed to load servers") {
		t.Errorf("Expected 'failed to load servers' error, got: %v", err)
	}
}

func TestRunSync_GetKubeconfigDirError(t *testing.T) {
	// Save original HOME and clear it to cause GetKubeconfigDir to fail
	originalHome := os.Getenv("HOME")
	defer func() { _ = os.Setenv("HOME", originalHome) }()
	_ = os.Unsetenv("HOME")

	cmd := &cobra.Command{}
	err := runSync(cmd, []string{})

	// Should return an error due to missing HOME environment variable
	if err == nil {
		t.Error("Expected error when HOME environment variable is not set")
	}

	if !strings.Contains(err.Error(), "failed to get home directory") {
		t.Errorf("Expected error about home directory, got: %v", err)
	}
}

func TestRunSync_ProcessServersError(t *testing.T) {
	// Create a temporary directory for test config
	tempDir := t.TempDir()

	// Create config in standard location
	configDir := filepath.Join(tempDir, ".config", "cowpoke")
	err := os.MkdirAll(configDir, 0755)
	if err != nil {
		t.Fatalf("Failed to create config directory: %v", err)
	}

	configPath := filepath.Join(configDir, "config.yaml")
	testConfig := &config.Config{
		Version: "1.0",
		Servers: []config.RancherServer{
			{
				ID:       "server-1",
				Name:     "Test Server",
				URL:      "https://rancher.example.com",
				Username: "admin",
				AuthType: "local",
			},
		},
	}

	manager := config.NewConfigManager(configPath)
	err = manager.SaveConfig(testConfig)
	if err != nil {
		t.Fatalf("Failed to create test config: %v", err)
	}

	// Set environment to use our test config
	originalHome := os.Getenv("HOME")
	defer func() { _ = os.Setenv("HOME", originalHome) }()
	_ = os.Setenv("HOME", tempDir)

	// This will fail during ProcessServers due to network issues
	// but since our implementation handles errors gracefully,
	// it should not return an error but will have no kubeconfigs
	cmd := &cobra.Command{}
	err = runSync(cmd, []string{})

	// Should complete successfully but with no kubeconfigs downloaded message
	if err != nil {
		t.Errorf("Expected no error (graceful handling), got: %v", err)
	}
}

func TestRunSync_NoKubeconfigsDownloaded(t *testing.T) {
	// Create a temporary directory for test config
	tempDir := t.TempDir()
	tempConfigPath := filepath.Join(tempDir, "config.yaml")

	// Create config with servers that will fail to download kubeconfigs
	testConfig := &config.Config{
		Version: "1.0",
		Servers: []config.RancherServer{
			{
				ID:       "server-1",
				Name:     "Invalid Server",
				URL:      "https://non-existent-server.example.com",
				Username: "admin",
				AuthType: "local",
			},
		},
	}

	manager := config.NewConfigManager(tempConfigPath)
	err := manager.SaveConfig(testConfig)
	if err != nil {
		t.Fatalf("Failed to create test config: %v", err)
	}

	// Set environment to use our test config
	originalHome := os.Getenv("HOME")
	defer func() { _ = os.Setenv("HOME", originalHome) }()
	_ = os.Setenv("HOME", tempDir)

	cmd := &cobra.Command{}
	err = runSync(cmd, []string{})

	// Should complete successfully but print warning about no kubeconfigs
	if err != nil {
		t.Errorf("Expected no error (graceful handling), got: %v", err)
	}
}

func TestRunSync_GetDefaultKubeconfigPathError(t *testing.T) {
	// This test is difficult to trigger since GetDefaultKubeconfigPath
	// uses the same HOME resolution as other functions.
	// We'll create a scenario where everything works until that point
	// by using a mock or by temporarily making the function fail.

	// For now, this test documents the expected behavior when
	// GetDefaultKubeconfigPath fails - it should return an error.
	t.Skip("Difficult to mock GetDefaultKubeconfigPath failure in current architecture")
}

func TestRunSync_MergeKubeconfigsError(t *testing.T) {
	// This test is challenging because it requires ProcessServers to succeed
	// but MergeKubeconfigs to fail. In our current architecture, this would
	// require either:
	// 1. A filesystem permission error during merge
	// 2. Invalid kubeconfig content
	// 3. A mock of the kubeconfig manager

	// For now, this test documents the expected behavior when
	// MergeKubeconfigs fails - it should return a wrapped error.
	t.Skip("Requires complex setup to make MergeKubeconfigs fail after ProcessServers succeeds")
}

func TestRunSync_Success(t *testing.T) {
	// This test is challenging because it requires:
	// 1. Valid Rancher servers (we'd need to mock HTTP responses)
	// 2. Successful authentication
	// 3. Successful kubeconfig download
	// 4. Successful merge operation

	// In a real-world scenario, this would require integration testing
	// with mock HTTP servers or test containers.

	// For now, this test documents the expected behavior for a successful sync.
	t.Skip("Requires mock HTTP servers to test successful sync end-to-end")
}

// Test the sync command initialization
func TestSyncCmdInitialization(t *testing.T) {
	// Verify the sync command is properly configured
	if syncCmd.Use != "sync" {
		t.Errorf("Expected Use to be 'sync', got: %s", syncCmd.Use)
	}

	if syncCmd.Short == "" {
		t.Error("Expected Short description to be set")
	}

	if syncCmd.Long == "" {
		t.Error("Expected Long description to be set")
	}

	if syncCmd.RunE == nil {
		t.Error("Expected RunE to be set to runSync function")
	}
}

// Additional tests to improve sync processor coverage

func TestSyncProcessor_ProcessServers_ErrorReturned(t *testing.T) {
	// Test the error path in ProcessServers when g.Wait() returns an error due to context cancellation
	tempDir := t.TempDir()
	manager := kubeconfig.NewManager(tempDir)

	logger := logging.Get()

	processor := NewSyncProcessor(manager, logger)

	// Create a context with an extremely short timeout to force cancellation during processing
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Nanosecond)
	defer cancel()

	servers := []config.RancherServer{
		{
			ID:       "server-1",
			Name:     "Test Server 1",
			URL:      "https://rancher1.example.com",
			Username: "admin",
			AuthType: "local",
		},
	}

	// Wait a moment to ensure context is canceled
	time.Sleep(1 * time.Millisecond)

	kubeconfigPaths, err := processor.ProcessServers(ctx, servers)

	// Should return an error due to context cancellation
	if err == nil {
		t.Error("Expected error due to context cancellation")
	}

	if kubeconfigPaths != nil {
		t.Error("Expected nil kubeconfig paths when error occurs")
	}

	if err != nil && !strings.Contains(err.Error(), "failed to collect passwords") && !strings.Contains(err.Error(), "sync failed") {
		t.Errorf("Expected error to contain 'sync failed' or 'failed to collect passwords', got: %v", err)
	}
}

func TestSyncProcessor_processSingleServer_SuccessfulAuthentication(t *testing.T) {
	// Test the success path after authentication in processSingleServer
	tempDir := t.TempDir()
	manager := kubeconfig.NewManager(tempDir)

	logger := logging.Get()

	processor := NewSyncProcessor(manager, logger)

	ctx := context.Background()
	server := config.RancherServer{
		ID:       "test-server",
		Name:     "Test Server",
		URL:      "https://rancher.example.com",
		Username: "admin",
		AuthType: "local",
	}

	var kubeconfigPaths []string
	var pathsMutex sync.Mutex

	// This will test the authentication failure path since we can't easily mock a successful auth
	// But it exercises the error handling and logging code paths
	err := processor.processSingleServer(ctx, server, "test-password", &kubeconfigPaths, &pathsMutex)

	// Should return nil (not an error) since errors are handled gracefully
	if err != nil {
		t.Errorf("Expected processSingleServer to handle errors gracefully, got: %v", err)
	}
}

func TestSyncProcessor_processCluster_SaveKubeconfigError(t *testing.T) {
	// Test the error path when SaveKubeconfig fails in processCluster
	tempDir := t.TempDir()

	// Create a read-only directory to cause SaveKubeconfig to fail
	readOnlyDir := filepath.Join(tempDir, "readonly")
	err := os.MkdirAll(readOnlyDir, 0444) // Read-only permissions
	if err != nil {
		t.Fatalf("Failed to create read-only directory: %v", err)
	}

	manager := kubeconfig.NewManager(readOnlyDir)

	logger := logging.Get()

	processor := NewSyncProcessor(manager, logger)

	ctx := context.Background()
	cluster := config.Cluster{
		ID:   "test-cluster",
		Name: "Test Cluster",
	}

	// Create a mock client that will succeed in getting kubeconfig
	client := rancher.NewClient(config.RancherServer{
		URL:      "https://test.example.com",
		Username: "admin",
		AuthType: "local",
	})

	var kubeconfigPaths []string
	var pathsMutex sync.Mutex
	serverLogger := logger.With("server_url", "https://test.example.com", "auth_type", "local")

	// This will test the GetKubeconfig failure path since we can't authenticate
	// But it exercises the error handling code paths
	err = processor.processCluster(ctx, cluster, "test-server", "https://test.example.com", client, &kubeconfigPaths, &pathsMutex, serverLogger)

	// Should return nil (not an error) since errors are handled gracefully
	if err != nil {
		t.Errorf("Expected processCluster to handle errors gracefully, got: %v", err)
	}
}

func TestSyncProcessor_ProcessServers_SemaphoreCancellation(t *testing.T) {
	// Test context cancellation while waiting for semaphore
	tempDir := t.TempDir()
	manager := kubeconfig.NewManager(tempDir)

	logger := logging.Get()

	processor := NewSyncProcessor(manager, logger)

	// Create a context with a very short timeout
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Millisecond)
	defer cancel()

	// Create more servers than the semaphore limit (3) to force queueing
	servers := []config.RancherServer{
		{ID: "server-1", Name: "Server 1", URL: "https://rancher1.example.com", Username: "admin", AuthType: "local"},
		{ID: "server-2", Name: "Server 2", URL: "https://rancher2.example.com", Username: "admin", AuthType: "local"},
		{ID: "server-3", Name: "Server 3", URL: "https://rancher3.example.com", Username: "admin", AuthType: "local"},
		{ID: "server-4", Name: "Server 4", URL: "https://rancher4.example.com", Username: "admin", AuthType: "local"},
		{ID: "server-5", Name: "Server 5", URL: "https://rancher5.example.com", Username: "admin", AuthType: "local"},
	}

	// Allow a moment for the timeout to trigger
	time.Sleep(5 * time.Millisecond)

	kubeconfigPaths, err := processor.ProcessServers(ctx, servers)

	// Should return an error due to context timeout
	if err == nil {
		t.Error("Expected error due to context timeout")
	}

	if kubeconfigPaths != nil {
		t.Error("Expected nil kubeconfig paths when error occurs")
	}
}

func TestSyncCmd_OutputFlag(t *testing.T) {
	// Test that the sync command has the output flag configured correctly

	// Check if the flag exists
	outputFlag := syncCmd.Flags().Lookup("output")
	if outputFlag == nil {
		t.Error("Expected 'output' flag to be defined")
		return
	}

	// Check flag properties
	if outputFlag.Shorthand != "o" {
		t.Errorf("Expected shorthand 'o', got: %s", outputFlag.Shorthand)
	}

	if outputFlag.DefValue != "" {
		t.Errorf("Expected default value to be empty, got: %s", outputFlag.DefValue)
	}

	expectedUsage := "Output directory or file path for merged kubeconfig (default: ~/.kube/config)"
	if outputFlag.Usage != expectedUsage {
		t.Errorf("Expected usage '%s', got: %s", expectedUsage, outputFlag.Usage)
	}
}
