package backtest

import "github.com/davidmiguel22573/ak-engine/internal/strategy"

type FastAccumulationSummary struct {
	WindowDecisionCount int `json:"window_decision_count"`
	FullTradeCount      int `json:"full_trade_count"`
	ProbeTradeCount     int `json:"probe_trade_count"`
	HardBlockCount      int `json:"hard_block_count"`
	HoldCount           int `json:"hold_count"`
	ExitCount           int `json:"exit_count"`
	ReverseCount        int `json:"reverse_count"`
}

type FastAccumulationReport struct {
	DecisionSummary      FastAccumulationSummary         `json:"decision_summary"`
	Config               strategy.FastAccumulationConfig `json:"config"`
	TradesByAction       map[string]int                  `json:"trades_by_action"`
	PnLByAction          map[string]float64              `json:"pnl_by_action"`
	TradesByScoreBucket  map[string]int                  `json:"trades_by_score_bucket"`
	PnLByScoreBucket     map[string]float64              `json:"pnl_by_score_bucket"`
	WinRateByScoreBucket map[string]float64              `json:"win_rate_by_score_bucket"`
	AvgPnLByScoreBucket  map[string]float64              `json:"avg_pnl_by_score_bucket"`
	HardBlocksByReason   map[string]int                  `json:"hard_blocks_by_reason"`
	LossesByReasonCode   map[string]int                  `json:"losses_by_reason_code"`
	FeesByAction         map[string]float64              `json:"fees_by_action"`
	SlippageByAction     map[string]float64              `json:"slippage_by_action"`
	LongVsShortMetrics   map[string]float64              `json:"long_vs_short_metrics"`
	AvgMFEBPS            float64                         `json:"avg_mfe_bps"`
	AvgMAEBPS            float64                         `json:"avg_mae_bps"`
	AvgMFER              float64                         `json:"avg_mfe_r"`
	AvgMAER              float64                         `json:"avg_mae_r"`
	MedianMFER           float64                         `json:"median_mfe_r"`
	MedianMAER           float64                         `json:"median_mae_r"`
	WinnerAvgMFER        float64                         `json:"winner_avg_mfe_r"`
	LoserAvgMFER         float64                         `json:"loser_avg_mfe_r"`
	WinnerAvgMAER        float64                         `json:"winner_avg_mae_r"`
	LoserAvgMAER         float64                         `json:"loser_avg_mae_r"`
	AvgRealizedR         float64                         `json:"avg_realized_r"`
	AvgMaxPossibleR      float64                         `json:"avg_max_possible_r"`
	AvgAdverseR          float64                         `json:"avg_adverse_r"`
	WinnerAvgRealizedR   float64                         `json:"winner_avg_realized_r"`
	LoserAvgRealizedR    float64                         `json:"loser_avg_realized_r"`
	MFEByAction          map[string]float64              `json:"mfe_by_action"`
	MAEByAction          map[string]float64              `json:"mae_by_action"`
	MFEByScoreBucket     map[string]float64              `json:"mfe_by_score_bucket"`
	MAEByScoreBucket     map[string]float64              `json:"mae_by_score_bucket"`
	MFEByReasonCode      map[string]float64              `json:"mfe_by_reason_code"`
	MAEByReasonCode      map[string]float64              `json:"mae_by_reason_code"`
}

type Report struct {
	Source               string                    `json:"source"`
	Market               string                    `json:"market"`
	Symbol               string                    `json:"symbol"`
	Interval             string                    `json:"interval"`
	Strategy             string                    `json:"strategy"`
	PresetName           string                    `json:"preset_name,omitempty"`
	FromMS               int64                     `json:"from_ms"`
	ToMS                 int64                     `json:"to_ms"`
	TotalCandles         int                       `json:"total_candles"`
	TotalTrades          int                       `json:"total_trades"`
	Wins                 int                       `json:"wins"`
	Losses               int                       `json:"losses"`
	WinRate              float64                   `json:"win_rate"`
	GrossPnL             float64                   `json:"gross_pnl"`
	NetPnL               float64                   `json:"net_pnl"`
	FeesPaid             float64                   `json:"fees_paid"`
	SlippagePaid         float64                   `json:"slippage_paid"`
	ProfitFactor         float64                   `json:"profit_factor"`
	MaxDrawdown          float64                   `json:"max_drawdown"`
	MaxConsecutiveLosses int                       `json:"max_consecutive_losses"`
	AverageWin           float64                   `json:"average_win"`
	AverageLoss          float64                   `json:"average_loss"`
	Expectancy           float64                   `json:"expectancy"`
	AverageHoldMinutes   float64                   `json:"average_hold_minutes"`
	Status               string                    `json:"status"`
	PromotionCandidate   bool                      `json:"promotion_candidate"`
	RejectionReasons     []string                  `json:"rejection_reasons"`
	StartingCash         float64                   `json:"starting_cash"`
	EndingCash           float64                   `json:"ending_cash"`
	MaxPosition          float64                   `json:"max_position_size"`
	SlippageBPS          float64                   `json:"slippage_bps"`
	MakerFeeBPS          float64                   `json:"maker_fee_bps"`
	TakerFeeBPS          float64                   `json:"taker_fee_bps"`
	WindowDecisionCount  int                       `json:"window_decision_count"`
	FullTradeCount       int                       `json:"full_trade_count"`
	ProbeTradeCount      int                       `json:"probe_trade_count"`
	HardBlockCount       int                       `json:"hard_block_count"`
	HoldCount            int                       `json:"hold_count"`
	ExitCount            int                       `json:"exit_count"`
	ReverseCount         int                       `json:"reverse_count"`
	FastAccumulation     *FastAccumulationReport   `json:"fast_accumulation,omitempty"`
	Decisions            []strategy.WindowDecision `json:"decisions,omitempty"`
	Trades               []Trade                   `json:"trades"`
	Metrics              Metrics                   `json:"metrics"`
	GeneratedAtMS        int64                     `json:"generated_at_ms"`
}
