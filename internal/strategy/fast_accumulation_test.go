package strategy

import (
	"context"
	"reflect"
	"testing"
	"time"

	"github.com/davidmiguel22573/ak-engine/pkg/protocol"
)

func TestFastAccumulationEveryCompleted15mWindowProducesDecision(t *testing.T) {
	t.Parallel()

	cfg := DefaultFastAccumulationConfig()
	cfg.EstimatedCostBPS = 1
	cfg.CostMultipleRequired = 1
	strat, err := NewFastAccumulation(cfg)
	if err != nil {
		t.Fatalf("NewFastAccumulation() error = %v", err)
	}

	decisions := collectDecisions(t, strat, sample5mTrendCandles()[:6], State{})
	if got := len(decisions); got != 2 {
		t.Fatalf("len(decisions) = %d, want 2", got)
	}
}

func TestFastAccumulationSkipsIncomplete15mWindow(t *testing.T) {
	t.Parallel()

	cfg := DefaultFastAccumulationConfig()
	cfg.EstimatedCostBPS = 1
	cfg.CostMultipleRequired = 1
	strat, err := NewFastAccumulation(cfg)
	if err != nil {
		t.Fatalf("NewFastAccumulation() error = %v", err)
	}

	decisions := collectDecisions(t, strat, sample5mTrendCandles()[:5], State{})
	if got := len(decisions); got != 1 {
		t.Fatalf("len(decisions) = %d, want 1", got)
	}
}

func TestFastAccumulationDecisionTimestampsMatchWindowBoundaries(t *testing.T) {
	t.Parallel()

	cfg := DefaultFastAccumulationConfig()
	cfg.EstimatedCostBPS = 1
	cfg.CostMultipleRequired = 1
	strat, err := NewFastAccumulation(cfg)
	if err != nil {
		t.Fatalf("NewFastAccumulation() error = %v", err)
	}

	decisions := collectDecisions(t, strat, sample5mTrendCandles()[:3], State{})
	if got := len(decisions); got != 1 {
		t.Fatalf("len(decisions) = %d, want 1", got)
	}
	if decisions[0].WindowStartMS != sample5mTrendCandles()[0].OpenTimeMS {
		t.Fatalf("WindowStartMS = %d, want %d", decisions[0].WindowStartMS, sample5mTrendCandles()[0].OpenTimeMS)
	}
	wantEnd := sample5mTrendCandles()[0].OpenTimeMS + 15*60*1000 - 1
	if decisions[0].WindowEndMS != wantEnd {
		t.Fatalf("WindowEndMS = %d, want %d", decisions[0].WindowEndMS, wantEnd)
	}
}

func TestFastAccumulationDecisionActionSupported(t *testing.T) {
	t.Parallel()

	cfg := DefaultFastAccumulationConfig()
	cfg.EstimatedCostBPS = 1
	cfg.CostMultipleRequired = 1
	strat, err := NewFastAccumulation(cfg)
	if err != nil {
		t.Fatalf("NewFastAccumulation() error = %v", err)
	}

	decisions := collectDecisions(t, strat, sample5mTrendCandles(), State{})
	valid := map[WindowAction]bool{
		ActionFullLong:         true,
		ActionFullShort:        true,
		ActionProbeLong:        true,
		ActionProbeShort:       true,
		ActionHold:             true,
		ActionExit:             true,
		ActionReverse:          true,
		ActionNoTradeHardBlock: true,
	}
	for _, decision := range decisions {
		if !valid[decision.Action] {
			t.Fatalf("unsupported action %q", decision.Action)
		}
	}
}

func TestScoreWindowBullishFavorsLong(t *testing.T) {
	t.Parallel()

	result := ScoreWindow(ScoreInput{
		Window:               bullishWindow(101, 103.5, 100.8, 103.2, 15*time.Minute),
		Previous:             []AggregatedWindow{bullishWindow(100, 101.2, 99.7, 101.1, 0)},
		RecentCandles:        sample5mTrendCandles()[:6],
		EstimatedCostBPS:     1,
		CostMultipleRequired: 1,
		MaxChopScore:         95,
	})
	if result.LongScore <= result.ShortScore {
		t.Fatalf("LongScore = %f, ShortScore = %f, want long > short", result.LongScore, result.ShortScore)
	}
}

func TestScoreWindowBearishFavorsShort(t *testing.T) {
	t.Parallel()

	result := ScoreWindow(ScoreInput{
		Window:               bearishWindow(103, 103.2, 100.8, 101.0, 15*time.Minute),
		Previous:             []AggregatedWindow{bullishWindow(104, 104.2, 102.9, 103.2, 0)},
		RecentCandles:        sample5mTrendCandles()[3:9],
		EstimatedCostBPS:     1,
		CostMultipleRequired: 1,
		MaxChopScore:         95,
	})
	if result.ShortScore <= result.LongScore {
		t.Fatalf("ShortScore = %f, LongScore = %f, want short > long", result.ShortScore, result.LongScore)
	}
}

func TestScoreWindowChopRaisesChopScore(t *testing.T) {
	t.Parallel()

	result := ScoreWindow(ScoreInput{
		Window:               bullishWindow(100.0, 100.8, 99.9, 100.1, 15*time.Minute),
		Previous:             []AggregatedWindow{bullishWindow(100.0, 100.7, 99.8, 100.05, 0)},
		EstimatedCostBPS:     1,
		CostMultipleRequired: 1,
		MaxChopScore:         95,
	})
	if result.ChopScore < 60 {
		t.Fatalf("ChopScore = %f, want >= 60", result.ChopScore)
	}
}

func TestScoreWindowExpectedMoveBelowCostHardBlocks(t *testing.T) {
	t.Parallel()

	result := ScoreWindow(ScoreInput{
		Window:               bullishWindow(100, 100.05, 99.98, 100.01, 15*time.Minute),
		Previous:             []AggregatedWindow{bullishWindow(100, 100.04, 99.99, 100.0, 0)},
		EstimatedCostBPS:     10,
		CostMultipleRequired: 3,
		MaxChopScore:         95,
	})
	if !result.HardBlock {
		t.Fatal("HardBlock = false, want true")
	}
}

func TestFastAccumulationProbeActionInProbeRange(t *testing.T) {
	t.Parallel()

	cfg := DefaultFastAccumulationConfig()
	cfg.EstimatedCostBPS = 1
	cfg.CostMultipleRequired = 1
	cfg.NormalTradeMinScore = 80
	cfg.ProbeMinScore = 45
	cfg.FullTradeMinScore = 95
	strat, err := NewFastAccumulation(cfg)
	if err != nil {
		t.Fatalf("NewFastAccumulation() error = %v", err)
	}

	decisions := collectDecisions(t, strat, sample5mTrendCandles()[:6], State{})
	last := decisions[len(decisions)-1]
	if last.Action != ActionProbeLong && last.Action != ActionProbeShort {
		t.Fatalf("Action = %q, want probe action", last.Action)
	}
}

func TestFastAccumulationHighScoreCreatesFullAction(t *testing.T) {
	t.Parallel()

	cfg := DefaultFastAccumulationConfig()
	cfg.EstimatedCostBPS = 1
	cfg.CostMultipleRequired = 1
	cfg.FullTradeMinScore = 60
	cfg.NormalTradeMinScore = 55
	cfg.ProbeMinScore = 45
	strat, err := NewFastAccumulation(cfg)
	if err != nil {
		t.Fatalf("NewFastAccumulation() error = %v", err)
	}

	decisions := collectDecisions(t, strat, sample5mTrendCandles()[:6], State{})
	last := decisions[len(decisions)-1]
	if last.Action != ActionFullLong && last.Action != ActionFullShort {
		t.Fatalf("Action = %q, want full action", last.Action)
	}
}

func TestFastAccumulationNoLookahead(t *testing.T) {
	t.Parallel()

	cfg := DefaultFastAccumulationConfig()
	cfg.EstimatedCostBPS = 1
	cfg.CostMultipleRequired = 1

	firstRun, err := NewFastAccumulation(cfg)
	if err != nil {
		t.Fatalf("NewFastAccumulation() error = %v", err)
	}
	secondRun, err := NewFastAccumulation(cfg)
	if err != nil {
		t.Fatalf("NewFastAccumulation() error = %v", err)
	}

	candles := sample5mTrendCandles()
	firstDecisions := collectDecisions(t, firstRun, candles[:6], State{})
	secondDecisions := collectDecisions(t, secondRun, append(append([]protocol.Candle{}, candles[:6]...), sampleFutureShockCandles()...), State{})
	if len(firstDecisions) < 2 || len(secondDecisions) < 2 {
		t.Fatalf("need at least 2 decisions, got %d and %d", len(firstDecisions), len(secondDecisions))
	}
	if !reflect.DeepEqual(firstDecisions[1], secondDecisions[1]) {
		t.Fatalf("second window decision changed with future candles: %#v != %#v", firstDecisions[1], secondDecisions[1])
	}
}

func collectDecisions(t *testing.T, strat *FastAccumulation, candles []protocol.Candle, state State) []WindowDecision {
	t.Helper()

	var decisions []WindowDecision
	history := make([]protocol.Candle, 0, len(candles))
	for _, candle := range candles {
		signal, err := strat.OnCandle(context.Background(), State{
			Candles:      history,
			HasPosition:  state.HasPosition,
			PositionSide: state.PositionSide,
			HeldCandles:  state.HeldCandles,
		}, candle)
		if err != nil {
			t.Fatalf("OnCandle() error = %v", err)
		}
		if signal.Decision != nil {
			decisions = append(decisions, *signal.Decision)
		}
		history = append(history, candle)
	}
	return decisions
}

func sample5mTrendCandles() []protocol.Candle {
	return []protocol.Candle{
		test5mCandle(0, 100.0, 100.8, 99.9, 100.5),
		test5mCandle(1, 100.5, 101.0, 100.4, 100.9),
		test5mCandle(2, 100.9, 101.8, 100.8, 101.6),
		test5mCandle(3, 101.6, 102.0, 101.3, 101.7),
		test5mCandle(4, 101.7, 103.0, 101.6, 102.7),
		test5mCandle(5, 102.7, 104.0, 102.6, 103.8),
		test5mCandle(6, 103.8, 104.1, 103.0, 103.2),
		test5mCandle(7, 103.2, 103.4, 102.4, 102.6),
		test5mCandle(8, 102.6, 102.8, 101.8, 102.0),
	}
}

func sampleFutureShockCandles() []protocol.Candle {
	return []protocol.Candle{
		test5mCandle(6, 103.8, 105.5, 103.7, 105.2),
		test5mCandle(7, 105.2, 105.4, 101.0, 101.5),
		test5mCandle(8, 101.5, 101.7, 98.0, 98.5),
	}
}

func test5mCandle(index int, open, high, low, close float64) protocol.Candle {
	base := time.Unix(0, 0)
	openTime := base.Add(time.Duration(index) * 5 * time.Minute)
	closeTime := openTime.Add(5*time.Minute - time.Millisecond)
	return protocol.Candle{
		Market:      "futures-um",
		Symbol:      "BTCUSDT",
		Interval:    "5m",
		OpenTimeMS:  openTime.UnixMilli(),
		CloseTimeMS: closeTime.UnixMilli(),
		Open:        open,
		High:        high,
		Low:         low,
		Close:       close,
		Volume:      1,
	}
}

func bullishWindow(open, high, low, close float64, offset time.Duration) AggregatedWindow {
	start := time.Unix(0, 0).Add(offset).UnixMilli()
	return AggregatedWindow{
		Symbol:        "BTCUSDT",
		Market:        "futures-um",
		Interval:      "15m",
		WindowStartMS: start,
		WindowEndMS:   start + 15*60*1000 - 1,
		Open:          open,
		High:          high,
		Low:           low,
		Close:         close,
		CandleCount:   3,
	}
}

func bearishWindow(open, high, low, close float64, offset time.Duration) AggregatedWindow {
	return bullishWindow(open, high, low, close, offset)
}
