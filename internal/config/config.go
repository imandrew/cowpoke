package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/google/uuid"
	"gopkg.in/yaml.v3"

	"cowpoke/internal/errors"
)

// RancherServer represents a Rancher server configuration.
type RancherServer struct {
	ID       string `yaml:"id"`
	Name     string `yaml:"name"`
	URL      string `yaml:"url"`
	Username string `yaml:"username"`
	AuthType string `yaml:"authType"`
}

// Config represents the main configuration structure.
type Config struct {
	Version string          `yaml:"version"`
	Servers []RancherServer `yaml:"servers"`
}

// Cluster represents a Kubernetes cluster managed by Rancher.
type Cluster struct {
	ID         string `yaml:"id"`
	Name       string `yaml:"name"`
	Type       string `yaml:"type"`
	ServerID   string `yaml:"serverId"`
	ServerName string `yaml:"serverName"`
	ServerURL  string `yaml:"serverUrl"`
}

// validAuthTypes contains all supported Rancher authentication types.
//
//nolint:gochecknoglobals // Package-level constants for auth validation
var validAuthTypes = []string{
	"local",
	"github",
	"openldap",
	"activedirectory",
	"azuread",
	"okta",
	"ping",
	"keycloak",
	"shibboleth",
	"googleoauth",
}

// IsValidAuthType checks if the provided auth type is supported.
func IsValidAuthType(authType string) bool {
	for _, valid := range validAuthTypes {
		if valid == authType {
			return true
		}
	}
	return false
}

// GetSupportedAuthTypesString returns a formatted string of all supported auth types.
func GetSupportedAuthTypesString() string {
	return strings.Join(validAuthTypes, ", ")
}

// Manager handles configuration file operations.
type Manager struct {
	configPath string
}

// NewConfigManager creates a new configuration manager.
func NewConfigManager(configPath string) *Manager {
	return &Manager{
		configPath: configPath,
	}
}

// LoadConfig loads the configuration from the file system.
func (cm *Manager) LoadConfig() (*Config, error) {
	if _, err := os.Stat(cm.configPath); os.IsNotExist(err) {
		return &Config{
			Version: "1.0",
			Servers: []RancherServer{},
		}, nil
	}

	data, err := os.ReadFile(cm.configPath)
	if err != nil {
		return nil, errors.NewConfigurationError("config_path", cm.configPath, "failed to read config file", err)
	}

	if len(data) == 0 {
		return &Config{
			Version: "1.0",
			Servers: []RancherServer{},
		}, nil
	}

	var config Config
	err = yaml.Unmarshal(data, &config)
	if err != nil {
		return nil, errors.NewConfigurationError("config_format", "yaml", "failed to unmarshal config", err)
	}

	return &config, nil
}

// SaveConfig saves the configuration to the file system.
func (cm *Manager) SaveConfig(config *Config) error {
	if err := os.MkdirAll(filepath.Dir(cm.configPath), 0o750); err != nil {
		return errors.NewConfigurationError(
			"config_directory",
			filepath.Dir(cm.configPath),
			"failed to create config directory",
			err,
		)
	}

	data, err := yaml.Marshal(config)
	if err != nil {
		return errors.NewConfigurationError("config_format", "yaml", "failed to marshal config", err)
	}

	err = os.WriteFile(cm.configPath, data, 0o600)
	if err != nil {
		return errors.NewConfigurationError("config_path", cm.configPath, "failed to write config file", err)
	}

	return nil
}

// AddServer adds a new Rancher server to the configuration.
func (cm *Manager) AddServer(server RancherServer) error {
	if !IsValidAuthType(server.AuthType) {
		return errors.NewValidationError(
			"auth_type",
			server.AuthType,
			"supported_values",
			fmt.Sprintf("auth type must be one of: %s", GetSupportedAuthTypesString()),
		)
	}

	config, err := cm.LoadConfig()
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	if server.ID == "" {
		server.ID = uuid.New().String()
	}

	config.Servers = append(config.Servers, server)

	return cm.SaveConfig(config)
}

// RemoveServer removes a Rancher server by ID from the configuration.
func (cm *Manager) RemoveServer(serverID string) error {
	config, err := cm.LoadConfig()
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	found := false
	newServers := make([]RancherServer, 0, len(config.Servers))

	for _, server := range config.Servers {
		if server.ID != serverID {
			newServers = append(newServers, server)
		} else {
			found = true
		}
	}

	if !found {
		return errors.NewValidationError("server_id", serverID, "exists", "server not found")
	}

	config.Servers = newServers
	return cm.SaveConfig(config)
}

// RemoveServerByURL removes a Rancher server by URL from the configuration.
func (cm *Manager) RemoveServerByURL(url string) error {
	config, err := cm.LoadConfig()
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	found := false
	newServers := make([]RancherServer, 0, len(config.Servers))

	for _, server := range config.Servers {
		if server.URL != url {
			newServers = append(newServers, server)
		} else {
			found = true
		}
	}

	if !found {
		return errors.NewValidationError("server_url", url, "exists", "server not found")
	}

	config.Servers = newServers
	return cm.SaveConfig(config)
}

// GetServers returns all configured Rancher servers.
func (cm *Manager) GetServers() ([]RancherServer, error) {
	config, err := cm.LoadConfig()
	if err != nil {
		return nil, err
	}

	return config.Servers, nil
}
