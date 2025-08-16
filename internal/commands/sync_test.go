package commands_test

import (
	"context"
	"errors"
	"log/slog"
	"testing"

	"github.com/stretchr/testify/require"

	"cowpoke/internal/commands"
	"cowpoke/internal/domain"
	"cowpoke/internal/mocks"
)

func TestSyncCommand_Execute_NoServers(t *testing.T) {
	ctx := context.Background()
	logger := slog.Default()

	// Setup mocks with expectations for empty server list
	mockConfigRepo := mocks.NewMockConfigRepository(t)
	mockConfigProvider := mocks.NewMockConfigProvider(t)
	mockPasswordReader := mocks.NewMockPasswordReader(t)

	// Create mock RancherServices
	mockKubeconfigSyncer := mocks.NewMockKubeconfigSyncer(t)
	rancherServices := domain.RancherServices{
		KubeconfigSyncer: mockKubeconfigSyncer,
	}

	// Set expectations for no servers case
	mockConfigRepo.EXPECT().GetServers(ctx).Return([]domain.ConfigServer{}, nil)

	syncCommand := commands.NewSyncCommand(
		mockConfigRepo,
		mockConfigProvider,
		mockPasswordReader,
		logger,
	)

	err := syncCommand.Execute(ctx, commands.SyncRequest{
		Output: "/tmp/config",
	}, rancherServices)

	require.NoError(t, err)
}

func TestSyncCommand_Execute_SuccessfulSync(t *testing.T) {
	ctx := context.Background()
	logger := slog.Default()

	// Setup mocks
	mockConfigRepo := mocks.NewMockConfigRepository(t)
	mockConfigProvider := mocks.NewMockConfigProvider(t)
	mockPasswordReader := mocks.NewMockPasswordReader(t)
	mockKubeconfigSyncer := mocks.NewMockKubeconfigSyncer(t)
	mockKubeconfigMerger := mocks.NewMockKubeconfigMerger(t)

	// Create test servers
	servers := []domain.ConfigServer{
		{URL: "https://rancher1.example.com", Username: "admin", AuthType: "local"},
		{URL: "https://rancher2.example.com", Username: "admin", AuthType: "local"},
	}

	// Create mock RancherServices
	rancherServices := domain.RancherServices{
		KubeconfigSyncer: mockKubeconfigSyncer,
		KubeconfigMerger: mockKubeconfigMerger,
	}

	// Set expectations for successful sync
	mockConfigRepo.EXPECT().GetServers(ctx).Return(servers, nil)
	mockPasswordReader.EXPECT().ReadPassword(ctx, "Password for https://rancher1.example.com: ").Return("pass1", nil)
	mockPasswordReader.EXPECT().ReadPassword(ctx, "Password for https://rancher2.example.com: ").Return("pass2", nil)

	expectedPaths := []string{"/tmp/kubeconfig1", "/tmp/kubeconfig2"}
	expectedPasswords := map[string]string{
		servers[0].ID(): "pass1",
		servers[1].ID(): "pass2",
	}
	mockKubeconfigSyncer.EXPECT().SyncAllServers(ctx, servers, expectedPasswords).Return(expectedPaths, nil)
	mockKubeconfigMerger.EXPECT().MergeKubeconfigs(ctx, expectedPaths, "/custom/output").Return(nil)

	syncCommand := commands.NewSyncCommand(
		mockConfigRepo,
		mockConfigProvider,
		mockPasswordReader,
		logger,
	)

	err := syncCommand.Execute(ctx, commands.SyncRequest{
		Output: "/custom/output",
	}, rancherServices)

	require.NoError(t, err)
}

func TestSyncCommand_Execute_PasswordCollectionFails(t *testing.T) {
	ctx := context.Background()
	logger := slog.Default()

	// Setup mocks
	mockConfigRepo := mocks.NewMockConfigRepository(t)
	mockConfigProvider := mocks.NewMockConfigProvider(t)
	mockPasswordReader := mocks.NewMockPasswordReader(t)
	mockKubeconfigSyncer := mocks.NewMockKubeconfigSyncer(t)

	// Create test servers
	servers := []domain.ConfigServer{
		{URL: "https://rancher1.example.com", Username: "admin", AuthType: "local"},
	}

	rancherServices := domain.RancherServices{
		KubeconfigSyncer: mockKubeconfigSyncer,
	}

	// Set expectations for password failure
	mockConfigRepo.EXPECT().GetServers(ctx).Return(servers, nil)
	mockPasswordReader.EXPECT().
		ReadPassword(ctx, "Password for https://rancher1.example.com: ").
		Return("", errors.New("password read failed"))

	syncCommand := commands.NewSyncCommand(
		mockConfigRepo,
		mockConfigProvider,
		mockPasswordReader,
		logger,
	)

	err := syncCommand.Execute(ctx, commands.SyncRequest{
		Output: "/tmp/config",
	}, rancherServices)

	require.Error(t, err)
	require.Contains(t, err.Error(), "failed to collect passwords")
}

func TestSyncCommand_Execute_NoKubeconfigsDownloaded(t *testing.T) {
	ctx := context.Background()
	logger := slog.Default()

	// Setup mocks
	mockConfigRepo := mocks.NewMockConfigRepository(t)
	mockConfigProvider := mocks.NewMockConfigProvider(t)
	mockPasswordReader := mocks.NewMockPasswordReader(t)
	mockKubeconfigSyncer := mocks.NewMockKubeconfigSyncer(t)

	// Create test servers
	servers := []domain.ConfigServer{
		{URL: "https://rancher1.example.com", Username: "admin", AuthType: "local"},
	}

	rancherServices := domain.RancherServices{
		KubeconfigSyncer: mockKubeconfigSyncer,
	}

	// Set expectations for no downloads
	mockConfigRepo.EXPECT().GetServers(ctx).Return(servers, nil)
	mockPasswordReader.EXPECT().ReadPassword(ctx, "Password for https://rancher1.example.com: ").Return("pass1", nil)

	expectedPasswords := map[string]string{servers[0].ID(): "pass1"}
	mockKubeconfigSyncer.EXPECT().SyncAllServers(ctx, servers, expectedPasswords).Return([]string{}, nil)

	syncCommand := commands.NewSyncCommand(
		mockConfigRepo,
		mockConfigProvider,
		mockPasswordReader,
		logger,
	)

	err := syncCommand.Execute(ctx, commands.SyncRequest{
		Output: "/tmp/config",
	}, rancherServices)

	require.Error(t, err)
	require.Contains(t, err.Error(), "no kubeconfigs downloaded successfully")
}
