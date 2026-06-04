package app

import (
	"os"
	"strings"
	"testing"
)

func TestMakefileHasProofBacktestLocalTarget(t *testing.T) {
	data, err := os.ReadFile("../../Makefile")
	if err != nil {
		t.Fatalf("ReadFile() error = %v", err)
	}

	content := string(data)
	required := []string{
		"proof-backtest-local: ci",
		"proof-fast-accumulation-local: ci",
		"go run ./cmd/ak-engine backtest",
		"--source local-json",
		"--path testdata/candles/btc_5m_sample.json",
		"--path testdata/candles/btc_5m_fast_accumulation_sample.json",
		"--strategy baseline",
		"--strategy fast_accumulation",
		"--format json",
	}
	for _, needle := range required {
		if !strings.Contains(content, needle) {
			t.Fatalf("Makefile missing %q", needle)
		}
	}
}
