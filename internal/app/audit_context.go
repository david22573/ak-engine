package app

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/davidmiguel22573/ak-engine/internal/features"
	"github.com/spf13/cobra"
)

var (
	acFeaturesPath string
	acOutPath      string
)

type contextStats struct {
	NonZero    int
	NonZeroPct float64
	Min        float64
	Median     float64
	Max        float64
}

type thresholdStats struct {
	Threshold float64
	Up        int
	UpPct     float64
	Down      int
	DownPct   float64
	Flat      int
	FlatPct   float64
}

var auditContextCmd = &cobra.Command{
	Use:   "audit-context",
	Short: "Audit BTC/ETH context propagation",
	RunE: func(cmd *cobra.Command, args []string) error {
		file, err := os.Open(acFeaturesPath)
		if err != nil {
			return err
		}
		defer file.Close()

		var rows []features.Row
		if err := json.NewDecoder(file).Decode(&rows); err != nil {
			return err
		}

		report, err := renderContextAudit(rows)
		if err != nil {
			return err
		}

		fmt.Print(report)
		if acOutPath != "" {
			if err := os.MkdirAll(filepath.Dir(acOutPath), 0755); err != nil {
				return err
			}
			if err := os.WriteFile(acOutPath, []byte(report), 0644); err != nil {
				return err
			}
		}

		return nil
	},
}

func renderContextAudit(rows []features.Row) (string, error) {
	total := len(rows)
	if total == 0 {
		return "", fmt.Errorf("no rows found")
	}

	btc := summarizeContext(rows, func(r features.Row) float64 { return r.BTCReturn60 })
	eth := summarizeContext(rows, func(r features.Row) float64 { return r.ETHReturn60 })
	btcThresholds := summarizeThresholds(rows, func(r features.Row) float64 { return r.BTCReturn60 })
	ethThresholds := summarizeThresholds(rows, func(r features.Row) float64 { return r.ETHReturn60 })

	var b strings.Builder
	fmt.Fprintf(&b, "# Context Audit\n\n")
	fmt.Fprintf(&b, "Total Rows: %d\n\n", total)
	fmt.Fprintf(&b, "## Non-Zero Context\n")
	fmt.Fprintf(&b, "| Context | Non-Zero | Percentage | Min | Median | Max |\n")
	fmt.Fprintf(&b, "|---|---:|---:|---:|---:|---:|\n")
	fmt.Fprintf(&b, "| BTC | %d | %.2f%% | %.6f | %.6f | %.6f |\n", btc.NonZero, btc.NonZeroPct, btc.Min, btc.Median, btc.Max)
	fmt.Fprintf(&b, "| ETH | %d | %.2f%% | %.6f | %.6f | %.6f |\n\n", eth.NonZero, eth.NonZeroPct, eth.Min, eth.Median, eth.Max)

	writeThresholdTable := func(title string, stats []thresholdStats) {
		fmt.Fprintf(&b, "## %s Market Beta Distribution\n", title)
		fmt.Fprintf(&b, "| Threshold | Up | Up %% | Down | Down %% | Flat | Flat %% |\n")
		fmt.Fprintf(&b, "|---:|---:|---:|---:|---:|---:|---:|\n")
		for _, s := range stats {
			fmt.Fprintf(&b, "| %.3f | %d | %.2f%% | %d | %.2f%% | %d | %.2f%% |\n",
				s.Threshold, s.Up, s.UpPct, s.Down, s.DownPct, s.Flat, s.FlatPct)
		}
		fmt.Fprintf(&b, "\n")
	}
	writeThresholdTable("BTC", btcThresholds)
	writeThresholdTable("ETH", ethThresholds)

	return b.String(), nil
}

func summarizeContext(rows []features.Row, get func(features.Row) float64) contextStats {
	values := make([]float64, 0, len(rows))
	nonZero := 0
	for _, r := range rows {
		v := get(r)
		values = append(values, v)
		if v != 0 {
			nonZero++
		}
	}
	sort.Float64s(values)
	return contextStats{
		NonZero:    nonZero,
		NonZeroPct: float64(nonZero) / float64(len(rows)) * 100,
		Min:        values[0],
		Median:     medianSorted(values),
		Max:        values[len(values)-1],
	}
}

func summarizeThresholds(rows []features.Row, get func(features.Row) float64) []thresholdStats {
	bands := []float64{0.001, 0.002, 0.003, 0.005}
	out := make([]thresholdStats, 0, len(bands))
	total := float64(len(rows))
	for _, band := range bands {
		var up, down, flat int
		for _, r := range rows {
			v := get(r)
			if v > band {
				up++
			} else if v < -band {
				down++
			} else {
				flat++
			}
		}
		out = append(out, thresholdStats{
			Threshold: band,
			Up:        up,
			UpPct:     float64(up) / total * 100,
			Down:      down,
			DownPct:   float64(down) / total * 100,
			Flat:      flat,
			FlatPct:   float64(flat) / total * 100,
		})
	}
	return out
}

func medianSorted(values []float64) float64 {
	if len(values) == 0 {
		return 0
	}
	mid := len(values) / 2
	if len(values)%2 == 1 {
		return values[mid]
	}
	return (values[mid-1] + values[mid]) / 2
}

func init() {
	auditContextCmd.Flags().StringVar(&acFeaturesPath, "features", "", "Path to features JSON")
	auditContextCmd.Flags().StringVar(&acOutPath, "out", "", "Path to output markdown report")
	_ = auditContextCmd.MarkFlagRequired("features")
	rootCmd.AddCommand(auditContextCmd)
}
