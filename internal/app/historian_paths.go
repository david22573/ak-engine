package app

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
)

const (
	defaultHistorianWorkdir = ".ak-historian/work"
	historianWorkdirEnv     = "AK_HISTORIAN_WORKDIR"
)

func resolveHistorianWorkdir(cmd *cobra.Command, configured string) string {
	configured = strings.TrimSpace(configured)
	if cmd != nil && cmd.Flags().Changed("workdir") && configured != "" {
		return configured
	}
	if env := strings.TrimSpace(os.Getenv(historianWorkdirEnv)); env != "" {
		return env
	}
	if configured != "" {
		return configured
	}
	return defaultHistorianWorkdir
}

func fundingRateDerivativePath(workdir, symbol, month string) string {
	symbol = strings.ToUpper(strings.TrimSpace(symbol))
	year := month[:4]
	monthOnly := month[5:7]
	return filepath.Join(workdir, "datasets", "derivatives", "source=binance", "dataset=funding_rate", "market=futures-um", "symbol="+symbol, "interval=8h", "year="+year, "month="+monthOnly, symbol+"-funding_rate-"+month+".parquet")
}
