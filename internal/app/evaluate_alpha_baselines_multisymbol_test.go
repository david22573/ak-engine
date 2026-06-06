package app

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestEvaluateAlphaBaselinesMultisymbol(t *testing.T) {
	dir := t.TempDir()

	rep1 := BaselineReportJSON{
		Global: map[string]BaselineMetrics{
			"FamilyA_LONG": {EventCount: 100, Expectancy15: 0.005, ProfitFactor15: 1.5, WinRate15: 0.55, SampleWarning: "USABLE_SAMPLE"},
		},
	}
	rep2 := BaselineReportJSON{
		Global: map[string]BaselineMetrics{
			"FamilyB_SHORT": {EventCount: 200, Expectancy15: -0.001, ProfitFactor15: 0.9, WinRate15: 0.45, SampleWarning: "USABLE_SAMPLE"},
		},
	}

	data1, _ := json.Marshal(rep1)
	data2, _ := json.Marshal(rep2)

	os.WriteFile(filepath.Join(dir, "BTCUSDT-report.json"), data1, 0644)
	os.WriteFile(filepath.Join(dir, "ETHUSDT-report.json"), data2, 0644)

	outPath := filepath.Join(dir, "out.md")
	
	eabmDir = dir
	eabmOut = outPath

	err := evaluateAlphaBaselinesMultisymbolCmd.RunE(nil, nil)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	outMD, err := os.ReadFile(outPath)
	if err != nil {
		t.Fatalf("expected out.md to exist: %v", err)
	}
	content := string(outMD)
	if !strings.Contains(content, "BTCUSDT") || !strings.Contains(content, "ETHUSDT") {
		t.Errorf("expected both symbols in report")
	}
	if !strings.Contains(content, "FamilyA_LONG") {
		t.Errorf("expected FamilyA_LONG in report")
	}

	outJSON, err := os.ReadFile(filepath.Join(dir, "out.json"))
	if err != nil {
		t.Fatalf("expected out.json to exist: %v", err)
	}
	if !strings.Contains(string(outJSON), "BTCUSDT") {
		t.Errorf("expected BTCUSDT in json")
	}
}
