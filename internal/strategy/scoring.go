package strategy

import (
	"math"

	"github.com/davidmiguel22573/ak-engine/pkg/protocol"
)

type ScoreInput struct {
	Window               AggregatedWindow
	Previous             []AggregatedWindow
	RecentCandles        []protocol.Candle
	HourContext          *AggregatedWindow
	EstimatedCostBPS     float64
	CostMultipleRequired float64
	MaxChopScore         float64
}

type ScoreResult struct {
	LongScore        float64
	ShortScore       float64
	ChopScore        float64
	VolatilityScore  float64
	TrendScore       float64
	PullbackScore    float64
	BreakoutScore    float64
	ExpectedMoveBPS  float64
	EstimatedCostBPS float64
	Confidence       float64
	DataFreshness    DataFreshness
	HardBlock        bool
	ReasonCodes      []string
}

func ScoreWindow(input ScoreInput) ScoreResult {
	result := ScoreResult{
		EstimatedCostBPS: input.EstimatedCostBPS,
		DataFreshness:    DataFreshnessPass,
	}

	if len(input.Previous) < 1 {
		result.HardBlock = true
		result.ReasonCodes = appendReason(result.ReasonCodes, "INSUFFICIENT_DATA")
		return result
	}

	current := input.Window
	prev := input.Previous[len(input.Previous)-1]
	bodyStrength := bodyStrength(current)
	rangeBPS := current.RangeBPS()
	momentumBPS := 0.0
	if prev.Close > 0 {
		momentumBPS = ((current.Close - prev.Close) / prev.Close) * 10000
	}

	longTrend := 0.0
	shortTrend := 0.0
	switch current.Direction() {
	case SideLong:
		longTrend += 30 + bodyStrength*20
		result.ReasonCodes = appendReason(result.ReasonCodes, "15M_BULLISH")
	case SideShort:
		shortTrend += 30 + bodyStrength*20
		result.ReasonCodes = appendReason(result.ReasonCodes, "15M_BEARISH")
	default:
		result.ReasonCodes = appendReason(result.ReasonCodes, "15M_CHOP")
	}
	if momentumBPS > 0 {
		longTrend += clampFloat(momentumBPS*1.2, 0, 25)
	} else if momentumBPS < 0 {
		shortTrend += clampFloat(math.Abs(momentumBPS)*1.2, 0, 25)
	}

	hourBiasLong, hourBiasShort := 0.0, 0.0
	if input.HourContext != nil && input.HourContext.Open > 0 {
		hourMomentum := ((input.HourContext.Close - input.HourContext.Open) / input.HourContext.Open) * 10000
		if hourMomentum > 0 {
			hourBiasLong = clampFloat(hourMomentum*0.3, 0, 15)
		} else if hourMomentum < 0 {
			hourBiasShort = clampFloat(math.Abs(hourMomentum)*0.3, 0, 15)
		}
	}

	overlap := overlapRatio(current, prev)
	result.ChopScore = clampFloat((overlap*70)+((1-bodyStrength)*30), 0, 100)
	result.VolatilityScore = clampFloat((rangeBPS/30)*100, 0, 100)
	if result.VolatilityScore < 30 {
		result.ReasonCodes = appendReason(result.ReasonCodes, "VOLATILITY_TOO_LOW")
	} else {
		result.ReasonCodes = appendReason(result.ReasonCodes, "VOLATILITY_OK")
	}
	if result.VolatilityScore > 90 {
		result.ReasonCodes = appendReason(result.ReasonCodes, "VOLATILITY_TOO_HIGH")
	}

	longPullback, shortPullback := pullbackScores(current, input.RecentCandles)
	result.PullbackScore = math.Max(longPullback, shortPullback)
	longBreakout, shortBreakout := breakoutScores(current, prev)
	result.BreakoutScore = math.Max(longBreakout, shortBreakout)

	result.TrendScore = clampFloat(math.Max(longTrend+hourBiasLong, shortTrend+hourBiasShort), 0, 100)
	chopPenalty := result.ChopScore * 0.35
	result.LongScore = clampFloat((longTrend+hourBiasLong)*0.5+result.VolatilityScore*0.15+longPullback*0.15+longBreakout*0.2-chopPenalty, 0, 100)
	result.ShortScore = clampFloat((shortTrend+hourBiasShort)*0.5+result.VolatilityScore*0.15+shortPullback*0.15+shortBreakout*0.2-chopPenalty, 0, 100)
	result.ExpectedMoveBPS = clampFloat(rangeBPS*(0.45+bodyStrength*0.35)+math.Abs(momentumBPS)*0.35, 0, 500)
	result.Confidence = clampFloat(math.Abs(result.LongScore-result.ShortScore)+math.Max(result.LongScore, result.ShortScore)*0.25, 0, 100)

	switch {
	case result.LongScore > result.ShortScore:
		if longPullback >= 45 {
			result.ReasonCodes = appendReason(result.ReasonCodes, "5M_PULLBACK_RECLAIM")
		}
	case result.ShortScore > result.LongScore:
		if shortPullback >= 45 {
			result.ReasonCodes = appendReason(result.ReasonCodes, "5M_PULLBACK_REJECT")
		}
	default:
		result.ReasonCodes = appendReason(result.ReasonCodes, "NO_EDGE")
	}

	if input.EstimatedCostBPS <= 0 {
		result.HardBlock = true
		result.ReasonCodes = appendReason(result.ReasonCodes, "INVALID_COST")
	}
	if result.ExpectedMoveBPS < input.EstimatedCostBPS*input.CostMultipleRequired {
		result.HardBlock = true
		result.ReasonCodes = appendReason(result.ReasonCodes, "EXPECTED_MOVE_BELOW_COST")
	}
	if result.ChopScore >= input.MaxChopScore {
		result.HardBlock = true
		result.ReasonCodes = appendReason(result.ReasonCodes, "15M_CHOP")
	}
	if math.Max(result.LongScore, result.ShortScore) < 40 {
		result.ReasonCodes = appendReason(result.ReasonCodes, "NO_EDGE")
	}

	return result
}

func bodyStrength(window AggregatedWindow) float64 {
	rng := window.High - window.Low
	if rng <= 0 {
		return 0
	}
	return clampFloat(math.Abs(window.Close-window.Open)/rng, 0, 1)
}

func overlapRatio(current, prev AggregatedWindow) float64 {
	overlapHigh := math.Min(current.High, prev.High)
	overlapLow := math.Max(current.Low, prev.Low)
	if overlapHigh <= overlapLow {
		return 0
	}
	base := math.Max(current.High, prev.High) - math.Min(current.Low, prev.Low)
	if base <= 0 {
		return 1
	}
	return clampFloat((overlapHigh-overlapLow)/base, 0, 1)
}

func pullbackScores(window AggregatedWindow, recent []protocol.Candle) (float64, float64) {
	if len(recent) == 0 {
		return 0, 0
	}
	last := recent[len(recent)-1]
	rng := window.High - window.Low
	if rng <= 0 {
		return 0, 0
	}
	upperClose := clampFloat((window.Close-window.Low)/rng, 0, 1)
	lowerClose := clampFloat((window.High-window.Close)/rng, 0, 1)
	longScore := 0.0
	shortScore := 0.0
	if window.Close >= last.Close {
		longScore = clampFloat(upperClose*100, 0, 100)
	}
	if window.Close <= last.Close {
		shortScore = clampFloat(lowerClose*100, 0, 100)
	}
	return longScore, shortScore
}

func breakoutScores(current, prev AggregatedWindow) (float64, float64) {
	longScore := 0.0
	shortScore := 0.0
	if current.Close > prev.High {
		longScore = clampFloat(((current.Close-prev.High)/prev.High)*10000, 0, 100)
	}
	if current.Close < prev.Low {
		shortScore = clampFloat(((prev.Low-current.Close)/prev.Low)*10000, 0, 100)
	}
	return longScore, shortScore
}

func appendReason(reasons []string, code string) []string {
	for _, existing := range reasons {
		if existing == code {
			return reasons
		}
	}
	return append(reasons, code)
}

func clampFloat(v, min, max float64) float64 {
	if v < min {
		return min
	}
	if v > max {
		return max
	}
	return v
}

func absFloat(v float64) float64 {
	if v < 0 {
		return -v
	}
	return v
}
