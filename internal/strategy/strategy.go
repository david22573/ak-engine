package strategy

import (
	"context"

	"github.com/davidmiguel22573/ak-engine/pkg/protocol"
)

type Side string

const (
	SideNone  Side = ""
	SideLong  Side = "long"
	SideShort Side = "short"
)

type Signal struct {
	Side           Side
	StopLossBPS    float64
	TakeProfitBPS  float64
	MaxHoldCandles int
	RiskFraction   float64
	ClosePosition  bool
	Decision       *WindowDecision
}

type State struct {
	Candles            []protocol.Candle
	HasPosition        bool
	PositionSide       Side
	PositionEntryPrice float64
	HeldCandles        int
}

type Strategy interface {
	Name() string
	OnCandle(ctx context.Context, state State, candle protocol.Candle) (Signal, error)
}
