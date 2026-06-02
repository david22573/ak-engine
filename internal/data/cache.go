package data

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

type Cache struct {
	Dir string
}

func NewCache(dir string) *Cache {
	if dir == "" {
		dir = ".ak-engine/cache/r2"
	}
	return &Cache{Dir: dir}
}

// ResolvePath resolves the local cache path for a given object key,
// preserving the path structure and preventing path traversal.
func (c *Cache) ResolvePath(key string) (string, error) {
	cleanedKey := filepath.Clean(key)
	// Reject absolute paths or relative upward traversal in key
	if strings.HasPrefix(cleanedKey, "..") || strings.HasPrefix(cleanedKey, "/") || strings.Contains(cleanedKey, "../") {
		return "", fmt.Errorf("invalid object key (path traversal detected): %s", key)
	}

	absDir, err := filepath.Abs(c.Dir)
	if err != nil {
		return "", fmt.Errorf("failed to get absolute cache directory path: %w", err)
	}

	resolvedPath := filepath.Join(absDir, cleanedKey)

	// Verify that the resolved path is strictly within the cache directory
	if !strings.HasPrefix(resolvedPath, absDir) {
		return "", fmt.Errorf("path traversal attempt detected: resolved path %s is outside cache dir %s", resolvedPath, absDir)
	}

	return resolvedPath, nil
}

// EnsureDir creates parent directory for the resolved path
func (c *Cache) EnsureDir(path string) error {
	parentDir := filepath.Dir(path)
	if err := os.MkdirAll(parentDir, 0755); err != nil {
		return fmt.Errorf("failed to create directory %s: %w", parentDir, err)
	}
	return nil
}
