package backtest

import "fmt"

type FeeConfig struct {
	MakerFeeBPS float64
	TakerFeeBPS float64
	UseMaker    bool
}

func CalculateFee(notional float64, cfg FeeConfig) (float64, error) {
	if notional < 0 {
		return 0, fmt.Errorf("notional must be >= 0")
	}
	bps := cfg.TakerFeeBPS
	if cfg.UseMaker {
		bps = cfg.MakerFeeBPS
	}
	if bps < 0 {
		return 0, fmt.Errorf("fee_bps must be >= 0")
	}
	return notional * bps / 10000, nil
}
