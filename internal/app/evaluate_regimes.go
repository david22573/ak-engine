package app

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/davidmiguel22573/ak-engine/internal/regime"
	"github.com/davidmiguel22573/ak-engine/internal/research"
	"github.com/spf13/cobra"
)

var (
	erRegimes string
	erOut     string
)

var evaluateRegimesCmd = &cobra.Command{
	Use:   "evaluate-regimes",
	Short: "Evaluate classified regimes distribution and transitions",
	RunE: func(cmd *cobra.Command, args []string) error {
		// Read regimes
		var labels []regime.Label
		var err error
		if strings.HasSuffix(erRegimes, ".json") {
			labels, err = regime.ReadLabelsJSON(erRegimes)
		} else if strings.HasSuffix(erRegimes, ".csv") {
			labels, err = regime.ReadLabelsCSV(erRegimes)
		} else {
			return fmt.Errorf("unsupported regimes file format (must end with .json or .csv)")
		}

		if err != nil {
			return fmt.Errorf("failed to read regimes: %w", err)
		}

		// Generate report
		report, md, err := research.GenerateReport(labels)
		if err != nil {
			return fmt.Errorf("failed to generate report: %w", err)
		}

		// Ensure output directory exists
		if erOut != "" {
			if err := os.MkdirAll(filepath.Dir(erOut), 0755); err != nil {
				return fmt.Errorf("failed to create output directory: %w", err)
			}
		}

		// Determine paths
		mdPath := erOut
		jsonPath := ""
		if strings.HasSuffix(erOut, ".md") {
			jsonPath = strings.TrimSuffix(erOut, ".md") + ".json"
		} else {
			jsonPath = erOut + ".json"
		}

		// Write reports
		if err := research.WriteEvalReport(jsonPath, mdPath, report, md); err != nil {
			return fmt.Errorf("failed to write eval reports: %w", err)
		}

		fmt.Printf("Regime evaluation report generated successfully at:\n  - %s\n  - %s\n", mdPath, jsonPath)
		return nil
	},
}

func init() {
	evaluateRegimesCmd.Flags().StringVar(&erRegimes, "regimes", "", "Path to classified regimes file (JSON/CSV)")
	evaluateRegimesCmd.Flags().StringVar(&erOut, "out", "", "Output report path (markdown)")

	_ = evaluateRegimesCmd.MarkFlagRequired("regimes")
	_ = evaluateRegimesCmd.MarkFlagRequired("out")

	rootCmd.AddCommand(evaluateRegimesCmd)
}
