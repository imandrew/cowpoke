package cmd

import (
	"os"
	"path/filepath"
	"testing"

	"cowpoke/internal/config"

	"github.com/spf13/cobra"
)

func TestRunAdd(t *testing.T) {
	tempDir := t.TempDir()
	t.Setenv("HOME", tempDir)

	// Create .config/cowpoke directory
	cowpokeDir := filepath.Join(tempDir, ".config", "cowpoke")
	err := os.MkdirAll(cowpokeDir, 0o755)
	if err != nil {
		t.Fatalf("Failed to create .config/cowpoke directory: %v", err)
	}

	// Set test flags
	cmd := &cobra.Command{}
	cmd.Flags().String("url", "", "")
	cmd.Flags().String("username", "", "")
	cmd.Flags().String("authtype", "", "")
	_ = cmd.Flags().Set("url", "https://rancher.example.com")
	_ = cmd.Flags().Set("username", "testuser")
	_ = cmd.Flags().Set("authtype", "local")
	err = runAdd(cmd, nil)
	if err != nil {
		t.Fatalf("runAdd failed: %v", err)
	}

	// Verify server was added
	configPath := filepath.Join(cowpokeDir, "config.yaml")
	configManager := config.NewConfigManager(configPath)

	servers, err := configManager.GetServers()
	if err != nil {
		t.Fatalf("Failed to get servers: %v", err)
	}

	if len(servers) != 1 {
		t.Errorf("Expected 1 server, got: %d", len(servers))
	}

	server := servers[0]
	if server.URL != "https://rancher.example.com" {
		t.Errorf("Expected URL https://rancher.example.com, got: %s", server.URL)
	}
	if server.Name != "https://rancher.example.com" {
		t.Errorf("Expected Name https://rancher.example.com, got: %s", server.Name)
	}
	if server.Username != "testuser" {
		t.Errorf("Expected Username testuser, got: %s", server.Username)
	}
	if server.AuthType != "local" {
		t.Errorf("Expected AuthType local, got: %s", server.AuthType)
	}
}

func TestRunAdd_HomeDirectoryError(t *testing.T) {
	t.Setenv("HOME", "")

	cmd := &cobra.Command{}
	err := runAdd(cmd, nil)
	if err == nil {
		t.Error("Expected error when HOME is not set")
	}
}

func TestRunAdd_InvalidAuthType(t *testing.T) {
	tempDir := t.TempDir()
	t.Setenv("HOME", tempDir)

	// Create .config/cowpoke directory
	cowpokeDir := filepath.Join(tempDir, ".config", "cowpoke")
	err := os.MkdirAll(cowpokeDir, 0o755)
	if err != nil {
		t.Fatalf("Failed to create .config/cowpoke directory: %v", err)
	}

	// Set test flags with invalid auth type
	cmd := &cobra.Command{}
	cmd.Flags().String("url", "", "")
	cmd.Flags().String("username", "", "")
	cmd.Flags().String("authtype", "", "")
	_ = cmd.Flags().Set("url", "https://rancher.example.com")
	_ = cmd.Flags().Set("username", "testuser")
	_ = cmd.Flags().Set("authtype", "invalid-auth-type")
	err = runAdd(cmd, nil)
	if err == nil {
		t.Error("Expected error when using invalid auth type")
	}

	expectedErrorContains := "validation error in field 'auth_type'"
	if !contains(err.Error(), expectedErrorContains) {
		t.Errorf("Expected error to contain '%s', got: %s", expectedErrorContains, err.Error())
	}

	supportedTypesContains := "auth type must be one of:"
	if !contains(err.Error(), supportedTypesContains) {
		t.Errorf("Expected error to contain '%s', got: %s", supportedTypesContains, err.Error())
	}
}

// Helper function to check if string contains substring.
func contains(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
