package app

import (
	"encoding/json"
	"fmt"

	"github.com/spf13/cobra"
)

var (
	efcSymbol      string
	efcMonth       string
	efcFeatureFile string
	efcContextFile string
	efcChunksDir   string
	efcIn          string
)

var evaluateFundingChunkCmd = &cobra.Command{
	Use:   "evaluate-funding-chunk",
	Short: "Evaluate one funding chunk into real event rows",
	RunE: func(cmd *cobra.Command, args []string) error {
		featureFile := efcFeatureFile
		if featureFile == "" {
			featureFile = efcIn
		}
		summary, _, err := evaluateFundingChunkFiles(fundingChunkConfig{
			Symbol:      efcSymbol,
			Month:       efcMonth,
			FeatureFile: featureFile,
			ContextFile: efcContextFile,
			ChunksDir:   efcChunksDir,
		})
		out, _ := json.Marshal(summary)
		fmt.Println(string(out))
		return err
	},
}

func init() {
	evaluateFundingChunkCmd.Flags().StringVar(&efcSymbol, "symbol", "", "symbol")
	evaluateFundingChunkCmd.Flags().StringVar(&efcMonth, "month", "", "month")
	evaluateFundingChunkCmd.Flags().StringVar(&efcFeatureFile, "features", "", "feature funding JSON input")
	evaluateFundingChunkCmd.Flags().StringVar(&efcContextFile, "context", "", "regime context JSON input")
	evaluateFundingChunkCmd.Flags().StringVar(&efcChunksDir, "chunks-dir", "", "report chunks output directory")
	evaluateFundingChunkCmd.Flags().StringVar(&efcIn, "in", "", "legacy alias for --features")
	_ = evaluateFundingChunkCmd.MarkFlagRequired("symbol")
	_ = evaluateFundingChunkCmd.MarkFlagRequired("month")
	rootCmd.AddCommand(evaluateFundingChunkCmd)
}
