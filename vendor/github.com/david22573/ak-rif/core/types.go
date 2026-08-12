package core

import "time"

// ResearchIdentitySchemaVersion identifies the immutable candidate research
// registration contract. Every field is caller-supplied; RIF never derives
// identity from paths, environment variables, reports, or mutable aliases.
const (
	ResearchIdentitySchemaVersion   = "ak.rif.research_identity.v1"
	ResearchIdentitySchemaVersionV2 = "ak.rif.research_identity.v2"
	ResearchIdentitySchemaVersionV3 = "ak.rif.research_identity.v3"
)

// ResearchIdentity binds a candidate version to the exact data revision,
// half-open UTC research window, source-availability cutoff, archived manifest,
// and policy versions used by the research run.
type ResearchIdentity struct {
	SchemaVersion             string    `json:"schema_version"`
	DatasetID                 string    `json:"dataset_id"`
	DatasetVersion            string    `json:"dataset_version"`
	ResearchWindowStart       time.Time `json:"research_window_start"`
	ResearchWindowEnd         time.Time `json:"research_window_end"`
	EvaluationCutoff          time.Time `json:"evaluation_cutoff"`
	ManifestID                string    `json:"manifest_id"`
	ManifestHash              string    `json:"manifest_hash"`
	AvailabilityPolicyVersion string    `json:"availability_policy_version"`
	CoveragePolicyVersion     string    `json:"coverage_policy_version"`
	IndependencePolicyHash    string    `json:"independence_policy_hash,omitempty"`
	UncertaintyMethodHash     string    `json:"uncertainty_method_hash,omitempty"`
	SourceSchemaHash          string    `json:"source_schema_hash,omitempty"`
	ManifestContractHash      string    `json:"manifest_contract_hash,omitempty"`
	GovernanceDecisionHash    string    `json:"governance_decision_hash,omitempty"`
}

// DatasetType identifies the intended use of a dataset.
type DatasetType string

const (
	TrainingDataset     DatasetType = "TRAINING"
	ValidationDataset   DatasetType = "VALIDATION"
	WalkForwardDataset  DatasetType = "WALK_FORWARD"
	FinalHoldoutDataset DatasetType = "FINAL_HOLDOUT"
)

// Dataset represents an immutable record of a data slice used in research.
type Dataset struct {
	ID        string
	Type      DatasetType
	Hash      string
	Frozen    bool
	CreatedAt time.Time
}

// Experiment represents a verifiable, reproducible run of a strategy candidate.
type Experiment struct {
	ExperimentID     string
	GitCommitHash    string
	DatasetHash      string
	FeatureHash      string
	ParameterHash    string
	CandidateVersion string
	Timestamp        time.Time
	RandomSeed       int64
	EngineVersion    string
}

// CandidateState tracks the immutable lifecycle state of a strategy.
type CandidateState string

const (
	StateDiscovery      CandidateState = "DISCOVERY"
	StateResearch       CandidateState = "RESEARCH"
	StateValidated      CandidateState = "VALIDATED"
	StatePaperReady     CandidateState = "PAPER_READY"
	StateShadowReady    CandidateState = "SHADOW_READY"
	StateExecutionReady CandidateState = "EXECUTION_READY"
	StateRejected       CandidateState = "REJECTED"
	StateRetired        CandidateState = "RETIRED"
)

// Candidate represents a trading strategy or model progressing through the research pipeline.
type Candidate struct {
	ID               string            `json:"candidate_id"`
	Version          string            `json:"candidate_version"`
	ResearchIdentity *ResearchIdentity `json:"research_identity"`
	State            CandidateState    `json:"lifecycle_state"`
	LifecycleEpoch   uint64            `json:"lifecycle_epoch"`
	CreatedAt        time.Time         `json:"created_at"`
}
