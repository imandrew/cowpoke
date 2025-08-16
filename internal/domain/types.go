package domain

import (
	"context"
	"time"
)

// AuthToken represents an authenticated session.
type AuthToken interface {
	Value() string
	IsValid() bool
	ExpiresAt() time.Time
}

// Cluster represents a Kubernetes cluster in Rancher.
type Cluster struct {
	ID   string
	Name string
	Type string
}

// PasswordReader handles secure password input from users.
type PasswordReader interface {
	ReadPassword(ctx context.Context, prompt string) (string, error)
	IsInteractive() bool
}

// ClusterFilter determines whether a cluster should be excluded from operations.
type ClusterFilter interface {
	ShouldExclude(clusterName string) bool
}
