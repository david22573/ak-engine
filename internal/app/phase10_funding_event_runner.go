package app

import (
	"fmt"
	"os"
	"runtime"
	"strings"
	"time"
)

func runPhase10FundingEventPipeline(cfg phase10FundingEventPipelineConfig, steps phase10FundingEventPipelineSteps) (phase10FundingEventPipelineReport, error) {
	cfg = normalizePhase10FundingEventPipelineConfig(cfg)
	if cfg.Chunk != "" && cfg.Chunk != "monthly" {
		return phase10FundingEventPipelineReport{}, fmt.Errorf("unsupported chunk %q", cfg.Chunk)
	}
	if len(cfg.Symbols) == 0 {
		return phase10FundingEventPipelineReport{}, fmt.Errorf("no symbols")
	}
	if len(cfg.Months) == 0 {
		return phase10FundingEventPipelineReport{}, fmt.Errorf("no months")
	}

	manifest := loadPhase10FundingEventManifest(cfg.ManifestPath)
	var chunkErrs []string
	processedKeys := make(map[string]struct{})
	inputChunksRebuilt := 0

	for _, symbol := range cfg.Symbols {
		for _, month := range cfg.Months {
			key := phase10FundingEventChunkKey(symbol, month)
			processedKeys[key] = struct{}{}
			if manifest.Chunks[key] == nil {
				manifest.Chunks[key] = &Phase10FundingEventChunkStatus{Symbol: symbol, Month: month}
			}
			status := manifest.Chunks[key]
			status.Symbol = symbol
			status.Month = month
			status.ContextSymbols = effectiveFundingContextSymbols(symbol, cfg.ContextSymbols)
			if cfg.Force {
				status.FeatureBuildStatus = ""
				status.RegimeClassifyStatus = ""
				status.FundingJoinStatus = ""
				status.EventEvalStatus = ""
				status.CleanupStatus = ""
				status.SummaryStatus = ""
			}
			status.DeletedHeavyFiles = nil
			status.BytesFreed = 0
			paths := phase10FundingEventPaths(cfg, symbol, month)
			status.EventFile = paths.EventFile
			status.SummaryFile = paths.SummaryFile
			status.DiskFreeBeforeBytes = getDiskSpace(cfg.RootDir)
			status.DiskStatus = "OK"

			err := processPhase10FundingEventChunk(cfg, steps, paths, status)
			if status.FeatureBuildStatus == "DONE" || status.RegimeClassifyStatus == "DONE" || strings.HasPrefix(status.FundingJoinStatus, "DONE") {
				inputChunksRebuilt++
			}
			status.DiskFreeAfterBytes = getDiskSpace(cfg.RootDir)
			if err != nil {
				status.ErrorIfFailed = err.Error()
				chunkErrs = append(chunkErrs, fmt.Sprintf("%s: %v", key, err))
				savePhase10FundingEventManifest(cfg.ManifestPath, manifest)
				if !cfg.ContinueOnError {
					report := phase10FundingEventBuildReport(cfg, manifest, nil, processedKeys, inputChunksRebuilt)
					_ = writePhase10FundingEventPipelineReport(cfg, report)
					return report, fmt.Errorf("chunk %s failed: %w", key, err)
				}
			} else {
				status.ErrorIfFailed = ""
				status.CompletedAt = time.Now()
				savePhase10FundingEventManifest(cfg.ManifestPath, manifest)
			}

			if cfg.GCBetweenChunks {
				runtime.GC()
			}
		}
	}

	leaderboard, _, integrity, err := buildFundingAggregationReports(fundingAggregationConfig{
		Symbols:    cfg.Symbols,
		Months:     cfg.Months,
		ChunksDir:  cfg.ChunksDir,
		ReportsDir: cfg.ReportsDir,
	})
	if err == nil {
		enrichPhase10FundingEventIntegrity(&integrity, manifest, processedKeys, inputChunksRebuilt)
		err = writePhase10FundingEventReports(cfg, leaderboard, integrity)
		if err == nil && cfg.SummaryOnlyAfterAgg {
			for _, symbol := range cfg.Symbols {
				for _, month := range cfg.Months {
					paths := phase10FundingEventPaths(cfg, symbol, month)
					os.Remove(paths.EventFile)
				}
			}
		}
	}
	if err != nil {
		chunkErrs = append(chunkErrs, fmt.Sprintf("aggregate: %v", err))
	}

	report := phase10FundingEventBuildReport(cfg, manifest, &leaderboard, processedKeys, inputChunksRebuilt)
	if len(chunkErrs) > 0 {
		report.Status = "pipeline_blocked"
		report.DetailedStatus = "pipeline_blocked"
	}
	if err := writePhase10FundingEventPipelineReport(cfg, report); err != nil {
		return report, err
	}
	if len(chunkErrs) > 0 {
		return report, fmt.Errorf("%s", strings.Join(chunkErrs, "; "))
	}
	return report, nil
}

func processPhase10FundingEventChunk(cfg phase10FundingEventPipelineConfig, steps phase10FundingEventPipelineSteps, paths phase10FundingEventChunkPaths, status *Phase10FundingEventChunkStatus) error {
	if strings.EqualFold(status.Symbol, "BTCUSDT") && status.ContextSymbols == "" {
		return writeUnsupportedFundingContextChunk(paths, status)
	}
	if !cfg.Force && status.EventEvalStatus == "DONE" {
		return nil
	}
	if cfg.MinFreeGB > 0 && float64(status.DiskFreeBeforeBytes) < cfg.MinFreeGB*1024*1024*1024 {
		status.DiskStatus = "BLOCKED_MIN_FREE"
		return fmt.Errorf("free disk below threshold: %.2f GB", float64(status.DiskFreeBeforeBytes)/(1024*1024*1024))
	}
	if cfg.DiskBudgetGB > 0 {
		runsBytes := getDirSize(rootJoin(cfg.RootDir, "runs"))
		if float64(runsBytes) > cfg.DiskBudgetGB*1024*1024*1024 {
			status.DiskStatus = "BLOCKED_DISK_BUDGET"
			return fmt.Errorf("runs artifact size exceeds budget: %.2f GB", float64(runsBytes)/(1024*1024*1024))
		}
	}

	if err := steps.BuildFeatureChunk(cfg, paths, status); err != nil {
		status.FeatureBuildStatus = "FAILED"
		return fmt.Errorf("build context feature chunk: %w", err)
	}
	if status.FeatureBuildStatus == "" {
		status.FeatureBuildStatus = "DONE"
	}
	if cfg.MaxRows > 0 && status.FeatureRows > cfg.MaxRows {
		status.FeatureBuildStatus = "FAILED_MAX_ROWS"
		return fmt.Errorf("feature rows %d exceed max rows %d", status.FeatureRows, cfg.MaxRows)
	}

	if err := steps.ClassifyRegimeChunk(cfg, paths, status); err != nil {
		status.RegimeClassifyStatus = "FAILED"
		return fmt.Errorf("classify regime chunk: %w", err)
	}
	if status.RegimeClassifyStatus == "" {
		status.RegimeClassifyStatus = "DONE"
	}

	if err := steps.JoinFundingChunk(cfg, paths, status); err != nil {
		status.FundingJoinStatus = "FAILED"
		return fmt.Errorf("join funding chunk: %w", err)
	}
	if status.FundingJoinStatus == "" {
		status.FundingJoinStatus = "DONE"
	}

	summary, err := steps.EvaluateFunding(cfg, paths, status)
	if err != nil {
		status.EventEvalStatus = "FAILED"
		return fmt.Errorf("evaluate funding chunk: %w", err)
	}
	status.EventEvalStatus = "DONE"
	status.SummaryStatus = summary.Status
	status.FeatureRows = summary.FeatureRows
	status.RegimeRows = summary.ContextRows
	status.FundingRows = summary.RowsWithFunding
	status.EventRows = summary.EventCount

	if _, _, err := verifyFundingEventOutputs(paths.SummaryFile, paths.EventFile); err != nil {
		status.EventEvalStatus = "FAILED_VERIFY"
		return fmt.Errorf("verify funding event outputs: %w", err)
	}

	contextAudit, err := writePhase10FundingContextAudit(cfg, paths, status)
	if err != nil {
		status.ContextStatus = "UNKNOWN_FAILURE"
		return fmt.Errorf("write context audit: %w", err)
	}
	status.ContextStatus = contextAudit.ContextStatus

	if err := steps.CleanupChunk(cfg, paths, status); err != nil {
		status.CleanupStatus = "FAILED"
		return fmt.Errorf("cleanup heavy chunks: %w", err)
	}
	if status.CleanupStatus == "" {
		status.CleanupStatus = "DONE"
	}
	return nil
}
