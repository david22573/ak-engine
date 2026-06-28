package app

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
)

var (
	auditV2Path        string
	auditV2FailOnError bool
)

var auditV2IntegrityCmd = &cobra.Command{
	Use:   "audit-v2-integrity",
	Short: "Audit Native Summary V2 integrity",
	RunE: func(cmd *cobra.Command, args []string) error {
		var loaded []fundingLoadedEventFile

		err := filepath.Walk(auditV2Path, func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return err
			}
			if !info.IsDir() && strings.HasSuffix(info.Name(), "-native-summary-v2.json") {
				data, err := os.ReadFile(path)
				if err != nil {
					return err
				}
				var v2Rows []NativeSummaryV2Row
				if err := json.Unmarshal(data, &v2Rows); err != nil {
					return err
				}
				item := fundingLoadedEventFile{
					V2Missing: false,
					V2Summary: v2Rows,
				}
				loaded = append(loaded, item)
			}
			return nil
		})

		if err != nil {
			return err
		}

		audit := FundingEventIntegrityAudit{Status: "PASS"}

		verifyNativeSummaryV2(loaded, &audit)

		outData, _ := json.MarshalIndent(audit, "", "  ")
		fmt.Println(string(outData))

		if auditV2FailOnError && audit.Status == "FAIL" {
			return fmt.Errorf("audit failed: %v", audit.Failures)
		}

		return nil
	},
}

func init() {
	auditV2IntegrityCmd.Flags().StringVar(&auditV2Path, "path", "runs/reports/chunks", "Path to chunks")
	auditV2IntegrityCmd.Flags().BoolVar(&auditV2FailOnError, "fail-on-error", false, "Fail on error")
	rootCmd.AddCommand(auditV2IntegrityCmd)
}
