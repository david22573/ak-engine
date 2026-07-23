package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
)

type AuditReport struct {
	FinalLabel           string `json:"final_label"`
	WorstQuarter         string `json:"worst_quarter"`
	BestQuarterBreakdown struct {
		Quarter string  `json:"quarter"`
		Pf5bps  float64 `json:"pf_5bps"`
	} `json:"best_quarter_breakdown"`
	WorstQuarterBreakdown struct {
		Quarter string  `json:"quarter"`
		Pf5bps  float64 `json:"pf_5bps"`
	} `json:"worst_quarter_breakdown"`
}

type EvalReport struct {
	RawEventDetailRetained bool `json:"raw_event_detail_retained"`
	RetainedSummaries      []struct {
		Symbol string `json:"symbol"`
		Month  string `json:"month"`
	} `json:"retained_summaries"`
}

type FilterDesignJson struct {
	FinalLabel       string   `json:"final_label"`
	FieldsAvailable  []string `json:"fields_available"`
	FieldsMissing    []string `json:"fields_missing"`
	FiltersRejected  []string `json:"filters_rejected"`
	FiltersTested    []string `json:"filters_tested"`
	TopCandidates    []string `json:"top_candidates"`
	Q1_2025_Reduced  bool     `json:"q1_2025_reduced"`
	WorstQuarterPf   float64  `json:"worst_quarter_pf"`
	ClusteringStatus string   `json:"clustering_status"`
	Recommendation   string   `json:"recommendation"`
}

func main() {
	auditData, err := ioutil.ReadFile("runs/reports/phase12_4_downtrend_midvol_relief_near_miss_audit.json")
	if err != nil {
		fmt.Println("Error reading audit data:", err)
		return
	}
	var audit AuditReport
	if err := json.Unmarshal(auditData, &audit); err != nil {
		fmt.Println("Error unmarshaling audit data:", err)
		return
	}

	evalData, err := ioutil.ReadFile("runs/reports/phase12_3_downtrend_midvol_relief_eval.json")
	if err != nil {
		fmt.Println("Error reading eval data:", err)
		return
	}
	var eval EvalReport
	if err := json.Unmarshal(evalData, &eval); err != nil {
		fmt.Println("Error unmarshaling eval data:", err)
		return
	}

	fieldsAvailable := []string{
		"symbol",
		"month",
		"quarter",
	}

	fieldsMissing := []string{
		"regime_fields",
		"volatility_bucket",
		"funding_bucket",
		"btc_eth_context",
		"trend_strength",
		"volume_liquidity_buckets",
		"cluster_spacing",
		"cluster_timestamps",
	}

	filtersRejected := []string{
		"exclude Q1 2025 (disallowed hindsight/month exclusion)",
		"exclude weak symbols (disallowed hindsight)",
	}

	report := FilterDesignJson{
		FinalLabel:       "NO_VALID_FILTER_FOUND",
		FieldsAvailable:  fieldsAvailable,
		FieldsMissing:    fieldsMissing,
		FiltersRejected:  filtersRejected,
		FiltersTested:    []string{},
		TopCandidates:    []string{},
		Q1_2025_Reduced:  false,
		WorstQuarterPf:   audit.WorstQuarterBreakdown.Pf5bps,
		ClusteringStatus: "Cannot run stricter dedup. Event-level timestamps and cluster mapping are missing from retained summaries.",
		Recommendation:   "REJECT_CANDIDATE_STRUCTURAL_FAILURE",
	}

	outData, _ := json.MarshalIndent(report, "", "  ")
	err = ioutil.WriteFile("runs/reports/phase12_5_candidate_risk_filter_design.json", outData, 0644)
	if err != nil {
		fmt.Println("Error writing report json:", err)
		return
	}

	md := "# Phase 12.5 - Candidate Risk Filter Design\n\n"
	md += "## Executive Verdict\n"
	md += "The candidate strategy `DowntrendMidVolReliefLong240m` exhibited structural weakness in Q1 2025. This phase attempted to design a pre-entry risk filter to mitigate this damage without curve-fitting. However, **no valid filter could be designed** because the necessary pre-entry context fields (e.g., regime, funding, volatility, BTC/ETH context) were not retained during the Phase 12.3 evaluation (`raw_event_detail_retained` = false). The available symbol-month aggregate summaries do not provide the granularity required to test pre-entry condition filters or perform a stricter de-clustering pass.\n\n"

	md += "## Available vs. Missing Fields\n"
	md += "### Fields Available\n"
	for _, f := range fieldsAvailable {
		md += "- " + f + "\n"
	}
	md += "\n### Fields Missing\n"
	for _, f := range fieldsMissing {
		md += "- " + f + "\n"
	}

	md += "\n## Invalid / Leaky Filters Rejected\n"
	for _, f := range filtersRejected {
		md += "- " + f + "\n"
	}

	md += "\n## Valid Filters Tested\n"
	md += "None. Cannot test valid pre-entry filters without event-level properties.\n"

	md += "\n## Top Filter Candidates\n"
	md += "None.\n"

	md += "\n## Impact Analysis (Before / After)\n"
	md += "- **Q1 2025 Before:** PF " + fmt.Sprintf("%.4f", audit.WorstQuarterBreakdown.Pf5bps) + "\n"
	md += "- **Q1 2025 After:** N/A\n"
	md += "- **Worst Quarter Before:** PF " + fmt.Sprintf("%.4f", audit.WorstQuarterBreakdown.Pf5bps) + " (Q1 2025)\n"
	md += "- **Worst Quarter After:** N/A\n"
	md += "- **Q4 2024 Dependency Before:** High (PF " + fmt.Sprintf("%.4f", audit.BestQuarterBreakdown.Pf5bps) + ")\n"
	md += "- **Q4 2024 Dependency After:** N/A\n"

	md += "\n## Concentration Analysis\n"
	md += "Symbol/month concentration cannot be evaluated post-filter because no filter could be applied.\n"

	md += "\n## Clustering / De-duplication Analysis\n"
	md += "A stricter de-clustering sensitivity pass was requested, but is impossible because individual cluster mapping, size, and timestamps are absent from the symbol-month aggregate summaries.\n"

	md += "\n## Final Label\n"
	md += "**NO_VALID_FILTER_FOUND**\n"

	md += "\n## Recommendation for Phase 12.6\n"
	md += "**REJECT_CANDIDATE_STRUCTURAL_FAILURE**. Since the strategy suffered broad Q1 2025 structural failure and we lack the context fields to retroactively rescue it without a full rerun (which is expensive and risks curve-fitting), this candidate should be fully abandoned.\n"

	err = ioutil.WriteFile("runs/reports/phase12_5_candidate_risk_filter_design.md", []byte(md), 0644)
	if err != nil {
		fmt.Println("Error writing md:", err)
		return
	}
	fmt.Println("Phase 12.5 reports generated successfully.")
}
