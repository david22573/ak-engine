package features

type Row struct {
	Market        string `json:"market"`
	Symbol        string `json:"symbol"`
	Interval      string `json:"interval"`
	EventTimeMS   int64  `json:"event_time_ms"`
	AvailableAtMS int64  `json:"available_at_ms"`

	Close float64 `json:"close"`

	Return1          float64 `json:"return_1"`
	Return5          float64 `json:"return_5"`
	Return15         float64 `json:"return_15"`
	RealizedVol20    float64 `json:"realized_vol_20"`
	RealizedVol60    float64 `json:"realized_vol_60"`
	ATR14            float64 `json:"atr_14"`
	ATRPct14         float64 `json:"atr_pct_14"`
	BBWidth20        float64 `json:"bb_width_20"`
	BBWidthPctRank60 float64 `json:"bb_width_pct_rank_60"`

	EMA20       float64 `json:"ema_20"`
	EMA50       float64 `json:"ema_50"`
	EMA200      float64 `json:"ema_200"`
	TrendSlope20 float64 `json:"trend_slope_20"`

	VolumeRatio20      float64 `json:"volume_ratio_20"`
	QuoteVolumeRatio20 float64 `json:"quote_volume_ratio_20"`
	TakerBuyRatio      float64 `json:"taker_buy_ratio"`

	BTCReturn60 float64 `json:"btc_return_60,omitempty"`
	ETHReturn60 float64 `json:"eth_return_60,omitempty"`

	Warmup bool `json:"warmup"`
}
