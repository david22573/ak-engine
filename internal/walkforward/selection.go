package walkforward

import (
	"context"
	"sort"

	"github.com/davidmiguel22573/ak-engine/internal/backtest"
	"github.com/davidmiguel22573/ak-engine/internal/data"
	"github.com/davidmiguel22573/ak-engine/internal/strategy"
	"github.com/davidmiguel22573/ak-engine/pkg/protocol"
)

func SelectCandidates(ctx context.Context, cfg Config, src data.CandleSource, req data.CandleRequest, trainCandles []protocol.Candle) (int, []CandidateResult, error) {
	if cfg.FixedParams != nil {
		res, err := EvaluateCandidate(ctx, *cfg.FixedParams, src, req, trainCandles)
		if err != nil {
			return 1, nil, err
		}
		if res.TotalTrades < cfg.MinTrades {
			return 1, nil, nil
		}
		return 1, []CandidateResult{res}, nil
	}

	fullScores := []float64{85, 90}
	normalScores := []float64{70, 75, 80}
	probeScores := []float64{55, 60, 65}
	costMultiples := []float64{3, 4, 5}
	maxHolds := []int{2, 4}
	timeStops := []int{1, 2}
	allowProbes := []bool{true, false}

	var results []CandidateResult
	count := 0

	if cfg.SweepProfile == "strict" {
		maxChopScores := []float64{40, 50, 60}
		minTrendScores := []float64{50, 60, 70}
		req70s := []bool{true, false}
		dis55s := []bool{true, false}
		minMoves := []float64{0, 10, 15, 20}
		longEnables := []bool{true, false}
		shortEnables := []bool{true, false}

		for _, mcs := range maxChopScores {
			for _, mts := range minTrendScores {
				for _, r70 := range req70s {
					for _, d55 := range dis55s {
						for _, mm := range minMoves {
							for _, le := range longEnables {
								for _, se := range shortEnables {
									if !le && !se {
										continue
									}
									count++
									p := Params{
										FullTradeMinScore:                 85,
										NormalTradeMinScore:               70,
										ProbeMinScore:                     55,
										CostMultipleRequired:              4,
										MaxHoldWindows:                    4,
										TimeStopWindows:                   2,
										AllowProbeTrade:                   false,
										DisableProbeTrades:                true,
										MaxChopScore:                      mcs,
										MinTrendScore:                     mts,
										RequireScoreBucket70Plus:          r70,
										DisableScoreBucket55To69:          d55,
										MinExpectedMoveBPS:                mm,
										RequireExpectedMoveGtCostMultiple: true,
										LongEnabled:                       le,
										ShortEnabled:                      se,
									}
									res, err := EvaluateCandidate(ctx, p, src, req, trainCandles)
									if err == nil {
										results = append(results, res)
									}
								}
							}
						}
					}
				}
			}
		}
	} else if cfg.SweepProfile == "calibration" {
		longEnableds := []bool{true, false}
		shortEnableds := []bool{true, false}
		scorePairs := []struct{ long, short float64 }{
			{80, 75},
			{85, 80},
			{85, 85},
			{90, 90},
		}
		costPairs := []struct{ long, short float64 }{
			{4, 3},
			{4, 4},
			{5, 4},
			{6, 5},
		}
		chopPairs := []struct{ long, short float64 }{
			{50, 60},
			{45, 50},
			{40, 45},
			{40, 60},
		}
		movePairs := []struct{ long, short float64 }{
			{10, 10},
			{15, 10},
			{20, 15},
			{25, 20},
		}
		disableLong7084s := []bool{true, false}
		frequencyPairs := []struct{ maxTrades, minMinutes int }{
			{0, 0},
			{4, 0},
			{4, 30},
			{4, 60},
			{8, 30},
		}

		for _, le := range longEnableds {
			for _, se := range shortEnableds {
				if !le && !se {
					continue
				}
				for _, scorePair := range scorePairs {
					for _, costPair := range costPairs {
						for _, chopPair := range chopPairs {
							for _, movePair := range movePairs {
								for _, disableLong7084 := range disableLong7084s {
									for _, frequencyPair := range frequencyPairs {
										count++
										p := Params{
											FullTradeMinScore:                 85,
											NormalTradeMinScore:               70,
											ProbeMinScore:                     55,
											CostMultipleRequired:              4,
											MaxHoldWindows:                    4,
											TimeStopWindows:                   2,
											AllowProbeTrade:                   false,
											DisableProbeTrades:                true,
											MaxChopScore:                      60,
											MinTrendScore:                     60,
											DisableScoreBucket55To69:          true,
											RequireScoreBucket70Plus:          true,
											RequireExpectedMoveGtCostMultiple: true,
											LongEnabled:                       le,
											ShortEnabled:                      se,
											LongMinEntryScore:                 scorePair.long,
											ShortMinEntryScore:                scorePair.short,
											LongCostMultipleRequired:          costPair.long,
											ShortCostMultipleRequired:         costPair.short,
											LongMaxChopScore:                  chopPair.long,
											ShortMaxChopScore:                 chopPair.short,
											LongMinExpectedMoveBPS:            movePair.long,
											ShortMinExpectedMoveBPS:           movePair.short,
											DisableLongScoreBucket70To84:      disableLong7084,
											DisableShortScoreBucket70To84:     false,
											MaxTradesPerDay:                   frequencyPair.maxTrades,
											MinMinutesBetweenEntries:          frequencyPair.minMinutes,
										}
										res, err := EvaluateCandidate(ctx, p, src, req, trainCandles)
										if err == nil {
											results = append(results, res)
										}
									}
								}
							}
						}
					}
				}
			}
		}
	} else {
		for _, fs := range fullScores {
			for _, ns := range normalScores {
				for _, ps := range probeScores {
					if fs < ns || ns < ps {
						continue
					}
					for _, cm := range costMultiples {
						for _, mh := range maxHolds {
							for _, ts := range timeStops {
								for _, ap := range allowProbes {
									count++
									p := Params{
										FullTradeMinScore:    fs,
										NormalTradeMinScore:  ns,
										ProbeMinScore:        ps,
										CostMultipleRequired: cm,
										MaxHoldWindows:       mh,
										TimeStopWindows:      ts,
										AllowProbeTrade:      ap,
									}
									res, err := EvaluateCandidate(ctx, p, src, req, trainCandles)
									if err != nil {
										continue
									}
									results = append(results, res)
								}
							}
						}
					}
				}
			}
		}
	}

	var filtered []CandidateResult
	for _, r := range results {
		if r.TotalTrades >= cfg.MinTrades {
			filtered = append(filtered, r)
		}
	}

	sort.Slice(filtered, func(i, j int) bool {
		if filtered[i].NetPnL != filtered[j].NetPnL {
			return filtered[i].NetPnL > filtered[j].NetPnL
		}
		if filtered[i].MaxDrawdown != filtered[j].MaxDrawdown {
			return filtered[i].MaxDrawdown < filtered[j].MaxDrawdown
		}
		return filtered[i].MaxConsecutiveLosses < filtered[j].MaxConsecutiveLosses
	})

	if len(filtered) > cfg.TopCandidates {
		filtered = filtered[:cfg.TopCandidates]
	}

	return count, filtered, nil
}

func EvaluateCandidate(ctx context.Context, p Params, src data.CandleSource, req data.CandleRequest, candles []protocol.Candle) (CandidateResult, error) {
	stratCfg := strategy.DefaultFastAccumulationConfig()
	stratCfg.StrategyName = p.StrategyName
	stratCfg.DecisionWindowMinutes = p.DecisionWindowMinutes
	stratCfg.EntryVariant = strategy.EntryVariant(p.EntryVariant)
	stratCfg.ExitModel = strategy.ExitModel(p.ExitModel)
	stratCfg.FullTradeMinScore = p.FullTradeMinScore
	stratCfg.NormalTradeMinScore = p.NormalTradeMinScore
	stratCfg.ProbeMinScore = p.ProbeMinScore
	stratCfg.CostMultipleRequired = p.CostMultipleRequired
	stratCfg.MaxHoldWindows = p.MaxHoldWindows
	stratCfg.TimeStopWindows = p.TimeStopWindows
	stratCfg.AllowProbeTrade = p.AllowProbeTrade
	stratCfg.DisableProbeTrades = p.DisableProbeTrades
	if p.MaxChopScore > 0 {
		stratCfg.MaxChopScore = p.MaxChopScore
	}
	stratCfg.MinEntryScore = p.MinEntryScore
	stratCfg.MinTrendScore = p.MinTrendScore
	stratCfg.DisableScoreBucket55To69 = p.DisableScoreBucket55To69
	stratCfg.RequireScoreBucket70Plus = p.RequireScoreBucket70Plus
	stratCfg.MinExpectedMoveBPS = p.MinExpectedMoveBPS
	stratCfg.RequireExpectedMoveGtCostMultiple = p.RequireExpectedMoveGtCostMultiple
	stratCfg.LongEnabled = p.LongEnabled
	stratCfg.ShortEnabled = p.ShortEnabled
	stratCfg.LongMinEntryScore = p.LongMinEntryScore
	stratCfg.ShortMinEntryScore = p.ShortMinEntryScore
	stratCfg.LongMinTrendScore = p.LongMinTrendScore
	stratCfg.ShortMinTrendScore = p.ShortMinTrendScore
	stratCfg.LongMaxChopScore = p.LongMaxChopScore
	stratCfg.ShortMaxChopScore = p.ShortMaxChopScore
	stratCfg.LongMinExpectedMoveBPS = p.LongMinExpectedMoveBPS
	stratCfg.ShortMinExpectedMoveBPS = p.ShortMinExpectedMoveBPS
	stratCfg.LongCostMultipleRequired = p.LongCostMultipleRequired
	stratCfg.ShortCostMultipleRequired = p.ShortCostMultipleRequired
	stratCfg.DisableLongScoreBucket70To84 = p.DisableLongScoreBucket70To84
	stratCfg.DisableShortScoreBucket70To84 = p.DisableShortScoreBucket70To84
	stratCfg.MaxTradesPerDay = p.MaxTradesPerDay
	stratCfg.MinMinutesBetweenEntries = p.MinMinutesBetweenEntries
	stratCfg.MinExpectedRAfterCost = p.MinExpectedRAfterCost
	stratCfg.MinTargetBPSAfterCost = p.MinTargetBPSAfterCost
	stratCfg.MinRewardToRisk = p.MinRewardToRisk
	stratCfg.PartialTakeProfitR = p.PartialTakeProfitR
	stratCfg.PartialTakeProfitFraction = p.PartialTakeProfitFraction
	stratCfg.BreakevenTriggerR = p.BreakevenTriggerR
	stratCfg.TrailAfterMFER = p.TrailAfterMFER
	stratCfg.TrailDistanceR = p.TrailDistanceR
	stratCfg.CutNoProgressR = p.CutNoProgressR
	stratCfg.CutNoProgressWindows = p.CutNoProgressWindows
	stratCfg.EstimatedCostBPS = 6.0

	strat, err := strategy.NewFastAccumulation(stratCfg)
	if err != nil {
		return CandidateResult{}, err
	}

	engine, err := backtest.NewEngine(src, strat, backtest.Config{
		StartingCash:    10000,
		MaxPositionSize: 1,
		SlippageBPS:     1,
		Fees: backtest.FeeConfig{
			MakerFeeBPS: 0,
			TakerFeeBPS: 5,
		},
	})
	if err != nil {
		return CandidateResult{}, err
	}

	report, err := engine.RunCandles(ctx, req, candles)
	if err != nil {
		return CandidateResult{}, err
	}

	var grossWins, grossLosses float64
	for _, t := range report.Trades {
		if t.NetPnL > 0 {
			grossWins += t.NetPnL
		} else if t.NetPnL < 0 {
			grossLosses -= t.NetPnL
		}
	}

	diag := DiagnosticSummary{
		PnLByAction:          make(map[string]float64),
		PnLByScoreBucket:     make(map[string]float64),
		WinRateByScoreBucket: make(map[string]float64),
		AvgPnLByScoreBucket:  make(map[string]float64),
		PnLByReasonCode:      make(map[string]float64),
		LossesByReasonCode:   make(map[string]int),
		FeesByAction:         make(map[string]float64),
		SlippageByAction:     make(map[string]float64),
		LongVsShortMetrics:   make(map[string]float64),
		HardBlocksByReason:   make(map[string]int),
	}

	if report.FastAccumulation != nil {
		for k, v := range report.FastAccumulation.PnLByAction {
			diag.PnLByAction[k] = v
		}
		for k, v := range report.FastAccumulation.PnLByScoreBucket {
			diag.PnLByScoreBucket[k] = v
		}
		for k, v := range report.FastAccumulation.WinRateByScoreBucket {
			diag.WinRateByScoreBucket[k] = v
		}
		for k, v := range report.FastAccumulation.AvgPnLByScoreBucket {
			diag.AvgPnLByScoreBucket[k] = v
		}
		for k, v := range report.FastAccumulation.LossesByReasonCode {
			diag.LossesByReasonCode[k] = v
		}
		for k, v := range report.FastAccumulation.FeesByAction {
			diag.FeesByAction[k] = v
		}
		for k, v := range report.FastAccumulation.SlippageByAction {
			diag.SlippageByAction[k] = v
		}
		for k, v := range report.FastAccumulation.LongVsShortMetrics {
			diag.LongVsShortMetrics[k] = v
		}
		for k, v := range report.FastAccumulation.HardBlocksByReason {
			diag.HardBlocksByReason[k] = v
		}
	}

	var longPnL, shortPnL float64
	var longTrades, shortTrades int
	for _, t := range report.Trades {
		if t.Side == strategy.SideLong {
			longPnL += t.NetPnL
			longTrades++
		} else if t.Side == strategy.SideShort {
			shortPnL += t.NetPnL
			shortTrades++
		}
		if t.ExitReason != "" {
			diag.PnLByReasonCode[string(t.ExitReason)] += t.NetPnL
		}
	}
	if len(diag.LongVsShortMetrics) == 0 {
		diag.LongVsShortMetrics["long_pnl"] = longPnL
		diag.LongVsShortMetrics["short_pnl"] = shortPnL
		diag.LongVsShortMetrics["long_trades"] = float64(longTrades)
		diag.LongVsShortMetrics["short_trades"] = float64(shortTrades)
	}

	return CandidateResult{
		Params:               p,
		TotalTrades:          report.TotalTrades,
		Wins:                 report.Wins,
		Losses:               report.Losses,
		WinRate:              report.WinRate,
		NetPnL:               report.NetPnL,
		FeesPaid:             report.FeesPaid,
		SlippagePaid:         report.SlippagePaid,
		ProfitFactor:         report.ProfitFactor,
		MaxDrawdown:          report.MaxDrawdown,
		MaxConsecutiveLosses: report.MaxConsecutiveLosses,
		Expectancy:           report.Expectancy,
		AverageHoldMinutes:   report.AverageHoldMinutes,
		EndingCash:           report.EndingCash,
		Status:               "PASS",
		Diagnostics:          diag,
		GrossWins:            grossWins,
		GrossLosses:          grossLosses,
	}, nil
}
