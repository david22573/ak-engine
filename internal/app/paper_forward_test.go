package app

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/david22573/ak-engine/internal/canonicalcontract"
	"github.com/david22573/ak-engine/internal/executionseries"
	"github.com/david22573/ak-engine/internal/features"
	"github.com/david22573/ak-engine/internal/papersignal"
	"github.com/david22573/ak-engine/internal/regime"
	"github.com/david22573/ak-engine/internal/researchidentity"
	"github.com/david22573/ak-engine/internal/rifbridge"
	"github.com/david22573/ak-engine/pkg/protocol"
)

func TestPaperForwardObservationFlow(t *testing.T) {
	base := time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC)
	tmp := t.TempDir()
	passManifest := writePaperTestManifest(t, tmp, "pass_manifest.json", "COVERS_WINDOW")
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

	t.Run("legacy_manifest_hard_fails_before_output", func(t *testing.T) {
		outDir := filepath.Join(tmp, "forward_allowed")
		journal := filepath.Join(tmp, "allowed.jsonl")
		err := runPaperForwardForTest(passManifest, snapshotDir, outDir, journal, "NegativeFundingLong", "BTCUSDT", 1, base.Add(-2*time.Hour))
		if err == nil || !strings.Contains(err.Error(), "legacy --dataset-manifest") {
			t.Fatalf("legacy manifest did not hard fail: %v", err)
		}
		if _, err := os.Stat(journal); !os.IsNotExist(err) {
			t.Fatalf("legacy failure wrote journal: %v", err)
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
		writePaperCandles(t, filepath.Join(marketDir, "BTCUSDT_1m.json"), "BTCUSDT", base.Add(-2*time.Hour), []paperTestCandleSpec{
			{Offset: 0, Open: 100, High: 100.2, Low: 99.8, Close: 100},
			{Offset: 30 * time.Minute, Open: 100, High: 101.5, Low: 99.8, Close: 101.2},
			{Offset: 59 * time.Minute, Open: 101.2, High: 101.3, Low: 100.9, Close: 101.0},
		})
		writePaperCandles(t, filepath.Join(marketDir, "ETHUSDT_1m.json"), "ETHUSDT", base.Add(-2*time.Hour), []paperTestCandleSpec{
			{Offset: 0, Open: 100, High: 100.2, Low: 99.8, Close: 100},
			{Offset: 30 * time.Minute, Open: 100, High: 100.2, Low: 99.0, Close: 99.2},
			{Offset: 59 * time.Minute, Open: 99.2, High: 99.4, Low: 99.1, Close: 99.3},
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

	t.Run("canonical_paper_surface_has_no_execution_import", func(t *testing.T) {
		root, err := researchidentity.FindRepositoryRoot("")
		if err != nil {
			t.Fatal(err)
		}
		files := defaultPaperForwardSafetyFiles()
		for i := range files {
			files[i] = filepath.Join(root, files[i])
		}
		report, err := runPaperForwardSafetyCheck(files)
		if err != nil {
			t.Fatal(err)
		}
		if report.Status != "PASS" || len(report.Findings) != 0 {
			t.Fatalf("canonical paper surface crossed execution boundary: %#v", report)
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

func TestPaperRIFCompatibilityFlagsCannotAuthorizeLegacyManifest(t *testing.T) {
	path := writePaperTestManifest(t, t.TempDir(), "legacy.json", "PIT_ELIGIBLE")
	for _, allowWarnings := range []bool{false, true} {
		evidence := evaluatePaperRIF(path, allowWarnings)
		if !evidence.BlocksSignal || evidence.Status != "RIF_BLOCKED" || !strings.Contains(evidence.BlockReason, "readiness path retired") {
			t.Fatalf("allow_warnings=%t authorized legacy evidence: %+v", allowWarnings, evidence)
		}
	}
}

func TestCanonicalPaperDecisionExecutesExactCompiledCandidate(t *testing.T) {
	tmp := t.TempDir()
	evidencePath, evidence := writeCurrentPaperEvidenceFixture(t, tmp)
	loaded, err := loadPaperCanonicalEvidence(evidencePath, evidence.Candidate.CandidateID, evidence.Configuration.Symbol, evidence.Configuration.Market, evidence.Configuration.Interval)
	if err != nil {
		t.Fatal(err)
	}
	open := time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC)
	triggerPath := writePaperDecisionFixture(t, tmp, "trigger.json", loaded, open, true)
	generatedAt := open.Add(time.Minute)
	decision, err := loadPaperDecision(triggerPath, loaded, generatedAt)
	if err != nil {
		t.Fatal(err)
	}
	if !decision.Triggered || decision.EntryPrice != 101 || decision.InputHash == "" {
		t.Fatalf("exact candidate did not trigger reproducibly: %#v", decision)
	}
	repeated, err := loadPaperDecision(triggerPath, loaded, generatedAt)
	if err != nil || repeated != decision {
		t.Fatalf("canonical decision was not reproducible: first=%#v second=%#v err=%v", decision, repeated, err)
	}
	journal := filepath.Join(tmp, "canonical.jsonl")
	pfCandidate = loaded.Candidate.CandidateID
	pfSymbols = loaded.Configuration.Symbol
	pfTimeframe = loaded.Configuration.Interval
	pfMarketType = loaded.Configuration.Market
	pfDatasetManifest = ""
	pfResearchEvidence = evidencePath
	pfDecisionInput = triggerPath
	pfResearchLock = ""
	pfSnapshotDir = ""
	pfOutDir = filepath.Join(tmp, "canonical_out")
	pfJournal = journal
	pfMode = papersignal.ModePaperReplay
	pfGeneratedAtUTC = generatedAt.Format(time.RFC3339Nano)
	pfDryRun = false
	pfPaperOnly = true
	if err := paperForwardCmd.RunE(paperForwardCmd, nil); err != nil {
		t.Fatal(err)
	}
	rows := readPaperRowsForTest(t, journal)
	if len(rows) != 1 || rows[0].SignalStatus != papersignal.StatusAllowed || rows[0].OutcomeStatus != papersignal.OutcomePending || rows[0].CandidateHash != loaded.Candidate.ArtifactHash || rows[0].ConfigurationHash != loaded.Configuration.ArtifactHash || rows[0].ResearchEvidenceHash != loaded.EvidenceHash || rows[0].DecisionInputHash != decision.InputHash {
		t.Fatalf("canonical command did not preserve exact identity: %#v", rows)
	}
	legacyJournal := filepath.Join(tmp, "legacy_mixed.jsonl")
	if err := papersignal.AppendToJournal(legacyJournal, papersignal.PaperJournalRow{SignalID: "old-synthetic", CandidateID: loaded.Candidate.CandidateID}); err != nil {
		t.Fatal(err)
	}
	pfJournal = legacyJournal
	pfOutDir = filepath.Join(tmp, "must_not_write")
	if err := paperForwardCmd.RunE(paperForwardCmd, nil); err == nil || !strings.Contains(err.Error(), "historical/noncanonical") {
		t.Fatalf("canonical sample mixed into old synthetic journal: %v", err)
	}

	waitPath := writePaperDecisionFixture(t, tmp, "wait.json", loaded, open, false)
	wait, err := loadPaperDecision(waitPath, loaded, generatedAt)
	if err != nil {
		t.Fatal(err)
	}
	if wait.Triggered {
		t.Fatal("no-trigger candidate emitted an actionable decision")
	}
	meta, err := paperMetadataFromEvidence(loaded)
	if err != nil {
		t.Fatal(err)
	}
	sig := buildCanonicalPaperSignal(meta, loaded, wait, loaded.Configuration.Symbol, loaded.Configuration.Market, loaded.Configuration.Interval, generatedAt.Format(time.RFC3339Nano))
	if sig.SignalStatus != papersignal.StatusWait || papersignal.IsActionableStatus(sig.SignalStatus) || sig.OutcomeStatus != "" {
		t.Fatalf("no-trigger observation became order-like: %#v", sig)
	}
	pfDecisionInput = waitPath
	pfOutDir = filepath.Join(tmp, "wait_out")
	pfJournal = filepath.Join(tmp, "wait.jsonl")
	if err := paperForwardCmd.RunE(paperForwardCmd, nil); err != nil {
		t.Fatal(err)
	}
	waitRows := readPaperRowsForTest(t, pfJournal)
	if len(waitRows) != 1 || waitRows[0].SignalStatus != papersignal.StatusWait || waitRows[0].OutcomeStatus != "" || waitRows[0].EntryReferencePrice != 0 {
		t.Fatalf("no-trigger command emitted an order-like row: %#v", waitRows)
	}
	var waitRun papersignal.ForwardObservationRun
	mustReadJSON(t, filepath.Join(pfOutDir, "forward_paper_observation_run.json"), &waitRun)
	if waitRun.WaitObservations != 1 || waitRun.AllowedSignals != 0 || waitRun.PendingOutcomes != 0 {
		t.Fatalf("no-trigger run counts are inconsistent: %#v", waitRun)
	}

	if _, err := loadPaperCanonicalEvidence("", evidence.Candidate.CandidateID, evidence.Configuration.Symbol, evidence.Configuration.Market, evidence.Configuration.Interval); err == nil {
		t.Fatal("missing canonical evidence passed")
	}
	raw, err := os.ReadFile(evidencePath)
	if err != nil {
		t.Fatal(err)
	}
	tampered := strings.Replace(string(raw), evidence.Candidate.CandidateID, evidence.Candidate.CandidateID+"-tampered", 1)
	tamperedPath := filepath.Join(tmp, "tampered_evidence.json")
	if err := os.WriteFile(tamperedPath, []byte(tampered), 0644); err != nil {
		t.Fatal(err)
	}
	if _, err := loadPaperCanonicalEvidence(tamperedPath, evidence.Candidate.CandidateID, evidence.Configuration.Symbol, evidence.Configuration.Market, evidence.Configuration.Interval); err == nil {
		t.Fatal("tampered canonical evidence passed")
	}
	if _, err := loadPaperCanonicalEvidence(evidencePath, evidence.Candidate.CandidateID, "ETHUSDT", evidence.Configuration.Market, evidence.Configuration.Interval); err == nil {
		t.Fatal("wrong-symbol evidence passed")
	}
	if _, err := loadPaperDecision(triggerPath, loaded, generatedAt.Add(time.Millisecond)); err == nil {
		t.Fatal("stale/future generated timestamp passed")
	}
	triggerSource := filepath.Join(tmp, "trigger_source.json")
	if err := os.WriteFile(triggerSource, []byte("[]\n"), 0644); err != nil {
		t.Fatal(err)
	}
	if _, err := loadPaperDecision(triggerPath, loaded, generatedAt); err == nil || !strings.Contains(err.Error(), "source candle hash mismatch") {
		t.Fatalf("mutated as-of source passed: %v", err)
	}
}

func TestCanonicalPaperGradingAndIdentityIsolation(t *testing.T) {
	fill := time.Date(2026, 1, 2, 0, 0, 0, 0, time.UTC)
	row := paperTestRow("boundary", "BTCUSDT", papersignal.SideLong, fill, 2)
	row.FillTimeUTC = fill.Format(time.RFC3339Nano)
	candles := []protocol.Candle{
		{Market: "SPOT", Symbol: "BTCUSDT", Interval: "1m", OpenTimeMS: fill.UnixMilli(), CloseTimeMS: fill.Add(time.Minute).UnixMilli() - 1, Open: 100, High: 101.1, Low: 99.8, Close: 100.1},
		{Market: "SPOT", Symbol: "BTCUSDT", Interval: "1m", OpenTimeMS: fill.Add(time.Minute).UnixMilli(), CloseTimeMS: fill.Add(2*time.Minute).UnixMilli() - 1, Open: 100.1, High: 100.2, Low: 100, Close: 100.1},
	}
	graded, err := gradePaperOutcome(row, candles, fill.Add(2*time.Minute))
	if err != nil {
		t.Fatal(err)
	}
	if graded.OutcomeStatus != papersignal.OutcomeLongTPFirst || graded.OutcomeReturnBPS == nil || *graded.OutcomeReturnBPS != 100 {
		t.Fatalf("target status/return do not share boundary fill: %#v", graded)
	}
	candles[0].Low = 99
	ambiguous, err := gradePaperOutcome(row, candles, fill.Add(2*time.Minute))
	if err != nil {
		t.Fatal(err)
	}
	if ambiguous.OutcomeStatus != papersignal.OutcomeAmbiguousIntrabar || ambiguous.OutcomeReturnBPS != nil {
		t.Fatalf("same-bar ambiguity was assigned a fabricated return: %#v", ambiguous)
	}

	rows := make([]papersignal.PaperJournalRow, 100)
	for i := range rows {
		rows[i] = paperGradedRow("isolated", i, papersignal.OutcomeLongTPFirst, 20)
	}
	rows[99].ConfigurationHash = "other-config"
	report := buildShadowReadinessReport(rows, "NegativeFundingLong", fill)
	if report.IdentityConflicts != 1 || report.ReadinessLabel != papersignal.ReadinessBlockedByResults {
		t.Fatalf("mixed configuration identity contaminated readiness: %#v", report)
	}
	for i := range rows {
		rows[i].ConfigurationHash = "config-hash"
		rows[i].OutcomeReturnBPS = nil
	}
	report = buildShadowReadinessReport(rows, "NegativeFundingLong", fill)
	if report.ReadinessLabel == papersignal.ReadinessShadowCandidate {
		t.Fatalf("status-only rows qualified without returns: %#v", report)
	}

	tmp := t.TempDir()
	badPath := filepath.Join(tmp, "BTCUSDT_1m.json")
	mustWriteJSON(t, badPath, []protocol.Candle{
		{Market: "SPOT", Symbol: "BTCUSDT", Interval: "1m", OpenTimeMS: fill.UnixMilli(), CloseTimeMS: fill.Add(time.Minute).UnixMilli() - 1, Open: 100, High: 100, Low: 100, Close: 100},
		{Market: "SPOT", Symbol: "BTCUSDT", Interval: "1m", OpenTimeMS: fill.Add(2 * time.Minute).UnixMilli(), CloseTimeMS: fill.Add(3*time.Minute).UnixMilli() - 1, Open: 100, High: 100, Low: 100, Close: 100},
	})
	if _, _, err := loadPaperMarketCandles(tmp, "", row); err == nil || !strings.Contains(err.Error(), "cadence mismatch") {
		t.Fatalf("gapped grading data did not fail validation: %v", err)
	}
	if err := paperSignalGradeCmd.RunE(paperSignalGradeCmd, nil); err == nil || !strings.Contains(err.Error(), "retired") {
		t.Fatalf("parallel mock grader remained authoritative: %v", err)
	}
}

func writeCurrentPaperEvidenceFixture(t *testing.T, parent string) (string, paperCanonicalEvidence) {
	t.Helper()
	fixture, err := researchidentity.BuildDiagnosticSmokeFixture(parent)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(fixture.Cleanup)
	bridge := rifbridge.NewBridgeWithDeriver(fixture.Deriver)
	result, err := bridge.EmitResearchDiagnostics(rifbridge.ResearchAssessment{
		Stem: filepath.Join(parent, "paper_evidence"), Classification: rifbridge.ResearchStatusValidatedResearchLead,
		ClassificationGates:       []rifbridge.ClassificationGate{{Name: "execution_series_identity", Passed: true, Critical: true}},
		ExecutionSeriesGeneration: executionseries.GenerationVersion, IdentityRequest: fixture.Request,
	})
	if err != nil {
		t.Fatal(err)
	}
	var diagnostic rifbridge.LocalResearchDiagnostics
	mustReadJSON(t, result.ArtifactPath, &diagnostic)
	root, err := researchidentity.FindRepositoryRoot("")
	if err != nil {
		t.Fatal(err)
	}
	registry, err := researchidentity.DefaultRegistry()
	if err != nil {
		t.Fatal(err)
	}
	candidate, err := registry.Resolve(root, "CompressionBreakout", "LONG")
	if err != nil {
		t.Fatal(err)
	}
	evidence := *diagnostic.ResearchEvidence
	identity := evidence.ResearchIdentity
	identity.Candidate = candidate
	identity.ArtifactHash = ""
	identity.ArtifactHash = rehashPaperArtifact(t, identity.Contract, identity)
	evidence.ResearchIdentity = identity
	evidence.EvidenceID = identity.ArtifactHash
	evidence.ArtifactHash = ""
	evidence.ArtifactHash = rehashPaperArtifact(t, evidence.Contract, evidence)
	diagnostic.CandidateResult.CandidateID = candidate.CandidateID
	diagnostic.CandidateResult.CandidateVersion = candidate.CandidateVersion
	diagnostic.ResearchEvidence = &evidence
	diagnostic.ArtifactHash = ""
	diagnostic.ArtifactHash = rehashPaperArtifact(t, diagnostic.Contract, diagnostic)
	raw, err := canonicalcontract.CanonicalizeValue(diagnostic)
	if err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(parent, "canonical_paper_evidence.json")
	if err := os.WriteFile(path, raw, 0644); err != nil {
		t.Fatal(err)
	}
	return path, paperCanonicalEvidence{Candidate: candidate, Configuration: identity.Configuration}
}

func rehashPaperArtifact(t *testing.T, header canonicalcontract.ContractHeader, value any) string {
	t.Helper()
	hash, _, err := canonicalcontract.HashArtifactValue(header.SchemaName, header.SchemaVersion, header.ArtifactRole, "artifact_hash", value)
	if err != nil {
		t.Fatal(err)
	}
	return hash
}

func writePaperDecisionFixture(t *testing.T, dir, name string, evidence paperCanonicalEvidence, open time.Time, trigger bool) string {
	t.Helper()
	closePrice := 101.0
	ema := 100.0
	beta := "btc_up"
	if !trigger {
		closePrice = 99
		beta = "btc_down"
	}
	available := open.Add(time.Minute).UnixMilli() - 1
	sourceName := strings.TrimSuffix(name, filepath.Ext(name)) + "_source.json"
	sourcePath := filepath.Join(dir, sourceName)
	mustWriteJSON(t, sourcePath, []protocol.Candle{{
		Market: evidence.Configuration.Market, Symbol: evidence.Configuration.Symbol, Interval: evidence.Configuration.Interval,
		OpenTimeMS: open.UnixMilli(), CloseTimeMS: available, Open: closePrice, High: closePrice, Low: closePrice, Close: closePrice,
	}})
	sourceHash, err := papersignal.HashFile(sourcePath)
	if err != nil {
		t.Fatal(err)
	}
	input := paperDecisionInput{
		Contract: paperDecisionInputVersion, Symbol: evidence.Configuration.Symbol, Market: evidence.Configuration.Market, Timeframe: evidence.Configuration.Interval,
		SourceCandlesPath: sourceName, SourceCandlesHash: sourceHash,
		FeatureImplementationHash: evidence.FeatureImplementationHash, RegimeImplementationHash: evidence.RegimeImplementationHash,
		Feature:             features.Row{Market: evidence.Configuration.Market, Symbol: evidence.Configuration.Symbol, Interval: evidence.Configuration.Interval, EventTimeMS: open.UnixMilli(), AvailableAtMS: available, Close: closePrice, EMA20: ema, EMA50: ema, EMA200: ema},
		Regime:              regime.Label{Market: evidence.Configuration.Market, Symbol: evidence.Configuration.Symbol, Interval: evidence.Configuration.Interval, EventTimeMS: open.UnixMilli(), AvailableAtMS: available, Volatility: "compressed", Trend: "range", Liquidity: "normal", MarketBeta: beta},
		TradableObservation: paperTradableObservation{Symbol: evidence.Configuration.Symbol, Market: evidence.Configuration.Market, TimestampUTC: open.Add(time.Minute).Format(time.RFC3339Nano), Price: 101},
	}
	path := filepath.Join(dir, name)
	mustWriteJSON(t, path, input)
	return path
}

func runPaperForwardForTest(manifest, snapshotDir, outDir, journal, candidate, symbols string, maxSignals int, generatedAt time.Time) error {
	pfCandidate = candidate
	pfSymbols = symbols
	pfTimeframe = "1m"
	pfMarketType = "SPOT"
	pfDatasetManifest = manifest
	pfResearchEvidence = ""
	pfDecisionInput = ""
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
	return paperForwardCmd.RunE(paperForwardCmd, []string{})
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
		CandidateVersion:     "1.0.0",
		CandidateHash:        "candidate-hash",
		ConfigurationHash:    "config-hash",
		ResearchEvidenceHash: "evidence-hash",
		DecisionInputHash:    "decision-input-hash-" + id,
		GeneratedAtUTC:       generatedAt.UTC().Format(time.RFC3339),
		DecisionTimeUTC:      generatedAt.Add(-time.Millisecond).UTC().Format(time.RFC3339Nano),
		FillTimeUTC:          generatedAt.UTC().Format(time.RFC3339Nano),
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
		DatasetHash:          "dataset-hash",
		PitCoverageHash:      "pit-hash",
		RIFStatus:            "RIF_PASS",
	}
}

func paperGradedRow(prefix string, idx int, outcome papersignal.OutcomeStatus, ret float64) papersignal.PaperJournalRow {
	fill := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC).Add(time.Duration(idx) * time.Minute)
	return papersignal.PaperJournalRow{
		SignalID:             prefix + "-" + time.Unix(int64(idx), 0).UTC().Format("150405"),
		CandidateID:          "NegativeFundingLong",
		CandidateVersion:     "1.0.0",
		CandidateHash:        "candidate-hash",
		ConfigurationHash:    "config-hash",
		ResearchEvidenceHash: "evidence-hash",
		DecisionInputHash:    "decision-input-" + time.Unix(int64(idx), 0).UTC().Format("150405"),
		GeneratedAtUTC:       fill.Format(time.RFC3339Nano),
		DecisionTimeUTC:      fill.Add(-time.Millisecond).Format(time.RFC3339Nano),
		FillTimeUTC:          fill.Format(time.RFC3339Nano),
		Symbol:               "BTCUSDT",
		MarketType:           "SPOT",
		Timeframe:            "1m",
		SignalStatus:         papersignal.StatusAllowed,
		OutcomeStatus:        outcome,
		OutcomeReturnBPS:     &ret,
		DatasetHash:          "dataset-hash",
		PitCoverageHash:      "pit-hash",
	}
}

type paperTestCandleSpec struct {
	Offset time.Duration
	Open   float64
	High   float64
	Low    float64
	Close  float64
}

func writePaperCandles(t *testing.T, path, symbol string, generatedAt time.Time, specs []paperTestCandleSpec) {
	t.Helper()
	byOffset := make(map[time.Duration]paperTestCandleSpec, len(specs))
	for _, spec := range specs {
		byOffset[spec.Offset] = spec
	}
	var rows []map[string]any
	last := specs[len(specs)-1]
	previous := specs[0]
	for offset := time.Duration(0); offset <= last.Offset; offset += time.Minute {
		spec, ok := byOffset[offset]
		if !ok {
			spec = paperTestCandleSpec{Offset: offset, Open: previous.Close, High: previous.Close, Low: previous.Close, Close: previous.Close}
		}
		previous = spec
		openTime := generatedAt.Add(offset)
		rows = append(rows, map[string]any{
			"market":        "SPOT",
			"symbol":        symbol,
			"interval":      "1m",
			"open_time_ms":  openTime.UnixMilli(),
			"open":          spec.Open,
			"high":          spec.High,
			"low":           spec.Low,
			"close":         spec.Close,
			"close_time_ms": openTime.Add(time.Minute).UnixMilli() - 1,
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
