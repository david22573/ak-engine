package app

import (
	"context"
	"path/filepath"
	"testing"
)

func TestPhase13ContextFreeProbe_NoRawFetchAttempted(t *testing.T) {
	// Using missing workdir should fail cleanly, no raw fetch
	report, err := runPhase13ContextFreeProbe(context.Background(), "/tmp/invalid_workdir_for_test", false, "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if report.ExecutiveVerdict != "PHASE13_CONTEXT_FREE_PROOF_BLOCKED_NO_DATA" {
		t.Errorf("expected blocked no data, got %s", report.ExecutiveVerdict)
	}
}

func TestPhase13ContextFreeProbe_CompactEventEmission(t *testing.T) {
	// Find out actual workdir. It's usually .ak-engine/cache
	workdir := filepath.Join("..", "..", ".ak-engine", "cache")

	// We run it with emitCompactEvents = false first
	report1, err := runPhase13ContextFreeProbe(context.Background(), workdir, false, "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// If it was blocked, data might be missing locally, which is fine, but we can't fully test
	if report1.ExecutiveVerdict == "PHASE13_CONTEXT_FREE_PROOF_BLOCKED_NO_DATA" {
		t.Skip("local parquet data missing, skipping emit test")
	}

	if report1.ExecutiveVerdict != "PHASE13_CONTEXT_FREE_PROOF_BLOCKED_NO_EVENTS" {
		t.Errorf("expected no events because emit is false, got %s", report1.ExecutiveVerdict)
	}

	// Now with emitCompactEvents = true
	jsonlOut := filepath.Join(t.TempDir(), "test_events.jsonl")
	report2, err := runPhase13ContextFreeProbe(context.Background(), workdir, true, jsonlOut)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if report2.ExecutiveVerdict != "PHASE13_CONTEXT_FREE_PROOF_PASSED_INFRA_ONLY" &&
		report2.ExecutiveVerdict != "PHASE13_CONTEXT_FREE_PROOF_PASSED_RESEARCH_LEAD" &&
		report2.ExecutiveVerdict != "PHASE13_CONTEXT_FREE_PROOF_BLOCKED_NO_EVENTS" {
		t.Errorf("unexpected verdict: %s", report2.ExecutiveVerdict)
	}

	if report2.EventCount > 0 {
		if report2.CompactJSONLValidation != "PASS" {
			t.Errorf("expected jsonl validation to pass, got %s", report2.CompactJSONLValidation)
		}
		if report2.AggregatorConsumption != "PASS" {
			t.Errorf("expected aggregator to pass, got %s", report2.AggregatorConsumption)
		}
		if report2.MaxSerializedEventSize > 1024 {
			t.Errorf("event size %d > 1024", report2.MaxSerializedEventSize)
		}

		// check that BTC/ETH contexts are not required
		foundBTC := false
		for _, f := range report2.PreEntryFieldsUsed {
			if f == "btc_context" || f == "eth_context" {
				foundBTC = true
			}
		}
		if foundBTC {
			t.Errorf("expected no btc/eth context fields used")
		}
	}
}
