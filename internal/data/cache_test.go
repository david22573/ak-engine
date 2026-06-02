package data

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestCachePathResolutionAndSafety(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "test_cache_safety_*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	c := NewCache(tmpDir)

	// 1. Normal resolution preserves key structure
	key := "candles/futures-um/1m/symbol=LINKUSDT/year=2023/month=01/LINKUSDT-1m-2023-01.parquet"
	resolved, err := c.ResolvePath(key)
	if err != nil {
		t.Fatalf("unexpected error resolving valid key: %v", err)
	}

	expected := filepath.Join(tmpDir, key)
	if resolved != expected {
		t.Errorf("expected path %q, got %q", expected, resolved)
	}

	// 2. Reject relative upward directory traversal
	badKeys := []string{
		"../traversal.parquet",
		"candles/../../traversal.parquet",
		"/absolute/path/traversal.parquet",
	}

	for _, bk := range badKeys {
		_, err := c.ResolvePath(bk)
		if err == nil {
			t.Errorf("expected error for traversal key %q, got nil", bk)
		} else if !strings.Contains(err.Error(), "traversal") {
			t.Errorf("expected error message to contain 'traversal', got: %v", err)
		}
	}

	// 3. Ensure dir creates parents
	resolvedFile, err := c.ResolvePath(key)
	if err != nil {
		t.Fatalf("unexpected ResolvePath error: %v", err)
	}

	if err := c.EnsureDir(resolvedFile); err != nil {
		t.Fatalf("failed to ensure dir: %v", err)
	}

	parentDir := filepath.Dir(resolvedFile)
	fi, err := os.Stat(parentDir)
	if err != nil {
		t.Fatalf("parent directory was not created: %v", err)
	}
	if !fi.IsDir() {
		t.Errorf("expected %q to be a directory", parentDir)
	}
}
