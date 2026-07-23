package app

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/david22573/ak-engine/internal/papersignal"
)

func TestPaperForwardObservationFlow(t *testing.T) {
	base := time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC)
	tmp := t.TempDir()
	passManifest := writePaperTestManifest(t, tmp, "pass_manifest.json", "COVERS_WINDOW")
	blockManifest := writePaperTestManifest(t, tmp, "block_manifest.json", "PIT_NOT_ELIGIBLE")
	snapshotDir := filepath.Join(tmp, "snapshots")
	mustWriteJSON(t, filepath.Join(snapshotDir, "BTCUSDT.json"), map[string]any{
		"symbol":          "BTCUSDT",
		"timestamp_utc":   base.Add(-2 * time.Hour).Format(time.RFC3339),
		"reference_price": 100.0,
		"snapshot_hash":   "snap-btc",
	})
	mustWriteJSON(t, filepath.Join(snapshotDir, "ETHUSDT.json"), map[string]any{
		"symbol":          "ETHUSDT",
		"timestamp_utc":   base.Add(-2 * time.Hour).Format(time.RFC3339),
		"reference_price": 100.0,
		"snapshot_hash":   "snap-eth",
	})

	t.Run("paper_forward_allowed_blocked_cap_and_jsonl", func(t *testing.T) {
		outDir := filepath.Join(tmp, "forward_allowed")
		journal := filepath.Join(tmp, "allowed.jsonl")
		runPaperForwardForTest(t, passManifest, snapshotDir, outDir, journal, "NegativeFundingLong", "BTCUSDT,ETHUSDT", 1, base.Add(-2*time.Hour))

		rows := readPaperRowsForTest(t, journal)
		if len(rows) != 1 {
			t.Fatalf("max-signals cap not enforced: got %d rows", len(rows))
		}
		if rows[0].SignalStatus != papersignal.StatusAllowed {
			t.Fatalf("allowed signal status = %s", rows[0].SignalStatus)
		}
		if rows[0].OutcomeDueAtUTC == "" || rows[0].OutcomeStatus != papersignal.OutcomePending {
			t.Fatalf("allowed row missing pending outcome fields: %#v", rows[0])
		}
		jsonPath, _ := papersignal.SignalArtifactPaths(outDir, rows[0].SignalID)
		if _, err := os.Stat(jsonPath); err != nil {
			t.Fatalf("signal artifact missing: %v", err)
		}
		var run papersignal.ForwardObservationRun
		mustReadJSON(t, filepath.Join(outDir, "forward_paper_observation_run.json"), &run)
		if run.GeneratedSignals != 1 || run.AllowedSignals != 1 || run.BlockedSignals != 0 {
			t.Fatalf("unexpected run counts: %#v", run)
		}
		for _, row := range rows {
			if row.SignalID == "" {
				t.Fatal("journal row missing signal id")
			}
		}

		blockOut := filepath.Join(tmp, "forward_blocked")
		blockJournal := filepath.Join(tmp, "blocked.jsonl")
		runPaperForwardForTest(t, blockManifest, snapshotDir, blockOut, blockJournal, "NegativeFundingLong", "BTCUSDT", 1, base.Add(-2*time.Hour))
		blockRows := readPaperRowsForTest(t, blockJournal)
		if len(blockRows) != 1 || blockRows[0].SignalStatus != papersignal.StatusBlockedByRIF {
			t.Fatalf("expected blocked RIF row, got %#v", blockRows)
		}
		if blockRows[0].OutcomeStatus != "" {
			t.Fatalf("blocked row should not carry a pending outcome: %#v", blockRows[0])
		}
		var blockedSignal papersignal.PaperSignal
		mustReadJSON(t, filepath.Join(blockOut, "paper_signal.json"), &blockedSignal)
		if blockedSignal.SignalStatus != papersignal.StatusBlockedByRIF {
			t.Fatalf("blocked artifact status = %s", blockedSignal.SignalStatus)
		}
	})

	t.Run("pending_scanner_due_not_due_and_insufficient", func(t *testing.T) {
		journal := filepath.Join(tmp, "grade.jsonl")
		rows := []papersignal.PaperJournalRow{
			paperTestRow("long-tp", "BTCUSDT", papersignal.SideLong, base.Add(-2*time.Hour), 60),
			paperTestRow("long-stop", "ETHUSDT", papersignal.SideLong, base.Add(-2*time.Hour), 60),
			paperTestRow("not-due", "BTCUSDT", papersignal.SideLong, base.Add(-5*time.Minute), 60),
		}
		if err := papersignal.WriteJournalAtomic(journal, rows); err != nil {
			t.Fatal(err)
		}
		marketDir := filepath.Join(tmp, "market")
		writePaperCandles(t, filepath.Join(marketDir, "BTCUSDT_1m.json"), base.Add(-2*time.Hour), []paperTestCandleSpec{
			{Offset: 30 * time.Minute, Open: 100, High: 101.5, Low: 99.8, Close: 101.2},
			{Offset: 60 * time.Minute, Open: 101.2, High: 101.3, Low: 100.9, Close: 101.0},
		})
		writePaperCandles(t, filepath.Join(marketDir, "ETHUSDT_1m.json"), base.Add(-2*time.Hour), []paperTestCandleSpec{
			{Offset: 30 * time.Minute, Open: 100, High: 100.2, Low: 99.0, Close: 99.2},
			{Offset: 60 * time.Minute, Open: 99.2, High: 99.4, Low: 99.1, Close: 99.3},
		})

		runPaperGradeForTest(t, journal, marketDir, filepath.Join(tmp, "outcomes"), base, 50)
		graded := readPaperRowsForTest(t, journal)
		if graded[0].OutcomeStatus != papersignal.OutcomeLongTPFirst {
			t.Fatalf("BTC outcome = %s", graded[0].OutcomeStatus)
		}
		if graded[1].OutcomeStatus != papersignal.OutcomeLongStopFirst {
			t.Fatalf("ETH outcome = %s", graded[1].OutcomeStatus)
		}
		if graded[2].OutcomeStatus != papersignal.OutcomePending {
			t.Fatalf("not-due outcome = %s", graded[2].OutcomeStatus)
		}

		noDataJournal := filepath.Join(tmp, "no_data.jsonl")
		if err := papersignal.WriteJournalAtomic(noDataJournal, []papersignal.PaperJournalRow{
			paperTestRow("no-data", "BTCUSDT", papersignal.SideLong, base.Add(-2*time.Hour), 60),
		}); err != nil {
			t.Fatal(err)
		}
		runPaperGradeForTest(t, noDataJournal, "", filepath.Join(tmp, "no_data_outcomes"), base, 50)
		noDataRows := readPaperRowsForTest(t, noDataJournal)
		if noDataRows[0].OutcomeStatus != papersignal.OutcomeInsufficientData {
			t.Fatalf("missing future data outcome = %s", noDataRows[0].OutcomeStatus)
		}
	})

	t.Run("shadow_readiness_rules", func(t *testing.T) {
		insufficientRows := make([]papersignal.PaperJournalRow, 0, 25)
		for i := 0; i < 25; i++ {
			insufficientRows = append(insufficientRows, paperGradedRow("insufficient", i, papersignal.OutcomeLongTPFirst, 20))
		}
		rep := buildShadowReadinessReport(insufficientRows, "", base)
		if rep.SampleSizeLabel != papersignal.SampleInsufficient || rep.ReadinessLabel != papersignal.ReadinessBlockedBySampleSize {
			t.Fatalf("unexpected insufficient readiness: %#v", rep)
		}

		earlyRows := make([]papersignal.PaperJournalRow, 0, 45)
		for i := 0; i < 45; i++ {
			earlyRows = append(earlyRows, paperGradedRow("early", i, papersignal.OutcomeLongTPFirst, 15))
		}
		rep = buildShadowReadinessReport(earlyRows, "", base)
		if rep.SampleSizeLabel != papersignal.SampleEarly || rep.ReadinessLabel != papersignal.ReadinessContinuePaper {
			t.Fatalf("unexpected early readiness: %#v", rep)
		}
		if strings.Contains(strings.ToLower(rep.Recommendation), "live trading") {
			t.Fatalf("recommendation should not recommend live trading: %q", rep.Recommendation)
		}
	})

	t.Run("safety_check_detects_forbidden_import", func(t *testing.T) {
		badFile := filepath.Join(tmp, "bad.go")
		if err := os.WriteFile(badFile, []byte("package bad\nimport _ \"github.com/example/broker/execution\"\n"), 0644); err != nil {
			t.Fatal(err)
		}
		report, err := runPaperForwardSafetyCheck([]string{badFile})
		if err != nil {
			t.Fatal(err)
		}
		if report.Status != "FAIL" || len(report.Findings) != 1 {
			t.Fatalf("expected forbidden import finding, got %#v", report)
		}
	})
}

func TestPaperForwardDoesNotChangeStrategyCalculations(t *testing.T) {
	entry := 100.0
	target, stop := paperTargetAndStop(papersignal.SideLong, entry, 100, 75)
	if target != 101.0 || stop != 99.25 {
		t.Fatalf("paper target/stop calculation drifted: target=%f stop=%f", target, stop)
	}
}

func runPaperForwardForTest(t *testing.T, manifest, snapshotDir, outDir, journal, candidate, symbols string, maxSignals int, generatedAt time.Time) {
	t.Helper()
	pfCandidate = candidate
	pfSymbols = symbols
	pfTimeframe = "1m"
	pfMarketType = "SPOT"
	pfDatasetManifest = manifest
	pfResearchLock = ""
	pfSnapshotDir = snapshotDir
	pfOutDir = outDir
	pfJournal = journal
	pfMode = papersignal.ModePaperReplay
	pfMaxSignals = maxSignals
	pfGeneratedAtUTC = generatedAt.UTC().Format(time.RFC3339)
	pfDryRun = false
	pfAllowRIFWarnings = false
	pfPaperOnly = true
	if err := paperForwardCmd.RunE(paperForwardCmd, []string{}); err != nil {
		t.Fatal(err)
	}
}

func runPaperGradeForTest(t *testing.T, journal, marketDir, outDir string, now time.Time, maxGrade int) {
	t.Helper()
	pfgJournal = journal
	pfgMarketDataRoot = marketDir
	pfgSnapshotDir = ""
	pfgOutDir = outDir
	pfgMaxGrade = maxGrade
	pfgNowUTC = now.UTC().Format(time.RFC3339)
	if err := paperForwardGradePendingCmd.RunE(paperForwardGradePendingCmd, []string{}); err != nil {
		t.Fatal(err)
	}
}

func writePaperTestManifest(t *testing.T, dir, name, pitStatus string) string {
	t.Helper()
	path := filepath.Join(dir, name)
	mustWriteJSON(t, path, map[string]any{
		"dataset_hash": "dataset-" + pitStatus,
		"hashes": map[string]any{
			"dataset_hash":  "dataset-" + pitStatus,
			"manifest_hash": "manifest-" + pitStatus,
		},
		"validation": map[string]any{"status": "PASS"},
		"survivorship": map[string]any{
			"universe_hash":                          "universe-hash",
			"lifecycle_hash":                         "lifecycle-hash",
			"point_in_time_coverage_hash":            "pit-hash",
			"point_in_time_coverage_status":          pitStatus,
			"point_in_time_promotion_recommendation": "ALLOW_STRICT_PROMOTION",
		},
	})
	return path
}

func paperTestRow(id, symbol string, side papersignal.SignalSide, generatedAt time.Time, windowMinutes int) papersignal.PaperJournalRow {
	entry := 100.0
	target, stop := paperTargetAndStop(side, entry, 100, 75)
	return papersignal.PaperJournalRow{
		SignalID:             id,
		CandidateID:          "NegativeFundingLong",
		GeneratedAtUTC:       generatedAt.UTC().Format(time.RFC3339),
		Symbol:               symbol,
		MarketType:           "SPOT",
		Timeframe:            "1m",
		Side:                 side,
		SignalStatus:         papersignal.StatusAllowed,
		EntryReferencePrice:  entry,
		TargetReferencePrice: &target,
		StopReferencePrice:   &stop,
		ObservationWindow:    windowMinutes,
		OutcomeDueAtUTC:      generatedAt.Add(time.Duration(windowMinutes) * time.Minute).UTC().Format(time.RFC3339),
		OutcomeStatus:        papersignal.OutcomePending,
		RIFStatus:            "RIF_PASS",
	}
}

func paperGradedRow(prefix string, idx int, outcome papersignal.OutcomeStatus, ret float64) papersignal.PaperJournalRow {
	return papersignal.PaperJournalRow{
		SignalID:         prefix + "-" + time.Unix(int64(idx), 0).UTC().Format("150405"),
		CandidateID:      "NegativeFundingLong",
		SignalStatus:     papersignal.StatusAllowed,
		OutcomeStatus:    outcome,
		OutcomeReturnBPS: &ret,
	}
}

type paperTestCandleSpec struct {
	Offset time.Duration
	Open   float64
	High   float64
	Low    float64
	Close  float64
}

func writePaperCandles(t *testing.T, path string, generatedAt time.Time, specs []paperTestCandleSpec) {
	t.Helper()
	var rows []map[string]any
	for _, spec := range specs {
		openTime := generatedAt.Add(spec.Offset)
		rows = append(rows, map[string]any{
			"market":        "SPOT",
			"symbol":        "BTCUSDT",
			"interval":      "1m",
			"open_time_ms":  openTime.UnixMilli(),
			"open":          spec.Open,
			"high":          spec.High,
			"low":           spec.Low,
			"close":         spec.Close,
			"close_time_ms": openTime.UnixMilli(),
		})
	}
	mustWriteJSON(t, path, rows)
}

func readPaperRowsForTest(t *testing.T, path string) []papersignal.PaperJournalRow {
	t.Helper()
	rows, err := papersignal.ReadJournal(path)
	if err != nil {
		t.Fatal(err)
	}
	return rows
}

func mustWriteJSON(t *testing.T, path string, value any) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		t.Fatal(err)
	}
	data, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, data, 0644); err != nil {
		t.Fatal(err)
	}
}

func mustReadJSON(t *testing.T, path string, dest any) {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal(data, dest); err != nil {
		t.Fatal(err)
	}
}
