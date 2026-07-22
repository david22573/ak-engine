package main

import (
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"os"
	"path/filepath"

	"github.com/david22573/ak-engine/internal/epochorchestrator"
)

func main() {
	var parentConfig, preparedRoot, outputConfig, outputReport string
	var researchID, protocolID string
	var enginePath, engineCommit, rifPath, rifCommit, historianPath, historianCommit string
	flag.StringVar(&parentConfig, "parent-config", "", "immutable sealed v1 production configuration")
	flag.StringVar(&preparedRoot, "prepared-root", "", "new immutable child artifact store")
	flag.StringVar(&outputConfig, "output-config", "", "new sealed v2 configuration path")
	flag.StringVar(&outputReport, "output-report", "", "new boundary repair result JSON path")
	flag.StringVar(&researchID, "research-id", "", "fresh research identity")
	flag.StringVar(&protocolID, "protocol-id", "", "fresh protocol identity")
	flag.StringVar(&enginePath, "engine-path", "", "clean repaired Engine worktree")
	flag.StringVar(&engineCommit, "engine-commit", "", "repaired Engine commit")
	flag.StringVar(&rifPath, "rif-path", "", "clean RIF worktree")
	flag.StringVar(&rifCommit, "rif-commit", "", "RIF baseline commit")
	flag.StringVar(&historianPath, "historian-path", "", "clean Historian worktree")
	flag.StringVar(&historianCommit, "historian-commit", "", "Historian baseline commit")
	flag.Parse()

	for name, value := range map[string]string{"parent-config": parentConfig, "prepared-root": preparedRoot, "output-config": outputConfig, "output-report": outputReport, "research-id": researchID, "protocol-id": protocolID, "engine-path": enginePath, "engine-commit": engineCommit, "rif-path": rifPath, "rif-commit": rifCommit, "historian-path": historianPath, "historian-commit": historianCommit} {
		if value == "" {
			fatalf("-%s is required", name)
		}
	}
	if outputConfig == parentConfig {
		fatalf("output configuration must not overwrite the parent configuration")
	}
	data, err := os.ReadFile(parentConfig)
	if err != nil {
		fatalf("read parent config: %v", err)
	}
	parent, err := epochorchestrator.DecodeConfig(data)
	if err != nil {
		fatalf("decode parent config: %v", err)
	}
	baseline := epochorchestrator.BoundaryRepairBaseline{ResearchID: researchID, ProtocolID: protocolID, Engine: epochorchestrator.RepositoryCheck{Path: enginePath, Commit: engineCommit}, RIF: epochorchestrator.RepositoryCheck{Path: rifPath, Commit: rifCommit}, Historian: epochorchestrator.RepositoryCheck{Path: historianPath, Commit: historianCommit}}
	repaired, result, err := epochorchestrator.PrepareProductionBoundaryRepairConfig(parent, preparedRoot, baseline)
	if err != nil {
		fatalf("prepare production boundary repair: %v", err)
	}
	configBytes, err := epochorchestrator.EncodeConfig(repaired)
	if err != nil {
		fatalf("encode repaired config: %v", err)
	}
	reportBytes, err := json.MarshalIndent(result, "", "  ")
	if err != nil {
		fatalf("encode repair report: %v", err)
	}
	reportBytes = append(reportBytes, '\n')
	if err := writeNew(outputConfig, configBytes); err != nil {
		fatalf("write repaired config: %v", err)
	}
	if err := writeNew(outputReport, reportBytes); err != nil {
		fatalf("write repair report: %v", err)
	}
	fmt.Printf("config_sha256=%s\nresult_sha256=%s\nmemberships=%d\nunsafe_memberships=%d\nreal_registrations=0\nreal_accesses=0\nreal_outcomes=0\n", repaired.ConfigSHA256, result.ResultSHA256, result.Memberships, result.UnsafeMemberships)
}

func writeNew(path string, data []byte) error {
	if path == "" || filepath.Clean(path) != path || !filepath.IsAbs(path) {
		return errors.New("output path must be absolute and normalized")
	}
	file, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o400)
	if err != nil {
		return err
	}
	if _, err := file.Write(data); err != nil {
		_ = file.Close()
		return err
	}
	if err := file.Sync(); err != nil {
		_ = file.Close()
		return err
	}
	return file.Close()
}

func fatalf(format string, values ...any) {
	fmt.Fprintf(os.Stderr, format+"\n", values...)
	os.Exit(1)
}
