package app

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/david22573/ak-engine/internal/rifbridge"
	"github.com/spf13/cobra"
)

var rifSmokeOutDir string
var rsManifest string

var rifSmokeCmd = &cobra.Command{
	Use:   "rif-smoke",
	Short: "Run a deterministic smoke test for RIF integration",
	RunE: func(cmd *cobra.Command, args []string) error {
		if rifSmokeOutDir == "" {
			rifSmokeOutDir = filepath.Join("runs", "reports", "rif_smoke")
		}
		if err := os.MkdirAll(rifSmokeOutDir, 0755); err != nil {
			return fmt.Errorf("failed to create out dir: %w", err)
		}

		bridge := rifbridge.NewBridge()

		var manifestPath string
		if rsManifest != "" {
			manifestPath = rsManifest
		} else {
			manifestPath = filepath.Join(rifSmokeOutDir, "dataset_manifest.json")
			os.WriteFile(manifestPath, []byte(`{"dataset_id":"smoke-data","hashes":{"dataset_hash":"d-hash","manifest_hash":"m-hash"},"survivorship":{"universe_id":"smoke-point-in-time","universe_hash":"u-hash","universe_manifest_hash":"um-hash","universe_policy":"POINT_IN_TIME_EXCHANGE_UNIVERSE","includes_delisted_assets":"true","survivorship_bias_risk":"LOW","lifecycle_id":"smoke-life","lifecycle_hash":"life-hash","lifecycle_manifest_hash":"life-manifest-hash","lifecycle_evidence_level_summary":{"HISTORICAL_SNAPSHOT_EVIDENCE":1},"listing_evidence_status":"VERIFIED","delisting_evidence_status":"VERIFIED","survivorship_support_status":"LOW_SUPPORTED"}}`), 0644)
		}
		universeBlocksPromotion, err := datasetManifestBlocksStrictPromotion(manifestPath)
		if err != nil {
			return err
		}

		// 1. Clean Candidate (passes integrity)
		cleanStem := filepath.Join(rifSmokeOutDir, "clean_candidate")
		cleanOut, err := bridge.EvaluateAndEmit(
			cleanStem,
			"cand-clean-01",
			"v1.0.0",
			"sha-12345",
			[]string{"data-1"},
			[]string{"feat-1"},
			"hash-cand-clean",
			"hash-config-clean",
			[]float64{0.01, 0.02, -0.01, 0.03},
			[]int64{100, 200, 300, 400},
			2,
			50, // observations >= 30
			true,
			manifestPath,
		)
		if err != nil {
			return fmt.Errorf("clean candidate failed: %w", err)
		}
		if universeBlocksPromotion && cleanOut.IntegrityPassed {
			return fmt.Errorf("expected clean candidate promotion to be blocked by universe policy")
		}
		if !universeBlocksPromotion && !cleanOut.IntegrityPassed {
			return fmt.Errorf("expected clean candidate to pass integrity")
		}

		// 2. Low Sample Candidate
		lowSampleStem := filepath.Join(rifSmokeOutDir, "low_sample_candidate")
		lowSampleOut, err := bridge.EvaluateAndEmit(
			lowSampleStem,
			"cand-low-01",
			"v1.0.0",
			"sha-12345",
			[]string{"data-1"},
			[]string{"feat-1"},
			"hash-cand-low",
			"hash-config-low",
			[]float64{0.01, 0.02},
			[]int64{100, 200},
			2,
			10, // observations < 30
			true,
			"",
		)
		if err != nil {
			return fmt.Errorf("low sample candidate failed: %w", err)
		}
		if lowSampleOut.IntegrityPassed {
			return fmt.Errorf("expected low sample candidate to fail integrity")
		}

		// 3. Parameter Mining Risk
		paramMiningStem := filepath.Join(rifSmokeOutDir, "param_mining_candidate")
		paramMiningOut, err := bridge.EvaluateAndEmit(
			paramMiningStem,
			"cand-param-01",
			"v1.0.0",
			"sha-12345",
			[]string{"data-1"},
			[]string{"feat-1"},
			"hash-cand-param",
			"hash-config-param",
			[]float64{0.01, 0.02, 0.03},
			[]int64{100, 200, 300},
			10, // 10 parameters for 3 observations (ratio < 20)
			3,  // obs
			true,
			"",
		)
		if err != nil {
			return fmt.Errorf("param mining candidate failed: %w", err)
		}
		if paramMiningOut.IntegrityPassed {
			return fmt.Errorf("expected param mining candidate to fail integrity")
		}

		// 4. Bad Lockfile (VerifyLock)
		// We'll create a dummy lock and a changed lock and verify they fail
		lock1 := rifbridge.ResearchLock{GitSHA: "sha1"}
		lock2 := rifbridge.ResearchLock{GitSHA: "sha2"}
		if err := bridge.VerifyLock(lock1, lock2); err == nil {
			return fmt.Errorf("expected lock verification to fail for mismatched locks")
		}

		// 5. Holdout Overexposure
		// We log exposure 11 times. The limit is 10.
		for i := 0; i < 11; i++ {
			err = bridge.LogHoldoutExposure("dataset-holdout-1", fmt.Sprintf("cand-%d", i), "exp-1")
			if i < 10 {
				if err != nil {
					return fmt.Errorf("unexpected holdout error at exposure %d: %w", i, err)
				}
			} else {
				if err == nil {
					return fmt.Errorf("expected holdout error at exposure 11, but got nil")
				}
			}
		}

		// 6. Run Finalization
		runStem := filepath.Join(rifSmokeOutDir, "run_final")
		err = bridge.EmitRunFinalization(
			runStem,
			"sha-12345",
			[]string{"data-1"},
			[]string{"feat-1"},
			"hash-config-run",
			"",
		)
		if err != nil {
			return fmt.Errorf("run finalization failed: %w", err)
		}

		fmt.Println("rif-smoke completed successfully")
		return nil
	},
}

func datasetManifestBlocksStrictPromotion(path string) (bool, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return false, fmt.Errorf("failed to read dataset manifest for smoke expectation: %w", err)
	}
	var m struct {
		Survivorship struct {
			UniversePolicy                       string         `json:"universe_policy"`
			IncludesDelistedAssets               string         `json:"includes_delisted_assets"`
			SurvivorshipBiasRisk                 string         `json:"survivorship_bias_risk"`
			UniverseHash                         string         `json:"universe_hash"`
			UniverseManifestHash                 string         `json:"universe_manifest_hash"`
			LifecycleHash                        string         `json:"lifecycle_hash"`
			LifecycleManifestHash                string         `json:"lifecycle_manifest_hash"`
			LifecycleEvidenceLevelSummary        map[string]int `json:"lifecycle_evidence_level_summary"`
			ListingEvidenceStatus                string         `json:"listing_evidence_status"`
			DelistingEvidenceStatus              string         `json:"delisting_evidence_status"`
			SurvivorshipSupportStatus            string         `json:"survivorship_support_status"`
			ExchangeMetadataSnapshotHash         string         `json:"exchange_metadata_snapshot_hash"`
			ExchangeMetadataSnapshotManifestHash string         `json:"exchange_metadata_snapshot_manifest_hash"`
			ExchangeMetadataSnapshotCurrentOnly  bool           `json:"exchange_metadata_snapshot_current_only"`
			PointInTimeCoverageStatus            string         `json:"point_in_time_coverage_status"`
		} `json:"survivorship"`
	}
	if err := json.Unmarshal(data, &m); err != nil {
		return false, fmt.Errorf("failed to parse dataset manifest for smoke expectation: %w", err)
	}
	if m.Survivorship.UniverseHash == "" || m.Survivorship.UniverseManifestHash == "" {
		return true, nil
	}
	if m.Survivorship.LifecycleHash == "" || m.Survivorship.LifecycleManifestHash == "" {
		return true, nil
	}
	if m.Survivorship.ListingEvidenceStatus != "VERIFIED" ||
		m.Survivorship.DelistingEvidenceStatus != "VERIFIED" ||
		m.Survivorship.SurvivorshipSupportStatus != "LOW_SUPPORTED" ||
		smokeHasWeakLifecycleEvidence(m.Survivorship.LifecycleEvidenceLevelSummary) {
		return true, nil
	}
	if m.Survivorship.ExchangeMetadataSnapshotHash != "" || m.Survivorship.ExchangeMetadataSnapshotManifestHash != "" {
		if m.Survivorship.ExchangeMetadataSnapshotCurrentOnly || m.Survivorship.PointInTimeCoverageStatus != "COVERS_WINDOW" {
			return true, nil
		}
	}
	switch m.Survivorship.UniversePolicy {
	case "POINT_IN_TIME_EXCHANGE_UNIVERSE", "POINT_IN_TIME_VOLUME_FILTERED_UNIVERSE", "POINT_IN_TIME_MARKET_CAP_FILTERED_UNIVERSE":
		return m.Survivorship.SurvivorshipBiasRisk != "LOW" || m.Survivorship.IncludesDelistedAssets != "true", nil
	default:
		return true, nil
	}
}

func smokeHasWeakLifecycleEvidence(summary map[string]int) bool {
	for level, count := range summary {
		if count <= 0 {
			continue
		}
		switch level {
		case "LOCAL_DATA_FIRST_SEEN", "CURRENT_ACTIVE_ONLY", "USER_PROVIDED_UNVERIFIED", "UNKNOWN":
			return true
		}
	}
	return false
}

func init() {
	rifSmokeCmd.Flags().StringVar(&rifSmokeOutDir, "out-dir", "", "Output directory for smoke test artifacts")
	rifSmokeCmd.Flags().StringVar(&rsManifest, "dataset-manifest", "", "Path to dataset manifest")
	rootCmd.AddCommand(rifSmokeCmd)
}
