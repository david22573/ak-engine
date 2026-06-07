package app

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/davidmiguel22573/ak-engine/internal/data"
	"github.com/davidmiguel22573/ak-engine/internal/features"
	"github.com/davidmiguel22573/ak-engine/internal/regime"
	"github.com/davidmiguel22573/ak-engine/pkg/protocol"
	"github.com/spf13/cobra"
)

const (
	phase103BStatusRejected              = "rejected"
	phase103BStatusFragileEventArtifact  = "fragile_event_artifact"
	phase103BStatusFragileNeedsOOS       = "fragile_needs_extended_oos"
	phase103BStatusValidatedResearchLead = "validated_research_lead"

	phase103BClassificationDistributed         = "distributed"
	phase103BClassificationMonthConcentrated   = "month_concentrated"
	phase103BClassificationClusterConcentrated = "cluster_concentrated"
	phase103BClassificationBothConcentrated    = "both_month_and_cluster_concentrated"
)

var phase103BAllowedStatuses = map[string]bool{
	phase103BStatusRejected:              true,
	phase103BStatusFragileEventArtifact:  true,
	phase103BStatusFragileNeedsOOS:       true,
	phase103BStatusValidatedResearchLead: true,
}

var phase103BFragileTargets = []phase103BTarget{
	{Symbol: "AVAXUSDT", Family: "ShockFade", Side: "LONG"},
	{Symbol: "LINKUSDT", Family: "CompressionBreakout", Side: "SHORT"},
	{Symbol: "SOLUSDT", Family: "ShockFade", Side: "LONG"},
}

var (
	aflLeaderboard string
	aflOut         string
	aflPath        string
	aflMarket      string
	aflInterval    string
)

type phase103BTarget struct {
	Symbol string
	Family string
	Side   string
}

type Phase103BFragileLeadsReport struct {
	Summary             Phase103BFragileSummary       `json:"summary"`
	Leads               []Phase103BLeadAnalysis       `json:"leads"`
	OptionalValidations []Phase103BOptionalValidation `json:"optional_validations"`
}

type Phase103BFragileSummary struct {
	SourceLeaderboard    string   `json:"source_leaderboard"`
	OutputMarkdown       string   `json:"output_markdown"`
	OutputJSON           string   `json:"output_json"`
	CandlePath           string   `json:"candle_path"`
	Market               string   `json:"market"`
	Interval             string   `json:"interval"`
	ExpectancyUnit       string   `json:"expectancy_unit"`
	FinalRecommendation  string   `json:"final_recommendation"`
	AllowedFinalStatuses []string `json:"allowed_final_statuses"`
}

type Phase103BLeadAnalysis struct {
	Symbol                          string                           `json:"symbol"`
	Family                          string                           `json:"family"`
	Side                            string                           `json:"side"`
	SourceLeaderboardVerdict        string                           `json:"source_leaderboard_verdict,omitempty"`
	SourceLeaderboardFailedGates    []string                         `json:"source_leaderboard_failed_gates,omitempty"`
	EventCount                      int                              `json:"event_count"`
	LeakageStatus                   string                           `json:"leakage_status"`
	FYPF5bps                        float64                          `json:"fy_pf_5bps"`
	H2PF5bps                        float64                          `json:"h2_pf_5bps"`
	WorstQuarterPF5bps              float64                          `json:"worst_quarter_pf_5bps"`
	FYExpectancy5bpsBps             float64                          `json:"fy_expectancy_5bps_bps"`
	H2Expectancy5bpsBps             float64                          `json:"h2_expectancy_5bps_bps"`
	PositiveMonthCount              int                              `json:"positive_month_count"`
	EntryDelay1cExpectancyBps       float64                          `json:"entry_delay_1c_expectancy_bps"`
	MonthlyContribution             []Phase103BMonthlyContribution   `json:"monthly_contribution"`
	Top1MonthContributionPct        float64                          `json:"top_1_month_contribution_pct"`
	Top2MonthContributionPct        float64                          `json:"top_2_month_contribution_pct"`
	Top3MonthContributionPct        float64                          `json:"top_3_month_contribution_pct"`
	EventClusterCount               int                              `json:"event_cluster_count"`
	LargestClusterContributionPct   float64                          `json:"largest_cluster_contribution_pct"`
	Largest5ClustersContributionPct float64                          `json:"largest_5_clusters_contribution_pct"`
	TopClusters                     []Phase103BEventCluster          `json:"top_clusters"`
	ConcentrationClassification     string                           `json:"concentration_classification"`
	LeaveOneMonthOut                []Phase103BLeaveOneOutResult     `json:"leave_one_month_out"`
	EventArtifactRisk               string                           `json:"event_artifact_risk"`
	LeaveOneQuarterOut              []Phase103BLeaveOneOutResult     `json:"leave_one_quarter_out"`
	QuarterFragile                  bool                             `json:"quarter_fragile"`
	BracketFailureDiagnosis         Phase103BBracketFailureDiagnosis `json:"bracket_failure_diagnosis"`
	FinalStatus                     string                           `json:"final_status"`
	FinalStatusReasons              []string                         `json:"final_status_reasons"`
	ExtendedOOSStatus               string                           `json:"extended_oos_status"`
}

type Phase103BMonthlyContribution struct {
	Month                    string  `json:"month"`
	EventCount               int     `json:"event_count"`
	NetContribution5bpsBps   float64 `json:"net_contribution_5bps_bps"`
	ProfitFactor5bps         float64 `json:"profit_factor_5bps"`
	Expectancy5bpsBps        float64 `json:"expectancy_5bps_bps"`
	ContributionPctOfNetEdge float64 `json:"contribution_pct_of_net_edge"`
}

type Phase103BEventCluster struct {
	ClusterID                int     `json:"cluster_id"`
	StartTime                string  `json:"start_time"`
	EndTime                  string  `json:"end_time"`
	EventCount               int     `json:"event_count"`
	NetContribution5bpsBps   float64 `json:"net_contribution_5bps_bps"`
	ContributionPctOfNetEdge float64 `json:"contribution_pct_of_net_edge"`
}

type Phase103BLeaveOneOutResult struct {
	MonthRemoved                     string  `json:"month_removed,omitempty"`
	QuarterRemoved                   string  `json:"quarter_removed,omitempty"`
	RemainingEventCount              int     `json:"remaining_event_count"`
	RemainingPF5bps                  float64 `json:"remaining_pf_5bps"`
	RemainingExpectancyBps           float64 `json:"remaining_expectancy_bps"`
	RemainingPositiveMonthCount      int     `json:"remaining_positive_month_count"`
	RemainingSingleMonthContribution float64 `json:"remaining_single_month_contribution_pct"`
	VerdictAfterRemoval              string  `json:"verdict_after_removal"`
}

type Phase103BBracketFailureDiagnosis struct {
	MedianTimeToMFEMinutes                           float64  `json:"median_time_to_mfe_minutes"`
	MedianTimeToMAEMinutes                           float64  `json:"median_time_to_mae_minutes"`
	PercentHittingSLBeforeLaterProfitable            float64  `json:"percent_hitting_sl_before_later_becoming_profitable"`
	PercentProfitable60mDespiteTouchingMinus5First   float64  `json:"percent_profitable_at_60m_despite_touching_minus_5_bps_first"`
	PercentProfitable240mDespiteTouchingMinus10First float64  `json:"percent_profitable_at_240m_despite_touching_minus_10_bps_first"`
	SameCandleTPSLConflictPct                        float64  `json:"same_candle_tp_sl_conflict_pct"`
	BestBracketName                                  string   `json:"best_bracket_name"`
	BestBracketPF5bps                                float64  `json:"best_bracket_pf_5bps"`
	BestBracketExpectancyBps                         float64  `json:"best_bracket_expectancy_bps"`
	FailureModes                                     []string `json:"failure_modes"`
}

type Phase103BOOSCoverageReport struct {
	Summary Phase103BOOSCoverageSummary `json:"summary"`
	Rows    []Phase103BOOSCoverageRow   `json:"rows"`
}

type Phase103BOOSCoverageSummary struct {
	CandlePath           string   `json:"candle_path"`
	Market               string   `json:"market"`
	Interval             string   `json:"interval"`
	Symbols              []string `json:"symbols"`
	Years                []int    `json:"years"`
	Full2024             bool     `json:"full_2024"`
	Full2025             bool     `json:"full_2025"`
	BlockedByMissingData bool     `json:"blocked_by_missing_data"`
	FetchCommands        []string `json:"fetch_commands,omitempty"`
}

type Phase103BOOSCoverageRow struct {
	Symbol             string   `json:"symbol"`
	Year               int      `json:"year"`
	HasFullCoverage    bool     `json:"has_full_coverage"`
	RowCount           int      `json:"row_count"`
	ExpectedRows       int      `json:"expected_rows"`
	MissingCandleCount int      `json:"missing_candle_count"`
	MissingMonths      []string `json:"missing_months,omitempty"`
	FirstCandleTime    string   `json:"first_candle_time,omitempty"`
	LastCandleTime     string   `json:"last_candle_time,omitempty"`
	Error              string   `json:"error,omitempty"`
	FetchCommand       string   `json:"fetch_command,omitempty"`
}

type Phase103BOptionalValidation struct {
	Symbol          string  `json:"symbol"`
	Family          string  `json:"family"`
	Side            string  `json:"side"`
	Year            int     `json:"year"`
	Status          string  `json:"status"`
	Reason          string  `json:"reason,omitempty"`
	OutputMarkdown  string  `json:"output_markdown,omitempty"`
	OutputJSON      string  `json:"output_json,omitempty"`
	FYPF5bps        float64 `json:"fy_pf_5bps,omitempty"`
	FYExpectancyBps float64 `json:"fy_expectancy_bps,omitempty"`
	FinalStatus     string  `json:"final_status,omitempty"`
}

type phase103BReturnEvent struct {
	EventTimeMS int64
	ReturnBps   float64
	Month       string
	Quarter     string
}

type phase103BLeadContext struct {
	Rows          []features.Row
	Labels        []regime.Label
	Candles       []protocol.Candle
	Events        []deepCandidateEvent
	ReturnEvents  []phase103BReturnEvent
	FeaturePath   string
	RegimePath    string
	CandlePath    string
	LeakageStatus string
	Year          int
}

var analyzeFragileLeadsCmd = &cobra.Command{
	Use:   "analyze-fragile-leads",
	Short: "Deep-test Phase 10.3B fragile research leads",
	RunE: func(cmd *cobra.Command, args []string) error {
		if aflLeaderboard == "" {
			return errors.New("missing --leaderboard")
		}
		if aflOut == "" {
			return errors.New("missing --out")
		}
		market := aflMarket
		if market == "" {
			market = "futures-um"
		}
		interval := aflInterval
		if interval == "" {
			interval = "1m"
		}
		candlePath := phase103BInferCandlePath(aflPath, market, interval)
		mdOut, jsonOut := normalizeMDAndJSONPaths(aflOut)
		report, coverage, err := buildPhase103BFragileReport(cmd.Context(), Phase103BFragileRequest{
			LeaderboardPath: aflLeaderboard,
			OutMarkdown:     mdOut,
			OutJSON:         jsonOut,
			CandlePath:      candlePath,
			Market:          market,
			Interval:        interval,
		})
		if err != nil {
			return err
		}
		if err := writeJSONFile(jsonOut, report); err != nil {
			return err
		}
		if err := os.WriteFile(mdOut, []byte(buildPhase103BFragileMD(report)), 0644); err != nil {
			return fmt.Errorf("write fragile leads markdown: %w", err)
		}
		coverageBase := filepath.Join(filepath.Dir(mdOut), "phase10_3b_extended_oos_coverage")
		if err := writeJSONFile(coverageBase+".json", coverage); err != nil {
			return err
		}
		if err := os.WriteFile(coverageBase+".md", []byte(buildPhase103BOOSCoverageMD(coverage)), 0644); err != nil {
			return fmt.Errorf("write extended oos coverage markdown: %w", err)
		}
		fmt.Printf("Phase 10.3B fragile leads report written to %s\n", mdOut)
		fmt.Printf("Phase 10.3B extended OOS coverage written to %s\n", coverageBase+".md")
		return nil
	},
}

type Phase103BFragileRequest struct {
	LeaderboardPath string
	OutMarkdown     string
	OutJSON         string
	CandlePath      string
	Market          string
	Interval        string
}

func init() {
	analyzeFragileLeadsCmd.Flags().StringVar(&aflLeaderboard, "leaderboard", "", "Phase 10.2C candidate leaderboard JSON")
	analyzeFragileLeadsCmd.Flags().StringVar(&aflOut, "out", "", "Output Markdown path")
	analyzeFragileLeadsCmd.Flags().StringVar(&aflPath, "path", "", "Optional local parquet candle workdir")
	analyzeFragileLeadsCmd.Flags().StringVar(&aflMarket, "market", "futures-um", "Market")
	analyzeFragileLeadsCmd.Flags().StringVar(&aflInterval, "interval", "1m", "Interval")
	rootCmd.AddCommand(analyzeFragileLeadsCmd)
}

func buildPhase103BFragileReport(ctx context.Context, req Phase103BFragileRequest) (Phase103BFragileLeadsReport, Phase103BOOSCoverageReport, error) {
	leaderboardRows, err := phase103BReadLeaderboardRows(req.LeaderboardPath)
	if err != nil {
		return Phase103BFragileLeadsReport{}, Phase103BOOSCoverageReport{}, err
	}
	coverage, err := buildPhase103BOOSCoverage(ctx, req.CandlePath, req.Market, req.Interval)
	if err != nil {
		return Phase103BFragileLeadsReport{}, Phase103BOOSCoverageReport{}, err
	}

	report := Phase103BFragileLeadsReport{
		Summary: Phase103BFragileSummary{
			SourceLeaderboard: req.LeaderboardPath,
			OutputMarkdown:    req.OutMarkdown,
			OutputJSON:        req.OutJSON,
			CandlePath:        req.CandlePath,
			Market:            req.Market,
			Interval:          req.Interval,
			ExpectancyUnit:    expectancyUnitBps,
			AllowedFinalStatuses: []string{
				phase103BStatusRejected,
				phase103BStatusFragileEventArtifact,
				phase103BStatusFragileNeedsOOS,
				phase103BStatusValidatedResearchLead,
			},
		},
	}

	for _, target := range phase103BFragileTargets {
		ctx2023, err := loadPhase103BLeadContext(ctx, req.CandlePath, req.Market, req.Interval, target, 2023)
		if err != nil {
			report.Leads = append(report.Leads, Phase103BLeadAnalysis{
				Symbol:             target.Symbol,
				Family:             target.Family,
				Side:               target.Side,
				FinalStatus:        phase103BStatusRejected,
				FinalStatusReasons: []string{err.Error()},
				ExtendedOOSStatus:  phase103BExtendedOOSStatus(coverage, target.Symbol),
			})
			continue
		}
		lead := analyzePhase103BLead(ctx2023, target, leaderboardRows[phase103BTargetKey(target)], coverage)
		report.Leads = append(report.Leads, lead)
	}

	report.OptionalValidations = buildPhase103BOptionalValidations(ctx, req, coverage)
	report.Summary.FinalRecommendation = phase103BFinalRecommendation(report.Leads)
	return report, coverage, nil
}

func phase103BReadLeaderboardRows(path string) (map[string]LeaderboardRow, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read leaderboard: %w", err)
	}
	var leaderboard LeaderboardJSON
	if err := json.Unmarshal(data, &leaderboard); err != nil {
		return nil, fmt.Errorf("unmarshal leaderboard: %w", err)
	}
	rows := make(map[string]LeaderboardRow)
	for _, row := range leaderboard.Rows {
		rows[phase103BTargetKey(phase103BTarget{Symbol: row.Symbol, Family: row.Family, Side: row.Side})] = row
	}
	return rows, nil
}

func phase103BTargetKey(t phase103BTarget) string {
	return strings.ToUpper(t.Symbol) + "_" + t.Family + "_" + strings.ToUpper(t.Side)
}

func phase103BInferCandlePath(explicit, market, interval string) string {
	if explicit != "" {
		return explicit
	}
	candidates := []string{
		filepath.Join("..", "ak-historian", ".ak-historian", "work"),
		filepath.Join(".ak-engine", "cache", "r2"),
	}
	for _, p := range candidates {
		probe := filepath.Join(p, "candles", market, interval)
		if st, err := os.Stat(probe); err == nil && st.IsDir() {
			return p
		}
	}
	return candidates[0]
}

func loadPhase103BLeadContext(ctx context.Context, candlePath, market, interval string, target phase103BTarget, year int) (phase103BLeadContext, error) {
	featurePath, regimePath, err := phase103BEnsureArtifacts(ctx, candlePath, market, interval, target.Symbol, year)
	if err != nil {
		return phase103BLeadContext{}, err
	}
	rows, err := features.ReadRowsJSON(featurePath)
	if err != nil {
		return phase103BLeadContext{}, fmt.Errorf("read features for %s %d: %w", target.Symbol, year, err)
	}
	labels, err := regime.ReadLabelsJSON(regimePath)
	if err != nil {
		return phase103BLeadContext{}, fmt.Errorf("read regimes for %s %d: %w", target.Symbol, year, err)
	}
	sort.Slice(rows, func(i, j int) bool { return rows[i].EventTimeMS < rows[j].EventTimeMS })
	sort.Slice(labels, func(i, j int) bool { return labels[i].AvailableAtMS < labels[j].AvailableAtMS })
	if len(rows) == 0 {
		return phase103BLeadContext{}, fmt.Errorf("features empty for %s %d", target.Symbol, year)
	}
	candles, err := loadDeepCandles(ctx, candlePath, market, interval, target.Symbol, rows)
	if err != nil {
		return phase103BLeadContext{}, fmt.Errorf("load candles for %s %d: %w", target.Symbol, year, err)
	}
	sort.Slice(candles, func(i, j int) bool { return candles[i].OpenTimeMS < candles[j].OpenTimeMS })
	events, _, leakageIssues, err := generateDeepCandidateEvents(rows, labels, candles, target.Family, strings.ToUpper(target.Side))
	if err != nil {
		return phase103BLeadContext{}, err
	}
	if len(events) == 0 {
		return phase103BLeadContext{}, fmt.Errorf("no accepted candidates for %s %s_%s", target.Symbol, target.Family, target.Side)
	}
	leakage := "PASS"
	if len(leakageIssues) > 0 {
		leakage = "FAIL"
	}
	_, yearEnd := phase103BYearBounds(year)
	returnEvents := phase103BReturnEvents(events, candles, yearEnd.UnixMilli())
	return phase103BLeadContext{
		Rows:          rows,
		Labels:        labels,
		Candles:       candles,
		Events:        events,
		ReturnEvents:  returnEvents,
		FeaturePath:   featurePath,
		RegimePath:    regimePath,
		CandlePath:    candlePath,
		LeakageStatus: leakage,
		Year:          year,
	}, nil
}

func phase103BEnsureArtifacts(ctx context.Context, candlePath, market, interval, symbol string, year int) (string, string, error) {
	if year == 2023 {
		featurePath, ok := discoverPhaseArtifactPath("features", symbol)
		if !ok {
			return "", "", fmt.Errorf("missing 2023 features for %s", symbol)
		}
		regimePath, ok := discoverPhaseArtifactPath("regimes", symbol)
		if !ok {
			return "", "", fmt.Errorf("missing 2023 regimes for %s", symbol)
		}
		return featurePath, regimePath, nil
	}
	featurePath := filepath.Join("runs", "features", fmt.Sprintf("%s-%d-FY-context.json", symbol, year))
	regimePath := filepath.Join("runs", "regimes", fmt.Sprintf("%s-%d-FY-context.json", symbol, year))
	if _, err := os.Stat(featurePath); err == nil {
		if _, err := os.Stat(regimePath); err == nil {
			return featurePath, regimePath, nil
		}
	}
	if err := phase103BBuildYearArtifacts(ctx, candlePath, market, interval, symbol, year, featurePath, regimePath); err != nil {
		return "", "", err
	}
	return featurePath, regimePath, nil
}

func phase103BBuildYearArtifacts(ctx context.Context, candlePath, market, interval, symbol string, year int, featurePath, regimePath string) error {
	start, end := phase103BYearBounds(year)
	src := data.NewLocalParquetSource()
	req := data.CandleRequest{Path: candlePath, Market: market, Interval: interval, Symbol: symbol, From: start, To: end}
	candles, err := src.LoadCandles(ctx, req)
	if err != nil {
		return fmt.Errorf("load %s %d candles: %w", symbol, year, err)
	}
	var btcCandles, ethCandles []protocol.Candle
	for _, ctxSym := range contextSymbolListForTarget(symbol, "BTCUSDT,ETHUSDT") {
		ctxReq := req
		ctxReq.Symbol = ctxSym
		ctxCandles, err := src.LoadCandles(ctx, ctxReq)
		if err != nil {
			return fmt.Errorf("load %s context for %s %d: %w", ctxSym, symbol, year, err)
		}
		if ctxSym == "BTCUSDT" {
			btcCandles = ctxCandles
		}
		if ctxSym == "ETHUSDT" {
			ethCandles = ctxCandles
		}
	}
	rows, err := features.BuildRows(candles, features.BuildOptions{
		Market:     market,
		Symbol:     symbol,
		Interval:   interval,
		ContextBTC: btcCandles,
		ContextETH: ethCandles,
	})
	if err != nil {
		return fmt.Errorf("build %s %d features: %w", symbol, year, err)
	}
	if err := os.MkdirAll(filepath.Dir(featurePath), 0755); err != nil {
		return err
	}
	if err := features.WriteRowsJSON(featurePath, rows); err != nil {
		return fmt.Errorf("write %s features: %w", symbol, err)
	}
	classifier := regime.NewClassifier(regime.ThresholdOptions{})
	labels, err := classifier.ClassifyRows(rows)
	if err != nil {
		return fmt.Errorf("classify %s %d regimes: %w", symbol, year, err)
	}
	if err := os.MkdirAll(filepath.Dir(regimePath), 0755); err != nil {
		return err
	}
	if err := regime.WriteLabelsJSON(regimePath, labels); err != nil {
		return fmt.Errorf("write %s regimes: %w", symbol, err)
	}
	return nil
}

func phase103BReturnEvents(events []deepCandidateEvent, candles []protocol.Candle, periodEndMS int64) []phase103BReturnEvent {
	out := make([]phase103BReturnEvent, 0, len(events))
	for _, event := range events {
		set := deepReturnsForEvents([]deepCandidateEvent{event}, candles, 60, 5, 0, periodEndMS)
		if len(set.ReturnsBps) != 1 {
			continue
		}
		t := time.UnixMilli(event.EventTimeMS).UTC()
		out = append(out, phase103BReturnEvent{
			EventTimeMS: event.EventTimeMS,
			ReturnBps:   set.ReturnsBps[0],
			Month:       t.Format("2006-01"),
			Quarter:     phase103BQuarter(t),
		})
	}
	return out
}

func phase103BReturnEventsForPeriod(events []deepCandidateEvent, candles []protocol.Candle, start, end time.Time) []phase103BReturnEvent {
	filtered := make([]deepCandidateEvent, 0, len(events))
	for _, event := range events {
		t := time.UnixMilli(event.EventTimeMS).UTC()
		if !t.Before(start) && !t.After(end) {
			filtered = append(filtered, event)
		}
	}
	return phase103BReturnEvents(filtered, candles, end.UnixMilli())
}

func phase103BMonthlyContributionsForEvents(events []deepCandidateEvent, candles []protocol.Candle, year int) []Phase103BMonthlyContribution {
	var out []Phase103BMonthlyContribution
	for month := time.January; month <= time.December; month++ {
		start := time.Date(year, month, 1, 0, 0, 0, 0, time.UTC)
		end := start.AddDate(0, 1, 0).Add(-time.Millisecond)
		monthEvents := phase103BReturnEventsForPeriod(events, candles, start, end)
		if len(monthEvents) == 0 {
			continue
		}
		m := phase103BMetricForReturns(monthEvents)
		out = append(out, Phase103BMonthlyContribution{
			Month:                  start.Format("2006-01"),
			EventCount:             m.EventCount,
			NetContribution5bpsBps: phase103BSumReturnBps(monthEvents),
			ProfitFactor5bps:       m.ProfitFactor,
			Expectancy5bpsBps:      m.ExpectancyBps,
		})
	}
	totalNet := 0.0
	for _, month := range out {
		totalNet += month.NetContribution5bpsBps
	}
	for i := range out {
		if totalNet > 0 && out[i].NetContribution5bpsBps > 0 {
			out[i].ContributionPctOfNetEdge = out[i].NetContribution5bpsBps / totalNet * 100.0
		} else if totalNet <= 0 && out[i].NetContribution5bpsBps > 0 {
			out[i].ContributionPctOfNetEdge = 100.0
		}
	}
	return out
}

func analyzePhase103BLead(ctx phase103BLeadContext, target phase103BTarget, source LeaderboardRow, coverage Phase103BOOSCoverageReport) Phase103BLeadAnalysis {
	fy := phase103BMetricForReturns(ctx.ReturnEvents)
	h2Start := time.Date(ctx.Year, time.July, 1, 0, 0, 0, 0, time.UTC)
	_, yearEnd := phase103BYearBounds(ctx.Year)
	h2 := phase103BMetricForReturns(phase103BReturnEventsForPeriod(ctx.Events, ctx.Candles, h2Start, yearEnd))
	worstQPF := math.MaxFloat64
	for _, bounds := range phase103BQuarterBounds(ctx.Year) {
		m := phase103BMetricForReturns(phase103BReturnEventsForPeriod(ctx.Events, ctx.Candles, bounds.Start, bounds.End))
		if m.ProfitFactor < worstQPF {
			worstQPF = m.ProfitFactor
		}
	}
	if worstQPF == math.MaxFloat64 {
		worstQPF = 0
	}
	delaySet := deepReturnsForEvents(ctx.Events, ctx.Candles, 60, 5, 1, yearEnd.UnixMilli())
	delayMetric := deepMetricFromBps(delaySet.ReturnsBps)
	monthly := phase103BMonthlyContributionsForEvents(ctx.Events, ctx.Candles, ctx.Year)
	top1 := phase103BTopContributionPctFromMonthly(monthly, 1)
	top2 := phase103BTopContributionPctFromMonthly(monthly, 2)
	top3 := phase103BTopContributionPctFromMonthly(monthly, 3)
	clusters := phase103BClusterEvents(ctx.ReturnEvents)
	topClusters := phase103BTopClusters(clusters, 5)
	largestCluster := phase103BTopClusterContributionPct(clusters, 1)
	largest5Clusters := phase103BTopClusterContributionPct(clusters, 5)
	classification := phase103BConcentrationClassification(top1, top2, largestCluster, largest5Clusters)
	lomo := phase103BLeaveOneMonthOutForEvents(ctx.Events, ctx.Candles, ctx.Year)
	loqo := phase103BLeaveOneQuarterOutForEvents(ctx.Events, ctx.Candles, ctx.Year)
	bracketDiag := phase103BBracketFailureDiagnosisForEvents(ctx.Events, ctx.Candles, target.Side)
	lead := Phase103BLeadAnalysis{
		Symbol:                          target.Symbol,
		Family:                          target.Family,
		Side:                            target.Side,
		SourceLeaderboardVerdict:        source.Verdict,
		SourceLeaderboardFailedGates:    source.FailedGates,
		EventCount:                      fy.EventCount,
		LeakageStatus:                   ctx.LeakageStatus,
		FYPF5bps:                        fy.ProfitFactor,
		H2PF5bps:                        h2.ProfitFactor,
		WorstQuarterPF5bps:              worstQPF,
		FYExpectancy5bpsBps:             fy.ExpectancyBps,
		H2Expectancy5bpsBps:             h2.ExpectancyBps,
		PositiveMonthCount:              phase103BPositiveMonthCount(monthly),
		EntryDelay1cExpectancyBps:       delayMetric.Expectancy,
		MonthlyContribution:             monthly,
		Top1MonthContributionPct:        top1,
		Top2MonthContributionPct:        top2,
		Top3MonthContributionPct:        top3,
		EventClusterCount:               len(clusters),
		LargestClusterContributionPct:   largestCluster,
		Largest5ClustersContributionPct: largest5Clusters,
		TopClusters:                     topClusters,
		ConcentrationClassification:     classification,
		LeaveOneMonthOut:                lomo,
		EventArtifactRisk:               phase103BEventArtifactRisk(lomo),
		LeaveOneQuarterOut:              loqo,
		QuarterFragile:                  phase103BQuarterFragile(loqo),
		BracketFailureDiagnosis:         bracketDiag,
		ExtendedOOSStatus:               phase103BExtendedOOSStatus(coverage, target.Symbol),
	}
	lead.FinalStatus, lead.FinalStatusReasons = phase103BFinalStatus(lead)
	if !phase103BAllowedStatuses[lead.FinalStatus] {
		lead.FinalStatus = phase103BStatusRejected
		lead.FinalStatusReasons = append(lead.FinalStatusReasons, "invalid 10.3B final status replaced")
	}
	return lead
}

func phase103BMetricForReturns(events []phase103BReturnEvent) CandidateMetric {
	rets := make([]float64, 0, len(events))
	for _, event := range events {
		rets = append(rets, event.ReturnBps)
	}
	m := deepMetricFromBps(rets)
	return CandidateMetric{
		EventCount:    m.Count,
		ProfitFactor:  m.PF,
		ExpectancyBps: m.Expectancy,
		WinRate:       m.WinRate,
	}
}

func phase103BFilterReturnEvents(events []phase103BReturnEvent, keep func(phase103BReturnEvent) bool) []phase103BReturnEvent {
	out := make([]phase103BReturnEvent, 0, len(events))
	for _, event := range events {
		if keep(event) {
			out = append(out, event)
		}
	}
	return out
}

func phase103BMonthlyContributions(events []phase103BReturnEvent) []Phase103BMonthlyContribution {
	byMonth := make(map[string][]phase103BReturnEvent)
	for _, event := range events {
		byMonth[event.Month] = append(byMonth[event.Month], event)
	}
	months := make([]string, 0, len(byMonth))
	for month := range byMonth {
		months = append(months, month)
	}
	sort.Strings(months)
	out := make([]Phase103BMonthlyContribution, 0, len(months))
	for _, month := range months {
		m := phase103BMetricForReturns(byMonth[month])
		net := phase103BSumReturnBps(byMonth[month])
		out = append(out, Phase103BMonthlyContribution{
			Month:                  month,
			EventCount:             m.EventCount,
			NetContribution5bpsBps: net,
			ProfitFactor5bps:       m.ProfitFactor,
			Expectancy5bpsBps:      m.ExpectancyBps,
		})
	}
	totalNet := 0.0
	for _, month := range out {
		totalNet += month.NetContribution5bpsBps
	}
	for i := range out {
		if totalNet > 0 && out[i].NetContribution5bpsBps > 0 {
			out[i].ContributionPctOfNetEdge = out[i].NetContribution5bpsBps / totalNet * 100.0
		} else if totalNet <= 0 && out[i].NetContribution5bpsBps > 0 {
			out[i].ContributionPctOfNetEdge = 100.0
		}
	}
	return out
}

func phase103BSumReturnBps(events []phase103BReturnEvent) float64 {
	total := 0.0
	for _, event := range events {
		total += event.ReturnBps
	}
	return total
}

func phase103BTopContributionPctFromMonthly(months []Phase103BMonthlyContribution, topN int) float64 {
	nets := make([]float64, 0, len(months))
	for _, month := range months {
		nets = append(nets, month.NetContribution5bpsBps)
	}
	return phase103BTopContributionPct(nets, topN)
}

func phase103BTopContributionPct(nets []float64, topN int) float64 {
	if topN <= 0 {
		return 0
	}
	total := 0.0
	var positives []float64
	for _, net := range nets {
		total += net
		if net > 0 {
			positives = append(positives, net)
		}
	}
	if len(positives) == 0 {
		return 0
	}
	sort.Sort(sort.Reverse(sort.Float64Slice(positives)))
	if topN > len(positives) {
		topN = len(positives)
	}
	top := 0.0
	for i := 0; i < topN; i++ {
		top += positives[i]
	}
	if total > 0 {
		return top / total * 100.0
	}
	return 100.0
}

func phase103BPositiveMonthCount(months []Phase103BMonthlyContribution) int {
	count := 0
	for _, month := range months {
		if month.NetContribution5bpsBps > 0 {
			count++
		}
	}
	return count
}

func phase103BClusterEvents(events []phase103BReturnEvent) []Phase103BEventCluster {
	if len(events) == 0 {
		return nil
	}
	sorted := append([]phase103BReturnEvent(nil), events...)
	sort.Slice(sorted, func(i, j int) bool { return sorted[i].EventTimeMS < sorted[j].EventTimeMS })
	totalNet := phase103BSumReturnBps(sorted)
	var clusters []Phase103BEventCluster
	clusterID := 1
	startIdx := 0
	for i := 1; i <= len(sorted); i++ {
		if i < len(sorted) && sorted[i].EventTimeMS-sorted[i-1].EventTimeMS <= 60*60*1000 {
			continue
		}
		evs := sorted[startIdx:i]
		net := phase103BSumReturnBps(evs)
		cluster := Phase103BEventCluster{
			ClusterID:              clusterID,
			StartTime:              time.UnixMilli(evs[0].EventTimeMS).UTC().Format(time.RFC3339),
			EndTime:                time.UnixMilli(evs[len(evs)-1].EventTimeMS).UTC().Format(time.RFC3339),
			EventCount:             len(evs),
			NetContribution5bpsBps: net,
		}
		if totalNet > 0 && net > 0 {
			cluster.ContributionPctOfNetEdge = net / totalNet * 100.0
		} else if totalNet <= 0 && net > 0 {
			cluster.ContributionPctOfNetEdge = 100.0
		}
		clusters = append(clusters, cluster)
		clusterID++
		startIdx = i
	}
	return clusters
}

func phase103BTopClusters(clusters []Phase103BEventCluster, n int) []Phase103BEventCluster {
	sorted := append([]Phase103BEventCluster(nil), clusters...)
	sort.Slice(sorted, func(i, j int) bool {
		return sorted[i].NetContribution5bpsBps > sorted[j].NetContribution5bpsBps
	})
	if n > len(sorted) {
		n = len(sorted)
	}
	return sorted[:n]
}

func phase103BTopClusterContributionPct(clusters []Phase103BEventCluster, topN int) float64 {
	nets := make([]float64, 0, len(clusters))
	for _, cluster := range clusters {
		nets = append(nets, cluster.NetContribution5bpsBps)
	}
	return phase103BTopContributionPct(nets, topN)
}

func phase103BConcentrationClassification(top1Month, top2Month, largestCluster, largest5Clusters float64) string {
	monthConcentrated := top1Month > 50 || top2Month > 70
	clusterConcentrated := largestCluster > 50 || largest5Clusters > 70
	switch {
	case monthConcentrated && clusterConcentrated:
		return phase103BClassificationBothConcentrated
	case monthConcentrated:
		return phase103BClassificationMonthConcentrated
	case clusterConcentrated:
		return phase103BClassificationClusterConcentrated
	default:
		return phase103BClassificationDistributed
	}
}

func phase103BLeaveOneMonthOut(events []phase103BReturnEvent, year int) []Phase103BLeaveOneOutResult {
	var out []Phase103BLeaveOneOutResult
	for month := time.January; month <= time.December; month++ {
		monthKey := time.Date(year, month, 1, 0, 0, 0, 0, time.UTC).Format("2006-01")
		filtered := phase103BFilterReturnEvents(events, func(ev phase103BReturnEvent) bool {
			return ev.Month != monthKey
		})
		out = append(out, phase103BLeaveOneOutResult(filtered, monthKey, ""))
	}
	return out
}

func phase103BLeaveOneQuarterOut(events []phase103BReturnEvent) []Phase103BLeaveOneOutResult {
	var out []Phase103BLeaveOneOutResult
	for _, q := range []string{"Q1", "Q2", "Q3", "Q4"} {
		filtered := phase103BFilterReturnEvents(events, func(ev phase103BReturnEvent) bool {
			return ev.Quarter != q
		})
		out = append(out, phase103BLeaveOneOutResult(filtered, "", q))
	}
	return out
}

func phase103BLeaveOneMonthOutForEvents(events []deepCandidateEvent, candles []protocol.Candle, year int) []Phase103BLeaveOneOutResult {
	var out []Phase103BLeaveOneOutResult
	_, yearEnd := phase103BYearBounds(year)
	for month := time.January; month <= time.December; month++ {
		monthKey := time.Date(year, month, 1, 0, 0, 0, 0, time.UTC).Format("2006-01")
		filtered := make([]deepCandidateEvent, 0, len(events))
		for _, event := range events {
			if time.UnixMilli(event.EventTimeMS).UTC().Format("2006-01") != monthKey {
				filtered = append(filtered, event)
			}
		}
		returnEvents := phase103BReturnEvents(filtered, candles, yearEnd.UnixMilli())
		out = append(out, phase103BLeaveOneOutResultForEvents(filtered, returnEvents, candles, year, monthKey, ""))
	}
	return out
}

func phase103BLeaveOneQuarterOutForEvents(events []deepCandidateEvent, candles []protocol.Candle, year int) []Phase103BLeaveOneOutResult {
	var out []Phase103BLeaveOneOutResult
	_, yearEnd := phase103BYearBounds(year)
	for _, q := range []string{"Q1", "Q2", "Q3", "Q4"} {
		filtered := make([]deepCandidateEvent, 0, len(events))
		for _, event := range events {
			if phase103BQuarter(time.UnixMilli(event.EventTimeMS).UTC()) != q {
				filtered = append(filtered, event)
			}
		}
		returnEvents := phase103BReturnEvents(filtered, candles, yearEnd.UnixMilli())
		out = append(out, phase103BLeaveOneOutResultForEvents(filtered, returnEvents, candles, year, "", q))
	}
	return out
}

func phase103BLeaveOneOutResultForEvents(events []deepCandidateEvent, returnEvents []phase103BReturnEvent, candles []protocol.Candle, year int, monthRemoved, quarterRemoved string) Phase103BLeaveOneOutResult {
	m := phase103BMetricForReturns(returnEvents)
	monthly := phase103BMonthlyContributionsForEvents(events, candles, year)
	result := Phase103BLeaveOneOutResult{
		MonthRemoved:                     monthRemoved,
		QuarterRemoved:                   quarterRemoved,
		RemainingEventCount:              m.EventCount,
		RemainingPF5bps:                  m.ProfitFactor,
		RemainingExpectancyBps:           m.ExpectancyBps,
		RemainingPositiveMonthCount:      phase103BPositiveMonthCount(monthly),
		RemainingSingleMonthContribution: phase103BTopContributionPctFromMonthly(monthly, 1),
		VerdictAfterRemoval:              "edge_positive",
	}
	if m.EventCount == 0 {
		result.VerdictAfterRemoval = "no_events"
	} else if m.ExpectancyBps <= 0 {
		result.VerdictAfterRemoval = "edge_destroyed"
	}
	return result
}

func phase103BLeaveOneOutResult(events []phase103BReturnEvent, monthRemoved, quarterRemoved string) Phase103BLeaveOneOutResult {
	m := phase103BMetricForReturns(events)
	monthly := phase103BMonthlyContributions(events)
	result := Phase103BLeaveOneOutResult{
		MonthRemoved:                     monthRemoved,
		QuarterRemoved:                   quarterRemoved,
		RemainingEventCount:              m.EventCount,
		RemainingPF5bps:                  m.ProfitFactor,
		RemainingExpectancyBps:           m.ExpectancyBps,
		RemainingPositiveMonthCount:      phase103BPositiveMonthCount(monthly),
		RemainingSingleMonthContribution: phase103BTopContributionPctFromMonthly(monthly, 1),
		VerdictAfterRemoval:              "edge_positive",
	}
	if m.EventCount == 0 {
		result.VerdictAfterRemoval = "no_events"
	} else if m.ExpectancyBps <= 0 {
		result.VerdictAfterRemoval = "edge_destroyed"
	}
	return result
}

func phase103BEventArtifactRisk(rows []Phase103BLeaveOneOutResult) string {
	for _, row := range rows {
		if row.VerdictAfterRemoval == "edge_destroyed" || row.VerdictAfterRemoval == "no_events" {
			return "high"
		}
	}
	return "low"
}

func phase103BQuarterFragile(rows []Phase103BLeaveOneOutResult) bool {
	for _, row := range rows {
		if row.RemainingExpectancyBps < 0 {
			return true
		}
	}
	return false
}

func phase103BBracketFailureDiagnosisForEvents(events []deepCandidateEvent, candles []protocol.Candle, side string) Phase103BBracketFailureDiagnosis {
	diag := Phase103BBracketFailureDiagnosis{}
	if len(events) == 0 || len(candles) == 0 {
		return diag
	}
	var tMFE, tMAE []float64
	var slBeforeProfit, minus5Profit60, minus10Profit240, sameCandle float64
	for _, event := range events {
		window := phase103BEventWindow(candles, event.CandleIndex, 240)
		mfeMin, maeMin := phase103BTimeToMFEAndMAE(window, side)
		tMFE = append(tMFE, mfeMin)
		tMAE = append(tMAE, maeMin)
		if phase103BTouchesAdverseBeforeLaterProfit(window, side, 10) {
			slBeforeProfit++
		}
		if phase103BProfitableAfterAdverseFirst(candles, event, side, 5, 60) {
			minus5Profit60++
		}
		if phase103BProfitableAfterAdverseFirst(candles, event, side, 10, 240) {
			minus10Profit240++
		}
		if phase103BSameCandleConflict(window, side, 20, 10) {
			sameCandle++
		}
	}
	sort.Float64s(tMFE)
	sort.Float64s(tMAE)
	n := math.Max(1, float64(len(events)))
	diag.MedianTimeToMFEMinutes = deepPercentileSorted(tMFE, 0.50)
	diag.MedianTimeToMAEMinutes = deepPercentileSorted(tMAE, 0.50)
	diag.PercentHittingSLBeforeLaterProfitable = slBeforeProfit / n * 100.0
	diag.PercentProfitable60mDespiteTouchingMinus5First = minus5Profit60 / n * 100.0
	diag.PercentProfitable240mDespiteTouchingMinus10First = minus10Profit240 / n * 100.0
	diag.SameCandleTPSLConflictPct = sameCandle / n * 100.0

	for _, cfg := range []struct {
		name string
		tp   float64
		sl   float64
	}{
		{"TP 5 bps / SL 5 bps", 5, 5},
		{"TP 10 bps / SL 5 bps", 10, 5},
		{"TP 15 bps / SL 7.5 bps", 15, 7.5},
		{"TP 20 bps / SL 10 bps", 20, 10},
		{"TP 30 bps / SL 15 bps", 30, 15},
		{"TP 50 bps / SL 25 bps", 50, 25},
	} {
		b := deepBracketMetric(events, candles, side, cfg.name, cfg.tp, cfg.sl)
		if b.NetExpectancyBpsAfter5bps > diag.BestBracketExpectancyBps || diag.BestBracketName == "" {
			diag.BestBracketName = b.Name
			diag.BestBracketPF5bps = b.PFAfter5bps
			diag.BestBracketExpectancyBps = b.NetExpectancyBpsAfter5bps
		}
	}
	diag.FailureModes = phase103BFailureModes(diag)
	return diag
}

func phase103BFailureModes(diag Phase103BBracketFailureDiagnosis) []string {
	var modes []string
	if diag.PercentProfitable60mDespiteTouchingMinus5First >= 10 || diag.MedianTimeToMAEMinutes <= diag.MedianTimeToMFEMinutes {
		modes = append(modes, "noisy_path")
	}
	if diag.MedianTimeToMFEMinutes > 60 || diag.PercentProfitable240mDespiteTouchingMinus10First >= 10 {
		modes = append(modes, "delayed_drift")
	}
	if diag.SameCandleTPSLConflictPct >= 5 {
		modes = append(modes, "same_candle_conservative_assumption")
	}
	if strings.Contains(diag.BestBracketName, "50 bps") || diag.BestBracketPF5bps < 1.0 {
		modes = append(modes, "insufficient_TP_SL_scale")
	}
	if len(modes) == 0 {
		modes = append(modes, "noisy_path")
	}
	return modes
}

func phase103BEventWindow(candles []protocol.Candle, startIdx, minutes int) []protocol.Candle {
	if startIdx < 0 || startIdx >= len(candles) {
		return nil
	}
	endIdx := startIdx + minutes
	if endIdx >= len(candles) {
		endIdx = len(candles) - 1
	}
	return candles[startIdx : endIdx+1]
}

func phase103BTimeToMFEAndMAE(candles []protocol.Candle, side string) (float64, float64) {
	if len(candles) == 0 || candles[0].Close == 0 {
		return 0, 0
	}
	entry := candles[0].Close
	bestFav := -math.MaxFloat64
	bestAdv := -math.MaxFloat64
	var mfeMin, maeMin int
	for i, candle := range candles {
		fav, adv := phase103BFavAdvBps(entry, candle, side)
		if fav > bestFav {
			bestFav = fav
			mfeMin = i
		}
		if adv > bestAdv {
			bestAdv = adv
			maeMin = i
		}
	}
	return float64(mfeMin), float64(maeMin)
}

func phase103BFavAdvBps(entry float64, candle protocol.Candle, side string) (float64, float64) {
	if strings.ToUpper(side) == "SHORT" {
		return (entry - candle.Low) / entry * 10000.0, (candle.High - entry) / entry * 10000.0
	}
	return (candle.High - entry) / entry * 10000.0, (entry - candle.Low) / entry * 10000.0
}

func phase103BTouchesAdverseBeforeLaterProfit(candles []protocol.Candle, side string, adverseBps float64) bool {
	if len(candles) == 0 || candles[0].Close == 0 {
		return false
	}
	entry := candles[0].Close
	touched := false
	for _, candle := range candles {
		_, adv := phase103BFavAdvBps(entry, candle, side)
		if adv >= adverseBps {
			touched = true
		}
		if touched && deepSignedReturnBps(entry, candle.Close, side) > 0 {
			return true
		}
	}
	return false
}

func phase103BProfitableAfterAdverseFirst(candles []protocol.Candle, event deepCandidateEvent, side string, adverseBps float64, horizonMinutes int) bool {
	window := phase103BEventWindow(candles, event.CandleIndex, horizonMinutes)
	if len(window) == 0 || window[0].Close == 0 {
		return false
	}
	entry := window[0].Close
	adverseIdx := -1
	for i, candle := range window {
		_, adv := phase103BFavAdvBps(entry, candle, side)
		if adv >= adverseBps {
			adverseIdx = i
			break
		}
	}
	if adverseIdx < 0 {
		return false
	}
	return deepSignedReturnBps(entry, window[len(window)-1].Close, side) > 0
}

func phase103BSameCandleConflict(candles []protocol.Candle, side string, tpBps, slBps float64) bool {
	if len(candles) == 0 || candles[0].Close == 0 {
		return false
	}
	entry := candles[0].Close
	for _, candle := range candles {
		fav, adv := phase103BFavAdvBps(entry, candle, side)
		if fav >= tpBps && adv >= slBps {
			return true
		}
	}
	return false
}

func buildPhase103BOOSCoverage(ctx context.Context, candlePath, market, interval string) (Phase103BOOSCoverageReport, error) {
	symbols := []string{"AVAXUSDT", "LINKUSDT", "SOLUSDT", "BTCUSDT", "ETHUSDT"}
	years := []int{2024, 2025}
	report := Phase103BOOSCoverageReport{
		Summary: Phase103BOOSCoverageSummary{
			CandlePath: candlePath,
			Market:     market,
			Interval:   interval,
			Symbols:    symbols,
			Years:      years,
			Full2024:   true,
			Full2025:   true,
		},
	}
	for _, year := range years {
		for _, symbol := range symbols {
			row := phase103BCoverageRow(ctx, candlePath, market, interval, symbol, year)
			if !row.HasFullCoverage {
				row.FetchCommand = phase103BFetchCommand(candlePath, market, interval, symbol, year)
				report.Summary.FetchCommands = append(report.Summary.FetchCommands, row.FetchCommand)
				report.Summary.BlockedByMissingData = true
				if year == 2024 {
					report.Summary.Full2024 = false
				}
				if year == 2025 {
					report.Summary.Full2025 = false
				}
			}
			report.Rows = append(report.Rows, row)
		}
	}
	report.Summary.FetchCommands = phase103BUniqueStrings(report.Summary.FetchCommands)
	return report, nil
}

func phase103BCoverageRow(ctx context.Context, candlePath, market, interval, symbol string, year int) Phase103BOOSCoverageRow {
	row := Phase103BOOSCoverageRow{Symbol: symbol, Year: year}
	stepMS, err := data.ParseIntervalToMS(interval)
	if err != nil {
		row.Error = err.Error()
		return row
	}
	src := data.NewLocalParquetSource()
	for month := time.January; month <= time.December; month++ {
		start := time.Date(year, month, 1, 0, 0, 0, 0, time.UTC)
		next := start.AddDate(0, 1, 0)
		end := next.Add(-time.Millisecond)
		expected := int(next.Sub(start).Milliseconds() / stepMS)
		row.ExpectedRows += expected
		candles, err := src.LoadCandles(ctx, data.CandleRequest{
			Path:     candlePath,
			Market:   market,
			Interval: interval,
			Symbol:   symbol,
			From:     start,
			To:       end,
		})
		monthKey := start.Format("2006-01")
		if err != nil {
			row.MissingMonths = append(row.MissingMonths, monthKey)
			row.MissingCandleCount += expected
			if row.Error == "" {
				row.Error = err.Error()
			}
			continue
		}
		row.RowCount += len(candles)
		if len(candles) != expected {
			row.MissingMonths = append(row.MissingMonths, monthKey)
			if expected > len(candles) {
				row.MissingCandleCount += expected - len(candles)
			}
		}
		if row.FirstCandleTime == "" && len(candles) > 0 {
			row.FirstCandleTime = time.UnixMilli(candles[0].OpenTimeMS).UTC().Format(time.RFC3339)
		}
		if len(candles) > 0 {
			row.LastCandleTime = time.UnixMilli(candles[len(candles)-1].OpenTimeMS).UTC().Format(time.RFC3339)
		}
	}
	row.HasFullCoverage = row.RowCount == row.ExpectedRows && row.MissingCandleCount == 0 && len(row.MissingMonths) == 0
	if row.HasFullCoverage {
		row.Error = ""
	}
	return row
}

func phase103BFetchCommand(candlePath, market, interval, symbol string, year int) string {
	return fmt.Sprintf("ak-historian fetch --market %s --symbols %s --interval %s --period monthly --start %d-01 --end %d-12 --workdir %s --keep", market, symbol, interval, year, year, candlePath)
}

func phase103BUniqueStrings(values []string) []string {
	seen := make(map[string]bool)
	var out []string
	for _, value := range values {
		if value == "" || seen[value] {
			continue
		}
		seen[value] = true
		out = append(out, value)
	}
	sort.Strings(out)
	return out
}

func phase103BExtendedOOSStatus(coverage Phase103BOOSCoverageReport, symbol string) string {
	for _, year := range []int{2024, 2025} {
		if phase103BHasFullOOSYear(coverage, symbol, year) {
			return fmt.Sprintf("full_%d_available", year)
		}
	}
	return "blocked_missing_data"
}

func phase103BHasFullOOSYear(coverage Phase103BOOSCoverageReport, symbol string, year int) bool {
	required := map[string]bool{symbol: false, "BTCUSDT": false, "ETHUSDT": false}
	for _, row := range coverage.Rows {
		if row.Year != year {
			continue
		}
		if _, ok := required[row.Symbol]; ok && row.HasFullCoverage {
			required[row.Symbol] = true
		}
	}
	for _, ok := range required {
		if !ok {
			return false
		}
	}
	return true
}

func buildPhase103BOptionalValidations(ctx context.Context, req Phase103BFragileRequest, coverage Phase103BOOSCoverageReport) []Phase103BOptionalValidation {
	var out []Phase103BOptionalValidation
	for _, target := range phase103BFragileTargets {
		for _, year := range []int{2024, 2025} {
			item := Phase103BOptionalValidation{
				Symbol: target.Symbol,
				Family: target.Family,
				Side:   target.Side,
				Year:   year,
			}
			if !phase103BHasFullOOSYear(coverage, target.Symbol, year) {
				item.Status = "blocked_missing_data"
				item.Reason = "full target+BTCUSDT+ETHUSDT local coverage not available"
				out = append(out, item)
				continue
			}
			ctxYear, err := loadPhase103BLeadContext(ctx, req.CandlePath, req.Market, req.Interval, target, year)
			if err != nil {
				item.Status = "blocked_evaluation_error"
				item.Reason = err.Error()
				out = append(out, item)
				continue
			}
			lead := analyzePhase103BLead(ctxYear, target, LeaderboardRow{}, coverage)
			item.Status = "evaluated"
			item.FYPF5bps = lead.FYPF5bps
			item.FYExpectancyBps = lead.FYExpectancy5bpsBps
			item.FinalStatus = lead.FinalStatus
			base := filepath.Join(filepath.Dir(req.OutMarkdown), fmt.Sprintf("phase10_3b_%s_%s_%s_%d", target.Symbol, target.Family, target.Side, year))
			item.OutputMarkdown = base + ".md"
			item.OutputJSON = base + ".json"
			_ = writeJSONFile(item.OutputJSON, lead)
			_ = os.WriteFile(item.OutputMarkdown, []byte(buildPhase103BLeadMD(lead)), 0644)
			out = append(out, item)
		}
	}
	return out
}

func phase103BFinalStatus(lead Phase103BLeadAnalysis) (string, []string) {
	var reasons []string
	if lead.EventCount == 0 {
		return phase103BStatusRejected, []string{"event_count == 0"}
	}
	if lead.LeakageStatus != "PASS" {
		return phase103BStatusRejected, []string{"leakage status is not PASS"}
	}
	if lead.FYExpectancy5bpsBps <= 0 || lead.FYPF5bps < 1.0 || lead.H2Expectancy5bpsBps <= 0 || lead.H2PF5bps < 1.0 {
		return phase103BStatusRejected, []string{"core FY/H2 edge is non-positive after 5 bps"}
	}
	if lead.Top1MonthContributionPct > 50 {
		reasons = append(reasons, "top 1 month contribution > 50%")
	}
	if lead.Top2MonthContributionPct > 70 {
		reasons = append(reasons, "top 2 month contribution > 70%")
	}
	if !phase103BAllLeaveOneOutPositive(lead.LeaveOneMonthOut) {
		reasons = append(reasons, "leave-one-month-out does not remain positive for all months")
	}
	if !phase103BAllLeaveOneOutPositive(lead.LeaveOneQuarterOut) {
		reasons = append(reasons, "leave-one-quarter-out does not remain positive for all quarters")
	}
	if lead.WorstQuarterPF5bps < 0.95 {
		reasons = append(reasons, "worst-quarter PF after 5 bps < 0.95")
	}
	if lead.H2PF5bps < 1.10 {
		reasons = append(reasons, "H2 PF after 5 bps < 1.10")
	}
	if lead.FYPF5bps < 1.05 {
		reasons = append(reasons, "FY PF after 5 bps < 1.05")
	}
	if len(reasons) > 0 {
		return phase103BStatusFragileEventArtifact, reasons
	}
	if lead.ExtendedOOSStatus == "blocked_missing_data" {
		return phase103BStatusFragileNeedsOOS, []string{"passes 2023 robustness gates but extended OOS data missing"}
	}
	return phase103BStatusValidatedResearchLead, []string{"passes Phase 10.3B robustness gates"}
}

func phase103BAllLeaveOneOutPositive(rows []Phase103BLeaveOneOutResult) bool {
	if len(rows) == 0 {
		return false
	}
	for _, row := range rows {
		if row.RemainingEventCount == 0 || row.RemainingExpectancyBps <= 0 {
			return false
		}
	}
	return true
}

func phase103BFinalRecommendation(leads []Phase103BLeadAnalysis) string {
	hasValidated := false
	hasNeedsOOS := false
	for _, lead := range leads {
		if lead.FinalStatus == phase103BStatusValidatedResearchLead {
			hasValidated = true
		}
		if lead.FinalStatus == phase103BStatusFragileNeedsOOS {
			hasNeedsOOS = true
		}
	}
	if hasValidated {
		return "one or more validated research leads worth Phase 10.4 strategy-shape research"
	}
	if hasNeedsOOS {
		return "one or more fragile leads need extended OOS"
	}
	return "no validated leads"
}

func phase103BYearBounds(year int) (time.Time, time.Time) {
	start := time.Date(year, time.January, 1, 0, 0, 0, 0, time.UTC)
	return start, start.AddDate(1, 0, 0).Add(-time.Millisecond)
}

type phase103BPeriodBounds struct {
	Name  string
	Start time.Time
	End   time.Time
}

func phase103BQuarterBounds(year int) []phase103BPeriodBounds {
	return []phase103BPeriodBounds{
		{
			Name:  "Q1",
			Start: time.Date(year, time.January, 1, 0, 0, 0, 0, time.UTC),
			End:   time.Date(year, time.April, 1, 0, 0, 0, 0, time.UTC).Add(-time.Millisecond),
		},
		{
			Name:  "Q2",
			Start: time.Date(year, time.April, 1, 0, 0, 0, 0, time.UTC),
			End:   time.Date(year, time.July, 1, 0, 0, 0, 0, time.UTC).Add(-time.Millisecond),
		},
		{
			Name:  "Q3",
			Start: time.Date(year, time.July, 1, 0, 0, 0, 0, time.UTC),
			End:   time.Date(year, time.October, 1, 0, 0, 0, 0, time.UTC).Add(-time.Millisecond),
		},
		{
			Name:  "Q4",
			Start: time.Date(year, time.October, 1, 0, 0, 0, 0, time.UTC),
			End:   time.Date(year+1, time.January, 1, 0, 0, 0, 0, time.UTC).Add(-time.Millisecond),
		},
	}
}

func phase103BQuarter(t time.Time) string {
	switch t.Month() {
	case time.January, time.February, time.March:
		return "Q1"
	case time.April, time.May, time.June:
		return "Q2"
	case time.July, time.August, time.September:
		return "Q3"
	default:
		return "Q4"
	}
}

func buildPhase103BFragileMD(report Phase103BFragileLeadsReport) string {
	var sb strings.Builder
	sb.WriteString("# Phase 10.3B Fragile Lead Robustness Extension\n\n")
	sb.WriteString("## Summary\n")
	sb.WriteString(fmt.Sprintf("- Source leaderboard: `%s`\n", report.Summary.SourceLeaderboard))
	sb.WriteString(fmt.Sprintf("- Candle path: `%s`\n", report.Summary.CandlePath))
	sb.WriteString(fmt.Sprintf("- Market: `%s`\n", report.Summary.Market))
	sb.WriteString(fmt.Sprintf("- Interval: `%s`\n", report.Summary.Interval))
	sb.WriteString(fmt.Sprintf("- Final recommendation: %s\n\n", report.Summary.FinalRecommendation))

	sb.WriteString("## Final Verdicts\n")
	sb.WriteString("| Symbol | Family | Side | Final Status | Classification | Top 1 Month % | Top 2 Month % | Top 3 Month % | Largest Cluster % | Largest 5 Clusters % | Event Artifact Risk | Quarter Fragile | Extended OOS |\n")
	sb.WriteString("|---|---|---|---|---|---:|---:|---:|---:|---:|---|---:|---|\n")
	for _, lead := range report.Leads {
		sb.WriteString(fmt.Sprintf("| %s | %s | %s | %s | %s | %.2f | %.2f | %.2f | %.2f | %.2f | %s | %t | %s |\n",
			lead.Symbol, lead.Family, lead.Side, lead.FinalStatus, lead.ConcentrationClassification, lead.Top1MonthContributionPct, lead.Top2MonthContributionPct, lead.Top3MonthContributionPct, lead.LargestClusterContributionPct, lead.Largest5ClustersContributionPct, lead.EventArtifactRisk, lead.QuarterFragile, lead.ExtendedOOSStatus))
	}
	sb.WriteString("\n")
	for _, lead := range report.Leads {
		sb.WriteString(buildPhase103BLeadMD(lead))
	}
	sb.WriteString("## Optional Extended OOS Validations\n")
	sb.WriteString("| Symbol | Family | Side | Year | Status | Reason | Output |\n")
	sb.WriteString("|---|---|---|---:|---|---|---|\n")
	for _, item := range report.OptionalValidations {
		out := item.OutputMarkdown
		if out == "" {
			out = "-"
		}
		reason := item.Reason
		if reason == "" {
			reason = "-"
		}
		sb.WriteString(fmt.Sprintf("| %s | %s | %s | %d | %s | %s | %s |\n", item.Symbol, item.Family, item.Side, item.Year, item.Status, reason, out))
	}
	sb.WriteString("\n")
	return sb.String()
}

func buildPhase103BLeadMD(lead Phase103BLeadAnalysis) string {
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("## %s %s_%s\n", lead.Symbol, lead.Family, lead.Side))
	sb.WriteString(fmt.Sprintf("- Final status: `%s`\n", lead.FinalStatus))
	sb.WriteString(fmt.Sprintf("- Reasons: %s\n", phase103BJoinOrDash(lead.FinalStatusReasons)))
	sb.WriteString(fmt.Sprintf("- FY PF after 5 bps: %.4f\n", lead.FYPF5bps))
	sb.WriteString(fmt.Sprintf("- H2 PF after 5 bps: %.4f\n", lead.H2PF5bps))
	sb.WriteString(fmt.Sprintf("- Worst quarter PF after 5 bps: %.4f\n", lead.WorstQuarterPF5bps))
	sb.WriteString(fmt.Sprintf("- FY expectancy after 5 bps: %.4f bps\n", lead.FYExpectancy5bpsBps))
	sb.WriteString(fmt.Sprintf("- H2 expectancy after 5 bps: %.4f bps\n\n", lead.H2Expectancy5bpsBps))

	sb.WriteString("### Monthly Contribution\n")
	sb.WriteString("| Month | Events | Net 5bps (bps) | PF 5bps | Expectancy 5bps (bps) | Contribution % |\n")
	sb.WriteString("|---|---:|---:|---:|---:|---:|\n")
	for _, m := range lead.MonthlyContribution {
		sb.WriteString(fmt.Sprintf("| %s | %d | %.4f | %.4f | %.4f | %.2f |\n", m.Month, m.EventCount, m.NetContribution5bpsBps, m.ProfitFactor5bps, m.Expectancy5bpsBps, m.ContributionPctOfNetEdge))
	}
	sb.WriteString("\n")

	sb.WriteString("### Leave-One-Month-Out\n")
	sb.WriteString("| Month Removed | Remaining Events | PF 5bps | Expectancy (bps) | Positive Months | Single Month % | Verdict |\n")
	sb.WriteString("|---|---:|---:|---:|---:|---:|---|\n")
	for _, row := range lead.LeaveOneMonthOut {
		sb.WriteString(fmt.Sprintf("| %s | %d | %.4f | %.4f | %d | %.2f | %s |\n", row.MonthRemoved, row.RemainingEventCount, row.RemainingPF5bps, row.RemainingExpectancyBps, row.RemainingPositiveMonthCount, row.RemainingSingleMonthContribution, row.VerdictAfterRemoval))
	}
	sb.WriteString("\n")

	sb.WriteString("### Leave-One-Quarter-Out\n")
	sb.WriteString("| Quarter Removed | Remaining Events | PF 5bps | Expectancy (bps) | Positive Months | Single Month % | Verdict |\n")
	sb.WriteString("|---|---:|---:|---:|---:|---:|---|\n")
	for _, row := range lead.LeaveOneQuarterOut {
		sb.WriteString(fmt.Sprintf("| %s | %d | %.4f | %.4f | %d | %.2f | %s |\n", row.QuarterRemoved, row.RemainingEventCount, row.RemainingPF5bps, row.RemainingExpectancyBps, row.RemainingPositiveMonthCount, row.RemainingSingleMonthContribution, row.VerdictAfterRemoval))
	}
	sb.WriteString("\n")

	sb.WriteString("### Event Clusters\n")
	sb.WriteString(fmt.Sprintf("- Event cluster count: %d\n", lead.EventClusterCount))
	sb.WriteString(fmt.Sprintf("- Largest cluster contribution: %.2f%%\n", lead.LargestClusterContributionPct))
	sb.WriteString(fmt.Sprintf("- Largest 5 clusters contribution: %.2f%%\n\n", lead.Largest5ClustersContributionPct))
	sb.WriteString("| Cluster | Start | End | Events | Net 5bps (bps) | Contribution % |\n")
	sb.WriteString("|---:|---|---|---:|---:|---:|\n")
	for _, c := range lead.TopClusters {
		sb.WriteString(fmt.Sprintf("| %d | %s | %s | %d | %.4f | %.2f |\n", c.ClusterID, c.StartTime, c.EndTime, c.EventCount, c.NetContribution5bpsBps, c.ContributionPctOfNetEdge))
	}
	sb.WriteString("\n")

	d := lead.BracketFailureDiagnosis
	sb.WriteString("### Bracket Failure Diagnosis\n")
	sb.WriteString(fmt.Sprintf("- Median time to MFE: %.2f minutes\n", d.MedianTimeToMFEMinutes))
	sb.WriteString(fmt.Sprintf("- Median time to MAE: %.2f minutes\n", d.MedianTimeToMAEMinutes))
	sb.WriteString(fmt.Sprintf("- Percent hitting SL before later becoming profitable: %.2f%%\n", d.PercentHittingSLBeforeLaterProfitable))
	sb.WriteString(fmt.Sprintf("- Percent profitable at 60m despite touching -5 bps first: %.2f%%\n", d.PercentProfitable60mDespiteTouchingMinus5First))
	sb.WriteString(fmt.Sprintf("- Percent profitable at 240m despite touching -10 bps first: %.2f%%\n", d.PercentProfitable240mDespiteTouchingMinus10First))
	sb.WriteString(fmt.Sprintf("- Same-candle TP/SL conflict: %.2f%%\n", d.SameCandleTPSLConflictPct))
	sb.WriteString(fmt.Sprintf("- Best bracket: `%s`, PF %.4f, expectancy %.4f bps\n", d.BestBracketName, d.BestBracketPF5bps, d.BestBracketExpectancyBps))
	sb.WriteString(fmt.Sprintf("- Failure modes: %s\n\n", phase103BJoinOrDash(d.FailureModes)))
	return sb.String()
}

func buildPhase103BOOSCoverageMD(report Phase103BOOSCoverageReport) string {
	var sb strings.Builder
	sb.WriteString("# Phase 10.3B Extended OOS Coverage\n\n")
	sb.WriteString("## Summary\n")
	sb.WriteString(fmt.Sprintf("- Candle path: `%s`\n", report.Summary.CandlePath))
	sb.WriteString(fmt.Sprintf("- Market: `%s`\n", report.Summary.Market))
	sb.WriteString(fmt.Sprintf("- Interval: `%s`\n", report.Summary.Interval))
	sb.WriteString(fmt.Sprintf("- 2024 coverage status: `%t`\n", report.Summary.Full2024))
	sb.WriteString(fmt.Sprintf("- 2025 coverage status: `%t`\n", report.Summary.Full2025))
	sb.WriteString(fmt.Sprintf("- Blocked by missing data: `%t`\n\n", report.Summary.BlockedByMissingData))
	sb.WriteString("| Symbol | Year | Full Coverage | Rows | Expected | Missing Candles | Missing Months | Fetch Command |\n")
	sb.WriteString("|---|---:|---:|---:|---:|---:|---|---|\n")
	for _, row := range report.Rows {
		missing := strings.Join(row.MissingMonths, ", ")
		if missing == "" {
			missing = "-"
		}
		fetch := row.FetchCommand
		if fetch == "" {
			fetch = "-"
		}
		sb.WriteString(fmt.Sprintf("| %s | %d | %t | %d | %d | %d | %s | `%s` |\n", row.Symbol, row.Year, row.HasFullCoverage, row.RowCount, row.ExpectedRows, row.MissingCandleCount, missing, fetch))
	}
	if len(report.Summary.FetchCommands) > 0 {
		sb.WriteString("\n## Fetch Commands\n")
		for _, cmd := range report.Summary.FetchCommands {
			sb.WriteString(fmt.Sprintf("- `%s`\n", cmd))
		}
	}
	sb.WriteString("\n")
	return sb.String()
}

func phase103BJoinOrDash(values []string) string {
	if len(values) == 0 {
		return "-"
	}
	return strings.Join(values, "; ")
}
