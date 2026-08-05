package researchidentity

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestGitRepositoryStateProviderDerivesActualCleanCommitTreeAndDirtyState(t *testing.T) {
	root := t.TempDir()
	write := func(path, contents string) {
		t.Helper()
		absolute := filepath.Join(root, path)
		if err := os.MkdirAll(filepath.Dir(absolute), 0755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(absolute, []byte(contents), 0644); err != nil {
			t.Fatal(err)
		}
	}
	write("go.mod", "module fixture.local/engine\n\ngo 1.25\n")
	write("main.go", "package main\nfunc main() {}\n")
	runGitFixture(t, root, "init", "-q")
	runGitFixture(t, root, "add", "go.mod", "main.go")
	runGitFixture(t, root, "-c", "user.name=Fixture", "-c", "user.email=fixture@example.invalid", "commit", "-qm", "fixture")

	provider := gitRepositoryStateProvider{}
	state, err := provider.Resolve(root)
	if err != nil {
		t.Fatal(err)
	}
	if state.Dirty || state.RepositoryID != "fixture.local/engine" || !isLowerHex40(state.CommitSHA) || !isLowerHex40(state.TreeSHA) || state.CommitSHA == state.TreeSHA {
		t.Fatalf("actual clean repository identity is incomplete: %#v", state)
	}
	write("untracked.txt", "dirty\n")
	dirty, err := provider.Resolve(root)
	if err != nil {
		t.Fatal(err)
	}
	if !dirty.Dirty || dirty.CommitSHA != state.CommitSHA || dirty.TreeSHA != state.TreeSHA {
		t.Fatalf("dirty worktree masqueraded or changed clean HEAD facts: clean=%#v dirty=%#v", state, dirty)
	}

	subdir := filepath.Join(root, "subdir")
	if err := os.MkdirAll(subdir, 0755); err != nil {
		t.Fatal(err)
	}
	if _, err := provider.Resolve(subdir); err == nil || !strings.Contains(err.Error(), "ambiguous") {
		t.Fatalf("ambiguous repository root did not fail: %v", err)
	}
}

func runGitFixture(t *testing.T, root string, args ...string) {
	t.Helper()
	cmd := exec.Command("git", args...)
	cmd.Dir = root
	if output, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("git %v: %s: %v", args, strings.TrimSpace(string(output)), err)
	}
}
