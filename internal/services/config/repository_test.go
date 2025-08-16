package config

import (
	"context"
	"errors"
	"os"
	"testing"

	"cowpoke/internal/domain"
	"cowpoke/internal/migrations"
	"cowpoke/internal/mocks"
	"cowpoke/internal/testutil"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"
)

func TestNewRepository_Success(t *testing.T) {
	// Arrange
	mockFS := mocks.NewMockFileSystemAdapter(t)
	configPath := "/home/user/.cowpoke/config.yaml"
	configDir := "/home/user/.cowpoke"

	mockFS.On("MkdirAll", configDir, os.FileMode(0o700)).Return(nil)
	mockFS.On("ReadFile", configPath).Return(nil, os.ErrNotExist)

	// Act
	repo, err := NewRepository(mockFS, configPath, testutil.Logger())

	// Assert
	require.NoError(t, err)
	assert.NotNil(t, repo)
	assert.Equal(t, mockFS, repo.fs)
	assert.Equal(t, configPath, repo.configPath)
	assert.Equal(t, "2.0", repo.config.Version)
	assert.Empty(t, repo.config.Servers)
	mockFS.AssertExpectations(t)
}

func TestNewRepository_DirectoryCreationError(t *testing.T) {
	// Arrange
	mockFS := mocks.NewMockFileSystemAdapter(t)
	configPath := "/root/restricted/config.yaml"
	configDir := "/root/restricted"

	expectedErr := errors.New("permission denied")
	mockFS.On("MkdirAll", configDir, os.FileMode(0o700)).Return(expectedErr)

	// Act
	repo, err := NewRepository(mockFS, configPath, testutil.Logger())

	// Assert
	require.Error(t, err)
	assert.Nil(t, repo)
	assert.Contains(t, err.Error(), "failed to create config directory")
	assert.Contains(t, err.Error(), expectedErr.Error())
	mockFS.AssertExpectations(t)
}

func TestNewRepository_LoadExistingConfig(t *testing.T) {
	// Arrange
	mockFS := mocks.NewMockFileSystemAdapter(t)
	configPath := "/home/user/.cowpoke/config.yaml"

	existingConfig := Config{
		Version: "2.0",
		Servers: []domain.ConfigServer{
			{URL: "https://rancher.example.com", Username: "admin", AuthType: "local"},
		},
	}
	configData, _ := yaml.Marshal(existingConfig)

	mockFS.On("MkdirAll", mock.Anything, mock.Anything).Return(nil)
	mockFS.On("ReadFile", configPath).Return(configData, nil)

	// Act
	repo, err := NewRepository(mockFS, configPath, testutil.Logger())

	// Assert
	require.NoError(t, err)
	assert.NotNil(t, repo)
	assert.Equal(t, "2.0", repo.config.Version)
	assert.Len(t, repo.config.Servers, 1)
	assert.Equal(t, "https://rancher.example.com", repo.config.Servers[0].URL)
	mockFS.AssertExpectations(t)
}

func TestGetServers_Success(t *testing.T) {
	// Arrange
	mockFS := mocks.NewMockFileSystemAdapter(t)
	repo := &Repository{
		fs:     mockFS,
		logger: testutil.Logger(),
		config: &Config{
			Version: "2.0",
			Servers: []domain.ConfigServer{
				{URL: "https://rancher1.example.com", Username: "admin", AuthType: "local"},
				{URL: "https://rancher2.example.com", Username: "user", AuthType: "ldap"},
			},
		},
	}
	ctx := context.Background()

	// Act
	servers, err := repo.GetServers(ctx)

	// Assert
	require.NoError(t, err)
	assert.Len(t, servers, 2)
	assert.Equal(t, "https://rancher1.example.com", servers[0].URL)
	assert.Equal(t, "https://rancher2.example.com", servers[1].URL)
}

func TestGetServers_EmptyConfig(t *testing.T) {
	// Arrange
	mockFS := mocks.NewMockFileSystemAdapter(t)
	repo := &Repository{
		fs:     mockFS,
		logger: testutil.Logger(),
		config: &Config{Version: "2.0", Servers: []domain.ConfigServer{}},
	}
	ctx := context.Background()

	// Act
	servers, err := repo.GetServers(ctx)

	// Assert
	require.NoError(t, err)
	assert.Empty(t, servers)
}

func TestAddServer_Success(t *testing.T) {
	// Arrange
	mockFS := mocks.NewMockFileSystemAdapter(t)
	logger := testutil.Logger()
	repo := &Repository{
		fs:         mockFS,
		configPath: "/test/config.yaml",
		logger:     logger,
		migrator:   migrations.NewMigrator(logger),
		config:     &Config{Version: "2.0", Servers: []domain.ConfigServer{}},
	}

	newServer := domain.ConfigServer{
		URL:      "https://rancher.example.com",
		Username: "admin",
		AuthType: "local",
	}

	mockFS.On("WriteFile", "/test/config.yaml", mock.MatchedBy(func(data []byte) bool {
		var config Config
		yaml.Unmarshal(data, &config)
		return len(config.Servers) == 1 && config.Servers[0].URL == newServer.URL
	}), os.FileMode(0o600)).Return(nil)

	ctx := context.Background()

	// Act
	err := repo.AddServer(ctx, newServer)

	// Assert
	require.NoError(t, err)
	assert.Len(t, repo.config.Servers, 1)
	assert.Equal(t, newServer.URL, repo.config.Servers[0].URL)
	mockFS.AssertExpectations(t)
}

func TestAddServer_DuplicateURL(t *testing.T) {
	// Arrange
	mockFS := mocks.NewMockFileSystemAdapter(t)
	existingServer := domain.ConfigServer{
		URL:      "https://rancher.example.com",
		Username: "admin",
		AuthType: "local",
	}

	repo := &Repository{
		fs:         mockFS,
		configPath: "/test/config.yaml",
		logger:     testutil.Logger(),
		config:     &Config{Version: "2.0", Servers: []domain.ConfigServer{existingServer}},
	}

	duplicateServer := domain.ConfigServer{
		URL:      "https://rancher.example.com", // Same URL
		Username: "user",
		AuthType: "ldap",
	}

	ctx := context.Background()

	// Act
	err := repo.AddServer(ctx, duplicateServer)

	// Assert
	require.Error(t, err)
	assert.Contains(t, err.Error(), "server https://rancher.example.com already exists")
	assert.Len(t, repo.config.Servers, 1) // No change
	// No filesystem operations should occur
	mockFS.AssertExpectations(t)
}

func TestAddServer_SaveError_Rollback(t *testing.T) {
	// Arrange
	mockFS := mocks.NewMockFileSystemAdapter(t)
	logger := testutil.Logger()
	repo := &Repository{
		fs:         mockFS,
		configPath: "/test/config.yaml",
		logger:     logger,
		migrator:   migrations.NewMigrator(logger),
		config:     &Config{Version: "2.0", Servers: []domain.ConfigServer{}},
	}

	newServer := domain.ConfigServer{
		URL:      "https://rancher.example.com",
		Username: "admin",
		AuthType: "local",
	}

	expectedErr := errors.New("disk full")
	mockFS.On("WriteFile", mock.Anything, mock.Anything, mock.Anything).Return(expectedErr)

	ctx := context.Background()

	// Act
	err := repo.AddServer(ctx, newServer)

	// Assert
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to save configuration")
	assert.Contains(t, err.Error(), expectedErr.Error())
	assert.Empty(t, repo.config.Servers) // Rollback occurred
	mockFS.AssertExpectations(t)
}

func TestRemoveServer_Success(t *testing.T) {
	// Arrange
	mockFS := mocks.NewMockFileSystemAdapter(t)
	serverToRemove := domain.ConfigServer{
		URL:      "https://rancher1.example.com",
		Username: "admin",
		AuthType: "local",
	}
	serverToKeep := domain.ConfigServer{
		URL:      "https://rancher2.example.com",
		Username: "user",
		AuthType: "ldap",
	}

	repo := &Repository{
		fs:         mockFS,
		configPath: "/test/config.yaml",
		logger:     testutil.Logger(),
		config:     &Config{Version: "2.0", Servers: []domain.ConfigServer{serverToRemove, serverToKeep}},
	}

	mockFS.On("WriteFile", "/test/config.yaml", mock.MatchedBy(func(data []byte) bool {
		var config Config
		yaml.Unmarshal(data, &config)
		return len(config.Servers) == 1 && config.Servers[0].URL == serverToKeep.URL
	}), os.FileMode(0o600)).Return(nil)

	ctx := context.Background()

	// Act
	err := repo.RemoveServer(ctx, serverToRemove.URL)

	// Assert
	require.NoError(t, err)
	assert.Len(t, repo.config.Servers, 1)
	assert.Equal(t, serverToKeep.URL, repo.config.Servers[0].URL)
	mockFS.AssertExpectations(t)
}

func TestRemoveServer_NotFound(t *testing.T) {
	// Arrange
	mockFS := mocks.NewMockFileSystemAdapter(t)
	existingServer := domain.ConfigServer{
		URL:      "https://rancher.example.com",
		Username: "admin",
		AuthType: "local",
	}

	repo := &Repository{
		fs:         mockFS,
		configPath: "/test/config.yaml",
		logger:     testutil.Logger(),
		config:     &Config{Version: "2.0", Servers: []domain.ConfigServer{existingServer}},
	}

	ctx := context.Background()

	// Act
	err := repo.RemoveServer(ctx, "https://nonexistent.example.com")

	// Assert
	require.Error(t, err)
	assert.Contains(t, err.Error(), "server https://nonexistent.example.com not found")
	assert.Len(t, repo.config.Servers, 1) // No change
	// No filesystem operations should occur
	mockFS.AssertExpectations(t)
}

func TestRemoveServer_SaveError_Rollback(t *testing.T) {
	// Arrange
	mockFS := mocks.NewMockFileSystemAdapter(t)
	serverToRemove := domain.ConfigServer{
		URL:      "https://rancher.example.com",
		Username: "admin",
		AuthType: "local",
	}

	repo := &Repository{
		fs:         mockFS,
		configPath: "/test/config.yaml",
		logger:     testutil.Logger(),
		config:     &Config{Version: "2.0", Servers: []domain.ConfigServer{serverToRemove}},
	}

	expectedErr := errors.New("write failure")
	mockFS.On("WriteFile", mock.Anything, mock.Anything, mock.Anything).Return(expectedErr)

	ctx := context.Background()

	// Act
	err := repo.RemoveServer(ctx, serverToRemove.URL)

	// Assert
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to save configuration")
	assert.Contains(t, err.Error(), expectedErr.Error())
	assert.Len(t, repo.config.Servers, 1) // Rollback occurred
	assert.Equal(t, serverToRemove.URL, repo.config.Servers[0].URL)
	mockFS.AssertExpectations(t)
}

func TestRemoveServerByID_Success(t *testing.T) {
	// Arrange
	mockFS := mocks.NewMockFileSystemAdapter(t)
	serverToRemove := domain.ConfigServer{
		URL:      "https://rancher1.example.com",
		Username: "admin",
		AuthType: "local",
	}
	serverToKeep := domain.ConfigServer{
		URL:      "https://rancher2.example.com",
		Username: "user",
		AuthType: "ldap",
	}

	repo := &Repository{
		fs:         mockFS,
		configPath: "/test/config.yaml",
		logger:     testutil.Logger(),
		config:     &Config{Version: "2.0", Servers: []domain.ConfigServer{serverToRemove, serverToKeep}},
	}

	mockFS.On("WriteFile", "/test/config.yaml", mock.MatchedBy(func(data []byte) bool {
		var config Config
		yaml.Unmarshal(data, &config)
		return len(config.Servers) == 1 && config.Servers[0].URL == serverToKeep.URL
	}), os.FileMode(0o600)).Return(nil)

	ctx := context.Background()

	// Act
	err := repo.RemoveServerByID(ctx, serverToRemove.ID())

	// Assert
	require.NoError(t, err)
	assert.Len(t, repo.config.Servers, 1)
	assert.Equal(t, serverToKeep.URL, repo.config.Servers[0].URL)
	mockFS.AssertExpectations(t)
}

func TestRemoveServerByID_NotFound(t *testing.T) {
	// Arrange
	mockFS := mocks.NewMockFileSystemAdapter(t)
	existingServer := domain.ConfigServer{
		URL:      "https://rancher.example.com",
		Username: "admin",
		AuthType: "local",
	}

	repo := &Repository{
		fs:         mockFS,
		configPath: "/test/config.yaml",
		logger:     testutil.Logger(),
		config:     &Config{Version: "2.0", Servers: []domain.ConfigServer{existingServer}},
	}

	ctx := context.Background()

	// Act
	err := repo.RemoveServerByID(ctx, "nonexistent")

	// Assert
	require.Error(t, err)
	assert.Contains(t, err.Error(), "server with ID nonexistent not found")
	assert.Len(t, repo.config.Servers, 1) // No change
	// No filesystem operations should occur
	mockFS.AssertExpectations(t)
}

func TestRemoveServerByID_SaveError_Rollback(t *testing.T) {
	// Arrange
	mockFS := mocks.NewMockFileSystemAdapter(t)
	serverToRemove := domain.ConfigServer{
		URL:      "https://rancher.example.com",
		Username: "admin",
		AuthType: "local",
	}

	repo := &Repository{
		fs:         mockFS,
		configPath: "/test/config.yaml",
		logger:     testutil.Logger(),
		config:     &Config{Version: "2.0", Servers: []domain.ConfigServer{serverToRemove}},
	}

	expectedErr := errors.New("write failure")
	mockFS.On("WriteFile", mock.Anything, mock.Anything, mock.Anything).Return(expectedErr)

	ctx := context.Background()

	// Act
	err := repo.RemoveServerByID(ctx, serverToRemove.ID())

	// Assert
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to save configuration")
	assert.Contains(t, err.Error(), expectedErr.Error())
	assert.Len(t, repo.config.Servers, 1) // Rollback occurred
	assert.Equal(t, serverToRemove.URL, repo.config.Servers[0].URL)
	mockFS.AssertExpectations(t)
}

func TestSaveConfig_Success(t *testing.T) {
	// Arrange
	mockFS := mocks.NewMockFileSystemAdapter(t)
	config := &Config{
		Version: "2.0",
		Servers: []domain.ConfigServer{
			{URL: "https://rancher.example.com", Username: "admin", AuthType: "local"},
		},
	}

	repo := &Repository{
		fs:         mockFS,
		configPath: "/test/config.yaml",
		logger:     testutil.Logger(),
		config:     config,
	}

	mockFS.On("WriteFile", "/test/config.yaml", mock.MatchedBy(func(data []byte) bool {
		var savedConfig Config
		err := yaml.Unmarshal(data, &savedConfig)
		return err == nil && savedConfig.Version == "2.0" && len(savedConfig.Servers) == 1
	}), os.FileMode(0o600)).Return(nil)

	ctx := context.Background()

	// Act
	err := repo.SaveConfig(ctx)

	// Assert
	require.NoError(t, err)
	mockFS.AssertExpectations(t)
}

func TestSaveConfig_WriteError(t *testing.T) {
	// Arrange
	mockFS := mocks.NewMockFileSystemAdapter(t)
	logger := testutil.Logger()
	repo := &Repository{
		fs:         mockFS,
		configPath: "/test/config.yaml",
		logger:     logger,
		migrator:   migrations.NewMigrator(logger),
		config:     &Config{Version: "2.0", Servers: []domain.ConfigServer{}},
	}

	expectedErr := errors.New("disk full")
	mockFS.On("WriteFile", mock.Anything, mock.Anything, mock.Anything).Return(expectedErr)

	ctx := context.Background()

	// Act
	err := repo.SaveConfig(ctx)

	// Assert
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to write configuration file")
	assert.Contains(t, err.Error(), expectedErr.Error())
	mockFS.AssertExpectations(t)
}

func TestLoadConfig_FileNotExists(t *testing.T) {
	// Arrange
	mockFS := mocks.NewMockFileSystemAdapter(t)
	logger := testutil.Logger()
	repo := &Repository{
		fs:         mockFS,
		configPath: "/test/config.yaml",
		logger:     logger,
		migrator:   migrations.NewMigrator(logger),
		config:     &Config{Version: "2.0", Servers: []domain.ConfigServer{}},
	}

	mockFS.On("ReadFile", "/test/config.yaml").Return(nil, os.ErrNotExist)

	ctx := context.Background()

	// Act
	err := repo.LoadConfig(ctx)

	// Assert
	require.ErrorIs(t, err, os.ErrNotExist)
	mockFS.AssertExpectations(t)
}

func TestLoadConfig_Success(t *testing.T) {
	// Arrange
	mockFS := mocks.NewMockFileSystemAdapter(t)
	configData := `version: "2.0"
servers:
  - url: "https://rancher.example.com"
    username: "admin"
    authType: "local"`

	logger := testutil.Logger()
	repo := &Repository{
		fs:         mockFS,
		configPath: "/test/config.yaml",
		logger:     logger,
		migrator:   migrations.NewMigrator(logger),
		config:     &Config{Version: "2.0", Servers: []domain.ConfigServer{}},
	}

	mockFS.On("ReadFile", "/test/config.yaml").Return([]byte(configData), nil)

	ctx := context.Background()

	// Act
	err := repo.LoadConfig(ctx)

	// Assert
	require.NoError(t, err)
	assert.Equal(t, "2.0", repo.config.Version)
	assert.Len(t, repo.config.Servers, 1)
	assert.Equal(t, "https://rancher.example.com", repo.config.Servers[0].URL)
	mockFS.AssertExpectations(t)
}

func TestLoadConfig_ReadError(t *testing.T) {
	// Arrange
	mockFS := mocks.NewMockFileSystemAdapter(t)
	logger := testutil.Logger()
	repo := &Repository{
		fs:         mockFS,
		configPath: "/test/config.yaml",
		logger:     logger,
		migrator:   migrations.NewMigrator(logger),
		config:     &Config{Version: "2.0", Servers: []domain.ConfigServer{}},
	}

	expectedErr := errors.New("permission denied")
	mockFS.On("ReadFile", "/test/config.yaml").Return(nil, expectedErr)

	ctx := context.Background()

	// Act
	err := repo.LoadConfig(ctx)

	// Assert
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to read configuration file")
	assert.Contains(t, err.Error(), expectedErr.Error())
	mockFS.AssertExpectations(t)
}

func TestLoadConfig_InvalidYAML(t *testing.T) {
	// Arrange
	mockFS := mocks.NewMockFileSystemAdapter(t)
	invalidYAML := `version: "2.0"
servers: {invalid yaml structure`

	logger := testutil.Logger()
	repo := &Repository{
		fs:         mockFS,
		configPath: "/test/config.yaml",
		logger:     logger,
		migrator:   migrations.NewMigrator(logger),
		config:     &Config{Version: "2.0", Servers: []domain.ConfigServer{}},
	}

	mockFS.On("ReadFile", "/test/config.yaml").Return([]byte(invalidYAML), nil)

	ctx := context.Background()

	// Act
	err := repo.LoadConfig(ctx)

	// Assert
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to unmarshal configuration")
	mockFS.AssertExpectations(t)
}

func TestRepository_EmptyServersList(t *testing.T) {
	// Test behavior with empty servers list vs nil servers list

	// Arrange
	mockFS := mocks.NewMockFileSystemAdapter(t)
	repo := &Repository{
		fs:     mockFS,
		logger: testutil.Logger(),
		config: &Config{Version: "2.0", Servers: []domain.ConfigServer{}},
	}
	ctx := context.Background()

	// Act
	servers, err := repo.GetServers(ctx)

	// Assert
	require.NoError(t, err)
	assert.NotNil(t, servers) // Empty slice, not nil
	assert.Empty(t, servers)
}

func TestRepository_ServerIDConsistency(t *testing.T) {
	// Test that server IDs are consistent between operations

	// Arrange
	mockFS := mocks.NewMockFileSystemAdapter(t)
	server := domain.ConfigServer{
		URL:      "https://rancher1.example.com",
		Username: "admin",
		AuthType: "local",
	}

	logger := testutil.Logger()
	repo := &Repository{
		fs:         mockFS,
		configPath: "/test/config.yaml",
		logger:     logger,
		migrator:   migrations.NewMigrator(logger),
		config:     &Config{Version: "2.0", Servers: []domain.ConfigServer{server}},
	}

	ctx := context.Background()
	expectedID := server.ID() // e737f2fb for rancher1.example.com

	// Act & Assert - Get servers and verify ID
	servers, err := repo.GetServers(ctx)
	require.NoError(t, err)
	assert.Equal(t, expectedID, servers[0].ID())

	// Mock the save operation for removal
	mockFS.On("WriteFile", mock.Anything, mock.Anything, mock.Anything).Return(nil)

	// Act & Assert - Remove by ID using the same ID
	err = repo.RemoveServerByID(ctx, expectedID)
	require.NoError(t, err)
	assert.Empty(t, repo.config.Servers)

	mockFS.AssertExpectations(t)
}
