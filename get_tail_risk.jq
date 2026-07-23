{
  worst_monthly_pf: (
    .retained_summaries | map(select(.side == "long")) | group_by(.month) | map({
      month: .[0].month,
      pf_5bps: ((map(.stats.cost_stress[] | select(.cost_bps == 5) | .gross_profit_bps) | add) / (if (map(.stats.cost_stress[] | select(.cost_bps == 5) | .gross_loss_bps) | add) == 0 then 1 else (map(.stats.cost_stress[] | select(.cost_bps == 5) | .gross_loss_bps) | add) end)),
      expectancy_5bps: ((map(.stats.cost_stress[] | select(.cost_bps == 5) | .net_bps) | add) / (if (map(.stats.event_count) | add) == 0 then 1 else (map(.stats.event_count) | add) end))
    }) | sort_by(.pf_5bps) | .[0]
  ),
  worst_monthly_expectancy: (
    .retained_summaries | map(select(.side == "long")) | group_by(.month) | map({
      month: .[0].month,
      expectancy_5bps: ((map(.stats.cost_stress[] | select(.cost_bps == 5) | .net_bps) | add) / (if (map(.stats.event_count) | add) == 0 then 1 else (map(.stats.event_count) | add) end))
    }) | sort_by(.expectancy_5bps) | .[0]
  ),
  worst_symbol_month: (
    .retained_summaries | map(select(.side == "long" and .stats.event_count > 0)) | map({
      symbol: .symbol,
      month: .month,
      pf_5bps: .stats.pf_after_5_bps,
      net_bps: .stats.net_bps,
      event_count: .stats.event_count
    }) | sort_by(.pf_5bps) | .[0]
  ),
  worst_symbol_month_by_net: (
    .retained_summaries | map(select(.side == "long" and .stats.event_count > 0)) | map({
      symbol: .symbol,
      month: .month,
      pf_5bps: .stats.pf_after_5_bps,
      net_bps: .stats.net_bps,
      event_count: .stats.event_count
    }) | sort_by(.net_bps) | .[0]
  )
}
