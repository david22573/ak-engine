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
	pfResearchEvidence string
	pfDecisionInput    string
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
		if pfMarketType == "" {
			pfMarketType = "SPOT"
		}

		symbols := parsePaperSymbols(pfSymbols)
		if len(symbols) != 1 {
			return fmt.Errorf("canonical paper-forward requires exactly one symbol/evidence/input identity per run")
		}
		if pfDatasetManifest != "" {
			return fmt.Errorf("legacy --dataset-manifest is diagnostic-only and cannot authorize canonical paper-forward")
		}

		generatedAt := time.Now().UTC()
		if pfGeneratedAtUTC != "" {
			parsed, err := time.Parse(time.RFC3339, pfGeneratedAtUTC)
			if err != nil {
				return fmt.Errorf("invalid --generated-at-utc: %w", err)
			}
			generatedAt = parsed.UTC()
		}
		generatedAtStr := generatedAt.Format(time.RFC3339Nano)
		evidence, err := loadPaperCanonicalEvidence(pfResearchEvidence, pfCandidate, symbols[0], pfMarketType, pfTimeframe)
		if err != nil {
			return err
		}
		meta, err := paperMetadataFromEvidence(evidence)
		if err != nil {
			return err
		}
		decisionPath := pfDecisionInput
		if decisionPath == "" {
			var ok bool
			decisionPath, ok = resolvePaperSnapshotPath(pfSnapshotDir, symbols[0])
			if !ok {
				return fmt.Errorf("canonical paper decision input not found for %s", symbols[0])
			}
		}
		decision, err := loadPaperDecision(decisionPath, evidence, generatedAt)
		if err != nil {
			return err
		}
		runID := deterministicPaperRunID(pfCandidate, symbols, pfTimeframe, generatedAtStr, mode, evidence.EvidenceHash+decision.InputHash)

		researchLockHash := ""
		if pfResearchLock != "" {
			hash, err := papersignal.HashFile(pfResearchLock)
			if err != nil {
				return fmt.Errorf("failed to hash research lock: %w", err)
			}
			researchLockHash = hash
		}

		hashes := map[string]string{
			"research_diagnostic": evidence.DiagnosticHash,
			"research_evidence":   evidence.EvidenceHash,
			"candidate":           evidence.Candidate.ArtifactHash,
			"configuration":       evidence.Configuration.ArtifactHash,
			"dataset":             evidence.DatasetHash,
			"pit":                 evidence.PITHash,
			"decision_input":      decision.InputHash,
		}
		if researchLockHash != "" {
			hashes["research_lock"] = researchLockHash
		}

		runArtifact := papersignal.ForwardObservationRun{
			SchemaVersion:        "2.0",
			RunID:                runID,
			GeneratedAtUTC:       generatedAtStr,
			Mode:                 mode,
			Candidates:           []string{pfCandidate},
			Symbols:              symbols,
			Timeframes:           []string{pfTimeframe},
			ResearchEvidencePath: pfResearchEvidence,
			RIFStatus:            "CANONICAL_RESEARCH_EVIDENCE_VALIDATED_RESEARCH_ONLY",
			JournalPath:          pfJournal,
			Warnings:             []string{},
			Hashes:               hashes,
		}

		sig := buildCanonicalPaperSignal(meta, evidence, decision, symbols[0], pfMarketType, pfTimeframe, generatedAtStr)
		row := canonicalPaperJournalRow(sig, decision, meta.TargetBPS, meta.StopBPS)
		runArtifact.GeneratedSignals = 1
		if papersignal.IsActionableStatus(sig.SignalStatus) {
			runArtifact.AllowedSignals = 1
			runArtifact.PendingOutcomes = 1
		} else {
			runArtifact.WaitObservations = 1
		}
		if !pfDryRun {
			if pfJournal != "" {
				if err := validateCanonicalPaperJournalDestination(pfJournal, row); err != nil {
					return err
				}
			}
			if err := papersignal.WritePaperSignal(pfOutDir, sig); err != nil {
				return fmt.Errorf("failed to write paper signal: %w", err)
			}
			if pfJournal != "" {
				if err := appendCanonicalPaperObservation(pfJournal, row); err != nil {
					return fmt.Errorf("failed to append to journal: %w", err)
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
	paperForwardCmd.Flags().StringVar(&pfResearchEvidence, "research-evidence", "", "Canonical Engine research diagnostic containing exact candidate evidence")
	paperForwardCmd.Flags().StringVar(&pfDecisionInput, "decision-input", "", "Versioned local as-of decision input JSON")
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
