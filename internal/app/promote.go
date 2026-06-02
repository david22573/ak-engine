package app

import (
	"errors"

	"github.com/spf13/cobra"
)

var promoteCmd = &cobra.Command{
	Use:   "promote",
	Short: "Promote a walk-forward run (not implemented)",
	RunE: func(cmd *cobra.Command, args []string) error {
		return errors.New("promote is not implemented")
	},
}

func init() {
	rootCmd.AddCommand(promoteCmd)
}
