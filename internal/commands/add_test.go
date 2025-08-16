package commands_test

import (
	"context"
	"errors"
	"log/slog"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"cowpoke/internal/commands"
	"cowpoke/internal/domain"
	"cowpoke/internal/mocks"
)

func TestAddCommand_Execute_Success(t *testing.T) {
	ctx := context.Background()
	logger := slog.Default()

	mockConfigRepo := mocks.NewMockConfigRepository(t)

	// Set expectations: AddServer should be called once and return no error
	mockConfigRepo.EXPECT().AddServer(ctx, mock.MatchedBy(func(server domain.ConfigServer) bool {
		return server.URL == "https://rancher.example.com" &&
			server.Username == "admin" &&
			server.AuthType == "local" &&
			server.ID() != "" // Verify ID is generated
	})).Return(nil)

	addCommand := commands.NewAddCommand(mockConfigRepo, logger)

	err := addCommand.Execute(ctx, commands.AddRequest{
		URL:      "https://rancher.example.com",
		Username: "admin",
		AuthType: "local",
	})
	require.NoError(t, err)
}

func TestAddCommand_Execute_RepositoryError(t *testing.T) {
	ctx := context.Background()
	logger := slog.Default()

	mockConfigRepo := mocks.NewMockConfigRepository(t)

	// Set expectation: AddServer should be called and return an error
	expectedError := errors.New("failed to save configuration")
	mockConfigRepo.EXPECT().AddServer(ctx, mock.AnythingOfType("domain.ConfigServer")).Return(expectedError)

	addCommand := commands.NewAddCommand(mockConfigRepo, logger)

	err := addCommand.Execute(ctx, commands.AddRequest{
		URL:      "https://rancher.example.com",
		Username: "admin",
		AuthType: "local",
	})

	require.Error(t, err, "Expected error when repository fails")
	assert.Equal(t, "failed to add server: failed to save configuration", err.Error())
}
