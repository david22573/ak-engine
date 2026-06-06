package research

import (
	"encoding/json"
	"fmt"
	"os"
	"sort"

	"github.com/davidmiguel22573/ak-engine/internal/regime"
)

type EvalReport struct {
	TotalRows              int                       `json:"total_rows"`
	LeakageStatus          string                    `json:"leakage_status"`
	VolDistribution        map[string]int            `json:"vol_distribution"`
	TrendDistribution      map[string]int            `json:"trend_distribution"`
	CompositeDistribution  map[string]int            `json:"composite_distribution"`
	MarketBetaDistribution map[string]int            `json:"market_beta_distribution"`
	VolTrendMatrix         map[string]map[string]int `json:"vol_trend_matrix"`
	CompositeTransitions   map[string]map[string]int `json:"composite_transitions"`
	Warnings               []string                  `json:"warnings"`
}

func GenerateReport(labels []regime.Label) (EvalReport, string, error) {
	report := EvalReport{
		TotalRows:              len(labels),
		VolDistribution:        make(map[string]int),
		TrendDistribution:      make(map[string]int),
		CompositeDistribution:  make(map[string]int),
		MarketBetaDistribution: make(map[string]int),
		VolTrendMatrix:         make(map[string]map[string]int),
		CompositeTransitions:   make(map[string]map[string]int),
	}

	leakReport := CheckLabels(labels)
	report.LeakageStatus = leakReport.Status

	volTypes := []string{"normal", "compressed", "expanded", "shock"}
	trendTypes := []string{"range", "bull_trend", "bear_trend", "chop"}

	for _, v := range volTypes {
		report.VolTrendMatrix[v] = make(map[string]int)
		for _, t := range trendTypes {
			report.VolTrendMatrix[v][t] = 0
		}
	}

	for i, l := range labels {
		if l.Warmup {
			continue
		}
		// Increment distributions
		report.VolDistribution[l.Volatility]++
		report.TrendDistribution[l.Trend]++
		report.CompositeDistribution[l.Composite]++
		report.MarketBetaDistribution[l.MarketBeta]++

		// Matrix
		if report.VolTrendMatrix[l.Volatility] != nil {
			report.VolTrendMatrix[l.Volatility][l.Trend]++
		}

		// Transitions
		if i > 0 && !labels[i-1].Warmup {
			prevComp := labels[i-1].Composite
			currComp := l.Composite
			if report.CompositeTransitions[prevComp] == nil {
				report.CompositeTransitions[prevComp] = make(map[string]int)
			}
			report.CompositeTransitions[prevComp][currComp]++
		}
	}

	// Generate warnings for low-sample buckets (< 100)
	// 1. Composite buckets
	for comp, count := range report.CompositeDistribution {
		if count < 100 {
			report.Warnings = append(report.Warnings, fmt.Sprintf("UNTRUSTED_LOW_SAMPLE: composite %s has only %d rows (< 100)", comp, count))
		}
	}
	// 2. Vol x Trend cells
	for vol, trends := range report.VolTrendMatrix {
		for trend, count := range trends {
			if count > 0 && count < 100 {
				report.Warnings = append(report.Warnings, fmt.Sprintf("UNTRUSTED_LOW_SAMPLE: matrix %s x %s has only %d rows (< 100)", vol, trend, count))
			}
		}
	}

	// Build markdown representation
	md := ""
	md += "# Regime Evaluation Report\n\n"
	md += "## Summary\n"
	md += fmt.Sprintf("- **Total Rows**: %d\n", report.TotalRows)
	md += fmt.Sprintf("- **Leakage Status**: %s\n\n", report.LeakageStatus)

	md += "## Regime Distribution\n"
	md += "| Volatility | Count | Percentage |\n"
	md += "|---|---|---|\n"
	// Sort keys
	var volKeys []string
	for k := range report.VolDistribution {
		volKeys = append(volKeys, k)
	}
	sort.Strings(volKeys)
	for _, k := range volKeys {
		cnt := report.VolDistribution[k]
		pct := 0.0
		if report.TotalRows > 0 {
			pct = float64(cnt) / float64(report.TotalRows) * 100
		}
		md += fmt.Sprintf("| %s | %d | %.2f%% |\n", k, cnt, pct)
	}
	md += "\n"

	md += "## Composite Distribution\n"
	md += "| Composite | Count | Percentage |\n"
	md += "|---|---|---|\n"
	var compKeys []string
	for k := range report.CompositeDistribution {
		compKeys = append(compKeys, k)
	}
	sort.Strings(compKeys)
	for _, k := range compKeys {
		cnt := report.CompositeDistribution[k]
		pct := 0.0
		if report.TotalRows > 0 {
			pct = float64(cnt) / float64(report.TotalRows) * 100
		}
		md += fmt.Sprintf("| %s | %d | %.2f%% |\n", k, cnt, pct)
	}
	md += "\n"

	md += "## Volatility x Trend Matrix\n"
	md += "| Volatility \\ Trend | range | bull_trend | bear_trend | chop |\n"
	md += "|---|---|---|---|---|\n"
	for _, v := range volTypes {
		md += fmt.Sprintf("| %s | %d | %d | %d | %d |\n",
			v,
			report.VolTrendMatrix[v]["range"],
			report.VolTrendMatrix[v]["bull_trend"],
			report.VolTrendMatrix[v]["bear_trend"],
			report.VolTrendMatrix[v]["chop"],
		)
	}
	md += "\n"

	md += "## Market Beta Distribution\n"
	md += "| Market Beta | Count | Percentage |\n"
	md += "|---|---|---|\n"
	var betaKeys []string
	for k := range report.MarketBetaDistribution {
		betaKeys = append(betaKeys, k)
	}
	sort.Strings(betaKeys)
	for _, k := range betaKeys {
		cnt := report.MarketBetaDistribution[k]
		pct := 0.0
		if report.TotalRows > 0 {
			pct = float64(cnt) / float64(report.TotalRows) * 100
		}
		md += fmt.Sprintf("| %s | %d | %.2f%% |\n", k, cnt, pct)
	}
	md += "\n"

	md += "## Regime Transitions\n"
	md += "| From Composite \\ To |"
	for _, c := range compKeys {
		md += " " + c + " |"
	}
	md += "\n|---|"
	for range compKeys {
		md += "---|"
	}
	md += "\n"
	sort.Strings(compKeys)
	for _, from := range compKeys {
		md += fmt.Sprintf("| %s |", from)
		for _, to := range compKeys {
			count := 0
			if report.CompositeTransitions[from] != nil {
				count = report.CompositeTransitions[from][to]
			}
			md += fmt.Sprintf(" %d |", count)
		}
		md += "\n"
	}
	md += "\n"

	md += "## Low Sample Warnings\n"
	if len(report.Warnings) == 0 {
		md += "No low-sample warnings.\n\n"
	} else {
		for _, w := range report.Warnings {
			md += fmt.Sprintf("- %s\n", w)
		}
		md += "\n"
	}

	md += "## Leakage Status\n"
	if report.LeakageStatus == "PASS" {
		md += "No data leakage issues detected.\n"
	} else {
		md += "WARNING: Data leakage issues detected! Please check JSON report for detailed issues.\n"
	}

	return report, md, nil
}

func WriteEvalReport(jsonPath, mdPath string, report EvalReport, md string) error {
	jsonData, err := json.MarshalIndent(report, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal JSON report: %w", err)
	}
	if err := os.WriteFile(jsonPath, jsonData, 0644); err != nil {
		return fmt.Errorf("write JSON file: %w", err)
	}
	if err := os.WriteFile(mdPath, []byte(md), 0644); err != nil {
		return fmt.Errorf("write MD file: %w", err)
	}
	return nil
}
