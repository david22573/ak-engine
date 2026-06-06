package regime

import (
	"testing"

	"github.com/davidmiguel22573/ak-engine/internal/features"
)

func TestClassifier_ShockAndCompressed(t *testing.T) {
	// Build enough historical rows to compute thresholds
	var rows []features.Row
	// We want p95 for ATRPct14 to be around 0.05, p20 around 0.01
	// p95 for VolumeRatio20 around 5.0, p20 around 0.5
	for i := 0; i < 250; i++ {
		val := 0.02
		vol := 1.0
		if i%10 == 0 {
			val = 0.05 // high values
			vol = 5.0
		} else if i%10 == 1 {
			val = 0.01 // low values
			vol = 0.5
		}
		rows = append(rows, features.Row{
			Interval:      "1m",
			Warmup:        false,
			ATRPct14:      val,
			BBWidth20:     val, // BBWidth matches ATR for simplicity
			VolumeRatio20: vol,
			Close:         100.0,
			EMA20:         100.0,
			EMA50:         100.0,
		})
	}

	opts := ThresholdOptions{
		LookbackRows: 250,
		MinRows:      100,
	}
	classifier := NewClassifier(opts)

	// Add shock test row at index 248
	rows[248].ATRPct14 = 0.10      // extreme ATR
	rows[248].VolumeRatio20 = 10.0 // extreme volume ratio
	lblShock, err := classifier.ClassifyOne(rows, 248)
	if err != nil {
		t.Fatalf("failed to classify: %v", err)
	}
	if lblShock.Volatility != "shock" {
		t.Errorf("expected volatility 'shock', got %s", lblShock.Volatility)
	}
	if lblShock.Composite != "shock_event" {
		t.Errorf("expected composite 'shock_event', got %s", lblShock.Composite)
	}

	// Add compressed test row at index 249
	rows[249].ATRPct14 = 0.005  // very low ATR
	rows[249].BBWidth20 = 0.005 // very low BB width
	lblComp, err := classifier.ClassifyOne(rows, 249)
	if err != nil {
		t.Fatalf("failed to classify: %v", err)
	}
	if lblComp.Volatility != "compressed" {
		t.Errorf("expected volatility 'compressed', got %s", lblComp.Volatility)
	}
}
