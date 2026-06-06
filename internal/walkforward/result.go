package walkforward

type Params struct {
	StrategyName                      string  `json:"strategy_name,omitempty"`
	DecisionWindowMinutes             int     `json:"decision_window_minutes,omitempty"`
	EntryVariant                      string  `json:"entry_variant,omitempty"`
	ExitModel                         string  `json:"exit_model,omitempty"`
	FullTradeMinScore                 float64 `json:"full_trade_min_score"`
	NormalTradeMinScore               float64 `json:"normal_trade_min_score"`
	ProbeMinScore                     float64 `json:"probe_min_score"`
	CostMultipleRequired              float64 `json:"cost_multiple_required"`
	MaxHoldWindows                    int     `json:"max_hold_windows"`
	TimeStopWindows                   int     `json:"time_stop_windows"`
	AllowProbeTrade                   bool    `json:"allow_probe_trade"`
	DisableProbeTrades                bool    `json:"disable_probe_trades,omitempty"`
	MaxChopScore                      float64 `json:"max_chop_score,omitempty"`
	MinEntryScore                     float64 `json:"min_entry_score,omitempty"`
	MinTrendScore                     float64 `json:"min_trend_score,omitempty"`
	DisableScoreBucket55To69          bool    `json:"disable_score_bucket_55_69,omitempty"`
	RequireScoreBucket70Plus          bool    `json:"require_score_bucket_70_plus,omitempty"`
	MinExpectedMoveBPS                float64 `json:"min_expected_move_bps,omitempty"`
	RequireExpectedMoveGtCostMultiple bool    `json:"require_expected_move_gt_cost_multiple,omitempty"`
	LongEnabled                       bool    `json:"long_enabled"`
	ShortEnabled                      bool    `json:"short_enabled"`
	LongMinEntryScore                 float64 `json:"long_min_entry_score"`
	ShortMinEntryScore                float64 `json:"short_min_entry_score"`
	LongMinTrendScore                 float64 `json:"long_min_trend_score"`
	ShortMinTrendScore                float64 `json:"short_min_trend_score"`
	LongMaxChopScore                  float64 `json:"long_max_chop_score"`
	ShortMaxChopScore                 float64 `json:"short_max_chop_score"`
	LongMinExpectedMoveBPS            float64 `json:"long_min_expected_move_bps"`
	ShortMinExpectedMoveBPS           float64 `json:"short_min_expected_move_bps"`
	LongCostMultipleRequired          float64 `json:"long_cost_multiple_required"`
	ShortCostMultipleRequired         float64 `json:"short_cost_multiple_required"`
	DisableLongScoreBucket70To84      bool    `json:"disable_long_score_bucket_70_84"`
	DisableShortScoreBucket70To84     bool    `json:"disable_short_score_bucket_70_84"`
	MaxTradesPerDay                   int     `json:"max_trades_per_day"`
	MinMinutesBetweenEntries          int     `json:"min_minutes_between_entries"`
	MinExpectedRAfterCost             float64 `json:"min_expected_r_after_cost"`
	MinTargetBPSAfterCost             float64 `json:"min_target_bps_after_cost"`
	MinRewardToRisk                   float64 `json:"min_reward_to_risk"`
	PartialTakeProfitR                float64 `json:"partial_take_profit_r"`
	PartialTakeProfitFraction         float64 `json:"partial_take_profit_fraction"`
	BreakevenTriggerR                 float64 `json:"breakeven_trigger_r"`
	TrailAfterMFER                    float64 `json:"trail_after_mfe_r"`
	TrailDistanceR                    float64 `json:"trail_distance_r"`
	CutNoProgressR                    float64 `json:"cut_no_progress_r"`
	CutNoProgressWindows              int     `json:"cut_no_progress_windows"`
}

type DiagnosticSummary struct {
	PnLByAction          map[string]float64 `json:"pnl_by_action"`
	PnLByScoreBucket     map[string]float64 `json:"pnl_by_score_bucket"`
	WinRateByScoreBucket map[string]float64 `json:"win_rate_by_score_bucket"`
	AvgPnLByScoreBucket  map[string]float64 `json:"avg_pnl_by_score_bucket"`
	PnLByReasonCode      map[string]float64 `json:"pnl_by_reason_code"`
	LossesByReasonCode   map[string]int     `json:"losses_by_reason_code"`
	FeesByAction         map[string]float64 `json:"fees_by_action"`
	SlippageByAction     map[string]float64 `json:"slippage_by_action"`
	LongVsShortMetrics   map[string]float64 `json:"long_vs_short_metrics"`
	HardBlocksByReason   map[string]int     `json:"hard_blocks_by_reason"`
}

type CandidateResult struct {
	Params               Params            `json:"params"`
	TotalTrades          int               `json:"total_trades"`
	Wins                 int               `json:"wins"`
	Losses               int               `json:"losses"`
	WinRate              float64           `json:"win_rate"`
	NetPnL               float64           `json:"net_pnl"`
	FeesPaid             float64           `json:"fees_paid"`
	SlippagePaid         float64           `json:"slippage_paid"`
	ProfitFactor         float64           `json:"profit_factor"`
	MaxDrawdown          float64           `json:"max_drawdown"`
	MaxConsecutiveLosses int               `json:"max_consecutive_losses"`
	Expectancy           float64           `json:"expectancy"`
	AverageHoldMinutes   float64           `json:"average_hold_minutes"`
	EndingCash           float64           `json:"ending_cash"`
	Status               string            `json:"status"`
	Diagnostics          DiagnosticSummary `json:"diagnostics,omitempty"`

	GrossWins   float64 `json:"-"`
	GrossLosses float64 `json:"-"`
}

type SplitResult struct {
	SplitIndex                   int               `json:"split_index"`
	TrainStartMs                 int64             `json:"train_start_ms"`
	TrainEndMs                   int64             `json:"train_end_ms"`
	TestStartMs                  int64             `json:"test_start_ms"`
	TestEndMs                    int64             `json:"test_end_ms"`
	TrainCandidateCount          int               `json:"train_candidate_count"`
	SelectedCandidates           []CandidateResult `json:"selected_candidates"`
	TestResults                  []CandidateResult `json:"test_results"`
	BestTrainCandidate           CandidateResult   `json:"best_train_candidate"`
	CorrespondingTestResult      CandidateResult   `json:"corresponding_test_result"`
	BestTestResult               CandidateResult   `json:"best_test_result"`
	TrainToTestPnLDelta          float64           `json:"train_to_test_pnl_delta"`
	TrainToTestProfitFactorDelta float64           `json:"train_to_test_profit_factor_delta"`
	TrainToTestExpectancyDelta   float64           `json:"train_to_test_expectancy_delta"`
	Status                       string            `json:"status"`
}

type AggregateMetrics struct {
	TotalTrades          int               `json:"total_trades"`
	Wins                 int               `json:"wins"`
	Losses               int               `json:"losses"`
	WinRate              float64           `json:"win_rate"`
	NetPnL               float64           `json:"net_pnl"`
	FeesPaid             float64           `json:"fees_paid"`
	SlippagePaid         float64           `json:"slippage_paid"`
	ProfitFactor         float64           `json:"profit_factor"`
	MaxDrawdown          float64           `json:"max_drawdown"`
	MaxConsecutiveLosses int               `json:"max_consecutive_losses"`
	Expectancy           float64           `json:"expectancy"`
	AverageHoldMinutes   float64           `json:"average_hold_minutes"`
	ProfitableSplitCount int               `json:"profitable_split_count"`
	LosingSplitCount     int               `json:"losing_split_count"`
	Diagnostics          DiagnosticSummary `json:"diagnostics,omitempty"`
}

type WalkForwardResult struct {
	Source                 string               `json:"source"`
	Market                 string               `json:"market"`
	Symbol                 string               `json:"symbol"`
	Interval               string               `json:"interval"`
	Strategy               string               `json:"strategy"`
	FromMs                 int64                `json:"from_ms"`
	ToMs                   int64                `json:"to_ms"`
	TrainWindow            string               `json:"train_window"`
	TestWindow             string               `json:"test_window"`
	SplitCount             int                  `json:"split_count"`
	CandidateCount         int                  `json:"candidate_count"`
	SelectedCandidateCount int                  `json:"selected_candidate_count"`
	AggregateTrain         AggregateMetrics     `json:"aggregate_train"`
	AggregateTest          AggregateMetrics     `json:"aggregate_test"`
	Splits                 []SplitResult        `json:"splits"`
	CandidateStability     []CandidateStability `json:"candidate_stability"`
	Status                 string               `json:"status"`
	PromotionCandidate     bool                 `json:"promotion_candidate"`
	RejectionReasons       []string             `json:"rejection_reasons"`
}

type CandidateStability struct {
	CandidateSignature   string  `json:"candidate_signature"`
	SelectionCount       int     `json:"selection_count"`
	TrainNetPnLTotal     float64 `json:"train_net_pnl_total"`
	TestNetPnLTotal      float64 `json:"test_net_pnl_total"`
	TrainProfitFactorAvg float64 `json:"train_profit_factor_avg"`
	TestProfitFactorAvg  float64 `json:"test_profit_factor_avg"`
	TestProfitableCount  int     `json:"test_profitable_count"`
	TestLosingCount      int     `json:"test_losing_count"`
	MaxTestDrawdown      float64 `json:"max_test_drawdown"`
	MaxTestLossStreak    int     `json:"max_test_loss_streak"`
}
