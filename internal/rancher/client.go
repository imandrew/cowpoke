package rancher

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"math"
	"net/http"
	"time"

	"cowpoke/internal/config"
	"cowpoke/internal/errors"
	"cowpoke/internal/logging"
)

// Client represents a Rancher API client
type Client struct {
	server     config.RancherServer
	httpClient *http.Client
	token      string
}

// NewClient creates a new Rancher client
func NewClient(server config.RancherServer) *Client {
	return &Client{
		server: server,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// Request/Response types for authentication
type authRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

type authResponse struct {
	Token string `json:"token"`
	Type  string `json:"type"`
	User  string `json:"user"`
}

// Request/Response types for clusters API
type clustersResponse struct {
	Data []clusterData `json:"data"`
}

type clusterData struct {
	ID   string `json:"id"`
	Name string `json:"name"`
	Type string `json:"type"`
}

// Request/Response types for kubeconfig API
type kubeconfigResponse struct {
	Config string `json:"config"`
}

// Authenticate authenticates the client with Rancher
func (c *Client) Authenticate(ctx context.Context, password string) (string, error) {
	authURL := fmt.Sprintf("%s/v3-public/%sProviders/%s?action=login",
		c.server.URL, c.server.AuthType, c.server.AuthType)

	payload := map[string]string{
		"username": c.server.Username,
		"password": password,
	}

	var response authResponse
	err := c.doRetryableRequest(ctx, "POST", authURL, payload, &response)
	if err != nil {
		return "", errors.NewAuthenticationError(c.server.URL, c.server.AuthType, c.server.Username, err)
	}

	c.token = response.Token
	return response.Token, nil
}

// makeJSONRequest makes an HTTP request with JSON payload and returns response body
func (c *Client) makeJSONRequest(ctx context.Context, method, url string, payload any) ([]byte, error) {
	var body io.Reader
	if payload != nil {
		jsonData, err := json.Marshal(payload)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal request: %w", err)
		}
		body = bytes.NewBuffer(jsonData)
	}

	req, err := http.NewRequestWithContext(ctx, method, url, body)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	if c.token != "" {
		req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", c.token))
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to make request: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode >= 400 {
		return nil, errors.NewHTTPError(resp.StatusCode, method, url, string(respBody))
	}

	return respBody, nil
}

// GetClusters retrieves all clusters from the Rancher server
func (c *Client) GetClusters(ctx context.Context) ([]config.Cluster, error) {
	if c.token == "" {
		return nil, errors.NewAuthenticationError(c.server.URL, c.server.AuthType, c.server.Username, errors.ErrUnauthorized)
	}

	clustersURL := fmt.Sprintf("%s/v3/clusters", c.server.URL)

	var clustersResp clustersResponse
	err := c.doRetryableRequest(ctx, "GET", clustersURL, nil, &clustersResp)
	if err != nil {
		return nil, fmt.Errorf("failed to get clusters: %w", err)
	}

	clusters := make([]config.Cluster, 0, len(clustersResp.Data))
	for _, clusterData := range clustersResp.Data {
		cluster := config.Cluster{
			ID:         clusterData.ID,
			Name:       clusterData.Name,
			Type:       clusterData.Type,
			ServerID:   c.server.ID,
			ServerName: c.server.Name,
			ServerURL:  c.server.URL,
		}
		clusters = append(clusters, cluster)
	}

	return clusters, nil
}

// GetKubeconfig retrieves the kubeconfig for a specific cluster
func (c *Client) GetKubeconfig(ctx context.Context, clusterID string) ([]byte, error) {
	if c.token == "" {
		return nil, errors.NewAuthenticationError(c.server.URL, c.server.AuthType, c.server.Username, errors.ErrUnauthorized)
	}

	kubeconfigURL := fmt.Sprintf("%s/v3/clusters/%s?action=generateKubeconfig", c.server.URL, clusterID)

	var kubeconfigResp kubeconfigResponse
	err := c.doRetryableRequest(ctx, "POST", kubeconfigURL, nil, &kubeconfigResp)
	if err != nil {
		return nil, fmt.Errorf("failed to get kubeconfig: %w", err)
	}

	return []byte(kubeconfigResp.Config), nil
}

// doRetryableRequest performs an HTTP request with retry logic
func (c *Client) doRetryableRequest(ctx context.Context, method, url string, payload any, response any) error {
	strategy := HTTPStrategy()
	return c.doWithRetry(ctx, strategy, func() error {
		body, err := c.makeJSONRequest(ctx, method, url, payload)
		if err != nil {
			return err
		}

		if response != nil {
			return json.Unmarshal(body, response)
		}
		return nil
	})
}

// Retry Strategy and Logic (integrated from retry package)

// Strategy defines retry behavior
type Strategy struct {
	MaxAttempts int
	BaseDelay   time.Duration
	MaxDelay    time.Duration
	Multiplier  float64
	Jitter      bool
}

// DefaultStrategy returns a sensible default retry strategy
func DefaultStrategy() Strategy {
	return Strategy{
		MaxAttempts: 3,
		BaseDelay:   time.Second,
		MaxDelay:    30 * time.Second,
		Multiplier:  2.0,
		Jitter:      true,
	}
}

// HTTPStrategy returns a retry strategy optimized for HTTP requests
func HTTPStrategy() Strategy {
	return Strategy{
		MaxAttempts: 3,
		BaseDelay:   500 * time.Millisecond,
		MaxDelay:    10 * time.Second,
		Multiplier:  1.5,
		Jitter:      true,
	}
}

// RetryFunc represents a function that can be retried
type RetryFunc func() error

// isRetryable determines if an error should be retried
func isRetryable(err error) bool {
	if err == nil {
		return false
	}

	// Check for specific error types that are typically retryable
	if errors.IsNetwork(err) {
		return true
	}

	// Check for specific HTTP status codes that are retryable
	if errors.IsHTTPStatus(err, 429) || // Too Many Requests
		errors.IsHTTPStatus(err, 502) || // Bad Gateway
		errors.IsHTTPStatus(err, 503) || // Service Unavailable
		errors.IsHTTPStatus(err, 504) { // Gateway Timeout
		return true
	}

	// Check for timeout errors (context deadline exceeded)
	if err == context.DeadlineExceeded {
		return true
	}

	return false
}

// calculateDelay computes the delay for the given attempt
func (s Strategy) calculateDelay(attempt int) time.Duration {
	if attempt <= 0 {
		return 0
	}

	delay := float64(s.BaseDelay) * math.Pow(s.Multiplier, float64(attempt-1))

	if delay > float64(s.MaxDelay) {
		delay = float64(s.MaxDelay)
	}

	// Add jitter to avoid thundering herd
	if s.Jitter {
		jitter := delay * 0.1 * (2.0*float64(time.Now().UnixNano()%1000)/1000.0 - 1.0)
		delay += jitter
	}

	return time.Duration(delay)
}

// doWithRetry executes the given function with retries according to the strategy
func (c *Client) doWithRetry(ctx context.Context, strategy Strategy, fn RetryFunc) error {
	logger := logging.Default().With("operation", "retry")

	var lastErr error
	var allErrors []error

	for attempt := 1; attempt <= strategy.MaxAttempts; attempt++ {
		// Execute the function
		err := fn()
		if err == nil {
			if attempt > 1 {
				logger.Info("Operation succeeded after retry",
					"attempt", attempt,
					"total_attempts", strategy.MaxAttempts)
			}
			return nil
		}

		lastErr = err
		allErrors = append(allErrors, err)

		logger.Warn("Operation failed",
			"error", err,
			"attempt", attempt,
			"max_attempts", strategy.MaxAttempts)

		// Check if this error is retryable
		if !isRetryable(err) {
			logger.Debug("Error is not retryable, stopping retry attempts", "error", err)
			break
		}

		// If this is the last attempt, don't wait
		if attempt == strategy.MaxAttempts {
			break
		}

		// Calculate delay and wait
		delay := strategy.calculateDelay(attempt)
		logger.Debug("Waiting before retry",
			"delay", delay,
			"next_attempt", attempt+1)

		select {
		case <-ctx.Done():
			logger.Debug("Context cancelled during retry delay")
			return fmt.Errorf("retry cancelled: %w", ctx.Err())
		case <-time.After(delay):
			// Continue to next attempt
		}
	}

	// All attempts failed, return appropriate error
	if len(allErrors) > 1 {
		logger.Error("All retry attempts failed",
			"attempts", len(allErrors),
			"last_error", lastErr)
		return errors.NewMultiError(allErrors)
	}

	return lastErr
}
