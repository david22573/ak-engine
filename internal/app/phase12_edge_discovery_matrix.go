package app

import (
	"context"
	"encoding/csv"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/david22573/ak-engine/internal/data"
	"github.com/david22573/ak-engine/internal/features"
	"github.com/david22573/ak-engine/internal/research"
	"github.com/david22573/ak-engine/pkg/protocol"
	"github.com/spf13/cobra"
)

var (
	p120Workdir  string
	p120Symbols  string
	p120Market   string
	p120Interval string
	p120From     string
	p120To       string
	p120Out      string
)

var phase120HorizonSet = []string{"5m", "15m", "30m", "60m", "120m", "240m"}

const (
	phase120SampleEveryMS     = int64(15 * 60 * 1000)
	phase120LabelRejected     = "DISCOVERY_REJECTED"
	phase120LabelWeak         = "WEAK_EDGE"
	phase120LabelRobust       = "ROBUST_EDGE_CANDIDATE"
	phase120LabelInsufficient = "DATA_INSUFFICIENT"
)

type phase120Report struct {
	Phase                     string                  `json:"phase"`
	Mode                      string                  `json:"mode"`
	MeasurementCadence        string                  `json:"measurement_cadence"`
	FinalLabel                string                  `json:"final_label"`
	CoverageCount             int                     `json:"coverage_count"`
	RequiredSymbolMonths      int                     `json:"required_symbol_months"`
	MissingSymbolMonths       []string                `json:"missing_symbol_months,omitempty"`
	FeaturesEvaluated         []string                `json:"features_evaluated"`
	FeatureInventory          []phase120FeatureInfo   `json:"feature_inventory"`
	BucketCount               int                     `json:"bucket_count"`
	RobustEdgeCandidateCount  int                     `json:"robust_edge_candidate_count"`
	WeakEdgeCount             int                     `json:"weak_edge_count"`
	RejectedCount             int                     `json:"rejected_count"`
	DataInsufficientCount     int                     `json:"data_insufficient_count"`
	StrongestBucket           *phase120BucketSummary  `json:"strongest_bucket,omitempty"`
	RecommendedNextPhase      string                  `json:"recommended_next_phase"`
	AKTraderTouched           bool                    `json:"ak_trader_touched"`
	PromotionAllowed          bool                    `json:"promotion_allowed"`
	RawRequired               bool                    `json:"raw_required"`
	RawEventDetailRetained    bool                    `json:"raw_event_detail_retained"`
	Boundaries                []string                `json:"boundaries"`
	Buckets                   []phase120BucketSummary `json:"buckets"`
	TopExpectancyAfter5Bps    []phase120BucketSummary `json:"top_expectancy_after_5_bps"`
	TopExpectancyAfter7_5Bps  []phase120BucketSummary `json:"top_expectancy_after_7_5_bps"`
	TopProfitFactor           []phase120BucketSummary `json:"top_profit_factor"`
	RobustEdgeCandidateBucket []phase120BucketSummary `json:"robust_edge_candidate_buckets"`
	WeakEdgeBuckets           []phase120BucketSummary `json:"weak_edge_buckets"`
}

type phase120FeatureInfo struct {
	Feature   string   `json:"feature"`
	Available bool     `json:"available"`
	Buckets   []string `json:"buckets,omitempty"`
	Source    string   `json:"source"`
}

type phase120BucketSummary struct {
	Feature                   string                  `json:"feature"`
	Bucket                    string                  `json:"bucket"`
	Regime                    string                  `json:"regime"`
	Side                      string                  `json:"side"`
	Horizon                   string                  `json:"horizon"`
	Label                     string                  `json:"label"`
	FailedRules               []string                `json:"failed_rules,omitempty"`
	SampleCount               int                     `json:"sample_count"`
	ClusterCount              int                     `json:"cluster_count"`
	RawExpectancyBps          float64                 `json:"raw_expectancy_bps"`
	ExpectancyAfter5Bps       float64                 `json:"expectancy_after_5_bps"`
	ExpectancyAfter7_5Bps     float64                 `json:"expectancy_after_7_5_bps"`
	ExpectancyAfter10Bps      float64                 `json:"expectancy_after_10_bps"`
	PFAfter5Bps               float64                 `json:"pf_after_5_bps"`
	PFAfter7_5Bps             float64                 `json:"pf_after_7_5_bps"`
	PFAfter10Bps              float64                 `json:"pf_after_10_bps"`
	WinRateAfter5Bps          float64                 `json:"win_rate_after_5_bps"`
	WinRateAfter7_5Bps        float64                 `json:"win_rate_after_7_5_bps"`
	WinRateAfter10Bps         float64                 `json:"win_rate_after_10_bps"`
	GrossProfitAfter5Bps      float64                 `json:"gross_profit_after_5_bps"`
	GrossLossAfter5Bps        float64                 `json:"gross_loss_after_5_bps"`
	GrossProfitAfter7_5Bps    float64                 `json:"gross_profit_after_7_5_bps"`
	GrossLossAfter7_5Bps      float64                 `json:"gross_loss_after_7_5_bps"`
	GrossProfitAfter10Bps     float64                 `json:"gross_profit_after_10_bps"`
	GrossLossAfter10Bps       float64                 `json:"gross_loss_after_10_bps"`
	WinCountAfter5Bps         int                     `json:"win_count_after_5_bps"`
	LossCountAfter5Bps        int                     `json:"loss_count_after_5_bps"`
	WinCountAfter7_5Bps       int                     `json:"win_count_after_7_5_bps"`
	LossCountAfter7_5Bps      int                     `json:"loss_count_after_7_5_bps"`
	WinCountAfter10Bps        int                     `json:"win_count_after_10_bps"`
	LossCountAfter10Bps       int                     `json:"loss_count_after_10_bps"`
	WinRate                   float64                 `json:"win_rate"`
	ProfitFactor              float64                 `json:"profit_factor"`
	AverageForwardReturn      float64                 `json:"average_forward_return"`
	MedianForwardReturn       float64                 `json:"median_forward_return"`
	WorstMonthExpectancy      float64                 `json:"worst_month_expectancy"`
	WorstSymbolExpectancy     float64                 `json:"worst_symbol_expectancy"`
	LeaveOneSymbolOut         phase120LeaveOneOut     `json:"leave_one_symbol_out_result"`
	LeaveOneMonthOut          phase120LeaveOneOut     `json:"leave_one_month_out_result"`
	LeaveOneQuarterOut        phase120LeaveOneOut     `json:"leave_one_quarter_out_result"`
	TopSymbolContributionPct  float64                 `json:"top_symbol_contribution_pct"`
	TopMonthContributionPct   float64                 `json:"top_month_contribution_pct"`
	TopQuarterContributionPct float64                 `json:"top_quarter_contribution_pct"`
	ContributingSymbolCount   int                     `json:"contributing_symbol_count"`
	ContributingMonthCount    int                     `json:"contributing_month_count"`
	ContributingQuarterCount  int                     `json:"contributing_quarter_count"`
	SymbolMetrics             []phase120DimensionStat `json:"symbol_metrics,omitempty"`
	MonthMetrics              []phase120DimensionStat `json:"month_metrics,omitempty"`
	QuarterMetrics            []phase120DimensionStat `json:"quarter_metrics,omitempty"`
}

type phase120LeaveOneOut struct {
	Passed                 bool    `json:"passed"`
	WorstExpectancyAfter5  float64 `json:"worst_expectancy_after_5_bps"`
	WorstExcludedDimension string  `json:"worst_excluded_dimension,omitempty"`
}

type phase120DimensionStat struct {
	Key                 string  `json:"key"`
	SampleCount         int     `json:"sample_count"`
	ExpectancyAfter5Bps float64 `json:"expectancy_after_5_bps"`
	NetAfter5Bps        float64 `json:"net_after_5_bps"`
}

type phase120AggKey struct {
	Feature string
	Bucket  string
	Regime  string
	Side    string
	Horizon string
}

type phase120Agg struct {
	key      phase120AggKey
	raw      phase120Stats
	cost5    phase120Stats
	cost75   phase120Stats
	cost10   phase120Stats
	bins     map[int]int
	clusters map[string]*phase120ClusterCounter
	symbols  map[string]*phase120Stats
	months   map[string]*phase120Stats
	quarters map[string]*phase120Stats
}

type phase120Stats struct {
	count  int
	sum    float64
	profit float64
	loss   float64
	wins   int
	losses int
}

type phase120ClusterCounter struct {
	count int
	last  int64
}

var phase120EdgeDiscoveryMatrixCmd = &cobra.Command{
	Use:   "phase12-edge-discovery-matrix",
	Short: "Generate Phase 12.0 feature-bucket edge discovery matrix",
	RunE: func(cmd *cobra.Command, args []string) error {
		cfgSymbols := parseFundingSymbols(p120Symbols)
		if len(cfgSymbols) == 0 {
			cfgSymbols = []string{"ADAUSDT", "AVAXUSDT", "BNBUSDT", "DOGEUSDT", "ETHUSDT", "LINKUSDT", "SOLUSDT", "XRPUSDT"}
		}
		months := parseFundingMonths(p120From, p120To)
		if len(months) == 0 {
			return fmt.Errorf("missing or invalid --from/--to month range")
		}
		mdOut, jsonOut := normalizeMDAndJSONPaths(p120Out)
		csvOut := strings.TrimSuffix(jsonOut, ".json") + "_top_edge_buckets.csv"
		if filepath.Base(jsonOut) == "phase12_0_edge_discovery_matrix.json" {
			csvOut = filepath.Join(filepath.Dir(jsonOut), "phase12_0_top_edge_buckets.csv")
		}
		report, err := runPhase120EdgeDiscoveryMatrix(cmd.Context(), resolveHistorianWorkdir(cmd, p120Workdir), p120Market, p120Interval, cfgSymbols, months)
		if err != nil {
			return err
		}
		if err := writeJSONFile(jsonOut, report); err != nil {
			return err
		}
		if err := os.WriteFile(mdOut, []byte(renderPhase120Markdown(report)), 0644); err != nil {
			return fmt.Errorf("write markdown: %w", err)
		}
		if err := writePhase120CSV(csvOut, report.Buckets); err != nil {
			return err
		}
		fmt.Printf("Phase 12.0 edge discovery reports written to %s, %s, %s\n", jsonOut, mdOut, csvOut)
		return nil
	},
}

func init() {
	phase120EdgeDiscoveryMatrixCmd.Flags().StringVar(&p120Workdir, "workdir", defaultHistorianWorkdir, "local historian workdir")
	phase120EdgeDiscoveryMatrixCmd.Flags().StringVar(&p120Symbols, "symbols", "ADAUSDT,AVAXUSDT,BNBUSDT,DOGEUSDT,ETHUSDT,LINKUSDT,SOLUSDT,XRPUSDT", "comma-separated target symbols")
	phase120EdgeDiscoveryMatrixCmd.Flags().StringVar(&p120Market, "market", "futures-um", "market")
	phase120EdgeDiscoveryMatrixCmd.Flags().StringVar(&p120Interval, "interval", "1m", "candle interval")
	phase120EdgeDiscoveryMatrixCmd.Flags().StringVar(&p120From, "from", "2024-01", "from month YYYY-MM")
	phase120EdgeDiscoveryMatrixCmd.Flags().StringVar(&p120To, "to", "2025-12", "to month YYYY-MM")
	phase120EdgeDiscoveryMatrixCmd.Flags().StringVar(&p120Out, "out", filepath.Join("runs", "reports", "phase12_0_edge_discovery_matrix.md"), "output markdown path")
	rootCmd.AddCommand(phase120EdgeDiscoveryMatrixCmd)
}

func runPhase120EdgeDiscoveryMatrix(ctx context.Context, workdir, market, interval string, symbols, months []string) (phase120Report, error) {
	aggs := make(map[phase120AggKey]*phase120Agg)
	report := phase120Report{
		Phase:                  "Phase 12.0",
		Mode:                   "measurement/reporting only",
		MeasurementCadence:     "15m fixed grid over existing 1m candle/features rows",
		RequiredSymbolMonths:   len(symbols) * len(months),
		AKTraderTouched:        false,
		PromotionAllowed:       false,
		RawRequired:            false,
		RawEventDetailRetained: false,
		Boundaries: []string{
			"no ak-trader changes",
			"no promotion",
			"no live trading or execution logic",
			"no new strategy family",
			"no threshold tuning after results",
			"no new data fetch",
			"funding not used as primary trigger",
			"retained summaries only",
		},
	}
	report.FeatureInventory = phase120FeatureInventory()
	for _, info := range report.FeatureInventory {
		if info.Available {
			report.FeaturesEvaluated = append(report.FeaturesEvaluated, info.Feature)
		}
	}
	contextCache := make(map[string][]protocol.Candle)
	for _, symbol := range symbols {
		fmt.Printf("phase12.0 processing %s\n", symbol)
		rows, err := buildPhase120Inputs(ctx, workdir, market, interval, symbol, months, contextCache)
		if err != nil {
			for _, month := range months {
				report.MissingSymbolMonths = append(report.MissingSymbolMonths, symbol+"|"+month)
			}
			continue
		}
		fmt.Printf("phase12.0 %s feature rows=%d\n", symbol, len(rows))
		covered := phase120CoveredMonths(rows, months)
		for _, month := range months {
			if covered[month] {
				report.CoverageCount++
			} else {
				report.MissingSymbolMonths = append(report.MissingSymbolMonths, symbol+"|"+month)
			}
		}
		phase120AccumulateRows(aggs, rows, months)
		fmt.Printf("phase12.0 %s aggregates=%d\n", symbol, len(aggs))
		rows = nil
		runtime.GC()
	}
	report.Buckets = phase120Summaries(aggs)
	report.BucketCount = len(report.Buckets)
	for i := range report.Buckets {
		switch report.Buckets[i].Label {
		case phase120LabelRobust:
			report.RobustEdgeCandidateCount++
			report.RobustEdgeCandidateBucket = append(report.RobustEdgeCandidateBucket, report.Buckets[i])
		case phase120LabelWeak:
			report.WeakEdgeCount++
			report.WeakEdgeBuckets = append(report.WeakEdgeBuckets, report.Buckets[i])
		case phase120LabelInsufficient:
			report.DataInsufficientCount++
		default:
			report.RejectedCount++
		}
	}
	report.TopExpectancyAfter5Bps = topPhase120Buckets(report.Buckets, func(a, b phase120BucketSummary) bool {
		return a.ExpectancyAfter5Bps > b.ExpectancyAfter5Bps
	}, 20)
	report.TopExpectancyAfter7_5Bps = topPhase120Buckets(report.Buckets, func(a, b phase120BucketSummary) bool {
		return a.ExpectancyAfter7_5Bps > b.ExpectancyAfter7_5Bps
	}, 20)
	report.TopProfitFactor = topPhase120Buckets(report.Buckets, func(a, b phase120BucketSummary) bool {
		return a.ProfitFactor > b.ProfitFactor
	}, 20)
	if strongest, ok := phase120StrongestBucket(report.Buckets); ok {
		report.StrongestBucket = &strongest
	}
	report.FinalLabel, report.RecommendedNextPhase = phase120FinalDecision(report)
	return report, nil
}

func buildPhase120Inputs(ctx context.Context, workdir, market, interval, symbol string, months []string, contextCache map[string][]protocol.Candle) ([]features.Row, error) {
	fromTime, toTime, err := phase120LoadWindow(months)
	if err != nil {
		return nil, err
	}
	src := data.NewLocalParquetSource()
	req := data.CandleRequest{Source: "local-parquet", Path: workdir, Market: market, Symbol: symbol, Interval: interval, From: fromTime, To: toTime}
	fmt.Printf("phase12.0 %s loading target candles\n", symbol)
	candles, err := src.LoadCandles(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("load target candles: %w", err)
	}
	fmt.Printf("phase12.0 %s target candles=%d\n", symbol, len(candles))
	fmt.Printf("phase12.0 %s loading BTC context\n", symbol)
	btc, err := loadPhase120Context(ctx, src, req, "BTCUSDT", symbol, contextCache)
	if err != nil {
		return nil, err
	}
	fmt.Printf("phase12.0 %s BTC context=%d\n", symbol, len(btc))
	fmt.Printf("phase12.0 %s loading ETH context\n", symbol)
	eth, err := loadPhase120Context(ctx, src, req, "ETHUSDT", symbol, contextCache)
	if err != nil {
		return nil, err
	}
	fmt.Printf("phase12.0 %s ETH context=%d\n", symbol, len(eth))
	fmt.Printf("phase12.0 %s building features\n", symbol)
	rows, err := features.BuildRows(candles, features.BuildOptions{Market: market, Symbol: symbol, Interval: interval, ContextBTC: btc, ContextETH: eth})
	if err != nil {
		return nil, err
	}
	if research.CheckFeatureRows(rows).Status != "PASS" {
		return nil, fmt.Errorf("feature leakage check failed")
	}
	return rows, nil
}

func loadPhase120Context(ctx context.Context, src *data.LocalParquetSource, req data.CandleRequest, contextSymbol, targetSymbol string, contextCache map[string][]protocol.Candle) ([]protocol.Candle, error) {
	if contextSymbol == targetSymbol {
		return nil, nil
	}
	cacheKey := contextSymbol
	if cached, ok := contextCache[cacheKey]; ok {
		return cached, nil
	}
	req.Symbol = contextSymbol
	candles, err := src.LoadCandles(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("load context candles %s: %w", contextSymbol, err)
	}
	contextCache[cacheKey] = candles
	return candles, nil
}

func phase120LoadWindow(months []string) (time.Time, time.Time, error) {
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

func phase120AccumulateRows(aggs map[phase120AggKey]*phase120Agg, rows []features.Row, months []string) {
	monthSet := make(map[string]struct{}, len(months))
	for _, month := range months {
		monthSet[month] = struct{}{}
	}
	for i, row := range rows {
		if row.Warmup || row.Close <= 0 || row.AvailableAtMS < row.EventTimeMS {
			continue
		}
		month := monthFromEventTime(row.EventTimeMS)
		if _, ok := monthSet[month]; !ok {
			continue
		}
		if row.EventTimeMS%phase120SampleEveryMS != 0 {
			continue
		}
		forward := phase120ForwardReturns(rows, i)
		if len(forward) != len(phase120HorizonSet) {
			continue
		}
		regime := phase120Regime(row)
		buckets := phase120Buckets(row, regime)
		for _, fb := range buckets {
			for _, side := range []string{"long", "short"} {
				for _, horizon := range phase120HorizonSet {
					ret := forward[horizon]
					if side == "short" {
						ret = -ret
					}
					key := phase120AggKey{Feature: fb.Feature, Bucket: fb.Bucket, Regime: regime, Side: side, Horizon: horizon}
					phase120AggFor(aggs, key).add(ret, row.EventTimeMS, row.Symbol, month, quarterFromMonth(month))
				}
			}
		}
	}
}

type phase120FeatureBucket struct {
	Feature string
	Bucket  string
}

func phase120Buckets(row features.Row, regime string) []phase120FeatureBucket {
	closeVsEMA := "ema20_unknown"
	if row.EMA20 > 0 {
		closeVsEMA = phase120SignedBucket((row.Close-row.EMA20)/row.EMA20, 0.001, 0.004, "below_ema20", "near_ema20", "above_ema20")
	}
	return []phase120FeatureBucket{
		{"Return15", phase120SignedBucket(row.Return15, 0.001, 0.005, "return15_down", "return15_flat", "return15_up")},
		{"TrendSlope20", phase120SignedBucket(row.TrendSlope20, 0.0002, 0.002, "slope_down", "slope_flat", "slope_up")},
		{"CloseRelativeEMA20", closeVsEMA},
		{"BBWidthPctRank60", phase120RankBucket(row.BBWidthPctRank60, "bb_width")},
		{"VolumeRatio20", phase120RatioBucket(row.VolumeRatio20, "volume_ratio")},
		{"TakerBuyRatio", phase120SignedBucket(row.TakerBuyRatio-0.5, 0.03, 0.12, "taker_sell_heavy", "taker_balanced", "taker_buy_heavy")},
		{"CompositeRegime", regime},
		{"BTCContext60", phase120SignedBucket(row.BTCReturn60, 0.0015, 0.006, "btc_down", "btc_flat", "btc_up")},
		{"ETHContext60", phase120SignedBucket(row.ETHReturn60, 0.0015, 0.006, "eth_down", "eth_flat", "eth_up")},
		{"SessionUTC", phase120SessionBucket(row.EventTimeMS)},
		{"ATRPct14", phase120VolBucket(row.ATRPct14, "atr_pct_14")},
		{"RealizedVol60", phase120VolBucket(row.RealizedVol60, "realized_vol_60")},
	}
}

func phase120AggFor(aggs map[phase120AggKey]*phase120Agg, key phase120AggKey) *phase120Agg {
	if agg := aggs[key]; agg != nil {
		return agg
	}
	agg := &phase120Agg{
		key:      key,
		bins:     make(map[int]int),
		clusters: make(map[string]*phase120ClusterCounter),
		symbols:  make(map[string]*phase120Stats),
		months:   make(map[string]*phase120Stats),
		quarters: make(map[string]*phase120Stats),
	}
	aggs[key] = agg
	return agg
}

func (a *phase120Agg) add(rawRet float64, eventTimeMS int64, symbol, month, quarter string) {
	a.raw.add(rawRet)
	a.cost5.add(rawRet - 5)
	a.cost75.add(rawRet - 7.5)
	a.cost10.add(rawRet - 10)
	a.bins[int(math.Round(rawRet*10))]++
	a.addCluster(symbol, eventTimeMS)
	phase120StatsFor(a.symbols, symbol).add(rawRet - 5)
	phase120StatsFor(a.months, month).add(rawRet - 5)
	phase120StatsFor(a.quarters, quarter).add(rawRet - 5)
}

func (a *phase120Agg) addCluster(symbol string, eventTimeMS int64) {
	c := a.clusters[symbol]
	if c == nil {
		c = &phase120ClusterCounter{last: math.MinInt64}
		a.clusters[symbol] = c
	}
	if c.count == 0 || eventTimeMS-c.last >= fundingClusterWindowMS {
		c.count++
		c.last = eventTimeMS
	}
}

func (s *phase120Stats) add(v float64) {
	s.count++
	s.sum += v
	if v > 0 {
		s.profit += v
		s.wins++
	} else if v < 0 {
		s.loss += -v
		s.losses++
	}
}

func phase120StatsFor(m map[string]*phase120Stats, key string) *phase120Stats {
	if s := m[key]; s != nil {
		return s
	}
	s := &phase120Stats{}
	m[key] = s
	return s
}

func phase120Summaries(aggs map[phase120AggKey]*phase120Agg) []phase120BucketSummary {
	out := make([]phase120BucketSummary, 0, len(aggs))
	for _, agg := range aggs {
		row := phase120BucketSummary{
			Feature:                   agg.key.Feature,
			Bucket:                    agg.key.Bucket,
			Regime:                    agg.key.Regime,
			Side:                      agg.key.Side,
			Horizon:                   agg.key.Horizon,
			SampleCount:               agg.raw.count,
			ClusterCount:              phase120ClusterCount(agg.clusters),
			RawExpectancyBps:          phase120Expectancy(agg.raw),
			ExpectancyAfter5Bps:       phase120Expectancy(agg.cost5),
			ExpectancyAfter7_5Bps:     phase120Expectancy(agg.cost75),
			ExpectancyAfter10Bps:      phase120Expectancy(agg.cost10),
			PFAfter5Bps:               phase120ProfitFactor(agg.cost5),
			PFAfter7_5Bps:             phase120ProfitFactor(agg.cost75),
			PFAfter10Bps:              phase120ProfitFactor(agg.cost10),
			WinRateAfter5Bps:          phase120WinRate(agg.cost5),
			WinRateAfter7_5Bps:        phase120WinRate(agg.cost75),
			WinRateAfter10Bps:         phase120WinRate(agg.cost10),
			GrossProfitAfter5Bps:      agg.cost5.profit,
			GrossLossAfter5Bps:        agg.cost5.loss,
			GrossProfitAfter7_5Bps:    agg.cost75.profit,
			GrossLossAfter7_5Bps:      agg.cost75.loss,
			GrossProfitAfter10Bps:     agg.cost10.profit,
			GrossLossAfter10Bps:       agg.cost10.loss,
			WinCountAfter5Bps:         agg.cost5.wins,
			LossCountAfter5Bps:        agg.cost5.losses,
			WinCountAfter7_5Bps:       agg.cost75.wins,
			LossCountAfter7_5Bps:      agg.cost75.losses,
			WinCountAfter10Bps:        agg.cost10.wins,
			LossCountAfter10Bps:       agg.cost10.losses,
			WinRate:                   phase120WinRate(agg.cost5),
			ProfitFactor:              phase120ProfitFactor(agg.cost5),
			AverageForwardReturn:      phase120Expectancy(agg.raw),
			MedianForwardReturn:       phase120MedianFromBins(agg.bins),
			WorstMonthExpectancy:      phase120WorstExpectancy(agg.months),
			WorstSymbolExpectancy:     phase120WorstExpectancy(agg.symbols),
			LeaveOneSymbolOut:         phase120LeaveOne(agg.cost5, agg.symbols),
			LeaveOneMonthOut:          phase120LeaveOne(agg.cost5, agg.months),
			LeaveOneQuarterOut:        phase120LeaveOne(agg.cost5, agg.quarters),
			TopSymbolContributionPct:  phase120TopContributionPct(agg.symbols),
			TopMonthContributionPct:   phase120TopContributionPct(agg.months),
			TopQuarterContributionPct: phase120TopContributionPct(agg.quarters),
			ContributingSymbolCount:   phase120MeaningfulCount(agg.symbols, 50),
			ContributingMonthCount:    phase120MeaningfulCount(agg.months, 50),
			ContributingQuarterCount:  phase120MeaningfulCount(agg.quarters, 50),
			SymbolMetrics:             phase120DimensionStats(agg.symbols),
			MonthMetrics:              phase120DimensionStats(agg.months),
			QuarterMetrics:            phase120DimensionStats(agg.quarters),
		}
		row.Label, row.FailedRules = phase120Label(row)
		out = append(out, phase120RoundBucket(row))
	}
	sort.Slice(out, func(i, j int) bool {
		a, b := out[i], out[j]
		if a.Label != b.Label {
			return phase120LabelRank(a.Label) < phase120LabelRank(b.Label)
		}
		if a.ExpectancyAfter5Bps != b.ExpectancyAfter5Bps {
			return a.ExpectancyAfter5Bps > b.ExpectancyAfter5Bps
		}
		return strings.Join([]string{a.Feature, a.Bucket, a.Regime, a.Side, a.Horizon}, "|") < strings.Join([]string{b.Feature, b.Bucket, b.Regime, b.Side, b.Horizon}, "|")
	})
	return out
}

func phase120Label(row phase120BucketSummary) (string, []string) {
	var failed []string
	if row.SampleCount < 500 || row.ClusterCount < 100 {
		return phase120LabelInsufficient, []string{"sample_count_or_cluster_count_too_low"}
	}
	if row.ExpectancyAfter5Bps <= 0 {
		failed = append(failed, "expectancy_after_5_bps_non_positive")
	}
	if row.ProfitFactor <= 1.0 {
		failed = append(failed, "profit_factor_lte_1")
	}
	if row.TopSymbolContributionPct > 45 || row.TopMonthContributionPct > 35 {
		failed = append(failed, "edge_concentrated")
	}
	if len(failed) > 0 {
		return phase120LabelRejected, failed
	}
	robustFails := []string{}
	if row.ExpectancyAfter7_5Bps <= 0 {
		robustFails = append(robustFails, "expectancy_after_7_5_bps_non_positive")
	}
	if row.ExpectancyAfter10Bps <= 0 {
		robustFails = append(robustFails, "expectancy_after_10_bps_non_positive")
	}
	if row.ProfitFactor <= 1.05 {
		robustFails = append(robustFails, "profit_factor_lte_1_05")
	}
	if !row.LeaveOneSymbolOut.Passed || !row.LeaveOneMonthOut.Passed || !row.LeaveOneQuarterOut.Passed {
		robustFails = append(robustFails, "leave_one_out_destroyed_edge")
	}
	if row.ContributingSymbolCount < 5 {
		robustFails = append(robustFails, "fewer_than_5_symbols_contribute")
	}
	if row.ContributingMonthCount < 12 {
		robustFails = append(robustFails, "fewer_than_12_months_contribute")
	}
	if len(robustFails) > 0 {
		return phase120LabelWeak, robustFails
	}
	return phase120LabelRobust, nil
}

func phase120ForwardReturns(rows []features.Row, idx int) map[string]float64 {
	out := make(map[string]float64, len(phase120HorizonSet))
	for _, horizon := range phase120HorizonSet {
		future, ok := phase120FutureClose(rows, idx, horizon)
		if !ok || future <= 0 {
			return nil
		}
		out[horizon] = (future - rows[idx].Close) / rows[idx].Close * 10000
	}
	return out
}

func phase120FutureClose(rows []features.Row, startIdx int, horizon string) (float64, bool) {
	offsetMS := fundingHorizonMS[horizon]
	j := startIdx + int(offsetMS/60000)
	if j >= len(rows) {
		return 0, false
	}
	target := rows[startIdx].EventTimeMS + offsetMS
	if rows[j].EventTimeMS != target {
		return 0, false
	}
	return rows[j].Close, true
}

func phase120CoveredMonths(rows []features.Row, months []string) map[string]bool {
	want := make(map[string]struct{}, len(months))
	for _, month := range months {
		want[month] = struct{}{}
	}
	out := make(map[string]bool, len(months))
	for _, row := range rows {
		month := monthFromEventTime(row.EventTimeMS)
		if _, ok := want[month]; ok && !row.Warmup {
			out[month] = true
		}
	}
	return out
}

func phase120FeatureInventory() []phase120FeatureInfo {
	return []phase120FeatureInfo{
		{"Return15", true, []string{"return15_down", "return15_flat", "return15_up"}, "features.Row.Return15"},
		{"TrendSlope20", true, []string{"slope_down", "slope_flat", "slope_up"}, "features.Row.TrendSlope20"},
		{"CloseRelativeEMA20", true, []string{"below_ema20", "near_ema20", "above_ema20"}, "features.Row.Close and EMA20"},
		{"BBWidthPctRank60", true, []string{"bb_width_low", "bb_width_mid", "bb_width_high"}, "features.Row.BBWidthPctRank60"},
		{"VolumeRatio20", true, []string{"volume_ratio_low", "volume_ratio_normal", "volume_ratio_high"}, "features.Row.VolumeRatio20"},
		{"TakerBuyRatio", true, []string{"taker_sell_heavy", "taker_balanced", "taker_buy_heavy"}, "features.Row.TakerBuyRatio"},
		{"CompositeRegime", true, []string{"compressed_down", "compressed_up", "trend_down", "trend_up", "range"}, "derived from existing features"},
		{"BTCContext60", true, []string{"btc_down", "btc_flat", "btc_up"}, "features.Row.BTCReturn60"},
		{"ETHContext60", true, []string{"eth_down", "eth_flat", "eth_up"}, "features.Row.ETHReturn60"},
		{"SessionUTC", true, []string{"asia", "europe", "us", "late_us"}, "derived from event_time_ms"},
		{"ATRPct14", true, []string{"atr_pct_14_low", "atr_pct_14_mid", "atr_pct_14_high"}, "features.Row.ATRPct14"},
		{"RealizedVol60", true, []string{"realized_vol_60_low", "realized_vol_60_mid", "realized_vol_60_high"}, "features.Row.RealizedVol60"},
	}
}

func phase120Regime(row features.Row) string {
	if row.BBWidthPctRank60 <= 0.25 && row.BTCReturn60 < -0.002 {
		return "compressed_down"
	}
	if row.BBWidthPctRank60 <= 0.25 && row.BTCReturn60 > 0.002 {
		return "compressed_up"
	}
	if row.Close > row.EMA50 && row.EMA50 > row.EMA200 && row.TrendSlope20 > 0 {
		return "trend_up"
	}
	if row.Close < row.EMA50 && row.EMA50 < row.EMA200 && row.TrendSlope20 < 0 {
		return "trend_down"
	}
	return "range"
}

func phase120SignedBucket(v, neutral, strong float64, down, flat, up string) string {
	if v <= -strong {
		return down + "_strong"
	}
	if v < -neutral {
		return down
	}
	if v >= strong {
		return up + "_strong"
	}
	if v > neutral {
		return up
	}
	return flat
}

func phase120RankBucket(v float64, prefix string) string {
	if v <= 0.25 {
		return prefix + "_low"
	}
	if v >= 0.75 {
		return prefix + "_high"
	}
	return prefix + "_mid"
}

func phase120RatioBucket(v float64, prefix string) string {
	if v <= 0 || math.IsNaN(v) || math.IsInf(v, 0) {
		return prefix + "_unknown"
	}
	if v < 0.8 {
		return prefix + "_low"
	}
	if v > 1.5 {
		return prefix + "_high"
	}
	return prefix + "_normal"
}

func phase120VolBucket(v float64, prefix string) string {
	if v <= 0 || math.IsNaN(v) || math.IsInf(v, 0) {
		return prefix + "_unknown"
	}
	if v < 0.0015 {
		return prefix + "_low"
	}
	if v > 0.006 {
		return prefix + "_high"
	}
	return prefix + "_mid"
}

func phase120SessionBucket(eventTimeMS int64) string {
	hour := time.UnixMilli(eventTimeMS).UTC().Hour()
	switch {
	case hour < 7:
		return "asia"
	case hour < 13:
		return "europe"
	case hour < 20:
		return "us"
	default:
		return "late_us"
	}
}

func phase120Expectancy(s phase120Stats) float64 {
	if s.count == 0 {
		return 0
	}
	return s.sum / float64(s.count)
}

func phase120WinRate(s phase120Stats) float64 {
	if s.count == 0 {
		return 0
	}
	return float64(s.wins) / float64(s.count) * 100
}

func phase120ProfitFactor(s phase120Stats) float64 {
	if s.loss > 0 {
		return s.profit / s.loss
	}
	if s.profit > 0 {
		return 999
	}
	return 0
}

func phase120ClusterCount(clusters map[string]*phase120ClusterCounter) int {
	count := 0
	for _, c := range clusters {
		count += c.count
	}
	return count
}

func phase120MedianFromBins(bins map[int]int) float64 {
	total := 0
	for _, n := range bins {
		total += n
	}
	if total == 0 {
		return 0
	}
	keys := make([]int, 0, len(bins))
	for k := range bins {
		keys = append(keys, k)
	}
	sort.Ints(keys)
	target1 := (total - 1) / 2
	target2 := total / 2
	seen := 0
	var v1, v2 float64
	for _, k := range keys {
		next := seen + bins[k]
		if seen <= target1 && target1 < next {
			v1 = float64(k) / 10
		}
		if seen <= target2 && target2 < next {
			v2 = float64(k) / 10
			break
		}
		seen = next
	}
	return (v1 + v2) / 2
}

func phase120WorstExpectancy(m map[string]*phase120Stats) float64 {
	worst := math.MaxFloat64
	found := false
	for _, s := range m {
		if s.count == 0 {
			continue
		}
		v := phase120Expectancy(*s)
		if v < worst {
			worst = v
			found = true
		}
	}
	if !found {
		return 0
	}
	return worst
}

func phase120LeaveOne(total phase120Stats, dims map[string]*phase120Stats) phase120LeaveOneOut {
	out := phase120LeaveOneOut{Passed: total.count > 0, WorstExpectancyAfter5: phase120Expectancy(total)}
	for key, s := range dims {
		remaining := total
		remaining.count -= s.count
		remaining.sum -= s.sum
		remaining.profit -= s.profit
		remaining.loss -= s.loss
		remaining.wins -= s.wins
		remaining.losses -= s.losses
		if remaining.count <= 0 {
			out.Passed = false
			out.WorstExpectancyAfter5 = 0
			out.WorstExcludedDimension = key
			return out
		}
		exp := phase120Expectancy(remaining)
		if exp < out.WorstExpectancyAfter5 {
			out.WorstExpectancyAfter5 = exp
			out.WorstExcludedDimension = key
		}
		if exp <= 0 {
			out.Passed = false
		}
	}
	return out
}

func phase120TopContributionPct(m map[string]*phase120Stats) float64 {
	total := 0.0
	top := 0.0
	for _, s := range m {
		if s.sum <= 0 {
			continue
		}
		total += s.sum
		if s.sum > top {
			top = s.sum
		}
	}
	if total <= 0 {
		return 0
	}
	return top / total * 100
}

func phase120MeaningfulCount(m map[string]*phase120Stats, minSamples int) int {
	count := 0
	for _, s := range m {
		if s.count >= minSamples {
			count++
		}
	}
	return count
}

func phase120DimensionStats(m map[string]*phase120Stats) []phase120DimensionStat {
	out := make([]phase120DimensionStat, 0, len(m))
	for key, s := range m {
		out = append(out, phase120DimensionStat{
			Key:                 key,
			SampleCount:         s.count,
			ExpectancyAfter5Bps: roundMetric(phase120Expectancy(*s)),
			NetAfter5Bps:        roundMetric(s.sum),
		})
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Key < out[j].Key })
	return out
}

func phase120RoundBucket(row phase120BucketSummary) phase120BucketSummary {
	row.RawExpectancyBps = roundMetric(row.RawExpectancyBps)
	row.ExpectancyAfter5Bps = roundMetric(row.ExpectancyAfter5Bps)
	row.ExpectancyAfter7_5Bps = roundMetric(row.ExpectancyAfter7_5Bps)
	row.ExpectancyAfter10Bps = roundMetric(row.ExpectancyAfter10Bps)
	row.PFAfter5Bps = roundMetric(row.PFAfter5Bps)
	row.PFAfter7_5Bps = roundMetric(row.PFAfter7_5Bps)
	row.PFAfter10Bps = roundMetric(row.PFAfter10Bps)
	row.WinRateAfter5Bps = roundMetric(row.WinRateAfter5Bps)
	row.WinRateAfter7_5Bps = roundMetric(row.WinRateAfter7_5Bps)
	row.WinRateAfter10Bps = roundMetric(row.WinRateAfter10Bps)
	row.GrossProfitAfter5Bps = roundMetric(row.GrossProfitAfter5Bps)
	row.GrossLossAfter5Bps = roundMetric(row.GrossLossAfter5Bps)
	row.GrossProfitAfter7_5Bps = roundMetric(row.GrossProfitAfter7_5Bps)
	row.GrossLossAfter7_5Bps = roundMetric(row.GrossLossAfter7_5Bps)
	row.GrossProfitAfter10Bps = roundMetric(row.GrossProfitAfter10Bps)
	row.GrossLossAfter10Bps = roundMetric(row.GrossLossAfter10Bps)
	row.WinRate = roundMetric(row.WinRate)
	row.ProfitFactor = roundMetric(row.ProfitFactor)
	row.AverageForwardReturn = roundMetric(row.AverageForwardReturn)
	row.MedianForwardReturn = roundMetric(row.MedianForwardReturn)
	row.WorstMonthExpectancy = roundMetric(row.WorstMonthExpectancy)
	row.WorstSymbolExpectancy = roundMetric(row.WorstSymbolExpectancy)
	row.LeaveOneSymbolOut.WorstExpectancyAfter5 = roundMetric(row.LeaveOneSymbolOut.WorstExpectancyAfter5)
	row.LeaveOneMonthOut.WorstExpectancyAfter5 = roundMetric(row.LeaveOneMonthOut.WorstExpectancyAfter5)
	row.LeaveOneQuarterOut.WorstExpectancyAfter5 = roundMetric(row.LeaveOneQuarterOut.WorstExpectancyAfter5)
	row.TopSymbolContributionPct = roundMetric(row.TopSymbolContributionPct)
	row.TopMonthContributionPct = roundMetric(row.TopMonthContributionPct)
	row.TopQuarterContributionPct = roundMetric(row.TopQuarterContributionPct)
	return row
}

func topPhase120Buckets(rows []phase120BucketSummary, less func(a, b phase120BucketSummary) bool, n int) []phase120BucketSummary {
	cp := append([]phase120BucketSummary(nil), rows...)
	sort.Slice(cp, func(i, j int) bool {
		if less(cp[i], cp[j]) {
			return true
		}
		if less(cp[j], cp[i]) {
			return false
		}
		return cp[i].SampleCount > cp[j].SampleCount
	})
	if len(cp) > n {
		cp = cp[:n]
	}
	return cp
}

func phase120StrongestBucket(rows []phase120BucketSummary) (phase120BucketSummary, bool) {
	for _, label := range []string{phase120LabelRobust, phase120LabelWeak, phase120LabelRejected, phase120LabelInsufficient} {
		var filtered []phase120BucketSummary
		for _, row := range rows {
			if row.Label == label {
				filtered = append(filtered, row)
			}
		}
		if len(filtered) == 0 {
			continue
		}
		return topPhase120Buckets(filtered, func(a, b phase120BucketSummary) bool {
			return a.ExpectancyAfter5Bps > b.ExpectancyAfter5Bps
		}, 1)[0], true
	}
	return phase120BucketSummary{}, false
}

func phase120FinalDecision(report phase120Report) (string, string) {
	if report.CoverageCount < report.RequiredSymbolMonths {
		return "blocked_missing_data", "Phase 12.1 - restore missing local candle/features coverage for " + strings.Join(report.MissingSymbolMonths, ",")
	}
	if report.RobustEdgeCandidateCount > 0 {
		return "robust_edges_found", "Phase 12.1 - Convert Top Robust Edge Bucket Into Candidate Family"
	}
	if report.WeakEdgeCount > 0 {
		return "weak_edges_only", "Phase 12.1 - Narrow Edge Discovery / Add Robustness Filters"
	}
	return "no_edges_found", "Phase 12.1 - Stop Current Feature Set / Data Source Upgrade Plan"
}

func phase120LabelRank(label string) int {
	switch label {
	case phase120LabelRobust:
		return 0
	case phase120LabelWeak:
		return 1
	case phase120LabelRejected:
		return 2
	default:
		return 3
	}
}

func renderPhase120Markdown(report phase120Report) string {
	var sb strings.Builder
	sb.WriteString("# Phase 12.0 - Edge Discovery Matrix\n\n")
	sb.WriteString("## Summary\n")
	sb.WriteString(fmt.Sprintf("- final_label: `%s`\n", report.FinalLabel))
	sb.WriteString(fmt.Sprintf("- measurement cadence: `%s`\n", report.MeasurementCadence))
	sb.WriteString(fmt.Sprintf("- coverage achieved: `%d/%d`\n", report.CoverageCount, report.RequiredSymbolMonths))
	sb.WriteString(fmt.Sprintf("- ak_trader_touched: `%t`\n", report.AKTraderTouched))
	sb.WriteString(fmt.Sprintf("- promotion_allowed: `%t`\n", report.PromotionAllowed))
	sb.WriteString(fmt.Sprintf("- raw_required: `%t`\n", report.RawRequired))
	sb.WriteString(fmt.Sprintf("- raw_event_detail_retained: `%t`\n", report.RawEventDetailRetained))
	sb.WriteString(fmt.Sprintf("- bucket_count: `%d`\n", report.BucketCount))
	sb.WriteString(fmt.Sprintf("- rejected bucket count: `%d`\n", report.RejectedCount))
	sb.WriteString(fmt.Sprintf("- data-insufficient bucket count: `%d`\n", report.DataInsufficientCount))
	sb.WriteString(fmt.Sprintf("- weak bucket count: `%d`\n", report.WeakEdgeCount))
	sb.WriteString(fmt.Sprintf("- robust bucket count: `%d`\n", report.RobustEdgeCandidateCount))
	if report.StrongestBucket != nil {
		sb.WriteString(fmt.Sprintf("- strongest bucket: `%s/%s/%s/%s/%s` exp5=`%.6f` PF=`%.6f` label=`%s`\n",
			report.StrongestBucket.Feature, report.StrongestBucket.Bucket, report.StrongestBucket.Regime, report.StrongestBucket.Side, report.StrongestBucket.Horizon,
			report.StrongestBucket.ExpectancyAfter5Bps, report.StrongestBucket.ProfitFactor, report.StrongestBucket.Label))
	}
	sb.WriteString(fmt.Sprintf("- worth turning into a strategy: `%t`\n", report.RobustEdgeCandidateCount > 0))
	sb.WriteString(fmt.Sprintf("- recommended next phase: `%s`\n", report.RecommendedNextPhase))
	sb.WriteString("\n## Feature Inventory\n")
	for _, info := range report.FeatureInventory {
		sb.WriteString(fmt.Sprintf("- %s: available=`%t`, source=`%s`, buckets=`%s`\n", info.Feature, info.Available, info.Source, strings.Join(info.Buckets, ", ")))
	}
	sb.WriteString("\n## Top 20 By Expectancy After 5 bps\n")
	phase120WriteBucketTable(&sb, report.TopExpectancyAfter5Bps)
	sb.WriteString("\n## Top 20 By Expectancy After 7.5 bps\n")
	phase120WriteBucketTable(&sb, report.TopExpectancyAfter7_5Bps)
	sb.WriteString("\n## Top 20 By Profit Factor\n")
	phase120WriteBucketTable(&sb, report.TopProfitFactor)
	sb.WriteString("\n## Robust Edge Candidate Buckets\n")
	phase120WriteBucketTable(&sb, report.RobustEdgeCandidateBucket)
	sb.WriteString("\n## Weak Edge Buckets\n")
	phase120WriteBucketTable(&sb, report.WeakEdgeBuckets)
	return sb.String()
}

func phase120WriteBucketTable(sb *strings.Builder, rows []phase120BucketSummary) {
	sb.WriteString("| Feature | Bucket | Regime | Side | Horizon | Label | Samples | Clusters | Exp5 | Exp7.5 | Exp10 | PF | Win% | Worst Month | Worst Symbol | Top Symbol% | Top Month% |\n")
	sb.WriteString("|---|---|---|---|---|---|---:|---:|---:|---:|---:|---:|---:|---:|---:|---:|---:|\n")
	for _, row := range rows {
		sb.WriteString(fmt.Sprintf("| %s | %s | %s | %s | %s | %s | %d | %d | %.6f | %.6f | %.6f | %.6f | %.4f | %.6f | %.6f | %.4f | %.4f |\n",
			row.Feature, row.Bucket, row.Regime, row.Side, row.Horizon, row.Label, row.SampleCount, row.ClusterCount,
			row.ExpectancyAfter5Bps, row.ExpectancyAfter7_5Bps, row.ExpectancyAfter10Bps, row.ProfitFactor, row.WinRate,
			row.WorstMonthExpectancy, row.WorstSymbolExpectancy, row.TopSymbolContributionPct, row.TopMonthContributionPct))
	}
}

func writePhase120CSV(path string, rows []phase120BucketSummary) error {
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return fmt.Errorf("create csv output dir: %w", err)
	}
	f, err := os.Create(path)
	if err != nil {
		return fmt.Errorf("create csv: %w", err)
	}
	defer f.Close()
	w := csv.NewWriter(f)
	defer w.Flush()
	header := []string{
		"feature", "bucket", "regime", "side", "horizon", "label", "sample_count", "cluster_count",
		"raw_expectancy_bps", "expectancy_after_5_bps", "expectancy_after_7_5_bps", "expectancy_after_10_bps",
		"pf_after_5_bps", "pf_after_7_5_bps", "pf_after_10_bps",
		"win_rate_after_5_bps", "win_rate_after_7_5_bps", "win_rate_after_10_bps",
		"gross_profit_after_5_bps", "gross_loss_after_5_bps",
		"gross_profit_after_7_5_bps", "gross_loss_after_7_5_bps",
		"gross_profit_after_10_bps", "gross_loss_after_10_bps",
		"win_count_after_5_bps", "loss_count_after_5_bps",
		"win_count_after_7_5_bps", "loss_count_after_7_5_bps",
		"win_count_after_10_bps", "loss_count_after_10_bps",
		"win_rate", "profit_factor", "average_forward_return", "median_forward_return",
		"worst_month_expectancy", "worst_symbol_expectancy",
		"leave_one_symbol_out_passed", "leave_one_month_out_passed", "leave_one_quarter_out_passed",
		"top_symbol_contribution_pct", "top_month_contribution_pct", "top_quarter_contribution_pct",
	}
	if err := w.Write(header); err != nil {
		return err
	}
	for _, row := range rows {
		rec := []string{
			row.Feature, row.Bucket, row.Regime, row.Side, row.Horizon, row.Label,
			strconv.Itoa(row.SampleCount), strconv.Itoa(row.ClusterCount),
			phase120Fmt(row.RawExpectancyBps), phase120Fmt(row.ExpectancyAfter5Bps), phase120Fmt(row.ExpectancyAfter7_5Bps), phase120Fmt(row.ExpectancyAfter10Bps),
			phase120Fmt(row.PFAfter5Bps), phase120Fmt(row.PFAfter7_5Bps), phase120Fmt(row.PFAfter10Bps),
			phase120Fmt(row.WinRateAfter5Bps), phase120Fmt(row.WinRateAfter7_5Bps), phase120Fmt(row.WinRateAfter10Bps),
			phase120Fmt(row.GrossProfitAfter5Bps), phase120Fmt(row.GrossLossAfter5Bps),
			phase120Fmt(row.GrossProfitAfter7_5Bps), phase120Fmt(row.GrossLossAfter7_5Bps),
			phase120Fmt(row.GrossProfitAfter10Bps), phase120Fmt(row.GrossLossAfter10Bps),
			strconv.Itoa(row.WinCountAfter5Bps), strconv.Itoa(row.LossCountAfter5Bps),
			strconv.Itoa(row.WinCountAfter7_5Bps), strconv.Itoa(row.LossCountAfter7_5Bps),
			strconv.Itoa(row.WinCountAfter10Bps), strconv.Itoa(row.LossCountAfter10Bps),
			phase120Fmt(row.WinRate), phase120Fmt(row.ProfitFactor), phase120Fmt(row.AverageForwardReturn), phase120Fmt(row.MedianForwardReturn),
			phase120Fmt(row.WorstMonthExpectancy), phase120Fmt(row.WorstSymbolExpectancy),
			strconv.FormatBool(row.LeaveOneSymbolOut.Passed), strconv.FormatBool(row.LeaveOneMonthOut.Passed), strconv.FormatBool(row.LeaveOneQuarterOut.Passed),
			phase120Fmt(row.TopSymbolContributionPct), phase120Fmt(row.TopMonthContributionPct), phase120Fmt(row.TopQuarterContributionPct),
		}
		if err := w.Write(rec); err != nil {
			return err
		}
	}
	return w.Error()
}

func phase120Fmt(v float64) string {
	return strconv.FormatFloat(v, 'f', 6, 64)
}
