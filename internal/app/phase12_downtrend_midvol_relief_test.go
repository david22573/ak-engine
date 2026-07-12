package app

import (
	"context"
	"testing"
	"strings"

	"github.com/david22573/ak-engine/internal/features"
)

func TestPhase12DTMVRSignalLongAccepted(t *testing.T) {
	rows := phase12DTMVRTestRows("long")

	side, gate := phase12DTMVRSignal(rows, 20)
	if side != "long" {
		t.Fatalf("side=%q, want long gate=%+v", side, gate)
	}
	if !gate.Regime || !gate.TrendAlignment || !gate.Pullback || !gate.Continuation || !gate.VolumeChop {
		t.Fatalf("expected all long gates to pass: %+v", gate)
	}
}

func TestPhase12DTMVRSignalRejectsWhenTrendNotDown(t *testing.T) {
	rows := phase12DTMVRTestRows("long")
	rows[20].Close = 105 // Break trend_down logic: row.Close < row.EMA50
	
	side, gate := phase12DTMVRSignal(rows, 20)
	if side != "" {
		t.Fatalf("side=%q, want rejected", side)
	}
	if gate.TrendAlignment {
		t.Fatalf("trend alignment gate passed unexpectedly: %+v", gate)
	}
}

func TestPhase12DTMVRSignalRejectsWhenVolNotMid(t *testing.T) {
	rows := phase12DTMVRTestRows("long")
	rows[20].RealizedVol60 = 0.01 // High Vol (not between 0.0015 and 0.006)
	
	side, gate := phase12DTMVRSignal(rows, 20)
	if side != "" {
		t.Fatalf("side=%q, want rejected", side)
	}
	if gate.VolumeChop {
		t.Fatalf("volume chop (volatility) gate passed unexpectedly: %+v", gate)
	}
}

func TestPhase12DTMVRSignalIsLongOnly(t *testing.T) {
	// The implementation simply returns "long" when conditions are met. 
	// We verify that even if we try to simulate a short condition, it either rejects or returns long.
	// Since our test rows are purely based on trend_down + vol_mid, we can just check there's no short side.
	rows := phase12DTMVRTestRows("long")
	
	side, _ := phase12DTMVRSignal(rows, 20)
	if side == "short" {
		t.Fatalf("side=%q, want long only", side)
	}
}

func TestPhase12DTMVRReportHasNoFundingPrimaryDependency(t *testing.T) {
	report, err := runPhase12DowntrendMidVolRelief(context.Background(), t.TempDir(), "futures-um", "1m", nil, []string{"2024-01"})
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

func TestPhase12DTMVRIs240mOnly(t *testing.T) {
	got := strings.Join(phase12DTMVRHorizons, ",")
	if got != "240m" {
		t.Fatalf("phase12DTMVRHorizons=%s, want 240m", got)
	}
}

func phase12DTMVRTestRows(side string) []features.Row {
	rows := make([]features.Row, 21)
	for i := range rows {
		// Valid DowntrendMidVolRelief row
		// Trend: Close < EMA50 && EMA50 < EMA200 && TrendSlope20 < 0
		// Vol: RealizedVol60 >= 0.0015 && RealizedVol60 <= 0.006
		rows[i] = features.Row{
			Symbol:           "TESTUSDT",
			EventTimeMS:      int64(i * 60000),
			AvailableAtMS:    int64(i * 60000),
			Close:            90,
			EMA20:            95,
			EMA50:            100,
			EMA200:           110,
			TrendSlope20:     -0.02,
			RealizedVol60:    0.003,
		}
	}
	return rows
}


