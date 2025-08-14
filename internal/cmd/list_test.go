package cmd

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"cowpoke/internal/config"

	"github.com/spf13/cobra"
)

func TestRunList_EmptyConfig(t *testing.T) {
	tempDir := t.TempDir()
	t.Setenv("HOME", tempDir)

	// Create .config/cowpoke directory
	cowpokeDir := filepath.Join(tempDir, ".config", "cowpoke")
	err := os.MkdirAll(cowpokeDir, 0o755)
	if err != nil {
		t.Fatalf("Failed to create .config/cowpoke directory: %v", err)
	}

	// Capture output
	var buf bytes.Buffer
	cmd := &cobra.Command{}
	cmd.SetOut(&buf)

	err = runList(cmd, nil)
	if err != nil {
		t.Fatalf("runList failed: %v", err)
	}

	outputStr := buf.String()

	if !strings.Contains(outputStr, "No Rancher servers configured") {
		t.Errorf("Expected 'No Rancher servers configured' message, got: %s", outputStr)
	}
}

func TestRunList_WithServers(t *testing.T) {
	tempDir := t.TempDir()
	t.Setenv("HOME", tempDir)

	// Create .config/cowpoke directory
	cowpokeDir := filepath.Join(tempDir, ".config", "cowpoke")
	err := os.MkdirAll(cowpokeDir, 0o755)
	if err != nil {
		t.Fatalf("Failed to create .config/cowpoke directory: %v", err)
	}

	// Add servers
	configPath := filepath.Join(cowpokeDir, "config.yaml")
	configManager := config.NewConfigManager(configPath)

	server1 := config.RancherServer{
		Name:     "https://rancher1.example.com",
		URL:      "https://rancher1.example.com",
		Username: "admin1",
		AuthType: "local",
	}

	server2 := config.RancherServer{
		Name:     "https://rancher2.example.com",
		URL:      "https://rancher2.example.com",
		Username: "admin2",
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

	// Capture output
	var buf bytes.Buffer
	cmd := &cobra.Command{}
	cmd.SetOut(&buf)

	err = runList(cmd, nil)
	if err != nil {
		t.Fatalf("runList failed: %v", err)
	}

	outputStr := buf.String()

	// Check that both servers are listed
	if !strings.Contains(outputStr, "Configured Rancher servers (2)") {
		t.Errorf("Expected server count message, got: %s", outputStr)
	}
	if !strings.Contains(outputStr, "https://rancher1.example.com") {
		t.Errorf("Expected rancher1 URL in output, got: %s", outputStr)
	}
	if !strings.Contains(outputStr, "https://rancher2.example.com") {
		t.Errorf("Expected rancher2 URL in output, got: %s", outputStr)
	}
	if !strings.Contains(outputStr, "admin1") {
		t.Errorf("Expected admin1 username in output, got: %s", outputStr)
	}
	if !strings.Contains(outputStr, "admin2") {
		t.Errorf("Expected admin2 username in output, got: %s", outputStr)
	}
	if !strings.Contains(outputStr, "local") {
		t.Errorf("Expected local auth type in output, got: %s", outputStr)
	}
	if !strings.Contains(outputStr, "github") {
		t.Errorf("Expected github auth type in output, got: %s", outputStr)
	}
}

func TestRunList_HomeDirectoryError(t *testing.T) {
	t.Setenv("HOME", "")

	cmd := &cobra.Command{}
	err := runList(cmd, nil)
	if err == nil {
		t.Error("Expected error when HOME is not set")
	}
}
