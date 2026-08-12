package app

import (
	"encoding/json"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/david22573/ak-engine/internal/data"
	"github.com/david22573/ak-engine/internal/papersignal"
	"github.com/david22573/ak-engine/pkg/protocol"
	"github.com/spf13/cobra"
)

var (
	pfgJournal        string
	pfgMarketDataRoot string
	pfgSnapshotDir    string
	pfgOutDir         string
	pfgMaxGrade       int
	pfgNowUTC         string
)

type GradeSummary struct {
	SchemaVersion       string         `json:"schema_version"`
	GeneratedAtUTC      string         `json:"generated_at_utc"`
	JournalPath         string         `json:"journal_path"`
	GradedSignals       int            `json:"graded_signals"`
	PendingSkipped      int            `json:"pending_skipped_not_due"`
	InsufficientData    int            `json:"insufficient_data"`
	AlreadyFinal        int            `json:"already_final"`
	BlockedSkipped      int            `json:"blocked_skipped"`
	OutcomeDistribution map[string]int `json:"outcome_distribution"`
	Warnings            []string       `json:"warnings"`
}

var paperForwardGradePendingCmd = &cobra.Command{
	Use:   "paper-forward-grade-pending",
	Short: "Scan journal for PENDING outcomes and grade due paper signals",
	RunE: func(cmd *cobra.Command, args []string) error {
		if pfgJournal == "" {
			return fmt.Errorf("missing journal path")
		}
		if pfgMaxGrade < 1 {
			return fmt.Errorf("--max-grade must be >= 1")
		}

		nowUTC := time.Now().UTC()
		if pfgNowUTC != "" {
			parsed, err := time.Parse(time.RFC3339, pfgNowUTC)
			if err != nil {
				return fmt.Errorf("invalid --now-utc: %w", err)
			}
			nowUTC = parsed.UTC()
		}

		summary := GradeSummary{
			SchemaVersion:       "1.0",
			GeneratedAtUTC:      nowUTC.Format(time.RFC3339),
			JournalPath:         pfgJournal,
			OutcomeDistribution: make(map[string]int),
			Warnings:            []string{},
		}

		rows, err := papersignal.ReadJournal(pfgJournal)
		if err != nil {
			if os.IsNotExist(err) {
				summary.Warnings = append(summary.Warnings, "journal not found; nothing to grade")
				return writeGradeSummary(pfgOutDir, summary)
			}
			return err
		}

		updated := false
		gradeAttempts := 0
		for i := range rows {
			row := &rows[i]
			if row.SignalStatus != "" && papersignal.IsBlockedStatus(row.SignalStatus) {
				summary.BlockedSkipped++
				continue
			}
			if row.OutcomeStatus != papersignal.OutcomePending {
				summary.AlreadyFinal++
				continue
			}
			if err := validateCanonicalPaperJournalRow(*row); err != nil {
				return fmt.Errorf("signal %s is not a canonical paper observation: %w", row.SignalID, err)
			}
			if gradeAttempts >= pfgMaxGrade {
				break
			}
			dueTime, err := paperOutcomeDueTime(*row)
			if err != nil {
				return fmt.Errorf("signal %s has invalid due time: %w", row.SignalID, err)
			}
			if nowUTC.Before(dueTime) {
				summary.PendingSkipped++
				continue
			}

			gradeAttempts++
			candles, sourcePath, err := loadPaperMarketCandles(pfgMarketDataRoot, pfgSnapshotDir, *row)
			if err != nil {
				if os.IsNotExist(err) {
					markPaperInsufficient(row, nowUTC, "future market data unavailable")
					summary.InsufficientData++
					summary.OutcomeDistribution[string(row.OutcomeStatus)]++
					updated = true
					if err := writeOutcomeArtifact(pfgOutDir, *row); err != nil {
						return err
					}
					continue
				}
				return err
			}
			graded, err := gradePaperOutcome(*row, candles, dueTime)
			if err != nil {
				if err == errPaperInsufficientData {
					markPaperInsufficient(row, nowUTC, "future market data does not cover observation window")
					summary.InsufficientData++
					summary.OutcomeDistribution[string(row.OutcomeStatus)]++
					updated = true
					if err := writeOutcomeArtifact(pfgOutDir, *row); err != nil {
						return err
					}
					continue
				}
				return err
			}
			graded.OutcomeCheckedAtUTC = nowUTC.Format(time.RFC3339)
			if sourcePath != "" {
				if hash, err := papersignal.HashFile(sourcePath); err == nil {
					graded.SourceSnapshotHash = &hash
				}
			}
			*row = graded
			summary.GradedSignals++
			summary.OutcomeDistribution[string(row.OutcomeStatus)]++
			updated = true
			if err := writeOutcomeArtifact(pfgOutDir, *row); err != nil {
				return err
			}
		}

		if updated {
			if err := papersignal.WriteJournalAtomic(pfgJournal, rows); err != nil {
				return err
			}
		}
		return writeGradeSummary(pfgOutDir, summary)
	},
}

var errPaperInsufficientData = fmt.Errorf("insufficient paper market data")

func validateCanonicalPaperJournalRow(row papersignal.PaperJournalRow) error {
	if err := validatePaperObservationIdentity(row); err != nil {
		return err
	}
	if row.Side != papersignal.SideLong && row.Side != papersignal.SideShort {
		return fmt.Errorf("actionable side must be LONG or SHORT")
	}
	if row.EntryReferencePrice <= 0 || row.TargetReferencePrice == nil || row.StopReferencePrice == nil {
		return fmt.Errorf("canonical entry/target/stop references are required")
	}
	return nil
}

func validatePaperObservationIdentity(row papersignal.PaperJournalRow) error {
	if row.SignalID == "" || row.CandidateID == "" || row.CandidateVersion == "" || row.CandidateHash == "" || row.ConfigurationHash == "" || row.ResearchEvidenceHash == "" || row.DecisionInputHash == "" {
		return fmt.Errorf("exact signal/candidate/configuration/evidence/input identity is required")
	}
	if row.Symbol == "" || row.MarketType == "" || row.Timeframe == "" || row.DecisionTimeUTC == "" || row.FillTimeUTC == "" || row.DatasetHash == "" || row.PitCoverageHash == "" {
		return fmt.Errorf("exact scope/time/dataset/PIT identity is required")
	}
	decision, err := time.Parse(time.RFC3339Nano, row.DecisionTimeUTC)
	if err != nil {
		return fmt.Errorf("invalid decision time: %w", err)
	}
	fill, err := time.Parse(time.RFC3339Nano, row.FillTimeUTC)
	if err != nil {
		return fmt.Errorf("invalid fill time: %w", err)
	}
	if fill.UnixMilli() != decision.UnixMilli()+1 {
		return fmt.Errorf("fill is not the canonical first tradable observation")
	}
	return nil
}

func paperOutcomeDueTime(row papersignal.PaperJournalRow) (time.Time, error) {
	if row.OutcomeDueAtUTC != "" {
		return time.Parse(time.RFC3339, row.OutcomeDueAtUTC)
	}
	generatedAt, err := time.Parse(time.RFC3339, row.GeneratedAtUTC)
	if err != nil {
		return time.Time{}, err
	}
	window := row.ObservationWindow
	if window <= 0 {
		window = 60
	}
	return generatedAt.Add(time.Duration(window) * time.Minute), nil
}

func loadPaperMarketCandles(marketDataRoot, snapshotDir string, row papersignal.PaperJournalRow) ([]protocol.Candle, string, error) {
	for _, root := range []string{marketDataRoot, snapshotDir} {
		path, ok := resolvePaperMarketDataPath(root, row.Symbol, row.Timeframe)
		if !ok {
			continue
		}
		raw, err := os.ReadFile(path)
		if err != nil {
			return nil, "", err
		}
		req := data.CandleRequest{Symbol: row.Symbol, Market: firstNonEmpty(row.MarketType, "SPOT"), Interval: firstNonEmpty(row.Timeframe, "1m")}
		candles, err := data.ParseJSONCandlesNoValidate(raw, req)
		if err != nil {
			return nil, "", fmt.Errorf("parse market data %s: %w", path, err)
		}
		if err := data.ValidateCandlesForRequest(req, candles); err != nil {
			return nil, "", fmt.Errorf("validate market data %s: %w", path, err)
		}
		return candles, path, nil
	}
	return nil, "", os.ErrNotExist
}

func resolvePaperMarketDataPath(rootOrFile, symbol, timeframe string) (string, bool) {
	if rootOrFile == "" {
		return "", false
	}
	info, err := os.Stat(rootOrFile)
	if err != nil {
		return "", false
	}
	if !info.IsDir() {
		return rootOrFile, true
	}
	names := []string{
		symbol + "_" + timeframe + ".json",
		symbol + "-" + timeframe + ".json",
		symbol + ".json",
		strings.ToLower(symbol) + "_" + strings.ToLower(timeframe) + ".json",
		strings.ToLower(symbol) + "-" + strings.ToLower(timeframe) + ".json",
		strings.ToLower(symbol) + ".json",
		"market_data.json",
	}
	for _, name := range names {
		path := filepath.Join(rootOrFile, name)
		if _, err := os.Stat(path); err == nil {
			return path, true
		}
	}
	return "", false
}

func gradePaperOutcome(row papersignal.PaperJournalRow, candles []protocol.Candle, dueTime time.Time) (papersignal.PaperJournalRow, error) {
	fillTimeRaw := firstNonEmpty(row.FillTimeUTC, row.GeneratedAtUTC)
	generatedAt, err := time.Parse(time.RFC3339Nano, fillTimeRaw)
	if err != nil {
		return row, err
	}
	entry := row.EntryReferencePrice
	if entry <= 0 {
		return row, fmt.Errorf("signal %s has non-positive entry reference price", row.SignalID)
	}
	target := row.TargetReferencePrice
	stop := row.StopReferencePrice
	if target == nil || stop == nil {
		t, s := paperTargetAndStop(row.Side, entry, 100, 75)
		target = &t
		stop = &s
	}

	window := make([]protocol.Candle, 0, len(candles))
	for _, candle := range candles {
		openTime := time.UnixMilli(candle.OpenTimeMS).UTC()
		if !openTime.Before(generatedAt) && openTime.Before(dueTime) {
			window = append(window, candle)
		}
	}
	if len(window) == 0 {
		return row, errPaperInsufficientData
	}
	if window[0].OpenTimeMS != generatedAt.UnixMilli() || window[len(window)-1].CloseTimeMS != dueTime.UnixMilli()-1 {
		return row, errPaperInsufficientData
	}

	var high, low float64
	high = window[0].High
	low = window[0].Low
	for _, candle := range window {
		if candle.High > high {
			high = candle.High
		}
		if candle.Low < low {
			low = candle.Low
		}
		switch row.Side {
		case papersignal.SideShort:
			targetHit := candle.Low <= *target
			stopHit := candle.High >= *stop
			if targetHit && stopHit {
				row.OutcomeStatus = papersignal.OutcomeAmbiguousIntrabar
				row.OutcomeReason = "target and stop touched in same candle; intrabar order unknown"
				return finalizePaperExcursions(row, entry, high, low), nil
			}
			if targetHit {
				row.OutcomeStatus = papersignal.OutcomeShortTPFirst
				row.OutcomeReason = "short target touched before stop"
				return finalizePaperOutcome(row, entry, *target, high, low), nil
			}
			if stopHit {
				row.OutcomeStatus = papersignal.OutcomeShortStopFirst
				row.OutcomeReason = "short stop touched before target"
				return finalizePaperOutcome(row, entry, *stop, high, low), nil
			}
		default:
			targetHit := candle.High >= *target
			stopHit := candle.Low <= *stop
			if targetHit && stopHit {
				row.OutcomeStatus = papersignal.OutcomeAmbiguousIntrabar
				row.OutcomeReason = "target and stop touched in same candle; intrabar order unknown"
				return finalizePaperExcursions(row, entry, high, low), nil
			}
			if targetHit {
				row.OutcomeStatus = papersignal.OutcomeLongTPFirst
				row.OutcomeReason = "long target touched before stop"
				return finalizePaperOutcome(row, entry, *target, high, low), nil
			}
			if stopHit {
				row.OutcomeStatus = papersignal.OutcomeLongStopFirst
				row.OutcomeReason = "long stop touched before target"
				return finalizePaperOutcome(row, entry, *stop, high, low), nil
			}
		}
	}
	row.OutcomeStatus = papersignal.OutcomeNoEdgeChop
	row.OutcomeReason = "neither target nor stop touched before due time"
	return finalizePaperOutcome(row, entry, window[len(window)-1].Close, high, low), nil
}

func paperCandleTime(candle protocol.Candle) time.Time {
	if candle.CloseTimeMS > 0 {
		return time.UnixMilli(candle.CloseTimeMS).UTC()
	}
	return time.UnixMilli(candle.OpenTimeMS).UTC()
}

func finalizePaperOutcome(row papersignal.PaperJournalRow, entry, exit, high, low float64) papersignal.PaperJournalRow {
	ret := paperSignedReturnBPS(row.Side, entry, exit)
	row.OutcomeReturnBPS = &ret
	return finalizePaperExcursions(row, entry, high, low)
}

func finalizePaperExcursions(row papersignal.PaperJournalRow, entry, high, low float64) papersignal.PaperJournalRow {
	mfe, mae := paperExcursionsBPS(row.Side, entry, high, low)
	row.MaxFavorableExcursionBPS = &mfe
	row.MaxAdverseExcursionBPS = &mae
	return row
}

func paperSignedReturnBPS(side papersignal.SignalSide, entry, exit float64) float64 {
	if entry <= 0 {
		return 0
	}
	if side == papersignal.SideShort {
		return (entry - exit) / entry * 10000
	}
	return (exit - entry) / entry * 10000
}

func paperExcursionsBPS(side papersignal.SignalSide, entry, high, low float64) (float64, float64) {
	if entry <= 0 {
		return 0, 0
	}
	if side == papersignal.SideShort {
		return math.Max(0, (entry-low)/entry*10000), math.Max(0, (high-entry)/entry*10000)
	}
	return math.Max(0, (high-entry)/entry*10000), math.Max(0, (entry-low)/entry*10000)
}

func markPaperInsufficient(row *papersignal.PaperJournalRow, nowUTC time.Time, reason string) {
	row.OutcomeStatus = papersignal.OutcomeInsufficientData
	row.OutcomeCheckedAtUTC = nowUTC.Format(time.RFC3339)
	row.OutcomeReason = reason
}

func writeOutcomeArtifact(outDir string, row papersignal.PaperJournalRow) error {
	if outDir == "" {
		return nil
	}
	if err := os.MkdirAll(outDir, 0755); err != nil {
		return err
	}
	stem := filepath.Join(outDir, row.SignalID+"_outcome")
	data, err := json.MarshalIndent(row, "", "  ")
	if err != nil {
		return err
	}
	if err := os.WriteFile(stem+".json", data, 0644); err != nil {
		return err
	}
	md := fmt.Sprintf("# Paper Outcome: %s\n\n- Outcome: `%s`\n- Reason: %s\n", row.SignalID, row.OutcomeStatus, row.OutcomeReason)
	return os.WriteFile(stem+".md", []byte(md), 0644)
}

func writeGradeSummary(outDir string, summary GradeSummary) error {
	if outDir == "" {
		return nil
	}
	if err := os.MkdirAll(outDir, 0755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(summary, "", "  ")
	if err != nil {
		return err
	}
	if err := os.WriteFile(filepath.Join(outDir, "grading_summary.json"), data, 0644); err != nil {
		return err
	}
	var md strings.Builder
	md.WriteString("# Forward Paper Pending Grading Summary\n\n")
	md.WriteString(fmt.Sprintf("- Graded signals: %d\n", summary.GradedSignals))
	md.WriteString(fmt.Sprintf("- Pending not due: %d\n", summary.PendingSkipped))
	md.WriteString(fmt.Sprintf("- Insufficient data: %d\n", summary.InsufficientData))
	md.WriteString(fmt.Sprintf("- Blocked skipped: %d\n", summary.BlockedSkipped))
	return os.WriteFile(filepath.Join(outDir, "grading_summary.md"), []byte(md.String()), 0644)
}

func init() {
	paperForwardGradePendingCmd.Flags().StringVar(&pfgJournal, "journal", "runs/paper/signals/paper_signal_journal.jsonl", "Journal path")
	paperForwardGradePendingCmd.Flags().StringVar(&pfgMarketDataRoot, "market-data-root", "", "Read-only local market data root or JSON file")
	paperForwardGradePendingCmd.Flags().StringVar(&pfgSnapshotDir, "snapshot-dir", "", "Read-only snapshot directory or JSON file")
	paperForwardGradePendingCmd.Flags().StringVar(&pfgOutDir, "out-dir", "runs/paper/outcomes", "Output directory")
	paperForwardGradePendingCmd.Flags().IntVar(&pfgMaxGrade, "max-grade", 50, "Max due signals to grade")
	paperForwardGradePendingCmd.Flags().StringVar(&pfgNowUTC, "now-utc", "", "Override current time for deterministic paper replay")
	rootCmd.AddCommand(paperForwardGradePendingCmd)
}
