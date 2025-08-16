package terminal

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"

	"golang.org/x/term"
)

// Adapter handles secure password input from terminal.
type Adapter struct {
	stdin  io.Reader
	stderr io.Writer
}

// NewAdapter creates a new terminal adapter.
func NewAdapter(stdin io.Reader, stderr io.Writer) *Adapter {
	return &Adapter{
		stdin:  stdin,
		stderr: stderr,
	}
}

// ReadPassword reads a password from the terminal with echo disabled.
func (a *Adapter) ReadPassword(ctx context.Context, prompt string) (string, error) {
	// Check if context is cancelled
	select {
	case <-ctx.Done():
		return "", ctx.Err()
	default:
	}

	// Check for RANCHER_PASSWORD environment variable first (useful for CI/CD).
	if envPassword := os.Getenv("RANCHER_PASSWORD"); envPassword != "" {
		return envPassword, nil
	}

	if !a.IsInteractive() {
		// For non-interactive environments (e.g., CI/CD), return empty string
		// The caller should handle this appropriately (e.g., use token auth)
		return "", errors.New("cannot read password: non-interactive terminal")
	}

	fmt.Fprint(a.stderr, prompt)

	// Type assertion to check if stdin is a file
	if file, ok := a.stdin.(*os.File); ok {
		password, err := term.ReadPassword(int(file.Fd()))
		fmt.Fprintln(a.stderr) // Print newline after password input
		if err != nil {
			return "", fmt.Errorf("failed to read password: %w", err)
		}
		return string(password), nil
	}

	return "", errors.New("cannot read password from non-terminal input")
}

// IsInteractive returns true if the terminal is interactive.
func (a *Adapter) IsInteractive() bool {
	if file, ok := a.stdin.(*os.File); ok {
		return term.IsTerminal(int(file.Fd()))
	}
	return false
}
