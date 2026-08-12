package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/david22573/ak-engine/internal/preconditions"
)

func TestGenerateCanonicalP3BArtifacts(t *testing.T) {
	dir := t.TempDir()
	if err := generate(dir); err != nil {
		t.Fatal(err)
	}
	names := []string{"pr4b0_r1p3b_concentration_governance_decision", "pr4b0_r1p3b_independence_contract", "pr4b0_r1p3b_qualification_enforcement", "pr4b0_r1p3b_final_decision"}
	for _, name := range names {
		data, err := os.ReadFile(filepath.Join(dir, name+".json"))
		if err != nil {
			t.Fatal(err)
		}
		var object map[string]any
		if err := json.Unmarshal(data, &object); err != nil {
			t.Fatal(err)
		}
		if err := validateArtifactHash(object); err != nil {
			t.Fatalf("%s: %v", name, err)
		}
		if _, err := os.Stat(filepath.Join(dir, name+".md")); err != nil {
			t.Fatal(err)
		}
	}
}

func TestArtifactsBindExactDecisionAndV3Hashes(t *testing.T) {
	dir := t.TempDir()
	if err := generate(dir); err != nil {
		t.Fatal(err)
	}
	decision := readArtifact(t, filepath.Join(dir, "pr4b0_r1p3b_concentration_governance_decision.json"))
	record := decision["governance_decision"].(map[string]any)
	if record["decision"] != "ACCEPT_ALTERNATIVE" || record["selected_alternative"] != "STRUCTURAL_COUNT_BASED_CONCENTRATION" || record["decision_scope"] != "FUTURE_PR4B0_R1_RESEARCH_ONLY" || record["historical_authority_claimed"] != false {
		t.Fatalf("decision mismatch: %+v", record)
	}
	contract := readArtifact(t, filepath.Join(dir, "pr4b0_r1p3b_independence_contract.json"))
	policy := preconditions.AcceptedIndependencePolicyV3Default()
	hash, err := preconditions.AcceptedIndependencePolicyHashV3(policy)
	if err != nil {
		t.Fatal(err)
	}
	if contract["contract_version"] != policy.Version || contract["contract_hash"] != hash || contract["governance_decision_hash"] != policy.GovernanceDecisionHash {
		t.Fatalf("contract identity mismatch: %+v", contract)
	}
	final := readArtifact(t, filepath.Join(dir, "pr4b0_r1p3b_final_decision.json"))
	if final["final_label"] != finalLabel || final["real_candidate_execution_count"] != float64(0) || final["prospective_record_collection_or_inspection_count"] != float64(0) || final["next_phase_started"] != false {
		t.Fatalf("final boundary mismatch: %+v", final)
	}
}

func readArtifact(t *testing.T, path string) map[string]any {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	var object map[string]any
	if err := json.Unmarshal(data, &object); err != nil {
		t.Fatal(err)
	}
	return object
}
