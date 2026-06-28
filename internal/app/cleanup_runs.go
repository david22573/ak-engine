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
	cleanupRoot         string
	cleanupRetainPolicy string
	cleanupOlderThan    int
	cleanupDryRun       bool
	cleanupForce        bool
	cleanupMaxDeleteGb  float64
)

var cleanupRunsCmd = &cobra.Command{
	Use:   "cleanup-runs",
	Short: "Safe cleanup command for Phase 10.5 harness",
	RunE: func(cmd *cobra.Command, args []string) error {
		return runCleanupRuns(cmd)
	},
}

func init() {
	cleanupRunsCmd.Flags().StringVar(&cleanupRoot, "root", ".", "Root directory")
	cleanupRunsCmd.Flags().StringVar(&cleanupRetainPolicy, "retain-policy", "reports_only", "reports_only|latest_only|failed_only|keep_all")
	cleanupRunsCmd.Flags().IntVar(&cleanupOlderThan, "older-than-days", 0, "Delete chunks older than N days")
	cleanupRunsCmd.Flags().BoolVar(&cleanupDryRun, "dry-run", false, "Dry run, show what would be deleted")
	cleanupRunsCmd.Flags().BoolVar(&cleanupForce, "force", false, "Actually delete files")
	cleanupRunsCmd.Flags().Float64Var(&cleanupMaxDeleteGb, "max-delete-gb", 0, "Max GB to delete (0 = unlimited)")

	rootCmd.AddCommand(cleanupRunsCmd)
}

type CleanupReport struct {
	FilesToDelete []string `json:"files_to_delete"`
	BytesToDelete int64    `json:"bytes_to_delete"`
	FilesToKeep   []string `json:"files_to_keep"`
	Reason        string   `json:"reason"`
}

func runCleanupRuns(cmd *cobra.Command) error {
	if cleanupRetainPolicy == "keep_all" {
		fmt.Println("retain-policy is keep_all. Nothing to delete.")
		return nil
	}

	allowedRoots := []string{
		filepath.Join("runs", "features", "chunks"),
		filepath.Join("runs", "regimes", "chunks"),
		filepath.Join("runs", "reports", "chunks"),
		filepath.Join("runs", "tmp"),
	}

	manifestPath := filepath.Join(cleanupRoot, "runs", "manifests", "phase10_5_low_resource_manifest.json")
	manifest := loadManifest(manifestPath)
	
	// Create map of successful / failed chunks
	latestSuccessfulChunk := make(map[string]time.Time) // track latest successful month per symbol
	failedChunks := make(map[string]bool)               // symbol_month -> is failed
	successfulChunks := make(map[string]bool)           // symbol_month -> is successful
	
	if manifest != nil {
		for key, status := range manifest.Chunks {
			isOk := isComplete(status)
			if isOk {
				successfulChunks[key] = true
				monthTime, err := time.Parse("2006-01", status.Month)
				if err == nil {
					if currentLatest, exists := latestSuccessfulChunk[status.Symbol]; !exists || monthTime.After(currentLatest) {
						latestSuccessfulChunk[status.Symbol] = monthTime
					}
				}
			} else {
				failedChunks[key] = true
			}
		}
	}

	report := &CleanupReport{
		Reason: fmt.Sprintf("Policy: %s", cleanupRetainPolicy),
	}

	var totalDeletedBytes int64

	for _, relRoot := range allowedRoots {
		fullRoot := filepath.Join(cleanupRoot, relRoot)
		
		if _, err := os.Stat(fullRoot); os.IsNotExist(err) {
			continue
		}

		filepath.Walk(fullRoot, func(p string, info os.FileInfo, err error) error {
			if err != nil || info.IsDir() {
				return nil
			}

			// Do not follow symlinks outside allowed roots (by default Walk doesn't follow symlinks, but just to be safe)
			if info.Mode()&os.ModeSymlink != 0 {
				return nil
			}

			filename := filepath.Base(p)
			
			// Never delete manifests, though they aren't in these directories
			if strings.HasSuffix(filename, "manifest.json") {
				report.FilesToKeep = append(report.FilesToKeep, p)
				return nil
			}
			
			// Never delete final reports
			if strings.HasPrefix(p, filepath.Join(cleanupRoot, "runs", "reports")) && !strings.Contains(p, "chunks") {
				report.FilesToKeep = append(report.FilesToKeep, p)
				return nil
			}
			
			// Summaries usually shouldn't be deleted for most policies
			isSummary := strings.HasSuffix(filename, "-summary.json")
			if isSummary {
				report.FilesToKeep = append(report.FilesToKeep, p)
				return nil
			}

			// Extract sym and month to check status
			parts := strings.Split(filename, "-")
			if len(parts) >= 2 {
				// e.g., 2024-01-context.json -> 2024-01
				monthStr := fmt.Sprintf("%s-%s", parts[0], parts[1])
				dirPath := filepath.Dir(p)
				sym := filepath.Base(dirPath)
				chunkKey := sym + "_" + monthStr

				// Apply older-than
				if cleanupOlderThan > 0 {
					if time.Since(info.ModTime()).Hours() < float64(cleanupOlderThan*24) {
						report.FilesToKeep = append(report.FilesToKeep, p)
						return nil
					}
				}

				shouldDelete := false

				switch cleanupRetainPolicy {
				case "reports_only":
					shouldDelete = true
				case "latest_only":
					monthTime, _ := time.Parse("2006-01", monthStr)
					if latest, exists := latestSuccessfulChunk[sym]; exists {
						if monthTime.Before(latest) {
							shouldDelete = true
						}
					}
				case "failed_only":
					if successfulChunks[chunkKey] {
						shouldDelete = true
					}
				}

				if shouldDelete {
					if cleanupMaxDeleteGb > 0 && float64(totalDeletedBytes+info.Size())/(1024*1024*1024) > cleanupMaxDeleteGb {
						report.FilesToKeep = append(report.FilesToKeep, p)
					} else {
						report.FilesToDelete = append(report.FilesToDelete, p)
						report.BytesToDelete += info.Size()
						totalDeletedBytes += info.Size()
					}
				} else {
					report.FilesToKeep = append(report.FilesToKeep, p)
				}
			}
			return nil
		})
	}

	if cleanupForce && !cleanupDryRun {
		for _, f := range report.FilesToDelete {
			os.Remove(f)
		}
	}

	os.MkdirAll(filepath.Join(cleanupRoot, "runs", "reports"), 0755)
	
	if cleanupDryRun {
		fmt.Printf("Dry run completed. Would delete %d files (%.1f MB).\n", len(report.FilesToDelete), float64(report.BytesToDelete)/(1024*1024))
	} else if cleanupForce {
		jsonPath := filepath.Join(cleanupRoot, "runs", "reports", "phase10_5d_cleanup_report.json")
		mdPath := filepath.Join(cleanupRoot, "runs", "reports", "phase10_5d_cleanup_report.md")

		jsonData, _ := json.MarshalIndent(report, "", "  ")
		os.WriteFile(jsonPath, jsonData, 0644)

		var md string
		md += "# Phase 10.5D Cleanup Report\n\n"
		md += fmt.Sprintf("**Policy:** %s\n", cleanupRetainPolicy)
		md += fmt.Sprintf("**Files Deleted:** %d\n", len(report.FilesToDelete))
		md += fmt.Sprintf("**Bytes Freed:** %.1f MB\n", float64(report.BytesToDelete)/(1024*1024))
		
		os.WriteFile(mdPath, []byte(md), 0644)
		
		fmt.Printf("Cleanup completed. Deleted %d files (%.1f MB).\n", len(report.FilesToDelete), float64(report.BytesToDelete)/(1024*1024))
	} else {
		fmt.Println("Neither --dry-run nor --force specified. Doing nothing.")
	}

	return nil
}
