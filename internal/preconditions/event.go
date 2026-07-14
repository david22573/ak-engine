package preconditions

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math"
	"sort"
	"strings"
	"time"
)

const RetainedEventSchemaVersion = "ak.engine.retained-event.downtrend-midvol-relief.v1"

type DecisionFeatures struct {
	Close         float64 `json:"close"`
	EMA50         float64 `json:"ema_50"`
	EMA200        float64 `json:"ema_200"`
	TrendSlope20  float64 `json:"trend_slope_20"`
	RealizedVol60 float64 `json:"realized_vol_60"`
}

type ContextInput struct {
	Symbol          string    `json:"symbol"`
	SnapshotID      string    `json:"snapshot_id"`
	SourceInputHash string    `json:"source_input_hash"`
	AvailableAt     time.Time `json:"available_at"`
	Return60        float64   `json:"return_60"`
}

type CostInputs struct {
	FeeBPS              float64 `json:"fee_bps"`
	SpreadBPS           float64 `json:"spread_bps"`
	SlippageBPS         float64 `json:"slippage_bps"`
	FundingBPS          float64 `json:"funding_bps"`
	AdverseSelectionBPS float64 `json:"adverse_selection_bps"`
}

type EventAttribution struct {
	Month   string `json:"month"`
	Quarter string `json:"quarter"`
	Regime  string `json:"regime"`
}

// RetainedEvent contains decision-time replay inputs only. Outcome values and
// independent-cluster assignments intentionally live outside this schema.
type RetainedEvent struct {
	SchemaVersion          string           `json:"schema_version"`
	EventID                string           `json:"event_id"`
	CandidateFamily        string           `json:"candidate_family"`
	CandidateVersion       string           `json:"candidate_version"`
	ImplementationHash     string           `json:"implementation_hash"`
	PrimarySymbol          string           `json:"primary_symbol"`
	EventTimestamp         time.Time        `json:"event_timestamp"`
	DecisionTimestamp      time.Time        `json:"decision_timestamp"`
	SourcePartitionID      string           `json:"source_partition_id"`
	SourceSnapshotID       string           `json:"source_snapshot_id"`
	SourceInputHash        string           `json:"source_input_hash"`
	FeatureSchemaVersion   string           `json:"feature_schema_version"`
	TrendState             string           `json:"trend_state"`
	PrimaryRegime          string           `json:"primary_regime"`
	VolatilityBucket       string           `json:"volatility_bucket"`
	Features               DecisionFeatures `json:"decision_features"`
	BTCContext             ContextInput     `json:"btc_context"`
	ETHContext             ContextInput     `json:"eth_context"`
	ReferencePrice         float64          `json:"reference_price"`
	EvaluationHorizon      string           `json:"evaluation_horizon"`
	EvaluationHorizonMS    int64            `json:"evaluation_horizon_ms"`
	WarmupSufficient       bool             `json:"warmup_sufficient"`
	DeterministicExclusion string           `json:"deterministic_exclusion_reason,omitempty"`
	CostInputs             CostInputs       `json:"cost_inputs"`
	Attribution            EventAttribution `json:"attribution"`
	ReplayInputHash        string           `json:"replay_input_hash"`
}

func SealRetainedEvent(event RetainedEvent) (RetainedEvent, error) {
	if event.SchemaVersion == "" {
		event.SchemaVersion = RetainedEventSchemaVersion
	}
	event.EventID = ""
	event.ReplayInputHash = ""
	id, err := computeEventID(event)
	if err != nil {
		return RetainedEvent{}, err
	}
	event.EventID = id
	hash, err := computeReplayInputHash(event)
	if err != nil {
		return RetainedEvent{}, err
	}
	event.ReplayInputHash = hash
	if err := ValidateRetainedEvent(event); err != nil {
		return RetainedEvent{}, err
	}
	return event, nil
}

func ValidateRetainedEvent(event RetainedEvent) error {
	if event.SchemaVersion != RetainedEventSchemaVersion {
		return errors.New("unsupported retained-event schema")
	}
	for name, value := range map[string]string{
		"event_id": event.EventID, "candidate_family": event.CandidateFamily, "candidate_version": event.CandidateVersion,
		"primary_symbol": event.PrimarySymbol, "source_partition_id": event.SourcePartitionID, "source_snapshot_id": event.SourceSnapshotID,
		"feature_schema_version": event.FeatureSchemaVersion, "trend_state": event.TrendState, "primary_regime": event.PrimaryRegime,
		"volatility_bucket": event.VolatilityBucket, "evaluation_horizon": event.EvaluationHorizon,
	} {
		if strings.TrimSpace(value) == "" {
			return fmt.Errorf("%s is required", name)
		}
	}
	for name, value := range map[string]string{"implementation_hash": event.ImplementationHash, "source_input_hash": event.SourceInputHash, "btc_context.source_input_hash": event.BTCContext.SourceInputHash, "eth_context.source_input_hash": event.ETHContext.SourceInputHash, "replay_input_hash": event.ReplayInputHash} {
		if !validSHA256(value) {
			return fmt.Errorf("%s must be a lowercase SHA-256 digest", name)
		}
	}
	if event.EventTimestamp.IsZero() || event.DecisionTimestamp.IsZero() || event.DecisionTimestamp.Before(event.EventTimestamp) {
		return errors.New("ordered event and decision timestamps are required")
	}
	if event.BTCContext.Symbol != "BTCUSDT" || event.ETHContext.Symbol != "ETHUSDT" {
		return errors.New("explicit BTCUSDT and ETHUSDT context is required")
	}
	for name, context := range map[string]ContextInput{"BTC": event.BTCContext, "ETH": event.ETHContext} {
		if context.SnapshotID == "" || context.AvailableAt.IsZero() || context.AvailableAt.After(event.DecisionTimestamp) || !finite(context.Return60) {
			return fmt.Errorf("%s context is incomplete or unavailable at decision time", name)
		}
	}
	if !event.WarmupSufficient {
		return errors.New("warm-up evidence is insufficient")
	}
	if !finite(event.ReferencePrice) || event.ReferencePrice <= 0 {
		return errors.New("positive finite reference price is required")
	}
	if event.EvaluationHorizon != "240m" || event.EvaluationHorizonMS != int64(4*time.Hour/time.Millisecond) {
		return errors.New("authoritative 240m evaluation horizon is required")
	}
	for name, value := range map[string]float64{"close": event.Features.Close, "ema_50": event.Features.EMA50, "ema_200": event.Features.EMA200, "trend_slope_20": event.Features.TrendSlope20, "realized_vol_60": event.Features.RealizedVol60} {
		if !finite(value) {
			return fmt.Errorf("%s must be finite", name)
		}
	}
	if event.Features.Close != event.ReferencePrice {
		return errors.New("reference price must match the authoritative decision close")
	}
	if event.TrendState != "DOWN" || event.PrimaryRegime != "DOWNTREND" || event.VolatilityBucket != "MID" {
		return errors.New("candidate decision classifications do not match the authoritative contract")
	}
	wantMonth := event.EventTimestamp.UTC().Format("2006-01")
	wantQuarter := fmt.Sprintf("%04d-Q%d", event.EventTimestamp.UTC().Year(), (int(event.EventTimestamp.UTC().Month())-1)/3+1)
	if event.Attribution.Month != wantMonth || event.Attribution.Quarter != wantQuarter || event.Attribution.Regime != event.PrimaryRegime {
		return errors.New("event attribution is inconsistent")
	}
	wantID, err := computeEventID(event)
	if err != nil || wantID != event.EventID {
		return errors.New("event_id does not match canonical identity")
	}
	wantHash, err := computeReplayInputHash(event)
	if err != nil || wantHash != event.ReplayInputHash {
		return errors.New("replay input hash mismatch")
	}
	return nil
}

func EncodeRetainedEvent(event RetainedEvent) ([]byte, error) {
	if err := ValidateRetainedEvent(event); err != nil {
		return nil, err
	}
	data, err := json.Marshal(event)
	if err != nil {
		return nil, err
	}
	return append(data, '\n'), nil
}

func DecodeRetainedEvent(data []byte) (RetainedEvent, error) {
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()
	var event RetainedEvent
	if err := decoder.Decode(&event); err != nil {
		return RetainedEvent{}, err
	}
	var extra any
	if err := decoder.Decode(&extra); err != io.EOF {
		return RetainedEvent{}, errors.New("retained event has trailing JSON data")
	}
	if err := ValidateRetainedEvent(event); err != nil {
		return RetainedEvent{}, err
	}
	return event, nil
}

func DeduplicateRetainedEvents(events []RetainedEvent) ([]RetainedEvent, error) {
	byID := make(map[string]RetainedEvent, len(events))
	for _, event := range events {
		if err := ValidateRetainedEvent(event); err != nil {
			return nil, err
		}
		if prior, exists := byID[event.EventID]; exists {
			if prior.ReplayInputHash != event.ReplayInputHash {
				return nil, fmt.Errorf("conflicting duplicate event %s", event.EventID)
			}
			continue
		}
		byID[event.EventID] = event
	}
	result := make([]RetainedEvent, 0, len(byID))
	for _, event := range byID {
		result = append(result, event)
	}
	sort.Slice(result, func(i, j int) bool {
		if !result[i].DecisionTimestamp.Equal(result[j].DecisionTimestamp) {
			return result[i].DecisionTimestamp.Before(result[j].DecisionTimestamp)
		}
		return result[i].EventID < result[j].EventID
	})
	return result, nil
}

type SchemaDescriptor struct {
	Version string   `json:"version"`
	Fields  []string `json:"fields"`
}

func RetainedEventSchemaDescriptor() SchemaDescriptor {
	return SchemaDescriptor{RetainedEventSchemaVersion, []string{
		"event_id", "candidate_family", "candidate_version", "implementation_hash", "primary_symbol", "event_timestamp", "decision_timestamp",
		"source_partition_id", "source_snapshot_id", "source_input_hash", "feature_schema_version", "trend_state", "primary_regime", "volatility_bucket",
		"decision_features.close", "decision_features.ema_50", "decision_features.ema_200", "decision_features.trend_slope_20", "decision_features.realized_vol_60",
		"btc_context", "eth_context", "reference_price", "evaluation_horizon", "evaluation_horizon_ms", "warmup_sufficient", "deterministic_exclusion_reason",
		"cost_inputs", "attribution", "replay_input_hash",
	}}
}

func RetainedEventSchemaHash() (string, error) {
	return canonicalDigest(RetainedEventSchemaDescriptor())
}

func computeEventID(event RetainedEvent) (string, error) {
	digest, err := canonicalDigest(struct{ Schema, Family, Version, Implementation, Symbol, EventTime, DecisionTime, Partition, Snapshot string }{
		event.SchemaVersion, event.CandidateFamily, event.CandidateVersion, strings.ToLower(event.ImplementationHash), event.PrimarySymbol,
		canonicalTime(event.EventTimestamp), canonicalTime(event.DecisionTimestamp), event.SourcePartitionID, event.SourceSnapshotID,
	})
	if err != nil {
		return "", err
	}
	return "event:" + strings.TrimPrefix(digest, "sha256:"), nil
}

func computeReplayInputHash(event RetainedEvent) (string, error) {
	copyEvent := event
	copyEvent.EventID = ""
	copyEvent.ReplayInputHash = ""
	return canonicalDigest(copyEvent)
}

func validSHA256(value string) bool {
	if len(value) != len("sha256:")+sha256.Size*2 || !strings.HasPrefix(value, "sha256:") || strings.ToLower(value) != value {
		return false
	}
	_, err := hex.DecodeString(strings.TrimPrefix(value, "sha256:"))
	return err == nil
}

func finite(value float64) bool { return !math.IsNaN(value) && !math.IsInf(value, 0) }
