package app

import (
	"bufio"
	"compress/gzip"
	"encoding/json"
	"fmt"
	"io"
	"math"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"

	"github.com/david22573/ak-engine/internal/regime"
)

const fundingClusterWindowMS int64 = 60 * 60 * 1000
const fundingRateIntervalMS int64 = 8 * 60 * 60 * 1000

var defaultFundingSymbols = []string{"LINKUSDT", "SOLUSDT", "AVAXUSDT", "DOGEUSDT", "ADAUSDT", "BNBUSDT", "XRPUSDT", "ETHUSDT"}
var defaultFundingFamilies = []string{"NegativeFundingLong", "PositiveFundingShort", "FundingFlipLong", "FundingFlipShort", "RegimeFundingLong", "RegimeFundingShort", "ConfirmedNegativeFundingLong", "ConfirmedPositiveFundingShort", "BreakoutFundingLong", "BreakoutFundingShort", "VolumeImbalanceFundingReversionProxyLong", "VolumeImbalanceFundingReversionProxyShort"}
var defaultFundingHorizons = []string{"5m", "15m", "30m", "60m", "120m", "240m"}

var fundingHorizonMS = map[string]int64{
	"5m":   5 * 60 * 1000,
	"15m":  15 * 60 * 1000,
	"30m":  30 * 60 * 1000,
	"60m":  60 * 60 * 1000,
	"120m": 120 * 60 * 1000,
	"240m": 240 * 60 * 1000,
}

type FundingEventRow struct {
	Symbol             string   `json:"symbol"`
	Family             string   `json:"family"`
	Side               string   `json:"side"`
	EventTimeMS        int64    `json:"event_time_ms"`
	AvailableAtMS      int64    `json:"available_at_ms"`
	EntryPrice         float64  `json:"entry_price"`
	FundingRate        float64  `json:"funding_rate"`
	FundingRateZScore  float64  `json:"funding_rate_zscore"`
	FundingBucket      string   `json:"funding_bucket"`
	FundingRateUnknown bool     `json:"funding_rate_unknown"`
	RegimeComposite    string   `json:"regime_composite"`
	Volatility         string   `json:"volatility"`
	Trend              string   `json:"trend"`
	Liquidity          string   `json:"liquidity"`
	MarketBeta         string   `json:"market_beta"`
	Return5mBps        float64  `json:"return_5m_bps"`
	Return15mBps       float64  `json:"return_15m_bps"`
	Return30mBps       float64  `json:"return_30m_bps"`
	Return60mBps       float64  `json:"return_60m_bps"`
	Return120mBps      float64  `json:"return_120m_bps"`
	Return240mBps      float64  `json:"return_240m_bps"`
	Return5m5bpsBps    float64  `json:"return_5m_5bps_bps"`
	Return15m5bpsBps   float64  `json:"return_15m_5bps_bps"`
	Return30m5bpsBps   float64  `json:"return_30m_5bps_bps"`
	Return60m5bpsBps   float64  `json:"return_60m_5bps_bps"`
	Return120m5bpsBps  float64  `json:"return_120m_5bps_bps"`
	Return240m5bpsBps  float64  `json:"return_240m_5bps_bps"`
	EntryDelay1c60mBps *float64 `json:"entry_delay_1c_return_60m_5bps_bps,omitempty"`
	SignalReasons      []string `json:"signal_reasons,omitempty"`
	LeakageStatus      string   `json:"leakage_status"`
}

type FundingChunkSummary struct {
	Symbol                        string         `json:"symbol"`
	Year                          string         `json:"year"`
	Month                         string         `json:"month"`
	Status                        string         `json:"status"`
	FeatureFile                   string         `json:"feature_file"`
	ContextFile                   string         `json:"context_file"`
	EventFile                     string         `json:"event_file"`
	FeatureRows                   int            `json:"feature_rows"`
	ContextRows                   int            `json:"context_rows"`
	EventCount                    int            `json:"event_count"`
	MissingFeatureFile            bool           `json:"missing_feature_file"`
	MissingContextFile            bool           `json:"missing_context_file"`
	ParseError                    string         `json:"parse_error,omitempty"`
	RowsWithFunding               int            `json:"rows_with_funding"`
	RowsWithFundingUnknown        int            `json:"rows_with_funding_unknown"`
	WarmupRows                    int            `json:"warmup_rows"`
	FutureFundingJoinRowsRejected int            `json:"future_funding_join_rows_rejected"`
	UnsupportedContextRows        int            `json:"unsupported_context_rows"`
	RowsMissingForwardReturns     int            `json:"rows_missing_forward_returns"`
	DuplicateEventRowsRejected    int            `json:"duplicate_event_rows_rejected"`
	FundingCoveragePct            float64        `json:"funding_coverage_pct"`
	MinFundingRate                float64        `json:"min_funding_rate"`
	MedianFundingRate             float64        `json:"median_funding_rate"`
	MaxFundingRate                float64        `json:"max_funding_rate"`
	FundingRateZScoreAvailable    bool           `json:"funding_rate_zscore_available"`
	AsOfJoinLeakageStatus         string         `json:"asof_join_leakage_status"`
	LeakageStatus                 string         `json:"leakage_status"`
	FamilyEventCounts             map[string]int `json:"family_event_counts"`
	SideEventCounts               map[string]int `json:"side_event_counts"`
	ZeroEventMonth                bool           `json:"zero_event_month"`
}

type FundingDiagnostics struct {
	Symbol                                string `json:"symbol"`
	Month                                 string `json:"month"`
	RowsSeen                              int    `json:"rows_seen"`
	RowsWithFunding                       int    `json:"rows_with_funding"`
	RowsUnknownFunding                    int    `json:"rows_unknown_funding"`
	RowsWarmupFunding                     int    `json:"rows_warmup_funding"`
	RowsContextUnsupported                int    `json:"rows_context_unsupported"`
	RowsBetaFlat                          int    `json:"rows_beta_flat"`
	RowsBetaUp                            int    `json:"rows_beta_up"`
	RowsBetaDown                          int    `json:"rows_beta_down"`
	NegativeFundingCandidates             int    `json:"negative_funding_candidates"`
	NegativeFundingCandidatesAfterContext int    `json:"negative_funding_candidates_after_context"`
	NegativeFundingEventsEmitted          int    `json:"negative_funding_events_emitted"`
	PositiveFundingCandidates             int    `json:"positive_funding_candidates"`
	FundingFlipCandidates                 int    `json:"funding_flip_candidates"`
	BreakoutFundingLongCandidates         int    `json:"breakout_funding_long_candidates"`
	BreakoutFundingShortCandidates        int    `json:"breakout_funding_short_candidates"`
	BreakoutFundingLongEventsEmitted      int    `json:"breakout_funding_long_events_emitted"`
	BreakoutFundingShortEventsEmitted     int    `json:"breakout_funding_short_events_emitted"`
	BreakoutRejectedFundingCondition      int    `json:"breakout_rejected_funding_condition"`
	BreakoutRejectedPriceConfirmation     int    `json:"breakout_rejected_price_confirmation"`
	BreakoutRejectedVolatilityExpansion   int    `json:"breakout_rejected_volatility_expansion"`
	BreakoutRejectedVolumeConfirmation    int    `json:"breakout_rejected_volume_confirmation"`
	BreakoutRejectedDirectionTrend        int    `json:"breakout_rejected_direction_trend"`
	VolumeImbalanceFundingLongCandidates  int    `json:"volume_imbalance_funding_long_candidates"`
	VolumeImbalanceFundingShortCandidates int    `json:"volume_imbalance_funding_short_candidates"`
	VolumeImbalanceFundingLongEmitted     int    `json:"volume_imbalance_funding_long_emitted"`
	VolumeImbalanceFundingShortEmitted    int    `json:"volume_imbalance_funding_short_emitted"`
	VolumeImbalanceRejectedFunding        int    `json:"volume_imbalance_rejected_funding"`
	VolumeImbalanceRejectedProxySignal    int    `json:"volume_imbalance_rejected_proxy_signal"`
	VolumeImbalanceRejectedProxyMissing   int    `json:"volume_imbalance_rejected_proxy_missing"`
	RejectedByContext                     int    `json:"rejected_by_context"`
	RejectedByFundingThreshold            int    `json:"rejected_by_funding_threshold"`
	RejectedByWarmup                      int    `json:"rejected_by_warmup"`
	RejectedByMissingForwardWindow        int    `json:"rejected_by_missing_forward_window"`
}

type fundingChunkConfig struct {
	Symbol            string
	Month             string
	FeatureFile       string
	ContextFile       string
	ChunksDir         string
	EventFormat       string
	RetainEventDetail bool
	MaxEventFileMB    int
}

func evaluateFundingChunkFiles(cfg fundingChunkConfig) (FundingChunkSummary, []FundingEventRow, error) {
	if cfg.ChunksDir == "" {
		cfg.ChunksDir = filepath.Join("runs", "reports", "chunks")
	}
	cfg.Symbol = strings.ToUpper(strings.TrimSpace(cfg.Symbol))
	if cfg.FeatureFile == "" {
		cfg.FeatureFile = filepath.Join("runs", "features", "chunks", cfg.Symbol, cfg.Month+"-funding.json")
	}
	if cfg.ContextFile == "" {
		cfg.ContextFile = filepath.Join("runs", "regimes", "chunks", cfg.Symbol, cfg.Month+"-context.json")
	}

	outDir := filepath.Join(cfg.ChunksDir, cfg.Symbol)
	ext := "jsonl"
	if cfg.EventFormat == "jsonl.gz" {
		ext = "jsonl.gz"
	}
	eventFile := filepath.Join(outDir, cfg.Month+"-funding-events."+ext)
	summaryFile := filepath.Join(outDir, cfg.Month+"-funding-summary.json")
	diagnosticsFile := filepath.Join(outDir, cfg.Month+"-funding-diagnostics.json")
	alphaSummaryFile := filepath.Join(outDir, cfg.Month+"-alpha-summary.json")
	summary := FundingChunkSummary{
		Symbol:                cfg.Symbol,
		Year:                  monthYear(cfg.Month),
		Month:                 cfg.Month,
		Status:                "PASS",
		FeatureFile:           cfg.FeatureFile,
		ContextFile:           cfg.ContextFile,
		EventFile:             eventFile,
		AsOfJoinLeakageStatus: "PASS",
		LeakageStatus:         "PASS",
		FamilyEventCounts:     make(map[string]int),
		SideEventCounts:       make(map[string]int),
	}

	if _, err := os.Stat(cfg.FeatureFile); err != nil {
		summary.MissingFeatureFile = true
	}
	if _, err := os.Stat(cfg.ContextFile); err != nil {
		summary.MissingContextFile = true
	}
	if summary.MissingFeatureFile || summary.MissingContextFile {
		summary.Status = "missing_data"
		summary.ZeroEventMonth = true
		diagnostics := FundingDiagnostics{Symbol: cfg.Symbol, Month: cfg.Month}
		return summary, nil, writeFundingChunkOutputs(summaryFile, eventFile, diagnosticsFile, alphaSummaryFile, summary, nil, diagnostics, nil, cfg.RetainEventDetail)
	}

	rows, err := readFundingFeatureRows(cfg.FeatureFile)
	if err != nil {
		summary.Status = "invalid_report_artifact"
		summary.ParseError = err.Error()
		summary.ZeroEventMonth = true
		_ = writeFundingChunkOutputs(summaryFile, eventFile, diagnosticsFile, alphaSummaryFile, summary, nil, FundingDiagnostics{Symbol: cfg.Symbol, Month: cfg.Month}, nil, cfg.RetainEventDetail)
		return summary, nil, err
	}
	labels, err := regime.ReadLabelsJSON(cfg.ContextFile)
	if err != nil {
		summary.Status = "unsupported_context"
		summary.ParseError = err.Error()
		summary.ZeroEventMonth = true
		_ = writeFundingChunkOutputs(summaryFile, eventFile, diagnosticsFile, alphaSummaryFile, summary, nil, FundingDiagnostics{Symbol: cfg.Symbol, Month: cfg.Month}, nil, cfg.RetainEventDetail)
		return summary, nil, err
	}
	summary.FeatureRows = len(rows)
	summary.ContextRows = len(labels)

	diagnostics := FundingDiagnostics{Symbol: cfg.Symbol, Month: cfg.Month}
	events := buildFundingEventsWithDiagnostics(rows, labels, &summary, &diagnostics)
	summary.EventCount = len(events)
	summary.ZeroEventMonth = len(events) == 0
	if summary.FutureFundingJoinRowsRejected > 0 {
		summary.AsOfJoinLeakageStatus = "FAIL"
		summary.LeakageStatus = "FAIL"
	}
	if summary.EventCount == 0 && summary.Status == "PASS" {
		if summary.FeatureRows > 0 && summary.UnsupportedContextRows >= summary.FeatureRows {
			summary.Status = "unsupported_context"
		} else {
			summary.Status = "zero_events"
		}
	}

	var alphaSummary []FundingAlphaSummaryRow
	if len(events) > 0 {
		eventsByCandidateHorizon := make(map[string][]FundingEventRow)
		for _, event := range events {
			for _, horizon := range defaultFundingHorizons {
				key := candidateHorizonKey(event.Symbol, event.Family, event.Side, horizon)
				eventsByCandidateHorizon[key] = append(eventsByCandidateHorizon[key], event)
			}
		}
		for key, groupEvents := range eventsByCandidateHorizon {
			parts := strings.Split(key, "|")
			if len(parts) != 4 {
				continue
			}
			symbol, family, side, horizon := parts[0], parts[1], parts[2], parts[3]
			metrics := computeFundingMetrics(groupEvents, horizon)
			alphaSummary = append(alphaSummary, FundingAlphaSummaryRow{
				Symbol:               symbol,
				Year:                 monthYear(cfg.Month),
				Quarter:              quarterFromMonth(cfg.Month),
				Month:                cfg.Month,
				Family:               family,
				Side:                 side,
				Horizon:              horizon,
				SummarySchemaVersion: fundingAlphaSummarySchemaVersion,
				ClusterKeyVersion:    "1.0-native",
				Stats:                metrics,
			})
		}
		sort.Slice(alphaSummary, func(i, j int) bool {
			return strings.Join([]string{alphaSummary[i].Family, alphaSummary[i].Side, alphaSummary[i].Horizon}, "|") <
				strings.Join([]string{alphaSummary[j].Family, alphaSummary[j].Side, alphaSummary[j].Horizon}, "|")
		})
	}

	var v2Rows []NativeSummaryV2Row
	if len(events) > 0 {
		inputHash := hashString(fmt.Sprintf("%s|%d|%d", cfg.Symbol, len(rows), len(labels)))
		v2Rows = computeNativeSummaryV2(events, rows, inputHash)
		v2File := filepath.Join(outDir, cfg.Month+"-native-summary-v2.json")
		v2Data, _ := json.MarshalIndent(v2Rows, "", "  ")
		os.WriteFile(v2File, v2Data, 0644)
	}

	return summary, events, writeFundingChunkOutputs(summaryFile, eventFile, diagnosticsFile, alphaSummaryFile, summary, events, diagnostics, alphaSummary, cfg.RetainEventDetail)
}

type FundingAlphaSummaryRow struct {
	Symbol               string         `json:"symbol"`
	Year                 string         `json:"year"`
	Quarter              string         `json:"quarter"`
	Month                string         `json:"month"`
	Family               string         `json:"family"`
	Side                 string         `json:"side"`
	Horizon              string         `json:"horizon"`
	SummarySchemaVersion string         `json:"summary_schema_version"`
	ClusterKeyVersion    string         `json:"cluster_key_version"`
	Stats                FundingMetrics `json:"stats"`
}

const fundingAlphaSummarySchemaVersion = "10.7K-native"

func readFundingFeatureRows(path string) ([]ResearchFeatureRow, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var rows []ResearchFeatureRow
	if err := json.Unmarshal(data, &rows); err != nil {
		return nil, err
	}
	sort.Slice(rows, func(i, j int) bool {
		if rows[i].EventTimeMS == rows[j].EventTimeMS {
			return rows[i].AvailableAtMS < rows[j].AvailableAtMS
		}
		return rows[i].EventTimeMS < rows[j].EventTimeMS
	})
	return rows, nil
}

func writeFundingChunkOutputs(summaryFile, eventFile, diagnosticsFile, alphaSummaryFile string, summary FundingChunkSummary, events []FundingEventRow, diagnostics FundingDiagnostics, alphaSummary []FundingAlphaSummaryRow, retainEventDetail bool) error {
	if err := os.MkdirAll(filepath.Dir(summaryFile), 0755); err != nil {
		return err
	}

	if retainEventDetail {
		if err := writeFundingEventsJSONL(eventFile, events); err != nil {
			return err
		}
	}

	if len(alphaSummary) > 0 {
		alphaData, err := json.MarshalIndent(alphaSummary, "", "  ")
		if err != nil {
			return err
		}
		if err := os.WriteFile(alphaSummaryFile, alphaData, 0644); err != nil {
			return err
		}
	} else if alphaSummary != nil {
		if err := os.WriteFile(alphaSummaryFile, []byte("[]"), 0644); err != nil {
			return err
		}
	}

	data, err := json.MarshalIndent(summary, "", "  ")
	if err != nil {
		return err
	}
	if err := os.WriteFile(summaryFile, data, 0644); err != nil {
		return err
	}
	if diagnosticsFile != "" {
		diagData, err := json.MarshalIndent(diagnostics, "", "  ")
		if err != nil {
			return err
		}
		if err := os.WriteFile(diagnosticsFile, diagData, 0644); err != nil {
			return err
		}
	}
	return nil
}

func buildFundingEvents(rows []ResearchFeatureRow, labels []regime.Label, summary *FundingChunkSummary) []FundingEventRow {
	return buildFundingEventsWithDiagnostics(rows, labels, summary, nil)
}

func buildFundingEventsWithDiagnostics(rows []ResearchFeatureRow, labels []regime.Label, summary *FundingChunkSummary, diagnostics *FundingDiagnostics) []FundingEventRow {
	sort.Slice(labels, func(i, j int) bool {
		if labels[i].AvailableAtMS == labels[j].AvailableAtMS {
			return labels[i].EventTimeMS < labels[j].EventTimeMS
		}
		return labels[i].AvailableAtMS < labels[j].AvailableAtMS
	})

	var events []FundingEventRow
	var fundingHistory []float64
	lastObservedFunding := math.NaN()
	lastObservedFundingBucket := int64(-1)
	seen := make(map[string]struct{})
	var fundingRates []float64
	familyFilter := fundingFamilyFilterFromEnv()

	for i := range rows {
		row := rows[i]
		if diagnostics != nil {
			diagnostics.RowsSeen++
		}
		rate, ok := rowFundingRate(row)
		if !ok {
			summary.RowsWithFundingUnknown++
			if diagnostics != nil {
				diagnostics.RowsUnknownFunding++
			}
			continue
		}
		summary.RowsWithFunding++
		if diagnostics != nil {
			diagnostics.RowsWithFunding++
		}
		fundingRates = append(fundingRates, rate)

		if row.AvailableAtMS < row.EventTimeMS {
			summary.FutureFundingJoinRowsRejected++
			updateFundingHistory(&fundingHistory, &lastObservedFunding, &lastObservedFundingBucket, row.EventTimeMS, rate)
			continue
		}

		if row.Warmup || len(fundingHistory) < 20 {
			summary.WarmupRows++
			if diagnostics != nil {
				diagnostics.RowsWarmupFunding++
				diagnostics.RejectedByWarmup++
			}
			updateFundingHistory(&fundingHistory, &lastObservedFunding, &lastObservedFundingBucket, row.EventTimeMS, rate)
			continue
		}

		z, p20, p80, mean, sd, ok := fundingRollingStats(fundingHistory, rate)
		if !ok {
			if row.Symbol == "XRPUSDT" {
				emitParityLedgerRow(ParityLedgerRow{
					EventTimeMS: row.EventTimeMS, DecisionTimeMS: row.AvailableAtMS,
					Symbol: "XRPUSDT", Family: "NegativeFundingLong", Side: "long", HorizonMinutes: 60,
					FundingRate: rate, TrailingFundingMean: 0, TrailingFundingStd: 0,
					TrailingFundingZ: 0, TrailingFundingP20: 0,
					TriggerZLTEMinus1: false, TriggerRateLTEP20: false,
					FundingBucket: "unknown", RegimeBucket: "unknown", FundingXRegimeBucket: "unknown",
					DelayCandles: 0, CostBps: 10, ExpectedEdgeBps: 0, RealizedReturnBps: 0,
					ValidFeatureState: true, ValidFundingState: false, ValidRegimeState: false,
					NoTradeReason: "warmup", InputHash: "", SignalHash: "",
				})
			}
			summary.WarmupRows++
			if diagnostics != nil {
				diagnostics.RowsWarmupFunding++
				diagnostics.RejectedByWarmup++
			}
			updateFundingHistory(&fundingHistory, &lastObservedFunding, &lastObservedFundingBucket, row.EventTimeMS, rate)
			continue
		}
		summary.FundingRateZScoreAvailable = true

		change, changeOK := fundingRateChange(row, rate, fundingHistory)
		priorRate := rate
		if changeOK {
			priorRate = rate - change
		}

		negativeExtreme := z <= -1 || rate <= p20
		positiveExtreme := z >= 1 || rate >= p80
		bucket := fundingBucket(rate, z, p20, p80)
		flipLong := changeOK && priorRate < 0 && change > 0
		flipShort := changeOK && priorRate > 0 && change < 0
		breakoutLong := evaluateBreakoutFundingLong(row, negativeExtreme, flipLong)
		breakoutShort := evaluateBreakoutFundingShort(row, positiveExtreme, flipShort)
		volumeImbalanceLong := evaluateVolumeImbalanceFundingReversionProxyLong(row, negativeExtreme, flipLong)
		volumeImbalanceShort := evaluateVolumeImbalanceFundingReversionProxyShort(row, positiveExtreme, flipShort)
		if diagnostics != nil {
			if negativeExtreme {
				diagnostics.NegativeFundingCandidates++
			}
			if positiveExtreme {
				diagnostics.PositiveFundingCandidates++
			}
			if flipLong || flipShort {
				diagnostics.FundingFlipCandidates++
			}
			accumulateBreakoutDiagnostics(diagnostics, breakoutLong)
			accumulateBreakoutDiagnostics(diagnostics, breakoutShort)
			accumulateVolumeImbalanceDiagnostics(diagnostics, volumeImbalanceLong)
			accumulateVolumeImbalanceDiagnostics(diagnostics, volumeImbalanceShort)
		}

		label, ok := fundingContextAt(labels, row.AvailableAtMS)
		if !ok || fundingUnsupportedContextLabel(label) {
			if row.Symbol == "XRPUSDT" {
				emitParityLedgerRow(ParityLedgerRow{
					EventTimeMS: row.EventTimeMS, DecisionTimeMS: row.AvailableAtMS,
					Symbol: "XRPUSDT", Family: "NegativeFundingLong", Side: "long", HorizonMinutes: 60,
					FundingRate: rate, TrailingFundingMean: mean, TrailingFundingStd: sd,
					TrailingFundingZ: z, TrailingFundingP20: p20,
					TriggerZLTEMinus1: z <= -1, TriggerRateLTEP20: rate <= p20,
					FundingBucket: bucket, RegimeBucket: "unknown", FundingXRegimeBucket: "unknown",
					DelayCandles: 0, CostBps: 10, ExpectedEdgeBps: 0, RealizedReturnBps: 0,
					ValidFeatureState: true, ValidFundingState: true, ValidRegimeState: false,
					NoTradeReason: "unsupported_context", InputHash: "", SignalHash: "",
				})
			}
			summary.UnsupportedContextRows++
			if diagnostics != nil {
				diagnostics.RowsContextUnsupported++
				if negativeExtreme || positiveExtreme || flipLong || flipShort {
					diagnostics.RejectedByContext++
				}
			}
			updateFundingHistory(&fundingHistory, &lastObservedFunding, &lastObservedFundingBucket, row.EventTimeMS, rate)
			continue
		}
		if label.Warmup {
			summary.WarmupRows++
			if diagnostics != nil {
				diagnostics.RowsWarmupFunding++
				diagnostics.RejectedByWarmup++
			}
			updateFundingHistory(&fundingHistory, &lastObservedFunding, &lastObservedFundingBucket, row.EventTimeMS, rate)
			continue
		}
		if diagnostics != nil {
			switch strings.ToLower(label.MarketBeta) {
			case "btc_up":
				diagnostics.RowsBetaUp++
			case "btc_down":
				diagnostics.RowsBetaDown++
			default:
				diagnostics.RowsBetaFlat++
			}
			if negativeExtreme {
				diagnostics.NegativeFundingCandidatesAfterContext++
			}
			if !negativeExtreme && !positiveExtreme && !flipLong && !flipShort {
				diagnostics.RejectedByFundingThreshold++
			}
		}

		longReturns, ok := fundingForwardReturns(rows, i, "long")
		if !ok {
			summary.RowsMissingForwardReturns++
			if diagnostics != nil && (negativeExtreme || flipLong) {
				diagnostics.RejectedByMissingForwardWindow++
			}
			updateFundingHistory(&fundingHistory, &lastObservedFunding, &lastObservedFundingBucket, row.EventTimeMS, rate)
			continue
		}
		shortReturns, ok := fundingForwardReturns(rows, i, "short")
		if !ok {
			summary.RowsMissingForwardReturns++
			if diagnostics != nil && (positiveExtreme || flipShort) {
				diagnostics.RejectedByMissingForwardWindow++
			}
			updateFundingHistory(&fundingHistory, &lastObservedFunding, &lastObservedFundingBucket, row.EventTimeMS, rate)
			continue
		}

		candidates := []struct {
			family string
			side   string
			match  bool
			ret    fundingReturnSet
		}{
			{"NegativeFundingLong", "long", negativeExtreme, longReturns},
			{"PositiveFundingShort", "short", positiveExtreme, shortReturns},
			{"FundingFlipLong", "long", flipLong, longReturns},
			{"FundingFlipShort", "short", flipShort, shortReturns},
			{"RegimeFundingLong", "long", negativeExtreme && fundingLongRegime(label), longReturns},
			{"RegimeFundingShort", "short", positiveExtreme && fundingShortRegime(label), shortReturns},
			{"ConfirmedNegativeFundingLong", "long", negativeExtreme && (row.TrendSlope20 > 0 || row.Return15 > 0 || row.Close > row.EMA20), longReturns},
			{"ConfirmedPositiveFundingShort", "short", positiveExtreme && (row.TrendSlope20 < 0 || row.Return15 < 0 || row.Close < row.EMA20), shortReturns},
			{"BreakoutFundingLong", "long", breakoutLong.Match, longReturns},
			{"BreakoutFundingShort", "short", breakoutShort.Match, shortReturns},
			{"VolumeImbalanceFundingReversionProxyLong", "long", volumeImbalanceLong.Match, longReturns},
			{"VolumeImbalanceFundingReversionProxyShort", "short", volumeImbalanceShort.Match, shortReturns},
		}

		for _, candidate := range candidates {
			if len(familyFilter) > 0 && !familyFilter[candidate.family] {
				continue
			}
			if !candidate.match {
				continue
			}
			key := row.Symbol + "|" + candidate.family + "|" + candidate.side + "|" + strconv.FormatInt(row.EventTimeMS, 10)
			if _, exists := seen[key]; exists {
				summary.DuplicateEventRowsRejected++
				continue
			}
			seen[key] = struct{}{}
			event := FundingEventRow{
				Symbol:             strings.ToUpper(row.Symbol),
				Family:             candidate.family,
				Side:               candidate.side,
				EventTimeMS:        row.EventTimeMS,
				AvailableAtMS:      row.AvailableAtMS,
				EntryPrice:         row.Close,
				FundingRate:        rate,
				FundingRateZScore:  z,
				FundingBucket:      bucket,
				FundingRateUnknown: false,
				RegimeComposite:    label.Composite,
				Volatility:         label.Volatility,
				Trend:              label.Trend,
				Liquidity:          label.Liquidity,
				MarketBeta:         label.MarketBeta,
				Return5mBps:        candidate.ret.byHorizon["5m"],
				Return15mBps:       candidate.ret.byHorizon["15m"],
				Return30mBps:       candidate.ret.byHorizon["30m"],
				Return60mBps:       candidate.ret.byHorizon["60m"],
				Return120mBps:      candidate.ret.byHorizon["120m"],
				Return240mBps:      candidate.ret.byHorizon["240m"],
				Return5m5bpsBps:    candidate.ret.byHorizon["5m"] - 5,
				Return15m5bpsBps:   candidate.ret.byHorizon["15m"] - 5,
				Return30m5bpsBps:   candidate.ret.byHorizon["30m"] - 5,
				Return60m5bpsBps:   candidate.ret.byHorizon["60m"] - 5,
				Return120m5bpsBps:  candidate.ret.byHorizon["120m"] - 5,
				Return240m5bpsBps:  candidate.ret.byHorizon["240m"] - 5,
				EntryDelay1c60mBps: fundingEntryDelayReturn(rows, i, candidate.side),
				SignalReasons:      fundingSignalReasons(candidate.family, breakoutLong, breakoutShort, volumeImbalanceLong, volumeImbalanceShort),
				LeakageStatus:      "PASS",
			}
			events = append(events, event)
			summary.FamilyEventCounts[candidate.family]++
			summary.SideEventCounts[candidate.side]++
			if diagnostics != nil && candidate.family == "NegativeFundingLong" {
				diagnostics.NegativeFundingEventsEmitted++
			}
			if diagnostics != nil && candidate.family == "BreakoutFundingLong" {
				diagnostics.BreakoutFundingLongEventsEmitted++
			}
			if diagnostics != nil && candidate.family == "BreakoutFundingShort" {
				diagnostics.BreakoutFundingShortEventsEmitted++
			}
			if diagnostics != nil && candidate.family == "VolumeImbalanceFundingReversionProxyLong" {
				diagnostics.VolumeImbalanceFundingLongEmitted++
			}
			if diagnostics != nil && candidate.family == "VolumeImbalanceFundingReversionProxyShort" {
				diagnostics.VolumeImbalanceFundingShortEmitted++
			}
		}

		if row.Symbol == "XRPUSDT" && !fundingUnsupportedContextLabel(label) && !label.Warmup {
			reason := "not_extreme"
			if negativeExtreme {
				reason = ""
			}
			realizedReturn := 0.0
			if longReturns.byHorizon != nil {
				realizedReturn = longReturns.byHorizon["60m"]
			}
			lRow := ParityLedgerRow{
				EventTimeMS: row.EventTimeMS, DecisionTimeMS: row.AvailableAtMS,
				Symbol: "XRPUSDT", Family: "NegativeFundingLong", Side: "long", HorizonMinutes: 60,
				FundingRate: rate, TrailingFundingMean: mean, TrailingFundingStd: sd,
				TrailingFundingZ: z, TrailingFundingP20: p20,
				TriggerZLTEMinus1: z <= -1, TriggerRateLTEP20: rate <= p20,
				FundingBucket: bucket, RegimeBucket: label.Composite, FundingXRegimeBucket: bucket + "|" + label.Composite,
				DelayCandles: 0, CostBps: 10, ExpectedEdgeBps: 0, RealizedReturnBps: realizedReturn,
				ValidFeatureState: true, ValidFundingState: true, ValidRegimeState: true,
				NoTradeReason: reason,
			}
			lRow.InputHash = computeParityInputHash(lRow)
			lRow.SignalHash = computeParitySignalHash(lRow)
			emitParityLedgerRow(lRow)
		}
		updateFundingHistory(&fundingHistory, &lastObservedFunding, &lastObservedFundingBucket, row.EventTimeMS, rate)
	}

	if len(rows) > 0 {
		summary.FundingCoveragePct = float64(summary.RowsWithFunding) / float64(len(rows)) * 100
	}
	if len(fundingRates) > 0 {
		sort.Float64s(fundingRates)
		summary.MinFundingRate = fundingRates[0]
		summary.MedianFundingRate = median(fundingRates)
		summary.MaxFundingRate = fundingRates[len(fundingRates)-1]
	}
	return events
}

func fundingUnsupportedContextLabel(label regime.Label) bool {
	beta := strings.TrimSpace(label.MarketBeta)
	return beta == "" || strings.EqualFold(beta, "unknown") || strings.EqualFold(beta, "unsupported_context")
}

func rowFundingRate(row ResearchFeatureRow) (float64, bool) {
	if row.Derivatives.FundingRateUnknown || row.Derivatives.FundingRate == nil {
		return 0, false
	}
	return *row.Derivatives.FundingRate, true
}

func fundingRateChange(row ResearchFeatureRow, rate float64, history []float64) (float64, bool) {
	if !row.Derivatives.FundingRateChangeUnknown && row.Derivatives.FundingRateChange != nil {
		return *row.Derivatives.FundingRateChange, true
	}
	if len(history) == 0 {
		return 0, false
	}
	prev := history[len(history)-1]
	if rate == prev {
		return 0, false
	}
	return rate - prev, true
}

func fundingContextAt(labels []regime.Label, eventTimeMS int64) (regime.Label, bool) {
	idx := sort.Search(len(labels), func(i int) bool {
		return labels[i].AvailableAtMS > eventTimeMS
	}) - 1
	if idx < 0 {
		return regime.Label{}, false
	}
	label := labels[idx]
	if label.EventTimeMS > eventTimeMS || label.AvailableAtMS > eventTimeMS {
		return regime.Label{}, false
	}
	return label, true
}

func updateFundingHistory(history *[]float64, lastObserved *float64, lastBucket *int64, eventTimeMS int64, rate float64) {
	bucket := eventTimeMS / fundingRateIntervalMS
	if math.IsNaN(*lastObserved) || rate != *lastObserved || bucket != *lastBucket {
		*history = append(*history, rate)
		*lastObserved = rate
		*lastBucket = bucket
	}
}

func fundingRollingStats(history []float64, rate float64) (float64, float64, float64, float64, float64, bool) {
	if len(history) < 20 {
		return 0, 0, 0, 0, 0, false
	}
	mean := 0.0
	for _, value := range history {
		mean += value
	}
	mean /= float64(len(history))
	ss := 0.0
	for _, value := range history {
		d := value - mean
		ss += d * d
	}
	sd := math.Sqrt(ss / float64(len(history)))
	if sd == 0 {
		return 0, 0, 0, mean, sd, false
	}
	return (rate - mean) / sd, percentile(history, 0.20), percentile(history, 0.80), mean, sd, true
}

func fundingBucket(rate, z, p20, p80 float64) string {
	if z <= -1 || rate <= p20 {
		return "negative_extreme"
	}
	if z >= 1 || rate >= p80 {
		return "positive_extreme"
	}
	if rate < 0 {
		return "negative"
	}
	if rate > 0 {
		return "positive"
	}
	return "neutral"
}

func fundingLongRegime(label regime.Label) bool {
	return label.Composite == "compressed_range" ||
		(label.Volatility == "compressed" && label.Trend == "range") ||
		label.MarketBeta == "btc_up"
}

func fundingShortRegime(label regime.Label) bool {
	return label.Composite == "compressed_range" ||
		(label.Volatility == "compressed" && label.Trend == "range") ||
		label.MarketBeta == "btc_down"
}

type breakoutFundingDecision struct {
	Family                  string
	Match                   bool
	FundingCondition        bool
	BreakoutConfirmation    bool
	VolatilityExpansion     bool
	VolumeConfirmation      bool
	DirectionTrendAlignment bool
	Reasons                 []string
}

func evaluateBreakoutFundingLong(row ResearchFeatureRow, negativeExtreme, flipLong bool) breakoutFundingDecision {
	fundingOK := negativeExtreme || flipLong
	breakoutOK := row.Close > 0 && row.EMA20 > 0 && (row.Return15 > 0 || row.Close > row.EMA20)
	volatilityOK := row.BBWidthPctRank60 >= 0.60
	volumeOK := row.VolumeRatio20 >= 1.05
	trendOK := row.TrendSlope20 >= 0
	return buildBreakoutFundingDecision("BreakoutFundingLong", fundingOK, breakoutOK, volatilityOK, volumeOK, trendOK)
}

func evaluateBreakoutFundingShort(row ResearchFeatureRow, positiveExtreme, flipShort bool) breakoutFundingDecision {
	fundingOK := positiveExtreme || flipShort
	breakoutOK := row.Close > 0 && row.EMA20 > 0 && (row.Return15 < 0 || row.Close < row.EMA20)
	volatilityOK := row.BBWidthPctRank60 >= 0.60
	volumeOK := row.VolumeRatio20 >= 1.05
	trendOK := row.TrendSlope20 <= 0
	return buildBreakoutFundingDecision("BreakoutFundingShort", fundingOK, breakoutOK, volatilityOK, volumeOK, trendOK)
}

func buildBreakoutFundingDecision(family string, fundingOK, breakoutOK, volatilityOK, volumeOK, trendOK bool) breakoutFundingDecision {
	decision := breakoutFundingDecision{
		Family:                  family,
		FundingCondition:        fundingOK,
		BreakoutConfirmation:    breakoutOK,
		VolatilityExpansion:     volatilityOK,
		VolumeConfirmation:      volumeOK,
		DirectionTrendAlignment: trendOK,
		Reasons: []string{
			breakoutReason("funding_condition", fundingOK),
			breakoutReason("breakout_confirmation", breakoutOK),
			breakoutReason("volatility_expansion", volatilityOK),
			breakoutReason("volume_confirmation", volumeOK),
			breakoutReason("direction_trend_alignment", trendOK),
		},
	}
	decision.Match = fundingOK && breakoutOK && volatilityOK && volumeOK && trendOK
	return decision
}

func breakoutReason(name string, ok bool) string {
	if ok {
		return name + ":pass"
	}
	return name + ":fail"
}

func accumulateBreakoutDiagnostics(diagnostics *FundingDiagnostics, decision breakoutFundingDecision) {
	if diagnostics == nil {
		return
	}
	if decision.Match {
		switch decision.Family {
		case "BreakoutFundingLong":
			diagnostics.BreakoutFundingLongCandidates++
		case "BreakoutFundingShort":
			diagnostics.BreakoutFundingShortCandidates++
		}
		return
	}
	if !decision.FundingCondition {
		diagnostics.BreakoutRejectedFundingCondition++
	}
	if !decision.BreakoutConfirmation {
		diagnostics.BreakoutRejectedPriceConfirmation++
	}
	if !decision.VolatilityExpansion {
		diagnostics.BreakoutRejectedVolatilityExpansion++
	}
	if !decision.VolumeConfirmation {
		diagnostics.BreakoutRejectedVolumeConfirmation++
	}
	if !decision.DirectionTrendAlignment {
		diagnostics.BreakoutRejectedDirectionTrend++
	}
}

type volumeImbalanceFundingDecision struct {
	Family           string
	Match            bool
	FundingCondition bool
	ProxyAvailable   bool
	ProxySignal      bool
	Reasons          []string
}

func evaluateVolumeImbalanceFundingReversionProxyLong(row ResearchFeatureRow, negativeExtreme, flipLong bool) volumeImbalanceFundingDecision {
	return buildVolumeImbalanceFundingDecision(
		"VolumeImbalanceFundingReversionProxyLong",
		negativeExtreme || flipLong,
		row.TakerBuyRatio,
		row.TakerBuyRatio > 0,
		func(ratio float64) bool { return ratio <= 0.45 },
	)
}

func evaluateVolumeImbalanceFundingReversionProxyShort(row ResearchFeatureRow, positiveExtreme, flipShort bool) volumeImbalanceFundingDecision {
	return buildVolumeImbalanceFundingDecision(
		"VolumeImbalanceFundingReversionProxyShort",
		positiveExtreme || flipShort,
		row.TakerBuyRatio,
		row.TakerBuyRatio > 0,
		func(ratio float64) bool { return ratio >= 0.55 },
	)
}

func buildVolumeImbalanceFundingDecision(family string, fundingOK bool, takerBuyRatio float64, proxyAvailable bool, proxyRule func(float64) bool) volumeImbalanceFundingDecision {
	proxySignal := proxyAvailable && proxyRule(takerBuyRatio)
	decision := volumeImbalanceFundingDecision{
		Family:           family,
		FundingCondition: fundingOK,
		ProxyAvailable:   proxyAvailable,
		ProxySignal:      proxySignal,
		Reasons: []string{
			breakoutReason("funding_condition", fundingOK),
			breakoutReason("taker_buy_ratio_proxy_available", proxyAvailable),
			breakoutReason("taker_buy_ratio_proxy_reversion_signal", proxySignal),
			"full_taker_buy_sell_volume_join:not_implemented",
		},
	}
	decision.Match = fundingOK && proxySignal
	return decision
}

func accumulateVolumeImbalanceDiagnostics(diagnostics *FundingDiagnostics, decision volumeImbalanceFundingDecision) {
	if diagnostics == nil {
		return
	}
	if decision.Match {
		switch decision.Family {
		case "VolumeImbalanceFundingReversionProxyLong":
			diagnostics.VolumeImbalanceFundingLongCandidates++
		case "VolumeImbalanceFundingReversionProxyShort":
			diagnostics.VolumeImbalanceFundingShortCandidates++
		}
		return
	}
	if !decision.FundingCondition {
		diagnostics.VolumeImbalanceRejectedFunding++
	}
	if !decision.ProxyAvailable {
		diagnostics.VolumeImbalanceRejectedProxyMissing++
		return
	}
	if !decision.ProxySignal {
		diagnostics.VolumeImbalanceRejectedProxySignal++
	}
}

func fundingSignalReasons(family string, longDecision, shortDecision breakoutFundingDecision, volumeImbalanceLong, volumeImbalanceShort volumeImbalanceFundingDecision) []string {
	switch family {
	case "BreakoutFundingLong":
		return append([]string(nil), longDecision.Reasons...)
	case "BreakoutFundingShort":
		return append([]string(nil), shortDecision.Reasons...)
	case "VolumeImbalanceFundingReversionProxyLong":
		return append([]string(nil), volumeImbalanceLong.Reasons...)
	case "VolumeImbalanceFundingReversionProxyShort":
		return append([]string(nil), volumeImbalanceShort.Reasons...)
	default:
		return nil
	}
}

func fundingFamilyFilterFromEnv() map[string]bool {
	value := strings.TrimSpace(os.Getenv("AK_ENGINE_FUNDING_FAMILY_FILTER"))
	if value == "" {
		return nil
	}
	out := make(map[string]bool)
	for _, part := range strings.Split(value, ",") {
		family := strings.TrimSpace(part)
		if family != "" {
			out[family] = true
		}
	}
	return out
}

type fundingReturnSet struct {
	byHorizon map[string]float64
}

func fundingForwardReturns(rows []ResearchFeatureRow, start int, side string) (fundingReturnSet, bool) {
	entry := rows[start].Close
	if entry <= 0 {
		return fundingReturnSet{}, false
	}
	out := fundingReturnSet{byHorizon: make(map[string]float64)}
	for _, horizon := range defaultFundingHorizons {
		target := rows[start].EventTimeMS + fundingHorizonMS[horizon]
		future, ok := futureFundingClose(rows, start, target)
		if !ok || future <= 0 {
			return fundingReturnSet{}, false
		}
		out.byHorizon[horizon] = directionalReturnBps(entry, future, side)
	}
	return out, true
}

func fundingEntryDelayReturn(rows []ResearchFeatureRow, start int, side string) *float64 {
	if start+1 >= len(rows) || rows[start+1].Close <= 0 {
		return nil
	}
	target := rows[start+1].EventTimeMS + fundingHorizonMS["60m"]
	future, ok := futureFundingClose(rows, start+1, target)
	if !ok || future <= 0 {
		return nil
	}
	value := directionalReturnBps(rows[start+1].Close, future, side) - 5
	return &value
}

func futureFundingClose(rows []ResearchFeatureRow, start int, targetTimeMS int64) (float64, bool) {
	for i := start + 1; i < len(rows); i++ {
		if rows[i].EventTimeMS >= targetTimeMS {
			return rows[i].Close, true
		}
	}
	return 0, false
}

func directionalReturnBps(entry, future float64, side string) float64 {
	if strings.EqualFold(side, "short") {
		return (entry - future) / entry * 10000
	}
	return (future - entry) / entry * 10000
}

func writeFundingEventsJSONL(path string, events []FundingEventRow) error {
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return err
	}

	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()

	var w io.Writer = f
	if strings.HasSuffix(path, ".gz") {
		gw := gzip.NewWriter(f)
		defer gw.Close()
		w = gw
	}

	enc := json.NewEncoder(w)
	for _, event := range events {
		if err := enc.Encode(event); err != nil {
			return err
		}
	}
	return nil
}

func readFundingEventsJSONL(path string) ([]FundingEventRow, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	var r io.Reader = file
	if strings.HasSuffix(path, ".gz") {
		gr, err := gzip.NewReader(file)
		if err != nil {
			return nil, err
		}
		defer gr.Close()
		r = gr
	}

	var events []FundingEventRow
	scanner := bufio.NewScanner(r)
	// increase buffer size if rows get huge
	buf := make([]byte, 1024*1024)
	scanner.Buffer(buf, 10*1024*1024)

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}
		var event FundingEventRow
		if err := json.Unmarshal([]byte(line), &event); err != nil {
			return nil, err
		}
		events = append(events, event)
	}
	if err := scanner.Err(); err != nil {
		return nil, err
	}
	return events, nil
}

func monthYear(month string) string {
	if len(month) >= 4 {
		return month[:4]
	}
	return ""
}

func quarterFromMonth(month string) string {
	if len(month) != 7 {
		return ""
	}
	m, err := strconv.Atoi(month[5:7])
	if err != nil || m < 1 || m > 12 {
		return ""
	}
	return fmt.Sprintf("%s-Q%d", month[:4], ((m-1)/3)+1)
}

func monthFromEventTime(eventTimeMS int64) string {
	if eventTimeMS <= 0 {
		return ""
	}
	// UTC calendar month without importing time in hot aggregation paths.
	// Go's Unix conversion is still deterministic.
	return unixMonth(eventTimeMS)
}

func percentile(values []float64, p float64) float64 {
	if len(values) == 0 {
		return 0
	}
	cp := append([]float64(nil), values...)
	sort.Float64s(cp)
	if len(cp) == 1 {
		return cp[0]
	}
	pos := p * float64(len(cp)-1)
	lo := int(math.Floor(pos))
	hi := int(math.Ceil(pos))
	if lo == hi {
		return cp[lo]
	}
	weight := pos - float64(lo)
	return cp[lo]*(1-weight) + cp[hi]*weight
}

func median(values []float64) float64 {
	if len(values) == 0 {
		return 0
	}
	cp := append([]float64(nil), values...)
	sort.Float64s(cp)
	mid := len(cp) / 2
	if len(cp)%2 == 1 {
		return cp[mid]
	}
	return (cp[mid-1] + cp[mid]) / 2
}

func roundMetric(value float64) float64 {
	return math.Round(value*1_000_000) / 1_000_000
}
