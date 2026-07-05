# Phase 12.3 - DowntrendMidVolRelief Evaluation

## Boundary
- no funding primary trigger
- no ak-trader changes
- no data fetch
- retained summaries only
- no threshold tuning to force success

## Coverage
- Expected symbol-months: `192`
- Completed symbol-months: `192`
- Raw event detail retained: `false`

## Verdict Counts
- rejected: `2`
- fragile: `0`
- inconclusive: `0`
- research_lead: `0`
- missing_data: `0`
- unsupported_context: `0`

## Leaderboard
| Family | Side | Horizon | Verdict | Events | De-clustered | PF 5bps | Exp 5bps (bps) | Positive Months | Worst Q PF | Delay 1c Exp | 10bps PF | Failed Gates |
|---|---|---|---|---:|---:|---:|---:|---:|---:|---:|---:|---|
| DowntrendMidVolRelief | long | 240m | rejected | 329842 | 13178 | 1.1690 | 16.4731 | 125 | 0.8479 | 16.5544 | 1.1150 | worst_quarter_pf_5bps |
| DowntrendMidVolRelief | short | 240m | rejected | 0 | 0 | 0.0000 | 0.0000 | 0 | 0.0000 | 0.0000 | 0.0000 | H2 PF after 5 bps; H2 expectancy after 5 bps; FY PF after 5 bps; event_count; positive_month_count; entry_delay_1c_expectancy_bps; worst_quarter_pf_5bps; H1 event_count == 0; H2 event_count == 0 |

## Final Recommendation
DowntrendMidVolRelief rejected for promotion under Phase 12.3 gates
