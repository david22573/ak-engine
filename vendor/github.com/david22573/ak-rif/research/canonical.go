package research

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"reflect"
	"sort"
	"strings"
	"time"
	"unicode"
)

var permittedDimensions = []string{"context-agreement", "cooldown/independence", "event-quality"}

func CanonicalizeIdentity(identity IdentityV4) (IdentityV4, error) {
	if err := ValidateIdentity(identity); err != nil {
		return IdentityV4{}, err
	}
	out := cloneIdentity(identity)
	out.Dataset.EligibleInterval = normalizeInterval(out.Dataset.EligibleInterval)
	out.Dataset.AvailabilityCutoff = out.Dataset.AvailabilityCutoff.UTC()
	for i := range out.Dataset.ProhibitedPriorExposure {
		out.Dataset.ProhibitedPriorExposure[i] = normalizeInterval(out.Dataset.ProhibitedPriorExposure[i])
	}
	sort.Slice(out.Dataset.ProhibitedPriorExposure, func(i, j int) bool {
		return out.Dataset.ProhibitedPriorExposure[i].Start.Before(out.Dataset.ProhibitedPriorExposure[j].Start)
	})
	sort.Strings(out.Dataset.RequiredSymbols)
	sort.Strings(out.Dataset.CandidateTargetSymbols)
	sort.Strings(out.Dataset.ContextOnlySymbols)
	for i := range out.Partitions {
		out.Partitions[i].Interval = normalizeInterval(out.Partitions[i].Interval)
	}
	sort.Slice(out.Partitions, func(i, j int) bool { return out.Partitions[i].Name < out.Partitions[j].Name })
	for i := range out.VariantLedger.Variants {
		sort.Strings(out.VariantLedger.Variants[i].Dimensions)
	}
	sort.Slice(out.VariantLedger.Variants, func(i, j int) bool { return out.VariantLedger.Variants[i].ID < out.VariantLedger.Variants[j].ID })
	sort.Strings(out.VariantLedger.PermittedDimensions)
	for i := range out.VariantLedger.StabilityNeighborhoods {
		sort.Strings(out.VariantLedger.StabilityNeighborhoods[i].NeighborIDs)
	}
	sort.Slice(out.VariantLedger.StabilityNeighborhoods, func(i, j int) bool {
		return out.VariantLedger.StabilityNeighborhoods[i].VariantID < out.VariantLedger.StabilityNeighborhoods[j].VariantID
	})
	sort.Slice(out.Authorities.QualificationGateHashes, func(i, j int) bool {
		return out.Authorities.QualificationGateHashes[i].ID < out.Authorities.QualificationGateHashes[j].ID
	})
	sort.Strings(out.AccessPolicy.DevelopmentPrerequisites)
	sort.Strings(out.AccessPolicy.ValidationPrerequisites)
	sort.Strings(out.AccessPolicy.FinalHoldoutPrerequisites)
	sort.Strings(out.AccessPolicy.CandidateFreezeRequirements)
	return out, nil
}

func HashIdentityV4(identity IdentityV4) (string, error) {
	canonical, err := CanonicalizeIdentity(identity)
	if err != nil {
		return "", err
	}
	return hashCanonical(canonical)
}

func HashVariantLedger(ledger VariantLedger) (string, error) {
	identity := IdentityV4{
		SchemaVersion: IdentitySchemaVersion, ResearchID: "canonical-ledger-validation",
		Repositories:   RepositoryIdentity{EngineStartingCommit: strings.Repeat("a", 40), HistorianStartingCommit: strings.Repeat("b", 40), RIFStartingCommit: strings.Repeat("c", 40), ProtocolGitCommit: strings.Repeat("d", 40), RunnerGitCommit: strings.Repeat("e", 40), RunnerExecutableSHA256: hashOf('a')},
		Protocol:       ProtocolIdentity{ID: "protocol", SHA256: hashOf('b'), ContentAddressedIdentity: hashOf('b'), SchemaVersion: "protocol.v1"},
		CandidateScope: CandidateScope{FamilyID: "family", StrategySide: "LONG", Horizon: "240m", SemanticsFrozen: false},
		Dataset:        DatasetIdentity{Checkpoint: HashIdentity{"checkpoint", hashOf('c')}, SourceIdentitySHA256: hashOf('d'), ReacquisitionProtocol: HashIdentity{"reacquisition", hashOf('e')}, PreAcquisitionSealSHA256: hashOf('f'), SealedBinarySHA256: hashOf('1'), AbandonedEvidenceRegistry: HashIdentity{"abandoned", hashOf('2')}, HistorianCheckpointCommit: strings.Repeat("f", 40), RequiredSymbols: []string{"SYNTH"}, CandidateTargetSymbols: []string{"SYNTH"}, ContextOnlySymbols: []string{}, UniverseContractSHA256: hashOf('a'), EligibleInterval: interval(2030, 1, 1, 2030, 4, 1), ProhibitedPriorExposure: []Interval{interval(2020, 1, 1, 2021, 1, 1)}, AvailabilityCutoff: time.Date(2030, 4, 2, 0, 0, 0, 0, time.UTC)},
		Partitions:     syntheticPartitions(), VariantLedger: ledger,
		Authorities: acceptedSyntheticAuthorities(), AccessPolicy: validAccessPolicy(),
	}
	canonical, err := CanonicalizeIdentity(identity)
	if err != nil {
		return "", fmt.Errorf("variant ledger: %w", err)
	}
	return hashCanonical(canonical.VariantLedger)
}

func HashAuthoritySet(authority AuthorityIdentity) (string, error) {
	copyAuthority := authority
	sort.Slice(copyAuthority.QualificationGateHashes, func(i, j int) bool {
		return copyAuthority.QualificationGateHashes[i].ID < copyAuthority.QualificationGateHashes[j].ID
	})
	if err := validateAuthority(copyAuthority); err != nil {
		return "", err
	}
	return hashCanonical(copyAuthority)
}

func ValidateIdentity(identity IdentityV4) error {
	if identity.SchemaVersion != IdentitySchemaVersion {
		return fmt.Errorf("unsupported research identity schema_version %q", identity.SchemaVersion)
	}
	if !validText(identity.ResearchID) {
		return errors.New("research_id is required")
	}
	for name, commit := range map[string]string{
		"engine_starting_commit":       identity.Repositories.EngineStartingCommit,
		"historian_starting_commit":    identity.Repositories.HistorianStartingCommit,
		"rif_starting_commit":          identity.Repositories.RIFStartingCommit,
		"protocol_git_commit":          identity.Repositories.ProtocolGitCommit,
		"evaluation_runner_git_commit": identity.Repositories.RunnerGitCommit,
	} {
		if !validHex(commit, 40) {
			return fmt.Errorf("%s must be an exact lowercase Git commit", name)
		}
	}
	if !validSHA256(identity.Repositories.RunnerExecutableSHA256) {
		return errors.New("evaluation_runner_executable_sha256 is required")
	}
	if !validText(identity.Protocol.ID) || !validSHA256(identity.Protocol.SHA256) || identity.Protocol.ContentAddressedIdentity != identity.Protocol.SHA256 || !validText(identity.Protocol.SchemaVersion) {
		return errors.New("complete content-addressed protocol identity is required")
	}
	if !validText(identity.CandidateScope.FamilyID) || (identity.CandidateScope.StrategySide != "LONG" && identity.CandidateScope.StrategySide != "SHORT") || !validText(identity.CandidateScope.Horizon) {
		return errors.New("candidate family, supported side, and horizon are required")
	}
	if identity.CandidateScope.SemanticsFrozen {
		return errors.New("candidate semantics must be explicitly not frozen at reservation identity registration")
	}
	if err := validateDataset(identity.Dataset); err != nil {
		return err
	}
	if err := validatePartitions(identity.Partitions, identity.Dataset.EligibleInterval); err != nil {
		return err
	}
	if err := validateVariantLedger(identity.VariantLedger); err != nil {
		return err
	}
	if err := validateAuthority(identity.Authorities); err != nil {
		return err
	}
	if err := validateAccessPolicy(identity.AccessPolicy); err != nil {
		return err
	}
	return nil
}

func validateDataset(dataset DatasetIdentity) error {
	if !validHashIdentity(dataset.Checkpoint) || !validSHA256(dataset.SourceIdentitySHA256) || !validHashIdentity(dataset.ReacquisitionProtocol) || !validSHA256(dataset.PreAcquisitionSealSHA256) || !validSHA256(dataset.SealedBinarySHA256) || !validHashIdentity(dataset.AbandonedEvidenceRegistry) || !validHex(dataset.HistorianCheckpointCommit, 40) || !validSHA256(dataset.UniverseContractSHA256) {
		return errors.New("dataset identity is incomplete or contains an invalid immutable hash")
	}
	if err := validateUniqueTexts("required_symbols", dataset.RequiredSymbols); err != nil {
		return err
	}
	if err := validateUniqueTexts("candidate_target_symbols", dataset.CandidateTargetSymbols); err != nil {
		return err
	}
	if len(dataset.ContextOnlySymbols) > 0 {
		if err := validateUniqueTexts("context_only_symbols", dataset.ContextOnlySymbols); err != nil {
			return err
		}
	}
	targets := make(map[string]struct{}, len(dataset.CandidateTargetSymbols))
	for _, symbol := range dataset.CandidateTargetSymbols {
		targets[symbol] = struct{}{}
	}
	union := make(map[string]struct{}, len(dataset.RequiredSymbols))
	for symbol := range targets {
		union[symbol] = struct{}{}
	}
	for _, symbol := range dataset.ContextOnlySymbols {
		if _, overlap := targets[symbol]; overlap {
			return errors.New("candidate target and context-only symbols overlap")
		}
		union[symbol] = struct{}{}
	}
	if len(union) != len(dataset.RequiredSymbols) {
		return errors.New("dataset-required symbols must equal the target/context-only union")
	}
	for _, symbol := range dataset.RequiredSymbols {
		if _, ok := union[symbol]; !ok {
			return errors.New("dataset-required symbols must equal the target/context-only union")
		}
	}
	if !validInterval(dataset.EligibleInterval) || dataset.EligibleInterval.Start.Before(time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)) {
		return errors.New("eligible interval must be a nonempty post-2025 UTC interval")
	}
	if len(dataset.ProhibitedPriorExposure) == 0 {
		return errors.New("at least one prohibited prior-exposure interval is required")
	}
	for _, prohibited := range dataset.ProhibitedPriorExposure {
		if !validInterval(prohibited) {
			return errors.New("prohibited prior-exposure interval is invalid")
		}
	}
	if dataset.AvailabilityCutoff.IsZero() || dataset.AvailabilityCutoff.Before(dataset.EligibleInterval.End) {
		return errors.New("availability cutoff must not precede the eligible interval end")
	}
	return nil
}

func validatePartitions(partitions []Partition, eligible Interval) error {
	if len(partitions) != 3 {
		return errors.New("exactly DEVELOPMENT, VALIDATION, and FINAL_HOLDOUT partitions are required")
	}
	byName := map[PartitionName]Partition{}
	for _, partition := range partitions {
		if partition.Name != PartitionDevelopment && partition.Name != PartitionValidation && partition.Name != PartitionFinalHoldout {
			return fmt.Errorf("unknown partition %q", partition.Name)
		}
		if _, duplicate := byName[partition.Name]; duplicate {
			return fmt.Errorf("duplicate partition %q", partition.Name)
		}
		if !validInterval(partition.Interval) || partition.StructuralDayCount <= 0 || !validSHA256(partition.RequiredSymbolCoverageSHA256) {
			return fmt.Errorf("partition %q is structurally incomplete", partition.Name)
		}
		if partition.Interval.Start.Before(eligible.Start) || partition.Interval.End.After(eligible.End) {
			return fmt.Errorf("partition %q lies outside eligible interval", partition.Name)
		}
		byName[partition.Name] = partition
	}
	development, dok := byName[PartitionDevelopment]
	validation, vok := byName[PartitionValidation]
	holdout, hok := byName[PartitionFinalHoldout]
	if !dok || !vok || !hok || development.Interval.End.After(validation.Interval.Start) || validation.Interval.End.After(holdout.Interval.Start) {
		return errors.New("registered partitions must be chronological and nonoverlapping")
	}
	return nil
}

func validateVariantLedger(ledger VariantLedger) error {
	if ledger.MaximumRegisteredVariants <= 0 || ledger.MaximumRegisteredVariants > MaxRegisteredVariants || len(ledger.Variants) == 0 || len(ledger.Variants) > ledger.MaximumRegisteredVariants {
		return errors.New("variant ledger must contain between one and twelve variants within its registered maximum")
	}
	wantDimensions := append([]string(nil), permittedDimensions...)
	gotDimensions := append([]string(nil), ledger.PermittedDimensions...)
	sort.Strings(gotDimensions)
	if !reflect.DeepEqual(gotDimensions, wantDimensions) {
		return errors.New("permitted configuration dimensions must be exactly context-agreement, event-quality, and cooldown/independence")
	}
	if ledger.V00ID != "V00" || !validText(ledger.DevelopmentNomineeRule) {
		return errors.New("V00 and a deterministic DEVELOPMENT nominee-selection rule are required")
	}
	seen := map[string]struct{}{}
	for _, variant := range ledger.Variants {
		if !validText(variant.ID) || !validSHA256(variant.ConfigurationSHA256) {
			return errors.New("every variant requires an immutable ID and canonical configuration hash")
		}
		if _, duplicate := seen[variant.ID]; duplicate {
			return fmt.Errorf("duplicate variant %q", variant.ID)
		}
		seen[variant.ID] = struct{}{}
		dimensionSeen := map[string]struct{}{}
		for _, dimension := range variant.Dimensions {
			if !validText(dimension) {
				return errors.New("configuration dimension is invalid")
			}
			if _, duplicate := dimensionSeen[dimension]; duplicate {
				return fmt.Errorf("duplicate configuration dimension %q", dimension)
			}
			dimensionSeen[dimension] = struct{}{}
			if !contains(permittedDimensions, dimension) {
				return fmt.Errorf("unsupported configuration dimension %q", dimension)
			}
		}
	}
	if _, ok := seen[ledger.V00ID]; !ok {
		return errors.New("V00 is not registered in the variant ledger")
	}
	for _, relationship := range ledger.StabilityNeighborhoods {
		if _, ok := seen[relationship.VariantID]; !ok || len(relationship.NeighborIDs) == 0 {
			return errors.New("stability relationship references an unknown variant or has no neighbors")
		}
		neighborSeen := map[string]struct{}{}
		for _, neighbor := range relationship.NeighborIDs {
			if neighbor == relationship.VariantID {
				return errors.New("variant cannot be its own stability neighbor")
			}
			if _, ok := seen[neighbor]; !ok {
				return fmt.Errorf("stability neighbor %q is not registered", neighbor)
			}
			if _, duplicate := neighborSeen[neighbor]; duplicate {
				return fmt.Errorf("duplicate stability neighbor %q", neighbor)
			}
			neighborSeen[neighbor] = struct{}{}
		}
	}
	return nil
}

func validateAuthority(authority AuthorityIdentity) error {
	if authority.Independence.ID != AcceptedIndependenceID || authority.Independence.SHA256 != AcceptedIndependenceHash {
		return errors.New("exact accepted independence V3 identity is required")
	}
	if authority.Uncertainty.ID != AcceptedUncertaintyID || authority.Uncertainty.SHA256 != AcceptedUncertaintyHash {
		return errors.New("exact accepted uncertainty V2 identity is required")
	}
	if authority.ConcentrationSHA256 != AcceptedConcentrationHash {
		return errors.New("exact accepted concentration-governance hash is required")
	}
	if !validHashIdentity(authority.QualificationGateSet) || len(authority.QualificationGateHashes) == 0 || !validHashIdentity(authority.TransactionCostPolicy) || !validHashIdentity(authority.DeterministicSeedPolicy) {
		return errors.New("gate-set, cost-policy, and deterministic-seed identities are mandatory")
	}
	seen := map[string]struct{}{}
	for _, gate := range authority.QualificationGateHashes {
		if !validHashIdentity(gate) {
			return errors.New("qualification gate identity is invalid")
		}
		if _, duplicate := seen[gate.ID]; duplicate {
			return errors.New("qualification gate identity is duplicated")
		}
		seen[gate.ID] = struct{}{}
	}
	return nil
}

func validateAccessPolicy(policy AccessPolicy) error {
	if !policy.NoAccessBeforeReservation || !policy.DurableAccessReceiptRequired || policy.PermittedAccessCountPerPartition != 1 || policy.RetryPolicy != "NO_RETRY_AFTER_ACCESS" {
		return errors.New("access policy must fail closed with durable one-shot receipts and no retry after access")
	}
	for name, values := range map[string][]string{"development": policy.DevelopmentPrerequisites, "validation": policy.ValidationPrerequisites, "final_holdout": policy.FinalHoldoutPrerequisites, "candidate_freeze": policy.CandidateFreezeRequirements} {
		if err := validateUniqueTexts(name+" prerequisites", values); err != nil {
			return err
		}
	}
	return nil
}

func partition(identity IdentityV4, name PartitionName) Partition {
	for _, item := range identity.Partitions {
		if item.Name == name {
			return item
		}
	}
	return Partition{}
}

func variant(identity IdentityV4, id string) (Variant, bool) {
	for _, item := range identity.VariantLedger.Variants {
		if item.ID == id {
			return item, true
		}
	}
	return Variant{}, false
}

func normalizeInterval(value Interval) Interval {
	return Interval{Start: value.Start.UTC(), End: value.End.UTC()}
}
func validInterval(value Interval) bool {
	return !value.Start.IsZero() && !value.End.IsZero() && value.Start.Equal(value.Start.UTC()) && value.End.Equal(value.End.UTC()) && value.Start.Before(value.End)
}

func validHashIdentity(value HashIdentity) bool {
	return validText(value.ID) && validSHA256(value.SHA256)
}
func validSHA256(value string) bool {
	return len(value) == 71 && strings.HasPrefix(value, "sha256:") && value == strings.ToLower(value) && validHex(strings.TrimPrefix(value, "sha256:"), 64)
}
func validHex(value string, length int) bool {
	if len(value) != length || value != strings.ToLower(value) {
		return false
	}
	_, err := hex.DecodeString(value)
	return err == nil
}
func validText(value string) bool {
	if value == "" || len(value) > 512 || strings.TrimSpace(value) != value {
		return false
	}
	for _, r := range value {
		if unicode.IsControl(r) {
			return false
		}
	}
	return true
}

func validateUniqueTexts(name string, values []string) error {
	if len(values) == 0 {
		return fmt.Errorf("%s must be explicit and nonempty", name)
	}
	seen := map[string]struct{}{}
	for _, value := range values {
		if !validText(value) {
			return fmt.Errorf("%s contains an invalid value", name)
		}
		if _, duplicate := seen[value]; duplicate {
			return fmt.Errorf("%s contains duplicate %q", name, value)
		}
		seen[value] = struct{}{}
	}
	return nil
}

func contains(values []string, want string) bool {
	for _, value := range values {
		if value == want {
			return true
		}
	}
	return false
}
func hashCanonical(value any) (string, error) {
	data, err := json.Marshal(value)
	if err != nil {
		return "", err
	}
	digest := sha256.Sum256(data)
	return "sha256:" + hex.EncodeToString(digest[:]), nil
}
func hashOf(char byte) string { return "sha256:" + strings.Repeat(string(char), 64) }

func cloneIdentity(identity IdentityV4) IdentityV4 {
	data, _ := json.Marshal(identity)
	var out IdentityV4
	_ = json.Unmarshal(data, &out)
	return out
}

func interval(y1 int, m1 time.Month, d1, y2 int, m2 time.Month, d2 int) Interval {
	return Interval{Start: time.Date(y1, m1, d1, 0, 0, 0, 0, time.UTC), End: time.Date(y2, m2, d2, 0, 0, 0, 0, time.UTC)}
}
func syntheticPartitions() []Partition {
	return []Partition{{PartitionDevelopment, interval(2030, 1, 1, 2030, 2, 1), 31, hashOf('3')}, {PartitionValidation, interval(2030, 2, 1, 2030, 3, 1), 28, hashOf('4')}, {PartitionFinalHoldout, interval(2030, 3, 1, 2030, 4, 1), 31, hashOf('5')}}
}
func acceptedSyntheticAuthorities() AuthorityIdentity {
	return AuthorityIdentity{HashIdentity{AcceptedIndependenceID, AcceptedIndependenceHash}, HashIdentity{AcceptedUncertaintyID, AcceptedUncertaintyHash}, AcceptedConcentrationHash, HashIdentity{"gates", hashOf('6')}, []HashIdentity{{"gate", hashOf('7')}}, HashIdentity{"cost", hashOf('8')}, HashIdentity{"seed", hashOf('9')}}
}
func validAccessPolicy() AccessPolicy {
	return AccessPolicy{true, []string{"reservation"}, []string{"development sealed"}, []string{"candidate frozen"}, []string{"exact identities"}, 1, "NO_RETRY_AFTER_ACCESS", true}
}
