package app

import (
	"fmt"
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
	Short: "Retired compatibility command; use paper-forward",
	RunE: func(cmd *cobra.Command, args []string) error {
		return fmt.Errorf("paper-signal is retired because it fabricated candidate decisions and prices; use canonical paper-forward")
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
