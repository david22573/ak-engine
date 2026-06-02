package data

import (
	"fmt"
	"sort"
	"strconv"

	"github.com/davidmiguel22573/ak-engine/pkg/protocol"
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

// ValidateCandles validates a slice of candles according to roadmap requirements.
// It sorts the candles by OpenTimeMS first.
func ValidateCandles(interval string, candles []protocol.Candle) error {
	if len(candles) == 0 {
		return fmt.Errorf("reject empty dataset")
	}

	// 1. Sort by OpenTimeMS
	sort.Slice(candles, func(i, j int) bool {
		return candles[i].OpenTimeMS < candles[j].OpenTimeMS
	})

	expectedDuration, err := ParseIntervalToMS(interval)
	if err != nil {
		return fmt.Errorf("failed to parse interval: %w", err)
	}

	for i := 0; i < len(candles); i++ {
		c := candles[i]
		// 2. Reject malformed OHLC
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
			// 3. Reject duplicate OpenTimeMS
			if c.OpenTimeMS == prev.OpenTimeMS {
				return fmt.Errorf("duplicate OpenTimeMS detected: %d", c.OpenTimeMS)
			}
			// 4. Reject gaps for fixed intervals
			if c.OpenTimeMS-prev.OpenTimeMS > expectedDuration {
				return fmt.Errorf("gap detected: OpenTimeMS jumped from %d to %d (expected step %d)", prev.OpenTimeMS, c.OpenTimeMS, expectedDuration)
			}
			// Additional safety: if order is somehow reversed, fail
			if c.OpenTimeMS < prev.OpenTimeMS {
				return fmt.Errorf("out of order OpenTimeMS detected: prev %d, current %d", prev.OpenTimeMS, c.OpenTimeMS)
			}
		}
	}

	return nil
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
