package regime

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"os"
	"strconv"
	"strings"
)

func ReadLabelsJSON(path string) ([]Label, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read file: %w", err)
	}
	var labels []Label
	if err := json.Unmarshal(data, &labels); err != nil {
		return nil, fmt.Errorf("unmarshal JSON: %w", err)
	}
	return labels, nil
}

func ReadLabelsCSV(path string) ([]Label, error) {
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

	header := records[0]
	headerMap := make(map[string]int)
	for idx, col := range header {
		headerMap[col] = idx
	}

	var labels []Label
	for i := 1; i < len(records); i++ {
		rec := records[i]
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
		getSlice := func(col string) []string {
			idx, ok := headerMap[col]
			if !ok || idx >= len(rec) || rec[idx] == "" {
				return nil
			}
			return strings.Split(rec[idx], "|")
		}

		labels = append(labels, Label{
			Market:        getString("market"),
			Symbol:        getString("symbol"),
			Interval:      getString("interval"),
			EventTimeMS:   getInt64("event_time_ms"),
			AvailableAtMS: getInt64("available_at_ms"),
			Volatility:    getString("volatility"),
			Trend:         getString("trend"),
			Liquidity:     getString("liquidity"),
			MarketBeta:    getString("market_beta"),
			Sentiment:     getString("sentiment"),
			Composite:     getString("composite"),
			Reasons:       getSlice("reasons"),
			Warmup:        getBool("warmup"),
		})
	}

	return labels, nil
}
