.retained_summaries | map(select(.side == "long")) | group_by(.quarter) | map({
  quarter: .[0].quarter,
  event_count: (map(.stats.event_count) | add),
  de_clustered_event_count: (map(.stats.de_clustered_event_count) | add),
  cost_5bps: {
    gross_profit: (map(.stats.cost_stress[] | select(.cost_bps == 5) | .gross_profit_bps) | add),
    gross_loss: (map(.stats.cost_stress[] | select(.cost_bps == 5) | .gross_loss_bps) | add),
    net: (map(.stats.cost_stress[] | select(.cost_bps == 5) | .net_bps) | add),
    win_count: (map(.stats.cost_stress[] | select(.cost_bps == 5) | .win_count) | add),
    loss_count: (map(.stats.cost_stress[] | select(.cost_bps == 5) | .loss_count) | add)
  },
  cost_7_5bps: {
    gross_profit: (map(.stats.cost_stress[] | select(.cost_bps == 7.5) | .gross_profit_bps) | add),
    gross_loss: (map(.stats.cost_stress[] | select(.cost_bps == 7.5) | .gross_loss_bps) | add),
    net: (map(.stats.cost_stress[] | select(.cost_bps == 7.5) | .net_bps) | add),
    win_count: (map(.stats.cost_stress[] | select(.cost_bps == 7.5) | .win_count) | add),
    loss_count: (map(.stats.cost_stress[] | select(.cost_bps == 7.5) | .loss_count) | add)
  },
  cost_10bps: {
    gross_profit: (map(.stats.cost_stress[] | select(.cost_bps == 10) | .gross_profit_bps) | add),
    gross_loss: (map(.stats.cost_stress[] | select(.cost_bps == 10) | .gross_loss_bps) | add),
    net: (map(.stats.cost_stress[] | select(.cost_bps == 10) | .net_bps) | add),
    win_count: (map(.stats.cost_stress[] | select(.cost_bps == 10) | .win_count) | add),
    loss_count: (map(.stats.cost_stress[] | select(.cost_bps == 10) | .loss_count) | add)
  }
}) | map(
  . + {
    pf_5bps: (.cost_5bps.gross_profit / (if .cost_5bps.gross_loss == 0 then 1 else .cost_5bps.gross_loss end)),
    pf_7_5bps: (.cost_7_5bps.gross_profit / (if .cost_7_5bps.gross_loss == 0 then 1 else .cost_7_5bps.gross_loss end)),
    pf_10bps: (.cost_10bps.gross_profit / (if .cost_10bps.gross_loss == 0 then 1 else .cost_10bps.gross_loss end)),
    expectancy_5bps: (.cost_5bps.net / (if .event_count == 0 then 1 else .event_count end)),
    expectancy_7_5bps: (.cost_7_5bps.net / (if .event_count == 0 then 1 else .event_count end)),
    expectancy_10bps: (.cost_10bps.net / (if .event_count == 0 then 1 else .event_count end)),
    win_rate_5bps: (.cost_5bps.win_count / (if .event_count == 0 then 1 else .event_count end) * 100),
    win_rate_7_5bps: (.cost_7_5bps.win_count / (if .event_count == 0 then 1 else .event_count end) * 100),
    win_rate_10bps: (.cost_10bps.win_count / (if .event_count == 0 then 1 else .event_count end) * 100)
  }
)
