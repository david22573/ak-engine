package app

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"math"
	"reflect"
	"sort"
	"strings"
)

const fundingSufficientStatisticsContract = "ak.engine.funding_sufficient_statistics.v1"

type FundingSufficientEventV1 struct {
	Symbol                 string   `json:"symbol"`
	Family                 string   `json:"family"`
	Side                   string   `json:"side"`
	EventTimeMS            int64    `json:"event_time_ms"`
	AvailableAtMS          int64    `json:"available_at_ms"`
	FundingBucket          string   `json:"funding_bucket"`
	RegimeComposite        string   `json:"regime_composite"`
	Volatility             string   `json:"volatility"`
	Trend                  string   `json:"trend"`
	Liquidity              string   `json:"liquidity"`
	MarketBeta             string   `json:"market_beta"`
	Return5mBps            float64  `json:"return_5m_bps"`
	Return15mBps           float64  `json:"return_15m_bps"`
	Return30mBps           float64  `json:"return_30m_bps"`
	Return60mBps           float64  `json:"return_60m_bps"`
	Return120mBps          float64  `json:"return_120m_bps"`
	Return240mBps          float64  `json:"return_240m_bps"`
	EntryDelay1c60m5bpsBps *float64 `json:"entry_delay_1c_return_60m_5bps_bps,omitempty"`
	LeakageStatus          string   `json:"leakage_status"`
}

type FundingSufficientStatisticsV1 struct {
	Contract       string                     `json:"contract"`
	PartitionMonth string                     `json:"partition_month"`
	Symbol         string                     `json:"symbol"`
	Family         string                     `json:"family"`
	Side           string                     `json:"side"`
	Events         []FundingSufficientEventV1 `json:"events"`
	SummaryHash    string                     `json:"summary_hash"`
}

func buildFundingSufficientStatistics(events []FundingEventRow, month, symbol, family, side string) (FundingSufficientStatisticsV1, error) {
	stats := FundingSufficientStatisticsV1{
		Contract:       fundingSufficientStatisticsContract,
		PartitionMonth: month,
		Symbol:         strings.ToUpper(symbol),
		Family:         family,
		Side:           strings.ToLower(side),
		Events:         make([]FundingSufficientEventV1, 0, len(events)),
	}
	for _, event := range events {
		if strings.ToUpper(event.Symbol) != stats.Symbol || event.Family != stats.Family || strings.ToLower(event.Side) != stats.Side || monthFromEventTime(event.EventTimeMS) != month {
			return FundingSufficientStatisticsV1{}, fmt.Errorf("event does not match sufficient-statistics partition")
		}
		stats.Events = append(stats.Events, sufficientEventFromFundingEvent(event))
	}
	sort.Slice(stats.Events, func(i, j int) bool { return sufficientEventKey(stats.Events[i]) < sufficientEventKey(stats.Events[j]) })
	if err := validateSufficientEventOrder(stats.Events); err != nil {
		return FundingSufficientStatisticsV1{}, err
	}
	var err error
	stats.SummaryHash, err = hashFundingSufficientStatistics(stats)
	if err != nil {
		return FundingSufficientStatisticsV1{}, err
	}
	return stats, nil
}

func validateFundingSufficientStatistics(stats FundingSufficientStatisticsV1) error {
	if stats.Contract != fundingSufficientStatisticsContract || stats.PartitionMonth == "" || stats.Symbol == "" || stats.Family == "" || stats.Side == "" || len(stats.Events) == 0 || stats.SummaryHash == "" {
		return fmt.Errorf("incomplete funding sufficient-statistics contract")
	}
	if stats.Symbol != strings.ToUpper(stats.Symbol) || stats.Side != strings.ToLower(stats.Side) {
		return fmt.Errorf("non-canonical funding sufficient-statistics identity")
	}
	if err := validateSufficientEventOrder(stats.Events); err != nil {
		return err
	}
	for _, event := range stats.Events {
		if event.Symbol != stats.Symbol || event.Family != stats.Family || strings.ToLower(event.Side) != stats.Side || monthFromEventTime(event.EventTimeMS) != stats.PartitionMonth {
			return fmt.Errorf("sufficient event does not match partition identity")
		}
		if event.EventTimeMS <= 0 || event.AvailableAtMS < event.EventTimeMS {
			return fmt.Errorf("invalid sufficient event time")
		}
		if event.FundingBucket == "" || event.RegimeComposite == "" || event.Volatility == "" || event.Trend == "" || event.Liquidity == "" || event.MarketBeta == "" || event.LeakageStatus == "" {
			return fmt.Errorf("incomplete sufficient event dimensions")
		}
		values := []float64{
			event.Return5mBps, event.Return15mBps, event.Return30mBps,
			event.Return60mBps, event.Return120mBps, event.Return240mBps,
		}
		if event.EntryDelay1c60m5bpsBps != nil {
			values = append(values, *event.EntryDelay1c60m5bpsBps)
		}
		for _, value := range values {
			if math.IsNaN(value) || math.IsInf(value, 0) {
				return fmt.Errorf("non-finite sufficient event return")
			}
		}
	}
	want, err := hashFundingSufficientStatistics(stats)
	if err != nil {
		return err
	}
	if stats.SummaryHash != want {
		return fmt.Errorf("funding sufficient-statistics hash mismatch")
	}
	return nil
}

func validateFundingAlphaSummaryRow(row FundingAlphaSummaryRow) error {
	if row.SummarySchemaVersion != fundingAlphaSummarySchemaVersion || row.ClusterKeyVersion != "1.0-native" {
		return fmt.Errorf("unsupported funding alpha summary contract")
	}
	if err := validateFundingSufficientStatistics(row.SufficientStatistics); err != nil {
		return err
	}
	stats := row.SufficientStatistics
	if row.Symbol != stats.Symbol || row.Month != stats.PartitionMonth || row.Family != stats.Family || strings.ToLower(row.Side) != stats.Side {
		return fmt.Errorf("alpha summary identity does not match sufficient statistics")
	}
	if row.Year != monthYear(row.Month) || row.Quarter != quarterFromMonth(row.Month) || !isCanonicalFundingHorizon(row.Horizon) {
		return fmt.Errorf("invalid alpha summary calendar or horizon identity")
	}
	want := computeFundingMetrics(fundingEventsFromSufficient(stats.Events), row.Horizon)
	if !reflect.DeepEqual(want, row.Stats) {
		return fmt.Errorf("alpha summary metrics do not match sufficient statistics")
	}
	return nil
}

func aggregateFundingSufficientStatistics(rows []FundingAlphaSummaryRow, horizon string) (FundingMetrics, error) {
	var events []FundingSufficientEventV1
	seen := make(map[string]struct{})
	for _, row := range rows {
		if row.Horizon != horizon {
			return FundingMetrics{}, fmt.Errorf("summary horizon %q does not match reducer horizon %q", row.Horizon, horizon)
		}
		if err := validateFundingAlphaSummaryRow(row); err != nil {
			return FundingMetrics{}, err
		}
		for _, event := range row.SufficientStatistics.Events {
			key := sufficientEventKey(event)
			if _, exists := seen[key]; exists {
				return FundingMetrics{}, fmt.Errorf("duplicate sufficient event %s", key)
			}
			seen[key] = struct{}{}
			events = append(events, event)
		}
	}
	sort.Slice(events, func(i, j int) bool { return sufficientEventKey(events[i]) < sufficientEventKey(events[j]) })
	return computeFundingMetrics(fundingEventsFromSufficient(events), horizon), nil
}

func canonicalFundingAggregationHash(rows []FundingAlphaSummaryRow) (string, error) {
	unique := make(map[string]FundingSufficientEventV1)
	for _, row := range rows {
		if err := validateFundingAlphaSummaryRow(row); err != nil {
			return "", err
		}
		for _, event := range row.SufficientStatistics.Events {
			key := sufficientEventKey(event)
			if previous, ok := unique[key]; ok && !reflect.DeepEqual(previous, event) {
				return "", fmt.Errorf("conflicting sufficient event %s", key)
			}
			unique[key] = event
		}
	}
	keys := make([]string, 0, len(unique))
	for key := range unique {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	events := make([]FundingSufficientEventV1, 0, len(keys))
	for _, key := range keys {
		events = append(events, unique[key])
	}
	payload := struct {
		Contract string                     `json:"contract"`
		Events   []FundingSufficientEventV1 `json:"events"`
	}{fundingSufficientStatisticsContract, events}
	data, err := json.Marshal(payload)
	if err != nil {
		return "", err
	}
	hash := sha256.Sum256(data)
	return "sha256:" + hex.EncodeToString(hash[:]), nil
}

func hashFundingSufficientStatistics(stats FundingSufficientStatisticsV1) (string, error) {
	stats.SummaryHash = ""
	data, err := json.Marshal(stats)
	if err != nil {
		return "", err
	}
	hash := sha256.Sum256(data)
	return "sha256:" + hex.EncodeToString(hash[:]), nil
}

func validateSufficientEventOrder(events []FundingSufficientEventV1) error {
	for i := range events {
		if i > 0 && sufficientEventKey(events[i]) <= sufficientEventKey(events[i-1]) {
			return fmt.Errorf("sufficient events are duplicate or out of order")
		}
	}
	return nil
}

func sufficientEventKey(event FundingSufficientEventV1) string {
	return fmt.Sprintf("%s|%s|%s|%020d", event.Symbol, event.Family, strings.ToLower(event.Side), event.EventTimeMS)
}

func sufficientEventFromFundingEvent(event FundingEventRow) FundingSufficientEventV1 {
	return FundingSufficientEventV1{
		Symbol: strings.ToUpper(event.Symbol), Family: event.Family, Side: strings.ToLower(event.Side),
		EventTimeMS: event.EventTimeMS, AvailableAtMS: event.AvailableAtMS,
		FundingBucket: event.FundingBucket, RegimeComposite: event.RegimeComposite, Volatility: event.Volatility,
		Trend: event.Trend, Liquidity: event.Liquidity, MarketBeta: event.MarketBeta,
		Return5mBps: event.Return5mBps, Return15mBps: event.Return15mBps, Return30mBps: event.Return30mBps,
		Return60mBps: event.Return60mBps, Return120mBps: event.Return120mBps, Return240mBps: event.Return240mBps,
		EntryDelay1c60m5bpsBps: event.EntryDelay1c60mBps, LeakageStatus: event.LeakageStatus,
	}
}

func fundingEventsFromSufficient(events []FundingSufficientEventV1) []FundingEventRow {
	out := make([]FundingEventRow, 0, len(events))
	for _, event := range events {
		out = append(out, FundingEventRow{
			Symbol: event.Symbol, Family: event.Family, Side: event.Side, EventTimeMS: event.EventTimeMS, AvailableAtMS: event.AvailableAtMS,
			FundingBucket: event.FundingBucket, RegimeComposite: event.RegimeComposite, Volatility: event.Volatility,
			Trend: event.Trend, Liquidity: event.Liquidity, MarketBeta: event.MarketBeta,
			Return5mBps: event.Return5mBps, Return15mBps: event.Return15mBps, Return30mBps: event.Return30mBps,
			Return60mBps: event.Return60mBps, Return120mBps: event.Return120mBps, Return240mBps: event.Return240mBps,
			EntryDelay1c60mBps: event.EntryDelay1c60m5bpsBps, LeakageStatus: event.LeakageStatus,
		})
	}
	return out
}

func fundingAlphaRowsFromEvents(events []FundingEventRow, month string) ([]FundingAlphaSummaryRow, error) {
	byCandidate := make(map[string][]FundingEventRow)
	for _, event := range events {
		key := candidateHorizonKey(event.Symbol, event.Family, event.Side, "")
		byCandidate[key] = append(byCandidate[key], event)
	}
	var rows []FundingAlphaSummaryRow
	for key, group := range byCandidate {
		parts := strings.Split(key, "|")
		if len(parts) != 4 {
			return nil, fmt.Errorf("invalid funding candidate key %q", key)
		}
		sufficient, err := buildFundingSufficientStatistics(group, month, parts[0], parts[1], parts[2])
		if err != nil {
			return nil, err
		}
		for _, horizon := range defaultFundingHorizons {
			rows = append(rows, FundingAlphaSummaryRow{
				Symbol: parts[0], Year: monthYear(month), Quarter: quarterFromMonth(month), Month: month,
				Family: parts[1], Side: parts[2], Horizon: horizon,
				SummarySchemaVersion: fundingAlphaSummarySchemaVersion, ClusterKeyVersion: "1.0-native",
				Stats: computeFundingMetrics(group, horizon), SufficientStatistics: sufficient,
			})
		}
	}
	sort.Slice(rows, func(i, j int) bool {
		return strings.Join([]string{rows[i].Symbol, rows[i].Family, rows[i].Side, rows[i].Horizon, rows[i].Month}, "|") <
			strings.Join([]string{rows[j].Symbol, rows[j].Family, rows[j].Side, rows[j].Horizon, rows[j].Month}, "|")
	})
	return rows, nil
}

func canonicalFundingAlphaRows(loaded []fundingLoadedEventFile) ([]FundingAlphaSummaryRow, error) {
	var out []FundingAlphaSummaryRow
	for _, item := range loaded {
		if item.ParseError != "" {
			return nil, fmt.Errorf("%s|%s parse error: %s", item.Symbol, item.Month, item.ParseError)
		}
		if len(item.Events) > 0 {
			for _, event := range item.Events {
				if strings.ToUpper(event.Symbol) != strings.ToUpper(item.Symbol) || monthFromEventTime(event.EventTimeMS) != item.Month {
					return nil, fmt.Errorf("%s|%s raw event violates file partition identity", item.Symbol, item.Month)
				}
			}
			derived, err := fundingAlphaRowsFromEvents(item.Events, item.Month)
			if err != nil {
				return nil, err
			}
			if len(item.AlphaSummary) > 0 {
				if err := compareFundingAlphaRows(derived, item.AlphaSummary); err != nil {
					return nil, fmt.Errorf("%s|%s raw/summary disagreement: %w", item.Symbol, item.Month, err)
				}
			}
			out = append(out, derived...)
			continue
		}
		for _, row := range item.AlphaSummary {
			if row.Symbol != strings.ToUpper(item.Symbol) || row.Month != item.Month {
				return nil, fmt.Errorf("%s|%s summary violates file partition identity", item.Symbol, item.Month)
			}
			if err := validateFundingAlphaSummaryRow(row); err != nil {
				return nil, fmt.Errorf("%s|%s invalid summary: %w", item.Symbol, item.Month, err)
			}
		}
		out = append(out, item.AlphaSummary...)
	}
	return out, nil
}

func isCanonicalFundingHorizon(horizon string) bool {
	for _, candidate := range defaultFundingHorizons {
		if horizon == candidate {
			return true
		}
	}
	return false
}

func compareFundingAlphaRows(expected, actual []FundingAlphaSummaryRow) error {
	if len(expected) != len(actual) {
		return fmt.Errorf("row count got %d want %d", len(actual), len(expected))
	}
	keyed := make(map[string]FundingAlphaSummaryRow, len(expected))
	for _, row := range expected {
		keyed[fundingAlphaRowKey(row)] = row
	}
	for _, row := range actual {
		if err := validateFundingAlphaSummaryRow(row); err != nil {
			return err
		}
		want, ok := keyed[fundingAlphaRowKey(row)]
		if !ok || !reflect.DeepEqual(want, row) {
			return fmt.Errorf("row %s differs", fundingAlphaRowKey(row))
		}
		delete(keyed, fundingAlphaRowKey(row))
	}
	if len(keyed) != 0 {
		return fmt.Errorf("missing derived rows")
	}
	return nil
}

func canonicalFundingEvents(rows []FundingAlphaSummaryRow) ([]FundingEventRow, error) {
	unique := make(map[string]FundingSufficientEventV1)
	for _, row := range rows {
		if err := validateFundingAlphaSummaryRow(row); err != nil {
			return nil, err
		}
		for _, event := range row.SufficientStatistics.Events {
			key := sufficientEventKey(event)
			if previous, ok := unique[key]; ok && !reflect.DeepEqual(previous, event) {
				return nil, fmt.Errorf("conflicting sufficient event %s", key)
			}
			unique[key] = event
		}
	}
	keys := make([]string, 0, len(unique))
	for key := range unique {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	projected := make([]FundingSufficientEventV1, 0, len(keys))
	for _, key := range keys {
		projected = append(projected, unique[key])
	}
	return fundingEventsFromSufficient(projected), nil
}

func fundingAlphaRowKey(row FundingAlphaSummaryRow) string {
	return strings.Join([]string{row.Symbol, row.Family, strings.ToLower(row.Side), row.Horizon, row.Month}, "|")
}
