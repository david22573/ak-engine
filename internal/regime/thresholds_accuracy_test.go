package regime

import (
	"math"
	"math/rand"
	"sort"
	"testing"

	"github.com/davidmiguel22573/ak-engine/internal/features"
)

func exactThresholds(rows []features.Row, idx int, lookback int) Thresholds {
	start := idx - lookback
	if start < 0 { start = 0 }
	var atr, bb, vol []float64
	for i := start; i < idx; i++ {
		r := &rows[i]
		if !r.Warmup {
			atr = append(atr, r.ATRPct14)
			bb = append(bb, r.BBWidth20)
			vol = append(vol, r.VolumeRatio20)
		}
	}
	sort.Float64s(atr)
	sort.Float64s(bb)
	sort.Float64s(vol)
	return Thresholds{
		ATRPctP20:      percentile(atr, 0.20),
		ATRPctP80:      percentile(atr, 0.80),
		ATRPctP95:      percentile(atr, 0.95),
		BBWidthP20:     percentile(bb, 0.20),
		BBWidthP80:     percentile(bb, 0.80),
		VolumeRatioP20: percentile(vol, 0.20),
		VolumeRatioP80: percentile(vol, 0.80),
		VolumeRatioP95: percentile(vol, 0.95),
	}
}

func TestComputeTrailingThresholds_SubsamplingAccuracy(t *testing.T) {
	rand.Seed(42)
	// Create 50,000 rows
	n := 50000
	rows := make([]features.Row, n)
	for i := 0; i < n; i++ {
		rows[i] = features.Row{
			Interval:      "1m",
			Warmup:        false,
			ATRPct14:      rand.Float64() * 0.05,
			BBWidth20:     rand.Float64() * 0.1,
			VolumeRatio20: 0.5 + rand.Float64() * 2.0,
		}
	}

	opts := ThresholdOptions{
		LookbackRows: 43200,
		MinRows:      50,
	}

	idx := 45000
	subsampled, ok := ComputeTrailingThresholds(rows, idx, opts)
	if !ok {
		t.Fatal("failed")
	}
	exact := exactThresholds(rows, idx, 43200)

	maxErr := 0.0
	compare := func(name string, ex, sub float64) {
		err := math.Abs(ex - sub) / ex
		if err > maxErr {
			maxErr = err
		}
		t.Logf("%s: exact=%f subsampled=%f err=%f%%", name, ex, sub, err*100)
	}

	compare("ATRPctP20", exact.ATRPctP20, subsampled.ATRPctP20)
	compare("ATRPctP80", exact.ATRPctP80, subsampled.ATRPctP80)
	compare("ATRPctP95", exact.ATRPctP95, subsampled.ATRPctP95)

	t.Logf("Max observed error: %f%%", maxErr*100)
}
