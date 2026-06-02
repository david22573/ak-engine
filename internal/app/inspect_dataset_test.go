package app

import (
	"bytes"
	"io"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/xitongsys/parquet-go-source/local"
	pqsource "github.com/xitongsys/parquet-go/source"
	"github.com/xitongsys/parquet-go/writer"
)

func TestInspectDatasetLocalJSONSuccess(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "test_inspect_*")
	if err != nil {
		t.Fatalf("failed to create tmp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	jsonContent := `[
		{
			"open_time_ms": 1704067200000,
			"open": 42000.0, "high": 42500.0, "low": 41800.0, "close": 42200.0,
			"volume": 100.0, "close_time_ms": 1704067499999, "interval": "5m"
		},
		{
			"open_time_ms": 1704067500000,
			"open": 42200.0, "high": 42600.0, "low": 42100.0, "close": 42400.0,
			"volume": 120.0, "close_time_ms": 1704067799999, "interval": "5m"
		}
	]`

	filePath := filepath.Join(tmpDir, "valid.json")
	if err := os.WriteFile(filePath, []byte(jsonContent), 0644); err != nil {
		t.Fatalf("failed to write file: %v", err)
	}

	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	rootCmd.SetArgs([]string{
		"inspect-dataset",
		"--source", "local-json",
		"--path", filePath,
		"--market", "futures-um",
		"--symbol", "BTCUSDT",
		"--interval", "5m",
		"--format", "json",
	})

	err = rootCmd.Execute()

	w.Close()
	os.Stdout = oldStdout

	var buf bytes.Buffer
	io.Copy(&buf, r)
	output := buf.String()

	if err != nil {
		t.Fatalf("expected no error, got: %v. Output: %s", err, output)
	}

	if !bytes.Contains(buf.Bytes(), []byte(`"status": "PASS"`)) {
		t.Errorf("expected output to contain PASS status, got: %s", output)
	}
}

func TestInspectDatasetLocalJSONFailure(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "test_inspect_*")
	if err != nil {
		t.Fatalf("failed to create tmp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	jsonContent := `[
		{
			"open_time_ms": 1704067200000,
			"open": 42000.0, "high": 42500.0, "low": 41800.0, "close": 42200.0,
			"volume": 100.0, "close_time_ms": 1704067499999, "interval": "5m"
		},
		{
			"open_time_ms": 1704067200000,
			"open": 42200.0, "high": 42600.0, "low": 42100.0, "close": 42400.0,
			"volume": 120.0, "close_time_ms": 1704067799999, "interval": "5m"
		}
	]`

	filePath := filepath.Join(tmpDir, "invalid.json")
	if err := os.WriteFile(filePath, []byte(jsonContent), 0644); err != nil {
		t.Fatalf("failed to write file: %v", err)
	}

	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	rootCmd.SetArgs([]string{
		"inspect-dataset",
		"--source", "local-json",
		"--path", filePath,
		"--market", "futures-um",
		"--symbol", "BTCUSDT",
		"--interval", "5m",
		"--format", "json",
	})

	err = rootCmd.Execute()

	w.Close()
	os.Stdout = oldStdout

	var buf bytes.Buffer
	io.Copy(&buf, r)
	output := buf.String()

	if err == nil {
		t.Fatal("expected validation error, got nil")
	}

	if !bytes.Contains(buf.Bytes(), []byte(`"status": "FAIL"`)) {
		t.Errorf("expected output to contain FAIL status, got: %s", output)
	}
}

func TestInspectDatasetLocalParquetPassesWithFixture(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "test_inspect_parquet_*")
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

	// We can use a map or struct to write it. We import data so we can use data.ParquetCandle
	rows := []struct {
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
	}{
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

	// Create local writer
	// Directly use xitongsys imports in test code
	fr, err := local_writer(filePath)
	if err != nil {
		t.Fatalf("failed to create writer: %v", err)
	}
	defer fr.Close()

	pw, err := parquet_writer(fr, &rows[0])
	if err != nil {
		t.Fatalf("failed to create parquet writer: %v", err)
	}

	for _, r := range rows {
		if err := pw.Write(r); err != nil {
			t.Fatalf("write failed: %v", err)
		}
	}
	pw.WriteStop()

	oldStdout := os.Stdout
	rChan, w, _ := os.Pipe()
	os.Stdout = w

	rootCmd.SetArgs([]string{
		"inspect-dataset",
		"--source", "local-parquet",
		"--path", tmpDir,
		"--market", market,
		"--symbol", symbol,
		"--interval", interval,
		"--from", "2023-01-01",
		"--to", "2023-01-02",
		"--format", "json",
	})

	err = rootCmd.Execute()

	w.Close()
	os.Stdout = oldStdout

	var buf bytes.Buffer
	io.Copy(&buf, rChan)
	output := buf.String()

	if err != nil {
		t.Fatalf("expected no error, got: %v. Output: %s", err, output)
	}

	if !bytes.Contains(buf.Bytes(), []byte(`"status": "PASS"`)) {
		t.Errorf("expected output to contain PASS status, got: %s", output)
	}
	if !bytes.Contains(buf.Bytes(), []byte(`"count": 2`)) {
		t.Errorf("expected count to be 2, got: %s", output)
	}
}

// Helpers to avoid import block issues
func local_writer(path string) (pqsource.ParquetFile, error) {
	return local.NewLocalFileWriter(path)
}

func parquet_writer(w pqsource.ParquetFile, obj interface{}) (*writer.ParquetWriter, error) {
	return writer.NewParquetWriter(w, obj, 1)
}

func TestDateBoundaryParsing(t *testing.T) {
	// Test date-only format
	fromT, err := parseFromTime("2023-01-01")
	if err != nil {
		t.Fatalf("unexpected parseFromTime error: %v", err)
	}
	expectedFrom := time.Date(2023, time.January, 1, 0, 0, 0, 0, time.UTC)
	if !fromT.Equal(expectedFrom) {
		t.Errorf("expected from %v, got %v", expectedFrom, fromT)
	}

	toT, err := parseToTime("2023-01-31")
	if err != nil {
		t.Fatalf("unexpected parseToTime error: %v", err)
	}
	expectedTo := time.Date(2023, time.January, 31, 23, 59, 59, 999000000, time.UTC)
	if !toT.Equal(expectedTo) {
		t.Errorf("expected to %v, got %v", expectedTo, toT)
	}

	// Test RFC3339 format
	fromRFC, err := parseFromTime("2023-01-01T12:00:00Z")
	if err != nil {
		t.Fatalf("unexpected parseFromTime RFC error: %v", err)
	}
	expectedFromRFC := time.Date(2023, time.January, 1, 12, 0, 0, 0, time.UTC)
	if !fromRFC.Equal(expectedFromRFC) {
		t.Errorf("expected from RFC %v, got %v", expectedFromRFC, fromRFC)
	}

	toRFC, err := parseToTime("2023-01-31T12:00:00Z")
	if err != nil {
		t.Fatalf("unexpected parseToTime RFC error: %v", err)
	}
	expectedToRFC := time.Date(2023, time.January, 31, 12, 0, 0, 0, time.UTC)
	if !toRFC.Equal(expectedToRFC) {
		t.Errorf("expected to RFC %v, got %v", expectedToRFC, toRFC)
	}
}
