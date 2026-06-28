package app

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"sort"
)

type NativeSummaryV2Row struct {
	Family                 string  `json:"family"`
	Side                   string  `json:"side"`
	Symbol                 string  `json:"symbol"`
	Month                  string  `json:"month"`
	Quarter                string  `json:"quarter"`
	Year                   string  `json:"year"`
	HorizonMinutes         int     `json:"horizon_minutes"`
	CostBps                float64 `json:"cost_bps"`
	DelayCandles           int     `json:"delay_candles"`
	FundingBucket          string  `json:"funding_bucket"`
	RegimeBucket           string  `json:"regime_bucket"`
	FundingXRegimeBucket   string  `json:"funding_x_regime_bucket"`
	EventCount             int     `json:"event_count"`
	DeClusteredEventCount  int     `json:"declustered_event_count"`
	GrossProfitBps         float64 `json:"gross_profit_bps"`
	GrossLossBps           float64 `json:"gross_loss_bps"`
	NetBps                 float64 `json:"net_bps"`
	ExpectancyBps          float64 `json:"expectancy_bps"`
	ProfitFactor           float64 `json:"profit_factor"`
	WinCount               int     `json:"win_count"`
	LossCount              int     `json:"loss_count"`
	WinRate                float64 `json:"win_rate"`
	InputHash              string  `json:"input_hash"`
	SummaryHash            string  `json:"summary_hash"`
}

type v2GroupKey struct {
	Family         string
	Side           string
	Symbol         string
	Month          string
	HorizonMinutes int
	CostBps        float64
	DelayCandles   int
	FundingBucket  string
	RegimeBucket   string
}

type v2EventData struct {
	Ret         float64
	EventTimeMS int64
}

func hashString(s string) string {
	h := sha256.Sum256([]byte(s))
	return hex.EncodeToString(h[:])
}

func computeNativeSummaryV2(events []FundingEventRow, rows []ResearchFeatureRow, inputHash string) []NativeSummaryV2Row {
	rowMap := make(map[int64]int)
	for i, r := range rows {
		rowMap[r.EventTimeMS] = i
	}

	horizons := []string{"5m", "15m", "30m", "60m", "120m", "240m"}
	costs := []float64{5, 7.5, 10, 15}
	delays := []int{0, 1, 2}

	groups := make(map[v2GroupKey][]v2EventData)

	for _, ev := range events {
		idx, ok := rowMap[ev.EventTimeMS]
		if !ok {
			continue
		}

		for _, hor := range horizons {
			horMin := int(fundingHorizonMS[hor] / (60 * 1000))

			for _, delay := range delays {
				var entryPrice float64
				var futurePrice float64
				var valid bool
				var eventTimeMS int64

				if delay == 0 {
					entryPrice = ev.EntryPrice
					eventTimeMS = ev.EventTimeMS
					futurePrice, valid = futureFundingClose(rows, idx, ev.EventTimeMS+fundingHorizonMS[hor])
				} else {
					delayIdx := idx + delay
					if delayIdx < len(rows) && rows[delayIdx].Close > 0 {
						entryPrice = rows[delayIdx].Close
						eventTimeMS = rows[delayIdx].EventTimeMS
						targetTimeMS := rows[delayIdx].EventTimeMS + fundingHorizonMS[hor]
						futurePrice, valid = futureFundingClose(rows, delayIdx, targetTimeMS)
					}
				}

				if !valid || futurePrice <= 0 || entryPrice <= 0 {
					continue
				}

				rawRet := directionalReturnBps(entryPrice, futurePrice, ev.Side)

				for _, cost := range costs {
					netRet := rawRet - cost
					key := v2GroupKey{
						Family:         ev.Family,
						Side:           ev.Side,
						Symbol:         ev.Symbol,
						Month:          monthFromEventTime(ev.EventTimeMS), // original event month
						HorizonMinutes: horMin,
						CostBps:        cost,
						DelayCandles:   delay,
						FundingBucket:  ev.FundingBucket,
						RegimeBucket:   ev.RegimeComposite,
					}
					groups[key] = append(groups[key], v2EventData{Ret: netRet, EventTimeMS: eventTimeMS})
				}
			}
		}
	}

	var result []NativeSummaryV2Row
	for k, dataList := range groups {
		var grossProf, grossLoss float64
		var wins, losses int
		var mockEvents []FundingEventRow
		
		for _, d := range dataList {
			if d.Ret > 0 {
				grossProf += d.Ret
				wins++
			} else if d.Ret < 0 {
				grossLoss -= d.Ret
				losses++
			}
			mockEvents = append(mockEvents, FundingEventRow{EventTimeMS: d.EventTimeMS})
		}

		declustered := len(deClusterFundingEvents(mockEvents))
		eventCount := len(dataList)
		netBps := grossProf - grossLoss
		expectancy := netBps / float64(eventCount)
		winRate := float64(wins) / float64(eventCount)
		pf := 0.0
		if grossLoss > 0 {
			pf = grossProf / grossLoss
		} else if grossProf > 0 {
			pf = 999.0 // standard cap for zero loss
		}

		row := NativeSummaryV2Row{
			Family:                 k.Family,
			Side:                   k.Side,
			Symbol:                 k.Symbol,
			Month:                  k.Month,
			Quarter:                quarterFromMonth(k.Month),
			Year:                   monthYear(k.Month),
			HorizonMinutes:         k.HorizonMinutes,
			CostBps:                k.CostBps,
			DelayCandles:           k.DelayCandles,
			FundingBucket:          k.FundingBucket,
			RegimeBucket:           k.RegimeBucket,
			FundingXRegimeBucket:   fmt.Sprintf("%s|%s", k.FundingBucket, k.RegimeBucket),
			EventCount:             eventCount,
			DeClusteredEventCount:  declustered,
			GrossProfitBps:         roundMetric(grossProf),
			GrossLossBps:           roundMetric(grossLoss),
			NetBps:                 roundMetric(netBps),
			ExpectancyBps:          roundMetric(expectancy),
			ProfitFactor:           roundMetric(pf),
			WinCount:               wins,
			LossCount:              losses,
			WinRate:                roundMetric(winRate),
			InputHash:              inputHash,
		}

		// Compute summary hash
		hashPayload := fmt.Sprintf("%s|%s|%s|%s|%d|%.1f|%d|%d|%.2f|%.2f",
			row.Family, row.Side, row.Symbol, row.Month, row.HorizonMinutes, row.CostBps, row.DelayCandles, row.EventCount, row.ExpectancyBps, row.ProfitFactor)
		row.SummaryHash = hashString(hashPayload)

		result = append(result, row)
	}
	
	// Sort to keep deterministic
	sort.Slice(result, func(i, j int) bool {
		a, b := result[i], result[j]
		keyA := fmt.Sprintf("%s|%s|%s|%s|%d|%.1f|%d|%s|%s", a.Family, a.Side, a.Symbol, a.Month, a.HorizonMinutes, a.CostBps, a.DelayCandles, a.FundingBucket, a.RegimeBucket)
		keyB := fmt.Sprintf("%s|%s|%s|%s|%d|%.1f|%d|%s|%s", b.Family, b.Side, b.Symbol, b.Month, b.HorizonMinutes, b.CostBps, b.DelayCandles, b.FundingBucket, b.RegimeBucket)
		return keyA < keyB
	})

	return result
}
