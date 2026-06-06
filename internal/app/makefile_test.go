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
		"proof-fast-accumulation-strict-local: ci",
		"proof-fast-accumulation-calibration-local: ci",
		"proof-fast-accumulation-economics-local: ci",
		"proof-fast-accumulation-entry-variants-local: ci",
		"proof-walk-forward-local: ci",
		"proof-walk-forward-strict-local: ci",
		"proof-walk-forward-calibration-local: ci",
		"go run ./cmd/ak-engine backtest",
		"go run ./cmd/ak-engine walk-forward",
		"--source local-json",
		"--path testdata/candles/btc_5m_sample.json",
		"--path testdata/candles/btc_5m_fast_accumulation_sample.json",
		"--path testdata/candles/btc_5m_walk_forward_sample.json",
		"--strategy baseline",
		"--strategy fast_accumulation",
		"--strategy fast_accumulation_strict",
		"--strategy fast_accumulation_strict_no_70_84_longs",
		"--strategy fast_accumulation_strict_low_frequency",
		"--strategy fast_accumulation_economics_guard",
		"--strategy fast_accumulation_pullback_reclaim",
		"--format json",
	}
	for _, needle := range required {
		if !strings.Contains(content, needle) {
			t.Fatalf("Makefile missing %q", needle)
		}
	}
}
