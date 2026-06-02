package data

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"
)

type FakeS3Downloader struct {
	DownloadFunc func(ctx context.Context, key string, destPath string) error
}

func (f *FakeS3Downloader) Download(ctx context.Context, key string, destPath string) error {
	if f.DownloadFunc != nil {
		return f.DownloadFunc(ctx, key, destPath)
	}
	return nil
}

func TestManifestRejectsNonPassStatus(t *testing.T) {
	manifestBytes := []byte(`{
		"schema_version": 1,
		"market": "futures-um",
		"symbol": "LINKUSDT",
		"interval": "1m",
		"status": "FAILED",
		"objects": []
	}`)

	_, err := ParseManifest(manifestBytes)
	if err == nil {
		t.Fatal("expected parse error for non-PASS manifest, got nil")
	}
	if err.Error() != "manifest status is not PASS: FAILED" {
		t.Fatalf("unexpected parse error: %v", err)
	}

	// Verify LoadCandles rejects non-PASS manifest
	tmpDir, err := os.MkdirTemp("", "test_r2_manifest_reject_*")
	if err != nil {
		t.Fatalf("failed to create tmp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	manifestKey := "manifests/futures-um/1m/symbol=LINKUSDT/manifest.json"
	manifestPath := filepath.Join(tmpDir, manifestKey)
	if err := os.MkdirAll(filepath.Dir(manifestPath), 0755); err != nil {
		t.Fatalf("failed to create dirs: %v", err)
	}
	if err := os.WriteFile(manifestPath, manifestBytes, 0644); err != nil {
		t.Fatalf("failed to write manifest: %v", err)
	}

	src := &R2ParquetSource{
		downloader: &FakeS3Downloader{},
		cacheDir:   tmpDir,
	}

	_, err = src.LoadCandles(context.Background(), CandleRequest{
		Market:   "futures-um",
		Symbol:   "LINKUSDT",
		Interval: "1m",
		From:     time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC),
		To:       time.Date(2023, 1, 31, 23, 59, 59, 0, time.UTC),
	})

	if err == nil {
		t.Error("expected error for non-PASS manifest status, got nil")
	} else if err.Error() != "manifest status is not PASS: FAILED" {
		t.Errorf("expected error 'manifest status is not PASS: FAILED', got %q", err.Error())
	}
}

func TestManifestRejectsEmptyObjects(t *testing.T) {
	manifestBytes := []byte(`{
		"schema_version": 1,
		"market": "futures-um",
		"symbol": "LINKUSDT",
		"interval": "1m",
		"status": "PASS",
		"objects": []
	}`)

	_, err := ParseManifest(manifestBytes)
	if err == nil {
		t.Fatal("expected parse error for empty objects, got nil")
	}
	if err.Error() != "manifest has no objects" {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestManifestFiltersObjectsByMinMaxRange(t *testing.T) {
	src := NewR2ParquetSource()
	objects := []ObjectStats{
		{
			Key:           "obj1.parquet",
			MinOpenTimeMS: time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC).UnixMilli(),
			MaxOpenTimeMS: time.Date(2023, 1, 15, 23, 59, 59, 0, time.UTC).UnixMilli(),
		},
		{
			Key:           "obj2.parquet",
			MinOpenTimeMS: time.Date(2023, 1, 16, 0, 0, 0, 0, time.UTC).UnixMilli(),
			MaxOpenTimeMS: time.Date(2023, 1, 31, 23, 59, 59, 0, time.UTC).UnixMilli(),
		},
		{
			Key:           "obj3.parquet",
			MinOpenTimeMS: time.Date(2023, 2, 1, 0, 0, 0, 0, time.UTC).UnixMilli(),
			MaxOpenTimeMS: time.Date(2023, 2, 15, 23, 59, 59, 0, time.UTC).UnixMilli(),
		},
	}

	req := CandleRequest{
		From: time.Date(2023, 1, 10, 0, 0, 0, 0, time.UTC),
		To:   time.Date(2023, 1, 20, 23, 59, 59, 0, time.UTC),
	}

	matched, err := src.FilterObjects(objects, req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(matched) != 2 {
		t.Fatalf("expected 2 matched objects, got %d", len(matched))
	}
	if matched[0].Key != "obj1.parquet" || matched[1].Key != "obj2.parquet" {
		t.Errorf("matched objects mismatch: %+v", matched)
	}
}

func TestManifestFiltersObjectsByFilenameMonthly(t *testing.T) {
	src := NewR2ParquetSource()
	objects := []ObjectStats{
		{Key: "candles/futures-um/1m/symbol=LINKUSDT/year=2023/month=01/LINKUSDT-1m-2023-01.parquet"},
		{Key: "candles/futures-um/1m/symbol=LINKUSDT/year=2023/month=02/LINKUSDT-1m-2023-02.parquet"},
		{Key: "candles/futures-um/1m/symbol=LINKUSDT/year=2023/month=03/LINKUSDT-1m-2023-03.parquet"},
	}

	req := CandleRequest{
		From: time.Date(2023, 1, 15, 0, 0, 0, 0, time.UTC),
		To:   time.Date(2023, 2, 15, 23, 59, 59, 0, time.UTC),
	}

	matched, err := src.FilterObjects(objects, req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(matched) != 2 {
		t.Fatalf("expected 2 matched objects, got %d", len(matched))
	}
	if matched[0].Key != objects[0].Key || matched[1].Key != objects[1].Key {
		t.Errorf("matched objects mismatch: %+v", matched)
	}
}

func TestManifestFiltersObjectsByFilenameDaily(t *testing.T) {
	src := NewR2ParquetSource()
	objects := []ObjectStats{
		{Key: "candles/futures-um/1m/symbol=LINKUSDT/year=2023/month=01/LINKUSDT-1m-2023-01-01.parquet"},
		{Key: "candles/futures-um/1m/symbol=LINKUSDT/year=2023/month=01/LINKUSDT-1m-2023-01-02.parquet"},
		{Key: "candles/futures-um/1m/symbol=LINKUSDT/year=2023/month=01/LINKUSDT-1m-2023-01-03.parquet"},
	}

	req := CandleRequest{
		From: time.Date(2023, 1, 2, 0, 0, 0, 0, time.UTC),
		To:   time.Date(2023, 1, 2, 23, 59, 59, 0, time.UTC),
	}

	matched, err := src.FilterObjects(objects, req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(matched) != 1 {
		t.Fatalf("expected 1 matched object, got %d", len(matched))
	}
	if matched[0].Key != objects[1].Key {
		t.Errorf("expected matched object %q, got %q", objects[1].Key, matched[0].Key)
	}
}

func TestR2ConfigRejectsMissingEnv(t *testing.T) {
	// Backup original env vars
	origAccountID := os.Getenv("R2_ACCOUNT_ID")
	origAccessKeyID := os.Getenv("R2_ACCESS_KEY_ID")
	origSecretAccessKey := os.Getenv("R2_SECRET_ACCESS_KEY")
	origBucketName := os.Getenv("R2_BUCKET_NAME")

	defer func() {
		os.Setenv("R2_ACCOUNT_ID", origAccountID)
		os.Setenv("R2_ACCESS_KEY_ID", origAccessKeyID)
		os.Setenv("R2_SECRET_ACCESS_KEY", origSecretAccessKey)
		os.Setenv("R2_BUCKET_NAME", origBucketName)
	}()

	// Clear env
	os.Unsetenv("R2_ACCOUNT_ID")
	os.Unsetenv("R2_ACCESS_KEY_ID")
	os.Unsetenv("R2_SECRET_ACCESS_KEY")
	os.Unsetenv("R2_BUCKET_NAME")

	_, err := LoadR2Config()
	if err == nil {
		t.Fatal("expected error from missing R2 config, got nil")
	}
}

func TestR2CachePathPreservesObjectKey(t *testing.T) {
	cacheDir := "/tmp/r2_cache"
	objectKey := "candles/futures-um/1m/symbol=LINKUSDT/year=2023/month=01/LINKUSDT-1m-2023-01.parquet"

	expected := filepath.Join(cacheDir, objectKey)
	actual := filepath.Join(cacheDir, objectKey)

	if actual != expected {
		t.Errorf("cache path mismatch: expected %q, got %q", expected, actual)
	}

	// Verify it does not flatten
	if filepath.Base(actual) == objectKey {
		t.Errorf("objectKey was somehow flattened or unmodified: %q", actual)
	}
}

func TestR2SourceRejectsNoMatchingObjects(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "test_r2_no_match_*")
	if err != nil {
		t.Fatalf("failed to create tmp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	manifestBytes := []byte(`{
		"schema_version": 1,
		"market": "futures-um",
		"symbol": "LINKUSDT",
		"interval": "1m",
		"status": "PASS",
		"objects": [
			{
				"key": "LINKUSDT-1m-2023-01.parquet",
				"row_count": 44640,
				"min_open_time_ms": 1672531200000,
				"max_open_time_ms": 1675209540000
			}
		]
	}`)

	manifestKey := "manifests/futures-um/1m/symbol=LINKUSDT/manifest.json"
	manifestPath := filepath.Join(tmpDir, manifestKey)
	if err := os.MkdirAll(filepath.Dir(manifestPath), 0755); err != nil {
		t.Fatalf("failed to create dirs: %v", err)
	}
	if err := os.WriteFile(manifestPath, manifestBytes, 0644); err != nil {
		t.Fatalf("failed to write manifest: %v", err)
	}

	src := &R2ParquetSource{
		downloader: &FakeS3Downloader{},
		cacheDir:   tmpDir,
	}

	// Range way outside manifest objects
	_, err = src.LoadCandles(context.Background(), CandleRequest{
		Market:   "futures-um",
		Symbol:   "LINKUSDT",
		Interval: "1m",
		From:     time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
		To:       time.Date(2024, 1, 31, 23, 59, 59, 0, time.UTC),
	})

	if err == nil {
		t.Error("expected error for no matching objects, got nil")
	} else if err.Error() != "no matching objects found for the requested range" {
		t.Errorf("expected error 'no matching objects found for the requested range', got %q", err.Error())
	}
}

func TestR2SourceRejectsManifestRequestMismatch(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "test_r2_manifest_mismatch_*")
	if err != nil {
		t.Fatalf("failed to create tmp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	manifestBytes := []byte(`{
		"schema_version": 1,
		"market": "futures-um",
		"symbol": "BTCUSDT",
		"interval": "1m",
		"status": "PASS",
		"objects": [
			{
				"key": "candles/futures-um/1m/symbol=BTCUSDT/year=2023/month=01/BTCUSDT-1m-2023-01.parquet",
				"row_count": 44640,
				"min_open_time_ms": 1672531200000,
				"max_open_time_ms": 1675209540000
			}
		]
	}`)

	manifestKey := "manifests/futures-um/1m/symbol=LINKUSDT/manifest.json"
	manifestPath := filepath.Join(tmpDir, manifestKey)
	if err := os.MkdirAll(filepath.Dir(manifestPath), 0755); err != nil {
		t.Fatalf("failed to create dirs: %v", err)
	}
	if err := os.WriteFile(manifestPath, manifestBytes, 0644); err != nil {
		t.Fatalf("failed to write manifest: %v", err)
	}

	src := &R2ParquetSource{
		downloader: &FakeS3Downloader{},
		cacheDir:   tmpDir,
	}

	_, err = src.LoadCandles(context.Background(), CandleRequest{
		Market:   "futures-um",
		Symbol:   "LINKUSDT",
		Interval: "1m",
		From:     time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC),
		To:       time.Date(2023, 1, 31, 23, 59, 59, 0, time.UTC),
	})

	if err == nil {
		t.Fatal("expected manifest/request mismatch error, got nil")
	}
	expected := `manifest symbol "BTCUSDT" does not match request symbol "LINKUSDT"`
	if err.Error() != expected {
		t.Fatalf("expected %q, got %q", expected, err.Error())
	}
}

func TestR2SourceLoadsCachedParquetFixture(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "test_r2_cached_load_*")
	if err != nil {
		t.Fatalf("failed to create tmp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	manifestBytes := []byte(`{
		"schema_version": 1,
		"market": "futures-um",
		"symbol": "LINKUSDT",
		"interval": "1m",
		"status": "PASS",
		"objects": [
			{
				"key": "candles/futures-um/1m/symbol=LINKUSDT/year=2023/month=01/LINKUSDT-1m-2023-01.parquet",
				"row_count": 2,
				"min_open_time_ms": 1672531200000,
				"max_open_time_ms": 1672531260000
			}
		]
	}`)

	manifestKey := "manifests/futures-um/1m/symbol=LINKUSDT/manifest.json"
	manifestPath := filepath.Join(tmpDir, manifestKey)
	if err := os.MkdirAll(filepath.Dir(manifestPath), 0755); err != nil {
		t.Fatalf("failed to create dirs: %v", err)
	}
	if err := os.WriteFile(manifestPath, manifestBytes, 0644); err != nil {
		t.Fatalf("failed to write manifest: %v", err)
	}

	// Write cached parquet file directly
	parquetKey := "candles/futures-um/1m/symbol=LINKUSDT/year=2023/month=01/LINKUSDT-1m-2023-01.parquet"
	parquetPath := filepath.Join(tmpDir, parquetKey)
	if err := os.MkdirAll(filepath.Dir(parquetPath), 0755); err != nil {
		t.Fatalf("failed to create parquet dirs: %v", err)
	}

	// Write candles
	mkt := "futures-um"
	sym := "LINKUSDT"
	iv := "1m"
	t1 := int64(1672531200000)
	c1 := int64(1672531259999)
	o1 := 10.0
	h1 := 10.5
	l1 := 9.8
	cl1 := 10.2
	v1 := 100.0

	t2 := int64(1672531260000)
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

	createTestParquetFile(t, parquetPath, rows)

	// Since files exist, downloader should not be called
	downloaderCalled := false
	src := &R2ParquetSource{
		downloader: &FakeS3Downloader{
			DownloadFunc: func(ctx context.Context, key string, destPath string) error {
				downloaderCalled = true
				return nil
			},
		},
		cacheDir: tmpDir,
	}

	candles, err := src.LoadCandles(context.Background(), CandleRequest{
		Market:   "futures-um",
		Symbol:   "LINKUSDT",
		Interval: "1m",
		From:     time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC),
		To:       time.Date(2023, 1, 2, 0, 0, 0, 0, time.UTC),
	})

	if err != nil {
		t.Fatalf("unexpected load error: %v", err)
	}

	if downloaderCalled {
		t.Error("expected downloader NOT to be called because files were already cached")
	}

	if len(candles) != 2 {
		t.Errorf("expected 2 candles, got %d", len(candles))
	}
}

func TestR2SourceDownloadsThenLoadsParquetFixture(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "test_r2_download_load_*")
	if err != nil {
		t.Fatalf("failed to create tmp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	manifestBytes := []byte(`{
		"schema_version": 1,
		"market": "futures-um",
		"symbol": "LINKUSDT",
		"interval": "1m",
		"status": "PASS",
		"objects": [
			{
				"key": "candles/futures-um/1m/symbol=LINKUSDT/year=2023/month=01/LINKUSDT-1m-2023-01.parquet",
				"row_count": 2,
				"min_open_time_ms": 1672531200000,
				"max_open_time_ms": 1672531260000
			}
		]
	}`)

	parquetKey := "candles/futures-um/1m/symbol=LINKUSDT/year=2023/month=01/LINKUSDT-1m-2023-01.parquet"

	// Create fake downloader that writes the files to local path when called
	src := &R2ParquetSource{
		downloader: &FakeS3Downloader{
			DownloadFunc: func(ctx context.Context, key string, destPath string) error {
				if err := os.MkdirAll(filepath.Dir(destPath), 0755); err != nil {
					return err
				}
				if key == "manifests/futures-um/1m/symbol=LINKUSDT/manifest.json" {
					return os.WriteFile(destPath, manifestBytes, 0644)
				}
				if key == parquetKey {
					mkt := "futures-um"
					sym := "LINKUSDT"
					iv := "1m"
					t1 := int64(1672531200000)
					c1 := int64(1672531259999)
					o1 := 10.0
					h1 := 10.5
					l1 := 9.8
					cl1 := 10.2
					v1 := 100.0

					t2 := int64(1672531260000)
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
					createTestParquetFile(t, destPath, rows)
					return nil
				}
				return fmt.Errorf("unexpected download key: %s", key)
			},
		},
		cacheDir: tmpDir,
	}

	candles, err := src.LoadCandles(context.Background(), CandleRequest{
		Market:   "futures-um",
		Symbol:   "LINKUSDT",
		Interval: "1m",
		From:     time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC),
		To:       time.Date(2023, 1, 2, 0, 0, 0, 0, time.UTC),
	})

	if err != nil {
		t.Fatalf("unexpected load error: %v", err)
	}

	if len(candles) != 2 {
		t.Errorf("expected 2 candles, got %d", len(candles))
	}
}

func TestR2Integration(t *testing.T) {
	if os.Getenv("AK_ENGINE_R2_INTEGRATION") != "1" {
		t.Skip("skipping live R2 integration test")
	}

	src := NewR2ParquetSource()
	_, err := src.LoadCandles(context.Background(), CandleRequest{
		Market:   "futures-um",
		Symbol:   "LINKUSDT",
		Interval: "1m",
		From:     time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC),
		To:       time.Date(2023, 1, 31, 23, 59, 59, 0, time.UTC),
	})
	if err != nil {
		t.Fatalf("failed to load candles from R2: %v", err)
	}
}
