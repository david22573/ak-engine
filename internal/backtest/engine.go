package backtest

import (
	"context"
	"fmt"
	"math"
	"time"

	"github.com/davidmiguel22573/ak-engine/internal/data"
	"github.com/davidmiguel22573/ak-engine/internal/strategy"
	"github.com/davidmiguel22573/ak-engine/pkg/protocol"
)

type Config struct {
	StartingCash     float64
	MaxPositionSize  float64
	SlippageBPS      float64
	Fees             FeeConfig
	IncludeDecisions bool
}

type Engine struct {
	source   data.CandleSource
	strategy strategy.Strategy
	cfg      Config
}

func NewEngine(source data.CandleSource, strat strategy.Strategy, cfg Config) (*Engine, error) {
	if source == nil {
		return nil, fmt.Errorf("source is required")
	}
	if strat == nil {
		return nil, fmt.Errorf("strategy is required")
	}
	if cfg.StartingCash <= 0 {
		return nil, fmt.Errorf("starting_cash must be > 0")
	}
	if cfg.MaxPositionSize <= 0 || cfg.MaxPositionSize > 1 {
		return nil, fmt.Errorf("max_position_size must be in (0, 1]")
	}
	if cfg.SlippageBPS < 0 {
		return nil, fmt.Errorf("slippage_bps must be >= 0")
	}
	return &Engine{source: source, strategy: strat, cfg: cfg}, nil
}

func (e *Engine) Run(ctx context.Context, req data.CandleRequest) (Report, error) {
	candles, err := e.source.LoadCandles(ctx, req)
	if err != nil {
		return Report{}, fmt.Errorf("load candles: %w", err)
	}
	if err := data.ValidateCandles(req.Interval, candles); err != nil {
		return Report{}, fmt.Errorf("validate candles: %w", err)
	}
	return e.runCandles(ctx, req, candles)
}

func (e *Engine) RunCandles(ctx context.Context, req data.CandleRequest, candles []protocol.Candle) (Report, error) {
	return e.runCandles(ctx, req, candles)
}

func (e *Engine) runCandles(ctx context.Context, req data.CandleRequest, candles []protocol.Candle) (Report, error) {
	equity := e.cfg.StartingCash
	equityCurve := []float64{equity}
	var trades []Trade
	var open *Position
	history := make([]protocol.Candle, 0, len(candles))
	var decisions []strategy.WindowDecision

	for idx, candle := range candles {
		if open != nil {
			open.HeldCandles++

			exitBasePrice, reason, hit, err := ResolveExitPrice(*open, candle)
			if err != nil {
				return Report{}, err
			}
			if hit {
				trade, nextEquity, err := e.closePosition(candles, *open, candle, idx, exitBasePrice, reason, equity)
				if err != nil {
					return Report{}, err
				}
				trades = append(trades, trade)
				equity = nextEquity
				equityCurve = append(equityCurve, equity)
				open = nil
			} else if open.HeldCandles >= open.MaxHoldCandles {
				trade, nextEquity, err := e.closePosition(candles, *open, candle, idx, candle.Close, ExitReasonTimeStop, equity)
				if err != nil {
					return Report{}, err
				}
				trades = append(trades, trade)
				equity = nextEquity
				equityCurve = append(equityCurve, equity)
				open = nil
			}
		}

		strategyState := strategy.State{Candles: history}
		if open != nil {
			strategyState.HasPosition = true
			strategyState.PositionSide = open.Side
			strategyState.PositionEntryPrice = open.EntryPrice
			strategyState.HeldCandles = open.HeldCandles
		}
		signal, err := e.strategy.OnCandle(ctx, strategyState, candle)
		if err != nil {
			return Report{}, fmt.Errorf("strategy on candle %d: %w", idx, err)
		}
		if signal.Decision != nil {
			decisions = append(decisions, *signal.Decision)
		}
		if open != nil && signal.ClosePosition {
			exitReason := ExitReasonStrategy
			if signal.Decision != nil && signal.Decision.Action == strategy.ActionReverse {
				exitReason = ExitReasonReverse
			}
			trade, nextEquity, err := e.closePosition(candles, *open, candle, idx, candle.Close, exitReason, equity)
			if err != nil {
				return Report{}, err
			}
			trades = append(trades, trade)
			equity = nextEquity
			equityCurve = append(equityCurve, equity)
			open = nil
		}
		if open == nil && signal.Side != strategy.SideNone {
			pos, err := e.openPosition(candle, idx, signal, equity)
			if err != nil {
				return Report{}, err
			}
			if pos != nil {
				open = pos
			}
		}

		history = append(history, candle)
	}

	if open != nil {
		finalCandle := candles[len(candles)-1]
		trade, nextEquity, err := e.closePosition(candles, *open, finalCandle, len(candles)-1, finalCandle.Close, ExitReasonEndOfData, equity)
		if err != nil {
			return Report{}, err
		}
		trades = append(trades, trade)
		equity = nextEquity
		equityCurve = append(equityCurve, equity)
		open = nil
	}

	metrics := ComputeMetrics(e.cfg.StartingCash, equity, trades, equityCurve, boolToInt(open != nil))
	decisionSummary := summarizeDecisions(decisions)

	// Initialize buckets maps
	tradesByAction := make(map[string]int)
	pnlByAction := make(map[string]float64)
	tradesByScoreBucket := make(map[string]int)
	pnlByScoreBucket := make(map[string]float64)
	winRateByScoreBucket := make(map[string]float64)
	avgPnLByScoreBucket := make(map[string]float64)
	hardBlocksByReason := make(map[string]int)
	lossesByReasonCode := make(map[string]int)
	feesByAction := make(map[string]float64)
	slippageByAction := make(map[string]float64)

	// Pre-populate score buckets
	scoreBuckets := []string{"0-39", "40-54", "55-69", "70-84", "85-100"}
	for _, b := range scoreBuckets {
		tradesByScoreBucket[b] = 0
		pnlByScoreBucket[b] = 0.0
		winRateByScoreBucket[b] = 0.0
		avgPnLByScoreBucket[b] = 0.0
	}

	// Helper function for checking if string slice contains a string
	containsStr := func(slice []string, s string) bool {
		for _, x := range slice {
			if x == s {
				return true
			}
		}
		return false
	}

	// Populate hard blocks by reason from decisions
	for _, d := range decisions {
		isHardBlock := d.Action == strategy.ActionNoTradeHardBlock || containsStr(d.ReasonCodes, "HARD_BLOCK_EXIT")
		if isHardBlock {
			for _, rc := range d.ReasonCodes {
				if rc == "EXPECTED_MOVE_BELOW_COST" || rc == "15M_CHOP" || rc == "INVALID_COST" || rc == "INSUFFICIENT_DATA" {
					hardBlocksByReason[rc]++
				}
			}
		}
	}

	// Track wins per bucket for win rate
	winsByScoreBucket := make(map[string]int)

	for _, t := range trades {
		act := t.EntryAction
		if act == "" {
			act = "UNKNOWN"
		}
		tradesByAction[act]++
		pnlByAction[act] += t.NetPnL
		feesByAction[act] += (t.EntryFee + t.ExitFee)
		slippageByAction[act] += t.SlippagePaid

		// Score bucket classification
		score := t.ScoreAtEntry
		var bucket string
		switch {
		case score >= 0 && score <= 39:
			bucket = "0-39"
		case score >= 40 && score <= 54:
			bucket = "40-54"
		case score >= 55 && score <= 69:
			bucket = "55-69"
		case score >= 70 && score <= 84:
			bucket = "70-84"
		case score >= 85 && score <= 100:
			bucket = "85-100"
		default:
			bucket = "unknown"
		}

		if bucket != "unknown" {
			tradesByScoreBucket[bucket]++
			pnlByScoreBucket[bucket] += t.NetPnL
			if t.NetPnL > 0 {
				winsByScoreBucket[bucket]++
			}
		}

		if t.NetPnL < 0 {
			for _, rc := range t.EntryReasonCodes {
				lossesByReasonCode[rc]++
			}
		}
	}

	for _, b := range scoreBuckets {
		count := tradesByScoreBucket[b]
		if count > 0 {
			winRateByScoreBucket[b] = float64(winsByScoreBucket[b]) / float64(count)
			avgPnLByScoreBucket[b] = pnlByScoreBucket[b] / float64(count)
		}
	}

	report := Report{
		Source:               e.source.Name(),
		Market:               req.Market,
		Symbol:               req.Symbol,
		Interval:             req.Interval,
		Strategy:             e.strategy.Name(),
		FromMS:               req.From.UnixMilli(),
		ToMS:                 req.To.UnixMilli(),
		TotalCandles:         len(candles),
		TotalTrades:          metrics.TradeCount,
		Wins:                 metrics.WinCount,
		Losses:               metrics.LossCount,
		WinRate:              metrics.WinRate,
		GrossPnL:             metrics.GrossPnL,
		NetPnL:               metrics.NetPnL,
		FeesPaid:             metrics.TotalFees,
		SlippagePaid:         metrics.SlippagePaid,
		ProfitFactor:         metrics.ProfitFactor,
		MaxDrawdown:          metrics.MaxDrawdown,
		MaxConsecutiveLosses: metrics.MaxConsecutiveLosses,
		AverageWin:           metrics.AverageWin,
		AverageLoss:          metrics.AverageLoss,
		Expectancy:           metrics.Expectancy,
		AverageHoldMinutes:   metrics.AverageHoldMinutes,
		Status:               "PASS",
		StartingCash:         e.cfg.StartingCash,
		EndingCash:           equity,
		MaxPosition:          e.cfg.MaxPositionSize,
		SlippageBPS:          e.cfg.SlippageBPS,
		MakerFeeBPS:          e.cfg.Fees.MakerFeeBPS,
		TakerFeeBPS:          e.cfg.Fees.TakerFeeBPS,
		WindowDecisionCount:  decisionSummary.WindowDecisionCount,
		FullTradeCount:       decisionSummary.FullTradeCount,
		ProbeTradeCount:      decisionSummary.ProbeTradeCount,
		HardBlockCount:       decisionSummary.HardBlockCount,
		HoldCount:            decisionSummary.HoldCount,
		ExitCount:            decisionSummary.ExitCount,
		ReverseCount:         decisionSummary.ReverseCount,
		Trades:               trades,
		Metrics:              metrics,
		GeneratedAtMS:        time.Now().UnixMilli(),
	}

	if e.cfg.IncludeDecisions {
		report.Decisions = decisions
	}

	if cfgStrategy, ok := e.strategy.(interface {
		ConfigSnapshot() strategy.FastAccumulationConfig
	}); ok {
		report.FastAccumulation = &FastAccumulationReport{
			DecisionSummary:      decisionSummary,
			Config:               cfgStrategy.ConfigSnapshot(),
			TradesByAction:       tradesByAction,
			PnLByAction:          pnlByAction,
			TradesByScoreBucket:  tradesByScoreBucket,
			PnLByScoreBucket:     pnlByScoreBucket,
			WinRateByScoreBucket: winRateByScoreBucket,
			AvgPnLByScoreBucket:  avgPnLByScoreBucket,
			HardBlocksByReason:   hardBlocksByReason,
			LossesByReasonCode:   lossesByReasonCode,
			FeesByAction:         feesByAction,
			SlippageByAction:     slippageByAction,
		}
	}
	return report, nil
}

func (e *Engine) openPosition(candle protocol.Candle, idx int, signal strategy.Signal, equity float64) (*Position, error) {
	if signal.MaxHoldCandles <= 0 {
		return nil, fmt.Errorf("signal max_hold_candles must be > 0")
	}
	baseEntryPrice := candle.Close
	entryPrice, err := ApplySlippage(baseEntryPrice, signal.Side, FillActionEntry, e.cfg.SlippageBPS)
	if err != nil {
		return nil, err
	}

	riskFraction := signal.RiskFraction
	if riskFraction == 0 {
		riskFraction = 1
	}
	if riskFraction < 0 || riskFraction > 1 {
		return nil, fmt.Errorf("signal risk_fraction must be in [0, 1]")
	}
	notional := equity * e.cfg.MaxPositionSize * riskFraction
	if notional <= 0 {
		return nil, nil
	}
	quantity := notional / entryPrice
	entryFee, err := CalculateFee(notional, e.cfg.Fees)
	if err != nil {
		return nil, err
	}

	pos := &Position{
		Symbol:           candle.Symbol,
		Market:           candle.Market,
		Interval:         candle.Interval,
		Side:             signal.Side,
		EntryTimeMS:      candle.CloseTimeMS,
		BaseEntryPrice:   baseEntryPrice,
		EntryPrice:       entryPrice,
		Quantity:         quantity,
		Notional:         notional,
		MaxHoldCandles:   signal.MaxHoldCandles,
		EntryFee:         entryFee,
		Strategy:         e.strategy.Name(),
		EntryCandleIndex: idx,
	}

	if signal.Decision != nil {
		pos.EntryWindowMS = signal.Decision.WindowStartMS
		pos.EntryReasonCodes = append([]string(nil), signal.Decision.ReasonCodes...)
		pos.ScoreAtEntry = math.Max(signal.Decision.LongScore, signal.Decision.ShortScore)
		pos.RiskFraction = signal.Decision.RiskFraction
		pos.EstimatedCostBPS = signal.Decision.EstimatedCostBPS
		pos.ExpectedMoveBPS = signal.Decision.ExpectedMoveBPS
		pos.EntryAction = string(signal.Decision.Action)
	} else {
		pos.RiskFraction = 1.0
	}

	switch signal.Side {
	case strategy.SideLong:
		pos.StopPrice = entryPrice * (1 - signal.StopLossBPS/10000)
		pos.TargetPrice = entryPrice * (1 + signal.TakeProfitBPS/10000)
	case strategy.SideShort:
		pos.StopPrice = entryPrice * (1 + signal.StopLossBPS/10000)
		pos.TargetPrice = entryPrice * (1 - signal.TakeProfitBPS/10000)
	default:
		return nil, fmt.Errorf("unsupported signal side %q", signal.Side)
	}
	return pos, nil
}

func summarizeDecisions(decisions []strategy.WindowDecision) FastAccumulationSummary {
	summary := FastAccumulationSummary{WindowDecisionCount: len(decisions)}
	for _, decision := range decisions {
		switch decision.Action {
		case strategy.ActionFullLong, strategy.ActionFullShort:
			summary.FullTradeCount++
		case strategy.ActionProbeLong, strategy.ActionProbeShort:
			summary.ProbeTradeCount++
		case strategy.ActionNoTradeHardBlock:
			summary.HardBlockCount++
		case strategy.ActionHold:
			summary.HoldCount++
		case strategy.ActionExit:
			summary.ExitCount++
		case strategy.ActionReverse:
			summary.ReverseCount++
		}
	}
	return summary
}

func (e *Engine) closePosition(candles []protocol.Candle, pos Position, candle protocol.Candle, idx int, baseExitPrice float64, reason ExitReason, equity float64) (Trade, float64, error) {
	exitPrice, err := ApplySlippage(baseExitPrice, pos.Side, FillActionExit, e.cfg.SlippageBPS)
	if err != nil {
		return Trade{}, 0, err
	}
	exitNotional := exitPrice * pos.Quantity
	exitFee, err := CalculateFee(exitNotional, e.cfg.Fees)
	if err != nil {
		return Trade{}, 0, err
	}

	grossPnL := (exitPrice - pos.EntryPrice) * pos.Quantity
	if pos.Side == strategy.SideShort {
		grossPnL = (pos.EntryPrice - exitPrice) * pos.Quantity
	}
	slippagePaid := calculateSlippagePaid(pos.Side, pos.Quantity, pos.BaseEntryPrice, pos.EntryPrice, baseExitPrice, exitPrice)
	netPnL := grossPnL - pos.EntryFee - exitFee
	nextEquity := equity + netPnL

	// MAE and MFE calculation
	minLow := pos.EntryPrice
	maxHigh := pos.EntryPrice
	if pos.EntryCandleIndex >= 0 && pos.EntryCandleIndex < len(candles) && idx >= pos.EntryCandleIndex && idx < len(candles) {
		for i := pos.EntryCandleIndex; i <= idx; i++ {
			c := candles[i]
			if i == pos.EntryCandleIndex {
				minLow = c.Low
				maxHigh = c.High
			} else {
				if c.Low < minLow {
					minLow = c.Low
				}
				if c.High > maxHigh {
					maxHigh = c.High
				}
			}
		}
	}

	var maeBPS, mfeBPS float64
	if pos.Side == strategy.SideLong {
		maeBPS = ((pos.EntryPrice - minLow) / pos.EntryPrice) * 10000
		mfeBPS = ((maxHigh - pos.EntryPrice) / pos.EntryPrice) * 10000
	} else if pos.Side == strategy.SideShort {
		maeBPS = ((maxHigh - pos.EntryPrice) / pos.EntryPrice) * 10000
		mfeBPS = ((pos.EntryPrice - minLow) / pos.EntryPrice) * 10000
	}
	if maeBPS < 0 {
		maeBPS = 0
	}
	if mfeBPS < 0 {
		mfeBPS = 0
	}

	// Hold Windows calculation
	candlesPerWindow := 1
	if pos.Interval != "" {
		inputMS, err := data.ParseIntervalToMS(pos.Interval)
		if err == nil && inputMS > 0 {
			candlesPerWindow = int((15 * 60 * 1000) / inputMS)
		}
	}
	if candlesPerWindow <= 0 {
		candlesPerWindow = 1
	}
	holdWindows := pos.HeldCandles / candlesPerWindow

	// RMultiple calculation
	plannedRisk := math.Abs(pos.EntryPrice - pos.StopPrice)
	rMultiple := 0.0
	if plannedRisk > 0 {
		if pos.Side == strategy.SideLong {
			rMultiple = (exitPrice - pos.EntryPrice) / plannedRisk
		} else if pos.Side == strategy.SideShort {
			rMultiple = (pos.EntryPrice - exitPrice) / plannedRisk
		}
	}

	trade := Trade{
		Symbol:           pos.Symbol,
		Market:           pos.Market,
		Interval:         pos.Interval,
		Side:             pos.Side,
		EntryTimeMS:      pos.EntryTimeMS,
		ExitTimeMS:       candle.CloseTimeMS,
		EntryPrice:       pos.EntryPrice,
		ExitPrice:        exitPrice,
		Quantity:         pos.Quantity,
		Notional:         pos.Notional,
		StopPrice:        pos.StopPrice,
		TargetPrice:      pos.TargetPrice,
		MaxHoldCandles:   pos.MaxHoldCandles,
		HeldCandles:      pos.HeldCandles,
		EntryFee:         pos.EntryFee,
		ExitFee:          exitFee,
		SlippagePaid:     slippagePaid,
		GrossPnL:         grossPnL,
		NetPnL:           netPnL,
		NetReturnBPS:     (netPnL / pos.Notional) * 10000,
		ExitReason:       reason,
		Strategy:         pos.Strategy,
		EntryCandleIndex: pos.EntryCandleIndex,
		ExitCandleIndex:  idx,
		EntryWindowMS:    pos.EntryWindowMS,
		ExitWindowMS:     alignWindowStart(candle.OpenTimeMS, 15*60*1000),
		EntryReasonCodes: pos.EntryReasonCodes,
		ScoreAtEntry:     pos.ScoreAtEntry,
		RiskFraction:     pos.RiskFraction,
		EstimatedCostBPS: pos.EstimatedCostBPS,
		ExpectedMoveBPS:  pos.ExpectedMoveBPS,
		RMultiple:        rMultiple,
		MAEBPS:           maeBPS,
		MFEBPS:           mfeBPS,
		HoldWindows:      holdWindows,
		EntryAction:      pos.EntryAction,
	}
	return trade, nextEquity, nil
}

func alignWindowStart(openTimeMS, targetMS int64) int64 {
	return openTimeMS - (openTimeMS % targetMS)
}

func calculateSlippagePaid(side strategy.Side, qty, baseEntryPrice, entryPrice, baseExitPrice, exitPrice float64) float64 {
	entryPaid := 0.0
	exitPaid := 0.0
	switch side {
	case strategy.SideLong:
		entryPaid = (entryPrice - baseEntryPrice) * qty
		exitPaid = (baseExitPrice - exitPrice) * qty
	case strategy.SideShort:
		entryPaid = (baseEntryPrice - entryPrice) * qty
		exitPaid = (exitPrice - baseExitPrice) * qty
	}
	return entryPaid + exitPaid
}

func boolToInt(v bool) int {
	if v {
		return 1
	}
	return 0
}
