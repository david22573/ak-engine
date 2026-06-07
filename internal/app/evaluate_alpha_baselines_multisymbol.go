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
	"github.com/davidmiguel22573/ak-engine/internal/research"
	"github.com/spf13/cobra"
)

var (
	eabmPath     string
	eabmSymbols  string
	eabmMarket   string
	eabmInterval string
	eabmFrom     string
	eabmTo       string
	eabmOut      string
)

const (
	phase102CYear             = "2023"
	expectancyUnitBps         = "basis_points"
	verdictRejected           = "rejected"
	verdictFragile            = "fragile"
	verdictInconclusive       = "inconclusive"
	verdictResearchLead       = "research_lead"
	verdictMissingData        = "missing_data"
	verdictUnsupportedContext = "unsupported_context"
)

var requiredFamilies = []string{
	"TrendContinuation_LONG",
	"TrendContinuation_SHORT",
	"VolumeMomentum_LONG",
	"VolumeMomentum_SHORT",
	"ShockFade_LONG",
	"ShockFade_SHORT",
	"CompressionBreakout_LONG",
	"CompressionBreakout_SHORT",
	"BetaAgrees_LONG",
	"BetaAgrees_SHORT",
	"BetaDiverges_LONG",
	"BetaDiverges_SHORT",
}

var verdictOrder = []string{
	verdictRejected,
	verdictFragile,
	verdictInconclusive,
	verdictResearchLead,
	verdictMissingData,
	verdictUnsupportedContext,
}

var focusedFragileCandidates = []string{
	"LINKUSDT_CompressionBreakout_SHORT",
	"LINKUSDT_ShockFade_LONG",
	"LINKUSDT_BetaDiverges_SHORT",
}

type ArtifactAuditSummary struct {
	Market                            string   `json:"market"`
	Interval                          string   `json:"interval"`
	From                              string   `json:"from"`
	To                                string   `json:"to"`
	Symbols                           []string `json:"symbols"`
	UnsupportedContextSymbolsExcluded []string `json:"unsupported_context_symbols_excluded,omitempty"`
	CandlesAvailableCount             int      `json:"candles_available_count"`
	MissingFeaturesBefore             int      `json:"missing_features_before"`
	MissingRegimesBefore              int      `json:"missing_regimes_before"`
	BuildNeededCount                  int      `json:"build_needed_count"`
}

type ArtifactAuditRow struct {
	Symbol           string `json:"symbol"`
	CandlesAvailable bool   `json:"candles_available"`
	CandleRows       int    `json:"candle_rows"`
	CandleError      string `json:"candle_error,omitempty"`
	FeaturesExist    bool   `json:"features_exist"`
	RegimesExist     bool   `json:"regimes_exist"`
	FeatureRows      int    `json:"feature_rows"`
	RegimeRows       int    `json:"regime_rows"`
	FeaturePath      string `json:"feature_path"`
	RegimePath       string `json:"regime_path"`
	FeatureError     string `json:"feature_error,omitempty"`
	RegimeError      string `json:"regime_error,omitempty"`
	BuildNeeded      bool   `json:"build_needed"`
	ReasonIfMissing  string `json:"reason_if_missing"`
}

type ArtifactAuditReport struct {
	Summary ArtifactAuditSummary `json:"summary"`
	Rows    []ArtifactAuditRow   `json:"rows"`
}

type LeaderboardSummary struct {
	Market           string   `json:"market"`
	Interval         string   `json:"interval"`
	From             string   `json:"from"`
	To               string   `json:"to"`
	Symbols          []string `json:"symbols"`
	FamiliesRequired int      `json:"families_required"`
	ExpectancyUnit   string   `json:"expectancy_unit"`
	CostUnit         string   `json:"cost_unit"`
}

type LeaderboardRow struct {
	Symbol                     string   `json:"symbol"`
	Family                     string   `json:"family"`
	Side                       string   `json:"side"`
	TimeSplit                  string   `json:"time_split"`
	EventCount                 int      `json:"event_count"`
	H1PF5bps                   float64  `json:"h1_pf_5bps"`
	H2PF5bps                   float64  `json:"h2_pf_5bps"`
	FYPF5bps                   float64  `json:"fy_pf_5bps"`
	H2Expectancy5bpsBps        float64  `json:"h2_expectancy_5bps_bps"`
	FYExpectancy5bpsBps        float64  `json:"fy_expectancy_5bps_bps"`
	PositiveMonthCount         int      `json:"positive_month_count"`
	EntryDelay1cExpectancyBps  float64  `json:"entry_delay_1c_expectancy_bps"`
	BestQuarter                string   `json:"best_quarter"`
	WorstQuarter               string   `json:"worst_quarter"`
	WorstQuarterPF5bps         float64  `json:"worst_quarter_pf_5bps"`
	SingleMonthContributionPct float64  `json:"single_month_contribution_pct"`
	Top2MonthContributionPct   float64  `json:"top_2_month_contribution_pct"`
	Cost10bpsPF                float64  `json:"cost_10bps_pf"`
	Cost10bpsExpectancyBps     float64  `json:"cost_10bps_expectancy_bps"`
	LeakageStatus              string   `json:"leakage_status"`
	Verdict                    string   `json:"verdict"`
	FailedGates                []string `json:"failed_gates"`
	h2PositiveMonthCount       int
}

type LeaderboardJSON struct {
	Summary       LeaderboardSummary `json:"summary"`
	VerdictCounts map[string]int     `json:"verdict_counts"`
	Rows          []LeaderboardRow   `json:"rows"`
}

type CandidateMetric struct {
	EventCount    int     `json:"event_count"`
	ProfitFactor  float64 `json:"profit_factor"`
	ExpectancyBps float64 `json:"expectancy_bps"`
	WinRate       float64 `json:"win_rate"`
}

type SplitMetrics struct {
	EventCount             int     `json:"event_count"`
	ProfitFactor5bps       float64 `json:"profit_factor_5bps"`
	Expectancy5bpsBps      float64 `json:"expectancy_5bps_bps"`
	WinRate5bps            float64 `json:"win_rate_5bps"`
	NetContribution5bpsBps float64 `json:"net_contribution_5bps_bps"`
}

type CostHaircutMetric struct {
	CostBps       int     `json:"cost_bps"`
	EventCount    int     `json:"event_count"`
	ProfitFactor  float64 `json:"profit_factor"`
	ExpectancyBps float64 `json:"expectancy_bps"`
	WinRate       float64 `json:"win_rate"`
}

type EntryDelayMetric struct {
	DelayCandles  int     `json:"delay_candles"`
	CostBps       int     `json:"cost_bps"`
	EventCount    int     `json:"event_count"`
	ProfitFactor  float64 `json:"profit_factor"`
	ExpectancyBps float64 `json:"expectancy_bps"`
	WinRate       float64 `json:"win_rate"`
}

type MonthlyContribution struct {
	Month                    string  `json:"month"`
	EventCount               int     `json:"event_count"`
	ProfitFactor5bps         float64 `json:"profit_factor_5bps"`
	Expectancy5bpsBps        float64 `json:"expectancy_5bps_bps"`
	NetContribution5bpsBps   float64 `json:"net_contribution_5bps_bps"`
	ContributionPctOfNetGain float64 `json:"contribution_pct_of_net_gain"`
}

type RegimeBreakdownMetric struct {
	Regime             string  `json:"regime"`
	EventCount         int     `json:"event_count"`
	ProfitFactor5bps   float64 `json:"profit_factor_5bps"`
	Expectancy5bpsBps  float64 `json:"expectancy_5bps_bps"`
	WinRate5bps        float64 `json:"win_rate_5bps"`
	NetContributionBps float64 `json:"net_contribution_5bps_bps"`
}

type GateResult struct {
	Name      string `json:"name"`
	Passed    bool   `json:"passed"`
	Actual    string `json:"actual"`
	Threshold string `json:"threshold"`
}

type CandidateVerdictInput struct {
	H2Expectancy5bpsBps        float64
	H2PF5bps                   float64
	FYPF5bps                   float64
	EventCount                 int
	H1EventCount               int
	H2EventCount               int
	PositiveMonthCount         int
	EntryDelay1cExpectancyBps  float64
	SingleMonthContributionPct float64
	Top2MonthContributionPct   float64
	WorstQuarterPF5bps         float64
	Cost10bpsReported          bool
	LeakageStatus              string
}

type FragileCandidateReport struct {
	Symbol                     string                  `json:"symbol"`
	Family                     string                  `json:"family"`
	Side                       string                  `json:"side"`
	Verdict                    string                  `json:"verdict"`
	FailedGates                []string                `json:"failed_gates"`
	PassedGates                []string                `json:"passed_gates"`
	Gates                      []GateResult            `json:"gates"`
	ExpectancyUnit             string                  `json:"expectancy_unit"`
	LeakageStatus              string                  `json:"leakage_status"`
	EventCount                 int                     `json:"event_count"`
	Splits                     map[string]SplitMetrics `json:"splits"`
	CostHaircuts               []CostHaircutMetric     `json:"cost_haircuts"`
	EntryDelays                []EntryDelayMetric      `json:"entry_delays"`
	MonthlyContribution        []MonthlyContribution   `json:"monthly_contribution"`
	SingleMonthContributionPct float64                 `json:"single_month_contribution_pct"`
	Top2MonthContributionPct   float64                 `json:"top_2_month_contribution_pct"`
	AvgMFEBps                  float64                 `json:"avg_mfe_bps"`
	AvgMAEBps                  float64                 `json:"avg_mae_bps"`
	RegimeBreakdown            []RegimeBreakdownMetric `json:"regime_breakdown"`
}

type phaseEvent struct {
	Index       int
	Family      string
	Side        string
	EventTimeMS int64
	Label       regime.Label
	MFE         float64
	MAE         float64
}

var evaluateAlphaBaselinesMultisymbolCmd = &cobra.Command{
	Use:   "evaluate-alpha-baselines-multisymbol",
	Short: "Evaluate deterministic multisymbol alpha baselines into Phase 10.2C reports",
	RunE: func(cmd *cobra.Command, args []string) error {
		if eabmPath == "" {
			return errors.New("missing --path")
		}
		if eabmSymbols == "" {
			return errors.New("missing --symbols")
		}
		if eabmMarket == "" {
			return errors.New("missing --market")
		}
		if eabmInterval == "" {
			return errors.New("missing --interval")
		}
		if eabmFrom == "" || eabmTo == "" {
			return errors.New("missing --from or --to")
		}
		if eabmOut == "" {
			return errors.New("missing --out")
		}

		symbols := parseSymbolList(eabmSymbols)
		fromTime, err := parseFromTime(eabmFrom)
		if err != nil {
			return fmt.Errorf("invalid from time: %w", err)
		}
		toTime, err := parseToTime(eabmTo)
		if err != nil {
			return fmt.Errorf("invalid to time: %w", err)
		}

		mdOut, jsonOut := normalizeMDAndJSONPaths(eabmOut)
		reportDir := filepath.Dir(mdOut)

		auditSymbols, excludedUnsupported := artifactAuditSymbols(symbols)
		audit, err := buildPhase102CArtifactAudit(cmd.Context(), eabmPath, eabmMarket, eabmInterval, eabmFrom, eabmTo, fromTime, toTime, auditSymbols, excludedUnsupported)
		if err != nil {
			return err
		}
		if err := writePhase102CArtifactAudit(reportDir, audit); err != nil {
			return err
		}

		leaderboard, fragileReports, err := evaluatePhase102CLeaderboard(cmd.Context(), eabmPath, eabmMarket, eabmInterval, eabmFrom, eabmTo, fromTime, toTime, symbols, audit)
		if err != nil {
			return err
		}

		if err := writeJSONFile(jsonOut, leaderboard); err != nil {
			return err
		}
		if err := os.WriteFile(mdOut, []byte(buildMultisymbolAlphaBaselinesMD(leaderboard, audit)), 0644); err != nil {
			return fmt.Errorf("write multisymbol report: %w", err)
		}

		candidateMD := filepath.Join(reportDir, "phase10_2c_candidate_leaderboard.md")
		candidateJSON := filepath.Join(reportDir, "phase10_2c_candidate_leaderboard.json")
		if err := writeJSONFile(candidateJSON, leaderboard); err != nil {
			return err
		}
		if err := os.WriteFile(candidateMD, []byte(buildCandidateLeaderboardMD(leaderboard)), 0644); err != nil {
			return fmt.Errorf("write candidate leaderboard: %w", err)
		}

		for _, key := range focusedFragileCandidates {
			report, ok := fragileReports[key]
			if !ok {
				return fmt.Errorf("focused fragile candidate %s was not evaluated", key)
			}
			base := filepath.Join(reportDir, "phase10_2c_fragile_"+key)
			if err := writeJSONFile(base+".json", report); err != nil {
				return err
			}
			if err := os.WriteFile(base+".md", []byte(buildFragileCandidateMD(report)), 0644); err != nil {
				return fmt.Errorf("write fragile report %s: %w", key, err)
			}
		}

		fmt.Printf("Phase 10.2C multisymbol report written to %s\n", mdOut)
		fmt.Printf("Phase 10.2C candidate leaderboard written to %s\n", candidateMD)
		return nil
	},
}

func init() {
	evaluateAlphaBaselinesMultisymbolCmd.Flags().StringVar(&eabmPath, "path", "", "Path to local parquet historian workdir")
	evaluateAlphaBaselinesMultisymbolCmd.Flags().StringVar(&eabmSymbols, "symbols", "", "Comma-separated symbols")
	evaluateAlphaBaselinesMultisymbolCmd.Flags().StringVar(&eabmMarket, "market", "", "Market")
	evaluateAlphaBaselinesMultisymbolCmd.Flags().StringVar(&eabmInterval, "interval", "", "Interval")
	evaluateAlphaBaselinesMultisymbolCmd.Flags().StringVar(&eabmFrom, "from", "", "From date")
	evaluateAlphaBaselinesMultisymbolCmd.Flags().StringVar(&eabmTo, "to", "", "To date")
	evaluateAlphaBaselinesMultisymbolCmd.Flags().StringVar(&eabmOut, "out", "", "Output markdown path")
	rootCmd.AddCommand(evaluateAlphaBaselinesMultisymbolCmd)
}

func parseSymbolList(raw string) []string {
	parts := strings.Split(raw, ",")
	symbols := make([]string, 0, len(parts))
	for _, p := range parts {
		sym := strings.TrimSpace(p)
		if sym != "" {
			symbols = append(symbols, sym)
		}
	}
	return symbols
}

func artifactAuditSymbols(symbols []string) ([]string, []string) {
	var audit []string
	var excluded []string
	for _, sym := range symbols {
		if sym == "BTCUSDT" {
			excluded = append(excluded, sym)
			continue
		}
		audit = append(audit, sym)
	}
	return audit, excluded
}

func normalizeMDAndJSONPaths(path string) (string, string) {
	if strings.HasSuffix(path, ".md") {
		return path, strings.TrimSuffix(path, ".md") + ".json"
	}
	return path + ".md", path + ".json"
}

func phaseFeaturePath(symbol string) string {
	return filepath.Join("runs", "features", fmt.Sprintf("%s-%s-FY-context.json", symbol, phase102CYear))
}

func phaseRegimePath(symbol string) string {
	return filepath.Join("runs", "regimes", fmt.Sprintf("%s-%s-FY-context.json", symbol, phase102CYear))
}

func discoverPhaseArtifactPath(kind, symbol string) (string, bool) {
	var stable string
	var dir string
	switch kind {
	case "features":
		stable = phaseFeaturePath(symbol)
		dir = filepath.Join("runs", "features")
	case "regimes":
		stable = phaseRegimePath(symbol)
		dir = filepath.Join("runs", "regimes")
	default:
		return "", false
	}
	if _, err := os.Stat(stable); err == nil {
		return stable, true
	}

	matches, _ := filepath.Glob(filepath.Join(dir, fmt.Sprintf("%s-%s-FY*.json", symbol, phase102CYear)))
	sort.Strings(matches)
	for _, match := range matches {
		if strings.Contains(filepath.Base(match), "context") {
			return match, true
		}
	}
	if len(matches) > 0 {
		return matches[0], true
	}
	return stable, false
}

func buildPhase102CArtifactAudit(ctx context.Context, path, market, interval, from, to string, fromTime, toTime time.Time, symbols, excludedUnsupported []string) (ArtifactAuditReport, error) {
	src := data.NewLocalParquetSource()
	report := ArtifactAuditReport{
		Summary: ArtifactAuditSummary{
			Market:                            market,
			Interval:                          interval,
			From:                              from,
			To:                                to,
			Symbols:                           symbols,
			UnsupportedContextSymbolsExcluded: excludedUnsupported,
		},
	}

	for _, sym := range symbols {
		row := ArtifactAuditRow{Symbol: sym}

		candles, err := src.LoadCandles(ctx, data.CandleRequest{
			Source:   "local-parquet",
			Path:     path,
			Market:   market,
			Symbol:   sym,
			Interval: interval,
			From:     fromTime,
			To:       toTime,
		})
		if err != nil {
			row.CandleError = err.Error()
		} else {
			row.CandlesAvailable = len(candles) > 0
			row.CandleRows = len(candles)
			if row.CandlesAvailable {
				report.Summary.CandlesAvailableCount++
			}
		}

		featurePath, featuresExist := discoverPhaseArtifactPath("features", sym)
		row.FeaturePath = featurePath
		row.FeaturesExist = featuresExist
		if !featuresExist {
			report.Summary.MissingFeaturesBefore++
		} else {
			rows, err := features.ReadRowsJSON(featurePath)
			if err != nil {
				row.FeatureError = err.Error()
			} else {
				row.FeatureRows = len(rows)
			}
		}

		regimePath, regimesExist := discoverPhaseArtifactPath("regimes", sym)
		row.RegimePath = regimePath
		row.RegimesExist = regimesExist
		if !regimesExist {
			report.Summary.MissingRegimesBefore++
		} else {
			labels, err := regime.ReadLabelsJSON(regimePath)
			if err != nil {
				row.RegimeError = err.Error()
			} else {
				row.RegimeRows = len(labels)
			}
		}

		row = finalizeArtifactAuditRow(row)
		if row.BuildNeeded {
			report.Summary.BuildNeededCount++
		}

		report.Rows = append(report.Rows, row)
	}

	return report, nil
}

func finalizeArtifactAuditRow(row ArtifactAuditRow) ArtifactAuditRow {
	row.BuildNeeded = row.CandlesAvailable && (!row.FeaturesExist || !row.RegimesExist || row.FeatureError != "" || row.RegimeError != "")
	row.ReasonIfMissing = artifactMissingReason(row)
	return row
}

func artifactMissingReason(row ArtifactAuditRow) string {
	if !row.CandlesAvailable {
		if row.CandleError != "" {
			return "missing candle coverage: " + row.CandleError
		}
		return "missing candle coverage"
	}
	var reasons []string
	if !row.FeaturesExist {
		reasons = append(reasons, "feature artifact missing")
	}
	if !row.RegimesExist {
		reasons = append(reasons, "regime artifact missing")
	}
	if row.FeatureError != "" {
		reasons = append(reasons, "feature artifact unreadable: "+row.FeatureError)
	}
	if row.RegimeError != "" {
		reasons = append(reasons, "regime artifact unreadable: "+row.RegimeError)
	}
	return strings.Join(reasons, "; ")
}

func writePhase102CArtifactAudit(reportDir string, report ArtifactAuditReport) error {
	jsonPath := filepath.Join(reportDir, "phase10_2c_artifact_audit.json")
	mdPath := filepath.Join(reportDir, "phase10_2c_artifact_audit.md")
	if err := writeJSONFile(jsonPath, report); err != nil {
		return err
	}
	if err := os.WriteFile(mdPath, []byte(buildArtifactAuditMD(report)), 0644); err != nil {
		return fmt.Errorf("write artifact audit: %w", err)
	}
	return nil
}

func writeJSONFile(path string, v any) error {
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return fmt.Errorf("create output dir for %s: %w", path, err)
	}
	data, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal %s: %w", path, err)
	}
	if err := os.WriteFile(path, data, 0644); err != nil {
		return fmt.Errorf("write %s: %w", path, err)
	}
	return nil
}

func evaluatePhase102CLeaderboard(ctx context.Context, path, market, interval, from, to string, fromTime, toTime time.Time, symbols []string, audit ArtifactAuditReport) (LeaderboardJSON, map[string]FragileCandidateReport, error) {
	auditBySymbol := make(map[string]ArtifactAuditRow)
	for _, row := range audit.Rows {
		auditBySymbol[row.Symbol] = row
	}

	leaderboard := LeaderboardJSON{
		Summary: LeaderboardSummary{
			Market:           market,
			Interval:         interval,
			From:             from,
			To:               to,
			Symbols:          symbols,
			FamiliesRequired: len(requiredFamilies),
			ExpectancyUnit:   expectancyUnitBps,
			CostUnit:         "basis_points",
		},
		VerdictCounts: make(map[string]int),
	}
	focusedReports := make(map[string]FragileCandidateReport)

	for _, sym := range symbols {
		if sym == "BTCUSDT" {
			for _, row := range unsupportedContextRows(sym) {
				leaderboard.Rows = append(leaderboard.Rows, row)
			}
			continue
		}

		auditRow, ok := auditBySymbol[sym]
		if !ok {
			rows := artifactMissingRows(sym, "artifact audit row missing")
			leaderboard.Rows = append(leaderboard.Rows, rows...)
			continue
		}
		if !auditRow.CandlesAvailable {
			rows := artifactMissingRows(sym, auditRow.ReasonIfMissing)
			leaderboard.Rows = append(leaderboard.Rows, rows...)
			continue
		}
		if auditRow.BuildNeeded {
			reason := auditRow.ReasonIfMissing
			if reason == "" {
				reason = "feature/regime artifact build needed"
			}
			rows := artifactMissingRows(sym, reason)
			leaderboard.Rows = append(leaderboard.Rows, rows...)
			continue
		}

		featureRows, labels, err := readPhaseArtifacts(auditRow)
		if err != nil {
			rows := artifactMissingRows(sym, err.Error())
			leaderboard.Rows = append(leaderboard.Rows, rows...)
			continue
		}

		rows, reports := evaluateSymbolPhaseCandidates(sym, featureRows, labels)
		leaderboard.Rows = append(leaderboard.Rows, rows...)
		for key, report := range reports {
			if containsString(focusedFragileCandidates, key) {
				focusedReports[key] = report
			}
		}
	}

	sortLeaderboardRows(leaderboard.Rows)
	for _, row := range leaderboard.Rows {
		leaderboard.VerdictCounts[row.Verdict]++
	}
	for _, verdict := range verdictOrder {
		if _, ok := leaderboard.VerdictCounts[verdict]; !ok {
			leaderboard.VerdictCounts[verdict] = 0
		}
	}
	return leaderboard, focusedReports, nil
}

func readPhaseArtifacts(row ArtifactAuditRow) ([]features.Row, []regime.Label, error) {
	featureRows, err := features.ReadRowsJSON(row.FeaturePath)
	if err != nil {
		return nil, nil, fmt.Errorf("read feature artifact %s: %w", row.FeaturePath, err)
	}
	labels, err := regime.ReadLabelsJSON(row.RegimePath)
	if err != nil {
		return nil, nil, fmt.Errorf("read regime artifact %s: %w", row.RegimePath, err)
	}
	sort.Slice(featureRows, func(i, j int) bool { return featureRows[i].EventTimeMS < featureRows[j].EventTimeMS })
	sort.Slice(labels, func(i, j int) bool { return labels[i].AvailableAtMS < labels[j].AvailableAtMS })
	return featureRows, labels, nil
}

func unsupportedContextRows(symbol string) []LeaderboardRow {
	rows := make([]LeaderboardRow, 0, len(requiredFamilies))
	for _, famKey := range requiredFamilies {
		family, side := splitFamilySide(famKey)
		rows = append(rows, LeaderboardRow{
			Symbol:        symbol,
			Family:        family,
			Side:          side,
			TimeSplit:     "FY",
			Verdict:       verdictUnsupportedContext,
			FailedGates:   []string{"BTCUSDT is unsupported_context because target symbol cannot use itself as BTC beta context"},
			LeakageStatus: "N/A",
		})
	}
	return rows
}

func artifactMissingRows(symbol, reason string) []LeaderboardRow {
	if reason == "" {
		reason = "feature/regime artifact unavailable"
	}
	rows := make([]LeaderboardRow, 0, len(requiredFamilies))
	for _, famKey := range requiredFamilies {
		family, side := splitFamilySide(famKey)
		rows = append(rows, LeaderboardRow{
			Symbol:        symbol,
			Family:        family,
			Side:          side,
			TimeSplit:     "FY",
			Verdict:       verdictMissingData,
			FailedGates:   []string{reason},
			LeakageStatus: "N/A",
		})
	}
	return rows
}

func evaluateSymbolPhaseCandidates(symbol string, rows []features.Row, labels []regime.Label) ([]LeaderboardRow, map[string]FragileCandidateReport) {
	events, leakageStatus := generatePhaseEvents(rows, labels)
	byFamily := make(map[string][]phaseEvent)
	for _, event := range events {
		byFamily[event.Family+"_"+event.Side] = append(byFamily[event.Family+"_"+event.Side], event)
	}

	var out []LeaderboardRow
	reports := make(map[string]FragileCandidateReport)
	for _, famKey := range requiredFamilies {
		family, side := splitFamilySide(famKey)
		row, report := analyzeCandidate(symbol, family, side, byFamily[famKey], rows, leakageStatus)
		out = append(out, row)
		reports[symbol+"_"+family+"_"+side] = report
	}
	return out, reports
}

func generatePhaseEvents(rows []features.Row, labels []regime.Label) ([]phaseEvent, string) {
	leakageStatus := "PASS"
	if research.CheckFeatureRows(rows).Status != "PASS" || research.CheckLabels(labels).Status != "PASS" {
		leakageStatus = "FAIL"
	}

	var events []phaseEvent
	for i, row := range rows {
		if row.Warmup {
			continue
		}
		idx := sort.Search(len(labels), func(k int) bool {
			return labels[k].AvailableAtMS > row.EventTimeMS
		})
		if idx == 0 {
			continue
		}
		label := labels[idx-1]
		if label.AvailableAtMS > row.EventTimeMS {
			leakageStatus = "FAIL"
			continue
		}

		var candidates []struct {
			family string
			side   string
		}
		if row.Close > row.EMA20 && row.EMA20 > row.EMA50 && row.TrendSlope20 > 0 {
			candidates = append(candidates, struct {
				family string
				side   string
			}{"TrendContinuation", "LONG"})
		} else if row.Close < row.EMA20 && row.EMA20 < row.EMA50 && row.TrendSlope20 < 0 {
			candidates = append(candidates, struct {
				family string
				side   string
			}{"TrendContinuation", "SHORT"})
		}

		if label.Volatility == "compressed" || label.Composite == "compressed_range" {
			if row.Close > row.EMA20 && label.MarketBeta == "btc_up" {
				candidates = append(candidates, struct {
					family string
					side   string
				}{"CompressionBreakout", "LONG"})
			} else if row.Close < row.EMA20 && label.MarketBeta == "btc_down" {
				candidates = append(candidates, struct {
					family string
					side   string
				}{"CompressionBreakout", "SHORT"})
			}
		}

		if label.Volatility == "shock" {
			if row.Return5 < 0 {
				candidates = append(candidates, struct {
					family string
					side   string
				}{"ShockFade", "LONG"})
			} else if row.Return5 > 0 {
				candidates = append(candidates, struct {
					family string
					side   string
				}{"ShockFade", "SHORT"})
			}
		}

		if label.Liquidity == "heavy" {
			if row.Return5 > 0 && row.Return15 > 0 {
				candidates = append(candidates, struct {
					family string
					side   string
				}{"VolumeMomentum", "LONG"})
			} else if row.Return5 < 0 && row.Return15 < 0 {
				candidates = append(candidates, struct {
					family string
					side   string
				}{"VolumeMomentum", "SHORT"})
			}
		}

		if row.Return5 > 0 && label.MarketBeta == "btc_up" {
			candidates = append(candidates, struct {
				family string
				side   string
			}{"BetaAgrees", "LONG"})
		} else if row.Return5 < 0 && label.MarketBeta == "btc_down" {
			candidates = append(candidates, struct {
				family string
				side   string
			}{"BetaAgrees", "SHORT"})
		} else if row.Return5 > 0 && label.MarketBeta == "btc_down" {
			candidates = append(candidates, struct {
				family string
				side   string
			}{"BetaDiverges", "LONG"})
		} else if row.Return5 < 0 && label.MarketBeta == "btc_up" {
			candidates = append(candidates, struct {
				family string
				side   string
			}{"BetaDiverges", "SHORT"})
		}

		for _, c := range candidates {
			mae, mfe := excursionReturns(rows, i, 60*60000, c.side)
			events = append(events, phaseEvent{
				Index:       i,
				Family:      c.family,
				Side:        c.side,
				EventTimeMS: row.EventTimeMS,
				Label:       label,
				MFE:         mfe,
				MAE:         mae,
			})
		}
	}
	return events, leakageStatus
}

func analyzeCandidate(symbol, family, side string, events []phaseEvent, rows []features.Row, leakageStatus string) (LeaderboardRow, FragileCandidateReport) {
	splits := map[string]SplitMetrics{
		"Q1": metricForSplit(events, rows, 5, "Q1"),
		"Q2": metricForSplit(events, rows, 5, "Q2"),
		"Q3": metricForSplit(events, rows, 5, "Q3"),
		"Q4": metricForSplit(events, rows, 5, "Q4"),
		"H1": metricForSplit(events, rows, 5, "H1"),
		"H2": metricForSplit(events, rows, 5, "H2"),
		"FY": metricForSplit(events, rows, 5, "FY"),
	}
	costHaircuts := make([]CostHaircutMetric, 0, 4)
	for _, costBps := range []int{0, 2, 5, 10} {
		m := metricForEvents(events, rows, costBps, 0)
		costHaircuts = append(costHaircuts, CostHaircutMetric{
			CostBps:       costBps,
			EventCount:    m.EventCount,
			ProfitFactor:  m.ProfitFactor,
			ExpectancyBps: m.ExpectancyBps,
			WinRate:       m.WinRate,
		})
	}

	entryDelays := make([]EntryDelayMetric, 0, 4)
	entryDelay1cExpectancy := 0.0
	for _, delay := range []int{0, 1, 3, 5} {
		m := metricForEvents(events, rows, 5, delay)
		if delay == 1 {
			entryDelay1cExpectancy = m.ExpectancyBps
		}
		entryDelays = append(entryDelays, EntryDelayMetric{
			DelayCandles:  delay,
			CostBps:       5,
			EventCount:    m.EventCount,
			ProfitFactor:  m.ProfitFactor,
			ExpectancyBps: m.ExpectancyBps,
			WinRate:       m.WinRate,
		})
	}

	monthly := monthlyContribution(events, rows)
	positiveMonthCount := 0
	h2PositiveMonthCount := 0
	singleMonthPct := topMonthContributionPct(monthly, 1)
	top2MonthPct := topMonthContributionPct(monthly, 2)
	for _, month := range monthly {
		if month.NetContribution5bpsBps > 0 {
			positiveMonthCount++
			if strings.HasSuffix(month.Month, "-07") || strings.HasSuffix(month.Month, "-08") ||
				strings.HasSuffix(month.Month, "-09") || strings.HasSuffix(month.Month, "-10") ||
				strings.HasSuffix(month.Month, "-11") || strings.HasSuffix(month.Month, "-12") {
				h2PositiveMonthCount++
			}
		}
	}

	bestQuarter, worstQuarter := bestAndWorstQuarter(events, rows)
	worstQuarterPF := worstQuarterPF5bps(splits)
	cost10 := costHaircutByBps(costHaircuts, 10)
	avgMFE, avgMAE := avgExcursionBps(events)
	regimeBreakdown := regimeBreakdown(events, rows)

	row := LeaderboardRow{
		Symbol:                     symbol,
		Family:                     family,
		Side:                       side,
		TimeSplit:                  "FY",
		EventCount:                 splits["FY"].EventCount,
		H1PF5bps:                   splits["H1"].ProfitFactor5bps,
		H2PF5bps:                   splits["H2"].ProfitFactor5bps,
		FYPF5bps:                   splits["FY"].ProfitFactor5bps,
		H2Expectancy5bpsBps:        splits["H2"].Expectancy5bpsBps,
		FYExpectancy5bpsBps:        splits["FY"].Expectancy5bpsBps,
		PositiveMonthCount:         positiveMonthCount,
		EntryDelay1cExpectancyBps:  entryDelay1cExpectancy,
		BestQuarter:                bestQuarter,
		WorstQuarter:               worstQuarter,
		WorstQuarterPF5bps:         worstQuarterPF,
		SingleMonthContributionPct: singleMonthPct,
		Top2MonthContributionPct:   top2MonthPct,
		Cost10bpsPF:                cost10.ProfitFactor,
		Cost10bpsExpectancyBps:     cost10.ExpectancyBps,
		LeakageStatus:              leakageStatus,
		h2PositiveMonthCount:       h2PositiveMonthCount,
	}
	verdictInput := CandidateVerdictInput{
		H2Expectancy5bpsBps:        row.H2Expectancy5bpsBps,
		H2PF5bps:                   row.H2PF5bps,
		FYPF5bps:                   row.FYPF5bps,
		EventCount:                 row.EventCount,
		H1EventCount:               splits["H1"].EventCount,
		H2EventCount:               splits["H2"].EventCount,
		PositiveMonthCount:         row.PositiveMonthCount,
		EntryDelay1cExpectancyBps:  row.EntryDelay1cExpectancyBps,
		SingleMonthContributionPct: row.SingleMonthContributionPct,
		Top2MonthContributionPct:   row.Top2MonthContributionPct,
		WorstQuarterPF5bps:         row.WorstQuarterPF5bps,
		Cost10bpsReported:          true,
		LeakageStatus:              row.LeakageStatus,
	}
	row.Verdict, row.FailedGates = EvaluateCandidateVerdictWithMetrics(verdictInput)

	gates := CandidateGateResultsWithMetrics(verdictInput)
	report := FragileCandidateReport{
		Symbol:                     symbol,
		Family:                     family,
		Side:                       side,
		Verdict:                    row.Verdict,
		FailedGates:                row.FailedGates,
		PassedGates:                passedGateNames(gates),
		Gates:                      gates,
		ExpectancyUnit:             expectancyUnitBps,
		LeakageStatus:              leakageStatus,
		EventCount:                 row.EventCount,
		Splits:                     splits,
		CostHaircuts:               costHaircuts,
		EntryDelays:                entryDelays,
		MonthlyContribution:        monthly,
		SingleMonthContributionPct: singleMonthPct,
		Top2MonthContributionPct:   top2MonthPct,
		AvgMFEBps:                  avgMFE,
		AvgMAEBps:                  avgMAE,
		RegimeBreakdown:            regimeBreakdown,
	}
	return row, report
}

func metricForSplit(events []phaseEvent, rows []features.Row, costBps int, split string) SplitMetrics {
	filtered := filterEventsBySplit(events, split)
	metric := metricForEvents(filtered, rows, costBps, 0)
	return SplitMetrics{
		EventCount:             metric.EventCount,
		ProfitFactor5bps:       metric.ProfitFactor,
		Expectancy5bpsBps:      metric.ExpectancyBps,
		WinRate5bps:            metric.WinRate,
		NetContribution5bpsBps: sumReturnsBps(filtered, rows, costBps, 0),
	}
}

func metricForEvents(events []phaseEvent, rows []features.Row, costBps int, entryDelayCandles int) CandidateMetric {
	return metricFromReturns(returnsForEvents(events, rows, costBps, entryDelayCandles))
}

func returnsForEvents(events []phaseEvent, rows []features.Row, costBps int, entryDelayCandles int) []float64 {
	rets := make([]float64, 0, len(events))
	for _, event := range events {
		if event.Index < 0 || event.Index >= len(rows) {
			continue
		}
		entryIdx := event.Index + entryDelayCandles
		if entryIdx >= len(rows) {
			entryIdx = len(rows) - 1
		}
		entryClose := rows[entryIdx].Close
		if entryClose == 0 {
			continue
		}
		futureClose := futureClose(rows, event.Index, 60*60000)
		ret := (futureClose - entryClose) / entryClose
		if event.Side == "SHORT" {
			ret = -ret
		}
		ret -= float64(costBps) / 10000.0
		rets = append(rets, ret)
	}
	return rets
}

func metricFromReturns(rets []float64) CandidateMetric {
	metric := CandidateMetric{EventCount: len(rets)}
	if len(rets) == 0 {
		return metric
	}

	var grossProfit, grossLoss, net float64
	var wins int
	for _, ret := range rets {
		net += ret
		if ret > 0 {
			wins++
			grossProfit += ret
		} else if ret < 0 {
			grossLoss += -ret
		}
	}
	if grossLoss > 0 {
		metric.ProfitFactor = grossProfit / grossLoss
	} else if grossProfit > 0 {
		metric.ProfitFactor = 999.0
	}
	metric.ExpectancyBps = (net / float64(len(rets))) * 10000.0
	metric.WinRate = float64(wins) / float64(len(rets))
	return metric
}

func sumReturnsBps(events []phaseEvent, rows []features.Row, costBps int, entryDelayCandles int) float64 {
	var total float64
	for _, ret := range returnsForEvents(events, rows, costBps, entryDelayCandles) {
		total += ret
	}
	return total * 10000.0
}

func filterEventsBySplit(events []phaseEvent, split string) []phaseEvent {
	if split == "FY" {
		return append([]phaseEvent(nil), events...)
	}
	var filtered []phaseEvent
	for _, event := range events {
		month := time.UnixMilli(event.EventTimeMS).UTC().Month()
		switch split {
		case "Q1":
			if month >= time.January && month <= time.March {
				filtered = append(filtered, event)
			}
		case "Q2":
			if month >= time.April && month <= time.June {
				filtered = append(filtered, event)
			}
		case "Q3":
			if month >= time.July && month <= time.September {
				filtered = append(filtered, event)
			}
		case "Q4":
			if month >= time.October && month <= time.December {
				filtered = append(filtered, event)
			}
		case "H1":
			if month >= time.January && month <= time.June {
				filtered = append(filtered, event)
			}
		case "H2":
			if month >= time.July && month <= time.December {
				filtered = append(filtered, event)
			}
		}
	}
	return filtered
}

func monthlyContribution(events []phaseEvent, rows []features.Row) []MonthlyContribution {
	byMonth := make(map[string][]phaseEvent)
	for _, event := range events {
		month := time.UnixMilli(event.EventTimeMS).UTC().Format("2006-01")
		byMonth[month] = append(byMonth[month], event)
	}

	var totalPositiveBps float64
	metrics := make(map[string]MonthlyContribution)
	for month, monthEvents := range byMonth {
		m := metricForEvents(monthEvents, rows, 5, 0)
		netBps := sumReturnsBps(monthEvents, rows, 5, 0)
		if netBps > 0 {
			totalPositiveBps += netBps
		}
		metrics[month] = MonthlyContribution{
			Month:                  month,
			EventCount:             m.EventCount,
			ProfitFactor5bps:       m.ProfitFactor,
			Expectancy5bpsBps:      m.ExpectancyBps,
			NetContribution5bpsBps: netBps,
		}
	}

	var months []string
	for month := range metrics {
		months = append(months, month)
	}
	sort.Strings(months)

	out := make([]MonthlyContribution, 0, len(months))
	for _, month := range months {
		m := metrics[month]
		if totalPositiveBps > 0 && m.NetContribution5bpsBps > 0 {
			m.ContributionPctOfNetGain = m.NetContribution5bpsBps / totalPositiveBps * 100.0
		}
		out = append(out, m)
	}
	return out
}

func topMonthContributionPct(months []MonthlyContribution, topN int) float64 {
	if topN <= 0 {
		return 0
	}
	nets := make([]float64, 0, len(months))
	var totalPositive float64
	for _, month := range months {
		if month.NetContribution5bpsBps > 0 {
			nets = append(nets, month.NetContribution5bpsBps)
			totalPositive += month.NetContribution5bpsBps
		}
	}
	if totalPositive <= 0 {
		return 0
	}
	sort.Slice(nets, func(i, j int) bool { return nets[i] > nets[j] })
	if topN > len(nets) {
		topN = len(nets)
	}
	var top float64
	for i := 0; i < topN; i++ {
		top += nets[i]
	}
	return top / totalPositive * 100.0
}

func bestAndWorstQuarter(events []phaseEvent, rows []features.Row) (string, string) {
	bestQuarter := ""
	worstQuarter := ""
	best := -math.MaxFloat64
	worst := math.MaxFloat64
	for _, quarter := range []string{"Q1", "Q2", "Q3", "Q4"} {
		net := sumReturnsBps(filterEventsBySplit(events, quarter), rows, 5, 0)
		if net > best {
			best = net
			bestQuarter = quarter
		}
		if net < worst {
			worst = net
			worstQuarter = quarter
		}
	}
	return bestQuarter, worstQuarter
}

func worstQuarterPF5bps(splits map[string]SplitMetrics) float64 {
	worst := math.MaxFloat64
	found := false
	for _, quarter := range []string{"Q1", "Q2", "Q3", "Q4"} {
		m := splits[quarter]
		if m.EventCount == 0 {
			continue
		}
		if m.ProfitFactor5bps < worst {
			worst = m.ProfitFactor5bps
			found = true
		}
	}
	if !found {
		return 0
	}
	return worst
}

func costHaircutByBps(costs []CostHaircutMetric, bps int) CostHaircutMetric {
	for _, cost := range costs {
		if cost.CostBps == bps {
			return cost
		}
	}
	return CostHaircutMetric{CostBps: bps}
}

func regimeBreakdown(events []phaseEvent, rows []features.Row) []RegimeBreakdownMetric {
	byRegime := make(map[string][]phaseEvent)
	for _, event := range events {
		reg := event.Label.Composite
		if reg == "" {
			reg = "unknown"
		}
		byRegime[reg] = append(byRegime[reg], event)
	}
	var regimes []string
	for reg := range byRegime {
		regimes = append(regimes, reg)
	}
	sort.Strings(regimes)

	out := make([]RegimeBreakdownMetric, 0, len(regimes))
	for _, reg := range regimes {
		m := metricForEvents(byRegime[reg], rows, 5, 0)
		out = append(out, RegimeBreakdownMetric{
			Regime:             reg,
			EventCount:         m.EventCount,
			ProfitFactor5bps:   m.ProfitFactor,
			Expectancy5bpsBps:  m.ExpectancyBps,
			WinRate5bps:        m.WinRate,
			NetContributionBps: sumReturnsBps(byRegime[reg], rows, 5, 0),
		})
	}
	return out
}

func avgExcursionBps(events []phaseEvent) (float64, float64) {
	if len(events) == 0 {
		return 0, 0
	}
	var mfe, mae float64
	for _, event := range events {
		mfe += event.MFE
		mae += event.MAE
	}
	return mfe / float64(len(events)) * 10000.0, mae / float64(len(events)) * 10000.0
}

func futureClose(rows []features.Row, startIndex int, offsetMS int64) float64 {
	if len(rows) == 0 {
		return 0
	}
	targetTime := rows[startIndex].EventTimeMS + offsetMS
	if offsetMS%(60*1000) == 0 {
		offsetRows := int(offsetMS / (60 * 1000))
		idx := startIndex + offsetRows
		if idx < len(rows) && rows[idx].EventTimeMS == targetTime {
			return rows[idx].Close
		}
	}
	offset := sort.Search(len(rows)-startIndex, func(i int) bool {
		return rows[startIndex+i].EventTimeMS >= targetTime
	})
	if idx := startIndex + offset; idx < len(rows) {
		return rows[idx].Close
	}
	return rows[len(rows)-1].Close
}

func excursionReturns(rows []features.Row, startIndex int, offsetMS int64, side string) (float64, float64) {
	if len(rows) == 0 || startIndex >= len(rows) {
		return 0, 0
	}
	startClose := rows[startIndex].Close
	if startClose == 0 {
		return 0, 0
	}
	targetTime := rows[startIndex].EventTimeMS + offsetMS
	maxClose := startClose
	minClose := startClose
	for i := startIndex; i < len(rows); i++ {
		if rows[i].EventTimeMS > targetTime {
			break
		}
		if rows[i].Close > maxClose {
			maxClose = rows[i].Close
		}
		if rows[i].Close < minClose {
			minClose = rows[i].Close
		}
	}
	if side == "SHORT" {
		return (startClose - maxClose) / startClose, (startClose - minClose) / startClose
	}
	return (minClose - startClose) / startClose, (maxClose - startClose) / startClose
}

func splitFamilySide(famKey string) (string, string) {
	idx := strings.LastIndex(famKey, "_")
	if idx == -1 {
		return famKey, ""
	}
	return famKey[:idx], famKey[idx+1:]
}

func CandidateGateResults(h2ExpBps, h2PF, fyPF float64, eventCount, positiveMonthCount int, entryDelay1cExpBps, singleMonthContributionPct float64, leakageStatus string) []GateResult {
	return CandidateGateResultsWithMetrics(CandidateVerdictInput{
		H2Expectancy5bpsBps:        h2ExpBps,
		H2PF5bps:                   h2PF,
		FYPF5bps:                   fyPF,
		EventCount:                 eventCount,
		PositiveMonthCount:         positiveMonthCount,
		EntryDelay1cExpectancyBps:  entryDelay1cExpBps,
		SingleMonthContributionPct: singleMonthContributionPct,
		Top2MonthContributionPct:   0,
		WorstQuarterPF5bps:         1,
		Cost10bpsReported:          true,
		LeakageStatus:              leakageStatus,
	})
}

func CandidateGateResultsWithMetrics(in CandidateVerdictInput) []GateResult {
	return []GateResult{
		{Name: "H2 PF after 5 bps", Passed: in.H2PF5bps >= 1.10, Actual: fmt.Sprintf("%.4f", in.H2PF5bps), Threshold: ">= 1.10"},
		{Name: "H2 expectancy after 5 bps", Passed: in.H2Expectancy5bpsBps > 0, Actual: fmt.Sprintf("%.4f bps", in.H2Expectancy5bpsBps), Threshold: "> 0 bps"},
		{Name: "FY PF after 5 bps", Passed: in.FYPF5bps >= 1.05, Actual: fmt.Sprintf("%.4f", in.FYPF5bps), Threshold: ">= 1.05"},
		{Name: "event_count", Passed: in.EventCount >= 300, Actual: fmt.Sprintf("%d", in.EventCount), Threshold: ">= 300"},
		{Name: "positive_month_count", Passed: in.PositiveMonthCount >= 3, Actual: fmt.Sprintf("%d", in.PositiveMonthCount), Threshold: ">= 3"},
		{Name: "entry_delay_1c_expectancy_bps", Passed: in.EntryDelay1cExpectancyBps > 0, Actual: fmt.Sprintf("%.4f bps", in.EntryDelay1cExpectancyBps), Threshold: "> 0 bps"},
		{Name: "worst_quarter_pf_5bps", Passed: in.WorstQuarterPF5bps >= 0.95, Actual: fmt.Sprintf("%.4f", in.WorstQuarterPF5bps), Threshold: ">= 0.95"},
		{Name: "single_month_contribution_pct", Passed: in.SingleMonthContributionPct <= 50, Actual: fmt.Sprintf("%.2f%%", in.SingleMonthContributionPct), Threshold: "<= 50%"},
		{Name: "top_2_month_contribution_pct", Passed: in.Top2MonthContributionPct <= 70, Actual: fmt.Sprintf("%.2f%%", in.Top2MonthContributionPct), Threshold: "<= 70%"},
		{Name: "cost_10bps_sensitivity_reported", Passed: in.Cost10bpsReported, Actual: fmt.Sprintf("%t", in.Cost10bpsReported), Threshold: "true"},
		{Name: "leakage_status", Passed: in.LeakageStatus == "PASS", Actual: in.LeakageStatus, Threshold: "PASS"},
	}
}

func EvaluateCandidateVerdict(h2ExpBps, h2PF, fyPF float64, eventCount, h1EventCount, h2EventCount, positiveMonthCount int, entryDelay1cExpBps, singleMonthContributionPct float64, leakageStatus string) (string, []string) {
	return EvaluateCandidateVerdictWithMetrics(CandidateVerdictInput{
		H2Expectancy5bpsBps:        h2ExpBps,
		H2PF5bps:                   h2PF,
		FYPF5bps:                   fyPF,
		EventCount:                 eventCount,
		H1EventCount:               h1EventCount,
		H2EventCount:               h2EventCount,
		PositiveMonthCount:         positiveMonthCount,
		EntryDelay1cExpectancyBps:  entryDelay1cExpBps,
		SingleMonthContributionPct: singleMonthContributionPct,
		Top2MonthContributionPct:   0,
		WorstQuarterPF5bps:         1,
		Cost10bpsReported:          true,
		LeakageStatus:              leakageStatus,
	})
}

func EvaluateCandidateVerdictWithMetrics(in CandidateVerdictInput) (string, []string) {
	gates := CandidateGateResultsWithMetrics(in)
	failed := failedGateNames(gates)
	if in.H1EventCount == 0 {
		failed = append(failed, "H1 event_count == 0")
	}
	if in.H2EventCount == 0 {
		failed = append(failed, "H2 event_count == 0")
	}
	if len(failed) == 0 {
		return verdictResearchLead, nil
	}

	if in.LeakageStatus != "PASS" ||
		in.H1EventCount == 0 ||
		in.H2EventCount == 0 ||
		in.EventCount < 300 ||
		in.H2Expectancy5bpsBps <= 0 ||
		in.H2PF5bps < 1.0 ||
		in.FYPF5bps < 1.0 ||
		in.WorstQuarterPF5bps < 0.95 ||
		in.SingleMonthContributionPct > 50 ||
		in.Top2MonthContributionPct > 70 ||
		!in.Cost10bpsReported {
		return verdictRejected, failed
	}
	return verdictFragile, failed
}

func failedGateNames(gates []GateResult) []string {
	var failed []string
	for _, gate := range gates {
		if !gate.Passed {
			failed = append(failed, gate.Name)
		}
	}
	return failed
}

func passedGateNames(gates []GateResult) []string {
	var passed []string
	for _, gate := range gates {
		if gate.Passed {
			passed = append(passed, gate.Name)
		}
	}
	return passed
}

func countVerdicts(rows []LeaderboardRow) map[string]int {
	counts := make(map[string]int)
	for _, row := range rows {
		counts[row.Verdict]++
	}
	for _, verdict := range verdictOrder {
		if _, ok := counts[verdict]; !ok {
			counts[verdict] = 0
		}
	}
	return counts
}

func sortLeaderboardRows(rows []LeaderboardRow) {
	sort.Slice(rows, func(i, j int) bool {
		if rows[i].Verdict != rows[j].Verdict {
			return verdictRank(rows[i].Verdict) < verdictRank(rows[j].Verdict)
		}
		if rows[i].Symbol != rows[j].Symbol {
			return rows[i].Symbol < rows[j].Symbol
		}
		if rows[i].Family != rows[j].Family {
			return rows[i].Family < rows[j].Family
		}
		return rows[i].Side < rows[j].Side
	})
}

func verdictRank(verdict string) int {
	for i, v := range verdictOrder {
		if verdict == v {
			return i
		}
	}
	return len(verdictOrder)
}

func containsString(list []string, target string) bool {
	for _, item := range list {
		if item == target {
			return true
		}
	}
	return false
}

func buildArtifactAuditMD(report ArtifactAuditReport) string {
	var sb strings.Builder
	sb.WriteString("# Phase 10.2C Artifact Audit\n\n")
	sb.WriteString("## Summary\n")
	sb.WriteString(fmt.Sprintf("- Market: `%s`\n", report.Summary.Market))
	sb.WriteString(fmt.Sprintf("- Interval: `%s`\n", report.Summary.Interval))
	sb.WriteString(fmt.Sprintf("- Window: `%s` through `%s`\n", report.Summary.From, report.Summary.To))
	sb.WriteString(fmt.Sprintf("- Candles available: %d/%d\n", report.Summary.CandlesAvailableCount, len(report.Rows)))
	sb.WriteString(fmt.Sprintf("- Missing features before: %d\n", report.Summary.MissingFeaturesBefore))
	sb.WriteString(fmt.Sprintf("- Missing regimes before: %d\n", report.Summary.MissingRegimesBefore))
	sb.WriteString(fmt.Sprintf("- Build needed: %d\n", report.Summary.BuildNeededCount))
	if len(report.Summary.UnsupportedContextSymbolsExcluded) > 0 {
		sb.WriteString(fmt.Sprintf("- Unsupported context symbols excluded from artifact audit: `%s`\n", strings.Join(report.Summary.UnsupportedContextSymbolsExcluded, "`, `")))
	}
	sb.WriteString("\n")
	sb.WriteString("| Symbol | Candles Available | Candle Rows | Features Exist | Regimes Exist | Feature Rows | Regime Rows | Reason If Missing |\n")
	sb.WriteString("|---|---:|---:|---:|---:|---:|---:|---|\n")
	for _, row := range report.Rows {
		reason := row.ReasonIfMissing
		if reason == "" {
			reason = "-"
		}
		sb.WriteString(fmt.Sprintf("| %s | %t | %d | %t | %t | %d | %d | %s |\n",
			row.Symbol, row.CandlesAvailable, row.CandleRows, row.FeaturesExist, row.RegimesExist, row.FeatureRows, row.RegimeRows, reason))
	}
	sb.WriteString("\n")
	return sb.String()
}

func buildMultisymbolAlphaBaselinesMD(report LeaderboardJSON, audit ArtifactAuditReport) string {
	var sb strings.Builder
	sb.WriteString("# Phase 10.2C Multi-Symbol Alpha Baselines\n\n")
	sb.WriteString("## Units\n")
	sb.WriteString("- Expectancy unit: basis points (`*_expectancy_*_bps`).\n")
	sb.WriteString("- Profit factor is unitless.\n")
	sb.WriteString("- Cost haircuts are basis points and are explicit in field names or table headers.\n\n")
	sb.WriteString("## Artifact Audit Summary\n")
	sb.WriteString(fmt.Sprintf("- Missing features before: %d\n", audit.Summary.MissingFeaturesBefore))
	sb.WriteString(fmt.Sprintf("- Missing regimes before: %d\n", audit.Summary.MissingRegimesBefore))
	sb.WriteString(fmt.Sprintf("- Build needed: %d\n\n", audit.Summary.BuildNeededCount))
	sb.WriteString("## Verdict Counts\n")
	for _, verdict := range verdictOrder {
		sb.WriteString(fmt.Sprintf("- %s: %d\n", verdict, report.VerdictCounts[verdict]))
	}
	sb.WriteString("\n")
	sb.WriteString("## All Candidate Rows\n")
	writeLeaderboardTable(&sb, report.Rows)
	sb.WriteString("\n")
	return sb.String()
}

func buildCandidateLeaderboardMD(report LeaderboardJSON) string {
	var sb strings.Builder
	sb.WriteString("# Phase 10.2C Candidate Leaderboard\n\n")
	sb.WriteString("## Units\n")
	sb.WriteString("- Expectancy is reported in basis points (`bps`).\n")
	sb.WriteString("- `5bps` columns are after a 5 basis point cost haircut.\n\n")
	sb.WriteString("## Verdict Counts\n")
	for _, verdict := range verdictOrder {
		sb.WriteString(fmt.Sprintf("- %s: %d\n", verdict, report.VerdictCounts[verdict]))
	}
	sb.WriteString("\n")
	for _, verdict := range []string{verdictResearchLead, verdictFragile, verdictRejected, verdictMissingData, verdictUnsupportedContext, verdictInconclusive} {
		sb.WriteString(fmt.Sprintf("## %s Rows\n", strings.Title(strings.ReplaceAll(verdict, "_", " "))))
		var rows []LeaderboardRow
		for _, row := range report.Rows {
			if row.Verdict == verdict {
				rows = append(rows, row)
			}
		}
		if len(rows) == 0 {
			sb.WriteString("none\n\n")
			continue
		}
		writeLeaderboardTable(&sb, rows)
		sb.WriteString("\n")
	}
	sb.WriteString("## Final Recommendation\n")
	if report.VerdictCounts[verdictResearchLead] == 0 {
		if report.VerdictCounts[verdictFragile] > 0 {
			sb.WriteString("one or more candidates worth Phase 10.3 fragile follow-up\n")
		} else if report.VerdictCounts[verdictMissingData] > 0 {
			sb.WriteString("inconclusive due to remaining artifact/report limitations\n")
		} else {
			sb.WriteString("no research leads found\n")
		}
	} else {
		sb.WriteString("one or more candidates worth Phase 10.3\n")
	}
	sb.WriteString("\n")
	return sb.String()
}

func writeLeaderboardTable(sb *strings.Builder, rows []LeaderboardRow) {
	sb.WriteString("| Symbol | Family | Side | Verdict | Events | H2 PF 5bps | H2 Exp 5bps (bps) | FY PF 5bps | FY Exp 5bps (bps) | Worst Q PF | Delay 1c Exp (bps) | Positive Months | Top 1 Month % | Top 2 Month % | 10bps PF | Leakage | Failed Gates |\n")
	sb.WriteString("|---|---|---|---|---:|---:|---:|---:|---:|---:|---:|---:|---:|---:|---:|---|---|\n")
	for _, row := range rows {
		failed := "-"
		if len(row.FailedGates) > 0 {
			failed = strings.Join(row.FailedGates, "; ")
		}
		sb.WriteString(fmt.Sprintf("| %s | %s | %s | %s | %d | %.4f | %.4f | %.4f | %.4f | %.4f | %.4f | %d | %.2f | %.2f | %.4f | %s | %s |\n",
			row.Symbol,
			row.Family,
			row.Side,
			row.Verdict,
			row.EventCount,
			row.H2PF5bps,
			row.H2Expectancy5bpsBps,
			row.FYPF5bps,
			row.FYExpectancy5bpsBps,
			row.WorstQuarterPF5bps,
			row.EntryDelay1cExpectancyBps,
			row.PositiveMonthCount,
			row.SingleMonthContributionPct,
			row.Top2MonthContributionPct,
			row.Cost10bpsPF,
			row.LeakageStatus,
			failed))
	}
}

func buildFragileCandidateMD(report FragileCandidateReport) string {
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("# Phase 10.2C Fragile Validation: %s %s_%s\n\n", report.Symbol, report.Family, report.Side))
	sb.WriteString("## Verdict\n")
	sb.WriteString(fmt.Sprintf("- Verdict: `%s`\n", report.Verdict))
	sb.WriteString(fmt.Sprintf("- Leakage status: `%s`\n", report.LeakageStatus))
	sb.WriteString("- Expectancy unit: basis points (`bps`).\n")
	sb.WriteString(fmt.Sprintf("- Event count: %d\n", report.EventCount))
	sb.WriteString(fmt.Sprintf("- Single-month contribution: %.2f%%\n", report.SingleMonthContributionPct))
	sb.WriteString(fmt.Sprintf("- Top-2 month contribution: %.2f%%\n", report.Top2MonthContributionPct))
	sb.WriteString(fmt.Sprintf("- Avg MFE: %.4f bps\n", report.AvgMFEBps))
	sb.WriteString(fmt.Sprintf("- Avg MAE: %.4f bps\n\n", report.AvgMAEBps))

	sb.WriteString("## Gates\n")
	sb.WriteString("| Gate | Passed | Actual | Threshold |\n")
	sb.WriteString("|---|---:|---:|---|\n")
	for _, gate := range report.Gates {
		sb.WriteString(fmt.Sprintf("| %s | %t | %s | %s |\n", gate.Name, gate.Passed, gate.Actual, gate.Threshold))
	}
	sb.WriteString("\n")

	sb.WriteString("## Time Splits\n")
	sb.WriteString("| Split | Events | PF 5bps | Expectancy 5bps (bps) | Win Rate 5bps | Net Contribution 5bps (bps) |\n")
	sb.WriteString("|---|---:|---:|---:|---:|---:|\n")
	for _, split := range []string{"Q1", "Q2", "Q3", "Q4", "H1", "H2", "FY"} {
		m := report.Splits[split]
		sb.WriteString(fmt.Sprintf("| %s | %d | %.4f | %.4f | %.4f | %.4f |\n", split, m.EventCount, m.ProfitFactor5bps, m.Expectancy5bpsBps, m.WinRate5bps, m.NetContribution5bpsBps))
	}
	sb.WriteString("\n")

	sb.WriteString("## Cost Haircuts\n")
	sb.WriteString("| Cost (bps) | Events | PF | Expectancy (bps) | Win Rate |\n")
	sb.WriteString("|---:|---:|---:|---:|---:|\n")
	for _, m := range report.CostHaircuts {
		sb.WriteString(fmt.Sprintf("| %d | %d | %.4f | %.4f | %.4f |\n", m.CostBps, m.EventCount, m.ProfitFactor, m.ExpectancyBps, m.WinRate))
	}
	sb.WriteString("\n")

	sb.WriteString("## Entry Delays\n")
	sb.WriteString("| Delay (candles) | Cost (bps) | Events | PF | Expectancy (bps) | Win Rate |\n")
	sb.WriteString("|---:|---:|---:|---:|---:|---:|\n")
	for _, m := range report.EntryDelays {
		sb.WriteString(fmt.Sprintf("| %d | %d | %d | %.4f | %.4f | %.4f |\n", m.DelayCandles, m.CostBps, m.EventCount, m.ProfitFactor, m.ExpectancyBps, m.WinRate))
	}
	sb.WriteString("\n")

	sb.WriteString("## Monthly Contribution\n")
	sb.WriteString("| Month | Events | PF 5bps | Expectancy 5bps (bps) | Net Contribution 5bps (bps) | Contribution % Of Positive Net |\n")
	sb.WriteString("|---|---:|---:|---:|---:|---:|\n")
	for _, m := range report.MonthlyContribution {
		sb.WriteString(fmt.Sprintf("| %s | %d | %.4f | %.4f | %.4f | %.2f |\n", m.Month, m.EventCount, m.ProfitFactor5bps, m.Expectancy5bpsBps, m.NetContribution5bpsBps, m.ContributionPctOfNetGain))
	}
	sb.WriteString("\n")

	sb.WriteString("## Regime Breakdown\n")
	sb.WriteString("| Composite Regime | Events | PF 5bps | Expectancy 5bps (bps) | Win Rate 5bps | Net Contribution 5bps (bps) |\n")
	sb.WriteString("|---|---:|---:|---:|---:|---:|\n")
	for _, m := range report.RegimeBreakdown {
		sb.WriteString(fmt.Sprintf("| %s | %d | %.4f | %.4f | %.4f | %.4f |\n", m.Regime, m.EventCount, m.ProfitFactor5bps, m.Expectancy5bpsBps, m.WinRate5bps, m.NetContributionBps))
	}
	sb.WriteString("\n")
	return sb.String()
}
