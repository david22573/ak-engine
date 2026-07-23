package rifbridge

import (
	"encoding/json"
	"fmt"
)

const PromotionEvidenceSchemaVersion = "ak.rif.promotion_evidence.v1"

// PromotionEvidence is the minimal RIF artifact envelope that ak-engine may
// inspect without importing or reimplementing RIF lifecycle policy.
type PromotionEvidence struct {
	SchemaVersion          string   `json:"schema_version"`
	CandidateID            string   `json:"candidate_id"`
	CandidateVersion       string   `json:"candidate_version"`
	ResearchLockHash       string   `json:"research_lock_hash"`
	DatasetManifestHash    string   `json:"dataset_manifest_hash"`
	StrictPromotionAllowed bool     `json:"strict_promotion_allowed"`
	Warnings               []string `json:"warnings,omitempty"`
}

func ParsePromotionEvidenceJSON(data []byte) (PromotionEvidence, error) {
	var evidence PromotionEvidence
	if err := json.Unmarshal(data, &evidence); err != nil {
		return PromotionEvidence{}, fmt.Errorf("parse RIF promotion evidence: %w", err)
	}
	if evidence.SchemaVersion != PromotionEvidenceSchemaVersion {
		return PromotionEvidence{}, fmt.Errorf("unsupported RIF promotion evidence schema_version %q", evidence.SchemaVersion)
	}
	if evidence.CandidateID == "" {
		return PromotionEvidence{}, fmt.Errorf("invalid RIF promotion evidence: candidate_id is required")
	}
	if evidence.CandidateVersion == "" {
		return PromotionEvidence{}, fmt.Errorf("invalid RIF promotion evidence: candidate_version is required")
	}
	if evidence.ResearchLockHash == "" {
		return PromotionEvidence{}, fmt.Errorf("invalid RIF promotion evidence: research_lock_hash is required")
	}
	if evidence.DatasetManifestHash == "" {
		return PromotionEvidence{}, fmt.Errorf("invalid RIF promotion evidence: dataset_manifest_hash is required")
	}
	return evidence, nil
}
