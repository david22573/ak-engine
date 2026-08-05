package researchidentity

import (
	"fmt"
	"time"

	"github.com/david22573/ak-engine/internal/canonicalcontract"
	"github.com/david22573/ak-engine/internal/features"
	"github.com/david22573/ak-engine/internal/regime"
	"github.com/david22573/ak-engine/pkg/protocol"
)

type IdentityStatus string

const (
	StatusComplete             IdentityStatus = "COMPLETE_RESEARCH_IDENTITY"
	StatusCandidateIncomplete  IdentityStatus = "INCOMPLETE_CANDIDATE_IDENTITY"
	StatusConfigurationMissing IdentityStatus = "INCOMPLETE_CONFIGURATION_IDENTITY"
	StatusDirtyEngineSource    IdentityStatus = "DIRTY_ENGINE_SOURCE"
	StatusDatasetIncomplete    IdentityStatus = "INCOMPLETE_DATASET_IDENTITY"
	StatusPITIncomplete        IdentityStatus = "INCOMPLETE_PIT_IDENTITY"
	StatusFeatureIncomplete    IdentityStatus = "INCOMPLETE_FEATURE_IDENTITY"
	StatusRegimeIncomplete     IdentityStatus = "INCOMPLETE_REGIME_IDENTITY"
	StatusConsumedIncomplete   IdentityStatus = "INCOMPLETE_CONSUMED_INPUT_IDENTITY"
	StatusSeriesIncomplete     IdentityStatus = "INCOMPLETE_SERIES_IDENTITY"
	StatusConflict             IdentityStatus = "IDENTITY_CONFLICT"
	StatusValidationFailed     IdentityStatus = "IDENTITY_VALIDATION_FAILED"
)

type Finding struct {
	Code     string         `json:"code"`
	Domain   string         `json:"domain"`
	Reason   string         `json:"reason"`
	Status   IdentityStatus `json:"status"`
	Blocking bool           `json:"blocking"`
}

type DerivationError struct {
	Status IdentityStatus
	Code   string
	Err    error
}

func (e *DerivationError) Error() string {
	if e == nil {
		return ""
	}
	return fmt.Sprintf("%s: %s: %v", e.Status, e.Code, e.Err)
}

func (e *DerivationError) Unwrap() error { return e.Err }

type FileInventoryEntry struct {
	Path            string `json:"path"`
	SizeBytes       int64  `json:"size_bytes"`
	SHA256          string `json:"object_hash"`
	ModeClass       string `json:"mode_class"`
	InclusionReason string `json:"inclusion_reason"`
}

type CandidateImplementationIdentity struct {
	Contract           canonicalcontract.ContractHeader `json:"contract"`
	ArtifactHash       string                           `json:"artifact_hash"`
	Files              []FileInventoryEntry             `json:"files"`
	InventoryEncoding  string                           `json:"-"`
	ImplementationHash string                           `json:"-"`
}

type CandidateIdentity struct {
	Contract               canonicalcontract.ContractHeader `json:"contract"`
	ArtifactHash           string                           `json:"artifact_hash"`
	CandidateID            string                           `json:"candidate_id"`
	RegistryID             string                           `json:"registry_id"`
	RegistryVersion        string                           `json:"registry_version"`
	RegistrySchemaVersion  int                              `json:"registry_schema_version"`
	CandidateVersion       string                           `json:"candidate_version"`
	CandidateType          string                           `json:"candidate_type"`
	Family                 string                           `json:"family"`
	Side                   string                           `json:"side"`
	Aliases                []string                         `json:"aliases"`
	ImplementationLocator  string                           `json:"implementation_locator"`
	RegistrationRecordHash string                           `json:"-"`
	UsesRegimes            bool                             `json:"uses_regimes"`
	Implementation         CandidateImplementationIdentity  `json:"implementation"`
}

type BracketConfiguration struct {
	Name  string  `json:"name"`
	TPBPS float64 `json:"tp_bps"`
	SLBPS float64 `json:"sl_bps"`
}

type GateThresholds struct {
	MinimumEvents                  int     `json:"minimum_events"`
	MinimumH2PF                    float64 `json:"minimum_h2_pf"`
	MinimumH2ExpectancyBPS         float64 `json:"minimum_h2_expectancy_bps"`
	MinimumFYPF                    float64 `json:"minimum_fy_pf"`
	MinimumFYExpectancyBPS         float64 `json:"minimum_fy_expectancy_bps"`
	MinimumPositiveMonths          int     `json:"minimum_positive_months"`
	MinimumDelayOneExpectancyBPS   float64 `json:"minimum_delay_one_expectancy_bps"`
	MaximumSingleMonthContribution float64 `json:"maximum_single_month_contribution_pct"`
	MinimumWorstQuarterPF          float64 `json:"minimum_worst_quarter_pf"`
	MinimumBracketPF               float64 `json:"minimum_bracket_pf"`
	MinimumBracketExpectancyBPS    float64 `json:"minimum_bracket_expectancy_bps"`
}

// ResolvedResearchConfiguration is the exact effective configuration used by
// deep evaluation and converted to the canonical contract at the boundary.
type ResolvedResearchConfiguration struct {
	EncodingVersion           string                 `json:"encoding_version"`
	Symbol                    string                 `json:"symbol"`
	Market                    string                 `json:"market"`
	Interval                  string                 `json:"interval"`
	EvaluationStartMS         int64                  `json:"evaluation_start_ms"`
	EvaluationEndMS           int64                  `json:"evaluation_end_ms"`
	ForwardHorizonsMinutes    []int                  `json:"forward_horizons_minutes"`
	CostHaircutsBPS           []float64              `json:"cost_haircuts_bps"`
	CostHaircutHorizonMinutes int                    `json:"cost_haircut_horizon_minutes"`
	EntryDelayCandles         []int                  `json:"entry_delay_candles"`
	EntryDelayHorizonMinutes  int                    `json:"entry_delay_horizon_minutes"`
	EntryDelayCostBPS         float64                `json:"entry_delay_cost_bps"`
	SeriesHorizonMinutes      int                    `json:"series_horizon_minutes"`
	SeriesCostBPS             float64                `json:"series_cost_bps"`
	SeriesEntryDelayCandles   int                    `json:"series_entry_delay_candles"`
	ExcursionWindowsMinutes   []int                  `json:"excursion_windows_minutes"`
	Brackets                  []BracketConfiguration `json:"brackets"`
	BracketWindowMinutes      int                    `json:"bracket_window_minutes"`
	BracketCostBPS            float64                `json:"bracket_cost_bps"`
	StabilityAggregatePeriods []string               `json:"stability_aggregate_periods"`
	StabilityHorizonMinutes   int                    `json:"stability_horizon_minutes"`
	StabilityCostBPS          float64                `json:"stability_cost_bps"`
	StabilityEntryDelay       int                    `json:"stability_entry_delay_candles"`
	RegimeGroups              []string               `json:"regime_groups"`
	RegimeHorizonMinutes      int                    `json:"regime_horizon_minutes"`
	RegimeCostBPS             float64                `json:"regime_cost_bps"`
	RegimeLowSampleMinimum    int                    `json:"regime_low_sample_minimum"`
	ClusterSeparationMinutes  int                    `json:"cluster_separation_minutes"`
	DiagnosticMinimumSamples  int                    `json:"diagnostic_minimum_samples"`
	ObservationsPerParameter  float64                `json:"observations_per_parameter"`
	ModelParameterCount       int                    `json:"model_parameter_count"`
	MetricRiskFreeRate        float64                `json:"metric_risk_free_rate"`
	MetricPeriodsPerYear      float64                `json:"metric_periods_per_year"`
	FeatureSetID              string                 `json:"feature_set_id"`
	FeatureSetVersion         string                 `json:"feature_set_version"`
	RegimeDefinitionID        string                 `json:"regime_definition_id"`
	RegimeVersion             string                 `json:"regime_version"`
	MissingValuePolicy        string                 `json:"missing_value_policy"`
	FilteringPolicy           string                 `json:"filtering_policy"`
	GateThresholds            GateThresholds         `json:"gate_thresholds"`
	BuildTags                 []string               `json:"build_tags"`
}

type CanonicalBracketConfiguration struct {
	Name  string `json:"name"`
	TPBPS string `json:"tp_bps"`
	SLBPS string `json:"sl_bps"`
}

type CanonicalGateThresholds struct {
	MinimumEvents                     int    `json:"minimum_events"`
	MinimumH2PF                       string `json:"minimum_h2_pf"`
	MinimumH2ExpectancyBPS            string `json:"minimum_h2_expectancy_bps"`
	MinimumFYPF                       string `json:"minimum_fy_pf"`
	MinimumFYExpectancyBPS            string `json:"minimum_fy_expectancy_bps"`
	MinimumPositiveMonths             int    `json:"minimum_positive_months"`
	MinimumDelayOneExpectancyBPS      string `json:"minimum_delay_one_expectancy_bps"`
	MaximumSingleMonthContributionPct string `json:"maximum_single_month_contribution_pct"`
	MinimumWorstQuarterPF             string `json:"minimum_worst_quarter_pf"`
	MinimumBracketPF                  string `json:"minimum_bracket_pf"`
	MinimumBracketExpectancyBPS       string `json:"minimum_bracket_expectancy_bps"`
}

type CanonicalResearchConfiguration struct {
	Symbol                     string                          `json:"symbol"`
	Market                     string                          `json:"market"`
	Interval                   string                          `json:"interval"`
	EvaluationStartUTC         string                          `json:"evaluation_start_utc"`
	EvaluationEndUTC           string                          `json:"evaluation_end_utc"`
	ForwardHorizonsNS          []int64                         `json:"forward_horizons_ns"`
	CostHaircutsBPS            []string                        `json:"cost_haircuts_bps"`
	CostHaircutHorizonNS       int64                           `json:"cost_haircut_horizon_ns"`
	EntryDelayCandles          []int                           `json:"entry_delay_candles"`
	EntryDelayHorizonNS        int64                           `json:"entry_delay_horizon_ns"`
	EntryDelayCostBPS          string                          `json:"entry_delay_cost_bps"`
	SeriesHorizonNS            int64                           `json:"series_horizon_ns"`
	SeriesCostBPS              string                          `json:"series_cost_bps"`
	SeriesEntryDelayCandles    int                             `json:"series_entry_delay_candles"`
	ExcursionWindowsNS         []int64                         `json:"excursion_windows_ns"`
	Brackets                   []CanonicalBracketConfiguration `json:"brackets"`
	BracketWindowNS            int64                           `json:"bracket_window_ns"`
	BracketCostBPS             string                          `json:"bracket_cost_bps"`
	StabilityAggregatePeriods  []string                        `json:"stability_aggregate_periods"`
	StabilityHorizonNS         int64                           `json:"stability_horizon_ns"`
	StabilityCostBPS           string                          `json:"stability_cost_bps"`
	StabilityEntryDelayCandles int                             `json:"stability_entry_delay_candles"`
	RegimeGroups               []string                        `json:"regime_groups"`
	RegimeHorizonNS            int64                           `json:"regime_horizon_ns"`
	RegimeCostBPS              string                          `json:"regime_cost_bps"`
	RegimeLowSampleMinimum     int                             `json:"regime_low_sample_minimum"`
	ClusterSeparationNS        int64                           `json:"cluster_separation_ns"`
	DiagnosticMinimumSamples   int                             `json:"diagnostic_minimum_samples"`
	ObservationsPerParameter   string                          `json:"observations_per_parameter"`
	ModelParameterCount        int                             `json:"model_parameter_count"`
	MetricRiskFreeRate         string                          `json:"metric_risk_free_rate"`
	MetricPeriodsPerYear       string                          `json:"metric_periods_per_year"`
	FeatureSetID               string                          `json:"feature_set_id"`
	FeatureSetVersion          string                          `json:"feature_set_version"`
	RegimeDefinitionID         string                          `json:"regime_definition_id"`
	RegimeVersion              string                          `json:"regime_version"`
	MissingValuePolicy         string                          `json:"missing_value_policy"`
	FilteringPolicy            string                          `json:"filtering_policy"`
	GateThresholds             CanonicalGateThresholds         `json:"gate_thresholds"`
	BuildTags                  []string                        `json:"build_tags"`
}

type ConfigurationIdentity struct {
	Contract     canonicalcontract.ContractHeader `json:"contract"`
	ArtifactHash string                           `json:"artifact_hash"`
	CanonicalResearchConfiguration
	EncodingVersion string                        `json:"-"`
	Hash            string                        `json:"-"`
	Effective       ResolvedResearchConfiguration `json:"-"`
}

type RepositoryState struct {
	Root           string
	RepositoryID   string
	CommitSHA      string
	TreeSHA        string
	Dirty          bool
	BuildVersion   string
	GoVersion      string
	GoOS           string
	GoARCH         string
	Compiler       string
	CGOEnabled     string
	BuildTags      []string
	BinaryRevision string
	BinaryModified bool
}

type RepositoryStateProvider interface {
	Resolve(repositoryRoot string) (RepositoryState, error)
}

type EngineSourceIdentity struct {
	Contract     canonicalcontract.ContractHeader `json:"contract"`
	ArtifactHash string                           `json:"artifact_hash"`
	RepositoryID string                           `json:"repository_id"`
	CommitSHA    string                           `json:"commit_sha"`
	TreeSHA      string                           `json:"tree_sha"`
	Dirty        bool                             `json:"dirty"`
	BuildVersion string                           `json:"build_version"`
	GoVersion    string                           `json:"go_version"`
	GoOS         string                           `json:"goos"`
	GoARCH       string                           `json:"goarch"`
	Compiler     string                           `json:"compiler"`
	CGOEnabled   string                           `json:"cgo_enabled"`
	BuildTags    []string                         `json:"build_tags"`
}

type DatasetObjectIdentity struct {
	RelativePath       string `json:"relative_path"`
	Symbol             string `json:"symbol"`
	Interval           string `json:"interval"`
	SizeBytes          int64  `json:"size_bytes"`
	SHA256             string `json:"raw_object_hash"`
	RowCount           int64  `json:"row_count"`
	WindowRowCount     int64  `json:"window_row_count"`
	EarliestEventUTC   string `json:"earliest_event_utc"`
	LatestEventUTC     string `json:"latest_event_utc"`
	LatestAvailableUTC string `json:"latest_available_utc"`
}

type DatasetIdentity struct {
	Contract                  canonicalcontract.ContractHeader `json:"contract"`
	ArtifactHash              string                           `json:"artifact_hash"`
	DatasetID                 string                           `json:"dataset_id"`
	DatasetVersion            string                           `json:"dataset_version"`
	InstrumentUniverseID      string                           `json:"instrument_universe_id"`
	Symbols                   []string                         `json:"symbols"`
	StartUTC                  string                           `json:"dataset_start_utc"`
	EndUTC                    string                           `json:"dataset_end_utc"`
	Objects                   []DatasetObjectIdentity          `json:"objects"`
	ManifestID                string                           `json:"-"`
	ManifestVersion           string                           `json:"-"`
	ManifestHash              string                           `json:"-"`
	ManifestRawHash           string                           `json:"-"`
	DatasetHash               string                           `json:"-"`
	SourceArchiveID           string                           `json:"-"`
	SourceArchiveHash         string                           `json:"-"`
	PointInTimeCutoffUTC      string                           `json:"-"`
	AvailabilityPolicyID      string                           `json:"-"`
	AvailabilityPolicyVersion string                           `json:"-"`
	AvailabilityPolicyHash    string                           `json:"-"`
	CoveragePolicyID          string                           `json:"-"`
	CoveragePolicyVersion     string                           `json:"-"`
	CoveragePolicyHash        string                           `json:"-"`
}

type PITEvidenceIdentity struct {
	Contract                canonicalcontract.ContractHeader `json:"contract"`
	ArtifactHash            string                           `json:"artifact_hash"`
	EvidenceID              string                           `json:"evidence_id"`
	EvidenceVersion         string                           `json:"evidence_version"`
	DatasetID               string                           `json:"dataset_id"`
	DatasetVersion          string                           `json:"dataset_version"`
	DatasetHash             string                           `json:"dataset_hash"`
	SourceArchiveHash       string                           `json:"source_archive_hash"`
	AvailabilityPolicyHash  string                           `json:"availability_policy_hash"`
	CoveragePolicyHash      string                           `json:"coverage_policy_hash"`
	EvidenceHash            string                           `json:"-"`
	PITPolicyID             string                           `json:"-"`
	PITPolicyVersion        string                           `json:"-"`
	PITPolicyHash           string                           `json:"-"`
	CoveragePolicyID        string                           `json:"-"`
	CoveragePolicyVersion   string                           `json:"-"`
	SourceArchiveID         string                           `json:"-"`
	Status                  string                           `json:"status"`
	EvaluationCutoffUTC     string                           `json:"evaluation_cutoff_utc"`
	AvailabilityDelayNS     int64                            `json:"availability_delay_ns"`
	AvailabilityDelayMS     int64                            `json:"-"`
	FullWindowCoverage      bool                             `json:"full_window_coverage"`
	EarliestEventUTC        string                           `json:"earliest_event_utc"`
	LatestEventUTC          string                           `json:"latest_event_utc"`
	EarliestAvailableUTC    string                           `json:"earliest_available_utc"`
	LatestAvailableUTC      string                           `json:"latest_available_utc"`
	GapCount                int64                            `json:"gap_count"`
	DuplicateTimestampCount int64                            `json:"duplicate_timestamp_count"`
	OutOfOrderCount         int64                            `json:"out_of_order_count"`
}

type FeatureIdentity struct {
	Contract             canonicalcontract.ContractHeader `json:"contract"`
	ArtifactHash         string                           `json:"artifact_hash"`
	FeatureSetID         string                           `json:"feature_set_id"`
	FeatureSetVersion    string                           `json:"feature_set_version"`
	ConfigurationHash    string                           `json:"configuration_hash"`
	ImplementationHash   string                           `json:"implementation_hash"`
	ImplementationFiles  []FileInventoryEntry             `json:"implementation_files"`
	InputDatasetHash     string                           `json:"input_dataset_hash"`
	OutputArtifactHash   string                           `json:"output_artifact_hash"`
	WindowStartUTC       string                           `json:"window_start_utc"`
	WindowEndUTC         string                           `json:"window_end_utc"`
	RowCount             int                              `json:"row_count"`
	ImplementationCommit string                           `json:"implementation_commit"`
	WindowStartMS        int64                            `json:"-"`
	WindowEndMS          int64                            `json:"-"`
}

type RegimeIdentity struct {
	Contract             canonicalcontract.ContractHeader `json:"contract"`
	ArtifactHash         string                           `json:"artifact_hash"`
	RegimeDefinitionID   string                           `json:"regime_definition_id"`
	RegimeVersion        string                           `json:"regime_version"`
	ConfigurationHash    string                           `json:"configuration_hash"`
	ImplementationHash   string                           `json:"implementation_hash"`
	ImplementationFiles  []FileInventoryEntry             `json:"implementation_files"`
	InputDatasetHash     string                           `json:"input_dataset_hash"`
	InputFeatureHash     string                           `json:"input_feature_hash"`
	OutputArtifactHash   string                           `json:"output_artifact_hash"`
	WindowStartUTC       string                           `json:"window_start_utc"`
	WindowEndUTC         string                           `json:"window_end_utc"`
	RowCount             int                              `json:"row_count"`
	ImplementationCommit string                           `json:"implementation_commit"`
	WindowStartMS        int64                            `json:"-"`
	WindowEndMS          int64                            `json:"-"`
}

type ConsumedDatasetObject struct {
	RelativePath string `json:"relative_path"`
	ObjectHash   string `json:"object_hash"`
}

type CanonicalFeatureRow struct {
	Market             string `json:"market"`
	Symbol             string `json:"symbol"`
	Interval           string `json:"interval"`
	EventTimeUTC       string `json:"event_time_utc"`
	AvailableAtUTC     string `json:"available_at_utc"`
	Close              string `json:"close"`
	Return1            string `json:"return_1"`
	Return5            string `json:"return_5"`
	Return15           string `json:"return_15"`
	RealizedVol20      string `json:"realized_vol_20"`
	RealizedVol60      string `json:"realized_vol_60"`
	ATR14              string `json:"atr_14"`
	ATRPct14           string `json:"atr_pct_14"`
	BBWidth20          string `json:"bb_width_20"`
	BBWidthPctRank60   string `json:"bb_width_pct_rank_60"`
	EMA20              string `json:"ema_20"`
	EMA50              string `json:"ema_50"`
	EMA200             string `json:"ema_200"`
	TrendSlope20       string `json:"trend_slope_20"`
	VolumeRatio20      string `json:"volume_ratio_20"`
	QuoteVolumeRatio20 string `json:"quote_volume_ratio_20"`
	TakerBuyRatio      string `json:"taker_buy_ratio"`
	BTCReturn60        string `json:"btc_return_60"`
	ETHReturn60        string `json:"eth_return_60"`
	Warmup             bool   `json:"warmup"`
}

type CanonicalRegimeRow struct {
	Market         string   `json:"market"`
	Symbol         string   `json:"symbol"`
	Interval       string   `json:"interval"`
	EventTimeUTC   string   `json:"event_time_utc"`
	AvailableAtUTC string   `json:"available_at_utc"`
	Volatility     string   `json:"volatility"`
	Trend          string   `json:"trend"`
	Liquidity      string   `json:"liquidity"`
	MarketBeta     string   `json:"market_beta"`
	Sentiment      string   `json:"sentiment"`
	Composite      string   `json:"composite"`
	Reasons        []string `json:"reasons"`
	Warmup         bool     `json:"warmup"`
}

type CanonicalCandleRow struct {
	Market              string `json:"market"`
	Symbol              string `json:"symbol"`
	Interval            string `json:"interval"`
	OpenTimeUTC         string `json:"open_time_utc"`
	Open                string `json:"open"`
	High                string `json:"high"`
	Low                 string `json:"low"`
	Close               string `json:"close"`
	Volume              string `json:"volume"`
	CloseTimeUTC        string `json:"close_time_utc"`
	QuoteAssetVolume    string `json:"quote_asset_volume"`
	NumberOfTrades      int64  `json:"number_of_trades"`
	TakerBuyBaseVolume  string `json:"taker_buy_base_volume"`
	TakerBuyQuoteVolume string `json:"taker_buy_quote_volume"`
}

type ConsumedInputIdentity struct {
	Contract                  canonicalcontract.ContractHeader `json:"contract"`
	ArtifactHash              string                           `json:"artifact_hash"`
	DatasetHash               string                           `json:"dataset_hash"`
	DatasetObjectRecords      []ConsumedDatasetObject          `json:"dataset_objects"`
	FeatureHash               string                           `json:"feature_artifact_hash"`
	RegimeHash                string                           `json:"regime_artifact_hash,omitempty"`
	Symbols                   []string                         `json:"symbols"`
	EvaluationStartUTC        string                           `json:"evaluation_start_utc"`
	EvaluationEndUTC          string                           `json:"evaluation_end_utc"`
	PointInTimeCutoffUTC      string                           `json:"point_in_time_cutoff_utc"`
	MissingValuePolicy        string                           `json:"missing_value_policy"`
	FilteringPolicy           string                           `json:"filtering_policy"`
	FeatureRows               []CanonicalFeatureRow            `json:"feature_rows"`
	RegimeRows                []CanonicalRegimeRow             `json:"regime_rows,omitempty"`
	Candles                   []CanonicalCandleRow             `json:"candles"`
	EvaluationEventTimestamps []string                         `json:"evaluation_event_timestamps"`
	EncodingVersion           string                           `json:"-"`
	Hash                      string                           `json:"-"`
	DatasetObjects            []string                         `json:"-"`
	EvaluationStartMS         int64                            `json:"-"`
	EvaluationEndMS           int64                            `json:"-"`
	FeatureRowCount           int                              `json:"-"`
	RegimeRowCount            int                              `json:"-"`
	CandleRowCount            int                              `json:"-"`
	EvaluationEventCount      int                              `json:"-"`
	InputSeriesCount          int                              `json:"-"`
}

type EvaluationSeriesIdentity struct {
	Contract                canonicalcontract.ContractHeader `json:"contract"`
	ArtifactHash            string                           `json:"artifact_hash"`
	SeriesGenerationVersion string                           `json:"generation_version"`
	Returns                 []string                         `json:"returns"`
	Timestamps              []string                         `json:"timestamps"`
	ObservationCount        int                              `json:"observation_count"`
	EncodingVersion         string                           `json:"-"`
	ReturnSeriesHash        string                           `json:"-"`
	TimestampSeriesHash     string                           `json:"-"`
	SeriesHash              string                           `json:"-"`
	ReturnCount             int                              `json:"-"`
	TimestampCount          int                              `json:"-"`
}

type BoundResearchIdentity struct {
	Contract                 canonicalcontract.ContractHeader `json:"contract"`
	ArtifactHash             string                           `json:"artifact_hash"`
	Candidate                CandidateIdentity                `json:"candidate"`
	Configuration            ConfigurationIdentity            `json:"configuration"`
	EngineSource             EngineSourceIdentity             `json:"engine_source"`
	Dataset                  DatasetIdentity                  `json:"dataset"`
	PIT                      PITEvidenceIdentity              `json:"pit_evidence"`
	Feature                  FeatureIdentity                  `json:"feature"`
	Regime                   *RegimeIdentity                  `json:"regime,omitempty"`
	ConsumedInput            ConsumedInputIdentity            `json:"consumed_input"`
	Series                   EvaluationSeriesIdentity         `json:"evaluation_series"`
	HistorianManifestHash    string                           `json:"historian_manifest_hash"`
	HistorianManifestRawHash string                           `json:"historian_manifest_raw_hash"`
	EncodingVersion          string                           `json:"-"`
	IdentityHash             string                           `json:"-"`
}

type Assessment struct {
	Status   IdentityStatus         `json:"status"`
	Identity *BoundResearchIdentity `json:"identity,omitempty"`
	Findings []Finding              `json:"findings"`
}

type DerivationRequest struct {
	RepositoryRoot            string
	CandidateFamily           string
	CandidateSide             string
	Configuration             ResolvedResearchConfiguration
	HistorianManifestPath     string
	DatasetRoot               string
	FeatureArtifactPath       string
	RegimeArtifactPath        string
	ConsumedDatasetPaths      []string
	FeatureRows               []features.Row
	RegimeLabels              []regime.Label
	Candles                   []protocol.Candle
	EvaluationEventTimestamps []int64
	Returns                   []float64
	Timestamps                []int64
}

type Deriver struct {
	sourceProvider RepositoryStateProvider
	registry       *Registry
	now            func() time.Time
}

func NewDeriver() *Deriver {
	return &Deriver{sourceProvider: gitRepositoryStateProvider{}, now: time.Now}
}

func NewDeriverWithProvider(provider RepositoryStateProvider, now func() time.Time) *Deriver {
	if now == nil {
		now = time.Now
	}
	return &Deriver{sourceProvider: provider, now: now}
}

func NewDeriverWithDependencies(registry *Registry, provider RepositoryStateProvider, now func() time.Time) *Deriver {
	if now == nil {
		now = time.Now
	}
	return &Deriver{sourceProvider: provider, registry: registry, now: now}
}
