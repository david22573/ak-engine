package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

const testCommit = "1111111111111111111111111111111111111111"

func TestBuildArtifactsCompleteAndUnaccepted(t *testing.T) {
	artifacts, err := buildArtifacts("PASS", testCommit, testCommit)
	if err != nil {
		t.Fatal(err)
	}
	if len(artifacts) != 5 {
		t.Fatalf("artifacts=%d, want 5", len(artifacts))
	}
	for _, item := range artifacts {
		if err := verifyHash(item.json); err != nil {
			t.Fatalf("%s: %v", item.base, err)
		}
		if _, err := json.Marshal(item.json); err != nil {
			t.Fatal(err)
		}
		if !strings.Contains(item.md, "Artifact hash") {
			t.Fatalf("%s markdown omits artifact hash", item.base)
		}
	}
	final := artifacts[4].json
	if final["final_label"] != finalLabel || final["accepted_independence_hash"] != nil || final["next_phase"] != "USER_CONCENTRATION_GOVERNANCE_DECISION_REQUIRED" {
		t.Fatalf("unexpected final decision: %#v", final)
	}
	packet := artifacts[3].json
	if packet["packet_status"] != "DECISION_READY_UNACCEPTED" || packet["accepted_contract_created"] != false {
		t.Fatalf("packet accidentally accepted: %#v", packet)
	}
	request := packet["reviewer_decision_request"].(map[string]any)
	if request["preselected_decision"] != nil {
		t.Fatal("governance decision was preselected")
	}
}

func TestAuthorityMatrixHasFourExplicitRows(t *testing.T) {
	artifacts, err := buildArtifacts("PENDING", testCommit, testCommit)
	if err != nil {
		t.Fatal(err)
	}
	rows := artifacts[1].json["authority_rows"].([]map[string]any)
	if len(rows) != 4 {
		t.Fatalf("rows=%d, want 4", len(rows))
	}
	if rows[0]["authority_classification"] != "ACCEPTED_AUTHORITY_PARTIAL" || rows[2]["authority_classification"] != "NO_AUTHORITY_FOUND" {
		t.Fatalf("unexpected authority classifications: %#v", rows)
	}
	for _, row := range rows {
		for _, field := range []string{"numerator_definition", "denominator_definition", "aggregation_unit", "bucket_type", "timezone", "empty_bucket_handling", "deduplication_and_clustering_stage", "rounding_rule", "failure_semantics", "partition_and_combined_scope"} {
			if _, ok := row[field]; !ok {
				t.Fatalf("%s missing %s", row["metric"], field)
			}
		}
	}
}

func TestWriteAndVerifyTenRequiredFiles(t *testing.T) {
	artifacts, err := buildArtifacts("PASS", testCommit, testCommit)
	if err != nil {
		t.Fatal(err)
	}
	dir := t.TempDir()
	for _, item := range artifacts {
		data, err := json.MarshalIndent(item.json, "", "  ")
		if err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(dir, item.base+".json"), append(data, '\n'), 0o600); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(dir, item.base+".md"), []byte(item.md), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	entries, err := os.ReadDir(dir)
	if err != nil || len(entries) != 10 {
		t.Fatalf("files=%d err=%v", len(entries), err)
	}
	if err := verifyArtifacts(dir, artifacts); err != nil {
		t.Fatal(err)
	}
}

func TestArtifactMutationAndInvalidCommitFail(t *testing.T) {
	artifacts, err := buildArtifacts("PASS", testCommit, testCommit)
	if err != nil {
		t.Fatal(err)
	}
	artifacts[0].json["conclusion"] = "mutated"
	if err := verifyHash(artifacts[0].json); err == nil {
		t.Fatal("artifact mutation retained valid hash")
	}
	if validCommit("not-a-commit") || !validCommit(testCommit) {
		t.Fatal("commit validation is not fail closed")
	}
}
