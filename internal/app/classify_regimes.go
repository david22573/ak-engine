package app

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/davidmiguel22573/ak-engine/internal/features"
	"github.com/davidmiguel22573/ak-engine/internal/regime"
	"github.com/davidmiguel22573/ak-engine/internal/research"
	"github.com/spf13/cobra"
)

var (
	crFeatures          string
	crOut               string
	crFormat            string
	crThresholdLookback int
	crThresholdMinRows  int
)

type classifyRegimesResult struct {
	Status       string         `json:"status"`
	Labels       int            `json:"labels"`
	WarmupLabels int            `json:"warmup_labels"`
	Composites   map[string]int `json:"composites"`
	Out          string         `json:"out"`
}

var classifyRegimesCmd = &cobra.Command{
	Use:   "classify-regimes",
	Short: "Classify market regimes from feature rows",
	RunE: func(cmd *cobra.Command, args []string) error {
		// Read features
		var rows []features.Row
		var err error
		if strings.HasSuffix(crFeatures, ".json") {
			rows, err = features.ReadRowsJSON(crFeatures)
		} else if strings.HasSuffix(crFeatures, ".csv") {
			rows, err = features.ReadRowsCSV(crFeatures)
		} else {
			return fmt.Errorf("unsupported features file format (must end with .json or .csv)")
		}

		if err != nil {
			return fmt.Errorf("failed to read features: %w", err)
		}

		// Run feature leakage check
		leakReport := research.CheckFeatureRows(rows)
		if leakReport.Status != "PASS" {
			issuesJSON, _ := json.Marshal(leakReport.Issues)
			return fmt.Errorf("feature leakage check failed: %s", string(issuesJSON))
		}

		// Setup classifier
		opts := regime.ThresholdOptions{
			LookbackRows: crThresholdLookback,
			MinRows:      crThresholdMinRows,
		}
		classifier := regime.NewClassifier(opts)

		// Classify
		labels, err := classifier.ClassifyRows(rows)
		if err != nil {
			return fmt.Errorf("classification failed: %w", err)
		}

		// Run label leakage check
		labelLeakReport := research.CheckLabels(labels)
		if labelLeakReport.Status != "PASS" {
			issuesJSON, _ := json.Marshal(labelLeakReport.Issues)
			return fmt.Errorf("label leakage check failed: %s", string(issuesJSON))
		}

		// Ensure parent directory of output exists
		if crOut != "" {
			if err := os.MkdirAll(filepath.Dir(crOut), 0755); err != nil {
				return fmt.Errorf("failed to create output directory: %w", err)
			}
		}

		// Write output
		switch crFormat {
		case "json":
			if err := regime.WriteLabelsJSON(crOut, labels); err != nil {
				return fmt.Errorf("failed to write labels JSON: %w", err)
			}
		case "csv":
			if err := regime.WriteLabelsCSV(crOut, labels); err != nil {
				return fmt.Errorf("failed to write labels CSV: %w", err)
			}
		case "parquet":
			tmpCsv := crOut + ".tmp.csv"
			if err := regime.WriteLabelsCSV(tmpCsv, labels); err != nil {
				return fmt.Errorf("failed to write temporary CSV: %w", err)
			}
			defer os.Remove(tmpCsv)
			if err := regime.WriteLabelsParquet(tmpCsv, crOut); err != nil {
				return fmt.Errorf("failed to write labels Parquet: %w", err)
			}
		default:
			return fmt.Errorf("unsupported format %q", crFormat)
		}

		// Summarize
		warmupCount := 0
		composites := make(map[string]int)
		for _, l := range labels {
			if l.Warmup {
				warmupCount++
			} else {
				composites[l.Composite]++
			}
		}

		res := classifyRegimesResult{
			Status:       "PASS",
			Labels:       len(labels),
			WarmupLabels: warmupCount,
			Composites:   composites,
			Out:          crOut,
		}

		resBytes, _ := json.MarshalIndent(res, "", "  ")
		fmt.Println(string(resBytes))
		return nil
	},
}

func init() {
	classifyRegimesCmd.Flags().StringVar(&crFeatures, "features", "", "Path to feature rows file (JSON/CSV)")
	classifyRegimesCmd.Flags().StringVar(&crOut, "out", "", "Output path")
	classifyRegimesCmd.Flags().StringVar(&crFormat, "format", "json", "Output format (json | csv | parquet)")
	classifyRegimesCmd.Flags().IntVar(&crThresholdLookback, "threshold-lookback", 0, "Lookback rows for trailing thresholds")
	classifyRegimesCmd.Flags().IntVar(&crThresholdMinRows, "threshold-min-rows", 0, "Minimum rows for thresholds")

	_ = classifyRegimesCmd.MarkFlagRequired("features")
	_ = classifyRegimesCmd.MarkFlagRequired("out")

	rootCmd.AddCommand(classifyRegimesCmd)
}
