package commands_test

import (
	"context"
	"errors"
	"log/slog"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"cowpoke/internal/commands"
	"cowpoke/internal/domain"
	"cowpoke/internal/mocks"
)

func TestListCommand_Execute_Success_WithServers(t *testing.T) {
	ctx := context.Background()
	logger := slog.Default()

	servers := []domain.ConfigServer{
		{
			URL:      "https://rancher.prod.example.com",
			Username: "admin",
			AuthType: "local",
		},
		{
			URL:      "https://rancher.staging.example.com",
			Username: "admin",
			AuthType: "openldap",
		},
	}

	mockConfigRepo := mocks.NewMockConfigRepository(t)
	mockConfigRepo.EXPECT().GetServers(ctx).Return(servers, nil)

	listCommand := commands.NewListCommand(mockConfigRepo, logger)

	result, err := listCommand.Execute(ctx, commands.ListRequest{})
	require.NoError(t, err)
	require.NotNil(t, result)

	// Verify the result structure
	want := &commands.ListResult{
		Servers: servers,
		Count:   2,
	}

	assert.Equal(t, want, result, "ListCommand result should match expected")
}

func TestListCommand_Execute_Success_EmptyList(t *testing.T) {
	ctx := context.Background()
	logger := slog.Default()

	mockConfigRepo := mocks.NewMockConfigRepository(t)
	mockConfigRepo.EXPECT().GetServers(ctx).Return([]domain.ConfigServer{}, nil)

	listCommand := commands.NewListCommand(mockConfigRepo, logger)

	result, err := listCommand.Execute(ctx, commands.ListRequest{})
	require.NoError(t, err)
	require.NotNil(t, result)

	// Verify empty result structure
	want := &commands.ListResult{
		Servers: []domain.ConfigServer{},
		Count:   0,
	}

	assert.Equal(t, want, result, "Empty list result should match expected")
}

func TestListCommand_Execute_RepositoryError(t *testing.T) {
	ctx := context.Background()
	logger := slog.Default()

	mockConfigRepo := mocks.NewMockConfigRepository(t)
	expectedError := errors.New("failed to read configuration file")
	mockConfigRepo.EXPECT().GetServers(ctx).Return(nil, expectedError)

	listCommand := commands.NewListCommand(mockConfigRepo, logger)

	result, err := listCommand.Execute(ctx, commands.ListRequest{})

	require.Error(t, err, "Expected error when repository fails")
	assert.Nil(t, result, "Expected nil result when error occurs")
	assert.Equal(t, "failed to get servers: failed to read configuration file", err.Error())
}
