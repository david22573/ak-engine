package app

import (
	"errors"

	"github.com/spf13/cobra"
)

var walkforwardCmd = &cobra.Command{
	Use:   "walk-forward",
	Short: "Run a walk-forward optimization (not implemented)",
	RunE: func(cmd *cobra.Command, args []string) error {
		return errors.New("walk-forward is not implemented")
	},
}

func init() {
	rootCmd.AddCommand(walkforwardCmd)
}
