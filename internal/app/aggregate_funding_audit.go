package app

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
)

var aggregateFundingAuditCmd = &cobra.Command{
	Use:   "aggregate-funding-audit",
	Short: "Aggregate funding audit chunks",
	RunE: func(cmd *cobra.Command, args []string) error {
		symbols := []string{"LINKUSDT", "SOLUSDT", "AVAXUSDT", "DOGEUSDT", "ADAUSDT", "BNBUSDT", "XRPUSDT", "ETHUSDT"}

		var allRows []ReportRow

		for _, sym := range symbols {
			summaryDir := filepath.Join("runs", "reports", "chunks", sym)
			files, err := os.ReadDir(summaryDir)
			if err != nil {
				continue
			}

			for _, f := range files {
				if filepath.Ext(f.Name()) == ".json" && len(f.Name()) >= 21 && f.Name()[len(f.Name())-20:] == "funding-summary.json" {
					data, err := os.ReadFile(filepath.Join(summaryDir, f.Name()))
					if err == nil {
						var r ReportRow
						if err := json.Unmarshal(data, &r); err == nil && r.Month != "" {
							allRows = append(allRows, r)
						}
					}
				}
			}
		}

		outDir := filepath.Join("runs", "reports")
		os.MkdirAll(outDir, 0755)

		jsonPath := filepath.Join(outDir, "phase10_6_funding_join_audit.json")
		jsonData, _ := json.MarshalIndent(allRows, "", "  ")
		os.WriteFile(jsonPath, jsonData, 0644)

		var md bytes.Buffer
		md.WriteString("# Phase 10.6 Funding Join Audit\n\n")
		md.WriteString("| Symbol | Month | Rows | Funding Rows | Unknown | Coverage % | Min | Median | Max | ZScore | Leakage |\n")
		md.WriteString("|---|---|---|---|---|---|---|---|---|---|---|\n")
		for _, r := range allRows {
			md.WriteString(fmt.Sprintf("| %s | %s | %d | %d | %d | %.2f%% | %.6f | %.6f | %.6f | %v | %s |\n",
				r.Symbol, r.Month, r.FeatureRows, r.RowsWithFunding, r.RowsWithFundingUnknown,
				r.FundingCoveragePct, r.MinFundingRate, r.MedianFundingRate, r.MaxFundingRate,
				r.FundingRateZScoreAvailable, r.AsOfJoinLeakageStatus))
		}
		os.WriteFile(filepath.Join(outDir, "phase10_6_funding_join_audit.md"), md.Bytes(), 0644)

		return nil
	},
}

func init() {
	rootCmd.AddCommand(aggregateFundingAuditCmd)
}
