package data

import (
	"testing"
)

func TestSourceFactory(t *testing.T) {
	// 1. Supported sources
	jsonSrc, err := NewCandleSource("local-json")
	if err != nil {
		t.Fatalf("failed to create local-json source: %v", err)
	}
	if jsonSrc.Name() != "local-json" {
		t.Errorf("expected source name 'local-json', got %q", jsonSrc.Name())
	}

	parquetSrc, err := NewCandleSource("local-parquet")
	if err != nil {
		t.Fatalf("failed to create local-parquet source: %v", err)
	}
	if parquetSrc.Name() != "local-parquet" {
		t.Errorf("expected source name 'local-parquet', got %q", parquetSrc.Name())
	}

	r2Src, err := NewCandleSource("r2")
	if err != nil {
		t.Fatalf("failed to create r2 source: %v", err)
	}
	if r2Src.Name() != "r2" {
		t.Errorf("expected source name 'r2', got %q", r2Src.Name())
	}

	// 2. Reject unsupported source with clear error listing supported options
	_, err = NewCandleSource("unsupported")
	if err == nil {
		t.Error("expected error for unsupported source, got nil")
	} else {
		expectedMsg := `unknown source "unsupported"; supported sources are: local-json, local-parquet, r2`
		if err.Error() != expectedMsg {
			t.Errorf("expected error %q, got %q", expectedMsg, err.Error())
		}
	}
}
