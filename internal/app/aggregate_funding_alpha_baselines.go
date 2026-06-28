package app

import (
	"fmt"
	"path/filepath"

	"github.com/spf13/cobra"
)

var (
	afabChunks  string
	afabReports string
	afabSymbols string
	afabFrom    string
	afabTo      string
)

var aggregateFundingAlphaBaselinesCmd = &cobra.Command{
	Use:   "aggregate-funding-alpha-baselines",
	Short: "Aggregate real funding event JSONL baselines",
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg := fundingAggregationConfig{
			Symbols:    parseFundingSymbols(afabSymbols),
			Months:     parseFundingMonths(afabFrom, afabTo),
			ChunksDir:  afabChunks,
			ReportsDir: afabReports,
		}
		report, join, integrity, err := buildFundingAggregationReports(cfg)
		if err != nil {
			return err
		}
		if err := writeFundingAggregationReports(cfg, report, join, integrity); err != nil {
			return err
		}
		fmt.Printf("Aggregated %d event rows from %d files.\n", report.Summary.TotalEventRows, report.Summary.EventFilesFound)
		return nil
	},
}

func init() {
	aggregateFundingAlphaBaselinesCmd.Flags().StringVar(&afabChunks, "chunks", filepath.Join("runs", "reports", "chunks"), "chunks directory")
	aggregateFundingAlphaBaselinesCmd.Flags().StringVar(&afabReports, "reports-dir", filepath.Join("runs", "reports"), "reports output directory")
	aggregateFundingAlphaBaselinesCmd.Flags().StringVar(&afabSymbols, "symbols", "", "comma-separated symbols")
	aggregateFundingAlphaBaselinesCmd.Flags().StringVar(&afabFrom, "from", "", "from month YYYY-MM")
	aggregateFundingAlphaBaselinesCmd.Flags().StringVar(&afabTo, "to", "", "to month YYYY-MM")
	rootCmd.AddCommand(aggregateFundingAlphaBaselinesCmd)
}
