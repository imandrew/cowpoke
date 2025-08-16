// Package migrations handles configuration migrations between versions.
package migrations

import (
	"gopkg.in/yaml.v3"

	"cowpoke/internal/domain"
)

// V1RancherServer represents the old v1.x configuration structure.
type V1RancherServer struct {
	ID       string `yaml:"id"`
	Name     string `yaml:"name"`
	URL      string `yaml:"url"`
	Username string `yaml:"username"`
	AuthType string `yaml:"authType"`
}

// V1Config represents the old v1.x configuration structure.
type V1Config struct {
	Version string            `yaml:"version"`
	Servers []V1RancherServer `yaml:"servers"`
}

// migrateFromV1 converts v1 configuration to v2 format.
func migrateFromV1(data []byte) ([]domain.ConfigServer, error) {
	var v1Config V1Config
	if err := yaml.Unmarshal(data, &v1Config); err != nil {
		return nil, err
	}

	// Convert v1 servers to v2 format.
	servers := make([]domain.ConfigServer, len(v1Config.Servers))
	for i, v1Server := range v1Config.Servers {
		servers[i] = domain.ConfigServer{
			URL:      v1Server.URL,
			Username: v1Server.Username,
			AuthType: v1Server.AuthType,
			// Note: ID and Name fields are dropped - v2 uses dynamic ID generation.
		}
	}

	return servers, nil
}
