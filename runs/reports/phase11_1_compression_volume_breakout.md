# Phase 11.1 - CompressionVolumeBreakout Evaluation

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
- rejected: `6`
- fragile: `0`
- inconclusive: `0`
- research_lead: `0`
- missing_data: `0`
- unsupported_context: `0`

## Leaderboard
| Family | Side | Horizon | Verdict | Events | De-clustered | PF 5bps | Exp 5bps (bps) | Positive Months | Worst Q PF | Delay 1c Exp | 10bps PF | Failed Gates |
|---|---|---|---|---:|---:|---:|---:|---:|---:|---:|---:|---|
| CompressionVolumeBreakout | long | 15m | rejected | 2836 | 2693 | 0.6847 | -6.2012 | 51 | 0.4861 | -6.0811 | 0.5089 | H2 PF after 5 bps; H2 expectancy after 5 bps; FY PF after 5 bps; entry_delay_1c_expectancy_bps; worst_quarter_pf_5bps |
| CompressionVolumeBreakout | short | 15m | rejected | 2444 | 2311 | 0.7510 | -5.5725 | 62 | 0.5604 | -5.5273 | 0.5850 | H2 PF after 5 bps; H2 expectancy after 5 bps; FY PF after 5 bps; entry_delay_1c_expectancy_bps; worst_quarter_pf_5bps |
| CompressionVolumeBreakout | long | 240m | rejected | 2836 | 2693 | 0.8774 | -8.2003 | 85 | 0.5953 | -8.2457 | 0.8101 | H2 PF after 5 bps; H2 expectancy after 5 bps; FY PF after 5 bps; entry_delay_1c_expectancy_bps; worst_quarter_pf_5bps |
| CompressionVolumeBreakout | short | 240m | rejected | 2444 | 2311 | 0.8316 | -12.4881 | 82 | 0.6542 | -11.9995 | 0.7725 | H2 PF after 5 bps; H2 expectancy after 5 bps; FY PF after 5 bps; entry_delay_1c_expectancy_bps; worst_quarter_pf_5bps |
| CompressionVolumeBreakout | long | 60m | rejected | 2836 | 2693 | 0.7987 | -7.0086 | 65 | 0.6443 | -6.8240 | 0.6815 | H2 PF after 5 bps; H2 expectancy after 5 bps; FY PF after 5 bps; entry_delay_1c_expectancy_bps; worst_quarter_pf_5bps |
| CompressionVolumeBreakout | short | 60m | rejected | 2444 | 2311 | 0.8216 | -7.0701 | 82 | 0.4906 | -7.0471 | 0.7163 | H2 PF after 5 bps; H2 expectancy after 5 bps; FY PF after 5 bps; entry_delay_1c_expectancy_bps; worst_quarter_pf_5bps |

## Final Recommendation
CompressionVolumeBreakout rejected for promotion under Phase 11.1 gates
