package researchidentity

import (
	"fmt"
	"math"
	"time"

	"github.com/david22573/ak-engine/internal/canonicalcontract"
	"github.com/david22573/ak-engine/internal/features"
	"github.com/david22573/ak-engine/internal/regime"
	"github.com/david22573/ak-engine/pkg/protocol"
)

const (
	candidateRegistrationSchemaName   = "ak.engine.candidate_registration"
	candidateRegistrationRole         = "candidate_registration"
	candidateImplementationSchemaName = "ak.engine.candidate_implementation_inventory"
	candidateImplementationRole       = "candidate_implementation_inventory"
	configurationSchemaName           = "ak.engine.effective_research_configuration"
	configurationRole                 = "effective_research_configuration"
	engineSourceSchemaName            = "ak.engine.source_identity"
	engineSourceRole                  = "engine_source_identity"
	featureSchemaName                 = "ak.engine.feature_identity"
	featureRole                       = "feature_identity"
	regimeSchemaName                  = "ak.engine.regime_identity"
	regimeRole                        = "regime_identity"
	consumedInputSchemaName           = "ak.engine.consumed_input_identity"
	consumedInputRole                 = "consumed_input_identity"
	evaluationSeriesSchemaName        = "ak.engine.evaluation_series_identity"
	evaluationSeriesRole              = "evaluation_series_identity"
	boundIdentitySchemaName           = "ak.engine.bound_research_identity"
	boundIdentityRole                 = "bound_research_identity"
	historianDatasetSchemaName        = "ak.historian.dataset_identity"
	historianDatasetRole              = "dataset_identity"
	historianPITSchemaName            = "ak.historian.pit_evidence"
	historianPITRole                  = "pit_evidence"
	rawObjectContractName             = "ak.raw.object"
	canonicalContractVersion          = 1
	canonicalDecimalScale             = 18
	canonicalMetricScale              = 8
)

func artifactHash(schemaName, role string, value any) (string, error) {
	hash, _, err := canonicalcontract.HashArtifactValue(schemaName, canonicalContractVersion, role, "artifact_hash", value)
	return hash, err
}

func canonicalConfiguration(config ResolvedResearchConfiguration) (CanonicalResearchConfiguration, error) {
	start, err := canonicalcontract.FormatTimestampMillis(config.EvaluationStartMS)
	if err != nil {
		return CanonicalResearchConfiguration{}, err
	}
	end, err := canonicalcontract.FormatTimestampMillis(config.EvaluationEndMS)
	if err != nil {
		return CanonicalResearchConfiguration{}, err
	}
	result := CanonicalResearchConfiguration{
		Symbol:                     config.Symbol,
		Market:                     config.Market,
		Interval:                   config.Interval,
		EvaluationStartUTC:         start,
		EvaluationEndUTC:           end,
		EntryDelayCandles:          append([]int{}, config.EntryDelayCandles...),
		SeriesEntryDelayCandles:    config.SeriesEntryDelayCandles,
		StabilityAggregatePeriods:  append([]string{}, config.StabilityAggregatePeriods...),
		StabilityEntryDelayCandles: config.StabilityEntryDelay,
		RegimeGroups:               append([]string{}, config.RegimeGroups...),
		RegimeLowSampleMinimum:     config.RegimeLowSampleMinimum,
		DiagnosticMinimumSamples:   config.DiagnosticMinimumSamples,
		ModelParameterCount:        config.ModelParameterCount,
		FeatureSetID:               config.FeatureSetID,
		FeatureSetVersion:          config.FeatureSetVersion,
		RegimeDefinitionID:         config.RegimeDefinitionID,
		RegimeVersion:              config.RegimeVersion,
		MissingValuePolicy:         config.MissingValuePolicy,
		FilteringPolicy:            config.FilteringPolicy,
		BuildTags:                  append([]string{}, config.BuildTags...),
	}
	if result.ForwardHorizonsNS, err = minutesSliceToNanoseconds(config.ForwardHorizonsMinutes); err != nil {
		return CanonicalResearchConfiguration{}, err
	}
	if result.CostHaircutHorizonNS, err = minutesToNanoseconds(config.CostHaircutHorizonMinutes); err != nil {
		return CanonicalResearchConfiguration{}, err
	}
	if result.EntryDelayHorizonNS, err = minutesToNanoseconds(config.EntryDelayHorizonMinutes); err != nil {
		return CanonicalResearchConfiguration{}, err
	}
	if result.SeriesHorizonNS, err = minutesToNanoseconds(config.SeriesHorizonMinutes); err != nil {
		return CanonicalResearchConfiguration{}, err
	}
	if result.ExcursionWindowsNS, err = minutesSliceToNanoseconds(config.ExcursionWindowsMinutes); err != nil {
		return CanonicalResearchConfiguration{}, err
	}
	if result.BracketWindowNS, err = minutesToNanoseconds(config.BracketWindowMinutes); err != nil {
		return CanonicalResearchConfiguration{}, err
	}
	if result.StabilityHorizonNS, err = minutesToNanoseconds(config.StabilityHorizonMinutes); err != nil {
		return CanonicalResearchConfiguration{}, err
	}
	if result.RegimeHorizonNS, err = minutesToNanoseconds(config.RegimeHorizonMinutes); err != nil {
		return CanonicalResearchConfiguration{}, err
	}
	if result.ClusterSeparationNS, err = minutesToNanoseconds(config.ClusterSeparationMinutes); err != nil {
		return CanonicalResearchConfiguration{}, err
	}
	if result.CostHaircutsBPS, err = decimalSlice(config.CostHaircutsBPS, canonicalMetricScale); err != nil {
		return CanonicalResearchConfiguration{}, err
	}
	if result.EntryDelayCostBPS, err = canonicalcontract.FormatDecimal(config.EntryDelayCostBPS, canonicalMetricScale); err != nil {
		return CanonicalResearchConfiguration{}, err
	}
	if result.SeriesCostBPS, err = canonicalcontract.FormatDecimal(config.SeriesCostBPS, canonicalMetricScale); err != nil {
		return CanonicalResearchConfiguration{}, err
	}
	if result.BracketCostBPS, err = canonicalcontract.FormatDecimal(config.BracketCostBPS, canonicalMetricScale); err != nil {
		return CanonicalResearchConfiguration{}, err
	}
	if result.StabilityCostBPS, err = canonicalcontract.FormatDecimal(config.StabilityCostBPS, canonicalMetricScale); err != nil {
		return CanonicalResearchConfiguration{}, err
	}
	if result.RegimeCostBPS, err = canonicalcontract.FormatDecimal(config.RegimeCostBPS, canonicalMetricScale); err != nil {
		return CanonicalResearchConfiguration{}, err
	}
	if result.ObservationsPerParameter, err = canonicalcontract.FormatDecimal(config.ObservationsPerParameter, canonicalMetricScale); err != nil {
		return CanonicalResearchConfiguration{}, err
	}
	if result.MetricRiskFreeRate, err = canonicalcontract.FormatDecimal(config.MetricRiskFreeRate, canonicalDecimalScale); err != nil {
		return CanonicalResearchConfiguration{}, err
	}
	if result.MetricPeriodsPerYear, err = canonicalcontract.FormatDecimal(config.MetricPeriodsPerYear, canonicalMetricScale); err != nil {
		return CanonicalResearchConfiguration{}, err
	}
	result.Brackets = make([]CanonicalBracketConfiguration, len(config.Brackets))
	for index, bracket := range config.Brackets {
		tp, err := canonicalcontract.FormatDecimal(bracket.TPBPS, canonicalMetricScale)
		if err != nil {
			return CanonicalResearchConfiguration{}, err
		}
		sl, err := canonicalcontract.FormatDecimal(bracket.SLBPS, canonicalMetricScale)
		if err != nil {
			return CanonicalResearchConfiguration{}, err
		}
		result.Brackets[index] = CanonicalBracketConfiguration{Name: bracket.Name, TPBPS: tp, SLBPS: sl}
	}
	thresholds := config.GateThresholds
	result.GateThresholds = CanonicalGateThresholds{
		MinimumEvents:         thresholds.MinimumEvents,
		MinimumPositiveMonths: thresholds.MinimumPositiveMonths,
	}
	thresholdValues := []struct {
		source float64
		target *string
	}{
		{thresholds.MinimumH2PF, &result.GateThresholds.MinimumH2PF},
		{thresholds.MinimumH2ExpectancyBPS, &result.GateThresholds.MinimumH2ExpectancyBPS},
		{thresholds.MinimumFYPF, &result.GateThresholds.MinimumFYPF},
		{thresholds.MinimumFYExpectancyBPS, &result.GateThresholds.MinimumFYExpectancyBPS},
		{thresholds.MinimumDelayOneExpectancyBPS, &result.GateThresholds.MinimumDelayOneExpectancyBPS},
		{thresholds.MaximumSingleMonthContribution, &result.GateThresholds.MaximumSingleMonthContributionPct},
		{thresholds.MinimumWorstQuarterPF, &result.GateThresholds.MinimumWorstQuarterPF},
		{thresholds.MinimumBracketPF, &result.GateThresholds.MinimumBracketPF},
		{thresholds.MinimumBracketExpectancyBPS, &result.GateThresholds.MinimumBracketExpectancyBPS},
	}
	for _, threshold := range thresholdValues {
		*threshold.target, err = canonicalcontract.FormatDecimal(threshold.source, canonicalMetricScale)
		if err != nil {
			return CanonicalResearchConfiguration{}, err
		}
	}
	return result, nil
}

func minutesToNanoseconds(value int) (int64, error) {
	if value < 0 || int64(value) > math.MaxInt64/int64(time.Minute) {
		return 0, fmt.Errorf("duration minutes %d overflows nanoseconds", value)
	}
	return int64(value) * int64(time.Minute), nil
}

func minutesSliceToNanoseconds(values []int) ([]int64, error) {
	result := make([]int64, len(values))
	for index, value := range values {
		converted, err := minutesToNanoseconds(value)
		if err != nil {
			return nil, err
		}
		result[index] = converted
	}
	return result, nil
}

func decimalSlice(values []float64, scale int) ([]string, error) {
	result := make([]string, len(values))
	for index, value := range values {
		formatted, err := canonicalcontract.FormatDecimal(value, scale)
		if err != nil {
			return nil, err
		}
		result[index] = formatted
	}
	return result, nil
}

func canonicalFeatureRows(rows []features.Row) ([]CanonicalFeatureRow, error) {
	result := make([]CanonicalFeatureRow, len(rows))
	for index, row := range rows {
		eventTime, err := canonicalcontract.FormatTimestampMillis(row.EventTimeMS)
		if err != nil {
			return nil, err
		}
		availableAt, err := canonicalcontract.FormatTimestampMillis(row.AvailableAtMS)
		if err != nil {
			return nil, err
		}
		values := []float64{row.Close, row.Return1, row.Return5, row.Return15, row.RealizedVol20, row.RealizedVol60, row.ATR14, row.ATRPct14, row.BBWidth20, row.BBWidthPctRank60, row.EMA20, row.EMA50, row.EMA200, row.TrendSlope20, row.VolumeRatio20, row.QuoteVolumeRatio20, row.TakerBuyRatio, row.BTCReturn60, row.ETHReturn60}
		formatted, err := decimalSlice(values, canonicalDecimalScale)
		if err != nil {
			return nil, err
		}
		result[index] = CanonicalFeatureRow{
			Market: row.Market, Symbol: row.Symbol, Interval: row.Interval, EventTimeUTC: eventTime, AvailableAtUTC: availableAt,
			Close: formatted[0], Return1: formatted[1], Return5: formatted[2], Return15: formatted[3], RealizedVol20: formatted[4], RealizedVol60: formatted[5], ATR14: formatted[6], ATRPct14: formatted[7], BBWidth20: formatted[8], BBWidthPctRank60: formatted[9], EMA20: formatted[10], EMA50: formatted[11], EMA200: formatted[12], TrendSlope20: formatted[13], VolumeRatio20: formatted[14], QuoteVolumeRatio20: formatted[15], TakerBuyRatio: formatted[16], BTCReturn60: formatted[17], ETHReturn60: formatted[18], Warmup: row.Warmup,
		}
	}
	return result, nil
}

func canonicalRegimeRows(rows []regime.Label) ([]CanonicalRegimeRow, error) {
	result := make([]CanonicalRegimeRow, len(rows))
	for index, row := range rows {
		eventTime, err := canonicalcontract.FormatTimestampMillis(row.EventTimeMS)
		if err != nil {
			return nil, err
		}
		availableAt, err := canonicalcontract.FormatTimestampMillis(row.AvailableAtMS)
		if err != nil {
			return nil, err
		}
		result[index] = CanonicalRegimeRow{
			Market: row.Market, Symbol: row.Symbol, Interval: row.Interval, EventTimeUTC: eventTime, AvailableAtUTC: availableAt,
			Volatility: row.Volatility, Trend: row.Trend, Liquidity: row.Liquidity, MarketBeta: row.MarketBeta, Sentiment: row.Sentiment, Composite: row.Composite, Reasons: append([]string{}, row.Reasons...), Warmup: row.Warmup,
		}
	}
	return result, nil
}

func canonicalCandles(rows []protocol.Candle) ([]CanonicalCandleRow, error) {
	result := make([]CanonicalCandleRow, len(rows))
	for index, row := range rows {
		openTime, err := canonicalcontract.FormatTimestampMillis(row.OpenTimeMS)
		if err != nil {
			return nil, err
		}
		closeTime, err := canonicalcontract.FormatTimestampMillis(row.CloseTimeMS)
		if err != nil {
			return nil, err
		}
		formatted, err := decimalSlice([]float64{row.Open, row.High, row.Low, row.Close, row.Volume, row.QuoteAssetVolume, row.TakerBuyBaseVolume, row.TakerBuyQuoteVolume}, canonicalDecimalScale)
		if err != nil {
			return nil, err
		}
		result[index] = CanonicalCandleRow{
			Market: row.Market, Symbol: row.Symbol, Interval: row.Interval, OpenTimeUTC: openTime,
			Open: formatted[0], High: formatted[1], Low: formatted[2], Close: formatted[3], Volume: formatted[4], CloseTimeUTC: closeTime,
			QuoteAssetVolume: formatted[5], NumberOfTrades: row.NumberOfTrades, TakerBuyBaseVolume: formatted[6], TakerBuyQuoteVolume: formatted[7],
		}
	}
	return result, nil
}

func canonicalTimestamps(values []int64) ([]string, error) {
	result := make([]string, len(values))
	for index, value := range values {
		formatted, err := canonicalcontract.FormatTimestampMillis(value)
		if err != nil {
			return nil, err
		}
		result[index] = formatted
	}
	return result, nil
}
