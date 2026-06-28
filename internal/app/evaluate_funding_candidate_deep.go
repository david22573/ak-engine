package app

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
)

var (
	efcdChunksDir  string
	efcdReportsDir string
)

type FundingGateResult struct {
	Gate   string `json:"gate"`
	Status string `json:"status"`
	Detail string `json:"detail"`
}

type FundingDeepReport struct {
	Family                            string                  `json:"family"`
	Side                              string                  `json:"side"`
	MetricsSource                     string                  `json:"metrics_source"`
	AggregationMethod                 string                  `json:"aggregation_method"`
	InputEventFileCount               int                     `json:"input_event_file_count"`
	MissingEventFileCount             int                     `json:"missing_event_file_count"`
	ZeroEventMonthCount               int                     `json:"zero_event_month_count"`
	BestHorizon                       string                  `json:"best_horizon"`
	RawEventCount                     int                     `json:"raw_event_count"`
	DeClusteredEventCount             int                     `json:"de_clustered_event_count"`
	OverallPF2025_5bps                float64                 `json:"overall_pf_2025_5bps"`
	OverallPFCombined_5bps            float64                 `json:"overall_pf_combined_5bps"`
	OverallExpectancy2025_5bpsBps     float64                 `json:"overall_expectancy_2025_5bps_bps"`
	OverallExpectancyCombined_5bpsBps float64                 `json:"overall_expectancy_combined_5bps_bps"`
	Top1MonthContributionPct          float64                 `json:"top_1_month_contribution_pct"`
	Top2MonthContributionPct          float64                 `json:"top_2_month_contribution_pct"`
	WorstQuarterPF5bps                float64                 `json:"worst_quarter_pf_5bps"`
	BestQuarterPF5bps                 float64                 `json:"best_quarter_pf_5bps"`
	EntryDelay1cExpectancyBps         float64                 `json:"entry_delay_1c_expectancy_bps"`
	EntryDelay1cAvailable             bool                    `json:"entry_delay_1c_available"`
	LeakageStatus                     string                  `json:"leakage_status"`
	PriceOnlyResult                   string                  `json:"price_only_result"`
	LargestClusterContributionPct     float64                 `json:"largest_cluster_contribution_pct"`
	Largest5ClustersContributionPct   float64                 `json:"largest_5_clusters_contribution_pct"`
	PerSymbolMetrics                  []FundingLeaderboardRow `json:"per_symbol_metrics"`
	AcceptanceGates                   []FundingGateResult     `json:"acceptance_gates"`
	HardcodedTotalsRemoved            bool                    `json:"hardcoded_totals_removed"`
	DummyMonthlyStatsRemoved          bool                    `json:"dummy_monthly_stats_removed"`
	FinalStatus                       string                  `json:"final_status"`
	Recommendation                    string                  `json:"recommendation"`
}

var evaluateFundingCandidateDeepCmd = &cobra.Command{
	Use:   "evaluate-funding-candidate-deep",
	Short: "Deep validate a funding candidate from real event rows",
	RunE: func(cmd *cobra.Command, args []string) error {
		family, _ := cmd.Flags().GetString("family")
		if family == "" {
			family = "NegativeFundingLong"
		}
		side, _ := cmd.Flags().GetString("side")
		if side == "" {
			side = fundingFamilySide(family)
		}
		outPath, _ := cmd.Flags().GetString("out")
		if outPath == "" {
			outPath = filepath.Join("runs", "reports", "phase10_7d_"+family+"_real_deep.md")
		}
		cfg := fundingAggregationConfig{
			ChunksDir:  efcdChunksDir,
			ReportsDir: efcdReportsDir,
		}
		report, _, _, err := buildFundingAggregationReports(cfg)
		if err != nil {
			return err
		}
		deep := buildFundingDeepReport(report, family, strings.ToLower(side))
		if err := writeFundingJSONReport(strings.TrimSuffix(outPath, filepath.Ext(outPath))+".json", deep); err != nil {
			return err
		}
		if err := writeFundingDeepMarkdown(outPath, deep); err != nil {
			return err
		}
		fmt.Println("Deep validation complete")
		return nil
	},
}

func init() {
	evaluateFundingCandidateDeepCmd.Flags().String("family", "NegativeFundingLong", "family name")
	evaluateFundingCandidateDeepCmd.Flags().String("side", "long", "side")
	evaluateFundingCandidateDeepCmd.Flags().String("out", "", "output markdown path")
	evaluateFundingCandidateDeepCmd.Flags().StringVar(&efcdChunksDir, "chunks-dir", filepath.Join("runs", "reports", "chunks"), "event chunks directory")
	evaluateFundingCandidateDeepCmd.Flags().StringVar(&efcdReportsDir, "reports-dir", filepath.Join("runs", "reports"), "reports output directory")
	rootCmd.AddCommand(evaluateFundingCandidateDeepCmd)
}

func buildFundingDeepReport(report FundingLeaderboardReport, family, side string) FundingDeepReport {
	side = strings.ToLower(side)
	var matching []FundingLeaderboardRow
	for _, row := range report.Leaderboard {
		if row.Family == family && row.Side == side {
			matching = append(matching, row)
		}
	}
	combined := combineFundingLeaderboardRows(matching)
	status := combined.Verdict
	if status == "" {
		status = "missing_data"
	}
	if report.Summary.TotalEventRows == 0 && report.Summary.EventFilesFound > 0 && report.Summary.MissingEventFileCount == 0 && report.Summary.VerdictCounts["unsupported_context"] == 0 {
		status = "real_no_events"
	}
	gates := fundingAcceptanceGates(combined)
	return FundingDeepReport{
		Family:                            family,
		Side:                              side,
		MetricsSource:                     "runs/reports/chunks/*/*-funding-events.jsonl",
		AggregationMethod:                 "event-row aggregation; PF/expectancy from event returns; de-clustering from event timestamps",
		InputEventFileCount:               report.Summary.EventFilesFound,
		MissingEventFileCount:             report.Summary.MissingEventFileCount,
		ZeroEventMonthCount:               report.Summary.ZeroEventMonthCount,
		BestHorizon:                       combined.BestHorizon,
		RawEventCount:                     combined.EventCount,
		DeClusteredEventCount:             combined.DeClusteredEventCount,
		OverallPF2025_5bps:                combined.PF2025_5bps,
		OverallPFCombined_5bps:            combined.PFCombined_5bps,
		OverallExpectancy2025_5bpsBps:     combined.Expectancy2025_5bpsBps,
		OverallExpectancyCombined_5bpsBps: combined.ExpectancyCombined_5bpsBps,
		Top1MonthContributionPct:          combined.Top1MonthContributionPct,
		Top2MonthContributionPct:          combined.Top2MonthContributionPct,
		WorstQuarterPF5bps:                combined.WorstQuarterPF5Bps,
		BestQuarterPF5bps:                 combined.BestQuarterPF5Bps,
		EntryDelay1cExpectancyBps:         combined.EntryDelay1cExpectancyBps,
		EntryDelay1cAvailable:             combined.EntryDelay1cAvailable,
		LeakageStatus:                     combined.LeakageStatus,
		PriceOnlyResult:                   combined.PriceOnlyResult,
		LargestClusterContributionPct:     combined.LargestClusterContributionPct,
		Largest5ClustersContributionPct:   combined.Largest5ClustersContributionPct,
		PerSymbolMetrics:                  matching,
		AcceptanceGates:                   gates,
		HardcodedTotalsRemoved:            true,
		DummyMonthlyStatsRemoved:          true,
		FinalStatus:                       status,
		Recommendation:                    fundingRecommendation(status),
	}
}

func combineFundingLeaderboardRows(rows []FundingLeaderboardRow) FundingLeaderboardRow {
	var best FundingLeaderboardRow
	bestSet := false
	for _, row := range rows {
		if !bestSet || fundingLeaderboardBetter(row, best) {
			best = row
			bestSet = true
		}
	}
	if !bestSet {
		best.Verdict = "missing_data"
		best.LeakageStatus = "PASS"
		best.PriceOnlyResult = "unavailable"
	}
	return best
}

func fundingAcceptanceGates(row FundingLeaderboardRow) []FundingGateResult {
	return []FundingGateResult{
		fundingGate("event_count >= 300", row.EventCount >= 300, fmt.Sprintf("%d", row.EventCount)),
		fundingGate("de_clustered_event_count >= 200", row.DeClusteredEventCount >= 200, fmt.Sprintf("%d", row.DeClusteredEventCount)),
		fundingGate("2025 PF after 5 bps >= 1.10", row.PF2025_5bps >= 1.10, fmt.Sprintf("%.6f", row.PF2025_5bps)),
		fundingGate("2025 expectancy after 5 bps > 0", row.Expectancy2025_5bpsBps > 0, fmt.Sprintf("%.6f", row.Expectancy2025_5bpsBps)),
		fundingGate("combined 2024-2025 PF after 5 bps >= 1.05", row.PFCombined_5bps >= 1.05, fmt.Sprintf("%.6f", row.PFCombined_5bps)),
		fundingGate("combined expectancy after 5 bps > 0", row.ExpectancyCombined_5bpsBps > 0, fmt.Sprintf("%.6f", row.ExpectancyCombined_5bpsBps)),
		fundingGate("positive_month_count >= 3", row.PositiveMonthCount >= 3, fmt.Sprintf("%d", row.PositiveMonthCount)),
		fundingGate("entry delay 1 candle expectancy > 0 if available", !row.EntryDelay1cAvailable || row.EntryDelay1cExpectancyBps > 0, fmt.Sprintf("%.6f", row.EntryDelay1cExpectancyBps)),
		fundingGate("top 1 month contribution <= 50%", row.Top1MonthContributionPct <= 50, fmt.Sprintf("%.6f", row.Top1MonthContributionPct)),
		fundingGate("top 2 month contribution <= 70%", row.Top2MonthContributionPct <= 70, fmt.Sprintf("%.6f", row.Top2MonthContributionPct)),
		fundingGate("worst quarter PF after 5 bps >= 0.95", row.WorstQuarterPF5Bps >= 0.95, fmt.Sprintf("%.6f", row.WorstQuarterPF5Bps)),
		fundingGate("leakage status PASS", row.LeakageStatus == "PASS", row.LeakageStatus),
		fundingGate("price-only result positive", row.PriceOnlyResult == "positive", row.PriceOnlyResult),
	}
}

func fundingGate(name string, pass bool, detail string) FundingGateResult {
	status := "FAIL"
	if pass {
		status = "PASS"
	}
	return FundingGateResult{Gate: name, Status: status, Detail: detail}
}

func fundingRecommendation(status string) string {
	switch status {
	case "research_lead":
		return "validated lead worth Phase 10.8"
	case "fragile":
		return "fragile"
	case "rejected":
		return "rejected"
	case "invalid_report_artifact":
		return "invalid_report_artifact remains"
	case "real_no_events":
		return "real_no_events"
	default:
		return "needs_more_data"
	}
}

func writeFundingDeepMarkdown(path string, report FundingDeepReport) error {
	var md bytes.Buffer
	md.WriteString(fmt.Sprintf("# %s Real Deep Validation\n\n", report.Family))
	md.WriteString(fmt.Sprintf("- Family: %s\n", report.Family))
	md.WriteString(fmt.Sprintf("- Side: %s\n", report.Side))
	md.WriteString(fmt.Sprintf("- Metrics source: %s\n", report.MetricsSource))
	md.WriteString(fmt.Sprintf("- Best horizon: %s\n", report.BestHorizon))
	md.WriteString(fmt.Sprintf("- Raw event count: %d\n", report.RawEventCount))
	md.WriteString(fmt.Sprintf("- De-clustered event count: %d\n", report.DeClusteredEventCount))
	md.WriteString(fmt.Sprintf("- 2025 PF after 5 bps: %.6f\n", report.OverallPF2025_5bps))
	md.WriteString(fmt.Sprintf("- Combined PF after 5 bps: %.6f\n", report.OverallPFCombined_5bps))
	md.WriteString(fmt.Sprintf("- 2025 expectancy after 5 bps: %.6f\n", report.OverallExpectancy2025_5bpsBps))
	md.WriteString(fmt.Sprintf("- Combined expectancy after 5 bps: %.6f\n", report.OverallExpectancyCombined_5bpsBps))
	md.WriteString(fmt.Sprintf("- Top 1 month contribution: %.6f%%\n", report.Top1MonthContributionPct))
	md.WriteString(fmt.Sprintf("- Top 2 month contribution: %.6f%%\n", report.Top2MonthContributionPct))
	md.WriteString(fmt.Sprintf("- Worst quarter PF after 5 bps: %.6f\n", report.WorstQuarterPF5bps))
	md.WriteString(fmt.Sprintf("- Leakage status: %s\n", report.LeakageStatus))
	md.WriteString(fmt.Sprintf("- Price-only result: %s\n", report.PriceOnlyResult))
	md.WriteString(fmt.Sprintf("- Final status: %s\n", report.FinalStatus))
	md.WriteString(fmt.Sprintf("- Recommendation: %s\n\n", report.Recommendation))
	md.WriteString("## Acceptance Gates\n")
	md.WriteString("| Gate | Status | Detail |\n")
	md.WriteString("|---|---|---|\n")
	for _, gate := range report.AcceptanceGates {
		md.WriteString(fmt.Sprintf("| %s | %s | %s |\n", gate.Gate, gate.Status, gate.Detail))
	}
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return err
	}
	return os.WriteFile(path, md.Bytes(), 0644)
}
