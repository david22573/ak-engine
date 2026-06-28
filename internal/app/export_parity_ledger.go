package app

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"sync"
)

var parityLedgerMu sync.Mutex

type ParityLedgerRow struct {
	EventTimeMS           int64   `json:"event_time_ms"`
	DecisionTimeMS        int64   `json:"decision_time_ms"`
	Symbol                string  `json:"symbol"`
	Family                string  `json:"family"`
	Side                  string  `json:"side"`
	HorizonMinutes        int     `json:"horizon_minutes"`
	FundingRate           float64 `json:"funding_rate"`
	TrailingFundingMean   float64 `json:"trailing_funding_mean"`
	TrailingFundingStd    float64 `json:"trailing_funding_std"`
	TrailingFundingZ      float64 `json:"trailing_funding_z"`
	TrailingFundingP20    float64 `json:"trailing_funding_p20"`
	TriggerZLTEMinus1     bool    `json:"trigger_z_lte_minus_1"`
	TriggerRateLTEP20     bool    `json:"trigger_rate_lte_p20"`
	FundingBucket         string  `json:"funding_bucket"`
	RegimeBucket          string  `json:"regime_bucket"`
	FundingXRegimeBucket  string  `json:"funding_x_regime_bucket"`
	DelayCandles          int     `json:"delay_candles"`
	CostBps               float64 `json:"cost_bps"`
	ExpectedEdgeBps       float64 `json:"expected_edge_bps"`
	RealizedReturnBps     float64 `json:"realized_return_bps"`
	ValidFeatureState     bool    `json:"valid_feature_state"`
	ValidFundingState     bool    `json:"valid_funding_state"`
	ValidRegimeState      bool    `json:"valid_regime_state"`
	NoTradeReason         string  `json:"no_trade_reason"`
	InputHash             string  `json:"input_hash"`
	SignalHash            string  `json:"signal_hash"`
}

func emitParityLedgerRow(row ParityLedgerRow) {
	if row.Symbol != "XRPUSDT" {
		return
	}
	parityLedgerMu.Lock()
	defer parityLedgerMu.Unlock()
	
	err := os.MkdirAll("runs/reports", 0755)
	if err != nil {
		return
	}
	
	f, err := os.OpenFile("runs/reports/phase10_7r_xrpusdt_negative_funding_long_parity_ledger.jsonl", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return
	}
	defer f.Close()
	
	b, _ := json.Marshal(row)
	f.Write(b)
	f.WriteString("\n")
}

func computeParityInputHash(row ParityLedgerRow) string {
	payload := fmt.Sprintf("%d|%d|%s|%s|%s|%.8f|%.8f|%.8f|%t|%t|%t",
		row.EventTimeMS, row.DecisionTimeMS, row.Symbol, row.Family, row.Side,
		row.FundingRate, row.TrailingFundingMean, row.TrailingFundingStd,
		row.ValidFeatureState, row.ValidFundingState, row.ValidRegimeState)
	h := sha256.Sum256([]byte(payload))
	return hex.EncodeToString(h[:])
}

func computeParitySignalHash(row ParityLedgerRow) string {
	payload := fmt.Sprintf("%d|%s|%s|%t", row.EventTimeMS, row.Symbol, row.NoTradeReason, row.TriggerZLTEMinus1 || row.TriggerRateLTEP20)
	h := sha256.Sum256([]byte(payload))
	return hex.EncodeToString(h[:])
}
