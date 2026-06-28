package app

import (
	"github.com/spf13/cobra"
)

var (
	efabChunks   string
	efabFeatures string
	efabSymbols  string
	efabFrom     string
	efabTo       string
	efabOut      string
)

var evaluateFundingAlphaBaselinesCmd = &cobra.Command{
	Use:   "evaluate-funding-alpha-baselines",
	Short: "Evaluate funding alpha baselines",
	RunE: func(cmd *cobra.Command, args []string) error {
		// implement me
		return nil
	},
}

func init() {
	evaluateFundingAlphaBaselinesCmd.Flags().StringVar(&efabChunks, "chunks", "", "chunks dir")
	evaluateFundingAlphaBaselinesCmd.Flags().StringVar(&efabFeatures, "features", "", "features dir")
	evaluateFundingAlphaBaselinesCmd.Flags().StringVar(&efabSymbols, "symbols", "", "symbols")
	evaluateFundingAlphaBaselinesCmd.Flags().StringVar(&efabFrom, "from", "", "from")
	evaluateFundingAlphaBaselinesCmd.Flags().StringVar(&efabTo, "to", "", "to")
	evaluateFundingAlphaBaselinesCmd.Flags().StringVar(&efabOut, "out", "", "out file")
	rootCmd.AddCommand(evaluateFundingAlphaBaselinesCmd)
}
