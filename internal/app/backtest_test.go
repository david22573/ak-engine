package app

import (
	"bytes"
	"encoding/json"
	"math"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestBacktestBaselineLocalJSON(t *testing.T) {
	filePath := writeBacktestFixture(t, `[
		{"open_time_ms":1704067200000,"open":100,"high":100,"low":100,"close":100,"volume":1,"close_time_ms":1704067499999,"interval":"5m"},
		{"open_time_ms":1704067500000,"open":100,"high":101,"low":100,"close":101,"volume":1,"close_time_ms":1704067799999,"interval":"5m"},
		{"open_time_ms":1704067800000,"open":101,"high":102,"low":101,"close":101.5,"volume":1,"close_time_ms":1704068099999,"interval":"5m"}
	]`)

	report := runBacktestCommand(t, []string{
		"backtest",
		"--source", "local-json",
		"--path", filePath,
		"--market", "futures-um",
		"--symbol", "BTCUSDT",
		"--interval", "5m",
		"--from", "2024-01-01",
		"--to", "2024-01-02",
		"--strategy", "baseline",
		"--baseline-threshold-bps", "0",
		"--baseline-stop-loss-bps", "10",
		"--baseline-take-profit-bps", "10",
		"--baseline-max-hold-candles", "1",
		"--starting-cash", "1000",
		"--max-position-size", "1",
		"--slippage-bps", "0",
		"--taker-fee-bps", "0",
		"--format", "json",
	})

	requiredFields := []string{
		"source",
		"market",
		"symbol",
		"interval",
		"strategy",
		"from_ms",
		"to_ms",
		"total_candles",
		"total_trades",
		"wins",
		"losses",
		"win_rate",
		"gross_pnl",
		"net_pnl",
		"fees_paid",
		"slippage_paid",
		"profit_factor",
		"max_drawdown",
		"max_consecutive_losses",
		"average_win",
		"average_loss",
		"expectancy",
		"average_hold_minutes",
		"status",
	}
	for _, field := range requiredFields {
		if _, ok := report[field]; !ok {
			t.Fatalf("report missing field %q: %#v", field, report)
		}
	}
	if got := report["strategy"]; got != "baseline" {
		t.Fatalf("strategy = %#v, want baseline", got)
	}
	if got := report["status"]; got != "PASS" {
		t.Fatalf("status = %#v, want PASS", got)
	}
	totalTrades, ok := report["total_trades"].(float64)
	if !ok {
		t.Fatalf("total_trades type = %T", report["total_trades"])
	}
	if totalTrades < 1 {
		t.Fatalf("total_trades = %v, want >= 1", totalTrades)
	}
	trades, ok := report["trades"].([]any)
	if !ok || len(trades) == 0 {
		t.Fatalf("trades missing or empty: %#v", report["trades"])
	}
	lastTrade := trades[len(trades)-1].(map[string]any)
	if got := lastTrade["exit_reason"]; got != "END_OF_DATA" {
		t.Fatalf("exit_reason = %#v, want END_OF_DATA", got)
	}
	if got := report["metrics"].(map[string]any)["open_position_count"]; got != float64(0) {
		t.Fatalf("open_position_count = %#v, want 0", got)
	}
}

func TestBacktestLosingRunStillPasses(t *testing.T) {
	filePath := writeBacktestFixture(t, `[
		{"open_time_ms":1704067200000,"open":100,"high":100,"low":100,"close":100,"volume":1,"close_time_ms":1704067499999,"interval":"5m"},
		{"open_time_ms":1704067500000,"open":100,"high":101,"low":100,"close":101,"volume":1,"close_time_ms":1704067799999,"interval":"5m"},
		{"open_time_ms":1704067800000,"open":101,"high":101,"low":100.4,"close":100.5,"volume":1,"close_time_ms":1704068099999,"interval":"5m"}
	]`)

	report := runBacktestCommand(t, []string{
		"backtest",
		"--source", "local-json",
		"--path", filePath,
		"--market", "futures-um",
		"--symbol", "BTCUSDT",
		"--interval", "5m",
		"--from", "2024-01-01",
		"--to", "2024-01-02",
		"--strategy", "baseline",
		"--baseline-threshold-bps", "0",
		"--baseline-stop-loss-bps", "10",
		"--baseline-take-profit-bps", "100",
		"--baseline-max-hold-candles", "5",
		"--starting-cash", "1000",
		"--max-position-size", "1",
		"--slippage-bps", "0",
		"--taker-fee-bps", "0",
		"--format", "json",
	})

	if got := report["status"]; got != "PASS" {
		t.Fatalf("status = %#v, want PASS", got)
	}
	netPnL, ok := report["net_pnl"].(float64)
	if !ok {
		t.Fatalf("net_pnl type = %T", report["net_pnl"])
	}
	if netPnL >= 0 {
		t.Fatalf("net_pnl = %f, want negative", netPnL)
	}
}

func TestBacktestFastAccumulationLocalJSON(t *testing.T) {
	filePath := writeBacktestFixture(t, `[
		{"open_time_ms":1704067200000,"open":100.0,"high":100.8,"low":99.9,"close":100.5,"volume":1,"close_time_ms":1704067499999,"interval":"5m"},
		{"open_time_ms":1704067500000,"open":100.5,"high":101.0,"low":100.4,"close":100.9,"volume":1,"close_time_ms":1704067799999,"interval":"5m"},
		{"open_time_ms":1704067800000,"open":100.9,"high":101.8,"low":100.8,"close":101.6,"volume":1,"close_time_ms":1704068099999,"interval":"5m"},
		{"open_time_ms":1704068100000,"open":101.6,"high":102.0,"low":101.3,"close":101.7,"volume":1,"close_time_ms":1704068399999,"interval":"5m"},
		{"open_time_ms":1704068400000,"open":101.7,"high":103.0,"low":101.6,"close":102.7,"volume":1,"close_time_ms":1704068699999,"interval":"5m"},
		{"open_time_ms":1704068700000,"open":102.7,"high":104.0,"low":102.6,"close":103.8,"volume":1,"close_time_ms":1704068999999,"interval":"5m"},
		{"open_time_ms":1704069000000,"open":103.8,"high":104.1,"low":103.0,"close":103.2,"volume":1,"close_time_ms":1704069299999,"interval":"5m"},
		{"open_time_ms":1704069300000,"open":103.2,"high":103.4,"low":102.4,"close":102.6,"volume":1,"close_time_ms":1704069599999,"interval":"5m"},
		{"open_time_ms":1704069600000,"open":102.6,"high":102.8,"low":101.8,"close":102.0,"volume":1,"close_time_ms":1704069899999,"interval":"5m"}
	]`)

	report := runBacktestCommand(t, []string{
		"backtest",
		"--source", "local-json",
		"--path", filePath,
		"--market", "futures-um",
		"--symbol", "BTCUSDT",
		"--interval", "5m",
		"--from", "2024-01-01",
		"--to", "2024-01-02",
		"--strategy", "fast_accumulation",
		"--slippage-bps", "1",
		"--taker-fee-bps", "5",
		"--fa-cost-multiple", "1",
		"--format", "json",
	})

	requiredFields := []string{
		"window_decision_count",
		"full_trade_count",
		"probe_trade_count",
		"hard_block_count",
		"hold_count",
		"exit_count",
		"reverse_count",
		"fast_accumulation",
	}
	for _, field := range requiredFields {
		if _, ok := report[field]; !ok {
			t.Fatalf("report missing field %q: %#v", field, report)
		}
	}
	if got := report["strategy"]; got != "fast_accumulation" {
		t.Fatalf("strategy = %#v, want fast_accumulation", got)
	}
	if got := report["window_decision_count"]; got != float64(3) {
		t.Fatalf("window_decision_count = %#v, want 3", got)
	}

	summary := report["fast_accumulation"].(map[string]any)["decision_summary"].(map[string]any)
	totalCounted := summary["full_trade_count"].(float64) +
		summary["probe_trade_count"].(float64) +
		summary["hard_block_count"].(float64) +
		summary["hold_count"].(float64) +
		summary["exit_count"].(float64) +
		summary["reverse_count"].(float64)
	if totalCounted != summary["window_decision_count"].(float64) {
		t.Fatalf("decision counts sum = %v, window_decision_count = %v", totalCounted, summary["window_decision_count"])
	}
}

func TestBacktestFastAccumulationLosingRunStillPasses(t *testing.T) {
	filePath := writeBacktestFixture(t, `[
		{"open_time_ms":1704067200000,"open":100.0,"high":101.0,"low":99.8,"close":100.8,"volume":1,"close_time_ms":1704067499999,"interval":"5m"},
		{"open_time_ms":1704067500000,"open":100.8,"high":101.4,"low":100.7,"close":101.2,"volume":1,"close_time_ms":1704067799999,"interval":"5m"},
		{"open_time_ms":1704067800000,"open":101.2,"high":102.2,"low":101.1,"close":102.0,"volume":1,"close_time_ms":1704068099999,"interval":"5m"},
		{"open_time_ms":1704068100000,"open":102.0,"high":102.4,"low":101.7,"close":101.9,"volume":1,"close_time_ms":1704068399999,"interval":"5m"},
		{"open_time_ms":1704068400000,"open":101.9,"high":103.0,"low":101.8,"close":102.8,"volume":1,"close_time_ms":1704068699999,"interval":"5m"},
		{"open_time_ms":1704068700000,"open":102.8,"high":103.8,"low":102.7,"close":103.6,"volume":1,"close_time_ms":1704068999999,"interval":"5m"},
		{"open_time_ms":1704069000000,"open":103.6,"high":103.7,"low":101.0,"close":101.4,"volume":1,"close_time_ms":1704069299999,"interval":"5m"},
		{"open_time_ms":1704069300000,"open":101.4,"high":101.5,"low":100.0,"close":100.2,"volume":1,"close_time_ms":1704069599999,"interval":"5m"},
		{"open_time_ms":1704069600000,"open":100.2,"high":100.3,"low":99.4,"close":99.6,"volume":1,"close_time_ms":1704069899999,"interval":"5m"}
	]`)

	report := runBacktestCommand(t, []string{
		"backtest",
		"--source", "local-json",
		"--path", filePath,
		"--market", "futures-um",
		"--symbol", "BTCUSDT",
		"--interval", "5m",
		"--from", "2024-01-01",
		"--to", "2024-01-02",
		"--strategy", "fast_accumulation",
		"--slippage-bps", "1",
		"--taker-fee-bps", "5",
		"--fa-cost-multiple", "1",
		"--format", "json",
	})

	if got := report["status"]; got != "PASS" {
		t.Fatalf("status = %#v, want PASS", got)
	}
	netPnL, ok := report["net_pnl"].(float64)
	if !ok {
		t.Fatalf("net_pnl type = %T", report["net_pnl"])
	}
	if netPnL >= 0 {
		t.Fatalf("net_pnl = %f, want negative", netPnL)
	}
}

func TestBacktestInvalidConfigFailsClearly(t *testing.T) {
	resetGlobals()
	filePath := writeBacktestFixture(t, `[
		{"open_time_ms":1704067200000,"open":100,"high":100,"low":100,"close":100,"volume":1,"close_time_ms":1704067499999,"interval":"5m"}
	]`)

	var stdout bytes.Buffer
	rootCmd.SetOut(&stdout)
	rootCmd.SetErr(&stdout)
	rootCmd.SetArgs([]string{
		"backtest",
		"--source", "local-json",
		"--path", filePath,
		"--market", "futures-um",
		"--symbol", "BTCUSDT",
		"--interval", "5m",
		"--strategy", "baseline",
		"--max-position-size", "2",
		"--format", "json",
	})

	err := rootCmd.Execute()
	if err == nil {
		t.Fatal("Execute() error = nil, want failure")
	}
	if !strings.Contains(err.Error(), "max_position_size must be in (0, 1]") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func writeBacktestFixture(t *testing.T, content string) string {
	t.Helper()

	tmpDir, err := os.MkdirTemp("", "test_backtest_*")
	if err != nil {
		t.Fatalf("MkdirTemp() error = %v", err)
	}
	t.Cleanup(func() { os.RemoveAll(tmpDir) })

	filePath := filepath.Join(tmpDir, "candles.json")
	if err := os.WriteFile(filePath, []byte(content), 0644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}
	return filePath
}

func runBacktestCommand(t *testing.T, args []string) map[string]any {
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

func resetGlobals() {
	source = ""
	path = ""
	market = ""
	symbol = ""
	interval = ""
	from = ""
	to = ""
	format = "json"

	backtestStrategy = "baseline"
	startingCash = 10000
	maxPositionSize = 1
	makerFeeBPS = 0
	takerFeeBPS = 5
	slippageBPS = 1
	includeDecisions = false
	faFullTradeMinScore = 85
	faNormalTradeMinScore = 70
	faProbeMinScore = 55
	faCostMultiple = 3
	faAllowProbe = true
	faForceFullTrade = false

	sweepStrategy = "fast_accumulation"
	sweepStartingCash = 10000
	sweepMaxPositionSize = 1
	sweepMakerFeeBPS = 0
	sweepTakerFeeBPS = 5
	sweepSlippageBPS = 1
}

func TestBacktestIncludeDecisions(t *testing.T) {
	filePath := writeBacktestFixture(t, `[
		{"open_time_ms":1704067200000,"open":100.0,"high":101.0,"low":99.8,"close":100.8,"volume":1,"close_time_ms":1704067499999,"interval":"5m"},
		{"open_time_ms":1704067500000,"open":100.8,"high":101.4,"low":100.7,"close":101.2,"volume":1,"close_time_ms":1704067799999,"interval":"5m"},
		{"open_time_ms":1704067800000,"open":101.2,"high":102.2,"low":101.1,"close":102.0,"volume":1,"close_time_ms":1704068099999,"interval":"5m"}
	]`)

	reportFalse := runBacktestCommand(t, []string{
		"backtest",
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
	if _, exists := reportFalse["decisions"]; exists {
		t.Fatal("decisions array should be omitted when --include-decisions=false")
	}

	reportTrue := runBacktestCommand(t, []string{
		"backtest",
		"--source", "local-json",
		"--path", filePath,
		"--market", "futures-um",
		"--symbol", "BTCUSDT",
		"--interval", "5m",
		"--from", "2024-01-01",
		"--to", "2024-01-02",
		"--strategy", "fast_accumulation",
		"--include-decisions",
		"--format", "json",
	})
	decisionsVal, exists := reportTrue["decisions"]
	if !exists {
		t.Fatal("decisions array should be present when --include-decisions=true")
	}
	decisionsList, ok := decisionsVal.([]any)
	if !ok || len(decisionsList) == 0 {
		t.Fatalf("decisions array should be non-empty slice, got type %T and len %v", decisionsVal, len(decisionsList))
	}

	d := decisionsList[0].(map[string]any)
	requiredDecisionFields := []string{
		"window_start_ms", "window_end_ms", "action", "confidence",
		"long_score", "short_score", "chop_score", "volatility_score",
		"trend_score", "pullback_score", "breakout_score", "expected_move_bps",
		"estimated_cost_bps", "reason_codes", "risk_fraction",
	}
	for _, field := range requiredDecisionFields {
		if _, ok := d[field]; !ok {
			t.Errorf("decision missing required export field %q", field)
		}
	}

	faReport := reportTrue["fast_accumulation"].(map[string]any)
	decSummary := faReport["decision_summary"].(map[string]any)
	var totalSummaryCount int
	for _, k := range []string{"full_trade_count", "probe_trade_count", "hard_block_count", "hold_count", "exit_count", "reverse_count"} {
		totalSummaryCount += int(decSummary[k].(float64))
	}
	if totalSummaryCount != len(decisionsList) {
		t.Errorf("total decisions summary count (%d) does not match decisions array len (%d)", totalSummaryCount, len(decisionsList))
	}
}

func TestBacktestDiagnosticsAndSummaryBuckets(t *testing.T) {
	filePath := writeBacktestFixture(t, `[
		{"open_time_ms":1704067200000,"open":100.0,"high":101.0,"low":99.8,"close":100.8,"volume":1,"close_time_ms":1704067499999,"interval":"5m"},
		{"open_time_ms":1704067500000,"open":100.8,"high":101.4,"low":100.7,"close":101.2,"volume":1,"close_time_ms":1704067799999,"interval":"5m"},
		{"open_time_ms":1704067800000,"open":101.2,"high":102.2,"low":101.1,"close":102.0,"volume":1,"close_time_ms":1704068099999,"interval":"5m"},
		{"open_time_ms":1704068100000,"open":102.0,"high":102.4,"low":101.7,"close":101.9,"volume":1,"close_time_ms":1704068399999,"interval":"5m"},
		{"open_time_ms":1704068400000,"open":101.9,"high":103.0,"low":101.8,"close":102.8,"volume":1,"close_time_ms":1704068699999,"interval":"5m"},
		{"open_time_ms":1704068700000,"open":102.8,"high":103.8,"low":102.7,"close":103.6,"volume":1,"close_time_ms":1704068999999,"interval":"5m"},
		{"open_time_ms":1704069000000,"open":103.6,"high":103.7,"low":101.0,"close":101.4,"volume":1,"close_time_ms":1704069299999,"interval":"5m"},
		{"open_time_ms":1704069300000,"open":101.4,"high":101.5,"low":100.0,"close":100.2,"volume":1,"close_time_ms":1704069599999,"interval":"5m"},
		{"open_time_ms":1704069600000,"open":100.2,"high":100.3,"low":99.4,"close":99.6,"volume":1,"close_time_ms":1704069899999,"interval":"5m"}
	]`)

	report := runBacktestCommand(t, []string{
		"backtest",
		"--source", "local-json",
		"--path", filePath,
		"--market", "futures-um",
		"--symbol", "BTCUSDT",
		"--interval", "5m",
		"--from", "2024-01-01",
		"--to", "2024-01-02",
		"--strategy", "fast_accumulation",
		"--slippage-bps", "1",
		"--taker-fee-bps", "5",
		"--fa-cost-multiple", "1",
		"--format", "json",
	})

	tradesList := report["trades"].([]any)
	if len(tradesList) == 0 {
		t.Fatal("expected at least one trade to check diagnostics")
	}

	trade := tradesList[0].(map[string]any)
	requiredTradeDiagFields := []string{
		"entry_window_ms", "exit_window_ms", "entry_reason_codes", "exit_reason",
		"score_at_entry", "risk_fraction", "estimated_cost_bps", "expected_move_bps",
		"r_multiple", "mae_bps", "mfe_bps", "hold_windows", "entry_action",
	}
	for _, field := range requiredTradeDiagFields {
		if _, ok := trade[field]; !ok {
			t.Errorf("trade missing diagnostic field: %q", field)
		}
	}

	maeBPS := trade["mae_bps"].(float64)
	mfeBPS := trade["mfe_bps"].(float64)
	if maeBPS < 0 || mfeBPS < 0 {
		t.Errorf("mae_bps (%f) or mfe_bps (%f) cannot be negative", maeBPS, mfeBPS)
	}

	holdWindows := trade["hold_windows"].(float64)
	if holdWindows < 0 {
		t.Errorf("hold_windows (%f) cannot be negative", holdWindows)
	}

	faReport := report["fast_accumulation"].(map[string]any)
	tradesByScore := faReport["trades_by_score_bucket"].(map[string]any)
	pnlByScore := faReport["pnl_by_score_bucket"].(map[string]any)
	avgPnLByScore := faReport["avg_pnl_by_score_bucket"].(map[string]any)
	winRateByScore := faReport["win_rate_by_score_bucket"].(map[string]any)

	scoreBuckets := []string{"0-39", "40-54", "55-69", "70-84", "85-100"}
	for _, b := range scoreBuckets {
		tradesCount := tradesByScore[b].(float64)
		pnl := pnlByScore[b].(float64)
		avgPnL := avgPnLByScore[b].(float64)
		winRate := winRateByScore[b].(float64)

		if tradesCount == 0 {
			if pnl != 0 || avgPnL != 0 || winRate != 0 {
				t.Errorf("score bucket %s has 0 trades but non-zero metrics: pnl=%f, avgPnL=%f, winRate=%f", b, pnl, avgPnL, winRate)
			}
		} else {
			expectedAvg := pnl / tradesCount
			if math.Abs(avgPnL-expectedAvg) > 1e-6 {
				t.Errorf("score bucket %s avg_pnl mismatch: got %f, want %f", b, avgPnL, expectedAvg)
			}
			if winRate < 0 || winRate > 1 {
				t.Errorf("score bucket %s win_rate (%f) must be in [0, 1]", b, winRate)
			}
		}
	}

	hardBlocks := faReport["hard_blocks_by_reason"].(map[string]any)
	if len(hardBlocks) == 0 {
		t.Log("no hard blocks occurred in this run")
	} else {
		for reason, count := range hardBlocks {
			if count.(float64) <= 0 {
				t.Errorf("hard block reason %q has non-positive count: %v", reason, count)
			}
		}
	}
}
