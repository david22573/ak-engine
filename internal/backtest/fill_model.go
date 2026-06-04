package backtest

import (
	"fmt"

	"github.com/davidmiguel22573/ak-engine/internal/strategy"
	"github.com/davidmiguel22573/ak-engine/pkg/protocol"
)

func ResolveExitPrice(pos Position, candle protocol.Candle) (float64, ExitReason, bool, error) {
	switch pos.Side {
	case strategy.SideLong:
		stopHit := candle.Low <= pos.StopPrice
		targetHit := candle.High >= pos.TargetPrice
		switch {
		case stopHit && targetHit:
			return pos.StopPrice, ExitReasonStopLoss, true, nil
		case stopHit:
			return pos.StopPrice, ExitReasonStopLoss, true, nil
		case targetHit:
			return pos.TargetPrice, ExitReasonTakeProfit, true, nil
		default:
			return 0, "", false, nil
		}
	case strategy.SideShort:
		stopHit := candle.High >= pos.StopPrice
		targetHit := candle.Low <= pos.TargetPrice
		switch {
		case stopHit && targetHit:
			return pos.StopPrice, ExitReasonStopLoss, true, nil
		case stopHit:
			return pos.StopPrice, ExitReasonStopLoss, true, nil
		case targetHit:
			return pos.TargetPrice, ExitReasonTakeProfit, true, nil
		default:
			return 0, "", false, nil
		}
	default:
		return 0, "", false, fmt.Errorf("unsupported side %q", pos.Side)
	}
}
