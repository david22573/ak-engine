package app

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/david22573/ak-engine/internal/researchidentity"
	"github.com/david22573/ak-engine/internal/rifbridge"
	"github.com/spf13/cobra"
)

var researchDiagnosticsSmokeOutDir string

var researchDiagnosticsSmokeCmd = &cobra.Command{
	Use:   "research-diagnostics-smoke",
	Short: "Run deterministic Engine-local exact-identity diagnostics fixtures",
	RunE: func(cmd *cobra.Command, args []string) error {
		if researchDiagnosticsSmokeOutDir == "" {
			researchDiagnosticsSmokeOutDir = filepath.Join("runs", "reports", "research_diagnostics_smoke")
		}
		if err := os.MkdirAll(researchDiagnosticsSmokeOutDir, 0755); err != nil {
			return fmt.Errorf("create research diagnostics output directory: %w", err)
		}

		fixture, err := researchidentity.BuildDiagnosticSmokeFixture(researchDiagnosticsSmokeOutDir)
		if err != nil {
			return fmt.Errorf("build exact-identity smoke fixture: %w", err)
		}
		defer fixture.Cleanup()
		bridge := rifbridge.NewBridgeWithDeriver(fixture.Deriver)

		complete, err := bridge.EmitResearchDiagnostics(rifbridge.ResearchAssessment{
			Stem:            filepath.Join(researchDiagnosticsSmokeOutDir, "complete_candidate"),
			Classification:  rifbridge.ResearchStatusValidatedResearchLead,
			IdentityRequest: fixture.Request,
		})
		if err != nil {
			return fmt.Errorf("complete exact-identity diagnostic: %w", err)
		}
		if complete.IdentityStatus != researchidentity.StatusComplete || !complete.EligibleForReview || complete.LocalIntegrity != rifbridge.LocalIntegrityPassed {
			return fmt.Errorf("complete fixture was not reviewable: %#v", complete)
		}

		incompleteRequest := fixture.Request
		incompleteRequest.HistorianManifestPath = ""
		incomplete, err := bridge.EmitResearchDiagnostics(rifbridge.ResearchAssessment{
			Stem:            filepath.Join(researchDiagnosticsSmokeOutDir, "incomplete_candidate"),
			Classification:  rifbridge.ResearchStatusValidatedResearchLead,
			IdentityRequest: incompleteRequest,
		})
		var derivationErr *researchidentity.DerivationError
		if !errors.As(err, &derivationErr) {
			return fmt.Errorf("incomplete fixture did not return typed derivation error: %w", err)
		}
		if incomplete.ArtifactDisposition != rifbridge.ArtifactEmitted || incomplete.EligibleForReview || incomplete.IdentityStatus == researchidentity.StatusComplete {
			return fmt.Errorf("incomplete fixture did not fail closed: %#v", incomplete)
		}

		rejected, err := bridge.EmitResearchDiagnostics(rifbridge.ResearchAssessment{
			Stem:            filepath.Join(researchDiagnosticsSmokeOutDir, "rejected_candidate"),
			Classification:  rifbridge.ResearchStatusRejected,
			IdentityRequest: fixture.Request,
		})
		if err != nil {
			return fmt.Errorf("rejected complete-identity diagnostic: %w", err)
		}
		if rejected.IdentityStatus != researchidentity.StatusComplete || rejected.EligibleForReview {
			return fmt.Errorf("rejected classification became reviewable: %#v", rejected)
		}

		fmt.Println("research-diagnostics-smoke completed successfully")
		return nil
	},
}

func init() {
	researchDiagnosticsSmokeCmd.Flags().StringVar(&researchDiagnosticsSmokeOutDir, "out-dir", "", "Output directory for local research diagnostics")
	rootCmd.AddCommand(researchDiagnosticsSmokeCmd)
}
