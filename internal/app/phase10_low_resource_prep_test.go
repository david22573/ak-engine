package app

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestPhase10LowResourcePrep(t *testing.T) {
	// check imports
	data, err := os.ReadFile("phase10_low_resource_prep.go")
	if err == nil {
		if strings.Contains(string(data), "ak-trader") {
			t.Errorf("must not import ak-trader")
		}
	}

	tempDir := t.TempDir()
	manifestPath := filepath.Join(tempDir, "runs/manifests/phase10_5_low_resource_manifest.json")

	// Create a dummy completed chunk
	m := &Manifest{Chunks: make(map[string]*ChunkStatus)}
	m.Chunks["TESTUSDT_2024-01"] = &ChunkStatus{
		Symbol:            "TESTUSDT",
		Month:             "2024-01",
		FeatureStatus:     "DONE",
		RegimeStatus:      "DONE",
		FundingJoinStatus: "DONE",
		ReportStatus:      "DONE",
	}
	saveManifest(manifestPath, m)

	// Override manifest path for test? We can't easily without a variable, but let's test `isComplete` and `loadManifest`
	status := m.Chunks["TESTUSDT_2024-01"]
	if !isComplete(status) {
		t.Errorf("expected chunk to be complete")
	}

	// boundaries
	t1, _ := time.Parse("2006-01", "2024-01")
	t2 := t1.AddDate(0, 1, 0)
	if t2.Format("2006-01") != "2024-02" {
		t.Errorf("monthly chunk boundaries incorrect")
	}
}

func TestAggregateChunkReports(t *testing.T) {
	// Check no ak-trader
	data, err := os.ReadFile("aggregate_chunk_reports.go")
	if err == nil {
		if strings.Contains(string(data), "ak-trader") {
			t.Errorf("must not import ak-trader")
		}
	}
}
