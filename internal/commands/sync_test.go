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

// Test helper to create SyncCommand with test logger.
func newTestSyncCommand(
	repo domain.ConfigRepository,
	provider domain.ConfigProvider,
	reader domain.PasswordReader,
) *SyncCommand {
	return NewSyncCommand(repo, provider, reader, testutil.Logger())
}

func TestSyncCommand_Execute_NoServersConfigured(t *testing.T) {
	// Arrange
	mockConfigRepo := mocks.NewMockConfigRepository(t)
	mockConfigProvider := mocks.NewMockConfigProvider(t)
	mockPasswordReader := mocks.NewMockPasswordReader(t)
	mockSyncOrchestrator := mocks.NewMockSyncOrchestrator(t)
	mockKubeconfigHandler := mocks.NewMockKubeconfigHandler(t)

	mockConfigRepo.On("GetServers", mock.Anything).Return([]domain.ConfigServer{}, nil)

	cmd := newTestSyncCommand(mockConfigRepo, mockConfigProvider, mockPasswordReader)
	ctx := context.Background()
	req := SyncRequest{}

	// Act
	err := cmd.Execute(ctx, req, mockSyncOrchestrator, mockKubeconfigHandler)

	// Assert
	require.NoError(t, err)
	mockConfigRepo.AssertExpectations(t)
}

func TestSyncCommand_Execute_GetServersFails(t *testing.T) {
	// Arrange
	mockConfigRepo := mocks.NewMockConfigRepository(t)
	mockConfigProvider := mocks.NewMockConfigProvider(t)
	mockPasswordReader := mocks.NewMockPasswordReader(t)
	mockSyncOrchestrator := mocks.NewMockSyncOrchestrator(t)
	mockKubeconfigHandler := mocks.NewMockKubeconfigHandler(t)

	expectedErr := errors.New("config read error")
	mockConfigRepo.On("GetServers", mock.Anything).Return(nil, expectedErr)

	cmd := newTestSyncCommand(mockConfigRepo, mockConfigProvider, mockPasswordReader)
	ctx := context.Background()
	req := SyncRequest{}

	// Act
	err := cmd.Execute(ctx, req, mockSyncOrchestrator, mockKubeconfigHandler)

	// Assert
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to get servers")
	assert.Contains(t, err.Error(), expectedErr.Error())
	mockConfigRepo.AssertExpectations(t)
}

func TestSyncCommand_Execute_PasswordCollectionFails(t *testing.T) {
	// Arrange
	mockConfigRepo := mocks.NewMockConfigRepository(t)
	mockConfigProvider := mocks.NewMockConfigProvider(t)
	mockPasswordReader := mocks.NewMockPasswordReader(t)
	mockSyncOrchestrator := mocks.NewMockSyncOrchestrator(t)
	mockKubeconfigHandler := mocks.NewMockKubeconfigHandler(t)

	servers := []domain.ConfigServer{
		{URL: "https://rancher.example.com", Username: "admin", AuthType: "local"},
	}
	expectedErr := errors.New("password read error")

	mockConfigRepo.On("GetServers", mock.Anything).Return(servers, nil)
	mockPasswordReader.On("ReadPassword", mock.Anything, mock.AnythingOfType("string")).Return("", expectedErr)

	cmd := newTestSyncCommand(mockConfigRepo, mockConfigProvider, mockPasswordReader)
	ctx := context.Background()
	req := SyncRequest{}

	// Act
	err := cmd.Execute(ctx, req, mockSyncOrchestrator, mockKubeconfigHandler)

	// Assert
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to collect passwords")
	assert.Contains(t, err.Error(), expectedErr.Error())
	mockConfigRepo.AssertExpectations(t)
	mockPasswordReader.AssertExpectations(t)
}

func TestSyncCommand_Execute_InvalidExcludePattern(t *testing.T) {
	// Arrange
	mockConfigRepo := mocks.NewMockConfigRepository(t)
	mockConfigProvider := mocks.NewMockConfigProvider(t)
	mockPasswordReader := mocks.NewMockPasswordReader(t)
	mockSyncOrchestrator := mocks.NewMockSyncOrchestrator(t)
	mockKubeconfigHandler := mocks.NewMockKubeconfigHandler(t)

	servers := []domain.ConfigServer{
		{URL: "https://rancher.example.com", Username: "admin", AuthType: "local"},
	}

	mockConfigRepo.On("GetServers", mock.Anything).Return(servers, nil)
	mockPasswordReader.On("ReadPassword", mock.Anything, mock.AnythingOfType("string")).Return("password123", nil)

	cmd := newTestSyncCommand(mockConfigRepo, mockConfigProvider, mockPasswordReader)
	ctx := context.Background()
	req := SyncRequest{
		ExcludePatterns: []string{"[invalid-regex"},
	}

	// Act
	err := cmd.Execute(ctx, req, mockSyncOrchestrator, mockKubeconfigHandler)

	// Assert
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to create exclude filter")
	mockConfigRepo.AssertExpectations(t)
	mockPasswordReader.AssertExpectations(t)
}

func TestSyncCommand_Execute_SyncFails(t *testing.T) {
	// Arrange
	mockConfigRepo := mocks.NewMockConfigRepository(t)
	mockConfigProvider := mocks.NewMockConfigProvider(t)
	mockPasswordReader := mocks.NewMockPasswordReader(t)
	mockSyncOrchestrator := mocks.NewMockSyncOrchestrator(t)
	mockKubeconfigHandler := mocks.NewMockKubeconfigHandler(t)

	servers := []domain.ConfigServer{
		{URL: "https://rancher.example.com", Username: "admin", AuthType: "local"},
	}
	expectedErr := errors.New("sync orchestrator error")

	mockConfigRepo.On("GetServers", mock.Anything).Return(servers, nil)
	mockPasswordReader.On("ReadPassword", mock.Anything, mock.AnythingOfType("string")).Return("password123", nil)
	mockSyncOrchestrator.On("SyncServers", mock.Anything, mock.Anything, mock.Anything).
		Return(nil, expectedErr)

	cmd := newTestSyncCommand(mockConfigRepo, mockConfigProvider, mockPasswordReader)
	ctx := context.Background()
	req := SyncRequest{}

	// Act
	err := cmd.Execute(ctx, req, mockSyncOrchestrator, mockKubeconfigHandler)

	// Assert
	require.Error(t, err)
	assert.Contains(t, err.Error(), "concurrent sync failed")
	assert.Contains(t, err.Error(), expectedErr.Error())
	mockConfigRepo.AssertExpectations(t)
	mockPasswordReader.AssertExpectations(t)
	mockSyncOrchestrator.AssertExpectations(t)
}

func TestSyncCommand_Execute_NoKubeconfigsDownloaded(t *testing.T) {
	// Arrange
	mockConfigRepo := mocks.NewMockConfigRepository(t)
	mockConfigProvider := mocks.NewMockConfigProvider(t)
	mockPasswordReader := mocks.NewMockPasswordReader(t)
	mockSyncOrchestrator := mocks.NewMockSyncOrchestrator(t)
	mockKubeconfigHandler := mocks.NewMockKubeconfigHandler(t)

	servers := []domain.ConfigServer{
		{URL: "https://rancher.example.com", Username: "admin", AuthType: "local"},
	}

	mockConfigRepo.On("GetServers", mock.Anything).Return(servers, nil)
	mockPasswordReader.On("ReadPassword", mock.Anything, mock.AnythingOfType("string")).Return("password123", nil)
	mockSyncOrchestrator.On("SyncServers", mock.Anything, mock.Anything, mock.Anything).
		Return(&domain.SyncResult{
			KubeconfigPaths:    []string{},
			TotalClustersFound: 0,
		}, nil)

	cmd := newTestSyncCommand(mockConfigRepo, mockConfigProvider, mockPasswordReader)
	ctx := context.Background()
	req := SyncRequest{}

	// Act
	err := cmd.Execute(ctx, req, mockSyncOrchestrator, mockKubeconfigHandler)

	// Assert
	require.Error(t, err)
	assert.Contains(t, err.Error(), "no kubeconfigs downloaded successfully")
	mockConfigRepo.AssertExpectations(t)
	mockPasswordReader.AssertExpectations(t)
	mockSyncOrchestrator.AssertExpectations(t)
}

func TestSyncCommand_Execute_DefaultOutputPath(t *testing.T) {
	// Arrange
	mockConfigRepo := mocks.NewMockConfigRepository(t)
	mockConfigProvider := mocks.NewMockConfigProvider(t)
	mockPasswordReader := mocks.NewMockPasswordReader(t)
	mockSyncOrchestrator := mocks.NewMockSyncOrchestrator(t)
	mockKubeconfigHandler := mocks.NewMockKubeconfigHandler(t)

	servers := []domain.ConfigServer{
		{URL: "https://rancher.example.com", Username: "admin", AuthType: "local"},
	}
	kubeconfigPaths := []string{"/tmp/cluster1.yaml", "/tmp/cluster2.yaml"}
	defaultPath := "/home/user/.kube/config"

	mockConfigRepo.On("GetServers", mock.Anything).Return(servers, nil)
	mockPasswordReader.On("ReadPassword", mock.Anything, mock.AnythingOfType("string")).Return("password123", nil)
	mockSyncOrchestrator.On("SyncServers", mock.Anything, mock.Anything, mock.Anything).
		Return(&domain.SyncResult{
			KubeconfigPaths:    kubeconfigPaths,
			TotalClustersFound: 2,
		}, nil)
	mockConfigProvider.On("GetDefaultKubeconfigPath").Return(defaultPath, nil)
	mockKubeconfigHandler.On("MergeKubeconfigs", mock.Anything, kubeconfigPaths, defaultPath, mock.AnythingOfType("*filter.NoOpFilter")).
		Return(nil)

	cmd := newTestSyncCommand(mockConfigRepo, mockConfigProvider, mockPasswordReader)
	ctx := context.Background()
	req := SyncRequest{} // No output specified, should use default

	// Act
	err := cmd.Execute(ctx, req, mockSyncOrchestrator, mockKubeconfigHandler)

	// Assert
	require.NoError(t, err)
	mockConfigRepo.AssertExpectations(t)
	mockPasswordReader.AssertExpectations(t)
	mockSyncOrchestrator.AssertExpectations(t)
	mockConfigProvider.AssertExpectations(t)
	mockKubeconfigHandler.AssertExpectations(t)
}

func TestSyncCommand_Execute_CustomOutputPath(t *testing.T) {
	// Arrange
	mockConfigRepo := mocks.NewMockConfigRepository(t)
	mockConfigProvider := mocks.NewMockConfigProvider(t)
	mockPasswordReader := mocks.NewMockPasswordReader(t)
	mockSyncOrchestrator := mocks.NewMockSyncOrchestrator(t)
	mockKubeconfigHandler := mocks.NewMockKubeconfigHandler(t)

	servers := []domain.ConfigServer{
		{URL: "https://rancher.example.com", Username: "admin", AuthType: "local"},
	}
	kubeconfigPaths := []string{"/tmp/cluster1.yaml", "/tmp/cluster2.yaml"}
	customPath := "/custom/path/kubeconfig"

	mockConfigRepo.On("GetServers", mock.Anything).Return(servers, nil)
	mockPasswordReader.On("ReadPassword", mock.Anything, mock.AnythingOfType("string")).Return("password123", nil)
	mockSyncOrchestrator.On("SyncServers", mock.Anything, mock.Anything, mock.Anything).
		Return(&domain.SyncResult{
			KubeconfigPaths:    kubeconfigPaths,
			TotalClustersFound: 2,
		}, nil)
	mockKubeconfigHandler.On("MergeKubeconfigs", mock.Anything, kubeconfigPaths, customPath, mock.AnythingOfType("*filter.NoOpFilter")).
		Return(nil)

	cmd := newTestSyncCommand(mockConfigRepo, mockConfigProvider, mockPasswordReader)
	ctx := context.Background()
	req := SyncRequest{
		Output: customPath,
	}

	// Act
	err := cmd.Execute(ctx, req, mockSyncOrchestrator, mockKubeconfigHandler)

	// Assert
	require.NoError(t, err)
	mockConfigRepo.AssertExpectations(t)
	mockPasswordReader.AssertExpectations(t)
	mockSyncOrchestrator.AssertExpectations(t)
	mockKubeconfigHandler.AssertExpectations(t)
}

func TestSyncCommand_Execute_WithCleanupTempFiles(t *testing.T) {
	// Arrange
	mockConfigRepo := mocks.NewMockConfigRepository(t)
	mockConfigProvider := mocks.NewMockConfigProvider(t)
	mockPasswordReader := mocks.NewMockPasswordReader(t)
	mockSyncOrchestrator := mocks.NewMockSyncOrchestrator(t)
	mockKubeconfigHandler := mocks.NewMockKubeconfigHandler(t)

	servers := []domain.ConfigServer{
		{URL: "https://rancher.example.com", Username: "admin", AuthType: "local"},
	}
	kubeconfigPaths := []string{"/tmp/cluster1.yaml", "/tmp/cluster2.yaml"}
	customPath := "/custom/path/kubeconfig"

	mockConfigRepo.On("GetServers", mock.Anything).Return(servers, nil)
	mockPasswordReader.On("ReadPassword", mock.Anything, mock.AnythingOfType("string")).Return("password123", nil)
	mockSyncOrchestrator.On("SyncServers", mock.Anything, mock.Anything, mock.Anything).
		Return(&domain.SyncResult{
			KubeconfigPaths:    kubeconfigPaths,
			TotalClustersFound: 2,
		}, nil)
	mockKubeconfigHandler.On("MergeKubeconfigs", mock.Anything, kubeconfigPaths, customPath, mock.AnythingOfType("*filter.NoOpFilter")).
		Return(nil)
	mockKubeconfigHandler.On("CleanupTempFiles", mock.Anything, kubeconfigPaths).Return(nil)

	cmd := newTestSyncCommand(mockConfigRepo, mockConfigProvider, mockPasswordReader)
	ctx := context.Background()
	req := SyncRequest{
		Output:           customPath,
		CleanupTempFiles: true,
	}

	// Act
	err := cmd.Execute(ctx, req, mockSyncOrchestrator, mockKubeconfigHandler)

	// Assert
	require.NoError(t, err)
	mockConfigRepo.AssertExpectations(t)
	mockPasswordReader.AssertExpectations(t)
	mockSyncOrchestrator.AssertExpectations(t)
	mockKubeconfigHandler.AssertExpectations(t)
}

func TestSyncCommand_Execute_WithExcludePatterns(t *testing.T) {
	// Arrange
	mockConfigRepo := mocks.NewMockConfigRepository(t)
	mockConfigProvider := mocks.NewMockConfigProvider(t)
	mockPasswordReader := mocks.NewMockPasswordReader(t)
	mockSyncOrchestrator := mocks.NewMockSyncOrchestrator(t)
	mockKubeconfigHandler := mocks.NewMockKubeconfigHandler(t)

	servers := []domain.ConfigServer{
		{URL: "https://rancher.example.com", Username: "admin", AuthType: "local"},
	}
	kubeconfigPaths := []string{"/tmp/cluster1.yaml"}
	excludePatterns := []string{"^test-.*", ".*-staging$"}

	mockConfigRepo.On("GetServers", mock.Anything).Return(servers, nil)
	mockPasswordReader.On("ReadPassword", mock.Anything, mock.AnythingOfType("string")).Return("password123", nil)

	// Orchestrator now downloads ALL kubeconfigs without filtering
	mockSyncOrchestrator.On("SyncServers", mock.Anything, mock.Anything, mock.Anything).
		Return(&domain.SyncResult{
			KubeconfigPaths:    kubeconfigPaths,
			TotalClustersFound: 3,
		}, nil)

	mockConfigProvider.On("GetDefaultKubeconfigPath").Return("/home/user/.kube/config", nil)
	// Filter is now passed to MergeKubeconfigs instead
	mockKubeconfigHandler.On("MergeKubeconfigs", mock.Anything, kubeconfigPaths, "/home/user/.kube/config", mock.MatchedBy(func(filter domain.ClusterFilter) bool {
		// Test that it's an actual exclude filter, not NoOp
		return filter.ShouldExclude("test-cluster") && filter.ShouldExclude("prod-staging") &&
			!filter.ShouldExclude("production")
	})).
		Return(nil)

	cmd := newTestSyncCommand(mockConfigRepo, mockConfigProvider, mockPasswordReader)
	ctx := context.Background()
	req := SyncRequest{
		ExcludePatterns: excludePatterns,
	}

	// Act
	err := cmd.Execute(ctx, req, mockSyncOrchestrator, mockKubeconfigHandler)

	// Assert
	require.NoError(t, err)
	mockConfigRepo.AssertExpectations(t)
	mockPasswordReader.AssertExpectations(t)
	mockSyncOrchestrator.AssertExpectations(t)
	mockKubeconfigHandler.AssertExpectations(t)
}

func TestSyncCommand_collectPasswords(t *testing.T) {
	tests := []struct {
		name    string
		servers []domain.ConfigServer
		setup   func(*mocks.MockPasswordReader)
		want    map[string]string
		wantErr bool
	}{
		{
			name: "single server",
			servers: []domain.ConfigServer{
				{URL: "https://rancher1.example.com", Username: "admin", AuthType: "local"},
			},
			setup: func(mockReader *mocks.MockPasswordReader) {
				mockReader.On("ReadPassword", mock.Anything, "Password for https://rancher1.example.com: ").
					Return("password1", nil)
			},
			want: map[string]string{
				"e737f2fb": "password1", // This is the ID() for rancher1.example.com
			},
			wantErr: false,
		},
		{
			name: "multiple servers",
			servers: []domain.ConfigServer{
				{URL: "https://rancher1.example.com", Username: "admin", AuthType: "local"},
				{URL: "https://rancher2.example.com", Username: "admin", AuthType: "local"},
			},
			setup: func(mockReader *mocks.MockPasswordReader) {
				mockReader.On("ReadPassword", mock.Anything, "Password for https://rancher1.example.com: ").
					Return("password1", nil)
				mockReader.On("ReadPassword", mock.Anything, "Password for https://rancher2.example.com: ").
					Return("password2", nil)
			},
			want: map[string]string{
				"e737f2fb": "password1", // rancher1.example.com
				"31807b28": "password2", // rancher2.example.com
			},
			wantErr: false,
		},
		{
			name: "password read error",
			servers: []domain.ConfigServer{
				{URL: "https://rancher1.example.com", Username: "admin", AuthType: "local"},
			},
			setup: func(mockReader *mocks.MockPasswordReader) {
				mockReader.On("ReadPassword", mock.Anything, "Password for https://rancher1.example.com: ").
					Return("", errors.New("read error"))
			},
			want:    nil,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Arrange
			mockPasswordReader := mocks.NewMockPasswordReader(t)
			tt.setup(mockPasswordReader)

			cmd := &SyncCommand{
				passwordReader: mockPasswordReader,
				logger:         testutil.Logger(),
			}
			ctx := context.Background()

			// Act
			got, err := cmd.collectPasswords(ctx, tt.servers)

			// Assert
			if tt.wantErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.want, got)
			}
			mockPasswordReader.AssertExpectations(t)
		})
	}
}
