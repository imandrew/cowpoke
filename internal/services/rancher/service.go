package rancher

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"

	"cowpoke/internal/domain"
)

// Service handles Rancher API operations.
type Service struct {
	httpAdapter domain.HTTPAdapter
	logger      *slog.Logger
}

// NewClusterLister creates a new cluster lister service.
func NewClusterLister(httpAdapter domain.HTTPAdapter, logger *slog.Logger) *Service {
	return &Service{
		httpAdapter: httpAdapter,
		logger:      logger,
	}
}

// NewKubeconfigFetcher creates a new kubeconfig fetcher service.
func NewKubeconfigFetcher(httpAdapter domain.HTTPAdapter, logger *slog.Logger) *Service {
	return &Service{
		httpAdapter: httpAdapter,
		logger:      logger,
	}
}

// ListClusters retrieves all clusters from a Rancher server.
func (s *Service) ListClusters(
	ctx context.Context,
	token domain.AuthToken,
	server domain.ConfigServer,
) ([]domain.Cluster, error) {
	clustersURL := fmt.Sprintf("%s/v3/clusters", server.URL)

	s.logger.InfoContext(ctx, "Fetching clusters from Rancher server", "server", server.URL)

	resp, err := s.httpAdapter.GetWithAuth(ctx, clustersURL, token.Value())
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
	for _, c := range clustersResp.Data {
		// Skip clusters that are not active or don't have a valid ID
		if c.ID == "" || c.Name == "" {
			s.logger.WarnContext(ctx, "Skipping invalid cluster", "id", c.ID, "name", c.Name)
			continue
		}

		clusters = append(clusters, domain.Cluster{
			ID:   c.ID,
			Name: c.Name,
			Type: c.Type,
		})
	}

	s.logger.InfoContext(ctx, "Successfully fetched clusters",
		"server", server.URL,
		"count", len(clusters))

	return clusters, nil
}

// GetKubeconfig retrieves the kubeconfig for a specific cluster.
func (s *Service) GetKubeconfig(
	ctx context.Context,
	token domain.AuthToken,
	server domain.ConfigServer,
	clusterID string,
) ([]byte, error) {
	kubeconfigURL := fmt.Sprintf(
		"%s/v3/clusters/%s?action=generateKubeconfig",
		server.URL,
		clusterID,
	)

	s.logger.InfoContext(ctx, "Fetching kubeconfig for cluster",
		"server", server.URL,
		"clusterID", clusterID)

	resp, err := s.httpAdapter.PostWithAuth(ctx, kubeconfigURL, token.Value(), nil)
	if err != nil {
		return nil, fmt.Errorf("failed to get kubeconfig: %w", err)
	}
	defer resp.Body.Close()

	// Accept both 200 (OK) and 201 (Created) for generateKubeconfig action
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
		return nil, errors.New("kubeconfig response was empty")
	}

	s.logger.InfoContext(ctx, "Successfully fetched kubeconfig",
		"server", server.URL,
		"clusterID", clusterID)

	return []byte(kubeconfigResp.Config), nil
}

// clustersResponse represents the Rancher clusters API response.
type clustersResponse struct {
	Data []clusterData `json:"data"`
}

// clusterData represents a single cluster in the response.
type clusterData struct {
	ID   string `json:"id"`
	Name string `json:"name"`
	Type string `json:"type"`
}

// kubeconfigResponse represents the Rancher kubeconfig API response.
type kubeconfigResponse struct {
	Config string `json:"config"`
}
