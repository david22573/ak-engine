package epochorchestrator

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/david22573/ak-engine/internal/partitionpipeline"
	"github.com/david22573/ak-rif/research"
)

func TestProductionShapedSyntheticEpochAndResume(t *testing.T) {
	config, err := CreateSyntheticConfig(filepath.Join(t.TempDir(), "checkpoint"))
	if err != nil {
		t.Fatal(err)
	}
	epoch := filepath.Join(t.TempDir(), "epoch")
	orchestrator, err := New(epoch, config)
	if err != nil {
		t.Fatal(err)
	}
	status, err := orchestrator.Preflight()
	if err != nil || status != ReadyStatus {
		t.Fatalf("preflight=%s err=%v", status, err)
	}
	if _, err := os.Stat(orchestrator.rifPath()); !os.IsNotExist(err) {
		t.Fatal("preflight created RIF governance state")
	}
	if err := orchestrator.RegisterIdentity(); err == nil {
		t.Fatal("identity registered before protocol")
	}
	steps := []struct {
		name string
		run  func() error
	}{{"protocol", orchestrator.CommitProtocol}, {"identity", orchestrator.RegisterIdentity}, {"reserve", orchestrator.ReserveHoldout}, {"authorize development", orchestrator.AuthorizeDevelopmentSet}, {"run development", orchestrator.RunDevelopmentSet}, {"seal development", orchestrator.SealDevelopmentSet}, {"nominee", orchestrator.DeriveNominee}, {"authorize validation", orchestrator.AuthorizeValidationSet}, {"run validation", orchestrator.RunValidationSet}, {"seal validation", orchestrator.SealValidationSet}, {"freeze", orchestrator.FreezeCandidate}, {"authorize final", orchestrator.AuthorizeFinalHoldout}, {"run final", orchestrator.RunFinalHoldout}, {"seal final", orchestrator.SealFinalHoldout}, {"closeout", orchestrator.Closeout}}
	for _, step := range steps {
		if err := step.run(); err != nil {
			t.Fatalf("%s: %v", step.name, err)
		}
		if step.name == "run final" {
			if err := orchestrator.RunFinalHoldout(); err == nil {
				t.Fatal("FINAL_HOLDOUT replay succeeded")
			}
		}
	}
	final, err := orchestrator.Status()
	if err != nil {
		t.Fatal(err)
	}
	if final.Orchestrator.Phase != "QUALIFIED" && final.Orchestrator.Phase != "REJECTED" {
		t.Fatalf("nonterminal phase %s", final.Orchestrator.Phase)
	}
	if final.RIFState != final.Orchestrator.Phase {
		t.Fatalf("RIF/orchestrator state mismatch %s/%s", final.RIFState, final.Orchestrator.Phase)
	}
	before := final.RIFIntegritySHA256
	resumed, err := orchestrator.Resume()
	if err != nil {
		t.Fatal(err)
	}
	if resumed.RIFIntegritySHA256 != before {
		t.Fatal("read-only resume mutated durable state")
	}
	authority, err := research.OpenAuthority(orchestrator.rifPath())
	if err != nil {
		t.Fatal(err)
	}
	snapshot, err := authority.Snapshot()
	if err != nil {
		t.Fatal(err)
	}
	validation := findStageSet(snapshot, "VALIDATION")
	if validation == nil || validation.Plan.ExpectedExecutions != 3 || validation.Plan.Configurations[0].VariantID != snapshot.DevelopmentNominee.VariantID {
		t.Fatalf("VALIDATION was not restricted to nominee plus mandatory neighbors: %#v", validation)
	}
}

func TestStageReplayAndWrongConfigFailClosed(t *testing.T) {
	config, err := CreateSyntheticConfig(filepath.Join(t.TempDir(), "checkpoint"))
	if err != nil {
		t.Fatal(err)
	}
	epoch := filepath.Join(t.TempDir(), "epoch")
	o, _ := New(epoch, config)
	if _, err := o.Preflight(); err != nil {
		t.Fatal(err)
	}
	for _, run := range []func() error{o.CommitProtocol, o.RegisterIdentity, o.ReserveHoldout, o.AuthorizeDevelopmentSet, o.RunDevelopmentSet} {
		if err := run(); err != nil {
			t.Fatal(err)
		}
	}
	if err := o.RunDevelopmentSet(); err != nil {
		t.Fatalf("idempotent completed DEVELOPMENT resume failed: %v", err)
	}
	wrong := config
	wrong.Identity.Repositories.RunnerGitCommit = "0000000000000000000000000000000000000000"
	wrong.ConfigSHA256 = ""
	wrong, _ = SealConfig(wrong)
	other, _ := New(epoch, wrong)
	if _, err := other.Status(); err == nil {
		t.Fatal("another runner/configuration reopened epoch")
	}
}

func TestDurableZeroAccessProofAuthorizesOneRetry(t *testing.T) {
	o := authorizedSyntheticDevelopment(t)
	authority, snapshot, err := o.authority()
	if err != nil {
		t.Fatal(err)
	}
	set := findStageSet(snapshot, "DEVELOPMENT")
	first := set.Authorizations[0]
	if _, err := authority.ConsumeStageVariantBeforeAccess(first, func() error { return nil }); err != nil {
		t.Fatal(err)
	}
	if err := o.RunDevelopmentSet(); err != nil {
		t.Fatalf("proven zero-access attempt did not resume: %v", err)
	}
	_, snapshot, err = o.authority()
	if err != nil {
		t.Fatal(err)
	}
	set = findStageSet(snapshot, "DEVELOPMENT")
	if len(set.RetryProofs) != 1 || set.RetryProofs[0].RowsAccessed != 0 || set.RetryProofs[0].OutcomeArtifacts != 0 || latestStageAuthorization(set, first.Configuration.VariantID).Attempt != 2 || stageReceipt(set, first.Configuration.VariantID) == nil {
		t.Fatalf("zero-access retry chain is incomplete: %#v", set)
	}
}

func TestIndeterminatePostMaterializationAccessBlocks(t *testing.T) {
	o := authorizedSyntheticDevelopment(t)
	authority, snapshot, err := o.authority()
	if err != nil {
		t.Fatal(err)
	}
	set := findStageSet(snapshot, "DEVELOPMENT")
	first := set.Authorizations[0]
	access, err := authority.ConsumeStageVariantBeforeAccess(first, func() error { return nil })
	if err != nil {
		t.Fatal(err)
	}
	plan := o.config.Plans["DEVELOPMENT"]
	authorization, err := partitionpipeline.SealMaterializationAuthorization(partitionpipeline.MaterializationAuthorization{PlanSHA256: plan.PlanSHA256, CheckpointSHA256: plan.Checkpoint.SHA256, Partition: plan.PartitionName, RIFAuthorizationID: first.AuthorizationID, RIFAuthorizationSHA256: first.RecordHash, RIFAccessReceiptSHA256: access.RecordHash, AuthorizedAt: time.Now().UTC()})
	if err != nil {
		t.Fatal(err)
	}
	if err := partitionpipeline.AuthorizeMaterialization(o.partitionRegistryPath(), plan.PlanSHA256, authorization); err != nil {
		t.Fatal(err)
	}
	if _, _, _, err := partitionpipeline.Materialize(o.partitionRegistryPath(), plan.PlanSHA256, time.Now().UTC()); err != nil {
		t.Fatal(err)
	}
	if err := o.RunDevelopmentSet(); err == nil {
		t.Fatal("indeterminate receipt-bearing attempt resumed")
	}
	status, err := o.Status()
	if err != nil {
		t.Fatal(err)
	}
	if status.Orchestrator.Phase != "BLOCKED" || status.RIFState != "BLOCKED" {
		t.Fatalf("indeterminate access did not block both authorities: %#v", status)
	}
}

func authorizedSyntheticDevelopment(t *testing.T) *Orchestrator {
	t.Helper()
	config, err := CreateSyntheticConfig(filepath.Join(t.TempDir(), "checkpoint"))
	if err != nil {
		t.Fatal(err)
	}
	o, err := New(filepath.Join(t.TempDir(), "epoch"), config)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := o.Preflight(); err != nil {
		t.Fatal(err)
	}
	for _, run := range []func() error{o.CommitProtocol, o.RegisterIdentity, o.ReserveHoldout, o.AuthorizeDevelopmentSet} {
		if err := run(); err != nil {
			t.Fatal(err)
		}
	}
	return o
}
