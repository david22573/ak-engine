package strategy

import (
	"context"
	"fmt"
	"math"

	"github.com/davidmiguel22573/ak-engine/pkg/protocol"
)

type BaselineConfig struct {
	ThresholdBPS   float64
	StopLossBPS    float64
	TakeProfitBPS  float64
	MaxHoldCandles int
}

type Baseline struct {
	cfg BaselineConfig
}

func NewBaseline(cfg BaselineConfig) (*Baseline, error) {
	if cfg.ThresholdBPS < 0 {
		return nil, fmt.Errorf("threshold_bps must be >= 0")
	}
	if cfg.StopLossBPS <= 0 {
		return nil, fmt.Errorf("stop_loss_bps must be > 0")
	}
	if cfg.TakeProfitBPS <= 0 {
		return nil, fmt.Errorf("take_profit_bps must be > 0")
	}
	if cfg.MaxHoldCandles <= 0 {
		return nil, fmt.Errorf("max_hold_candles must be > 0")
	}
	return &Baseline{cfg: cfg}, nil
}

func (b *Baseline) Name() string {
	return "baseline"
}

func (b *Baseline) OnCandle(_ context.Context, state State, candle protocol.Candle) (Signal, error) {
	if len(state.Candles) == 0 {
		return Signal{}, nil
	}

	prev := state.Candles[len(state.Candles)-1]
	if prev.Close <= 0 {
		return Signal{}, fmt.Errorf("previous candle close must be > 0")
	}

	changeBPS := ((candle.Close - prev.Close) / prev.Close) * 10000
	switch {
	case changeBPS > b.cfg.ThresholdBPS:
		return b.signal(SideLong), nil
	case changeBPS < -b.cfg.ThresholdBPS:
		return b.signal(SideShort), nil
	default:
		if math.Abs(changeBPS) <= b.cfg.ThresholdBPS {
			return Signal{}, nil
		}
		return Signal{}, nil
	}
}

func (b *Baseline) signal(side Side) Signal {
	return Signal{
		Side:           side,
		StopLossBPS:    b.cfg.StopLossBPS,
		TakeProfitBPS:  b.cfg.TakeProfitBPS,
		MaxHoldCandles: b.cfg.MaxHoldCandles,
	}
}
