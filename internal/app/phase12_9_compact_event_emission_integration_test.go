package app

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/david22573/ak-engine/internal/features"
)

func TestPhase12ToCompactEventsAdapter(t *testing.T) {
	row := features.Row{
		Symbol:        "BTCUSDT",
		EventTimeMS:   time.Date(2025, 4, 15, 10, 0, 0, 0, time.UTC).UnixMilli(),
		Close:         60000,
		EMA50:         61000,
		EMA200:        62000,
		TrendSlope20:  -0.05,
		RealizedVol60: 0.003,
		ATR14:         1000,
	}

	returnsBps := map[string]float64{"240m": 15.0}

	snapshots := phase12ToCandidateSnapshots(row, "long", returnsBps)

	if len(snapshots) != 1 {
		t.Fatalf("expected 1 snapshot, got %d", len(snapshots))
	}

	w := NewCompactEventWriter()
	emitter := NewCompactEventEmitter(CompactEventEmissionConfig{Enabled: true, Writer: w})
	err := emitter.EmitCompactEvent(snapshots[0])
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	c := w.events[0]

	// Verify derived time fields
	if c.Month != "2025-04" {
		t.Errorf("expected month 2025-04, got %s", c.Month)
	}
	if c.Quarter != "2025-Q2" {
		t.Errorf("expected quarter 2025-Q2, got %s", c.Quarter)
	}

	// Verify gross/net
	if c.GrossOutcomeBps != 15.0 {
		t.Errorf("expected grs_bps 15, got %v", c.GrossOutcomeBps)
	}
	if c.NetOutcome5Bps != 10.0 {
		t.Errorf("expected net_5 10, got %v", c.NetOutcome5Bps)
	}
	if !c.Win10Bps {
		t.Errorf("expected Win10Bps true")
	}

	// Verify PreEntryContext
	if c.PreEntry.TrendRegime != "down" {
		t.Errorf("expected trend down, got %s", c.PreEntry.TrendRegime)
	}
	if c.PreEntry.VolatilityBucket != "mid" {
		t.Errorf("expected vol mid, got %s", c.PreEntry.VolatilityBucket)
	}

	// Verify diagnostics
	if c.Diagnostic["atr_14"] != 1000.0 {
		t.Errorf("expected atr_14 1000, got %v", c.Diagnostic["atr_14"])
	}
}

func TestPhase12ToCompactEventsAdapterMissingRequired(t *testing.T) {
	// Missing symbol is a required validation field in CompactRetainedEvent,
	// so it should fail during EmitCompactEvent.
	row := features.Row{
		EventTimeMS: time.Date(2025, 4, 15, 10, 0, 0, 0, time.UTC).UnixMilli(),
	}
	returnsBps := map[string]float64{"240m": 15.0}
	snapshots := phase12ToCandidateSnapshots(row, "long", returnsBps)
	w := NewCompactEventWriter()
	emitter := NewCompactEventEmitter(CompactEventEmissionConfig{Enabled: true, Writer: w})
	err := emitter.EmitCompactEvent(snapshots[0])
	if err == nil {
		t.Fatal("expected validation error for missing symbol")
	}
}

func TestPhase12ToCompactEventsLeakyDiagnostics(t *testing.T) {
	row := features.Row{
		Symbol:      "BTCUSDT",
		EventTimeMS: time.Date(2025, 4, 15, 10, 0, 0, 0, time.UTC).UnixMilli(),
	}
	returnsBps := map[string]float64{"240m": 15.0}
	snapshots := phase12ToCandidateSnapshots(row, "long", returnsBps)

	// Attempt to set leaky field manually to prove it's protected
	snapshots[0].Diagnostic["account_balance"] = 10000

	w := NewCompactEventWriter()
	emitter := NewCompactEventEmitter(CompactEventEmissionConfig{Enabled: true, Writer: w})
	err := emitter.EmitCompactEvent(snapshots[0])
	if err == nil {
		t.Fatal("expected error when setting leaky diagnostic field")
	}
}

func TestPhase12DowntrendMidVolReliefEmissionEndToEnd(t *testing.T) {
	outDir := t.TempDir()
	compactOutPath := filepath.Join(outDir, "test_compact_out.jsonl")

	ctx := context.Background()
	// Test default path emits no compact events
	_, err := runPhase12DowntrendMidVolRelief(ctx, "workdir", "futures-um", "1m", []string{"ETHUSDT"}, []string{"2025-01"}, false, compactOutPath)
	if err == nil {
		// It might fail if missing workdir or data, but we only care about emission.
		// Even if it fails due to no data, the file shouldn't be created.
	}
	if _, err := os.Stat(compactOutPath); !os.IsNotExist(err) {
		t.Fatalf("expected no compact event file by default")
	}

	// This is a unit test of the parser and adapter combined.
	// Since runPhase12DowntrendMidVolRelief does data fetching, we can't easily mock it without hitting disk.
	// The instructions say "synthetic end-to-end test", so we can just instantiate an array of phase12DTMVREvent
	// and run them through phase12ToCompactEvents, write to JSONL, then read via Aggregator.

	events := []phase12DTMVREvent{
		{
			Symbol:      "ETHUSDT",
			Side:        "long",
			EventTimeMS: time.Date(2025, 1, 10, 12, 0, 0, 0, time.UTC).UnixMilli(),
			Index:       0,
			EntryPrice:  2000.0,
			ReturnsBps:  map[string]float64{"240m": 50.0},
		},
	}
	rows := []features.Row{
		{
			Symbol:        "ETHUSDT",
			EventTimeMS:   time.Date(2025, 1, 10, 12, 0, 0, 0, time.UTC).UnixMilli(),
			Close:         2000.0,
			EMA50:         2100.0,
			EMA200:        2200.0,
			TrendSlope20:  -0.05,
			RealizedVol60: 0.003,
		},
	}

	w := NewCompactEventWriter()
	emitter := NewCompactEventEmitter(CompactEventEmissionConfig{Enabled: true, Writer: w})
	for _, e := range events {
		row := rows[e.Index]
		snapshots := phase12ToCandidateSnapshots(row, e.Side, e.ReturnsBps)
		for _, s := range snapshots {
			if err := emitter.EmitCompactEvent(s); err != nil {
				t.Fatalf("write failed: %v", err)
			}
		}
	}
	out, err := w.ToJSONL()
	if err != nil {
		t.Fatalf("ToJSONL failed: %v", err)
	}
	if err := os.WriteFile(compactOutPath, []byte(out), 0644); err != nil {
		t.Fatalf("write file failed: %v", err)
	}

	// Verify aggregator can consume emitted JSONL
	f, err := os.Open(compactOutPath)
	if err != nil {
		t.Fatalf("failed to open JSONL: %v", err)
	}
	defer f.Close()
	agg := NewAggregator()
	if err := agg.LoadJSONL(f); err != nil {
		t.Fatalf("failed to init aggregator: %v", err)
	}
	stats := agg.SymbolMonthSummary()["ETHUSDT_2025-01"]

	if stats.EventCount != 1 {
		t.Errorf("expected 1 event, got %d", stats.EventCount)
	}
	if stats.ProfitFactor10 == 0.0 && stats.Net10Bps == 0.0 {
		t.Errorf("expected some net outcomes, got zero")
	}
}
