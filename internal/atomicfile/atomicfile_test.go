package atomicfile

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
)

func TestRenameFailurePreservesPriorArtifact(t *testing.T) {
	path := filepath.Join(t.TempDir(), "evidence.json")
	if err := os.WriteFile(path, []byte("prior"), 0644); err != nil {
		t.Fatal(err)
	}
	ops := systemOps
	ops.rename = func(string, string) error { return errors.New("injected rename failure") }
	if err := writeFile(path, []byte("new"), 0644, ops); err == nil {
		t.Fatal("writeFile() error=nil, want failure")
	}
	got, err := os.ReadFile(path)
	if err != nil || string(got) != "prior" {
		t.Fatalf("prior artifact changed: %q err=%v", got, err)
	}
}
