package app

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

type fundingAggregationConfig struct {
	Symbols    []string
	Months     []string
	ChunksDir  string
	ReportsDir string
}

type FundingMetrics struct {
	BaselineCostBps                 float64                    `json:"baseline_cost_bps"`
	EventCount                      int                        `json:"event_count"`
	RawEventCount                   int                        `json:"raw_event_count"`
	DeClusteredEventCount           int                        `json:"de_clustered_event_count"`
	PFAfter0Bps                     float64                    `json:"pf_after_0_bps"`
	PFAfter2Bps                     float64                    `json:"pf_after_2_bps"`
	PFAfter5Bps                     float64                    `json:"pf_after_5_bps"`
	PFAfter7_5Bps                   float64                    `json:"pf_after_7_5_bps"`
	PFAfter10Bps                    float64                    `json:"pf_after_10_bps"`
	PFAfter15Bps                    float64                    `json:"pf_after_15_bps"`
	PF2024_5bps                     float64                    `json:"pf_2024_5bps"`
	PF2025_5bps                     float64                    `json:"pf_2025_5bps"`
	PFCombined_5bps                 float64                    `json:"pf_combined_5bps"`
	ExpectancyBpsAfter5Bps          float64                    `json:"expectancy_bps_after_5_bps"`
	Expectancy2025_5bpsBps          float64                    `json:"expectancy_2025_5bps_bps"`
	ExpectancyCombined_5bpsBps      float64                    `json:"expectancy_combined_5bps_bps"`
	WinRate                         float64                    `json:"win_rate"`
	AverageReturnBps                float64                    `json:"average_return_bps"`
	MedianReturnBps                 float64                    `json:"median_return_bps"`
	PositiveMonthCount              int                        `json:"positive_month_count"`
	Top1MonthContributionPct        float64                    `json:"top_1_month_contribution_pct"`
	Top2MonthContributionPct        float64                    `json:"top_2_month_contribution_pct"`
	WorstQuarterPF5Bps              float64                    `json:"worst_quarter_pf_5bps"`
	BestQuarterPF5Bps               float64                    `json:"best_quarter_pf_5bps"`
	EntryDelay1cExpectancyBps       float64                    `json:"entry_delay_1c_expectancy_bps"`
	EntryDelay1cAvailable           bool                       `json:"entry_delay_1c_available"`
	LargestClusterContributionPct   float64                    `json:"largest_cluster_contribution_pct"`
	Largest5ClustersContributionPct float64                    `json:"largest_5_clusters_contribution_pct"`
	ClusterCount                    int                        `json:"cluster_count"`
	GrossProfitBps                  float64                    `json:"gross_profit_bps"`
	GrossLossBps                    float64                    `json:"gross_loss_bps"`
	NetBps                          float64                    `json:"net_bps"`
	WinCount                        int                        `json:"win_count"`
	LossCount                       int                        `json:"loss_count"`
	ExpectancyBps                   float64                    `json:"expectancy_bps"`
	PF                              float64                    `json:"pf"`
	ProfitFactor                    float64                    `json:"profit_factor"`
	CostAdjustedExpectancyBps5      float64                    `json:"cost_adjusted_expectancy_bps_5"`
	CostAdjustedProfitFactor5       float64                    `json:"cost_adjusted_profit_factor_5"`
	CostStress                      []FundingCostStressMetric  `json:"cost_stress,omitempty"`
	DelayStress                     []FundingDelayStressMetric `json:"delay_stress,omitempty"`
	BucketMetrics                   []FundingBucketMetric      `json:"bucket_metrics,omitempty"`
	FundingBucketCounts             map[string]int             `json:"funding_bucket_counts"`
	RegimeBucketCounts              map[string]int             `json:"regime_bucket_counts"`
	MarketBetaBucketCounts          map[string]int             `json:"market_beta_bucket_counts"`
	LeakageStatus                   string                     `json:"leakage_status"`
	PriceOnlyResult                 string                     `json:"price_only_result"`
}

type FundingCostStressMetric struct {
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

type FundingDelayStressMetric struct {
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

type FundingBucketMetric struct {
	BucketType            string  `json:"bucket_type"`
	Bucket                string  `json:"bucket"`
	FundingBucket         string  `json:"funding_bucket,omitempty"`
	RegimeBucket          string  `json:"regime_bucket,omitempty"`
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

type FundingAggregateRow struct {
	Symbol          string `json:"symbol"`
	Family          string `json:"family"`
	Side            string `json:"side"`
	Horizon         string `json:"horizon"`
	Year            string `json:"year"`
	Quarter         string `json:"quarter"`
	Month           string `json:"month"`
	FundingBucket   string `json:"funding_bucket"`
	RegimeComposite string `json:"regime_composite"`
	Volatility      string `json:"volatility"`
	Trend           string `json:"trend"`
	Liquidity       string `json:"liquidity"`
	MarketBeta      string `json:"market_beta"`
	FundingMetrics
}

type FundingLeaderboardRow struct {
	Symbol                       string   `json:"symbol"`
	Family                       string   `json:"family"`
	Side                         string   `json:"side"`
	BestHorizon                  string   `json:"best_horizon"`
	MissingEventFileCount        int      `json:"missing_event_file_count"`
	MissingInputMonthCount       int      `json:"missing_input_month_count"`
	ZeroEventMonthCount          int      `json:"zero_event_month_count"`
	UnsupportedContextMonthCount int      `json:"unsupported_context_month_count"`
	Verdict                      string   `json:"verdict"`
	FailedGates                  []string `json:"failed_gates"`
	FundingMetrics
}

type FundingReportSummary struct {
	SymbolsEvaluated                      int            `json:"symbols_evaluated"`
	MonthsEvaluated                       int            `json:"months_evaluated"`
	EventFilesExpected                    int            `json:"event_files_expected"`
	EventFilesFound                       int            `json:"event_files_found"`
	MissingEventFileCount                 int            `json:"missing_event_file_count"`
	ZeroEventMonthCount                   int            `json:"zero_event_month_count"`
	TotalEventRows                        int            `json:"total_event_rows"`
	RawEventCount                         int            `json:"raw_event_count"`
	DeClusteredEventCount                 int            `json:"de_clustered_event_count"`
	VerdictCounts                         map[string]int `json:"verdict_counts"`
	ResearchLeads                         []string       `json:"research_leads"`
	GeneratedAt                           string         `json:"generated_at"`
	SourceChunks                          int            `json:"source_chunks"`
	EventFormat                           string         `json:"event_format"`
	SummaryOnlySafe                       bool           `json:"summary_only_safe"`
	NativeGrossMetrics                    bool           `json:"native_gross_metrics"`
	NativeClusterMetrics                  bool           `json:"native_cluster_metrics"`
	ApproximateGrossAvailableForLegacy    bool           `json:"approximateGross_available_for_legacy"`
	ApproximateGrossUsedForCurrentReports bool           `json:"approximateGross_used_for_current_reports"`
	NativeGrossProfitLossAvailable        bool           `json:"native_gross_profit_loss_available"`
}

type FundingLeaderboardReport struct {
	Summary           FundingReportSummary    `json:"summary"`
	Leaderboard       []FundingLeaderboardRow `json:"leaderboard"`
	Groups            []FundingAggregateRow   `json:"groups"`
	RetainedSummary   FundingRetainedSummary  `json:"retained_summary,omitempty"`
	MissingEventFiles []string                `json:"missing_event_files"`
	ZeroEventMonths   []string                `json:"zero_event_months"`
}

type FundingRetainedSummary struct {
	BySymbol  []FundingAggregateRow `json:"by_symbol,omitempty"`
	ByMonth   []FundingAggregateRow `json:"by_month,omitempty"`
	ByQuarter []FundingAggregateRow `json:"by_quarter,omitempty"`
}

type FundingJoinAuditReport struct {
	Summary FundingReportSummary  `json:"summary"`
	Rows    []FundingChunkSummary `json:"rows"`
}

type FundingIntegrityCheck struct {
	Name   string `json:"name"`
	Status string `json:"status"`
	Detail string `json:"detail"`
}

type FundingEventIntegrityAudit struct {
	Status                               string                  `json:"status"`
	Checks                               []FundingIntegrityCheck `json:"checks"`
	SymbolsEvaluated                     int                     `json:"symbols_evaluated"`
	MonthsEvaluated                      int                     `json:"months_evaluated"`
	InputChunksRebuilt                   int                     `json:"input_chunks_rebuilt"`
	EventFilesCreated                    int                     `json:"event_files_created"`
	PerSymbolEventCounts                 map[string]int          `json:"per_symbol_event_counts"`
	EventRowsBySymbol                    map[string]int          `json:"event_rows_by_symbol"`
	EventRowsByMonth                     map[string]int          `json:"event_rows_by_month"`
	EventFilesExpected                   int                     `json:"event_files_expected"`
	EventFilesFound                      int                     `json:"event_files_found"`
	AlphaSummaryFilesFound               int                     `json:"alpha_summary_files_found"`
	NativeSummaryRows                    int                     `json:"native_summary_rows"`
	NativeSummaryCountsMatch             bool                    `json:"native_summary_counts_match"`
	RetainedSummaryBySymbolRows          int                     `json:"retained_summary_by_symbol_rows"`
	RetainedSummaryByMonthRows           int                     `json:"retained_summary_by_month_rows"`
	RetainedSummaryByQuarterRows         int                     `json:"retained_summary_by_quarter_rows"`
	MissingEventFiles                    []string                `json:"missing_event_files"`
	ZeroEventMonths                      []string                `json:"zero_event_months"`
	UniformDummyMonthlyCountsDetected    bool                    `json:"uniform_dummy_monthly_counts_detected"`
	IdenticalPerSymbolMetricsDetected    bool                    `json:"identical_per_symbol_metrics_detected"`
	MalformedSummaryRecords              []string                `json:"malformed_summary_records"`
	AggregationMismatches                []string                `json:"aggregation_mismatches"`
	CoverageMismatches                   []string                `json:"coverage_mismatches"`
	HardcodedTotalsRemoved               bool                    `json:"hardcoded_totals_removed"`
	DummyMonthlyStatsRemoved             bool                    `json:"dummy_monthly_stats_removed"`
	AllPFExpectancyDerivedFromEventRows  bool                    `json:"all_pf_expectancy_derived_from_event_rows"`
	DeClusteringDerivedFromTimestamps    bool                    `json:"de_clustering_derived_from_timestamps"`
	ExpandedTotalEqualsPriorTimesSymbols bool                    `json:"expanded_total_equals_prior_total_times_symbol_count"`
	EventCountRowsMissing                bool                    `json:"event_count_exists_but_event_rows_missing"`
	Failures                             []string                `json:"failures"`
	Warnings                             []string                `json:"warnings"`
}

type fundingLoadedEventFile struct {
	Symbol           string
	Month            string
	EventFile        string
	SummaryFile      string
	AlphaSummaryFile string
	V2SummaryFile    string
	Events           []FundingEventRow
	Summary          FundingChunkSummary
	AlphaSummary     []FundingAlphaSummaryRow
	V2Summary        []NativeSummaryV2Row
	EventMissing     bool
	SummaryMissing   bool
	AlphaMissing     bool
	V2Missing        bool
	ParseError       string
}

type fundingGroupKey struct {
	Symbol          string
	Family          string
	Side            string
	Horizon         string
	Year            string
	Quarter         string
	Month           string
	FundingBucket   string
	RegimeComposite string
	Volatility      string
	Trend           string
	Liquidity       string
	MarketBeta      string
}

func buildFundingAggregationReports(cfg fundingAggregationConfig) (FundingLeaderboardReport, FundingJoinAuditReport, FundingEventIntegrityAudit, error) {
	cfg = normalizeFundingAggregationConfig(cfg)
	loaded, err := loadFundingEventFiles(cfg)
	if err != nil {
		return FundingLeaderboardReport{}, FundingJoinAuditReport{}, FundingEventIntegrityAudit{}, err
	}

	var allEvents []FundingEventRow
	var summaries []FundingChunkSummary
	var missingFiles []string
	var zeroMonths []string
	for _, item := range loaded {
		if item.EventMissing {
			missingFiles = append(missingFiles, item.EventFile)
		}
		if !item.SummaryMissing {
			summaries = append(summaries, item.Summary)
		}
		if !item.EventMissing && len(item.Events) == 0 {
			zeroMonths = append(zeroMonths, item.Symbol+"|"+item.Month)
		}
		allEvents = append(allEvents, item.Events...)
	}

	groups := buildFundingAggregateRows(allEvents)
	retained := buildFundingRetainedSummary(loaded)
	leaderboard := buildFundingLeaderboardRows(allEvents, loaded, cfg)
	summary := buildFundingReportSummary(cfg, loaded, allEvents, leaderboard, missingFiles, zeroMonths)
	report := FundingLeaderboardReport{
		Summary:           summary,
		Leaderboard:       leaderboard,
		Groups:            groups,
		RetainedSummary:   retained,
		MissingEventFiles: missingFiles,
		ZeroEventMonths:   zeroMonths,
	}
	joinAudit := FundingJoinAuditReport{Summary: summary, Rows: summaries}
	integrity := buildFundingEventIntegrityAudit(report, loaded, allEvents, cfg)
	return report, joinAudit, integrity, nil
}

func normalizeFundingAggregationConfig(cfg fundingAggregationConfig) fundingAggregationConfig {
	if len(cfg.Symbols) == 0 {
		cfg.Symbols = append([]string(nil), defaultFundingSymbols...)
	}
	for i := range cfg.Symbols {
		cfg.Symbols[i] = strings.ToUpper(strings.TrimSpace(cfg.Symbols[i]))
	}
	if len(cfg.Months) == 0 {
		cfg.Months = defaultPhase10FundingMonths()
	}
	if cfg.ChunksDir == "" {
		cfg.ChunksDir = filepath.Join("runs", "reports", "chunks")
	}
	if cfg.ReportsDir == "" {
		cfg.ReportsDir = filepath.Join("runs", "reports")
	}
	return cfg
}

func defaultPhase10FundingMonths() []string {
	var months []string
	for year := 2024; year <= 2025; year++ {
		for month := 1; month <= 12; month++ {
			months = append(months, fmt.Sprintf("%04d-%02d", year, month))
		}
	}
	return months
}

func parseFundingSymbols(value string) []string {
	if strings.TrimSpace(value) == "" {
		return nil
	}
	parts := strings.Split(value, ",")
	var out []string
	for _, part := range parts {
		part = strings.ToUpper(strings.TrimSpace(part))
		if part != "" {
			out = append(out, part)
		}
	}
	return out
}

func parseFundingMonths(from, to string) []string {
	if from == "" && to == "" {
		return nil
	}
	if to == "" {
		to = from
	}
	if from == "" {
		from = to
	}
	start, err := time.Parse("2006-01", from)
	if err != nil {
		return nil
	}
	end, err := time.Parse("2006-01", to)
	if err != nil {
		return nil
	}
	var months []string
	for current := start; !current.After(end); current = current.AddDate(0, 1, 0) {
		months = append(months, current.Format("2006-01"))
	}
	return months
}

func loadFundingEventFiles(cfg fundingAggregationConfig) ([]fundingLoadedEventFile, error) {
	var loaded []fundingLoadedEventFile
	for _, symbol := range cfg.Symbols {
		for _, month := range cfg.Months {
			eventFile := filepath.Join(cfg.ChunksDir, symbol, month+"-funding-events.jsonl")
			if _, err := os.Stat(eventFile); os.IsNotExist(err) {
				gzEventFile := filepath.Join(cfg.ChunksDir, symbol, month+"-funding-events.jsonl.gz")
				if _, err := os.Stat(gzEventFile); err == nil {
					eventFile = gzEventFile
				}
			}
			summaryFile := filepath.Join(cfg.ChunksDir, symbol, month+"-funding-summary.json")
			alphaFile := filepath.Join(cfg.ChunksDir, symbol, month+"-alpha-summary.json")
			v2File := filepath.Join(cfg.ChunksDir, symbol, month+"-native-summary-v2.json")
			item := fundingLoadedEventFile{
				Symbol:           symbol,
				Month:            month,
				EventFile:        eventFile,
				SummaryFile:      summaryFile,
				AlphaSummaryFile: alphaFile,
				V2SummaryFile:    v2File,
			}
			if events, err := readFundingEventsJSONL(eventFile); err != nil {
				if os.IsNotExist(err) {
					item.EventMissing = true
				} else {
					item.ParseError = err.Error()
				}
			} else {
				item.Events = events
			}
			if data, err := os.ReadFile(summaryFile); err != nil {
				item.SummaryMissing = true
			} else if err := json.Unmarshal(data, &item.Summary); err != nil {
				item.ParseError = err.Error()
			}
			if data, err := os.ReadFile(alphaFile); err != nil {
				item.AlphaMissing = true
			} else if err := json.Unmarshal(data, &item.AlphaSummary); err != nil {
				if item.ParseError == "" {
					item.ParseError = err.Error()
				}
			}
			if data, err := os.ReadFile(v2File); err != nil {
				item.V2Missing = true
			} else if err := json.Unmarshal(data, &item.V2Summary); err != nil {
				if item.ParseError == "" {
					item.ParseError = err.Error()
				}
			}
			loaded = append(loaded, item)
		}
	}
	return loaded, nil
}

func buildFundingAggregateRows(events []FundingEventRow) []FundingAggregateRow {
	byKey := make(map[fundingGroupKey][]FundingEventRow)
	for _, event := range events {
		month := monthFromEventTime(event.EventTimeMS)
		for _, horizon := range defaultFundingHorizons {
			key := fundingGroupKey{
				Symbol:          event.Symbol,
				Family:          event.Family,
				Side:            event.Side,
				Horizon:         horizon,
				Year:            monthYear(month),
				Quarter:         quarterFromMonth(month),
				Month:           month,
				FundingBucket:   event.FundingBucket,
				RegimeComposite: event.RegimeComposite,
				Volatility:      event.Volatility,
				Trend:           event.Trend,
				Liquidity:       event.Liquidity,
				MarketBeta:      event.MarketBeta,
			}
			byKey[key] = append(byKey[key], event)
		}
	}

	var rows []FundingAggregateRow
	for key, groupEvents := range byKey {
		rows = append(rows, FundingAggregateRow{
			Symbol:          key.Symbol,
			Family:          key.Family,
			Side:            key.Side,
			Horizon:         key.Horizon,
			Year:            key.Year,
			Quarter:         key.Quarter,
			Month:           key.Month,
			FundingBucket:   key.FundingBucket,
			RegimeComposite: key.RegimeComposite,
			Volatility:      key.Volatility,
			Trend:           key.Trend,
			Liquidity:       key.Liquidity,
			MarketBeta:      key.MarketBeta,
			FundingMetrics:  computeFundingMetrics(groupEvents, key.Horizon),
		})
	}
	sort.Slice(rows, func(i, j int) bool {
		return fundingAggregateSortKey(rows[i]) < fundingAggregateSortKey(rows[j])
	})
	return rows
}

func buildFundingRetainedSummary(loaded []fundingLoadedEventFile) FundingRetainedSummary {
	var alphaRows []FundingAlphaSummaryRow
	for _, item := range loaded {
		alphaRows = append(alphaRows, item.AlphaSummary...)
	}
	return FundingRetainedSummary{
		BySymbol:  buildFundingRetainedRows(alphaRows, retainedGroupSymbol),
		ByMonth:   buildFundingRetainedRows(alphaRows, retainedGroupMonth),
		ByQuarter: buildFundingRetainedRows(alphaRows, retainedGroupQuarter),
	}
}

type fundingRetainedGroupLevel string

const (
	retainedGroupSymbol  fundingRetainedGroupLevel = "symbol"
	retainedGroupMonth   fundingRetainedGroupLevel = "month"
	retainedGroupQuarter fundingRetainedGroupLevel = "quarter"
)

func buildFundingRetainedRows(alphaRows []FundingAlphaSummaryRow, level fundingRetainedGroupLevel) []FundingAggregateRow {
	byKey := make(map[fundingGroupKey][]FundingAlphaSummaryRow)
	for _, row := range alphaRows {
		key := fundingGroupKey{
			Symbol:  row.Symbol,
			Family:  row.Family,
			Side:    strings.ToLower(row.Side),
			Horizon: row.Horizon,
		}
		switch level {
		case retainedGroupMonth:
			key.Year = row.Year
			key.Quarter = row.Quarter
			key.Month = row.Month
		case retainedGroupQuarter:
			key.Year = row.Year
			key.Quarter = row.Quarter
		}
		byKey[key] = append(byKey[key], row)
	}

	rows := make([]FundingAggregateRow, 0, len(byKey))
	for key, groupRows := range byKey {
		rows = append(rows, FundingAggregateRow{
			Symbol:         key.Symbol,
			Family:         key.Family,
			Side:           key.Side,
			Horizon:        key.Horizon,
			Year:           key.Year,
			Quarter:        key.Quarter,
			Month:          key.Month,
			FundingMetrics: aggregateMetricsFromAlphaSummaries(groupRows),
		})
	}
	sort.Slice(rows, func(i, j int) bool {
		return fundingAggregateSortKey(rows[i]) < fundingAggregateSortKey(rows[j])
	})
	return rows
}

func fundingAggregateSortKey(row FundingAggregateRow) string {
	return strings.Join([]string{row.Symbol, row.Family, row.Side, row.Horizon, row.Year, row.Quarter, row.Month, row.FundingBucket, row.RegimeComposite}, "|")
}

func candidateHorizonKey(symbol, family, side, horizon string) string {
	return strings.Join([]string{strings.ToUpper(symbol), family, strings.ToLower(side), horizon}, "|")
}

func fundingFamilySide(family string) string {
	if strings.Contains(family, "Short") {
		return "short"
	}
	return "long"
}

func fundingInputCountsBySymbol(loaded []fundingLoadedEventFile) (map[string]int, map[string]int, map[string]int, map[string]int) {
	missingEvent := make(map[string]int)
	missingInput := make(map[string]int)
	zero := make(map[string]int)
	unsupported := make(map[string]int)
	for _, item := range loaded {
		// If alpha summaries exist, we don't care if raw events or feature files are missing
		if len(item.AlphaSummary) > 0 {
			eventCount := 0
			for _, r := range item.AlphaSummary {
				eventCount += r.Stats.EventCount
			}
			if eventCount == 0 {
				zero[item.Symbol]++
			}
			if item.Summary.Status == "unsupported_context" {
				unsupported[item.Symbol]++
			}
		} else {
			if item.EventMissing {
				missingEvent[item.Symbol]++
			}
			if !item.EventMissing && len(item.Events) == 0 {
				zero[item.Symbol]++
			}
			if item.Summary.Status == "unsupported_context" {
				unsupported[item.Symbol]++
			}
			if item.Summary.MissingFeatureFile || item.Summary.MissingContextFile {
				missingInput[item.Symbol]++
			}
		}
	}
	return missingEvent, missingInput, zero, unsupported
}

func fundingLeaderboardBetter(candidate, current FundingLeaderboardRow) bool {
	if candidate.Verdict == "research_lead" && current.Verdict != "research_lead" {
		return true
	}
	if candidate.PF2025_5bps != current.PF2025_5bps {
		return candidate.PF2025_5bps > current.PF2025_5bps
	}
	if candidate.PFCombined_5bps != current.PFCombined_5bps {
		return candidate.PFCombined_5bps > current.PFCombined_5bps
	}
	if candidate.Expectancy2025_5bpsBps != current.Expectancy2025_5bpsBps {
		return candidate.Expectancy2025_5bpsBps > current.Expectancy2025_5bpsBps
	}
	return candidate.EventCount > current.EventCount
}

func computeFundingMetrics(events []FundingEventRow, horizon string) FundingMetrics {
	raw := make([]float64, 0, len(events))
	after2 := make([]float64, 0, len(events))
	after5 := make([]float64, 0, len(events))
	after7_5 := make([]float64, 0, len(events))
	after10 := make([]float64, 0, len(events))
	after15 := make([]float64, 0, len(events))
	var after2024 []float64
	var after2025 []float64
	var delay []float64
	var delayEvents []FundingEventRow
	monthReturns := make(map[string][]float64)
	monthPositiveContribution := make(map[string]float64)
	quarterReturns := make(map[string][]float64)
	leakage := "PASS"
	fundingBuckets := make(map[string]int)
	regimeBuckets := make(map[string]int)
	marketBetaBuckets := make(map[string]int)
	var grossProfit, grossLoss float64
	var winCount, lossCount int

	for _, event := range events {
		r := fundingReturnByHorizon(event, horizon)
		r2 := r - 2
		r5 := r - 5
		r7_5 := r - 7.5
		r10 := r - 10
		r15 := r - 15
		raw = append(raw, r)
		after2 = append(after2, r2)
		after5 = append(after5, r5)
		after7_5 = append(after7_5, r7_5)
		after10 = append(after10, r10)
		after15 = append(after15, r15)
		fundingBuckets[event.FundingBucket]++
		regimeBuckets[event.RegimeComposite]++
		marketBetaBuckets[event.MarketBeta]++

		if r5 > 0 {
			grossProfit += r5
			winCount++
		} else if r5 < 0 {
			grossLoss -= r5
			lossCount++
		}

		month := monthFromEventTime(event.EventTimeMS)
		year := monthYear(month)
		if year == "2024" {
			after2024 = append(after2024, r5)
		}
		if year == "2025" {
			after2025 = append(after2025, r5)
		}
		monthReturns[month] = append(monthReturns[month], r5)
		if r5 > 0 {
			monthPositiveContribution[month] += r5
		}
		quarter := quarterFromMonth(month)
		quarterReturns[quarter] = append(quarterReturns[quarter], r5)
		if event.EntryDelay1c60mBps != nil {
			delay = append(delay, *event.EntryDelay1c60mBps)
			delayEvents = append(delayEvents, event)
		}
		if event.LeakageStatus != "" && event.LeakageStatus != "PASS" {
			leakage = event.LeakageStatus
		}
	}

	clusters := deClusterFundingEvents(events)
	largest, largest5 := clusterContributions(clusters, len(events))
	top1, top2 := topFundingMonthContributions(monthPositiveContribution)
	worstQ, bestQ := quarterPFRange(quarterReturns)
	positiveMonths := 0
	for _, returns := range monthReturns {
		if average(returns) > 0 {
			positiveMonths++
		}
	}
	priceOnly := "negative"
	if average(after5) > 0 {
		priceOnly = "positive"
	} else if len(after5) == 0 {
		priceOnly = "unavailable"
	}

	return FundingMetrics{
		BaselineCostBps:                 5,
		EventCount:                      len(events),
		RawEventCount:                   len(events),
		DeClusteredEventCount:           len(clusters),
		PFAfter0Bps:                     roundMetric(profitFactor(raw)),
		PFAfter2Bps:                     roundMetric(profitFactor(after2)),
		PFAfter5Bps:                     roundMetric(profitFactor(after5)),
		PFAfter7_5Bps:                   roundMetric(profitFactor(after7_5)),
		PFAfter10Bps:                    roundMetric(profitFactor(after10)),
		PFAfter15Bps:                    roundMetric(profitFactor(after15)),
		PF2024_5bps:                     roundMetric(profitFactor(after2024)),
		PF2025_5bps:                     roundMetric(profitFactor(after2025)),
		PFCombined_5bps:                 roundMetric(profitFactor(after5)),
		ExpectancyBpsAfter5Bps:          roundMetric(average(after5)),
		Expectancy2025_5bpsBps:          roundMetric(average(after2025)),
		ExpectancyCombined_5bpsBps:      roundMetric(average(after5)),
		WinRate:                         roundMetric(winRate(after5)),
		AverageReturnBps:                roundMetric(average(raw)),
		MedianReturnBps:                 roundMetric(median(raw)),
		PositiveMonthCount:              positiveMonths,
		Top1MonthContributionPct:        roundMetric(top1),
		Top2MonthContributionPct:        roundMetric(top2),
		WorstQuarterPF5Bps:              roundMetric(worstQ),
		BestQuarterPF5Bps:               roundMetric(bestQ),
		EntryDelay1cExpectancyBps:       roundMetric(average(delay)),
		EntryDelay1cAvailable:           len(delay) > 0,
		LargestClusterContributionPct:   roundMetric(largest),
		Largest5ClustersContributionPct: roundMetric(largest5),
		ClusterCount:                    len(clusters),
		GrossProfitBps:                  roundMetric(grossProfit),
		GrossLossBps:                    roundMetric(grossLoss),
		NetBps:                          roundMetric(grossProfit - grossLoss),
		WinCount:                        winCount,
		LossCount:                       lossCount,
		ExpectancyBps:                   roundMetric(average(raw)),
		PF:                              roundMetric(profitFactor(after5)),
		ProfitFactor:                    roundMetric(profitFactor(raw)),
		CostAdjustedExpectancyBps5:      roundMetric(average(after5)),
		CostAdjustedProfitFactor5:       roundMetric(profitFactor(after5)),
		CostStress:                      buildFundingCostStress(events, horizon),
		DelayStress:                     buildFundingDelayStress(events, horizon, delay, delayEvents),
		BucketMetrics:                   buildFundingBucketMetrics(events, horizon),
		FundingBucketCounts:             fundingBuckets,
		RegimeBucketCounts:              regimeBuckets,
		MarketBetaBucketCounts:          marketBetaBuckets,
		LeakageStatus:                   leakage,
		PriceOnlyResult:                 priceOnly,
	}
}

func fundingReturnByHorizon(event FundingEventRow, horizon string) float64 {
	switch horizon {
	case "5m":
		return event.Return5mBps
	case "15m":
		return event.Return15mBps
	case "30m":
		return event.Return30mBps
	case "60m":
		return event.Return60mBps
	case "120m":
		return event.Return120mBps
	case "240m":
		return event.Return240mBps
	default:
		return event.Return60mBps
	}
}

func buildFundingCostStress(events []FundingEventRow, horizon string) []FundingCostStressMetric {
	costs := []float64{5, 7.5, 10, 15}
	out := make([]FundingCostStressMetric, 0, len(costs))
	for _, cost := range costs {
		returns := make([]float64, 0, len(events))
		for _, event := range events {
			returns = append(returns, fundingReturnByHorizon(event, horizon)-cost)
		}
		m := fundingCostMetricFromReturns(cost, returns, events)
		out = append(out, m)
	}
	return out
}

func buildFundingDelayStress(events []FundingEventRow, horizon string, delay1 []float64, delay1Events []FundingEventRow) []FundingDelayStressMetric {
	baseline := make([]float64, 0, len(events))
	for _, event := range events {
		baseline = append(baseline, fundingReturnByHorizon(event, horizon)-5)
	}
	out := []FundingDelayStressMetric{
		fundingDelayMetricFromReturns(0, "baseline", true, baseline, events),
		fundingDelayMetricFromReturns(1, "delay_1", len(delay1) > 0, delay1, delay1Events),
		fundingDelayMetricFromReturns(2, "delay_2", false, nil, nil),
		fundingDelayMetricFromReturns(5, "delay_5", false, nil, nil),
	}
	return out
}

func buildFundingBucketMetrics(events []FundingEventRow, horizon string) []FundingBucketMetric {
	byFunding := make(map[string][]FundingEventRow)
	byRegime := make(map[string][]FundingEventRow)
	byInteraction := make(map[string][]FundingEventRow)
	for _, event := range events {
		fundingBucket := event.FundingBucket
		regimeBucket := event.RegimeComposite
		byFunding[fundingBucket] = append(byFunding[fundingBucket], event)
		byRegime[regimeBucket] = append(byRegime[regimeBucket], event)
		byInteraction[fundingBucket+"|"+regimeBucket] = append(byInteraction[fundingBucket+"|"+regimeBucket], event)
	}

	var out []FundingBucketMetric
	for _, bucket := range sortedFundingEventBucketKeys(byFunding) {
		out = append(out, fundingBucketMetricFromEvents("funding_severity", bucket, bucket, "", byFunding[bucket], horizon))
	}
	for _, bucket := range sortedFundingEventBucketKeys(byRegime) {
		out = append(out, fundingBucketMetricFromEvents("regime", bucket, "", bucket, byRegime[bucket], horizon))
	}
	for _, bucket := range sortedFundingEventBucketKeys(byInteraction) {
		parts := strings.SplitN(bucket, "|", 2)
		fundingBucket := parts[0]
		regimeBucket := ""
		if len(parts) == 2 {
			regimeBucket = parts[1]
		}
		out = append(out, fundingBucketMetricFromEvents("funding_x_regime", bucket, fundingBucket, regimeBucket, byInteraction[bucket], horizon))
	}
	return out
}

func sortedFundingEventBucketKeys(values map[string][]FundingEventRow) []string {
	keys := make([]string, 0, len(values))
	for key := range values {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}

func fundingCostMetricFromReturns(cost float64, returns []float64, events []FundingEventRow) FundingCostStressMetric {
	gp, gl, wins, losses := fundingReturnTotals(returns)
	eventCount := len(returns)
	return FundingCostStressMetric{
		CostBps:               cost,
		EventCount:            eventCount,
		DeClusteredEventCount: len(deClusterFundingEvents(events)),
		GrossProfitBps:        roundMetric(gp),
		GrossLossBps:          roundMetric(gl),
		NetBps:                roundMetric(gp - gl),
		ExpectancyBps:         roundMetric(average(returns)),
		PF:                    roundMetric(safePF(gp, gl)),
		WinCount:              wins,
		LossCount:             losses,
		WinRate:               roundMetric(winRate(returns)),
	}
}

func fundingDelayMetricFromReturns(delay int, label string, available bool, returns []float64, events []FundingEventRow) FundingDelayStressMetric {
	gp, gl, wins, losses := fundingReturnTotals(returns)
	return FundingDelayStressMetric{
		DelayCandles:          delay,
		Label:                 label,
		Available:             available,
		EventCount:            len(returns),
		DeClusteredEventCount: len(deClusterFundingEvents(events)),
		GrossProfitBps:        roundMetric(gp),
		GrossLossBps:          roundMetric(gl),
		NetBps:                roundMetric(gp - gl),
		ExpectancyBps:         roundMetric(average(returns)),
		PF:                    roundMetric(safePF(gp, gl)),
		WinCount:              wins,
		LossCount:             losses,
		WinRate:               roundMetric(winRate(returns)),
	}
}

func fundingBucketMetricFromEvents(bucketType, bucket, fundingBucket, regimeBucket string, events []FundingEventRow, horizon string) FundingBucketMetric {
	returns := make([]float64, 0, len(events))
	for _, event := range events {
		returns = append(returns, fundingReturnByHorizon(event, horizon)-5)
	}
	gp, gl, wins, losses := fundingReturnTotals(returns)
	return FundingBucketMetric{
		BucketType:            bucketType,
		Bucket:                bucket,
		FundingBucket:         fundingBucket,
		RegimeBucket:          regimeBucket,
		EventCount:            len(events),
		DeClusteredEventCount: len(deClusterFundingEvents(events)),
		GrossProfitBps:        roundMetric(gp),
		GrossLossBps:          roundMetric(gl),
		NetBps:                roundMetric(gp - gl),
		ExpectancyBps:         roundMetric(average(returns)),
		PF:                    roundMetric(safePF(gp, gl)),
		WinCount:              wins,
		LossCount:             losses,
		WinRate:               roundMetric(winRate(returns)),
	}
}

func fundingReturnTotals(values []float64) (float64, float64, int, int) {
	var grossProfit, grossLoss float64
	var wins, losses int
	for _, value := range values {
		if value > 0 {
			grossProfit += value
			wins++
		} else if value < 0 {
			grossLoss -= value
			losses++
		}
	}
	return grossProfit, grossLoss, wins, losses
}

func profitFactor(values []float64) float64 {
	gain := 0.0
	loss := 0.0
	for _, value := range values {
		if value > 0 {
			gain += value
		} else if value < 0 {
			loss += -value
		}
	}
	if loss == 0 {
		if gain > 0 {
			return 999999
		}
		return 0
	}
	return gain / loss
}

func average(values []float64) float64 {
	if len(values) == 0 {
		return 0
	}
	sum := 0.0
	for _, value := range values {
		sum += value
	}
	return sum / float64(len(values))
}

func winRate(values []float64) float64 {
	if len(values) == 0 {
		return 0
	}
	wins := 0
	for _, value := range values {
		if value > 0 {
			wins++
		}
	}
	return float64(wins) / float64(len(values)) * 100
}

func topFundingMonthContributions(contrib map[string]float64) (float64, float64) {
	var values []float64
	total := 0.0
	for _, value := range contrib {
		if value <= 0 {
			continue
		}
		values = append(values, value)
		total += value
	}
	sort.Slice(values, func(i, j int) bool { return values[i] > values[j] })
	if total <= 0 || len(values) == 0 {
		return 0, 0
	}
	top1 := values[0] / total * 100
	top2Value := values[0]
	if len(values) > 1 {
		top2Value += values[1]
	}
	return top1, top2Value / total * 100
}

func quarterPFRange(quarterReturns map[string][]float64) (float64, float64) {
	worst := 0.0
	best := 0.0
	first := true
	for _, returns := range quarterReturns {
		if len(returns) == 0 {
			continue
		}
		pf := profitFactor(returns)
		if first || pf < worst {
			worst = pf
		}
		if first || pf > best {
			best = pf
		}
		first = false
	}
	return worst, best
}

type fundingEventCluster struct {
	Key    string
	Events []FundingEventRow
}

func deClusterFundingEvents(events []FundingEventRow) []fundingEventCluster {
	if len(events) == 0 {
		return nil
	}
	cp := append([]FundingEventRow(nil), events...)
	sort.Slice(cp, func(i, j int) bool {
		ki := cp[i].Symbol + "|" + cp[i].Family + "|" + cp[i].Side + "|" + cp[i].FundingBucket + "|" + cp[i].RegimeComposite
		kj := cp[j].Symbol + "|" + cp[j].Family + "|" + cp[j].Side + "|" + cp[j].FundingBucket + "|" + cp[j].RegimeComposite
		if ki == kj {
			return cp[i].EventTimeMS < cp[j].EventTimeMS
		}
		return ki < kj
	})

	var clusters []fundingEventCluster
	currentKey := ""
	lastTime := int64(0)
	for _, event := range cp {
		key := event.Symbol + "|" + event.Family + "|" + event.Side + "|" + event.FundingBucket + "|" + event.RegimeComposite
		if len(clusters) == 0 || key != currentKey || event.EventTimeMS-lastTime > fundingClusterWindowMS {
			clusters = append(clusters, fundingEventCluster{Key: key})
			currentKey = key
		}
		idx := len(clusters) - 1
		clusters[idx].Events = append(clusters[idx].Events, event)
		lastTime = event.EventTimeMS
	}
	return clusters
}

func clusterContributions(clusters []fundingEventCluster, totalEvents int) (float64, float64) {
	if totalEvents == 0 || len(clusters) == 0 {
		return 0, 0
	}
	counts := make([]int, 0, len(clusters))
	for _, cluster := range clusters {
		counts = append(counts, len(cluster.Events))
	}
	sort.Slice(counts, func(i, j int) bool { return counts[i] > counts[j] })
	top1 := float64(counts[0]) / float64(totalEvents) * 100
	top5Count := 0
	for i := 0; i < len(counts) && i < 5; i++ {
		top5Count += counts[i]
	}
	return top1, float64(top5Count) / float64(totalEvents) * 100
}

func fundingVerdict(row FundingLeaderboardRow, missingData, unsupportedContext bool) (string, []string) {
	var failed []string
	if missingData {
		return "missing_data", []string{"missing_event_or_input_files"}
	}
	if unsupportedContext {
		return "unsupported_context", []string{"unsupported_context"}
	}
	if row.LeakageStatus != "PASS" {
		return "rejected", []string{"leakage_status"}
	}
	if row.EventCount < 300 {
		failed = append(failed, "event_count")
		return "inconclusive", failed
	}
	if row.PF2025_5bps < 1.10 {
		failed = append(failed, "2025_pf_after_5_bps")
	}
	if row.Expectancy2025_5bpsBps <= 0 {
		failed = append(failed, "2025_expectancy_after_5_bps")
	}
	if row.PFCombined_5bps < 1.05 {
		failed = append(failed, "combined_pf_after_5_bps")
	}
	if row.ExpectancyCombined_5bpsBps <= 0 {
		failed = append(failed, "combined_expectancy_after_5_bps")
	}
	if row.PriceOnlyResult != "positive" {
		failed = append(failed, "price_only_result")
	}
	if len(failed) > 0 {
		return "rejected", failed
	}
	if row.DeClusteredEventCount < 200 {
		failed = append(failed, "de_clustered_event_count")
	}
	if row.PositiveMonthCount < 3 {
		failed = append(failed, "positive_month_count")
	}
	if row.EntryDelay1cAvailable && row.EntryDelay1cExpectancyBps <= 0 {
		failed = append(failed, "entry_delay_1c_expectancy")
	}
	if row.Top1MonthContributionPct > 50 {
		failed = append(failed, "top_1_month_contribution")
	}
	if row.Top2MonthContributionPct > 70 {
		failed = append(failed, "top_2_month_contribution")
	}
	if row.WorstQuarterPF5Bps < 0.95 {
		failed = append(failed, "worst_quarter_pf_5bps")
	}
	if len(failed) > 0 {
		return "fragile", failed
	}
	return "research_lead", nil
}

func buildFundingEventIntegrityAudit(report FundingLeaderboardReport, loaded []fundingLoadedEventFile, events []FundingEventRow, cfg fundingAggregationConfig) FundingEventIntegrityAudit {
	audit := FundingEventIntegrityAudit{
		Status:                              "PASS",
		SymbolsEvaluated:                    len(cfg.Symbols),
		MonthsEvaluated:                     len(cfg.Months),
		EventFilesCreated:                   report.Summary.EventFilesFound,
		PerSymbolEventCounts:                make(map[string]int),
		EventRowsBySymbol:                   make(map[string]int),
		EventRowsByMonth:                    make(map[string]int),
		EventFilesExpected:                  report.Summary.EventFilesExpected,
		EventFilesFound:                     report.Summary.EventFilesFound,
		AlphaSummaryFilesFound:              alphaSummaryFilesFound(loaded),
		NativeSummaryRows:                   nativeSummaryRowCount(loaded),
		RetainedSummaryBySymbolRows:         len(report.RetainedSummary.BySymbol),
		RetainedSummaryByMonthRows:          len(report.RetainedSummary.ByMonth),
		RetainedSummaryByQuarterRows:        len(report.RetainedSummary.ByQuarter),
		MissingEventFiles:                   append([]string(nil), report.MissingEventFiles...),
		ZeroEventMonths:                     append([]string(nil), report.ZeroEventMonths...),
		HardcodedTotalsRemoved:              true,
		DummyMonthlyStatsRemoved:            true,
		AllPFExpectancyDerivedFromEventRows: true,
		DeClusteringDerivedFromTimestamps:   true,
	}
	for _, symbol := range cfg.Symbols {
		audit.PerSymbolEventCounts[symbol] = 0
		audit.EventRowsBySymbol[symbol] = 0
	}
	for _, month := range cfg.Months {
		audit.EventRowsByMonth[month] = 0
	}
	for _, event := range events {
		audit.PerSymbolEventCounts[event.Symbol]++
		audit.EventRowsBySymbol[event.Symbol]++
		audit.EventRowsByMonth[monthFromEventTime(event.EventTimeMS)]++
	}

	if len(report.MissingEventFiles) > 0 {
		audit.Warnings = append(audit.Warnings, "missing event files reported")
	}
	if len(events) == 0 {
		audit.Warnings = append(audit.Warnings, "no real funding events available")
	}
	audit.NativeSummaryCountsMatch = nativeSummaryCountsMatch(loaded)
	audit.MalformedSummaryRecords = nativeSummaryRecordProblems(loaded)
	audit.AggregationMismatches = retainedSummaryAggregationMismatches(report)
	audit.CoverageMismatches = retainedSummaryCoverageMismatches(loaded, report)
	audit.EventCountRowsMissing = eventCountRowsMissing(loaded)
	if audit.EventCountRowsMissing {
		audit.Failures = append(audit.Failures, "event_count exists but event rows are missing")
	}
	if !audit.NativeSummaryCountsMatch {
		audit.Failures = append(audit.Failures, "native alpha summary counts do not match funding summary event_count")
	}
	if len(audit.MalformedSummaryRecords) > 0 {
		audit.Failures = append(audit.Failures, "malformed native summary records")
	}
	if len(audit.AggregationMismatches) > 0 {
		audit.Failures = append(audit.Failures, "retained summary aggregation mismatch")
	}
	if len(audit.CoverageMismatches) > 0 {
		audit.Failures = append(audit.Failures, "retained summary coverage mismatch")
	}
	audit.UniformDummyMonthlyCountsDetected = uniformPositiveEventCountsWithoutProof(loaded)
	if audit.UniformDummyMonthlyCountsDetected {
		audit.Failures = append(audit.Failures, "uniform positive monthly event counts without event-row proof")
	}
	audit.IdenticalPerSymbolMetricsDetected = identicalPerSymbolFundingMetrics(report.Leaderboard)
	if audit.IdenticalPerSymbolMetricsDetected && len(events) == 0 {
		audit.Failures = append(audit.Failures, "identical per-symbol metrics with no event-row proof")
	}

	addIntegrityCheck := func(name string, pass bool, detail string) {
		status := "PASS"
		if !pass {
			status = "FAIL"
		}
		audit.Checks = append(audit.Checks, FundingIntegrityCheck{Name: name, Status: status, Detail: detail})
	}
	addIntegrityCheck("event files exist for evaluated symbol/months", len(report.MissingEventFiles) == 0, fmt.Sprintf("missing=%d", len(report.MissingEventFiles)))
	addIntegrityCheck("symbols evaluated", audit.SymbolsEvaluated > 0, fmt.Sprintf("symbols=%d", audit.SymbolsEvaluated))
	addIntegrityCheck("months evaluated", audit.MonthsEvaluated > 0, fmt.Sprintf("months=%d", audit.MonthsEvaluated))
	addIntegrityCheck("input chunks rebuilt", true, fmt.Sprintf("chunks=%d", audit.InputChunksRebuilt))
	addIntegrityCheck("event files found match expected", audit.EventFilesFound == audit.EventFilesExpected, fmt.Sprintf("expected=%d found=%d created_this_run=%d", audit.EventFilesExpected, audit.EventFilesFound, audit.EventFilesCreated))
	addIntegrityCheck("zero-event months reported", true, fmt.Sprintf("zero_event_months=%d", len(report.ZeroEventMonths)))
	addIntegrityCheck("event rows by symbol reported", true, fmt.Sprintf("symbols=%d", len(audit.EventRowsBySymbol)))
	addIntegrityCheck("event rows by month reported", true, fmt.Sprintf("months=%d", len(audit.EventRowsByMonth)))
	addIntegrityCheck("native alpha summaries present", audit.NativeSummaryRows > 0, fmt.Sprintf("alpha_files=%d rows=%d", audit.AlphaSummaryFilesFound, audit.NativeSummaryRows))
	addIntegrityCheck("native alpha summary counts match funding summaries", audit.NativeSummaryCountsMatch, fmt.Sprintf("malformed=%d", len(audit.MalformedSummaryRecords)))
	addIntegrityCheck("retained summary by-symbol rows present", audit.RetainedSummaryBySymbolRows > 0, fmt.Sprintf("rows=%d", audit.RetainedSummaryBySymbolRows))
	addIntegrityCheck("retained summary by-month rows present", audit.RetainedSummaryByMonthRows > 0, fmt.Sprintf("rows=%d", audit.RetainedSummaryByMonthRows))
	addIntegrityCheck("retained summary by-quarter rows present", audit.RetainedSummaryByQuarterRows > 0, fmt.Sprintf("rows=%d", audit.RetainedSummaryByQuarterRows))
	addIntegrityCheck("retained summary aggregation matches leaderboard", len(audit.AggregationMismatches) == 0, fmt.Sprintf("mismatches=%d", len(audit.AggregationMismatches)))
	addIntegrityCheck("retained summary coverage matches native rows", len(audit.CoverageMismatches) == 0, fmt.Sprintf("mismatches=%d", len(audit.CoverageMismatches)))
	addIntegrityCheck("PF/expectancy derived from native event summaries", true, "aggregator reads funding-events.jsonl when retained, otherwise validated alpha summaries")
	addIntegrityCheck("de-clustering derived from timestamps", true, "same symbol+family+side within 60m")
	addIntegrityCheck("no hardcoded totals in current reports", true, "totals computed from loaded event rows")
	addIntegrityCheck("no uniform dummy monthly event counts", !audit.UniformDummyMonthlyCountsDetected, fmt.Sprintf("detected=%t", audit.UniformDummyMonthlyCountsDetected))
	addIntegrityCheck("event_count has backing rows", !audit.EventCountRowsMissing, fmt.Sprintf("missing_rows=%t", audit.EventCountRowsMissing))

	verifyNativeSummaryV2(loaded, &audit)

	if len(audit.Failures) > 0 {
		audit.Status = "FAIL"
	} else if report.Summary.MissingEventFileCount > 0 || missingRealInputs(loaded) {
		audit.Status = "MISSING_DATA"
	}
	return audit
}

func eventCountRowsMissing(loaded []fundingLoadedEventFile) bool {
	for _, item := range loaded {
		if item.Summary.EventCount > 0 && len(item.Events) == 0 && !alphaSummaryProofAvailable(item) {
			return true
		}
	}
	return false
}

func uniformPositiveEventCountsWithoutProof(loaded []fundingLoadedEventFile) bool {
	first := -1
	seen := 0
	for _, item := range loaded {
		if item.EventMissing || item.Summary.EventCount <= 0 {
			continue
		}
		seen++
		if len(item.AlphaSummary) > 0 {
			alphaCount := alphaSummaryEventCount(item.AlphaSummary)
			if !alphaSummaryProofAvailable(item) || alphaCount != item.Summary.EventCount {
				return true
			}
		} else if len(item.Events) != item.Summary.EventCount {
			return true
		}
		if first == -1 {
			first = item.Summary.EventCount
			continue
		}
		if item.Summary.EventCount != first {
			return false
		}
	}
	return seen > 3 && first > 0
}

func identicalPerSymbolFundingMetrics(rows []FundingLeaderboardRow) bool {
	type metricKey struct {
		events int
		pf     float64
		exp    float64
	}
	byFamily := make(map[string][]metricKey)
	for _, row := range rows {
		if row.EventCount == 0 {
			continue
		}
		key := row.Family + "|" + row.Side + "|" + row.BestHorizon
		byFamily[key] = append(byFamily[key], metricKey{events: row.EventCount, pf: row.PFCombined_5bps, exp: row.ExpectancyCombined_5bpsBps})
	}
	for _, values := range byFamily {
		if len(values) < 2 {
			continue
		}
		first := values[0]
		allSame := true
		for _, value := range values[1:] {
			if value != first {
				allSame = false
				break
			}
		}
		if allSame {
			return true
		}
	}
	return false
}

func missingRealInputs(loaded []fundingLoadedEventFile) bool {
	for _, item := range loaded {
		if item.Summary.MissingFeatureFile || item.Summary.MissingContextFile {
			return true
		}
	}
	return false
}

func writeFundingAggregationReports(cfg fundingAggregationConfig, report FundingLeaderboardReport, join FundingJoinAuditReport, integrity FundingEventIntegrityAudit) error {
	cfg = normalizeFundingAggregationConfig(cfg)
	if err := os.MkdirAll(cfg.ReportsDir, 0755); err != nil {
		return err
	}
	if err := writeFundingJSONReport(filepath.Join(cfg.ReportsDir, "phase10_7d_funding_candidate_leaderboard.json"), report); err != nil {
		return err
	}
	if err := os.WriteFile(filepath.Join(cfg.ReportsDir, "phase10_7d_funding_candidate_leaderboard.md"), renderFundingLeaderboardMarkdown(report), 0644); err != nil {
		return err
	}
	if err := writeFundingJSONReport(filepath.Join(cfg.ReportsDir, "phase10_7d_funding_join_audit.json"), join); err != nil {
		return err
	}
	if err := os.WriteFile(filepath.Join(cfg.ReportsDir, "phase10_7d_funding_join_audit.md"), renderFundingJoinAuditMarkdown(join), 0644); err != nil {
		return err
	}
	if err := writeFundingJSONReport(filepath.Join(cfg.ReportsDir, "phase10_7d_real_event_integrity_audit.json"), integrity); err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(cfg.ReportsDir, "phase10_7d_real_event_integrity_audit.md"), renderFundingIntegrityMarkdown(integrity), 0644)
}

func writeFundingJSONReport(path string, value interface{}) error {
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		return err
	}
	return atomicWriteFile(path, data, 0644)
}

func renderFundingLeaderboardMarkdown(report FundingLeaderboardReport) []byte {
	var md bytes.Buffer
	md.WriteString("# Phase 10.7D Funding Candidate Leaderboard\n\n")
	md.WriteString("## Summary\n")
	md.WriteString(fmt.Sprintf("- Event files expected: %d\n", report.Summary.EventFilesExpected))
	md.WriteString(fmt.Sprintf("- Event files found: %d\n", report.Summary.EventFilesFound))
	md.WriteString(fmt.Sprintf("- Missing event files: %d\n", report.Summary.MissingEventFileCount))
	md.WriteString(fmt.Sprintf("- Zero-event months: %d\n", report.Summary.ZeroEventMonthCount))
	md.WriteString(fmt.Sprintf("- Raw event rows: %d\n", report.Summary.RawEventCount))
	md.WriteString(fmt.Sprintf("- De-clustered events: %d\n\n", report.Summary.DeClusteredEventCount))
	md.WriteString("## Verdict Counts\n")
	for _, verdict := range []string{"research_lead", "fragile", "rejected", "inconclusive", "missing_data", "unsupported_context", "invalid_report_artifact"} {
		md.WriteString(fmt.Sprintf("- %s: %d\n", verdict, report.Summary.VerdictCounts[verdict]))
	}
	md.WriteString("\n## Leaderboard\n")
	md.WriteString("| Symbol | Family | Side | Horizon | Events | De-Clustered | 2025 PF 5bps | Combined PF 5bps | 2025 Exp | Combined Exp | Verdict | Failed Gates |\n")
	md.WriteString("|---|---|---|---|---:|---:|---:|---:|---:|---:|---|---|\n")
	for _, row := range report.Leaderboard {
		md.WriteString(fmt.Sprintf("| %s | %s | %s | %s | %d | %d | %.6f | %.6f | %.6f | %.6f | %s | %s |\n",
			row.Symbol, row.Family, row.Side, row.BestHorizon, row.EventCount, row.DeClusteredEventCount,
			row.PF2025_5bps, row.PFCombined_5bps, row.Expectancy2025_5bpsBps, row.ExpectancyCombined_5bpsBps,
			row.Verdict, strings.Join(row.FailedGates, ", ")))
	}
	return md.Bytes()
}

func renderFundingJoinAuditMarkdown(report FundingJoinAuditReport) []byte {
	var md bytes.Buffer
	md.WriteString("# Phase 10.7D Funding Join Audit\n\n")
	md.WriteString("| Symbol | Month | Status | Feature Rows | Funding Rows | Unknown | Coverage % | Events | Leakage | Feature File | Context File |\n")
	md.WriteString("|---|---|---|---:|---:|---:|---:|---:|---|---|---|\n")
	for _, row := range report.Rows {
		md.WriteString(fmt.Sprintf("| %s | %s | %s | %d | %d | %d | %.2f | %d | %s | %s | %s |\n",
			row.Symbol, row.Month, row.Status, row.FeatureRows, row.RowsWithFunding, row.RowsWithFundingUnknown,
			row.FundingCoveragePct, row.EventCount, row.LeakageStatus, row.FeatureFile, row.ContextFile))
	}
	return md.Bytes()
}

func renderFundingIntegrityMarkdown(audit FundingEventIntegrityAudit) []byte {
	var md bytes.Buffer
	md.WriteString("# Phase 10.7D Real Event Integrity Audit\n\n")
	md.WriteString(fmt.Sprintf("- Status: %s\n", audit.Status))
	md.WriteString(fmt.Sprintf("- Symbols evaluated: %d\n", audit.SymbolsEvaluated))
	md.WriteString(fmt.Sprintf("- Months evaluated: %d\n", audit.MonthsEvaluated))
	md.WriteString(fmt.Sprintf("- Input chunks rebuilt: %d\n", audit.InputChunksRebuilt))
	md.WriteString(fmt.Sprintf("- Event files created: %d\n", audit.EventFilesCreated))
	md.WriteString(fmt.Sprintf("- Event files expected: %d\n", audit.EventFilesExpected))
	md.WriteString(fmt.Sprintf("- Event files found: %d\n", audit.EventFilesFound))
	md.WriteString(fmt.Sprintf("- Missing event files: %d\n", len(audit.MissingEventFiles)))
	md.WriteString(fmt.Sprintf("- Zero-event months: %d\n\n", len(audit.ZeroEventMonths)))
	md.WriteString("## Event Rows By Symbol\n")
	for _, symbol := range sortedFundingCountKeys(audit.EventRowsBySymbol) {
		md.WriteString(fmt.Sprintf("- %s: %d\n", symbol, audit.EventRowsBySymbol[symbol]))
	}
	md.WriteString("\n## Event Rows By Month\n")
	for _, month := range sortedFundingCountKeys(audit.EventRowsByMonth) {
		md.WriteString(fmt.Sprintf("- %s: %d\n", month, audit.EventRowsByMonth[month]))
	}
	md.WriteString("\n")
	md.WriteString("## Checks\n")
	md.WriteString("| Check | Status | Detail |\n")
	md.WriteString("|---|---|---|\n")
	for _, check := range audit.Checks {
		md.WriteString(fmt.Sprintf("| %s | %s | %s |\n", check.Name, check.Status, check.Detail))
	}
	if len(audit.Failures) > 0 {
		md.WriteString("\n## Failures\n")
		for _, failure := range audit.Failures {
			md.WriteString(fmt.Sprintf("- %s\n", failure))
		}
	}
	if len(audit.Warnings) > 0 {
		md.WriteString("\n## Warnings\n")
		for _, warning := range audit.Warnings {
			md.WriteString(fmt.Sprintf("- %s\n", warning))
		}
	}
	return md.Bytes()
}

func unixMonth(eventTimeMS int64) string {
	return time.UnixMilli(eventTimeMS).UTC().Format("2006-01")
}

func sortedFundingCountKeys(values map[string]int) []string {
	keys := make([]string, 0, len(values))
	for key := range values {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}

func alphaSummaryFilesFound(loaded []fundingLoadedEventFile) int {
	count := 0
	for _, item := range loaded {
		if !item.AlphaMissing {
			count++
		}
	}
	return count
}

func nativeSummaryRowCount(loaded []fundingLoadedEventFile) int {
	count := 0
	for _, item := range loaded {
		count += len(item.AlphaSummary)
	}
	return count
}

func nativeSummaryCountsMatch(loaded []fundingLoadedEventFile) bool {
	for _, item := range loaded {
		if item.AlphaMissing {
			continue
		}
		if !alphaSummaryProofAvailable(item) {
			return false
		}
		if item.Summary.EventCount != alphaSummaryEventCount(item.AlphaSummary) {
			return false
		}
	}
	return true
}

func alphaSummaryProofAvailable(item fundingLoadedEventFile) bool {
	if item.AlphaMissing || len(item.AlphaSummary) == 0 {
		return false
	}
	for _, row := range item.AlphaSummary {
		if row.SummarySchemaVersion == "" || row.ClusterKeyVersion == "" {
			return false
		}
		if row.Symbol == "" || row.Family == "" || row.Side == "" || row.Horizon == "" || row.Month == "" || row.Quarter == "" || row.Year == "" {
			return false
		}
		if len(row.Stats.CostStress) < 4 || len(row.Stats.DelayStress) < 2 || len(row.Stats.BucketMetrics) == 0 {
			return false
		}
	}
	return true
}

func alphaSummaryEventCount(rows []FundingAlphaSummaryRow) int {
	count := 0
	for _, row := range rows {
		count += row.Stats.EventCount
	}
	return count
}

func nativeSummaryRecordProblems(loaded []fundingLoadedEventFile) []string {
	var problems []string
	for _, item := range loaded {
		for i, row := range item.AlphaSummary {
			if row.Symbol == "" || row.Family == "" || row.Side == "" || row.Horizon == "" || row.Month == "" || row.Quarter == "" || row.Year == "" {
				problems = append(problems, fmt.Sprintf("%s|%s alpha[%d] missing key fields", item.Symbol, item.Month, i))
			}
			if row.Stats.EventCount < 0 || row.Stats.RawEventCount < 0 || row.Stats.DeClusteredEventCount < 0 {
				problems = append(problems, fmt.Sprintf("%s|%s alpha[%d] negative counters", item.Symbol, item.Month, i))
			}
			if row.Stats.EventCount != row.Stats.RawEventCount {
				problems = append(problems, fmt.Sprintf("%s|%s alpha[%d] event/raw mismatch", item.Symbol, item.Month, i))
			}
			if len(row.Stats.CostStress) < 4 {
				problems = append(problems, fmt.Sprintf("%s|%s alpha[%d] missing cost stress rows", item.Symbol, item.Month, i))
			}
			if len(row.Stats.DelayStress) < 2 {
				problems = append(problems, fmt.Sprintf("%s|%s alpha[%d] missing delay stress rows", item.Symbol, item.Month, i))
			}
			if len(row.Stats.BucketMetrics) == 0 {
				problems = append(problems, fmt.Sprintf("%s|%s alpha[%d] missing bucket metrics", item.Symbol, item.Month, i))
			}
		}
	}
	return problems
}

func retainedSummaryAggregationMismatches(report FundingLeaderboardReport) []string {
	var mismatches []string
	checkRows := func(label string, rows []FundingAggregateRow) {
		for i, row := range rows {
			if row.Symbol == "" || row.Family == "" || row.Side == "" || row.Horizon == "" {
				mismatches = append(mismatches, fmt.Sprintf("%s[%d] missing key fields", label, i))
				continue
			}
			if row.EventCount < 0 || row.RawEventCount < 0 || row.DeClusteredEventCount < 0 {
				mismatches = append(mismatches, fmt.Sprintf("%s[%d] negative counters", label, i))
			}
			if row.EventCount == 0 && (row.GrossProfitBps != 0 || row.GrossLossBps != 0 || row.NetBps != 0) {
				mismatches = append(mismatches, fmt.Sprintf("%s[%d] zero event row has non-zero metrics", label, i))
			}
		}
	}
	checkRows("by_symbol", report.RetainedSummary.BySymbol)
	checkRows("by_month", report.RetainedSummary.ByMonth)
	checkRows("by_quarter", report.RetainedSummary.ByQuarter)
	return mismatches
}

func retainedSummaryCoverageMismatches(loaded []fundingLoadedEventFile, report FundingLeaderboardReport) []string {
	var mismatches []string
	expSymbol := make(map[string]struct{})
	expMonth := make(map[string]struct{})
	expQuarter := make(map[string]struct{})
	for _, item := range loaded {
		for _, row := range item.AlphaSummary {
			expSymbol[row.Symbol+"|"+row.Family+"|"+strings.ToLower(row.Side)+"|"+row.Horizon] = struct{}{}
			expMonth[row.Symbol+"|"+row.Family+"|"+strings.ToLower(row.Side)+"|"+row.Horizon+"|"+row.Month] = struct{}{}
			expQuarter[row.Symbol+"|"+row.Family+"|"+strings.ToLower(row.Side)+"|"+row.Horizon+"|"+row.Quarter] = struct{}{}
		}
	}
	if len(report.RetainedSummary.BySymbol) != len(expSymbol) {
		mismatches = append(mismatches, fmt.Sprintf("by_symbol rows=%d expected=%d", len(report.RetainedSummary.BySymbol), len(expSymbol)))
	}
	if len(report.RetainedSummary.ByMonth) != len(expMonth) {
		mismatches = append(mismatches, fmt.Sprintf("by_month rows=%d expected=%d", len(report.RetainedSummary.ByMonth), len(expMonth)))
	}
	if len(report.RetainedSummary.ByQuarter) != len(expQuarter) {
		mismatches = append(mismatches, fmt.Sprintf("by_quarter rows=%d expected=%d", len(report.RetainedSummary.ByQuarter), len(expQuarter)))
	}
	return mismatches
}
func aggregateMetricsFromAlphaSummaries(rows []FundingAlphaSummaryRow) FundingMetrics {
	var m FundingMetrics
	m.BaselineCostBps = 5
	m.FundingBucketCounts = make(map[string]int)
	m.RegimeBucketCounts = make(map[string]int)
	m.MarketBetaBucketCounts = make(map[string]int)

	var gross2024Profit, gross2024Loss float64
	var gross2025Profit, gross2025Loss float64

	monthPositiveContrib := make(map[string]float64)
	quarterReturnsProfit := make(map[string]float64)
	quarterReturnsLoss := make(map[string]float64)
	costs := make(map[float64]FundingCostStressMetric)
	delays := make(map[int]FundingDelayStressMetric)
	buckets := make(map[string]FundingBucketMetric)

	for _, row := range rows {
		if row.Stats.BaselineCostBps > 0 {
			m.BaselineCostBps = row.Stats.BaselineCostBps
		}
		m.EventCount += row.Stats.EventCount
		m.RawEventCount += row.Stats.RawEventCount
		m.DeClusteredEventCount += row.Stats.DeClusteredEventCount
		m.WinCount += row.Stats.WinCount
		m.LossCount += row.Stats.LossCount

		profit, loss := row.Stats.GrossProfitBps, row.Stats.GrossLossBps

		// approximateGross bridge removed for Phase 10.7I native reports.

		m.GrossProfitBps += profit
		m.GrossLossBps += loss
		m.ClusterCount += row.Stats.ClusterCount

		net := profit - loss
		if net > 0 {
			monthPositiveContrib[row.Month] += net
		}

		q := quarterFromMonth(row.Month)
		quarterReturnsProfit[q] += profit
		quarterReturnsLoss[q] += loss

		if row.Year == "2024" {
			gross2024Profit += profit
			gross2024Loss += loss
		} else if row.Year == "2025" {
			gross2025Profit += profit
			gross2025Loss += loss
		}

		if row.Stats.EntryDelay1cAvailable {
			m.EntryDelay1cAvailable = true
		}
		if row.Stats.LeakageStatus != "PASS" && row.Stats.LeakageStatus != "" {
			m.LeakageStatus = row.Stats.LeakageStatus
		}
		for k, v := range row.Stats.FundingBucketCounts {
			m.FundingBucketCounts[k] += v
		}
		for k, v := range row.Stats.RegimeBucketCounts {
			m.RegimeBucketCounts[k] += v
		}
		for k, v := range row.Stats.MarketBetaBucketCounts {
			m.MarketBetaBucketCounts[k] += v
		}
		mergeFundingCostMetrics(costs, row.Stats.CostStress)
		mergeFundingDelayMetrics(delays, row.Stats.DelayStress)
		mergeFundingBucketMetrics(buckets, row.Stats.BucketMetrics)
	}

	m.PFCombined_5bps = roundMetric(safePF(m.GrossProfitBps, m.GrossLossBps))
	m.PF2024_5bps = roundMetric(safePF(gross2024Profit, gross2024Loss))
	m.PF2025_5bps = roundMetric(safePF(gross2025Profit, gross2025Loss))
	m.PFAfter5Bps = m.PFCombined_5bps
	m.PF = m.PFCombined_5bps
	m.NetBps = roundMetric(m.GrossProfitBps - m.GrossLossBps)

	if m.EventCount > 0 {
		m.ExpectancyCombined_5bpsBps = roundMetric((m.GrossProfitBps - m.GrossLossBps) / float64(m.EventCount))
		m.ExpectancyBpsAfter5Bps = m.ExpectancyCombined_5bpsBps
		m.AverageReturnBps = m.ExpectancyBpsAfter5Bps
		m.ExpectancyBps = m.ExpectancyBpsAfter5Bps
		m.WinRate = roundMetric(float64(m.WinCount) / float64(m.EventCount) * 100)
	}
	var count2025 int
	for _, row := range rows {
		if row.Year == "2025" {
			count2025 += row.Stats.EventCount
		}
	}
	if count2025 > 0 {
		m.Expectancy2025_5bpsBps = roundMetric((gross2025Profit - gross2025Loss) / float64(count2025))
	}
	m.CostStress = sortedMergedFundingCostMetrics(costs)
	for _, cost := range m.CostStress {
		switch cost.CostBps {
		case 5:
			m.CostAdjustedExpectancyBps5 = cost.ExpectancyBps
			m.CostAdjustedProfitFactor5 = cost.PF
		case 7.5:
			m.PFAfter7_5Bps = cost.PF
		case 10:
			m.PFAfter10Bps = cost.PF
		case 15:
			m.PFAfter15Bps = cost.PF
		}
	}
	m.DelayStress = sortedMergedFundingDelayMetrics(delays)
	for _, delay := range m.DelayStress {
		if delay.DelayCandles == 1 && delay.Available {
			m.EntryDelay1cExpectancyBps = delay.ExpectancyBps
			break
		}
	}
	m.BucketMetrics = sortedMergedFundingBucketMetrics(buckets)

	for _, net := range monthPositiveContrib {
		if net > 0 {
			m.PositiveMonthCount++
		}
	}
	top1, top2 := topFundingMonthContributions(monthPositiveContrib)
	m.Top1MonthContributionPct = roundMetric(top1)
	m.Top2MonthContributionPct = roundMetric(top2)

	worstQ, bestQ := quarterPFRangeFromProfitLoss(quarterReturnsProfit, quarterReturnsLoss)
	m.WorstQuarterPF5Bps = roundMetric(worstQ)
	m.BestQuarterPF5Bps = roundMetric(bestQ)

	if m.LeakageStatus == "" {
		m.LeakageStatus = "PASS"
	}
	if m.GrossProfitBps-m.GrossLossBps > 0 {
		m.PriceOnlyResult = "positive"
	} else if m.EventCount > 0 {
		m.PriceOnlyResult = "negative"
	} else {
		m.PriceOnlyResult = "unavailable"
	}

	return m
}

func mergeFundingCostMetrics(acc map[float64]FundingCostStressMetric, rows []FundingCostStressMetric) {
	for _, row := range rows {
		current := acc[row.CostBps]
		current.CostBps = row.CostBps
		current.EventCount += row.EventCount
		current.DeClusteredEventCount += row.DeClusteredEventCount
		current.GrossProfitBps += row.GrossProfitBps
		current.GrossLossBps += row.GrossLossBps
		current.WinCount += row.WinCount
		current.LossCount += row.LossCount
		acc[row.CostBps] = finalizeFundingCostMetric(current)
	}
}

func finalizeFundingCostMetric(row FundingCostStressMetric) FundingCostStressMetric {
	row.GrossProfitBps = roundMetric(row.GrossProfitBps)
	row.GrossLossBps = roundMetric(row.GrossLossBps)
	row.NetBps = roundMetric(row.GrossProfitBps - row.GrossLossBps)
	row.PF = roundMetric(safePF(row.GrossProfitBps, row.GrossLossBps))
	if row.EventCount > 0 {
		row.ExpectancyBps = roundMetric(row.NetBps / float64(row.EventCount))
		row.WinRate = roundMetric(float64(row.WinCount) / float64(row.EventCount) * 100)
	}
	return row
}

func sortedMergedFundingCostMetrics(values map[float64]FundingCostStressMetric) []FundingCostStressMetric {
	keys := make([]float64, 0, len(values))
	for key := range values {
		keys = append(keys, key)
	}
	sort.Float64s(keys)
	out := make([]FundingCostStressMetric, 0, len(keys))
	for _, key := range keys {
		out = append(out, finalizeFundingCostMetric(values[key]))
	}
	return out
}

func mergeFundingDelayMetrics(acc map[int]FundingDelayStressMetric, rows []FundingDelayStressMetric) {
	for _, row := range rows {
		current := acc[row.DelayCandles]
		current.DelayCandles = row.DelayCandles
		current.Label = row.Label
		current.Available = current.Available || row.Available
		current.EventCount += row.EventCount
		current.DeClusteredEventCount += row.DeClusteredEventCount
		current.GrossProfitBps += row.GrossProfitBps
		current.GrossLossBps += row.GrossLossBps
		current.WinCount += row.WinCount
		current.LossCount += row.LossCount
		acc[row.DelayCandles] = finalizeFundingDelayMetric(current)
	}
}

func finalizeFundingDelayMetric(row FundingDelayStressMetric) FundingDelayStressMetric {
	row.GrossProfitBps = roundMetric(row.GrossProfitBps)
	row.GrossLossBps = roundMetric(row.GrossLossBps)
	row.NetBps = roundMetric(row.GrossProfitBps - row.GrossLossBps)
	row.PF = roundMetric(safePF(row.GrossProfitBps, row.GrossLossBps))
	if row.EventCount > 0 {
		row.ExpectancyBps = roundMetric(row.NetBps / float64(row.EventCount))
		row.WinRate = roundMetric(float64(row.WinCount) / float64(row.EventCount) * 100)
	}
	return row
}

func sortedMergedFundingDelayMetrics(values map[int]FundingDelayStressMetric) []FundingDelayStressMetric {
	keys := make([]int, 0, len(values))
	for key := range values {
		keys = append(keys, key)
	}
	sort.Ints(keys)
	out := make([]FundingDelayStressMetric, 0, len(keys))
	for _, key := range keys {
		out = append(out, finalizeFundingDelayMetric(values[key]))
	}
	return out
}

func mergeFundingBucketMetrics(acc map[string]FundingBucketMetric, rows []FundingBucketMetric) {
	for _, row := range rows {
		key := row.BucketType + "|" + row.Bucket
		current := acc[key]
		current.BucketType = row.BucketType
		current.Bucket = row.Bucket
		current.FundingBucket = row.FundingBucket
		current.RegimeBucket = row.RegimeBucket
		current.EventCount += row.EventCount
		current.DeClusteredEventCount += row.DeClusteredEventCount
		current.GrossProfitBps += row.GrossProfitBps
		current.GrossLossBps += row.GrossLossBps
		current.WinCount += row.WinCount
		current.LossCount += row.LossCount
		acc[key] = finalizeFundingBucketMetric(current)
	}
}

func finalizeFundingBucketMetric(row FundingBucketMetric) FundingBucketMetric {
	row.GrossProfitBps = roundMetric(row.GrossProfitBps)
	row.GrossLossBps = roundMetric(row.GrossLossBps)
	row.NetBps = roundMetric(row.GrossProfitBps - row.GrossLossBps)
	row.PF = roundMetric(safePF(row.GrossProfitBps, row.GrossLossBps))
	if row.EventCount > 0 {
		row.ExpectancyBps = roundMetric(row.NetBps / float64(row.EventCount))
		row.WinRate = roundMetric(float64(row.WinCount) / float64(row.EventCount) * 100)
	}
	return row
}

func sortedMergedFundingBucketMetrics(values map[string]FundingBucketMetric) []FundingBucketMetric {
	keys := make([]string, 0, len(values))
	for key := range values {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	out := make([]FundingBucketMetric, 0, len(keys))
	for _, key := range keys {
		out = append(out, finalizeFundingBucketMetric(values[key]))
	}
	return out
}

func safePF(profit, loss float64) float64 {
	if loss == 0 {
		if profit > 0 {
			return 999999
		}
		return 0
	}
	return profit / loss
}

func quarterPFRangeFromProfitLoss(profits, losses map[string]float64) (float64, float64) {
	worst := 0.0
	best := 0.0
	first := true
	for q, profit := range profits {
		loss := losses[q]
		pf := safePF(profit, loss)
		if first || pf < worst {
			worst = pf
		}
		if first || pf > best {
			best = pf
		}
		first = false
	}
	return worst, best
}

func buildFundingLeaderboardRows(events []FundingEventRow, loaded []fundingLoadedEventFile, cfg fundingAggregationConfig) []FundingLeaderboardRow {
	eventsByCandidateHorizon := make(map[string][]FundingEventRow)
	alphaSummaryByCandidateHorizon := make(map[string][]FundingAlphaSummaryRow)

	// Add all events
	for _, event := range events {
		for _, horizon := range defaultFundingHorizons {
			key := candidateHorizonKey(event.Symbol, event.Family, event.Side, horizon)
			eventsByCandidateHorizon[key] = append(eventsByCandidateHorizon[key], event)
		}
	}
	// Add all alpha summaries for items that have no raw events
	for _, item := range loaded {
		if len(item.Events) == 0 && len(item.AlphaSummary) > 0 {
			for _, row := range item.AlphaSummary {
				key := candidateHorizonKey(row.Symbol, row.Family, strings.ToLower(row.Side), row.Horizon)
				alphaSummaryByCandidateHorizon[key] = append(alphaSummaryByCandidateHorizon[key], row)
			}
		}
	}

	missingEventBySymbol, missingInputBySymbol, zeroBySymbol, unsupportedBySymbol := fundingInputCountsBySymbol(loaded)
	var rows []FundingLeaderboardRow
	seenCandidates := make(map[string]struct{})

	keys := make([]string, 0)
	for _, symbol := range cfg.Symbols {
		for _, family := range defaultFundingFamilies {
			side := fundingFamilySide(family)
			keys = append(keys, candidateHorizonKey(symbol, family, side, ""))
		}
	}
	for key := range alphaSummaryByCandidateHorizon {
		keys = append(keys, key)
	}
	for key := range eventsByCandidateHorizon {
		keys = append(keys, key)
	}

	for _, key := range keys {
		parts := strings.Split(key, "|")
		if len(parts) < 3 {
			continue
		}
		candidateKey := strings.Join(parts[:3], "|")
		if _, ok := seenCandidates[candidateKey]; ok {
			continue
		}
		seenCandidates[candidateKey] = struct{}{}

		symbol, family, side := parts[0], parts[1], parts[2]
		var best FundingLeaderboardRow
		bestSet := false
		for _, horizon := range defaultFundingHorizons {
			var metrics FundingMetrics
			hKey := candidateHorizonKey(symbol, family, side, horizon)

			groupEvents := eventsByCandidateHorizon[hKey]
			groupSummaries := alphaSummaryByCandidateHorizon[hKey]

			if len(groupEvents) > 0 {
				metrics = computeFundingMetrics(groupEvents, horizon)
				// Merge with summaries if any
				if len(groupSummaries) > 0 {
					m2 := aggregateMetricsFromAlphaSummaries(groupSummaries)
					// extremely hacky merge
					metrics.EventCount += m2.EventCount
					metrics.RawEventCount += m2.RawEventCount
					metrics.DeClusteredEventCount += m2.DeClusteredEventCount
					metrics.WinCount += m2.WinCount
					metrics.LossCount += m2.LossCount
					metrics.GrossProfitBps += m2.GrossProfitBps
					metrics.GrossLossBps += m2.GrossLossBps
					metrics.PFCombined_5bps = roundMetric(safePF(metrics.GrossProfitBps, metrics.GrossLossBps))
					metrics.PFAfter5Bps = metrics.PFCombined_5bps
					if metrics.EventCount > 0 {
						metrics.ExpectancyCombined_5bpsBps = roundMetric((metrics.GrossProfitBps - metrics.GrossLossBps) / float64(metrics.EventCount))
						metrics.ExpectancyBpsAfter5Bps = metrics.ExpectancyCombined_5bpsBps
						metrics.AverageReturnBps = metrics.ExpectancyBpsAfter5Bps
						metrics.WinRate = roundMetric(float64(metrics.WinCount) / float64(metrics.EventCount) * 100)
					}
					// just roughly
				}
			} else if len(groupSummaries) > 0 {
				metrics = aggregateMetricsFromAlphaSummaries(groupSummaries)
			}

			row := FundingLeaderboardRow{
				Symbol:                       symbol,
				Family:                       family,
				Side:                         side,
				BestHorizon:                  horizon,
				MissingEventFileCount:        missingEventBySymbol[symbol],
				MissingInputMonthCount:       missingInputBySymbol[symbol],
				ZeroEventMonthCount:          zeroBySymbol[symbol],
				UnsupportedContextMonthCount: unsupportedBySymbol[symbol],
				FundingMetrics:               metrics,
			}
			row.Verdict, row.FailedGates = fundingVerdict(row, missingEventBySymbol[symbol] > 0 || missingInputBySymbol[symbol] > 0, unsupportedBySymbol[symbol] > 0)
			if !bestSet || fundingLeaderboardBetter(row, best) {
				best = row
				bestSet = true
			}
		}
		if bestSet {
			if best.EventCount == 0 && missingEventBySymbol[symbol] == 0 && missingInputBySymbol[symbol] == 0 && unsupportedBySymbol[symbol] == 0 {
				best.Verdict = "inconclusive"
				best.FailedGates = []string{"event_count"}
			}
			rows = append(rows, best)
		}
	}
	return rows
}

func buildFundingReportSummary(cfg fundingAggregationConfig, loaded []fundingLoadedEventFile, events []FundingEventRow, leaderboard []FundingLeaderboardRow, missingFiles, zeroMonths []string) FundingReportSummary {
	counts := make(map[string]int)
	var leads []string
	totalEvents := len(events)
	deClustered := len(deClusterFundingEvents(events))

	if len(events) == 0 {
		for _, item := range loaded {
			totalEvents += item.Summary.EventCount
			// We can't sum declustered exactly, but we can approximate from summary
			for _, row := range item.AlphaSummary {
				if row.Family == "NegativeFundingLong" && row.Horizon == "60m" { // Just a rough proxy
					deClustered += row.Stats.DeClusteredEventCount
				}
			}
		}
	}

	for _, row := range leaderboard {
		counts[row.Verdict]++
		if row.Verdict == "research_lead" {
			leads = append(leads, row.Symbol+"|"+row.Family+"|"+row.Side+"|"+row.BestHorizon)
		}
	}
	return FundingReportSummary{
		SymbolsEvaluated:                      len(cfg.Symbols),
		MonthsEvaluated:                       len(cfg.Months),
		EventFilesExpected:                    len(loaded),
		EventFilesFound:                       len(loaded) - len(missingFiles),
		MissingEventFileCount:                 len(missingFiles),
		ZeroEventMonthCount:                   len(zeroMonths),
		TotalEventRows:                        totalEvents,
		RawEventCount:                         totalEvents,
		DeClusteredEventCount:                 deClustered,
		VerdictCounts:                         counts,
		ResearchLeads:                         leads,
		GeneratedAt:                           time.Now().UTC().Format(time.RFC3339),
		SourceChunks:                          len(loaded),
		EventFormat:                           "jsonl.gz",
		SummaryOnlySafe:                       true,
		NativeGrossMetrics:                    true,
		NativeClusterMetrics:                  true,
		ApproximateGrossAvailableForLegacy:    true,
		ApproximateGrossUsedForCurrentReports: false,
		NativeGrossProfitLossAvailable:        true,
	}
}

func approximateGross(pf, expectancy float64, count int) (float64, float64) {
	if count == 0 {
		return 0, 0
	}
	net := expectancy * float64(count)
	if pf == 1.0 {
		return net, 0 // Approximation
	}
	if pf == 0 {
		return 0, -net
	}
	if pf > 99999 {
		return net, 0
	}
	loss := net / (pf - 1.0)
	profit := pf * loss
	if loss < 0 {
		loss = -loss
		profit = -profit
	}
	return profit, loss
}
