package app

import (
	"bytes"
	"encoding/json"
	"path/filepath"
	"testing"
)

func runWalkForwardCommand(t *testing.T, args []string) map[string]any {
	t.Helper()
	resetGlobals()

	var stdout bytes.Buffer
	rootCmd.SetOut(&stdout)
	rootCmd.SetErr(&stdout)
	rootCmd.SetArgs(args)
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("Execute() error = %v; output=%s", err, stdout.String())
	}

	var report map[string]any
	if err := json.Unmarshal(stdout.Bytes(), &report); err != nil {
		t.Fatalf("Unmarshal() error = %v; output=%s", err, stdout.String())
	}
	return report
}

func TestWalkForwardStrictCommandIncludesDiagnosticsAndStability(t *testing.T) {
	filePath := filepath.Clean("../../testdata/candles/btc_5m_walk_forward_sample.json")

	report := runWalkForwardCommand(t, []string{
		"walk-forward",
		"--source", "local-json",
		"--path", filePath,
		"--market", "futures-um",
		"--symbol", "BTCUSDT",
		"--interval", "5m",
		"--from", "2024-01-01",
		"--to", "2024-01-02",
		"--strategy", "fast_accumulation_strict",
		"--train-window", "45m",
		"--test-window", "15m",
		"--format", "json",
	})

	if got := report["strategy"]; got != "fast_accumulation_strict" {
		t.Fatalf("strategy = %#v, want fast_accumulation_strict", got)
	}

	for _, field := range []string{"aggregate_train", "aggregate_test", "splits", "candidate_stability"} {
		if _, ok := report[field]; !ok {
			t.Fatalf("report missing field %q", field)
		}
	}

	aggregateTrain := report["aggregate_train"].(map[string]any)
	diagnostics := aggregateTrain["diagnostics"].(map[string]any)
	for _, field := range []string{
		"pnl_by_action",
		"pnl_by_score_bucket",
		"win_rate_by_score_bucket",
		"avg_pnl_by_score_bucket",
		"pnl_by_reason_code",
		"losses_by_reason_code",
		"fees_by_action",
		"slippage_by_action",
		"long_vs_short_metrics",
		"hard_blocks_by_reason",
	} {
		if _, ok := diagnostics[field]; !ok {
			t.Fatalf("diagnostics missing %q", field)
		}
	}

	splits := report["splits"].([]any)
	if len(splits) == 0 {
		t.Fatal("expected at least one split")
	}
	split := splits[0].(map[string]any)
	for _, field := range []string{
		"best_train_candidate",
		"corresponding_test_result",
		"train_to_test_pnl_delta",
		"train_to_test_profit_factor_delta",
		"train_to_test_expectancy_delta",
	} {
		if _, ok := split[field]; !ok {
			t.Fatalf("split missing %q", field)
		}
	}
}

func TestWalkForwardCalibrationPresetSelection(t *testing.T) {
	filePath := filepath.Clean("../../testdata/candles/btc_5m_walk_forward_sample.json")

	report := runWalkForwardCommand(t, []string{
		"walk-forward",
		"--source", "local-json",
		"--path", filePath,
		"--market", "futures-um",
		"--symbol", "BTCUSDT",
		"--interval", "5m",
		"--from", "2024-01-01",
		"--to", "2024-01-02",
		"--strategy", "fast_accumulation_strict_low_frequency",
		"--train-window", "45m",
		"--test-window", "15m",
		"--min-trades", "0",
		"--format", "json",
	})

	if got := report["strategy"]; got != "fast_accumulation_strict_low_frequency" {
		t.Fatalf("strategy = %#v, want fast_accumulation_strict_low_frequency", got)
	}

	splits := report["splits"].([]any)
	if len(splits) == 0 {
		t.Fatal("expected at least one split")
	}
	firstSplit := splits[0].(map[string]any)
	selected := firstSplit["selected_candidates"].([]any)
	if len(selected) > 1 {
		t.Fatalf("expected fixed preset walk-forward to evaluate one candidate per split, got %d", len(selected))
	}
	if len(selected) == 1 {
		params := selected[0].(map[string]any)["params"].(map[string]any)
		if got := params["max_trades_per_day"]; got != float64(4) {
			t.Fatalf("max_trades_per_day = %#v, want 4", got)
		}
		if got := params["min_minutes_between_entries"]; got != float64(60) {
			t.Fatalf("min_minutes_between_entries = %#v, want 60", got)
		}
	}
}
