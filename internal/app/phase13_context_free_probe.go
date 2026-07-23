package app

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"time"

	"github.com/david22573/ak-engine/internal/data"
	"github.com/david22573/ak-engine/internal/features"
	"github.com/spf13/cobra"
)

var (
	p13cfpWorkdir           string
	p13cfpEmitCompactEvents bool
)

const phase13CFPFamily = "ContextFreeLinkLocalProbe"
const phase13CFPHorizon = "60m" // 60 minutes horizon

type Phase13ProofReport struct {
	Phase                  string             `json:"phase"`
	ExecutiveVerdict       string             `json:"executive_verdict"`
	LocalDataUsed          string             `json:"local_data_used"`
	CandidateDefinition    string             `json:"candidate_definition"`
	PreEntryFieldsUsed     []string           `json:"pre_entry_fields_used"`
	HorizonSelected        string             `json:"horizon_selected"`
	EventCount             int                `json:"event_count"`
	ClusterCount           int                `json:"cluster_count"`
	MaxSerializedEventSize int                `json:"max_serialized_event_size"`
	CompactJSONLValidation string             `json:"compact_jsonl_validation_result"`
	AggregatorConsumption  string             `json:"aggregator_consumption_result"`
	CostStress             map[string]float64 `json:"cost_stress_results"`
	Concentration          map[string]int     `json:"concentration_results"`
	ClusterAudit           string             `json:"cluster_audit_results"`
	LeaveOneOutStatus      string             `json:"leave_one_out_status"`
	FilterSimulation       string             `json:"filter_simulation_results"`
	Limitations            string             `json:"limitations"`
	ProvesCompactPipeline  bool               `json:"proves_compact_pipeline"`
	ResearchMerit          string             `json:"research_merit"`
	RecommendedPhase13_1   string             `json:"recommended_phase_13_1"`
}

var phase13ContextFreeProbeCmd = &cobra.Command{
	Use:   "phase13-context-free-probe",
	Short: "Run context-free probe on LINKUSDT to prove compact event pipeline",
	RunE: func(cmd *cobra.Command, args []string) error {
		jsonlOut := filepath.Join("runs", "reports", "phase13_0_context_free_compact_events.jsonl")
		jsonOut := filepath.Join("runs", "reports", "phase13_0_context_free_compact_candidate_proof.json")
		mdOut := filepath.Join("runs", "reports", "phase13_0_context_free_compact_candidate_proof.md")

		report, err := runPhase13ContextFreeProbe(cmd.Context(), p13cfpWorkdir, p13cfpEmitCompactEvents, jsonlOut)
		if err != nil {
			return err
		}

		if err := writeJSONFile(jsonOut, report); err != nil {
			return err
		}

		mdStr := renderPhase13CFPMarkdown(report)
		if err := os.WriteFile(mdOut, []byte(mdStr), 0644); err != nil {
			return err
		}

		fmt.Printf("Phase 13 context-free probe report written to %s\n", mdOut)
		return nil
	},
}

func init() {
	phase13ContextFreeProbeCmd.Flags().StringVar(&p13cfpWorkdir, "workdir", defaultHistorianWorkdir, "local historian workdir")
	phase13ContextFreeProbeCmd.Flags().BoolVar(&p13cfpEmitCompactEvents, "emit-compact-events", false, "emit retained events in compact JSONL format")
	rootCmd.AddCommand(phase13ContextFreeProbeCmd)
}

func runPhase13ContextFreeProbe(ctx context.Context, workdir string, emitCompactEvents bool, jsonlOut string) (Phase13ProofReport, error) {
	report := Phase13ProofReport{
		Phase:                 "Phase 13.0",
		LocalDataUsed:         "LINKUSDT 1m 2024-01 local parquet only",
		CandidateDefinition:   "Momentum breakout context-free probe",
		PreEntryFieldsUsed:    []string{"trend_regime", "volatility_bucket", "funding_bucket"},
		HorizonSelected:       phase13CFPHorizon,
		Limitations:           "Restricted to local context-free data",
		ProvesCompactPipeline: false,
		ResearchMerit:         "Minimal, mainly for pipeline verification",
		RecommendedPhase13_1:  "Proceed to distributed context-free evaluation or acquire missing context data",
	}

	caps := CandidateCapabilities{
		CandidateName:                 "ContextFreeLinkLocalProbe",
		FamilyName:                    phase13CFPFamily,
		SupportedSymbols:              []string{"LINKUSDT"},
		SupportedIntervals:            []string{"1m"},
		SupportedHorizons:             []string{phase13CFPHorizon},
		RequiresBTCContext:            false,
		RequiresETHContext:            false,
		RequiresFundingContext:        false,
		RequiresVolumeContext:         false,
		RequiresClusterContext:        true,
		SupportsCompactEmission:       true,
		ContextFreeModeAllowed:        true,
		AllowedMissingContextBehavior: AllowMissingContext,
		IsResearchOnly:                true,
		IsPromotable:                  false,
	}

	if err := ValidateCandidateInputs(caps, false, false, false, emitCompactEvents); err != nil {
		report.ExecutiveVerdict = "PHASE13_CONTEXT_FREE_PROOF_BLOCKED_INVALID_CAPS"
		return report, fmt.Errorf("capability validation failed: %w", err)
	}

	src := data.NewLocalParquetSource()
	req := data.CandleRequest{
		Source:   "local-parquet",
		Path:     workdir,
		Market:   "futures-um",
		Symbol:   "LINKUSDT",
		Interval: "1m",
		From:     time.Date(2023, 11, 1, 0, 0, 0, 0, time.UTC),
		To:       time.Date(2024, 2, 2, 0, 0, 0, 0, time.UTC),
	}

	candles, err := src.LoadCandles(ctx, req)
	if err != nil {
		report.ExecutiveVerdict = "PHASE13_CONTEXT_FREE_PROOF_BLOCKED_NO_DATA"
		return report, nil
	}
	if len(candles) == 0 {
		report.ExecutiveVerdict = "PHASE13_CONTEXT_FREE_PROOF_BLOCKED_NO_DATA"
		return report, nil
	}

	rows, err := features.BuildRows(candles, features.BuildOptions{
		Market:     req.Market,
		Symbol:     req.Symbol,
		Interval:   req.Interval,
		ContextBTC: nil,
		ContextETH: nil,
	})
	if err != nil {
		return report, err
	}

	var writer *CompactEventWriter
	if emitCompactEvents {
		writer = NewCompactEventWriter()
	}
	emitter := NewCompactEventEmitter(CompactEventEmissionConfig{
		Enabled: emitCompactEvents,
		Writer:  writer,
	})

	horizonMS := int64(60 * 60 * 1000)

	var maxEventSize int

	for i := 20; i < len(rows); i++ {
		row := rows[i]

		// Ensure only Jan 2024
		t := time.UnixMilli(row.EventTimeMS).UTC()
		if t.Year() != 2024 || t.Month() != 1 {
			continue
		}

		if row.Warmup || row.AvailableAtMS < row.EventTimeMS {
			continue
		}

		// Simple context-free entry logic: trend is UP, mid volatility
		trendRegime := "other"
		if row.Close > row.EMA50 && row.EMA50 > row.EMA200 {
			trendRegime = "up"
		}
		volBucket := "other"
		if row.RealizedVol60 > 0.002 && row.RealizedVol60 < 0.005 {
			volBucket = "mid"
		}

		if trendRegime != "up" || volBucket != "mid" {
			continue
		}

		// We need a pull-back to enter
		if row.Close >= row.EMA20 {
			continue
		}

		futureClose, ok := findFutureClose(rows, i, horizonMS)
		if !ok || futureClose <= 0 {
			continue
		}

		retBps := (futureClose - row.Close) / row.Close * 10000.0

		snapshot := CandidateEventSnapshot{
			CandidateFamily: phase13CFPFamily,
			Symbol:          row.Symbol,
			Side:            "long",
			Horizon:         phase13CFPHorizon,
			EventTimeMS:     row.EventTimeMS,
			PreEntry: PreEntryContextSnapshot{
				TrendRegime:      trendRegime,
				VolatilityBucket: volBucket,
				FundingBucket:    "neutral",
			},
			Cluster: ClusterContextSnapshot{
				Key:       fmt.Sprintf("%s|long|%d", row.Symbol, row.EventTimeMS),
				Timestamp: row.EventTimeMS / 1000,
				Size:      1,
				Ordinal:   1,
			},
			Cost: CostStressSnapshot{
				GrossOutcomeBps: retBps,
				NetOutcome5Bps:  retBps - 5.0,
				NetOutcome75Bps: retBps - 7.5,
				NetOutcome10Bps: retBps - 10.0,
				Win5Bps:         (retBps - 5.0) > 0,
				Win75Bps:        (retBps - 7.5) > 0,
				Win10Bps:        (retBps - 10.0) > 0,
			},
			Diagnostic: map[string]interface{}{
				"close": row.Close,
			},
		}

		if emitCompactEvents {
			if err := emitter.EmitCompactEvent(snapshot); err != nil {
				return report, fmt.Errorf("emit error: %w", err)
			}
			// Size tracking (we serialize manually here just to get the max size, which is inefficient but OK for proof)
			ev := CompactRetainedEvent{
				CandidateID:    phase13CFPFamily + "Long" + phase13CFPHorizon,
				Symbol:         snapshot.Symbol,
				Side:           1,
				EventTimestamp: snapshot.EventTimeMS / 1000,
				Horizon:        60,
				PreEntry: PreEntryContext{
					TrendRegime:      snapshot.PreEntry.TrendRegime,
					VolatilityBucket: snapshot.PreEntry.VolatilityBucket,
					FundingBucket:    snapshot.PreEntry.FundingBucket,
				},
				Cluster: ClusterContext{
					Key:       snapshot.Cluster.Key,
					Timestamp: snapshot.Cluster.Timestamp,
					Size:      1,
					Ordinal:   1,
				},
			}
			ev.DeriveTimeFields()
			b, _ := json.Marshal(ev)
			if len(b) > maxEventSize {
				maxEventSize = len(b)
			}
		}
	}

	report.MaxSerializedEventSize = maxEventSize

	if emitCompactEvents && writer != nil {
		out, err := writer.ToJSONL()
		if err != nil {
			report.ExecutiveVerdict = "PHASE13_CONTEXT_FREE_PROOF_FAILED_VALIDATION"
			report.CompactJSONLValidation = err.Error()
			return report, nil
		}

		report.CompactJSONLValidation = "PASS"

		if err := os.MkdirAll(filepath.Dir(jsonlOut), 0755); err != nil {
			return report, err
		}
		if err := os.WriteFile(jsonlOut, []byte(out), 0644); err != nil {
			return report, err
		}

		// Run Aggregator
		agg := NewAggregator()
		err = agg.LoadJSONL(bytes.NewReader([]byte(out)))
		if err != nil {
			report.ExecutiveVerdict = "PHASE13_CONTEXT_FREE_PROOF_FAILED_VALIDATION"
			report.AggregatorConsumption = err.Error()
			return report, nil
		}

		report.AggregatorConsumption = "PASS"

		summary := agg.FullSummary()
		report.EventCount = summary.EventCount
		report.ClusterCount = summary.ClusterCount
		report.ClusterAudit = fmt.Sprintf("Cluster size: %.2f avg, max %d", summary.AverageClusterSize, summary.MaxClusterSize)

		report.CostStress = map[string]float64{
			"net_5_bps":  summary.Net5Bps,
			"net_75_bps": summary.Net75Bps,
			"net_10_bps": summary.Net10Bps,
			"pf_5_bps":   summary.ProfitFactor5,
			"pf_75_bps":  summary.ProfitFactor75,
			"pf_10_bps":  summary.ProfitFactor10,
		}

		report.Concentration = make(map[string]int)
		for k, v := range summary.SymbolConcentration {
			report.Concentration["symbol_"+k] = v
		}
		for k, v := range summary.MonthConcentration {
			report.Concentration["month_"+k] = v
		}
		for k, v := range summary.QuarterConcentration {
			report.Concentration["quarter_"+k] = v
		}

		// Leave-one-out
		monthSummaries := agg.LeaveOneMonthOutSummary()
		if len(monthSummaries) <= 1 {
			report.LeaveOneOutStatus = "INSUFFICIENT_SEGMENTS"
		} else {
			report.LeaveOneOutStatus = "COMPLETED"
		}

		// Simulation
		f := EventFilter{IncludeRegime: "up"}
		fSummary, err := agg.SimulateFilter(f)
		if err != nil {
			report.FilterSimulation = err.Error()
		} else {
			report.FilterSimulation = fmt.Sprintf("Filtered event count: %d", fSummary.EventCount)
		}

		if summary.EventCount == 0 {
			report.ExecutiveVerdict = "PHASE13_CONTEXT_FREE_PROOF_BLOCKED_NO_EVENTS"
		} else {
			if summary.ProfitFactor10 > 1.2 && summary.EventCount > 100 {
				report.ExecutiveVerdict = "PHASE13_CONTEXT_FREE_PROOF_PASSED_RESEARCH_LEAD"
				report.ResearchMerit = "Strong PF under 10bps cost"
			} else {
				report.ExecutiveVerdict = "PHASE13_CONTEXT_FREE_PROOF_PASSED_INFRA_ONLY"
				report.ResearchMerit = "No proven edge, infra success only"
			}
			report.ProvesCompactPipeline = true
		}
	} else {
		report.ExecutiveVerdict = "PHASE13_CONTEXT_FREE_PROOF_BLOCKED_NO_EVENTS"
	}

	return report, nil
}

func findFutureClose(rows []features.Row, startIdx int, offsetMS int64) (float64, bool) {
	target := rows[startIdx].EventTimeMS + offsetMS
	idx := sort.Search(len(rows)-startIdx, func(i int) bool { return rows[startIdx+i].EventTimeMS >= target })
	j := startIdx + idx
	if j >= len(rows) || rows[j].EventTimeMS < target {
		return 0, false
	}
	return rows[j].Close, true
}

func renderPhase13CFPMarkdown(report Phase13ProofReport) string {
	var b bytes.Buffer
	b.WriteString("# Phase 13.0 Context-Free Compact Candidate Pipeline Proof\n\n")
	b.WriteString(fmt.Sprintf("**Executive Verdict**: %s\n\n", report.ExecutiveVerdict))
	b.WriteString(fmt.Sprintf("- **Local Data Used**: %s\n", report.LocalDataUsed))
	b.WriteString(fmt.Sprintf("- **Candidate**: %s\n", report.CandidateDefinition))
	b.WriteString(fmt.Sprintf("- **Horizon**: %s\n", report.HorizonSelected))
	b.WriteString(fmt.Sprintf("- **Event Count**: %d\n", report.EventCount))
	b.WriteString(fmt.Sprintf("- **Cluster Count**: %d\n", report.ClusterCount))
	b.WriteString(fmt.Sprintf("- **Max Serialized Event Size**: %d bytes\n", report.MaxSerializedEventSize))
	b.WriteString(fmt.Sprintf("- **Compact JSONL Validation**: %s\n", report.CompactJSONLValidation))
	b.WriteString(fmt.Sprintf("- **Aggregator Consumption**: %s\n", report.AggregatorConsumption))
	b.WriteString(fmt.Sprintf("- **Cluster Audit**: %s\n", report.ClusterAudit))
	b.WriteString(fmt.Sprintf("- **Leave-One-Out Status**: %s\n", report.LeaveOneOutStatus))
	b.WriteString(fmt.Sprintf("- **Filter Simulation**: %s\n", report.FilterSimulation))
	b.WriteString(fmt.Sprintf("- **Pipeline Proved**: %v\n", report.ProvesCompactPipeline))
	b.WriteString(fmt.Sprintf("- **Limitations**: %s\n", report.Limitations))
	b.WriteString(fmt.Sprintf("- **Research Merit**: %s\n", report.ResearchMerit))
	b.WriteString(fmt.Sprintf("- **Recommended Phase 13.1**: %s\n\n", report.RecommendedPhase13_1))

	b.WriteString("## Cost Stress Results\n")
	for k, v := range report.CostStress {
		b.WriteString(fmt.Sprintf("- %s: %.2f\n", k, v))
	}

	b.WriteString("\n## Concentration Results\n")
	for k, v := range report.Concentration {
		b.WriteString(fmt.Sprintf("- %s: %d\n", k, v))
	}

	return b.String()
}
