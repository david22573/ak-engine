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

	"github.com/david22573/ak-engine/internal/data"
	"github.com/david22573/ak-engine/internal/executionseries"
	"github.com/david22573/ak-engine/internal/features"
	"github.com/david22573/ak-engine/internal/regime"
	"github.com/david22573/ak-engine/internal/researchidentity"
	"github.com/david22573/ak-engine/internal/rifbridge"
	"github.com/david22573/ak-engine/internal/temporal"
	"github.com/david22573/ak-engine/pkg/protocol"
	"github.com/spf13/cobra"
)

const (
	phase103StatusRejected              = "rejected"
	phase103StatusFragile               = "fragile"
	phase103StatusValidatedResearchLead = "validated_research_lead"
	phase103StatusNeedsMoreData         = "needs_more_data"
)

var phase103AllowedStatuses = map[string]bool{
	phase103StatusRejected:              true,
	phase103StatusFragile:               true,
	phase103StatusValidatedResearchLead: true,
	phase103StatusNeedsMoreData:         true,
}

var (
	ecdFeatures     string
	ecdRegimes      string
	ecdFamily       string
	ecdSide         string
	ecdSymbol       string
	ecdOut          string
	ecdCandlePath   string
	ecdMarket       string
	ecdInterval     string
	ecdManifest     string
	ecdConfig       string
	ecdDiagnostics  bool
	erldLeaderboard string
	erldOutDir      string
	erldCandlePath  string
	erldManifest    string
	erldConfig      string
	erldDiagnostics bool
)

type Phase103InventoryReport struct {
	Summary Phase103InventorySummary `json:"summary"`
	Rows    []Phase103InventoryRow   `json:"rows"`
}

type Phase103InventorySummary struct {
	SourceMarkdown    string         `json:"source_markdown"`
	SourceJSON        string         `json:"source_json"`
	ExpectedCount     int            `json:"expected_count"`
	ResearchLeadCount int            `json:"research_lead_count"`
	CountMismatch     bool           `json:"count_mismatch"`
	VerdictCounts     map[string]int `json:"verdict_counts"`
}

type Phase103InventoryRow struct {
	Symbol                     string   `json:"symbol"`
	Family                     string   `json:"family"`
	Side                       string   `json:"side"`
	EventCount                 int      `json:"event_count"`
	H2PF5bps                   float64  `json:"h2_pf_5bps"`
	H2Expectancy5bpsBps        float64  `json:"h2_expectancy_5bps_bps"`
	FYPF5bps                   float64  `json:"fy_pf_5bps"`
	FYExpectancy5bpsBps        float64  `json:"fy_expectancy_5bps_bps"`
	PositiveMonthCount         int      `json:"positive_month_count"`
	EntryDelay1cExpectancyBps  float64  `json:"entry_delay_1c_expectancy_bps"`
	SingleMonthContributionPct float64  `json:"single_month_contribution_pct"`
	LeakageStatus              string   `json:"leakage_status"`
	PassedGates                []string `json:"passed_gates"`
}

type DeepCandidateReport struct {
	Symbol                     string                        `json:"symbol"`
	Family                     string                        `json:"family"`
	Side                       string                        `json:"side"`
	SourceFeatures             string                        `json:"source_features"`
	SourceRegimes              string                        `json:"source_regimes"`
	CandlePath                 string                        `json:"candle_path"`
	ExpectancyUnit             string                        `json:"expectancy_unit"`
	CostUnit                   string                        `json:"cost_unit"`
	CandidateGenerationAudit   DeepCandidateAudit            `json:"candidate_generation_audit"`
	ForwardReturns             []DeepHorizonMetric           `json:"forward_returns"`
	CostHaircuts               []DeepCostHaircutMetric       `json:"cost_haircuts"`
	EntryDelays                []DeepEntryDelayMetric        `json:"entry_delays"`
	MFEMAE                     []DeepExcursionMetric         `json:"mfe_mae"`
	Brackets                   []DeepBracketMetric           `json:"brackets"`
	Stability                  []DeepStabilityMetric         `json:"stability"`
	RegimeBreakdown            map[string][]DeepRegimeMetric `json:"regime_breakdown"`
	LeakageStatus              string                        `json:"leakage_status"`
	LeakageIssues              []string                      `json:"leakage_issues,omitempty"`
	TruncationReport           DeepTruncationReport          `json:"truncation_report"`
	Gates                      []DeepGateResult              `json:"gates"`
	PassedGates                []string                      `json:"passed_gates"`
	FailedGates                []string                      `json:"failed_gates"`
	FinalStatus                string                        `json:"final_status"`
	FixedHoldOnlyJustification string                        `json:"fixed_hold_only_justification,omitempty"`
	ComparisonMetrics          DeepComparisonMetrics         `json:"comparison_metrics"`
	ExecutionSeriesID          string                        `json:"execution_series_id"`
	ExecutionSeriesVersion     string                        `json:"execution_series_version"`
	RIFReturns                 []float64                     `json:"-"`
	RIFTimestamps              []int64                       `json:"-"`
	researchIdentityRequest    researchidentity.DerivationRequest
}

type DeepCandidateAudit struct {
	Family                                     string `json:"family"`
	Side                                       string `json:"side"`
	TriggerRows                                int    `json:"trigger_rows"`
	CompressedRangeRows                        int    `json:"compressed_range_rows"`
	ShockRows                                  int    `json:"shock_rows"`
	EligibleSideCandidates                     int    `json:"eligible_side_candidates"`
	EligibleShortCandidates                    int    `json:"eligible_short_candidates,omitempty"`
	EligibleLongCandidates                     int    `json:"eligible_long_candidates,omitempty"`
	CandidatesRejectedByDirectionRule          int    `json:"candidates_rejected_by_direction_rule"`
	CandidatesRejectedByBTCETHBetaConfirmation int    `json:"candidates_rejected_by_btc_eth_beta_confirmation"`
	AcceptedCandidates                         int    `json:"accepted_candidates"`
	ClusteredCandidates                        int    `json:"clustered_candidates"`
	UniqueEventClusters                        int    `json:"unique_event_clusters"`
}

type DeepHorizonMetric struct {
	HorizonMinutes     int     `json:"horizon_minutes"`
	EventCount         int     `json:"event_count"`
	AverageReturnBps   float64 `json:"average_return_bps"`
	MedianReturnBps    float64 `json:"median_return_bps"`
	WinRate            float64 `json:"win_rate"`
	PFProxy            float64 `json:"pf_proxy"`
	ExpectancyBps      float64 `json:"expectancy_bps"`
	P25ReturnBps       float64 `json:"p25_return_bps"`
	P75ReturnBps       float64 `json:"p75_return_bps"`
	Worst5PctReturnBps float64 `json:"worst_5_pct_return_bps"`
	Best5PctReturnBps  float64 `json:"best_5_pct_return_bps"`
	TruncatedCount     int     `json:"truncated_count"`
}

type DeepCostHaircutMetric struct {
	CostBps        float64 `json:"cost_bps"`
	EventCount     int     `json:"event_count"`
	PF             float64 `json:"pf"`
	ExpectancyBps  float64 `json:"expectancy_bps"`
	WinRate        float64 `json:"win_rate"`
	PFAbove105     bool    `json:"pf_above_1_05"`
	PFAbove110     bool    `json:"pf_above_1_10"`
	PFAbove120     bool    `json:"pf_above_1_20"`
	TruncatedCount int     `json:"truncated_count"`
}

type DeepEntryDelayMetric struct {
	DelayCandles   int     `json:"delay_candles"`
	EventCount     int     `json:"event_count"`
	ExpectancyBps  float64 `json:"expectancy_bps"`
	PFAfter5bps    float64 `json:"pf_after_5bps"`
	WinRate        float64 `json:"win_rate"`
	TruncatedCount int     `json:"truncated_count"`
}

type DeepExcursionMetric struct {
	WindowMinutes              int     `json:"window_minutes"`
	EventCount                 int     `json:"event_count"`
	MedianMFEBps               float64 `json:"median_mfe_bps"`
	MedianMAEBps               float64 `json:"median_mae_bps"`
	AverageMFEBps              float64 `json:"average_mfe_bps"`
	AverageMAEBps              float64 `json:"average_mae_bps"`
	MFEMAERatio                float64 `json:"mfe_mae_ratio"`
	Reach5BpsBeforeMinus5Pct   float64 `json:"percent_reaching_5_bps_before_minus_5_bps"`
	Reach10BpsBeforeMinus5Pct  float64 `json:"percent_reaching_10_bps_before_minus_5_bps"`
	Reach15BpsBeforeMinus10Pct float64 `json:"percent_reaching_15_bps_before_minus_10_bps"`
	TruncatedCount             int     `json:"truncated_count"`
}

type DeepBracketMetric struct {
	Name                      string  `json:"name"`
	TPBps                     float64 `json:"tp_bps"`
	SLBps                     float64 `json:"sl_bps"`
	TradeCount                int     `json:"trade_count"`
	WinRate                   float64 `json:"win_rate"`
	PFAfter5bps               float64 `json:"pf_after_5bps"`
	NetExpectancyBpsAfter5bps float64 `json:"net_expectancy_bps_after_5bps"`
	AverageHoldMinutes        float64 `json:"average_hold_minutes"`
	UnresolvedCount           int     `json:"unresolved_count"`
	NotCatastrophicAfter5bps  bool    `json:"not_catastrophic_after_5bps"`
}

type DeepStabilityMetric struct {
	Period                     string  `json:"period"`
	EventCount                 int     `json:"event_count"`
	PFAfter5bps                float64 `json:"pf_after_5bps"`
	ExpectancyBpsAfter5bps     float64 `json:"expectancy_bps_after_5bps"`
	WinRate                    float64 `json:"win_rate"`
	BestMonth                  string  `json:"best_month"`
	WorstMonth                 string  `json:"worst_month"`
	SingleMonthContributionPct float64 `json:"single_month_contribution_pct"`
	PositiveMonthCount         int     `json:"positive_month_count"`
	TruncatedCount             int     `json:"truncated_count"`
}

type DeepRegimeMetric struct {
	Bucket                 string  `json:"bucket"`
	EventCount             int     `json:"event_count"`
	PFAfter5bps            float64 `json:"pf_after_5bps"`
	ExpectancyBpsAfter5bps float64 `json:"expectancy_bps_after_5bps"`
	WinRate                float64 `json:"win_rate"`
	LowSampleWarning       string  `json:"low_sample_warning,omitempty"`
}

type DeepTruncationReport struct {
	LoadedFirstCandleMS      int64 `json:"loaded_first_candle_ms"`
	LoadedLastCandleMS       int64 `json:"loaded_last_candle_ms"`
	ForwardReturnTruncations int   `json:"forward_return_truncations"`
	ExcursionTruncations     int   `json:"excursion_truncations"`
	StabilityTruncations     int   `json:"stability_truncations"`
}

type DeepGateResult struct {
	Name      string `json:"name"`
	Passed    bool   `json:"passed"`
	Critical  bool   `json:"critical"`
	Actual    string `json:"actual"`
	Threshold string `json:"threshold"`
}

type DeepComparisonMetrics struct {
	EventCount                 int     `json:"event_count"`
	FYPFAfter5bps              float64 `json:"fy_pf_after_5bps"`
	H2PFAfter5bps              float64 `json:"h2_pf_after_5bps"`
	WorstQuarterPFAfter5bps    float64 `json:"worst_quarter_pf_after_5bps"`
	FYExpectancyBps            float64 `json:"fy_expectancy_bps"`
	H2ExpectancyBps            float64 `json:"h2_expectancy_bps"`
	EntryDelay1cExpectancyBps  float64 `json:"entry_delay_1c_expectancy_bps"`
	PositiveMonthCount         int     `json:"positive_month_count"`
	SingleMonthContributionPct float64 `json:"single_month_contribution_pct"`
	BestBracketPFAfter5bps     float64 `json:"best_bracket_pf_after_5bps"`
	BestBracketExpectancyBps   float64 `json:"best_bracket_expectancy_bps_after_5bps"`
	MFEMAERatio                float64 `json:"mfe_mae_ratio"`
	CostSensitivity            string  `json:"cost_sensitivity"`
	LeakageStatus              string  `json:"leakage_status"`
}

type Phase103ComparisonReport struct {
	Summary Phase103ComparisonSummary `json:"summary"`
	Rows    []Phase103ComparisonRow   `json:"rows"`
}

type Phase103ComparisonSummary struct {
	CandidateCount      int    `json:"candidate_count"`
	StrongestCandidate  string `json:"strongest_candidate"`
	WeakestCandidate    string `json:"weakest_candidate"`
	FinalRecommendation string `json:"final_recommendation"`
}

type Phase103ComparisonRow struct {
	Symbol                     string   `json:"symbol"`
	Family                     string   `json:"family"`
	Side                       string   `json:"side"`
	EventCount                 int      `json:"event_count"`
	FYPFAfter5bps              float64  `json:"fy_pf_after_5bps"`
	H2PFAfter5bps              float64  `json:"h2_pf_after_5bps"`
	WorstQuarterPFAfter5bps    float64  `json:"worst_quarter_pf_after_5bps"`
	FYExpectancyBps            float64  `json:"fy_expectancy_bps"`
	H2ExpectancyBps            float64  `json:"h2_expectancy_bps"`
	EntryDelay1cExpectancyBps  float64  `json:"entry_delay_1c_expectancy_bps"`
	PositiveMonthCount         int      `json:"positive_month_count"`
	SingleMonthContributionPct float64  `json:"single_month_contribution_pct"`
	BestBracketPFAfter5bps     float64  `json:"best_bracket_pf_after_5bps"`
	BestBracketExpectancyBps   float64  `json:"best_bracket_expectancy_bps_after_5bps"`
	MFEMAERatio                float64  `json:"mfe_mae_ratio"`
	CostSensitivity            string   `json:"cost_sensitivity"`
	LeakageStatus              string   `json:"leakage_status"`
	FinalStatus                string   `json:"final_status"`
	FailedGates                []string `json:"failed_gates"`
}

type deepCandidateEvent struct {
	FeatureIndex   int
	CandleIndex    int
	EventTimeMS    int64
	DecisionTimeMS int64
	Family         string
	Side           string
	Label          regime.Label
}

type deepReturnSet struct {
	ReturnsBps      []float64
	EventTimestamps []int64
	TruncatedCount  int
}

type deepBracketResult struct {
	ReturnBps   float64
	HoldMinutes int
	Outcome     string
}

var evaluateCandidateDeepCmd = &cobra.Command{
	Use:   "evaluate-candidate-deep",
	Short: "Deep-validate a research-only candidate",
	RunE: func(cmd *cobra.Command, args []string) error {
		if ecdFeatures == "" {
			return errors.New("missing --features")
		}
		if ecdRegimes == "" {
			return errors.New("missing --regimes")
		}
		if ecdFamily == "" {
			return errors.New("missing --family")
		}
		if ecdSide == "" {
			return errors.New("missing --side")
		}
		if ecdSymbol == "" {
			return errors.New("missing --symbol")
		}
		if ecdOut == "" {
			return errors.New("missing --out")
		}
		configOverride, err := readOptionalResearchConfiguration(ecdConfig)
		if err != nil {
			return err
		}
		report, err := buildDeepCandidateReport(cmd.Context(), deepCandidateRequest{
			FeaturesPath:              ecdFeatures,
			RegimesPath:               ecdRegimes,
			Family:                    ecdFamily,
			Side:                      ecdSide,
			Symbol:                    ecdSymbol,
			CandlePath:                ecdCandlePath,
			Market:                    ecdMarket,
			Interval:                  ecdInterval,
			ConfigurationOverrideJSON: configOverride,
		})
		if err != nil {
			return err
		}

		outDir := filepath.Dir(ecdOut)
		stem := filepath.Join(outDir, fmt.Sprintf("phase10_3_%s_%s_%s_deep", report.Symbol, report.Family, report.Side))
		if ecdOut != "" && strings.HasSuffix(ecdOut, ".md") {
			stem = strings.TrimSuffix(ecdOut, ".md")
		}

		manifestPath := ecdManifest
		if manifestPath == "" && report.CandlePath != "" {
			probe := filepath.Join(report.CandlePath, "research_identity_manifest.json")
			if _, err := os.Stat(probe); err == nil {
				manifestPath = probe
			}
		}

		if err := writeDeepCandidateReport(ecdOut, report); err != nil {
			return err
		}
		if !ecdDiagnostics {
			return nil
		}

		bridge := rifbridge.NewBridge()
		identityRequest := report.researchIdentityRequest
		identityRequest.HistorianManifestPath = manifestPath
		_, err = bridge.EmitResearchDiagnostics(rifbridge.ResearchAssessment{
			Stem:                      stem,
			Classification:            report.FinalStatus,
			ClassificationGates:       rifClassificationGates(report.Gates),
			ExecutionSeriesGeneration: report.ExecutionSeriesID + ".v" + report.ExecutionSeriesVersion,
			IdentityRequest:           identityRequest,
		})
		if err != nil {
			return fmt.Errorf("emit local research diagnostics: %w", err)
		}
		return nil
	},
}

var evaluateResearchLeadsDeepCmd = &cobra.Command{
	Use:   "evaluate-research-leads-deep",
	Short: "Generate Phase 10.3 inventory, deep validations, and comparison",
	RunE: func(cmd *cobra.Command, args []string) error {
		leaderboard := erldLeaderboard
		if leaderboard == "" {
			leaderboard = filepath.Join("runs", "reports", "phase10_2c_candidate_leaderboard.json")
		}
		outDir := erldOutDir
		if outDir == "" {
			outDir = filepath.Join("runs", "reports")
		}
		inventory, err := buildPhase103Inventory(filepath.Join(strings.TrimSuffix(leaderboard, ".json")+".md"), leaderboard)
		if err != nil {
			return err
		}
		if err := writePhase103Inventory(outDir, inventory); err != nil {
			return err
		}
		configOverride, err := readOptionalResearchConfiguration(erldConfig)
		if err != nil {
			return err
		}

		reports := make([]DeepCandidateReport, 0, len(inventory.Rows))
		for _, row := range inventory.Rows {
			featurePath, ok := discoverPhaseArtifactPath("features", row.Symbol)
			if !ok {
				return fmt.Errorf("missing features for %s", row.Symbol)
			}
			regimePath, ok := discoverPhaseArtifactPath("regimes", row.Symbol)
			if !ok {
				return fmt.Errorf("missing regimes for %s", row.Symbol)
			}
			report, err := buildDeepCandidateReport(cmd.Context(), deepCandidateRequest{
				FeaturesPath:              featurePath,
				RegimesPath:               regimePath,
				Family:                    row.Family,
				Side:                      row.Side,
				Symbol:                    row.Symbol,
				CandlePath:                erldCandlePath,
				ConfigurationOverrideJSON: configOverride,
			})
			if err != nil {
				return err
			}

			stem := filepath.Join(outDir, fmt.Sprintf("phase10_3_%s_%s_%s_deep", report.Symbol, report.Family, report.Side))

			manifestPath := erldManifest
			if manifestPath == "" && report.CandlePath != "" {
				probe := filepath.Join(report.CandlePath, "research_identity_manifest.json")
				if _, err := os.Stat(probe); err == nil {
					manifestPath = probe
				}
			}

			out := stem + ".md"
			if err := writeDeepCandidateReport(out, report); err != nil {
				return err
			}
			if erldDiagnostics {
				bridge := rifbridge.NewBridge()
				identityRequest := report.researchIdentityRequest
				identityRequest.HistorianManifestPath = manifestPath
				_, err = bridge.EmitResearchDiagnostics(rifbridge.ResearchAssessment{
					Stem:                      stem,
					Classification:            report.FinalStatus,
					ClassificationGates:       rifClassificationGates(report.Gates),
					ExecutionSeriesGeneration: report.ExecutionSeriesID + ".v" + report.ExecutionSeriesVersion,
					IdentityRequest:           identityRequest,
				})
				if err != nil {
					return fmt.Errorf("emit local research diagnostics: %w", err)
				}
			}
			reports = append(reports, report)
		}
		comparison := buildPhase103Comparison(reports)
		return writePhase103Comparison(outDir, comparison)
	},
}

func rifClassificationGates(gates []DeepGateResult) []rifbridge.ClassificationGate {
	out := make([]rifbridge.ClassificationGate, len(gates))
	for i, gate := range gates {
		out[i] = rifbridge.ClassificationGate{Name: gate.Name, Passed: gate.Passed, Critical: gate.Critical}
	}
	return out
}

type deepCandidateRequest struct {
	FeaturesPath              string
	RegimesPath               string
	Family                    string
	Side                      string
	Symbol                    string
	CandlePath                string
	Market                    string
	Interval                  string
	ConfigurationOverrideJSON []byte
}

func init() {
	evaluateCandidateDeepCmd.Flags().StringVar(&ecdFeatures, "features", "", "Path to features JSON")
	evaluateCandidateDeepCmd.Flags().StringVar(&ecdRegimes, "regimes", "", "Path to regimes JSON")
	evaluateCandidateDeepCmd.Flags().StringVar(&ecdFamily, "family", "", "Candidate family")
	evaluateCandidateDeepCmd.Flags().StringVar(&ecdSide, "side", "", "Candidate side")
	evaluateCandidateDeepCmd.Flags().StringVar(&ecdSymbol, "symbol", "", "Candidate symbol")
	evaluateCandidateDeepCmd.Flags().StringVar(&ecdOut, "out", "", "Output Markdown path")
	evaluateCandidateDeepCmd.Flags().StringVar(&ecdCandlePath, "path", "", "Optional local parquet candle workdir")
	evaluateCandidateDeepCmd.Flags().StringVar(&ecdMarket, "market", "", "Optional market override")
	evaluateCandidateDeepCmd.Flags().StringVar(&ecdInterval, "interval", "", "Optional interval override")
	evaluateCandidateDeepCmd.Flags().StringVar(&ecdManifest, "dataset-manifest", "", "Path to Historian research_identity_manifest.json")
	evaluateCandidateDeepCmd.Flags().StringVar(&ecdConfig, "research-config", "", "Optional strict research-configuration override JSON")
	evaluateCandidateDeepCmd.Flags().BoolVar(&ecdDiagnostics, "research-diagnostics", false, "Emit explicit Engine-local research diagnostics")
	rootCmd.AddCommand(evaluateCandidateDeepCmd)

	evaluateResearchLeadsDeepCmd.Flags().StringVar(&erldLeaderboard, "leaderboard", "", "Phase 10.2C leaderboard JSON")
	evaluateResearchLeadsDeepCmd.Flags().StringVar(&erldOutDir, "out-dir", "", "Output directory")
	evaluateResearchLeadsDeepCmd.Flags().StringVar(&erldCandlePath, "path", "", "Optional local parquet candle workdir")
	evaluateResearchLeadsDeepCmd.Flags().StringVar(&erldManifest, "dataset-manifest", "", "Path to Historian research_identity_manifest.json")
	evaluateResearchLeadsDeepCmd.Flags().StringVar(&erldConfig, "research-config", "", "Optional strict research-configuration override JSON")
	evaluateResearchLeadsDeepCmd.Flags().BoolVar(&erldDiagnostics, "research-diagnostics", false, "Emit explicit Engine-local research diagnostics")
	rootCmd.AddCommand(evaluateResearchLeadsDeepCmd)
}

func buildPhase103Inventory(markdownPath, jsonPath string) (Phase103InventoryReport, error) {
	if _, err := os.ReadFile(markdownPath); err != nil {
		return Phase103InventoryReport{}, fmt.Errorf("read leaderboard markdown: %w", err)
	}
	data, err := os.ReadFile(jsonPath)
	if err != nil {
		return Phase103InventoryReport{}, fmt.Errorf("read leaderboard json: %w", err)
	}
	var leaderboard LeaderboardJSON
	if err := json.Unmarshal(data, &leaderboard); err != nil {
		return Phase103InventoryReport{}, fmt.Errorf("unmarshal leaderboard json: %w", err)
	}

	report := Phase103InventoryReport{
		Summary: Phase103InventorySummary{
			SourceMarkdown: markdownPath,
			SourceJSON:     jsonPath,
			ExpectedCount:  3,
			VerdictCounts:  leaderboard.VerdictCounts,
		},
	}
	for _, row := range leaderboard.Rows {
		if row.Verdict != verdictResearchLead {
			continue
		}
		gates := CandidateGateResults(
			row.H2Expectancy5bpsBps,
			row.H2PF5bps,
			row.FYPF5bps,
			row.EventCount,
			row.PositiveMonthCount,
			row.EntryDelay1cExpectancyBps,
			row.SingleMonthContributionPct,
			row.LeakageStatus,
		)
		report.Rows = append(report.Rows, Phase103InventoryRow{
			Symbol:                     row.Symbol,
			Family:                     row.Family,
			Side:                       strings.ToUpper(row.Side),
			EventCount:                 row.EventCount,
			H2PF5bps:                   row.H2PF5bps,
			H2Expectancy5bpsBps:        row.H2Expectancy5bpsBps,
			FYPF5bps:                   row.FYPF5bps,
			FYExpectancy5bpsBps:        row.FYExpectancy5bpsBps,
			PositiveMonthCount:         row.PositiveMonthCount,
			EntryDelay1cExpectancyBps:  row.EntryDelay1cExpectancyBps,
			SingleMonthContributionPct: row.SingleMonthContributionPct,
			LeakageStatus:              row.LeakageStatus,
			PassedGates:                passedGateNames(gates),
		})
	}
	sort.Slice(report.Rows, func(i, j int) bool {
		if report.Rows[i].Symbol != report.Rows[j].Symbol {
			return report.Rows[i].Symbol < report.Rows[j].Symbol
		}
		if report.Rows[i].Family != report.Rows[j].Family {
			return report.Rows[i].Family < report.Rows[j].Family
		}
		return report.Rows[i].Side < report.Rows[j].Side
	})
	report.Summary.ResearchLeadCount = len(report.Rows)
	report.Summary.CountMismatch = report.Summary.ResearchLeadCount != report.Summary.ExpectedCount
	return report, nil
}

func writePhase103Inventory(outDir string, report Phase103InventoryReport) error {
	jsonPath := filepath.Join(outDir, "phase10_3_research_leads_inventory.json")
	mdPath := filepath.Join(outDir, "phase10_3_research_leads_inventory.md")
	if err := writeJSONFile(jsonPath, report); err != nil {
		return err
	}
	if err := os.WriteFile(mdPath, []byte(buildPhase103InventoryMD(report)), 0644); err != nil {
		return fmt.Errorf("write inventory markdown: %w", err)
	}
	return nil
}

func buildPhase103InventoryMD(report Phase103InventoryReport) string {
	var sb strings.Builder
	sb.WriteString("# Phase 10.3 Research Leads Inventory\n\n")
	sb.WriteString("## Summary\n")
	sb.WriteString(fmt.Sprintf("- Source Markdown: `%s`\n", report.Summary.SourceMarkdown))
	sb.WriteString(fmt.Sprintf("- Source JSON: `%s`\n", report.Summary.SourceJSON))
	sb.WriteString(fmt.Sprintf("- Research lead count: %d\n", report.Summary.ResearchLeadCount))
	sb.WriteString(fmt.Sprintf("- Expected count: %d\n", report.Summary.ExpectedCount))
	sb.WriteString(fmt.Sprintf("- Count mismatch: `%t`\n\n", report.Summary.CountMismatch))
	sb.WriteString("## Verdict Counts\n")
	for _, verdict := range verdictOrder {
		sb.WriteString(fmt.Sprintf("- %s: %d\n", verdict, report.Summary.VerdictCounts[verdict]))
	}
	sb.WriteString("\n## Research Leads\n")
	sb.WriteString("| Symbol | Family | Side | Events | H2 PF 5bps | H2 Exp 5bps (bps) | FY PF 5bps | FY Exp 5bps (bps) | Positive Months | Delay 1c Exp (bps) | Single Month % | Leakage | Passed Gates |\n")
	sb.WriteString("|---|---|---|---:|---:|---:|---:|---:|---:|---:|---:|---|---|\n")
	for _, row := range report.Rows {
		sb.WriteString(fmt.Sprintf("| %s | %s | %s | %d | %.4f | %.4f | %.4f | %.4f | %d | %.4f | %.2f | %s | %s |\n",
			row.Symbol,
			row.Family,
			row.Side,
			row.EventCount,
			row.H2PF5bps,
			row.H2Expectancy5bpsBps,
			row.FYPF5bps,
			row.FYExpectancy5bpsBps,
			row.PositiveMonthCount,
			row.EntryDelay1cExpectancyBps,
			row.SingleMonthContributionPct,
			row.LeakageStatus,
			strings.Join(row.PassedGates, "; ")))
	}
	sb.WriteString("\n")
	return sb.String()
}

func buildDeepCandidateReport(ctx context.Context, req deepCandidateRequest) (DeepCandidateReport, error) {
	rows, err := features.ReadRowsJSON(req.FeaturesPath)
	if err != nil {
		return DeepCandidateReport{}, fmt.Errorf("read features: %w", err)
	}
	labels, err := regime.ReadLabelsJSON(req.RegimesPath)
	if err != nil {
		return DeepCandidateReport{}, fmt.Errorf("read regimes: %w", err)
	}
	if len(rows) == 0 {
		return DeepCandidateReport{}, errors.New("features empty")
	}
	sort.Slice(rows, func(i, j int) bool { return rows[i].EventTimeMS < rows[j].EventTimeMS })
	sort.Slice(labels, func(i, j int) bool { return labels[i].AvailableAtMS < labels[j].AvailableAtMS })

	market := req.Market
	if market == "" {
		market = rows[0].Market
	}
	interval := req.Interval
	if interval == "" {
		interval = rows[0].Interval
	}
	configuration, err := researchidentity.ResolveConfiguration(researchidentity.ConfigurationContext{
		Symbol:            req.Symbol,
		Market:            market,
		Interval:          interval,
		EvaluationStartMS: rows[0].EventTimeMS,
		EvaluationEndMS:   rows[len(rows)-1].EventTimeMS,
	}, req.ConfigurationOverrideJSON)
	if err != nil {
		return DeepCandidateReport{}, fmt.Errorf("resolve effective research configuration: %w", err)
	}
	candlePath, err := inferDeepCandlePath(req.CandlePath, market, interval, req.Symbol)
	if err != nil {
		return DeepCandidateReport{}, err
	}
	candles, consumedDatasetPaths, err := loadDeepCandlesWithInventory(ctx, candlePath, market, interval, req.Symbol, rows, deepMaximumHorizon(configuration))
	if err != nil {
		return DeepCandidateReport{}, err
	}
	sort.Slice(candles, func(i, j int) bool { return candles[i].OpenTimeMS < candles[j].OpenTimeMS })

	registry, err := researchidentity.DefaultRegistry()
	if err != nil {
		return DeepCandidateReport{}, fmt.Errorf("load candidate registry: %w", err)
	}
	registration, err := registry.Lookup(req.Family, req.Side)
	if err != nil {
		return DeepCandidateReport{}, err
	}
	family := registration.Family
	side := registration.Side
	events, audit, leakageIssues, err := generateDeepCandidateEventsConfigured(rows, labels, candles, family, side, configuration.ClusterSeparationMinutes)
	if err != nil {
		return DeepCandidateReport{}, err
	}
	if len(events) == 0 {
		return DeepCandidateReport{}, fmt.Errorf("no accepted candidates for %s %s_%s", req.Symbol, family, side)
	}

	report := DeepCandidateReport{
		Symbol:                   req.Symbol,
		Family:                   family,
		Side:                     side,
		SourceFeatures:           req.FeaturesPath,
		SourceRegimes:            req.RegimesPath,
		CandlePath:               candlePath,
		ExpectancyUnit:           expectancyUnitBps,
		CostUnit:                 "basis_points",
		CandidateGenerationAudit: audit,
		RegimeBreakdown:          make(map[string][]DeepRegimeMetric),
		LeakageStatus:            "PASS",
		ExecutionSeriesID:        executionseries.ID,
		ExecutionSeriesVersion:   executionseries.Version,
	}
	if len(candles) > 0 {
		report.TruncationReport.LoadedFirstCandleMS = candles[0].OpenTimeMS
		report.TruncationReport.LoadedLastCandleMS = candles[len(candles)-1].OpenTimeMS
	}
	if len(leakageIssues) > 0 {
		report.LeakageStatus = "FAIL"
		report.LeakageIssues = leakageIssues
	}

	for _, horizon := range configuration.ForwardHorizonsMinutes {
		set := deepReturnsForEvents(events, candles, horizon, 0, 0, 0)
		metric := deepHorizonMetric(horizon, set)
		report.ForwardReturns = append(report.ForwardReturns, metric)
		report.TruncationReport.ForwardReturnTruncations += metric.TruncatedCount
	}
	for _, cost := range configuration.CostHaircutsBPS {
		set := deepReturnsForEvents(events, candles, configuration.CostHaircutHorizonMinutes, cost, 0, 0)
		m := deepMetricFromBps(set.ReturnsBps)
		report.CostHaircuts = append(report.CostHaircuts, DeepCostHaircutMetric{
			CostBps:        cost,
			EventCount:     len(set.ReturnsBps),
			PF:             m.PF,
			ExpectancyBps:  m.Expectancy,
			WinRate:        m.WinRate,
			PFAbove105:     m.PF >= 1.05,
			PFAbove110:     m.PF >= 1.10,
			PFAbove120:     m.PF >= 1.20,
			TruncatedCount: set.TruncatedCount,
		})
		report.TruncationReport.ForwardReturnTruncations += set.TruncatedCount
	}
	for _, delay := range configuration.EntryDelayCandles {
		set := deepReturnsForEvents(events, candles, configuration.EntryDelayHorizonMinutes, configuration.EntryDelayCostBPS, delay, 0)
		m := deepMetricFromBps(set.ReturnsBps)
		report.EntryDelays = append(report.EntryDelays, DeepEntryDelayMetric{
			DelayCandles:   delay,
			EventCount:     len(set.ReturnsBps),
			ExpectancyBps:  m.Expectancy,
			PFAfter5bps:    m.PF,
			WinRate:        m.WinRate,
			TruncatedCount: set.TruncatedCount,
		})
		report.TruncationReport.ForwardReturnTruncations += set.TruncatedCount
	}
	seriesSet := deepReturnsForEvents(events, candles, configuration.SeriesHorizonMinutes, configuration.SeriesCostBPS, configuration.SeriesEntryDelayCandles, candles[len(candles)-1].OpenTimeMS)
	for i, value := range seriesSet.ReturnsBps {
		report.RIFReturns = append(report.RIFReturns, value/10000.0)
		report.RIFTimestamps = append(report.RIFTimestamps, seriesSet.EventTimestamps[i])
	}
	for _, window := range configuration.ExcursionWindowsMinutes {
		m := deepExcursionMetric(events, candles, window, side)
		report.MFEMAE = append(report.MFEMAE, m)
		report.TruncationReport.ExcursionTruncations += m.TruncatedCount
	}
	for _, bracket := range configuration.Brackets {
		report.Brackets = append(report.Brackets, deepBracketMetricConfigured(events, candles, side, bracket.Name, bracket.TPBPS, bracket.SLBPS, configuration.BracketWindowMinutes, configuration.BracketCostBPS, configuration.GateThresholds))
	}
	for _, period := range append(monthPeriods(rows), configuration.StabilityAggregatePeriods...) {
		m := deepStabilityMetricConfigured(events, candles, period, configuration)
		report.Stability = append(report.Stability, m)
		report.TruncationReport.StabilityTruncations += m.TruncatedCount
	}
	for _, group := range configuration.RegimeGroups {
		report.RegimeBreakdown[group] = deepRegimeBreakdownConfigured(events, candles, group, configuration)
	}

	report.ComparisonMetrics = buildDeepComparisonMetrics(report)
	report.Gates = deepAcceptanceGatesConfigured(report, configuration.GateThresholds)
	report.PassedGates, report.FailedGates = splitDeepGates(report.Gates)
	report.FinalStatus = deepFinalStatus(report.Gates)
	if !phase103AllowedStatuses[report.FinalStatus] {
		return DeepCandidateReport{}, fmt.Errorf("invalid final status %q", report.FinalStatus)
	}
	if !deepAnyBracketNotCatastrophic(report.Brackets) {
		report.FixedHoldOnlyJustification = "No bracket model cleared the not-catastrophic threshold; fixed-hold evidence is reported but bracket gate remains failed."
	}
	repositoryRoot, err := researchidentity.FindRepositoryRoot("")
	if err != nil {
		return DeepCandidateReport{}, fmt.Errorf("resolve Engine repository root: %w", err)
	}
	eventTimestamps := make([]int64, len(report.RIFTimestamps))
	copy(eventTimestamps, report.RIFTimestamps)
	report.researchIdentityRequest = researchidentity.DerivationRequest{
		RepositoryRoot:            repositoryRoot,
		CandidateFamily:           family,
		CandidateSide:             side,
		Configuration:             configuration,
		DatasetRoot:               candlePath,
		FeatureArtifactPath:       req.FeaturesPath,
		RegimeArtifactPath:        req.RegimesPath,
		ConsumedDatasetPaths:      append([]string(nil), consumedDatasetPaths...),
		FeatureRows:               append([]features.Row(nil), rows...),
		RegimeLabels:              append([]regime.Label(nil), labels...),
		Candles:                   append([]protocol.Candle(nil), candles...),
		EvaluationEventTimestamps: eventTimestamps,
		Returns:                   append([]float64(nil), report.RIFReturns...),
		Timestamps:                append([]int64(nil), report.RIFTimestamps...),
	}
	return report, nil
}

func inferDeepCandlePath(explicit, market, interval, symbol string) (string, error) {
	candidates := []string{}
	if explicit != "" {
		candidates = append(candidates, explicit)
	}
	candidates = append(candidates,
		defaultHistorianWorkdir,
		filepath.Join(".ak-engine", "cache", "r2"),
	)
	for _, p := range candidates {
		if p == "" {
			continue
		}
		probe := filepath.Join(p, "candles", market, interval, "symbol="+symbol)
		if st, err := os.Stat(probe); err == nil && st.IsDir() {
			return p, nil
		}
	}
	return "", fmt.Errorf("could not infer local parquet candle path for %s/%s/%s; pass --path", market, interval, symbol)
}

func loadDeepCandles(ctx context.Context, path, market, interval, symbol string, rows []features.Row) ([]protocol.Candle, error) {
	candles, _, err := loadDeepCandlesWithInventory(ctx, path, market, interval, symbol, rows, 240)
	return candles, err
}

func loadDeepCandlesWithInventory(ctx context.Context, path, market, interval, symbol string, rows []features.Row, maximumHorizonMinutes int) ([]protocol.Candle, []string, error) {
	if len(rows) == 0 {
		return nil, nil, errors.New("features empty")
	}
	if maximumHorizonMinutes <= 0 {
		return nil, nil, errors.New("maximum candle horizon must be positive")
	}
	from := time.UnixMilli(rows[0].EventTimeMS).UTC()
	to := time.UnixMilli(rows[len(rows)-1].EventTimeMS + int64(maximumHorizonMinutes)*60*1000).UTC()
	return data.NewLocalParquetSource().LoadCandlesWithInventory(ctx, data.CandleRequest{
		Source:   "local-parquet",
		Path:     path,
		Market:   market,
		Symbol:   symbol,
		Interval: interval,
		From:     from,
		To:       to,
	})
}

func deepMaximumHorizon(config researchidentity.ResolvedResearchConfiguration) int {
	maximum := max(config.CostHaircutHorizonMinutes, config.EntryDelayHorizonMinutes, config.SeriesHorizonMinutes, config.BracketWindowMinutes, config.StabilityHorizonMinutes, config.RegimeHorizonMinutes)
	for _, values := range [][]int{config.ForwardHorizonsMinutes, config.ExcursionWindowsMinutes} {
		for _, value := range values {
			maximum = max(maximum, value)
		}
	}
	return maximum
}

func readOptionalResearchConfiguration(path string) ([]byte, error) {
	if strings.TrimSpace(path) == "" {
		return nil, nil
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read research configuration: %w", err)
	}
	return data, nil
}

func generateDeepCandidateEvents(rows []features.Row, labels []regime.Label, candles []protocol.Candle, family, side string) ([]deepCandidateEvent, DeepCandidateAudit, []string, error) {
	return generateDeepCandidateEventsConfigured(rows, labels, candles, family, side, 60)
}

func generateDeepCandidateEventsConfigured(rows []features.Row, labels []regime.Label, candles []protocol.Candle, family, side string, clusterSeparationMinutes int) ([]deepCandidateEvent, DeepCandidateAudit, []string, error) {
	if clusterSeparationMinutes <= 0 {
		return nil, DeepCandidateAudit{}, nil, errors.New("cluster separation must be positive")
	}
	if err := validateDeepFeatureRows(rows); err != nil {
		return nil, DeepCandidateAudit{}, nil, err
	}
	if err := validateDeepLabels(labels); err != nil {
		return nil, DeepCandidateAudit{}, nil, err
	}
	if err := validateDeepCandles(candles); err != nil {
		return nil, DeepCandidateAudit{}, nil, err
	}
	candleIndex := make(map[int64]int, len(candles))
	for i, candle := range candles {
		candleIndex[candle.OpenTimeMS] = i
	}

	audit := DeepCandidateAudit{Family: family, Side: side}
	var events []deepCandidateEvent
	var leakageIssues []string
	lastAccepted := int64(0)
	for i, row := range rows {
		if row.Warmup {
			continue
		}
		decisionTimeMS := row.AvailableAtMS
		labelIdx := sort.Search(len(labels), func(k int) bool {
			return labels[k].AvailableAtMS > decisionTimeMS
		})
		if labelIdx == 0 {
			continue
		}
		label := labels[labelIdx-1]
		if label.AvailableAtMS > decisionTimeMS {
			leakageIssues = append(leakageIssues, fmt.Sprintf("regime available_at_ms > decision time at row %d", i))
			continue
		}

		base, directionOK, betaOK, betaApplicable := deepCandidateRule(row, label, family, side)
		if !base {
			continue
		}
		audit.TriggerRows++
		if label.Volatility == "compressed" || label.Composite == "compressed_range" {
			audit.CompressedRangeRows++
		}
		if label.Volatility == "shock" {
			audit.ShockRows++
		}
		if !directionOK {
			audit.CandidatesRejectedByDirectionRule++
			continue
		}
		audit.EligibleSideCandidates++
		if side == "SHORT" {
			audit.EligibleShortCandidates++
		} else if side == "LONG" {
			audit.EligibleLongCandidates++
		}
		if betaApplicable && !betaOK {
			audit.CandidatesRejectedByBTCETHBetaConfirmation++
			continue
		}
		ci, ok := candleIndex[row.EventTimeMS]
		if !ok {
			return nil, DeepCandidateAudit{}, leakageIssues, fmt.Errorf("missing candle for accepted candidate at %d", row.EventTimeMS)
		}
		audit.AcceptedCandidates++
		if lastAccepted == 0 || row.EventTimeMS-lastAccepted >= int64(clusterSeparationMinutes)*60*1000 {
			audit.UniqueEventClusters++
		} else {
			audit.ClusteredCandidates++
		}
		lastAccepted = row.EventTimeMS
		events = append(events, deepCandidateEvent{
			FeatureIndex:   i,
			CandleIndex:    ci,
			EventTimeMS:    row.EventTimeMS,
			DecisionTimeMS: decisionTimeMS,
			Family:         family,
			Side:           side,
			Label:          label,
		})
	}
	return events, audit, leakageIssues, nil
}

func deepCandidateRule(row features.Row, label regime.Label, family, side string) (base, directionOK, betaOK, betaApplicable bool) {
	betaOK = true
	switch family {
	case "CompressionBreakout":
		base = label.Volatility == "compressed" || label.Composite == "compressed_range"
		if side == "LONG" {
			directionOK = row.Close > row.EMA20
			betaOK = label.MarketBeta == "btc_up"
		} else {
			directionOK = row.Close < row.EMA20
			betaOK = label.MarketBeta == "btc_down"
		}
		betaApplicable = true
	case "ShockFade":
		base = label.Volatility == "shock"
		if side == "LONG" {
			directionOK = row.Return5 < 0
		} else {
			directionOK = row.Return5 > 0
		}
	case "TrendContinuation":
		base = true
		if side == "LONG" {
			directionOK = row.Close > row.EMA20 && row.EMA20 > row.EMA50 && row.TrendSlope20 > 0
		} else {
			directionOK = row.Close < row.EMA20 && row.EMA20 < row.EMA50 && row.TrendSlope20 < 0
		}
	case "VolumeMomentum":
		base = label.Liquidity == "heavy"
		if side == "LONG" {
			directionOK = row.Return5 > 0 && row.Return15 > 0
		} else {
			directionOK = row.Return5 < 0 && row.Return15 < 0
		}
	case "BetaAgrees":
		base = true
		if side == "LONG" {
			directionOK = row.Return5 > 0
			betaOK = label.MarketBeta == "btc_up"
		} else {
			directionOK = row.Return5 < 0
			betaOK = label.MarketBeta == "btc_down"
		}
		betaApplicable = true
	case "BetaDiverges":
		base = true
		if side == "LONG" {
			directionOK = row.Return5 > 0
			betaOK = label.MarketBeta == "btc_down"
		} else {
			directionOK = row.Return5 < 0
			betaOK = label.MarketBeta == "btc_up"
		}
		betaApplicable = true
	default:
		return false, false, false, false
	}
	return base, directionOK, betaOK, betaApplicable
}

func validateDeepFeatureRows(rows []features.Row) error {
	for i := range rows {
		if err := (temporal.Observation{
			SourceEventMS:     rows[i].EventTimeMS,
			SourceAvailableMS: rows[i].AvailableAtMS,
			DecisionMS:        rows[i].AvailableAtMS,
		}).Validate(); err != nil {
			return fmt.Errorf("feature row %d temporal contract: %w", i, err)
		}
		if i > 0 && rows[i].EventTimeMS <= rows[i-1].EventTimeMS {
			return fmt.Errorf("duplicate timestamps break evaluation")
		}
		if i > 0 && rows[i].AvailableAtMS <= rows[i-1].AvailableAtMS {
			return fmt.Errorf("feature availability timestamps are not strictly increasing")
		}
	}
	return nil
}

func validateDeepLabels(labels []regime.Label) error {
	for i := range labels {
		if err := (temporal.Observation{
			SourceEventMS:     labels[i].EventTimeMS,
			SourceAvailableMS: labels[i].AvailableAtMS,
			DecisionMS:        labels[i].AvailableAtMS,
		}).Validate(); err != nil {
			return fmt.Errorf("regime row %d temporal contract: %w", i, err)
		}
		if i > 0 && labels[i].AvailableAtMS <= labels[i-1].AvailableAtMS {
			return fmt.Errorf("duplicate timestamps break evaluation")
		}
	}
	return nil
}

func validateDeepCandles(candles []protocol.Candle) error {
	for i := range candles {
		if i > 0 && candles[i].OpenTimeMS <= candles[i-1].OpenTimeMS {
			return fmt.Errorf("duplicate timestamps break evaluation")
		}
	}
	return nil
}

func deepReturnsForEvents(events []deepCandidateEvent, candles []protocol.Candle, horizonMinutes int, costBps float64, entryDelayCandles int, periodEndMS int64) deepReturnSet {
	var out deepReturnSet
	for _, event := range events {
		window, err := executionseries.Resolve(
			event.EventTimeMS, event.DecisionTimeMS, event.DecisionTimeMS,
			candles, entryDelayCandles, horizonMinutes, periodEndMS,
		)
		if err != nil {
			out.TruncatedCount++
			continue
		}
		ret := deepSignedReturnBps(window.EntryPrice, window.ExitPrice, event.Side) - costBps
		out.ReturnsBps = append(out.ReturnsBps, ret)
		out.EventTimestamps = append(out.EventTimestamps, event.EventTimeMS)
	}
	return out
}

func deepSignedReturnBps(entry, exit float64, side string) float64 {
	if entry == 0 {
		return 0
	}
	if strings.ToUpper(side) == "SHORT" {
		return (entry - exit) / entry * 10000.0
	}
	return (exit - entry) / entry * 10000.0
}

func candleIndexAtOrAfter(candles []protocol.Candle, start int, targetMS int64) int {
	offset := sort.Search(len(candles)-start, func(i int) bool {
		return candles[start+i].OpenTimeMS >= targetMS
	})
	idx := start + offset
	if idx >= len(candles) {
		return -1
	}
	return idx
}

type deepBasicMetric struct {
	Count      int
	PF         float64
	Expectancy float64
	WinRate    float64
	NetBps     float64
}

func deepMetricFromBps(rets []float64) deepBasicMetric {
	var m deepBasicMetric
	m.Count = len(rets)
	if len(rets) == 0 {
		return m
	}
	var wins int
	var grossWin, grossLoss, net float64
	for _, ret := range rets {
		net += ret
		if ret > 0 {
			wins++
			grossWin += ret
		} else if ret < 0 {
			grossLoss += -ret
		}
	}
	if grossLoss > 0 {
		m.PF = grossWin / grossLoss
	} else if grossWin > 0 {
		m.PF = 999.0
	}
	m.Expectancy = net / float64(len(rets))
	m.WinRate = float64(wins) / float64(len(rets))
	m.NetBps = net
	return m
}

func deepHorizonMetric(horizon int, set deepReturnSet) DeepHorizonMetric {
	m := deepMetricFromBps(set.ReturnsBps)
	sorted := append([]float64(nil), set.ReturnsBps...)
	sort.Float64s(sorted)
	return DeepHorizonMetric{
		HorizonMinutes:     horizon,
		EventCount:         len(set.ReturnsBps),
		AverageReturnBps:   m.Expectancy,
		MedianReturnBps:    deepPercentileSorted(sorted, 0.50),
		WinRate:            m.WinRate,
		PFProxy:            m.PF,
		ExpectancyBps:      m.Expectancy,
		P25ReturnBps:       deepPercentileSorted(sorted, 0.25),
		P75ReturnBps:       deepPercentileSorted(sorted, 0.75),
		Worst5PctReturnBps: deepPercentileSorted(sorted, 0.05),
		Best5PctReturnBps:  deepPercentileSorted(sorted, 0.95),
		TruncatedCount:     set.TruncatedCount,
	}
}

func deepPercentileSorted(sorted []float64, p float64) float64 {
	if len(sorted) == 0 {
		return 0
	}
	if p <= 0 {
		return sorted[0]
	}
	if p >= 1 {
		return sorted[len(sorted)-1]
	}
	idx := int(math.Floor(p * float64(len(sorted)-1)))
	return sorted[idx]
}

func deepExcursionMetric(events []deepCandidateEvent, candles []protocol.Candle, windowMinutes int, side string) DeepExcursionMetric {
	var mfes, maes []float64
	var reach5, reach10, reach15 float64
	truncated := 0
	for _, event := range events {
		window, err := executionseries.Resolve(event.EventTimeMS, event.DecisionTimeMS, event.DecisionTimeMS, candles, 0, windowMinutes, 0)
		if err != nil {
			truncated++
			continue
		}
		executionCandles := candles[window.EntryIndex : window.EndIndex+1]
		mfe, mae := deepMFEAndMAEForWindowAtEntry(executionCandles, window.EntryPrice, side)
		mfes = append(mfes, mfe)
		maes = append(maes, mae)
		if deepThresholdFirstAtEntry(executionCandles, window.EntryPrice, side, 5, 5) {
			reach5++
		}
		if deepThresholdFirstAtEntry(executionCandles, window.EntryPrice, side, 10, 5) {
			reach10++
		}
		if deepThresholdFirstAtEntry(executionCandles, window.EntryPrice, side, 15, 10) {
			reach15++
		}
	}
	sort.Float64s(mfes)
	sort.Float64s(maes)
	avgMFE := deepAverage(mfes)
	avgMAE := deepAverage(maes)
	ratio := 0.0
	if avgMAE > 0 {
		ratio = avgMFE / avgMAE
	}
	n := float64(len(mfes))
	if n == 0 {
		n = 1
	}
	return DeepExcursionMetric{
		WindowMinutes:              windowMinutes,
		EventCount:                 len(mfes),
		MedianMFEBps:               deepPercentileSorted(mfes, 0.50),
		MedianMAEBps:               deepPercentileSorted(maes, 0.50),
		AverageMFEBps:              avgMFE,
		AverageMAEBps:              avgMAE,
		MFEMAERatio:                ratio,
		Reach5BpsBeforeMinus5Pct:   reach5 / n * 100.0,
		Reach10BpsBeforeMinus5Pct:  reach10 / n * 100.0,
		Reach15BpsBeforeMinus10Pct: reach15 / n * 100.0,
		TruncatedCount:             truncated,
	}
}

func deepMFEAndMAEForWindow(candles []protocol.Candle, side string) (float64, float64) {
	if len(candles) == 0 || candles[0].Close == 0 {
		return 0, 0
	}
	return deepMFEAndMAEForWindowAtEntry(candles, candles[0].Close, side)
}

func deepMFEAndMAEForWindowAtEntry(candles []protocol.Candle, entry float64, side string) (float64, float64) {
	if len(candles) == 0 || entry <= 0 {
		return 0, 0
	}
	high := candles[0].High
	low := candles[0].Low
	for _, candle := range candles {
		if candle.High > high {
			high = candle.High
		}
		if candle.Low < low {
			low = candle.Low
		}
	}
	return deepMFEAndMAEBps(entry, high, low, side)
}

func deepMFEAndMAEBps(entry, highestHigh, lowestLow float64, side string) (float64, float64) {
	if entry == 0 {
		return 0, 0
	}
	if strings.ToUpper(side) == "SHORT" {
		return (entry - lowestLow) / entry * 10000.0, (highestHigh - entry) / entry * 10000.0
	}
	return (highestHigh - entry) / entry * 10000.0, (entry - lowestLow) / entry * 10000.0
}

func deepThresholdFirst(candles []protocol.Candle, side string, favorableBps, adverseBps float64) bool {
	return simulateDeepBracketEvent(candles, side, favorableBps, adverseBps, 0).Outcome == "win"
}

func deepThresholdFirstAtEntry(candles []protocol.Candle, entry float64, side string, favorableBps, adverseBps float64) bool {
	return simulateDeepBracketEventAtEntry(candles, entry, side, favorableBps, adverseBps, 0).Outcome == "win"
}

func deepBracketMetric(events []deepCandidateEvent, candles []protocol.Candle, side, name string, tpBps, slBps float64) DeepBracketMetric {
	return deepBracketMetricConfigured(events, candles, side, name, tpBps, slBps, 240, 5, researchidentity.GateThresholds{MinimumBracketPF: 0.50, MinimumBracketExpectancyBPS: -10})
}

func deepBracketMetricConfigured(events []deepCandidateEvent, candles []protocol.Candle, side, name string, tpBps, slBps float64, windowMinutes int, costBPS float64, thresholds researchidentity.GateThresholds) DeepBracketMetric {
	var returns []float64
	var wins, unresolved, holdSum int
	tradeCount := 0
	for _, event := range events {
		window, err := executionseries.Resolve(event.EventTimeMS, event.DecisionTimeMS, event.DecisionTimeMS, candles, 0, windowMinutes, 0)
		if err != nil {
			continue
		}
		tradeCount++
		res := simulateDeepBracketEventAtEntry(candles[window.EntryIndex:window.EndIndex+1], window.EntryPrice, side, tpBps, slBps, costBPS)
		returns = append(returns, res.ReturnBps)
		if res.Outcome == "win" {
			wins++
		}
		if res.Outcome == "unresolved" {
			unresolved++
		}
		holdSum += res.HoldMinutes
	}
	m := deepMetricFromBps(returns)
	avgHold := 0.0
	if tradeCount > 0 {
		avgHold = float64(holdSum) / float64(tradeCount)
	}
	return DeepBracketMetric{
		Name:                      name,
		TPBps:                     tpBps,
		SLBps:                     slBps,
		TradeCount:                tradeCount,
		WinRate:                   float64(wins) / math.Max(1, float64(tradeCount)),
		PFAfter5bps:               m.PF,
		NetExpectancyBpsAfter5bps: m.Expectancy,
		AverageHoldMinutes:        avgHold,
		UnresolvedCount:           unresolved,
		NotCatastrophicAfter5bps:  m.PF >= thresholds.MinimumBracketPF && m.Expectancy >= thresholds.MinimumBracketExpectancyBPS,
	}
}

func simulateDeepBracketEvent(candles []protocol.Candle, side string, tpBps, slBps, costBps float64) deepBracketResult {
	if len(candles) == 0 || candles[0].Close == 0 {
		return deepBracketResult{Outcome: "unresolved", ReturnBps: -costBps}
	}
	return simulateDeepBracketEventAtEntry(candles, candles[0].Close, side, tpBps, slBps, costBps)
}

func simulateDeepBracketEventAtEntry(candles []protocol.Candle, entry float64, side string, tpBps, slBps, costBps float64) deepBracketResult {
	if len(candles) == 0 || entry <= 0 {
		return deepBracketResult{Outcome: "unresolved", ReturnBps: -costBps}
	}
	side = strings.ToUpper(side)
	for i, candle := range candles {
		if side == "SHORT" {
			slTouched := (candle.High-entry)/entry*10000.0 >= slBps
			tpTouched := (entry-candle.Low)/entry*10000.0 >= tpBps
			if slTouched {
				return deepBracketResult{Outcome: "loss", ReturnBps: -slBps - costBps, HoldMinutes: i}
			}
			if tpTouched {
				return deepBracketResult{Outcome: "win", ReturnBps: tpBps - costBps, HoldMinutes: i}
			}
		} else {
			slTouched := (entry-candle.Low)/entry*10000.0 >= slBps
			tpTouched := (candle.High-entry)/entry*10000.0 >= tpBps
			if slTouched {
				return deepBracketResult{Outcome: "loss", ReturnBps: -slBps - costBps, HoldMinutes: i}
			}
			if tpTouched {
				return deepBracketResult{Outcome: "win", ReturnBps: tpBps - costBps, HoldMinutes: i}
			}
		}
	}
	exit := candles[len(candles)-1].Close
	return deepBracketResult{
		Outcome:     "unresolved",
		ReturnBps:   deepSignedReturnBps(entry, exit, side) - costBps,
		HoldMinutes: len(candles) - 1,
	}
}

func deepAverage(values []float64) float64 {
	if len(values) == 0 {
		return 0
	}
	var sum float64
	for _, v := range values {
		sum += v
	}
	return sum / float64(len(values))
}

func monthPeriods(rows []features.Row) []string {
	if len(rows) == 0 {
		return nil
	}
	start := time.UnixMilli(rows[0].EventTimeMS).UTC()
	end := time.UnixMilli(rows[len(rows)-1].EventTimeMS).UTC()
	cur := time.Date(start.Year(), start.Month(), 1, 0, 0, 0, 0, time.UTC)
	last := time.Date(end.Year(), end.Month(), 1, 0, 0, 0, 0, time.UTC)
	var periods []string
	for !cur.After(last) {
		periods = append(periods, cur.Format("2006-01"))
		cur = cur.AddDate(0, 1, 0)
	}
	return periods
}

func deepStabilityMetric(events []deepCandidateEvent, candles []protocol.Candle, period string) DeepStabilityMetric {
	return deepStabilityMetricWithParameters(events, candles, period, 2023, 60, 5, 0)
}

func deepStabilityMetricConfigured(events []deepCandidateEvent, candles []protocol.Candle, period string, config researchidentity.ResolvedResearchConfiguration) DeepStabilityMetric {
	year := time.UnixMilli(config.EvaluationStartMS).UTC().Year()
	return deepStabilityMetricWithParameters(events, candles, period, year, config.StabilityHorizonMinutes, config.StabilityCostBPS, config.StabilityEntryDelay)
}

func deepStabilityMetricWithParameters(events []deepCandidateEvent, candles []protocol.Candle, period string, year, horizonMinutes int, costBPS float64, entryDelay int) DeepStabilityMetric {
	start, end := deepPeriodBoundsForYear(period, year)
	var filtered []deepCandidateEvent
	for _, event := range events {
		t := time.UnixMilli(event.EventTimeMS).UTC()
		if !t.Before(start) && !t.After(end) {
			filtered = append(filtered, event)
		}
	}
	set := deepReturnsForEvents(filtered, candles, horizonMinutes, costBPS, entryDelay, end.UnixMilli())
	m := deepMetricFromBps(set.ReturnsBps)
	monthMetrics := deepMonthNetsWithParameters(filtered, candles, start, end, horizonMinutes, costBPS, entryDelay)
	bestMonth, worstMonth, positiveMonths, singleMonthPct := summarizeDeepMonths(monthMetrics)
	return DeepStabilityMetric{
		Period:                     period,
		EventCount:                 len(set.ReturnsBps),
		PFAfter5bps:                m.PF,
		ExpectancyBpsAfter5bps:     m.Expectancy,
		WinRate:                    m.WinRate,
		BestMonth:                  bestMonth,
		WorstMonth:                 worstMonth,
		SingleMonthContributionPct: singleMonthPct,
		PositiveMonthCount:         positiveMonths,
		TruncatedCount:             set.TruncatedCount,
	}
}

func deepPeriodBounds(period string) (time.Time, time.Time) {
	return deepPeriodBoundsForYear(period, 2023)
}

func deepPeriodBoundsForYear(period string, year int) (time.Time, time.Time) {
	if month, err := time.Parse("2006-01", period); err == nil {
		start := time.Date(month.Year(), month.Month(), 1, 0, 0, 0, 0, time.UTC)
		return start, start.AddDate(0, 1, 0).Add(-time.Millisecond)
	}
	monthByName := map[string]time.Month{
		"Jan": time.January, "Feb": time.February, "Mar": time.March, "Apr": time.April,
		"May": time.May, "Jun": time.June, "Jul": time.July, "Aug": time.August,
		"Sep": time.September, "Oct": time.October, "Nov": time.November, "Dec": time.December,
	}
	if m, ok := monthByName[period]; ok {
		start := time.Date(year, m, 1, 0, 0, 0, 0, time.UTC)
		return start, start.AddDate(0, 1, 0).Add(-time.Millisecond)
	}
	switch period {
	case "Q1":
		return time.Date(year, time.January, 1, 0, 0, 0, 0, time.UTC), time.Date(year, time.April, 1, 0, 0, 0, 0, time.UTC).Add(-time.Millisecond)
	case "Q2":
		return time.Date(year, time.April, 1, 0, 0, 0, 0, time.UTC), time.Date(year, time.July, 1, 0, 0, 0, 0, time.UTC).Add(-time.Millisecond)
	case "Q3":
		return time.Date(year, time.July, 1, 0, 0, 0, 0, time.UTC), time.Date(year, time.October, 1, 0, 0, 0, 0, time.UTC).Add(-time.Millisecond)
	case "Q4":
		return time.Date(year, time.October, 1, 0, 0, 0, 0, time.UTC), time.Date(year+1, time.January, 1, 0, 0, 0, 0, time.UTC).Add(-time.Millisecond)
	case "H1":
		return time.Date(year, time.January, 1, 0, 0, 0, 0, time.UTC), time.Date(year, time.July, 1, 0, 0, 0, 0, time.UTC).Add(-time.Millisecond)
	case "H2":
		return time.Date(year, time.July, 1, 0, 0, 0, 0, time.UTC), time.Date(year+1, time.January, 1, 0, 0, 0, 0, time.UTC).Add(-time.Millisecond)
	default:
		return time.Date(year, time.January, 1, 0, 0, 0, 0, time.UTC), time.Date(year+1, time.January, 1, 0, 0, 0, 0, time.UTC).Add(-time.Millisecond)
	}
}

type deepMonthMetric struct {
	Month  string
	NetBps float64
}

func deepMonthNets(events []deepCandidateEvent, candles []protocol.Candle, start, end time.Time) []deepMonthMetric {
	return deepMonthNetsWithParameters(events, candles, start, end, 60, 5, 0)
}

func deepMonthNetsWithParameters(events []deepCandidateEvent, candles []protocol.Candle, start, end time.Time, horizonMinutes int, costBPS float64, entryDelay int) []deepMonthMetric {
	byMonth := make(map[string][]deepCandidateEvent)
	for _, event := range events {
		key := time.UnixMilli(event.EventTimeMS).UTC().Format("2006-01")
		byMonth[key] = append(byMonth[key], event)
	}
	var out []deepMonthMetric
	for cur := time.Date(start.Year(), start.Month(), 1, 0, 0, 0, 0, time.UTC); !cur.After(end); cur = cur.AddDate(0, 1, 0) {
		monthEnd := cur.AddDate(0, 1, 0).Add(-time.Millisecond)
		if monthEnd.After(end) {
			monthEnd = end
		}
		set := deepReturnsForEvents(byMonth[cur.Format("2006-01")], candles, horizonMinutes, costBPS, entryDelay, monthEnd.UnixMilli())
		m := deepMetricFromBps(set.ReturnsBps)
		out = append(out, deepMonthMetric{Month: cur.Format("2006-01"), NetBps: m.NetBps})
	}
	return out
}

func summarizeDeepMonths(months []deepMonthMetric) (string, string, int, float64) {
	bestMonth, worstMonth := "", ""
	bestNet := -math.MaxFloat64
	worstNet := math.MaxFloat64
	totalNet := 0.0
	maxPositive := 0.0
	positiveMonths := 0
	for _, month := range months {
		if month.NetBps > bestNet {
			bestNet = month.NetBps
			bestMonth = month.Month
		}
		if month.NetBps < worstNet {
			worstNet = month.NetBps
			worstMonth = month.Month
		}
		totalNet += month.NetBps
		if month.NetBps > 0 {
			positiveMonths++
			if month.NetBps > maxPositive {
				maxPositive = month.NetBps
			}
		}
	}
	pct := 0.0
	if totalNet > 0 && maxPositive > 0 {
		pct = maxPositive / totalNet * 100.0
	} else if maxPositive > 0 {
		pct = 100.0
	}
	return bestMonth, worstMonth, positiveMonths, pct
}

func deepRegimeBreakdown(events []deepCandidateEvent, candles []protocol.Candle, group string) []DeepRegimeMetric {
	return deepRegimeBreakdownWithParameters(events, candles, group, 60, 5, 100)
}

func deepRegimeBreakdownConfigured(events []deepCandidateEvent, candles []protocol.Candle, group string, config researchidentity.ResolvedResearchConfiguration) []DeepRegimeMetric {
	return deepRegimeBreakdownWithParameters(events, candles, group, config.RegimeHorizonMinutes, config.RegimeCostBPS, config.RegimeLowSampleMinimum)
}

func deepRegimeBreakdownWithParameters(events []deepCandidateEvent, candles []protocol.Candle, group string, horizonMinutes int, costBPS float64, lowSampleMinimum int) []DeepRegimeMetric {
	byBucket := make(map[string][]deepCandidateEvent)
	for _, event := range events {
		bucket := deepRegimeBucket(event.Label, group)
		if bucket == "" {
			bucket = "unknown"
		}
		byBucket[bucket] = append(byBucket[bucket], event)
	}
	var keys []string
	for key := range byBucket {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	out := make([]DeepRegimeMetric, 0, len(keys))
	for _, key := range keys {
		set := deepReturnsForEvents(byBucket[key], candles, horizonMinutes, costBPS, 0, 0)
		m := deepMetricFromBps(set.ReturnsBps)
		warn := ""
		if len(set.ReturnsBps) < lowSampleMinimum {
			warn = "LOW_SAMPLE"
		}
		out = append(out, DeepRegimeMetric{
			Bucket:                 key,
			EventCount:             len(set.ReturnsBps),
			PFAfter5bps:            m.PF,
			ExpectancyBpsAfter5bps: m.Expectancy,
			WinRate:                m.WinRate,
			LowSampleWarning:       warn,
		})
	}
	return out
}

func deepRegimeBucket(label regime.Label, group string) string {
	switch group {
	case "composite":
		return label.Composite
	case "volatility":
		return label.Volatility
	case "trend":
		return label.Trend
	case "liquidity":
		return label.Liquidity
	case "market_beta":
		return label.MarketBeta
	default:
		return ""
	}
}

func buildDeepComparisonMetrics(report DeepCandidateReport) DeepComparisonMetrics {
	byPeriod := make(map[string]DeepStabilityMetric)
	for _, m := range report.Stability {
		byPeriod[m.Period] = m
	}
	worstQuarterPF := math.MaxFloat64
	for _, q := range []string{"Q1", "Q2", "Q3", "Q4"} {
		if byPeriod[q].PFAfter5bps < worstQuarterPF {
			worstQuarterPF = byPeriod[q].PFAfter5bps
		}
	}
	if worstQuarterPF == math.MaxFloat64 {
		worstQuarterPF = 0
	}
	bestBracketPF := 0.0
	bestBracketExp := -math.MaxFloat64
	for _, b := range report.Brackets {
		if b.NetExpectancyBpsAfter5bps > bestBracketExp {
			bestBracketExp = b.NetExpectancyBpsAfter5bps
			bestBracketPF = b.PFAfter5bps
		}
	}
	if bestBracketExp == -math.MaxFloat64 {
		bestBracketExp = 0
	}
	mfeRatio := 0.0
	for _, m := range report.MFEMAE {
		if m.WindowMinutes == 60 {
			mfeRatio = m.MFEMAERatio
			break
		}
	}
	return DeepComparisonMetrics{
		EventCount:                 byPeriod["FY"].EventCount,
		FYPFAfter5bps:              byPeriod["FY"].PFAfter5bps,
		H2PFAfter5bps:              byPeriod["H2"].PFAfter5bps,
		WorstQuarterPFAfter5bps:    worstQuarterPF,
		FYExpectancyBps:            byPeriod["FY"].ExpectancyBpsAfter5bps,
		H2ExpectancyBps:            byPeriod["H2"].ExpectancyBpsAfter5bps,
		EntryDelay1cExpectancyBps:  entryDelayExpectancy(report.EntryDelays, 1),
		PositiveMonthCount:         byPeriod["FY"].PositiveMonthCount,
		SingleMonthContributionPct: byPeriod["FY"].SingleMonthContributionPct,
		BestBracketPFAfter5bps:     bestBracketPF,
		BestBracketExpectancyBps:   bestBracketExp,
		MFEMAERatio:                mfeRatio,
		CostSensitivity:            deepCostSensitivity(report.CostHaircuts),
		LeakageStatus:              report.LeakageStatus,
	}
}

func entryDelayExpectancy(delays []DeepEntryDelayMetric, target int) float64 {
	for _, delay := range delays {
		if delay.DelayCandles == target {
			return delay.ExpectancyBps
		}
	}
	return 0
}

func deepCostSensitivity(costs []DeepCostHaircutMetric) string {
	for _, cost := range costs {
		if cost.CostBps == 15 && cost.PFAbove105 {
			return "PF >= 1.05 through 15 bps"
		}
	}
	for _, cost := range costs {
		if cost.CostBps == 10 && cost.PFAbove105 {
			return "PF >= 1.05 through 10 bps"
		}
	}
	for _, cost := range costs {
		if cost.CostBps == 5 && cost.PFAbove105 {
			return "PF >= 1.05 through 5 bps"
		}
	}
	return "PF < 1.05 at 5 bps"
}

func deepAcceptanceGates(report DeepCandidateReport) []DeepGateResult {
	return deepAcceptanceGatesConfigured(report, researchidentity.GateThresholds{
		MinimumEvents: 300, MinimumH2PF: 1.10, MinimumH2ExpectancyBPS: 0,
		MinimumFYPF: 1.05, MinimumFYExpectancyBPS: 0, MinimumPositiveMonths: 3,
		MinimumDelayOneExpectancyBPS: 0, MaximumSingleMonthContribution: 50,
		MinimumWorstQuarterPF: 0.95, MinimumBracketPF: 0.50, MinimumBracketExpectancyBPS: -10,
	})
}

func deepAcceptanceGatesConfigured(report DeepCandidateReport, thresholds researchidentity.GateThresholds) []DeepGateResult {
	m := report.ComparisonMetrics
	bracketOK := deepAnyBracketNotCatastrophic(report.Brackets)
	return []DeepGateResult{
		{Name: "execution_series_identity", Passed: report.ExecutionSeriesID == executionseries.ID && report.ExecutionSeriesVersion == executionseries.Version, Critical: true, Actual: report.ExecutionSeriesID + ".v" + report.ExecutionSeriesVersion, Threshold: executionseries.GenerationVersion},
		{Name: "H2 PF after 5 bps", Passed: m.H2PFAfter5bps >= thresholds.MinimumH2PF, Critical: true, Actual: fmt.Sprintf("%.4f", m.H2PFAfter5bps), Threshold: fmt.Sprintf(">= %.4f", thresholds.MinimumH2PF)},
		{Name: "H2 expectancy after 5 bps", Passed: m.H2ExpectancyBps > thresholds.MinimumH2ExpectancyBPS, Critical: true, Actual: fmt.Sprintf("%.4f bps", m.H2ExpectancyBps), Threshold: fmt.Sprintf("> %.4f bps", thresholds.MinimumH2ExpectancyBPS)},
		{Name: "FY PF after 5 bps", Passed: m.FYPFAfter5bps >= thresholds.MinimumFYPF, Actual: fmt.Sprintf("%.4f", m.FYPFAfter5bps), Threshold: fmt.Sprintf(">= %.4f", thresholds.MinimumFYPF)},
		{Name: "FY expectancy after 5 bps", Passed: m.FYExpectancyBps > thresholds.MinimumFYExpectancyBPS, Actual: fmt.Sprintf("%.4f bps", m.FYExpectancyBps), Threshold: fmt.Sprintf("> %.4f bps", thresholds.MinimumFYExpectancyBPS)},
		{Name: "Worst quarter PF after 5 bps", Passed: m.WorstQuarterPFAfter5bps >= thresholds.MinimumWorstQuarterPF, Actual: fmt.Sprintf("%.4f", m.WorstQuarterPFAfter5bps), Threshold: fmt.Sprintf(">= %.4f", thresholds.MinimumWorstQuarterPF)},
		{Name: "event_count", Passed: m.EventCount >= thresholds.MinimumEvents, Critical: true, Actual: fmt.Sprintf("%d", m.EventCount), Threshold: fmt.Sprintf(">= %d", thresholds.MinimumEvents)},
		{Name: "positive_month_count", Passed: m.PositiveMonthCount >= thresholds.MinimumPositiveMonths, Actual: fmt.Sprintf("%d", m.PositiveMonthCount), Threshold: fmt.Sprintf(">= %d", thresholds.MinimumPositiveMonths)},
		{Name: "single_month_contribution_pct", Passed: m.SingleMonthContributionPct <= thresholds.MaximumSingleMonthContribution, Actual: fmt.Sprintf("%.2f%%", m.SingleMonthContributionPct), Threshold: fmt.Sprintf("<= %.2f%%", thresholds.MaximumSingleMonthContribution)},
		{Name: "leakage_status", Passed: m.LeakageStatus == "PASS", Critical: true, Actual: m.LeakageStatus, Threshold: "PASS"},
		{Name: "bracket_model_not_catastrophic", Passed: bracketOK, Actual: fmt.Sprintf("%t", bracketOK), Threshold: fmt.Sprintf("at least one bracket PF >= %.4f and expectancy >= %.4f bps", thresholds.MinimumBracketPF, thresholds.MinimumBracketExpectancyBPS)},
	}
}

func deepAnyBracketNotCatastrophic(brackets []DeepBracketMetric) bool {
	for _, b := range brackets {
		if b.NotCatastrophicAfter5bps {
			return true
		}
	}
	return false
}

func splitDeepGates(gates []DeepGateResult) ([]string, []string) {
	var passed, failed []string
	for _, gate := range gates {
		if gate.Passed {
			passed = append(passed, gate.Name)
		} else {
			failed = append(failed, gate.Name)
		}
	}
	return passed, failed
}

func deepFinalStatus(gates []DeepGateResult) string {
	var failed []DeepGateResult
	criticalFailed := false
	for _, gate := range gates {
		if !gate.Passed {
			failed = append(failed, gate)
			if gate.Critical {
				criticalFailed = true
			}
		}
	}
	if len(failed) == 0 {
		return phase103StatusValidatedResearchLead
	}
	if criticalFailed {
		return phase103StatusRejected
	}
	if len(failed) <= 2 {
		return phase103StatusFragile
	}
	return phase103StatusRejected
}

func writeDeepCandidateReport(mdPath string, report DeepCandidateReport) error {
	mdOut, jsonOut := normalizeMDAndJSONPaths(mdPath)
	if err := writeJSONFile(jsonOut, report); err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(mdOut), 0755); err != nil {
		return err
	}
	if err := os.WriteFile(mdOut, []byte(buildDeepCandidateMD(report)), 0644); err != nil {
		return fmt.Errorf("write deep markdown: %w", err)
	}
	return nil
}

func buildDeepCandidateMD(report DeepCandidateReport) string {
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("# Phase 10.3 Deep Validation: %s %s_%s\n\n", report.Symbol, report.Family, report.Side))
	sb.WriteString("## Summary\n")
	sb.WriteString(fmt.Sprintf("- Final status: `%s`\n", report.FinalStatus))
	sb.WriteString(fmt.Sprintf("- Leakage status: `%s`\n", report.LeakageStatus))
	sb.WriteString(fmt.Sprintf("- Event count: %d\n", report.ComparisonMetrics.EventCount))
	sb.WriteString(fmt.Sprintf("- H2 PF after 5 bps: %.4f\n", report.ComparisonMetrics.H2PFAfter5bps))
	sb.WriteString(fmt.Sprintf("- FY PF after 5 bps: %.4f\n", report.ComparisonMetrics.FYPFAfter5bps))
	sb.WriteString(fmt.Sprintf("- Worst quarter PF after 5 bps: %.4f\n", report.ComparisonMetrics.WorstQuarterPFAfter5bps))
	sb.WriteString(fmt.Sprintf("- H2 expectancy after 5 bps: %.4f bps\n", report.ComparisonMetrics.H2ExpectancyBps))
	sb.WriteString(fmt.Sprintf("- FY expectancy after 5 bps: %.4f bps\n\n", report.ComparisonMetrics.FYExpectancyBps))

	sb.WriteString("## Candidate Generation Audit\n")
	sb.WriteString(fmt.Sprintf("- compressed_range rows: %d\n", report.CandidateGenerationAudit.CompressedRangeRows))
	sb.WriteString(fmt.Sprintf("- eligible %s candidates: %d\n", strings.ToLower(report.Side), report.CandidateGenerationAudit.EligibleSideCandidates))
	sb.WriteString(fmt.Sprintf("- candidates rejected by direction rule: %d\n", report.CandidateGenerationAudit.CandidatesRejectedByDirectionRule))
	sb.WriteString(fmt.Sprintf("- candidates rejected by BTC/ETH beta confirmation: %d\n", report.CandidateGenerationAudit.CandidatesRejectedByBTCETHBetaConfirmation))
	sb.WriteString(fmt.Sprintf("- accepted candidates: %d\n", report.CandidateGenerationAudit.AcceptedCandidates))
	sb.WriteString(fmt.Sprintf("- clustered candidates: %d\n", report.CandidateGenerationAudit.ClusteredCandidates))
	sb.WriteString(fmt.Sprintf("- unique event clusters: %d\n\n", report.CandidateGenerationAudit.UniqueEventClusters))

	sb.WriteString("## Forward Return Analysis\n")
	sb.WriteString("| Horizon | Events | Avg Return (bps) | Median (bps) | Win Rate | PF Proxy | Expectancy (bps) | P25 | P75 | Worst 5% | Best 5% | Truncated |\n")
	sb.WriteString("|---|---:|---:|---:|---:|---:|---:|---:|---:|---:|---:|---:|\n")
	for _, m := range report.ForwardReturns {
		sb.WriteString(fmt.Sprintf("| %dm | %d | %.4f | %.4f | %.4f | %.4f | %.4f | %.4f | %.4f | %.4f | %.4f | %d |\n",
			m.HorizonMinutes, m.EventCount, m.AverageReturnBps, m.MedianReturnBps, m.WinRate, m.PFProxy, m.ExpectancyBps, m.P25ReturnBps, m.P75ReturnBps, m.Worst5PctReturnBps, m.Best5PctReturnBps, m.TruncatedCount))
	}
	sb.WriteString("\n## Cost Haircut Analysis\n")
	sb.WriteString("| Cost (bps) | Events | PF | Expectancy (bps) | Win Rate | PF > 1.05 | PF > 1.10 | PF > 1.20 |\n")
	sb.WriteString("|---:|---:|---:|---:|---:|---:|---:|---:|\n")
	for _, m := range report.CostHaircuts {
		sb.WriteString(fmt.Sprintf("| %.0f | %d | %.4f | %.4f | %.4f | %t | %t | %t |\n", m.CostBps, m.EventCount, m.PF, m.ExpectancyBps, m.WinRate, m.PFAbove105, m.PFAbove110, m.PFAbove120))
	}
	sb.WriteString("\n## Entry Delay Analysis\n")
	sb.WriteString("| Delay (candles) | Events | Expectancy (bps) | PF after 5 bps | Win Rate | Truncated |\n")
	sb.WriteString("|---:|---:|---:|---:|---:|---:|\n")
	for _, m := range report.EntryDelays {
		sb.WriteString(fmt.Sprintf("| %d | %d | %.4f | %.4f | %.4f | %d |\n", m.DelayCandles, m.EventCount, m.ExpectancyBps, m.PFAfter5bps, m.WinRate, m.TruncatedCount))
	}
	sb.WriteString("\n## MFE / MAE Analysis\n")
	sb.WriteString("| Window | Events | Median MFE (bps) | Median MAE (bps) | Avg MFE (bps) | Avg MAE (bps) | MFE/MAE | +5 before -5 | +10 before -5 | +15 before -10 |\n")
	sb.WriteString("|---|---:|---:|---:|---:|---:|---:|---:|---:|---:|\n")
	for _, m := range report.MFEMAE {
		sb.WriteString(fmt.Sprintf("| %dm | %d | %.4f | %.4f | %.4f | %.4f | %.4f | %.2f%% | %.2f%% | %.2f%% |\n",
			m.WindowMinutes, m.EventCount, m.MedianMFEBps, m.MedianMAEBps, m.AverageMFEBps, m.AverageMAEBps, m.MFEMAERatio, m.Reach5BpsBeforeMinus5Pct, m.Reach10BpsBeforeMinus5Pct, m.Reach15BpsBeforeMinus10Pct))
	}
	sb.WriteString("\n## Conservative Bracket Simulation\n")
	sb.WriteString("| Bracket | Trades | Win Rate | PF after 5 bps | Net Exp after 5 bps | Avg Hold Min | Unresolved | Not Catastrophic |\n")
	sb.WriteString("|---|---:|---:|---:|---:|---:|---:|---:|\n")
	for _, b := range report.Brackets {
		sb.WriteString(fmt.Sprintf("| %s | %d | %.4f | %.4f | %.4f | %.2f | %d | %t |\n", b.Name, b.TradeCount, b.WinRate, b.PFAfter5bps, b.NetExpectancyBpsAfter5bps, b.AverageHoldMinutes, b.UnresolvedCount, b.NotCatastrophicAfter5bps))
	}
	sb.WriteString("\n## Monthly And Quarterly Stability\n")
	sb.WriteString("| Period | Events | PF after 5 bps | Expectancy (bps) | Win Rate | Best Month | Worst Month | Single Month % | Positive Months |\n")
	sb.WriteString("|---|---:|---:|---:|---:|---|---|---:|---:|\n")
	for _, m := range report.Stability {
		sb.WriteString(fmt.Sprintf("| %s | %d | %.4f | %.4f | %.4f | %s | %s | %.2f | %d |\n", m.Period, m.EventCount, m.PFAfter5bps, m.ExpectancyBpsAfter5bps, m.WinRate, m.BestMonth, m.WorstMonth, m.SingleMonthContributionPct, m.PositiveMonthCount))
	}
	sb.WriteString("\n## Regime Breakdown\n")
	for _, group := range []string{"composite", "volatility", "trend", "liquidity", "market_beta"} {
		sb.WriteString(fmt.Sprintf("### %s\n", group))
		sb.WriteString("| Bucket | Events | PF after 5 bps | Expectancy (bps) | Win Rate | Low Sample |\n")
		sb.WriteString("|---|---:|---:|---:|---:|---|\n")
		for _, m := range report.RegimeBreakdown[group] {
			sb.WriteString(fmt.Sprintf("| %s | %d | %.4f | %.4f | %.4f | %s |\n", m.Bucket, m.EventCount, m.PFAfter5bps, m.ExpectancyBpsAfter5bps, m.WinRate, m.LowSampleWarning))
		}
		sb.WriteString("\n")
	}
	sb.WriteString("## Gates\n")
	sb.WriteString("| Gate | Passed | Critical | Actual | Threshold |\n")
	sb.WriteString("|---|---:|---:|---:|---|\n")
	for _, gate := range report.Gates {
		sb.WriteString(fmt.Sprintf("| %s | %t | %t | %s | %s |\n", gate.Name, gate.Passed, gate.Critical, gate.Actual, gate.Threshold))
	}
	sb.WriteString("\n")
	if len(report.FailedGates) > 0 {
		sb.WriteString("## Failed Gates\n")
		for _, gate := range report.FailedGates {
			sb.WriteString(fmt.Sprintf("- %s\n", gate))
		}
		sb.WriteString("\n")
	}
	return sb.String()
}

func buildPhase103Comparison(reports []DeepCandidateReport) Phase103ComparisonReport {
	out := Phase103ComparisonReport{
		Summary: Phase103ComparisonSummary{CandidateCount: len(reports)},
	}
	bestScore := -math.MaxFloat64
	worstScore := math.MaxFloat64
	validated := 0
	for _, report := range reports {
		m := report.ComparisonMetrics
		row := Phase103ComparisonRow{
			Symbol:                     report.Symbol,
			Family:                     report.Family,
			Side:                       report.Side,
			EventCount:                 m.EventCount,
			FYPFAfter5bps:              m.FYPFAfter5bps,
			H2PFAfter5bps:              m.H2PFAfter5bps,
			WorstQuarterPFAfter5bps:    m.WorstQuarterPFAfter5bps,
			FYExpectancyBps:            m.FYExpectancyBps,
			H2ExpectancyBps:            m.H2ExpectancyBps,
			EntryDelay1cExpectancyBps:  m.EntryDelay1cExpectancyBps,
			PositiveMonthCount:         m.PositiveMonthCount,
			SingleMonthContributionPct: m.SingleMonthContributionPct,
			BestBracketPFAfter5bps:     m.BestBracketPFAfter5bps,
			BestBracketExpectancyBps:   m.BestBracketExpectancyBps,
			MFEMAERatio:                m.MFEMAERatio,
			CostSensitivity:            m.CostSensitivity,
			LeakageStatus:              m.LeakageStatus,
			FinalStatus:                report.FinalStatus,
			FailedGates:                report.FailedGates,
		}
		out.Rows = append(out.Rows, row)
		score := m.H2ExpectancyBps + m.FYExpectancyBps + (m.H2PFAfter5bps-1)*10 + (m.FYPFAfter5bps-1)*10
		name := fmt.Sprintf("%s %s_%s", report.Symbol, report.Family, report.Side)
		if score > bestScore {
			bestScore = score
			out.Summary.StrongestCandidate = name
		}
		if score < worstScore {
			worstScore = score
			out.Summary.WeakestCandidate = name
		}
		if report.FinalStatus == phase103StatusValidatedResearchLead {
			validated++
		}
	}
	sort.Slice(out.Rows, func(i, j int) bool {
		if out.Rows[i].FinalStatus != out.Rows[j].FinalStatus {
			return out.Rows[i].FinalStatus < out.Rows[j].FinalStatus
		}
		if out.Rows[i].Symbol != out.Rows[j].Symbol {
			return out.Rows[i].Symbol < out.Rows[j].Symbol
		}
		return out.Rows[i].Family < out.Rows[j].Family
	})
	if validated > 0 {
		out.Summary.FinalRecommendation = "one or more validated research leads worth Phase 10.4 strategy-shape research"
	} else if len(reports) == 0 {
		out.Summary.FinalRecommendation = "needs more data"
	} else {
		out.Summary.FinalRecommendation = "no validated research leads"
	}
	return out
}

func writePhase103Comparison(outDir string, report Phase103ComparisonReport) error {
	jsonPath := filepath.Join(outDir, "phase10_3_research_lead_comparison.json")
	mdPath := filepath.Join(outDir, "phase10_3_research_lead_comparison.md")
	if err := writeJSONFile(jsonPath, report); err != nil {
		return err
	}
	if err := os.WriteFile(mdPath, []byte(buildPhase103ComparisonMD(report)), 0644); err != nil {
		return fmt.Errorf("write comparison markdown: %w", err)
	}
	return nil
}

func buildPhase103ComparisonMD(report Phase103ComparisonReport) string {
	var sb strings.Builder
	sb.WriteString("# Phase 10.3 Research Lead Comparison\n\n")
	sb.WriteString("## Summary\n")
	sb.WriteString(fmt.Sprintf("- Candidate count: %d\n", report.Summary.CandidateCount))
	sb.WriteString(fmt.Sprintf("- Strongest candidate: `%s`\n", report.Summary.StrongestCandidate))
	sb.WriteString(fmt.Sprintf("- Weakest candidate: `%s`\n", report.Summary.WeakestCandidate))
	sb.WriteString(fmt.Sprintf("- Final recommendation: %s\n\n", report.Summary.FinalRecommendation))
	sb.WriteString("| Symbol | Family | Side | Status | Events | FY PF | H2 PF | Worst Q PF | FY Exp | H2 Exp | Delay 1c Exp | Pos Months | Single Month % | Best Bracket PF | Best Bracket Exp | MFE/MAE | Cost Sensitivity | Leakage |\n")
	sb.WriteString("|---|---|---|---|---:|---:|---:|---:|---:|---:|---:|---:|---:|---:|---:|---:|---|---|\n")
	for _, row := range report.Rows {
		sb.WriteString(fmt.Sprintf("| %s | %s | %s | %s | %d | %.4f | %.4f | %.4f | %.4f | %.4f | %.4f | %d | %.2f | %.4f | %.4f | %.4f | %s | %s |\n",
			row.Symbol, row.Family, row.Side, row.FinalStatus, row.EventCount, row.FYPFAfter5bps, row.H2PFAfter5bps, row.WorstQuarterPFAfter5bps, row.FYExpectancyBps, row.H2ExpectancyBps, row.EntryDelay1cExpectancyBps, row.PositiveMonthCount, row.SingleMonthContributionPct, row.BestBracketPFAfter5bps, row.BestBracketExpectancyBps, row.MFEMAERatio, row.CostSensitivity, row.LeakageStatus))
	}
	sb.WriteString("\n")
	return sb.String()
}
