package app

import (
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/davidmiguel22573/ak-engine/internal/features"
	"github.com/davidmiguel22573/ak-engine/internal/regime"
)

func TestEphemeralPipelineEvaluatesBeforeCleanupAndDeletesHeavyFiles(t *testing.T) {
	dir := t.TempDir()
	cfg := ephemeralTestConfig(dir)
	status := &Phase10FundingEventChunkStatus{Symbol: "LINKUSDT", Month: "2025-01"}
	paths := phase10FundingEventPaths(cfg, status.Symbol, status.Month)
	var order []string
	steps := ephemeralTestSteps(t, fundingRowsWithCandidate(status.Symbol, 20, -0.05, nil, false), fundingContextFixture(status.Symbol, "btc_up"), &order)

	if err := processPhase10FundingEventChunk(cfg, steps, paths, status); err != nil {
		t.Fatal(err)
	}
	wantOrder := []string{"build", "regime", "join", "eval", "cleanup"}
	if !reflect.DeepEqual(order, wantOrder) {
		t.Fatalf("order=%v, want %v", order, wantOrder)
	}
	if status.EventRows == 0 {
		t.Fatalf("expected event rows, status=%+v", status)
	}
	for _, path := range []string{paths.FeatureContextFile, paths.RegimeContextFile, paths.FundingFeatureFile} {
		if _, err := os.Stat(path); !os.IsNotExist(err) {
			t.Fatalf("heavy file retained after reports_only cleanup: %s err=%v", path, err)
		}
	}
	if _, _, err := verifyFundingEventOutputs(paths.SummaryFile, paths.EventFile); err != nil {
		t.Fatalf("event outputs invalid after cleanup: %v", err)
	}
}

func TestEphemeralAggregationWorksAfterHeavyChunksDeleted(t *testing.T) {
	dir := t.TempDir()
	cfg := ephemeralTestConfig(dir)
	status := &Phase10FundingEventChunkStatus{Symbol: "LINKUSDT", Month: "2025-01"}
	paths := phase10FundingEventPaths(cfg, status.Symbol, status.Month)
	steps := ephemeralTestSteps(t, fundingRowsWithCandidate(status.Symbol, 20, -0.05, nil, false), fundingContextFixture(status.Symbol, "btc_up"), nil)

	if err := processPhase10FundingEventChunk(cfg, steps, paths, status); err != nil {
		t.Fatal(err)
	}
	report, _, _, err := buildFundingAggregationReports(fundingAggregationConfig{
		Symbols:   []string{status.Symbol},
		Months:    []string{status.Month},
		ChunksDir: cfg.ChunksDir,
	})
	if err != nil {
		t.Fatal(err)
	}
	if report.Summary.TotalEventRows == 0 {
		t.Fatalf("aggregate did not read event JSONL after cleanup: %+v", report.Summary)
	}
	if report.Summary.MissingEventFileCount != 0 {
		t.Fatalf("aggregate should not need heavy chunks: %+v", report.Summary)
	}
}

func TestEphemeralMissingInputChunkTriggersRebuildNotFinalMissingData(t *testing.T) {
	dir := t.TempDir()
	cfg := ephemeralTestConfig(dir)
	manifest := &Phase10FundingEventManifest{Chunks: map[string]*Phase10FundingEventChunkStatus{}}
	key := phase10FundingEventChunkKey("LINKUSDT", "2025-01")
	manifest.Chunks[key] = &Phase10FundingEventChunkStatus{
		Symbol:               "LINKUSDT",
		Month:                "2025-01",
		FeatureBuildStatus:   "DONE",
		RegimeClassifyStatus: "DONE",
		FundingJoinStatus:    "DONE",
	}
	savePhase10FundingEventManifest(cfg.ManifestPath, manifest)
	var order []string
	steps := ephemeralTestSteps(t, fundingRowsWithCandidate("LINKUSDT", 20, -0.05, nil, false), fundingContextFixture("LINKUSDT", "btc_up"), &order)

	report, err := runPhase10FundingEventPipeline(cfg, steps)
	if err != nil {
		t.Fatal(err)
	}
	if report.MissingInputMonths != 0 {
		t.Fatalf("rebuilt chunk treated as missing input: %+v", report)
	}
	if len(order) == 0 || order[0] != "build" {
		t.Fatalf("missing input did not trigger rebuild, order=%v", order)
	}
	summaryPath := filepath.Join(cfg.ChunksDir, "LINKUSDT", "2025-01-funding-summary.json")
	data, err := os.ReadFile(summaryPath)
	if err != nil {
		t.Fatal(err)
	}
	var summary FundingChunkSummary
	if err := json.Unmarshal(data, &summary); err != nil {
		t.Fatal(err)
	}
	if summary.Status == "missing_data" {
		t.Fatalf("rebuilt chunk reported missing_data: %+v", summary)
	}
}

func TestEphemeralZeroEventMonthDistinctFromMissingInput(t *testing.T) {
	dir := t.TempDir()
	cfg := ephemeralTestConfig(dir)
	var order []string
	unknownRows := fundingRowsWithCandidate("LINKUSDT", 20, -0.05, nil, true)
	steps := ephemeralTestSteps(t, unknownRows, fundingContextFixture("LINKUSDT", "btc_up"), &order)

	report, err := runPhase10FundingEventPipeline(cfg, steps)
	if err != nil {
		t.Fatal(err)
	}
	if report.Status != "real_no_events" {
		t.Fatalf("status=%s, want real_no_events: %+v", report.Status, report)
	}
	if report.ZeroEventMonths != 1 || report.MissingInputMonths != 0 {
		t.Fatalf("zero-event month not distinct from missing input: %+v", report)
	}
	summaryPath := filepath.Join(cfg.ChunksDir, "LINKUSDT", "2025-01-funding-summary.json")
	data, err := os.ReadFile(summaryPath)
	if err != nil {
		t.Fatal(err)
	}
	var summary FundingChunkSummary
	if err := json.Unmarshal(data, &summary); err != nil {
		t.Fatal(err)
	}
	if summary.Status != "zero_events" {
		t.Fatalf("summary status=%s, want zero_events", summary.Status)
	}
}

func TestFundingFutureFeatureJoinRejectedDuringEventGeneration(t *testing.T) {
	rows := fundingRowsWithCandidate("LINKUSDT", 20, -0.05, nil, false)
	rows[20].AvailableAtMS = rows[20].EventTimeMS - 1
	for i := 21; i < len(rows); i++ {
		rows[i].Derivatives.FundingRate = nil
		rows[i].Derivatives.FundingRateUnknown = true
	}
	summary := FundingChunkSummary{FamilyEventCounts: map[string]int{}, SideEventCounts: map[string]int{}, LeakageStatus: "PASS"}
	events := buildFundingEvents(rows, fundingContextFixture("LINKUSDT", "btc_up"), &summary)
	if len(events) != 0 {
		t.Fatalf("future funding feature emitted events: %+v", events)
	}
	if summary.FutureFundingJoinRowsRejected != 1 {
		t.Fatalf("future funding rows rejected=%d, want 1", summary.FutureFundingJoinRowsRejected)
	}
}

func TestFundingNoHardcodedTotalsAllowed(t *testing.T) {
	files := []string{
		"funding_events.go",
		"funding_aggregation.go",
		"evaluate_funding_candidate_deep.go",
		"phase10_funding_event_pipeline.go",
	}
	forbidden := []string{"41652", "249912", "124956", "hardcoded_total_event_count"}
	for _, file := range files {
		data, err := os.ReadFile(file)
		if err != nil {
			t.Fatal(err)
		}
		for _, value := range forbidden {
			if strings.Contains(string(data), value) {
				t.Fatalf("hardcoded funding total %q found in %s", value, file)
			}
		}
	}
}

func TestContextLINKUSDTBuildFeaturesUsesBTCAndETH(t *testing.T) {
	cfg := ephemeralTestConfig(t.TempDir())
	cfg.ContextSymbols = "BTCUSDT,ETHUSDT"
	status := &Phase10FundingEventChunkStatus{Symbol: "LINKUSDT", Month: "2025-01"}
	args := phase10BuildFeatureArgs(cfg, phase10FundingEventPaths(cfg, status.Symbol, status.Month), status)
	if got := argValue(args, "--context-symbols"); got != "BTCUSDT,ETHUSDT" {
		t.Fatalf("LINKUSDT context symbols=%q, want BTCUSDT,ETHUSDT", got)
	}
}

func TestContextETHUSDTBuildFeaturesUsesBTCOnly(t *testing.T) {
	cfg := ephemeralTestConfig(t.TempDir())
	cfg.ContextSymbols = "BTCUSDT,ETHUSDT"
	status := &Phase10FundingEventChunkStatus{Symbol: "ETHUSDT", Month: "2025-01"}
	args := phase10BuildFeatureArgs(cfg, phase10FundingEventPaths(cfg, status.Symbol, status.Month), status)
	if got := argValue(args, "--context-symbols"); got != "BTCUSDT" {
		t.Fatalf("ETHUSDT context symbols=%q, want BTCUSDT", got)
	}
}

func TestContextBTCUSDTIsUnsupportedContext(t *testing.T) {
	dir := t.TempDir()
	cfg := ephemeralTestConfig(dir)
	cfg.Symbols = []string{"BTCUSDT"}
	status := &Phase10FundingEventChunkStatus{Symbol: "BTCUSDT", Month: "2025-01"}
	paths := phase10FundingEventPaths(cfg, status.Symbol, status.Month)
	steps := ephemeralTestSteps(t, nil, nil, nil)

	if err := processPhase10FundingEventChunk(cfg, steps, paths, status); err != nil {
		t.Fatal(err)
	}
	if status.SummaryStatus != "unsupported_context" || status.ContextStatus != "SELF_CONTEXT_UNSUPPORTED" {
		t.Fatalf("BTC status=%+v, want unsupported self-context", status)
	}
	data, err := os.ReadFile(paths.ContextAuditFile)
	if err != nil {
		t.Fatal(err)
	}
	var audit Phase10FundingContextAudit
	if err := json.Unmarshal(data, &audit); err != nil {
		t.Fatal(err)
	}
	if audit.ContextStatus != "SELF_CONTEXT_UNSUPPORTED" {
		t.Fatalf("context status=%s, want SELF_CONTEXT_UNSUPPORTED", audit.ContextStatus)
	}
}

func TestContextBtcFlatIsValidFundingContext(t *testing.T) {
	rows := fundingRowsWithCandidate("LINKUSDT", 20, -0.05, nil, false)
	summary := FundingChunkSummary{FamilyEventCounts: map[string]int{}, SideEventCounts: map[string]int{}, LeakageStatus: "PASS"}
	events := buildFundingEvents(rows, fundingContextFixture("LINKUSDT", "btc_flat"), &summary)
	if summary.UnsupportedContextRows != 0 {
		t.Fatalf("btc_flat marked unsupported: %+v", summary)
	}
	if !fundingHasFamily(events, "NegativeFundingLong") {
		t.Fatalf("btc_flat should allow funding event evaluation, events=%+v", events[:min(3, len(events))])
	}
}

func TestContextMissingBTCProducesMissingBTCContext(t *testing.T) {
	dir := t.TempDir()
	cfg := ephemeralTestConfig(dir)
	status := &Phase10FundingEventChunkStatus{Symbol: "LINKUSDT", Month: "2025-01"}
	paths := phase10FundingEventPaths(cfg, status.Symbol, status.Month)
	rows := fundingRowsWithCandidate(status.Symbol, 20, -0.05, nil, false)
	for i := range rows {
		rows[i].BTCReturn60 = 0
		rows[i].ETHReturn60 = 0.01
	}
	writeFeatureRowsFixture(t, paths.FeatureContextFile, rows)
	writeRegimeLabelsFixture(t, paths.RegimeContextFile, fundingLabelsForRows(rows, "btc_flat"))

	audit, err := writePhase10FundingContextAudit(cfg, paths, status)
	if err != nil {
		t.Fatal(err)
	}
	if audit.ContextStatus != "MISSING_BTC_CONTEXT" {
		t.Fatalf("context status=%s, want MISSING_BTC_CONTEXT: %+v", audit.ContextStatus, audit)
	}
}

func TestFundingDiagnosticsCountersWritten(t *testing.T) {
	dir := t.TempDir()
	symbol := "LINKUSDT"
	featureFile := filepath.Join(dir, "features.json")
	contextFile := filepath.Join(dir, "context.json")
	chunksDir := filepath.Join(dir, "chunks")
	writeFundingRowsFixture(t, featureFile, fundingRowsWithCandidate(symbol, 20, -0.05, nil, false))
	writeFundingContextFixture(t, contextFile, symbol, "btc_up")

	summary, _, err := evaluateFundingChunkFiles(fundingChunkConfig{
		Symbol:      symbol,
		Month:       "2025-01",
		FeatureFile: featureFile,
		ContextFile: contextFile,
		ChunksDir:   chunksDir,
	})
	if err != nil {
		t.Fatal(err)
	}
	if summary.EventCount == 0 {
		t.Fatalf("fixture should emit events: %+v", summary)
	}
	diagnostics, err := readFundingDiagnostics(filepath.Join(chunksDir, symbol, "2025-01-funding-diagnostics.json"))
	if err != nil {
		t.Fatal(err)
	}
	if diagnostics.RowsSeen == 0 || diagnostics.RowsWithFunding == 0 || diagnostics.NegativeFundingEventsEmitted == 0 {
		t.Fatalf("diagnostics counters not populated: %+v", diagnostics)
	}
}

func TestFundingThresholdFailureDistinctFromUnsupportedContext(t *testing.T) {
	rows := fundingRowsWithCandidate("LINKUSDT", 20, 0.015, nil, false)
	summary := FundingChunkSummary{FamilyEventCounts: map[string]int{}, SideEventCounts: map[string]int{}, LeakageStatus: "PASS"}
	diagnostics := FundingDiagnostics{Symbol: "LINKUSDT", Month: "2025-01"}
	_ = buildFundingEventsWithDiagnostics(rows, fundingContextFixture("LINKUSDT", "btc_flat"), &summary, &diagnostics)
	if diagnostics.RowsContextUnsupported != 0 || summary.UnsupportedContextRows != 0 {
		t.Fatalf("threshold failure marked unsupported: summary=%+v diagnostics=%+v", summary, diagnostics)
	}
	if diagnostics.RejectedByFundingThreshold == 0 {
		t.Fatalf("expected funding threshold rejection, diagnostics=%+v", diagnostics)
	}
}

func ephemeralTestConfig(root string) phase10FundingEventPipelineConfig {
	cfg := phase10FundingEventPipelineConfig{
		RootDir:         root,
		Workdir:         root,
		Symbols:         []string{"LINKUSDT"},
		ContextSymbols:  "BTCUSDT,ETHUSDT",
		Months:          []string{"2025-01"},
		Chunk:           "monthly",
		MaxRows:         50000,
		RetainPolicy:    "reports_only",
		ManifestPath:    filepath.Join(root, phase10FundingEventManifestPath),
		ReportsDir:      filepath.Join(root, "runs", "reports"),
		ChunksDir:       filepath.Join(root, "runs", "reports", "chunks"),
		ContinueOnError: false,
		MinFreeGB:       0,
		DiskBudgetGB:    8,
		RetainEventDetail: true,
		EventFormat:     "jsonl.gz",
	}
	return normalizePhase10FundingEventPipelineConfig(cfg)
}

func ephemeralTestSteps(t *testing.T, rows []ResearchFeatureRow, labels []regime.Label, order *[]string) phase10FundingEventPipelineSteps {
	t.Helper()
	appendOrder := func(step string) {
		if order != nil {
			*order = append(*order, step)
		}
	}
	return phase10FundingEventPipelineSteps{
		BuildFeatureChunk: func(_ phase10FundingEventPipelineConfig, paths phase10FundingEventChunkPaths, status *Phase10FundingEventChunkStatus) error {
			appendOrder("build")
			featureRows := cloneFundingRows(rows)
			for i := range featureRows {
				if i >= 60 {
					featureRows[i].BTCReturn60 = 0.01
					featureRows[i].ETHReturn60 = 0.02
				}
			}
			writeFeatureRowsFixture(t, paths.FeatureContextFile, featureRows)
			status.FeatureRows = len(rows)
			status.FeatureBuildStatus = "DONE"
			return nil
		},
		ClassifyRegimeChunk: func(_ phase10FundingEventPipelineConfig, paths phase10FundingEventChunkPaths, status *Phase10FundingEventChunkStatus) error {
			appendOrder("regime")
			labelsToWrite := labels
			if len(labelsToWrite) != len(rows) && len(labelsToWrite) > 0 {
				labelsToWrite = fundingLabelsForRows(rows, labelsToWrite[0].MarketBeta)
			}
			writeRegimeLabelsFixture(t, paths.RegimeContextFile, labelsToWrite)
			status.RegimeRows = len(labelsToWrite)
			status.RegimeClassifyStatus = "DONE"
			return nil
		},
		JoinFundingChunk: func(_ phase10FundingEventPipelineConfig, paths phase10FundingEventChunkPaths, status *Phase10FundingEventChunkStatus) error {
			appendOrder("join")
			writeFundingRowsFixture(t, paths.FundingFeatureFile, rows)
			status.FundingJoinStatus = "DONE"
			return nil
		},
		EvaluateFunding: func(cfg phase10FundingEventPipelineConfig, paths phase10FundingEventChunkPaths, status *Phase10FundingEventChunkStatus) (FundingChunkSummary, error) {
			appendOrder("eval")
			for _, path := range []string{paths.FeatureContextFile, paths.RegimeContextFile, paths.FundingFeatureFile} {
				if _, err := os.Stat(path); err != nil {
					t.Fatalf("input missing before eval: %s: %v", path, err)
				}
			}
			return defaultPhase10EvaluateFundingChunk(cfg, paths, status)
		},
		CleanupChunk: func(cfg phase10FundingEventPipelineConfig, paths phase10FundingEventChunkPaths, status *Phase10FundingEventChunkStatus) error {
			appendOrder("cleanup")
			if _, _, err := verifyFundingEventOutputs(paths.SummaryFile, paths.EventFile); err != nil {
				t.Fatalf("event outputs not valid before cleanup: %v", err)
			}
			return defaultPhase10CleanupFundingChunk(cfg, paths, status)
		},
	}
}

func cloneFundingRows(rows []ResearchFeatureRow) []ResearchFeatureRow {
	out := append([]ResearchFeatureRow(nil), rows...)
	return out
}

func fundingLabelsForRows(rows []ResearchFeatureRow, beta string) []regime.Label {
	labels := make([]regime.Label, 0, len(rows))
	for _, row := range rows {
		labels = append(labels, regime.Label{
			Market:        row.Market,
			Symbol:        row.Symbol,
			Interval:      row.Interval,
			EventTimeMS:   row.EventTimeMS,
			AvailableAtMS: row.AvailableAtMS,
			Volatility:    "compressed",
			Trend:         "range",
			Liquidity:     "normal",
			MarketBeta:    beta,
			Composite:     "compressed_range",
		})
	}
	return labels
}

func argValue(args []string, name string) string {
	for i := 0; i+1 < len(args); i++ {
		if args[i] == name {
			return args[i+1]
		}
	}
	return ""
}

func writeFeatureRowsFixture(t *testing.T, path string, rows []ResearchFeatureRow) {
	t.Helper()
	baseRows := make([]features.Row, 0, len(rows))
	for _, row := range rows {
		baseRows = append(baseRows, row.Row)
	}
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		t.Fatal(err)
	}
	data, err := json.Marshal(baseRows)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, data, 0644); err != nil {
		t.Fatal(err)
	}
}

func writeRegimeLabelsFixture(t *testing.T, path string, labels []regime.Label) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		t.Fatal(err)
	}
	data, err := json.Marshal(labels)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, data, 0644); err != nil {
		t.Fatal(err)
	}
}
