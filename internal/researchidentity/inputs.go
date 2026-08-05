package researchidentity

import (
	"context"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"strings"
	"time"

	"github.com/david22573/ak-engine/internal/canonicalcontract"
	"github.com/david22573/ak-engine/internal/data"
	"github.com/david22573/ak-engine/internal/features"
	"github.com/david22573/ak-engine/internal/regime"
)

const (
	consumedInputEncoding   = "ak.canonical.json.v1"
	seriesEncoding          = "ak.canonical.json.v1"
	seriesGenerationVersion = "deep-return-series.v1"
)

var featureImplementationFiles = []ImplementationFile{
	{Path: "internal/features/candles.go", InclusionReason: "feature generation from candle inputs"},
	{Path: "internal/features/correlation.go", InclusionReason: "cross-asset feature behavior"},
	{Path: "internal/features/liquidity.go", InclusionReason: "liquidity feature behavior"},
	{Path: "internal/features/reader.go", InclusionReason: "feature artifact decoding"},
	{Path: "internal/features/row.go", InclusionReason: "feature row schema"},
	{Path: "internal/features/trend.go", InclusionReason: "trend feature behavior"},
	{Path: "internal/features/volatility.go", InclusionReason: "volatility feature behavior"},
	{Path: "internal/features/writer.go", InclusionReason: "feature artifact encoding"},
}

var regimeImplementationFiles = []ImplementationFile{
	{Path: "internal/regime/classifier.go", InclusionReason: "regime classification behavior"},
	{Path: "internal/regime/reader.go", InclusionReason: "regime artifact decoding"},
	{Path: "internal/regime/report.go", InclusionReason: "regime artifact encoding"},
	{Path: "internal/regime/taxonomy.go", InclusionReason: "regime schema and taxonomy"},
	{Path: "internal/regime/thresholds.go", InclusionReason: "regime threshold behavior"},
}

func deriveFeatureIdentity(root, artifactPath string, rows []features.Row, config ResolvedResearchConfiguration, source EngineSourceIdentity, dataset DatasetIdentity) (FeatureIdentity, error) {
	if strings.TrimSpace(artifactPath) == "" || len(rows) == 0 {
		return FeatureIdentity{}, fmt.Errorf("feature artifact and rows are required")
	}
	if err := validateFeatureRows(rows, config, dataset); err != nil {
		return FeatureIdentity{}, err
	}
	if err := verifyFeatureArtifactRows(artifactPath, rows); err != nil {
		return FeatureIdentity{}, err
	}
	outputHash, _, err := hashFileRole(artifactPath, "feature_artifact")
	if err != nil {
		return FeatureIdentity{}, err
	}
	implementationHash, files, err := buildNamedSourceInventory(root, featureSchemaName, "feature_implementation", featureImplementationFiles)
	if err != nil {
		return FeatureIdentity{}, err
	}
	configurationHash, err := featureConfigurationHash(config)
	if err != nil {
		return FeatureIdentity{}, err
	}
	windowStart, err := canonicalcontract.FormatTimestampMillis(rows[0].EventTimeMS)
	if err != nil {
		return FeatureIdentity{}, err
	}
	windowEnd, err := canonicalcontract.FormatTimestampMillis(rows[len(rows)-1].EventTimeMS)
	if err != nil {
		return FeatureIdentity{}, err
	}
	identity := FeatureIdentity{
		Contract:     canonicalcontract.NewHeader(featureSchemaName, canonicalContractVersion, featureRole),
		FeatureSetID: config.FeatureSetID, FeatureSetVersion: config.FeatureSetVersion,
		ConfigurationHash: configurationHash, ImplementationHash: implementationHash, ImplementationFiles: files,
		InputDatasetHash: dataset.ArtifactHash, OutputArtifactHash: outputHash,
		WindowStartUTC: windowStart, WindowEndUTC: windowEnd,
		WindowStartMS: rows[0].EventTimeMS, WindowEndMS: rows[len(rows)-1].EventTimeMS, RowCount: len(rows),
		ImplementationCommit: source.CommitSHA,
	}
	identity.ArtifactHash, err = artifactHash(featureSchemaName, featureRole, identity)
	if err != nil {
		return FeatureIdentity{}, err
	}
	return identity, nil
}

func deriveRegimeIdentity(root, artifactPath string, labels []regime.Label, config ResolvedResearchConfiguration, source EngineSourceIdentity, dataset DatasetIdentity, feature FeatureIdentity) (RegimeIdentity, error) {
	if strings.TrimSpace(artifactPath) == "" || len(labels) == 0 {
		return RegimeIdentity{}, fmt.Errorf("regime artifact and rows are required")
	}
	if err := validateRegimeRows(labels, config, dataset, feature); err != nil {
		return RegimeIdentity{}, err
	}
	if err := verifyRegimeArtifactRows(artifactPath, labels); err != nil {
		return RegimeIdentity{}, err
	}
	outputHash, _, err := hashFileRole(artifactPath, "regime_artifact")
	if err != nil {
		return RegimeIdentity{}, err
	}
	implementationHash, files, err := buildNamedSourceInventory(root, regimeSchemaName, "regime_implementation", regimeImplementationFiles)
	if err != nil {
		return RegimeIdentity{}, err
	}
	configurationHash, err := regimeConfigurationHash(config)
	if err != nil {
		return RegimeIdentity{}, err
	}
	windowStart, err := canonicalcontract.FormatTimestampMillis(labels[0].EventTimeMS)
	if err != nil {
		return RegimeIdentity{}, err
	}
	windowEnd, err := canonicalcontract.FormatTimestampMillis(labels[len(labels)-1].EventTimeMS)
	if err != nil {
		return RegimeIdentity{}, err
	}
	identity := RegimeIdentity{
		Contract:           canonicalcontract.NewHeader(regimeSchemaName, canonicalContractVersion, regimeRole),
		RegimeDefinitionID: config.RegimeDefinitionID, RegimeVersion: config.RegimeVersion,
		ConfigurationHash: configurationHash, ImplementationHash: implementationHash, ImplementationFiles: files,
		InputDatasetHash: dataset.ArtifactHash, InputFeatureHash: feature.OutputArtifactHash, OutputArtifactHash: outputHash,
		WindowStartUTC: windowStart, WindowEndUTC: windowEnd,
		WindowStartMS: labels[0].EventTimeMS, WindowEndMS: labels[len(labels)-1].EventTimeMS, RowCount: len(labels),
		ImplementationCommit: source.CommitSHA,
	}
	identity.ArtifactHash, err = artifactHash(regimeSchemaName, regimeRole, identity)
	if err != nil {
		return RegimeIdentity{}, err
	}
	return identity, nil
}

func featureConfigurationHash(config ResolvedResearchConfiguration) (string, error) {
	value := struct {
		FeatureSetID       string `json:"feature_set_id"`
		FeatureSetVersion  string `json:"feature_set_version"`
		Market             string `json:"market"`
		Symbol             string `json:"symbol"`
		Interval           string `json:"interval"`
		MissingValuePolicy string `json:"missing_value_policy"`
	}{config.FeatureSetID, config.FeatureSetVersion, config.Market, config.Symbol, config.Interval, config.MissingValuePolicy}
	hash, _, err := canonicalcontract.HashValue(featureSchemaName, canonicalContractVersion, "feature_configuration", value)
	return hash, err
}

func regimeConfigurationHash(config ResolvedResearchConfiguration) (string, error) {
	value := struct {
		RegimeDefinitionID string   `json:"regime_definition_id"`
		RegimeVersion      string   `json:"regime_version"`
		RegimeGroups       []string `json:"regime_groups"`
		MissingValuePolicy string   `json:"missing_value_policy"`
	}{config.RegimeDefinitionID, config.RegimeVersion, append([]string(nil), config.RegimeGroups...), config.MissingValuePolicy}
	hash, _, err := canonicalcontract.HashValue(regimeSchemaName, canonicalContractVersion, "regime_configuration", value)
	return hash, err
}

func buildNamedSourceInventory(root, schemaName, role string, files []ImplementationFile) (string, []FileInventoryEntry, error) {
	identity, err := buildImplementationIdentityForRole(root, files, role+"_source")
	if err != nil {
		return "", nil, err
	}
	hash, _, err := canonicalcontract.HashValue(schemaName, canonicalContractVersion, role, identity.Files)
	return hash, identity.Files, err
}

func validateFeatureRows(rows []features.Row, config ResolvedResearchConfiguration, dataset DatasetIdentity) error {
	cutoff, err := parseUTC(dataset.PointInTimeCutoffUTC)
	if err != nil {
		return err
	}
	for i, row := range rows {
		if row.EventTimeMS <= 0 || row.AvailableAtMS <= 0 || row.AvailableAtMS > row.EventTimeMS || time.UnixMilli(row.AvailableAtMS).After(cutoff) {
			return fmt.Errorf("feature row %d has invalid or late availability", i)
		}
		if i > 0 && row.EventTimeMS <= rows[i-1].EventTimeMS {
			return fmt.Errorf("feature timestamps are not strictly increasing")
		}
		if strings.ToUpper(row.Symbol) != config.Symbol || row.Market != config.Market || row.Interval != config.Interval {
			return fmt.Errorf("feature row %d scope conflicts with configuration", i)
		}
		for name, value := range featureFloats(row) {
			if math.IsNaN(value) || math.IsInf(value, 0) {
				return fmt.Errorf("feature row %d %s is non-finite", i, name)
			}
		}
	}
	if rows[0].EventTimeMS != config.EvaluationStartMS || rows[len(rows)-1].EventTimeMS != config.EvaluationEndMS {
		return fmt.Errorf("feature window conflicts with effective configuration")
	}
	start, _ := parseUTC(dataset.StartUTC)
	end, _ := parseUTC(dataset.EndUTC)
	if time.UnixMilli(rows[0].EventTimeMS).Before(start) || time.UnixMilli(rows[len(rows)-1].EventTimeMS).After(end) {
		return fmt.Errorf("feature window is outside dataset window")
	}
	return nil
}

func validateRegimeRows(labels []regime.Label, config ResolvedResearchConfiguration, dataset DatasetIdentity, feature FeatureIdentity) error {
	cutoff, err := parseUTC(dataset.PointInTimeCutoffUTC)
	if err != nil {
		return err
	}
	for i, label := range labels {
		if label.EventTimeMS <= 0 || label.AvailableAtMS <= 0 || label.AvailableAtMS > label.EventTimeMS || time.UnixMilli(label.AvailableAtMS).After(cutoff) {
			return fmt.Errorf("regime row %d has invalid or late availability", i)
		}
		if i > 0 && label.AvailableAtMS <= labels[i-1].AvailableAtMS {
			return fmt.Errorf("regime availability timestamps are not strictly increasing")
		}
		if i > 0 && label.EventTimeMS <= labels[i-1].EventTimeMS {
			return fmt.Errorf("regime event timestamps are not strictly increasing")
		}
		if strings.ToUpper(label.Symbol) != config.Symbol || label.Market != config.Market || label.Interval != config.Interval {
			return fmt.Errorf("regime row %d scope conflicts with configuration", i)
		}
	}
	if labels[0].EventTimeMS != feature.WindowStartMS || labels[len(labels)-1].EventTimeMS != feature.WindowEndMS {
		return fmt.Errorf("regime window conflicts with feature window")
	}
	return nil
}

func deriveConsumedInput(request DerivationRequest, dataset DatasetIdentity, feature FeatureIdentity, regimeIdentity *RegimeIdentity) (ConsumedInputIdentity, error) {
	if len(request.FeatureRows) == 0 || len(request.Candles) == 0 || len(request.EvaluationEventTimestamps) == 0 {
		return ConsumedInputIdentity{}, fmt.Errorf("actual consumed rows/events are incomplete")
	}
	if err := validateActualConsumedInputs(request, dataset); err != nil {
		return ConsumedInputIdentity{}, err
	}
	objectByPath := map[string]DatasetObjectIdentity{}
	for _, object := range dataset.Objects {
		objectByPath[object.RelativePath] = object
	}
	objectBindings := make([]string, 0, len(request.ConsumedDatasetPaths))
	objectRecords := make([]ConsumedDatasetObject, 0, len(request.ConsumedDatasetPaths))
	seen := map[string]struct{}{}
	for _, path := range request.ConsumedDatasetPaths {
		relative, err := relativePathWithin(request.DatasetRoot, path)
		if err != nil {
			return ConsumedInputIdentity{}, err
		}
		if _, exists := seen[relative]; exists {
			return ConsumedInputIdentity{}, fmt.Errorf("duplicate consumed dataset path %s", relative)
		}
		seen[relative] = struct{}{}
		object, exists := objectByPath[relative]
		if !exists {
			return ConsumedInputIdentity{}, fmt.Errorf("consumed object %s not bound by dataset", relative)
		}
		objectBindings = append(objectBindings, relative+"@"+object.SHA256)
		objectRecords = append(objectRecords, ConsumedDatasetObject{RelativePath: relative, ObjectHash: object.SHA256})
	}
	if len(objectBindings) == 0 {
		return ConsumedInputIdentity{}, fmt.Errorf("consumed dataset object list is empty")
	}
	sort.Strings(objectBindings)
	sort.Slice(objectRecords, func(i, j int) bool { return objectRecords[i].RelativePath < objectRecords[j].RelativePath })
	regimeHash := ""
	if regimeIdentity != nil {
		regimeHash = regimeIdentity.OutputArtifactHash
	}
	evaluationStart, err := canonicalcontract.FormatTimestampMillis(request.Configuration.EvaluationStartMS)
	if err != nil {
		return ConsumedInputIdentity{}, err
	}
	evaluationEnd, err := canonicalcontract.FormatTimestampMillis(request.Configuration.EvaluationEndMS)
	if err != nil {
		return ConsumedInputIdentity{}, err
	}
	featureRows, err := canonicalFeatureRows(request.FeatureRows)
	if err != nil {
		return ConsumedInputIdentity{}, err
	}
	regimeRows, err := canonicalRegimeRows(request.RegimeLabels)
	if err != nil {
		return ConsumedInputIdentity{}, err
	}
	candles, err := canonicalCandles(request.Candles)
	if err != nil {
		return ConsumedInputIdentity{}, err
	}
	eventTimestamps, err := canonicalTimestamps(request.EvaluationEventTimestamps)
	if err != nil {
		return ConsumedInputIdentity{}, err
	}
	inputSeriesCount := 2
	if regimeIdentity != nil {
		inputSeriesCount++
	}
	identity := ConsumedInputIdentity{
		Contract:        canonicalcontract.NewHeader(consumedInputSchemaName, canonicalContractVersion, consumedInputRole),
		EncodingVersion: consumedInputEncoding, DatasetHash: dataset.ArtifactHash,
		DatasetObjectRecords: objectRecords,
		DatasetObjects:       objectBindings, FeatureHash: feature.OutputArtifactHash, RegimeHash: regimeHash,
		Symbols: append([]string(nil), dataset.Symbols...), EvaluationStartUTC: evaluationStart, EvaluationEndUTC: evaluationEnd,
		EvaluationStartMS: request.Configuration.EvaluationStartMS,
		EvaluationEndMS:   request.Configuration.EvaluationEndMS, PointInTimeCutoffUTC: dataset.PointInTimeCutoffUTC,
		MissingValuePolicy: request.Configuration.MissingValuePolicy, FilteringPolicy: request.Configuration.FilteringPolicy,
		FeatureRows: featureRows, RegimeRows: regimeRows, Candles: candles, EvaluationEventTimestamps: eventTimestamps,
		FeatureRowCount: len(request.FeatureRows), RegimeRowCount: len(request.RegimeLabels), CandleRowCount: len(request.Candles),
		EvaluationEventCount: len(request.EvaluationEventTimestamps), InputSeriesCount: inputSeriesCount,
	}
	identity.ArtifactHash, err = artifactHash(consumedInputSchemaName, consumedInputRole, identity)
	if err != nil {
		return ConsumedInputIdentity{}, err
	}
	identity.Hash = identity.ArtifactHash
	return identity, nil
}

func verifyFeatureArtifactRows(path string, expected []features.Row) error {
	raw, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	var actual []features.Row
	if err := decodeStrictJSON(raw, &actual); err != nil {
		return fmt.Errorf("feature artifact parse: %w", err)
	}
	if !reflect.DeepEqual(actual, expected) {
		return fmt.Errorf("feature artifact rows differ from consumed rows")
	}
	return nil
}

func verifyRegimeArtifactRows(path string, expected []regime.Label) error {
	raw, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	var actual []regime.Label
	if err := decodeStrictJSON(raw, &actual); err != nil {
		return fmt.Errorf("regime artifact parse: %w", err)
	}
	if !reflect.DeepEqual(actual, expected) {
		return fmt.Errorf("regime artifact rows differ from consumed rows")
	}
	return nil
}

func validateActualConsumedInputs(request DerivationRequest, dataset DatasetIdentity) error {
	featureTimes := make(map[int64]struct{}, len(request.FeatureRows))
	for _, row := range request.FeatureRows {
		featureTimes[row.EventTimeMS] = struct{}{}
	}
	candleTimes := make(map[int64]struct{}, len(request.Candles))
	for i, candle := range request.Candles {
		if i > 0 && candle.OpenTimeMS <= request.Candles[i-1].OpenTimeMS {
			return fmt.Errorf("consumed candle timestamps are not strictly increasing")
		}
		if strings.ToUpper(candle.Symbol) != request.Configuration.Symbol || candle.Market != request.Configuration.Market || candle.Interval != request.Configuration.Interval {
			return fmt.Errorf("consumed candle %d scope conflicts with configuration", i)
		}
		candleTimes[candle.OpenTimeMS] = struct{}{}
	}
	for i, timestamp := range request.EvaluationEventTimestamps {
		if i > 0 && timestamp <= request.EvaluationEventTimestamps[i-1] {
			return fmt.Errorf("evaluation event timestamps are not strictly increasing")
		}
		if timestamp < request.Configuration.EvaluationStartMS || timestamp > request.Configuration.EvaluationEndMS {
			return fmt.Errorf("evaluation event timestamp %d is outside configured window", i)
		}
		if _, exists := featureTimes[timestamp]; !exists {
			return fmt.Errorf("evaluation event timestamp %d has no consumed feature row", i)
		}
		if _, exists := candleTimes[timestamp]; !exists {
			return fmt.Errorf("evaluation event timestamp %d has no consumed candle", i)
		}
	}
	start, err := parseUTC(dataset.StartUTC)
	if err != nil {
		return err
	}
	end, err := parseUTC(dataset.EndUTC)
	if err != nil {
		return err
	}
	firstCandle := request.Candles[0].OpenTimeMS
	lastCandle := request.Candles[len(request.Candles)-1].OpenTimeMS
	if time.UnixMilli(firstCandle).Before(start) || time.UnixMilli(lastCandle).After(end) {
		return fmt.Errorf("consumed candle window is outside dataset window")
	}
	reloaded, err := data.LoadExactParquetFiles(context.Background(), data.CandleRequest{
		Source: "local-parquet", Path: request.DatasetRoot,
		Market: request.Configuration.Market, Symbol: request.Configuration.Symbol, Interval: request.Configuration.Interval,
		From: time.UnixMilli(firstCandle).UTC(), To: time.UnixMilli(lastCandle).UTC(),
	}, request.ConsumedDatasetPaths)
	if err != nil {
		return fmt.Errorf("reconstruct consumed parquet rows: %w", err)
	}
	if !reflect.DeepEqual(reloaded, request.Candles) {
		return fmt.Errorf("claimed consumed candle rows differ from exact parquet objects")
	}
	return nil
}

func deriveSeriesIdentity(returns []float64, timestamps []int64, startMS, endMS int64) (EvaluationSeriesIdentity, error) {
	if len(returns) == 0 || len(timestamps) == 0 || len(returns) != len(timestamps) {
		return EvaluationSeriesIdentity{}, fmt.Errorf("return/timestamp/observation counts are empty or inconsistent")
	}
	for i, value := range returns {
		if math.IsNaN(value) || math.IsInf(value, 0) {
			return EvaluationSeriesIdentity{}, fmt.Errorf("return %d is non-finite", i)
		}
	}
	for i, value := range timestamps {
		if value < startMS || value > endMS {
			return EvaluationSeriesIdentity{}, fmt.Errorf("timestamp %d is outside evaluation window", i)
		}
		if i > 0 && value <= timestamps[i-1] {
			return EvaluationSeriesIdentity{}, fmt.Errorf("timestamps are not strictly increasing")
		}
	}
	canonicalReturns, err := decimalSlice(returns, canonicalDecimalScale)
	if err != nil {
		return EvaluationSeriesIdentity{}, err
	}
	canonicalTimes, err := canonicalTimestamps(timestamps)
	if err != nil {
		return EvaluationSeriesIdentity{}, err
	}
	returnHash, _, err := canonicalcontract.HashValue(evaluationSeriesSchemaName, canonicalContractVersion, "return_series", canonicalReturns)
	if err != nil {
		return EvaluationSeriesIdentity{}, err
	}
	timestampHash, _, err := canonicalcontract.HashValue(evaluationSeriesSchemaName, canonicalContractVersion, "timestamp_series", canonicalTimes)
	if err != nil {
		return EvaluationSeriesIdentity{}, err
	}
	identity := EvaluationSeriesIdentity{
		Contract:        canonicalcontract.NewHeader(evaluationSeriesSchemaName, canonicalContractVersion, evaluationSeriesRole),
		EncodingVersion: seriesEncoding, SeriesGenerationVersion: seriesGenerationVersion, Returns: canonicalReturns, Timestamps: canonicalTimes,
		ReturnSeriesHash: returnHash, TimestampSeriesHash: timestampHash,
		ObservationCount: len(returns), ReturnCount: len(returns), TimestampCount: len(timestamps),
	}
	identity.ArtifactHash, err = artifactHash(evaluationSeriesSchemaName, evaluationSeriesRole, identity)
	if err != nil {
		return EvaluationSeriesIdentity{}, err
	}
	identity.SeriesHash = identity.ArtifactHash
	return identity, nil
}

type namedFloat struct {
	name  string
	value float64
}

func rangeOrderedFeatureFloats(row features.Row) []namedFloat {
	return []namedFloat{
		{"close", row.Close}, {"return_1", row.Return1}, {"return_5", row.Return5}, {"return_15", row.Return15},
		{"realized_vol_20", row.RealizedVol20}, {"realized_vol_60", row.RealizedVol60}, {"atr_14", row.ATR14}, {"atr_pct_14", row.ATRPct14},
		{"bb_width_20", row.BBWidth20}, {"bb_width_pct_rank_60", row.BBWidthPctRank60}, {"ema_20", row.EMA20}, {"ema_50", row.EMA50},
		{"ema_200", row.EMA200}, {"trend_slope_20", row.TrendSlope20}, {"volume_ratio_20", row.VolumeRatio20},
		{"quote_volume_ratio_20", row.QuoteVolumeRatio20}, {"taker_buy_ratio", row.TakerBuyRatio}, {"btc_return_60", row.BTCReturn60}, {"eth_return_60", row.ETHReturn60},
	}
}

func featureFloats(row features.Row) map[string]float64 {
	values := map[string]float64{}
	for _, item := range rangeOrderedFeatureFloats(row) {
		values[item.name] = item.value
	}
	return values
}

func relativeCleanPath(root, path string) (string, error) {
	rootAbs, err := filepath.Abs(root)
	if err != nil {
		return "", err
	}
	pathAbs, err := filepath.Abs(path)
	if err != nil {
		return "", err
	}
	relative, err := filepath.Rel(rootAbs, pathAbs)
	if err != nil {
		return "", err
	}
	return normalizeRepositoryPath(filepath.ToSlash(relative))
}
