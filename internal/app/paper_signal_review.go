package app

import (
	"fmt"
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
	Short: "Retired compatibility command; use paper-shadow-readiness",
	RunE: func(cmd *cobra.Command, args []string) error {
		return fmt.Errorf("paper-signal-review is retired because it did not isolate candidate/version/config/evidence identity; use paper-shadow-readiness")
	},
}

func init() {
	paperSignalReviewCmd.Flags().StringVar(&psrJournal, "journal", "runs/paper/signals/paper_signal_journal.jsonl", "Path to journal")
	paperSignalReviewCmd.Flags().StringVar(&psrOut, "out", "runs/reports/paper_signal_review.md", "MD output")
	paperSignalReviewCmd.Flags().StringVar(&psrJsonOut, "json-out", "runs/reports/paper_signal_review.json", "JSON output")
	rootCmd.AddCommand(paperSignalReviewCmd)
}
