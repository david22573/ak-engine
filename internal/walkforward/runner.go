package walkforward

import (
	"context"
	"fmt"
	"sort"

	"github.com/davidmiguel22573/ak-engine/internal/data"
	"github.com/davidmiguel22573/ak-engine/pkg/protocol"
)

func Run(ctx context.Context, cfg Config, src data.CandleSource, req data.CandleRequest, candles []protocol.Candle) (WalkForwardResult, error) {
	if cfg.TrainWindow <= 0 || cfg.TestWindow <= 0 {
		return WalkForwardResult{}, fmt.Errorf("invalid config: windows must be > 0")
	}
	fromMs := req.From.UnixMilli()
	toMs := req.To.UnixMilli()

	splits := GenerateSplits(fromMs, toMs, cfg.TrainWindow, cfg.TestWindow)

	var splitResults []SplitResult
	var selectedCount int
	var candidateCount int

	for _, split := range splits {
		trainCandles := FilterCandles(candles, split.TrainStartMs, split.TrainEndMs)
		testCandles := FilterCandles(candles, split.TestStartMs, split.TestEndMs)

		cCount, candidates, err := SelectCandidates(ctx, cfg, src, req, trainCandles)
		if err != nil {
			return WalkForwardResult{}, err
		}

		candidateCount += cCount
		if len(candidates) > 0 {
			selectedCount += len(candidates)
		}

		sr := SplitResult{
			SplitIndex:          split.Index,
			TrainStartMs:        split.TrainStartMs,
			TrainEndMs:          split.TrainEndMs,
			TestStartMs:         split.TestStartMs,
			TestEndMs:           split.TestEndMs,
			TrainCandidateCount: cCount,
			SelectedCandidates:  candidates,
			Status:              "PASS",
		}

		if len(candidates) > 0 {
			sr.BestTrainCandidate = candidates[0]
			sr.TestResults = make([]CandidateResult, 0, len(candidates))
			for _, candidate := range candidates {
				testRes, err := EvaluateCandidate(ctx, candidate.Params, src, req, testCandles)
				if err != nil {
					return WalkForwardResult{}, err
				}
				sr.TestResults = append(sr.TestResults, testRes)
			}
			sr.CorrespondingTestResult = sr.TestResults[0]
			sr.BestTestResult = bestCandidate(sr.TestResults)

			sr.TrainToTestPnLDelta = sr.CorrespondingTestResult.NetPnL - sr.BestTrainCandidate.NetPnL
			sr.TrainToTestProfitFactorDelta = sr.CorrespondingTestResult.ProfitFactor - sr.BestTrainCandidate.ProfitFactor
			sr.TrainToTestExpectancyDelta = sr.CorrespondingTestResult.Expectancy - sr.BestTrainCandidate.Expectancy
		} else {
			sr.Status = "NO_CANDIDATE"
		}

		splitResults = append(splitResults, sr)
	}

	aggTrain := computeAggregate(splitResults, true)
	aggTest := computeAggregate(splitResults, false)

	stability := buildCandidateStability(splitResults)

	res := WalkForwardResult{
		Source:                 req.Source,
		Market:                 req.Market,
		Symbol:                 req.Symbol,
		Interval:               req.Interval,
		Strategy:               "fast_accumulation",
		FromMs:                 fromMs,
		ToMs:                   toMs,
		TrainWindow:            cfg.TrainWindow.String(),
		TestWindow:             cfg.TestWindow.String(),
		SplitCount:             len(splits),
		CandidateCount:         candidateCount,
		SelectedCandidateCount: selectedCount,
		AggregateTrain:         aggTrain,
		AggregateTest:          aggTest,
		Splits:                 splitResults,
		CandidateStability:     stability,
		Status:                 "PASS",
	}

	Validate(&res, cfg)
	return res, nil
}

func computeAggregate(splits []SplitResult, train bool) AggregateMetrics {
	var agg AggregateMetrics
	var grossWins, grossLosses float64
	var totalHoldMinutes float64

	agg.Diagnostics = DiagnosticSummary{
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

	for _, sr := range splits {
		if sr.Status != "PASS" || len(sr.SelectedCandidates) == 0 {
			continue
		}
		var c CandidateResult
		if train {
			c = sr.BestTrainCandidate
		} else {
			c = sr.CorrespondingTestResult
			if c.NetPnL > 0 {
				agg.ProfitableSplitCount++
			} else {
				agg.LosingSplitCount++
			}
		}

		agg.TotalTrades += c.TotalTrades
		agg.Wins += c.Wins
		agg.Losses += c.Losses
		agg.NetPnL += c.NetPnL
		agg.FeesPaid += c.FeesPaid
		agg.SlippagePaid += c.SlippagePaid
		totalHoldMinutes += c.AverageHoldMinutes * float64(c.TotalTrades)
		grossWins += c.GrossWins
		grossLosses += c.GrossLosses

		if c.MaxDrawdown > agg.MaxDrawdown {
			agg.MaxDrawdown = c.MaxDrawdown
		}
		if c.MaxConsecutiveLosses > agg.MaxConsecutiveLosses {
			agg.MaxConsecutiveLosses = c.MaxConsecutiveLosses
		}

		for k, v := range c.Diagnostics.PnLByAction {
			agg.Diagnostics.PnLByAction[k] += v
		}
		for k, v := range c.Diagnostics.PnLByScoreBucket {
			agg.Diagnostics.PnLByScoreBucket[k] += v
		}
		for k, v := range c.Diagnostics.WinRateByScoreBucket {
			agg.Diagnostics.WinRateByScoreBucket[k] += v
		} // Note: Needs avg later
		for k, v := range c.Diagnostics.AvgPnLByScoreBucket {
			agg.Diagnostics.AvgPnLByScoreBucket[k] += v
		} // Note: Needs avg later
		for k, v := range c.Diagnostics.PnLByReasonCode {
			agg.Diagnostics.PnLByReasonCode[k] += v
		}
		for k, v := range c.Diagnostics.LossesByReasonCode {
			agg.Diagnostics.LossesByReasonCode[k] += v
		}
		for k, v := range c.Diagnostics.FeesByAction {
			agg.Diagnostics.FeesByAction[k] += v
		}
		for k, v := range c.Diagnostics.SlippageByAction {
			agg.Diagnostics.SlippageByAction[k] += v
		}
		for k, v := range c.Diagnostics.LongVsShortMetrics {
			agg.Diagnostics.LongVsShortMetrics[k] += v
		}
		for k, v := range c.Diagnostics.HardBlocksByReason {
			agg.Diagnostics.HardBlocksByReason[k] += v
		}
	}

	// Average out the rate/avg buckets across splits
	if len(splits) > 0 {
		for k := range agg.Diagnostics.WinRateByScoreBucket {
			agg.Diagnostics.WinRateByScoreBucket[k] /= float64(len(splits))
		}
		for k := range agg.Diagnostics.AvgPnLByScoreBucket {
			agg.Diagnostics.AvgPnLByScoreBucket[k] /= float64(len(splits))
		}
	}

	if agg.TotalTrades > 0 {
		agg.WinRate = float64(agg.Wins) / float64(agg.TotalTrades)
		agg.Expectancy = agg.NetPnL / float64(agg.TotalTrades)
		agg.AverageHoldMinutes = totalHoldMinutes / float64(agg.TotalTrades)
	}

	if grossLosses > 0 {
		agg.ProfitFactor = grossWins / grossLosses
	} else if grossWins > 0 {
		agg.ProfitFactor = grossWins
	}

	return agg
}

func buildCandidateStability(splits []SplitResult) []CandidateStability {
	sigMap := make(map[string]*CandidateStability)
	for _, sr := range splits {
		if sr.Status != "PASS" {
			continue
		}
		for i, trainCandidate := range sr.SelectedCandidates {
			if i >= len(sr.TestResults) {
				break
			}
			testResult := sr.TestResults[i]
			sig := candidateSignature(trainCandidate.Params)
			cs := sigMap[sig]
			if cs == nil {
				cs = &CandidateStability{CandidateSignature: sig}
				sigMap[sig] = cs
			}

			cs.SelectionCount++
			cs.TrainNetPnLTotal += trainCandidate.NetPnL
			cs.TestNetPnLTotal += testResult.NetPnL
			cs.TrainProfitFactorAvg += trainCandidate.ProfitFactor
			cs.TestProfitFactorAvg += testResult.ProfitFactor
			if testResult.NetPnL > 0 {
				cs.TestProfitableCount++
			} else if testResult.TotalTrades > 0 {
				cs.TestLosingCount++
			}
			if testResult.MaxDrawdown > cs.MaxTestDrawdown {
				cs.MaxTestDrawdown = testResult.MaxDrawdown
			}
			if testResult.MaxConsecutiveLosses > cs.MaxTestLossStreak {
				cs.MaxTestLossStreak = testResult.MaxConsecutiveLosses
			}
		}
	}

	stability := make([]CandidateStability, 0, len(sigMap))
	for _, cs := range sigMap {
		cs.TrainProfitFactorAvg /= float64(cs.SelectionCount)
		cs.TestProfitFactorAvg /= float64(cs.SelectionCount)
		stability = append(stability, *cs)
	}

	sort.Slice(stability, func(i, j int) bool {
		if stability[i].SelectionCount != stability[j].SelectionCount {
			return stability[i].SelectionCount > stability[j].SelectionCount
		}
		return stability[i].CandidateSignature < stability[j].CandidateSignature
	})

	return stability
}

func candidateSignature(p Params) string {
	return fmt.Sprintf(
		"SN:%s_EV:%s_XM:%s_FS:%.1f_NS:%.1f_PS:%.1f_CM:%.1f_MH:%d_TS:%d_AP:%t_DP:%t_MCS:%.1f_MES:%.1f_MTS:%.1f_D55:%t_R70:%t_RCM:%t_L:%t_S:%t_LMES:%.1f_SMES:%.1f_LMTS:%.1f_SMTS:%.1f_LMCS:%.1f_SMCS:%.1f_LMM:%.1f_SMM:%.1f_LCM:%.1f_SCM:%.1f_DL7084:%t_DS7084:%t_MTD:%d_MMBE:%d_MER:%.2f_MTB:%.1f_MRR:%.2f_PTR:%.2f_PTF:%.2f_BER:%.2f_TMR:%.2f_TDR:%.2f_CNR:%.2f_CNW:%d",
		p.StrategyName,
		p.EntryVariant,
		p.ExitModel,
		p.FullTradeMinScore,
		p.NormalTradeMinScore,
		p.ProbeMinScore,
		p.CostMultipleRequired,
		p.MaxHoldWindows,
		p.TimeStopWindows,
		p.AllowProbeTrade,
		p.DisableProbeTrades,
		p.MaxChopScore,
		p.MinExpectedMoveBPS,
		p.MinTrendScore,
		p.DisableScoreBucket55To69,
		p.RequireScoreBucket70Plus,
		p.RequireExpectedMoveGtCostMultiple,
		p.LongEnabled,
		p.ShortEnabled,
		p.LongMinEntryScore,
		p.ShortMinEntryScore,
		p.LongMinTrendScore,
		p.ShortMinTrendScore,
		p.LongMaxChopScore,
		p.ShortMaxChopScore,
		p.LongMinExpectedMoveBPS,
		p.ShortMinExpectedMoveBPS,
		p.LongCostMultipleRequired,
		p.ShortCostMultipleRequired,
		p.DisableLongScoreBucket70To84,
		p.DisableShortScoreBucket70To84,
		p.MaxTradesPerDay,
		p.MinMinutesBetweenEntries,
		p.MinExpectedRAfterCost,
		p.MinTargetBPSAfterCost,
		p.MinRewardToRisk,
		p.PartialTakeProfitR,
		p.PartialTakeProfitFraction,
		p.BreakevenTriggerR,
		p.TrailAfterMFER,
		p.TrailDistanceR,
		p.CutNoProgressR,
		p.CutNoProgressWindows,
	)
}

func bestCandidate(results []CandidateResult) CandidateResult {
	best := results[0]
	for _, candidate := range results[1:] {
		if candidate.NetPnL > best.NetPnL {
			best = candidate
			continue
		}
		if candidate.NetPnL == best.NetPnL && candidate.MaxDrawdown < best.MaxDrawdown {
			best = candidate
			continue
		}
		if candidate.NetPnL == best.NetPnL && candidate.MaxDrawdown == best.MaxDrawdown && candidate.MaxConsecutiveLosses < best.MaxConsecutiveLosses {
			best = candidate
		}
	}
	return best
}
