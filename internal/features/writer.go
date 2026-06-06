package features

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"strings"
)

func WriteRowsJSON(path string, rows []Row) error {
	data, err := json.MarshalIndent(rows, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal rows: %w", err)
	}
	if err := os.WriteFile(path, data, 0644); err != nil {
		return fmt.Errorf("write file: %w", err)
	}
	return nil
}

func WriteRowsCSV(path string, rows []Row) error {
	file, err := os.Create(path)
	if err != nil {
		return fmt.Errorf("create file: %w", err)
	}
	defer file.Close()

	w := csv.NewWriter(file)
	defer w.Flush()

	header := []string{
		"market", "symbol", "interval", "event_time_ms", "available_at_ms",
		"close", "return_1", "return_5", "return_15", "realized_vol_20",
		"realized_vol_60", "atr_14", "atr_pct_14", "bb_width_20", "bb_width_pct_rank_60",
		"ema_20", "ema_50", "ema_200", "trend_slope_20", "volume_ratio_20",
		"quote_volume_ratio_20", "taker_buy_ratio", "btc_return_60", "eth_return_60",
		"warmup",
	}

	if err := w.Write(header); err != nil {
		return fmt.Errorf("write header: %w", err)
	}

	for _, r := range rows {
		record := []string{
			r.Market, r.Symbol, r.Interval,
			strconv.FormatInt(r.EventTimeMS, 10),
			strconv.FormatInt(r.AvailableAtMS, 10),
			strconv.FormatFloat(r.Close, 'f', -1, 64),
			strconv.FormatFloat(r.Return1, 'f', -1, 64),
			strconv.FormatFloat(r.Return5, 'f', -1, 64),
			strconv.FormatFloat(r.Return15, 'f', -1, 64),
			strconv.FormatFloat(r.RealizedVol20, 'f', -1, 64),
			strconv.FormatFloat(r.RealizedVol60, 'f', -1, 64),
			strconv.FormatFloat(r.ATR14, 'f', -1, 64),
			strconv.FormatFloat(r.ATRPct14, 'f', -1, 64),
			strconv.FormatFloat(r.BBWidth20, 'f', -1, 64),
			strconv.FormatFloat(r.BBWidthPctRank60, 'f', -1, 64),
			strconv.FormatFloat(r.EMA20, 'f', -1, 64),
			strconv.FormatFloat(r.EMA50, 'f', -1, 64),
			strconv.FormatFloat(r.EMA200, 'f', -1, 64),
			strconv.FormatFloat(r.TrendSlope20, 'f', -1, 64),
			strconv.FormatFloat(r.VolumeRatio20, 'f', -1, 64),
			strconv.FormatFloat(r.QuoteVolumeRatio20, 'f', -1, 64),
			strconv.FormatFloat(r.TakerBuyRatio, 'f', -1, 64),
			strconv.FormatFloat(r.BTCReturn60, 'f', -1, 64),
			strconv.FormatFloat(r.ETHReturn60, 'f', -1, 64),
			strconv.FormatBool(r.Warmup),
		}
		if err := w.Write(record); err != nil {
			return fmt.Errorf("write record: %w", err)
		}
	}

	return nil
}

func WriteRowsParquet(csvPath, parquetPath string) error {
	_, err := exec.LookPath("duckdb")
	if err != nil {
		return fmt.Errorf("parquet output requires duckdb installed")
	}

	escapedCsv := strings.ReplaceAll(csvPath, "'", "''")
	escapedParquet := strings.ReplaceAll(parquetPath, "'", "''")

	query := fmt.Sprintf(
		"COPY (SELECT * FROM read_csv_auto('%s')) TO '%s' (FORMAT PARQUET, COMPRESSION ZSTD);",
		escapedCsv, escapedParquet,
	)

	cmd := exec.Command("duckdb", "-c", query)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("duckdb parquet conversion failed: %s: %w", string(output), err)
	}

	return nil
}
