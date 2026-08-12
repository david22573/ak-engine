package app

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/david22573/ak-engine/internal/papersignal"
)

type paperCandidateMetadata struct {
	ID                       string
	Version                  string
	Hash                     string
	Side                     papersignal.SignalSide
	ObservationWindowMinutes int
	TargetBPS                float64
	StopBPS                  float64
	SupportedTimeframes      []string
	Source                   string
}

type paperRIFEvidence struct {
	Status        string
	DatasetHash   string
	ManifestHash  string
	UniverseHash  string
	LifecycleHash string
	PITHash       string
	Warnings      []string
	BlocksSignal  bool
	BlockReason   string
}

func parsePaperSymbols(symbolsCSV string) []string {
	var symbols []string
	for _, raw := range strings.Split(symbolsCSV, ",") {
		symbol := strings.ToUpper(strings.TrimSpace(raw))
		if symbol != "" {
			symbols = append(symbols, symbol)
		}
	}
	return symbols
}

func evaluatePaperRIF(datasetManifestPath string, _ bool) paperRIFEvidence {
	blocked := func(reason string, warning string) paperRIFEvidence {
		return paperRIFEvidence{
			Status:       "RIF_BLOCKED",
			Warnings:     []string{warning},
			BlocksSignal: true,
			BlockReason:  reason,
		}
	}
	if datasetManifestPath == "" {
		return blocked("dataset manifest missing", "dataset manifest missing")
	}
	data, err := os.ReadFile(datasetManifestPath)
	if err != nil {
		return blocked("dataset manifest unreadable", fmt.Sprintf("dataset manifest unreadable: %v", err))
	}
	var document map[string]json.RawMessage
	if err := json.Unmarshal(data, &document); err != nil {
		return blocked("dataset manifest malformed", fmt.Sprintf("dataset manifest malformed: %v", err))
	}
	if len(document) == 0 {
		return blocked("dataset manifest malformed", "dataset manifest is empty")
	}
	return blocked(
		"paper-forward manifest readiness path retired",
		"dataset/PIT manifest readiness is diagnostic-only; canonical candidate-bound research evidence is required",
	)
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if value != "" {
			return value
		}
	}
	return ""
}

func emptyAs(value, replacement string) string {
	if value == "" {
		return replacement
	}
	return value
}

func resolvePaperSnapshotPath(rootOrFile, symbol string) (string, bool) {
	info, err := os.Stat(rootOrFile)
	if err != nil {
		return "", false
	}
	if !info.IsDir() {
		return rootOrFile, true
	}
	candidates := []string{
		filepath.Join(rootOrFile, symbol+".json"),
		filepath.Join(rootOrFile, strings.ToLower(symbol)+".json"),
		filepath.Join(rootOrFile, symbol+"_snapshot.json"),
		filepath.Join(rootOrFile, strings.ToLower(symbol)+"_snapshot.json"),
		filepath.Join(rootOrFile, "snapshot_"+symbol+".json"),
		filepath.Join(rootOrFile, "snapshot_"+strings.ToLower(symbol)+".json"),
	}
	for _, candidate := range candidates {
		if _, err := os.Stat(candidate); err == nil {
			return candidate, true
		}
	}
	return "", false
}

func paperTargetAndStop(side papersignal.SignalSide, entryPrice, targetBPS, stopBPS float64) (float64, float64) {
	if side == papersignal.SideShort {
		return entryPrice * (1 - targetBPS/10000), entryPrice * (1 + stopBPS/10000)
	}
	return entryPrice * (1 + targetBPS/10000), entryPrice * (1 - stopBPS/10000)
}
