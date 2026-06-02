package data

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestLocalJSONLoadsNormalizedCandles(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "test_local_json_*")
	if err != nil {
		t.Fatalf("failed to create tmp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	jsonContent := `[
		{
			"open_time_ms": 1704067200000,
			"open": 42000.0,
			"high": 42500.0,
			"low": 41800.0,
			"close": 42200.0,
			"volume": 100.0,
			"close_time_ms": 1704067499999,
			"quote_asset_volume": 4200000.0,
			"number_of_trades": 500,
			"taker_buy_base_volume": 60.0,
			"taker_buy_quote_volume": 2520000.0
		},
		{
			"open_time_ms": 1704067500000,
			"open": 42200.0,
			"high": 42600.0,
			"low": 42100.0,
			"close": 42400.0,
			"volume": 120.0,
			"close_time_ms": 1704067799999,
			"quote_asset_volume": 5040000.0,
			"number_of_trades": 600,
			"taker_buy_base_volume": 70.0,
			"taker_buy_quote_volume": 2940000.0
		}
	]`

	filePath := filepath.Join(tmpDir, "normalized.json")
	if err := os.WriteFile(filePath, []byte(jsonContent), 0644); err != nil {
		t.Fatalf("failed to write tmp file: %v", err)
	}

	src := NewLocalJSONSource()
	req := CandleRequest{
		Path:     filePath,
		Market:   "futures-um",
		Symbol:   "BTCUSDT",
		Interval: "5m",
	}

	candles, err := src.LoadCandles(context.Background(), req)
	if err != nil {
		t.Fatalf("failed to load candles: %v", err)
	}

	if len(candles) != 2 {
		t.Errorf("expected 2 candles, got %d", len(candles))
	}

	for _, c := range candles {
		if c.Market != "futures-um" {
			t.Errorf("expected market 'futures-um', got %q", c.Market)
		}
		if c.Symbol != "BTCUSDT" {
			t.Errorf("expected symbol 'BTCUSDT', got %q", c.Symbol)
		}
		if c.Interval != "5m" {
			t.Errorf("expected interval '5m', got %q", c.Interval)
		}
	}

	if candles[0].Open != 42000.0 || candles[1].Close != 42400.0 {
		t.Errorf("incorrect candle OHLC values loaded")
	}
}

func TestLocalJSONLoadsBinanceKline(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "test_local_json_*")
	if err != nil {
		t.Fatalf("failed to create tmp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Binance klines shape: array of arrays.
	// Using mix of string representations for numeric fields, which is standard.
	jsonContent := `[
		[
			1704067200000,
			"42000.0",
			"42500.0",
			"41800.0",
			"42200.0",
			"100.0",
			1704067499999,
			"4200000.0",
			500,
			"60.0",
			"2520000.0",
			"0"
		],
		[
			1704067500000,
			"42200.0",
			"42600.0",
			"42100.0",
			"42400.0",
			"120.0",
			1704067799999,
			"5040000.0",
			600,
			"70.0",
			"2940000.0",
			"0"
		]
	]`

	filePath := filepath.Join(tmpDir, "binance.json")
	if err := os.WriteFile(filePath, []byte(jsonContent), 0644); err != nil {
		t.Fatalf("failed to write tmp file: %v", err)
	}

	src := NewLocalJSONSource()
	req := CandleRequest{
		Path:     filePath,
		Market:   "futures-um",
		Symbol:   "BTCUSDT",
		Interval: "5m",
	}

	candles, err := src.LoadCandles(context.Background(), req)
	if err != nil {
		t.Fatalf("failed to load candles: %v", err)
	}

	if len(candles) != 2 {
		t.Errorf("expected 2 candles, got %d", len(candles))
	}

	if candles[0].OpenTimeMS != 1704067200000 || candles[0].Open != 42000.0 {
		t.Errorf("first candle parsed incorrectly: %+v", candles[0])
	}
	if candles[1].OpenTimeMS != 1704067500000 || candles[1].Close != 42400.0 {
		t.Errorf("second candle parsed incorrectly: %+v", candles[1])
	}
	if candles[0].NumberOfTrades != 500 || candles[1].NumberOfTrades != 600 {
		t.Errorf("incorrect number of trades parsed")
	}
}

func TestLocalJSONRejectsMalformedJSON(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "test_local_json_*")
	if err != nil {
		t.Fatalf("failed to create tmp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	jsonContent := `[ { "open_time_ms": 1704067200000, ` // Unclosed braces

	filePath := filepath.Join(tmpDir, "malformed.json")
	if err := os.WriteFile(filePath, []byte(jsonContent), 0644); err != nil {
		t.Fatalf("failed to write tmp file: %v", err)
	}

	src := NewLocalJSONSource()
	req := CandleRequest{
		Path:     filePath,
		Market:   "futures-um",
		Symbol:   "BTCUSDT",
		Interval: "5m",
	}

	_, err = src.LoadCandles(context.Background(), req)
	if err == nil {
		t.Error("expected error for malformed JSON, got nil")
	}
}

func TestLocalJSONRejectsUnsupportedShape(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "test_local_json_*")
	if err != nil {
		t.Fatalf("failed to create tmp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Array of simple values is unsupported
	jsonContent := `[ 1, 2, 3, 4 ]`

	filePath := filepath.Join(tmpDir, "unsupported_shape.json")
	if err := os.WriteFile(filePath, []byte(jsonContent), 0644); err != nil {
		t.Fatalf("failed to write tmp file: %v", err)
	}

	src := NewLocalJSONSource()
	req := CandleRequest{
		Path:     filePath,
		Market:   "futures-um",
		Symbol:   "BTCUSDT",
		Interval: "5m",
	}

	_, err = src.LoadCandles(context.Background(), req)
	if err == nil {
		t.Error("expected error for unsupported JSON shape, got nil")
	}
}

func TestLocalJSONFiltersByDateRange(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "test_local_json_*")
	if err != nil {
		t.Fatalf("failed to create tmp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// 3 candles spanning 1704067200000 (2024-01-01 00:00:00 UTC) to 1704067800000 (2024-01-01 00:10:00 UTC)
	jsonContent := `[
		{
			"open_time_ms": 1704067200000,
			"open": 42000.0, "high": 42100.0, "low": 41900.0, "close": 42050.0,
			"interval": "5m"
		},
		{
			"open_time_ms": 1704067500000,
			"open": 42050.0, "high": 42150.0, "low": 42000.0, "close": 42100.0,
			"interval": "5m"
		},
		{
			"open_time_ms": 1704067800000,
			"open": 42100.0, "high": 42200.0, "low": 42050.0, "close": 42150.0,
			"interval": "5m"
		}
	]`

	filePath := filepath.Join(tmpDir, "filter.json")
	if err := os.WriteFile(filePath, []byte(jsonContent), 0644); err != nil {
		t.Fatalf("failed to write tmp file: %v", err)
	}

	src := NewLocalJSONSource()

	// Filter case 1: From 2024-01-01 00:02:00 to 2024-01-01 00:07:00 -> Only the middle candle (1704067500000)
	req := CandleRequest{
		Path:     filePath,
		Market:   "futures-um",
		Symbol:   "BTCUSDT",
		Interval: "5m",
		From:     time.Date(2024, 1, 1, 0, 2, 0, 0, time.UTC),
		To:       time.Date(2024, 1, 1, 0, 7, 0, 0, time.UTC),
	}

	candles, err := src.LoadCandles(context.Background(), req)
	if err != nil {
		t.Fatalf("failed to load candles: %v", err)
	}

	if len(candles) != 1 {
		t.Fatalf("expected exactly 1 filtered candle, got %d", len(candles))
	}
	if candles[0].OpenTimeMS != 1704067500000 {
		t.Errorf("expected candle at 1704067500000, got %d", candles[0].OpenTimeMS)
	}
}
