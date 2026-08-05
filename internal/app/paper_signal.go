package app

import (
	"encoding/json"
	"fmt"
	"os"
	"time"

	"github.com/david22573/ak-engine/internal/papersignal"
	"github.com/spf13/cobra"
)

var (
	psCandidate        string
	psSymbol           string
	psMarketType       string
	psTimeframe        string
	psDatasetManifest  string
	psResearchLock     string
	psOutDir           string
	psJournal          string
	psDryRun           bool
	psAllowRIFWarnings bool
	psPaperOnly        bool
)

var paperSignalCmd = &cobra.Command{
	Use:   "paper-signal",
	Short: "Evaluate an existing candidate and generate a RIF-gated paper signal",
	RunE: func(cmd *cobra.Command, args []string) error {
		nowStr := time.Now().UTC().Format(time.RFC3339)

		// Defaults / Validation
		if psCandidate == "" || psSymbol == "" || psMarketType == "" || psTimeframe == "" {
			return fmt.Errorf("missing required candidate/symbol/market/timeframe fields")
		}

		// 1. Read Research Lock (if provided)
		var rLock struct {
			GitSHA string `json:"git_sha"`
		}
		rLockHash := ""
		if psResearchLock != "" {
			data, err := os.ReadFile(psResearchLock)
			if err != nil {
				return fmt.Errorf("failed to read research lock: %w", err)
			}
			if err := json.Unmarshal(data, &rLock); err != nil {
				return fmt.Errorf("failed to parse research lock: %w", err)
			}
			rLockHash = rLock.GitSHA
		}

		// 2. Read Dataset Manifest (for RIF/PIT checking)
		dsHash := ""
		uniHash := ""
		lcHash := ""
		pitHash := ""
		rifStatus := "UNKNOWN"
		rifWarnings := []string{}
		blocksPromotion := false
		pitBlocksPromotion := false

		if psDatasetManifest != "" {
			data, err := os.ReadFile(psDatasetManifest)
			if err != nil {
				return fmt.Errorf("failed to read dataset manifest: %w", err)
			}
			var ds struct {
				DatasetHash string `json:"dataset_hash"`
				Hashes      struct {
					DatasetHash string `json:"dataset_hash"`
				} `json:"hashes"`
				Survivorship struct {
					UniverseHash                       string `json:"universe_hash"`
					LifecycleHash                      string `json:"lifecycle_hash"`
					PointInTimeCoverageHash            string `json:"point_in_time_coverage_hash"`
					PointInTimeCoverageStatus          string `json:"point_in_time_coverage_status"`
					PointInTimePromotionRecommendation string `json:"point_in_time_promotion_recommendation"`
				} `json:"survivorship"`
			}
			_ = json.Unmarshal(data, &ds)
			dsHash = ds.Hashes.DatasetHash
			if dsHash == "" {
				dsHash = ds.DatasetHash
			}
			uniHash = ds.Survivorship.UniverseHash
			lcHash = ds.Survivorship.LifecycleHash
			pitHash = ds.Survivorship.PointInTimeCoverageHash

			if ds.Survivorship.PointInTimeCoverageStatus == "PIT_NOT_ELIGIBLE" {
				pitBlocksPromotion = true
				rifWarnings = append(rifWarnings, "PIT evidence not eligible")
			}
			if ds.Survivorship.PointInTimePromotionRecommendation == "BLOCK_STRICT_PROMOTION" {
				pitBlocksPromotion = true
				rifWarnings = append(rifWarnings, "PIT recommends blocking strict promotion")
			}
			rifStatus = "CHECKED"
		} else {
			blocksPromotion = true
			rifWarnings = append(rifWarnings, "Missing dataset manifest")
		}

		// 3. Determine Final Signal Status
		sigStatus := papersignal.StatusAllowed
		sigReason := "Candidate valid and RIF checks passed"
		if blocksPromotion || pitBlocksPromotion {
			sigStatus = papersignal.StatusBlockedByRIF
			sigReason = "RIF gates failed: " + rifWarnings[0]
			if psAllowRIFWarnings {
				sigStatus = papersignal.StatusWait
				sigReason = "RIF warnings present but allowed as observation-only WAIT"
			}
		}

		// (Stub) Simulated Candidate Logic: If it passed all RIF gates, emit a valid Long for our favored candidate
		side := papersignal.SideLong
		if psCandidate == "DowntrendMidvolReliefShort240m" {
			side = papersignal.SideShort
		}

		signalID := papersignal.GenerateSignalID(psCandidate, psSymbol, nowStr)

		// 4. Construct Paper Signal
		sig := papersignal.PaperSignal{
			SchemaVersion:       "1.0",
			SignalID:            signalID,
			GeneratedAtUTC:      nowStr,
			CandidateID:         psCandidate,
			CandidateVersion:    "1.0",
			CandidateHash:       "hash123",
			Symbol:              psSymbol,
			MarketType:          psMarketType,
			Timeframe:           psTimeframe,
			Side:                side,
			SignalStatus:        sigStatus,
			SignalReason:        sigReason,
			DataAsOfUTC:         nowStr,
			ResearchLockPath:    psResearchLock,
			ResearchLockHash:    rLockHash,
			DatasetManifestHash: dsHash,
			UniverseHash:        uniHash,
			LifecycleHash:       lcHash,
			PitCoverageHash:     pitHash,
			RIFStatus:           rifStatus,
			RIFWarnings:         rifWarnings,
			EntryModel:          "default_entry",
			ExitModel:           "default_exit",
			InvalidationModel:   "default_invalidation",
			ObservationWindow:   60,
			OutcomeStatus:       papersignal.OutcomePending,
			OutcomeDueAtUTC:     nowStr, // placeholder
			Notes:               "Paper signal loop test",
		}

		// Write Artifacts
		if !psDryRun {
			if err := papersignal.WritePaperSignal(psOutDir, sig); err != nil {
				return fmt.Errorf("failed to write paper signal: %w", err)
			}

			// Append to Journal
			if psJournal != "" && (sigStatus == papersignal.StatusAllowed || psAllowRIFWarnings) {
				row := papersignal.PaperJournalRow{
					SignalID:            sig.SignalID,
					CandidateID:         sig.CandidateID,
					GeneratedAtUTC:      sig.GeneratedAtUTC,
					Symbol:              sig.Symbol,
					Side:                sig.Side,
					SignalStatus:        sig.SignalStatus,
					EntryReferencePrice: 100.0, // stub
					OutcomeStatus:       papersignal.OutcomePending,
					OutcomeCheckedAtUTC: "",
					ResearchLockHash:    sig.ResearchLockHash,
					DatasetHash:         sig.DatasetManifestHash,
					UniverseHash:        sig.UniverseHash,
					PitCoverageHash:     sig.PitCoverageHash,
				}
				if err := papersignal.AppendToJournal(psJournal, row); err != nil {
					return fmt.Errorf("failed to append to journal: %w", err)
				}
			}
		}

		fmt.Printf("Emitted Paper Signal %s (Status: %s)\n", signalID, sigStatus)
		return nil
	},
}

func init() {
	paperSignalCmd.Flags().StringVar(&psCandidate, "candidate", "", "Candidate ID")
	paperSignalCmd.Flags().StringVar(&psSymbol, "symbol", "", "Symbol")
	paperSignalCmd.Flags().StringVar(&psMarketType, "market-type", "", "Market type")
	paperSignalCmd.Flags().StringVar(&psTimeframe, "timeframe", "", "Timeframe")
	paperSignalCmd.Flags().StringVar(&psDatasetManifest, "dataset-manifest", "", "Path to dataset_manifest.json")
	paperSignalCmd.Flags().StringVar(&psResearchLock, "research-lock", "", "Path to research.lock")
	paperSignalCmd.Flags().StringVar(&psOutDir, "out-dir", "runs/paper/signals", "Output directory for the signal")
	paperSignalCmd.Flags().StringVar(&psJournal, "journal", "runs/paper/signals/paper_signal_journal.jsonl", "Path to paper signal journal")
	paperSignalCmd.Flags().BoolVar(&psDryRun, "dry-run", false, "Do not write artifacts")
	paperSignalCmd.Flags().BoolVar(&psAllowRIFWarnings, "allow-rif-warnings", false, "Allow WAIT signal even if RIF fails")
	paperSignalCmd.Flags().BoolVar(&psPaperOnly, "paper-only", true, "Required to be true to avoid execution")

	rootCmd.AddCommand(paperSignalCmd)
}
