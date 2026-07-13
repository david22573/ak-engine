package app

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/david22573/ak-engine/internal/qualification"
)

func TestPR4B0InventoryIsCompleteAndPreservesPriorLabels(t *testing.T) {
	inventory, err := buildPR4B0Inventory()
	if err != nil {
		t.Fatal(err)
	}
	registered := pr4b0RegisteredCandidateIDs()
	inventory.RegisteredCandidateIDs = registered
	inventory.UnknownImplementations = qualification.FindUnknownImplementations(inventory.Candidates, registered)
	inventory.CandidateCount = len(inventory.Candidates)
	if err := qualification.ValidateInventory(inventory, registered); err != nil {
		t.Fatalf("ValidateInventory() error = %v", err)
	}
	if inventory.CandidateCount != 99 {
		t.Fatalf("candidate count = %d, want 99", inventory.CandidateCount)
	}
	if len(inventory.RegisteredCandidateIDs) != 17 || len(inventory.UnknownImplementations) != 0 || len(inventory.OmittedCandidates) != 0 {
		t.Fatalf("inventory coverage mismatch: registered=%d unknown=%v omitted=%v", len(inventory.RegisteredCandidateIDs), inventory.UnknownImplementations, inventory.OmittedCandidates)
	}
	byID := make(map[string]qualification.CandidateRecord, len(inventory.Candidates))
	familyCounts := make(map[string]int)
	for _, candidate := range inventory.Candidates {
		byID[candidate.CandidateID] = candidate
		familyCounts[candidate.StrategyFamily]++
		if candidate.FinalStatus == qualification.StatusQualified {
			t.Fatalf("unexpected qualified candidate %q", candidate.CandidateID)
		}
	}
	for family, want := range map[string]int{
		"NegativeFundingLong":                      6,
		"VolumeImbalanceFundingReversionProxyLong": 6,
		"CompressionVolumeBreakout":                6,
		"RegimeTrendPullbackContinuation":          10,
	} {
		if got := familyCounts[family]; got != want {
			t.Fatalf("family %q candidate rows = %d, want %d", family, got, want)
		}
	}
	checks := map[string]qualification.FinalStatus{
		"fast_accumulation":                                 qualification.StatusRejected,
		"price-alpha/CompressionBreakout/long":              qualification.StatusNearMiss,
		"funding-alpha/NegativeFundingLong|long|240m":       qualification.StatusRejected,
		"phase11/CompressionVolumeBreakout|long|240m":       qualification.StatusRejected,
		"phase11/RegimeTrendPullbackContinuation|long|240m": qualification.StatusRejected,
		"phase12/DowntrendMidVolReliefLong240m":             qualification.StatusNearMiss,
		"phase13/ContextFreeMomentumBreakoutProbe":          qualification.StatusInsufficientSample,
	}
	for id, want := range checks {
		candidate, ok := byID[id]
		if !ok {
			t.Fatalf("candidate %q missing", id)
		}
		if candidate.FinalStatus != want {
			t.Fatalf("candidate %q status = %s, want %s", id, candidate.FinalStatus, want)
		}
	}
	if byID["phase11/RegimeTrendPullbackContinuation|long|240m"].ImplementationReproducible {
		t.Fatal("untracked Phase 11.2 implementation reported reproducible")
	}
}

func TestPR4B0GeneratorWritesOnlyMandatoryNoCandidateArtifacts(t *testing.T) {
	outDir := t.TempDir()
	if err := runPR4B0CandidateQualification(outDir, "source-commit", false, ""); err != nil {
		t.Fatalf("runPR4B0CandidateQualification() error = %v", err)
	}
	wantFiles := []string{
		"pr4b0_candidate_inventory.json", "pr4b0_candidate_inventory.md",
		"pr4b0_candidate_qualification.json", "pr4b0_candidate_qualification.md",
	}
	entries, err := os.ReadDir(outDir)
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != len(wantFiles) {
		t.Fatalf("generated %d files, want %d", len(entries), len(wantFiles))
	}
	for _, name := range wantFiles {
		if _, err := os.Stat(filepath.Join(outDir, name)); err != nil {
			t.Fatalf("missing %s: %v", name, err)
		}
	}
	for _, forbidden := range []string{"pr4b0_candidate_qualification_protocol.json", "pr4b0_frozen_candidate.json", "pr4b0_candidate_registration_request.json"} {
		if _, err := os.Stat(filepath.Join(outDir, forbidden)); !os.IsNotExist(err) {
			t.Fatalf("forbidden artifact %s was generated", forbidden)
		}
	}

	var inventory qualification.CandidateInventory
	readJSON(t, filepath.Join(outDir, "pr4b0_candidate_inventory.json"), &inventory)
	if inventory.CandidateCount != 99 {
		t.Fatalf("generated inventory count = %d", inventory.CandidateCount)
	}
	var report pr4b0QualificationReport
	readJSON(t, filepath.Join(outDir, "pr4b0_candidate_qualification.json"), &report)
	if report.FinalLabel != pr4b0NoCandidateLabel || report.SelectedCandidate != nil || report.FrozenIdentity != nil || report.CandidateRegistrationArtifact != nil {
		t.Fatalf("unexpected qualification output: label=%s selected=%v frozen=%v registration=%v", report.FinalLabel, report.SelectedCandidate, report.FrozenIdentity, report.CandidateRegistrationArtifact)
	}
	wantStatusCounts := map[string]int{"REJECTED": 71, "NEAR_MISS": 3, "PIT_EVIDENCE_MISSING": 24, "INSUFFICIENT_SAMPLE": 1}
	encodedCounts, err := json.Marshal(report.CandidateInventorySummary["status_counts"])
	if err != nil {
		t.Fatal(err)
	}
	var gotStatusCounts map[string]int
	if err := json.Unmarshal(encodedCounts, &gotStatusCounts); err != nil {
		t.Fatal(err)
	}
	for status, want := range wantStatusCounts {
		if got := gotStatusCounts[status]; got != want {
			t.Fatalf("status %s count = %d, want %d", status, got, want)
		}
	}
	wantHash, err := hashReport(report)
	if err != nil {
		t.Fatal(err)
	}
	if report.QualificationReportHash != wantHash {
		t.Fatalf("report hash = %s, want %s", report.QualificationReportHash, wantHash)
	}
	markdown, err := os.ReadFile(filepath.Join(outDir, "pr4b0_candidate_qualification.md"))
	if err != nil {
		t.Fatal(err)
	}
	for _, statement := range []string{
		"No paper evaluator was implemented", "RIF did not authorize a candidate", "No candidate was promoted", "Trader behavior was unchanged", "Do not begin PR4B1",
	} {
		if !strings.Contains(string(markdown), statement) {
			t.Fatalf("qualification markdown missing %q", statement)
		}
	}
}

func TestPR4B0CompletedVerificationIsExplicit(t *testing.T) {
	inventory, err := buildPR4B0Inventory()
	if err != nil {
		t.Fatal(err)
	}
	inventory.RegisteredCandidateIDs = pr4b0RegisteredCandidateIDs()
	inventory.CandidateCount = len(inventory.Candidates)
	report := buildPR4B0QualificationReport(inventory, "source", true, "fresh")
	for _, result := range report.TestsAndRace {
		if result.Status != "PASS" || result.ExitCode == nil || *result.ExitCode != 0 {
			t.Fatalf("verification result not explicit: %#v", result)
		}
	}
	if report.FreshClone["status"] != "PASS" || report.FreshClone["commit"] != "fresh" {
		t.Fatalf("fresh clone result = %#v", report.FreshClone)
	}
}

func readJSON(t *testing.T, path string, target any) {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	decoder := json.NewDecoder(strings.NewReader(string(data)))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(target); err != nil {
		t.Fatalf("decode %s: %v", path, err)
	}
}
