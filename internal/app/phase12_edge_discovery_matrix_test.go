package app

import "testing"

func TestPhase120LabelRobustCandidate(t *testing.T) {
	row := phase120BucketSummary{
		SampleCount:              2000,
		ClusterCount:             1200,
		ExpectancyAfter5Bps:      8,
		ExpectancyAfter7_5Bps:    5.5,
		ExpectancyAfter10Bps:     3,
		ProfitFactor:             1.12,
		LeaveOneSymbolOut:        phase120LeaveOneOut{Passed: true},
		LeaveOneMonthOut:         phase120LeaveOneOut{Passed: true},
		LeaveOneQuarterOut:       phase120LeaveOneOut{Passed: true},
		ContributingSymbolCount:  6,
		ContributingMonthCount:   16,
		TopSymbolContributionPct: 30,
		TopMonthContributionPct:  20,
	}
	label, failed := phase120Label(row)
	if label != phase120LabelRobust {
		t.Fatalf("label=%s failed=%v, want %s", label, failed, phase120LabelRobust)
	}
}

func TestPhase120StrongestBucketPrefersActionableLabel(t *testing.T) {
	rows := []phase120BucketSummary{
		{Feature: "tiny", Label: phase120LabelInsufficient, ExpectancyAfter5Bps: 100, SampleCount: 10},
		{Feature: "robust", Label: phase120LabelRobust, ExpectancyAfter5Bps: 8, SampleCount: 2000},
		{Feature: "weak", Label: phase120LabelWeak, ExpectancyAfter5Bps: 20, SampleCount: 2000},
	}
	got, ok := phase120StrongestBucket(rows)
	if !ok {
		t.Fatalf("strongest bucket not found")
	}
	if got.Feature != "robust" {
		t.Fatalf("strongest feature=%s, want robust", got.Feature)
	}
}

func TestPhase120PostCostDistribution(t *testing.T) {
	agg := phase120Agg{
		key:      phase120AggKey{Feature: "F", Bucket: "B", Regime: "R", Side: "long", Horizon: "60m"},
		bins:     make(map[int]int),
		clusters: make(map[string]*phase120ClusterCounter),
		symbols:  make(map[string]*phase120Stats),
		months:   make(map[string]*phase120Stats),
		quarters: make(map[string]*phase120Stats),
	}
	
	// Add trades. Costs are 5, 7.5, 10 bps.
	// Trade 1: raw return = 6 bps
	// Cost5 = +1, Cost7.5 = -1.5, Cost10 = -4
	agg.add(6, 1000, "BTC", "2024-01", "2024-Q1")
	// Trade 2: raw return = 12 bps
	// Cost5 = +7, Cost7.5 = +4.5, Cost10 = +2
	agg.add(12, 2000, "BTC", "2024-01", "2024-Q1")
	// Trade 3: raw return = 8 bps
	// Cost5 = +3, Cost7.5 = +0.5, Cost10 = -2
	agg.add(8, 3000, "BTC", "2024-01", "2024-Q1")

	aggs := map[phase120AggKey]*phase120Agg{agg.key: &agg}
	summaries := phase120Summaries(aggs)
	if len(summaries) != 1 {
		t.Fatalf("expected 1 summary, got %d", len(summaries))
	}
	s := summaries[0]

	// Check post-cost metrics
	// 5 bps cost: 3 wins (+1, +7, +3), 0 losses
	// Gross profit: 11, Gross loss: 0
	if s.WinCountAfter5Bps != 3 {
		t.Errorf("WinCountAfter5Bps = %d, want 3", s.WinCountAfter5Bps)
	}
	if s.LossCountAfter5Bps != 0 {
		t.Errorf("LossCountAfter5Bps = %d, want 0", s.LossCountAfter5Bps)
	}
	if s.GrossProfitAfter5Bps != 11 {
		t.Errorf("GrossProfitAfter5Bps = %f, want 11", s.GrossProfitAfter5Bps)
	}
	if s.PFAfter5Bps != 999 {
		t.Errorf("PFAfter5Bps = %f, want 999", s.PFAfter5Bps)
	}

	// 7.5 bps cost: 2 wins (+4.5, +0.5), 1 loss (-1.5)
	// Gross profit: 5.0, Gross loss: 1.5 -> PF: 5.0 / 1.5 = 3.333333
	if s.WinCountAfter7_5Bps != 2 {
		t.Errorf("WinCountAfter7_5Bps = %d, want 2", s.WinCountAfter7_5Bps)
	}
	if s.LossCountAfter7_5Bps != 1 {
		t.Errorf("LossCountAfter7_5Bps = %d, want 1", s.LossCountAfter7_5Bps)
	}
	if s.GrossProfitAfter7_5Bps != 5.0 {
		t.Errorf("GrossProfitAfter7_5Bps = %f, want 5.0", s.GrossProfitAfter7_5Bps)
	}
	if s.GrossLossAfter7_5Bps != 1.5 {
		t.Errorf("GrossLossAfter7_5Bps = %f, want 1.5", s.GrossLossAfter7_5Bps)
	}
	expectedPF75 := roundMetric(5.0 / 1.5)
	if s.PFAfter7_5Bps != expectedPF75 {
		t.Errorf("PFAfter7_5Bps = %f, want %f", s.PFAfter7_5Bps, expectedPF75)
	}

	// 10 bps cost: 1 win (+2), 2 losses (-4, -2)
	// Gross profit: 2, Gross loss: 6 -> PF: 2/6 = 0.333333
	if s.WinCountAfter10Bps != 1 {
		t.Errorf("WinCountAfter10Bps = %d, want 1", s.WinCountAfter10Bps)
	}
	if s.LossCountAfter10Bps != 2 {
		t.Errorf("LossCountAfter10Bps = %d, want 2", s.LossCountAfter10Bps)
	}
	expectedPF10 := roundMetric(2.0 / 6.0)
	if s.PFAfter10Bps != expectedPF10 {
		t.Errorf("PFAfter10Bps = %f, want %f", s.PFAfter10Bps, expectedPF10)
	}
}

func TestPhase120ProfitFactorNoWinsNoLosses(t *testing.T) {
	// If count == 0, PF should be 0
	agg := phase120Agg{
		key:      phase120AggKey{Feature: "F", Bucket: "B", Regime: "R", Side: "long", Horizon: "60m"},
		bins:     make(map[int]int),
		clusters: make(map[string]*phase120ClusterCounter),
		symbols:  make(map[string]*phase120Stats),
		months:   make(map[string]*phase120Stats),
		quarters: make(map[string]*phase120Stats),
	}
	aggs := map[phase120AggKey]*phase120Agg{agg.key: &agg}
	summaries := phase120Summaries(aggs)
	s := summaries[0]
	if s.PFAfter5Bps != 0 || s.PFAfter7_5Bps != 0 || s.PFAfter10Bps != 0 {
		t.Errorf("Expected PF 0 when no trades, got 5=%f 7.5=%f 10=%f", s.PFAfter5Bps, s.PFAfter7_5Bps, s.PFAfter10Bps)
	}
}
