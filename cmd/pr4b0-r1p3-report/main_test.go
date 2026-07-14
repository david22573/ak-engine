package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestBuildArtifactsHashesAndDecisions(t *testing.T) {
	artifacts, err := buildArtifacts("PASS")
	if err != nil {
		t.Fatal(err)
	}
	if len(artifacts) != 4 {
		t.Fatalf("artifacts=%d, want 4", len(artifacts))
	}
	for _, item := range artifacts {
		if err := verifyHash(item.json, "artifact_hash"); err != nil {
			t.Fatalf("%s: %v", item.base, err)
		}
		data, err := json.Marshal(item.json)
		if err != nil || strings.Contains(string(data), "DowntrendMidVolReliefLong240m") {
			t.Fatalf("%s contains prohibited candidate execution identity or malformed JSON", item.base)
		}
		if !strings.Contains(item.md, "Artifact hash") {
			t.Fatalf("%s markdown omits hash", item.base)
		}
	}
	if err := verifyHash(artifacts[0].json, "decision_record_hash"); err != nil {
		t.Fatal(err)
	}
	if err := verifyHash(artifacts[1].json, "decision_record_hash"); err != nil {
		t.Fatal(err)
	}
	if artifacts[0].json["governance_decision"] != "REVISE" || artifacts[0].json["accepted_replacement"] != nil {
		t.Fatal("independence was incorrectly accepted")
	}
	if artifacts[1].json["governance_decision"] != "REVISE" || artifacts[1].json["accepted_replacement"] == nil {
		t.Fatal("uncertainty replacement was not accepted")
	}
	if artifacts[3].json["final_label"] != partialLabel {
		t.Fatal("incorrect final label")
	}
}

func TestArtifactHashMutationFails(t *testing.T) {
	artifacts, err := buildArtifacts("PENDING")
	if err != nil {
		t.Fatal(err)
	}
	artifacts[2].json["verification_status"] = "MUTATED"
	if err := verifyHash(artifacts[2].json, "artifact_hash"); err == nil {
		t.Fatal("artifact mutation retained a valid hash")
	}
}

func TestWriteShapeUsesEightRequiredFiles(t *testing.T) {
	artifacts, err := buildArtifacts("PASS")
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
	if err != nil || len(entries) != 8 {
		t.Fatalf("required files=%d err=%v", len(entries), err)
	}
}

func TestSortedKeysDeterministic(t *testing.T) {
	keys := sortedKeys(map[string]any{"z": 1, "a": 2})
	if strings.Join(keys, ",") != "a,z" {
		t.Fatal("map key ordering helper is nondeterministic")
	}
}
