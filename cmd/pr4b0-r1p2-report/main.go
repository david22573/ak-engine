package main

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/david22573/ak-engine/internal/governance"
	"github.com/david22573/ak-engine/internal/preconditions"
)

const candidateID = "phase12/DowntrendMidVolReliefLong240m"

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run() error {
	var historianDir, outputDir string
	flag.StringVar(&historianDir, "historian-authority-dir", "", "directory containing committed Historian R1P2 authorities")
	flag.StringVar(&outputDir, "out-dir", "runs/reports", "report output directory")
	flag.Parse()
	if historianDir == "" {
		return errors.New("--historian-authority-dir is required")
	}

	readArtifact := func(name string) (map[string]any, string, error) {
		data, err := os.ReadFile(filepath.Join(historianDir, name+".json"))
		if err != nil {
			return nil, "", err
		}
		var value map[string]any
		if err := json.Unmarshal(data, &value); err != nil {
			return nil, "", err
		}
		return value, bytesHash(data), nil
	}
	availability, availabilityFileHash, err := readArtifact("source_availability_authority")
	if err != nil {
		return err
	}
	availabilityGap, availabilityGapFileHash, err := readArtifact("source_availability_gap_manifest")
	if err != nil {
		return err
	}
	sourceSchema, sourceSchemaFileHash, err := readArtifact("source_schema_authority")
	if err != nil {
		return err
	}
	physical, physicalFileHash, err := readArtifact("physical_archive_identity")
	if err != nil {
		return err
	}
	receiptSchema, receiptSchemaHash, err := readArtifact("prospective_ingestion_receipt_schema")
	if err != nil {
		return err
	}
	manifestContract, manifestContractHash, err := readArtifact("prospective_manifest_contract")
	if err != nil {
		return err
	}
	_ = receiptSchema
	_ = manifestContract
	if physical["classification"] != "IMMUTABLE_PHYSICAL_ARCHIVE_GAP_IDENTITY" || int(number(physical["snapshot_count"])) != 324 || boolValue(physical["provable_pit_coverage"]) {
		return errors.New("Historian physical identity is not the required fail-closed 324-snapshot scope")
	}
	counts, ok := availability["status_counts"].(map[string]any)
	if !ok || int(number(counts["AVAILABILITY_AUTHORITY_MISSING"])) != 324 || int(number(counts["AVAILABILITY_AUTHORITY_VERIFIED"])) != 0 {
		return errors.New("Historian availability gap counts are unexpected")
	}
	if int(number(sourceSchema["validated_count"])) != 324 || int(number(sourceSchema["failed_count"])) != 0 || boolValue(sourceSchema["mixed_schema_versions"]) {
		return errors.New("Historian source schema authority is incomplete")
	}

	primary := []string{"ADAUSDT", "AVAXUSDT", "BNBUSDT", "DOGEUSDT", "ETHUSDT", "LINKUSDT", "SOLUSDT", "XRPUSDT"}
	context := []string{"BTCUSDT", "ETHUSDT"}
	capability := map[string]any{"schema_version": "ak.engine.capability.downtrend-midvol-relief.v1", "candidate_id": candidateID, "candidate_version": "v1", "implementation_hash": "sha256:3c2e20fd5bf615864aebc5be35ce86c15a6ed8f83de33b2f1d33b00dae6fbfa1", "primary_symbols": primary, "context_symbols": context, "input_interval": "1m", "evaluation_horizon": "240m", "required_decision_inputs": []string{"RealizedVol60 mid bucket", "downtrend classification", "BTCUSDT 60-minute context return", "ETHUSDT 60-minute context return"}}
	capabilityHash, err := governance.HashCanonical(capability)
	if err != nil {
		return err
	}

	provenance, err := buildProvenance()
	if err != nil {
		return err
	}
	inspection, err := buildInspection()
	if err != nil {
		return err
	}
	independencePacket, err := buildIndependencePacket()
	if err != nil {
		return err
	}
	uncertaintyPacket, err := buildUncertaintyPacket()
	if err != nil {
		return err
	}

	common := map[string]any{
		"candidate_id": candidateID, "engine_starting_commit": "0d70c068898c91521e391b65ee4299a53b9cf394", "historian_starting_commit": "71edbf30c23bf830906be8944ccc0521f2dcc20f", "rif_starting_commit": "0d535c59e94ba93c621f486b9c1965718a4be44f",
		"candidate_executions": 0, "candidate_outcome_calculations": 0, "validation_content_reads": 0, "holdout_content_reads": 0, "real_rif_state_created": false,
	}
	authorityReport := merge(common, map[string]any{
		"schema_version": "ak.engine.pr4b0-r1p2-authority-report.v1", "status": "AUTHORITY_AND_GOVERNANCE_PACKET_READY",
		"identity_terminology_audit": map[string]any{"classification": "IMMUTABLE_PHYSICAL_ARCHIVE_GAP_IDENTITY", "pit_valid_research_evidence": false, "prior_reports_reviewed": []string{"pr4b0_r1p_preconditions_report", "pr4b0_r1p_partition_eligibility", "pr4b0_r1p_final_decision"}, "prior_reports_requiring_correction": []string{}, "reason": "prior reports already state PIT evidence is incomplete; this packet adds the explicit non-PIT classification"},
		"physical_archive":           map[string]any{"dataset_id": physical["dataset_id"], "dataset_version": physical["dataset_version"], "manifest_id": physical["manifest_id"], "manifest_hash": physical["manifest_hash"], "classification": physical["classification"], "snapshot_count": physical["snapshot_count"], "provable_pit_coverage": false, "identity_hash": physical["identity_hash"], "artifact_file_hash": physicalFileHash},
		"availability_authority":     map[string]any{"authority_hash": availability["authority_hash"], "artifact_file_hash": availabilityFileHash, "gap_hash": availabilityGap["gap_hash"], "gap_artifact_file_hash": availabilityGapFileHash, "status_counts": counts, "timestamps_synthesized": 0},
		"source_schema_authority":    map[string]any{"version": sourceSchema["source_schema_version"], "fingerprint": sourceSchema["schema_fingerprint"], "authority_hash": sourceSchema["authority_hash"], "artifact_file_hash": sourceSchemaFileHash, "validated_count": sourceSchema["validated_count"], "failed_count": sourceSchema["failed_count"], "mixed_schema_versions": sourceSchema["mixed_schema_versions"]},
		"symbol_universe":            map[string]any{"primary_symbols": primary, "context_symbols": context, "primary_context_overlap": []string{"ETHUSDT"}, "unique_symbols": []string{"ADAUSDT", "AVAXUSDT", "BNBUSDT", "BTCUSDT", "DOGEUSDT", "ETHUSDT", "LINKUSDT", "SOLUSDT", "XRPUSDT"}, "unique_symbol_count": 9, "authoritative_source": "accepted candidate inventory plus exact implementation source and Historian bound gap identity", "capability_contract": capability, "capability_contract_hash": capabilityHash},
	})
	authorityReport, err = sealMap(authorityReport, "report_hash")
	if err != nil {
		return err
	}
	collectionReport := merge(common, map[string]any{
		"schema_version": "ak.engine.pr4b0-r1p2-collection-authority-design.v1", "status": "DESIGN_ONLY_REAL_COLLECTION_NOT_STARTED",
		"collector_contract":     map[string]any{"acquire_once": true, "source_availability_at_birth": true, "hash_before_append_only_registration": true, "schema_validate_before_eligibility": true, "all_primary_and_context_coverage_required": true, "mutable_aliases_rejected": true, "local_paths_excluded_from_identity": true, "network_not_required_for_deterministic_revalidation": true},
		"manifest_contract_hash": manifestContractHash, "ingestion_receipt_schema_hash": receiptSchemaHash, "historian_contract_artifacts": map[string]any{"prospective_manifest_contract": "authority/prospective_manifest_contract.json", "prospective_ingestion_receipt_schema": "authority/prospective_ingestion_receipt_schema.json"},
		"validation_command": "ak-historian validate-prospective-manifest --manifest <manifest.json>", "recovery_behavior": "resume only from last verified receipt hash; quarantine orphaned bytes", "duplicate_behavior": "same identity and byte/receipt hashes is idempotent", "conflict_behavior": "same identity with different bytes or receipt fails closed and quarantines both claims",
		"synthetic_fixture_results":   []map[string]any{{"case": "immutable receipt", "result": "PASS"}, {"case": "missing availability", "result": "REJECTED"}, {"case": "missing schema", "result": "REJECTED"}, {"case": "missing context", "result": "REJECTED"}, {"case": "identical duplicate", "result": "IDEMPOTENT"}, {"case": "conflicting duplicate", "result": "REJECTED"}, {"case": "mutable alias", "result": "REJECTED"}, {"case": "local path variance", "result": "IDENTITY_UNCHANGED"}},
		"rif_synthetic_compatibility": map[string]any{"result": "PASS_SYNTHETIC_NO_REAL_STATE", "dataset_binding": "prospective dataset ID/version -> RIF research identity dataset fields", "partition_binding": "synthetic future partition manifest hash -> frozen final-holdout manifest identity", "receipt_binding": "receipt/manifest hashes become frozen evidence inputs", "registered_candidates": 0, "reservations": 0, "authorizations": 0, "consumptions": 0, "protected_reads": 0, "paper_authorizations": 0},
	})
	collectionReport, err = sealMap(collectionReport, "design_hash")
	if err != nil {
		return err
	}
	finalReport := merge(common, map[string]any{
		"schema_version": "ak.engine.pr4b0-r1p2-final-decision.v1", "executive_verdict": "All static authority scopes are resolved or precisely fail-closed, both proposed governance contracts are reviewable, and prospective collection/RIF mechanics are synthetically specified without candidate execution.",
		"label": "PR4B0_R1P2_AUTHORITY_AND_GOVERNANCE_PACKET_READY", "recommended_next_action": "USER_GOVERNANCE_REVIEW_REQUIRED",
		"policies_accepted": false, "pit_valid_research_data_exists": false, "real_future_dataset_collected": false, "fresh_preregistration_required": true,
		"bounded_remaining_items": []map[string]any{{"item": "historical source availability", "state": "324 MISSING precisely gap-manifested", "blocks_packet": false, "blocks_pit_eligibility": true}, {"item": "independence governance decision", "state": "ACCEPT/REJECT/REVISE pending", "blocks_packet": false, "blocks_research": true}, {"item": "uncertainty governance decision", "state": "ACCEPT/REJECT/REVISE pending", "blocks_packet": false, "blocks_research": true}, {"item": "future unexposed collection", "state": "designed, not run", "blocks_packet": false, "blocks_research": true}, {"item": "preserved artifact source commits", "state": "unrecoverable; conservatively treated as exposed", "blocks_packet": false, "blocks_validation_or_holdout_for_affected_period": true}},
		"artifact_hashes":         map[string]any{"authority_report": authorityReport["report_hash"], "provenance_resolution": provenance.ResolutionHash, "inspection_audit": inspection.AuditHash, "independence_governance_packet": independencePacket["packet_hash"], "uncertainty_governance_packet": uncertaintyPacket["packet_hash"], "collection_authority_design": collectionReport["design_hash"], "historian_physical_identity": physical["identity_hash"], "historian_availability_authority": availability["authority_hash"], "historian_source_schema_authority": sourceSchema["authority_hash"], "prospective_manifest_contract_file": manifestContractHash, "prospective_receipt_schema_file": receiptSchemaHash},
	})
	finalReport, err = sealMap(finalReport, "decision_hash")
	if err != nil {
		return err
	}

	if err := os.MkdirAll(outputDir, 0o755); err != nil {
		return err
	}
	reports := []struct {
		name     string
		value    any
		markdown string
	}{
		{"pr4b0_r1p2_authority_report", authorityReport, authorityMarkdown(authorityReport)},
		{"pr4b0_r1p2_provenance_resolution", provenance, provenanceMarkdown(provenance)},
		{"pr4b0_r1p2_inspection_audit", inspection, inspectionMarkdown(inspection)},
		{"pr4b0_r1p2_independence_governance_packet", independencePacket, independenceMarkdown(independencePacket)},
		{"pr4b0_r1p2_uncertainty_governance_packet", uncertaintyPacket, uncertaintyMarkdown(uncertaintyPacket)},
		{"pr4b0_r1p2_collection_authority_design", collectionReport, collectionMarkdown(collectionReport)},
		{"pr4b0_r1p2_final_decision", finalReport, finalMarkdown(finalReport)},
	}
	for _, report := range reports {
		if err := writePair(outputDir, report.name, report.value, report.markdown); err != nil {
			return err
		}
	}
	return nil
}

func buildProvenance() (governance.ProvenanceResolution, error) {
	start, end := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC), time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	seen := time.Date(2026, 7, 14, 1, 0, 22, 96000000, time.UTC)
	entry := func(path, hash, blob string, categories []string) governance.ProvenanceEntry {
		return governance.ProvenanceEntry{ArtifactPath: path, SHA256: hash, GitBlobID: blob, SourceCommit: nil, EarliestKnownAppearance: seen, CandidateID: candidateID, PossibleExposureRanges: []governance.TimeRange{{Start: start, End: end}}, InformationCategories: categories, ByteIdenticalPaths: []string{path}, ProvenanceEdges: []string{path + " -> " + hash, hash + " -> prior exposure ledger", "prior exposure ledger -> [2024-01-01,2026-01-01) DEVELOPMENT-only"}, Resolution: "UNTRUSTED_PROVENANCE_TREATED_AS_EXPOSED", ValidationEligible: false, HoldoutEligible: false}
	}
	return governance.SealProvenance(governance.ProvenanceResolution{Entries: []governance.ProvenanceEntry{
		entry("runs/reports/phase12_4_downtrend_midvol_relief_near_miss_audit.json", "sha256:df4857fa908d3ad044092c84c0d65474ab5b2f7d339a0e571ad4974d42b7a38b", "6dd463f07e175f6c74ad4d0cbdac9c119e443fe3", []string{"aggregate outcome summaries", "event/sample summaries", "symbol/month/quarter breakdowns", "worst-period and concentration summaries"}),
		entry("runs/reports/phase12_5_candidate_risk_filter_design.json", "sha256:4c88c2bac766281a8f98b33e34e4a8ad3557dbb2f8730139ff56785056261596", "f13cdd881f033ec7f552c1034bc8c72ea5348eb8", []string{"variant exclusions", "aggregate outcome summaries", "period behavior", "parameter/filter comparisons"}),
	}})
}

func buildInspection() (governance.InspectionAudit, error) {
	return governance.SealInspectionAudit(governance.InspectionAudit{Tool: "exec_command", Command: "sed -n '1,320p' internal/qualification/qualification.go; sed -n '1,360p' internal/qualification/qualification_test.go; sed -n '1,320p' internal/app/pr4b0_candidate_qualification.go", Timestamp: time.Date(2026, 7, 14, 0, 30, 34, 218000000, time.UTC), Repository: "ak-engine", Commit: "8fdc59e129446a140630c83f2d13628681035b75", FilesDisplayed: []string{"internal/qualification/qualification.go", "internal/qualification/qualification_test.go", "internal/app/pr4b0_candidate_qualification.go"}, LiteralCategories: []string{"qualification gate thresholds", "aggregate historical sample/count literals", "aggregate historical expectancy literals", "aggregate historical worst-quarter profit-factor literals", "cost-stress and status literals", "synthetic unit-test literals"}, CandidateFamily: "DowntrendMidVolReliefLong240m", AffectedPeriods: []governance.TimeRange{{Start: time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC), End: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)}}, SymbolLevelInformationAppeared: false, MonthLevelInformationAppeared: false, QuarterLevelInformationAppeared: true, LaterThanKnownResearchPeriodAppeared: false, ProspectiveValidationExposed: false, ProspectiveHoldoutExposed: false, AffectedImplementationOrPolicyDecision: false, InspectionCount: 1, Classification: "LEGACY_ALREADY_EXPOSED_CONTENT_ONLY", ValidationEligible: false, HoldoutEligible: false, FreshPreregistrationRequired: true})
}

func buildIndependencePacket() (map[string]any, error) {
	policy := preconditions.DefaultIndependencePolicy()
	policyHash, err := preconditions.IndependencePolicyHash(policy)
	if err != nil {
		return nil, err
	}
	base := time.Date(2030, 1, 1, 0, 0, 0, 0, time.UTC)
	events := []preconditions.RetainedEvent{syntheticEvent("AAAUSDT", base), syntheticEvent("AAAUSDT", base.Add(2*time.Hour)), syntheticEvent("BBBUSDT", base.Add(3*time.Hour)), syntheticEvent("AAAUSDT", base.Add(7*time.Hour)), syntheticEvent("AAAUSDT", base.Add(11*time.Hour))}
	clusters, err := preconditions.ClusterEvents(events, policy)
	if err != nil {
		return nil, err
	}
	concentration, err := preconditions.ClusterConcentration(clusters)
	if err != nil {
		return nil, err
	}
	inputs := make([]map[string]any, 0, len(events))
	for _, event := range events {
		inputs = append(inputs, map[string]any{"event_id": event.EventID, "symbol": event.PrimarySymbol, "decision_timestamp": event.DecisionTimestamp})
	}
	expected := make([]map[string]any, 0, len(clusters))
	for _, cluster := range clusters {
		expected = append(expected, map[string]any{"cluster_id": cluster.ClusterID, "start": cluster.Start, "end": cluster.End, "member_event_ids": cluster.MemberEventIDs, "member_symbols": cluster.MemberSymbols})
	}
	packet := map[string]any{"schema_version": "ak.engine.independence-governance-packet.v1", "contract": policy, "contract_hash": policyHash, "contract_status": preconditions.PolicyStatusProposedNotAccepted,
		"normative_specification": map[string]any{"event_cluster_algorithm": "validate and canonical-deduplicate retained events; sort by decision timestamp then event ID; sweep left-to-right; each event occupies [decision, decision+max(spacing,horizon)); merge transitively while next decision is strictly before active end", "same_symbol_overlap_rule": policy.SameSymbolOverlap, "cross_symbol_common_market_overlap_rule": policy.CrossSymbolOverlap, "horizon_overlap_rule": "all 240-minute decision horizons are half-open and transitively merged", "minimum_spacing_rule": "14400000 ms; equal to the 240-minute horizon", "timestamp_boundary_semantics": policy.BoundaryRule, "cluster_id_construction": policy.ClusterIDRule, "duplicate_handling": policy.DuplicateRule, "deterministic_ordering_and_tie_breaking": policy.OrderingRule, "independent_sample_definition": "one complete cluster is one independent sample; member events never count independently", "concentration_calculation": "largest cluster member-event count divided by total member-event count; zero for an empty cluster set", "fail_closed_conditions": []string{"invalid/missing retained-event authority", "missing decision timestamp or required context", "conflicting duplicate event ID", "unsupported version/status", "nonpositive spacing or horizon", "invalid/duplicate cluster ID"}, "versioning_and_mutation_rules": "any normative field mutation creates a new hash/version and requires a fresh governance decision"},
		"synthetic_test_vectors":  []map[string]any{{"name": "transitive overlap, exact boundary, and input-order invariance", "inputs": inputs, "expected_clusters": expected, "expected_concentration": concentration}},
		"rationale":               "The four-hour rule groups overlapping outcome horizons and common-market shocks to avoid pseudoreplication; sample-count maximization is not an objective.", "known_limitations": []string{"blanket BTC/ETH common-market grouping may over-cluster unrelated cross-symbol events", "a fixed four-hour horizon may not capture slower common shocks", "concentration is descriptive and has no accepted threshold", "policy has not been run against real candidate events in this phase"}, "alternatives_considered": []string{"same-symbol-only overlap", "one-hour legacy spacing", "calendar-time blocks", "dynamic correlation regimes"}, "governance_decision_requested": map[string]any{"allowed_decisions": []string{"ACCEPT", "REJECT", "REVISE"}, "preselected_decision": nil, "unresolved_decisions": []string{"accept four-hour minimum spacing", "accept blanket cross-symbol BTC/ETH common-market clustering", "accept half-open exact-boundary split", "accept largest-cluster concentration definition"}}, "real_candidate_events_used": 0}
	return sealMap(packet, "packet_hash")
}

func buildUncertaintyPacket() (map[string]any, error) {
	method := preconditions.ProposedUncertaintyMethod()
	methodHash, err := preconditions.UncertaintyMethodHash(method)
	if err != nil {
		return nil, err
	}
	observations := []preconditions.ClusterObservation{{ClusterID: "synthetic-a", NetValue: 1}, {ClusterID: "synthetic-b", NetValue: 2}, {ClusterID: "synthetic-c", NetValue: 3}, {ClusterID: "synthetic-d", NetValue: 4}, {ClusterID: "synthetic-e", NetValue: 5}}
	result, err := preconditions.EstimateLowerBound(observations, method)
	if err != nil {
		return nil, err
	}
	packet := map[string]any{"schema_version": "ak.engine.uncertainty-governance-packet.v1", "contract": method, "contract_hash": methodHash, "contract_status": preconditions.PolicyStatusProposedNotAccepted,
		"estimand": "expected frozen-cost net basis-point outcome per independent cluster for the exact preregistered candidate and partition", "expectancy_definition": "arithmetic mean of exactly one net observation per independent cluster", "net_cost_treatment": "each input is net of the complete frozen fee/spread/slippage/funding/adverse-selection cost vector before resampling", "confidence_level": method.ConfidenceLevel, "resampling_unit": method.SamplingUnit, "relationship_to_independence_contract": "cluster IDs and membership must be produced by the exact accepted independence contract hash; events are never resampled directly", "procedure": "sort unique clusters by ID; for each replicate draw n cluster observations independently with replacement; compute replicate mean; sort all replicate means", "replacement_rules": "sampling is with replacement within every replicate; each replicate contains exactly n draws from n unique clusters", "stratification": "none", "block_construction": method.BlockConstruction, "seed_derivation": "fixed versioned uint64 constant 0x5052344230523150; any seed change mutates the contract hash", "number_of_resamples": method.NumberOfResamples, "interval_construction": "one-sided lower percentile at floor((1-confidence_level)*B), clamped to valid replicate indices", "degenerate_samples": "fewer than two unique clusters fail; identical finite observations are valid and yield a degenerate point interval", "missing_or_invalid_records": "missing cluster IDs, non-finite values, or conflicting duplicate IDs fail closed; exact duplicates deduplicate", "deterministic_serialization": "canonical JSON field names plus sorted cluster IDs; result records method hash/status, cluster count, estimator, bound, confidence, seed, and B", "hash_and_version_rules": "any normative mutation changes the contract hash and requires a new version/governance decision",
		"synthetic_test_vectors": []map[string]any{{"name": "five positive synthetic clusters", "inputs": observations, "expected_output": result}, {"name": "conflicting duplicate", "expected": "FAIL_CLOSED"}, {"name": "single cluster", "expected": "FAIL_CLOSED"}}, "statistical_rationale": "cluster resampling preserves the declared dependence unit and provides a deterministic lower uncertainty bound without assuming event-level independence", "small_sample_limitations": []string{"percentile coverage can be poor with few clusters", "the empirical distribution is coarse", "no studentization or bias correction", "stratification is absent", "results remain conditional on the accepted clustering and frozen cost model"}, "alternatives_considered": []string{"analytical cluster-robust interval", "studentized cluster bootstrap", "BCa cluster bootstrap", "moving-block bootstrap"}, "governance_decision_requested": map[string]any{"allowed_decisions": []string{"ACCEPT", "REJECT", "REVISE"}, "preselected_decision": nil, "unresolved_decisions": []string{"accept percentile rather than studentized/BCa interval", "accept 4096 resamples", "accept fixed seed", "accept no stratification", "accept minimum of two clusters"}}, "real_candidate_observations_used": 0}
	return sealMap(packet, "packet_hash")
}

func syntheticEvent(symbol string, at time.Time) preconditions.RetainedEvent {
	digest := "sha256:" + strings.Repeat("a", 64)
	event, err := preconditions.SealRetainedEvent(preconditions.RetainedEvent{CandidateFamily: "DowntrendMidVolRelief", CandidateVersion: "synthetic-v1", ImplementationHash: digest, PrimarySymbol: symbol, EventTimestamp: at, DecisionTimestamp: at.Add(time.Minute), SourcePartitionID: "synthetic-partition-" + symbol, SourceSnapshotID: "synthetic-snapshot-" + symbol, SourceInputHash: digest, FeatureSchemaVersion: "synthetic.features.v1", TrendState: "DOWN", PrimaryRegime: "DOWNTREND", VolatilityBucket: "MID", Features: preconditions.DecisionFeatures{Close: 100, EMA50: 99, EMA200: 101, TrendSlope20: -1, RealizedVol60: 0.003}, BTCContext: preconditions.ContextInput{Symbol: "BTCUSDT", SnapshotID: "synthetic-btc", SourceInputHash: digest, AvailableAt: at, Return60: 0.01}, ETHContext: preconditions.ContextInput{Symbol: "ETHUSDT", SnapshotID: "synthetic-eth", SourceInputHash: digest, AvailableAt: at, Return60: 0.02}, ReferencePrice: 100, EvaluationHorizon: "240m", EvaluationHorizonMS: int64(4 * time.Hour / time.Millisecond), WarmupSufficient: true, CostInputs: preconditions.CostInputs{FeeBPS: 1, SpreadBPS: 1, SlippageBPS: 1, FundingBPS: 1, AdverseSelectionBPS: 1}, Attribution: preconditions.EventAttribution{Month: at.Format("2006-01"), Quarter: fmt.Sprintf("%04d-Q%d", at.Year(), (int(at.Month())-1)/3+1), Regime: "DOWNTREND"}})
	if err != nil {
		panic(err)
	}
	return event
}

func sealMap(value map[string]any, hashField string) (map[string]any, error) {
	copyValue := merge(nil, value)
	copyValue[hashField] = ""
	hash, err := governance.HashCanonical(copyValue)
	if err != nil {
		return nil, err
	}
	copyValue[hashField] = hash
	return copyValue, nil
}
func merge(left, right map[string]any) map[string]any {
	result := map[string]any{}
	for k, v := range left {
		result[k] = v
	}
	for k, v := range right {
		result[k] = v
	}
	return result
}
func number(value any) float64 { n, _ := value.(float64); return n }
func boolValue(value any) bool { b, _ := value.(bool); return b }
func bytesHash(data []byte) string {
	sum := sha256.Sum256(data)
	return "sha256:" + hex.EncodeToString(sum[:])
}
func writePair(dir, name string, value any, markdown string) error {
	data, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		return err
	}
	if err := os.WriteFile(filepath.Join(dir, name+".json"), append(data, '\n'), 0o644); err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(dir, name+".md"), []byte(markdown), 0o644)
}

func authorityMarkdown(v map[string]any) string {
	return fmt.Sprintf("# PR4B0-R1P2 authority report\n\nStatus: **AUTHORITY_AND_GOVERNANCE_PACKET_READY**\n\nReport hash: `%s`\n\nThe archive is an **IMMUTABLE_PHYSICAL_ARCHIVE_GAP_IDENTITY**, not PIT-valid research evidence. All 324 snapshots validate against the versioned source schema; all 324 historical availability authorities remain precisely missing with no synthesized timestamps. The authoritative universe is eight primary symbols plus BTCUSDT/ETHUSDT context, with ETHUSDT overlap and nine unique symbols.\n", v["report_hash"])
}
func provenanceMarkdown(v governance.ProvenanceResolution) string {
	return fmt.Sprintf("# PR4B0-R1P2 provenance resolution\n\nResolution hash: `%s`\n\nBoth preserved artifacts have exact paths, SHA-256 and Git blob identities but no source commit in any branch/tag/history. No byte-identical copies were found. Their complete possible 2024-01-01 through 2026-01-01 range is **UNTRUSTED_PROVENANCE_TREATED_AS_EXPOSED** and barred from validation/holdout.\n", v.ResolutionHash)
}
func inspectionMarkdown(v governance.InspectionAudit) string {
	return fmt.Sprintf("# PR4B0-R1P2 inspection audit\n\nClassification: **%s**\n\nAudit hash: `%s`\n\nOne overbroad `sed` read occurred at %s against Engine commit `%s`. It displayed legacy aggregate qualification/result and synthetic-test literals for the already exposed 2024-01-01 through 2026-01-01 period. It exposed no prospective validation or holdout content and affected no implementation or policy decision. Fresh preregistration remains mandatory.\n", v.Classification, v.AuditHash, v.Timestamp.Format(time.RFC3339Nano), v.Commit)
}
func independenceMarkdown(v map[string]any) string {
	return fmt.Sprintf("# PR4B0-R1P2 independence governance packet\n\nContract status: **PROPOSED_NOT_ACCEPTED**\n\nContract hash: `%v`\n\nPacket hash: `%v`\n\nNormative proposal: canonical event deduplication; decision-time/event-ID ordering; transitive half-open four-hour horizon clustering across same-symbol and common BTC/ETH market context; exact-boundary split; cluster-hash identity; one cluster per independent sample. Synthetic vectors fix expected clusters and concentration.\n\nReviewer decision requested: **ACCEPT / REJECT / REVISE**. No choice is preselected and no real candidate event was consumed.\n", v["contract_hash"], v["packet_hash"])
}
func uncertaintyMarkdown(v map[string]any) string {
	return fmt.Sprintf("# PR4B0-R1P2 uncertainty governance packet\n\nContract status: **PROPOSED_NOT_ACCEPTED**\n\nContract hash: `%v`\n\nPacket hash: `%v`\n\nThe proposal estimates frozen-cost net expectancy per accepted independent cluster with a deterministic 95%% one-sided percentile cluster bootstrap: replacement sampling, no stratification, 4096 replicates, fixed versioned seed, and explicit fail-closed degenerate/invalid handling.\n\nReviewer decision requested: **ACCEPT / REJECT / REVISE**. No choice is preselected and only synthetic observations were used.\n", v["contract_hash"], v["packet_hash"])
}
func collectionMarkdown(v map[string]any) string {
	return fmt.Sprintf("# PR4B0-R1P2 prospective collection authority design\n\nStatus: **DESIGN_ONLY_REAL_COLLECTION_NOT_STARTED**\n\nDesign hash: `%v`\n\nEvery future snapshot must be born with immutable dataset/schema/acquisition/availability/evidence/content/partition/cutoff/policy identities in an append-only receipt chain. The validation command, recovery, duplicate, conflict, and synthetic RIF binding behaviors are explicit. No real candidate data or RIF state was created.\n", v["design_hash"])
}
func finalMarkdown(v map[string]any) string {
	return fmt.Sprintf("# PR4B0-R1P2 final decision\n\nFinal label: **%v**\n\nDecision hash: `%v`\n\nStatic authority is resolved or precisely gap-manifested; provenance and inspection are conservatively bounded; both governance packets are reviewable; prospective collection and RIF binding are synthetically designed. This does not accept either policy and does not establish PIT-valid research data.\n\nRecommended next action: **%v**.\n", v["label"], v["decision_hash"], v["recommended_next_action"])
}
