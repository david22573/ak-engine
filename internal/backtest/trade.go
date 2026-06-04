package backtest

import "github.com/davidmiguel22573/ak-engine/internal/strategy"

type ExitReason string

const (
	ExitReasonStopLoss   ExitReason = "stop_loss"
	ExitReasonTakeProfit ExitReason = "take_profit"
	ExitReasonTimeStop   ExitReason = "time_stop"
	ExitReasonStrategy   ExitReason = "strategy_exit"
	ExitReasonReverse    ExitReason = "reverse"
	ExitReasonEndOfData  ExitReason = "END_OF_DATA"
)

type Trade struct {
	Symbol           string        `json:"symbol"`
	Market           string        `json:"market"`
	Interval         string        `json:"interval"`
	Side             strategy.Side `json:"side"`
	EntryTimeMS      int64         `json:"entry_time_ms"`
	ExitTimeMS       int64         `json:"exit_time_ms"`
	EntryPrice       float64       `json:"entry_price"`
	ExitPrice        float64       `json:"exit_price"`
	Quantity         float64       `json:"quantity"`
	Notional         float64       `json:"notional"`
	StopPrice        float64       `json:"stop_price"`
	TargetPrice      float64       `json:"target_price"`
	MaxHoldCandles   int           `json:"max_hold_candles"`
	HeldCandles      int           `json:"held_candles"`
	EntryFee         float64       `json:"entry_fee"`
	ExitFee          float64       `json:"exit_fee"`
	SlippagePaid     float64       `json:"slippage_paid"`
	GrossPnL         float64       `json:"gross_pnl"`
	NetPnL           float64       `json:"net_pnl"`
	NetReturnBPS     float64       `json:"net_return_bps"`
	ExitReason       ExitReason    `json:"exit_reason"`
	Strategy         string        `json:"strategy"`
	EntryCandleIndex int           `json:"entry_candle_index"`
	ExitCandleIndex  int           `json:"exit_candle_index"`
	EntryWindowMS    int64         `json:"entry_window_ms"`
	ExitWindowMS     int64         `json:"exit_window_ms"`
	EntryReasonCodes []string      `json:"entry_reason_codes"`
	ScoreAtEntry     float64       `json:"score_at_entry"`
	RiskFraction     float64       `json:"risk_fraction"`
	EstimatedCostBPS float64       `json:"estimated_cost_bps"`
	ExpectedMoveBPS  float64       `json:"expected_move_bps"`
	RMultiple        float64       `json:"r_multiple"`
	MAEBPS           float64       `json:"mae_bps"`
	MFEBPS           float64       `json:"mfe_bps"`
	HoldWindows      int           `json:"hold_windows"`
	EntryAction      string        `json:"entry_action"`
}
