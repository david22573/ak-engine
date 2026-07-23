package app

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/spf13/cobra"
)

type compactSummaryAnalyzerConfig struct {
	Symbols    []string
	Months     []string
	ChunksDir  string
	ReportsDir string
	Family     string
	Side       string
	Horizon    string
}

type retainedCoverageInputs struct {
	ExpectedSymbols []string
	ExpectedMonths  []string
}

type compactSummaryChunk struct {
	Symbol          string
	Month           string
	AlphaPath       string
	SummaryPath     string
	AlphaRows       []FundingAlphaSummaryRow
	AlphaMissing    bool
	SummaryMissing  bool
	SummaryStatus   string
	ZeroEventMonth  bool
	UnexplainedZero bool
}

type compactThresholds struct {
	MinEventCount                   int     `json:"min_event_count"`
	MinClusterCount                 int     `json:"min_cluster_count"`
	CalibrationReadyEventCount      int     `json:"calibration_ready_event_count"`
	TopSymbolContributionFailPct    float64 `json:"top_symbol_contribution_fail_pct"`
	TopMonthContributionFailPct     float64 `json:"top_month_contribution_fail_pct"`
	TopQuarterContributionWarnPct   float64 `json:"top_quarter_contribution_warn_pct"`
	TopBucketContributionWarnPct    float64 `json:"top_bucket_contribution_warn_pct"`
	MinBaselinePF                   float64 `json:"min_baseline_pf"`
	MinResearchLeadPF               float64 `json:"min_research_lead_pf"`
	MaxDelayDecayFailPct            float64 `json:"max_delay_decay_fail_pct"`
	MaxShadowDelayDecayPct          float64 `json:"max_shadow_delay_decay_pct"`
	MinCost10PFForShadow            float64 `json:"min_cost_10_pf_for_shadow"`
	MinCost10ExpectancyForShadowBps float64 `json:"min_cost_10_expectancy_for_shadow_bps"`
}

type compactIntegrityReport struct {
	Status               string   `json:"status"`
	MonthsExpected       int      `json:"months_expected"`
	MonthsFound          int      `json:"months_found"`
	SymbolsExpected      int      `json:"symbols_expected"`
	SymbolsFound         int      `json:"symbols_found"`
	MissingSummaries     []string `json:"missing_summaries"`
	ZeroEventSummaries   []string `json:"zero_event_summaries"`
	UnexplainedZeroEvent []string `json:"unexplained_zero_event_summaries"`
	MissingRequired      []string `json:"missing_required_metrics"`
	RawRequired          bool     `json:"raw_required"`
}

type compactDimensionRow struct {
	Key                  string  `json:"key"`
	EventCount           int     `json:"event_count"`
	ClusterCount         int     `json:"cluster_count"`
	GrossProfitBps       float64 `json:"gross_profit_bps"`
	GrossLossBps         float64 `json:"gross_loss_bps"`
	NetBps               float64 `json:"net_bps"`
	ExpectancyBps        float64 `json:"expectancy_bps"`
	ProfitFactor         float64 `json:"profit_factor"`
	PositiveContribution float64 `json:"positive_contribution_pct"`
}

type compactLeaveOneOutRow struct {
	RemovedKey    string  `json:"removed_key"`
	EventCount    int     `json:"event_count"`
	ClusterCount  int     `json:"cluster_count"`
	NetBps        float64 `json:"net_bps"`
	ExpectancyBps float64 `json:"expectancy_bps"`
	ProfitFactor  float64 `json:"profit_factor"`
	Pass          bool    `json:"pass"`
}

type compactLeaveOneOutReport struct {
	Supported      bool                    `json:"supported"`
	WorstRemaining *compactLeaveOneOutRow  `json:"worst_remaining,omitempty"`
	Rows           []compactLeaveOneOutRow `json:"rows"`
	Pass           bool                    `json:"pass"`
}

type compactConcentrationReport struct {
	TopSymbolContributionPct  float64               `json:"top_symbol_contribution_pct"`
	TopMonthContributionPct   float64               `json:"top_month_contribution_pct"`
	TopQuarterContributionPct float64               `json:"top_quarter_contribution_pct"`
	TopBucketContributionPct  float64               `json:"top_bucket_contribution_pct"`
	TopBucketBasis            string                `json:"top_bucket_basis"`
	BySymbol                  []compactDimensionRow `json:"by_symbol"`
	ByMonth                   []compactDimensionRow `json:"by_month"`
	ByQuarter                 []compactDimensionRow `json:"by_quarter"`
	ByBucket                  []compactDimensionRow `json:"by_bucket"`
}

type compactDelayReport struct {
	Baseline       *FundingDelayStressMetric `json:"baseline,omitempty"`
	Delay1         *FundingDelayStressMetric `json:"delay_1,omitempty"`
	Delay2         *FundingDelayStressMetric `json:"delay_2,omitempty"`
	Delay1DecayPct *float64                  `json:"delay_1_decay_pct,omitempty"`
	MissingMetrics []string                  `json:"missing_metrics,omitempty"`
}

type compactCandidateReport struct {
	CandidateKey          string                     `json:"candidate_key"`
	Family                string                     `json:"family"`
	Side                  string                     `json:"side"`
	Horizon               string                     `json:"horizon"`
	LeadSymbol            string                     `json:"lead_symbol"`
	EventCount            int                        `json:"event_count"`
	ClusterCount          int                        `json:"cluster_count"`
	MonthsRepresented     int                        `json:"months_represented"`
	SymbolsRepresented    int                        `json:"symbols_represented"`
	QuartersRepresented   int                        `json:"quarters_represented"`
	SampleLabel           string                     `json:"sample_label"`
	BaselineCostBps       *float64                   `json:"baseline_cost_bps,omitempty"`
	Baseline              FundingMetrics             `json:"baseline"`
	CostStress            []FundingCostStressMetric  `json:"cost_stress"`
	DelaySensitivity      compactDelayReport         `json:"delay_sensitivity"`
	Concentration         compactConcentrationReport `json:"concentration"`
	LeaveOneSymbolOut     compactLeaveOneOutReport   `json:"leave_one_symbol_out"`
	LeaveOneMonthOut      compactLeaveOneOutReport   `json:"leave_one_month_out"`
	LeaveOneQuarterOut    compactLeaveOneOutReport   `json:"leave_one_quarter_out"`
	MissingRequiredMetric []string                   `json:"missing_required_metrics"`
	Warnings              []string                   `json:"warnings"`
	Failures              []string                   `json:"failures"`
	FinalLabel            string                     `json:"final_label"`
	PromotionAllowed      bool                       `json:"promotion_allowed"`
	ShadowCandidate       bool                       `json:"shadow_candidate"`
	AkTraderReady         bool                       `json:"ak_trader_ready"`
	RecommendedNextAction string                     `json:"recommended_next_action"`
}

type compactRobustnessReport struct {
	Phase               string                   `json:"phase"`
	CompactSummaryOnly  bool                     `json:"compact_summary_only"`
	RawRequired         bool                     `json:"raw_required"`
	TargetCandidateKey  string                   `json:"target_candidate_key"`
	FinalLabel          string                   `json:"final_label"`
	PromotionAllowed    bool                     `json:"promotion_allowed"`
	ShadowCandidate     bool                     `json:"shadow_candidate"`
	Integrity           compactIntegrityReport   `json:"integrity"`
	Thresholds          compactThresholds        `json:"thresholds"`
	Candidates          []compactCandidateReport `json:"candidates"`
	TargetCandidate     compactCandidateReport   `json:"target_candidate"`
	Warnings            []string                 `json:"warnings"`
	Failures            []string                 `json:"failures"`
	AkTraderIntegration string                   `json:"ak_trader_integration"`
}

type compactPromotionGateReport struct {
	Phase               string                 `json:"phase"`
	FinalLabel          string                 `json:"final_label"`
	PromotionAllowed    bool                   `json:"promotion_allowed"`
	ShadowCandidate     bool                   `json:"shadow_candidate"`
	RawRequired         bool                   `json:"raw_required"`
	IntegrityStatus     string                 `json:"integrity_status"`
	Metrics             compactCandidateReport `json:"metrics"`
	Failures            []string               `json:"failures"`
	Warnings            []string               `json:"warnings"`
	Thresholds          compactThresholds      `json:"thresholds"`
	AkTraderIntegration string                 `json:"ak_trader_integration"`
}

type retainedSummaryScan struct {
	SummaryFiles         []retainedSummaryFile     `json:"summary_files"`
	SummaryFileCount     int                       `json:"summary_file_count"`
	FoundSymbols         []string                  `json:"found_symbols"`
	MonthsBySymbol       map[string][]string       `json:"months_by_symbol"`
	FamiliesFound        []string                  `json:"families_found"`
	SidesFound           []string                  `json:"sides_found"`
	HorizonsFound        []string                  `json:"horizons_found"`
	ZeroEventSummaries   []string                  `json:"zero_event_summaries"`
	MalformedSummaries   []retainedMalformedRecord `json:"malformed_summaries"`
	ValidRowsByCandidate map[string][]FundingAlphaSummaryRow
}

type retainedSummaryFile struct {
	Symbol        string `json:"symbol"`
	Month         string `json:"month"`
	Path          string `json:"path"`
	ZeroEvent     bool   `json:"zero_event"`
	RowCount      int    `json:"row_count"`
	SummaryStatus string `json:"summary_status,omitempty"`
}

type retainedMalformedRecord struct {
	Path   string `json:"path"`
	Symbol string `json:"symbol"`
	Month  string `json:"month"`
	Error  string `json:"error"`
}

type retainedCoverageReport struct {
	FinalLabel              string                    `json:"final_label"`
	CoverageStatus          string                    `json:"coverage_status"`
	ExpectedSymbols         []string                  `json:"expected_symbols"`
	ExpectedMonths          []string                  `json:"expected_months,omitempty"`
	FoundSymbols            []string                  `json:"found_symbols"`
	MissingSymbols          []string                  `json:"missing_symbols"`
	MonthsBySymbol          map[string][]string       `json:"months_by_symbol"`
	MissingMonthsBySymbol   map[string][]string       `json:"missing_expected_months_by_symbol,omitempty"`
	SummaryFilesFound       []string                  `json:"summary_files_found"`
	FamiliesFound           []string                  `json:"families_found"`
	SidesFound              []string                  `json:"sides_found"`
	HorizonsFound           []string                  `json:"horizons_found"`
	ZeroEventSummaries      []string                  `json:"zero_event_summaries"`
	MalformedSummaries      []retainedMalformedRecord `json:"malformed_summaries"`
	SummaryFileCount        int                       `json:"summary_file_count"`
	MalformedFileCount      int                       `json:"malformed_file_count"`
	ZeroEventCount          int                       `json:"zero_event_count"`
	FullUniverseReady       bool                      `json:"full_universe_ready"`
	ReasonNotFullUniverse   string                    `json:"reason_not_full_universe,omitempty"`
	ExpectedMonthRangeKnown bool                      `json:"expected_month_range_known"`
	RawRequired             bool                      `json:"raw_required"`
}

type rankedInventoryReport struct {
	Title                    string                   `json:"title"`
	Phase                    string                   `json:"phase"`
	SymbolUniverse           string                   `json:"symbol_universe"`
	CandidateScope           string                   `json:"candidate_scope"`
	FinalLabel               string                   `json:"final_label"`
	CoverageStatus           string                   `json:"coverage_status"`
	FullUniverseReady        bool                     `json:"full_universe_ready"`
	UniverseQualifier        string                   `json:"universe_qualifier"`
	RawRequired              bool                     `json:"raw_required"`
	AkTraderTouched          bool                     `json:"ak_trader_touched"`
	FoundSymbols             []string                 `json:"found_symbols"`
	MissingSymbols           []string                 `json:"missing_symbols"`
	MonthsBySymbol           map[string][]string      `json:"months_by_symbol"`
	CandidateCount           int                      `json:"candidate_count"`
	InventoryLabelCounts     map[string]int           `json:"inventory_label_counts"`
	ShadowCandidateCount     int                      `json:"shadow_candidate_count"`
	ResearchLeadCount        int                      `json:"research_lead_count"`
	FragileResearchLeadCount int                      `json:"fragile_research_lead_count"`
	RejectedCount            int                      `json:"rejected_count"`
	RankedInventory          []compactCandidateReport `json:"ranked_inventory"`
}

var (
	acompChunks       string
	acompReports      string
	acompSymbols      string
	acompFrom         string
	acompTo           string
	acompFamily       string
	acompSide         string
	acompHorizon      string
	acompCoverageOnly bool
	ainvFamily        string
	ainvSide          string
	ainvHorizon       string
)

var analyzeCompactRobustnessCmd = &cobra.Command{
	Use:   "analyze-compact-robustness",
	Short: "Analyze retained funding compact summaries without raw JSONL files",
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg := compactSummaryAnalyzerConfig{
			Symbols:    parseFundingSymbols(acompSymbols),
			Months:     parseFundingMonths(acompFrom, acompTo),
			ChunksDir:  acompChunks,
			ReportsDir: acompReports,
			Family:     strings.TrimSpace(acompFamily),
			Side:       strings.ToLower(strings.TrimSpace(acompSide)),
			Horizon:    strings.TrimSpace(acompHorizon),
		}
		finalLabel := "COVERAGE_ONLY"
		if !acompCoverageOnly {
			report, gate, err := runCompactRobustnessAnalysis(cfg)
			if err != nil {
				return err
			}
			if err := writeCompactRobustnessOutputs(cfg.ReportsDir, report, gate); err != nil {
				return err
			}
			finalLabel = report.FinalLabel
		}
		inventoryCfg, err := writeCompactCoverageAndInventoryOutputs(cfg, finalLabel)
		if err != nil {
			return err
		}
		if !acompCoverageOnly {
			fmt.Printf("Phase 10.8 compact robustness written to %s\n", filepath.Join(cfg.ReportsDir, "phase10_8_compact_robustness.md"))
			fmt.Printf("Phase 10.8 promotion gate written to %s\n", filepath.Join(cfg.ReportsDir, "phase10_8_promotion_gate.md"))
		}
		fmt.Printf("Phase 10.8C retained coverage written to %s\n", filepath.Join(cfg.ReportsDir, "phase10_8c_retained_coverage.md"))
		_, inventoryMDPath := rankedInventoryOutputPaths(cfg.ReportsDir, inventoryCfg)
		fmt.Printf("Phase 10.8 ranked inventory written to %s\n", inventoryMDPath)
		return nil
	},
}

func init() {
	analyzeCompactRobustnessCmd.Flags().StringVar(&acompChunks, "chunks-dir", filepath.Join("runs", "reports", "chunks"), "retained chunk summary directory")
	analyzeCompactRobustnessCmd.Flags().StringVar(&acompReports, "reports-dir", filepath.Join("runs", "reports"), "reports output directory")
	analyzeCompactRobustnessCmd.Flags().StringVar(&acompSymbols, "symbols", "", "comma-separated symbols")
	analyzeCompactRobustnessCmd.Flags().StringVar(&acompFrom, "from", "", "from month YYYY-MM")
	analyzeCompactRobustnessCmd.Flags().StringVar(&acompTo, "to", "", "to month YYYY-MM")
	analyzeCompactRobustnessCmd.Flags().StringVar(&acompFamily, "family", "NegativeFundingLong", "candidate family")
	analyzeCompactRobustnessCmd.Flags().StringVar(&acompSide, "side", "long", "candidate side")
	analyzeCompactRobustnessCmd.Flags().StringVar(&acompHorizon, "horizon", "", "candidate horizon")
	analyzeCompactRobustnessCmd.Flags().BoolVar(&acompCoverageOnly, "coverage-only", false, "write retained coverage and ranked inventory without requiring a target candidate")
	analyzeCompactRobustnessCmd.Flags().StringVar(&ainvFamily, "inventory-family", "", "ranked inventory family filter (empty scans all retained candidates)")
	analyzeCompactRobustnessCmd.Flags().StringVar(&ainvSide, "inventory-side", "", "ranked inventory side filter (empty scans all retained candidates)")
	analyzeCompactRobustnessCmd.Flags().StringVar(&ainvHorizon, "inventory-horizon", "", "ranked inventory horizon filter (empty scans all retained candidates)")
	rootCmd.AddCommand(analyzeCompactRobustnessCmd)
}

func writeCompactCoverageAndInventoryOutputs(cfg compactSummaryAnalyzerConfig, finalLabel string) (compactSummaryAnalyzerConfig, error) {
	scan, err := scanRetainedAlphaSummaries(cfg.ChunksDir)
	if err != nil {
		return compactSummaryAnalyzerConfig{}, err
	}
	coverage := buildRetainedCoverageReport(retainedCoverageInputs{
		ExpectedSymbols: expectedCompactCoverageSymbols(acompSymbols),
		ExpectedMonths:  expectedCompactCoverageMonths(acompFrom, acompTo),
	}, scan, finalLabel)
	inventoryCfg := compactSummaryAnalyzerConfig{
		Family:  strings.TrimSpace(ainvFamily),
		Side:    strings.ToLower(strings.TrimSpace(ainvSide)),
		Horizon: strings.TrimSpace(ainvHorizon),
	}
	inventoryCandidates := buildRankedInventoryCandidates(scan, inventoryCfg, defaultCompactThresholds())
	inventory := buildRankedInventoryReport(inventoryCandidates, coverage, finalLabel, inventoryCfg)
	if err := writeCompactCoverageOutputs(cfg.ReportsDir, coverage, inventory, inventoryCfg); err != nil {
		return compactSummaryAnalyzerConfig{}, err
	}
	return inventoryCfg, nil
}

func runCompactRobustnessAnalysis(cfg compactSummaryAnalyzerConfig) (compactRobustnessReport, compactPromotionGateReport, error) {
	cfg = normalizeCompactSummaryAnalyzerConfig(cfg)
	chunks, err := loadCompactSummaryChunks(cfg)
	if err != nil {
		return compactRobustnessReport{}, compactPromotionGateReport{}, err
	}
	thresholds := defaultCompactThresholds()
	integrity := buildCompactIntegrityReport(cfg, chunks)
	candidates, err := buildCompactCandidateReports(cfg, chunks, integrity, thresholds)
	if err != nil {
		return compactRobustnessReport{}, compactPromotionGateReport{}, err
	}
	if len(candidates) == 0 {
		return compactRobustnessReport{}, compactPromotionGateReport{}, fmt.Errorf("no compact summary candidates found for family=%s side=%s horizon=%s", cfg.Family, cfg.Side, cfg.Horizon)
	}
	target := candidates[0]
	report := compactRobustnessReport{
		Phase:               "10.8",
		CompactSummaryOnly:  true,
		RawRequired:         false,
		TargetCandidateKey:  target.CandidateKey,
		FinalLabel:          target.FinalLabel,
		PromotionAllowed:    target.PromotionAllowed,
		ShadowCandidate:     target.ShadowCandidate,
		Integrity:           integrity,
		Thresholds:          thresholds,
		Candidates:          candidates,
		TargetCandidate:     target,
		Warnings:            append([]string(nil), target.Warnings...),
		Failures:            append([]string(nil), target.Failures...),
		AkTraderIntegration: "Phase 10.8 does not perform ak-trader integration or promotion.",
	}
	if integrity.Status == "FAIL" {
		report.Failures = uniqueStrings(append(report.Failures, "summary_only_integrity_fail"))
	}
	gate := compactPromotionGateReport{
		Phase:               "10.8",
		FinalLabel:          target.FinalLabel,
		PromotionAllowed:    target.PromotionAllowed,
		ShadowCandidate:     target.ShadowCandidate,
		RawRequired:         false,
		IntegrityStatus:     integrity.Status,
		Metrics:             target,
		Failures:            append([]string(nil), target.Failures...),
		Warnings:            append([]string(nil), target.Warnings...),
		Thresholds:          thresholds,
		AkTraderIntegration: "Not ready for ak-trader integration in this phase.",
	}
	return report, gate, nil
}

func normalizeCompactSummaryAnalyzerConfig(cfg compactSummaryAnalyzerConfig) compactSummaryAnalyzerConfig {
	if cfg.ChunksDir == "" {
		cfg.ChunksDir = filepath.Join("runs", "reports", "chunks")
	}
	if cfg.ReportsDir == "" {
		cfg.ReportsDir = filepath.Join("runs", "reports")
	}
	discoveredSymbols, discoveredMonths := discoverCompactSummaryCoverage(cfg.ChunksDir)
	if len(cfg.Symbols) == 0 {
		if len(discoveredSymbols) > 0 {
			cfg.Symbols = discoveredSymbols
		} else {
			cfg.Symbols = append([]string(nil), defaultFundingSymbols...)
		}
	}
	if len(cfg.Months) == 0 {
		if len(discoveredMonths) > 0 {
			cfg.Months = discoveredMonths
		} else {
			cfg.Months = defaultPhase10FundingMonths()
		}
	}
	if cfg.Family == "" {
		cfg.Family = "NegativeFundingLong"
	}
	if cfg.Side == "" {
		cfg.Side = "long"
	}
	return cfg
}

func expectedCompactCoverageSymbols(value string) []string {
	if parsed := parseFundingSymbols(value); len(parsed) > 0 {
		return parsed
	}
	return append([]string(nil), defaultFundingSymbols...)
}

func expectedCompactCoverageMonths(from, to string) []string {
	if parsed := parseFundingMonths(from, to); len(parsed) > 0 {
		return parsed
	}
	return defaultPhase10FundingMonths()
}

func discoverCompactSummaryCoverage(chunksDir string) ([]string, []string) {
	entries, err := os.ReadDir(chunksDir)
	if err != nil {
		return nil, nil
	}
	symbolSet := make(map[string]struct{})
	monthSet := make(map[string]struct{})
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		symbol := strings.ToUpper(strings.TrimSpace(entry.Name()))
		files, err := os.ReadDir(filepath.Join(chunksDir, entry.Name()))
		if err != nil {
			continue
		}
		found := false
		for _, file := range files {
			if file.IsDir() {
				continue
			}
			name := file.Name()
			if !strings.HasSuffix(name, "-alpha-summary.json") {
				continue
			}
			found = true
			if len(name) >= len("2006-01-alpha-summary.json") {
				monthSet[strings.TrimSuffix(name, "-alpha-summary.json")] = struct{}{}
			}
		}
		if found {
			symbolSet[symbol] = struct{}{}
		}
	}
	symbols := make([]string, 0, len(symbolSet))
	for symbol := range symbolSet {
		symbols = append(symbols, symbol)
	}
	months := make([]string, 0, len(monthSet))
	for month := range monthSet {
		months = append(months, month)
	}
	sort.Strings(symbols)
	sort.Strings(months)
	return symbols, months
}

func scanRetainedAlphaSummaries(chunksDir string) (retainedSummaryScan, error) {
	entries, err := os.ReadDir(chunksDir)
	if err != nil {
		if os.IsNotExist(err) {
			return retainedSummaryScan{MonthsBySymbol: map[string][]string{}, ValidRowsByCandidate: map[string][]FundingAlphaSummaryRow{}}, nil
		}
		return retainedSummaryScan{}, err
	}
	scan := retainedSummaryScan{
		MonthsBySymbol:       make(map[string][]string),
		ValidRowsByCandidate: make(map[string][]FundingAlphaSummaryRow),
	}
	symbolSet := make(map[string]struct{})
	familySet := make(map[string]struct{})
	sideSet := make(map[string]struct{})
	horizonSet := make(map[string]struct{})
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		symbol := strings.ToUpper(strings.TrimSpace(entry.Name()))
		files, err := os.ReadDir(filepath.Join(chunksDir, entry.Name()))
		if err != nil {
			return retainedSummaryScan{}, err
		}
		for _, file := range files {
			if file.IsDir() || !strings.HasSuffix(file.Name(), "-alpha-summary.json") {
				continue
			}
			month := strings.TrimSuffix(file.Name(), "-alpha-summary.json")
			path := filepath.Join(chunksDir, entry.Name(), file.Name())
			record := retainedSummaryFile{
				Symbol: symbol,
				Month:  month,
				Path:   path,
			}
			scan.SummaryFiles = append(scan.SummaryFiles, record)
			scan.SummaryFileCount++
			symbolSet[symbol] = struct{}{}
			scan.MonthsBySymbol[symbol] = append(scan.MonthsBySymbol[symbol], month)

			data, err := os.ReadFile(path)
			if err != nil {
				scan.MalformedSummaries = append(scan.MalformedSummaries, retainedMalformedRecord{Path: path, Symbol: symbol, Month: month, Error: err.Error()})
				continue
			}
			var rows []FundingAlphaSummaryRow
			if err := json.Unmarshal(data, &rows); err != nil {
				scan.MalformedSummaries = append(scan.MalformedSummaries, retainedMalformedRecord{Path: path, Symbol: symbol, Month: month, Error: err.Error()})
				continue
			}
			record.RowCount = len(rows)
			record.ZeroEvent = len(rows) == 0
			if summary, ok := readFundingChunkSummary(filepath.Join(chunksDir, entry.Name(), month+"-funding-summary.json")); ok {
				record.SummaryStatus = summary.Status
				if summary.ZeroEventMonth || summary.EventCount == 0 {
					record.ZeroEvent = true
				}
			}
			if record.ZeroEvent {
				scan.ZeroEventSummaries = append(scan.ZeroEventSummaries, symbol+"|"+month)
			}
			scan.SummaryFiles[len(scan.SummaryFiles)-1] = record
			for _, row := range rows {
				key := strings.Join([]string{row.Family, strings.ToLower(row.Side), row.Horizon}, "|")
				scan.ValidRowsByCandidate[key] = append(scan.ValidRowsByCandidate[key], row)
				familySet[row.Family] = struct{}{}
				sideSet[strings.ToLower(row.Side)] = struct{}{}
				horizonSet[row.Horizon] = struct{}{}
			}
		}
	}
	scan.FoundSymbols = sortedKeys(symbolSet)
	scan.FamiliesFound = sortedKeys(familySet)
	scan.SidesFound = sortedKeys(sideSet)
	scan.HorizonsFound = sortedKeys(horizonSet)
	for symbol, months := range scan.MonthsBySymbol {
		scan.MonthsBySymbol[symbol] = uniqueStrings(months)
	}
	sort.Slice(scan.SummaryFiles, func(i, j int) bool { return scan.SummaryFiles[i].Path < scan.SummaryFiles[j].Path })
	sort.Slice(scan.MalformedSummaries, func(i, j int) bool { return scan.MalformedSummaries[i].Path < scan.MalformedSummaries[j].Path })
	scan.ZeroEventSummaries = uniqueStrings(scan.ZeroEventSummaries)
	return scan, nil
}

func readFundingChunkSummary(path string) (FundingChunkSummary, bool) {
	data, err := os.ReadFile(path)
	if err != nil {
		return FundingChunkSummary{}, false
	}
	var summary FundingChunkSummary
	if err := json.Unmarshal(data, &summary); err != nil {
		return FundingChunkSummary{}, false
	}
	return summary, true
}

func defaultCompactThresholds() compactThresholds {
	return compactThresholds{
		MinEventCount:                   30,
		MinClusterCount:                 10,
		CalibrationReadyEventCount:      100,
		TopSymbolContributionFailPct:    50,
		TopMonthContributionFailPct:     40,
		TopQuarterContributionWarnPct:   60,
		TopBucketContributionWarnPct:    60,
		MinBaselinePF:                   1.0,
		MinResearchLeadPF:               1.05,
		MaxDelayDecayFailPct:            75,
		MaxShadowDelayDecayPct:          60,
		MinCost10PFForShadow:            1.0,
		MinCost10ExpectancyForShadowBps: -0.5,
	}
}

func loadCompactSummaryChunks(cfg compactSummaryAnalyzerConfig) ([]compactSummaryChunk, error) {
	chunks := make([]compactSummaryChunk, 0, len(cfg.Symbols)*len(cfg.Months))
	for _, symbol := range cfg.Symbols {
		for _, month := range cfg.Months {
			chunk := compactSummaryChunk{
				Symbol:      symbol,
				Month:       month,
				AlphaPath:   filepath.Join(cfg.ChunksDir, symbol, month+"-alpha-summary.json"),
				SummaryPath: filepath.Join(cfg.ChunksDir, symbol, month+"-funding-summary.json"),
			}
			if data, err := os.ReadFile(chunk.AlphaPath); err != nil {
				chunk.AlphaMissing = true
			} else if err := json.Unmarshal(data, &chunk.AlphaRows); err != nil {
				return nil, fmt.Errorf("parse alpha summary %s: %w", chunk.AlphaPath, err)
			}
			var summary FundingChunkSummary
			if data, err := os.ReadFile(chunk.SummaryPath); err != nil {
				chunk.SummaryMissing = true
			} else if err := json.Unmarshal(data, &summary); err != nil {
				return nil, fmt.Errorf("parse funding summary %s: %w", chunk.SummaryPath, err)
			} else {
				chunk.SummaryStatus = summary.Status
				chunk.ZeroEventMonth = summary.ZeroEventMonth || summary.EventCount == 0
			}
			if len(chunk.AlphaRows) == 0 && !chunk.AlphaMissing {
				chunk.ZeroEventMonth = true
			}
			chunk.UnexplainedZero = chunk.ZeroEventMonth && (chunk.SummaryMissing || (chunk.SummaryStatus != "zero_events" && chunk.SummaryStatus != "unsupported_context" && chunk.SummaryStatus != "missing_data" && chunk.SummaryStatus != "PASS"))
			chunks = append(chunks, chunk)
		}
	}
	return chunks, nil
}

func buildCompactIntegrityReport(cfg compactSummaryAnalyzerConfig, chunks []compactSummaryChunk) compactIntegrityReport {
	monthsFound := make(map[string]struct{})
	symbolsFound := make(map[string]struct{})
	var missing []string
	var zero []string
	var unexplained []string
	for _, chunk := range chunks {
		if !chunk.AlphaMissing {
			monthsFound[chunk.Month] = struct{}{}
			symbolsFound[chunk.Symbol] = struct{}{}
		} else {
			missing = append(missing, chunk.Symbol+"|"+chunk.Month)
		}
		if chunk.ZeroEventMonth {
			zero = append(zero, chunk.Symbol+"|"+chunk.Month)
		}
		if chunk.UnexplainedZero {
			unexplained = append(unexplained, chunk.Symbol+"|"+chunk.Month)
		}
	}
	status := "PASS"
	if len(missing) > 0 || len(unexplained) > 0 {
		status = "FAIL"
	}
	return compactIntegrityReport{
		Status:               status,
		MonthsExpected:       len(cfg.Months),
		MonthsFound:          len(monthsFound),
		SymbolsExpected:      len(cfg.Symbols),
		SymbolsFound:         len(symbolsFound),
		MissingSummaries:     missing,
		ZeroEventSummaries:   zero,
		UnexplainedZeroEvent: unexplained,
		RawRequired:          false,
	}
}

func buildCompactCandidateReports(cfg compactSummaryAnalyzerConfig, chunks []compactSummaryChunk, integrity compactIntegrityReport, thresholds compactThresholds) ([]compactCandidateReport, error) {
	byCandidate := make(map[string][]FundingAlphaSummaryRow)
	for _, chunk := range chunks {
		for _, row := range chunk.AlphaRows {
			key := strings.Join([]string{row.Family, strings.ToLower(row.Side), row.Horizon}, "|")
			byCandidate[key] = append(byCandidate[key], row)
		}
	}
	keys := make([]string, 0, len(byCandidate))
	for key := range byCandidate {
		parts := strings.Split(key, "|")
		if len(parts) != 3 {
			continue
		}
		if cfg.Family != "" && parts[0] != cfg.Family {
			continue
		}
		if cfg.Side != "" && parts[1] != cfg.Side {
			continue
		}
		if cfg.Horizon != "" && parts[2] != cfg.Horizon {
			continue
		}
		keys = append(keys, key)
	}
	sort.Strings(keys)
	out := make([]compactCandidateReport, 0, len(keys))
	for _, key := range keys {
		out = append(out, evaluateCompactCandidate(key, byCandidate[key], integrity, thresholds))
	}
	sort.Slice(out, func(i, j int) bool { return compactCandidateBetter(out[i], out[j]) })
	return out, nil
}

func buildRankedInventoryCandidates(scan retainedSummaryScan, cfg compactSummaryAnalyzerConfig, thresholds compactThresholds) []compactCandidateReport {
	keys := make([]string, 0, len(scan.ValidRowsByCandidate))
	for key := range scan.ValidRowsByCandidate {
		parts := strings.Split(key, "|")
		if len(parts) != 3 {
			continue
		}
		if cfg.Family != "" && parts[0] != cfg.Family {
			continue
		}
		if cfg.Side != "" && parts[1] != cfg.Side {
			continue
		}
		if cfg.Horizon != "" && parts[2] != cfg.Horizon {
			continue
		}
		keys = append(keys, key)
	}
	sort.Strings(keys)
	out := make([]compactCandidateReport, 0, len(keys))
	integrity := compactIntegrityReport{Status: "PASS", RawRequired: false}
	for _, key := range keys {
		out = append(out, evaluateCompactCandidate(key, scan.ValidRowsByCandidate[key], integrity, thresholds))
	}
	sort.Slice(out, func(i, j int) bool { return compactCandidateBetter(out[i], out[j]) })
	return out
}

func evaluateCompactCandidate(candidateKey string, rows []FundingAlphaSummaryRow, integrity compactIntegrityReport, thresholds compactThresholds) compactCandidateReport {
	parts := strings.Split(candidateKey, "|")
	baseline := aggregateMetricsFromAlphaSummaries(rows)
	bySymbol := aggregateCompactDimensions(rows, "symbol")
	byMonth := aggregateCompactDimensions(rows, "month")
	byQuarter := aggregateCompactDimensions(rows, "quarter")
	byBucket, bucketBasis := aggregateCompactBucketDimensions(rows)
	delay := compactDelaySensitivity(baseline.DelayStress)
	missing := compactMissingRequiredMetrics(baseline, delay, byMonth, byQuarter)
	symbolLOO := compactLeaveOneOut(bySymbol)
	monthLOO := compactLeaveOneOut(byMonth)
	quarterLOO := compactLeaveOneOut(byQuarter)
	report := compactCandidateReport{
		CandidateKey:          candidateKey,
		Family:                parts[0],
		Side:                  parts[1],
		Horizon:               parts[2],
		LeadSymbol:            topContributionKey(bySymbol),
		EventCount:            baseline.EventCount,
		ClusterCount:          baseline.ClusterCount,
		MonthsRepresented:     distinctDimensionCount(rows, "month"),
		SymbolsRepresented:    distinctDimensionCount(rows, "symbol"),
		QuartersRepresented:   distinctDimensionCount(rows, "quarter"),
		SampleLabel:           compactSampleLabel(baseline.EventCount, baseline.ClusterCount, thresholds),
		Baseline:              baseline,
		CostStress:            baseline.CostStress,
		DelaySensitivity:      delay,
		Concentration:         compactConcentration(bySymbol, byMonth, byQuarter, byBucket, bucketBasis),
		LeaveOneSymbolOut:     symbolLOO,
		LeaveOneMonthOut:      monthLOO,
		LeaveOneQuarterOut:    quarterLOO,
		MissingRequiredMetric: missing,
		AkTraderReady:         false,
	}
	if baseline.BaselineCostBps > 0 {
		report.BaselineCostBps = &baseline.BaselineCostBps
	}
	label, allowed, shadow, failures, warnings, next := compactPromotionDecision(report, integrity, thresholds)
	report.FinalLabel = label
	report.PromotionAllowed = allowed
	report.ShadowCandidate = shadow
	report.Failures = failures
	report.Warnings = warnings
	report.RecommendedNextAction = next
	return report
}

func aggregateCompactDimensions(rows []FundingAlphaSummaryRow, level string) []compactDimensionRow {
	grouped := make(map[string][]FundingAlphaSummaryRow)
	for _, row := range rows {
		var key string
		switch level {
		case "symbol":
			key = row.Symbol
		case "month":
			key = row.Month
		case "quarter":
			key = row.Quarter
		}
		grouped[key] = append(grouped[key], row)
	}
	out := make([]compactDimensionRow, 0, len(grouped))
	for key, groupRows := range grouped {
		agg := aggregateMetricsFromAlphaSummaries(groupRows)
		out = append(out, compactDimensionRow{
			Key:            key,
			EventCount:     agg.EventCount,
			ClusterCount:   agg.ClusterCount,
			GrossProfitBps: agg.GrossProfitBps,
			GrossLossBps:   agg.GrossLossBps,
			NetBps:         roundMetric(agg.GrossProfitBps - agg.GrossLossBps),
			ExpectancyBps:  agg.ExpectancyCombined_5bpsBps,
			ProfitFactor:   agg.PFCombined_5bps,
		})
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Key < out[j].Key })
	addPositiveContribution(out)
	return out
}

func aggregateCompactBucketDimensions(rows []FundingAlphaSummaryRow) ([]compactDimensionRow, string) {
	agg := aggregateMetricsFromAlphaSummaries(rows)
	var filtered []FundingBucketMetric
	basis := "missing"
	for _, bucketType := range []string{"funding_x_regime", "regime", "funding_severity"} {
		for _, row := range agg.BucketMetrics {
			if row.BucketType == bucketType {
				filtered = append(filtered, row)
			}
		}
		if len(filtered) > 0 {
			basis = bucketType
			break
		}
	}
	out := make([]compactDimensionRow, 0, len(filtered))
	for _, row := range filtered {
		out = append(out, compactDimensionRow{
			Key:            row.Bucket,
			EventCount:     row.EventCount,
			ClusterCount:   row.DeClusteredEventCount,
			GrossProfitBps: row.GrossProfitBps,
			GrossLossBps:   row.GrossLossBps,
			NetBps:         row.NetBps,
			ExpectancyBps:  row.ExpectancyBps,
			ProfitFactor:   row.PF,
		})
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Key < out[j].Key })
	addPositiveContribution(out)
	return out, basis
}

func compactDelaySensitivity(rows []FundingDelayStressMetric) compactDelayReport {
	var report compactDelayReport
	for i := range rows {
		row := rows[i]
		switch row.DelayCandles {
		case 0:
			report.Baseline = &row
		case 1:
			report.Delay1 = &row
		case 2:
			report.Delay2 = &row
		}
	}
	if report.Baseline == nil {
		report.MissingMetrics = append(report.MissingMetrics, "delay_0")
	}
	if report.Delay1 == nil || !report.Delay1.Available {
		report.MissingMetrics = append(report.MissingMetrics, "delay_1")
	}
	if report.Delay2 == nil {
		report.MissingMetrics = append(report.MissingMetrics, "delay_2")
	}
	if report.Baseline != nil && report.Delay1 != nil && report.Baseline.ExpectancyBps != 0 {
		decay := roundMetric((1 - (report.Delay1.ExpectancyBps / report.Baseline.ExpectancyBps)) * 100)
		report.Delay1DecayPct = &decay
	}
	return report
}

func compactConcentration(bySymbol, byMonth, byQuarter, byBucket []compactDimensionRow, bucketBasis string) compactConcentrationReport {
	return compactConcentrationReport{
		TopSymbolContributionPct:  topPositiveContribution(bySymbol),
		TopMonthContributionPct:   topPositiveContribution(byMonth),
		TopQuarterContributionPct: topPositiveContribution(byQuarter),
		TopBucketContributionPct:  topPositiveContribution(byBucket),
		TopBucketBasis:            bucketBasis,
		BySymbol:                  bySymbol,
		ByMonth:                   byMonth,
		ByQuarter:                 byQuarter,
		ByBucket:                  byBucket,
	}
}

func compactLeaveOneOut(rows []compactDimensionRow) compactLeaveOneOutReport {
	if len(rows) == 0 {
		return compactLeaveOneOutReport{}
	}
	out := make([]compactLeaveOneOutRow, 0, len(rows))
	pass := true
	for _, removed := range rows {
		var grossProfit, grossLoss float64
		var events, clusters int
		for _, row := range rows {
			if row.Key == removed.Key {
				continue
			}
			grossProfit += row.GrossProfitBps
			grossLoss += row.GrossLossBps
			events += row.EventCount
			clusters += row.ClusterCount
		}
		net := roundMetric(grossProfit - grossLoss)
		expectancy := 0.0
		if events > 0 {
			expectancy = roundMetric(net / float64(events))
		}
		row := compactLeaveOneOutRow{
			RemovedKey:    removed.Key,
			EventCount:    events,
			ClusterCount:  clusters,
			NetBps:        net,
			ExpectancyBps: expectancy,
			ProfitFactor:  roundMetric(safePF(grossProfit, grossLoss)),
			Pass:          events > 0 && expectancy > 0 && safePF(grossProfit, grossLoss) > 1.0,
		}
		if !row.Pass {
			pass = false
		}
		out = append(out, row)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].ExpectancyBps < out[j].ExpectancyBps })
	report := compactLeaveOneOutReport{Supported: true, Rows: out, Pass: pass}
	if len(out) > 0 {
		report.WorstRemaining = &out[0]
	}
	return report
}

func compactMissingRequiredMetrics(metrics FundingMetrics, delay compactDelayReport, byMonth, byQuarter []compactDimensionRow) []string {
	var missing []string
	if metrics.BaselineCostBps == 0 {
		missing = append(missing, "baseline_cost_bps")
	}
	if !hasCostMetric(metrics.CostStress, 5) {
		missing = append(missing, "cost_5")
	}
	if !hasCostMetric(metrics.CostStress, 7.5) {
		missing = append(missing, "cost_7_5")
	}
	if !hasCostMetric(metrics.CostStress, 10) {
		missing = append(missing, "cost_10")
	}
	if reportMissing(delay.MissingMetrics, "delay_1") {
		missing = append(missing, "delay_1")
	}
	if len(byMonth) == 0 {
		missing = append(missing, "by_month")
	}
	if len(byQuarter) == 0 {
		missing = append(missing, "by_quarter")
	}
	return uniqueStrings(missing)
}

func compactSampleLabel(eventCount, clusterCount int, thresholds compactThresholds) string {
	if eventCount < thresholds.MinEventCount || clusterCount < thresholds.MinClusterCount {
		return "insufficient_sample"
	}
	if eventCount < thresholds.CalibrationReadyEventCount {
		return "early_sample"
	}
	return "calibration_ready"
}

func compactPromotionDecision(report compactCandidateReport, integrity compactIntegrityReport, thresholds compactThresholds) (string, bool, bool, []string, []string, string) {
	failures := append([]string(nil), report.MissingRequiredMetric...)
	warnings := append([]string(nil), report.MissingRequiredMetric...)
	if integrity.Status == "FAIL" {
		failures = append(failures, "integrity_fail")
	}
	if report.SampleLabel == "insufficient_sample" {
		failures = append(failures, "insufficient_sample")
	}
	if report.Baseline.ExpectancyCombined_5bpsBps <= 0 {
		failures = append(failures, "baseline_expectancy_non_positive")
	}
	if report.Baseline.PFCombined_5bps <= thresholds.MinBaselinePF {
		failures = append(failures, "baseline_pf_non_positive")
	}

	cost7 := metricByCost(report.CostStress, 7.5)
	cost10 := metricByCost(report.CostStress, 10)
	if cost7 == nil {
		failures = append(failures, "cost_7_5_missing")
	} else if cost7.ExpectancyBps <= 0 || cost7.PF <= 1.0 {
		failures = append(failures, "cost_7_5_fail")
	}
	if cost10 == nil {
		failures = append(failures, "cost_10_missing")
	} else if cost10.ExpectancyBps <= 0 || cost10.PF <= 1.0 {
		warnings = append(warnings, "cost_10_fragile")
	}

	if report.Concentration.TopSymbolContributionPct > thresholds.TopSymbolContributionFailPct {
		failures = append(failures, "concentration_symbol")
	}
	if report.Concentration.TopMonthContributionPct > thresholds.TopMonthContributionFailPct {
		failures = append(failures, "concentration_month")
	}
	if report.Concentration.TopQuarterContributionPct > thresholds.TopQuarterContributionWarnPct {
		warnings = append(warnings, "concentration_quarter")
	}
	if report.Concentration.TopBucketContributionPct > thresholds.TopBucketContributionWarnPct {
		warnings = append(warnings, "concentration_bucket")
	}
	if !report.LeaveOneSymbolOut.Pass {
		failures = append(failures, "leave_one_symbol_out_fail")
	}
	if !report.LeaveOneMonthOut.Pass {
		failures = append(failures, "leave_one_month_out_fail")
	}
	if !report.LeaveOneQuarterOut.Pass {
		failures = append(failures, "leave_one_quarter_out_fail")
	}
	delayFail := false
	if report.DelaySensitivity.Delay1 == nil || !report.DelaySensitivity.Delay1.Available {
		failures = append(failures, "delay_1_missing")
		delayFail = true
	} else {
		if report.DelaySensitivity.Delay1.ExpectancyBps <= 0 {
			failures = append(failures, "delay_1_expectancy_non_positive")
			delayFail = true
		}
		if report.DelaySensitivity.Delay1DecayPct != nil && *report.DelaySensitivity.Delay1DecayPct > thresholds.MaxDelayDecayFailPct {
			failures = append(failures, "delay_1_decay_fail")
			delayFail = true
		}
	}

	failures = uniqueStrings(failures)
	warnings = uniqueStrings(warnings)
	if containsAny(failures, []string{"integrity_fail", "insufficient_sample", "baseline_expectancy_non_positive", "baseline_pf_non_positive"}) {
		return "REJECTED", false, false, failures, warnings, "Fix baseline integrity/sample issues before any promotion review."
	}
	if len(report.MissingRequiredMetric) > 0 || containsAny(failures, []string{"cost_7_5_fail", "leave_one_symbol_out_fail", "leave_one_month_out_fail", "leave_one_quarter_out_fail", "concentration_symbol", "concentration_month"}) || delayFail {
		return "FRAGILE_RESEARCH_LEAD", false, false, failures, warnings, "Expand retained-summary coverage or improve robustness before promotion."
	}
	shadowEligible := report.Baseline.PFCombined_5bps > thresholds.MinResearchLeadPF &&
		cost7 != nil && cost7.ExpectancyBps > 0 && cost7.PF > thresholds.MinResearchLeadPF &&
		cost10 != nil && cost10.ExpectancyBps >= thresholds.MinCost10ExpectancyForShadowBps && cost10.PF >= thresholds.MinCost10PFForShadow &&
		report.Concentration.TopQuarterContributionPct <= thresholds.TopQuarterContributionWarnPct &&
		report.Concentration.TopBucketContributionPct <= thresholds.TopBucketContributionWarnPct &&
		report.SampleLabel == "calibration_ready" &&
		report.DelaySensitivity.Delay1DecayPct != nil && *report.DelaySensitivity.Delay1DecayPct <= thresholds.MaxShadowDelayDecayPct &&
		len(warnings) == 0
	if shadowEligible {
		return "SHADOW_CANDIDATE", true, true, failures, warnings, "Retained summaries support a shadow-candidate review, but ak-trader integration is still out of scope."
	}
	return "RESEARCH_LEAD", true, false, failures, warnings, "Candidate remains a research lead; monitor concentration and 10 bps stress before any promotion."
}

func compactCandidateBetter(a, b compactCandidateReport) bool {
	rank := map[string]int{"SHADOW_CANDIDATE": 4, "RESEARCH_LEAD": 3, "FRAGILE_RESEARCH_LEAD": 2, "REJECTED": 1}
	if rank[a.FinalLabel] != rank[b.FinalLabel] {
		return rank[a.FinalLabel] > rank[b.FinalLabel]
	}
	if a.Baseline.ExpectancyCombined_5bpsBps != b.Baseline.ExpectancyCombined_5bpsBps {
		return a.Baseline.ExpectancyCombined_5bpsBps > b.Baseline.ExpectancyCombined_5bpsBps
	}
	if stressedExpectancy(a) != stressedExpectancy(b) {
		return stressedExpectancy(a) > stressedExpectancy(b)
	}
	if a.Baseline.PFCombined_5bps != b.Baseline.PFCombined_5bps {
		return a.Baseline.PFCombined_5bps > b.Baseline.PFCombined_5bps
	}
	if a.EventCount != b.EventCount {
		return a.EventCount > b.EventCount
	}
	if a.ClusterCount != b.ClusterCount {
		return a.ClusterCount > b.ClusterCount
	}
	return a.EventCount > b.EventCount
}

func stressedExpectancy(report compactCandidateReport) float64 {
	if metric := metricByCost(report.CostStress, 10); metric != nil {
		return metric.ExpectancyBps
	}
	if metric := metricByCost(report.CostStress, 7.5); metric != nil {
		return metric.ExpectancyBps
	}
	return report.Baseline.ExpectancyCombined_5bpsBps
}

func addPositiveContribution(rows []compactDimensionRow) {
	total := 0.0
	for _, row := range rows {
		if row.NetBps > 0 {
			total += row.NetBps
		}
	}
	for i := range rows {
		rows[i].PositiveContribution = percentOfPositive(rows[i].NetBps, total)
	}
}

func topPositiveContribution(rows []compactDimensionRow) float64 {
	best := 0.0
	for _, row := range rows {
		if row.PositiveContribution > best {
			best = row.PositiveContribution
		}
	}
	return roundMetric(best)
}

func topContributionKey(rows []compactDimensionRow) string {
	bestKey := ""
	bestPct := -1.0
	for _, row := range rows {
		if row.PositiveContribution > bestPct {
			bestPct = row.PositiveContribution
			bestKey = row.Key
		}
	}
	return bestKey
}

func percentOfPositive(net, total float64) float64 {
	if net <= 0 || total <= 0 {
		return 0
	}
	return roundMetric(net / total * 100)
}

func distinctDimensionCount(rows []FundingAlphaSummaryRow, level string) int {
	seen := make(map[string]struct{})
	for _, row := range rows {
		switch level {
		case "month":
			seen[row.Month] = struct{}{}
		case "symbol":
			seen[row.Symbol] = struct{}{}
		case "quarter":
			seen[row.Quarter] = struct{}{}
		}
	}
	return len(seen)
}

func hasCostMetric(rows []FundingCostStressMetric, target float64) bool {
	return metricByCost(rows, target) != nil
}

func metricByCost(rows []FundingCostStressMetric, target float64) *FundingCostStressMetric {
	for i := range rows {
		if rows[i].CostBps == target {
			return &rows[i]
		}
	}
	return nil
}

func reportMissing(values []string, target string) bool {
	for _, value := range values {
		if value == target {
			return true
		}
	}
	return false
}

func containsAny(values, needles []string) bool {
	seen := make(map[string]struct{}, len(values))
	for _, value := range values {
		seen[value] = struct{}{}
	}
	for _, needle := range needles {
		if _, ok := seen[needle]; ok {
			return true
		}
	}
	return false
}

func uniqueStrings(values []string) []string {
	if len(values) == 0 {
		return nil
	}
	seen := make(map[string]struct{}, len(values))
	out := make([]string, 0, len(values))
	for _, value := range values {
		if value == "" {
			continue
		}
		if _, ok := seen[value]; ok {
			continue
		}
		seen[value] = struct{}{}
		out = append(out, value)
	}
	sort.Strings(out)
	return out
}

func sortedKeys(values map[string]struct{}) []string {
	out := make([]string, 0, len(values))
	for value := range values {
		out = append(out, value)
	}
	sort.Strings(out)
	return out
}

func buildRetainedCoverageReport(inputs retainedCoverageInputs, scan retainedSummaryScan, finalLabel string) retainedCoverageReport {
	foundSymbols := append([]string(nil), scan.FoundSymbols...)
	expectedSymbols := uniqueStrings(append([]string(nil), inputs.ExpectedSymbols...))
	expectedMonths := uniqueStrings(append([]string(nil), inputs.ExpectedMonths...))
	missingSymbols := diffStrings(expectedSymbols, foundSymbols)
	missingMonthsBySymbol := make(map[string][]string)
	for _, symbol := range expectedSymbols {
		if len(expectedMonths) == 0 {
			continue
		}
		missing := diffStrings(expectedMonths, scan.MonthsBySymbol[symbol])
		if len(missing) > 0 {
			missingMonthsBySymbol[symbol] = missing
		}
	}
	fullUniverseReady := len(expectedSymbols) > 0 && len(missingSymbols) == 0 && len(missingMonthsBySymbol) == 0 && len(scan.MalformedSummaries) == 0 && scan.SummaryFileCount > 0
	status := "insufficient_retained_coverage"
	reason := "no retained alpha summaries found"
	switch {
	case fullUniverseReady:
		status = "full_universe_ready"
		reason = ""
	case scan.SummaryFileCount == 0:
	case len(foundSymbols) == 1 && len(expectedSymbols) > 1:
		status = "single_symbol_only"
		reason = fmt.Sprintf("only %s is retained; missing %d expected symbols", foundSymbols[0], len(missingSymbols))
	case len(foundSymbols) > 0:
		status = "partial_universe_only"
		if len(missingSymbols) > 0 {
			reason = fmt.Sprintf("missing expected symbols: %s", strings.Join(missingSymbols, ", "))
		} else if len(missingMonthsBySymbol) > 0 {
			reason = "one or more expected months are missing from retained summaries"
		} else if len(scan.MalformedSummaries) > 0 {
			reason = "one or more retained summaries are malformed"
		} else {
			reason = "retained coverage is incomplete"
		}
	}
	paths := make([]string, 0, len(scan.SummaryFiles))
	for _, file := range scan.SummaryFiles {
		paths = append(paths, file.Path)
	}
	report := retainedCoverageReport{
		FinalLabel:              finalLabel,
		CoverageStatus:          status,
		ExpectedSymbols:         expectedSymbols,
		ExpectedMonths:          expectedMonths,
		FoundSymbols:            foundSymbols,
		MissingSymbols:          missingSymbols,
		MonthsBySymbol:          scan.MonthsBySymbol,
		MissingMonthsBySymbol:   missingMonthsBySymbol,
		SummaryFilesFound:       paths,
		FamiliesFound:           scan.FamiliesFound,
		SidesFound:              scan.SidesFound,
		HorizonsFound:           scan.HorizonsFound,
		ZeroEventSummaries:      scan.ZeroEventSummaries,
		MalformedSummaries:      scan.MalformedSummaries,
		SummaryFileCount:        scan.SummaryFileCount,
		MalformedFileCount:      len(scan.MalformedSummaries),
		ZeroEventCount:          len(scan.ZeroEventSummaries),
		FullUniverseReady:       fullUniverseReady,
		ReasonNotFullUniverse:   reason,
		ExpectedMonthRangeKnown: len(expectedMonths) > 0,
		RawRequired:             false,
	}
	if fullUniverseReady {
		report.ReasonNotFullUniverse = ""
	}
	return report
}

func diffStrings(expected, actual []string) []string {
	if len(expected) == 0 {
		return nil
	}
	set := make(map[string]struct{}, len(actual))
	for _, value := range actual {
		set[value] = struct{}{}
	}
	var out []string
	for _, value := range expected {
		if _, ok := set[value]; !ok {
			out = append(out, value)
		}
	}
	return out
}

func buildRankedInventoryReport(candidates []compactCandidateReport, coverage retainedCoverageReport, finalLabel string, cfg compactSummaryAnalyzerConfig) rankedInventoryReport {
	title := "Phase 10.8 Ranked Inventory"
	qualifier := coverageStatusTitle(coverage)
	if qualifier != "" {
		title += " - " + qualifier
	}
	scope := rankedInventoryScope(cfg)
	if scope != "all retained candidates" {
		title += " - " + scope
	}
	counts := map[string]int{
		"SHADOW_CANDIDATE":      0,
		"RESEARCH_LEAD":         0,
		"FRAGILE_RESEARCH_LEAD": 0,
		"REJECTED":              0,
	}
	for _, candidate := range candidates {
		counts[candidate.FinalLabel]++
	}
	return rankedInventoryReport{
		Title:                    title,
		Phase:                    "10.8C",
		SymbolUniverse:           strings.Join(coverage.FoundSymbols, ", "),
		CandidateScope:           scope,
		FinalLabel:               finalLabel,
		CoverageStatus:           coverage.CoverageStatus,
		FullUniverseReady:        coverage.FullUniverseReady,
		UniverseQualifier:        qualifier,
		RawRequired:              false,
		AkTraderTouched:          false,
		FoundSymbols:             coverage.FoundSymbols,
		MissingSymbols:           coverage.MissingSymbols,
		MonthsBySymbol:           coverage.MonthsBySymbol,
		CandidateCount:           len(candidates),
		InventoryLabelCounts:     counts,
		ShadowCandidateCount:     counts["SHADOW_CANDIDATE"],
		ResearchLeadCount:        counts["RESEARCH_LEAD"],
		FragileResearchLeadCount: counts["FRAGILE_RESEARCH_LEAD"],
		RejectedCount:            counts["REJECTED"],
		RankedInventory:          candidates,
	}
}

func rankedInventoryScope(cfg compactSummaryAnalyzerConfig) string {
	parts := make([]string, 0, 3)
	if cfg.Family != "" {
		parts = append(parts, cfg.Family)
	}
	if cfg.Side != "" {
		parts = append(parts, cfg.Side)
	}
	if cfg.Horizon != "" {
		parts = append(parts, cfg.Horizon)
	}
	if len(parts) == 0 {
		return "all retained candidates"
	}
	return strings.Join(parts, " ")
}

func coverageStatusTitle(coverage retainedCoverageReport) string {
	switch coverage.CoverageStatus {
	case "full_universe_ready":
		return "full retained universe"
	case "single_symbol_only":
		if len(coverage.FoundSymbols) == 1 {
			return coverage.FoundSymbols[0] + "-only retained universe"
		}
		return "single-symbol retained universe"
	case "partial_universe_only":
		return "partial retained universe"
	default:
		return "insufficient retained coverage"
	}
}

func writeCompactCoverageOutputs(reportsDir string, coverage retainedCoverageReport, inventory rankedInventoryReport, inventoryCfg compactSummaryAnalyzerConfig) error {
	if err := atomicWriteJSONFile(filepath.Join(reportsDir, "phase10_8c_retained_coverage.json"), coverage, "", "  ", 0644); err != nil {
		return err
	}
	if err := atomicWriteFile(filepath.Join(reportsDir, "phase10_8c_retained_coverage.md"), []byte(renderRetainedCoverageMarkdown(coverage)), 0644); err != nil {
		return err
	}
	jsonPath, mdPath := rankedInventoryOutputPaths(reportsDir, inventoryCfg)
	if err := atomicWriteJSONFile(jsonPath, inventory, "", "  ", 0644); err != nil {
		return err
	}
	return atomicWriteFile(mdPath, []byte(renderRankedInventoryMarkdown(inventory)), 0644)
}

func rankedInventoryOutputPaths(reportsDir string, cfg compactSummaryAnalyzerConfig) (string, string) {
	base := "phase10_8_ranked_inventory"
	suffixParts := make([]string, 0, 3)
	if cfg.Family != "" {
		suffixParts = append(suffixParts, cfg.Family)
	}
	if cfg.Side != "" {
		suffixParts = append(suffixParts, cfg.Side)
	}
	if cfg.Horizon != "" {
		suffixParts = append(suffixParts, cfg.Horizon)
	}
	if len(suffixParts) > 0 {
		base += "_" + strings.Join(suffixParts, "_")
	}
	return filepath.Join(reportsDir, base+".json"), filepath.Join(reportsDir, base+".md")
}

func renderRetainedCoverageMarkdown(report retainedCoverageReport) string {
	var sb strings.Builder
	sb.WriteString("# Phase 10.8C Retained Coverage\n\n")
	sb.WriteString(fmt.Sprintf("- Coverage status: `%s`\n", report.CoverageStatus))
	sb.WriteString(fmt.Sprintf("- full_universe_ready: `%t`\n", report.FullUniverseReady))
	sb.WriteString(fmt.Sprintf("- final_label: `%s`\n", report.FinalLabel))
	sb.WriteString(fmt.Sprintf("- raw_required: `%t`\n", report.RawRequired))
	if report.ReasonNotFullUniverse != "" {
		sb.WriteString(fmt.Sprintf("- reason_not_full_universe: `%s`\n", report.ReasonNotFullUniverse))
	}
	sb.WriteString("\n## Coverage\n\n")
	sb.WriteString(fmt.Sprintf("- expected symbols: `%s`\n", strings.Join(report.ExpectedSymbols, ", ")))
	sb.WriteString(fmt.Sprintf("- found symbols: `%s`\n", strings.Join(report.FoundSymbols, ", ")))
	sb.WriteString(fmt.Sprintf("- missing symbols: `%s`\n", strings.Join(report.MissingSymbols, ", ")))
	if report.ExpectedMonthRangeKnown {
		sb.WriteString(fmt.Sprintf("- expected months: `%s`\n", strings.Join(report.ExpectedMonths, ", ")))
	} else {
		sb.WriteString("- expected months: `unknown`\n")
	}
	sb.WriteString(fmt.Sprintf("- summary files found: `%d`\n", report.SummaryFileCount))
	sb.WriteString(fmt.Sprintf("- malformed summaries: `%d`\n", report.MalformedFileCount))
	sb.WriteString(fmt.Sprintf("- zero-event summaries: `%d`\n", report.ZeroEventCount))
	sb.WriteString("\n## Interpretation\n\n")
	sb.WriteString(fmt.Sprintf("- `%s`\n", report.CoverageStatus))
	sb.WriteString("\n## Months By Symbol\n\n")
	for _, symbol := range report.FoundSymbols {
		sb.WriteString(fmt.Sprintf("- `%s`: `%s`\n", symbol, strings.Join(report.MonthsBySymbol[symbol], ", ")))
		if missing := report.MissingMonthsBySymbol[symbol]; len(missing) > 0 {
			sb.WriteString(fmt.Sprintf("- `%s` missing expected months: `%s`\n", symbol, strings.Join(missing, ", ")))
		}
	}
	if len(report.MalformedSummaries) > 0 {
		sb.WriteString("\n## Malformed Summaries\n\n")
		for _, malformed := range report.MalformedSummaries {
			sb.WriteString(fmt.Sprintf("- `%s`: `%s`\n", malformed.Path, malformed.Error))
		}
	}
	return sb.String()
}

func renderRankedInventoryMarkdown(report rankedInventoryReport) string {
	var sb strings.Builder
	sb.WriteString("# " + report.Title + "\n\n")
	sb.WriteString(fmt.Sprintf("- symbol_universe: `%s`\n", report.SymbolUniverse))
	sb.WriteString(fmt.Sprintf("- candidate_scope: `%s`\n", report.CandidateScope))
	sb.WriteString(fmt.Sprintf("- coverage_status: `%s`\n", report.CoverageStatus))
	sb.WriteString(fmt.Sprintf("- full_universe_ready: `%t`\n", report.FullUniverseReady))
	sb.WriteString(fmt.Sprintf("- raw_required: `%t`\n", report.RawRequired))
	sb.WriteString(fmt.Sprintf("- ak_trader_touched: `%t`\n", report.AkTraderTouched))
	sb.WriteString(fmt.Sprintf("- candidate_count: `%d`\n", report.CandidateCount))
	sb.WriteString("\n## Label Counts\n\n")
	sb.WriteString(fmt.Sprintf("- SHADOW_CANDIDATE: `%d`\n", report.ShadowCandidateCount))
	sb.WriteString(fmt.Sprintf("- RESEARCH_LEAD: `%d`\n", report.ResearchLeadCount))
	sb.WriteString(fmt.Sprintf("- FRAGILE_RESEARCH_LEAD: `%d`\n", report.FragileResearchLeadCount))
	sb.WriteString(fmt.Sprintf("- REJECTED: `%d`\n", report.RejectedCount))
	sb.WriteString("\n## Ranked Candidates\n\n")
	sb.WriteString("| Candidate | Label | Baseline Exp | Stressed Exp | PF | Events | Clusters |\n|---|---|---:|---:|---:|---:|---:|\n")
	for _, candidate := range report.RankedInventory {
		sb.WriteString(fmt.Sprintf("| `%s` | `%s` | `%.6f` | `%.6f` | `%.6f` | `%d` | `%d` |\n",
			candidate.CandidateKey,
			candidate.FinalLabel,
			candidate.Baseline.ExpectancyCombined_5bpsBps,
			stressedExpectancy(candidate),
			candidate.Baseline.PFCombined_5bps,
			candidate.EventCount,
			candidate.ClusterCount,
		))
	}
	return sb.String()
}

func writeCompactRobustnessOutputs(reportsDir string, report compactRobustnessReport, gate compactPromotionGateReport) error {
	if err := atomicWriteJSONFile(filepath.Join(reportsDir, "phase10_8_compact_robustness.json"), report, "", "  ", 0644); err != nil {
		return err
	}
	if err := atomicWriteJSONFile(filepath.Join(reportsDir, "phase10_8_promotion_gate.json"), gate, "", "  ", 0644); err != nil {
		return err
	}
	if err := atomicWriteFile(filepath.Join(reportsDir, "phase10_8_compact_robustness.md"), []byte(renderCompactRobustnessMarkdown(report)), 0644); err != nil {
		return err
	}
	return atomicWriteFile(filepath.Join(reportsDir, "phase10_8_promotion_gate.md"), []byte(renderCompactPromotionMarkdown(gate)), 0644)
}

func renderCompactRobustnessMarkdown(report compactRobustnessReport) string {
	target := report.TargetCandidate
	var sb strings.Builder
	sb.WriteString("# Phase 10.8 Compact Robustness\n\n")
	sb.WriteString(fmt.Sprintf("- Final label: `%s`\n", report.FinalLabel))
	sb.WriteString(fmt.Sprintf("- Candidate/family/side/horizon evaluated: `%s` `%s` `%s`\n", target.CandidateKey, target.Family, target.Horizon))
	sb.WriteString(fmt.Sprintf("- Compact-summary-only status: `%t`\n", report.CompactSummaryOnly))
	sb.WriteString(fmt.Sprintf("- Raw files required: `%t`\n", report.RawRequired))
	sb.WriteString(fmt.Sprintf("- ak-trader integration: `%s`\n\n", report.AkTraderIntegration))

	sb.WriteString("## Integrity\n\n")
	sb.WriteString("| Metric | Value |\n|---|---|\n")
	sb.WriteString(fmt.Sprintf("| status | `%s` |\n", report.Integrity.Status))
	sb.WriteString(fmt.Sprintf("| months expected/found | `%d / %d` |\n", report.Integrity.MonthsExpected, report.Integrity.MonthsFound))
	sb.WriteString(fmt.Sprintf("| symbols expected/found | `%d / %d` |\n", report.Integrity.SymbolsExpected, report.Integrity.SymbolsFound))
	sb.WriteString(fmt.Sprintf("| missing summaries | `%d` |\n", len(report.Integrity.MissingSummaries)))
	sb.WriteString(fmt.Sprintf("| zero-event summaries | `%d` |\n", len(report.Integrity.ZeroEventSummaries)))
	sb.WriteString(fmt.Sprintf("| missing required metrics | `%d` |\n\n", len(target.MissingRequiredMetric)))

	sb.WriteString("## Baseline Metrics\n\n")
	sb.WriteString("| Metric | Value |\n|---|---|\n")
	sb.WriteString(fmt.Sprintf("| event_count | `%d` |\n", target.EventCount))
	sb.WriteString(fmt.Sprintf("| cluster_count | `%d` |\n", target.ClusterCount))
	sb.WriteString(fmt.Sprintf("| months represented | `%d` |\n", target.MonthsRepresented))
	sb.WriteString(fmt.Sprintf("| symbols represented | `%d` |\n", target.SymbolsRepresented))
	if target.BaselineCostBps != nil {
		sb.WriteString(fmt.Sprintf("| baseline cost bps | `%.1f` |\n", *target.BaselineCostBps))
	} else {
		sb.WriteString("| baseline cost bps | `MISSING` |\n")
	}
	sb.WriteString(fmt.Sprintf("| expectancy bps | `%.6f` |\n", target.Baseline.ExpectancyCombined_5bpsBps))
	sb.WriteString(fmt.Sprintf("| profit factor | `%.6f` |\n\n", target.Baseline.PFCombined_5bps))

	sb.WriteString("## Cost Stress\n\n")
	sb.WriteString("| Cost Bps | Event Count | Net Bps | Expectancy Bps | PF |\n|---|---:|---:|---:|---:|\n")
	for _, row := range target.CostStress {
		sb.WriteString(fmt.Sprintf("| `%.1f` | `%d` | `%.6f` | `%.6f` | `%.6f` |\n", row.CostBps, row.EventCount, row.NetBps, row.ExpectancyBps, row.PF))
	}
	sb.WriteString("\n")
	writeCompactDimensionTable(&sb, "Concentration By Symbol", target.Concentration.BySymbol)
	writeCompactDimensionTable(&sb, "Concentration By Month", target.Concentration.ByMonth)
	writeCompactDimensionTable(&sb, "Concentration By Quarter", target.Concentration.ByQuarter)
	writeCompactLeaveOneOutTable(&sb, "Leave-One-Symbol-Out", target.LeaveOneSymbolOut)
	writeCompactLeaveOneOutTable(&sb, "Leave-One-Month-Out", target.LeaveOneMonthOut)
	writeCompactLeaveOneOutTable(&sb, "Leave-One-Quarter-Out", target.LeaveOneQuarterOut)

	sb.WriteString("## Delay Sensitivity\n\n")
	sb.WriteString("| Delay | Available | Expectancy Bps | PF |\n|---|---|---:|---:|\n")
	for _, row := range []*FundingDelayStressMetric{target.DelaySensitivity.Baseline, target.DelaySensitivity.Delay1, target.DelaySensitivity.Delay2} {
		if row == nil {
			continue
		}
		sb.WriteString(fmt.Sprintf("| `%s` | `%t` | `%.6f` | `%.6f` |\n", row.Label, row.Available, row.ExpectancyBps, row.PF))
	}
	if target.DelaySensitivity.Delay1DecayPct != nil {
		sb.WriteString(fmt.Sprintf("\n- delay_1 decay from baseline: `%.6f%%`\n", *target.DelaySensitivity.Delay1DecayPct))
	}

	sb.WriteString("\n## Blockers\n\n")
	if len(target.Failures) == 0 {
		sb.WriteString("- none\n")
	} else {
		for _, failure := range target.Failures {
			sb.WriteString(fmt.Sprintf("- `%s`\n", failure))
		}
	}
	sb.WriteString("\n## Recommended Next Action\n\n")
	sb.WriteString(fmt.Sprintf("- %s\n", target.RecommendedNextAction))
	return sb.String()
}

func writeCompactDimensionTable(sb *strings.Builder, title string, rows []compactDimensionRow) {
	sb.WriteString(fmt.Sprintf("## %s\n\n", title))
	sb.WriteString("| Key | Event Count | Cluster Count | Net Bps | Expectancy Bps | PF | Positive Contribution % |\n|---|---:|---:|---:|---:|---:|---:|\n")
	for _, row := range rows {
		sb.WriteString(fmt.Sprintf("| `%s` | `%d` | `%d` | `%.6f` | `%.6f` | `%.6f` | `%.6f` |\n", row.Key, row.EventCount, row.ClusterCount, row.NetBps, row.ExpectancyBps, row.ProfitFactor, row.PositiveContribution))
	}
	sb.WriteString("\n")
}

func writeCompactLeaveOneOutTable(sb *strings.Builder, title string, report compactLeaveOneOutReport) {
	sb.WriteString(fmt.Sprintf("## %s\n\n", title))
	sb.WriteString("| Removed | Event Count | Cluster Count | Net Bps | Expectancy Bps | PF | Pass |\n|---|---:|---:|---:|---:|---:|---|\n")
	for _, row := range report.Rows {
		sb.WriteString(fmt.Sprintf("| `%s` | `%d` | `%d` | `%.6f` | `%.6f` | `%.6f` | `%t` |\n", row.RemovedKey, row.EventCount, row.ClusterCount, row.NetBps, row.ExpectancyBps, row.ProfitFactor, row.Pass))
	}
	sb.WriteString("\n")
}

func renderCompactPromotionMarkdown(report compactPromotionGateReport) string {
	var sb strings.Builder
	sb.WriteString("# Phase 10.8 Promotion Gate\n\n")
	sb.WriteString(fmt.Sprintf("- Final label: `%s`\n", report.FinalLabel))
	sb.WriteString(fmt.Sprintf("- Promotion allowed: `%t`\n", report.PromotionAllowed))
	sb.WriteString(fmt.Sprintf("- Shadow candidate: `%t`\n", report.ShadowCandidate))
	sb.WriteString(fmt.Sprintf("- Raw files required: `%t`\n", report.RawRequired))
	sb.WriteString(fmt.Sprintf("- Integrity status: `%s`\n", report.IntegrityStatus))
	sb.WriteString(fmt.Sprintf("- Candidate: `%s`\n", report.Metrics.CandidateKey))
	sb.WriteString(fmt.Sprintf("- ak-trader integration: `%s`\n\n", report.AkTraderIntegration))
	sb.WriteString("## Failures\n\n")
	if len(report.Failures) == 0 {
		sb.WriteString("- none\n")
	} else {
		for _, failure := range report.Failures {
			sb.WriteString(fmt.Sprintf("- `%s`\n", failure))
		}
	}
	sb.WriteString("\n## Warnings\n\n")
	if len(report.Warnings) == 0 {
		sb.WriteString("- none\n")
	} else {
		for _, warning := range report.Warnings {
			sb.WriteString(fmt.Sprintf("- `%s`\n", warning))
		}
	}
	return sb.String()
}
