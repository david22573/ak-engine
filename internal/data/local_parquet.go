package data

import (
	"context"
	"fmt"
	"path/filepath"
	"sort"

	"github.com/davidmiguel22573/ak-engine/pkg/protocol"
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
	if req.Path == "" {
		return nil, fmt.Errorf("empty path")
	}

	pattern := filepath.Join(req.Path, "candles", req.Market, req.Interval, "symbol="+req.Symbol, "year=*", "month=*", "*.parquet")
	matches, err := filepath.Glob(pattern)
	if err != nil {
		return nil, fmt.Errorf("glob pattern failed: %w", err)
	}

	if len(matches) == 0 {
		return nil, fmt.Errorf("no matching files found under path: %s", req.Path)
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
		return nil, fmt.Errorf("no matching files in range")
	}

	var allCandles []protocol.Candle
	for _, file := range candidates {
		candles, err := readParquetFile(file, req)
		if err != nil {
			return nil, fmt.Errorf("unreadable parquet file %s: %w", file, err)
		}
		allCandles = append(allCandles, candles...)
	}

	if len(allCandles) == 0 {
		return nil, fmt.Errorf("empty candle result")
	}

	// Sort candles by OpenTimeMS before validating
	sort.Slice(allCandles, func(i, j int) bool {
		return allCandles[i].OpenTimeMS < allCandles[j].OpenTimeMS
	})

	// Filter candles inside the files by exact requested From/To
	var filtered []protocol.Candle
	for _, c := range allCandles {
		if !req.From.IsZero() && c.OpenTimeMS < req.From.UnixMilli() {
			continue
		}
		if !req.To.IsZero() && c.OpenTimeMS > req.To.UnixMilli() {
			continue
		}
		filtered = append(filtered, c)
	}

	if len(filtered) == 0 {
		return nil, fmt.Errorf("empty candle result")
	}

	// Validate using existing ValidateCandles
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
