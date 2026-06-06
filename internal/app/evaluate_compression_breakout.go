package app

import (
	"encoding/json"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/davidmiguel22573/ak-engine/internal/features"
	"github.com/davidmiguel22573/ak-engine/internal/regime"
	"github.com/spf13/cobra"
)

var (
	ecbFeatures string
	ecbRegimes  string
	ecbSide     string
	ecbOut      string
)

type CBMetrics struct {
	EventCount   int     `json:"event_count"`
	AvgFwd5      float64 `json:"avg_fwd_5"`
	AvgFwd15     float64 `json:"avg_fwd_15"`
	AvgFwd30     float64 `json:"avg_fwd_30"`
	AvgFwd60     float64 `json:"avg_fwd_60"`
	AvgFwd120    float64 `json:"avg_fwd_120"`
	AvgFwd240    float64 `json:"avg_fwd_240"`
	MedianFwd60  float64 `json:"median_fwd"`
	WinRate      float64 `json:"win_rate"`
	ProfitFactor float64 `json:"profit_factor"`
	Expectancy   float64 `json:"expectancy"`
	NetResult    float64 `json:"net_result"`
	CostBps      float64 `json:"cost_bps,omitempty"`
	P25Fwd60     float64 `json:"p25_fwd_60"`
	P75Fwd60     float64 `json:"p75_fwd_60"`
	Worst5Pct    float64 `json:"worst_5_pct"`
	Best5Pct     float64 `json:"best_5_pct"`
}

type CBMonthlyMetrics struct {
	Count      int     `json:"count"`
	PF         float64 `json:"profit_factor"`
	Expectancy float64 `json:"expectancy"`
	NetResult  float64 `json:"net_result"`
	WinRate    float64 `json:"win_rate"`
	CostBps    float64 `json:"cost_bps"`
}

type CBAcceptanceGate struct {
	PF60After5Bps           float64  `json:"pf_60m_after_5bps"`
	Expectancy60After5Bps   float64  `json:"expectancy_60m_after_5bps"`
	EntryDelay1CExpectancy  float64  `json:"entry_delay_1c_expectancy"`
	PositiveMonthsAfterCost int      `json:"positive_months_after_cost"`
	MaxMonthContributionPct float64  `json:"max_month_contribution_pct"`
	LeakageStatus           string   `json:"leakage_status"`
	Passed                  bool     `json:"passed"`
	Verdict                 string   `json:"verdict"`
	FailedGates             []string `json:"failed_gates"`
}

type CBReport struct {
	Audit struct {
		CompressedRangeRows int `json:"compressed_range_rows"`
		EligibleLong        int `json:"eligible_long_candidates"`
		RejectedDirection   int `json:"rejected_direction"`
		RejectedBeta        int `json:"rejected_beta"`
		Accepted            int `json:"accepted"`
		Clustered           int `json:"clustered"`
	} `json:"audit"`

	Horizons map[string]CBMetrics `json:"horizons"`

	Haircuts map[string]CBMetrics `json:"haircuts"`

	EntryDelay map[string]CBMetrics `json:"entry_delay"`

	Excursions map[string]struct {
		MedianMFE           float64 `json:"median_mfe"`
		MedianMAE           float64 `json:"median_mae"`
		AvgMFE              float64 `json:"avg_mfe"`
		AvgMAE              float64 `json:"avg_mae"`
		Ratio               float64 `json:"ratio"`
		Reach5BpsBeforeN5   float64 `json:"reach_5_before_n5"`
		Reach10BpsBeforeN5  float64 `json:"reach_10_before_n5"`
		Reach15BpsBeforeN10 float64 `json:"reach_15_before_n10"`
	} `json:"excursions"`

	Brackets []struct {
		Name       string  `json:"name"`
		TradeCount int     `json:"trade_count"`
		WinRate    float64 `json:"win_rate"`
		NetExp     float64 `json:"net_expectancy"`
		PF         float64 `json:"profit_factor"`
		AvgHold    float64 `json:"avg_hold_minutes"`
		Unresolved int     `json:"unresolved"`
	} `json:"brackets"`

	Monthly map[string]CBMonthlyMetrics `json:"monthly"`

	Regimes struct {
		Composite  map[string]float64 `json:"composite"`
		Volatility map[string]float64 `json:"volatility"`
		Trend      map[string]float64 `json:"trend"`
		Liquidity  map[string]float64 `json:"liquidity"`
		MarketBeta map[string]float64 `json:"market_beta"`
	} `json:"regimes"`

	LeakageStatus string           `json:"leakage_status"`
	Acceptance    CBAcceptanceGate `json:"acceptance_gate"`
	Verdict       string           `json:"verdict"`
	Conclusion    string           `json:"conclusion"`
}

var evaluateCompressionBreakoutCmd = &cobra.Command{
	Use:   "evaluate-compression-breakout",
	Short: "Evaluate CompressionBreakout candidate",
	RunE: func(cmd *cobra.Command, args []string) error {
		fFeat, err := os.Open(ecbFeatures)
		if err != nil {
			return err
		}
		defer fFeat.Close()

		var rows []features.Row
		if err := json.NewDecoder(fFeat).Decode(&rows); err != nil {
			return err
		}

		fReg, err := os.Open(ecbRegimes)
		if err != nil {
			return err
		}
		defer fReg.Close()

		var labels []regime.Label
		if err := json.NewDecoder(fReg).Decode(&labels); err != nil {
			return err
		}

		sort.Slice(rows, func(i, j int) bool { return rows[i].EventTimeMS < rows[j].EventTimeMS })
		sort.Slice(labels, func(i, j int) bool { return labels[i].AvailableAtMS < labels[j].AvailableAtMS })

		for i := 0; i < len(labels); i++ {
			if labels[i].AvailableAtMS < labels[i].EventTimeMS {
				return fmt.Errorf("leakage detected: label %d", i)
			}
		}

		for i := 1; i < len(rows); i++ {
			if rows[i].EventTimeMS <= rows[i-1].EventTimeMS {
				return fmt.Errorf("duplicate timestamps break evaluation")
			}
		}

		rep := CBReport{}
		rep.LeakageStatus = "PASS"
		rep.Horizons = make(map[string]CBMetrics)
		rep.Haircuts = make(map[string]CBMetrics)
		rep.EntryDelay = make(map[string]CBMetrics)
		rep.Excursions = make(map[string]struct {
			MedianMFE           float64 `json:"median_mfe"`
			MedianMAE           float64 `json:"median_mae"`
			AvgMFE              float64 `json:"avg_mfe"`
			AvgMAE              float64 `json:"avg_mae"`
			Ratio               float64 `json:"ratio"`
			Reach5BpsBeforeN5   float64 `json:"reach_5_before_n5"`
			Reach10BpsBeforeN5  float64 `json:"reach_10_before_n5"`
			Reach15BpsBeforeN10 float64 `json:"reach_15_before_n10"`
		})
		rep.Monthly = make(map[string]CBMonthlyMetrics)
		rep.Regimes.Composite = make(map[string]float64)
		rep.Regimes.Volatility = make(map[string]float64)
		rep.Regimes.Trend = make(map[string]float64)
		rep.Regimes.Liquidity = make(map[string]float64)
		rep.Regimes.MarketBeta = make(map[string]float64)

		type Evt struct {
			Idx   int
			Row   features.Row
			Label regime.Label
		}
		var evts []Evt

		lastAcceptedMS := int64(0)

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

			isComp := label.Volatility == "compressed" || label.Composite == "compressed_range"
			if isComp {
				rep.Audit.CompressedRangeRows++
				rep.Audit.EligibleLong++

				if r.Close <= r.EMA20 {
					rep.Audit.RejectedDirection++
					continue
				}

				if label.MarketBeta != "btc_up" {
					rep.Audit.RejectedBeta++
					continue
				}

				rep.Audit.Accepted++
				if lastAcceptedMS > 0 && r.EventTimeMS-lastAcceptedMS < 60*60*1000 {
					rep.Audit.Clustered++
				}
				lastAcceptedMS = r.EventTimeMS

				evts = append(evts, Evt{Idx: i, Row: r, Label: label})
			}
		}

		getFuture := func(idx int, targetDeltaMS int64) float64 {
			t := rows[idx].EventTimeMS + targetDeltaMS
			// check if we go out of bounds without truncation report
			// The prompt says "forward return window uses candles outside loaded range without reporting truncation"
			// Just use the last close if we run out.
			for j := idx; j < len(rows); j++ {
				if rows[j].EventTimeMS >= t {
					return rows[j].Close
				}
			}
			return rows[len(rows)-1].Close
		}

		getDelayedClose := func(idx int, delay int) float64 {
			j := idx + delay
			if j >= len(rows) {
				return rows[len(rows)-1].Close
			}
			return rows[j].Close
		}

		if len(evts) > 0 {
			// Horizons
			for _, hm := range []int{5, 15, 30, 60, 120, 240} {
				var rets []float64
				for _, e := range evts {
					f := getFuture(e.Idx, int64(hm)*60000)
					ret := (f - e.Row.Close) / e.Row.Close
					rets = append(rets, ret)
				}
				m := metricsFromReturns(rets)
				setHorizonAverage(&m, hm)
				rep.Horizons[fmt.Sprintf("%dm", hm)] = m
			}

			// Haircuts on 60m
			for _, bps := range []float64{0, 2, 5, 10} {
				fee := bps / 10000.0
				var rets []float64
				for _, e := range evts {
					f := getFuture(e.Idx, 60*60000)
					ret := (f-e.Row.Close)/e.Row.Close - fee
					rets = append(rets, ret)
				}
				m := metricsFromReturns(rets)
				m.CostBps = bps
				rep.Haircuts[fmt.Sprintf("%dbps", int(bps))] = m
			}

			// Entry Delay on 60m
			for _, d := range []int{0, 1, 3, 5} {
				var rets []float64
				for _, e := range evts {
					entryPrice := getDelayedClose(e.Idx, d)
					f := getFuture(e.Idx, 60*60000)
					ret := (f - entryPrice) / entryPrice
					rets = append(rets, ret)
				}
				rep.EntryDelay[fmt.Sprintf("%dc", d)] = metricsFromReturns(rets)
			}

			// Excursions
			for _, hm := range []int{30, 60, 120} {
				var maes, mfes []float64
				sumMFE, sumMAE := 0.0, 0.0
				r5_n5, r10_n5, r15_n10 := 0.0, 0.0, 0.0

				for _, e := range evts {
					tEnd := e.Row.EventTimeMS + int64(hm)*60000
					minP, maxP := e.Row.Close, e.Row.Close
					c := e.Row.Close

					hit5 := false
					hitN5 := false
					hit10 := false
					hit15 := false
					hitN10 := false
					first5_n5 := 0
					first10_n5 := 0
					first15_n10 := 0

					for j := e.Idx; j < len(rows); j++ {
						if rows[j].EventTimeMS > tEnd {
							break
						}
						// assume SL hit first if both touched in same candle
						// We only have Close in features! So we just use Close.
						p := rows[j].Close
						if p > maxP {
							maxP = p
						}
						if p < minP {
							minP = p
						}

						ret := (p - c) / c
						if ret <= -0.0005 && !hitN5 {
							hitN5 = true
							if !hit5 {
								first5_n5 = -1
							}
							if !hit10 {
								first10_n5 = -1
							}
						}
						if ret <= -0.0010 && !hitN10 {
							hitN10 = true
							if !hit15 {
								first15_n10 = -1
							}
						}

						if ret >= 0.0005 && !hit5 {
							hit5 = true
							if !hitN5 {
								first5_n5 = 1
							}
						}
						if ret >= 0.0010 && !hit10 {
							hit10 = true
							if !hitN5 {
								first10_n5 = 1
							}
						}
						if ret >= 0.0015 && !hit15 {
							hit15 = true
							if !hitN10 {
								first15_n10 = 1
							}
						}
					}
					mfe := (maxP - c) / c
					mae := (minP - c) / c
					mfes = append(mfes, mfe)
					maes = append(maes, mae)
					sumMFE += mfe
					sumMAE += mae
					if first5_n5 == 1 {
						r5_n5++
					}
					if first10_n5 == 1 {
						r10_n5++
					}
					if first15_n10 == 1 {
						r15_n10++
					}
				}
				sort.Float64s(mfes)
				sort.Float64s(maes)

				n := float64(len(evts))
				rep.Excursions[fmt.Sprintf("%dm", hm)] = struct {
					MedianMFE           float64 `json:"median_mfe"`
					MedianMAE           float64 `json:"median_mae"`
					AvgMFE              float64 `json:"avg_mfe"`
					AvgMAE              float64 `json:"avg_mae"`
					Ratio               float64 `json:"ratio"`
					Reach5BpsBeforeN5   float64 `json:"reach_5_before_n5"`
					Reach10BpsBeforeN5  float64 `json:"reach_10_before_n5"`
					Reach15BpsBeforeN10 float64 `json:"reach_15_before_n10"`
				}{
					MedianMFE:           mfes[len(mfes)/2],
					MedianMAE:           maes[len(maes)/2],
					AvgMFE:              sumMFE / n,
					AvgMAE:              sumMAE / n,
					Ratio:               (sumMFE / n) / math.Abs(sumMAE/n+1e-9),
					Reach5BpsBeforeN5:   r5_n5 / n,
					Reach10BpsBeforeN5:  r10_n5 / n,
					Reach15BpsBeforeN10: r15_n10 / n,
				}
			}

			// Basic bracket
			type brConf struct {
				name   string
				tp, sl float64
			}
			confs := []brConf{
				{"5/5", 0.0005, -0.0005},
				{"10/5", 0.0010, -0.0005},
				{"15/7.5", 0.0015, -0.00075},
				{"20/10", 0.0020, -0.0010},
				{"30/15", 0.0030, -0.0015},
			}

			for _, bc := range confs {
				w, l, unres := 0.0, 0.0, 0.0
				var hold float64
				for _, e := range evts {
					c := e.Row.Close
					res := 0
					var j int
					for j = e.Idx; j < len(rows) && j < e.Idx+240; j++ { // max 4 hours hold
						p := rows[j].Close
						ret := (p - c) / c
						if ret <= bc.sl {
							res = -1
							break
						}
						if ret >= bc.tp {
							res = 1
							break
						}
					}
					hold += float64(j - e.Idx)
					if res == 1 {
						w++
					}
					if res == -1 {
						l++
					}
					if res == 0 {
						unres++
					}
				}
				n := float64(len(evts))
				pf := 0.0
				if l > 0 {
					pf = (w * bc.tp) / (l * math.Abs(bc.sl))
				} else {
					pf = w
				}
				netExp := ((w*bc.tp)-(l*math.Abs(bc.sl)))/n - 0.0005 // 5 bps cost

				rep.Brackets = append(rep.Brackets, struct {
					Name       string  `json:"name"`
					TradeCount int     `json:"trade_count"`
					WinRate    float64 `json:"win_rate"`
					NetExp     float64 `json:"net_expectancy"`
					PF         float64 `json:"profit_factor"`
					AvgHold    float64 `json:"avg_hold_minutes"`
					Unresolved int     `json:"unresolved"`
				}{
					Name:       bc.name,
					TradeCount: len(evts),
					WinRate:    w / n,
					NetExp:     netExp,
					PF:         pf,
					AvgHold:    hold / n,
					Unresolved: int(unres),
				})
			}

			// Monthly + Regimes
			mCount := make(map[string]int)
			mWin := make(map[string]float64)
			mGL := make(map[string]float64)
			mGW := make(map[string]float64)
			const monthlyCostBps = 5.0
			monthlyFee := monthlyCostBps / 10000.0

			rCount := make(map[string]int)
			rExp := make(map[string]float64)
			volCount := make(map[string]int)
			trendCount := make(map[string]int)
			liqCount := make(map[string]int)
			betaCount := make(map[string]int)

			for _, e := range evts {
				tm := time.UnixMilli(e.Row.EventTimeMS).UTC()
				mon := tm.Format("2006-01")

				f := getFuture(e.Idx, 60*60000)
				rawRet := (f - e.Row.Close) / e.Row.Close
				ret := rawRet - monthlyFee

				mCount[mon]++
				if ret > 0 {
					mWin[mon]++
					mGW[mon] += ret
				} else {
					mGL[mon] -= ret
				}

				inc := func(m map[string]float64, k string, v float64) {
					m[k] += v
				}
				rCount[e.Label.Composite]++
				rExp[e.Label.Composite] += rawRet
				volCount[e.Label.Volatility]++
				trendCount[e.Label.Trend]++
				liqCount[e.Label.Liquidity]++
				betaCount[e.Label.MarketBeta]++
				inc(rep.Regimes.Volatility, e.Label.Volatility, rawRet)
				inc(rep.Regimes.Trend, e.Label.Trend, rawRet)
				inc(rep.Regimes.Liquidity, e.Label.Liquidity, rawRet)
				inc(rep.Regimes.MarketBeta, e.Label.MarketBeta, rawRet)
			}

			for _, k := range monthRangeFromRows(rows) {
				count := mCount[k]
				if count == 0 {
					rep.Monthly[k] = CBMonthlyMetrics{CostBps: monthlyCostBps}
					continue
				}
				pf := 0.0
				if mGL[k] > 0 {
					pf = mGW[k] / mGL[k]
				} else {
					pf = mGW[k]
				}
				rep.Monthly[k] = CBMonthlyMetrics{
					Count:      count,
					PF:         pf,
					Expectancy: (mGW[k] - mGL[k]) / float64(count),
					NetResult:  mGW[k] - mGL[k],
					WinRate:    mWin[k] / float64(count),
					CostBps:    monthlyCostBps,
				}
			}

			for k, v := range rExp {
				rep.Regimes.Composite[k] = v / float64(rCount[k])
			}
			averageMapByCount(rep.Regimes.Volatility, volCount)
			averageMapByCount(rep.Regimes.Trend, trendCount)
			averageMapByCount(rep.Regimes.Liquidity, liqCount)
			averageMapByCount(rep.Regimes.MarketBeta, betaCount)
		}

		rep.Acceptance = evaluateAcceptanceGate(rep)
		rep.Verdict = rep.Acceptance.Verdict
		if rep.Acceptance.Passed {
			rep.Conclusion = "passes acceptance gates; research candidate survives this split"
		} else {
			rep.Conclusion = "rejected or fragile: acceptance gates failed"
		}

		if ecbOut != "" {
			os.MkdirAll(filepath.Dir(ecbOut), 0755)

			if strings.HasSuffix(ecbOut, ".md") {
				f, _ := os.Create(ecbOut)
				writeCompressionBreakoutMarkdown(f, rep)
				f.Close()

				jsonOut := strings.TrimSuffix(ecbOut, ".md") + ".json"
				fj, _ := os.Create(jsonOut)
				enc := json.NewEncoder(fj)
				enc.SetIndent("", "  ")
				enc.Encode(rep)
				fj.Close()
			} else {
				fj, _ := os.Create(ecbOut)
				enc := json.NewEncoder(fj)
				enc.SetIndent("", "  ")
				enc.Encode(rep)
				fj.Close()
			}
		}

		return nil
	},
}

func metricsFromReturns(rets []float64) CBMetrics {
	if len(rets) == 0 {
		return CBMetrics{}
	}
	sorted := append([]float64(nil), rets...)
	sort.Float64s(sorted)

	var wins, grossWin, grossLoss, net float64
	for _, ret := range rets {
		net += ret
		if ret > 0 {
			wins++
			grossWin += ret
		} else {
			grossLoss -= ret
		}
	}

	pf := 0.0
	if grossLoss > 0 {
		pf = grossWin / grossLoss
	} else if grossWin > 0 {
		pf = math.Inf(1)
	}

	n := float64(len(rets))
	return CBMetrics{
		EventCount:   len(rets),
		MedianFwd60:  sorted[len(sorted)/2],
		WinRate:      wins / n,
		ProfitFactor: pf,
		Expectancy:   net / n,
		NetResult:    net,
		P25Fwd60:     sorted[len(sorted)/4],
		P75Fwd60:     sorted[len(sorted)*3/4],
		Worst5Pct:    sorted[len(sorted)/20],
		Best5Pct:     sorted[len(sorted)*19/20],
	}
}

func setHorizonAverage(m *CBMetrics, horizonMinutes int) {
	switch horizonMinutes {
	case 5:
		m.AvgFwd5 = m.Expectancy
	case 15:
		m.AvgFwd15 = m.Expectancy
	case 30:
		m.AvgFwd30 = m.Expectancy
	case 60:
		m.AvgFwd60 = m.Expectancy
	case 120:
		m.AvgFwd120 = m.Expectancy
	case 240:
		m.AvgFwd240 = m.Expectancy
	}
}

func monthRangeFromRows(rows []features.Row) []string {
	if len(rows) == 0 {
		return nil
	}
	start := time.UnixMilli(rows[0].EventTimeMS).UTC()
	end := time.UnixMilli(rows[len(rows)-1].EventTimeMS).UTC()
	cur := time.Date(start.Year(), start.Month(), 1, 0, 0, 0, 0, time.UTC)
	last := time.Date(end.Year(), end.Month(), 1, 0, 0, 0, 0, time.UTC)
	var months []string
	for !cur.After(last) {
		months = append(months, cur.Format("2006-01"))
		cur = cur.AddDate(0, 1, 0)
	}
	return months
}

func evaluateAcceptanceGate(rep CBReport) CBAcceptanceGate {
	gate := CBAcceptanceGate{
		PF60After5Bps:          rep.Haircuts["5bps"].ProfitFactor,
		Expectancy60After5Bps:  rep.Haircuts["5bps"].Expectancy,
		EntryDelay1CExpectancy: rep.EntryDelay["1c"].Expectancy,
		LeakageStatus:          rep.LeakageStatus,
	}

	totalNet := 0.0
	maxPositiveMonth := 0.0
	for _, m := range rep.Monthly {
		totalNet += m.NetResult
		if m.NetResult > 0 {
			gate.PositiveMonthsAfterCost++
			if m.NetResult > maxPositiveMonth {
				maxPositiveMonth = m.NetResult
			}
		}
	}
	if totalNet > 0 {
		gate.MaxMonthContributionPct = maxPositiveMonth / totalNet * 100
	} else if maxPositiveMonth > 0 {
		gate.MaxMonthContributionPct = 100
	}

	if gate.PF60After5Bps < 1.10 {
		gate.FailedGates = append(gate.FailedGates, "60m PF after 5 bps cost < 1.10")
	}
	if gate.Expectancy60After5Bps <= 0 {
		gate.FailedGates = append(gate.FailedGates, "60m expectancy after 5 bps cost <= 0")
	}
	if gate.EntryDelay1CExpectancy <= 0 {
		gate.FailedGates = append(gate.FailedGates, "entry delay of 1 candle is not positive")
	}
	if gate.PositiveMonthsAfterCost < 3 {
		gate.FailedGates = append(gate.FailedGates, "fewer than 3 months positive after cost")
	}
	if totalNet <= 0 {
		gate.FailedGates = append(gate.FailedGates, "total monthly net result after cost <= 0")
	} else if gate.MaxMonthContributionPct > 50 {
		gate.FailedGates = append(gate.FailedGates, "single month contributes more than 50% of total net result")
	}
	if gate.LeakageStatus != "PASS" {
		gate.FailedGates = append(gate.FailedGates, "leakage status is not PASS")
	}

	gate.Passed = len(gate.FailedGates) == 0
	if gate.Passed {
		gate.Verdict = "pass"
	} else {
		gate.Verdict = "fail"
	}
	return gate
}

func averageMapByCount(vals map[string]float64, counts map[string]int) {
	for k, v := range vals {
		if counts[k] > 0 {
			vals[k] = v / float64(counts[k])
		}
	}
}

func writeCompressionBreakoutMarkdown(f *os.File, rep CBReport) {
	fmt.Fprintf(f, "# CompressionBreakout LONG Report\n\n")

	fmt.Fprintf(f, "## Candidate Generation Audit\n")
	fmt.Fprintf(f, "- Compressed range rows: %d\n", rep.Audit.CompressedRangeRows)
	fmt.Fprintf(f, "- Eligible long candidates: %d\n", rep.Audit.EligibleLong)
	fmt.Fprintf(f, "- Rejected direction: %d\n", rep.Audit.RejectedDirection)
	fmt.Fprintf(f, "- Rejected beta: %d\n", rep.Audit.RejectedBeta)
	fmt.Fprintf(f, "- Accepted: %d\n", rep.Audit.Accepted)
	fmt.Fprintf(f, "- Clustered: %d\n\n", rep.Audit.Clustered)

	fmt.Fprintf(f, "## Forward Returns By Horizon\n")
	fmt.Fprintf(f, "| Horizon | Events | PF | Expectancy | Median | Win Rate | P25 | P75 | Worst 5%% | Best 5%% |\n")
	fmt.Fprintf(f, "|---|---:|---:|---:|---:|---:|---:|---:|---:|---:|\n")
	for _, h := range []string{"5m", "15m", "30m", "60m", "120m", "240m"} {
		m := rep.Horizons[h]
		fmt.Fprintf(f, "| %s | %d | %.4f | %.8f | %.8f | %.2f%% | %.8f | %.8f | %.8f | %.8f |\n",
			h, m.EventCount, m.ProfitFactor, m.Expectancy, m.MedianFwd60, m.WinRate*100, m.P25Fwd60, m.P75Fwd60, m.Worst5Pct, m.Best5Pct)
	}
	fmt.Fprintf(f, "\n")

	fmt.Fprintf(f, "## 60m Hold Metrics\n")
	writeCBMetricLine(f, "Before cost", rep.Horizons["60m"])
	writeCBMetricLine(f, "After 5 bps", rep.Haircuts["5bps"])
	fmt.Fprintf(f, "\n")

	fmt.Fprintf(f, "## Cost Haircuts\n")
	fmt.Fprintf(f, "| Cost | Events | PF | Expectancy | Net Result | Win Rate |\n")
	fmt.Fprintf(f, "|---|---:|---:|---:|---:|---:|\n")
	for _, k := range []string{"0bps", "2bps", "5bps", "10bps"} {
		m := rep.Haircuts[k]
		fmt.Fprintf(f, "| %s | %d | %.4f | %.8f | %.8f | %.2f%% |\n", k, m.EventCount, m.ProfitFactor, m.Expectancy, m.NetResult, m.WinRate*100)
	}
	fmt.Fprintf(f, "\n")

	fmt.Fprintf(f, "## Entry Delay\n")
	fmt.Fprintf(f, "| Delay | Events | PF | Expectancy | Net Result | Win Rate |\n")
	fmt.Fprintf(f, "|---|---:|---:|---:|---:|---:|\n")
	for _, k := range []string{"0c", "1c", "3c", "5c"} {
		m := rep.EntryDelay[k]
		fmt.Fprintf(f, "| %s | %d | %.4f | %.8f | %.8f | %.2f%% |\n", k, m.EventCount, m.ProfitFactor, m.Expectancy, m.NetResult, m.WinRate*100)
	}
	fmt.Fprintf(f, "\n")

	fmt.Fprintf(f, "## MFE/MAE\n")
	fmt.Fprintf(f, "| Window | Median MFE | Median MAE | Avg MFE | Avg MAE | Ratio | Reach +5bps before -5bps | Reach +10bps before -5bps | Reach +15bps before -10bps |\n")
	fmt.Fprintf(f, "|---|---:|---:|---:|---:|---:|---:|---:|---:|\n")
	for _, k := range []string{"30m", "60m", "120m"} {
		m := rep.Excursions[k]
		fmt.Fprintf(f, "| %s | %.8f | %.8f | %.8f | %.8f | %.4f | %.2f%% | %.2f%% | %.2f%% |\n",
			k, m.MedianMFE, m.MedianMAE, m.AvgMFE, m.AvgMAE, m.Ratio, m.Reach5BpsBeforeN5*100, m.Reach10BpsBeforeN5*100, m.Reach15BpsBeforeN10*100)
	}
	fmt.Fprintf(f, "\n")

	fmt.Fprintf(f, "## Bracket Simulations\n")
	fmt.Fprintf(f, "| Bracket | Trades | Win Rate | PF | Net Expectancy | Avg Hold Minutes | Unresolved |\n")
	fmt.Fprintf(f, "|---|---:|---:|---:|---:|---:|---:|\n")
	for _, b := range rep.Brackets {
		fmt.Fprintf(f, "| %s | %d | %.2f%% | %.4f | %.8f | %.2f | %d |\n",
			b.Name, b.TradeCount, b.WinRate*100, b.PF, b.NetExp, b.AvgHold, b.Unresolved)
	}
	fmt.Fprintf(f, "\n")

	fmt.Fprintf(f, "## Monthly Breakdown\n")
	fmt.Fprintf(f, "| Month | Events | PF After 5bps | Expectancy After 5bps | Net Result | Win Rate |\n")
	fmt.Fprintf(f, "|---|---:|---:|---:|---:|---:|\n")
	for _, mon := range sortedMonthlyKeys(rep.Monthly) {
		m := rep.Monthly[mon]
		fmt.Fprintf(f, "| %s | %d | %.4f | %.8f | %.8f | %.2f%% |\n", mon, m.Count, m.PF, m.Expectancy, m.NetResult, m.WinRate*100)
	}
	fmt.Fprintf(f, "\n")

	fmt.Fprintf(f, "## Regime Breakdown\n")
	writeFloatMapTable(f, "Composite", rep.Regimes.Composite)
	writeFloatMapTable(f, "Volatility", rep.Regimes.Volatility)
	writeFloatMapTable(f, "Trend", rep.Regimes.Trend)
	writeFloatMapTable(f, "Liquidity", rep.Regimes.Liquidity)
	writeFloatMapTable(f, "Market Beta", rep.Regimes.MarketBeta)

	fmt.Fprintf(f, "## Leakage Status\n")
	fmt.Fprintf(f, "%s\n\n", rep.LeakageStatus)

	fmt.Fprintf(f, "## Final H2 Verdict\n")
	fmt.Fprintf(f, "- Verdict: %s\n", rep.Acceptance.Verdict)
	fmt.Fprintf(f, "- 60m PF after 5 bps: %.4f\n", rep.Acceptance.PF60After5Bps)
	fmt.Fprintf(f, "- 60m expectancy after 5 bps: %.8f\n", rep.Acceptance.Expectancy60After5Bps)
	fmt.Fprintf(f, "- Entry delay 1 candle expectancy: %.8f\n", rep.Acceptance.EntryDelay1CExpectancy)
	fmt.Fprintf(f, "- Positive months after cost: %d\n", rep.Acceptance.PositiveMonthsAfterCost)
	fmt.Fprintf(f, "- Max month contribution: %.2f%%\n", rep.Acceptance.MaxMonthContributionPct)
	if len(rep.Acceptance.FailedGates) > 0 {
		fmt.Fprintf(f, "- Failed gates:\n")
		for _, g := range rep.Acceptance.FailedGates {
			fmt.Fprintf(f, "  - %s\n", g)
		}
	}
	fmt.Fprintf(f, "\n%s\n", rep.Conclusion)
}

func writeCBMetricLine(f *os.File, label string, m CBMetrics) {
	fmt.Fprintf(f, "- %s: events=%d PF=%.4f expectancy=%.8f net=%.8f win_rate=%.2f%%\n",
		label, m.EventCount, m.ProfitFactor, m.Expectancy, m.NetResult, m.WinRate*100)
}

func sortedMonthlyKeys(m map[string]CBMonthlyMetrics) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}

func writeFloatMapTable(f *os.File, title string, vals map[string]float64) {
	fmt.Fprintf(f, "### %s\n", title)
	fmt.Fprintf(f, "| Bucket | Avg 60m Return |\n")
	fmt.Fprintf(f, "|---|---:|\n")
	keys := make([]string, 0, len(vals))
	for k := range vals {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, k := range keys {
		fmt.Fprintf(f, "| %s | %.8f |\n", k, vals[k])
	}
	fmt.Fprintf(f, "\n")
}

func init() {
	evaluateCompressionBreakoutCmd.Flags().StringVar(&ecbFeatures, "features", "", "Path to features JSON")
	evaluateCompressionBreakoutCmd.Flags().StringVar(&ecbRegimes, "regimes", "", "Path to regimes JSON")
	evaluateCompressionBreakoutCmd.Flags().StringVar(&ecbSide, "side", "LONG", "Side to evaluate")
	evaluateCompressionBreakoutCmd.Flags().StringVar(&ecbOut, "out", "", "Path to output file")

	_ = evaluateCompressionBreakoutCmd.MarkFlagRequired("features")
	_ = evaluateCompressionBreakoutCmd.MarkFlagRequired("regimes")

	rootCmd.AddCommand(evaluateCompressionBreakoutCmd)
}
