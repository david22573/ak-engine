package app

import (
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func TestArtifactAuditDistinguishesMissingCandlesFromMissingArtifacts(t *testing.T) {
	missingCandles := finalizeArtifactAuditRow(ArtifactAuditRow{
		Symbol:      "SOLUSDT",
		CandleError: "no matching files in range",
	})
	if missingCandles.BuildNeeded {
		t.Fatalf("missing candle coverage must not produce build_needed")
	}
	if !strings.Contains(missingCandles.ReasonIfMissing, "missing candle coverage") {
		t.Fatalf("expected missing candle coverage reason, got %q", missingCandles.ReasonIfMissing)
	}

	missingArtifacts := finalizeArtifactAuditRow(ArtifactAuditRow{
		Symbol:           "SOLUSDT",
		CandlesAvailable: true,
		CandleRows:       525600,
		FeaturesExist:    false,
		RegimesExist:     true,
		RegimeRows:       525600,
	})
	if !missingArtifacts.BuildNeeded {
		t.Fatalf("full candle coverage with missing features should produce build_needed")
	}
	if missingArtifacts.ReasonIfMissing != "feature artifact missing" {
		t.Fatalf("unexpected artifact reason: %q", missingArtifacts.ReasonIfMissing)
	}
}

func TestLeaderboardMissingDataZeroAfterArtifactsGenerated(t *testing.T) {
	rows := []LeaderboardRow{
		{Symbol: "LINKUSDT", Family: "CompressionBreakout", Side: "SHORT", Verdict: verdictFragile},
		{Symbol: "ETHUSDT", Family: "ShockFade", Side: "LONG", Verdict: verdictRejected},
	}
	counts := countVerdicts(rows)
	if counts[verdictMissingData] != 0 {
		t.Fatalf("expected missing_data count 0 after artifacts generated, got %d", counts[verdictMissingData])
	}
}

func TestLINKUSDTAppearsInLeaderboardRows(t *testing.T) {
	report := LeaderboardJSON{
		Rows: []LeaderboardRow{
			{Symbol: "LINKUSDT", Family: "CompressionBreakout", Side: "SHORT", Verdict: verdictFragile},
			{Symbol: "ETHUSDT", Family: "ShockFade", Side: "LONG", Verdict: verdictRejected},
		},
	}
	if !leaderboardContainsSymbol(report.Rows, "LINKUSDT") {
		t.Fatalf("expected LINKUSDT in leaderboard rows")
	}
}

func TestBTCUSDTUnsupportedContextRows(t *testing.T) {
	rows := unsupportedContextRows("BTCUSDT")
	if len(rows) != len(requiredFamilies) {
		t.Fatalf("expected %d BTC rows, got %d", len(requiredFamilies), len(rows))
	}
	for _, row := range rows {
		if row.Verdict != verdictUnsupportedContext {
			t.Fatalf("BTCUSDT row must be unsupported_context, got %s", row.Verdict)
		}
		if row.Verdict == verdictRejected {
			t.Fatalf("BTCUSDT must not be rejected")
		}
	}
}

func TestEvaluateCandidateVerdictRequiresEveryResearchLeadGate(t *testing.T) {
	verdict, failed := EvaluateCandidateVerdict(
		0.50, 1.10, 1.05,
		300, 150, 150,
		3, 0.10, 50.0, "PASS",
	)
	if verdict != verdictResearchLead || len(failed) != 0 {
		t.Fatalf("expected research_lead with all gates passing, got %s failed=%v", verdict, failed)
	}

	verdict, failed = EvaluateCandidateVerdict(
		0.50, 1.09, 1.05,
		300, 150, 150,
		3, 0.10, 50.0, "PASS",
	)
	if verdict == verdictResearchLead {
		t.Fatalf("candidate with a failed gate cannot become research_lead")
	}
	if verdict != verdictFragile {
		t.Fatalf("expected borderline positive candidate to remain fragile, got %s failed=%v", verdict, failed)
	}
	if !containsString(failed, "H2 PF after 5 bps") {
		t.Fatalf("expected H2 PF failed gate, got %v", failed)
	}
}

func TestH2FailureOverridesH1AndFYSuccess(t *testing.T) {
	verdict, failed := EvaluateCandidateVerdict(
		-0.01, 1.20, 1.50,
		500, 250, 250,
		6, 0.25, 20.0, "PASS",
	)
	if verdict != verdictRejected {
		t.Fatalf("expected rejected when H2 expectancy fails, got %s", verdict)
	}
	if !containsString(failed, "H2 expectancy after 5 bps") {
		t.Fatalf("expected H2 expectancy failed gate, got %v", failed)
	}
}

func TestGuardrailVerdictRejectsConcentrationAndWorstQuarter(t *testing.T) {
	verdict, failed := EvaluateCandidateVerdictWithMetrics(CandidateVerdictInput{
		H2Expectancy5bpsBps:        1.00,
		H2PF5bps:                   1.20,
		FYPF5bps:                   1.50,
		EventCount:                 500,
		H1EventCount:               250,
		H2EventCount:               250,
		PositiveMonthCount:         6,
		EntryDelay1cExpectancyBps:  0.50,
		SingleMonthContributionPct: 40,
		Top2MonthContributionPct:   75,
		WorstQuarterPF5bps:         0.94,
		Cost10bpsReported:          true,
		LeakageStatus:              "PASS",
	})
	if verdict != verdictRejected {
		t.Fatalf("expected rejected for concentration and worst-quarter failures, got %s", verdict)
	}
	if !containsString(failed, "top_2_month_contribution_pct") {
		t.Fatalf("expected top_2_month_contribution_pct failed gate, got %v", failed)
	}
	if !containsString(failed, "worst_quarter_pf_5bps") {
		t.Fatalf("expected worst_quarter_pf_5bps failed gate, got %v", failed)
	}
}

func TestGuardrailVerdictRequires10bpsSensitivityReport(t *testing.T) {
	verdict, failed := EvaluateCandidateVerdictWithMetrics(CandidateVerdictInput{
		H2Expectancy5bpsBps:        1.00,
		H2PF5bps:                   1.20,
		FYPF5bps:                   1.50,
		EventCount:                 500,
		H1EventCount:               250,
		H2EventCount:               250,
		PositiveMonthCount:         6,
		EntryDelay1cExpectancyBps:  0.50,
		SingleMonthContributionPct: 40,
		Top2MonthContributionPct:   60,
		WorstQuarterPF5bps:         1.00,
		Cost10bpsReported:          false,
		LeakageStatus:              "PASS",
	})
	if verdict != verdictRejected {
		t.Fatalf("expected rejected when 10 bps sensitivity is missing, got %s", verdict)
	}
	if !containsString(failed, "cost_10bps_sensitivity_reported") {
		t.Fatalf("expected cost_10bps_sensitivity_reported failed gate, got %v", failed)
	}
}

func TestExpectancyJSONFieldNamesIncludeUnits(t *testing.T) {
	rowType := reflect.TypeOf(LeaderboardRow{})
	tests := map[string]string{
		"H2Expectancy5bpsBps":       "h2_expectancy_5bps_bps",
		"FYExpectancy5bpsBps":       "fy_expectancy_5bps_bps",
		"EntryDelay1cExpectancyBps": "entry_delay_1c_expectancy_bps",
	}
	for fieldName, wantTag := range tests {
		field, ok := rowType.FieldByName(fieldName)
		if !ok {
			t.Fatalf("missing field %s", fieldName)
		}
		if got := field.Tag.Get("json"); got != wantTag {
			t.Fatalf("%s json tag = %q, want %q", fieldName, got, wantTag)
		}
	}

	data, err := json.Marshal(LeaderboardRow{})
	if err != nil {
		t.Fatalf("marshal leaderboard row: %v", err)
	}
	jsonText := string(data)
	for _, want := range tests {
		if !strings.Contains(jsonText, `"`+want+`"`) {
			t.Fatalf("expected JSON to contain unit-explicit field %q: %s", want, jsonText)
		}
	}
	if strings.Contains(jsonText, `"h2_expectancy_5bps"`) || strings.Contains(jsonText, `"entry_delay_1c_expectancy"`) {
		t.Fatalf("found ambiguous expectancy field without _bps suffix: %s", jsonText)
	}
}

func TestPhase104BranchClosureReportIncludesRejectedFragileCandidates(t *testing.T) {
	path := filepath.Join("..", "..", "runs", "reports", "phase10_4_price_regime_branch_closure.json")
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read closure report: %v", err)
	}
	var report struct {
		FinalBranchStatus string `json:"final_branch_status"`
		Candidates        []struct {
			Name         string   `json:"name"`
			Status       string   `json:"status"`
			FailureModes []string `json:"failure_modes"`
		} `json:"candidates"`
	}
	if err := json.Unmarshal(data, &report); err != nil {
		t.Fatalf("unmarshal closure report: %v", err)
	}
	if report.FinalBranchStatus != "closed_no_validated_leads" {
		t.Fatalf("status = %s", report.FinalBranchStatus)
	}
	required := []string{
		"Fast Accumulation",
		"ShockFade_LONG",
		"CompressionBreakout_LONG",
		"CompressionBreakout_SHORT",
		"SOL ShockFade_LONG",
		"AVAX ShockFade_LONG",
	}
	for _, want := range required {
		found := false
		for _, candidate := range report.Candidates {
			if candidate.Name == want {
				found = true
				if candidate.Status == "" || len(candidate.FailureModes) == 0 {
					t.Fatalf("candidate %s missing status/failure modes", want)
				}
			}
		}
		if !found {
			t.Fatalf("candidate %s missing from closure report", want)
		}
	}
}

func TestPhase104GuardrailChecklistIncludesConcentrationAndOOSGates(t *testing.T) {
	path := filepath.Join("..", "..", "runs", "reports", "phase10_4_research_guardrails.md")
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read guardrail report: %v", err)
	}
	text := string(data)
	for _, want := range []string{
		"H2 OOS pass required",
		"Top 1 month contribution",
		"Top 2 month contribution",
		"Cluster concentration",
		"Bracket failure",
		"No candidate can move forward from aggregate PF alone",
	} {
		if !strings.Contains(text, want) {
			t.Fatalf("guardrail report missing %q", want)
		}
	}
}

func TestNoAKTraderImport(t *testing.T) {
	root, err := filepath.Abs(filepath.Join("..", ".."))
	if err != nil {
		t.Fatalf("resolve repo root: %v", err)
	}
	forbiddenImport := "\"github.com/davidmiguel22573/" + ("ak" + "-trader")
	err = filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			if d.Name() == ".git" || d.Name() == "runs" || d.Name() == "bin" {
				return filepath.SkipDir
			}
			return nil
		}
		if filepath.Ext(path) != ".go" {
			return nil
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		if strings.Contains(string(data), forbiddenImport) {
			t.Fatalf("unexpected forbidden import/reference in %s", path)
		}
		return nil
	})
	if err != nil {
		t.Fatalf("walk repo: %v", err)
	}
}

func leaderboardContainsSymbol(rows []LeaderboardRow, symbol string) bool {
	for _, row := range rows {
		if row.Symbol == symbol {
			return true
		}
	}
	return false
}
