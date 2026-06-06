package app

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/spf13/cobra"
)

var (
	eabmDir string
	eabmOut string
)

type SymbolCandidate struct {
	Symbol   string
	Family   string
	Exp      float64
	PF       float64
	Count    int
	WinRate  float64
}

var evaluateAlphaBaselinesMultisymbolCmd = &cobra.Command{
	Use:   "evaluate-alpha-baselines-multisymbol",
	Short: "Aggregate multiple evaluate-alpha-baselines JSON reports into a single multisymbol leaderboard",
	RunE: func(cmd *cobra.Command, args []string) error {
		if eabmDir == "" || eabmOut == "" {
			return errors.New("missing --dir or --out")
		}

		files, err := os.ReadDir(eabmDir)
		if err != nil {
			return fmt.Errorf("read dir: %w", err)
		}

		var allCands []SymbolCandidate
		processed := 0

		for _, f := range files {
			if !strings.HasSuffix(f.Name(), ".json") {
				continue
			}
			path := filepath.Join(eabmDir, f.Name())
			data, err := os.ReadFile(path)
			if err != nil {
				continue
			}

			var rep BaselineReportJSON
			if err := json.Unmarshal(data, &rep); err != nil {
				continue
			}

			if rep.Global == nil {
				continue // not a valid report
			}

			// Extract symbol from filename (e.g., BTCUSDT-alpha-baselines.json or similar)
			parts := strings.Split(f.Name(), "-")
			sym := parts[0]
			if sym == "" {
				sym = f.Name()
			}

			processed++

			for family, m := range rep.Global {
				if m.SampleWarning == "USABLE_SAMPLE" || m.SampleWarning == "WEAK_SAMPLE" {
					allCands = append(allCands, SymbolCandidate{
						Symbol:  sym,
						Family:  family,
						Exp:     m.Expectancy15,
						PF:      m.ProfitFactor15,
						Count:   m.EventCount,
						WinRate: m.WinRate15,
					})
				}
			}
		}

		if processed == 0 {
			return fmt.Errorf("no valid JSON reports found in %s", eabmDir)
		}

		// Sort by Expectancy
		sort.Slice(allCands, func(i, j int) bool {
			return allCands[i].Exp > allCands[j].Exp
		})

		var sb strings.Builder
		sb.WriteString("# Phase 10.2: Multi-Symbol Baseline Leaderboard\n\n")
		sb.WriteString(fmt.Sprintf("Aggregated from %d reports in `%s`.\n\n", processed, eabmDir))

		sb.WriteString("## Top Candidates by 15m Expectancy\n")
		sb.WriteString("| Rank | Symbol | Family | Exp (bps) | PF | WinRate | Count |\n")
		sb.WriteString("|---|---|---|---|---|---|---|\n")

		limit := 50
		if len(allCands) < limit {
			limit = len(allCands)
		}

		for i := 0; i < limit; i++ {
			c := allCands[i]
			expBps := c.Exp * 10000.0 // Assuming Exp is fractional
			sb.WriteString(fmt.Sprintf("| %d | %s | %s | %.2f | %.2f | %.2f%% | %d |\n",
				i+1, c.Symbol, c.Family, expBps, c.PF, c.WinRate*100, c.Count))
		}

		if err := os.WriteFile(eabmOut, []byte(sb.String()), 0644); err != nil {
			return fmt.Errorf("write out: %w", err)
		}

		jsonOut := strings.Replace(eabmOut, ".md", ".json", 1)
		if strings.HasSuffix(jsonOut, ".json") {
			data, _ := json.MarshalIndent(allCands[:limit], "", "  ")
			os.WriteFile(jsonOut, data, 0644)
		}

		fmt.Printf("Multisymbol leaderboard written to %s\n", eabmOut)
		return nil
	},
}

func init() {
	evaluateAlphaBaselinesMultisymbolCmd.Flags().StringVar(&eabmDir, "dir", "", "Directory containing evaluate-alpha-baselines JSON reports")
	evaluateAlphaBaselinesMultisymbolCmd.Flags().StringVar(&eabmOut, "out", "", "Output markdown path")
	rootCmd.AddCommand(evaluateAlphaBaselinesMultisymbolCmd)
}
