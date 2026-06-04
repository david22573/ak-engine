package strategy

import (
	"context"
	"fmt"
	"math"

	"github.com/davidmiguel22573/ak-engine/pkg/protocol"
)

type FastAccumulationConfig struct {
	ForceDecision        bool    `json:"force_decision"`
	ForceFullTrade       bool    `json:"force_full_trade"`
	AllowProbeTrade      bool    `json:"allow_probe_trade"`
	FullTradeMinScore    float64 `json:"full_trade_min_score"`
	NormalTradeMinScore  float64 `json:"normal_trade_min_score"`
	ProbeMinScore        float64 `json:"probe_min_score"`
	CostMultipleRequired float64 `json:"cost_multiple_required"`
	MaxHoldWindows       int     `json:"max_hold_windows"`
	TimeStopWindows      int     `json:"time_stop_windows"`
	AddToWinnersOnly     bool    `json:"add_to_winners_only"`
	NoMartingale         bool    `json:"no_martingale"`
	ProbeRiskFraction    float64 `json:"probe_risk_fraction"`
	NormalRiskFraction   float64 `json:"normal_risk_fraction"`
	FullRiskFraction     float64 `json:"full_risk_fraction"`
	EstimatedCostBPS     float64 `json:"estimated_cost_bps"`
	MaxChopScore         float64 `json:"max_chop_score"`
}

type FastAccumulation struct {
	cfg           FastAccumulationConfig
	agg15m        *WindowAggregator
	completed15m  []AggregatedWindow
	recentCandles []protocol.Candle
	decisions     []WindowDecision
}

func DefaultFastAccumulationConfig() FastAccumulationConfig {
	return FastAccumulationConfig{
		ForceDecision:        true,
		ForceFullTrade:       false,
		AllowProbeTrade:      true,
		FullTradeMinScore:    85,
		NormalTradeMinScore:  70,
		ProbeMinScore:        55,
		CostMultipleRequired: 3,
		MaxHoldWindows:       4,
		TimeStopWindows:      2,
		AddToWinnersOnly:     true,
		NoMartingale:         true,
		ProbeRiskFraction:    0.25,
		NormalRiskFraction:   0.5,
		FullRiskFraction:     1.0,
		EstimatedCostBPS:     6.0,
		MaxChopScore:         70,
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
	if cfg.ProbeRiskFraction <= 0 || cfg.ProbeRiskFraction > 1 {
		return nil, fmt.Errorf("probe_risk_fraction must be in (0, 1]")
	}
	if cfg.NormalRiskFraction <= 0 || cfg.NormalRiskFraction > 1 {
		return nil, fmt.Errorf("normal_risk_fraction must be in (0, 1]")
	}
	if cfg.FullRiskFraction <= 0 || cfg.FullRiskFraction > 1 {
		return nil, fmt.Errorf("full_risk_fraction must be in (0, 1]")
	}

	agg15m, err := NewWindowAggregator("15m")
	if err != nil {
		return nil, err
	}
	return &FastAccumulation{
		cfg:    cfg,
		agg15m: agg15m,
	}, nil
}

func (s *FastAccumulation) Name() string {
	return "fast_accumulation"
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

	window, err := s.agg15m.Add(candle)
	if err != nil {
		return Signal{}, err
	}
	if window == nil {
		return Signal{}, nil
	}

	s.completed15m = append(s.completed15m, *window)
	hourly, _ := BuildHourlyContext(s.completed15m)
	scored := ScoreWindow(ScoreInput{
		Window:               *window,
		Previous:             s.completed15m[:len(s.completed15m)-1],
		RecentCandles:        s.recentCandles,
		HourContext:          hourly,
		EstimatedCostBPS:     s.cfg.EstimatedCostBPS,
		CostMultipleRequired: s.cfg.CostMultipleRequired,
		MaxChopScore:         s.cfg.MaxChopScore,
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
	if perWindow := s.agg15m.CandlesPerWindow(); perWindow > 0 {
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
	candlesPerWindow := s.agg15m.CandlesPerWindow()
	if candlesPerWindow <= 0 {
		candlesPerWindow = 1
	}
	maxHoldCandles := s.cfg.MaxHoldWindows * candlesPerWindow
	if maxHoldCandles <= 0 {
		maxHoldCandles = candlesPerWindow
	}

	edgeScore := math.Max(decision.LongScore, decision.ShortScore)
	stopLoss := clampFloat(math.Max(decision.EstimatedCostBPS*2.5, decision.ExpectedMoveBPS*0.6), 10, 150)
	takeProfit := clampFloat(math.Max(stopLoss*1.2, decision.ExpectedMoveBPS*1.1), 15, 250)
	signal := Signal{
		MaxHoldCandles: maxHoldCandles,
		StopLossBPS:    stopLoss,
		TakeProfitBPS:  takeProfit,
		RiskFraction:   decision.RiskFraction,
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
