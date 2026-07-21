package epochorchestrator

import (
	"encoding/json"
	"strings"
	"time"

	"github.com/david22573/ak-engine/internal/partitionpipeline"
	"github.com/david22573/ak-engine/internal/preconditions"
	"github.com/david22573/ak-engine/internal/qualification"
	"github.com/david22573/ak-engine/internal/qualificationrunner"
	"github.com/david22573/ak-rif/research"
)

func CreateSyntheticConfig(sourceRoot string) (Config, error) {
	plans, err := partitionpipeline.CreateSyntheticCheckpointFixture(sourceRoot)
	if err != nil {
		return Config{}, err
	}
	v00 := qualificationrunner.V00Configuration()
	v01 := qualificationrunner.V00Configuration()
	v01.ContextAgreement = "REQUIRE_POSITIVE_BTC_ETH_CONTEXT"
	v02 := qualificationrunner.V00Configuration()
	v02.EventQuality = "STRICT_CENTER_VOLATILITY"
	v03 := qualificationrunner.V00Configuration()
	v03.CooldownMinutes = 60
	ledger, err := qualificationrunner.SealVariantLedger(qualificationrunner.VariantLedger{SchemaVersion: qualificationrunner.VariantLedgerVersion, MaximumVariants: 4, V00ID: "V00", Variants: []qualificationrunner.RegisteredVariant{{ID: "V00", Dimensions: []string{}, Configuration: v00}, {ID: "V01", Dimensions: []string{"context-agreement"}, Configuration: v01}, {ID: "V02", Dimensions: []string{"event-quality"}, Configuration: v02}, {ID: "V03", Dimensions: []string{"cooldown/independence"}, Configuration: v03}}, StabilityNeighborhoods: []qualificationrunner.StabilityNeighborhood{{VariantID: "V00", NeighborIDs: []string{"V01", "V02"}}, {VariantID: "V01", NeighborIDs: []string{"V00", "V02"}}, {VariantID: "V02", NeighborIDs: []string{"V00", "V01"}}, {VariantID: "V03", NeighborIDs: []string{"V01", "V02"}}}})
	if err != nil {
		return Config{}, err
	}
	protocol := json.RawMessage(`{"candidate_family":"phase12/DowntrendMidVolReliefLong240m","schema_version":"ak.engine.pr4b0_r1.protocol.v1","synthetic_fixture":true}`)
	protocolHash := byteHash(protocol)
	gateSet := qualification.AcceptedPR4B0GateSet()
	gateHash, err := qualification.PR4B0GateSetHash(gateSet)
	if err != nil {
		return Config{}, err
	}
	gateRefs, err := qualification.PR4B0GateIdentities(gateSet)
	if err != nil {
		return Config{}, err
	}
	gates := make([]research.HashIdentity, len(gateRefs))
	for i, ref := range gateRefs {
		gates[i] = research.HashIdentity{ID: ref.ArtifactID, SHA256: ref.SHA256}
	}
	independenceHash, err := preconditions.AcceptedIndependencePolicyHashV3(preconditions.AcceptedIndependencePolicyV3Default())
	if err != nil {
		return Config{}, err
	}
	uncertaintyHash, err := preconditions.AcceptedUncertaintyMethodHashV2(preconditions.AcceptedUncertaintyMethodV2())
	if err != nil {
		return Config{}, err
	}
	universe, err := qualificationrunner.V00UniverseContract()
	if err != nil {
		return Config{}, err
	}
	runner := research.RunnerImplementationIdentity{SchemaVersion: research.RunnerIdentityVersion, SourceCommit: strings.Repeat("5", 40), PackageIdentity: research.HashIdentity{ID: "ak.engine.qualificationrunner", SHA256: hashChar('a')}, BuildInputsSHA256: hashChar('b'), CompilerIdentity: "go1.synthetic linux/amd64", BuildModeIdentity: research.HashIdentity{ID: "trimpath-buildvcs-false", SHA256: hashChar('c')}, BinarySHA256: hashChar('d')}
	variants := make([]research.Variant, len(ledger.Variants))
	for i, v := range ledger.Variants {
		variants[i] = research.Variant{ID: v.ID, ConfigurationSHA256: v.ConfigurationSHA256, Dimensions: append([]string(nil), v.Dimensions...)}
	}
	neighbors := make([]research.StabilityNeighborhood, len(ledger.StabilityNeighborhoods))
	for i, n := range ledger.StabilityNeighborhoods {
		neighbors[i] = research.StabilityNeighborhood{VariantID: n.VariantID, NeighborIDs: append([]string(nil), n.NeighborIDs...)}
	}
	partitions := []research.Partition{}
	for _, name := range []string{"DEVELOPMENT", "VALIDATION", "FINAL_HOLDOUT"} {
		plan := plans[name]
		partitions = append(partitions, research.Partition{Name: research.PartitionName(name), Interval: research.Interval{Start: plan.PartitionInterval.Start, End: plan.PartitionInterval.End}, StructuralDayCount: plan.ExpectedStructuralDays, RequiredSymbolCoverageSHA256: plan.PlanSHA256})
	}
	identity := research.IdentityV4{SchemaVersion: research.IdentitySchemaVersion, ResearchID: "synthetic-pr4b0-r1p8-drill", Repositories: research.RepositoryIdentity{EngineStartingCommit: strings.Repeat("1", 40), HistorianStartingCommit: strings.Repeat("2", 40), RIFStartingCommit: strings.Repeat("3", 40), ProtocolGitCommit: strings.Repeat("4", 40), RunnerGitCommit: runner.SourceCommit, RunnerExecutableSHA256: runner.BinarySHA256}, Protocol: research.ProtocolIdentity{ID: "synthetic-pr4b0-r1-protocol", SHA256: protocolHash, ContentAddressedIdentity: protocolHash, SchemaVersion: "ak.engine.pr4b0_r1.protocol.v1"}, CandidateScope: research.CandidateScope{FamilyID: qualificationrunner.V00CandidateFamily, StrategySide: "LONG", Horizon: "240m", SemanticsFrozen: false}, Dataset: research.DatasetIdentity{Checkpoint: research.HashIdentity{ID: plans["DEVELOPMENT"].Checkpoint.ID, SHA256: plans["DEVELOPMENT"].Checkpoint.SHA256}, SourceIdentitySHA256: plans["DEVELOPMENT"].SourceIdentitySHA256, ReacquisitionProtocol: research.HashIdentity{ID: plans["DEVELOPMENT"].ReacquisitionProtocol.ID, SHA256: plans["DEVELOPMENT"].ReacquisitionProtocol.SHA256}, PreAcquisitionSealSHA256: plans["DEVELOPMENT"].PreAcquisitionSealSHA256, SealedBinarySHA256: plans["DEVELOPMENT"].SealedBinarySHA256, AbandonedEvidenceRegistry: research.HashIdentity{ID: plans["DEVELOPMENT"].AbandonedEvidenceRegistry.ID, SHA256: plans["DEVELOPMENT"].AbandonedEvidenceRegistry.SHA256}, HistorianCheckpointCommit: strings.Repeat("6", 40), RequiredSymbols: universe.DatasetRequiredSymbols, CandidateTargetSymbols: universe.CandidateTargetSymbols, ContextOnlySymbols: universe.ContextOnlySymbols, UniverseContractSHA256: universe.ContractSHA256, EligibleInterval: research.Interval{Start: time.Date(2030, 1, 1, 0, 0, 0, 0, time.UTC), End: time.Date(2033, 1, 1, 0, 0, 0, 0, time.UTC)}, ProhibitedPriorExposure: []research.Interval{{Start: time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC), End: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)}}, AvailabilityCutoff: time.Date(2033, 1, 2, 0, 0, 0, 0, time.UTC)}, Partitions: partitions, VariantLedger: research.VariantLedger{Variants: variants, MaximumRegisteredVariants: 4, V00ID: "V00", PermittedDimensions: []string{"context-agreement", "cooldown/independence", "event-quality"}, DevelopmentNomineeRule: "LOWEST_NUMERIC_VARIANT_ID_PASSING_ALL_MANDATORY_DEVELOPMENT_GATES", StabilityNeighborhoods: neighbors}, Authorities: research.AuthorityIdentity{Independence: research.HashIdentity{ID: preconditions.AcceptedIndependencePolicyVersionV3, SHA256: independenceHash}, Uncertainty: research.HashIdentity{ID: preconditions.AcceptedUncertaintyMethodVersion, SHA256: uncertaintyHash}, ConcentrationSHA256: preconditions.DefaultConcentrationGovernanceDecisionV3().CanonicalDecisionHash, QualificationGateSet: research.HashIdentity{ID: qualification.PR4B0GateSetID, SHA256: gateHash}, QualificationGateHashes: gates, TransactionCostPolicy: research.HashIdentity{ID: "synthetic-cost-policy", SHA256: hashChar('e')}, DeterministicSeedPolicy: research.HashIdentity{ID: "synthetic-seed-policy", SHA256: hashChar('f')}}, AccessPolicy: research.AccessPolicy{NoAccessBeforeReservation: true, DevelopmentPrerequisites: []string{"exact runner", "holdout reservation"}, ValidationPrerequisites: []string{"development sealed", "registered nominee"}, FinalHoldoutPrerequisites: []string{"candidate frozen", "validation sealed"}, CandidateFreezeRequirements: []string{"dataset", "executable", "no unresolved defaults"}, PermittedAccessCountPerPartition: 1, RetryPolicy: "NO_RETRY_AFTER_ACCESS", DurableAccessReceiptRequired: true}}
	cfg := Config{SchemaVersion: ConfigSchemaVersion, Synthetic: true, Repositories: map[string]RepositoryCheck{}, Protocol: protocol, Identity: identity, VariantLedger: ledger, Runner: runner, Plans: plans}
	return SealConfig(cfg)
}

func hashChar(value byte) string { return "sha256:" + strings.Repeat(string(value), 64) }
