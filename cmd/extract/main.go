package main

import (
	"os"

	"github.com/david22573/ak-engine/internal/qualificationrunner"
)

func main() {
	if len(os.Args) > 1 {
		qualificationrunner.ExtractMetrics(os.Args[1])
	}
}
