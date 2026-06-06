package walkforward

import (
	"context"
	"math"
	"testing"
	"time"

	"github.com/davidmiguel22573/ak-engine/internal/data"
)

func TestGenerateSplits(t *testing.T) {
	fromMs := int64(1000)
	toMs := int64(10000)

	trainWindow := 2 * time.Second
	testWindow := 1 * time.Second

	splits := GenerateSplits(fromMs, toMs, trainWindow, testWindow)

	if len(splits) == 0 {
		t.Fatal("expected splits")
	}

	for _, s := range splits {
		if s.TrainEndMs > s.TestStartMs {
			t.Errorf("train and test overlap: train end %d, test start %d", s.TrainEndMs, s.TestStartMs)
		}
		if s.TestEndMs > toMs {
			t.Errorf("incomplete final split was not skipped")
		}
	}
}

func TestValidation(t *testing.T) {
	cfg := Config{
		MinTrades:       5,
		MaxLossStreak:   3,
		MinProfitFactor: 1.10,
	}

	res := &WalkForwardResult{
		AggregateTest: AggregateMetrics{
			NetPnL:               100,
			ProfitFactor:         1.2,
			ProfitableSplitCount: 3,
			LosingSplitCount:     1,
			MaxConsecutiveLosses: 2,
			TotalTrades:          10,
		},
	}

	Validate(res, cfg)
	if !res.PromotionCandidate {
		t.Errorf("expected true, got false")
	}

	// losing but valid walk-forward returns status PASS with promotion_candidate false
	res.AggregateTest.NetPnL = -100
	Validate(res, cfg)
	if res.PromotionCandidate {
		t.Errorf("expected false for negative pnl")
	}
}

func TestInvalidConfig(t *testing.T) {
	cfg := Config{
		TrainWindow: 0,
		TestWindow:  0,
	}
	_, err := Run(context.Background(), cfg, nil, data.CandleRequest{}, nil)
	if err == nil {
		t.Errorf("expected error for invalid config")
	}
}

func TestComputeAggregateAverageHoldMinutes(t *testing.T) {
	splits := []SplitResult{
		{
			Status: "PASS",
			SelectedCandidates: []CandidateResult{
				{},
			},
			BestTrainCandidate: CandidateResult{
				TotalTrades:        2,
				AverageHoldMinutes: 15,
			},
			BestTestResult: CandidateResult{
				TotalTrades:        3,
				AverageHoldMinutes: 10,
				NetPnL:             1,
			},
			CorrespondingTestResult: CandidateResult{
				TotalTrades:        3,
				AverageHoldMinutes: 10,
				NetPnL:             1,
			},
		},
		{
			Status: "PASS",
			SelectedCandidates: []CandidateResult{
				{},
			},
			BestTrainCandidate: CandidateResult{
				TotalTrades:        1,
				AverageHoldMinutes: 45,
			},
			BestTestResult: CandidateResult{
				TotalTrades:        1,
				AverageHoldMinutes: 30,
				NetPnL:             -1,
			},
			CorrespondingTestResult: CandidateResult{
				TotalTrades:        1,
				AverageHoldMinutes: 30,
				NetPnL:             -1,
			},
		},
	}

	trainAgg := computeAggregate(splits, true)
	if math.Abs(trainAgg.AverageHoldMinutes-25) > 1e-9 {
		t.Fatalf("expected weighted train average hold 25, got %v", trainAgg.AverageHoldMinutes)
	}

	testAgg := computeAggregate(splits, false)
	if math.Abs(testAgg.AverageHoldMinutes-15) > 1e-9 {
		t.Fatalf("expected weighted test average hold 15, got %v", testAgg.AverageHoldMinutes)
	}
}

func TestComputeAggregateIncludesDiagnostics(t *testing.T) {
	splits := []SplitResult{
		{
			Status:             "PASS",
			SelectedCandidates: []CandidateResult{{}},
			BestTrainCandidate: CandidateResult{
				Diagnostics: DiagnosticSummary{
					PnLByAction:        map[string]float64{"FULL_LONG": 10},
					PnLByReasonCode:    map[string]float64{"TIME_STOP": -2},
					HardBlocksByReason: map[string]int{"LOW_TREND_SCORE": 3},
				},
			},
			CorrespondingTestResult: CandidateResult{
				NetPnL: 1,
				Diagnostics: DiagnosticSummary{
					FeesByAction:       map[string]float64{"FULL_LONG": 0.6},
					SlippageByAction:   map[string]float64{"FULL_LONG": 0.1},
					LongVsShortMetrics: map[string]float64{"long_pnl": 1},
				},
			},
		},
	}

	trainAgg := computeAggregate(splits, true)
	if got := trainAgg.Diagnostics.PnLByAction["FULL_LONG"]; got != 10 {
		t.Fatalf("train pnl_by_action[FULL_LONG] = %v, want 10", got)
	}
	if got := trainAgg.Diagnostics.PnLByReasonCode["TIME_STOP"]; got != -2 {
		t.Fatalf("train pnl_by_reason_code[TIME_STOP] = %v, want -2", got)
	}
	if got := trainAgg.Diagnostics.HardBlocksByReason["LOW_TREND_SCORE"]; got != 3 {
		t.Fatalf("train hard_blocks_by_reason[LOW_TREND_SCORE] = %v, want 3", got)
	}

	testAgg := computeAggregate(splits, false)
	if got := testAgg.Diagnostics.FeesByAction["FULL_LONG"]; got != 0.6 {
		t.Fatalf("test fees_by_action[FULL_LONG] = %v, want 0.6", got)
	}
	if got := testAgg.Diagnostics.LongVsShortMetrics["long_pnl"]; got != 1 {
		t.Fatalf("test long_vs_short_metrics[long_pnl] = %v, want 1", got)
	}
}

func TestCandidateStabilityCountsRepeatedSignatures(t *testing.T) {
	params := Params{
		FullTradeMinScore:                 85,
		NormalTradeMinScore:               70,
		ProbeMinScore:                     55,
		CostMultipleRequired:              4,
		MaxHoldWindows:                    4,
		TimeStopWindows:                   2,
		AllowProbeTrade:                   false,
		DisableProbeTrades:                true,
		MaxChopScore:                      50,
		MinTrendScore:                     60,
		DisableScoreBucket55To69:          true,
		RequireScoreBucket70Plus:          true,
		RequireExpectedMoveGtCostMultiple: true,
		LongEnabled:                       true,
		ShortEnabled:                      false,
	}

	stability := buildCandidateStability([]SplitResult{
		{
			Status: "PASS",
			SelectedCandidates: []CandidateResult{
				{Params: params, NetPnL: 5, ProfitFactor: 1.4},
			},
			TestResults: []CandidateResult{
				{NetPnL: -2, ProfitFactor: 0.8, TotalTrades: 2, MaxDrawdown: 4, MaxConsecutiveLosses: 2},
			},
		},
		{
			Status: "PASS",
			SelectedCandidates: []CandidateResult{
				{Params: params, NetPnL: 3, ProfitFactor: 1.2},
			},
			TestResults: []CandidateResult{
				{NetPnL: 1, ProfitFactor: 1.1, TotalTrades: 1, MaxDrawdown: 2, MaxConsecutiveLosses: 1},
			},
		},
	})

	if len(stability) != 1 {
		t.Fatalf("len(stability) = %d, want 1", len(stability))
	}
	if got := stability[0].SelectionCount; got != 2 {
		t.Fatalf("selection_count = %d, want 2", got)
	}
	if got := stability[0].TestLosingCount; got != 1 {
		t.Fatalf("test_losing_count = %d, want 1", got)
	}
	if got := stability[0].TestProfitableCount; got != 1 {
		t.Fatalf("test_profitable_count = %d, want 1", got)
	}
	if got := stability[0].MaxTestLossStreak; got != 2 {
		t.Fatalf("max_test_loss_streak = %d, want 2", got)
	}
}
