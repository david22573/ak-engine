package epochorchestrator

import (
	"path/filepath"
	"testing"
)

func TestBoundarySemanticDeltaPermitsOnlyInfrastructureIdentityChanges(t *testing.T) {
	config, err := CreateSyntheticConfig(filepath.Join(t.TempDir(), "checkpoint"))
	if err != nil {
		t.Fatal(err)
	}
	before, err := boundarySemanticSnapshot(config)
	if err != nil {
		t.Fatal(err)
	}
	config.Protocol, config.Identity.Protocol, err = boundaryProtocolIdentity(config.Protocol, config.Identity.Protocol, "synthetic-r1p9-fresh-protocol")
	if err != nil {
		t.Fatal(err)
	}
	config.Identity.ResearchID = "synthetic-r1p9-fresh-epoch"
	after, err := boundarySemanticSnapshot(config)
	if err != nil {
		t.Fatal(err)
	}
	if differences := semanticDifferences(before, after); len(differences) != 0 {
		t.Fatalf("infrastructure identity change altered semantics: %v", differences)
	}
	config.Identity.CandidateScope.FamilyID = "substituted-family"
	drifted, err := boundarySemanticSnapshot(config)
	if err != nil {
		t.Fatal(err)
	}
	if differences := semanticDifferences(before, drifted); len(differences) != 1 || differences[0] != "candidate_scope" {
		t.Fatalf("candidate semantic drift was not isolated: %v", differences)
	}
}
