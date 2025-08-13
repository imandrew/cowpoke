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
	origHome := os.Getenv("HOME")
	defer func() {
		os.Setenv("HOME", origHome)
	}()
	os.Setenv("HOME", tempDir)

	// Create .config/cowpoke directory
	cowpokeDir := filepath.Join(tempDir, ".config", "cowpoke")
	err := os.MkdirAll(cowpokeDir, 0755)
	if err != nil {
		t.Fatalf("Failed to create .config/cowpoke directory: %v", err)
	}

	// Capture output
	var buf bytes.Buffer
	cmd := &cobra.Command{}
	cmd.SetOut(&buf)

	// Redirect stdout for runList
	origStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	err = runList(cmd, nil)
	if err != nil {
		t.Fatalf("runList failed: %v", err)
	}

	// Restore stdout and get output
	w.Close()
	os.Stdout = origStdout

	output := make([]byte, 1024)
	n, _ := r.Read(output)
	outputStr := string(output[:n])

	if !strings.Contains(outputStr, "No Rancher servers configured") {
		t.Errorf("Expected 'No Rancher servers configured' message, got: %s", outputStr)
	}
}

func TestRunList_WithServers(t *testing.T) {
	tempDir := t.TempDir()
	origHome := os.Getenv("HOME")
	defer func() {
		os.Setenv("HOME", origHome)
	}()
	os.Setenv("HOME", tempDir)

	// Create .config/cowpoke directory
	cowpokeDir := filepath.Join(tempDir, ".config", "cowpoke")
	err := os.MkdirAll(cowpokeDir, 0755)
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
	origStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	err = runList(nil, nil)
	if err != nil {
		t.Fatalf("runList failed: %v", err)
	}

	// Restore stdout and get output
	w.Close()
	os.Stdout = origStdout

	output := make([]byte, 2048)
	n, _ := r.Read(output)
	outputStr := string(output[:n])

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
	origHome := os.Getenv("HOME")
	defer func() {
		os.Setenv("HOME", origHome)
	}()
	os.Unsetenv("HOME")

	err := runList(nil, nil)
	if err == nil {
		t.Error("Expected error when HOME is not set")
	}
}
