package qualificationrunner

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"reflect"
	"sort"
	"strings"
	"time"

	"github.com/david22573/ak-engine/internal/preconditions"
	"github.com/david22573/ak-engine/internal/qualification"
	"github.com/david22573/ak-engine/internal/rifbridge"
)

func Verify(request ExecutionRequest) (VerifiedRequest, ReadinessArtifact, error) {
	verified, err := verifyRequest(request, false)
	if err != nil {
		return VerifiedRequest{}, ReadinessArtifact{}, err
	}
	readiness := ReadinessArtifact{
		SchemaVersion: ReadinessSchemaVersion, Label: NoOutcomesLabel, Mode: request.Mode,
		ResearchIdentitySHA256: request.RIF.Snapshot.IdentityHash,
		ReservationID:          request.RIF.Snapshot.Reservation.ReservationID,
		VariantID:              request.VariantID, ConfigurationSHA256: request.ConfigurationSHA256,
		GateSetSHA256:          verified.GateSetSHA256,
		AuthorityHashes:        []string{request.Independence.SHA256, request.Uncertainty.SHA256, request.Concentration.SHA256},
		RunnerExecutableSHA256: request.Runner.ExecutableSHA256, Partition: request.Partition.Name,
		DataLoads: 0, CandidateEvents: 0, CandidateOutcomes: 0,
	}
	sort.Strings(readiness.AuthorityHashes)
	hash, err := readinessHash(readiness)
	if err != nil {
		return VerifiedRequest{}, ReadinessArtifact{}, err
	}
	readiness.ArtifactSHA256 = hash
	return verified, readiness, nil
}

func verifyRequest(request ExecutionRequest, requireConsumed bool) (VerifiedRequest, error) {
	if request.SchemaVersion != RequestSchemaVersion {
		return VerifiedRequest{}, errors.New("unsupported qualification execution request schema")
	}
	if request.Mode != ModeVerify && request.Mode != ModeDevelopment && request.Mode != ModeValidation && request.Mode != ModeFinalHoldout {
		return VerifiedRequest{}, fmt.Errorf("unknown runner mode %q", request.Mode)
	}
	if err := rifbridge.VerifyResearchGovernanceEnvelope(request.RIF); err != nil {
		return VerifiedRequest{}, fmt.Errorf("RIF governance: %w", err)
	}
	identity, identityHash, err := decodeIdentity(request.RIF.Snapshot.Identity)
	if err != nil {
		return VerifiedRequest{}, err
	}
	if identityHash != request.RIF.Snapshot.IdentityHash {
		return VerifiedRequest{}, errors.New("RIF research identity hash mismatch")
	}
	if err := validateIdentity(identity); err != nil {
		return VerifiedRequest{}, err
	}
	if !reflect.DeepEqual(request.Protocol, identity.Protocol) || request.Protocol.SHA256 != request.Protocol.ContentAddressedIdentity {
		return VerifiedRequest{}, errors.New("protocol identity substitution")
	}
	if request.CandidateFamily != identity.CandidateScope.FamilyID || request.CandidateFamily != V00CandidateFamily || identity.CandidateScope.StrategySide != "LONG" || identity.CandidateScope.Horizon != "240m" || identity.CandidateScope.SemanticsFrozen {
		return VerifiedRequest{}, errors.New("candidate family, side, horizon, or reservation-time semantics mismatch")
	}
	if !reflect.DeepEqual(request.Dataset, DatasetBinding{identity.Dataset.Checkpoint, identity.Dataset.SourceIdentitySHA256, identity.Dataset.SealedBinarySHA256, identity.Dataset.RequiredSymbols, identity.Dataset.EligibleInterval, identity.Dataset.ProhibitedPriorExposure, identity.Dataset.AvailabilityCutoff}) {
		return VerifiedRequest{}, errors.New("dataset, source, interval, symbol, cutoff, or barred-exposure substitution")
	}
	registeredPartition, ok := findPartition(identity, request.Partition.Name)
	if !ok || !reflect.DeepEqual(request.Partition, registeredPartition) {
		return VerifiedRequest{}, errors.New("partition interval or structural identity substitution")
	}
	if request.Partition.Interval.Start.Before(time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)) {
		return VerifiedRequest{}, errors.New("pre-2026 research partition is prohibited")
	}
	ledger, err := ResolveVariantLedger(request.VariantLedger, identity.VariantLedger)
	if err != nil {
		return VerifiedRequest{}, err
	}
	selected, ok := findVariant(ledger, request.VariantID)
	if !ok || selected.ConfigurationSHA256 != request.ConfigurationSHA256 {
		return VerifiedRequest{}, errors.New("unregistered variant or configuration hash")
	}
	rifLedgerHash, err := canonicalHash(identity.VariantLedger)
	if err != nil || request.RIF.Snapshot.Reservation.VariantLedgerSHA256 != rifLedgerHash {
		return VerifiedRequest{}, errors.New("RIF reservation variant-ledger binding mismatch")
	}
	rifAuthorityHash, err := canonicalHash(identity.Authorities)
	if err != nil || request.RIF.Snapshot.Reservation.AuthoritySetSHA256 != rifAuthorityHash {
		return VerifiedRequest{}, errors.New("RIF reservation authority-set binding mismatch")
	}
	if request.RIF.Snapshot.Reservation.ProtocolSHA256 != identity.Protocol.SHA256 || request.RIF.Snapshot.Reservation.CheckpointSHA256 != identity.Dataset.Checkpoint.SHA256 {
		return VerifiedRequest{}, errors.New("RIF reservation protocol or checkpoint binding mismatch")
	}

	policy := preconditions.AcceptedIndependencePolicyV3Default()
	independenceHash, err := preconditions.AcceptedIndependencePolicyHashV3(policy)
	if err != nil {
		return VerifiedRequest{}, err
	}
	method := preconditions.AcceptedUncertaintyMethodV2()
	uncertaintyHash, err := preconditions.AcceptedUncertaintyMethodHashV2(method)
	if err != nil {
		return VerifiedRequest{}, err
	}
	concentrationHash := preconditions.DefaultConcentrationGovernanceDecisionV3().CanonicalDecisionHash
	if request.Independence.ID != preconditions.AcceptedIndependencePolicyVersionV3 || request.Independence.SHA256 != independenceHash || !reflect.DeepEqual(request.Independence, identity.Authorities.Independence) {
		return VerifiedRequest{}, errors.New("exact accepted independence V3 implementation is required")
	}
	if request.Uncertainty.ID != preconditions.AcceptedUncertaintyMethodVersion || request.Uncertainty.SHA256 != uncertaintyHash || !reflect.DeepEqual(request.Uncertainty, identity.Authorities.Uncertainty) {
		return VerifiedRequest{}, errors.New("exact accepted uncertainty V2 implementation is required")
	}
	if request.Concentration.ID == "" || request.Concentration.SHA256 != concentrationHash || identity.Authorities.ConcentrationSHA256 != concentrationHash {
		return VerifiedRequest{}, errors.New("exact accepted concentration governance implementation is required")
	}
	gates := qualification.AcceptedPR4B0GateSet()
	gateHash, err := qualification.PR4B0GateSetHash(gates)
	if err != nil {
		return VerifiedRequest{}, err
	}
	if request.QualificationGateSet.ID != qualification.PR4B0GateSetID || request.QualificationGateSet.SHA256 != gateHash || !reflect.DeepEqual(request.QualificationGateSet, identity.Authorities.QualificationGateSet) {
		return VerifiedRequest{}, errors.New("accepted PR4B0 gate-set mismatch")
	}
	gateIdentities, err := qualification.PR4B0GateIdentities(gates)
	if err != nil {
		return VerifiedRequest{}, err
	}
	if !gateHashesEqual(gateIdentities, identity.Authorities.QualificationGateHashes) {
		return VerifiedRequest{}, errors.New("qualification gate identity list mismatch")
	}
	if !reflect.DeepEqual(request.CostPolicy, identity.Authorities.TransactionCostPolicy) || !reflect.DeepEqual(request.SeedPolicy, identity.Authorities.DeterministicSeedPolicy) {
		return VerifiedRequest{}, errors.New("cost or deterministic-seed policy substitution")
	}
	if request.Runner.GitCommit != identity.Repositories.RunnerGitCommit || request.Runner.ExecutableSHA256 != identity.Repositories.RunnerExecutableSHA256 || request.Runner.V00SourceSHA256 != V00SourceSHA256 {
		return VerifiedRequest{}, errors.New("runner identity or V00 executable source mismatch")
	}
	if err := validateModeAuthority(request, selected, requireConsumed); err != nil {
		return VerifiedRequest{}, err
	}
	return VerifiedRequest{Request: request, Identity: identity, Variant: selected, Gates: gates, GateSetSHA256: gateHash}, nil
}

func validateModeAuthority(request ExecutionRequest, selected RegisteredVariant, requireConsumed bool) error {
	expectedState, expectedPartition := "", ""
	switch request.Mode {
	case ModeVerify:
		if request.RIF.Snapshot.State == "RESEARCH_IDENTITY_REGISTERED" {
			return errors.New("verify requires a durable holdout reservation")
		}
		return nil
	case ModeDevelopment:
		expectedState, expectedPartition = "DEVELOPMENT_AUTHORIZED", "DEVELOPMENT"
	case ModeValidation:
		expectedState, expectedPartition = "VALIDATION_AUTHORIZED", "VALIDATION"
	case ModeFinalHoldout:
		expectedState, expectedPartition = "FINAL_HOLDOUT_AUTHORIZED", "FINAL_HOLDOUT"
	}
	if request.Partition.Name != expectedPartition || request.RIF.Snapshot.State != expectedState || request.RIF.Authorization == nil {
		return errors.New("runner mode lacks exact RIF lifecycle state and partition authorization")
	}
	authorization := request.RIF.Authorization
	binding := authorization.Binding
	if authorization.LifecycleState != expectedState || binding.Partition != expectedPartition || binding.VariantID != request.VariantID || binding.ConfigurationSHA256 != request.ConfigurationSHA256 || binding.ProtocolSHA256 != request.Protocol.SHA256 || binding.CheckpointSHA256 != request.Dataset.Checkpoint.SHA256 || binding.IndependenceSHA256 != request.Independence.SHA256 || binding.UncertaintySHA256 != request.Uncertainty.SHA256 || binding.ConcentrationSHA256 != request.Concentration.SHA256 || binding.QualificationGateSHA256 != request.QualificationGateSet.SHA256 || binding.RunnerGitCommit != request.Runner.GitCommit || binding.RunnerExecutableSHA256 != request.Runner.ExecutableSHA256 {
		return errors.New("RIF partition authorization does not bind exact execution request")
	}
	if authorization.ExpiresAt != nil && !time.Now().Before(authorization.ExpiresAt.UTC()) {
		return errors.New("RIF partition authorization is expired")
	}
	matchingReceipts := 0
	for _, receipt := range request.RIF.Snapshot.AccessReceipts {
		if receipt.AuthorizationID == authorization.AuthorizationID && reflect.DeepEqual(receipt.Binding, authorization.Binding) {
			matchingReceipts++
		}
	}
	if requireConsumed && matchingReceipts != 1 {
		return errors.New("execution requires exactly one durable RIF access receipt")
	}
	if !requireConsumed && matchingReceipts != 0 {
		return errors.New("dry verification requires an unconsumed authorization shape")
	}
	if request.Mode == ModeFinalHoldout {
		frozen := request.RIF.Snapshot.FrozenCandidate
		if frozen == nil || frozen.VariantID != selected.ID || frozen.ConfigurationSHA256 != selected.ConfigurationSHA256 || frozen.ExecutableSHA256 != request.Runner.ExecutableSHA256 || frozen.ProtocolSHA256 != request.Protocol.SHA256 || frozen.CheckpointSHA256 != request.Dataset.Checkpoint.SHA256 || frozen.IndependenceSHA256 != request.Independence.SHA256 || frozen.UncertaintySHA256 != request.Uncertainty.SHA256 || frozen.ConcentrationSHA256 != request.Concentration.SHA256 || frozen.QualificationGateSHA256 != request.QualificationGateSet.SHA256 || !frozen.NoUnresolvedDefaults {
			return errors.New("FINAL_HOLDOUT requires the exact frozen candidate; neighbors and alternatives are prohibited")
		}
	}
	return nil
}

func decodeIdentity(data json.RawMessage) (ResearchIdentityV4, string, error) {
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()
	var identity ResearchIdentityV4
	if err := decoder.Decode(&identity); err != nil {
		return ResearchIdentityV4{}, "", fmt.Errorf("decode RIF V4 research identity: %w", err)
	}
	var trailing any
	if err := decoder.Decode(&trailing); err != io.EOF {
		return ResearchIdentityV4{}, "", errors.New("RIF V4 research identity has trailing data")
	}
	var compact bytes.Buffer
	if err := json.Compact(&compact, data); err != nil {
		return ResearchIdentityV4{}, "", err
	}
	digest := sha256.Sum256(compact.Bytes())
	return identity, "sha256:" + hex.EncodeToString(digest[:]), nil
}

func validateIdentity(identity ResearchIdentityV4) error {
	if identity.SchemaVersion != "ak.rif.research_identity.v4" || identity.ResearchID == "" || identity.Protocol.ID == "" || !validSHA(identity.Protocol.SHA256) || identity.Protocol.ContentAddressedIdentity != identity.Protocol.SHA256 || identity.Protocol.SchemaVersion == "" {
		return errors.New("complete RIF V4 protocol identity is required")
	}
	for _, commit := range []string{identity.Repositories.EngineStartingCommit, identity.Repositories.HistorianStartingCommit, identity.Repositories.RIFStartingCommit, identity.Repositories.ProtocolGitCommit, identity.Repositories.RunnerGitCommit} {
		if len(commit) != 40 {
			return errors.New("complete repository commit identity is required")
		}
	}
	if !validSHA(identity.Repositories.RunnerExecutableSHA256) || !validSHA(identity.Dataset.Checkpoint.SHA256) || !validSHA(identity.Dataset.SourceIdentitySHA256) || !validSHA(identity.Dataset.SealedBinarySHA256) || len(identity.Dataset.RequiredSymbols) == 0 || len(identity.Partitions) != 3 || identity.VariantLedger.MaximumRegisteredVariants > 12 || len(identity.VariantLedger.Variants) == 0 || !identity.AccessPolicy.NoAccessBeforeReservation || !identity.AccessPolicy.DurableAccessReceiptRequired || identity.AccessPolicy.PermittedAccessCountPerPartition != 1 {
		return errors.New("RIF V4 repository, dataset, partition, ledger, or access identity is incomplete")
	}
	if !reflect.DeepEqual(identity.Dataset.RequiredSymbols, acceptedSymbols) {
		return errors.New("registered symbol universe differs from accepted PR4B0 gate universe")
	}
	return nil
}

func findPartition(identity ResearchIdentityV4, name string) (Partition, bool) {
	for _, item := range identity.Partitions {
		if item.Name == name {
			return item, true
		}
	}
	return Partition{}, false
}
func findVariant(ledger VariantLedger, id string) (RegisteredVariant, bool) {
	for _, item := range ledger.Variants {
		if item.ID == id {
			return item, true
		}
	}
	return RegisteredVariant{}, false
}

func gateHashesEqual(expected []qualification.EvidenceReference, actual []HashIdentity) bool {
	if len(expected) != len(actual) {
		return false
	}
	want := make([]HashIdentity, len(expected))
	copyActual := append([]HashIdentity(nil), actual...)
	for i, item := range expected {
		want[i] = HashIdentity{item.ArtifactID, item.SHA256}
	}
	sort.Slice(want, func(i, j int) bool { return want[i].ID < want[j].ID })
	sort.Slice(copyActual, func(i, j int) bool { return copyActual[i].ID < copyActual[j].ID })
	return reflect.DeepEqual(want, copyActual)
}

func readinessHash(value ReadinessArtifact) (string, error) {
	value.ArtifactSHA256 = ""
	return canonicalHash(value)
}
func validSHA(value string) bool {
	if len(value) != 71 || !strings.HasPrefix(value, "sha256:") || value != strings.ToLower(value) {
		return false
	}
	_, err := hex.DecodeString(strings.TrimPrefix(value, "sha256:"))
	return err == nil
}
