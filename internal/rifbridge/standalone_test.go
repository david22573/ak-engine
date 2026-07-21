package rifbridge_test

import (
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestEngineDoesNotDependOnRIFSourceCheckout(t *testing.T) {
	root := repoRoot(t)
	goMod, err := os.ReadFile(filepath.Join(root, "go.mod"))
	if err != nil {
		t.Fatalf("read go.mod: %v", err)
	}
	const version = "github.com/david22573/ak-rif v0.0.0-20260720214045-23be9a8ef9b7"
	if !strings.Contains(string(goMod), version) || strings.Contains(string(goMod), "replace github.com/david22573/ak-rif") {
		t.Fatalf("go.mod must require the accepted vendored RIF snapshot without a source-checkout replacement")
	}
	commit, err := os.ReadFile(filepath.Join(root, "vendor", "github.com", "david22573", "ak-rif", "RIF_SOURCE_COMMIT"))
	if err != nil || strings.TrimSpace(string(commit)) != "23be9a8ef9b754af4fe61eacea3650404707b484" {
		t.Fatalf("vendored RIF provenance is missing or incorrect: %v", err)
	}

	err = filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			switch d.Name() {
			case ".git", ".cache", "bin", "runs":
				return filepath.SkipDir
			}
			return nil
		}
		if filepath.Ext(path) != ".go" {
			return nil
		}
		file, err := parser.ParseFile(token.NewFileSet(), path, nil, parser.ImportsOnly)
		if err != nil {
			return err
		}
		for _, imported := range file.Imports {
			importPath := strings.Trim(imported.Path.Value, "\"")
			if importPath == "ak-rif" || strings.HasPrefix(importPath, "ak-rif/") ||
				importPath == "github.com/david22573/ak-rif" || strings.HasPrefix(importPath, "github.com/david22573/ak-rif/") {
				relative, _ := filepath.Rel(root, path)
				if !strings.HasPrefix(filepath.ToSlash(relative), "internal/epochorchestrator/") && !strings.HasPrefix(filepath.ToSlash(relative), "vendor/") {
					t.Fatalf("RIF implementation import escaped the production orchestrator adapter: %s", path)
				}
			}
		}
		return nil
	})
	if err != nil {
		t.Fatalf("walk source: %v", err)
	}
}

func repoRoot(t *testing.T) string {
	t.Helper()
	dir, err := os.Getwd()
	if err != nil {
		t.Fatalf("get cwd: %v", err)
	}
	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			t.Fatal("go.mod not found")
		}
		dir = parent
	}
}
