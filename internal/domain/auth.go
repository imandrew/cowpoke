package domain

import (
	"context"
	"time"
)

// PasswordReader handles secure password input from users.
type PasswordReader interface {
	ReadPassword(ctx context.Context, prompt string) (string, error)
	IsInteractive() bool
}

// RancherAuthenticator handles authentication with Rancher servers.
type RancherAuthenticator interface {
	Authenticate(ctx context.Context, server ConfigServer, password string) (AuthToken, error)
}

// AuthToken represents an authenticated session.
type AuthToken interface {
	Value() string
	IsValid() bool
	ExpiresAt() time.Time
}
