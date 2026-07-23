package app

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/david22573/ak-engine/internal/papersignal"
	"github.com/spf13/cobra"
)

var (
	psrJournal string
	psrOut     string
	psrJsonOut string
)

type ReviewReport struct {
	TotalSignals      int            `json:"total_signals"`
	AllowedSignals    int            `json:"allowed_signals"`
	BlockedSignals    int            `json:"blocked_signals"`
	PendingSignals    int            `json:"pending_signals"`
	GradedSignals     int            `json:"graded_signals"`
	SampleSizeWarning string         `json:"sample_size_warning"`
	OutcomeDist       map[string]int `json:"outcome_distribution"`
	SymbolSummary     map[string]int `json:"symbol_summary"`
	RIFBlockReasons   map[string]int `json:"rif_block_reasons"`
}

var paperSignalReviewCmd = &cobra.Command{
	Use:   "paper-signal-review",
	Short: "Review the paper signal journal and generate outcome reports",
	RunE: func(cmd *cobra.Command, args []string) error {
		rep := ReviewReport{
			OutcomeDist:     make(map[string]int),
			SymbolSummary:   make(map[string]int),
			RIFBlockReasons: make(map[string]int),
		}

		rows, err := papersignal.ReadJournal(psrJournal)
		if err != nil {
			if !os.IsNotExist(err) {
				return err
			}
		}
		for _, row := range rows {
			rep.TotalSignals++
			if papersignal.IsBlockedStatus(row.SignalStatus) {
				rep.BlockedSignals++
				rep.RIFBlockReasons[firstNonEmpty(row.SignalReason, string(row.SignalStatus))]++
			} else if row.SignalStatus == papersignal.StatusAllowed || row.SignalStatus == "" {
				rep.AllowedSignals++
			}

			switch row.OutcomeStatus {
			case papersignal.OutcomePending:
				rep.PendingSignals++
			case "":
			default:
				rep.GradedSignals++
				rep.OutcomeDist[string(row.OutcomeStatus)]++
			}
			rep.SymbolSummary[row.Symbol]++
		}

		if rep.GradedSignals < 30 {
			rep.SampleSizeWarning = "PAPER_INSUFFICIENT_SAMPLE"
		} else if rep.GradedSignals < 100 {
			rep.SampleSizeWarning = "PAPER_EARLY_SAMPLE"
		} else {
			rep.SampleSizeWarning = "PAPER_CALIBRATION_READY"
		}

		jb, _ := json.MarshalIndent(rep, "", "  ")
		if err := os.MkdirAll(filepath.Dir(psrJsonOut), 0755); err != nil {
			return err
		}
		os.WriteFile(psrJsonOut, jb, 0644)

		var sb strings.Builder
		sb.WriteString("# Paper Signal Review\n\n")
		sb.WriteString(fmt.Sprintf("- Total: %d\n", rep.TotalSignals))
		sb.WriteString(fmt.Sprintf("- Allowed: %d\n", rep.AllowedSignals))
		sb.WriteString(fmt.Sprintf("- Graded: %d\n", rep.GradedSignals))
		sb.WriteString(fmt.Sprintf("- Sample Size Warning: **%s**\n", rep.SampleSizeWarning))
		if err := os.MkdirAll(filepath.Dir(psrOut), 0755); err != nil {
			return err
		}
		os.WriteFile(psrOut, []byte(sb.String()), 0644)

		return nil
	},
}

func init() {
	paperSignalReviewCmd.Flags().StringVar(&psrJournal, "journal", "runs/paper/signals/paper_signal_journal.jsonl", "Path to journal")
	paperSignalReviewCmd.Flags().StringVar(&psrOut, "out", "runs/reports/paper_signal_review.md", "MD output")
	paperSignalReviewCmd.Flags().StringVar(&psrJsonOut, "json-out", "runs/reports/paper_signal_review.json", "JSON output")
	rootCmd.AddCommand(paperSignalReviewCmd)
}
