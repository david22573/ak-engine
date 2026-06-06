package app

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"
)

var (
	acdName     string
	acdH1Path   string
	acdH2Path   string
	acdOutPath  string
)

var analyzeCandidateDecayCmd = &cobra.Command{
	Use:   "analyze-candidate-decay",
	Short: "Analyze performance decay between H1 and H2 and output a rejection/decay report",
	RunE: func(cmd *cobra.Command, args []string) error {
		if acdH1Path == "" || acdH2Path == "" || acdOutPath == "" {
			return errors.New("missing required flags: --h1-report, --h2-report, --out")
		}

		h1Data, err := os.ReadFile(acdH1Path)
		if err != nil {
			return fmt.Errorf("read h1: %w", err)
		}
		var h1 CBReport
		if err := json.Unmarshal(h1Data, &h1); err != nil {
			return fmt.Errorf("unmarshal h1: %w", err)
		}

		h2Data, err := os.ReadFile(acdH2Path)
		if err != nil {
			return fmt.Errorf("read h2: %w", err)
		}
		var h2 CBReport
		if err := json.Unmarshal(h2Data, &h2); err != nil {
			return fmt.Errorf("unmarshal h2: %w", err)
		}

		var sb strings.Builder
		sb.WriteString(fmt.Sprintf("# Phase 10.2: Candidate Decay & Rejection Record - %s\n\n", acdName))

		sb.WriteString("## Formal Status: REJECTED\n")
		sb.WriteString("This candidate failed out-of-sample validation. The edge demonstrated in H1 decayed completely in H2, resulting in negative expectancy after costs.\n\n")

		sb.WriteString("## Decay Analysis (60m Horizon)\n")
		sb.WriteString("| Metric | H1 (In-Sample) | H2 (Out-of-Sample) | Decay |\n")
		sb.WriteString("|---|---|---|---|\n")

		h1m := h1.Horizons["60m"]
		h2m := h2.Horizons["60m"]

		diffPF := h2m.ProfitFactor - h1m.ProfitFactor
		diffExp := h2m.Expectancy - h1m.Expectancy
		diffWin := (h2m.WinRate - h1m.WinRate) * 100.0

		sb.WriteString(fmt.Sprintf("| Trade Count | %d | %d | - |\n", h1m.EventCount, h2m.EventCount))
		sb.WriteString(fmt.Sprintf("| Win Rate | %.2f%% | %.2f%% | %.2f%% |\n", h1m.WinRate*100, h2m.WinRate*100, diffWin))
		sb.WriteString(fmt.Sprintf("| Profit Factor (Gross) | %.2f | %.2f | %.2f |\n", h1m.ProfitFactor, h2m.ProfitFactor, diffPF))
		sb.WriteString(fmt.Sprintf("| Expectancy (Gross bps) | %.2f | %.2f | %.2f |\n", h1m.Expectancy, h2m.Expectancy, diffExp))

		sb.WriteString("\n## Cost Adjustment (5 bps per leg)\n")
		sb.WriteString("| Metric | H1 | H2 | Status |\n")
		sb.WriteString("|---|---|---|---|\n")
		
		h1c := h1.Haircuts["5_bps"]
		h2c := h2.Haircuts["5_bps"]
		statusH2 := "PASS"
		if h2c.Expectancy <= 0 {
			statusH2 = "FAIL"
		}

		sb.WriteString(fmt.Sprintf("| PF After Cost | %.4f | %.4f | %s |\n", h1c.ProfitFactor, h2c.ProfitFactor, statusH2))
		sb.WriteString(fmt.Sprintf("| Exp After Cost (bps) | %.4f | %.4f | %s |\n", h1c.Expectancy, h2c.Expectancy, statusH2))

		sb.WriteString("\n## Rejection Reasons\n")
		if h2c.Expectancy <= 0 {
			sb.WriteString("- Out-of-sample expectancy after minimum costs is negative.\n")
		}
		if h2c.ProfitFactor < 1.0 {
			sb.WriteString("- Out-of-sample profit factor is less than 1.0.\n")
		}
		sb.WriteString("- Lack of temporal stability (H1 edge did not survive into H2).\n")

		if err := os.WriteFile(acdOutPath, []byte(sb.String()), 0644); err != nil {
			return fmt.Errorf("write output: %w", err)
		}

		jsonOut := strings.Replace(acdOutPath, ".md", ".json", 1)
		if strings.HasSuffix(jsonOut, ".json") {
			type DecayJson struct {
				Status string `json:"status"`
				H1     CBMetrics `json:"h1"`
				H2     CBMetrics `json:"h2"`
				H1Cost CBMetrics `json:"h1_cost"`
				H2Cost CBMetrics `json:"h2_cost"`
			}
			dj := DecayJson{
				Status: "REJECTED",
				H1:     h1m,
				H2:     h2m,
				H1Cost: h1c,
				H2Cost: h2c,
			}
			data, _ := json.MarshalIndent(dj, "", "  ")
			os.WriteFile(jsonOut, data, 0644)
		}

		fmt.Printf("Decay report written to %s\n", acdOutPath)
		return nil
	},
}

func init() {
	analyzeCandidateDecayCmd.Flags().StringVar(&acdName, "name", "Candidate", "Candidate name")
	analyzeCandidateDecayCmd.Flags().StringVar(&acdH1Path, "h1-report", "", "Path to H1 json report")
	analyzeCandidateDecayCmd.Flags().StringVar(&acdH2Path, "h2-report", "", "Path to H2 json report")
	analyzeCandidateDecayCmd.Flags().StringVar(&acdOutPath, "out", "", "Path to output markdown report")
	rootCmd.AddCommand(analyzeCandidateDecayCmd)
}
