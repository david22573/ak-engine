package protocol

type Side string

const (
	SideLong  Side = "LONG"
	SideShort Side = "SHORT"
	SideHold  Side = "HOLD"
)

type Signal struct {
	Symbol      string             `json:"symbol"`
	Side        Side               `json:"side"`
	Entry       float64            `json:"entry"`
	StopLoss    float64            `json:"stop_loss"`
	TakeProfit  float64            `json:"take_profit"`
	Confidence  float64            `json:"confidence"`
	Score       float64            `json:"score"`
	Reason      string             `json:"reason"`
	Features    map[string]float64 `json:"features,omitempty"`
	GeneratedAt int64              `json:"generated_at"`
}
