package data

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/xitongsys/parquet-go-source/local"
	"github.com/xitongsys/parquet-go/writer"
)

type ParquetCandle struct {
	Market              *string  `parquet:"name=market, type=BYTE_ARRAY, convertedtype=UTF8, repetitiontype=OPTIONAL"`
	Symbol              *string  `parquet:"name=symbol, type=BYTE_ARRAY, convertedtype=UTF8, repetitiontype=OPTIONAL"`
	Interval            *string  `parquet:"name=interval, type=BYTE_ARRAY, convertedtype=UTF8, repetitiontype=OPTIONAL"`
	OpenTime            *int64   `parquet:"name=open_time, type=INT64, repetitiontype=OPTIONAL"`
	OpenTimeMS          *int64   `parquet:"name=open_time_ms, type=INT64, repetitiontype=OPTIONAL"`
	Open                *float64 `parquet:"name=open, type=DOUBLE, repetitiontype=OPTIONAL"`
	High                *float64 `parquet:"name=high, type=DOUBLE, repetitiontype=OPTIONAL"`
	Low                 *float64 `parquet:"name=low, type=DOUBLE, repetitiontype=OPTIONAL"`
	Close               *float64 `parquet:"name=close, type=DOUBLE, repetitiontype=OPTIONAL"`
	Volume              *float64 `parquet:"name=volume, type=DOUBLE, repetitiontype=OPTIONAL"`
	CloseTime           *int64   `parquet:"name=close_time, type=INT64, repetitiontype=OPTIONAL"`
	CloseTimeMS         *int64   `parquet:"name=close_time_ms, type=INT64, repetitiontype=OPTIONAL"`
	QuoteAssetVolume    *float64 `parquet:"name=quote_asset_volume, type=DOUBLE, repetitiontype=OPTIONAL"`
	NumberOfTrades      *int64   `parquet:"name=number_of_trades, type=INT64, repetitiontype=OPTIONAL"`
	TakerBuyBaseVolume  *float64 `parquet:"name=taker_buy_base_volume, type=DOUBLE, repetitiontype=OPTIONAL"`
	TakerBuyQuoteVolume *float64 `parquet:"name=taker_buy_quote_volume, type=DOUBLE, repetitiontype=OPTIONAL"`
}

func TestFilterParquetFilesByRangeMonthly(t *testing.T) {
	// LINKUSDT-1m-2023-01.parquet
	start, end, err := ParseDateRangeFromFilename("LINKUSDT-1m-2023-01.parquet")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	expectedStart := time.Date(2023, time.January, 1, 0, 0, 0, 0, time.UTC)
	expectedEnd := time.Date(2023, time.February, 1, 0, 0, 0, 0, time.UTC).Add(-time.Millisecond)

	if !start.Equal(expectedStart) {
		t.Errorf("expected start %v, got %v", expectedStart, start)
	}
	if !end.Equal(expectedEnd) {
		t.Errorf("expected end %v, got %v", expectedEnd, end)
	}
}

func TestFilterParquetFilesByRangeDaily(t *testing.T) {
	// LINKUSDT-1m-2026-05-01.parquet
	start, end, err := ParseDateRangeFromFilename("LINKUSDT-1m-2026-05-01.parquet")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	expectedStart := time.Date(2026, time.May, 1, 0, 0, 0, 0, time.UTC)
	expectedEnd := time.Date(2026, time.May, 2, 0, 0, 0, 0, time.UTC).Add(-time.Millisecond)

	if !start.Equal(expectedStart) {
		t.Errorf("expected start %v, got %v", expectedStart, start)
	}
	if !end.Equal(expectedEnd) {
		t.Errorf("expected end %v, got %v", expectedEnd, end)
	}
}

func TestFilterParquetFilesRejectsBadNames(t *testing.T) {
	badNames := []string{
		"LINKUSDT-1m.parquet",
		"LINKUSDT-1m-2023.parquet",
		"invalid-filename.parquet",
	}

	for _, name := range badNames {
		_, _, err := ParseDateRangeFromFilename(name)
		if err == nil {
			t.Errorf("expected error for bad name %q, got nil", name)
		}
	}
}

func TestLocalParquetSourceRejectsEmptyPath(t *testing.T) {
	src := NewLocalParquetSource()
	_, err := src.LoadCandles(context.Background(), CandleRequest{
		Path: "",
	})
	if err == nil {
		t.Error("expected error for empty path, got nil")
	} else if err.Error() != "empty path" {
		t.Errorf("expected error 'empty path', got %q", err.Error())
	}
}

func TestLocalParquetSourceRejectsNoMatches(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "test_parquet_no_matches_*")
	if err != nil {
		t.Fatalf("failed to create tmp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	src := NewLocalParquetSource()
	_, err = src.LoadCandles(context.Background(), CandleRequest{
		Path:     tmpDir,
		Market:   "futures-um",
		Symbol:   "LINKUSDT",
		Interval: "1m",
	})
	if err == nil {
		t.Error("expected error for no matches, got nil")
	}
}

func createTestParquetFile(t *testing.T, filepathStr string, rows []ParquetCandle) {
	fr, err := local.NewLocalFileWriter(filepathStr)
	if err != nil {
		t.Fatalf("failed to create file writer: %v", err)
	}
	defer fr.Close()

	pw, err := writer.NewParquetWriter(fr, new(ParquetCandle), 1)
	if err != nil {
		t.Fatalf("failed to create parquet writer: %v", err)
	}
	defer pw.WriteStop()

	for _, r := range rows {
		if err := pw.Write(r); err != nil {
			t.Fatalf("failed to write row: %v", err)
		}
	}
}

func TestLocalParquetSourceLoadsFixture(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "test_parquet_load_*")
	if err != nil {
		t.Fatalf("failed to create tmp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	market := "futures-um"
	symbol := "LINKUSDT"
	interval := "1m"

	// Create daily directories
	dirPath := filepath.Join(tmpDir, "candles", market, interval, "symbol="+symbol, "year=2023", "month=01")
	if err := os.MkdirAll(dirPath, 0755); err != nil {
		t.Fatalf("failed to create dirs: %v", err)
	}

	// Prepare rows
	mkt := "futures-um"
	sym := "LINKUSDT"
	iv := "1m"
	t1 := int64(1672531200000) // 2023-01-01 00:00:00
	c1 := int64(1672531259999)
	o1 := 10.0
	h1 := 10.5
	l1 := 9.8
	cl1 := 10.2
	v1 := 100.0

	t2 := int64(1672531260000) // 2023-01-01 00:01:00
	c2 := int64(1672531319999)
	o2 := 10.2
	h2 := 10.6
	l2 := 10.1
	cl2 := 10.5
	v2 := 150.0

	rows := []ParquetCandle{
		{
			Market:      &mkt,
			Symbol:      &sym,
			Interval:    &iv,
			OpenTimeMS:  &t1,
			Open:        &o1,
			High:        &h1,
			Low:         &l1,
			Close:       &cl1,
			Volume:      &v1,
			CloseTimeMS: &c1,
		},
		{
			Market:      &mkt,
			Symbol:      &sym,
			Interval:    &iv,
			OpenTimeMS:  &t2,
			Open:        &o2,
			High:        &h2,
			Low:         &l2,
			Close:       &cl2,
			Volume:      &v2,
			CloseTimeMS: &c2,
		},
	}

	filePath := filepath.Join(dirPath, "LINKUSDT-1m-2023-01.parquet")
	createTestParquetFile(t, filePath, rows)

	src := NewLocalParquetSource()
	req := CandleRequest{
		Path:     tmpDir,
		Market:   market,
		Symbol:   symbol,
		Interval: interval,
		From:     time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC),
		To:       time.Date(2023, 1, 1, 0, 5, 0, 0, time.UTC),
	}

	candles, err := src.LoadCandles(context.Background(), req)
	if err != nil {
		t.Fatalf("unexpected error loading candles: %v", err)
	}

	if len(candles) != 2 {
		t.Errorf("expected 2 candles, got %d", len(candles))
	}

	if candles[0].OpenTimeMS != t1 || candles[0].Open != o1 || candles[0].Close != cl1 {
		t.Errorf("first candle data mismatch: %+v", candles[0])
	}
	if candles[1].OpenTimeMS != t2 || candles[1].Open != o2 || candles[1].Close != cl2 {
		t.Errorf("second candle data mismatch: %+v", candles[1])
	}
}
