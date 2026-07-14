package main

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/david22573/ak-engine/internal/preconditions"
)

const (
	finalLabel              = "PR4B0_R1P3A_CONCENTRATION_GOVERNANCE_PACKET_READY"
	engineStart             = "4973a36a8d7a9f236011a4936dbf8229a6e41160"
	historianStart          = "cd848370a04f274531fea55a7370509466b6e7bd"
	rifStart                = "e22357692e5b0e837451805f61c6ec3fb2fd6529"
	acceptedReportCommit    = "205cf59555006ce23fc58bc2c73262660a894850"
	acceptedGeneratorCommit = "25efa97ca89f8dcb724f9872e798bc789123caac"
	pendingIndependenceHash = "sha256:006f19c3f89650f6905931164d6c98ead20800a2346369dadda708cfadf36528"
	acceptedUncertaintyHash = "sha256:1a91541c94378cc6f34e62a39ae504d3d013b5dab63a2b622641cdd1088148fb"
)

type artifact struct {
	base string
	json map[string]any
	md   string
}

func main() {
	out := flag.String("out", "runs/reports", "artifact output directory")
	verification := flag.String("verification-status", "PENDING", "PENDING or PASS")
	engineCommit := flag.String("engine-code-commit", "", "committed Engine implementation revision")
	rifCommit := flag.String("rif-code-commit", "", "committed RIF implementation revision")
	verifyOnly := flag.Bool("verify-only", false, "verify existing artifacts without writing")
	flag.Parse()
	if *verification != "PENDING" && *verification != "PASS" {
		fatal(errors.New("verification-status must be PENDING or PASS"))
	}
	if !validCommit(*engineCommit) || !validCommit(*rifCommit) {
		fatal(errors.New("engine-code-commit and rif-code-commit must be exact lowercase 40-character Git commits"))
	}
	artifacts, err := buildArtifacts(*verification, *engineCommit, *rifCommit)
	if err != nil {
		fatal(err)
	}
	if *verifyOnly {
		fatal(verifyArtifacts(*out, artifacts))
		return
	}
	if err := os.MkdirAll(*out, 0o755); err != nil {
		fatal(err)
	}
	for _, item := range artifacts {
		data, err := json.MarshalIndent(item.json, "", "  ")
		if err != nil {
			fatal(err)
		}
		if err := os.WriteFile(filepath.Join(*out, item.base+".json"), append(data, '\n'), 0o644); err != nil {
			fatal(err)
		}
		if err := os.WriteFile(filepath.Join(*out, item.base+".md"), []byte(item.md), 0o644); err != nil {
			fatal(err)
		}
	}
}

func buildArtifacts(verification, engineCommit, rifCommit string) ([]artifact, error) {
	policies := preconditions.GovernanceConcentrationAlternatives()
	alternatives := make([]map[string]any, 0, len(policies))
	for _, policy := range policies {
		hash, err := preconditions.ConcentrationAlternativeHash(policy)
		if err != nil {
			return nil, err
		}
		alternatives = append(alternatives, map[string]any{
			"version": policy.MetricVersion, "hash": hash, "status": policy.Status, "contract": policy,
			"formulas":  alternativeFormulas(policy.Basis),
			"rationale": alternativeRationale(policy.Basis),
			"clustering_risks": []string{
				"inherits conservative under-clustering risk when distinct context provenance describes one common move",
				"inherits transitive over-clustering risk when same-symbol overlap bridges otherwise distinct context episodes",
			},
			"over_conservatism_risk":                           alternativeRisk(policy.Basis, true),
			"under_conservatism_risk":                          alternativeRisk(policy.Basis, false),
			"sensitivity_to_cluster_size":                      alternativeSensitivity(policy.Basis)["cluster_size"],
			"sensitivity_to_symbol_count":                      alternativeSensitivity(policy.Basis)["symbol_count"],
			"sensitivity_to_sample_size":                       alternativeSensitivity(policy.Basis)["sample_size"],
			"compatibility_with_minimum_300_independent_units": "compatible: the accepted uncertainty gate counts one accepted V2 independent cluster as one unit; N>=300 remains mandatory and concentration is evaluated per required partition",
			"changes_or_replaces_earlier_gate":                 "turns the accepted display-only 50% symbol/temporal limits into fully specified numeric gates and adds explicit 50% largest-cluster and 70% top-five-cluster gates; this is a governance change, not recovered accepted behavior",
		})
	}

	common := map[string]any{
		"phase":       "PR4B0-R1P3A",
		"final_label": finalLabel,
		"starting_commits": map[string]any{
			"engine": engineStart, "historian": historianStart, "rif": rifStart,
		},
		"resulting_code_commits": map[string]any{
			"engine": engineCommit, "historian_unchanged": historianStart, "rif": rifCommit,
			"artifact_container": "the subsequent Engine report-only commit is recorded in the completion response to avoid a self-referential Git hash",
		},
		"research_boundary": map[string]any{
			"real_candidate_executions": 0, "new_candidate_outcome_calculations": 0,
			"prospective_validation_or_holdout_content_inspections": 0, "real_prospective_records_collected": 0,
			"real_rif_state_reads_or_mutations": 0, "fixtures": "synthetic only",
		},
		"verification_status":  verification,
		"verification_results": verificationResults(verification),
	}

	archaeology := cloneMap(common)
	archaeology["schema_version"] = "ak.engine.pr4b0-r1p3a.concentration-archaeology.v1"
	archaeology["scope"] = "all reachable Engine history and refs, accepted report production sources, historical concentration implementations, later governance records, archived patch manifests, and unreachable Git objects"
	archaeology["accepted_report"] = map[string]any{
		"commit":                  acceptedReportCommit,
		"producing_source_commit": acceptedGeneratorCommit,
		"invocation":              "GOWORK=off go run ./cmd/ak-engine pr4b0-candidate-qualification --out-dir runs/reports --resulting-commit 25efa97ca89f8dcb724f9872e798bc789123caac --verification-complete --fresh-clone-commit 25efa97ca89f8dcb724f9872e798bc789123caac",
		"json_artifact_sha256":    "sha256:1caac0af112610905229f2c620919508ac9db3a4fe98d1a9640696c066c52765",
		"embedded_canonical_hash": "sha256:1ada24828f159a386f5337441b347f204f1e29ca5165c70c69ebe4259f4e1252",
	}
	archaeology["git_audit"] = map[string]any{
		"tags": 0, "hidden_authority_refs": 0, "deleted_concentration_files_in_reachable_history": 0,
		"unreachable_commits": 0, "unreachable_trees": 0, "unreachable_tags": 0, "unreachable_blobs": 7,
		"unreachable_concentration_blobs":  2,
		"unreachable_concentration_result": "post-accepted R1P2 proposal variants; explicitly unaccepted and not authority",
		"archived_patch_result":            "PR1 module/test path changes only; no alternate metric semantics or report",
		"archive_manifest_result":          "no Phase 10.3B, fragile-leads, concentration, or cluster report path",
	}
	archaeology["source_records"] = archaeologySourceRecords()
	completeInventory, err := completeConcentrationPathInventory()
	if err != nil {
		return nil, err
	}
	archaeology["complete_path_inventory"] = completeInventory
	archaeology["conclusion"] = "no complete accepted formula-plus-denominator authority exists for any of the four required metrics; Path 2 is mandatory"
	if err := setHash(archaeology); err != nil {
		return nil, err
	}

	matrix := cloneMap(common)
	matrix["schema_version"] = "ak.engine.pr4b0-r1p3a.denominator-authority-matrix.v1"
	matrix["authority_rows"] = authorityRows()
	matrix["path_decision"] = "PATH_2_GOVERNANCE_PACKET_REQUIRED"
	matrix["accepted_contract_created"] = false
	matrix["accepted_independence_version"] = nil
	matrix["accepted_independence_hash"] = nil
	if err := setHash(matrix); err != nil {
		return nil, err
	}

	trace := cloneMap(common)
	trace["schema_version"] = "ak.engine.pr4b0-r1p3a.report-production-trace.v1"
	trace["accepted_report_commit"] = acceptedReportCommit
	trace["producing_source_commit"] = acceptedGeneratorCommit
	trace["generator"] = map[string]any{
		"path": "internal/app/pr4b0_candidate_qualification.go", "git_blob": "59f04150763ea7a180969798e8baf2d816c98959",
		"sha256":    "sha256:e04b57f560aa4ac1c693dccbc95b30a1e9df06a2ae97d53cafcc5abea6d12f23",
		"test_path": "internal/app/pr4b0_candidate_qualification_test.go", "test_git_blob": "39300f9867045cfabdeee7edfc0b0e0664646ae0",
		"test_sha256": "sha256:4afcad1cceb51e2e0a285607b8d4edae97a6b422368617df33e28652c2a0bd6d",
	}
	trace["invocation"] = archaeology["accepted_report"].(map[string]any)["invocation"]
	trace["input_inventory"] = []string{"tracked Phase 10.4 closure JSON", "workspace-hash-verified Phase 10.8/11/12/13 artifacts", "literal candidate inventory rows", "accepted reusable gate-policy declarations"}
	trace["schema_and_gate_registry"] = map[string]any{
		"concentration_schema":              "free-form copied strings plus generic required booleans",
		"required_boolean":                  "GateEvidence.ConcentrationResults must be true when a populated qualification record is evaluated",
		"numeric_evaluator":                 "none in the accepted report generator",
		"largest_and_aggregate_gate_fields": "absent",
		"reason_codes":                      "generic missing/failed gate reasons only; no numeric concentration reason code is emitted by this report path",
	}
	trace["field_traces"] = reportFieldTraces()
	trace["classification_conclusion"] = "the accepted report displayed thresholds and copied historical statuses but did not numerically calculate or enforce any of the four metrics"
	if err := setHash(trace); err != nil {
		return nil, err
	}

	packet := cloneMap(common)
	packet["schema_version"] = "ak.engine.pr4b0-r1p3a.concentration-governance-packet.v1"
	packet["packet_status"] = "DECISION_READY_UNACCEPTED"
	packet["alternatives"] = alternatives
	packet["metric_decision_scope"] = metricDecisionScope()
	packet["reviewer_decision_request"] = map[string]any{
		"allowed_decisions":    []string{"ACCEPT_ALTERNATIVE", "REJECT_ALL", "REVISE"},
		"preselected_decision": nil,
		"accept_requirement":   "identify exactly one complete alternative version/hash; partial or mixed acceptance requires a revised versioned contract",
		"revise_requirement":   "supply exact formula, numerator, denominator, unit, temporal bucket/timezone, symbol attribution, dedup/clustering stage, threshold/operator, rounding, zero-denominator rule, failure semantics, and partition scope",
	}
	packet["current_independence"] = map[string]any{
		"version": preconditions.RevisedIndependencePolicyVersion, "hash": pendingIndependenceHash,
		"status": "REVISED_UNACCEPTED_PENDING_CONCENTRATION_GOVERNANCE",
	}
	packet["accepted_uncertainty"] = map[string]any{"hash": acceptedUncertaintyHash, "minimum_independent_units": 300}
	packet["accepted_contract_created"] = false
	if err := setHash(packet); err != nil {
		return nil, err
	}

	finalDecision := cloneMap(common)
	finalDecision["schema_version"] = "ak.engine.pr4b0-r1p3a.final-decision.v1"
	finalDecision["executive_verdict"] = "The accepted report provenance is fully reconstructed, but accepted denominator authority is incomplete for symbol/temporal concentration and absent for largest/top-five cluster concentration. A decision-ready two-alternative packet is ready; no alternative is accepted."
	finalDecision["final_label"] = finalLabel
	finalDecision["authority_result"] = "NO_COMPLETE_ACCEPTED_CONCENTRATION_CONTRACT"
	finalDecision["governance_packet_status"] = "READY_FOR_USER_DECISION"
	finalDecision["accepted_independence_version"] = nil
	finalDecision["accepted_independence_hash"] = nil
	finalDecision["pending_independence_version"] = preconditions.RevisedIndependencePolicyVersion
	finalDecision["pending_independence_hash"] = pendingIndependenceHash
	finalDecision["accepted_uncertainty_hash"] = acceptedUncertaintyHash
	finalDecision["engine_qualification_boundary"] = "fail closed: all-pass report booleans return CONCENTRATION_AUTHORITY_MISSING, and no accepted independence-policy hash is registered for V2 candidate registration"
	finalDecision["rif_real_lifecycle_boundary"] = "fail closed: persistent V2 registration requires an explicitly configured governance verifier and rejects pending/unapproved identity without state mutation"
	finalDecision["fail_closed_proofs"] = map[string]any{
		"report_only_all_pass_cannot_qualify":                       "PASS",
		"missing_or_zero_denominator_cannot_default_to_zero":        "PASS",
		"missing_or_mutated_threshold_cannot_default_to_pass":       "PASS",
		"legacy_rejected_near_miss_and_probe_labels_cannot_upgrade": "PASS",
		"pending_v2_registration_cannot_reach_rif":                  "PASS",
	}
	finalDecision["prospective_collection_authorized"] = false
	finalDecision["candidate_rerun_authorized"] = false
	finalDecision["next_phase"] = "USER_CONCENTRATION_GOVERNANCE_DECISION_REQUIRED"
	if err := setHash(finalDecision); err != nil {
		return nil, err
	}

	return []artifact{
		{"pr4b0_r1p3a_concentration_archaeology", archaeology, archaeologyMarkdown(archaeology)},
		{"pr4b0_r1p3a_denominator_authority_matrix", matrix, matrixMarkdown(matrix)},
		{"pr4b0_r1p3a_report_production_trace", trace, traceMarkdown(trace)},
		{"pr4b0_r1p3a_concentration_governance_packet", packet, packetMarkdown(packet, alternatives)},
		{"pr4b0_r1p3a_final_decision", finalDecision, finalMarkdown(finalDecision)},
	}, nil
}

func archaeologySourceRecords() []map[string]any {
	return []map[string]any{
		source("runs/reports/pr4b0_candidate_qualification.json", acceptedReportCommit, "429166feee0ddda39811f202ff1f4c1f68cec2e2", "sha256:1caac0af112610905229f2c620919508ac9db3a4fe98d1a9640696c066c52765", "accepted report artifact", "all", "50% symbol, 50% temporal, no cluster threshold", "none; copied/displayed fields", "none", "report inventory", "accepted PR4B0", "display/missing only", true, "ACCEPTED_AUTHORITY_PARTIAL", "THRESHOLD_DISPLAYED_ONLY"),
		source("internal/app/pr4b0_candidate_qualification.go", acceptedGeneratorCommit, "59f04150763ea7a180969798e8baf2d816c98959", "sha256:e04b57f560aa4ac1c693dccbc95b30a1e9df06a2ae97d53cafcc5abea6d12f23", "accepted report generator", "symbol, temporal", "50% each", "not calculated", "not calculated", "candidate inventory row", "accepted PR4B0", "generic boolean fail-closed when populated", true, "ACCEPTED_AUTHORITY_PARTIAL", "THRESHOLD_DISPLAYED_ONLY"),
		source("internal/app/testdata/phase10_4_price_regime_branch_closure.json", acceptedReportCommit, "f1e97ceec2e32cea430f7d0fdaff6e7cd1f6af5c", "sha256:1c811d816ea744859332f639c86add2f8ade6fb5419a7c32331a92444d74f396", "accepted input evidence", "temporal status", "none", "status string only", "none", "historical candidate status", "Phase 10.4", "concentration_failed/top_month_concentration labels", true, "HISTORICAL_SUPPORT_ONLY", "THRESHOLD_INFERRED_FROM_STATUS"),
		source("internal/app/testdata/phase10_4_research_guardrails.md", acceptedReportCommit, "6a2ac723ced655696d04d047ec2a828a4fe12e78", "sha256:b24ef3a0ea899d364030c1279eb0beb07ffb1ed70e58429136ac987d05d2e1cb", "guardrail prose", "temporal, cluster", "none", "unspecified", "unspecified", "unspecified", "Phase 10.4", "calls metrics rejection gates without semantics", false, "HISTORICAL_SUPPORT_ONLY", "THRESHOLD_INFERRED_FROM_STATUS"),
		source("internal/app/analyze_fragile_leads.go", acceptedReportCommit, "67a392556e61418501a5ba2edb5e85d09a67202b", "sha256:c17f3f08e000015446f0744608667eeab4d4c1c86ce9ce6ab42e0322d86e7854", "historical implementation", "temporal, largest, top five", "50%, 50%, 70%", "top positive signed-net contribution", "signed total net; nonpositive fallback", "UTC month or 60-minute adjacency cluster", "Phase 10.3B", "temporal gates final status; cluster only descriptive classification", false, "HISTORICAL_SUPPORT_ONLY", "THRESHOLD_PRESENT_BUT_UNUSED"),
		source("internal/app/analyze_compact_robustness.go", acceptedReportCommit, "e8f518b873c7365488f8685cb39f7a1994c8ac18", "sha256:c4faed26dcafda5044fc47f8031c9e754074a0b7f821716504f7cd36ead06c85", "historical implementation", "symbol, month, quarter, bucket", "50%, 40%, 60%, 60%", "largest positive dimension net", "sum positive dimension net", "combined retained-summary dimension", "Phase 10.8", "symbol/month fail; quarter/bucket warn", false, "HISTORICAL_SUPPORT_ONLY", "THRESHOLD_PRESENT_BUT_UNUSED"),
		source("phase10_8_ranked_inventory.json", "UNTRACKED_WORKSPACE_ARTIFACT", "UNAVAILABLE", "sha256:ebf255c9bcb8e317f11efb58147cbdaed9c269699f268d646b6221ef29ef4f4d", "hash-matched historical workspace input", "symbol, month, quarter, bucket", "implicit producer defaults; no embedded contract", "reported positive contribution", "reported positive total", "retained-summary dimension", "Phase 10.8", "omitted from accepted report concentration fields", true, "HISTORICAL_SUPPORT_ONLY", "THRESHOLD_PRESENT_BUT_UNUSED"),
		source("internal/app/evaluate_alpha_baselines_multisymbol.go", acceptedReportCommit, "d4f820dae017ac3aa74c619d20756f2931088946", "sha256:25a074e253d46e7c638dad31d1be2d4d7d997c021aa7f0778ac5e669de0566be", "historical implementation", "temporal", "top one 50%, top two 70%", "positive month net after 5 bps", "total positive month net", "UTC month per symbol/family/side", "alpha baseline", "strict greater-than rejects", false, "HISTORICAL_SUPPORT_ONLY", "THRESHOLD_PRESENT_BUT_UNUSED"),
		source("internal/app/funding_aggregation.go", acceptedReportCommit, "71271dba7096ab076405958573ef8bc68140ebf0", "sha256:7c71b0cad66600d569bfaadf08ee9416d50171b68d6c03ff8777398d8847bee9", "historical implementation", "temporal, largest, top five", "50%, 50%, 70%", "positive month return or cluster member count", "positive total return or raw event count", "UTC month; 60-minute composite-key cluster", "funding aggregation", "temporal enforced; clusters descriptive", false, "HISTORICAL_SUPPORT_ONLY", "THRESHOLD_PRESENT_BUT_UNUSED"),
		source("internal/app/evaluate_funding_candidate_deep.go", acceptedReportCommit, "7acaef39a699f9836b243d98aa57ebef6bb76e95", "sha256:8fd3a4d4bb157535a0b008edfd675d2d6291ca96698fa4e85289b503ab6746eb", "historical consumer", "temporal", "top one 50%, top two 70%", "supplied row", "supplied row", "funding candidate row", "funding deep evaluation", "strict greater-than rejects", false, "HISTORICAL_SUPPORT_ONLY", "THRESHOLD_PRESENT_BUT_UNUSED"),
		source("runs/reports/pr4b0_r1_research_protocol.json", engineStart, "TRACKED_AT_ENGINE_START", "AVAILABLE_BY_GIT_OBJECT_VALIDATION", "later governance record", "symbol, temporal", "50% each", "missing", "missing", "future protocol declaration", "R1", "blocked/non-executable", false, "ACCEPTED_AUTHORITY_PARTIAL", "THRESHOLD_DISPLAYED_ONLY"),
		source("runs/reports/pr4b0_r1p2_independence_governance_packet.json", engineStart, "TRACKED_AT_ENGINE_START", "AVAILABLE_BY_GIT_OBJECT_VALIDATION", "later proposal", "largest cluster", "none accepted", "member-event count", "member-event count", "proposed V1 cluster", "R1P2", "explicitly proposed not accepted", false, "HISTORICAL_SUPPORT_ONLY", "THRESHOLD_NOT_RECOVERABLE"),
		source("runs/reports/pr4b0_r1p3_independence_decision.json", engineStart, "TRACKED_AT_ENGINE_START", "AVAILABLE_BY_GIT_OBJECT_VALIDATION", "immediate predecessor governance decision", "all", "symbol/temporal threshold only", "unresolved", "unresolved", "accepted V2 clusters", "R1P3", "V2 revised but unaccepted", false, "HISTORICAL_SUPPORT_ONLY", "THRESHOLD_NOT_RECOVERABLE"),
		source("unreachable blobs 26b1c157... and 3f1aee55...", "UNREACHABLE_POST_ACCEPTED_OBJECTS", "26b1c157...;3f1aee55...", "AVAILABLE_BY_OBJECT_INSPECTION", "discarded proposal variants", "largest cluster", "explicitly unaccepted", "member-event count", "member-event count", "proposed cluster", "R1P2 pre-commit", "never accepted or reachable", false, "NO_AUTHORITY_FOUND", "THRESHOLD_NOT_RECOVERABLE"),
	}
}

func source(path, commit, blob, digest, relationship, metric, threshold, numerator, denominator, unit, phase, failure string, used bool, authority, trace string) map[string]any {
	return map[string]any{
		"path": path, "source_commit": commit, "git_blob": blob, "sha256": digest,
		"relationship_to_accepted_report": relationship, "metric": metric, "threshold": threshold,
		"numerator_definition": numerator, "denominator_definition": denominator, "aggregation_unit": unit,
		"period": "as stated in record; UTC where explicitly identified", "candidate_scope": "record-specific; no cross-scope inference",
		"phase": phase, "failure_semantics": failure, "used_by_accepted_run": used,
		"authority_status": authority, "trace_status": trace,
	}
}

func completeConcentrationPathInventory() ([]map[string]any, error) {
	paths := []string{
		"cmd/pr4b0-r1p-report/main.go",
		"cmd/pr4b0-r1p2-report/main.go",
		"cmd/pr4b0-r1p2-report/main_test.go",
		"cmd/pr4b0-r1p3-report/main.go",
		"internal/app/analyze_compact_robustness.go",
		"internal/app/analyze_compact_robustness_test.go",
		"internal/app/analyze_fragile_leads.go",
		"internal/app/evaluate_alpha_baselines_multisymbol_test.go",
		"internal/app/evaluate_funding_candidate_deep.go",
		"internal/app/funding_aggregation.go",
		"internal/app/pr4b0_candidate_qualification.go",
		"internal/app/testdata/phase10_4_price_regime_branch_closure.json",
		"internal/app/testdata/phase10_4_research_guardrails.md",
		"internal/preconditions/clustering.go",
		"internal/preconditions/independence_v2.go",
		"internal/preconditions/independence_v2_test.go",
		"internal/preconditions/preconditions_test.go",
		"internal/qualification/qualification.go",
		"internal/qualification/qualification_test.go",
		"progress.md",
		"runs/reports/pr4b0_candidate_inventory.json",
		"runs/reports/pr4b0_candidate_inventory.md",
		"runs/reports/pr4b0_candidate_qualification.json",
		"runs/reports/pr4b0_candidate_qualification.md",
		"runs/reports/pr4b0_r1_evidence_supplement.json",
		"runs/reports/pr4b0_r1_evidence_supplement.md",
		"runs/reports/pr4b0_r1_final_decision.json",
		"runs/reports/pr4b0_r1_final_decision.md",
		"runs/reports/pr4b0_r1_research_protocol.json",
		"runs/reports/pr4b0_r1_research_protocol.md",
		"runs/reports/pr4b0_r1_variant_results.json",
		"runs/reports/pr4b0_r1_variant_results.md",
		"runs/reports/pr4b0_r1p2_independence_governance_packet.json",
		"runs/reports/pr4b0_r1p2_independence_governance_packet.md",
		"runs/reports/pr4b0_r1p2_provenance_resolution.json",
		"runs/reports/pr4b0_r1p3_final_decision.json",
		"runs/reports/pr4b0_r1p3_final_decision.md",
		"runs/reports/pr4b0_r1p3_governance_report.json",
		"runs/reports/pr4b0_r1p3_governance_report.md",
		"runs/reports/pr4b0_r1p3_independence_decision.json",
		"runs/reports/pr4b0_r1p3_independence_decision.md",
		"runs/reports/pr4b0_r1p_prior_exposure_ledger.json",
	}
	records := make([]map[string]any, 0, len(paths))
	for _, path := range paths {
		blob, digest, err := gitObjectIdentity(engineStart, path)
		if err != nil {
			return nil, err
		}
		records = append(records, source(path, engineStart, blob, digest,
			"complete concentration-bearing path inventory at the accepted Engine starting commit; primary semantic records control authority classification",
			"concentration-related support, mirror, test, generator, protocol, or report", "NO_INDEPENDENT_THRESHOLD_INFERENCE",
			"SEE_PRIMARY_SEMANTIC_RECORD_IF_APPLICABLE", "SEE_PRIMARY_SEMANTIC_RECORD_IF_APPLICABLE",
			"PATH_SPECIFIC", "accepted start snapshot", "no authority inferred from path membership alone", false,
			"HISTORICAL_SUPPORT_ONLY", "THRESHOLD_NOT_RECOVERABLE"))
	}
	return records, nil
}

func gitObjectIdentity(commit, path string) (string, string, error) {
	blobBytes, err := exec.Command("git", "rev-parse", commit+":"+path).Output()
	if err != nil {
		return "", "", fmt.Errorf("resolve Git blob for %s: %w", path, err)
	}
	content, err := exec.Command("git", "show", commit+":"+path).Output()
	if err != nil {
		return "", "", fmt.Errorf("read Git blob for %s: %w", path, err)
	}
	digest := sha256.Sum256(content)
	return strings.TrimSpace(string(blobBytes)), "sha256:" + hex.EncodeToString(digest[:]), nil
}

func authorityRows() []map[string]any {
	return []map[string]any{
		metricRow("symbol_concentration", "ACCEPTED_AUTHORITY_PARTIAL", "50%", "<=", "generic required boolean only", []string{"numerator", "denominator", "counting unit", "symbol attribution", "deduplication/clustering stage", "rounding", "partition/combined scope"}),
		metricRow("temporal_concentration", "ACCEPTED_AUTHORITY_PARTIAL", "50%", "<=", "generic required boolean only", []string{"numerator", "denominator", "counting unit", "bucket type", "timezone", "empty bucket handling", "deduplication/clustering stage", "rounding", "partition/combined scope"}),
		metricRow("largest_cluster_concentration", "NO_AUTHORITY_FOUND", "NOT_RECOVERED", "NOT_RECOVERED", "not an accepted report gate", []string{"formula", "numerator", "denominator", "counting unit", "cluster identity binding", "threshold", "operator", "rounding", "failure semantics", "scope"}),
		metricRow("aggregate_cluster_concentration", "NO_AUTHORITY_FOUND", "NOT_RECOVERED", "NOT_RECOVERED", "not an accepted report gate", []string{"formula", "top-N definition", "tie handling", "numerator", "denominator", "counting unit", "cluster identity binding", "threshold", "operator", "rounding", "failure semantics", "scope"}),
	}
}

func metricRow(metric, status, threshold, operator, failure string, missing []string) map[string]any {
	sourceCommit := acceptedGeneratorCommit
	sourcePath := "internal/app/pr4b0_candidate_qualification.go"
	sourceHash := "sha256:e04b57f560aa4ac1c693dccbc95b30a1e9df06a2ae97d53cafcc5abea6d12f23"
	if status == "NO_AUTHORITY_FOUND" {
		sourceCommit, sourcePath, sourceHash = "NOT_RECOVERED", "NOT_RECOVERED", "NOT_RECOVERED"
	}
	return map[string]any{
		"metric_id": metric, "metric_version": "NOT_RECOVERED", "numerator_definition": "NOT_RECOVERED",
		"denominator_definition": "NOT_RECOVERED", "counting_unit": "NOT_RECOVERED",
		"deduplication_stage": "NOT_RECOVERED", "clustering_stage": "NOT_RECOVERED",
		"partition_scope": "NOT_RECOVERED", "combined_scope": "NOT_RECOVERED",
		"time_bucket": "NOT_RECOVERED", "timezone": "NOT_RECOVERED", "empty_bucket_handling": "NOT_RECOVERED",
		"symbol_attribution": "NOT_RECOVERED", "threshold": threshold, "comparison_operator": operator,
		"rounding_rule": "NOT_RECOVERED", "failure_code": failure,
		"authority_source_commit": sourceCommit, "authority_source_path": sourcePath,
		"authority_source_hash": sourceHash, "authority_status": status, "missing_authority": missing,
	}
}

func reportFieldTraces() []map[string]any {
	return []map[string]any{
		{"metric": "symbol_concentration", "schema_field": "candidate concentration_results strings / displayed gate policy", "calculation_path": "none", "accepted_threshold": "50% displayed", "reason_code": "none numeric", "test_proof": "literal output and generic structure", "trace_status": "THRESHOLD_DISPLAYED_ONLY"},
		{"metric": "temporal_concentration", "schema_field": "candidate concentration_results strings / displayed gate policy", "calculation_path": "none; Phase 10.4 status copied", "accepted_threshold": "50% displayed", "reason_code": "none numeric", "test_proof": "literal output and generic structure", "trace_status": "THRESHOLD_DISPLAYED_ONLY"},
		{"metric": "largest_cluster_concentration", "schema_field": "absent; only average_events_per_cluster and cluster_pnl_available appear for one row", "calculation_path": "none", "accepted_threshold": "none", "reason_code": "none", "test_proof": "absence from generator and gate registry", "trace_status": "THRESHOLD_NOT_RECOVERABLE"},
		{"metric": "aggregate_cluster_concentration", "schema_field": "absent", "calculation_path": "none", "accepted_threshold": "none", "reason_code": "none", "test_proof": "absence from generator and gate registry", "trace_status": "THRESHOLD_NOT_RECOVERABLE"},
		{"metric": "phase10_4_status_label", "schema_field": "concentration_failed/top_month_concentration", "calculation_path": "copied string", "accepted_threshold": "not encoded", "reason_code": "historical label", "test_proof": "fixture only", "trace_status": "THRESHOLD_INFERRED_FROM_STATUS"},
		{"metric": "phase10_8_concentration", "schema_field": "present in hash-matched input but omitted from accepted candidate rows", "calculation_path": "historical producer defaults", "accepted_threshold": "50/40/60 conflict", "reason_code": "historical", "test_proof": "not used by accepted report", "trace_status": "THRESHOLD_PRESENT_BUT_UNUSED"},
	}
}

func metricDecisionScope() []map[string]any {
	return []map[string]any{
		{"metric": "symbol_concentration", "choices": []string{preconditions.ConcentrationAlternativeCountV1, preconditions.ConcentrationAlternativeReturnV1}, "unresolved_authority": "numerator, denominator, unit, attribution, stage, rounding, scope"},
		{"metric": "temporal_concentration", "choices": []string{preconditions.ConcentrationAlternativeCountV1, preconditions.ConcentrationAlternativeReturnV1}, "unresolved_authority": "numerator, denominator, unit, bucket/timezone, empty periods, stage, rounding, scope"},
		{"metric": "largest_cluster_concentration", "choices": []string{preconditions.ConcentrationAlternativeCountV1, preconditions.ConcentrationAlternativeReturnV1}, "unresolved_authority": "complete accepted metric and threshold authority absent"},
		{"metric": "aggregate_cluster_concentration", "choices": []string{preconditions.ConcentrationAlternativeCountV1, preconditions.ConcentrationAlternativeReturnV1}, "unresolved_authority": "complete accepted top-five metric and threshold authority absent"},
	}
}

func alternativeRationale(basis preconditions.ConcentrationBasis) string {
	if basis == preconditions.ConcentrationBasisClusterCount {
		return "uses accepted V2 clusters as the primary independence unit and member-event share only for within-cluster dominance, minimizing dependence on return scale"
	}
	return "measures concentration in the same positive mandatory-cost net-return mass that motivates economic qualification, while retaining accepted V2 clusters as the independent unit"
}

func alternativeFormulas(basis preconditions.ConcentrationBasis) map[string]any {
	if basis == preconditions.ConcentrationBasisClusterCount {
		return map[string]any{
			"symbol":            "max fractionally attributed independent-cluster count / independent-cluster count * 100",
			"temporal":          "max independent-cluster count assigned by earliest event to one UTC calendar month / independent-cluster count * 100",
			"largest_cluster":   "largest deduplicated member-event count / all deduplicated member-event count * 100",
			"aggregate_cluster": "deduplicated member-event count in five largest clusters / all deduplicated member-event count * 100",
		}
	}
	return map[string]any{
		"symbol":            "max fractionally attributed positive mandatory-cost cluster net return / total positive cluster net return * 100",
		"temporal":          "max positive cluster net return assigned by earliest event to one UTC calendar month / total positive cluster net return * 100",
		"largest_cluster":   "largest positive mandatory-cost cluster net return / total positive cluster net return * 100",
		"aggregate_cluster": "five largest positive mandatory-cost cluster net returns / total positive cluster net return * 100",
	}
}

func alternativeRisk(basis preconditions.ConcentrationBasis, over bool) string {
	if basis == preconditions.ConcentrationBasisClusterCount {
		if over {
			return "large multi-event clusters can fail member-share gates even when each event has small economic weight"
		}
		return "many singleton clusters can pass count-based gates while a small subset carries most economic return"
	}
	if over {
		return "one legitimately large profitable cluster can fail even when event/cluster counts are broadly diversified"
	}
	return "excluding zero and negative clusters can hide concentration in gross exposure or loss-producing dependence"
}

func alternativeSensitivity(basis preconditions.ConcentrationBasis) map[string]string {
	if basis == preconditions.ConcentrationBasisClusterCount {
		return map[string]string{
			"cluster_size": "largest/top-five metrics change directly with deduplicated members per cluster; symbol/temporal metrics change with cluster splitting or merging",
			"symbol_count": "one cluster is fractionally divided across its distinct symbols, so attribution sums to one regardless of symbol count",
			"sample_size":  "percentages are discrete at 1/N cluster increments; N>=300 reduces but does not eliminate threshold granularity",
		}
	}
	return map[string]string{
		"cluster_size": "member count does not affect the metric directly, but cluster splitting/merging reallocates positive return mass",
		"symbol_count": "positive return is fractionally divided across distinct symbols, conserving return mass while diluting multi-symbol attribution",
		"sample_size":  "N>=300 remains required, but effective denominator mass can still be concentrated in few positive-return clusters",
	}
}

func verificationResults(status string) map[string]any {
	return map[string]any{
		"engine":                map[string]any{"gofmt": status, "go_mod_tidy_diff": status, "go_vet": status, "go_test": status, "go_test_race": status, "go_build": status, "make_verify": status, "git_diff_check": status, "standalone_fresh_clone": status},
		"rif":                   map[string]any{"gofmt": status, "go_mod_tidy_diff": status, "go_vet": status, "go_test": status, "go_test_race": status, "go_build": status, "make_verify": status, "git_diff_check": status, "standalone_fresh_clone": status},
		"historian":             map[string]any{"unchanged": status, "clean_worktree": status},
		"artifacts":             map[string]any{"json_parse": status, "embedded_hashes": status, "recorded_sha256": status, "git_blob_ids": status, "generator_verify_only": status},
		"scans":                 map[string]any{"secrets_credentials": status, "absolute_paths": status, "sibling_dependencies": status, "trader_imports": status, "network_dependencies": status, "prohibited_real_candidate_identity": status},
		"zero_prospective_work": status,
	}
}

func archaeologyMarkdown(v map[string]any) string {
	return fmt.Sprintf("# PR4B0-R1P3A Concentration Archaeology\n\nResult: **no complete accepted concentration formula-plus-denominator authority was recovered**.\n\nThe accepted report at `%s` was produced from source `%s`. Its recorded invocation, source/blob identities, reachable and unreachable Git archaeology, archived-patch checks, historical formulas, and accepted-run relationships are machine-recorded in the paired JSON.\n\nPath 2 is mandatory. No real candidate run, outcome calculation, prospective content inspection, or real RIF state access occurred.\n\nVerification: `%v`\n\nArtifact hash: `%v`\n", acceptedReportCommit, acceptedGeneratorCommit, v["verification_status"], v["artifact_hash"])
}

func matrixMarkdown(v map[string]any) string {
	return fmt.Sprintf("# PR4B0-R1P3A Denominator Authority Matrix\n\n- Symbol concentration: **ACCEPTED_AUTHORITY_PARTIAL**.\n- Temporal concentration: **ACCEPTED_AUTHORITY_PARTIAL**.\n- Largest-cluster concentration: **NO_AUTHORITY_FOUND**.\n- Aggregate/top-five cluster concentration: **NO_AUTHORITY_FOUND**.\n\nEvery required formula, numerator, denominator, unit, temporal, symbol-attribution, stage, threshold/operator, rounding, failure, and scope field is explicit in the paired JSON; unrecovered fields are marked `NOT_RECOVERED`.\n\nDecision path: **PATH_2_GOVERNANCE_PACKET_REQUIRED**. No accepted contract or independence hash was created.\n\nVerification: `%v`\n\nArtifact hash: `%v`\n", v["verification_status"], v["artifact_hash"])
}

func traceMarkdown(v map[string]any) string {
	return fmt.Sprintf("# PR4B0-R1P3A Accepted Report Production Trace\n\nThe exact accepted generator displayed 50%% symbol and temporal limits, copied or omitted historical concentration strings, and had no numeric evaluator for the four required metrics. Largest and aggregate cluster gates were absent.\n\nClassifications: symbol/temporal `THRESHOLD_DISPLAYED_ONLY`; largest/aggregate `THRESHOLD_NOT_RECOVERABLE`; Phase 10.4 status `THRESHOLD_INFERRED_FROM_STATUS`; Phase 10.8 `THRESHOLD_PRESENT_BUT_UNUSED`.\n\nProducing source commit: `%s`\n\nVerification: `%v`\n\nArtifact hash: `%v`\n", acceptedGeneratorCommit, v["verification_status"], v["artifact_hash"])
}

func packetMarkdown(v map[string]any, alternatives []map[string]any) string {
	return fmt.Sprintf("# PR4B0-R1P3A Concentration Governance Packet\n\nStatus: **DECISION_READY_UNACCEPTED**.\n\nTwo holistic alternatives are supplied for every unresolved metric:\n\n1. `%v` — independent-cluster count plus member-event cluster shares; hash `%v`.\n2. `%v` — positive independent-cluster net-return contribution; hash `%v`.\n\nBoth use 50%% symbol, 50%% temporal, 50%% largest-cluster, and 70%% top-five-cluster maximums; equality passes, comparison uses unrounded values, reporting uses six decimals, and invalid/zero-denominator evidence fails closed. Neither is preselected or accepted.\n\nReviewer decision required: **ACCEPT_ALTERNATIVE**, **REJECT_ALL**, or **REVISE**.\n\nVerification: `%v`\n\nArtifact hash: `%v`\n", alternatives[0]["version"], alternatives[0]["hash"], alternatives[1]["version"], alternatives[1]["hash"], v["verification_status"], v["artifact_hash"])
}

func finalMarkdown(v map[string]any) string {
	return fmt.Sprintf("# PR4B0-R1P3A Final Decision\n\nExecutive verdict: %v\n\nFinal label: `%s`\n\nNo concentration alternative is accepted and no accepted independence version/hash was created. Engine V2 registration and RIF persistent V2 lifecycle registration remain fail closed. Prospective collection and candidate rerun remain unauthorized.\n\nNext phase: **USER_CONCENTRATION_GOVERNANCE_DECISION_REQUIRED**.\n\nVerification: `%v`\n\nArtifact hash: `%v`\n", v["executive_verdict"], finalLabel, v["verification_status"], v["artifact_hash"])
}

func verifyArtifacts(dir string, expected []artifact) error {
	for _, item := range expected {
		jsonPath := filepath.Join(dir, item.base+".json")
		data, err := os.ReadFile(jsonPath)
		if err != nil {
			return err
		}
		var actual map[string]any
		decoder := json.NewDecoder(strings.NewReader(string(data)))
		decoder.UseNumber()
		if err := decoder.Decode(&actual); err != nil {
			return fmt.Errorf("%s: %w", jsonPath, err)
		}
		if err := verifyHash(actual); err != nil {
			return fmt.Errorf("%s: %w", jsonPath, err)
		}
		want, err := canonicalBytes(item.json, false)
		if err != nil {
			return err
		}
		got, err := canonicalBytes(actual, false)
		if err != nil {
			return err
		}
		if string(want) != string(got) {
			return fmt.Errorf("%s does not match generated contract", jsonPath)
		}
		markdown, err := os.ReadFile(filepath.Join(dir, item.base+".md"))
		if err != nil {
			return err
		}
		if string(markdown) != item.md {
			return fmt.Errorf("%s does not match generated report", item.base+".md")
		}
	}
	return nil
}

func setHash(object map[string]any) error {
	data, err := canonicalBytes(object, true)
	if err != nil {
		return err
	}
	digest := sha256.Sum256(data)
	object["artifact_hash"] = "sha256:" + hex.EncodeToString(digest[:])
	return nil
}

func verifyHash(object map[string]any) error {
	recorded, ok := object["artifact_hash"].(string)
	if !ok || recorded == "" {
		return errors.New("artifact_hash is missing")
	}
	data, err := canonicalBytes(object, true)
	if err != nil {
		return err
	}
	digest := sha256.Sum256(data)
	if want := "sha256:" + hex.EncodeToString(digest[:]); recorded != want {
		return fmt.Errorf("artifact_hash mismatch: got %s want %s", recorded, want)
	}
	return nil
}

func canonicalBytes(object map[string]any, excludeHash bool) ([]byte, error) {
	data, err := json.Marshal(object)
	if err != nil {
		return nil, err
	}
	decoder := json.NewDecoder(strings.NewReader(string(data)))
	decoder.UseNumber()
	var normalized map[string]any
	if err := decoder.Decode(&normalized); err != nil {
		return nil, err
	}
	if excludeHash {
		delete(normalized, "artifact_hash")
	}
	return json.Marshal(normalized)
}

func cloneMap(input map[string]any) map[string]any {
	output := make(map[string]any, len(input))
	for key, value := range input {
		output[key] = value
	}
	return output
}

func validCommit(value string) bool {
	if len(value) != 40 || strings.ToLower(value) != value {
		return false
	}
	_, err := hex.DecodeString(value)
	return err == nil
}

func fatal(err error) {
	if err == nil {
		return
	}
	fmt.Fprintln(os.Stderr, strings.TrimSpace(err.Error()))
	os.Exit(1)
}
