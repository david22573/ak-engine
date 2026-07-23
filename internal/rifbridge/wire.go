package rifbridge

import (
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"os"
	"sort"
)

// ResearchLock is the engine's wire schema for RIF research.lock artifacts.
type ResearchLock struct {
	GitSHA                                   string         `json:"git_sha"`
	EngineGitSHA                             string         `json:"engine_git_sha,omitempty"`
	DatasetID                                string         `json:"dataset_id,omitempty"`
	DatasetHash                              string         `json:"dataset_hash,omitempty"`
	DatasetManifestHash                      string         `json:"dataset_manifest_hash,omitempty"`
	SourceGitSHA                             string         `json:"source_git_sha,omitempty"`
	DatasetHashes                            []string       `json:"dataset_hashes,omitempty"`
	FeatureHashes                            []string       `json:"feature_hashes,omitempty"`
	FeatureHash                              string         `json:"feature_hash,omitempty"`
	CandidateHash                            string         `json:"candidate_hash"`
	ParameterHash                            string         `json:"parameter_hash,omitempty"`
	ConfigurationHash                        string         `json:"configuration_hash"`
	RIFVersion                               string         `json:"rif_version,omitempty"`
	CoverageStatus                           string         `json:"coverage_status,omitempty"`
	UniverseID                               string         `json:"universe_id,omitempty"`
	UniverseHash                             string         `json:"universe_hash,omitempty"`
	UniverseManifestHash                     string         `json:"universe_manifest_hash,omitempty"`
	UniversePolicy                           string         `json:"universe_policy,omitempty"`
	SurvivorshipBiasRisk                     string         `json:"survivorship_bias_risk,omitempty"`
	LifecycleID                              string         `json:"lifecycle_id,omitempty"`
	LifecycleHash                            string         `json:"lifecycle_hash,omitempty"`
	LifecycleManifestHash                    string         `json:"lifecycle_manifest_hash,omitempty"`
	LifecycleEvidenceLevelSummary            map[string]int `json:"lifecycle_evidence_level_summary,omitempty"`
	ExchangeMetadataSnapshotHash             string         `json:"exchange_metadata_snapshot_hash,omitempty"`
	ExchangeMetadataSnapshotManifestHash     string         `json:"exchange_metadata_snapshot_manifest_hash,omitempty"`
	ExchangeMetadataSnapshotArchiveHash      string         `json:"exchange_metadata_snapshot_archive_hash,omitempty"`
	ExchangeMetadataSnapshotCoverageStartUTC string         `json:"exchange_metadata_snapshot_coverage_start_utc,omitempty"`
	ExchangeMetadataSnapshotCoverageEndUTC   string         `json:"exchange_metadata_snapshot_coverage_end_utc,omitempty"`
	ExchangeMetadataSnapshotEvidenceLevel    string         `json:"exchange_metadata_snapshot_evidence_level,omitempty"`
	ExchangeMetadataSnapshotCurrentOnly      bool           `json:"exchange_metadata_snapshot_current_only,omitempty"`
	PointInTimeCoverageStatus                string         `json:"point_in_time_coverage_status,omitempty"`
	PointInTimeCoverageHash                  string         `json:"point_in_time_coverage_hash,omitempty"`
	PointInTimePromotionRecommendation       string         `json:"point_in_time_promotion_recommendation,omitempty"`
}

type ResearchAudit struct {
	DataSources          []string           `json:"data_sources"`
	ExcludedData         []string           `json:"excluded_data"`
	Filters              []string           `json:"filters"`
	RegimeDefinitions    []string           `json:"regime_definitions"`
	FeatureVersions      map[string]string  `json:"feature_versions"`
	SlippageModel        string             `json:"slippage_model"`
	CommissionModel      string             `json:"commission_model"`
	ExecutionAssumptions string             `json:"execution_assumptions"`
	DatasetProvenance    *DatasetProvenance `json:"dataset_provenance,omitempty"`
	Warnings             []RIFWarning       `json:"warnings,omitempty"`
}

type RIFWarning struct {
	Code             string `json:"code"`
	Severity         string `json:"severity"`
	Reason           string `json:"reason"`
	AffectedUniverse string `json:"affected_universe,omitempty"`
	AffectedDataset  string `json:"affected_dataset,omitempty"`
	AffectedSymbol   string `json:"affected_symbol,omitempty"`
	BlocksPromotion  bool   `json:"blocks_promotion"`
	RecommendedFix   string `json:"recommended_fix"`
}

type DatasetProvenance struct {
	ManifestPath                             string         `json:"manifest_path,omitempty"`
	ValidationStatus                         string         `json:"validation_status,omitempty"`
	SurvivorshipWarning                      string         `json:"survivorship_warning,omitempty"`
	MissingMetadata                          []string       `json:"missing_metadata,omitempty"`
	SourceRepositorySHA                      string         `json:"source_repository_sha,omitempty"`
	ManifestHash                             string         `json:"manifest_hash,omitempty"`
	FileCount                                int            `json:"file_count,omitempty"`
	RowCount                                 *int64         `json:"row_count,omitempty"`
	CoverageStartTimestamp                   string         `json:"coverage_start_timestamp,omitempty"`
	CoverageEndTimestamp                     string         `json:"coverage_end_timestamp,omitempty"`
	CoverageStatus                           string         `json:"coverage_status,omitempty"`
	CoveragePct                              *float64       `json:"coverage_pct,omitempty"`
	GapWarnings                              int            `json:"gap_warnings,omitempty"`
	DuplicateWarnings                        int            `json:"duplicate_warnings,omitempty"`
	OutOfOrderWarnings                       int            `json:"out_of_order_warnings,omitempty"`
	MissingRowCount                          *int64         `json:"missing_row_count,omitempty"`
	UniversePolicy                           string         `json:"universe_policy,omitempty"`
	SurvivorshipBiasRisk                     string         `json:"survivorship_bias_risk,omitempty"`
	IncludesDelistedAssets                   string         `json:"includes_delisted_assets,omitempty"`
	UniverseWarnings                         []string       `json:"universe_warnings,omitempty"`
	LifecycleID                              string         `json:"lifecycle_id,omitempty"`
	LifecycleHash                            string         `json:"lifecycle_hash,omitempty"`
	LifecycleManifestHash                    string         `json:"lifecycle_manifest_hash,omitempty"`
	LifecycleEvidenceLevelSummary            map[string]int `json:"lifecycle_evidence_level_summary,omitempty"`
	LifecycleWarnings                        []string       `json:"lifecycle_warnings,omitempty"`
	ListingEvidenceStatus                    string         `json:"listing_evidence_status,omitempty"`
	DelistingEvidenceStatus                  string         `json:"delisting_evidence_status,omitempty"`
	SurvivorshipSupportStatus                string         `json:"survivorship_support_status,omitempty"`
	ExchangeMetadataSnapshotHash             string         `json:"exchange_metadata_snapshot_hash,omitempty"`
	ExchangeMetadataSnapshotManifestHash     string         `json:"exchange_metadata_snapshot_manifest_hash,omitempty"`
	ExchangeMetadataSnapshotArchiveHash      string         `json:"exchange_metadata_snapshot_archive_hash,omitempty"`
	ExchangeMetadataSnapshotCoverageStartUTC string         `json:"exchange_metadata_snapshot_coverage_start_utc,omitempty"`
	ExchangeMetadataSnapshotCoverageEndUTC   string         `json:"exchange_metadata_snapshot_coverage_end_utc,omitempty"`
	ExchangeMetadataSnapshotEvidenceLevel    string         `json:"exchange_metadata_snapshot_evidence_level,omitempty"`
	ExchangeMetadataSnapshotCurrentOnly      bool           `json:"exchange_metadata_snapshot_current_only,omitempty"`
	PointInTimeCoverageStatus                string         `json:"point_in_time_coverage_status,omitempty"`
	PointInTimeCoverageHash                  string         `json:"point_in_time_coverage_hash,omitempty"`
	PointInTimePromotionRecommendation       string         `json:"point_in_time_promotion_recommendation,omitempty"`
	RIFWarnings                              []RIFWarning   `json:"rif_warnings,omitempty"`
}

type PromotionPacket struct {
	CandidateID                   string        `json:"candidate_id"`
	CandidateVersion              string        `json:"candidate_version"`
	ResearchAudit                 ResearchAudit `json:"research_audit"`
	ResearchLock                  ResearchLock  `json:"research_lock"`
	RiskMetrics                   MetricsReport `json:"risk_metrics"`
	PassedIntegrityChecks         bool          `json:"passed_integrity_checks"`
	FragilityScore                float64       `json:"fragility_score"`
	PointInTimeEligibilitySummary string        `json:"point_in_time_eligibility_summary,omitempty"`
	StrictPromotionAllowed        bool          `json:"strict_promotion_allowed"`
}

type MetricsReport struct {
	SharpeRatio      float64
	SortinoRatio     float64
	CalmarRatio      float64
	OmegaRatio       float64
	UlcerIndex       float64
	TailRatio        float64
	RecoveryFactor   float64
	MaxDrawdown      float64
	Skewness         float64
	Kurtosis         float64
	AnnualizedReturn float64
	AnnualizedVol    float64
}

func writeJSONFile(path string, value any) error {
	data, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0644)
}

func verifyLock(current, stored ResearchLock) error {
	if current.GitSHA != stored.GitSHA {
		return fmt.Errorf("git SHA mismatch: current %s, stored %s", current.GitSHA, stored.GitSHA)
	}
	if current.CandidateHash != stored.CandidateHash {
		return fmt.Errorf("candidate hash mismatch: current %s, stored %s", current.CandidateHash, stored.CandidateHash)
	}
	if current.ConfigurationHash != stored.ConfigurationHash {
		return fmt.Errorf("configuration hash mismatch")
	}
	if current.DatasetHash != "" && stored.DatasetHash != "" && current.DatasetHash != stored.DatasetHash {
		return fmt.Errorf("dataset hash mismatch: current %s, stored %s", current.DatasetHash, stored.DatasetHash)
	}
	if current.DatasetManifestHash != "" && stored.DatasetManifestHash != "" && current.DatasetManifestHash != stored.DatasetManifestHash {
		return fmt.Errorf("manifest hash mismatch: current %s, stored %s", current.DatasetManifestHash, stored.DatasetManifestHash)
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
	for i := 1; i < len(timestamps); i++ {
		if timestamps[i] <= timestamps[i-1] {
			return fmt.Errorf("look-ahead bias detected: timestamp at index %d (%d) is not strictly after previous (%d)", i, timestamps[i], timestamps[i-1])
		}
	}
	return nil
}

func checkParameterMining(numObservations, numParameters int) error {
	if numParameters > 0 && float64(numObservations)/float64(numParameters) < 20.0 {
		return fmt.Errorf("risk of parameter mining: ratio is %.2f, expected >= 20.0", float64(numObservations)/float64(numParameters))
	}
	return nil
}

func calculateMetrics(returns []float64, riskFreeRate float64, periodsPerYear float64) (MetricsReport, error) {
	if len(returns) == 0 {
		return MetricsReport{}, errors.New("empty returns data")
	}

	report := MetricsReport{}
	n := float64(len(returns))

	var sum, sumSq, sumCubed, sumQuad float64
	for _, r := range returns {
		sum += r
	}
	mean := sum / n

	for _, r := range returns {
		dev := r - mean
		sumSq += dev * dev
		sumCubed += dev * dev * dev
		sumQuad += dev * dev * dev * dev
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
	for _, r := range returns {
		if r < 0 {
			downsideSumSq += r * r
			downsideCount++
		}
	}
	if downsideCount > 0 {
		downsideDev := math.Sqrt(downsideSumSq / n)
		annualizedDownsideVol := downsideDev * math.Sqrt(periodsPerYear)
		report.SortinoRatio = (report.AnnualizedReturn - riskFreeRate) / annualizedDownsideVol
	}

	maxPeak := 1.0
	currentComp := 1.0
	var maxDrawdown, ulcerSum float64
	for _, r := range returns {
		currentComp *= 1 + r
		if currentComp > maxPeak {
			maxPeak = currentComp
		}
		dd := (maxPeak - currentComp) / maxPeak
		if dd > maxDrawdown {
			maxDrawdown = dd
		}
		ulcerSum += dd * dd
	}

	report.MaxDrawdown = maxDrawdown
	report.UlcerIndex = math.Sqrt(ulcerSum / n)
	if maxDrawdown > 0 {
		report.CalmarRatio = report.AnnualizedReturn / maxDrawdown
		report.RecoveryFactor = (currentComp - 1) / maxDrawdown
	}

	var winSum, lossSum float64
	for _, r := range returns {
		if r > 0 {
			winSum += r
		} else {
			lossSum -= r
		}
	}
	if lossSum > 0 {
		report.OmegaRatio = winSum / lossSum
	}

	sortedReturns := make([]float64, len(returns))
	copy(sortedReturns, returns)
	sort.Float64s(sortedReturns)
	if len(sortedReturns) >= 20 {
		p95Idx := int(float64(len(sortedReturns)) * 0.95)
		p5Idx := int(float64(len(sortedReturns)) * 0.05)
		if p5Idx >= 0 && p95Idx < len(sortedReturns) && sortedReturns[p5Idx] != 0 {
			report.TailRatio = math.Abs(sortedReturns[p95Idx]) / math.Abs(sortedReturns[p5Idx])
		}
	}

	return report, nil
}
