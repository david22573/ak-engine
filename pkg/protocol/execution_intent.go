package protocol

type ExecutionIntentType string

const (
	IntentNone        ExecutionIntentType = "NONE"
	IntentMarketEntry ExecutionIntentType = "MARKET_ENTRY"
	IntentLimitEntry  ExecutionIntentType = "LIMIT_ENTRY"
	IntentBracket     ExecutionIntentType = "BRACKET"
)

type ExecutionIntent struct {
	Type       ExecutionIntentType `json:"type"`
	Symbol     string              `json:"symbol"`
	Side       Side                `json:"side"`
	Quantity   float64             `json:"quantity"`
	Entry      float64             `json:"entry"`
	StopLoss   float64             `json:"stop_loss"`
	TakeProfit float64             `json:"take_profit"`
	ClientTag  string              `json:"client_tag"`
	Reason     string              `json:"reason"`
}
