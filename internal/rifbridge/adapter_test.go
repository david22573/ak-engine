package rifbridge_test

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/david22573/ak-engine/internal/rifbridge"
)

func TestEvaluateAndEmit(t *testing.T) {
	tempDir := t.TempDir()
	stem := filepath.Join(tempDir, "test_candidate")

	bridge := rifbridge.NewBridge()

	// Create a point-in-time universe manifest for the success case.
	manifestPath := filepath.Join(tempDir, "dataset_manifest.json")
	os.WriteFile(manifestPath, []byte(`{
		"dataset_id":"test-dataset",
		"hashes":{"dataset_hash":"hash-d","manifest_hash":"hash-m"},
		"survivorship":{
			"universe_id":"pit-universe",
			"universe_hash":"hash-u",
			"universe_manifest_hash":"hash-um",
			"universe_policy":"POINT_IN_TIME_EXCHANGE_UNIVERSE",
			"includes_delisted_assets":"true",
			"survivorship_bias_risk":"LOW",
			"lifecycle_id":"life-1",
			"lifecycle_hash":"hash-life",
			"lifecycle_manifest_hash":"hash-life-manifest",
			"lifecycle_evidence_level_summary":{"HISTORICAL_SNAPSHOT_EVIDENCE":1},
			"listing_evidence_status":"VERIFIED",
			"delisting_evidence_status":"VERIFIED",
			"survivorship_support_status":"LOW_SUPPORTED"
		}
	}`), 0644)

	// 1. Success case: Should emit lock, audit, and promotion packet
	out, err := bridge.EvaluateAndEmit(
		stem,
		"cand-123",
		"v1.0.0",
		"abcd123",
		[]string{"d1"},
		[]string{"f1"},
		"c1",
		"cfg1",
		[]float64{0.01, 0.02, 0.03},
		[]int64{100, 200, 300},
		2,
		40,   // > 30, should pass sample size
		true, // isPromoted
		manifestPath,
	)
	if err != nil {
		t.Fatalf("EvaluateAndEmit failed: %v", err)
	}

	if !out.IntegrityPassed {
		t.Fatalf("Expected integrity to pass, got warnings: %v", out.Warnings)
	}

	// Verify research.lock
	lockData, err := os.ReadFile(stem + ".research.lock")
	if err != nil {
		t.Fatalf("Expected research.lock to be created: %v", err)
	}
	var lock rifbridge.ResearchLock
	if err := json.Unmarshal(lockData, &lock); err != nil {
		t.Fatalf("Failed to parse research.lock: %v", err)
	}
	if lock.GitSHA != "abcd123" {
		t.Errorf("Expected git SHA abcd123, got %s", lock.GitSHA)
	}
	if lock.LifecycleHash != "hash-life" || lock.LifecycleManifestHash != "hash-life-manifest" {
		t.Errorf("Expected lifecycle hashes in research.lock, got %#v", lock)
	}
	if lock.LifecycleEvidenceLevelSummary["HISTORICAL_SNAPSHOT_EVIDENCE"] != 1 {
		t.Errorf("Expected lifecycle evidence summary in research.lock")
	}

	// Verify research_audit.json
	if _, err := os.Stat(stem + ".research_audit.json"); err != nil {
		t.Fatalf("Expected research_audit.json to be created: %v", err)
	}

	// Verify promotion_packet.json
	packetData, err := os.ReadFile(stem + ".promotion_packet.json")
	if err != nil {
		t.Fatalf("Expected promotion_packet.json to be created: %v", err)
	}
	var packet rifbridge.PromotionPacket
	if err := json.Unmarshal(packetData, &packet); err != nil {
		t.Fatalf("Failed to parse promotion_packet: %v", err)
	}
	if !packet.PassedIntegrityChecks {
		t.Errorf("Expected PassedIntegrityChecks to be true in packet")
	}

	// 2. Failure case: Integrity fails (small sample size)
	stemFail := filepath.Join(tempDir, "fail_candidate")
	outFail, err := bridge.EvaluateAndEmit(
		stemFail,
		"cand-fail",
		"v1.0.0",
		"abcd123",
		[]string{},
		[]string{},
		"c2",
		"cfg2",
		[]float64{0.01},
		[]int64{100},
		2,
		5, // < 30, should fail sample size
		true,
		"",
	)
	if err != nil {
		t.Fatalf("EvaluateAndEmit failed: %v", err)
	}

	if outFail.IntegrityPassed {
		t.Fatalf("Expected integrity to fail due to sample size")
	}

	// Promotion packet should NOT be generated if integrity fails, even if isPromoted is true
	if _, err := os.Stat(stemFail + ".promotion_packet.json"); !os.IsNotExist(err) {
		t.Fatalf("Expected promotion packet to NOT be created for failed integrity")
	}
}

func TestEmitRunFinalization(t *testing.T) {
	tempDir := t.TempDir()
	stem := filepath.Join(tempDir, "run_summary")

	bridge := rifbridge.NewBridge()
	err := bridge.EmitRunFinalization(stem, "abcd123", []string{"d1"}, []string{"f1"}, "cfg1", "")
	if err != nil {
		t.Fatalf("EmitRunFinalization failed: %v", err)
	}

	if _, err := os.Stat(stem + ".research.lock"); err != nil {
		t.Fatalf("Expected research.lock to exist")
	}
	if _, err := os.Stat(stem + ".research_audit.json"); err != nil {
		t.Fatalf("Expected research_audit.json to exist")
	}
}
