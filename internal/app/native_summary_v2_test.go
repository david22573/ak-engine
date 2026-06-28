package app

import (
	"github.com/davidmiguel22573/ak-engine/internal/features"
	"testing"
)

func TestNativeSummaryV2Math(t *testing.T) {
	// 1. Synthetic Events
	events := []FundingEventRow{
		{
			Family:          "NegativeFundingLong",
			Side:            "LONG",
			Symbol:          "FAKEUSDT",
			EventTimeMS:     1704067200000, // 2024-01-01 00:00:00 UTC
			EntryPrice:      100.0,
			FundingBucket:   "bkt_0",
			RegimeComposite: "bull",
		},
		{
			Family:          "NegativeFundingLong",
			Side:            "LONG",
			Symbol:          "FAKEUSDT",
			EventTimeMS:     1704070800000, // 2024-01-01 01:00:00 UTC
			EntryPrice:      105.0,
			FundingBucket:   "bkt_0",
			RegimeComposite: "bull",
		},
	}

	rows := []ResearchFeatureRow{
		{Row: features.Row{EventTimeMS: 1704067200000, Close: 100.0}},                        // event 1
		{Row: features.Row{EventTimeMS: 1704067200000 + 5*60*1000, Close: 102.0}},            // delay 0, +5m target (2% profit) -> 200 bps
		{Row: features.Row{EventTimeMS: 1704067200000 + 300000*2, Close: 100.0}},             // delay 1
		{Row: features.Row{EventTimeMS: 1704067200000 + 300000*2 + 5*60*1000, Close: 101.0}}, // delay 1 target (1% profit) -> 100 bps
		{Row: features.Row{EventTimeMS: 1704070800000, Close: 105.0}},                        // event 2
		{Row: features.Row{EventTimeMS: 1704070800000 + 5*60*1000, Close: 102.9}},            // target for event 2 (-2% loss) -> -200 bps
	}

	result := computeNativeSummaryV2(events, rows, "dummy_hash")

	// Verify cost stress math (5, 7.5, 10, 15)
	// Verify delay stress math (0, 1, 2)
	// Verify PF math, event count, declustered count
	var found5mDelay0Cost5 bool
	for _, r := range result {
		if r.HorizonMinutes == 5 && r.DelayCandles == 0 && r.CostBps == 5.0 {
			found5mDelay0Cost5 = true
			if r.EventCount != 2 {
				t.Errorf("expected 2 events, got %d", r.EventCount)
			}
			if r.DeClusteredEventCount != 1 {
				t.Errorf("expected 1 declustered event, got %d", r.DeClusteredEventCount)
			}
			// event 1 net = 200 bps - 5 bps = +195
			// event 2 net = -200 bps - 5 bps = -205
			// gross profit = 195
			// gross loss = 205
			// net bps = -10
			// PF = 195 / 205
			expectedPF := roundMetric(195.0 / 205.0)
			if r.ProfitFactor != expectedPF {
				t.Errorf("expected PF %f, got %f", expectedPF, r.ProfitFactor)
			}
			if r.NetBps != -10.0 {
				t.Errorf("expected NetBps -10.0, got %f", r.NetBps)
			}
			if r.GrossProfitBps != 195.0 {
				t.Errorf("expected GrossProfit 195.0, got %f", r.GrossProfitBps)
			}
			if r.GrossLossBps != 205.0 {
				t.Errorf("expected GrossLoss 205.0, got %f", r.GrossLossBps)
			}
			if r.FundingBucket != "bkt_0" || r.RegimeBucket != "bull" {
				t.Errorf("expected correct buckets")
			}
			if r.FundingXRegimeBucket != "bkt_0|bull" {
				t.Errorf("expected correct crossed bucket")
			}
			if r.WinCount != 1 || r.LossCount != 1 {
				t.Errorf("expected 1 win 1 loss")
			}
			expectedExpectancy := -10.0 / 2.0
			if r.ExpectancyBps != expectedExpectancy {
				t.Errorf("expected expectancy %f, got %f", expectedExpectancy, r.ExpectancyBps)
			}
		}
	}
	if !found5mDelay0Cost5 {
		t.Errorf("did not find 5m horizon, 0 delay, 5.0 cost bps bucket")
	}

	// 2. Audit Behavior
	loaded := []fundingLoadedEventFile{
		{
			V2Missing: false,
			V2Summary: result,
		},
	}
	audit := &FundingEventIntegrityAudit{Checks: []FundingIntegrityCheck{}}
	verifyNativeSummaryV2(loaded, audit)
	if audit.Status == "FAIL" {
		t.Errorf("expected PASS, got FAIL. Failures: %v", audit.Failures)
	}
	if got := fundingIntegrityCheckStatus(audit, "v2_aggregate_totals_match"); got != "PASS" {
		t.Errorf("expected aggregate totals check PASS, got %s", got)
	}

	// Now test missing V2
	auditFail := &FundingEventIntegrityAudit{Checks: []FundingIntegrityCheck{}}
	verifyNativeSummaryV2([]fundingLoadedEventFile{{V2Missing: true}}, auditFail)
	if auditFail.Status != "FAIL" {
		t.Errorf("expected FAIL for missing V2")
	}
}

func TestNativeSummaryV2IntegrityPassFailUnknown(t *testing.T) {
	good := NativeSummaryV2Row{
		Symbol:                "LINKUSDT",
		Month:                 "2025-01",
		Quarter:               "2025Q1",
		Year:                  "2025",
		CostBps:               5,
		DelayCandles:          1,
		EventCount:            2,
		DeClusteredEventCount: 1,
		GrossProfitBps:        10,
		GrossLossBps:          -5,
		WinCount:              1,
		LossCount:             1,
		InputHash:             "input",
		SummaryHash:           "summary",
	}

	passAudit := &FundingEventIntegrityAudit{}
	verifyNativeSummaryV2([]fundingLoadedEventFile{{V2Summary: []NativeSummaryV2Row{good}}}, passAudit)
	if got := fundingIntegrityCheckStatus(passAudit, "v2_aggregate_totals_match"); got != "PASS" {
		t.Fatalf("aggregate totals check = %s, want PASS", got)
	}

	bad := good
	bad.WinCount = 2
	failAudit := &FundingEventIntegrityAudit{}
	verifyNativeSummaryV2([]fundingLoadedEventFile{{V2Summary: []NativeSummaryV2Row{bad}}}, failAudit)
	if got := fundingIntegrityCheckStatus(failAudit, "v2_aggregate_totals_match"); got != "FAIL" {
		t.Fatalf("aggregate totals check = %s, want FAIL", got)
	}

	unknownAudit := &FundingEventIntegrityAudit{}
	verifyNativeSummaryV2(nil, unknownAudit)
	if got := fundingIntegrityCheckStatus(unknownAudit, "v2_aggregate_totals_match"); got != "UNKNOWN" {
		t.Fatalf("aggregate totals check = %s, want UNKNOWN", got)
	}
}

func fundingIntegrityCheckStatus(audit *FundingEventIntegrityAudit, name string) string {
	for _, check := range audit.Checks {
		if check.Name == name {
			return check.Status
		}
	}
	return ""
}
