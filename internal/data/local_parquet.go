package data

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sort"

	"github.com/david22573/ak-engine/pkg/protocol"
	"github.com/xitongsys/parquet-go-source/local"
	"github.com/xitongsys/parquet-go/reader"
)

type LocalParquetSource struct{}

func NewLocalParquetSource() *LocalParquetSource {
	return &LocalParquetSource{}
}

func (s *LocalParquetSource) Name() string {
	return "local-parquet"
}

type CheckMS struct {
	OpenTimeMS *int64 `parquet:"name=open_time_ms, type=INT64, repetitiontype=OPTIONAL"`
}

type ParquetCandleWithMS struct {
	Market              *string  `parquet:"name=market, type=BYTE_ARRAY, convertedtype=UTF8, repetitiontype=OPTIONAL"`
	Symbol              *string  `parquet:"name=symbol, type=BYTE_ARRAY, convertedtype=UTF8, repetitiontype=OPTIONAL"`
	Interval            *string  `parquet:"name=interval, type=BYTE_ARRAY, convertedtype=UTF8, repetitiontype=OPTIONAL"`
	OpenTimeMS          *int64   `parquet:"name=open_time_ms, type=INT64, repetitiontype=OPTIONAL"`
	Open                *float64 `parquet:"name=open, type=DOUBLE, repetitiontype=OPTIONAL"`
	High                *float64 `parquet:"name=high, type=DOUBLE, repetitiontype=OPTIONAL"`
	Low                 *float64 `parquet:"name=low, type=DOUBLE, repetitiontype=OPTIONAL"`
	Close               *float64 `parquet:"name=close, type=DOUBLE, repetitiontype=OPTIONAL"`
	Volume              *float64 `parquet:"name=volume, type=DOUBLE, repetitiontype=OPTIONAL"`
	CloseTimeMS         *int64   `parquet:"name=close_time_ms, type=INT64, repetitiontype=OPTIONAL"`
	QuoteAssetVolume    *float64 `parquet:"name=quote_asset_volume, type=DOUBLE, repetitiontype=OPTIONAL"`
	NumberOfTrades      *int64   `parquet:"name=number_of_trades, type=INT64, repetitiontype=OPTIONAL"`
	TakerBuyBaseVolume  *float64 `parquet:"name=taker_buy_base_volume, type=DOUBLE, repetitiontype=OPTIONAL"`
	TakerBuyQuoteVolume *float64 `parquet:"name=taker_buy_quote_volume, type=DOUBLE, repetitiontype=OPTIONAL"`
}

type ParquetCandleWithoutMS struct {
	Market              *string  `parquet:"name=market, type=BYTE_ARRAY, convertedtype=UTF8, repetitiontype=OPTIONAL"`
	Symbol              *string  `parquet:"name=symbol, type=BYTE_ARRAY, convertedtype=UTF8, repetitiontype=OPTIONAL"`
	Interval            *string  `parquet:"name=interval, type=BYTE_ARRAY, convertedtype=UTF8, repetitiontype=OPTIONAL"`
	OpenTime            *int64   `parquet:"name=open_time, type=INT64, repetitiontype=OPTIONAL"`
	Open                *float64 `parquet:"name=open, type=DOUBLE, repetitiontype=OPTIONAL"`
	High                *float64 `parquet:"name=high, type=DOUBLE, repetitiontype=OPTIONAL"`
	Low                 *float64 `parquet:"name=low, type=DOUBLE, repetitiontype=OPTIONAL"`
	Close               *float64 `parquet:"name=close, type=DOUBLE, repetitiontype=OPTIONAL"`
	Volume              *float64 `parquet:"name=volume, type=DOUBLE, repetitiontype=OPTIONAL"`
	CloseTime           *int64   `parquet:"name=close_time, type=INT64, repetitiontype=OPTIONAL"`
	QuoteAssetVolume    *float64 `parquet:"name=quote_asset_volume, type=DOUBLE, repetitiontype=OPTIONAL"`
	NumberOfTrades      *int64   `parquet:"name=number_of_trades, type=INT64, repetitiontype=OPTIONAL"`
	TakerBuyBaseVolume  *float64 `parquet:"name=taker_buy_base_volume, type=DOUBLE, repetitiontype=OPTIONAL"`
	TakerBuyQuoteVolume *float64 `parquet:"name=taker_buy_quote_volume, type=DOUBLE, repetitiontype=OPTIONAL"`
}

func (s *LocalParquetSource) LoadCandles(ctx context.Context, req CandleRequest) ([]protocol.Candle, error) {
	candles, _, err := s.LoadCandlesWithInventory(ctx, req)
	return candles, err
}

// LoadCandlesWithInventory returns the exact regular parquet paths opened to
// produce the result. The inventory is stable, absolute, and suitable for
// independent identity verification by the caller.
func (s *LocalParquetSource) LoadCandlesWithInventory(ctx context.Context, req CandleRequest) ([]protocol.Candle, []string, error) {
	if req.Path == "" {
		return nil, nil, fmt.Errorf("empty path")
	}

	pattern1 := filepath.Join(req.Path, "candles", req.Market, req.Interval, "symbol="+req.Symbol, "year=*", "month=*", "*.parquet")
	pattern2 := filepath.Join(req.Path, req.Market, req.Interval, req.Symbol, "monthly", "*", "*.parquet")
	matches1, _ := filepath.Glob(pattern1)
	matches2, _ := filepath.Glob(pattern2)

	uniqueMatches := make(map[string]bool)
	var matches []string

	for _, m := range append(matches1, matches2...) {
		absolute, err := filepath.Abs(m)
		if err != nil {
			return nil, nil, fmt.Errorf("resolve parquet path %s: %w", m, err)
		}
		if !uniqueMatches[absolute] {
			uniqueMatches[absolute] = true
			matches = append(matches, absolute)
		}
	}
	sort.Strings(matches)

	if len(matches) == 0 {
		return nil, nil, fmt.Errorf("no matching files found under path: %s (tried %s and %s)", req.Path, pattern1, pattern2)
	}

	// Filter files by date range
	var candidates []string
	for _, match := range matches {
		start, end, err := ParseDateRangeFromFilename(match)
		if err != nil {
			// Skip files with invalid naming structure
			continue
		}
		if !req.From.IsZero() && end.Before(req.From) {
			continue
		}
		if !req.To.IsZero() && start.After(req.To) {
			continue
		}
		candidates = append(candidates, match)
	}

	if len(candidates) == 0 {
		return nil, nil, fmt.Errorf("no matching files in range")
	}
	sort.Strings(candidates)

	filtered, err := LoadExactParquetFiles(ctx, req, candidates)
	if err != nil {
		return nil, nil, err
	}

	return filtered, append([]string(nil), candidates...), nil
}

// LoadExactParquetFiles independently reconstructs the candle rows from the
// exact object paths supplied by an identity request. It rejects symlinks,
// duplicate paths, empty input, and any row-validation defect.
func LoadExactParquetFiles(ctx context.Context, req CandleRequest, paths []string) ([]protocol.Candle, error) {
	if len(paths) == 0 {
		return nil, fmt.Errorf("exact parquet file inventory is empty")
	}
	ordered := append([]string(nil), paths...)
	sort.Strings(ordered)
	var allCandles []protocol.Candle
	for i, file := range ordered {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
		}
		if i > 0 && ordered[i-1] == file {
			return nil, fmt.Errorf("duplicate exact parquet path %s", file)
		}
		info, err := os.Lstat(file)
		if err != nil {
			return nil, fmt.Errorf("inspect parquet file %s: %w", file, err)
		}
		if !info.Mode().IsRegular() {
			return nil, fmt.Errorf("parquet file %s is not a regular file", file)
		}
		candles, err := readParquetFile(file, req)
		if err != nil {
			return nil, fmt.Errorf("unreadable parquet file %s: %w", file, err)
		}
		allCandles = append(allCandles, candles...)
	}
	if len(allCandles) == 0 {
		return nil, fmt.Errorf("empty candle result")
	}
	sort.Slice(allCandles, func(i, j int) bool { return allCandles[i].OpenTimeMS < allCandles[j].OpenTimeMS })
	filtered := make([]protocol.Candle, 0, len(allCandles))
	for _, candle := range allCandles {
		if !req.From.IsZero() && candle.OpenTimeMS < req.From.UnixMilli() {
			continue
		}
		if !req.To.IsZero() && candle.OpenTimeMS > req.To.UnixMilli() {
			continue
		}
		filtered = append(filtered, candle)
	}
	if len(filtered) == 0 {
		return nil, fmt.Errorf("empty candle result")
	}
	if err := ValidateCandles(req.Interval, filtered); err != nil {
		return nil, err
	}
	return filtered, nil
}

func checkHasMS(path string) (bool, error) {
	fr, err := local.NewLocalFileReader(path)
	if err != nil {
		return false, err
	}
	defer fr.Close()

	pr, err := reader.NewParquetReader(fr, new(CheckMS), 1)
	if err != nil {
		return false, nil
	}
	defer pr.ReadStop()
	if pr.GetNumRows() == 0 {
		return false, nil
	}
	rows := make([]CheckMS, 1)
	if err := pr.Read(&rows); err != nil {
		return false, nil
	}
	return rows[0].OpenTimeMS != nil, nil
}

func readParquetFile(path string, req CandleRequest) (candles []protocol.Candle, err error) {
	defer func() {
		if r := recover(); r != nil {
			err = fmt.Errorf("panic during parquet read: %v", r)
		}
	}()

	hasMS, err := checkHasMS(path)
	if err != nil {
		return nil, fmt.Errorf("check schema failed: %w", err)
	}

	fr, err := local.NewLocalFileReader(path)
	if err != nil {
		return nil, fmt.Errorf("open file failed: %w", err)
	}
	defer fr.Close()

	if hasMS {
		pr, err := reader.NewParquetReader(fr, new(ParquetCandleWithMS), 4)
		if err != nil {
			return nil, fmt.Errorf("create reader failed: %w", err)
		}
		defer pr.ReadStop()

		numRows := int(pr.GetNumRows())
		raw := make([]ParquetCandleWithMS, numRows)
		if err := pr.Read(&raw); err != nil {
			return nil, fmt.Errorf("read rows failed: %w", err)
		}

		res := make([]protocol.Candle, 0, len(raw))
		for i, pc := range raw {
			if pc.OpenTimeMS == nil {
				return nil, fmt.Errorf("open_time_ms is nil at index %d", i)
			}
			if pc.CloseTimeMS == nil {
				return nil, fmt.Errorf("close_time_ms is nil at index %d", i)
			}
			if pc.Open == nil || pc.High == nil || pc.Low == nil || pc.Close == nil || pc.Volume == nil {
				return nil, fmt.Errorf("missing OHLCV at index %d", i)
			}

			var c protocol.Candle
			c.OpenTimeMS = *pc.OpenTimeMS
			c.CloseTimeMS = *pc.CloseTimeMS

			// Validate timestamp units
			if c.OpenTimeMS < 1000000000000 || c.OpenTimeMS > 9999999999999 {
				return nil, fmt.Errorf("open_time_ms %d at index %d is not in milliseconds", c.OpenTimeMS, i)
			}
			if c.CloseTimeMS < 1000000000000 || c.CloseTimeMS > 9999999999999 {
				return nil, fmt.Errorf("close_time_ms %d at index %d is not in milliseconds", c.CloseTimeMS, i)
			}

			c.Open = *pc.Open
			c.High = *pc.High
			c.Low = *pc.Low
			c.Close = *pc.Close
			c.Volume = *pc.Volume

			if pc.QuoteAssetVolume != nil {
				c.QuoteAssetVolume = *pc.QuoteAssetVolume
			}
			if pc.NumberOfTrades != nil {
				c.NumberOfTrades = *pc.NumberOfTrades
			}
			if pc.TakerBuyBaseVolume != nil {
				c.TakerBuyBaseVolume = *pc.TakerBuyBaseVolume
			}
			if pc.TakerBuyQuoteVolume != nil {
				c.TakerBuyQuoteVolume = *pc.TakerBuyQuoteVolume
			}

			if pc.Market != nil && *pc.Market != "" {
				c.Market = *pc.Market
			} else {
				c.Market = req.Market
			}

			if pc.Symbol != nil && *pc.Symbol != "" {
				c.Symbol = *pc.Symbol
			} else {
				c.Symbol = req.Symbol
			}

			if pc.Interval != nil && *pc.Interval != "" {
				c.Interval = *pc.Interval
			} else {
				c.Interval = req.Interval
			}

			res = append(res, c)
		}
		return res, nil
	} else {
		pr, err := reader.NewParquetReader(fr, new(ParquetCandleWithoutMS), 4)
		if err != nil {
			return nil, fmt.Errorf("create reader failed: %w", err)
		}
		defer pr.ReadStop()

		numRows := int(pr.GetNumRows())
		raw := make([]ParquetCandleWithoutMS, numRows)
		if err := pr.Read(&raw); err != nil {
			return nil, fmt.Errorf("read rows failed: %w", err)
		}

		res := make([]protocol.Candle, 0, len(raw))
		for i, pc := range raw {
			if pc.OpenTime == nil {
				return nil, fmt.Errorf("open_time is nil at index %d", i)
			}
			if pc.CloseTime == nil {
				return nil, fmt.Errorf("close_time is nil at index %d", i)
			}
			if pc.Open == nil || pc.High == nil || pc.Low == nil || pc.Close == nil || pc.Volume == nil {
				return nil, fmt.Errorf("missing OHLCV at index %d", i)
			}

			var c protocol.Candle
			c.OpenTimeMS = *pc.OpenTime
			c.CloseTimeMS = *pc.CloseTime

			// Validate timestamp units
			if c.OpenTimeMS < 1000000000000 || c.OpenTimeMS > 9999999999999 {
				return nil, fmt.Errorf("open_time %d at index %d is not in milliseconds", c.OpenTimeMS, i)
			}
			if c.CloseTimeMS < 1000000000000 || c.CloseTimeMS > 9999999999999 {
				return nil, fmt.Errorf("close_time %d at index %d is not in milliseconds", c.CloseTimeMS, i)
			}

			c.Open = *pc.Open
			c.High = *pc.High
			c.Low = *pc.Low
			c.Close = *pc.Close
			c.Volume = *pc.Volume

			if pc.QuoteAssetVolume != nil {
				c.QuoteAssetVolume = *pc.QuoteAssetVolume
			}
			if pc.NumberOfTrades != nil {
				c.NumberOfTrades = *pc.NumberOfTrades
			}
			if pc.TakerBuyBaseVolume != nil {
				c.TakerBuyBaseVolume = *pc.TakerBuyBaseVolume
			}
			if pc.TakerBuyQuoteVolume != nil {
				c.TakerBuyQuoteVolume = *pc.TakerBuyQuoteVolume
			}

			if pc.Market != nil && *pc.Market != "" {
				c.Market = *pc.Market
			} else {
				c.Market = req.Market
			}

			if pc.Symbol != nil && *pc.Symbol != "" {
				c.Symbol = *pc.Symbol
			} else {
				c.Symbol = req.Symbol
			}

			if pc.Interval != nil && *pc.Interval != "" {
				c.Interval = *pc.Interval
			} else {
				c.Interval = req.Interval
			}

			res = append(res, c)
		}
		return res, nil
	}
}
