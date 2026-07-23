package rifbridge

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestRIFBridgeUniverseIntegration(t *testing.T) {
	bridge := NewBridge()
	dir := t.TempDir()

	stem := filepath.Join(dir, "test_run")
	manifestPath := filepath.Join(dir, "dataset_manifest.json")

	// Create a dummy dataset manifest with embedded universe data
	manifest := map[string]interface{}{
		"dataset_id": "test_dataset",
		"survivorship": map[string]interface{}{
			"universe_id":            "test_universe",
			"universe_hash":          "u_hash_123",
			"universe_manifest_hash": "um_hash_123",
			"universe_policy":        "EXPLICIT_SYMBOL_LIST",
			"survivorship_bias_risk": "HIGH",
			"warnings":               []string{"TEST_WARN"},
		},
	}
	b, _ := json.Marshal(manifest)
	_ = os.WriteFile(manifestPath, b, 0644)

	// Simulate a candidate evaluation that tries to promote
	out, err := bridge.EvaluateAndEmit(
		stem, "CAND_1", "v1", "git123", nil, nil, "chash", "cfghash",
		[]float64{0.01, -0.01, 0.02}, []int64{100, 200, 300}, 2, 35,
		true, // isPromoted
		manifestPath,
	)

	if err != nil {
		t.Fatalf("EvaluateAndEmit failed: %v", err)
	}

	if out.IntegrityPassed {
		t.Errorf("Expected integrity to fail due to universe policy EXPLICIT_SYMBOL_LIST")
	}

	foundWarning := false
	for _, w := range out.Warnings {
		if stringsContains(w, "RIF_SURVIVORSHIP_BIAS_RISK") {
			foundWarning = true
			break
		}
	}
	if !foundWarning {
		t.Errorf("Expected RIF_SURVIVORSHIP_BIAS_RISK warning, got %v", out.Warnings)
	}
}

func TestRIFBridgeLocalDataDiscoveryHighRisk(t *testing.T) {
	bridge := NewBridge()
	dir := t.TempDir()

	stem := filepath.Join(dir, "test_run")
	manifestPath := filepath.Join(dir, "dataset_manifest.json")

	manifest := map[string]interface{}{
		"dataset_id": "test_dataset",
		"survivorship": map[string]interface{}{
			"universe_policy":        "LOCAL_DATA_DISCOVERED_SYMBOLS",
			"survivorship_bias_risk": "MEDIUM",
		},
	}
	b, _ := json.Marshal(manifest)
	_ = os.WriteFile(manifestPath, b, 0644)

	out, _ := bridge.EvaluateAndEmit(
		stem, "CAND_1", "v1", "git123", nil, nil, "chash", "cfghash",
		[]float64{0.01, -0.01, 0.02}, []int64{100, 200, 300}, 2, 35,
		true, // isPromoted
		manifestPath,
	)

	if out.IntegrityPassed {
		t.Errorf("Expected integrity to fail due to local data discovery without LOW risk")
	}

	foundWarning := false
	for _, w := range out.Warnings {
		if stringsContains(w, "RIF_UNIVERSE_NOT_POINT_IN_TIME") {
			foundWarning = true
			break
		}
	}
	if !foundWarning {
		t.Errorf("Expected RIF_UNIVERSE_NOT_POINT_IN_TIME warning, got %v", out.Warnings)
	}
}

func TestRIFBridgeExplicitListExploratorySucceedsWithStructuredWarnings(t *testing.T) {
	bridge := NewBridge()
	dir := t.TempDir()
	stem := filepath.Join(dir, "exploratory")
	manifestPath := filepath.Join(dir, "dataset_manifest.json")

	manifest := map[string]interface{}{
		"dataset_id": "test_dataset",
		"hashes": map[string]interface{}{
			"dataset_hash":  "d_hash",
			"manifest_hash": "m_hash",
		},
		"survivorship": map[string]interface{}{
			"universe_id":            "test_universe",
			"universe_hash":          "u_hash_123",
			"universe_manifest_hash": "um_hash_123",
			"universe_policy":        "EXPLICIT_SYMBOL_LIST",
			"survivorship_bias_risk": "HIGH",
			"warnings":               []string{"UNIVERSE_EXPLICIT_SYMBOL_LIST_SURVIVORSHIP_RISK"},
		},
	}
	b, _ := json.Marshal(manifest)
	_ = os.WriteFile(manifestPath, b, 0644)

	out, err := bridge.EvaluateAndEmit(
		stem, "CAND_1", "v1", "git123", nil, nil, "chash", "cfghash",
		[]float64{0.01, -0.01, 0.02}, []int64{100, 200, 300}, 1, 35,
		false,
		manifestPath,
	)
	if err != nil {
		t.Fatalf("EvaluateAndEmit failed: %v", err)
	}
	if !out.IntegrityPassed {
		t.Fatalf("Expected exploratory output to pass with warnings: %v", out.Warnings)
	}

	auditData, err := os.ReadFile(stem + ".research_audit.json")
	if err != nil {
		t.Fatalf("Expected research_audit.json: %v", err)
	}
	var audit map[string]interface{}
	if err := json.Unmarshal(auditData, &audit); err != nil {
		t.Fatalf("Failed to parse audit: %v", err)
	}
	warnings, ok := audit["warnings"].([]interface{})
	if !ok || len(warnings) == 0 {
		t.Fatalf("Expected structured audit warnings, got %#v", audit["warnings"])
	}
	first, ok := warnings[0].(map[string]interface{})
	if !ok || first["blocks_promotion"] != true {
		t.Fatalf("Expected warning to include blocks_promotion=true, got %#v", warnings[0])
	}
}

func TestRIFBridgeLifecycleMetadataFlowsAndBlocksWeakEvidence(t *testing.T) {
	bridge := NewBridge()
	dir := t.TempDir()
	stem := filepath.Join(dir, "weak_lifecycle")
	manifestPath := filepath.Join(dir, "dataset_manifest.json")

	manifest := map[string]interface{}{
		"dataset_id": "test_dataset",
		"hashes": map[string]interface{}{
			"dataset_hash":  "d_hash",
			"manifest_hash": "m_hash",
		},
		"survivorship": map[string]interface{}{
			"universe_id":                      "test_universe",
			"universe_hash":                    "u_hash_123",
			"universe_manifest_hash":           "um_hash_123",
			"universe_policy":                  "POINT_IN_TIME_EXCHANGE_UNIVERSE",
			"includes_delisted_assets":         "unknown",
			"survivorship_bias_risk":           "MEDIUM",
			"lifecycle_id":                     "life_1",
			"lifecycle_hash":                   "life_hash_123",
			"lifecycle_manifest_hash":          "life_manifest_hash_123",
			"lifecycle_evidence_level_summary": map[string]interface{}{"LOCAL_DATA_FIRST_SEEN": 1},
			"lifecycle_warnings":               []string{"LIFECYCLE_LOCAL_DATA_ONLY_NOT_LISTING_PROOF"},
			"listing_evidence_status":          "FIRST_SEEN_ONLY",
			"delisting_evidence_status":        "MISSING",
			"survivorship_support_status":      "ELEVATED",
		},
	}
	b, _ := json.Marshal(manifest)
	_ = os.WriteFile(manifestPath, b, 0644)

	out, err := bridge.EvaluateAndEmit(
		stem, "CAND_1", "v1", "git123", nil, nil, "chash", "cfghash",
		[]float64{0.01, -0.01, 0.02}, []int64{100, 200, 300}, 2, 35,
		true,
		manifestPath,
	)
	if err != nil {
		t.Fatalf("EvaluateAndEmit failed: %v", err)
	}
	if out.IntegrityPassed {
		t.Fatalf("Expected weak lifecycle evidence to block promotion")
	}
	for _, code := range []string{"RIF_LIFECYCLE_EVIDENCE_WEAK", "RIF_LIFECYCLE_DELISTING_EVIDENCE_MISSING", "RIF_LIFECYCLE_LISTING_EVIDENCE_MISSING", "RIF_SURVIVORSHIP_NOT_SOLVED"} {
		found := false
		for _, w := range out.Warnings {
			if stringsContains(w, code) {
				found = true
				break
			}
		}
		if !found {
			t.Fatalf("Expected %s in warnings, got %v", code, out.Warnings)
		}
	}

	lockData, err := os.ReadFile(stem + ".research.lock")
	if err != nil {
		t.Fatalf("Expected research.lock: %v", err)
	}
	var lock map[string]interface{}
	if err := json.Unmarshal(lockData, &lock); err != nil {
		t.Fatalf("parse lock: %v", err)
	}
	if lock["lifecycle_hash"] != "life_hash_123" {
		t.Fatalf("lifecycle hash missing from lock: %#v", lock)
	}

	auditData, err := os.ReadFile(stem + ".research_audit.json")
	if err != nil {
		t.Fatalf("Expected research_audit.json: %v", err)
	}
	var audit map[string]interface{}
	if err := json.Unmarshal(auditData, &audit); err != nil {
		t.Fatalf("parse audit: %v", err)
	}
	prov := audit["dataset_provenance"].(map[string]interface{})
	if prov["lifecycle_hash"] != "life_hash_123" || prov["listing_evidence_status"] != "FIRST_SEEN_ONLY" {
		t.Fatalf("lifecycle metadata missing from audit provenance: %#v", prov)
	}
}

func TestRIFBridgeExchangeSnapshotMetadataFlowsAndBlocksCurrentOnly(t *testing.T) {
	bridge := NewBridge()
	dir := t.TempDir()
	stem := filepath.Join(dir, "snapshot_lifecycle")
	manifestPath := filepath.Join(dir, "dataset_manifest.json")

	manifest := map[string]interface{}{
		"dataset_id": "test_dataset",
		"hashes": map[string]interface{}{
			"dataset_hash":  "d_hash",
			"manifest_hash": "m_hash",
		},
		"survivorship": map[string]interface{}{
			"universe_id":                                   "test_universe",
			"universe_hash":                                 "u_hash_123",
			"universe_manifest_hash":                        "um_hash_123",
			"universe_policy":                               "POINT_IN_TIME_EXCHANGE_UNIVERSE",
			"includes_delisted_assets":                      "unknown",
			"survivorship_bias_risk":                        "MEDIUM",
			"lifecycle_id":                                  "life_1",
			"lifecycle_hash":                                "life_hash_123",
			"lifecycle_manifest_hash":                       "life_manifest_hash_123",
			"lifecycle_evidence_level_summary":              map[string]interface{}{"HISTORICAL_SNAPSHOT_EVIDENCE": 1},
			"lifecycle_warnings":                            []string{"LIFECYCLE_EXCHANGE_SNAPSHOT_CURRENT_ONLY"},
			"listing_evidence_status":                       "VERIFIED",
			"delisting_evidence_status":                     "MISSING",
			"survivorship_support_status":                   "ELEVATED",
			"exchange_metadata_snapshot_hash":               "snap_hash_123",
			"exchange_metadata_snapshot_manifest_hash":      "snap_manifest_hash_123",
			"exchange_metadata_snapshot_archive_hash":       "snap_archive_hash_123",
			"exchange_metadata_snapshot_coverage_start_utc": "2024-01-15T00:00:00Z",
			"exchange_metadata_snapshot_coverage_end_utc":   "2024-01-15T00:00:00Z",
			"exchange_metadata_snapshot_evidence_level":     "HISTORICAL_SNAPSHOT_EVIDENCE",
			"exchange_metadata_snapshot_current_only":       true,
			"point_in_time_coverage_status":                 "CURRENT_ONLY",
		},
	}
	b, _ := json.Marshal(manifest)
	_ = os.WriteFile(manifestPath, b, 0644)

	out, err := bridge.EvaluateAndEmit(
		stem, "CAND_1", "v1", "git123", nil, nil, "chash", "cfghash",
		[]float64{0.01, -0.01, 0.02}, []int64{100, 200, 300}, 2, 35,
		true,
		manifestPath,
	)
	if err != nil {
		t.Fatalf("EvaluateAndEmit failed: %v", err)
	}
	if out.IntegrityPassed {
		t.Fatalf("Expected current-only snapshot evidence to block promotion")
	}
	for _, code := range []string{"RIF_EXCHANGE_SNAPSHOT_CURRENT_ONLY", "RIF_POINT_IN_TIME_EVIDENCE_PARTIAL", "RIF_SNAPSHOT_ARCHIVE_DOES_NOT_COVER_RESEARCH_WINDOW", "RIF_DELISTING_NOT_PROVEN"} {
		found := false
		for _, w := range out.Warnings {
			if stringsContains(w, code) {
				found = true
				break
			}
		}
		if !found {
			t.Fatalf("Expected %s in warnings, got %v", code, out.Warnings)
		}
	}

	lockData, err := os.ReadFile(stem + ".research.lock")
	if err != nil {
		t.Fatalf("Expected research.lock: %v", err)
	}
	var lock map[string]interface{}
	if err := json.Unmarshal(lockData, &lock); err != nil {
		t.Fatalf("parse lock: %v", err)
	}
	if lock["exchange_metadata_snapshot_hash"] != "snap_hash_123" || lock["point_in_time_coverage_status"] != "CURRENT_ONLY" {
		t.Fatalf("snapshot metadata missing from lock: %#v", lock)
	}

	auditData, err := os.ReadFile(stem + ".research_audit.json")
	if err != nil {
		t.Fatalf("Expected research_audit.json: %v", err)
	}
	var audit map[string]interface{}
	if err := json.Unmarshal(auditData, &audit); err != nil {
		t.Fatalf("parse audit: %v", err)
	}
	prov := audit["dataset_provenance"].(map[string]interface{})
	if prov["exchange_metadata_snapshot_manifest_hash"] != "snap_manifest_hash_123" || prov["exchange_metadata_snapshot_current_only"] != true {
		t.Fatalf("snapshot metadata missing from audit provenance: %#v", prov)
	}
}

func stringsContains(s, substr string) bool {
	return len(s) >= len(substr) && func() bool {
		for i := 0; i <= len(s)-len(substr); i++ {
			if s[i:i+len(substr)] == substr {
				return true
			}
		}
		return false
	}()
}
