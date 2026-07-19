package qualificationrunner

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/david22573/ak-engine/internal/preconditions"
	"github.com/david22573/ak-engine/internal/qualification"
	"github.com/david22573/ak-engine/internal/rifbridge"
)

const (
	StageReadinessSchemaVersion    = "ak.engine.stage_plan_readiness.v1"
	StageResultEnvelopeVersion     = "ak.rif.execution_result_envelope.v1"
	StageProgressSchemaVersion     = "ak.engine.stage_execution_progress.v1"
	StageBatchReceiptVersion       = "ak.engine.stage_batch_execution_receipt.v1"
	StageBatchManifestVersion      = "ak.engine.stage_completion_manifest.v1"
	PrebuildReceiptVersion         = "ak.engine.runner_prebuild_receipt.v1"
	NoRealPartitionAccessLabel     = "NO_REAL_PARTITION_ACCESS"
	PreexecutionCycleAbsentLabel   = "PREEXECUTION_IDENTITY_CYCLE_ABSENT"
	MultivariantAuthorizationLabel = "MULTIVARIANT_STAGE_AUTHORIZATION_AVAILABLE"
)

type RunnerPrebuildInput struct {
	SourceCommit             string
	PackageID                string
	CanonicalPackage         []byte
	DeterministicBuildInputs []rifbridge.StageHashIdentity
	CompilerIdentity         string
	BuildModeID              string
	CanonicalBuildMode       []byte
	Binary                   []byte
}

type RunnerPrebuildReceipt struct {
	SchemaVersion     string                                 `json:"schema_version"`
	Runner            rifbridge.RunnerImplementationIdentity `json:"runner_identity"`
	DataLoads         int                                    `json:"data_loads"`
	CandidateEvents   int                                    `json:"candidate_events"`
	CandidateOutcomes int                                    `json:"candidate_outcomes"`
	Labels            []string                               `json:"labels"`
	ReceiptHash       string                                 `json:"receipt_hash"`
}

type StagePlanReadiness struct {
	SchemaVersion             string   `json:"schema_version"`
	ExecutionSetID            string   `json:"execution_set_id"`
	PlanHash                  string   `json:"plan_hash"`
	Stage                     string   `json:"stage"`
	AuthorizedVariants        int      `json:"authorized_variants"`
	ValidationSubsetDerivable bool     `json:"validation_subset_derivable"`
	FinalHoldoutMode          string   `json:"final_holdout_mode"`
	DataLoads                 int      `json:"data_loads"`
	CandidateEvents           int      `json:"candidate_events"`
	CandidateOutcomes         int      `json:"candidate_outcomes"`
	Labels                    []string `json:"labels"`
	ArtifactHash              string   `json:"artifact_hash"`
}

type StageAuthorityInvocationEvidence struct {
	Identity       rifbridge.StageHashIdentity `json:"identity"`
	Invoked        bool                        `json:"invoked"`
	EvidenceSHA256 string                      `json:"evidence_sha256"`
}

type StageVariantOutput struct {
	ResultArtifact       json.RawMessage
	OutputManifestSHA256 string
	AuthorityInvocations []StageAuthorityInvocationEvidence
	MandatoryGatesPassed bool
}

type StageResultEnvelope struct {
	SchemaVersion        string                                    `json:"schema_version"`
	ExecutionSetID       string                                    `json:"execution_set_id"`
	PlanHash             string                                    `json:"plan_hash"`
	AuthorizationID      string                                    `json:"authorization_id"`
	DeterministicRunID   string                                    `json:"deterministic_run_id"`
	Configuration        rifbridge.RegisteredConfigurationIdentity `json:"registered_configuration"`
	Runner               rifbridge.RunnerImplementationIdentity    `json:"runner_identity"`
	Partition            rifbridge.StagePartition                  `json:"partition"`
	Protocol             rifbridge.StageProtocolIdentity           `json:"protocol_identity"`
	Checkpoint           rifbridge.StageHashIdentity               `json:"checkpoint"`
	Authorities          rifbridge.StageAuthorityIdentity          `json:"authority_identities"`
	GateSet              rifbridge.StageHashIdentity               `json:"gate_set_identity"`
	AccessReceiptHash    string                                    `json:"access_receipt_hash"`
	ResultArtifact       json.RawMessage                           `json:"result_artifact"`
	ResultArtifactSHA256 string                                    `json:"result_artifact_sha256"`
	OutputManifestSHA256 string                                    `json:"output_manifest_sha256"`
	AuthorityInvocations []StageAuthorityInvocationEvidence        `json:"authority_invocation_evidence"`
	ResultStatus         string                                    `json:"result_status"`
	MandatoryGatesPassed bool                                      `json:"mandatory_gates_passed"`
	EnvelopeHash         string                                    `json:"envelope_hash"`
}

type StageActiveAttempt struct {
	AuthorizationID   string    `json:"authorization_id"`
	AccessReceiptHash string    `json:"access_receipt_hash"`
	VariantID         string    `json:"variant_id"`
	Attempt           int       `json:"attempt"`
	StartedAt         time.Time `json:"started_at"`
}

type StageBatchExecutionReceipt struct {
	SchemaVersion        string    `json:"schema_version"`
	ExecutionSetID       string    `json:"execution_set_id"`
	PlanHash             string    `json:"plan_hash"`
	Ordinal              int       `json:"ordinal"`
	AuthorizationID      string    `json:"authorization_id"`
	DeterministicRunID   string    `json:"deterministic_run_id"`
	VariantID            string    `json:"variant_id"`
	ConfigurationSHA256  string    `json:"configuration_sha256"`
	ResultArtifactSHA256 string    `json:"result_artifact_sha256"`
	OutputManifestSHA256 string    `json:"output_manifest_sha256"`
	ResultEnvelopeHash   string    `json:"result_envelope_hash"`
	MandatoryGatesPassed bool      `json:"mandatory_gates_passed"`
	CompletedAt          time.Time `json:"completed_at"`
	PreviousHash         string    `json:"previous_hash"`
	ReceiptHash          string    `json:"receipt_hash"`
}

type StageBatchManifest struct {
	SchemaVersion        string   `json:"schema_version"`
	ExecutionSetID       string   `json:"execution_set_id"`
	PlanHash             string   `json:"plan_hash"`
	Stage                string   `json:"stage"`
	OrderedVariantIDs    []string `json:"ordered_variant_ids"`
	OrderedReceiptHashes []string `json:"ordered_execution_receipt_hashes"`
	OrderedResultHashes  []string `json:"ordered_result_artifact_hashes"`
	ManifestHash         string   `json:"manifest_hash"`
}

type StageExecutionProgress struct {
	SchemaVersion  string                       `json:"schema_version"`
	ExecutionSetID string                       `json:"execution_set_id"`
	PlanHash       string                       `json:"plan_hash"`
	State          string                       `json:"state"`
	NextOrdinal    int                          `json:"next_ordinal"`
	ActiveAttempt  *StageActiveAttempt          `json:"active_attempt,omitempty"`
	Receipts       []StageBatchExecutionReceipt `json:"execution_receipts"`
	Manifest       *StageBatchManifest          `json:"completion_manifest,omitempty"`
	IntegrityHash  string                       `json:"integrity_hash"`
}

type stageVariantExecutor func(rifbridge.StageVariantAuthorization, CandidateConfiguration) (StageVariantOutput, error)

type StageBatchRunner struct {
	progressPath string
	runner       rifbridge.RunnerImplementationIdentity
	now          func() time.Time
}

func ComputePreexecutionRunnerIdentity(input RunnerPrebuildInput) (rifbridge.RunnerImplementationIdentity, RunnerPrebuildReceipt, error) {
	if len(input.SourceCommit) != 40 || input.PackageID == "" || len(input.CanonicalPackage) == 0 || len(input.DeterministicBuildInputs) == 0 || input.CompilerIdentity == "" || input.BuildModeID == "" || len(input.CanonicalBuildMode) == 0 || len(input.Binary) == 0 {
		return rifbridge.RunnerImplementationIdentity{}, RunnerPrebuildReceipt{}, errors.New("complete deterministic data-independent build inputs are required")
	}
	inputs := append([]rifbridge.StageHashIdentity(nil), input.DeterministicBuildInputs...)
	sort.Slice(inputs, func(i, j int) bool { return inputs[i].ID < inputs[j].ID })
	for i, item := range inputs {
		if item.ID == "" || !validSHA(item.SHA256) || (i > 0 && inputs[i-1].ID == item.ID) {
			return rifbridge.RunnerImplementationIdentity{}, RunnerPrebuildReceipt{}, errors.New("deterministic build-input identity is invalid or duplicated")
		}
	}
	buildInputsHash, err := canonicalHash(inputs)
	if err != nil {
		return rifbridge.RunnerImplementationIdentity{}, RunnerPrebuildReceipt{}, err
	}
	runner := rifbridge.RunnerImplementationIdentity{SchemaVersion: rifbridge.RunnerImplementationSchemaVersion, SourceCommit: input.SourceCommit, PackageIdentity: rifbridge.StageHashIdentity{ID: input.PackageID, SHA256: byteHash(input.CanonicalPackage)}, BuildInputsSHA256: buildInputsHash, CompilerIdentity: input.CompilerIdentity, BuildModeIdentity: rifbridge.StageHashIdentity{ID: input.BuildModeID, SHA256: byteHash(input.CanonicalBuildMode)}, BinarySHA256: byteHash(input.Binary)}
	receipt := RunnerPrebuildReceipt{SchemaVersion: PrebuildReceiptVersion, Runner: runner, Labels: []string{NoOutcomesLabel, NoRealPartitionAccessLabel, PreexecutionCycleAbsentLabel}}
	receipt.ReceiptHash, err = hashPrebuildReceipt(receipt)
	return runner, receipt, err
}

func VerifyStageExecutionPlan(envelope rifbridge.StageExecutionEnvelope, localRunner rifbridge.RunnerImplementationIdentity) (ResearchIdentityV4, StagePlanReadiness, error) {
	if err := rifbridge.VerifyStageExecutionEnvelope(envelope); err != nil {
		return ResearchIdentityV4{}, StagePlanReadiness{}, fmt.Errorf("RIF stage governance: %w", err)
	}
	identity, identityHash, err := decodeIdentity(envelope.Snapshot.Identity)
	if err != nil {
		return ResearchIdentityV4{}, StagePlanReadiness{}, err
	}
	if identityHash != envelope.Snapshot.IdentityHash || envelope.ExecutionSet.Plan.ResearchIdentityHash != identityHash {
		return ResearchIdentityV4{}, StagePlanReadiness{}, errors.New("stage plan research identity mismatch")
	}
	if err := validateIdentity(identity); err != nil {
		return ResearchIdentityV4{}, StagePlanReadiness{}, err
	}
	set, plan := envelope.ExecutionSet, envelope.ExecutionSet.Plan
	if plan.Stage != "DEVELOPMENT" && plan.Stage != "VALIDATION" {
		return ResearchIdentityV4{}, StagePlanReadiness{}, errors.New("multi-variant stage plan cannot target FINAL_HOLDOUT")
	}
	if set.CompletionState != "OPEN" || plan.Partition.Name != plan.Stage || plan.ExpectedExecutions != len(plan.Configurations) || plan.ExpectedExecutions < 1 || plan.ExpectedExecutions > 12 || !plan.Complete || plan.OrderingRule != "NUMERIC_VARIANT_ID_ASCENDING" {
		return ResearchIdentityV4{}, StagePlanReadiness{}, errors.New("stage plan is incomplete, sealed, or has invalid cardinality/order")
	}
	if !reflect.DeepEqual(localRunner, plan.Runner) || errRunnerIdentity(localRunner) != nil || localRunner.SourceCommit != identity.Repositories.RunnerGitCommit || localRunner.BinarySHA256 != identity.Repositories.RunnerExecutableSHA256 {
		return ResearchIdentityV4{}, StagePlanReadiness{}, errors.New("local pre-execution runner identity differs from RIF authorization")
	}
	if plan.Protocol.ID != identity.Protocol.ID || plan.Protocol.SHA256 != identity.Protocol.SHA256 || plan.Protocol.ContentAddressedIdentity != identity.Protocol.ContentAddressedIdentity || plan.Checkpoint.ID != identity.Dataset.Checkpoint.ID || plan.Checkpoint.SHA256 != identity.Dataset.Checkpoint.SHA256 {
		return ResearchIdentityV4{}, StagePlanReadiness{}, errors.New("stage protocol or checkpoint identity substitution")
	}
	registeredPartition, ok := findPartition(identity, plan.Stage)
	if !ok || plan.Partition.Name != registeredPartition.Name || !plan.Partition.Interval.Start.Equal(registeredPartition.Interval.Start) || !plan.Partition.Interval.End.Equal(registeredPartition.Interval.End) || plan.Partition.StructuralDayCount != registeredPartition.StructuralDayCount || plan.Partition.RequiredSymbolCoverageSHA256 != registeredPartition.RequiredSymbolCoverageSHA256 {
		return ResearchIdentityV4{}, StagePlanReadiness{}, errors.New("stage partition identity substitution")
	}
	datasetHash, _ := canonicalHash(identity.Dataset)
	if plan.DatasetIdentitySHA256 != datasetHash {
		return ResearchIdentityV4{}, StagePlanReadiness{}, errors.New("stage dataset identity hash mismatch")
	}
	if err := verifyStageAuthorities(plan, identity); err != nil {
		return ResearchIdentityV4{}, StagePlanReadiness{}, err
	}
	configurations, err := verifyStageConfigurations(plan, identity, envelope.Snapshot.DevelopmentNominee)
	if err != nil {
		return ResearchIdentityV4{}, StagePlanReadiness{}, err
	}
	if len(set.Authorizations) < len(configurations) {
		return ResearchIdentityV4{}, StagePlanReadiness{}, errors.New("stage plan is missing variant-specific authorizations")
	}
	for ordinal, configuration := range plan.Configurations {
		authorization := set.Authorizations[ordinal]
		if authorization.Ordinal != ordinal || authorization.Attempt != 1 || authorization.Configuration.VariantID != configuration.VariantID || authorization.Configuration.ConfigurationSHA256 != configuration.ConfigurationSHA256 || !reflect.DeepEqual(authorization.Runner, plan.Runner) || authorization.Partition.Name != plan.Partition.Name || authorization.Protocol.SHA256 != plan.Protocol.SHA256 || authorization.Checkpoint.SHA256 != plan.Checkpoint.SHA256 || authorization.GateSet.SHA256 != plan.GateSet.SHA256 {
			return ResearchIdentityV4{}, StagePlanReadiness{}, errors.New("variant authorization is missing, reordered, or identity-mutated")
		}
	}
	derivable := validationSubsetDerivable(identity, plan.Configurations)
	readiness := StagePlanReadiness{SchemaVersion: StageReadinessSchemaVersion, ExecutionSetID: set.ExecutionSetID, PlanHash: plan.PlanHash, Stage: plan.Stage, AuthorizedVariants: len(configurations), ValidationSubsetDerivable: derivable, FinalHoldoutMode: "SINGLE_FROZEN_CANDIDATE_ONLY", Labels: []string{NoOutcomesLabel, NoRealPartitionAccessLabel, PreexecutionCycleAbsentLabel, MultivariantAuthorizationLabel}}
	readiness.ArtifactHash, err = hashStageReadiness(readiness)
	return identity, readiness, err
}

func NewStageBatchRunner(progressPath string, runner rifbridge.RunnerImplementationIdentity) (*StageBatchRunner, error) {
	if progressPath == "" || errRunnerIdentity(runner) != nil {
		return nil, errors.New("stage batch runner requires a durable progress path and complete runner identity")
	}
	return &StageBatchRunner{progressPath: progressPath, runner: runner, now: time.Now}, nil
}

func (r *StageBatchRunner) ExecuteStage(envelopes []rifbridge.StageExecutionEnvelope, artifacts [][]byte) ([]StageResultEnvelope, []StageBatchExecutionReceipt, StageBatchManifest, error) {
	if len(envelopes) == 0 {
		return nil, nil, StageBatchManifest{}, errors.New("complete ordered stage envelope set is required")
	}
	expected := envelopes[0].ExecutionSet.Plan.ExpectedExecutions
	if len(envelopes) != expected {
		return nil, nil, StageBatchManifest{}, errors.New("missing or additional stage variant envelope")
	}
	if len(artifacts) != expected {
		return nil, nil, StageBatchManifest{}, errors.New("each authorized stage variant requires one exact partition input artifact")
	}
	results := make([]StageResultEnvelope, 0, expected)
	receipts := make([]StageBatchExecutionReceipt, 0, expected)
	for ordinal, envelope := range envelopes {
		if envelope.Authorization == nil || envelope.Authorization.Ordinal != ordinal || envelope.ExecutionSet.ExecutionSetID != envelopes[0].ExecutionSet.ExecutionSetID || envelope.ExecutionSet.Plan.PlanHash != envelopes[0].ExecutionSet.Plan.PlanHash {
			return nil, nil, StageBatchManifest{}, errors.New("stage envelope set is duplicate, reordered, or belongs to another plan")
		}
		result, receipt, err := r.ExecuteAuthorizedVariant(envelope, artifacts[ordinal])
		if err != nil {
			return results, receipts, StageBatchManifest{}, err
		}
		results = append(results, result)
		receipts = append(receipts, receipt)
	}
	manifest, err := r.Seal(envelopes[len(envelopes)-1])
	return results, receipts, manifest, err
}

func (r *StageBatchRunner) ExecuteAuthorizedVariant(envelope rifbridge.StageExecutionEnvelope, artifactJSON []byte) (StageResultEnvelope, StageBatchExecutionReceipt, error) {
	return r.executeAuthorizedVariant(envelope, func(authorization rifbridge.StageVariantAuthorization, configuration CandidateConfiguration) (StageVariantOutput, error) {
		return executeStageArtifact(envelope, authorization, configuration, artifactJSON)
	})
}

func (r *StageBatchRunner) executeAuthorizedVariant(envelope rifbridge.StageExecutionEnvelope, executor stageVariantExecutor) (StageResultEnvelope, StageBatchExecutionReceipt, error) {
	_, _, err := VerifyStageExecutionPlan(envelope, r.runner)
	if err != nil {
		return StageResultEnvelope{}, StageBatchExecutionReceipt{}, err
	}
	if envelope.Authorization == nil {
		return StageResultEnvelope{}, StageBatchExecutionReceipt{}, errors.New("exact RIF variant authorization is required before execution")
	}
	lock, err := r.acquireLock()
	if err != nil {
		return StageResultEnvelope{}, StageBatchExecutionReceipt{}, err
	}
	defer releaseStageLock(lock)
	progress, err := r.loadOrInitialize(envelope.ExecutionSet)
	if err != nil {
		return StageResultEnvelope{}, StageBatchExecutionReceipt{}, err
	}
	if progress.State == "SEALED" || envelope.ExecutionSet.CompletionState == "SEALED" {
		return StageResultEnvelope{}, StageBatchExecutionReceipt{}, errors.New("sealed stage execution set cannot execute another configuration")
	}
	if progress.ActiveAttempt != nil {
		return StageResultEnvelope{}, StageBatchExecutionReceipt{}, errors.New("indeterminate prior attempt requires an exact zero-access retry proof; outcome-based rerun is prohibited")
	}
	if progress.NextOrdinal >= len(envelope.ExecutionSet.Plan.Configurations) {
		return StageResultEnvelope{}, StageBatchExecutionReceipt{}, errors.New("all authorized stage configurations are already complete")
	}
	authorization := *envelope.Authorization
	expected := envelope.ExecutionSet.Plan.Configurations[progress.NextOrdinal]
	if authorization.Ordinal != progress.NextOrdinal || authorization.Configuration.VariantID != expected.VariantID || authorization.Configuration.ConfigurationSHA256 != expected.ConfigurationSHA256 {
		return StageResultEnvelope{}, StageBatchExecutionReceipt{}, errors.New("missing, additional, duplicate, reordered, or mutated stage variant")
	}
	accessReceipt, err := exactConsumedStageAccess(envelope.ExecutionSet, authorization)
	if err != nil {
		return StageResultEnvelope{}, StageBatchExecutionReceipt{}, err
	}
	var configuration CandidateConfiguration
	if err := strictJSON(authorization.Configuration.CanonicalConfiguration, &configuration); err != nil {
		return StageResultEnvelope{}, StageBatchExecutionReceipt{}, fmt.Errorf("decode registered configuration: %w", err)
	}
	progress.ActiveAttempt = &StageActiveAttempt{AuthorizationID: authorization.AuthorizationID, AccessReceiptHash: accessReceipt.RecordHash, VariantID: authorization.Configuration.VariantID, Attempt: authorization.Attempt, StartedAt: r.now().UTC()}
	if err := r.writeProgress(progress); err != nil {
		return StageResultEnvelope{}, StageBatchExecutionReceipt{}, err
	}
	output, executeErr := executor(authorization, configuration)
	if executeErr != nil {
		return StageResultEnvelope{}, StageBatchExecutionReceipt{}, executeErr
	}
	result, err := buildStageResultEnvelope(envelope.ExecutionSet, authorization, *accessReceipt, output)
	if err != nil {
		return StageResultEnvelope{}, StageBatchExecutionReceipt{}, err
	}
	previous := ""
	if len(progress.Receipts) > 0 {
		previous = progress.Receipts[len(progress.Receipts)-1].ReceiptHash
	}
	receipt := StageBatchExecutionReceipt{SchemaVersion: StageBatchReceiptVersion, ExecutionSetID: progress.ExecutionSetID, PlanHash: progress.PlanHash, Ordinal: progress.NextOrdinal, AuthorizationID: authorization.AuthorizationID, DeterministicRunID: result.DeterministicRunID, VariantID: authorization.Configuration.VariantID, ConfigurationSHA256: authorization.Configuration.ConfigurationSHA256, ResultArtifactSHA256: result.ResultArtifactSHA256, OutputManifestSHA256: result.OutputManifestSHA256, ResultEnvelopeHash: result.EnvelopeHash, MandatoryGatesPassed: result.MandatoryGatesPassed, CompletedAt: r.now().UTC(), PreviousHash: previous}
	receipt.ReceiptHash, err = hashBatchReceipt(receipt)
	if err != nil {
		return StageResultEnvelope{}, StageBatchExecutionReceipt{}, err
	}
	progress.Receipts = append(progress.Receipts, receipt)
	progress.NextOrdinal++
	progress.ActiveAttempt = nil
	if err := r.writeProgress(progress); err != nil {
		return StageResultEnvelope{}, StageBatchExecutionReceipt{}, err
	}
	return result, receipt, nil
}

func (r *StageBatchRunner) ResumeWithZeroAccessRetry(envelope rifbridge.StageExecutionEnvelope) error {
	if _, _, err := VerifyStageExecutionPlan(envelope, r.runner); err != nil {
		return err
	}
	if envelope.Authorization == nil || envelope.Authorization.Attempt <= 1 {
		return errors.New("RIF retry authorization is required")
	}
	lock, err := r.acquireLock()
	if err != nil {
		return err
	}
	defer releaseStageLock(lock)
	progress, err := r.readProgress()
	if err != nil {
		return err
	}
	active := progress.ActiveAttempt
	if active == nil || active.VariantID != envelope.Authorization.Configuration.VariantID || envelope.Authorization.Attempt <= active.Attempt {
		return errors.New("retry authorization does not match the indeterminate attempt")
	}
	proven := false
	for _, proof := range envelope.ExecutionSet.RetryProofs {
		if proof.PriorAuthorizationID == active.AuthorizationID && proof.PriorAccessReceiptHash == active.AccessReceiptHash && proof.VariantID == active.VariantID && proof.RowsAccessed == 0 && proof.OutcomeArtifacts == 0 && validNonPlaceholderHash(proof.DurableProofSHA256) {
			proven = true
			break
		}
	}
	if !proven {
		return errors.New("durable zero-row and zero-outcome retry proof is missing")
	}
	progress.ActiveAttempt = nil
	return r.writeProgress(progress)
}

func (r *StageBatchRunner) Seal(envelope rifbridge.StageExecutionEnvelope) (StageBatchManifest, error) {
	if _, _, err := VerifyStageExecutionPlan(envelope, r.runner); err != nil {
		return StageBatchManifest{}, err
	}
	lock, err := r.acquireLock()
	if err != nil {
		return StageBatchManifest{}, err
	}
	defer releaseStageLock(lock)
	progress, err := r.readProgress()
	if err != nil {
		return StageBatchManifest{}, err
	}
	if progress.State == "SEALED" {
		return StageBatchManifest{}, errors.New("stage batch is already sealed")
	}
	if progress.ActiveAttempt != nil || progress.NextOrdinal != envelope.ExecutionSet.Plan.ExpectedExecutions || len(progress.Receipts) != envelope.ExecutionSet.Plan.ExpectedExecutions {
		return StageBatchManifest{}, errors.New("incomplete stage execution set cannot seal")
	}
	manifest := StageBatchManifest{SchemaVersion: StageBatchManifestVersion, ExecutionSetID: progress.ExecutionSetID, PlanHash: progress.PlanHash, Stage: envelope.ExecutionSet.Plan.Stage}
	for ordinal, receipt := range progress.Receipts {
		if receipt.Ordinal != ordinal || receipt.VariantID != envelope.ExecutionSet.Plan.Configurations[ordinal].VariantID {
			return StageBatchManifest{}, errors.New("stage progress receipt order mismatch")
		}
		manifest.OrderedVariantIDs = append(manifest.OrderedVariantIDs, receipt.VariantID)
		manifest.OrderedReceiptHashes = append(manifest.OrderedReceiptHashes, receipt.ReceiptHash)
		manifest.OrderedResultHashes = append(manifest.OrderedResultHashes, receipt.ResultArtifactSHA256)
	}
	manifest.ManifestHash, err = hashBatchManifest(manifest)
	if err != nil {
		return StageBatchManifest{}, err
	}
	progress.State, progress.Manifest = "SEALED", &manifest
	if err := r.writeProgress(progress); err != nil {
		return StageBatchManifest{}, err
	}
	return manifest, nil
}

func (r *StageBatchRunner) Progress() (StageExecutionProgress, error) {
	lock, err := r.acquireLock()
	if err != nil {
		return StageExecutionProgress{}, err
	}
	defer releaseStageLock(lock)
	return r.readProgress()
}

func verifyStageConfigurations(plan rifbridge.StageExecutionPlan, identity ResearchIdentityV4, nominee *rifbridge.DevelopmentNominee) ([]CandidateConfiguration, error) {
	if len(plan.Configurations) == 0 || !numericStageOrder(plan.Configurations) {
		return nil, errors.New("stage configuration order is not numeric and deterministic")
	}
	identityByID := map[string]IdentityVariant{}
	for _, variant := range identity.VariantLedger.Variants {
		identityByID[variant.ID] = variant
	}
	decoded := make([]CandidateConfiguration, len(plan.Configurations))
	seen := map[string]struct{}{}
	for index, registered := range plan.Configurations {
		if registered.SchemaVersion != rifbridge.RegisteredConfigurationSchemaVersion || registered.CandidateFamilyID != identity.CandidateScope.FamilyID || registered.ProtocolID != identity.Protocol.ID || registered.ProtocolSHA256 != identity.Protocol.SHA256 {
			return nil, errors.New("registered stage configuration identity is incomplete or substituted")
		}
		if _, duplicate := seen[registered.VariantID]; duplicate {
			return nil, errors.New("duplicate stage configuration")
		}
		seen[registered.VariantID] = struct{}{}
		if err := strictJSON(registered.CanonicalConfiguration, &decoded[index]); err != nil {
			return nil, err
		}
		hash, err := CanonicalConfigurationHash(decoded[index])
		identityVariant, ok := identityByID[registered.VariantID]
		if err != nil || hash != registered.ConfigurationSHA256 || !ok || identityVariant.ConfigurationSHA256 != hash {
			return nil, errors.New("stage canonical configuration hash or RIF registration mismatch")
		}
		if err := validateVariantAgainstV00(RegisteredVariant{ID: registered.VariantID, Dimensions: identityVariant.Dimensions, Configuration: decoded[index], ConfigurationSHA256: hash}); err != nil {
			return nil, err
		}
	}
	if plan.Stage == "DEVELOPMENT" {
		if len(seen) != len(identityByID) {
			return nil, errors.New("DEVELOPMENT stage plan is not the complete registered ledger")
		}
		for id := range identityByID {
			if _, ok := seen[id]; !ok {
				return nil, errors.New("DEVELOPMENT stage plan omits a registered variant")
			}
		}
	} else {
		if nominee == nil || !nominee.Exists {
			return nil, errors.New("VALIDATION requires a sealed DEVELOPMENT nominee")
		}
		expected, err := expectedValidationIDs(identity, nominee.VariantID)
		if err != nil {
			return nil, err
		}
		actual := make([]string, len(plan.Configurations))
		for i := range plan.Configurations {
			actual[i] = plan.Configurations[i].VariantID
		}
		if !reflect.DeepEqual(actual, expected) {
			return nil, errors.New("VALIDATION is not exactly nominee plus mandatory registered neighbors")
		}
	}
	return decoded, nil
}

func verifyStageAuthorities(plan rifbridge.StageExecutionPlan, identity ResearchIdentityV4) error {
	policyHash, err := preconditions.AcceptedIndependencePolicyHashV3(preconditions.AcceptedIndependencePolicyV3Default())
	if err != nil {
		return err
	}
	uncertaintyHash, err := preconditions.AcceptedUncertaintyMethodHashV2(preconditions.AcceptedUncertaintyMethodV2())
	if err != nil {
		return err
	}
	concentrationHash := preconditions.DefaultConcentrationGovernanceDecisionV3().CanonicalDecisionHash
	gates := qualification.AcceptedPR4B0GateSet()
	gateHash, err := qualification.PR4B0GateSetHash(gates)
	if err != nil {
		return err
	}
	if plan.Authorities.Independence.ID != preconditions.AcceptedIndependencePolicyVersionV3 || plan.Authorities.Independence.SHA256 != policyHash || plan.Authorities.Uncertainty.ID != preconditions.AcceptedUncertaintyMethodVersion || plan.Authorities.Uncertainty.SHA256 != uncertaintyHash || plan.Authorities.ConcentrationSHA256 != concentrationHash || plan.GateSet.ID != qualification.PR4B0GateSetID || plan.GateSet.SHA256 != gateHash || plan.Authorities.QualificationGateSet != plan.GateSet {
		return errors.New("stage plan does not bind exact accepted authority and gate implementations")
	}
	if identity.Authorities.Independence.ID != plan.Authorities.Independence.ID || identity.Authorities.Independence.SHA256 != plan.Authorities.Independence.SHA256 || identity.Authorities.Uncertainty.ID != plan.Authorities.Uncertainty.ID || identity.Authorities.Uncertainty.SHA256 != plan.Authorities.Uncertainty.SHA256 || identity.Authorities.ConcentrationSHA256 != plan.Authorities.ConcentrationSHA256 || identity.Authorities.QualificationGateSet.ID != plan.GateSet.ID || identity.Authorities.QualificationGateSet.SHA256 != plan.GateSet.SHA256 {
		return errors.New("stage authority set differs from registered RIF identity")
	}
	if plan.DeterministicSeedPolicy.ID != identity.Authorities.DeterministicSeedPolicy.ID || plan.DeterministicSeedPolicy.SHA256 != identity.Authorities.DeterministicSeedPolicy.SHA256 || plan.Authorities.TransactionCostPolicy.ID != identity.Authorities.TransactionCostPolicy.ID || plan.Authorities.TransactionCostPolicy.SHA256 != identity.Authorities.TransactionCostPolicy.SHA256 {
		return errors.New("stage seed or cost policy substitution")
	}
	return nil
}

func executeStageArtifact(envelope rifbridge.StageExecutionEnvelope, authorization rifbridge.StageVariantAuthorization, configuration CandidateConfiguration, artifactJSON []byte) (StageVariantOutput, error) {
	identity, _, err := decodeIdentity(envelope.Snapshot.Identity)
	if err != nil {
		return StageVariantOutput{}, err
	}
	artifact, err := decodePartitionArtifact(artifactJSON)
	if err != nil {
		return StageVariantOutput{}, err
	}
	identityVariant := IdentityVariant{}
	for _, candidate := range identity.VariantLedger.Variants {
		if candidate.ID == authorization.Configuration.VariantID {
			identityVariant = candidate
			break
		}
	}
	if identityVariant.ID == "" {
		return StageVariantOutput{}, errors.New("authorized stage variant is absent from the RIF identity")
	}
	mode := ModeDevelopment
	if envelope.ExecutionSet.Plan.Stage == "VALIDATION" {
		mode = ModeValidation
	}
	partition, ok := findPartition(identity, envelope.ExecutionSet.Plan.Stage)
	if !ok {
		return StageVariantOutput{}, errors.New("authorized stage partition is absent from the RIF identity")
	}
	request := ExecutionRequest{
		SchemaVersion:        RequestSchemaVersion,
		Mode:                 mode,
		Protocol:             identity.Protocol,
		VariantLedger:        VariantLedger{SchemaVersion: VariantLedgerVersion, MaximumVariants: identity.VariantLedger.MaximumRegisteredVariants, V00ID: identity.VariantLedger.V00ID, StabilityNeighborhoods: append([]StabilityNeighborhood(nil), identity.VariantLedger.StabilityNeighborhoods...)},
		VariantID:            identityVariant.ID,
		ConfigurationSHA256:  identityVariant.ConfigurationSHA256,
		Dataset:              DatasetBinding{Checkpoint: identity.Dataset.Checkpoint, SourceIdentitySHA256: identity.Dataset.SourceIdentitySHA256, SealedBinarySHA256: identity.Dataset.SealedBinarySHA256, RequiredSymbols: append([]string(nil), identity.Dataset.RequiredSymbols...), EligibleInterval: identity.Dataset.EligibleInterval, ProhibitedPriorExposure: append([]Interval(nil), identity.Dataset.ProhibitedPriorExposure...), AvailabilityCutoff: identity.Dataset.AvailabilityCutoff},
		Partition:            partition,
		CandidateFamily:      identity.CandidateScope.FamilyID,
		Independence:         identity.Authorities.Independence,
		Uncertainty:          identity.Authorities.Uncertainty,
		Concentration:        HashIdentity{ID: "ak.engine.governance.concentration-decision.v1", SHA256: identity.Authorities.ConcentrationSHA256},
		QualificationGateSet: identity.Authorities.QualificationGateSet,
		CostPolicy:           identity.Authorities.TransactionCostPolicy,
		SeedPolicy:           identity.Authorities.DeterministicSeedPolicy,
		Runner:               RunnerIdentity{GitCommit: identity.Repositories.RunnerGitCommit, ExecutableSHA256: identity.Repositories.RunnerExecutableSHA256, V00SourceSHA256: V00SourceSHA256},
	}
	gates := qualification.AcceptedPR4B0GateSet()
	gateHash, err := qualification.PR4B0GateSetHash(gates)
	if err != nil {
		return StageVariantOutput{}, err
	}
	verified := VerifiedRequest{Request: request, Identity: identity, Variant: RegisteredVariant{ID: identityVariant.ID, Dimensions: append([]string(nil), identityVariant.Dimensions...), Configuration: configuration, ConfigurationSHA256: identityVariant.ConfigurationSHA256}, Gates: gates, GateSetSHA256: gateHash}
	if err := validateArtifactBindingWithPolicy(verified, artifact, false); err != nil {
		return StageVariantOutput{}, err
	}
	result, err := executeVerifiedArtifact(verified, artifact)
	if err != nil {
		return StageVariantOutput{}, err
	}
	resultBytes, err := json.Marshal(result)
	if err != nil {
		return StageVariantOutput{}, err
	}
	invocations := []StageAuthorityInvocationEvidence{
		{Identity: rifbridge.StageHashIdentity{ID: result.Concentration.ID, SHA256: result.Concentration.SHA256}, Invoked: result.Concentration.Invoked, EvidenceSHA256: result.Concentration.EvidenceSHA256},
		{Identity: rifbridge.StageHashIdentity{ID: result.Independence.ID, SHA256: result.Independence.SHA256}, Invoked: result.Independence.Invoked, EvidenceSHA256: result.Independence.EvidenceSHA256},
		{Identity: rifbridge.StageHashIdentity{ID: result.Uncertainty.ID, SHA256: result.Uncertainty.SHA256}, Invoked: result.Uncertainty.Invoked, EvidenceSHA256: result.Uncertainty.EvidenceSHA256},
	}
	sort.Slice(invocations, func(i, j int) bool { return invocations[i].Identity.ID < invocations[j].Identity.ID })
	outputManifest, err := canonicalHash(struct {
		ExecutionSetID  string `json:"execution_set_id"`
		PlanHash        string `json:"plan_hash"`
		AuthorizationID string `json:"authorization_id"`
		ResultSHA256    string `json:"result_sha256"`
	}{envelope.ExecutionSet.ExecutionSetID, envelope.ExecutionSet.Plan.PlanHash, authorization.AuthorizationID, result.ResultSHA256})
	if err != nil {
		return StageVariantOutput{}, err
	}
	return StageVariantOutput{ResultArtifact: resultBytes, OutputManifestSHA256: outputManifest, AuthorityInvocations: invocations, MandatoryGatesPassed: stageMandatoryGatesPassed(result.GateDecision)}, nil
}

func stageMandatoryGatesPassed(decision GateDecision) bool {
	for _, failed := range decision.FailedGateIDs {
		if failed != "minimum_stable_neighbors" {
			return false
		}
	}
	return true
}

func buildStageResultEnvelope(set rifbridge.StageExecutionSet, authorization rifbridge.StageVariantAuthorization, access rifbridge.StageVariantAccessReceipt, output StageVariantOutput) (StageResultEnvelope, error) {
	canonicalArtifact, err := canonicalJSONBytes(output.ResultArtifact)
	if err != nil {
		return StageResultEnvelope{}, errors.New("stage result artifact must be nonempty canonical JSON")
	}
	if !validNonPlaceholderHash(output.OutputManifestSHA256) {
		return StageResultEnvelope{}, errors.New("stage output manifest hash is mandatory")
	}
	if err := verifyInvocationEvidence(output.AuthorityInvocations, set.Plan.Authorities); err != nil {
		return StageResultEnvelope{}, err
	}
	runID, err := deterministicStageRunID(set.ExecutionSetID, authorization.AuthorizationID, authorization.Configuration)
	if err != nil {
		return StageResultEnvelope{}, err
	}
	result := StageResultEnvelope{SchemaVersion: StageResultEnvelopeVersion, ExecutionSetID: set.ExecutionSetID, PlanHash: set.Plan.PlanHash, AuthorizationID: authorization.AuthorizationID, DeterministicRunID: runID, Configuration: authorization.Configuration, Runner: authorization.Runner, Partition: authorization.Partition, Protocol: authorization.Protocol, Checkpoint: authorization.Checkpoint, Authorities: authorization.Authorities, GateSet: authorization.GateSet, AccessReceiptHash: access.RecordHash, ResultArtifact: canonicalArtifact, ResultArtifactSHA256: byteHash(canonicalArtifact), OutputManifestSHA256: output.OutputManifestSHA256, AuthorityInvocations: output.AuthorityInvocations, ResultStatus: "COMPLETED", MandatoryGatesPassed: output.MandatoryGatesPassed}
	result.EnvelopeHash, err = hashStageResult(result)
	return result, err
}

func EncodeStageResultEnvelopeJSON(result StageResultEnvelope) ([]byte, error) {
	canonicalArtifact, err := canonicalJSONBytes(result.ResultArtifact)
	if err != nil || !bytes.Equal(canonicalArtifact, result.ResultArtifact) || byteHash(canonicalArtifact) != result.ResultArtifactSHA256 {
		return nil, errors.New("stage result artifact hash mismatch")
	}
	want, err := hashStageResult(result)
	if err != nil || result.EnvelopeHash != want {
		return nil, errors.New("stage result envelope hash mismatch")
	}
	data, err := json.MarshalIndent(result, "", "  ")
	if err != nil {
		return nil, err
	}
	return append(data, '\n'), nil
}

func EncodeStageReadinessJSON(readiness StagePlanReadiness) ([]byte, error) {
	want, err := hashStageReadiness(readiness)
	if err != nil || readiness.ArtifactHash != want || readiness.DataLoads != 0 || readiness.CandidateEvents != 0 || readiness.CandidateOutcomes != 0 {
		return nil, errors.New("stage readiness artifact is invalid")
	}
	data, err := json.MarshalIndent(readiness, "", "  ")
	if err != nil {
		return nil, err
	}
	return append(data, '\n'), nil
}

func EncodeStageBatchManifestJSON(manifest StageBatchManifest) ([]byte, error) {
	want, err := hashBatchManifest(manifest)
	if err != nil || manifest.ManifestHash != want {
		return nil, errors.New("stage batch manifest hash mismatch")
	}
	data, err := json.MarshalIndent(manifest, "", "  ")
	if err != nil {
		return nil, err
	}
	return append(data, '\n'), nil
}

func verifyInvocationEvidence(actual []StageAuthorityInvocationEvidence, authorities rifbridge.StageAuthorityIdentity) error {
	expected := []rifbridge.StageHashIdentity{authorities.Independence, authorities.Uncertainty, {ID: "ak.engine.governance.concentration-decision.v1", SHA256: authorities.ConcentrationSHA256}}
	sort.Slice(expected, func(i, j int) bool { return expected[i].ID < expected[j].ID })
	if len(actual) != len(expected) {
		return errors.New("complete actual authority invocation evidence is required")
	}
	for i := range actual {
		if actual[i].Identity != expected[i] || !actual[i].Invoked || !validNonPlaceholderHash(actual[i].EvidenceSHA256) {
			return errors.New("authority invocation evidence mismatch")
		}
	}
	return nil
}

func exactConsumedStageAccess(set rifbridge.StageExecutionSet, authorization rifbridge.StageVariantAuthorization) (*rifbridge.StageVariantAccessReceipt, error) {
	var found *rifbridge.StageVariantAccessReceipt
	for i := range set.AccessReceipts {
		receipt := &set.AccessReceipts[i]
		if receipt.AuthorizationID == authorization.AuthorizationID {
			if found != nil {
				return nil, errors.New("variant authorization has duplicate access receipts")
			}
			found = receipt
		}
	}
	if found == nil || found.VariantID != authorization.Configuration.VariantID || found.Attempt != authorization.Attempt {
		return nil, errors.New("execution requires exactly one durable exact RIF access receipt")
	}
	return found, nil
}

func expectedValidationIDs(identity ResearchIdentityV4, nominee string) ([]string, error) {
	ids := []string{nominee}
	found := false
	for _, neighborhood := range identity.VariantLedger.StabilityNeighborhoods {
		if neighborhood.VariantID == nominee {
			ids = append(ids, neighborhood.NeighborIDs...)
			found = true
			break
		}
	}
	if !found || len(ids) < 2 {
		return nil, errors.New("nominee has no registered mandatory stability neighborhood")
	}
	sort.Slice(ids, func(i, j int) bool { return stageVariantLess(ids[i], ids[j]) })
	return ids, nil
}

func validationSubsetDerivable(identity ResearchIdentityV4, configurations []rifbridge.RegisteredConfigurationIdentity) bool {
	for _, configuration := range configurations {
		if _, err := expectedValidationIDs(identity, configuration.VariantID); err != nil {
			return false
		}
	}
	return true
}

func numericStageOrder(configurations []rifbridge.RegisteredConfigurationIdentity) bool {
	for i := range configurations {
		if _, ok := stageVariantNumber(configurations[i].VariantID); !ok {
			return false
		}
		if i > 0 && !stageVariantLess(configurations[i-1].VariantID, configurations[i].VariantID) {
			return false
		}
	}
	return true
}

func stageVariantNumber(id string) (int, bool) {
	if len(id) < 2 || id[0] != 'V' {
		return 0, false
	}
	n, err := strconv.Atoi(id[1:])
	return n, err == nil && n >= 0
}
func stageVariantLess(a, b string) bool {
	left, lok := stageVariantNumber(a)
	right, rok := stageVariantNumber(b)
	if lok && rok && left != right {
		return left < right
	}
	return a < b
}

func errRunnerIdentity(runner rifbridge.RunnerImplementationIdentity) error {
	if runner.SchemaVersion != rifbridge.RunnerImplementationSchemaVersion || len(runner.SourceCommit) != 40 || runner.PackageIdentity.ID == "" || !validSHA(runner.PackageIdentity.SHA256) || !validSHA(runner.BuildInputsSHA256) || runner.CompilerIdentity == "" || runner.BuildModeIdentity.ID == "" || !validSHA(runner.BuildModeIdentity.SHA256) || !validSHA(runner.BinarySHA256) {
		return errors.New("complete pre-execution runner identity is required")
	}
	return nil
}

func deterministicStageRunID(setID, authorizationID string, configuration rifbridge.RegisteredConfigurationIdentity) (string, error) {
	return canonicalHash(struct {
		ExecutionSetID  string                                    `json:"execution_set_id"`
		AuthorizationID string                                    `json:"authorization_id"`
		Configuration   rifbridge.RegisteredConfigurationIdentity `json:"registered_configuration"`
	}{setID, authorizationID, configuration})
}

func canonicalJSONBytes(raw json.RawMessage) ([]byte, error) {
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.UseNumber()
	var value any
	if err := decoder.Decode(&value); err != nil {
		return nil, err
	}
	var trailing any
	if err := decoder.Decode(&trailing); err != io.EOF {
		return nil, errors.New("trailing JSON data")
	}
	return json.Marshal(value)
}

func strictJSON(data []byte, target any) error {
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(target); err != nil {
		return err
	}
	var trailing any
	if err := decoder.Decode(&trailing); err != io.EOF {
		return errors.New("trailing JSON data")
	}
	return nil
}

func validNonPlaceholderHash(value string) bool {
	return validSHA(value) && value != "sha256:"+strings.Repeat("0", 64)
}
func byteHash(data []byte) string {
	digest := sha256.Sum256(data)
	return "sha256:" + hex.EncodeToString(digest[:])
}
func hashPrebuildReceipt(value RunnerPrebuildReceipt) (string, error) {
	value.ReceiptHash = ""
	return canonicalHash(value)
}
func hashStageReadiness(value StagePlanReadiness) (string, error) {
	value.ArtifactHash = ""
	return canonicalHash(value)
}
func hashStageResult(value StageResultEnvelope) (string, error) {
	value.EnvelopeHash = ""
	return canonicalHash(value)
}
func hashBatchReceipt(value StageBatchExecutionReceipt) (string, error) {
	value.ReceiptHash = ""
	return canonicalHash(value)
}
func hashBatchManifest(value StageBatchManifest) (string, error) {
	value.ManifestHash = ""
	return canonicalHash(value)
}
func hashStageProgress(value StageExecutionProgress) (string, error) {
	value.IntegrityHash = ""
	return canonicalHash(value)
}

func (r *StageBatchRunner) acquireLock() (*os.File, error) {
	if err := os.MkdirAll(filepath.Dir(r.progressPath), 0o700); err != nil {
		return nil, err
	}
	fd, err := syscall.Open(r.progressPath+".lock", syscall.O_CREAT|syscall.O_RDWR|syscall.O_CLOEXEC|syscall.O_NOFOLLOW, 0o600)
	if err != nil {
		return nil, err
	}
	file := os.NewFile(uintptr(fd), r.progressPath+".lock")
	if err := syscall.Flock(fd, syscall.LOCK_EX); err != nil {
		_ = file.Close()
		return nil, err
	}
	return file, nil
}

func releaseStageLock(file *os.File) {
	if file != nil {
		_ = syscall.Flock(int(file.Fd()), syscall.LOCK_UN)
		_ = file.Close()
	}
}

func (r *StageBatchRunner) loadOrInitialize(set rifbridge.StageExecutionSet) (StageExecutionProgress, error) {
	progress, err := r.readProgress()
	if errors.Is(err, os.ErrNotExist) {
		progress = StageExecutionProgress{SchemaVersion: StageProgressSchemaVersion, ExecutionSetID: set.ExecutionSetID, PlanHash: set.Plan.PlanHash, State: "OPEN", Receipts: []StageBatchExecutionReceipt{}}
		if err := r.writeProgress(progress); err != nil {
			return StageExecutionProgress{}, err
		}
		return progress, nil
	}
	if err != nil {
		return StageExecutionProgress{}, err
	}
	if progress.ExecutionSetID != set.ExecutionSetID || progress.PlanHash != set.Plan.PlanHash {
		return StageExecutionProgress{}, errors.New("durable progress belongs to another stage execution set")
	}
	return progress, nil
}

func (r *StageBatchRunner) readProgress() (StageExecutionProgress, error) {
	info, err := os.Lstat(r.progressPath)
	if err != nil {
		return StageExecutionProgress{}, err
	}
	if !info.Mode().IsRegular() {
		return StageExecutionProgress{}, errors.New("stage progress path is not a regular file")
	}
	data, err := os.ReadFile(r.progressPath)
	if err != nil {
		return StageExecutionProgress{}, err
	}
	var progress StageExecutionProgress
	if err := strictJSON(data, &progress); err != nil {
		return StageExecutionProgress{}, err
	}
	if err := validateProgress(progress); err != nil {
		return StageExecutionProgress{}, err
	}
	return progress, nil
}

func validateProgress(progress StageExecutionProgress) error {
	if progress.SchemaVersion != StageProgressSchemaVersion || progress.ExecutionSetID == "" || !validSHA(progress.PlanHash) || (progress.State != "OPEN" && progress.State != "SEALED") || progress.NextOrdinal != len(progress.Receipts) {
		return errors.New("stage progress is incomplete")
	}
	previous := ""
	for ordinal, receipt := range progress.Receipts {
		if receipt.SchemaVersion != StageBatchReceiptVersion || receipt.ExecutionSetID != progress.ExecutionSetID || receipt.PlanHash != progress.PlanHash || receipt.Ordinal != ordinal || receipt.PreviousHash != previous {
			return errors.New("stage progress receipt chain is invalid")
		}
		want, err := hashBatchReceipt(receipt)
		if err != nil || want != receipt.ReceiptHash {
			return errors.New("stage progress receipt hash mismatch")
		}
		previous = receipt.ReceiptHash
	}
	if progress.State == "SEALED" && (progress.ActiveAttempt != nil || progress.Manifest == nil) {
		return errors.New("sealed stage progress is incomplete")
	}
	if progress.Manifest != nil {
		want, err := hashBatchManifest(*progress.Manifest)
		if err != nil || want != progress.Manifest.ManifestHash {
			return errors.New("stage progress manifest hash mismatch")
		}
	}
	want, err := hashStageProgress(progress)
	if err != nil || want != progress.IntegrityHash {
		return errors.New("stage progress integrity hash mismatch")
	}
	return nil
}

func (r *StageBatchRunner) writeProgress(progress StageExecutionProgress) error {
	hash, err := hashStageProgress(progress)
	if err != nil {
		return err
	}
	progress.IntegrityHash = hash
	data, err := json.MarshalIndent(progress, "", "  ")
	if err != nil {
		return err
	}
	data = append(data, '\n')
	temporary, err := os.CreateTemp(filepath.Dir(r.progressPath), ".stage-progress-*")
	if err != nil {
		return err
	}
	temporaryPath := temporary.Name()
	defer os.Remove(temporaryPath)
	if err := temporary.Chmod(0o600); err != nil {
		_ = temporary.Close()
		return err
	}
	if _, err := temporary.Write(data); err != nil {
		_ = temporary.Close()
		return err
	}
	if err := temporary.Sync(); err != nil {
		_ = temporary.Close()
		return err
	}
	if err := temporary.Close(); err != nil {
		return err
	}
	if info, err := os.Lstat(r.progressPath); err == nil && !info.Mode().IsRegular() {
		return errors.New("stage progress target is not a regular file")
	} else if err != nil && !errors.Is(err, os.ErrNotExist) {
		return err
	}
	if err := os.Rename(temporaryPath, r.progressPath); err != nil {
		return err
	}
	directory, err := os.Open(filepath.Dir(r.progressPath))
	if err == nil {
		_ = directory.Sync()
		_ = directory.Close()
	}
	return nil
}
