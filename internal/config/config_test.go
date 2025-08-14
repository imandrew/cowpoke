package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// ConfigManager Tests

func TestNewConfigManager(t *testing.T) {
	path := "/test/config/path"
	manager := NewConfigManager(path)

	if manager == nil {
		t.Fatal("Expected NewConfigManager() to return a non-nil manager")
	}

	if manager.configPath != path {
		t.Errorf("Expected config path to be '%s', got '%s'", path, manager.configPath)
	}
}

func TestConfigManager_LoadConfig_NonExistentFile(t *testing.T) {
	tempDir := t.TempDir()
	configPath := filepath.Join(tempDir, "non-existent-config.yaml")

	manager := NewConfigManager(configPath)
	config, err := manager.LoadConfig()
	if err != nil {
		t.Fatalf("Expected no error for non-existent file, got: %v", err)
	}

	if config == nil {
		t.Fatal("Expected LoadConfig() to return a non-nil config")
	}

	if len(config.Servers) != 0 {
		t.Errorf("Expected config to have no servers, got %d", len(config.Servers))
	}
}

func TestConfigManager_LoadConfig_EmptyFile(t *testing.T) {
	tempDir := t.TempDir()
	configPath := filepath.Join(tempDir, "empty-config.yaml")

	// Create empty file
	file, err := os.Create(configPath)
	if err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}
	_ = file.Close()

	manager := NewConfigManager(configPath)
	config, err := manager.LoadConfig()
	if err != nil {
		t.Fatalf("Expected no error for empty file, got: %v", err)
	}

	if config == nil {
		t.Fatal("Expected LoadConfig() to return a non-nil config")
	}

	if len(config.Servers) != 0 {
		t.Errorf("Expected config to have no servers, got %d", len(config.Servers))
	}
}

func TestConfigManager_AddServer(t *testing.T) {
	tempDir := t.TempDir()
	configPath := filepath.Join(tempDir, "test-config.yaml")

	manager := NewConfigManager(configPath)

	server := RancherServer{
		Name:     "Test Server",
		URL:      "https://rancher.example.com",
		Username: "admin",
		AuthType: "local",
	}

	err := manager.AddServer(server)
	if err != nil {
		t.Fatalf("Expected no error adding server, got: %v", err)
	}

	// Verify server was added
	config, err := manager.LoadConfig()
	if err != nil {
		t.Fatalf("Failed to load config after adding server: %v", err)
	}

	if len(config.Servers) != 1 {
		t.Errorf("Expected 1 server in config, got %d", len(config.Servers))
	}

	if config.Servers[0].URL != server.URL {
		t.Errorf("Expected server URL to be '%s', got '%s'", server.URL, config.Servers[0].URL)
	}
}

func TestConfigManager_AddServer_InvalidAuthType(t *testing.T) {
	tempDir := t.TempDir()
	configPath := filepath.Join(tempDir, "test-config.yaml")

	manager := NewConfigManager(configPath)

	server := RancherServer{
		Name:     "Test Server",
		URL:      "https://rancher.example.com",
		Username: "admin",
		AuthType: "invalid-auth-type",
	}

	err := manager.AddServer(server)
	if err == nil {
		t.Error("Expected error when adding server with invalid auth type")
	}

	// Verify error message contains information about valid auth types
	if !strings.Contains(err.Error(), "auth type must be one of") {
		t.Errorf("Expected error message to mention valid auth types, got: %s", err.Error())
	}
}

// Auth Type Validation Tests

func TestIsValidAuthType(t *testing.T) {
	// Test valid auth types
	validTypes := []string{
		"local", "github", "openldap", "activedirectory", "azuread",
		"okta", "ping", "keycloak", "shibboleth", "googleoauth",
	}
	for _, authType := range validTypes {
		if !IsValidAuthType(authType) {
			t.Errorf("Expected '%s' to be valid", authType)
		}
	}

	// Test invalid auth types
	invalidTypes := []string{"invalid", "nonexistent", "", "GITHUB", "Local"}
	for _, authType := range invalidTypes {
		if IsValidAuthType(authType) {
			t.Errorf("Expected '%s' to be invalid", authType)
		}
	}
}

func TestGetSupportedAuthTypesString(t *testing.T) {
	str := GetSupportedAuthTypesString()

	if str == "" {
		t.Error("Expected non-empty string from GetSupportedAuthTypesString")
	}

	// Should contain known auth types
	expectedTypes := []string{"local", "github", "openldap"}
	for _, authType := range expectedTypes {
		if !strings.Contains(str, authType) {
			t.Errorf("Expected string to contain '%s', got: %s", authType, str)
		}
	}

	// Should be comma-separated
	if !strings.Contains(str, ", ") {
		t.Errorf("Expected comma-separated values, got: %s", str)
	}
}
