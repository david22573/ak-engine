package features

import (
	"fmt"
	"sort"

	"github.com/davidmiguel22573/ak-engine/internal/data"
	"github.com/davidmiguel22573/ak-engine/pkg/protocol"
)

type BuildOptions struct {
	Market       string
	Symbol       string
	Interval     string
	DropWarmup   bool
	ContextBTC   []protocol.Candle
	ContextETH   []protocol.Candle
}

func BuildRows(candles []protocol.Candle, opts BuildOptions) ([]Row, error) {
	if len(candles) == 0 {
		return nil, nil
	}

	// Sort candles by OpenTimeMS
	sort.Slice(candles, func(i, j int) bool {
		return candles[i].OpenTimeMS < candles[j].OpenTimeMS
	})

	// Validate candles using data.ValidateCandles
	if err := data.ValidateCandles(opts.Interval, candles); err != nil {
		return nil, fmt.Errorf("validate primary candles: %w", err)
	}

	// Sort context candles if present
	if len(opts.ContextBTC) > 0 {
		sort.Slice(opts.ContextBTC, func(i, j int) bool {
			return opts.ContextBTC[i].OpenTimeMS < opts.ContextBTC[j].OpenTimeMS
		})
	}
	if len(opts.ContextETH) > 0 {
		sort.Slice(opts.ContextETH, func(i, j int) bool {
			return opts.ContextETH[i].OpenTimeMS < opts.ContextETH[j].OpenTimeMS
		})
	}

	// Build context maps for O(1) timestamp lookup
	btcCloseMap := make(map[int64]float64)
	for _, c := range opts.ContextBTC {
		btcCloseMap[c.OpenTimeMS] = c.Close
	}
	ethCloseMap := make(map[int64]float64)
	for _, c := range opts.ContextETH {
		ethCloseMap[c.OpenTimeMS] = c.Close
	}

	n := len(candles)
	closes := make([]float64, n)
	volumes := make([]float64, n)
	quoteVolumes := make([]float64, n)
	for i, c := range candles {
		closes[i] = c.Close
		volumes[i] = c.Volume
		quoteVolumes[i] = c.QuoteAssetVolume
	}

	// Precompute EMAs for the entire series
	ema20s := EMA(closes, 20)
	ema50s := EMA(closes, 50)
	ema200s := EMA(closes, 200)

	// Precompute BBWidth20 for the entire series
	bbWidth20s := make([]float64, n)
	bbWidth20Ok := make([]bool, n)
	for i := 0; i < n; i++ {
		bbWidth20s[i], bbWidth20Ok[i] = BBWidth(closes, i, 20, 2.0)
	}

	var rows []Row
	for i := 0; i < n; i++ {
		c := candles[i]
		availableAt := c.CloseTimeMS
		if availableAt <= c.OpenTimeMS {
			// Fallback: estimate close time
			// standard interval duration could be derived, but CloseTimeMS should be correct.
			availableAt = c.OpenTimeMS + 60000 // default 1m fallback if invalid
		}

		row := Row{
			Market:        opts.Market,
			Symbol:        opts.Symbol,
			Interval:      opts.Interval,
			EventTimeMS:   c.OpenTimeMS,
			AvailableAtMS: availableAt,
			Close:         c.Close,
		}

		warmup := false

		// Return 1
		if i >= 1 && closes[i-1] > 0 {
			row.Return1 = (closes[i] - closes[i-1]) / closes[i-1]
		} else {
			warmup = true
		}

		// Return 5
		if i >= 5 && closes[i-5] > 0 {
			row.Return5 = (closes[i] - closes[i-5]) / closes[i-5]
		} else {
			warmup = true
		}

		// Return 15
		if i >= 15 && closes[i-15] > 0 {
			row.Return15 = (closes[i] - closes[i-15]) / closes[i-15]
		} else {
			warmup = true
		}

		// Realized Volatility 20
		var ok bool
		if row.RealizedVol20, ok = RealizedVol(closes, i, 20); !ok {
			warmup = true
		}

		// Realized Volatility 60
		if row.RealizedVol60, ok = RealizedVol(closes, i, 60); !ok {
			warmup = true
		}

		// ATR 14
		if row.ATR14, ok = ATR(candles, i, 14); ok {
			if c.Close > 0 {
				row.ATRPct14 = row.ATR14 / c.Close
			} else {
				warmup = true
			}
		} else {
			warmup = true
		}

		// Bollinger Width 20
		if bbWidth20Ok[i] {
			row.BBWidth20 = bbWidth20s[i]
		} else {
			warmup = true
		}

		// BB Width Percentile Rank 60
		if row.BBWidthPctRank60, ok = PercentRankTrailing(bbWidth20s, i, 60); !ok {
			warmup = true
		}

		// EMAs
		if i >= 19 {
			row.EMA20 = ema20s[i]
		} else {
			warmup = true
		}
		if i >= 49 {
			row.EMA50 = ema50s[i]
		} else {
			warmup = true
		}
		if i >= 199 {
			row.EMA200 = ema200s[i]
		} else {
			warmup = true
		}

		// Linear Slope 20
		if row.TrendSlope20, ok = LinearSlope(closes, i, 20); !ok {
			warmup = true
		}

		// Volume Ratio 20
		if i >= 20 {
			volSum := 0.0
			for k := i - 20; k < i; k++ {
				volSum += volumes[k]
			}
			volSMA := volSum / 20.0
			row.VolumeRatio20 = SafeRatio(c.Volume, volSMA)

			qVolSum := 0.0
			for k := i - 20; k < i; k++ {
				qVolSum += quoteVolumes[k]
			}
			qVolSMA := qVolSum / 20.0
			row.QuoteVolumeRatio20 = SafeRatio(c.QuoteAssetVolume, qVolSMA)
		} else {
			warmup = true
		}

		// Taker Buy Ratio
		row.TakerBuyRatio = SafeRatio(c.TakerBuyBaseVolume, c.Volume)

		// Context Returns 60
		if i >= 60 {
			tCurr := c.OpenTimeMS
			tPrev := candles[i-60].OpenTimeMS
			if btcCurr, ok1 := btcCloseMap[tCurr]; ok1 {
				if btcPrev, ok2 := btcCloseMap[tPrev]; ok2 && btcPrev > 0 {
					row.BTCReturn60 = (btcCurr - btcPrev) / btcPrev
				}
			}
			if ethCurr, ok1 := ethCloseMap[tCurr]; ok1 {
				if ethPrev, ok2 := ethCloseMap[tPrev]; ok2 && ethPrev > 0 {
					row.ETHReturn60 = (ethCurr - ethPrev) / ethPrev
				}
			}
		}

		row.Warmup = warmup

		if !opts.DropWarmup || !row.Warmup {
			rows = append(rows, row)
		}
	}

	return rows, nil
}
