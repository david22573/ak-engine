package app

import (
	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "ak-engine",
	Short: "AK Engine is a deterministic research, simulation, and decision kernel",
	Long:  `AK Engine is the deterministic research, simulation, strategy, and decision kernel for the AK ecosystem.`,
}

func Execute() error {
	return rootCmd.Execute()
}

func init() {
	// Add global flags if needed
}
