package app

import (
	"encoding/json"
	"strings"
	"testing"
	"time"
)

func createValidEvent() CompactRetainedEvent {
	return CompactRetainedEvent{
		CandidateID:    "TestCandidate",
		Symbol:         "BTCUSDT",
		Side:           1,
		EventTimestamp: time.Date(2025, 1, 15, 10, 0, 0, 0, time.UTC).Unix(),
		Horizon:        240,
		PreEntry: PreEntryContext{
			TrendRegime:      "downtrend",
			VolatilityBucket: "mid",
			FundingBucket:    "neutral",
			BTCContextBucket: "bullish",
			ETHContextBucket: "bearish",
		},
		Cluster: ClusterContext{
			Key:       "CL-1",
			Timestamp: time.Date(2025, 1, 15, 8, 0, 0, 0, time.UTC).Unix(),
			SpacingMs: 7200000,
			Size:      2,
			Ordinal:   1,
		},
		GrossOutcomeBps: 20.0,
		NetOutcome5Bps:  10.0,
		NetOutcome75Bps: 5.0,
		NetOutcome10Bps: 0.0,
		Win5Bps:         true,
		Win75Bps:        true,
		Win10Bps:        false,
		OutcomeLabel:    "TP",
	}
}

func TestCompactRetainedEvent_Valid(t *testing.T) {
	e := createValidEvent()

	err := e.Validate()
	if err != nil {
		t.Fatalf("Expected valid event, got error: %v", err)
	}

	// Test DeriveTimeFields
	if e.Month != "2025-01" {
		t.Errorf("Expected Month 2025-01, got %s", e.Month)
	}
	if e.Quarter != "2025-Q1" {
		t.Errorf("Expected Quarter 2025-Q1, got %s", e.Quarter)
	}

	j, err := e.ToJSON()
	if err != nil {
		t.Fatalf("Failed to serialize: %v", err)
	}

	if len(j) > 1024 {
		t.Errorf("Size too large: %d", len(j))
	}
	t.Logf("Serialized size: %d", len(j))
}

func TestCompactRetainedEvent_MissingRequired(t *testing.T) {
	e := createValidEvent()
	e.PreEntry.TrendRegime = ""

	err := e.Validate()
	if err == nil {
		t.Fatal("Expected error for missing trend regime, got none")
	}
	if !strings.Contains(err.Error(), "pre_entry_trend_regime is required") {
		t.Errorf("Unexpected error message: %v", err)
	}
}

func TestCompactRetainedEvent_LeakyFieldRejection(t *testing.T) {
	e := createValidEvent()

	// PreEntry doesn't have leaky fields (compile-time checked because it's a struct)
	// But we can test Diagnostic fields with raw/account data

	err := e.SetDiagnostic("raw_candle_payload", "large_data")
	if err == nil {
		t.Fatal("Expected error for raw_candle_payload, got none")
	}

	err = e.SetDiagnostic("account_key", "secret123")
	if err == nil {
		t.Fatal("Expected error for account_key, got none")
	}

	// Valid diagnostic field (leaky but allowed as diagnostic)
	err = e.SetDiagnostic("future_max_drawdown", -15.5)
	if err != nil {
		t.Fatalf("Valid diagnostic field rejected: %v", err)
	}

	err = e.Validate()
	if err != nil {
		t.Fatalf("Event with valid diagnostic field failed validation: %v", err)
	}
}

func TestCompactEventWriter_JSONL(t *testing.T) {
	w := NewCompactEventWriter()

	e1 := createValidEvent()
	e2 := createValidEvent()
	e2.Symbol = "ETHUSDT"
	e2.Cluster.Key = "CL-2"

	if err := w.Write(e1); err != nil {
		t.Fatal(err)
	}
	if err := w.Write(e2); err != nil {
		t.Fatal(err)
	}

	jsonl, err := w.ToJSONL()
	if err != nil {
		t.Fatal(err)
	}

	lines := strings.Split(strings.TrimSpace(jsonl), "\n")
	if len(lines) != 2 {
		t.Fatalf("Expected 2 lines, got %d", len(lines))
	}

	// Round-trip check
	var eRound CompactRetainedEvent
	if err := json.Unmarshal([]byte(lines[0]), &eRound); err != nil {
		t.Fatal(err)
	}
	if eRound.Symbol != "BTCUSDT" {
		t.Errorf("Expected BTCUSDT, got %s", eRound.Symbol)
	}

	// Check cluster fields preserved
	if eRound.Cluster.Key != "CL-1" {
		t.Errorf("Expected CL-1, got %s", eRound.Cluster.Key)
	}

	// Check cost stress preserved
	if eRound.NetOutcome5Bps != 10.0 {
		t.Errorf("Expected NetOutcome5Bps=10.0, got %f", eRound.NetOutcome5Bps)
	}
}
