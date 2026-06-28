package app

import (
	"os/exec"
	"strings"
	"testing"
)

func TestNoAkTraderImport(t *testing.T) {
	cmd := exec.Command("go", "list", "-f", "{{.Imports}}", "./...")
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("go list failed: %v", err)
	}

	if strings.Contains(string(out), "ak-trader") {
		t.Errorf("Found ak-trader import in ak-engine!")
	}
}
