package backtest

import (
	"fmt"

	"github.com/davidmiguel22573/ak-engine/internal/strategy"
)

type FillAction string

const (
	FillActionEntry FillAction = "entry"
	FillActionExit  FillAction = "exit"
)

func ApplySlippage(price float64, side strategy.Side, action FillAction, slippageBPS float64) (float64, error) {
	if price <= 0 {
		return 0, fmt.Errorf("price must be > 0")
	}
	if slippageBPS < 0 {
		return 0, fmt.Errorf("slippage_bps must be >= 0")
	}

	factor := slippageBPS / 10000
	switch side {
	case strategy.SideLong:
		if action == FillActionEntry {
			return price * (1 + factor), nil
		}
		return price * (1 - factor), nil
	case strategy.SideShort:
		if action == FillActionEntry {
			return price * (1 - factor), nil
		}
		return price * (1 + factor), nil
	default:
		return 0, fmt.Errorf("unsupported side %q", side)
	}
}
