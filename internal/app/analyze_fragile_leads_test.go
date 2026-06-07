package app

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestPhase103BTopMonthContributionCalculatesTop1Top2Top3(t *testing.T) {
	nets := []float64{60, 25, 15}
	if got := phase103BTopContributionPct(nets, 1); got != 60 {
		t.Fatalf("top1 = %.4f, want 60", got)
	}
	if got := phase103BTopContributionPct(nets, 2); got != 85 {
		t.Fatalf("top2 = %.4f, want 85", got)
	}
	if got := phase103BTopContributionPct(nets, 3); got != 100 {
		t.Fatalf("top3 = %.4f, want 100", got)
	}
}

func TestPhase103BLeaveOneMonthOutRecomputesMetrics(t *testing.T) {
	events := []phase103BReturnEvent{
		testPhase103BReturnEvent("2023-01-05", 10),
		testPhase103BReturnEvent("2023-02-05", 20),
		testPhase103BReturnEvent("2023-03-05", -5),
	}
	rows := phase103BLeaveOneMonthOut(events, 2023)
	row := findPhase103BLOMO(t, rows, "2023-02")
	if row.RemainingEventCount != 2 {
		t.Fatalf("remaining count = %d, want 2", row.RemainingEventCount)
	}
	if row.RemainingExpectancyBps != 2.5 {
		t.Fatalf("expectancy = %.4f, want 2.5", row.RemainingExpectancyBps)
	}
	if row.RemainingPF5bps != 2 {
		t.Fatalf("PF = %.4f, want 2", row.RemainingPF5bps)
	}
	if row.VerdictAfterRemoval != "edge_positive" {
		t.Fatalf("verdict = %s, want edge_positive", row.VerdictAfterRemoval)
	}
}

func TestPhase103BLeaveOneQuarterOutRecomputesMetrics(t *testing.T) {
	events := []phase103BReturnEvent{
		testPhase103BReturnEvent("2023-01-05", 10),
		testPhase103BReturnEvent("2023-04-05", -5),
		testPhase103BReturnEvent("2023-07-05", 20),
		testPhase103BReturnEvent("2023-10-05", 5),
	}
	rows := phase103BLeaveOneQuarterOut(events)
	row := findPhase103BLOQO(t, rows, "Q3")
	if row.RemainingEventCount != 3 {
		t.Fatalf("remaining count = %d, want 3", row.RemainingEventCount)
	}
	if row.RemainingPF5bps != 3 {
		t.Fatalf("PF = %.4f, want 3", row.RemainingPF5bps)
	}
	if row.RemainingExpectancyBps != 10.0/3.0 {
		t.Fatalf("expectancy = %.8f, want %.8f", row.RemainingExpectancyBps, 10.0/3.0)
	}
}

func TestPhase103BClusterGroupingWithin60Minutes(t *testing.T) {
	base := time.Date(2023, time.January, 1, 0, 0, 0, 0, time.UTC).UnixMilli()
	events := []phase103BReturnEvent{
		{EventTimeMS: base, ReturnBps: 10},
		{EventTimeMS: base + 60*60*1000, ReturnBps: 20},
		{EventTimeMS: base + 121*60*1000, ReturnBps: 30},
	}
	clusters := phase103BClusterEvents(events)
	if len(clusters) != 2 {
		t.Fatalf("cluster count = %d, want 2", len(clusters))
	}
	if clusters[0].EventCount != 2 {
		t.Fatalf("first cluster event count = %d, want 2", clusters[0].EventCount)
	}
}

func TestPhase103BEventArtifactVerdictWhenOneMonthDominates(t *testing.T) {
	lead := phase103BPassingLead()
	lead.Top1MonthContributionPct = 60
	lead.Top2MonthContributionPct = 65
	status, reasons := phase103BFinalStatus(lead)
	if status != phase103BStatusFragileEventArtifact {
		t.Fatalf("status = %s, want %s", status, phase103BStatusFragileEventArtifact)
	}
	if !strings.Contains(strings.Join(reasons, ";"), "top 1 month") {
		t.Fatalf("expected top 1 month reason, got %v", reasons)
	}
}

func TestPhase103BExtendedOOSMissingDataReported(t *testing.T) {
	dir := t.TempDir()
	report, err := buildPhase103BOOSCoverage(context.Background(), dir, "futures-um", "1m")
	if err != nil {
		t.Fatalf("coverage: %v", err)
	}
	if !report.Summary.BlockedByMissingData {
		t.Fatalf("expected blocked by missing data")
	}
	if len(report.Summary.FetchCommands) == 0 {
		t.Fatalf("expected fetch commands")
	}
	for _, row := range report.Rows {
		if row.HasFullCoverage {
			t.Fatalf("unexpected full coverage for %s %d", row.Symbol, row.Year)
		}
		if row.FetchCommand == "" {
			t.Fatalf("missing fetch command for %s %d", row.Symbol, row.Year)
		}
	}
}

func TestPhase103BFinalStatusAllowlistExcludesRuntimeTerms(t *testing.T) {
	for _, status := range []string{"promoted", "runtime_ready", "testnet_ready", "shadow_ready"} {
		if phase103BAllowedStatuses[status] {
			t.Fatalf("forbidden status %q is allowed", status)
		}
	}
	for _, status := range []string{
		phase103BStatusRejected,
		phase103BStatusFragileEventArtifact,
		phase103BStatusFragileNeedsOOS,
		phase103BStatusValidatedResearchLead,
	} {
		if !phase103BAllowedStatuses[status] {
			t.Fatalf("allowed status %q missing", status)
		}
	}
}

func TestPhase103BNoAKTraderImport(t *testing.T) {
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
			switch d.Name() {
			case ".git", "runs", "bin":
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

func testPhase103BReturnEvent(day string, ret float64) phase103BReturnEvent {
	t, err := time.Parse("2006-01-02", day)
	if err != nil {
		panic(err)
	}
	return phase103BReturnEvent{
		EventTimeMS: t.UnixMilli(),
		ReturnBps:   ret,
		Month:       t.Format("2006-01"),
		Quarter:     phase103BQuarter(t),
	}
}

func findPhase103BLOMO(t *testing.T, rows []Phase103BLeaveOneOutResult, month string) Phase103BLeaveOneOutResult {
	t.Helper()
	for _, row := range rows {
		if row.MonthRemoved == month {
			return row
		}
	}
	t.Fatalf("missing LOMO row %s", month)
	return Phase103BLeaveOneOutResult{}
}

func findPhase103BLOQO(t *testing.T, rows []Phase103BLeaveOneOutResult, quarter string) Phase103BLeaveOneOutResult {
	t.Helper()
	for _, row := range rows {
		if row.QuarterRemoved == quarter {
			return row
		}
	}
	t.Fatalf("missing LOQO row %s", quarter)
	return Phase103BLeaveOneOutResult{}
}

func phase103BPassingLead() Phase103BLeadAnalysis {
	positiveLOO := []Phase103BLeaveOneOutResult{
		{RemainingEventCount: 10, RemainingExpectancyBps: 1},
	}
	return Phase103BLeadAnalysis{
		EventCount:               300,
		LeakageStatus:            "PASS",
		FYPF5bps:                 1.05,
		H2PF5bps:                 1.10,
		WorstQuarterPF5bps:       0.95,
		FYExpectancy5bpsBps:      1,
		H2Expectancy5bpsBps:      1,
		Top1MonthContributionPct: 50,
		Top2MonthContributionPct: 70,
		LeaveOneMonthOut:         positiveLOO,
		LeaveOneQuarterOut:       positiveLOO,
		ExtendedOOSStatus:        "full_2024_available",
	}
}
