package app

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/davidmiguel22573/ak-engine/internal/features"
	"github.com/davidmiguel22573/ak-engine/internal/regime"
)

func phase10FundingEventBuildReport(cfg phase10FundingEventPipelineConfig, manifest *Phase10FundingEventManifest, leaderboard *FundingLeaderboardReport, processedKeys map[string]struct{}, inputChunksRebuilt int) phase10FundingEventPipelineReport {
	prefix := strings.TrimSuffix(filepath.Base(cfg.Out), ".md")
	report := phase10FundingEventPipelineReport{
		Status:                  "needs_more_data",
		DetailedStatus:          "needs_more_data",
		SymbolsProcessed:        append([]string(nil), cfg.Symbols...),
		MonthsProcessed:         append([]string(nil), cfg.Months...),
		ChunksProcessed:         len(processedKeys),
		InputChunksRebuilt:      inputChunksRebuilt,
		ManifestPath:            cfg.ManifestPath,
		IntegrityAuditPath:      filepath.Join(cfg.ReportsDir, prefix+"_integrity_audit.md"),
		LeaderboardPath:         filepath.Join(cfg.ReportsDir, prefix+"_leaderboard.md"),
		NegativeFundingDeepPath: filepath.Join(cfg.ReportsDir, prefix+"_NegativeFundingLong_deep.md"),
	}
	for key := range processedKeys {
		status := manifest.Chunks[key]
		if status == nil {
			continue
		}
		report.RealEventRows += status.EventRows
		report.HeavyFilesDeleted += len(status.DeletedHeavyFiles)
		report.BytesFreed += status.BytesFreed
		if status.EventEvalStatus == "DONE" {
			report.EventFilesCreated++
		}
		if status.EventEvalStatus == "DONE" && status.EventRows == 0 {
			report.ZeroEventMonths++
		}
		if phase10FundingChunkMissingInput(status) {
			report.MissingInputMonths++
		}
	}
	if report.MissingInputMonths > 0 {
		report.Status = "pipeline_blocked"
		report.DetailedStatus = "pipeline_blocked_missing_ephemeral_chunks"
		return report
	}
	for key := range processedKeys {
		status := manifest.Chunks[key]
		if status != nil && status.ContextStatus != "" && status.ContextStatus != "PASS" {
			report.Status = "pipeline_blocked"
			report.DetailedStatus = strings.ToLower(status.ContextStatus)
			return report
		}
	}
	if leaderboard != nil {
		if leaderboard.Summary.MissingEventFileCount > 0 {
			report.Status = "pipeline_blocked"
			report.DetailedStatus = "pipeline_blocked_missing_event_files"
			return report
		}
		if leaderboard.Summary.VerdictCounts["unsupported_context"] > 0 {
			report.Status = "pipeline_blocked"
			report.DetailedStatus = "unsupported_context"
			return report
		}
		if leaderboard.Summary.TotalEventRows == 0 && report.EventFilesCreated > 0 {
			report.Status = "real_no_events"
			report.DetailedStatus = "real_no_events"
			return report
		}
		for _, lead := range leaderboard.Summary.ResearchLeads {
			if lead != "" {
				report.Status = "validated_research_lead"
				report.DetailedStatus = "validated_research_lead"
				return report
			}
		}
		if leaderboard.Summary.VerdictCounts["fragile"] > 0 {
			report.Status = "fragile"
			report.DetailedStatus = "fragile"
			return report
		}
		if leaderboard.Summary.VerdictCounts["rejected"] > 0 {
			report.Status = "rejected"
			report.DetailedStatus = "rejected"
			return report
		}
	}
	return report
}

func phase10FundingChunkMissingInput(status *Phase10FundingEventChunkStatus) bool {
	return strings.HasPrefix(status.FeatureBuildStatus, "FAILED") ||
		strings.HasPrefix(status.RegimeClassifyStatus, "FAILED") ||
		strings.HasPrefix(status.FundingJoinStatus, "FAILED") ||
		status.EventEvalStatus == "FAILED_VERIFY"
}

func writePhase10FundingEventPipelineReport(cfg phase10FundingEventPipelineConfig, report phase10FundingEventPipelineReport) error {
	if err := writeFundingJSONReport(strings.TrimSuffix(cfg.Out, filepath.Ext(cfg.Out))+".json", report); err != nil {
		return err
	}
	var md bytes.Buffer
	md.WriteString("# Phase 10.7E Funding Event Pipeline\n\n")
	md.WriteString(fmt.Sprintf("- Status: %s\n", report.Status))
	md.WriteString(fmt.Sprintf("- Detailed status: %s\n", report.DetailedStatus))
	md.WriteString(fmt.Sprintf("- Symbols processed: %s\n", strings.Join(report.SymbolsProcessed, ",")))
	md.WriteString(fmt.Sprintf("- Months processed: %s\n", strings.Join(report.MonthsProcessed, ",")))
	md.WriteString(fmt.Sprintf("- Input chunks rebuilt: %d\n", report.InputChunksRebuilt))
	md.WriteString(fmt.Sprintf("- Event files created: %d\n", report.EventFilesCreated))
	md.WriteString(fmt.Sprintf("- Real event rows: %d\n", report.RealEventRows))
	md.WriteString(fmt.Sprintf("- Zero-event months: %d\n", report.ZeroEventMonths))
	md.WriteString(fmt.Sprintf("- Missing-input months: %d\n", report.MissingInputMonths))
	md.WriteString(fmt.Sprintf("- Heavy files deleted: %d\n", report.HeavyFilesDeleted))
	md.WriteString(fmt.Sprintf("- Bytes freed: %d\n", report.BytesFreed))
	md.WriteString(fmt.Sprintf("- Manifest: %s\n", report.ManifestPath))
	md.WriteString(fmt.Sprintf("- Leaderboard: %s\n", report.LeaderboardPath))
	md.WriteString(fmt.Sprintf("- NegativeFundingLong deep report: %s\n", report.NegativeFundingDeepPath))
	if err := os.MkdirAll(filepath.Dir(cfg.Out), 0755); err != nil {
		return err
	}
	return atomicWriteFile(cfg.Out, md.Bytes(), 0644)
}

func writePhase10FundingEventReports(cfg phase10FundingEventPipelineConfig, leaderboard FundingLeaderboardReport, integrity FundingEventIntegrityAudit) error {
	if err := os.MkdirAll(cfg.ReportsDir, 0755); err != nil {
		return err
	}
	prefix := strings.TrimSuffix(filepath.Base(cfg.Out), ".md")

	if err := writeFundingJSONReport(filepath.Join(cfg.ReportsDir, prefix+"_leaderboard.json"), leaderboard); err != nil {
		return err
	}
	if err := atomicWriteFile(filepath.Join(cfg.ReportsDir, prefix+"_leaderboard.md"), phase10FundingEventMarkdown(renderFundingLeaderboardMarkdown(leaderboard)), 0644); err != nil {
		return err
	}
	if err := writeFundingJSONReport(filepath.Join(cfg.ReportsDir, prefix+"_integrity_audit.json"), integrity); err != nil {
		return err
	}
	if err := atomicWriteFile(filepath.Join(cfg.ReportsDir, prefix+"_integrity_audit.md"), phase10FundingEventMarkdown(renderFundingIntegrityMarkdown(integrity)), 0644); err != nil {
		return err
	}
	deep := buildFundingDeepReport(leaderboard, "NegativeFundingLong", "long")
	if leaderboard.Summary.TotalEventRows == 0 && leaderboard.Summary.EventFilesFound > 0 && leaderboard.Summary.MissingEventFileCount == 0 && leaderboard.Summary.VerdictCounts["unsupported_context"] == 0 {
		deep.FinalStatus = "real_no_events"
		deep.Recommendation = "real_no_events"
	}
	if err := writeFundingJSONReport(filepath.Join(cfg.ReportsDir, prefix+"_NegativeFundingLong_deep.json"), deep); err != nil {
		return err
	}
	if err := writeFundingDeepMarkdown(filepath.Join(cfg.ReportsDir, prefix+"_NegativeFundingLong_deep.md"), deep); err != nil {
		return err
	}
	return writePhase10Funding7FReports(cfg, leaderboard, deep)
}

func phase10FundingEventMarkdown(data []byte) []byte {
	text := strings.ReplaceAll(string(data), "Phase 10.7D", "Phase 10.7E")
	text = strings.ReplaceAll(text, "10.7D", "10.7E")
	return []byte(text)
}

type Phase10FundingContextAudit struct {
	Symbol                   string         `json:"symbol"`
	Month                    string         `json:"month"`
	FeatureRows              int            `json:"feature_rows"`
	RegimeRows               int            `json:"regime_rows"`
	BTCReturn60NonzeroCount  int            `json:"btc_return_60_nonzero_count"`
	ETHReturn60NonzeroCount  int            `json:"eth_return_60_nonzero_count"`
	BTCReturn60NonzeroPct    float64        `json:"btc_return_60_nonzero_pct"`
	ETHReturn60NonzeroPct    float64        `json:"eth_return_60_nonzero_pct"`
	BTCReturn60Min           float64        `json:"btc_return_60_min"`
	BTCReturn60Median        float64        `json:"btc_return_60_median"`
	BTCReturn60Max           float64        `json:"btc_return_60_max"`
	ETHReturn60Min           float64        `json:"eth_return_60_min"`
	ETHReturn60Median        float64        `json:"eth_return_60_median"`
	ETHReturn60Max           float64        `json:"eth_return_60_max"`
	MarketBetaCounts         map[string]int `json:"market_beta_counts"`
	UnsupportedContextCount  int            `json:"unsupported_context_count"`
	UnsupportedContextReason string         `json:"unsupported_context_reason"`
	ContextStatus            string         `json:"context_status"`
}

func writePhase10FundingContextAudit(cfg phase10FundingEventPipelineConfig, paths phase10FundingEventChunkPaths, status *Phase10FundingEventChunkStatus) (Phase10FundingContextAudit, error) {
	audit := Phase10FundingContextAudit{
		Symbol:           status.Symbol,
		Month:            status.Month,
		MarketBetaCounts: make(map[string]int),
		ContextStatus:    "PASS",
	}
	rows, err := features.ReadRowsJSON(paths.FeatureContextFile)
	if err != nil {
		audit.ContextStatus = "CONTEXT_JOIN_FAILED"
		audit.UnsupportedContextReason = err.Error()
		_ = writeFundingJSONReport(paths.ContextAuditFile, audit)
		return audit, nil
	}
	labels, err := regime.ReadLabelsJSON(paths.RegimeContextFile)
	if err != nil {
		audit.ContextStatus = "CONTEXT_JOIN_FAILED"
		audit.UnsupportedContextReason = err.Error()
		_ = writeFundingJSONReport(paths.ContextAuditFile, audit)
		return audit, nil
	}
	audit.FeatureRows = len(rows)
	audit.RegimeRows = len(labels)
	if len(rows) != len(labels) {
		audit.ContextStatus = "CONTEXT_JOIN_FAILED"
		audit.UnsupportedContextReason = fmt.Sprintf("feature rows %d != regime rows %d", len(rows), len(labels))
	}
	var btcValues []float64
	var ethValues []float64
	for _, row := range rows {
		btcValues = append(btcValues, row.BTCReturn60)
		ethValues = append(ethValues, row.ETHReturn60)
		if row.BTCReturn60 != 0 {
			audit.BTCReturn60NonzeroCount++
		}
		if row.ETHReturn60 != 0 {
			audit.ETHReturn60NonzeroCount++
		}
	}
	audit.BTCReturn60NonzeroPct = pct(audit.BTCReturn60NonzeroCount, len(rows))
	audit.ETHReturn60NonzeroPct = pct(audit.ETHReturn60NonzeroCount, len(rows))
	audit.BTCReturn60Min, audit.BTCReturn60Median, audit.BTCReturn60Max = minMedianMax(btcValues)
	audit.ETHReturn60Min, audit.ETHReturn60Median, audit.ETHReturn60Max = minMedianMax(ethValues)
	for _, label := range labels {
		beta := strings.TrimSpace(label.MarketBeta)
		if beta == "" {
			beta = "missing"
		}
		audit.MarketBetaCounts[beta]++
		if !label.Warmup && fundingUnsupportedContextLabel(label) {
			audit.UnsupportedContextCount++
		}
	}
	target := strings.ToUpper(status.Symbol)
	switch {
	case target == "BTCUSDT":
		audit.ContextStatus = "SELF_CONTEXT_UNSUPPORTED"
		audit.UnsupportedContextReason = "BTCUSDT target has no safe non-self context"
	case audit.ContextStatus == "CONTEXT_JOIN_FAILED":
	case audit.BTCReturn60NonzeroCount == 0:
		audit.ContextStatus = "MISSING_BTC_CONTEXT"
		audit.UnsupportedContextReason = "btc_return_60 is zero for every row"
	case target != "ETHUSDT" && audit.ETHReturn60NonzeroCount == 0:
		audit.ContextStatus = "MISSING_ETH_CONTEXT"
		audit.UnsupportedContextReason = "eth_return_60 is zero for every row"
	case audit.UnsupportedContextCount > 0 || len(audit.MarketBetaCounts) == 0:
		audit.ContextStatus = "REGIME_BETA_MISSING"
		audit.UnsupportedContextReason = "non-warmup regime labels missing usable market_beta"
	default:
		audit.ContextStatus = "PASS"
		audit.UnsupportedContextReason = ""
	}
	return audit, writeFundingJSONReport(paths.ContextAuditFile, audit)
}

func writePhase10Funding7FReports(cfg phase10FundingEventPipelineConfig, leaderboard FundingLeaderboardReport, deep FundingDeepReport) error {
	audits, err := loadPhase10FundingContextAudits(cfg)
	if err != nil {
		return err
	}
	if len(audits) > 0 {
		if err := writeFundingJSONReport(filepath.Join(cfg.ReportsDir, "phase10_7f_context_audit.json"), audits); err != nil {
			return err
		}
		if err := atomicWriteFile(filepath.Join(cfg.ReportsDir, "phase10_7f_context_audit.md"), renderPhase10FundingContextAuditMarkdown(audits), 0644); err != nil {
			return err
		}
	}
	for _, symbol := range cfg.Symbols {
		for _, month := range cfg.Months {
			paths := phase10FundingEventPaths(cfg, symbol, month)
			diagnostics, err := readFundingDiagnostics(paths.DiagnosticsFile)
			if err != nil {
				if os.IsNotExist(err) {
					continue
				}
				return err
			}
			base := fmt.Sprintf("phase10_7f_%s_%s_funding_diagnostics", symbol, strings.ReplaceAll(month, "-", "_"))
			if err := writeFundingJSONReport(filepath.Join(cfg.ReportsDir, base+".json"), diagnostics); err != nil {
				return err
			}
			if err := atomicWriteFile(filepath.Join(cfg.ReportsDir, base+".md"), renderFundingDiagnosticsMarkdown(diagnostics), 0644); err != nil {
				return err
			}
		}
	}
	if err := writeFundingJSONReport(filepath.Join(cfg.ReportsDir, "phase10_7f_NegativeFundingLong_real_deep.json"), deep); err != nil {
		return err
	}
	if err := writeFundingDeepMarkdown(filepath.Join(cfg.ReportsDir, "phase10_7f_NegativeFundingLong_real_deep.md"), deep); err != nil {
		return err
	}
	_ = leaderboard
	return nil
}

func loadPhase10FundingContextAudits(cfg phase10FundingEventPipelineConfig) ([]Phase10FundingContextAudit, error) {
	var audits []Phase10FundingContextAudit
	for _, symbol := range cfg.Symbols {
		for _, month := range cfg.Months {
			path := phase10FundingEventPaths(cfg, symbol, month).ContextAuditFile
			data, err := os.ReadFile(path)
			if err != nil {
				if os.IsNotExist(err) {
					continue
				}
				return nil, err
			}
			var audit Phase10FundingContextAudit
			if err := json.Unmarshal(data, &audit); err != nil {
				return nil, err
			}
			audits = append(audits, audit)
		}
	}
	return audits, nil
}

func renderPhase10FundingContextAuditMarkdown(audits []Phase10FundingContextAudit) []byte {
	var md bytes.Buffer
	md.WriteString("# Phase 10.7F Context Audit\n\n")
	md.WriteString("| Symbol | Month | Status | Feature Rows | Regime Rows | BTC Nonzero % | ETH Nonzero % | Unsupported | Reason |\n")
	md.WriteString("|---|---|---|---:|---:|---:|---:|---:|---|\n")
	for _, audit := range audits {
		md.WriteString(fmt.Sprintf("| %s | %s | %s | %d | %d | %.2f | %.2f | %d | %s |\n",
			audit.Symbol, audit.Month, audit.ContextStatus, audit.FeatureRows, audit.RegimeRows,
			audit.BTCReturn60NonzeroPct, audit.ETHReturn60NonzeroPct, audit.UnsupportedContextCount,
			audit.UnsupportedContextReason))
	}
	return md.Bytes()
}

func readFundingDiagnostics(path string) (FundingDiagnostics, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return FundingDiagnostics{}, err
	}
	var diagnostics FundingDiagnostics
	if err := json.Unmarshal(data, &diagnostics); err != nil {
		return FundingDiagnostics{}, err
	}
	return diagnostics, nil
}

func renderFundingDiagnosticsMarkdown(diagnostics FundingDiagnostics) []byte {
	var md bytes.Buffer
	md.WriteString("# Phase 10.7F Funding Diagnostics\n\n")
	md.WriteString(fmt.Sprintf("- Symbol: %s\n", diagnostics.Symbol))
	md.WriteString(fmt.Sprintf("- Month: %s\n", diagnostics.Month))
	md.WriteString(fmt.Sprintf("- Rows seen: %d\n", diagnostics.RowsSeen))
	md.WriteString(fmt.Sprintf("- Rows with funding: %d\n", diagnostics.RowsWithFunding))
	md.WriteString(fmt.Sprintf("- Rows unknown funding: %d\n", diagnostics.RowsUnknownFunding))
	md.WriteString(fmt.Sprintf("- Rows warmup funding: %d\n", diagnostics.RowsWarmupFunding))
	md.WriteString(fmt.Sprintf("- Rows context unsupported: %d\n", diagnostics.RowsContextUnsupported))
	md.WriteString(fmt.Sprintf("- Rows beta flat: %d\n", diagnostics.RowsBetaFlat))
	md.WriteString(fmt.Sprintf("- Rows beta up: %d\n", diagnostics.RowsBetaUp))
	md.WriteString(fmt.Sprintf("- Rows beta down: %d\n", diagnostics.RowsBetaDown))
	md.WriteString(fmt.Sprintf("- Negative funding candidates: %d\n", diagnostics.NegativeFundingCandidates))
	md.WriteString(fmt.Sprintf("- Negative funding candidates after context: %d\n", diagnostics.NegativeFundingCandidatesAfterContext))
	md.WriteString(fmt.Sprintf("- Negative funding events emitted: %d\n", diagnostics.NegativeFundingEventsEmitted))
	md.WriteString(fmt.Sprintf("- Positive funding candidates: %d\n", diagnostics.PositiveFundingCandidates))
	md.WriteString(fmt.Sprintf("- Funding flip candidates: %d\n", diagnostics.FundingFlipCandidates))
	md.WriteString(fmt.Sprintf("- Rejected by context: %d\n", diagnostics.RejectedByContext))
	md.WriteString(fmt.Sprintf("- Rejected by funding threshold: %d\n", diagnostics.RejectedByFundingThreshold))
	md.WriteString(fmt.Sprintf("- Rejected by warmup: %d\n", diagnostics.RejectedByWarmup))
	md.WriteString(fmt.Sprintf("- Rejected by missing forward window: %d\n", diagnostics.RejectedByMissingForwardWindow))
	return md.Bytes()
}

func pct(count, total int) float64 {
	if total == 0 {
		return 0
	}
	return float64(count) / float64(total) * 100
}

func minMedianMax(values []float64) (float64, float64, float64) {
	if len(values) == 0 {
		return 0, 0, 0
	}
	cp := append([]float64(nil), values...)
	sort.Float64s(cp)
	return cp[0], median(cp), cp[len(cp)-1]
}

func enrichPhase10FundingEventIntegrity(audit *FundingEventIntegrityAudit, manifest *Phase10FundingEventManifest, processedKeys map[string]struct{}, inputChunksRebuilt int) {
	audit.InputChunksRebuilt = inputChunksRebuilt
	audit.EventFilesCreated = 0
	missing := make(map[string]bool)
	for _, path := range audit.MissingEventFiles {
		missing[path] = true
	}
	for key := range processedKeys {
		status := manifest.Chunks[key]
		if status == nil {
			continue
		}
		if status.EventEvalStatus == "DONE" && status.EventFile != "" && !missing[status.EventFile] {
			audit.EventFilesCreated++
		}
	}
	updateFundingIntegrityCheck(audit, "input chunks rebuilt", true, fmt.Sprintf("chunks=%d", audit.InputChunksRebuilt))
	updateFundingIntegrityCheck(audit, "event files created", audit.EventFilesCreated == audit.EventFilesFound, fmt.Sprintf("created=%d found=%d", audit.EventFilesCreated, audit.EventFilesFound))
}

func updateFundingIntegrityCheck(audit *FundingEventIntegrityAudit, name string, pass bool, detail string) {
	status := "PASS"
	if !pass {
		status = "FAIL"
	}
	for i := range audit.Checks {
		if audit.Checks[i].Name == name {
			audit.Checks[i].Status = status
			audit.Checks[i].Detail = detail
			return
		}
	}
	audit.Checks = append(audit.Checks, FundingIntegrityCheck{Name: name, Status: status, Detail: detail})
}

func monthLastDay(month string) string {
	t, err := time.Parse("2006-01", month)
	if err != nil {
		return month + "-31"
	}
	return t.AddDate(0, 1, -1).Format("2006-01-02")
}
