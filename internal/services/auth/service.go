package auth

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"time"

	"cowpoke/internal/domain"
)

// AuthenticatorImpl handles authentication with Rancher servers.
type AuthenticatorImpl struct {
	httpAdapter domain.HTTPAdapter
	logger      *slog.Logger
}

// NewAuthenticator creates a new Rancher authenticator.
func NewAuthenticator(httpAdapter domain.HTTPAdapter, logger *slog.Logger) *AuthenticatorImpl {
	return &AuthenticatorImpl{
		httpAdapter: httpAdapter,
		logger:      logger,
	}
}

// Authenticate performs authentication with a Rancher server.
func (a *AuthenticatorImpl) Authenticate(
	ctx context.Context,
	server domain.ConfigServer,
	password string,
) (domain.AuthToken, error) {
	// Build the authentication URL based on auth type
	authURL := fmt.Sprintf("%s/v3-public/%sProviders/%s?action=login",
		server.URL, server.AuthType, server.AuthType)

	// Prepare the authentication payload
	payload := map[string]string{
		"username": server.Username,
		"password": password,
	}

	a.logger.DebugContext(ctx, "Authenticating with Rancher",
		"server", server.URL,
		"username", server.Username,
		"authType", server.AuthType)

	// Make the authentication request
	resp, err := a.httpAdapter.Post(ctx, authURL, payload)
	if err != nil {
		return nil, fmt.Errorf("authentication request failed: %w", err)
	}
	defer resp.Body.Close()

	// Check response status - Rancher login returns 201 (Created) for new tokens
	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf(
			"authentication failed with status %d: %s",
			resp.StatusCode,
			string(body),
		)
	}

	// Parse the authentication response
	var authResp authResponse
	err = json.NewDecoder(resp.Body).Decode(&authResp)
	if err != nil {
		return nil, fmt.Errorf("failed to decode auth response: %w", err)
	}

	// Validate the response
	if authResp.Token == "" {
		return nil, errors.New("authentication succeeded but no token was returned")
	}

	a.logger.DebugContext(ctx, "Authentication successful",
		"server", server.URL,
		"userID", authResp.UserID)

	// Calculate expiry time from response
	var expiresAt time.Time
	if authResp.ExpiresAt != "" {
		// Parse ISO 8601 timestamp
		if parsedTime, parseErr := time.Parse(time.RFC3339, authResp.ExpiresAt); parseErr == nil {
			expiresAt = parsedTime
		}
	}
	if expiresAt.IsZero() && authResp.TTL > 0 {
		// Fallback to TTL in milliseconds
		expiresAt = time.Now().Add(time.Duration(authResp.TTL) * time.Millisecond)
	}
	if expiresAt.IsZero() {
		// Final fallback to default 16 hours (Rancher's typical default)
		expiresAt = time.Now().Add(16 * time.Hour) //nolint:mnd // Rancher default session TTL
	}

	// Create and return the auth token
	return &token{
		value:     authResp.Token,
		expiresAt: expiresAt,
	}, nil
}

// authResponse represents the Rancher authentication response.
type authResponse struct {
	Token     string `json:"token"`
	Type      string `json:"type"`
	UserID    string `json:"userId"`
	TTL       int64  `json:"ttl"`       // TTL in milliseconds
	ExpiresAt string `json:"expiresAt"` // ISO 8601 timestamp
	Expired   bool   `json:"expired"`
}

// token implements the AuthToken interface.
type token struct {
	value     string
	expiresAt time.Time
}

func (t *token) Value() string        { return t.value }
func (t *token) IsValid() bool        { return time.Now().Before(t.expiresAt) }
func (t *token) ExpiresAt() time.Time { return t.expiresAt }
