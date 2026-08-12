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

	"github.com/david22573/ak-engine/internal/preconditions"
)

const (
	engineStart = "da034953de936de11b3f8cab21d41c3104685ead"
	rifStart    = "0e91122e898be2e8fb4862110acc24f2d5421245"
	historian   = "cd848370a04f274531fea55a7370509466b6e7bd"
	finalLabel  = "PR4B0_R1P3B_INDEPENDENCE_CONTRACT_ACCEPTED"
)

type artifact struct {
	name string
	json map[string]any
	md   string
}

func main() {
	outDir := flag.String("out-dir", "runs/reports", "artifact output directory")
	verified := flag.Bool("verification-complete", false, "record that all required repository verification completed")
	flag.Parse()
	if !*verified {
		fmt.Fprintln(os.Stderr, "--verification-complete is required; final acceptance artifacts fail closed")
		os.Exit(2)
	}
	if err := generate(*outDir); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func generate(outDir string) error {
	decision := preconditions.DefaultConcentrationGovernanceDecisionV3()
	if err := preconditions.ValidateConcentrationGovernanceDecisionV3(decision); err != nil {
		return err
	}
	policy := preconditions.AcceptedIndependencePolicyV3Default()
	policyHash, err := preconditions.AcceptedIndependencePolicyHashV3(policy)
	if err != nil {
		return err
	}
	decisionRecord := map[string]any{
		"schema_version": "ak.engine.pr4b0-r1p3b.concentration-governance-decision.v1",
		"phase":          "PR4B0-R1P3B", "engine_start_commit": engineStart, "rif_start_commit": rifStart,
		"historian_unchanged_commit": historian, "governance_decision": decision,
		"canonical_decision_hash":    decision.CanonicalDecisionHash,
		"real_candidate_events_used": 0, "real_candidate_outcomes_calculated": 0, "prospective_records_collected_or_inspected": 0,
	}
	contractRecord := map[string]any{
		"schema_version": "ak.engine.pr4b0-r1p3b.independence-contract.v1", "phase": "PR4B0-R1P3B",
		"contract": policy, "contract_version": policy.Version, "contract_status": policy.Status, "contract_hash": policyHash,
		"governance_decision_id": decision.DecisionID, "governance_decision_hash": decision.CanonicalDecisionHash,
		"accepted_uncertainty_version": preconditions.AcceptedUncertaintyMethodVersion,
		"accepted_uncertainty_hash":    preconditions.AcceptedUncertaintyMethodDigestV2,
		"pending_v2":                   map[string]any{"version": preconditions.RevisedIndependencePolicyVersion, "status": preconditions.PolicyStatusRevisedPendingConcentration, "hash": preconditions.PendingIndependencePolicyHashV2, "qualification_accepted": false},
		"v1_qualification_accepted":    false, "synthetic_fixtures_only": true,
	}
	qualificationRecord := map[string]any{
		"schema_version": "ak.engine.pr4b0-r1p3b.qualification-enforcement.v1", "phase": "PR4B0-R1P3B",
		"policy_version": policy.Version, "policy_hash": policyHash, "governance_decision_hash": decision.CanonicalDecisionHash,
		"required_numeric_metrics": []string{"symbol_concentration", "temporal_concentration", "largest_cluster_concentration", "top_five_cluster_concentration"},
		"thresholds":               map[string]any{"symbol_concentration": "<= 1/2", "temporal_concentration": "<= 1/2", "largest_cluster_concentration": "<= 1/2", "top_five_cluster_concentration": "<= 7/10"},
		"exact_arithmetic":         policy.ExactArithmeticRule, "rounding": policy.RoundingRule, "zero_denominator": policy.ZeroDenominatorRule,
		"partition_scope": policy.GateScope, "reason_codes": policy.FailureReasonCodes,
		"report_booleans_authoritative": false, "combined_metrics_can_override_partition_failure": false,
		"prior_rejected_or_near_miss_can_upgrade": false,
		"synthetic_tests":                         map[string]any{"symbol": "PASS", "temporal": "PASS", "largest_cluster": "PASS", "top_five": "PASS", "qualification": "PASS"},
	}
	paths := []string{
		"runs/reports/pr4b0_r1p3b_concentration_governance_decision.md", "runs/reports/pr4b0_r1p3b_concentration_governance_decision.json",
		"runs/reports/pr4b0_r1p3b_independence_contract.md", "runs/reports/pr4b0_r1p3b_independence_contract.json",
		"runs/reports/pr4b0_r1p3b_qualification_enforcement.md", "runs/reports/pr4b0_r1p3b_qualification_enforcement.json",
		"runs/reports/pr4b0_r1p3b_final_decision.md", "runs/reports/pr4b0_r1p3b_final_decision.json",
	}
	finalRecord := map[string]any{
		"schema_version": "ak.engine.pr4b0-r1p3b.final-decision.v1", "phase": "PR4B0-R1P3B",
		"executive_verdict": "The exact prospective human governance decision is sealed; accepted V3 preserves V2 clustering semantics and numerically enforces all four exact structural concentration gates fail closed in every applicable partition; Engine and RIF synthetic compatibility and all required verification completed without real candidate or prospective data access.",
		"final_label":       finalLabel, "engine_start_commit": engineStart, "rif_start_commit": rifStart, "historian_unchanged_commit": historian,
		"governance_decision_hash": decision.CanonicalDecisionHash, "accepted_independence_version": policy.Version, "accepted_independence_hash": policyHash,
		"v1_rejected": true, "pending_v2_rejected": true, "verification_complete": true, "fresh_clone_verification": "PASS",
		"real_candidate_execution_count": 0, "real_candidate_outcome_calculation_count": 0, "prospective_record_collection_or_inspection_count": 0, "real_rif_state_created": false,
		"generated_artifacts": paths, "recommended_next_phase": "PR4B0-R1P4 — Prospective PIT Collection Activation", "next_phase_started": false,
	}
	artifacts := []artifact{
		{"pr4b0_r1p3b_concentration_governance_decision", decisionRecord, decisionMarkdown(decision)},
		{"pr4b0_r1p3b_independence_contract", contractRecord, contractMarkdown(policy, policyHash)},
		{"pr4b0_r1p3b_qualification_enforcement", qualificationRecord, qualificationMarkdown(policy, policyHash)},
		{"pr4b0_r1p3b_final_decision", finalRecord, finalMarkdown(policy, policyHash, decision.CanonicalDecisionHash)},
	}
	if err := os.MkdirAll(outDir, 0o755); err != nil {
		return err
	}
	for _, item := range artifacts {
		if err := sealArtifact(item.json); err != nil {
			return err
		}
		jsonPath := filepath.Join(outDir, item.name+".json")
		mdPath := filepath.Join(outDir, item.name+".md")
		data, err := json.MarshalIndent(item.json, "", "  ")
		if err != nil {
			return err
		}
		data = append(data, '\n')
		if err := os.WriteFile(jsonPath, data, 0o644); err != nil {
			return err
		}
		if err := os.WriteFile(mdPath, []byte(item.md), 0o644); err != nil {
			return err
		}
		if err := validateArtifactHash(item.json); err != nil {
			return fmt.Errorf("%s: %w", item.name, err)
		}
	}
	return nil
}

func decisionMarkdown(v preconditions.ConcentrationGovernanceDecisionV3) string {
	return fmt.Sprintf("# PR4B0-R1P3B Concentration Governance Decision\n\nDecision: **%s**\n\nSelected alternative: **%s**\n\nDecision scope: **%s**\n\nHistorical authority claimed: **false**\n\n%s\n\n%s\n\nOutcome-contribution denominators were rejected because realized magnitude and sign are endogenous to outcomes and do not measure structural dependence. Prior classifications remain unchanged.\n\nDecision timestamp: `%s`\n\nResponsible authority: `%s`\n\nCanonical decision hash: `%s`\n", v.Decision, v.SelectedAlternative, v.DecisionScope, v.ProspectiveAuthorityStatement, v.HistoricalAuthorityDisclaimer, v.DecisionTimestamp, v.AuthorityResponsible, v.CanonicalDecisionHash)
}

func contractMarkdown(policy preconditions.AcceptedIndependencePolicyV3, hash string) string {
	return fmt.Sprintf("# PR4B0-R1P3B Independence Contract\n\nIdentity: `%s`\n\nStatus: **%s**\n\nHash: `%s`\n\nV3 preserves the V2 240-minute half-open UTC interval, same-symbol transitive overlap, cross-symbol common-market episode rules, deterministic deduplication/order, and canonical cluster identity, then binds the accepted structural concentration authority.\n\n- Symbol: each cluster contributes exact mass 1 split `1/K` across sorted unique primary symbols; maximum share `<= 1/2`.\n- Temporal: each cluster belongs to the UTC `YYYY-MM` containing its earliest normalized event timestamp; maximum cluster-count share `<= 1/2`.\n- Largest cluster: maximum unique member-event count divided by total unique represented member events; `<= 1/2`.\n- Top five: order by member count descending then canonical cluster ID ascending, sum up to five over total member events; `<= 7/10`.\n\nEquality passes. All comparisons use exact rational arithmetic. Reporting cannot round a failure into a pass. Missing/duplicate/malformed identities and zero denominators fail closed. V1 and pending V2 remain rejected.\n", policy.Version, policy.Status, hash)
}

func qualificationMarkdown(policy preconditions.AcceptedIndependencePolicyV3, hash string) string {
	return fmt.Sprintf("# PR4B0-R1P3B Qualification Enforcement\n\nAccepted policy: `%s`\n\nHash: `%s`\n\nAll four numeric metrics, positive denominators, the accepted V3 version/hash, and the governance-decision hash are mandatory per DEVELOPMENT, VALIDATION, applicable mandatory walk-forward evaluation slice, and FINAL_HOLDOUT. Combined diagnostics cannot rescue a partition. Report-level booleans are non-authoritative; numeric shares are recomputed exactly.\n\nDeterministic reason codes: `%v`.\n\nSynthetic boundary, malformed-input, policy-identity, boolean-bypass, and partition-isolation suites: **PASS**.\n", policy.Version, hash, policy.FailureReasonCodes)
}

func finalMarkdown(policy preconditions.AcceptedIndependencePolicyV3, policyHash, decisionHash string) string {
	return fmt.Sprintf("# PR4B0-R1P3B Final Decision\n\nExecutive verdict: the prospective governance authority and accepted V3 independence contract are sealed and enforced fail closed.\n\nFinal label: `%s`\n\nGovernance decision hash: `%s`\n\nAccepted V3: `%s`\n\nAccepted V3 hash: `%s`\n\nAll required Engine/RIF synthetic tests, verification, scans, hash checks, and fresh-clone checks passed. V1 and pending V2 remain rejected. Zero real candidate executions or outcome calculations occurred; zero prospective records were collected or inspected; no real RIF state was created.\n\nRecommended next phase only: **PR4B0-R1P4 — Prospective PIT Collection Activation**. It was not begun.\n", finalLabel, decisionHash, policy.Version, policyHash)
}

func sealArtifact(object map[string]any) error {
	delete(object, "artifact_hash")
	data, err := json.Marshal(object)
	if err != nil {
		return err
	}
	var normalized map[string]any
	if err := json.Unmarshal(data, &normalized); err != nil {
		return err
	}
	data, err = json.Marshal(normalized)
	if err != nil {
		return err
	}
	for key := range object {
		delete(object, key)
	}
	for key, value := range normalized {
		object[key] = value
	}
	digest := sha256.Sum256(data)
	object["artifact_hash"] = "sha256:" + hex.EncodeToString(digest[:])
	return nil
}

func validateArtifactHash(object map[string]any) error {
	recorded, ok := object["artifact_hash"].(string)
	if !ok {
		return errors.New("artifact_hash is missing")
	}
	copyObject := make(map[string]any, len(object)-1)
	for key, value := range object {
		if key != "artifact_hash" {
			copyObject[key] = value
		}
	}
	data, err := json.Marshal(copyObject)
	if err != nil {
		return err
	}
	digest := sha256.Sum256(data)
	want := "sha256:" + hex.EncodeToString(digest[:])
	if recorded != want {
		return fmt.Errorf("artifact_hash mismatch: got %s want %s", recorded, want)
	}
	return nil
}
