package data

import (
	"encoding/json"
	"fmt"
	"time"
)

type Manifest struct {
	SchemaVersion      int           `json:"schema_version"`
	Market             string        `json:"market"`
	Symbol             string        `json:"symbol"`
	Interval           string        `json:"interval"`
	CoverageStart      time.Time     `json:"coverage_start"`
	CoverageEnd        time.Time     `json:"coverage_end"`
	ExpectedCandles    int64         `json:"expected_candles"`
	ActualCandles      int64         `json:"actual_candles"`
	UniqueOpenTimes    int64         `json:"unique_open_times"`
	DuplicateOpenTimes int64         `json:"duplicate_open_times"`
	MissingCandles     int64         `json:"missing_candles"`
	ObjectCount        int           `json:"object_count"`
	Objects            []ObjectStats `json:"objects"`
	LastVerifiedAt     time.Time     `json:"last_verified_at"`
	Status             string        `json:"status"`
}

type ObjectStats struct {
	Key           string `json:"key"`
	Period        string `json:"period"`
	SourceDate    string `json:"source_date"`
	RowCount      int64  `json:"row_count"`
	MinOpenTimeMS int64  `json:"min_open_time_ms"`
	MaxOpenTimeMS int64  `json:"max_open_time_ms"`
}

func ParseManifest(data []byte) (*Manifest, error) {
	var m Manifest
	if err := json.Unmarshal(data, &m); err != nil {
		return nil, fmt.Errorf("failed to unmarshal manifest JSON: %w", err)
	}

	if m.Status != "PASS" {
		return nil, fmt.Errorf("manifest status is not PASS: %s", m.Status)
	}
	if len(m.Objects) == 0 {
		return nil, fmt.Errorf("manifest has no objects")
	}

	for i, obj := range m.Objects {
		if obj.Key == "" {
			return nil, fmt.Errorf("missing object key at index %d", i)
		}
	}

	return &m, nil
}

func (m *Manifest) ValidateForRequest(req CandleRequest) error {
	if m.Market != "" && m.Market != req.Market {
		return fmt.Errorf("manifest market %q does not match request market %q", m.Market, req.Market)
	}
	if m.Symbol != "" && m.Symbol != req.Symbol {
		return fmt.Errorf("manifest symbol %q does not match request symbol %q", m.Symbol, req.Symbol)
	}
	if m.Interval != "" && m.Interval != req.Interval {
		return fmt.Errorf("manifest interval %q does not match request interval %q", m.Interval, req.Interval)
	}
	return nil
}
