package protocol

type RiskPlan struct {
	PositionSize   float64 `json:"position_size"`
	MaxLossQuote   float64 `json:"max_loss_quote"`
	RiskPct        float64 `json:"risk_pct"`
	RewardRisk     float64 `json:"reward_risk"`
	Leverage       float64 `json:"leverage"`
	SizeMultiplier float64 `json:"size_multiplier"`
	Reason         string  `json:"reason"`
}
