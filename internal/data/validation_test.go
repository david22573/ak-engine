package data

import (
	"testing"

	"github.com/davidmiguel22573/ak-engine/pkg/protocol"
)

func TestValidateCandlesRejectsEmpty(t *testing.T) {
	err := ValidateCandles("1m", []protocol.Candle{})
	if err == nil {
		t.Error("expected error for empty candles, got nil")
	}
}

func TestValidateCandlesRejectsDuplicates(t *testing.T) {
	candles := []protocol.Candle{
		{OpenTimeMS: 1000, Open: 10, High: 12, Low: 9, Close: 11},
		{OpenTimeMS: 1000, Open: 11, High: 13, Low: 10, Close: 12},
	}
	err := ValidateCandles("1m", candles)
	if err == nil {
		t.Error("expected error for duplicate timestamps, got nil")
	}
}

func TestValidateCandlesRejectsGaps(t *testing.T) {
	// 1m interval = 60000ms
	candles := []protocol.Candle{
		{OpenTimeMS: 1672531200000, Open: 10, High: 12, Low: 9, Close: 11},
		{OpenTimeMS: 1672531200000 + 120000, Open: 11, High: 13, Low: 10, Close: 12}, // Gap of 1 candle
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
			candle: protocol.Candle{OpenTimeMS: 1000, Open: 10, High: 8, Low: 9, Close: 9.5},
		},
		{
			name:   "Open < Low",
			candle: protocol.Candle{OpenTimeMS: 1000, Open: 7, High: 10, Low: 8, Close: 9},
		},
		{
			name:   "Open > High",
			candle: protocol.Candle{OpenTimeMS: 1000, Open: 11, High: 10, Low: 8, Close: 9},
		},
		{
			name:   "Close < Low",
			candle: protocol.Candle{OpenTimeMS: 1000, Open: 9, High: 10, Low: 8, Close: 7},
		},
		{
			name:   "Close > High",
			candle: protocol.Candle{OpenTimeMS: 1000, Open: 9, High: 10, Low: 8, Close: 11},
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
		{OpenTimeMS: baseTime, Open: 10, High: 12, Low: 9, Close: 11},
		{OpenTimeMS: baseTime + 60000, Open: 11, High: 13, Low: 10, Close: 12},
		{OpenTimeMS: baseTime + 120000, Open: 12, High: 14, Low: 11, Close: 13},
	}
	err := ValidateCandles("1m", candles)
	if err != nil {
		t.Errorf("expected no error for continuous 1m, got: %v", err)
	}
}

func TestValidateCandlesAcceptsContinuous5m(t *testing.T) {
	baseTime := int64(1672531200000)
	candles := []protocol.Candle{
		{OpenTimeMS: baseTime, Open: 10, High: 12, Low: 9, Close: 11},
		{OpenTimeMS: baseTime + 300000, Open: 11, High: 13, Low: 10, Close: 12},
		{OpenTimeMS: baseTime + 600000, Open: 12, High: 14, Low: 11, Close: 13},
	}
	err := ValidateCandles("5m", candles)
	if err != nil {
		t.Errorf("expected no error for continuous 5m, got: %v", err)
	}
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
