package config_test

import (
	"context"
	"errors"
	"log/slog"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"

	"cowpoke/internal/domain"
	"cowpoke/internal/mocks"
	"cowpoke/internal/services/config"
)

// RepositoryTestSuite provides common setup for repository tests.
type RepositoryTestSuite struct {
	suite.Suite

	ctx        context.Context
	logger     *slog.Logger
	mockFS     *mocks.MockFileSystemAdapter
	tempDir    string
	configPath string
}

func (s *RepositoryTestSuite) SetupTest() {
	s.ctx = context.Background()
	s.logger = slog.Default()
	s.mockFS = mocks.NewMockFileSystemAdapter(s.T())
	s.tempDir = s.T().TempDir()
	s.configPath = s.tempDir + "/.config/cowpoke/config.yaml"
}

// Helper to create a repository with current setup.
func (s *RepositoryTestSuite) createRepository() *config.Repository {
	// Set up default expectations for repository creation
	s.mockFS.EXPECT().MkdirAll(s.tempDir+"/.config/cowpoke", os.FileMode(0o700)).Return(nil).Maybe()
	s.mockFS.EXPECT().ReadFile(s.configPath).Return(nil, os.ErrNotExist).Maybe()

	repo, err := config.NewRepository(s.mockFS, s.configPath, s.logger)
	s.Require().NoError(err)
	s.Require().NotNil(repo)
	return repo
}

func (s *RepositoryTestSuite) TestNewRepository_Success() {
	s.mockFS.EXPECT().MkdirAll(s.tempDir+"/.config/cowpoke", os.FileMode(0o700)).Return(nil)
	s.mockFS.EXPECT().ReadFile(s.configPath).Return(nil, os.ErrNotExist)

	repo, err := config.NewRepository(s.mockFS, s.configPath, s.logger)

	s.Require().NoError(err)
	s.NotNil(repo)
}

func (s *RepositoryTestSuite) TestNewRepository_WithExistingConfig() {
	existingConfig := `servers:
  - name: production
    url: https://rancher.prod.example.com
    username: admin
    authType: local`

	s.mockFS.EXPECT().MkdirAll(s.tempDir+"/.config/cowpoke", os.FileMode(0o700)).Return(nil)
	s.mockFS.EXPECT().ReadFile(s.configPath).Return([]byte(existingConfig), nil)

	// Expect migration save call
	s.mockFS.EXPECT().WriteFile(s.configPath, mock.AnythingOfType("[]uint8"), os.FileMode(0o600)).Return(nil)

	// Expect migration chmod calls for security
	s.mockFS.EXPECT().Chmod(s.configPath, os.FileMode(0o600)).Return(nil)
	s.mockFS.EXPECT().Chmod(s.tempDir+"/.config/cowpoke", os.FileMode(0o700)).Return(nil)
	s.mockFS.EXPECT().Stat(s.tempDir+"/.kube").Return(nil, os.ErrNotExist)

	repo, err := config.NewRepository(s.mockFS, s.configPath, s.logger)
	s.Require().NoError(err)

	servers, err := repo.GetServers(s.ctx)
	s.Require().NoError(err)

	s.Len(servers, 1, "Expected 1 server")
	s.Equal("https://rancher.prod.example.com", servers[0].URL)
}

func (s *RepositoryTestSuite) TestNewRepository_DirectoryCreationError() {
	s.mockFS.EXPECT().MkdirAll(s.tempDir+"/.config/cowpoke", os.FileMode(0o700)).Return(errors.New("permission denied"))

	repo, err := config.NewRepository(s.mockFS, s.configPath, s.logger)

	s.Require().Error(err, "Expected error when directory creation fails")
	s.Nil(repo, "Expected nil repository when error occurs")

	expectedErrorMsg := "failed to create config directory: permission denied"
	s.Equal(expectedErrorMsg, err.Error())
}

func (s *RepositoryTestSuite) TestGetServers_Empty() {
	repo := s.createRepository()

	servers, err := repo.GetServers(s.ctx)
	s.Require().NoError(err)
	s.Empty(servers, "Expected 0 servers")
}

func (s *RepositoryTestSuite) TestAddServer_Success() {
	repo := s.createRepository()

	server := domain.ConfigServer{
		URL:      "https://rancher.prod.example.com",
		Username: "admin",
		AuthType: "local",
	}

	// Expect the WriteFile call for saving config
	s.mockFS.EXPECT().WriteFile(s.configPath, mock.AnythingOfType("[]uint8"), os.FileMode(0o600)).Return(nil)

	err := repo.AddServer(s.ctx, server)
	s.Require().NoError(err)

	servers, err := repo.GetServers(s.ctx)
	s.Require().NoError(err)

	s.Len(servers, 1, "Expected 1 server")
	s.Equal(server.URL, servers[0].URL)
}

// TestRepositoryTestSuite runs the repository test suite.
func TestRepositoryTestSuite(t *testing.T) {
	suite.Run(t, new(RepositoryTestSuite))
}

// Additional non-suite tests for complex scenarios.
func TestRepository_AddServer_DuplicateURL(t *testing.T) {
	ctx := context.Background()
	logger := slog.Default()

	mockFS := mocks.NewMockFileSystemAdapter(t)
	tempDir := t.TempDir()
	configPath := tempDir + "/.config/cowpoke/config.yaml"

	mockFS.EXPECT().MkdirAll(tempDir+"/.config/cowpoke", os.FileMode(0o700)).Return(nil)
	mockFS.EXPECT().ReadFile(configPath).Return(nil, os.ErrNotExist)

	repo, err := config.NewRepository(mockFS, configPath, logger)
	require.NoError(t, err)

	server := domain.ConfigServer{
		URL:      "https://rancher.prod.example.com",
		Username: "admin",
		AuthType: "local",
	}

	// First add should succeed
	mockFS.EXPECT().WriteFile(configPath, mock.AnythingOfType("[]uint8"), os.FileMode(0o600)).Return(nil)
	err = repo.AddServer(ctx, server)
	require.NoError(t, err)

	// Second add should fail with duplicate error
	err = repo.AddServer(ctx, server)
	require.Error(t, err)
	assert.Equal(t, "server https://rancher.prod.example.com already exists in configuration", err.Error())
}

func TestRepository_RemoveServer_Success(t *testing.T) {
	ctx := context.Background()
	logger := slog.Default()

	mockFS := mocks.NewMockFileSystemAdapter(t)
	tempDir := t.TempDir()
	configPath := tempDir + "/.config/cowpoke/config.yaml"

	// Setup with two servers
	existingConfig := `servers:
  - url: https://rancher.prod.example.com
    username: admin
    authType: local
  - url: https://rancher.staging.example.com
    username: admin
    authType: openldap`

	mockFS.EXPECT().MkdirAll(tempDir+"/.config/cowpoke", os.FileMode(0o700)).Return(nil)
	mockFS.EXPECT().ReadFile(configPath).Return([]byte(existingConfig), nil)

	// Expect migration save call
	mockFS.EXPECT().WriteFile(configPath, mock.AnythingOfType("[]uint8"), os.FileMode(0o600)).Return(nil)

	// Expect migration chmod calls for security
	mockFS.EXPECT().Chmod(configPath, os.FileMode(0o600)).Return(nil)
	mockFS.EXPECT().Chmod(tempDir+"/.config/cowpoke", os.FileMode(0o700)).Return(nil)
	mockFS.EXPECT().Stat(tempDir+"/.kube").Return(nil, os.ErrNotExist)

	repo, err := config.NewRepository(mockFS, configPath, logger)
	require.NoError(t, err)

	// Remove staging server
	mockFS.EXPECT().WriteFile(configPath, mock.AnythingOfType("[]uint8"), os.FileMode(0o600)).Return(nil)
	err = repo.RemoveServer(ctx, "https://rancher.staging.example.com")
	require.NoError(t, err)

	servers, err := repo.GetServers(ctx)
	require.NoError(t, err)

	assert.Len(t, servers, 1)
	assert.Equal(t, "https://rancher.prod.example.com", servers[0].URL)
}

func TestRepository_LoadConfig_V1Migration(t *testing.T) {
	ctx := context.Background()
	logger := slog.Default()

	// V1 configuration with old structure
	v1Config := `version: "1.0"
servers:
  - id: "server-1"
    name: "Production Rancher"
    url: "https://rancher.prod.example.com"
    username: "admin"
    authType: "local"
  - id: "server-2"
    name: "Staging Rancher"
    url: "https://rancher.staging.example.com"
    username: "testuser"
    authType: "openldap"`

	mockFS := mocks.NewMockFileSystemAdapter(t)
	tempDir := t.TempDir()
	configPath := tempDir + "/.config/cowpoke/config.yaml"

	mockFS.EXPECT().MkdirAll(tempDir+"/.config/cowpoke", os.FileMode(0o700)).Return(nil)
	mockFS.EXPECT().ReadFile(configPath).Return([]byte(v1Config), nil)
	// Expect save after migration
	mockFS.EXPECT().WriteFile(configPath, mock.AnythingOfType("[]uint8"), os.FileMode(0o600)).Return(nil)

	// Expect migration chmod calls for security
	mockFS.EXPECT().Chmod(configPath, os.FileMode(0o600)).Return(nil)
	mockFS.EXPECT().Chmod(tempDir+"/.config/cowpoke", os.FileMode(0o700)).Return(nil)
	mockFS.EXPECT().Stat(tempDir+"/.kube").Return(nil, os.ErrNotExist)

	repo, err := config.NewRepository(mockFS, configPath, logger)
	require.NoError(t, err)

	servers, err := repo.GetServers(ctx)
	require.NoError(t, err)

	// Verify migrated servers
	want := []domain.ConfigServer{
		{
			URL:      "https://rancher.prod.example.com",
			Username: "admin",
			AuthType: "local",
		},
		{
			URL:      "https://rancher.staging.example.com",
			Username: "testuser",
			AuthType: "openldap",
		},
	}

	assert.Equal(t, want, servers, "Migrated servers should match expected")
}
