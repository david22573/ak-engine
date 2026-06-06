package features

func RatioToSMA(values []float64, idx int, period int) (float64, bool) {
	if period <= 0 || idx-period+1 < 0 || idx >= len(values) {
		return 0, false
	}

	sum := 0.0
	for i := idx - period + 1; i <= idx; i++ {
		sum += values[i]
	}
	sma := sum / float64(period)
	if sma <= 0 {
		return 0, false
	}

	return SafeRatio(values[idx], sma), true
}

func SafeRatio(num, den float64) float64 {
	if den == 0 {
		return 0
	}
	return num / den
}
