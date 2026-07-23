package rifbridge_test

import (
	"strings"
	"testing"

	"github.com/david22573/ak-engine/internal/rifbridge"
)

func TestParsePromotionEvidenceValidArtifact(t *testing.T) {
	data := []byte(`{
		"schema_version":"ak.rif.promotion_evidence.v1",
		"candidate_id":"cand-123",
		"candidate_version":"v1",
		"research_lock_hash":"lock-hash",
		"dataset_manifest_hash":"manifest-hash",
		"strict_promotion_allowed":true
	}`)

	evidence, err := rifbridge.ParsePromotionEvidenceJSON(data)
	if err != nil {
		t.Fatalf("ParsePromotionEvidenceJSON returned error: %v", err)
	}
	if evidence.CandidateID != "cand-123" || !evidence.StrictPromotionAllowed {
		t.Fatalf("unexpected parsed evidence: %#v", evidence)
	}
}

func TestParsePromotionEvidenceRejectsUnknownSchemaVersion(t *testing.T) {
	data := []byte(`{
		"schema_version":"ak.rif.promotion_evidence.v2",
		"candidate_id":"cand-123",
		"candidate_version":"v1",
		"research_lock_hash":"lock-hash",
		"dataset_manifest_hash":"manifest-hash"
	}`)

	_, err := rifbridge.ParsePromotionEvidenceJSON(data)
	if err == nil || !strings.Contains(err.Error(), "unsupported RIF promotion evidence schema_version") {
		t.Fatalf("expected unsupported schema error, got %v", err)
	}
}

func TestParsePromotionEvidenceFailsClosedOnMissingRequiredField(t *testing.T) {
	data := []byte(`{
		"schema_version":"ak.rif.promotion_evidence.v1",
		"candidate_id":"cand-123",
		"candidate_version":"v1",
		"dataset_manifest_hash":"manifest-hash"
	}`)

	_, err := rifbridge.ParsePromotionEvidenceJSON(data)
	if err == nil || !strings.Contains(err.Error(), "research_lock_hash is required") {
		t.Fatalf("expected required field error, got %v", err)
	}
}

func TestParsePromotionEvidenceMalformedJSONReturnsError(t *testing.T) {
	_, err := rifbridge.ParsePromotionEvidenceJSON([]byte(`{"schema_version":`))
	if err == nil || !strings.Contains(err.Error(), "parse RIF promotion evidence") {
		t.Fatalf("expected parse error, got %v", err)
	}
}
