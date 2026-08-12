package executionseries

import (
	"testing"

	"github.com/david22573/ak-engine/pkg/protocol"
)

func TestResolveStartsAtFirstCandleAfterDecision(t *testing.T) {
	candles := []protocol.Candle{
		{OpenTimeMS: 1_000, Open: 100, High: 200, Low: 1, Close: 100},
		{OpenTimeMS: 61_000, Open: 101, High: 102, Low: 100, Close: 101.5},
		{OpenTimeMS: 121_000, Open: 102, High: 103, Low: 101, Close: 102.5},
	}
	window, err := Resolve(1_000, 60_999, 60_999, candles, 0, 1, 0)
	if err != nil {
		t.Fatal(err)
	}
	if window.EntryIndex != 1 || window.EndIndex != 1 || window.EntryPrice != 101 || window.ExitPrice != 101.5 || window.FillTimeMS != 61_000 {
		t.Fatalf("unexpected canonical window: %#v", window)
	}
}

func TestResolveDiagnosticDelayIsAfterCanonicalTradableCandle(t *testing.T) {
	candles := []protocol.Candle{
		{OpenTimeMS: 1_000, Open: 100, Close: 100},
		{OpenTimeMS: 61_000, Open: 101, Close: 101},
		{OpenTimeMS: 121_000, Open: 102, Close: 102},
		{OpenTimeMS: 181_000, Open: 103, Close: 103},
	}
	window, err := Resolve(1_000, 60_999, 60_999, candles, 1, 1, 0)
	if err != nil {
		t.Fatal(err)
	}
	if window.NextTradableTimeMS != 61_000 || window.FillTimeMS != 121_000 || window.EntryIndex != 2 {
		t.Fatalf("unexpected delayed window: %#v", window)
	}
}
