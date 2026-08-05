package researchidentity

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestDefaultRegistryDerivesStableCanonicalIdentity(t *testing.T) {
	root := implementationRepositoryFixture(t)
	registry, err := DefaultRegistry()
	if err != nil {
		t.Fatal(err)
	}
	canonical, err := registry.Resolve(root, "CompressionBreakout", "LONG")
	if err != nil {
		t.Fatal(err)
	}
	alias, err := registry.Resolve(root, "compression_breakout", "long")
	if err != nil {
		t.Fatal(err)
	}
	if canonical.CandidateID != alias.CandidateID || canonical.RegistrationRecordHash != alias.RegistrationRecordHash || canonical.Implementation.ImplementationHash != alias.Implementation.ImplementationHash {
		t.Fatalf("alias did not resolve to canonical identity")
	}
	registration, err := registry.Lookup("compression_breakout", "long")
	if err != nil || registration.Family != "CompressionBreakout" || registration.Side != "LONG" {
		t.Fatalf("alias did not resolve to canonical evaluation rule: %#v %v", registration, err)
	}
	if canonical.CandidateVersion == "" || canonical.RegistryVersion == "" || len(canonical.Implementation.Files) != 1 {
		t.Fatalf("candidate identity is incomplete: %#v", canonical)
	}
	if _, err := registry.Resolve(root, "not-registered", "LONG"); err == nil {
		t.Fatal("unknown candidate did not fail")
	}
}

func TestRegistryRejectsDuplicateRegistrationAndDerivesVersionFromRecord(t *testing.T) {
	base := Registration{
		CandidateID: "candidate.one", CandidateVersion: "7", CandidateType: "test", Family: "One", Side: "LONG",
		ImplementationLocator: "path:one.go", ImplementationFiles: []ImplementationFile{{Path: "one.go", InclusionReason: "test"}},
	}
	if _, err := NewRegistry("registry", "1", []Registration{base, base}); err == nil || !strings.Contains(err.Error(), "duplicate") {
		t.Fatalf("duplicate registration did not fail: %v", err)
	}
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "one.go"), []byte("package one\n"), 0644); err != nil {
		t.Fatal(err)
	}
	registry, err := NewRegistry("registry", "1", []Registration{base})
	if err != nil {
		t.Fatal(err)
	}
	identity, err := registry.Resolve(root, "One", "LONG")
	if err != nil {
		t.Fatal(err)
	}
	if identity.CandidateVersion != "7" {
		t.Fatalf("candidate version was not derived from registration: %q", identity.CandidateVersion)
	}
}

func TestImplementationInventoryIsStableAndByteSensitive(t *testing.T) {
	root := t.TempDir()
	for path, data := range map[string]string{"a.go": "a", "b.go": "b"} {
		if err := os.WriteFile(filepath.Join(root, path), []byte(data), 0644); err != nil {
			t.Fatal(err)
		}
	}
	first, err := buildImplementationIdentity(root, []ImplementationFile{{Path: "b.go", InclusionReason: "b"}, {Path: "a.go", InclusionReason: "a"}})
	if err != nil {
		t.Fatal(err)
	}
	second, err := buildImplementationIdentity(root, []ImplementationFile{{Path: "a.go", InclusionReason: "a"}, {Path: "b.go", InclusionReason: "b"}})
	if err != nil {
		t.Fatal(err)
	}
	if first.ImplementationHash != second.ImplementationHash {
		t.Fatal("file discovery order changed implementation identity")
	}
	if err := os.WriteFile(filepath.Join(root, "a.go"), []byte("A"), 0644); err != nil {
		t.Fatal(err)
	}
	changed, err := buildImplementationIdentity(root, []ImplementationFile{{Path: "a.go", InclusionReason: "a"}, {Path: "b.go", InclusionReason: "b"}})
	if err != nil {
		t.Fatal(err)
	}
	if changed.ImplementationHash == first.ImplementationHash {
		t.Fatal("implementation byte change did not change identity")
	}
}

func TestImplementationInventoryRejectsUnsafePaths(t *testing.T) {
	root := t.TempDir()
	outside := filepath.Join(t.TempDir(), "outside.go")
	if err := os.WriteFile(outside, []byte("outside"), 0644); err != nil {
		t.Fatal(err)
	}
	if _, err := buildImplementationIdentity(root, []ImplementationFile{{Path: "../outside.go", InclusionReason: "bad"}}); err == nil {
		t.Fatal("path traversal did not fail")
	}
	if _, err := buildImplementationIdentity(root, []ImplementationFile{{Path: "missing.go", InclusionReason: "missing"}}); err == nil {
		t.Fatal("missing implementation file did not fail")
	}
	link := filepath.Join(root, "link.go")
	if err := os.Symlink(outside, link); err != nil {
		t.Fatal(err)
	}
	if _, err := buildImplementationIdentity(root, []ImplementationFile{{Path: "link.go", InclusionReason: "bad"}}); err == nil || !strings.Contains(err.Error(), "escapes") {
		t.Fatalf("escaping symlink did not fail: %v", err)
	}
	if _, err := buildImplementationIdentity(root, []ImplementationFile{{Path: "missing.go", InclusionReason: "a"}, {Path: "missing.go", InclusionReason: "b"}}); err == nil {
		t.Fatal("duplicate implementation path did not fail")
	}
}

func TestEquivalentEffectiveConfigurationHasStableIdentity(t *testing.T) {
	context := ConfigurationContext{Symbol: "BTCUSDT", Market: "futures-um", Interval: "1m", EvaluationStartMS: 1000, EvaluationEndMS: 2000}
	implicit, err := ResolveConfiguration(context, nil)
	if err != nil {
		t.Fatal(err)
	}
	explicit, err := ResolveConfiguration(context, []byte(`{"series_cost_bps":5}`))
	if err != nil {
		t.Fatal(err)
	}
	implicitIdentity, err := ConfigurationHash(implicit)
	if err != nil {
		t.Fatal(err)
	}
	explicitIdentity, err := ConfigurationHash(explicit)
	if err != nil {
		t.Fatal(err)
	}
	if implicitIdentity.Hash != explicitIdentity.Hash {
		t.Fatalf("explicit default changed effective configuration identity")
	}
}

func TestConfigurationJSONKeyOrderAndFeatureRegimeSubidentitiesAreDeterministic(t *testing.T) {
	context := ConfigurationContext{Symbol: "BTCUSDT", Market: "futures-um", Interval: "1m", EvaluationStartMS: 1000, EvaluationEndMS: 2000}
	first, err := ResolveConfiguration(context, []byte(`{"series_cost_bps":6,"metric_risk_free_rate":0.04}`))
	if err != nil {
		t.Fatal(err)
	}
	second, err := ResolveConfiguration(context, []byte(`{"metric_risk_free_rate":0.04,"series_cost_bps":6}`))
	if err != nil {
		t.Fatal(err)
	}
	firstIdentity, _ := ConfigurationHash(first)
	secondIdentity, _ := ConfigurationHash(second)
	if firstIdentity.Hash != secondIdentity.Hash {
		t.Fatal("configuration JSON key order changed identity")
	}

	featureBase, _ := featureConfigurationHash(first)
	changedFeature := first
	changedFeature.Interval = "5m"
	featureChanged, _ := featureConfigurationHash(changedFeature)
	if featureBase == featureChanged {
		t.Fatal("changed feature configuration did not change feature configuration identity")
	}

	regimeBase, _ := regimeConfigurationHash(first)
	changedRegime := first
	changedRegime.RegimeGroups = changedRegime.RegimeGroups[:len(changedRegime.RegimeGroups)-1]
	regimeChanged, _ := regimeConfigurationHash(changedRegime)
	if regimeBase == regimeChanged {
		t.Fatal("changed regime configuration did not change regime configuration identity")
	}
}

func TestConfigurationChangesAndStrictParsing(t *testing.T) {
	context := ConfigurationContext{Symbol: "BTCUSDT", Market: "futures-um", Interval: "1m", EvaluationStartMS: 1000, EvaluationEndMS: 2000}
	base, _ := ResolveConfiguration(context, nil)
	baseHash, _ := ConfigurationHash(base)
	changed, err := ResolveConfiguration(context, []byte(`{"series_cost_bps":6}`))
	if err != nil {
		t.Fatal(err)
	}
	changedHash, _ := ConfigurationHash(changed)
	if baseHash.Hash == changedHash.Hash {
		t.Fatal("changed threshold/cost did not change identity")
	}
	changedThreshold, err := ResolveConfiguration(context, []byte(`{"gate_thresholds":{"minimum_h2_pf":1.11}}`))
	if err != nil {
		t.Fatal(err)
	}
	thresholdHash, _ := ConfigurationHash(changedThreshold)
	if baseHash.Hash == thresholdHash.Hash {
		t.Fatal("changed gate threshold did not change identity")
	}
	changedParameterCount, err := ResolveConfiguration(context, []byte(`{"model_parameter_count":3}`))
	if err != nil {
		t.Fatal(err)
	}
	parameterCountHash, _ := ConfigurationHash(changedParameterCount)
	if baseHash.Hash == parameterCountHash.Hash {
		t.Fatal("changed model parameter count did not change identity")
	}

	changedTimeframe, err := ResolveConfiguration(ConfigurationContext{Symbol: "BTCUSDT", Market: "futures-um", Interval: "5m", EvaluationStartMS: 1000, EvaluationEndMS: 2000}, nil)
	if err != nil {
		t.Fatal(err)
	}
	timeframeHash, _ := ConfigurationHash(changedTimeframe)
	if baseHash.Hash == timeframeHash.Hash {
		t.Fatal("changed timeframe did not change identity")
	}

	for name, raw := range map[string]string{
		"unknown":   `{"unknown":1}`,
		"duplicate": `{"series_cost_bps":5,"series_cost_bps":6}`,
		"secret":    `{"api_secret":"do-not-bind"}`,
	} {
		t.Run(name, func(t *testing.T) {
			if _, err := ResolveConfiguration(context, []byte(raw)); err == nil {
				t.Fatalf("strict configuration accepted %s field", name)
			}
		})
	}
}

func implementationRepositoryFixture(t *testing.T) string {
	t.Helper()
	root := t.TempDir()
	paths := map[string]struct{}{"internal/app/evaluate_candidate_deep.go": {}}
	for _, file := range append(append([]ImplementationFile(nil), featureImplementationFiles...), regimeImplementationFiles...) {
		paths[file.Path] = struct{}{}
	}
	for path := range paths {
		absolute := filepath.Join(root, filepath.FromSlash(path))
		if err := os.MkdirAll(filepath.Dir(absolute), 0755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(absolute, []byte("package fixture\n// "+path+"\n"), 0644); err != nil {
			t.Fatal(err)
		}
	}
	return root
}
