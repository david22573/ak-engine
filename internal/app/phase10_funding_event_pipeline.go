package app

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/davidmiguel22573/ak-engine/internal/features"
	"github.com/spf13/cobra"
)

var (
	p10fepWorkdir             string
	p10fepSymbols             string
	p10fepContextSymbols      string
	p10fepFrom                string
	p10fepTo                  string
	p10fepChunk               string
	p10fepMaxSymbols          int
	p10fepMaxMonths           int
	p10fepMaxRows             int
	p10fepMinFreeGB           float64
	p10fepDiskBudgetGB        float64
	p10fepRetainPolicy        string
	p10fepGcBetweenChunks     bool
	p10fepContinueOnError     bool
	p10fepForce               bool
	p10fepOut                 string
	p10fepEventFormat         string
	p10fepRetainEventDetail   bool
	p10fepSummaryOnlyAfterAgg bool
	p10fepMaxEventFileMB      int
)

const phase10FundingEventManifestPath = "runs/manifests/phase10_7e_funding_event_manifest.json"

type Phase10FundingEventManifest struct {
	Chunks    map[string]*Phase10FundingEventChunkStatus `json:"chunks"`
	UpdatedAt time.Time                                  `json:"updated_at,omitempty"`
}

type Phase10FundingEventChunkStatus struct {
	Symbol               string    `json:"symbol"`
	Month                string    `json:"month"`
	FeatureBuildStatus   string    `json:"feature_build_status"`
	RegimeClassifyStatus string    `json:"regime_classify_status"`
	FundingJoinStatus    string    `json:"funding_join_status"`
	EventEvalStatus      string    `json:"event_eval_status"`
	CleanupStatus        string    `json:"cleanup_status"`
	SummaryStatus        string    `json:"summary_status,omitempty"`
	FeatureRows          int       `json:"feature_rows"`
	RegimeRows           int       `json:"regime_rows"`
	FundingRows          int       `json:"funding_rows"`
	EventRows            int       `json:"event_rows"`
	DeletedHeavyFiles    []string  `json:"deleted_heavy_files"`
	BytesFreed           int64     `json:"bytes_freed"`
	ErrorIfFailed        string    `json:"error_if_failed,omitempty"`
	DiskFreeBeforeBytes  int64     `json:"disk_free_before_bytes"`
	DiskFreeAfterBytes   int64     `json:"disk_free_after_bytes"`
	DiskStatus           string    `json:"disk_status,omitempty"`
	EventFile            string    `json:"event_file,omitempty"`
	SummaryFile          string    `json:"summary_file,omitempty"`
	ContextSymbols       string    `json:"context_symbols,omitempty"`
	ContextStatus        string    `json:"context_status,omitempty"`
	CompletedAt          time.Time `json:"completed_at,omitempty"`
}

type phase10FundingEventPipelineConfig struct {
	RootDir             string
	Workdir             string
	Symbols             []string
	ContextSymbols      string
	Months              []string
	Chunk               string
	MaxSymbols          int
	MaxMonths           int
	MaxRows             int
	MinFreeGB           float64
	DiskBudgetGB        float64
	RetainPolicy        string
	GCBetweenChunks     bool
	ContinueOnError     bool
	Force               bool
	Out                 string
	EventFormat         string
	RetainEventDetail   bool
	SummaryOnlyAfterAgg bool
	MaxEventFileMB      int
	ManifestPath        string
	ReportsDir          string
	ChunksDir           string
}

type phase10FundingEventChunkPaths struct {
	FeatureContextFile string
	RegimeContextFile  string
	FundingFeatureFile string
	EventFile          string
	SummaryFile        string
	ContextAuditFile   string
	DiagnosticsFile    string
	AlphaSummaryFile   string
	ChunksDir          string
}

type phase10FundingEventPipelineSteps struct {
	BuildFeatureChunk   func(phase10FundingEventPipelineConfig, phase10FundingEventChunkPaths, *Phase10FundingEventChunkStatus) error
	ClassifyRegimeChunk func(phase10FundingEventPipelineConfig, phase10FundingEventChunkPaths, *Phase10FundingEventChunkStatus) error
	JoinFundingChunk    func(phase10FundingEventPipelineConfig, phase10FundingEventChunkPaths, *Phase10FundingEventChunkStatus) error
	EvaluateFunding     func(phase10FundingEventPipelineConfig, phase10FundingEventChunkPaths, *Phase10FundingEventChunkStatus) (FundingChunkSummary, error)
	CleanupChunk        func(phase10FundingEventPipelineConfig, phase10FundingEventChunkPaths, *Phase10FundingEventChunkStatus) error
}

type phase10FundingEventPipelineReport struct {
	Status                  string   `json:"status"`
	DetailedStatus          string   `json:"detailed_status"`
	SymbolsProcessed        []string `json:"symbols_processed"`
	MonthsProcessed         []string `json:"months_processed"`
	ChunksProcessed         int      `json:"chunks_processed"`
	InputChunksRebuilt      int      `json:"input_chunks_rebuilt"`
	EventFilesCreated       int      `json:"event_files_created"`
	RealEventRows           int      `json:"real_event_rows"`
	ZeroEventMonths         int      `json:"zero_event_months"`
	MissingInputMonths      int      `json:"missing_input_months"`
	HeavyFilesDeleted       int      `json:"heavy_files_deleted"`
	BytesFreed              int64    `json:"bytes_freed"`
	ManifestPath            string   `json:"manifest_path"`
	IntegrityAuditPath      string   `json:"integrity_audit_path"`
	LeaderboardPath         string   `json:"leaderboard_path"`
	NegativeFundingDeepPath string   `json:"negative_funding_deep_path"`
}

var phase10FundingEventPipelineCmd = &cobra.Command{
	Use:   "phase10-funding-event-pipeline",
	Short: "Build ephemeral monthly funding event chunks, evaluate, then clean heavy files",
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg := phase10FundingEventPipelineConfig{
			RootDir:             ".",
			Workdir:             resolveHistorianWorkdir(cmd, p10fepWorkdir),
			Symbols:             parseFundingSymbols(p10fepSymbols),
			ContextSymbols:      p10fepContextSymbols,
			Months:              parseFundingMonths(p10fepFrom, p10fepTo),
			Chunk:               p10fepChunk,
			MaxSymbols:          p10fepMaxSymbols,
			MaxMonths:           p10fepMaxMonths,
			MaxRows:             p10fepMaxRows,
			MinFreeGB:           p10fepMinFreeGB,
			DiskBudgetGB:        p10fepDiskBudgetGB,
			RetainPolicy:        p10fepRetainPolicy,
			GCBetweenChunks:     p10fepGcBetweenChunks,
			ContinueOnError:     p10fepContinueOnError,
			Force:               p10fepForce,
			Out:                 p10fepOut,
			EventFormat:         p10fepEventFormat,
			RetainEventDetail:   p10fepRetainEventDetail,
			SummaryOnlyAfterAgg: p10fepSummaryOnlyAfterAgg,
			MaxEventFileMB:      p10fepMaxEventFileMB,
		}
		report, err := runPhase10FundingEventPipeline(cfg, defaultPhase10FundingEventPipelineSteps())
		if report.ManifestPath != "" {
			fmt.Printf("phase10 funding event pipeline status=%s events=%d manifest=%s\n", report.Status, report.RealEventRows, report.ManifestPath)
		}
		return err
	},
}

func init() {
	phase10FundingEventPipelineCmd.Flags().StringVar(&p10fepWorkdir, "workdir", defaultHistorianWorkdir, "historian workdir")
	phase10FundingEventPipelineCmd.Flags().StringVar(&p10fepSymbols, "symbols", "", "comma-separated target symbols")
	phase10FundingEventPipelineCmd.Flags().StringVar(&p10fepContextSymbols, "context-symbols", "BTCUSDT,ETHUSDT", "comma-separated context symbols")
	phase10FundingEventPipelineCmd.Flags().StringVar(&p10fepFrom, "from", "", "from month YYYY-MM")
	phase10FundingEventPipelineCmd.Flags().StringVar(&p10fepTo, "to", "", "to month YYYY-MM")
	phase10FundingEventPipelineCmd.Flags().StringVar(&p10fepChunk, "chunk", "monthly", "chunk size")
	phase10FundingEventPipelineCmd.Flags().IntVar(&p10fepMaxSymbols, "max-symbols", 0, "max symbols this run; 0 means all")
	phase10FundingEventPipelineCmd.Flags().IntVar(&p10fepMaxMonths, "max-months", 0, "max months this run; 0 means all")
	phase10FundingEventPipelineCmd.Flags().IntVar(&p10fepMaxRows, "max-rows", 50000, "max rows per monthly chunk")
	phase10FundingEventPipelineCmd.Flags().Float64Var(&p10fepMinFreeGB, "min-free-gb", 5, "minimum free disk GB before each chunk")
	phase10FundingEventPipelineCmd.Flags().Float64Var(&p10fepDiskBudgetGB, "disk-budget-gb", 8, "artifact disk budget GB")
	phase10FundingEventPipelineCmd.Flags().StringVar(&p10fepRetainPolicy, "retain-policy", "reports_only", "retain policy")
	phase10FundingEventPipelineCmd.Flags().BoolVar(&p10fepGcBetweenChunks, "gc-between-chunks", false, "run GC between chunks")
	phase10FundingEventPipelineCmd.Flags().BoolVar(&p10fepContinueOnError, "continue-on-error", false, "continue on chunk errors")
	phase10FundingEventPipelineCmd.Flags().BoolVar(&p10fepForce, "force", false, "force rebuild and overwrite chunk reports")
	phase10FundingEventPipelineCmd.Flags().StringVar(&p10fepOut, "out", filepath.Join("runs", "reports", "phase10_7e_funding_event_pipeline.md"), "pipeline report markdown path")
	phase10FundingEventPipelineCmd.Flags().StringVar(&p10fepEventFormat, "event-format", "jsonl.gz", "event format (jsonl|jsonl.gz)")
	phase10FundingEventPipelineCmd.Flags().BoolVar(&p10fepRetainEventDetail, "retain-event-detail", true, "retain event detail for current chunk")
	phase10FundingEventPipelineCmd.Flags().BoolVar(&p10fepSummaryOnlyAfterAgg, "summary-only-after-aggregate", false, "allow deletion/compression of raw event JSONL after aggregate if requested")
	phase10FundingEventPipelineCmd.Flags().IntVar(&p10fepMaxEventFileMB, "max-event-file-mb", 100, "max event file size in MB")
	_ = phase10FundingEventPipelineCmd.MarkFlagRequired("symbols")
	_ = phase10FundingEventPipelineCmd.MarkFlagRequired("from")
	_ = phase10FundingEventPipelineCmd.MarkFlagRequired("to")
	rootCmd.AddCommand(phase10FundingEventPipelineCmd)
}

func defaultPhase10FundingEventPipelineSteps() phase10FundingEventPipelineSteps {
	return phase10FundingEventPipelineSteps{
		BuildFeatureChunk:   defaultPhase10BuildFeatureChunk,
		ClassifyRegimeChunk: defaultPhase10ClassifyRegimeChunk,
		JoinFundingChunk:    defaultPhase10JoinFundingChunk,
		EvaluateFunding:     defaultPhase10EvaluateFundingChunk,
		CleanupChunk:        defaultPhase10CleanupFundingChunk,
	}
}

func defaultPhase10BuildFeatureChunk(cfg phase10FundingEventPipelineConfig, paths phase10FundingEventChunkPaths, status *Phase10FundingEventChunkStatus) error {
	res := buildFeaturesResult{}
	args := phase10BuildFeatureArgs(cfg, paths, status)
	if err := runPhase10FundingSubcommand(cfg.RootDir, args, &res); err != nil {
		return err
	}
	status.FeatureRows = res.Rows
	status.FeatureBuildStatus = "DONE"
	return nil
}

func phase10BuildFeatureArgs(cfg phase10FundingEventPipelineConfig, paths phase10FundingEventChunkPaths, status *Phase10FundingEventChunkStatus) []string {
	return []string{
		"build-features",
		"--source", "local-parquet",
		"--path", cfg.Workdir,
		"--market", "futures-um",
		"--symbol", status.Symbol,
		"--interval", "1m",
		"--from", status.Month + "-01",
		"--to", monthLastDay(status.Month),
		"--context-symbols", effectiveFundingContextSymbols(status.Symbol, cfg.ContextSymbols),
		"--out", paths.FeatureContextFile,
		"--format", "json",
	}
}

func defaultPhase10ClassifyRegimeChunk(cfg phase10FundingEventPipelineConfig, paths phase10FundingEventChunkPaths, status *Phase10FundingEventChunkStatus) error {
	res := classifyRegimesResult{}
	args := []string{
		"classify-regimes",
		"--features", paths.FeatureContextFile,
		"--out", paths.RegimeContextFile,
		"--format", "json",
	}
	if err := runPhase10FundingSubcommand(cfg.RootDir, args, &res); err != nil {
		return err
	}
	status.RegimeRows = res.Labels
	status.RegimeClassifyStatus = "DONE"
	return nil
}

func defaultPhase10JoinFundingChunk(cfg phase10FundingEventPipelineConfig, paths phase10FundingEventChunkPaths, status *Phase10FundingEventChunkStatus) error {
	derivativePath := phase10FundingDerivativePath(cfg.Workdir, status.Symbol, status.Month)
	if _, err := os.Stat(derivativePath); err != nil {
		if !os.IsNotExist(err) {
			return err
		}
		rows, readErr := features.ReadRowsJSON(paths.FeatureContextFile)
		if readErr != nil {
			return readErr
		}
		if err := writeUnknownFundingFeatureRows(paths.FundingFeatureFile, rows); err != nil {
			return err
		}
		status.FundingJoinStatus = "DONE_UNKNOWN_FUNDING"
		return nil
	}

	args := []string{
		"join-research-features",
		"--features", paths.FeatureContextFile,
		"--derivatives", derivativePath,
		"--out", paths.FundingFeatureFile,
	}
	if err := runPhase10FundingSubcommand(cfg.RootDir, args, nil); err != nil {
		return err
	}
	status.FundingJoinStatus = "DONE"
	return nil
}

func defaultPhase10EvaluateFundingChunk(cfg phase10FundingEventPipelineConfig, paths phase10FundingEventChunkPaths, status *Phase10FundingEventChunkStatus) (FundingChunkSummary, error) {
	summary, _, err := evaluateFundingChunkFiles(fundingChunkConfig{
		Symbol:            status.Symbol,
		Month:             status.Month,
		FeatureFile:       paths.FundingFeatureFile,
		ContextFile:       paths.RegimeContextFile,
		ChunksDir:         paths.ChunksDir,
		EventFormat:       cfg.EventFormat,
		RetainEventDetail: cfg.RetainEventDetail,
		MaxEventFileMB:    cfg.MaxEventFileMB,
	})
	return summary, err
}

func defaultPhase10CleanupFundingChunk(cfg phase10FundingEventPipelineConfig, paths phase10FundingEventChunkPaths, status *Phase10FundingEventChunkStatus) error {
	if cfg.RetainPolicy != "reports_only" {
		status.CleanupStatus = "SKIPPED"
		return nil
	}
	for _, path := range []string{paths.FeatureContextFile, paths.RegimeContextFile, paths.FundingFeatureFile} {
		info, err := os.Stat(path)
		if err != nil {
			if os.IsNotExist(err) {
				continue
			}
			return err
		}
		size := info.Size()
		if err := os.Remove(path); err != nil {
			return err
		}
		status.DeletedHeavyFiles = append(status.DeletedHeavyFiles, path)
		status.BytesFreed += size
	}
	status.CleanupStatus = "DONE"
	return nil
}

func effectiveFundingContextSymbols(target, contextCSV string) string {
	target = strings.ToUpper(strings.TrimSpace(target))
	if target == "ETHUSDT" {
		return "BTCUSDT"
	}
	ctxList := contextSymbolListForTarget(target, contextCSV)
	if target == "BTCUSDT" {
		var safe []string
		for _, sym := range ctxList {
			if sym != "BTCUSDT" {
				safe = append(safe, sym)
			}
		}
		return strings.Join(safe, ",")
	}
	return strings.Join(ctxList, ",")
}

func writeUnknownFundingFeatureRows(path string, rows []features.Row) error {
	out := make([]ResearchFeatureRow, 0, len(rows))
	for _, row := range rows {
		out = append(out, ResearchFeatureRow{
			Row: row,
			Derivatives: ResearchDerivativeFeatures{
				FundingRateUnknown:        true,
				FundingRateZScoreUnknown:  true,
				FundingRateChangeUnknown:  true,
				OpenInterestChangeUnknown: true,
				TakerBuySellUnknown:       true,
				LongShortRatioUnknown:     true,
				TopTraderLongShortUnknown: true,
				PositioningUnknown:        true,
			},
		})
	}
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return err
	}
	data, err := json.Marshal(out)
	if err != nil {
		return err
	}
	return atomicWriteFile(path, data, 0644)
}

func verifyFundingEventOutputs(summaryFile, eventFile string) (FundingChunkSummary, []FundingEventRow, error) {
	events, err := readFundingEventsJSONL(eventFile)
	if err != nil {
		return FundingChunkSummary{}, nil, err
	}
	data, err := os.ReadFile(summaryFile)
	if err != nil {
		return FundingChunkSummary{}, nil, err
	}
	var summary FundingChunkSummary
	if err := json.Unmarshal(data, &summary); err != nil {
		return FundingChunkSummary{}, nil, err
	}
	if summary.EventCount != len(events) {
		return FundingChunkSummary{}, nil, fmt.Errorf("summary event_count %d != event rows %d", summary.EventCount, len(events))
	}
	return summary, events, nil
}

func runPhase10FundingSubcommand(root string, args []string, out interface{}) error {
	cmd := exec.Command(os.Args[0], args...)
	cmd.Dir = root
	var stdout bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stdout
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("%s failed: %s: %w", strings.Join(args, " "), strings.TrimSpace(stdout.String()), err)
	}
	if out != nil {
		if err := decodeJSONFromCommandOutput(stdout.Bytes(), out); err != nil {
			return fmt.Errorf("decode %s output: %w", args[0], err)
		}
	}
	return nil
}

func decodeJSONFromCommandOutput(data []byte, out interface{}) error {
	trimmed := bytes.TrimSpace(data)
	start := bytes.IndexByte(trimmed, '{')
	end := bytes.LastIndexByte(trimmed, '}')
	if start < 0 || end < start {
		return fmt.Errorf("no JSON object in output %q", string(trimmed))
	}
	return json.Unmarshal(trimmed[start:end+1], out)
}

func writeUnsupportedFundingContextChunk(paths phase10FundingEventChunkPaths, status *Phase10FundingEventChunkStatus) error {
	summary := FundingChunkSummary{
		Symbol:            status.Symbol,
		Year:              monthYear(status.Month),
		Month:             status.Month,
		Status:            "unsupported_context",
		EventFile:         paths.EventFile,
		ZeroEventMonth:    true,
		FamilyEventCounts: map[string]int{},
		SideEventCounts:   map[string]int{},
	}
	diagnostics := FundingDiagnostics{Symbol: status.Symbol, Month: status.Month}
	if err := writeFundingChunkOutputs(paths.SummaryFile, paths.EventFile, paths.DiagnosticsFile, paths.AlphaSummaryFile, summary, nil, diagnostics, nil, true); err != nil {
		return err
	}
	audit := Phase10FundingContextAudit{
		Symbol:                   status.Symbol,
		Month:                    status.Month,
		MarketBetaCounts:         map[string]int{},
		UnsupportedContextReason: "BTCUSDT target has no safe non-self context",
		ContextStatus:            "SELF_CONTEXT_UNSUPPORTED",
	}
	if err := writeFundingJSONReport(paths.ContextAuditFile, audit); err != nil {
		return err
	}
	status.FeatureBuildStatus = "unsupported_context"
	status.RegimeClassifyStatus = "unsupported_context"
	status.FundingJoinStatus = "unsupported_context"
	status.EventEvalStatus = "DONE"
	status.CleanupStatus = "SKIPPED"
	status.SummaryStatus = "unsupported_context"
	status.ContextStatus = "SELF_CONTEXT_UNSUPPORTED"
	status.EventRows = 0
	return nil
}
