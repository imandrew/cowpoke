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

// Test helper to create RemoveCommand with test logger.
func newTestRemoveCommand(repo domain.ConfigRepository) *RemoveCommand {
	return NewRemoveCommand(repo, testutil.Logger())
}

func TestRemoveCommand_Execute_ByURL_Success(t *testing.T) {
	// Arrange
	mockConfigRepo := mocks.NewMockConfigRepository(t)

	serverURL := "https://rancher.example.com"
	mockConfigRepo.On("RemoveServer", mock.Anything, serverURL).Return(nil)

	cmd := newTestRemoveCommand(mockConfigRepo)
	ctx := context.Background()
	req := RemoveRequest{
		ServerURL: serverURL,
	}

	// Act
	err := cmd.Execute(ctx, req)

	// Assert
	require.NoError(t, err)
	mockConfigRepo.AssertExpectations(t)
}

func TestRemoveCommand_Execute_ByID_Success(t *testing.T) {
	// Arrange
	mockConfigRepo := mocks.NewMockConfigRepository(t)

	serverID := "abc123"
	mockConfigRepo.On("RemoveServerByID", mock.Anything, serverID).Return(nil)

	cmd := newTestRemoveCommand(mockConfigRepo)
	ctx := context.Background()
	req := RemoveRequest{
		ServerID: serverID,
	}

	// Act
	err := cmd.Execute(ctx, req)

	// Assert
	require.NoError(t, err)
	mockConfigRepo.AssertExpectations(t)
}

func TestRemoveCommand_Execute_ByURL_RepositoryError(t *testing.T) {
	// Arrange
	mockConfigRepo := mocks.NewMockConfigRepository(t)

	serverURL := "https://rancher.example.com"
	expectedErr := errors.New("server not found")
	mockConfigRepo.On("RemoveServer", mock.Anything, serverURL).Return(expectedErr)

	cmd := newTestRemoveCommand(mockConfigRepo)
	ctx := context.Background()
	req := RemoveRequest{
		ServerURL: serverURL,
	}

	// Act
	err := cmd.Execute(ctx, req)

	// Assert
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to remove server")
	assert.Contains(t, err.Error(), expectedErr.Error())
	mockConfigRepo.AssertExpectations(t)
}

func TestRemoveCommand_Execute_ByID_RepositoryError(t *testing.T) {
	// Arrange
	mockConfigRepo := mocks.NewMockConfigRepository(t)

	serverID := "abc123"
	expectedErr := errors.New("server not found")
	mockConfigRepo.On("RemoveServerByID", mock.Anything, serverID).Return(expectedErr)

	cmd := newTestRemoveCommand(mockConfigRepo)
	ctx := context.Background()
	req := RemoveRequest{
		ServerID: serverID,
	}

	// Act
	err := cmd.Execute(ctx, req)

	// Assert
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to remove server")
	assert.Contains(t, err.Error(), expectedErr.Error())
	mockConfigRepo.AssertExpectations(t)
}

func TestRemoveCommand_Execute_NeitherURLNorID(t *testing.T) {
	// Arrange
	mockConfigRepo := mocks.NewMockConfigRepository(t)

	cmd := newTestRemoveCommand(mockConfigRepo)
	ctx := context.Background()
	req := RemoveRequest{
		// Both ServerURL and ServerID are empty
	}

	// Act
	err := cmd.Execute(ctx, req)

	// Assert
	require.Error(t, err)
	assert.Contains(t, err.Error(), "either ServerURL or ServerID must be specified")
	// No repository calls should be made
	mockConfigRepo.AssertExpectations(t)
}

func TestRemoveCommand_Execute_BothURLAndID(t *testing.T) {
	// Test the business logic: if both are provided, URL takes precedence
	// This tests the switch statement logic where URL case is checked first

	// Arrange
	mockConfigRepo := mocks.NewMockConfigRepository(t)

	serverURL := "https://rancher.example.com"
	serverID := "abc123"

	// Should only call RemoveServer (by URL), not RemoveServerByID
	mockConfigRepo.On("RemoveServer", mock.Anything, serverURL).Return(nil)

	cmd := newTestRemoveCommand(mockConfigRepo)
	ctx := context.Background()
	req := RemoveRequest{
		ServerURL: serverURL,
		ServerID:  serverID, // This should be ignored
	}

	// Act
	err := cmd.Execute(ctx, req)

	// Assert
	require.NoError(t, err)
	mockConfigRepo.AssertExpectations(t)
}

func TestRemoveCommand_Execute_PrecedenceLogic(t *testing.T) {
	// Test various combinations to verify the precedence logic

	tests := []struct {
		name           string
		request        RemoveRequest
		expectedMethod string // "url", "id", or "error"
		expectedValue  string
	}{
		{
			name: "URL only",
			request: RemoveRequest{
				ServerURL: "https://rancher1.example.com",
			},
			expectedMethod: "url",
			expectedValue:  "https://rancher1.example.com",
		},
		{
			name: "ID only",
			request: RemoveRequest{
				ServerID: "abc123",
			},
			expectedMethod: "id",
			expectedValue:  "abc123",
		},
		{
			name: "both provided - URL takes precedence",
			request: RemoveRequest{
				ServerURL: "https://rancher2.example.com",
				ServerID:  "def456",
			},
			expectedMethod: "url",
			expectedValue:  "https://rancher2.example.com",
		},
		{
			name:           "neither provided",
			request:        RemoveRequest{},
			expectedMethod: "error",
		},
		{
			name: "empty strings",
			request: RemoveRequest{
				ServerURL: "",
				ServerID:  "",
			},
			expectedMethod: "error",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Arrange
			mockConfigRepo := mocks.NewMockConfigRepository(t)

			switch tt.expectedMethod {
			case "url":
				mockConfigRepo.On("RemoveServer", mock.Anything, tt.expectedValue).Return(nil)
			case "id":
				mockConfigRepo.On("RemoveServerByID", mock.Anything, tt.expectedValue).Return(nil)
			case "error":
				// No repository calls expected
			}

			cmd := newTestRemoveCommand(mockConfigRepo)
			ctx := context.Background()

			// Act
			err := cmd.Execute(ctx, tt.request)

			// Assert
			if tt.expectedMethod == "error" {
				require.Error(t, err)
				assert.Contains(t, err.Error(), "either ServerURL or ServerID must be specified")
			} else {
				require.NoError(t, err)
			}
			mockConfigRepo.AssertExpectations(t)
		})
	}
}

func TestRemoveCommand_Execute_EmptyStringHandling(t *testing.T) {
	// Test that empty strings are treated as not provided

	tests := []struct {
		name        string
		request     RemoveRequest
		shouldError bool
	}{
		{
			name: "whitespace in URL is still considered provided",
			request: RemoveRequest{
				ServerURL: " ",
			},
			shouldError: false,
		},
		{
			name: "whitespace in ID is still considered provided",
			request: RemoveRequest{
				ServerID: " ",
			},
			shouldError: false,
		},
		{
			name: "empty URL and whitespace ID",
			request: RemoveRequest{
				ServerURL: "",
				ServerID:  " ",
			},
			shouldError: false, // ID with whitespace is considered provided
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Arrange
			mockConfigRepo := mocks.NewMockConfigRepository(t)

			if !tt.shouldError {
				if tt.request.ServerURL != "" {
					mockConfigRepo.On("RemoveServer", mock.Anything, tt.request.ServerURL).Return(nil)
				} else {
					mockConfigRepo.On("RemoveServerByID", mock.Anything, tt.request.ServerID).Return(nil)
				}
			}

			cmd := newTestRemoveCommand(mockConfigRepo)
			ctx := context.Background()

			// Act
			err := cmd.Execute(ctx, tt.request)

			// Assert
			if tt.shouldError {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}
			mockConfigRepo.AssertExpectations(t)
		})
	}
}

func TestNewRemoveCommand(t *testing.T) {
	// Arrange
	mockConfigRepo := mocks.NewMockConfigRepository(t)

	// Act
	cmd := newTestRemoveCommand(mockConfigRepo)

	// Assert
	assert.NotNil(t, cmd)
	assert.Equal(t, mockConfigRepo, cmd.configRepo)
}
