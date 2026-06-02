package app

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"time"

	"github.com/davidmiguel22573/ak-engine/internal/data"
	"github.com/davidmiguel22573/ak-engine/pkg/protocol"
	"github.com/spf13/cobra"
)

var (
	source   string
	path     string
	market   string
	symbol   string
	interval string
	from     string
	to       string
	format   string
)

type inspectResult struct {
	Source     string `json:"source"`
	Market     string `json:"market"`
	Symbol     string `json:"symbol"`
	Interval   string `json:"interval"`
	Count      int    `json:"count"`
	FirstMS    int64  `json:"first_ms"`
	LastMS     int64  `json:"last_ms"`
	Duplicates int    `json:"duplicates"`
	Gaps       int    `json:"gaps"`
	Status     string `json:"status"`
}

var inspectDatasetCmd = &cobra.Command{
	Use:   "inspect-dataset",
	Short: "Inspect a dataset to validate candles",
	RunE: func(cmd *cobra.Command, args []string) error {
		src, err := data.NewCandleSource(source)
		if err != nil {
			if format == "json" {
				printResultJSON(data.CandleRequest{Source: source, Market: market, Symbol: symbol, Interval: interval}, nil, "FAIL")
			}
			return err
		}

		fromTime, err := parseFromTime(from)
		if err != nil {
			return fmt.Errorf("invalid from time: %w", err)
		}
		toTime, err := parseToTime(to)
		if err != nil {
			return fmt.Errorf("invalid to time: %w", err)
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

		if source != "r2" && path == "" {
			if format == "json" {
				printResultJSON(req, nil, "FAIL")
			}
			return errors.New("missing path in request")
		}

		var candles []protocol.Candle
		var loadErr error

		if source == "local-json" {
			file, err := os.Open(path)
			var fileData []byte
			if err == nil {
				defer file.Close()
				fileData, err = io.ReadAll(file)
			}

			if err != nil {
				if format == "json" {
					printResultJSON(req, nil, "FAIL")
				}
				return fmt.Errorf("failed to read file %s: %w", path, err)
			}

			candles, loadErr = data.ParseJSONCandlesNoValidate(fileData, req)
		} else {
			candles, loadErr = src.LoadCandles(cmd.Context(), req)
		}

		if loadErr != nil {
			if format == "json" {
				printResultJSON(req, nil, "FAIL")
			}
			return fmt.Errorf("failed to load candles: %w", loadErr)
		}

		valErr := data.ValidateCandles(interval, candles)
		if valErr != nil {
			if format == "json" {
				printResultJSON(req, candles, "FAIL")
			}
			return fmt.Errorf("candle validation failed: %w", valErr)
		}

		if format == "json" {
			printResultJSON(req, candles, "PASS")
			return nil
		}

		fmt.Println("Dataset inspection passed.")
		return nil
	},
}

func parseFromTime(s string) (time.Time, error) {
	if s == "" {
		return time.Time{}, nil
	}
	if t, err := time.Parse("2006-01-02", s); err == nil {
		return t, nil
	}
	return time.Parse(time.RFC3339, s)
}

func parseToTime(s string) (time.Time, error) {
	if s == "" {
		return time.Time{}, nil
	}
	if t, err := time.Parse("2006-01-02", s); err == nil {
		return t.AddDate(0, 0, 1).Add(-time.Millisecond), nil
	}
	return time.Parse(time.RFC3339, s)
}

func printResultJSON(req data.CandleRequest, candles []protocol.Candle, status string) {
	var count int
	var firstMS, lastMS int64
	var duplicates, gaps int

	outMarket := req.Market
	outSymbol := req.Symbol
	outInterval := req.Interval

	if len(candles) > 0 {
		analysis := data.AnalyzeCandles(req.Interval, candles)
		count = analysis.Count
		firstMS = analysis.FirstMS
		lastMS = analysis.LastMS
		duplicates = analysis.Duplicates
		gaps = analysis.Gaps

		if outMarket == "" {
			outMarket = candles[0].Market
		}
		if outSymbol == "" {
			outSymbol = candles[0].Symbol
		}
		if outInterval == "" {
			outInterval = candles[0].Interval
		}
	}

	res := inspectResult{
		Source:     req.Source,
		Market:     outMarket,
		Symbol:     outSymbol,
		Interval:   outInterval,
		Count:      count,
		FirstMS:    firstMS,
		LastMS:     lastMS,
		Duplicates: duplicates,
		Gaps:       gaps,
		Status:     status,
	}

	bytes, _ := json.MarshalIndent(res, "", "  ")
	fmt.Println(string(bytes))
}

func init() {
	inspectDatasetCmd.Flags().StringVar(&source, "source", "", "Source of dataset (e.g. local-json)")
	inspectDatasetCmd.Flags().StringVar(&path, "path", "", "Path to the dataset file")
	inspectDatasetCmd.Flags().StringVar(&market, "market", "", "Market type (e.g. futures-um)")
	inspectDatasetCmd.Flags().StringVar(&symbol, "symbol", "", "Trading symbol (e.g. BTCUSDT)")
	inspectDatasetCmd.Flags().StringVar(&interval, "interval", "", "Candle interval (e.g. 5m)")
	inspectDatasetCmd.Flags().StringVar(&from, "from", "", "From date (YYYY-MM-DD)")
	inspectDatasetCmd.Flags().StringVar(&to, "to", "", "To date (YYYY-MM-DD)")
	inspectDatasetCmd.Flags().StringVar(&format, "format", "", "Output format (e.g. json)")

	rootCmd.AddCommand(inspectDatasetCmd)
}
