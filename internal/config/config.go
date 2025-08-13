package config

import (
	"fmt"
	"os"
	"path/filepath"

	"cowpoke/internal/errors"
	"github.com/google/uuid"
	"gopkg.in/yaml.v3"
)

// RancherServer represents a Rancher server configuration
type RancherServer struct {
	ID       string `yaml:"id"`
	Name     string `yaml:"name"`
	URL      string `yaml:"url"`
	Username string `yaml:"username"`
	AuthType string `yaml:"authType"`
}

// Config represents the main configuration structure
type Config struct {
	Version string          `yaml:"version"`
	Servers []RancherServer `yaml:"servers"`
}

// Cluster represents a Kubernetes cluster managed by Rancher
type Cluster struct {
	ID         string `yaml:"id"`
	Name       string `yaml:"name"`
	Type       string `yaml:"type"`
	ServerID   string `yaml:"serverId"`
	ServerName string `yaml:"serverName"`
	ServerURL  string `yaml:"serverUrl"`
}

// AuthProvider represents an authentication provider configuration
type AuthProvider struct {
	ID          string `json:"id" yaml:"id"`
	DisplayName string `json:"display_name" yaml:"display_name"`
	Provider    string `json:"provider" yaml:"provider"`
	Description string `json:"description,omitempty" yaml:"description,omitempty"`
}

func (a AuthProvider) String() string {
	return a.ID
}

func (a AuthProvider) URLPath() string {
	return a.Provider
}

// AuthRegistry manages authentication providers
type AuthRegistry struct {
	providers map[string]AuthProvider
	order     []string
}

// NewAuthRegistry creates a new authentication registry
func NewAuthRegistry() *AuthRegistry {
	return &AuthRegistry{
		providers: make(map[string]AuthProvider),
		order:     make([]string, 0),
	}
}

// Register adds or updates an authentication provider
func (r *AuthRegistry) Register(provider AuthProvider) {
	if _, exists := r.providers[provider.ID]; !exists {
		r.order = append(r.order, provider.ID)
	}
	r.providers[provider.ID] = provider
}

// Get retrieves an authentication provider by ID
func (r *AuthRegistry) Get(id string) (AuthProvider, bool) {
	provider, exists := r.providers[id]
	return provider, exists
}

// All returns all registered authentication providers in order
func (r *AuthRegistry) All() []AuthProvider {
	result := make([]AuthProvider, 0, len(r.order))
	for _, id := range r.order {
		result = append(result, r.providers[id])
	}
	return result
}

// IsValid checks if an authentication provider ID is registered
func (r *AuthRegistry) IsValid(id string) bool {
	_, exists := r.providers[id]
	return exists
}

// IDs returns all registered provider IDs in order
func (r *AuthRegistry) IDs() []string {
	return append([]string(nil), r.order...)
}

// String returns a string representation of all provider IDs
func (r *AuthRegistry) String() string {
	ids := r.IDs()
	return fmt.Sprintf("%v", ids)
}

// SupportedAuthTypes is the global registry of supported authentication types
var SupportedAuthTypes = NewAuthRegistry()

func init() {
	// Register all supported Rancher authentication providers
	SupportedAuthTypes.Register(AuthProvider{
		ID:          "local",
		DisplayName: "Local Authentication",
		Provider:    "local",
		Description: "Rancher local user authentication",
	})
	SupportedAuthTypes.Register(AuthProvider{
		ID:          "github",
		DisplayName: "GitHub",
		Provider:    "github",
		Description: "GitHub OAuth authentication",
	})
	SupportedAuthTypes.Register(AuthProvider{
		ID:          "openldap",
		DisplayName: "OpenLDAP",
		Provider:    "openldap",
		Description: "OpenLDAP authentication",
	})
	SupportedAuthTypes.Register(AuthProvider{
		ID:          "activedirectory",
		DisplayName: "Active Directory",
		Provider:    "activedirectory",
		Description: "Microsoft Active Directory authentication",
	})
	SupportedAuthTypes.Register(AuthProvider{
		ID:          "azuread",
		DisplayName: "Azure AD",
		Provider:    "azuread",
		Description: "Azure Active Directory authentication",
	})
	SupportedAuthTypes.Register(AuthProvider{
		ID:          "okta",
		DisplayName: "Okta",
		Provider:    "okta",
		Description: "Okta SAML authentication",
	})
	SupportedAuthTypes.Register(AuthProvider{
		ID:          "ping",
		DisplayName: "PingIdentity",
		Provider:    "ping",
		Description: "PingIdentity SAML authentication",
	})
	SupportedAuthTypes.Register(AuthProvider{
		ID:          "keycloak",
		DisplayName: "Keycloak",
		Provider:    "keycloak",
		Description: "Keycloak OIDC/SAML authentication",
	})
	SupportedAuthTypes.Register(AuthProvider{
		ID:          "shibboleth",
		DisplayName: "Shibboleth",
		Provider:    "shibboleth",
		Description: "Shibboleth SAML authentication",
	})
	SupportedAuthTypes.Register(AuthProvider{
		ID:          "googleoauth",
		DisplayName: "Google OAuth",
		Provider:    "googleoauth",
		Description: "Google OAuth authentication",
	})
}

// IsValidAuthType checks if the provided auth type is supported
func IsValidAuthType(authType string) bool {
	return SupportedAuthTypes.IsValid(authType)
}

// GetSupportedAuthTypesString returns a formatted string of all supported auth types
func GetSupportedAuthTypesString() string {
	return SupportedAuthTypes.String()
}

// ConfigManager handles configuration file operations
type ConfigManager struct {
	configPath string
}

// NewConfigManager creates a new configuration manager
func NewConfigManager(configPath string) *ConfigManager {
	return &ConfigManager{
		configPath: configPath,
	}
}

// LoadConfig loads the configuration from the file system
func (cm *ConfigManager) LoadConfig() (*Config, error) {
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

// SaveConfig saves the configuration to the file system
func (cm *ConfigManager) SaveConfig(config *Config) error {
	if err := os.MkdirAll(filepath.Dir(cm.configPath), 0755); err != nil {
		return errors.NewConfigurationError("config_directory", filepath.Dir(cm.configPath), "failed to create config directory", err)
	}

	data, err := yaml.Marshal(config)
	if err != nil {
		return errors.NewConfigurationError("config_format", "yaml", "failed to marshal config", err)
	}

	err = os.WriteFile(cm.configPath, data, 0644)
	if err != nil {
		return errors.NewConfigurationError("config_path", cm.configPath, "failed to write config file", err)
	}

	return nil
}

// AddServer adds a new Rancher server to the configuration
func (cm *ConfigManager) AddServer(server RancherServer) error {
	if !IsValidAuthType(server.AuthType) {
		return errors.NewValidationError("auth_type", server.AuthType, "supported_values", fmt.Sprintf("auth type must be one of: %s", GetSupportedAuthTypesString()))
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

// RemoveServer removes a Rancher server by ID from the configuration
func (cm *ConfigManager) RemoveServer(serverID string) error {
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

// RemoveServerByURL removes a Rancher server by URL from the configuration
func (cm *ConfigManager) RemoveServerByURL(url string) error {
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

// GetServers returns all configured Rancher servers
func (cm *ConfigManager) GetServers() ([]RancherServer, error) {
	config, err := cm.LoadConfig()
	if err != nil {
		return nil, err
	}

	return config.Servers, nil
}