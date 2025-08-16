package http

import (
	"context"
	"crypto/tls"
	"fmt"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/go-resty/resty/v2"
	"golang.org/x/time/rate"
)

const (
	// HTTP client retry configuration.
	defaultRetryCount       = 3
	defaultRetryMaxWaitTime = 5 * time.Second

	// Rate limiting configuration.
	rateLimitRequestsPerSecond = 10
	rateLimitBurst             = 20
)

const (
	// Standard HTTP content types.
	contentTypeJSON = "application/json"
)

// Adapter is an HTTP client adapter using resty with rate limiting.
type Adapter struct {
	client  *resty.Client
	limiter *rate.Limiter
	logger  *slog.Logger
}

// NewAdapter creates a new HTTP adapter with rate limiting and retry capabilities.
// Rate limit: 10 requests per second with burst of 20.
func NewAdapter(timeout time.Duration, insecureSkipVerify bool, logger *slog.Logger) *Adapter {
	client := resty.New().
		SetTimeout(timeout).
		SetRetryCount(defaultRetryCount).
		SetRetryWaitTime(time.Second).
		SetRetryMaxWaitTime(defaultRetryMaxWaitTime).
		SetTLSClientConfig(&tls.Config{
			InsecureSkipVerify: insecureSkipVerify, //nolint:gosec // User-configurable for self-signed certificates
		})

	// Rate limiter: 10 requests/second with burst of 20
	limiter := rate.NewLimiter(rate.Limit(rateLimitRequestsPerSecond), rateLimitBurst)

	// Add rate limiting middleware
	client.OnBeforeRequest(func(_ *resty.Client, req *resty.Request) error {
		return limiter.Wait(req.Context())
	})

	// Add logging middleware
	client.OnBeforeRequest(func(_ *resty.Client, req *resty.Request) error {
		logger.DebugContext(req.Context(), "HTTP request",
			"method", req.Method,
			"url", req.URL,
		)
		return nil
	})

	client.OnAfterResponse(func(_ *resty.Client, resp *resty.Response) error {
		logger.DebugContext(resp.Request.Context(), "HTTP response",
			"method", resp.Request.Method,
			"url", resp.Request.URL,
			"status", resp.StatusCode(),
			"duration", resp.Time(),
		)
		return nil
	})

	return &Adapter{
		client:  client,
		limiter: limiter,
		logger:  logger,
	}
}

// Get performs a GET request.
func (a *Adapter) Get(ctx context.Context, url string) (*http.Response, error) {
	resp, err := a.client.R().SetContext(ctx).SetDoNotParseResponse(true).Get(url)
	if err != nil {
		return nil, fmt.Errorf("failed to execute GET request: %w", err)
	}
	return resp.RawResponse, nil
}

// GetWithAuth performs a GET request with authentication.
func (a *Adapter) GetWithAuth(ctx context.Context, url, token string) (*http.Response, error) {
	resp, err := a.client.R().
		SetContext(ctx).
		SetAuthToken(token).
		SetDoNotParseResponse(true).
		Get(url)
	if err != nil {
		return nil, fmt.Errorf("failed to execute authenticated GET request: %w", err)
	}
	return resp.RawResponse, nil
}

// Post performs a POST request with optional JSON payload.
func (a *Adapter) Post(
	ctx context.Context,
	url string,
	payload any,
) (*http.Response, error) {
	request := a.client.R().SetContext(ctx).SetDoNotParseResponse(true)

	if payload != nil {
		request.SetHeader("Content-Type", contentTypeJSON).SetBody(payload)
	}

	resp, err := request.Post(url)
	if err != nil {
		// Handle resty marshaling errors
		if strings.Contains(err.Error(), "unsupported 'Body' type/value") {
			return nil, fmt.Errorf("failed to prepare POST payload: %w", err)
		}
		return nil, fmt.Errorf("failed to execute POST request: %w", err)
	}
	return resp.RawResponse, nil
}

// PostWithAuth performs a POST request with authentication and optional JSON payload.
func (a *Adapter) PostWithAuth(
	ctx context.Context,
	url, token string,
	payload any,
) (*http.Response, error) {
	request := a.client.R().
		SetContext(ctx).
		SetAuthToken(token).
		SetDoNotParseResponse(true)

	if payload != nil {
		request.SetHeader("Content-Type", contentTypeJSON).SetBody(payload)
	}

	resp, err := request.Post(url)
	if err != nil {
		// Handle resty marshaling errors
		if strings.Contains(err.Error(), "unsupported 'Body' type/value") {
			return nil, fmt.Errorf("failed to prepare POST payload: %w", err)
		}
		return nil, fmt.Errorf("failed to execute authenticated POST request: %w", err)
	}
	return resp.RawResponse, nil
}

// SetRateLimit allows configuring the rate limiter after creation.
// Useful for different rate limits per Rancher server or API endpoint.
func (a *Adapter) SetRateLimit(requestsPerSecond float64, burst int) {
	a.limiter = rate.NewLimiter(rate.Limit(requestsPerSecond), burst)
}
