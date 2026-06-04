package backtest

import "math"

type Metrics struct {
	TradeCount           int     `json:"trade_count"`
	WinCount             int     `json:"win_count"`
	LossCount            int     `json:"loss_count"`
	WinRate              float64 `json:"win_rate"`
	GrossPnL             float64 `json:"gross_pnl"`
	NetPnL               float64 `json:"net_pnl"`
	TotalFees            float64 `json:"total_fees"`
	SlippagePaid         float64 `json:"slippage_paid"`
	AverageNetPnL        float64 `json:"average_net_pnl"`
	AverageReturnBPS     float64 `json:"average_return_bps"`
	ProfitFactor         float64 `json:"profit_factor"`
	MaxDrawdown          float64 `json:"max_drawdown"`
	MaxDrawdownPct       float64 `json:"max_drawdown_pct"`
	MaxConsecutiveLosses int     `json:"max_consecutive_losses"`
	AverageWin           float64 `json:"average_win"`
	AverageLoss          float64 `json:"average_loss"`
	Expectancy           float64 `json:"expectancy"`
	AverageHoldMinutes   float64 `json:"average_hold_minutes"`
	FinalEquity          float64 `json:"final_equity"`
	StartingEquity       float64 `json:"starting_equity"`
	OpenPositionCount    int     `json:"open_position_count"`
}

func ComputeMetrics(startingEquity, finalEquity float64, trades []Trade, equityCurve []float64, openPositions int) Metrics {
	m := Metrics{
		TradeCount:        len(trades),
		FinalEquity:       finalEquity,
		StartingEquity:    startingEquity,
		OpenPositionCount: openPositions,
	}
	if len(equityCurve) == 0 {
		equityCurve = []float64{startingEquity, finalEquity}
	}

	var totalReturnBPS float64
	var grossWins float64
	var grossLosses float64
	var winSum float64
	var lossSum float64
	var totalHoldMinutes float64
	var consecutiveLosses int
	for _, trade := range trades {
		m.GrossPnL += trade.GrossPnL
		m.NetPnL += trade.NetPnL
		m.TotalFees += trade.EntryFee + trade.ExitFee
		m.SlippagePaid += trade.SlippagePaid
		totalReturnBPS += trade.NetReturnBPS
		totalHoldMinutes += float64(trade.ExitTimeMS-trade.EntryTimeMS) / 60000
		if trade.NetPnL > 0 {
			m.WinCount++
			grossWins += trade.NetPnL
			winSum += trade.NetPnL
			consecutiveLosses = 0
		} else if trade.NetPnL < 0 {
			m.LossCount++
			grossLosses += math.Abs(trade.NetPnL)
			lossSum += trade.NetPnL
			consecutiveLosses++
			if consecutiveLosses > m.MaxConsecutiveLosses {
				m.MaxConsecutiveLosses = consecutiveLosses
			}
		} else {
			consecutiveLosses = 0
		}
	}

	if m.TradeCount > 0 {
		m.WinRate = float64(m.WinCount) / float64(m.TradeCount)
		m.AverageNetPnL = m.NetPnL / float64(m.TradeCount)
		m.AverageReturnBPS = totalReturnBPS / float64(m.TradeCount)
		m.Expectancy = m.AverageNetPnL
		m.AverageHoldMinutes = totalHoldMinutes / float64(m.TradeCount)
	}
	if m.WinCount > 0 {
		m.AverageWin = winSum / float64(m.WinCount)
	}
	if m.LossCount > 0 {
		m.AverageLoss = lossSum / float64(m.LossCount)
	}
	if grossLosses > 0 {
		m.ProfitFactor = grossWins / grossLosses
	}

	peak := equityCurve[0]
	maxDrawdown := 0.0
	for _, equity := range equityCurve {
		if equity > peak {
			peak = equity
		}
		drawdown := peak - equity
		if drawdown > maxDrawdown {
			maxDrawdown = drawdown
		}
	}
	m.MaxDrawdown = maxDrawdown
	if peak > 0 {
		m.MaxDrawdownPct = math.Abs(maxDrawdown / peak)
	}

	return m
}
