package qualificationrunner

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"strings"
	"testing"
	"time"

	"github.com/david22573/ak-engine/internal/preconditions"
	"github.com/david22573/ak-engine/internal/qualification"
	"github.com/david22573/ak-engine/internal/rifbridge"
)

func TestPreexecutionRunnerIdentityIsDeterministicAndDataIndependent(t *testing.T) {
	input := RunnerPrebuildInput{SourceCommit: strings.Repeat("5", 40), PackageID: "ak.engine.qualificationrunner", CanonicalPackage: []byte("synthetic canonical package"), DeterministicBuildInputs: []rifbridge.StageHashIdentity{{ID: "go.mod", SHA256: testHash('1')}, {ID: "go.sum", SHA256: testHash('2')}}, CompilerIdentity: "go1.synthetic linux/amd64", BuildModeID: "trimpath-buildvcs-false", CanonicalBuildMode: []byte("-trimpath -buildvcs=false"), Binary: []byte("synthetic deterministic runner binary")}
	first, firstReceipt, err := ComputePreexecutionRunnerIdentity(input)
	if err != nil {
		t.Fatal(err)
	}
	second, secondReceipt, err := ComputePreexecutionRunnerIdentity(input)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(first, second) || firstReceipt.ReceiptHash != secondReceipt.ReceiptHash {
		t.Fatal("pre-execution runner identity is not deterministic")
	}
	if firstReceipt.DataLoads != 0 || firstReceipt.CandidateEvents != 0 || firstReceipt.CandidateOutcomes != 0 || !containsString(firstReceipt.Labels, NoOutcomesLabel) || !containsString(firstReceipt.Labels, NoRealPartitionAccessLabel) {
		t.Fatalf("prebuild produced candidate evidence: %#v", firstReceipt)
	}
	mutated := input
	mutated.Binary = []byte("semantic build change")
	changed, _, _ := ComputePreexecutionRunnerIdentity(mutated)
	if changed.BinarySHA256 == first.BinarySHA256 {
		t.Fatal("post-build semantic change retained the authorized binary identity")
	}
}

func TestStageExecutionAPIHasNoCallerResultOrExecutorBypass(t *testing.T) {
	batchType := reflect.TypeOf((*StageBatchRunner).ExecuteStage)
	variantType := reflect.TypeOf((*StageBatchRunner).ExecuteAuthorizedVariant)
	if batchType.NumIn() != 3 || batchType.In(1) != reflect.TypeOf([]rifbridge.StageExecutionEnvelope{}) || batchType.In(2) != reflect.TypeOf([][]byte{}) {
		t.Fatalf("stage batch API gained a caller execution or result seam: %s", batchType)
	}
	if variantType.NumIn() != 3 || variantType.In(1) != reflect.TypeOf(rifbridge.StageExecutionEnvelope{}) || variantType.In(2) != reflect.TypeOf([]byte{}) {
		t.Fatalf("stage variant API gained a caller execution or result seam: %s", variantType)
	}
}

func TestStagePlanDryVerificationAndSubstitutionRejection(t *testing.T) {
	envelopes, runner := developmentStageEnvelopes(t)
	identity, readiness, err := VerifyStageExecutionPlan(envelopes[0], runner)
	if err != nil {
		t.Fatal(err)
	}
	if readiness.AuthorizedVariants != 4 || readiness.Stage != "DEVELOPMENT" || !readiness.ValidationSubsetDerivable || readiness.FinalHoldoutMode != "SINGLE_FROZEN_CANDIDATE_ONLY" || readiness.DataLoads != 0 || readiness.CandidateOutcomes != 0 {
		t.Fatalf("dry stage verification is incomplete: %#v", readiness)
	}
	for _, label := range []string{NoOutcomesLabel, NoRealPartitionAccessLabel, PreexecutionCycleAbsentLabel, MultivariantAuthorizationLabel} {
		if !containsString(readiness.Labels, label) {
			t.Fatalf("dry verification omitted %s", label)
		}
	}
	if _, err := EncodeStageReadinessJSON(readiness); err != nil {
		t.Fatalf("canonical readiness artifact did not encode: %v", err)
	}
	if identity.Repositories.RunnerExecutableSHA256 != runner.BinarySHA256 || strings.Contains(string(mustJSON(t, envelopes[0].ExecutionSet.Plan)), "result_artifact_sha256") {
		t.Fatal("stage authorization depends on a future result artifact or wrong runner")
	}
	wrongRunner := runner
	wrongRunner.BinarySHA256 = testHash('0')
	if _, _, err := VerifyStageExecutionPlan(envelopes[0], wrongRunner); err == nil {
		t.Fatal("another executable build used the stage authorization")
	}
	mutated := cloneStageEnvelope(envelopes[0])
	mutated.ExecutionSet.Plan.Configurations[0].ConfigurationSHA256 = testHash('0')
	if _, _, err := VerifyStageExecutionPlan(mutated, runner); err == nil {
		t.Fatal("mutated canonical configuration verified")
	}
	mutated = cloneStageEnvelope(envelopes[0])
	mutated.ExecutionSet.Plan.Configurations[0], mutated.ExecutionSet.Plan.Configurations[1] = mutated.ExecutionSet.Plan.Configurations[1], mutated.ExecutionSet.Plan.Configurations[0]
	if _, _, err := VerifyStageExecutionPlan(mutated, runner); err == nil {
		t.Fatal("reordered stage plan verified")
	}
}

func TestValidationPlanIsExactlySealedNomineeAndNeighbors(t *testing.T) {
	valid, runner := validationStageEnvelope(t, []string{"V00", "V01", "V02"})
	_, readiness, err := VerifyStageExecutionPlan(valid, runner)
	if err != nil {
		t.Fatal(err)
	}
	if readiness.Stage != "VALIDATION" || readiness.AuthorizedVariants != 3 || readiness.FinalHoldoutMode != "SINGLE_FROZEN_CANDIDATE_ONLY" {
		t.Fatalf("VALIDATION readiness mismatch: %#v", readiness)
	}
	for name, ids := range map[string][]string{
		"missing neighbor":  {"V00", "V01"},
		"unrelated variant": {"V00", "V01", "V03"},
		"reordered":         {"V01", "V00", "V02"},
	} {
		t.Run(name, func(t *testing.T) {
			envelope, localRunner := validationStageEnvelope(t, ids)
			if _, _, err := VerifyStageExecutionPlan(envelope, localRunner); err == nil {
				t.Fatal("invalid deterministic VALIDATION subset verified")
			}
		})
	}
	if valid.Snapshot.DevelopmentNominee.VariantID != "V00" {
		t.Fatal("VALIDATION plan replaced the sealed DEVELOPMENT nominee")
	}
}

func TestBatchExecutesCompleteLedgerAndSealsCanonicalManifest(t *testing.T) {
	envelopes, runnerIdentity := developmentStageEnvelopes(t)
	runner, err := NewStageBatchRunner(filepath.Join(t.TempDir(), "progress.json"), runnerIdentity)
	if err != nil {
		t.Fatal(err)
	}
	artifacts := stageArtifacts(t, envelopes)
	results, receipts, manifest, err := runner.ExecuteStage(envelopes, artifacts)
	if err != nil {
		t.Fatal(err)
	}
	if len(results) != 4 || len(receipts) != 4 || len(manifest.OrderedVariantIDs) != 4 || !reflect.DeepEqual(manifest.OrderedVariantIDs, []string{"V00", "V01", "V02", "V03"}) {
		t.Fatalf("complete deterministic batch did not execute and seal: manifest=%#v", manifest)
	}
	for index, result := range results {
		if result.AuthorizationID != envelopes[index].Authorization.AuthorizationID || result.Configuration.ConfigurationSHA256 != envelopes[index].Authorization.Configuration.ConfigurationSHA256 || result.Runner != runnerIdentity || result.Partition.Name != "DEVELOPMENT" || result.Checkpoint.SHA256 != envelopes[index].ExecutionSet.Plan.Checkpoint.SHA256 || !validNonPlaceholderHash(result.ResultArtifactSHA256) || !validNonPlaceholderHash(result.EnvelopeHash) {
			t.Fatalf("result %d lost a sealing identity: %#v", index, result)
		}
		if _, err := EncodeStageResultEnvelopeJSON(result); err != nil {
			t.Fatalf("stage result %d did not encode canonically: %v", index, err)
		}
		var artifact ResultArtifact
		if err := json.Unmarshal(result.ResultArtifact, &artifact); err != nil || !artifact.Independence.Invoked || !artifact.Uncertainty.Invoked || !artifact.Concentration.Invoked {
			t.Fatalf("stage result %d did not come from the governed qualification implementation: %v %#v", index, err, artifact)
		}
	}
	if _, err := EncodeStageBatchManifestJSON(manifest); err != nil {
		t.Fatalf("stage batch manifest did not encode canonically: %v", err)
	}
	reopened, err := NewStageBatchRunner(runner.progressPath, runnerIdentity)
	if err != nil {
		t.Fatal(err)
	}
	progress, err := reopened.Progress()
	if err != nil || progress.State != "SEALED" || progress.Manifest.ManifestHash != manifest.ManifestHash {
		t.Fatalf("sealed batch progress did not recover: %v %#v", err, progress)
	}
	if _, _, err := reopened.ExecuteAuthorizedVariant(envelopes[0], artifacts[0]); err == nil {
		t.Fatal("completed or sealed variant reran")
	}
}

func TestBatchRejectsMissingDuplicateReorderedAndIncompleteSets(t *testing.T) {
	envelopes, runnerIdentity := developmentStageEnvelopes(t)
	for name, candidate := range map[string][]rifbridge.StageExecutionEnvelope{
		"missing":    envelopes[:3],
		"additional": append(append([]rifbridge.StageExecutionEnvelope(nil), envelopes...), envelopes[3]),
		"duplicate":  {envelopes[0], envelopes[0], envelopes[2], envelopes[3]},
		"reordered":  {envelopes[1], envelopes[0], envelopes[2], envelopes[3]},
	} {
		t.Run(name, func(t *testing.T) {
			runner, _ := NewStageBatchRunner(filepath.Join(t.TempDir(), "progress.json"), runnerIdentity)
			if _, _, _, err := runner.ExecuteStage(candidate, stageArtifacts(t, candidate)); err == nil {
				t.Fatal("invalid batch envelope set executed")
			}
		})
	}
	runner, _ := NewStageBatchRunner(filepath.Join(t.TempDir(), "progress.json"), runnerIdentity)
	if _, _, err := runner.ExecuteAuthorizedVariant(envelopes[0], stageArtifact(t, envelopes[0])); err != nil {
		t.Fatal(err)
	}
	if _, err := runner.Seal(envelopes[0]); err == nil {
		t.Fatal("incomplete stage execution set sealed")
	}
}

func TestBatchCrashRestartAndTamperEvidence(t *testing.T) {
	envelopes, runnerIdentity := developmentStageEnvelopes(t)
	path := filepath.Join(t.TempDir(), "progress.json")
	runner, _ := NewStageBatchRunner(path, runnerIdentity)
	if _, _, err := runner.ExecuteAuthorizedVariant(envelopes[0], []byte("{invalid synthetic artifact")); err == nil {
		t.Fatal("invalid post-authorization execution did not fail")
	}
	restarted, _ := NewStageBatchRunner(path, runnerIdentity)
	if _, _, err := restarted.ExecuteAuthorizedVariant(envelopes[0], stageArtifact(t, envelopes[0])); err == nil {
		t.Fatal("indeterminate row-authorized attempt reran after restart")
	}
	retryEnvelope := stageRetryEnvelope(t, envelopes[0])
	if err := restarted.ResumeWithZeroAccessRetry(retryEnvelope); err != nil {
		t.Fatalf("durably proven zero-access attempt could not resume: %v", err)
	}
	if _, _, err := restarted.ExecuteAuthorizedVariant(retryEnvelope, stageArtifact(t, retryEnvelope)); err != nil {
		t.Fatalf("authorized zero-access retry did not execute: %v", err)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	tampered := strings.Replace(string(data), `"next_ordinal": 1`, `"next_ordinal": 2`, 1)
	if err := os.WriteFile(path, []byte(tampered), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := restarted.Progress(); err == nil {
		t.Fatal("tampered filesystem progress was accepted as completion proof")
	}

	resumePath := filepath.Join(t.TempDir(), "resume.json")
	first, _ := NewStageBatchRunner(resumePath, runnerIdentity)
	if _, _, err := first.ExecuteAuthorizedVariant(envelopes[0], stageArtifact(t, envelopes[0])); err != nil {
		t.Fatal(err)
	}
	second, _ := NewStageBatchRunner(resumePath, runnerIdentity)
	if _, _, err := second.ExecuteAuthorizedVariant(envelopes[1], stageArtifact(t, envelopes[1])); err != nil {
		t.Fatalf("incomplete stage did not resume at next variant: %v", err)
	}
	progress, _ := second.Progress()
	if progress.NextOrdinal != 2 || len(progress.Receipts) != 2 {
		t.Fatal("durable restart lost completed receipt-bearing variants")
	}
}

func developmentStageEnvelopes(t *testing.T) ([]rifbridge.StageExecutionEnvelope, rifbridge.RunnerImplementationIdentity) {
	t.Helper()
	identity, ledger, runner := stageIdentity(t)
	identityBytes, _ := json.Marshal(identity)
	identityHash := hashBytes(identityBytes)
	reservation, history := stageReservationHistory(t, identity, identityHash)
	base := rifbridge.ResearchGovernanceSnapshot{SchemaVersion: rifbridge.ResearchGovernanceStoreSchemaVersion, Identity: identityBytes, IdentityHash: identityHash, Reservation: reservation, State: "HOLDOUT_RESERVED", Sequence: 2, Authorizations: []rifbridge.PartitionAuthorization{}, AccessReceipts: []rifbridge.PartitionAccessReceipt{}, LifecycleHistory: history}
	base.IntegrityHash = mustLocalHash(t, base)
	plan := stagePlan(t, identity, ledger, runner, identityHash, "DEVELOPMENT", []string{"V00", "V01", "V02", "V03"})
	set := issueStageSet(t, plan, base.IntegrityHash)
	stageEvent := rifbridge.ResearchLifecycleRecord{SchemaVersion: rifbridge.ResearchLifecycleRecordSchemaVersion, Sequence: 3, EventType: "DEVELOPMENT_SET_AUTHORIZED", FromState: "HOLDOUT_RESERVED", ToState: "DEVELOPMENT_SET_AUTHORIZED", OccurredAt: set.IssuedAt, EvidenceSHA256: set.RecordHash, PriorStateHash: base.IntegrityHash, PreviousHash: history[len(history)-1].RecordHash}
	stageEvent.RecordHash = mustLocalHash(t, stageEvent)
	snapshot := base
	snapshot.SchemaVersion = rifbridge.ResearchGovernanceStoreSchemaVersionV2
	snapshot.State, snapshot.Sequence = "DEVELOPMENT_SET_AUTHORIZED", 3
	snapshot.StageExecutionSets = []rifbridge.StageExecutionSet{set}
	snapshot.LifecycleHistory = append(snapshot.LifecycleHistory, stageEvent)
	snapshot.IntegrityHash = ""
	snapshot.IntegrityHash = mustLocalHash(t, snapshot)
	envelopes := make([]rifbridge.StageExecutionEnvelope, len(set.Authorizations))
	for ordinal := range set.Authorizations {
		authorization := set.Authorizations[ordinal]
		previous := ""
		if len(set.AccessReceipts) > 0 {
			previous = set.AccessReceipts[len(set.AccessReceipts)-1].RecordHash
		}
		receipt := rifbridge.StageVariantAccessReceipt{SchemaVersion: rifbridge.StageAccessReceiptSchemaVersion, ExecutionSetID: set.ExecutionSetID, AuthorizationID: authorization.AuthorizationID, VariantID: authorization.Configuration.VariantID, Attempt: authorization.Attempt, ConsumedAt: time.Date(2029, 1, 1, 0, 1, ordinal, 0, time.UTC), PriorStateHash: snapshot.IntegrityHash, PreviousHash: previous}
		receipt.RecordHash = mustLocalHash(t, receipt)
		set.AccessReceipts = append(set.AccessReceipts, receipt)
		set.RecordHash = ""
		set.RecordHash = mustLocalHash(t, set)
		event := rifbridge.ResearchLifecycleRecord{SchemaVersion: rifbridge.ResearchLifecycleRecordSchemaVersion, Sequence: snapshot.Sequence + 1, EventType: "DEVELOPMENT_VARIANT_ACCESS_CONSUMED", FromState: snapshot.State, ToState: snapshot.State, OccurredAt: receipt.ConsumedAt, EvidenceSHA256: receipt.RecordHash, PriorStateHash: snapshot.IntegrityHash, PreviousHash: snapshot.LifecycleHistory[len(snapshot.LifecycleHistory)-1].RecordHash}
		event.RecordHash = mustLocalHash(t, event)
		snapshot.Sequence++
		snapshot.LifecycleHistory = append(snapshot.LifecycleHistory, event)
		snapshot.StageExecutionSets[0] = set
		snapshot.IntegrityHash = ""
		snapshot.IntegrityHash = mustLocalHash(t, snapshot)
		authorizationCopy := authorization
		envelope := rifbridge.StageExecutionEnvelope{SchemaVersion: rifbridge.StageEnvelopeSchemaVersion, Snapshot: snapshot, ExecutionSet: set, Authorization: &authorizationCopy}
		envelope.EnvelopeHash = mustLocalHash(t, envelope)
		if err := rifbridge.VerifyStageExecutionEnvelope(envelope); err != nil {
			t.Fatalf("synthetic RIF stage envelope %d invalid: %v", ordinal, err)
		}
		envelopes[ordinal] = cloneStageEnvelope(envelope)
	}
	return envelopes, runner
}

func validationStageEnvelope(t *testing.T, ids []string) (rifbridge.StageExecutionEnvelope, rifbridge.RunnerImplementationIdentity) {
	t.Helper()
	identity, ledger, runner := stageIdentity(t)
	identityBytes, _ := json.Marshal(identity)
	identityHash := hashBytes(identityBytes)
	reservation, history := stageReservationHistory(t, identity, identityHash)
	base := rifbridge.ResearchGovernanceSnapshot{SchemaVersion: rifbridge.ResearchGovernanceStoreSchemaVersion, Identity: identityBytes, IdentityHash: identityHash, Reservation: reservation, State: "HOLDOUT_RESERVED", Sequence: 2, Authorizations: []rifbridge.PartitionAuthorization{}, AccessReceipts: []rifbridge.PartitionAccessReceipt{}, LifecycleHistory: history}
	base.IntegrityHash = mustLocalHash(t, base)
	development := issueStageSet(t, stagePlan(t, identity, ledger, runner, identityHash, "DEVELOPMENT", []string{"V00", "V01", "V02", "V03"}), base.IntegrityHash)
	previousAccess, previousExecution := "", ""
	for ordinal, authorization := range development.Authorizations {
		access := rifbridge.StageVariantAccessReceipt{SchemaVersion: rifbridge.StageAccessReceiptSchemaVersion, ExecutionSetID: development.ExecutionSetID, AuthorizationID: authorization.AuthorizationID, VariantID: authorization.Configuration.VariantID, Attempt: 1, ConsumedAt: time.Date(2029, 1, 2, 0, 0, ordinal, 0, time.UTC), PriorStateHash: base.IntegrityHash, PreviousHash: previousAccess}
		access.RecordHash = mustLocalHash(t, access)
		development.AccessReceipts = append(development.AccessReceipts, access)
		previousAccess = access.RecordHash
		runID, _ := deterministicStageRunID(development.ExecutionSetID, authorization.AuthorizationID, authorization.Configuration)
		receipt := rifbridge.StageExecutionReceipt{SchemaVersion: rifbridge.StageExecutionReceiptSchemaVersion, ExecutionSetID: development.ExecutionSetID, PlanHash: development.Plan.PlanHash, AuthorizationID: authorization.AuthorizationID, DeterministicRunID: runID, VariantID: authorization.Configuration.VariantID, ConfigurationSHA256: authorization.Configuration.ConfigurationSHA256, RunnerIdentitySHA256: testHash('1'), Partition: "DEVELOPMENT", CheckpointSHA256: development.Plan.Checkpoint.SHA256, AccessReceiptHash: access.RecordHash, ResultArtifactSHA256: testHash(byte('2' + ordinal)), OutputManifestSHA256: testHash('7'), AuthorityEvidenceSHA256: testHash('8'), ResultStatus: "COMPLETED", MandatoryGatesPassed: true, CompletedAt: time.Date(2029, 1, 2, 0, 1, ordinal, 0, time.UTC), PreviousHash: previousExecution}
		receipt.RecordHash = mustLocalHash(t, receipt)
		development.ExecutionReceipts = append(development.ExecutionReceipts, receipt)
		previousExecution = receipt.RecordHash
	}
	manifest := rifbridge.StageCompletionManifest{SchemaVersion: rifbridge.StageManifestSchemaVersion, ExecutionSetID: development.ExecutionSetID, PlanHash: development.Plan.PlanHash, Stage: "DEVELOPMENT"}
	for index, configuration := range development.Plan.Configurations {
		manifest.OrderedVariantIDs = append(manifest.OrderedVariantIDs, configuration.VariantID)
		manifest.OrderedReceiptHashes = append(manifest.OrderedReceiptHashes, development.ExecutionReceipts[index].RecordHash)
		manifest.OrderedResultHashes = append(manifest.OrderedResultHashes, development.ExecutionReceipts[index].ResultArtifactSHA256)
	}
	manifest.ManifestHash = mustLocalHash(t, manifest)
	sealedAt := time.Date(2029, 1, 2, 1, 0, 0, 0, time.UTC)
	development.CompletionState, development.CompletionManifest, development.SealedAt = "SEALED", &manifest, &sealedAt
	sealIdentity := struct {
		ExecutionSetID string `json:"execution_set_id"`
		PlanHash       string `json:"plan_hash"`
		ManifestHash   string `json:"manifest_hash"`
	}{development.ExecutionSetID, development.Plan.PlanHash, manifest.ManifestHash}
	development.FinalStageSeal = mustLocalHash(t, sealIdentity)
	development.RecordHash = ""
	development.RecordHash = mustLocalHash(t, development)
	nominee := &rifbridge.DevelopmentNominee{SchemaVersion: rifbridge.NomineeSelectionSchemaVersion, DevelopmentSetID: development.ExecutionSetID, Rule: "LOWEST_NUMERIC_VARIANT_ID_PASSING_ALL_MANDATORY_DEVELOPMENT_GATES", Exists: true, VariantID: "V00", ConfigurationSHA256: development.Plan.Configurations[0].ConfigurationSHA256, SelectedAt: time.Date(2029, 1, 2, 1, 1, 0, 0, time.UTC)}
	nominee.RecordHash = mustLocalHash(t, *nominee)
	validationPlan := stagePlan(t, identity, ledger, runner, identityHash, "VALIDATION", ids)
	validation := issueStageSet(t, validationPlan, base.IntegrityHash)
	event := rifbridge.ResearchLifecycleRecord{SchemaVersion: rifbridge.ResearchLifecycleRecordSchemaVersion, Sequence: 3, EventType: "VALIDATION_SET_AUTHORIZED", FromState: "DEVELOPMENT_SET_SEALED", ToState: "VALIDATION_SET_AUTHORIZED", OccurredAt: validation.IssuedAt, EvidenceSHA256: validation.RecordHash, PriorStateHash: base.IntegrityHash, PreviousHash: history[len(history)-1].RecordHash}
	event.RecordHash = mustLocalHash(t, event)
	snapshot := base
	snapshot.SchemaVersion, snapshot.State, snapshot.Sequence = rifbridge.ResearchGovernanceStoreSchemaVersionV2, "VALIDATION_SET_AUTHORIZED", 3
	snapshot.StageExecutionSets = []rifbridge.StageExecutionSet{development, validation}
	snapshot.DevelopmentNominee = nominee
	snapshot.LifecycleHistory = append(snapshot.LifecycleHistory, event)
	snapshot.IntegrityHash = ""
	snapshot.IntegrityHash = mustLocalHash(t, snapshot)
	envelope := rifbridge.StageExecutionEnvelope{SchemaVersion: rifbridge.StageEnvelopeSchemaVersion, Snapshot: snapshot, ExecutionSet: validation}
	envelope.EnvelopeHash = mustLocalHash(t, envelope)
	if err := rifbridge.VerifyStageExecutionEnvelope(envelope); err != nil {
		t.Fatalf("synthetic VALIDATION envelope invalid: %v", err)
	}
	return envelope, runner
}

func stageIdentity(t *testing.T) (ResearchIdentityV4, VariantLedger, rifbridge.RunnerImplementationIdentity) {
	t.Helper()
	v00 := V00Configuration()
	v01 := V00Configuration()
	v01.ContextAgreement = "REQUIRE_POSITIVE_BTC_ETH_CONTEXT"
	v02 := V00Configuration()
	v02.EventQuality = "STRICT_CENTER_VOLATILITY"
	v03 := V00Configuration()
	v03.CooldownMinutes = 60
	ledger, err := SealVariantLedger(VariantLedger{SchemaVersion: VariantLedgerVersion, MaximumVariants: 4, V00ID: "V00", Variants: []RegisteredVariant{{"V00", []string{}, v00, ""}, {"V01", []string{"context-agreement"}, v01, ""}, {"V02", []string{"event-quality"}, v02, ""}, {"V03", []string{"cooldown/independence"}, v03, ""}}, StabilityNeighborhoods: []StabilityNeighborhood{{"V00", []string{"V01", "V02"}}, {"V01", []string{"V00", "V02"}}, {"V02", []string{"V00", "V01"}}, {"V03", []string{"V01", "V02"}}}})
	if err != nil {
		t.Fatal(err)
	}
	gates := qualification.AcceptedPR4B0GateSet()
	gateHash, _ := qualification.PR4B0GateSetHash(gates)
	gateRefs, _ := qualification.PR4B0GateIdentities(gates)
	gateIDs := make([]HashIdentity, len(gateRefs))
	for i, item := range gateRefs {
		gateIDs[i] = HashIdentity{item.ArtifactID, item.SHA256}
	}
	sort.Slice(gateIDs, func(i, j int) bool { return gateIDs[i].ID < gateIDs[j].ID })
	policy := preconditions.AcceptedIndependencePolicyV3Default()
	independenceHash, _ := preconditions.AcceptedIndependencePolicyHashV3(policy)
	uncertaintyHash, _ := preconditions.AcceptedUncertaintyMethodHashV2(preconditions.AcceptedUncertaintyMethodV2())
	universe, err := V00UniverseContract()
	if err != nil {
		t.Fatal(err)
	}
	runner := rifbridge.RunnerImplementationIdentity{SchemaVersion: rifbridge.RunnerImplementationSchemaVersion, SourceCommit: strings.Repeat("5", 40), PackageIdentity: rifbridge.StageHashIdentity{ID: "ak.engine.qualificationrunner", SHA256: testHash('a')}, BuildInputsSHA256: testHash('b'), CompilerIdentity: "go1.synthetic linux/amd64", BuildModeIdentity: rifbridge.StageHashIdentity{ID: "trimpath-buildvcs-false", SHA256: testHash('c')}, BinarySHA256: testHash('1')}
	partitions := []Partition{{"DEVELOPMENT", Interval{time.Date(2030, 1, 1, 0, 0, 0, 0, time.UTC), time.Date(2031, 1, 1, 0, 0, 0, 0, time.UTC)}, 365, testHash('d')}, {"FINAL_HOLDOUT", Interval{time.Date(2032, 1, 1, 0, 0, 0, 0, time.UTC), time.Date(2033, 1, 1, 0, 0, 0, 0, time.UTC)}, 366, testHash('e')}, {"VALIDATION", Interval{time.Date(2031, 1, 1, 0, 0, 0, 0, time.UTC), time.Date(2032, 1, 1, 0, 0, 0, 0, time.UTC)}, 365, testHash('f')}}
	identity := ResearchIdentityV4{"ak.rif.research_identity.v4", "synthetic-stage-runner", RepositoryIdentity{strings.Repeat("1", 40), strings.Repeat("2", 40), strings.Repeat("3", 40), strings.Repeat("4", 40), runner.SourceCommit, runner.BinarySHA256}, ProtocolIdentity{"synthetic-protocol", testHash('2'), testHash('2'), "synthetic.protocol.v1"}, CandidateScope{V00CandidateFamily, "LONG", "240m", false}, DatasetIdentity{HashIdentity{"synthetic-checkpoint", testHash('3')}, testHash('4'), HashIdentity{"synthetic-reacquisition", testHash('5')}, testHash('6'), testHash('7'), HashIdentity{"synthetic-abandoned", testHash('8')}, strings.Repeat("6", 40), append([]string(nil), universe.DatasetRequiredSymbols...), append([]string(nil), universe.CandidateTargetSymbols...), append([]string(nil), universe.ContextOnlySymbols...), universe.ContractSHA256, Interval{time.Date(2030, 1, 1, 0, 0, 0, 0, time.UTC), time.Date(2033, 1, 1, 0, 0, 0, 0, time.UTC)}, []Interval{{time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC), time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)}}, time.Date(2033, 1, 2, 0, 0, 0, 0, time.UTC)}, partitions, testIdentityLedger(ledger), AuthorityIdentity{HashIdentity{preconditions.AcceptedIndependencePolicyVersionV3, independenceHash}, HashIdentity{preconditions.AcceptedUncertaintyMethodVersion, uncertaintyHash}, policy.GovernanceDecisionHash, HashIdentity{qualification.PR4B0GateSetID, gateHash}, gateIDs, HashIdentity{"synthetic-cost-policy", testHash('9')}, HashIdentity{"synthetic-seed-policy", testHash('a')}}, AccessPolicy{true, []string{"exact runner", "reservation"}, []string{"development sealed", "registered nominee"}, []string{"candidate frozen", "validation sealed"}, []string{"dataset", "executable", "no defaults"}, 1, "NO_RETRY_AFTER_ACCESS", true}}
	return identity, ledger, runner
}

func stagePlan(t *testing.T, identity ResearchIdentityV4, ledger VariantLedger, runner rifbridge.RunnerImplementationIdentity, identityHash, stage string, ids []string) rifbridge.StageExecutionPlan {
	t.Helper()
	configurations := make([]rifbridge.RegisteredConfigurationIdentity, len(ids))
	for i, id := range ids {
		variant, _ := findVariant(ledger, id)
		raw, _ := json.Marshal(variant.Configuration)
		configurations[i] = rifbridge.RegisteredConfigurationIdentity{SchemaVersion: rifbridge.RegisteredConfigurationSchemaVersion, VariantID: id, CanonicalConfiguration: raw, ConfigurationSHA256: variant.ConfigurationSHA256, CandidateFamilyID: identity.CandidateScope.FamilyID, ProtocolID: identity.Protocol.ID, ProtocolSHA256: identity.Protocol.SHA256}
	}
	partition := findIdentityPartition(identity, stage)
	authorities := stageAuthorities(identity.Authorities)
	plan := rifbridge.StageExecutionPlan{SchemaVersion: rifbridge.StageExecutionPlanSchemaVersion, ResearchIdentityHash: identityHash, Protocol: rifbridge.StageProtocolIdentity{ID: identity.Protocol.ID, SHA256: identity.Protocol.SHA256, ContentAddressedIdentity: identity.Protocol.ContentAddressedIdentity, SchemaVersion: identity.Protocol.SchemaVersion}, Stage: stage, Partition: bridgeStagePartition(partition), Checkpoint: rifbridge.StageHashIdentity{ID: identity.Dataset.Checkpoint.ID, SHA256: identity.Dataset.Checkpoint.SHA256}, DatasetIdentitySHA256: mustLocalHash(t, identity.Dataset), Runner: runner, Configurations: configurations, DeterministicSeedPolicy: authorities.DeterministicSeedPolicy, ExpectedExecutions: len(configurations), Complete: true, OrderingRule: "NUMERIC_VARIANT_ID_ASCENDING", Authorities: authorities, GateSet: authorities.QualificationGateSet}
	plan.PlanHash = mustLocalHash(t, plan)
	return plan
}

func issueStageSet(t *testing.T, plan rifbridge.StageExecutionPlan, prior string) rifbridge.StageExecutionSet {
	t.Helper()
	setIdentity := struct {
		PlanHash string `json:"plan_hash"`
		Stage    string `json:"stage"`
	}{plan.PlanHash, plan.Stage}
	set := rifbridge.StageExecutionSet{SchemaVersion: rifbridge.StageExecutionSetSchemaVersion, ExecutionSetID: "execution-set:" + strings.TrimPrefix(mustLocalHash(t, setIdentity), "sha256:"), Plan: plan, IssuanceState: "ISSUED", CompletionState: "OPEN", Authorizations: []rifbridge.StageVariantAuthorization{}, AccessReceipts: []rifbridge.StageVariantAccessReceipt{}, RetryProofs: []rifbridge.ZeroAccessRetryProof{}, ExecutionReceipts: []rifbridge.StageExecutionReceipt{}, IssuedAt: time.Date(2029, 1, 1, 0, 0, 3, 0, time.UTC)}
	previous := ""
	for ordinal, configuration := range plan.Configurations {
		authIdentity := struct {
			ExecutionSetID string `json:"execution_set_id"`
			VariantID      string `json:"variant_id"`
			Ordinal        int    `json:"ordinal"`
		}{set.ExecutionSetID, configuration.VariantID, ordinal}
		auth := rifbridge.StageVariantAuthorization{SchemaVersion: rifbridge.StageAuthorizationSchemaVersion, AuthorizationID: "stage-authorization:" + strings.TrimPrefix(mustLocalHash(t, authIdentity), "sha256:"), ExecutionSetID: set.ExecutionSetID, PlanHash: plan.PlanHash, Ordinal: ordinal, Attempt: 1, Configuration: configuration, Runner: plan.Runner, Partition: plan.Partition, Protocol: plan.Protocol, Checkpoint: plan.Checkpoint, Authorities: plan.Authorities, GateSet: plan.GateSet, IssuedAt: set.IssuedAt, PriorStateHash: prior, PreviousHash: previous}
		auth.RecordHash = mustLocalHash(t, auth)
		set.Authorizations = append(set.Authorizations, auth)
		previous = auth.RecordHash
	}
	set.RecordHash = mustLocalHash(t, set)
	return set
}

func stageReservationHistory(t *testing.T, identity ResearchIdentityV4, identityHash string) (*rifbridge.HoldoutReservation, []rifbridge.ResearchLifecycleRecord) {
	final := findIdentityPartition(identity, "FINAL_HOLDOUT")
	finalBytes, _ := json.Marshal(final)
	ledgerHash, _ := canonicalHash(identity.VariantLedger)
	authorityHash, _ := canonicalHash(identity.Authorities)
	reservation := &rifbridge.HoldoutReservation{SchemaVersion: rifbridge.HoldoutReservationSchemaVersion, ReservationID: "reservation:synthetic-stage", ResearchIdentityHash: identityHash, FinalHoldout: finalBytes, ProtocolSHA256: identity.Protocol.SHA256, CheckpointSHA256: identity.Dataset.Checkpoint.SHA256, VariantLedgerSHA256: ledgerHash, AuthoritySetSHA256: authorityHash, CreatedAt: time.Date(2029, 1, 1, 0, 0, 1, 0, time.UTC)}
	reservation.RecordHash = mustLocalHash(t, *reservation)
	history := []rifbridge.ResearchLifecycleRecord{{SchemaVersion: rifbridge.ResearchLifecycleRecordSchemaVersion, Sequence: 1, EventType: "RESEARCH_IDENTITY_REGISTERED", ToState: "RESEARCH_IDENTITY_REGISTERED", OccurredAt: time.Date(2029, 1, 1, 0, 0, 0, 0, time.UTC), EvidenceSHA256: identityHash, PriorStateHash: testHash('b')}, {SchemaVersion: rifbridge.ResearchLifecycleRecordSchemaVersion, Sequence: 2, EventType: "HOLDOUT_RESERVED", FromState: "RESEARCH_IDENTITY_REGISTERED", ToState: "HOLDOUT_RESERVED", OccurredAt: reservation.CreatedAt, EvidenceSHA256: reservation.RecordHash, PriorStateHash: testHash('c')}}
	history[0].RecordHash = mustLocalHash(t, history[0])
	history[1].PreviousHash = history[0].RecordHash
	history[1].RecordHash = mustLocalHash(t, history[1])
	return reservation, history
}

func stageAuthorities(value AuthorityIdentity) rifbridge.StageAuthorityIdentity {
	gates := make([]rifbridge.StageHashIdentity, len(value.QualificationGateHashes))
	for i, gate := range value.QualificationGateHashes {
		gates[i] = rifbridge.StageHashIdentity{ID: gate.ID, SHA256: gate.SHA256}
	}
	return rifbridge.StageAuthorityIdentity{Independence: rifbridge.StageHashIdentity{ID: value.Independence.ID, SHA256: value.Independence.SHA256}, Uncertainty: rifbridge.StageHashIdentity{ID: value.Uncertainty.ID, SHA256: value.Uncertainty.SHA256}, ConcentrationSHA256: value.ConcentrationSHA256, QualificationGateSet: rifbridge.StageHashIdentity{ID: value.QualificationGateSet.ID, SHA256: value.QualificationGateSet.SHA256}, QualificationGateHashes: gates, TransactionCostPolicy: rifbridge.StageHashIdentity{ID: value.TransactionCostPolicy.ID, SHA256: value.TransactionCostPolicy.SHA256}, DeterministicSeedPolicy: rifbridge.StageHashIdentity{ID: value.DeterministicSeedPolicy.ID, SHA256: value.DeterministicSeedPolicy.SHA256}}
}

func bridgeStagePartition(value Partition) rifbridge.StagePartition {
	return rifbridge.StagePartition{Name: value.Name, Interval: rifbridge.StageInterval{Start: value.Interval.Start, End: value.Interval.End}, StructuralDayCount: value.StructuralDayCount, RequiredSymbolCoverageSHA256: value.RequiredSymbolCoverageSHA256}
}

func syntheticStageOutput(authorization rifbridge.StageVariantAuthorization, passed bool) StageVariantOutput {
	invocations := []StageAuthorityInvocationEvidence{{Identity: rifbridge.StageHashIdentity{ID: "ak.engine.governance.concentration-decision.v1", SHA256: authorization.Authorities.ConcentrationSHA256}, Invoked: true, EvidenceSHA256: testHash('1')}, {Identity: authorization.Authorities.Independence, Invoked: true, EvidenceSHA256: testHash('2')}, {Identity: authorization.Authorities.Uncertainty, Invoked: true, EvidenceSHA256: testHash('3')}}
	sort.Slice(invocations, func(i, j int) bool { return invocations[i].Identity.ID < invocations[j].Identity.ID })
	return StageVariantOutput{ResultArtifact: json.RawMessage(fmt.Sprintf(`{"synthetic":true,"variant":%q}`, authorization.Configuration.VariantID)), OutputManifestSHA256: testHash('4'), AuthorityInvocations: invocations, MandatoryGatesPassed: passed}
}

func stageArtifacts(t *testing.T, envelopes []rifbridge.StageExecutionEnvelope) [][]byte {
	t.Helper()
	artifacts := make([][]byte, len(envelopes))
	for i := range envelopes {
		artifacts[i] = stageArtifact(t, envelopes[i])
	}
	return artifacts
}

func stageRetryEnvelope(t *testing.T, original rifbridge.StageExecutionEnvelope) rifbridge.StageExecutionEnvelope {
	t.Helper()
	envelope := cloneStageEnvelope(original)
	set := &envelope.ExecutionSet
	priorAuthorization := *envelope.Authorization
	priorAccess := set.AccessReceipts[0]
	proof := rifbridge.ZeroAccessRetryProof{SchemaVersion: rifbridge.StageRetryProofSchemaVersion, ExecutionSetID: set.ExecutionSetID, PriorAuthorizationID: priorAuthorization.AuthorizationID, PriorAccessReceiptHash: priorAccess.RecordHash, VariantID: priorAuthorization.Configuration.VariantID, RowsAccessed: 0, OutcomeArtifacts: 0, DurableProofSHA256: testHash('9'), ProvenAt: time.Date(2029, 1, 1, 0, 2, 0, 0, time.UTC)}
	proof.RecordHash = mustLocalHash(t, proof)
	set.RetryProofs = append(set.RetryProofs, proof)
	retry := priorAuthorization
	retry.AuthorizationID += ":retry:2"
	retry.Attempt = 2
	retry.IssuedAt = time.Date(2029, 1, 1, 0, 2, 1, 0, time.UTC)
	retry.PriorStateHash = envelope.Snapshot.IntegrityHash
	retry.PreviousHash = set.Authorizations[len(set.Authorizations)-1].RecordHash
	retry.RecordHash = ""
	retry.RecordHash = mustLocalHash(t, retry)
	set.Authorizations = append(set.Authorizations, retry)
	access := rifbridge.StageVariantAccessReceipt{SchemaVersion: rifbridge.StageAccessReceiptSchemaVersion, ExecutionSetID: set.ExecutionSetID, AuthorizationID: retry.AuthorizationID, VariantID: retry.Configuration.VariantID, Attempt: retry.Attempt, ConsumedAt: time.Date(2029, 1, 1, 0, 2, 2, 0, time.UTC), PriorStateHash: envelope.Snapshot.IntegrityHash, PreviousHash: set.AccessReceipts[len(set.AccessReceipts)-1].RecordHash}
	access.RecordHash = mustLocalHash(t, access)
	set.AccessReceipts = append(set.AccessReceipts, access)
	set.RecordHash = ""
	set.RecordHash = mustLocalHash(t, *set)
	for index, evidence := range []string{proof.RecordHash, access.RecordHash} {
		event := rifbridge.ResearchLifecycleRecord{SchemaVersion: rifbridge.ResearchLifecycleRecordSchemaVersion, Sequence: envelope.Snapshot.Sequence + 1, EventType: []string{"DEVELOPMENT_ZERO_ACCESS_RETRY_AUTHORIZED", "DEVELOPMENT_VARIANT_ACCESS_CONSUMED"}[index], FromState: envelope.Snapshot.State, ToState: envelope.Snapshot.State, OccurredAt: []time.Time{proof.ProvenAt, access.ConsumedAt}[index], EvidenceSHA256: evidence, PriorStateHash: envelope.Snapshot.IntegrityHash, PreviousHash: envelope.Snapshot.LifecycleHistory[len(envelope.Snapshot.LifecycleHistory)-1].RecordHash}
		event.RecordHash = mustLocalHash(t, event)
		envelope.Snapshot.Sequence++
		envelope.Snapshot.LifecycleHistory = append(envelope.Snapshot.LifecycleHistory, event)
	}
	envelope.Snapshot.StageExecutionSets[0] = *set
	envelope.Snapshot.IntegrityHash = ""
	envelope.Snapshot.IntegrityHash = mustLocalHash(t, envelope.Snapshot)
	envelope.Authorization = &retry
	envelope.EnvelopeHash = ""
	envelope.EnvelopeHash = mustLocalHash(t, envelope)
	if err := rifbridge.VerifyStageExecutionEnvelope(envelope); err != nil {
		t.Fatalf("synthetic zero-access retry envelope invalid: %v", err)
	}
	return envelope
}

func stageArtifact(t *testing.T, envelope rifbridge.StageExecutionEnvelope) []byte {
	t.Helper()
	identity, _, err := decodeIdentity(envelope.Snapshot.Identity)
	if err != nil {
		t.Fatal(err)
	}
	start := envelope.ExecutionSet.Plan.Partition.Interval.Start
	artifact, err := SealPartitionArtifact(PartitionArtifact{Label: SyntheticLabel, CheckpointSHA256: identity.Dataset.Checkpoint.SHA256, SourceIdentitySHA256: identity.Dataset.SourceIdentitySHA256, SealedBinarySHA256: identity.Dataset.SealedBinarySHA256, Partition: envelope.ExecutionSet.Plan.Stage, PartitionPlanSHA256: envelope.ExecutionSet.Plan.Partition.RequiredSymbolCoverageSHA256, DatasetSymbols: append([]string(nil), identity.Dataset.RequiredSymbols...), TargetSymbols: append([]string(nil), identity.Dataset.CandidateTargetSymbols...), ContextOnlySymbols: append([]string(nil), identity.Dataset.ContextOnlySymbols...), Rows: passingRows(envelope.ExecutionSet.Plan.Stage, start, 300)})
	if err != nil {
		t.Fatal(err)
	}
	encoded, err := EncodePartitionArtifact(artifact)
	if err != nil {
		t.Fatal(err)
	}
	return encoded
}

func cloneStageEnvelope(value rifbridge.StageExecutionEnvelope) rifbridge.StageExecutionEnvelope {
	data, _ := json.Marshal(value)
	var out rifbridge.StageExecutionEnvelope
	_ = json.Unmarshal(data, &out)
	return out
}
func mustJSON(t *testing.T, value any) []byte {
	t.Helper()
	data, err := json.Marshal(value)
	if err != nil {
		t.Fatal(err)
	}
	return data
}
