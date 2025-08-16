package config

import (
	"fmt"
	"path/filepath"

	"cowpoke/internal/domain"
)

// Provider provides configuration paths.
type Provider struct {
	fs domain.FileSystemAdapter
}

// NewProvider creates a new configuration provider.
func NewProvider(fs domain.FileSystemAdapter) *Provider {
	return &Provider{
		fs: fs,
	}
}

// GetDefaultKubeconfigPath returns the default kubeconfig path.
func (p *Provider) GetDefaultKubeconfigPath() (string, error) {
	homeDir, err := p.fs.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("failed to get home directory: %w", err)
	}
	return filepath.Join(homeDir, ".kube", "config"), nil
}

// GetKubeconfigDir returns the directory for storing individual kubeconfigs.
func (p *Provider) GetKubeconfigDir() (string, error) {
	homeDir, err := p.fs.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("failed to get home directory: %w", err)
	}
	return filepath.Join(homeDir, ".config", "cowpoke", "kubeconfigs"), nil
}

// GetConfigPath returns the path to the cowpoke configuration file.
func (p *Provider) GetConfigPath() (string, error) {
	homeDir, err := p.fs.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("failed to get home directory: %w", err)
	}
	return filepath.Join(homeDir, ".config", "cowpoke", "config.yaml"), nil
}
