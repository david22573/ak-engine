package epochorchestrator

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"sort"
	"strings"
	"time"

	"github.com/david22573/ak-engine/internal/partitionpipeline"
	"github.com/david22573/ak-engine/internal/qualificationrunner"
	"github.com/david22573/ak-engine/internal/rifbridge"
	"github.com/david22573/ak-rif/research"
)

const (
	ConfigSchemaVersion = "ak.engine.pr4b0_r1.epoch_orchestrator_config.v1"
	StateSchemaVersion  = "ak.engine.pr4b0_r1.epoch_orchestrator_state.v1"
	ReadyStatus         = "READY_TO_REGISTER_PROTOCOL"
	NotReadyStatus      = "NOT_READY_TO_REGISTER_PROTOCOL"
)

type RepositoryCheck struct {
	Path   string `json:"path"`
	Commit string `json:"commit"`
}

type Config struct {
	SchemaVersion string                                `json:"schema_version"`
	Synthetic     bool                                  `json:"synthetic_fixture"`
	Repositories  map[string]RepositoryCheck            `json:"repositories"`
	Protocol      json.RawMessage                       `json:"canonical_protocol"`
	Identity      research.IdentityV4                   `json:"research_identity"`
	VariantLedger qualificationrunner.VariantLedger     `json:"engine_variant_ledger"`
	Runner        research.RunnerImplementationIdentity `json:"runner_identity"`
	RunnerBuild   *ProductionRunnerBuild                `json:"production_runner_build,omitempty"`
	Plans         map[string]partitionpipeline.Plan     `json:"partition_plans"`
	ConfigSHA256  string                                `json:"config_sha256"`
}

type State struct {
	SchemaVersion     string `json:"schema_version"`
	ConfigSHA256      string `json:"config_sha256"`
	PreflightStatus   string `json:"preflight_status"`
	Phase             string `json:"phase"`
	ProtocolSHA256    string `json:"protocol_sha256,omitempty"`
	FinalResultSHA256 string `json:"final_result_sha256,omitempty"`
	FinalPassed       bool   `json:"final_passed"`
	LastError         string `json:"last_error,omitempty"`
	StateSHA256       string `json:"state_sha256"`
}

type Status struct {
	Orchestrator       State  `json:"orchestrator"`
	RIFState           string `json:"rif_lifecycle_state,omitempty"`
	RIFSequence        uint64 `json:"rif_sequence,omitempty"`
	RIFIntegritySHA256 string `json:"rif_integrity_sha256,omitempty"`
	NextOperation      string `json:"next_operation"`
}

type Orchestrator struct {
	root   string
	config Config
	now    func() time.Time
}

func DecodeConfig(data []byte) (Config, error) {
	var cfg Config
	if err := strictJSON(data, &cfg); err != nil {
		return Config{}, err
	}
	want, err := SealConfig(cfg)
	if err != nil {
		return Config{}, err
	}
	if !reflect.DeepEqual(cfg, want) {
		return Config{}, errors.New("orchestrator configuration is noncanonical or hash-mutated")
	}
	return cfg, nil
}
func EncodeConfig(cfg Config) ([]byte, error) {
	sealed, err := SealConfig(cfg)
	if err != nil {
		return nil, err
	}
	data, err := json.Marshal(sealed)
	if err != nil {
		return nil, err
	}
	return append(data, '\n'), nil
}
func SealConfig(cfg Config) (Config, error) {
	cfg.SchemaVersion = ConfigSchemaVersion
	cfg.ConfigSHA256 = ""
	hash, err := canonicalHash(cfg)
	if err != nil {
		return Config{}, err
	}
	cfg.ConfigSHA256 = hash
	return cfg, nil
}

// ValidateConfigStructure performs the complete immutable configuration and
// plan validation without creating an epoch root, registry, authorization,
// access receipt, or outcome.
func ValidateConfigStructure(cfg Config) error {
	sealed, err := SealConfig(cfg)
	if err != nil {
		return err
	}
	if cfg.ConfigSHA256 != sealed.ConfigSHA256 {
		return errors.New("configuration is not canonically sealed")
	}
	orchestrator := &Orchestrator{config: sealed}
	return orchestrator.validateConfig()
}

func New(root string, cfg Config) (*Orchestrator, error) {
	sealed, err := SealConfig(cfg)
	if err != nil {
		return nil, err
	}
	if cfg.ConfigSHA256 != "" && cfg.ConfigSHA256 != sealed.ConfigSHA256 {
		return nil, errors.New("configuration hash mismatch")
	}
	cfg = sealed
	if root == "" || filepath.Clean(root) != root {
		return nil, errors.New("epoch root must be explicit and normalized")
	}
	return &Orchestrator{root: root, config: cfg, now: time.Now}, nil
}

func (o *Orchestrator) Preflight() (string, error) {
	if err := o.validateConfig(); err != nil {
		o.recordNotReady(err)
		return NotReadyStatus, err
	}
	if !o.config.Synthetic {
		for name, repo := range o.config.Repositories {
			if err := verifyRepository(repo); err != nil {
				o.recordNotReady(fmt.Errorf("%s repository: %w", name, err))
				return NotReadyStatus, err
			}
		}
		if o.config.RunnerBuild == nil {
			o.recordNotReady(errors.New("production deterministic runner build contract is required"))
			return NotReadyStatus, errors.New("production deterministic runner build contract is required")
		}
		built, err := ComputeProductionRunnerIdentity(*o.config.RunnerBuild, o.config.Runner.SourceCommit)
		if err != nil || !reflect.DeepEqual(built, o.config.Runner) {
			cause := errors.New("deterministic production runner build identity mismatch")
			if err != nil {
				cause = fmt.Errorf("deterministic production runner build failed: %w", err)
			}
			o.recordNotReady(cause)
			return NotReadyStatus, cause
		}
	}
	if err := o.ensureRoot(); err != nil {
		return NotReadyStatus, err
	}
	registry := o.partitionRegistryPath()
	if _, err := os.Stat(registry); errors.Is(err, os.ErrNotExist) {
		if err := partitionpipeline.CreateRegistry(registry); err != nil {
			return NotReadyStatus, err
		}
	} else if err != nil {
		return NotReadyStatus, err
	}
	for _, name := range []string{"DEVELOPMENT", "VALIDATION", "FINAL_HOLDOUT"} {
		plan := o.config.Plans[name]
		if err := partitionpipeline.RegisterPlan(registry, plan); err != nil {
			o.recordNotReady(err)
			return NotReadyStatus, err
		}
	}
	state := State{SchemaVersion: StateSchemaVersion, ConfigSHA256: o.config.ConfigSHA256, PreflightStatus: ReadyStatus, Phase: "PREFLIGHT_VERIFIED"}
	if err := o.writeState(state); err != nil {
		return NotReadyStatus, err
	}
	return ReadyStatus, nil
}

func (o *Orchestrator) CommitProtocol() error {
	state, err := o.requirePhase("PREFLIGHT_VERIFIED")
	if err != nil {
		return err
	}
	hash := byteHash(o.config.Protocol)
	if hash != o.config.Identity.Protocol.SHA256 {
		return errors.New("canonical protocol bytes differ from registered identity")
	}
	if err := atomicWrite(filepath.Join(o.root, "protocol.json"), append(bytes.TrimSpace(o.config.Protocol), '\n'), 0o600); err != nil {
		return err
	}
	state.Phase = "PROTOCOL_COMMITTED"
	state.ProtocolSHA256 = hash
	return o.writeState(state)
}

func (o *Orchestrator) RegisterIdentity() error {
	state, err := o.requirePhase("PROTOCOL_COMMITTED")
	if err != nil {
		return err
	}
	if _, err := os.Stat(o.rifPath()); errors.Is(err, os.ErrNotExist) {
		authority, err := research.CreateAuthority(o.rifPath())
		if err != nil {
			return err
		}
		if _, err := authority.RegisterIdentity(o.config.Identity); err != nil {
			return err
		}
	} else if err != nil {
		return err
	} else {
		authority, err := research.OpenAuthority(o.rifPath())
		if err != nil {
			return err
		}
		snapshot, err := authority.Snapshot()
		if err != nil || snapshot.IdentityHash == "" {
			return errors.New("existing RIF identity state is invalid")
		}
	}
	state.Phase = "RESEARCH_IDENTITY_REGISTERED"
	return o.writeState(state)
}

func (o *Orchestrator) ReserveHoldout() error {
	state, err := o.requirePhase("RESEARCH_IDENTITY_REGISTERED")
	if err != nil {
		return err
	}
	authority, snapshot, err := o.authority()
	if err != nil {
		return err
	}
	ledgerHash, err := research.HashVariantLedger(o.config.Identity.VariantLedger)
	if err != nil {
		return err
	}
	authorityHash, err := research.HashAuthoritySet(o.config.Identity.Authorities)
	if err != nil {
		return err
	}
	final := findPartition(o.config.Identity, research.PartitionFinalHoldout)
	_, err = authority.ReserveHoldout(research.ReservationRequest{SchemaVersion: research.ReservationSchemaVersion, ResearchIdentityHash: snapshot.IdentityHash, FinalHoldout: final, ProtocolSHA256: o.config.Identity.Protocol.SHA256, CheckpointSHA256: o.config.Identity.Dataset.Checkpoint.SHA256, VariantLedgerSHA256: ledgerHash, AuthoritySetSHA256: authorityHash, ExpectedSequence: snapshot.Sequence, ExpectedStateHash: snapshot.IntegrityHash})
	if err != nil {
		return err
	}
	state.Phase = "HOLDOUT_RESERVED"
	return o.writeState(state)
}

func (o *Orchestrator) AuthorizeDevelopmentSet() error {
	state, err := o.requirePhase("HOLDOUT_RESERVED")
	if err != nil {
		return err
	}
	authority, snapshot, err := o.authority()
	if err != nil {
		return err
	}
	configs, err := o.registeredConfigurations()
	if err != nil {
		return err
	}
	set, err := authority.AuthorizeDevelopmentExecutionSet(research.StageExecutionSetRequest{SchemaVersion: research.StageExecutionSetVersion, ExpectedSequence: snapshot.Sequence, ExpectedStateHash: snapshot.IntegrityHash, Runner: o.config.Runner, Configurations: configs, OrderingRule: "NUMERIC_VARIANT_ID_ASCENDING", Complete: true})
	if err != nil {
		return err
	}
	if set.Plan.Partition.RequiredSymbolCoverageSHA256 != o.config.Plans["DEVELOPMENT"].PlanSHA256 {
		return errors.New("RIF DEVELOPMENT plan does not bind production partition plan")
	}
	state.Phase = "DEVELOPMENT_SET_AUTHORIZED"
	return o.writeState(state)
}

func (o *Orchestrator) RunDevelopmentSet() error {
	return o.runStage("DEVELOPMENT")
}
func (o *Orchestrator) SealDevelopmentSet() error {
	state, err := o.requirePhase("DEVELOPMENT_SET_AUTHORIZED")
	if err != nil {
		return err
	}
	if err := o.sealStage("DEVELOPMENT"); err != nil {
		return err
	}
	state.Phase = "DEVELOPMENT_SET_SEALED"
	return o.writeState(state)
}
func (o *Orchestrator) DeriveNominee() error {
	state, err := o.requirePhase("DEVELOPMENT_SET_SEALED")
	if err != nil {
		return err
	}
	authority, snapshot, err := o.authority()
	if err != nil {
		return err
	}
	nominee, err := authority.SelectDevelopmentNominee(snapshot.Sequence, snapshot.IntegrityHash)
	if err != nil {
		return err
	}
	if !nominee.Exists {
		state.Phase = "REJECTED"
	} else {
		state.Phase = "NOMINEE_DERIVED"
	}
	return o.writeState(state)
}

func (o *Orchestrator) AuthorizeValidationSet() error {
	state, err := o.requirePhase("NOMINEE_DERIVED")
	if err != nil {
		return err
	}
	authority, snapshot, err := o.authority()
	if err != nil {
		return err
	}
	set, err := authority.AuthorizeValidationExecutionSet(snapshot.Sequence, snapshot.IntegrityHash)
	if err != nil {
		return err
	}
	if set.Plan.Partition.RequiredSymbolCoverageSHA256 != o.config.Plans["VALIDATION"].PlanSHA256 {
		return errors.New("RIF VALIDATION plan does not bind production partition plan")
	}
	state.Phase = "VALIDATION_SET_AUTHORIZED"
	return o.writeState(state)
}
func (o *Orchestrator) RunValidationSet() error {
	if _, err := o.requirePhase("VALIDATION_SET_AUTHORIZED"); err != nil {
		return err
	}
	return o.runStage("VALIDATION")
}
func (o *Orchestrator) SealValidationSet() error {
	state, err := o.requirePhase("VALIDATION_SET_AUTHORIZED")
	if err != nil {
		return err
	}
	if err := o.sealStage("VALIDATION"); err != nil {
		return err
	}
	state.Phase = "VALIDATION_SET_SEALED"
	return o.writeState(state)
}

func (o *Orchestrator) FreezeCandidate() error {
	state, err := o.requirePhase("VALIDATION_SET_SEALED")
	if err != nil {
		return err
	}
	authority, snapshot, err := o.authority()
	if err != nil {
		return err
	}
	if snapshot.DevelopmentNominee == nil || !snapshot.DevelopmentNominee.Exists {
		return errors.New("sealed DEVELOPMENT nominee missing")
	}
	candidate := research.FrozenCandidate{VariantID: snapshot.DevelopmentNominee.VariantID, ConfigurationSHA256: snapshot.DevelopmentNominee.ConfigurationSHA256, ExecutableSHA256: o.config.Runner.BinarySHA256, ProtocolSHA256: o.config.Identity.Protocol.SHA256, CheckpointSHA256: o.config.Identity.Dataset.Checkpoint.SHA256, IndependenceSHA256: o.config.Identity.Authorities.Independence.SHA256, UncertaintySHA256: o.config.Identity.Authorities.Uncertainty.SHA256, ConcentrationSHA256: o.config.Identity.Authorities.ConcentrationSHA256, QualificationGateSHA256: o.config.Identity.Authorities.QualificationGateSet.SHA256, NoUnresolvedDefaults: true}
	if _, err := authority.FreezeCandidate(snapshot.Sequence, snapshot.IntegrityHash, candidate); err != nil {
		return err
	}
	state.Phase = "CANDIDATE_FROZEN"
	return o.writeState(state)
}

func (o *Orchestrator) AuthorizeFinalHoldout() error {
	state, err := o.requirePhase("CANDIDATE_FROZEN")
	if err != nil {
		return err
	}
	authority, snapshot, err := o.authority()
	if err != nil {
		return err
	}
	binding, err := o.finalBinding(snapshot)
	if err != nil {
		return err
	}
	if _, err := authority.AuthorizeFinalHoldout(research.TransitionRequest{ExpectedSequence: snapshot.Sequence, ExpectedStateHash: snapshot.IntegrityHash, Binding: binding}); err != nil {
		return err
	}
	state.Phase = "FINAL_HOLDOUT_AUTHORIZED"
	return o.writeState(state)
}

func (o *Orchestrator) RunFinalHoldout() error {
	state, err := o.requirePhase("FINAL_HOLDOUT_AUTHORIZED")
	if err != nil {
		return err
	}
	authority, snapshot, err := o.authority()
	if err != nil {
		return err
	}
	binding, err := o.finalBinding(snapshot)
	if err != nil {
		return err
	}
	authorization := latestAuthorization(snapshot, research.PartitionFinalHoldout)
	if authorization == nil {
		return errors.New("FINAL_HOLDOUT authorization missing")
	}
	receipt, err := authority.ConsumeBeforeAccess(*authorization, binding, func() error { return nil })
	if err != nil {
		return err
	}
	artifactBytes, err := o.materializeAndConsumeFinal(*authorization, receipt, snapshot.FrozenCandidate.VariantID)
	if err != nil {
		return o.blockAfterAccess(err)
	}
	envelope, err := authority.ExportEnvelope(authorization.AuthorizationID)
	if err != nil {
		return err
	}
	engineEnvelope := rifbridge.ResearchGovernanceEnvelope{}
	if err := convert(envelope, &engineEnvelope); err != nil {
		return err
	}
	engineIdentity, err := o.engineIdentity()
	if err != nil {
		return err
	}
	variant := findEngineVariant(o.config.VariantLedger, snapshot.FrozenCandidate.VariantID)
	partition := findEnginePartition(engineIdentity, "FINAL_HOLDOUT")
	request := qualificationrunner.ExecutionRequest{SchemaVersion: qualificationrunner.RequestSchemaVersion, Mode: qualificationrunner.ModeFinalHoldout, RIF: engineEnvelope, Protocol: engineIdentity.Protocol, VariantLedger: o.config.VariantLedger, VariantID: variant.ID, ConfigurationSHA256: variant.ConfigurationSHA256, Dataset: qualificationrunner.DatasetBinding{Checkpoint: engineIdentity.Dataset.Checkpoint, SourceIdentitySHA256: engineIdentity.Dataset.SourceIdentitySHA256, SealedBinarySHA256: engineIdentity.Dataset.SealedBinarySHA256, RequiredSymbols: engineIdentity.Dataset.RequiredSymbols, CandidateTargetSymbols: engineIdentity.Dataset.CandidateTargetSymbols, ContextOnlySymbols: engineIdentity.Dataset.ContextOnlySymbols, UniverseContractSHA256: engineIdentity.Dataset.UniverseContractSHA256, EligibleInterval: engineIdentity.Dataset.EligibleInterval, ProhibitedPriorExposure: engineIdentity.Dataset.ProhibitedPriorExposure, AvailabilityCutoff: engineIdentity.Dataset.AvailabilityCutoff}, Partition: partition, CandidateFamily: qualificationrunner.V00CandidateFamily, Independence: engineIdentity.Authorities.Independence, Uncertainty: engineIdentity.Authorities.Uncertainty, Concentration: qualificationrunner.HashIdentity{ID: research.AcceptedConcentrationID, SHA256: engineIdentity.Authorities.ConcentrationSHA256}, QualificationGateSet: engineIdentity.Authorities.QualificationGateSet, CostPolicy: engineIdentity.Authorities.TransactionCostPolicy, SeedPolicy: engineIdentity.Authorities.DeterministicSeedPolicy, Runner: qualificationrunner.RunnerIdentity{GitCommit: engineIdentity.Repositories.RunnerGitCommit, ExecutableSHA256: engineIdentity.Repositories.RunnerExecutableSHA256, V00SourceSHA256: qualificationrunner.V00SourceSHA256}}
	result, err := qualificationrunner.Execute(request, artifactBytes)
	if err != nil {
		return o.blockAfterAccess(err)
	}
	data, err := qualificationrunner.EncodeResultArtifact(result)
	if err != nil {
		return err
	}
	if err := atomicWrite(filepath.Join(o.root, "final-holdout-result.json"), data, 0o600); err != nil {
		return err
	}
	state.FinalResultSHA256 = result.ResultSHA256
	state.FinalPassed = result.GateDecision.Passed
	return o.writeState(state)
}

func (o *Orchestrator) SealFinalHoldout() error {
	state, err := o.requirePhase("FINAL_HOLDOUT_AUTHORIZED")
	if err != nil {
		return err
	}
	if state.FinalResultSHA256 == "" {
		return errors.New("FINAL_HOLDOUT has no sealed result")
	}
	authority, snapshot, err := o.authority()
	if err != nil {
		return err
	}
	binding, err := o.finalBinding(snapshot)
	if err != nil {
		return err
	}
	authorization := latestAuthorization(snapshot, research.PartitionFinalHoldout)
	receipt := latestAccess(snapshot, authorization.AuthorizationID)
	if receipt == nil {
		return errors.New("FINAL_HOLDOUT access receipt missing")
	}
	if _, err := authority.SealFinalHoldout(research.SealRequest{ExpectedSequence: snapshot.Sequence, ExpectedStateHash: snapshot.IntegrityHash, Binding: binding, AccessReceiptHash: receipt.RecordHash, ExecutionReceiptSHA256: state.FinalResultSHA256, ResultSealSHA256: state.FinalResultSHA256}); err != nil {
		return err
	}
	state.Phase = "FINAL_HOLDOUT_SEALED"
	return o.writeState(state)
}

func (o *Orchestrator) Closeout() error {
	state, err := o.requirePhase("FINAL_HOLDOUT_SEALED")
	if err != nil {
		return err
	}
	authority, snapshot, err := o.authority()
	if err != nil {
		return err
	}
	if state.FinalPassed {
		_, err = authority.Qualify(snapshot.Sequence, snapshot.IntegrityHash, state.FinalResultSHA256)
		state.Phase = "QUALIFIED"
	} else {
		_, err = authority.RejectPerformance(snapshot.Sequence, snapshot.IntegrityHash, state.FinalResultSHA256, "frozen candidate failed accepted FINAL_HOLDOUT gates")
		state.Phase = "REJECTED"
	}
	if err != nil {
		return err
	}
	return o.writeState(state)
}

func (o *Orchestrator) Resume() (Status, error) { return o.Status() }
func (o *Orchestrator) Status() (Status, error) {
	state, err := o.readState()
	if err != nil {
		return Status{}, err
	}
	status := Status{Orchestrator: state, NextOperation: nextOperation(state.Phase)}
	if _, err := os.Stat(o.rifPath()); err == nil {
		authority, err := research.OpenAuthority(o.rifPath())
		if err != nil {
			return Status{}, err
		}
		snapshot, err := authority.Snapshot()
		if err != nil {
			return Status{}, err
		}
		status.RIFState = string(snapshot.State)
		status.RIFSequence = snapshot.Sequence
		status.RIFIntegritySHA256 = snapshot.IntegrityHash
	}
	return status, nil
}

func (o *Orchestrator) runStage(stage string) error {
	authority, snapshot, err := o.authority()
	if err != nil {
		return err
	}
	set := findStageSet(snapshot, stage)
	if set == nil || set.CompletionState != "OPEN" {
		return errors.New("active open RIF stage set missing")
	}
	bridgeRunner := rifbridge.RunnerImplementationIdentity{}
	if err := convert(o.config.Runner, &bridgeRunner); err != nil {
		return err
	}
	runner, err := qualificationrunner.NewStageBatchRunner(filepath.Join(o.root, "engine-"+strings.ToLower(stage)+"-progress.json"), bridgeRunner)
	if err != nil {
		return err
	}
	for _, authorization := range set.Authorizations {
		if latestStageAuthorization(set, authorization.Configuration.VariantID).AuthorizationID != authorization.AuthorizationID {
			continue
		}
		if stageReceipt(set, authorization.Configuration.VariantID) != nil {
			continue
		}
		if priorAccess := stageAccess(set, authorization.AuthorizationID); priorAccess != nil {
			proof, proofErr := partitionpipeline.ProveZeroAccess(o.partitionRegistryPath(), o.config.Plans[stage].PlanSHA256)
			if proofErr != nil {
				return o.blockAfterAccess(fmt.Errorf("indeterminate prior stage access blocks implicit retry: %w", proofErr))
			}
			retry, retryErr := authority.AuthorizeZeroAccessRetry(snapshot.Sequence, snapshot.IntegrityHash, set.ExecutionSetID, authorization.AuthorizationID, priorAccess.RecordHash, proof.ProofSHA256, proof.RowsOpened, proof.OutcomeArtifacts)
			if retryErr != nil {
				return o.blockAfterAccess(retryErr)
			}
			authorization = retry
			snapshot, err = authority.Snapshot()
			if err != nil {
				return err
			}
			set = findStageSet(snapshot, stage)
		}
		access, err := authority.ConsumeStageVariantBeforeAccess(authorization, func() error { return nil })
		if err != nil {
			return err
		}
		snapshot, err = authority.Snapshot()
		if err != nil {
			return err
		}
		artifact, err := o.materializeAndConsumeStage(authorization, access)
		if err != nil {
			return o.blockAfterAccess(err)
		}
		envelope, err := authority.ExportStageExecutionEnvelope(set.ExecutionSetID, authorization.AuthorizationID)
		if err != nil {
			return err
		}
		bridgeEnvelope := rifbridge.StageExecutionEnvelope{}
		if err := convert(envelope, &bridgeEnvelope); err != nil {
			return err
		}
		result, _, err := runner.ExecuteAuthorizedVariant(bridgeEnvelope, artifact)
		if err != nil {
			return o.blockAfterAccess(err)
		}
		rifResult := research.ExecutionResultEnvelope{}
		if err := convert(result, &rifResult); err != nil {
			return err
		}
		snapshot, err = authority.Snapshot()
		if err != nil {
			return err
		}
		if _, err := authority.RecordStageExecutionResult(snapshot.Sequence, snapshot.IntegrityHash, rifResult); err != nil {
			return o.blockAfterAccess(err)
		}
		snapshot, err = authority.Snapshot()
		if err != nil {
			return err
		}
		set = findStageSet(snapshot, stage)
	}
	return nil
}

func (o *Orchestrator) sealStage(stage string) error {
	authority, snapshot, err := o.authority()
	if err != nil {
		return err
	}
	set := findStageSet(snapshot, stage)
	if set == nil {
		return errors.New("stage set missing")
	}
	if len(set.ExecutionReceipts) != set.Plan.ExpectedExecutions {
		return errors.New("incomplete stage set cannot seal")
	}
	bridgeRunner := rifbridge.RunnerImplementationIdentity{}
	convert(o.config.Runner, &bridgeRunner)
	runner, err := qualificationrunner.NewStageBatchRunner(filepath.Join(o.root, "engine-"+strings.ToLower(stage)+"-progress.json"), bridgeRunner)
	if err != nil {
		return err
	}
	last := set.Authorizations[set.Plan.ExpectedExecutions-1]
	envelope, err := authority.ExportStageExecutionEnvelope(set.ExecutionSetID, last.AuthorizationID)
	if err != nil {
		return err
	}
	bridgeEnvelope := rifbridge.StageExecutionEnvelope{}
	if err := convert(envelope, &bridgeEnvelope); err != nil {
		return err
	}
	if _, err := runner.Seal(bridgeEnvelope); err != nil {
		return err
	}
	snapshot, err = authority.Snapshot()
	if err != nil {
		return err
	}
	_, err = authority.SealStageExecutionSet(snapshot.Sequence, snapshot.IntegrityHash, set.ExecutionSetID)
	return err
}

func (o *Orchestrator) materializeAndConsumeStage(auth research.StageVariantAuthorization, access research.StageVariantAccessReceipt) ([]byte, error) {
	plan := o.config.Plans[string(auth.Partition.Name)]
	state, err := partitionpipeline.PlanState(o.partitionRegistryPath(), plan.PlanSHA256)
	if err != nil {
		return nil, err
	}
	if state == partitionpipeline.PlanVerified {
		authorization, err := partitionpipeline.SealMaterializationAuthorization(partitionpipeline.MaterializationAuthorization{PlanSHA256: plan.PlanSHA256, CheckpointSHA256: plan.Checkpoint.SHA256, Partition: plan.PartitionName, RIFAuthorizationID: auth.AuthorizationID, RIFAuthorizationSHA256: auth.RecordHash, RIFAccessReceiptSHA256: access.RecordHash, AuthorizedAt: o.now().UTC()})
		if err != nil {
			return nil, err
		}
		if err := partitionpipeline.AuthorizeMaterialization(o.partitionRegistryPath(), plan.PlanSHA256, authorization); err != nil {
			return nil, err
		}
		if _, _, _, err := partitionpipeline.Materialize(o.partitionRegistryPath(), plan.PlanSHA256, o.now()); err != nil {
			return nil, err
		}
	}
	return o.consumePlan(plan, auth.Configuration.VariantID, auth.AuthorizationID, access.RecordHash)
}
func (o *Orchestrator) materializeAndConsumeFinal(auth research.AuthorizationRecord, access research.AccessReceipt, variant string) ([]byte, error) {
	plan := o.config.Plans["FINAL_HOLDOUT"]
	state, err := partitionpipeline.PlanState(o.partitionRegistryPath(), plan.PlanSHA256)
	if err != nil {
		return nil, err
	}
	if state == partitionpipeline.PlanVerified {
		authorization, err := partitionpipeline.SealMaterializationAuthorization(partitionpipeline.MaterializationAuthorization{PlanSHA256: plan.PlanSHA256, CheckpointSHA256: plan.Checkpoint.SHA256, Partition: plan.PartitionName, RIFAuthorizationID: auth.AuthorizationID, RIFAuthorizationSHA256: auth.RecordHash, RIFAccessReceiptSHA256: access.RecordHash, AuthorizedAt: o.now().UTC()})
		if err != nil {
			return nil, err
		}
		if err := partitionpipeline.AuthorizeMaterialization(o.partitionRegistryPath(), plan.PlanSHA256, authorization); err != nil {
			return nil, err
		}
		if _, _, _, err := partitionpipeline.Materialize(o.partitionRegistryPath(), plan.PlanSHA256, o.now()); err != nil {
			return nil, err
		}
	}
	return o.consumePlan(plan, variant, auth.AuthorizationID, access.RecordHash)
}
func (o *Orchestrator) consumePlan(plan partitionpipeline.Plan, variant, authID, accessHash string) ([]byte, error) {
	state, err := partitionpipeline.PlanState(o.partitionRegistryPath(), plan.PlanSHA256)
	if err != nil {
		return nil, err
	}
	if state != partitionpipeline.MaterializationSealed && state != partitionpipeline.ConsumptionSealed {
		return nil, errors.New("partition artifact is not sealed for consumption")
	}
	artifactSHA, err := o.artifactSHA(plan)
	if err != nil {
		return nil, err
	}
	authorization, err := partitionpipeline.SealConsumptionAuthorization(partitionpipeline.ConsumptionAuthorization{PlanSHA256: plan.PlanSHA256, ArtifactSHA256: artifactSHA, Partition: plan.PartitionName, VariantID: variant, RIFAuthorizationID: authID, RIFAccessReceiptSHA256: accessHash, AuthorizedAt: o.now().UTC()})
	if err != nil {
		return nil, err
	}
	if err := partitionpipeline.AuthorizeConsumption(o.partitionRegistryPath(), plan.PlanSHA256, authorization); err != nil {
		return nil, err
	}
	data, _, err := partitionpipeline.ConsumeArtifact(o.partitionRegistryPath(), plan.PlanSHA256, o.now())
	return data, err
}

func (o *Orchestrator) validateConfig() error {
	if o.config.SchemaVersion != ConfigSchemaVersion || len(o.config.Protocol) == 0 || len(o.config.Plans) != 3 {
		return errors.New("complete orchestrator configuration is required")
	}
	if err := research.ValidateIdentity(o.config.Identity); err != nil {
		return err
	}
	engineIdentity, err := o.engineIdentity()
	if err != nil {
		return err
	}
	if _, err := qualificationrunner.ResolveVariantLedger(o.config.VariantLedger, engineIdentity.VariantLedger); err != nil {
		return err
	}
	universe, err := qualificationrunner.V00UniverseContract()
	if err != nil {
		return err
	}
	if !reflect.DeepEqual(o.config.Identity.Dataset.RequiredSymbols, universe.DatasetRequiredSymbols) || !reflect.DeepEqual(o.config.Identity.Dataset.CandidateTargetSymbols, universe.CandidateTargetSymbols) || !reflect.DeepEqual(o.config.Identity.Dataset.ContextOnlySymbols, universe.ContextOnlySymbols) || o.config.Identity.Dataset.UniverseContractSHA256 != universe.ContractSHA256 {
		return errors.New("V00 universe contract mismatch")
	}
	if o.config.Runner.SchemaVersion != research.RunnerIdentityVersion || o.config.Runner.SourceCommit != o.config.Identity.Repositories.RunnerGitCommit || o.config.Runner.BinarySHA256 != o.config.Identity.Repositories.RunnerExecutableSHA256 {
		return errors.New("runner identity mismatch")
	}
	if !o.config.Synthetic && o.config.RunnerBuild == nil {
		return errors.New("production runner build contract is missing")
	}
	for _, name := range []string{"DEVELOPMENT", "VALIDATION", "FINAL_HOLDOUT"} {
		plan, ok := o.config.Plans[name]
		if !ok || partitionpipeline.VerifyPlan(plan) != nil {
			return fmt.Errorf("%s partition plan invalid", name)
		}
		if plan.SourceIdentitySHA256 != o.config.Identity.Dataset.SourceIdentitySHA256 || plan.Checkpoint.SHA256 != o.config.Identity.Dataset.Checkpoint.SHA256 {
			return fmt.Errorf("%s dataset identity binding mismatch", name)
		}
		partition := findPartition(o.config.Identity, research.PartitionName(name))
		if partition.RequiredSymbolCoverageSHA256 != plan.PlanSHA256 || !partition.Interval.Start.Equal(plan.PartitionInterval.Start) || !partition.Interval.End.Equal(plan.PartitionInterval.End) {
			return fmt.Errorf("%s RIF/Engine partition plan binding mismatch", name)
		}
	}
	return nil
}

func (o *Orchestrator) registeredConfigurations() ([]research.RegisteredConfigurationIdentity, error) {
	values := append([]qualificationrunner.RegisteredVariant(nil), o.config.VariantLedger.Variants...)
	sort.Slice(values, func(i, j int) bool { return values[i].ID < values[j].ID })
	out := make([]research.RegisteredConfigurationIdentity, len(values))
	for i, v := range values {
		raw, err := json.Marshal(v.Configuration)
		if err != nil {
			return nil, err
		}
		out[i] = research.RegisteredConfigurationIdentity{SchemaVersion: research.RegisteredConfigVersion, VariantID: v.ID, CanonicalConfiguration: raw, ConfigurationSHA256: v.ConfigurationSHA256, CandidateFamilyID: o.config.Identity.CandidateScope.FamilyID, ProtocolID: o.config.Identity.Protocol.ID, ProtocolSHA256: o.config.Identity.Protocol.SHA256}
	}
	return out, nil
}
func (o *Orchestrator) finalBinding(snapshot research.Snapshot) (research.ExecutionBinding, error) {
	if snapshot.FrozenCandidate == nil {
		return research.ExecutionBinding{}, errors.New("frozen candidate missing")
	}
	f := snapshot.FrozenCandidate
	return research.ExecutionBinding{VariantID: f.VariantID, ConfigurationSHA256: f.ConfigurationSHA256, ProtocolSHA256: f.ProtocolSHA256, CheckpointSHA256: f.CheckpointSHA256, IndependenceSHA256: f.IndependenceSHA256, UncertaintySHA256: f.UncertaintySHA256, ConcentrationSHA256: f.ConcentrationSHA256, QualificationGateSHA256: f.QualificationGateSHA256, RunnerGitCommit: o.config.Identity.Repositories.RunnerGitCommit, RunnerExecutableSHA256: f.ExecutableSHA256, Partition: research.PartitionFinalHoldout}, nil
}
func (o *Orchestrator) engineIdentity() (qualificationrunner.ResearchIdentityV4, error) {
	var out qualificationrunner.ResearchIdentityV4
	return out, convert(o.config.Identity, &out)
}
func (o *Orchestrator) authority() (*research.Authority, research.Snapshot, error) {
	authority, err := research.OpenAuthority(o.rifPath())
	if err != nil {
		return nil, research.Snapshot{}, err
	}
	snapshot, err := authority.Snapshot()
	return authority, snapshot, err
}
func (o *Orchestrator) blockAfterAccess(cause error) error {
	authority, snapshot, err := o.authority()
	if err == nil && snapshot.State != research.StateBlocked {
		evidence := byteHash([]byte(cause.Error()))
		_, _ = authority.BlockIntegrity(snapshot.Sequence, snapshot.IntegrityHash, evidence, cause.Error())
	}
	state, stateErr := o.readState()
	if stateErr == nil {
		state.Phase = "BLOCKED"
		state.LastError = cause.Error()
		_ = o.writeState(state)
	}
	return cause
}
func (o *Orchestrator) artifactSHA(plan partitionpipeline.Plan) (string, error) {
	path := filepath.Join(o.partitionRegistryPath(), "artifacts", strings.TrimPrefix(plan.Checkpoint.SHA256, "sha256:"), plan.PartitionName, strings.TrimPrefix(plan.PlanSHA256, "sha256:")+".json")
	data, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	var artifact qualificationrunner.PartitionArtifact
	if err := strictJSON(data, &artifact); err != nil {
		return "", err
	}
	return artifact.ArtifactSHA256, nil
}
func (o *Orchestrator) requirePhase(phase string) (State, error) {
	state, err := o.readState()
	if err != nil {
		return State{}, err
	}
	if state.ConfigSHA256 != o.config.ConfigSHA256 || state.PreflightStatus != ReadyStatus || state.Phase != phase {
		return State{}, fmt.Errorf("operation requires %s, got %s", phase, state.Phase)
	}
	return state, nil
}
func (o *Orchestrator) ensureRoot() error {
	if info, err := os.Lstat(o.root); errors.Is(err, os.ErrNotExist) {
		parent := filepath.Dir(o.root)
		if stat, err := os.Stat(parent); err != nil || !stat.IsDir() {
			return errors.New("epoch parent must exist")
		}
		return os.Mkdir(o.root, 0o700)
	} else if err != nil {
		return err
	} else if info.Mode()&os.ModeSymlink != 0 || !info.IsDir() || info.Mode().Perm()&0o077 != 0 {
		return errors.New("epoch root must be isolated nonsymlink 0700 directory")
	}
	return nil
}
func (o *Orchestrator) writeState(state State) error {
	state.SchemaVersion = StateSchemaVersion
	state.ConfigSHA256 = o.config.ConfigSHA256
	state.StateSHA256 = ""
	hash, err := canonicalHash(state)
	if err != nil {
		return err
	}
	state.StateSHA256 = hash
	return atomicWrite(filepath.Join(o.root, "orchestrator.json"), mustJSONLine(state), 0o600)
}
func (o *Orchestrator) readState() (State, error) {
	data, err := os.ReadFile(filepath.Join(o.root, "orchestrator.json"))
	if err != nil {
		return State{}, err
	}
	var state State
	if err := strictJSON(data, &state); err != nil {
		return State{}, err
	}
	want := state
	want.StateSHA256 = ""
	hash, _ := canonicalHash(want)
	if state.SchemaVersion != StateSchemaVersion || state.StateSHA256 != hash {
		return State{}, errors.New("orchestrator state integrity mismatch")
	}
	if state.ConfigSHA256 != o.config.ConfigSHA256 {
		return State{}, errors.New("orchestrator state belongs to another sealed configuration")
	}
	return state, nil
}
func (o *Orchestrator) recordNotReady(cause error) {
	if o.ensureRoot() != nil {
		return
	}
	state := State{PreflightStatus: NotReadyStatus, Phase: "PREFLIGHT_FAILED", LastError: cause.Error()}
	_ = o.writeState(state)
}
func (o *Orchestrator) rifPath() string { return filepath.Join(o.root, "rif-research-governance.json") }
func (o *Orchestrator) partitionRegistryPath() string {
	return filepath.Join(o.root, "partition-registry")
}

func verifyRepository(repo RepositoryCheck) error {
	if repo.Path == "" || len(repo.Commit) != 40 {
		return errors.New("repository path and exact commit required")
	}
	head, err := git(repo.Path, "rev-parse", "HEAD")
	if err != nil || head != repo.Commit {
		return errors.New("repository commit mismatch")
	}
	status, err := git(repo.Path, "status", "--porcelain")
	if err != nil || status != "" {
		return errors.New("repository worktree is not clean")
	}
	return nil
}
func git(dir string, args ...string) (string, error) {
	cmd := exec.Command("git", append([]string{"-C", dir}, args...)...)
	out, err := cmd.Output()
	return strings.TrimSpace(string(out)), err
}
func findPartition(identity research.IdentityV4, name research.PartitionName) research.Partition {
	for _, p := range identity.Partitions {
		if p.Name == name {
			return p
		}
	}
	return research.Partition{}
}
func findEnginePartition(identity qualificationrunner.ResearchIdentityV4, name string) qualificationrunner.Partition {
	for _, p := range identity.Partitions {
		if p.Name == name {
			return p
		}
	}
	return qualificationrunner.Partition{}
}
func findEngineVariant(ledger qualificationrunner.VariantLedger, id string) qualificationrunner.RegisteredVariant {
	for _, v := range ledger.Variants {
		if v.ID == id {
			return v
		}
	}
	return qualificationrunner.RegisteredVariant{}
}
func findStageSet(snapshot research.Snapshot, stage string) *research.StageExecutionSet {
	for i := range snapshot.StageExecutionSets {
		if string(snapshot.StageExecutionSets[i].Plan.Stage) == stage {
			return &snapshot.StageExecutionSets[i]
		}
	}
	return nil
}
func stageReceipt(set *research.StageExecutionSet, variant string) *research.StageExecutionReceipt {
	for i := range set.ExecutionReceipts {
		if set.ExecutionReceipts[i].VariantID == variant {
			return &set.ExecutionReceipts[i]
		}
	}
	return nil
}
func stageAccess(set *research.StageExecutionSet, auth string) *research.StageVariantAccessReceipt {
	for i := range set.AccessReceipts {
		if set.AccessReceipts[i].AuthorizationID == auth {
			return &set.AccessReceipts[i]
		}
	}
	return nil
}
func latestStageAuthorization(set *research.StageExecutionSet, variant string) research.StageVariantAuthorization {
	for index := len(set.Authorizations) - 1; index >= 0; index-- {
		if set.Authorizations[index].Configuration.VariantID == variant {
			return set.Authorizations[index]
		}
	}
	return research.StageVariantAuthorization{}
}
func latestAuthorization(snapshot research.Snapshot, partition research.PartitionName) *research.AuthorizationRecord {
	for i := len(snapshot.Authorizations) - 1; i >= 0; i-- {
		if snapshot.Authorizations[i].Binding.Partition == partition {
			return &snapshot.Authorizations[i]
		}
	}
	return nil
}
func latestAccess(snapshot research.Snapshot, auth string) *research.AccessReceipt {
	for i := len(snapshot.AccessReceipts) - 1; i >= 0; i-- {
		if snapshot.AccessReceipts[i].AuthorizationID == auth {
			return &snapshot.AccessReceipts[i]
		}
	}
	return nil
}
func nextOperation(phase string) string {
	mapping := map[string]string{"PREFLIGHT_VERIFIED": "register-protocol", "PROTOCOL_COMMITTED": "register-research-identity", "RESEARCH_IDENTITY_REGISTERED": "reserve-holdout", "HOLDOUT_RESERVED": "authorize-development-set", "DEVELOPMENT_SET_AUTHORIZED": "run-development-set", "DEVELOPMENT_SET_SEALED": "derive-nominee", "NOMINEE_DERIVED": "authorize-validation-set", "VALIDATION_SET_AUTHORIZED": "run-validation-set", "VALIDATION_SET_SEALED": "freeze-candidate", "CANDIDATE_FROZEN": "authorize-final-holdout", "FINAL_HOLDOUT_AUTHORIZED": "run-final-holdout", "FINAL_HOLDOUT_SEALED": "closeout"}
	return mapping[phase]
}
func convert(in, out any) error {
	data, err := json.Marshal(in)
	if err != nil {
		return err
	}
	return strictJSON(data, out)
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
func canonicalHash(value any) (string, error) {
	data, err := json.Marshal(value)
	if err != nil {
		return "", err
	}
	return byteHash(data), nil
}
func byteHash(data []byte) string {
	digest := sha256.Sum256(data)
	return "sha256:" + hex.EncodeToString(digest[:])
}
func mustJSONLine(value any) []byte { data, _ := json.Marshal(value); return append(data, '\n') }
func atomicWrite(path string, data []byte, mode os.FileMode) error {
	if info, err := os.Lstat(path); err == nil && info.Mode()&os.ModeSymlink != 0 {
		return errors.New("symlink output rejected")
	}
	tmp, err := os.CreateTemp(filepath.Dir(path), ".epoch-")
	if err != nil {
		return err
	}
	name := tmp.Name()
	defer os.Remove(name)
	if err := tmp.Chmod(mode); err != nil {
		tmp.Close()
		return err
	}
	if _, err := tmp.Write(data); err != nil {
		tmp.Close()
		return err
	}
	if err := tmp.Sync(); err != nil {
		tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	return os.Rename(name, path)
}
