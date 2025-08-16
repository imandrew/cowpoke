package migrations

import (
	"context"
	"fmt"
	"log/slog"
	"path/filepath"

	"gopkg.in/yaml.v3"

	"cowpoke/internal/domain"
)

// ConfigMigrator handles configuration migrations between versions.
type ConfigMigrator interface {
	Migrate(ctx context.Context, data []byte, currentVersion string) ([]domain.ConfigServer, bool, error)
	FixPermissionsPostMigration(ctx context.Context, configPath string, fs domain.FileSystemAdapter) error
}

// Migrator implements configuration migration logic.
type Migrator struct {
	logger *slog.Logger
}

// NewMigrator creates a new configuration migrator.
func NewMigrator(logger *slog.Logger) *Migrator {
	return &Migrator{
		logger: logger,
	}
}

// Migrate attempts to migrate configuration data to the current version.
// Returns: servers, wasMigrated, error.
func (m *Migrator) Migrate(
	ctx context.Context,
	data []byte,
	currentVersion string,
) ([]domain.ConfigServer, bool, error) {
	// First try to detect the version.
	version, err := m.detectVersion(data)
	if err != nil {
		return nil, false, fmt.Errorf("failed to detect config version: %w", err)
	}

	m.logger.DebugContext(ctx, "Detected configuration version", "version", version, "current", currentVersion)

	// If already current version, no migration needed.
	if version == currentVersion {
		return nil, false, nil
	}

	// Handle migration based on detected version.
	switch version {
	case "", "1.0":
		return m.migrateFromV1(ctx, data)
	default:
		return nil, false, fmt.Errorf("unsupported configuration version: %s", version)
	}
}

// detectVersion attempts to detect the configuration version.
func (m *Migrator) detectVersion(data []byte) (string, error) {
	var versionCheck struct {
		Version string `yaml:"version"`
	}

	if err := yaml.Unmarshal(data, &versionCheck); err != nil {
		return "", err
	}

	// Empty version indicates v1.0 or earlier.
	if versionCheck.Version == "" {
		return "1.0", nil
	}

	return versionCheck.Version, nil
}

// migrateFromV1 handles migration from v1.x to current version.
func (m *Migrator) migrateFromV1(ctx context.Context, data []byte) ([]domain.ConfigServer, bool, error) {
	servers, err := migrateFromV1(data)
	if err != nil {
		return nil, false, fmt.Errorf("failed to migrate from v1: %w", err)
	}

	m.logger.InfoContext(ctx, "Successfully migrated configuration from v1.x", "servers", len(servers))
	return servers, true, nil
}

// FixPermissionsPostMigration fixes file and directory permissions after migration from v1.x.
// This ensures that users upgrading from v1.0.1 (which may not have had secure permissions)
// get the proper security settings for v2.0.
func (m *Migrator) FixPermissionsPostMigration(
	ctx context.Context,
	configPath string,
	fs domain.FileSystemAdapter,
) error {
	const (
		dirPermissions  = 0o700 // Owner-only access for security
		filePermissions = 0o600 // Read/write owner only
	)

	// Fix config file permissions.
	if err := fs.Chmod(configPath, filePermissions); err != nil {
		m.logger.WarnContext(ctx, "Failed to fix config file permissions",
			"path", configPath, "error", err)
		return fmt.Errorf("failed to fix config file permissions: %w", err)
	}

	// Fix config directory permissions.
	configDir := filepath.Dir(configPath)
	if err := fs.Chmod(configDir, dirPermissions); err != nil {
		m.logger.WarnContext(ctx, "Failed to fix config directory permissions",
			"path", configDir, "error", err)
		return fmt.Errorf("failed to fix config directory permissions: %w", err)
	}

	// Fix kubeconfig directory permissions if it exists.
	// This handles the case where users had ~/.kube or other kubeconfig dirs from v1.x.
	kubeconfigDirs := []string{
		filepath.Join(filepath.Dir(configDir), "..", ".kube"), // ~/.kube if config is in ~/.config/cowpoke
	}

	for _, kubeconfigDir := range kubeconfigDirs {
		// Only fix permissions if directory exists (don't create it).
		if _, err := fs.Stat(kubeconfigDir); err == nil {
			if chmodErr := fs.Chmod(kubeconfigDir, dirPermissions); chmodErr != nil {
				m.logger.WarnContext(ctx, "Failed to fix kubeconfig directory permissions",
					"path", kubeconfigDir, "error", chmodErr)
				// Don't fail the migration for kubeconfig directory permission issues.
			} else {
				m.logger.InfoContext(ctx, "Fixed kubeconfig directory permissions",
					"path", kubeconfigDir)
			}
		}
	}

	m.logger.InfoContext(ctx, "Fixed file and directory permissions post-migration",
		"config_file", configPath, "config_dir", configDir)
	return nil
}
