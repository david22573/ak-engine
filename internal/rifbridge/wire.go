package rifbridge

import (
	"errors"
	"fmt"
	"math"
	"sort"

	"github.com/david22573/ak-engine/internal/atomicfile"
	"github.com/david22573/ak-engine/internal/canonicalcontract"
	"github.com/david22573/ak-engine/internal/researchidentity"
)

const (
	LocalResearchDiagnosticsSchemaName    = "ak.engine.research_diagnostic"
	LocalResearchDiagnosticsSchemaVersion = 1
	LocalResearchDiagnosticsArtifactRole  = "research_diagnostic"
	ResearchEvidenceSchemaName            = "ak.engine.research_evidence"
	ResearchEvidenceArtifactRole          = "research_evidence"
	MetricResultsSchemaName               = "ak.engine.metric_inputs_and_results"
	MetricResultsArtifactRole             = "metric_inputs_and_results"
	AuthorityStatusNoneResearchOnly       = "NONE_RESEARCH_ONLY"
)

type ArtifactDisposition string

const (
	ArtifactEmitted    ArtifactDisposition = "EMITTED"
	ArtifactSuppressed ArtifactDisposition = "SUPPRESSED"
)

type ResearchDiagnosticsFailure string

const (
	DiagnosticsFailureNone               ResearchDiagnosticsFailure = "NONE"
	DiagnosticsFailureInvalidInput       ResearchDiagnosticsFailure = "INVALID_RESEARCH_INPUT"
	DiagnosticsFailureIdentityDerivation ResearchDiagnosticsFailure = "IDENTITY_DERIVATION_FAILED"
	DiagnosticsFailureSerialization      ResearchDiagnosticsFailure = "SERIALIZATION_FAILURE"
	DiagnosticsFailurePersistence        ResearchDiagnosticsFailure = "PERSISTENCE_FAILURE"
)

type LocalIntegrityStatus string

const (
	LocalIntegrityPassed LocalIntegrityStatus = "PASSED"
	LocalIntegrityFailed LocalIntegrityStatus = "FAILED"
)

const (
	ResearchStatusValidatedResearchLead = "validated_research_lead"
	ResearchStatusFragile               = "fragile"
	ResearchStatusNeedsMoreData         = "needs_more_data"
	ResearchStatusRejected              = "rejected"
)

var (
	ErrInvalidResearchInput             = errors.New("invalid research diagnostics input")
	ErrResearchDiagnosticsSerialization = errors.New("research diagnostics serialization failure")
	ErrResearchDiagnosticsPersistence   = errors.New("research diagnostics persistence failure")
)

// ResearchAssessment contains only Engine-local research facts and the raw
// derivation request. It intentionally has no lifecycle, readiness,
// authorization, caller-validated boolean, or caller-supplied identity hash.
type ResearchAssessment struct {
	Stem                      string
	Classification            string
	ClassificationGates       []ClassificationGate
	ExecutionSeriesGeneration string
	IdentityRequest           researchidentity.DerivationRequest
}

// ClassificationGate is the minimal fact set needed for the bridge to verify
// that a caller's classification is the deterministic result of its gates.
type ClassificationGate struct {
	Name     string
	Passed   bool
	Critical bool
}

// ResearchDiagnosticsResult reports only local artifact and integrity outcomes.
// It does not represent acceptance by RIF or any lifecycle decision.
type ResearchDiagnosticsResult struct {
	ArtifactDisposition ArtifactDisposition
	ArtifactPath        string
	Failure             ResearchDiagnosticsFailure
	IdentityStatus      researchidentity.IdentityStatus
	IdentityFindings    []researchidentity.Finding
	EligibleForReview   bool
	LocalIntegrity      LocalIntegrityStatus
	BlockingFindings    int
	NonBlockingWarnings int
}

// LocalResearchDiagnostics is the final Engine-local, non-authoritative
// diagnostic contract. ResearchEvidence is present only for a complete bound
// identity and reproducible metric result.
type LocalResearchDiagnostics struct {
	Contract             canonicalcontract.ContractHeader `json:"contract"`
	ArtifactHash         string                           `json:"artifact_hash"`
	AuthorityStatus      string                           `json:"authority_status"`
	IdentityStatus       researchidentity.IdentityStatus  `json:"identity_status"`
	EligibleForRIFReview bool                             `json:"eligible_for_rif_review"`
	CandidateResult      ResearchCandidateResult          `json:"candidate_result"`
	IdentityFindings     []researchidentity.Finding       `json:"identity_findings"`
	LocalIntegrity       LocalIntegrityStatus             `json:"local_integrity_status"`
	BlockingFindings     []researchidentity.Finding       `json:"blocking_findings"`
	NonBlockingWarnings  []researchidentity.Finding       `json:"non_blocking_warnings"`
	ResearchEvidence     *ResearchEvidence                `json:"research_evidence,omitempty"`
}

type ResearchCandidateResult struct {
	CandidateID      string `json:"candidate_id,omitempty"`
	CandidateVersion string `json:"candidate_version,omitempty"`
	Classification   string `json:"classification"`
	ObservationCount int    `json:"observation_count"`
}

type CanonicalMetric struct {
	MetricID    string `json:"metric_id"`
	Value       string `json:"value"`
	Unit        string `json:"unit"`
	SampleCount int    `json:"sample_count"`
}

type MetricResults struct {
	Contract             canonicalcontract.ContractHeader `json:"contract"`
	ArtifactHash         string                           `json:"artifact_hash"`
	EvaluationSeriesHash string                           `json:"evaluation_series_hash"`
	MetricPolicyID       string                           `json:"metric_policy_id"`
	MetricPolicyVersion  string                           `json:"metric_policy_version"`
	RiskFreeRate         string                           `json:"risk_free_rate"`
	PeriodsPerYear       string                           `json:"periods_per_year"`
	Metrics              []CanonicalMetric                `json:"metrics"`
}

type ResearchEvidence struct {
	Contract             canonicalcontract.ContractHeader       `json:"contract"`
	ArtifactHash         string                                 `json:"artifact_hash"`
	EvidenceID           string                                 `json:"evidence_id"`
	CreatedAtUTC         string                                 `json:"created_at_utc"`
	ProducerVersion      string                                 `json:"producer_version"`
	AuthorityStatus      string                                 `json:"authority_status"`
	Classification       string                                 `json:"classification"`
	EligibleForRIFReview bool                                   `json:"eligible_for_rif_review"`
	ResearchIdentity     researchidentity.BoundResearchIdentity `json:"research_identity"`
	MetricResults        MetricResults                          `json:"metric_results"`
	BlockingFindings     []researchidentity.Finding             `json:"blocking_findings"`
	NonBlockingWarnings  []researchidentity.Finding             `json:"non_blocking_warnings"`
}

type MetricsReport struct {
	SharpeRatio      float64 `json:"sharpe_ratio"`
	SortinoRatio     float64 `json:"sortino_ratio"`
	CalmarRatio      float64 `json:"calmar_ratio"`
	OmegaRatio       float64 `json:"omega_ratio"`
	UlcerIndex       float64 `json:"ulcer_index"`
	TailRatio        float64 `json:"tail_ratio"`
	RecoveryFactor   float64 `json:"recovery_factor"`
	MaxDrawdown      float64 `json:"max_drawdown"`
	Skewness         float64 `json:"skewness"`
	Kurtosis         float64 `json:"kurtosis"`
	AnnualizedReturn float64 `json:"annualized_return"`
	AnnualizedVol    float64 `json:"annualized_volatility"`
}

func contractArtifactHash(schemaName, role string, value any) (string, error) {
	hash, _, err := canonicalcontract.HashArtifactValue(schemaName, LocalResearchDiagnosticsSchemaVersion, role, "artifact_hash", value)
	return hash, err
}

func buildMetricResults(identity researchidentity.BoundResearchIdentity, report MetricsReport, riskFreeRate, periodsPerYear float64, sampleCount int) (MetricResults, error) {
	riskFree, err := canonicalcontract.FormatDecimal(riskFreeRate, 18)
	if err != nil {
		return MetricResults{}, err
	}
	periods, err := canonicalcontract.FormatDecimal(periodsPerYear, 8)
	if err != nil {
		return MetricResults{}, err
	}
	specs := []struct {
		id    string
		value float64
		unit  string
	}{
		{"annualized_return", report.AnnualizedReturn, "fraction"},
		{"annualized_volatility", report.AnnualizedVol, "fraction"},
		{"calmar_ratio", report.CalmarRatio, "ratio"},
		{"kurtosis", report.Kurtosis, "ratio"},
		{"max_drawdown", report.MaxDrawdown, "fraction"},
		{"omega_ratio", report.OmegaRatio, "ratio"},
		{"recovery_factor", report.RecoveryFactor, "ratio"},
		{"sharpe_ratio", report.SharpeRatio, "ratio"},
		{"skewness", report.Skewness, "ratio"},
		{"sortino_ratio", report.SortinoRatio, "ratio"},
		{"tail_ratio", report.TailRatio, "ratio"},
		{"ulcer_index", report.UlcerIndex, "fraction"},
	}
	metrics := make([]CanonicalMetric, len(specs))
	for index, spec := range specs {
		value, err := canonicalcontract.FormatDecimal(spec.value, 8)
		if err != nil {
			return MetricResults{}, fmt.Errorf("metric %s: %w", spec.id, err)
		}
		metrics[index] = CanonicalMetric{MetricID: spec.id, Value: value, Unit: spec.unit, SampleCount: sampleCount}
	}
	result := MetricResults{
		Contract:             canonicalcontract.NewHeader(MetricResultsSchemaName, LocalResearchDiagnosticsSchemaVersion, MetricResultsArtifactRole),
		EvaluationSeriesHash: identity.Series.ArtifactHash,
		MetricPolicyID:       "ak.engine.research_metrics",
		MetricPolicyVersion:  "1",
		RiskFreeRate:         riskFree,
		PeriodsPerYear:       periods,
		Metrics:              metrics,
	}
	result.ArtifactHash, err = contractArtifactHash(MetricResultsSchemaName, MetricResultsArtifactRole, result)
	return result, err
}

func buildResearchEvidence(identity researchidentity.BoundResearchIdentity, metrics MetricResults, classification string, eligible bool, blocking, warnings []researchidentity.Finding) (ResearchEvidence, error) {
	result := ResearchEvidence{
		Contract:             canonicalcontract.NewHeader(ResearchEvidenceSchemaName, LocalResearchDiagnosticsSchemaVersion, ResearchEvidenceArtifactRole),
		EvidenceID:           identity.ArtifactHash,
		CreatedAtUTC:         identity.PIT.EvaluationCutoffUTC,
		ProducerVersion:      identity.EngineSource.BuildVersion,
		AuthorityStatus:      AuthorityStatusNoneResearchOnly,
		Classification:       classification,
		EligibleForRIFReview: eligible,
		ResearchIdentity:     identity,
		MetricResults:        metrics,
		BlockingFindings:     append([]researchidentity.Finding{}, blocking...),
		NonBlockingWarnings:  append([]researchidentity.Finding{}, warnings...),
	}
	var err error
	result.ArtifactHash, err = contractArtifactHash(ResearchEvidenceSchemaName, ResearchEvidenceArtifactRole, result)
	return result, err
}

func writeResearchDiagnosticsFile(path string, value LocalResearchDiagnostics) error {
	data, err := canonicalcontract.CanonicalizeValue(value)
	if err != nil {
		return fmt.Errorf("%w: %v", ErrResearchDiagnosticsSerialization, err)
	}
	if _, err := canonicalcontract.ValidateArtifact(data, true); err != nil {
		return fmt.Errorf("%w: %v", ErrResearchDiagnosticsSerialization, err)
	}
	if err := atomicfile.WriteFile(path, data, 0644); err != nil {
		return fmt.Errorf("%w: %v", ErrResearchDiagnosticsPersistence, err)
	}
	return nil
}

func checkSampleSize(numObservations int, minRequired int) error {
	if numObservations < minRequired {
		return fmt.Errorf("insufficient sample size: expected >= %d, got %d", minRequired, numObservations)
	}
	return nil
}

func checkLookAheadBias(timestamps []int64) error {
	if len(timestamps) == 0 {
		return errors.New("timestamp series is empty")
	}
	for i := 1; i < len(timestamps); i++ {
		if timestamps[i] <= timestamps[i-1] {
			return fmt.Errorf("timestamp at index %d (%d) is not strictly after previous (%d)", i, timestamps[i], timestamps[i-1])
		}
	}
	return nil
}

func checkParameterMining(numObservations, numParameters int, minimumRatio float64) error {
	if numParameters > 0 && float64(numObservations)/float64(numParameters) < minimumRatio {
		return fmt.Errorf("observations/parameter ratio is %.2f, expected >= %.2f", float64(numObservations)/float64(numParameters), minimumRatio)
	}
	return nil
}

func calculateMetrics(returns []float64, riskFreeRate float64, periodsPerYear float64) (MetricsReport, error) {
	if len(returns) == 0 {
		return MetricsReport{}, errors.New("empty returns data")
	}
	if math.IsNaN(riskFreeRate) || math.IsInf(riskFreeRate, 0) || math.IsNaN(periodsPerYear) || math.IsInf(periodsPerYear, 0) || periodsPerYear <= 0 {
		return MetricsReport{}, errors.New("invalid metric configuration")
	}
	for i, value := range returns {
		if math.IsNaN(value) || math.IsInf(value, 0) {
			return MetricsReport{}, fmt.Errorf("return %d is non-finite", i)
		}
	}

	report := MetricsReport{}
	n := float64(len(returns))
	var sum, sumSq, sumCubed, sumQuad float64
	for _, value := range returns {
		sum += value
	}
	mean := sum / n
	for _, value := range returns {
		deviation := value - mean
		sumSq += deviation * deviation
		sumCubed += deviation * deviation * deviation
		sumQuad += deviation * deviation * deviation * deviation
	}
	variance := sumSq / n
	stdDev := math.Sqrt(variance)
	report.AnnualizedReturn = mean * periodsPerYear
	report.AnnualizedVol = stdDev * math.Sqrt(periodsPerYear)
	if stdDev > 0 {
		report.SharpeRatio = (report.AnnualizedReturn - riskFreeRate) / report.AnnualizedVol
		report.Skewness = (sumCubed / n) / math.Pow(stdDev, 3)
		report.Kurtosis = (sumQuad/n)/math.Pow(stdDev, 4) - 3
	}

	var downsideSumSq, downsideCount float64
	for _, value := range returns {
		if value < 0 {
			downsideSumSq += value * value
			downsideCount++
		}
	}
	if downsideCount > 0 {
		downsideDev := math.Sqrt(downsideSumSq / n)
		annualizedDownsideVol := downsideDev * math.Sqrt(periodsPerYear)
		if annualizedDownsideVol > 0 {
			report.SortinoRatio = (report.AnnualizedReturn - riskFreeRate) / annualizedDownsideVol
		}
	}

	maxPeak, currentComp := 1.0, 1.0
	var maxDrawdown, ulcerSum float64
	for _, value := range returns {
		currentComp *= 1 + value
		if currentComp > maxPeak {
			maxPeak = currentComp
		}
		if maxPeak != 0 {
			drawdown := (maxPeak - currentComp) / maxPeak
			if drawdown > maxDrawdown {
				maxDrawdown = drawdown
			}
			ulcerSum += drawdown * drawdown
		}
	}
	report.MaxDrawdown = maxDrawdown
	report.UlcerIndex = math.Sqrt(ulcerSum / n)
	if maxDrawdown > 0 {
		report.CalmarRatio = report.AnnualizedReturn / maxDrawdown
		report.RecoveryFactor = (currentComp - 1) / maxDrawdown
	}

	var winSum, lossSum float64
	for _, value := range returns {
		if value > 0 {
			winSum += value
		} else {
			lossSum -= value
		}
	}
	if lossSum > 0 {
		report.OmegaRatio = winSum / lossSum
	}

	sortedReturns := append([]float64(nil), returns...)
	sort.Float64s(sortedReturns)
	if len(sortedReturns) >= 20 {
		p95Index := int(float64(len(sortedReturns)) * 0.95)
		p5Index := int(float64(len(sortedReturns)) * 0.05)
		if p5Index >= 0 && p95Index < len(sortedReturns) && sortedReturns[p5Index] != 0 {
			report.TailRatio = math.Abs(sortedReturns[p95Index]) / math.Abs(sortedReturns[p5Index])
		}
	}
	for name, value := range map[string]float64{
		"sharpe": report.SharpeRatio, "sortino": report.SortinoRatio, "calmar": report.CalmarRatio,
		"omega": report.OmegaRatio, "ulcer": report.UlcerIndex, "tail": report.TailRatio,
		"recovery": report.RecoveryFactor, "max_drawdown": report.MaxDrawdown,
		"skewness": report.Skewness, "kurtosis": report.Kurtosis,
		"annualized_return": report.AnnualizedReturn, "annualized_volatility": report.AnnualizedVol,
	} {
		if math.IsNaN(value) || math.IsInf(value, 0) {
			return MetricsReport{}, fmt.Errorf("metric %s is non-finite", name)
		}
	}
	return report, nil
}
