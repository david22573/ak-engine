package researchidentity

import (
	"bufio"
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"runtime/debug"
	"sort"
	"strings"

	"github.com/david22573/ak-engine/internal/canonicalcontract"
)

type gitRepositoryStateProvider struct{}

// FindRepositoryRoot resolves one unambiguous Git worktree root from start.
// The returned path is a locator only; commit/tree identity is still derived
// and verified by the repository-state provider.
func FindRepositoryRoot(start string) (string, error) {
	if strings.TrimSpace(start) == "" {
		var err error
		start, err = os.Getwd()
		if err != nil {
			return "", err
		}
	}
	start, err := filepath.Abs(start)
	if err != nil {
		return "", err
	}
	root, err := gitOutput(start, "rev-parse", "--show-toplevel")
	if err != nil {
		return "", err
	}
	return filepath.Abs(root)
}

func (gitRepositoryStateProvider) Resolve(repositoryRoot string) (RepositoryState, error) {
	rootAbs, err := filepath.Abs(repositoryRoot)
	if err != nil {
		return RepositoryState{}, err
	}
	resolvedRoot, err := gitOutput(rootAbs, "rev-parse", "--show-toplevel")
	if err != nil {
		return RepositoryState{}, err
	}
	resolvedRoot, err = filepath.Abs(resolvedRoot)
	if err != nil {
		return RepositoryState{}, err
	}
	if resolvedRoot != rootAbs {
		return RepositoryState{}, fmt.Errorf("repository root is ambiguous: requested %s resolved %s", rootAbs, resolvedRoot)
	}
	commit, err := gitOutput(rootAbs, "rev-parse", "HEAD^{commit}")
	if err != nil {
		return RepositoryState{}, err
	}
	tree, err := gitOutput(rootAbs, "rev-parse", "HEAD^{tree}")
	if err != nil {
		return RepositoryState{}, err
	}
	status, err := gitOutputAllowEmpty(rootAbs, "status", "--porcelain=v1", "--untracked-files=all")
	if err != nil {
		return RepositoryState{}, err
	}
	repositoryID, err := moduleIdentity(filepath.Join(rootAbs, "go.mod"))
	if err != nil {
		return RepositoryState{}, err
	}
	state := RepositoryState{
		Root:         rootAbs,
		RepositoryID: repositoryID,
		CommitSHA:    commit,
		TreeSHA:      tree,
		Dirty:        strings.TrimSpace(status) != "",
		BuildVersion: "0.1.0-devel",
		GoVersion:    runtime.Version(),
		GoOS:         runtime.GOOS,
		GoARCH:       runtime.GOARCH,
		Compiler:     runtime.Compiler,
		CGOEnabled:   "unknown",
	}
	if info, ok := debug.ReadBuildInfo(); ok {
		if info.Main.Version != "" && info.Main.Version != "(devel)" {
			state.BuildVersion = info.Main.Version
		}
		for _, setting := range info.Settings {
			switch setting.Key {
			case "vcs.revision":
				state.BinaryRevision = setting.Value
			case "vcs.modified":
				state.BinaryModified = setting.Value == "true"
			case "-tags":
				state.BuildTags = splitBuildTags(setting.Value)
			case "CGO_ENABLED":
				state.CGOEnabled = setting.Value
			}
		}
	}
	return state, nil
}

func deriveEngineSource(provider RepositoryStateProvider, root string, expectedBuildTags []string) (EngineSourceIdentity, error) {
	if provider == nil {
		return EngineSourceIdentity{}, fmt.Errorf("repository state provider is nil")
	}
	state, err := provider.Resolve(root)
	if err != nil {
		return EngineSourceIdentity{}, err
	}
	if state.Dirty || state.BinaryModified {
		return EngineSourceIdentity{}, &DerivationError{Status: StatusDirtyEngineSource, Code: "ENGINE_SOURCE_DIRTY", Err: fmt.Errorf("executing Engine source is dirty")}
	}
	if state.RepositoryID == "" || !isLowerHex40(state.CommitSHA) || !isLowerHex40(state.TreeSHA) || strings.TrimSpace(state.BuildVersion) == "" || strings.TrimSpace(state.GoVersion) == "" || strings.TrimSpace(state.GoOS) == "" || strings.TrimSpace(state.GoARCH) == "" || strings.TrimSpace(state.Compiler) == "" || strings.TrimSpace(state.CGOEnabled) == "" {
		return EngineSourceIdentity{}, fmt.Errorf("clean Engine source identity is incomplete")
	}
	if state.CGOEnabled != "0" && state.CGOEnabled != "1" {
		return EngineSourceIdentity{}, fmt.Errorf("executing CGO_ENABLED setting is unavailable or invalid")
	}
	if state.BinaryRevision != "" && state.BinaryRevision != state.CommitSHA {
		return EngineSourceIdentity{}, fmt.Errorf("executing binary revision %s does not match repository commit %s", state.BinaryRevision, state.CommitSHA)
	}
	actualTags := append([]string{}, state.BuildTags...)
	expectedTags := append([]string{}, expectedBuildTags...)
	sort.Strings(actualTags)
	sort.Strings(expectedTags)
	if strings.Join(actualTags, "\x00") != strings.Join(expectedTags, "\x00") {
		return EngineSourceIdentity{}, fmt.Errorf("executing build tags do not match resolved configuration")
	}
	identity := EngineSourceIdentity{
		Contract:     canonicalcontract.NewHeader(engineSourceSchemaName, canonicalContractVersion, engineSourceRole),
		RepositoryID: state.RepositoryID,
		CommitSHA:    state.CommitSHA,
		TreeSHA:      state.TreeSHA,
		Dirty:        false,
		BuildVersion: state.BuildVersion,
		GoVersion:    state.GoVersion,
		GoOS:         state.GoOS,
		GoARCH:       state.GoARCH,
		Compiler:     state.Compiler,
		CGOEnabled:   state.CGOEnabled,
		BuildTags:    actualTags,
	}
	identity.ArtifactHash, err = artifactHash(engineSourceSchemaName, engineSourceRole, identity)
	if err != nil {
		return EngineSourceIdentity{}, err
	}
	return identity, nil
}

func gitOutput(root string, args ...string) (string, error) {
	value, err := gitOutputAllowEmpty(root, args...)
	if err != nil {
		return "", err
	}
	if strings.TrimSpace(value) == "" {
		return "", fmt.Errorf("git %s returned empty output", strings.Join(args, " "))
	}
	return strings.TrimSpace(value), nil
}

func gitOutputAllowEmpty(root string, args ...string) (string, error) {
	cmd := exec.Command("git", args...)
	cmd.Dir = root
	output, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("git %s: %s: %w", strings.Join(args, " "), strings.TrimSpace(string(output)), err)
	}
	return string(output), nil
}

func moduleIdentity(path string) (string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	scanner := bufio.NewScanner(bytes.NewReader(data))
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if strings.HasPrefix(line, "module ") {
			value := strings.TrimSpace(strings.TrimPrefix(line, "module "))
			if value == "" {
				break
			}
			return value, nil
		}
	}
	if err := scanner.Err(); err != nil {
		return "", err
	}
	return "", fmt.Errorf("go.mod module identity is missing")
}

func splitBuildTags(value string) []string {
	fields := strings.FieldsFunc(value, func(r rune) bool { return r == ',' || r == ' ' })
	tags := make([]string, 0, len(fields))
	for _, field := range fields {
		if field = strings.TrimSpace(field); field != "" {
			tags = append(tags, field)
		}
	}
	sort.Strings(tags)
	return tags
}

func isLowerHex40(value string) bool {
	if len(value) != 40 {
		return false
	}
	for _, r := range value {
		if !(r >= '0' && r <= '9') && !(r >= 'a' && r <= 'f') {
			return false
		}
	}
	return true
}
