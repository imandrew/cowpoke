package filesystem

import (
	"os"
)

// Adapter provides file system operations.
type Adapter struct{}

// New creates a new filesystem adapter.
func New() *Adapter {
	return &Adapter{}
}

// ReadFile reads a file from disk.
func (a *Adapter) ReadFile(path string) ([]byte, error) {
	return os.ReadFile(path)
}

// WriteFile writes data to a file.
func (a *Adapter) WriteFile(path string, data []byte, perm os.FileMode) error {
	return os.WriteFile(path, data, perm)
}

// MkdirAll creates a directory and all necessary parents.
func (a *Adapter) MkdirAll(path string, perm os.FileMode) error {
	return os.MkdirAll(path, perm)
}

// Remove deletes a file.
func (a *Adapter) Remove(path string) error {
	return os.Remove(path)
}

// Stat returns file info.
func (a *Adapter) Stat(path string) (os.FileInfo, error) {
	return os.Stat(path)
}

// Chmod changes the file permissions.
func (a *Adapter) Chmod(path string, perm os.FileMode) error {
	return os.Chmod(path, perm)
}

// UserHomeDir returns the user's home directory.
func (a *Adapter) UserHomeDir() (string, error) {
	return os.UserHomeDir()
}

// TempDir returns the temporary directory.
func (a *Adapter) TempDir() string {
	return os.TempDir()
}
