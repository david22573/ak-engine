package backtest

import "github.com/davidmiguel22573/ak-engine/internal/strategy"

type Position struct {
	Symbol           string
	Market           string
	Interval         string
	Side             strategy.Side
	EntryTimeMS      int64
	BaseEntryPrice   float64
	EntryPrice       float64
	Quantity         float64
	Notional         float64
	StopPrice        float64
	TargetPrice      float64
	MaxHoldCandles   int
	HeldCandles      int
	EntryFee         float64
	Strategy         string
	EntryCandleIndex int
	EntryWindowMS    int64
	EntryReasonCodes []string
	ScoreAtEntry     float64
	RiskFraction     float64
	EstimatedCostBPS float64
	ExpectedMoveBPS  float64
	EntryAction      string
}
