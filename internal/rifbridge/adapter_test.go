package rifbridge_test

import (
	"encoding/json"
	"errors"
	"math"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/david22573/ak-engine/internal/executionseries"
	"github.com/david22573/ak-engine/internal/researchidentity"
	"github.com/david22573/ak-engine/internal/rifbridge"
)

type fixtureDeriver struct {
	assessment researchidentity.Assessment
	err        error
}

func (d fixtureDeriver) Derive(researchidentity.DerivationRequest) (researchidentity.Assessment, error) {
	return d.assessment, d.err
}

func TestCompleteIdentityMakesOnlyEligibleResearchLeadReviewable(t *testing.T) {
	for _, classification := range []string{
		rifbridge.ResearchStatusValidatedResearchLead,
		rifbridge.ResearchStatusRejected,
		rifbridge.ResearchStatusFragile,
	} {
		t.Run(classification, func(t *testing.T) {
			input := completeBridgeInput(t, classification)
			result, err := rifbridge.NewBridgeWithDeriver(completeFixtureDeriver(t)).EmitResearchDiagnostics(input)
			if err != nil {
				t.Fatalf("EmitResearchDiagnostics: %v", err)
			}
			wantEligible := classification == rifbridge.ResearchStatusValidatedResearchLead
			if result.ArtifactDisposition != rifbridge.ArtifactEmitted || result.EligibleForReview != wantEligible {
				t.Fatalf("unexpected result: %#v", result)
			}
			diagnostic := readDiagnostic(t, result.ArtifactPath)
			if diagnostic.IdentityStatus != researchidentity.StatusComplete || diagnostic.EligibleForRIFReview != wantEligible || diagnostic.ResearchEvidence == nil {
				t.Fatalf("unexpected diagnostic: %#v", diagnostic)
			}
			if diagnostic.CandidateResult.Classification != classification {
				t.Fatalf("classification changed from %q to %q", classification, diagnostic.CandidateResult.Classification)
			}
			if diagnostic.AuthorityStatus != rifbridge.AuthorityStatusNoneResearchOnly {
				t.Fatalf("authority = %q", diagnostic.AuthorityStatus)
			}
		})
	}
}

func TestCallerClassificationMustMatchGateDerivation(t *testing.T) {
	input := completeBridgeInput(t, rifbridge.ResearchStatusValidatedResearchLead)
	input.Classification = rifbridge.ResearchStatusRejected
	result, err := rifbridge.NewBridgeWithDeriver(completeFixtureDeriver(t)).EmitResearchDiagnostics(input)
	if !errors.Is(err, rifbridge.ErrInvalidResearchInput) {
		t.Fatalf("classification mismatch error = %v, want invalid research input", err)
	}
	if result.ArtifactDisposition != rifbridge.ArtifactSuppressed || result.EligibleForReview {
		t.Fatalf("classification mismatch emitted evidence: %#v", result)
	}
}

func TestClassificationFromDifferentExecutionSeriesIsRejected(t *testing.T) {
	input := completeBridgeInput(t, rifbridge.ResearchStatusValidatedResearchLead)
	input.ExecutionSeriesGeneration = "deep-return-series.v1"
	result, err := rifbridge.NewBridgeWithDeriver(completeFixtureDeriver(t)).EmitResearchDiagnostics(input)
	if !errors.Is(err, rifbridge.ErrInvalidResearchInput) {
		t.Fatalf("series mismatch error = %v, want invalid research input", err)
	}
	if result.ArtifactDisposition != rifbridge.ArtifactSuppressed || result.EligibleForReview {
		t.Fatalf("series mismatch emitted reviewable evidence: %#v", result)
	}
}

func TestIncompleteDirtyAndConflictRemainVisibleAndNonReviewable(t *testing.T) {
	for _, status := range []researchidentity.IdentityStatus{
		researchidentity.StatusCandidateIncomplete,
		researchidentity.StatusDirtyEngineSource,
		researchidentity.StatusConflict,
	} {
		t.Run(string(status), func(t *testing.T) {
			input := completeBridgeInput(t, rifbridge.ResearchStatusValidatedResearchLead)
			derivationErr := &researchidentity.DerivationError{Status: status, Code: "FIXTURE_BLOCK", Err: errors.New("fixture identity is blocked")}
			deriver := fixtureDeriver{
				assessment: researchidentity.Assessment{Status: status, Findings: []researchidentity.Finding{{Code: "FIXTURE_BLOCK", Domain: "fixture", Reason: "fixture identity is blocked", Status: status, Blocking: true}}},
				err:        derivationErr,
			}
			result, err := rifbridge.NewBridgeWithDeriver(deriver).EmitResearchDiagnostics(input)
			if !errors.Is(err, derivationErr) {
				t.Fatalf("expected derivation error, got %v", err)
			}
			if result.ArtifactDisposition != rifbridge.ArtifactEmitted || result.EligibleForReview || result.IdentityStatus != status {
				t.Fatalf("unexpected result: %#v", result)
			}
			diagnostic := readDiagnostic(t, result.ArtifactPath)
			if diagnostic.ResearchEvidence != nil || diagnostic.EligibleForRIFReview || diagnostic.IdentityStatus != status || len(diagnostic.IdentityFindings) == 0 {
				t.Fatalf("unsafe incomplete diagnostic: %#v", diagnostic)
			}
		})
	}
}

func TestLocalIntegrityFailuresBlockEligibilityWithoutChangingCompleteIdentity(t *testing.T) {
	input := completeBridgeInput(t, rifbridge.ResearchStatusValidatedResearchLead)
	input.IdentityRequest.Returns = input.IdentityRequest.Returns[:2]
	input.IdentityRequest.Timestamps = input.IdentityRequest.Timestamps[:2]
	input.IdentityRequest.EvaluationEventTimestamps = input.IdentityRequest.EvaluationEventTimestamps[:2]
	result, err := rifbridge.NewBridgeWithDeriver(completeFixtureDeriver(t)).EmitResearchDiagnostics(input)
	if err != nil {
		t.Fatalf("local integrity finding should not become identity error: %v", err)
	}
	if result.IdentityStatus != researchidentity.StatusComplete || result.LocalIntegrity != rifbridge.LocalIntegrityFailed || result.EligibleForReview {
		t.Fatalf("unexpected result: %#v", result)
	}
}

func TestBoundModelParameterCountControlsMiningRisk(t *testing.T) {
	input := completeBridgeInput(t, rifbridge.ResearchStatusValidatedResearchLead)
	input.IdentityRequest.Configuration.ModelParameterCount = 3
	result, err := rifbridge.NewBridgeWithDeriver(completeFixtureDeriver(t)).EmitResearchDiagnostics(input)
	if err != nil {
		t.Fatalf("parameter-risk finding should remain a local integrity result: %v", err)
	}
	if result.IdentityStatus != researchidentity.StatusComplete || result.LocalIntegrity != rifbridge.LocalIntegrityFailed || result.EligibleForReview || result.BlockingFindings == 0 {
		t.Fatalf("bound parameter count did not block review eligibility: %#v", result)
	}
	diagnostic := readDiagnostic(t, result.ArtifactPath)
	found := false
	for _, finding := range diagnostic.BlockingFindings {
		if finding.Code == "RESEARCH_PARAMETER_MINING_RISK" {
			found = true
		}
	}
	if !found {
		t.Fatalf("parameter-risk finding is absent: %#v", diagnostic.BlockingFindings)
	}
}

func TestNonFiniteMetricInputIsBlockedBeforeSerialization(t *testing.T) {
	input := completeBridgeInput(t, rifbridge.ResearchStatusValidatedResearchLead)
	input.IdentityRequest.Returns[0] = math.NaN()
	result, err := rifbridge.NewBridgeWithDeriver(completeFixtureDeriver(t)).EmitResearchDiagnostics(input)
	var derivation *researchidentity.DerivationError
	if !errors.As(err, &derivation) || derivation.Status != researchidentity.StatusSeriesIncomplete {
		t.Fatalf("NaN metric input must return typed series failure: %v", err)
	}
	if result.ArtifactDisposition != rifbridge.ArtifactEmitted || result.Failure != rifbridge.DiagnosticsFailureIdentityDerivation || result.IdentityStatus != researchidentity.StatusSeriesIncomplete || result.EligibleForReview {
		t.Fatalf("unexpected result: %#v", result)
	}
	diagnostic := readDiagnostic(t, result.ArtifactPath)
	if diagnostic.ResearchEvidence != nil || diagnostic.EligibleForRIFReview {
		t.Fatalf("unsafe NaN values or complete identity were emitted: %#v", diagnostic)
	}
}

func TestUnknownClassificationAndMissingStemSuppressArtifact(t *testing.T) {
	for _, mutate := range []func(*rifbridge.ResearchAssessment){
		func(input *rifbridge.ResearchAssessment) { input.Classification = "unexpected" },
		func(input *rifbridge.ResearchAssessment) { input.Stem = "" },
	} {
		input := completeBridgeInput(t, rifbridge.ResearchStatusValidatedResearchLead)
		mutate(&input)
		result, err := rifbridge.NewBridgeWithDeriver(completeFixtureDeriver(t)).EmitResearchDiagnostics(input)
		if !errors.Is(err, rifbridge.ErrInvalidResearchInput) || result.ArtifactDisposition != rifbridge.ArtifactSuppressed {
			t.Fatalf("invalid input did not fail closed: result=%#v err=%v", result, err)
		}
	}
}

func TestPersistenceFailureSuppressesArtifact(t *testing.T) {
	input := completeBridgeInput(t, rifbridge.ResearchStatusValidatedResearchLead)
	root := t.TempDir()
	blockedParent := filepath.Join(root, "not-a-directory")
	if err := os.WriteFile(blockedParent, []byte("file"), 0644); err != nil {
		t.Fatal(err)
	}
	input.Stem = filepath.Join(blockedParent, "candidate")
	result, err := rifbridge.NewBridgeWithDeriver(completeFixtureDeriver(t)).EmitResearchDiagnostics(input)
	if !errors.Is(err, rifbridge.ErrResearchDiagnosticsPersistence) || result.ArtifactDisposition != rifbridge.ArtifactSuppressed || result.Failure != rifbridge.DiagnosticsFailurePersistence {
		t.Fatalf("unexpected persistence result: %#v err=%v", result, err)
	}
}

func TestDiagnosticContainsNoAcceptanceOrLifecycleAuthority(t *testing.T) {
	input := completeBridgeInput(t, rifbridge.ResearchStatusValidatedResearchLead)
	result, err := rifbridge.NewBridgeWithDeriver(completeFixtureDeriver(t)).EmitResearchDiagnostics(input)
	if err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(result.ArtifactPath)
	if err != nil {
		t.Fatal(err)
	}
	lower := strings.ToLower(string(data))
	for _, prohibited := range []string{`"is_promoted"`, `"approved"`, `"frozen"`, `"paper_ready"`, `"paper_eligible"`, `"runtime_ready"`, `"authorized"`} {
		if strings.Contains(lower, prohibited) {
			t.Fatalf("diagnostic contains prohibited authority field/status %q", prohibited)
		}
	}
	for _, suffix := range []string{".research.lock", ".research_audit.json", ".promotion_packet.json"} {
		if _, err := os.Stat(input.Stem + suffix); !os.IsNotExist(err) {
			t.Fatalf("legacy artifact exists: %s", input.Stem+suffix)
		}
	}
}

func completeBridgeInput(t *testing.T, classification string) rifbridge.ResearchAssessment {
	t.Helper()
	fixture, err := researchidentity.BuildDiagnosticSmokeFixture(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(fixture.Cleanup)
	gates := []rifbridge.ClassificationGate{{Name: "execution_series_identity", Passed: true, Critical: true}}
	switch classification {
	case rifbridge.ResearchStatusFragile:
		gates = append(gates, rifbridge.ClassificationGate{Name: "fixture_noncritical", Passed: false})
	case rifbridge.ResearchStatusRejected:
		gates = append(gates, rifbridge.ClassificationGate{Name: "fixture_critical", Passed: false, Critical: true})
	}
	return rifbridge.ResearchAssessment{
		Stem:                      filepath.Join(t.TempDir(), "candidate"),
		Classification:            classification,
		ClassificationGates:       gates,
		ExecutionSeriesGeneration: executionseries.GenerationVersion,
		IdentityRequest:           fixture.Request,
	}
}

func completeFixtureDeriver(t *testing.T) rifbridge.IdentityDeriver {
	t.Helper()
	fixture, err := researchidentity.BuildDiagnosticSmokeFixture(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(fixture.Cleanup)
	return fixture.Deriver
}

func readDiagnostic(t *testing.T, path string) rifbridge.LocalResearchDiagnostics {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	var diagnostic rifbridge.LocalResearchDiagnostics
	if err := json.Unmarshal(data, &diagnostic); err != nil {
		t.Fatal(err)
	}
	return diagnostic
}
