package app

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/davidmiguel22573/ak-engine/internal/data"
	"github.com/spf13/cobra"
)

var (
	planPath     string
	planSymbols  string
	planMarket   string
	planInterval string
	planFrom     string
	planTo       string
	planOut      string
)

type SymbolCoverage struct {
	Symbol           string   `json:"symbol"`
	HasLocalData     bool     `json:"has_local_data"`
	FirstCandleTime  string   `json:"first_candle_time,omitempty"`
	LastCandleTime   string   `json:"last_candle_time,omitempty"`
	RowCount         int      `json:"row_count"`
	MissingMonths    []int    `json:"missing_months"`
	BtcEthContext    bool     `json:"btc_eth_context_available"`
	UsableQ1         bool     `json:"usable_q1"`
	UsableH1         bool     `json:"usable_h1"`
	UsableH2         bool     `json:"usable_h2"`
	UsableFull2023   bool     `json:"usable_full_year_2023"`
	FetchCommand     string   `json:"fetch_command,omitempty"`
}

type OOSValidationPlan struct {
	TimeSplits struct {
		Train []string `json:"train_discovery"`
		OOS   []string `json:"oos_candidates"`
	} `json:"time_splits"`
	SymbolSplits     map[string][]string `json:"symbol_splits"`
	AcceptanceGates  []string            `json:"acceptance_gates"`
	CoverageSummary  map[string]SymbolCoverage `json:"coverage_summary"`
	Blocked          bool                `json:"blocked_by_missing_data"`
	Conclusion       string              `json:"conclusion"`
}

var planCompressionBreakoutOosCmd = &cobra.Command{
	Use:   "plan-compression-breakout-oos",
	Short: "Plan OOS validation for RegimeAwareCompressionBreakout_LONG",
	RunE: func(cmd *cobra.Command, args []string) error {
		symbols := strings.Split(planSymbols, ",")
		fromT, err := time.Parse("2006-01-02", planFrom)
		if err != nil {
			return err
		}
		toT, err := time.Parse("2006-01-02", planTo)
		if err != nil {
			return err
		}
		
		toT = toT.Add(24*time.Hour - time.Millisecond) // end of day

		src := data.NewLocalParquetSource()
		ctx := context.Background()

		coverage := make(map[string]SymbolCoverage)
		
		hasBTC := false
		hasETH := false
		// quick check for context availability
		if btc, _ := src.LoadCandles(ctx, data.CandleRequest{
			Path: planPath, Market: planMarket, Interval: planInterval, Symbol: "BTCUSDT", From: fromT, To: toT,
		}); len(btc) > 0 {
			hasBTC = true
		}
		if eth, _ := src.LoadCandles(ctx, data.CandleRequest{
			Path: planPath, Market: planMarket, Interval: planInterval, Symbol: "ETHUSDT", From: fromT, To: toT,
		}); len(eth) > 0 {
			hasETH = true
		}

		blocked := false

		for _, sym := range symbols {
			sym = strings.TrimSpace(sym)
			if sym == "" {
				continue
			}

			candles, err := src.LoadCandles(ctx, data.CandleRequest{
				Path: planPath, Market: planMarket, Interval: planInterval, Symbol: sym, From: fromT, To: toT,
			})
			
			cov := SymbolCoverage{
				Symbol: sym,
				HasLocalData: len(candles) > 0 && err == nil,
				BtcEthContext: hasBTC && hasETH,
			}
			
			if cov.HasLocalData {
				cov.FirstCandleTime = time.UnixMilli(candles[0].OpenTimeMS).UTC().Format(time.RFC3339)
				cov.LastCandleTime = time.UnixMilli(candles[len(candles)-1].OpenTimeMS).UTC().Format(time.RFC3339)
				cov.RowCount = len(candles)
				
				// Calculate missing months based on standard 2023 calendar (1 to 12)
				monthsPresent := make(map[int]bool)
				for _, c := range candles {
					tm := time.UnixMilli(c.OpenTimeMS).UTC()
					if tm.Year() == 2023 {
						monthsPresent[int(tm.Month())] = true
					}
				}
				for m := 1; m <= 12; m++ {
					if !monthsPresent[m] {
						cov.MissingMonths = append(cov.MissingMonths, m)
					}
				}
			} else {
				cov.MissingMonths = []int{1,2,3,4,5,6,7,8,9,10,11,12}
			}
			
			// Assess usable
			if len(cov.MissingMonths) == 0 {
				cov.UsableQ1 = true
				cov.UsableH1 = true
				cov.UsableH2 = true
				cov.UsableFull2023 = true
			} else {
				hasQ1 := true
				hasH1 := true
				hasH2 := true
				for _, m := range cov.MissingMonths {
					if m >= 1 && m <= 3 { hasQ1 = false }
					if m >= 1 && m <= 6 { hasH1 = false }
					if m >= 7 && m <= 12 { hasH2 = false }
				}
				cov.UsableQ1 = hasQ1
				cov.UsableH1 = hasH1
				cov.UsableH2 = hasH2
			}
			
			if len(cov.MissingMonths) > 0 {
				cov.FetchCommand = fmt.Sprintf("ak-historian fetch --symbol %s --market %s --interval %s --from %s --to %s", sym, planMarket, planInterval, planFrom, planTo)
				blocked = true
			}

			coverage[sym] = cov
		}
		
		// generate reports
		os.MkdirAll(filepath.Dir(planOut), 0755)
		
		// 1. Symbol coverage
		covDir := filepath.Dir(planOut)
		covJSONPath := filepath.Join(covDir, "phase10_0_symbol_coverage.json")
		covMDPath := filepath.Join(covDir, "phase10_0_symbol_coverage.md")
		
		cj, _ := os.Create(covJSONPath)
		enc := json.NewEncoder(cj)
		enc.SetIndent("", "  ")
		enc.Encode(coverage)
		cj.Close()
		
		cm, _ := os.Create(covMDPath)
		fmt.Fprintf(cm, "# Phase 10.0 Symbol Coverage\n\n")
		for _, sym := range symbols {
			c := coverage[sym]
			fmt.Fprintf(cm, "## %s\n", sym)
			fmt.Fprintf(cm, "- Has Local Data: %v\n", c.HasLocalData)
			if c.HasLocalData {
				fmt.Fprintf(cm, "- Rows: %d\n", c.RowCount)
				fmt.Fprintf(cm, "- First: %s\n", c.FirstCandleTime)
				fmt.Fprintf(cm, "- Last: %s\n", c.LastCandleTime)
			}
			fmt.Fprintf(cm, "- Missing Months: %v\n", c.MissingMonths)
			fmt.Fprintf(cm, "- BTC/ETH Context: %v\n", c.BtcEthContext)
			if c.FetchCommand != "" {
				fmt.Fprintf(cm, "\nNeed backfill:\n`%s`\n", c.FetchCommand)
			}
			fmt.Fprintf(cm, "\n")
		}
		cm.Close()
		
		// 2. OOS Validation Plan
		plan := OOSValidationPlan{}
		plan.TimeSplits.Train = []string{"Q1 2023", "H1 2023"}
		plan.TimeSplits.OOS = []string{"Q3 2023", "Q4 2023", "H2 2023", "Full 2024 if available later"}
		plan.SymbolSplits = map[string][]string{
			"original": {"LINKUSDT"},
			"majors": {"BTCUSDT", "ETHUSDT"},
			"high_beta": {"SOLUSDT", "ADAUSDT", "DOGEUSDT"},
			"expansion": {"BNBUSDT", "XRPUSDT", "AVAXUSDT"},
		}
		plan.AcceptanceGates = []string{
			"H2 2023 PF after 5 bps cost remains >= 1.10.",
			"H2 2023 expectancy after 5 bps cost remains positive.",
			"At least 3 months are positive after cost.",
			"No single month contributes more than 50% of total net expectancy.",
			"Entry delay of 1 candle remains positive.",
			"Same-candle conservative bracket assumptions remain enforced.",
			"No future-data leakage.",
			"At least one non-LINK symbol shows positive behavior, or the report explicitly justifies LINK-only specialization.",
		}
		plan.CoverageSummary = coverage
		plan.Blocked = blocked
		
		if blocked {
			plan.Conclusion = "blocked by missing data"
		} else {
			plan.Conclusion = "ready for H2 OOS validation"
		}
		
		pj, _ := os.Create(strings.TrimSuffix(planOut, ".md") + ".json")
		enc = json.NewEncoder(pj)
		enc.SetIndent("", "  ")
		enc.Encode(plan)
		pj.Close()
		
		pm, _ := os.Create(planOut)
		fmt.Fprintf(pm, "# Phase 10.0 OOS Validation Plan\n\n")
		fmt.Fprintf(pm, "## Conclusion: %s\n\n", plan.Conclusion)
		fmt.Fprintf(pm, "## Time Splits\nTrain: %v\nOOS: %v\n\n", plan.TimeSplits.Train, plan.TimeSplits.OOS)
		fmt.Fprintf(pm, "## Acceptance Gates\n")
		for _, g := range plan.AcceptanceGates {
			fmt.Fprintf(pm, "- %s\n", g)
		}
		pm.Close()

		return nil
	},
}

func init() {
	planCompressionBreakoutOosCmd.Flags().StringVar(&planPath, "path", "", "Path to data")
	planCompressionBreakoutOosCmd.Flags().StringVar(&planSymbols, "symbols", "", "Comma separated symbols")
	planCompressionBreakoutOosCmd.Flags().StringVar(&planMarket, "market", "", "Market")
	planCompressionBreakoutOosCmd.Flags().StringVar(&planInterval, "interval", "", "Interval")
	planCompressionBreakoutOosCmd.Flags().StringVar(&planFrom, "from", "", "From date (YYYY-MM-DD)")
	planCompressionBreakoutOosCmd.Flags().StringVar(&planTo, "to", "", "To date (YYYY-MM-DD)")
	planCompressionBreakoutOosCmd.Flags().StringVar(&planOut, "out", "", "Output markdown path")

	_ = planCompressionBreakoutOosCmd.MarkFlagRequired("path")
	_ = planCompressionBreakoutOosCmd.MarkFlagRequired("symbols")

	rootCmd.AddCommand(planCompressionBreakoutOosCmd)
}
