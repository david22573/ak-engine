package partitionpipeline

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"strings"
	"time"
)

const (
	PlanSchemaVersion               = "ak.engine.partition_artifact_plan.v1"
	RegistrySchemaVersion           = "ak.engine.partition_plan_registry.v1"
	AuthorizationSchemaVersion      = "ak.engine.partition_materialization_authorization.v1"
	AccessReceiptSchemaVersion      = "ak.engine.partition_access_receipt.v1"
	ConsumptionAuthorizationVersion = "ak.engine.partition_consumption_authorization.v1"
	ConsumptionReceiptVersion       = "ak.engine.partition_consumption_receipt.v1"
	ArtifactManifestVersion         = "ak.engine.partition_artifact_manifest.v1"
	ZeroAccessProofVersion          = "ak.engine.partition_zero_access_proof.v1"
	OutputFormat                    = "ak.engine.qualification_partition_artifact.canonical-json.v1"
	OrderingPolicy                  = "EVENT_TIME_ASCENDING_THEN_TARGET_SYMBOL_ASCENDING"
	OutputPathPolicy                = "REGISTRY_ROOT/artifacts/<checkpoint>/<partition>/<plan_sha256>.json"
	SymlinkPolicy                   = "REJECT_ALL_SOURCE_REGISTRY_CACHE_AND_OUTPUT_SYMLINKS"
	CachePolicy                     = "REGISTERED_REGISTRY_ROOT_ONLY_NO_CALLER_CACHE"
	CheckpointSourceRootID          = "R1P5R_CHECKPOINT_STORE"
	ProspectiveSourceRootID         = "P4_PROSPECTIVE_STORE"
	BackfillFragmentEncoding        = "R1P5R_GZIP_CANONICAL_JSON_V1"
	ProspectiveFragmentEncoding     = "P4_CANONICAL_JSON_V1"
	SyntheticFragmentEncoding       = "SYNTHETIC_GZIP_CANONICAL_JSON_V1"
)

type LifecycleState string

const (
	PlanCreated               LifecycleState = "PLAN_CREATED"
	PlanVerified              LifecycleState = "PLAN_VERIFIED"
	MaterializationAuthorized LifecycleState = "MATERIALIZATION_AUTHORIZED"
	MaterializationStarted    LifecycleState = "MATERIALIZATION_STARTED"
	MaterializationSealed     LifecycleState = "MATERIALIZATION_SEALED"
	ConsumptionAuthorized     LifecycleState = "CONSUMPTION_AUTHORIZED"
	ConsumptionSealed         LifecycleState = "CONSUMPTION_SEALED"
)

type HashIdentity struct {
	ID     string `json:"id"`
	SHA256 string `json:"sha256"`
}

type Interval struct {
	Start time.Time `json:"start_inclusive"`
	End   time.Time `json:"end_exclusive"`
}

type SourceArtifact struct {
	SourceRootID           string    `json:"source_root_id"`
	RelativePath           string    `json:"relative_path"`
	CanonicalSHA256        string    `json:"canonical_sha256"`
	Encoding               string    `json:"encoding,omitempty"`
	ReceiptSHA256          string    `json:"receipt_sha256,omitempty"`
	ObservedAvailableAtUTC time.Time `json:"observed_available_at_utc,omitempty"`
}

type SourceManifest struct {
	Symbol            string           `json:"symbol"`
	UTCDate           string           `json:"utc_date"`
	RelativePath      string           `json:"relative_path"`
	FileSHA256        string           `json:"manifest_file_sha256"`
	PartitionSHA256   string           `json:"partition_sha256"`
	ExpectedRows      int              `json:"expected_rows"`
	ReceiptArtifacts  []SourceArtifact `json:"receipt_artifacts"`
	FragmentArtifacts []SourceArtifact `json:"fragment_artifacts"`
}

type Plan struct {
	SchemaVersion             string           `json:"schema_version"`
	Checkpoint                HashIdentity     `json:"checkpoint"`
	HistorianCommit           string           `json:"historian_commit"`
	HistorianTree             string           `json:"historian_tree"`
	SourceIdentitySHA256      string           `json:"source_identity_sha256"`
	ReacquisitionProtocol     HashIdentity     `json:"reacquisition_protocol"`
	PreAcquisitionSealSHA256  string           `json:"pre_acquisition_seal_sha256"`
	SealedBinarySHA256        string           `json:"sealed_acquisition_binary_sha256"`
	AbandonedEvidenceRegistry HashIdentity     `json:"abandoned_evidence_registry"`
	DatasetRequiredSymbols    []string         `json:"dataset_required_symbols"`
	CandidateTargetSymbols    []string         `json:"candidate_target_symbols"`
	ContextOnlySymbols        []string         `json:"context_only_symbols"`
	UniverseContractSHA256    string           `json:"universe_contract_sha256"`
	EligibleInterval          Interval         `json:"eligible_interval"`
	PartitionName             string           `json:"partition_name"`
	PartitionInterval         Interval         `json:"partition_interval"`
	SourceManifests           []SourceManifest `json:"ordered_source_manifest_membership"`
	ExpectedStructuralDays    int              `json:"expected_structural_utc_day_count"`
	SchemaIdentitySHA256      string           `json:"source_schema_identity_sha256"`
	OutputFormat              string           `json:"deterministic_output_format"`
	OrderingPolicy            string           `json:"deterministic_ordering"`
	OutputPathPolicy          string           `json:"output_path_policy"`
	SymlinkPolicy             string           `json:"symlink_policy"`
	CachePolicy               string           `json:"cache_policy"`
	AvailabilityCutoff        time.Time        `json:"availability_cutoff"`
	SourceRoot                string           `json:"canonical_source_root"`
	ProspectiveSourceRoot     string           `json:"canonical_prospective_source_root,omitempty"`
	SyntheticFixture          bool             `json:"synthetic_fixture"`
	PlanSHA256                string           `json:"canonical_plan_sha256"`
}

type MaterializationAuthorization struct {
	SchemaVersion          string    `json:"schema_version"`
	PlanSHA256             string    `json:"plan_sha256"`
	CheckpointSHA256       string    `json:"checkpoint_sha256"`
	Partition              string    `json:"partition"`
	RIFAuthorizationID     string    `json:"rif_authorization_id"`
	RIFAuthorizationSHA256 string    `json:"rif_authorization_sha256"`
	RIFAccessReceiptSHA256 string    `json:"rif_access_receipt_sha256"`
	AuthorizedAt           time.Time `json:"authorized_at"`
	AuthorizationSHA256    string    `json:"authorization_sha256"`
}

type AccessReceipt struct {
	SchemaVersion          string    `json:"schema_version"`
	PlanSHA256             string    `json:"plan_sha256"`
	CheckpointSHA256       string    `json:"checkpoint_sha256"`
	Partition              string    `json:"partition"`
	RIFAuthorizationID     string    `json:"rif_authorization_id"`
	RIFAccessReceiptSHA256 string    `json:"rif_access_receipt_sha256"`
	SourceManifestCount    int       `json:"source_manifest_count"`
	SourceFragmentCount    int       `json:"source_fragment_count"`
	RowsOpened             int       `json:"rows_opened"`
	CandidateInputRows     int       `json:"candidate_input_rows"`
	OpenedAt               time.Time `json:"opened_at"`
	ArtifactSHA256         string    `json:"artifact_sha256"`
	ArtifactManifestSHA256 string    `json:"artifact_manifest_sha256"`
	PreviousReceiptSHA256  string    `json:"previous_receipt_sha256"`
	ReceiptSHA256          string    `json:"receipt_sha256"`
}

type ZeroAccessProof struct {
	SchemaVersion    string         `json:"schema_version"`
	PlanSHA256       string         `json:"plan_sha256"`
	RegistrySHA256   string         `json:"registry_sha256"`
	LifecycleState   LifecycleState `json:"lifecycle_state"`
	RowsOpened       int            `json:"rows_opened"`
	OutcomeArtifacts int            `json:"outcome_artifacts"`
	ResultArtifacts  int            `json:"result_artifacts"`
	ProofSHA256      string         `json:"proof_sha256"`
}

type ArtifactManifest struct {
	SchemaVersion          string   `json:"schema_version"`
	PlanSHA256             string   `json:"plan_sha256"`
	CheckpointSHA256       string   `json:"checkpoint_sha256"`
	Partition              string   `json:"partition"`
	UniverseContractSHA256 string   `json:"universe_contract_sha256"`
	OrderedSourceSHA256    []string `json:"ordered_source_manifest_sha256"`
	ArtifactSHA256         string   `json:"artifact_sha256"`
	ManifestSHA256         string   `json:"manifest_sha256"`
}

type RegistryEntry struct {
	PlanSHA256               string                        `json:"plan_sha256"`
	State                    LifecycleState                `json:"lifecycle_state"`
	Authorization            *MaterializationAuthorization `json:"materialization_authorization,omitempty"`
	ArtifactSHA256           string                        `json:"artifact_sha256,omitempty"`
	ArtifactManifestSHA256   string                        `json:"artifact_manifest_sha256,omitempty"`
	AccessReceiptSHA256      string                        `json:"access_receipt_sha256,omitempty"`
	ConsumptionAuthorization *ConsumptionAuthorization     `json:"consumption_authorization,omitempty"`
	ConsumptionReceipts      []ConsumptionReceipt          `json:"consumption_receipts,omitempty"`
}

type ConsumptionAuthorization struct {
	SchemaVersion          string    `json:"schema_version"`
	PlanSHA256             string    `json:"plan_sha256"`
	ArtifactSHA256         string    `json:"artifact_sha256"`
	Partition              string    `json:"partition"`
	VariantID              string    `json:"variant_id"`
	RIFAuthorizationID     string    `json:"rif_authorization_id"`
	RIFAccessReceiptSHA256 string    `json:"rif_access_receipt_sha256"`
	AuthorizedAt           time.Time `json:"authorized_at"`
	AuthorizationSHA256    string    `json:"authorization_sha256"`
}

type ConsumptionReceipt struct {
	SchemaVersion          string    `json:"schema_version"`
	PlanSHA256             string    `json:"plan_sha256"`
	ArtifactSHA256         string    `json:"artifact_sha256"`
	Partition              string    `json:"partition"`
	VariantID              string    `json:"variant_id"`
	RIFAuthorizationID     string    `json:"rif_authorization_id"`
	RIFAccessReceiptSHA256 string    `json:"rif_access_receipt_sha256"`
	ConsumedAt             time.Time `json:"consumed_at"`
	PreviousReceiptSHA256  string    `json:"previous_receipt_sha256"`
	ReceiptSHA256          string    `json:"receipt_sha256"`
}

type Registry struct {
	SchemaVersion  string                   `json:"schema_version"`
	Entries        map[string]RegistryEntry `json:"plans"`
	RegistrySHA256 string                   `json:"registry_sha256"`
}

func canonicalHash(value any) (string, error) {
	data, err := json.Marshal(value)
	if err != nil {
		return "", err
	}
	digest := sha256.Sum256(data)
	return "sha256:" + hex.EncodeToString(digest[:]), nil
}

func byteHash(data []byte) string {
	digest := sha256.Sum256(data)
	return "sha256:" + hex.EncodeToString(digest[:])
}

func SealMaterializationAuthorization(value MaterializationAuthorization) (MaterializationAuthorization, error) {
	value.SchemaVersion = AuthorizationSchemaVersion
	value.AuthorizationSHA256 = ""
	hash, err := canonicalHash(value)
	if err != nil {
		return MaterializationAuthorization{}, err
	}
	value.AuthorizationSHA256 = hash
	return value, nil
}

func SealConsumptionAuthorization(value ConsumptionAuthorization) (ConsumptionAuthorization, error) {
	value.SchemaVersion = ConsumptionAuthorizationVersion
	value.AuthorizationSHA256 = ""
	hash, err := canonicalHash(value)
	if err != nil {
		return ConsumptionAuthorization{}, err
	}
	value.AuthorizationSHA256 = hash
	return value, nil
}

func validSHA(value string) bool {
	if len(value) != 71 || !strings.HasPrefix(value, "sha256:") || value != strings.ToLower(value) {
		return false
	}
	_, err := hex.DecodeString(strings.TrimPrefix(value, "sha256:"))
	return err == nil
}

func intervalValid(value Interval) bool {
	return !value.Start.IsZero() && value.Start.Location() == time.UTC && !value.End.IsZero() && value.End.Location() == time.UTC && value.Start.Before(value.End)
}

var errUnsafePath = errors.New("unsafe, alternate, or symlinked path rejected")
