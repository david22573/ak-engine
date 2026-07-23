package app

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/david22573/ak-engine/internal/papersignal"
	"github.com/spf13/cobra"
)

var (
	pfCandidate        string
	pfSymbols          string
	pfTimeframe        string
	pfMarketType       string
	pfDatasetManifest  string
	pfResearchLock     string
	pfSnapshotDir      string
	pfOutDir           string
	pfJournal          string
	pfMode             string
	pfMaxSignals       int
	pfGeneratedAtUTC   string
	pfDryRun           bool
	pfAllowRIFWarnings bool
	pfPaperOnly        bool
)

var paperForwardCmd = &cobra.Command{
	Use:   "paper-forward",
	Short: "Run the forward paper observation loop for an existing candidate",
	RunE: func(cmd *cobra.Command, args []string) error {
		if !pfPaperOnly {
			return fmt.Errorf("--paper-only must remain true")
		}
		mode := pfMode
		if pfDryRun {
			mode = papersignal.ModeDryRun
		}
		if !papersignal.IsAllowedMode(mode) {
			return fmt.Errorf("invalid mode %q; allowed values: %s, %s, %s", mode, papersignal.ModeDryRun, papersignal.ModePaperForward, papersignal.ModePaperReplay)
		}
		if pfCandidate == "" || pfSymbols == "" || pfTimeframe == "" {
			return fmt.Errorf("missing required candidate/symbols/timeframe fields")
		}
		if pfMaxSignals < 1 {
			return fmt.Errorf("--max-signals must be >= 1")
		}
		if pfMarketType == "" {
			pfMarketType = "SPOT"
		}

		symbols := parsePaperSymbols(pfSymbols)
		if len(symbols) == 0 {
			return fmt.Errorf("no valid symbols supplied")
		}
		meta, err := loadPaperCandidateMetadata(pfCandidate, pfTimeframe)
		if err != nil {
			return err
		}

		generatedAt := time.Now().UTC()
		if pfGeneratedAtUTC != "" {
			parsed, err := time.Parse(time.RFC3339, pfGeneratedAtUTC)
			if err != nil {
				return fmt.Errorf("invalid --generated-at-utc: %w", err)
			}
			generatedAt = parsed.UTC()
		}
		generatedAtStr := generatedAt.Format(time.RFC3339)
		runID := deterministicPaperRunID(pfCandidate, symbols, pfTimeframe, generatedAtStr, mode, pfDatasetManifest)

		researchLockHash := ""
		if pfResearchLock != "" {
			hash, err := papersignal.HashFile(pfResearchLock)
			if err != nil {
				return fmt.Errorf("failed to hash research lock: %w", err)
			}
			researchLockHash = hash
		}

		rif := evaluatePaperRIF(pfDatasetManifest, pfAllowRIFWarnings)
		hashes := map[string]string{}
		if pfDatasetManifest != "" {
			if hash, err := papersignal.HashFile(pfDatasetManifest); err == nil {
				hashes["dataset_manifest"] = hash
			}
		}
		if rif.DatasetHash != "" {
			hashes["dataset"] = rif.DatasetHash
		}
		if rif.ManifestHash != "" {
			hashes["manifest"] = rif.ManifestHash
		}
		if researchLockHash != "" {
			hashes["research_lock"] = researchLockHash
		}

		runArtifact := papersignal.ForwardObservationRun{
			SchemaVersion:       "1.0",
			RunID:               runID,
			GeneratedAtUTC:      generatedAtStr,
			Mode:                mode,
			Candidates:          []string{pfCandidate},
			Symbols:             symbols,
			Timeframes:          []string{pfTimeframe},
			DatasetManifestPath: pfDatasetManifest,
			RIFStatus:           rif.Status,
			JournalPath:         pfJournal,
			Warnings:            append([]string(nil), rif.Warnings...),
			Hashes:              hashes,
		}

		limit := pfMaxSignals
		if limit > len(symbols) {
			limit = len(symbols)
		}
		for _, symbol := range symbols[:limit] {
			entryPrice, snapshotHash, priceWarnings, err := loadPaperReferencePrice(pfSnapshotDir, symbol, 100.0)
			if err != nil {
				return err
			}
			runArtifact.Warnings = append(runArtifact.Warnings, priceWarnings...)
			sig := buildPaperSignal(meta, rif, symbol, pfMarketType, pfTimeframe, generatedAtStr, pfResearchLock, researchLockHash, entryPrice, snapshotHash)
			sig.RIFWarnings = append(sig.RIFWarnings, priceWarnings...)

			row := paperJournalRowFromSignal(sig, entryPrice, meta.TargetBPS, meta.StopBPS, snapshotHash)
			if !papersignal.IsActionableStatus(sig.SignalStatus) {
				row.OutcomeStatus = ""
				row.OutcomeDueAtUTC = ""
			}

			runArtifact.GeneratedSignals++
			if papersignal.IsActionableStatus(sig.SignalStatus) {
				runArtifact.AllowedSignals++
				runArtifact.PendingOutcomes++
			}
			if papersignal.IsBlockedStatus(sig.SignalStatus) {
				runArtifact.BlockedSignals++
			}

			if !pfDryRun {
				if err := papersignal.WritePaperSignal(pfOutDir, sig); err != nil {
					return fmt.Errorf("failed to write paper signal: %w", err)
				}
				if pfJournal != "" {
					if err := papersignal.AppendToJournal(pfJournal, row); err != nil {
						return fmt.Errorf("failed to append to journal: %w", err)
					}
				}
			}
		}

		runArtifact.Warnings = paperUniqueStrings(runArtifact.Warnings)
		if !pfDryRun {
			if err := os.MkdirAll(pfOutDir, 0755); err != nil {
				return err
			}
			runArtifactPath := filepath.Join(pfOutDir, "forward_paper_observation_run.json")
			data, err := json.MarshalIndent(runArtifact, "", "  ")
			if err != nil {
				return err
			}
			if err := os.WriteFile(runArtifactPath, data, 0644); err != nil {
				return err
			}
		}

		fmt.Printf("Completed Paper Forward Run %s: generated=%d allowed=%d blocked=%d\n", runID, runArtifact.GeneratedSignals, runArtifact.AllowedSignals, runArtifact.BlockedSignals)
		return nil
	},
}

func deterministicPaperRunID(candidateID string, symbols []string, timeframe, generatedAtUTC, mode, datasetManifestPath string) string {
	parts := []string{candidateID, timeframe, generatedAtUTC, mode, datasetManifestPath}
	parts = append(parts, symbols...)
	sum := sha256.Sum256([]byte(strings.Join(parts, "|")))
	return "paper-" + hex.EncodeToString(sum[:])[:16]
}

func paperUniqueStrings(values []string) []string {
	seen := map[string]struct{}{}
	out := make([]string, 0, len(values))
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" {
			continue
		}
		if _, ok := seen[value]; ok {
			continue
		}
		seen[value] = struct{}{}
		out = append(out, value)
	}
	return out
}

func init() {
	paperForwardCmd.Flags().StringVar(&pfCandidate, "candidate", "", "Candidate ID")
	paperForwardCmd.Flags().StringVar(&pfSymbols, "symbols", "", "Comma-separated symbols")
	paperForwardCmd.Flags().StringVar(&pfTimeframe, "timeframe", "", "Timeframe")
	paperForwardCmd.Flags().StringVar(&pfMarketType, "market-type", "SPOT", "Market type")
	paperForwardCmd.Flags().StringVar(&pfDatasetManifest, "dataset-manifest", "", "Path to dataset manifest")
	paperForwardCmd.Flags().StringVar(&pfResearchLock, "research-lock", "", "Path to research lock (optional)")
	paperForwardCmd.Flags().StringVar(&pfSnapshotDir, "snapshot-dir", "", "Read-only snapshot directory or file for reference prices")
	paperForwardCmd.Flags().StringVar(&pfOutDir, "out-dir", "runs/paper/forward", "Output directory")
	paperForwardCmd.Flags().StringVar(&pfJournal, "journal", "runs/paper/signals/paper_signal_journal.jsonl", "Journal path")
	paperForwardCmd.Flags().StringVar(&pfMode, "mode", papersignal.ModePaperForward, "Mode: DRY_RUN, PAPER_FORWARD, or PAPER_REPLAY")
	paperForwardCmd.Flags().IntVar(&pfMaxSignals, "max-signals", 1, "Max signals to generate")
	paperForwardCmd.Flags().StringVar(&pfGeneratedAtUTC, "generated-at-utc", "", "Override generation time for deterministic paper replay")
	paperForwardCmd.Flags().BoolVar(&pfDryRun, "dry-run", false, "Do not write output")
	paperForwardCmd.Flags().BoolVar(&pfAllowRIFWarnings, "allow-rif-warnings", false, "Allow non-blocking RIF warnings")
	paperForwardCmd.Flags().BoolVar(&pfPaperOnly, "paper-only", true, "Required true for safety")

	rootCmd.AddCommand(paperForwardCmd)
}
