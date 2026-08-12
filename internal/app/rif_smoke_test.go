package app

import (
	"bytes"
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/david22573/ak-engine/internal/executionseries"
	"github.com/david22573/ak-engine/internal/researchidentity"
	"github.com/david22573/ak-engine/internal/rifbridge"
)

func TestResearchDiagnosticsSmokeProvesCompleteIncompleteAndRejectedBehavior(t *testing.T) {
	prior := researchDiagnosticsSmokeOutDir
	researchDiagnosticsSmokeOutDir = t.TempDir()
	defer func() { researchDiagnosticsSmokeOutDir = prior }()
	researchDiagnosticsSmokeCmd.SetContext(context.Background())
	if err := researchDiagnosticsSmokeCmd.RunE(researchDiagnosticsSmokeCmd, nil); err != nil {
		t.Fatal(err)
	}
	tests := []struct {
		name           string
		status         researchidentity.IdentityStatus
		eligible       bool
		hasIdentity    bool
		classification string
	}{
		{name: "complete_candidate", status: researchidentity.StatusComplete, eligible: true, hasIdentity: true, classification: rifbridge.ResearchStatusValidatedResearchLead},
		{name: "incomplete_candidate", status: researchidentity.StatusDatasetIncomplete, eligible: false, hasIdentity: false, classification: rifbridge.ResearchStatusValidatedResearchLead},
		{name: "rejected_candidate", status: researchidentity.StatusComplete, eligible: false, hasIdentity: true, classification: rifbridge.ResearchStatusRejected},
	}
	for _, tc := range tests {
		path := filepath.Join(researchDiagnosticsSmokeOutDir, tc.name+".research_diagnostics.json")
		raw, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		var diagnostic rifbridge.LocalResearchDiagnostics
		if err := json.Unmarshal(raw, &diagnostic); err != nil {
			t.Fatal(err)
		}
		if diagnostic.IdentityStatus != tc.status || diagnostic.EligibleForRIFReview != tc.eligible || (diagnostic.ResearchEvidence != nil) != tc.hasIdentity || diagnostic.CandidateResult.Classification != tc.classification || diagnostic.AuthorityStatus != rifbridge.AuthorityStatusNoneResearchOnly {
			t.Fatalf("unexpected %s diagnostic: %#v", tc.name, diagnostic)
		}
		if tc.name == "complete_candidate" && diagnostic.ArtifactHash != "sha256:e71b26d1f757b39416a7a93c680a00a5fa372b6a5ca83cb2b5d0c1e4dcfbcbc9" {
			t.Fatalf("complete canonical diagnostic hash = %s", diagnostic.ArtifactHash)
		}
		lower := strings.ToLower(string(raw))
		for _, prohibited := range []string{`"is_promoted"`, `"approved"`, `"frozen"`, `"paper_ready"`, `"paper_eligible"`, `"authorized"`} {
			if strings.Contains(lower, prohibited) {
				t.Fatalf("%s contains %q", tc.name, prohibited)
			}
		}
	}
	for _, suffix := range []string{".research.lock", ".research_audit.json", ".promotion_packet.json"} {
		matches, err := filepath.Glob(filepath.Join(researchDiagnosticsSmokeOutDir, "*"+suffix))
		if err != nil {
			t.Fatal(err)
		}
		if len(matches) != 0 {
			t.Fatalf("legacy artifacts emitted: %v", matches)
		}
	}
}

func TestResearchDiagnosticsCanonicalEvidenceIsOrderIndependentAndValueBound(t *testing.T) {
	fixture, err := researchidentity.BuildDiagnosticSmokeFixture(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	defer fixture.Cleanup()
	base := fixture.Request.Configuration
	context := researchidentity.ConfigurationContext{
		Symbol: base.Symbol, Market: base.Market, Interval: base.Interval,
		EvaluationStartMS: base.EvaluationStartMS, EvaluationEndMS: base.EvaluationEndMS,
		BuildTags: base.BuildTags,
	}
	firstConfig, err := researchidentity.ResolveConfiguration(context, []byte(`{"diagnostic_minimum_samples":30,"series_horizon_minutes":1,"gate_thresholds":{"minimum_events":300}}`))
	if err != nil {
		t.Fatal(err)
	}
	secondConfig, err := researchidentity.ResolveConfiguration(context, []byte(`{"gate_thresholds":{"minimum_events":300},"series_horizon_minutes":1,"diagnostic_minimum_samples":30}`))
	if err != nil {
		t.Fatal(err)
	}
	changedConfig, err := researchidentity.ResolveConfiguration(context, []byte(`{"gate_thresholds":{"minimum_events":301},"series_horizon_minutes":1,"diagnostic_minimum_samples":30}`))
	if err != nil {
		t.Fatal(err)
	}
	bridge := rifbridge.NewBridgeWithDeriver(fixture.Deriver)
	emit := func(name string, config researchidentity.ResolvedResearchConfiguration) ([]byte, rifbridge.LocalResearchDiagnostics) {
		request := fixture.Request
		request.Configuration = config
		result, err := bridge.EmitResearchDiagnostics(rifbridge.ResearchAssessment{
			Stem: filepath.Join(t.TempDir(), name), Classification: rifbridge.ResearchStatusValidatedResearchLead,
			ClassificationGates:       []rifbridge.ClassificationGate{{Name: "execution_series_identity", Passed: true, Critical: true}},
			ExecutionSeriesGeneration: executionseries.GenerationVersion, IdentityRequest: request,
		})
		if err != nil {
			t.Fatal(err)
		}
		raw, err := os.ReadFile(result.ArtifactPath)
		if err != nil {
			t.Fatal(err)
		}
		var diagnostic rifbridge.LocalResearchDiagnostics
		if err := json.Unmarshal(raw, &diagnostic); err != nil {
			t.Fatal(err)
		}
		return raw, diagnostic
	}
	firstBytes, first := emit("first", firstConfig)
	secondBytes, second := emit("second", secondConfig)
	if !bytes.Equal(firstBytes, secondBytes) || first.ArtifactHash != second.ArtifactHash || first.ResearchEvidence == nil || second.ResearchEvidence == nil || first.ResearchEvidence.ArtifactHash != second.ResearchEvidence.ArtifactHash {
		t.Fatal("reordered configuration JSON changed canonical evidence")
	}
	_, changed := emit("changed", changedConfig)
	if changed.ResearchEvidence == nil || changed.ArtifactHash == first.ArtifactHash || changed.ResearchEvidence.ArtifactHash == first.ResearchEvidence.ArtifactHash {
		t.Fatal("one identity-critical configuration value did not change evidence hashes")
	}
}
