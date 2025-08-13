package utils

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"syscall"

	"cowpoke/internal/config"

	"golang.org/x/term"
)

// Input/Terminal utilities

// PromptForPassword prompts the user for a password without echoing it to the terminal
func PromptForPassword(prompt string) (string, error) {
	fmt.Print(prompt)

	// Get the file descriptor for stdin
	fd := int(syscall.Stdin)

	// Read password without echo
	password, err := term.ReadPassword(fd)
	if err != nil {
		return "", fmt.Errorf("failed to read password: %w", err)
	}

	// Print a newline since ReadPassword doesn't
	fmt.Println()

	return string(password), nil
}

// IsTerminal checks if the given file descriptor is a terminal
func IsTerminal(fd int) bool {
	return term.IsTerminal(fd)
}

// CanPromptForPassword checks if we can prompt for password (i.e., running in a terminal)
func CanPromptForPassword() bool {
	return IsTerminal(int(os.Stdin.Fd()))
}

// Path utilities

// GetConfigPath returns the path to the configuration file
func GetConfigPath() (string, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("failed to get home directory: %w", err)
	}
	return filepath.Join(homeDir, ".config", "cowpoke", "config.yaml"), nil
}

// GetKubeconfigDir returns the path to the kubeconfig directory
func GetKubeconfigDir() (string, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("failed to get home directory: %w", err)
	}
	return filepath.Join(homeDir, ".config", "cowpoke", "kubeconfigs"), nil
}

// GetConfigManager creates a new ConfigManager with the standard config path
func GetConfigManager() (*config.ConfigManager, error) {
	configPath, err := GetConfigPath()
	if err != nil {
		return nil, err
	}
	return config.NewConfigManager(configPath), nil
}

// GetDefaultKubeconfigPath returns the path to the default kubectl config file
func GetDefaultKubeconfigPath() (string, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("failed to get home directory: %w", err)
	}
	return filepath.Join(homeDir, ".kube", "config"), nil
}

// Filename/URL sanitization utilities

// SanitizeFilename removes or replaces invalid characters from filenames
func SanitizeFilename(filename string) string {
	// Replace invalid characters with underscores
	invalidChars := regexp.MustCompile(`[<>:"/\\|?*]`)
	sanitized := invalidChars.ReplaceAllString(filename, "_")

	// Remove consecutive underscores
	multipleUnderscores := regexp.MustCompile(`_+`)
	sanitized = multipleUnderscores.ReplaceAllString(sanitized, "_")

	// Trim underscores from start and end
	sanitized = strings.Trim(sanitized, "_")

	// Ensure filename is not empty
	if sanitized == "" {
		sanitized = "unnamed"
	}

	return sanitized
}

// SanitizeURL converts a URL into a safe filename component
// Example: "https://rancher.example.com" -> "rancher-example-com"
func SanitizeURL(url string) string {
	// Remove protocol
	cleaned := strings.TrimPrefix(url, "https://")
	cleaned = strings.TrimPrefix(cleaned, "http://")

	// Remove port if present
	if idx := strings.Index(cleaned, ":"); idx != -1 {
		cleaned = cleaned[:idx]
	}

	// Remove path if present
	if idx := strings.Index(cleaned, "/"); idx != -1 {
		cleaned = cleaned[:idx]
	}

	// Replace dots with dashes
	cleaned = strings.ReplaceAll(cleaned, ".", "-")

	// Replace any other invalid characters with dashes
	invalidChars := regexp.MustCompile(`[^a-zA-Z0-9\-]`)
	cleaned = invalidChars.ReplaceAllString(cleaned, "-")

	// Remove consecutive dashes
	multipleDashes := regexp.MustCompile(`-+`)
	cleaned = multipleDashes.ReplaceAllString(cleaned, "-")

	// Trim dashes from start and end
	cleaned = strings.Trim(cleaned, "-")

	// Ensure result is not empty
	if cleaned == "" {
		cleaned = "unknown-server"
	}

	return cleaned
}
