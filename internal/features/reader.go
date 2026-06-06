package features

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"os"
	"strconv"
)

func ReadRowsJSON(path string) ([]Row, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read file: %w", err)
	}
	var rows []Row
	if err := json.Unmarshal(data, &rows); err != nil {
		return nil, fmt.Errorf("unmarshal JSON: %w", err)
	}
	return rows, nil
}

func ReadRowsCSV(path string) ([]Row, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("open file: %w", err)
	}
	defer file.Close()

	r := csv.NewReader(file)
	records, err := r.ReadAll()
	if err != nil {
		return nil, fmt.Errorf("read CSV: %w", err)
	}

	if len(records) == 0 {
		return nil, nil
	}

	// First row is header
	header := records[0]
	headerMap := make(map[string]int)
	for idx, col := range header {
		headerMap[col] = idx
	}

	var rows []Row
	for i := 1; i < len(records); i++ {
		rec := records[i]
		getFloat := func(col string) float64 {
			idx, ok := headerMap[col]
			if !ok || idx >= len(rec) {
				return 0
			}
			val, _ := strconv.ParseFloat(rec[idx], 64)
			return val
		}
		getInt64 := func(col string) int64 {
			idx, ok := headerMap[col]
			if !ok || idx >= len(rec) {
				return 0
			}
			val, _ := strconv.ParseInt(rec[idx], 10, 64)
			return val
		}
		getBool := func(col string) bool {
			idx, ok := headerMap[col]
			if !ok || idx >= len(rec) {
				return false
			}
			val, _ := strconv.ParseBool(rec[idx])
			return val
		}
		getString := func(col string) string {
			idx, ok := headerMap[col]
			if !ok || idx >= len(rec) {
				return ""
			}
			return rec[idx]
		}

		rows = append(rows, Row{
			Market:             getString("market"),
			Symbol:             getString("symbol"),
			Interval:           getString("interval"),
			EventTimeMS:        getInt64("event_time_ms"),
			AvailableAtMS:      getInt64("available_at_ms"),
			Close:              getFloat("close"),
			Return1:            getFloat("return_1"),
			Return5:            getFloat("return_5"),
			Return15:           getFloat("return_15"),
			RealizedVol20:      getFloat("realized_vol_20"),
			RealizedVol60:      getFloat("realized_vol_60"),
			ATR14:              getFloat("atr_14"),
			ATRPct14:           getFloat("atr_pct_14"),
			BBWidth20:          getFloat("bb_width_20"),
			BBWidthPctRank60:   getFloat("bb_width_pct_rank_60"),
			EMA20:              getFloat("ema_20"),
			EMA50:              getFloat("ema_50"),
			EMA200:             getFloat("ema_200"),
			TrendSlope20:       getFloat("trend_slope_20"),
			VolumeRatio20:      getFloat("volume_ratio_20"),
			QuoteVolumeRatio20: getFloat("quote_volume_ratio_20"),
			TakerBuyRatio:      getFloat("taker_buy_ratio"),
			BTCReturn60:        getFloat("btc_return_60"),
			ETHReturn60:        getFloat("eth_return_60"),
			Warmup:             getBool("warmup"),
		})
	}

	return rows, nil
}
