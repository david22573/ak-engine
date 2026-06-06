package app

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"
)

func runSweepCommand(t *testing.T, args []string) []map[string]any {
	t.Helper()
	resetGlobals()

	var stdout bytes.Buffer
	rootCmd.SetOut(&stdout)
	rootCmd.SetErr(&stdout)
	rootCmd.SetArgs(args)
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("Execute() error = %v; output=%s", err, stdout.String())
	}

	var results []map[string]any
	if err := json.Unmarshal(stdout.Bytes(), &results); err != nil {
		t.Fatalf("Unmarshal() error = %v; output=%s", err, stdout.String())
	}
	return results
}

func TestSweepCommandSuccessAndSorting(t *testing.T) {
	filePath := writeBacktestFixture(t, `[
		{"open_time_ms":1704067200000,"open":100.0,"high":101.0,"low":99.8,"close":100.8,"volume":1,"close_time_ms":1704067499999,"interval":"5m"},
		{"open_time_ms":1704067500000,"open":100.8,"high":101.4,"low":100.7,"close":101.2,"volume":1,"close_time_ms":1704067799999,"interval":"5m"},
		{"open_time_ms":1704067800000,"open":101.2,"high":102.2,"low":101.1,"close":102.0,"volume":1,"close_time_ms":1704068099999,"interval":"5m"}
	]`)

	results := runSweepCommand(t, []string{
		"sweep",
		"--source", "local-json",
		"--path", filePath,
		"--market", "futures-um",
		"--symbol", "BTCUSDT",
		"--interval", "5m",
		"--from", "2024-01-01",
		"--to", "2024-01-02",
		"--strategy", "fast_accumulation",
		"--format", "json",
	})

	if len(results) == 0 {
		t.Fatal("expected parameter sweep to produce multiple run results")
	}

	if len(results) <= 1 {
		t.Fatalf("expected sweep to run multiple combinations, got %d", len(results))
	}

	for i := 1; i < len(results); i++ {
		prevPnL := results[i-1]["net_pnl"].(float64)
		currPnL := results[i]["net_pnl"].(float64)
		if currPnL > prevPnL {
			t.Errorf("sweep results not sorted by net_pnl desc: results[%d]=%f, results[%d]=%f", i-1, prevPnL, i, currPnL)
		}
	}

	for _, res := range results {
		if got := res["status"].(string); got != "PASS" {
			t.Errorf("expected status 'PASS', got %q", got)
		}
	}
}

func TestSweepCommandInvalidConfig(t *testing.T) {
	filePath := writeBacktestFixture(t, `[
		{"open_time_ms":1704067200000,"open":100,"high":100,"low":100,"close":100,"volume":1,"close_time_ms":1704067499999,"interval":"5m"}
	]`)

	resetGlobals()
	var stdout bytes.Buffer
	rootCmd.SetOut(&stdout)
	rootCmd.SetErr(&stdout)
	rootCmd.SetArgs([]string{
		"sweep",
		"--source", "local-json",
		"--path", filePath,
		"--market", "futures-um",
		"--symbol", "BTCUSDT",
		"--interval", "5m",
		"--strategy", "baseline",
		"--format", "json",
	})

	err := rootCmd.Execute()
	if err == nil {
		t.Fatal("expected sweep to fail for non-fast_accumulation strategy")
	}
	if !strings.Contains(err.Error(), "sweep command only supports Fast Accumulation strategies") {
		t.Fatalf("unexpected error: %v", err)
	}

	resetGlobals()
	rootCmd.SetArgs([]string{
		"sweep",
		"--path", filePath,
		"--market", "futures-um",
		"--symbol", "BTCUSDT",
		"--interval", "5m",
		"--strategy", "fast_accumulation",
		"--format", "json",
	})
	err = rootCmd.Execute()
	if err == nil {
		t.Fatal("expected sweep to fail for missing source")
	}
	if !strings.Contains(err.Error(), "missing source in request") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestSweepCommandStrictProfile(t *testing.T) {
	filePath := writeBacktestFixture(t, `[
		{"open_time_ms":1704067200000,"open":100.0,"high":100.8,"low":99.9,"close":100.5,"volume":1,"close_time_ms":1704067499999,"interval":"5m"},
		{"open_time_ms":1704067500000,"open":100.5,"high":101.0,"low":100.4,"close":100.9,"volume":1,"close_time_ms":1704067799999,"interval":"5m"},
		{"open_time_ms":1704067800000,"open":100.9,"high":101.8,"low":100.8,"close":101.6,"volume":1,"close_time_ms":1704068099999,"interval":"5m"},
		{"open_time_ms":1704068100000,"open":101.6,"high":102.0,"low":101.3,"close":101.7,"volume":1,"close_time_ms":1704068399999,"interval":"5m"},
		{"open_time_ms":1704068400000,"open":101.7,"high":103.0,"low":101.6,"close":102.7,"volume":1,"close_time_ms":1704068699999,"interval":"5m"},
		{"open_time_ms":1704068700000,"open":102.7,"high":104.0,"low":102.6,"close":103.8,"volume":1,"close_time_ms":1704068999999,"interval":"5m"}
	]`)

	results := runSweepCommand(t, []string{
		"sweep",
		"--source", "local-json",
		"--path", filePath,
		"--market", "futures-um",
		"--symbol", "BTCUSDT",
		"--interval", "5m",
		"--from", "2024-01-01",
		"--to", "2024-01-02",
		"--strategy", "fast_accumulation",
		"--sweep-profile", "strict",
		"--format", "json",
	})

	if len(results) == 0 {
		t.Fatal("expected strict sweep to produce results")
	}

	params := results[0]["params"].(map[string]any)
	if _, ok := params["max_chop_score"]; !ok {
		t.Fatal("strict params missing max_chop_score")
	}
	if _, ok := params["min_trend_score"]; !ok {
		t.Fatal("strict params missing min_trend_score")
	}
	if got := params["disable_probe_trades"]; got != true {
		t.Fatalf("disable_probe_trades = %#v, want true", got)
	}
	if got := params["require_expected_move_gt_cost_multiple"]; got != true {
		t.Fatalf("require_expected_move_gt_cost_multiple = %#v, want true", got)
	}
	if params["long_enabled"] == false && params["short_enabled"] == false {
		t.Fatal("strict sweep produced invalid both-disabled directional combo")
	}
}

func TestSweepCommandCalibrationProfileSkipsBothDisabledCombo(t *testing.T) {
	filePath := writeBacktestFixture(t, `[
		{"open_time_ms":1704067200000,"open":100.0,"high":100.8,"low":99.9,"close":100.5,"volume":1,"close_time_ms":1704067499999,"interval":"5m"},
		{"open_time_ms":1704067500000,"open":100.5,"high":101.0,"low":100.4,"close":100.9,"volume":1,"close_time_ms":1704067799999,"interval":"5m"},
		{"open_time_ms":1704067800000,"open":100.9,"high":101.8,"low":100.8,"close":101.6,"volume":1,"close_time_ms":1704068099999,"interval":"5m"},
		{"open_time_ms":1704068100000,"open":101.6,"high":102.0,"low":101.3,"close":101.7,"volume":1,"close_time_ms":1704068399999,"interval":"5m"},
		{"open_time_ms":1704068400000,"open":101.7,"high":103.0,"low":101.6,"close":102.7,"volume":1,"close_time_ms":1704068699999,"interval":"5m"},
		{"open_time_ms":1704068700000,"open":102.7,"high":104.0,"low":102.6,"close":103.8,"volume":1,"close_time_ms":1704068999999,"interval":"5m"}
	]`)

	results := runSweepCommand(t, []string{
		"sweep",
		"--source", "local-json",
		"--path", filePath,
		"--market", "futures-um",
		"--symbol", "BTCUSDT",
		"--interval", "5m",
		"--from", "2024-01-01",
		"--to", "2024-01-02",
		"--strategy", "fast_accumulation",
		"--sweep-profile", "calibration",
		"--format", "json",
	})

	if len(results) == 0 {
		t.Fatal("expected calibration sweep to produce results")
	}

	params := results[0]["params"].(map[string]any)
	for _, field := range []string{
		"long_min_entry_score",
		"short_min_entry_score",
		"long_cost_multiple_required",
		"short_cost_multiple_required",
		"disable_long_score_bucket_70_84",
		"max_trades_per_day",
		"min_minutes_between_entries",
	} {
		if _, ok := params[field]; !ok {
			t.Fatalf("calibration params missing %q", field)
		}
	}

	for _, res := range results {
		p := res["params"].(map[string]any)
		if p["long_enabled"] == false && p["short_enabled"] == false {
			t.Fatal("calibration sweep produced invalid both-disabled directional combo")
		}
	}
}

func TestSweepCommandCalibrationAllowsStrictPresetStrategy(t *testing.T) {
	filePath := writeBacktestFixture(t, `[
		{"open_time_ms":1704067200000,"open":100.0,"high":100.8,"low":99.9,"close":100.5,"volume":1,"close_time_ms":1704067499999,"interval":"5m"},
		{"open_time_ms":1704067500000,"open":100.5,"high":101.0,"low":100.4,"close":100.9,"volume":1,"close_time_ms":1704067799999,"interval":"5m"},
		{"open_time_ms":1704067800000,"open":100.9,"high":101.8,"low":100.8,"close":101.6,"volume":1,"close_time_ms":1704068099999,"interval":"5m"},
		{"open_time_ms":1704068100000,"open":101.6,"high":102.0,"low":101.3,"close":101.7,"volume":1,"close_time_ms":1704068399999,"interval":"5m"},
		{"open_time_ms":1704068400000,"open":101.7,"high":103.0,"low":101.6,"close":102.7,"volume":1,"close_time_ms":1704068699999,"interval":"5m"},
		{"open_time_ms":1704068700000,"open":102.7,"high":104.0,"low":102.6,"close":103.8,"volume":1,"close_time_ms":1704068999999,"interval":"5m"}
	]`)

	results := runSweepCommand(t, []string{
		"sweep",
		"--source", "local-json",
		"--path", filePath,
		"--market", "futures-um",
		"--symbol", "BTCUSDT",
		"--interval", "5m",
		"--from", "2024-01-01",
		"--to", "2024-01-02",
		"--strategy", "fast_accumulation_strict",
		"--sweep-profile", "calibration",
		"--format", "json",
	})

	if len(results) == 0 {
		t.Fatal("expected calibration sweep to accept fast_accumulation_strict")
	}
}
