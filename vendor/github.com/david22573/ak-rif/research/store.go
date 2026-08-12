package research

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"reflect"
	"syscall"
	"time"

	"github.com/david22573/ak-rif/persistence"
)

const maxStoreBytes = 32 << 20

type Authority struct {
	path string
	now  func() time.Time
}

// NewAuthority returns an unconfigured compatibility value whose operations
// fail closed. Use CreateAuthority or OpenAuthority for authoritative state.
func NewAuthority() *Authority { return &Authority{now: time.Now} }

func CreateAuthority(path string) (*Authority, error) {
	authority := &Authority{path: path, now: time.Now}
	lock, err := authority.acquireLock()
	if err != nil {
		return nil, err
	}
	defer releaseLock(lock)
	if _, err := os.Lstat(path); err == nil {
		return nil, errors.New("research governance store already exists")
	} else if !errors.Is(err, os.ErrNotExist) {
		return nil, fmt.Errorf("inspect research governance store: %w", err)
	}
	state := Snapshot{SchemaVersion: StoreSchemaVersion, Authorizations: []AuthorizationRecord{}, AccessReceipts: []AccessReceipt{}, LifecycleHistory: []LifecycleRecord{}}
	if err := sealSnapshot(&state); err != nil {
		return nil, err
	}
	if err := authority.write(state); err != nil {
		return nil, err
	}
	return authority, nil
}

func OpenAuthority(path string) (*Authority, error) {
	authority := &Authority{path: path, now: time.Now}
	if _, err := authority.read(); err != nil {
		return nil, err
	}
	return authority, nil
}

func (a *Authority) Snapshot() (Snapshot, error) {
	if a == nil || a.path == "" {
		return Snapshot{}, errors.New("research governance authority is not configured")
	}
	return a.read()
}

func (a *Authority) ExportEnvelope(authorizationID string) (Envelope, error) {
	state, err := a.Snapshot()
	if err != nil {
		return Envelope{}, err
	}
	envelopeVersion := EnvelopeSchemaVersion
	if state.SchemaVersion == StoreSchemaVersionV2 {
		envelopeVersion = EnvelopeSchemaVersionV2
	}
	envelope := Envelope{SchemaVersion: envelopeVersion, Snapshot: state}
	if authorizationID != "" {
		for i := range state.Authorizations {
			if state.Authorizations[i].AuthorizationID == authorizationID {
				record := state.Authorizations[i]
				envelope.Authorization = &record
				break
			}
		}
		if envelope.Authorization == nil {
			return Envelope{}, errors.New("partition authorization was not found")
		}
	}
	hash, err := hashEnvelope(envelope)
	if err != nil {
		return Envelope{}, err
	}
	envelope.EnvelopeHash = hash
	return envelope, nil
}

func VerifyEnvelope(envelope Envelope) error {
	if (envelope.SchemaVersion != EnvelopeSchemaVersion && envelope.SchemaVersion != EnvelopeSchemaVersionV2) || (envelope.Snapshot.SchemaVersion == StoreSchemaVersion && envelope.SchemaVersion != EnvelopeSchemaVersion) || (envelope.Snapshot.SchemaVersion == StoreSchemaVersionV2 && envelope.SchemaVersion != EnvelopeSchemaVersionV2) {
		return errors.New("unsupported research governance envelope schema")
	}
	if err := validateSnapshot(envelope.Snapshot); err != nil {
		return err
	}
	if envelope.Authorization != nil {
		found := false
		for _, record := range envelope.Snapshot.Authorizations {
			if record.AuthorizationID == envelope.Authorization.AuthorizationID && record.RecordHash == envelope.Authorization.RecordHash {
				found = true
				break
			}
		}
		if !found {
			return errors.New("envelope authorization is not in the sealed snapshot")
		}
	}
	want, err := hashEnvelope(envelope)
	if err != nil {
		return err
	}
	if envelope.EnvelopeHash != want {
		return errors.New("research governance envelope hash mismatch")
	}
	return nil
}

func hashEnvelope(envelope Envelope) (string, error) {
	copyEnvelope := envelope
	copyEnvelope.EnvelopeHash = ""
	return hashCanonical(copyEnvelope)
}

func (a *Authority) mutate(change func(*Snapshot) error) error {
	lock, err := a.acquireLock()
	if err != nil {
		return err
	}
	defer releaseLock(lock)
	state, err := a.read()
	if err != nil {
		return err
	}
	if err := change(&state); err != nil {
		return err
	}
	if err := sealSnapshot(&state); err != nil {
		return err
	}
	return a.write(state)
}

func (a *Authority) read() (Snapshot, error) {
	if a == nil || a.path == "" {
		return Snapshot{}, errors.New("research governance authority is not configured")
	}
	file, err := persistence.OpenRegularFile(a.path, "read_research_governance_store")
	if err != nil {
		return Snapshot{}, err
	}
	defer file.Close()
	info, err := file.Stat()
	if err != nil {
		return Snapshot{}, err
	}
	if info.Size() > maxStoreBytes {
		return Snapshot{}, errors.New("research governance store exceeds size limit")
	}
	decoder := json.NewDecoder(io.LimitReader(file, maxStoreBytes+1))
	decoder.DisallowUnknownFields()
	var state Snapshot
	if err := decoder.Decode(&state); err != nil {
		return Snapshot{}, fmt.Errorf("decode research governance store: %w", err)
	}
	var trailing any
	if err := decoder.Decode(&trailing); err != io.EOF {
		return Snapshot{}, errors.New("research governance store has trailing data")
	}
	if err := validateSnapshot(state); err != nil {
		return Snapshot{}, err
	}
	return state, nil
}

func (a *Authority) write(state Snapshot) error {
	data, err := json.MarshalIndent(state, "", "  ")
	if err != nil {
		return err
	}
	if len(data)+1 > maxStoreBytes {
		return errors.New("research governance store exceeds size limit")
	}
	return persistence.WriteFileAtomic(a.path, append(data, '\n'), 0o600)
}

func (a *Authority) acquireLock() (*os.File, error) {
	if a == nil || a.path == "" {
		return nil, errors.New("research governance authority is not configured")
	}
	lockPath := a.path + ".lock"
	fd, err := syscall.Open(lockPath, syscall.O_CREAT|syscall.O_RDWR|syscall.O_CLOEXEC|syscall.O_NOFOLLOW, 0o600)
	if err != nil {
		return nil, fmt.Errorf("open research governance lock: %w", err)
	}
	file := os.NewFile(uintptr(fd), lockPath)
	if err := syscall.Flock(fd, syscall.LOCK_EX); err != nil {
		_ = file.Close()
		return nil, fmt.Errorf("lock research governance store: %w", err)
	}
	return file, nil
}

func releaseLock(file *os.File) {
	if file != nil {
		_ = syscall.Flock(int(file.Fd()), syscall.LOCK_UN)
		_ = file.Close()
	}
}

func sealSnapshot(state *Snapshot) error {
	copyState := *state
	copyState.IntegrityHash = ""
	hash, err := hashCanonical(copyState)
	if err != nil {
		return err
	}
	state.IntegrityHash = hash
	return nil
}

func validateSnapshot(state Snapshot) error {
	if err := validateStageExecutionState(state); err != nil {
		return err
	}
	if state.Identity == nil {
		if state.IdentityHash != "" || state.State != "" || state.Sequence != 0 || state.Reservation != nil || state.FrozenCandidate != nil || state.Disposition != nil || len(state.Authorizations) != 0 || len(state.AccessReceipts) != 0 || len(state.StageExecutionSets) != 0 || state.DevelopmentNominee != nil || len(state.LifecycleHistory) != 0 {
			return errors.New("unregistered research governance store contains state")
		}
	} else {
		canonical, err := CanonicalizeIdentity(*state.Identity)
		if err != nil {
			return fmt.Errorf("research identity: %w", err)
		}
		if !reflect.DeepEqual(canonical, *state.Identity) {
			return errors.New("stored research identity is not canonical")
		}
		hash, _ := HashIdentityV4(canonical)
		if state.IdentityHash != hash {
			return errors.New("stored research identity hash mismatch")
		}
		if state.State == "" || state.Sequence == 0 || len(state.LifecycleHistory) == 0 {
			return errors.New("registered research state is incomplete")
		}
	}
	previous := ""
	var lastSequence uint64
	for _, record := range state.LifecycleHistory {
		if record.SchemaVersion != LifecycleRecordVersion || record.Sequence <= lastSequence || record.PreviousHash != previous || !validSHA256(record.EvidenceSHA256) || !validSHA256(record.PriorStateHash) {
			return errors.New("lifecycle history sequence or hash chain is invalid")
		}
		want, err := hashLifecycleRecord(record)
		if err != nil || record.RecordHash != want {
			return errors.New("lifecycle history record hash mismatch")
		}
		previous, lastSequence = record.RecordHash, record.Sequence
	}
	if len(state.LifecycleHistory) > 0 && (lastSequence != state.Sequence || state.LifecycleHistory[len(state.LifecycleHistory)-1].ToState != state.State) {
		return errors.New("lifecycle history does not match current state")
	}
	previous = ""
	for _, record := range state.Authorizations {
		if record.PreviousHash != previous {
			return errors.New("authorization hash chain is invalid")
		}
		if err := VerifyAuthorizationRecord(record); err != nil {
			return err
		}
		previous = record.RecordHash
	}
	previous = ""
	for _, receipt := range state.AccessReceipts {
		if receipt.PreviousHash != previous {
			return errors.New("access receipt hash chain is invalid")
		}
		want, err := hashAccessReceipt(receipt)
		if err != nil || receipt.RecordHash != want {
			return errors.New("access receipt hash mismatch")
		}
		previous = receipt.RecordHash
	}
	if state.Reservation != nil {
		want, err := hashReservation(*state.Reservation)
		if err != nil || state.Reservation.RecordHash != want || state.Reservation.ResearchIdentityHash != state.IdentityHash || state.Reservation.CandidateFrozen {
			return errors.New("holdout reservation integrity mismatch")
		}
	}
	if state.FrozenCandidate != nil {
		want, err := hashFrozenCandidate(*state.FrozenCandidate)
		if err != nil || state.FrozenCandidate.FrozenIdentityHash != want {
			return errors.New("frozen candidate identity hash mismatch")
		}
	}
	copyState := state
	copyState.IntegrityHash = ""
	want, err := hashCanonical(copyState)
	if err != nil || state.IntegrityHash != want {
		return errors.New("research governance store integrity hash mismatch")
	}
	return nil
}

func VerifyAuthorizationRecord(record AuthorizationRecord) error {
	if record.SchemaVersion != AuthorizationSchemaVersion || !validSHA256(record.ResearchIdentityHash) || !validSHA256(record.PriorLifecycleStateHash) || !record.OneShot || record.AuthorizationID == "" || record.Sequence == 0 || record.IssuedAt.IsZero() {
		return errors.New("partition authorization is incomplete")
	}
	want, err := hashAuthorization(record)
	if err != nil || record.RecordHash != want {
		return errors.New("partition authorization record hash mismatch")
	}
	return nil
}

func hashLifecycleRecord(record LifecycleRecord) (string, error) {
	record.RecordHash = ""
	return hashCanonical(record)
}
func hashAuthorization(record AuthorizationRecord) (string, error) {
	record.RecordHash = ""
	return hashCanonical(record)
}
func hashAccessReceipt(record AccessReceipt) (string, error) {
	record.RecordHash = ""
	return hashCanonical(record)
}
func hashReservation(record ReservationRecord) (string, error) {
	record.RecordHash = ""
	return hashCanonical(record)
}
func hashFrozenCandidate(record FrozenCandidate) (string, error) {
	record.FrozenIdentityHash = ""
	return hashCanonical(record)
}

func strictDecode(data []byte, value any) error {
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(value); err != nil {
		return err
	}
	var extra any
	if err := decoder.Decode(&extra); err != io.EOF {
		return errors.New("trailing JSON data")
	}
	return nil
}
