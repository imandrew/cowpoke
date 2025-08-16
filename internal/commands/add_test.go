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

// Test helper to create AddCommand with test logger.
func newTestAddCommand(repo domain.ConfigRepository) *AddCommand {
	return NewAddCommand(repo, testutil.Logger())
}

func TestAddCommand_Execute_Success(t *testing.T) {
	// Arrange
	mockConfigRepo := mocks.NewMockConfigRepository(t)

	expectedServer := domain.ConfigServer{
		URL:      "https://rancher.example.com",
		Username: "admin",
		AuthType: "local",
	}

	mockConfigRepo.On("AddServer", mock.Anything, expectedServer).Return(nil)

	cmd := newTestAddCommand(mockConfigRepo)
	ctx := context.Background()
	req := AddRequest{
		URL:      "https://rancher.example.com",
		Username: "admin",
		AuthType: "local",
	}

	// Act
	err := cmd.Execute(ctx, req)

	// Assert
	require.NoError(t, err)
	mockConfigRepo.AssertExpectations(t)
}

func TestAddCommand_Execute_RepositoryError(t *testing.T) {
	// Arrange
	mockConfigRepo := mocks.NewMockConfigRepository(t)

	expectedServer := domain.ConfigServer{
		URL:      "https://rancher.example.com",
		Username: "admin",
		AuthType: "local",
	}
	expectedErr := errors.New("repository error")

	mockConfigRepo.On("AddServer", mock.Anything, expectedServer).Return(expectedErr)

	cmd := newTestAddCommand(mockConfigRepo)
	ctx := context.Background()
	req := AddRequest{
		URL:      "https://rancher.example.com",
		Username: "admin",
		AuthType: "local",
	}

	// Act
	err := cmd.Execute(ctx, req)

	// Assert
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to add server")
	assert.Contains(t, err.Error(), expectedErr.Error())
	mockConfigRepo.AssertExpectations(t)
}

func TestAddCommand_Execute_DifferentAuthTypes(t *testing.T) {
	tests := []struct {
		name     string
		authType string
	}{
		{
			name:     "local auth type",
			authType: "local",
		},
		{
			name:     "ldap auth type",
			authType: "ldap",
		},
		{
			name:     "openldap auth type",
			authType: "openldap",
		},
		{
			name:     "activedirectory auth type",
			authType: "activedirectory",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Arrange
			mockConfigRepo := mocks.NewMockConfigRepository(t)

			expectedServer := domain.ConfigServer{
				URL:      "https://rancher.example.com",
				Username: "testuser",
				AuthType: tt.authType,
			}

			mockConfigRepo.On("AddServer", mock.Anything, expectedServer).Return(nil)

			cmd := newTestAddCommand(mockConfigRepo)
			ctx := context.Background()
			req := AddRequest{
				URL:      "https://rancher.example.com",
				Username: "testuser",
				AuthType: tt.authType,
			}

			// Act
			err := cmd.Execute(ctx, req)

			// Assert
			require.NoError(t, err)
			mockConfigRepo.AssertExpectations(t)
		})
	}
}

func TestAddCommand_Execute_ServerIDGeneration(t *testing.T) {
	// This test verifies that servers with different URLs get different IDs
	// and the same URLs always get the same ID

	tests := []struct {
		name       string
		url        string
		expectedID string
	}{
		{
			name:       "rancher1.example.com",
			url:        "https://rancher1.example.com",
			expectedID: "e737f2fb", // This is the deterministic ID for rancher1.example.com
		},
		{
			name:       "rancher2.example.com",
			url:        "https://rancher2.example.com",
			expectedID: "31807b28", // This is the deterministic ID for rancher2.example.com
		},
		{
			name:       "same URL should get same ID",
			url:        "https://rancher1.example.com",
			expectedID: "e737f2fb", // Same as first test
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Arrange
			mockConfigRepo := mocks.NewMockConfigRepository(t)

			// Use a matcher to verify the server has the expected ID
			mockConfigRepo.On("AddServer", mock.Anything, mock.MatchedBy(func(server domain.ConfigServer) bool {
				return server.ID() == tt.expectedID && server.URL == tt.url
			})).Return(nil)

			cmd := newTestAddCommand(mockConfigRepo)
			ctx := context.Background()
			req := AddRequest{
				URL:      tt.url,
				Username: "admin",
				AuthType: "local",
			}

			// Act
			err := cmd.Execute(ctx, req)

			// Assert
			require.NoError(t, err)
			mockConfigRepo.AssertExpectations(t)
		})
	}
}

func TestAddCommand_Execute_EmptyFields(t *testing.T) {
	// Test behavior with empty/minimal required fields
	// Note: Field validation is typically done by the CLI layer,
	// but we test the command handles empty values gracefully

	tests := []struct {
		name    string
		request AddRequest
		wantErr bool
	}{
		{
			name: "all fields provided",
			request: AddRequest{
				URL:      "https://rancher.example.com",
				Username: "admin",
				AuthType: "local",
			},
			wantErr: false,
		},
		{
			name: "empty URL",
			request: AddRequest{
				URL:      "",
				Username: "admin",
				AuthType: "local",
			},
			wantErr: false, // Command doesn't validate, CLI layer does
		},
		{
			name: "empty username",
			request: AddRequest{
				URL:      "https://rancher.example.com",
				Username: "",
				AuthType: "local",
			},
			wantErr: false, // Command doesn't validate, CLI layer does
		},
		{
			name: "empty authtype",
			request: AddRequest{
				URL:      "https://rancher.example.com",
				Username: "admin",
				AuthType: "",
			},
			wantErr: false, // Command doesn't validate, CLI layer does
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Arrange
			mockConfigRepo := mocks.NewMockConfigRepository(t)

			expectedServer := domain.ConfigServer{
				URL:      tt.request.URL,
				Username: tt.request.Username,
				AuthType: tt.request.AuthType,
			}

			if !tt.wantErr {
				mockConfigRepo.On("AddServer", mock.Anything, expectedServer).Return(nil)
			}

			cmd := newTestAddCommand(mockConfigRepo)
			ctx := context.Background()

			// Act
			err := cmd.Execute(ctx, tt.request)

			// Assert
			if tt.wantErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}
			mockConfigRepo.AssertExpectations(t)
		})
	}
}

func TestNewAddCommand(t *testing.T) {
	// Arrange
	mockConfigRepo := mocks.NewMockConfigRepository(t)

	// Act
	cmd := newTestAddCommand(mockConfigRepo)

	// Assert
	assert.NotNil(t, cmd)
	assert.Equal(t, mockConfigRepo, cmd.configRepo)
}
