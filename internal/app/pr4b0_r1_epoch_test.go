package app

import (
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/david22573/ak-engine/internal/epochorchestrator"
)

func TestProductionEpochCommandIsExplicitAndPreflightOnly(t *testing.T) {
	command, _, err := rootCmd.Find([]string{"pr4b0-r1-epoch"})
	if err != nil || command != pr4b0R1EpochCmd {
		t.Fatalf("production epoch command is unavailable: %v", err)
	}
	if err := runPR4B0R1Epoch("for x in variants", "missing", "missing"); err == nil || !strings.Contains(err.Error(), "unknown epoch operation") {
		t.Fatalf("undocumented shell-loop bypass was accepted: %v", err)
	}
	config, err := epochorchestrator.CreateSyntheticConfig(filepath.Join(t.TempDir(), "checkpoint"))
	if err != nil {
		t.Fatal(err)
	}
	encoded, err := epochorchestrator.EncodeConfig(config)
	if err != nil {
		t.Fatal(err)
	}
	configPath := filepath.Join(t.TempDir(), "epoch-config.json")
	if err := os.WriteFile(configPath, encoded, 0o600); err != nil {
		t.Fatal(err)
	}
	epochRoot := filepath.Join(t.TempDir(), "epoch")
	reader, writer, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	prior := os.Stdout
	os.Stdout = writer
	runErr := runPR4B0R1Epoch("preflight", configPath, epochRoot)
	_ = writer.Close()
	os.Stdout = prior
	output, readErr := io.ReadAll(reader)
	_ = reader.Close()
	if runErr != nil || readErr != nil || string(output) != epochorchestrator.ReadyStatus+"\n" {
		t.Fatalf("production preflight status=%q run=%v read=%v", output, runErr, readErr)
	}
	if _, err := os.Stat(filepath.Join(epochRoot, "rif-research-governance.json")); !os.IsNotExist(err) {
		t.Fatal("CLI preflight created governance state")
	}
}
