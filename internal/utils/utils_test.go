//nolint:revive // Test file follows main package naming
package utils

import (
	"strings"
	"testing"
)

// Path utility tests

func TestGetConfigPath(t *testing.T) {
	path, err := GetConfigPath()
	if err != nil {
		t.Fatalf("GetConfigPath failed: %v", err)
	}

	if !strings.HasSuffix(path, ".config/cowpoke/config.yaml") {
		t.Errorf("Expected path to end with .config/cowpoke/config.yaml, got: %s", path)
	}
}

func TestGetKubeconfigDir(t *testing.T) {
	dir, err := GetKubeconfigDir()
	if err != nil {
		t.Fatalf("GetKubeconfigDir failed: %v", err)
	}

	if !strings.HasSuffix(dir, ".config/cowpoke/kubeconfigs") {
		t.Errorf("Expected dir to end with .config/cowpoke/kubeconfigs, got: %s", dir)
	}
}

func TestGetDefaultKubeconfigPath(t *testing.T) {
	path, err := GetDefaultKubeconfigPath()
	if err != nil {
		t.Fatalf("GetDefaultKubeconfigPath failed: %v", err)
	}

	if !strings.HasSuffix(path, ".kube/config") {
		t.Errorf("Expected path to end with .kube/config, got: %s", path)
	}
}

func TestGetConfigManager(t *testing.T) {
	manager, err := GetConfigManager()
	if err != nil {
		t.Fatalf("GetConfigManager failed: %v", err)
	}

	if manager == nil {
		t.Error("Expected manager to be non-nil")
	}
}

func TestPathsWithInvalidHome(t *testing.T) {
	// Set invalid HOME
	t.Setenv("HOME", "")

	_, err := GetConfigPath()
	if err == nil {
		t.Error("Expected error when HOME is not set")
	}

	_, err = GetKubeconfigDir()
	if err == nil {
		t.Error("Expected error when HOME is not set")
	}

	_, err = GetDefaultKubeconfigPath()
	if err == nil {
		t.Error("Expected error when HOME is not set")
	}

	_, err = GetConfigManager()
	if err == nil {
		t.Error("Expected error when HOME is not set")
	}
}

// Filename sanitization tests

func TestSanitizeFilename(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"normal-filename", "normal-filename"},
		{"file:with*invalid?chars", "file_with_invalid_chars"},
		{"file/with\\path|separators", "file_with_path_separators"},
		{"file<with>quotes\"", "file_with_quotes"},
		{"___multiple___underscores___", "multiple_underscores"},
		{"", "unnamed"},
		{"____", "unnamed"},
		{"cluster:name/with*many?issues", "cluster_name_with_many_issues"},
	}

	for _, test := range tests {
		result := SanitizeFilename(test.input)
		if result != test.expected {
			t.Errorf("SanitizeFilename(%q) = %q; expected %q", test.input, result, test.expected)
		}
	}
}

func TestSanitizeURL(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"https://rancher.example.com", "rancher-example-com"},
		{"http://rancher.example.com", "rancher-example-com"},
		{"https://rancher.example.com:8443", "rancher-example-com"},
		{"https://rancher.example.com/path/to/resource", "rancher-example-com"},
		{"https://my-rancher.staging.company.io", "my-rancher-staging-company-io"},
		{"localhost:3000", "localhost"},
		{"", "unknown-server"},
		{"https://", "unknown-server"},
		{"invalid-url-with-special!@#chars", "invalid-url-with-special-chars"},
	}

	for _, test := range tests {
		result := SanitizeURL(test.input)
		if result != test.expected {
			t.Errorf("SanitizeURL(%q) = %q; expected %q", test.input, result, test.expected)
		}
	}
}

// Input/Terminal utility tests

func TestIsTerminal(t *testing.T) {
	// Test with a file descriptor that's definitely not a terminal
	// We can't easily test with a real terminal in unit tests
	result := IsTerminal(-1)
	if result {
		t.Error("Expected IsTerminal(-1) to return false")
	}
}

func TestCanPromptForPassword(_ *testing.T) {
	// This test just ensures the function doesn't panic
	// The actual terminal detection depends on the environment
	result := CanPromptForPassword()
	// Don't assert on the result since it depends on how tests are run
	_ = result
}
