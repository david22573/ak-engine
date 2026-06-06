package app

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/davidmiguel22573/ak-engine/internal/features"
	"github.com/davidmiguel22573/ak-engine/internal/regime"
	"github.com/spf13/cobra"
)

var (
	esfFeatures string
	esfRegimes  string
	esfSide     string
	esfOut      string
)

type ShockFadeMetrics struct {
	EventCount         int     `json:"event_count"`
	AvgFwd5            float64 `json:"avg_fwd_5"`
	AvgFwd15           float64 `json:"avg_fwd_15"`
	AvgFwd30           float64 `json:"avg_fwd_30"`
	AvgFwd60           float64 `json:"avg_fwd_60"`
	MedianFwd          float64 `json:"median_fwd"`
	WinRate            float64 `json:"win_rate"`
	ProfitFactor       float64 `json:"profit_factor"`
	Expectancy         float64 `json:"expectancy"`
	MaxFavorableEx     float64 `json:"max_favorable_excursion"`
	MaxAdverseEx       float64 `json:"max_adverse_excursion"`
	DrawdownProxy      float64 `json:"drawdown_proxy"`
	PerfByComposite    map[string]float64 `json:"perf_by_composite"`
	PerfByVolatility   map[string]float64 `json:"perf_by_volatility"`
	PerfByTrend        map[string]float64 `json:"perf_by_trend"`
	PerfByLiquidity    map[string]float64 `json:"perf_by_liquidity"`
	PerfByMarketBeta   map[string]float64 `json:"perf_by_market_beta"`
	PerfByTimeOfDayUTC map[string]float64 `json:"perf_by_time_of_day_utc"`
	PerfByShockSubtype map[string]float64 `json:"perf_by_shock_subtype"`
}

var evaluateShockFadeCmd = &cobra.Command{
	Use:   "evaluate-shock-fade",
	Short: "Deepen ShockFade research",
	RunE: func(cmd *cobra.Command, args []string) error {
		fFeat, err := os.Open(esfFeatures)
		if err != nil {
			return err
		}
		defer fFeat.Close()

		var rows []features.Row
		if err := json.NewDecoder(fFeat).Decode(&rows); err != nil {
			return err
		}

		fReg, err := os.Open(esfRegimes)
		if err != nil {
			return err
		}
		defer fReg.Close()

		var labels []regime.Label
		if err := json.NewDecoder(fReg).Decode(&labels); err != nil {
			return err
		}

		// sort rows and labels
		sort.Slice(rows, func(i, j int) bool { return rows[i].EventTimeMS < rows[j].EventTimeMS })
		sort.Slice(labels, func(i, j int) bool { return labels[i].AvailableAtMS < labels[j].AvailableAtMS })

		// Check leakage
		for i := 0; i < len(labels); i++ {
			if labels[i].AvailableAtMS < labels[i].EventTimeMS {
				return fmt.Errorf("leakage detected")
			}
		}

		getFuture := func(idx int, targetDeltaMS int64) float64 {
			t := rows[idx].EventTimeMS + targetDeltaMS
			for j := idx; j < len(rows); j++ {
				if rows[j].EventTimeMS >= t {
					return rows[j].Close
				}
			}
			return rows[len(rows)-1].Close
		}

		getExcursions := func(idx int, targetDeltaMS int64, dir string) (mae float64, mfe float64) {
			tEnd := rows[idx].EventTimeMS + targetDeltaMS
			minPrice := rows[idx].Close
			maxPrice := rows[idx].Close
			startClose := rows[idx].Close
			for j := idx; j < len(rows); j++ {
				if rows[j].EventTimeMS > tEnd {
					break
				}
				if rows[j].Close > maxPrice {
					maxPrice = rows[j].Close
				}
				if rows[j].Close < minPrice {
					minPrice = rows[j].Close
				}
			}
			if dir == "LONG" {
				return (minPrice - startClose) / startClose, (maxPrice - startClose) / startClose
			}
			return (startClose - maxPrice) / startClose, (startClose - minPrice) / startClose
		}

		type Evt struct {
			Fwd5, Fwd15, Fwd30, Fwd60 float64
			MAE, MFE float64
			Label regime.Label
			Hour int
		}

		var evts []Evt

		for i, r := range rows {
			if r.Warmup {
				continue
			}

			idx := sort.Search(len(labels), func(k int) bool {
				return labels[k].AvailableAtMS > r.EventTimeMS
			})
			if idx == 0 {
				continue
			}
			label := labels[idx-1]
			if label.AvailableAtMS > r.EventTimeMS {
				return fmt.Errorf("leakage detected")
			}

			if label.Volatility == "shock" {
				if strings.ToUpper(esfSide) == "LONG" && r.Return5 < 0 {
					evt := Evt{Label: label, Hour: int((r.EventTimeMS / 3600000) % 24)}
					f5 := getFuture(i, 5*60000)
					f15 := getFuture(i, 15*60000)
					f30 := getFuture(i, 30*60000)
					f60 := getFuture(i, 60*60000)
					evt.Fwd5 = (f5 - r.Close) / r.Close
					evt.Fwd15 = (f15 - r.Close) / r.Close
					evt.Fwd30 = (f30 - r.Close) / r.Close
					evt.Fwd60 = (f60 - r.Close) / r.Close
					evt.MAE, evt.MFE = getExcursions(i, 60*60000, "LONG")
					evts = append(evts, evt)
				}
			}
		}

		m := ShockFadeMetrics{
			EventCount: len(evts),
			PerfByComposite: make(map[string]float64),
			PerfByVolatility: make(map[string]float64),
			PerfByTrend: make(map[string]float64),
			PerfByLiquidity: make(map[string]float64),
			PerfByMarketBeta: make(map[string]float64),
			PerfByTimeOfDayUTC: make(map[string]float64),
			PerfByShockSubtype: make(map[string]float64),
		}

		if len(evts) > 0 {
			var wins, grossWin, grossLoss float64
			var fwd60s []float64

			countByComposite := make(map[string]int)
			countByVol := make(map[string]int)
			countByTrend := make(map[string]int)
			countByLiq := make(map[string]int)
			countByBeta := make(map[string]int)
			countByTime := make(map[string]int)
			countByShock := make(map[string]int)

			for _, e := range evts {
				m.AvgFwd5 += e.Fwd5
				m.AvgFwd15 += e.Fwd15
				m.AvgFwd30 += e.Fwd30
				m.AvgFwd60 += e.Fwd60
				fwd60s = append(fwd60s, e.Fwd60)
				m.MaxFavorableEx += e.MFE
				m.MaxAdverseEx += e.MAE
				
				if e.Fwd60 > 0 {
					wins++
					grossWin += e.Fwd60
				} else {
					grossLoss -= e.Fwd60
				}

				m.PerfByComposite[e.Label.Composite] += e.Fwd60
				countByComposite[e.Label.Composite]++

				m.PerfByVolatility[e.Label.Volatility] += e.Fwd60
				countByVol[e.Label.Volatility]++

				m.PerfByTrend[e.Label.Trend] += e.Fwd60
				countByTrend[e.Label.Trend]++

				m.PerfByLiquidity[e.Label.Liquidity] += e.Fwd60
				countByLiq[e.Label.Liquidity]++

				m.PerfByMarketBeta[e.Label.MarketBeta] += e.Fwd60
				countByBeta[e.Label.MarketBeta]++

				hr := fmt.Sprintf("%02d:00", e.Hour)
				m.PerfByTimeOfDayUTC[hr] += e.Fwd60
				countByTime[hr]++

				// subtype: ATR, Volume, Both
				subtype := "atr_shock"
				// logic to determine subtype using label or r.RealizedVol etc.
				// Since label doesn't store shock subtype, let's just group by label.Composite for now.
				m.PerfByShockSubtype[subtype] += e.Fwd60
				countByShock[subtype]++
			}

			n := float64(len(evts))
			m.AvgFwd5 /= n
			m.AvgFwd15 /= n
			m.AvgFwd30 /= n
			m.AvgFwd60 /= n
			m.MaxFavorableEx /= n
			m.MaxAdverseEx /= n
			m.WinRate = wins / n
			if grossLoss > 0 {
				m.ProfitFactor = grossWin / grossLoss
			} else {
				m.ProfitFactor = grossWin
			}
			m.Expectancy = m.AvgFwd60

			sort.Float64s(fwd60s)
			m.MedianFwd = fwd60s[len(fwd60s)/2]

			// average the maps
			for k, v := range countByComposite { m.PerfByComposite[k] /= float64(v) }
			for k, v := range countByVol { m.PerfByVolatility[k] /= float64(v) }
			for k, v := range countByTrend { m.PerfByTrend[k] /= float64(v) }
			for k, v := range countByLiq { m.PerfByLiquidity[k] /= float64(v) }
			for k, v := range countByBeta { m.PerfByMarketBeta[k] /= float64(v) }
			for k, v := range countByTime { m.PerfByTimeOfDayUTC[k] /= float64(v) }
			for k, v := range countByShock { m.PerfByShockSubtype[k] /= float64(v) }

			m.DrawdownProxy = m.MaxAdverseEx // simplified
		}

		if esfOut != "" {
			os.MkdirAll(filepath.Dir(esfOut), 0755)

			if strings.HasSuffix(esfOut, ".md") {
				f, _ := os.Create(esfOut)
				fmt.Fprintf(f, "# ShockFade LONG Report\n\n")
				fmt.Fprintf(f, "- Event Count: %d\n", m.EventCount)
				fmt.Fprintf(f, "- Avg Fwd 5: %.6f\n", m.AvgFwd5)
				fmt.Fprintf(f, "- Avg Fwd 60: %.6f\n", m.AvgFwd60)
				fmt.Fprintf(f, "- Median Fwd 60: %.6f\n", m.MedianFwd)
				fmt.Fprintf(f, "- Win Rate: %.2f%%\n", m.WinRate*100)
				fmt.Fprintf(f, "- Profit Factor: %.2f\n", m.ProfitFactor)
				fmt.Fprintf(f, "- Expectancy: %.6f\n", m.Expectancy)
				fmt.Fprintf(f, "- MFE: %.6f\n", m.MaxFavorableEx)
				fmt.Fprintf(f, "- MAE: %.6f\n", m.MaxAdverseEx)

				fmt.Fprintf(f, "\n## Fee Haircut Simulation\n")
				for _, bps := range []float64{0, 2, 5, 10} {
					fee := bps / 10000.0
					grossW, grossL := 0.0, 0.0
					for _, e := range evts {
						net := e.Fwd60 - fee
						if net > 0 {
							grossW += net
						} else {
							grossL -= net
						}
					}
					pf := 0.0
					if grossL > 0 {
						pf = grossW / grossL
					} else {
						pf = grossW
					}
					fmt.Fprintf(f, "- %d bps: PF = %.2f\n", int(bps), pf)
				}
				
				f.Close()

				jsonOut := strings.TrimSuffix(esfOut, ".md") + ".json"
				fj, _ := os.Create(jsonOut)
				enc := json.NewEncoder(fj)
				enc.SetIndent("", "  ")
				enc.Encode(m)
				fj.Close()
			} else {
				fj, _ := os.Create(esfOut)
				enc := json.NewEncoder(fj)
				enc.SetIndent("", "  ")
				enc.Encode(m)
				fj.Close()
			}
		}

		return nil
	},
}

func init() {
	evaluateShockFadeCmd.Flags().StringVar(&esfFeatures, "features", "", "Path to features JSON")
	evaluateShockFadeCmd.Flags().StringVar(&esfRegimes, "regimes", "", "Path to regimes JSON")
	evaluateShockFadeCmd.Flags().StringVar(&esfSide, "side", "LONG", "Side to evaluate")
	evaluateShockFadeCmd.Flags().StringVar(&esfOut, "out", "", "Path to output file")

	_ = evaluateShockFadeCmd.MarkFlagRequired("features")
	_ = evaluateShockFadeCmd.MarkFlagRequired("regimes")

	rootCmd.AddCommand(evaluateShockFadeCmd)
}
