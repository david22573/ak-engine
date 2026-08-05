package researchidentity

import (
	"encoding/json"
	"os"
	"strings"
	"testing"
)

func TestConfigurationThresholdEnvironmentAndInvariantValidation(t *testing.T) {
	context := ConfigurationContext{Symbol: "BTCUSDT", Market: "futures-um", Interval: "1m", EvaluationStartMS: 1_704_067_200_000, EvaluationEndMS: 1_704_067_260_000}
	base, err := ResolveConfiguration(context, nil)
	if err != nil {
		t.Fatal(err)
	}
	changed, err := ResolveConfiguration(context, []byte(`{"gate_thresholds":{"minimum_events":301}}`))
	if err != nil {
		t.Fatal(err)
	}
	baseIdentity, _ := ConfigurationHash(base)
	changedIdentity, _ := ConfigurationHash(changed)
	if changed.GateThresholds.MinimumEvents != 301 || changed.GateThresholds.MinimumH2PF != base.GateThresholds.MinimumH2PF || baseIdentity.Hash == changedIdentity.Hash {
		t.Fatal("partial gate override did not preserve defaults and change identity")
	}

	withTags, err := ResolveConfiguration(ConfigurationContext{Symbol: context.Symbol, Market: context.Market, Interval: context.Interval, EvaluationStartMS: context.EvaluationStartMS, EvaluationEndMS: context.EvaluationEndMS, BuildTags: []string{"z", "a"}}, nil)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Join(withTags.BuildTags, ",") != "a,z" {
		t.Fatalf("build tags are not normalized: %v", withTags.BuildTags)
	}
	withTagsIdentity, _ := ConfigurationHash(withTags)
	if baseIdentity.Hash == withTagsIdentity.Hash {
		t.Fatal("behavioral build tags did not change configuration identity")
	}

	for name, raw := range map[string]string{
		"immutable feature version":  `{"feature_set_version":"invented"}`,
		"immutable filtering policy": `{"filtering_policy":"invented"}`,
		"nested duplicate":           `{"gate_thresholds":{"minimum_events":300,"minimum_events":301}}`,
		"empty mandatory array":      `{"brackets":[]}`,
		"duplicate horizon":          `{"forward_horizons_minutes":[5,5]}`,
		"nonfinite":                  `{"series_cost_bps":1e999}`,
	} {
		t.Run(name, func(t *testing.T) {
			if _, err := ResolveConfiguration(context, []byte(raw)); err == nil {
				t.Fatal("invalid configuration passed")
			}
		})
	}

	for name, mutate := range map[string]func(*ResolvedResearchConfiguration){
		"invented feature identity": func(config *ResolvedResearchConfiguration) { config.FeatureSetVersion = "invented" },
		"invented regime identity":  func(config *ResolvedResearchConfiguration) { config.RegimeVersion = "invented" },
	} {
		t.Run(name, func(t *testing.T) {
			changed := base
			mutate(&changed)
			if _, err := ConfigurationHash(changed); err == nil {
				t.Fatal("unsupported compiled implementation identity passed")
			}
		})
	}
}

func TestSourcePlatformChangesAreBound(t *testing.T) {
	request, deriver := completeIdentityFixture(t)
	base, err := deriver.Derive(request)
	if err != nil {
		t.Fatal(err)
	}
	state := deriver.sourceProvider.(fixedRepositoryProvider).state
	state.GoARCH = "arm64"
	changed, err := NewDeriverWithProvider(fixedRepositoryProvider{state: state}, deriver.now).Derive(request)
	if err != nil {
		t.Fatal(err)
	}
	if base.Identity.EngineSource.GoARCH == changed.Identity.EngineSource.GoARCH || base.Identity.IdentityHash == changed.Identity.IdentityHash {
		t.Fatal("GOARCH change was not source-bound")
	}
	state.GoARCH = ""
	assessment, err := NewDeriverWithProvider(fixedRepositoryProvider{state: state}, deriver.now).Derive(request)
	if err == nil || assessment.Status != StatusValidationFailed {
		t.Fatalf("missing platform identity passed: %#v %v", assessment, err)
	}
}

func TestHistorianWindowPITCoverageAndStructureFailures(t *testing.T) {
	tests := []struct {
		name       string
		wantStatus IdentityStatus
		mutate     func(*historianManifest)
	}{
		{name: "archive hash missing", wantStatus: StatusDatasetIncomplete, mutate: func(m *historianManifest) { m.SourceArchive.RawObjectHash = "" }},
		{name: "invalid window", wantStatus: StatusDatasetIncomplete, mutate: func(m *historianManifest) {
			m.DatasetStartUTC = m.DatasetEndUTC
			m.Dataset.StartUTC = m.DatasetEndUTC
			m.CoverageEvidence.RequestedStartUTC = m.DatasetEndUTC
		}},
		{name: "coverage window mismatch", wantStatus: StatusConflict, mutate: func(m *historianManifest) { m.CoverageEvidence.RequestedStartUTC = m.DatasetEndUTC }},
		{name: "partial coverage", wantStatus: StatusDatasetIncomplete, mutate: func(m *historianManifest) { m.CoverageEvidence.FullWindow = false }},
		{name: "coverage gap", wantStatus: StatusDatasetIncomplete, mutate: func(m *historianManifest) { m.CoverageEvidence.GapCount = 1 }},
		{name: "coverage duplicate", wantStatus: StatusDatasetIncomplete, mutate: func(m *historianManifest) { m.CoverageEvidence.DuplicateTimestampCount = 1 }},
		{name: "coverage out of order", wantStatus: StatusDatasetIncomplete, mutate: func(m *historianManifest) { m.CoverageEvidence.OutOfOrderCount = 1 }},
		{name: "unknown PIT", wantStatus: StatusDatasetIncomplete, mutate: func(m *historianManifest) { m.PITEvidence.Status = "UNKNOWN" }},
		{name: "nonpass PIT", wantStatus: StatusDatasetIncomplete, mutate: func(m *historianManifest) { m.PITEvidence.Status = "FAIL" }},
		{name: "late PIT", wantStatus: StatusPITIncomplete, mutate: func(m *historianManifest) { m.PITEvidence.LatestAvailableUTC = "2024-01-01T00:41:00.000000000Z" }},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			request, deriver := completeIdentityFixture(t)
			manifest := readHistorianFixture(t, request.HistorianManifestPath)
			tc.mutate(&manifest)
			rehashHistorianFixture(t, &manifest)
			writeFixtureCanonical(t, request.HistorianManifestPath, manifest)
			assessment, err := deriver.Derive(request)
			if err == nil || assessment.Status != tc.wantStatus || assessment.Identity != nil {
				t.Fatalf("failure status=%s, want %s, err=%v", assessment.Status, tc.wantStatus, err)
			}
		})
	}

	t.Run("unknown manifest field", func(t *testing.T) {
		request, deriver := completeIdentityFixture(t)
		raw, _ := os.ReadFile(request.HistorianManifestPath)
		var object map[string]any
		if err := json.Unmarshal(raw, &object); err != nil {
			t.Fatal(err)
		}
		object["invented_identity"] = "sha256:" + strings.Repeat("a", 64)
		writeFixtureJSON(t, request.HistorianManifestPath, object)
		assessment, err := deriver.Derive(request)
		if err == nil || assessment.Status != StatusDatasetIncomplete {
			t.Fatalf("unknown manifest field passed: %#v %v", assessment, err)
		}
	})
	t.Run("duplicate manifest field", func(t *testing.T) {
		request, deriver := completeIdentityFixture(t)
		raw, _ := os.ReadFile(request.HistorianManifestPath)
		duplicate := strings.Replace(string(raw), `"manifest_id":`, `"manifest_id":"duplicate","manifest_id":`, 1)
		if err := os.WriteFile(request.HistorianManifestPath, []byte(duplicate), 0644); err != nil {
			t.Fatal(err)
		}
		assessment, err := deriver.Derive(request)
		if err == nil || assessment.Status != StatusDatasetIncomplete {
			t.Fatalf("duplicate manifest field passed: %#v %v", assessment, err)
		}
	})
}

func TestFeatureRegimeAndConsumedEvidenceMustMatchActualArtifacts(t *testing.T) {
	t.Run("feature missing", func(t *testing.T) {
		request, deriver := completeIdentityFixture(t)
		request.FeatureArtifactPath = ""
		assessment, err := deriver.Derive(request)
		if err == nil || assessment.Status != StatusFeatureIncomplete {
			t.Fatalf("missing feature passed: %#v %v", assessment, err)
		}
	})
	t.Run("feature rows differ from artifact", func(t *testing.T) {
		request, deriver := completeIdentityFixture(t)
		request.FeatureRows[1].Close++
		assessment, err := deriver.Derive(request)
		if err == nil || assessment.Status != StatusFeatureIncomplete {
			t.Fatalf("feature mismatch passed: %#v %v", assessment, err)
		}
	})
	t.Run("feature unavailable at event", func(t *testing.T) {
		request, deriver := completeIdentityFixture(t)
		request.FeatureRows[1].AvailableAtMS = request.FeatureRows[1].EventTimeMS + 1
		writeFixtureJSON(t, request.FeatureArtifactPath, request.FeatureRows)
		assessment, err := deriver.Derive(request)
		if err == nil || assessment.Status != StatusFeatureIncomplete {
			t.Fatalf("late feature passed: %#v %v", assessment, err)
		}
	})
	t.Run("regime missing", func(t *testing.T) {
		request, deriver := completeIdentityFixture(t)
		request.RegimeArtifactPath = ""
		request.RegimeLabels = nil
		assessment, err := deriver.Derive(request)
		if err == nil || assessment.Status != StatusRegimeIncomplete {
			t.Fatalf("missing regime passed: %#v %v", assessment, err)
		}
	})
	t.Run("regime rows differ from artifact", func(t *testing.T) {
		request, deriver := completeIdentityFixture(t)
		request.RegimeLabels[1].Trend = "changed"
		assessment, err := deriver.Derive(request)
		if err == nil || assessment.Status != StatusRegimeIncomplete {
			t.Fatalf("regime mismatch passed: %#v %v", assessment, err)
		}
	})
	t.Run("regime unavailable at event", func(t *testing.T) {
		request, deriver := completeIdentityFixture(t)
		request.RegimeLabels[1].AvailableAtMS = request.RegimeLabels[1].EventTimeMS + 1
		writeFixtureJSON(t, request.RegimeArtifactPath, request.RegimeLabels)
		assessment, err := deriver.Derive(request)
		if err == nil || assessment.Status != StatusRegimeIncomplete {
			t.Fatalf("late regime passed: %#v %v", assessment, err)
		}
	})
	t.Run("event timestamp lacks consumed row", func(t *testing.T) {
		request, deriver := completeIdentityFixture(t)
		request.EvaluationEventTimestamps[1]++
		request.Timestamps[1]++
		assessment, err := deriver.Derive(request)
		if err == nil || assessment.Status != StatusConsumedIncomplete {
			t.Fatalf("event mismatch passed: %#v %v", assessment, err)
		}
	})
	t.Run("meaningful row reorder", func(t *testing.T) {
		request, deriver := completeIdentityFixture(t)
		request.FeatureRows[0], request.FeatureRows[1] = request.FeatureRows[1], request.FeatureRows[0]
		writeFixtureJSON(t, request.FeatureArtifactPath, request.FeatureRows)
		assessment, err := deriver.Derive(request)
		if err == nil || assessment.Status != StatusFeatureIncomplete {
			t.Fatalf("row reorder passed: %#v %v", assessment, err)
		}
	})
	t.Run("symbol ordering", func(t *testing.T) {
		request, deriver := completeIdentityFixture(t)
		manifest := readHistorianFixture(t, request.HistorianManifestPath)
		manifest.Dataset.Symbols = []string{"ZZZUSDT", "BTCUSDT"}
		rehashHistorianFixture(t, &manifest)
		writeFixtureCanonical(t, request.HistorianManifestPath, manifest)
		assessment, err := deriver.Derive(request)
		if err == nil || assessment.Status != StatusDatasetIncomplete {
			t.Fatalf("unsorted symbols passed: %#v %v", assessment, err)
		}
	})
	t.Run("unsupported filtering policy", func(t *testing.T) {
		request, deriver := completeIdentityFixture(t)
		request.Configuration.FilteringPolicy = "INVENTED_FILTER"
		assessment, err := deriver.Derive(request)
		if err == nil || assessment.Status != StatusConfigurationMissing {
			t.Fatalf("invented filter passed: %#v %v", assessment, err)
		}
	})
}

func TestExplicitNoRegimeRegistrationCompletesWithoutRegimeEvidence(t *testing.T) {
	request, deriver := completeIdentityFixture(t)
	registry, err := NewRegistry("registry.no-regime", "1", []Registration{{
		CandidateID: "candidate.no-regime.long", CandidateVersion: "1", CandidateType: "deep_research_strategy",
		Family: "CompressionBreakout", Side: "LONG", ImplementationLocator: "path:internal/app/evaluate_candidate_deep.go#fixture",
		ImplementationFiles: []ImplementationFile{{Path: "internal/app/evaluate_candidate_deep.go", InclusionReason: "fixture"}}, UsesRegimes: false,
	}})
	if err != nil {
		t.Fatal(err)
	}
	request.CandidateFamily = "CompressionBreakout"
	request.RegimeArtifactPath = ""
	request.RegimeLabels = nil
	provider := deriver.sourceProvider.(fixedRepositoryProvider)
	assessment, err := NewDeriverWithDependencies(registry, provider, deriver.now).Derive(request)
	if err != nil || assessment.Status != StatusComplete || assessment.Identity == nil || assessment.Identity.Regime != nil {
		t.Fatalf("explicit no-regime registration failed: %#v %v", assessment, err)
	}
}
