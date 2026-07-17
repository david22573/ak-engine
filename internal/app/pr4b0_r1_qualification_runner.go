package app

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/david22573/ak-engine/internal/qualificationrunner"
	"github.com/spf13/cobra"
)

var (
	pr4b0R1RunnerRequestPath  string
	pr4b0R1RunnerArtifactPath string
	pr4b0R1RunnerOutputPath   string
)

var pr4b0R1QualificationRunnerCmd = &cobra.Command{
	Use:   "pr4b0-r1-qualification-runner",
	Short: "Run the registration-enforcing PR4B0-R1 qualification path",
	RunE: func(cmd *cobra.Command, args []string) error {
		return runPR4B0R1QualificationRunner(pr4b0R1RunnerRequestPath, pr4b0R1RunnerArtifactPath, pr4b0R1RunnerOutputPath)
	},
}

func init() {
	pr4b0R1QualificationRunnerCmd.Flags().StringVar(&pr4b0R1RunnerRequestPath, "request", "", "strict canonical execution request")
	pr4b0R1QualificationRunnerCmd.Flags().StringVar(&pr4b0R1RunnerArtifactPath, "partition-artifact", "", "registered partition artifact; prohibited in verify mode")
	pr4b0R1QualificationRunnerCmd.Flags().StringVar(&pr4b0R1RunnerOutputPath, "out", "", "output readiness or result artifact")
	_ = pr4b0R1QualificationRunnerCmd.MarkFlagRequired("request")
	_ = pr4b0R1QualificationRunnerCmd.MarkFlagRequired("out")
	rootCmd.AddCommand(pr4b0R1QualificationRunnerCmd)
}

func runPR4B0R1QualificationRunner(requestPath, artifactPath, outputPath string) error {
	requestBytes, err := readQualificationAuthorityFile(requestPath)
	if err != nil {
		return err
	}
	request, err := qualificationrunner.DecodeExecutionRequestJSON(requestBytes)
	if err != nil {
		return fmt.Errorf("decode execution request: %w", err)
	}
	if request.Mode == qualificationrunner.ModeVerify {
		if artifactPath != "" {
			return errors.New("verify mode prohibits partition artifacts")
		}
		_, readiness, err := qualificationrunner.Verify(request)
		if err != nil {
			return err
		}
		data, err := qualificationrunner.EncodeReadinessArtifact(readiness)
		if err != nil {
			return err
		}
		return atomicWriteFile(outputPath, data, 0o644)
	}
	if artifactPath == "" {
		return errors.New("execution mode requires a RIF-consumed registered partition artifact")
	}
	artifactBytes, err := readQualificationAuthorityFile(artifactPath)
	if err != nil {
		return err
	}
	result, err := qualificationrunner.Execute(request, artifactBytes)
	if err != nil {
		return err
	}
	data, err := qualificationrunner.EncodeResultArtifact(result)
	if err != nil {
		return err
	}
	return atomicWriteFile(outputPath, data, 0o644)
}

func readQualificationAuthorityFile(path string) ([]byte, error) {
	if path == "" || filepath.Clean(path) != path {
		return nil, errors.New("authority file path must be explicit and normalized")
	}
	info, err := os.Lstat(path)
	if err != nil {
		return nil, err
	}
	if info.Mode()&os.ModeSymlink != 0 || !info.Mode().IsRegular() {
		return nil, errors.New("authority file must be a nonsymlink regular file")
	}
	return os.ReadFile(path)
}
