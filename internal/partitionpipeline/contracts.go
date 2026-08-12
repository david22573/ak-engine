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
	PreparedPlanSchemaVersion       = "ak.engine.partition_artifact_plan.v2"
	ChildArtifactSchemaVersion      = "ak.engine.boundary_child_artifact.v1"
	PreparationManifestVersion      = "ak.engine.boundary_preparation_manifest.v1"
	BoundaryAuditSchemaVersion      = "ak.engine.membership_boundary_audit.v1"
	BoundaryTransformationVersion   = "ak.engine.fragment-boundary-slice.v1"
	RegistrySchemaVersion           = "ak.engine.partition_plan_registry.v1"
	AuthorizationSchemaVersion      = "ak.engine.partition_materialization_authorization.v1"
	AccessReceiptSchemaVersion      = "ak.engine.partition_access_receipt.v2"
	ConsumptionAuthorizationVersion = "ak.engine.partition_consumption_authorization.v1"
	ConsumptionReceiptVersion       = "ak.engine.partition_consumption_receipt.v1"
	ArtifactManifestVersion         = "ak.engine.partition_artifact_manifest.v2"
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
	PreparedFragmentEncoding        = "BOUNDARY_CHILD_GZIP_CANONICAL_JSON_V1"
	PreparedSourceRootID            = "BOUNDARY_CHILD_STORE"
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
	SourceRootID           string     `json:"source_root_id"`
	RelativePath           string     `json:"relative_path"`
	CanonicalSHA256        string     `json:"canonical_sha256"`
	Encoding               string     `json:"encoding,omitempty"`
	ReceiptSHA256          string     `json:"receipt_sha256,omitempty"`
	ObservedAvailableAtUTC time.Time  `json:"observed_available_at_utc,omitempty"`
	StoredFileSHA256       string     `json:"stored_file_sha256,omitempty"`
	ParentSourceRootID     string     `json:"parent_source_root_id,omitempty"`
	ParentReceiptSHA256    string     `json:"parent_receipt_sha256,omitempty"`
	ParentFragmentSHA256   string     `json:"parent_fragment_sha256,omitempty"`
	ParentProvenanceSHA256 string     `json:"parent_provenance_sha256,omitempty"`
	AuthorizedStartUTC     *time.Time `json:"authorized_start_utc,omitempty"`
	AuthorizedEndUTC       *time.Time `json:"authorized_end_utc,omitempty"`
	ChildRowCount          int        `json:"child_row_count,omitempty"`
	ChildFirstTimestampUTC *time.Time `json:"child_first_timestamp_utc,omitempty"`
	ChildLastTimestampUTC  *time.Time `json:"child_last_timestamp_utc,omitempty"`
	TransformationVersion  string     `json:"transformation_schema_version,omitempty"`
	BoundaryClass          string     `json:"boundary_class,omitempty"`
}

type SourceManifest struct {
	Symbol             string           `json:"symbol"`
	UTCDate            string           `json:"utc_date"`
	RelativePath       string           `json:"relative_path"`
	FileSHA256         string           `json:"manifest_file_sha256"`
	PartitionSHA256    string           `json:"partition_sha256"`
	ExpectedRows       int              `json:"expected_rows"`
	ReceiptArtifacts   []SourceArtifact `json:"receipt_artifacts"`
	FragmentArtifacts  []SourceArtifact `json:"fragment_artifacts"`
	MembershipInterval *Interval        `json:"membership_interval,omitempty"`
}

type Plan struct {
	SchemaVersion              string           `json:"schema_version"`
	Checkpoint                 HashIdentity     `json:"checkpoint"`
	HistorianCommit            string           `json:"historian_commit"`
	HistorianTree              string           `json:"historian_tree"`
	SourceIdentitySHA256       string           `json:"source_identity_sha256"`
	ReacquisitionProtocol      HashIdentity     `json:"reacquisition_protocol"`
	PreAcquisitionSealSHA256   string           `json:"pre_acquisition_seal_sha256"`
	SealedBinarySHA256         string           `json:"sealed_acquisition_binary_sha256"`
	AbandonedEvidenceRegistry  HashIdentity     `json:"abandoned_evidence_registry"`
	DatasetRequiredSymbols     []string         `json:"dataset_required_symbols"`
	CandidateTargetSymbols     []string         `json:"candidate_target_symbols"`
	ContextOnlySymbols         []string         `json:"context_only_symbols"`
	UniverseContractSHA256     string           `json:"universe_contract_sha256"`
	EligibleInterval           Interval         `json:"eligible_interval"`
	PartitionName              string           `json:"partition_name"`
	PartitionInterval          Interval         `json:"partition_interval"`
	SourceManifests            []SourceManifest `json:"ordered_source_manifest_membership"`
	ExpectedStructuralDays     int              `json:"expected_structural_utc_day_count"`
	SchemaIdentitySHA256       string           `json:"source_schema_identity_sha256"`
	OutputFormat               string           `json:"deterministic_output_format"`
	OrderingPolicy             string           `json:"deterministic_ordering"`
	OutputPathPolicy           string           `json:"output_path_policy"`
	SymlinkPolicy              string           `json:"symlink_policy"`
	CachePolicy                string           `json:"cache_policy"`
	AvailabilityCutoff         time.Time        `json:"availability_cutoff"`
	SourceRoot                 string           `json:"canonical_source_root"`
	ProspectiveSourceRoot      string           `json:"canonical_prospective_source_root,omitempty"`
	PreparedSourceRoot         string           `json:"canonical_prepared_source_root,omitempty"`
	ParentPlanSHA256           string           `json:"parent_plan_sha256,omitempty"`
	ParentSourceIdentitySHA256 string           `json:"parent_source_identity_sha256,omitempty"`
	PreparedPartitionIdentity  string           `json:"prepared_partition_source_identity_sha256,omitempty"`
	PreparationManifest        *HashIdentity    `json:"preparation_manifest,omitempty"`
	SyntheticFixture           bool             `json:"synthetic_fixture"`
	PlanSHA256                 string           `json:"canonical_plan_sha256"`
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
	SchemaVersion             string    `json:"schema_version"`
	PlanSHA256                string    `json:"plan_sha256"`
	CheckpointSHA256          string    `json:"checkpoint_sha256"`
	Partition                 string    `json:"partition"`
	RIFAuthorizationID        string    `json:"rif_authorization_id"`
	RIFAccessReceiptSHA256    string    `json:"rif_access_receipt_sha256"`
	SourceManifestCount       int       `json:"source_manifest_count"`
	SourceFragmentCount       int       `json:"source_fragment_count"`
	ParentFragmentCount       int       `json:"parent_fragment_count"`
	PreparationManifestSHA256 string    `json:"preparation_manifest_sha256"`
	ParentProvenanceSHA256    string    `json:"parent_provenance_sha256"`
	RowsOpened                int       `json:"rows_opened"`
	CandidateInputRows        int       `json:"candidate_input_rows"`
	OpenedAt                  time.Time `json:"opened_at"`
	ArtifactSHA256            string    `json:"artifact_sha256"`
	ArtifactManifestSHA256    string    `json:"artifact_manifest_sha256"`
	PreviousReceiptSHA256     string    `json:"previous_receipt_sha256"`
	ReceiptSHA256             string    `json:"receipt_sha256"`
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
	SchemaVersion                 string   `json:"schema_version"`
	PlanSHA256                    string   `json:"plan_sha256"`
	CheckpointSHA256              string   `json:"checkpoint_sha256"`
	Partition                     string   `json:"partition"`
	UniverseContractSHA256        string   `json:"universe_contract_sha256"`
	OrderedSourceSHA256           []string `json:"ordered_source_manifest_sha256"`
	OrderedChildSHA256            []string `json:"ordered_child_artifact_sha256"`
	OrderedParentProvenanceSHA256 []string `json:"ordered_parent_provenance_sha256"`
	PreparationManifestSHA256     string   `json:"preparation_manifest_sha256"`
	ArtifactSHA256                string   `json:"artifact_sha256"`
	ManifestSHA256                string   `json:"manifest_sha256"`
}

type ParentProvenance struct {
	SourceRootID           string    `json:"source_root_id"`
	ReceiptSHA256          string    `json:"receipt_sha256"`
	FragmentSHA256         string    `json:"fragment_sha256"`
	ParentObjectID         string    `json:"parent_object_id"`
	ParentStartUTC         time.Time `json:"parent_start_utc"`
	ParentEndUTC           time.Time `json:"parent_end_utc"`
	ParentRowCount         int       `json:"parent_row_count"`
	ObservedAvailableAtUTC time.Time `json:"observed_available_at_utc"`
	ProvenanceSHA256       string    `json:"provenance_sha256"`
}

type ChildArtifact struct {
	SchemaVersion         string             `json:"schema_version"`
	Symbol                string             `json:"symbol"`
	AuthorizedInterval    Interval           `json:"authorized_interval"`
	Parent                ParentProvenance   `json:"parent_provenance"`
	TransformationVersion string             `json:"transformation_schema_version"`
	Records               []normalizedRecord `json:"records"`
	ChildSHA256           string             `json:"child_sha256"`
}

type PreparationManifestEntry struct {
	Partition              string           `json:"partition"`
	Symbol                 string           `json:"symbol"`
	UTCDate                string           `json:"utc_date"`
	MembershipInterval     Interval         `json:"membership_interval"`
	ParentManifestSHA256   string           `json:"parent_manifest_sha256"`
	ParentPartitionSHA256  string           `json:"parent_partition_sha256"`
	ParentExpectedRows     int              `json:"parent_expected_rows"`
	ChildRelativePath      string           `json:"child_relative_path"`
	ChildSHA256            string           `json:"child_sha256"`
	StoredFileSHA256       string           `json:"stored_file_sha256"`
	ChildRowCount          int              `json:"child_row_count"`
	ChildFirstTimestampUTC time.Time        `json:"child_first_timestamp_utc"`
	ChildLastTimestampUTC  time.Time        `json:"child_last_timestamp_utc"`
	BoundaryClass          string           `json:"boundary_class"`
	Parent                 ParentProvenance `json:"parent_provenance"`
}

type PreparationManifest struct {
	SchemaVersion                string                     `json:"schema_version"`
	ParentPlanSHA256             string                     `json:"parent_plan_sha256"`
	Partition                    string                     `json:"partition"`
	CheckpointSHA256             string                     `json:"checkpoint_sha256"`
	ParentSourceIdentitySHA256   string                     `json:"parent_source_identity_sha256"`
	PreparedSourceIdentitySHA256 string                     `json:"prepared_source_identity_sha256"`
	TransformationVersion        string                     `json:"transformation_schema_version"`
	Entries                      []PreparationManifestEntry `json:"entries"`
	ManifestSHA256               string                     `json:"manifest_sha256"`
}

type BoundaryClassCounts struct {
	Exact int `json:"exact_boundary_artifacts"`
	Left  int `json:"left_clipped_artifacts"`
	Right int `json:"right_clipped_artifacts"`
	Both  int `json:"both_boundary_artifacts"`
}

type BoundarySymbolSummary struct {
	Symbol      string              `json:"symbol"`
	Memberships int                 `json:"memberships"`
	Artifacts   int                 `json:"artifacts"`
	Classes     BoundaryClassCounts `json:"classes"`
	Rejected    int                 `json:"rejected_artifacts"`
	Missing     int                 `json:"missing_artifacts"`
}

type BoundaryAudit struct {
	SchemaVersion             string                  `json:"schema_version"`
	Partition                 string                  `json:"partition"`
	ParentPlanSHA256          string                  `json:"parent_plan_sha256"`
	PreparedPlanSHA256        string                  `json:"prepared_plan_sha256"`
	PreparationManifestSHA256 string                  `json:"preparation_manifest_sha256"`
	Memberships               int                     `json:"memberships"`
	Artifacts                 int                     `json:"artifacts"`
	Classes                   BoundaryClassCounts     `json:"classes"`
	Rejected                  int                     `json:"rejected_artifacts"`
	Missing                   int                     `json:"missing_artifacts"`
	UnsafeMemberships         int                     `json:"unsafe_memberships"`
	Symbols                   []BoundarySymbolSummary `json:"symbols"`
	AuditSHA256               string                  `json:"audit_sha256"`
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
