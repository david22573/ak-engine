package qualificationrunner

import (
	"time"

	"github.com/david22573/ak-engine/internal/preconditions"
	"github.com/david22573/ak-engine/internal/qualification"
	"github.com/david22573/ak-engine/internal/rifbridge"
)

const (
	RequestSchemaVersion    = "ak.engine.qualification_execution_request.v1"
	ReadinessSchemaVersion  = "ak.engine.qualification_structural_readiness.v1"
	ResultSchemaVersion     = "ak.engine.qualification_execution_result.v1"
	DatasetArtifactVersion  = "ak.engine.synthetic_partition_fixture.v1"
	ConfigurationVersion    = "ak.engine.downtrend-midvol-relief.configuration.v1"
	VariantLedgerVersion    = "ak.engine.variant_ledger.v1"
	V00SourceSHA256         = "sha256:3c2e20fd5bf615864aebc5be35ce86c15a6ed8f83de33b2f1d33b00dae6fbfa1"
	V00CandidateFamily      = "phase12/DowntrendMidVolReliefLong240m"
	NoOutcomesLabel         = "NO_CANDIDATE_OUTCOMES_PRODUCED"
	SyntheticLabel          = "SYNTHETIC_NON_RESEARCH_EVIDENCE"
	RegisteredResearchLabel = "REGISTERED_RESEARCH_PARTITION"
)

type Mode string

const (
	ModeVerify       Mode = "verify"
	ModeDevelopment  Mode = "development"
	ModeValidation   Mode = "validation"
	ModeFinalHoldout Mode = "final-holdout"
)

type HashIdentity struct {
	ID     string `json:"id"`
	SHA256 string `json:"sha256"`
}
type Interval struct {
	Start time.Time `json:"start_inclusive"`
	End   time.Time `json:"end_exclusive"`
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
type DatasetIdentity struct {
	Checkpoint                HashIdentity `json:"checkpoint"`
	SourceIdentitySHA256      string       `json:"source_identity_sha256"`
	ReacquisitionProtocol     HashIdentity `json:"reacquisition_protocol"`
	PreAcquisitionSealSHA256  string       `json:"pre_acquisition_seal_sha256"`
	SealedBinarySHA256        string       `json:"sealed_binary_sha256"`
	AbandonedEvidenceRegistry HashIdentity `json:"abandoned_evidence_registry"`
	HistorianCheckpointCommit string       `json:"historian_checkpoint_commit"`
	RequiredSymbols           []string     `json:"required_symbols"`
	EligibleInterval          Interval     `json:"eligible_interval"`
	ProhibitedPriorExposure   []Interval   `json:"prohibited_prior_exposure_intervals"`
	AvailabilityCutoff        time.Time    `json:"availability_cutoff"`
}
type Partition struct {
	Name                         string   `json:"name"`
	Interval                     Interval `json:"interval"`
	StructuralDayCount           int      `json:"structural_day_count"`
	RequiredSymbolCoverageSHA256 string   `json:"required_symbol_coverage_sha256"`
}
type IdentityVariant struct {
	ID                  string   `json:"id"`
	ConfigurationSHA256 string   `json:"canonical_configuration_sha256"`
	Dimensions          []string `json:"configuration_dimensions"`
}
type StabilityNeighborhood struct {
	VariantID   string   `json:"variant_id"`
	NeighborIDs []string `json:"neighbor_ids"`
}
type IdentityVariantLedger struct {
	Variants                  []IdentityVariant       `json:"variants"`
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
type ResearchIdentityV4 struct {
	SchemaVersion  string                `json:"schema_version"`
	ResearchID     string                `json:"research_id"`
	Repositories   RepositoryIdentity    `json:"repository_identity"`
	Protocol       ProtocolIdentity      `json:"protocol_identity"`
	CandidateScope CandidateScope        `json:"candidate_scope"`
	Dataset        DatasetIdentity       `json:"dataset_identity"`
	Partitions     []Partition           `json:"partitions"`
	VariantLedger  IdentityVariantLedger `json:"variant_governance"`
	Authorities    AuthorityIdentity     `json:"authority_identity"`
	AccessPolicy   AccessPolicy          `json:"access_and_lifecycle_policy"`
}

type CandidateConfiguration struct {
	SchemaVersion         string   `json:"schema_version"`
	CandidateFamily       string   `json:"candidate_family"`
	Side                  string   `json:"side"`
	Horizon               string   `json:"horizon"`
	TrendState            string   `json:"trend_state"`
	RealizedVol60Minimum  float64  `json:"realized_vol_60_minimum"`
	RealizedVol60Maximum  float64  `json:"realized_vol_60_maximum"`
	ContextAgreement      string   `json:"context_agreement"`
	EventQuality          string   `json:"event_quality"`
	CooldownMinutes       int      `json:"cooldown_independence_minutes"`
	Symbols               []string `json:"symbols"`
	DateExclusions        []string `json:"date_exclusions"`
	QuarterExclusions     []string `json:"quarter_exclusions"`
	TransactionCostBPS    float64  `json:"transaction_cost_bps"`
	SizingPolicy          string   `json:"sizing_policy"`
	OutcomeDerivedFilters []string `json:"outcome_derived_filters"`
	Indicators            []string `json:"indicators"`
	Features              []string `json:"features"`
}

type RegisteredVariant struct {
	ID                  string                 `json:"id"`
	Dimensions          []string               `json:"dimensions"`
	Configuration       CandidateConfiguration `json:"configuration"`
	ConfigurationSHA256 string                 `json:"configuration_sha256"`
}
type VariantLedger struct {
	SchemaVersion          string                  `json:"schema_version"`
	MaximumVariants        int                     `json:"maximum_variants"`
	V00ID                  string                  `json:"v00_id"`
	Variants               []RegisteredVariant     `json:"variants"`
	StabilityNeighborhoods []StabilityNeighborhood `json:"stability_neighborhoods"`
	LedgerSHA256           string                  `json:"ledger_sha256"`
}

type DatasetBinding struct {
	Checkpoint              HashIdentity `json:"checkpoint"`
	SourceIdentitySHA256    string       `json:"source_identity_sha256"`
	SealedBinarySHA256      string       `json:"sealed_binary_sha256"`
	RequiredSymbols         []string     `json:"required_symbols"`
	EligibleInterval        Interval     `json:"eligible_interval"`
	ProhibitedPriorExposure []Interval   `json:"prohibited_prior_exposure_intervals"`
	AvailabilityCutoff      time.Time    `json:"availability_cutoff"`
}
type RunnerIdentity struct {
	GitCommit        string `json:"git_commit"`
	ExecutableSHA256 string `json:"executable_sha256"`
	V00SourceSHA256  string `json:"v00_source_sha256"`
}

type ExecutionRequest struct {
	SchemaVersion        string                               `json:"schema_version"`
	Mode                 Mode                                 `json:"mode"`
	RIF                  rifbridge.ResearchGovernanceEnvelope `json:"rif_governance"`
	Protocol             ProtocolIdentity                     `json:"protocol_identity"`
	VariantLedger        VariantLedger                        `json:"variant_ledger"`
	VariantID            string                               `json:"registered_variant_id"`
	ConfigurationSHA256  string                               `json:"registered_configuration_sha256"`
	Dataset              DatasetBinding                       `json:"dataset_identity"`
	Partition            Partition                            `json:"partition"`
	CandidateFamily      string                               `json:"candidate_family_identity"`
	Independence         HashIdentity                         `json:"independence_authority"`
	Uncertainty          HashIdentity                         `json:"uncertainty_authority"`
	Concentration        HashIdentity                         `json:"concentration_authority"`
	QualificationGateSet HashIdentity                         `json:"qualification_gate_set"`
	CostPolicy           HashIdentity                         `json:"transaction_cost_policy"`
	SeedPolicy           HashIdentity                         `json:"deterministic_seed_policy"`
	Runner               RunnerIdentity                       `json:"runner_identity"`
}

type ReadinessArtifact struct {
	SchemaVersion          string   `json:"schema_version"`
	Label                  string   `json:"label"`
	Mode                   Mode     `json:"mode"`
	ResearchIdentitySHA256 string   `json:"research_identity_sha256"`
	ReservationID          string   `json:"reservation_id"`
	VariantID              string   `json:"variant_id"`
	ConfigurationSHA256    string   `json:"configuration_sha256"`
	GateSetSHA256          string   `json:"gate_set_sha256"`
	AuthorityHashes        []string `json:"authority_hashes"`
	RunnerExecutableSHA256 string   `json:"runner_executable_sha256"`
	Partition              string   `json:"partition"`
	DataLoads              int      `json:"data_loads"`
	CandidateEvents        int      `json:"candidate_events"`
	CandidateOutcomes      int      `json:"candidate_outcomes"`
	ArtifactSHA256         string   `json:"artifact_sha256"`
}

type Context struct {
	SnapshotID        string    `json:"snapshot_id"`
	SourceInputSHA256 string    `json:"source_input_sha256"`
	AvailableAt       time.Time `json:"available_at"`
	Return60          float64   `json:"return_60"`
}
type InputRow struct {
	Partition        string    `json:"partition"`
	Symbol           string    `json:"symbol"`
	EventTime        time.Time `json:"event_time"`
	AvailableAt      time.Time `json:"available_at"`
	Close            float64   `json:"close"`
	FutureClose240m  float64   `json:"future_close_240m"`
	EMA50            float64   `json:"ema_50"`
	EMA200           float64   `json:"ema_200"`
	TrendSlope20     float64   `json:"trend_slope_20"`
	RealizedVol60    float64   `json:"realized_vol_60"`
	WarmupSufficient bool      `json:"warmup_sufficient"`
	BTC              Context   `json:"btc_context"`
	ETH              Context   `json:"eth_context"`
}
type PartitionArtifact struct {
	SchemaVersion        string     `json:"schema_version"`
	Label                string     `json:"label"`
	CheckpointSHA256     string     `json:"checkpoint_sha256"`
	SourceIdentitySHA256 string     `json:"source_identity_sha256"`
	SealedBinarySHA256   string     `json:"sealed_binary_sha256"`
	Partition            string     `json:"partition"`
	Symbols              []string   `json:"symbols"`
	Rows                 []InputRow `json:"rows"`
	ArtifactSHA256       string     `json:"artifact_sha256"`
}

type GateMetrics struct {
	EventCount              int     `json:"event_count"`
	IndependentClusterCount int     `json:"independent_cluster_count"`
	TradesOrDecisions       int     `json:"trades_or_decisions"`
	SymbolsRepresented      int     `json:"symbols_represented"`
	MonthsRepresented       int     `json:"months_represented"`
	PositiveRegimes         int     `json:"positive_regimes"`
	NegativeRegimes         int     `json:"negative_regimes"`
	NetExpectancyBPS        float64 `json:"net_expectancy_bps"`
	ProfitFactor            float64 `json:"profit_factor"`
	MaximumDrawdownBPS      float64 `json:"maximum_drawdown_bps"`
	WorstPeriodProfitFactor float64 `json:"worst_period_profit_factor"`
	StableNeighbors         int     `json:"stable_neighbors"`
	StressProfitFactor      float64 `json:"stress_profit_factor"`
	StressExpectancyBPS     float64 `json:"stress_expectancy_bps"`
}
type GateDecision struct {
	Passed        bool     `json:"passed"`
	FailedGateIDs []string `json:"failed_gate_ids"`
}
type ExecutedAuthority struct {
	ID             string `json:"id"`
	SHA256         string `json:"sha256"`
	Invoked        bool   `json:"invoked"`
	EvidenceSHA256 string `json:"evidence_sha256"`
}
type ResultArtifact struct {
	SchemaVersion       string                                  `json:"schema_version"`
	Label               string                                  `json:"label"`
	Mode                Mode                                    `json:"mode"`
	VariantID           string                                  `json:"variant_id"`
	ConfigurationSHA256 string                                  `json:"configuration_sha256"`
	Partition           string                                  `json:"partition"`
	Metrics             GateMetrics                             `json:"metrics"`
	GateDecision        GateDecision                            `json:"gate_decision"`
	Independence        ExecutedAuthority                       `json:"independence_authority"`
	Uncertainty         ExecutedAuthority                       `json:"uncertainty_authority"`
	Concentration       ExecutedAuthority                       `json:"concentration_authority"`
	UncertaintyResult   preconditions.UncertaintyResultV2       `json:"uncertainty_result"`
	ConcentrationResult preconditions.ConcentrationEvaluationV3 `json:"concentration_result"`
	ResultSHA256        string                                  `json:"result_sha256"`
}

type VerifiedRequest struct {
	Request       ExecutionRequest
	Identity      ResearchIdentityV4
	Variant       RegisteredVariant
	Gates         qualification.PR4B0GateSet
	GateSetSHA256 string
}
