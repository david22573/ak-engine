package strategy

type WindowAction string

const (
	ActionFullLong         WindowAction = "FULL_LONG"
	ActionFullShort        WindowAction = "FULL_SHORT"
	ActionProbeLong        WindowAction = "PROBE_LONG"
	ActionProbeShort       WindowAction = "PROBE_SHORT"
	ActionHold             WindowAction = "HOLD"
	ActionExit             WindowAction = "EXIT"
	ActionReverse          WindowAction = "REVERSE"
	ActionNoTradeHardBlock WindowAction = "NO_TRADE_HARD_BLOCK"
)

type DataFreshness string

const (
	DataFreshnessPass  DataFreshness = "PASS"
	DataFreshnessStale DataFreshness = "STALE"
)

type WindowDecision struct {
	Symbol           string        `json:"symbol"`
	WindowStartMS    int64         `json:"window_start_ms"`
	WindowEndMS      int64         `json:"window_end_ms"`
	Action           WindowAction  `json:"action"`
	Confidence       float64       `json:"confidence"`
	LongScore        float64       `json:"long_score"`
	ShortScore       float64       `json:"short_score"`
	ChopScore        float64       `json:"chop_score"`
	VolatilityScore  float64       `json:"volatility_score"`
	TrendScore       float64       `json:"trend_score"`
	PullbackScore    float64       `json:"pullback_score"`
	BreakoutScore    float64       `json:"breakout_score"`
	ExpectedMoveBPS  float64       `json:"expected_move_bps"`
	EstimatedCostBPS float64       `json:"estimated_cost_bps"`
	ReasonCodes      []string      `json:"reason_codes"`
	RiskFraction     float64       `json:"risk_fraction"`
	DataFreshness    DataFreshness `json:"data_freshness"`
	StrategyName     string        `json:"strategy_name"`
}
