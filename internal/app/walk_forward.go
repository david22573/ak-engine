package app

import (
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/davidmiguel22573/ak-engine/internal/data"
	"github.com/davidmiguel22573/ak-engine/internal/walkforward"
	"github.com/spf13/cobra"
)

var (
	wfTrainWindow     string
	wfTestWindow      string
	wfTopCandidates   int
	wfMinTrades       int
	wfMaxDrawdown     float64
	wfMaxLossStreak   int
	wfMinProfitFactor float64
	wfSweepProfile    string
	wfStrategy        string
)

var walkforwardCmd = &cobra.Command{
	Use:   "walk-forward",
	Short: "Run a walk-forward optimization",
	RunE: func(cmd *cobra.Command, args []string) error {
		if format != "" && format != "json" {
			return fmt.Errorf("unsupported format %q; only json is supported", format)
		}
		if !isFastAccumulationStrategyName(wfStrategy) {
			return fmt.Errorf("walk-forward command only supports Fast Accumulation strategy families")
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

		parseDuration := func(s string) (time.Duration, error) {
			if len(s) > 1 && s[len(s)-1] == 'd' {
				var days int
				if _, err := fmt.Sscanf(s, "%dd", &days); err == nil {
					return time.Duration(days) * 24 * time.Hour, nil
				}
			}
			return time.ParseDuration(s)
		}

		trainWindow, err := parseDuration(wfTrainWindow)
		if err != nil {
			return fmt.Errorf("invalid train-window: %w", err)
		}
		testWindow, err := parseDuration(wfTestWindow)
		if err != nil {
			return fmt.Errorf("invalid test-window: %w", err)
		}

		cfg := walkforward.Config{
			TrainWindow:     trainWindow,
			TestWindow:      testWindow,
			TopCandidates:   wfTopCandidates,
			MinTrades:       wfMinTrades,
			MaxDrawdown:     wfMaxDrawdown,
			MaxLossStreak:   wfMaxLossStreak,
			MinProfitFactor: wfMinProfitFactor,
			SweepProfile:    wfSweepProfile,
		}
		fixedParams, err := fixedWalkForwardParamsForStrategy(wfStrategy)
		if err != nil {
			return err
		}
		cfg.FixedParams = fixedParams

		src, err := data.NewCandleSource(source)
		if err != nil {
			return err
		}

		req := data.CandleRequest{
			Source:   source,
			Market:   market,
			Symbol:   symbol,
			Interval: interval,
			From:     fromTime,
			To:       toTime,
			Path:     path,
		}

		candles, err := src.LoadCandles(cmd.Context(), req)
		if err != nil {
			return err
		}
		if err := data.ValidateCandles(interval, candles); err != nil {
			return err
		}

		if wfStrategy == strategyFastAccumulationStrict {
			cfg.SweepProfile = "strict"
		} else if fixedParams == nil && wfSweepProfile == "default" && wfStrategy != strategyFastAccumulation {
			cfg.SweepProfile = "calibration"
		}

		res, err := walkforward.Run(cmd.Context(), cfg, src, req, candles)
		if err != nil {
			return err
		}
		res.Strategy = wfStrategy

		enc := json.NewEncoder(cmd.OutOrStdout())
		enc.SetIndent("", "  ")
		return enc.Encode(res)
	},
}

func init() {
	walkforwardCmd.Flags().StringVar(&source, "source", "", "Source of candles (local-json, local-parquet, r2)")
	walkforwardCmd.Flags().StringVar(&path, "path", "", "Path to local candle dataset")
	walkforwardCmd.Flags().StringVar(&market, "market", "", "Market type (e.g. futures-um)")
	walkforwardCmd.Flags().StringVar(&symbol, "symbol", "", "Trading symbol (e.g. BTCUSDT)")
	walkforwardCmd.Flags().StringVar(&interval, "interval", "", "Candle interval (e.g. 5m)")
	walkforwardCmd.Flags().StringVar(&from, "from", "", "From date (YYYY-MM-DD)")
	walkforwardCmd.Flags().StringVar(&to, "to", "", "To date (YYYY-MM-DD)")
	walkforwardCmd.Flags().StringVar(&format, "format", "json", "Output format (json)")

	walkforwardCmd.Flags().StringVar(&wfStrategy, "strategy", "fast_accumulation", "Strategy to run")
	walkforwardCmd.Flags().StringVar(&wfTrainWindow, "train-window", "", "Train window duration (e.g. 90d, 45m)")
	walkforwardCmd.Flags().StringVar(&wfTestWindow, "test-window", "", "Test window duration (e.g. 30d, 15m)")
	walkforwardCmd.Flags().IntVar(&wfTopCandidates, "top-candidates", 5, "Number of top candidates to keep")
	walkforwardCmd.Flags().IntVar(&wfMinTrades, "min-trades", 5, "Minimum number of trades required")
	walkforwardCmd.Flags().Float64Var(&wfMaxDrawdown, "max-drawdown", 1000000, "Maximum allowed drawdown")
	walkforwardCmd.Flags().IntVar(&wfMaxLossStreak, "max-loss-streak", 999, "Maximum allowed consecutive losses")
	walkforwardCmd.Flags().Float64Var(&wfMinProfitFactor, "min-profit-factor", 0, "Minimum required profit factor")
	walkforwardCmd.Flags().StringVar(&wfSweepProfile, "sweep-profile", "default", "Sweep profile (default, strict, calibration)")

	rootCmd.AddCommand(walkforwardCmd)
}
