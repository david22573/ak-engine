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
