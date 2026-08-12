// Package temporal defines the canonical ordering of research timestamps.
package temporal

import "fmt"

const ContractID = "ak.temporal.observation.v1"

// Observation records the timestamps that can exist between a source event
// and an execution. Zero-valued optional timestamps mean that the concept does
// not apply at the validation boundary. Source event and availability are
// always required.
//
// When present, the canonical ordering is:
//
//	source event <= source availability <= ingestion <= decision
//	    <= next tradable <= fill
//
// Ingestion may be omitted when the source artifact does not carry it.
// NextTradableMS and FillMS are deliberately defined here but become
// authoritative for evaluation only through AK-RM-004's execution-series
// specification.
type Observation struct {
	SourceEventMS     int64
	SourceAvailableMS int64
	IngestionMS       int64
	DecisionMS        int64
	NextTradableMS    int64
	FillMS            int64
}

func (o Observation) Validate() error {
	if o.SourceEventMS <= 0 {
		return fmt.Errorf("%s: source event must be positive", ContractID)
	}
	if o.SourceAvailableMS <= 0 {
		return fmt.Errorf("%s: source availability must be positive", ContractID)
	}
	if o.SourceAvailableMS < o.SourceEventMS {
		return fmt.Errorf("%s: source availability %d precedes source event %d", ContractID, o.SourceAvailableMS, o.SourceEventMS)
	}
	if o.IngestionMS != 0 && o.IngestionMS < o.SourceAvailableMS {
		return fmt.Errorf("%s: ingestion %d precedes source availability %d", ContractID, o.IngestionMS, o.SourceAvailableMS)
	}
	if o.DecisionMS != 0 {
		lowerBound := o.SourceAvailableMS
		if o.IngestionMS > lowerBound {
			lowerBound = o.IngestionMS
		}
		if o.DecisionMS < lowerBound {
			return fmt.Errorf("%s: decision %d precedes available input %d", ContractID, o.DecisionMS, lowerBound)
		}
	}
	if o.NextTradableMS != 0 {
		if o.DecisionMS == 0 {
			return fmt.Errorf("%s: next tradable time requires decision time", ContractID)
		}
		if o.NextTradableMS < o.DecisionMS {
			return fmt.Errorf("%s: next tradable time %d precedes decision %d", ContractID, o.NextTradableMS, o.DecisionMS)
		}
	}
	if o.FillMS != 0 {
		if o.DecisionMS == 0 {
			return fmt.Errorf("%s: fill time requires decision time", ContractID)
		}
		lowerBound := o.DecisionMS
		if o.NextTradableMS > lowerBound {
			lowerBound = o.NextTradableMS
		}
		if o.FillMS < lowerBound {
			return fmt.Errorf("%s: fill %d precedes tradable boundary %d", ContractID, o.FillMS, lowerBound)
		}
	}
	return nil
}

// ValidateCandleClose proves that a close-derived value becomes available at
// the exchange candle's inclusive close timestamp. It never estimates or
// backdates a missing/invalid close.
func ValidateCandleClose(openTimeMS, closeTimeMS, intervalMS int64) error {
	if intervalMS <= 0 {
		return fmt.Errorf("%s: candle interval must be positive", ContractID)
	}
	if err := (Observation{
		SourceEventMS:     openTimeMS,
		SourceAvailableMS: closeTimeMS,
		DecisionMS:        closeTimeMS,
	}).Validate(); err != nil {
		return err
	}
	wantClose := openTimeMS + intervalMS - 1
	if closeTimeMS != wantClose {
		return fmt.Errorf("%s: candle close %d does not equal interval close %d", ContractID, closeTimeMS, wantClose)
	}
	return nil
}
