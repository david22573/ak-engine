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
	"sort"
	"strings"

	"github.com/david22573/ak-engine/internal/preconditions"
)

const (
	decisionTimestamp = "2026-07-13T19:00:00Z"
	partialLabel      = "PR4B0_R1P3_GOVERNANCE_PARTIALLY_RESOLVED"
	engineStart       = "c009664dc6778ad2b8f257bc7d0f881c2387ec75"
	historianStart    = "cd848370a04f274531fea55a7370509466b6e7bd"
	rifStart          = "9a2865e13071f39a1d24524bfe5f97c0f2f717cd"
	sourceSchemaHash  = "sha256:4ff5ef49773e3d9a65d50e64d3a7d3ecc6a50d32699c8af1de41ed3518cc99c5"
	manifestHash      = "sha256:a07c34075721250db17e56a13d55d66cbcea934d013866305b61506a37323882"
)

type artifact struct {
	base string
	json map[string]any
	md   string
}

func main() {
	out := flag.String("out", "runs/reports", "artifact output directory")
	verification := flag.String("verification-status", "PENDING", "PENDING or PASS")
	verifyOnly := flag.Bool("verify-only", false, "verify existing artifacts without writing")
	flag.Parse()
	if *verification != "PENDING" && *verification != "PASS" {
		fatal(errors.New("verification-status must be PENDING or PASS"))
	}
	artifacts, err := buildArtifacts(*verification)
	if err != nil {
		fatal(err)
	}
	if *verifyOnly {
		if err := verifyArtifacts(*out, artifacts); err != nil {
			fatal(err)
		}
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
		data = append(data, '\n')
		if err := os.WriteFile(filepath.Join(*out, item.base+".json"), data, 0o644); err != nil {
			fatal(err)
		}
		if err := os.WriteFile(filepath.Join(*out, item.base+".md"), []byte(item.md), 0o644); err != nil {
			fatal(err)
		}
	}
}

func verifyArtifacts(dir string, artifacts []artifact) error {
	for _, expected := range artifacts {
		jsonPath := filepath.Join(dir, expected.base+".json")
		data, err := os.ReadFile(jsonPath)
		if err != nil {
			return err
		}
		var actual map[string]any
		if err := json.Unmarshal(data, &actual); err != nil {
			return fmt.Errorf("%s: %w", jsonPath, err)
		}
		if err := verifyHash(actual, "artifact_hash"); err != nil {
			return fmt.Errorf("%s: %w", jsonPath, err)
		}
		if _, decision := actual["decision_record_hash"]; decision {
			if err := verifyHash(actual, "decision_record_hash"); err != nil {
				return fmt.Errorf("%s: %w", jsonPath, err)
			}
		}
		expectedJSON, err := canonicalMapBytes(expected.json)
		if err != nil {
			return err
		}
		actualJSON, err := canonicalMapBytes(actual)
		if err != nil {
			return err
		}
		if string(expectedJSON) != string(actualJSON) {
			return fmt.Errorf("%s does not match the canonical generated contract", jsonPath)
		}
		markdownPath := filepath.Join(dir, expected.base+".md")
		markdown, err := os.ReadFile(markdownPath)
		if err != nil {
			return err
		}
		if string(markdown) != expected.md {
			return fmt.Errorf("%s does not match the canonical generated report", markdownPath)
		}
	}
	return nil
}

func buildArtifacts(verification string) ([]artifact, error) {
	proposalIndependence := preconditions.DefaultIndependencePolicy()
	proposalIndependenceHash, err := preconditions.IndependencePolicyHash(proposalIndependence)
	if err != nil {
		return nil, err
	}
	revisedIndependence := preconditions.RevisedIndependencePolicyV2()
	revisedIndependenceHash, err := preconditions.RevisedIndependencePolicyHashV2(revisedIndependence)
	if err != nil {
		return nil, err
	}
	proposalUncertainty := preconditions.ProposedUncertaintyMethod()
	proposalUncertaintyHash, err := preconditions.UncertaintyMethodHash(proposalUncertainty)
	if err != nil {
		return nil, err
	}
	acceptedUncertainty := preconditions.AcceptedUncertaintyMethodV2()
	acceptedUncertaintyHash, err := preconditions.AcceptedUncertaintyMethodHashV2(acceptedUncertainty)
	if err != nil {
		return nil, err
	}

	independenceDecision := map[string]any{
		"schema_version":       "ak.engine.governance-decision.independence.v1",
		"contract_kind":        "independence",
		"proposal_contract":    map[string]any{"version": proposalIndependence.Version, "hash": proposalIndependenceHash, "status": proposalIndependence.Status},
		"governance_decision":  "REVISE",
		"accepted_replacement": nil,
		"revised_replacement":  map[string]any{"version": revisedIndependence.Version, "hash": revisedIndependenceHash, "status": revisedIndependence.Status},
		"replacement_contract": revisedIndependence,
		"exact_normative_decisions": []string{
			"event exposure is [event_timestamp,event_timestamp+240m) after UTC RFC3339Nano normalization",
			"same-symbol overlapping exposures form transitive connected components",
			"cross-symbol edges require overlapping exposures and identical exact BTC/ETH decision-time context-provenance episode hashes",
			"missing common-context provenance fails closed and blanket market grouping is prohibited",
			"canonical cluster SHA-256 binds policy, normalized membership, exposure endpoints, symbols, and applicable episode identities",
			"one cluster is one qualification decision and later cluster net return is a deterministic sum of exactly one mandatory-cost net member return",
		},
		"unresolved_items":   revisedIndependence.UnresolvedItems,
		"effective_scope":    "future prospective PR4B0-R1 only after a later accepted concentration-complete replacement; synthetic fixtures in R1P3",
		"supersession_rules": "V1 remains PROPOSED_NOT_ACCEPTED; V2 is revised but unaccepted and supersedes no accepted policy",
		"mutation_policy":    "any normative mutation changes the contract hash, requires a new version, and requires a new governance decision",
		"reviewer_authority": "PR4B0-R1 governance authority acting under the R1P3 objective and accepted PR4B0 qualification record",
		"decision_timestamp": decisionTimestamp,
		"accepted_without_candidate_performance_results": true,
		"performance_results_inspected":                  false,
		"decision_record_hash":                           "",
	}
	if err := setHash(independenceDecision, "decision_record_hash"); err != nil {
		return nil, err
	}
	if err := setHash(independenceDecision, "artifact_hash"); err != nil {
		return nil, err
	}

	uncertaintyDecision := map[string]any{
		"schema_version":       "ak.engine.governance-decision.uncertainty.v1",
		"contract_kind":        "uncertainty",
		"proposal_contract":    map[string]any{"version": proposalUncertainty.Version, "hash": proposalUncertaintyHash, "status": proposalUncertainty.Status},
		"governance_decision":  "REVISE",
		"accepted_replacement": map[string]any{"version": acceptedUncertainty.Version, "hash": acceptedUncertaintyHash, "status": acceptedUncertainty.Status},
		"replacement_contract": acceptedUncertainty,
		"exact_normative_decisions": []string{
			"estimand is mean mandatory-cost net expectancy per accepted independent cluster",
			"primary statistic is a one-sided 95% lower percentile bound and qualification requires lower bound > 0",
			"draw N clusters with replacement for each of exactly 10000 unstratified replicates",
			"sort means ascending and use nearest-rank fifth percentile at zero-based ceil(0.05*10000)-1 = 499 without interpolation",
			"derive SplitMix64 seed from canonical contract, candidate, dataset, manifest, partition, and cost identities",
			"N<30 not reportable; 30<=N<300 reportable but sample gate fails; N>=300 qualification eligible",
			"all invalid identity, duplicate, nonfinite, serialization, version, and hash conditions fail closed",
		},
		"unresolved_items":   []string{},
		"effective_scope":    "future prospective PR4B0-R1 qualification after an independence contract is accepted; synthetic fixtures in R1P3",
		"supersession_rules": "V1 remains PROPOSED_NOT_ACCEPTED; accepted V2 replaces the proposal for future authorized use",
		"mutation_policy":    "any normative mutation changes the method hash, requires a new version, and requires a new governance decision",
		"reviewer_authority": "PR4B0-R1 governance authority acting under the normative R1P3 bootstrap decisions",
		"decision_timestamp": decisionTimestamp,
		"accepted_without_candidate_performance_results": true,
		"performance_results_inspected":                  false,
		"decision_record_hash":                           "",
	}
	if err := setHash(uncertaintyDecision, "decision_record_hash"); err != nil {
		return nil, err
	}
	if err := setHash(uncertaintyDecision, "artifact_hash"); err != nil {
		return nil, err
	}

	common := map[string]any{
		"phase":            "PR4B0-R1P3",
		"final_label":      partialLabel,
		"starting_commits": map[string]any{"engine": engineStart, "historian": historianStart, "rif": rifStart},
		"resulting_code_commits": map[string]any{
			"engine_contract_implementation": "2426acb4060618b485cde88eb149de07f68d3198",
			"historian_unchanged":            historianStart,
			"rif_governance_binding":         "e22357692e5b0e837451805f61c6ec3fb2fd6529",
			"engine_artifact_container":      "the later commit containing this self-referentially immutable report is recorded in the final response",
		},
		"contract_hashes": map[string]any{
			"independence_v1_proposal":            proposalIndependenceHash,
			"independence_v2_revised_unaccepted":  revisedIndependenceHash,
			"uncertainty_v1_proposal":             proposalUncertaintyHash,
			"uncertainty_v2_accepted":             acceptedUncertaintyHash,
			"prospective_source_schema_authority": sourceSchemaHash,
			"prospective_manifest_contract":       manifestHash,
		},
		"research_boundary": map[string]any{
			"real_candidate_executions": 0, "real_candidate_outcome_calculations": 0, "real_prospective_records_collected": 0,
			"real_rif_state_reads_or_mutations": 0, "fixtures": "synthetic only",
		},
		"verification_results": verificationResults(verification),
	}

	governance := cloneMap(common)
	governance["schema_version"] = "ak.engine.pr4b0-r1p3-governance-report.v1"
	governance["executive_verdict"] = "Uncertainty contracts are accepted; independence is revised but cannot be accepted because exact historical largest-cluster and aggregate cluster-concentration authority is unrecoverable."
	governance["independence_decision"] = map[string]any{"decision": "REVISE", "accepted": false, "version": revisedIndependence.Version, "hash": revisedIndependenceHash, "blocker": "missing accepted source-report thresholds and denominators"}
	governance["uncertainty_decision"] = map[string]any{"decision": "REVISE_V1_ACCEPT_V2", "accepted": true, "version": acceptedUncertainty.Version, "hash": acceptedUncertaintyHash}
	governance["concentration_authority_recovery"] = revisedIndependence.ConcentrationAuthorities
	governance["synthetic_tests"] = map[string]any{
		"independence_required_vectors": "PASS", "uncertainty_required_vectors": "PASS", "canonical_hash_mutation": "PASS",
		"real_candidate_records_used": 0,
	}
	governance["synthetic_rif_compatibility"] = map[string]any{
		"identity_shape": "PASS", "temporary_lifecycle_sequence": "PASS", "persistent_real_state": false,
		"four_hash_binding": "SUPPORTED_WITH_REVISED_INDEPENDENCE_HASH; FUTURE ACCEPTED BINDING BLOCKED UNTIL INDEPENDENCE ACCEPTANCE",
	}
	governance["verification_status"] = verification
	governance["remaining_blockers"] = revisedIndependence.UnresolvedItems
	governance["recommended_next_phase"] = "Restore and govern exact concentration authority before PR4B0-R1P4; do not activate prospective collection"
	if err := setHash(governance, "artifact_hash"); err != nil {
		return nil, err
	}

	finalDecision := cloneMap(common)
	finalDecision["schema_version"] = "ak.engine.pr4b0-r1p3-final-decision.v1"
	finalDecision["executive_verdict"] = governance["executive_verdict"]
	finalDecision["independence_governance_decision"] = "REVISE"
	finalDecision["accepted_independence_version"] = nil
	finalDecision["accepted_independence_hash"] = nil
	finalDecision["revised_independence_version"] = revisedIndependence.Version
	finalDecision["revised_independence_hash"] = revisedIndependenceHash
	finalDecision["uncertainty_governance_decision"] = "REVISE_V1_ACCEPT_V2"
	finalDecision["accepted_uncertainty_version"] = acceptedUncertainty.Version
	finalDecision["accepted_uncertainty_hash"] = acceptedUncertaintyHash
	finalDecision["verification_status"] = verification
	finalDecision["verification_results"] = verificationResults(verification)
	finalDecision["remaining_blockers"] = revisedIndependence.UnresolvedItems
	finalDecision["next_phase"] = "GOVERNANCE_REMEDIATION_REQUIRED; PR4B0-R1P4 NOT AUTHORIZED"
	finalDecision["decision_timestamp"] = decisionTimestamp
	if err := setHash(finalDecision, "artifact_hash"); err != nil {
		return nil, err
	}

	return []artifact{
		{"pr4b0_r1p3_independence_decision", independenceDecision, independenceMarkdown(independenceDecision, revisedIndependenceHash)},
		{"pr4b0_r1p3_uncertainty_decision", uncertaintyDecision, uncertaintyMarkdown(uncertaintyDecision, acceptedUncertaintyHash)},
		{"pr4b0_r1p3_governance_report", governance, governanceMarkdown(governance, revisedIndependenceHash, acceptedUncertaintyHash)},
		{"pr4b0_r1p3_final_decision", finalDecision, finalMarkdown(finalDecision, revisedIndependenceHash, acceptedUncertaintyHash)},
	}, nil
}

func independenceMarkdown(record map[string]any, hash string) string {
	return fmt.Sprintf("# PR4B0-R1P3 Independence Decision\n\nDecision: **REVISE**. The V1 proposal remains unaccepted.\n\nRevised V2: `%s`\n\nRevised contract hash: `%s`\n\nV2 implements half-open 240-minute UTC exposure, transitive same-symbol overlap, exact episode-qualified cross-symbol overlap, deterministic canonical cluster IDs, deduplication, and one-cluster/one-decision semantics. It is **not accepted** because no accepted source report supplies exact largest-cluster or aggregate cluster-concentration thresholds and denominators. Symbol and temporal `<=50%%` thresholds were recovered from `runs/reports/pr4b0_candidate_qualification.json` at commit `205cf59555006ce23fc58bc2c73262660a894850`, but their denominator definitions are incomplete. No threshold was invented.\n\nCommon-market episode identity is the canonical SHA-256 of exact BTCUSDT/ETHUSDT context symbols, snapshot IDs, source hashes, and UTC availability instants. This conservative rule can under-cluster when distinct provenance records describe one move; same-symbol bridges can join multiple episode identities; and it intentionally does not infer latent dependence from future or retrospective values.\n\nAcceptance was decided without inspecting candidate-performance results.\n\nDecision-record hash: `%v`\nArtifact hash: `%v`\n", preconditions.RevisedIndependencePolicyVersion, hash, record["decision_record_hash"], record["artifact_hash"])
}

func uncertaintyMarkdown(record map[string]any, hash string) string {
	return fmt.Sprintf("# PR4B0-R1P3 Uncertainty Decision\n\nDecision: **REVISE V1; ACCEPT V2**. V1 remains proposed and unaccepted.\n\nAccepted version: `%s`\n\nAccepted hash: `%s`\n\nThe accepted method estimates mean mandatory-cost net expectancy per independent cluster. It uses exactly 10,000 unstratified nonparametric cluster-bootstrap replicates, N draws with replacement per replicate, ascending numeric sort, and the nearest-rank fifth-percentile element at zero-based index 499. Qualification requires N >= 300 and lower bound > 0. N < 30 is not reportable; 30 <= N < 300 is reportable but fails the sample gate. The seed binds every required canonical identity.\n\nAcceptance was decided using synthetic fixtures only and without inspecting candidate-performance results.\n\nDecision-record hash: `%v`\nArtifact hash: `%v`\n", preconditions.AcceptedUncertaintyMethodVersion, hash, record["decision_record_hash"], record["artifact_hash"])
}

func governanceMarkdown(record map[string]any, independenceHash, uncertaintyHash string) string {
	return fmt.Sprintf("# PR4B0-R1P3 Governance Report\n\nExecutive verdict: %v\n\nFinal label: `%s`\n\n- Independence: REVISE, V2 `%s` is implemented but unaccepted.\n- Uncertainty: V2 `%s` is accepted.\n- Synthetic RIF V2 identity/lifecycle compatibility: PASS for the identity shape; future accepted four-hash binding remains blocked by independence acceptance.\n- Research boundary: zero real candidate executions, calculations, prospective records, or real RIF state access.\n- Verification status: %v.\n\nPR4B0-R1P4 is not authorized. Restore and govern the missing concentration authority first.\n\nArtifact hash: `%v`\n", record["executive_verdict"], partialLabel, independenceHash, uncertaintyHash, record["verification_status"], record["artifact_hash"])
}

func finalMarkdown(record map[string]any, independenceHash, uncertaintyHash string) string {
	return fmt.Sprintf("# PR4B0-R1P3 Final Decision\n\nExecutive verdict: %v\n\nFinal label: `%s`\n\nRevised unaccepted independence V2 hash: `%s`\n\nAccepted uncertainty V2 hash: `%s`\n\nRemaining blocker: accepted largest-cluster and aggregate cluster-concentration source-report authority is missing; symbol/temporal denominator authority is incomplete. No real candidate result was generated or inspected, no real prospective data was collected, and no real RIF state was accessed.\n\nNext phase: governance remediation. **Do not begin PR4B0-R1P4.**\n\nArtifact hash: `%v`\n", record["executive_verdict"], partialLabel, independenceHash, uncertaintyHash, record["artifact_hash"])
}

func cloneMap(input map[string]any) map[string]any {
	output := make(map[string]any, len(input))
	for key, value := range input {
		output[key] = value
	}
	return output
}

func verificationResults(status string) map[string]any {
	result := map[string]any{
		"engine":                     map[string]any{"gofmt": status, "go_mod_tidy_diff": status, "go_vet": status, "go_test": status, "go_test_race": status, "go_build": status, "make_verify": status, "git_diff_check": status},
		"rif":                        map[string]any{"gofmt": status, "go_mod_tidy_diff": status, "go_vet": status, "go_test": status, "go_test_race": status, "go_build": status, "make_verify": status, "git_diff_check": status},
		"scans":                      map[string]any{"secret_credentials": status, "absolute_paths": status, "sibling_dependencies": status, "trader_imports": status, "network_dependencies": status},
		"json_validation":            status,
		"recorded_sha256_validation": status,
		"fresh_clone":                map[string]any{"engine": status, "rif": status, "no_sibling_ak_repositories": status, "clean_worktrees": status},
	}
	return result
}

func setHash(object map[string]any, field string) error {
	data, err := canonicalMapBytesWithout(object, field)
	if err != nil {
		return err
	}
	digest := sha256.Sum256(data)
	object[field] = "sha256:" + hex.EncodeToString(digest[:])
	return nil
}

func verifyHash(object map[string]any, field string) error {
	recorded, ok := object[field].(string)
	if !ok || recorded == "" {
		return fmt.Errorf("%s is missing", field)
	}
	excluded := []string{field}
	if field == "decision_record_hash" {
		excluded = append(excluded, "artifact_hash")
	}
	data, err := canonicalMapBytesWithout(object, excluded...)
	if err != nil {
		return err
	}
	digest := sha256.Sum256(data)
	want := "sha256:" + hex.EncodeToString(digest[:])
	if recorded != want {
		return fmt.Errorf("%s mismatch", field)
	}
	return nil
}

func canonicalMapBytes(object map[string]any) ([]byte, error) {
	return canonicalMapBytesWithout(object)
}

func canonicalMapBytesWithout(object map[string]any, excluded ...string) ([]byte, error) {
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
	for _, field := range excluded {
		delete(normalized, field)
	}
	return json.Marshal(normalized)
}

func sortedKeys(object map[string]any) []string {
	keys := make([]string, 0, len(object))
	for key := range object {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}

func fatal(err error) {
	fmt.Fprintln(os.Stderr, strings.TrimSpace(err.Error()))
	os.Exit(1)
}
