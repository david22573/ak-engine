package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/david22573/ak-engine/internal/preconditions"
)

const (
	expectedIndependenceHash = "sha256:68641244ae31bb47dd7752410763308bf8a88362660f9df347362f42fbbb339a"
	expectedUncertaintyHash  = "sha256:8dfe7591c0bf2b9cbb3b3d9ce4ad6af8a1898d5d199bb4756b3d2b17e9e4a76d"
)

func TestGovernancePacketsPinProposedContractsAndSyntheticVectors(t *testing.T) {
	builders := []struct {
		name         string
		expectedHash string
		build        func() (map[string]any, error)
		realCountKey string
	}{
		{"independence", expectedIndependenceHash, buildIndependencePacket, "real_candidate_events_used"},
		{"uncertainty", expectedUncertaintyHash, buildUncertaintyPacket, "real_candidate_observations_used"},
	}
	for _, test := range builders {
		t.Run(test.name, func(t *testing.T) {
			first, err := test.build()
			if err != nil {
				t.Fatal(err)
			}
			second, err := test.build()
			if err != nil {
				t.Fatal(err)
			}
			firstJSON, _ := json.Marshal(first)
			secondJSON, _ := json.Marshal(second)
			if string(firstJSON) != string(secondJSON) {
				t.Fatal("synthetic packet is not deterministic")
			}
			if first["contract_hash"] != test.expectedHash || first["contract_status"] != preconditions.PolicyStatusProposedNotAccepted {
				t.Fatalf("contract identity/status drifted: %v %v", first["contract_hash"], first["contract_status"])
			}
			if first[test.realCountKey] != 0 || !strings.HasPrefix(first["packet_hash"].(string), "sha256:") {
				t.Fatal("packet used real inputs or lacks an identity")
			}
			decision := first["governance_decision_requested"].(map[string]any)
			if decision["preselected_decision"] != nil {
				t.Fatal("governance decision was preselected")
			}
			if len(decision["allowed_decisions"].([]string)) != 3 || len(decision["unresolved_decisions"].([]string)) == 0 {
				t.Fatal("reviewer decision surface is incomplete")
			}
		})
	}
}

func TestCommittedEvidenceAndGovernanceReportsMatchBuilders(t *testing.T) {
	provenance, err := buildProvenance()
	if err != nil {
		t.Fatal(err)
	}
	inspection, err := buildInspection()
	if err != nil {
		t.Fatal(err)
	}
	independence, err := buildIndependencePacket()
	if err != nil {
		t.Fatal(err)
	}
	uncertainty, err := buildUncertaintyPacket()
	if err != nil {
		t.Fatal(err)
	}
	for name, value := range map[string]any{
		"pr4b0_r1p2_provenance_resolution":          provenance,
		"pr4b0_r1p2_inspection_audit":               inspection,
		"pr4b0_r1p2_independence_governance_packet": independence,
		"pr4b0_r1p2_uncertainty_governance_packet":  uncertainty,
	} {
		want, err := json.MarshalIndent(value, "", "  ")
		if err != nil {
			t.Fatal(err)
		}
		got, err := os.ReadFile(filepath.Join("..", "..", "runs", "reports", name+".json"))
		if err != nil {
			t.Fatal(err)
		}
		if string(got) != string(append(want, '\n')) {
			t.Fatalf("%s is not synchronized with its canonical builder", name)
		}
	}
}

func TestGovernancePacketHashBindsNormativeContent(t *testing.T) {
	packet, err := buildIndependencePacket()
	if err != nil {
		t.Fatal(err)
	}
	original := packet["packet_hash"]
	packet["rationale"] = "mutated"
	mutated, err := sealMap(packet, "packet_hash")
	if err != nil {
		t.Fatal(err)
	}
	if mutated["packet_hash"] == original {
		t.Fatal("normative packet mutation did not change packet hash")
	}
}

func TestContractHashesBindEveryNormativeProposal(t *testing.T) {
	policy := preconditions.DefaultIndependencePolicy()
	policyHash, err := preconditions.IndependencePolicyHash(policy)
	if err != nil {
		t.Fatal(err)
	}
	policy.MinimumSpacingMS++
	mutatedPolicyHash, err := preconditions.IndependencePolicyHash(policy)
	if err != nil {
		t.Fatal(err)
	}
	if mutatedPolicyHash == policyHash {
		t.Fatal("independence contract mutation did not change its hash")
	}

	method := preconditions.ProposedUncertaintyMethod()
	methodHash, err := preconditions.UncertaintyMethodHash(method)
	if err != nil {
		t.Fatal(err)
	}
	method.NumberOfResamples++
	mutatedMethodHash, err := preconditions.UncertaintyMethodHash(method)
	if err != nil {
		t.Fatal(err)
	}
	if mutatedMethodHash == methodHash {
		t.Fatal("uncertainty contract mutation did not change its hash")
	}

	independence, err := buildIndependencePacket()
	if err != nil {
		t.Fatal(err)
	}
	normative := independence["normative_specification"].(map[string]any)
	for _, key := range []string{"event_cluster_algorithm", "same_symbol_overlap_rule", "cross_symbol_common_market_overlap_rule", "horizon_overlap_rule", "minimum_spacing_rule", "timestamp_boundary_semantics", "cluster_id_construction", "duplicate_handling", "deterministic_ordering_and_tie_breaking", "independent_sample_definition", "concentration_calculation", "fail_closed_conditions", "versioning_and_mutation_rules"} {
		if _, ok := normative[key]; !ok {
			t.Fatalf("independence normative field %s is missing", key)
		}
	}
	uncertainty, err := buildUncertaintyPacket()
	if err != nil {
		t.Fatal(err)
	}
	for _, key := range []string{"estimand", "expectancy_definition", "net_cost_treatment", "confidence_level", "resampling_unit", "relationship_to_independence_contract", "procedure", "replacement_rules", "stratification", "block_construction", "seed_derivation", "number_of_resamples", "interval_construction", "degenerate_samples", "missing_or_invalid_records", "deterministic_serialization", "hash_and_version_rules"} {
		if _, ok := uncertainty[key]; !ok {
			t.Fatalf("uncertainty normative field %s is missing", key)
		}
	}
}
