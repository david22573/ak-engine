package regime

type VolatilityLabel string
type TrendLabel string
type LiquidityLabel string
type MarketBetaLabel string
type SentimentLabel string

const (
	VolCompressed VolatilityLabel = "compressed"
	VolNormal     VolatilityLabel = "normal"
	VolExpanded   VolatilityLabel = "expanded"
	VolShock      VolatilityLabel = "shock"

	TrendBull  TrendLabel = "bull_trend"
	TrendBear  TrendLabel = "bear_trend"
	TrendRange TrendLabel = "range"
	TrendChop  TrendLabel = "chop"

	LiquidityThin     LiquidityLabel = "thin"
	LiquidityNormal   LiquidityLabel = "normal"
	LiquidityHeavy    LiquidityLabel = "heavy"
	LiquidityAbnormal LiquidityLabel = "abnormal_spike"

	BetaBTCUp   MarketBetaLabel = "btc_up"
	BetaBTCDown MarketBetaLabel = "btc_down"
	BetaBTCFlat MarketBetaLabel = "btc_flat"

	SentimentUnknown SentimentLabel = "unknown"
)

type Label struct {
	Market        string   `json:"market"`
	Symbol        string   `json:"symbol"`
	Interval      string   `json:"interval"`
	EventTimeMS   int64    `json:"event_time_ms"`
	AvailableAtMS int64    `json:"available_at_ms"`
	Volatility    string   `json:"volatility"`
	Trend         string   `json:"trend"`
	Liquidity     string   `json:"liquidity"`
	MarketBeta    string   `json:"market_beta"`
	Sentiment     string   `json:"sentiment"`
	Composite     string   `json:"composite"`
	Reasons       []string `json:"reasons"`
	Warmup        bool     `json:"warmup"`
}
