package app

import (
	"fmt"
	"strconv"
	"strings"
)

// PreEntryContextSnapshot defines the snapshot of valid pre-entry fields for a candidate.
type PreEntryContextSnapshot struct {
	TrendRegime        string
	TrendStrength      float64
	VolatilityBucket   string
	FundingBucket      string
	BTCContextBucket   string
	ETHContextBucket   string
	VolumeLiquidBucket string
	EntryReasonCode    string
}

// ClusterContextSnapshot defines the snapshot for deduplication audits.
type ClusterContextSnapshot struct {
	Key       string
	Timestamp int64
	SpacingMs int64
	Size      int
	Ordinal   int
}

// CostStressSnapshot holds the outcomes across different baseline costs.
type CostStressSnapshot struct {
	GrossOutcomeBps float64
	NetOutcome5Bps  float64
	NetOutcome75Bps float64
	NetOutcome10Bps float64
	Win5Bps         bool
	Win75Bps        bool
	Win10Bps        bool
}

// CandidateEventSnapshot represents a clean snapshot of a generic candidate event.
// This decouples the JSON writing, size limits, and leaky field validation
// from the candidate-specific strategy logic.
type CandidateEventSnapshot struct {
	CandidateFamily string
	Symbol          string
	Side            string // "long" or "short"
	Horizon         string // e.g. "240m"
	EventTimeMS     int64

	PreEntry PreEntryContextSnapshot
	Cluster  ClusterContextSnapshot
	Cost     CostStressSnapshot

	// Diagnostic-only fields for analysis. Leaky fields are rejected by the underlying validator.
	Diagnostic map[string]interface{}
}

// CompactEventSource defines an interface for emitting compact retained events.
type CompactEventSource interface {
	EmitCompactEvent(snapshot CandidateEventSnapshot) error
}

// CompactEventEmissionConfig configuration for generic compact event emission.
type CompactEventEmissionConfig struct {
	Enabled bool
	Writer  *CompactEventWriter
}

// CompactEventEmitter implements CompactEventSource.
type CompactEventEmitter struct {
	config CompactEventEmissionConfig
}

// NewCompactEventEmitter creates a new generic emitter.
func NewCompactEventEmitter(cfg CompactEventEmissionConfig) *CompactEventEmitter {
	return &CompactEventEmitter{config: cfg}
}

// EmitCompactEvent converts a CandidateEventSnapshot to a CompactRetainedEvent
// and writes it to the underlying JSONL writer.
func (e *CompactEventEmitter) EmitCompactEvent(snapshot CandidateEventSnapshot) error {
	if !e.config.Enabled || e.config.Writer == nil {
		return nil
	}

	sideInt := 1
	candidateSide := "Long"
	if strings.ToLower(snapshot.Side) == "short" {
		sideInt = -1
		candidateSide = "Short"
	}

	horizonInt, err := strconv.Atoi(strings.TrimSuffix(snapshot.Horizon, "m"))
	if err != nil {
		return fmt.Errorf("invalid horizon format: %s", snapshot.Horizon)
	}

	candidateID := fmt.Sprintf("%s%s%s", snapshot.CandidateFamily, candidateSide, snapshot.Horizon)

	event := CompactRetainedEvent{
		CandidateID:    candidateID,
		Symbol:         snapshot.Symbol,
		Side:           sideInt,
		EventTimestamp: snapshot.EventTimeMS / 1000,
		Horizon:        horizonInt,
		PreEntry: PreEntryContext{
			TrendRegime:        snapshot.PreEntry.TrendRegime,
			TrendStrength:      snapshot.PreEntry.TrendStrength,
			VolatilityBucket:   snapshot.PreEntry.VolatilityBucket,
			FundingBucket:      snapshot.PreEntry.FundingBucket,
			BTCContextBucket:   snapshot.PreEntry.BTCContextBucket,
			ETHContextBucket:   snapshot.PreEntry.ETHContextBucket,
			VolumeLiquidBucket: snapshot.PreEntry.VolumeLiquidBucket,
			EntryReasonCode:    snapshot.PreEntry.EntryReasonCode,
		},
		Cluster: ClusterContext{
			Key:       snapshot.Cluster.Key,
			Timestamp: snapshot.Cluster.Timestamp,
			SpacingMs: snapshot.Cluster.SpacingMs,
			Size:      snapshot.Cluster.Size,
			Ordinal:   snapshot.Cluster.Ordinal,
		},
		GrossOutcomeBps: snapshot.Cost.GrossOutcomeBps,
		NetOutcome5Bps:  snapshot.Cost.NetOutcome5Bps,
		NetOutcome75Bps: snapshot.Cost.NetOutcome75Bps,
		NetOutcome10Bps: snapshot.Cost.NetOutcome10Bps,
		Win5Bps:         snapshot.Cost.Win5Bps,
		Win75Bps:        snapshot.Cost.Win75Bps,
		Win10Bps:        snapshot.Cost.Win10Bps,
	}

	event.DeriveTimeFields()

	for k, v := range snapshot.Diagnostic {
		if err := event.SetDiagnostic(k, v); err != nil {
			return fmt.Errorf("failed to set diagnostic field %s: %w", k, err)
		}
	}

	if err := event.Validate(); err != nil {
		return fmt.Errorf("invalid compact event snapshot: %w", err)
	}

	return e.config.Writer.Write(event)
}
