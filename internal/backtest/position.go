package backtest

import "github.com/davidmiguel22573/ak-engine/internal/strategy"

type Position struct {
	Symbol             string
	Market             string
	Interval           string
	Side               strategy.Side
	EntryTimeMS        int64
	BaseEntryPrice     float64
	EntryPrice         float64
	Quantity           float64
	OriginalQuantity   float64
	Notional           float64
	OriginalNotional   float64
	StopPrice          float64
	InitialStopPrice   float64
	TargetPrice        float64
	InitialTargetPrice float64
	InitialRiskBPS     float64
	MaxHoldCandles     int
	HeldCandles        int
	EntryFee           float64
	Strategy           string
	EntryCandleIndex   int
	EntryWindowMS      int64
	EntryReasonCodes   []string
	ScoreAtEntry       float64
	RiskFraction       float64
	EstimatedCostBPS   float64
	ExpectedMoveBPS    float64
	EntryAction        string
	ExitPlan           strategy.ExitPlan
	RealizedGrossPnL   float64
	RealizedFees       float64
	RealizedSlippage   float64
	PartialExitCount   int
	BreakevenArmed     bool
	MaxFavorablePrice  float64
	MaxAdversePrice    float64
	TimeToMFEMinutes   float64
	TimeToMAEMinutes   float64
}
