package app

import (
	"errors"

	"github.com/spf13/cobra"
)

var backtestCmd = &cobra.Command{
	Use:   "backtest",
	Short: "Run a backtest simulation (not implemented)",
	RunE: func(cmd *cobra.Command, args []string) error {
		return errors.New("backtest is not implemented")
	},
}

func init() {
	rootCmd.AddCommand(backtestCmd)
}
