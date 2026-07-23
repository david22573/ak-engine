package app

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/david22573/ak-engine/internal/papersignal"
)

type paperCandidateMetadata struct {
	ID                       string
	Version                  string
	Hash                     string
	Side                     papersignal.SignalSide
	ObservationWindowMinutes int
	TargetBPS                float64
	StopBPS                  float64
	SupportedTimeframes      []string
	Source                   string
}

type paperRIFEvidence struct {
	Status        string
	DatasetHash   string
	ManifestHash  string
	UniverseHash  string
	LifecycleHash string
	PITHash       string
	Warnings      []string
	BlocksSignal  bool
	BlockReason   string
}

type paperReferenceSnapshot struct {
	Symbol         string  `json:"symbol"`
	TimestampUTC   string  `json:"timestamp_utc"`
	ReferencePrice float64 `json:"reference_price"`
	Price          float64 `json:"price"`
	Close          float64 `json:"close"`
	SnapshotHash   string  `json:"snapshot_hash"`
}

var paperCandidateRegistry = map[string]paperCandidateMetadata{
	"DowntrendMidvolReliefShort240m":            newPaperCandidate("DowntrendMidvolReliefShort240m", papersignal.SideShort, 240, "internal/app/phase12_downtrend_midvol_relief.go"),
	"NegativeFundingLong":                       newPaperCandidate("NegativeFundingLong", papersignal.SideLong, 60, "internal/app/funding_events.go"),
	"PositiveFundingShort":                      newPaperCandidate("PositiveFundingShort", papersignal.SideShort, 60, "internal/app/funding_events.go"),
	"FundingFlipLong":                           newPaperCandidate("FundingFlipLong", papersignal.SideLong, 60, "internal/app/funding_events.go"),
	"FundingFlipShort":                          newPaperCandidate("FundingFlipShort", papersignal.SideShort, 60, "internal/app/funding_events.go"),
	"RegimeFundingLong":                         newPaperCandidate("RegimeFundingLong", papersignal.SideLong, 60, "internal/app/funding_events.go"),
	"RegimeFundingShort":                        newPaperCandidate("RegimeFundingShort", papersignal.SideShort, 60, "internal/app/funding_events.go"),
	"ConfirmedNegativeFundingLong":              newPaperCandidate("ConfirmedNegativeFundingLong", papersignal.SideLong, 60, "internal/app/funding_events.go"),
	"ConfirmedPositiveFundingShort":             newPaperCandidate("ConfirmedPositiveFundingShort", papersignal.SideShort, 60, "internal/app/funding_events.go"),
	"BreakoutFundingLong":                       newPaperCandidate("BreakoutFundingLong", papersignal.SideLong, 60, "internal/app/funding_events.go"),
	"BreakoutFundingShort":                      newPaperCandidate("BreakoutFundingShort", papersignal.SideShort, 60, "internal/app/funding_events.go"),
	"VolumeImbalanceFundingReversionProxyLong":  newPaperCandidate("VolumeImbalanceFundingReversionProxyLong", papersignal.SideLong, 60, "internal/app/funding_events.go"),
	"VolumeImbalanceFundingReversionProxyShort": newPaperCandidate("VolumeImbalanceFundingReversionProxyShort", papersignal.SideShort, 60, "internal/app/funding_events.go"),
	"CompressionBreakoutLong":                   newPaperCandidate("CompressionBreakoutLong", papersignal.SideLong, 60, "internal/app/evaluate_compression_breakout.go"),
	"CompressionBreakoutShort":                  newPaperCandidate("CompressionBreakoutShort", papersignal.SideShort, 60, "internal/app/evaluate_compression_breakout.go"),
	"RegimeAwareCompressionBreakout_LONG":       newPaperCandidate("RegimeAwareCompressionBreakout_LONG", papersignal.SideLong, 60, "internal/app/plan_compression_breakout_oos.go"),
}

func newPaperCandidate(id string, side papersignal.SignalSide, observationMinutes int, source string) paperCandidateMetadata {
	sum := sha256.Sum256([]byte(id + "|" + source))
	return paperCandidateMetadata{
		ID:                       id,
		Version:                  "1.0",
		Hash:                     hex.EncodeToString(sum[:])[:16],
		Side:                     side,
		ObservationWindowMinutes: observationMinutes,
		TargetBPS:                100,
		StopBPS:                  75,
		SupportedTimeframes:      []string{"1m", "5m", "15m", "1h", "60m", "240m"},
		Source:                   source,
	}
}

func loadPaperCandidateMetadata(candidateID, timeframe string) (paperCandidateMetadata, error) {
	meta, ok := paperCandidateRegistry[candidateID]
	if !ok {
		return paperCandidateMetadata{}, fmt.Errorf("unknown paper candidate %q; known existing candidates: %s", candidateID, strings.Join(sortedPaperCandidateIDs(), ","))
	}
	if timeframe != "" && !containsPaperString(meta.SupportedTimeframes, timeframe) {
		return paperCandidateMetadata{}, fmt.Errorf("candidate %s does not support timeframe %s", candidateID, timeframe)
	}
	return meta, nil
}

func sortedPaperCandidateIDs() []string {
	ids := make([]string, 0, len(paperCandidateRegistry))
	for id := range paperCandidateRegistry {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	return ids
}

func containsPaperString(values []string, needle string) bool {
	for _, value := range values {
		if value == needle {
			return true
		}
	}
	return false
}

func parsePaperSymbols(symbolsCSV string) []string {
	var symbols []string
	for _, raw := range strings.Split(symbolsCSV, ",") {
		symbol := strings.ToUpper(strings.TrimSpace(raw))
		if symbol != "" {
			symbols = append(symbols, symbol)
		}
	}
	return symbols
}

func evaluatePaperRIF(datasetManifestPath string, allowWarnings bool) paperRIFEvidence {
	if datasetManifestPath == "" {
		return paperRIFEvidence{
			Status:       "RIF_BLOCKED",
			Warnings:     []string{"dataset manifest missing"},
			BlocksSignal: true,
			BlockReason:  "dataset manifest missing",
		}
	}

	data, err := os.ReadFile(datasetManifestPath)
	if err != nil {
		return paperRIFEvidence{
			Status:       "RIF_BLOCKED",
			Warnings:     []string{fmt.Sprintf("dataset manifest unreadable: %v", err)},
			BlocksSignal: true,
			BlockReason:  "dataset manifest unreadable",
		}
	}

	var manifest struct {
		DatasetHash string `json:"dataset_hash"`
		Hashes      struct {
			DatasetHash  string `json:"dataset_hash"`
			ManifestHash string `json:"manifest_hash"`
		} `json:"hashes"`
		Validation struct {
			Status   string   `json:"status"`
			Warnings []string `json:"warnings"`
		} `json:"validation"`
		Survivorship struct {
			UniverseHash                         string   `json:"universe_hash"`
			LifecycleHash                        string   `json:"lifecycle_hash"`
			PointInTimeCoverageHash              string   `json:"point_in_time_coverage_hash"`
			PointInTimeCoverageStatus            string   `json:"point_in_time_coverage_status"`
			PointInTimePromotionRecommendation   string   `json:"point_in_time_promotion_recommendation"`
			ExchangeMetadataSnapshotCurrentOnly  bool     `json:"exchange_metadata_snapshot_current_only"`
			ExchangeMetadataSnapshotManifestHash string   `json:"exchange_metadata_snapshot_manifest_hash"`
			Warnings                             []string `json:"warnings"`
		} `json:"survivorship"`
	}
	if err := json.Unmarshal(data, &manifest); err != nil {
		return paperRIFEvidence{
			Status:       "RIF_BLOCKED",
			Warnings:     []string{fmt.Sprintf("dataset manifest malformed: %v", err)},
			BlocksSignal: true,
			BlockReason:  "dataset manifest malformed",
		}
	}

	evidence := paperRIFEvidence{
		Status:        "RIF_PASS",
		DatasetHash:   firstNonEmpty(manifest.Hashes.DatasetHash, manifest.DatasetHash),
		ManifestHash:  manifest.Hashes.ManifestHash,
		UniverseHash:  manifest.Survivorship.UniverseHash,
		LifecycleHash: manifest.Survivorship.LifecycleHash,
		PITHash:       manifest.Survivorship.PointInTimeCoverageHash,
	}
	if evidence.DatasetHash == "" {
		evidence.Warnings = append(evidence.Warnings, "dataset hash missing")
	}
	if evidence.UniverseHash == "" {
		evidence.Warnings = append(evidence.Warnings, "universe hash missing")
	}
	if evidence.LifecycleHash == "" {
		evidence.Warnings = append(evidence.Warnings, "lifecycle hash missing")
	}
	if evidence.PITHash == "" {
		evidence.Warnings = append(evidence.Warnings, "point-in-time coverage hash missing")
	}
	if manifest.Validation.Status != "" && !strings.EqualFold(manifest.Validation.Status, "PASS") {
		evidence.Warnings = append(evidence.Warnings, "dataset validation status "+manifest.Validation.Status)
		if strings.EqualFold(manifest.Validation.Status, "FAIL") || strings.EqualFold(manifest.Validation.Status, "ERROR") {
			evidence.BlocksSignal = true
			evidence.BlockReason = "dataset validation failed"
		}
	}
	evidence.Warnings = append(evidence.Warnings, manifest.Validation.Warnings...)
	evidence.Warnings = append(evidence.Warnings, manifest.Survivorship.Warnings...)

	pitStatus := strings.ToUpper(strings.TrimSpace(manifest.Survivorship.PointInTimeCoverageStatus))
	switch pitStatus {
	case "", "UNKNOWN", "CURRENT_ONLY", "PIT_NOT_ELIGIBLE", "NOT_ELIGIBLE":
		evidence.Warnings = append(evidence.Warnings, "point-in-time coverage status "+emptyAs(pitStatus, "UNKNOWN"))
		evidence.BlocksSignal = true
		evidence.BlockReason = "point-in-time coverage not eligible"
	case "ELIGIBLE", "PIT_ELIGIBLE", "COVERS_WINDOW", "POINT_IN_TIME_ELIGIBLE":
	default:
		evidence.Warnings = append(evidence.Warnings, "unrecognized point-in-time coverage status "+pitStatus)
		if !allowWarnings {
			evidence.BlocksSignal = true
			evidence.BlockReason = "unrecognized point-in-time coverage status"
		}
	}

	if strings.EqualFold(manifest.Survivorship.PointInTimePromotionRecommendation, "BLOCK_STRICT_PROMOTION") {
		evidence.Warnings = append(evidence.Warnings, "point-in-time promotion recommendation blocks strict promotion")
		evidence.BlocksSignal = true
		evidence.BlockReason = "point-in-time promotion recommendation blocks strict promotion"
	}
	if manifest.Survivorship.ExchangeMetadataSnapshotCurrentOnly {
		evidence.Warnings = append(evidence.Warnings, "exchange metadata snapshot evidence is current-only")
		evidence.BlocksSignal = true
		evidence.BlockReason = "current-only exchange metadata snapshot"
	}
	if len(evidence.Warnings) > 0 && !allowWarnings && !evidence.BlocksSignal {
		evidence.BlocksSignal = true
		evidence.BlockReason = "RIF warnings present and allow-rif-warnings=false"
	}
	if evidence.BlocksSignal {
		evidence.Status = "RIF_BLOCKED"
	}
	return evidence
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if value != "" {
			return value
		}
	}
	return ""
}

func emptyAs(value, replacement string) string {
	if value == "" {
		return replacement
	}
	return value
}

func loadPaperReferencePrice(snapshotDir, symbol string, fallbackPrice float64) (float64, *string, []string, error) {
	if snapshotDir == "" {
		return fallbackPrice, nil, []string{"reference price defaulted because no read-only snapshot was supplied"}, nil
	}
	path, ok := resolvePaperSnapshotPath(snapshotDir, symbol)
	if !ok {
		return fallbackPrice, nil, []string{"reference price defaulted because snapshot was not found for " + symbol}, nil
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return 0, nil, nil, fmt.Errorf("read reference snapshot: %w", err)
	}
	var snap paperReferenceSnapshot
	if err := json.Unmarshal(data, &snap); err != nil {
		return 0, nil, nil, fmt.Errorf("parse reference snapshot %s: %w", path, err)
	}
	price := firstPositive(snap.ReferencePrice, snap.Price, snap.Close)
	if price <= 0 {
		return 0, nil, nil, fmt.Errorf("reference snapshot %s has no positive reference price", path)
	}
	hash := snap.SnapshotHash
	if hash == "" {
		fileHash, err := papersignal.HashFile(path)
		if err != nil {
			return 0, nil, nil, err
		}
		hash = fileHash
	}
	return price, &hash, nil, nil
}

func resolvePaperSnapshotPath(rootOrFile, symbol string) (string, bool) {
	info, err := os.Stat(rootOrFile)
	if err != nil {
		return "", false
	}
	if !info.IsDir() {
		return rootOrFile, true
	}
	candidates := []string{
		filepath.Join(rootOrFile, symbol+".json"),
		filepath.Join(rootOrFile, strings.ToLower(symbol)+".json"),
		filepath.Join(rootOrFile, symbol+"_snapshot.json"),
		filepath.Join(rootOrFile, strings.ToLower(symbol)+"_snapshot.json"),
		filepath.Join(rootOrFile, "snapshot_"+symbol+".json"),
		filepath.Join(rootOrFile, "snapshot_"+strings.ToLower(symbol)+".json"),
	}
	for _, candidate := range candidates {
		if _, err := os.Stat(candidate); err == nil {
			return candidate, true
		}
	}
	return "", false
}

func firstPositive(values ...float64) float64 {
	for _, value := range values {
		if value > 0 {
			return value
		}
	}
	return 0
}

func buildPaperSignal(meta paperCandidateMetadata, rif paperRIFEvidence, symbol, marketType, timeframe, generatedAtUTC, researchLockPath, researchLockHash string, entryPrice float64, snapshotHash *string) papersignal.PaperSignal {
	status := papersignal.StatusAllowed
	reason := "Candidate valid and RIF checks passed"
	if rif.BlocksSignal {
		status = papersignal.StatusBlockedByRIF
		reason = "RIF gates failed: " + rif.BlockReason
	}
	dueAtUTC := generatedAtUTC
	if generatedAt, err := time.Parse(time.RFC3339, generatedAtUTC); err == nil {
		dueAtUTC = generatedAt.Add(time.Duration(meta.ObservationWindowMinutes) * time.Minute).UTC().Format(time.RFC3339)
	}
	return papersignal.PaperSignal{
		SchemaVersion:       "1.0",
		SignalID:            papersignal.GenerateSignalID(meta.ID, symbol, generatedAtUTC),
		GeneratedAtUTC:      generatedAtUTC,
		CandidateID:         meta.ID,
		CandidateVersion:    meta.Version,
		CandidateHash:       meta.Hash,
		Symbol:              symbol,
		MarketType:          marketType,
		Timeframe:           timeframe,
		Side:                meta.Side,
		SignalStatus:        status,
		SignalReason:        reason,
		DataAsOfUTC:         generatedAtUTC,
		ResearchLockPath:    researchLockPath,
		ResearchLockHash:    researchLockHash,
		DatasetManifestHash: rif.DatasetHash,
		UniverseHash:        rif.UniverseHash,
		LifecycleHash:       rif.LifecycleHash,
		PitCoverageHash:     rif.PITHash,
		RIFStatus:           rif.Status,
		RIFWarnings:         append([]string(nil), rif.Warnings...),
		EntryModel:          "paper_reference_price_file",
		ExitModel:           "target_stop_first_touch",
		InvalidationModel:   "paper_stop_reference",
		ObservationWindow:   meta.ObservationWindowMinutes,
		OutcomeStatus:       papersignal.OutcomePending,
		OutcomeDueAtUTC:     dueAtUTC,
		Notes:               "Forward paper observation loop; paper-only, no execution.",
	}
}

func paperJournalRowFromSignal(sig papersignal.PaperSignal, entryPrice float64, targetBPS, stopBPS float64, snapshotHash *string) papersignal.PaperJournalRow {
	target, stop := paperTargetAndStop(sig.Side, entryPrice, targetBPS, stopBPS)
	return papersignal.PaperJournalRow{
		SignalID:             sig.SignalID,
		CandidateID:          sig.CandidateID,
		CandidateVersion:     sig.CandidateVersion,
		CandidateHash:        sig.CandidateHash,
		GeneratedAtUTC:       sig.GeneratedAtUTC,
		Symbol:               sig.Symbol,
		MarketType:           sig.MarketType,
		Timeframe:            sig.Timeframe,
		Side:                 sig.Side,
		SignalStatus:         sig.SignalStatus,
		SignalReason:         sig.SignalReason,
		EntryReferencePrice:  entryPrice,
		StopReferencePrice:   &stop,
		TargetReferencePrice: &target,
		ObservationWindow:    sig.ObservationWindow,
		OutcomeDueAtUTC:      sig.OutcomeDueAtUTC,
		OutcomeStatus:        sig.OutcomeStatus,
		SourceSnapshotHash:   snapshotHash,
		ResearchLockHash:     sig.ResearchLockHash,
		DatasetHash:          sig.DatasetManifestHash,
		UniverseHash:         sig.UniverseHash,
		PitCoverageHash:      sig.PitCoverageHash,
		RIFStatus:            sig.RIFStatus,
		RIFWarnings:          append([]string(nil), sig.RIFWarnings...),
	}
}

func paperTargetAndStop(side papersignal.SignalSide, entryPrice, targetBPS, stopBPS float64) (float64, float64) {
	if side == papersignal.SideShort {
		return entryPrice * (1 - targetBPS/10000), entryPrice * (1 + stopBPS/10000)
	}
	return entryPrice * (1 + targetBPS/10000), entryPrice * (1 - stopBPS/10000)
}
