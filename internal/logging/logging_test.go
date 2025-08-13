package logging

import (
	"testing"
)

func TestGet(t *testing.T) {
	logger := Get()
	if logger == nil {
		t.Fatal("Expected Get() to return a logger, got nil")
	}
}

func TestSetVerbose(t *testing.T) {
	// Test that debug messages are hidden by default
	SetVerbose(false)
	logger := Get()

	// Note: We can't easily test log levels without more complex setup
	// This is a basic smoke test
	logger.Debug("This is a debug message")
	logger.Info("This is an info message")

	// Enable verbose mode
	SetVerbose(true)
	logger = Get()
	logger.Debug("This is another debug message")

	// Just ensure no panic occurs - this is a smoke test
}

func TestLoggerWith(t *testing.T) {
	logger := Get()

	// Test adding fields with With()
	serverLogger := logger.With("server_url", "https://example.com", "auth_type", "local")
	if serverLogger == nil {
		t.Fatal("Expected With() to return a logger, got nil")
	}

	// Test chaining With() calls
	clusterLogger := serverLogger.With("cluster_id", "c-123", "cluster_name", "test")
	if clusterLogger == nil {
		t.Fatal("Expected chained With() to return a logger, got nil")
	}
}

func TestDefault(t *testing.T) {
	// Test that Default() returns the same logger as Get()
	logger1 := Get()
	logger2 := Default()

	if logger1 != logger2 {
		t.Error("Expected Default() and Get() to return the same logger instance")
	}
}

func TestLoggingMethods(t *testing.T) {
	logger := Get()

	// Smoke test all logging methods to ensure they don't panic
	logger.Debug("debug message", "key", "value")
	logger.Info("info message", "key", "value")
	logger.Warn("warn message", "key", "value")
	logger.Error("error message", "key", "value")

	// Test with structured fields
	logger.With("field1", "value1").Info("message with fields")
}
