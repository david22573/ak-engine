package qualificationrunner

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"reflect"
	"sort"
	"strings"
	"testing"
	"time"

	"github.com/david22573/ak-engine/internal/preconditions"
	"github.com/david22573/ak-engine/internal/qualification"
	"github.com/david22573/ak-engine/internal/rifbridge"
)

func TestDryVerificationProducesNoCandidateOutcomes(t *testing.T) {
	request, _ := syntheticRequest(t, ModeVerify, "V00", false, false, nil)
	verified, readiness, err := Verify(request)
	if err != nil {
		t.Fatal(err)
	}
	if readiness.Label != NoOutcomesLabel || readiness.DataLoads != 0 || readiness.CandidateEvents != 0 || readiness.CandidateOutcomes != 0 {
		t.Fatalf("dry verification produced outcomes: %#v", readiness)
	}
	if verified.Variant.ID != "V00" || !reflect.DeepEqual(verified.Variant.Configuration, V00Configuration()) {
		t.Fatal("V00 did not resolve to exact baseline")
	}
	want, _ := readinessHash(readiness)
	if readiness.ArtifactSHA256 != want {
		t.Fatal("readiness artifact hash mismatch")
	}
}

func TestV00AndCanonicalConfigurationAreDeterministic(t *testing.T) {
	configuration := V00Configuration()
	first, err := CanonicalConfigurationHash(configuration)
	if err != nil {
		t.Fatal(err)
	}
	second, err := CanonicalConfigurationHash(V00Configuration())
	if err != nil {
		t.Fatal(err)
	}
	if first != second {
		t.Fatal("canonical configuration hash is unstable")
	}
	ledger := testLedger(t)
	resolved, err := ResolveVariantLedger(ledger, testIdentityLedger(ledger))
	if err != nil {
		t.Fatal(err)
	}
	v00, _ := findVariant(resolved, "V00")
	if v00.ConfigurationSHA256 != first {
		t.Fatal("V00 hash does not bind complete defaults")
	}
}

func TestVariantLedgerRejectsUnregisteredAndProhibitedChanges(t *testing.T) {
	base := testLedger(t)
	tests := []struct {
		name   string
		mutate func(*VariantLedger)
	}{
		{"unknown default", func(l *VariantLedger) { l.Variants[0].Configuration.SizingPolicy = "" }},
		{"unregistered variant", func(l *VariantLedger) { l.Variants[0].ID = "VX" }},
		{"more than twelve", func(l *VariantLedger) {
			for n := 3; n < 13; n++ {
				c := V00Configuration()
				c.ContextAgreement = "REQUIRE_POSITIVE_BTC_ETH_CONTEXT"
				l.Variants = append(l.Variants, RegisteredVariant{fmt.Sprintf("V%02d", n), []string{"context-agreement"}, c, ""})
			}
			l.MaximumVariants = 12
		}},
		{"unsupported dimension", func(l *VariantLedger) { l.Variants[1].Dimensions = []string{"symbols"} }},
		{"outcome filter", func(l *VariantLedger) { l.Variants[1].Configuration.OutcomeDerivedFilters = []string{"winning_only"} }},
		{"symbol filter", func(l *VariantLedger) { l.Variants[1].Configuration.Symbols = l.Variants[1].Configuration.Symbols[:7] }},
		{"date exclusion", func(l *VariantLedger) { l.Variants[1].Configuration.DateExclusions = []string{"2030-01-01"} }},
		{"quarter exclusion", func(l *VariantLedger) { l.Variants[1].Configuration.QuarterExclusions = []string{"2030-Q1"} }},
		{"cost change", func(l *VariantLedger) { l.Variants[1].Configuration.TransactionCostBPS = 9 }},
		{"side change", func(l *VariantLedger) { l.Variants[1].Configuration.Side = "SHORT" }},
		{"horizon change", func(l *VariantLedger) { l.Variants[1].Configuration.Horizon = "60m" }},
		{"sizing change", func(l *VariantLedger) { l.Variants[1].Configuration.SizingPolicy = "RISK_WEIGHTED" }},
		{"indicator change", func(l *VariantLedger) {
			l.Variants[1].Configuration.Indicators = append(l.Variants[1].Configuration.Indicators, "RSI")
		}},
		{"feature change", func(l *VariantLedger) {
			l.Variants[1].Configuration.Features = append(l.Variants[1].Configuration.Features, "future_return")
		}},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			ledger := cloneLedger(base)
			test.mutate(&ledger)
			sealed, err := SealVariantLedger(ledger)
			if err == nil {
				_, err = ResolveVariantLedger(sealed, testIdentityLedger(base))
			}
			if err == nil {
				t.Fatal("prohibited ledger mutation passed")
			}
		})
	}
}

func TestVerificationRejectsAuthorityGateDatasetAndRunnerSubstitution(t *testing.T) {
	base, _ := syntheticRequest(t, ModeVerify, "V00", false, false, nil)
	tests := []struct {
		name   string
		mutate func(*ExecutionRequest)
	}{
		{"independence omission", func(r *ExecutionRequest) { r.Independence = HashIdentity{} }},
		{"independence matching name wrong bytes", func(r *ExecutionRequest) { r.Independence.SHA256 = testHash('0') }},
		{"uncertainty omission", func(r *ExecutionRequest) { r.Uncertainty = HashIdentity{} }},
		{"uncertainty newer alias", func(r *ExecutionRequest) { r.Uncertainty.ID += ".v3" }},
		{"concentration substitution", func(r *ExecutionRequest) { r.Concentration.SHA256 = testHash('1') }},
		{"gate set substitution", func(r *ExecutionRequest) { r.QualificationGateSet.SHA256 = testHash('2') }},
		{"checkpoint substitution", func(r *ExecutionRequest) { r.Dataset.Checkpoint.SHA256 = testHash('3') }},
		{"source substitution", func(r *ExecutionRequest) { r.Dataset.SourceIdentitySHA256 = testHash('4') }},
		{"newer dataset", func(r *ExecutionRequest) { r.Dataset.Checkpoint.ID = "newer" }},
		{"extended interval", func(r *ExecutionRequest) {
			r.Dataset.EligibleInterval.End = r.Dataset.EligibleInterval.End.Add(time.Hour)
		}},
		{"symbol omission", func(r *ExecutionRequest) { r.Dataset.RequiredSymbols = r.Dataset.RequiredSymbols[:7] }},
		{"runner commit", func(r *ExecutionRequest) { r.Runner.GitCommit = strings.Repeat("0", 40) }},
		{"runner executable", func(r *ExecutionRequest) { r.Runner.ExecutableSHA256 = testHash('5') }},
		{"alternate equivalent executor", func(r *ExecutionRequest) { r.Runner.V00SourceSHA256 = testHash('6') }},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			request := cloneRequest(base)
			test.mutate(&request)
			if _, _, err := Verify(request); err == nil {
				t.Fatal("substitution passed")
			}
		})
	}
}

func TestModePartitionAndFrozenCandidateEnforcement(t *testing.T) {
	dev, _ := syntheticRequest(t, ModeDevelopment, "V00", false, false, nil)
	if _, _, err := Verify(dev); err != nil {
		t.Fatal(err)
	}
	dev.Partition = findPartitionMust(t, dev, "VALIDATION")
	if _, _, err := Verify(dev); err == nil {
		t.Fatal("DEVELOPMENT authorization read VALIDATION")
	}
	validation, _ := syntheticRequest(t, ModeValidation, "V01", false, false, nil)
	if _, _, err := Verify(validation); err != nil {
		t.Fatal(err)
	}
	validation.Partition = findPartitionMust(t, validation, "FINAL_HOLDOUT")
	if _, _, err := Verify(validation); err == nil {
		t.Fatal("VALIDATION authorization read FINAL_HOLDOUT")
	}
	final, _ := syntheticRequest(t, ModeFinalHoldout, "V00", false, true, nil)
	if _, _, err := Verify(final); err != nil {
		t.Fatal(err)
	}
	neighbor, _ := syntheticRequest(t, ModeFinalHoldout, "V01", false, true, nil)
	frozen := neighbor.RIF.Snapshot.FrozenCandidate
	frozen.VariantID = "V00"
	v00, _ := findVariant(neighbor.VariantLedger, "V00")
	frozen.ConfigurationSHA256 = v00.ConfigurationSHA256
	frozen.FrozenIdentityHash = ""
	frozen.FrozenIdentityHash = mustLocalHash(t, *frozen)
	neighbor.RIF = resealEnvelope(t, neighbor.RIF.Snapshot, neighbor.RIF.Authorization)
	if _, _, err := Verify(neighbor); err == nil {
		t.Fatal("FINAL_HOLDOUT accepted a stability neighbor instead of frozen V00")
	}
}

func TestSyntheticExecutionInvokesExactAuthoritiesAndBindsResults(t *testing.T) {
	rows := passingRows("DEVELOPMENT", time.Date(2030, 1, 1, 0, 0, 0, 0, time.UTC), 300)
	request, artifactBytes := syntheticRequest(t, ModeDevelopment, "V00", true, false, rows)
	result, err := Execute(request, artifactBytes)
	if err != nil {
		t.Fatal(err)
	}
	if result.Label != SyntheticLabel || !result.Independence.Invoked || !result.Uncertainty.Invoked || !result.Concentration.Invoked {
		t.Fatalf("accepted authorities were not invoked: %#v", result)
	}
	policy := preconditions.AcceptedIndependencePolicyV3Default()
	independenceHash, _ := preconditions.AcceptedIndependencePolicyHashV3(policy)
	methodHash, _ := preconditions.AcceptedUncertaintyMethodHashV2(preconditions.AcceptedUncertaintyMethodV2())
	if result.Independence.SHA256 != independenceHash || result.Uncertainty.SHA256 != methodHash || result.Concentration.SHA256 != policy.GovernanceDecisionHash {
		t.Fatal("result did not bind actually executed authorities")
	}
	if result.Metrics.EventCount != 300 || result.Metrics.IndependentClusterCount != 300 || result.UncertaintyResult.ClusterCount != 300 {
		t.Fatalf("unexpected synthetic execution metrics: %#v", result.Metrics)
	}
	want, _ := resultHash(result)
	if result.ResultSHA256 != want {
		t.Fatal("result self-hash mismatch")
	}
}

func TestSyntheticGovernedLifecycleExecutesEveryPartition(t *testing.T) {
	tests := []struct {
		mode      Mode
		partition string
		start     time.Time
		frozen    bool
	}{
		{ModeDevelopment, "DEVELOPMENT", time.Date(2030, 1, 1, 0, 0, 0, 0, time.UTC), false},
		{ModeValidation, "VALIDATION", time.Date(2031, 1, 1, 0, 0, 0, 0, time.UTC), false},
		{ModeFinalHoldout, "FINAL_HOLDOUT", time.Date(2032, 1, 1, 0, 0, 0, 0, time.UTC), true},
	}
	for _, test := range tests {
		t.Run(string(test.mode), func(t *testing.T) {
			request, artifact := syntheticRequest(t, test.mode, "V00", true, test.frozen, passingRows(test.partition, test.start, 300))
			result, err := Execute(request, artifact)
			if err != nil {
				t.Fatal(err)
			}
			if result.Partition != test.partition || result.Label != SyntheticLabel {
				t.Fatalf("unexpected synthetic lifecycle result: %#v", result)
			}
		})
	}
}

func TestExecutionAPIHasNoCallerProvidedExecutorBypass(t *testing.T) {
	typeOfExecute := reflect.TypeOf(Execute)
	if typeOfExecute.NumIn() != 2 || typeOfExecute.In(0) != reflect.TypeOf(ExecutionRequest{}) || typeOfExecute.In(1) != reflect.TypeOf([]byte{}) {
		t.Fatalf("execution API gained a caller implementation seam: %s", typeOfExecute)
	}
}

func TestUnregisteredCacheArtifactFails(t *testing.T) {
	rows := passingRows("DEVELOPMENT", time.Date(2030, 1, 1, 0, 0, 0, 0, time.UTC), 300)
	request, artifactBytes := syntheticRequest(t, ModeDevelopment, "V00", true, false, rows)
	var artifact PartitionArtifact
	if err := json.Unmarshal(artifactBytes, &artifact); err != nil {
		t.Fatal(err)
	}
	artifact.ArtifactSHA256 = testHash('0')
	mutated, _ := json.Marshal(artifact)
	mutated = append(mutated, '\n')
	if _, err := Execute(request, mutated); err == nil {
		t.Fatal("unregistered cache artifact executed")
	}
}

func TestDatasetRowsFailClosed(t *testing.T) {
	base := passingRows("DEVELOPMENT", time.Date(2030, 1, 1, 0, 0, 0, 0, time.UTC), 300)
	tests := []struct {
		name   string
		mutate func([]InputRow)
	}{
		{"cross partition", func(rows []InputRow) { rows[0].Partition = "VALIDATION" }},
		{"pre 2026", func(rows []InputRow) {
			rows[0].EventTime = time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)
			rows[0].AvailableAt = rows[0].EventTime
			rows[0].BTC.AvailableAt = rows[0].EventTime
			rows[0].ETH.AvailableAt = rows[0].EventTime
		}},
		{"wrong symbol", func(rows []InputRow) { rows[0].Symbol = "BTCUSDT" }},
		{"future availability", func(rows []InputRow) { rows[0].AvailableAt = rows[0].EventTime.Add(time.Second) }},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			rows := append([]InputRow(nil), base...)
			test.mutate(rows)
			request, artifact := syntheticRequest(t, ModeDevelopment, "V00", true, false, rows)
			if _, err := Execute(request, artifact); err == nil {
				t.Fatal("invalid dataset rows executed")
			}
		})
	}
}

func TestGateSetThresholdAndComparisonRegression(t *testing.T) {
	gates := qualification.AcceptedPR4B0GateSet()
	hash, err := qualification.PR4B0GateSetHash(gates)
	if err != nil {
		t.Fatal(err)
	}
	if !validSHA(hash) {
		t.Fatal("gate set has no canonical identity")
	}
	want := map[string]string{"minimum_profit_factor": ">=1.10", "uncertainty_lower_bound": ">0", "minimum_worst_period_pf": ">=0.95", "maximum_symbol_concentration": "<=1/2", "maximum_temporal_concentration": "<=1/2", "maximum_largest_cluster": "<=1/2", "maximum_top_five_clusters": "<=7/10", "minimum_stress_profit_factor": ">=1.01"}
	for _, gate := range gates.Comparisons {
		delete(want, gate.GateID)
		if expected, ok := map[string]string{"minimum_profit_factor": ">=1.10", "uncertainty_lower_bound": ">0", "minimum_worst_period_pf": ">=0.95", "maximum_symbol_concentration": "<=1/2", "maximum_temporal_concentration": "<=1/2", "maximum_largest_cluster": "<=1/2", "maximum_top_five_clusters": "<=7/10", "minimum_stress_profit_factor": ">=1.01"}[gate.GateID]; ok && gate.Operator+gate.Threshold != expected {
			t.Fatalf("gate %s semantic changed: %s%s", gate.GateID, gate.Operator, gate.Threshold)
		}
	}
	if len(want) != 0 {
		t.Fatalf("missing accepted gate regression coverage: %v", want)
	}
}

func testLedger(t *testing.T) VariantLedger {
	t.Helper()
	v00 := V00Configuration()
	v01 := V00Configuration()
	v01.ContextAgreement = "REQUIRE_POSITIVE_BTC_ETH_CONTEXT"
	v02 := V00Configuration()
	v02.EventQuality = "STRICT_CENTER_VOLATILITY"
	ledger := VariantLedger{SchemaVersion: VariantLedgerVersion, MaximumVariants: 3, V00ID: "V00", Variants: []RegisteredVariant{{"V00", []string{}, v00, ""}, {"V01", []string{"context-agreement"}, v01, ""}, {"V02", []string{"event-quality"}, v02, ""}}, StabilityNeighborhoods: []StabilityNeighborhood{{"V00", []string{"V01", "V02"}}, {"V01", []string{"V00", "V02"}}, {"V02", []string{"V00", "V01"}}}}
	sealed, err := SealVariantLedger(ledger)
	if err != nil {
		t.Fatal(err)
	}
	return sealed
}

func testIdentityLedger(ledger VariantLedger) IdentityVariantLedger {
	variants := make([]IdentityVariant, len(ledger.Variants))
	for i, item := range ledger.Variants {
		variants[i] = IdentityVariant{item.ID, item.ConfigurationSHA256, append([]string{}, item.Dimensions...)}
	}
	return IdentityVariantLedger{variants, ledger.MaximumVariants, ledger.V00ID, []string{"context-agreement", "cooldown/independence", "event-quality"}, "lowest registered canonical score, then lexicographic variant ID", append([]StabilityNeighborhood(nil), ledger.StabilityNeighborhoods...)}
}

func syntheticRequest(t *testing.T, mode Mode, variantID string, consumed, frozen bool, rows []InputRow) (ExecutionRequest, []byte) {
	t.Helper()
	ledger := testLedger(t)
	gates := qualification.AcceptedPR4B0GateSet()
	gateHash, _ := qualification.PR4B0GateSetHash(gates)
	gateRefs, _ := qualification.PR4B0GateIdentities(gates)
	gateIDs := make([]HashIdentity, len(gateRefs))
	for i, item := range gateRefs {
		gateIDs[i] = HashIdentity{item.ArtifactID, item.SHA256}
	}
	sort.Slice(gateIDs, func(i, j int) bool { return gateIDs[i].ID < gateIDs[j].ID })
	partitionName := map[Mode]string{ModeVerify: "DEVELOPMENT", ModeDevelopment: "DEVELOPMENT", ModeValidation: "VALIDATION", ModeFinalHoldout: "FINAL_HOLDOUT"}[mode]
	checkpoint := HashIdentity{"synthetic-checkpoint", testHash('a')}
	source := testHash('b')
	sealedBinary := testHash('c')
	universe, err := V00UniverseContract()
	if err != nil {
		t.Fatal(err)
	}
	artifactBytes := []byte(nil)
	coverage := testHash('d')
	if rows != nil {
		artifact, err := SealPartitionArtifact(PartitionArtifact{CheckpointSHA256: checkpoint.SHA256, SourceIdentitySHA256: source, SealedBinarySHA256: sealedBinary, Partition: partitionName, PartitionPlanSHA256: coverage, DatasetSymbols: append([]string(nil), acceptedDatasetSymbols...), TargetSymbols: append([]string(nil), acceptedTargetSymbols...), ContextOnlySymbols: append([]string(nil), acceptedContextOnlySymbols...), Rows: rows})
		if err != nil {
			t.Fatal(err)
		}
		artifactBytes, err = EncodePartitionArtifact(artifact)
		if err != nil {
			t.Fatal(err)
		}
	}
	partitions := []Partition{{"DEVELOPMENT", Interval{time.Date(2030, 1, 1, 0, 0, 0, 0, time.UTC), time.Date(2031, 1, 1, 0, 0, 0, 0, time.UTC)}, 365, coverage}, {"FINAL_HOLDOUT", Interval{time.Date(2032, 1, 1, 0, 0, 0, 0, time.UTC), time.Date(2033, 1, 1, 0, 0, 0, 0, time.UTC)}, 366, coverage}, {"VALIDATION", Interval{time.Date(2031, 1, 1, 0, 0, 0, 0, time.UTC), time.Date(2032, 1, 1, 0, 0, 0, 0, time.UTC)}, 365, coverage}}
	policy := preconditions.AcceptedIndependencePolicyV3Default()
	independenceHash, _ := preconditions.AcceptedIndependencePolicyHashV3(policy)
	uncertaintyHash, _ := preconditions.AcceptedUncertaintyMethodHashV2(preconditions.AcceptedUncertaintyMethodV2())
	authorities := AuthorityIdentity{HashIdentity{preconditions.AcceptedIndependencePolicyVersionV3, independenceHash}, HashIdentity{preconditions.AcceptedUncertaintyMethodVersion, uncertaintyHash}, policy.GovernanceDecisionHash, HashIdentity{qualification.PR4B0GateSetID, gateHash}, gateIDs, HashIdentity{"synthetic-cost-policy", testHash('e')}, HashIdentity{"synthetic-seed-policy", testHash('f')}}
	identity := ResearchIdentityV4{"ak.rif.research_identity.v4", "synthetic-engine-runner", RepositoryIdentity{strings.Repeat("1", 40), strings.Repeat("2", 40), strings.Repeat("3", 40), strings.Repeat("4", 40), strings.Repeat("5", 40), testHash('1')}, ProtocolIdentity{"synthetic-protocol", testHash('2'), testHash('2'), "synthetic.protocol.v1"}, CandidateScope{V00CandidateFamily, "LONG", "240m", false}, DatasetIdentity{checkpoint, source, HashIdentity{"synthetic-reacquisition", testHash('3')}, testHash('4'), sealedBinary, HashIdentity{"synthetic-abandoned", testHash('5')}, strings.Repeat("6", 40), append([]string(nil), universe.DatasetRequiredSymbols...), append([]string(nil), universe.CandidateTargetSymbols...), append([]string(nil), universe.ContextOnlySymbols...), universe.ContractSHA256, Interval{time.Date(2030, 1, 1, 0, 0, 0, 0, time.UTC), time.Date(2033, 1, 1, 0, 0, 0, 0, time.UTC)}, []Interval{{time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC), time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)}}, time.Date(2033, 1, 2, 0, 0, 0, 0, time.UTC)}, partitions, testIdentityLedger(ledger), authorities, AccessPolicy{true, []string{"exact runner", "reservation"}, []string{"development sealed", "registered nominee"}, []string{"candidate frozen", "validation sealed"}, []string{"dataset", "executable", "no defaults"}, 1, "NO_RETRY_AFTER_ACCESS", true}}
	identityBytes, _ := json.Marshal(identity)
	identityHash := hashBytes(identityBytes)
	partition := findIdentityPartition(identity, partitionName)
	selected, _ := findVariant(ledger, variantID)
	binding := rifbridge.ResearchExecutionBinding{VariantID: variantID, ConfigurationSHA256: selected.ConfigurationSHA256, ProtocolSHA256: identity.Protocol.SHA256, CheckpointSHA256: checkpoint.SHA256, IndependenceSHA256: independenceHash, UncertaintySHA256: uncertaintyHash, ConcentrationSHA256: policy.GovernanceDecisionHash, QualificationGateSHA256: gateHash, RunnerGitCommit: identity.Repositories.RunnerGitCommit, RunnerExecutableSHA256: identity.Repositories.RunnerExecutableSHA256, Partition: partitionName}
	state := "HOLDOUT_RESERVED"
	var authorization *rifbridge.PartitionAuthorization
	var frozenCandidate *rifbridge.FrozenResearchCandidate
	if mode != ModeVerify {
		state = map[Mode]string{ModeDevelopment: "DEVELOPMENT_AUTHORIZED", ModeValidation: "VALIDATION_AUTHORIZED", ModeFinalHoldout: "FINAL_HOLDOUT_AUTHORIZED"}[mode]
		authorization = &rifbridge.PartitionAuthorization{SchemaVersion: rifbridge.PartitionAuthorizationSchemaVersion, AuthorizationID: "authorization:synthetic", Sequence: 3, ResearchIdentityHash: identityHash, LifecycleState: state, Binding: binding, IssuedAt: time.Date(2029, 1, 1, 0, 0, 0, 0, time.UTC), OneShot: true, PriorLifecycleStateHash: testHash('6')}
	}
	if frozen {
		frozenCandidate = &rifbridge.FrozenResearchCandidate{VariantID: variantID, ConfigurationSHA256: selected.ConfigurationSHA256, ExecutableSHA256: identity.Repositories.RunnerExecutableSHA256, ProtocolSHA256: identity.Protocol.SHA256, CheckpointSHA256: checkpoint.SHA256, IndependenceSHA256: independenceHash, UncertaintySHA256: uncertaintyHash, ConcentrationSHA256: policy.GovernanceDecisionHash, QualificationGateSHA256: gateHash, NoUnresolvedDefaults: true, FrozenAt: time.Date(2029, 1, 1, 0, 0, 1, 0, time.UTC)}
		frozenCandidate.FrozenIdentityHash = mustLocalHash(t, *frozenCandidate)
	}
	envelope := buildEnvelope(t, identityBytes, identityHash, identity, state, authorization, consumed, frozenCandidate)
	request := ExecutionRequest{RequestSchemaVersion, mode, envelope, identity.Protocol, ledger, variantID, selected.ConfigurationSHA256, DatasetBinding{checkpoint, source, sealedBinary, append([]string(nil), universe.DatasetRequiredSymbols...), append([]string(nil), universe.CandidateTargetSymbols...), append([]string(nil), universe.ContextOnlySymbols...), universe.ContractSHA256, identity.Dataset.EligibleInterval, identity.Dataset.ProhibitedPriorExposure, identity.Dataset.AvailabilityCutoff}, partition, V00CandidateFamily, identity.Authorities.Independence, identity.Authorities.Uncertainty, HashIdentity{"ak.engine.concentration-governance.structural.v1", policy.GovernanceDecisionHash}, identity.Authorities.QualificationGateSet, identity.Authorities.TransactionCostPolicy, identity.Authorities.DeterministicSeedPolicy, RunnerIdentity{identity.Repositories.RunnerGitCommit, identity.Repositories.RunnerExecutableSHA256, V00SourceSHA256}}
	return request, artifactBytes
}

func buildEnvelope(t *testing.T, identityBytes []byte, identityHash string, identity ResearchIdentityV4, state string, authorization *rifbridge.PartitionAuthorization, consumed bool, frozen *rifbridge.FrozenResearchCandidate) rifbridge.ResearchGovernanceEnvelope {
	t.Helper()
	final := findIdentityPartition(identity, "FINAL_HOLDOUT")
	finalBytes, _ := json.Marshal(final)
	ledgerHash, _ := canonicalHash(identity.VariantLedger)
	authorityHash, _ := canonicalHash(identity.Authorities)
	reservation := &rifbridge.HoldoutReservation{SchemaVersion: rifbridge.HoldoutReservationSchemaVersion, ReservationID: "reservation:synthetic", ResearchIdentityHash: identityHash, FinalHoldout: finalBytes, ProtocolSHA256: identity.Protocol.SHA256, CheckpointSHA256: identity.Dataset.Checkpoint.SHA256, VariantLedgerSHA256: ledgerHash, AuthoritySetSHA256: authorityHash, CreatedAt: time.Date(2028, 1, 1, 0, 0, 0, 0, time.UTC)}
	reservation.RecordHash = mustLocalHash(t, *reservation)
	history := []rifbridge.ResearchLifecycleRecord{{SchemaVersion: rifbridge.ResearchLifecycleRecordSchemaVersion, Sequence: 1, EventType: "RESEARCH_IDENTITY_REGISTERED", ToState: "RESEARCH_IDENTITY_REGISTERED", OccurredAt: time.Date(2028, 1, 1, 0, 0, 1, 0, time.UTC), EvidenceSHA256: identityHash, PriorStateHash: testHash('7')}, {SchemaVersion: rifbridge.ResearchLifecycleRecordSchemaVersion, Sequence: 2, EventType: "HOLDOUT_RESERVED", FromState: "RESEARCH_IDENTITY_REGISTERED", ToState: "HOLDOUT_RESERVED", OccurredAt: time.Date(2028, 1, 1, 0, 0, 2, 0, time.UTC), EvidenceSHA256: reservation.RecordHash, PriorStateHash: testHash('8')}}
	history[0].RecordHash = mustLocalHash(t, history[0])
	history[1].PreviousHash = history[0].RecordHash
	history[1].RecordHash = mustLocalHash(t, history[1])
	sequence := uint64(2)
	var authorizations []rifbridge.PartitionAuthorization
	if authorization != nil {
		authorization.PreviousHash = ""
		authorization.RecordHash = mustLocalHash(t, *authorization)
		authorizations = []rifbridge.PartitionAuthorization{*authorization}
		sequence = 3
		event := rifbridge.ResearchLifecycleRecord{SchemaVersion: rifbridge.ResearchLifecycleRecordSchemaVersion, Sequence: 3, EventType: state, FromState: "HOLDOUT_RESERVED", ToState: state, OccurredAt: time.Date(2028, 1, 1, 0, 0, 3, 0, time.UTC), EvidenceSHA256: authorization.RecordHash, PriorStateHash: testHash('9'), PreviousHash: history[len(history)-1].RecordHash}
		event.RecordHash = mustLocalHash(t, event)
		history = append(history, event)
	}
	var receipts []rifbridge.PartitionAccessReceipt
	if consumed {
		receipt := rifbridge.PartitionAccessReceipt{SchemaVersion: rifbridge.PartitionAccessReceiptSchemaVersion, Sequence: sequence + 1, AuthorizationID: authorization.AuthorizationID, ResearchIdentityHash: identityHash, Binding: authorization.Binding, AccessedAt: time.Date(2028, 1, 1, 0, 0, 4, 0, time.UTC), PriorLifecycleStateHash: testHash('a')}
		receipt.RecordHash = mustLocalHash(t, receipt)
		receipts = []rifbridge.PartitionAccessReceipt{receipt}
		sequence++
		event := rifbridge.ResearchLifecycleRecord{SchemaVersion: rifbridge.ResearchLifecycleRecordSchemaVersion, Sequence: sequence, EventType: authorization.Binding.Partition + "_ACCESS_CONSUMED", FromState: state, ToState: state, OccurredAt: receipt.AccessedAt, EvidenceSHA256: receipt.RecordHash, PriorStateHash: testHash('a'), PreviousHash: history[len(history)-1].RecordHash}
		event.RecordHash = mustLocalHash(t, event)
		history = append(history, event)
	}
	snapshot := rifbridge.ResearchGovernanceSnapshot{SchemaVersion: rifbridge.ResearchGovernanceStoreSchemaVersion, Identity: identityBytes, IdentityHash: identityHash, Reservation: reservation, State: state, Sequence: sequence, FrozenCandidate: frozen, Authorizations: authorizations, AccessReceipts: receipts, LifecycleHistory: history}
	snapshot.IntegrityHash = mustLocalHash(t, snapshot)
	envelope := rifbridge.ResearchGovernanceEnvelope{SchemaVersion: rifbridge.ResearchGovernanceEnvelopeSchemaVersion, Snapshot: snapshot, Authorization: authorization}
	envelope.EnvelopeHash = mustLocalHash(t, envelope)
	return envelope
}

func passingRows(partition string, start time.Time, count int) []InputRow {
	rows := make([]InputRow, count)
	for i := range rows {
		event := start.Add(time.Duration(i) * 25 * time.Hour)
		symbol := acceptedTargetSymbols[i%len(acceptedTargetSymbols)]
		gross := 20.0
		if i%10 == 0 {
			gross = -20
		}
		close := 100.0
		future := close * (1 + gross/10000)
		contextTime := event.Add(-time.Minute)
		rows[i] = InputRow{partition, symbol, event, event, close, future, 101, 102, -0.1, 0.003, true, Context{fmt.Sprintf("btc-%03d", i), testHash('b'), contextTime, 0.01}, Context{fmt.Sprintf("eth-%03d", i), testHash('c'), contextTime, 0.01}}
	}
	return rows
}

func findIdentityPartition(identity ResearchIdentityV4, name string) Partition {
	for _, item := range identity.Partitions {
		if item.Name == name {
			return item
		}
	}
	return Partition{}
}
func findPartitionMust(t *testing.T, request ExecutionRequest, name string) Partition {
	t.Helper()
	partition, ok := findPartition(requestIdentity(t, request), name)
	if !ok {
		t.Fatal("partition missing")
	}
	return partition
}
func requestIdentity(t *testing.T, request ExecutionRequest) ResearchIdentityV4 {
	t.Helper()
	identity, _, err := decodeIdentity(request.RIF.Snapshot.Identity)
	if err != nil {
		t.Fatal(err)
	}
	return identity
}
func mustLocalHash(t *testing.T, value any) string {
	t.Helper()
	hash, err := canonicalHash(value)
	if err != nil {
		t.Fatal(err)
	}
	return hash
}
func hashBytes(data []byte) string {
	digest := sha256.Sum256(data)
	return "sha256:" + hex.EncodeToString(digest[:])
}
func testHash(char byte) string { return "sha256:" + strings.Repeat(string(char), 64) }
func cloneLedger(value VariantLedger) VariantLedger {
	data, _ := json.Marshal(value)
	var out VariantLedger
	_ = json.Unmarshal(data, &out)
	return out
}
func cloneRequest(value ExecutionRequest) ExecutionRequest {
	data, _ := json.Marshal(value)
	var out ExecutionRequest
	_ = json.Unmarshal(data, &out)
	return out
}

func resealEnvelope(t *testing.T, snapshot rifbridge.ResearchGovernanceSnapshot, authorization *rifbridge.PartitionAuthorization) rifbridge.ResearchGovernanceEnvelope {
	t.Helper()
	snapshot.IntegrityHash = ""
	snapshot.IntegrityHash = mustLocalHash(t, snapshot)
	envelope := rifbridge.ResearchGovernanceEnvelope{SchemaVersion: rifbridge.ResearchGovernanceEnvelopeSchemaVersion, Snapshot: snapshot, Authorization: authorization}
	envelope.EnvelopeHash = mustLocalHash(t, envelope)
	return envelope
}
