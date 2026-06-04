package strategy

import (
	"fmt"

	"github.com/davidmiguel22573/ak-engine/internal/data"
	"github.com/davidmiguel22573/ak-engine/pkg/protocol"
)

type AggregatedWindow struct {
	Symbol        string
	Market        string
	Interval      string
	WindowStartMS int64
	WindowEndMS   int64
	Open          float64
	High          float64
	Low           float64
	Close         float64
	Volume        float64
	CandleCount   int
}

func (w AggregatedWindow) RangeBPS() float64 {
	if w.Close <= 0 {
		return 0
	}
	return ((w.High - w.Low) / w.Close) * 10000
}

func (w AggregatedWindow) BodyBPS() float64 {
	if w.Open <= 0 {
		return 0
	}
	return absFloat((w.Close - w.Open) / w.Open * 10000)
}

func (w AggregatedWindow) Direction() Side {
	switch {
	case w.Close > w.Open:
		return SideLong
	case w.Close < w.Open:
		return SideShort
	default:
		return SideNone
	}
}

type WindowAggregator struct {
	targetMS  int64
	inputMS   int64
	activeMS  int64
	candleBuf []protocol.Candle
}

func NewWindowAggregator(targetInterval string) (*WindowAggregator, error) {
	targetMS, err := data.ParseIntervalToMS(targetInterval)
	if err != nil {
		return nil, fmt.Errorf("parse target interval: %w", err)
	}
	return &WindowAggregator{targetMS: targetMS}, nil
}

func (a *WindowAggregator) CandlesPerWindow() int {
	if a.inputMS <= 0 {
		return 0
	}
	return int(a.targetMS / a.inputMS)
}

func (a *WindowAggregator) Add(candle protocol.Candle) (*AggregatedWindow, error) {
	inputMS, err := candleDurationMS(candle)
	if err != nil {
		return nil, err
	}
	if a.inputMS == 0 {
		if a.targetMS%inputMS != 0 {
			return nil, fmt.Errorf("target interval %dms not divisible by input interval %dms", a.targetMS, inputMS)
		}
		a.inputMS = inputMS
	}
	if inputMS != a.inputMS {
		return nil, fmt.Errorf("input interval changed from %dms to %dms", a.inputMS, inputMS)
	}

	windowStart := alignWindowStart(candle.OpenTimeMS, a.targetMS)
	if len(a.candleBuf) == 0 || windowStart != a.activeMS {
		a.activeMS = windowStart
		a.candleBuf = a.candleBuf[:0]
	}

	a.candleBuf = append(a.candleBuf, candle)
	windowEndExclusive := candle.CloseTimeMS + 1
	if windowEndExclusive < windowStart+a.targetMS {
		return nil, nil
	}
	if windowEndExclusive > windowStart+a.targetMS {
		return nil, fmt.Errorf("candle close %d exceeded window end %d", windowEndExclusive, windowStart+a.targetMS)
	}
	if len(a.candleBuf) != a.CandlesPerWindow() {
		a.candleBuf = a.candleBuf[:0]
		return nil, nil
	}

	window := aggregateCandles(a.candleBuf, a.activeMS, a.targetMS)
	a.candleBuf = a.candleBuf[:0]
	return &window, nil
}

func BuildHourlyContext(windows []AggregatedWindow) (*AggregatedWindow, bool) {
	if len(windows) < 4 {
		return nil, false
	}
	last := windows[len(windows)-4:]
	if alignWindowStart(last[0].WindowStartMS, int64(60*60*1000)) != last[0].WindowStartMS {
		return nil, false
	}
	for i := 1; i < len(last); i++ {
		if last[i].WindowStartMS != last[i-1].WindowStartMS+15*60*1000 {
			return nil, false
		}
	}
	hourly := AggregatedWindow{
		Symbol:        last[0].Symbol,
		Market:        last[0].Market,
		Interval:      "1h",
		WindowStartMS: last[0].WindowStartMS,
		WindowEndMS:   last[len(last)-1].WindowEndMS,
		Open:          last[0].Open,
		High:          last[0].High,
		Low:           last[0].Low,
		Close:         last[len(last)-1].Close,
	}
	for _, window := range last {
		if window.High > hourly.High {
			hourly.High = window.High
		}
		if window.Low < hourly.Low {
			hourly.Low = window.Low
		}
		hourly.Volume += window.Volume
		hourly.CandleCount += window.CandleCount
	}
	return &hourly, true
}

func alignWindowStart(openTimeMS, targetMS int64) int64 {
	return openTimeMS - (openTimeMS % targetMS)
}

func candleDurationMS(candle protocol.Candle) (int64, error) {
	if candle.Interval != "" {
		ms, err := data.ParseIntervalToMS(candle.Interval)
		if err == nil {
			return ms, nil
		}
	}
	dur := candle.CloseTimeMS - candle.OpenTimeMS + 1
	if dur <= 0 {
		return 0, fmt.Errorf("candle duration must be > 0")
	}
	return dur, nil
}

func aggregateCandles(candles []protocol.Candle, startMS, targetMS int64) AggregatedWindow {
	window := AggregatedWindow{
		Symbol:        candles[0].Symbol,
		Market:        candles[0].Market,
		Interval:      "15m",
		WindowStartMS: startMS,
		WindowEndMS:   startMS + targetMS - 1,
		Open:          candles[0].Open,
		High:          candles[0].High,
		Low:           candles[0].Low,
		Close:         candles[len(candles)-1].Close,
		CandleCount:   len(candles),
	}
	for _, candle := range candles {
		if candle.High > window.High {
			window.High = candle.High
		}
		if candle.Low < window.Low {
			window.Low = candle.Low
		}
		window.Volume += candle.Volume
	}
	return window
}
