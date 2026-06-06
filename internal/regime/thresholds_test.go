package regime

import (
	"testing"

	"github.com/davidmiguel22573/ak-engine/internal/features"
)

func TestComputeTrailingThresholds_ExcludesCurrentAndFuture(t *testing.T) {
	// Create 300 mock feature rows (non-warmup)
	var rows []features.Row
	for i := 0; i < 300; i++ {
		rows = append(rows, features.Row{
			Interval:      "1m",
			Warmup:        false,
			ATRPct14:      0.01,
			BBWidth20:     0.02,
			VolumeRatio20: 1.0,
		})
	}

	opts := ThresholdOptions{
		LookbackRows: 250,
		MinRows:      50,
	}

	// Base thresholds at index 260 (uses rows 10 to 259)
	t1, ok := ComputeTrailingThresholds(rows, 260, opts)
	if !ok {
		t.Fatal("failed to compute thresholds")
	}

	// Now modify row 260 (current row) and row 261 (future row)
	rows[260].ATRPct14 = 999.0      // current
	rows[261].VolumeRatio20 = 999.0 // future

	t2, ok := ComputeTrailingThresholds(rows, 260, opts)
	if !ok {
		t.Fatal("failed to compute thresholds")
	}

	// Thresholds t1 and t2 should be identical because current and future rows are excluded
	if t1.ATRPctP20 != t2.ATRPctP20 ||
		t1.ATRPctP95 != t2.ATRPctP95 ||
		t1.VolumeRatioP95 != t2.VolumeRatioP95 {
		t.Error("leakage detected: current or future row affected the trailing thresholds")
	}
}
