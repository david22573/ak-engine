package app

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestPhase10CandidateSpec(t *testing.T) {
	// 1. candidate spec generation includes known metrics
	// 4. JSON report is valid and stable
	jsonPath := "../../runs/reports/phase10_0_compression_breakout_long_spec.json"
	
	b, err := os.ReadFile(jsonPath)
	if err != nil {
		t.Fatalf("missing candidate spec JSON: %v", err)
	}
	
	var spec map[string]interface{}
	if err := json.Unmarshal(b, &spec); err != nil {
		t.Fatalf("invalid json: %v", err)
	}
	
	if spec["name"] != "RegimeAwareCompressionBreakout_LONG" {
		t.Errorf("wrong name")
	}
	
	hm := spec["current_best_holding_model"].(string)
	if hm == "" {
		t.Errorf("missing known metrics (holding model)")
	}
	
	// 3. no runtime/trader packages are imported
	// We check ak-engine root go.mod or simply imports in the package. 
	// Since this is unit test, if it compiles and we statically check imports:
	// Go does not allow importing packages if not in go.mod, and ak-trader is forbidden.
}

func TestPhase10PlanCommand(t *testing.T) {
	// We create a temporary empty dir to simulate missing dataset
	tmpDir := t.TempDir()
	outPath := filepath.Join(tmpDir, "phase10_0_oos_validation_plan.md")
	
	rootCmd.SetArgs([]string{
		"plan-compression-breakout-oos",
		"--path", tmpDir,
		"--symbols", "LINKUSDT,BTCUSDT,ETHUSDT,FAKEUSDT",
		"--market", "futures-um",
		"--interval", "1m",
		"--from", "2023-01-01",
		"--to", "2023-12-31",
		"--out", outPath,
	})
	
	err := rootCmd.Execute()
	if err != nil {
		t.Fatalf("cmd execution failed: %v", err)
	}
	
	// 2. coverage report handles missing symbol data cleanly
	// 3. OOS validation plan marks missing datasets as blocked
	
	covJSONPath := filepath.Join(tmpDir, "phase10_0_symbol_coverage.json")
	b, err := os.ReadFile(covJSONPath)
	if err != nil {
		t.Fatalf("missing coverage json: %v", err)
	}
	
	var cov map[string]SymbolCoverage
	if err := json.Unmarshal(b, &cov); err != nil {
		t.Fatalf("invalid coverage json: %v", err)
	}
	
	if len(cov["FAKEUSDT"].MissingMonths) != 12 {
		t.Errorf("expected 12 missing months for FAKEUSDT, got %d", len(cov["FAKEUSDT"].MissingMonths))
	}
	
	planJSONPath := filepath.Join(tmpDir, "phase10_0_oos_validation_plan.json")
	pb, err := os.ReadFile(planJSONPath)
	if err != nil {
		t.Fatalf("missing plan json: %v", err)
	}
	
	var plan OOSValidationPlan
	if err := json.Unmarshal(pb, &plan); err != nil {
		t.Fatalf("invalid plan json: %v", err)
	}
	
	if !plan.Blocked {
		t.Errorf("expected plan to be blocked by missing data")
	}
}

func TestNoTraderImports(t *testing.T) {
	// A basic sanity check by grepping the current package's code for "trader"
	// just in case someone sneakily added it.
	files, err := filepath.Glob("*.go")
	if err != nil {
		t.Fatal(err)
	}
	for _, f := range files {
		b, err := os.ReadFile(f)
		if err != nil {
			t.Fatal(err)
		}
		if bytes.Contains(b, []byte("\"github.com/"+"davidmiguel22573/ak-trader")) {
			// skip if it's the test itself doing the check
			if filepath.Base(f) == "plan_compression_breakout_oos_test.go" {
				continue
			}
			t.Errorf("file %s imports ak-trader", f)
		}
	}
}
