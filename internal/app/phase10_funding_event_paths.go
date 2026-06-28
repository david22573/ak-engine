package app

import (
	"path/filepath"
	"strings"
)

func normalizePhase10FundingEventPipelineConfig(cfg phase10FundingEventPipelineConfig) phase10FundingEventPipelineConfig {
	if cfg.RootDir == "" {
		cfg.RootDir = "."
	}
	if cfg.Workdir == "" {
		cfg.Workdir = resolveHistorianWorkdir(nil, "")
	}
	if cfg.Chunk == "" {
		cfg.Chunk = "monthly"
	}
	if cfg.ContextSymbols == "" {
		cfg.ContextSymbols = "BTCUSDT,ETHUSDT"
	}
	if cfg.RetainPolicy == "" {
		cfg.RetainPolicy = "reports_only"
	}
	if cfg.ManifestPath == "" {
		cfg.ManifestPath = rootJoin(cfg.RootDir, phase10FundingEventManifestPath)
	}
	if cfg.ReportsDir == "" {
		cfg.ReportsDir = rootJoin(cfg.RootDir, "runs", "reports")
	}
	if cfg.ChunksDir == "" {
		cfg.ChunksDir = rootJoin(cfg.RootDir, "runs", "reports", "chunks")
	}
	if cfg.EventFormat == "" {
		cfg.EventFormat = "jsonl.gz"
	}
	if cfg.Out == "" {
		cfg.Out = rootJoin(cfg.RootDir, "runs", "reports", "phase10_7e_funding_event_pipeline.md")
	} else if !filepath.IsAbs(cfg.Out) {
		cfg.Out = rootJoin(cfg.RootDir, cfg.Out)
	}
	cfg.Symbols = normalizedLimitedSymbols(cfg.Symbols, cfg.MaxSymbols)
	if cfg.MaxMonths > 0 && len(cfg.Months) > cfg.MaxMonths {
		cfg.Months = append([]string(nil), cfg.Months[:cfg.MaxMonths]...)
	}
	return cfg
}

func normalizedLimitedSymbols(symbols []string, max int) []string {
	var out []string
	seen := make(map[string]bool)
	for _, symbol := range symbols {
		symbol = strings.ToUpper(strings.TrimSpace(symbol))
		if symbol == "" || seen[symbol] {
			continue
		}
		seen[symbol] = true
		out = append(out, symbol)
		if max > 0 && len(out) >= max {
			break
		}
	}
	return out
}

func phase10FundingEventPaths(cfg phase10FundingEventPipelineConfig, symbol, month string) phase10FundingEventChunkPaths {
	ext := "jsonl"
	if cfg.EventFormat == "jsonl.gz" {
		ext = "jsonl.gz"
	}
	return phase10FundingEventChunkPaths{
		FeatureContextFile: rootJoin(cfg.RootDir, "runs", "features", "chunks", symbol, month+"-context.json"),
		RegimeContextFile:  rootJoin(cfg.RootDir, "runs", "regimes", "chunks", symbol, month+"-context.json"),
		FundingFeatureFile: rootJoin(cfg.RootDir, "runs", "features", "chunks", symbol, month+"-funding.json"),
		EventFile:          filepath.Join(cfg.ChunksDir, symbol, month+"-funding-events."+ext),
		SummaryFile:        filepath.Join(cfg.ChunksDir, symbol, month+"-funding-summary.json"),
		AlphaSummaryFile:   filepath.Join(cfg.ChunksDir, symbol, month+"-alpha-summary.json"),
		ContextAuditFile:   filepath.Join(cfg.ChunksDir, symbol, month+"-context-audit.json"),
		DiagnosticsFile:    filepath.Join(cfg.ChunksDir, symbol, month+"-funding-diagnostics.json"),
		ChunksDir:          cfg.ChunksDir,
	}
}

func phase10FundingDerivativePath(workdir, symbol, month string) string {
	return fundingRateDerivativePath(workdir, symbol, month)
}

func rootJoin(root string, parts ...string) string {
	if len(parts) == 0 {
		return root
	}
	if filepath.IsAbs(parts[0]) {
		return filepath.Join(parts...)
	}
	all := append([]string{root}, parts...)
	return filepath.Join(all...)
}
