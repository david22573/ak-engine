package app

import (
	"fmt"
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
	Short: "Retired compatibility command; use paper-forward-grade-pending",
	RunE: func(cmd *cobra.Command, args []string) error {
		return fmt.Errorf("paper-signal-grade is retired because it did not execute the canonical grading model; use paper-forward-grade-pending")
	},
}

func init() {
	paperSignalGradeCmd.Flags().StringVar(&psgSignal, "signal", "", "Path to paper_signal.json")
	paperSignalGradeCmd.Flags().StringVar(&psgMarketData, "market-data", "", "Path to local parquet or snapshot")
	paperSignalGradeCmd.Flags().StringVar(&psgOutDir, "out", "runs/paper/outcomes", "Output directory")
	paperSignalGradeCmd.Flags().StringVar(&psgJournal, "journal", "runs/paper/signals/paper_signal_journal.jsonl", "Path to journal")
	rootCmd.AddCommand(paperSignalGradeCmd)
}
