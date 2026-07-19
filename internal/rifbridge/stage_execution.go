package rifbridge

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"reflect"
	"strings"
	"time"
)

const (
	StageExecutionPlanSchemaVersion      = "ak.rif.stage_execution_plan.v1"
	StageExecutionSetSchemaVersion       = "ak.rif.stage_execution_set.v1"
	StageAuthorizationSchemaVersion      = "ak.rif.stage_variant_authorization.v1"
	StageAccessReceiptSchemaVersion      = "ak.rif.stage_variant_access_receipt.v1"
	StageExecutionReceiptSchemaVersion   = "ak.rif.stage_execution_receipt.v1"
	StageRetryProofSchemaVersion         = "ak.rif.zero_access_retry_proof.v1"
	StageManifestSchemaVersion           = "ak.rif.stage_completion_manifest.v1"
	StageEnvelopeSchemaVersion           = "ak.rif.stage_execution_envelope.v1"
	RunnerImplementationSchemaVersion    = "ak.rif.runner_implementation_identity.v1"
	RegisteredConfigurationSchemaVersion = "ak.rif.registered_configuration_identity.v1"
	NomineeSelectionSchemaVersion        = "ak.rif.development_nominee_selection.v1"
)

type StageHashIdentity struct {
	ID     string `json:"id"`
	SHA256 string `json:"sha256"`
}

type StageInterval struct {
	Start time.Time `json:"start_inclusive"`
	End   time.Time `json:"end_exclusive"`
}

type StagePartition struct {
	Name                         string        `json:"name"`
	Interval                     StageInterval `json:"interval"`
	StructuralDayCount           int           `json:"structural_day_count"`
	RequiredSymbolCoverageSHA256 string        `json:"required_symbol_coverage_sha256"`
}

type StageProtocolIdentity struct {
	ID                       string `json:"id"`
	SHA256                   string `json:"sha256"`
	ContentAddressedIdentity string `json:"content_addressed_identity"`
	SchemaVersion            string `json:"schema_version"`
}

type StageAuthorityIdentity struct {
	Independence            StageHashIdentity   `json:"independence"`
	Uncertainty             StageHashIdentity   `json:"uncertainty"`
	ConcentrationSHA256     string              `json:"concentration_governance_sha256"`
	QualificationGateSet    StageHashIdentity   `json:"qualification_gate_set"`
	QualificationGateHashes []StageHashIdentity `json:"qualification_gate_hashes"`
	TransactionCostPolicy   StageHashIdentity   `json:"transaction_cost_policy"`
	DeterministicSeedPolicy StageHashIdentity   `json:"deterministic_seed_policy"`
}

type RunnerImplementationIdentity struct {
	SchemaVersion     string            `json:"schema_version"`
	SourceCommit      string            `json:"source_commit"`
	PackageIdentity   StageHashIdentity `json:"package_identity"`
	BuildInputsSHA256 string            `json:"deterministic_build_inputs_sha256"`
	CompilerIdentity  string            `json:"compiler_identity"`
	BuildModeIdentity StageHashIdentity `json:"build_mode_identity"`
	BinarySHA256      string            `json:"runner_binary_sha256"`
}

type RegisteredConfigurationIdentity struct {
	SchemaVersion          string          `json:"schema_version"`
	VariantID              string          `json:"variant_id"`
	CanonicalConfiguration json.RawMessage `json:"canonical_configuration"`
	ConfigurationSHA256    string          `json:"canonical_configuration_sha256"`
	CandidateFamilyID      string          `json:"candidate_family_id"`
	ProtocolID             string          `json:"protocol_id"`
	ProtocolSHA256         string          `json:"protocol_sha256"`
}

type StageExecutionPlan struct {
	SchemaVersion           string                            `json:"schema_version"`
	ResearchIdentityHash    string                            `json:"research_identity_hash"`
	Protocol                StageProtocolIdentity             `json:"protocol_identity"`
	Stage                   string                            `json:"stage"`
	Partition               StagePartition                    `json:"partition"`
	Checkpoint              StageHashIdentity                 `json:"checkpoint"`
	DatasetIdentitySHA256   string                            `json:"dataset_identity_sha256"`
	Runner                  RunnerImplementationIdentity      `json:"runner_identity"`
	Configurations          []RegisteredConfigurationIdentity `json:"ordered_registered_configurations"`
	DeterministicSeedPolicy StageHashIdentity                 `json:"deterministic_seed_policy"`
	ExpectedExecutions      int                               `json:"expected_execution_cardinality"`
	Complete                bool                              `json:"complete_set_required"`
	OrderingRule            string                            `json:"deterministic_ordering_rule"`
	Authorities             StageAuthorityIdentity            `json:"authority_identities"`
	GateSet                 StageHashIdentity                 `json:"gate_set_identity"`
	PlanHash                string                            `json:"plan_hash"`
}

type StageVariantAuthorization struct {
	SchemaVersion   string                          `json:"schema_version"`
	AuthorizationID string                          `json:"authorization_id"`
	ExecutionSetID  string                          `json:"execution_set_id"`
	PlanHash        string                          `json:"plan_hash"`
	Ordinal         int                             `json:"ordinal"`
	Attempt         int                             `json:"attempt"`
	Configuration   RegisteredConfigurationIdentity `json:"registered_configuration"`
	Runner          RunnerImplementationIdentity    `json:"runner_identity"`
	Partition       StagePartition                  `json:"partition"`
	Protocol        StageProtocolIdentity           `json:"protocol_identity"`
	Checkpoint      StageHashIdentity               `json:"checkpoint"`
	Authorities     StageAuthorityIdentity          `json:"authority_identities"`
	GateSet         StageHashIdentity               `json:"gate_set_identity"`
	IssuedAt        time.Time                       `json:"issued_at"`
	PriorStateHash  string                          `json:"prior_state_hash"`
	PreviousHash    string                          `json:"previous_hash"`
	RecordHash      string                          `json:"record_hash"`
}

type StageVariantAccessReceipt struct {
	SchemaVersion   string    `json:"schema_version"`
	ExecutionSetID  string    `json:"execution_set_id"`
	AuthorizationID string    `json:"authorization_id"`
	VariantID       string    `json:"variant_id"`
	Attempt         int       `json:"attempt"`
	ConsumedAt      time.Time `json:"consumed_at"`
	PriorStateHash  string    `json:"prior_state_hash"`
	PreviousHash    string    `json:"previous_hash"`
	RecordHash      string    `json:"record_hash"`
}

type ZeroAccessRetryProof struct {
	SchemaVersion          string    `json:"schema_version"`
	ExecutionSetID         string    `json:"execution_set_id"`
	PriorAuthorizationID   string    `json:"prior_authorization_id"`
	PriorAccessReceiptHash string    `json:"prior_access_receipt_hash"`
	VariantID              string    `json:"variant_id"`
	RowsAccessed           int       `json:"rows_accessed"`
	OutcomeArtifacts       int       `json:"outcome_artifacts"`
	DurableProofSHA256     string    `json:"durable_proof_sha256"`
	ProvenAt               time.Time `json:"proven_at"`
	PreviousHash           string    `json:"previous_hash"`
	RecordHash             string    `json:"record_hash"`
}

type StageExecutionReceipt struct {
	SchemaVersion           string    `json:"schema_version"`
	ExecutionSetID          string    `json:"execution_set_id"`
	PlanHash                string    `json:"plan_hash"`
	AuthorizationID         string    `json:"authorization_id"`
	DeterministicRunID      string    `json:"deterministic_run_id"`
	VariantID               string    `json:"variant_id"`
	ConfigurationSHA256     string    `json:"configuration_sha256"`
	RunnerIdentitySHA256    string    `json:"runner_identity_sha256"`
	Partition               string    `json:"partition"`
	CheckpointSHA256        string    `json:"checkpoint_sha256"`
	AccessReceiptHash       string    `json:"access_receipt_hash"`
	ResultArtifactSHA256    string    `json:"result_artifact_sha256"`
	OutputManifestSHA256    string    `json:"output_manifest_sha256"`
	AuthorityEvidenceSHA256 string    `json:"authority_evidence_sha256"`
	ResultStatus            string    `json:"result_status"`
	MandatoryGatesPassed    bool      `json:"mandatory_gates_passed"`
	CompletedAt             time.Time `json:"completed_at"`
	PreviousHash            string    `json:"previous_hash"`
	RecordHash              string    `json:"record_hash"`
}

type StageCompletionManifest struct {
	SchemaVersion        string   `json:"schema_version"`
	ExecutionSetID       string   `json:"execution_set_id"`
	PlanHash             string   `json:"plan_hash"`
	Stage                string   `json:"stage"`
	OrderedVariantIDs    []string `json:"ordered_variant_ids"`
	OrderedReceiptHashes []string `json:"ordered_execution_receipt_hashes"`
	OrderedResultHashes  []string `json:"ordered_result_artifact_hashes"`
	ManifestHash         string   `json:"manifest_hash"`
}

type StageExecutionSet struct {
	SchemaVersion      string                      `json:"schema_version"`
	ExecutionSetID     string                      `json:"execution_set_id"`
	Plan               StageExecutionPlan          `json:"execution_plan"`
	IssuanceState      string                      `json:"issuance_state"`
	CompletionState    string                      `json:"completion_state"`
	Authorizations     []StageVariantAuthorization `json:"variant_authorizations"`
	AccessReceipts     []StageVariantAccessReceipt `json:"access_receipts"`
	RetryProofs        []ZeroAccessRetryProof      `json:"zero_access_retry_proofs"`
	ExecutionReceipts  []StageExecutionReceipt     `json:"execution_receipts"`
	CompletionManifest *StageCompletionManifest    `json:"completion_manifest,omitempty"`
	FinalStageSeal     string                      `json:"final_stage_seal,omitempty"`
	IssuedAt           time.Time                   `json:"issued_at"`
	SealedAt           *time.Time                  `json:"sealed_at,omitempty"`
	RecordHash         string                      `json:"record_hash"`
}

type DevelopmentNominee struct {
	SchemaVersion       string    `json:"schema_version"`
	DevelopmentSetID    string    `json:"development_set_id"`
	Rule                string    `json:"rule"`
	Exists              bool      `json:"exists"`
	VariantID           string    `json:"variant_id,omitempty"`
	ConfigurationSHA256 string    `json:"configuration_sha256,omitempty"`
	SelectedAt          time.Time `json:"selected_at"`
	RecordHash          string    `json:"record_hash"`
}

type StageExecutionEnvelope struct {
	SchemaVersion string                     `json:"schema_version"`
	Snapshot      ResearchGovernanceSnapshot `json:"snapshot"`
	ExecutionSet  StageExecutionSet          `json:"stage_execution_set"`
	Authorization *StageVariantAuthorization `json:"variant_authorization,omitempty"`
	EnvelopeHash  string                     `json:"envelope_hash"`
}

func ParseStageExecutionEnvelopeJSON(data []byte) (StageExecutionEnvelope, error) {
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()
	var envelope StageExecutionEnvelope
	if err := decoder.Decode(&envelope); err != nil {
		return StageExecutionEnvelope{}, err
	}
	var trailing any
	if err := decoder.Decode(&trailing); err != io.EOF {
		return StageExecutionEnvelope{}, errors.New("RIF stage execution envelope has trailing JSON data")
	}
	if err := VerifyStageExecutionEnvelope(envelope); err != nil {
		return StageExecutionEnvelope{}, err
	}
	return envelope, nil
}

func VerifyStageExecutionEnvelope(envelope StageExecutionEnvelope) error {
	if envelope.SchemaVersion != StageEnvelopeSchemaVersion || envelope.Snapshot.SchemaVersion != ResearchGovernanceStoreSchemaVersionV2 {
		return errors.New("unsupported RIF stage execution envelope schema")
	}
	if err := verifyResearchSnapshot(envelope.Snapshot); err != nil {
		return err
	}
	found := false
	for _, set := range envelope.Snapshot.StageExecutionSets {
		if set.ExecutionSetID == envelope.ExecutionSet.ExecutionSetID && set.RecordHash == envelope.ExecutionSet.RecordHash {
			found = true
			break
		}
	}
	if !found || verifyStageExecutionSet(envelope.ExecutionSet) != nil {
		return errors.New("RIF stage execution set is not an exact valid persisted record")
	}
	if envelope.Authorization != nil {
		found = false
		for _, authorization := range envelope.ExecutionSet.Authorizations {
			if authorization.AuthorizationID == envelope.Authorization.AuthorizationID && authorization.RecordHash == envelope.Authorization.RecordHash {
				found = true
				break
			}
		}
		if !found || hashStageAuthorization(*envelope.Authorization) != envelope.Authorization.RecordHash {
			return errors.New("RIF stage authorization is not an exact valid persisted record")
		}
	}
	want, err := hashStageEnvelope(envelope)
	if err != nil || envelope.EnvelopeHash != want {
		return errors.New("RIF stage execution envelope hash mismatch")
	}
	return nil
}

func verifyStageExecutionState(snapshot ResearchGovernanceSnapshot) error {
	if snapshot.SchemaVersion == ResearchGovernanceStoreSchemaVersion {
		if len(snapshot.StageExecutionSets) != 0 || snapshot.DevelopmentNominee != nil {
			return errors.New("RIF V1 snapshot contains execution-set state")
		}
		return nil
	}
	if len(snapshot.StageExecutionSets) == 0 || len(snapshot.StageExecutionSets) > 2 {
		return errors.New("RIF V2 snapshot has invalid execution-set cardinality")
	}
	seen := map[string]struct{}{}
	var development, validation *StageExecutionSet
	for _, set := range snapshot.StageExecutionSets {
		if _, duplicate := seen[set.Plan.Stage]; duplicate || (set.Plan.Stage != "DEVELOPMENT" && set.Plan.Stage != "VALIDATION") {
			return errors.New("RIF V2 snapshot has duplicate or invalid execution-set stage")
		}
		seen[set.Plan.Stage] = struct{}{}
		if err := verifyStageExecutionSet(set); err != nil {
			return err
		}
		copySet := set
		if set.Plan.Stage == "DEVELOPMENT" {
			development = &copySet
		} else {
			validation = &copySet
		}
	}
	if _, ok := seen["DEVELOPMENT"]; !ok {
		return errors.New("RIF V2 snapshot has no DEVELOPMENT execution set")
	}
	if snapshot.DevelopmentNominee != nil {
		if snapshot.DevelopmentNominee.SchemaVersion != NomineeSelectionSchemaVersion || hashNominee(*snapshot.DevelopmentNominee) != snapshot.DevelopmentNominee.RecordHash || development == nil || snapshot.DevelopmentNominee.DevelopmentSetID != development.ExecutionSetID || development.CompletionState != "SEALED" {
			return errors.New("RIF DEVELOPMENT nominee hash mismatch")
		}
	}
	if validation != nil && (development == nil || development.CompletionState != "SEALED" || snapshot.DevelopmentNominee == nil || !snapshot.DevelopmentNominee.Exists) {
		return errors.New("RIF VALIDATION execution set lacks sealed DEVELOPMENT nominee authority")
	}
	return nil
}

func verifyStageExecutionSet(set StageExecutionSet) error {
	if set.SchemaVersion != StageExecutionSetSchemaVersion || set.ExecutionSetID == "" || set.Plan.SchemaVersion != StageExecutionPlanSchemaVersion || set.Plan.ExpectedExecutions != len(set.Plan.Configurations) || !set.Plan.Complete || set.Plan.OrderingRule != "NUMERIC_VARIANT_ID_ASCENDING" || set.IssuanceState != "ISSUED" || (set.CompletionState != "OPEN" && set.CompletionState != "SEALED") {
		return errors.New("RIF stage execution set or plan is incomplete")
	}
	if hashStagePlan(set.Plan) != set.Plan.PlanHash || len(set.Authorizations) < set.Plan.ExpectedExecutions {
		return errors.New("RIF stage execution plan hash or authorization cardinality mismatch")
	}
	previous := ""
	for index, authorization := range set.Authorizations {
		if authorization.SchemaVersion != StageAuthorizationSchemaVersion || authorization.ExecutionSetID != set.ExecutionSetID || authorization.PlanHash != set.Plan.PlanHash || authorization.PreviousHash != previous || authorization.Attempt < 1 || hashStageAuthorization(authorization) != authorization.RecordHash {
			return errors.New("RIF stage authorization chain is invalid")
		}
		if authorization.Attempt == 1 && (index >= len(set.Plan.Configurations) || authorization.Ordinal != index || !reflect.DeepEqual(authorization.Configuration, set.Plan.Configurations[index])) {
			return errors.New("RIF initial stage authorizations are reordered or mutated")
		}
		if !reflect.DeepEqual(authorization.Runner, set.Plan.Runner) || !reflect.DeepEqual(authorization.Partition, set.Plan.Partition) || !reflect.DeepEqual(authorization.Protocol, set.Plan.Protocol) || !reflect.DeepEqual(authorization.Checkpoint, set.Plan.Checkpoint) || !reflect.DeepEqual(authorization.Authorities, set.Plan.Authorities) || authorization.GateSet != set.Plan.GateSet {
			return errors.New("RIF stage authorization pre-execution identity substitution")
		}
		previous = authorization.RecordHash
	}
	previous = ""
	for _, receipt := range set.AccessReceipts {
		authorization := findStageAuthorization(set, receipt.AuthorizationID)
		if receipt.SchemaVersion != StageAccessReceiptSchemaVersion || receipt.ExecutionSetID != set.ExecutionSetID || receipt.PreviousHash != previous || hashStageAccess(receipt) != receipt.RecordHash || authorization == nil || receipt.VariantID != authorization.Configuration.VariantID || receipt.Attempt != authorization.Attempt {
			return errors.New("RIF stage access receipt chain is invalid")
		}
		previous = receipt.RecordHash
	}
	previous = ""
	for _, proof := range set.RetryProofs {
		if proof.SchemaVersion != StageRetryProofSchemaVersion || proof.ExecutionSetID != set.ExecutionSetID || proof.RowsAccessed != 0 || proof.OutcomeArtifacts != 0 || proof.PreviousHash != previous || !stageValidNonPlaceholderSHA(proof.DurableProofSHA256) || hashRetryProof(proof) != proof.RecordHash || findStageAccess(set, proof.PriorAccessReceiptHash) == nil {
			return errors.New("RIF zero-access retry proof chain is invalid")
		}
		previous = proof.RecordHash
	}
	previous = ""
	seenVariants := map[string]struct{}{}
	for _, receipt := range set.ExecutionReceipts {
		authorization := findStageAuthorization(set, receipt.AuthorizationID)
		access := findStageAccess(set, receipt.AccessReceiptHash)
		if receipt.SchemaVersion != StageExecutionReceiptSchemaVersion || receipt.ExecutionSetID != set.ExecutionSetID || receipt.PlanHash != set.Plan.PlanHash || receipt.Partition != set.Plan.Stage || receipt.PreviousHash != previous || hashStageReceipt(receipt) != receipt.RecordHash || authorization == nil || access == nil || receipt.VariantID != authorization.Configuration.VariantID || receipt.ConfigurationSHA256 != authorization.Configuration.ConfigurationSHA256 || receipt.CheckpointSHA256 != set.Plan.Checkpoint.SHA256 || !stageValidNonPlaceholderSHA(receipt.ResultArtifactSHA256) || !stageValidNonPlaceholderSHA(receipt.OutputManifestSHA256) || !stageValidNonPlaceholderSHA(receipt.AuthorityEvidenceSHA256) {
			return errors.New("RIF stage execution receipt chain is invalid")
		}
		if _, duplicate := seenVariants[receipt.VariantID]; duplicate {
			return errors.New("RIF duplicate result-bearing stage execution")
		}
		seenVariants[receipt.VariantID] = struct{}{}
		previous = receipt.RecordHash
	}
	if set.CompletionState == "OPEN" {
		if set.CompletionManifest != nil || set.FinalStageSeal != "" || set.SealedAt != nil {
			return errors.New("RIF open stage execution set contains seal")
		}
	} else {
		if set.CompletionManifest == nil || set.SealedAt == nil || !stageValidNonPlaceholderSHA(set.FinalStageSeal) || set.CompletionManifest.SchemaVersion != StageManifestSchemaVersion || hashStageManifest(*set.CompletionManifest) != set.CompletionManifest.ManifestHash || len(set.ExecutionReceipts) != set.Plan.ExpectedExecutions {
			return errors.New("RIF sealed stage execution set is incomplete")
		}
		manifest := StageCompletionManifest{SchemaVersion: StageManifestSchemaVersion, ExecutionSetID: set.ExecutionSetID, PlanHash: set.Plan.PlanHash, Stage: set.Plan.Stage}
		for _, configuration := range set.Plan.Configurations {
			receipt := findStageReceipt(set, configuration.VariantID)
			if receipt == nil {
				return errors.New("RIF sealed stage execution set is missing a registered result")
			}
			manifest.OrderedVariantIDs = append(manifest.OrderedVariantIDs, configuration.VariantID)
			manifest.OrderedReceiptHashes = append(manifest.OrderedReceiptHashes, receipt.RecordHash)
			manifest.OrderedResultHashes = append(manifest.OrderedResultHashes, receipt.ResultArtifactSHA256)
		}
		manifest.ManifestHash = hashStageManifest(manifest)
		if !reflect.DeepEqual(manifest, *set.CompletionManifest) {
			return errors.New("RIF stage completion manifest content mismatch")
		}
		wantSeal, _ := canonicalHash(struct {
			ExecutionSetID string `json:"execution_set_id"`
			PlanHash       string `json:"plan_hash"`
			ManifestHash   string `json:"manifest_hash"`
		}{set.ExecutionSetID, set.Plan.PlanHash, manifest.ManifestHash})
		if set.FinalStageSeal != wantSeal {
			return errors.New("RIF final stage seal mismatch")
		}
	}
	if hashStageSet(set) != set.RecordHash {
		return errors.New("RIF stage execution set record hash mismatch")
	}
	return nil
}

func findStageAuthorization(set StageExecutionSet, id string) *StageVariantAuthorization {
	for i := range set.Authorizations {
		if set.Authorizations[i].AuthorizationID == id {
			return &set.Authorizations[i]
		}
	}
	return nil
}

func findStageAccess(set StageExecutionSet, hash string) *StageVariantAccessReceipt {
	for i := range set.AccessReceipts {
		if set.AccessReceipts[i].RecordHash == hash {
			return &set.AccessReceipts[i]
		}
	}
	return nil
}

func findStageReceipt(set StageExecutionSet, variantID string) *StageExecutionReceipt {
	for i := range set.ExecutionReceipts {
		if set.ExecutionReceipts[i].VariantID == variantID {
			return &set.ExecutionReceipts[i]
		}
	}
	return nil
}

func hashStagePlan(value StageExecutionPlan) string {
	value.PlanHash = ""
	hash, _ := canonicalHash(value)
	return hash
}
func hashStageAuthorization(value StageVariantAuthorization) string {
	value.RecordHash = ""
	hash, _ := canonicalHash(value)
	return hash
}
func hashStageAccess(value StageVariantAccessReceipt) string {
	value.RecordHash = ""
	hash, _ := canonicalHash(value)
	return hash
}
func hashRetryProof(value ZeroAccessRetryProof) string {
	value.RecordHash = ""
	hash, _ := canonicalHash(value)
	return hash
}
func hashStageReceipt(value StageExecutionReceipt) string {
	value.RecordHash = ""
	hash, _ := canonicalHash(value)
	return hash
}
func hashStageManifest(value StageCompletionManifest) string {
	value.ManifestHash = ""
	hash, _ := canonicalHash(value)
	return hash
}
func hashStageSet(value StageExecutionSet) string {
	value.RecordHash = ""
	hash, _ := canonicalHash(value)
	return hash
}
func hashNominee(value DevelopmentNominee) string {
	value.RecordHash = ""
	hash, _ := canonicalHash(value)
	return hash
}
func hashStageEnvelope(value StageExecutionEnvelope) (string, error) {
	value.EnvelopeHash = ""
	return canonicalHash(value)
}

func stageValidNonPlaceholderSHA(value string) bool {
	return validSHA(value) && value != "sha256:"+strings.Repeat("0", 64)
}
