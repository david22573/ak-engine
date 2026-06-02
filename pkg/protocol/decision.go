package protocol

type DecisionAction string

const (
	DecisionTake      DecisionAction = "TAKE"
	DecisionReject    DecisionAction = "REJECT"
	DecisionWatch     DecisionAction = "WATCH"
	DecisionReduce    DecisionAction = "REDUCE"
	DecisionSoftBlock DecisionAction = "SOFT_BLOCK"
)

type Decision struct {
	ID              string          `json:"id"`
	Symbol          string          `json:"symbol"`
	Action          DecisionAction  `json:"action"`
	Signal          Signal          `json:"signal"`
	RiskPlan        RiskPlan        `json:"risk_plan"`
	ExecutionIntent ExecutionIntent `json:"execution_intent"`
	Reasons         []string        `json:"reasons"`
	Rejectors       []string        `json:"rejectors"`
	Confidence      float64         `json:"confidence"`
	CreatedAt       int64           `json:"created_at"`
}
