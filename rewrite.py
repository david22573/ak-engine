import sys

with open("internal/app/phase10_low_resource_prep.go", "r") as f:
    content = f.read()

import re

# Add imports
content = content.replace('"strings"', '"strings"\n\t"syscall"')

# Add flags
flags_var = """	plrpCompactOutput   bool
	plrpDiskBudgetGb    float64
	plrpMinFreeGb       float64
	plrpFailIfBudget    bool
	plrpCleanupAfter    bool
	plrpRetainPolicy    string
	plrpCompress        bool
"""
content = content.replace("	plrpCompactOutput   bool", flags_var)

flags_init = """	phase10LowResourcePrepCmd.Flags().BoolVar(&plrpCompactOutput, "compact-output", false, "Compact output JSON")
	phase10LowResourcePrepCmd.Flags().Float64Var(&plrpDiskBudgetGb, "disk-budget-gb", 8, "Disk budget in GB")
	phase10LowResourcePrepCmd.Flags().Float64Var(&plrpMinFreeGb, "min-free-gb", 5, "Min free disk space in GB")
	phase10LowResourcePrepCmd.Flags().BoolVar(&plrpFailIfBudget, "fail-if-budget-exceeded", false, "Fail if disk budget exceeded")
	phase10LowResourcePrepCmd.Flags().BoolVar(&plrpCleanupAfter, "cleanup-after-chunk", false, "Cleanup after each chunk")
	phase10LowResourcePrepCmd.Flags().StringVar(&plrpRetainPolicy, "retain-policy", "reports_only", "Cleanup retain policy")
	phase10LowResourcePrepCmd.Flags().BoolVar(&plrpCompress, "compress-before-delete", false, "Compress before delete")
"""
content = content.replace("	phase10LowResourcePrepCmd.Flags().BoolVar(&plrpCompactOutput, \"compact-output\", false, \"Compact output JSON\")", flags_init)

# Add ChunkStatus fields
fields = """	RowCount           int       `json:"row_count"`
	DiskFreeBeforeBytes     int64     `json:"disk_free_before_bytes,omitempty"`
	DiskFreeAfterBytes      int64     `json:"disk_free_after_bytes,omitempty"`
	ArtifactSizeBeforeBytes int64     `json:"artifact_size_before_bytes,omitempty"`
	ArtifactSizeAfterBytes  int64     `json:"artifact_size_after_bytes,omitempty"`
	DiskStatus              string    `json:"disk_status,omitempty"`"""
content = content.replace('	RowCount           int       `json:"row_count"`', fields)

# Add helper functions at end
helpers = """

func getDiskSpace(path string) int64 {
	var stat syscall.Statfs_t
	if err := syscall.Statfs(path, &stat); err != nil {
		return 0
	}
	return int64(stat.Bavail) * int64(stat.Bsize)
}

func getDirSize(path string) int64 {
	var size int64
	filepath.Walk(path, func(_ string, info os.FileInfo, err error) error {
		if err == nil && !info.IsDir() {
			size += info.Size()
		}
		return nil
	})
	return size
}
"""
content += helpers

# Insert Disk Guard Logic before processChunk
guard_logic = """
			// Disk guard
			freeBytes := getDiskSpace(".")
			artBytes := getDirSize("runs")
			status.DiskFreeBeforeBytes = freeBytes
			status.ArtifactSizeBeforeBytes = artBytes
			
			if float64(freeBytes) < plrpMinFreeGb*1024*1024*1024 {
			    status.DiskStatus = "BLOCKED"
			    saveManifest(manifestPath, manifest)
			    return fmt.Errorf("Free space below threshold: %.1f GB", float64(freeBytes)/(1024*1024*1024))
			}
			
			if plrpFailIfBudget && float64(artBytes) > plrpDiskBudgetGb*1024*1024*1024 {
			    status.DiskStatus = "BUDGET_EXCEEDED"
			    saveManifest(manifestPath, manifest)
			    return fmt.Errorf("Artifact size exceeds budget: %.1f GB", float64(artBytes)/(1024*1024*1024))
			}
			status.DiskStatus = "OK"

			if plrpForce {
"""
content = content.replace("			if plrpForce {", guard_logic)

# Insert Disk Post Check after processChunk
post_logic = """
				status.ErrorIfFailed = ""
				status.CompletedAt = time.Now()
				
				// Cleanup logic
				if plrpCleanupAfter {
				    args := []string{
				        "cleanup-runs",
				        "--root", ".",
				        "--retain-policy", plrpRetainPolicy,
				        "--force",
				    }
				    cmd := exec.Command(os.Args[0], args...)
				    cmd.Run()
				    status.DiskStatus = "CLEANED"
				}

				status.DiskFreeAfterBytes = getDiskSpace(".")
				status.ArtifactSizeAfterBytes = getDirSize("runs")
				
				saveManifest(manifestPath, manifest)
			}
"""
content = content.replace("""				status.ErrorIfFailed = ""
				status.CompletedAt = time.Now()
				saveManifest(manifestPath, manifest)
			}""", post_logic)


# Write detailed summary in processChunk
detailed_summary = """
	// 4. Report
	status.ReportStatus = "DONE"
	reportMap := map[string]interface{}{
		"symbol": sym,
		"month": monthStr,
		"feature_rows": status.RowCount,
		"regime_rows": status.RowCount,
		"status": "PASS",
	}
	reportBytes, _ := json.Marshal(reportMap)
	os.WriteFile(reportPath, reportBytes, 0644)
"""
content = content.replace("""	// 4. Report
	status.ReportStatus = "DONE"
	reportJson := fmt.Sprintf(`{"status":"PASS","rows":%d}`, status.RowCount)
	os.WriteFile(reportPath, []byte(reportJson), 0644)""", detailed_summary)


with open("internal/app/phase10_low_resource_prep.go", "w") as f:
    f.write(content)
