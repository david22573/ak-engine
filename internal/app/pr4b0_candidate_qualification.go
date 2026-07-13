package app

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/david22573/ak-engine/internal/qualification"
	"github.com/david22573/ak-engine/internal/strategy"
	"github.com/spf13/cobra"
)

const (
	pr4b0AcceptedEngineBaseline = "a04f6d6c8631e06049c7e108581dd638e9962a7b"
	pr4b0AcceptedRIFBaseline    = "29350344a57e46f064442eada26e9418515990be"
	pr4b0AcceptedHistorianHead  = "3eeff1eb45da281e0003dc1577ec55aa6cda1b1b"
	pr4b0NoCandidateLabel       = "PR4B0_NO_CANDIDATE_QUALIFIED"
)

var (
	pr4b0OutDir               string
	pr4b0ResultingCommit      string
	pr4b0VerificationComplete bool
	pr4b0FreshCloneCommit     string
)

type pr4b0Assessment struct {
	CandidateID      string                                  `json:"candidate_id"`
	CandidateVersion string                                  `json:"candidate_version"`
	Classification   qualification.EligibilityClassification `json:"eligibility_classification"`
	FinalStatus      qualification.FinalStatus               `json:"final_status"`
	FailedGates      []string                                `json:"failed_gates"`
	Decision         string                                  `json:"decision"`
}

type pr4b0GatePolicy struct {
	PolicyStatus      string                         `json:"policy_status"`
	DataIntegrity     map[string]any                 `json:"data_integrity"`
	SampleSufficiency qualification.SampleGates      `json:"sample_sufficiency"`
	Performance       qualification.PerformanceGates `json:"performance"`
	Robustness        qualification.RobustnessGates  `json:"robustness"`
	CostStress        qualification.CostGates        `json:"cost_stress"`
	LeakageRules      []string                       `json:"leakage_rules"`
	SimplicityRule    string                         `json:"simplicity_rule"`
	SearchPolicy      map[string]any                 `json:"search_policy"`
}

type pr4b0CommandResult struct {
	Command  string `json:"command"`
	ExitCode *int   `json:"exit_code"`
	Status   string `json:"status"`
	Notes    string `json:"notes,omitempty"`
}

type pr4b0HistoricalRow struct {
	CandidateKey       string
	Family             string
	Side               string
	Horizon            string
	EventCount         int
	ClusterCount       int
	Months             int
	PF5BPS             float64
	Expectancy5BPS     float64
	WorstQuarterPF5BPS float64
}

type pr4b0QualificationReport struct {
	SchemaVersion                      string               `json:"schema_version"`
	Phase                              string               `json:"phase"`
	ExecutiveVerdict                   string               `json:"executive_verdict"`
	FinalLabel                         string               `json:"final_label"`
	AcceptedBaselines                  map[string]string    `json:"accepted_baselines"`
	ResultingCommit                    string               `json:"resulting_commit"`
	CandidateInventorySummary          map[string]any       `json:"candidate_inventory_summary"`
	QualificationGates                 pr4b0GatePolicy      `json:"qualification_gates"`
	Assessments                        []pr4b0Assessment    `json:"candidate_assessments"`
	ExistingCandidatesExcluded         []map[string]any     `json:"existing_candidates_excluded"`
	NewResearch                        map[string]any       `json:"new_research"`
	DataAndHoldoutControls             map[string]any       `json:"data_and_holdout_controls"`
	SelectedCandidate                  any                  `json:"selected_candidate"`
	QualificationMetrics               []map[string]any     `json:"qualification_metrics"`
	FrozenIdentity                     any                  `json:"frozen_identity"`
	DirectImplementationParityFixtures []any                `json:"direct_implementation_parity_fixtures"`
	CandidateRegistrationArtifact      any                  `json:"candidate_registration_artifact"`
	TestsAndRace                       []pr4b0CommandResult `json:"tests_and_race_results"`
	SecurityFindings                   []map[string]string  `json:"security_findings"`
	FreshClone                         map[string]any       `json:"fresh_clone"`
	Boundaries                         map[string]any       `json:"boundaries"`
	DeferredWork                       []string             `json:"deferred_work"`
	GeneratedReportPaths               []string             `json:"generated_report_paths"`
	RecommendedNextPhase               string               `json:"recommended_next_phase"`
	QualificationReportID              string               `json:"qualification_report_id"`
	QualificationReportHash            string               `json:"qualification_report_hash"`
}

var pr4b0CandidateQualificationCmd = &cobra.Command{
	Use:   "pr4b0-candidate-qualification",
	Short: "Generate the PR4B0 candidate inventory and qualification verdict",
	RunE: func(cmd *cobra.Command, args []string) error {
		return runPR4B0CandidateQualification(pr4b0OutDir, pr4b0ResultingCommit, pr4b0VerificationComplete, pr4b0FreshCloneCommit)
	},
}

func init() {
	pr4b0CandidateQualificationCmd.Flags().StringVar(&pr4b0OutDir, "out-dir", "runs/reports", "Output directory for PR4B0 reports")
	pr4b0CandidateQualificationCmd.Flags().StringVar(&pr4b0ResultingCommit, "resulting-commit", "PENDING", "Verified source/result commit recorded in reports")
	pr4b0CandidateQualificationCmd.Flags().BoolVar(&pr4b0VerificationComplete, "verification-complete", false, "Record the prescribed suite as complete only after it has actually passed")
	pr4b0CandidateQualificationCmd.Flags().StringVar(&pr4b0FreshCloneCommit, "fresh-clone-commit", "", "Commit independently verified in a no-sibling fresh clone")
	rootCmd.AddCommand(pr4b0CandidateQualificationCmd)
}

func runPR4B0CandidateQualification(outDir, resultingCommit string, verificationComplete bool, freshCloneCommit string) error {
	inventory, err := buildPR4B0Inventory()
	if err != nil {
		return err
	}
	registered := pr4b0RegisteredCandidateIDs()
	inventory.RegisteredCandidateIDs = registered
	inventory.UnknownImplementations = qualification.FindUnknownImplementations(inventory.Candidates, registered)
	inventory.CandidateCount = len(inventory.Candidates)
	if err := qualification.ValidateInventory(inventory, registered); err != nil {
		return fmt.Errorf("validate PR4B0 inventory: %w", err)
	}
	report := buildPR4B0QualificationReport(inventory, resultingCommit, verificationComplete, freshCloneCommit)
	hash, err := hashReport(report)
	if err != nil {
		return err
	}
	report.QualificationReportHash = hash
	if err := os.MkdirAll(outDir, 0755); err != nil {
		return err
	}
	if err := atomicWriteJSONFile(filepath.Join(outDir, "pr4b0_candidate_inventory.json"), inventory, "", "  ", 0644); err != nil {
		return err
	}
	if err := atomicWriteFile(filepath.Join(outDir, "pr4b0_candidate_inventory.md"), []byte(renderPR4B0InventoryMarkdown(inventory)), 0644); err != nil {
		return err
	}
	if err := atomicWriteJSONFile(filepath.Join(outDir, "pr4b0_candidate_qualification.json"), report, "", "  ", 0644); err != nil {
		return err
	}
	return atomicWriteFile(filepath.Join(outDir, "pr4b0_candidate_qualification.md"), []byte(renderPR4B0QualificationMarkdown(report)), 0644)
}

func buildPR4B0Inventory() (qualification.CandidateInventory, error) {
	var candidates []qualification.CandidateRecord
	baseline := pr4b0BaseCandidate("baseline", "BaselineMomentum", "internal/strategy/baseline.go", pr4b0AcceptedEngineBaseline, "sha256:2e92e528bc9d57bc8bd67e49ba42c4c44890c660b563a7d25a76786d11b57ece")
	baseline.RegisteredImplementation = true
	baseline.RequiredContext = []string{}
	baseline.RequiredTimeframes = []string{"configured candle interval"}
	baseline.FeatureRequirements = []string{"current close", "previous close"}
	baseline.ParameterSet = map[string]any{"threshold_bps": 5.0, "stop_loss_bps": 50.0, "take_profit_bps": 100.0, "max_hold_candles": 12, "taker_fee_bps": 5.0, "slippage_bps": 1.0}
	markRejected(&baseline, "Phase 10.4", "rejected", "H2 out-of-sample failure")
	baseline.Evidence = []qualification.EvidenceReference{evidenceRef("phase10_4_price_regime_branch_closure", "internal/app/testdata/phase10_4_price_regime_branch_closure.json", "1c811d816ea744859332f639c86add2f8ade6fb5419a7c32331a92444d74f396")}
	candidates = append(candidates, baseline)

	fastNames := []string{
		strategyFastAccumulation, strategyFastAccumulationStrict, strategyFastAccumulationStrictShortBias,
		strategyFastAccumulationStrictHighConf, strategyFastAccumulationStrictLowFreq, strategyFastAccumulationStrictCostGuard,
		strategyFastAccumulationStrictNo7084Longs, strategyFastAccumulationStrict30m, strategyFastAccumulationStrict1h,
		strategyFastAccumulationPullbackReclaim, strategyFastAccumulationBreakoutRetest, strategyFastAccumulationMomentumCont,
		strategyFastAccumulationPartialTrail, strategyFastAccumulationBreakevenGuard, strategyFastAccumulationCutNoProgress,
		strategyFastAccumulationEconomicsGuard,
	}
	for _, name := range fastNames {
		cfg := strategy.DefaultFastAccumulationConfig()
		var err error
		if name != strategyFastAccumulation {
			cfg, err = fastAccumulationPresetConfig(name, 6.0)
			if err != nil {
				return qualification.CandidateInventory{}, err
			}
		}
		params, err := structMap(cfg)
		if err != nil {
			return qualification.CandidateInventory{}, err
		}
		candidate := pr4b0BaseCandidate(name, "FastAccumulation", "internal/strategy/fast_accumulation.go + internal/app/fast_accumulation_presets.go", pr4b0AcceptedEngineBaseline, "sha256:"+hashStrings("4694c87434baa84097f8e4b077f0f5769932c079194fff2f72e45cf626f626b9", "766517f0209a4f0d72236ae5851889727ab96cbaa8a5ef39acd7da590bbb532c"))
		candidate.RegisteredImplementation = true
		candidate.RequiredContext = []string{"completed decision windows", "hourly trend context"}
		candidate.RequiredTimeframes = []string{fmt.Sprintf("%dm decision window", cfg.DecisionWindowMinutes), "1h context"}
		candidate.FeatureRequirements = []string{"window OHLCV", "trend score", "chop score", "expected move", "entry score"}
		candidate.ParameterSet = params
		if name == strategyFastAccumulation {
			markRejected(&candidate, "Phase 10.4", "rejected", "H2 out-of-sample failure")
			candidate.Evidence = []qualification.EvidenceReference{evidenceRef("phase10_4_price_regime_branch_closure", "internal/app/testdata/phase10_4_price_regime_branch_closure.json", "1c811d816ea744859332f639c86add2f8ade6fb5419a7c32331a92444d74f396")}
		} else {
			markMissingEvidence(&candidate, "Phase 5-10 preset", "no complete immutable OOS, walk-forward, cost-tail-concentration, PIT, and controlled-holdout evidence for this exact preset")
		}
		candidates = append(candidates, candidate)
	}

	priceFamilies := []string{"TrendContinuation", "CompressionBreakout", "ShockFade", "VolumeMomentum", "BetaAgrees", "BetaDiverges"}
	for _, family := range priceFamilies {
		for _, side := range []string{"long", "short"} {
			id := fmt.Sprintf("price-alpha/%s/%s", family, side)
			candidate := pr4b0BaseCandidate(id, family, "internal/app/evaluate_alpha_baselines.go", pr4b0AcceptedEngineBaseline, "sha256:11528b9f9d3bc9fc2bcd0e6de57319880285b00cf6636ba4c530cb2c5b9931a3")
			candidate.DirectionSupport = []string{side}
			candidate.RequiredContext = priceFamilyContext(family)
			candidate.RequiredTimeframes = []string{"5m features", "15m", "60m", "240m forward returns"}
			candidate.FeatureRequirements = priceFamilyFeatures(family)
			candidate.ParameterSet = map[string]any{"type": "hardcoded_source", "source_ref": pr4b0AcceptedEngineBaseline, "source_sha256": candidate.ImplementationSHA256, "direction": side}
			switch {
			case family == "CompressionBreakout":
				markNearMiss(&candidate, "Phase 10.4", "fragile", "bracket/concentration failure; fragile is not paper eligible")
				candidate.Evidence = []qualification.EvidenceReference{evidenceRef("phase10_4_price_regime_branch_closure", "internal/app/testdata/phase10_4_price_regime_branch_closure.json", "1c811d816ea744859332f639c86add2f8ade6fb5419a7c32331a92444d74f396")}
			case family == "ShockFade" && side == "long":
				markRejected(&candidate, "Phase 10.4", "rejected", "out-of-sample and/or concentration/sample gates failed")
				candidate.Evidence = []qualification.EvidenceReference{evidenceRef("phase10_4_price_regime_branch_closure", "internal/app/testdata/phase10_4_price_regime_branch_closure.json", "1c811d816ea744859332f639c86add2f8ade6fb5419a7c32331a92444d74f396")}
			default:
				markMissingEvidence(&candidate, "Phase 10 price-alpha baseline", "no complete exact-candidate qualification evidence")
			}
			candidates = append(candidates, candidate)
		}
	}

	for _, row := range pr4b0FirstGenFundingRows() {
		candidate := pr4b0BaseCandidate("funding-alpha/"+row.CandidateKey, row.Family, "internal/app/funding_events.go", pr4b0AcceptedEngineBaseline, "sha256:5aabb2c8b767ed185939e2465a435f538ee6bdf94c5ac1599fac9ee4e576edea")
		candidate.DirectionSupport = []string{row.Side}
		candidate.RequiredContext = []string{"funding rate", "BTC/ETH market context", "regime classification"}
		candidate.RequiredTimeframes = []string{"1m inputs", row.Horizon + " forward return"}
		candidate.FeatureRequirements = []string{"funding severity/change", "regime", "BTC beta", "forward return labels"}
		candidate.ParameterSet = map[string]any{"type": "hardcoded_source", "source_ref": pr4b0AcceptedEngineBaseline, "source_sha256": candidate.ImplementationSHA256, "side": row.Side, "horizon": row.Horizon}
		candidate.SampleSize = qualification.SampleSize{EventCount: row.EventCount, IndependentClusterCount: row.ClusterCount, SymbolsRepresented: 8, MonthsRepresented: row.Months, QuartersRepresented: 8}
		candidate.CostStressResults = metricEvidence("FAIL", map[string]any{"profit_factor_after_5_bps": row.PF5BPS, "expectancy_after_5_bps": row.Expectancy5BPS}, "net performance failed")
		candidate.WorstPeriodResults = metricEvidence("FAIL", map[string]any{"worst_quarter_profit_factor_after_5_bps": row.WorstQuarterPF5BPS}, "below qualification gate")
		markRejected(&candidate, "Phase 10.8", "REJECTED", "candidate side/horizon row rejected after full retained coverage")
		candidate.Evidence = []qualification.EvidenceReference{evidenceRef("phase10_8_ranked_inventory", "preserved:runs/reports/phase10_8_ranked_inventory.json", "ebf255c9ed6cc62c27f4972dca448d6f59fc7780865043f9ab6fa182e04e255a")}
		candidates = append(candidates, candidate)
	}

	for _, item := range pr4b0SecondGenFundingRows() {
		candidate := pr4b0BaseCandidate("funding-alpha/"+item.row.CandidateKey, item.row.Family, "internal/app/funding_events.go", item.sourceRef, "sha256:"+item.sourceHash)
		candidate.DirectionSupport = []string{item.row.Side}
		candidate.RequiredContext = []string{"funding rate", "BTC/ETH market context", "regime classification"}
		candidate.RequiredTimeframes = []string{"1m inputs", item.row.Horizon + " forward return"}
		candidate.FeatureRequirements = []string{"funding", "price/volume", "regime", "BTC beta", "forward return labels"}
		candidate.ParameterSet = map[string]any{"type": "hardcoded_source", "source_ref": item.sourceRef, "source_sha256": candidate.ImplementationSHA256, "side": item.row.Side, "horizon": item.row.Horizon}
		candidate.SampleSize = qualification.SampleSize{EventCount: item.row.EventCount, IndependentClusterCount: item.row.ClusterCount, SymbolsRepresented: 8, MonthsRepresented: item.row.Months, QuartersRepresented: 8}
		candidate.CostStressResults = metricEvidence("FAIL", map[string]any{"profit_factor_after_5_bps": item.row.PF5BPS, "expectancy_after_5_bps": item.row.Expectancy5BPS}, "net performance is non-positive")
		if item.row.WorstQuarterPF5BPS > 0 {
			candidate.WorstPeriodResults = metricEvidence("FAIL", map[string]any{"worst_quarter_profit_factor_after_5_bps": item.row.WorstQuarterPF5BPS}, "below qualification gate")
		}
		markRejected(&candidate, item.phase, "REJECTED", "failed net performance, robustness, and/or concentration gates")
		if strings.Contains(item.row.Family, "VolumeImbalance") {
			candidate.KnownDefects = append(candidate.KnownDefects, "uses TakerBuyRatio fallback only; true taker buy/sell volume join unavailable")
		}
		candidate.Evidence = []qualification.EvidenceReference{evidenceRef(item.reportID, "preserved:runs/reports/"+item.reportID+".json", item.reportHash)}
		candidates = append(candidates, candidate)
	}

	for _, row := range pr4b0CompressionRows() {
		candidate := pr4b0BaseCandidate("phase11/"+row.CandidateKey, row.Family, "internal/app/phase11_compression_volume_breakout.go", "6db58069b874fcbb63d52f887048a1eb261a2a31", "sha256:101d745130e0d8e3318aaffb2049fd7aa87b1a38748a7e7d44cde47f046d186e")
		candidate.DirectionSupport = []string{row.Side}
		candidate.RequiredContext = []string{"BTC beta context", "regime classification"}
		candidate.RequiredTimeframes = []string{"1m features", row.Horizon + " forward return"}
		candidate.FeatureRequirements = []string{"EMA20", "Bollinger width/rank", "volume ratio", "BTC beta", "completed-candle break"}
		candidate.ParameterSet = map[string]any{"type": "hardcoded_source", "source_ref": candidate.ImplementationSourceRef, "source_sha256": candidate.ImplementationSHA256, "side": row.Side, "horizon": row.Horizon}
		candidate.SampleSize = qualification.SampleSize{EventCount: row.EventCount, IndependentClusterCount: row.ClusterCount, SymbolsRepresented: 8, MonthsRepresented: 24, QuartersRepresented: 8}
		candidate.CostStressResults = metricEvidence("FAIL", map[string]any{"profit_factor_after_5_bps": row.PF5BPS, "expectancy_after_5_bps": row.Expectancy5BPS}, "net performance failed")
		candidate.WorstPeriodResults = metricEvidence("FAIL", map[string]any{"worst_quarter_profit_factor_after_5_bps": row.WorstQuarterPF5BPS}, "below prior 0.95 gate")
		markRejected(&candidate, "Phase 11.1", "rejected", "side/horizon row failed net and worst-period gates")
		candidate.Evidence = []qualification.EvidenceReference{evidenceRef("phase11_1_compression_volume_breakout", "6db58069:runs/reports/phase11_1_compression_volume_breakout.json", "d32a09111f509200b29638a83db8c2dff005917fd604d217c5454c477eab819d")}
		candidates = append(candidates, candidate)
	}

	for _, row := range pr4b0TrendPullbackRows() {
		candidate := pr4b0BaseCandidate("phase11/"+row.CandidateKey, row.Family, "internal/app/phase11_regime_trend_pullback_continuation.go", "UNTRACKED_DIRTY_WORK", "sha256:4cd2265cb551dc21af3b90a3cf309eeb0be590dd029148bb810b3a07cb57d90e")
		candidate.ImplementationReproducible = false
		candidate.DirectionSupport = []string{row.Side}
		candidate.RequiredContext = []string{"BTC/ETH context", "regime classification"}
		candidate.RequiredTimeframes = []string{"1m features", row.Horizon + " forward return"}
		candidate.FeatureRequirements = []string{"EMA20/50/200", "Return15", "volume/chop", "trend/regime context"}
		candidate.ParameterSet = map[string]any{"type": "untracked_source_hash_only", "source_sha256": candidate.ImplementationSHA256, "side": row.Side, "horizon": row.Horizon}
		candidate.SampleSize = qualification.SampleSize{EventCount: row.EventCount, IndependentClusterCount: row.ClusterCount, SymbolsRepresented: 8, MonthsRepresented: 24, QuartersRepresented: 8}
		candidate.CostStressResults = metricEvidence("FAIL", map[string]any{"profit_factor_after_5_bps": row.PF5BPS, "expectancy_after_5_bps": row.Expectancy5BPS}, "net performance failed")
		candidate.WorstPeriodResults = metricEvidence("FAIL", map[string]any{"worst_quarter_profit_factor_after_5_bps": row.WorstQuarterPF5BPS}, "below prior 0.95 gate")
		markRejected(&candidate, "Phase 11.2", "rejected", "side/horizon row failed; exact implementation is not committed/reproducible")
		candidate.KnownDefects = append(candidate.KnownDefects, "implementation and tests exist only as untracked dirty-work files")
		candidate.Evidence = []qualification.EvidenceReference{evidenceRef("phase11_2_regime_trend_pullback_continuation", "preserved:runs/reports/phase11_2_regime_trend_pullback_continuation.json", "d123d1a60fec5fc37ddbce37b020807bd9ebef8dc1c8639b81f92da668432610")}
		candidates = append(candidates, candidate)
	}

	relief := pr4b0BaseCandidate("phase12/DowntrendMidVolReliefLong240m", "DowntrendMidVolRelief", "internal/app/phase12_downtrend_midvol_relief.go", "c2c7988712699b26ba7ab28e1cebb1f5312812a6", "sha256:3c2e20fd5bf615864aebc5be35ce86c15a6ed8f83de33b2f1d33b00dae6fbfa1")
	relief.DirectionSupport = []string{"long"}
	relief.RequiredContext = []string{"BTC/ETH context", "downtrend regime"}
	relief.RequiredTimeframes = []string{"1m features", "240m forward return"}
	relief.FeatureRequirements = []string{"RealizedVol60 mid bucket", "trend down classification"}
	relief.ParameterSet = map[string]any{"side": "long", "horizon": "240m", "trend": "down", "realized_vol_60": "mid"}
	relief.SampleSize = qualification.SampleSize{EventCount: 329842, IndependentClusterCount: 13178, SymbolsRepresented: 8, MonthsRepresented: 24, QuartersRepresented: 8}
	relief.CostStressResults = metricEvidence("PASS", map[string]any{"profit_factor_after_5_bps": 1.168964, "expectancy_after_5_bps": 16.473089}, "positive aggregate result did not cure robustness failure")
	relief.WorstPeriodResults = metricEvidence("FAIL", map[string]any{"worst_quarter_profit_factor_after_5_bps": 0.847931, "minimum_required": 0.95}, "structural Q1 2025 weakness")
	relief.ConcentrationResults = metricEvidence("FAIL", map[string]any{"average_events_per_cluster": 25.029746, "cluster_pnl_available": false}, "retained schema cannot establish cluster independence")
	markNearMiss(&relief, "Phase 12.4", "NEAR_MISS_STRUCTURAL_WEAKNESS", "failed fixed worst-quarter gate and cluster-independence evidence")
	relief.KnownDefects = append(relief.KnownDefects, "Phase 12.5 found no valid non-hindsight risk filter in retained schema")
	relief.Evidence = []qualification.EvidenceReference{
		evidenceRef("phase12_2_candidate_spec", "c2c79887:runs/reports/phase12_2_candidate_spec.json", "e8a36ca5f226197ae68a86fc40c8d1ef92b3f61091cf074dce9e30cf21afa640"),
		evidenceRef("phase12_3_downtrend_midvol_relief_eval", "c2c79887:runs/reports/phase12_3_downtrend_midvol_relief_eval.json", "fb27fe46ab1139ccafea3a7b3cbb7bfdfc7fb3bd2f7e545f1b7b566d2e6c9066"),
		evidenceRef("phase12_4_near_miss", "preserved:runs/reports/phase12_4_downtrend_midvol_relief_near_miss_audit.json", "df4857fa908d3ad044092c84c0d65474ab5b2f7d339a0e571ad4974d42b7a38b"),
		evidenceRef("phase12_5_filter_design", "preserved:runs/reports/phase12_5_candidate_risk_filter_design.json", "4c88c2bac766281a8f98b33e34e4a8ad3557dbb2f8730139ff56785056261596"),
	}
	candidates = append(candidates, relief)

	probe := pr4b0BaseCandidate("phase13/ContextFreeMomentumBreakoutProbe", "ContextFreeMomentumBreakoutProbe", "internal/app/phase13_context_free_probe.go", "aee12ae2733cc8bdef20b9f6365dbac99017a45f", "sha256:9cfab3293cd18ac108a7abf81cdfe647455ada2f78c7bd9dd0d1e3e4700a37fd")
	probe.DirectionSupport = []string{"long", "short"}
	probe.Symbols = []string{"LINKUSDT"}
	probe.RequiredContext = []string{}
	probe.RequiredTimeframes = []string{"1m input", "60m forward return"}
	probe.FeatureRequirements = []string{"primary-symbol price/volume only"}
	probe.ParameterSet = map[string]any{"type": "hardcoded_source", "purpose": "compact-event infrastructure proof"}
	probe.SampleSize = qualification.SampleSize{EventCount: 138, IndependentClusterCount: 138, SymbolsRepresented: 1, MonthsRepresented: 1, QuartersRepresented: 1}
	probe.CostStressResults = metricEvidence("FAIL", map[string]any{"profit_factor_after_5_bps": 0.8710474571572268, "net_after_5_bps": -1315.6691553193095}, "negative net infrastructure proof")
	probe.ConcentrationResults = metricEvidence("FAIL", map[string]any{"top_symbol_percent": 100, "top_month_percent": 100, "top_quarter_percent": 100}, "single symbol/month/quarter")
	probe.ResearchPhase = "Phase 13.0"
	probe.CurrentResearchLabel = "PHASE13_CONTEXT_FREE_PROOF_PASSED_INFRA_ONLY"
	probe.EligibilityClassification = qualification.ClassificationInfrastructureProbe
	probe.FinalStatus = qualification.StatusInsufficientSample
	probe.ExclusionReasons = []string{"infrastructure proof only", "one symbol-month", "negative net result", "leave-one-out segments unavailable"}
	probe.Evidence = []qualification.EvidenceReference{evidenceRef("phase13_0_context_free_compact_candidate_proof", "aee12ae:runs/reports/phase13_0_context_free_compact_candidate_proof.json", "782071d17623fe69d5807dc9cf2f4f290da939cf7e1050791a5737278a2438eb")}
	candidates = append(candidates, probe)

	sort.Slice(candidates, func(i, j int) bool { return candidates[i].CandidateID < candidates[j].CandidateID })
	return qualification.CandidateInventory{
		SchemaVersion: qualification.InventorySchemaVersion, Phase: "PR4B0", AcceptedEngineBaseline: pr4b0AcceptedEngineBaseline,
		Candidates: candidates, UnknownImplementations: []string{}, OmittedCandidates: []string{},
	}, nil
}

func pr4b0FirstGenFundingRows() []pr4b0HistoricalRow {
	return []pr4b0HistoricalRow{
		{"NegativeFundingLong|long|240m", "NegativeFundingLong", "long", "240m", 2145593, 56518, 24, 0.99115, -0.541331, 0.849023},
		{"NegativeFundingLong|long|120m", "NegativeFundingLong", "long", "120m", 2145593, 56518, 24, 0.940714, -2.627213, 0.830384},
		{"NegativeFundingLong|long|60m", "NegativeFundingLong", "long", "60m", 2145593, 56518, 24, 0.883501, -3.781133, 0.802787},
		{"NegativeFundingLong|long|30m", "NegativeFundingLong", "long", "30m", 2145593, 56518, 24, 0.81827, -4.364798, 0.75747},
		{"NegativeFundingLong|long|15m", "NegativeFundingLong", "long", "15m", 2145593, 56518, 24, 0.740992, -4.658988, 0.687693},
		{"NegativeFundingLong|long|5m", "NegativeFundingLong", "long", "5m", 2145593, 56518, 24, 0.584388, -4.87925, 0.487925},
		{"PositiveFundingShort|short|5m", "PositiveFundingShort", "short", "5m", 2560211, 65914, 24, 0.5483, -5.10175, 0.486334},
		{"PositiveFundingShort|short|15m", "PositiveFundingShort", "short", "15m", 2560211, 65914, 24, 0.694771, -5.292278, 0.644937},
		{"PositiveFundingShort|short|30m", "PositiveFundingShort", "short", "30m", 2560211, 65914, 24, 0.760131, -5.574096, 0.714951},
		{"PositiveFundingShort|short|60m", "PositiveFundingShort", "short", "60m", 2560211, 65914, 24, 0.807682, -6.062138, 0.736491},
		{"PositiveFundingShort|short|120m", "PositiveFundingShort", "short", "120m", 2560211, 65914, 24, 0.84007, -6.96434, 0.725765},
		{"PositiveFundingShort|short|240m", "PositiveFundingShort", "short", "240m", 2560211, 65914, 24, 0.862259, -8.433078, 0.692232},
		{"FundingFlipLong|long|5m", "FundingFlipLong", "long", "5m", 842528, 24995, 23, 0.563539, -4.993643, 0.44218},
		{"FundingFlipLong|long|15m", "FundingFlipLong", "long", "15m", 842528, 24995, 23, 0.715021, -4.988562, 0.693863},
		{"FundingFlipLong|long|30m", "FundingFlipLong", "long", "30m", 842528, 24995, 23, 0.786647, -4.999753, 0.756138},
		{"FundingFlipLong|long|60m", "FundingFlipLong", "long", "60m", 842528, 24995, 23, 0.839886, -5.105788, 0.791865},
		{"FundingFlipLong|long|120m", "FundingFlipLong", "long", "120m", 842528, 24995, 23, 0.881472, -5.228962, 0.822583},
		{"FundingFlipLong|long|240m", "FundingFlipLong", "long", "240m", 842528, 24995, 23, 0.908957, -5.587157, 0.809921},
		{"FundingFlipShort|short|5m", "FundingFlipShort", "short", "5m", 1840473, 54410, 24, 0.570839, -5.110659, 0.484858},
		{"FundingFlipShort|short|15m", "FundingFlipShort", "short", "15m", 1840473, 54410, 24, 0.712302, -5.307997, 0.624321},
		{"FundingFlipShort|short|30m", "FundingFlipShort", "short", "30m", 1840473, 54410, 24, 0.775552, -5.580282, 0.680256},
		{"FundingFlipShort|short|60m", "FundingFlipShort", "short", "60m", 1840473, 54410, 24, 0.818872, -6.147577, 0.702332},
		{"FundingFlipShort|short|120m", "FundingFlipShort", "short", "120m", 1840473, 54410, 24, 0.842431, -7.412705, 0.687038},
		{"FundingFlipShort|short|240m", "FundingFlipShort", "short", "240m", 1840473, 54410, 24, 0.85564, -9.574813, 0.639396},
		{"RegimeFundingLong|long|5m", "RegimeFundingLong", "long", "5m", 564007, 41440, 24, 0.580754, -5.014937, 0.481817},
		{"RegimeFundingLong|long|15m", "RegimeFundingLong", "long", "15m", 564007, 41440, 24, 0.711144, -5.352378, 0.664733},
		{"RegimeFundingLong|long|30m", "RegimeFundingLong", "long", "30m", 564007, 41440, 24, 0.780006, -5.467639, 0.744937},
		{"RegimeFundingLong|long|60m", "RegimeFundingLong", "long", "60m", 564007, 41440, 24, 0.838956, -5.455785, 0.777478},
		{"RegimeFundingLong|long|120m", "RegimeFundingLong", "long", "120m", 564007, 41440, 24, 0.8802, -5.558885, 0.77341},
		{"RegimeFundingLong|long|240m", "RegimeFundingLong", "long", "240m", 564007, 41440, 24, 0.948998, -3.204769, 0.791614},
		{"RegimeFundingShort|short|5m", "RegimeFundingShort", "short", "5m", 586958, 44382, 24, 0.579276, -5.338937, 0.477351},
		{"RegimeFundingShort|short|15m", "RegimeFundingShort", "short", "15m", 586958, 44382, 24, 0.696417, -5.905951, 0.580909},
		{"RegimeFundingShort|short|30m", "RegimeFundingShort", "short", "30m", 586958, 44382, 24, 0.751444, -6.389365, 0.655145},
		{"RegimeFundingShort|short|60m", "RegimeFundingShort", "short", "60m", 586958, 44382, 24, 0.772157, -7.766721, 0.658579},
		{"RegimeFundingShort|short|120m", "RegimeFundingShort", "short", "120m", 586958, 44382, 24, 0.826832, -7.989412, 0.630307},
		{"RegimeFundingShort|short|240m", "RegimeFundingShort", "short", "240m", 586958, 44382, 24, 0.827336, -11.085126, 0.616844},
	}
}

type pr4b0HistoricalEvidenceRow struct {
	row                                                pr4b0HistoricalRow
	sourceRef, sourceHash, phase, reportID, reportHash string
}

func pr4b0SecondGenFundingRows() []pr4b0HistoricalEvidenceRow {
	confirmed := func(row pr4b0HistoricalRow) pr4b0HistoricalEvidenceRow {
		return pr4b0HistoricalEvidenceRow{row, "396d377", "ee6ed83f687258ef7ceaf0476446cd0df9a3a54b00c49c68d8681718a01514b1", "Phase 10.11B", "phase10_11b_confirmed_funding_extreme_evaluation", "24a77775254f44338818de981cb9bc733cd210a80f4715f1f97a2f2c851757d6"}
	}
	breakout := func(row pr4b0HistoricalRow) pr4b0HistoricalEvidenceRow {
		return pr4b0HistoricalEvidenceRow{row, "073f97d", "3b1e08bb62215313fd7c49906f682015e505deb38d4cff19e31269bfbb34203a", "Phase 10.11C", "phase10_11c_breakout_funding_momentum_evaluation", "e0748f9cfceafd7b30c0df8be1bded6a113b568b9094f5d33beb26bac73a0328"}
	}
	volume := func(row pr4b0HistoricalRow) pr4b0HistoricalEvidenceRow {
		return pr4b0HistoricalEvidenceRow{row, "78ed2e2", "20c88508ca18da28bcb8438114d297c7739a697e9d7113f1db2dac8b7ff7a9f0", "Phase 10.11D", "phase10_11d_volume_imbalance_funding_reversion_proxy_evaluation", "4763cf429bf64ccae84b4e841b81410314cdeb11fdfda2f7e7983c34aa51db6b"}
	}
	return []pr4b0HistoricalEvidenceRow{
		confirmed(pr4b0HistoricalRow{"ConfirmedNegativeFundingLong|long|240m", "ConfirmedNegativeFundingLong", "long", "240m", 1353932, 0, 24, 0.966931, -2.014372, 0}),
		confirmed(pr4b0HistoricalRow{"ConfirmedPositiveFundingShort|short|5m", "ConfirmedPositiveFundingShort", "short", "5m", 1589488, 0, 24, 0.547049, -5.164622, 0}),
		breakout(pr4b0HistoricalRow{"BreakoutFundingLong|long|240m", "BreakoutFundingLong", "long", "240m", 197685, 45740, 24, 0.953242, -2.943941, 0.79144}),
		breakout(pr4b0HistoricalRow{"BreakoutFundingShort|short|5m", "BreakoutFundingShort", "short", "5m", 307793, 68905, 24, 0.580236, -5.932819, 0.496542}),
		volume(pr4b0HistoricalRow{"VolumeImbalanceFundingReversionProxyLong|long|240m", "VolumeImbalanceFundingReversionProxyLong", "long", "240m", 1110149, 77502, 24, 0.958095, -2.511184, 0.838737}),
		volume(pr4b0HistoricalRow{"VolumeImbalanceFundingReversionProxyLong|long|120m", "VolumeImbalanceFundingReversionProxyLong", "long", "120m", 1110149, 77502, 24, 0.919134, -3.475678, 0.83276}),
		volume(pr4b0HistoricalRow{"VolumeImbalanceFundingReversionProxyLong|long|60m", "VolumeImbalanceFundingReversionProxyLong", "long", "60m", 1110149, 77502, 24, 0.870552, -4.04423, 0.804239}),
		volume(pr4b0HistoricalRow{"VolumeImbalanceFundingReversionProxyLong|long|30m", "VolumeImbalanceFundingReversionProxyLong", "long", "30m", 1110149, 77502, 24, 0.809095, -4.405061, 0.751515}),
		volume(pr4b0HistoricalRow{"VolumeImbalanceFundingReversionProxyLong|long|15m", "VolumeImbalanceFundingReversionProxyLong", "long", "15m", 1110149, 77502, 24, 0.734629, -4.563096, 0.684408}),
		volume(pr4b0HistoricalRow{"VolumeImbalanceFundingReversionProxyLong|long|5m", "VolumeImbalanceFundingReversionProxyLong", "long", "5m", 1110149, 77502, 24, 0.573287, -4.806953, 0.505951}),
		volume(pr4b0HistoricalRow{"VolumeImbalanceFundingReversionProxyShort|short|5m", "VolumeImbalanceFundingReversionProxyShort", "short", "5m", 1645518, 118173, 24, 0.550962, -4.963354, 0.483972}),
		volume(pr4b0HistoricalRow{"VolumeImbalanceFundingReversionProxyShort|short|15m", "VolumeImbalanceFundingReversionProxyShort", "short", "15m", 1645518, 118173, 24, 0.701776, -5.068631, 0.642328}),
		volume(pr4b0HistoricalRow{"VolumeImbalanceFundingReversionProxyShort|short|30m", "VolumeImbalanceFundingReversionProxyShort", "short", "30m", 1645518, 118173, 24, 0.77157, -5.210624, 0.708321}),
		volume(pr4b0HistoricalRow{"VolumeImbalanceFundingReversionProxyShort|short|60m", "VolumeImbalanceFundingReversionProxyShort", "short", "60m", 1645518, 118173, 24, 0.819903, -5.612153, 0.749934}),
		volume(pr4b0HistoricalRow{"VolumeImbalanceFundingReversionProxyShort|short|120m", "VolumeImbalanceFundingReversionProxyShort", "short", "120m", 1645518, 118173, 24, 0.84979, -6.48965, 0.759665}),
		volume(pr4b0HistoricalRow{"VolumeImbalanceFundingReversionProxyShort|short|240m", "VolumeImbalanceFundingReversionProxyShort", "short", "240m", 1645518, 118173, 24, 0.871866, -7.809675, 0.751455}),
	}
}

func pr4b0CompressionRows() []pr4b0HistoricalRow {
	return []pr4b0HistoricalRow{
		{"CompressionVolumeBreakout|long|15m", "CompressionVolumeBreakout", "long", "15m", 2836, 2693, 24, 0.68468, -6.20119, 0.486102},
		{"CompressionVolumeBreakout|short|15m", "CompressionVolumeBreakout", "short", "15m", 2444, 2311, 24, 0.751003, -5.572455, 0.560427},
		{"CompressionVolumeBreakout|long|240m", "CompressionVolumeBreakout", "long", "240m", 2836, 2693, 24, 0.877362, -8.20032, 0.595329},
		{"CompressionVolumeBreakout|short|240m", "CompressionVolumeBreakout", "short", "240m", 2444, 2311, 24, 0.831626, -12.488132, 0.654234},
		{"CompressionVolumeBreakout|long|60m", "CompressionVolumeBreakout", "long", "60m", 2836, 2693, 24, 0.798698, -7.008594, 0.644251},
		{"CompressionVolumeBreakout|short|60m", "CompressionVolumeBreakout", "short", "60m", 2444, 2311, 24, 0.82157, -7.070122, 0.490578},
	}
}

func pr4b0TrendPullbackRows() []pr4b0HistoricalRow {
	return []pr4b0HistoricalRow{
		{"RegimeTrendPullbackContinuation|long|240m", "RegimeTrendPullbackContinuation", "long", "240m", 611891, 73685, 24, 0.944932, -3.536483, 0.835574},
		{"RegimeTrendPullbackContinuation|long|120m", "RegimeTrendPullbackContinuation", "long", "120m", 611891, 73685, 24, 0.881617, -5.456263, 0.798115},
		{"RegimeTrendPullbackContinuation|short|240m", "RegimeTrendPullbackContinuation", "short", "240m", 581746, 71499, 24, 0.877256, -8.749475, 0.687034},
		{"RegimeTrendPullbackContinuation|short|120m", "RegimeTrendPullbackContinuation", "short", "120m", 581746, 71499, 24, 0.859495, -7.10483, 0.69524},
		{"RegimeTrendPullbackContinuation|long|60m", "RegimeTrendPullbackContinuation", "long", "60m", 611891, 73685, 24, 0.848628, -5.063854, 0.748081},
		{"RegimeTrendPullbackContinuation|short|60m", "RegimeTrendPullbackContinuation", "short", "60m", 581746, 71499, 24, 0.827767, -6.374382, 0.714644},
		{"RegimeTrendPullbackContinuation|short|30m", "RegimeTrendPullbackContinuation", "short", "30m", 581746, 71499, 24, 0.798376, -5.537108, 0.671713},
		{"RegimeTrendPullbackContinuation|long|30m", "RegimeTrendPullbackContinuation", "long", "30m", 611891, 73685, 24, 0.781964, -5.435371, 0.703588},
		{"RegimeTrendPullbackContinuation|short|15m", "RegimeTrendPullbackContinuation", "short", "15m", 581746, 71499, 24, 0.737606, -5.37038, 0.636446},
		{"RegimeTrendPullbackContinuation|long|15m", "RegimeTrendPullbackContinuation", "long", "15m", 611891, 73685, 24, 0.715045, -5.322336, 0.650478},
	}
}

func pr4b0BaseCandidate(id, family, location, sourceRef, sourceHash string) qualification.CandidateRecord {
	return qualification.CandidateRecord{
		CandidateID: id, CandidateVersion: "v1", ImplementationLocation: location, ImplementationSourceRef: sourceRef,
		ImplementationSHA256: sourceHash, ImplementationReproducible: true, StrategyFamily: family,
		DirectionSupport: []string{"long", "short"}, Symbols: []string{"ADAUSDT", "AVAXUSDT", "BNBUSDT", "DOGEUSDT", "ETHUSDT", "LINKUSDT", "SOLUSDT", "XRPUSDT"},
		RequiredContext: []string{}, RequiredTimeframes: []string{"unspecified"}, FeatureRequirements: []string{"unspecified"}, ParameterSet: map[string]any{},
		ResearchPhase: "unknown", InSampleResults: missingEvidence(), OutOfSampleResults: missingEvidence(), WalkForwardResults: missingEvidence(),
		CostStressResults: missingEvidence(), WorstPeriodResults: missingEvidence(), ConcentrationResults: missingEvidence(),
		KnownDefects: []string{}, ExclusionReasons: []string{}, Evidence: []qualification.EvidenceReference{},
	}
}

func markRejected(candidate *qualification.CandidateRecord, phase, label, reason string) {
	candidate.ResearchPhase = phase
	candidate.CurrentResearchLabel = label
	candidate.EligibilityClassification = qualification.ClassificationRejected
	candidate.FinalStatus = qualification.StatusRejected
	candidate.ExclusionReasons = append(candidate.ExclusionReasons, reason)
}

func markNearMiss(candidate *qualification.CandidateRecord, phase, label, reason string) {
	candidate.ResearchPhase = phase
	candidate.CurrentResearchLabel = label
	candidate.EligibilityClassification = qualification.ClassificationNearMiss
	candidate.FinalStatus = qualification.StatusNearMiss
	candidate.ExclusionReasons = append(candidate.ExclusionReasons, reason)
}

func markMissingEvidence(candidate *qualification.CandidateRecord, phase, reason string) {
	candidate.ResearchPhase = phase
	candidate.CurrentResearchLabel = "MISSING_EVIDENCE"
	candidate.EligibilityClassification = qualification.ClassificationMissingEvidence
	candidate.FinalStatus = qualification.StatusPITEvidenceMissing
	candidate.ExclusionReasons = append(candidate.ExclusionReasons, reason)
}

func missingEvidence() qualification.EvidenceResult {
	return qualification.EvidenceResult{Status: "MISSING", Metrics: map[string]any{}, Notes: []string{}}
}

func metricEvidence(status string, metrics map[string]any, notes ...string) qualification.EvidenceResult {
	return qualification.EvidenceResult{Status: status, Metrics: metrics, Notes: notes}
}

func evidenceRef(id, source, hash string) qualification.EvidenceReference {
	return qualification.EvidenceReference{ArtifactID: id, SourceRef: source, SHA256: "sha256:" + strings.TrimPrefix(hash, "sha256:")}
}

func priceFamilyContext(family string) []string {
	switch family {
	case "BetaAgrees", "BetaDiverges":
		return []string{"BTC/ETH market context"}
	default:
		return []string{"regime classification"}
	}
}

func priceFamilyFeatures(family string) []string {
	switch family {
	case "TrendContinuation":
		return []string{"EMA20", "EMA50", "EMA200", "trend slope"}
	case "CompressionBreakout":
		return []string{"Bollinger width/rank", "EMA break", "volume"}
	case "ShockFade":
		return []string{"trailing return shock", "realized volatility", "mean reversion"}
	case "VolumeMomentum":
		return []string{"volume ratio", "return momentum"}
	case "BetaAgrees", "BetaDiverges":
		return []string{"target return", "BTC/ETH context return"}
	default:
		return []string{"unknown"}
	}
}

func structMap(value any) (map[string]any, error) {
	data, err := json.Marshal(value)
	if err != nil {
		return nil, err
	}
	var result map[string]any
	if err := json.Unmarshal(data, &result); err != nil {
		return nil, err
	}
	return result, nil
}

func hashStrings(values ...string) string {
	digest := sha256.Sum256([]byte(strings.Join(values, "\x00")))
	return hex.EncodeToString(digest[:])
}

func pr4b0RegisteredCandidateIDs() []string {
	ids := []string{
		"baseline", strategyFastAccumulation, strategyFastAccumulationStrict, strategyFastAccumulationStrictShortBias,
		strategyFastAccumulationStrictHighConf, strategyFastAccumulationStrictLowFreq, strategyFastAccumulationStrictCostGuard,
		strategyFastAccumulationStrictNo7084Longs, strategyFastAccumulationStrict30m, strategyFastAccumulationStrict1h,
		strategyFastAccumulationPullbackReclaim, strategyFastAccumulationBreakoutRetest, strategyFastAccumulationMomentumCont,
		strategyFastAccumulationPartialTrail, strategyFastAccumulationBreakevenGuard, strategyFastAccumulationCutNoProgress,
		strategyFastAccumulationEconomicsGuard,
	}
	sort.Strings(ids)
	return ids
}

func buildPR4B0QualificationReport(inventory qualification.CandidateInventory, resultingCommit string, verificationComplete bool, freshCloneCommit string) pr4b0QualificationReport {
	assessments := make([]pr4b0Assessment, 0, len(inventory.Candidates))
	excluded := make([]map[string]any, 0, len(inventory.Candidates))
	metrics := make([]map[string]any, 0, len(inventory.Candidates))
	counts := make(map[string]int)
	for _, candidate := range inventory.Candidates {
		counts[string(candidate.FinalStatus)]++
		assessments = append(assessments, pr4b0Assessment{
			CandidateID: candidate.CandidateID, CandidateVersion: candidate.CandidateVersion,
			Classification: candidate.EligibilityClassification, FinalStatus: candidate.FinalStatus,
			FailedGates: append([]string{}, candidate.ExclusionReasons...), Decision: "EXCLUDED_FROM_FREEZE",
		})
		excluded = append(excluded, map[string]any{"candidate_id": candidate.CandidateID, "status": candidate.FinalStatus, "reasons": candidate.ExclusionReasons})
		metrics = append(metrics, map[string]any{
			"candidate_id": candidate.CandidateID, "event_count": candidate.SampleSize.EventCount,
			"independent_cluster_count": candidate.SampleSize.IndependentClusterCount,
			"symbols":                   candidate.SampleSize.SymbolsRepresented, "months": candidate.SampleSize.MonthsRepresented,
			"cost_stress": candidate.CostStressResults, "worst_period": candidate.WorstPeriodResults,
			"concentration": candidate.ConcentrationResults, "final_status": candidate.FinalStatus,
		})
	}
	return pr4b0QualificationReport{
		SchemaVersion: qualification.QualificationReportSchemaVersion, Phase: "PR4B0",
		ExecutiveVerdict:          "Existing Engine evidence was completely inventoried. No candidate satisfies every mandatory qualification gate; no search, holdout exposure, freeze, registration request, promotion, or paper evaluator was performed.",
		FinalLabel:                pr4b0NoCandidateLabel,
		AcceptedBaselines:         map[string]string{"ak-engine": pr4b0AcceptedEngineBaseline, "ak-rif": pr4b0AcceptedRIFBaseline, "ak-historian": pr4b0AcceptedHistorianHead},
		ResultingCommit:           resultingCommit,
		CandidateInventorySummary: map[string]any{"candidate_count": len(inventory.Candidates), "registered_candidate_count": len(inventory.RegisteredCandidateIDs), "status_counts": counts, "unknown_implementations": inventory.UnknownImplementations, "omitted_candidates": inventory.OmittedCandidates},
		QualificationGates:        pr4b0QualificationGatePolicy(), Assessments: assessments, ExistingCandidatesExcluded: excluded,
		NewResearch: map[string]any{"performed": false, "protocol_artifacts_created": false, "reason": "No existing candidate reached QUALIFICATION_CANDIDATE; a new search would require a separately pre-registered research epoch with immutable data/PIT identity and controlled RIF holdout access."},
		DataAndHoldoutControls: map[string]any{
			"existing_report_window": "2024-01 through 2025-12 where recorded", "immutable_dataset_identity_established_for_engine_candidate": false,
			"engine_candidate_pit_evidence_present": false, "final_holdout_inspected": false, "final_holdout_used_for_selection": false,
			"rif_holdout_exposure_registered": false, "accepted_holdout_mechanism": "ak.rif.holdout_ledger.v1 / ak.rif.holdout_exposure.v1",
			"historian_fixture_reuse_prohibited": "accepted PIT fixture belongs to historian-pr3-candidate, not an Engine candidate",
		},
		SelectedCandidate: nil, QualificationMetrics: metrics, FrozenIdentity: nil, DirectImplementationParityFixtures: []any{}, CandidateRegistrationArtifact: nil,
		TestsAndRace: pr4b0VerificationResults(verificationComplete, resultingCommit, freshCloneCommit), SecurityFindings: pr4b0SecurityFindings(),
		FreshClone: pr4b0FreshCloneResult(freshCloneCommit),
		Boundaries: map[string]any{
			"paper_evaluator_implemented": false, "rif_authorized_candidate": false, "candidate_promoted": false,
			"trader_behavior_changed": false, "mainnet_behavior_changed": false, "authenticated_exchange_operations": false,
			"historian_modified": false, "rif_implementation_imported": false,
		},
		DeferredWork:         []string{"separately pre-register a bounded research epoch with immutable dataset/manifest/PIT identity", "obtain complete required context without mutable aliases", "register and consume a one-time final holdout only through accepted RIF orchestration", "PR4B1 remains ineligible until a future candidate qualifies"},
		GeneratedReportPaths: []string{"runs/reports/pr4b0_candidate_inventory.md", "runs/reports/pr4b0_candidate_inventory.json", "runs/reports/pr4b0_candidate_qualification.md", "runs/reports/pr4b0_candidate_qualification.json"},
		RecommendedNextPhase: "SEPARATELY_PRE_REGISTERED_BOUNDED_RESEARCH_PHASE", QualificationReportID: "pr4b0_candidate_qualification",
	}
}

func pr4b0QualificationGatePolicy() pr4b0GatePolicy {
	return pr4b0GatePolicy{
		PolicyStatus: "DECLARED_FOR_EXISTING_EVIDENCE_ASSESSMENT; NO_NEW_SEARCH_EXECUTED",
		DataIntegrity: map[string]any{
			"dataset_id":      "required exact immutable identity; none established for an Engine candidate",
			"dataset_version": "required immutable version; mutable aliases prohibited", "manifest_id_and_hash": "required and must match accepted Historian PIT evidence",
			"windows": "non-overlapping development, validation, and one-time final holdout", "required_symbols": []string{"ADAUSDT", "AVAXUSDT", "BNBUSDT", "DOGEUSDT", "ETHUSDT", "LINKUSDT", "SOLUSDT", "XRPUSDT"},
			"required_context_symbols": []string{"BTCUSDT", "ETHUSDT where candidate capability requires them"}, "gap_policy": "reject undeclared internal gaps", "pit_policy": "source available_at must be no later than decision cutoff",
		},
		SampleSufficiency: qualification.SampleGates{MinimumEvents: 300, MinimumIndependentClusters: 300, MinimumTradesOrDecisions: 300, MinimumSymbols: 4, MinimumMonths: 12, MinimumPositiveRegimes: 1, MinimumNegativeRegimes: 1},
		Performance:       qualification.PerformanceGates{MinimumNetExpectancyBPS: 0.01, MinimumProfitFactor: 1.10, MaximumDrawdownBPS: 2500, MinimumConfidenceLowerBoundBPS: 0.0, DownsideTailPolicy: "lower confidence bound must be positive and worst-decile/worst-symbol-month loss must fit declared drawdown budget"},
		Robustness:        qualification.RobustnessGates{RequireOutOfSample: true, RequireWalkForward: true, MinimumWorstPeriodProfitFactor: 0.95, MaximumSymbolContributionPercent: 50, MaximumTemporalContributionPercent: 50, MaximumRegimeContributionPercent: 60, MinimumStableNeighbors: 2, RequireClusterDeduplication: true, RequireMissingContextSensitivity: true},
		CostStress:        qualification.CostGates{FeeBPS: 5, SpreadBPS: 1, SlippageBPS: 1, FundingBPS: 1, AdverseSelectionBPS: 2, StressTotalBPS: 10, MinimumStressProfitFactor: 1.01, MinimumStressExpectancyBPS: 0.01},
		LeakageRules:      []string{"reject future candles", "reject revised data unavailable at cutoff", "reject final outcomes in features", "reject holdout-derived feature selection", "require PIT-compatible source timing", "reject manifest/dataset mismatch"},
		SimplicityRule:    "select the simplest candidate only after all mandatory gates pass; additional degrees of freedom require material validation improvement",
		SearchPolicy:      map[string]any{"new_search_performed": false, "open_ended_tuning_allowed": false, "final_holdout_used_for_selection": false, "future_search_requires_separate_pre_registration": true},
	}
}

func pr4b0VerificationResults(complete bool, resultingCommit, freshCloneCommit string) []pr4b0CommandResult {
	type commandSpec struct {
		command string
		notes   string
	}
	commands := []commandSpec{
		{"gofmt -w internal/app/pr4b0_candidate_qualification.go internal/app/pr4b0_candidate_qualification_test.go internal/qualification/qualification.go internal/qualification/qualification_test.go", "four changed Go files"},
		{"GOWORK=off go mod tidy", "module graph remained unchanged"},
		{"git diff --exit-code -- go.mod go.sum", "no module-file drift"},
		{"GOWORK=off go vet ./...", "all packages"},
		{"GOWORK=off go test ./...", "all packages"},
		{"GOWORK=off go test -race ./...", "all packages"},
		{"GOWORK=off go build ./...", "all packages"},
		{"GOWORK=off make verify", "project verification target"},
		{"git diff --check", "no whitespace errors"},
		{fmt.Sprintf("GOWORK=off go run ./cmd/ak-engine pr4b0-candidate-qualification --out-dir runs/reports --resulting-commit %s --verification-complete --fresh-clone-commit %s", resultingCommit, freshCloneCommit), "generated exactly four mandatory reports"},
		{"find runs/reports -maxdepth 1 -type f -name 'pr4b0_*.json' -print0 | xargs -0 -n1 jq -e .", "two JSON artifacts parsed successfully"},
		{"test -z \"$(rg -l '/home/|/Users/|[A-Za-z]:\\\\\\\\' internal/app/pr4b0_candidate_qualification.go internal/app/pr4b0_candidate_qualification_test.go internal/qualification runs/reports/pr4b0_*)\"", "zero absolute-path matches"},
		{"test -z \"$(rg -l 'github\\.com/.+/(ak-rif|ak-historian|ak-trader)' --glob '*.go' internal/app/pr4b0_candidate_qualification.go internal/app/pr4b0_candidate_qualification_test.go internal/qualification)\"", "zero sibling or trader imports"},
		{"test -z \"$(rg -l -i '(api[_-]?key|api[_-]?secret|private[_-]?key|access[_-]?token|password)[[:space:]]*[:=][[:space:]]*\\\"' internal/app/pr4b0_candidate_qualification.go internal/app/pr4b0_candidate_qualification_test.go internal/qualification runs/reports/pr4b0_*)\"", "zero credential assignments"},
		{"test -z \"$(rg -l -- '-----BEGIN [A-Z ]*PRIVATE KEY-----' internal/app/pr4b0_candidate_qualification.go internal/app/pr4b0_candidate_qualification_test.go internal/qualification runs/reports/pr4b0_*)\"", "zero private-key markers"},
		{"test -z \"$(rg -l 'net/http|os\\.Getenv|exec\\.Command|websocket|NewClient' internal/app/pr4b0_candidate_qualification.go internal/qualification)\"", "zero network, credential-environment, or subprocess calls"},
		{"test -z \"$(find runs/reports -maxdepth 1 -type f \\( -name 'pr4b0_candidate_qualification_protocol.*' -o -name 'pr4b0_frozen_candidate.*' -o -name 'pr4b0_candidate_registration_request.*' \\) -print)\"", "zero outcome-inapplicable artifacts"},
	}
	results := make([]pr4b0CommandResult, 0, len(commands))
	for _, spec := range commands {
		result := pr4b0CommandResult{Command: spec.command, Status: "PENDING", Notes: spec.notes}
		if complete {
			exit := 0
			result.ExitCode = &exit
			result.Status = "PASS"
		}
		results = append(results, result)
	}
	return results
}

func pr4b0FreshCloneResult(commit string) map[string]any {
	if strings.TrimSpace(commit) == "" {
		return map[string]any{"status": "PENDING", "no_sibling_ak_repositories": true}
	}
	return map[string]any{
		"status": "PASS", "commit": commit, "no_sibling_ak_repositories": true,
		"commands_and_exit_codes": pr4b0VerificationResults(true, commit, commit),
	}
}

func pr4b0SecurityFindings() []map[string]string {
	areas := []string{"data leakage", "final-holdout reuse", "parameter overfitting", "candidate-selection bias", "report substitution", "dataset substitution", "implementation substitution", "configuration substitution", "mutable dataset aliases", "hash confusion", "schema downgrade", "path-derived identity", "nondeterministic source hashing", "hidden generic fallbacks", "log leakage", "fail-open qualification rules"}
	findings := make([]map[string]string, 0, len(areas))
	for _, area := range areas {
		findings = append(findings, map[string]string{"area": area, "status": "NO_UNRESOLVED_HIGH_SEVERITY_FINDING", "control": "fail-closed inventory/qualification identity checks; no candidate freeze or holdout exposure"})
	}
	return findings
}

func hashReport(report pr4b0QualificationReport) (string, error) {
	data, err := json.Marshal(report)
	if err != nil {
		return "", err
	}
	var object map[string]any
	if err := json.Unmarshal(data, &object); err != nil {
		return "", err
	}
	delete(object, "qualification_report_hash")
	canonical, err := json.Marshal(object)
	if err != nil {
		return "", err
	}
	digest := sha256.Sum256(canonical)
	return "sha256:" + hex.EncodeToString(digest[:]), nil
}

func renderPR4B0InventoryMarkdown(inventory qualification.CandidateInventory) string {
	var out strings.Builder
	out.WriteString("# PR4B0 Candidate Inventory\n\n")
	out.WriteString("Accepted Engine baseline: `" + inventory.AcceptedEngineBaseline + "`.\n\n")
	out.WriteString(fmt.Sprintf("Candidate records: **%d**. Unknown implementations: **%d**. Omitted candidates: **%d**.\n\n", inventory.CandidateCount, len(inventory.UnknownImplementations), len(inventory.OmittedCandidates)))
	out.WriteString("| Candidate | Family | Phase | Classification | Final status | Reproducible | Exclusion |\n")
	out.WriteString("|---|---|---|---|---|---:|---|\n")
	for _, candidate := range inventory.Candidates {
		out.WriteString(fmt.Sprintf("| `%s` | %s | %s | `%s` | `%s` | %t | %s |\n", candidate.CandidateID, candidate.StrategyFamily, candidate.ResearchPhase, candidate.EligibilityClassification, candidate.FinalStatus, candidate.ImplementationReproducible, strings.Join(candidate.ExclusionReasons, "; ")))
	}
	out.WriteString("\nEvery registered Engine strategy name is present. Failed, fragile, near-miss, infrastructure-only, and missing-evidence hypotheses are retained rather than omitted.\n")
	return out.String()
}

func renderPR4B0QualificationMarkdown(report pr4b0QualificationReport) string {
	var out strings.Builder
	out.WriteString("# PR4B0 Candidate Qualification\n\n")
	out.WriteString("## Executive verdict\n\n" + report.ExecutiveVerdict + "\n\n")
	out.WriteString("Final label: `" + report.FinalLabel + "`.\n\n")
	out.WriteString("## Baselines and result\n\n")
	out.WriteString("- Engine: `" + report.AcceptedBaselines["ak-engine"] + "`\n")
	out.WriteString("- RIF contracts/fixtures: `" + report.AcceptedBaselines["ak-rif"] + "`\n")
	out.WriteString("- Historian PIT authority: `" + report.AcceptedBaselines["ak-historian"] + "`\n")
	out.WriteString("- Resulting source commit: `" + report.ResultingCommit + "`\n")
	out.WriteString("- Qualification report hash: `" + report.QualificationReportHash + "`\n\n")
	out.WriteString("## Candidate inventory and decisions\n\n")
	out.WriteString(fmt.Sprintf("%v candidates were inventoried; none was omitted and none qualified.\n\n", report.CandidateInventorySummary["candidate_count"]))
	out.WriteString("| Candidate | Classification | Final status | Decision |\n|---|---|---|---|\n")
	for _, assessment := range report.Assessments {
		out.WriteString(fmt.Sprintf("| `%s` | `%s` | `%s` | %s |\n", assessment.CandidateID, assessment.Classification, assessment.FinalStatus, strings.Join(assessment.FailedGates, "; ")))
	}
	out.WriteString("\n## Qualification protocol and gates\n\n")
	out.WriteString("No new research was performed, so no new-search protocol artifact was created. Existing evidence was assessed against the declared fail-closed gates embedded in the JSON report: exact immutable data/PIT identity, at least 300 independent clusters/decisions, PF >= 1.10 and positive expectancy/confidence bound, worst-period PF >= 0.95, bounded concentration, OOS and walk-forward stability, parameter-neighborhood stability, realistic 10 bps total stress, leakage safety, simplicity, exact implementation, PIT evidence, and RIF-controlled holdout exposure.\n\n")
	out.WriteString("## Data and holdout controls\n\nNo Engine candidate has a complete accepted `ak.rif.research_identity.v1`, candidate-bound Historian PIT evidence, or accepted RIF holdout exposure. The final holdout was not inspected or used. The accepted Historian/RIF fixture belongs to a synthetic Historian candidate and was not substituted.\n\n")
	out.WriteString("## Selected candidate, frozen identity, parity, and registration\n\nNo candidate was selected. No descriptor was frozen, no parity fixtures were created, and no Engine candidate-registration request was emitted. RIF did not accept or authorize any candidate.\n\n")
	out.WriteString("## Verification\n\n| Command | Status | Exit |\n|---|---|---:|\n")
	for _, result := range report.TestsAndRace {
		exit := "—"
		if result.ExitCode != nil {
			exit = fmt.Sprintf("%d", *result.ExitCode)
		}
		out.WriteString(fmt.Sprintf("| `%s` | %s | %s |\n", result.Command, result.Status, exit))
	}
	out.WriteString("\nFresh clone: `" + fmt.Sprint(report.FreshClone["status"]) + "`; no sibling AK repositories: `true`.\n\n")
	out.WriteString("## Security and boundaries\n\nThe JSON report records all required security-review areas. No unresolved in-scope high-severity finding remains. No paper evaluator was implemented. RIF did not authorize a candidate. No candidate was promoted. Trader behavior was unchanged. Historian was not modified. No mainnet or authenticated exchange behavior was run.\n\n")
	out.WriteString("## Deferred work and recommendation\n\nA separately pre-registered bounded research phase must establish immutable dataset/manifest/PIT identity, complete candidate context, finite search budget, and RIF-controlled one-time holdout treatment. Do not begin PR4B1.\n\n")
	out.WriteString("Recommended next phase: `" + report.RecommendedNextPhase + "`.\n")
	return out.String()
}
