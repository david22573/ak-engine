package data

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"

	"github.com/davidmiguel22573/ak-engine/pkg/protocol"
)

type LocalJSONSource struct{}

func NewLocalJSONSource() *LocalJSONSource {
	return &LocalJSONSource{}
}

func (s *LocalJSONSource) Name() string {
	return "local-json"
}

func (s *LocalJSONSource) LoadCandles(ctx context.Context, req CandleRequest) ([]protocol.Candle, error) {
	if req.Path == "" {
		return nil, fmt.Errorf("missing path in candle request")
	}

	file, err := os.Open(req.Path)
	if err != nil {
		return nil, fmt.Errorf("failed to open file %s: %w", req.Path, err)
	}
	defer file.Close()

	data, err := io.ReadAll(file)
	if err != nil {
		return nil, fmt.Errorf("failed to read file: %w", err)
	}

	return ParseJSONCandles(data, req)
}

func ParseJSONCandles(data []byte, req CandleRequest) ([]protocol.Candle, error) {
	candles, err := ParseJSONCandlesNoValidate(data, req)
	if err != nil {
		return nil, err
	}

	if err := ValidateCandles(req.Interval, candles); err != nil {
		return nil, err
	}

	return candles, nil
}

func ParseJSONCandlesNoValidate(data []byte, req CandleRequest) ([]protocol.Candle, error) {
	var candles []protocol.Candle

	trimmed := strings.TrimSpace(string(data))
	if len(trimmed) == 0 {
		return nil, fmt.Errorf("empty JSON data")
	}

	if !strings.HasPrefix(trimmed, "[") {
		return nil, fmt.Errorf("malformed JSON: must be a JSON array")
	}

	// Try standard []protocol.Candle first
	err := json.Unmarshal(data, &candles)
	if err == nil && len(candles) > 0 && candles[0].OpenTimeMS != 0 {
		// Loaded successfully as []protocol.Candle!
		// Normalize missing fields
		for i := range candles {
			if candles[i].Symbol == "" {
				candles[i].Symbol = req.Symbol
			}
			if candles[i].Market == "" {
				candles[i].Market = req.Market
			}
			if candles[i].Interval == "" {
				candles[i].Interval = req.Interval
			}
		}

		// Filter by from and to
		filtered := filterCandles(candles, req)
		return filtered, nil
	}

	// If that failed or parsed to zero/empty, try parsing as Binance kline array: [][]interface{}
	var rawArray [][]interface{}
	if err := json.Unmarshal(data, &rawArray); err != nil {
		return nil, fmt.Errorf("malformed JSON or unsupported shape: %w", err)
	}

	if len(rawArray) == 0 {
		return nil, fmt.Errorf("empty result")
	}

	candles = make([]protocol.Candle, 0, len(rawArray))
	for idx, item := range rawArray {
		if len(item) < 11 {
			return nil, fmt.Errorf("unsupported shape: kline at index %d has only %d elements (minimum 11 expected)", idx, len(item))
		}

		var c protocol.Candle
		openTimeVal, err := toInt64(item[0])
		if err != nil {
			return nil, fmt.Errorf("unsupported shape at index %d: invalid open_time: %w", idx, err)
		}
		c.OpenTimeMS = openTimeVal

		openVal, err := toFloat64(item[1])
		if err != nil {
			return nil, fmt.Errorf("unsupported shape at index %d: invalid open: %w", idx, err)
		}
		c.Open = openVal

		highVal, err := toFloat64(item[2])
		if err != nil {
			return nil, fmt.Errorf("unsupported shape at index %d: invalid high: %w", idx, err)
		}
		c.High = highVal

		lowVal, err := toFloat64(item[3])
		if err != nil {
			return nil, fmt.Errorf("unsupported shape at index %d: invalid low: %w", idx, err)
		}
		c.Low = lowVal

		closeVal, err := toFloat64(item[4])
		if err != nil {
			return nil, fmt.Errorf("unsupported shape at index %d: invalid close: %w", idx, err)
		}
		c.Close = closeVal

		volumeVal, err := toFloat64(item[5])
		if err != nil {
			return nil, fmt.Errorf("unsupported shape at index %d: invalid volume: %w", idx, err)
		}
		c.Volume = volumeVal

		closeTimeVal, err := toInt64(item[6])
		if err != nil {
			return nil, fmt.Errorf("unsupported shape at index %d: invalid close_time: %w", idx, err)
		}
		c.CloseTimeMS = closeTimeVal

		quoteVolVal, err := toFloat64(item[7])
		if err != nil {
			return nil, fmt.Errorf("unsupported shape at index %d: invalid quote_asset_volume: %w", idx, err)
		}
		c.QuoteAssetVolume = quoteVolVal

		tradesVal, err := toInt64(item[8])
		if err != nil {
			return nil, fmt.Errorf("unsupported shape at index %d: invalid number_of_trades: %w", idx, err)
		}
		c.NumberOfTrades = tradesVal

		takerBaseVal, err := toFloat64(item[9])
		if err != nil {
			return nil, fmt.Errorf("unsupported shape at index %d: invalid taker_buy_base_volume: %w", idx, err)
		}
		c.TakerBuyBaseVolume = takerBaseVal

		takerQuoteVal, err := toFloat64(item[10])
		if err != nil {
			return nil, fmt.Errorf("unsupported shape at index %d: invalid taker_buy_quote_volume: %w", idx, err)
		}
		c.TakerBuyQuoteVolume = takerQuoteVal

		c.Symbol = req.Symbol
		c.Market = req.Market
		c.Interval = req.Interval

		candles = append(candles, c)
	}

	filtered := filterCandles(candles, req)
	return filtered, nil
}

func filterCandles(candles []protocol.Candle, req CandleRequest) []protocol.Candle {
	var filtered []protocol.Candle
	for _, c := range candles {
		if !req.From.IsZero() {
			if c.OpenTimeMS < req.From.UnixMilli() {
				continue
			}
		}
		if !req.To.IsZero() {
			if c.OpenTimeMS > req.To.UnixMilli() {
				continue
			}
		}
		filtered = append(filtered, c)
	}
	return filtered
}

func toFloat64(val interface{}) (float64, error) {
	switch v := val.(type) {
	case float64:
		return v, nil
	case string:
		f, err := strconv.ParseFloat(v, 64)
		if err != nil {
			return 0, fmt.Errorf("cannot parse string %q as float64: %w", v, err)
		}
		return f, nil
	default:
		return 0, fmt.Errorf("unsupported type %T for float64 conversion", val)
	}
}

func toInt64(val interface{}) (int64, error) {
	switch v := val.(type) {
	case float64:
		return int64(v), nil
	case string:
		i, err := strconv.ParseInt(v, 10, 64)
		if err != nil {
			return 0, fmt.Errorf("cannot parse string %q as int64: %w", v, err)
		}
		return i, nil
	default:
		return 0, fmt.Errorf("unsupported type %T for int64 conversion", val)
	}
}
