package features

func EMA(values []float64, period int) []float64 {
	result := make([]float64, len(values))
	if len(values) < period || period <= 0 {
		return result
	}

	k := 2.0 / float64(period+1)

	// Seed with SMA of first period
	sum := 0.0
	for i := 0; i < period; i++ {
		sum += values[i]
	}
	result[period-1] = sum / float64(period)

	for i := period; i < len(values); i++ {
		result[i] = values[i]*k + result[i-1]*(1.0-k)
	}

	return result
}

func LinearSlope(values []float64, idx int, period int) (float64, bool) {
	if period <= 1 || idx-period+1 < 0 || idx >= len(values) {
		return 0, false
	}

	sumX := 0.0
	sumY := 0.0
	sumX2 := 0.0
	sumXY := 0.0

	for i := 0; i < period; i++ {
		x := float64(i)
		y := values[idx-period+1+i]
		if y <= 0 {
			// Ignore invalid zero/negative close values
			return 0, false
		}
		sumX += x
		sumY += y
		sumX2 += x * x
		sumXY += x * y
	}

	n := float64(period)
	num := n*sumXY - sumX*sumY
	den := n*sumX2 - sumX*sumX

	if den == 0 {
		return 0, false
	}

	return num / den, true
}
