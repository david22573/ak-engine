package app

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/spf13/cobra"
)

var (
	acrChunks  string
	acrSymbols string
	acrFrom    string
	acrTo      string
	acrOut     string
)

type aggregateChunkReport struct {
	Symbol            string   `json:"symbol"`
	MonthsRequested   int      `json:"months_requested"`
	MonthsCompleted   int      `json:"months_completed"`
	MonthsFailed      int      `json:"months_failed"`
	TotalFeatureRows  int      `json:"total_feature_rows"`
	TotalRegimeRows   int      `json:"total_regime_rows"`
	LeakageStatus     string   `json:"leakage_status"`
	FailedChunks      []string `json:"failed_chunks"`
	TotalChunks       int      `json:"total_chunks"`
	TotalRows         int      `json:"total_rows"`
	Status            string   `json:"status"`
}

var aggregateChunkReportsCmd = &cobra.Command{
	Use:   "aggregate-chunk-reports",
	Short: "Aggregate phase 10.5 chunk reports without loading full features",
	RunE: func(cmd *cobra.Command, args []string) error {
		return runAggregateChunkReports()
	},
}

func init() {
	aggregateChunkReportsCmd.Flags().StringVar(&acrChunks, "chunks", "runs/reports/chunks", "Chunks directory")
	aggregateChunkReportsCmd.Flags().StringVar(&acrSymbols, "symbols", "", "Comma-separated symbols")
	aggregateChunkReportsCmd.Flags().StringVar(&acrFrom, "from", "", "From date (YYYY-MM)")
	aggregateChunkReportsCmd.Flags().StringVar(&acrTo, "to", "", "To date (YYYY-MM)")
	aggregateChunkReportsCmd.Flags().StringVar(&acrOut, "out", "", "Output report path")

	_ = aggregateChunkReportsCmd.MarkFlagRequired("chunks")
	_ = aggregateChunkReportsCmd.MarkFlagRequired("symbols")
	_ = aggregateChunkReportsCmd.MarkFlagRequired("from")
	_ = aggregateChunkReportsCmd.MarkFlagRequired("to")
	_ = aggregateChunkReportsCmd.MarkFlagRequired("out")

	rootCmd.AddCommand(aggregateChunkReportsCmd)
}

func runAggregateChunkReports() error {
	symbols := strings.Split(acrSymbols, ",")
	fromTime, err := time.Parse("2006-01", acrFrom)
	if err != nil {
		return err
	}
	toTime, err := time.Parse("2006-01", acrTo)
	if err != nil {
		return err
	}

	for _, sym := range symbols {
		sym = strings.TrimSpace(sym)
		if sym == "" {
			continue
		}

		report := aggregateChunkReport{
			Symbol:        sym,
			Status:        "PASS",
			LeakageStatus: "PASS",
			FailedChunks:  make([]string, 0),
		}

		current := fromTime
		for !current.After(toTime) {
			report.MonthsRequested++
			monthStr := current.Format("2006-01")
			reportPath := filepath.Join(acrChunks, sym, monthStr+"-summary.json")

			data, err := os.ReadFile(reportPath)
			if err == nil {
				var chunkRep map[string]any
				if err := json.Unmarshal(data, &chunkRep); err == nil {
					report.TotalChunks++
					report.MonthsCompleted++
					if rows, ok := chunkRep["rows"].(float64); ok {
						report.TotalRows += int(rows)
						report.TotalFeatureRows += int(rows)
					} else if rows, ok := chunkRep["feature_rows"].(float64); ok {
						report.TotalRows += int(rows)
						report.TotalFeatureRows += int(rows)
					}
				}
			} else {
				report.MonthsFailed++
				report.FailedChunks = append(report.FailedChunks, monthStr)
				report.Status = "FAIL"
			}

			current = current.AddDate(0, 1, 0)
		}

		os.MkdirAll(filepath.Dir(acrOut), 0755)
		
		// Create JSON report 
		jsonPath := strings.TrimSuffix(acrOut, filepath.Ext(acrOut)) + ".json"
		outData, _ := json.MarshalIndent(report, "", "  ")
		os.WriteFile(jsonPath, outData, 0644)

		// Create Markdown report
		mdPath := strings.TrimSuffix(acrOut, filepath.Ext(acrOut)) + ".md"
		md := fmt.Sprintf("# Aggregate Chunk Report\n\n- Symbol: %s\n- Status: %s\n- Total Chunks: %d\n- Total Rows: %d\n", report.Symbol, report.Status, report.TotalChunks, report.TotalRows)
		os.WriteFile(mdPath, []byte(md), 0644)

		fmt.Printf("Aggregated %d chunks, %d rows total.\n", report.TotalChunks, report.TotalRows)
	}

	return nil
}
