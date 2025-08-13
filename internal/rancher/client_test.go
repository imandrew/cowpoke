package rancher

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"cowpoke/internal/config"
)

// Client Tests

func TestNewClient(t *testing.T) {
	server := config.RancherServer{
		ID:       "test-id",
		Name:     "Test Server",
		URL:      "https://rancher.example.com",
		Username: "admin",
		AuthType: "local",
	}

	client := NewClient(server)
	if client == nil {
		t.Fatal("Expected client to be created, got nil")
	}

	if client.server.URL != server.URL {
		t.Errorf("Expected URL %s, got %s", server.URL, client.server.URL)
	}

	if client.server.Username != server.Username {
		t.Errorf("Expected username %s, got %s", server.Username, client.server.Username)
	}

	if client.server.AuthType != server.AuthType {
		t.Errorf("Expected auth type %s, got %s", server.AuthType, client.server.AuthType)
	}

	if client.httpClient == nil {
		t.Error("Expected HTTP client to be initialized")
	}

	if client.httpClient.Timeout != 30*time.Second {
		t.Errorf("Expected timeout 30s, got %v", client.httpClient.Timeout)
	}

	if client.token != "" {
		t.Error("Expected token to be empty initially")
	}
}


// Authentication Tests

func TestClient_Authenticate_Success(t *testing.T) {
	mockResponse := `{
		"token": "token-12345",
		"type": "token",
		"user": "admin"
	}`

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v3-public/localProviders/local" {
			t.Errorf("Expected path /v3-public/localProviders/local, got %s", r.URL.Path)
		}

		if r.URL.Query().Get("action") != "login" {
			t.Errorf("Expected action=login query parameter, got %s", r.URL.Query().Get("action"))
		}

		if r.Method != "POST" {
			t.Errorf("Expected POST request, got %s", r.Method)
		}

		if r.Header.Get("Content-Type") != "application/json" {
			t.Errorf("Expected Content-Type application/json, got %s", r.Header.Get("Content-Type"))
		}

		// Verify request body
		var authReq authRequest
		err := json.NewDecoder(r.Body).Decode(&authReq)
		if err != nil {
			t.Errorf("Failed to decode request body: %v", err)
		}

		if authReq.Username != "admin" {
			t.Errorf("Expected username 'admin', got %s", authReq.Username)
		}

		if authReq.Password != "password123" {
			t.Errorf("Expected password 'password123', got %s", authReq.Password)
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(mockResponse))
	}))
	defer server.Close()

	rancherServer := config.RancherServer{
		ID:       "test-id",
		Name:     "Test Server",
		URL:      server.URL,
		Username: "admin",
		AuthType: "local",
	}

	client := NewClient(rancherServer)
	token, err := client.Authenticate(context.Background(), "password123")

	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	if token != "token-12345" {
		t.Errorf("Expected token 'token-12345', got: %s", token)
	}

	if client.token != "token-12345" {
		t.Errorf("Expected client token to be set to 'token-12345', got: %s", client.token)
	}
}

func TestClient_Authenticate_WrongCredentials(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		_, _ = w.Write([]byte(`{"message": "Unauthorized"}`))
	}))
	defer server.Close()

	rancherServer := config.RancherServer{
		URL:      server.URL,
		Username: "admin",
		AuthType: "local",
	}

	client := NewClient(rancherServer)
	_, err := client.Authenticate(context.Background(), "wrongpassword")

	if err == nil {
		t.Error("Expected error for wrong credentials")
	}

	// Should be an authentication error
	if !strings.Contains(err.Error(), "authentication failed") {
		t.Errorf("Expected authentication error, got: %v", err)
	}
}

func TestClient_Authenticate_InvalidJSON(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`invalid json`))
	}))
	defer server.Close()

	rancherServer := config.RancherServer{
		URL:      server.URL,
		Username: "admin",
		AuthType: "local",
	}

	client := NewClient(rancherServer)
	_, err := client.Authenticate(context.Background(), "password")

	if err == nil {
		t.Error("Expected error for invalid JSON response")
	}

	if !strings.Contains(err.Error(), "authentication failed") {
		t.Errorf("Expected authentication error, got: %v", err)
	}
}

func TestClient_Authenticate_NetworkError(t *testing.T) {
	// Use a server that doesn't exist
	rancherServer := config.RancherServer{
		URL:      "http://localhost:99999", // Non-existent port
		Username: "admin",
		AuthType: "local",
	}

	client := NewClient(rancherServer)
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	_, err := client.Authenticate(ctx, "password")

	if err == nil {
		t.Error("Expected error for network failure")
	}

	// Should be an authentication error wrapping the network error
	if !strings.Contains(err.Error(), "authentication failed") {
		t.Errorf("Expected authentication error, got: %v", err)
	}
}

func TestClient_Authenticate_ContextCancellation(t *testing.T) {
	// Create a server that delays response
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(200 * time.Millisecond)
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"token": "token-12345"}`))
	}))
	defer server.Close()

	rancherServer := config.RancherServer{
		URL:      server.URL,
		Username: "admin",
		AuthType: "local",
	}

	client := NewClient(rancherServer)
	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()

	_, err := client.Authenticate(ctx, "password")

	if err == nil {
		t.Error("Expected error for context cancellation")
	}
}

// API Tests

func TestClient_GetClusters_Success(t *testing.T) {
	mockResponse := `{
		"data": [
			{
				"id": "c-cluster1",
				"name": "production-cluster",
				"type": "cluster"
			},
			{
				"id": "c-cluster2",
				"name": "staging-cluster",
				"type": "cluster"
			}
		]
	}`

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v3/clusters" {
			t.Errorf("Expected path /v3/clusters, got %s", r.URL.Path)
		}

		if r.Method != "GET" {
			t.Errorf("Expected GET request, got %s", r.Method)
		}

		authHeader := r.Header.Get("Authorization")
		if authHeader != "Bearer token-12345" {
			t.Errorf("Expected Authorization header 'Bearer token-12345', got: %s", authHeader)
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(mockResponse))
	}))
	defer server.Close()

	rancherServer := config.RancherServer{
		ID:   "server-1",
		Name: "Test Server",
		URL:  server.URL,
	}

	client := NewClient(rancherServer)
	client.token = "token-12345" // Set token directly for testing

	clusters, err := client.GetClusters(context.Background())

	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	if len(clusters) != 2 {
		t.Errorf("Expected 2 clusters, got: %d", len(clusters))
	}

	expectedCluster1 := config.Cluster{
		ID:         "c-cluster1",
		Name:       "production-cluster",
		ServerID:   "server-1",
		ServerName: "Test Server",
	}

	if clusters[0].ID != expectedCluster1.ID {
		t.Errorf("Expected cluster ID %s, got: %s", expectedCluster1.ID, clusters[0].ID)
	}

	if clusters[0].Name != expectedCluster1.Name {
		t.Errorf("Expected cluster name %s, got: %s", expectedCluster1.Name, clusters[0].Name)
	}

	if clusters[0].ServerID != expectedCluster1.ServerID {
		t.Errorf("Expected server ID %s, got: %s", expectedCluster1.ServerID, clusters[0].ServerID)
	}

	if clusters[0].ServerName != expectedCluster1.ServerName {
		t.Errorf("Expected server name %s, got: %s", expectedCluster1.ServerName, clusters[0].ServerName)
	}
}

func TestClient_GetClusters_Unauthorized(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		_, _ = w.Write([]byte(`{"message": "Unauthorized"}`))
	}))
	defer server.Close()

	rancherServer := config.RancherServer{
		URL: server.URL,
	}

	client := NewClient(rancherServer)
	client.token = "invalid-token"

	_, err := client.GetClusters(context.Background())

	if err == nil {
		t.Error("Expected error for unauthorized request")
	}
}

func TestClient_GetClusters_NotAuthenticated(t *testing.T) {
	rancherServer := config.RancherServer{
		URL:      "https://rancher.example.com",
		Username: "admin",
		AuthType: "local",
	}

	client := NewClient(rancherServer)
	// Don't set token

	_, err := client.GetClusters(context.Background())

	if err == nil {
		t.Error("Expected error for unauthenticated client")
	}

	if !strings.Contains(err.Error(), "authentication failed") {
		t.Errorf("Expected authentication error, got: %v", err)
	}
}

func TestClient_GetClusters_InvalidJSON(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`invalid json`))
	}))
	defer server.Close()

	rancherServer := config.RancherServer{
		URL: server.URL,
	}

	client := NewClient(rancherServer)
	client.token = "token-12345"

	_, err := client.GetClusters(context.Background())

	if err == nil {
		t.Error("Expected error for invalid JSON")
	}

	if !strings.Contains(err.Error(), "failed to get clusters") {
		t.Errorf("Expected clusters error, got: %v", err)
	}
}

func TestClient_GetClusters_EmptyResponse(t *testing.T) {
	mockResponse := `{"data": []}`

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(mockResponse))
	}))
	defer server.Close()

	rancherServer := config.RancherServer{
		ID:   "server-1",
		Name: "Test Server",
		URL:  server.URL,
	}

	client := NewClient(rancherServer)
	client.token = "token-12345"

	clusters, err := client.GetClusters(context.Background())

	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	if len(clusters) != 0 {
		t.Errorf("Expected 0 clusters, got: %d", len(clusters))
	}
}

// Kubeconfig Tests

func TestClient_GetKubeconfig_Success(t *testing.T) {
	mockKubeconfig := `apiVersion: v1
kind: Config
clusters:
- cluster:
    server: https://test.example.com
  name: test-cluster
contexts:
- context:
    cluster: test-cluster
    user: test-user
  name: test-context
current-context: test-context
users:
- name: test-user
  user:
    token: test-token`

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v3/clusters/c-test" {
			t.Errorf("Expected path /v3/clusters/c-test, got %s", r.URL.Path)
		}
		
		if r.URL.Query().Get("action") != "generateKubeconfig" {
			t.Errorf("Expected action=generateKubeconfig query parameter, got %s", r.URL.Query().Get("action"))
		}

		if r.Method != "POST" {
			t.Errorf("Expected POST request, got %s", r.Method)
		}

		authHeader := r.Header.Get("Authorization")
		if authHeader != "Bearer token-12345" {
			t.Errorf("Expected Authorization header 'Bearer token-12345', got: %s", authHeader)
		}

		response := struct {
			Config string `json:"config"`
		}{
			Config: mockKubeconfig,
		}
		
		jsonResp, _ := json.Marshal(response)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(jsonResp)
	}))
	defer server.Close()

	rancherServer := config.RancherServer{
		URL: server.URL,
	}

	client := NewClient(rancherServer)
	client.token = "token-12345"

	kubeconfig, err := client.GetKubeconfig(context.Background(), "c-test")

	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	if string(kubeconfig) != mockKubeconfig {
		t.Errorf("Expected kubeconfig to match mock, got: %s", string(kubeconfig))
	}
}

func TestClient_GetKubeconfig_Unauthorized(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		_, _ = w.Write([]byte(`{"message": "Unauthorized"}`))
	}))
	defer server.Close()

	rancherServer := config.RancherServer{
		URL: server.URL,
	}

	client := NewClient(rancherServer)
	client.token = "invalid-token"

	_, err := client.GetKubeconfig(context.Background(), "c-test")

	if err == nil {
		t.Error("Expected error for unauthorized request")
	}
}

func TestClient_GetKubeconfig_ClusterNotFound(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		_, _ = w.Write([]byte(`{"message": "Cluster not found"}`))
	}))
	defer server.Close()

	rancherServer := config.RancherServer{
		URL: server.URL,
	}

	client := NewClient(rancherServer)
	client.token = "token-12345"

	_, err := client.GetKubeconfig(context.Background(), "non-existent-cluster")

	if err == nil {
		t.Error("Expected error for non-existent cluster")
	}
}

func TestClient_GetKubeconfig_NotAuthenticated(t *testing.T) {
	rancherServer := config.RancherServer{
		URL:      "https://rancher.example.com",
		Username: "admin",
		AuthType: "local",
	}

	client := NewClient(rancherServer)
	// Don't set token

	_, err := client.GetKubeconfig(context.Background(), "c-test")

	if err == nil {
		t.Error("Expected error for unauthenticated client")
	}

	if !strings.Contains(err.Error(), "authentication failed") {
		t.Errorf("Expected authentication error, got: %v", err)
	}
}

func TestClient_GetKubeconfig_InvalidJSON(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`invalid json`))
	}))
	defer server.Close()

	rancherServer := config.RancherServer{
		URL: server.URL,
	}

	client := NewClient(rancherServer)
	client.token = "token-12345"

	_, err := client.GetKubeconfig(context.Background(), "c-test")

	if err == nil {
		t.Error("Expected error for invalid JSON")
	}

	if !strings.Contains(err.Error(), "failed to get kubeconfig") {
		t.Errorf("Expected kubeconfig error, got: %v", err)
	}
}

// HTTP Request Tests

func TestClient_makeJSONRequest_GET(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "GET" {
			t.Errorf("Expected GET request, got %s", r.Method)
		}

		if r.Header.Get("Content-Type") != "application/json" {
			t.Errorf("Expected Content-Type application/json, got %s", r.Header.Get("Content-Type"))
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"status": "ok"}`))
	}))
	defer server.Close()

	client := NewClient(config.RancherServer{URL: server.URL})
	
	body, err := client.makeJSONRequest(context.Background(), "GET", server.URL+"/test", nil)

	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	expected := `{"status": "ok"}`
	if string(body) != expected {
		t.Errorf("Expected response %s, got %s", expected, string(body))
	}
}

func TestClient_makeJSONRequest_POST_WithPayload(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" {
			t.Errorf("Expected POST request, got %s", r.Method)
		}

		// Verify payload
		var payload map[string]string
		err := json.NewDecoder(r.Body).Decode(&payload)
		if err != nil {
			t.Errorf("Failed to decode request body: %v", err)
		}

		if payload["test"] != "value" {
			t.Errorf("Expected payload test=value, got %s", payload["test"])
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"received": "ok"}`))
	}))
	defer server.Close()

	client := NewClient(config.RancherServer{URL: server.URL})
	payload := map[string]string{"test": "value"}
	
	body, err := client.makeJSONRequest(context.Background(), "POST", server.URL+"/test", payload)

	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	expected := `{"received": "ok"}`
	if string(body) != expected {
		t.Errorf("Expected response %s, got %s", expected, string(body))
	}
}

func TestClient_makeJSONRequest_WithAuthToken(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		authHeader := r.Header.Get("Authorization")
		if authHeader != "Bearer test-token" {
			t.Errorf("Expected Authorization header 'Bearer test-token', got: %s", authHeader)
		}

		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"authenticated": true}`))
	}))
	defer server.Close()

	client := NewClient(config.RancherServer{URL: server.URL})
	client.token = "test-token"
	
	_, err := client.makeJSONRequest(context.Background(), "GET", server.URL+"/test", nil)

	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}
}

func TestClient_makeJSONRequest_HTTPError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte(`{"error": "server error"}`))
	}))
	defer server.Close()

	client := NewClient(config.RancherServer{URL: server.URL})
	
	_, err := client.makeJSONRequest(context.Background(), "GET", server.URL+"/test", nil)

	if err == nil {
		t.Error("Expected error for HTTP 500")
	}

	// Should be an HTTP error
	if !strings.Contains(err.Error(), "HTTP 500") {
		t.Errorf("Expected HTTP error, got: %v", err)
	}
}

func TestClient_makeJSONRequest_InvalidURL(t *testing.T) {
	client := NewClient(config.RancherServer{})
	
	_, err := client.makeJSONRequest(context.Background(), "GET", "://invalid-url", nil)

	if err == nil {
		t.Error("Expected error for invalid URL")
	}

	if !strings.Contains(err.Error(), "failed to create request") {
		t.Errorf("Expected request creation error, got: %v", err)
	}
}

func TestClient_makeJSONRequest_InvalidPayload(t *testing.T) {
	client := NewClient(config.RancherServer{})
	
	// Use a channel as payload (not JSON serializable)
	invalidPayload := make(chan int)
	
	_, err := client.makeJSONRequest(context.Background(), "POST", "http://example.com", invalidPayload)

	if err == nil {
		t.Error("Expected error for invalid payload")
	}

	if !strings.Contains(err.Error(), "failed to marshal request") {
		t.Errorf("Expected marshal error, got: %v", err)
	}
}