package app

import (
	"fmt"

	"github.com/davidmiguel22573/ak-engine/internal/strategy"
	"github.com/davidmiguel22573/ak-engine/internal/walkforward"
)

const (
	strategyFastAccumulation                  = "fast_accumulation"
	strategyFastAccumulationStrict            = "fast_accumulation_strict"
	strategyFastAccumulationStrictShortBias   = "fast_accumulation_strict_short_bias"
	strategyFastAccumulationStrictHighConf    = "fast_accumulation_strict_high_confidence"
	strategyFastAccumulationStrictLowFreq     = "fast_accumulation_strict_low_frequency"
	strategyFastAccumulationStrictCostGuard   = "fast_accumulation_strict_cost_guard"
	strategyFastAccumulationStrictNo7084Longs = "fast_accumulation_strict_no_70_84_longs"
	strategyFastAccumulationStrict30m         = "fast_accumulation_strict_30m"
	strategyFastAccumulationStrict1h          = "fast_accumulation_strict_1h"
	strategyFastAccumulationPullbackReclaim   = "fast_accumulation_pullback_reclaim"
	strategyFastAccumulationBreakoutRetest    = "fast_accumulation_breakout_retest"
	strategyFastAccumulationMomentumCont      = "fast_accumulation_momentum_continuation"
	strategyFastAccumulationPartialTrail      = "fast_accumulation_partial_trail"
	strategyFastAccumulationBreakevenGuard    = "fast_accumulation_breakeven_guard"
	strategyFastAccumulationCutNoProgress     = "fast_accumulation_cut_no_progress"
	strategyFastAccumulationEconomicsGuard    = "fast_accumulation_economics_guard"
)

func isFastAccumulationStrategyName(name string) bool {
	switch name {
	case strategyFastAccumulation,
		strategyFastAccumulationStrict,
		strategyFastAccumulationStrictShortBias,
		strategyFastAccumulationStrictHighConf,
		strategyFastAccumulationStrictLowFreq,
		strategyFastAccumulationStrictCostGuard,
		strategyFastAccumulationStrictNo7084Longs,
		strategyFastAccumulationStrict30m,
		strategyFastAccumulationStrict1h,
		strategyFastAccumulationPullbackReclaim,
		strategyFastAccumulationBreakoutRetest,
		strategyFastAccumulationMomentumCont,
		strategyFastAccumulationPartialTrail,
		strategyFastAccumulationBreakevenGuard,
		strategyFastAccumulationCutNoProgress,
		strategyFastAccumulationEconomicsGuard:
		return true
	default:
		return false
	}
}

func strictFastAccumulationConfig(estimatedCostBPS float64) strategy.FastAccumulationConfig {
	cfg := strategy.DefaultFastAccumulationConfig()
	cfg.StrategyName = strategyFastAccumulationStrict
	cfg.AllowProbeTrade = false
	cfg.DisableProbeTrades = true
	cfg.RequireScoreBucket70Plus = true
	cfg.DisableScoreBucket55To69 = true
	cfg.CostMultipleRequired = 4
	cfg.RequireExpectedMoveGtCostMultiple = true
	cfg.MaxHoldWindows = 4
	cfg.TimeStopWindows = 2
	cfg.MaxChopScore = 50
	cfg.MinTrendScore = 60
	cfg.EstimatedCostBPS = estimatedCostBPS
	return cfg
}

func fastAccumulationPresetConfig(name string, estimatedCostBPS float64) (strategy.FastAccumulationConfig, error) {
	switch name {
	case strategyFastAccumulationStrict:
		return strictFastAccumulationConfig(estimatedCostBPS), nil
	case strategyFastAccumulationStrictShortBias:
		cfg := strictFastAccumulationConfig(estimatedCostBPS)
		cfg.StrategyName = name
		cfg.LongEnabled = false
		cfg.ShortEnabled = true
		return cfg, nil
	case strategyFastAccumulationStrictHighConf:
		cfg := strictFastAccumulationConfig(estimatedCostBPS)
		cfg.StrategyName = name
		cfg.MinEntryScore = 85
		return cfg, nil
	case strategyFastAccumulationStrictLowFreq:
		cfg := strictFastAccumulationConfig(estimatedCostBPS)
		cfg.StrategyName = name
		cfg.MinEntryScore = 85
		cfg.MaxTradesPerDay = 4
		cfg.MinMinutesBetweenEntries = 60
		return cfg, nil
	case strategyFastAccumulationStrictCostGuard:
		cfg := strictFastAccumulationConfig(estimatedCostBPS)
		cfg.StrategyName = name
		cfg.CostMultipleRequired = 5
		cfg.MinExpectedMoveBPS = 20
		return cfg, nil
	case strategyFastAccumulationStrictNo7084Longs:
		cfg := strictFastAccumulationConfig(estimatedCostBPS)
		cfg.StrategyName = name
		cfg.DisableLongScoreBucket70To84 = true
		cfg.LongMinEntryScore = 85
		cfg.ShortMinEntryScore = 75
		return cfg, nil
	case strategyFastAccumulationStrict30m:
		cfg := strictFastAccumulationConfig(estimatedCostBPS)
		cfg.StrategyName = name
		cfg.DecisionWindowMinutes = 30
		return cfg, nil
	case strategyFastAccumulationStrict1h:
		cfg := strictFastAccumulationConfig(estimatedCostBPS)
		cfg.StrategyName = name
		cfg.DecisionWindowMinutes = 60
		return cfg, nil
	case strategyFastAccumulationPullbackReclaim:
		cfg := strictFastAccumulationConfig(estimatedCostBPS)
		cfg.StrategyName = name
		cfg.EntryVariant = strategy.EntryVariantPullbackReclaim
		cfg.LongMinEntryScore = 85
		cfg.ShortMinEntryScore = 75
		return cfg, nil
	case strategyFastAccumulationBreakoutRetest:
		cfg := strictFastAccumulationConfig(estimatedCostBPS)
		cfg.StrategyName = name
		cfg.EntryVariant = strategy.EntryVariantBreakoutRetest
		cfg.MinTrendScore = 65
		cfg.MinExpectedMoveBPS = 15
		return cfg, nil
	case strategyFastAccumulationMomentumCont:
		cfg := strictFastAccumulationConfig(estimatedCostBPS)
		cfg.StrategyName = name
		cfg.EntryVariant = strategy.EntryVariantMomentumContinuation
		cfg.MinTrendScore = 70
		cfg.MinExpectedMoveBPS = 20
		cfg.CostMultipleRequired = 5
		return cfg, nil
	case strategyFastAccumulationPartialTrail:
		cfg := strictFastAccumulationConfig(estimatedCostBPS)
		cfg.StrategyName = name
		cfg.ExitModel = strategy.ExitModelPartialTPTrail
		cfg.PartialTakeProfitR = 1.0
		cfg.PartialTakeProfitFraction = 0.5
		cfg.TrailAfterMFER = 1.2
		cfg.TrailDistanceR = 0.5
		return cfg, nil
	case strategyFastAccumulationBreakevenGuard:
		cfg := strictFastAccumulationConfig(estimatedCostBPS)
		cfg.StrategyName = name
		cfg.ExitModel = strategy.ExitModelBreakevenAfter1R
		cfg.BreakevenTriggerR = 1.0
		return cfg, nil
	case strategyFastAccumulationCutNoProgress:
		cfg := strictFastAccumulationConfig(estimatedCostBPS)
		cfg.StrategyName = name
		cfg.ExitModel = strategy.ExitModelCutIfNoProgress
		cfg.CutNoProgressR = 0.5
		cfg.CutNoProgressWindows = 1
		return cfg, nil
	case strategyFastAccumulationEconomicsGuard:
		cfg := strictFastAccumulationConfig(estimatedCostBPS)
		cfg.StrategyName = name
		cfg.MinExpectedRAfterCost = 0.8
		cfg.MinTargetBPSAfterCost = 12
		cfg.MinRewardToRisk = 1.4
		return cfg, nil
	default:
		return strategy.FastAccumulationConfig{}, fmt.Errorf("unknown strategy %q", name)
	}
}

func fixedWalkForwardParamsForStrategy(name string) (*walkforward.Params, error) {
	switch name {
	case strategyFastAccumulationStrictShortBias,
		strategyFastAccumulationStrictHighConf,
		strategyFastAccumulationStrictLowFreq,
		strategyFastAccumulationStrictCostGuard,
		strategyFastAccumulationStrictNo7084Longs,
		strategyFastAccumulationStrict30m,
		strategyFastAccumulationStrict1h,
		strategyFastAccumulationPullbackReclaim,
		strategyFastAccumulationBreakoutRetest,
		strategyFastAccumulationMomentumCont,
		strategyFastAccumulationPartialTrail,
		strategyFastAccumulationBreakevenGuard,
		strategyFastAccumulationCutNoProgress,
		strategyFastAccumulationEconomicsGuard:
		cfg, err := fastAccumulationPresetConfig(name, 6.0)
		if err != nil {
			return nil, err
		}
		params := walkforwardParamsFromConfig(cfg)
		return &params, nil
	default:
		return nil, nil
	}
}

func walkforwardParamsFromConfig(cfg strategy.FastAccumulationConfig) walkforward.Params {
	return walkforward.Params{
		StrategyName:                      cfg.StrategyName,
		DecisionWindowMinutes:             cfg.DecisionWindowMinutes,
		EntryVariant:                      string(cfg.EntryVariant),
		ExitModel:                         string(cfg.ExitModel),
		FullTradeMinScore:                 cfg.FullTradeMinScore,
		NormalTradeMinScore:               cfg.NormalTradeMinScore,
		ProbeMinScore:                     cfg.ProbeMinScore,
		CostMultipleRequired:              cfg.CostMultipleRequired,
		MaxHoldWindows:                    cfg.MaxHoldWindows,
		TimeStopWindows:                   cfg.TimeStopWindows,
		AllowProbeTrade:                   cfg.AllowProbeTrade,
		DisableProbeTrades:                cfg.DisableProbeTrades,
		MaxChopScore:                      cfg.MaxChopScore,
		MinEntryScore:                     cfg.MinEntryScore,
		MinTrendScore:                     cfg.MinTrendScore,
		DisableScoreBucket55To69:          cfg.DisableScoreBucket55To69,
		RequireScoreBucket70Plus:          cfg.RequireScoreBucket70Plus,
		MinExpectedMoveBPS:                cfg.MinExpectedMoveBPS,
		RequireExpectedMoveGtCostMultiple: cfg.RequireExpectedMoveGtCostMultiple,
		LongEnabled:                       cfg.LongEnabled,
		ShortEnabled:                      cfg.ShortEnabled,
		LongMinEntryScore:                 cfg.LongMinEntryScore,
		ShortMinEntryScore:                cfg.ShortMinEntryScore,
		LongMinTrendScore:                 cfg.LongMinTrendScore,
		ShortMinTrendScore:                cfg.ShortMinTrendScore,
		LongMaxChopScore:                  cfg.LongMaxChopScore,
		ShortMaxChopScore:                 cfg.ShortMaxChopScore,
		LongMinExpectedMoveBPS:            cfg.LongMinExpectedMoveBPS,
		ShortMinExpectedMoveBPS:           cfg.ShortMinExpectedMoveBPS,
		LongCostMultipleRequired:          cfg.LongCostMultipleRequired,
		ShortCostMultipleRequired:         cfg.ShortCostMultipleRequired,
		DisableLongScoreBucket70To84:      cfg.DisableLongScoreBucket70To84,
		DisableShortScoreBucket70To84:     cfg.DisableShortScoreBucket70To84,
		MaxTradesPerDay:                   cfg.MaxTradesPerDay,
		MinMinutesBetweenEntries:          cfg.MinMinutesBetweenEntries,
		MinExpectedRAfterCost:             cfg.MinExpectedRAfterCost,
		MinTargetBPSAfterCost:             cfg.MinTargetBPSAfterCost,
		MinRewardToRisk:                   cfg.MinRewardToRisk,
		PartialTakeProfitR:                cfg.PartialTakeProfitR,
		PartialTakeProfitFraction:         cfg.PartialTakeProfitFraction,
		BreakevenTriggerR:                 cfg.BreakevenTriggerR,
		TrailAfterMFER:                    cfg.TrailAfterMFER,
		TrailDistanceR:                    cfg.TrailDistanceR,
		CutNoProgressR:                    cfg.CutNoProgressR,
		CutNoProgressWindows:              cfg.CutNoProgressWindows,
	}
}
