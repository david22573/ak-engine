package main

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/david22573/ak-engine/internal/preconditions"
)

const candidateID = "phase12/DowntrendMidVolReliefLong240m"

const (
	candidateVersion            = "v1"
	candidateImplementationHash = "sha256:3c2e20fd5bf615864aebc5be35ce86c15a6ed8f83de33b2f1d33b00dae6fbfa1"
	historianDatasetVersion     = "sha256:afb3b656d6d97d61e8a10941e0553a90875c52dc8315b7e8e6f4de8ca6a725d2"
	historianManifestHash       = "sha256:2bdad377474ea79b830b54aab32df45d43f07070fbc4e8e4b39c41634b12e515"
	historianArtifactHash       = "sha256:8390ee15597143cd2d51cd8ef4edd1839d09e733b49f02e59478ebfa10e6779c"
	historianResultCommit       = "71edbf30c23bf830906be8944ccc0521f2dcc20f"
	rifResultCommit             = "0d535c59e94ba93c621f486b9c1965718a4be44f"
)

type historianGap struct {
	SchemaVersion         string    `json:"schema_version"`
	Status                string    `json:"status"`
	DatasetID             string    `json:"dataset_id"`
	DatasetVersion        string    `json:"dataset_version"`
	ManifestID            string    `json:"manifest_id"`
	ManifestHash          string    `json:"manifest_hash"`
	CandidateID           string    `json:"candidate_id"`
	CandidateVersion      string    `json:"candidate_version"`
	ImplementationHash    string    `json:"implementation_hash"`
	PhysicalCoverageStart time.Time `json:"physical_coverage_start"`
	PhysicalCoverageEnd   time.Time `json:"physical_coverage_end"`
	ProvablePITCoverage   struct {
		Available bool       `json:"available"`
		Start     *time.Time `json:"start"`
		End       *time.Time `json:"end"`
	} `json:"provable_pit_coverage"`
	EvaluationCutoff          time.Time         `json:"evaluation_cutoff"`
	CoveragePolicyVersion     string            `json:"coverage_policy_version"`
	AvailabilityPolicyVersion string            `json:"availability_policy_version"`
	EventSchemaVersion        string            `json:"event_schema_version"`
	RequiredSymbols           []string          `json:"required_symbols"`
	RequiredContextSymbols    []string          `json:"required_context_symbols"`
	ExpectedPartitions        []string          `json:"expected_partitions"`
	Snapshots                 []json.RawMessage `json:"snapshots"`
	MissingEvidence           []json.RawMessage `json:"missing_evidence"`
	SnapshotSetHash           string            `json:"snapshot_set_hash"`
	ManifestCreatedAt         time.Time         `json:"manifest_created_at"`
	HistorianBuild            string            `json:"historian_build"`
}

type historianSnapshotIdentity struct {
	PartitionKey      string     `json:"partition_key"`
	RelativePath      string     `json:"relative_path"`
	SourceAvailableAt *time.Time `json:"source_available_at"`
	ContentHash       string     `json:"content_hash"`
	PartitionHash     string     `json:"partition_hash"`
	EvidenceGaps      []string   `json:"evidence_gaps"`
}

type historianMissingEvidence struct {
	PartitionKey string   `json:"partition_key"`
	Reasons      []string `json:"reasons"`
}

func main() {
	var historianPath, outputDir, generatedAt string
	flag.StringVar(&historianPath, "historian-gap", "", "Historian gap-manifest JSON")
	flag.StringVar(&outputDir, "out-dir", "runs/reports", "report output directory")
	flag.StringVar(&generatedAt, "generated-at", "2026-07-13T08:00:00Z", "deterministic report timestamp")
	flag.Parse()
	if err := run(historianPath, outputDir, generatedAt); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run(historianPath, outputDir, generatedAt string) error {
	created, err := time.Parse(time.RFC3339, generatedAt)
	if err != nil {
		return err
	}
	gapBytes, err := os.ReadFile(historianPath)
	if err != nil {
		return err
	}
	gap, err := decodeHistorianGap(gapBytes)
	if err != nil {
		return err
	}
	verifiedHistorianArtifactHash := bytesHash(gapBytes)
	if verifiedHistorianArtifactHash != historianArtifactHash {
		return fmt.Errorf("Historian artifact bytes do not match the independently recorded identity")
	}
	schemaHash, err := preconditions.RetainedEventSchemaHash()
	if err != nil {
		return err
	}
	clusterSchemaHash, err := preconditions.IndependentClusterSchemaHash()
	if err != nil {
		return err
	}
	independence := preconditions.DefaultIndependencePolicy()
	independenceHash, err := preconditions.IndependencePolicyHash(independence)
	if err != nil {
		return err
	}
	uncertainty := preconditions.ProposedUncertaintyMethod()
	uncertaintyHash, err := preconditions.UncertaintyMethodHash(uncertainty)
	if err != nil {
		return err
	}
	partitionHash, err := preconditions.PartitionPolicyHash()
	if err != nil {
		return err
	}
	ledger, err := buildLedger()
	if err != nil {
		return err
	}

	common := map[string]any{
		"generated_at": created.UTC(), "candidate_id": candidateID,
		"engine_starting_commit":    "8fdc59e129446a140630c83f2d13628681035b75",
		"historian_starting_commit": "3eeff1eb45da281e0003dc1577ec55aa6cda1b1b",
		"rif_starting_commit":       "29350344a57e46f064442eada26e9418515990be",
		"historian_result_commit":   historianResultCommit,
		"rif_result_commit":         rifResultCommit,
	}
	preconditionReport := merge(common, map[string]any{
		"schema_version":     "ak.engine.pr4b0-r1p-preconditions-report.v1",
		"phase0":             map[string]any{"accepted_closeout_consistent": true, "correction_commit_required": false, "accepted_label": "PR4B0_R1_RESEARCH_BLOCKED"},
		"candidate_contract": map[string]any{"candidate_version": "v1", "family": "DowntrendMidVolRelief", "implementation_source_ref": "c2c7988712699b26ba7ab28e1cebb1f5312812a6", "implementation_hash": "sha256:3c2e20fd5bf615864aebc5be35ce86c15a6ed8f83de33b2f1d33b00dae6fbfa1", "required_primary_symbols": gap.RequiredSymbols, "required_context_symbols": gap.RequiredContextSymbols, "input_interval": "1m", "evaluation_horizon": "240m"},
		"candidate_capability_field_matrix": []map[string]string{
			{"field": "stable event ID", "status": "REQUIRED_DERIVED"},
			{"field": "candidate family/version", "status": "REQUIRED"},
			{"field": "primary symbol and event/decision timestamps", "status": "REQUIRED"},
			{"field": "source partition/snapshot/input hash", "status": "REQUIRED"},
			{"field": "feature schema version", "status": "REQUIRED"},
			{"field": "trend state/regime/volatility bucket", "status": "REQUIRED"},
			{"field": "relief magnitude", "status": "NOT_APPLICABLE_NO_AUTHORITATIVE_SOURCE_FIELD"},
			{"field": "BTC context", "status": "REQUIRED_FAIL_CLOSED"},
			{"field": "ETH context", "status": "REQUIRED_FAIL_CLOSED"},
			{"field": "funding or basis context", "status": "NOT_APPLICABLE_NOT_USED_BY_IMPLEMENTATION"},
			{"field": "reference price/evaluation horizon/warm-up", "status": "REQUIRED"},
			{"field": "same-symbol and common-market cluster identities", "status": "SUPPORTED_IN_SEPARATE_CLUSTER_SCHEMA"},
			{"field": "deterministic exclusion/cost inputs/attribution", "status": "REQUIRED"},
			{"field": "outcome-derived values", "status": "PROHIBITED_NOT_REPRESENTED"},
		},
		"historian": map[string]any{
			"status": gap.Status, "dataset_id": gap.DatasetID, "dataset_version": gap.DatasetVersion,
			"manifest_id": gap.ManifestID, "manifest_hash": gap.ManifestHash, "artifact_hash": verifiedHistorianArtifactHash,
			"physical_coverage_start": gap.PhysicalCoverageStart, "physical_coverage_end": gap.PhysicalCoverageEnd,
			"provable_pit_coverage": gap.ProvablePITCoverage, "evaluation_cutoff": gap.EvaluationCutoff,
			"coverage_policy_version": gap.CoveragePolicyVersion, "availability_policy_version": gap.AvailabilityPolicyVersion,
			"expected_partition_count": len(gap.ExpectedPartitions), "hashed_snapshot_count": len(gap.Snapshots),
			"missing_evidence_count": len(gap.MissingEvidence), "missing_evidence_reasons": []string{"AVAILABILITY_TIMESTAMP_MISSING", "SNAPSHOT_SCHEMA_UNSUPPORTED"},
			"snapshot_set_hash": gap.SnapshotSetHash, "event_schema_version": gap.EventSchemaVersion,
			"manifest_created_at": gap.ManifestCreatedAt, "historian_build": gap.HistorianBuild,
		},
		"independence_authority_recovery": map[string]any{
			"historical_source_ref":     "c2c7988712699b26ba7ab28e1cebb1f5312812a6:internal/app/phase12_downtrend_midvol_relief.go",
			"historical_behavior":       "count-only timestamp spacing with a one-hour threshold",
			"missing_authority":         []string{"retained cluster IDs", "same-symbol overlap identities", "cross-symbol common-market identities", "accepted overlapping-240m-horizon rule"},
			"accepted_policy_recovered": false,
			"proposed_policy_status":    independence.Status,
		},
		"contracts":                    map[string]any{"retained_event_schema_version": preconditions.RetainedEventSchemaVersion, "retained_event_schema_hash": schemaHash, "independent_cluster_schema_version": preconditions.IndependentClusterSchemaVersion, "independent_cluster_schema_hash": clusterSchemaHash, "independence_policy_version": independence.Version, "independence_policy_hash": independenceHash, "independence_policy_status": independence.Status, "uncertainty_method_version": uncertainty.Version, "uncertainty_method_hash": uncertaintyHash, "uncertainty_method_status": uncertainty.Status, "partition_policy_version": preconditions.PartitionPolicyVersion, "partition_policy_hash": partitionHash},
		"prohibited_activity_counters": map[string]any{"candidate_executions": 0, "candidate_outcome_calculations": 0, "accidental_legacy_literal_exposures_in_tool_output": 1, "validation_content_reads": 0, "holdout_content_reads": 0, "real_rif_registrations": 0, "real_rif_reservations": 0, "real_rif_consumptions": 0},
		"status":                       "PARTIALLY_RESTORED",
	})

	partitionReport := merge(common, map[string]any{
		"schema_version": "ak.engine.pr4b0-r1p-partition-eligibility.v1", "policy_version": preconditions.PartitionPolicyVersion, "policy_hash": partitionHash,
		"minimum_independent_decision_gate": 300, "outcome_data_accessed": false,
		"pit_coverage_available": false, "development": nil, "validation": nil, "final_holdout": nil,
		"structural_findings": []string{"All previously exposed candidate periods are DEVELOPMENT-only.", "No all-input PIT-valid interval is currently provable.", "No chronologically later all-symbol archive scope exists for unexposed VALIDATION and FINAL_HOLDOUT.", "Warm-up and embargo boundaries therefore cannot yet be instantiated."},
		"eligibility":         "INELIGIBLE_MISSING_PIT_AND_LATER_UNEXPOSED_COVERAGE",
	})
	uncertaintyReport := merge(common, map[string]any{
		"schema_version": "ak.engine.pr4b0-r1p-uncertainty-authority.v1", "accepted_method_recovered": false,
		"accepted_search_result": map[string]any{"gate": "lower confidence bound must be positive", "estimator": nil, "confidence_level": nil, "sampling_unit": nil, "resampling_method": nil, "block_construction": nil, "seed": nil, "number_of_resamples": nil, "interval_type": nil},
		"method":                 uncertainty, "method_hash": uncertaintyHash,
		"alternatives":                     []string{"analytical cluster-robust interval", "studentized cluster bootstrap", "moving-block bootstrap"},
		"governance_requirement":           "Explicit acceptance is required before a future PR4B0-R1 rerun.",
		"real_candidate_observations_used": 0,
	})
	rifReport := merge(common, map[string]any{
		"schema_version": "ak.engine.pr4b0-r1p-rif-preflight.v1", "rif_commit": rifResultCommit, "synthetic_or_static_only": true,
		"checks": []map[string]any{
			{"requirement": "exact research identity matching", "result": "PASS"},
			{"requirement": "exact candidate identity matching", "result": "PASS"},
			{"requirement": "frozen implementation/configuration/parameter/capability/feature/research-lock identities", "result": "PASS"},
			{"requirement": "one-time final-holdout reservation", "result": "PASS_SYNTHETIC"},
			{"requirement": "durable authorization before access", "result": "PASS_SYNTHETIC"},
			{"requirement": "atomic exposure consumption before protected callback", "result": "PASS_SYNTHETIC"},
			{"requirement": "replay/conflicting-evidence rejection", "result": "PASS"},
			{"requirement": "no manual bypass through the RIF authority gateway", "result": "PASS; FUTURE_DATA_OWNER_MUST_EXPOSE_CONTENT_ONLY_INSIDE_CALLBACK"},
			{"requirement": "no real registration/reservation/authorization/exposure", "result": "PASS"},
		},
		"real_state_created": false, "result": "COMPATIBLE_SYNTHETIC_PREFLIGHT_PASS",
		"future_sequence": []string{"register and advance the exact candidate version to VALIDATED using accepted evidence", "construct the frozen identity from independently verified accepted artifacts", "reserve once with a unique immutable operation ID and frozen final-holdout identity", "have the authorized governance actor verify and authorize the exact reservation and frozen identity", "consume authorization and exposure durably before protected access", "let the final-holdout data owner open content only inside the post-consumption callback", "bind consumed exposure evidence to any later lifecycle decision; reject all replay or conflict"},
	})
	finalReport := merge(common, map[string]any{
		"schema_version": "ak.engine.pr4b0-r1p-final-decision.v1", "label": "PR4B0_R1P_PRECONDITIONS_PARTIALLY_RESTORED",
		"meaningful_infrastructure_completed": []string{"verified hashed fail-closed Historian gap bundle", "versioned retained-event and independent-cluster schemas", "deterministic proposed independence policy", "deterministic proposed cluster-aware uncertainty method", "prior-exposure ledger", "structural partition policy", "RIF frozen-identity reserve/authorize/consume-before-access preflight"},
		"blockers":                            []string{"source availability timestamps are absent for every expected partition", "archive event schema is unversioned", "accepted independence authority is not established", "uncertainty method remains PROPOSED_NOT_ACCEPTED", "no legal later validation/final-holdout coverage exists", "two preserved exposure artifacts lack recoverable source commits", "one accidental legacy candidate-result literal exposure occurred during source inspection"},
		"candidate_executed":                  false, "validation_outcomes_inspected": false, "final_holdout_outcomes_inspected": false,
		"zero_candidate_outcome_inspection_proven": false,
		"recommended_next_phase":                   "NARROW_REMEDIATION_ONLY: obtain source availability/event-schema authority, recover preserved-artifact provenance, accept governance contracts, and establish later unexposed all-symbol PIT coverage.",
	})

	if err := writePair(outputDir, "pr4b0_r1p_preconditions_report", preconditionReport, preconditionsMarkdown(preconditionReport)); err != nil {
		return err
	}
	if err := writePair(outputDir, "pr4b0_r1p_prior_exposure_ledger", ledger, ledgerMarkdown(ledger)); err != nil {
		return err
	}
	if err := writePair(outputDir, "pr4b0_r1p_partition_eligibility", partitionReport, partitionMarkdown(partitionReport)); err != nil {
		return err
	}
	if err := writePair(outputDir, "pr4b0_r1p_uncertainty_authority", uncertaintyReport, uncertaintyMarkdown(uncertaintyHash)); err != nil {
		return err
	}
	if err := writePair(outputDir, "pr4b0_r1p_rif_preflight", rifReport, rifMarkdown()); err != nil {
		return err
	}
	return writePair(outputDir, "pr4b0_r1p_final_decision", finalReport, finalMarkdown())
}

func decodeHistorianGap(data []byte) (historianGap, error) {
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()
	var gap historianGap
	if err := decoder.Decode(&gap); err != nil {
		return historianGap{}, err
	}
	if err := ensureJSONEOF(decoder); err != nil {
		return historianGap{}, err
	}
	if err := validateHistorianGap(gap); err != nil {
		return historianGap{}, err
	}
	return gap, nil
}

func validateHistorianGap(gap historianGap) error {
	if gap.SchemaVersion != "ak-historian.pit-gap-manifest.v1" || gap.Status != "PIT_EVIDENCE_INCOMPLETE" || gap.ProvablePITCoverage.Available || gap.ProvablePITCoverage.Start != nil || gap.ProvablePITCoverage.End != nil {
		return fmt.Errorf("unexpected Historian authority state")
	}
	if gap.CandidateID != candidateID || gap.CandidateVersion != candidateVersion || gap.ImplementationHash != candidateImplementationHash {
		return fmt.Errorf("Historian candidate identity does not match the authoritative Engine contract")
	}
	for name, value := range map[string]string{"dataset_version": gap.DatasetVersion, "manifest_hash": gap.ManifestHash, "snapshot_set_hash": gap.SnapshotSetHash, "implementation_hash": gap.ImplementationHash} {
		if !validReportDigest(value) {
			return fmt.Errorf("Historian %s is not a lowercase prefixed SHA-256 digest", name)
		}
	}
	if gap.DatasetVersion != gap.SnapshotSetHash {
		return fmt.Errorf("Historian dataset and snapshot-set identities differ")
	}
	if gap.DatasetVersion != historianDatasetVersion || gap.ManifestHash != historianManifestHash {
		return fmt.Errorf("Historian dataset or manifest substitution detected")
	}
	if gap.DatasetID != "ak-historian-candles-futures-um-1m-pr4b0-r1p" || gap.ManifestID != "pr4b0-r1p-pit-gap-2023-2025-v1" || strings.ContainsAny(gap.DatasetID+gap.ManifestID, `/\\`) {
		return fmt.Errorf("Historian dataset or manifest identity is unexpected")
	}
	wantStart := time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC)
	wantEnd := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	if !gap.PhysicalCoverageStart.Equal(wantStart) || !gap.PhysicalCoverageEnd.Equal(wantEnd) || gap.EvaluationCutoff.Before(wantEnd) || gap.ManifestCreatedAt.Before(gap.EvaluationCutoff) {
		return fmt.Errorf("Historian coverage, cutoff, or creation identity is unexpected")
	}
	if gap.CoveragePolicyVersion != "ak-historian.coverage-policy.v1" || gap.AvailabilityPolicyVersion != "ak-historian.availability-policy.v1" || gap.EventSchemaVersion != "legacy-unversioned" || strings.TrimSpace(gap.HistorianBuild) == "" {
		return fmt.Errorf("Historian policy, schema, or build identity is unexpected")
	}
	wantPrimary := []string{"ADAUSDT", "AVAXUSDT", "BNBUSDT", "DOGEUSDT", "ETHUSDT", "LINKUSDT", "SOLUSDT", "XRPUSDT"}
	wantContext := []string{"BTCUSDT", "ETHUSDT"}
	if !sameStringSet(gap.RequiredSymbols, wantPrimary) || !sameStringSet(gap.RequiredContextSymbols, wantContext) {
		return fmt.Errorf("Historian symbol universe does not match the authoritative Engine contract")
	}
	if len(gap.ExpectedPartitions) != 324 || len(gap.Snapshots) != len(gap.ExpectedPartitions) || len(gap.MissingEvidence) != len(gap.ExpectedPartitions) {
		return fmt.Errorf("Historian partition, snapshot, or missing-evidence scope is incomplete")
	}
	expected := make(map[string]struct{}, len(gap.ExpectedPartitions))
	for _, partition := range gap.ExpectedPartitions {
		parts := strings.Split(partition, "/")
		if len(parts) != 4 || parts[0] != "futures-um" || parts[1] != "1m" {
			return fmt.Errorf("Historian expected partition identity is invalid")
		}
		if _, duplicate := expected[partition]; duplicate {
			return fmt.Errorf("Historian expected partitions contain a duplicate")
		}
		expected[partition] = struct{}{}
	}
	snapshots := make(map[string]struct{}, len(gap.Snapshots))
	for _, raw := range gap.Snapshots {
		var snapshot historianSnapshotIdentity
		if err := json.Unmarshal(raw, &snapshot); err != nil {
			return err
		}
		clean := filepath.Clean(snapshot.RelativePath)
		if _, declared := expected[snapshot.PartitionKey]; !declared || filepath.IsAbs(snapshot.RelativePath) || clean != snapshot.RelativePath || strings.HasPrefix(clean, "..") || strings.Contains(snapshot.RelativePath, `\`) || !validReportDigest(snapshot.ContentHash) || snapshot.PartitionHash != snapshot.ContentHash || snapshot.SourceAvailableAt != nil || !sameStringSet(snapshot.EvidenceGaps, []string{"AVAILABILITY_TIMESTAMP_MISSING", "SNAPSHOT_SCHEMA_UNSUPPORTED"}) {
			return fmt.Errorf("Historian snapshot identity or fail-closed evidence is invalid")
		}
		if _, duplicate := snapshots[snapshot.PartitionKey]; duplicate {
			return fmt.Errorf("Historian snapshots contain a duplicate partition")
		}
		snapshots[snapshot.PartitionKey] = struct{}{}
	}
	missing := make(map[string]struct{}, len(gap.MissingEvidence))
	for _, raw := range gap.MissingEvidence {
		var evidence historianMissingEvidence
		if err := json.Unmarshal(raw, &evidence); err != nil {
			return err
		}
		if _, declared := expected[evidence.PartitionKey]; !declared || !sameStringSet(evidence.Reasons, []string{"AVAILABILITY_TIMESTAMP_MISSING", "SNAPSHOT_SCHEMA_UNSUPPORTED"}) {
			return fmt.Errorf("Historian missing-partition evidence is invalid")
		}
		if _, duplicate := missing[evidence.PartitionKey]; duplicate {
			return fmt.Errorf("Historian missing evidence contains a duplicate partition")
		}
		missing[evidence.PartitionKey] = struct{}{}
	}
	return nil
}

func ensureJSONEOF(decoder *json.Decoder) error {
	var extra any
	if err := decoder.Decode(&extra); err == io.EOF {
		return nil
	} else if err != nil {
		return err
	}
	return fmt.Errorf("Historian artifact contains trailing JSON")
}

func validReportDigest(value string) bool {
	if len(value) != len("sha256:")+sha256.Size*2 || !strings.HasPrefix(value, "sha256:") || strings.ToLower(value) != value {
		return false
	}
	_, err := hex.DecodeString(strings.TrimPrefix(value, "sha256:"))
	return err == nil
}

func sameStringSet(left, right []string) bool {
	if len(left) != len(right) {
		return false
	}
	seen := make(map[string]struct{}, len(left))
	for _, value := range left {
		if _, duplicate := seen[value]; duplicate {
			return false
		}
		seen[value] = struct{}{}
	}
	for _, value := range right {
		if _, ok := seen[value]; !ok {
			return false
		}
	}
	return true
}

func buildLedger() (preconditions.ExposureLedger, error) {
	start := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	end := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	window := preconditions.TimeWindow{Start: start, End: end}
	entry := func(report, commit, hash string, granularity, metrics []string, symbol, month, quarter bool) preconditions.ExposureEntry {
		return preconditions.ExposureEntry{SourceReport: report, SourceCommit: commit, CandidateID: candidateID, ExposedWindow: window, GranularityExposed: granularity, MetricsExposed: metrics, SymbolLevelExposed: symbol, MonthLevelExposed: month, QuarterLevelExposed: quarter, EvidenceHash: hash}
	}
	fileDigest := func(path string) string {
		data, err := os.ReadFile(path)
		if err != nil {
			panic(err)
		}
		return bytesHash(data)
	}
	entries := []preconditions.ExposureEntry{
		entry("runs/reports/phase12_3_downtrend_midvol_relief_eval.json", "c2c7988712699b26ba7ab28e1cebb1f5312812a6", "sha256:fb27fe46ab1139ccafea3a7b3cbb7bfdfc7fb3bd2f7e545f1b7b566d2e6c9066", []string{"aggregate", "symbol", "month", "quarter"}, []string{"event/sample summaries", "return summaries", "profit factor", "expectancy", "win/loss summaries", "cost/delay sensitivity", "worst-period behavior"}, true, true, true),
		entry("preserved:runs/reports/phase12_4_downtrend_midvol_relief_near_miss_audit.json", "UNTRACKED_PRESERVED_EVIDENCE", "sha256:df4857fa908d3ad044092c84c0d65474ab5b2f7d339a0e571ad4974d42b7a38b", []string{"aggregate", "symbol", "month", "quarter"}, []string{"sample summaries", "net outcome summaries", "worst-period behavior", "concentration summaries"}, true, true, true),
		entry("preserved:runs/reports/phase12_5_candidate_risk_filter_design.json", "UNTRACKED_PRESERVED_EVIDENCE", "sha256:4c88c2bac766281a8f98b33e34e4a8ad3557dbb2f8730139ff56785056261596", []string{"aggregate", "variant", "month", "quarter"}, []string{"variant exclusions", "net outcome summaries", "period behavior", "parameter/filter comparisons"}, false, true, true),
		entry("runs/reports/pr4b0_candidate_inventory.json", "205cf59555006ce23fc58bc2c73262660a894850", fileDigest("runs/reports/pr4b0_candidate_inventory.json"), []string{"aggregate", "symbol", "month", "quarter"}, []string{"sample summaries", "profit factor", "expectancy", "worst-period behavior", "structural concentration"}, true, true, true),
		entry("runs/reports/pr4b0_candidate_qualification.json", "205cf59555006ce23fc58bc2c73262660a894850", fileDigest("runs/reports/pr4b0_candidate_qualification.json"), []string{"aggregate", "qualification"}, []string{"qualification gate evidence", "aggregate outcome summaries"}, false, false, false),
		entry("runs/reports/pr4b0_r1_variant_results.json", "8f4df1e61455541262cc1c95e6a32e6b8948f980", fileDigest("runs/reports/pr4b0_r1_variant_results.json"), []string{"aggregate", "variant", "quarter"}, []string{"legacy aggregate reproduction", "variant comparisons", "worst-period behavior"}, false, false, true),
		entry("runs/reports/pr4b0_r1_final_decision.json", "945640fcd16537b8e1a82c49c4de28b5899982b9", fileDigest("runs/reports/pr4b0_r1_final_decision.json"), []string{"aggregate", "decision"}, []string{"legacy aggregate reproduction", "research blockers"}, false, false, false),
		entry("runs/reports/pr4b0_r1_evidence_supplement.json", "8fdc59e129446a140630c83f2d13628681035b75", fileDigest("runs/reports/pr4b0_r1_evidence_supplement.json"), []string{"aggregate", "evidence correction"}, []string{"legacy aggregate reproduction", "replay-evidence status"}, false, false, false),
	}
	return preconditions.SealExposureLedger(preconditions.ExposureLedger{SchemaVersion: preconditions.ExposureLedgerSchemaVersion, CandidateID: candidateID, Entries: entries})
}

func writePair(dir, name string, value any, markdown string) error {
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		return err
	}
	if err := os.WriteFile(filepath.Join(dir, name+".json"), append(data, '\n'), 0o644); err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(dir, name+".md"), []byte(markdown), 0o644)
}

func bytesHash(data []byte) string {
	digest := sha256.Sum256(data)
	return "sha256:" + hex.EncodeToString(digest[:])
}
func merge(left, right map[string]any) map[string]any {
	result := map[string]any{}
	for key, value := range left {
		result[key] = value
	}
	for key, value := range right {
		result[key] = value
	}
	return result
}

func preconditionsMarkdown(report map[string]any) string {
	return "# PR4B0-R1P preconditions report\n\nStatus: **PARTIALLY_RESTORED**. The closeout is consistent. Immutable bytes and the complete identity chain were verified for the explicit archive scope, but PIT availability and event-schema authority remain absent. Engine replay, clustering, uncertainty, exposure, and partition contracts and the RIF pre-access authority were verified with synthetic tests only. One accidental source-output exposure of legacy result literals prevents a zero-inspection claim; no candidate execution or calculation occurred.\n"
}
func ledgerMarkdown(ledger preconditions.ExposureLedger) string {
	return fmt.Sprintf("# PR4B0-R1P prior-exposure ledger\n\nLedger hash: `%s`\n\nThe ledger records %d pre-existing evidence artifacts over the already exposed 2024-01-01 through 2026-01-01 interval. It records metric categories, never values. Two preserved artifacts lack a recoverable source commit and remain explicit authority gaps.\n", ledger.LedgerHash, len(ledger.Entries))
}
func partitionMarkdown(report map[string]any) string {
	return "# PR4B0-R1P partition eligibility\n\nNo structural DEVELOPMENT/VALIDATION/FINAL_HOLDOUT boundaries are currently eligible. Previously exposed periods are DEVELOPMENT-only; no complete PIT-valid interval exists, and no later all-symbol scope exists for both validation and final holdout. The 300-independent-decision gate is unchanged. No validation or holdout content was read.\n"
}
func uncertaintyMarkdown(hash string) string {
	return fmt.Sprintf("# PR4B0-R1P uncertainty authority\n\nThe accepted gate names a positive lower confidence bound but specifies no estimator, confidence level, sampling unit, resampling method, seed, resample count, block construction, or interval rule. The implemented cluster-bootstrap contract is **PROPOSED_NOT_ACCEPTED** with hash `%s`; governance acceptance is required before rerun. Only synthetic observations were used.\n", hash)
}
func rifMarkdown() string {
	return "# PR4B0-R1P RIF preflight\n\nResult: **COMPATIBLE_SYNTHETIC_PREFLIGHT_PASS** at RIF commit `0d535c59e94ba93c621f486b9c1965718a4be44f`. Synthetic tests prove exact registered research/candidate/lifecycle and frozen implementation/configuration/parameter/capability/feature/research-lock identities, one-time reservation, durable authorization, exposure and consumption before the protected callback, crash retry, and replay/conflict rejection. The future data owner must expose content only inside that callback; direct filesystem access is outside RIF's process boundary. No real registration, reservation, authorization, exposure, protected-data read, or paper action was created.\n"
}
func finalMarkdown() string {
	return strings.TrimSpace(`# PR4B0-R1P final decision

Final label: **PR4B0_R1P_PRECONDITIONS_PARTIALLY_RESTORED**

Meaningful infrastructure is complete, including the RIF pre-access authority, but PIT availability/event-schema authority, preserved-artifact provenance, accepted clustering and uncertainty governance, and legal later partitions remain unresolved. One accidental legacy literal exposure also prevents the required zero-inspection proof. The candidate was not executed; validation and final-holdout outcomes were not inspected.

Recommended next phase: narrow remediation only—obtain source availability and event-schema authority, recover preserved-artifact provenance, accept the proposed governance contracts, and collect a later unexposed all-symbol PIT-valid scope.
`) + "\n"
}
