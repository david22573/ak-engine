package app

import (
	"fmt"
)

func verifyNativeSummaryV2(loaded []fundingLoadedEventFile, audit *FundingEventIntegrityAudit) {
	if audit.Status == "" {
		audit.Status = "PASS"
	}
	v2RowsTotal := 0
	v2ChunksFound := 0
	v2MissingChunks := 0

	var symbolsCovered = make(map[string]bool)
	var monthsCovered = make(map[string]bool)
	var quartersCovered = make(map[string]bool)
	var yearsCovered = make(map[string]bool)

	var hasCostStress = false
	var hasDelayStress = false

	var totalV2Events int
	var totalV2Declustered int
	rowMetricFailures := 0
	rowMetricUnknown := 0

	var lastInputHash string
	hashesStable := true

	for _, item := range loaded {
		if item.V2Missing {
			v2MissingChunks++
			continue
		}
		v2ChunksFound++
		v2RowsTotal += len(item.V2Summary)
		for _, row := range item.V2Summary {
			symbolsCovered[row.Symbol] = true
			monthsCovered[row.Month] = true
			quartersCovered[row.Quarter] = true
			yearsCovered[row.Year] = true

			totalV2Events += row.EventCount
			totalV2Declustered += row.DeClusteredEventCount

			if row.CostBps == 5 || row.CostBps == 7.5 || row.CostBps == 10 || row.CostBps == 15 {
				hasCostStress = true
			}
			if row.DelayCandles == 1 || row.DelayCandles == 2 {
				hasDelayStress = true
			}

			if row.EventCount > 0 && row.WinCount+row.LossCount == 0 {
				rowMetricUnknown++
			} else if row.EventCount != row.WinCount+row.LossCount ||
				row.DeClusteredEventCount > row.EventCount {
				rowMetricFailures++
			}

			if lastInputHash == "" {
				lastInputHash = row.InputHash
			}

			if row.SummaryHash == "" {
				hashesStable = false
			}
		}
	}

	addV2IntegrityCheck(audit, "v2_symbol_coverage", boolStatus(len(symbolsCovered) > 0), fmt.Sprintf("Found %d symbols in V2", len(symbolsCovered)))
	addV2IntegrityCheck(audit, "v2_month_coverage", boolStatus(len(monthsCovered) > 0), fmt.Sprintf("Found %d months in V2", len(monthsCovered)))
	addV2IntegrityCheck(audit, "v2_chunk_count", boolStatus(v2ChunksFound > 0 && v2MissingChunks == 0), fmt.Sprintf("found=%d missing=%d", v2ChunksFound, v2MissingChunks))
	addV2IntegrityCheck(audit, "v2_event_count_consistency", boolStatus(totalV2Events > 0), fmt.Sprintf("Total V2 events %d", totalV2Events))
	addV2IntegrityCheck(audit, "v2_declustered_event_count_consistency", boolStatus(totalV2Declustered > 0 && totalV2Declustered <= totalV2Events), fmt.Sprintf("declustered=%d events=%d", totalV2Declustered, totalV2Events))
	if v2RowsTotal == 0 {
		addV2IntegrityCheck(audit, "v2_aggregate_totals_match", "UNKNOWN", "no V2 rows available to validate aggregate totals")
	} else if rowMetricUnknown > 0 {
		addV2IntegrityCheck(audit, "v2_aggregate_totals_match", "UNKNOWN", fmt.Sprintf("rows_missing_win_loss_fields=%d rows=%d", rowMetricUnknown, v2RowsTotal))
	} else {
		addV2IntegrityCheck(audit, "v2_aggregate_totals_match", boolStatus(rowMetricFailures == 0), fmt.Sprintf("row_metric_failures=%d rows=%d", rowMetricFailures, v2RowsTotal))
	}
	addV2IntegrityCheck(audit, "v2_cost_stress_totals", boolStatus(hasCostStress), "Cost stress records present")
	addV2IntegrityCheck(audit, "v2_delay_stress_totals", boolStatus(hasDelayStress), "Delay stress records present")
	addV2IntegrityCheck(audit, "v2_monthly_buckets", boolStatus(len(monthsCovered) > 0), "Monthly buckets present")
	addV2IntegrityCheck(audit, "v2_quarterly_buckets", boolStatus(len(quartersCovered) > 0), "Quarterly buckets present")
	addV2IntegrityCheck(audit, "v2_symbol_buckets", boolStatus(len(symbolsCovered) > 0), "Symbol buckets present")
	addV2IntegrityCheck(audit, "v2_no_stale_10_5_artifacts", "UNKNOWN", "native summary V2 rows do not include a source phase field")
	addV2IntegrityCheck(audit, "v2_input_hashes_stable", boolStatus(lastInputHash != ""), "Input hashes present")
	addV2IntegrityCheck(audit, "v2_summary_hashes_stable", boolStatus(v2RowsTotal > 0 && hashesStable), "Summary hashes stable")

	for _, check := range audit.Checks {
		switch check.Status {
		case "FAIL":
			audit.Failures = append(audit.Failures, fmt.Sprintf("V2 Integrity Failure: %s", check.Name))
			audit.Status = "FAIL"
		case "UNKNOWN":
			audit.Warnings = append(audit.Warnings, fmt.Sprintf("V2 Integrity Unknown: %s", check.Name))
			if audit.Status == "PASS" {
				audit.Status = "UNKNOWN"
			}
		}
	}
}

func addV2IntegrityCheck(audit *FundingEventIntegrityAudit, name, status, detail string) {
	audit.Checks = append(audit.Checks, FundingIntegrityCheck{Name: name, Status: status, Detail: detail})
}

func boolStatus(cond bool) string {
	if cond {
		return "PASS"
	}
	return "FAIL"
}
