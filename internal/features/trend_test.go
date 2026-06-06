package features

import "testing"

func TestEMA_ShortRows(t *testing.T) {
	// Should not panic, should return slice of zeroes
	res := EMA([]float64{1.0, 2.0}, 20)
	if len(res) != 2 {
		t.Errorf("expected length 2, got %d", len(res))
	}
	if res[0] != 0 || res[1] != 0 {
		t.Errorf("expected zeroes, got %v", res)
	}
}

func TestEMA_Normal(t *testing.T) {
	values := []float64{10, 10, 10, 10, 10}
	res := EMA(values, 3)
	// For period=3, k = 2 / (3+1) = 0.5.
	// First 2 elements (indices 0, 1) are 0.
	// index 2 is SMA of first 3 elements: (10+10+10)/3 = 10.
	// index 3: values[3]*0.5 + result[2]*0.5 = 10*0.5 + 10*0.5 = 10.
	if res[0] != 0 || res[1] != 0 {
		t.Errorf("expected index 0 and 1 to be 0, got %v", res)
	}
	if res[2] != 10 {
		t.Errorf("expected index 2 to be 10, got %f", res[2])
	}
	if res[3] != 10 {
		t.Errorf("expected index 3 to be 10, got %f", res[3])
	}
}

func TestLinearSlope(t *testing.T) {
	values := []float64{1, 2, 3, 4, 5}
	slope, ok := LinearSlope(values, 4, 5)
	if !ok {
		t.Error("expected LinearSlope to succeed")
	}
	// values are exactly 1, 2, 3, 4, 5 (linear increase by 1 per step)
	if slope != 1.0 {
		t.Errorf("expected slope 1.0, got %f", slope)
	}

	// Insufficient data
	_, ok = LinearSlope(values, 4, 10)
	if ok {
		t.Error("expected LinearSlope to return false for insufficient lookback")
	}
}
