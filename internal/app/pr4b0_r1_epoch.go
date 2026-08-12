package app

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/david22573/ak-engine/internal/epochorchestrator"
	"github.com/spf13/cobra"
)

var (
	pr4b0R1EpochConfig string
	pr4b0R1EpochRoot   string
)

var pr4b0R1EpochOperations = map[string]struct{}{
	"preflight": {}, "register-protocol": {}, "register-research-identity": {}, "reserve-holdout": {},
	"authorize-development-set": {}, "run-development-set": {}, "seal-development-set": {}, "derive-nominee": {},
	"authorize-validation-set": {}, "run-validation-set": {}, "seal-validation-set": {}, "freeze-candidate": {},
	"authorize-final-holdout": {}, "run-final-holdout": {}, "seal-final-holdout": {}, "closeout": {}, "status": {}, "resume": {},
}

var pr4b0R1EpochCmd = &cobra.Command{
	Use:   "pr4b0-r1-epoch <operation>",
	Short: "Drive the production PR4B0-R1 governed epoch lifecycle",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		return runPR4B0R1Epoch(args[0], pr4b0R1EpochConfig, pr4b0R1EpochRoot)
	},
}

func init() {
	pr4b0R1EpochCmd.Flags().StringVar(&pr4b0R1EpochConfig, "config", "", "canonical production epoch configuration")
	pr4b0R1EpochCmd.Flags().StringVar(&pr4b0R1EpochRoot, "epoch-root", "", "isolated durable epoch state directory")
	_ = pr4b0R1EpochCmd.MarkFlagRequired("config")
	_ = pr4b0R1EpochCmd.MarkFlagRequired("epoch-root")
	rootCmd.AddCommand(pr4b0R1EpochCmd)
}

func runPR4B0R1Epoch(operation, configPath, epochRoot string) error {
	if _, ok := pr4b0R1EpochOperations[operation]; !ok {
		return errors.New("unknown epoch operation")
	}
	data, err := readEpochAuthorityFile(configPath)
	if err != nil {
		return err
	}
	config, err := epochorchestrator.DecodeConfig(data)
	if err != nil {
		return fmt.Errorf("decode epoch configuration: %w", err)
	}
	orchestrator, err := epochorchestrator.New(epochRoot, config)
	if err != nil {
		return err
	}
	switch operation {
	case "preflight":
		status, runErr := orchestrator.Preflight()
		fmt.Fprintln(os.Stdout, status)
		return runErr
	case "register-protocol":
		return orchestrator.CommitProtocol()
	case "register-research-identity":
		return orchestrator.RegisterIdentity()
	case "reserve-holdout":
		return orchestrator.ReserveHoldout()
	case "authorize-development-set":
		return orchestrator.AuthorizeDevelopmentSet()
	case "run-development-set":
		return orchestrator.RunDevelopmentSet()
	case "seal-development-set":
		return orchestrator.SealDevelopmentSet()
	case "derive-nominee":
		return orchestrator.DeriveNominee()
	case "authorize-validation-set":
		return orchestrator.AuthorizeValidationSet()
	case "run-validation-set":
		return orchestrator.RunValidationSet()
	case "seal-validation-set":
		return orchestrator.SealValidationSet()
	case "freeze-candidate":
		return orchestrator.FreezeCandidate()
	case "authorize-final-holdout":
		return orchestrator.AuthorizeFinalHoldout()
	case "run-final-holdout":
		return orchestrator.RunFinalHoldout()
	case "seal-final-holdout":
		return orchestrator.SealFinalHoldout()
	case "closeout":
		return orchestrator.Closeout()
	case "status", "resume":
		status, statusErr := orchestrator.Status()
		if statusErr != nil {
			return statusErr
		}
		encoded, encodeErr := json.Marshal(status)
		if encodeErr != nil {
			return encodeErr
		}
		fmt.Fprintln(os.Stdout, string(encoded))
		return nil
	}
	return errors.New("unreachable epoch operation")
}

func readEpochAuthorityFile(path string) ([]byte, error) {
	if path == "" || filepath.Clean(path) != path {
		return nil, errors.New("epoch configuration path must be explicit and normalized")
	}
	info, err := os.Lstat(path)
	if err != nil {
		return nil, err
	}
	if info.Mode()&os.ModeSymlink != 0 || !info.Mode().IsRegular() {
		return nil, errors.New("epoch configuration must be a nonsymlink regular file")
	}
	return os.ReadFile(path)
}
