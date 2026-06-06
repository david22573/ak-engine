package regime

import (
	"testing"

	"github.com/davidmiguel22573/ak-engine/internal/features"
)

func TestClassify_MarketBeta(t *testing.T) {
	c := NewClassifier(ThresholdOptions{LookbackRows: 1, MinRows: 1})

	// up
	rowUp := features.Row{BTCReturn60: 0.005}
	lblUp, _ := c.ClassifyOne([]features.Row{{}, rowUp}, 1)
	if lblUp.MarketBeta != "btc_up" {
		t.Errorf("expected btc_up, got %s", lblUp.MarketBeta)
	}

	// down
	rowDown := features.Row{BTCReturn60: -0.005}
	lblDown, _ := c.ClassifyOne([]features.Row{{}, rowDown}, 1)
	if lblDown.MarketBeta != "btc_down" {
		t.Errorf("expected btc_down, got %s", lblDown.MarketBeta)
	}

	// flat
	rowFlat := features.Row{BTCReturn60: 0.001}
	lblFlat, _ := c.ClassifyOne([]features.Row{{}, rowFlat}, 1)
	if lblFlat.MarketBeta != "btc_flat" {
		t.Errorf("expected btc_flat, got %s", lblFlat.MarketBeta)
	}
}
