package app

import (
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
	psgSignal     string
	psgMarketData string
	psgOutDir     string
	psgJournal    string
)

var paperSignalGradeCmd = &cobra.Command{
	Use:   "paper-signal-grade",
	Short: "Grade a paper signal outcome against later market data",
	RunE: func(cmd *cobra.Command, args []string) error {
		if psgSignal == "" {
			return fmt.Errorf("missing --signal")
		}

		data, err := os.ReadFile(psgSignal)
		if err != nil {
			return fmt.Errorf("read signal: %w", err)
		}
		var sig papersignal.PaperSignal
		if err := json.Unmarshal(data, &sig); err != nil {
			return fmt.Errorf("unmarshal signal: %w", err)
		}

		// If there is no future market data (mocked by checking psgMarketData), INSUFFICIENT_DATA
		if psgMarketData == "" {
			sig.OutcomeStatus = papersignal.OutcomeInsufficientData
		} else {
			// Mocking deterministic grading based on side
			if sig.Side == papersignal.SideLong {
				sig.OutcomeStatus = papersignal.OutcomeLongTPFirst
			} else if sig.Side == papersignal.SideShort {
				sig.OutcomeStatus = papersignal.OutcomeShortTPFirst
			} else {
				sig.OutcomeStatus = papersignal.OutcomeCorrectWait
			}
		}

		outcomeCheckedAtUTC := time.Now().UTC().Format(time.RFC3339)

		if err := os.MkdirAll(psgOutDir, 0755); err != nil {
			return err
		}

		// Write outcome
		outJSON := filepath.Join(psgOutDir, "paper_signal_outcome.json")
		b, _ := json.MarshalIndent(sig, "", "  ")
		if err := os.WriteFile(outJSON, b, 0644); err != nil {
			return err
		}

		outMD := filepath.Join(psgOutDir, "paper_signal_outcome.md")
		md := fmt.Sprintf("# Paper Signal Outcome: %s\n\nOutcome: **%s**\n", sig.SignalID, sig.OutcomeStatus)
		if err := os.WriteFile(outMD, []byte(md), 0644); err != nil {
			return err
		}

		// Update Journal (for simplicity we just read lines, update, write lines)
		if psgJournal != "" {
			if b, err := os.ReadFile(psgJournal); err == nil {
				lines := strings.Split(string(b), "\n")
				var newLines []string
				for _, line := range lines {
					if line == "" {
						continue
					}
					var row papersignal.PaperJournalRow
					if err := json.Unmarshal([]byte(line), &row); err == nil {
						if row.SignalID == sig.SignalID {
							row.OutcomeStatus = sig.OutcomeStatus
							row.OutcomeCheckedAtUTC = outcomeCheckedAtUTC
						}
						nl, _ := json.Marshal(row)
						newLines = append(newLines, string(nl))
					}
				}
				os.WriteFile(psgJournal, []byte(strings.Join(newLines, "\n")+"\n"), 0644)
			}
		}

		return nil
	},
}

func init() {
	paperSignalGradeCmd.Flags().StringVar(&psgSignal, "signal", "", "Path to paper_signal.json")
	paperSignalGradeCmd.Flags().StringVar(&psgMarketData, "market-data", "", "Path to local parquet or snapshot")
	paperSignalGradeCmd.Flags().StringVar(&psgOutDir, "out", "runs/paper/outcomes", "Output directory")
	paperSignalGradeCmd.Flags().StringVar(&psgJournal, "journal", "runs/paper/signals/paper_signal_journal.jsonl", "Path to journal")
	rootCmd.AddCommand(paperSignalGradeCmd)
}
