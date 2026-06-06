package strategy

import (
	"context"
	"fmt"
	"math"
	"time"

	"github.com/davidmiguel22573/ak-engine/pkg/protocol"
)

type FastAccumulationConfig struct {
	StrategyName                      string       `json:"strategy_name,omitempty"`
	DecisionWindowMinutes             int          `json:"decision_window_minutes,omitempty"`
	EntryVariant                      EntryVariant `json:"entry_variant,omitempty"`
	ExitModel                         ExitModel    `json:"exit_model,omitempty"`
	ForceDecision                     bool         `json:"force_decision"`
	ForceFullTrade                    bool         `json:"force_full_trade"`
	AllowProbeTrade                   bool         `json:"allow_probe_trade"`
	FullTradeMinScore                 float64      `json:"full_trade_min_score"`
	NormalTradeMinScore               float64      `json:"normal_trade_min_score"`
	ProbeMinScore                     float64      `json:"probe_min_score"`
	CostMultipleRequired              float64      `json:"cost_multiple_required"`
	MaxHoldWindows                    int          `json:"max_hold_windows"`
	TimeStopWindows                   int          `json:"time_stop_windows"`
	AddToWinnersOnly                  bool         `json:"add_to_winners_only"`
	NoMartingale                      bool         `json:"no_martingale"`
	ProbeRiskFraction                 float64      `json:"probe_risk_fraction"`
	NormalRiskFraction                float64      `json:"normal_risk_fraction"`
	FullRiskFraction                  float64      `json:"full_risk_fraction"`
	EstimatedCostBPS                  float64      `json:"estimated_cost_bps"`
	MaxChopScore                      float64      `json:"max_chop_score"`
	DisableProbeTrades                bool         `json:"disable_probe_trades"`
	MinEntryScore                     float64      `json:"min_entry_score"`
	MinExpectedMoveBPS                float64      `json:"min_expected_move_bps"`
	MinTrendScore                     float64      `json:"min_trend_score"`
	DisableScoreBucket55To69          bool         `json:"disable_score_bucket_55_69"`
	RequireScoreBucket70Plus          bool         `json:"require_score_bucket_70_plus"`
	RequireExpectedMoveGtCostMultiple bool         `json:"require_expected_move_gt_cost_multiple"`
	LongEnabled                       bool         `json:"long_enabled"`
	ShortEnabled                      bool         `json:"short_enabled"`
	LongMinEntryScore                 float64      `json:"long_min_entry_score,omitempty"`
	ShortMinEntryScore                float64      `json:"short_min_entry_score,omitempty"`
	LongMinTrendScore                 float64      `json:"long_min_trend_score,omitempty"`
	ShortMinTrendScore                float64      `json:"short_min_trend_score,omitempty"`
	LongMaxChopScore                  float64      `json:"long_max_chop_score,omitempty"`
	ShortMaxChopScore                 float64      `json:"short_max_chop_score,omitempty"`
	LongMinExpectedMoveBPS            float64      `json:"long_min_expected_move_bps,omitempty"`
	ShortMinExpectedMoveBPS           float64      `json:"short_min_expected_move_bps,omitempty"`
	LongCostMultipleRequired          float64      `json:"long_cost_multiple_required,omitempty"`
	ShortCostMultipleRequired         float64      `json:"short_cost_multiple_required,omitempty"`
	DisableLongScoreBucket70To84      bool         `json:"disable_long_score_bucket_70_84,omitempty"`
	DisableShortScoreBucket70To84     bool         `json:"disable_short_score_bucket_70_84,omitempty"`
	MaxTradesPerDay                   int          `json:"max_trades_per_day,omitempty"`
	MinMinutesBetweenEntries          int          `json:"min_minutes_between_entries,omitempty"`
	MinExpectedRAfterCost             float64      `json:"min_expected_r_after_cost,omitempty"`
	MinTargetBPSAfterCost             float64      `json:"min_target_bps_after_cost,omitempty"`
	MinRewardToRisk                   float64      `json:"min_reward_to_risk,omitempty"`
	PartialTakeProfitR                float64      `json:"partial_take_profit_r,omitempty"`
	PartialTakeProfitFraction         float64      `json:"partial_take_profit_fraction,omitempty"`
	BreakevenTriggerR                 float64      `json:"breakeven_trigger_r,omitempty"`
	TrailAfterMFER                    float64      `json:"trail_after_mfe_r,omitempty"`
	TrailDistanceR                    float64      `json:"trail_distance_r,omitempty"`
	CutNoProgressR                    float64      `json:"cut_no_progress_r,omitempty"`
	CutNoProgressWindows              int          `json:"cut_no_progress_windows,omitempty"`
}

type FastAccumulation struct {
	cfg           FastAccumulationConfig
	aggWindow        *WindowAggregator
	completedWindows  []AggregatedWindow
	recentCandles []protocol.Candle
	decisions     []WindowDecision
}

func DefaultFastAccumulationConfig() FastAccumulationConfig {
	return FastAccumulationConfig{
		StrategyName:              "fast_accumulation",
		DecisionWindowMinutes:     15,
		EntryVariant:              EntryVariantScore,
		ExitModel:                 ExitModelFixedTPSL,
		ForceDecision:             true,
		ForceFullTrade:            false,
		AllowProbeTrade:           true,
		FullTradeMinScore:         85,
		NormalTradeMinScore:       70,
		ProbeMinScore:             55,
		CostMultipleRequired:      3,
		MaxHoldWindows:            4,
		TimeStopWindows:           2,
		AddToWinnersOnly:          true,
		NoMartingale:              true,
		ProbeRiskFraction:         0.25,
		NormalRiskFraction:        0.5,
		FullRiskFraction:          1.0,
		EstimatedCostBPS:          6.0,
		MaxChopScore:              70,
		LongEnabled:               true,
		ShortEnabled:              true,
		PartialTakeProfitR:        1.0,
		PartialTakeProfitFraction: 0.5,
		BreakevenTriggerR:         1.0,
		TrailAfterMFER:            1.2,
		TrailDistanceR:            0.5,
		CutNoProgressR:            0.3,
		CutNoProgressWindows:      2,
	}
}

func NewFastAccumulation(cfg FastAccumulationConfig) (*FastAccumulation, error) {
	if cfg.FullTradeMinScore < cfg.NormalTradeMinScore {
		return nil, fmt.Errorf("full_trade_min_score must be >= normal_trade_min_score")
	}
	if cfg.NormalTradeMinScore < cfg.ProbeMinScore {
		return nil, fmt.Errorf("normal_trade_min_score must be >= probe_min_score")
	}
	if cfg.CostMultipleRequired <= 0 {
		return nil, fmt.Errorf("cost_multiple_required must be > 0")
	}
	if cfg.MaxHoldWindows <= 0 {
		return nil, fmt.Errorf("max_hold_windows must be > 0")
	}
	if cfg.TimeStopWindows <= 0 {
		return nil, fmt.Errorf("time_stop_windows must be > 0")
	}
	if cfg.StrategyName == "" {
		cfg.StrategyName = "fast_accumulation"
	}
	if cfg.DecisionWindowMinutes <= 0 {
		cfg.DecisionWindowMinutes = 15
	}
	if cfg.EntryVariant == "" {
		cfg.EntryVariant = EntryVariantScore
	}
	if cfg.ExitModel == "" {
		cfg.ExitModel = ExitModelFixedTPSL
	}
	if cfg.ProbeRiskFraction <= 0 || cfg.ProbeRiskFraction > 1 {
		return nil, fmt.Errorf("probe_risk_fraction must be in (0, 1]")
	}
	if cfg.NormalRiskFraction <= 0 || cfg.NormalRiskFraction > 1 {
		return nil, fmt.Errorf("normal_risk_fraction must be in (0, 1]")
	}
	if cfg.FullRiskFraction <= 0 || cfg.FullRiskFraction > 1 {
		return nil, fmt.Errorf("full_risk_fraction must be in (0, 1]")
	}
	if !cfg.LongEnabled && !cfg.ShortEnabled {
		return nil, fmt.Errorf("invalid config: long_enabled and short_enabled cannot both be false")
	}
	if err := validateScoreBound("min_entry_score", cfg.MinEntryScore); err != nil {
		return nil, err
	}
	if err := validateScoreBound("min_trend_score", cfg.MinTrendScore); err != nil {
		return nil, err
	}
	if err := validateScoreBound("long_min_entry_score", cfg.LongMinEntryScore); err != nil {
		return nil, err
	}
	if err := validateScoreBound("short_min_entry_score", cfg.ShortMinEntryScore); err != nil {
		return nil, err
	}
	if err := validateScoreBound("long_min_trend_score", cfg.LongMinTrendScore); err != nil {
		return nil, err
	}
	if err := validateScoreBound("short_min_trend_score", cfg.ShortMinTrendScore); err != nil {
		return nil, err
	}
	if err := validateScoreBound("max_chop_score", cfg.MaxChopScore); err != nil {
		return nil, err
	}
	if err := validateScoreBound("long_max_chop_score", cfg.LongMaxChopScore); err != nil {
		return nil, err
	}
	if err := validateScoreBound("short_max_chop_score", cfg.ShortMaxChopScore); err != nil {
		return nil, err
	}
	if cfg.LongMinExpectedMoveBPS < 0 {
		return nil, fmt.Errorf("long_min_expected_move_bps must be >= 0")
	}
	if cfg.ShortMinExpectedMoveBPS < 0 {
		return nil, fmt.Errorf("short_min_expected_move_bps must be >= 0")
	}
	if cfg.LongCostMultipleRequired < 0 {
		return nil, fmt.Errorf("long_cost_multiple_required must be >= 0")
	}
	if cfg.ShortCostMultipleRequired < 0 {
		return nil, fmt.Errorf("short_cost_multiple_required must be >= 0")
	}
	if cfg.MaxTradesPerDay < 0 {
		return nil, fmt.Errorf("max_trades_per_day must be >= 0")
	}
	if cfg.MinMinutesBetweenEntries < 0 {
		return nil, fmt.Errorf("min_minutes_between_entries must be >= 0")
	}
	if cfg.MinExpectedRAfterCost < 0 {
		return nil, fmt.Errorf("min_expected_r_after_cost must be >= 0")
	}
	if cfg.MinTargetBPSAfterCost < 0 {
		return nil, fmt.Errorf("min_target_bps_after_cost must be >= 0")
	}
	if cfg.MinRewardToRisk < 0 {
		return nil, fmt.Errorf("min_reward_to_risk must be >= 0")
	}
	if cfg.PartialTakeProfitR < 0 {
		return nil, fmt.Errorf("partial_take_profit_r must be >= 0")
	}
	if cfg.PartialTakeProfitFraction < 0 || cfg.PartialTakeProfitFraction > 1 {
		return nil, fmt.Errorf("partial_take_profit_fraction must be in [0, 1]")
	}
	if cfg.BreakevenTriggerR < 0 {
		return nil, fmt.Errorf("breakeven_trigger_r must be >= 0")
	}
	if cfg.TrailAfterMFER < 0 {
		return nil, fmt.Errorf("trail_after_mfe_r must be >= 0")
	}
	if cfg.TrailDistanceR < 0 {
		return nil, fmt.Errorf("trail_distance_r must be >= 0")
	}
	if cfg.CutNoProgressR < 0 {
		return nil, fmt.Errorf("cut_no_progress_r must be >= 0")
	}
	if cfg.CutNoProgressWindows < 0 {
		return nil, fmt.Errorf("cut_no_progress_windows must be >= 0")
	}

	aggWindow, err := NewWindowAggregator(fmt.Sprintf("%dm", cfg.DecisionWindowMinutes))
	if err != nil {
		return nil, err
	}
	return &FastAccumulation{
		cfg:    cfg,
		aggWindow: aggWindow,
	}, nil
}

func (s *FastAccumulation) Name() string {
	return s.cfg.StrategyName
}

func (s *FastAccumulation) ConfigSnapshot() FastAccumulationConfig {
	return s.cfg
}

func (s *FastAccumulation) DecisionsSnapshot() []WindowDecision {
	out := make([]WindowDecision, len(s.decisions))
	copy(out, s.decisions)
	return out
}

func (s *FastAccumulation) OnCandle(_ context.Context, state State, candle protocol.Candle) (Signal, error) {
	s.recentCandles = append(s.recentCandles, candle)
	if len(s.recentCandles) > 24 {
		s.recentCandles = s.recentCandles[len(s.recentCandles)-24:]
	}

	window, err := s.aggWindow.Add(candle)
	if err != nil {
		return Signal{}, err
	}
	if window == nil {
		return Signal{}, nil
	}

	s.completedWindows = append(s.completedWindows, *window)
	hourly, _ := BuildHourlyContext(s.completedWindows)
	scored := ScoreWindow(ScoreInput{
		Window:               *window,
		Previous:             s.completedWindows[:len(s.completedWindows)-1],
		RecentCandles:        s.recentCandles,
		HourContext:          hourly,
		EstimatedCostBPS:     s.cfg.EstimatedCostBPS,
		CostMultipleRequired: s.scoringCostMultipleRequired(),
		MaxChopScore:         s.scoringMaxChopScore(),
	})

	decision := s.selectDecision(state, *window, scored)
	s.decisions = append(s.decisions, decision)
	signal := s.signalFromDecision(state, decision)
	signal.Decision = &decision
	return signal, nil
}

func (s *FastAccumulation) selectDecision(state State, window AggregatedWindow, scored ScoreResult) WindowDecision {
	decision := WindowDecision{
		Symbol:           window.Symbol,
		WindowStartMS:    window.WindowStartMS,
		WindowEndMS:      window.WindowEndMS,
		Action:           ActionHold,
		Confidence:       scored.Confidence,
		LongScore:        scored.LongScore,
		ShortScore:       scored.ShortScore,
		ChopScore:        scored.ChopScore,
		VolatilityScore:  scored.VolatilityScore,
		TrendScore:       scored.TrendScore,
		PullbackScore:    scored.PullbackScore,
		BreakoutScore:    scored.BreakoutScore,
		ExpectedMoveBPS:  scored.ExpectedMoveBPS,
		EstimatedCostBPS: scored.EstimatedCostBPS,
		ReasonCodes:      append([]string(nil), scored.ReasonCodes...),
		DataFreshness:    scored.DataFreshness,
		StrategyName:     s.Name(),
	}

	longBias := scored.LongScore >= scored.ShortScore
	edgeScore := math.Max(scored.LongScore, scored.ShortScore)
	biasSide := SideLong
	if !longBias {
		biasSide = SideShort
	}

	if scored.HardBlock {
		if state.HasPosition {
			decision.Action = ActionExit
			decision.ReasonCodes = appendReason(decision.ReasonCodes, "HARD_BLOCK_EXIT")
		} else {
			decision.Action = ActionNoTradeHardBlock
		}
		return decision
	}

	if state.HasPosition {
		return s.positionDecision(state, decision, biasSide, edgeScore)
	}

	if !s.cfg.LongEnabled && biasSide == SideLong {
		decision.Action = ActionNoTradeHardBlock
		decision.ReasonCodes = appendReason(decision.ReasonCodes, "LONG_DISABLED")
		return decision
	}
	if !s.cfg.ShortEnabled && biasSide == SideShort {
		decision.Action = ActionNoTradeHardBlock
		decision.ReasonCodes = appendReason(decision.ReasonCodes, "SHORT_DISABLED")
		return decision
	}
	if maxChopScore := s.maxChopScoreForSide(biasSide); maxChopScore > 0 && scored.ChopScore > maxChopScore {
		decision.Action = ActionNoTradeHardBlock
		decision.ReasonCodes = appendReason(decision.ReasonCodes, fmt.Sprintf("%dM_CHOP", s.cfg.DecisionWindowMinutes))
		return decision
	}
	if minTrendScore := s.minTrendScoreForSide(biasSide); minTrendScore > 0 && scored.TrendScore < minTrendScore {
		decision.Action = ActionNoTradeHardBlock
		decision.ReasonCodes = appendReason(decision.ReasonCodes, "LOW_TREND_SCORE")
		return decision
	}
	if minExpectedMove := s.minExpectedMoveBPSForSide(biasSide); minExpectedMove > 0 && scored.ExpectedMoveBPS < minExpectedMove {
		decision.Action = ActionNoTradeHardBlock
		decision.ReasonCodes = appendReason(decision.ReasonCodes, "LOW_EXPECTED_MOVE")
		return decision
	}
	if s.cfg.RequireExpectedMoveGtCostMultiple {
		if scored.ExpectedMoveBPS <= scored.EstimatedCostBPS*s.costMultipleRequiredForSide(biasSide) {
			decision.Action = ActionNoTradeHardBlock
			decision.ReasonCodes = appendReason(decision.ReasonCodes, "EXPECTED_MOVE_TOO_LOW_VS_COST")
			return decision
		}
	}
	if s.cfg.DisableScoreBucket55To69 && edgeScore >= 55 && edgeScore <= 69 {
		decision.Action = ActionNoTradeHardBlock
		decision.ReasonCodes = appendReason(decision.ReasonCodes, "SCORE_BUCKET_DISABLED")
		return decision
	}
	if s.cfg.RequireScoreBucket70Plus && edgeScore < 70 {
		decision.Action = ActionNoTradeHardBlock
		decision.ReasonCodes = appendReason(decision.ReasonCodes, "REQUIRES_70_PLUS")
		return decision
	}
	if s.isDisabledScoreBucket70To84(biasSide, edgeScore) {
		decision.Action = ActionNoTradeHardBlock
		decision.ReasonCodes = appendReason(decision.ReasonCodes, "SCORE_BUCKET_70_84_DISABLED")
		return decision
	}
	if minEntryScore := s.minEntryScoreForSide(biasSide); minEntryScore > 0 && edgeScore < minEntryScore {
		decision.Action = ActionNoTradeHardBlock
		decision.ReasonCodes = appendReason(decision.ReasonCodes, "BELOW_MIN_ENTRY_SCORE")
		return decision
	}
	if blockedReason := s.entryQualityBlockReason(window, scored, biasSide); blockedReason != "" {
		decision.Action = ActionNoTradeHardBlock
		decision.ReasonCodes = appendReason(decision.ReasonCodes, blockedReason)
		return decision
	}
	if blockedReason := s.economicsBlockReason(decision, biasSide); blockedReason != "" {
		decision.Action = ActionNoTradeHardBlock
		decision.ReasonCodes = appendReason(decision.ReasonCodes, blockedReason)
		return decision
	}
	if s.cfg.DisableProbeTrades && edgeScore < s.cfg.NormalTradeMinScore {
		decision.Action = ActionNoTradeHardBlock
		decision.ReasonCodes = appendReason(decision.ReasonCodes, "PROBE_DISABLED")
		return decision
	}
	if blockedReason := s.entryRateLimitReason(decision.WindowEndMS); blockedReason != "" {
		decision.Action = ActionNoTradeHardBlock
		decision.ReasonCodes = appendReason(decision.ReasonCodes, blockedReason)
		return decision
	}

	switch {
	case edgeScore >= s.cfg.FullTradeMinScore:
		decision.Action = actionForSide(biasSide, true)
		decision.RiskFraction = s.cfg.FullRiskFraction
		decision.ReasonCodes = appendReason(decision.ReasonCodes, "FULL_SIZE")
	case edgeScore >= s.cfg.NormalTradeMinScore:
		decision.Action = actionForSide(biasSide, true)
		decision.RiskFraction = s.cfg.NormalRiskFraction
		decision.ReasonCodes = appendReason(decision.ReasonCodes, "FULL_SIZE")
	case edgeScore >= s.cfg.ProbeMinScore && s.cfg.AllowProbeTrade:
		decision.Action = actionForSide(biasSide, false)
		decision.RiskFraction = s.cfg.ProbeRiskFraction
		decision.ReasonCodes = appendReason(decision.ReasonCodes, "PROBE_SIZE")
	case edgeScore >= 40 && s.cfg.ForceDecision:
		decision.Action = ActionHold
	default:
		decision.Action = ActionNoTradeHardBlock
		decision.ReasonCodes = appendReason(decision.ReasonCodes, "NO_EDGE")
	}

	return decision
}

func (s *FastAccumulation) positionDecision(state State, decision WindowDecision, biasSide Side, edgeScore float64) WindowDecision {
	heldWindows := 0
	if perWindow := s.aggWindow.CandlesPerWindow(); perWindow > 0 {
		heldWindows = state.HeldCandles / perWindow
	}

	if heldWindows >= s.cfg.MaxHoldWindows {
		decision.Action = ActionExit
		decision.ReasonCodes = appendReason(decision.ReasonCodes, "MAX_HOLD_REACHED")
		return decision
	}
	if heldWindows >= s.cfg.TimeStopWindows && edgeScore < s.cfg.NormalTradeMinScore {
		decision.Action = ActionExit
		decision.ReasonCodes = appendReason(decision.ReasonCodes, "TIME_STOP")
		return decision
	}
	if biasSide != state.PositionSide && edgeScore >= s.cfg.NormalTradeMinScore {
		decision.Action = ActionReverse
		decision.RiskFraction = s.cfg.NormalRiskFraction
		if biasSide == SideLong {
			decision.ReasonCodes = appendReason(decision.ReasonCodes, "REVERSE_TO_LONG")
		} else {
			decision.ReasonCodes = appendReason(decision.ReasonCodes, "REVERSE_TO_SHORT")
		}
		return decision
	}
	if biasSide != state.PositionSide && edgeScore >= s.cfg.ProbeMinScore {
		decision.Action = ActionExit
		decision.ReasonCodes = appendReason(decision.ReasonCodes, "EDGE_FLIP_EXIT")
		return decision
	}
	if biasSide == state.PositionSide && edgeScore >= s.cfg.ProbeMinScore {
		decision.Action = ActionHold
		decision.ReasonCodes = appendReason(decision.ReasonCodes, "WINNER_HOLD")
		return decision
	}

	decision.Action = ActionHold
	return decision
}

func (s *FastAccumulation) signalFromDecision(state State, decision WindowDecision) Signal {
	candlesPerWindow := s.aggWindow.CandlesPerWindow()
	if candlesPerWindow <= 0 {
		candlesPerWindow = 1
	}
	maxHoldCandles := s.cfg.MaxHoldWindows * candlesPerWindow
	if maxHoldCandles <= 0 {
		maxHoldCandles = candlesPerWindow
	}

	edgeScore := math.Max(decision.LongScore, decision.ShortScore)
	stopLoss, takeProfit := s.estimateTradePlan(decision)
	signal := Signal{
		MaxHoldCandles: maxHoldCandles,
		StopLossBPS:    stopLoss,
		TakeProfitBPS:  takeProfit,
		RiskFraction:   decision.RiskFraction,
		ExitPlan:       s.buildExitPlan(candlesPerWindow),
	}

	switch decision.Action {
	case ActionFullLong, ActionProbeLong:
		signal.Side = SideLong
	case ActionFullShort, ActionProbeShort:
		signal.Side = SideShort
	case ActionExit:
		if state.HasPosition {
			signal.ClosePosition = true
		}
	case ActionReverse:
		if decision.ShortScore > decision.LongScore {
			signal.Side = SideShort
		} else {
			signal.Side = SideLong
		}
		signal.ClosePosition = true
		if signal.RiskFraction == 0 {
			signal.RiskFraction = s.cfg.NormalRiskFraction
		}
	case ActionHold, ActionNoTradeHardBlock:
	}

	if edgeScore >= s.cfg.FullTradeMinScore && s.cfg.ForceFullTrade {
		signal.RiskFraction = s.cfg.FullRiskFraction
	}
	return signal
}

func (s *FastAccumulation) estimateTradePlan(decision WindowDecision) (float64, float64) {
	stopLoss := clampFloat(math.Max(decision.EstimatedCostBPS*2.5, decision.ExpectedMoveBPS*0.6), 10, 150)
	takeProfit := clampFloat(math.Max(stopLoss*1.2, decision.ExpectedMoveBPS*1.1), 15, 250)

	switch s.cfg.EntryVariant {
	case EntryVariantPullbackReclaim:
		stopLoss = clampFloat(stopLoss*0.95, 10, 150)
		takeProfit = clampFloat(takeProfit*1.15, 15, 250)
	case EntryVariantBreakoutRetest:
		stopLoss = clampFloat(stopLoss*0.9, 10, 150)
		takeProfit = clampFloat(takeProfit*1.2, 15, 250)
	case EntryVariantMomentumContinuation:
		stopLoss = clampFloat(stopLoss, 12, 150)
		takeProfit = clampFloat(math.Max(takeProfit*1.25, stopLoss*1.5), 20, 280)
	}

	switch s.cfg.ExitModel {
	case ExitModelPartialTPTrail:
		takeProfit = clampFloat(math.Max(takeProfit, stopLoss*1.8), 20, 320)
	case ExitModelBreakevenAfter1R:
		takeProfit = clampFloat(math.Max(takeProfit, stopLoss*1.4), 20, 280)
	case ExitModelTrailAfterMFE:
		takeProfit = clampFloat(math.Max(takeProfit, stopLoss*2.0), 20, 360)
	case ExitModelCutIfNoProgress:
		stopLoss = clampFloat(stopLoss*0.95, 10, 150)
	}

	return stopLoss, takeProfit
}

func (s *FastAccumulation) buildExitPlan(candlesPerWindow int) ExitPlan {
	cutCandles := 0
	if s.cfg.CutNoProgressWindows > 0 && candlesPerWindow > 0 {
		cutCandles = s.cfg.CutNoProgressWindows * candlesPerWindow
	}
	return ExitPlan{
		Model:                     s.cfg.ExitModel,
		PartialTakeProfitR:        s.cfg.PartialTakeProfitR,
		PartialTakeProfitFraction: s.cfg.PartialTakeProfitFraction,
		BreakevenTriggerR:         s.cfg.BreakevenTriggerR,
		TrailAfterMFER:            s.cfg.TrailAfterMFER,
		TrailDistanceR:            s.cfg.TrailDistanceR,
		CutNoProgressR:            s.cfg.CutNoProgressR,
		CutNoProgressCandles:      cutCandles,
	}
}

func actionForSide(side Side, full bool) WindowAction {
	if side == SideShort {
		if full {
			return ActionFullShort
		}
		return ActionProbeShort
	}
	if full {
		return ActionFullLong
	}
	return ActionProbeLong
}

func (s *FastAccumulation) minEntryScoreForSide(side Side) float64 {
	if side == SideShort && s.cfg.ShortMinEntryScore > 0 {
		return s.cfg.ShortMinEntryScore
	}
	if side == SideLong && s.cfg.LongMinEntryScore > 0 {
		return s.cfg.LongMinEntryScore
	}
	return s.cfg.MinEntryScore
}

func (s *FastAccumulation) minTrendScoreForSide(side Side) float64 {
	if side == SideShort && s.cfg.ShortMinTrendScore > 0 {
		return s.cfg.ShortMinTrendScore
	}
	if side == SideLong && s.cfg.LongMinTrendScore > 0 {
		return s.cfg.LongMinTrendScore
	}
	return s.cfg.MinTrendScore
}

func (s *FastAccumulation) maxChopScoreForSide(side Side) float64 {
	if side == SideShort && s.cfg.ShortMaxChopScore > 0 {
		return s.cfg.ShortMaxChopScore
	}
	if side == SideLong && s.cfg.LongMaxChopScore > 0 {
		return s.cfg.LongMaxChopScore
	}
	return s.cfg.MaxChopScore
}

func (s *FastAccumulation) minExpectedMoveBPSForSide(side Side) float64 {
	if side == SideShort && s.cfg.ShortMinExpectedMoveBPS > 0 {
		return s.cfg.ShortMinExpectedMoveBPS
	}
	if side == SideLong && s.cfg.LongMinExpectedMoveBPS > 0 {
		return s.cfg.LongMinExpectedMoveBPS
	}
	return s.cfg.MinExpectedMoveBPS
}

func (s *FastAccumulation) costMultipleRequiredForSide(side Side) float64 {
	if side == SideShort && s.cfg.ShortCostMultipleRequired > 0 {
		return s.cfg.ShortCostMultipleRequired
	}
	if side == SideLong && s.cfg.LongCostMultipleRequired > 0 {
		return s.cfg.LongCostMultipleRequired
	}
	return s.cfg.CostMultipleRequired
}

func (s *FastAccumulation) isDisabledScoreBucket70To84(side Side, edgeScore float64) bool {
	if edgeScore < 70 || edgeScore > 84 {
		return false
	}
	if side == SideShort {
		return s.cfg.DisableShortScoreBucket70To84
	}
	return s.cfg.DisableLongScoreBucket70To84
}

func (s *FastAccumulation) entryQualityBlockReason(window AggregatedWindow, scored ScoreResult, side Side) string {
	switch s.cfg.EntryVariant {
	case EntryVariantPullbackReclaim:
		if !containsReasonCode(scored.ReasonCodes, pullbackReasonForSide(side)) || scored.PullbackScore < 55 {
			return "ENTRY_REQUIRES_PULLBACK_RECLAIM"
		}
		if windowRangeBPS(window) > 0 && bodySizeBPS(window) > windowRangeBPS(window)*0.8 {
			return "ENTRY_OVEREXTENDED"
		}
	case EntryVariantBreakoutRetest:
		if len(s.completedWindows) < 2 || scored.BreakoutScore < 55 {
			return "ENTRY_REQUIRES_BREAKOUT_RETEST"
		}
		prev := s.completedWindows[len(s.completedWindows)-2]
		if !hasBreakoutRetest(s.recentCandles, prev, side) {
			return "ENTRY_REQUIRES_BREAKOUT_RETEST"
		}
	case EntryVariantMomentumContinuation:
		if scored.ExpectedMoveBPS <= scored.EstimatedCostBPS*5 || scored.TrendScore < 65 {
			return "ENTRY_REQUIRES_MOMENTUM_CONTINUATION"
		}
		if len(s.recentCandles) < 3 || !hasMomentumContinuation(s.recentCandles[len(s.recentCandles)-3:], side) {
			return "ENTRY_REQUIRES_MOMENTUM_CONTINUATION"
		}
	}
	return ""
}

func (s *FastAccumulation) economicsBlockReason(decision WindowDecision, _ Side) string {
	stopLoss, takeProfit := s.estimateTradePlan(decision)
	if stopLoss <= 0 {
		return "INVALID_STOP_PLAN"
	}
	targetAfterCost := takeProfit - decision.EstimatedCostBPS
	if s.cfg.MinTargetBPSAfterCost > 0 && targetAfterCost < s.cfg.MinTargetBPSAfterCost {
		return "TARGET_AFTER_COST_TOO_SMALL"
	}
	rewardToRisk := targetAfterCost / stopLoss
	if s.cfg.MinRewardToRisk > 0 && rewardToRisk < s.cfg.MinRewardToRisk {
		return "REWARD_TO_RISK_TOO_SMALL"
	}
	expectedReward := math.Min(decision.ExpectedMoveBPS, takeProfit) - decision.EstimatedCostBPS
	expectedRAfterCost := expectedReward / stopLoss
	if s.cfg.MinExpectedRAfterCost > 0 && expectedRAfterCost < s.cfg.MinExpectedRAfterCost {
		return "EXPECTED_R_AFTER_COST_TOO_SMALL"
	}
	return ""
}

func containsReasonCode(codes []string, want string) bool {
	for _, code := range codes {
		if code == want {
			return true
		}
	}
	return false
}

func pullbackReasonForSide(side Side) string {
	if side == SideShort {
		return "5M_PULLBACK_REJECT"
	}
	return "5M_PULLBACK_RECLAIM"
}

func windowRangeBPS(window AggregatedWindow) float64 {
	if window.Open <= 0 {
		return 0
	}
	return ((window.High - window.Low) / window.Open) * 10000
}

func bodySizeBPS(window AggregatedWindow) float64 {
	if window.Open <= 0 {
		return 0
	}
	return math.Abs(window.Close-window.Open) / window.Open * 10000
}

func hasBreakoutRetest(candles []protocol.Candle, prev AggregatedWindow, side Side) bool {
	if len(candles) < 3 {
		return false
	}
	recent := candles[len(candles)-3:]
	switch side {
	case SideLong:
		level := prev.High
		return recent[0].Close > level && recent[1].Low <= level && recent[2].Close >= level
	case SideShort:
		level := prev.Low
		return recent[0].Close < level && recent[1].High >= level && recent[2].Close <= level
	default:
		return false
	}
}

func hasMomentumContinuation(candles []protocol.Candle, side Side) bool {
	if len(candles) < 3 {
		return false
	}
	for _, candle := range candles {
		switch side {
		case SideLong:
			if candle.Close <= candle.Open {
				return false
			}
		case SideShort:
			if candle.Close >= candle.Open {
				return false
			}
		default:
			return false
		}
	}
	return true
}

func (s *FastAccumulation) scoringCostMultipleRequired() float64 {
	costMultiple := s.cfg.CostMultipleRequired
	if s.cfg.LongCostMultipleRequired > 0 && (costMultiple == 0 || s.cfg.LongCostMultipleRequired < costMultiple) {
		costMultiple = s.cfg.LongCostMultipleRequired
	}
	if s.cfg.ShortCostMultipleRequired > 0 && (costMultiple == 0 || s.cfg.ShortCostMultipleRequired < costMultiple) {
		costMultiple = s.cfg.ShortCostMultipleRequired
	}
	return costMultiple
}

func (s *FastAccumulation) scoringMaxChopScore() float64 {
	maxChopScore := s.cfg.MaxChopScore
	if s.cfg.LongMaxChopScore > maxChopScore {
		maxChopScore = s.cfg.LongMaxChopScore
	}
	if s.cfg.ShortMaxChopScore > maxChopScore {
		maxChopScore = s.cfg.ShortMaxChopScore
	}
	return maxChopScore
}

func (s *FastAccumulation) entryRateLimitReason(windowEndMS int64) string {
	if s.cfg.MaxTradesPerDay > 0 && s.entryCountOnDay(windowEndMS) >= s.cfg.MaxTradesPerDay {
		return "MAX_TRADES_PER_DAY"
	}
	if s.cfg.MinMinutesBetweenEntries > 0 {
		lastEntryMS, ok := s.lastEntryWindowEndMS()
		if ok {
			minSpacing := int64(time.Duration(s.cfg.MinMinutesBetweenEntries) * time.Minute / time.Millisecond)
			if windowEndMS-lastEntryMS < minSpacing {
				return "MIN_ENTRY_SPACING"
			}
		}
	}
	return ""
}

func (s *FastAccumulation) entryCountOnDay(windowEndMS int64) int {
	count := 0
	targetDay := time.UnixMilli(windowEndMS).UTC().Format("2006-01-02")
	for _, decision := range s.decisions {
		if !isEntryAction(decision.Action) {
			continue
		}
		if time.UnixMilli(decision.WindowEndMS).UTC().Format("2006-01-02") == targetDay {
			count++
		}
	}
	return count
}

func (s *FastAccumulation) lastEntryWindowEndMS() (int64, bool) {
	for i := len(s.decisions) - 1; i >= 0; i-- {
		if isEntryAction(s.decisions[i].Action) {
			return s.decisions[i].WindowEndMS, true
		}
	}
	return 0, false
}

func isEntryAction(action WindowAction) bool {
	switch action {
	case ActionFullLong, ActionFullShort, ActionProbeLong, ActionProbeShort, ActionReverse:
		return true
	default:
		return false
	}
}

func validateScoreBound(name string, value float64) error {
	if value < 0 || value > 100 {
		return fmt.Errorf("%s must be between 0 and 100", name)
	}
	return nil
}
