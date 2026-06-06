package walkforward

func Validate(res *WalkForwardResult, cfg Config) {
	var reasons []string
	valid := true

	if res.AggregateTest.NetPnL <= 0 {
		valid = false
		reasons = append(reasons, "aggregate_test.net_pnl <= 0")
	}
	if res.AggregateTest.ProfitFactor < cfg.MinProfitFactor {
		valid = false
		reasons = append(reasons, "aggregate_test.profit_factor < min_profit_factor")
	}
	if res.AggregateTest.ProfitableSplitCount <= res.AggregateTest.LosingSplitCount {
		valid = false
		reasons = append(reasons, "profitable_split_count <= losing_split_count")
	}
	if res.AggregateTest.MaxConsecutiveLosses > cfg.MaxLossStreak {
		valid = false
		reasons = append(reasons, "aggregate_test.max_consecutive_losses > max_loss_streak")
	}
	if res.AggregateTest.TotalTrades < cfg.MinTrades {
		valid = false
		reasons = append(reasons, "aggregate_test.total_trades < min_trades")
	}

	res.PromotionCandidate = valid
	if len(reasons) > 0 {
		res.RejectionReasons = reasons
	} else {
		res.RejectionReasons = []string{}
	}
}
