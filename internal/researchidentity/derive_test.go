package researchidentity

import (
	"encoding/json"
	"math"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/david22573/ak-engine/internal/canonicalcontract"
	"github.com/david22573/ak-engine/internal/executionseries"
	"github.com/david22573/ak-engine/internal/features"
)

type fixedRepositoryProvider struct {
	state RepositoryState
	err   error
}

func (p fixedRepositoryProvider) Resolve(string) (RepositoryState, error) { return p.state, p.err }

func TestDeriverProducesCompleteStableIdentity(t *testing.T) {
	request, deriver := completeIdentityFixture(t)
	first, err := deriver.Derive(request)
	if err != nil {
		t.Fatal(err)
	}
	second, err := deriver.Derive(request)
	if err != nil {
		t.Fatal(err)
	}
	if first.Status != StatusComplete || first.Identity == nil || first.Identity.IdentityHash == "" {
		t.Fatalf("identity is incomplete: %#v", first)
	}
	if first.Identity.IdentityHash != second.Identity.IdentityHash {
		t.Fatal("same inputs did not produce stable top-level identity")
	}
	if first.Identity.Series.ObservationCount != len(request.Returns) || first.Identity.ConsumedInput.Hash == "" {
		t.Fatalf("series/consumed identity missing: %#v", first.Identity)
	}
	if first.Identity.Series.SeriesGenerationVersion != executionseries.GenerationVersion {
		t.Fatalf("series generation = %q, want %q", first.Identity.Series.SeriesGenerationVersion, executionseries.GenerationVersion)
	}
}

func TestBuildRowsTruthfulAvailabilityPassesCanonicalIdentity(t *testing.T) {
	request, deriver := completeIdentityFixture(t)
	rows, err := features.BuildRows(request.Candles, features.BuildOptions{
		Market: request.Configuration.Market, Symbol: request.Configuration.Symbol,
		Interval: request.Configuration.Interval,
	})
	if err != nil {
		t.Fatalf("BuildRows() error = %v", err)
	}
	if len(rows) != len(request.FeatureRows) {
		t.Fatalf("BuildRows() rows = %d, want %d", len(rows), len(request.FeatureRows))
	}
	request.FeatureRows = rows
	for i := range request.RegimeLabels {
		request.RegimeLabels[i].EventTimeMS = rows[i].EventTimeMS
		request.RegimeLabels[i].AvailableAtMS = rows[i].AvailableAtMS
	}
	writeFixtureJSON(t, request.FeatureArtifactPath, request.FeatureRows)
	writeFixtureJSON(t, request.RegimeArtifactPath, request.RegimeLabels)

	assessment, err := deriver.Derive(request)
	if err != nil {
		t.Fatalf("canonical identity rejected BuildRows output: %v", err)
	}
	if assessment.Status != StatusComplete || assessment.Identity == nil {
		t.Fatalf("canonical identity incomplete for BuildRows output: %#v", assessment)
	}
	for i, row := range rows {
		if row.AvailableAtMS != request.Candles[i].CloseTimeMS || row.AvailableAtMS <= row.EventTimeMS {
			t.Fatalf("row %d is not truthfully close-available: %#v", i, row)
		}
	}
}

func TestCrossIdentityRejectsPITPolicyAndArchiveConflict(t *testing.T) {
	request, deriver := completeIdentityFixture(t)
	assessment, err := deriver.Derive(request)
	if err != nil {
		t.Fatal(err)
	}
	identity := *assessment.Identity
	identity.PIT.PITPolicyHash = "sha256:" + strings.Repeat("c", 64)
	if err := validateCrossIdentity(identity, request); err == nil {
		t.Fatal("PIT policy disagreement did not fail cross-identity validation")
	}
	identity = *assessment.Identity
	identity.PIT.SourceArchiveHash = "sha256:" + strings.Repeat("d", 64)
	if err := validateCrossIdentity(identity, request); err == nil {
		t.Fatal("PIT source-archive disagreement did not fail cross-identity validation")
	}
}

func TestDirtyAndInvalidEngineSourceFailClosed(t *testing.T) {
	request, deriver := completeIdentityFixture(t)
	state := deriver.sourceProvider.(fixedRepositoryProvider).state
	state.Dirty = true
	dirty := NewDeriverWithProvider(fixedRepositoryProvider{state: state}, deriver.now)
	assessment, err := dirty.Derive(request)
	if err == nil || assessment.Status != StatusDirtyEngineSource || assessment.Identity != nil {
		t.Fatalf("dirty source did not fail closed: %#v %v", assessment, err)
	}

	state.Dirty = false
	state.BinaryModified = true
	binaryDirty := NewDeriverWithProvider(fixedRepositoryProvider{state: state}, deriver.now)
	assessment, err = binaryDirty.Derive(request)
	if err == nil || assessment.Status != StatusDirtyEngineSource {
		t.Fatalf("modified executing binary did not fail dirty: %#v %v", assessment, err)
	}

	state.BinaryModified = false
	state.BinaryRevision = strings.Repeat("c", 40)
	binaryMismatch := NewDeriverWithProvider(fixedRepositoryProvider{state: state}, deriver.now)
	assessment, err = binaryMismatch.Derive(request)
	if err == nil || assessment.Status != StatusValidationFailed {
		t.Fatalf("binary/repository revision mismatch did not fail: %#v %v", assessment, err)
	}

	state.BinaryRevision = ""
	state.CommitSHA = "not-a-commit"
	invalid := NewDeriverWithProvider(fixedRepositoryProvider{state: state}, deriver.now)
	assessment, err = invalid.Derive(request)
	if err == nil || assessment.Status != StatusValidationFailed {
		t.Fatalf("invalid source commit did not fail: %#v %v", assessment, err)
	}

	state.CommitSHA = strings.Repeat("a", 40)
	state.CGOEnabled = "unknown"
	unknownCGO := NewDeriverWithProvider(fixedRepositoryProvider{state: state}, deriver.now)
	assessment, err = unknownCGO.Derive(request)
	if err == nil || assessment.Status != StatusValidationFailed {
		t.Fatalf("unknown CGO setting did not fail: %#v %v", assessment, err)
	}
}

func TestBuildTagChangesAffectAndMustMatchSourceIdentity(t *testing.T) {
	request, deriver := completeIdentityFixture(t)
	state := deriver.sourceProvider.(fixedRepositoryProvider).state
	state.BuildTags = []string{"identity_test"}
	mismatched := NewDeriverWithProvider(fixedRepositoryProvider{state: state}, deriver.now)
	assessment, err := mismatched.Derive(request)
	if err == nil || assessment.Status != StatusValidationFailed {
		t.Fatalf("build-tag mismatch did not fail: %#v %v", assessment, err)
	}
	request.Configuration.BuildTags = []string{"identity_test"}
	assessment, err = mismatched.Derive(request)
	if err != nil || assessment.Status != StatusComplete {
		t.Fatalf("matching build tags did not complete identity: %#v %v", assessment, err)
	}
}

func TestHistorianTamperAndMissingIdentityFailClosed(t *testing.T) {
	t.Run("dataset hash", func(t *testing.T) {
		request, deriver := completeIdentityFixture(t)
		manifest := readHistorianFixture(t, request.HistorianManifestPath)
		manifest.Dataset.ArtifactHash = "sha256:" + strings.Repeat("a", 64)
		writeFixtureCanonical(t, request.HistorianManifestPath, manifest)
		assessment, err := deriver.Derive(request)
		if err == nil || assessment.Status != StatusConflict {
			t.Fatalf("manifest tamper: %#v %v", assessment, err)
		}
	})
	t.Run("manifest self hash", func(t *testing.T) {
		request, deriver := completeIdentityFixture(t)
		manifest := readHistorianFixture(t, request.HistorianManifestPath)
		manifest.ArtifactHash = "sha256:" + strings.Repeat("a", 64)
		writeFixtureCanonical(t, request.HistorianManifestPath, manifest)
		assessment, err := deriver.Derive(request)
		if err == nil || assessment.Status != StatusConflict {
			t.Fatalf("embedded manifest hash tamper: %#v %v", assessment, err)
		}
	})
	t.Run("raw manifest bytes", func(t *testing.T) {
		request, deriver := completeIdentityFixture(t)
		data, err := os.ReadFile(request.HistorianManifestPath)
		if err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(request.HistorianManifestPath, append(data, '\n'), 0644); err != nil {
			t.Fatal(err)
		}
		assessment, err := deriver.Derive(request)
		if err == nil || assessment.Identity != nil || assessment.Status != StatusDatasetIncomplete {
			t.Fatalf("non-canonical manifest bytes passed: %#v %v", assessment, err)
		}
	})
	t.Run("dataset object", func(t *testing.T) {
		request, deriver := completeIdentityFixture(t)
		if err := os.WriteFile(request.ConsumedDatasetPaths[0], []byte("changed object"), 0644); err != nil {
			t.Fatal(err)
		}
		assessment, err := deriver.Derive(request)
		if err == nil || assessment.Status != StatusConflict {
			t.Fatalf("object tamper: %#v %v", assessment, err)
		}
	})
	t.Run("source archive", func(t *testing.T) {
		request, deriver := completeIdentityFixture(t)
		manifest := readHistorianFixture(t, request.HistorianManifestPath)
		archive := filepath.Join(filepath.Dir(request.HistorianManifestPath), filepath.FromSlash(manifest.SourceArchive.RelativePath))
		if err := os.WriteFile(archive, []byte("changed archive"), 0644); err != nil {
			t.Fatal(err)
		}
		assessment, err := deriver.Derive(request)
		if err == nil || assessment.Status != StatusConflict {
			t.Fatalf("archive tamper: %#v %v", assessment, err)
		}
	})
	t.Run("availability policy bytes", func(t *testing.T) {
		request, deriver := completeIdentityFixture(t)
		manifest := readHistorianFixture(t, request.HistorianManifestPath)
		policyPath := filepath.Join(filepath.Dir(request.HistorianManifestPath), filepath.FromSlash(manifest.AvailabilityPolicySource.RelativePath))
		policy := manifest.AvailabilityPolicy
		policy.AvailabilityDelayNS = 0
		policy.ArtifactHash = ""
		var err error
		policy.ArtifactHash, err = artifactHash(historianAvailabilitySchema, "availability_policy", policy)
		if err != nil {
			t.Fatal(err)
		}
		writeFixtureCanonical(t, policyPath, policy)
		assessment, err := deriver.Derive(request)
		if err == nil || assessment.Status != StatusConflict {
			t.Fatalf("availability policy tamper: %#v %v", assessment, err)
		}
	})
	t.Run("dataset version", func(t *testing.T) {
		request, deriver := completeIdentityFixture(t)
		manifest := readHistorianFixture(t, request.HistorianManifestPath)
		manifest.Dataset.DatasetVersion = ""
		writeFixtureCanonical(t, request.HistorianManifestPath, manifest)
		assessment, err := deriver.Derive(request)
		if err == nil || assessment.Status != StatusDatasetIncomplete {
			t.Fatalf("missing version: %#v %v", assessment, err)
		}
	})
	t.Run("PIT hash", func(t *testing.T) {
		request, deriver := completeIdentityFixture(t)
		manifest := readHistorianFixture(t, request.HistorianManifestPath)
		manifest.PITEvidence.ArtifactHash = ""
		writeFixtureCanonical(t, request.HistorianManifestPath, manifest)
		assessment, err := deriver.Derive(request)
		if err == nil || assessment.Status != StatusDatasetIncomplete {
			t.Fatalf("missing PIT: %#v %v", assessment, err)
		}
	})
}

func TestEngineConfigurationAndHistorianScopeConflictFailsAsConflict(t *testing.T) {
	request, deriver := completeIdentityFixture(t)
	request.Configuration.Symbol = "ETHUSDT"
	assessment, err := deriver.Derive(request)
	if err == nil || assessment.Status != StatusConflict || assessment.Identity != nil {
		t.Fatalf("Engine/Historian symbol conflict was not distinguished: %#v %v", assessment, err)
	}

	request, deriver = completeIdentityFixture(t)
	request.Configuration.EvaluationEndMS += 60_000
	assessment, err = deriver.Derive(request)
	if err == nil || assessment.Status != StatusConflict || assessment.Identity != nil {
		t.Fatalf("Engine/Historian window conflict was not distinguished: %#v %v", assessment, err)
	}
}

func TestFeatureRegimeConsumedAndSeriesChangesAreBound(t *testing.T) {
	request, deriver := completeIdentityFixture(t)
	base, err := deriver.Derive(request)
	if err != nil {
		t.Fatal(err)
	}

	request.FeatureRows[1].Close++
	writeFixtureJSON(t, request.FeatureArtifactPath, request.FeatureRows)
	changed, err := deriver.Derive(request)
	if err != nil {
		t.Fatal(err)
	}
	if changed.Identity.Feature.OutputArtifactHash == base.Identity.Feature.OutputArtifactHash || changed.Identity.ConsumedInput.Hash == base.Identity.ConsumedInput.Hash || changed.Identity.IdentityHash == base.Identity.IdentityHash {
		t.Fatal("feature value change did not change feature/consumed/top identity")
	}

	request, deriver = completeIdentityFixture(t)
	base, _ = deriver.Derive(request)
	request.RegimeLabels[1].Trend = "bear_trend"
	writeFixtureJSON(t, request.RegimeArtifactPath, request.RegimeLabels)
	changed, err = deriver.Derive(request)
	if err != nil {
		t.Fatal(err)
	}
	if changed.Identity.Regime.OutputArtifactHash == base.Identity.Regime.OutputArtifactHash || changed.Identity.ConsumedInput.Hash == base.Identity.ConsumedInput.Hash {
		t.Fatal("regime change did not change bound identities")
	}

	request, deriver = completeIdentityFixture(t)
	request.Returns[1] += 0.01
	assessment, err := deriver.Derive(request)
	if err == nil || assessment.Status != StatusConsumedIncomplete || assessment.Identity != nil {
		t.Fatalf("caller-invented return passed canonical execution recomputation: %#v %v", assessment, err)
	}

	request, deriver = completeIdentityFixture(t)
	request.Candles[1].Close++
	assessment, err = deriver.Derive(request)
	if err == nil || assessment.Status != StatusConsumedIncomplete {
		t.Fatalf("claimed consumed values without matching parquet rows did not fail: %#v %v", assessment, err)
	}
}

func TestSeriesValidationRejectsCardinalityAndTimestampDefects(t *testing.T) {
	validReturns := []float64{0.1, -0.1, 0.2}
	validTimes := []int64{1000, 2000, 3000}
	first, err := deriveSeriesIdentity(validReturns, validTimes, 1000, 3000)
	if err != nil {
		t.Fatal(err)
	}
	second, err := deriveSeriesIdentity(validReturns, validTimes, 1000, 3000)
	if err != nil || first.SeriesHash != second.SeriesHash {
		t.Fatal("stable series identity failed")
	}
	tests := []struct {
		name    string
		returns []float64
		times   []int64
	}{
		{name: "empty", returns: nil, times: nil},
		{name: "count mismatch", returns: validReturns[:2], times: validTimes},
		{name: "NaN", returns: []float64{0.1, math.NaN(), 0.2}, times: validTimes},
		{name: "infinity", returns: []float64{0.1, math.Inf(1), 0.2}, times: validTimes},
		{name: "duplicate timestamp", returns: validReturns, times: []int64{1000, 1000, 3000}},
		{name: "out of window", returns: validReturns, times: []int64{0, 2000, 3000}},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if _, err := deriveSeriesIdentity(tc.returns, tc.times, 1000, 3000); err == nil {
				t.Fatal("defective series passed")
			}
		})
	}
}

func completeIdentityFixture(t *testing.T) (DerivationRequest, *Deriver) {
	t.Helper()
	fixture, err := BuildDiagnosticSmokeFixture(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(fixture.Cleanup)
	state := fixture.Deriver.sourceProvider.(smokeRepositoryProvider).state
	deriver := NewDeriverWithProvider(fixedRepositoryProvider{state: state}, fixture.Deriver.now)
	return fixture.Request, deriver
}

func readHistorianFixture(t *testing.T, path string) historianManifest {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	var manifest historianManifest
	if err := json.Unmarshal(data, &manifest); err != nil {
		t.Fatal(err)
	}
	return manifest
}

func writeFixtureJSON(t *testing.T, path string, value any) {
	t.Helper()
	data, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, data, 0644); err != nil {
		t.Fatal(err)
	}
}

func writeFixtureCanonical(t *testing.T, path string, value any) {
	t.Helper()
	data, err := canonicalcontract.CanonicalizeValue(value)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, data, 0644); err != nil {
		t.Fatal(err)
	}
}

func rehashHistorianFixture(t *testing.T, manifest *historianManifest) {
	t.Helper()
	assign := func(target *string, schema, role string, value any) {
		hash, err := artifactHash(schema, role, value)
		if err != nil {
			t.Fatal(err)
		}
		*target = hash
	}
	assign(&manifest.SourceArchive.ArtifactHash, historianArchiveSchema, "source_archive", manifest.SourceArchive)
	assign(&manifest.AvailabilityPolicy.ArtifactHash, historianAvailabilitySchema, "availability_policy", manifest.AvailabilityPolicy)
	assign(&manifest.CoveragePolicy.ArtifactHash, historianCoverageSchema, "coverage_policy", manifest.CoveragePolicy)
	assign(&manifest.Dataset.ArtifactHash, historianDatasetSchemaName, historianDatasetRole, manifest.Dataset)
	assign(&manifest.CoverageEvidence.ArtifactHash, "ak.historian.coverage_evidence", "coverage_evidence", manifest.CoverageEvidence)
	manifest.PITEvidence.DatasetHash = manifest.Dataset.ArtifactHash
	manifest.PITEvidence.SourceArchiveHash = manifest.SourceArchive.ArtifactHash
	manifest.PITEvidence.AvailabilityPolicyHash = manifest.AvailabilityPolicy.ArtifactHash
	manifest.PITEvidence.CoveragePolicyHash = manifest.CoveragePolicy.ArtifactHash
	assign(&manifest.PITEvidence.ArtifactHash, historianPITSchemaName, historianPITRole, manifest.PITEvidence)
	assign(&manifest.ArtifactHash, historianManifestSchemaName, "research_identity_manifest", *manifest)
}
