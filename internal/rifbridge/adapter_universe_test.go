package rifbridge

import (
	"errors"
	"testing"

	"github.com/david22573/ak-engine/internal/researchidentity"
)

func TestNormalizeIdentityAssessmentRejectsUnknownStatus(t *testing.T) {
	assessment, err := normalizeIdentityAssessment(researchidentity.Assessment{Status: "FUTURE_STATUS"}, nil)
	if err == nil || assessment.Status != researchidentity.StatusValidationFailed || assessment.Identity != nil || len(assessment.Findings) != 1 {
		t.Fatalf("unknown status did not fail closed: %#v err=%v", assessment, err)
	}
}

func TestNormalizeIdentityAssessmentRejectsInvalidCompleteShape(t *testing.T) {
	assessment, err := normalizeIdentityAssessment(researchidentity.Assessment{Status: researchidentity.StatusComplete}, nil)
	if err == nil || assessment.Status != researchidentity.StatusValidationFailed || assessment.Identity != nil {
		t.Fatalf("invalid complete result did not fail closed: %#v err=%v", assessment, err)
	}
}

func TestNormalizeIdentityAssessmentAddsErrorForIncompleteStatus(t *testing.T) {
	assessment, err := normalizeIdentityAssessment(researchidentity.Assessment{Status: researchidentity.StatusDatasetIncomplete}, nil)
	var derivation *researchidentity.DerivationError
	if !errors.As(err, &derivation) || derivation.Status != researchidentity.StatusDatasetIncomplete || len(assessment.Findings) != 1 {
		t.Fatalf("incomplete status was not normalized: %#v err=%v", assessment, err)
	}
}

func TestBoundMetricSeriesReproducesMetrics(t *testing.T) {
	returns := make([]float64, 40)
	for i := range returns {
		returns[i] = float64((i%5)-2) / 1000
	}
	first, err := calculateMetrics(returns, 0.05, 365)
	if err != nil {
		t.Fatal(err)
	}
	boundCopy := append([]float64(nil), returns...)
	second, err := calculateMetrics(boundCopy, 0.05, 365)
	if err != nil {
		t.Fatal(err)
	}
	if first != second {
		t.Fatalf("same bound series did not reproduce metrics: %#v %#v", first, second)
	}
	boundCopy[0] += 0.01
	changed, err := calculateMetrics(boundCopy, 0.05, 365)
	if err != nil {
		t.Fatal(err)
	}
	if changed == first {
		t.Fatal("changed bound series did not change metrics")
	}
}
