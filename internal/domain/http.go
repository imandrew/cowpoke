package domain

import (
	"context"
	"net/http"
)

// HTTPAdapter defines the interface for HTTP operations.
type HTTPAdapter interface {
	Get(ctx context.Context, url string) (*http.Response, error)
	GetWithAuth(ctx context.Context, url, token string) (*http.Response, error)
	Post(ctx context.Context, url string, payload any) (*http.Response, error)
	PostWithAuth(
		ctx context.Context,
		url, token string,
		payload any,
	) (*http.Response, error)
}
