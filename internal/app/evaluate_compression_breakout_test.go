package app

import (
	"encoding/json"
	"io/ioutil"
	"os"
	"strings"
	"testing"

	"github.com/davidmiguel22573/ak-engine/internal/features"
	"github.com/davidmiguel22573/ak-engine/internal/regime"
	"github.com/spf13/cobra"
)

func TestEvaluateCompressionBreakout_LeakageLabels(t *testing.T) {
	featFile, _ := ioutil.TempFile("", "feat-*.json")
	defer os.Remove(featFile.Name())

	regFile, _ := ioutil.TempFile("", "reg-*.json")
	defer os.Remove(regFile.Name())

	rows := []features.Row{
		{EventTimeMS: 100, Close: 10},
	}
	labels := []regime.Label{
		{EventTimeMS: 100, AvailableAtMS: 90, Volatility: "compressed"},
	}

	json.NewEncoder(featFile).Encode(rows)
	json.NewEncoder(regFile).Encode(labels)

	ecbFeatures = featFile.Name()
	ecbRegimes = regFile.Name()
	ecbSide = "LONG"
	ecbOut = ""

	err := evaluateCompressionBreakoutCmd.RunE(nil, nil)
	if err == nil {
		t.Errorf("expected leakage error, got nil")
	} else if err.Error() != "leakage detected: label 0" {
		t.Errorf("expected 'leakage detected: label 0' error, got: %v", err)
	}
}

func TestEvaluateCompressionBreakout_LeakageDuplicateTimestamps(t *testing.T) {
	featFile, _ := ioutil.TempFile("", "feat-*.json")
	defer os.Remove(featFile.Name())

	regFile, _ := ioutil.TempFile("", "reg-*.json")
	defer os.Remove(regFile.Name())

	rows := []features.Row{
		{EventTimeMS: 100, Close: 10},
		{EventTimeMS: 100, Close: 11},
	}
	labels := []regime.Label{
		{EventTimeMS: 90, AvailableAtMS: 90, Volatility: "compressed"},
	}

	json.NewEncoder(featFile).Encode(rows)
	json.NewEncoder(regFile).Encode(labels)

	ecbFeatures = featFile.Name()
	ecbRegimes = regFile.Name()
	ecbSide = "LONG"
	ecbOut = ""

	err := evaluateCompressionBreakoutCmd.RunE(nil, nil)
	if err == nil {
		t.Errorf("expected leakage error, got nil")
	} else if err.Error() != "duplicate timestamps break evaluation" {
		t.Errorf("expected 'duplicate timestamps break evaluation' error, got: %v", err)
	}
}

func TestEvaluateCompressionBreakout_AcceptanceGateCalculation(t *testing.T) {
	rep := CBReport{
		Haircuts:      map[string]CBMetrics{"5bps": {ProfitFactor: 1.11, Expectancy: 0.0002}},
		EntryDelay:    map[string]CBMetrics{"1c": {Expectancy: 0.0001}},
		Monthly:       map[string]CBMonthlyMetrics{"2023-07": {NetResult: 0.3}, "2023-08": {NetResult: 0.3}, "2023-09": {NetResult: 0.4}},
		LeakageStatus: "PASS",
	}

	gate := evaluateAcceptanceGate(rep)
	if !gate.Passed {
		t.Fatalf("expected gate pass, got failures: %v", gate.FailedGates)
	}
	if gate.PositiveMonthsAfterCost != 3 {
		t.Fatalf("expected 3 positive months, got %d", gate.PositiveMonthsAfterCost)
	}
	if gate.MaxMonthContributionPct != 40 {
		t.Fatalf("expected max month contribution 40%%, got %.2f", gate.MaxMonthContributionPct)
	}
}

func TestEvaluateCompressionBreakout_H2FailureNotOverriddenByFY(t *testing.T) {
	h2 := CBReport{
		Haircuts:      map[string]CBMetrics{"5bps": {ProfitFactor: 0.90, Expectancy: -0.0001}},
		EntryDelay:    map[string]CBMetrics{"1c": {Expectancy: 0.0001}},
		Monthly:       map[string]CBMonthlyMetrics{"2023-07": {NetResult: 0.3}, "2023-08": {NetResult: 0.3}, "2023-09": {NetResult: 0.4}},
		LeakageStatus: "PASS",
	}
	fy := CBReport{
		Haircuts:      map[string]CBMetrics{"5bps": {ProfitFactor: 1.20, Expectancy: 0.0002}},
		EntryDelay:    map[string]CBMetrics{"1c": {Expectancy: 0.0001}},
		Monthly:       map[string]CBMonthlyMetrics{"2023-01": {NetResult: 0.3}, "2023-02": {NetResult: 0.3}, "2023-03": {NetResult: 0.4}},
		LeakageStatus: "PASS",
	}

	h2Gate := evaluateAcceptanceGate(h2)
	fyGate := evaluateAcceptanceGate(fy)
	if h2Gate.Passed {
		t.Fatal("expected H2 gate to fail")
	}
	if !fyGate.Passed {
		t.Fatalf("expected FY gate to pass in control setup, got %v", fyGate.FailedGates)
	}
	if h2Gate.Verdict == "pass" {
		t.Fatal("H2 failure must not be overridden by FY pass")
	}
}

func TestBuildFeatures_SkipsSelfContext(t *testing.T) {
	btcCtx := contextSymbolListForTarget("BTCUSDT", "BTCUSDT,ETHUSDT")
	if len(btcCtx) != 1 || btcCtx[0] != "ETHUSDT" {
		t.Fatalf("expected BTCUSDT context to exclude BTCUSDT, got %v", btcCtx)
	}

	ethCtx := contextSymbolListForTarget("ETHUSDT", "BTCUSDT,ETHUSDT")
	if len(ethCtx) != 1 || ethCtx[0] != "BTCUSDT" {
		t.Fatalf("expected ETHUSDT context to exclude ETHUSDT, got %v", ethCtx)
	}
}

func TestBuildFeatures_MissingSymbolDataReported(t *testing.T) {
	oldSource, oldPath, oldMarket, oldSymbol := bfSource, bfPath, bfMarket, bfSymbol
	oldInterval, oldFrom, oldTo := bfInterval, bfFrom, bfTo
	oldContext, oldOut, oldFormat, oldDropWarmup := bfContextSymbols, bfOut, bfFormat, bfDropWarmup
	defer func() {
		bfSource, bfPath, bfMarket, bfSymbol = oldSource, oldPath, oldMarket, oldSymbol
		bfInterval, bfFrom, bfTo = oldInterval, oldFrom, oldTo
		bfContextSymbols, bfOut, bfFormat, bfDropWarmup = oldContext, oldOut, oldFormat, oldDropWarmup
	}()

	bfSource = "local-parquet"
	bfPath = t.TempDir()
	bfMarket = "futures-um"
	bfSymbol = "MISSINGUSDT"
	bfInterval = "1m"
	bfFrom = "2023-01-01"
	bfTo = "2023-01-02"
	bfContextSymbols = ""
	bfOut = ""
	bfFormat = "json"
	bfDropWarmup = false

	err := buildFeaturesCmd.RunE(&cobra.Command{}, nil)
	if err == nil {
		t.Fatal("expected missing local parquet data error")
	}
	if !strings.Contains(err.Error(), "failed to load primary candles") {
		t.Fatalf("expected primary candle load error, got %v", err)
	}
}
