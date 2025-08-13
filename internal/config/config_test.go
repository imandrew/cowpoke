package config

import (
	"os"
	"path/filepath"
	"testing"
)

// AuthProvider Tests

func TestAuthProvider_String(t *testing.T) {
	provider := AuthProvider{
		ID:          "test-id",
		DisplayName: "Test Provider",
		Provider:    "test",
		Description: "Test description",
	}

	if provider.String() != "test-id" {
		t.Errorf("Expected String() to return 'test-id', got %s", provider.String())
	}
}

func TestAuthProvider_URLPath(t *testing.T) {
	provider := AuthProvider{
		ID:          "github",
		DisplayName: "GitHub",
		Provider:    "github",
		Description: "GitHub OAuth",
	}

	if provider.URLPath() != "github" {
		t.Errorf("Expected URLPath() to return 'github', got %s", provider.URLPath())
	}
}

// AuthRegistry Tests

func TestNewAuthRegistry(t *testing.T) {
	registry := NewAuthRegistry()
	if registry == nil {
		t.Fatal("Expected NewAuthRegistry to return non-nil registry")
	}

	if registry.providers == nil {
		t.Error("Expected providers map to be initialized")
	}

	if registry.order == nil {
		t.Error("Expected order slice to be initialized")
	}

	if len(registry.providers) != 0 {
		t.Errorf("Expected empty registry, got %d providers", len(registry.providers))
	}
}

func TestAuthRegistry_Register(t *testing.T) {
	registry := NewAuthRegistry()
	provider := AuthProvider{
		ID:          "test",
		DisplayName: "Test Provider",
		Provider:    "test",
		Description: "Test description",
	}

	// Test initial registration
	registry.Register(provider)

	if len(registry.providers) != 1 {
		t.Errorf("Expected 1 provider after registration, got %d", len(registry.providers))
	}

	if len(registry.order) != 1 {
		t.Errorf("Expected 1 item in order slice, got %d", len(registry.order))
	}

	if registry.order[0] != "test" {
		t.Errorf("Expected first order item to be 'test', got %s", registry.order[0])
	}

	// Test updating existing provider (shouldn't duplicate in order)
	updatedProvider := AuthProvider{
		ID:          "test",
		DisplayName: "Updated Test Provider",
		Provider:    "test",
		Description: "Updated description",
	}

	registry.Register(updatedProvider)

	if len(registry.providers) != 1 {
		t.Errorf("Expected 1 provider after update, got %d", len(registry.providers))
	}

	if len(registry.order) != 1 {
		t.Errorf("Expected 1 item in order slice after update, got %d", len(registry.order))
	}

	retrieved, _ := registry.Get("test")
	if retrieved.DisplayName != "Updated Test Provider" {
		t.Errorf("Expected updated display name, got %s", retrieved.DisplayName)
	}
}

func TestAuthRegistry_Get(t *testing.T) {
	registry := NewAuthRegistry()
	provider := AuthProvider{
		ID:          "local",
		DisplayName: "Local Auth",
		Provider:    "local",
		Description: "Local authentication",
	}

	registry.Register(provider)

	// Test successful retrieval
	retrieved, exists := registry.Get("local")
	if !exists {
		t.Error("Expected provider to exist")
	}

	if retrieved.ID != "local" {
		t.Errorf("Expected ID 'local', got %s", retrieved.ID)
	}

	if retrieved.DisplayName != "Local Auth" {
		t.Errorf("Expected DisplayName 'Local Auth', got %s", retrieved.DisplayName)
	}

	// Test non-existent provider
	_, exists = registry.Get("nonexistent")
	if exists {
		t.Error("Expected provider 'nonexistent' to not exist")
	}
}

func TestAuthRegistry_All(t *testing.T) {
	registry := NewAuthRegistry()

	// Test empty registry
	all := registry.All()
	if len(all) != 0 {
		t.Errorf("Expected empty slice from empty registry, got %d items", len(all))
	}

	// Add providers in specific order
	providers := []AuthProvider{
		{ID: "first", DisplayName: "First", Provider: "first"},
		{ID: "second", DisplayName: "Second", Provider: "second"},
		{ID: "third", DisplayName: "Third", Provider: "third"},
	}

	for _, provider := range providers {
		registry.Register(provider)
	}

	all = registry.All()
	if len(all) != 3 {
		t.Errorf("Expected 3 providers, got %d", len(all))
	}

	// Verify order is maintained
	expectedOrder := []string{"first", "second", "third"}
	for i, provider := range all {
		if provider.ID != expectedOrder[i] {
			t.Errorf("Expected provider %d to have ID %s, got %s", i, expectedOrder[i], provider.ID)
		}
	}
}

func TestAuthRegistry_IsValid(t *testing.T) {
	registry := NewAuthRegistry()

	// Test with empty registry
	if registry.IsValid("local") {
		t.Error("Expected 'local' to be invalid in empty registry")
	}

	// Register a provider
	provider := AuthProvider{ID: "local", DisplayName: "Local", Provider: "local"}
	registry.Register(provider)

	// Test valid provider
	if !registry.IsValid("local") {
		t.Error("Expected 'local' to be valid after registration")
	}

	// Test invalid provider
	if registry.IsValid("github") {
		t.Error("Expected 'github' to be invalid when not registered")
	}
}

func TestAuthRegistry_IDs(t *testing.T) {
	registry := NewAuthRegistry()

	// Test empty registry
	ids := registry.IDs()
	if len(ids) != 0 {
		t.Errorf("Expected empty IDs slice, got %d items", len(ids))
	}

	// Add providers
	registry.Register(AuthProvider{ID: "local", DisplayName: "Local", Provider: "local"})
	registry.Register(AuthProvider{ID: "github", DisplayName: "GitHub", Provider: "github"})

	ids = registry.IDs()
	if len(ids) != 2 {
		t.Errorf("Expected 2 IDs, got %d", len(ids))
	}

	// Verify order
	expectedIDs := []string{"local", "github"}
	for i, id := range ids {
		if id != expectedIDs[i] {
			t.Errorf("Expected ID %d to be %s, got %s", i, expectedIDs[i], id)
		}
	}

	// Verify it's a copy (modification shouldn't affect original)
	ids[0] = "modified"
	originalIDs := registry.IDs()
	if originalIDs[0] == "modified" {
		t.Error("Expected IDs() to return a copy, not the original slice")
	}
}

func TestAuthRegistry_String(t *testing.T) {
	registry := NewAuthRegistry()

	// Test empty registry
	str := registry.String()
	if str != "[]" {
		t.Errorf("Expected empty registry string to be '[]', got %s", str)
	}

	// Add providers
	registry.Register(AuthProvider{ID: "local", DisplayName: "Local", Provider: "local"})
	registry.Register(AuthProvider{ID: "github", DisplayName: "GitHub", Provider: "github"})

	str = registry.String()
	expected := "[local github]"
	if str != expected {
		t.Errorf("Expected string representation to be %s, got %s", expected, str)
	}
}

// ConfigManager Tests

func TestNewConfigManager(t *testing.T) {
	path := "/test/config/path"
	manager := NewConfigManager(path)

	if manager == nil {
		t.Fatal("Expected NewConfigManager to return non-nil manager")
	}

	if manager.configPath != path {
		t.Errorf("Expected config path to be %s, got %s", path, manager.configPath)
	}
}

func TestConfigManager_LoadConfig_NonExistentFile(t *testing.T) {
	tempDir := t.TempDir()
	configPath := filepath.Join(tempDir, "nonexistent.yaml")

	manager := NewConfigManager(configPath)
	config, err := manager.LoadConfig()

	if err != nil {
		t.Fatalf("Expected no error for non-existent file, got: %v", err)
	}

	if config == nil {
		t.Fatal("Expected config to be non-nil")
	}

	if config.Version != "1.0" {
		t.Errorf("Expected version '1.0', got %s", config.Version)
	}

	if len(config.Servers) != 0 {
		t.Errorf("Expected empty servers slice, got %d servers", len(config.Servers))
	}
}

func TestConfigManager_LoadConfig_EmptyFile(t *testing.T) {
	tempDir := t.TempDir()
	configPath := filepath.Join(tempDir, "empty.yaml")

	// Create empty file
	err := os.WriteFile(configPath, []byte(""), 0644)
	if err != nil {
		t.Fatalf("Failed to create empty config file: %v", err)
	}

	manager := NewConfigManager(configPath)
	config, err := manager.LoadConfig()

	if err != nil {
		t.Fatalf("Expected no error for empty file, got: %v", err)
	}

	if config.Version != "1.0" {
		t.Errorf("Expected version '1.0', got %s", config.Version)
	}

	if len(config.Servers) != 0 {
		t.Errorf("Expected empty servers slice, got %d servers", len(config.Servers))
	}
}

func TestConfigManager_LoadConfig_ValidFile(t *testing.T) {
	tempDir := t.TempDir()
	configPath := filepath.Join(tempDir, "valid.yaml")

	validYAML := `version: "1.0"
servers:
  - id: "test-id"
    name: "Test Server"
    url: "https://rancher.example.com"
    username: "admin"
    authType: "local"
`

	err := os.WriteFile(configPath, []byte(validYAML), 0644)
	if err != nil {
		t.Fatalf("Failed to create valid config file: %v", err)
	}

	manager := NewConfigManager(configPath)
	config, err := manager.LoadConfig()

	if err != nil {
		t.Fatalf("Expected no error for valid file, got: %v", err)
	}

	if config.Version != "1.0" {
		t.Errorf("Expected version '1.0', got %s", config.Version)
	}

	if len(config.Servers) != 1 {
		t.Fatalf("Expected 1 server, got %d", len(config.Servers))
	}

	server := config.Servers[0]
	if server.ID != "test-id" {
		t.Errorf("Expected server ID 'test-id', got %s", server.ID)
	}
	if server.Name != "Test Server" {
		t.Errorf("Expected server name 'Test Server', got %s", server.Name)
	}
	if server.URL != "https://rancher.example.com" {
		t.Errorf("Expected server URL 'https://rancher.example.com', got %s", server.URL)
	}
	if server.Username != "admin" {
		t.Errorf("Expected username 'admin', got %s", server.Username)
	}
	if server.AuthType != "local" {
		t.Errorf("Expected auth type 'local', got %s", server.AuthType)
	}
}

func TestConfigManager_LoadConfig_InvalidYAML(t *testing.T) {
	tempDir := t.TempDir()
	configPath := filepath.Join(tempDir, "invalid.yaml")

	invalidYAML := `version: "1.0"
servers:
  - id: "test-id"
    name: "Test Server"
    url: "https://rancher.example.com"
    username: "admin"
    authType: "local"
  invalid_yaml_here: [
`

	err := os.WriteFile(configPath, []byte(invalidYAML), 0644)
	if err != nil {
		t.Fatalf("Failed to create invalid config file: %v", err)
	}

	manager := NewConfigManager(configPath)
	_, err = manager.LoadConfig()

	if err == nil {
		t.Error("Expected error for invalid YAML file")
	}

	// Check that it's a configuration error
	if !contains(err.Error(), "configuration error") {
		t.Errorf("Expected configuration error, got: %v", err)
	}
}

func TestConfigManager_SaveConfig(t *testing.T) {
	tempDir := t.TempDir()
	configPath := filepath.Join(tempDir, "subdir", "config.yaml")

	manager := NewConfigManager(configPath)
	config := &Config{
		Version: "1.0",
		Servers: []RancherServer{
			{
				ID:       "test-id",
				Name:     "Test Server",
				URL:      "https://rancher.example.com",
				Username: "admin",
				AuthType: "local",
			},
		},
	}

	err := manager.SaveConfig(config)
	if err != nil {
		t.Fatalf("Expected no error saving config, got: %v", err)
	}

	// Verify file was created
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		t.Error("Expected config file to be created")
	}

	// Verify content by loading it back
	loadedConfig, err := manager.LoadConfig()
	if err != nil {
		t.Fatalf("Failed to load saved config: %v", err)
	}

	if loadedConfig.Version != config.Version {
		t.Errorf("Expected version %s, got %s", config.Version, loadedConfig.Version)
	}

	if len(loadedConfig.Servers) != len(config.Servers) {
		t.Errorf("Expected %d servers, got %d", len(config.Servers), len(loadedConfig.Servers))
	}

	if len(loadedConfig.Servers) > 0 {
		savedServer := loadedConfig.Servers[0]
		originalServer := config.Servers[0]
		if savedServer.ID != originalServer.ID {
			t.Errorf("Expected server ID %s, got %s", originalServer.ID, savedServer.ID)
		}
	}
}

func TestConfigManager_AddServer(t *testing.T) {
	tempDir := t.TempDir()
	configPath := filepath.Join(tempDir, "config.yaml")

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
		t.Fatalf("Expected 1 server, got %d", len(config.Servers))
	}

	addedServer := config.Servers[0]
	if addedServer.ID == "" {
		t.Error("Expected server ID to be generated")
	}
	if addedServer.Name != server.Name {
		t.Errorf("Expected server name %s, got %s", server.Name, addedServer.Name)
	}
}

func TestConfigManager_AddServer_InvalidAuthType(t *testing.T) {
	tempDir := t.TempDir()
	configPath := filepath.Join(tempDir, "config.yaml")

	manager := NewConfigManager(configPath)
	server := RancherServer{
		Name:     "Test Server",
		URL:      "https://rancher.example.com",
		Username: "admin",
		AuthType: "invalid-auth-type",
	}

	err := manager.AddServer(server)
	if err == nil {
		t.Error("Expected error for invalid auth type")
	}

	if !contains(err.Error(), "validation error") {
		t.Errorf("Expected validation error, got: %v", err)
	}

	if !contains(err.Error(), "auth type must be one of") {
		t.Errorf("Expected validation message about supported types, got: %v", err)
	}
}

func TestConfigManager_RemoveServer(t *testing.T) {
	tempDir := t.TempDir()
	configPath := filepath.Join(tempDir, "config.yaml")

	manager := NewConfigManager(configPath)

	// Add a server first
	server := RancherServer{
		ID:       "test-id",
		Name:     "Test Server",
		URL:      "https://rancher.example.com",
		Username: "admin",
		AuthType: "local",
	}

	err := manager.AddServer(server)
	if err != nil {
		t.Fatalf("Failed to add server: %v", err)
	}

	// Remove the server
	err = manager.RemoveServer("test-id")
	if err != nil {
		t.Fatalf("Expected no error removing server, got: %v", err)
	}

	// Verify server was removed
	config, err := manager.LoadConfig()
	if err != nil {
		t.Fatalf("Failed to load config after removing server: %v", err)
	}

	if len(config.Servers) != 0 {
		t.Errorf("Expected 0 servers after removal, got %d", len(config.Servers))
	}
}

func TestConfigManager_RemoveServer_NotFound(t *testing.T) {
	tempDir := t.TempDir()
	configPath := filepath.Join(tempDir, "config.yaml")

	manager := NewConfigManager(configPath)

	err := manager.RemoveServer("nonexistent-id")
	if err == nil {
		t.Error("Expected error when removing non-existent server")
	}

	if !contains(err.Error(), "validation error") {
		t.Errorf("Expected validation error, got: %v", err)
	}

	if !contains(err.Error(), "server not found") {
		t.Errorf("Expected 'server not found' message, got: %v", err)
	}
}

func TestConfigManager_RemoveServerByURL(t *testing.T) {
	tempDir := t.TempDir()
	configPath := filepath.Join(tempDir, "config.yaml")

	manager := NewConfigManager(configPath)

	// Add a server first
	server := RancherServer{
		Name:     "Test Server",
		URL:      "https://rancher.example.com",
		Username: "admin",
		AuthType: "local",
	}

	err := manager.AddServer(server)
	if err != nil {
		t.Fatalf("Failed to add server: %v", err)
	}

	// Remove the server by URL
	err = manager.RemoveServerByURL("https://rancher.example.com")
	if err != nil {
		t.Fatalf("Expected no error removing server by URL, got: %v", err)
	}

	// Verify server was removed
	config, err := manager.LoadConfig()
	if err != nil {
		t.Fatalf("Failed to load config after removing server: %v", err)
	}

	if len(config.Servers) != 0 {
		t.Errorf("Expected 0 servers after removal, got %d", len(config.Servers))
	}
}

func TestConfigManager_RemoveServerByURL_NotFound(t *testing.T) {
	tempDir := t.TempDir()
	configPath := filepath.Join(tempDir, "config.yaml")

	manager := NewConfigManager(configPath)

	err := manager.RemoveServerByURL("https://nonexistent.example.com")
	if err == nil {
		t.Error("Expected error when removing server by non-existent URL")
	}

	if !contains(err.Error(), "validation error") {
		t.Errorf("Expected validation error, got: %v", err)
	}

	if !contains(err.Error(), "server not found") {
		t.Errorf("Expected 'server not found' message, got: %v", err)
	}
}

func TestConfigManager_GetServers(t *testing.T) {
	tempDir := t.TempDir()
	configPath := filepath.Join(tempDir, "config.yaml")

	manager := NewConfigManager(configPath)

	// Test empty config
	servers, err := manager.GetServers()
	if err != nil {
		t.Fatalf("Expected no error getting servers from empty config, got: %v", err)
	}

	if len(servers) != 0 {
		t.Errorf("Expected 0 servers from empty config, got %d", len(servers))
	}

	// Add servers
	server1 := RancherServer{Name: "Server 1", URL: "https://server1.com", Username: "admin", AuthType: "local"}
	server2 := RancherServer{Name: "Server 2", URL: "https://server2.com", Username: "admin", AuthType: "github"}

	err = manager.AddServer(server1)
	if err != nil {
		t.Fatalf("Failed to add server 1: %v", err)
	}

	err = manager.AddServer(server2)
	if err != nil {
		t.Fatalf("Failed to add server 2: %v", err)
	}

	// Get servers
	servers, err = manager.GetServers()
	if err != nil {
		t.Fatalf("Expected no error getting servers, got: %v", err)
	}

	if len(servers) != 2 {
		t.Errorf("Expected 2 servers, got %d", len(servers))
	}
}

// Integration Tests

func TestSupportedAuthTypes_Integration(t *testing.T) {
	// Test that the global SupportedAuthTypes registry is properly initialized
	if SupportedAuthTypes == nil {
		t.Fatal("Expected SupportedAuthTypes to be initialized")
	}

	// Test that it contains expected auth types
	expectedTypes := []string{"local", "github", "openldap", "activedirectory", "azuread", "okta", "ping", "keycloak", "shibboleth", "googleoauth"}

	for _, authType := range expectedTypes {
		if !SupportedAuthTypes.IsValid(authType) {
			t.Errorf("Expected auth type '%s' to be supported", authType)
		}

		provider, exists := SupportedAuthTypes.Get(authType)
		if !exists {
			t.Errorf("Expected to be able to get provider for '%s'", authType)
		}

		if provider.ID != authType {
			t.Errorf("Expected provider ID to be '%s', got '%s'", authType, provider.ID)
		}

		if provider.Provider == "" {
			t.Errorf("Expected provider '%s' to have non-empty Provider field", authType)
		}

		if provider.DisplayName == "" {
			t.Errorf("Expected provider '%s' to have non-empty DisplayName", authType)
		}
	}

	// Test that the order is maintained
	allProviders := SupportedAuthTypes.All()
	if len(allProviders) < len(expectedTypes) {
		t.Errorf("Expected at least %d providers, got %d", len(expectedTypes), len(allProviders))
	}

	// Test that openldap is specifically supported (user requirement)
	if !SupportedAuthTypes.IsValid("openldap") {
		t.Error("Expected 'openldap' to be supported as specified in requirements")
	}

	openldapProvider, _ := SupportedAuthTypes.Get("openldap")
	if openldapProvider.Provider != "openldap" {
		t.Errorf("Expected openldap provider to have 'openldap' as provider, got '%s'", openldapProvider.Provider)
	}
}

func TestIsValidAuthType(t *testing.T) {
	// Test valid auth types
	validTypes := []string{"local", "github", "openldap"}
	for _, authType := range validTypes {
		if !IsValidAuthType(authType) {
			t.Errorf("Expected '%s' to be valid", authType)
		}
	}

	// Test invalid auth types
	invalidTypes := []string{"invalid", "nonexistent", ""}
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
		if !contains(str, authType) {
			t.Errorf("Expected string to contain '%s', got: %s", authType, str)
		}
	}
}

// Helper function
func contains(s, substr string) bool {
	return len(s) >= len(substr) && s[:len(substr)] == substr ||
		len(s) > len(substr) && s[len(s)-len(substr):] == substr ||
		len(s) > len(substr) && stringContains(s, substr)
}

func stringContains(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

// Additional tests to improve coverage

func TestConfigManager_LoadConfig_ReadError(t *testing.T) {
	// Test error path when file exists but can't be read (permissions)
	tempDir := t.TempDir()
	configPath := filepath.Join(tempDir, "unreadable.yaml")

	// Create file first
	err := os.WriteFile(configPath, []byte("test content"), 0644)
	if err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	// Make file unreadable (this might not work on all systems)
	err = os.Chmod(configPath, 0000)
	if err != nil {
		t.Skip("Cannot change file permissions on this system")
	}
	defer func() { _ = os.Chmod(configPath, 0644) }() // Restore permissions for cleanup

	manager := NewConfigManager(configPath)
	_, err = manager.LoadConfig()

	if err == nil {
		t.Skip("Expected error reading unreadable file, but got none (permissions not enforced)")
	}

	if !contains(err.Error(), "configuration error") {
		t.Errorf("Expected configuration error, got: %v", err)
	}
}

func TestConfigManager_SaveConfig_DirectoryCreateError(t *testing.T) {
	// Test error path when directory cannot be created
	// Use a path that should fail to create (root directory on Unix systems)
	invalidPath := "/root/cannot_create/config.yaml"

	manager := NewConfigManager(invalidPath)
	config := &Config{
		Version: "1.0",
		Servers: []RancherServer{},
	}

	err := manager.SaveConfig(config)
	if err == nil {
		t.Skip("Expected error creating directory, but operation succeeded")
	}

	if !contains(err.Error(), "configuration error") {
		t.Errorf("Expected configuration error, got: %v", err)
	}
}

func TestConfigManager_SaveConfig_WriteError(t *testing.T) {
	// Test error path when file write fails
	tempDir := t.TempDir()
	configPath := filepath.Join(tempDir, "readonly", "config.yaml")

	// Create directory and make it readonly
	err := os.MkdirAll(filepath.Dir(configPath), 0755)
	if err != nil {
		t.Fatalf("Failed to create directory: %v", err)
	}

	err = os.Chmod(filepath.Dir(configPath), 0555) // Read and execute only
	if err != nil {
		t.Skip("Cannot change directory permissions on this system")
	}
	defer func() { _ = os.Chmod(filepath.Dir(configPath), 0755) }() // Restore permissions

	manager := NewConfigManager(configPath)
	config := &Config{
		Version: "1.0",
		Servers: []RancherServer{},
	}

	err = manager.SaveConfig(config)
	if err == nil {
		t.Skip("Expected error writing to readonly directory, but operation succeeded")
	}

	if !contains(err.Error(), "configuration error") {
		t.Errorf("Expected configuration error, got: %v", err)
	}
}

func TestConfigManager_AddServer_LoadConfigError(t *testing.T) {
	// Test error path when LoadConfig fails in AddServer
	// Use an invalid config file that will fail to unmarshal
	tempDir := t.TempDir()
	configPath := filepath.Join(tempDir, "invalid.yaml")

	// Create invalid YAML
	invalidYAML := `version: "1.0"
servers:
  - id: "test"
    invalid_yaml: [unclosed_bracket
`
	err := os.WriteFile(configPath, []byte(invalidYAML), 0644)
	if err != nil {
		t.Fatalf("Failed to create invalid config file: %v", err)
	}

	manager := NewConfigManager(configPath)
	server := RancherServer{
		Name:     "Test Server",
		URL:      "https://rancher.example.com",
		Username: "admin",
		AuthType: "local",
	}

	err = manager.AddServer(server)
	if err == nil {
		t.Error("Expected error when LoadConfig fails")
	}

	if !contains(err.Error(), "failed to load config") {
		t.Errorf("Expected 'failed to load config' error, got: %v", err)
	}
}

func TestConfigManager_AddServer_SaveConfigError(t *testing.T) {
	// Test error path when SaveConfig fails in AddServer
	tempDir := t.TempDir()

	// Start with a valid config
	configPath := filepath.Join(tempDir, "config.yaml")
	manager := NewConfigManager(configPath)

	// Create initial empty config
	err := manager.SaveConfig(&Config{Version: "1.0", Servers: []RancherServer{}})
	if err != nil {
		t.Fatalf("Failed to create initial config: %v", err)
	}

	// Make directory readonly to cause SaveConfig to fail
	err = os.Chmod(tempDir, 0555)
	if err != nil {
		t.Skip("Cannot change directory permissions on this system")
	}
	defer func() { _ = os.Chmod(tempDir, 0755) }() // Restore permissions

	server := RancherServer{
		Name:     "Test Server",
		URL:      "https://rancher.example.com",
		Username: "admin",
		AuthType: "local",
	}

	err = manager.AddServer(server)
	if err == nil {
		t.Skip("Expected error when SaveConfig fails, but operation succeeded")
	}

	// Should fail at SaveConfig, not at validation
	if contains(err.Error(), "validation error") {
		t.Errorf("Expected save error, not validation error, got: %v", err)
	}
}

func TestConfigManager_AddServer_ExistingID(t *testing.T) {
	// Test adding server with pre-existing ID
	tempDir := t.TempDir()
	configPath := filepath.Join(tempDir, "config.yaml")

	manager := NewConfigManager(configPath)
	server := RancherServer{
		ID:       "existing-id",
		Name:     "Test Server",
		URL:      "https://rancher.example.com",
		Username: "admin",
		AuthType: "local",
	}

	err := manager.AddServer(server)
	if err != nil {
		t.Fatalf("Expected no error adding server with ID, got: %v", err)
	}

	// Verify the ID was preserved
	config, err := manager.LoadConfig()
	if err != nil {
		t.Fatalf("Failed to load config: %v", err)
	}

	if len(config.Servers) != 1 {
		t.Fatalf("Expected 1 server, got %d", len(config.Servers))
	}

	if config.Servers[0].ID != "existing-id" {
		t.Errorf("Expected ID to be preserved as 'existing-id', got %s", config.Servers[0].ID)
	}
}

func TestConfigManager_RemoveServer_LoadConfigError(t *testing.T) {
	// Test error path when LoadConfig fails in RemoveServer
	tempDir := t.TempDir()
	configPath := filepath.Join(tempDir, "invalid.yaml")

	// Create invalid YAML
	invalidYAML := `version: "1.0"
servers:
  - invalid_yaml: [
`
	err := os.WriteFile(configPath, []byte(invalidYAML), 0644)
	if err != nil {
		t.Fatalf("Failed to create invalid config file: %v", err)
	}

	manager := NewConfigManager(configPath)
	err = manager.RemoveServer("any-id")

	if err == nil {
		t.Error("Expected error when LoadConfig fails")
	}

	if !contains(err.Error(), "failed to load config") {
		t.Errorf("Expected 'failed to load config' error, got: %v", err)
	}
}

func TestConfigManager_RemoveServerByURL_LoadConfigError(t *testing.T) {
	// Test error path when LoadConfig fails in RemoveServerByURL
	tempDir := t.TempDir()
	configPath := filepath.Join(tempDir, "invalid.yaml")

	// Create invalid YAML
	invalidYAML := `invalid yaml content`
	err := os.WriteFile(configPath, []byte(invalidYAML), 0644)
	if err != nil {
		t.Fatalf("Failed to create invalid config file: %v", err)
	}

	manager := NewConfigManager(configPath)
	err = manager.RemoveServerByURL("https://any.url.com")

	if err == nil {
		t.Error("Expected error when LoadConfig fails")
	}

	if !contains(err.Error(), "failed to load config") {
		t.Errorf("Expected 'failed to load config' error, got: %v", err)
	}
}

func TestConfigManager_GetServers_LoadConfigError(t *testing.T) {
	// Test error path when LoadConfig fails in GetServers
	tempDir := t.TempDir()
	configPath := filepath.Join(tempDir, "invalid.yaml")

	// Create invalid YAML that will fail to unmarshal
	invalidYAML := `version: "1.0"
servers:
  - id: "test"
    invalid_yaml: [unclosed_bracket
`
	err := os.WriteFile(configPath, []byte(invalidYAML), 0644)
	if err != nil {
		t.Fatalf("Failed to create invalid config file: %v", err)
	}

	manager := NewConfigManager(configPath)
	servers, err := manager.GetServers()

	if err == nil {
		t.Error("Expected error when LoadConfig fails")
	}

	if servers != nil {
		t.Error("Expected servers to be nil when error occurs")
	}

	// The error should propagate from LoadConfig
	if err != nil && !contains(err.Error(), "configuration error") {
		t.Errorf("Expected configuration error, got: %v", err)
	}
}
