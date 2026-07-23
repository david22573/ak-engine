package rifbridge

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
)

// Bridge emits RIF-compatible JSON artifacts without importing ak-rif internals.
type Bridge struct {
	holdoutLimit     int
	holdoutExposures map[string]int
}

func NewBridge() *Bridge {
	return &Bridge{
		holdoutLimit:     10,
		holdoutExposures: make(map[string]int),
	}
}

// RIFOutput represents the outputs and status of RIF generation
type RIFOutput struct {
	IntegrityPassed bool
	Warnings        []string
}

// EvaluateAndEmit generates RIF artifacts for a deep candidate evaluation.
func (b *Bridge) EvaluateAndEmit(
	stem string,
	candID string,
	candVersion string,
	gitSHA string,
	datasetHashes []string,
	featureHashes []string,
	candHash string,
	configHash string,
	returns []float64,
	timestamps []int64,
	numParams int,
	numObs int,
	isPromoted bool,
	manifestPath string,
) (RIFOutput, error) {
	out := RIFOutput{
		IntegrityPassed: true,
	}

	// 1. Create the research.lock
	lock := ResearchLock{
		EngineGitSHA:      gitSHA,
		GitSHA:            gitSHA,
		DatasetHashes:     datasetHashes,
		FeatureHashes:     featureHashes,
		CandidateHash:     candHash,
		ConfigurationHash: configHash,
	}

	var prov *DatasetProvenance
	var rifWarnings []RIFWarning
	if manifestPath != "" {
		parsedProv, warnings, err := applyDatasetManifestToRIF(manifestPath, &lock)
		if err != nil {
			return out, err
		}
		prov = parsedProv
		for _, warning := range warnings {
			rifWarnings = append(rifWarnings, warning)
			out.Warnings = append(out.Warnings, formatRIFWarning(warning))
			if isPromoted && warning.BlocksPromotion {
				out.IntegrityPassed = false
			}
		}
	} else {
		out.Warnings = append(out.Warnings, "CODE: RIF_DATASET_MANIFEST_MISSING | SEVERITY: WARNING | REASON: dataset_manifest.json missing")
		warning := newRIFWarning("RIF_UNIVERSE_MANIFEST_MISSING", "WARNING", "dataset_manifest.json is missing, so no universe manifest metadata can be verified", "", "", "", true, "Generate dataset_manifest.json with ak-historian and pass it to ak-engine.")
		rifWarnings = append(rifWarnings, warning)
		out.Warnings = append(out.Warnings, formatRIFWarning(warning))
		if isPromoted {
			out.IntegrityPassed = false
			out.Warnings = append(out.Warnings, "CODE: DATASET_PROVENANCE_BLOCKED | SEVERITY: ERROR | REASON: Missing dataset provenance for promotion-grade output")
		}
	}

	lockPath := stem + ".research.lock"
	if err := writeJSONFile(lockPath, lock); err != nil {
		return out, fmt.Errorf("failed to generate lockfile: %w", err)
	}

	// 2. Create the research_audit.json
	aud := ResearchAudit{
		DataSources:          []string{"ak-engine"},
		ExcludedData:         []string{},
		Filters:              []string{"default_engine_filters"},
		RegimeDefinitions:    []string{"default_regimes"},
		FeatureVersions:      map[string]string{},
		SlippageModel:        "engine_simulated",
		CommissionModel:      "engine_simulated",
		ExecutionAssumptions: "market_maker",
		DatasetProvenance:    prov,
		Warnings:             rifWarnings,
	}

	audPath := stem + ".research_audit.json"
	if err := writeJSONFile(audPath, aud); err != nil {
		return out, fmt.Errorf("failed to generate audit: %w", err)
	}

	// 3. Run integrity checks
	if err := checkSampleSize(numObs, 30); err != nil {
		out.IntegrityPassed = false
		out.Warnings = append(out.Warnings, fmt.Sprintf("CODE: RIF_LOW_SAMPLE_SIZE | SEVERITY: ERROR | REASON: %v | AFFECTED: %s | BLOCKS PROMOTION: Yes | FIX: Increase sample period to include at least 30 observations.", err, candID))
	}
	if err := checkLookAheadBias(timestamps); err != nil {
		out.IntegrityPassed = false
		out.Warnings = append(out.Warnings, fmt.Sprintf("CODE: RIF_LOOKAHEAD_BIAS | SEVERITY: ERROR | REASON: %v | AFFECTED: %s | BLOCKS PROMOTION: Yes | FIX: Check event timestamps for proper monotonic ordering.", err, candID))
	}
	if err := checkParameterMining(numObs, numParams); err != nil {
		out.IntegrityPassed = false
		out.Warnings = append(out.Warnings, fmt.Sprintf("CODE: RIF_PARAMETER_MINING_RISK | SEVERITY: ERROR | REASON: %v | AFFECTED: %s | BLOCKS PROMOTION: Yes | FIX: Reduce parameter space or increase sample size.", err, candID))
	}

	// 4. Calculate metrics
	var riskMetrics MetricsReport
	if len(returns) > 0 {
		var err error
		riskMetrics, err = calculateMetrics(returns, 0.05, 365)
		if err != nil {
			out.Warnings = append(out.Warnings, fmt.Sprintf("MetricsCalc: %v", err))
		}
	}

	// 5. Emit promotion packet ONLY if the candidate is promoted
	// The caller will determine if it SHOULD be promoted based on out.IntegrityPassed.
	// If the caller calls this function with isPromoted=true AND out.IntegrityPassed=true,
	// we generate the packet. If it's false, the caller will downgrade it.
	// We'll trust the caller's isPromoted flag, but the caller should check IntegrityPassed.
	if isPromoted && out.IntegrityPassed {
		strictPromotionAllowed := true
		for _, w := range aud.Warnings {
			if w.BlocksPromotion {
				strictPromotionAllowed = false
				break
			}
		}

		packet := PromotionPacket{
			CandidateID:                   candID,
			CandidateVersion:              candVersion,
			ResearchAudit:                 aud,
			ResearchLock:                  lock,
			RiskMetrics:                   riskMetrics,
			PassedIntegrityChecks:         out.IntegrityPassed,
			FragilityScore:                0.0,
			PointInTimeEligibilitySummary: lock.PointInTimeCoverageStatus,
			StrictPromotionAllowed:        strictPromotionAllowed,
		}

		packetPath := stem + ".promotion_packet.json"
		if err := writeJSONFile(packetPath, packet); err != nil {
			return out, fmt.Errorf("failed to generate promotion packet: %w", err)
		}
	}

	return out, nil
}

// EmitRunFinalization generates RIF artifacts for a completed research run.
func (b *Bridge) EmitRunFinalization(
	stem string,
	gitSHA string,
	datasetHashes []string,
	featureHashes []string,
	configHash string,
	manifestPath string,
) error {
	lock := ResearchLock{
		EngineGitSHA:      gitSHA,
		GitSHA:            gitSHA,
		DatasetHashes:     datasetHashes,
		FeatureHashes:     featureHashes,
		CandidateHash:     "N/A (run-level)",
		ConfigurationHash: configHash,
	}

	var prov *DatasetProvenance
	var rifWarnings []RIFWarning
	if manifestPath != "" {
		parsedProv, warnings, err := applyDatasetManifestToRIF(manifestPath, &lock)
		if err != nil {
			return err
		}
		prov = parsedProv
		rifWarnings = append(rifWarnings, warnings...)
	}

	lockPath := stem + ".research.lock"
	if err := writeJSONFile(lockPath, lock); err != nil {
		return fmt.Errorf("failed to generate run-level lockfile: %w", err)
	}

	aud := ResearchAudit{
		DataSources:          []string{"ak-engine"},
		ExcludedData:         []string{},
		Filters:              []string{"default_engine_filters"},
		RegimeDefinitions:    []string{"default_regimes"},
		FeatureVersions:      map[string]string{},
		SlippageModel:        "engine_simulated",
		CommissionModel:      "engine_simulated",
		ExecutionAssumptions: "market_maker",
		DatasetProvenance:    prov,
		Warnings:             rifWarnings,
	}

	audPath := stem + ".research_audit.json"
	if err := writeJSONFile(audPath, aud); err != nil {
		return fmt.Errorf("failed to generate run-level audit: %w", err)
	}

	return nil
}

type datasetManifestForRIF struct {
	DatasetID       string `json:"dataset_id"`
	SourceGitSHA    string `json:"source_git_sha"`
	MinTimestampUTC string `json:"min_timestamp_utc"`
	MaxTimestampUTC string `json:"max_timestamp_utc"`
	Hashes          struct {
		DatasetHash  string `json:"dataset_hash"`
		ManifestHash string `json:"manifest_hash"`
	} `json:"hashes"`
	Validation struct {
		Status   string   `json:"status"`
		Warnings []string `json:"warnings"`
	} `json:"validation"`
	Survivorship struct {
		UniverseID                               string         `json:"universe_id"`
		UniverseHash                             string         `json:"universe_hash"`
		UniverseManifestHash                     string         `json:"universe_manifest_hash"`
		UniversePolicy                           string         `json:"universe_policy"`
		IncludesDelistedAssets                   string         `json:"includes_delisted_assets"`
		SurvivorshipBiasRisk                     string         `json:"survivorship_bias_risk"`
		LifecycleID                              string         `json:"lifecycle_id"`
		LifecycleHash                            string         `json:"lifecycle_hash"`
		LifecycleManifestHash                    string         `json:"lifecycle_manifest_hash"`
		LifecycleEvidenceLevelSummary            map[string]int `json:"lifecycle_evidence_level_summary"`
		LifecycleWarnings                        []string       `json:"lifecycle_warnings"`
		ListingEvidenceStatus                    string         `json:"listing_evidence_status"`
		DelistingEvidenceStatus                  string         `json:"delisting_evidence_status"`
		SurvivorshipSupportStatus                string         `json:"survivorship_support_status"`
		ExchangeMetadataSnapshotHash             string         `json:"exchange_metadata_snapshot_hash"`
		ExchangeMetadataSnapshotManifestHash     string         `json:"exchange_metadata_snapshot_manifest_hash"`
		ExchangeMetadataSnapshotArchiveHash      string         `json:"exchange_metadata_snapshot_archive_hash"`
		ExchangeMetadataSnapshotCoverageStartUTC string         `json:"exchange_metadata_snapshot_coverage_start_utc"`
		ExchangeMetadataSnapshotCoverageEndUTC   string         `json:"exchange_metadata_snapshot_coverage_end_utc"`
		ExchangeMetadataSnapshotEvidenceLevel    string         `json:"exchange_metadata_snapshot_evidence_level"`
		ExchangeMetadataSnapshotCurrentOnly      bool           `json:"exchange_metadata_snapshot_current_only"`
		PointInTimeCoverageStatus                string         `json:"point_in_time_coverage_status"`
		PointInTimeCoverageHash                  string         `json:"point_in_time_coverage_hash"`
		PointInTimePromotionRecommendation       string         `json:"point_in_time_promotion_recommendation"`
		WarningCode                              string         `json:"warning_code"`
		Warnings                                 []string       `json:"warnings"`
	} `json:"survivorship"`
	RowCountTotal *int64 `json:"row_count_total"`
	Files         []struct {
		Symbol string `json:"symbol"`
	} `json:"files"`
	Coverage *struct {
		Status  string `json:"status"`
		Symbols []struct {
			GapCount                 int     `json:"gap_count"`
			DuplicateTimestampCount  int     `json:"duplicate_timestamp_count"`
			OutOfOrderTimestampCount int     `json:"out_of_order_timestamp_count"`
			MissingRowCount          int64   `json:"missing_row_count"`
			CoveragePct              float64 `json:"coverage_pct"`
		} `json:"symbols"`
	} `json:"coverage"`
}

func applyDatasetManifestToRIF(manifestPath string, lock *ResearchLock) (*DatasetProvenance, []RIFWarning, error) {
	data, err := os.ReadFile(manifestPath)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to read dataset manifest: %w", err)
	}
	var m datasetManifestForRIF
	if err := json.Unmarshal(data, &m); err != nil {
		return nil, nil, fmt.Errorf("failed to parse dataset manifest: %w", err)
	}

	prov := &DatasetProvenance{
		ManifestPath:                             manifestPath,
		SourceRepositorySHA:                      m.SourceGitSHA,
		ManifestHash:                             m.Hashes.ManifestHash,
		ValidationStatus:                         m.Validation.Status,
		SurvivorshipWarning:                      m.Survivorship.WarningCode,
		CoverageStartTimestamp:                   m.MinTimestampUTC,
		CoverageEndTimestamp:                     m.MaxTimestampUTC,
		FileCount:                                len(m.Files),
		UniversePolicy:                           m.Survivorship.UniversePolicy,
		SurvivorshipBiasRisk:                     m.Survivorship.SurvivorshipBiasRisk,
		IncludesDelistedAssets:                   m.Survivorship.IncludesDelistedAssets,
		UniverseWarnings:                         append([]string{}, m.Survivorship.Warnings...),
		LifecycleID:                              m.Survivorship.LifecycleID,
		LifecycleHash:                            m.Survivorship.LifecycleHash,
		LifecycleManifestHash:                    m.Survivorship.LifecycleManifestHash,
		LifecycleEvidenceLevelSummary:            copyEvidenceSummary(m.Survivorship.LifecycleEvidenceLevelSummary),
		LifecycleWarnings:                        append([]string{}, m.Survivorship.LifecycleWarnings...),
		ListingEvidenceStatus:                    m.Survivorship.ListingEvidenceStatus,
		DelistingEvidenceStatus:                  m.Survivorship.DelistingEvidenceStatus,
		SurvivorshipSupportStatus:                m.Survivorship.SurvivorshipSupportStatus,
		ExchangeMetadataSnapshotHash:             m.Survivorship.ExchangeMetadataSnapshotHash,
		ExchangeMetadataSnapshotManifestHash:     m.Survivorship.ExchangeMetadataSnapshotManifestHash,
		ExchangeMetadataSnapshotArchiveHash:      m.Survivorship.ExchangeMetadataSnapshotArchiveHash,
		ExchangeMetadataSnapshotCoverageStartUTC: m.Survivorship.ExchangeMetadataSnapshotCoverageStartUTC,
		ExchangeMetadataSnapshotCoverageEndUTC:   m.Survivorship.ExchangeMetadataSnapshotCoverageEndUTC,
		ExchangeMetadataSnapshotEvidenceLevel:    m.Survivorship.ExchangeMetadataSnapshotEvidenceLevel,
		ExchangeMetadataSnapshotCurrentOnly:      m.Survivorship.ExchangeMetadataSnapshotCurrentOnly,
		PointInTimeCoverageStatus:                m.Survivorship.PointInTimeCoverageStatus,
		PointInTimeCoverageHash:                  m.Survivorship.PointInTimeCoverageHash,
		PointInTimePromotionRecommendation:       m.Survivorship.PointInTimePromotionRecommendation,
	}
	if m.RowCountTotal != nil {
		rowCount := *m.RowCountTotal
		prov.RowCount = &rowCount
	}

	lock.DatasetID = m.DatasetID
	lock.SourceGitSHA = m.SourceGitSHA
	lock.DatasetHash = m.Hashes.DatasetHash
	lock.DatasetManifestHash = m.Hashes.ManifestHash
	lock.UniverseID = m.Survivorship.UniverseID
	lock.UniverseHash = m.Survivorship.UniverseHash
	lock.UniverseManifestHash = m.Survivorship.UniverseManifestHash
	lock.UniversePolicy = m.Survivorship.UniversePolicy
	lock.SurvivorshipBiasRisk = m.Survivorship.SurvivorshipBiasRisk
	lock.LifecycleID = m.Survivorship.LifecycleID
	lock.LifecycleHash = m.Survivorship.LifecycleHash
	lock.LifecycleManifestHash = m.Survivorship.LifecycleManifestHash
	lock.LifecycleEvidenceLevelSummary = copyEvidenceSummary(m.Survivorship.LifecycleEvidenceLevelSummary)
	lock.ExchangeMetadataSnapshotHash = m.Survivorship.ExchangeMetadataSnapshotHash
	lock.ExchangeMetadataSnapshotManifestHash = m.Survivorship.ExchangeMetadataSnapshotManifestHash
	lock.ExchangeMetadataSnapshotArchiveHash = m.Survivorship.ExchangeMetadataSnapshotArchiveHash
	lock.ExchangeMetadataSnapshotCoverageStartUTC = m.Survivorship.ExchangeMetadataSnapshotCoverageStartUTC
	lock.ExchangeMetadataSnapshotCoverageEndUTC = m.Survivorship.ExchangeMetadataSnapshotCoverageEndUTC
	lock.ExchangeMetadataSnapshotEvidenceLevel = m.Survivorship.ExchangeMetadataSnapshotEvidenceLevel
	lock.ExchangeMetadataSnapshotCurrentOnly = m.Survivorship.ExchangeMetadataSnapshotCurrentOnly
	lock.PointInTimeCoverageStatus = m.Survivorship.PointInTimeCoverageStatus
	lock.PointInTimeCoverageHash = m.Survivorship.PointInTimeCoverageHash
	lock.PointInTimePromotionRecommendation = m.Survivorship.PointInTimePromotionRecommendation

	var warnings []RIFWarning
	addWarning := func(w RIFWarning) {
		warnings = append(warnings, w)
	}

	if m.Survivorship.UniversePolicy == "" || m.Survivorship.UniverseHash == "" || m.Survivorship.UniverseManifestHash == "" {
		addWarning(newRIFWarning(
			"RIF_UNIVERSE_MANIFEST_MISSING",
			"ERROR",
			"dataset manifest does not contain complete universe manifest hash metadata",
			m.Survivorship.UniverseID,
			m.DatasetID,
			"",
			true,
			"Generate a universe_manifest.json and pass it to ak-historian dataset-manifest with --universe-manifest.",
		))
	}

	if !isKnownUniversePolicy(m.Survivorship.UniversePolicy) || m.Survivorship.UniversePolicy == "UNKNOWN" {
		addWarning(newRIFWarning(
			"RIF_UNIVERSE_POLICY_UNKNOWN",
			"ERROR",
			"universe policy is missing or UNKNOWN",
			m.Survivorship.UniverseID,
			m.DatasetID,
			"",
			true,
			"Use EXPLICIT_SYMBOL_LIST for hand-picked research or a verified point-in-time policy when listing evidence exists.",
		))
	}

	if isNotPointInTimePolicy(m.Survivorship.UniversePolicy) {
		addWarning(newRIFWarning(
			"RIF_UNIVERSE_NOT_POINT_IN_TIME",
			"ERROR",
			fmt.Sprintf("universe policy %s is not point-in-time verified", m.Survivorship.UniversePolicy),
			m.Survivorship.UniverseID,
			m.DatasetID,
			"",
			true,
			"Use a point-in-time universe source with listing and delisting evidence before strict promotion.",
		))
	}

	if m.Survivorship.SurvivorshipBiasRisk != "" && m.Survivorship.SurvivorshipBiasRisk != "LOW" {
		addWarning(newRIFWarning(
			"RIF_SURVIVORSHIP_BIAS_RISK",
			"ERROR",
			fmt.Sprintf("survivorship bias risk is %s", m.Survivorship.SurvivorshipBiasRisk),
			m.Survivorship.UniverseID,
			m.DatasetID,
			"",
			true,
			"Use verified historical listing/delisting metadata or keep the result exploratory.",
		))
	}

	if m.Survivorship.SurvivorshipBiasRisk == "LOW" && (isNotPointInTimePolicy(m.Survivorship.UniversePolicy) || m.Survivorship.IncludesDelistedAssets != "true" || hasString(m.Survivorship.Warnings, "UNIVERSE_LOW_RISK_UNPROVEN")) {
		addWarning(newRIFWarning(
			"RIF_UNIVERSE_LOW_RISK_UNPROVEN",
			"ERROR",
			"LOW survivorship risk is not supported by point-in-time listing/delisting evidence",
			m.Survivorship.UniverseID,
			m.DatasetID,
			"",
			true,
			"Do not mark survivorship risk LOW until the source proves historical listing and delisting availability.",
		))
	}

	if m.Survivorship.LifecycleHash == "" || m.Survivorship.LifecycleManifestHash == "" {
		addWarning(newRIFWarning(
			"RIF_LIFECYCLE_MANIFEST_MISSING",
			"ERROR",
			"dataset manifest does not contain complete lifecycle manifest metadata",
			m.Survivorship.UniverseID,
			m.DatasetID,
			"",
			true,
			"Generate asset_lifecycle_manifest.json and pass it to ak-historian universe-manifest with --asset-lifecycle-manifest.",
		))
	} else {
		if m.Survivorship.ListingEvidenceStatus != "VERIFIED" {
			addWarning(newRIFWarning(
				"RIF_LIFECYCLE_LISTING_EVIDENCE_MISSING",
				"ERROR",
				fmt.Sprintf("lifecycle listing evidence status is %s", emptyAsUnknown(m.Survivorship.ListingEvidenceStatus)),
				m.Survivorship.UniverseID,
				m.DatasetID,
				"",
				true,
				"Provide verified exchange listing evidence before strict point-in-time promotion.",
			))
		}
		if m.Survivorship.DelistingEvidenceStatus != "VERIFIED" {
			addWarning(newRIFWarning(
				"RIF_LIFECYCLE_DELISTING_EVIDENCE_MISSING",
				"ERROR",
				fmt.Sprintf("lifecycle delisting evidence status is %s", emptyAsUnknown(m.Survivorship.DelistingEvidenceStatus)),
				m.Survivorship.UniverseID,
				m.DatasetID,
				"",
				true,
				"Provide verified delisting evidence or keep survivorship risk elevated.",
			))
		}
		if m.Survivorship.SurvivorshipSupportStatus != "LOW_SUPPORTED" {
			addWarning(newRIFWarning(
				"RIF_LIFECYCLE_EVIDENCE_WEAK",
				"ERROR",
				fmt.Sprintf("lifecycle survivorship support status is %s", emptyAsUnknown(m.Survivorship.SurvivorshipSupportStatus)),
				m.Survivorship.UniverseID,
				m.DatasetID,
				"",
				true,
				"Use verified lifecycle evidence before claiming low survivorship risk.",
			))
		}
		if hasWeakLifecycleEvidenceLevel(m.Survivorship.LifecycleEvidenceLevelSummary) {
			addWarning(newRIFWarning(
				"RIF_SURVIVORSHIP_NOT_SOLVED",
				"ERROR",
				"lifecycle evidence includes local-data-only, current-active-only, user-provided-unverified, or unknown evidence",
				m.Survivorship.UniverseID,
				m.DatasetID,
				"",
				true,
				"Treat this result as exploratory until historical listing and delisting evidence is verified.",
			))
		}
		if hasString(m.Survivorship.LifecycleWarnings, "LIFECYCLE_SNAPSHOT_OBSERVED_TIME_MISSING") {
			addWarning(newRIFWarning(
				"RIF_BACKFILL_OBSERVED_TIME_MISSING",
				"ERROR",
				"lifecycle evidence relies on exchange snapshots missing observed time",
				m.Survivorship.UniverseID,
				m.DatasetID,
				"",
				true,
				"Provide observed time for all backfilled exchange snapshots before strict promotion.",
			))
		}
		if hasString(m.Survivorship.LifecycleWarnings, "LIFECYCLE_SNAPSHOT_UNVERIFIED_SOURCE") || hasString(m.Survivorship.LifecycleWarnings, "LIFECYCLE_SNAPSHOT_TRUST_WEAK") {
			addWarning(newRIFWarning(
				"RIF_BACKFILL_EVIDENCE_UNVERIFIED",
				"ERROR",
				"lifecycle evidence relies on unverified exchange snapshots",
				m.Survivorship.UniverseID,
				m.DatasetID,
				"",
				true,
				"Use verified exchange metadata archives before claiming low survivorship risk.",
			))
		}
	}

	if m.Survivorship.ExchangeMetadataSnapshotHash != "" || m.Survivorship.ExchangeMetadataSnapshotManifestHash != "" {
		if m.Survivorship.ExchangeMetadataSnapshotCurrentOnly {
			addWarning(newRIFWarning(
				"RIF_EXCHANGE_SNAPSHOT_CURRENT_ONLY",
				"ERROR",
				"exchange metadata snapshot evidence is current-only and does not solve historical survivorship bias",
				m.Survivorship.UniverseID,
				m.DatasetID,
				"",
				true,
				"Backfill point-in-time exchange metadata snapshots covering the research window or keep the result exploratory.",
			))
		}
		switch m.Survivorship.PointInTimeCoverageStatus {
		case "COVERS_WINDOW":
			// Listing and delisting proof checks above still determine promotion eligibility.
		case "CURRENT_ONLY", "PARTIAL":
			addWarning(newRIFWarning(
				"RIF_POINT_IN_TIME_EVIDENCE_PARTIAL",
				"ERROR",
				fmt.Sprintf("snapshot point-in-time coverage status is %s", emptyAsUnknown(m.Survivorship.PointInTimeCoverageStatus)),
				m.Survivorship.UniverseID,
				m.DatasetID,
				"",
				true,
				"Use a snapshot archive that covers the full research window before strict promotion.",
			))
			addWarning(newRIFWarning(
				"RIF_SNAPSHOT_ARCHIVE_DOES_NOT_COVER_RESEARCH_WINDOW",
				"ERROR",
				"exchange metadata snapshot archive does not cover the full research window",
				m.Survivorship.UniverseID,
				m.DatasetID,
				"",
				true,
				"Collect or import snapshots covering the full research period.",
			))
		default:
			addWarning(newRIFWarning(
				"RIF_SNAPSHOT_ARCHIVE_DOES_NOT_COVER_RESEARCH_WINDOW",
				"ERROR",
				"exchange metadata snapshot archive coverage is unknown",
				m.Survivorship.UniverseID,
				m.DatasetID,
				"",
				true,
				"Provide snapshot archive coverage metadata.",
			))
		}
		if m.Survivorship.DelistingEvidenceStatus != "VERIFIED" {
			addWarning(newRIFWarning(
				"RIF_DELISTING_NOT_PROVEN",
				"ERROR",
				"exchange metadata snapshots do not prove delisting status for all symbols",
				m.Survivorship.UniverseID,
				m.DatasetID,
				"",
				true,
				"Use explicit delisting evidence before claiming survivorship-free research.",
			))
		}
	}

	if m.Survivorship.PointInTimeCoverageHash != "" {
		if m.Survivorship.PointInTimeCoverageStatus == "PIT_NOT_ELIGIBLE" || m.Survivorship.PointInTimeCoverageStatus == "UNKNOWN" {
			addWarning(newRIFWarning(
				"RIF_PIT_NOT_ELIGIBLE",
				"ERROR",
				fmt.Sprintf("PIT evidence coverage status is %s", m.Survivorship.PointInTimeCoverageStatus),
				m.Survivorship.UniverseID,
				m.DatasetID,
				"",
				true,
				"Strict promotion is blocked. Provide complete PIT evidence.",
			))
		} else if m.Survivorship.PointInTimeCoverageStatus == "PIT_PARTIAL" || m.Survivorship.PointInTimePromotionRecommendation == "DOWNGRADE_PROMOTION" {
			addWarning(newRIFWarning(
				"RIF_PIT_PARTIAL",
				"WARNING",
				"PIT evidence coverage is partial or downgraded",
				m.Survivorship.UniverseID,
				m.DatasetID,
				"",
				true,
				"Strict promotion is downgraded.",
			))
		} else if m.Survivorship.PointInTimePromotionRecommendation == "BLOCK_STRICT_PROMOTION" {
			addWarning(newRIFWarning(
				"RIF_PIT_PROMOTION_BLOCKED",
				"ERROR",
				"PIT evidence coverage report blocked strict promotion",
				m.Survivorship.UniverseID,
				m.DatasetID,
				"",
				true,
				"Resolve PIT blocking reasons.",
			))
		}
	}

	if m.Survivorship.SurvivorshipBiasRisk == "LOW" && m.Survivorship.SurvivorshipSupportStatus != "LOW_SUPPORTED" {
		addWarning(newRIFWarning(
			"RIF_LOW_SURVIVORSHIP_RISK_UNPROVEN",
			"ERROR",
			"LOW survivorship risk is not proven by lifecycle evidence",
			m.Survivorship.UniverseID,
			m.DatasetID,
			"",
			true,
			"Do not mark survivorship risk LOW until lifecycle evidence supports it.",
		))
	}

	for _, code := range m.Validation.Warnings {
		if isUniverseDatasetMismatch(code) {
			addWarning(newRIFWarning(
				"RIF_UNIVERSE_DATASET_MISMATCH",
				"ERROR",
				fmt.Sprintf("dataset manifest validation reported %s", code),
				m.Survivorship.UniverseID,
				m.DatasetID,
				"",
				true,
				"Regenerate the dataset manifest with matching symbols, market type, quote asset, and universe window.",
			))
		}
	}

	if m.Coverage != nil {
		lock.CoverageStatus = m.Coverage.Status
		prov.CoverageStatus = m.Coverage.Status
		if m.Coverage.Status == "FAIL" {
			addWarning(newRIFWarning("RIF_DATASET_COVERAGE_FAIL", "ERROR", "dataset coverage status is FAIL", m.Survivorship.UniverseID, m.DatasetID, "", true, "Repair missing/gapped data before strict promotion."))
		} else if m.Coverage.Status == "UNKNOWN" {
			addWarning(newRIFWarning("RIF_DATASET_COVERAGE_UNKNOWN", "ERROR", "dataset coverage status is UNKNOWN", m.Survivorship.UniverseID, m.DatasetID, "", true, "Run ak-historian dataset-manifest with coverage enabled."))
		} else if m.Coverage.Status == "WARN" {
			addWarning(newRIFWarning("RIF_DATASET_COVERAGE_WARN", "WARNING", "dataset coverage status is WARN", m.Survivorship.UniverseID, m.DatasetID, "", false, "Review coverage warnings before promotion."))
		}

		var gaps, dups, outOfOrder int
		var coveragePct float64
		var hasCoveragePct bool
		var missingRows int64
		for _, sym := range m.Coverage.Symbols {
			gaps += sym.GapCount
			dups += sym.DuplicateTimestampCount
			outOfOrder += sym.OutOfOrderTimestampCount
			missingRows += sym.MissingRowCount
			if sym.CoveragePct > 0 {
				coveragePct = sym.CoveragePct
				hasCoveragePct = true
			}
		}
		prov.GapWarnings = gaps
		prov.DuplicateWarnings = dups
		prov.OutOfOrderWarnings = outOfOrder
		if hasCoveragePct {
			prov.CoveragePct = &coveragePct
		}
		if missingRows > 0 {
			prov.MissingRowCount = &missingRows
		}
		if gaps > 0 {
			addWarning(newRIFWarning("RIF_DATASET_GAPS_DETECTED", "WARNING", "gaps detected in dataset coverage", m.Survivorship.UniverseID, m.DatasetID, "", false, "Inspect coverage gap details in research_audit.json."))
		}
		if dups > 0 {
			addWarning(newRIFWarning("RIF_DATASET_DUPLICATE_TIMESTAMPS", "WARNING", "duplicate timestamps detected in dataset coverage", m.Survivorship.UniverseID, m.DatasetID, "", false, "Deduplicate source data before strict promotion."))
		}
		if outOfOrder > 0 {
			addWarning(newRIFWarning("RIF_DATASET_OUT_OF_ORDER_TIMESTAMPS", "WARNING", "out-of-order timestamps detected in dataset coverage", m.Survivorship.UniverseID, m.DatasetID, "", false, "Sort or rebuild source data before strict promotion."))
		}
	}

	prov.RIFWarnings = append([]RIFWarning{}, warnings...)
	return prov, warnings, nil
}

func newRIFWarning(code, severity, reason, universe, dataset, symbol string, blocksPromotion bool, fix string) RIFWarning {
	return RIFWarning{
		Code:             code,
		Severity:         severity,
		Reason:           reason,
		AffectedUniverse: universe,
		AffectedDataset:  dataset,
		AffectedSymbol:   symbol,
		BlocksPromotion:  blocksPromotion,
		RecommendedFix:   fix,
	}
}

func formatRIFWarning(w RIFWarning) string {
	parts := []string{
		"CODE: " + w.Code,
		"SEVERITY: " + w.Severity,
		"REASON: " + w.Reason,
		"AFFECTED_UNIVERSE: " + emptyAsUnknown(w.AffectedUniverse),
		"AFFECTED_DATASET: " + emptyAsUnknown(w.AffectedDataset),
		"BLOCKS_PROMOTION: " + yesNo(w.BlocksPromotion),
		"RECOMMENDED_FIX: " + w.RecommendedFix,
	}
	if w.AffectedSymbol != "" {
		parts = append(parts, "AFFECTED_SYMBOL: "+w.AffectedSymbol)
	}
	return strings.Join(parts, " | ")
}

func emptyAsUnknown(value string) string {
	if value == "" {
		return "unknown"
	}
	return value
}

func yesNo(value bool) string {
	if value {
		return "Yes"
	}
	return "No"
}

func isKnownUniversePolicy(policy string) bool {
	switch policy {
	case "POINT_IN_TIME_EXCHANGE_UNIVERSE",
		"POINT_IN_TIME_VOLUME_FILTERED_UNIVERSE",
		"POINT_IN_TIME_MARKET_CAP_FILTERED_UNIVERSE",
		"EXPLICIT_SYMBOL_LIST",
		"CURRENT_ACTIVE_SYMBOL_LIST",
		"LOCAL_DATA_DISCOVERED_SYMBOLS",
		"UNKNOWN":
		return true
	default:
		return false
	}
}

func isNotPointInTimePolicy(policy string) bool {
	switch policy {
	case "EXPLICIT_SYMBOL_LIST", "CURRENT_ACTIVE_SYMBOL_LIST", "LOCAL_DATA_DISCOVERED_SYMBOLS", "UNKNOWN", "":
		return true
	default:
		return false
	}
}

func isUniverseDatasetMismatch(code string) bool {
	switch code {
	case "DATASET_SYMBOL_NOT_IN_UNIVERSE",
		"UNIVERSE_SYMBOL_MISSING_DATA",
		"DATASET_RANGE_OUTSIDE_UNIVERSE_WINDOW",
		"UNIVERSE_DATASET_MARKET_TYPE_MISMATCH",
		"UNIVERSE_DATASET_QUOTE_ASSET_MISMATCH":
		return true
	default:
		return false
	}
}

func hasString(values []string, target string) bool {
	for _, value := range values {
		if value == target {
			return true
		}
	}
	return false
}

func copyEvidenceSummary(in map[string]int) map[string]int {
	if len(in) == 0 {
		return nil
	}
	out := make(map[string]int, len(in))
	for k, v := range in {
		out[k] = v
	}
	return out
}

func hasWeakLifecycleEvidenceLevel(summary map[string]int) bool {
	for level, count := range summary {
		if count <= 0 {
			continue
		}
		switch level {
		case "LOCAL_DATA_FIRST_SEEN", "CURRENT_ACTIVE_ONLY", "USER_PROVIDED_UNVERIFIED", "UNKNOWN":
			return true
		}
	}
	return false
}

// VerifyLock verifies that two research locks match on reproducibility fields.
func (b *Bridge) VerifyLock(current, stored ResearchLock) error {
	return verifyLock(current, stored)
}

// LogHoldoutExposure exposes the holdout protection to tests.
func (b *Bridge) LogHoldoutExposure(datasetID, candID, expID string) error {
	key := datasetID + "\x00" + expID
	b.holdoutExposures[key]++
	if b.holdoutExposures[key] > b.holdoutLimit {
		return fmt.Errorf("holdout exposure limit exceeded for dataset %s experiment %s: %d > %d", datasetID, expID, b.holdoutExposures[key], b.holdoutLimit)
	}
	return nil
}
