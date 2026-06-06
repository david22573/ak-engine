package features

import (
	"math"

	"github.com/davidmiguel22573/ak-engine/pkg/protocol"
)

func ATR(candles []protocol.Candle, idx int, period int) (float64, bool) {
	if idx - period < 0 || idx >= len(candles) {
		return 0, false
	}

	trSum := 0.0
	for i := idx - period + 1; i <= idx; i++ {
		c := candles[i]
		prevC := candles[i-1]
		if c.Close <= 0 || prevC.Close <= 0 || c.High < c.Low {
			return 0, false
		}
		tr := c.High - c.Low
		if d1 := math.Abs(c.High - prevC.Close); d1 > tr {
			tr = d1
		}
		if d2 := math.Abs(c.Low - prevC.Close); d2 > tr {
			tr = d2
		}
		trSum += tr
	}
	return trSum / float64(period), true
}

func RealizedVol(closes []float64, idx int, period int) (float64, bool) {
	if idx - period < 0 || idx >= len(closes) {
		return 0, false
	}

	returns := make([]float64, period)
	sum := 0.0
	for k := 0; k < period; k++ {
		currIdx := idx - period + 1 + k
		prevIdx := currIdx - 1
		prev := closes[prevIdx]
		curr := closes[currIdx]
		if prev <= 0 || curr <= 0 {
			return 0, false
		}
		ret := (curr - prev) / prev
		returns[k] = ret
		sum += ret
	}

	mean := sum / float64(period)
	variance := 0.0
	for _, r := range returns {
		diff := r - mean
		variance += diff * diff
	}
	return math.Sqrt(variance / float64(period)), true
}

func BBWidth(closes []float64, idx int, period int, mult float64) (float64, bool) {
	if idx - period + 1 < 0 || idx >= len(closes) {
		return 0, false
	}

	sum := 0.0
	for i := idx - period + 1; i <= idx; i++ {
		val := closes[i]
		if val <= 0 {
			return 0, false
		}
		sum += val
	}
	midBB := sum / float64(period)
	if midBB <= 0 {
		return 0, false
	}

	variance := 0.0
	for i := idx - period + 1; i <= idx; i++ {
		diff := closes[i] - midBB
		variance += diff * diff
	}
	stdDev := math.Sqrt(variance / float64(period))

	return (2.0 * mult * stdDev) / midBB, true
}

func PercentRankTrailing(values []float64, idx int, lookback int) (float64, bool) {
	if idx - lookback < 0 || idx >= len(values) {
		return 0, false
	}

	current := values[idx]
	count := 0
	for i := idx - lookback; i < idx; i++ {
		if values[i] <= current {
			count++
		}
	}
	return float64(count) / float64(lookback), true
}
