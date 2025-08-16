package domain

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"net/url"
)

// ConfigRepository manages the cowpoke configuration.
type ConfigRepository interface {
	GetServers(ctx context.Context) ([]ConfigServer, error)
	AddServer(ctx context.Context, server ConfigServer) error
	RemoveServer(ctx context.Context, serverURL string) error
	RemoveServerByID(ctx context.Context, serverID string) error
	SaveConfig(ctx context.Context) error
	LoadConfig(ctx context.Context) error
}

// ConfigProvider provides configuration paths and defaults.
type ConfigProvider interface {
	GetDefaultKubeconfigPath() (string, error)
	GetKubeconfigDir() (string, error)
	GetConfigPath() (string, error)
}

// ConfigServer represents a Rancher server in the configuration.
type ConfigServer struct {
	URL      string `yaml:"url"`
	Username string `yaml:"username"`
	AuthType string `yaml:"authType"`
}

// ID returns a deterministic 8-character ID generated from the server domain.
func (cs *ConfigServer) ID() string {
	// Extract the domain part from the URL.
	domain := cs.extractDomain()
	hash := sha256.Sum256([]byte(domain))
	return hex.EncodeToString(hash[:])[:8]
}

// extractDomain extracts just the domain part from the server URL.
func (cs *ConfigServer) extractDomain() string {
	parsedURL, err := url.Parse(cs.URL)
	if err != nil {
		// Fallback to raw URL if parsing fails.
		return cs.URL
	}

	hostname := parsedURL.Hostname()
	if hostname == "" {
		// Fallback to raw URL if no hostname found.
		return cs.URL
	}

	return hostname
}
