package app

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestCompactMissingMetricDetection(t *testing.T) {
	metrics := FundingMetrics{
		EventCount:                 120,
		ClusterCount:               20,
		PFCombined_5bps:            1.2,
		ExpectancyCombined_5bpsBps: 2,
		CostStress: []FundingCostStressMetric{
			{CostBps: 5, ExpectancyBps: 2, PF: 1.2},
		},
	}
	delay := compactDelayReport{MissingMetrics: []string{"delay_1"}}
	missing := compactMissingRequiredMetrics(metrics, delay, []compactDimensionRow{{Key: "2025-01"}}, []compactDimensionRow{{Key: "2025-Q1"}})
	if !containsAny(missing, []string{"baseline_cost_bps", "cost_7_5", "cost_10", "delay_1"}) {
		t.Fatalf("missing=%v", missing)
	}
}

func TestCompactConcentrationAndLeaveOneOutFailures(t *testing.T) {
	rows := []compactDimensionRow{
		{Key: "A", EventCount: 50, ClusterCount: 10, GrossProfitBps: 100, GrossLossBps: 20, NetBps: 80, ExpectancyBps: 1.6, ProfitFactor: 5},
		{Key: "B", EventCount: 50, ClusterCount: 10, GrossProfitBps: 10, GrossLossBps: 40, NetBps: -30, ExpectancyBps: -0.6, ProfitFactor: 0.25},
	}
	addPositiveContribution(rows)
	if got := topPositiveContribution(rows); got <= 50 {
		t.Fatalf("top contribution=%f want >50", got)
	}
	loo := compactLeaveOneOut(rows)
	if loo.Pass {
		t.Fatalf("expected leave-one-out fail")
	}
	if loo.WorstRemaining == nil || loo.WorstRemaining.Pass {
		t.Fatalf("expected failing worst remaining row: %+v", loo.WorstRemaining)
	}
}

func TestCompactCostStressAndDelayDecision(t *testing.T) {
	report := compactCandidateReport{
		CandidateKey: "NegativeFundingLong|long|60m",
		SampleLabel:  "calibration_ready",
		Baseline: FundingMetrics{
			BaselineCostBps:            5,
			EventCount:                 120,
			ClusterCount:               20,
			PFCombined_5bps:            1.2,
			ExpectancyCombined_5bpsBps: 3,
		},
		CostStress: []FundingCostStressMetric{
			{CostBps: 5, ExpectancyBps: 3, PF: 1.2},
			{CostBps: 7.5, ExpectancyBps: 1, PF: 1.1},
			{CostBps: 10, ExpectancyBps: 0.2, PF: 1.01},
		},
		Concentration:      compactConcentrationReport{},
		LeaveOneSymbolOut:  compactLeaveOneOutReport{Supported: true, Pass: true},
		LeaveOneMonthOut:   compactLeaveOneOutReport{Supported: true, Pass: true},
		LeaveOneQuarterOut: compactLeaveOneOutReport{Supported: true, Pass: true},
		DelaySensitivity: compactDelayReport{
			Baseline:       &FundingDelayStressMetric{DelayCandles: 0, Label: "baseline", Available: true, ExpectancyBps: 3, PF: 1.2},
			Delay1:         &FundingDelayStressMetric{DelayCandles: 1, Label: "delay_1", Available: true, ExpectancyBps: 2.1, PF: 1.1},
			Delay1DecayPct: func() *float64 { v := 30.0; return &v }(),
		},
	}
	label, allowed, shadow, failures, _, _ := compactPromotionDecision(report, compactIntegrityReport{Status: "PASS"}, defaultCompactThresholds())
	if label != "SHADOW_CANDIDATE" || !allowed || !shadow || len(failures) != 0 {
		t.Fatalf("label=%s allowed=%t shadow=%t failures=%v", label, allowed, shadow, failures)
	}

	report.DelaySensitivity.Delay1.ExpectancyBps = -0.1
	label, allowed, shadow, failures, _, _ = compactPromotionDecision(report, compactIntegrityReport{Status: "PASS"}, defaultCompactThresholds())
	if label != "FRAGILE_RESEARCH_LEAD" || allowed || shadow || !containsAny(failures, []string{"delay_1_expectancy_non_positive"}) {
		t.Fatalf("label=%s allowed=%t shadow=%t failures=%v", label, allowed, shadow, failures)
	}
}

func TestCompactLabelAssignmentCases(t *testing.T) {
	base := compactCandidateReport{
		CandidateKey: "NegativeFundingLong|long|60m",
		SampleLabel:  "calibration_ready",
		Baseline: FundingMetrics{
			BaselineCostBps:            5,
			EventCount:                 120,
			ClusterCount:               20,
			PFCombined_5bps:            1.2,
			ExpectancyCombined_5bpsBps: 2,
		},
		CostStress: []FundingCostStressMetric{
			{CostBps: 5, ExpectancyBps: 2, PF: 1.2},
			{CostBps: 7.5, ExpectancyBps: 0.8, PF: 1.08},
			{CostBps: 10, ExpectancyBps: 0.1, PF: 1.01},
		},
		Concentration:      compactConcentrationReport{},
		LeaveOneSymbolOut:  compactLeaveOneOutReport{Supported: true, Pass: true},
		LeaveOneMonthOut:   compactLeaveOneOutReport{Supported: true, Pass: true},
		LeaveOneQuarterOut: compactLeaveOneOutReport{Supported: true, Pass: true},
		DelaySensitivity: compactDelayReport{
			Baseline:       &FundingDelayStressMetric{DelayCandles: 0, Label: "baseline", Available: true, ExpectancyBps: 2, PF: 1.2},
			Delay1:         &FundingDelayStressMetric{DelayCandles: 1, Label: "delay_1", Available: true, ExpectancyBps: 1.4, PF: 1.1},
			Delay1DecayPct: func() *float64 { v := 30.0; return &v }(),
		},
	}
	if label, _, _, _, _, _ := compactPromotionDecision(base, compactIntegrityReport{Status: "PASS"}, defaultCompactThresholds()); label != "SHADOW_CANDIDATE" {
		t.Fatalf("shadow case label=%s", label)
	}

	research := base
	research.CostStress[2] = FundingCostStressMetric{CostBps: 10, ExpectancyBps: -0.4, PF: 0.99}
	if label, allowed, shadow, _, warnings, _ := compactPromotionDecision(research, compactIntegrityReport{Status: "PASS"}, defaultCompactThresholds()); label != "RESEARCH_LEAD" || !allowed || shadow || len(warnings) == 0 {
		t.Fatalf("research label=%s allowed=%t shadow=%t warnings=%v", label, allowed, shadow, warnings)
	}

	fragile := base
	fragile.Concentration.TopMonthContributionPct = 55
	if label, allowed, shadow, failures, _, _ := compactPromotionDecision(fragile, compactIntegrityReport{Status: "PASS"}, defaultCompactThresholds()); label != "FRAGILE_RESEARCH_LEAD" || allowed || shadow || !containsAny(failures, []string{"concentration_month"}) {
		t.Fatalf("fragile label=%s allowed=%t shadow=%t failures=%v", label, allowed, shadow, failures)
	}

	rejected := base
	rejected.Baseline.ExpectancyCombined_5bpsBps = -1
	if label, allowed, shadow, failures, _, _ := compactPromotionDecision(rejected, compactIntegrityReport{Status: "PASS"}, defaultCompactThresholds()); label != "REJECTED" || allowed || shadow || !containsAny(failures, []string{"baseline_expectancy_non_positive"}) {
		t.Fatalf("rejected label=%s allowed=%t shadow=%t failures=%v", label, allowed, shadow, failures)
	}
}

func TestRetainedCoverageSingleSymbol(t *testing.T) {
	chunksDir := t.TempDir()
	writeRetainedTestChunk(t, chunksDir, "XRPUSDT", "2024-01", []FundingAlphaSummaryRow{retainedTestRow("XRPUSDT", "2024-01", "60m", 120, 20, 2.0, 1.2, 1.0, 0.4)}, FundingChunkSummary{Status: "PASS", EventCount: 120})
	writeRetainedTestChunk(t, chunksDir, "XRPUSDT", "2024-02", []FundingAlphaSummaryRow{retainedTestRow("XRPUSDT", "2024-02", "60m", 120, 20, 2.0, 1.2, 1.0, 0.4)}, FundingChunkSummary{Status: "PASS", EventCount: 120})

	scan, err := scanRetainedAlphaSummaries(chunksDir)
	if err != nil {
		t.Fatalf("scanRetainedAlphaSummaries error: %v", err)
	}
	report := buildRetainedCoverageReport(retainedCoverageInputs{
		ExpectedSymbols: []string{"ADAUSDT", "XRPUSDT"},
		ExpectedMonths:  []string{"2024-01", "2024-02"},
	}, scan, "FRAGILE_RESEARCH_LEAD")
	if report.CoverageStatus != "single_symbol_only" {
		t.Fatalf("coverage_status=%s", report.CoverageStatus)
	}
	if report.FullUniverseReady {
		t.Fatalf("full_universe_ready=true want false")
	}
	if got := strings.Join(report.FoundSymbols, ","); got != "XRPUSDT" {
		t.Fatalf("found_symbols=%s", got)
	}
	if !containsAny(report.MissingSymbols, []string{"ADAUSDT"}) {
		t.Fatalf("missing_symbols=%v", report.MissingSymbols)
	}
	if got := strings.Join(report.MonthsBySymbol["XRPUSDT"], ","); got != "2024-01,2024-02" {
		t.Fatalf("months_by_symbol=%s", got)
	}
}

func TestRetainedCoverageMultipleSymbolsFullUniverse(t *testing.T) {
	chunksDir := t.TempDir()
	writeRetainedTestChunk(t, chunksDir, "ADAUSDT", "2024-01", []FundingAlphaSummaryRow{retainedTestRow("ADAUSDT", "2024-01", "60m", 80, 12, 1.6, 1.1, 0.9, 0.2)}, FundingChunkSummary{Status: "PASS", EventCount: 80})
	writeRetainedTestChunk(t, chunksDir, "XRPUSDT", "2024-01", []FundingAlphaSummaryRow{retainedTestRow("XRPUSDT", "2024-01", "60m", 90, 14, 1.7, 1.12, 1.0, 0.3)}, FundingChunkSummary{Status: "PASS", EventCount: 90})

	scan, err := scanRetainedAlphaSummaries(chunksDir)
	if err != nil {
		t.Fatalf("scanRetainedAlphaSummaries error: %v", err)
	}
	report := buildRetainedCoverageReport(retainedCoverageInputs{
		ExpectedSymbols: []string{"ADAUSDT", "XRPUSDT"},
		ExpectedMonths:  []string{"2024-01"},
	}, scan, "RESEARCH_LEAD")
	if report.CoverageStatus != "full_universe_ready" {
		t.Fatalf("coverage_status=%s", report.CoverageStatus)
	}
	if !report.FullUniverseReady {
		t.Fatalf("full_universe_ready=false want true")
	}
	if len(report.MissingSymbols) != 0 {
		t.Fatalf("missing_symbols=%v", report.MissingSymbols)
	}
	if got := strings.Join(report.FamiliesFound, ","); got != "NegativeFundingLong" {
		t.Fatalf("families_found=%s", got)
	}
}

func TestRetainedCoverageMalformedSummaryHandling(t *testing.T) {
	chunksDir := t.TempDir()
	symbolDir := filepath.Join(chunksDir, "XRPUSDT")
	if err := os.MkdirAll(symbolDir, 0755); err != nil {
		t.Fatalf("mkdir error: %v", err)
	}
	if err := os.WriteFile(filepath.Join(symbolDir, "2024-01-alpha-summary.json"), []byte("{bad json"), 0644); err != nil {
		t.Fatalf("write malformed file error: %v", err)
	}

	scan, err := scanRetainedAlphaSummaries(chunksDir)
	if err != nil {
		t.Fatalf("scanRetainedAlphaSummaries error: %v", err)
	}
	report := buildRetainedCoverageReport(retainedCoverageInputs{
		ExpectedSymbols: []string{"XRPUSDT"},
		ExpectedMonths:  []string{"2024-01"},
	}, scan, "REJECTED")
	if report.MalformedFileCount != 1 {
		t.Fatalf("malformed_file_count=%d", report.MalformedFileCount)
	}
	if report.RawRequired {
		t.Fatalf("raw_required=true want false")
	}
}

func TestCompactCoverageOutputWritingAndInventoryCounts(t *testing.T) {
	reportsDir := t.TempDir()
	coverage := retainedCoverageReport{
		FinalLabel:        "FRAGILE_RESEARCH_LEAD",
		CoverageStatus:    "single_symbol_only",
		ExpectedSymbols:   []string{"ADAUSDT", "XRPUSDT"},
		ExpectedMonths:    []string{"2024-01"},
		FoundSymbols:      []string{"XRPUSDT"},
		MissingSymbols:    []string{"ADAUSDT"},
		MonthsBySymbol:    map[string][]string{"XRPUSDT": {"2024-01"}},
		SummaryFileCount:  1,
		FullUniverseReady: false,
		RawRequired:       false,
	}
	candidates := []compactCandidateReport{
		{CandidateKey: "a", FinalLabel: "SHADOW_CANDIDATE", Baseline: FundingMetrics{ExpectancyCombined_5bpsBps: 3, PFCombined_5bps: 1.3}, CostStress: []FundingCostStressMetric{{CostBps: 10, ExpectancyBps: 1.0}}, EventCount: 100, ClusterCount: 10},
		{CandidateKey: "b", FinalLabel: "RESEARCH_LEAD", Baseline: FundingMetrics{ExpectancyCombined_5bpsBps: 2, PFCombined_5bps: 1.2}, CostStress: []FundingCostStressMetric{{CostBps: 10, ExpectancyBps: -0.2}}, EventCount: 90, ClusterCount: 9},
		{CandidateKey: "c", FinalLabel: "FRAGILE_RESEARCH_LEAD", Baseline: FundingMetrics{ExpectancyCombined_5bpsBps: 1, PFCombined_5bps: 1.1}, CostStress: []FundingCostStressMetric{{CostBps: 10, ExpectancyBps: -0.5}}, EventCount: 80, ClusterCount: 8},
		{CandidateKey: "d", FinalLabel: "REJECTED", Baseline: FundingMetrics{ExpectancyCombined_5bpsBps: -1, PFCombined_5bps: 0.9}, EventCount: 70, ClusterCount: 7},
	}
	inventoryCfg := compactSummaryAnalyzerConfig{}
	inventory := buildRankedInventoryReport(candidates, coverage, "FRAGILE_RESEARCH_LEAD", inventoryCfg)
	if inventory.ShadowCandidateCount != 1 || inventory.ResearchLeadCount != 1 || inventory.FragileResearchLeadCount != 1 || inventory.RejectedCount != 1 {
		t.Fatalf("inventory counts=%v", inventory.InventoryLabelCounts)
	}
	if inventory.SymbolUniverse != "XRPUSDT" {
		t.Fatalf("symbol universe=%q", inventory.SymbolUniverse)
	}
	if inventory.CandidateScope != "all retained candidates" {
		t.Fatalf("candidate scope=%q", inventory.CandidateScope)
	}
	if err := writeCompactCoverageOutputs(reportsDir, coverage, inventory, inventoryCfg); err != nil {
		t.Fatalf("writeCompactCoverageOutputs error: %v", err)
	}
	mdBytes, err := os.ReadFile(filepath.Join(reportsDir, "phase10_8_ranked_inventory.md"))
	if err != nil {
		t.Fatalf("read markdown error: %v", err)
	}
	if !strings.Contains(string(mdBytes), "XRPUSDT-only retained universe") {
		t.Fatalf("markdown title=%s", string(mdBytes))
	}
	jsonBytes, err := os.ReadFile(filepath.Join(reportsDir, "phase10_8c_retained_coverage.json"))
	if err != nil {
		t.Fatalf("read coverage json error: %v", err)
	}
	var decoded retainedCoverageReport
	if err := json.Unmarshal(jsonBytes, &decoded); err != nil {
		t.Fatalf("coverage json unmarshal error: %v", err)
	}
	if decoded.RawRequired {
		t.Fatalf("decoded raw_required=true want false")
	}
}

func TestWriteCompactCoverageAndInventoryOutputsCoverageOnlyWithoutTargetCandidate(t *testing.T) {
	chunksDir := t.TempDir()
	reportsDir := t.TempDir()
	events := []FundingEventRow{
		fundingTestEvent("XRPUSDT", "FundingFlipShort", "short", 1704067200000, 12),
	}
	canonicalRows, err := fundingAlphaRowsFromEvents(events, "2024-01")
	if err != nil {
		t.Fatal(err)
	}
	writeRetainedTestChunk(t, chunksDir, "XRPUSDT", "2024-01", []FundingAlphaSummaryRow{
		fundingFindAlphaSummary(t, canonicalRows, "120m"),
	}, FundingChunkSummary{Status: "PASS", EventCount: 1})

	prevSymbols, prevFrom, prevTo := acompSymbols, acompFrom, acompTo
	prevInvFamily, prevInvSide, prevInvHorizon := ainvFamily, ainvSide, ainvHorizon
	t.Cleanup(func() {
		acompSymbols, acompFrom, acompTo = prevSymbols, prevFrom, prevTo
		ainvFamily, ainvSide, ainvHorizon = prevInvFamily, prevInvSide, prevInvHorizon
	})
	acompSymbols = "ADAUSDT,XRPUSDT"
	acompFrom = "2024-01"
	acompTo = "2024-01"
	ainvFamily, ainvSide, ainvHorizon = "", "", ""

	inventoryCfg, err := writeCompactCoverageAndInventoryOutputs(compactSummaryAnalyzerConfig{
		ChunksDir:  chunksDir,
		ReportsDir: reportsDir,
	}, "COVERAGE_ONLY")
	if err != nil {
		t.Fatalf("writeCompactCoverageAndInventoryOutputs error: %v", err)
	}
	if inventoryCfg.Family != "" || inventoryCfg.Side != "" || inventoryCfg.Horizon != "" {
		t.Fatalf("inventory cfg=%+v want unscoped inventory", inventoryCfg)
	}

	data, err := os.ReadFile(filepath.Join(reportsDir, "phase10_8c_retained_coverage.json"))
	if err != nil {
		t.Fatalf("read coverage json error: %v", err)
	}
	var coverage retainedCoverageReport
	if err := json.Unmarshal(data, &coverage); err != nil {
		t.Fatalf("unmarshal coverage error: %v", err)
	}
	if coverage.FinalLabel != "COVERAGE_ONLY" {
		t.Fatalf("final label=%q", coverage.FinalLabel)
	}
	if got := strings.Join(coverage.FoundSymbols, ","); got != "XRPUSDT" {
		t.Fatalf("found symbols=%q", got)
	}
	if got := strings.Join(coverage.MissingSymbols, ","); got != "ADAUSDT" {
		t.Fatalf("missing symbols=%q", got)
	}
}

func TestRankedInventoryOutputPathsScoped(t *testing.T) {
	reportsDir := t.TempDir()
	jsonPath, mdPath := rankedInventoryOutputPaths(reportsDir, compactSummaryAnalyzerConfig{
		Family:  "NegativeFundingLong",
		Side:    "long",
		Horizon: "60m",
	})
	if got, want := filepath.Base(jsonPath), "phase10_8_ranked_inventory_NegativeFundingLong_long_60m.json"; got != want {
		t.Fatalf("json path=%q want %q", got, want)
	}
	if got, want := filepath.Base(mdPath), "phase10_8_ranked_inventory_NegativeFundingLong_long_60m.md"; got != want {
		t.Fatalf("md path=%q want %q", got, want)
	}
}

func retainedTestRow(symbol, month, horizon string, eventCount, clusterCount int, expectancy, pf, stress7, stress10 float64) FundingAlphaSummaryRow {
	return FundingAlphaSummaryRow{
		Symbol:  symbol,
		Year:    month[:4],
		Quarter: quarterFromMonth(month),
		Month:   month,
		Family:  "NegativeFundingLong",
		Side:    "long",
		Horizon: horizon,
		Stats: FundingMetrics{
			BaselineCostBps:            5,
			EventCount:                 eventCount,
			RawEventCount:              eventCount,
			DeClusteredEventCount:      clusterCount,
			ClusterCount:               clusterCount,
			WinCount:                   eventCount / 2,
			LossCount:                  eventCount / 2,
			GrossProfitBps:             expectancy * float64(eventCount) * 1.8,
			GrossLossBps:               expectancy * float64(eventCount) * 0.8,
			PFCombined_5bps:            pf,
			ExpectancyCombined_5bpsBps: expectancy,
			CostStress: []FundingCostStressMetric{
				{CostBps: 5, EventCount: eventCount, DeClusteredEventCount: clusterCount, ExpectancyBps: expectancy, PF: pf},
				{CostBps: 7.5, EventCount: eventCount, DeClusteredEventCount: clusterCount, ExpectancyBps: stress7, PF: 1.05},
				{CostBps: 10, EventCount: eventCount, DeClusteredEventCount: clusterCount, ExpectancyBps: stress10, PF: 1.01},
			},
			DelayStress: []FundingDelayStressMetric{
				{DelayCandles: 0, Label: "baseline", Available: true, EventCount: eventCount, DeClusteredEventCount: clusterCount, ExpectancyBps: expectancy, PF: pf},
				{DelayCandles: 1, Label: "delay_1", Available: true, EventCount: eventCount, DeClusteredEventCount: clusterCount, ExpectancyBps: stress10, PF: 1.01},
			},
			FundingBucketCounts:    map[string]int{"negative_extreme": eventCount},
			RegimeBucketCounts:     map[string]int{"normal": eventCount},
			MarketBetaBucketCounts: map[string]int{"btc_flat": eventCount},
			LeakageStatus:          "PASS",
			EntryDelay1cAvailable:  true,
		},
	}
}

func writeRetainedTestChunk(t *testing.T, chunksDir, symbol, month string, rows []FundingAlphaSummaryRow, summary FundingChunkSummary) {
	t.Helper()
	symbolDir := filepath.Join(chunksDir, symbol)
	if err := os.MkdirAll(symbolDir, 0755); err != nil {
		t.Fatalf("mkdir error: %v", err)
	}
	alphaData, err := json.Marshal(rows)
	if err != nil {
		t.Fatalf("marshal alpha rows error: %v", err)
	}
	if err := os.WriteFile(filepath.Join(symbolDir, month+"-alpha-summary.json"), alphaData, 0644); err != nil {
		t.Fatalf("write alpha summary error: %v", err)
	}
	summaryData, err := json.Marshal(summary)
	if err != nil {
		t.Fatalf("marshal summary error: %v", err)
	}
	if err := os.WriteFile(filepath.Join(symbolDir, month+"-funding-summary.json"), summaryData, 0644); err != nil {
		t.Fatalf("write funding summary error: %v", err)
	}
}
