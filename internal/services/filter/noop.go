package filter

// NoOpFilter is a filter that never excludes any clusters.
type NoOpFilter struct{}

// NewNoOpFilter creates a new no-op filter.
func NewNoOpFilter() *NoOpFilter {
	return &NoOpFilter{}
}

// ShouldExclude always returns false, never excluding any clusters.
func (f *NoOpFilter) ShouldExclude(_ string) bool {
	return false
}
