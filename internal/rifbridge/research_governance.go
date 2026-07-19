package rifbridge

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"reflect"
	"strings"
	"time"
)

const (
	ResearchGovernanceEnvelopeSchemaVersion   = "ak.rif.research_governance_envelope.v1"
	ResearchGovernanceEnvelopeSchemaVersionV2 = "ak.rif.research_governance_envelope.v2"
	ResearchGovernanceStoreSchemaVersion      = "ak.rif.research_governance_store.v1"
	ResearchGovernanceStoreSchemaVersionV2    = "ak.rif.research_governance_store.v2"
	ResearchLifecycleRecordSchemaVersion      = "ak.rif.research_lifecycle_record.v1"
	PartitionAuthorizationSchemaVersion       = "ak.rif.partition_authorization.v1"
	PartitionAccessReceiptSchemaVersion       = "ak.rif.partition_access_receipt.v1"
	HoldoutReservationSchemaVersion           = "ak.rif.holdout_reservation.v1"
)

type ResearchExecutionBinding struct {
	VariantID               string `json:"variant_id"`
	ConfigurationSHA256     string `json:"configuration_sha256"`
	ProtocolSHA256          string `json:"protocol_sha256"`
	CheckpointSHA256        string `json:"checkpoint_sha256"`
	IndependenceSHA256      string `json:"independence_sha256"`
	UncertaintySHA256       string `json:"uncertainty_sha256"`
	ConcentrationSHA256     string `json:"concentration_sha256"`
	QualificationGateSHA256 string `json:"qualification_gate_set_sha256"`
	RunnerGitCommit         string `json:"runner_git_commit"`
	RunnerExecutableSHA256  string `json:"runner_executable_sha256"`
	Partition               string `json:"partition"`
}

type PartitionAuthorization struct {
	SchemaVersion           string                   `json:"schema_version"`
	AuthorizationID         string                   `json:"authorization_id"`
	Sequence                uint64                   `json:"sequence"`
	ResearchIdentityHash    string                   `json:"research_identity_hash"`
	LifecycleState          string                   `json:"lifecycle_state"`
	Binding                 ResearchExecutionBinding `json:"execution_binding"`
	IssuedAt                time.Time                `json:"issued_at"`
	ExpiresAt               *time.Time               `json:"expires_at,omitempty"`
	OneShot                 bool                     `json:"one_shot"`
	PriorLifecycleStateHash string                   `json:"prior_lifecycle_state_hash"`
	PreviousHash            string                   `json:"previous_hash"`
	RecordHash              string                   `json:"record_hash"`
}

type PartitionAccessReceipt struct {
	SchemaVersion           string                   `json:"schema_version"`
	Sequence                uint64                   `json:"sequence"`
	AuthorizationID         string                   `json:"authorization_id"`
	ResearchIdentityHash    string                   `json:"research_identity_hash"`
	Binding                 ResearchExecutionBinding `json:"execution_binding"`
	AccessedAt              time.Time                `json:"accessed_at"`
	PriorLifecycleStateHash string                   `json:"prior_lifecycle_state_hash"`
	PreviousHash            string                   `json:"previous_hash"`
	RecordHash              string                   `json:"record_hash"`
}

type ResearchLifecycleRecord struct {
	SchemaVersion  string    `json:"schema_version"`
	Sequence       uint64    `json:"sequence"`
	EventType      string    `json:"event_type"`
	FromState      string    `json:"from_state,omitempty"`
	ToState        string    `json:"to_state"`
	OccurredAt     time.Time `json:"occurred_at"`
	EvidenceSHA256 string    `json:"evidence_sha256"`
	PriorStateHash string    `json:"prior_state_hash"`
	PreviousHash   string    `json:"previous_hash"`
	RecordHash     string    `json:"record_hash"`
}

type HoldoutReservation struct {
	SchemaVersion        string          `json:"schema_version"`
	ReservationID        string          `json:"reservation_id"`
	ResearchIdentityHash string          `json:"research_identity_hash"`
	FinalHoldout         json.RawMessage `json:"final_holdout"`
	ProtocolSHA256       string          `json:"protocol_sha256"`
	CheckpointSHA256     string          `json:"checkpoint_sha256"`
	VariantLedgerSHA256  string          `json:"variant_ledger_sha256"`
	AuthoritySetSHA256   string          `json:"authority_set_sha256"`
	CandidateFrozen      bool            `json:"candidate_frozen"`
	CreatedAt            time.Time       `json:"created_at"`
	RecordHash           string          `json:"record_hash"`
}

type FrozenResearchCandidate struct {
	VariantID               string    `json:"variant_id"`
	ConfigurationSHA256     string    `json:"configuration_sha256"`
	ExecutableSHA256        string    `json:"executable_sha256"`
	ProtocolSHA256          string    `json:"protocol_sha256"`
	CheckpointSHA256        string    `json:"checkpoint_sha256"`
	IndependenceSHA256      string    `json:"independence_sha256"`
	UncertaintySHA256       string    `json:"uncertainty_sha256"`
	ConcentrationSHA256     string    `json:"concentration_sha256"`
	QualificationGateSHA256 string    `json:"qualification_gate_set_sha256"`
	NoUnresolvedDefaults    bool      `json:"no_unresolved_defaults"`
	FrozenAt                time.Time `json:"frozen_at"`
	FrozenIdentityHash      string    `json:"frozen_identity_hash"`
}

type ResearchDisposition struct {
	State  string `json:"state"`
	Kind   string `json:"kind"`
	Reason string `json:"reason"`
}

type ResearchGovernanceSnapshot struct {
	SchemaVersion      string                    `json:"schema_version"`
	Identity           json.RawMessage           `json:"research_identity,omitempty"`
	IdentityHash       string                    `json:"research_identity_hash,omitempty"`
	Reservation        *HoldoutReservation       `json:"holdout_reservation,omitempty"`
	State              string                    `json:"lifecycle_state,omitempty"`
	Sequence           uint64                    `json:"sequence"`
	FrozenCandidate    *FrozenResearchCandidate  `json:"frozen_candidate,omitempty"`
	Disposition        *ResearchDisposition      `json:"disposition,omitempty"`
	Authorizations     []PartitionAuthorization  `json:"authorizations"`
	AccessReceipts     []PartitionAccessReceipt  `json:"access_receipts"`
	StageExecutionSets []StageExecutionSet       `json:"stage_execution_sets,omitempty"`
	DevelopmentNominee *DevelopmentNominee       `json:"development_nominee,omitempty"`
	LifecycleHistory   []ResearchLifecycleRecord `json:"lifecycle_history"`
	IntegrityHash      string                    `json:"integrity_hash"`
}

type ResearchGovernanceEnvelope struct {
	SchemaVersion string                     `json:"schema_version"`
	Snapshot      ResearchGovernanceSnapshot `json:"snapshot"`
	Authorization *PartitionAuthorization    `json:"partition_authorization,omitempty"`
	EnvelopeHash  string                     `json:"envelope_hash"`
}

func ParseResearchGovernanceEnvelopeJSON(data []byte) (ResearchGovernanceEnvelope, error) {
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()
	var envelope ResearchGovernanceEnvelope
	if err := decoder.Decode(&envelope); err != nil {
		return ResearchGovernanceEnvelope{}, fmt.Errorf("parse RIF research governance envelope: %w", err)
	}
	var trailing any
	if err := decoder.Decode(&trailing); err != io.EOF {
		return ResearchGovernanceEnvelope{}, errors.New("RIF research governance envelope has trailing JSON data")
	}
	if err := VerifyResearchGovernanceEnvelope(envelope); err != nil {
		return ResearchGovernanceEnvelope{}, err
	}
	return envelope, nil
}

func VerifyResearchGovernanceEnvelope(envelope ResearchGovernanceEnvelope) error {
	if (envelope.SchemaVersion != ResearchGovernanceEnvelopeSchemaVersion && envelope.SchemaVersion != ResearchGovernanceEnvelopeSchemaVersionV2) || (envelope.Snapshot.SchemaVersion == ResearchGovernanceStoreSchemaVersion && envelope.SchemaVersion != ResearchGovernanceEnvelopeSchemaVersion) || (envelope.Snapshot.SchemaVersion == ResearchGovernanceStoreSchemaVersionV2 && envelope.SchemaVersion != ResearchGovernanceEnvelopeSchemaVersionV2) {
		return errors.New("unsupported RIF research governance envelope schema")
	}
	if err := verifyResearchSnapshot(envelope.Snapshot); err != nil {
		return err
	}
	if envelope.Authorization != nil {
		found := false
		for _, record := range envelope.Snapshot.Authorizations {
			if reflect.DeepEqual(record, *envelope.Authorization) {
				found = true
				break
			}
		}
		if !found {
			return errors.New("RIF envelope authorization is not in sealed snapshot")
		}
	}
	want, err := hashEnvelope(envelope)
	if err != nil {
		return err
	}
	if envelope.EnvelopeHash != want {
		return errors.New("RIF research governance envelope hash mismatch")
	}
	return nil
}

func verifyResearchSnapshot(snapshot ResearchGovernanceSnapshot) error {
	if (snapshot.SchemaVersion != ResearchGovernanceStoreSchemaVersion && snapshot.SchemaVersion != ResearchGovernanceStoreSchemaVersionV2) || len(snapshot.Identity) == 0 || !validSHA(snapshot.IdentityHash) || snapshot.State == "" || snapshot.Sequence == 0 {
		return errors.New("RIF research governance snapshot is incomplete")
	}
	previous := ""
	var sequence uint64
	for _, record := range snapshot.LifecycleHistory {
		if record.SchemaVersion != ResearchLifecycleRecordSchemaVersion || record.Sequence <= sequence || record.PreviousHash != previous || !validSHA(record.EvidenceSHA256) || !validSHA(record.PriorStateHash) {
			return errors.New("RIF lifecycle chain is invalid")
		}
		want, err := hashLifecycle(record)
		if err != nil || record.RecordHash != want {
			return errors.New("RIF lifecycle record hash mismatch")
		}
		previous, sequence = record.RecordHash, record.Sequence
	}
	if sequence != snapshot.Sequence || len(snapshot.LifecycleHistory) == 0 || snapshot.LifecycleHistory[len(snapshot.LifecycleHistory)-1].ToState != snapshot.State {
		return errors.New("RIF lifecycle state does not match history")
	}
	previous = ""
	for _, record := range snapshot.Authorizations {
		if record.SchemaVersion != PartitionAuthorizationSchemaVersion || record.PreviousHash != previous || !record.OneShot || !validSHA(record.ResearchIdentityHash) || !validSHA(record.PriorLifecycleStateHash) {
			return errors.New("RIF partition authorization chain is invalid")
		}
		want, err := hashAuthorization(record)
		if err != nil || record.RecordHash != want {
			return errors.New("RIF partition authorization hash mismatch")
		}
		previous = record.RecordHash
	}
	previous = ""
	for _, receipt := range snapshot.AccessReceipts {
		if receipt.SchemaVersion != PartitionAccessReceiptSchemaVersion || receipt.PreviousHash != previous {
			return errors.New("RIF partition access receipt chain is invalid")
		}
		want, err := hashAccess(receipt)
		if err != nil || receipt.RecordHash != want {
			return errors.New("RIF partition access receipt hash mismatch")
		}
		previous = receipt.RecordHash
	}
	if snapshot.Reservation == nil || snapshot.Reservation.SchemaVersion != HoldoutReservationSchemaVersion || snapshot.Reservation.ResearchIdentityHash != snapshot.IdentityHash || snapshot.Reservation.CandidateFrozen {
		return errors.New("RIF pre-development holdout reservation is missing or invalid")
	}
	wantReservation, err := hashReservation(*snapshot.Reservation)
	if err != nil || snapshot.Reservation.RecordHash != wantReservation {
		return errors.New("RIF holdout reservation hash mismatch")
	}
	if snapshot.FrozenCandidate != nil {
		wantFrozen, err := hashFrozen(*snapshot.FrozenCandidate)
		if err != nil || snapshot.FrozenCandidate.FrozenIdentityHash != wantFrozen {
			return errors.New("RIF frozen candidate hash mismatch")
		}
	}
	if err := verifyStageExecutionState(snapshot); err != nil {
		return err
	}
	wantState, err := hashSnapshot(snapshot)
	if err != nil || snapshot.IntegrityHash != wantState {
		return errors.New("RIF research governance snapshot hash mismatch")
	}
	return nil
}

func hashEnvelope(value ResearchGovernanceEnvelope) (string, error) {
	value.EnvelopeHash = ""
	return canonicalHash(value)
}
func hashSnapshot(value ResearchGovernanceSnapshot) (string, error) {
	value.IntegrityHash = ""
	return canonicalHash(value)
}
func hashLifecycle(value ResearchLifecycleRecord) (string, error) {
	value.RecordHash = ""
	return canonicalHash(value)
}
func hashAuthorization(value PartitionAuthorization) (string, error) {
	value.RecordHash = ""
	return canonicalHash(value)
}
func hashAccess(value PartitionAccessReceipt) (string, error) {
	value.RecordHash = ""
	return canonicalHash(value)
}
func hashReservation(value HoldoutReservation) (string, error) {
	value.RecordHash = ""
	return canonicalHash(value)
}
func hashFrozen(value FrozenResearchCandidate) (string, error) {
	value.FrozenIdentityHash = ""
	return canonicalHash(value)
}

func canonicalHash(value any) (string, error) {
	data, err := json.Marshal(value)
	if err != nil {
		return "", err
	}
	digest := sha256.Sum256(data)
	return "sha256:" + hex.EncodeToString(digest[:]), nil
}

func validSHA(value string) bool {
	if len(value) != 71 || !strings.HasPrefix(value, "sha256:") || value != strings.ToLower(value) {
		return false
	}
	_, err := hex.DecodeString(strings.TrimPrefix(value, "sha256:"))
	return err == nil
}
