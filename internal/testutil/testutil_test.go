package testutil

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestLogger(t *testing.T) {
	logger := Logger()
	require.NotNil(t, logger)
}
