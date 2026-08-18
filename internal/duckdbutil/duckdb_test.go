package duckdbutil

import (
	"context"
	"os"
	"os/exec"
	"strings"
	"testing"
)

func TestQuoteString(t *testing.T) {
	tests := []struct {
		in   string
		want string
	}{
		{"", "''"},
		{"simple", "'simple'"},
		{"it's", "'it''s'"},
		{"path/to/file.parquet", "'path/to/file.parquet'"},
	}

	for _, tc := range tests {
		if got := QuoteString(tc.in); got != tc.want {
			t.Errorf("QuoteString(%q) = %q, want %q", tc.in, got, tc.want)
		}
	}
}

func TestMemoryLimit(t *testing.T) {
	origAK := os.Getenv("AK_DUCKDB_MEMORY_LIMIT")
	origDuckDB := os.Getenv("DUCKDB_MEMORY_LIMIT")
	defer func() {
		os.Setenv("AK_DUCKDB_MEMORY_LIMIT", origAK)
		os.Setenv("DUCKDB_MEMORY_LIMIT", origDuckDB)
	}()

	os.Unsetenv("AK_DUCKDB_MEMORY_LIMIT")
	os.Unsetenv("DUCKDB_MEMORY_LIMIT")
	if got := MemoryLimit(); got != DefaultMemoryLimit {
		t.Errorf("MemoryLimit() default = %q, want %q", got, DefaultMemoryLimit)
	}

	os.Setenv("AK_DUCKDB_MEMORY_LIMIT", "1GB")
	if got := MemoryLimit(); got != "1GB" {
		t.Errorf("MemoryLimit() with AK_DUCKDB_MEMORY_LIMIT = %q, want '1GB'", got)
	}
}

func TestWrapWithPragmas(t *testing.T) {
	q := "SELECT 1;"
	wrapped := WrapWithPragmas(q)
	if !strings.Contains(wrapped, "PRAGMA memory_limit=") {
		t.Errorf("WrapWithPragmas() missing PRAGMA memory_limit: %s", wrapped)
	}
	if !strings.Contains(wrapped, "PRAGMA preserve_insertion_order=false;") {
		t.Errorf("WrapWithPragmas() missing preserve_insertion_order: %s", wrapped)
	}
}

func TestRunQuery(t *testing.T) {
	if _, err := exec.LookPath("duckdb"); err != nil {
		t.Skip("duckdb not found in PATH")
	}

	out, err := RunQuery(context.Background(), "SELECT 42 AS result;", "-csv", "-noheader")
	if err != nil {
		t.Fatalf("RunQuery() error = %v", err)
	}
	if strings.TrimSpace(string(out)) != "42" {
		t.Errorf("RunQuery() output = %q, want '42'", string(out))
	}
}
