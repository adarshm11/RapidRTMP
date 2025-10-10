package storage

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
)

// Storage interface for storing and retrieving stream segments
type Storage interface {
	// Write writes data to a file path
	Write(path string, data []byte) error

	// Read reads data from a file path
	Read(path string) ([]byte, error)

	// ReadSeeker returns a ReadSeeker for the file (useful for http.ServeContent)
	ReadSeeker(path string) (io.ReadSeeker, error)

	// Delete deletes a file
	Delete(path string) error

	// Exists checks if a file exists
	Exists(path string) (bool, error)

	// List lists files in a directory
	List(dir string) ([]string, error)
}

// LocalStorage implements Storage using local filesystem
type LocalStorage struct {
	baseDir string
}

// NewLocalStorage creates a new local storage instance
func NewLocalStorage(baseDir string) (*LocalStorage, error) {
	// Create base directory if it doesn't exist
	if err := os.MkdirAll(baseDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create base directory: %w", err)
	}

	return &LocalStorage{
		baseDir: baseDir,
	}, nil
}

// Write writes data to a file
func (s *LocalStorage) Write(path string, data []byte) error {
	fullPath := filepath.Join(s.baseDir, path)

	// Create parent directories
	dir := filepath.Dir(fullPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}

	// Write file
	if err := os.WriteFile(fullPath, data, 0644); err != nil {
		return fmt.Errorf("failed to write file: %w", err)
	}

	return nil
}

// Read reads data from a file
func (s *LocalStorage) Read(path string) ([]byte, error) {
	fullPath := filepath.Join(s.baseDir, path)

	data, err := os.ReadFile(fullPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read file: %w", err)
	}

	return data, nil
}

// ReadSeeker returns a ReadSeeker for the file
func (s *LocalStorage) ReadSeeker(path string) (io.ReadSeeker, error) {
	fullPath := filepath.Join(s.baseDir, path)

	file, err := os.Open(fullPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open file: %w", err)
	}

	return file, nil
}

// Delete deletes a file
func (s *LocalStorage) Delete(path string) error {
	fullPath := filepath.Join(s.baseDir, path)

	if err := os.Remove(fullPath); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to delete file: %w", err)
	}

	return nil
}

// Exists checks if a file exists
func (s *LocalStorage) Exists(path string) (bool, error) {
	fullPath := filepath.Join(s.baseDir, path)

	_, err := os.Stat(fullPath)
	if err != nil {
		if os.IsNotExist(err) {
			return false, nil
		}
		return false, fmt.Errorf("failed to check file existence: %w", err)
	}

	return true, nil
}

// List lists files in a directory
func (s *LocalStorage) List(dir string) ([]string, error) {
	fullPath := filepath.Join(s.baseDir, dir)

	entries, err := os.ReadDir(fullPath)
	if err != nil {
		return nil, fmt.Errorf("failed to list directory: %w", err)
	}

	files := make([]string, 0, len(entries))
	for _, entry := range entries {
		if !entry.IsDir() {
			files = append(files, entry.Name())
		}
	}

	return files, nil
}

// GetFullPath returns the full filesystem path for a relative path
func (s *LocalStorage) GetFullPath(path string) string {
	return filepath.Join(s.baseDir, path)
}
