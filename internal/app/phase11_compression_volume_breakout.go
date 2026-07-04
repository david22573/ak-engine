package app

import (
	"context"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/davidmiguel22573/ak-engine/internal/data"
	"github.com/davidmiguel22573/ak-engine/internal/features"
	"github.com/davidmiguel22573/ak-engine/internal/research"
	"github.com/davidmiguel22573/ak-engine/pkg/protocol"
	"github.com/spf13/cobra"
)

var (
	p11cvbWorkdir  string
	p11cvbSymbols  string
	p11cvbMarket   string
	p11cvbInterval string
	p11cvbFrom     string
	p11cvbTo       string
	p11cvbOut      string
)

const phase11CVBFamily = "CompressionVolumeBreakout"

var phase11CVBHorizons = []string{"15m", "60m", "240m"}

type phase11CVBSummaryRow struct {
	Symbol               string               `json:"symbol"`
	Year                 string               `json:"year"`
	Quarter              string               `json:"quarter"`
	Month                string               `json:"month"`
	Family               string               `json:"family"`
	Side                 string               `json:"side"`
	Horizon              string               `json:"horizon"`
	SummarySchemaVersion string               `json:"summary_schema_version"`
	ClusterKeyVersion    string               `json:"cluster_key_version"`
	Stats                phase11CVBStats      `json:"stats"`
	Diagnostics          phase11CVBDiagnostic `json:"diagnostics,omitempty"`
}

type phase11CVBStats struct {
	BaselineCostBps           float64                 `json:"baseline_cost_bps"`
	EventCount                int                     `json:"event_count"`
	RawEventCount             int                     `json:"raw_event_count"`
	DeClusteredEventCount     int                     `json:"de_clustered_event_count"`
	PFAfter5Bps               float64                 `json:"pf_after_5_bps"`
	PFAfter7_5Bps             float64                 `json:"pf_after_7_5_bps"`
	PFAfter10Bps              float64                 `json:"pf_after_10_bps"`
	PFAfter15Bps              float64                 `json:"pf_after_15_bps"`
	ExpectancyBpsAfter5Bps    float64                 `json:"expectancy_bps_after_5_bps"`
	WinRate                   float64                 `json:"win_rate"`
	AverageReturnBps          float64                 `json:"average_return_bps"`
	MedianReturnBps           float64                 `json:"median_return_bps"`
	EntryDelay1CExpectancyBps float64                 `json:"entry_delay_1c_expectancy_bps"`
	EntryDelay1CAvailable     bool                    `json:"entry_delay_1c_available"`
	GrossProfitBps            float64                 `json:"gross_profit_bps"`
	GrossLossBps              float64                 `json:"gross_loss_bps"`
	NetBps                    float64                 `json:"net_bps"`
	WinCount                  int                     `json:"win_count"`
	LossCount                 int                     `json:"loss_count"`
	CostStress                []phase11CVBCostMetric  `json:"cost_stress"`
	DelayStress               []phase11CVBDelayMetric `json:"delay_stress"`
}

type phase11CVBCostMetric struct {
	CostBps               float64 `json:"cost_bps"`
	EventCount            int     `json:"event_count"`
	DeClusteredEventCount int     `json:"de_clustered_event_count"`
	GrossProfitBps        float64 `json:"gross_profit_bps"`
	GrossLossBps          float64 `json:"gross_loss_bps"`
	NetBps                float64 `json:"net_bps"`
	ExpectancyBps         float64 `json:"expectancy_bps"`
	PF                    float64 `json:"pf"`
	WinCount              int     `json:"win_count"`
	LossCount             int     `json:"loss_count"`
	WinRate               float64 `json:"win_rate"`
}

type phase11CVBDelayMetric struct {
	DelayCandles          int     `json:"delay_candles"`
	Label                 string  `json:"label"`
	Available             bool    `json:"available"`
	EventCount            int     `json:"event_count"`
	DeClusteredEventCount int     `json:"de_clustered_event_count"`
	GrossProfitBps        float64 `json:"gross_profit_bps"`
	GrossLossBps          float64 `json:"gross_loss_bps"`
	NetBps                float64 `json:"net_bps"`
	ExpectancyBps         float64 `json:"expectancy_bps"`
	PF                    float64 `json:"pf"`
	WinCount              int     `json:"win_count"`
	LossCount             int     `json:"loss_count"`
	WinRate               float64 `json:"win_rate"`
}

type phase11CVBDiagnostic struct {
	Status                    string `json:"status"`
	FeatureRows               int    `json:"feature_rows"`
	RegimeRows                int    `json:"regime_rows"`
	CompressedRows            int    `json:"compressed_rows"`
	DirectionalBreakRows      int    `json:"directional_break_rows"`
	VolumeConfirmedRows       int    `json:"volume_confirmed_rows"`
	BBExpansionConfirmedRows  int    `json:"bb_expansion_confirmed_rows"`
	BetaAgreementRows         int    `json:"beta_agreement_rows"`
	RowsMissingForwardReturns int    `json:"rows_missing_forward_returns"`
	LeakageStatus             string `json:"leakage_status"`
	Error                     string `json:"error,omitempty"`
}

type phase11CVBEvent struct {
	Symbol      string
	Side        string
	EventTimeMS int64
	Index       int
	EntryPrice  float64
	ReturnsBps  map[string]float64
	DelayBps    map[string]float64
}

type phase11CVBReport struct {
	Phase               string                 `json:"phase"`
	Family              string                 `json:"family"`
	Mode                string                 `json:"mode"`
	Implementation      string                 `json:"implementation"`
	Boundaries          []string               `json:"boundaries"`
	Symbols             []string               `json:"symbols"`
	Months              []string               `json:"months"`
	Horizons            []string               `json:"horizons"`
	Coverage            phase11CVBCoverage     `json:"coverage"`
	VerdictCounts       map[string]int         `json:"verdict_counts"`
	Leaderboard         []phase11CVBLeaderRow  `json:"leaderboard"`
	RetainedSummaries   []phase11CVBSummaryRow `json:"retained_summaries"`
	FinalRecommendation string                 `json:"final_recommendation"`
}

type phase11CVBCoverage struct {
	ExpectedSymbolMonths   int      `json:"expected_symbol_months"`
	CompletedSymbolMonths  int      `json:"completed_symbol_months"`
	MissingSymbolMonths    []string `json:"missing_symbol_months,omitempty"`
	RawEventDetailRetained bool     `json:"raw_event_detail_retained"`
}

type phase11CVBLeaderRow struct {
	Family                    string   `json:"family"`
	Side                      string   `json:"side"`
	Horizon                   string   `json:"horizon"`
	EventCount                int      `json:"event_count"`
	DeClusteredEventCount     int      `json:"de_clustered_event_count"`
	PFCombined5Bps            float64  `json:"pf_combined_5bps"`
	ExpectancyCombined5BpsBps float64  `json:"expectancy_combined_5bps_bps"`
	PositiveMonthCount        int      `json:"positive_month_count"`
	WorstQuarterPF5Bps        float64  `json:"worst_quarter_pf_5bps"`
	Top1MonthContributionPct  float64  `json:"top_1_month_contribution_pct"`
	Top2MonthContributionPct  float64  `json:"top_2_month_contribution_pct"`
	EntryDelay1CExpectancyBps float64  `json:"entry_delay_1c_expectancy_bps"`
	Cost10BpsPF               float64  `json:"cost_10bps_pf"`
	LeakageStatus             string   `json:"leakage_status"`
	Verdict                   string   `json:"verdict"`
	FailedGates               []string `json:"failed_gates,omitempty"`
}

var phase11CompressionVolumeBreakoutCmd = &cobra.Command{
	Use:   "phase11-compression-volume-breakout",
	Short: "Evaluate Phase 11.1 CompressionVolumeBreakout from local retained inputs",
	RunE: func(cmd *cobra.Command, args []string) error {
		cfgSymbols := parseFundingSymbols(p11cvbSymbols)
		if len(cfgSymbols) == 0 {
			cfgSymbols = append([]string(nil), defaultFundingSymbols...)
		}
		months := parseFundingMonths(p11cvbFrom, p11cvbTo)
		if len(months) == 0 {
			return fmt.Errorf("missing or invalid --from/--to month range")
		}
		mdOut, jsonOut := normalizeMDAndJSONPaths(p11cvbOut)

		report, err := runPhase11CompressionVolumeBreakout(cmd.Context(), p11cvbWorkdir, p11cvbMarket, p11cvbInterval, cfgSymbols, months)
		if err != nil {
			return err
		}
		if err := writeJSONFile(jsonOut, report); err != nil {
			return err
		}
		if err := os.WriteFile(mdOut, []byte(renderPhase11CVBMarkdown(report)), 0644); err != nil {
			return err
		}
		fmt.Printf("Phase 11.1 CompressionVolumeBreakout report written to %s\n", mdOut)
		return nil
	},
}

func init() {
	phase11CompressionVolumeBreakoutCmd.Flags().StringVar(&p11cvbWorkdir, "workdir", defaultHistorianWorkdir, "local historian workdir")
	phase11CompressionVolumeBreakoutCmd.Flags().StringVar(&p11cvbSymbols, "symbols", strings.Join(defaultFundingSymbols, ","), "comma-separated target symbols")
	phase11CompressionVolumeBreakoutCmd.Flags().StringVar(&p11cvbMarket, "market", "futures-um", "market")
	phase11CompressionVolumeBreakoutCmd.Flags().StringVar(&p11cvbInterval, "interval", "1m", "candle interval")
	phase11CompressionVolumeBreakoutCmd.Flags().StringVar(&p11cvbFrom, "from", "2024-01", "from month YYYY-MM")
	phase11CompressionVolumeBreakoutCmd.Flags().StringVar(&p11cvbTo, "to", "2025-12", "to month YYYY-MM")
	phase11CompressionVolumeBreakoutCmd.Flags().StringVar(&p11cvbOut, "out", filepath.Join("runs", "reports", "phase11_1_compression_volume_breakout.md"), "output markdown path")
	rootCmd.AddCommand(phase11CompressionVolumeBreakoutCmd)
}

func runPhase11CompressionVolumeBreakout(ctx context.Context, workdir, market, interval string, symbols, months []string) (phase11CVBReport, error) {
	report := phase11CVBReport{
		Phase:          "Phase 11.1",
		Family:         phase11CVBFamily,
		Mode:           "research/evaluation only",
		Implementation: "CompressionVolumeBreakout only",
		Boundaries: []string{
			"no funding primary trigger",
			"no ak-trader changes",
			"no data fetch",
			"retained summaries only",
			"no threshold tuning to force success",
		},
		Symbols:       symbols,
		Months:        months,
		Horizons:      phase11CVBHorizons,
		VerdictCounts: make(map[string]int),
	}
	report.Coverage.ExpectedSymbolMonths = len(symbols) * len(months)
	report.Coverage.RawEventDetailRetained = false

	for _, symbol := range symbols {
		fmt.Printf("phase11.1 processing %s\n", symbol)
		rows, err := buildPhase11CVBInputs(ctx, workdir, market, interval, symbol, months)
		if err != nil {
			for _, month := range months {
				report.Coverage.MissingSymbolMonths = append(report.Coverage.MissingSymbolMonths, symbol+"|"+month)
				report.RetainedSummaries = append(report.RetainedSummaries, zeroPhase11CVBSummaries(symbol, month, err.Error())...)
			}
			continue
		}
		events, diagByMonth := buildPhase11CVBEvents(rows, months)
		report.Coverage.CompletedSymbolMonths += len(months)
		for _, month := range months {
			for _, row := range summarizePhase11CVBMonth(symbol, month, events[month], diagByMonth[month]) {
				report.RetainedSummaries = append(report.RetainedSummaries, row)
			}
		}
	}

	sortPhase11CVBSummaries(report.RetainedSummaries)
	report.Leaderboard = buildPhase11CVBLeaderboard(report.RetainedSummaries)
	for _, row := range report.Leaderboard {
		report.VerdictCounts[row.Verdict]++
	}
	for _, verdict := range verdictOrder {
		if _, ok := report.VerdictCounts[verdict]; !ok {
			report.VerdictCounts[verdict] = 0
		}
	}
	if report.VerdictCounts[verdictResearchLead] > 0 {
		report.FinalRecommendation = "research lead found; still no promotion or ak-trader implementation in Phase 11.1"
	} else if report.VerdictCounts[verdictFragile] > 0 {
		report.FinalRecommendation = "fragile only; do not promote, review robustness before any next step"
	} else if len(report.Coverage.MissingSymbolMonths) > 0 {
		report.FinalRecommendation = "inconclusive because local retained inputs are incomplete"
	} else {
		report.FinalRecommendation = "CompressionVolumeBreakout rejected for promotion under Phase 11.1 gates"
	}
	return report, nil
}

func buildPhase11CVBInputs(ctx context.Context, workdir, market, interval, symbol string, months []string) ([]features.Row, error) {
	fromTime, toTime, err := phase11CVBLoadWindow(months)
	if err != nil {
		return nil, err
	}
	src := data.NewLocalParquetSource()
	req := data.CandleRequest{Source: "local-parquet", Path: workdir, Market: market, Symbol: symbol, Interval: interval, From: fromTime, To: toTime}
	candles, err := src.LoadCandles(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("load target candles: %w", err)
	}
	if len(candles) == 0 {
		return nil, fmt.Errorf("load target candles: no rows")
	}
	btc, err := loadPhase11CVBContext(ctx, src, req, "BTCUSDT", symbol)
	if err != nil {
		return nil, err
	}
	eth, err := loadPhase11CVBContext(ctx, src, req, "ETHUSDT", symbol)
	if err != nil {
		return nil, err
	}
	rows, err := features.BuildRows(candles, features.BuildOptions{
		Market:     market,
		Symbol:     symbol,
		Interval:   interval,
		ContextBTC: btc,
		ContextETH: eth,
	})
	if err != nil {
		return nil, err
	}
	if research.CheckFeatureRows(rows).Status != "PASS" {
		return nil, fmt.Errorf("feature leakage check failed")
	}
	return rows, nil
}

func loadPhase11CVBContext(ctx context.Context, src *data.LocalParquetSource, req data.CandleRequest, contextSymbol, targetSymbol string) ([]protocol.Candle, error) {
	if contextSymbol == targetSymbol {
		return nil, nil
	}
	req.Symbol = contextSymbol
	candles, err := src.LoadCandles(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("load context candles %s: %w", contextSymbol, err)
	}
	return candles, nil
}

func phase11CVBLoadWindow(months []string) (time.Time, time.Time, error) {
	if len(months) == 0 {
		return time.Time{}, time.Time{}, fmt.Errorf("missing months")
	}
	start, err := time.Parse("2006-01-02", months[0]+"-01")
	if err != nil {
		return time.Time{}, time.Time{}, err
	}
	last, err := time.Parse("2006-01-02", months[len(months)-1]+"-01")
	if err != nil {
		return time.Time{}, time.Time{}, err
	}
	return start.AddDate(0, -2, 0), last.AddDate(0, 1, 2), nil
}

func buildPhase11CVBEvents(rows []features.Row, months []string) (map[string][]phase11CVBEvent, map[string]phase11CVBDiagnostic) {
	monthSet := make(map[string]struct{})
	events := make(map[string][]phase11CVBEvent)
	diag := make(map[string]phase11CVBDiagnostic)
	for _, month := range months {
		monthSet[month] = struct{}{}
		diag[month] = phase11CVBDiagnostic{Status: "PASS", FeatureRows: len(rows), RegimeRows: len(rows), LeakageStatus: "PASS"}
	}

	seen := make(map[string]struct{})
	for i := 1; i < len(rows); i++ {
		row := rows[i]
		month := monthFromEventTime(row.EventTimeMS)
		if _, ok := monthSet[month]; !ok {
			continue
		}
		d := diag[month]
		if row.Warmup || row.AvailableAtMS < row.EventTimeMS {
			if row.AvailableAtMS < row.EventTimeMS {
				d.LeakageStatus = "FAIL"
			}
			diag[month] = d
			continue
		}
		label := phase11CVBDerivedRegime(row)
		compressed := label.Volatility == "compressed" || label.Composite == "compressed_range"
		if !compressed {
			diag[month] = d
			continue
		}
		d.CompressedRows++
		prev := rows[i-1]
		longBreak := prev.Close <= prev.EMA20 && row.Close > row.EMA20
		shortBreak := prev.Close >= prev.EMA20 && row.Close < row.EMA20
		if !longBreak && !shortBreak {
			diag[month] = d
			continue
		}
		d.DirectionalBreakRows++
		if row.VolumeRatio20 < 1.20 {
			diag[month] = d
			continue
		}
		d.VolumeConfirmedRows++
		if !(prev.BBWidthPctRank60 <= 0.35 && row.BBWidthPctRank60 > prev.BBWidthPctRank60) {
			diag[month] = d
			continue
		}
		d.BBExpansionConfirmedRows++

		side := ""
		if longBreak && label.MarketBeta == "btc_up" {
			side = "long"
		}
		if shortBreak && label.MarketBeta == "btc_down" {
			side = "short"
		}
		if side == "" {
			diag[month] = d
			continue
		}
		d.BetaAgreementRows++
		ret, ok := phase11CVBForwardReturns(rows, i, side, 0)
		delay, delayOK := phase11CVBForwardReturns(rows, i, side, 1)
		if !ok || !delayOK {
			d.RowsMissingForwardReturns++
			diag[month] = d
			continue
		}
		key := fmt.Sprintf("%s|%s|%d", row.Symbol, side, row.EventTimeMS)
		if _, exists := seen[key]; exists {
			diag[month] = d
			continue
		}
		seen[key] = struct{}{}
		events[month] = append(events[month], phase11CVBEvent{
			Symbol:      row.Symbol,
			Side:        side,
			EventTimeMS: row.EventTimeMS,
			Index:       i,
			EntryPrice:  row.Close,
			ReturnsBps:  ret,
			DelayBps:    delay,
		})
		diag[month] = d
	}
	return events, diag
}

type phase11CVBDerivedRegimeLabel struct {
	Volatility string
	Composite  string
	MarketBeta string
}

func phase11CVBDerivedRegime(row features.Row) phase11CVBDerivedRegimeLabel {
	label := phase11CVBDerivedRegimeLabel{Volatility: "normal", Composite: "normal", MarketBeta: "btc_flat"}
	if row.BBWidthPctRank60 <= 0.20 {
		label.Volatility = "compressed"
		label.Composite = "compressed_range"
	}
	if row.BTCReturn60 > 0.003 {
		label.MarketBeta = "btc_up"
	} else if row.BTCReturn60 < -0.003 {
		label.MarketBeta = "btc_down"
	}
	return label
}

func phase11CVBForwardReturns(rows []features.Row, idx int, side string, delayCandles int) (map[string]float64, bool) {
	entryIdx := idx + delayCandles
	if entryIdx >= len(rows) || rows[entryIdx].Close <= 0 {
		return nil, false
	}
	out := make(map[string]float64)
	for _, horizon := range phase11CVBHorizons {
		future, ok := phase11CVBFutureClose(rows, entryIdx, fundingHorizonMS[horizon])
		if !ok || future <= 0 {
			return nil, false
		}
		ret := (future - rows[entryIdx].Close) / rows[entryIdx].Close * 10000.0
		if side == "short" {
			ret = -ret
		}
		out[horizon] = ret
	}
	return out, true
}

func phase11CVBFutureClose(rows []features.Row, startIdx int, offsetMS int64) (float64, bool) {
	target := rows[startIdx].EventTimeMS + offsetMS
	idx := sort.Search(len(rows)-startIdx, func(i int) bool { return rows[startIdx+i].EventTimeMS >= target })
	j := startIdx + idx
	if j >= len(rows) || rows[j].EventTimeMS < target {
		return 0, false
	}
	return rows[j].Close, true
}

func summarizePhase11CVBMonth(symbol, month string, events []phase11CVBEvent, diag phase11CVBDiagnostic) []phase11CVBSummaryRow {
	out := make([]phase11CVBSummaryRow, 0, 6)
	for _, side := range []string{"long", "short"} {
		sideEvents := filterPhase11CVBEvents(events, func(e phase11CVBEvent) bool { return e.Side == side })
		for _, horizon := range phase11CVBHorizons {
			stats := phase11CVBStatsForEvents(sideEvents, horizon)
			out = append(out, phase11CVBSummaryRow{
				Symbol:               symbol,
				Year:                 monthYear(month),
				Quarter:              quarterFromMonth(month),
				Month:                month,
				Family:               phase11CVBFamily,
				Side:                 side,
				Horizon:              horizon,
				SummarySchemaVersion: "11.1-retained",
				ClusterKeyVersion:    "1.0-native",
				Stats:                stats,
				Diagnostics:          diag,
			})
		}
	}
	return out
}

func zeroPhase11CVBSummaries(symbol, month, errText string) []phase11CVBSummaryRow {
	diag := phase11CVBDiagnostic{Status: "missing_data", LeakageStatus: "N/A", Error: errText}
	return summarizePhase11CVBMonth(symbol, month, nil, diag)
}

func phase11CVBStatsForEvents(events []phase11CVBEvent, horizon string) phase11CVBStats {
	stats := phase11CVBStats{BaselineCostBps: 5, RawEventCount: len(events)}
	for _, cost := range []float64{5, 7.5, 10, 15} {
		metric := phase11CVBMetric(events, horizon, cost, false)
		stats.CostStress = append(stats.CostStress, metric)
		switch cost {
		case 5:
			stats.EventCount = metric.EventCount
			stats.DeClusteredEventCount = metric.DeClusteredEventCount
			stats.PFAfter5Bps = metric.PF
			stats.ExpectancyBpsAfter5Bps = metric.ExpectancyBps
			stats.WinRate = metric.WinRate
			stats.GrossProfitBps = metric.GrossProfitBps
			stats.GrossLossBps = metric.GrossLossBps
			stats.NetBps = metric.NetBps
			stats.WinCount = metric.WinCount
			stats.LossCount = metric.LossCount
			stats.AverageReturnBps = averagePhase11CVBReturn(events, horizon)
			stats.MedianReturnBps = medianPhase11CVBReturn(events, horizon)
		case 7.5:
			stats.PFAfter7_5Bps = metric.PF
		case 10:
			stats.PFAfter10Bps = metric.PF
		case 15:
			stats.PFAfter15Bps = metric.PF
		}
	}
	for _, delay := range []int{0, 1} {
		metric := phase11CVBDelayMetricFromCostMetric(phase11CVBMetric(events, horizon, 5, delay == 1), delay)
		stats.DelayStress = append(stats.DelayStress, metric)
		if delay == 1 {
			stats.EntryDelay1CExpectancyBps = metric.ExpectancyBps
			stats.EntryDelay1CAvailable = metric.Available
		}
	}
	return stats
}

func phase11CVBMetric(events []phase11CVBEvent, horizon string, cost float64, delayed bool) phase11CVBCostMetric {
	var returns []float64
	var eventTimes []int64
	for _, e := range events {
		retMap := e.ReturnsBps
		if delayed {
			retMap = e.DelayBps
		}
		ret, ok := retMap[horizon]
		if !ok {
			continue
		}
		returns = append(returns, ret-cost)
		eventTimes = append(eventTimes, e.EventTimeMS)
	}
	metric := phase11CVBCostMetric{CostBps: cost, EventCount: len(returns), DeClusteredEventCount: phase11CVBDeclusteredCount(eventTimes)}
	if len(returns) == 0 {
		return metric
	}
	for _, ret := range returns {
		metric.NetBps += ret
		if ret > 0 {
			metric.GrossProfitBps += ret
			metric.WinCount++
		} else if ret < 0 {
			metric.GrossLossBps += -ret
			metric.LossCount++
		}
	}
	metric.ExpectancyBps = metric.NetBps / float64(len(returns))
	if metric.GrossLossBps > 0 {
		metric.PF = metric.GrossProfitBps / metric.GrossLossBps
	} else if metric.GrossProfitBps > 0 {
		metric.PF = 999
	}
	metric.WinRate = float64(metric.WinCount) / float64(len(returns)) * 100
	return roundPhase11CVBCostMetric(metric)
}

func phase11CVBDelayMetricFromCostMetric(m phase11CVBCostMetric, delay int) phase11CVBDelayMetric {
	label := "baseline"
	if delay > 0 {
		label = fmt.Sprintf("delay_%dc", delay)
	}
	return phase11CVBDelayMetric{
		DelayCandles:          delay,
		Label:                 label,
		Available:             m.EventCount > 0,
		EventCount:            m.EventCount,
		DeClusteredEventCount: m.DeClusteredEventCount,
		GrossProfitBps:        m.GrossProfitBps,
		GrossLossBps:          m.GrossLossBps,
		NetBps:                m.NetBps,
		ExpectancyBps:         m.ExpectancyBps,
		PF:                    m.PF,
		WinCount:              m.WinCount,
		LossCount:             m.LossCount,
		WinRate:               m.WinRate,
	}
}

func buildPhase11CVBLeaderboard(rows []phase11CVBSummaryRow) []phase11CVBLeaderRow {
	type agg struct {
		rows []phase11CVBSummaryRow
	}
	groups := make(map[string]agg)
	for _, row := range rows {
		key := row.Family + "|" + row.Side + "|" + row.Horizon
		g := groups[key]
		g.rows = append(g.rows, row)
		groups[key] = g
	}
	var out []phase11CVBLeaderRow
	for key, group := range groups {
		parts := strings.Split(key, "|")
		row := phase11CVBLeaderRow{Family: parts[0], Side: parts[1], Horizon: parts[2], LeakageStatus: "PASS"}
		var posMonths []float64
		var totalPositive float64
		quarterReturns := make(map[string][]float64)
		var delayNet float64
		var delayCount int
		var cost10Profit, cost10Loss float64
		for _, item := range group.rows {
			s := item.Stats
			row.EventCount += s.EventCount
			row.DeClusteredEventCount += s.DeClusteredEventCount
			if item.Diagnostics.LeakageStatus != "PASS" {
				row.LeakageStatus = item.Diagnostics.LeakageStatus
			}
			if s.NetBps > 0 {
				row.PositiveMonthCount++
				posMonths = append(posMonths, s.NetBps)
				totalPositive += s.NetBps
			}
			for _, c := range s.CostStress {
				if c.CostBps == 5 {
					quarterReturns[item.Quarter] = append(quarterReturns[item.Quarter], c.NetBps)
				}
				if c.CostBps == 10 {
					cost10Profit += c.GrossProfitBps
					cost10Loss += c.GrossLossBps
				}
			}
			if s.EntryDelay1CAvailable {
				delayNet += s.EntryDelay1CExpectancyBps * float64(s.EventCount)
				delayCount += s.EventCount
			}
		}
		base := phase11CVBCombinedMetric(group.rows, 5)
		row.PFCombined5Bps = base.PF
		row.ExpectancyCombined5BpsBps = base.ExpectancyBps
		if delayCount > 0 {
			row.EntryDelay1CExpectancyBps = delayNet / float64(delayCount)
		}
		if cost10Loss > 0 {
			row.Cost10BpsPF = cost10Profit / cost10Loss
		} else if cost10Profit > 0 {
			row.Cost10BpsPF = 999
		}
		row.WorstQuarterPF5Bps = phase11CVBWorstQuarterPF(group.rows)
		row.Top1MonthContributionPct = phase11CVBTopContributionPct(posMonths, totalPositive, 1)
		row.Top2MonthContributionPct = phase11CVBTopContributionPct(posMonths, totalPositive, 2)
		in := CandidateVerdictInput{
			H2Expectancy5bpsBps:        row.ExpectancyCombined5BpsBps,
			H2PF5bps:                   row.PFCombined5Bps,
			FYPF5bps:                   row.PFCombined5Bps,
			EventCount:                 row.EventCount,
			H1EventCount:               row.EventCount,
			H2EventCount:               row.EventCount,
			PositiveMonthCount:         row.PositiveMonthCount,
			EntryDelay1cExpectancyBps:  row.EntryDelay1CExpectancyBps,
			SingleMonthContributionPct: row.Top1MonthContributionPct,
			Top2MonthContributionPct:   row.Top2MonthContributionPct,
			WorstQuarterPF5bps:         row.WorstQuarterPF5Bps,
			Cost10bpsReported:          true,
			LeakageStatus:              row.LeakageStatus,
		}
		row.Verdict, row.FailedGates = EvaluateCandidateVerdictWithMetrics(in)
		out = append(out, roundPhase11CVBLeaderRow(row))
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].Verdict != out[j].Verdict {
			return verdictRank(out[i].Verdict) < verdictRank(out[j].Verdict)
		}
		if out[i].Horizon != out[j].Horizon {
			return out[i].Horizon < out[j].Horizon
		}
		return out[i].Side < out[j].Side
	})
	return out
}

func phase11CVBCombinedMetric(rows []phase11CVBSummaryRow, cost float64) phase11CVBCostMetric {
	var out phase11CVBCostMetric
	out.CostBps = cost
	for _, row := range rows {
		for _, c := range row.Stats.CostStress {
			if c.CostBps != cost {
				continue
			}
			out.EventCount += c.EventCount
			out.DeClusteredEventCount += c.DeClusteredEventCount
			out.GrossProfitBps += c.GrossProfitBps
			out.GrossLossBps += c.GrossLossBps
			out.NetBps += c.NetBps
			out.WinCount += c.WinCount
			out.LossCount += c.LossCount
		}
	}
	if out.EventCount > 0 {
		out.ExpectancyBps = out.NetBps / float64(out.EventCount)
		out.WinRate = float64(out.WinCount) / float64(out.EventCount) * 100
	}
	if out.GrossLossBps > 0 {
		out.PF = out.GrossProfitBps / out.GrossLossBps
	} else if out.GrossProfitBps > 0 {
		out.PF = 999
	}
	return roundPhase11CVBCostMetric(out)
}

func phase11CVBWorstQuarterPF(rows []phase11CVBSummaryRow) float64 {
	byQuarter := make(map[string][]phase11CVBSummaryRow)
	for _, row := range rows {
		byQuarter[row.Quarter] = append(byQuarter[row.Quarter], row)
	}
	worst := math.MaxFloat64
	found := false
	for _, qRows := range byQuarter {
		m := phase11CVBCombinedMetric(qRows, 5)
		if m.EventCount == 0 {
			continue
		}
		if m.PF < worst {
			worst = m.PF
			found = true
		}
	}
	if !found {
		return 0
	}
	return roundMetric(worst)
}

func phase11CVBTopContributionPct(values []float64, total float64, n int) float64 {
	if total <= 0 || n <= 0 {
		return 0
	}
	sort.Slice(values, func(i, j int) bool { return values[i] > values[j] })
	if n > len(values) {
		n = len(values)
	}
	var top float64
	for i := 0; i < n; i++ {
		top += values[i]
	}
	return roundMetric(top / total * 100)
}

func filterPhase11CVBEvents(events []phase11CVBEvent, keep func(phase11CVBEvent) bool) []phase11CVBEvent {
	var out []phase11CVBEvent
	for _, event := range events {
		if keep(event) {
			out = append(out, event)
		}
	}
	return out
}

func phase11CVBDeclusteredCount(eventTimes []int64) int {
	if len(eventTimes) == 0 {
		return 0
	}
	sort.Slice(eventTimes, func(i, j int) bool { return eventTimes[i] < eventTimes[j] })
	count := 0
	last := int64(math.MinInt64)
	for _, ts := range eventTimes {
		if count == 0 || ts-last >= fundingClusterWindowMS {
			count++
			last = ts
		}
	}
	return count
}

func averagePhase11CVBReturn(events []phase11CVBEvent, horizon string) float64 {
	if len(events) == 0 {
		return 0
	}
	var sum float64
	var n int
	for _, e := range events {
		if ret, ok := e.ReturnsBps[horizon]; ok {
			sum += ret
			n++
		}
	}
	if n == 0 {
		return 0
	}
	return roundMetric(sum / float64(n))
}

func medianPhase11CVBReturn(events []phase11CVBEvent, horizon string) float64 {
	var vals []float64
	for _, e := range events {
		if ret, ok := e.ReturnsBps[horizon]; ok {
			vals = append(vals, ret)
		}
	}
	if len(vals) == 0 {
		return 0
	}
	sort.Float64s(vals)
	return roundMetric(median(vals))
}

func sortPhase11CVBSummaries(rows []phase11CVBSummaryRow) {
	sort.Slice(rows, func(i, j int) bool {
		a := rows[i]
		b := rows[j]
		return strings.Join([]string{a.Symbol, a.Month, a.Side, a.Horizon}, "|") <
			strings.Join([]string{b.Symbol, b.Month, b.Side, b.Horizon}, "|")
	})
}

func roundPhase11CVBCostMetric(m phase11CVBCostMetric) phase11CVBCostMetric {
	m.GrossProfitBps = roundMetric(m.GrossProfitBps)
	m.GrossLossBps = roundMetric(m.GrossLossBps)
	m.NetBps = roundMetric(m.NetBps)
	m.ExpectancyBps = roundMetric(m.ExpectancyBps)
	m.PF = roundMetric(m.PF)
	m.WinRate = roundMetric(m.WinRate)
	return m
}

func roundPhase11CVBLeaderRow(row phase11CVBLeaderRow) phase11CVBLeaderRow {
	row.PFCombined5Bps = roundMetric(row.PFCombined5Bps)
	row.ExpectancyCombined5BpsBps = roundMetric(row.ExpectancyCombined5BpsBps)
	row.WorstQuarterPF5Bps = roundMetric(row.WorstQuarterPF5Bps)
	row.EntryDelay1CExpectancyBps = roundMetric(row.EntryDelay1CExpectancyBps)
	row.Cost10BpsPF = roundMetric(row.Cost10BpsPF)
	return row
}

func renderPhase11CVBMarkdown(report phase11CVBReport) string {
	var sb strings.Builder
	sb.WriteString("# Phase 11.1 - CompressionVolumeBreakout Evaluation\n\n")
	sb.WriteString("## Boundary\n")
	for _, boundary := range report.Boundaries {
		sb.WriteString(fmt.Sprintf("- %s\n", boundary))
	}
	sb.WriteString("\n## Coverage\n")
	sb.WriteString(fmt.Sprintf("- Expected symbol-months: `%d`\n", report.Coverage.ExpectedSymbolMonths))
	sb.WriteString(fmt.Sprintf("- Completed symbol-months: `%d`\n", report.Coverage.CompletedSymbolMonths))
	sb.WriteString(fmt.Sprintf("- Raw event detail retained: `%t`\n", report.Coverage.RawEventDetailRetained))
	if len(report.Coverage.MissingSymbolMonths) > 0 {
		sb.WriteString(fmt.Sprintf("- Missing symbol-months: `%s`\n", strings.Join(report.Coverage.MissingSymbolMonths, "`, `")))
	}
	sb.WriteString("\n## Verdict Counts\n")
	for _, verdict := range verdictOrder {
		sb.WriteString(fmt.Sprintf("- %s: `%d`\n", verdict, report.VerdictCounts[verdict]))
	}
	sb.WriteString("\n## Leaderboard\n")
	sb.WriteString("| Family | Side | Horizon | Verdict | Events | De-clustered | PF 5bps | Exp 5bps (bps) | Positive Months | Worst Q PF | Delay 1c Exp | 10bps PF | Failed Gates |\n")
	sb.WriteString("|---|---|---|---|---:|---:|---:|---:|---:|---:|---:|---:|---|\n")
	for _, row := range report.Leaderboard {
		failed := "-"
		if len(row.FailedGates) > 0 {
			failed = strings.Join(row.FailedGates, "; ")
		}
		sb.WriteString(fmt.Sprintf("| %s | %s | %s | %s | %d | %d | %.4f | %.4f | %d | %.4f | %.4f | %.4f | %s |\n",
			row.Family, row.Side, row.Horizon, row.Verdict, row.EventCount, row.DeClusteredEventCount,
			row.PFCombined5Bps, row.ExpectancyCombined5BpsBps, row.PositiveMonthCount, row.WorstQuarterPF5Bps,
			row.EntryDelay1CExpectancyBps, row.Cost10BpsPF, failed))
	}
	sb.WriteString("\n## Final Recommendation\n")
	sb.WriteString(report.FinalRecommendation + "\n")
	return sb.String()
}
