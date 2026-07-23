package app

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/david22573/ak-engine/internal/papersignal"
	"github.com/spf13/cobra"
)

var (
	pssrJournal   string
	pssrCandidate string
	pssrOut       string
	pssrJsonOut   string
)

var paperShadowReadinessCmd = &cobra.Command{
	Use:   "paper-shadow-readiness",
	Short: "Generate a shadow-readiness report from forward paper outcomes",
	RunE: func(cmd *cobra.Command, args []string) error {
		rows, err := papersignal.ReadJournal(pssrJournal)
		if err != nil {
			if os.IsNotExist(err) {
				return fmt.Errorf("journal not found: %s", pssrJournal)
			}
			return err
		}
		rep := buildShadowReadinessReport(rows, pssrCandidate, time.Now().UTC())

		if err := os.MkdirAll(filepath.Dir(pssrJsonOut), 0755); err != nil {
			return err
		}
		if err := os.MkdirAll(filepath.Dir(pssrOut), 0755); err != nil {
			return err
		}
		data, err := json.MarshalIndent(rep, "", "  ")
		if err != nil {
			return err
		}
		if err := os.WriteFile(pssrJsonOut, data, 0644); err != nil {
			return err
		}
		if err := os.WriteFile(pssrOut, []byte(renderShadowReadinessMarkdown(rep)), 0644); err != nil {
			return err
		}
		fmt.Printf("Readiness for %s: %s (%s)\n", rep.CandidateID, rep.ReadinessLabel, rep.SampleSizeLabel)
		return nil
	},
}

func buildShadowReadinessReport(rows []papersignal.PaperJournalRow, candidateFilter string, nowUTC time.Time) papersignal.ShadowReadinessReport {
	rep := papersignal.ShadowReadinessReport{
		SchemaVersion:       "1.0",
		GeneratedAtUTC:      nowUTC.UTC().Format(time.RFC3339),
		CandidateID:         firstNonEmpty(candidateFilter, "ALL"),
		OutcomeDistribution: make(map[string]int),
		RIFBlockSummary:     make(map[string]int),
		Blockers:            []string{},
	}

	var returns []float64
	for _, row := range rows {
		if candidateFilter != "" && row.CandidateID != candidateFilter {
			continue
		}
		if candidateFilter == "" && rep.CandidateID == "ALL" && row.CandidateID != "" {
			rep.CandidateID = row.CandidateID
		}
		rep.TotalSignals++
		if papersignal.IsBlockedStatus(row.SignalStatus) {
			rep.BlockedSignals++
			key := firstNonEmpty(row.SignalReason, string(row.SignalStatus))
			if len(row.RIFWarnings) > 0 {
				key = row.RIFWarnings[0]
			}
			rep.RIFBlockSummary[key]++
			continue
		}
		if papersignal.IsActionableStatus(row.SignalStatus) || row.SignalStatus == "" {
			rep.AllowedSignals++
		}
		switch row.OutcomeStatus {
		case papersignal.OutcomePending:
			rep.PendingSignals++
		case "":
		case papersignal.OutcomeInsufficientData:
			rep.OutcomeDistribution[string(row.OutcomeStatus)]++
		default:
			rep.GradedSignals++
			rep.OutcomeDistribution[string(row.OutcomeStatus)]++
			if row.OutcomeReturnBPS != nil {
				returns = append(returns, *row.OutcomeReturnBPS)
			}
			if row.MaxAdverseExcursionBPS != nil {
				rep.MaxAdverseExcursion = maxFloatPtr(rep.MaxAdverseExcursion, *row.MaxAdverseExcursionBPS)
			}
			if row.MaxFavorableExcursionBPS != nil {
				rep.MaxFavorableExcursion = maxFloatPtr(rep.MaxFavorableExcursion, *row.MaxFavorableExcursionBPS)
			}
		}
	}

	if rep.GradedSignals > 0 {
		wins := rep.OutcomeDistribution[string(papersignal.OutcomeLongTPFirst)] + rep.OutcomeDistribution[string(papersignal.OutcomeShortTPFirst)]
		rep.WinRatePaper = float64(wins) / float64(rep.GradedSignals)
	}
	if len(returns) > 0 {
		var total float64
		for _, ret := range returns {
			total += ret
		}
		expectancy := total / float64(len(returns))
		rep.ExpectancyPaper = &expectancy
	}

	applyShadowReadinessRules(&rep)
	return rep
}

func applyShadowReadinessRules(rep *papersignal.ShadowReadinessReport) {
	if rep.GradedSignals < 30 {
		rep.SampleSizeLabel = papersignal.SampleInsufficient
	} else if rep.GradedSignals < 100 {
		rep.SampleSizeLabel = papersignal.SampleEarly
	} else {
		rep.SampleSizeLabel = papersignal.SampleReady
	}

	if rep.TotalSignals > 0 && rep.AllowedSignals == 0 && rep.BlockedSignals > 0 {
		rep.ReadinessLabel = papersignal.ReadinessBlockedByRIF
		rep.Blockers = append(rep.Blockers, "All observed signals are blocked by RIF")
		rep.Recommendation = "Resolve RIF blockers before collecting additional paper observations."
		return
	}
	if rep.GradedSignals < 30 {
		rep.ReadinessLabel = papersignal.ReadinessBlockedBySampleSize
		rep.Blockers = append(rep.Blockers, "Less than 30 graded paper signals")
		rep.Recommendation = "Continue paper-only forward observation until at least 30 graded outcomes exist."
		return
	}
	if rep.GradedSignals < 100 {
		if paperResultsClearlyBad(rep) {
			rep.ReadinessLabel = papersignal.ReadinessBlockedByResults
			rep.Blockers = append(rep.Blockers, "Paper results are negative in the early sample")
			rep.Recommendation = "Continue paper observation only if research review accepts the weak early results."
			return
		}
		rep.ReadinessLabel = papersignal.ReadinessContinuePaper
		rep.Recommendation = "Continue paper-only observation toward 100 graded outcomes."
		return
	}
	if paperResultsSupportShadow(rep) && rep.BlockedSignals == 0 {
		rep.ReadinessLabel = papersignal.ReadinessShadowCandidate
		rep.Recommendation = "Candidate may be evaluated for shadow mode only."
		return
	}
	if rep.BlockedSignals > 0 {
		rep.ReadinessLabel = papersignal.ReadinessBlockedByRIF
		rep.Blockers = append(rep.Blockers, "RIF-blocked signals are present")
		rep.Recommendation = "Continue paper observation after resolving RIF blockers."
		return
	}
	rep.ReadinessLabel = papersignal.ReadinessBlockedByResults
	rep.Blockers = append(rep.Blockers, "Paper outcomes do not meet shadow-readiness thresholds")
	rep.Recommendation = "Do not advance beyond paper observation."
}

func paperResultsClearlyBad(rep *papersignal.ShadowReadinessReport) bool {
	if rep.WinRatePaper < 0.30 {
		return true
	}
	return rep.ExpectancyPaper != nil && *rep.ExpectancyPaper < 0
}

func paperResultsSupportShadow(rep *papersignal.ShadowReadinessReport) bool {
	if rep.WinRatePaper < 0.50 {
		return false
	}
	return rep.ExpectancyPaper == nil || *rep.ExpectancyPaper > 0
}

func maxFloatPtr(current *float64, next float64) *float64 {
	if current == nil || next > *current {
		value := next
		return &value
	}
	return current
}

func renderShadowReadinessMarkdown(rep papersignal.ShadowReadinessReport) string {
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("# Paper Shadow Readiness Report: %s\n\n", rep.CandidateID))
	sb.WriteString(fmt.Sprintf("- Total signals: %d\n", rep.TotalSignals))
	sb.WriteString(fmt.Sprintf("- Allowed signals: %d\n", rep.AllowedSignals))
	sb.WriteString(fmt.Sprintf("- Blocked signals: %d\n", rep.BlockedSignals))
	sb.WriteString(fmt.Sprintf("- Graded signals: %d\n", rep.GradedSignals))
	sb.WriteString(fmt.Sprintf("- Pending signals: %d\n", rep.PendingSignals))
	sb.WriteString(fmt.Sprintf("- Win rate paper: %.4f\n", rep.WinRatePaper))
	if rep.ExpectancyPaper != nil {
		sb.WriteString(fmt.Sprintf("- Expectancy paper bps: %.6f\n", *rep.ExpectancyPaper))
	}
	sb.WriteString(fmt.Sprintf("- Sample size label: `%s`\n", rep.SampleSizeLabel))
	sb.WriteString(fmt.Sprintf("- Readiness label: `%s`\n", rep.ReadinessLabel))
	sb.WriteString(fmt.Sprintf("- Recommendation: %s\n\n", rep.Recommendation))
	if len(rep.Blockers) > 0 {
		sb.WriteString("## Blockers\n")
		for _, blocker := range rep.Blockers {
			sb.WriteString(fmt.Sprintf("- %s\n", blocker))
		}
		sb.WriteString("\n")
	}
	if len(rep.RIFBlockSummary) > 0 {
		sb.WriteString("## RIF Blocks\n")
		for _, key := range sortedPaperMapKeys(rep.RIFBlockSummary) {
			sb.WriteString(fmt.Sprintf("- %s: %d\n", key, rep.RIFBlockSummary[key]))
		}
	}
	return sb.String()
}

func sortedPaperMapKeys(values map[string]int) []string {
	keys := make([]string, 0, len(values))
	for key := range values {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}

func init() {
	paperShadowReadinessCmd.Flags().StringVar(&pssrJournal, "journal", "runs/paper/signals/paper_signal_journal.jsonl", "Journal path")
	paperShadowReadinessCmd.Flags().StringVar(&pssrCandidate, "candidate", "", "Candidate ID (optional)")
	paperShadowReadinessCmd.Flags().StringVar(&pssrOut, "out", "runs/reports/paper_shadow_readiness.md", "MD output")
	paperShadowReadinessCmd.Flags().StringVar(&pssrJsonOut, "json-out", "runs/reports/paper_shadow_readiness.json", "JSON output")
	rootCmd.AddCommand(paperShadowReadinessCmd)
}
