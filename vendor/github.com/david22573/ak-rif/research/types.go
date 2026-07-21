// Package research implements the additive V4 pre-development research
// governance authority. It is deliberately separate from RIF's accepted
// candidate-promotion lifecycle and does not migrate V1-V3 records.
package research

import (
	"encoding/json"
	"time"
)

const (
	IdentitySchemaVersion        = "ak.rif.research_identity.v4"
	StoreSchemaVersion           = "ak.rif.research_governance_store.v1"
	LifecycleRecordVersion       = "ak.rif.research_lifecycle_record.v1"
	ReservationSchemaVersion     = "ak.rif.holdout_reservation.v1"
	AuthorizationSchemaVersion   = "ak.rif.partition_authorization.v1"
	AccessReceiptSchemaVersion   = "ak.rif.partition_access_receipt.v1"
	EnvelopeSchemaVersion        = "ak.rif.research_governance_envelope.v1"
	StoreSchemaVersionV2         = "ak.rif.research_governance_store.v2"
	EnvelopeSchemaVersionV2      = "ak.rif.research_governance_envelope.v2"
	RunnerIdentityVersion        = "ak.rif.runner_implementation_identity.v1"
	RegisteredConfigVersion      = "ak.rif.registered_configuration_identity.v1"
	StageExecutionPlanVersion    = "ak.rif.stage_execution_plan.v1"
	StageExecutionSetVersion     = "ak.rif.stage_execution_set.v1"
	StageAuthorizationVersion    = "ak.rif.stage_variant_authorization.v1"
	StageAccessReceiptVersion    = "ak.rif.stage_variant_access_receipt.v1"
	StageExecutionReceiptVersion = "ak.rif.stage_execution_receipt.v1"
	StageRetryProofVersion       = "ak.rif.zero_access_retry_proof.v1"
	StageManifestVersion         = "ak.rif.stage_completion_manifest.v1"
	StageEnvelopeVersion         = "ak.rif.stage_execution_envelope.v1"
	ExecutionResultVersion       = "ak.rif.execution_result_envelope.v1"
	NomineeSelectionVersion      = "ak.rif.development_nominee_selection.v1"

	AcceptedIndependenceID    = "ak.engine.independence.downtrend-midvol-relief.v3"
	AcceptedIndependenceHash  = "sha256:84a6863b354b453dbe13698b9854ec4adcd116466a0831e7107efb892042cc1f"
	AcceptedUncertaintyID     = "ak.engine.uncertainty.cluster-bootstrap.v2"
	AcceptedUncertaintyHash   = "sha256:1a91541c94378cc6f34e62a39ae504d3d013b5dab63a2b622641cdd1088148fb"
	AcceptedConcentrationID   = "ak.engine.governance.concentration-decision.v1"
	AcceptedConcentrationHash = "sha256:a126849e4cc0bd6457cf3f11079c4e3e2865ffce6c53a95ce92fa250130d39d5"

	MaxRegisteredVariants = 12
)

type LifecycleState string

const (
	StateIdentityRegistered       LifecycleState = "RESEARCH_IDENTITY_REGISTERED"
	StateHoldoutReserved          LifecycleState = "HOLDOUT_RESERVED"
	StateDevelopmentAuthorized    LifecycleState = "DEVELOPMENT_AUTHORIZED"
	StateDevelopmentSealed        LifecycleState = "DEVELOPMENT_SEALED"
	StateDevelopmentSetAuthorized LifecycleState = "DEVELOPMENT_SET_AUTHORIZED"
	StateDevelopmentSetSealed     LifecycleState = "DEVELOPMENT_SET_SEALED"
	StateValidationAuthorized     LifecycleState = "VALIDATION_AUTHORIZED"
	StateValidationSealed         LifecycleState = "VALIDATION_SEALED"
	StateValidationSetAuthorized  LifecycleState = "VALIDATION_SET_AUTHORIZED"
	StateValidationSetSealed      LifecycleState = "VALIDATION_SET_SEALED"
	StateCandidateFrozen          LifecycleState = "CANDIDATE_FROZEN"
	StateFinalAuthorized          LifecycleState = "FINAL_HOLDOUT_AUTHORIZED"
	StateFinalSealed              LifecycleState = "FINAL_HOLDOUT_SEALED"
	StateQualified                LifecycleState = "QUALIFIED"
	StateRejected                 LifecycleState = "REJECTED"
	StateBlocked                  LifecycleState = "BLOCKED"
)

type PartitionName string

const (
	PartitionDevelopment  PartitionName = "DEVELOPMENT"
	PartitionValidation   PartitionName = "VALIDATION"
	PartitionFinalHoldout PartitionName = "FINAL_HOLDOUT"
)

type HashIdentity struct {
	ID     string `json:"id"`
	SHA256 string `json:"sha256"`
}

type RepositoryIdentity struct {
	EngineStartingCommit    string `json:"engine_starting_commit"`
	HistorianStartingCommit string `json:"historian_starting_commit"`
	RIFStartingCommit       string `json:"rif_starting_commit"`
	ProtocolGitCommit       string `json:"protocol_git_commit"`
	RunnerGitCommit         string `json:"evaluation_runner_git_commit"`
	RunnerExecutableSHA256  string `json:"evaluation_runner_executable_sha256"`
}

type ProtocolIdentity struct {
	ID                       string `json:"id"`
	SHA256                   string `json:"sha256"`
	ContentAddressedIdentity string `json:"content_addressed_identity"`
	SchemaVersion            string `json:"schema_version"`
}

type CandidateScope struct {
	FamilyID        string `json:"candidate_family_id"`
	StrategySide    string `json:"permitted_strategy_side"`
	Horizon         string `json:"permitted_horizon"`
	SemanticsFrozen bool   `json:"candidate_semantics_frozen"`
}

type Interval struct {
	Start time.Time `json:"start_inclusive"`
	End   time.Time `json:"end_exclusive"`
}

type DatasetIdentity struct {
	Checkpoint                HashIdentity `json:"checkpoint"`
	SourceIdentitySHA256      string       `json:"source_identity_sha256"`
	ReacquisitionProtocol     HashIdentity `json:"reacquisition_protocol"`
	PreAcquisitionSealSHA256  string       `json:"pre_acquisition_seal_sha256"`
	SealedBinarySHA256        string       `json:"sealed_binary_sha256"`
	AbandonedEvidenceRegistry HashIdentity `json:"abandoned_evidence_registry"`
	HistorianCheckpointCommit string       `json:"historian_checkpoint_commit"`
	RequiredSymbols           []string     `json:"dataset_required_symbols"`
	CandidateTargetSymbols    []string     `json:"candidate_target_symbols"`
	ContextOnlySymbols        []string     `json:"context_only_symbols"`
	UniverseContractSHA256    string       `json:"universe_contract_sha256"`
	EligibleInterval          Interval     `json:"eligible_interval"`
	ProhibitedPriorExposure   []Interval   `json:"prohibited_prior_exposure_intervals"`
	AvailabilityCutoff        time.Time    `json:"availability_cutoff"`
}

type Partition struct {
	Name                         PartitionName `json:"name"`
	Interval                     Interval      `json:"interval"`
	StructuralDayCount           int           `json:"structural_day_count"`
	RequiredSymbolCoverageSHA256 string        `json:"required_symbol_coverage_sha256"`
}

type Variant struct {
	ID                  string   `json:"id"`
	ConfigurationSHA256 string   `json:"canonical_configuration_sha256"`
	Dimensions          []string `json:"configuration_dimensions"`
}

type StabilityNeighborhood struct {
	VariantID   string   `json:"variant_id"`
	NeighborIDs []string `json:"neighbor_ids"`
}

type VariantLedger struct {
	Variants                  []Variant               `json:"variants"`
	MaximumRegisteredVariants int                     `json:"maximum_registered_variant_count"`
	V00ID                     string                  `json:"v00_id"`
	PermittedDimensions       []string                `json:"permitted_configuration_dimensions"`
	DevelopmentNomineeRule    string                  `json:"deterministic_development_nominee_selection_rule"`
	StabilityNeighborhoods    []StabilityNeighborhood `json:"registered_stability_neighborhoods"`
}

type AuthorityIdentity struct {
	Independence            HashIdentity   `json:"independence"`
	Uncertainty             HashIdentity   `json:"uncertainty"`
	ConcentrationSHA256     string         `json:"concentration_governance_sha256"`
	QualificationGateSet    HashIdentity   `json:"qualification_gate_set"`
	QualificationGateHashes []HashIdentity `json:"qualification_gate_hashes"`
	TransactionCostPolicy   HashIdentity   `json:"transaction_cost_policy"`
	DeterministicSeedPolicy HashIdentity   `json:"deterministic_seed_policy"`
}

type AccessPolicy struct {
	NoAccessBeforeReservation        bool     `json:"no_partition_access_before_reservation"`
	DevelopmentPrerequisites         []string `json:"development_access_prerequisites"`
	ValidationPrerequisites          []string `json:"validation_access_prerequisites"`
	FinalHoldoutPrerequisites        []string `json:"final_holdout_access_prerequisites"`
	CandidateFreezeRequirements      []string `json:"candidate_freeze_requirements"`
	PermittedAccessCountPerPartition int      `json:"permitted_access_count_per_partition"`
	RetryPolicy                      string   `json:"retry_policy"`
	DurableAccessReceiptRequired     bool     `json:"durable_access_receipt_required"`
}

type IdentityV4 struct {
	SchemaVersion  string             `json:"schema_version"`
	ResearchID     string             `json:"research_id"`
	Repositories   RepositoryIdentity `json:"repository_identity"`
	Protocol       ProtocolIdentity   `json:"protocol_identity"`
	CandidateScope CandidateScope     `json:"candidate_scope"`
	Dataset        DatasetIdentity    `json:"dataset_identity"`
	Partitions     []Partition        `json:"partitions"`
	VariantLedger  VariantLedger      `json:"variant_governance"`
	Authorities    AuthorityIdentity  `json:"authority_identity"`
	AccessPolicy   AccessPolicy       `json:"access_and_lifecycle_policy"`
}

type ReservationRequest struct {
	SchemaVersion        string    `json:"schema_version"`
	ResearchIdentityHash string    `json:"research_identity_hash"`
	FinalHoldout         Partition `json:"final_holdout"`
	ProtocolSHA256       string    `json:"protocol_sha256"`
	CheckpointSHA256     string    `json:"checkpoint_sha256"`
	VariantLedgerSHA256  string    `json:"variant_ledger_sha256"`
	AuthoritySetSHA256   string    `json:"authority_set_sha256"`
	ExpectedSequence     uint64    `json:"expected_sequence"`
	ExpectedStateHash    string    `json:"expected_state_hash"`
}

type ReservationRecord struct {
	SchemaVersion        string    `json:"schema_version"`
	ReservationID        string    `json:"reservation_id"`
	ResearchIdentityHash string    `json:"research_identity_hash"`
	FinalHoldout         Partition `json:"final_holdout"`
	ProtocolSHA256       string    `json:"protocol_sha256"`
	CheckpointSHA256     string    `json:"checkpoint_sha256"`
	VariantLedgerSHA256  string    `json:"variant_ledger_sha256"`
	AuthoritySetSHA256   string    `json:"authority_set_sha256"`
	CandidateFrozen      bool      `json:"candidate_frozen"`
	CreatedAt            time.Time `json:"created_at"`
	RecordHash           string    `json:"record_hash"`
}

type ExecutionBinding struct {
	VariantID               string        `json:"variant_id"`
	ConfigurationSHA256     string        `json:"configuration_sha256"`
	ProtocolSHA256          string        `json:"protocol_sha256"`
	CheckpointSHA256        string        `json:"checkpoint_sha256"`
	IndependenceSHA256      string        `json:"independence_sha256"`
	UncertaintySHA256       string        `json:"uncertainty_sha256"`
	ConcentrationSHA256     string        `json:"concentration_sha256"`
	QualificationGateSHA256 string        `json:"qualification_gate_set_sha256"`
	RunnerGitCommit         string        `json:"runner_git_commit"`
	RunnerExecutableSHA256  string        `json:"runner_executable_sha256"`
	Partition               PartitionName `json:"partition"`
}

type TransitionRequest struct {
	ExpectedSequence  uint64           `json:"expected_sequence"`
	ExpectedStateHash string           `json:"expected_state_hash"`
	Binding           ExecutionBinding `json:"execution_binding"`
}

type SealRequest struct {
	ExpectedSequence       uint64           `json:"expected_sequence"`
	ExpectedStateHash      string           `json:"expected_state_hash"`
	Binding                ExecutionBinding `json:"execution_binding"`
	AccessReceiptHash      string           `json:"access_receipt_hash"`
	ExecutionReceiptSHA256 string           `json:"execution_receipt_sha256"`
	ResultSealSHA256       string           `json:"result_seal_sha256"`
}

type FrozenCandidate struct {
	VariantID               string    `json:"variant_id"`
	ConfigurationSHA256     string    `json:"configuration_sha256"`
	ExecutableSHA256        string    `json:"executable_sha256"`
	ProtocolSHA256          string    `json:"protocol_sha256"`
	CheckpointSHA256        string    `json:"checkpoint_sha256"`
	IndependenceSHA256      string    `json:"independence_sha256"`
	UncertaintySHA256       string    `json:"uncertainty_sha256"`
	ConcentrationSHA256     string    `json:"concentration_sha256"`
	QualificationGateSHA256 string    `json:"qualification_gate_set_sha256"`
	NoUnresolvedDefaults    bool      `json:"no_unresolved_defaults"`
	FrozenAt                time.Time `json:"frozen_at"`
	FrozenIdentityHash      string    `json:"frozen_identity_hash"`
}

type AuthorizationRecord struct {
	SchemaVersion           string           `json:"schema_version"`
	AuthorizationID         string           `json:"authorization_id"`
	Sequence                uint64           `json:"sequence"`
	ResearchIdentityHash    string           `json:"research_identity_hash"`
	LifecycleState          LifecycleState   `json:"lifecycle_state"`
	Binding                 ExecutionBinding `json:"execution_binding"`
	IssuedAt                time.Time        `json:"issued_at"`
	ExpiresAt               *time.Time       `json:"expires_at,omitempty"`
	OneShot                 bool             `json:"one_shot"`
	PriorLifecycleStateHash string           `json:"prior_lifecycle_state_hash"`
	PreviousHash            string           `json:"previous_hash"`
	RecordHash              string           `json:"record_hash"`
}

type AccessReceipt struct {
	SchemaVersion           string           `json:"schema_version"`
	Sequence                uint64           `json:"sequence"`
	AuthorizationID         string           `json:"authorization_id"`
	ResearchIdentityHash    string           `json:"research_identity_hash"`
	Binding                 ExecutionBinding `json:"execution_binding"`
	AccessedAt              time.Time        `json:"accessed_at"`
	PriorLifecycleStateHash string           `json:"prior_lifecycle_state_hash"`
	PreviousHash            string           `json:"previous_hash"`
	RecordHash              string           `json:"record_hash"`
}

type LifecycleRecord struct {
	SchemaVersion  string         `json:"schema_version"`
	Sequence       uint64         `json:"sequence"`
	EventType      string         `json:"event_type"`
	FromState      LifecycleState `json:"from_state,omitempty"`
	ToState        LifecycleState `json:"to_state"`
	OccurredAt     time.Time      `json:"occurred_at"`
	EvidenceSHA256 string         `json:"evidence_sha256"`
	PriorStateHash string         `json:"prior_state_hash"`
	PreviousHash   string         `json:"previous_hash"`
	RecordHash     string         `json:"record_hash"`
}

type Disposition struct {
	State  LifecycleState `json:"state"`
	Kind   string         `json:"kind"`
	Reason string         `json:"reason"`
}

type Snapshot struct {
	SchemaVersion      string                `json:"schema_version"`
	Identity           *IdentityV4           `json:"research_identity,omitempty"`
	IdentityHash       string                `json:"research_identity_hash,omitempty"`
	Reservation        *ReservationRecord    `json:"holdout_reservation,omitempty"`
	State              LifecycleState        `json:"lifecycle_state,omitempty"`
	Sequence           uint64                `json:"sequence"`
	FrozenCandidate    *FrozenCandidate      `json:"frozen_candidate,omitempty"`
	Disposition        *Disposition          `json:"disposition,omitempty"`
	Authorizations     []AuthorizationRecord `json:"authorizations"`
	AccessReceipts     []AccessReceipt       `json:"access_receipts"`
	StageExecutionSets []StageExecutionSet   `json:"stage_execution_sets,omitempty"`
	DevelopmentNominee *NomineeSelection     `json:"development_nominee,omitempty"`
	LifecycleHistory   []LifecycleRecord     `json:"lifecycle_history"`
	IntegrityHash      string                `json:"integrity_hash"`
}

// RunnerImplementationIdentity contains only deterministic, data-independent
// properties that exist before any partition authorization or access.
type RunnerImplementationIdentity struct {
	SchemaVersion     string       `json:"schema_version"`
	SourceCommit      string       `json:"source_commit"`
	PackageIdentity   HashIdentity `json:"package_identity"`
	BuildInputsSHA256 string       `json:"deterministic_build_inputs_sha256"`
	CompilerIdentity  string       `json:"compiler_identity"`
	BuildModeIdentity HashIdentity `json:"build_mode_identity"`
	BinarySHA256      string       `json:"runner_binary_sha256"`
}

type RegisteredConfigurationIdentity struct {
	SchemaVersion          string          `json:"schema_version"`
	VariantID              string          `json:"variant_id"`
	CanonicalConfiguration json.RawMessage `json:"canonical_configuration"`
	ConfigurationSHA256    string          `json:"canonical_configuration_sha256"`
	CandidateFamilyID      string          `json:"candidate_family_id"`
	ProtocolID             string          `json:"protocol_id"`
	ProtocolSHA256         string          `json:"protocol_sha256"`
}

type StageExecutionPlan struct {
	SchemaVersion           string                            `json:"schema_version"`
	ResearchIdentityHash    string                            `json:"research_identity_hash"`
	Protocol                ProtocolIdentity                  `json:"protocol_identity"`
	Stage                   PartitionName                     `json:"stage"`
	Partition               Partition                         `json:"partition"`
	Checkpoint              HashIdentity                      `json:"checkpoint"`
	DatasetIdentitySHA256   string                            `json:"dataset_identity_sha256"`
	Runner                  RunnerImplementationIdentity      `json:"runner_identity"`
	Configurations          []RegisteredConfigurationIdentity `json:"ordered_registered_configurations"`
	DeterministicSeedPolicy HashIdentity                      `json:"deterministic_seed_policy"`
	ExpectedExecutions      int                               `json:"expected_execution_cardinality"`
	Complete                bool                              `json:"complete_set_required"`
	OrderingRule            string                            `json:"deterministic_ordering_rule"`
	Authorities             AuthorityIdentity                 `json:"authority_identities"`
	GateSet                 HashIdentity                      `json:"gate_set_identity"`
	PlanHash                string                            `json:"plan_hash"`
}

type StageExecutionSetRequest struct {
	SchemaVersion     string                            `json:"schema_version"`
	ExpectedSequence  uint64                            `json:"expected_sequence"`
	ExpectedStateHash string                            `json:"expected_state_hash"`
	Runner            RunnerImplementationIdentity      `json:"runner_identity"`
	Configurations    []RegisteredConfigurationIdentity `json:"ordered_registered_configurations"`
	OrderingRule      string                            `json:"deterministic_ordering_rule"`
	Complete          bool                              `json:"complete_set_required"`
}

type StageVariantAuthorization struct {
	SchemaVersion   string                          `json:"schema_version"`
	AuthorizationID string                          `json:"authorization_id"`
	ExecutionSetID  string                          `json:"execution_set_id"`
	PlanHash        string                          `json:"plan_hash"`
	Ordinal         int                             `json:"ordinal"`
	Attempt         int                             `json:"attempt"`
	Configuration   RegisteredConfigurationIdentity `json:"registered_configuration"`
	Runner          RunnerImplementationIdentity    `json:"runner_identity"`
	Partition       Partition                       `json:"partition"`
	Protocol        ProtocolIdentity                `json:"protocol_identity"`
	Checkpoint      HashIdentity                    `json:"checkpoint"`
	Authorities     AuthorityIdentity               `json:"authority_identities"`
	GateSet         HashIdentity                    `json:"gate_set_identity"`
	IssuedAt        time.Time                       `json:"issued_at"`
	PriorStateHash  string                          `json:"prior_state_hash"`
	PreviousHash    string                          `json:"previous_hash"`
	RecordHash      string                          `json:"record_hash"`
}

type StageVariantAccessReceipt struct {
	SchemaVersion   string    `json:"schema_version"`
	ExecutionSetID  string    `json:"execution_set_id"`
	AuthorizationID string    `json:"authorization_id"`
	VariantID       string    `json:"variant_id"`
	Attempt         int       `json:"attempt"`
	ConsumedAt      time.Time `json:"consumed_at"`
	PriorStateHash  string    `json:"prior_state_hash"`
	PreviousHash    string    `json:"previous_hash"`
	RecordHash      string    `json:"record_hash"`
}

type ZeroAccessRetryProof struct {
	SchemaVersion          string    `json:"schema_version"`
	ExecutionSetID         string    `json:"execution_set_id"`
	PriorAuthorizationID   string    `json:"prior_authorization_id"`
	PriorAccessReceiptHash string    `json:"prior_access_receipt_hash"`
	VariantID              string    `json:"variant_id"`
	RowsAccessed           int       `json:"rows_accessed"`
	OutcomeArtifacts       int       `json:"outcome_artifacts"`
	DurableProofSHA256     string    `json:"durable_proof_sha256"`
	ProvenAt               time.Time `json:"proven_at"`
	PreviousHash           string    `json:"previous_hash"`
	RecordHash             string    `json:"record_hash"`
}

type AuthorityInvocationEvidence struct {
	Identity       HashIdentity `json:"identity"`
	Invoked        bool         `json:"invoked"`
	EvidenceSHA256 string       `json:"evidence_sha256"`
}

type ExecutionResultEnvelope struct {
	SchemaVersion        string                          `json:"schema_version"`
	ExecutionSetID       string                          `json:"execution_set_id"`
	PlanHash             string                          `json:"plan_hash"`
	AuthorizationID      string                          `json:"authorization_id"`
	DeterministicRunID   string                          `json:"deterministic_run_id"`
	Configuration        RegisteredConfigurationIdentity `json:"registered_configuration"`
	Runner               RunnerImplementationIdentity    `json:"runner_identity"`
	Partition            Partition                       `json:"partition"`
	Protocol             ProtocolIdentity                `json:"protocol_identity"`
	Checkpoint           HashIdentity                    `json:"checkpoint"`
	Authorities          AuthorityIdentity               `json:"authority_identities"`
	GateSet              HashIdentity                    `json:"gate_set_identity"`
	AccessReceiptHash    string                          `json:"access_receipt_hash"`
	ResultArtifact       json.RawMessage                 `json:"result_artifact"`
	ResultArtifactSHA256 string                          `json:"result_artifact_sha256"`
	OutputManifestSHA256 string                          `json:"output_manifest_sha256"`
	AuthorityInvocations []AuthorityInvocationEvidence   `json:"authority_invocation_evidence"`
	ResultStatus         string                          `json:"result_status"`
	MandatoryGatesPassed bool                            `json:"mandatory_gates_passed"`
	EnvelopeHash         string                          `json:"envelope_hash"`
}

type StageExecutionReceipt struct {
	SchemaVersion           string        `json:"schema_version"`
	ExecutionSetID          string        `json:"execution_set_id"`
	PlanHash                string        `json:"plan_hash"`
	AuthorizationID         string        `json:"authorization_id"`
	DeterministicRunID      string        `json:"deterministic_run_id"`
	VariantID               string        `json:"variant_id"`
	ConfigurationSHA256     string        `json:"configuration_sha256"`
	RunnerIdentitySHA256    string        `json:"runner_identity_sha256"`
	Partition               PartitionName `json:"partition"`
	CheckpointSHA256        string        `json:"checkpoint_sha256"`
	AccessReceiptHash       string        `json:"access_receipt_hash"`
	ResultArtifactSHA256    string        `json:"result_artifact_sha256"`
	OutputManifestSHA256    string        `json:"output_manifest_sha256"`
	AuthorityEvidenceSHA256 string        `json:"authority_evidence_sha256"`
	ResultStatus            string        `json:"result_status"`
	MandatoryGatesPassed    bool          `json:"mandatory_gates_passed"`
	CompletedAt             time.Time     `json:"completed_at"`
	PreviousHash            string        `json:"previous_hash"`
	RecordHash              string        `json:"record_hash"`
}

type StageCompletionManifest struct {
	SchemaVersion        string        `json:"schema_version"`
	ExecutionSetID       string        `json:"execution_set_id"`
	PlanHash             string        `json:"plan_hash"`
	Stage                PartitionName `json:"stage"`
	OrderedVariantIDs    []string      `json:"ordered_variant_ids"`
	OrderedReceiptHashes []string      `json:"ordered_execution_receipt_hashes"`
	OrderedResultHashes  []string      `json:"ordered_result_artifact_hashes"`
	ManifestHash         string        `json:"manifest_hash"`
}

type StageExecutionSet struct {
	SchemaVersion      string                      `json:"schema_version"`
	ExecutionSetID     string                      `json:"execution_set_id"`
	Plan               StageExecutionPlan          `json:"execution_plan"`
	IssuanceState      string                      `json:"issuance_state"`
	CompletionState    string                      `json:"completion_state"`
	Authorizations     []StageVariantAuthorization `json:"variant_authorizations"`
	AccessReceipts     []StageVariantAccessReceipt `json:"access_receipts"`
	RetryProofs        []ZeroAccessRetryProof      `json:"zero_access_retry_proofs"`
	ExecutionReceipts  []StageExecutionReceipt     `json:"execution_receipts"`
	CompletionManifest *StageCompletionManifest    `json:"completion_manifest,omitempty"`
	FinalStageSeal     string                      `json:"final_stage_seal,omitempty"`
	IssuedAt           time.Time                   `json:"issued_at"`
	SealedAt           *time.Time                  `json:"sealed_at,omitempty"`
	RecordHash         string                      `json:"record_hash"`
}

type NomineeSelection struct {
	SchemaVersion       string    `json:"schema_version"`
	DevelopmentSetID    string    `json:"development_set_id"`
	Rule                string    `json:"rule"`
	Exists              bool      `json:"exists"`
	VariantID           string    `json:"variant_id,omitempty"`
	ConfigurationSHA256 string    `json:"configuration_sha256,omitempty"`
	SelectedAt          time.Time `json:"selected_at"`
	RecordHash          string    `json:"record_hash"`
}

type StageExecutionEnvelope struct {
	SchemaVersion string                     `json:"schema_version"`
	Snapshot      Snapshot                   `json:"snapshot"`
	ExecutionSet  StageExecutionSet          `json:"stage_execution_set"`
	Authorization *StageVariantAuthorization `json:"variant_authorization,omitempty"`
	EnvelopeHash  string                     `json:"envelope_hash"`
}

type Envelope struct {
	SchemaVersion string               `json:"schema_version"`
	Snapshot      Snapshot             `json:"snapshot"`
	Authorization *AuthorizationRecord `json:"partition_authorization,omitempty"`
	EnvelopeHash  string               `json:"envelope_hash"`
}
