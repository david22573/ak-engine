package strategy

import (
	"context"
	"testing"

	"github.com/davidmiguel22573/ak-engine/pkg/protocol"
)

func TestBaselineSignalsMomentum(t *testing.T) {
	t.Parallel()

	strat, err := NewBaseline(BaselineConfig{
		ThresholdBPS:   10,
		StopLossBPS:    50,
		TakeProfitBPS:  100,
		MaxHoldCandles: 3,
	})
	if err != nil {
		t.Fatalf("NewBaseline() error = %v", err)
	}

	state := State{
		Candles: []protocol.Candle{{Close: 100}},
	}

	signal, err := strat.OnCandle(context.Background(), state, protocol.Candle{Close: 101})
	if err != nil {
		t.Fatalf("OnCandle() error = %v", err)
	}
	if signal.Side != SideLong {
		t.Fatalf("signal.Side = %q, want %q", signal.Side, SideLong)
	}

	signal, err = strat.OnCandle(context.Background(), state, protocol.Candle{Close: 99})
	if err != nil {
		t.Fatalf("OnCandle() error = %v", err)
	}
	if signal.Side != SideShort {
		t.Fatalf("signal.Side = %q, want %q", signal.Side, SideShort)
	}
}

func TestBaselineNoHistoryNoSignal(t *testing.T) {
	t.Parallel()

	strat, err := NewBaseline(BaselineConfig{
		ThresholdBPS:   0,
		StopLossBPS:    50,
		TakeProfitBPS:  100,
		MaxHoldCandles: 3,
	})
	if err != nil {
		t.Fatalf("NewBaseline() error = %v", err)
	}

	signal, err := strat.OnCandle(context.Background(), State{}, protocol.Candle{Close: 100})
	if err != nil {
		t.Fatalf("OnCandle() error = %v", err)
	}
	if signal.Side != SideNone {
		t.Fatalf("signal.Side = %q, want none", signal.Side)
	}
}
