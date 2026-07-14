package qualification

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"path/filepath"
	"sort"
	"strings"
	"time"
	"unicode"
)

const (
	InventorySchemaVersion             = "ak.engine.candidate_inventory.v1"
	ProtocolSchemaVersion              = "ak.engine.candidate_qualification_protocol.v1"
	QualificationReportSchemaVersion   = "ak.engine.candidate_qualification.v1"
	FrozenDescriptorSchemaVersion      = "ak.engine.frozen_candidate.v1"
	FrozenDescriptorSchemaVersionV2    = "ak.engine.frozen_candidate.v2"
	RegistrationRequestSchemaVersion   = "ak.engine.candidate_registration_request.v1"
	RegistrationRequestSchemaVersionV2 = "ak.engine.candidate_registration_request.v2"
	ResearchIdentitySchemaVersion      = "ak.rif.research_identity.v1"
	ResearchIdentitySchemaVersionV2    = "ak.rif.research_identity.v2"
	RegistrationRequestLabel           = "CANDIDATE_REGISTRATION_REQUEST"
)

type EligibilityClassification string

const (
	ClassificationQualificationCandidate EligibilityClassification = "QUALIFICATION_CANDIDATE"
	ClassificationRejected               EligibilityClassification = "REJECTED"
	ClassificationNearMiss               EligibilityClassification = "NEAR_MISS"
	ClassificationInfrastructureProbe    EligibilityClassification = "INFRASTRUCTURE_PROBE"
	ClassificationContextProofOnly       EligibilityClassification = "CONTEXT_PROOF_ONLY"
	ClassificationInsufficientSample     EligibilityClassification = "INSUFFICIENT_SAMPLE"
	ClassificationMissingEvidence        EligibilityClassification = "MISSING_EVIDENCE"
	ClassificationSuperseded             EligibilityClassification = "SUPERSEDED"
)

type FinalStatus string

const (
	StatusQualified                     FinalStatus = "QUALIFIED"
	StatusRejected                      FinalStatus = "REJECTED"
	StatusNearMiss                      FinalStatus = "NEAR_MISS"
	StatusInsufficientSample            FinalStatus = "INSUFFICIENT_SAMPLE"
	StatusMissingRequiredData           FinalStatus = "MISSING_REQUIRED_DATA"
	StatusPITEvidenceMissing            FinalStatus = "PIT_EVIDENCE_MISSING"
	StatusImplementationNotReproducible FinalStatus = "IMPLEMENTATION_NOT_REPRODUCIBLE"
	StatusHoldoutNotAuthorized          FinalStatus = "HOLDOUT_NOT_AUTHORIZED"
)

type EvidenceReference struct {
	ArtifactID string `json:"artifact_id"`
	SourceRef  string `json:"source_ref"`
	SHA256     string `json:"sha256"`
}

type EvidenceResult struct {
	Status  string         `json:"status"`
	Metrics map[string]any `json:"metrics"`
	Notes   []string       `json:"notes"`
}

type SampleSize struct {
	EventCount              int `json:"event_count"`
	IndependentClusterCount int `json:"independent_cluster_count"`
	TradesOrDecisions       int `json:"trades_or_decisions"`
	SymbolsRepresented      int `json:"symbols_represented"`
	MonthsRepresented       int `json:"months_represented"`
	QuartersRepresented     int `json:"quarters_represented"`
	PositiveRegimes         int `json:"positive_regimes"`
	NegativeRegimes         int `json:"negative_regimes"`
}

type CandidateRecord struct {
	CandidateID                string                    `json:"candidate_id"`
	CandidateVersion           string                    `json:"candidate_version"`
	RegisteredImplementation   bool                      `json:"registered_implementation"`
	ImplementationLocation     string                    `json:"implementation_location"`
	ImplementationSourceRef    string                    `json:"implementation_source_ref"`
	ImplementationSHA256       string                    `json:"implementation_sha256"`
	ImplementationReproducible bool                      `json:"implementation_reproducible"`
	StrategyFamily             string                    `json:"strategy_family"`
	DirectionSupport           []string                  `json:"direction_support"`
	Symbols                    []string                  `json:"symbols"`
	RequiredContext            []string                  `json:"required_context"`
	RequiredTimeframes         []string                  `json:"required_timeframes"`
	FeatureRequirements        []string                  `json:"feature_requirements"`
	ParameterSet               map[string]any            `json:"parameter_set"`
	ResearchPhase              string                    `json:"research_phase"`
	SampleSize                 SampleSize                `json:"sample_size"`
	InSampleResults            EvidenceResult            `json:"in_sample_results"`
	OutOfSampleResults         EvidenceResult            `json:"out_of_sample_results"`
	WalkForwardResults         EvidenceResult            `json:"walk_forward_results"`
	CostStressResults          EvidenceResult            `json:"cost_stress_results"`
	WorstPeriodResults         EvidenceResult            `json:"worst_period_results"`
	ConcentrationResults       EvidenceResult            `json:"concentration_results"`
	KnownDefects               []string                  `json:"known_defects"`
	CurrentResearchLabel       string                    `json:"current_research_label"`
	EligibilityClassification  EligibilityClassification `json:"current_eligibility_classification"`
	FinalStatus                FinalStatus               `json:"final_status"`
	ExclusionReasons           []string                  `json:"exclusion_reasons"`
	Evidence                   []EvidenceReference       `json:"evidence"`
}

type CandidateInventory struct {
	SchemaVersion          string            `json:"schema_version"`
	Phase                  string            `json:"phase"`
	AcceptedEngineBaseline string            `json:"accepted_engine_baseline"`
	CandidateCount         int               `json:"candidate_count"`
	Candidates             []CandidateRecord `json:"candidates"`
	RegisteredCandidateIDs []string          `json:"registered_candidate_ids"`
	UnknownImplementations []string          `json:"unknown_candidate_implementations"`
	OmittedCandidates      []string          `json:"omitted_candidates"`
}

func ValidateInventory(inventory CandidateInventory, actualRegisteredIDs []string) error {
	if inventory.SchemaVersion != InventorySchemaVersion {
		return fmt.Errorf("unsupported inventory schema_version %q", inventory.SchemaVersion)
	}
	if strings.TrimSpace(inventory.AcceptedEngineBaseline) == "" {
		return errors.New("accepted_engine_baseline is required")
	}
	if inventory.CandidateCount != len(inventory.Candidates) {
		return fmt.Errorf("candidate_count=%d does not match candidates=%d", inventory.CandidateCount, len(inventory.Candidates))
	}
	seen := make(map[string]CandidateRecord, len(inventory.Candidates))
	for _, candidate := range inventory.Candidates {
		if err := validateCandidateRecord(candidate); err != nil {
			return fmt.Errorf("candidate %q: %w", candidate.CandidateID, err)
		}
		if _, exists := seen[candidate.CandidateID]; exists {
			return fmt.Errorf("duplicate candidate_id %q", candidate.CandidateID)
		}
		seen[candidate.CandidateID] = candidate
	}
	wantRegistered, err := sortedUniqueStrings(actualRegisteredIDs)
	if err != nil {
		return fmt.Errorf("actual registered candidate IDs: %w", err)
	}
	gotRegistered, err := sortedUniqueStrings(inventory.RegisteredCandidateIDs)
	if err != nil {
		return fmt.Errorf("registered_candidate_ids: %w", err)
	}
	if !equalStrings(wantRegistered, gotRegistered) {
		return fmt.Errorf("registered_candidate_ids mismatch: want %v got %v", wantRegistered, gotRegistered)
	}
	for _, id := range wantRegistered {
		candidate, exists := seen[id]
		if !exists || !candidate.RegisteredImplementation {
			return fmt.Errorf("registered candidate %q is missing or not marked registered", id)
		}
	}
	unknown := FindUnknownImplementations(inventory.Candidates, actualRegisteredIDs)
	if !equalStrings(unknown, inventory.UnknownImplementations) {
		return fmt.Errorf("unknown_candidate_implementations mismatch: want %v got %v", unknown, inventory.UnknownImplementations)
	}
	if len(inventory.OmittedCandidates) != 0 {
		return fmt.Errorf("inventory declares omitted candidates: %v", inventory.OmittedCandidates)
	}
	return nil
}

func FindUnknownImplementations(candidates []CandidateRecord, actualRegisteredIDs []string) []string {
	known := make(map[string]struct{}, len(candidates))
	for _, candidate := range candidates {
		known[candidate.CandidateID] = struct{}{}
	}
	unknown := make([]string, 0)
	for _, id := range actualRegisteredIDs {
		if _, exists := known[id]; !exists {
			unknown = append(unknown, id)
		}
	}
	sort.Strings(unknown)
	return unknown
}

func validateCandidateRecord(candidate CandidateRecord) error {
	for name, value := range map[string]string{
		"candidate_id": candidate.CandidateID, "candidate_version": candidate.CandidateVersion,
		"implementation_location": candidate.ImplementationLocation, "implementation_source_ref": candidate.ImplementationSourceRef,
		"strategy_family": candidate.StrategyFamily,
		"research_phase":  candidate.ResearchPhase, "current_research_label": candidate.CurrentResearchLabel,
	} {
		if strings.TrimSpace(value) == "" {
			return fmt.Errorf("%s is required", name)
		}
	}
	if len(candidate.DirectionSupport) == 0 || len(candidate.Symbols) == 0 || len(candidate.RequiredTimeframes) == 0 || len(candidate.FeatureRequirements) == 0 {
		return errors.New("direction_support, symbols, required_timeframes, and feature_requirements are required")
	}
	if candidate.ParameterSet == nil {
		return errors.New("parameter_set is required")
	}
	if !validSHA256(candidate.ImplementationSHA256) {
		return errors.New("implementation_sha256 is invalid")
	}
	for _, evidence := range candidate.Evidence {
		if strings.TrimSpace(evidence.ArtifactID) == "" || strings.TrimSpace(evidence.SourceRef) == "" || !validSHA256(evidence.SHA256) {
			return errors.New("evidence references require artifact_id, source_ref, and valid sha256")
		}
	}
	if candidate.EligibilityClassification == "" || candidate.FinalStatus == "" {
		return errors.New("classification and final_status are required")
	}
	if len(candidate.ExclusionReasons) == 0 && candidate.FinalStatus != StatusQualified {
		return errors.New("non-qualified candidate requires explicit exclusion_reasons")
	}
	switch candidate.EligibilityClassification {
	case ClassificationRejected:
		if candidate.FinalStatus != StatusRejected {
			return errors.New("rejected candidate must remain REJECTED")
		}
	case ClassificationNearMiss:
		if candidate.FinalStatus != StatusNearMiss {
			return errors.New("near-miss candidate must remain NEAR_MISS")
		}
	case ClassificationInfrastructureProbe, ClassificationContextProofOnly:
		if candidate.FinalStatus == StatusQualified {
			return errors.New("infrastructure/context proof cannot be QUALIFIED")
		}
	case ClassificationInsufficientSample:
		if candidate.FinalStatus != StatusInsufficientSample {
			return errors.New("insufficient-sample candidate must remain INSUFFICIENT_SAMPLE")
		}
	case ClassificationMissingEvidence, ClassificationSuperseded, ClassificationQualificationCandidate:
	default:
		return fmt.Errorf("unknown eligibility classification %q", candidate.EligibilityClassification)
	}
	return nil
}

type Window struct {
	Start time.Time `json:"start"`
	End   time.Time `json:"end"`
}

type DataIntegrityGates struct {
	DatasetID              string   `json:"dataset_id"`
	DatasetVersion         string   `json:"dataset_version"`
	ManifestID             string   `json:"manifest_id"`
	ManifestHash           string   `json:"manifest_hash"`
	ResearchWindow         Window   `json:"research_window"`
	DevelopmentWindow      Window   `json:"development_window"`
	ValidationWindow       Window   `json:"validation_window"`
	FinalHoldoutWindow     Window   `json:"final_holdout_window"`
	RequiredSymbols        []string `json:"required_symbols"`
	RequiredContextSymbols []string `json:"required_context_symbols"`
	ExpectedPartitions     []string `json:"expected_partitions"`
	GapPolicy              string   `json:"gap_policy"`
	PITAvailabilityPolicy  string   `json:"pit_availability_policy"`
}

type SampleGates struct {
	MinimumEvents              int `json:"minimum_events"`
	MinimumIndependentClusters int `json:"minimum_independent_clusters"`
	MinimumTradesOrDecisions   int `json:"minimum_trades_or_decisions"`
	MinimumSymbols             int `json:"minimum_symbols"`
	MinimumMonths              int `json:"minimum_months"`
	MinimumPositiveRegimes     int `json:"minimum_positive_regimes"`
	MinimumNegativeRegimes     int `json:"minimum_negative_regimes"`
}

type PerformanceGates struct {
	MinimumNetExpectancyBPS        float64 `json:"minimum_net_expectancy_bps"`
	MinimumProfitFactor            float64 `json:"minimum_profit_factor"`
	MaximumDrawdownBPS             float64 `json:"maximum_drawdown_bps"`
	MinimumConfidenceLowerBoundBPS float64 `json:"minimum_confidence_lower_bound_bps"`
	DownsideTailPolicy             string  `json:"downside_tail_policy"`
}

type RobustnessGates struct {
	RequireOutOfSample                 bool    `json:"require_out_of_sample"`
	RequireWalkForward                 bool    `json:"require_walk_forward"`
	MinimumWorstPeriodProfitFactor     float64 `json:"minimum_worst_period_profit_factor"`
	MaximumSymbolContributionPercent   float64 `json:"maximum_symbol_contribution_percent"`
	MaximumTemporalContributionPercent float64 `json:"maximum_temporal_contribution_percent"`
	MaximumRegimeContributionPercent   float64 `json:"maximum_regime_contribution_percent"`
	MinimumStableNeighbors             int     `json:"minimum_stable_parameter_neighbors"`
	RequireClusterDeduplication        bool    `json:"require_cluster_deduplication"`
	RequireMissingContextSensitivity   bool    `json:"require_missing_context_sensitivity"`
}

type CostGates struct {
	FeeBPS                     float64 `json:"fee_bps"`
	SpreadBPS                  float64 `json:"spread_bps"`
	SlippageBPS                float64 `json:"slippage_bps"`
	FundingBPS                 float64 `json:"funding_bps"`
	AdverseSelectionBPS        float64 `json:"adverse_selection_bps"`
	StressTotalBPS             float64 `json:"stress_total_bps"`
	MinimumStressProfitFactor  float64 `json:"minimum_stress_profit_factor"`
	MinimumStressExpectancyBPS float64 `json:"minimum_stress_expectancy_bps"`
}

type SearchBudget struct {
	MaximumCandidateCount        int    `json:"maximum_candidate_count"`
	MaximumEvaluations           int    `json:"maximum_total_evaluations"`
	StoppingRule                 string `json:"stopping_rule"`
	SelectionRule                string `json:"selection_rule"`
	FinalHoldoutUsedForSelection bool   `json:"final_holdout_used_for_selection"`
}

type QualificationProtocol struct {
	SchemaVersion  string             `json:"schema_version"`
	DataIntegrity  DataIntegrityGates `json:"data_integrity"`
	Sample         SampleGates        `json:"sample_sufficiency"`
	Performance    PerformanceGates   `json:"performance"`
	Robustness     RobustnessGates    `json:"robustness"`
	Cost           CostGates          `json:"cost_stress"`
	LeakageRules   []string           `json:"leakage_rules"`
	SimplicityRule string             `json:"simplicity_rule"`
	Search         SearchBudget       `json:"search_budget"`
}

func ValidateProtocol(protocol QualificationProtocol) error {
	if protocol.SchemaVersion != ProtocolSchemaVersion {
		return fmt.Errorf("unsupported protocol schema_version %q", protocol.SchemaVersion)
	}
	for name, value := range map[string]string{
		"dataset_id": protocol.DataIntegrity.DatasetID, "dataset_version": protocol.DataIntegrity.DatasetVersion,
		"manifest_id": protocol.DataIntegrity.ManifestID, "manifest_hash": protocol.DataIntegrity.ManifestHash,
		"gap_policy": protocol.DataIntegrity.GapPolicy, "pit_availability_policy": protocol.DataIntegrity.PITAvailabilityPolicy,
		"downside_tail_policy": protocol.Performance.DownsideTailPolicy, "simplicity_rule": protocol.SimplicityRule,
		"stopping_rule": protocol.Search.StoppingRule, "selection_rule": protocol.Search.SelectionRule,
	} {
		if strings.TrimSpace(value) == "" {
			return fmt.Errorf("%s is required", name)
		}
	}
	if !validWindow(protocol.DataIntegrity.ResearchWindow) || !validWindow(protocol.DataIntegrity.DevelopmentWindow) || !validWindow(protocol.DataIntegrity.ValidationWindow) || !validWindow(protocol.DataIntegrity.FinalHoldoutWindow) {
		return errors.New("all protocol windows must be nonempty")
	}
	if protocol.DataIntegrity.DevelopmentWindow.End.After(protocol.DataIntegrity.ValidationWindow.Start) || protocol.DataIntegrity.ValidationWindow.End.After(protocol.DataIntegrity.FinalHoldoutWindow.Start) {
		return errors.New("development, validation, and final-holdout windows overlap")
	}
	if protocol.DataIntegrity.ResearchWindow.Start.After(protocol.DataIntegrity.DevelopmentWindow.Start) || protocol.DataIntegrity.ResearchWindow.End.Before(protocol.DataIntegrity.FinalHoldoutWindow.End) {
		return errors.New("research_window must contain all partitions")
	}
	if len(protocol.DataIntegrity.RequiredSymbols) == 0 || len(protocol.DataIntegrity.ExpectedPartitions) == 0 {
		return errors.New("required_symbols and expected_partitions must be explicit")
	}
	if _, err := sortedUniqueStrings(protocol.DataIntegrity.RequiredSymbols); err != nil {
		return fmt.Errorf("required_symbols: %w", err)
	}
	if _, err := sortedUniqueStrings(protocol.DataIntegrity.ExpectedPartitions); err != nil {
		return fmt.Errorf("expected_partitions: %w", err)
	}
	if !validSHA256(protocol.DataIntegrity.ManifestHash) {
		return errors.New("manifest_hash is invalid")
	}
	if protocol.Sample.MinimumEvents <= 0 || protocol.Sample.MinimumIndependentClusters <= 0 || protocol.Sample.MinimumTradesOrDecisions <= 0 || protocol.Sample.MinimumSymbols <= 0 || protocol.Sample.MinimumMonths <= 0 || protocol.Sample.MinimumPositiveRegimes <= 0 || protocol.Sample.MinimumNegativeRegimes <= 0 {
		return errors.New("all sample minimums must be positive")
	}
	if !allFinite(protocol.Performance.MinimumNetExpectancyBPS, protocol.Performance.MinimumProfitFactor, protocol.Performance.MaximumDrawdownBPS, protocol.Performance.MinimumConfidenceLowerBoundBPS) || protocol.Performance.MinimumProfitFactor <= 1 || protocol.Performance.MaximumDrawdownBPS <= 0 || protocol.Performance.MinimumConfidenceLowerBoundBPS < 0 {
		return errors.New("performance thresholds are invalid")
	}
	if !allFinite(protocol.Robustness.MinimumWorstPeriodProfitFactor, protocol.Robustness.MaximumSymbolContributionPercent, protocol.Robustness.MaximumTemporalContributionPercent, protocol.Robustness.MaximumRegimeContributionPercent) || protocol.Robustness.MinimumWorstPeriodProfitFactor <= 0 || protocol.Robustness.MaximumSymbolContributionPercent <= 0 || protocol.Robustness.MaximumSymbolContributionPercent > 100 || protocol.Robustness.MaximumTemporalContributionPercent <= 0 || protocol.Robustness.MaximumTemporalContributionPercent > 100 || protocol.Robustness.MaximumRegimeContributionPercent <= 0 || protocol.Robustness.MaximumRegimeContributionPercent > 100 || protocol.Robustness.MinimumStableNeighbors <= 0 {
		return errors.New("robustness thresholds are invalid")
	}
	if !allFinite(protocol.Cost.FeeBPS, protocol.Cost.SpreadBPS, protocol.Cost.SlippageBPS, protocol.Cost.FundingBPS, protocol.Cost.AdverseSelectionBPS, protocol.Cost.StressTotalBPS, protocol.Cost.MinimumStressProfitFactor, protocol.Cost.MinimumStressExpectancyBPS) || protocol.Cost.FeeBPS < 0 || protocol.Cost.SpreadBPS < 0 || protocol.Cost.SlippageBPS < 0 || protocol.Cost.FundingBPS < 0 || protocol.Cost.AdverseSelectionBPS < 0 || protocol.Cost.StressTotalBPS <= 0 || protocol.Cost.MinimumStressProfitFactor <= 1 {
		return errors.New("cost assumptions and stress gates must be explicit and nonnegative")
	}
	componentTotal := protocol.Cost.FeeBPS + protocol.Cost.SpreadBPS + protocol.Cost.SlippageBPS + protocol.Cost.FundingBPS + protocol.Cost.AdverseSelectionBPS
	if protocol.Cost.StressTotalBPS < componentTotal {
		return fmt.Errorf("stress_total_bps %.6f is below declared component total %.6f", protocol.Cost.StressTotalBPS, componentTotal)
	}
	if len(protocol.LeakageRules) < 5 {
		return errors.New("leakage rules are incomplete")
	}
	if protocol.Search.MaximumCandidateCount <= 0 || protocol.Search.MaximumEvaluations <= 0 {
		return errors.New("search budget must be finite and positive")
	}
	if protocol.Search.FinalHoldoutUsedForSelection {
		return errors.New("final holdout cannot be used for candidate selection")
	}
	return nil
}

func validWindow(window Window) bool {
	return !window.Start.IsZero() && !window.End.IsZero() && window.Start.Before(window.End)
}

type EvaluationObservation struct {
	EventID            string    `json:"event_id"`
	ClusterID          string    `json:"cluster_id"`
	Partition          string    `json:"partition"`
	DecisionAt         time.Time `json:"decision_at"`
	FeatureAvailableAt time.Time `json:"feature_available_at"`
	UsesFutureData     bool      `json:"uses_future_data"`
	ContextComplete    bool      `json:"context_complete"`
}

type EvaluationInput struct {
	DatasetID          string                  `json:"dataset_id"`
	DatasetVersion     string                  `json:"dataset_version"`
	ManifestID         string                  `json:"manifest_id"`
	ManifestHash       string                  `json:"manifest_hash"`
	ObservedPartitions []string                `json:"observed_partitions"`
	InternalGapCount   int                     `json:"internal_gap_count"`
	Observations       []EvaluationObservation `json:"observations"`
}

func ValidateEvaluationInput(protocol QualificationProtocol, input EvaluationInput) error {
	if err := ValidateProtocol(protocol); err != nil {
		return fmt.Errorf("protocol: %w", err)
	}
	if input.DatasetID != protocol.DataIntegrity.DatasetID {
		return errors.New("dataset_id mismatch")
	}
	if input.DatasetVersion != protocol.DataIntegrity.DatasetVersion {
		return errors.New("dataset_version mismatch")
	}
	if input.ManifestID != protocol.DataIntegrity.ManifestID || input.ManifestHash != protocol.DataIntegrity.ManifestHash {
		return errors.New("manifest identity mismatch")
	}
	if input.InternalGapCount < 0 {
		return errors.New("internal gap count cannot be negative")
	}
	if input.InternalGapCount > 0 && protocol.DataIntegrity.GapPolicy != "ALLOW_DECLARED_GAPS" {
		return errors.New("internal data gaps violate gap policy")
	}
	expectedPartitions, err := sortedUniqueStrings(protocol.DataIntegrity.ExpectedPartitions)
	if err != nil {
		return fmt.Errorf("expected partitions: %w", err)
	}
	observedPartitions, err := sortedUniqueStrings(input.ObservedPartitions)
	if err != nil {
		return fmt.Errorf("observed partitions: %w", err)
	}
	if !equalStrings(expectedPartitions, observedPartitions) {
		return errors.New("observed partitions do not exactly match expected partitions")
	}
	partitionSet := make(map[string]struct{}, len(observedPartitions))
	for _, partition := range observedPartitions {
		partitionSet[partition] = struct{}{}
	}
	seenEvents := make(map[string]struct{}, len(input.Observations))
	for _, observation := range input.Observations {
		if strings.TrimSpace(observation.EventID) == "" || strings.TrimSpace(observation.ClusterID) == "" || strings.TrimSpace(observation.Partition) == "" || observation.DecisionAt.IsZero() || observation.FeatureAvailableAt.IsZero() {
			return errors.New("observation identity and timing are required")
		}
		if _, duplicate := seenEvents[observation.EventID]; duplicate {
			return fmt.Errorf("duplicate event_id %q", observation.EventID)
		}
		seenEvents[observation.EventID] = struct{}{}
		if _, exists := partitionSet[observation.Partition]; !exists {
			return fmt.Errorf("event %q references an unexpected partition", observation.EventID)
		}
		if observation.DecisionAt.Before(protocol.DataIntegrity.ResearchWindow.Start) || observation.DecisionAt.After(protocol.DataIntegrity.ResearchWindow.End) {
			return fmt.Errorf("event %q decision is outside the research window", observation.EventID)
		}
		if observation.UsesFutureData {
			return fmt.Errorf("event %q uses future data", observation.EventID)
		}
		if observation.FeatureAvailableAt.After(observation.DecisionAt) || observation.FeatureAvailableAt.After(protocol.DataIntegrity.FinalHoldoutWindow.End) {
			return fmt.Errorf("event %q uses data unavailable at decision cutoff", observation.EventID)
		}
		if !observation.ContextComplete {
			return fmt.Errorf("event %q is missing required context", observation.EventID)
		}
	}
	return nil
}

func IndependentClusterCount(observations []EvaluationObservation) int {
	clusters := make(map[string]struct{}, len(observations))
	for _, observation := range observations {
		if strings.TrimSpace(observation.ClusterID) != "" {
			clusters[observation.ClusterID] = struct{}{}
		}
	}
	return len(clusters)
}

type GateEvidence struct {
	DataIntegrity             bool `json:"data_integrity"`
	SampleSufficiency         bool `json:"sample_sufficiency"`
	NetPerformance            bool `json:"net_performance"`
	DownsideTail              bool `json:"downside_tail"`
	UncertaintyBound          bool `json:"uncertainty_bound"`
	OutOfSample               bool `json:"out_of_sample"`
	WalkForward               bool `json:"walk_forward"`
	WorstPeriod               bool `json:"worst_period"`
	SymbolConcentration       bool `json:"symbol_concentration"`
	TemporalConcentration     bool `json:"temporal_concentration"`
	RegimeConcentration       bool `json:"regime_concentration"`
	ParameterNeighborhood     bool `json:"parameter_neighborhood"`
	ClusterDeduplication      bool `json:"cluster_deduplication"`
	MissingContextSensitivity bool `json:"missing_context_sensitivity"`
	CostStress                bool `json:"cost_stress"`
	LeakageSafety             bool `json:"leakage_safety"`
	Simplicity                bool `json:"simplicity"`
	ImplementationComplete    bool `json:"implementation_complete"`
	PITEvidence               bool `json:"pit_evidence"`
	HoldoutAuthorized         bool `json:"holdout_authorized"`
}

func (e GateEvidence) AllPassed() bool {
	return e.DataIntegrity && e.SampleSufficiency && e.NetPerformance && e.DownsideTail && e.UncertaintyBound && e.OutOfSample && e.WalkForward && e.WorstPeriod && e.SymbolConcentration && e.TemporalConcentration && e.RegimeConcentration && e.ParameterNeighborhood && e.ClusterDeduplication && e.MissingContextSensitivity && e.CostStress && e.LeakageSafety && e.Simplicity && e.ImplementationComplete && e.PITEvidence && e.HoldoutAuthorized
}

func QualificationStatus(candidate CandidateRecord, evidence GateEvidence, strongest bool) FinalStatus {
	_ = strongest // Ranking is deliberately irrelevant to mandatory gates.
	switch candidate.EligibilityClassification {
	case ClassificationRejected:
		return StatusRejected
	case ClassificationNearMiss:
		return StatusNearMiss
	case ClassificationInfrastructureProbe, ClassificationContextProofOnly, ClassificationInsufficientSample:
		return StatusInsufficientSample
	}
	if !candidate.ImplementationReproducible || !evidence.ImplementationComplete {
		return StatusImplementationNotReproducible
	}
	if !evidence.DataIntegrity {
		return StatusMissingRequiredData
	}
	if !evidence.SampleSufficiency {
		return StatusInsufficientSample
	}
	if !evidence.PITEvidence {
		return StatusPITEvidenceMissing
	}
	if !evidence.HoldoutAuthorized {
		return StatusHoldoutNotAuthorized
	}
	if !evidence.AllPassed() {
		return StatusRejected
	}
	return StatusQualified
}

type FrozenCandidateDescriptor struct {
	SchemaVersion             string    `json:"schema_version"`
	CandidateID               string    `json:"candidate_id"`
	CandidateVersion          string    `json:"candidate_version"`
	StrategyFamily            string    `json:"strategy_family"`
	DirectionModel            string    `json:"direction_model"`
	ImplementationHash        string    `json:"implementation_hash"`
	ConfigurationHash         string    `json:"configuration_hash"`
	ParameterHash             string    `json:"parameter_hash"`
	FeatureSchema             string    `json:"feature_schema"`
	CapabilityHash            string    `json:"capability_hash"`
	EngineModule              string    `json:"engine_module"`
	EngineBuildID             string    `json:"engine_build_id"`
	DatasetID                 string    `json:"dataset_id"`
	DatasetVersion            string    `json:"dataset_version"`
	ResearchWindowStart       time.Time `json:"research_window_start"`
	ResearchWindowEnd         time.Time `json:"research_window_end"`
	EvaluationCutoff          time.Time `json:"evaluation_cutoff"`
	ManifestID                string    `json:"manifest_id"`
	ManifestHash              string    `json:"manifest_hash"`
	CoveragePolicyVersion     string    `json:"coverage_policy_version"`
	AvailabilityPolicyVersion string    `json:"availability_policy_version"`
	IndependencePolicyHash    string    `json:"independence_policy_hash,omitempty"`
	UncertaintyMethodHash     string    `json:"uncertainty_method_hash,omitempty"`
	SourceSchemaHash          string    `json:"source_schema_hash,omitempty"`
	ManifestContractHash      string    `json:"manifest_contract_hash,omitempty"`
	QualificationReportID     string    `json:"qualification_report_id"`
	QualificationReportHash   string    `json:"qualification_report_hash"`
	FrozenAt                  time.Time `json:"frozen_at"`
	DescriptorHash            string    `json:"descriptor_hash"`
}

func (descriptor FrozenCandidateDescriptor) CanonicalHash() (string, error) {
	return canonicalObjectHash(descriptor, "descriptor_hash")
}

func (descriptor FrozenCandidateDescriptor) Verify() error {
	if descriptor.SchemaVersion != FrozenDescriptorSchemaVersion && descriptor.SchemaVersion != FrozenDescriptorSchemaVersionV2 {
		return fmt.Errorf("unsupported frozen descriptor schema_version %q", descriptor.SchemaVersion)
	}
	for name, value := range map[string]string{
		"candidate_id": descriptor.CandidateID, "candidate_version": descriptor.CandidateVersion,
		"strategy_family": descriptor.StrategyFamily, "direction_model": descriptor.DirectionModel,
		"feature_schema": descriptor.FeatureSchema, "engine_module": descriptor.EngineModule,
		"engine_build_id": descriptor.EngineBuildID, "dataset_id": descriptor.DatasetID,
		"dataset_version": descriptor.DatasetVersion, "manifest_id": descriptor.ManifestID,
		"coverage_policy_version":     descriptor.CoveragePolicyVersion,
		"availability_policy_version": descriptor.AvailabilityPolicyVersion,
		"qualification_report_id":     descriptor.QualificationReportID,
	} {
		if strings.TrimSpace(value) == "" {
			return fmt.Errorf("%s is required", name)
		}
	}
	if err := validateGovernanceHashes(descriptor.SchemaVersion == FrozenDescriptorSchemaVersionV2, map[string]string{
		"independence_policy_hash": descriptor.IndependencePolicyHash, "uncertainty_method_hash": descriptor.UncertaintyMethodHash,
		"source_schema_hash": descriptor.SourceSchemaHash, "manifest_contract_hash": descriptor.ManifestContractHash,
	}); err != nil {
		return err
	}
	for name, value := range map[string]string{
		"implementation_hash": descriptor.ImplementationHash, "configuration_hash": descriptor.ConfigurationHash,
		"parameter_hash": descriptor.ParameterHash, "capability_hash": descriptor.CapabilityHash,
		"manifest_hash": descriptor.ManifestHash, "qualification_report_hash": descriptor.QualificationReportHash,
		"descriptor_hash": descriptor.DescriptorHash,
	} {
		if !validSHA256(value) {
			return fmt.Errorf("%s must be sha256: followed by 64 lowercase hexadecimal characters", name)
		}
	}
	if descriptor.ResearchWindowStart.IsZero() || !descriptor.ResearchWindowStart.Before(descriptor.ResearchWindowEnd) || descriptor.EvaluationCutoff.Before(descriptor.ResearchWindowEnd) || descriptor.FrozenAt.IsZero() {
		return errors.New("descriptor time identities are invalid")
	}
	want, err := descriptor.CanonicalHash()
	if err != nil {
		return err
	}
	if descriptor.DescriptorHash != want {
		return errors.New("descriptor_hash mismatch")
	}
	return nil
}

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
}

type CandidateImplementationIdentity struct {
	CandidateID        string `json:"candidate_id"`
	CandidateVersion   string `json:"candidate_version"`
	ImplementationHash string `json:"implementation_hash"`
	ConfigurationHash  string `json:"configuration_hash"`
	ParameterHash      string `json:"parameter_hash"`
	CapabilityHash     string `json:"capability_hash"`
}

type CandidateRegistrationRequest struct {
	SchemaVersion                   string                          `json:"schema_version"`
	ArtifactLabel                   string                          `json:"artifact_label"`
	FrozenCandidate                 FrozenCandidateDescriptor       `json:"frozen_candidate_descriptor"`
	QualificationVerdict            FinalStatus                     `json:"qualification_verdict"`
	QualificationReportID           string                          `json:"qualification_report_id"`
	QualificationReportHash         string                          `json:"qualification_report_hash"`
	CandidateImplementationIdentity CandidateImplementationIdentity `json:"candidate_implementation_identity"`
	ResearchIdentity                ResearchIdentity                `json:"research_identity"`
	RequestedLifecycleStartingState string                          `json:"requested_lifecycle_starting_state"`
	RIFAuthorized                   bool                            `json:"rif_authorized"`
	ArtifactIntegrityHash           string                          `json:"artifact_integrity_hash"`
}

func (request CandidateRegistrationRequest) CanonicalHash() (string, error) {
	return canonicalObjectHash(request, "artifact_integrity_hash")
}

func (request CandidateRegistrationRequest) Verify() error {
	if request.SchemaVersion != RegistrationRequestSchemaVersion && request.SchemaVersion != RegistrationRequestSchemaVersionV2 {
		return fmt.Errorf("unsupported registration request schema_version %q", request.SchemaVersion)
	}
	if request.ArtifactLabel != RegistrationRequestLabel {
		return fmt.Errorf("artifact_label must be %s", RegistrationRequestLabel)
	}
	if request.QualificationVerdict != StatusQualified {
		return errors.New("registration request requires QUALIFIED verdict")
	}
	if request.RIFAuthorized {
		return errors.New("Engine registration request cannot claim RIF authorization")
	}
	if request.RequestedLifecycleStartingState != "DISCOVERY" {
		return errors.New("requested lifecycle starting state must be DISCOVERY")
	}
	if err := request.FrozenCandidate.Verify(); err != nil {
		return fmt.Errorf("frozen candidate: %w", err)
	}
	if request.SchemaVersion == RegistrationRequestSchemaVersionV2 {
		if request.FrozenCandidate.SchemaVersion != FrozenDescriptorSchemaVersionV2 || request.ResearchIdentity.SchemaVersion != ResearchIdentitySchemaVersionV2 {
			return errors.New("V2 registration requires V2 frozen candidate and research identity")
		}
	} else if request.FrozenCandidate.SchemaVersion != FrozenDescriptorSchemaVersion || request.ResearchIdentity.SchemaVersion != ResearchIdentitySchemaVersion {
		return errors.New("V1 registration cannot carry V2 frozen identity")
	}
	identity := request.CandidateImplementationIdentity
	if identity.CandidateID != request.FrozenCandidate.CandidateID || identity.CandidateVersion != request.FrozenCandidate.CandidateVersion || identity.ImplementationHash != request.FrozenCandidate.ImplementationHash || identity.ConfigurationHash != request.FrozenCandidate.ConfigurationHash || identity.ParameterHash != request.FrozenCandidate.ParameterHash || identity.CapabilityHash != request.FrozenCandidate.CapabilityHash {
		return errors.New("candidate implementation identity does not match frozen descriptor")
	}
	if request.QualificationReportID != request.FrozenCandidate.QualificationReportID || request.QualificationReportHash != request.FrozenCandidate.QualificationReportHash || !validSHA256(request.QualificationReportHash) {
		return errors.New("qualification report identity does not match frozen descriptor")
	}
	if err := validateResearchIdentity(request.ResearchIdentity); err != nil {
		return err
	}
	if request.ResearchIdentity.DatasetID != request.FrozenCandidate.DatasetID || request.ResearchIdentity.DatasetVersion != request.FrozenCandidate.DatasetVersion || request.ResearchIdentity.ManifestID != request.FrozenCandidate.ManifestID || request.ResearchIdentity.ManifestHash != request.FrozenCandidate.ManifestHash || request.ResearchIdentity.CoveragePolicyVersion != request.FrozenCandidate.CoveragePolicyVersion || request.ResearchIdentity.AvailabilityPolicyVersion != request.FrozenCandidate.AvailabilityPolicyVersion || request.ResearchIdentity.IndependencePolicyHash != request.FrozenCandidate.IndependencePolicyHash || request.ResearchIdentity.UncertaintyMethodHash != request.FrozenCandidate.UncertaintyMethodHash || request.ResearchIdentity.SourceSchemaHash != request.FrozenCandidate.SourceSchemaHash || request.ResearchIdentity.ManifestContractHash != request.FrozenCandidate.ManifestContractHash || !request.ResearchIdentity.ResearchWindowStart.Equal(request.FrozenCandidate.ResearchWindowStart) || !request.ResearchIdentity.ResearchWindowEnd.Equal(request.FrozenCandidate.ResearchWindowEnd) || !request.ResearchIdentity.EvaluationCutoff.Equal(request.FrozenCandidate.EvaluationCutoff) {
		return errors.New("research identity does not match frozen descriptor")
	}
	if !validSHA256(request.ArtifactIntegrityHash) {
		return errors.New("artifact_integrity_hash is invalid")
	}
	want, err := request.CanonicalHash()
	if err != nil {
		return err
	}
	if request.ArtifactIntegrityHash != want {
		return errors.New("artifact_integrity_hash mismatch")
	}
	return nil
}

func validateResearchIdentity(identity ResearchIdentity) error {
	if identity.SchemaVersion != ResearchIdentitySchemaVersion && identity.SchemaVersion != ResearchIdentitySchemaVersionV2 {
		return fmt.Errorf("unsupported research identity schema_version %q", identity.SchemaVersion)
	}
	for name, value := range map[string]string{
		"dataset_id": identity.DatasetID, "dataset_version": identity.DatasetVersion,
		"manifest_id": identity.ManifestID, "availability_policy_version": identity.AvailabilityPolicyVersion,
		"coverage_policy_version": identity.CoveragePolicyVersion,
	} {
		if !validIdentityToken(value, name != "dataset_id") {
			return fmt.Errorf("%s is missing, mutable, or path-derived", name)
		}
	}
	if !validSHA256(identity.ManifestHash) {
		return errors.New("manifest_hash is invalid")
	}
	if err := validateGovernanceHashes(identity.SchemaVersion == ResearchIdentitySchemaVersionV2, map[string]string{
		"independence_policy_hash": identity.IndependencePolicyHash, "uncertainty_method_hash": identity.UncertaintyMethodHash,
		"source_schema_hash": identity.SourceSchemaHash, "manifest_contract_hash": identity.ManifestContractHash,
	}); err != nil {
		return err
	}
	if identity.ResearchWindowStart.IsZero() || !identity.ResearchWindowStart.Before(identity.ResearchWindowEnd) || identity.EvaluationCutoff.Before(identity.ResearchWindowEnd) {
		return errors.New("research identity windows are invalid")
	}
	return nil
}

func validateGovernanceHashes(required bool, hashes map[string]string) error {
	for name, value := range hashes {
		if required {
			if !validSHA256(value) {
				return fmt.Errorf("%s is required and must be a lowercase SHA-256 digest", name)
			}
		} else if value != "" {
			return fmt.Errorf("%s requires the V2 identity schema", name)
		}
	}
	return nil
}

func canonicalObjectHash(value any, excludedField string) (string, error) {
	data, err := json.Marshal(value)
	if err != nil {
		return "", err
	}
	var object map[string]any
	if err := json.Unmarshal(data, &object); err != nil {
		return "", err
	}
	delete(object, excludedField)
	canonical, err := json.Marshal(object)
	if err != nil {
		return "", err
	}
	digest := sha256.Sum256(canonical)
	return "sha256:" + hex.EncodeToString(digest[:]), nil
}

func validSHA256(value string) bool {
	if len(value) != len("sha256:")+sha256.Size*2 || !strings.HasPrefix(value, "sha256:") || strings.ToLower(value) != value {
		return false
	}
	_, err := hex.DecodeString(strings.TrimPrefix(value, "sha256:"))
	return err == nil
}

func validIdentityToken(value string, rejectMutable bool) bool {
	if len(value) == 0 || len(value) > 256 || strings.TrimSpace(value) != value || filepath.IsAbs(value) || filepath.VolumeName(value) != "" || strings.ContainsAny(value, `/\\`) {
		return false
	}
	if rejectMutable {
		switch strings.ToLower(value) {
		case "latest", "current", "production", "default":
			return false
		}
	}
	for _, char := range value {
		if unicode.IsControl(char) || unicode.IsSpace(char) {
			return false
		}
	}
	return true
}

func equalStrings(left, right []string) bool {
	if len(left) != len(right) {
		return false
	}
	for i := range left {
		if left[i] != right[i] {
			return false
		}
	}
	return true
}

func sortedUniqueStrings(values []string) ([]string, error) {
	result := append([]string{}, values...)
	sort.Strings(result)
	for i, value := range result {
		if strings.TrimSpace(value) == "" {
			return nil, errors.New("entries must be nonempty")
		}
		if i > 0 && value == result[i-1] {
			return nil, fmt.Errorf("duplicate entry %q", value)
		}
	}
	return result, nil
}

func allFinite(values ...float64) bool {
	for _, value := range values {
		if math.IsNaN(value) || math.IsInf(value, 0) {
			return false
		}
	}
	return true
}
