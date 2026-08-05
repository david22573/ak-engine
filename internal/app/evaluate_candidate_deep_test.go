package app

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/david22573/ak-engine/internal/features"
	"github.com/david22573/ak-engine/internal/regime"
	"github.com/david22573/ak-engine/internal/researchidentity"
	"github.com/david22573/ak-engine/pkg/protocol"
	"github.com/spf13/cobra"
)

func TestPhase103InventoryExtractsOnlyResearchLeadRows(t *testing.T) {
	dir := t.TempDir()
	mdPath := filepath.Join(dir, "leaderboard.md")
	jsonPath := filepath.Join(dir, "leaderboard.json")
	if err := os.WriteFile(mdPath, []byte("# leaderboard\n"), 0644); err != nil {
		t.Fatalf("write md: %v", err)
	}
	leaderboard := LeaderboardJSON{
		VerdictCounts: map[string]int{verdictResearchLead: 2, verdictRejected: 1},
		Rows: []LeaderboardRow{
			{Symbol: "LINKUSDT", Family: "CompressionBreakout", Side: "SHORT", Verdict: verdictResearchLead, EventCount: 1000, H2PF5bps: 1.2, H2Expectancy5bpsBps: 1, FYPF5bps: 1.1, PositiveMonthCount: 3, EntryDelay1cExpectancyBps: 1, LeakageStatus: "PASS"},
			{Symbol: "SOLUSDT", Family: "ShockFade", Side: "LONG", Verdict: verdictResearchLead, EventCount: 2000, H2PF5bps: 1.3, H2Expectancy5bpsBps: 2, FYPF5bps: 1.2, PositiveMonthCount: 4, EntryDelay1cExpectancyBps: 2, LeakageStatus: "PASS"},
			{Symbol: "ETHUSDT", Family: "ShockFade", Side: "LONG", Verdict: verdictRejected, EventCount: 3000},
		},
	}
	data, err := json.Marshal(leaderboard)
	if err != nil {
		t.Fatalf("marshal leaderboard: %v", err)
	}
	if err := os.WriteFile(jsonPath, data, 0644); err != nil {
		t.Fatalf("write json: %v", err)
	}

	report, err := buildPhase103Inventory(mdPath, jsonPath)
	if err != nil {
		t.Fatalf("build inventory: %v", err)
	}
	if report.Summary.ResearchLeadCount != 2 {
		t.Fatalf("expected 2 research leads, got %d", report.Summary.ResearchLeadCount)
	}
	for _, row := range report.Rows {
		if row.Symbol == "ETHUSDT" {
			t.Fatalf("inventory included non-research-lead row")
		}
	}
}

func TestDeepShortSideReturnsSignedCorrectly(t *testing.T) {
	if got := deepSignedReturnBps(100, 99, "SHORT"); got != 100 {
		t.Fatalf("short profit signed wrong: got %.4f", got)
	}
	if got := deepSignedReturnBps(100, 101, "SHORT"); got != -100 {
		t.Fatalf("short loss signed wrong: got %.4f", got)
	}
}

func TestDeepShortSideMFEAndMAEComputedCorrectly(t *testing.T) {
	mfe, mae := deepMFEAndMAEBps(100, 101, 98, "SHORT")
	if mfe != 200 {
		t.Fatalf("short MFE = %.4f, want 200", mfe)
	}
	if mae != 100 {
		t.Fatalf("short MAE = %.4f, want 100", mae)
	}
}

func TestDeepShortBracketSimulationTPAndSL(t *testing.T) {
	win := simulateDeepBracketEvent([]protocol.Candle{
		{Close: 100, High: 100.01, Low: 99.80},
	}, "SHORT", 10, 5, 5)
	if win.Outcome != "win" || win.ReturnBps != 5 {
		t.Fatalf("expected short TP win net 5 bps, got outcome=%s return=%.4f", win.Outcome, win.ReturnBps)
	}

	loss := simulateDeepBracketEvent([]protocol.Candle{
		{Close: 100, High: 100.10, Low: 99.99},
	}, "SHORT", 10, 5, 5)
	if loss.Outcome != "loss" || loss.ReturnBps != -10 {
		t.Fatalf("expected short SL loss net -10 bps, got outcome=%s return=%.4f", loss.Outcome, loss.ReturnBps)
	}
}

func TestDeepBracketSameCandleTPSLAssumesSLFirst(t *testing.T) {
	res := simulateDeepBracketEvent([]protocol.Candle{
		{Close: 100, High: 100.10, Low: 99.80},
	}, "SHORT", 10, 5, 5)
	if res.Outcome != "loss" {
		t.Fatalf("same candle TP/SL must assume SL first, got %s", res.Outcome)
	}
}

func TestDeepSingleMonthContributionGateWorks(t *testing.T) {
	report := deepPassingGateReport()
	report.ComparisonMetrics.SingleMonthContributionPct = 60
	gate := findDeepGate(deepAcceptanceGates(report), "single_month_contribution_pct")
	if gate.Passed {
		t.Fatalf("single-month contribution > 50%% should fail")
	}
}

func TestDeepWorstQuarterPFGateWorks(t *testing.T) {
	report := deepPassingGateReport()
	report.ComparisonMetrics.WorstQuarterPFAfter5bps = 0.94
	gate := findDeepGate(deepAcceptanceGates(report), "Worst quarter PF after 5 bps")
	if gate.Passed {
		t.Fatalf("worst quarter PF < 0.95 should fail")
	}
}

func TestDeepH2FailureOverridesFYSuccess(t *testing.T) {
	report := deepPassingGateReport()
	report.ComparisonMetrics.H2PFAfter5bps = 0.90
	status := deepFinalStatus(deepAcceptanceGates(report))
	if status != phase103StatusRejected {
		t.Fatalf("H2 PF failure must reject, got %s", status)
	}
}

func TestDeepValidationStatusAllowlistExcludesRuntimePromotionTerms(t *testing.T) {
	for _, status := range []string{
		"approved",
		"promoted",
		"frozen",
		"paper_ready",
		"paper_eligible",
		"runtime_ready",
		"testnet_ready",
		"mainnet_ready",
		"authorized",
	} {
		if phase103AllowedStatuses[status] {
			t.Fatalf("forbidden status %q is allowed", status)
		}
	}
	for _, status := range []string{phase103StatusRejected, phase103StatusFragile, phase103StatusValidatedResearchLead, phase103StatusNeedsMoreData} {
		if !phase103AllowedStatuses[status] {
			t.Fatalf("allowed status %q missing", status)
		}
	}
}

func TestDeepResearchOutcomeClassificationIsUnchanged(t *testing.T) {
	passing := deepPassingGateReport()
	if got := deepFinalStatus(deepAcceptanceGates(passing)); got != phase103StatusValidatedResearchLead {
		t.Fatalf("passing research gates = %q, want %q", got, phase103StatusValidatedResearchLead)
	}

	rejected := deepPassingGateReport()
	rejected.ComparisonMetrics.H2PFAfter5bps = 0.9
	if got := deepFinalStatus(deepAcceptanceGates(rejected)); got != phase103StatusRejected {
		t.Fatalf("critical research gate failure = %q, want %q", got, phase103StatusRejected)
	}
}

func TestDeepConfiguredThresholdChangesGateBehavior(t *testing.T) {
	report := deepPassingGateReport()
	thresholds := researchidentity.GateThresholds{
		MinimumEvents: 301, MinimumH2PF: 1.10, MinimumH2ExpectancyBPS: 0,
		MinimumFYPF: 1.05, MinimumFYExpectancyBPS: 0, MinimumPositiveMonths: 3,
		MinimumDelayOneExpectancyBPS: 0, MaximumSingleMonthContribution: 50,
		MinimumWorstQuarterPF: 0.95, MinimumBracketPF: 0.50, MinimumBracketExpectancyBPS: -10,
	}
	gate := findDeepGate(deepAcceptanceGatesConfigured(report, thresholds), "event_count")
	if gate.Passed || gate.Threshold != ">= 301" {
		t.Fatalf("resolved threshold did not drive gate: %#v", gate)
	}
}

func TestDeepReturnSeriesCarriesExactUntruncatedEventTimestamps(t *testing.T) {
	candles := []protocol.Candle{
		{OpenTimeMS: 1_000, Close: 100},
		{OpenTimeMS: 61_000, Close: 101},
		{OpenTimeMS: 121_000, Close: 102},
	}
	events := []deepCandidateEvent{
		{CandleIndex: 0, EventTimeMS: 1_000, Side: "LONG"},
		{CandleIndex: 2, EventTimeMS: 121_000, Side: "LONG"},
	}
	set := deepReturnsForEvents(events, candles, 1, 0, 0, 0)
	if len(set.ReturnsBps) != 1 || len(set.EventTimestamps) != 1 || set.EventTimestamps[0] != events[0].EventTimeMS || set.TruncatedCount != 1 {
		t.Fatalf("return/timestamp truncation alignment failed: %#v", set)
	}
}

func TestDeepLateFeatureAvailabilityIsRejectedAsLeakage(t *testing.T) {
	row := features.Row{Market: "futures-um", Symbol: "BTCUSDT", Interval: "1m", EventTimeMS: 1_000, AvailableAtMS: 1_001, Close: 101, EMA20: 100}
	label := regime.Label{Market: "futures-um", Symbol: "BTCUSDT", Interval: "1m", EventTimeMS: 1_000, AvailableAtMS: 1_000, Volatility: "compressed", MarketBeta: "btc_up"}
	candle := protocol.Candle{Market: "futures-um", Symbol: "BTCUSDT", Interval: "1m", OpenTimeMS: 1_000, CloseTimeMS: 60_999, Open: 100, High: 101, Low: 99, Close: 100, Volume: 1}
	events, _, issues, err := generateDeepCandidateEvents([]features.Row{row}, []regime.Label{label}, []protocol.Candle{candle}, "CompressionBreakout", "LONG")
	if err != nil {
		t.Fatal(err)
	}
	if len(events) != 0 || len(issues) != 1 || !strings.Contains(issues[0], "feature available_at_ms") {
		t.Fatalf("late feature was not rejected as leakage: events=%v issues=%v", events, issues)
	}
}

func TestDeepProductionCallersUseFactOnlyResearchDiagnostics(t *testing.T) {
	data, err := os.ReadFile("evaluate_candidate_deep.go")
	if err != nil {
		t.Fatal(err)
	}
	source := string(data)
	if got := strings.Count(source, ".EmitResearchDiagnostics("); got != 2 {
		t.Fatalf("production diagnostics call count = %d, want 2", got)
	}
	for _, prohibited := range []string{
		"is" + "Promoted",
		"Evaluate" + "AndEmit(",
		"cand" + "hash",
		"config" + "hash",
		"promotion" + "_packet",
		"RIF " + "Integrity",
	} {
		if strings.Contains(source, prohibited) {
			t.Fatalf("production caller retains authority or placeholder path %q", prohibited)
		}
	}
	for _, cmd := range []*cobra.Command{evaluateCandidateDeepCmd, evaluateResearchLeadsDeepCmd} {
		flag := cmd.Flags().Lookup("research-diagnostics")
		if flag == nil || flag.DefValue != "false" {
			t.Fatalf("%s research diagnostics must be explicit opt-in", cmd.Use)
		}
		if configFlag := cmd.Flags().Lookup("research-config"); configFlag == nil {
			t.Fatalf("%s is missing strict research configuration input", cmd.Use)
		}
	}
}

func TestDeepNoAKTraderImport(t *testing.T) {
	root, err := filepath.Abs(filepath.Join("..", ".."))
	if err != nil {
		t.Fatalf("resolve repo root: %v", err)
	}
	forbiddenImport := "\"github.com/david22573/" + ("ak" + "-trader")
	err = filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			if d.Name() == ".git" || d.Name() == "runs" || d.Name() == "bin" {
				return filepath.SkipDir
			}
			return nil
		}
		if filepath.Ext(path) != ".go" {
			return nil
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		if strings.Contains(string(data), forbiddenImport) {
			t.Fatalf("unexpected forbidden import/reference in %s", path)
		}
		return nil
	})
	if err != nil {
		t.Fatalf("walk repo: %v", err)
	}
}

func deepPassingGateReport() DeepCandidateReport {
	return DeepCandidateReport{
		LeakageStatus: "PASS",
		Brackets: []DeepBracketMetric{
			{NotCatastrophicAfter5bps: true},
		},
		ComparisonMetrics: DeepComparisonMetrics{
			EventCount:                 300,
			FYPFAfter5bps:              1.05,
			H2PFAfter5bps:              1.10,
			WorstQuarterPFAfter5bps:    0.95,
			FYExpectancyBps:            0.01,
			H2ExpectancyBps:            0.01,
			EntryDelay1cExpectancyBps:  0.01,
			PositiveMonthCount:         3,
			SingleMonthContributionPct: 50,
			LeakageStatus:              "PASS",
		},
	}
}

func findDeepGate(gates []DeepGateResult, name string) DeepGateResult {
	for _, gate := range gates {
		if gate.Name == name {
			return gate
		}
	}
	return DeepGateResult{}
}
