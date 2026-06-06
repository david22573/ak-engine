package regime

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"strings"
)

func WriteLabelsJSON(path string, labels []Label) error {
	data, err := json.MarshalIndent(labels, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal labels: %w", err)
	}
	if err := os.WriteFile(path, data, 0644); err != nil {
		return fmt.Errorf("write file: %w", err)
	}
	return nil
}

func WriteLabelsCSV(path string, labels []Label) error {
	file, err := os.Create(path)
	if err != nil {
		return fmt.Errorf("create file: %w", err)
	}
	defer file.Close()

	w := csv.NewWriter(file)
	defer w.Flush()

	header := []string{
		"market", "symbol", "interval", "event_time_ms", "available_at_ms",
		"volatility", "trend", "liquidity", "market_beta", "sentiment",
		"composite", "reasons", "warmup",
	}

	if err := w.Write(header); err != nil {
		return fmt.Errorf("write header: %w", err)
	}

	for _, l := range labels {
		record := []string{
			l.Market, l.Symbol, l.Interval,
			strconv.FormatInt(l.EventTimeMS, 10),
			strconv.FormatInt(l.AvailableAtMS, 10),
			l.Volatility, l.Trend, l.Liquidity, l.MarketBeta, l.Sentiment,
			l.Composite, strings.Join(l.Reasons, "|"),
			strconv.FormatBool(l.Warmup),
		}
		if err := w.Write(record); err != nil {
			return fmt.Errorf("write record: %w", err)
		}
	}

	return nil
}

func WriteLabelsParquet(csvPath, parquetPath string) error {
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
