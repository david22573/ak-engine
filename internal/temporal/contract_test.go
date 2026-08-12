package temporal

import "testing"

func TestObservationCanonicalOrdering(t *testing.T) {
	valid := Observation{
		SourceEventMS:     1_000,
		SourceAvailableMS: 2_000,
		IngestionMS:       2_100,
		DecisionMS:        2_200,
		NextTradableMS:    3_000,
		FillMS:            3_000,
	}
	if err := valid.Validate(); err != nil {
		t.Fatalf("truthful observation rejected: %v", err)
	}

	tests := []struct {
		name   string
		mutate func(*Observation)
	}{
		{name: "backdated availability", mutate: func(o *Observation) { o.SourceAvailableMS = o.SourceEventMS - 1 }},
		{name: "ingestion before availability", mutate: func(o *Observation) { o.IngestionMS = o.SourceAvailableMS - 1 }},
		{name: "decision before ingestion", mutate: func(o *Observation) { o.DecisionMS = o.IngestionMS - 1 }},
		{name: "tradable before decision", mutate: func(o *Observation) { o.NextTradableMS = o.DecisionMS - 1 }},
		{name: "fill before tradable", mutate: func(o *Observation) { o.FillMS = o.NextTradableMS - 1 }},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			observation := valid
			tt.mutate(&observation)
			if err := observation.Validate(); err == nil {
				t.Fatal("invalid temporal ordering passed")
			}
		})
	}
}

func TestObservationAllowsAbsentNonApplicableTimes(t *testing.T) {
	if err := (Observation{SourceEventMS: 1_000, SourceAvailableMS: 2_000}).Validate(); err != nil {
		t.Fatalf("source-only observation rejected: %v", err)
	}
}

func TestValidateCandleClose(t *testing.T) {
	if err := ValidateCandleClose(1_000, 60_999, 60_000); err != nil {
		t.Fatalf("truthful candle close rejected: %v", err)
	}
	for _, closeTime := range []int64{0, 999, 1_000, 60_998, 61_000} {
		if err := ValidateCandleClose(1_000, closeTime, 60_000); err == nil {
			t.Fatalf("invalid close %d passed", closeTime)
		}
	}
}
