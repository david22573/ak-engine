package features

import (
	"testing"

	"github.com/david22573/ak-engine/pkg/protocol"
)

func TestBuildRows_WarmupAndFuture(t *testing.T) {
	const baseMS = int64(1_700_000_000_000)
	// Create 210 candles (minimum 200 needed for EMA200)
	var candles []protocol.Candle
	for i := 0; i < 210; i++ {
		candles = append(candles, protocol.Candle{
			Market:              "futures-um",
			Symbol:              "LINKUSDT",
			Interval:            "1m",
			OpenTimeMS:          baseMS + int64(i*60000),
			CloseTimeMS:         baseMS + int64(i*60000) + 59999,
			Open:                100,
			High:                101,
			Low:                 99,
			Close:               100,
			Volume:              1000,
			QuoteAssetVolume:    100000,
			TakerBuyBaseVolume:  500,
			TakerBuyQuoteVolume: 50000,
		})
	}

	opts := BuildOptions{
		Market:     "futures-um",
		Symbol:     "LINKUSDT",
		Interval:   "1m",
		DropWarmup: false,
	}

	rows, err := BuildRows(candles, opts)
	if err != nil {
		t.Fatalf("failed to build rows: %v", err)
	}

	if len(rows) != 210 {
		t.Errorf("expected 210 rows, got %d", len(rows))
	}

	// Warmup should be true for first 199 rows (index 0 to 198)
	for i := 0; i < 199; i++ {
		if !rows[i].Warmup {
			t.Errorf("expected row %d to be warmup", i)
		}
	}

	// Warmup should be false for row 199 and later
	for i := 199; i < len(rows); i++ {
		if rows[i].Warmup {
			t.Errorf("expected row %d to NOT be warmup", i)
		}
	}

	// Verify future leakage is prevented: modifying a future candle should not affect current row features
	// Let's modify candles[209] and rebuild, and compare features of rows[199]
	candles[209].Close = 100.5 // within [99, 101] to pass validation
	candles[209].High = 100.5
	rowsModified, err := BuildRows(candles, opts)
	if err != nil {
		t.Fatalf("failed to rebuild rows: %v", err)
	}

	// Compare rows[199] features between original and modified (they should be identical!)
	originalRow := rows[199]
	modifiedRow := rowsModified[199]

	if originalRow.Close != modifiedRow.Close ||
		originalRow.RealizedVol20 != modifiedRow.RealizedVol20 ||
		originalRow.EMA20 != modifiedRow.EMA20 ||
		originalRow.TrendSlope20 != modifiedRow.TrendSlope20 {
		t.Error("leakage detected: modifying a future candle changed features at a past index")
	}
}

func TestBuildRows_ContextReturns(t *testing.T) {
	const baseMS = int64(1_700_000_000_000)
	var candles []protocol.Candle
	var btcCandles []protocol.Candle
	var ethCandles []protocol.Candle
	for i := 0; i < 210; i++ {
		candles = append(candles, protocol.Candle{
			Market: "futures-um", Symbol: "LINKUSDT", Interval: "1m",
			OpenTimeMS: baseMS + int64(i*60000), CloseTimeMS: baseMS + int64(i*60000) + 59999,
			Open: 10, High: 11, Low: 9, Close: 10,
		})

		btcCandles = append(btcCandles, protocol.Candle{
			Market: "futures-um", Symbol: "BTCUSDT", Interval: "1m",
			OpenTimeMS: baseMS + int64(i*60000), CloseTimeMS: baseMS + int64(i*60000) + 59999,
			Open: 1000, High: 1100, Low: 900, Close: 1000 + float64(i),
		})
		ethCandles = append(ethCandles, protocol.Candle{
			Market: "futures-um", Symbol: "ETHUSDT", Interval: "1m",
			OpenTimeMS: baseMS + int64(i*60000), CloseTimeMS: baseMS + int64(i*60000) + 59999,
			Open: 100, High: 110, Low: 90, Close: 100 - float64(i),
		})
	}

	opts := BuildOptions{
		Market:     "futures-um",
		Symbol:     "LINKUSDT",
		Interval:   "1m",
		ContextBTC: btcCandles,
		ContextETH: ethCandles,
	}

	rows, err := BuildRows(candles, opts)
	if err != nil {
		t.Fatalf("failed to build rows: %v", err)
	}

	row := rows[100] // after 60 periods
	if row.BTCReturn60 == 0 {
		t.Errorf("expected BTCReturn60 != 0, got %f", row.BTCReturn60)
	}
	if row.ETHReturn60 == 0 {
		t.Errorf("expected ETHReturn60 != 0, got %f", row.ETHReturn60)
	}

	// btc prev=1040, curr=1100 -> return = 60/1040 = 0.0576923
	// eth prev=60, curr=0 -> return = -60/60 = -1
}

func TestBuildRowsUsesTruthfulCloseAvailabilityAcrossIntervals(t *testing.T) {
	const openMS = int64(1_700_000_000_000)
	for _, tt := range []struct {
		interval   string
		durationMS int64
	}{
		{interval: "1m", durationMS: 60_000},
		{interval: "5m", durationMS: 5 * 60_000},
		{interval: "1h", durationMS: 60 * 60_000},
	} {
		t.Run(tt.interval, func(t *testing.T) {
			closeMS := openMS + tt.durationMS - 1
			rows, err := BuildRows([]protocol.Candle{{
				Market: "futures-um", Symbol: "BTCUSDT", Interval: tt.interval,
				OpenTimeMS: openMS, CloseTimeMS: closeMS,
				Open: 100, High: 101, Low: 99, Close: 100, Volume: 1,
			}}, BuildOptions{Market: "futures-um", Symbol: "BTCUSDT", Interval: tt.interval})
			if err != nil {
				t.Fatalf("BuildRows() error = %v", err)
			}
			if len(rows) != 1 || rows[0].EventTimeMS != openMS || rows[0].AvailableAtMS != closeMS {
				t.Fatalf("feature time = %#v, want event=%d availability=%d", rows, openMS, closeMS)
			}
		})
	}
}

func TestBuildRowsRejectsInvalidCloseTimeWithoutFallback(t *testing.T) {
	const openMS = int64(1_700_000_000_000)
	for _, tt := range []struct {
		name        string
		interval    string
		closeTimeMS int64
	}{
		{name: "missing 1m", interval: "1m", closeTimeMS: 0},
		{name: "backdated 1m", interval: "1m", closeTimeMS: openMS - 1},
		{name: "short 5m", interval: "5m", closeTimeMS: openMS + 60_000 - 1},
		{name: "short 1h", interval: "1h", closeTimeMS: openMS + 5*60_000 - 1},
		{name: "exclusive close", interval: "1m", closeTimeMS: openMS + 60_000},
	} {
		t.Run(tt.name, func(t *testing.T) {
			_, err := BuildRows([]protocol.Candle{{
				Market: "futures-um", Symbol: "BTCUSDT", Interval: tt.interval,
				OpenTimeMS: openMS, CloseTimeMS: tt.closeTimeMS,
				Open: 100, High: 101, Low: 99, Close: 100, Volume: 1,
			}}, BuildOptions{Market: "futures-um", Symbol: "BTCUSDT", Interval: tt.interval})
			if err == nil {
				t.Fatal("invalid close time passed")
			}
		})
	}
}
