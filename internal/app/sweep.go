package app

import (
	"encoding/json"
	"errors"
	"fmt"
	"sort"

	"github.com/davidmiguel22573/ak-engine/internal/backtest"
	"github.com/davidmiguel22573/ak-engine/internal/data"
	"github.com/davidmiguel22573/ak-engine/internal/strategy"
	"github.com/spf13/cobra"
)

type SweepParams struct {
	FullTradeMinScore                 float64 `json:"full_trade_min_score"`
	NormalTradeMinScore               float64 `json:"normal_trade_min_score"`
	ProbeMinScore                     float64 `json:"probe_min_score"`
	CostMultipleRequired              float64 `json:"cost_multiple_required"`
	MaxHoldWindows                    int     `json:"max_hold_windows"`
	TimeStopWindows                   int     `json:"time_stop_windows"`
	AllowProbeTrade                   bool    `json:"allow_probe_trade"`
	DisableProbeTrades                bool    `json:"disable_probe_trades,omitempty"`
	MaxChopScore                      float64 `json:"max_chop_score,omitempty"`
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
}

type SweepResult struct {
	Params               SweepParams `json:"params"`
	TotalTrades          int         `json:"total_trades"`
	Wins                 int         `json:"wins"`
	Losses               int         `json:"losses"`
	WinRate              float64     `json:"win_rate"`
	NetPnL               float64     `json:"net_pnl"`
	FeesPaid             float64     `json:"fees_paid"`
	SlippagePaid         float64     `json:"slippage_paid"`
	ProfitFactor         float64     `json:"profit_factor"`
	MaxDrawdown          float64     `json:"max_drawdown"`
	MaxConsecutiveLosses int         `json:"max_consecutive_losses"`
	Expectancy           float64     `json:"expectancy"`
	AverageHoldMinutes   float64     `json:"average_hold_minutes"`
	EndingCash           float64     `json:"ending_cash"`
	Status               string      `json:"status"`
}

var (
	sweepStrategy        string
	sweepStartingCash    float64
	sweepMaxPositionSize float64
	sweepMakerFeeBPS     float64
	sweepTakerFeeBPS     float64
	sweepSlippageBPS     float64
	sweepProfile         string
)

var sweepCmd = &cobra.Command{
	Use:   "sweep",
	Short: "Run parameter sweep for Fast Accumulation strategy",
	RunE: func(cmd *cobra.Command, args []string) error {
		if format != "" && format != "json" {
			return fmt.Errorf("unsupported format %q; only json is supported", format)
		}
		if !isFastAccumulationStrategyName(sweepStrategy) {
			return fmt.Errorf("sweep command only supports Fast Accumulation strategies")
		}
		if sweepProfile != "default" && sweepProfile != "strict" && sweepProfile != "calibration" {
			return fmt.Errorf("unsupported sweep-profile %q; expected 'default', 'strict', or 'calibration'", sweepProfile)
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

		candles, err := src.LoadCandles(cmd.Context(), data.CandleRequest{
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
		if err := data.ValidateCandles(interval, candles); err != nil {
			return err
		}

		var results []SweepResult

		for _, p := range buildSweepParams(sweepProfile) {
			cfg := strategy.DefaultFastAccumulationConfig()
			cfg.FullTradeMinScore = p.FullTradeMinScore
			cfg.NormalTradeMinScore = p.NormalTradeMinScore
			cfg.ProbeMinScore = p.ProbeMinScore
			cfg.CostMultipleRequired = p.CostMultipleRequired
			cfg.MaxHoldWindows = p.MaxHoldWindows
			cfg.TimeStopWindows = p.TimeStopWindows
			cfg.AllowProbeTrade = p.AllowProbeTrade
			cfg.DisableProbeTrades = p.DisableProbeTrades
			if p.MaxChopScore > 0 {
				cfg.MaxChopScore = p.MaxChopScore
			}
			cfg.MinTrendScore = p.MinTrendScore
			cfg.DisableScoreBucket55To69 = p.DisableScoreBucket55To69
			cfg.RequireScoreBucket70Plus = p.RequireScoreBucket70Plus
			cfg.MinExpectedMoveBPS = p.MinExpectedMoveBPS
			cfg.RequireExpectedMoveGtCostMultiple = p.RequireExpectedMoveGtCostMultiple
			cfg.LongEnabled = p.LongEnabled
			cfg.ShortEnabled = p.ShortEnabled
			cfg.LongMinEntryScore = p.LongMinEntryScore
			cfg.ShortMinEntryScore = p.ShortMinEntryScore
			cfg.LongMinTrendScore = p.LongMinTrendScore
			cfg.ShortMinTrendScore = p.ShortMinTrendScore
			cfg.LongMaxChopScore = p.LongMaxChopScore
			cfg.ShortMaxChopScore = p.ShortMaxChopScore
			cfg.LongMinExpectedMoveBPS = p.LongMinExpectedMoveBPS
			cfg.ShortMinExpectedMoveBPS = p.ShortMinExpectedMoveBPS
			cfg.LongCostMultipleRequired = p.LongCostMultipleRequired
			cfg.ShortCostMultipleRequired = p.ShortCostMultipleRequired
			cfg.DisableLongScoreBucket70To84 = p.DisableLongScoreBucket70To84
			cfg.DisableShortScoreBucket70To84 = p.DisableShortScoreBucket70To84
			cfg.MaxTradesPerDay = p.MaxTradesPerDay
			cfg.MinMinutesBetweenEntries = p.MinMinutesBetweenEntries
			cfg.EstimatedCostBPS = sweepTakerFeeBPS + sweepSlippageBPS

			strat, err := strategy.NewFastAccumulation(cfg)
			if err != nil {
				continue
			}

			engine, err := backtest.NewEngine(src, strat, backtest.Config{
				StartingCash:    sweepStartingCash,
				MaxPositionSize: sweepMaxPositionSize,
				SlippageBPS:     sweepSlippageBPS,
				Fees: backtest.FeeConfig{
					MakerFeeBPS: sweepMakerFeeBPS,
					TakerFeeBPS: sweepTakerFeeBPS,
				},
			})
			if err != nil {
				continue
			}

			report, err := engine.RunCandles(cmd.Context(), data.CandleRequest{
				Source:   source,
				Market:   market,
				Symbol:   symbol,
				Interval: interval,
				From:     fromTime,
				To:       toTime,
				Path:     path,
			}, candles)
			if err != nil {
				continue
			}

			results = append(results, SweepResult{
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
			})
		}

		// Sort results by net_pnl desc, then max_drawdown asc, then max_consecutive_losses asc
		sort.Slice(results, func(i, j int) bool {
			if results[i].NetPnL != results[j].NetPnL {
				return results[i].NetPnL > results[j].NetPnL
			}
			if results[i].MaxDrawdown != results[j].MaxDrawdown {
				return results[i].MaxDrawdown < results[j].MaxDrawdown
			}
			return results[i].MaxConsecutiveLosses < results[j].MaxConsecutiveLosses
		})

		enc := json.NewEncoder(cmd.OutOrStdout())
		enc.SetIndent("", "  ")
		return enc.Encode(results)
	},
}

func init() {
	sweepCmd.Flags().StringVar(&source, "source", "", "Source of candles (local-json, local-parquet, r2)")
	sweepCmd.Flags().StringVar(&path, "path", "", "Path to local candle dataset")
	sweepCmd.Flags().StringVar(&market, "market", "", "Market type (e.g. futures-um)")
	sweepCmd.Flags().StringVar(&symbol, "symbol", "", "Trading symbol (e.g. BTCUSDT)")
	sweepCmd.Flags().StringVar(&interval, "interval", "", "Candle interval (e.g. 5m)")
	sweepCmd.Flags().StringVar(&from, "from", "", "From date (YYYY-MM-DD)")
	sweepCmd.Flags().StringVar(&to, "to", "", "To date (YYYY-MM-DD)")
	sweepCmd.Flags().StringVar(&format, "format", "json", "Output format (json)")
	sweepCmd.Flags().StringVar(&sweepStrategy, "strategy", "fast_accumulation", "Strategy to run")
	sweepCmd.Flags().Float64Var(&sweepStartingCash, "starting-cash", 10000, "Starting cash for simulation")
	sweepCmd.Flags().Float64Var(&sweepMaxPositionSize, "max-position-size", 1, "Max fraction of equity per trade")
	sweepCmd.Flags().Float64Var(&sweepMakerFeeBPS, "maker-fee-bps", 0, "Maker fee in basis points")
	sweepCmd.Flags().Float64Var(&sweepTakerFeeBPS, "taker-fee-bps", 5, "Taker fee in basis points")
	sweepCmd.Flags().Float64Var(&sweepSlippageBPS, "slippage-bps", 1, "Slippage in basis points")
	sweepCmd.Flags().StringVar(&sweepProfile, "sweep-profile", "default", "Sweep profile (default, strict, calibration)")

	rootCmd.AddCommand(sweepCmd)
}

func buildSweepParams(profile string) []SweepParams {
	if profile == "strict" {
		maxChopScores := []float64{60, 50, 40}
		minTrendScores := []float64{50, 60, 70}
		require70s := []bool{true, false}
		disable55s := []bool{true, false}
		minExpectedMoves := []float64{0, 10, 15, 20}
		longEnableds := []bool{true, false}
		shortEnableds := []bool{true, false}

		var params []SweepParams
		for _, maxChopScore := range maxChopScores {
			for _, minTrendScore := range minTrendScores {
				for _, require70 := range require70s {
					for _, disable55 := range disable55s {
						for _, minExpectedMove := range minExpectedMoves {
							for _, longEnabled := range longEnableds {
								for _, shortEnabled := range shortEnableds {
									if !longEnabled && !shortEnabled {
										continue
									}
									params = append(params, SweepParams{
										FullTradeMinScore:                 85,
										NormalTradeMinScore:               70,
										ProbeMinScore:                     55,
										CostMultipleRequired:              4,
										MaxHoldWindows:                    4,
										TimeStopWindows:                   2,
										AllowProbeTrade:                   false,
										DisableProbeTrades:                true,
										MaxChopScore:                      maxChopScore,
										MinTrendScore:                     minTrendScore,
										DisableScoreBucket55To69:          disable55,
										RequireScoreBucket70Plus:          require70,
										MinExpectedMoveBPS:                minExpectedMove,
										RequireExpectedMoveGtCostMultiple: true,
										LongEnabled:                       longEnabled,
										ShortEnabled:                      shortEnabled,
									})
								}
							}
						}
					}
				}
			}
		}
		return params
	}
	if profile == "calibration" {
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

		var params []SweepParams
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
										params = append(params, SweepParams{
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
										})
									}
								}
							}
						}
					}
				}
			}
		}
		return params
	}

	fullScores := []float64{85, 90}
	normalScores := []float64{70, 75, 80}
	probeScores := []float64{55, 60, 65}
	costMultiples := []float64{3, 4, 5}
	maxHolds := []int{2, 4}
	timeStops := []int{1, 2}
	allowProbes := []bool{true, false}

	var params []SweepParams
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
								params = append(params, SweepParams{
									FullTradeMinScore:    fs,
									NormalTradeMinScore:  ns,
									ProbeMinScore:        ps,
									CostMultipleRequired: cm,
									MaxHoldWindows:       mh,
									TimeStopWindows:      ts,
									AllowProbeTrade:      ap,
									LongEnabled:          true,
									ShortEnabled:         true,
								})
							}
						}
					}
				}
			}
		}
	}

	return params
}
