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
	for _, pattern := range f.patterns {
		if pattern.MatchString(clusterName) {
			f.logger.Debug("Cluster matches exclude pattern",
				"cluster", clusterName,
				"pattern", pattern.String())
			return true
		}
	}
	return false
}
