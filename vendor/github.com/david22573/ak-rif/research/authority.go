package research

import (
	"errors"
	"fmt"
	"reflect"
	"time"
)

func (a *Authority) RegisterIdentity(identity IdentityV4) (Snapshot, error) {
	canonical, err := CanonicalizeIdentity(identity)
	if err != nil {
		return Snapshot{}, err
	}
	identityHash, err := HashIdentityV4(canonical)
	if err != nil {
		return Snapshot{}, err
	}
	err = a.mutate(func(state *Snapshot) error {
		if state.Identity != nil {
			if state.IdentityHash == identityHash && reflect.DeepEqual(*state.Identity, canonical) {
				return nil
			}
			return errors.New("conflicting research identity registration")
		}
		prior := state.IntegrityHash
		state.Identity = &canonical
		state.IdentityHash = identityHash
		return appendLifecycle(state, "RESEARCH_IDENTITY_REGISTERED", "", StateIdentityRegistered, identityHash, prior, a.nowUTC())
	})
	if err != nil {
		return Snapshot{}, err
	}
	return a.Snapshot()
}

func (a *Authority) ReserveHoldout(request ReservationRequest) (ReservationRecord, error) {
	var out ReservationRecord
	err := a.mutate(func(state *Snapshot) error {
		if state.Identity == nil {
			return errors.New("complete V4 research identity must be registered before holdout reservation")
		}
		ledgerHash, err := HashVariantLedger(state.Identity.VariantLedger)
		if err != nil {
			return err
		}
		authorityHash, err := HashAuthoritySet(state.Identity.Authorities)
		if err != nil {
			return err
		}
		if request.SchemaVersion != ReservationSchemaVersion || request.ResearchIdentityHash != state.IdentityHash || !reflect.DeepEqual(normalizePartition(request.FinalHoldout), partition(*state.Identity, PartitionFinalHoldout)) || request.ProtocolSHA256 != state.Identity.Protocol.SHA256 || request.CheckpointSHA256 != state.Identity.Dataset.Checkpoint.SHA256 || request.VariantLedgerSHA256 != ledgerHash || request.AuthoritySetSHA256 != authorityHash {
			return errors.New("holdout reservation identity substitution or incomplete binding")
		}
		if state.Reservation != nil {
			if sameReservationRequest(*state.Reservation, request) {
				out = *state.Reservation
				return nil
			}
			return errors.New("conflicting duplicate holdout reservation")
		}
		if err := checkExpected(state, StateIdentityRegistered, request.ExpectedSequence, request.ExpectedStateHash); err != nil {
			return err
		}
		requestHash, err := hashCanonical(struct {
			Identity   string    `json:"research_identity_hash"`
			Holdout    Partition `json:"final_holdout"`
			Protocol   string    `json:"protocol_sha256"`
			Checkpoint string    `json:"checkpoint_sha256"`
			Ledger     string    `json:"variant_ledger_sha256"`
			Authority  string    `json:"authority_set_sha256"`
		}{request.ResearchIdentityHash, normalizePartition(request.FinalHoldout), request.ProtocolSHA256, request.CheckpointSHA256, request.VariantLedgerSHA256, request.AuthoritySetSHA256})
		if err != nil {
			return err
		}
		prior := state.IntegrityHash
		out = ReservationRecord{SchemaVersion: ReservationSchemaVersion, ReservationID: "reservation:" + requestHash[len("sha256:"):], ResearchIdentityHash: state.IdentityHash, FinalHoldout: partition(*state.Identity, PartitionFinalHoldout), ProtocolSHA256: request.ProtocolSHA256, CheckpointSHA256: request.CheckpointSHA256, VariantLedgerSHA256: request.VariantLedgerSHA256, AuthoritySetSHA256: request.AuthoritySetSHA256, CandidateFrozen: false, CreatedAt: a.nowUTC()}
		out.RecordHash, err = hashReservation(out)
		if err != nil {
			return err
		}
		state.Reservation = &out
		return appendLifecycle(state, "HOLDOUT_RESERVED", StateIdentityRegistered, StateHoldoutReserved, out.RecordHash, prior, out.CreatedAt)
	})
	return out, err
}

func (a *Authority) AuthorizeDevelopment(request TransitionRequest) (AuthorizationRecord, error) {
	return a.authorize(StateHoldoutReserved, StateDevelopmentAuthorized, PartitionDevelopment, "DEVELOPMENT_AUTHORIZED", request, false)
}

func (a *Authority) SealDevelopment(request SealRequest) (Snapshot, error) {
	return a.seal(StateDevelopmentAuthorized, StateDevelopmentSealed, PartitionDevelopment, "DEVELOPMENT_SEALED", request)
}

func (a *Authority) AuthorizeValidation(request TransitionRequest) (AuthorizationRecord, error) {
	return a.authorize(StateDevelopmentSealed, StateValidationAuthorized, PartitionValidation, "VALIDATION_AUTHORIZED", request, false)
}

func (a *Authority) RejectNoNominee(expectedSequence uint64, expectedStateHash, resultSealSHA256, reason string) (Snapshot, error) {
	if !validSHA256(resultSealSHA256) || !validText(reason) {
		return Snapshot{}, errors.New("terminal no-nominee result requires a result seal and reason")
	}
	err := a.mutate(func(state *Snapshot) error {
		if err := checkExpected(state, StateDevelopmentSealed, expectedSequence, expectedStateHash); err != nil {
			return err
		}
		prior := state.IntegrityHash
		state.Disposition = &Disposition{State: StateRejected, Kind: "PERFORMANCE", Reason: reason}
		return appendLifecycle(state, "NO_NOMINEE_REJECTED", StateDevelopmentSealed, StateRejected, resultSealSHA256, prior, a.nowUTC())
	})
	if err != nil {
		return Snapshot{}, err
	}
	return a.Snapshot()
}

func (a *Authority) SealValidation(request SealRequest) (Snapshot, error) {
	return a.seal(StateValidationAuthorized, StateValidationSealed, PartitionValidation, "VALIDATION_SEALED", request)
}

func (a *Authority) FreezeCandidate(expectedSequence uint64, expectedStateHash string, candidate FrozenCandidate) (FrozenCandidate, error) {
	var out FrozenCandidate
	err := a.mutate(func(state *Snapshot) error {
		if state.State != StateValidationSealed && state.State != StateValidationSetSealed {
			return fmt.Errorf("lifecycle transition requires %s or %s, got %s", StateValidationSealed, StateValidationSetSealed, state.State)
		}
		if state.Sequence != expectedSequence || state.IntegrityHash != expectedStateHash {
			return errors.New("stale research lifecycle request")
		}
		if state.Identity == nil {
			return errors.New("research identity missing")
		}
		if state.State == StateValidationSetSealed {
			if state.DevelopmentNominee == nil || !state.DevelopmentNominee.Exists || candidate.VariantID != state.DevelopmentNominee.VariantID || candidate.ConfigurationSHA256 != state.DevelopmentNominee.ConfigurationSHA256 {
				return errors.New("only the sealed DEVELOPMENT nominee may be frozen after multi-variant VALIDATION")
			}
			validationSet := executionSetByStage(state, PartitionValidation)
			if validationSet == nil || validationSet.CompletionState != "SEALED" {
				return errors.New("complete sealed VALIDATION execution set is required")
			}
			for _, receipt := range validationSet.ExecutionReceipts {
				if !receipt.MandatoryGatesPassed {
					return errors.New("failed nominee or mandatory stability neighbor cannot be frozen")
				}
			}
		}
		registered, ok := variant(*state.Identity, candidate.VariantID)
		if !ok || registered.ConfigurationSHA256 != candidate.ConfigurationSHA256 {
			return errors.New("frozen candidate is not an exact registered variant")
		}
		if !candidate.NoUnresolvedDefaults || !validSHA256(candidate.ExecutableSHA256) {
			return errors.New("candidate freeze requires exact executable identity and no unresolved defaults")
		}
		binding := ExecutionBinding{VariantID: candidate.VariantID, ConfigurationSHA256: candidate.ConfigurationSHA256, ProtocolSHA256: candidate.ProtocolSHA256, CheckpointSHA256: candidate.CheckpointSHA256, IndependenceSHA256: candidate.IndependenceSHA256, UncertaintySHA256: candidate.UncertaintySHA256, ConcentrationSHA256: candidate.ConcentrationSHA256, QualificationGateSHA256: candidate.QualificationGateSHA256, RunnerGitCommit: state.Identity.Repositories.RunnerGitCommit, RunnerExecutableSHA256: candidate.ExecutableSHA256, Partition: PartitionValidation}
		if err := validateBinding(*state.Identity, binding, PartitionValidation); err != nil {
			return err
		}
		if candidate.ExecutableSHA256 != state.Identity.Repositories.RunnerExecutableSHA256 {
			return errors.New("frozen executable does not match registered runner")
		}
		prior := state.IntegrityHash
		candidate.FrozenAt = a.nowUTC()
		candidate.FrozenIdentityHash = ""
		hash, err := hashFrozenCandidate(candidate)
		if err != nil {
			return err
		}
		candidate.FrozenIdentityHash = hash
		out = candidate
		state.FrozenCandidate = &out
		return appendLifecycle(state, "CANDIDATE_FROZEN", state.State, StateCandidateFrozen, hash, prior, candidate.FrozenAt)
	})
	return out, err
}

func (a *Authority) AuthorizeFinalHoldout(request TransitionRequest) (AuthorizationRecord, error) {
	return a.authorize(StateCandidateFrozen, StateFinalAuthorized, PartitionFinalHoldout, "FINAL_HOLDOUT_AUTHORIZED", request, true)
}

func (a *Authority) SealFinalHoldout(request SealRequest) (Snapshot, error) {
	return a.seal(StateFinalAuthorized, StateFinalSealed, PartitionFinalHoldout, "FINAL_HOLDOUT_SEALED", request)
}

func (a *Authority) Qualify(expectedSequence uint64, expectedStateHash, evidenceSHA256 string) (Snapshot, error) {
	return a.dispose(expectedSequence, expectedStateHash, StateQualified, "PERFORMANCE", evidenceSHA256, "QUALIFIED")
}

func (a *Authority) RejectPerformance(expectedSequence uint64, expectedStateHash, evidenceSHA256, reason string) (Snapshot, error) {
	return a.dispose(expectedSequence, expectedStateHash, StateRejected, "PERFORMANCE", evidenceSHA256, reason)
}

func (a *Authority) BlockIntegrity(expectedSequence uint64, expectedStateHash, evidenceSHA256, reason string) (Snapshot, error) {
	if !validSHA256(evidenceSHA256) || !validText(reason) {
		return Snapshot{}, errors.New("integrity block requires sealed evidence and a reason")
	}
	err := a.mutate(func(state *Snapshot) error {
		if state.State == StateQualified || state.State == StateRejected || state.State == StateBlocked {
			return errors.New("terminal research state cannot be changed")
		}
		if state.Sequence != expectedSequence || state.IntegrityHash != expectedStateHash {
			return errors.New("stale research lifecycle request")
		}
		prior, from := state.IntegrityHash, state.State
		state.Disposition = &Disposition{State: StateBlocked, Kind: "INTEGRITY", Reason: reason}
		return appendLifecycle(state, "INTEGRITY_BLOCKED", from, StateBlocked, evidenceSHA256, prior, a.nowUTC())
	})
	if err != nil {
		return Snapshot{}, err
	}
	return a.Snapshot()
}

func (a *Authority) ConsumeBeforeAccess(authorization AuthorizationRecord, binding ExecutionBinding, access func() error) (AccessReceipt, error) {
	if access == nil {
		return AccessReceipt{}, errors.New("protected access callback is required")
	}
	var receipt AccessReceipt
	err := a.mutate(func(state *Snapshot) error {
		if state.Identity == nil {
			return errors.New("research identity missing")
		}
		var persisted *AuthorizationRecord
		for i := range state.Authorizations {
			if state.Authorizations[i].AuthorizationID == authorization.AuthorizationID {
				persisted = &state.Authorizations[i]
				break
			}
		}
		if persisted == nil || !reflect.DeepEqual(*persisted, authorization) {
			return errors.New("authorization is not an exact persisted RIF record")
		}
		if state.State != authorization.LifecycleState || !reflect.DeepEqual(authorization.Binding, binding) {
			return errors.New("authorization cannot be replayed against another state or execution binding")
		}
		if authorization.ExpiresAt != nil && !a.nowUTC().Before(authorization.ExpiresAt.UTC()) {
			return errors.New("partition authorization expired")
		}
		for _, prior := range state.AccessReceipts {
			if prior.AuthorizationID == authorization.AuthorizationID {
				return errors.New("partition authorization has already been consumed")
			}
		}
		if err := validateBinding(*state.Identity, binding, binding.Partition); err != nil {
			return err
		}
		priorState := state.IntegrityHash
		previous := ""
		if len(state.AccessReceipts) > 0 {
			previous = state.AccessReceipts[len(state.AccessReceipts)-1].RecordHash
		}
		receipt = AccessReceipt{SchemaVersion: AccessReceiptSchemaVersion, Sequence: state.Sequence + 1, AuthorizationID: authorization.AuthorizationID, ResearchIdentityHash: state.IdentityHash, Binding: binding, AccessedAt: a.nowUTC(), PriorLifecycleStateHash: priorState, PreviousHash: previous}
		var err error
		receipt.RecordHash, err = hashAccessReceipt(receipt)
		if err != nil {
			return err
		}
		state.AccessReceipts = append(state.AccessReceipts, receipt)
		return appendLifecycle(state, string(binding.Partition)+"_ACCESS_CONSUMED", state.State, state.State, receipt.RecordHash, priorState, receipt.AccessedAt)
	})
	if err != nil {
		return AccessReceipt{}, err
	}
	callbackErr := access()
	return receipt, callbackErr
}

func (a *Authority) authorize(from, to LifecycleState, partitionName PartitionName, event string, request TransitionRequest, requireFrozen bool) (AuthorizationRecord, error) {
	var out AuthorizationRecord
	err := a.mutate(func(state *Snapshot) error {
		if err := checkExpected(state, from, request.ExpectedSequence, request.ExpectedStateHash); err != nil {
			return err
		}
		if state.Identity == nil || state.Reservation == nil {
			return errors.New("registered identity and immutable holdout reservation are required")
		}
		if request.Binding.Partition != partitionName {
			return errors.New("authorization partition mismatch")
		}
		if err := validateBinding(*state.Identity, request.Binding, partitionName); err != nil {
			return err
		}
		if requireFrozen {
			if state.FrozenCandidate == nil {
				return errors.New("FINAL_HOLDOUT authorization requires a frozen candidate")
			}
			frozen := state.FrozenCandidate
			if request.Binding.VariantID != frozen.VariantID || request.Binding.ConfigurationSHA256 != frozen.ConfigurationSHA256 || request.Binding.RunnerExecutableSHA256 != frozen.ExecutableSHA256 || request.Binding.ProtocolSHA256 != frozen.ProtocolSHA256 || request.Binding.CheckpointSHA256 != frozen.CheckpointSHA256 || request.Binding.IndependenceSHA256 != frozen.IndependenceSHA256 || request.Binding.UncertaintySHA256 != frozen.UncertaintySHA256 || request.Binding.ConcentrationSHA256 != frozen.ConcentrationSHA256 || request.Binding.QualificationGateSHA256 != frozen.QualificationGateSHA256 {
				return errors.New("FINAL_HOLDOUT authorization does not exactly match frozen candidate")
			}
		}
		prior := state.IntegrityHash
		previous := ""
		if len(state.Authorizations) > 0 {
			previous = state.Authorizations[len(state.Authorizations)-1].RecordHash
		}
		bindingHash, err := hashCanonical(request.Binding)
		if err != nil {
			return err
		}
		issued := a.nowUTC()
		out = AuthorizationRecord{SchemaVersion: AuthorizationSchemaVersion, AuthorizationID: "authorization:" + bindingHash[len("sha256:"):] + fmt.Sprintf(":%d", state.Sequence+1), Sequence: state.Sequence + 1, ResearchIdentityHash: state.IdentityHash, LifecycleState: to, Binding: request.Binding, IssuedAt: issued, OneShot: true, PriorLifecycleStateHash: prior, PreviousHash: previous}
		out.RecordHash, err = hashAuthorization(out)
		if err != nil {
			return err
		}
		state.Authorizations = append(state.Authorizations, out)
		return appendLifecycle(state, event, from, to, out.RecordHash, prior, issued)
	})
	return out, err
}

func (a *Authority) seal(from, to LifecycleState, partitionName PartitionName, event string, request SealRequest) (Snapshot, error) {
	if !validSHA256(request.AccessReceiptHash) || !validSHA256(request.ExecutionReceiptSHA256) || !validSHA256(request.ResultSealSHA256) {
		return Snapshot{}, errors.New("durable access, execution, and result seals are required")
	}
	err := a.mutate(func(state *Snapshot) error {
		if err := checkExpected(state, from, request.ExpectedSequence, request.ExpectedStateHash); err != nil {
			return err
		}
		if state.Identity == nil || request.Binding.Partition != partitionName {
			return errors.New("seal partition mismatch")
		}
		if err := validateBinding(*state.Identity, request.Binding, partitionName); err != nil {
			return err
		}
		found := false
		for _, receipt := range state.AccessReceipts {
			if receipt.RecordHash == request.AccessReceiptHash && reflect.DeepEqual(receipt.Binding, request.Binding) {
				found = true
				break
			}
		}
		if !found {
			return errors.New("durable exact partition access receipt was not found")
		}
		prior := state.IntegrityHash
		evidence, err := hashCanonical(struct {
			Access    string `json:"access_receipt_hash"`
			Execution string `json:"execution_receipt_sha256"`
			Result    string `json:"result_seal_sha256"`
		}{request.AccessReceiptHash, request.ExecutionReceiptSHA256, request.ResultSealSHA256})
		if err != nil {
			return err
		}
		return appendLifecycle(state, event, from, to, evidence, prior, a.nowUTC())
	})
	if err != nil {
		return Snapshot{}, err
	}
	return a.Snapshot()
}

func (a *Authority) dispose(expectedSequence uint64, expectedStateHash string, destination LifecycleState, kind, evidenceSHA256, reason string) (Snapshot, error) {
	if !validSHA256(evidenceSHA256) || !validText(reason) || (destination != StateQualified && destination != StateRejected) {
		return Snapshot{}, errors.New("final disposition is invalid")
	}
	err := a.mutate(func(state *Snapshot) error {
		if err := checkExpected(state, StateFinalSealed, expectedSequence, expectedStateHash); err != nil {
			return err
		}
		prior := state.IntegrityHash
		state.Disposition = &Disposition{State: destination, Kind: kind, Reason: reason}
		return appendLifecycle(state, string(destination), StateFinalSealed, destination, evidenceSHA256, prior, a.nowUTC())
	})
	if err != nil {
		return Snapshot{}, err
	}
	return a.Snapshot()
}

func checkExpected(state *Snapshot, required LifecycleState, sequence uint64, stateHash string) error {
	if state.State != required {
		return fmt.Errorf("lifecycle transition requires %s, got %s", required, state.State)
	}
	if state.Sequence != sequence || state.IntegrityHash != stateHash {
		return errors.New("stale research lifecycle request")
	}
	return nil
}

func validateBinding(identity IdentityV4, binding ExecutionBinding, expected PartitionName) error {
	registered, ok := variant(identity, binding.VariantID)
	if !ok || registered.ConfigurationSHA256 != binding.ConfigurationSHA256 {
		return errors.New("execution binding references an unregistered or mutated configuration")
	}
	if binding.Partition != expected || partition(identity, expected).Name == "" {
		return errors.New("execution binding partition is not registered")
	}
	if binding.ProtocolSHA256 != identity.Protocol.SHA256 || binding.CheckpointSHA256 != identity.Dataset.Checkpoint.SHA256 || binding.IndependenceSHA256 != identity.Authorities.Independence.SHA256 || binding.UncertaintySHA256 != identity.Authorities.Uncertainty.SHA256 || binding.ConcentrationSHA256 != identity.Authorities.ConcentrationSHA256 || binding.QualificationGateSHA256 != identity.Authorities.QualificationGateSet.SHA256 || binding.RunnerGitCommit != identity.Repositories.RunnerGitCommit || binding.RunnerExecutableSHA256 != identity.Repositories.RunnerExecutableSHA256 {
		return errors.New("execution binding protocol, checkpoint, authority, gate, or runner substitution")
	}
	return nil
}

func appendLifecycle(state *Snapshot, event string, from, to LifecycleState, evidence, prior string, at time.Time) error {
	previous := ""
	if len(state.LifecycleHistory) > 0 {
		previous = state.LifecycleHistory[len(state.LifecycleHistory)-1].RecordHash
	}
	record := LifecycleRecord{SchemaVersion: LifecycleRecordVersion, Sequence: state.Sequence + 1, EventType: event, FromState: from, ToState: to, OccurredAt: at.UTC(), EvidenceSHA256: evidence, PriorStateHash: prior, PreviousHash: previous}
	var err error
	record.RecordHash, err = hashLifecycleRecord(record)
	if err != nil {
		return err
	}
	state.LifecycleHistory = append(state.LifecycleHistory, record)
	state.State, state.Sequence = to, record.Sequence
	return nil
}

func normalizePartition(value Partition) Partition {
	value.Interval = normalizeInterval(value.Interval)
	return value
}
func sameReservationRequest(record ReservationRecord, request ReservationRequest) bool {
	return record.ResearchIdentityHash == request.ResearchIdentityHash && reflect.DeepEqual(record.FinalHoldout, normalizePartition(request.FinalHoldout)) && record.ProtocolSHA256 == request.ProtocolSHA256 && record.CheckpointSHA256 == request.CheckpointSHA256 && record.VariantLedgerSHA256 == request.VariantLedgerSHA256 && record.AuthoritySetSHA256 == request.AuthoritySetSHA256
}
func (a *Authority) nowUTC() time.Time {
	if a.now == nil {
		return time.Now().UTC()
	}
	return a.now().UTC()
}
