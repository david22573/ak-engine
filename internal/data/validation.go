package data

import (
	"fmt"
	"math"
	"sort"
	"strconv"

	"github.com/david22573/ak-engine/pkg/protocol"
)

type CandleAnalysis struct {
	Count      int   `json:"count"`
	FirstMS    int64 `json:"first_ms"`
	LastMS     int64 `json:"last_ms"`
	Duplicates int   `json:"duplicates"`
	Gaps       int   `json:"gaps"`
}

// ParseIntervalToMS converts interval string (e.g., "1m", "5m", "1h", "1d") to milliseconds
func ParseIntervalToMS(interval string) (int64, error) {
	if len(interval) < 2 {
		return 0, fmt.Errorf("invalid interval format: %s", interval)
	}
	unit := interval[len(interval)-1:]
	valStr := interval[:len(interval)-1]
	val, err := strconv.ParseInt(valStr, 10, 64)
	if err != nil {
		return 0, fmt.Errorf("invalid interval value: %s, err: %v", valStr, err)
	}
	switch unit {
	case "m":
		return val * 60 * 1000, nil
	case "h":
		return val * 60 * 60 * 1000, nil
	case "d":
		return val * 24 * 60 * 60 * 1000, nil
	case "w":
		return val * 7 * 24 * 60 * 60 * 1000, nil
	default:
		return 0, fmt.Errorf("unsupported interval unit: %s", unit)
	}
}

// ValidateCandles validates candles in their original order. Validation never
// normalizes input because doing so would hide source disorder.
func ValidateCandles(interval string, candles []protocol.Candle) error {
	if len(candles) == 0 {
		return fmt.Errorf("reject empty dataset")
	}

	expectedDuration, err := ParseIntervalToMS(interval)
	if err != nil {
		return fmt.Errorf("failed to parse interval: %w", err)
	}

	for i := 0; i < len(candles); i++ {
		c := candles[i]
		if c.OpenTimeMS <= 0 {
			return fmt.Errorf("invalid open time at index %d: %d", i, c.OpenTimeMS)
		}
		if c.CloseTimeMS != c.OpenTimeMS+expectedDuration-1 {
			return fmt.Errorf("invalid close time at index %d: got %d want %d", i, c.CloseTimeMS, c.OpenTimeMS+expectedDuration-1)
		}
		if c.Interval != "" && c.Interval != interval {
			return fmt.Errorf("interval mismatch at index %d: got %q want %q", i, c.Interval, interval)
		}
		if !finitePositive(c.Open) || !finitePositive(c.High) || !finitePositive(c.Low) || !finitePositive(c.Close) {
			return fmt.Errorf("non-finite or nonpositive OHLC at index %d", i)
		}
		if !finiteNonnegative(c.Volume) || !finiteNonnegative(c.QuoteAssetVolume) || !finiteNonnegative(c.TakerBuyBaseVolume) || !finiteNonnegative(c.TakerBuyQuoteVolume) {
			return fmt.Errorf("non-finite or negative volume at index %d", i)
		}
		// Reject malformed OHLC.
		if c.High < c.Low {
			return fmt.Errorf("malformed OHLC at index %d: High (%f) < Low (%f)", i, c.High, c.Low)
		}
		if c.Open < c.Low || c.Open > c.High {
			return fmt.Errorf("malformed OHLC at index %d: Open (%f) outside High/Low range [%f, %f]", i, c.Open, c.Low, c.High)
		}
		if c.Close < c.Low || c.Close > c.High {
			return fmt.Errorf("malformed OHLC at index %d: Close (%f) outside High/Low range [%f, %f]", i, c.Close, c.Low, c.High)
		}

		if i > 0 {
			prev := candles[i-1]
			// Reject duplicate OpenTimeMS.
			if c.OpenTimeMS == prev.OpenTimeMS {
				return fmt.Errorf("duplicate OpenTimeMS detected: %d", c.OpenTimeMS)
			}
			if c.OpenTimeMS < prev.OpenTimeMS {
				return fmt.Errorf("out of order OpenTimeMS detected: prev %d, current %d", prev.OpenTimeMS, c.OpenTimeMS)
			}
			if c.OpenTimeMS-prev.OpenTimeMS != expectedDuration {
				return fmt.Errorf("cadence mismatch: OpenTimeMS jumped from %d to %d (expected exact step %d)", prev.OpenTimeMS, c.OpenTimeMS, expectedDuration)
			}
		}
	}

	return nil
}

func ValidateCandlesForRequest(req CandleRequest, candles []protocol.Candle) error {
	if err := ValidateCandles(req.Interval, candles); err != nil {
		return err
	}
	for i, candle := range candles {
		if candle.Market != req.Market {
			return fmt.Errorf("market mismatch at index %d: got %q want %q", i, candle.Market, req.Market)
		}
		if candle.Symbol != req.Symbol {
			return fmt.Errorf("symbol mismatch at index %d: got %q want %q", i, candle.Symbol, req.Symbol)
		}
		if candle.Interval != req.Interval {
			return fmt.Errorf("interval mismatch at index %d: got %q want %q", i, candle.Interval, req.Interval)
		}
	}
	return nil
}

func finitePositive(value float64) bool {
	return value > 0 && !math.IsNaN(value) && !math.IsInf(value, 0)
}

func finiteNonnegative(value float64) bool {
	return value >= 0 && !math.IsNaN(value) && !math.IsInf(value, 0)
}

// AnalyzeCandles analyzes a slice of candles and returns metrics.
// It sorts the candles by OpenTimeMS first.
func AnalyzeCandles(interval string, candles []protocol.Candle) CandleAnalysis {
	analysis := CandleAnalysis{
		Count: len(candles),
	}
	if len(candles) == 0 {
		return analysis
	}

	// Sort a copy or in-place as per expectations
	sort.Slice(candles, func(i, j int) bool {
		return candles[i].OpenTimeMS < candles[j].OpenTimeMS
	})

	analysis.FirstMS = candles[0].OpenTimeMS
	analysis.LastMS = candles[len(candles)-1].OpenTimeMS

	expectedDuration, err := ParseIntervalToMS(interval)
	if err != nil {
		return analysis
	}

	for i := 1; i < len(candles); i++ {
		prev := candles[i-1]
		curr := candles[i]

		if curr.OpenTimeMS == prev.OpenTimeMS {
			analysis.Duplicates++
		} else if curr.OpenTimeMS-prev.OpenTimeMS > expectedDuration {
			missing := (curr.OpenTimeMS-prev.OpenTimeMS)/expectedDuration - 1
			if missing > 0 {
				analysis.Gaps += int(missing)
			}
		}
	}

	return analysis
}
