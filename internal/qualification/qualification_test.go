package qualification

import (
	"encoding/json"
	"strings"
	"testing"
	"time"
)

func TestInventoryCoversRegisteredCandidates(t *testing.T) {
	candidate := validCandidateRecord()
	inventory := CandidateInventory{
		SchemaVersion: InventorySchemaVersion, Phase: "PR4B0", AcceptedEngineBaseline: strings.Repeat("a", 40),
		CandidateCount: 1, Candidates: []CandidateRecord{candidate}, RegisteredCandidateIDs: []string{candidate.CandidateID},
		UnknownImplementations: []string{}, OmittedCandidates: []string{},
	}
	if err := ValidateInventory(inventory, []string{candidate.CandidateID}); err != nil {
		t.Fatalf("ValidateInventory() error = %v", err)
	}
}

func TestInventoryReportsUnknownImplementation(t *testing.T) {
	candidate := validCandidateRecord()
	got := FindUnknownImplementations([]CandidateRecord{candidate}, []string{candidate.CandidateID, "unknown-candidate"})
	if len(got) != 1 || got[0] != "unknown-candidate" {
		t.Fatalf("FindUnknownImplementations() = %v", got)
	}
}

func TestInventoryRejectsRegisteredIdentitySubstitution(t *testing.T) {
	candidate := validCandidateRecord()
	inventory := CandidateInventory{
		SchemaVersion: InventorySchemaVersion, Phase: "PR4B0", AcceptedEngineBaseline: strings.Repeat("a", 40),
		CandidateCount: 1, Candidates: []CandidateRecord{candidate}, RegisteredCandidateIDs: []string{"substituted"},
		UnknownImplementations: []string{}, OmittedCandidates: []string{},
	}
	if err := ValidateInventory(inventory, []string{candidate.CandidateID}); err == nil || !strings.Contains(err.Error(), "registered_candidate_ids mismatch") {
		t.Fatalf("ValidateInventory() error = %v", err)
	}
}

func TestInventoryPreservesRejectedNearMissAndProbeLabels(t *testing.T) {
	tests := []struct {
		name           string
		classification EligibilityClassification
		status         FinalStatus
		wantError      string
	}{
		{name: "rejected cannot qualify", classification: ClassificationRejected, status: StatusQualified, wantError: "must remain REJECTED"},
		{name: "near miss cannot qualify", classification: ClassificationNearMiss, status: StatusQualified, wantError: "must remain NEAR_MISS"},
		{name: "infrastructure probe cannot qualify", classification: ClassificationInfrastructureProbe, status: StatusQualified, wantError: "cannot be QUALIFIED"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			candidate := validCandidateRecord()
			candidate.EligibilityClassification = tt.classification
			candidate.FinalStatus = tt.status
			if err := validateCandidateRecord(candidate); err == nil || !strings.Contains(err.Error(), tt.wantError) {
				t.Fatalf("validateCandidateRecord() error = %v, want %q", err, tt.wantError)
			}
		})
	}
}

func TestInventoryRequiresExplicitMissingEvidenceReason(t *testing.T) {
	candidate := validCandidateRecord()
	candidate.EligibilityClassification = ClassificationMissingEvidence
	candidate.FinalStatus = StatusPITEvidenceMissing
	candidate.ExclusionReasons = nil
	if err := validateCandidateRecord(candidate); err == nil || !strings.Contains(err.Error(), "explicit exclusion_reasons") {
		t.Fatalf("validateCandidateRecord() error = %v", err)
	}
}

func TestProtocolValidationAndFiniteSearchBudget(t *testing.T) {
	protocol := validProtocol()
	if err := ValidateProtocol(protocol); err != nil {
		t.Fatalf("ValidateProtocol() error = %v", err)
	}
	encoded, err := json.Marshal(protocol)
	if err != nil {
		t.Fatal(err)
	}
	for _, field := range []string{"minimum_events", "minimum_independent_clusters", "minimum_profit_factor", "stress_total_bps", "maximum_candidate_count", "maximum_total_evaluations"} {
		if !strings.Contains(string(encoded), `"`+field+`"`) {
			t.Fatalf("numeric gate %q missing from JSON", field)
		}
	}
}

func TestProtocolRejectsOverlapHoldoutSelectionAndImplicitCosts(t *testing.T) {
	tests := []struct {
		name   string
		mutate func(*QualificationProtocol)
		want   string
	}{
		{name: "overlap", mutate: func(p *QualificationProtocol) {
			p.DataIntegrity.ValidationWindow.Start = p.DataIntegrity.DevelopmentWindow.End.Add(-time.Hour)
		}, want: "overlap"},
		{name: "holdout selection", mutate: func(p *QualificationProtocol) { p.Search.FinalHoldoutUsedForSelection = true }, want: "cannot be used"},
		{name: "costs", mutate: func(p *QualificationProtocol) { p.Cost.StressTotalBPS = 0 }, want: "cost assumptions"},
		{name: "understated component total", mutate: func(p *QualificationProtocol) { p.Cost.StressTotalBPS = 9 }, want: "component total"},
		{name: "sample minimum", mutate: func(p *QualificationProtocol) { p.Sample.MinimumIndependentClusters = 0 }, want: "sample minimums"},
		{name: "finite budget", mutate: func(p *QualificationProtocol) { p.Search.MaximumEvaluations = 0 }, want: "finite"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			protocol := validProtocol()
			tt.mutate(&protocol)
			if err := ValidateProtocol(protocol); err == nil || !strings.Contains(err.Error(), tt.want) {
				t.Fatalf("ValidateProtocol() error = %v, want %q", err, tt.want)
			}
		})
	}
}

func TestEvaluationInputRejectsLeakageIdentityContextAndGaps(t *testing.T) {
	protocol := validProtocol()
	input := validEvaluationInput(protocol)
	if err := ValidateEvaluationInput(protocol, input); err != nil {
		t.Fatalf("valid input rejected: %v", err)
	}
	tests := []struct {
		name   string
		mutate func(*EvaluationInput)
		want   string
	}{
		{name: "future", mutate: func(i *EvaluationInput) { i.Observations[0].UsesFutureData = true }, want: "future data"},
		{name: "availability", mutate: func(i *EvaluationInput) {
			i.Observations[0].FeatureAvailableAt = i.Observations[0].DecisionAt.Add(time.Second)
		}, want: "unavailable"},
		{name: "manifest", mutate: func(i *EvaluationInput) { i.ManifestHash = sha('9') }, want: "manifest"},
		{name: "dataset version", mutate: func(i *EvaluationInput) { i.DatasetVersion = "dataset-v2" }, want: "dataset_version"},
		{name: "missing context", mutate: func(i *EvaluationInput) { i.Observations[0].ContextComplete = false }, want: "context"},
		{name: "gap", mutate: func(i *EvaluationInput) { i.InternalGapCount = 1 }, want: "gaps"},
		{name: "duplicate event", mutate: func(i *EvaluationInput) { i.Observations = append(i.Observations, i.Observations[0]) }, want: "duplicate"},
		{name: "partition substitution", mutate: func(i *EvaluationInput) { i.ObservedPartitions[0] = "partition-2" }, want: "exactly match"},
		{name: "observation partition", mutate: func(i *EvaluationInput) { i.Observations[0].Partition = "partition-2" }, want: "unexpected partition"},
		{name: "outside research window", mutate: func(i *EvaluationInput) {
			i.Observations[0].DecisionAt = protocol.DataIntegrity.ResearchWindow.End.Add(time.Second)
		}, want: "outside the research window"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			input := validEvaluationInput(protocol)
			tt.mutate(&input)
			if err := ValidateEvaluationInput(protocol, input); err == nil || !strings.Contains(err.Error(), tt.want) {
				t.Fatalf("ValidateEvaluationInput() error = %v, want %q", err, tt.want)
			}
		})
	}
}

func TestDuplicateEventsDoNotInflateIndependentClusterCount(t *testing.T) {
	now := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)
	observations := []EvaluationObservation{
		{EventID: "a", ClusterID: "cluster-1", DecisionAt: now, FeatureAvailableAt: now},
		{EventID: "b", ClusterID: "cluster-1", DecisionAt: now, FeatureAvailableAt: now},
		{EventID: "c", ClusterID: "cluster-2", DecisionAt: now, FeatureAvailableAt: now},
	}
	if got := IndependentClusterCount(observations); got != 2 {
		t.Fatalf("IndependentClusterCount() = %d, want 2", got)
	}
}

func TestQualificationRequiresEveryMandatoryGate(t *testing.T) {
	candidate := validCandidateRecord()
	candidate.EligibilityClassification = ClassificationQualificationCandidate
	candidate.ImplementationReproducible = true
	all := allGateEvidence()
	if got := QualificationStatus(candidate, all, false); got != StatusConcentrationAuthorityMissing {
		t.Fatalf("report-only all-pass evidence bypassed concentration authority: %s", got)
	}
	mutations := []struct {
		name string
		set  func(*GateEvidence)
	}{
		{"data", func(e *GateEvidence) { e.DataIntegrity = false }},
		{"sample", func(e *GateEvidence) { e.SampleSufficiency = false }},
		{"net", func(e *GateEvidence) { e.NetPerformance = false }},
		{"tail", func(e *GateEvidence) { e.DownsideTail = false }},
		{"uncertainty", func(e *GateEvidence) { e.UncertaintyBound = false }},
		{"oos", func(e *GateEvidence) { e.OutOfSample = false }},
		{"walk forward", func(e *GateEvidence) { e.WalkForward = false }},
		{"worst period", func(e *GateEvidence) { e.WorstPeriod = false }},
		{"symbol concentration", func(e *GateEvidence) { e.SymbolConcentration = false }},
		{"temporal concentration", func(e *GateEvidence) { e.TemporalConcentration = false }},
		{"regime concentration", func(e *GateEvidence) { e.RegimeConcentration = false }},
		{"neighbors", func(e *GateEvidence) { e.ParameterNeighborhood = false }},
		{"clusters", func(e *GateEvidence) { e.ClusterDeduplication = false }},
		{"context", func(e *GateEvidence) { e.MissingContextSensitivity = false }},
		{"cost", func(e *GateEvidence) { e.CostStress = false }},
		{"leakage", func(e *GateEvidence) { e.LeakageSafety = false }},
		{"simplicity", func(e *GateEvidence) { e.Simplicity = false }},
		{"implementation", func(e *GateEvidence) { e.ImplementationComplete = false }},
		{"pit", func(e *GateEvidence) { e.PITEvidence = false }},
		{"holdout", func(e *GateEvidence) { e.HoldoutAuthorized = false }},
	}
	for _, mutation := range mutations {
		t.Run(mutation.name, func(t *testing.T) {
			evidence := all
			mutation.set(&evidence)
			if got := QualificationStatus(candidate, evidence, false); got == StatusQualified {
				t.Fatal("candidate qualified with a failed mandatory gate")
			}
		})
	}
}

func TestStrongestCandidateDoesNotOverrideGates(t *testing.T) {
	candidate := validCandidateRecord()
	candidate.EligibilityClassification = ClassificationQualificationCandidate
	evidence := allGateEvidence()
	evidence.CostStress = false
	if got := QualificationStatus(candidate, evidence, true); got == StatusQualified {
		t.Fatal("strongest candidate qualified despite failed cost stress")
	}
}

func TestPriorLabelsTakePrecedence(t *testing.T) {
	evidence := allGateEvidence()
	tests := []struct {
		classification EligibilityClassification
		want           FinalStatus
	}{
		{ClassificationRejected, StatusRejected},
		{ClassificationNearMiss, StatusNearMiss},
		{ClassificationInfrastructureProbe, StatusInsufficientSample},
	}
	for _, tt := range tests {
		candidate := validCandidateRecord()
		candidate.EligibilityClassification = tt.classification
		if got := QualificationStatus(candidate, evidence, true); got != tt.want {
			t.Fatalf("classification %s = %s, want %s", tt.classification, got, tt.want)
		}
	}
}

func TestFrozenDescriptorValidAndDeterministic(t *testing.T) {
	descriptor := validFrozenDescriptor(t)
	if err := descriptor.Verify(); err != nil {
		t.Fatalf("Verify() error = %v", err)
	}
	first, err := descriptor.CanonicalHash()
	if err != nil {
		t.Fatal(err)
	}
	second, err := descriptor.CanonicalHash()
	if err != nil || first != second {
		t.Fatalf("hashes differ: %q %q error=%v", first, second, err)
	}
}

func TestFrozenDescriptorIdentityMutationInvalidatesHash(t *testing.T) {
	mutations := []struct {
		name string
		set  func(*FrozenCandidateDescriptor)
	}{
		{"implementation", func(d *FrozenCandidateDescriptor) { d.ImplementationHash = sha('2') }},
		{"configuration", func(d *FrozenCandidateDescriptor) { d.ConfigurationHash = sha('3') }},
		{"parameters", func(d *FrozenCandidateDescriptor) { d.ParameterHash = sha('4') }},
		{"capability", func(d *FrozenCandidateDescriptor) { d.CapabilityHash = sha('5') }},
		{"dataset", func(d *FrozenCandidateDescriptor) { d.DatasetVersion = "dataset-v2" }},
		{"window", func(d *FrozenCandidateDescriptor) { d.ResearchWindowStart = d.ResearchWindowStart.Add(time.Second) }},
		{"manifest", func(d *FrozenCandidateDescriptor) { d.ManifestID = "manifest-v2" }},
		{"qualification report", func(d *FrozenCandidateDescriptor) { d.QualificationReportHash = sha('7') }},
	}
	for _, mutation := range mutations {
		t.Run(mutation.name, func(t *testing.T) {
			descriptor := validFrozenDescriptor(t)
			mutation.set(&descriptor)
			if err := descriptor.Verify(); err == nil {
				t.Fatal("mutation did not invalidate descriptor")
			}
		})
	}
}

func TestFrozenDescriptorRejectsUnknownSchema(t *testing.T) {
	descriptor := validFrozenDescriptor(t)
	descriptor.SchemaVersion = "ak.engine.frozen_candidate.v999"
	if err := descriptor.Verify(); err == nil || !strings.Contains(err.Error(), "unsupported") {
		t.Fatalf("Verify() error = %v", err)
	}
}

func TestRegistrationRequestValidAndMutationSensitive(t *testing.T) {
	request := validRegistrationRequest(t)
	if err := request.Verify(); err != nil {
		t.Fatalf("Verify() error = %v", err)
	}
	mutated := request
	mutated.ResearchIdentity.DatasetVersion = "dataset-v2"
	if err := mutated.Verify(); err == nil {
		t.Fatal("identity mutation did not invalidate request")
	}
}

func TestRegistrationRequestRequiresQualifiedAndCannotClaimAuthorization(t *testing.T) {
	request := validRegistrationRequest(t)
	request.QualificationVerdict = StatusNearMiss
	if err := request.Verify(); err == nil || !strings.Contains(err.Error(), "QUALIFIED") {
		t.Fatalf("near-miss Verify() error = %v", err)
	}
	request = validRegistrationRequest(t)
	request.RIFAuthorized = true
	if err := request.Verify(); err == nil || !strings.Contains(err.Error(), "cannot claim") {
		t.Fatalf("authorization Verify() error = %v", err)
	}
}

func TestRegistrationRequestIdentityMustMatchFrozenDescriptor(t *testing.T) {
	request := validRegistrationRequest(t)
	request.CandidateImplementationIdentity.ParameterHash = sha('7')
	if err := request.Verify(); err == nil || !strings.Contains(err.Error(), "does not match") {
		t.Fatalf("Verify() error = %v", err)
	}
}

func validCandidateRecord() CandidateRecord {
	return CandidateRecord{
		CandidateID: "candidate-1", CandidateVersion: "v1", RegisteredImplementation: true,
		ImplementationLocation: "internal/strategy/candidate.go", ImplementationSourceRef: strings.Repeat("a", 40),
		ImplementationSHA256: sha('1'), ImplementationReproducible: true, StrategyFamily: "Candidate",
		DirectionSupport: []string{"long"}, Symbols: []string{"BTCUSDT"}, RequiredContext: []string{"BTCUSDT"},
		RequiredTimeframes: []string{"1h"}, FeatureRequirements: []string{"close"}, ParameterSet: map[string]any{"threshold": 1},
		ResearchPhase: "test", InSampleResults: emptyEvidence(), OutOfSampleResults: emptyEvidence(), WalkForwardResults: emptyEvidence(),
		CostStressResults: emptyEvidence(), WorstPeriodResults: emptyEvidence(), ConcentrationResults: emptyEvidence(),
		CurrentResearchLabel: "MISSING_EVIDENCE", EligibilityClassification: ClassificationMissingEvidence,
		FinalStatus: StatusPITEvidenceMissing, ExclusionReasons: []string{"PIT evidence missing"}, Evidence: []EvidenceReference{},
	}
}

func emptyEvidence() EvidenceResult {
	return EvidenceResult{Status: "MISSING", Metrics: map[string]any{}, Notes: []string{}}
}

func validProtocol() QualificationProtocol {
	start := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	devEnd := start.Add(90 * 24 * time.Hour)
	validationEnd := devEnd.Add(30 * 24 * time.Hour)
	holdoutEnd := validationEnd.Add(30 * 24 * time.Hour)
	return QualificationProtocol{
		SchemaVersion: ProtocolSchemaVersion,
		DataIntegrity: DataIntegrityGates{
			DatasetID: "candles:BTCUSDT:1h", DatasetVersion: "dataset-v1", ManifestID: "manifest-v1", ManifestHash: sha('a'),
			ResearchWindow: Window{Start: start, End: holdoutEnd}, DevelopmentWindow: Window{Start: start, End: devEnd},
			ValidationWindow: Window{Start: devEnd, End: validationEnd}, FinalHoldoutWindow: Window{Start: validationEnd, End: holdoutEnd},
			RequiredSymbols: []string{"BTCUSDT"}, RequiredContextSymbols: []string{"ETHUSDT"}, ExpectedPartitions: []string{"partition-1"},
			GapPolicy: "REJECT_INTERNAL_GAPS", PITAvailabilityPolicy: "available-after-close-v1",
		},
		Sample:         SampleGates{MinimumEvents: 300, MinimumIndependentClusters: 300, MinimumTradesOrDecisions: 300, MinimumSymbols: 1, MinimumMonths: 3, MinimumPositiveRegimes: 1, MinimumNegativeRegimes: 1},
		Performance:    PerformanceGates{MinimumNetExpectancyBPS: 0.01, MinimumProfitFactor: 1.1, MaximumDrawdownBPS: 1000, MinimumConfidenceLowerBoundBPS: 0, DownsideTailPolicy: "worst decile expectancy must remain above -25 bps"},
		Robustness:     RobustnessGates{RequireOutOfSample: true, RequireWalkForward: true, MinimumWorstPeriodProfitFactor: 0.95, MaximumSymbolContributionPercent: 50, MaximumTemporalContributionPercent: 50, MaximumRegimeContributionPercent: 60, MinimumStableNeighbors: 2, RequireClusterDeduplication: true, RequireMissingContextSensitivity: true},
		Cost:           CostGates{FeeBPS: 5, SpreadBPS: 1, SlippageBPS: 1, FundingBPS: 1, AdverseSelectionBPS: 2, StressTotalBPS: 10, MinimumStressProfitFactor: 1.01, MinimumStressExpectancyBPS: 0.01},
		LeakageRules:   []string{"no future candles", "no revised data", "no outcome features", "no holdout feature selection", "PIT source timing"},
		SimplicityRule: "simplest passing candidate wins",
		Search:         SearchBudget{MaximumCandidateCount: 4, MaximumEvaluations: 16, StoppingRule: "stop at budget or first fully passing candidate", SelectionRule: "highest validation lower confidence bound", FinalHoldoutUsedForSelection: false},
	}
}

func validEvaluationInput(protocol QualificationProtocol) EvaluationInput {
	decision := protocol.DataIntegrity.DevelopmentWindow.Start.Add(time.Hour)
	return EvaluationInput{
		DatasetID: protocol.DataIntegrity.DatasetID, DatasetVersion: protocol.DataIntegrity.DatasetVersion,
		ManifestID: protocol.DataIntegrity.ManifestID, ManifestHash: protocol.DataIntegrity.ManifestHash,
		ObservedPartitions: []string{"partition-1"},
		Observations:       []EvaluationObservation{{EventID: "event-1", ClusterID: "cluster-1", Partition: "partition-1", DecisionAt: decision, FeatureAvailableAt: decision.Add(-time.Second), ContextComplete: true}},
	}
}

func allGateEvidence() GateEvidence {
	return GateEvidence{
		DataIntegrity: true, SampleSufficiency: true, NetPerformance: true, DownsideTail: true, UncertaintyBound: true,
		OutOfSample: true, WalkForward: true, WorstPeriod: true, SymbolConcentration: true, TemporalConcentration: true,
		RegimeConcentration: true, ParameterNeighborhood: true, ClusterDeduplication: true, MissingContextSensitivity: true,
		CostStress: true, LeakageSafety: true, Simplicity: true, ImplementationComplete: true, PITEvidence: true, HoldoutAuthorized: true,
	}
}

func validFrozenDescriptor(t *testing.T) FrozenCandidateDescriptor {
	t.Helper()
	start := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	descriptor := FrozenCandidateDescriptor{
		SchemaVersion: FrozenDescriptorSchemaVersion, CandidateID: "candidate-1", CandidateVersion: "v1",
		StrategyFamily: "Candidate", DirectionModel: "long", ImplementationHash: sha('1'), ConfigurationHash: sha('2'),
		ParameterHash: sha('3'), FeatureSchema: "features-v1", CapabilityHash: sha('4'), EngineModule: "github.com/david22573/ak-engine",
		EngineBuildID: "ak-engine/test-build", DatasetID: "candles:BTCUSDT:1h", DatasetVersion: "dataset-v1",
		ResearchWindowStart: start, ResearchWindowEnd: start.Add(24 * time.Hour), EvaluationCutoff: start.Add(25 * time.Hour),
		ManifestID: "manifest-v1", ManifestHash: sha('5'), CoveragePolicyVersion: "coverage-v1", AvailabilityPolicyVersion: "availability-v1",
		QualificationReportID: "qualification-report-v1", QualificationReportHash: sha('6'), FrozenAt: start.Add(26 * time.Hour),
	}
	hash, err := descriptor.CanonicalHash()
	if err != nil {
		t.Fatal(err)
	}
	descriptor.DescriptorHash = hash
	return descriptor
}

func validRegistrationRequest(t *testing.T) CandidateRegistrationRequest {
	t.Helper()
	descriptor := validFrozenDescriptor(t)
	request := CandidateRegistrationRequest{
		SchemaVersion: RegistrationRequestSchemaVersion, ArtifactLabel: RegistrationRequestLabel, FrozenCandidate: descriptor,
		QualificationVerdict: StatusQualified, QualificationReportID: descriptor.QualificationReportID, QualificationReportHash: descriptor.QualificationReportHash,
		CandidateImplementationIdentity: CandidateImplementationIdentity{CandidateID: descriptor.CandidateID, CandidateVersion: descriptor.CandidateVersion, ImplementationHash: descriptor.ImplementationHash, ConfigurationHash: descriptor.ConfigurationHash, ParameterHash: descriptor.ParameterHash, CapabilityHash: descriptor.CapabilityHash},
		ResearchIdentity:                ResearchIdentity{SchemaVersion: ResearchIdentitySchemaVersion, DatasetID: descriptor.DatasetID, DatasetVersion: descriptor.DatasetVersion, ResearchWindowStart: descriptor.ResearchWindowStart, ResearchWindowEnd: descriptor.ResearchWindowEnd, EvaluationCutoff: descriptor.EvaluationCutoff, ManifestID: descriptor.ManifestID, ManifestHash: descriptor.ManifestHash, AvailabilityPolicyVersion: descriptor.AvailabilityPolicyVersion, CoveragePolicyVersion: descriptor.CoveragePolicyVersion},
		RequestedLifecycleStartingState: "DISCOVERY", RIFAuthorized: false,
	}
	hash, err := request.CanonicalHash()
	if err != nil {
		t.Fatal(err)
	}
	request.ArtifactIntegrityHash = hash
	return request
}

func sha(char byte) string {
	return "sha256:" + strings.Repeat(string(char), 64)
}
