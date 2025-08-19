package filter

import (
	"errors"
	"fmt"
	"log/slog"
	"regexp"
)

// ExcludeFilter filters clusters based on exclude regex patterns.
type ExcludeFilter struct {
	patterns []*regexp.Regexp
	logger   *slog.Logger
}

// NewExcludeFilter creates a new exclude filter with the given patterns.
func NewExcludeFilter(patterns []string, logger *slog.Logger) (*ExcludeFilter, error) {
	if len(patterns) == 0 {
		return nil, errors.New("no patterns provided for exclude filter")
	}

	compiledPatterns := make([]*regexp.Regexp, 0, len(patterns))
	for _, pattern := range patterns {
		compiled, err := regexp.Compile(pattern)
		if err != nil {
			return nil, fmt.Errorf("invalid regex pattern %q: %w", pattern, err)
		}
		compiledPatterns = append(compiledPatterns, compiled)
	}

	return &ExcludeFilter{
		patterns: compiledPatterns,
		logger:   logger,
	}, nil
}

// ShouldExclude returns true if the cluster name matches any exclude pattern.
func (f *ExcludeFilter) ShouldExclude(clusterName string) bool {
	f.logger.Info("Checking cluster against exclude patterns",
		"cluster", fmt.Sprintf("%q", clusterName),
		"pattern_count", len(f.patterns))

	for i, pattern := range f.patterns {
		matches := pattern.MatchString(clusterName)
		f.logger.Info("Pattern evaluation",
			"cluster", fmt.Sprintf("%q", clusterName),
			"pattern", fmt.Sprintf("%q", pattern.String()),
			"pattern_index", i,
			"matches", matches)
		if matches {
			f.logger.Info("MATCH FOUND - cluster will be excluded",
				"cluster", fmt.Sprintf("%q", clusterName),
				"matched_pattern", fmt.Sprintf("%q", pattern.String()))
			return true
		}
	}

	f.logger.Info("No patterns matched - cluster will be included",
		"cluster", fmt.Sprintf("%q", clusterName))
	return false
}
