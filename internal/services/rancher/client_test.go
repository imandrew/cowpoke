package rancher

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNormalizeURL(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "URL with trailing slash",
			input:    "https://rancher.example.com/",
			expected: "https://rancher.example.com",
		},
		{
			name:     "URL without trailing slash",
			input:    "https://rancher.example.com",
			expected: "https://rancher.example.com",
		},
		{
			name:     "URL with multiple trailing slashes",
			input:    "https://rancher.example.com///",
			expected: "https://rancher.example.com//",
		},
		{
			name:     "URL with port and trailing slash",
			input:    "https://rancher.example.com:8443/",
			expected: "https://rancher.example.com:8443",
		},
		{
			name:     "URL with path and trailing slash",
			input:    "https://rancher.example.com/rancher/",
			expected: "https://rancher.example.com/rancher",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := normalizeURL(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}
