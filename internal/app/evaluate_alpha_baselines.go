package app

import (
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"os"
	"sort"
	"strings"

	"github.com/davidmiguel22573/ak-engine/internal/features"
	"github.com/davidmiguel22573/ak-engine/internal/regime"
	"github.com/spf13/cobra"
)

var (
	eabFeaturesPath string
	eabRegimesPath  string
	eabOutPath      string
)

type BaselineEvent struct {
	Family      string
	Direction   string
	EventTimeMS int64
	Label       regime.Label
	Fwd5        float64
	Fwd15       float64
	Fwd30       float64
	Fwd60       float64
	MAE         float64
	MFE         float64
}

type BaselineMetrics struct {
	EventCount     int     `json:"event_count"`
	AvgFwd5        float64 `json:"avg_fwd_5m"`
	AvgFwd15       float64 `json:"avg_fwd_15m"`
	AvgFwd30       float64 `json:"avg_fwd_30m"`
	AvgFwd60       float64 `json:"avg_fwd_60m"`
	MedianFwd15    float64 `json:"median_fwd_15m"`
	WinRate15      float64 `json:"win_rate_15m"`
	ProfitFactor15 float64 `json:"profit_factor_proxy_15m"`
	Expectancy15   float64 `json:"expectancy_15m"`
	AvgMAE         float64 `json:"avg_mae"`
	AvgMFE         float64 `json:"avg_mfe"`
	SampleWarning  string  `json:"sample_warning"`
}

type BaselineReportJSON struct {
	Summary struct {
		TotalFeatures int `json:"total_features"`
		TotalLabels   int `json:"total_labels"`
		TotalEvents   int `json:"total_events"`
	} `json:"summary"`
	LeakageStatus string `json:"leakage_status"`

	Global            map[string]BaselineMetrics            `json:"global"`
	ByCompositeRegime map[string]map[string]BaselineMetrics `json:"by_composite_regime"`
	ByVolatility      map[string]map[string]BaselineMetrics `json:"by_volatility"`
	ByTrend           map[string]map[string]BaselineMetrics `json:"by_trend"`
	ByLiquidity       map[string]map[string]BaselineMetrics `json:"by_liquidity"`
	ByMarketBeta      map[string]map[string]BaselineMetrics `json:"by_market_beta"`

	Conclusion string `json:"conclusion"`
}

var evaluateAlphaBaselinesCmd = &cobra.Command{
	Use:   "evaluate-alpha-baselines",
	Short: "Evaluate deterministic alpha baselines across regimes",
	RunE: func(cmd *cobra.Command, args []string) error {
		if eabFeaturesPath == "" {
			return errors.New("missing --features")
		}
		if eabRegimesPath == "" {
			return errors.New("missing --regimes")
		}
		if eabOutPath == "" {
			return errors.New("missing --out")
		}

		// Load Features
		featuresData, err := os.ReadFile(eabFeaturesPath)
		if err != nil {
			return fmt.Errorf("read features: %w", err)
		}
		var rows []features.Row
		if err := json.Unmarshal(featuresData, &rows); err != nil {
			return fmt.Errorf("unmarshal features: %w", err)
		}
		sort.Slice(rows, func(i, j int) bool {
			return rows[i].EventTimeMS < rows[j].EventTimeMS
		})

		// Load Regimes
		regimesData, err := os.ReadFile(eabRegimesPath)
		if err != nil {
			return fmt.Errorf("read regimes: %w", err)
		}
		var labels []regime.Label
		if err := json.Unmarshal(regimesData, &labels); err != nil {
			return fmt.Errorf("unmarshal regimes: %w", err)
		}
		sort.Slice(labels, func(i, j int) bool {
			return labels[i].AvailableAtMS < labels[j].AvailableAtMS
		})

		// Check leakage
		for i := 1; i < len(labels); i++ {
			if labels[i].AvailableAtMS < labels[i].EventTimeMS {
				return fmt.Errorf("leakage detected: label %d available before event", i)
			}
		}

		var events []BaselineEvent

		// Helper to find future close price
		getFuture := func(startIndex int, offsetMs int64) float64 {
			targetTime := rows[startIndex].EventTimeMS + offsetMs
			for i := startIndex; i < len(rows); i++ {
				if rows[i].EventTimeMS >= targetTime {
					return rows[i].Close
				}
			}
			if len(rows) > 0 {
				return rows[len(rows)-1].Close
			}
			return 0
		}

		getExcursions := func(startIndex int, offsetMs int64, dir string) (float64, float64) {
			targetTime := rows[startIndex].EventTimeMS + offsetMs
			startClose := rows[startIndex].Close
			maxPrice := startClose
			minPrice := startClose
			for i := startIndex; i < len(rows); i++ {
				if rows[i].EventTimeMS > targetTime {
					break
				}
				if rows[i].Close > maxPrice {
					maxPrice = rows[i].Close
				}
				if rows[i].Close < minPrice {
					minPrice = rows[i].Close
				}
			}
			if startClose == 0 {
				return 0, 0
			}
			if dir == "LONG" {
				return (minPrice - startClose) / startClose, (maxPrice - startClose) / startClose
			}
			return (startClose - maxPrice) / startClose, (startClose - minPrice) / startClose
		}

		for i, r := range rows {
			if r.Warmup {
				continue
			}
			// Find regime label
			idx := sort.Search(len(labels), func(k int) bool {
				return labels[k].AvailableAtMS > r.EventTimeMS
			})
			if idx == 0 {
				continue // No prior label
			}
			label := labels[idx-1]
			if label.AvailableAtMS > r.EventTimeMS {
				return fmt.Errorf("join leakage: label at %d used for event at %d", label.AvailableAtMS, r.EventTimeMS)
			}

			// Generate candidates
			var cands []BaselineEvent

			// 1. Trend continuation
			if r.Close > r.EMA20 && r.EMA20 > r.EMA50 && r.TrendSlope20 > 0 {
				cands = append(cands, BaselineEvent{Family: "TrendContinuation", Direction: "LONG"})
			} else if r.Close < r.EMA20 && r.EMA20 < r.EMA50 && r.TrendSlope20 < 0 {
				cands = append(cands, BaselineEvent{Family: "TrendContinuation", Direction: "SHORT"})
			}

			// 2. Compression breakout candidate
			if label.Volatility == "compressed" || label.Composite == "compressed_range" {
				if r.Close > r.EMA20 && label.MarketBeta == "btc_up" {
					cands = append(cands, BaselineEvent{Family: "CompressionBreakout", Direction: "LONG"})
				} else if r.Close < r.EMA20 && label.MarketBeta == "btc_down" {
					cands = append(cands, BaselineEvent{Family: "CompressionBreakout", Direction: "SHORT"})
				}
			}

			// 3. Shock avoidance / shock fade diagnostic
			if label.Volatility == "shock" {
				if r.Return5 < 0 {
					cands = append(cands, BaselineEvent{Family: "ShockFade", Direction: "LONG"})
				} else if r.Return5 > 0 {
					cands = append(cands, BaselineEvent{Family: "ShockFade", Direction: "SHORT"})
				}
			}

			// 4. Volume-confirmed momentum
			if label.Liquidity == "heavy" {
				if r.Return5 > 0 && r.Return15 > 0 {
					cands = append(cands, BaselineEvent{Family: "VolumeMomentum", Direction: "LONG"})
				} else if r.Return5 < 0 && r.Return15 < 0 {
					cands = append(cands, BaselineEvent{Family: "VolumeMomentum", Direction: "SHORT"})
				}
			}

			// 5. BTC/ETH beta confirmation
			if r.Return5 > 0 && label.MarketBeta == "btc_up" {
				cands = append(cands, BaselineEvent{Family: "BetaAgrees", Direction: "LONG"})
			} else if r.Return5 < 0 && label.MarketBeta == "btc_down" {
				cands = append(cands, BaselineEvent{Family: "BetaAgrees", Direction: "SHORT"})
			} else if r.Return5 > 0 && label.MarketBeta == "btc_down" {
				cands = append(cands, BaselineEvent{Family: "BetaDiverges", Direction: "LONG"})
			} else if r.Return5 < 0 && label.MarketBeta == "btc_up" {
				cands = append(cands, BaselineEvent{Family: "BetaDiverges", Direction: "SHORT"})
			}

			// Compute forward returns
			f5 := getFuture(i, 5*60000)
			f15 := getFuture(i, 15*60000)
			f30 := getFuture(i, 30*60000)
			f60 := getFuture(i, 60*60000)

			for _, c := range cands {
				c.EventTimeMS = r.EventTimeMS
				c.Label = label
				
				ret5 := (f5 - r.Close) / r.Close
				ret15 := (f15 - r.Close) / r.Close
				ret30 := (f30 - r.Close) / r.Close
				ret60 := (f60 - r.Close) / r.Close

				if c.Direction == "SHORT" {
					ret5 = -ret5
					ret15 = -ret15
					ret30 = -ret30
					ret60 = -ret60
				}
				c.Fwd5 = ret5
				c.Fwd15 = ret15
				c.Fwd30 = ret30
				c.Fwd60 = ret60

				mae, mfe := getExcursions(i, 60*60000, c.Direction)
				c.MAE = mae
				c.MFE = mfe

				events = append(events, c)
			}
		}

		// Calculate Metrics
		out := BaselineReportJSON{
			Global:            make(map[string]BaselineMetrics),
			ByCompositeRegime: make(map[string]map[string]BaselineMetrics),
			ByVolatility:      make(map[string]map[string]BaselineMetrics),
			ByTrend:           make(map[string]map[string]BaselineMetrics),
			ByLiquidity:       make(map[string]map[string]BaselineMetrics),
			ByMarketBeta:      make(map[string]map[string]BaselineMetrics),
		}

		out.Summary.TotalFeatures = len(rows)
		out.Summary.TotalLabels = len(labels)
		out.Summary.TotalEvents = len(events)
		out.LeakageStatus = "PASS"

		groupEvents := func(evs []BaselineEvent, grouper func(BaselineEvent) string) map[string][]BaselineEvent {
			m := make(map[string][]BaselineEvent)
			for _, e := range evs {
				k := grouper(e)
				m[k] = append(m[k], e)
			}
			return m
		}

		calcMetrics := func(evs []BaselineEvent) BaselineMetrics {
			var m BaselineMetrics
			m.EventCount = len(evs)
			if m.EventCount == 0 {
				return m
			}

			if m.EventCount < 100 {
				m.SampleWarning = "UNTRUSTED_LOW_SAMPLE"
			} else if m.EventCount <= 300 {
				m.SampleWarning = "WEAK_SAMPLE"
			} else {
				m.SampleWarning = "USABLE_SAMPLE"
			}

			var sum5, sum15, sum30, sum60, sumMae, sumMfe float64
			var wins, losses int
			var grossProf, grossLoss float64
			var rets15 []float64

			for _, e := range evs {
				sum5 += e.Fwd5
				sum15 += e.Fwd15
				sum30 += e.Fwd30
				sum60 += e.Fwd60
				sumMae += e.MAE
				sumMfe += e.MFE
				rets15 = append(rets15, e.Fwd15)

				if e.Fwd15 > 0 {
					wins++
					grossProf += e.Fwd15
				} else {
					losses++
					grossLoss += math.Abs(e.Fwd15)
				}
			}

			m.AvgFwd5 = sum5 / float64(m.EventCount)
			m.AvgFwd15 = sum15 / float64(m.EventCount)
			m.AvgFwd30 = sum30 / float64(m.EventCount)
			m.AvgFwd60 = sum60 / float64(m.EventCount)
			m.AvgMAE = sumMae / float64(m.EventCount)
			m.AvgMFE = sumMfe / float64(m.EventCount)

			sort.Float64s(rets15)
			if len(rets15) > 0 {
				m.MedianFwd15 = rets15[len(rets15)/2]
			}

			m.WinRate15 = float64(wins) / float64(m.EventCount)
			if grossLoss > 0 {
				m.ProfitFactor15 = grossProf / grossLoss
			} else if grossProf > 0 {
				m.ProfitFactor15 = 999.0
			}

			avgWin := 0.0
			if wins > 0 {
				avgWin = grossProf / float64(wins)
			}
			avgLoss := 0.0
			if losses > 0 {
				avgLoss = grossLoss / float64(losses)
			}
			m.Expectancy15 = (m.WinRate15 * avgWin) - ((1 - m.WinRate15) * avgLoss)

			return m
		}

		byGlobal := groupEvents(events, func(e BaselineEvent) string { return e.Family + "_" + e.Direction })
		for k, v := range byGlobal {
			out.Global[k] = calcMetrics(v)
		}

		populateMap := func(target map[string]map[string]BaselineMetrics, grouper func(BaselineEvent) string) {
			grouped := groupEvents(events, grouper)
			for groupKey, groupEvs := range grouped {
				target[groupKey] = make(map[string]BaselineMetrics)
				byFam := groupEvents(groupEvs, func(e BaselineEvent) string { return e.Family + "_" + e.Direction })
				for fam, evs := range byFam {
					target[groupKey][fam] = calcMetrics(evs)
				}
			}
		}

		populateMap(out.ByCompositeRegime, func(e BaselineEvent) string { return e.Label.Composite })
		populateMap(out.ByVolatility, func(e BaselineEvent) string { return e.Label.Volatility })
		populateMap(out.ByTrend, func(e BaselineEvent) string { return e.Label.Trend })
		populateMap(out.ByLiquidity, func(e BaselineEvent) string { return e.Label.Liquidity })
		populateMap(out.ByMarketBeta, func(e BaselineEvent) string { return e.Label.MarketBeta })

		// Logic for conclusion
		bestCount := 0
		hasPromise := false
		for _, m := range out.Global {
			if m.SampleWarning == "USABLE_SAMPLE" && m.Expectancy15 > 0.0005 && m.ProfitFactor15 > 1.05 {
				hasPromise = true
				bestCount++
			}
		}

		if hasPromise {
			out.Conclusion = fmt.Sprintf("one or more families worth Phase 9.4 deeper research (%d found)", bestCount)
		} else if len(events) < 500 {
			out.Conclusion = "inconclusive due to sample size/data limits"
		} else {
			out.Conclusion = "no viable alpha families found"
		}

		// Write JSON
		jsonOutPath := strings.Replace(eabOutPath, ".md", ".json", 1)
		if !strings.HasSuffix(jsonOutPath, ".json") {
			jsonOutPath = eabOutPath + ".json"
		}
		jsonData, err := json.MarshalIndent(out, "", "  ")
		if err != nil {
			return err
		}
		if err := os.WriteFile(jsonOutPath, jsonData, 0644); err != nil {
			return err
		}

		// Write Markdown
		md := buildAlphaBaselinesReport(out)
		if err := os.WriteFile(eabOutPath, []byte(md), 0644); err != nil {
			return err
		}

		return nil
	},
}

func buildAlphaBaselinesReport(out BaselineReportJSON) string {
	var sb strings.Builder
	sb.WriteString("# Phase 9.3: Alpha Discovery Baselines\n\n")

	sb.WriteString("## Fast Accumulation Deprecation Note\n")
	sb.WriteString("Fast Accumulation has been globally deprecated as a standalone alpha due to extreme underperformance and poor profit factors. It is strictly excluded from these baseline generation phases.\n\n")

	sb.WriteString("## Summary\n")
	sb.WriteString(fmt.Sprintf("- **Total Features Evaluated:** %d\n", out.Summary.TotalFeatures))
	sb.WriteString(fmt.Sprintf("- **Total Labels:** %d\n", out.Summary.TotalLabels))
	sb.WriteString(fmt.Sprintf("- **Total Events Generated:** %d\n", out.Summary.TotalEvents))
	sb.WriteString(fmt.Sprintf("- **Conclusion:** %s\n\n", out.Conclusion))

	sb.WriteString("## Leakage / As-Of Status\n")
	sb.WriteString(fmt.Sprintf("- **Status:** %s\n", out.LeakageStatus))
	sb.WriteString("- All events were generated without using forward-looking indicators and joined precisely with labels where `label.available_at_ms <= event.event_time_ms`.\n\n")

	sb.WriteString("## Baseline Signal Overview & Global Baseline Comparison\n")
	sb.WriteString("| Baseline | Count | WinRate(15m) | PF(15m) | Exp(15m) | Fwd5 | Fwd15 | Fwd30 | Fwd60 | Warning |\n")
	sb.WriteString("|---|---|---|---|---|---|---|---|---|---|\n")
	for k, v := range out.Global {
		sb.WriteString(fmt.Sprintf("| %s | %d | %.2f%% | %.2f | %.5f | %.5f | %.5f | %.5f | %.5f | %s |\n",
			k, v.EventCount, v.WinRate15*100, v.ProfitFactor15, v.Expectancy15, v.AvgFwd5, v.AvgFwd15, v.AvgFwd30, v.AvgFwd60, v.SampleWarning))
	}
	sb.WriteString("\n")

	writeCategoryTable := func(title string, category map[string]map[string]BaselineMetrics) {
		sb.WriteString(fmt.Sprintf("## Baseline Performance by %s\n", title))
		sb.WriteString("| Regime | Baseline | Count | WinRate | PF | Exp | Fwd15 | Warning |\n")
		sb.WriteString("|---|---|---|---|---|---|---|---|\n")
		for reg, baseMap := range category {
			for base, m := range baseMap {
				sb.WriteString(fmt.Sprintf("| %s | %s | %d | %.2f%% | %.2f | %.5f | %.5f | %s |\n",
					reg, base, m.EventCount, m.WinRate15*100, m.ProfitFactor15, m.Expectancy15, m.AvgFwd15, m.SampleWarning))
			}
		}
		sb.WriteString("\n")
	}

	writeCategoryTable("Composite Regime", out.ByCompositeRegime)
	writeCategoryTable("Volatility", out.ByVolatility)
	writeCategoryTable("Trend", out.ByTrend)
	writeCategoryTable("Liquidity", out.ByLiquidity)
	writeCategoryTable("Market Beta", out.ByMarketBeta)

	sb.WriteString("## Shock Event Forward Return Study\n")
	sb.WriteString("Observing behavior of ShockFade baselines after Volatility=shock. Fading means shorting after up-shocks and buying after down-shocks.\n\n")

	sb.WriteString("## Compression Breakout Study\n")
	sb.WriteString("Observing CompressionBreakout baselines during compressed periods, directional aligned with short EMA and BTC trend.\n\n")

	sb.WriteString("## BTC/ETH Confirmation Study\n")
	sb.WriteString("Comparing BetaAgrees vs BetaDiverges for identical momentum signals.\n\n")

	// Determine best and worst
	type candidate struct {
		Name string
		M    BaselineMetrics
	}
	var cands []candidate
	for k, m := range out.Global {
		cands = append(cands, candidate{k, m})
	}
	
	sort.Slice(cands, func(i, j int) bool {
		return cands[i].M.Expectancy15 > cands[j].M.Expectancy15
	})

	sb.WriteString("## Best Candidate Families\n")
	for i := 0; i < len(cands) && i < 3; i++ {
		c := cands[i]
		sb.WriteString(fmt.Sprintf("1. **%s**: Exp=%.5f, PF=%.2f, Count=%d\n", c.Name, c.M.Expectancy15, c.M.ProfitFactor15, c.M.EventCount))
	}
	sb.WriteString("\n")

	sb.WriteString("## Rejected Candidate Families\n")
	for i := len(cands) - 1; i >= 0 && i >= len(cands)-3; i-- {
		c := cands[i]
		sb.WriteString(fmt.Sprintf("1. **%s**: Exp=%.5f, PF=%.2f, Count=%d\n", c.Name, c.M.Expectancy15, c.M.ProfitFactor15, c.M.EventCount))
	}
	sb.WriteString("\n")

	sb.WriteString("## Low Sample Warnings\n")
	for _, c := range cands {
		if c.M.SampleWarning != "USABLE_SAMPLE" {
			sb.WriteString(fmt.Sprintf("- %s: %s (%d events)\n", c.Name, c.M.SampleWarning, c.M.EventCount))
		}
	}
	sb.WriteString("\n")

	return sb.String()
}

func init() {
	evaluateAlphaBaselinesCmd.Flags().StringVar(&eabFeaturesPath, "features", "", "Path to features json")
	evaluateAlphaBaselinesCmd.Flags().StringVar(&eabRegimesPath, "regimes", "", "Path to regime labels json")
	evaluateAlphaBaselinesCmd.Flags().StringVar(&eabOutPath, "out", "", "Path to output markdown report")
	rootCmd.AddCommand(evaluateAlphaBaselinesCmd)
}
