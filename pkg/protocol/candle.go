package protocol

type Candle struct {
	Market              string  `json:"market"`
	Symbol              string  `json:"symbol"`
	Interval            string  `json:"interval"`
	OpenTimeMS          int64   `json:"open_time_ms"`
	Open                float64 `json:"open"`
	High                float64 `json:"high"`
	Low                 float64 `json:"low"`
	Close               float64 `json:"close"`
	Volume              float64 `json:"volume"`
	CloseTimeMS         int64   `json:"close_time_ms"`
	QuoteAssetVolume    float64 `json:"quote_asset_volume"`
	NumberOfTrades      int64   `json:"number_of_trades"`
	TakerBuyBaseVolume  float64 `json:"taker_buy_base_volume"`
	TakerBuyQuoteVolume float64 `json:"taker_buy_quote_volume"`
}
