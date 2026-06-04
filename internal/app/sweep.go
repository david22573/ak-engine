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
	FullTradeMinScore    float64 `json:"full_trade_min_score"`
	NormalTradeMinScore  float64 `json:"normal_trade_min_score"`
	ProbeMinScore        float64 `json:"probe_min_score"`
	CostMultipleRequired float64 `json:"cost_multiple_required"`
	MaxHoldWindows       int     `json:"max_hold_windows"`
	TimeStopWindows      int     `json:"time_stop_windows"`
	AllowProbeTrade      bool    `json:"allow_probe_trade"`
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
)

var sweepCmd = &cobra.Command{
	Use:   "sweep",
	Short: "Run parameter sweep for Fast Accumulation strategy",
	RunE: func(cmd *cobra.Command, args []string) error {
		if format != "" && format != "json" {
			return fmt.Errorf("unsupported format %q; only json is supported", format)
		}
		if sweepStrategy != "fast_accumulation" {
			return fmt.Errorf("sweep command only supports strategy 'fast_accumulation'")
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

		// Define parameter sweep grid
		fullScores := []float64{85, 90}
		normalScores := []float64{70, 75, 80}
		probeScores := []float64{55, 60, 65}
		costMultiples := []float64{3, 4, 5}
		maxHolds := []int{2, 4}
		timeStops := []int{1, 2}
		allowProbes := []bool{true, false}

		var results []SweepResult

		for _, fs := range fullScores {
			for _, ns := range normalScores {
				for _, ps := range probeScores {
					// Validate constraint: full >= normal >= probe
					if fs < ns || ns < ps {
						continue
					}
					for _, cm := range costMultiples {
						for _, mh := range maxHolds {
							for _, ts := range timeStops {
								for _, ap := range allowProbes {
									p := SweepParams{
										FullTradeMinScore:    fs,
										NormalTradeMinScore:  ns,
										ProbeMinScore:        ps,
										CostMultipleRequired: cm,
										MaxHoldWindows:       mh,
										TimeStopWindows:      ts,
										AllowProbeTrade:      ap,
									}

									cfg := strategy.DefaultFastAccumulationConfig()
									cfg.FullTradeMinScore = p.FullTradeMinScore
									cfg.NormalTradeMinScore = p.NormalTradeMinScore
									cfg.ProbeMinScore = p.ProbeMinScore
									cfg.CostMultipleRequired = p.CostMultipleRequired
									cfg.MaxHoldWindows = p.MaxHoldWindows
									cfg.TimeStopWindows = p.TimeStopWindows
									cfg.AllowProbeTrade = p.AllowProbeTrade
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
							}
						}
					}
				}
			}
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

	rootCmd.AddCommand(sweepCmd)
}
