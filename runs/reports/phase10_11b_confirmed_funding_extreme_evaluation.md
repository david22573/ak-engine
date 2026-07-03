# Phase 10.11B ConfirmedFundingExtreme Evaluation

Date: 2026-07-03

## Result

- coverage_before: `132/192`
- coverage_after: `192/192`
- missing_after: `{}`
- full_evaluation_complete: `true`
- raw_required: `false`
- ak-trader untouched: `true`
- no promotion to ak-trader: `true`

## Phone Regeneration

- Required regenerated scope: `AVAXUSDT 2024-01..2025-12`, `SOLUSDT 2024-01..2025-12`, `LINKUSDT 2025-01..2025-12`.
- Repair scope after stale broad pull overwrite: `ADAUSDT`, `BNBUSDT`, `DOGEUSDT`, `ETHUSDT` all 2024-2025 plus `LINKUSDT 2024-01..2024-12`.
- Phone-side raw files for regenerated/repaired symbols remaining: `0`.
- Chromebook raw files remaining: `0`.

## Candidate Results

| Candidate | Label | Expectancy 5 bps | PF 5 bps | 7.5 bps stress | 10 bps stress | delay_1 |
| --- | --- | ---: | ---: | --- | --- | --- |
| `ConfirmedNegativeFundingLong|long|240m` | `REJECTED` | -2.014372 | 0.966931 | exp -4.514372, PF 0.927433 | exp -7.014372, PF 0.889589 | exp -4.269912, PF 0.867293 |
| `ConfirmedPositiveFundingShort|short|5m` | `REJECTED` | -5.164622 | 0.547049 | exp -7.664622, PF 0.413199 | exp -10.164622, PF 0.315360 | exp -6.558198, PF 0.792523 |

Strongest confirmed-family candidate: `ConfirmedNegativeFundingLong|long|240m`.

Both candidates remain `REJECTED`; neither became `RESEARCH_LEAD` nor `SHADOW_CANDIDATE`. No candidate became stronger in label or expectancy after full coverage.

## Robustness

| Candidate | Leave-one-symbol | Leave-one-month | Leave-one-quarter | Rejection reasons |
| --- | --- | --- | --- | --- |
| `ConfirmedNegativeFundingLong|long|240m` | `False` | `False` | `False` | `baseline_expectancy_non_positive, baseline_pf_non_positive, concentration_symbol, cost_7_5_fail, delay_1_expectancy_non_positive, leave_one_month_out_fail, leave_one_quarter_out_fail, leave_one_symbol_out_fail` |
| `ConfirmedPositiveFundingShort|short|5m` | `False` | `False` | `False` | `baseline_expectancy_non_positive, baseline_pf_non_positive, cost_7_5_fail, delay_1_expectancy_non_positive, leave_one_month_out_fail, leave_one_quarter_out_fail, leave_one_symbol_out_fail` |

## Boundary Checks

- ak-trader modified: `false`
- live trading/order/exchange-key/mainnet code added: `false`
- new data fetch performed: `false`
- R2 restore used: `false`
- threshold tuning performed: `false`
