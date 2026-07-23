package app

import (
	"encoding/json"
	"testing"
	"time"
)

// CompactRetainedEvent represents the minimal schema required to audit candidate strategies
// for risk filter validity, clustering, cost stress, and structural failure without needing
// raw candle data or full event payloads.
type SpecCompactRetainedEvent struct {
	// Identity & Context (Required)
	CandidateID    string `json:"candidate_id"`
	Symbol         string `json:"symbol"`
	Side           int    `json:"side"` // 1 for Long, -1 for Short
	EventTimestamp int64  `json:"ts"`

	// Derived Time Grouping (Useful for leave-one-out and aggregations)
	Month   string `json:"month"`   // e.g., "2024-01"
	Quarter string `json:"quarter"` // e.g., "2024-Q1"

	// Pre-Entry Risk Filter Fields (Required for phase 12.5 style audits)
	// These MUST be strictly pre-entry to prevent leaky filters.
	PreEntry struct {
		TrendRegime        string  `json:"trend"`     // e.g., "downtrend", "uptrend"
		TrendStrength      float64 `json:"trend_str"` // e.g., ADX or similar
		VolatilityBucket   string  `json:"vol"`       // e.g., "low", "mid", "high"
		FundingBucket      string  `json:"fund"`      // e.g., "negative", "neutral", "positive"
		BTCContextBucket   string  `json:"btc_ctx"`   // e.g., "bullish", "bearish"
		ETHContextBucket   string  `json:"eth_ctx"`   // e.g., "bullish", "bearish"
		VolumeLiquidBucket string  `json:"vol_liq"`   // e.g., "low_vol", "high_vol"
		EntryReasonCode    string  `json:"reason"`    // Setup classification
	} `json:"pre"`

	// Clustering & Deduplication (Required for phase 12.4 style audits)
	Cluster struct {
		Key       string `json:"key"`     // Unique ID for the cluster
		Timestamp int64  `json:"ts"`      // Timestamp of the first event in the cluster
		SpacingMs int64  `json:"spacing"` // Time since last cluster or previous event in cluster
		Size      int    `json:"size"`    // Total events in this cluster
		Ordinal   int    `json:"ordinal"` // This event's index within the cluster (1-based)
	} `json:"cluster"`

	// Outcomes (Required for cost stress and filter validation)
	Horizon         int     `json:"horizon"` // Duration of trade in minutes/candles
	GrossOutcomeBps float64 `json:"grs_bps"`
	NetOutcome5Bps  float64 `json:"net_5"`
	NetOutcome75Bps float64 `json:"net_75"`
	NetOutcome10Bps float64 `json:"net_10"`

	// Diagnostic / Optional (Useful but not strictly required for baseline audits)
	OutcomeLabel string `json:"label,omitempty"` // e.g., "TP", "SL", "TimeExit"
}

func TestCompactRetainedEventSchemaSize(t *testing.T) {
	evt := SpecCompactRetainedEvent{
		CandidateID:     "DowntrendMidVolReliefLong240m",
		Symbol:          "BTCUSDT",
		Side:            1,
		EventTimestamp:  time.Now().Unix(),
		Month:           "2025-01",
		Quarter:         "2025-Q1",
		Horizon:         240,
		GrossOutcomeBps: 25.5,
		NetOutcome5Bps:  15.5,
		NetOutcome75Bps: 10.5,
		NetOutcome10Bps: 5.5,
		OutcomeLabel:    "TP",
	}

	evt.PreEntry.TrendRegime = "downtrend"
	evt.PreEntry.TrendStrength = 35.2
	evt.PreEntry.VolatilityBucket = "mid"
	evt.PreEntry.FundingBucket = "neutral"
	evt.PreEntry.BTCContextBucket = "neutral"
	evt.PreEntry.ETHContextBucket = "neutral"
	evt.PreEntry.VolumeLiquidBucket = "mid"
	evt.PreEntry.EntryReasonCode = "relief_bounce"

	evt.Cluster.Key = "CL-20250101-BTC-1"
	evt.Cluster.Timestamp = evt.EventTimestamp - 3600
	evt.Cluster.SpacingMs = 3600000
	evt.Cluster.Size = 3
	evt.Cluster.Ordinal = 1

	data, err := json.Marshal(evt)
	if err != nil {
		t.Fatalf("Failed to marshal: %v", err)
	}

	t.Logf("Serialized Size: %d bytes", len(data))
	if len(data) > 600 { // We want this compact, definitely under 1KB, ideally ~300-500 bytes
		t.Errorf("Schema is too large: %d bytes", len(data))
	}
}
