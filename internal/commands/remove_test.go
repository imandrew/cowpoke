package commands_test

import (
	"context"
	"errors"
	"log/slog"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"cowpoke/internal/commands"
	"cowpoke/internal/mocks"
)

func TestRemoveCommand_Execute_Success(t *testing.T) {
	ctx := context.Background()
	logger := slog.Default()

	mockConfigRepo := mocks.NewMockConfigRepository(t)
	mockConfigRepo.EXPECT().RemoveServer(ctx, "https://rancher.staging.example.com").Return(nil)

	removeCommand := commands.NewRemoveCommand(mockConfigRepo, logger)

	err := removeCommand.Execute(ctx, commands.RemoveRequest{
		ServerURL: "https://rancher.staging.example.com",
	})
	require.NoError(t, err)
}

func TestRemoveCommand_Execute_RepositoryError(t *testing.T) {
	ctx := context.Background()
	logger := slog.Default()

	mockConfigRepo := mocks.NewMockConfigRepository(t)
	expectedError := errors.New("failed to save configuration")
	mockConfigRepo.EXPECT().RemoveServer(ctx, "https://rancher.example.com").Return(expectedError)

	removeCommand := commands.NewRemoveCommand(mockConfigRepo, logger)

	err := removeCommand.Execute(ctx, commands.RemoveRequest{
		ServerURL: "https://rancher.example.com",
	})

	require.Error(t, err, "Expected error when repository fails")
	assert.Equal(t, "failed to remove server: failed to save configuration", err.Error())
}

func TestRemoveCommand_Execute_EmptyURL(t *testing.T) {
	ctx := context.Background()
	logger := slog.Default()

	mockConfigRepo := mocks.NewMockConfigRepository(t)

	removeCommand := commands.NewRemoveCommand(mockConfigRepo, logger)

	err := removeCommand.Execute(ctx, commands.RemoveRequest{
		ServerURL: "",
	})
	require.Error(t, err, "Expected error when neither ServerURL nor ServerID is provided")
	assert.Equal(t, "either ServerURL or ServerID must be specified", err.Error())
}

func TestRemoveCommand_Execute_SuccessByID(t *testing.T) {
	ctx := context.Background()
	logger := slog.Default()

	mockConfigRepo := mocks.NewMockConfigRepository(t)
	mockConfigRepo.EXPECT().RemoveServerByID(ctx, "testid123").Return(nil)

	removeCommand := commands.NewRemoveCommand(mockConfigRepo, logger)

	err := removeCommand.Execute(ctx, commands.RemoveRequest{
		ServerID: "testid123",
	})
	require.NoError(t, err)
}

func TestRemoveCommand_Execute_RepositoryErrorByID(t *testing.T) {
	ctx := context.Background()
	logger := slog.Default()

	mockConfigRepo := mocks.NewMockConfigRepository(t)
	expectedError := errors.New("failed to save configuration")
	mockConfigRepo.EXPECT().RemoveServerByID(ctx, "testid123").Return(expectedError)

	removeCommand := commands.NewRemoveCommand(mockConfigRepo, logger)

	err := removeCommand.Execute(ctx, commands.RemoveRequest{
		ServerID: "testid123",
	})

	require.Error(t, err, "Expected error when repository fails")
	assert.Equal(t, "failed to remove server: failed to save configuration", err.Error())
}
