package app

import (
	"encoding/json"
	"errors"
	"fmt"

	"github.com/davidmiguel22573/ak-engine/internal/backtest"
	"github.com/davidmiguel22573/ak-engine/internal/data"
	"github.com/davidmiguel22573/ak-engine/internal/strategy"
	"github.com/spf13/cobra"
)

var (
	backtestStrategy      string
	startingCash          float64
	maxPositionSize       float64
	makerFeeBPS           float64
	takerFeeBPS           float64
	slippageBPS           float64
	baselineThresholdBPS  float64
	baselineStopLossBPS   float64
	baselineTakeProfitBPS float64
	baselineMaxHold       int
	faFullTradeMinScore   float64
	faNormalTradeMinScore float64
	faProbeMinScore       float64
	faCostMultiple        float64
	faAllowProbe          bool
	faForceFullTrade      bool
	includeDecisions      bool
)

var backtestCmd = &cobra.Command{
	Use:   "backtest",
	Short: "Run deterministic backtest simulation",
	RunE: func(cmd *cobra.Command, args []string) error {
		if format != "" && format != "json" {
			return fmt.Errorf("unsupported format %q; only json is supported", format)
		}
		if source == "" {
			return errors.New("missing source in request")
		}
		if source != "r2" && path == "" {
			return errors.New("missing path in request")
		}

		fromTime, err := parseFromTime(from)
		if err != nil {
			return fmt.Errorf("invalid from time: %w", err)
		}
		toTime, err := parseToTime(to)
		if err != nil {
			return fmt.Errorf("invalid to time: %w", err)
		}

		src, err := data.NewCandleSource(source)
		if err != nil {
			return err
		}

		strat, err := newBacktestStrategy()
		if err != nil {
			return err
		}

		engine, err := backtest.NewEngine(src, strat, backtest.Config{
			StartingCash:    startingCash,
			MaxPositionSize: maxPositionSize,
			SlippageBPS:     slippageBPS,
			Fees: backtest.FeeConfig{
				MakerFeeBPS: makerFeeBPS,
				TakerFeeBPS: takerFeeBPS,
			},
			IncludeDecisions: includeDecisions,
		})
		if err != nil {
			return err
		}

		report, err := engine.Run(cmd.Context(), data.CandleRequest{
			Source:   source,
			Market:   market,
			Symbol:   symbol,
			Interval: interval,
			From:     fromTime,
			To:       toTime,
			Path:     path,
		})
		if err != nil {
			return err
		}
		report.Strategy = backtestStrategy
		report.PresetName = backtestStrategy

		enc := json.NewEncoder(cmd.OutOrStdout())
		enc.SetIndent("", "  ")
		return enc.Encode(report)
	},
}

func newBacktestStrategy() (strategy.Strategy, error) {
	switch backtestStrategy {
	case "baseline":
		return strategy.NewBaseline(strategy.BaselineConfig{
			ThresholdBPS:   baselineThresholdBPS,
			StopLossBPS:    baselineStopLossBPS,
			TakeProfitBPS:  baselineTakeProfitBPS,
			MaxHoldCandles: baselineMaxHold,
		})
	case "fast_accumulation":
		cfg := strategy.DefaultFastAccumulationConfig()
		cfg.StrategyName = strategyFastAccumulation
		cfg.FullTradeMinScore = faFullTradeMinScore
		cfg.NormalTradeMinScore = faNormalTradeMinScore
		cfg.ProbeMinScore = faProbeMinScore
		cfg.CostMultipleRequired = faCostMultiple
		cfg.AllowProbeTrade = faAllowProbe
		cfg.ForceFullTrade = faForceFullTrade
		cfg.EstimatedCostBPS = takerFeeBPS + slippageBPS
		return strategy.NewFastAccumulation(cfg)
	case strategyFastAccumulationStrict,
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
		cfg, err := fastAccumulationPresetConfig(backtestStrategy, takerFeeBPS+slippageBPS)
		if err != nil {
			return nil, err
		}
		return strategy.NewFastAccumulation(cfg)
	default:
		return nil, fmt.Errorf("unknown strategy %q", backtestStrategy)
	}
}

func init() {
	backtestCmd.Flags().StringVar(&source, "source", "", "Source of candles (local-json, local-parquet, r2)")
	backtestCmd.Flags().StringVar(&path, "path", "", "Path to local candle dataset")
	backtestCmd.Flags().StringVar(&market, "market", "", "Market type (e.g. futures-um)")
	backtestCmd.Flags().StringVar(&symbol, "symbol", "", "Trading symbol (e.g. BTCUSDT)")
	backtestCmd.Flags().StringVar(&interval, "interval", "", "Candle interval (e.g. 5m)")
	backtestCmd.Flags().StringVar(&from, "from", "", "From date (YYYY-MM-DD)")
	backtestCmd.Flags().StringVar(&to, "to", "", "To date (YYYY-MM-DD)")
	backtestCmd.Flags().StringVar(&format, "format", "json", "Output format (json)")
	backtestCmd.Flags().StringVar(&backtestStrategy, "strategy", "baseline", "Strategy to run")
	backtestCmd.Flags().Float64Var(&startingCash, "starting-cash", 10000, "Starting cash for simulation")
	backtestCmd.Flags().Float64Var(&maxPositionSize, "max-position-size", 1, "Max fraction of equity per trade")
	backtestCmd.Flags().Float64Var(&makerFeeBPS, "maker-fee-bps", 0, "Maker fee in basis points")
	backtestCmd.Flags().Float64Var(&takerFeeBPS, "taker-fee-bps", 5, "Taker fee in basis points")
	backtestCmd.Flags().Float64Var(&slippageBPS, "slippage-bps", 1, "Slippage in basis points")
	backtestCmd.Flags().Float64Var(&baselineThresholdBPS, "baseline-threshold-bps", 5, "Baseline momentum threshold in basis points")
	backtestCmd.Flags().Float64Var(&baselineStopLossBPS, "baseline-stop-loss-bps", 50, "Baseline stop loss in basis points")
	backtestCmd.Flags().Float64Var(&baselineTakeProfitBPS, "baseline-take-profit-bps", 100, "Baseline take profit in basis points")
	backtestCmd.Flags().IntVar(&baselineMaxHold, "baseline-max-hold-candles", 12, "Baseline max hold candles")
	backtestCmd.Flags().Float64Var(&faFullTradeMinScore, "fa-full-trade-min-score", 85, "Fast Accumulation full-trade minimum score")
	backtestCmd.Flags().Float64Var(&faNormalTradeMinScore, "fa-normal-trade-min-score", 70, "Fast Accumulation normal-trade minimum score")
	backtestCmd.Flags().Float64Var(&faProbeMinScore, "fa-probe-min-score", 55, "Fast Accumulation probe-trade minimum score")
	backtestCmd.Flags().Float64Var(&faCostMultiple, "fa-cost-multiple", 3, "Fast Accumulation expected-move cost multiple")
	backtestCmd.Flags().BoolVar(&faAllowProbe, "fa-allow-probe", true, "Fast Accumulation allow probe trades")
	backtestCmd.Flags().BoolVar(&faForceFullTrade, "fa-force-full-trade", false, "Fast Accumulation force full-size trades when tradeable")
	backtestCmd.Flags().BoolVar(&includeDecisions, "include-decisions", false, "Include per-window decisions in the JSON report")

	rootCmd.AddCommand(backtestCmd)
}
