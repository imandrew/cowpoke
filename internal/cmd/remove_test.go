package cmd

import (
	"os"
	"path/filepath"
	"testing"

	"cowpoke/internal/config"
)

func TestRunRemove(t *testing.T) {
	tempDir := t.TempDir()
	origHome := os.Getenv("HOME")
	defer func() {
		_ = os.Setenv("HOME", origHome)
	}()
	_ = os.Setenv("HOME", tempDir)

	// Create .config/cowpoke directory
	cowpokeDir := filepath.Join(tempDir, ".config", "cowpoke")
	err := os.MkdirAll(cowpokeDir, 0755)
	if err != nil {
		t.Fatalf("Failed to create .config/cowpoke directory: %v", err)
	}

	// Add servers first
	configPath := filepath.Join(cowpokeDir, "config.yaml")
	configManager := config.NewConfigManager(configPath)

	server1 := config.RancherServer{
		Name:     "https://rancher1.example.com",
		URL:      "https://rancher1.example.com",
		Username: "admin",
		AuthType: "local",
	}

	server2 := config.RancherServer{
		Name:     "https://rancher2.example.com",
		URL:      "https://rancher2.example.com",
		Username: "admin",
		AuthType: "github",
	}

	err = configManager.AddServer(server1)
	if err != nil {
		t.Fatalf("Failed to add server1: %v", err)
	}

	err = configManager.AddServer(server2)
	if err != nil {
		t.Fatalf("Failed to add server2: %v", err)
	}

	// Set test flag and remove server
	removeURL = "https://rancher1.example.com"

	err = runRemove(nil, nil)
	if err != nil {
		t.Fatalf("runRemove failed: %v", err)
	}

	// Verify server was removed
	servers, err := configManager.GetServers()
	if err != nil {
		t.Fatalf("Failed to get servers: %v", err)
	}

	if len(servers) != 1 {
		t.Errorf("Expected 1 server after removal, got: %d", len(servers))
	}

	remainingServer := servers[0]
	if remainingServer.URL == "https://rancher1.example.com" {
		t.Error("Removed server still exists")
	}
	if remainingServer.URL != "https://rancher2.example.com" {
		t.Errorf("Expected remaining server to be rancher2, got: %s", remainingServer.URL)
	}
}

func TestRunRemove_NotFound(t *testing.T) {
	tempDir := t.TempDir()
	origHome := os.Getenv("HOME")
	defer func() {
		_ = os.Setenv("HOME", origHome)
	}()
	_ = os.Setenv("HOME", tempDir)

	// Create .config/cowpoke directory
	cowpokeDir := filepath.Join(tempDir, ".config", "cowpoke")
	err := os.MkdirAll(cowpokeDir, 0755)
	if err != nil {
		t.Fatalf("Failed to create .config/cowpoke directory: %v", err)
	}

	// Set test flag for non-existent server
	removeURL = "https://non-existent.example.com"

	err = runRemove(nil, nil)
	if err == nil {
		t.Error("Expected error when removing non-existent server")
	}
}

func TestRunRemove_HomeDirectoryError(t *testing.T) {
	origHome := os.Getenv("HOME")
	defer func() {
		_ = os.Setenv("HOME", origHome)
	}()
	_ = os.Unsetenv("HOME")

	err := runRemove(nil, nil)
	if err == nil {
		t.Error("Expected error when HOME is not set")
	}
}
