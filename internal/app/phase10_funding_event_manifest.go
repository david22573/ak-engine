package app

import (
	"encoding/json"
	"os"
	"strings"
	"time"
)

func loadPhase10FundingEventManifest(path string) *Phase10FundingEventManifest {
	data, err := os.ReadFile(path)
	if err != nil {
		return &Phase10FundingEventManifest{Chunks: make(map[string]*Phase10FundingEventChunkStatus)}
	}
	var manifest Phase10FundingEventManifest
	if err := json.Unmarshal(data, &manifest); err != nil || manifest.Chunks == nil {
		return &Phase10FundingEventManifest{Chunks: make(map[string]*Phase10FundingEventChunkStatus)}
	}
	return &manifest
}

func savePhase10FundingEventManifest(path string, manifest *Phase10FundingEventManifest) {
	manifest.UpdatedAt = time.Now()
	data, _ := json.MarshalIndent(manifest, "", "  ")
	_ = atomicWriteFile(path, data, 0644)
}

func phase10FundingEventChunkKey(symbol, month string) string {
	return strings.ToUpper(symbol) + "_" + month
}
