package app

import (
	"bytes"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"math"
	"os"
	"path/filepath"
	"reflect"
	"strconv"
	"strings"
	"time"

	"github.com/david22573/ak-engine/internal/canonicalcontract"
	"github.com/david22573/ak-engine/internal/data"
	"github.com/david22573/ak-engine/internal/features"
	"github.com/david22573/ak-engine/internal/papersignal"
	"github.com/david22573/ak-engine/internal/regime"
	"github.com/david22573/ak-engine/internal/researchidentity"
	"github.com/david22573/ak-engine/internal/rifbridge"
	"github.com/david22573/ak-engine/internal/temporal"
)

const paperDecisionInputVersion = "ak.engine.paper_decision_input.v1"

type paperTradableObservation struct {
	Symbol       string  `json:"symbol"`
	Market       string  `json:"market"`
	TimestampUTC string  `json:"timestamp_utc"`
	Price        float64 `json:"price"`
}

// paperDecisionInput is a versioned, immutable local observation. Its file
// hash is persisted with every paper row. It is deliberately market-data only:
// no broker or exchange client is reachable from the paper runner.
type paperDecisionInput struct {
	Contract                  string                   `json:"contract"`
	Symbol                    string                   `json:"symbol"`
	Market                    string                   `json:"market"`
	Timeframe                 string                   `json:"timeframe"`
	SourceCandlesPath         string                   `json:"source_candles_path"`
	SourceCandlesHash         string                   `json:"source_candles_hash"`
	FeatureImplementationHash string                   `json:"feature_implementation_hash"`
	RegimeImplementationHash  string                   `json:"regime_implementation_hash"`
	Feature                   features.Row             `json:"feature"`
	Regime                    regime.Label             `json:"regime"`
	TradableObservation       paperTradableObservation `json:"tradable_observation"`
}

type paperCanonicalEvidence struct {
	DiagnosticHash            string
	EvidenceHash              string
	Candidate                 researchidentity.CandidateIdentity
	Configuration             researchidentity.ConfigurationIdentity
	DatasetHash               string
	PITHash                   string
	FeatureImplementationHash string
	RegimeImplementationHash  string
}

type paperStrategyDecision struct {
	Triggered       bool
	Reason          string
	DecisionTimeUTC string
	FillTimeUTC     string
	DataAsOfUTC     string
	EntryPrice      float64
	InputHash       string
}

func paperMetadataFromEvidence(evidence paperCanonicalEvidence) (paperCandidateMetadata, error) {
	configuration := evidence.Configuration.CanonicalResearchConfiguration
	if len(configuration.Brackets) == 0 || configuration.BracketWindowNS <= 0 {
		return paperCandidateMetadata{}, fmt.Errorf("research configuration has no canonical bracket model")
	}
	tp, err := strconv.ParseFloat(configuration.Brackets[0].TPBPS, 64)
	if err != nil || tp <= 0 {
		return paperCandidateMetadata{}, fmt.Errorf("invalid canonical target bps %q", configuration.Brackets[0].TPBPS)
	}
	sl, err := strconv.ParseFloat(configuration.Brackets[0].SLBPS, 64)
	if err != nil || sl <= 0 {
		return paperCandidateMetadata{}, fmt.Errorf("invalid canonical stop bps %q", configuration.Brackets[0].SLBPS)
	}
	side := papersignal.SignalSide(evidence.Candidate.Side)
	if side != papersignal.SideLong && side != papersignal.SideShort {
		return paperCandidateMetadata{}, fmt.Errorf("invalid canonical candidate side %q", evidence.Candidate.Side)
	}
	return paperCandidateMetadata{
		ID: evidence.Candidate.CandidateID, Version: evidence.Candidate.CandidateVersion,
		Hash: evidence.Candidate.ArtifactHash, Side: side,
		ObservationWindowMinutes: int(time.Duration(configuration.BracketWindowNS) / time.Minute),
		TargetBPS:                tp, StopBPS: sl,
		SupportedTimeframes: []string{configuration.Interval}, Source: evidence.Candidate.ImplementationLocator,
	}, nil
}

func buildCanonicalPaperSignal(meta paperCandidateMetadata, evidence paperCanonicalEvidence, decision paperStrategyDecision, symbol, market, timeframe, generatedAtUTC string) papersignal.PaperSignal {
	status := papersignal.StatusWait
	side := papersignal.SideWait
	outcome := papersignal.OutcomeStatus("")
	due := ""
	if decision.Triggered {
		status = papersignal.StatusAllowed
		side = meta.Side
		outcome = papersignal.OutcomePending
		fillAt, _ := time.Parse(time.RFC3339Nano, decision.FillTimeUTC)
		due = fillAt.Add(time.Duration(meta.ObservationWindowMinutes) * time.Minute).UTC().Format(time.RFC3339Nano)
	}
	return papersignal.PaperSignal{
		SchemaVersion: "2.0", SignalID: papersignal.GenerateSignalID(meta.Hash+evidence.Configuration.ArtifactHash+decision.InputHash, symbol, decision.DataAsOfUTC),
		GeneratedAtUTC: generatedAtUTC, CandidateID: meta.ID, CandidateVersion: meta.Version, CandidateHash: meta.Hash,
		ConfigurationHash: evidence.Configuration.ArtifactHash, ResearchEvidenceHash: evidence.EvidenceHash, DecisionInputHash: decision.InputHash,
		Symbol: symbol, MarketType: market, Timeframe: timeframe, Side: side, SignalStatus: status, SignalReason: decision.Reason,
		DataAsOfUTC: decision.DataAsOfUTC, DecisionTimeUTC: decision.DecisionTimeUTC, FillTimeUTC: decision.FillTimeUTC,
		DatasetManifestHash: evidence.DatasetHash, PitCoverageHash: evidence.PITHash,
		RIFStatus:  "CANONICAL_RESEARCH_EVIDENCE_VALIDATED_RESEARCH_ONLY",
		EntryModel: "first_tradable_observation_after_completed_decision_candle",
		ExitModel:  "canonical_target_stop_first_touch_v1", InvalidationModel: "canonical_stop_reference",
		ObservationWindow: meta.ObservationWindowMinutes, OutcomeStatus: outcome, OutcomeDueAtUTC: due,
		Notes: "Canonical forward-paper observation; research-only and no execution.",
	}
}

func canonicalPaperJournalRow(sig papersignal.PaperSignal, decision paperStrategyDecision, targetBPS, stopBPS float64) papersignal.PaperJournalRow {
	row := papersignal.PaperJournalRow{
		SignalID: sig.SignalID, CandidateID: sig.CandidateID, CandidateVersion: sig.CandidateVersion, CandidateHash: sig.CandidateHash,
		ConfigurationHash: sig.ConfigurationHash, ResearchEvidenceHash: sig.ResearchEvidenceHash, DecisionInputHash: sig.DecisionInputHash,
		GeneratedAtUTC: sig.GeneratedAtUTC, DecisionTimeUTC: sig.DecisionTimeUTC, FillTimeUTC: sig.FillTimeUTC,
		Symbol: sig.Symbol, MarketType: sig.MarketType, Timeframe: sig.Timeframe, Side: sig.Side,
		SignalStatus: sig.SignalStatus, SignalReason: sig.SignalReason, ObservationWindow: sig.ObservationWindow,
		OutcomeDueAtUTC: sig.OutcomeDueAtUTC, OutcomeStatus: sig.OutcomeStatus,
		DatasetHash: sig.DatasetManifestHash, PitCoverageHash: sig.PitCoverageHash, RIFStatus: sig.RIFStatus,
	}
	if !decision.Triggered {
		return row
	}
	row.EntryReferencePrice = decision.EntryPrice
	target, stop := paperTargetAndStop(sig.Side, decision.EntryPrice, targetBPS, stopBPS)
	row.TargetReferencePrice = &target
	row.StopReferencePrice = &stop
	return row
}

func appendCanonicalPaperObservation(path string, row papersignal.PaperJournalRow) error {
	if err := validateCanonicalPaperJournalDestination(path, row); err != nil {
		return err
	}
	return papersignal.AppendToJournal(path, row)
}

func validateCanonicalPaperJournalDestination(path string, row papersignal.PaperJournalRow) error {
	if err := validatePaperObservationIdentity(row); err != nil {
		return fmt.Errorf("new canonical paper observation: %w", err)
	}
	existing, err := papersignal.ReadJournal(path)
	if err != nil && !os.IsNotExist(err) {
		return err
	}
	for i, prior := range existing {
		if err := validatePaperObservationIdentity(prior); err != nil {
			return fmt.Errorf("journal contains historical/noncanonical row %d; use a new identity-isolated journal: %w", i+1, err)
		}
		if prior.CandidateID != row.CandidateID || prior.CandidateVersion != row.CandidateVersion || prior.CandidateHash != row.CandidateHash || prior.ConfigurationHash != row.ConfigurationHash || prior.ResearchEvidenceHash != row.ResearchEvidenceHash {
			return fmt.Errorf("journal identity differs at row %d; old and regenerated samples must not mix", i+1)
		}
		if prior.SignalID == row.SignalID {
			return fmt.Errorf("duplicate canonical paper observation %s", row.SignalID)
		}
	}
	return nil
}

func loadPaperCanonicalEvidence(path, candidateID, symbol, market, timeframe string) (paperCanonicalEvidence, error) {
	if strings.TrimSpace(path) == "" {
		return paperCanonicalEvidence{}, fmt.Errorf("canonical research evidence is required")
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		return paperCanonicalEvidence{}, fmt.Errorf("read canonical research evidence: %w", err)
	}
	validated, err := canonicalcontract.ValidateArtifact(raw, true)
	if err != nil {
		return paperCanonicalEvidence{}, fmt.Errorf("validate canonical research evidence: %w", err)
	}
	if validated.SchemaName != rifbridge.LocalResearchDiagnosticsSchemaName {
		return paperCanonicalEvidence{}, fmt.Errorf("research evidence must be a canonical %s artifact", rifbridge.LocalResearchDiagnosticsSchemaName)
	}
	var diagnostic rifbridge.LocalResearchDiagnostics
	if err := json.Unmarshal(raw, &diagnostic); err != nil {
		return paperCanonicalEvidence{}, fmt.Errorf("decode canonical research evidence: %w", err)
	}
	if diagnostic.IdentityStatus != researchidentity.StatusComplete || diagnostic.LocalIntegrity != rifbridge.LocalIntegrityPassed || !diagnostic.EligibleForRIFReview || diagnostic.ResearchEvidence == nil {
		return paperCanonicalEvidence{}, fmt.Errorf("research evidence is not complete and eligible for RIF review")
	}
	evidence := diagnostic.ResearchEvidence
	if evidence.AuthorityStatus != rifbridge.AuthorityStatusNoneResearchOnly || !evidence.EligibleForRIFReview || evidence.Classification != rifbridge.ResearchStatusValidatedResearchLead || len(evidence.BlockingFindings) != 0 {
		return paperCanonicalEvidence{}, fmt.Errorf("research evidence is not a validated research lead")
	}
	identity := evidence.ResearchIdentity
	if identity.ArtifactHash == "" || evidence.EvidenceID != identity.ArtifactHash || evidence.MetricResults.EvaluationSeriesHash != identity.Series.ArtifactHash {
		return paperCanonicalEvidence{}, fmt.Errorf("research evidence identity bindings are inconsistent")
	}
	if candidateID != identity.Candidate.CandidateID {
		return paperCanonicalEvidence{}, fmt.Errorf("candidate mismatch: evidence %q request %q", identity.Candidate.CandidateID, candidateID)
	}
	if symbol != identity.Configuration.Symbol || market != identity.Configuration.Market || timeframe != identity.Configuration.Interval {
		return paperCanonicalEvidence{}, fmt.Errorf("research evidence scope mismatch: got %s/%s/%s want %s/%s/%s", identity.Configuration.Market, identity.Configuration.Symbol, identity.Configuration.Interval, market, symbol, timeframe)
	}
	root, err := researchidentity.FindRepositoryRoot("")
	if err != nil {
		return paperCanonicalEvidence{}, fmt.Errorf("resolve Engine repository root: %w", err)
	}
	registry, err := researchidentity.DefaultRegistry()
	if err != nil {
		return paperCanonicalEvidence{}, err
	}
	current, err := registry.Resolve(root, identity.Candidate.Family, identity.Candidate.Side)
	if err != nil {
		return paperCanonicalEvidence{}, fmt.Errorf("resolve current candidate implementation: %w", err)
	}
	if current.ArtifactHash != identity.Candidate.ArtifactHash || current.CandidateID != identity.Candidate.CandidateID || current.CandidateVersion != identity.Candidate.CandidateVersion {
		return paperCanonicalEvidence{}, fmt.Errorf("research evidence candidate/version/source identity does not match current compiled candidate")
	}
	return paperCanonicalEvidence{
		DiagnosticHash:            diagnostic.ArtifactHash,
		EvidenceHash:              evidence.ArtifactHash,
		Candidate:                 identity.Candidate,
		Configuration:             identity.Configuration,
		DatasetHash:               identity.Dataset.ArtifactHash,
		PITHash:                   identity.PIT.ArtifactHash,
		FeatureImplementationHash: identity.Feature.ImplementationHash,
		RegimeImplementationHash: func() string {
			if identity.Regime == nil {
				return ""
			}
			return identity.Regime.ImplementationHash
		}(),
	}, nil
}

func loadPaperDecision(path string, evidence paperCanonicalEvidence, generatedAt time.Time) (paperStrategyDecision, error) {
	if strings.TrimSpace(path) == "" {
		return paperStrategyDecision{}, fmt.Errorf("paper decision input is required")
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		return paperStrategyDecision{}, fmt.Errorf("read paper decision input: %w", err)
	}
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.DisallowUnknownFields()
	var input paperDecisionInput
	if err := decoder.Decode(&input); err != nil {
		return paperStrategyDecision{}, fmt.Errorf("decode paper decision input: %w", err)
	}
	if err := decoder.Decode(&struct{}{}); err != io.EOF {
		return paperStrategyDecision{}, fmt.Errorf("paper decision input must contain exactly one JSON object")
	}
	if input.Contract != paperDecisionInputVersion {
		return paperStrategyDecision{}, fmt.Errorf("unsupported paper decision input contract %q", input.Contract)
	}
	config := evidence.Configuration
	if input.FeatureImplementationHash == "" || input.FeatureImplementationHash != evidence.FeatureImplementationHash || input.RegimeImplementationHash == "" || input.RegimeImplementationHash != evidence.RegimeImplementationHash {
		return paperStrategyDecision{}, fmt.Errorf("paper decision feature/regime implementation identity mismatch")
	}
	if input.Symbol != config.Symbol || input.Market != config.Market || input.Timeframe != config.Interval {
		return paperStrategyDecision{}, fmt.Errorf("paper decision scope does not match research configuration")
	}
	if input.Feature.Symbol != input.Symbol || input.Feature.Market != input.Market || input.Feature.Interval != input.Timeframe || input.Regime.Symbol != input.Symbol || input.Regime.Market != input.Market || input.Regime.Interval != input.Timeframe {
		return paperStrategyDecision{}, fmt.Errorf("feature/regime scope does not match paper decision scope")
	}
	if input.Feature.EventTimeMS != input.Regime.EventTimeMS || input.Feature.AvailableAtMS != input.Regime.AvailableAtMS {
		return paperStrategyDecision{}, fmt.Errorf("feature and regime observations do not share one decision timestamp")
	}
	intervalMS, err := data.ParseIntervalToMS(input.Timeframe)
	if err != nil {
		return paperStrategyDecision{}, err
	}
	if err := temporal.ValidateCandleClose(input.Feature.EventTimeMS, input.Feature.AvailableAtMS, intervalMS); err != nil {
		return paperStrategyDecision{}, fmt.Errorf("invalid close-derived paper feature: %w", err)
	}
	if err := validatePaperDecisionSource(path, input); err != nil {
		return paperStrategyDecision{}, err
	}
	if err := validateDeepFeatureRows([]features.Row{input.Feature}); err != nil {
		return paperStrategyDecision{}, err
	}
	featureValue := reflect.ValueOf(input.Feature)
	for i := 0; i < featureValue.NumField(); i++ {
		field := featureValue.Field(i)
		if field.Kind() == reflect.Float64 && (math.IsNaN(field.Float()) || math.IsInf(field.Float(), 0)) {
			return paperStrategyDecision{}, fmt.Errorf("paper feature %s must be finite", featureValue.Type().Field(i).Name)
		}
	}
	if input.Feature.Warmup || input.Feature.Close <= 0 || input.Feature.EMA20 <= 0 || input.Feature.EMA50 <= 0 || input.Feature.EMA200 <= 0 {
		return paperStrategyDecision{}, fmt.Errorf("paper feature is warmup or has nonpositive price indicators")
	}
	if err := validateDeepLabels([]regime.Label{input.Regime}); err != nil {
		return paperStrategyDecision{}, err
	}
	if input.Regime.Warmup || input.Regime.Volatility == "" || input.Regime.Trend == "" || input.Regime.Liquidity == "" || input.Regime.MarketBeta == "" {
		return paperStrategyDecision{}, fmt.Errorf("paper regime is warmup or incomplete")
	}
	tradableAt, err := time.Parse(time.RFC3339, input.TradableObservation.TimestampUTC)
	if err != nil {
		return paperStrategyDecision{}, fmt.Errorf("invalid tradable observation timestamp: %w", err)
	}
	tradableAt = tradableAt.UTC()
	if input.TradableObservation.Symbol != input.Symbol || input.TradableObservation.Market != input.Market {
		return paperStrategyDecision{}, fmt.Errorf("tradable observation scope mismatch")
	}
	if input.TradableObservation.Price <= 0 || math.IsNaN(input.TradableObservation.Price) || math.IsInf(input.TradableObservation.Price, 0) {
		return paperStrategyDecision{}, fmt.Errorf("tradable observation price must be finite and positive")
	}
	decisionMS := input.Feature.AvailableAtMS
	if tradableAt.UnixMilli() != decisionMS+1 {
		return paperStrategyDecision{}, fmt.Errorf("tradable observation must be the first millisecond after the completed decision candle")
	}
	if !generatedAt.Equal(tradableAt) {
		return paperStrategyDecision{}, fmt.Errorf("generated-at must exactly equal the as-of tradable observation timestamp")
	}
	if err := (temporal.Observation{
		SourceEventMS: input.Feature.EventTimeMS, SourceAvailableMS: input.Feature.AvailableAtMS,
		DecisionMS: decisionMS, NextTradableMS: tradableAt.UnixMilli(), FillMS: tradableAt.UnixMilli(),
	}).Validate(); err != nil {
		return paperStrategyDecision{}, err
	}
	base, directionOK, betaOK, betaApplicable := deepCandidateRule(input.Feature, input.Regime, evidence.Candidate.Family, evidence.Candidate.Side)
	triggered := base && directionOK && (!betaApplicable || betaOK)
	reason := "compiled candidate rule did not trigger"
	if triggered {
		reason = "compiled candidate rule triggered on canonical as-of input"
	}
	hash, err := papersignal.HashFile(path)
	if err != nil {
		return paperStrategyDecision{}, err
	}
	return paperStrategyDecision{
		Triggered: triggered, Reason: reason,
		DecisionTimeUTC: time.UnixMilli(decisionMS).UTC().Format(time.RFC3339Nano),
		FillTimeUTC:     tradableAt.Format(time.RFC3339Nano), DataAsOfUTC: tradableAt.Format(time.RFC3339Nano),
		EntryPrice: input.TradableObservation.Price, InputHash: hash,
	}, nil
}

func validatePaperDecisionSource(decisionPath string, input paperDecisionInput) error {
	if len(input.SourceCandlesHash) != 64 {
		return fmt.Errorf("source candle hash must be 64 lowercase hexadecimal characters")
	}
	decodedHash, err := hex.DecodeString(input.SourceCandlesHash)
	if err != nil || len(decodedHash) != 32 || strings.ToLower(input.SourceCandlesHash) != input.SourceCandlesHash {
		return fmt.Errorf("source candle hash must be 64 lowercase hexadecimal characters")
	}
	if input.SourceCandlesPath == "" || filepath.IsAbs(input.SourceCandlesPath) || filepath.Clean(input.SourceCandlesPath) != input.SourceCandlesPath || input.SourceCandlesPath == "." || input.SourceCandlesPath == ".." || strings.HasPrefix(input.SourceCandlesPath, ".."+string(filepath.Separator)) {
		return fmt.Errorf("source candle path must be a clean contained relative path")
	}
	root, err := filepath.Abs(filepath.Dir(decisionPath))
	if err != nil {
		return err
	}
	sourcePath := filepath.Join(root, input.SourceCandlesPath)
	resolved, err := filepath.EvalSymlinks(sourcePath)
	if err != nil {
		return fmt.Errorf("resolve source candles: %w", err)
	}
	relative, err := filepath.Rel(root, resolved)
	if err != nil || filepath.IsAbs(relative) || relative == ".." || strings.HasPrefix(relative, ".."+string(filepath.Separator)) || resolved != sourcePath {
		return fmt.Errorf("source candle path escapes the decision-input directory or is a symlink")
	}
	info, err := os.Stat(sourcePath)
	if err != nil || !info.Mode().IsRegular() {
		return fmt.Errorf("source candle path must identify a regular file")
	}
	hash, err := papersignal.HashFile(sourcePath)
	if err != nil {
		return err
	}
	if hash != input.SourceCandlesHash {
		return fmt.Errorf("source candle hash mismatch")
	}
	raw, err := os.ReadFile(sourcePath)
	if err != nil {
		return err
	}
	req := data.CandleRequest{Market: input.Market, Symbol: input.Symbol, Interval: input.Timeframe}
	candles, err := data.ParseJSONCandlesNoValidate(raw, req)
	if err != nil {
		return fmt.Errorf("parse decision source candles: %w", err)
	}
	if err := data.ValidateCandlesForRequest(req, candles); err != nil {
		return fmt.Errorf("validate decision source candles: %w", err)
	}
	last := candles[len(candles)-1]
	if last.OpenTimeMS != input.Feature.EventTimeMS || last.CloseTimeMS != input.Feature.AvailableAtMS || last.Close != input.Feature.Close {
		return fmt.Errorf("latest source candle does not exactly match the decision feature observation")
	}
	return nil
}
