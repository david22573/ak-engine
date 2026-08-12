package epochorchestrator

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"sort"
	"strings"

	"github.com/david22573/ak-rif/research"
)

const productionBuildModeID = "go-build-trimpath-buildvcs-false-vendor-cgo-disabled"

type ProductionRunnerBuild struct {
	RepositoryPath string `json:"repository_path"`
	Package        string `json:"package"`
	GOOS           string `json:"goos"`
	GOARCH         string `json:"goarch"`
}

func ComputeProductionRunnerIdentity(build ProductionRunnerBuild, sourceCommit string) (research.RunnerImplementationIdentity, error) {
	if build.RepositoryPath == "" || filepath.Clean(build.RepositoryPath) != build.RepositoryPath || build.Package != "./cmd/ak-engine" || build.GOOS != runtime.GOOS || build.GOARCH != runtime.GOARCH || len(sourceCommit) != 40 {
		return research.RunnerImplementationIdentity{}, errors.New("exact native production runner build inputs are required")
	}
	real, err := filepath.EvalSymlinks(build.RepositoryPath)
	if err != nil || real != build.RepositoryPath {
		return research.RunnerImplementationIdentity{}, errors.New("production runner repository must be a canonical nonsymlink path")
	}
	head, err := git(build.RepositoryPath, "rev-parse", "HEAD")
	if err != nil || head != sourceCommit {
		return research.RunnerImplementationIdentity{}, errors.New("production runner source commit mismatch")
	}
	tree, err := git(build.RepositoryPath, "rev-parse", "HEAD^{tree}")
	if err != nil || len(tree) != 40 {
		return research.RunnerImplementationIdentity{}, errors.New("production runner source tree is unavailable")
	}
	packageBytes, _ := json.Marshal(struct {
		Package string `json:"package"`
		Commit  string `json:"commit"`
		Tree    string `json:"tree"`
	}{build.Package, sourceCommit, tree})
	inputs := []research.HashIdentity{}
	for _, relative := range []string{"go.mod", "go.sum", "vendor/modules.txt", "vendor/github.com/david22573/ak-rif/RIF_SOURCE_COMMIT"} {
		data, readErr := os.ReadFile(filepath.Join(build.RepositoryPath, filepath.FromSlash(relative)))
		if readErr != nil {
			return research.RunnerImplementationIdentity{}, fmt.Errorf("read deterministic build input %s: %w", relative, readErr)
		}
		inputs = append(inputs, research.HashIdentity{ID: relative, SHA256: byteHash(data)})
	}
	sort.Slice(inputs, func(i, j int) bool { return inputs[i].ID < inputs[j].ID })
	buildInputsSHA, err := canonicalHash(inputs)
	if err != nil {
		return research.RunnerImplementationIdentity{}, err
	}
	modeBytes := []byte(strings.Join([]string{"-mod=vendor", "-trimpath", "-buildvcs=false", "CGO_ENABLED=0", "GOOS=" + build.GOOS, "GOARCH=" + build.GOARCH, build.Package}, "\n"))
	compiler, err := goOutput(build.RepositoryPath, "version")
	if err != nil {
		return research.RunnerImplementationIdentity{}, err
	}
	temporary, err := os.MkdirTemp("", "ak-engine-pr4b0-r1p8-build-")
	if err != nil {
		return research.RunnerImplementationIdentity{}, err
	}
	defer os.RemoveAll(temporary)
	first := filepath.Join(temporary, "ak-engine.first")
	second := filepath.Join(temporary, "ak-engine.second")
	for _, output := range []string{first, second} {
		command := exec.Command("go", "build", "-mod=vendor", "-trimpath", "-buildvcs=false", "-o", output, build.Package)
		command.Dir = build.RepositoryPath
		command.Env = buildEnvironment(build.GOOS, build.GOARCH)
		if data, buildErr := command.CombinedOutput(); buildErr != nil {
			return research.RunnerImplementationIdentity{}, fmt.Errorf("deterministic production build failed: %w: %s", buildErr, strings.TrimSpace(string(data)))
		}
	}
	firstBytes, err := os.ReadFile(first)
	if err != nil {
		return research.RunnerImplementationIdentity{}, err
	}
	secondBytes, err := os.ReadFile(second)
	if err != nil {
		return research.RunnerImplementationIdentity{}, err
	}
	if !bytes.Equal(firstBytes, secondBytes) {
		return research.RunnerImplementationIdentity{}, errors.New("production runner build is not byte-deterministic")
	}
	return research.RunnerImplementationIdentity{SchemaVersion: research.RunnerIdentityVersion, SourceCommit: sourceCommit, PackageIdentity: research.HashIdentity{ID: "github.com/david22573/ak-engine/cmd/ak-engine", SHA256: byteHash(packageBytes)}, BuildInputsSHA256: buildInputsSHA, CompilerIdentity: compiler, BuildModeIdentity: research.HashIdentity{ID: productionBuildModeID, SHA256: byteHash(modeBytes)}, BinarySHA256: byteHash(firstBytes)}, nil
}

func goOutput(repository string, args ...string) (string, error) {
	command := exec.Command("go", args...)
	command.Dir = repository
	command.Env = buildEnvironment(runtime.GOOS, runtime.GOARCH)
	output, err := command.Output()
	return strings.TrimSpace(string(output)), err
}

func buildEnvironment(goos, goarch string) []string {
	result := []string{}
	for _, value := range os.Environ() {
		if strings.HasPrefix(value, "GOWORK=") || strings.HasPrefix(value, "CGO_ENABLED=") || strings.HasPrefix(value, "GOOS=") || strings.HasPrefix(value, "GOARCH=") {
			continue
		}
		result = append(result, value)
	}
	return append(result, "GOWORK=off", "CGO_ENABLED=0", "GOOS="+goos, "GOARCH="+goarch)
}
