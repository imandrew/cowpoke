package filter

import (
	"testing"

	"cowpoke/internal/testutil"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewExcludeFilter_Success(t *testing.T) {
	tests := []struct {
		name     string
		patterns []string
	}{
		{
			name:     "single pattern",
			patterns: []string{"^test-.*"},
		},
		{
			name:     "multiple patterns",
			patterns: []string{"^test-.*", ".*-staging$", "dev-.*"},
		},
		{
			name:     "complex regex patterns",
			patterns: []string{"^(test|dev)-.*", ".*-(staging|qa)$", "^temp\\d+$"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Act
			filter, err := NewExcludeFilter(tt.patterns, testutil.Logger())

			// Assert
			require.NoError(t, err)
			assert.NotNil(t, filter)
			assert.Len(t, filter.patterns, len(tt.patterns))
		})
	}
}

func TestNewExcludeFilter_EmptyPatterns(t *testing.T) {
	// Act
	filter, err := NewExcludeFilter([]string{}, testutil.Logger())

	// Assert
	require.Error(t, err)
	assert.Nil(t, filter)
	assert.Contains(t, err.Error(), "no patterns provided for exclude filter")
}

func TestNewExcludeFilter_NilPatterns(t *testing.T) {
	// Act
	filter, err := NewExcludeFilter(nil, testutil.Logger())

	// Assert
	require.Error(t, err)
	assert.Nil(t, filter)
	assert.Contains(t, err.Error(), "no patterns provided for exclude filter")
}

func TestNewExcludeFilter_InvalidRegex(t *testing.T) {
	tests := []struct {
		name     string
		patterns []string
		errorMsg string
	}{
		{
			name:     "invalid bracket",
			patterns: []string{"[invalid"},
			errorMsg: "invalid regex pattern",
		},
		{
			name:     "invalid escape",
			patterns: []string{"\\"},
			errorMsg: "invalid regex pattern",
		},
		{
			name:     "invalid quantifier",
			patterns: []string{"*invalid"},
			errorMsg: "invalid regex pattern",
		},
		{
			name:     "mixed valid and invalid",
			patterns: []string{"^valid-.*", "[invalid"},
			errorMsg: "invalid regex pattern",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Act
			filter, err := NewExcludeFilter(tt.patterns, testutil.Logger())

			// Assert
			require.Error(t, err)
			assert.Nil(t, filter)
			assert.Contains(t, err.Error(), tt.errorMsg)
		})
	}
}

func TestExcludeFilter_ShouldExclude_SinglePattern(t *testing.T) {
	// Arrange
	filter, err := NewExcludeFilter([]string{"^test-.*"}, testutil.Logger())
	require.NoError(t, err)

	tests := []struct {
		name        string
		clusterName string
		shouldMatch bool
	}{
		{
			name:        "matches pattern",
			clusterName: "test-cluster",
			shouldMatch: true,
		},
		{
			name:        "matches pattern with suffix",
			clusterName: "test-production",
			shouldMatch: true,
		},
		{
			name:        "does not match pattern",
			clusterName: "production",
			shouldMatch: false,
		},
		{
			name:        "does not match partial",
			clusterName: "mytest-cluster",
			shouldMatch: false,
		},
		{
			name:        "empty string",
			clusterName: "",
			shouldMatch: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Act
			result := filter.ShouldExclude(tt.clusterName)

			// Assert
			assert.Equal(t, tt.shouldMatch, result)
		})
	}
}

func TestExcludeFilter_ShouldExclude_MultiplePatterns(t *testing.T) {
	// Arrange
	patterns := []string{"^test-.*", ".*-staging$", "^dev\\d+$"}
	filter, err := NewExcludeFilter(patterns, testutil.Logger())
	require.NoError(t, err)

	tests := []struct {
		name        string
		clusterName string
		shouldMatch bool
		reason      string
	}{
		{
			name:        "matches first pattern",
			clusterName: "test-cluster",
			shouldMatch: true,
			reason:      "starts with test-",
		},
		{
			name:        "matches second pattern",
			clusterName: "prod-staging",
			shouldMatch: true,
			reason:      "ends with -staging",
		},
		{
			name:        "matches third pattern",
			clusterName: "dev1",
			shouldMatch: true,
			reason:      "dev followed by digits",
		},
		{
			name:        "matches multiple patterns",
			clusterName: "test-env-staging",
			shouldMatch: true,
			reason:      "matches both first and second pattern",
		},
		{
			name:        "does not match any pattern",
			clusterName: "production",
			shouldMatch: false,
			reason:      "doesn't match any pattern",
		},
		{
			name:        "partial match does not count",
			clusterName: "devserver",
			shouldMatch: false,
			reason:      "dev not followed by digits",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Act
			result := filter.ShouldExclude(tt.clusterName)

			// Assert
			assert.Equal(t, tt.shouldMatch, result, tt.reason)
		})
	}
}

func TestExcludeFilter_ShouldExclude_CaseSensitive(t *testing.T) {
	// Arrange
	filter, err := NewExcludeFilter([]string{"^Test-.*"}, testutil.Logger())
	require.NoError(t, err)

	tests := []struct {
		name        string
		clusterName string
		shouldMatch bool
	}{
		{
			name:        "exact case match",
			clusterName: "Test-cluster",
			shouldMatch: true,
		},
		{
			name:        "lowercase does not match",
			clusterName: "test-cluster",
			shouldMatch: false,
		},
		{
			name:        "uppercase does not match",
			clusterName: "TEST-cluster",
			shouldMatch: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Act
			result := filter.ShouldExclude(tt.clusterName)

			// Assert
			assert.Equal(t, tt.shouldMatch, result)
		})
	}
}

func TestExcludeFilter_ShouldExclude_ComplexPatterns(t *testing.T) {
	// Test real-world complex regex patterns

	// Arrange
	patterns := []string{
		"^(test|dev|staging)-.*",     // Starts with test-, dev-, or staging-
		".*-(temp|experimental)$",    // Ends with -temp or -experimental
		"^cluster-[0-9]{3,}$",        // cluster- followed by 3+ digits
		".*\\b(old|deprecated)\\b.*", // Contains word "old" or "deprecated"
	}
	filter, err := NewExcludeFilter(patterns, testutil.Logger())
	require.NoError(t, err)

	tests := []struct {
		name        string
		clusterName string
		shouldMatch bool
		reason      string
	}{
		{
			name:        "test environment",
			clusterName: "test-environment-1",
			shouldMatch: true,
			reason:      "starts with test-",
		},
		{
			name:        "dev cluster",
			clusterName: "dev-cluster",
			shouldMatch: true,
			reason:      "starts with dev-",
		},
		{
			name:        "temporary cluster",
			clusterName: "production-temp",
			shouldMatch: true,
			reason:      "ends with -temp",
		},
		{
			name:        "numbered cluster",
			clusterName: "cluster-123",
			shouldMatch: true,
			reason:      "cluster- with 3 digits",
		},
		{
			name:        "short numbered cluster not matched",
			clusterName: "cluster-12",
			shouldMatch: false,
			reason:      "only 2 digits, needs 3+",
		},
		{
			name:        "deprecated cluster",
			clusterName: "my-deprecated-cluster",
			shouldMatch: true,
			reason:      "contains word deprecated",
		},
		{
			name:        "old cluster",
			clusterName: "cluster-old-version",
			shouldMatch: true,
			reason:      "contains word old",
		},
		{
			name:        "production cluster",
			clusterName: "production-main",
			shouldMatch: false,
			reason:      "doesn't match any pattern",
		},
		{
			name:        "partial word match not counted",
			clusterName: "bold-cluster",
			shouldMatch: false,
			reason:      "bold contains 'old' but not as separate word",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Act
			result := filter.ShouldExclude(tt.clusterName)

			// Assert
			assert.Equal(t, tt.shouldMatch, result, tt.reason)
		})
	}
}

func TestExcludeFilter_ShouldExclude_EdgeCases(t *testing.T) {
	// Arrange
	filter, err := NewExcludeFilter([]string{".*"}, testutil.Logger()) // Match everything
	require.NoError(t, err)

	tests := []struct {
		name        string
		clusterName string
		shouldMatch bool
	}{
		{
			name:        "normal cluster name",
			clusterName: "production",
			shouldMatch: true,
		},
		{
			name:        "empty string",
			clusterName: "",
			shouldMatch: true,
		},
		{
			name:        "special characters",
			clusterName: "cluster-with-dashes_and_underscores.and.dots",
			shouldMatch: true,
		},
		{
			name:        "numbers",
			clusterName: "123456",
			shouldMatch: true,
		},
		{
			name:        "unicode characters",
			clusterName: "クラスター",
			shouldMatch: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Act
			result := filter.ShouldExclude(tt.clusterName)

			// Assert
			assert.Equal(t, tt.shouldMatch, result)
		})
	}
}

func TestExcludeFilter_ShouldExclude_NoMatches(t *testing.T) {
	// Test filter that shouldn't match anything realistic

	// Arrange
	filter, err := NewExcludeFilter([]string{"^$"}, testutil.Logger()) // Only matches empty string
	require.NoError(t, err)

	tests := []string{
		"production",
		"test-cluster",
		"dev-environment",
		"staging-app",
		"cluster-123",
		"a", // single character
		" ", // single space
	}

	for _, clusterName := range tests {
		t.Run("cluster_"+clusterName, func(t *testing.T) {
			// Act
			result := filter.ShouldExclude(clusterName)

			// Assert
			assert.False(t, result, "cluster %q should not be excluded", clusterName)
		})
	}

	// Empty string should match
	assert.True(t, filter.ShouldExclude(""))
}
