package researchidentity

import (
	"bytes"
	"fmt"
	"math"
	"sort"
	"strings"
	"time"

	"github.com/david22573/ak-engine/internal/canonicalcontract"
)

const (
	configurationEncoding = "ak.canonical.json.v1"
	featureSetID          = "ak.engine.features.deep-candidate"
	featureSetVersion     = "1"
	regimeDefinitionID    = "ak.engine.regime.default"
	regimeDefinitionVer   = "1"
	missingValuePolicy    = "REJECT_NON_FINITE_AND_MISSING"
	filteringPolicy       = "FEATURE_WINDOW_WITH_CONFIGURED_MAX_HORIZON"
)

type ConfigurationContext struct {
	Symbol            string
	Market            string
	Interval          string
	EvaluationStartMS int64
	EvaluationEndMS   int64
	BuildTags         []string
}

type configurationOverrides struct {
	ForwardHorizonsMinutes    *[]int                  `json:"forward_horizons_minutes"`
	CostHaircutsBPS           *[]float64              `json:"cost_haircuts_bps"`
	CostHaircutHorizonMinutes *int                    `json:"cost_haircut_horizon_minutes"`
	EntryDelayCandles         *[]int                  `json:"entry_delay_candles"`
	EntryDelayHorizonMinutes  *int                    `json:"entry_delay_horizon_minutes"`
	SeriesHorizonMinutes      *int                    `json:"series_horizon_minutes"`
	SeriesCostBPS             *float64                `json:"series_cost_bps"`
	SeriesEntryDelayCandles   *int                    `json:"series_entry_delay_candles"`
	ExcursionWindowsMinutes   *[]int                  `json:"excursion_windows_minutes"`
	Brackets                  *[]BracketConfiguration `json:"brackets"`
	BracketWindowMinutes      *int                    `json:"bracket_window_minutes"`
	StabilityAggregatePeriods *[]string               `json:"stability_aggregate_periods"`
	StabilityHorizonMinutes   *int                    `json:"stability_horizon_minutes"`
	StabilityEntryDelay       *int                    `json:"stability_entry_delay_candles"`
	RegimeGroups              *[]string               `json:"regime_groups"`
	RegimeHorizonMinutes      *int                    `json:"regime_horizon_minutes"`
	RegimeLowSampleMinimum    *int                    `json:"regime_low_sample_minimum"`
	ClusterSeparationMinutes  *int                    `json:"cluster_separation_minutes"`
	DiagnosticMinimumSamples  *int                    `json:"diagnostic_minimum_samples"`
	ObservationsPerParameter  *float64                `json:"observations_per_parameter"`
	ModelParameterCount       *int                    `json:"model_parameter_count"`
	MetricRiskFreeRate        *float64                `json:"metric_risk_free_rate"`
	MetricPeriodsPerYear      *float64                `json:"metric_periods_per_year"`
	GateThresholds            *gateThresholdOverrides `json:"gate_thresholds"`
}

type gateThresholdOverrides struct {
	MinimumEvents                  *int     `json:"minimum_events"`
	MinimumH2PF                    *float64 `json:"minimum_h2_pf"`
	MinimumH2ExpectancyBPS         *float64 `json:"minimum_h2_expectancy_bps"`
	MinimumFYPF                    *float64 `json:"minimum_fy_pf"`
	MinimumFYExpectancyBPS         *float64 `json:"minimum_fy_expectancy_bps"`
	MinimumPositiveMonths          *int     `json:"minimum_positive_months"`
	MinimumDelayOneExpectancyBPS   *float64 `json:"minimum_delay_one_expectancy_bps"`
	MaximumSingleMonthContribution *float64 `json:"maximum_single_month_contribution_pct"`
	MinimumWorstQuarterPF          *float64 `json:"minimum_worst_quarter_pf"`
	MinimumBracketPF               *float64 `json:"minimum_bracket_pf"`
	MinimumBracketExpectancyBPS    *float64 `json:"minimum_bracket_expectancy_bps"`
}

func ResolveConfiguration(context ConfigurationContext, overrideJSON []byte) (ResolvedResearchConfiguration, error) {
	if strings.TrimSpace(context.Symbol) == "" || strings.TrimSpace(context.Market) == "" || strings.TrimSpace(context.Interval) == "" {
		return ResolvedResearchConfiguration{}, fmt.Errorf("symbol, market, and interval are required")
	}
	if context.EvaluationStartMS <= 0 || context.EvaluationEndMS <= context.EvaluationStartMS {
		return ResolvedResearchConfiguration{}, fmt.Errorf("invalid evaluation window")
	}
	config := defaultConfiguration(context)
	if len(bytes.TrimSpace(overrideJSON)) > 0 {
		var overrides configurationOverrides
		if err := decodeStrictJSON(overrideJSON, &overrides); err != nil {
			return ResolvedResearchConfiguration{}, fmt.Errorf("research configuration: %w", err)
		}
		applyConfigurationOverrides(&config, overrides)
	}
	if err := validateResolvedConfiguration(config); err != nil {
		return ResolvedResearchConfiguration{}, err
	}
	return config, nil
}

func defaultConfiguration(context ConfigurationContext) ResolvedResearchConfiguration {
	buildTags := append([]string(nil), context.BuildTags...)
	sort.Strings(buildTags)
	return ResolvedResearchConfiguration{
		EncodingVersion:           configurationEncoding,
		Symbol:                    strings.ToUpper(strings.TrimSpace(context.Symbol)),
		Market:                    strings.TrimSpace(context.Market),
		Interval:                  strings.TrimSpace(context.Interval),
		EvaluationStartMS:         context.EvaluationStartMS,
		EvaluationEndMS:           context.EvaluationEndMS,
		ForwardHorizonsMinutes:    []int{5, 15, 30, 60, 120, 240},
		CostHaircutsBPS:           []float64{0, 2, 5, 10, 15},
		CostHaircutHorizonMinutes: 60,
		EntryDelayCandles:         []int{0, 1, 3, 5, 10},
		EntryDelayHorizonMinutes:  60,
		EntryDelayCostBPS:         5,
		SeriesHorizonMinutes:      1440,
		SeriesCostBPS:             5,
		SeriesEntryDelayCandles:   0,
		ExcursionWindowsMinutes:   []int{30, 60, 120, 240},
		Brackets: []BracketConfiguration{
			{Name: "TP 5 bps / SL 5 bps", TPBPS: 5, SLBPS: 5},
			{Name: "TP 10 bps / SL 5 bps", TPBPS: 10, SLBPS: 5},
			{Name: "TP 15 bps / SL 7.5 bps", TPBPS: 15, SLBPS: 7.5},
			{Name: "TP 20 bps / SL 10 bps", TPBPS: 20, SLBPS: 10},
			{Name: "TP 30 bps / SL 15 bps", TPBPS: 30, SLBPS: 15},
			{Name: "TP 50 bps / SL 25 bps", TPBPS: 50, SLBPS: 25},
		},
		BracketWindowMinutes:      240,
		BracketCostBPS:            5,
		StabilityAggregatePeriods: []string{"Q1", "Q2", "Q3", "Q4", "H1", "H2", "FY"},
		StabilityHorizonMinutes:   60,
		StabilityCostBPS:          5,
		StabilityEntryDelay:       0,
		RegimeGroups:              []string{"composite", "volatility", "trend", "liquidity", "market_beta"},
		RegimeHorizonMinutes:      60,
		RegimeCostBPS:             5,
		RegimeLowSampleMinimum:    100,
		ClusterSeparationMinutes:  60,
		DiagnosticMinimumSamples:  30,
		ObservationsPerParameter:  20,
		ModelParameterCount:       0,
		MetricRiskFreeRate:        0.05,
		MetricPeriodsPerYear:      365,
		FeatureSetID:              featureSetID,
		FeatureSetVersion:         featureSetVersion,
		RegimeDefinitionID:        regimeDefinitionID,
		RegimeVersion:             regimeDefinitionVer,
		MissingValuePolicy:        missingValuePolicy,
		FilteringPolicy:           filteringPolicy,
		GateThresholds: GateThresholds{
			MinimumEvents:                  300,
			MinimumH2PF:                    1.10,
			MinimumH2ExpectancyBPS:         0,
			MinimumFYPF:                    1.05,
			MinimumFYExpectancyBPS:         0,
			MinimumPositiveMonths:          3,
			MinimumDelayOneExpectancyBPS:   0,
			MaximumSingleMonthContribution: 50,
			MinimumWorstQuarterPF:          0.95,
			MinimumBracketPF:               0.50,
			MinimumBracketExpectancyBPS:    -10,
		},
		BuildTags: buildTags,
	}
}

func applyConfigurationOverrides(config *ResolvedResearchConfiguration, o configurationOverrides) {
	if o.ForwardHorizonsMinutes != nil {
		config.ForwardHorizonsMinutes = append([]int(nil), (*o.ForwardHorizonsMinutes)...)
	}
	if o.CostHaircutsBPS != nil {
		config.CostHaircutsBPS = append([]float64(nil), (*o.CostHaircutsBPS)...)
	}
	if o.CostHaircutHorizonMinutes != nil {
		config.CostHaircutHorizonMinutes = *o.CostHaircutHorizonMinutes
	}
	if o.EntryDelayCandles != nil {
		config.EntryDelayCandles = append([]int(nil), (*o.EntryDelayCandles)...)
	}
	if o.EntryDelayHorizonMinutes != nil {
		config.EntryDelayHorizonMinutes = *o.EntryDelayHorizonMinutes
	}
	if o.SeriesHorizonMinutes != nil {
		config.SeriesHorizonMinutes = *o.SeriesHorizonMinutes
	}
	if o.SeriesCostBPS != nil {
		config.SeriesCostBPS = *o.SeriesCostBPS
	}
	if o.SeriesEntryDelayCandles != nil {
		config.SeriesEntryDelayCandles = *o.SeriesEntryDelayCandles
	}
	if o.ExcursionWindowsMinutes != nil {
		config.ExcursionWindowsMinutes = append([]int(nil), (*o.ExcursionWindowsMinutes)...)
	}
	if o.Brackets != nil {
		config.Brackets = append([]BracketConfiguration(nil), (*o.Brackets)...)
	}
	if o.BracketWindowMinutes != nil {
		config.BracketWindowMinutes = *o.BracketWindowMinutes
	}
	if o.StabilityAggregatePeriods != nil {
		config.StabilityAggregatePeriods = append([]string(nil), (*o.StabilityAggregatePeriods)...)
	}
	if o.StabilityHorizonMinutes != nil {
		config.StabilityHorizonMinutes = *o.StabilityHorizonMinutes
	}
	if o.StabilityEntryDelay != nil {
		config.StabilityEntryDelay = *o.StabilityEntryDelay
	}
	if o.RegimeGroups != nil {
		config.RegimeGroups = append([]string(nil), (*o.RegimeGroups)...)
	}
	if o.RegimeHorizonMinutes != nil {
		config.RegimeHorizonMinutes = *o.RegimeHorizonMinutes
	}
	if o.RegimeLowSampleMinimum != nil {
		config.RegimeLowSampleMinimum = *o.RegimeLowSampleMinimum
	}
	if o.ClusterSeparationMinutes != nil {
		config.ClusterSeparationMinutes = *o.ClusterSeparationMinutes
	}
	if o.DiagnosticMinimumSamples != nil {
		config.DiagnosticMinimumSamples = *o.DiagnosticMinimumSamples
	}
	if o.ObservationsPerParameter != nil {
		config.ObservationsPerParameter = *o.ObservationsPerParameter
	}
	if o.ModelParameterCount != nil {
		config.ModelParameterCount = *o.ModelParameterCount
	}
	if o.MetricRiskFreeRate != nil {
		config.MetricRiskFreeRate = *o.MetricRiskFreeRate
	}
	if o.MetricPeriodsPerYear != nil {
		config.MetricPeriodsPerYear = *o.MetricPeriodsPerYear
	}
	if o.GateThresholds != nil {
		applyGateThresholdOverrides(&config.GateThresholds, *o.GateThresholds)
	}
}

func applyGateThresholdOverrides(target *GateThresholds, override gateThresholdOverrides) {
	if override.MinimumEvents != nil {
		target.MinimumEvents = *override.MinimumEvents
	}
	if override.MinimumH2PF != nil {
		target.MinimumH2PF = *override.MinimumH2PF
	}
	if override.MinimumH2ExpectancyBPS != nil {
		target.MinimumH2ExpectancyBPS = *override.MinimumH2ExpectancyBPS
	}
	if override.MinimumFYPF != nil {
		target.MinimumFYPF = *override.MinimumFYPF
	}
	if override.MinimumFYExpectancyBPS != nil {
		target.MinimumFYExpectancyBPS = *override.MinimumFYExpectancyBPS
	}
	if override.MinimumPositiveMonths != nil {
		target.MinimumPositiveMonths = *override.MinimumPositiveMonths
	}
	if override.MinimumDelayOneExpectancyBPS != nil {
		target.MinimumDelayOneExpectancyBPS = *override.MinimumDelayOneExpectancyBPS
	}
	if override.MaximumSingleMonthContribution != nil {
		target.MaximumSingleMonthContribution = *override.MaximumSingleMonthContribution
	}
	if override.MinimumWorstQuarterPF != nil {
		target.MinimumWorstQuarterPF = *override.MinimumWorstQuarterPF
	}
	if override.MinimumBracketPF != nil {
		target.MinimumBracketPF = *override.MinimumBracketPF
	}
	if override.MinimumBracketExpectancyBPS != nil {
		target.MinimumBracketExpectancyBPS = *override.MinimumBracketExpectancyBPS
	}
}

func validateResolvedConfiguration(config ResolvedResearchConfiguration) error {
	if config.EncodingVersion != configurationEncoding {
		return fmt.Errorf("configuration encoding version mismatch")
	}
	if len(config.ForwardHorizonsMinutes) == 0 || len(config.CostHaircutsBPS) == 0 || len(config.EntryDelayCandles) == 0 || len(config.ExcursionWindowsMinutes) == 0 || len(config.Brackets) == 0 || len(config.StabilityAggregatePeriods) == 0 || len(config.RegimeGroups) == 0 {
		return fmt.Errorf("configuration arrays must not be empty")
	}
	if strings.TrimSpace(config.Symbol) == "" || strings.TrimSpace(config.Market) == "" || strings.TrimSpace(config.Interval) == "" || config.EvaluationStartMS <= 0 || config.EvaluationEndMS <= config.EvaluationStartMS {
		return fmt.Errorf("configuration scope/window is incomplete")
	}
	if time.UnixMilli(config.EvaluationStartMS).UTC().Year() != time.UnixMilli(config.EvaluationEndMS).UTC().Year() {
		return fmt.Errorf("configuration evaluation window must stay within one calendar year for the current stability profile")
	}
	if config.CostHaircutHorizonMinutes <= 0 || config.EntryDelayHorizonMinutes <= 0 || config.SeriesHorizonMinutes <= 0 || config.SeriesEntryDelayCandles < 0 || config.BracketWindowMinutes <= 0 || config.StabilityHorizonMinutes <= 0 || config.StabilityEntryDelay < 0 || config.RegimeHorizonMinutes <= 0 || config.RegimeLowSampleMinimum <= 0 || config.ClusterSeparationMinutes <= 0 || config.DiagnosticMinimumSamples <= 0 || config.ObservationsPerParameter <= 0 || config.ModelParameterCount < 0 || config.MetricPeriodsPerYear <= 0 {
		return fmt.Errorf("configuration contains invalid numeric bounds")
	}
	if config.SeriesEntryDelayCandles != 0 || config.StabilityEntryDelay != 0 {
		return fmt.Errorf("authoritative series and stability delay must be zero additional candles after canonical next-tradable entry")
	}
	for name, values := range map[string][]int{
		"forward_horizons_minutes":  config.ForwardHorizonsMinutes,
		"excursion_windows_minutes": config.ExcursionWindowsMinutes,
	} {
		if duplicateInt(values) {
			return fmt.Errorf("configuration %s contains duplicates", name)
		}
		for _, value := range values {
			if value <= 0 {
				return fmt.Errorf("configuration %s contains non-positive value", name)
			}
		}
	}
	if duplicateInt(config.EntryDelayCandles) {
		return fmt.Errorf("configuration entry_delay_candles contains duplicates")
	}
	for _, value := range config.EntryDelayCandles {
		if value < 0 {
			return fmt.Errorf("configuration entry_delay_candles contains negative value")
		}
	}
	for name, value := range map[string]float64{
		"series_cost_bps":                       config.SeriesCostBPS,
		"entry_delay_cost_bps":                  config.EntryDelayCostBPS,
		"bracket_cost_bps":                      config.BracketCostBPS,
		"stability_cost_bps":                    config.StabilityCostBPS,
		"regime_cost_bps":                       config.RegimeCostBPS,
		"observations_per_parameter":            config.ObservationsPerParameter,
		"metric_risk_free_rate":                 config.MetricRiskFreeRate,
		"metric_periods_per_year":               config.MetricPeriodsPerYear,
		"minimum_h2_pf":                         config.GateThresholds.MinimumH2PF,
		"minimum_h2_expectancy_bps":             config.GateThresholds.MinimumH2ExpectancyBPS,
		"minimum_fy_pf":                         config.GateThresholds.MinimumFYPF,
		"minimum_fy_expectancy_bps":             config.GateThresholds.MinimumFYExpectancyBPS,
		"minimum_delay_one_expectancy_bps":      config.GateThresholds.MinimumDelayOneExpectancyBPS,
		"maximum_single_month_contribution_pct": config.GateThresholds.MaximumSingleMonthContribution,
		"minimum_worst_quarter_pf":              config.GateThresholds.MinimumWorstQuarterPF,
		"minimum_bracket_pf":                    config.GateThresholds.MinimumBracketPF,
		"minimum_bracket_expectancy_bps":        config.GateThresholds.MinimumBracketExpectancyBPS,
	} {
		if math.IsNaN(value) || math.IsInf(value, 0) {
			return fmt.Errorf("configuration %s is non-finite", name)
		}
	}
	for _, values := range [][]float64{config.CostHaircutsBPS, {config.EntryDelayCostBPS, config.SeriesCostBPS, config.BracketCostBPS, config.StabilityCostBPS, config.RegimeCostBPS}} {
		for _, value := range values {
			if math.IsNaN(value) || math.IsInf(value, 0) || value < 0 {
				return fmt.Errorf("configuration contains invalid cost value")
			}
		}
	}
	if duplicateFloatBits(config.CostHaircutsBPS) {
		return fmt.Errorf("configuration cost_haircuts_bps contains duplicates")
	}
	if config.GateThresholds.MinimumEvents <= 0 || config.GateThresholds.MinimumPositiveMonths < 0 || config.GateThresholds.MinimumH2PF < 0 || config.GateThresholds.MinimumFYPF < 0 || config.GateThresholds.MinimumWorstQuarterPF < 0 || config.GateThresholds.MinimumBracketPF < 0 || config.GateThresholds.MaximumSingleMonthContribution < 0 || config.GateThresholds.MaximumSingleMonthContribution > 100 {
		return fmt.Errorf("configuration gate thresholds contain invalid bounds")
	}
	if duplicateString(config.StabilityAggregatePeriods) || duplicateString(config.RegimeGroups) {
		return fmt.Errorf("configuration stability/regime arrays contain duplicates")
	}
	for _, value := range append(append([]string(nil), config.StabilityAggregatePeriods...), config.RegimeGroups...) {
		if strings.TrimSpace(value) == "" {
			return fmt.Errorf("configuration stability/regime arrays contain empty value")
		}
	}
	for _, bracket := range config.Brackets {
		if strings.TrimSpace(bracket.Name) == "" || bracket.TPBPS <= 0 || bracket.SLBPS <= 0 || math.IsNaN(bracket.TPBPS) || math.IsInf(bracket.TPBPS, 0) || math.IsNaN(bracket.SLBPS) || math.IsInf(bracket.SLBPS, 0) {
			return fmt.Errorf("configuration contains invalid bracket")
		}
	}
	for name, value := range map[string]string{
		"feature_set_id":       config.FeatureSetID,
		"feature_set_version":  config.FeatureSetVersion,
		"regime_definition_id": config.RegimeDefinitionID,
		"regime_version":       config.RegimeVersion,
		"missing_value_policy": config.MissingValuePolicy,
		"filtering_policy":     config.FilteringPolicy,
	} {
		if strings.TrimSpace(value) == "" {
			return fmt.Errorf("configuration %s is empty", name)
		}
	}
	if config.FeatureSetID != featureSetID || config.FeatureSetVersion != featureSetVersion || config.RegimeDefinitionID != regimeDefinitionID || config.RegimeVersion != regimeDefinitionVer || config.MissingValuePolicy != missingValuePolicy || config.FilteringPolicy != filteringPolicy {
		return fmt.Errorf("configuration declares an unsupported feature, regime, missing-value, or filtering identity")
	}
	if config.EntryDelayCostBPS != 5 || config.BracketCostBPS != 5 || config.StabilityCostBPS != 5 || config.RegimeCostBPS != 5 {
		return fmt.Errorf("configuration fixed after-5-bps metric costs must remain 5")
	}
	requiredPeriods := []string{"Q1", "Q2", "Q3", "Q4", "H1", "H2", "FY"}
	if !sameStringSet(config.StabilityAggregatePeriods, requiredPeriods) {
		return fmt.Errorf("configuration stability periods must contain the exact current aggregate profile")
	}
	allowedRegimeGroups := map[string]struct{}{"composite": {}, "volatility": {}, "trend": {}, "liquidity": {}, "market_beta": {}}
	for _, group := range config.RegimeGroups {
		if _, ok := allowedRegimeGroups[group]; !ok {
			return fmt.Errorf("configuration contains unsupported regime group %q", group)
		}
	}
	if duplicateString(config.BuildTags) || !sort.StringsAreSorted(config.BuildTags) {
		return fmt.Errorf("duplicate build tags")
	}
	return nil
}

func ConfigurationHash(config ResolvedResearchConfiguration) (ConfigurationIdentity, error) {
	if err := validateResolvedConfiguration(config); err != nil {
		return ConfigurationIdentity{}, err
	}
	canonical, err := canonicalConfiguration(config)
	if err != nil {
		return ConfigurationIdentity{}, err
	}
	identity := ConfigurationIdentity{
		Contract:                       canonicalcontract.NewHeader(configurationSchemaName, canonicalContractVersion, configurationRole),
		CanonicalResearchConfiguration: canonical,
		EncodingVersion:                configurationEncoding,
		Effective:                      config,
	}
	identity.ArtifactHash, err = artifactHash(configurationSchemaName, configurationRole, identity)
	if err != nil {
		return ConfigurationIdentity{}, err
	}
	identity.Hash = identity.ArtifactHash
	return identity, nil
}

func duplicateString(values []string) bool {
	seen := map[string]struct{}{}
	for _, value := range values {
		if _, exists := seen[value]; exists {
			return true
		}
		seen[value] = struct{}{}
	}
	return false
}

func duplicateInt(values []int) bool {
	seen := map[int]struct{}{}
	for _, value := range values {
		if _, exists := seen[value]; exists {
			return true
		}
		seen[value] = struct{}{}
	}
	return false
}

func duplicateFloatBits(values []float64) bool {
	seen := map[uint64]struct{}{}
	for _, value := range values {
		bits := math.Float64bits(value)
		if _, exists := seen[bits]; exists {
			return true
		}
		seen[bits] = struct{}{}
	}
	return false
}

func sameStringSet(actual, expected []string) bool {
	if len(actual) != len(expected) || duplicateString(actual) {
		return false
	}
	actualCopy := append([]string(nil), actual...)
	expectedCopy := append([]string(nil), expected...)
	sort.Strings(actualCopy)
	sort.Strings(expectedCopy)
	return strings.Join(actualCopy, "\x00") == strings.Join(expectedCopy, "\x00")
}
