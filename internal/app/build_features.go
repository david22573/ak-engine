package app

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/davidmiguel22573/ak-engine/internal/data"
	"github.com/davidmiguel22573/ak-engine/internal/features"
	"github.com/davidmiguel22573/ak-engine/internal/research"
	"github.com/davidmiguel22573/ak-engine/pkg/protocol"
	"github.com/spf13/cobra"
)

var (
	bfSource         string
	bfPath           string
	bfMarket         string
	bfSymbol         string
	bfInterval       string
	bfFrom           string
	bfTo             string
	bfContextSymbols string
	bfOut            string
	bfFormat         string
	bfDropWarmup     bool
)

type buildFeaturesResult struct {
	Status           string `json:"status"`
	Rows             int    `json:"rows"`
	WarmupRows       int    `json:"warmup_rows"`
	FirstEventTimeMS int64  `json:"first_event_time_ms"`
	LastEventTimeMS  int64  `json:"last_event_time_ms"`
	Out              string `json:"out"`
}

var buildFeaturesCmd = &cobra.Command{
	Use:   "build-features",
	Short: "Build deterministic candle-based features",
	RunE: func(cmd *cobra.Command, args []string) error {
		src, err := data.NewCandleSource(bfSource)
		if err != nil {
			return err
		}

		fromTime, err := parseFromTime(bfFrom)
		if err != nil {
			return fmt.Errorf("invalid from time: %w", err)
		}
		toTime, err := parseToTime(bfTo)
		if err != nil {
			return fmt.Errorf("invalid to time: %w", err)
		}

		req := data.CandleRequest{
			Source:   bfSource,
			Market:   bfMarket,
			Symbol:   bfSymbol,
			Interval: bfInterval,
			From:     fromTime,
			To:       toTime,
			Path:     bfPath,
		}

		if bfSource != "r2" && bfPath == "" {
			return fmt.Errorf("missing path in request")
		}

		// Load primary candles
		var candles []protocol.Candle
		if bfSource == "local-json" {
			fileData, err := os.ReadFile(bfPath)
			if err != nil {
				return fmt.Errorf("failed to read file %s: %w", bfPath, err)
			}
			candles, err = data.ParseJSONCandlesNoValidate(fileData, req)
			if err != nil {
				return fmt.Errorf("failed to parse primary JSON candles: %w", err)
			}
		} else {
			candles, err = src.LoadCandles(cmd.Context(), req)
			if err != nil {
				return fmt.Errorf("failed to load primary candles: %w", err)
			}
		}

		// Load context candles (e.g. BTC, ETH)
		var btcCandles []protocol.Candle
		var ethCandles []protocol.Candle

		if bfContextSymbols != "" {
			syms := contextSymbolListForTarget(bfSymbol, bfContextSymbols)
			for _, sym := range syms {
				ctxReq := req
				ctxReq.Symbol = sym

				var ctxCandles []protocol.Candle
				if bfSource == "local-json" {
					ctxPath := getLocalContextJSONPath(bfPath, bfSymbol, sym)
					if fileData, err := os.ReadFile(ctxPath); err == nil {
						ctxCandles, err = data.ParseJSONCandlesNoValidate(fileData, ctxReq)
						if err != nil {
							return fmt.Errorf("failed to parse context JSON for %s: %w", sym, err)
						}
					} else {
						return fmt.Errorf("failed to read context JSON for %s: %w", sym, err)
					}
				} else {
					ctxCandles, err = src.LoadCandles(cmd.Context(), ctxReq)
					if err != nil {
						return fmt.Errorf("failed to load context candles for %s: %w", sym, err)
					}
				}

				if sym == "BTCUSDT" {
					btcCandles = ctxCandles
				} else if sym == "ETHUSDT" {
					ethCandles = ctxCandles
				}
			}
		}

		// Build feature rows
		opts := features.BuildOptions{
			Market:     bfMarket,
			Symbol:     bfSymbol,
			Interval:   bfInterval,
			DropWarmup: bfDropWarmup,
			ContextBTC: btcCandles,
			ContextETH: ethCandles,
		}

		rows, err := features.BuildRows(candles, opts)
		if err != nil {
			return fmt.Errorf("failed to build feature rows: %w", err)
		}

		// Run leakage check
		leakReport := research.CheckFeatureRows(rows)
		if leakReport.Status != "PASS" {
			issuesJSON, _ := json.Marshal(leakReport.Issues)
			return fmt.Errorf("leakage check failed with issues: %s", string(issuesJSON))
		}

		// Ensure parent directory of output exists
		if bfOut != "" {
			if err := os.MkdirAll(filepath.Dir(bfOut), 0755); err != nil {
				return fmt.Errorf("failed to create output directory: %w", err)
			}
		}

		// Write output
		switch bfFormat {
		case "json":
			if err := features.WriteRowsJSON(bfOut, rows); err != nil {
				return fmt.Errorf("failed to write JSON: %w", err)
			}
		case "csv":
			if err := features.WriteRowsCSV(bfOut, rows); err != nil {
				return fmt.Errorf("failed to write CSV: %w", err)
			}
		case "parquet":
			// Write temporary CSV first, then convert to Parquet
			tmpCsv := bfOut + ".tmp.csv"
			if err := features.WriteRowsCSV(tmpCsv, rows); err != nil {
				return fmt.Errorf("failed to write temporary CSV: %w", err)
			}
			defer os.Remove(tmpCsv)
			if err := features.WriteRowsParquet(tmpCsv, bfOut); err != nil {
				return fmt.Errorf("failed to write Parquet: %w", err)
			}
		default:
			return fmt.Errorf("unsupported output format %q", bfFormat)
		}

		// Print summary JSON
		var warmupCount int
		for _, r := range rows {
			if r.Warmup {
				warmupCount++
			}
		}

		var firstMS, lastMS int64
		if len(rows) > 0 {
			firstMS = rows[0].EventTimeMS
			lastMS = rows[len(rows)-1].EventTimeMS
		}

		res := buildFeaturesResult{
			Status:           "PASS",
			Rows:             len(rows),
			WarmupRows:       warmupCount,
			FirstEventTimeMS: firstMS,
			LastEventTimeMS:  lastMS,
			Out:              bfOut,
		}

		resBytes, _ := json.MarshalIndent(res, "", "  ")
		fmt.Println(string(resBytes))
		return nil
	},
}

func getLocalContextJSONPath(primaryPath, primarySymbol, targetSymbol string) string {
	base := filepath.Base(primaryPath)
	dir := filepath.Dir(primaryPath)
	if strings.Contains(base, primarySymbol) {
		subBase := strings.Replace(base, primarySymbol, targetSymbol, 1)
		subPath := filepath.Join(dir, subBase)
		if _, err := os.Stat(subPath); err == nil {
			return subPath
		}
	}
	subPath := filepath.Join(dir, targetSymbol+".json")
	if _, err := os.Stat(subPath); err == nil {
		return subPath
	}
	return filepath.Join(dir, targetSymbol+".json")
}

func contextSymbolListForTarget(targetSymbol, contextCSV string) []string {
	targetSymbol = strings.ToUpper(strings.TrimSpace(targetSymbol))
	seen := make(map[string]bool)
	var symbols []string
	for _, raw := range strings.Split(contextCSV, ",") {
		sym := strings.ToUpper(strings.TrimSpace(raw))
		if sym == "" || sym == targetSymbol || seen[sym] {
			continue
		}
		seen[sym] = true
		symbols = append(symbols, sym)
	}
	return symbols
}

func init() {
	buildFeaturesCmd.Flags().StringVar(&bfSource, "source", "", "Source of dataset (local-json | local-parquet | r2)")
	buildFeaturesCmd.Flags().StringVar(&bfPath, "path", "", "Path to the dataset directory/file")
	buildFeaturesCmd.Flags().StringVar(&bfMarket, "market", "", "Market type (e.g. futures-um)")
	buildFeaturesCmd.Flags().StringVar(&bfSymbol, "symbol", "", "Trading symbol (e.g. LINKUSDT)")
	buildFeaturesCmd.Flags().StringVar(&bfInterval, "interval", "", "Candle interval (e.g. 1m)")
	buildFeaturesCmd.Flags().StringVar(&bfFrom, "from", "", "From date (YYYY-MM-DD or RFC3339)")
	buildFeaturesCmd.Flags().StringVar(&bfTo, "to", "", "To date (YYYY-MM-DD or RFC3339)")
	buildFeaturesCmd.Flags().StringVar(&bfContextSymbols, "context-symbols", "BTCUSDT,ETHUSDT", "Comma-separated context symbols (e.g. BTCUSDT,ETHUSDT)")
	buildFeaturesCmd.Flags().StringVar(&bfOut, "out", "", "Output path")
	buildFeaturesCmd.Flags().StringVar(&bfFormat, "format", "json", "Output format (json | csv | parquet)")
	buildFeaturesCmd.Flags().BoolVar(&bfDropWarmup, "drop-warmup", false, "Drop warmup rows from output")

	_ = buildFeaturesCmd.MarkFlagRequired("source")
	_ = buildFeaturesCmd.MarkFlagRequired("market")
	_ = buildFeaturesCmd.MarkFlagRequired("symbol")
	_ = buildFeaturesCmd.MarkFlagRequired("interval")
	_ = buildFeaturesCmd.MarkFlagRequired("out")

	rootCmd.AddCommand(buildFeaturesCmd)
}
