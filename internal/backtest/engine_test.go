package backtest

import (
	"context"
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
