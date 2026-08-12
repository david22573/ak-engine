package app

import (
	"reflect"
	"testing"

	"github.com/david22573/ak-engine/internal/features"
)

func TestAsOfJoinRejectsFutureDerivativesData(t *testing.T) {
	rows, err := joinResearchFeatureRows(
		[]features.Row{
			{
				Market:        "futures-um",
				Symbol:        "LINKUSDT",
				Interval:      "1m",
				EventTimeMS:   1000,
				AvailableAtMS: 1000,
				Close:         10,
			},
		},
		[]researchDerivativeRow{
			canonicalDerivativeRow(researchDerivativeRow{
				Source:        "binance",
				Dataset:       "funding_rate",
				Market:        "futures-um",
				Symbol:        "LINKUSDT",
				Interval:      "8h",
				EventTimeMS:   900,
				AvailableAtMS: 2000,
				IngestedAtMS:  2000,
				Value:         0.001,
			}),
		},
	)
	if err != nil {
		t.Fatalf("join features: %v", err)
	}
	if rows[0].Derivatives.FundingRate != nil {
		t.Fatalf("future derivative data was joined: %+v", rows[0].Derivatives.FundingRate)
	}
	if !rows[0].Derivatives.FundingRateUnknown {
		t.Fatalf("missing future-rejected funding rate must be marked unknown")
	}
}

func TestMissingDerivativesAreNullUnknownNotZero(t *testing.T) {
	rows, err := joinResearchFeatureRows(
		[]features.Row{
			{
				Market:        "futures-um",
				Symbol:        "LINKUSDT",
				Interval:      "1m",
				EventTimeMS:   1000,
				AvailableAtMS: 1000,
				Close:         10,
			},
		},
		nil,
	)
	if err != nil {
		t.Fatalf("join features: %v", err)
	}
	d := rows[0].Derivatives
	if d.FundingRate != nil || d.OpenInterestChange != nil || d.TakerBuySellImbalance != nil || d.LongShortRatio != nil {
		t.Fatalf("missing derivatives must be nil/null, got %+v", d)
	}
	if !d.FundingRateUnknown || !d.OpenInterestChangeUnknown || !d.TakerBuySellUnknown || !d.LongShortRatioUnknown {
		t.Fatalf("missing derivatives must have unknown flags: %+v", d)
	}
}

func TestJoinResearchFeaturesDerivesFundingAndPositioning(t *testing.T) {
	rows, err := joinResearchFeatureRows(
		[]features.Row{
			{Market: "futures-um", Symbol: "LINKUSDT", Interval: "1m", EventTimeMS: 3000, AvailableAtMS: 3000, Close: 10},
		},
		[]researchDerivativeRow{
			canonicalDerivativeRow(researchDerivativeRow{Source: "binance", Dataset: "funding_rate", Market: "futures-um", Symbol: "LINKUSDT", Interval: "8h", EventTimeMS: 1000, AvailableAtMS: 1000, Value: 0.0001}),
			canonicalDerivativeRow(researchDerivativeRow{Source: "binance", Dataset: "funding_rate", Market: "futures-um", Symbol: "LINKUSDT", Interval: "8h", EventTimeMS: 2000, AvailableAtMS: 2000, Value: 0.0008}),
			canonicalDerivativeRow(researchDerivativeRow{Source: "binance", Dataset: "long_short_ratio", Market: "futures-um", Symbol: "LINKUSDT", Interval: "5m", EventTimeMS: 2000, AvailableAtMS: 2000, Value: 1.8}),
			canonicalDerivativeRow(researchDerivativeRow{Source: "binance", Dataset: "taker_buy_sell_volume", Market: "futures-um", Symbol: "LINKUSDT", Interval: "5m", EventTimeMS: 2000, AvailableAtMS: 2000, Value: 1.5, Extra1: 60, Extra2: 40}),
		},
	)
	if err != nil {
		t.Fatalf("join features: %v", err)
	}
	d := rows[0].Derivatives
	if d.FundingRate == nil || *d.FundingRate != 0.0008 {
		t.Fatalf("funding rate not joined: %+v", d.FundingRate)
	}
	if d.FundingRateChange == nil || *d.FundingRateChange <= 0 {
		t.Fatalf("funding change not derived: %+v", d.FundingRateChange)
	}
	if d.TakerBuySellImbalance == nil || *d.TakerBuySellImbalance != 0.2 {
		t.Fatalf("taker imbalance not derived: %+v", d.TakerBuySellImbalance)
	}
	if d.PositioningCrowdedLong == nil || !*d.PositioningCrowdedLong {
		t.Fatalf("crowded long not derived: %+v", d.PositioningCrowdedLong)
	}
	if d.PositioningUnwindCandidate == nil || !*d.PositioningUnwindCandidate {
		t.Fatalf("unwind candidate not derived: %+v", d.PositioningUnwindCandidate)
	}
}

func TestDerivativeProvenanceSurvivesJoinByteForByte(t *testing.T) {
	source := canonicalDerivativeRow(researchDerivativeRow{
		Source: "binance", Dataset: "funding_rate", Market: "futures-um", Symbol: "LINKUSDT", Interval: "8h",
		EventTimeMS: 1000, AvailableAtMS: 1500, IngestedAtMS: 1500, Value: 0.001, Extra1: 101,
	})
	rows, err := joinResearchFeatureRows(
		[]features.Row{{Market: "futures-um", Symbol: "LINKUSDT", Interval: "1m", EventTimeMS: 1000, AvailableAtMS: 2000, Close: 10}},
		[]researchDerivativeRow{source},
	)
	if err != nil {
		t.Fatal(err)
	}
	if len(rows[0].Derivatives.Provenance) != 1 {
		t.Fatalf("provenance=%+v", rows[0].Derivatives.Provenance)
	}
	if got := researchDerivativeRow(rows[0].Derivatives.Provenance[0]); !reflect.DeepEqual(got, source) {
		t.Fatalf("provenance changed: got %+v want %+v", got, source)
	}
}

func TestDerivativeJoinRejectsWrongSymbolInsteadOfIgnoringIt(t *testing.T) {
	source := canonicalDerivativeRow(researchDerivativeRow{Source: "binance", Dataset: "funding_rate", Market: "futures-um", Symbol: "ETHUSDT", Interval: "8h", EventTimeMS: 1000, AvailableAtMS: 1500, Value: 0.001})
	_, err := joinResearchFeatureRows(
		[]features.Row{{Market: "futures-um", Symbol: "LINKUSDT", Interval: "1m", EventTimeMS: 1000, AvailableAtMS: 2000, Close: 10}},
		[]researchDerivativeRow{source},
	)
	if err == nil {
		t.Fatal("wrong derivative symbol was silently ignored")
	}
}

func canonicalDerivativeRow(row researchDerivativeRow) researchDerivativeRow {
	if row.IngestedAtMS == 0 {
		row.IngestedAtMS = row.AvailableAtMS
	}
	if row.SourceVersion == "" {
		row.SourceVersion = "test-source-v1"
	}
	row.AvailabilityPolicyID = researchDerivativeAvailabilityPolicyID
	row.AvailabilityPolicyVersion = researchDerivativeAvailabilityPolicyVersion
	return row
}
