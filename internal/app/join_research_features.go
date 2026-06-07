package app

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"math"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strconv"
	"strings"

	"github.com/davidmiguel22573/ak-engine/internal/features"
	"github.com/spf13/cobra"
)

var (
	jrfFeatures    string
	jrfDerivatives string
	jrfOut         string
)

var joinResearchFeaturesCmd = &cobra.Command{
	Use:   "join-research-features",
	Short: "Join research-only derivatives features by as-of availability",
	RunE: func(cmd *cobra.Command, args []string) error {
		rows, err := features.ReadRowsJSON(jrfFeatures)
		if err != nil {
			return fmt.Errorf("read features: %w", err)
		}
		derivativesRows, err := readResearchDerivativeRows(jrfDerivatives)
		if err != nil {
			return fmt.Errorf("read derivatives: %w", err)
		}
		joined, err := joinResearchFeatureRows(rows, derivativesRows)
		if err != nil {
			return err
		}
		if err := os.MkdirAll(filepath.Dir(jrfOut), 0755); err != nil {
			return fmt.Errorf("create output dir: %w", err)
		}
		data, err := json.MarshalIndent(joined, "", "  ")
		if err != nil {
			return fmt.Errorf("marshal joined features: %w", err)
		}
		if err := os.WriteFile(jrfOut, data, 0644); err != nil {
			return fmt.Errorf("write joined features: %w", err)
		}
		summary := map[string]any{
			"status":          "PASS",
			"feature_rows":    len(rows),
			"derivative_rows": len(derivativesRows),
			"out":             jrfOut,
		}
		out, _ := json.Marshal(summary)
		fmt.Println(string(out))
		return nil
	},
}

func init() {
	joinResearchFeaturesCmd.Flags().StringVar(&jrfFeatures, "features", "", "feature JSON input")
	joinResearchFeaturesCmd.Flags().StringVar(&jrfDerivatives, "derivatives", "", "derivatives file or directory")
	joinResearchFeaturesCmd.Flags().StringVar(&jrfOut, "out", "", "joined research JSON output")
	_ = joinResearchFeaturesCmd.MarkFlagRequired("features")
	_ = joinResearchFeaturesCmd.MarkFlagRequired("derivatives")
	_ = joinResearchFeaturesCmd.MarkFlagRequired("out")
	rootCmd.AddCommand(joinResearchFeaturesCmd)
}

type researchDerivativeRow struct {
	Source        string  `json:"source"`
	Dataset       string  `json:"dataset"`
	Market        string  `json:"market"`
	Symbol        string  `json:"symbol"`
	Interval      string  `json:"interval"`
	EventTimeMS   int64   `json:"event_time_ms"`
	AvailableAtMS int64   `json:"available_at_ms"`
	IngestedAtMS  int64   `json:"ingested_at_ms"`
	Value         float64 `json:"value"`
	Extra1        float64 `json:"extra_1"`
	Extra2        float64 `json:"extra_2"`
	SourceVersion string  `json:"source_version"`
}

type ResearchFeatureRow struct {
	features.Row
	Derivatives ResearchDerivativeFeatures `json:"derivatives"`
}

type ResearchDerivativeFeatures struct {
	FundingRate                *float64 `json:"funding_rate"`
	FundingRateUnknown         bool     `json:"funding_rate_unknown"`
	FundingRateZScore          *float64 `json:"funding_rate_zscore"`
	FundingRateZScoreUnknown   bool     `json:"funding_rate_zscore_unknown"`
	FundingRateChange          *float64 `json:"funding_rate_change"`
	FundingRateChangeUnknown   bool     `json:"funding_rate_change_unknown"`
	OpenInterestChange         *float64 `json:"open_interest_change"`
	OpenInterestChangeUnknown  bool     `json:"open_interest_change_unknown"`
	TakerBuySellImbalance      *float64 `json:"taker_buy_sell_imbalance"`
	TakerBuySellUnknown        bool     `json:"taker_buy_sell_imbalance_unknown"`
	LongShortRatio             *float64 `json:"long_short_ratio"`
	LongShortRatioUnknown      bool     `json:"long_short_ratio_unknown"`
	TopTraderLongShortRatio    *float64 `json:"top_trader_long_short_ratio"`
	TopTraderLongShortUnknown  bool     `json:"top_trader_long_short_ratio_unknown"`
	PositioningCrowdedLong     *bool    `json:"positioning_crowded_long"`
	PositioningCrowdedShort    *bool    `json:"positioning_crowded_short"`
	PositioningUnwindCandidate *bool    `json:"positioning_unwind_candidate"`
	PositioningUnknown         bool     `json:"positioning_unknown"`
}

func joinResearchFeatureRows(rows []features.Row, derivativesRows []researchDerivativeRow) ([]ResearchFeatureRow, error) {
	if err := validateResearchDerivativeRows(derivativesRows); err != nil {
		return nil, err
	}
	byKey := make(map[string][]researchDerivativeRow)
	for _, row := range derivativesRows {
		key := researchDerivativeKey(row.Market, row.Symbol, row.Dataset)
		byKey[key] = append(byKey[key], row)
	}
	for key := range byKey {
		sort.Slice(byKey[key], func(i, j int) bool {
			return byKey[key][i].AvailableAtMS < byKey[key][j].AvailableAtMS
		})
	}
	stats := buildFundingStats(byKey)

	out := make([]ResearchFeatureRow, 0, len(rows))
	for _, row := range rows {
		lookup := func(dataset string) (researchDerivativeRow, int, bool) {
			return asOfDerivative(byKey[researchDerivativeKey(row.Market, row.Symbol, dataset)], row.AvailableAtMS)
		}
		features := ResearchDerivativeFeatures{
			FundingRateUnknown:        true,
			FundingRateZScoreUnknown:  true,
			FundingRateChangeUnknown:  true,
			OpenInterestChangeUnknown: true,
			TakerBuySellUnknown:       true,
			LongShortRatioUnknown:     true,
			TopTraderLongShortUnknown: true,
			PositioningUnknown:        true,
		}
		if current, idx, ok := lookup("funding_rate"); ok {
			features.FundingRate = floatPtr(current.Value)
			features.FundingRateUnknown = false
			if idx > 0 {
				prev := byKey[researchDerivativeKey(row.Market, row.Symbol, "funding_rate")][idx-1]
				features.FundingRateChange = floatPtr(current.Value - prev.Value)
				features.FundingRateChangeUnknown = false
			}
			if z, ok := stats[fundingStatKey(row.Market, row.Symbol, current.EventTimeMS)]; ok {
				features.FundingRateZScore = floatPtr(z)
				features.FundingRateZScoreUnknown = false
			}
		}
		if current, idx, ok := lookup("open_interest"); ok && idx > 0 {
			prev := byKey[researchDerivativeKey(row.Market, row.Symbol, "open_interest")][idx-1]
			features.OpenInterestChange = floatPtr(current.Value - prev.Value)
			features.OpenInterestChangeUnknown = false
		}
		if current, _, ok := lookup("taker_buy_sell_volume"); ok {
			total := current.Extra1 + current.Extra2
			if total != 0 {
				features.TakerBuySellImbalance = floatPtr((current.Extra1 - current.Extra2) / total)
				features.TakerBuySellUnknown = false
			}
		}
		if current, _, ok := lookup("long_short_ratio"); ok {
			features.LongShortRatio = floatPtr(current.Value)
			features.LongShortRatioUnknown = false
		}
		if current, _, ok := lookup("top_trader_long_short_ratio"); ok {
			features.TopTraderLongShortRatio = floatPtr(current.Value)
			features.TopTraderLongShortUnknown = false
		}
		applyPositioningFlags(&features)
		out = append(out, ResearchFeatureRow{Row: row, Derivatives: features})
	}
	return out, nil
}

func applyPositioningFlags(f *ResearchDerivativeFeatures) {
	ratio := f.LongShortRatio
	if ratio == nil {
		ratio = f.TopTraderLongShortRatio
	}
	if ratio == nil {
		return
	}
	crowdedLong := *ratio >= 1.5
	crowdedShort := *ratio <= 0.67
	unwind := false
	if f.FundingRate != nil {
		unwind = (crowdedLong && *f.FundingRate > 0.0005) || (crowdedShort && *f.FundingRate < -0.0005)
	}
	f.PositioningCrowdedLong = boolPtr(crowdedLong)
	f.PositioningCrowdedShort = boolPtr(crowdedShort)
	f.PositioningUnwindCandidate = boolPtr(unwind)
	f.PositioningUnknown = false
}

func asOfDerivative(rows []researchDerivativeRow, availableAtMS int64) (researchDerivativeRow, int, bool) {
	idx := sort.Search(len(rows), func(i int) bool {
		return rows[i].AvailableAtMS > availableAtMS
	}) - 1
	if idx < 0 {
		return researchDerivativeRow{}, -1, false
	}
	return rows[idx], idx, true
}

func validateResearchDerivativeRows(rows []researchDerivativeRow) error {
	for i, row := range rows {
		if row.EventTimeMS <= 0 {
			return fmt.Errorf("derivative row %d: event_time_ms <= 0", i)
		}
		if row.AvailableAtMS <= 0 {
			return fmt.Errorf("derivative row %d: available_at_ms <= 0", i)
		}
		if row.AvailableAtMS < row.EventTimeMS {
			return fmt.Errorf("derivative row %d: available_at_ms < event_time_ms", i)
		}
	}
	return nil
}

func buildFundingStats(byKey map[string][]researchDerivativeRow) map[string]float64 {
	out := make(map[string]float64)
	for key, rows := range byKey {
		if !strings.HasSuffix(key, "|funding_rate") {
			continue
		}
		for i, row := range rows {
			start := i - 20
			if start < 0 {
				start = 0
			}
			window := rows[start : i+1]
			if len(window) < 2 {
				continue
			}
			var sum float64
			for _, item := range window {
				sum += item.Value
			}
			mean := sum / float64(len(window))
			var ss float64
			for _, item := range window {
				d := item.Value - mean
				ss += d * d
			}
			sd := math.Sqrt(ss / float64(len(window)))
			if sd == 0 {
				continue
			}
			out[fundingStatKey(row.Market, row.Symbol, row.EventTimeMS)] = (row.Value - mean) / sd
		}
	}
	return out
}

func researchDerivativeKey(market, symbol, dataset string) string {
	return market + "|" + strings.ToUpper(symbol) + "|" + dataset
}

func fundingStatKey(market, symbol string, eventTimeMS int64) string {
	return market + "|" + strings.ToUpper(symbol) + "|" + strconv.FormatInt(eventTimeMS, 10)
}

func readResearchDerivativeRows(path string) ([]researchDerivativeRow, error) {
	var files []string
	info, err := os.Stat(path)
	if err != nil {
		return nil, err
	}
	if info.IsDir() {
		err := filepath.WalkDir(path, func(p string, d os.DirEntry, err error) error {
			if err != nil {
				return err
			}
			if d.IsDir() {
				return nil
			}
			switch filepath.Ext(p) {
			case ".json", ".csv", ".parquet":
				files = append(files, p)
			}
			return nil
		})
		if err != nil {
			return nil, err
		}
	} else {
		files = append(files, path)
	}
	sort.Strings(files)

	var rows []researchDerivativeRow
	var parquetFiles []string
	for _, file := range files {
		switch filepath.Ext(file) {
		case ".json":
			more, err := readResearchDerivativeRowsJSON(file)
			if err != nil {
				return nil, err
			}
			rows = append(rows, more...)
		case ".csv":
			more, err := readResearchDerivativeRowsCSV(file)
			if err != nil {
				return nil, err
			}
			rows = append(rows, more...)
		case ".parquet":
			parquetFiles = append(parquetFiles, file)
		}
	}
	if len(parquetFiles) > 0 {
		more, err := readResearchDerivativeRowsParquet(parquetFiles)
		if err != nil {
			return nil, err
		}
		rows = append(rows, more...)
	}
	return rows, nil
}

func readResearchDerivativeRowsJSON(path string) ([]researchDerivativeRow, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var rows []researchDerivativeRow
	if err := json.Unmarshal(data, &rows); err != nil {
		return nil, err
	}
	return rows, nil
}

func readResearchDerivativeRowsCSV(path string) ([]researchDerivativeRow, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	r := csv.NewReader(f)
	records, err := r.ReadAll()
	if err != nil {
		return nil, err
	}
	if len(records) == 0 {
		return nil, nil
	}
	header := make(map[string]int)
	for i, name := range records[0] {
		header[name] = i
	}
	var rows []researchDerivativeRow
	for _, record := range records[1:] {
		rows = append(rows, researchDerivativeRow{
			Source:        csvString(record, header, "source"),
			Dataset:       csvString(record, header, "dataset"),
			Market:        csvString(record, header, "market"),
			Symbol:        csvString(record, header, "symbol"),
			Interval:      csvString(record, header, "interval"),
			EventTimeMS:   csvInt64(record, header, "event_time_ms"),
			AvailableAtMS: csvInt64(record, header, "available_at_ms"),
			IngestedAtMS:  csvInt64(record, header, "ingested_at_ms"),
			Value:         csvFloat(record, header, "value"),
			Extra1:        csvFloat(record, header, "extra_1"),
			Extra2:        csvFloat(record, header, "extra_2"),
			SourceVersion: csvString(record, header, "source_version"),
		})
	}
	return rows, nil
}

func readResearchDerivativeRowsParquet(files []string) ([]researchDerivativeRow, error) {
	if _, err := exec.LookPath("duckdb"); err != nil {
		return nil, fmt.Errorf("parquet derivative reads require duckdb installed")
	}
	var quoted []string
	for _, file := range files {
		quoted = append(quoted, "'"+strings.ReplaceAll(file, "'", "''")+"'")
	}
	query := fmt.Sprintf(`SELECT source, dataset, market, symbol, interval, event_time_ms, available_at_ms, ingested_at_ms, value, extra_1, extra_2, source_version FROM read_parquet([%s]) ORDER BY symbol, dataset, available_at_ms;`, strings.Join(quoted, ","))
	cmd := exec.Command("duckdb", "-json", "-c", query)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("duckdb parquet read failed: %s: %w", string(out), err)
	}
	var rows []researchDerivativeRow
	if err := json.Unmarshal(out, &rows); err != nil {
		return nil, fmt.Errorf("decode duckdb json: %w", err)
	}
	return rows, nil
}

func csvString(record []string, header map[string]int, name string) string {
	idx, ok := header[name]
	if !ok || idx >= len(record) {
		return ""
	}
	return record[idx]
}

func csvInt64(record []string, header map[string]int, name string) int64 {
	value, _ := strconv.ParseInt(csvString(record, header, name), 10, 64)
	return value
}

func csvFloat(record []string, header map[string]int, name string) float64 {
	value, _ := strconv.ParseFloat(csvString(record, header, name), 64)
	return value
}

func floatPtr(v float64) *float64 {
	return &v
}

func boolPtr(v bool) *bool {
	return &v
}
