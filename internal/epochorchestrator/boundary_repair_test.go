package epochorchestrator

import (
	"path/filepath"
	"testing"
	"time"

	"github.com/david22573/ak-engine/internal/partitionpipeline"
)

func TestOriginalADAUSDTDefectWitnessUsesSealedProductionProvenance(t *testing.T) {
	end := time.Date(2026, 4, 10, 0, 0, 0, 0, time.UTC)
	entry := partitionpipeline.PreparationManifestEntry{
		Symbol:                "ADAUSDT",
		UTCDate:               "2026-04-09",
		MembershipInterval:    partitionpipeline.Interval{Start: end.Add(-24 * time.Hour), End: end},
		ChildRowCount:         560,
		ChildLastTimestampUTC: end.Add(-time.Minute),
		BoundaryClass:         "RIGHT_CLIPPED",
		Parent: partitionpipeline.ParentProvenance{
			ReceiptSHA256:  "sha256:a51a972bdec54aa84149458e9fe3de472973a6673aa4121725fef06dd4ba1735",
			FragmentSHA256: "sha256:cfad25fa5ee30f53f012f6ce9e6d92565dfe6cec21af063ecff3c61e250506b1",
		},
	}
	if !isOriginalADAUSDTDefect("DEVELOPMENT", entry) {
		t.Fatal("sealed ADAUSDT production provenance did not match the regression witness")
	}
	entry.Parent.FragmentSHA256 = "sha256:cfad25fa8df42974951955c34965ab3bfa07615bfddfd0394d039a0b035d67a6"
	if isOriginalADAUSDTDefect("DEVELOPMENT", entry) {
		t.Fatal("the previously mistyped fragment identity matched the regression witness")
	}
}

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
