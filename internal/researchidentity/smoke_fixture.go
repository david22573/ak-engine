package researchidentity

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/david22573/ak-engine/internal/canonicalcontract"
	"github.com/david22573/ak-engine/internal/data"
	"github.com/david22573/ak-engine/internal/features"
	"github.com/david22573/ak-engine/internal/regime"
	"github.com/david22573/ak-engine/pkg/protocol"
	"github.com/xitongsys/parquet-go-source/local"
	"github.com/xitongsys/parquet-go/writer"
)

// DiagnosticSmokeFixture is an isolated, synthetic identity fixture. It is
// used only by the explicit research-diagnostics-smoke command; production
// evaluation always uses the real Git provider and actual Historian evidence.
type DiagnosticSmokeFixture struct {
	Request DerivationRequest
	Deriver *Deriver
	Cleanup func()
}

type smokeRepositoryProvider struct{ state RepositoryState }

func (p smokeRepositoryProvider) Resolve(string) (RepositoryState, error) { return p.state, nil }

func BuildDiagnosticSmokeFixture(parent string) (DiagnosticSmokeFixture, error) {
	fixtureRoot, err := os.MkdirTemp(parent, ".research-identity-smoke-")
	if err != nil {
		return DiagnosticSmokeFixture{}, err
	}
	failed := true
	defer func() {
		if failed {
			_ = os.RemoveAll(fixtureRoot)
		}
	}()
	repositoryRoot := filepath.Join(fixtureRoot, "engine")
	if err := writeSmokeImplementationFiles(repositoryRoot); err != nil {
		return DiagnosticSmokeFixture{}, err
	}

	const rowCount = 40
	startMS := time.Date(2024, time.January, 1, 0, 0, 0, 0, time.UTC).UnixMilli()
	evidenceRoot := filepath.Join(fixtureRoot, "evidence")
	datasetRoot := filepath.Join(evidenceRoot, "dataset")
	objectRelative := "candles/futures-um/1m/symbol=BTCUSDT/year=2024/month=01/BTCUSDT-1m-2024-01.parquet"
	objectPath := filepath.Join(datasetRoot, filepath.FromSlash(objectRelative))
	if err := writeSmokeParquet(objectPath, startMS, rowCount); err != nil {
		return DiagnosticSmokeFixture{}, err
	}
	objectHash, objectSize, err := hashFileRole(objectPath, "dataset_object")
	if err != nil {
		return DiagnosticSmokeFixture{}, err
	}

	archivePath := filepath.Join(evidenceRoot, "source", "archive.zip")
	if err := writeSmokeFile(archivePath, []byte("deterministic smoke source-archive bytes")); err != nil {
		return DiagnosticSmokeFixture{}, err
	}
	archiveRawHash, archiveSize, err := hashFileRole(archivePath, "source_archive")
	if err != nil {
		return DiagnosticSmokeFixture{}, err
	}
	availabilityPath := filepath.Join(evidenceRoot, "policies", "availability.json")
	coveragePath := filepath.Join(evidenceRoot, "policies", "coverage.json")
	availability := historianAvailabilityPolicy{
		Contract:            canonicalcontract.NewHeader(historianAvailabilitySchema, canonicalContractVersion, "availability_policy"),
		PolicyID:            "availability.smoke",
		PolicyVersion:       "1",
		AvailabilityDelayNS: int64(time.Minute),
	}
	availability.ArtifactHash, err = artifactHash(historianAvailabilitySchema, "availability_policy", availability)
	if err != nil {
		return DiagnosticSmokeFixture{}, err
	}
	coveragePolicy := historianCoveragePolicy{
		Contract:      canonicalcontract.NewHeader(historianCoverageSchema, canonicalContractVersion, "coverage_policy"),
		PolicyID:      "coverage.smoke",
		PolicyVersion: "1",
		Mode:          historianStrictCoverageMode,
		IntervalNS:    int64(time.Minute),
	}
	coveragePolicy.ArtifactHash, err = artifactHash(historianCoverageSchema, "coverage_policy", coveragePolicy)
	if err != nil {
		return DiagnosticSmokeFixture{}, err
	}
	if err := writeSmokeCanonicalArtifact(availabilityPath, availability); err != nil {
		return DiagnosticSmokeFixture{}, err
	}
	if err := writeSmokeCanonicalArtifact(coveragePath, coveragePolicy); err != nil {
		return DiagnosticSmokeFixture{}, err
	}
	availabilityData, err := os.ReadFile(availabilityPath)
	if err != nil {
		return DiagnosticSmokeFixture{}, err
	}
	coverageData, err := os.ReadFile(coveragePath)
	if err != nil {
		return DiagnosticSmokeFixture{}, err
	}

	availabilityRawHash, err := canonicalcontract.HashRaw(rawObjectContractName, canonicalContractVersion, "availability_policy_source", availabilityData)
	if err != nil {
		return DiagnosticSmokeFixture{}, err
	}
	coverageRawHash, err := canonicalcontract.HashRaw(rawObjectContractName, canonicalContractVersion, "coverage_policy_source", coverageData)
	if err != nil {
		return DiagnosticSmokeFixture{}, err
	}

	endMS := startMS + (rowCount-1)*60_000
	startUTC, err := canonicalcontract.FormatTimestampMillis(startMS)
	if err != nil {
		return DiagnosticSmokeFixture{}, err
	}
	endUTC, err := canonicalcontract.FormatTimestampMillis(endMS)
	if err != nil {
		return DiagnosticSmokeFixture{}, err
	}
	cutoffUTC, err := canonicalcontract.FormatTimestampMillis(endMS + 60_000)
	if err != nil {
		return DiagnosticSmokeFixture{}, err
	}
	earliestAvailableUTC, err := canonicalcontract.FormatTimestampMillis(startMS + 59_999)
	if err != nil {
		return DiagnosticSmokeFixture{}, err
	}
	latestAvailableUTC, err := canonicalcontract.FormatTimestampMillis(endMS + 59_999)
	if err != nil {
		return DiagnosticSmokeFixture{}, err
	}
	archive := historianArchiveIdentity{
		Contract:      canonicalcontract.NewHeader(historianArchiveSchema, canonicalContractVersion, "source_archive"),
		ArchiveID:     "archive.smoke",
		MediaType:     "application/octet-stream",
		RawObjectHash: archiveRawHash,
		RelativePath:  "source/archive.zip",
		SizeBytes:     archiveSize,
	}
	archive.ArtifactHash, err = artifactHash(historianArchiveSchema, "source_archive", archive)
	if err != nil {
		return DiagnosticSmokeFixture{}, err
	}
	dataset := DatasetIdentity{
		Contract:             canonicalcontract.NewHeader(historianDatasetSchemaName, canonicalContractVersion, historianDatasetRole),
		DatasetID:            "dataset.smoke",
		DatasetVersion:       "1",
		InstrumentUniverseID: "universe.smoke",
		Symbols:              []string{"BTCUSDT"},
		StartUTC:             startUTC,
		EndUTC:               endUTC,
		Objects: []DatasetObjectIdentity{{
			RelativePath: objectRelative, Symbol: "BTCUSDT", Interval: "1m", SizeBytes: objectSize,
			SHA256: objectHash, RowCount: rowCount, WindowRowCount: rowCount,
			EarliestEventUTC: startUTC, LatestEventUTC: endUTC, LatestAvailableUTC: latestAvailableUTC,
		}},
	}
	dataset.ArtifactHash, err = artifactHash(historianDatasetSchemaName, historianDatasetRole, dataset)
	if err != nil {
		return DiagnosticSmokeFixture{}, err
	}
	coverageEvidence := historianCoverageEvidence{
		Contract: canonicalcontract.NewHeader("ak.historian.coverage_evidence", canonicalContractVersion, "coverage_evidence"),
		Status:   "PASS", FullWindow: true, RequestedStartUTC: startUTC, RequestedEndUTC: endUTC,
		EarliestEventUTC: startUTC, LatestEventUTC: endUTC, RowCount: rowCount, ExpectedRowCount: rowCount,
		SeriesCount: 1,
	}
	coverageEvidence.ArtifactHash, err = artifactHash("ak.historian.coverage_evidence", "coverage_evidence", coverageEvidence)
	if err != nil {
		return DiagnosticSmokeFixture{}, err
	}
	pit := PITEvidenceIdentity{
		Contract:   canonicalcontract.NewHeader(historianPITSchemaName, canonicalContractVersion, historianPITRole),
		EvidenceID: "pit.smoke", EvidenceVersion: "1", Status: "PASS",
		DatasetID: dataset.DatasetID, DatasetVersion: dataset.DatasetVersion, DatasetHash: dataset.ArtifactHash,
		SourceArchiveHash: archive.ArtifactHash, AvailabilityPolicyHash: availability.ArtifactHash,
		CoveragePolicyHash: coveragePolicy.ArtifactHash, EvaluationCutoffUTC: cutoffUTC,
		EarliestEventUTC: startUTC, LatestEventUTC: endUTC, EarliestAvailableUTC: earliestAvailableUTC,
		LatestAvailableUTC: latestAvailableUTC, FullWindowCoverage: true, AvailabilityDelayNS: int64(time.Minute),
	}
	pit.ArtifactHash, err = artifactHash(historianPITSchemaName, historianPITRole, pit)
	if err != nil {
		return DiagnosticSmokeFixture{}, err
	}
	manifest := historianManifest{
		Contract:             canonicalcontract.NewHeader(historianManifestSchemaName, canonicalContractVersion, "research_identity_manifest"),
		ManifestID:           "manifest.smoke",
		ManifestVersion:      "1",
		DatasetStartUTC:      startUTC,
		DatasetEndUTC:        endUTC,
		PointInTimeCutoffUTC: cutoffUTC,
		Dataset:              dataset,
		SourceArchive:        archive,
		AvailabilityPolicy:   availability,
		AvailabilityPolicySource: historianRawObjectReference{
			LogicalID: availability.PolicyID, MediaType: "application/json", ObjectHash: availabilityRawHash,
			RelativePath: "policies/availability.json", SizeBytes: int64(len(availabilityData)),
		},
		CoveragePolicy: coveragePolicy,
		CoveragePolicySource: historianRawObjectReference{
			LogicalID: coveragePolicy.PolicyID, MediaType: "application/json", ObjectHash: coverageRawHash,
			RelativePath: "policies/coverage.json", SizeBytes: int64(len(coverageData)),
		},
		CoverageEvidence: coverageEvidence,
		PITEvidence:      pit,
	}
	manifest.ArtifactHash, err = artifactHash(historianManifestSchemaName, "research_identity_manifest", manifest)
	if err != nil {
		return DiagnosticSmokeFixture{}, err
	}
	manifestPath := filepath.Join(evidenceRoot, "research_identity_manifest.json")
	if err := writeSmokeCanonicalArtifact(manifestPath, manifest); err != nil {
		return DiagnosticSmokeFixture{}, err
	}

	featureRows := make([]features.Row, rowCount)
	labels := make([]regime.Label, rowCount)
	candles := make([]protocol.Candle, rowCount)
	timestamps := make([]int64, 0, rowCount-1)
	returns := make([]float64, 0, rowCount-1)
	for i := 0; i < rowCount; i++ {
		timestamp := startMS + int64(i)*60_000
		availableAtMS := timestamp + 59_999
		featureRows[i] = features.Row{Market: "futures-um", Symbol: "BTCUSDT", Interval: "1m", EventTimeMS: timestamp, AvailableAtMS: availableAtMS, Close: float64(100 + i), EMA20: 99, EMA50: 98}
		labels[i] = regime.Label{Market: "futures-um", Symbol: "BTCUSDT", Interval: "1m", EventTimeMS: timestamp, AvailableAtMS: availableAtMS, Volatility: "compressed", Trend: "bull_trend", Liquidity: "normal", MarketBeta: "btc_up", Sentiment: "unknown", Composite: "compressed_range"}
		candles[i] = protocol.Candle{Market: "futures-um", Symbol: "BTCUSDT", Interval: "1m", OpenTimeMS: timestamp, CloseTimeMS: timestamp + 59_999, Open: 100, High: 101, Low: 99, Close: 100.5, Volume: 1, QuoteAssetVolume: 100, NumberOfTrades: 1, TakerBuyBaseVolume: 0.5, TakerBuyQuoteVolume: 50}
		if i < rowCount-1 {
			timestamps = append(timestamps, timestamp)
			returns = append(returns, 0.0045)
		}
	}
	featurePath := filepath.Join(evidenceRoot, "features.json")
	regimePath := filepath.Join(evidenceRoot, "regimes.json")
	if err := writeSmokeJSON(featurePath, featureRows); err != nil {
		return DiagnosticSmokeFixture{}, err
	}
	if err := writeSmokeJSON(regimePath, labels); err != nil {
		return DiagnosticSmokeFixture{}, err
	}
	config, err := ResolveConfiguration(ConfigurationContext{Symbol: "BTCUSDT", Market: "futures-um", Interval: "1m", EvaluationStartMS: startMS, EvaluationEndMS: endMS}, nil)
	if err != nil {
		return DiagnosticSmokeFixture{}, err
	}
	config.SeriesHorizonMinutes = 1
	request := DerivationRequest{
		RepositoryRoot: repositoryRoot, CandidateFamily: "CompressionBreakout", CandidateSide: "LONG", Configuration: config,
		HistorianManifestPath: manifestPath, DatasetRoot: datasetRoot, FeatureArtifactPath: featurePath, RegimeArtifactPath: regimePath,
		ConsumedDatasetPaths: []string{objectPath}, FeatureRows: featureRows, RegimeLabels: labels, Candles: candles,
		EvaluationEventTimestamps: timestamps, Returns: returns, Timestamps: timestamps,
	}
	state := RepositoryState{Root: repositoryRoot, RepositoryID: "fixture.local/ak-engine", CommitSHA: strings.Repeat("a", 40), TreeSHA: strings.Repeat("b", 40), BuildVersion: "fixture-only", GoVersion: "go1.25.6", GoOS: "linux", GoARCH: "amd64", Compiler: "gc", CGOEnabled: "0"}
	failed = false
	return DiagnosticSmokeFixture{
		Request: request,
		Deriver: NewDeriverWithProvider(smokeRepositoryProvider{state: state}, func() time.Time { return time.Date(2024, time.February, 1, 0, 0, 0, 0, time.UTC) }),
		Cleanup: func() { _ = os.RemoveAll(fixtureRoot) },
	}, nil
}

func writeSmokeImplementationFiles(root string) error {
	paths := map[string]struct{}{}
	paths["internal/app/evaluate_candidate_deep.go"] = struct{}{}
	paths["internal/executionseries/spec.go"] = struct{}{}
	for _, file := range featureImplementationFiles {
		paths[file.Path] = struct{}{}
	}
	for _, file := range regimeImplementationFiles {
		paths[file.Path] = struct{}{}
	}
	for path := range paths {
		if err := writeSmokeFile(filepath.Join(root, filepath.FromSlash(path)), []byte("package fixture\n"+path+"\n")); err != nil {
			return err
		}
	}
	return nil
}

func writeSmokeFile(path string, data []byte) error {
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return err
	}
	if err := os.WriteFile(path, data, 0644); err != nil {
		return fmt.Errorf("write smoke fixture %s: %w", path, err)
	}
	return nil
}

func writeSmokeJSON(path string, value any) error {
	data, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		return err
	}
	return writeSmokeFile(path, data)
}

func writeSmokeCanonicalArtifact(path string, value any) error {
	data, err := canonicalcontract.CanonicalizeValue(value)
	if err != nil {
		return err
	}
	if _, err := canonicalcontract.ValidateArtifact(data, true); err != nil {
		return err
	}
	return writeSmokeFile(path, data)
}

func writeSmokeParquet(path string, startMS int64, count int) error {
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return err
	}
	fw, err := local.NewLocalFileWriter(path)
	if err != nil {
		return err
	}
	pw, err := writer.NewParquetWriter(fw, new(data.ParquetCandleWithMS), 1)
	if err != nil {
		_ = fw.Close()
		return err
	}
	market, symbol, interval := "futures-um", "BTCUSDT", "1m"
	for i := 0; i < count; i++ {
		openTime := startMS + int64(i)*60_000
		closeTime := openTime + 59_999
		open, high, low, close, volume := 100.0, 101.0, 99.0, 100.5, 1.0
		quote, trades, takerBase, takerQuote := 100.0, int64(1), 0.5, 50.0
		row := data.ParquetCandleWithMS{Market: &market, Symbol: &symbol, Interval: &interval, OpenTimeMS: &openTime, CloseTimeMS: &closeTime, Open: &open, High: &high, Low: &low, Close: &close, Volume: &volume, QuoteAssetVolume: &quote, NumberOfTrades: &trades, TakerBuyBaseVolume: &takerBase, TakerBuyQuoteVolume: &takerQuote}
		if err := pw.Write(row); err != nil {
			return err
		}
	}
	if err := pw.WriteStop(); err != nil {
		return err
	}
	return fw.Close()
}
