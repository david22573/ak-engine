package app

import (
	"context"
	"testing"

	"github.com/david22573/ak-engine/internal/features"
)

func TestPhase11RTPCSignalLongAccepted(t *testing.T) {
	rows := phase11RTPCTestRows("long")

	side, gate := phase11RTPCSignal(rows, 20)
	if side != "long" {
		t.Fatalf("side=%q, want long gate=%+v", side, gate)
	}
	if !gate.Regime || !gate.TrendAlignment || !gate.Pullback || !gate.Continuation || !gate.VolumeChop {
		t.Fatalf("expected all long gates to pass: %+v", gate)
	}
}

func TestPhase11RTPCSignalLongRejectsBrokenStructure(t *testing.T) {
	rows := phase11RTPCTestRows("long")
	for i := 5; i < 10; i++ {
		rows[i].Close = 95
		rows[i].Return15 = -0.05
	}

	side, gate := phase11RTPCSignal(rows, 20)
	if side != "" {
		t.Fatalf("side=%q, want rejected", side)
	}
	if gate.Pullback {
		t.Fatalf("pullback gate passed despite broken structure: %+v", gate)
	}
}

func TestPhase11RTPCSignalShortAccepted(t *testing.T) {
	rows := phase11RTPCTestRows("short")

	side, gate := phase11RTPCSignal(rows, 20)
	if side != "short" {
		t.Fatalf("side=%q, want short gate=%+v", side, gate)
	}
	if !gate.Regime || !gate.TrendAlignment || !gate.Pullback || !gate.Continuation || !gate.VolumeChop {
		t.Fatalf("expected all short gates to pass: %+v", gate)
	}
}

func TestPhase11RTPCSignalShortRejectsTrendRegimeFailure(t *testing.T) {
	rows := phase11RTPCTestRows("short")
	rows[20].Close = 99
	rows[20].EMA50 = 99
	rows[20].EMA200 = 99
	rows[20].TrendSlope20 = 0.03
	rows[20].BTCReturn60 = 0.004
	rows[20].ETHReturn60 = 0.004

	side, gate := phase11RTPCSignal(rows, 20)
	if side != "" {
		t.Fatalf("side=%q, want rejected", side)
	}
	if gate.Regime {
		t.Fatalf("regime gate passed unexpectedly: %+v", gate)
	}
}

func TestPhase11RTPCReportHasNoFundingPrimaryDependency(t *testing.T) {
	report, err := runPhase11RegimeTrendPullbackContinuation(context.Background(), t.TempDir(), "futures-um", "1m", nil, []string{"2024-01"})
	if err != nil {
		t.Fatalf("run report with missing local data: %v", err)
	}
	if report.FundingPrimaryTrigger {
		t.Fatalf("funding_primary_trigger=true, want false")
	}
	if report.RawEventDetailRetained {
		t.Fatalf("raw_event_detail_retained=true, want false")
	}
	if report.AKTraderTouched {
		t.Fatalf("ak_trader_touched=true, want false")
	}
}

func TestPhase11RTPCDoesNotAlterCompressionVolumeBreakoutMatrix(t *testing.T) {
	if phase11CVBFamily != "CompressionVolumeBreakout" {
		t.Fatalf("phase11CVBFamily=%q", phase11CVBFamily)
	}
	got := stringsJoin(phase11CVBHorizons, ",")
	if got != "15m,60m,240m" {
		t.Fatalf("phase11CVBHorizons=%s", got)
	}
}

func phase11RTPCTestRows(side string) []features.Row {
	rows := make([]features.Row, 21)
	for i := range rows {
		rows[i] = features.Row{
			Symbol:           "TESTUSDT",
			EventTimeMS:      int64(i * 60000),
			AvailableAtMS:    int64(i * 60000),
			Close:            100,
			EMA20:            100,
			EMA50:            99,
			EMA200:           98.5,
			TrendSlope20:     0.02,
			Return15:         -0.001,
			VolumeRatio20:    1.0,
			BBWidthPctRank60: 0.40,
			BTCReturn60:      0.001,
			ETHReturn60:      0.001,
		}
	}
	rows[8].Close = 99.8
	rows[19].Close = 99.9
	rows[20].Close = 100.8
	rows[20].Return15 = 0.002

	if side == "short" {
		for i := range rows {
			rows[i].Close = 100
			rows[i].EMA20 = 100
			rows[i].EMA50 = 101
			rows[i].EMA200 = 101.5
			rows[i].TrendSlope20 = -0.02
			rows[i].Return15 = 0.001
			rows[i].BTCReturn60 = -0.001
			rows[i].ETHReturn60 = -0.001
		}
		rows[8].Close = 100.2
		rows[19].Close = 100.1
		rows[20].Close = 99.2
		rows[20].Return15 = -0.002
	}
	return rows
}

func stringsJoin(values []string, sep string) string {
	out := ""
	for i, value := range values {
		if i > 0 {
			out += sep
		}
		out += value
	}
	return out
}
