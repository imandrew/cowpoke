package rancher

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"cowpoke/internal/domain"
)

// Client handles all Rancher API operations.
type Client struct {
	httpAdapter domain.HTTPAdapter
	logger      *slog.Logger
}

// normalizeURL removes trailing slashes from a URL to ensure consistent API endpoint construction.
func normalizeURL(url string) string {
	return strings.TrimSuffix(url, "/")
}

// NewClient creates a new Rancher client.
func NewClient(httpAdapter domain.HTTPAdapter, logger *slog.Logger) *Client {
	return &Client{
		httpAdapter: httpAdapter,
		logger:      logger,
	}
}

// Authenticate performs authentication with a Rancher server.
func (c *Client) Authenticate(
	ctx context.Context,
	server domain.ConfigServer,
	password string,
) (domain.AuthToken, error) {
	authURL := fmt.Sprintf("%s/v3-public/%sProviders/%s?action=login",
		normalizeURL(server.URL), server.AuthType, server.AuthType)

	payload := map[string]string{
		"username": server.Username,
		"password": password,
	}

	c.logger.InfoContext(ctx, "Authenticating with Rancher server",
		"server", server.URL,
		"username", server.Username,
		"authType", server.AuthType)
	resp, err := c.httpAdapter.Post(ctx, authURL, payload)
	if err != nil {
		return nil, fmt.Errorf("authentication request failed: %w", err)
	}
	defer resp.Body.Close()

	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read auth response body: %w", err)
	}

	var authResp authResponse
	err = json.Unmarshal(bodyBytes, &authResp)
	if err != nil {
		return nil, fmt.Errorf("authentication failed with status %d: %s", resp.StatusCode, string(bodyBytes))
	}

	if authResp.Token == "" {
		if len(bodyBytes) > 0 {
			return nil, fmt.Errorf("authentication failed with status %d: %s", resp.StatusCode, string(bodyBytes))
		}
		return nil, fmt.Errorf("authentication failed: no token in response (status %d)", resp.StatusCode)
	}

	c.logger.InfoContext(ctx, "Authentication successful",
		"server", server.URL,
		"userID", authResp.UserID)

	var expiresAt time.Time
	if authResp.ExpiresAt != "" {
		if parsedTime, parseErr := time.Parse(time.RFC3339, authResp.ExpiresAt); parseErr == nil {
			expiresAt = parsedTime
		}
	}
	if expiresAt.IsZero() && authResp.TTL > 0 {
		expiresAt = time.Now().Add(time.Duration(authResp.TTL) * time.Millisecond)
	}
	if expiresAt.IsZero() {
		// Final fallback to default 16 hours (Rancher's typical default).
		expiresAt = time.Now().Add(16 * time.Hour) //nolint:mnd // Rancher default session TTL
	}

	return &token{
		value:     authResp.Token,
		expiresAt: expiresAt,
	}, nil
}

// ListClusters retrieves all clusters from a Rancher server.
func (c *Client) ListClusters(
	ctx context.Context,
	token domain.AuthToken,
	server domain.ConfigServer,
) ([]domain.Cluster, error) {
	clustersURL := fmt.Sprintf("%s/v3/clusters", normalizeURL(server.URL))

	c.logger.InfoContext(ctx, "Fetching clusters from Rancher server", "server", server.URL)

	resp, err := c.httpAdapter.GetWithAuth(ctx, clustersURL, token.Value())
	if err != nil {
		return nil, fmt.Errorf("failed to list clusters: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf(
			"list clusters failed with status %d: %s",
			resp.StatusCode,
			string(body),
		)
	}

	var clustersResp clustersResponse
	if decodeErr := json.NewDecoder(resp.Body).Decode(&clustersResp); decodeErr != nil {
		return nil, fmt.Errorf("failed to decode clusters response: %w", decodeErr)
	}

	clusters := make([]domain.Cluster, 0, len(clustersResp.Data))
	for _, cluster := range clustersResp.Data {
		if cluster.ID == "" || cluster.Name == "" {
			c.logger.WarnContext(ctx, "Skipping invalid cluster", "id", cluster.ID, "name", cluster.Name)
			continue
		}

		clusters = append(clusters, domain.Cluster{
			ID:   cluster.ID,
			Name: cluster.Name,
			Type: cluster.Type,
		})
	}

	c.logger.InfoContext(ctx, "Successfully fetched clusters",
		"server", server.URL,
		"count", len(clusters))

	return clusters, nil
}

// GetKubeconfig retrieves the kubeconfig for a specific cluster.
func (c *Client) GetKubeconfig(
	ctx context.Context,
	token domain.AuthToken,
	server domain.ConfigServer,
	clusterID string,
) ([]byte, error) {
	kubeconfigURL := fmt.Sprintf(
		"%s/v3/clusters/%s?action=generateKubeconfig",
		normalizeURL(server.URL),
		clusterID,
	)

	c.logger.InfoContext(ctx, "Fetching kubeconfig for cluster",
		"server", server.URL,
		"cluster", clusterID)

	resp, err := c.httpAdapter.PostWithAuth(ctx, kubeconfigURL, token.Value(), nil)
	if err != nil {
		return nil, fmt.Errorf("failed to get kubeconfig: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf(
			"get kubeconfig failed with status %d: %s",
			resp.StatusCode,
			string(body),
		)
	}

	var kubeconfigResp kubeconfigResponse
	if decodeErr := json.NewDecoder(resp.Body).Decode(&kubeconfigResp); decodeErr != nil {
		return nil, fmt.Errorf("failed to decode kubeconfig response: %w", decodeErr)
	}

	if kubeconfigResp.Config == "" {
		return nil, errors.New("kubeconfig generation succeeded but no config was returned")
	}

	c.logger.InfoContext(ctx, "Successfully fetched kubeconfig",
		"server", server.URL,
		"cluster", clusterID)

	return []byte(kubeconfigResp.Config), nil
}

// authResponse represents the Rancher authentication response.
type authResponse struct {
	Token     string `json:"token"`
	Type      string `json:"type"`
	UserID    string `json:"userId"`
	TTL       int64  `json:"ttl"`       // TTL in milliseconds.
	ExpiresAt string `json:"expiresAt"` // ISO 8601 timestamp.
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

// clustersResponse represents the Rancher clusters list response.
type clustersResponse struct {
	Data []clusterData `json:"data"`
}

// clusterData represents a single cluster in the response.
type clusterData struct {
	ID   string `json:"id"`
	Name string `json:"name"`
	Type string `json:"type"`
}

// kubeconfigResponse represents the Rancher kubeconfig generation response.
type kubeconfigResponse struct {
	Config string `json:"config"`
}
