package migrations_test

import (
	"context"
	"errors"
	"log/slog"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"

	"cowpoke/internal/domain"
	"cowpoke/internal/migrations"
	"cowpoke/internal/mocks"
)

func TestMigrator_Migrate_V1ToV2(t *testing.T) {
	ctx := context.Background()
	logger := slog.Default()
	migrator := migrations.NewMigrator(logger)

	// V1 configuration data
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
    username: "admin"
    authType: "openldap"`

	servers, migrated, err := migrator.Migrate(ctx, []byte(v1Config), "2.0")
	if err != nil {
		t.Fatalf("Migration failed: %v", err)
	}

	if !migrated {
		t.Error("Expected migration to occur")
	}

	// Verify migrated servers
	want := []domain.ConfigServer{
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

	assert.Equal(t, want, servers, "Migrated servers should match expected")
}

func TestMigrator_Migrate_NoVersionToV2(t *testing.T) {
	ctx := context.Background()
	logger := slog.Default()
	migrator := migrations.NewMigrator(logger)

	// V1 configuration without version field (very old format)
	oldConfig := `servers:
  - id: "legacy-server"
    name: "Legacy Rancher"
    url: "https://rancher.legacy.example.com"
    username: "admin"
    authType: "local"`

	servers, migrated, err := migrator.Migrate(ctx, []byte(oldConfig), "2.0")
	if err != nil {
		t.Fatalf("Migration failed: %v", err)
	}

	if !migrated {
		t.Error("Expected migration to occur for versionless config")
	}

	// Verify migrated servers
	want := []domain.ConfigServer{
		{
			URL:      "https://rancher.legacy.example.com",
			Username: "admin",
			AuthType: "local",
		},
	}

	assert.Equal(t, want, servers, "Migrated servers should match expected")
}

func TestMigrator_Migrate_CurrentVersion(t *testing.T) {
	ctx := context.Background()
	logger := slog.Default()
	migrator := migrations.NewMigrator(logger)

	// V2 configuration (current version)
	v2Config := `version: "2.0"
servers:
  - url: "https://rancher.example.com"
    username: "admin"
    authType: "local"`

	servers, migrated, err := migrator.Migrate(ctx, []byte(v2Config), "2.0")
	if err != nil {
		t.Fatalf("Migration failed: %v", err)
	}

	if migrated {
		t.Error("Expected no migration for current version")
	}

	if servers != nil {
		t.Error("Expected nil servers when no migration occurs")
	}
}

func TestMigrator_Migrate_UnsupportedVersion(t *testing.T) {
	ctx := context.Background()
	logger := slog.Default()
	migrator := migrations.NewMigrator(logger)

	// Unsupported future version
	futureConfig := `version: "3.0"
servers: []`

	_, _, err := migrator.Migrate(ctx, []byte(futureConfig), "2.0")
	if err == nil {
		t.Error("Expected error for unsupported version")
	}

	expectedError := "unsupported configuration version: 3.0"
	if err.Error() != expectedError {
		t.Errorf("Expected error '%s', got: %v", expectedError, err)
	}
}

func TestMigrator_Migrate_InvalidYAML(t *testing.T) {
	ctx := context.Background()
	logger := slog.Default()
	migrator := migrations.NewMigrator(logger)

	invalidConfig := `invalid: yaml: structure
  - missing: proper
    formatting`

	_, _, err := migrator.Migrate(ctx, []byte(invalidConfig), "2.0")
	if err == nil {
		t.Error("Expected error for invalid YAML")
	}
}

func TestMigrator_FixPermissionsPostMigration_Success(t *testing.T) {
	logger := slog.Default()
	migrator := migrations.NewMigrator(logger)

	mockFS := mocks.NewMockFileSystemAdapter(t)

	tempDir := t.TempDir()
	configPath := tempDir + "/.config/cowpoke/config.yaml"
	configDir := tempDir + "/.config/cowpoke"

	// Set up expectations for the chmod calls
	mockFS.EXPECT().Chmod(configPath, os.FileMode(0o600)).Return(nil)
	mockFS.EXPECT().Chmod(configDir, os.FileMode(0o700)).Return(nil)

	// Set up expectations for stat calls that check if kubeconfig directory exists
	kubeconfigDir := tempDir + "/.kube"
	mockFS.EXPECT().Stat(kubeconfigDir).Return(nil, os.ErrNotExist)

	err := migrator.FixPermissionsPostMigration(context.Background(), configPath, mockFS)
	if err != nil {
		t.Errorf("Expected no error, got: %v", err)
	}
}

func TestMigrator_FixPermissionsPostMigration_ChmodError(t *testing.T) {
	logger := slog.Default()
	migrator := migrations.NewMigrator(logger)

	mockFS := mocks.NewMockFileSystemAdapter(t)

	tempDir := t.TempDir()
	configPath := tempDir + "/.config/cowpoke/config.yaml"

	// Set up expectation for chmod to fail
	chmodErr := errors.New("permission denied")
	mockFS.EXPECT().Chmod(configPath, os.FileMode(0o600)).Return(chmodErr)

	err := migrator.FixPermissionsPostMigration(context.Background(), configPath, mockFS)
	if err == nil {
		t.Error("Expected error when chmod fails")
	}

	expectedErrorMsg := "failed to fix config file permissions: permission denied"
	if err.Error() != expectedErrorMsg {
		t.Errorf("Expected error message '%s', got: %v", expectedErrorMsg, err)
	}
}

func TestMigrator_FixPermissionsPostMigration_WithExistingKubeconfigDir(t *testing.T) {
	logger := slog.Default()
	migrator := migrations.NewMigrator(logger)

	mockFS := mocks.NewMockFileSystemAdapter(t)

	tempDir := t.TempDir()
	configPath := tempDir + "/.config/cowpoke/config.yaml"
	configDir := tempDir + "/.config/cowpoke"
	kubeconfigDir := tempDir + "/.kube"

	// Set up expectations for successful chmod calls
	mockFS.EXPECT().Chmod(configPath, os.FileMode(0o600)).Return(nil)
	mockFS.EXPECT().Chmod(configDir, os.FileMode(0o700)).Return(nil)

	// Set up expectation that kubeconfig directory exists
	mockFS.EXPECT().Stat(kubeconfigDir).Return(nil, nil)
	mockFS.EXPECT().Chmod(kubeconfigDir, os.FileMode(0o700)).Return(nil)

	err := migrator.FixPermissionsPostMigration(context.Background(), configPath, mockFS)
	if err != nil {
		t.Errorf("Expected no error, got: %v", err)
	}
}
