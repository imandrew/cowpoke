// Package testutil provides test utilities and constructors with pre-injected dependencies.
package testutil

import (
	"log/slog"

	"cowpoke/internal/logging"
)

// Logger returns a test logger for use in tests.
func Logger() *slog.Logger {
	return logging.NewTestLogger()
}
