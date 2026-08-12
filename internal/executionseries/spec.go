// Package executionseries owns the canonical research execution-series rules.
package executionseries

import (
	"fmt"
	"sort"

	"github.com/david22573/ak-engine/internal/temporal"
	"github.com/david22573/ak-engine/pkg/protocol"
)

const (
	ID                = "ak.engine.deep_execution_series"
	Version           = "2"
	GenerationVersion = ID + ".v" + Version
)

// Window is the authoritative observation range for a candidate outcome.
// Entry is the first candle open strictly after the decision, plus an optional
// diagnostic delay. The horizon is anchored at fill, and ranges begin with the
// fill candle; the decision candle is never included.
type Window struct {
	EventTimeMS        int64
	DecisionTimeMS     int64
	NextTradableTimeMS int64
	FillTimeMS         int64
	EntryIndex         int
	EndIndex           int
	EntryPrice         float64
	ExitPrice          float64
}

func Resolve(
	eventTimeMS, availableAtMS, decisionTimeMS int64,
	candles []protocol.Candle,
	additionalDelayCandles, horizonMinutes int,
	periodEndMS int64,
) (Window, error) {
	if len(candles) == 0 {
		return Window{}, fmt.Errorf("%s: candles are empty", GenerationVersion)
	}
	if additionalDelayCandles < 0 || horizonMinutes <= 0 {
		return Window{}, fmt.Errorf("%s: invalid delay or horizon", GenerationVersion)
	}
	if err := (temporal.Observation{
		SourceEventMS: eventTimeMS, SourceAvailableMS: availableAtMS, DecisionMS: decisionTimeMS,
	}).Validate(); err != nil {
		return Window{}, err
	}
	firstTradable := sort.Search(len(candles), func(i int) bool {
		return candles[i].OpenTimeMS > decisionTimeMS
	})
	if firstTradable >= len(candles) {
		return Window{}, fmt.Errorf("%s: no tradable candle after decision", GenerationVersion)
	}
	entryIndex := firstTradable + additionalDelayCandles
	if entryIndex >= len(candles) {
		return Window{}, fmt.Errorf("%s: delayed fill is outside candle series", GenerationVersion)
	}
	entry := candles[entryIndex]
	if entry.Open <= 0 {
		return Window{}, fmt.Errorf("%s: entry open must be positive", GenerationVersion)
	}
	if err := (temporal.Observation{
		SourceEventMS: eventTimeMS, SourceAvailableMS: availableAtMS, DecisionMS: decisionTimeMS,
		NextTradableMS: candles[firstTradable].OpenTimeMS, FillMS: entry.OpenTimeMS,
	}).Validate(); err != nil {
		return Window{}, err
	}
	horizonEndMS := entry.OpenTimeMS + int64(horizonMinutes)*60_000 - 1
	if periodEndMS > 0 && horizonEndMS > periodEndMS {
		return Window{}, fmt.Errorf("%s: horizon exceeds period boundary", GenerationVersion)
	}
	endAfter := sort.Search(len(candles), func(i int) bool {
		return candles[i].OpenTimeMS > horizonEndMS
	})
	endIndex := endAfter - 1
	if endIndex < entryIndex {
		return Window{}, fmt.Errorf("%s: complete horizon is unavailable", GenerationVersion)
	}
	if endAfter == len(candles) && candles[endIndex].OpenTimeMS < horizonEndMS && candles[endIndex].CloseTimeMS < horizonEndMS {
		return Window{}, fmt.Errorf("%s: complete horizon is unavailable", GenerationVersion)
	}
	exit := candles[endIndex]
	if exit.Close <= 0 {
		return Window{}, fmt.Errorf("%s: exit close must be positive", GenerationVersion)
	}
	return Window{
		EventTimeMS: eventTimeMS, DecisionTimeMS: decisionTimeMS,
		NextTradableTimeMS: candles[firstTradable].OpenTimeMS, FillTimeMS: entry.OpenTimeMS,
		EntryIndex: entryIndex, EndIndex: endIndex, EntryPrice: entry.Open, ExitPrice: exit.Close,
	}, nil
}
