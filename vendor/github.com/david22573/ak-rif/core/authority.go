package core

import (
	"errors"
	"fmt"
	"time"
)

// ReasonCode is a stable, machine-readable authority decision code.
type ReasonCode string

const (
	CodeUnknownLifecycleState                 ReasonCode = "UNKNOWN_LIFECYCLE_STATE"
	CodeTransitionNotAllowed                  ReasonCode = "TRANSITION_NOT_ALLOWED"
	CodeCurrentStateMismatch                  ReasonCode = "CURRENT_STATE_MISMATCH"
	CodeLifecycleEpochMismatch                ReasonCode = "LIFECYCLE_EPOCH_MISMATCH"
	CodeCandidateNotRegistered                ReasonCode = "CANDIDATE_NOT_REGISTERED"
	CodeCandidateAlreadyExists                ReasonCode = "CANDIDATE_ALREADY_REGISTERED"
	CodeRequiredEvidenceMissing               ReasonCode = "REQUIRED_EVIDENCE_MISSING"
	CodeEvidenceInvalid                       ReasonCode = "EVIDENCE_INVALID"
	CodeRequiredLockFieldMissing              ReasonCode = "REQUIRED_LOCK_FIELD_MISSING"
	CodeLockFieldMismatch                     ReasonCode = "LOCK_FIELD_MISMATCH"
	CodeLockSchemaUnsupported                 ReasonCode = "LOCK_SCHEMA_UNSUPPORTED"
	CodeLockIntegrityMismatch                 ReasonCode = "LOCK_INTEGRITY_MISMATCH"
	CodeHoldoutLedgerMissing                  ReasonCode = "HOLDOUT_LEDGER_MISSING"
	CodeHoldoutLedgerCorrupt                  ReasonCode = "HOLDOUT_LEDGER_CORRUPT"
	CodeHoldoutLimitExceeded                  ReasonCode = "HOLDOUT_LIMIT_EXCEEDED"
	CodeHoldoutOperationConflict              ReasonCode = "HOLDOUT_OPERATION_CONFLICT"
	CodeHoldoutReservationMissing             ReasonCode = "HOLDOUT_RESERVATION_MISSING"
	CodeHoldoutReservationConflict            ReasonCode = "HOLDOUT_RESERVATION_CONFLICT"
	CodeHoldoutAuthorizationMissing           ReasonCode = "HOLDOUT_AUTHORIZATION_MISSING"
	CodeHoldoutAuthorizationConflict          ReasonCode = "HOLDOUT_AUTHORIZATION_CONFLICT"
	CodeHoldoutAccessConsumed                 ReasonCode = "HOLDOUT_ACCESS_ALREADY_CONSUMED"
	CodeFrozenIdentityMissing                 ReasonCode = "FROZEN_IDENTITY_MISSING"
	CodeFrozenIdentityMismatch                ReasonCode = "FROZEN_IDENTITY_MISMATCH"
	CodeArtifactSchemaUnsupported             ReasonCode = "ARTIFACT_SCHEMA_UNSUPPORTED"
	CodeArtifactInvalid                       ReasonCode = "ARTIFACT_INVALID"
	CodeArtifactTooLarge                      ReasonCode = "ARTIFACT_TOO_LARGE"
	CodeAtomicWriteFailed                     ReasonCode = "ATOMIC_WRITE_FAILED"
	CodeUnsafePath                            ReasonCode = "UNSAFE_PATH"
	CodeStateStoreMissing                     ReasonCode = "STATE_STORE_MISSING"
	CodeStateStoreCorrupt                     ReasonCode = "STATE_STORE_CORRUPT"
	CodeResearchIdentitySchemaUnsupported     ReasonCode = "RESEARCH_IDENTITY_SCHEMA_UNSUPPORTED"
	CodeDatasetIDMissing                      ReasonCode = "DATASET_ID_MISSING"
	CodeDatasetIDInvalid                      ReasonCode = "DATASET_ID_INVALID"
	CodeDatasetVersionMissing                 ReasonCode = "DATASET_VERSION_MISSING"
	CodeDatasetVersionInvalid                 ReasonCode = "DATASET_VERSION_INVALID"
	CodeResearchWindowStartMissing            ReasonCode = "RESEARCH_WINDOW_START_MISSING"
	CodeResearchWindowEndMissing              ReasonCode = "RESEARCH_WINDOW_END_MISSING"
	CodeResearchWindowInvalid                 ReasonCode = "RESEARCH_WINDOW_INVALID"
	CodeEvaluationCutoffMissing               ReasonCode = "EVALUATION_CUTOFF_MISSING"
	CodeEvaluationCutoffInvalid               ReasonCode = "EVALUATION_CUTOFF_INVALID"
	CodeManifestIDMissing                     ReasonCode = "MANIFEST_ID_MISSING"
	CodeManifestIDInvalid                     ReasonCode = "MANIFEST_ID_INVALID"
	CodeManifestHashMissing                   ReasonCode = "MANIFEST_HASH_MISSING"
	CodeManifestHashInvalid                   ReasonCode = "MANIFEST_HASH_INVALID"
	CodeCoveragePolicyVersionMissing          ReasonCode = "COVERAGE_POLICY_VERSION_MISSING"
	CodeAvailabilityPolicyVersionMissing      ReasonCode = "AVAILABILITY_POLICY_VERSION_MISSING"
	CodePolicyVersionInvalid                  ReasonCode = "POLICY_VERSION_INVALID"
	CodeGovernanceHashMissing                 ReasonCode = "GOVERNANCE_HASH_MISSING"
	CodeGovernanceHashInvalid                 ReasonCode = "GOVERNANCE_HASH_INVALID"
	CodeGovernanceBindingUnexpected           ReasonCode = "GOVERNANCE_BINDING_UNEXPECTED"
	CodeGovernanceAuthorityMissing            ReasonCode = "GOVERNANCE_AUTHORITY_MISSING"
	CodeGovernanceAuthorityRejected           ReasonCode = "GOVERNANCE_AUTHORITY_REJECTED"
	CodeResearchIdentityMissing               ReasonCode = "RESEARCH_IDENTITY_MISSING"
	CodeLegacyLockNotPITBindable              ReasonCode = "LEGACY_LOCK_NOT_PIT_BINDABLE"
	CodeLockSchemaUpgradeRequired             ReasonCode = "LOCK_SCHEMA_UPGRADE_REQUIRED"
	CodePITEvidenceMissing                    ReasonCode = "PIT_EVIDENCE_MISSING"
	CodePITEvidenceEmpty                      ReasonCode = "PIT_EVIDENCE_EMPTY"
	CodePITEvidenceTooLarge                   ReasonCode = "PIT_EVIDENCE_TOO_LARGE"
	CodePITEvidenceMalformed                  ReasonCode = "PIT_EVIDENCE_MALFORMED"
	CodePITEvidenceSchemaUnsupported          ReasonCode = "PIT_EVIDENCE_SCHEMA_UNSUPPORTED"
	CodePITEvidenceRequiredFieldMissing       ReasonCode = "PIT_EVIDENCE_REQUIRED_FIELD_MISSING"
	CodePITEvidenceTimestampInvalid           ReasonCode = "PIT_EVIDENCE_TIMESTAMP_INVALID"
	CodePITEvidenceDigestInvalid              ReasonCode = "PIT_EVIDENCE_DIGEST_INVALID"
	CodePITEvidenceVerdictIneligible          ReasonCode = "PIT_EVIDENCE_VERDICT_INELIGIBLE"
	CodePITEvidenceVerdictUnknown             ReasonCode = "PIT_EVIDENCE_VERDICT_UNKNOWN"
	CodePITEvidenceVerdictContradictory       ReasonCode = "PIT_EVIDENCE_VERDICT_CONTRADICTORY"
	CodePITEvidenceCoverageNotStrict          ReasonCode = "PIT_EVIDENCE_COVERAGE_NOT_STRICT"
	CodePITEvidenceAvailabilityNotStrict      ReasonCode = "PIT_EVIDENCE_AVAILABILITY_NOT_STRICT"
	CodePITEvidenceIntegrityMismatch          ReasonCode = "PIT_EVIDENCE_INTEGRITY_MISMATCH"
	CodePITEvidenceDatasetMismatch            ReasonCode = "PIT_EVIDENCE_DATASET_MISMATCH"
	CodePITEvidenceDatasetVersionMismatch     ReasonCode = "PIT_EVIDENCE_DATASET_VERSION_MISMATCH"
	CodePITEvidenceWindowStartMismatch        ReasonCode = "PIT_EVIDENCE_WINDOW_START_MISMATCH"
	CodePITEvidenceWindowEndMismatch          ReasonCode = "PIT_EVIDENCE_WINDOW_END_MISMATCH"
	CodePITEvidenceCutoffMismatch             ReasonCode = "PIT_EVIDENCE_CUTOFF_MISMATCH"
	CodePITEvidenceManifestIDMismatch         ReasonCode = "PIT_EVIDENCE_MANIFEST_ID_MISMATCH"
	CodePITEvidenceManifestHashMismatch       ReasonCode = "PIT_EVIDENCE_MANIFEST_HASH_MISMATCH"
	CodePITEvidenceCoveragePolicyMismatch     ReasonCode = "PIT_EVIDENCE_COVERAGE_POLICY_MISMATCH"
	CodePITEvidenceAvailabilityPolicyMismatch ReasonCode = "PIT_EVIDENCE_AVAILABILITY_POLICY_MISMATCH"
	CodePITEvidenceIDConflict                 ReasonCode = "PIT_EVIDENCE_ID_CONFLICT"
	CodePITEvidenceHashConflict               ReasonCode = "PIT_EVIDENCE_HASH_CONFLICT"
	CodePITEvidenceAlreadyBound               ReasonCode = "PIT_EVIDENCE_ALREADY_BOUND"
	CodePITEvidenceReplayConflict             ReasonCode = "PIT_EVIDENCE_REPLAY_CONFLICT"
	CodePITEvidenceResearchLockChanged        ReasonCode = "PIT_EVIDENCE_RESEARCH_LOCK_CHANGED"
	CodePITEvidenceEpochMismatch              ReasonCode = "PIT_EVIDENCE_EPOCH_MISMATCH"
	CodePITEvidenceBindingCorrupt             ReasonCode = "PIT_EVIDENCE_BINDING_CORRUPT"
)

// Reason describes one failed requirement without including protected values.
type Reason struct {
	Code    ReasonCode `json:"code"`
	Field   string     `json:"field,omitempty"`
	Message string     `json:"message"`
}

// AuthorityError carries all reasons for a rejected authority operation.
type AuthorityError struct {
	Operation string   `json:"operation"`
	Reasons   []Reason `json:"reasons"`
	Cause     error    `json:"-"`
}

func (e *AuthorityError) Error() string {
	if len(e.Reasons) == 0 {
		return fmt.Sprintf("%s failed", e.Operation)
	}
	return fmt.Sprintf("%s failed: %s", e.Operation, e.Reasons[0].Code)
}

func (e *AuthorityError) Unwrap() error { return e.Cause }

// HasReason reports whether err (including wrapped errors) has code.
func HasReason(err error, code ReasonCode) bool {
	for err != nil {
		var authorityErr *AuthorityError
		if errors.As(err, &authorityErr) && slicesContainReason(authorityErr.Reasons, code) {
			return true
		}
		err = errors.Unwrap(err)
	}
	return false
}

func slicesContainReason(reasons []Reason, code ReasonCode) bool {
	for _, reason := range reasons {
		if reason.Code == code {
			return true
		}
	}
	return false
}

// NewAuthorityError constructs a structured authority failure.
func NewAuthorityError(operation string, code ReasonCode, field, message string, cause error) *AuthorityError {
	return &AuthorityError{
		Operation: operation,
		Reasons:   []Reason{{Code: code, Field: field, Message: message}},
		Cause:     cause,
	}
}

// AuditEvent is a secret-free record of an authority decision.
type AuditEvent struct {
	SchemaVersion    string         `json:"schema_version"`
	EventType        string         `json:"event_type"`
	CandidateID      string         `json:"candidate_id,omitempty"`
	CandidateVersion string         `json:"candidate_version,omitempty"`
	FromState        CandidateState `json:"from_state,omitempty"`
	ToState          CandidateState `json:"to_state,omitempty"`
	LifecycleEpoch   uint64         `json:"lifecycle_epoch,omitempty"`
	OccurredAt       time.Time      `json:"occurred_at"`
	Reasons          []Reason       `json:"reasons,omitempty"`
}

const AuditEventSchemaVersion = "ak.rif.audit_event.v1"
