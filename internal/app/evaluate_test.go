package app

import (
	"encoding/json"
	"io/ioutil"
	"os"
	"testing"
	"github.com/davidmiguel22573/ak-engine/internal/features"
	"github.com/davidmiguel22573/ak-engine/internal/regime"
)

func TestEvaluateShockFade_Leakage(t *testing.T) {
	// Create dummy features and regimes
	featFile, _ := ioutil.TempFile("", "feat-*.json")
	defer os.Remove(featFile.Name())
	
	regFile, _ := ioutil.TempFile("", "reg-*.json")
	defer os.Remove(regFile.Name())

	// Leakage: label available at 90 but event at 100
	rows := []features.Row{
		{EventTimeMS: 100, Close: 10, Return5: -0.01},
	}
	labels := []regime.Label{
		{EventTimeMS: 100, AvailableAtMS: 90, Volatility: "shock"},
	}

	json.NewEncoder(featFile).Encode(rows)
	json.NewEncoder(regFile).Encode(labels)

	esfFeatures = featFile.Name()
	esfRegimes = regFile.Name()
	esfSide = "LONG"
	esfOut = ""

	err := evaluateShockFadeCmd.RunE(nil, nil)
	if err == nil {
		t.Errorf("expected leakage error, got nil")
	} else if err.Error() != "leakage detected" {
		t.Errorf("expected 'leakage detected' error, got: %v", err)
	}
}

func TestEvaluateAlphaBaselines_CompressionCandidates(t *testing.T) {
	// Not fully tested here, logic is in main command loop
}
