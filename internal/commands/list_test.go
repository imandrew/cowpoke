package commands

import (
	"context"
	"errors"
	"testing"

	"cowpoke/internal/domain"
	"cowpoke/internal/mocks"
	"cowpoke/internal/testutil"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

// Test helper to create ListCommand with test logger.
func newTestListCommand(repo domain.ConfigRepository) *ListCommand {
	return NewListCommand(repo, testutil.Logger())
}

func TestListCommand_Execute_Success_NoServers(t *testing.T) {
	// Arrange
	mockConfigRepo := mocks.NewMockConfigRepository(t)

	mockConfigRepo.On("GetServers", mock.Anything).Return([]domain.ConfigServer{}, nil)

	cmd := newTestListCommand(mockConfigRepo)
	ctx := context.Background()
	req := ListRequest{}

	// Act
	result, err := cmd.Execute(ctx, req)

	// Assert
	require.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, 0, result.Count)
	assert.Equal(t, []domain.ConfigServer{}, result.Servers)
	mockConfigRepo.AssertExpectations(t)
}

func TestListCommand_Execute_Success_WithServers(t *testing.T) {
	// Arrange
	mockConfigRepo := mocks.NewMockConfigRepository(t)

	expectedServers := []domain.ConfigServer{
		{URL: "https://rancher1.example.com", Username: "admin", AuthType: "local"},
		{URL: "https://rancher2.example.com", Username: "user", AuthType: "ldap"},
		{URL: "https://rancher3.example.com", Username: "testuser", AuthType: "activedirectory"},
	}

	mockConfigRepo.On("GetServers", mock.Anything).Return(expectedServers, nil)

	cmd := newTestListCommand(mockConfigRepo)
	ctx := context.Background()
	req := ListRequest{}

	// Act
	result, err := cmd.Execute(ctx, req)

	// Assert
	require.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, 3, result.Count)
	assert.Equal(t, expectedServers, result.Servers)
	mockConfigRepo.AssertExpectations(t)
}

func TestListCommand_Execute_RepositoryError(t *testing.T) {
	// Arrange
	mockConfigRepo := mocks.NewMockConfigRepository(t)

	expectedErr := errors.New("database connection failed")
	mockConfigRepo.On("GetServers", mock.Anything).Return(nil, expectedErr)

	cmd := newTestListCommand(mockConfigRepo)
	ctx := context.Background()
	req := ListRequest{}

	// Act
	result, err := cmd.Execute(ctx, req)

	// Assert
	require.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "failed to get servers")
	assert.Contains(t, err.Error(), expectedErr.Error())
	mockConfigRepo.AssertExpectations(t)
}

func TestListCommand_Execute_SingleServer(t *testing.T) {
	// Arrange
	mockConfigRepo := mocks.NewMockConfigRepository(t)

	expectedServers := []domain.ConfigServer{
		{URL: "https://rancher.example.com", Username: "admin", AuthType: "local"},
	}

	mockConfigRepo.On("GetServers", mock.Anything).Return(expectedServers, nil)

	cmd := newTestListCommand(mockConfigRepo)
	ctx := context.Background()
	req := ListRequest{}

	// Act
	result, err := cmd.Execute(ctx, req)

	// Assert
	require.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, 1, result.Count)
	assert.Equal(t, expectedServers, result.Servers)
	assert.Equal(t, "https://rancher.example.com", result.Servers[0].URL)
	assert.Equal(t, "admin", result.Servers[0].Username)
	assert.Equal(t, "local", result.Servers[0].AuthType)
	mockConfigRepo.AssertExpectations(t)
}

func TestListCommand_Execute_ServerIDGeneration(t *testing.T) {
	// Test that the result includes servers with their proper IDs
	// This verifies the business logic around ID generation

	// Arrange
	mockConfigRepo := mocks.NewMockConfigRepository(t)

	expectedServers := []domain.ConfigServer{
		{URL: "https://rancher1.example.com", Username: "admin", AuthType: "local"},
		{URL: "https://rancher2.example.com", Username: "user", AuthType: "ldap"},
	}

	mockConfigRepo.On("GetServers", mock.Anything).Return(expectedServers, nil)

	cmd := newTestListCommand(mockConfigRepo)
	ctx := context.Background()
	req := ListRequest{}

	// Act
	result, err := cmd.Execute(ctx, req)

	// Assert
	require.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, 2, result.Count)

	// Verify IDs are generated correctly
	assert.Equal(t, "e737f2fb", result.Servers[0].ID()) // rancher1.example.com
	assert.Equal(t, "31807b28", result.Servers[1].ID()) // rancher2.example.com

	mockConfigRepo.AssertExpectations(t)
}

func TestListCommand_Execute_DifferentAuthTypes(t *testing.T) {
	// Test listing servers with various auth types

	// Arrange
	mockConfigRepo := mocks.NewMockConfigRepository(t)

	expectedServers := []domain.ConfigServer{
		{URL: "https://local.example.com", Username: "admin", AuthType: "local"},
		{URL: "https://ldap.example.com", Username: "user", AuthType: "ldap"},
		{URL: "https://openldap.example.com", Username: "testuser", AuthType: "openldap"},
		{URL: "https://ad.example.com", Username: "domainuser", AuthType: "activedirectory"},
	}

	mockConfigRepo.On("GetServers", mock.Anything).Return(expectedServers, nil)

	cmd := newTestListCommand(mockConfigRepo)
	ctx := context.Background()
	req := ListRequest{}

	// Act
	result, err := cmd.Execute(ctx, req)

	// Assert
	require.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, 4, result.Count)

	// Verify all auth types are preserved
	authTypes := make([]string, len(result.Servers))
	for i, server := range result.Servers {
		authTypes[i] = server.AuthType
	}
	assert.Contains(t, authTypes, "local")
	assert.Contains(t, authTypes, "ldap")
	assert.Contains(t, authTypes, "openldap")
	assert.Contains(t, authTypes, "activedirectory")

	mockConfigRepo.AssertExpectations(t)
}

func TestListCommand_Execute_RequestParametersIgnored(t *testing.T) {
	// The current implementation ignores the ListRequest parameters
	// This test verifies that behavior and documents it

	// Arrange
	mockConfigRepo := mocks.NewMockConfigRepository(t)

	expectedServers := []domain.ConfigServer{
		{URL: "https://rancher.example.com", Username: "admin", AuthType: "local"},
	}

	mockConfigRepo.On("GetServers", mock.Anything).Return(expectedServers, nil)

	cmd := newTestListCommand(mockConfigRepo)
	ctx := context.Background()

	// Test with various request parameters - they should all be ignored
	requests := []ListRequest{
		{}, // empty request
		{OutputFormat: "json"},
		{OutputFormat: "yaml", Verbose: true},
		{Verbose: true},
	}

	for _, req := range requests {
		// Act
		result, err := cmd.Execute(ctx, req)

		// Assert
		require.NoError(t, err)
		assert.NotNil(t, result)
		assert.Equal(t, 1, result.Count)
		assert.Equal(t, expectedServers, result.Servers)
	}

	mockConfigRepo.AssertExpectations(t)
}

func TestListCommand_Execute_NilServersList(t *testing.T) {
	// Test behavior when repository returns nil slice

	// Arrange
	mockConfigRepo := mocks.NewMockConfigRepository(t)

	mockConfigRepo.On("GetServers", mock.Anything).Return(nil, nil)

	cmd := newTestListCommand(mockConfigRepo)
	ctx := context.Background()
	req := ListRequest{}

	// Act
	result, err := cmd.Execute(ctx, req)

	// Assert
	require.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, 0, result.Count)
	assert.Nil(t, result.Servers) // nil slice is preserved
	mockConfigRepo.AssertExpectations(t)
}

func TestNewListCommand(t *testing.T) {
	// Arrange
	mockConfigRepo := mocks.NewMockConfigRepository(t)

	// Act
	cmd := newTestListCommand(mockConfigRepo)

	// Assert
	assert.NotNil(t, cmd)
	assert.Equal(t, mockConfigRepo, cmd.configRepo)
}
