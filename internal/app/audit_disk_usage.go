package app

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/spf13/cobra"
)

var auditDiskRoot string

type DiskUsageEntry struct {
	Path             string   `json:"path"`
	Exists           bool     `json:"exists"`
	SizeBytes        int64    `json:"size_bytes"`
	SizeHuman        string   `json:"size_human"`
	FileCount        int      `json:"file_count"`
	LargestFiles     []string `json:"largest_files"`
	CleanupCandidate bool     `json:"cleanup_candidate"`
}

type DiskUsageReport struct {
	TotalSizeBytes int64             `json:"total_size_bytes"`
	TotalSizeHuman string            `json:"total_size_human"`
	Entries        []*DiskUsageEntry `json:"entries"`
}

var auditDiskUsageCmd = &cobra.Command{
	Use:   "audit-disk-usage",
	Short: "Audit disk usage for Phase 10.5 low resource harness",
	RunE: func(cmd *cobra.Command, args []string) error {
		return runAuditDiskUsage(cmd)
	},
}

func init() {
	auditDiskUsageCmd.Flags().StringVar(&auditDiskRoot, "root", ".", "Root directory")
	rootCmd.AddCommand(auditDiskUsageCmd)
}

func formatBytes(b int64) string {
	const unit = 1024
	if b < unit {
		return fmt.Sprintf("%d B", b)
	}
	div, exp := int64(unit), 0
	for n := b / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB", float64(b)/float64(div), "KMGTPE"[exp])
}

type fileInfo struct {
	path string
	size int64
}

func getDirStats(dir string) (int64, int, []string, bool) {
	var size int64
	var count int
	var files []fileInfo

	exists := false
	if _, err := os.Stat(dir); err == nil {
		exists = true
	}

	if exists {
		filepath.Walk(dir, func(p string, info os.FileInfo, err error) error {
			if err != nil {
				return nil
			}
			if !info.IsDir() {
				size += info.Size()
				count++
				files = append(files, fileInfo{path: p, size: info.Size()})
			}
			return nil
		})
	}

	sort.Slice(files, func(i, j int) bool {
		return files[i].size > files[j].size
	})

	var largest []string
	for i := 0; i < 5 && i < len(files); i++ {
		largest = append(largest, fmt.Sprintf("%s (%s)", files[i].path, formatBytes(files[i].size)))
	}

	return size, count, largest, exists
}

func runAuditDiskUsage(cmd *cobra.Command) error {
	historianWorkdir := resolveHistorianWorkdir(cmd, "")
	pathsToCheck := []string{
		"runs",
		filepath.Join("runs", "features"),
		filepath.Join("runs", "features", "chunks"),
		filepath.Join("runs", "regimes"),
		filepath.Join("runs", "regimes", "chunks"),
		filepath.Join("runs", "reports"),
		filepath.Join("runs", "reports", "chunks"),
		filepath.Join("runs", "manifests"),
		historianWorkdir,
		filepath.Join(historianWorkdir, "candles"),
		filepath.Join(historianWorkdir, "datasets"),
	}

	report := &DiskUsageReport{}
	var totalSize int64

	for _, relPath := range pathsToCheck {
		fullPath := filepath.Join(auditDiskRoot, relPath)
		size, count, largest, exists := getDirStats(fullPath)

		isCleanup := false
		if strings.Contains(relPath, "chunks") && !strings.Contains(relPath, "reports") {
			isCleanup = true
		}

		entry := &DiskUsageEntry{
			Path:             relPath,
			Exists:           exists,
			SizeBytes:        size,
			SizeHuman:        formatBytes(size),
			FileCount:        count,
			LargestFiles:     largest,
			CleanupCandidate: isCleanup,
		}

		report.Entries = append(report.Entries, entry)

		// Only add to total if it's "runs" or historian workdir to avoid double counting
		if relPath == "runs" || relPath == historianWorkdir {
			totalSize += size
		}
	}

	report.TotalSizeBytes = totalSize
	report.TotalSizeHuman = formatBytes(totalSize)

	// Outputs
	os.MkdirAll(filepath.Join(auditDiskRoot, "runs", "reports"), 0755)

	jsonPath := filepath.Join(auditDiskRoot, "runs", "reports", "phase10_5d_disk_usage.json")
	mdPath := filepath.Join(auditDiskRoot, "runs", "reports", "phase10_5d_disk_usage.md")

	jsonData, _ := json.MarshalIndent(report, "", "  ")
	os.WriteFile(jsonPath, jsonData, 0644)

	var md string
	md += "# Phase 10.5D Disk Usage Audit\n\n"
	md += fmt.Sprintf("**Total Estimated Usage:** %s\n\n", report.TotalSizeHuman)
	md += "| Path | Exists | Size | Files | Cleanup Candidate |\n"
	md += "|---|---|---|---|---|\n"
	for _, e := range report.Entries {
		md += fmt.Sprintf("| %s | %v | %s | %d | %v |\n", e.Path, e.Exists, e.SizeHuman, e.FileCount, e.CleanupCandidate)
	}

	os.WriteFile(mdPath, []byte(md), 0644)

	fmt.Printf("Audit complete. Total size: %s\n", report.TotalSizeHuman)
	fmt.Printf("Report saved to %s\n", mdPath)

	return nil
}
