package app

import (
	"strings"
	"testing"
)

const syntheticJSONL = `
{"candidate_id":"C1","symbol":"BTCUSDT","side":1,"ts":1736935200,"horizon":240,"month":"2025-01","quarter":"2025-Q1","pre":{"trend":"downtrend","vol":"mid","fund":"neutral"},"cluster":{"key":"CL-1","ts":1736930000,"spacing":7200000,"size":2,"ordinal":1},"grs_bps":20,"net_5":10,"net_75":5,"net_10":0,"win_5":true,"win_75":true,"win_10":false,"label":"TP"}
{"candidate_id":"C1","symbol":"ETHUSDT","side":-1,"ts":1736940000,"horizon":240,"month":"2025-01","quarter":"2025-Q1","pre":{"trend":"uptrend","vol":"high","fund":"positive"},"cluster":{"key":"CL-2","ts":1736940000,"spacing":3600000,"size":1,"ordinal":1},"grs_bps":-15,"net_5":-25,"net_75":-30,"net_10":-35,"win_5":false,"win_75":false,"win_10":false,"label":"SL"}
{"candidate_id":"C1","symbol":"BTCUSDT","side":1,"ts":1740000000,"horizon":240,"month":"2025-02","quarter":"2025-Q1","pre":{"trend":"downtrend","vol":"mid","fund":"neutral"},"cluster":{"key":"CL-3","ts":1740000000,"spacing":0,"size":1,"ordinal":1},"grs_bps":40,"net_5":30,"net_75":25,"net_10":20,"win_5":true,"win_75":true,"win_10":true,"label":"TP"}
`

const invalidJSONL = `
{"candidate_id":"","symbol":"BTCUSDT","side":1,"ts":1736935200,"horizon":240,"month":"2025-01","quarter":"2025-Q1","pre":{"trend":"downtrend","vol":"mid","fund":"neutral"},"cluster":{"key":"CL-1","ts":1736930000,"spacing":7200000,"size":2,"ordinal":1},"grs_bps":20,"net_5":10,"net_75":5,"net_10":0,"win_5":true,"win_75":true,"win_10":false,"label":"TP"}
`

func TestAggregator_LoadJSONL(t *testing.T) {
	agg := NewAggregator()
	err := agg.LoadJSONL(strings.NewReader(syntheticJSONL))
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if len(agg.Events) != 3 {
		t.Fatalf("expected 3 events, got %d", len(agg.Events))
	}
}

func TestAggregator_LoadInvalidJSONL(t *testing.T) {
	agg := NewAggregator()
	err := agg.LoadJSONL(strings.NewReader(invalidJSONL))
	if err == nil {
		t.Fatalf("expected error on invalid jsonl, got nil")
	}
}

func TestAggregator_FullSummary(t *testing.T) {
	agg := NewAggregator()
	_ = agg.LoadJSONL(strings.NewReader(syntheticJSONL))

	summary := agg.FullSummary()

	if summary.EventCount != 3 {
		t.Errorf("EventCount expected 3, got %d", summary.EventCount)
	}
	if summary.ClusterCount != 3 {
		t.Errorf("ClusterCount expected 3, got %d", summary.ClusterCount)
	}
	if summary.WinCount != 2 {
		t.Errorf("WinCount expected 2, got %d", summary.WinCount)
	}
	if summary.LossCount != 1 {
		t.Errorf("LossCount expected 1, got %d", summary.LossCount)
	}

	expectedNet5 := 10.0 - 25.0 + 30.0
	if summary.Net5Bps != expectedNet5 {
		t.Errorf("Net5Bps expected %f, got %f", expectedNet5, summary.Net5Bps)
	}

	expectedExpectancy5 := expectedNet5 / 3.0
	if summary.Expectancy5 != expectedExpectancy5 {
		t.Errorf("Expectancy5 expected %f, got %f", expectedExpectancy5, summary.Expectancy5)
	}

	// Profit factor at 5bps
	// wins = 10 + 30 = 40
	// losses = abs(-25) = 25
	// PF = 40 / 25 = 1.6
	if summary.ProfitFactor5 != 1.6 {
		t.Errorf("ProfitFactor5 expected 1.6, got %f", summary.ProfitFactor5)
	}

	if summary.BestMonth != "2025-02" {
		t.Errorf("BestMonth expected 2025-02, got %s", summary.BestMonth)
	}
	if summary.WorstMonth != "2025-01" {
		t.Errorf("WorstMonth expected 2025-01, got %s", summary.WorstMonth)
	}

	if summary.SymbolConcentration["BTCUSDT"] != 2 {
		t.Errorf("BTCUSDT concentration expected 2, got %d", summary.SymbolConcentration["BTCUSDT"])
	}
}

func TestAggregator_FilterSimulation(t *testing.T) {
	agg := NewAggregator()
	_ = agg.LoadJSONL(strings.NewReader(syntheticJSONL))

	// Valid filter: Only downtrend
	f1 := EventFilter{IncludeRegime: "downtrend"}
	s1, err := agg.SimulateFilter(f1)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if s1.EventCount != 2 {
		t.Errorf("expected 2 downtrend events, got %d", s1.EventCount)
	}

	// Valid filter: Exclude high vol
	f2 := EventFilter{ExcludeVolatility: "high"}
	s2, err := agg.SimulateFilter(f2)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if s2.EventCount != 2 {
		t.Errorf("expected 2 non-high vol events, got %d", s2.EventCount)
	}

	// Leaky filter rejection
	f3 := EventFilter{InvalidOutcomeFilter: "TP"}
	_, err = agg.SimulateFilter(f3)
	if err == nil {
		t.Fatalf("expected error on leaky filter, got nil")
	}
}

func TestAggregator_GroupSummaries(t *testing.T) {
	agg := NewAggregator()
	_ = agg.LoadJSONL(strings.NewReader(syntheticJSONL))

	sm := agg.SymbolMonthSummary()
	if len(sm) != 3 { // BTCUSDT_2025-01, ETHUSDT_2025-01, BTCUSDT_2025-02
		t.Errorf("SymbolMonthSummary expected 3 groups, got %d", len(sm))
	}

	sq := agg.SymbolQuarterSummary()
	if len(sq) != 2 { // BTCUSDT_2025-Q1, ETHUSDT_2025-Q1
		t.Errorf("SymbolQuarterSummary expected 2 groups, got %d", len(sq))
	}

	cq := agg.CandidateQuarterSummary()
	if len(cq) != 1 { // C1_2025-Q1
		t.Errorf("CandidateQuarterSummary expected 1 group, got %d", len(cq))
	}

	fsh := agg.FamilySideHorizonSummary()
	if len(fsh) != 2 { // BTCUSDT_long_240, ETHUSDT_short_240
		t.Errorf("FamilySideHorizonSummary expected 2 groups, got %d", len(fsh))
	}
}

func TestAggregator_LeaveOneOut(t *testing.T) {
	agg := NewAggregator()
	_ = agg.LoadJSONL(strings.NewReader(syntheticJSONL))

	lomo := agg.LeaveOneMonthOutSummary()
	if len(lomo) != 2 {
		t.Errorf("expected 2 leave-one-month-out scenarios, got %d", len(lomo))
	}
	// Without 2025-01, we only have 2025-02 which has 1 event
	if lomo["2025-01"].EventCount != 1 {
		t.Errorf("expected 1 event when leaving out 2025-01, got %d", lomo["2025-01"].EventCount)
	}

	loqo := agg.LeaveOneQuarterOutSummary()
	if len(loqo) != 1 {
		t.Errorf("expected 1 leave-one-quarter-out scenario, got %d", len(loqo))
	}
	if loqo["2025-Q1"].EventCount != 0 {
		t.Errorf("expected 0 events when leaving out 2025-Q1, got %d", loqo["2025-Q1"].EventCount)
	}
}
