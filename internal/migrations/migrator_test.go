package migrations

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"

	"cowpoke/internal/domain"
	"cowpoke/internal/mocks"
	"cowpoke/internal/testutil"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewMigrator(t *testing.T) {
	// Act
	migrator := NewMigrator(testutil.Logger())

	// Assert
	assert.NotNil(t, migrator)
}

func TestMigrator_detectVersion(t *testing.T) {
	migrator := NewMigrator(testutil.Logger())

	tests := []struct {
		name            string
		configData      string
		expectedVersion string
		shouldError     bool
	}{
		{
			name: "v2.0 config",
			configData: `version: "2.0"
servers:
  - url: "https://rancher.example.com"`,
			expectedVersion: "2.0",
			shouldError:     false,
		},
		{
			name: "v1.0 explicit",
			configData: `version: "1.0"
servers:
  - id: "test"`,
			expectedVersion: "1.0",
			shouldError:     false,
		},
		{
			name: "v1.0 implicit (no version field)",
			configData: `servers:
  - id: "test"
    name: "Test Server"`,
			expectedVersion: "1.0",
			shouldError:     false,
		},
		{
			name:            "empty config",
			configData:      `{}`,
			expectedVersion: "1.0", // Empty version defaults to 1.0
			shouldError:     false,
		},
		{
			name:            "invalid yaml",
			configData:      "{invalid yaml structure",
			expectedVersion: "",
			shouldError:     true,
		},
		{
			name: "future version",
			configData: `version: "3.0"
servers: []`,
			expectedVersion: "3.0",
			shouldError:     false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Act
			version, err := migrator.detectVersion([]byte(tt.configData))

			// Assert
			if tt.shouldError {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.expectedVersion, version)
			}
		})
	}
}

func TestMigrator_Migrate_NoMigrationNeeded(t *testing.T) {
	// Test when config is already current version
	migrator := NewMigrator(testutil.Logger())
	ctx := context.Background()

	configData := `version: "2.0"
servers:
  - url: "https://rancher.example.com"`

	// Act
	servers, wasMigrated, err := migrator.Migrate(ctx, []byte(configData), "2.0")

	// Assert
	require.NoError(t, err)
	assert.False(t, wasMigrated)
	assert.Nil(t, servers) // No servers returned when no migration needed
}

func TestMigrator_Migrate_FromV1_Success(t *testing.T) {
	// Test successful migration from v1 to v2
	migrator := NewMigrator(testutil.Logger())
	ctx := context.Background()

	v1ConfigData := `version: "1.0"
servers:
  - id: "server1"
    name: "Test Server 1"
    url: "https://rancher1.example.com"
    username: "admin"
    authType: "local"
  - id: "server2"
    name: "Test Server 2"
    url: "https://rancher2.example.com"
    username: "user"
    authType: "ldap"`

	// Act
	servers, wasMigrated, err := migrator.Migrate(ctx, []byte(v1ConfigData), "2.0")

	// Assert
	require.NoError(t, err)
	assert.True(t, wasMigrated)
	assert.Len(t, servers, 2)

	// Verify first server migration
	assert.Equal(t, "https://rancher1.example.com", servers[0].URL)
	assert.Equal(t, "admin", servers[0].Username)
	assert.Equal(t, "local", servers[0].AuthType)

	// Verify second server migration
	assert.Equal(t, "https://rancher2.example.com", servers[1].URL)
	assert.Equal(t, "user", servers[1].Username)
	assert.Equal(t, "ldap", servers[1].AuthType)

	// Verify ID generation works in v2 (not stored in config)
	assert.NotEmpty(t, servers[0].ID())
	assert.NotEmpty(t, servers[1].ID())
}

func TestMigrator_Migrate_FromV1_ImplicitVersion(t *testing.T) {
	// Test migration from v1 config without explicit version field
	migrator := NewMigrator(testutil.Logger())
	ctx := context.Background()

	v1ConfigData := `servers:
  - id: "old-server"
    name: "Legacy Server"
    url: "https://legacy.example.com"
    username: "admin"
    authType: "local"`

	// Act
	servers, wasMigrated, err := migrator.Migrate(ctx, []byte(v1ConfigData), "2.0")

	// Assert
	require.NoError(t, err)
	assert.True(t, wasMigrated)
	assert.Len(t, servers, 1)
	assert.Equal(t, "https://legacy.example.com", servers[0].URL)
	assert.Equal(t, "admin", servers[0].Username)
	assert.Equal(t, "local", servers[0].AuthType)
}

func TestMigrator_Migrate_UnsupportedVersion(t *testing.T) {
	// Test error handling for unsupported version
	migrator := NewMigrator(testutil.Logger())
	ctx := context.Background()

	futureConfigData := `version: "3.0"
servers: []`

	// Act
	servers, wasMigrated, err := migrator.Migrate(ctx, []byte(futureConfigData), "2.0")

	// Assert
	require.Error(t, err)
	assert.False(t, wasMigrated)
	assert.Nil(t, servers)
	assert.Contains(t, err.Error(), "unsupported configuration version: 3.0")
}

func TestMigrator_Migrate_InvalidYAML(t *testing.T) {
	// Test error handling for invalid YAML
	migrator := NewMigrator(testutil.Logger())
	ctx := context.Background()

	invalidYAML := `version: "1.0"
servers:
  - invalid structure [[[`

	// Act
	servers, wasMigrated, err := migrator.Migrate(ctx, []byte(invalidYAML), "2.0")

	// Assert
	require.Error(t, err)
	assert.False(t, wasMigrated)
	assert.Nil(t, servers)
	assert.Contains(t, err.Error(), "failed to migrate from v1")
}

func TestMigrator_Migrate_V1InvalidData(t *testing.T) {
	// Test error handling when v1 data is malformed
	migrator := NewMigrator(testutil.Logger())
	ctx := context.Background()

	malformedV1Data := `version: "1.0"
servers:
  - invalid structure [[[`

	// Act
	servers, wasMigrated, err := migrator.Migrate(ctx, []byte(malformedV1Data), "2.0")

	// Assert
	require.Error(t, err)
	assert.False(t, wasMigrated)
	assert.Nil(t, servers)
	assert.Contains(t, err.Error(), "failed to migrate from v1")
}

func TestMigrator_FixPermissionsPostMigration_Success(t *testing.T) {
	// Test successful permission fixing
	migrator := NewMigrator(testutil.Logger())
	ctx := context.Background()
	mockFS := mocks.NewMockFileSystemAdapter(t)

	tmpDir := t.TempDir()
	configDir := tmpDir + "/.cowpoke"
	configPath := configDir + "/config.yaml"
	// Calculate the actual kubeconfig dir path that the migrator will use
	kubeconfigDir := filepath.Join(filepath.Dir(configDir), "..", ".kube")

	// Mock successful permission changes
	mockFS.On("Chmod", configPath, os.FileMode(0o600)).Return(nil)
	mockFS.On("Chmod", configDir, os.FileMode(0o700)).Return(nil)
	mockFS.On("Stat", kubeconfigDir).Return(nil, nil) // Directory exists
	mockFS.On("Chmod", kubeconfigDir, os.FileMode(0o700)).Return(nil)

	// Act
	err := migrator.FixPermissionsPostMigration(ctx, configPath, mockFS)

	// Assert
	require.NoError(t, err)
	mockFS.AssertExpectations(t)
}

func TestMigrator_FixPermissionsPostMigration_ConfigFileError(t *testing.T) {
	// Test error when config file permission change fails
	migrator := NewMigrator(testutil.Logger())
	ctx := context.Background()
	mockFS := mocks.NewMockFileSystemAdapter(t)

	tmpDir := t.TempDir()
	configPath := tmpDir + "/.cowpoke/config.yaml"
	expectedErr := errors.New("permission denied")

	mockFS.On("Chmod", configPath, os.FileMode(0o600)).Return(expectedErr)

	// Act
	err := migrator.FixPermissionsPostMigration(ctx, configPath, mockFS)

	// Assert
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to fix config file permissions")
	assert.Contains(t, err.Error(), expectedErr.Error())
	mockFS.AssertExpectations(t)
}

func TestMigrator_FixPermissionsPostMigration_ConfigDirError(t *testing.T) {
	// Test error when config directory permission change fails
	migrator := NewMigrator(testutil.Logger())
	ctx := context.Background()
	mockFS := mocks.NewMockFileSystemAdapter(t)

	tmpDir := t.TempDir()
	configDir := tmpDir + "/.cowpoke"
	configPath := configDir + "/config.yaml"
	expectedErr := errors.New("directory permission denied")

	mockFS.On("Chmod", configPath, os.FileMode(0o600)).Return(nil)
	mockFS.On("Chmod", configDir, os.FileMode(0o700)).Return(expectedErr)

	// Act
	err := migrator.FixPermissionsPostMigration(ctx, configPath, mockFS)

	// Assert
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to fix config directory permissions")
	assert.Contains(t, err.Error(), expectedErr.Error())
	mockFS.AssertExpectations(t)
}

func TestMigrator_FixPermissionsPostMigration_KubeconfigDirNotExists(t *testing.T) {
	// Test behavior when kubeconfig directory doesn't exist (should not error)
	migrator := NewMigrator(testutil.Logger())
	ctx := context.Background()
	mockFS := mocks.NewMockFileSystemAdapter(t)

	tmpDir := t.TempDir()
	configDir := tmpDir + "/.cowpoke"
	configPath := configDir + "/config.yaml"
	// Calculate the actual kubeconfig dir path that the migrator will use
	kubeconfigDir := filepath.Join(filepath.Dir(configDir), "..", ".kube")

	mockFS.On("Chmod", configPath, os.FileMode(0o600)).Return(nil)
	mockFS.On("Chmod", configDir, os.FileMode(0o700)).Return(nil)
	mockFS.On("Stat", kubeconfigDir).Return(nil, os.ErrNotExist) // Directory doesn't exist

	// Act
	err := migrator.FixPermissionsPostMigration(ctx, configPath, mockFS)

	// Assert
	require.NoError(t, err) // Should not error when kubeconfig dir doesn't exist
	mockFS.AssertExpectations(t)
}

func TestMigrator_FixPermissionsPostMigration_KubeconfigDirError(t *testing.T) {
	// Test that kubeconfig directory permission errors don't fail the migration
	migrator := NewMigrator(testutil.Logger())
	ctx := context.Background()
	mockFS := mocks.NewMockFileSystemAdapter(t)

	tmpDir := t.TempDir()
	configDir := tmpDir + "/.cowpoke"
	configPath := configDir + "/config.yaml"
	// Calculate the actual kubeconfig dir path that the migrator will use
	kubeconfigDir := filepath.Join(filepath.Dir(configDir), "..", ".kube")
	kubeconfigErr := errors.New("kubeconfig permission denied")

	mockFS.On("Chmod", configPath, os.FileMode(0o600)).Return(nil)
	mockFS.On("Chmod", configDir, os.FileMode(0o700)).Return(nil)
	mockFS.On("Stat", kubeconfigDir).Return(nil, nil) // Directory exists
	mockFS.On("Chmod", kubeconfigDir, os.FileMode(0o700)).Return(kubeconfigErr)

	// Act
	err := migrator.FixPermissionsPostMigration(ctx, configPath, mockFS)

	// Assert
	require.NoError(t, err) // Should not fail for kubeconfig directory errors
	mockFS.AssertExpectations(t)
}

func TestMigrator_Integration_FullV1Migration(t *testing.T) {
	// Full integration test of v1 to v2 migration
	migrator := NewMigrator(testutil.Logger())
	ctx := context.Background()

	// Realistic v1 config with multiple servers and different auth types
	v1ConfigData := `servers:
  - id: "prod-server"
    name: "Production Rancher"
    url: "https://rancher-prod.company.com"
    username: "admin"
    authType: "local"
  - id: "staging-server"
    name: "Staging Environment"
    url: "https://rancher-staging.company.com"
    username: "service-account"
    authType: "ldap"
  - id: "dev-server"
    name: "Development Cluster"
    url: "https://rancher-dev.company.com"
    username: "developer"
    authType: "activedirectory"`

	// Act
	servers, wasMigrated, err := migrator.Migrate(ctx, []byte(v1ConfigData), "2.0")

	// Assert
	require.NoError(t, err)
	assert.True(t, wasMigrated)
	assert.Len(t, servers, 3)

	// Verify all servers migrated correctly
	serversByURL := make(map[string]domain.ConfigServer)
	for _, server := range servers {
		serversByURL[server.URL] = server
	}

	// Check production server
	prodServer := serversByURL["https://rancher-prod.company.com"]
	assert.Equal(t, "admin", prodServer.Username)
	assert.Equal(t, "local", prodServer.AuthType)
	assert.NotEmpty(t, prodServer.ID())

	// Check staging server
	stagingServer := serversByURL["https://rancher-staging.company.com"]
	assert.Equal(t, "service-account", stagingServer.Username)
	assert.Equal(t, "ldap", stagingServer.AuthType)
	assert.NotEmpty(t, stagingServer.ID())

	// Check development server
	devServer := serversByURL["https://rancher-dev.company.com"]
	assert.Equal(t, "developer", devServer.Username)
	assert.Equal(t, "activedirectory", devServer.AuthType)
	assert.NotEmpty(t, devServer.ID())

	// Verify all IDs are unique
	ids := make(map[string]bool)
	for _, server := range servers {
		assert.False(t, ids[server.ID()], "Duplicate ID found: %s", server.ID())
		ids[server.ID()] = true
	}
}

func TestMigrator_EdgeCases(t *testing.T) {
	migrator := NewMigrator(testutil.Logger())
	ctx := context.Background()

	tests := []struct {
		name        string
		configData  string
		expectError bool
		description string
	}{
		{
			name: "empty servers list",
			configData: `version: "1.0"
servers: []`,
			expectError: false,
			description: "Should handle empty servers list",
		},
		{
			name: "servers with empty fields",
			configData: `version: "1.0"
servers:
  - id: ""
    name: ""
    url: "https://example.com"
    username: ""
    authType: ""`,
			expectError: false,
			description: "Should handle servers with empty fields",
		},
		{
			name: "servers with special characters in URL",
			configData: `version: "1.0"
servers:
  - id: "special"
    name: "Special Server"
    url: "https://rancher-test.example.com:8443/v3"
    username: "test@domain.com"
    authType: "openldap"`,
			expectError: false,
			description: "Should handle URLs with ports and paths",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Act
			servers, wasMigrated, err := migrator.Migrate(ctx, []byte(tt.configData), "2.0")

			// Assert
			if tt.expectError {
				assert.Error(t, err, tt.description)
			} else {
				require.NoError(t, err, tt.description)
				assert.True(t, wasMigrated)
				assert.NotNil(t, servers)
			}
		})
	}
}
