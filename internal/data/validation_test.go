package data

import (
	"math"
	"testing"

	"github.com/david22573/ak-engine/pkg/protocol"
)

func TestValidateCandlesRejectsEmpty(t *testing.T) {
	err := ValidateCandles("1m", []protocol.Candle{})
	if err == nil {
		t.Error("expected error for empty candles, got nil")
	}
}

func TestValidateCandlesRejectsDuplicates(t *testing.T) {
	candles := []protocol.Candle{
		validCandle(1000, 60000, 10, 12, 9, 11),
		validCandle(1000, 60000, 11, 13, 10, 12),
	}
	err := ValidateCandles("1m", candles)
	if err == nil {
		t.Error("expected error for duplicate timestamps, got nil")
	}
}

func TestValidateCandlesRejectsGaps(t *testing.T) {
	// 1m interval = 60000ms
	candles := []protocol.Candle{
		validCandle(1672531200000, 60000, 10, 12, 9, 11),
		validCandle(1672531200000+120000, 60000, 11, 13, 10, 12), // Gap of 1 candle
	}
	err := ValidateCandles("1m", candles)
	if err == nil {
		t.Error("expected error for gap in timestamps, got nil")
	}
}

func TestValidateCandlesRejectsBadOHLC(t *testing.T) {
	tests := []struct {
		name   string
		candle protocol.Candle
	}{
		{
			name:   "High < Low",
			candle: validCandle(1000, 60000, 10, 8, 9, 9.5),
		},
		{
			name:   "Open < Low",
			candle: validCandle(1000, 60000, 7, 10, 8, 9),
		},
		{
			name:   "Open > High",
			candle: validCandle(1000, 60000, 11, 10, 8, 9),
		},
		{
			name:   "Close < Low",
			candle: validCandle(1000, 60000, 9, 10, 8, 7),
		},
		{
			name:   "Close > High",
			candle: validCandle(1000, 60000, 9, 10, 8, 11),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateCandles("1m", []protocol.Candle{tt.candle})
			if err == nil {
				t.Errorf("expected error for bad OHLC (%s), got nil", tt.name)
			}
		})
	}
}

func TestValidateCandlesAcceptsContinuous1m(t *testing.T) {
	baseTime := int64(1672531200000)
	candles := []protocol.Candle{
		validCandle(baseTime, 60000, 10, 12, 9, 11),
		validCandle(baseTime+60000, 60000, 11, 13, 10, 12),
		validCandle(baseTime+120000, 60000, 12, 14, 11, 13),
	}
	err := ValidateCandles("1m", candles)
	if err != nil {
		t.Errorf("expected no error for continuous 1m, got: %v", err)
	}
}

func TestValidateCandlesAcceptsContinuous5m(t *testing.T) {
	baseTime := int64(1672531200000)
	candles := []protocol.Candle{
		validCandle(baseTime, 300000, 10, 12, 9, 11),
		validCandle(baseTime+300000, 300000, 11, 13, 10, 12),
		validCandle(baseTime+600000, 300000, 12, 14, 11, 13),
	}
	err := ValidateCandles("5m", candles)
	if err != nil {
		t.Errorf("expected no error for continuous 5m, got: %v", err)
	}
}

func TestValidateCandlesRejectsDisorderSubCadenceNonfiniteNonpositiveAndBadClose(t *testing.T) {
	base := int64(1672531200000)
	tests := []struct {
		name    string
		candles []protocol.Candle
	}{
		{"out of order", []protocol.Candle{validCandle(base+60000, 60000, 10, 11, 9, 10), validCandle(base, 60000, 10, 11, 9, 10)}},
		{"30s in 1m", []protocol.Candle{validCandle(base, 60000, 10, 11, 9, 10), validCandle(base+30000, 60000, 10, 11, 9, 10)}},
		{"nan", []protocol.Candle{validCandle(base, 60000, math.NaN(), 11, 9, 10)}},
		{"nonpositive", []protocol.Candle{validCandle(base, 60000, 0, 11, 0, 10)}},
		{"bad close time", []protocol.Candle{func() protocol.Candle { c := validCandle(base, 60000, 10, 11, 9, 10); c.CloseTimeMS++; return c }()}},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if err := ValidateCandles("1m", tc.candles); err == nil {
				t.Fatal("expected rejection")
			}
		})
	}
}

func TestValidateCandlesForRequestRejectsWrongScope(t *testing.T) {
	c := validCandle(1672531200000, 60000, 10, 11, 9, 10)
	c.Market, c.Symbol, c.Interval = "spot", "ETHUSDT", "5m"
	if err := ValidateCandlesForRequest(CandleRequest{Market: "futures-um", Symbol: "BTCUSDT", Interval: "1m"}, []protocol.Candle{c}); err == nil {
		t.Fatal("expected exact scope rejection")
	}
}

func validCandle(openTime, duration int64, open, high, low, close float64) protocol.Candle {
	return protocol.Candle{OpenTimeMS: openTime, CloseTimeMS: openTime + duration - 1, Open: open, High: high, Low: low, Close: close, Volume: 1}
}

func TestAnalyzeCandles(t *testing.T) {
	baseTime := int64(1672531200000)
	candles := []protocol.Candle{
		{OpenTimeMS: baseTime, Open: 10, High: 12, Low: 9, Close: 11},
		{OpenTimeMS: baseTime, Open: 10, High: 12, Low: 9, Close: 11}, // 1 duplicate
		{OpenTimeMS: baseTime + 60000, Open: 11, High: 13, Low: 10, Close: 12},
		{OpenTimeMS: baseTime + 180000, Open: 12, High: 14, Low: 11, Close: 13}, // 1 gap (missing 120000)
	}

	analysis := AnalyzeCandles("1m", candles)
	if analysis.Count != 4 {
		t.Errorf("expected count 4, got %d", analysis.Count)
	}
	if analysis.FirstMS != baseTime {
		t.Errorf("expected first MS %d, got %d", baseTime, analysis.FirstMS)
	}
	if analysis.LastMS != baseTime+180000 {
		t.Errorf("expected last MS %d, got %d", baseTime+180000, analysis.LastMS)
	}
	if analysis.Duplicates != 1 {
		t.Errorf("expected 1 duplicate, got %d", analysis.Duplicates)
	}
	if analysis.Gaps != 1 {
		t.Errorf("expected 1 gap, got %d", analysis.Gaps)
	}
}
