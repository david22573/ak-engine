package governance

import (
	"slices"
	"strings"
	"testing"
	"time"
)

func TestProvenanceResolutionIsCanonicalAndFailClosed(t *testing.T) {
	start := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	entry := ProvenanceEntry{ArtifactPath: "preserved:a.json", SHA256: "sha256:" + strings.Repeat("a", 64), GitBlobID: strings.Repeat("b", 40), EarliestKnownAppearance: start, CandidateID: "candidate", PossibleExposureRanges: []TimeRange{{start.AddDate(1, 0, 0), start.AddDate(2, 0, 0)}, {start, start.AddDate(1, 0, 0)}}, InformationCategories: []string{"b", "a"}, ByteIdenticalPaths: []string{"preserved:copy.json", "preserved:a.json"}, ProvenanceEdges: []string{"hash -> exposure", "path -> hash"}, Resolution: "UNTRUSTED_PROVENANCE_TREATED_AS_EXPOSED"}
	first, err := SealProvenance(ProvenanceResolution{Entries: []ProvenanceEntry{entry}})
	if err != nil {
		t.Fatal(err)
	}
	slices.Reverse(entry.PossibleExposureRanges)
	slices.Reverse(entry.InformationCategories)
	slices.Reverse(entry.ByteIdenticalPaths)
	slices.Reverse(entry.ProvenanceEdges)
	second, err := SealProvenance(ProvenanceResolution{Entries: []ProvenanceEntry{entry}})
	if err != nil {
		t.Fatal(err)
	}
	if first.ResolutionHash != second.ResolutionHash {
		t.Fatal("exposure ranges did not survive input reordering")
	}
	entry.SHA256 = "sha256:" + strings.Repeat("c", 64)
	mutated, err := SealProvenance(ProvenanceResolution{Entries: []ProvenanceEntry{entry}})
	if err != nil {
		t.Fatal(err)
	}
	if mutated.ResolutionHash == first.ResolutionHash {
		t.Fatal("artifact mutation did not change evidence identity")
	}
	trusted := entry
	trusted.Resolution = "TRUSTED"
	trusted.ValidationEligible = true
	if _, err := SealProvenance(ProvenanceResolution{Entries: []ProvenanceEntry{trusted}}); err == nil {
		t.Fatal("unknown provenance became validation eligible")
	}
}

func TestInspectionAuditRequiresNonzeroBoundedScope(t *testing.T) {
	start := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	audit := InspectionAudit{Tool: "exec_command", Command: "sed -n ...", Timestamp: start, Repository: "ak-engine", Commit: strings.Repeat("a", 40), FilesDisplayed: []string{"b.go", "a.go"}, LiteralCategories: []string{"legacy result literals"}, CandidateFamily: "candidate", AffectedPeriods: []TimeRange{{start, start.AddDate(1, 0, 0)}}, InspectionCount: 1, Classification: "LEGACY_ALREADY_EXPOSED_CONTENT_ONLY", FreshPreregistrationRequired: true}
	sealed, err := SealInspectionAudit(audit)
	if err != nil || sealed.AuditHash == "" {
		t.Fatal(err)
	}
	audit.InspectionCount = 0
	if _, err := SealInspectionAudit(audit); err == nil {
		t.Fatal("audit claimed zero inspection")
	}
	audit.InspectionCount = 1
	audit.FilesDisplayed = nil
	audit.Classification = "INSPECTION_SCOPE_UNRESOLVED"
	if _, err := SealInspectionAudit(audit); err == nil {
		t.Fatal("unknown inspection scope did not fail closed")
	}
}
