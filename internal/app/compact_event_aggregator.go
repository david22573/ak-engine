package app

import (
	"bufio"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math"
	"strings"
)

// EventFilter allows declaratively filtering retained events using only valid pre-entry context fields.
type EventFilter struct {
	IncludeRegime     string
	ExcludeRegime     string
	IncludeVolatility string
	ExcludeVolatility string
	IncludeFunding    string
	ExcludeFunding    string
	IncludeBTCContext string
	ExcludeBTCContext string
	IncludeETHContext string
	ExcludeETHContext string

	// Invalid/Leaky fields for filter simulation
	InvalidOutcomeFilter    string
	InvalidPnLFilter        string
	InvalidBadMonthFilter   string
	InvalidBadQuarterFilter string
}

// Validate ensures no leaky/invalid fields are used in the filter.
func (f *EventFilter) Validate() error {
	if f.InvalidOutcomeFilter != "" || f.InvalidPnLFilter != "" || f.InvalidBadMonthFilter != "" || f.InvalidBadQuarterFilter != "" {
		return errors.New("leaky filter attempt rejected")
	}
	return nil
}

// Matches checks if a CompactRetainedEvent passes the valid pre-entry filters.
func (f *EventFilter) Matches(e CompactRetainedEvent) bool {
	if f.IncludeRegime != "" && e.PreEntry.TrendRegime != f.IncludeRegime {
		return false
	}
	if f.ExcludeRegime != "" && e.PreEntry.TrendRegime == f.ExcludeRegime {
		return false
	}
	if f.IncludeVolatility != "" && e.PreEntry.VolatilityBucket != f.IncludeVolatility {
		return false
	}
	if f.ExcludeVolatility != "" && e.PreEntry.VolatilityBucket == f.ExcludeVolatility {
		return false
	}
	if f.IncludeFunding != "" && e.PreEntry.FundingBucket != f.IncludeFunding {
		return false
	}
	if f.ExcludeFunding != "" && e.PreEntry.FundingBucket == f.ExcludeFunding {
		return false
	}
	if f.IncludeBTCContext != "" && e.PreEntry.BTCContextBucket != f.IncludeBTCContext {
		return false
	}
	if f.ExcludeBTCContext != "" && e.PreEntry.BTCContextBucket == f.ExcludeBTCContext {
		return false
	}
	if f.IncludeETHContext != "" && e.PreEntry.ETHContextBucket != f.IncludeETHContext {
		return false
	}
	if f.ExcludeETHContext != "" && e.PreEntry.ETHContextBucket == f.ExcludeETHContext {
		return false
	}
	return true
}

// AggregationSummary holds the core audit metrics.
type AggregationSummary struct {
	EventCount     int
	ClusterCount   int
	WinCount       int
	LossCount      int
	GrossBps       float64
	Net5Bps        float64
	Net75Bps       float64
	Net10Bps       float64
	ProfitFactor5  float64
	ProfitFactor75 float64
	ProfitFactor10 float64
	Expectancy5    float64
	Expectancy75   float64
	Expectancy10   float64

	AverageClusterSize float64
	MaxClusterSize     int
	AverageSpacingMs   float64

	WorstMonth   string
	BestMonth    string
	WorstQuarter string
	BestQuarter  string

	SymbolConcentration  map[string]int
	MonthConcentration   map[string]int
	QuarterConcentration map[string]int
}

// Aggregator is the harness for parsing and analyzing compact retained events.
type Aggregator struct {
	Events []CompactRetainedEvent
}

// NewAggregator creates a new Aggregator.
func NewAggregator() *Aggregator {
	return &Aggregator{}
}

// LoadJSONL parses a JSONL stream of CompactRetainedEvents.
func (a *Aggregator) LoadJSONL(r io.Reader) error {
	scanner := bufio.NewScanner(r)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}
		var e CompactRetainedEvent
		if err := json.Unmarshal([]byte(line), &e); err != nil {
			return fmt.Errorf("failed to parse event: %w", err)
		}
		if err := e.Validate(); err != nil {
			return fmt.Errorf("invalid event in JSONL: %w", err)
		}
		a.Events = append(a.Events, e)
	}
	return scanner.Err()
}

// summarizeEvents computes metrics for a slice of events.
func summarizeEvents(events []CompactRetainedEvent) AggregationSummary {
	s := AggregationSummary{
		SymbolConcentration:  make(map[string]int),
		MonthConcentration:   make(map[string]int),
		QuarterConcentration: make(map[string]int),
	}

	if len(events) == 0 {
		return s
	}

	s.EventCount = len(events)
	clusters := make(map[string]bool)
	var totalClusterSize, totalSpacingMs int64
	clusterSpacings := 0

	var win5, win75, win10, loss5, loss75, loss10 float64
	var sumNet5, sumNet75, sumNet10 float64

	monthGross := make(map[string]float64)
	quarterGross := make(map[string]float64)

	for _, e := range events {
		s.SymbolConcentration[e.Symbol]++
		s.MonthConcentration[e.Month]++
		s.QuarterConcentration[e.Quarter]++

		if !clusters[e.Cluster.Key] {
			clusters[e.Cluster.Key] = true
			s.ClusterCount++
			totalClusterSize += int64(e.Cluster.Size)
			if e.Cluster.Size > s.MaxClusterSize {
				s.MaxClusterSize = e.Cluster.Size
			}
			if e.Cluster.SpacingMs > 0 {
				totalSpacingMs += e.Cluster.SpacingMs
				clusterSpacings++
			}
		}

		s.GrossBps += e.GrossOutcomeBps
		sumNet5 += e.NetOutcome5Bps
		sumNet75 += e.NetOutcome75Bps
		sumNet10 += e.NetOutcome10Bps

		monthGross[e.Month] += e.NetOutcome5Bps
		quarterGross[e.Quarter] += e.NetOutcome5Bps

		// Calculate wins/losses for Profit Factor and Counts
		// Using 5bps as the standard for basic win/loss counts per requirement, or just global win count based on 5bps.
		if e.Win5Bps {
			win5 += e.NetOutcome5Bps
			s.WinCount++
		} else {
			loss5 += math.Abs(e.NetOutcome5Bps)
			s.LossCount++
		}

		if e.Win75Bps {
			win75 += e.NetOutcome75Bps
		} else {
			loss75 += math.Abs(e.NetOutcome75Bps)
		}
		if e.Win10Bps {
			win10 += e.NetOutcome10Bps
		} else {
			loss10 += math.Abs(e.NetOutcome10Bps)
		}
	}

	s.Net5Bps = sumNet5
	s.Net75Bps = sumNet75
	s.Net10Bps = sumNet10

	if loss5 > 0 {
		s.ProfitFactor5 = win5 / loss5
	} else if win5 > 0 {
		s.ProfitFactor5 = 999.0
	}
	if loss75 > 0 {
		s.ProfitFactor75 = win75 / loss75
	} else if win75 > 0 {
		s.ProfitFactor75 = 999.0
	}
	if loss10 > 0 {
		s.ProfitFactor10 = win10 / loss10
	} else if win10 > 0 {
		s.ProfitFactor10 = 999.0
	}

	s.Expectancy5 = sumNet5 / float64(s.EventCount)
	s.Expectancy75 = sumNet75 / float64(s.EventCount)
	s.Expectancy10 = sumNet10 / float64(s.EventCount)

	if s.ClusterCount > 0 {
		s.AverageClusterSize = float64(totalClusterSize) / float64(s.ClusterCount)
	}
	if clusterSpacings > 0 {
		s.AverageSpacingMs = float64(totalSpacingMs) / float64(clusterSpacings)
	}

	var bestMonth, worstMonth, bestQuarter, worstQuarter string
	var bestMonthVal, worstMonthVal float64 = -999999, 999999
	var bestQuarterVal, worstQuarterVal float64 = -999999, 999999

	for m, val := range monthGross {
		if val > bestMonthVal {
			bestMonthVal = val
			bestMonth = m
		}
		if val < worstMonthVal {
			worstMonthVal = val
			worstMonth = m
		}
	}
	for q, val := range quarterGross {
		if val > bestQuarterVal {
			bestQuarterVal = val
			bestQuarter = q
		}
		if val < worstQuarterVal {
			worstQuarterVal = val
			worstQuarter = q
		}
	}
	s.BestMonth = bestMonth
	s.WorstMonth = worstMonth
	s.BestQuarter = bestQuarter
	s.WorstQuarter = worstQuarter

	return s
}

// FullSummary provides the overall summary.
func (a *Aggregator) FullSummary() AggregationSummary {
	return summarizeEvents(a.Events)
}

// SimulateFilter evaluates a declarative pre-entry filter and returns the resulting summary.
func (a *Aggregator) SimulateFilter(f EventFilter) (AggregationSummary, error) {
	if err := f.Validate(); err != nil {
		return AggregationSummary{}, err
	}
	var filtered []CompactRetainedEvent
	for _, e := range a.Events {
		if f.Matches(e) {
			filtered = append(filtered, e)
		}
	}
	return summarizeEvents(filtered), nil
}

// SymbolMonthSummary groups by Symbol + Month
func (a *Aggregator) SymbolMonthSummary() map[string]AggregationSummary {
	groups := make(map[string][]CompactRetainedEvent)
	for _, e := range a.Events {
		k := e.Symbol + "_" + e.Month
		groups[k] = append(groups[k], e)
	}
	res := make(map[string]AggregationSummary)
	for k, evs := range groups {
		res[k] = summarizeEvents(evs)
	}
	return res
}

// SymbolQuarterSummary groups by Symbol + Quarter
func (a *Aggregator) SymbolQuarterSummary() map[string]AggregationSummary {
	groups := make(map[string][]CompactRetainedEvent)
	for _, e := range a.Events {
		k := e.Symbol + "_" + e.Quarter
		groups[k] = append(groups[k], e)
	}
	res := make(map[string]AggregationSummary)
	for k, evs := range groups {
		res[k] = summarizeEvents(evs)
	}
	return res
}

// CandidateQuarterSummary groups by CandidateID + Quarter
func (a *Aggregator) CandidateQuarterSummary() map[string]AggregationSummary {
	groups := make(map[string][]CompactRetainedEvent)
	for _, e := range a.Events {
		k := e.CandidateID + "_" + e.Quarter
		groups[k] = append(groups[k], e)
	}
	res := make(map[string]AggregationSummary)
	for k, evs := range groups {
		res[k] = summarizeEvents(evs)
	}
	return res
}

// FamilySideHorizonSummary groups by Symbol, Side, Horizon
func (a *Aggregator) FamilySideHorizonSummary() map[string]AggregationSummary {
	groups := make(map[string][]CompactRetainedEvent)
	for _, e := range a.Events {
		sideStr := "long"
		if e.Side == -1 {
			sideStr = "short"
		}
		k := fmt.Sprintf("%s_%s_%d", e.Symbol, sideStr, e.Horizon)
		groups[k] = append(groups[k], e)
	}
	res := make(map[string]AggregationSummary)
	for k, evs := range groups {
		res[k] = summarizeEvents(evs)
	}
	return res
}

// LeaveOneMonthOutSummary simulates excluding one month at a time.
func (a *Aggregator) LeaveOneMonthOutSummary() map[string]AggregationSummary {
	months := make(map[string]bool)
	for _, e := range a.Events {
		months[e.Month] = true
	}
	res := make(map[string]AggregationSummary)
	for m := range months {
		var filtered []CompactRetainedEvent
		for _, e := range a.Events {
			if e.Month != m {
				filtered = append(filtered, e)
			}
		}
		res[m] = summarizeEvents(filtered)
	}
	return res
}

// LeaveOneQuarterOutSummary simulates excluding one quarter at a time.
func (a *Aggregator) LeaveOneQuarterOutSummary() map[string]AggregationSummary {
	quarters := make(map[string]bool)
	for _, e := range a.Events {
		quarters[e.Quarter] = true
	}
	res := make(map[string]AggregationSummary)
	for q := range quarters {
		var filtered []CompactRetainedEvent
		for _, e := range a.Events {
			if e.Quarter != q {
				filtered = append(filtered, e)
			}
		}
		res[q] = summarizeEvents(filtered)
	}
	return res
}
