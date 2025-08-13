package logging

import (
	"bytes"
	"context"
	"strings"
	"testing"
)

func TestNewLogger(t *testing.T) {
	var buf bytes.Buffer
	config := Config{
		Level:  LevelDebug,
		Format: "json",
		Output: &buf,
	}

	logger := NewLogger(config)
	if logger == nil {
		t.Fatal("Expected logger to be created, got nil")
	}

	// Test that logger works
	logger.Info("test message", "key", "value")
	output := buf.String()

	if !strings.Contains(output, "test message") {
		t.Errorf("Expected output to contain 'test message', got: %s", output)
	}
	if !strings.Contains(output, "key") {
		t.Errorf("Expected output to contain 'key', got: %s", output)
	}
	if !strings.Contains(output, "value") {
		t.Errorf("Expected output to contain 'value', got: %s", output)
	}
}

func TestDefaultConfig(t *testing.T) {
	config := DefaultConfig()
	
	if config.Level != LevelInfo {
		t.Errorf("Expected default level to be %s, got: %s", LevelInfo, config.Level)
	}
	if config.Format != "text" {
		t.Errorf("Expected default format to be 'text', got: %s", config.Format)
	}
	if config.Output == nil {
		t.Error("Expected default output to be non-nil")
	}
}

func TestLoggerWithFields(t *testing.T) {
	var buf bytes.Buffer
	config := Config{
		Level:  LevelInfo,
		Format: "text",
		Output: &buf,
	}

	logger := NewLogger(config)
	
	fields := map[string]any{
		"server_url": "https://rancher.example.com",
		"auth_type":  "local",
	}
	
	loggerWithFields := logger.WithFields(fields)
	loggerWithFields.Info("test message")
	
	output := buf.String()
	if !strings.Contains(output, "test message") {
		t.Errorf("Expected output to contain 'test message', got: %s", output)
	}
	if !strings.Contains(output, "https://rancher.example.com") {
		t.Errorf("Expected output to contain server URL, got: %s", output)
	}
	if !strings.Contains(output, "local") {
		t.Errorf("Expected output to contain auth type, got: %s", output)
	}
}

func TestLoggerWithServer(t *testing.T) {
	var buf bytes.Buffer
	config := Config{
		Level:  LevelInfo,
		Format: "text",
		Output: &buf,
	}

	logger := NewLogger(config)
	serverLogger := logger.WithServer("https://rancher.example.com", "github")
	serverLogger.Info("connecting to server")
	
	output := buf.String()
	if !strings.Contains(output, "connecting to server") {
		t.Errorf("Expected output to contain message, got: %s", output)
	}
	if !strings.Contains(output, "https://rancher.example.com") {
		t.Errorf("Expected output to contain server URL, got: %s", output)
	}
	if !strings.Contains(output, "github") {
		t.Errorf("Expected output to contain auth type, got: %s", output)
	}
}

func TestLoggerWithCluster(t *testing.T) {
	var buf bytes.Buffer
	config := Config{
		Level:  LevelInfo,
		Format: "text",
		Output: &buf,
	}

	logger := NewLogger(config)
	clusterLogger := logger.WithCluster("c-cluster-123", "test-cluster")
	clusterLogger.Info("processing cluster")
	
	output := buf.String()
	if !strings.Contains(output, "processing cluster") {
		t.Errorf("Expected output to contain message, got: %s", output)
	}
	if !strings.Contains(output, "c-cluster-123") {
		t.Errorf("Expected output to contain cluster ID, got: %s", output)
	}
	if !strings.Contains(output, "test-cluster") {
		t.Errorf("Expected output to contain cluster name, got: %s", output)
	}
}

func TestLoggerWithOperation(t *testing.T) {
	var buf bytes.Buffer
	config := Config{
		Level:  LevelInfo,
		Format: "text",
		Output: &buf,
	}

	logger := NewLogger(config)
	opLogger := logger.WithOperation("sync")
	opLogger.Info("starting operation")
	
	output := buf.String()
	if !strings.Contains(output, "starting operation") {
		t.Errorf("Expected output to contain message, got: %s", output)
	}
	if !strings.Contains(output, "sync") {
		t.Errorf("Expected output to contain operation, got: %s", output)
	}
}

func TestLogLevels(t *testing.T) {
	tests := []struct {
		level    LogLevel
		logFunc  func(logger *Logger, msg string)
		expected bool
	}{
		{LevelDebug, func(l *Logger, msg string) { l.Debug(msg) }, true},
		{LevelInfo, func(l *Logger, msg string) { l.Debug(msg) }, false},
		{LevelInfo, func(l *Logger, msg string) { l.Info(msg) }, true},
		{LevelWarn, func(l *Logger, msg string) { l.Info(msg) }, false},
		{LevelWarn, func(l *Logger, msg string) { l.Warn(msg) }, true},
		{LevelError, func(l *Logger, msg string) { l.Warn(msg) }, false},
		{LevelError, func(l *Logger, msg string) { l.Error(msg) }, true},
	}

	for _, test := range tests {
		var buf bytes.Buffer
		config := Config{
			Level:  test.level,
			Format: "text",
			Output: &buf,
		}

		logger := NewLogger(config)
		test.logFunc(logger, "test message")
		
		output := buf.String()
		containsMessage := strings.Contains(output, "test message")
		
		if containsMessage != test.expected {
			t.Errorf("Level %s: expected message present = %v, got = %v, output: %s", 
				test.level, test.expected, containsMessage, output)
		}
	}
}

func TestGlobalLogger(t *testing.T) {
	var buf bytes.Buffer
	config := Config{
		Level:  LevelInfo,
		Format: "text",
		Output: &buf,
	}

	// Set a custom default logger
	SetDefault(NewLogger(config))
	
	Info("global info message")
	output := buf.String()
	
	if !strings.Contains(output, "global info message") {
		t.Errorf("Expected output to contain message, got: %s", output)
	}
}

func TestContextLogging(t *testing.T) {
	var buf bytes.Buffer
	config := Config{
		Level:  LevelInfo,
		Format: "text",
		Output: &buf,
	}

	logger := NewLogger(config)
	ctx := context.Background()
	
	logger.InfoContext(ctx, "context message", "key", "value")
	output := buf.String()
	
	if !strings.Contains(output, "context message") {
		t.Errorf("Expected output to contain message, got: %s", output)
	}
	if !strings.Contains(output, "key") {
		t.Errorf("Expected output to contain key, got: %s", output)
	}
}

func TestJSONFormat(t *testing.T) {
	var buf bytes.Buffer
	config := Config{
		Level:  LevelInfo,
		Format: "json",
		Output: &buf,
	}

	logger := NewLogger(config)
	logger.Info("json message", "key", "value")
	
	output := buf.String()
	if !strings.Contains(output, `"msg":"json message"`) {
		t.Errorf("Expected JSON format output, got: %s", output)
	}
	if !strings.Contains(output, `"key":"value"`) {
		t.Errorf("Expected JSON key-value pair, got: %s", output)
	}
}

// Test missing logging methods

func TestWithContext(t *testing.T) {
	var buf bytes.Buffer
	config := Config{
		Level:  LevelInfo,
		Format: "text",
		Output: &buf,
	}

	logger := NewLogger(config)
	ctx := context.Background()
	
	contextLogger := logger.WithContext(ctx)
	if contextLogger == nil {
		t.Error("Expected WithContext to return a logger")
	}
	
	// Test that the returned logger works
	contextLogger.Info("context test message")
	output := buf.String()
	if !strings.Contains(output, "context test message") {
		t.Errorf("Expected context logger to work, got: %s", output)
	}
}

func TestDefault(t *testing.T) {
	// Set a custom default logger
	var buf bytes.Buffer
	config := Config{
		Level:  LevelInfo,
		Format: "text",
		Output: &buf,
	}
	customLogger := NewLogger(config)
	SetDefault(customLogger)
	
	// Test Default() function
	defaultLogger := Default()
	if defaultLogger == nil {
		t.Error("Expected Default() to return a logger")
	}
	
	// Test that it's the same logger we set
	defaultLogger.Info("default test message")
	output := buf.String()
	if !strings.Contains(output, "default test message") {
		t.Errorf("Expected default logger to work, got: %s", output)
	}
}

func TestDebugFunction(t *testing.T) {
	var buf bytes.Buffer
	config := Config{
		Level:  LevelDebug,
		Format: "text",
		Output: &buf,
	}
	
	logger := NewLogger(config)
	SetDefault(logger)
	
	Debug("debug message", "key", "value")
	output := buf.String()
	
	if !strings.Contains(output, "debug message") {
		t.Errorf("Expected debug message in output, got: %s", output)
	}
	if !strings.Contains(output, "key") {
		t.Errorf("Expected key in output, got: %s", output)
	}
}

func TestWarnFunction(t *testing.T) {
	var buf bytes.Buffer
	config := Config{
		Level:  LevelWarn,
		Format: "text",
		Output: &buf,
	}
	
	logger := NewLogger(config)
	SetDefault(logger)
	
	Warn("warn message", "key", "value")
	output := buf.String()
	
	if !strings.Contains(output, "warn message") {
		t.Errorf("Expected warn message in output, got: %s", output)
	}
	if !strings.Contains(output, "key") {
		t.Errorf("Expected key in output, got: %s", output)
	}
}

func TestErrorFunction(t *testing.T) {
	var buf bytes.Buffer
	config := Config{
		Level:  LevelError,
		Format: "text",
		Output: &buf,
	}
	
	logger := NewLogger(config)
	SetDefault(logger)
	
	Error("error message", "key", "value")
	output := buf.String()
	
	if !strings.Contains(output, "error message") {
		t.Errorf("Expected error message in output, got: %s", output)
	}
	if !strings.Contains(output, "key") {
		t.Errorf("Expected key in output, got: %s", output)
	}
}

func TestDebugContextFunction(t *testing.T) {
	var buf bytes.Buffer
	config := Config{
		Level:  LevelDebug,
		Format: "text",
		Output: &buf,
	}
	
	logger := NewLogger(config)
	SetDefault(logger)
	
	ctx := context.Background()
	DebugContext(ctx, "debug context message", "key", "value")
	output := buf.String()
	
	if !strings.Contains(output, "debug context message") {
		t.Errorf("Expected debug context message in output, got: %s", output)
	}
	if !strings.Contains(output, "key") {
		t.Errorf("Expected key in output, got: %s", output)
	}
}

func TestInfoContextFunction(t *testing.T) {
	var buf bytes.Buffer
	config := Config{
		Level:  LevelInfo,
		Format: "text",
		Output: &buf,
	}
	
	logger := NewLogger(config)
	SetDefault(logger)
	
	ctx := context.Background()
	InfoContext(ctx, "info context message", "key", "value")
	output := buf.String()
	
	if !strings.Contains(output, "info context message") {
		t.Errorf("Expected info context message in output, got: %s", output)
	}
	if !strings.Contains(output, "key") {
		t.Errorf("Expected key in output, got: %s", output)
	}
}

func TestWarnContextFunction(t *testing.T) {
	var buf bytes.Buffer
	config := Config{
		Level:  LevelWarn,
		Format: "text",
		Output: &buf,
	}
	
	logger := NewLogger(config)
	SetDefault(logger)
	
	ctx := context.Background()
	WarnContext(ctx, "warn context message", "key", "value")
	output := buf.String()
	
	if !strings.Contains(output, "warn context message") {
		t.Errorf("Expected warn context message in output, got: %s", output)
	}
	if !strings.Contains(output, "key") {
		t.Errorf("Expected key in output, got: %s", output)
	}
}

func TestErrorContextFunction(t *testing.T) {
	var buf bytes.Buffer
	config := Config{
		Level:  LevelError,
		Format: "text",
		Output: &buf,
	}
	
	logger := NewLogger(config)
	SetDefault(logger)
	
	ctx := context.Background()
	ErrorContext(ctx, "error context message", "key", "value")
	output := buf.String()
	
	if !strings.Contains(output, "error context message") {
		t.Errorf("Expected error context message in output, got: %s", output)
	}
	if !strings.Contains(output, "key") {
		t.Errorf("Expected key in output, got: %s", output)
	}
}

// Test NewLogger edge cases for better coverage
func TestNewLogger_InvalidFormat(t *testing.T) {
	var buf bytes.Buffer
	config := Config{
		Level:  LevelInfo,
		Format: "invalid-format",
		Output: &buf,
	}

	logger := NewLogger(config)
	if logger == nil {
		t.Fatal("Expected logger to be created even with invalid format")
	}

	// Should fallback to text format
	logger.Info("test message")
	output := buf.String()
	if !strings.Contains(output, "test message") {
		t.Errorf("Expected logger to work with fallback format, got: %s", output)
	}
}

// Test all log levels to ensure proper filtering
func TestAllLogLevels(t *testing.T) {
	tests := []struct {
		name     string
		level    LogLevel
		logFunc  func(msg string, args ...any)
		expected bool
	}{
		{"Debug at Debug level", LevelDebug, Debug, true},
		{"Info at Debug level", LevelDebug, func(msg string, args ...any) { Info(msg, args...) }, true},
		{"Warn at Debug level", LevelDebug, func(msg string, args ...any) { Warn(msg, args...) }, true},
		{"Error at Debug level", LevelDebug, func(msg string, args ...any) { Error(msg, args...) }, true},
		
		{"Debug at Info level", LevelInfo, Debug, false},
		{"Info at Info level", LevelInfo, func(msg string, args ...any) { Info(msg, args...) }, true},
		{"Warn at Info level", LevelInfo, func(msg string, args ...any) { Warn(msg, args...) }, true},
		{"Error at Info level", LevelInfo, func(msg string, args ...any) { Error(msg, args...) }, true},
		
		{"Debug at Warn level", LevelWarn, Debug, false},
		{"Info at Warn level", LevelWarn, func(msg string, args ...any) { Info(msg, args...) }, false},
		{"Warn at Warn level", LevelWarn, func(msg string, args ...any) { Warn(msg, args...) }, true},
		{"Error at Warn level", LevelWarn, func(msg string, args ...any) { Error(msg, args...) }, true},
		
		{"Debug at Error level", LevelError, Debug, false},
		{"Info at Error level", LevelError, func(msg string, args ...any) { Info(msg, args...) }, false},
		{"Warn at Error level", LevelError, func(msg string, args ...any) { Warn(msg, args...) }, false},
		{"Error at Error level", LevelError, func(msg string, args ...any) { Error(msg, args...) }, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var buf bytes.Buffer
			config := Config{
				Level:  tt.level,
				Format: "text",
				Output: &buf,
			}

			logger := NewLogger(config)
			SetDefault(logger)

			tt.logFunc("test message")
			output := buf.String()
			hasMessage := strings.Contains(output, "test message")

			if hasMessage != tt.expected {
				t.Errorf("Expected message present = %v, got = %v, output: %s", 
					tt.expected, hasMessage, output)
			}
		})
	}
}

// Test context logging methods
func TestAllContextLogLevels(t *testing.T) {
	ctx := context.Background()
	
	tests := []struct {
		name     string
		level    LogLevel
		logFunc  func(ctx context.Context, msg string, args ...any)
		expected bool
	}{
		{"DebugContext at Debug level", LevelDebug, DebugContext, true},
		{"InfoContext at Debug level", LevelDebug, InfoContext, true},
		{"WarnContext at Debug level", LevelDebug, WarnContext, true},
		{"ErrorContext at Debug level", LevelDebug, ErrorContext, true},
		
		{"DebugContext at Info level", LevelInfo, DebugContext, false},
		{"InfoContext at Info level", LevelInfo, InfoContext, true},
		{"WarnContext at Info level", LevelInfo, WarnContext, true},
		{"ErrorContext at Info level", LevelInfo, ErrorContext, true},
		
		{"DebugContext at Warn level", LevelWarn, DebugContext, false},
		{"InfoContext at Warn level", LevelWarn, InfoContext, false},
		{"WarnContext at Warn level", LevelWarn, WarnContext, true},
		{"ErrorContext at Warn level", LevelWarn, ErrorContext, true},
		
		{"DebugContext at Error level", LevelError, DebugContext, false},
		{"InfoContext at Error level", LevelError, InfoContext, false},
		{"WarnContext at Error level", LevelError, WarnContext, false},
		{"ErrorContext at Error level", LevelError, ErrorContext, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var buf bytes.Buffer
			config := Config{
				Level:  tt.level,
				Format: "text",
				Output: &buf,
			}

			logger := NewLogger(config)
			SetDefault(logger)

			tt.logFunc(ctx, "test context message")
			output := buf.String()
			hasMessage := strings.Contains(output, "test context message")

			if hasMessage != tt.expected {
				t.Errorf("Expected message present = %v, got = %v, output: %s", 
					tt.expected, hasMessage, output)
			}
		})
	}
}