package rifbridge_test

import (
	"strings"
	"testing"

	"github.com/david22573/ak-engine/internal/rifbridge"
)

func TestParseResearchEvidenceValidArtifact(t *testing.T) {
	data := []byte(`{
		"schema_version":"ak.engine.research_evidence.v1",
		"candidate_id":"cand-123",
		"candidate_version":"v1",
		"research_lock_hash":"lock-hash",
		"dataset_manifest_hash":"manifest-hash"
	}`)

	evidence, err := rifbridge.ParseResearchEvidenceSummaryJSON(data)
	if err != nil {
		t.Fatalf("ParseResearchEvidenceSummaryJSON returned error: %v", err)
	}
	if evidence.CandidateID != "cand-123" {
		t.Fatalf("unexpected parsed evidence: %#v", evidence)
	}
}

func TestParseResearchEvidenceRejectsUnknownSchemaVersion(t *testing.T) {
	data := validResearchEvidenceJSON()
	data = strings.Replace(data, rifbridge.ResearchEvidenceSchemaVersion, "ak.engine.research_evidence.v2", 1)

	_, err := rifbridge.ParseResearchEvidenceSummaryJSON([]byte(data))
	if err == nil || !strings.Contains(err.Error(), "unsupported research evidence schema_version") {
		t.Fatalf("expected unsupported schema error, got %v", err)
	}
}

func TestParseResearchEvidenceRejectsMissingOrZeroRequiredFields(t *testing.T) {
	tests := []struct {
		name  string
		field string
		want  string
	}{
		{name: "candidate id", field: `"candidate_id":"cand-123",`, want: "candidate_id is required"},
		{name: "candidate version", field: `"candidate_version":"v1",`, want: "candidate_version is required"},
		{name: "research lock hash", field: `"research_lock_hash":"lock-hash",`, want: "research_lock_hash is required"},
		{name: "dataset manifest hash", field: `,"dataset_manifest_hash":"manifest-hash"`, want: "dataset_manifest_hash is required"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			data := strings.Replace(validResearchEvidenceJSON(), tt.field, "", 1)
			_, err := rifbridge.ParseResearchEvidenceSummaryJSON([]byte(data))
			if err == nil || !strings.Contains(err.Error(), tt.want) {
				t.Fatalf("expected %q error, got %v", tt.want, err)
			}
		})
	}

	data := strings.Replace(validResearchEvidenceJSON(), `"research_lock_hash":"lock-hash"`, `"research_lock_hash":"   "`, 1)
	_, err := rifbridge.ParseResearchEvidenceSummaryJSON([]byte(data))
	if err == nil || !strings.Contains(err.Error(), "research_lock_hash is required") {
		t.Fatalf("expected whitespace security hash rejection, got %v", err)
	}
}

func TestParseResearchEvidenceMalformedJSONReturnsErrorWithoutPanic(t *testing.T) {
	_, err := rifbridge.ParseResearchEvidenceSummaryJSON([]byte(`{"schema_version":`))
	if err == nil || !strings.Contains(err.Error(), "parse research evidence summary") {
		t.Fatalf("expected parse error, got %v", err)
	}
}

func validResearchEvidenceJSON() string {
	return `{"schema_version":"ak.engine.research_evidence.v1","candidate_id":"cand-123","candidate_version":"v1","research_lock_hash":"lock-hash","dataset_manifest_hash":"manifest-hash"}`
}
