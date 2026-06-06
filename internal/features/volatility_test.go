package features

import (
	"testing"

	"github.com/davidmiguel22573/ak-engine/pkg/protocol"
)

func TestATR(t *testing.T) {
	// Insufficient lookback
	candles := []protocol.Candle{
		{Open: 10, High: 12, Low: 9, Close: 11},
	}
	_, ok := ATR(candles, 0, 14)
	if ok {
		t.Error("expected ATR to return false for insufficient lookback")
	}

	// Build 15 candles for period=14
	var testCandles []protocol.Candle
	for i := 0; i < 15; i++ {
		testCandles = append(testCandles, protocol.Candle{
			Open: 100, High: 105, Low: 95, Close: 100,
		})
	}
	atr, ok := ATR(testCandles, 14, 14)
	if !ok {
		t.Error("expected ATR to succeed")
	}
	if atr <= 0 {
		t.Errorf("expected positive ATR, got %f", atr)
	}
}

func TestRealizedVol(t *testing.T) {
	closes := []float64{100, 101, 102, 103, 104, 105}
	_, ok := RealizedVol(closes, 5, 20)
	if ok {
		t.Error("expected RealizedVol to return false for insufficient lookback")
	}

	// 21 close values for period=20 (20 returns)
	var testCloses []float64
	for i := 0; i < 21; i++ {
		testCloses = append(testCloses, 100.0)
	}
	_, ok = RealizedVol(testCloses, 20, 20)
	if !ok {
		t.Error("expected RealizedVol to succeed")
	}
}

func TestBBWidth(t *testing.T) {
	closes := []float64{100, 101}
	_, ok := BBWidth(closes, 1, 20, 2.0)
	if ok {
		t.Error("expected BBWidth to return false for insufficient lookback")
	}
}

func TestPercentRankTrailing(t *testing.T) {
	values := []float64{10, 20, 30, 40, 50, 60, 25}
	rank, ok := PercentRankTrailing(values, 6, 6)
	if !ok {
		t.Error("expected PercentRankTrailing to succeed")
	}
	// values[0:6] are [10, 20, 30, 40, 50, 60]. current is 25.
	// Elements <= 25 are 10, 20 (2 elements).
	// Rank = 2/6 = 0.3333...
	expected := 2.0 / 6.0
	if mathAbs(rank-expected) > 1e-5 {
		t.Errorf("expected rank %f, got %f", expected, rank)
	}
}

func mathAbs(x float64) float64 {
	if x < 0 {
		return -x
	}
	return x
}
