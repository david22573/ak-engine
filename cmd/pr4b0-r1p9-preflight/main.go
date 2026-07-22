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
	var configPath, outputPath string
	flag.StringVar(&configPath, "config", "", "sealed v2 production configuration")
	flag.StringVar(&outputPath, "output", "", "new all-membership audit JSON path")
	flag.Parse()
	if configPath == "" || outputPath == "" {
		fatalf("-config and -output are required")
	}
	data, err := os.ReadFile(configPath)
	if err != nil {
		fatalf("read config: %v", err)
	}
	config, err := epochorchestrator.DecodeConfig(data)
	if err != nil {
		fatalf("decode config: %v", err)
	}
	audit, err := epochorchestrator.AuditBoundaryConfig(config)
	if err != nil {
		fatalf("boundary preflight: %v", err)
	}
	encoded, err := json.MarshalIndent(audit, "", "  ")
	if err != nil {
		fatalf("encode boundary preflight: %v", err)
	}
	encoded = append(encoded, '\n')
	if err := writeNew(outputPath, encoded); err != nil {
		fatalf("write boundary preflight: %v", err)
	}
	fmt.Printf("config_sha256=%s\naudit_sha256=%s\nmemberships=%d\nunsafe_memberships=%d\n", audit.ConfigSHA256, audit.AuditSHA256, audit.Memberships, audit.UnsafeMemberships)
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
