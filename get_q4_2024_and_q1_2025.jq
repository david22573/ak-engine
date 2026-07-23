def get_stats(data; quarter):
  data | map(select(.side == "long" and .quarter == quarter)) | {
    quarter: quarter,
    event_count: (map(.stats.event_count) | add),
    de_clustered_event_count: (map(.stats.de_clustered_event_count) | add),
    pf_5bps: ((map(.stats.cost_stress[] | select(.cost_bps == 5) | .gross_profit_bps) | add) / (map(.stats.cost_stress[] | select(.cost_bps == 5) | .gross_loss_bps) | add)),
    expectancy_5bps: ((map(.stats.cost_stress[] | select(.cost_bps == 5) | .net_bps) | add) / (map(.stats.event_count) | add)),
    months: (
      group_by(.month) | map({
        month: .[0].month,
        event_count: (map(.stats.event_count) | add),
        pf_5bps: ((map(.stats.cost_stress[] | select(.cost_bps == 5) | .gross_profit_bps) | add) / (if (map(.stats.cost_stress[] | select(.cost_bps == 5) | .gross_loss_bps) | add) == 0 then 1 else (map(.stats.cost_stress[] | select(.cost_bps == 5) | .gross_loss_bps) | add) end)),
        net_bps: (map(.stats.cost_stress[] | select(.cost_bps == 5) | .net_bps) | add)
      }) | sort_by(.pf_5bps)
    ),
    symbols: (
      group_by(.symbol) | map({
        symbol: .[0].symbol,
        event_count: (map(.stats.event_count) | add),
        pf_5bps: ((map(.stats.cost_stress[] | select(.cost_bps == 5) | .gross_profit_bps) | add) / (if (map(.stats.cost_stress[] | select(.cost_bps == 5) | .gross_loss_bps) | add) == 0 then 1 else (map(.stats.cost_stress[] | select(.cost_bps == 5) | .gross_loss_bps) | add) end)),
        net_bps: (map(.stats.cost_stress[] | select(.cost_bps == 5) | .net_bps) | add)
      }) | sort_by(.pf_5bps)
    )
  };

{
  "2024-Q4": get_stats(.retained_summaries; "2024-Q4"),
  "2025-Q1": get_stats(.retained_summaries; "2025-Q1"),
  "2025-Q4": get_stats(.retained_summaries; "2025-Q4")
}
