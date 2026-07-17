package app

import (
	"os"
	"path/filepath"
	"testing"
)

func TestQualificationRunnerAuthorityFileRejectsSymlink(t *testing.T) {
	dir := t.TempDir()
	realPath := filepath.Join(dir, "registered.json")
	if err := os.WriteFile(realPath, []byte("{}\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	linkPath := filepath.Join(dir, "alternate-cache.json")
	if err := os.Symlink(realPath, linkPath); err != nil {
		t.Fatal(err)
	}
	if _, err := readQualificationAuthorityFile(linkPath); err == nil {
		t.Fatal("symlinked alternate authority artifact was accepted")
	}
}
