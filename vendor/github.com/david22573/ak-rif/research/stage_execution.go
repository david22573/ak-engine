package research

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"reflect"
	"sort"
	"strconv"
	"strings"
	"time"
)

const numericVariantOrdering = "NUMERIC_VARIANT_ID_ASCENDING"

func HashRunnerImplementationIdentity(identity RunnerImplementationIdentity) (string, error) {
	if err := validateRunnerImplementationIdentity(identity); err != nil {
		return "", err
	}
	return hashCanonical(identity)
}

func HashRegisteredConfigurationIdentity(configuration RegisteredConfigurationIdentity) (string, error) {
	canonical, err := canonicalizeRegisteredConfiguration(configuration)
	if err != nil {
		return "", err
	}
	return hashCanonical(canonical)
}

func DeterministicRunID(executionSetID, authorizationID string, configuration RegisteredConfigurationIdentity) (string, error) {
	if !validText(executionSetID) || !validText(authorizationID) {
		return "", errors.New("execution set and authorization identities are required")
	}
	configuration, err := canonicalizeRegisteredConfiguration(configuration)
	if err != nil {
		return "", err
	}
	return hashCanonical(struct {
		ExecutionSetID  string                          `json:"execution_set_id"`
		AuthorizationID string                          `json:"authorization_id"`
		Configuration   RegisteredConfigurationIdentity `json:"registered_configuration"`
	}{executionSetID, authorizationID, configuration})
}

func (a *Authority) AuthorizeDevelopmentExecutionSet(request StageExecutionSetRequest) (StageExecutionSet, error) {
	var out StageExecutionSet
	err := a.mutate(func(state *Snapshot) error {
		if err := checkExpected(state, StateHoldoutReserved, request.ExpectedSequence, request.ExpectedStateHash); err != nil {
			return err
		}
		if state.Identity == nil || state.Reservation == nil {
			return errors.New("registered identity and immutable holdout reservation are required")
		}
		plan, err := buildDevelopmentPlan(*state.Identity, state.IdentityHash, request)
		if err != nil {
			return err
		}
		out, err = issueExecutionSet(plan, state.IntegrityHash, a.nowUTC())
		if err != nil {
			return err
		}
		if len(state.StageExecutionSets) != 0 {
			return errors.New("DEVELOPMENT execution set has already been issued")
		}
		prior := state.IntegrityHash
		state.SchemaVersion = StoreSchemaVersionV2
		state.StageExecutionSets = append(state.StageExecutionSets, out)
		return appendLifecycle(state, "DEVELOPMENT_SET_AUTHORIZED", StateHoldoutReserved, StateDevelopmentSetAuthorized, out.RecordHash, prior, out.IssuedAt)
	})
	return out, err
}

func (a *Authority) AuthorizeValidationExecutionSet(expectedSequence uint64, expectedStateHash string) (StageExecutionSet, error) {
	var out StageExecutionSet
	err := a.mutate(func(state *Snapshot) error {
		if err := checkExpected(state, StateDevelopmentSetSealed, expectedSequence, expectedStateHash); err != nil {
			return err
		}
		if state.Identity == nil || state.DevelopmentNominee == nil || !state.DevelopmentNominee.Exists {
			return errors.New("sealed DEVELOPMENT nominee is required before VALIDATION authorization")
		}
		if executionSetByStage(state, PartitionValidation) != nil {
			return errors.New("VALIDATION execution set has already been issued")
		}
		development := executionSetByStage(state, PartitionDevelopment)
		if development == nil || development.CompletionState != "SEALED" {
			return errors.New("complete sealed DEVELOPMENT execution set is required")
		}
		ids, err := validationVariantIDs(*state.Identity, state.DevelopmentNominee.VariantID)
		if err != nil {
			return err
		}
		configurations := make([]RegisteredConfigurationIdentity, 0, len(ids))
		for _, id := range ids {
			configuration, ok := planConfiguration(development.Plan, id)
			if !ok {
				return errors.New("registered validation configuration is absent from sealed DEVELOPMENT plan")
			}
			configurations = append(configurations, configuration)
		}
		plan, err := buildPlan(*state.Identity, state.IdentityHash, PartitionValidation, development.Plan.Runner, configurations, true, numericVariantOrdering)
		if err != nil {
			return err
		}
		out, err = issueExecutionSet(plan, state.IntegrityHash, a.nowUTC())
		if err != nil {
			return err
		}
		prior := state.IntegrityHash
		state.StageExecutionSets = append(state.StageExecutionSets, out)
		return appendLifecycle(state, "VALIDATION_SET_AUTHORIZED", StateDevelopmentSetSealed, StateValidationSetAuthorized, out.RecordHash, prior, out.IssuedAt)
	})
	return out, err
}

func (a *Authority) ConsumeStageVariantBeforeAccess(authorization StageVariantAuthorization, access func() error) (StageVariantAccessReceipt, error) {
	if access == nil {
		return StageVariantAccessReceipt{}, errors.New("protected access callback is required")
	}
	var receipt StageVariantAccessReceipt
	err := a.mutate(func(state *Snapshot) error {
		set := executionSetByID(state, authorization.ExecutionSetID)
		if set == nil || set.CompletionState == "SEALED" {
			return errors.New("stage execution set is absent or sealed")
		}
		expectedState, err := authorizedSetState(set.Plan.Stage)
		if err != nil || state.State != expectedState {
			return errors.New("stage authorization is not active in the current lifecycle state")
		}
		persisted := stageAuthorization(set, authorization.AuthorizationID)
		if persisted == nil || persisted.RecordHash != authorization.RecordHash {
			return errors.New("variant authorization is not an exact persisted execution-set record")
		}
		for _, prior := range set.AccessReceipts {
			if prior.AuthorizationID == authorization.AuthorizationID {
				return errors.New("variant authorization has already been consumed")
			}
		}
		if stageReceiptByVariant(set, authorization.Configuration.VariantID) != nil {
			return errors.New("completed variant cannot execute again")
		}
		previous := ""
		if len(set.AccessReceipts) > 0 {
			previous = set.AccessReceipts[len(set.AccessReceipts)-1].RecordHash
		}
		receipt = StageVariantAccessReceipt{SchemaVersion: StageAccessReceiptVersion, ExecutionSetID: set.ExecutionSetID, AuthorizationID: authorization.AuthorizationID, VariantID: authorization.Configuration.VariantID, Attempt: authorization.Attempt, ConsumedAt: a.nowUTC(), PriorStateHash: state.IntegrityHash, PreviousHash: previous}
		receipt.RecordHash, err = hashStageAccessReceipt(receipt)
		if err != nil {
			return err
		}
		set.AccessReceipts = append(set.AccessReceipts, receipt)
		set.RecordHash, err = hashStageExecutionSet(*set)
		if err != nil {
			return err
		}
		return appendLifecycle(state, string(set.Plan.Stage)+"_VARIANT_ACCESS_CONSUMED", state.State, state.State, receipt.RecordHash, state.IntegrityHash, receipt.ConsumedAt)
	})
	if err != nil {
		return StageVariantAccessReceipt{}, err
	}
	return receipt, access()
}

func (a *Authority) AuthorizeZeroAccessRetry(expectedSequence uint64, expectedStateHash, executionSetID, authorizationID, accessReceiptHash, durableProofSHA256 string, rowsAccessed, outcomeArtifacts int) (StageVariantAuthorization, error) {
	var out StageVariantAuthorization
	if rowsAccessed != 0 || outcomeArtifacts != 0 || !validNonPlaceholderSHA256(durableProofSHA256) {
		return out, errors.New("retry requires durable proof of zero rows and zero outcome artifacts")
	}
	err := a.mutate(func(state *Snapshot) error {
		set := executionSetByID(state, executionSetID)
		if set == nil || set.CompletionState == "SEALED" {
			return errors.New("stage execution set is absent or sealed")
		}
		expectedState, err := authorizedSetState(set.Plan.Stage)
		if err != nil || checkExpected(state, expectedState, expectedSequence, expectedStateHash) != nil {
			return errors.New("stale or inactive stage retry request")
		}
		priorAuthorization := stageAuthorization(set, authorizationID)
		accessReceipt := stageAccessReceipt(set, accessReceiptHash)
		if priorAuthorization == nil || accessReceipt == nil || accessReceipt.AuthorizationID != authorizationID || priorAuthorization.Configuration.VariantID != accessReceipt.VariantID {
			return errors.New("retry proof does not bind an exact consumed authorization")
		}
		if stageReceiptByVariant(set, accessReceipt.VariantID) != nil {
			return errors.New("successful or result-bearing execution cannot be retried")
		}
		for _, proof := range set.RetryProofs {
			if proof.PriorAuthorizationID == authorizationID {
				return errors.New("zero-access retry proof has already been used")
			}
		}
		previousProof := ""
		if len(set.RetryProofs) > 0 {
			previousProof = set.RetryProofs[len(set.RetryProofs)-1].RecordHash
		}
		proof := ZeroAccessRetryProof{SchemaVersion: StageRetryProofVersion, ExecutionSetID: set.ExecutionSetID, PriorAuthorizationID: authorizationID, PriorAccessReceiptHash: accessReceiptHash, VariantID: accessReceipt.VariantID, RowsAccessed: 0, OutcomeArtifacts: 0, DurableProofSHA256: durableProofSHA256, ProvenAt: a.nowUTC(), PreviousHash: previousProof}
		proof.RecordHash, err = hashZeroAccessRetryProof(proof)
		if err != nil {
			return err
		}
		set.RetryProofs = append(set.RetryProofs, proof)
		previousAuthorization := set.Authorizations[len(set.Authorizations)-1].RecordHash
		out = *priorAuthorization
		out.AuthorizationID = priorAuthorization.AuthorizationID + fmt.Sprintf(":retry:%d", priorAuthorization.Attempt+1)
		out.Attempt++
		out.IssuedAt = a.nowUTC()
		out.PriorStateHash = state.IntegrityHash
		out.PreviousHash = previousAuthorization
		out.RecordHash = ""
		out.RecordHash, err = hashStageAuthorization(out)
		if err != nil {
			return err
		}
		set.Authorizations = append(set.Authorizations, out)
		set.RecordHash, err = hashStageExecutionSet(*set)
		if err != nil {
			return err
		}
		return appendLifecycle(state, string(set.Plan.Stage)+"_ZERO_ACCESS_RETRY_AUTHORIZED", state.State, state.State, proof.RecordHash, state.IntegrityHash, out.IssuedAt)
	})
	return out, err
}

func (a *Authority) RecordStageExecutionResult(expectedSequence uint64, expectedStateHash string, result ExecutionResultEnvelope) (StageExecutionReceipt, error) {
	var out StageExecutionReceipt
	err := a.mutate(func(state *Snapshot) error {
		set := executionSetByID(state, result.ExecutionSetID)
		if set == nil || set.CompletionState == "SEALED" {
			return errors.New("stage execution set is absent or sealed")
		}
		expectedState, err := authorizedSetState(set.Plan.Stage)
		if err != nil {
			return err
		}
		if err := checkExpected(state, expectedState, expectedSequence, expectedStateHash); err != nil {
			return err
		}
		authorization := stageAuthorization(set, result.AuthorizationID)
		if authorization == nil {
			return errors.New("result references an unknown variant authorization")
		}
		accessReceipt := latestAccessReceiptForAuthorization(set, authorization.AuthorizationID)
		if accessReceipt == nil || accessReceipt.RecordHash != result.AccessReceiptHash {
			return errors.New("result does not bind the exact durable access receipt")
		}
		if stageReceiptByVariant(set, authorization.Configuration.VariantID) != nil {
			return errors.New("duplicate successful or result-bearing variant execution")
		}
		if err := validateExecutionResultEnvelope(*set, *authorization, result); err != nil {
			return err
		}
		runnerHash, _ := HashRunnerImplementationIdentity(result.Runner)
		authorityEvidenceHash, _ := hashCanonical(result.AuthorityInvocations)
		previous := ""
		if len(set.ExecutionReceipts) > 0 {
			previous = set.ExecutionReceipts[len(set.ExecutionReceipts)-1].RecordHash
		}
		out = StageExecutionReceipt{SchemaVersion: StageExecutionReceiptVersion, ExecutionSetID: set.ExecutionSetID, PlanHash: set.Plan.PlanHash, AuthorizationID: result.AuthorizationID, DeterministicRunID: result.DeterministicRunID, VariantID: result.Configuration.VariantID, ConfigurationSHA256: result.Configuration.ConfigurationSHA256, RunnerIdentitySHA256: runnerHash, Partition: set.Plan.Stage, CheckpointSHA256: result.Checkpoint.SHA256, AccessReceiptHash: result.AccessReceiptHash, ResultArtifactSHA256: result.ResultArtifactSHA256, OutputManifestSHA256: result.OutputManifestSHA256, AuthorityEvidenceSHA256: authorityEvidenceHash, ResultStatus: result.ResultStatus, MandatoryGatesPassed: result.MandatoryGatesPassed, CompletedAt: a.nowUTC(), PreviousHash: previous}
		out.RecordHash, err = hashStageExecutionReceipt(out)
		if err != nil {
			return err
		}
		set.ExecutionReceipts = append(set.ExecutionReceipts, out)
		set.RecordHash, err = hashStageExecutionSet(*set)
		if err != nil {
			return err
		}
		return appendLifecycle(state, string(set.Plan.Stage)+"_VARIANT_EXECUTION_RECORDED", state.State, state.State, out.RecordHash, state.IntegrityHash, out.CompletedAt)
	})
	return out, err
}

func (a *Authority) SealStageExecutionSet(expectedSequence uint64, expectedStateHash, executionSetID string) (StageExecutionSet, error) {
	var out StageExecutionSet
	err := a.mutate(func(state *Snapshot) error {
		set := executionSetByID(state, executionSetID)
		if set == nil || set.CompletionState == "SEALED" {
			return errors.New("stage execution set is absent or already sealed")
		}
		from, err := authorizedSetState(set.Plan.Stage)
		if err != nil {
			return err
		}
		if err := checkExpected(state, from, expectedSequence, expectedStateHash); err != nil {
			return err
		}
		manifest, err := buildStageManifest(*set)
		if err != nil {
			return err
		}
		seal, err := hashCanonical(struct {
			ExecutionSetID string `json:"execution_set_id"`
			PlanHash       string `json:"plan_hash"`
			ManifestHash   string `json:"manifest_hash"`
		}{set.ExecutionSetID, set.Plan.PlanHash, manifest.ManifestHash})
		if err != nil {
			return err
		}
		now := a.nowUTC()
		set.CompletionState = "SEALED"
		set.CompletionManifest = &manifest
		set.FinalStageSeal = seal
		set.SealedAt = &now
		set.RecordHash, err = hashStageExecutionSet(*set)
		if err != nil {
			return err
		}
		to, event, err := sealedSetState(set.Plan.Stage)
		if err != nil {
			return err
		}
		out = cloneStageSet(*set)
		return appendLifecycle(state, event, from, to, seal, state.IntegrityHash, now)
	})
	return out, err
}

func (a *Authority) SelectDevelopmentNominee(expectedSequence uint64, expectedStateHash string) (NomineeSelection, error) {
	var out NomineeSelection
	err := a.mutate(func(state *Snapshot) error {
		if err := checkExpected(state, StateDevelopmentSetSealed, expectedSequence, expectedStateHash); err != nil {
			return err
		}
		if state.DevelopmentNominee != nil {
			return errors.New("DEVELOPMENT nominee has already been selected")
		}
		set := executionSetByStage(state, PartitionDevelopment)
		if set == nil || set.CompletionState != "SEALED" {
			return errors.New("complete sealed DEVELOPMENT execution set is required")
		}
		out = NomineeSelection{SchemaVersion: NomineeSelectionVersion, DevelopmentSetID: set.ExecutionSetID, Rule: "LOWEST_NUMERIC_VARIANT_ID_PASSING_ALL_MANDATORY_DEVELOPMENT_GATES", SelectedAt: a.nowUTC()}
		for _, configuration := range set.Plan.Configurations {
			receipt := stageReceiptByVariant(set, configuration.VariantID)
			if receipt != nil && receipt.MandatoryGatesPassed {
				out.Exists, out.VariantID, out.ConfigurationSHA256 = true, configuration.VariantID, configuration.ConfigurationSHA256
				break
			}
		}
		var err error
		out.RecordHash, err = hashNomineeSelection(out)
		if err != nil {
			return err
		}
		state.DevelopmentNominee = &out
		if !out.Exists {
			state.Disposition = &Disposition{State: StateRejected, Kind: "PERFORMANCE", Reason: "no registered DEVELOPMENT variant passed every mandatory gate"}
			return appendLifecycle(state, "NO_NOMINEE_REJECTED", StateDevelopmentSetSealed, StateRejected, out.RecordHash, state.IntegrityHash, out.SelectedAt)
		}
		return appendLifecycle(state, "DEVELOPMENT_NOMINEE_SELECTED", StateDevelopmentSetSealed, StateDevelopmentSetSealed, out.RecordHash, state.IntegrityHash, out.SelectedAt)
	})
	return out, err
}

func (a *Authority) RejectFailedValidationSet(expectedSequence uint64, expectedStateHash string) (Snapshot, error) {
	err := a.mutate(func(state *Snapshot) error {
		if err := checkExpected(state, StateValidationSetSealed, expectedSequence, expectedStateHash); err != nil {
			return err
		}
		set := executionSetByStage(state, PartitionValidation)
		if set == nil || set.CompletionState != "SEALED" {
			return errors.New("sealed VALIDATION execution set is required")
		}
		for _, receipt := range set.ExecutionReceipts {
			if !receipt.MandatoryGatesPassed {
				state.Disposition = &Disposition{State: StateRejected, Kind: "PERFORMANCE", Reason: "nominee or required stability neighbor failed VALIDATION"}
				return appendLifecycle(state, "VALIDATION_SET_REJECTED", StateValidationSetSealed, StateRejected, set.FinalStageSeal, state.IntegrityHash, a.nowUTC())
			}
		}
		return errors.New("VALIDATION set passed; performance rejection is not applicable")
	})
	if err != nil {
		return Snapshot{}, err
	}
	return a.Snapshot()
}

func (a *Authority) ExportStageExecutionEnvelope(executionSetID, authorizationID string) (StageExecutionEnvelope, error) {
	state, err := a.Snapshot()
	if err != nil {
		return StageExecutionEnvelope{}, err
	}
	set := executionSetByID(&state, executionSetID)
	if set == nil {
		return StageExecutionEnvelope{}, errors.New("stage execution set was not found")
	}
	envelope := StageExecutionEnvelope{SchemaVersion: StageEnvelopeVersion, Snapshot: state, ExecutionSet: cloneStageSet(*set)}
	if authorizationID != "" {
		authorization := stageAuthorization(set, authorizationID)
		if authorization == nil {
			return StageExecutionEnvelope{}, errors.New("stage variant authorization was not found")
		}
		copyAuthorization := *authorization
		envelope.Authorization = &copyAuthorization
	}
	envelope.EnvelopeHash, err = hashStageEnvelope(envelope)
	return envelope, err
}

func VerifyStageExecutionEnvelope(envelope StageExecutionEnvelope) error {
	if envelope.SchemaVersion != StageEnvelopeVersion || envelope.Snapshot.SchemaVersion != StoreSchemaVersionV2 {
		return errors.New("unsupported stage execution envelope schema")
	}
	if err := validateSnapshot(envelope.Snapshot); err != nil {
		return err
	}
	set := executionSetByID(&envelope.Snapshot, envelope.ExecutionSet.ExecutionSetID)
	if set == nil || set.RecordHash != envelope.ExecutionSet.RecordHash {
		return errors.New("stage execution set is not an exact persisted record")
	}
	if envelope.Snapshot.Identity == nil || validateStageExecutionSet(*envelope.Snapshot.Identity, envelope.Snapshot.IdentityHash, envelope.ExecutionSet) != nil {
		return errors.New("stage execution envelope contains an invalid execution set")
	}
	if envelope.Authorization != nil {
		authorization := stageAuthorization(set, envelope.Authorization.AuthorizationID)
		if authorization == nil || authorization.RecordHash != envelope.Authorization.RecordHash {
			return errors.New("stage authorization is not an exact persisted record")
		}
		wantAuthorization, err := hashStageAuthorization(*envelope.Authorization)
		if err != nil || envelope.Authorization.RecordHash != wantAuthorization {
			return errors.New("stage execution envelope authorization hash mismatch")
		}
	}
	want, err := hashStageEnvelope(envelope)
	if err != nil || envelope.EnvelopeHash != want {
		return errors.New("stage execution envelope hash mismatch")
	}
	return nil
}

func DecodeStageExecutionEnvelopeJSON(data []byte) (StageExecutionEnvelope, error) {
	var envelope StageExecutionEnvelope
	if err := strictDecode(data, &envelope); err != nil {
		return StageExecutionEnvelope{}, err
	}
	if err := VerifyStageExecutionEnvelope(envelope); err != nil {
		return StageExecutionEnvelope{}, err
	}
	return envelope, nil
}

func EncodeStageExecutionEnvelopeJSON(envelope StageExecutionEnvelope) ([]byte, error) {
	if err := VerifyStageExecutionEnvelope(envelope); err != nil {
		return nil, err
	}
	data, err := json.MarshalIndent(envelope, "", "  ")
	if err != nil {
		return nil, err
	}
	return append(data, '\n'), nil
}

func DecodeExecutionResultEnvelopeJSON(data []byte) (ExecutionResultEnvelope, error) {
	var result ExecutionResultEnvelope
	if err := strictDecode(data, &result); err != nil {
		return ExecutionResultEnvelope{}, err
	}
	canonicalArtifact, err := canonicalJSON(result.ResultArtifact)
	if err != nil {
		return ExecutionResultEnvelope{}, err
	}
	result.ResultArtifact = canonicalArtifact
	return result, nil
}

func EncodeExecutionResultEnvelopeJSON(result ExecutionResultEnvelope) ([]byte, error) {
	want, err := hashExecutionResultEnvelope(result)
	if err != nil || result.EnvelopeHash != want {
		return nil, errors.New("execution result envelope hash mismatch")
	}
	data, err := json.MarshalIndent(result, "", "  ")
	if err != nil {
		return nil, err
	}
	return append(data, '\n'), nil
}

func validateStageExecutionState(state Snapshot) error {
	if state.SchemaVersion == StoreSchemaVersion {
		if len(state.StageExecutionSets) != 0 || state.DevelopmentNominee != nil {
			return errors.New("V1 research governance store cannot contain execution-set state")
		}
		return nil
	}
	if state.SchemaVersion != StoreSchemaVersionV2 {
		return errors.New("unsupported research governance store schema")
	}
	if state.Identity == nil || len(state.StageExecutionSets) == 0 || len(state.StageExecutionSets) > 2 {
		return errors.New("V2 research governance store requires one DEVELOPMENT and at most one VALIDATION execution set")
	}
	seenStages := map[PartitionName]struct{}{}
	for i := range state.StageExecutionSets {
		set := &state.StageExecutionSets[i]
		if _, duplicate := seenStages[set.Plan.Stage]; duplicate {
			return errors.New("duplicate stage execution set")
		}
		seenStages[set.Plan.Stage] = struct{}{}
		if err := validateStageExecutionSet(*state.Identity, state.IdentityHash, *set); err != nil {
			return err
		}
	}
	if _, ok := seenStages[PartitionDevelopment]; !ok {
		return errors.New("V2 research governance store is missing DEVELOPMENT execution set")
	}
	if state.DevelopmentNominee != nil {
		want, err := hashNomineeSelection(*state.DevelopmentNominee)
		development := executionSetByStage(&state, PartitionDevelopment)
		if err != nil || state.DevelopmentNominee.RecordHash != want || development == nil || development.CompletionState != "SEALED" || state.DevelopmentNominee.DevelopmentSetID != development.ExecutionSetID {
			return errors.New("DEVELOPMENT nominee selection integrity mismatch")
		}
		if state.DevelopmentNominee.Exists {
			configuration, ok := planConfiguration(development.Plan, state.DevelopmentNominee.VariantID)
			receipt := stageReceiptByVariant(development, state.DevelopmentNominee.VariantID)
			if !ok || receipt == nil || !receipt.MandatoryGatesPassed || configuration.ConfigurationSHA256 != state.DevelopmentNominee.ConfigurationSHA256 {
				return errors.New("DEVELOPMENT nominee is not the exact lowest passing registered result")
			}
			for _, prior := range development.Plan.Configurations {
				if prior.VariantID == configuration.VariantID {
					break
				}
				priorReceipt := stageReceiptByVariant(development, prior.VariantID)
				if priorReceipt != nil && priorReceipt.MandatoryGatesPassed {
					return errors.New("DEVELOPMENT nominee is not the lowest numeric passing variant")
				}
			}
		}
	}
	return nil
}

func validateStageExecutionSet(identity IdentityV4, identityHash string, set StageExecutionSet) error {
	if set.SchemaVersion != StageExecutionSetVersion || set.ExecutionSetID == "" || set.Plan.ResearchIdentityHash != identityHash || set.IssuanceState != "ISSUED" || (set.CompletionState != "OPEN" && set.CompletionState != "SEALED") || set.IssuedAt.IsZero() {
		return errors.New("stage execution set is incomplete")
	}
	wantPlan, err := hashStageExecutionPlan(set.Plan)
	if err != nil || set.Plan.SchemaVersion != StageExecutionPlanVersion || set.Plan.PlanHash != wantPlan || set.Plan.ExpectedExecutions != len(set.Plan.Configurations) || !set.Plan.Complete || set.Plan.OrderingRule != numericVariantOrdering {
		return errors.New("stage execution plan identity mismatch")
	}
	rebuilt, err := buildPlan(identity, identityHash, set.Plan.Stage, set.Plan.Runner, set.Plan.Configurations, set.Plan.Complete, set.Plan.OrderingRule)
	if err != nil || rebuilt.PlanHash != set.Plan.PlanHash {
		return fmt.Errorf("stage execution plan does not match registered identity: rebuild error=%v rebuilt_hash=%s stored_hash=%s", err, rebuilt.PlanHash, set.Plan.PlanHash)
	}
	if len(set.Authorizations) < set.Plan.ExpectedExecutions {
		return errors.New("stage execution set is missing per-variant authorizations")
	}
	previous := ""
	for i, authorization := range set.Authorizations {
		if authorization.SchemaVersion != StageAuthorizationVersion || authorization.ExecutionSetID != set.ExecutionSetID || authorization.PlanHash != set.Plan.PlanHash || authorization.PreviousHash != previous || authorization.Attempt <= 0 {
			return errors.New("stage variant authorization chain is invalid")
		}
		want, err := hashStageAuthorization(authorization)
		if err != nil || authorization.RecordHash != want {
			return errors.New("stage variant authorization hash mismatch")
		}
		if authorization.Attempt == 1 {
			if i >= len(set.Plan.Configurations) || authorization.Ordinal != i || !reflect.DeepEqual(authorization.Configuration, set.Plan.Configurations[i]) {
				return errors.New("initial stage authorizations are missing, duplicated, or reordered")
			}
		} else if authorization.Ordinal < 0 || authorization.Ordinal >= len(set.Plan.Configurations) || !reflect.DeepEqual(authorization.Configuration, set.Plan.Configurations[authorization.Ordinal]) {
			return errors.New("retry authorization configuration substitution")
		}
		if !reflect.DeepEqual(authorization.Runner, set.Plan.Runner) || !reflect.DeepEqual(authorization.Partition, set.Plan.Partition) || !reflect.DeepEqual(authorization.Protocol, set.Plan.Protocol) || !reflect.DeepEqual(authorization.Checkpoint, set.Plan.Checkpoint) || !reflect.DeepEqual(authorization.Authorities, set.Plan.Authorities) || !reflect.DeepEqual(authorization.GateSet, set.Plan.GateSet) {
			return errors.New("stage authorization pre-execution identity substitution")
		}
		previous = authorization.RecordHash
	}
	previous = ""
	for _, receipt := range set.AccessReceipts {
		if receipt.SchemaVersion != StageAccessReceiptVersion || receipt.ExecutionSetID != set.ExecutionSetID || receipt.PreviousHash != previous || stageAuthorization(&set, receipt.AuthorizationID) == nil {
			return errors.New("stage access receipt chain is invalid")
		}
		want, err := hashStageAccessReceipt(receipt)
		if err != nil || receipt.RecordHash != want {
			return errors.New("stage access receipt hash mismatch")
		}
		previous = receipt.RecordHash
	}
	previous = ""
	for _, proof := range set.RetryProofs {
		if proof.SchemaVersion != StageRetryProofVersion || proof.ExecutionSetID != set.ExecutionSetID || proof.RowsAccessed != 0 || proof.OutcomeArtifacts != 0 || !validNonPlaceholderSHA256(proof.DurableProofSHA256) || proof.PreviousHash != previous || stageAccessReceipt(&set, proof.PriorAccessReceiptHash) == nil {
			return errors.New("zero-access retry proof chain is invalid")
		}
		want, err := hashZeroAccessRetryProof(proof)
		if err != nil || proof.RecordHash != want {
			return errors.New("zero-access retry proof hash mismatch")
		}
		previous = proof.RecordHash
	}
	previous = ""
	seenResults := map[string]struct{}{}
	for _, receipt := range set.ExecutionReceipts {
		if receipt.SchemaVersion != StageExecutionReceiptVersion || receipt.ExecutionSetID != set.ExecutionSetID || receipt.PlanHash != set.Plan.PlanHash || receipt.Partition != set.Plan.Stage || receipt.PreviousHash != previous || !validNonPlaceholderSHA256(receipt.ResultArtifactSHA256) || !validNonPlaceholderSHA256(receipt.OutputManifestSHA256) || !validNonPlaceholderSHA256(receipt.AuthorityEvidenceSHA256) {
			return errors.New("stage execution receipt chain is invalid")
		}
		if _, duplicate := seenResults[receipt.VariantID]; duplicate {
			return errors.New("duplicate result-bearing execution receipt")
		}
		seenResults[receipt.VariantID] = struct{}{}
		configuration, ok := planConfiguration(set.Plan, receipt.VariantID)
		if !ok || configuration.ConfigurationSHA256 != receipt.ConfigurationSHA256 || stageAccessReceipt(&set, receipt.AccessReceiptHash) == nil {
			return errors.New("stage execution receipt configuration or access mismatch")
		}
		want, err := hashStageExecutionReceipt(receipt)
		if err != nil || receipt.RecordHash != want {
			return errors.New("stage execution receipt hash mismatch")
		}
		previous = receipt.RecordHash
	}
	if set.CompletionState == "OPEN" {
		if set.CompletionManifest != nil || set.FinalStageSeal != "" || set.SealedAt != nil {
			return errors.New("open execution set contains sealing fields")
		}
	} else {
		if set.CompletionManifest == nil || set.SealedAt == nil || !validNonPlaceholderSHA256(set.FinalStageSeal) {
			return errors.New("sealed execution set is incomplete")
		}
		manifest, err := buildStageManifest(set)
		if err != nil || !reflect.DeepEqual(manifest, *set.CompletionManifest) {
			return errors.New("stage completion manifest mismatch")
		}
		wantSeal, _ := hashCanonical(struct {
			ExecutionSetID string `json:"execution_set_id"`
			PlanHash       string `json:"plan_hash"`
			ManifestHash   string `json:"manifest_hash"`
		}{set.ExecutionSetID, set.Plan.PlanHash, manifest.ManifestHash})
		if set.FinalStageSeal != wantSeal {
			return errors.New("final stage seal mismatch")
		}
	}
	want, err := hashStageExecutionSet(set)
	if err != nil || set.RecordHash != want {
		return errors.New("stage execution set record hash mismatch")
	}
	return nil
}

func buildDevelopmentPlan(identity IdentityV4, identityHash string, request StageExecutionSetRequest) (StageExecutionPlan, error) {
	if request.SchemaVersion != StageExecutionSetVersion || !request.Complete || request.OrderingRule != numericVariantOrdering {
		return StageExecutionPlan{}, errors.New("DEVELOPMENT requires a complete deterministically ordered execution set")
	}
	if len(request.Configurations) != len(identity.VariantLedger.Variants) {
		return StageExecutionPlan{}, errors.New("DEVELOPMENT execution set must contain the complete registered ledger")
	}
	return buildPlan(identity, identityHash, PartitionDevelopment, request.Runner, request.Configurations, request.Complete, request.OrderingRule)
}

func buildPlan(identity IdentityV4, identityHash string, stage PartitionName, runner RunnerImplementationIdentity, configurations []RegisteredConfigurationIdentity, complete bool, orderingRule string) (StageExecutionPlan, error) {
	if stage != PartitionDevelopment && stage != PartitionValidation {
		return StageExecutionPlan{}, errors.New("execution sets are restricted to DEVELOPMENT and VALIDATION")
	}
	if !complete || orderingRule != numericVariantOrdering || len(configurations) == 0 {
		return StageExecutionPlan{}, errors.New("stage execution plan must be complete and deterministically ordered")
	}
	if err := validateRunnerAgainstIdentity(identity, runner); err != nil {
		return StageExecutionPlan{}, err
	}
	canonicalConfigurations := make([]RegisteredConfigurationIdentity, len(configurations))
	seen := map[string]struct{}{}
	for i, configuration := range configurations {
		canonical, err := canonicalizeRegisteredConfiguration(configuration)
		if err != nil {
			return StageExecutionPlan{}, err
		}
		registered, ok := variant(identity, canonical.VariantID)
		if !ok || registered.ConfigurationSHA256 != canonical.ConfigurationSHA256 || canonical.CandidateFamilyID != identity.CandidateScope.FamilyID || canonical.ProtocolID != identity.Protocol.ID || canonical.ProtocolSHA256 != identity.Protocol.SHA256 {
			return StageExecutionPlan{}, errors.New("execution plan contains an unregistered or substituted configuration")
		}
		if _, duplicate := seen[canonical.VariantID]; duplicate {
			return StageExecutionPlan{}, errors.New("execution plan contains a duplicate variant")
		}
		seen[canonical.VariantID] = struct{}{}
		canonicalConfigurations[i] = canonical
	}
	if !numericVariantOrder(canonicalConfigurations) {
		return StageExecutionPlan{}, errors.New("execution plan variants are missing, reordered, or not in numeric variant order")
	}
	if stage == PartitionDevelopment {
		if len(seen) != len(identity.VariantLedger.Variants) {
			return StageExecutionPlan{}, errors.New("DEVELOPMENT plan is not the complete registered ledger")
		}
		for _, registered := range identity.VariantLedger.Variants {
			if _, ok := seen[registered.ID]; !ok {
				return StageExecutionPlan{}, errors.New("DEVELOPMENT plan omits a registered variant")
			}
		}
	}
	datasetHash, err := hashCanonical(identity.Dataset)
	if err != nil {
		return StageExecutionPlan{}, err
	}
	plan := StageExecutionPlan{SchemaVersion: StageExecutionPlanVersion, ResearchIdentityHash: identityHash, Protocol: identity.Protocol, Stage: stage, Partition: partition(identity, stage), Checkpoint: identity.Dataset.Checkpoint, DatasetIdentitySHA256: datasetHash, Runner: runner, Configurations: canonicalConfigurations, DeterministicSeedPolicy: identity.Authorities.DeterministicSeedPolicy, ExpectedExecutions: len(canonicalConfigurations), Complete: true, OrderingRule: numericVariantOrdering, Authorities: identity.Authorities, GateSet: identity.Authorities.QualificationGateSet}
	plan.PlanHash, err = hashStageExecutionPlan(plan)
	return plan, err
}

func issueExecutionSet(plan StageExecutionPlan, priorStateHash string, at time.Time) (StageExecutionSet, error) {
	setIDHash, err := hashCanonical(struct {
		PlanHash string        `json:"plan_hash"`
		Stage    PartitionName `json:"stage"`
	}{plan.PlanHash, plan.Stage})
	if err != nil {
		return StageExecutionSet{}, err
	}
	set := StageExecutionSet{SchemaVersion: StageExecutionSetVersion, ExecutionSetID: "execution-set:" + strings.TrimPrefix(setIDHash, "sha256:"), Plan: plan, IssuanceState: "ISSUED", CompletionState: "OPEN", Authorizations: []StageVariantAuthorization{}, AccessReceipts: []StageVariantAccessReceipt{}, RetryProofs: []ZeroAccessRetryProof{}, ExecutionReceipts: []StageExecutionReceipt{}, IssuedAt: at}
	previous := ""
	for ordinal, configuration := range plan.Configurations {
		authorizationIDHash, err := hashCanonical(struct {
			ExecutionSetID string `json:"execution_set_id"`
			VariantID      string `json:"variant_id"`
			Ordinal        int    `json:"ordinal"`
		}{set.ExecutionSetID, configuration.VariantID, ordinal})
		if err != nil {
			return StageExecutionSet{}, err
		}
		authorization := StageVariantAuthorization{SchemaVersion: StageAuthorizationVersion, AuthorizationID: "stage-authorization:" + strings.TrimPrefix(authorizationIDHash, "sha256:"), ExecutionSetID: set.ExecutionSetID, PlanHash: plan.PlanHash, Ordinal: ordinal, Attempt: 1, Configuration: configuration, Runner: plan.Runner, Partition: plan.Partition, Protocol: plan.Protocol, Checkpoint: plan.Checkpoint, Authorities: plan.Authorities, GateSet: plan.GateSet, IssuedAt: at, PriorStateHash: priorStateHash, PreviousHash: previous}
		authorization.RecordHash, err = hashStageAuthorization(authorization)
		if err != nil {
			return StageExecutionSet{}, err
		}
		set.Authorizations = append(set.Authorizations, authorization)
		previous = authorization.RecordHash
	}
	set.RecordHash, err = hashStageExecutionSet(set)
	return set, err
}

func validateExecutionResultEnvelope(set StageExecutionSet, authorization StageVariantAuthorization, result ExecutionResultEnvelope) error {
	if result.SchemaVersion != ExecutionResultVersion || result.ExecutionSetID != set.ExecutionSetID || result.PlanHash != set.Plan.PlanHash || result.AuthorizationID != authorization.AuthorizationID || result.ResultStatus != "COMPLETED" {
		return errors.New("execution result envelope has an invalid identity or status")
	}
	runID, err := DeterministicRunID(set.ExecutionSetID, authorization.AuthorizationID, authorization.Configuration)
	if err != nil || result.DeterministicRunID != runID {
		return errors.New("execution result deterministic run identity mismatch")
	}
	if !canonicalEqual(result.Configuration, authorization.Configuration) || !canonicalEqual(result.Runner, authorization.Runner) || !canonicalEqual(result.Partition, authorization.Partition) || !canonicalEqual(result.Protocol, authorization.Protocol) || !canonicalEqual(result.Checkpoint, authorization.Checkpoint) || !canonicalEqual(result.Authorities, authorization.Authorities) || !canonicalEqual(result.GateSet, authorization.GateSet) {
		return errors.New("execution result runner, configuration, partition, protocol, checkpoint, authority, or gate substitution")
	}
	canonicalArtifact, err := canonicalJSON(result.ResultArtifact)
	if err != nil || !bytes.Equal(canonicalArtifact, result.ResultArtifact) {
		return errors.New("result artifact must be nonempty canonical JSON")
	}
	artifactHash, err := hashBytes(canonicalArtifact)
	if err != nil || !validNonPlaceholderSHA256(result.ResultArtifactSHA256) || result.ResultArtifactSHA256 != artifactHash {
		return errors.New("actual result artifact hash mismatch or placeholder")
	}
	if !validNonPlaceholderSHA256(result.OutputManifestSHA256) || !validNonPlaceholderSHA256(result.AccessReceiptHash) {
		return errors.New("result output manifest and access receipt hashes are mandatory")
	}
	if err := validateAuthorityInvocations(result.AuthorityInvocations, set.Plan.Authorities); err != nil {
		return err
	}
	want, err := hashExecutionResultEnvelope(result)
	if err != nil || !validNonPlaceholderSHA256(result.EnvelopeHash) || result.EnvelopeHash != want {
		return errors.New("execution result envelope hash mismatch")
	}
	return nil
}

func SealExecutionResultEnvelope(result *ExecutionResultEnvelope) error {
	if result == nil {
		return errors.New("execution result envelope is required")
	}
	canonical, err := canonicalJSON(result.ResultArtifact)
	if err != nil {
		return err
	}
	result.ResultArtifact = canonical
	result.ResultArtifactSHA256, _ = hashBytes(canonical)
	result.EnvelopeHash, err = hashExecutionResultEnvelope(*result)
	return err
}

func buildStageManifest(set StageExecutionSet) (StageCompletionManifest, error) {
	if set.Plan.ExpectedExecutions != len(set.Plan.Configurations) || len(set.ExecutionReceipts) != set.Plan.ExpectedExecutions {
		return StageCompletionManifest{}, errors.New("incomplete stage execution set cannot seal")
	}
	manifest := StageCompletionManifest{SchemaVersion: StageManifestVersion, ExecutionSetID: set.ExecutionSetID, PlanHash: set.Plan.PlanHash, Stage: set.Plan.Stage}
	usedReceipts := map[string]struct{}{}
	for _, configuration := range set.Plan.Configurations {
		receipt := stageReceiptByVariant(&set, configuration.VariantID)
		if receipt == nil || receipt.ConfigurationSHA256 != configuration.ConfigurationSHA256 {
			return StageCompletionManifest{}, errors.New("stage result is missing or configuration-mutated")
		}
		if _, duplicate := usedReceipts[receipt.RecordHash]; duplicate {
			return StageCompletionManifest{}, errors.New("duplicate successful execution receipt prevents sealing")
		}
		usedReceipts[receipt.RecordHash] = struct{}{}
		manifest.OrderedVariantIDs = append(manifest.OrderedVariantIDs, configuration.VariantID)
		manifest.OrderedReceiptHashes = append(manifest.OrderedReceiptHashes, receipt.RecordHash)
		manifest.OrderedResultHashes = append(manifest.OrderedResultHashes, receipt.ResultArtifactSHA256)
	}
	for _, access := range set.AccessReceipts {
		accounted := false
		for _, receipt := range set.ExecutionReceipts {
			if receipt.AccessReceiptHash == access.RecordHash {
				accounted = true
				break
			}
		}
		if !accounted {
			for _, proof := range set.RetryProofs {
				if proof.PriorAccessReceiptHash == access.RecordHash && proof.RowsAccessed == 0 && proof.OutcomeArtifacts == 0 {
					accounted = true
					break
				}
			}
		}
		if !accounted {
			return StageCompletionManifest{}, errors.New("unresolved accessed attempt prevents stage sealing")
		}
	}
	var err error
	manifest.ManifestHash, err = hashStageManifest(manifest)
	return manifest, err
}

func validationVariantIDs(identity IdentityV4, nominee string) ([]string, error) {
	registered, ok := variant(identity, nominee)
	if !ok || registered.ID == "" {
		return nil, errors.New("DEVELOPMENT nominee is not registered")
	}
	ids := []string{nominee}
	found := false
	for _, neighborhood := range identity.VariantLedger.StabilityNeighborhoods {
		if neighborhood.VariantID == nominee {
			found = true
			ids = append(ids, neighborhood.NeighborIDs...)
			break
		}
	}
	if !found || len(ids) < 2 {
		return nil, errors.New("nominee has no pre-registered mandatory stability neighborhood")
	}
	seen := map[string]struct{}{}
	for _, id := range ids {
		if _, duplicate := seen[id]; duplicate {
			return nil, errors.New("validation nominee and stability neighbors must be unique")
		}
		seen[id] = struct{}{}
	}
	sort.Slice(ids, func(i, j int) bool { return variantLess(ids[i], ids[j]) })
	return ids, nil
}

func validateRunnerAgainstIdentity(identity IdentityV4, runner RunnerImplementationIdentity) error {
	if err := validateRunnerImplementationIdentity(runner); err != nil {
		return err
	}
	if runner.SourceCommit != identity.Repositories.RunnerGitCommit || runner.BinarySHA256 != identity.Repositories.RunnerExecutableSHA256 {
		return errors.New("runner source commit or binary does not match registered pre-execution identity")
	}
	return nil
}

func validateRunnerImplementationIdentity(runner RunnerImplementationIdentity) error {
	if runner.SchemaVersion != RunnerIdentityVersion || !validHex(runner.SourceCommit, 40) || !validHashIdentity(runner.PackageIdentity) || !validSHA256(runner.BuildInputsSHA256) || !validText(runner.CompilerIdentity) || !validHashIdentity(runner.BuildModeIdentity) || !validSHA256(runner.BinarySHA256) {
		return errors.New("complete deterministic pre-execution runner identity is required")
	}
	return nil
}

func canonicalizeRegisteredConfiguration(configuration RegisteredConfigurationIdentity) (RegisteredConfigurationIdentity, error) {
	if configuration.SchemaVersion != RegisteredConfigVersion || !validText(configuration.VariantID) || !validSHA256(configuration.ConfigurationSHA256) || !validText(configuration.CandidateFamilyID) || !validText(configuration.ProtocolID) || !validSHA256(configuration.ProtocolSHA256) {
		return RegisteredConfigurationIdentity{}, errors.New("registered configuration identity is incomplete")
	}
	var compact bytes.Buffer
	if err := json.Compact(&compact, configuration.CanonicalConfiguration); err != nil || compact.Len() == 0 {
		return RegisteredConfigurationIdentity{}, errors.New("canonical configuration must be valid nonempty JSON")
	}
	hash, _ := hashBytes(compact.Bytes())
	if hash != configuration.ConfigurationSHA256 {
		return RegisteredConfigurationIdentity{}, errors.New("canonical configuration hash mismatch")
	}
	configuration.CanonicalConfiguration = append(json.RawMessage(nil), compact.Bytes()...)
	return configuration, nil
}

func validateAuthorityInvocations(invocations []AuthorityInvocationEvidence, authorities AuthorityIdentity) error {
	expected := []HashIdentity{authorities.Independence, authorities.Uncertainty, {ID: AcceptedConcentrationID, SHA256: authorities.ConcentrationSHA256}}
	sort.Slice(expected, func(i, j int) bool { return expected[i].ID < expected[j].ID })
	if len(invocations) != len(expected) {
		return errors.New("complete actual authority invocation evidence is required")
	}
	for i, invocation := range invocations {
		if !reflect.DeepEqual(invocation.Identity, expected[i]) || !invocation.Invoked || !validNonPlaceholderSHA256(invocation.EvidenceSHA256) {
			return errors.New("authority invocation identity or evidence mismatch")
		}
	}
	return nil
}

func numericVariantOrder(configurations []RegisteredConfigurationIdentity) bool {
	for i, configuration := range configurations {
		if _, ok := variantNumber(configuration.VariantID); !ok {
			return false
		}
		if i > 0 && !variantLess(configurations[i-1].VariantID, configuration.VariantID) {
			return false
		}
	}
	return true
}

func variantNumber(id string) (int, bool) {
	if len(id) < 2 || id[0] != 'V' {
		return 0, false
	}
	number, err := strconv.Atoi(id[1:])
	return number, err == nil && number >= 0
}

func variantLess(left, right string) bool {
	l, lok := variantNumber(left)
	r, rok := variantNumber(right)
	if lok && rok && l != r {
		return l < r
	}
	return left < right
}

func executionSetByID(state *Snapshot, id string) *StageExecutionSet {
	for i := range state.StageExecutionSets {
		if state.StageExecutionSets[i].ExecutionSetID == id {
			return &state.StageExecutionSets[i]
		}
	}
	return nil
}

func executionSetByStage(state *Snapshot, stage PartitionName) *StageExecutionSet {
	for i := range state.StageExecutionSets {
		if state.StageExecutionSets[i].Plan.Stage == stage {
			return &state.StageExecutionSets[i]
		}
	}
	return nil
}

func stageAuthorization(set *StageExecutionSet, id string) *StageVariantAuthorization {
	for i := range set.Authorizations {
		if set.Authorizations[i].AuthorizationID == id {
			return &set.Authorizations[i]
		}
	}
	return nil
}

func stageAccessReceipt(set *StageExecutionSet, hash string) *StageVariantAccessReceipt {
	for i := range set.AccessReceipts {
		if set.AccessReceipts[i].RecordHash == hash {
			return &set.AccessReceipts[i]
		}
	}
	return nil
}

func latestAccessReceiptForAuthorization(set *StageExecutionSet, id string) *StageVariantAccessReceipt {
	for i := len(set.AccessReceipts) - 1; i >= 0; i-- {
		if set.AccessReceipts[i].AuthorizationID == id {
			return &set.AccessReceipts[i]
		}
	}
	return nil
}

func stageReceiptByVariant(set *StageExecutionSet, variantID string) *StageExecutionReceipt {
	for i := range set.ExecutionReceipts {
		if set.ExecutionReceipts[i].VariantID == variantID {
			return &set.ExecutionReceipts[i]
		}
	}
	return nil
}

func planConfiguration(plan StageExecutionPlan, variantID string) (RegisteredConfigurationIdentity, bool) {
	for _, configuration := range plan.Configurations {
		if configuration.VariantID == variantID {
			return configuration, true
		}
	}
	return RegisteredConfigurationIdentity{}, false
}

func authorizedSetState(stage PartitionName) (LifecycleState, error) {
	switch stage {
	case PartitionDevelopment:
		return StateDevelopmentSetAuthorized, nil
	case PartitionValidation:
		return StateValidationSetAuthorized, nil
	default:
		return "", errors.New("FINAL_HOLDOUT does not support execution sets")
	}
}

func sealedSetState(stage PartitionName) (LifecycleState, string, error) {
	switch stage {
	case PartitionDevelopment:
		return StateDevelopmentSetSealed, "DEVELOPMENT_SET_SEALED", nil
	case PartitionValidation:
		return StateValidationSetSealed, "VALIDATION_SET_SEALED", nil
	default:
		return "", "", errors.New("FINAL_HOLDOUT does not support execution sets")
	}
}

func canonicalJSON(raw json.RawMessage) ([]byte, error) {
	if len(raw) == 0 {
		return nil, errors.New("empty JSON")
	}
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

func hashBytes(data []byte) (string, error) { return hashCanonical(json.RawMessage(data)) }

func validNonPlaceholderSHA256(value string) bool {
	return validSHA256(value) && value != "sha256:"+strings.Repeat("0", 64)
}

func canonicalEqual(left, right any) bool {
	leftHash, leftErr := hashCanonical(left)
	rightHash, rightErr := hashCanonical(right)
	return leftErr == nil && rightErr == nil && leftHash == rightHash
}

func hashStageExecutionPlan(plan StageExecutionPlan) (string, error) {
	plan.PlanHash = ""
	return hashCanonical(plan)
}
func hashStageAuthorization(value StageVariantAuthorization) (string, error) {
	value.RecordHash = ""
	return hashCanonical(value)
}
func hashStageAccessReceipt(value StageVariantAccessReceipt) (string, error) {
	value.RecordHash = ""
	return hashCanonical(value)
}
func hashZeroAccessRetryProof(value ZeroAccessRetryProof) (string, error) {
	value.RecordHash = ""
	return hashCanonical(value)
}
func hashExecutionResultEnvelope(value ExecutionResultEnvelope) (string, error) {
	value.EnvelopeHash = ""
	return hashCanonical(value)
}
func hashStageExecutionReceipt(value StageExecutionReceipt) (string, error) {
	value.RecordHash = ""
	return hashCanonical(value)
}
func hashStageManifest(value StageCompletionManifest) (string, error) {
	value.ManifestHash = ""
	return hashCanonical(value)
}
func hashStageExecutionSet(value StageExecutionSet) (string, error) {
	value.RecordHash = ""
	return hashCanonical(value)
}
func hashNomineeSelection(value NomineeSelection) (string, error) {
	value.RecordHash = ""
	return hashCanonical(value)
}
func hashStageEnvelope(value StageExecutionEnvelope) (string, error) {
	value.EnvelopeHash = ""
	return hashCanonical(value)
}

func cloneStageSet(value StageExecutionSet) StageExecutionSet {
	data, _ := json.Marshal(value)
	var out StageExecutionSet
	_ = json.Unmarshal(data, &out)
	return out
}
