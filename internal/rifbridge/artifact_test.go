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
	data := validPromotionEvidenceJSON()
	data = strings.Replace(data, rifbridge.PromotionEvidenceSchemaVersion, "ak.rif.promotion_evidence.v2", 1)

	_, err := rifbridge.ParsePromotionEvidenceJSON([]byte(data))
	if err == nil || !strings.Contains(err.Error(), "unsupported RIF promotion evidence schema_version") {
		t.Fatalf("expected unsupported schema error, got %v", err)
	}
}

func TestParsePromotionEvidenceRejectsMissingOrZeroRequiredFields(t *testing.T) {
	tests := []struct {
		name  string
		field string
		want  string
	}{
		{name: "candidate id", field: `"candidate_id":"cand-123",`, want: "candidate_id is required"},
		{name: "candidate version", field: `"candidate_version":"v1",`, want: "candidate_version is required"},
		{name: "research lock hash", field: `"research_lock_hash":"lock-hash",`, want: "research_lock_hash is required"},
		{name: "dataset manifest hash", field: `"dataset_manifest_hash":"manifest-hash",`, want: "dataset_manifest_hash is required"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			data := strings.Replace(validPromotionEvidenceJSON(), tt.field, "", 1)
			_, err := rifbridge.ParsePromotionEvidenceJSON([]byte(data))
			if err == nil || !strings.Contains(err.Error(), tt.want) {
				t.Fatalf("expected %q error, got %v", tt.want, err)
			}
		})
	}

	data := strings.Replace(validPromotionEvidenceJSON(), `"research_lock_hash":"lock-hash"`, `"research_lock_hash":"   "`, 1)
	_, err := rifbridge.ParsePromotionEvidenceJSON([]byte(data))
	if err == nil || !strings.Contains(err.Error(), "research_lock_hash is required") {
		t.Fatalf("expected whitespace security hash rejection, got %v", err)
	}
}

func TestParsePromotionEvidenceMalformedJSONReturnsErrorWithoutPanic(t *testing.T) {
	_, err := rifbridge.ParsePromotionEvidenceJSON([]byte(`{"schema_version":`))
	if err == nil || !strings.Contains(err.Error(), "parse RIF promotion evidence") {
		t.Fatalf("expected parse error, got %v", err)
	}
}

func validPromotionEvidenceJSON() string {
	return `{
		"schema_version":"ak.rif.promotion_evidence.v1",
		"candidate_id":"cand-123",
		"candidate_version":"v1",
		"research_lock_hash":"lock-hash",
		"dataset_manifest_hash":"manifest-hash",
		"strict_promotion_allowed":false
	}`
}
