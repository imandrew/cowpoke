package domain

import (
	"os"
)

// FileSystemAdapter defines the interface for file operations.
type FileSystemAdapter interface {
	ReadFile(path string) ([]byte, error)
	WriteFile(path string, data []byte, perm os.FileMode) error
	MkdirAll(path string, perm os.FileMode) error
	Remove(path string) error
	Stat(path string) (os.FileInfo, error)
	Chmod(path string, perm os.FileMode) error
	UserHomeDir() (string, error)
	TempDir() string
}
