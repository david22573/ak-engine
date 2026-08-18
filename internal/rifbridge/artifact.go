package rifbridge

import (
	"encoding/json"
	"fmt"
	"strings"
)

const ResearchEvidenceSchemaVersion = "ak.engine.research_evidence.v1"

// ResearchEvidenceSummary is the minimal serialized research evidence envelope that ak-engine inspects.
type ResearchEvidenceSummary struct {
	SchemaVersion       string   `json:"schema_version"`
	CandidateID         string   `json:"candidate_id"`
	CandidateVersion    string   `json:"candidate_version"`
	ResearchLockHash    string   `json:"research_lock_hash"`
	DatasetManifestHash string   `json:"dataset_manifest_hash"`
	Warnings            []string `json:"warnings,omitempty"`
}

// ParseResearchEvidenceSummaryJSON parses and validates the serialized boundary.
func ParseResearchEvidenceSummaryJSON(data []byte) (ResearchEvidenceSummary, error) {
	var evidence ResearchEvidenceSummary
	if err := json.Unmarshal(data, &evidence); err != nil {
		return ResearchEvidenceSummary{}, fmt.Errorf("parse research evidence summary: %w", err)
	}
	if evidence.SchemaVersion != ResearchEvidenceSchemaVersion {
		return ResearchEvidenceSummary{}, fmt.Errorf("unsupported research evidence schema_version %q", evidence.SchemaVersion)
	}
	if strings.TrimSpace(evidence.CandidateID) == "" {
		return ResearchEvidenceSummary{}, fmt.Errorf("invalid research evidence: candidate_id is required")
	}
	if strings.TrimSpace(evidence.CandidateVersion) == "" {
		return ResearchEvidenceSummary{}, fmt.Errorf("invalid research evidence: candidate_version is required")
	}
	if strings.TrimSpace(evidence.ResearchLockHash) == "" {
		return ResearchEvidenceSummary{}, fmt.Errorf("invalid research evidence: research_lock_hash is required")
	}
	if strings.TrimSpace(evidence.DatasetManifestHash) == "" {
		return ResearchEvidenceSummary{}, fmt.Errorf("invalid research evidence: dataset_manifest_hash is required")
	}
	return evidence, nil
}
