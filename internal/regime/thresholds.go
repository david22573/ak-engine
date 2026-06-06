package regime

import (
	"sort"
	"sync"

	"github.com/davidmiguel22573/ak-engine/internal/features"
)

type Thresholds struct {
	ATRPctP20      float64 `json:"atr_pct_p20"`
	ATRPctP80      float64 `json:"atr_pct_p80"`
	ATRPctP95      float64 `json:"atr_pct_p95"`
	BBWidthP20     float64 `json:"bb_width_p20"`
	BBWidthP80     float64 `json:"bb_width_p80"`
	VolumeRatioP20 float64 `json:"volume_ratio_p20"`
	VolumeRatioP80 float64 `json:"volume_ratio_p80"`
	VolumeRatioP95 float64 `json:"volume_ratio_p95"`
}

type ThresholdOptions struct {
	LookbackRows int
	MinRows      int
}

var atrPool = sync.Pool{New: func() interface{} { return make([]float64, 0, 45000) }}
var bbPool = sync.Pool{New: func() interface{} { return make([]float64, 0, 45000) }}
var volPool = sync.Pool{New: func() interface{} { return make([]float64, 0, 45000) }}

func ComputeTrailingThresholds(rows []features.Row, idx int, opts ThresholdOptions) (Thresholds, bool) {
	if idx <= 0 || idx > len(rows) {
		return Thresholds{}, false
	}

	lookback := opts.LookbackRows
	if lookback <= 0 {
		// Detect interval from rows
		interval := "1m"
		if len(rows) > 0 {
			interval = rows[0].Interval
		}
		switch interval {
		case "1m":
			lookback = 43200
		case "5m":
			lookback = 8640
		case "15m":
			lookback = 2880
		case "30m":
			lookback = 1440
		case "1h":
			lookback = 720
		default:
			lookback = 43200
		}
	}

	minRows := opts.MinRows
	if minRows <= 0 {
		calcMin := lookback / 4
		if calcMin < 200 {
			minRows = 200
		} else {
			minRows = calcMin
		}
	}

	start := idx - lookback
	if start < 0 {
		start = 0
	}

	// Gather non-warmup values
	atrPcts := atrPool.Get().([]float64)[:0]
	bbWidths := bbPool.Get().([]float64)[:0]
	volRatios := volPool.Get().([]float64)[:0]

	defer func() {
		atrPool.Put(atrPcts)
		bbPool.Put(bbWidths)
		volPool.Put(volRatios)
	}()

	step := 1
	nRows := idx - start
	if nRows > 1000 {
		step = nRows / 1000
	}

	for i := start; i < idx; i += step {
		r := &rows[i]
		if r.Warmup {
			continue
		}
		atrPcts = append(atrPcts, r.ATRPct14)
		bbWidths = append(bbWidths, r.BBWidth20)
		volRatios = append(volRatios, r.VolumeRatio20)
	}

	if len(atrPcts) < minRows {
		return Thresholds{}, false
	}

	sort.Float64s(atrPcts)
	sort.Float64s(bbWidths)
	sort.Float64s(volRatios)

	t := Thresholds{
		ATRPctP20:      percentile(atrPcts, 0.20),
		ATRPctP80:      percentile(atrPcts, 0.80),
		ATRPctP95:      percentile(atrPcts, 0.95),
		BBWidthP20:     percentile(bbWidths, 0.20),
		BBWidthP80:     percentile(bbWidths, 0.80),
		VolumeRatioP20: percentile(volRatios, 0.20),
		VolumeRatioP80: percentile(volRatios, 0.80),
		VolumeRatioP95: percentile(volRatios, 0.95),
	}

	return t, true
}

func percentile(sortedValues []float64, pct float64) float64 {
	n := len(sortedValues)
	if n == 0 {
		return 0
	}
	if n == 1 {
		return sortedValues[0]
	}
	pos := pct * float64(n-1)
	idx := int(pos)
	diff := pos - float64(idx)
	if idx >= n-1 {
		return sortedValues[n-1]
	}
	return sortedValues[idx]*(1.0-diff) + sortedValues[idx+1]*diff
}
