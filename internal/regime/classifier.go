package regime

import (
	"fmt"
	"math"

	"github.com/davidmiguel22573/ak-engine/internal/features"
)

type Classifier struct {
	ThresholdOptions ThresholdOptions
}

func NewClassifier(opts ThresholdOptions) *Classifier {
	return &Classifier{
		ThresholdOptions: opts,
	}
}

func (c *Classifier) ClassifyRows(rows []features.Row) ([]Label, error) {
	labels := make([]Label, len(rows))
	for i := 0; i < len(rows); i++ {
		lbl, err := c.ClassifyOne(rows, i)
		if err != nil {
			return nil, fmt.Errorf("classify row %d: %w", i, err)
		}
		labels[i] = lbl
	}
	return labels, nil
}

func (c *Classifier) ClassifyOne(rows []features.Row, idx int) (Label, error) {
	if idx < 0 || idx >= len(rows) {
		return Label{}, fmt.Errorf("index out of bounds: %d", idx)
	}

	r := rows[idx]
	lbl := Label{
		Market:        r.Market,
		Symbol:        r.Symbol,
		Interval:      r.Interval,
		EventTimeMS:   r.EventTimeMS,
		AvailableAtMS: r.AvailableAtMS,
		Volatility:    "normal",
		Trend:         "range",
		Liquidity:     "normal",
		MarketBeta:    "btc_flat",
		Sentiment:     "unknown",
		Composite:     "normal",
		Reasons:       nil,
		Warmup:        r.Warmup,
	}

	if r.Warmup {
		return lbl, nil
	}

	t, ok := ComputeTrailingThresholds(rows, idx, c.ThresholdOptions)
	if !ok {
		lbl.Warmup = true
		return lbl, nil
	}

	var reasons []string

	// Volatility classification
	if r.ATRPct14 >= t.ATRPctP95 || r.VolumeRatio20 >= t.VolumeRatioP95 {
		lbl.Volatility = "shock"
		reasons = append(reasons, fmt.Sprintf("volatility: shock (atr_pct %.6f >= p95 %.6f or vol_ratio %.4f >= p95 %.4f)", r.ATRPct14, t.ATRPctP95, r.VolumeRatio20, t.VolumeRatioP95))
	} else if r.ATRPct14 <= t.ATRPctP20 && r.BBWidth20 <= t.BBWidthP20 {
		lbl.Volatility = "compressed"
		reasons = append(reasons, fmt.Sprintf("volatility: compressed (atr_pct %.6f <= p20 %.6f and bb_width %.6f <= p20 %.6f)", r.ATRPct14, t.ATRPctP20, r.BBWidth20, t.BBWidthP20))
	} else if r.ATRPct14 >= t.ATRPctP80 || r.BBWidth20 >= t.BBWidthP80 {
		lbl.Volatility = "expanded"
		reasons = append(reasons, fmt.Sprintf("volatility: expanded (atr_pct %.6f >= p80 %.6f or bb_width %.6f >= p80 %.6f)", r.ATRPct14, t.ATRPctP80, r.BBWidth20, t.BBWidthP80))
	}

	// Trend classification
	nearZero := false
	if r.Close > 0 && math.Abs(r.TrendSlope20/r.Close) < 0.0002 {
		nearZero = true
	}
	if r.Close > r.EMA20 && r.EMA20 > r.EMA50 && r.TrendSlope20 > 0 {
		lbl.Trend = "bull_trend"
		reasons = append(reasons, fmt.Sprintf("trend: bull_trend (close %.4f > ema20 %.4f > ema50 %.4f and slope %.6f > 0)", r.Close, r.EMA20, r.EMA50, r.TrendSlope20))
	} else if r.Close < r.EMA20 && r.EMA20 < r.EMA50 && r.TrendSlope20 < 0 {
		lbl.Trend = "bear_trend"
		reasons = append(reasons, fmt.Sprintf("trend: bear_trend (close %.4f < ema20 %.4f < ema50 %.4f and slope %.6f < 0)", r.Close, r.EMA20, r.EMA50, r.TrendSlope20))
	} else if nearZero && r.BBWidthPctRank60 < 0.35 {
		lbl.Trend = "chop"
		reasons = append(reasons, fmt.Sprintf("trend: chop (slope ratio %.8f < 0.0002 and bb_rank %.4f < 0.35)", math.Abs(r.TrendSlope20/r.Close), r.BBWidthPctRank60))
	}

	// Liquidity classification
	if r.VolumeRatio20 >= t.VolumeRatioP95 {
		lbl.Liquidity = "abnormal_spike"
		reasons = append(reasons, fmt.Sprintf("liquidity: abnormal_spike (vol_ratio %.4f >= p95 %.4f)", r.VolumeRatio20, t.VolumeRatioP95))
	} else if r.VolumeRatio20 >= t.VolumeRatioP80 {
		lbl.Liquidity = "heavy"
		reasons = append(reasons, fmt.Sprintf("liquidity: heavy (vol_ratio %.4f >= p80 %.4f)", r.VolumeRatio20, t.VolumeRatioP80))
	} else if r.VolumeRatio20 <= t.VolumeRatioP20 {
		lbl.Liquidity = "thin"
		reasons = append(reasons, fmt.Sprintf("liquidity: thin (vol_ratio %.4f <= p20 %.4f)", r.VolumeRatio20, t.VolumeRatioP20))
	}

	// Market Beta classification
	if r.BTCReturn60 > 0.003 {
		lbl.MarketBeta = "btc_up"
		reasons = append(reasons, fmt.Sprintf("market_beta: btc_up (btc_return_60 %.6f > 0.003)", r.BTCReturn60))
	} else if r.BTCReturn60 < -0.003 {
		lbl.MarketBeta = "btc_down"
		reasons = append(reasons, fmt.Sprintf("market_beta: btc_down (btc_return_60 %.6f < -0.003)", r.BTCReturn60))
	}

	// Composite classification
	if lbl.Volatility == "shock" {
		lbl.Composite = "shock_event"
		reasons = append(reasons, "composite: shock_event (volatility is shock)")
	} else if lbl.Volatility == "compressed" && (lbl.Trend == "range" || lbl.Trend == "chop") {
		lbl.Composite = "compressed_range"
		reasons = append(reasons, fmt.Sprintf("composite: compressed_range (volatility is compressed and trend is %s)", lbl.Trend))
	} else if lbl.Volatility == "expanded" && lbl.Trend == "bull_trend" {
		lbl.Composite = "expanded_bull"
		reasons = append(reasons, "composite: expanded_bull (volatility is expanded and trend is bull_trend)")
	} else if lbl.Volatility == "expanded" && lbl.Trend == "bear_trend" {
		lbl.Composite = "expanded_bear"
		reasons = append(reasons, "composite: expanded_bear (volatility is expanded and trend is bear_trend)")
	} else if lbl.Trend == "bull_trend" && lbl.MarketBeta == "btc_up" {
		lbl.Composite = "risk_on_trend"
		reasons = append(reasons, "composite: risk_on_trend (trend is bull_trend and market_beta is btc_up)")
	} else if lbl.Trend == "bear_trend" && lbl.MarketBeta == "btc_down" {
		lbl.Composite = "risk_off_trend"
		reasons = append(reasons, "composite: risk_off_trend (trend is bear_trend and market_beta is btc_down)")
	} else if lbl.Liquidity == "thin" && lbl.Trend == "chop" {
		lbl.Composite = "thin_chop"
		reasons = append(reasons, "composite: thin_chop (liquidity is thin and trend is chop)")
	}

	lbl.Reasons = reasons
	return lbl, nil
}
