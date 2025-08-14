package cmd

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/spf13/viper"
)

// Root Command Tests

func TestExecute_Success(t *testing.T) {
	// Since Execute() calls os.Exit(), we can't easily test it directly.
	// Instead, we'll test the command structure and properties which
	// validates the command setup without triggering os.Exit().

	// Test that rootCmd is properly initialized
	if rootCmd.Use != "cowpoke" {
		t.Errorf("Expected Use to be 'cowpoke', got: %s", rootCmd.Use)
	}

	if rootCmd.Short == "" {
		t.Error("Expected Short description to be set")
	}

	if rootCmd.Long == "" {
		t.Error("Expected Long description to be set")
	}

	// Test that the command can be executed with --help without modification
	// This validates the command structure without global state changes
	cmd := rootCmd
	cmd.SetArgs([]string{"--help"})

	// We can't call Execute() because it calls os.Exit(), but we can
	// verify the command is properly structured for execution
	if cmd.Runnable() {
		t.Error("Expected root command to not be directly runnable (should only have subcommands)")
	}
}

func TestExecute_CommandStructure(t *testing.T) {
	// Test that all expected subcommands are registered
	commands := rootCmd.Commands()

	expectedCommands := []string{"add", "list", "remove", "sync"}
	foundCommands := make(map[string]bool)

	for _, cmd := range commands {
		foundCommands[cmd.Use] = true
	}

	for _, expected := range expectedCommands {
		if !foundCommands[expected] {
			t.Errorf("Expected command '%s' to be registered", expected)
		}
	}
}

func TestExecute_PersistentFlags(t *testing.T) {
	// Test that the config flag is properly set up
	configFlag := rootCmd.PersistentFlags().Lookup("config")
	if configFlag == nil {
		t.Error("Expected 'config' persistent flag to be defined")
		return
	}

	if configFlag.Usage == "" {
		t.Error("Expected config flag to have usage text")
	}

	if configFlag.DefValue != "" {
		t.Errorf("Expected config flag default to be empty, got: %s", configFlag.DefValue)
	}
}

// InitConfig Tests

func TestInitConfig_WithConfigFile(t *testing.T) {
	// Create a temporary config file
	tempDir := t.TempDir()
	configPath := filepath.Join(tempDir, "test-config.yaml")

	configContent := `version: "1.0"
servers: []
`
	err := os.WriteFile(configPath, []byte(configContent), 0o644)
	if err != nil {
		t.Fatalf("Failed to create test config file: %v", err)
	}

	// Save original cfgFile and restore after test
	originalCfgFile := cfgFile
	defer func() { cfgFile = originalCfgFile }()

	// Save original viper config and restore after test
	originalConfigFile := viper.ConfigFileUsed()
	defer func() {
		viper.Reset()
		if originalConfigFile != "" {
			viper.SetConfigFile(originalConfigFile)
			_ = viper.ReadInConfig()
		}
	}()

	// Set cfgFile to our test config
	cfgFile = configPath

	// Call initConfig
	initConfig()

	// Verify viper is using our config file
	if viper.ConfigFileUsed() != configPath {
		t.Errorf("Expected viper to use config file %s, got: %s", configPath, viper.ConfigFileUsed())
	}
}

func TestInitConfig_WithoutConfigFile(t *testing.T) {
	// Create a temporary home directory
	tempHome := t.TempDir()
	t.Setenv("HOME", tempHome)

	// Save original cfgFile and restore after test
	originalCfgFile := cfgFile
	defer func() { cfgFile = originalCfgFile }()

	// Save original viper config and restore after test
	defer func() {
		viper.Reset()
	}()

	// Clear cfgFile so it uses default behavior
	cfgFile = ""

	// Call initConfig
	initConfig()

	// Verify viper is configured for automatic env
	// We can't easily test the exact config paths without more complex setup,
	// but we can verify that the function completes without error

	// Check that viper has some configuration set
	viper.Set("test_key", "test_value")
	if viper.GetString("test_key") != "test_value" {
		t.Error("Expected viper to be functional after initConfig")
	}
}

func TestInitConfig_UserHomeDirError(t *testing.T) {
	// This test documents the expected behavior when HOME is not set.
	// The function calls cobra.CheckErr() which calls os.Exit(1),
	// making it difficult to test in a unit test environment.
	// In practice, this would cause the application to exit with status 1.

	t.Skip("Cannot test cobra.CheckErr() behavior that calls os.Exit(1) in unit tests")
}

func TestInitConfig_ConfigFileRead(t *testing.T) {
	// Create a temporary directory and config file
	tempDir := t.TempDir()
	configDir := filepath.Join(tempDir, ".config", "cowpoke")
	err := os.MkdirAll(configDir, 0o755)
	if err != nil {
		t.Fatalf("Failed to create config directory: %v", err)
	}

	configPath := filepath.Join(configDir, "config.yaml")
	configContent := `version: "1.0"
servers:
  - id: "test-server"
    name: "Test Server"
    url: "https://rancher.example.com"
    username: "admin"
    authType: "local"
`
	err = os.WriteFile(configPath, []byte(configContent), 0o644)
	if err != nil {
		t.Fatalf("Failed to create test config file: %v", err)
	}

	t.Setenv("HOME", tempDir)

	// Save original cfgFile and restore after test
	originalCfgFile := cfgFile
	defer func() { cfgFile = originalCfgFile }()

	// Save original viper config and restore after test
	defer func() {
		viper.Reset()
	}()

	// Clear cfgFile so it uses default behavior
	cfgFile = ""

	// Call initConfig
	initConfig()

	// Verify that the config was read
	if viper.GetString("version") != "1.0" {
		t.Errorf("Expected version '1.0' from config, got: %s", viper.GetString("version"))
	}
}

func TestInitConfig_NoConfigFile(t *testing.T) {
	// Create a temporary directory without config file
	tempDir := t.TempDir()

	t.Setenv("HOME", tempDir)

	// Save original cfgFile and restore after test
	originalCfgFile := cfgFile
	defer func() { cfgFile = originalCfgFile }()

	// Save original viper config and restore after test
	defer func() {
		viper.Reset()
	}()

	// Clear cfgFile so it uses default behavior
	cfgFile = ""

	// Call initConfig - should complete without error even if no config file exists
	initConfig()

	// Verify viper is still functional
	viper.Set("test_key", "test_value")
	if viper.GetString("test_key") != "test_value" {
		t.Error("Expected viper to be functional after initConfig even without config file")
	}
}

// Test cobra initialization.
func TestCobraInitialization(t *testing.T) {
	// Verify that cobra.OnInitialize was called with initConfig
	// This is difficult to test directly, but we can verify the flag setup

	// Test that the persistent flag was set up correctly
	configFlag := rootCmd.PersistentFlags().Lookup("config")
	if configFlag == nil {
		t.Error("Expected config flag to be set up during init")
		return
	}

	// Test flag properties
	if configFlag.Shorthand != "" {
		t.Errorf("Expected config flag to have no shorthand, got: %s", configFlag.Shorthand)
	}

	if !configFlag.Changed && configFlag.Value.String() != "" {
		t.Errorf("Expected config flag to be empty by default, got: %s", configFlag.Value.String())
	}
}

// Integration test for the complete root command setup.
func TestRootCommandIntegration(t *testing.T) {
	// Test that the root command has all expected properties
	if rootCmd.Use != "cowpoke" {
		t.Errorf("Expected Use to be 'cowpoke', got: %s", rootCmd.Use)
	}

	if len(rootCmd.Commands()) == 0 {
		t.Error("Expected root command to have subcommands")
	}

	// Test that persistent flags are inherited by subcommands
	for _, cmd := range rootCmd.Commands() {
		configFlag := cmd.InheritedFlags().Lookup("config")
		if configFlag == nil {
			t.Errorf("Expected subcommand '%s' to inherit config flag", cmd.Use)
		}
	}
}

// Test edge cases and error conditions.
func TestInitConfig_EdgeCases(t *testing.T) {
	// Save original state
	originalCfgFile := cfgFile
	defer func() { cfgFile = originalCfgFile }()
	defer func() { viper.Reset() }()

	// Test with a config file that doesn't exist
	cfgFile = "/path/that/does/not/exist/config.yaml"

	// This should not panic, even though the file doesn't exist
	// viper.ReadInConfig() will return an error, but it's handled gracefully
	initConfig()

	// Verify viper is still functional
	viper.Set("test", "value")
	if viper.GetString("test") != "value" {
		t.Error("Expected viper to remain functional even with non-existent config file")
	}
}
