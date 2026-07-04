package app

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/davidmiguel22573/ak-engine/internal/features"
	"github.com/davidmiguel22573/ak-engine/internal/regime"
)

const fundingTestBaseMS int64 = 1735689600000

func TestFundingEvaluateChunkEmitsRealEventRowsFromRealInput(t *testing.T) {
	dir := t.TempDir()
	symbol := "LINKUSDT"
	featureFile := filepath.Join(dir, "features.json")
	contextFile := filepath.Join(dir, "context.json")
	chunksDir := filepath.Join(dir, "chunks")
	writeFundingRowsFixture(t, featureFile, fundingRowsWithCandidate(symbol, 20, -0.05, nil, false))
	writeFundingContextFixture(t, contextFile, symbol, "btc_up")

	summary, events, err := evaluateFundingChunkFiles(fundingChunkConfig{
		Symbol:            symbol,
		Month:             "2025-01",
		FeatureFile:       featureFile,
		ContextFile:       contextFile,
		ChunksDir:         chunksDir,
		RetainEventDetail: true,
		EventFormat:       "jsonl",
	})
	if err != nil {
		t.Fatal(err)
	}
	if summary.EventCount == 0 || len(events) == 0 {
		t.Fatalf("expected real event rows, summary=%+v", summary)
	}
	if !fundingHasFamily(events, "NegativeFundingLong") {
		t.Fatalf("expected NegativeFundingLong event, got %+v", events[:min(3, len(events))])
	}
	if events[0].EntryPrice == 0 || events[0].Return5mBps == 0 {
		t.Fatalf("event did not carry real price returns: %+v", events[0])
	}
	eventFile := filepath.Join(chunksDir, symbol, "2025-01-funding-events.jsonl")
	diskEvents, err := readFundingEventsJSONL(eventFile)
	if err != nil {
		t.Fatal(err)
	}
	if len(diskEvents) != len(events) {
		t.Fatalf("disk event count %d, want %d", len(diskEvents), len(events))
	}
}

func TestFundingUnknownRowsGenerateNoEvents(t *testing.T) {
	rows := fundingRowsWithCandidate("LINKUSDT", 20, -0.05, nil, true)
	summary := FundingChunkSummary{FamilyEventCounts: map[string]int{}, SideEventCounts: map[string]int{}, LeakageStatus: "PASS"}
	events := buildFundingEvents(rows, fundingContextFixture("LINKUSDT", "btc_up"), &summary)
	if len(events) != 0 {
		t.Fatalf("unknown funding rows emitted events: %+v", events)
	}
	if summary.RowsWithFundingUnknown == 0 {
		t.Fatalf("unknown funding rows not counted: %+v", summary)
	}
}

func TestFundingWarmupRowsGenerateNoEvents(t *testing.T) {
	rows := fundingRowsWithLength("LINKUSDT", 10, -0.05, nil, false)
	summary := FundingChunkSummary{FamilyEventCounts: map[string]int{}, SideEventCounts: map[string]int{}, LeakageStatus: "PASS"}
	events := buildFundingEvents(rows, fundingContextFixture("LINKUSDT", "btc_up"), &summary)
	if len(events) != 0 {
		t.Fatalf("warmup rows emitted events: %+v", events)
	}
	if summary.WarmupRows == 0 {
		t.Fatalf("warmup rows not counted: %+v", summary)
	}
}

func TestFundingNegativeFundingLongConditionWorks(t *testing.T) {
	rows := fundingRowsWithCandidate("LINKUSDT", 20, -0.05, nil, false)
	summary := FundingChunkSummary{FamilyEventCounts: map[string]int{}, SideEventCounts: map[string]int{}, LeakageStatus: "PASS"}
	events := buildFundingEvents(rows, fundingContextFixture("LINKUSDT", "btc_up"), &summary)
	if !fundingHasFamilySide(events, "NegativeFundingLong", "long") {
		t.Fatalf("negative funding condition did not emit long event: %+v", events)
	}
}

func TestFundingPositiveFundingShortConditionWorks(t *testing.T) {
	rows := fundingRowsWithCandidate("LINKUSDT", 20, 0.05, nil, false)
	summary := FundingChunkSummary{FamilyEventCounts: map[string]int{}, SideEventCounts: map[string]int{}, LeakageStatus: "PASS"}
	events := buildFundingEvents(rows, fundingContextFixture("LINKUSDT", "btc_down"), &summary)
	if !fundingHasFamilySide(events, "PositiveFundingShort", "short") {
		t.Fatalf("positive funding condition did not emit short event: %+v", events)
	}
}

func TestFundingConfirmedNegativeFundingLongConditionWorks(t *testing.T) {
	rows := fundingRowsWithCandidate("LINKUSDT", 20, -0.05, nil, false)
	rows[20].TrendSlope20 = 1.0 // Price confirmation
	summary := FundingChunkSummary{FamilyEventCounts: map[string]int{}, SideEventCounts: map[string]int{}, LeakageStatus: "PASS"}
	events := buildFundingEvents(rows, fundingContextFixture("LINKUSDT", "btc_up"), &summary)
	if !fundingHasFamilySide(events, "ConfirmedNegativeFundingLong", "long") {
		t.Fatalf("confirmed negative funding condition did not emit long event: %+v", events)
	}
}

func TestFundingConfirmedPositiveFundingShortConditionWorks(t *testing.T) {
	rows := fundingRowsWithCandidate("LINKUSDT", 20, 0.05, nil, false)
	rows[20].TrendSlope20 = -1.0 // Price confirmation
	summary := FundingChunkSummary{FamilyEventCounts: map[string]int{}, SideEventCounts: map[string]int{}, LeakageStatus: "PASS"}
	events := buildFundingEvents(rows, fundingContextFixture("LINKUSDT", "btc_down"), &summary)
	if !fundingHasFamilySide(events, "ConfirmedPositiveFundingShort", "short") {
		t.Fatalf("confirmed positive funding condition did not emit short event: %+v", events)
	}
}

func TestBreakoutFundingLongAcceptedWhenFundingBreakoutVolatilityAndVolumePass(t *testing.T) {
	rows := fundingRowsWithCandidate("LINKUSDT", 20, -0.05, nil, false)
	applyLongBreakoutFundingFixture(&rows[20])
	summary := FundingChunkSummary{FamilyEventCounts: map[string]int{}, SideEventCounts: map[string]int{}, LeakageStatus: "PASS"}
	diagnostics := FundingDiagnostics{}
	events := buildFundingEventsWithDiagnostics(rows, fundingContextFixture("LINKUSDT", "btc_up"), &summary, &diagnostics)
	if !fundingHasFamilySide(events, "BreakoutFundingLong", "long") {
		t.Fatalf("breakout funding long did not emit: %+v", events)
	}
	event := fundingFindFamilySide(t, events, "BreakoutFundingLong", "long")
	if !fundingReasonsContain(event.SignalReasons, "funding_condition:pass", "breakout_confirmation:pass", "volatility_expansion:pass", "volume_confirmation:pass", "direction_trend_alignment:pass") {
		t.Fatalf("missing breakout reason codes: %+v", event.SignalReasons)
	}
	if diagnostics.BreakoutFundingLongEventsEmitted == 0 {
		t.Fatalf("breakout diagnostics did not count long event: %+v", diagnostics)
	}
}

func TestBreakoutFundingLongRejectedWhenPriceConfirmationFails(t *testing.T) {
	rows := fundingRowsWithCandidate("LINKUSDT", 20, -0.05, nil, false)
	applyLongBreakoutFundingFixture(&rows[20])
	rows[20].Return15 = -0.01
	rows[20].Close = 99
	rows[20].EMA20 = 100
	summary := FundingChunkSummary{FamilyEventCounts: map[string]int{}, SideEventCounts: map[string]int{}, LeakageStatus: "PASS"}
	diagnostics := FundingDiagnostics{}
	events := buildFundingEventsWithDiagnostics(rows, fundingContextFixture("LINKUSDT", "btc_up"), &summary, &diagnostics)
	if fundingHasFamilySide(events, "BreakoutFundingLong", "long") {
		t.Fatalf("breakout funding long emitted despite failed price confirmation: %+v", events)
	}
	if diagnostics.BreakoutRejectedPriceConfirmation == 0 {
		t.Fatalf("breakout diagnostics did not count price rejection: %+v", diagnostics)
	}
	if !fundingHasFamilySide(events, "NegativeFundingLong", "long") {
		t.Fatalf("base funding family regressed: %+v", events)
	}
}

func TestBreakoutFundingShortAcceptedWhenFundingBreakoutVolatilityAndVolumePass(t *testing.T) {
	rows := fundingRowsWithCandidate("LINKUSDT", 20, 0.05, nil, false)
	applyShortBreakoutFundingFixture(&rows[20])
	summary := FundingChunkSummary{FamilyEventCounts: map[string]int{}, SideEventCounts: map[string]int{}, LeakageStatus: "PASS"}
	diagnostics := FundingDiagnostics{}
	events := buildFundingEventsWithDiagnostics(rows, fundingContextFixture("LINKUSDT", "btc_down"), &summary, &diagnostics)
	if !fundingHasFamilySide(events, "BreakoutFundingShort", "short") {
		t.Fatalf("breakout funding short did not emit: %+v", events)
	}
	event := fundingFindFamilySide(t, events, "BreakoutFundingShort", "short")
	if !fundingReasonsContain(event.SignalReasons, "funding_condition:pass", "breakout_confirmation:pass", "volatility_expansion:pass", "volume_confirmation:pass", "direction_trend_alignment:pass") {
		t.Fatalf("missing breakout reason codes: %+v", event.SignalReasons)
	}
	if diagnostics.BreakoutFundingShortEventsEmitted == 0 {
		t.Fatalf("breakout diagnostics did not count short event: %+v", diagnostics)
	}
}

func TestBreakoutFundingShortRejectedWhenVolatilityOrVolumeFails(t *testing.T) {
	rows := fundingRowsWithCandidate("LINKUSDT", 20, 0.05, nil, false)
	applyShortBreakoutFundingFixture(&rows[20])
	rows[20].BBWidthPctRank60 = 0.20
	rows[20].VolumeRatio20 = 0.90
	summary := FundingChunkSummary{FamilyEventCounts: map[string]int{}, SideEventCounts: map[string]int{}, LeakageStatus: "PASS"}
	diagnostics := FundingDiagnostics{}
	events := buildFundingEventsWithDiagnostics(rows, fundingContextFixture("LINKUSDT", "btc_down"), &summary, &diagnostics)
	if fundingHasFamilySide(events, "BreakoutFundingShort", "short") {
		t.Fatalf("breakout funding short emitted despite failed volatility/volume confirmation: %+v", events)
	}
	if diagnostics.BreakoutRejectedVolatilityExpansion == 0 || diagnostics.BreakoutRejectedVolumeConfirmation == 0 {
		t.Fatalf("breakout diagnostics did not count volatility/volume rejection: %+v", diagnostics)
	}
	if !fundingHasFamilySide(events, "PositiveFundingShort", "short") {
		t.Fatalf("base funding family regressed: %+v", events)
	}
}

func TestFundingFlipConditionsWork(t *testing.T) {
	longChange := 0.004
	shortChange := -0.004
	rows := fundingRowsWithLength("LINKUSDT", 320, 0.001, nil, false)
	rows[20].Derivatives.FundingRate = floatPtr(0.002)
	rows[20].Derivatives.FundingRateChange = &longChange
	rows[20].Derivatives.FundingRateChangeUnknown = false
	rows[50].Derivatives.FundingRate = floatPtr(-0.002)
	rows[50].Derivatives.FundingRateChange = &shortChange
	rows[50].Derivatives.FundingRateChangeUnknown = false
	summary := FundingChunkSummary{FamilyEventCounts: map[string]int{}, SideEventCounts: map[string]int{}, LeakageStatus: "PASS"}
	events := buildFundingEvents(rows, fundingContextFixture("LINKUSDT", "btc_up"), &summary)
	if !fundingHasFamilySide(events, "FundingFlipLong", "long") {
		t.Fatalf("funding flip long missing: %+v", events)
	}
	if !fundingHasFamilySide(events, "FundingFlipShort", "short") {
		t.Fatalf("funding flip short missing: %+v", events)
	}
}

func TestFundingAggregatorReadsEventJSONLNotDummySummaries(t *testing.T) {
	dir := t.TempDir()
	chunkDir := filepath.Join(dir, "chunks", "AAAUSDT")
	if err := os.MkdirAll(chunkDir, 0755); err != nil {
		t.Fatal(err)
	}
	events := []FundingEventRow{
		fundingTestEvent("AAAUSDT", "NegativeFundingLong", "long", fundingTestBaseMS, 20),
		fundingTestEvent("AAAUSDT", "NegativeFundingLong", "long", fundingTestBaseMS+2*fundingClusterWindowMS, -5),
	}
	if err := writeFundingEventsJSONL(filepath.Join(chunkDir, "2025-01-funding-events.jsonl"), events); err != nil {
		t.Fatal(err)
	}
	dummy := `[{"symbol":"AAAUSDT","family":"NegativeFundingLong","side":"long","horizon":"60m","stats":{"event_count":999,"pf_after_5_bps":9.9}}]`
	if err := os.WriteFile(filepath.Join(chunkDir, "2025-01-alpha-summary.json"), []byte(dummy), 0644); err != nil {
		t.Fatal(err)
	}
	report, _, _, err := buildFundingAggregationReports(fundingAggregationConfig{
		Symbols:   []string{"AAAUSDT"},
		Months:    []string{"2025-01"},
		ChunksDir: filepath.Join(dir, "chunks"),
	})
	if err != nil {
		t.Fatal(err)
	}
	row := findFundingLeaderboardRow(t, report.Leaderboard, "AAAUSDT", "NegativeFundingLong")
	if row.EventCount != 2 {
		t.Fatalf("aggregator used dummy summary count, got %d", row.EventCount)
	}
}

func TestFundingPFExpectancyComputedFromEventRows(t *testing.T) {
	events := []FundingEventRow{
		fundingTestEvent("AAAUSDT", "NegativeFundingLong", "long", fundingTestBaseMS, 20),
		fundingTestEvent("AAAUSDT", "NegativeFundingLong", "long", fundingTestBaseMS+2*fundingClusterWindowMS, -5),
	}
	metrics := computeFundingMetrics(events, "60m")
	assertFundingFloatClose(t, metrics.PFAfter5Bps, 1.5)
	assertFundingFloatClose(t, metrics.ExpectancyBpsAfter5Bps, 2.5)
}

func TestFundingDeclusteringUsesEventTimestamps(t *testing.T) {
	events := []FundingEventRow{
		fundingTestEvent("AAAUSDT", "NegativeFundingLong", "long", fundingTestBaseMS, 10),
		fundingTestEvent("AAAUSDT", "NegativeFundingLong", "long", fundingTestBaseMS+30*60*1000, 10),
		fundingTestEvent("AAAUSDT", "NegativeFundingLong", "long", fundingTestBaseMS+2*fundingClusterWindowMS, 10),
	}
	clusters := deClusterFundingEvents(events)
	if len(clusters) != 2 {
		t.Fatalf("clusters=%d, want 2", len(clusters))
	}
}

func TestFundingDeclusteringKeepsCompositeKeysSeparate(t *testing.T) {
	events := []FundingEventRow{
		fundingTestEvent("AAAUSDT", "NegativeFundingLong", "long", fundingTestBaseMS, 10),
		fundingTestEvent("BBBUSDT", "NegativeFundingLong", "long", fundingTestBaseMS, 10),
		fundingTestEvent("AAAUSDT", "PositiveFundingShort", "short", fundingTestBaseMS+30*60*1000, 10),
	}
	clusters := deClusterFundingEvents(events)
	if len(clusters) != 3 {
		t.Fatalf("clusters=%d, want 3", len(clusters))
	}
}

func TestFundingIdenticalPerSymbolDummyMetricsTriggerIntegrityFailure(t *testing.T) {
	loaded := []fundingLoadedEventFile{
		{Symbol: "AAAUSDT", Month: "2025-01", Summary: FundingChunkSummary{EventCount: 10}},
		{Symbol: "BBBUSDT", Month: "2025-01", Summary: FundingChunkSummary{EventCount: 10}},
		{Symbol: "CCCUSDT", Month: "2025-01", Summary: FundingChunkSummary{EventCount: 10}},
		{Symbol: "DDDUSDT", Month: "2025-01", Summary: FundingChunkSummary{EventCount: 10}},
	}
	report := FundingLeaderboardReport{Summary: FundingReportSummary{EventFilesExpected: 4, EventFilesFound: 4}}
	audit := buildFundingEventIntegrityAudit(report, loaded, nil, fundingAggregationConfig{})
	if audit.Status != "FAIL" || !audit.EventCountRowsMissing {
		t.Fatalf("dummy metrics not rejected: %+v", audit)
	}
}

func TestFundingHardcodedDeepTotalsNotAllowed(t *testing.T) {
	report := FundingLeaderboardReport{
		Summary: FundingReportSummary{EventFilesFound: 1},
		Leaderboard: []FundingLeaderboardRow{
			{
				Symbol:  "AAAUSDT",
				Family:  "NegativeFundingLong",
				Side:    "long",
				Verdict: "inconclusive",
				FundingMetrics: FundingMetrics{
					EventCount:            2,
					RawEventCount:         2,
					DeClusteredEventCount: 2,
					LeakageStatus:         "PASS",
					PriceOnlyResult:       "positive",
				},
			},
		},
	}
	deep := buildFundingDeepReport(report, "NegativeFundingLong", "long")
	if deep.RawEventCount != 2 {
		t.Fatalf("deep count should come from leaderboard/event rows, got %d", deep.RawEventCount)
	}
	if !deep.HardcodedTotalsRemoved {
		t.Fatalf("deep report must mark constant prior totals removed")
	}
}

func TestNoAKTraderImportInFundingEventCode(t *testing.T) {
	files := []string{
		"funding_events.go",
		"funding_aggregation.go",
		"evaluate_funding_chunk.go",
		"aggregate_funding_alpha_baselines.go",
		"evaluate_funding_candidate_deep.go",
		"phase10_funding_event_pipeline.go",
	}
	for _, file := range files {
		data, err := os.ReadFile(file)
		if err != nil {
			t.Fatal(err)
		}
		forbidden := "ak" + "-" + "trader"
		if strings.Contains(string(data), forbidden) {
			t.Fatalf("forbidden repo reference in %s", file)
		}
	}
}

func fundingRowsWithCandidate(symbol string, candidateIndex int, candidateRate float64, candidateChange *float64, unknown bool) []ResearchFeatureRow {
	return fundingRowsWithLength(symbol, 320, candidateRate, candidateChange, unknown)
}

func fundingRowsWithLength(symbol string, length int, candidateRate float64, candidateChange *float64, unknown bool) []ResearchFeatureRow {
	rows := make([]ResearchFeatureRow, 0, length)
	for i := 0; i < length; i++ {
		rate := 0.001 + float64(i%20)*0.001
		if i >= 20 {
			rate = 0.010
		}
		if i == 20 {
			rate = candidateRate
		}
		changeUnknown := true
		var change *float64
		if i == 20 && candidateChange != nil {
			change = candidateChange
			changeUnknown = false
		}
		fundingUnknown := unknown
		var ratePtr *float64
		if !fundingUnknown {
			ratePtr = floatPtr(rate)
		}
		rows = append(rows, ResearchFeatureRow{
			Row: features.Row{
				Market:        "futures-um",
				Symbol:        symbol,
				Interval:      "1m",
				EventTimeMS:   fundingTestBaseMS + int64(i)*60*1000,
				AvailableAtMS: fundingTestBaseMS + int64(i)*60*1000,
				Close:         100 + float64(i)*0.1,
			},
			Derivatives: ResearchDerivativeFeatures{
				FundingRate:               ratePtr,
				FundingRateUnknown:        fundingUnknown,
				FundingRateChange:         change,
				FundingRateChangeUnknown:  changeUnknown,
				FundingRateZScoreUnknown:  true,
				OpenInterestChangeUnknown: true,
				TakerBuySellUnknown:       true,
				LongShortRatioUnknown:     true,
				TopTraderLongShortUnknown: true,
				PositioningUnknown:        true,
			},
		})
	}
	return rows
}

func fundingContextFixture(symbol, beta string) []regime.Label {
	return []regime.Label{
		{
			Market:        "futures-um",
			Symbol:        symbol,
			Interval:      "1m",
			EventTimeMS:   fundingTestBaseMS,
			AvailableAtMS: fundingTestBaseMS,
			Volatility:    "compressed",
			Trend:         "range",
			Liquidity:     "normal",
			MarketBeta:    beta,
			Composite:     "compressed_range",
		},
	}
}

func writeFundingRowsFixture(t *testing.T, path string, rows []ResearchFeatureRow) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		t.Fatal(err)
	}
	data, err := json.Marshal(rows)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, data, 0644); err != nil {
		t.Fatal(err)
	}
}

func writeFundingContextFixture(t *testing.T, path, symbol, beta string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		t.Fatal(err)
	}
	data, err := json.Marshal(fundingContextFixture(symbol, beta))
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, data, 0644); err != nil {
		t.Fatal(err)
	}
}

func fundingHasFamily(events []FundingEventRow, family string) bool {
	for _, event := range events {
		if event.Family == family {
			return true
		}
	}
	return false
}

func fundingHasFamilySide(events []FundingEventRow, family, side string) bool {
	for _, event := range events {
		if event.Family == family && event.Side == side {
			return true
		}
	}
	return false
}

func fundingFindFamilySide(t *testing.T, events []FundingEventRow, family, side string) FundingEventRow {
	t.Helper()
	for _, event := range events {
		if event.Family == family && event.Side == side {
			return event
		}
	}
	t.Fatalf("event not found for %s %s", family, side)
	return FundingEventRow{}
}

func fundingReasonsContain(reasons []string, want ...string) bool {
	seen := make(map[string]struct{})
	for _, reason := range reasons {
		seen[reason] = struct{}{}
	}
	for _, reason := range want {
		if _, ok := seen[reason]; !ok {
			return false
		}
	}
	return true
}

func applyLongBreakoutFundingFixture(row *ResearchFeatureRow) {
	row.Close = 101
	row.EMA20 = 100
	row.Return15 = 0.01
	row.TrendSlope20 = 0.25
	row.BBWidthPctRank60 = 0.75
	row.VolumeRatio20 = 1.20
}

func applyShortBreakoutFundingFixture(row *ResearchFeatureRow) {
	row.Close = 99
	row.EMA20 = 100
	row.Return15 = -0.01
	row.TrendSlope20 = -0.25
	row.BBWidthPctRank60 = 0.75
	row.VolumeRatio20 = 1.20
}

func fundingTestEvent(symbol, family, side string, ts int64, return60 float64) FundingEventRow {
	return FundingEventRow{
		Symbol:            symbol,
		Family:            family,
		Side:              side,
		EventTimeMS:       ts,
		AvailableAtMS:     ts,
		EntryPrice:        100,
		FundingRate:       -0.001,
		FundingRateZScore: -1.2,
		FundingBucket:     "negative_extreme",
		RegimeComposite:   "compressed_range",
		Volatility:        "compressed",
		Trend:             "range",
		Liquidity:         "normal",
		MarketBeta:        "btc_up",
		Return5mBps:       return60,
		Return15mBps:      return60,
		Return30mBps:      return60,
		Return60mBps:      return60,
		Return120mBps:     return60,
		Return240mBps:     return60,
		Return5m5bpsBps:   return60 - 5,
		Return15m5bpsBps:  return60 - 5,
		Return30m5bpsBps:  return60 - 5,
		Return60m5bpsBps:  return60 - 5,
		Return120m5bpsBps: return60 - 5,
		Return240m5bpsBps: return60 - 5,
		LeakageStatus:     "PASS",
	}
}

func findFundingLeaderboardRow(t *testing.T, rows []FundingLeaderboardRow, symbol, family string) FundingLeaderboardRow {
	t.Helper()
	for _, row := range rows {
		if row.Symbol == symbol && row.Family == family {
			return row
		}
	}
	t.Fatalf("row not found for %s %s", symbol, family)
	return FundingLeaderboardRow{}
}

func assertFundingFloatClose(t *testing.T, got, want float64) {
	t.Helper()
	if got < want-0.000001 || got > want+0.000001 {
		t.Fatalf("got %.9f, want %.9f", got, want)
	}
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
