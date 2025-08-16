package config

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"slices"

	"gopkg.in/yaml.v3"

	"cowpoke/internal/domain"
	"cowpoke/internal/migrations"
)

const (
	dirPermissions  = 0o700 // Owner-only access for security
	filePermissions = 0o600 // Read/write owner only
	configVersion   = "2.0" // Current configuration version
)

// Repository handles configuration persistence.
type Repository struct {
	fs         domain.FileSystemAdapter
	configPath string
	config     *Config
	migrator   migrations.ConfigMigrator
	logger     *slog.Logger
}

// Config represents the cowpoke configuration structure.
type Config struct {
	Version string                `yaml:"version"`
	Servers []domain.ConfigServer `yaml:"servers"`
}

// NewRepository creates a new configuration repository.
func NewRepository(
	fs domain.FileSystemAdapter,
	configPath string,
	logger *slog.Logger,
) (*Repository, error) {
	repo := &Repository{
		fs:         fs,
		configPath: configPath,
		config:     &Config{Version: configVersion, Servers: []domain.ConfigServer{}},
		migrator:   migrations.NewMigrator(logger),
		logger:     logger,
	}

	configDir := filepath.Dir(configPath)
	if err := fs.MkdirAll(configDir, dirPermissions); err != nil {
		return nil, fmt.Errorf("failed to create config directory: %w", err)
	}

	if err := repo.LoadConfig(context.Background()); err != nil {
		if !errors.Is(err, os.ErrNotExist) {
			logger.Warn("Failed to load existing config, starting with empty config", "error", err)
		}
	}

	return repo, nil
}

// GetServers returns all configured servers.
func (r *Repository) GetServers(ctx context.Context) ([]domain.ConfigServer, error) {
	r.logger.DebugContext(ctx, "Getting servers from config", "count", len(r.config.Servers))
	return r.config.Servers, nil
}

// AddServer adds a new server to the configuration.
func (r *Repository) AddServer(ctx context.Context, server domain.ConfigServer) error {
	for _, existing := range r.config.Servers {
		if existing.URL == server.URL {
			return fmt.Errorf("server %s already exists in configuration", server.URL)
		}
	}

	r.config.Servers = append(r.config.Servers, server)
	r.logger.InfoContext(ctx, "Added server to configuration", "url", server.URL, "id", server.ID())

	if err := r.SaveConfig(ctx); err != nil {
		r.config.Servers = r.config.Servers[:len(r.config.Servers)-1] // Rollback
		return fmt.Errorf("failed to save configuration after adding server: %w", err)
	}

	return nil
}

// RemoveServer removes a server from the configuration.
func (r *Repository) RemoveServer(ctx context.Context, serverURL string) error {
	initialLength := len(r.config.Servers)
	oldServers := slices.Clone(r.config.Servers)

	r.config.Servers = slices.DeleteFunc(r.config.Servers, func(server domain.ConfigServer) bool {
		return server.URL == serverURL
	})

	if len(r.config.Servers) == initialLength {
		return fmt.Errorf("server %s not found in configuration", serverURL)
	}

	r.logger.InfoContext(ctx, "Removed server from configuration", "url", serverURL)

	if err := r.SaveConfig(ctx); err != nil {
		r.config.Servers = oldServers // Rollback
		return fmt.Errorf("failed to save configuration after removing server: %w", err)
	}

	return nil
}

// RemoveServerByID removes a server from the configuration by its ID.
func (r *Repository) RemoveServerByID(ctx context.Context, serverID string) error {
	initialLength := len(r.config.Servers)
	oldServers := slices.Clone(r.config.Servers)

	var removedServerURL string
	// Find the server URL before deletion for logging
	for _, server := range r.config.Servers {
		if server.ID() == serverID {
			removedServerURL = server.URL
			break
		}
	}

	r.config.Servers = slices.DeleteFunc(r.config.Servers, func(server domain.ConfigServer) bool {
		return server.ID() == serverID
	})

	if len(r.config.Servers) == initialLength {
		return fmt.Errorf("server with ID %s not found in configuration", serverID)
	}

	r.logger.InfoContext(ctx, "Removed server from configuration", "id", serverID, "url", removedServerURL)

	if err := r.SaveConfig(ctx); err != nil {
		r.config.Servers = oldServers // Rollback
		return fmt.Errorf("failed to save configuration after removing server: %w", err)
	}

	return nil
}

// SaveConfig saves the current configuration to disk.
func (r *Repository) SaveConfig(ctx context.Context) error {
	data, err := yaml.Marshal(r.config)
	if err != nil {
		return fmt.Errorf("failed to marshal configuration: %w", err)
	}

	if writeErr := r.fs.WriteFile(r.configPath, data, filePermissions); writeErr != nil {
		return fmt.Errorf("failed to write configuration file: %w", writeErr)
	}

	r.logger.DebugContext(ctx, "Configuration saved", "path", r.configPath)
	return nil
}

// LoadConfig loads the configuration from disk.
func (r *Repository) LoadConfig(ctx context.Context) error {
	data, err := r.fs.ReadFile(r.configPath)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			r.logger.DebugContext(ctx, "Configuration file does not exist", "path", r.configPath)
			return os.ErrNotExist
		}
		return fmt.Errorf("failed to read configuration file: %w", err)
	}

	// Try migration first
	servers, migrated, migrationErr := r.migrator.Migrate(ctx, data, configVersion)
	if migrationErr != nil {
		r.logger.WarnContext(ctx, "Migration failed, attempting direct load", "error", migrationErr)
	} else if migrated {
		// Successfully migrated
		r.config = &Config{Version: configVersion, Servers: servers}

		// Fix file and directory permissions for security during migration from v1.x
		if permErr := r.migrator.FixPermissionsPostMigration(ctx, r.configPath, r.fs); permErr != nil {
			r.logger.WarnContext(ctx, "Failed to fix permissions during migration", "error", permErr)
		}

		if saveErr := r.SaveConfig(ctx); saveErr != nil {
			r.logger.WarnContext(ctx, "Failed to save migrated config", "error", saveErr)
		}
		r.logger.InfoContext(ctx, "Configuration migrated and loaded",
			"path", r.configPath,
			"version", configVersion,
			"servers", len(servers))
		return nil
	}

	// Load as current version (no migration needed or migration failed)
	var config Config
	if unmarshalErr := yaml.Unmarshal(data, &config); unmarshalErr != nil {
		return fmt.Errorf("failed to unmarshal configuration: %w", unmarshalErr)
	}

	r.config = &config
	r.logger.InfoContext(ctx, "Configuration loaded",
		"path", r.configPath,
		"version", config.Version,
		"servers", len(config.Servers))
	return nil
}
