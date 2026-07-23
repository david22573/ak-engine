package app

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"
)

// PreEntryContext contains strictly pre-entry fields valid for risk filters.
type PreEntryContext struct {
	TrendRegime        string  `json:"trend"`
	TrendStrength      float64 `json:"trend_str,omitempty"`
	VolatilityBucket   string  `json:"vol"`
	FundingBucket      string  `json:"fund"`
	BTCContextBucket   string  `json:"btc_ctx,omitempty"`
	ETHContextBucket   string  `json:"eth_ctx,omitempty"`
	VolumeLiquidBucket string  `json:"vol_liq,omitempty"`
	EntryReasonCode    string  `json:"reason,omitempty"`
}

// ClusterContext contains fields for deduplication and clustering audits.
type ClusterContext struct {
	Key       string `json:"key"`
	Timestamp int64  `json:"ts"`
	SpacingMs int64  `json:"spacing"`
	Size      int    `json:"size"`
	Ordinal   int    `json:"ordinal"`
}

// CompactRetainedEvent is a compact, raw-free event schema (~400 bytes).
type CompactRetainedEvent struct {
	CandidateID    string `json:"candidate_id"`
	Symbol         string `json:"symbol"`
	Side           int    `json:"side"` // 1 for Long, -1 for Short
	EventTimestamp int64  `json:"ts"`
	Horizon        int    `json:"horizon"`
	Month          string `json:"month"`
	Quarter        string `json:"quarter"`

	PreEntry PreEntryContext `json:"pre"`
	Cluster  ClusterContext  `json:"cluster"`

	GrossOutcomeBps float64 `json:"grs_bps"`
	NetOutcome5Bps  float64 `json:"net_5"`
	NetOutcome75Bps float64 `json:"net_75"`
	NetOutcome10Bps float64 `json:"net_10"`

	Win5Bps  bool `json:"win_5"`
	Win75Bps bool `json:"win_75"`
	Win10Bps bool `json:"win_10"`

	OutcomeLabel string `json:"label,omitempty"`

	// Diagnostic/Leaky fields. MUST NOT be used for pre-entry filters.
	Diagnostic map[string]interface{} `json:"diagnostic,omitempty"`
}

// SetDiagnostic adds a diagnostic/leaky field. It rejects raw/account fields.
func (e *CompactRetainedEvent) SetDiagnostic(key string, value interface{}) error {
	lowerK := strings.ToLower(key)
	if strings.Contains(lowerK, "candle") || strings.Contains(lowerK, "payload") || strings.Contains(lowerK, "account") || strings.Contains(lowerK, "order") || strings.Contains(lowerK, "key") {
		return fmt.Errorf("invalid field detected: %s (no raw/account data allowed)", key)
	}

	if e.Diagnostic == nil {
		e.Diagnostic = make(map[string]interface{})
	}
	e.Diagnostic[key] = value
	return nil
}

// DeriveTimeFields populates Month and Quarter from EventTimestamp if missing.
func (e *CompactRetainedEvent) DeriveTimeFields() {
	if e.EventTimestamp > 0 && (e.Month == "" || e.Quarter == "") {
		t := time.Unix(e.EventTimestamp, 0).UTC()
		if e.Month == "" {
			e.Month = t.Format("2006-01")
		}
		if e.Quarter == "" {
			q := (t.Month()-1)/3 + 1
			e.Quarter = fmt.Sprintf("%d-Q%d", t.Year(), q)
		}
	}
}

// Validate checks all required constraints for the retained event.
func (e *CompactRetainedEvent) Validate() error {
	if e.CandidateID == "" {
		return errors.New("candidate_id is required")
	}
	if e.Symbol == "" {
		return errors.New("symbol is required")
	}
	if e.Side != 1 && e.Side != -1 {
		return errors.New("side must be 1 or -1")
	}
	if e.EventTimestamp <= 0 {
		return errors.New("valid event timestamp is required")
	}

	e.DeriveTimeFields()
	if e.Month == "" || e.Quarter == "" {
		return errors.New("month and quarter are required")
	}

	// Pre-Entry Required
	if e.PreEntry.TrendRegime == "" {
		return errors.New("pre_entry_trend_regime is required")
	}
	if e.PreEntry.VolatilityBucket == "" {
		return errors.New("pre_entry_volatility_bucket is required")
	}
	if e.PreEntry.FundingBucket == "" {
		return errors.New("pre_entry_funding_bucket is required")
	}

	// Cluster Required
	if e.Cluster.Key == "" {
		return errors.New("cluster_key is required")
	}
	if e.Cluster.Timestamp <= 0 {
		return errors.New("cluster_timestamp is required")
	}
	if e.Cluster.Size <= 0 {
		return errors.New("cluster_size must be > 0")
	}
	if e.Cluster.Ordinal <= 0 {
		return errors.New("cluster_ordinal must be > 0")
	}

	// Diagnostic/Leaky constraints
	if e.Diagnostic != nil {
		for k := range e.Diagnostic {
			lowerK := strings.ToLower(k)
			if strings.Contains(lowerK, "candle") || strings.Contains(lowerK, "payload") || strings.Contains(lowerK, "account") || strings.Contains(lowerK, "order") || strings.Contains(lowerK, "key") {
				return fmt.Errorf("invalid field detected in diagnostic: %s (no raw/account data allowed)", k)
			}
		}
	}

	return nil
}

// ToJSON serializes the event and checks the 1KB limit constraint.
func (e *CompactRetainedEvent) ToJSON() (string, error) {
	if err := e.Validate(); err != nil {
		return "", err
	}
	data, err := json.Marshal(e)
	if err != nil {
		return "", err
	}
	if len(data) >= 1024 {
		return "", fmt.Errorf("serialized event size %d exceeds 1KB target", len(data))
	}
	return string(data), nil
}

// CompactEventWriter collects and writes compact retained events to a JSONL string.
type CompactEventWriter struct {
	events []CompactRetainedEvent
}

func NewCompactEventWriter() *CompactEventWriter {
	return &CompactEventWriter{}
}

func (w *CompactEventWriter) Write(e CompactRetainedEvent) error {
	if err := e.Validate(); err != nil {
		return err
	}
	w.events = append(w.events, e)
	return nil
}

func (w *CompactEventWriter) ToJSONL() (string, error) {
	var sb strings.Builder
	for _, e := range w.events {
		j, err := e.ToJSON()
		if err != nil {
			return "", err
		}
		sb.WriteString(j)
		sb.WriteString("\n")
	}
	return sb.String(), nil
}
