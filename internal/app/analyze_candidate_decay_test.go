package app

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestAnalyzeCandidateDecay(t *testing.T) {
	dir := t.TempDir()

	h1 := CBReport{
		Horizons: map[string]CBMetrics{
			"60m": {EventCount: 100, WinRate: 0.55, ProfitFactor: 1.5, Expectancy: 0.005},
		},
		Haircuts: map[string]CBMetrics{
			"5_bps": {ProfitFactor: 1.2, Expectancy: 0.002},
		},
	}
	h2 := CBReport{
		Horizons: map[string]CBMetrics{
			"60m": {EventCount: 100, WinRate: 0.45, ProfitFactor: 0.9, Expectancy: -0.001},
		},
		Haircuts: map[string]CBMetrics{
			"5_bps": {ProfitFactor: 0.8, Expectancy: -0.003},
		},
	}

	h1Data, _ := json.Marshal(h1)
	h2Data, _ := json.Marshal(h2)

	h1Path := filepath.Join(dir, "h1.json")
	h2Path := filepath.Join(dir, "h2.json")
	outPath := filepath.Join(dir, "out.md")

	os.WriteFile(h1Path, h1Data, 0644)
	os.WriteFile(h2Path, h2Data, 0644)

	acdName = "Test_Decay"
	acdH1Path = h1Path
	acdH2Path = h2Path
	acdOutPath = outPath

	err := analyzeCandidateDecayCmd.RunE(nil, nil)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	outMD, err := os.ReadFile(outPath)
	if err != nil {
		t.Fatalf("expected out.md to exist: %v", err)
	}
	content := string(outMD)
	if !strings.Contains(content, "REJECTED") {
		t.Errorf("expected REJECTED in report")
	}

	outJSON, err := os.ReadFile(filepath.Join(dir, "out.json"))
	if err != nil {
		t.Fatalf("expected out.json to exist: %v", err)
	}
	if !strings.Contains(string(outJSON), "REJECTED") {
		t.Errorf("expected REJECTED in json")
	}
}
