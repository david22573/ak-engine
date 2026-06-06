package backtest

import (
	"context"
	"math"
	"testing"
	"time"

	"github.com/davidmiguel22573/ak-engine/internal/data"
	"github.com/davidmiguel22573/ak-engine/internal/strategy"
	"github.com/davidmiguel22573/ak-engine/pkg/protocol"
)

type stubSource struct {
	name    string
	candles []protocol.Candle
}

func (s stubSource) LoadCandles(_ context.Context, _ data.CandleRequest) ([]protocol.Candle, error) {
	out := make([]protocol.Candle, len(s.candles))
	copy(out, s.candles)
	return out, nil
}

func (s stubSource) Name() string {
	return s.name
}

type stubStrategy struct {
	name    string
	signals map[int]strategy.Signal
}

func (s stubStrategy) Name() string {
	return s.name
}

func (s stubStrategy) OnCandle(_ context.Context, state strategy.State, _ protocol.Candle) (strategy.Signal, error) {
	return s.signals[len(state.Candles)], nil
}

func TestEngineLongTakeProfit(t *testing.T) {
	t.Parallel()

	engine := newTestEngine(t, []protocol.Candle{
		testCandle(0, 100, 100, 100, 100),
		testCandle(1, 100, 102, 100, 101),
		testCandle(2, 101, 103, 101, 102),
	}, stubStrategy{
		name: "test-long",
		signals: map[int]strategy.Signal{
			1: {Side: strategy.SideLong, StopLossBPS: 50, TakeProfitBPS: 50, MaxHoldCandles: 3},
		},
	})

	report, err := engine.Run(context.Background(), data.CandleRequest{
		Source:   "stub",
		Market:   "futures-um",
		Symbol:   "BTCUSDT",
		Interval: "5m",
	})
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}

	if got := len(report.Trades); got != 1 {
		t.Fatalf("len(report.Trades) = %d, want 1", got)
	}
	if report.Trades[0].ExitReason != ExitReasonTakeProfit {
		t.Fatalf("ExitReason = %q, want %q", report.Trades[0].ExitReason, ExitReasonTakeProfit)
	}
	if report.Trades[0].NetPnL <= 0 {
		t.Fatalf("NetPnL = %f, want > 0", report.Trades[0].NetPnL)
	}
}

func TestEngineConservativeSameCandleExitForShort(t *testing.T) {
	t.Parallel()

	engine := newTestEngine(t, []protocol.Candle{
		testCandle(0, 100, 100, 100, 100),
		testCandle(1, 100, 100, 98, 99),
		testCandle(2, 99, 100, 98, 99),
	}, stubStrategy{
		name: "test-short",
		signals: map[int]strategy.Signal{
			1: {Side: strategy.SideShort, StopLossBPS: 50, TakeProfitBPS: 50, MaxHoldCandles: 3},
		},
	})

	report, err := engine.Run(context.Background(), data.CandleRequest{
		Source:   "stub",
		Market:   "futures-um",
		Symbol:   "BTCUSDT",
		Interval: "5m",
	})
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}

	if got := len(report.Trades); got != 1 {
		t.Fatalf("len(report.Trades) = %d, want 1", got)
	}
	if report.Trades[0].ExitReason != ExitReasonStopLoss {
		t.Fatalf("ExitReason = %q, want %q", report.Trades[0].ExitReason, ExitReasonStopLoss)
	}
	if report.Trades[0].NetPnL >= 0 {
		t.Fatalf("NetPnL = %f, want < 0", report.Trades[0].NetPnL)
	}
}

func TestEngineTimeStop(t *testing.T) {
	t.Parallel()

	engine := newTestEngine(t, []protocol.Candle{
		testCandle(0, 100, 100, 100, 100),
		testCandle(1, 100, 101, 100, 101),
		testCandle(2, 101, 101.2, 100.8, 101.05),
		testCandle(3, 101.05, 101.2, 100.9, 101.1),
	}, stubStrategy{
		name: "test-time-stop",
		signals: map[int]strategy.Signal{
			1: {Side: strategy.SideLong, StopLossBPS: 500, TakeProfitBPS: 500, MaxHoldCandles: 2},
		},
	})

	report, err := engine.Run(context.Background(), data.CandleRequest{
		Source:   "stub",
		Market:   "futures-um",
		Symbol:   "BTCUSDT",
		Interval: "5m",
	})
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}

	if got := len(report.Trades); got != 1 {
		t.Fatalf("len(report.Trades) = %d, want 1", got)
	}
	if report.Trades[0].ExitReason != ExitReasonTimeStop {
		t.Fatalf("ExitReason = %q, want %q", report.Trades[0].ExitReason, ExitReasonTimeStop)
	}
}

func TestEngineForceClosesLongAtEndOfData(t *testing.T) {
	t.Parallel()

	engine := newTestEngineWithConfig(t, []protocol.Candle{
		testCandle(0, 100, 100, 100, 100),
		testCandle(1, 100, 101, 100, 101),
	}, stubStrategy{
		name: "test-long-eod",
		signals: map[int]strategy.Signal{
			1: {Side: strategy.SideLong, StopLossBPS: 500, TakeProfitBPS: 500, MaxHoldCandles: 10},
		},
	}, Config{
		StartingCash:    1000,
		MaxPositionSize: 1,
		SlippageBPS:     0,
		Fees:            FeeConfig{TakerFeeBPS: 10},
	})

	report, err := engine.Run(context.Background(), data.CandleRequest{
		Source:   "stub",
		Market:   "futures-um",
		Symbol:   "BTCUSDT",
		Interval: "5m",
	})
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}

	if got := len(report.Trades); got != 1 {
		t.Fatalf("len(report.Trades) = %d, want 1", got)
	}
	if report.Trades[0].ExitReason != ExitReasonEndOfData {
		t.Fatalf("ExitReason = %q, want %q", report.Trades[0].ExitReason, ExitReasonEndOfData)
	}
	if report.Metrics.OpenPositionCount != 0 {
		t.Fatalf("OpenPositionCount = %d, want 0", report.Metrics.OpenPositionCount)
	}
}

func TestEngineForceClosesShortAtEndOfData(t *testing.T) {
	t.Parallel()

	engine := newTestEngineWithConfig(t, []protocol.Candle{
		testCandle(0, 100, 100, 100, 100),
		testCandle(1, 100, 100, 99, 99),
	}, stubStrategy{
		name: "test-short-eod",
		signals: map[int]strategy.Signal{
			1: {Side: strategy.SideShort, StopLossBPS: 500, TakeProfitBPS: 500, MaxHoldCandles: 10},
		},
	}, Config{
		StartingCash:    1000,
		MaxPositionSize: 1,
		SlippageBPS:     0,
		Fees:            FeeConfig{TakerFeeBPS: 10},
	})

	report, err := engine.Run(context.Background(), data.CandleRequest{
		Source:   "stub",
		Market:   "futures-um",
		Symbol:   "BTCUSDT",
		Interval: "5m",
	})
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}

	if got := len(report.Trades); got != 1 {
		t.Fatalf("len(report.Trades) = %d, want 1", got)
	}
	if report.Trades[0].ExitReason != ExitReasonEndOfData {
		t.Fatalf("ExitReason = %q, want %q", report.Trades[0].ExitReason, ExitReasonEndOfData)
	}
	if report.Metrics.OpenPositionCount != 0 {
		t.Fatalf("OpenPositionCount = %d, want 0", report.Metrics.OpenPositionCount)
	}
}

func TestEngineForceCloseAppliesFeeAndSlippage(t *testing.T) {
	t.Parallel()

	engine := newTestEngineWithConfig(t, []protocol.Candle{
		testCandle(0, 100, 100, 100, 100),
		testCandle(1, 100, 101, 100, 101),
	}, stubStrategy{
		name: "test-eod-costs",
		signals: map[int]strategy.Signal{
			1: {Side: strategy.SideLong, StopLossBPS: 500, TakeProfitBPS: 500, MaxHoldCandles: 10},
		},
	}, Config{
		StartingCash:    1000,
		MaxPositionSize: 1,
		SlippageBPS:     10,
		Fees:            FeeConfig{TakerFeeBPS: 10},
	})

	report, err := engine.Run(context.Background(), data.CandleRequest{
		Source:   "stub",
		Market:   "futures-um",
		Symbol:   "BTCUSDT",
		Interval: "5m",
	})
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}

	trade := report.Trades[0]
	if trade.ExitFee <= 0 {
		t.Fatalf("ExitFee = %f, want > 0", trade.ExitFee)
	}
	if trade.SlippagePaid <= 0 {
		t.Fatalf("SlippagePaid = %f, want > 0", trade.SlippagePaid)
	}
	if report.SlippagePaid <= 0 {
		t.Fatalf("report.SlippagePaid = %f, want > 0", report.SlippagePaid)
	}
}

func TestEngineLongMFEAndMAEDiagnostics(t *testing.T) {
	t.Parallel()

	engine := newTestEngineWithConfig(t, []protocol.Candle{
		testCandle(0, 100, 100, 100, 100),
		testCandle(1, 100, 100, 100, 100),
		testCandle(2, 100, 103, 99, 102),
	}, stubStrategy{
		name: "diag-long",
		signals: map[int]strategy.Signal{
			1: {Side: strategy.SideLong, StopLossBPS: 100, TakeProfitBPS: 500, MaxHoldCandles: 5},
		},
	}, Config{StartingCash: 1000, MaxPositionSize: 1, SlippageBPS: 0, Fees: FeeConfig{}})

	report, err := engine.Run(context.Background(), data.CandleRequest{Source: "stub", Market: "futures-um", Symbol: "BTCUSDT", Interval: "5m"})
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}

	trade := report.Trades[0]
	if math.Abs(trade.MFEBPS-300) > 0.001 {
		t.Fatalf("MFEBPS = %f, want 300", trade.MFEBPS)
	}
	if math.Abs(trade.MAEBPS-100) > 0.001 {
		t.Fatalf("MAEBPS = %f, want 100", trade.MAEBPS)
	}
	if math.Abs(trade.MFER-3) > 0.001 {
		t.Fatalf("MFER = %f, want 3", trade.MFER)
	}
	if math.Abs(trade.MAER-1) > 0.001 {
		t.Fatalf("MAER = %f, want 1", trade.MAER)
	}
}

func TestEngineShortMFEAndMAEDiagnostics(t *testing.T) {
	t.Parallel()

	engine := newTestEngineWithConfig(t, []protocol.Candle{
		testCandle(0, 100, 100, 100, 100),
		testCandle(1, 100, 100, 100, 100),
		testCandle(2, 100, 101, 96, 97),
	}, stubStrategy{
		name: "diag-short",
		signals: map[int]strategy.Signal{
			1: {Side: strategy.SideShort, StopLossBPS: 100, TakeProfitBPS: 500, MaxHoldCandles: 5},
		},
	}, Config{StartingCash: 1000, MaxPositionSize: 1, SlippageBPS: 0, Fees: FeeConfig{}})

	report, err := engine.Run(context.Background(), data.CandleRequest{Source: "stub", Market: "futures-um", Symbol: "BTCUSDT", Interval: "5m"})
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}

	trade := report.Trades[0]
	if math.Abs(trade.MFEBPS-400) > 0.001 {
		t.Fatalf("MFEBPS = %f, want 400", trade.MFEBPS)
	}
	if math.Abs(trade.MAEBPS-100) > 0.001 {
		t.Fatalf("MAEBPS = %f, want 100", trade.MAEBPS)
	}
}

func TestEngineRMultipleCalculatedCorrectly(t *testing.T) {
	t.Parallel()

	engine := newTestEngineWithConfig(t, []protocol.Candle{
		testCandle(0, 100, 100, 100, 100),
		testCandle(1, 100, 100, 100, 100),
		testCandle(2, 100, 102, 100, 102),
	}, stubStrategy{
		name: "diag-r",
		signals: map[int]strategy.Signal{
			1: {Side: strategy.SideLong, StopLossBPS: 100, TakeProfitBPS: 500, MaxHoldCandles: 5},
		},
	}, Config{StartingCash: 1000, MaxPositionSize: 1, SlippageBPS: 0, Fees: FeeConfig{}})

	report, err := engine.Run(context.Background(), data.CandleRequest{Source: "stub", Market: "futures-um", Symbol: "BTCUSDT", Interval: "5m"})
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}

	trade := report.Trades[0]
	if math.Abs(trade.RealizedRMultiple-2) > 0.001 {
		t.Fatalf("RealizedRMultiple = %f, want 2", trade.RealizedRMultiple)
	}
	if math.Abs(trade.MaxPossibleRMultiple-2) > 0.001 {
		t.Fatalf("MaxPossibleRMultiple = %f, want 2", trade.MaxPossibleRMultiple)
	}
}

func TestPartialTakeProfitReducesPositionSize(t *testing.T) {
	t.Parallel()

	engine := newTestEngineWithConfig(t, nil, stubStrategy{name: "partial"}, Config{StartingCash: 1000, MaxPositionSize: 1, SlippageBPS: 0, Fees: FeeConfig{}})
	pos := &Position{
		Side:             strategy.SideLong,
		EntryPrice:       100,
		BaseEntryPrice:   100,
		Quantity:         10,
		OriginalQuantity: 10,
		EstimatedCostBPS: 0,
		InitialStopPrice: 99,
		ExitPlan: strategy.ExitPlan{
			Model:                     strategy.ExitModelPartialTPTrail,
			PartialTakeProfitR:        1,
			PartialTakeProfitFraction: 0.5,
		},
	}

	if err := engine.applyPartialTakeProfit(pos, testCandle(2, 100, 101.5, 100, 101), 1); err != nil {
		t.Fatalf("applyPartialTakeProfit() error = %v", err)
	}
	if math.Abs(pos.Quantity-5) > 0.001 {
		t.Fatalf("Quantity = %f, want 5", pos.Quantity)
	}
	if pos.PartialExitCount != 1 {
		t.Fatalf("PartialExitCount = %d, want 1", pos.PartialExitCount)
	}
}

func TestBreakevenStopMovesOnlyAfterThreshold(t *testing.T) {
	t.Parallel()

	engine := newTestEngineWithConfig(t, nil, stubStrategy{name: "be"}, Config{StartingCash: 1000, MaxPositionSize: 1, SlippageBPS: 0, Fees: FeeConfig{}})
	pos := &Position{
		Side:              strategy.SideLong,
		EntryPrice:        100,
		InitialStopPrice:  99,
		StopPrice:         99,
		EstimatedCostBPS:  10,
		MaxFavorablePrice: 100.8,
		ExitPlan: strategy.ExitPlan{
			Model:             strategy.ExitModelBreakevenAfter1R,
			BreakevenTriggerR: 1,
		},
	}

	if _, _, _, err := engine.applyResearchExitManagement(pos, testCandle(2, 100, 100.8, 99.9, 100.5)); err != nil {
		t.Fatalf("applyResearchExitManagement() error = %v", err)
	}
	if pos.StopPrice != 99 {
		t.Fatalf("StopPrice = %f, want unchanged 99", pos.StopPrice)
	}

	pos.MaxFavorablePrice = 101.2
	if _, _, _, err := engine.applyResearchExitManagement(pos, testCandle(3, 100.5, 101.2, 100.4, 101)); err != nil {
		t.Fatalf("applyResearchExitManagement() error = %v", err)
	}
	if pos.StopPrice <= 100 {
		t.Fatalf("StopPrice = %f, want moved to breakeven-plus-cost", pos.StopPrice)
	}
}

func TestCutIfNoProgressExitsStaleTrades(t *testing.T) {
	t.Parallel()

	engine := newTestEngineWithConfig(t, nil, stubStrategy{name: "cut"}, Config{StartingCash: 1000, MaxPositionSize: 1, SlippageBPS: 0, Fees: FeeConfig{}})
	pos := &Position{
		Side:              strategy.SideLong,
		EntryPrice:        100,
		InitialStopPrice:  99,
		HeldCandles:       3,
		MaxFavorablePrice: 100.2,
		ExitPlan: strategy.ExitPlan{
			Model:                strategy.ExitModelCutIfNoProgress,
			CutNoProgressR:       0.5,
			CutNoProgressCandles: 3,
		},
	}

	shouldExit, _, reason, err := engine.applyResearchExitManagement(pos, testCandle(3, 100.1, 100.2, 99.9, 100))
	if err != nil {
		t.Fatalf("applyResearchExitManagement() error = %v", err)
	}
	if !shouldExit || reason != ExitReasonCutNoProg {
		t.Fatalf("shouldExit=%v reason=%q, want true/%q", shouldExit, reason, ExitReasonCutNoProg)
	}
}

func newTestEngine(t *testing.T, candles []protocol.Candle, strat strategy.Strategy) *Engine {
	t.Helper()

	return newTestEngineWithConfig(t, candles, strat, Config{
		StartingCash:    1000,
		MaxPositionSize: 1,
		SlippageBPS:     0,
		Fees: FeeConfig{
			TakerFeeBPS: 10,
		},
	})
}

func newTestEngineWithConfig(t *testing.T, candles []protocol.Candle, strat strategy.Strategy, cfg Config) *Engine {
	t.Helper()

	engine, err := NewEngine(stubSource{name: "stub", candles: candles}, strat, cfg)
	if err != nil {
		t.Fatalf("NewEngine() error = %v", err)
	}
	return engine
}

func testCandle(index int, open, high, low, close float64) protocol.Candle {
	openTime := time.Unix(0, 0).Add(time.Duration(index) * 5 * time.Minute)
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
