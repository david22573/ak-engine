# Findings - Fast Accumulation Phase 7

## Requirements
- Implement Phase 7 only in `ak-engine`.
- Confirm Phase 6 remains green before continuing.
- No `ak-trader` changes, no live trading, no testnet runtime, no walk-forward, no promotion artifacts.
- Every completed 15m window must emit deterministic `WindowDecision`.
- Add trade diagnostics (MAE, MFE, hold windows, entry window start MS, R-multiple, etc.)
- Add decision summary buckets (trades/PnL/fees/slippage by action, trades/PnL/winrate/avg PnL by score bucket, hard blocks by reason, losses by reason code)
- Add sweep subcommand with sorting (net PnL desc, drawdown asc, consecutive losses asc) and pre-loading optimization.

## Repo Findings
- Worktree already dirty in `Makefile`, `internal/app/backtest.go`, new `internal/app/backtest_test.go`, new `internal/app/makefile_test.go`, new `internal/backtest/*`, new `internal/strategy/*`.
- Existing uncommitted work appears to be Phase 6 baseline backtest implementation. Must preserve and extend carefully.
- `go test ./...`, `go vet ./...`, `make ci`, and `make proof-backtest-local` were green before Phase 7 changes.
- Fast Accumulation uses internal 15m aggregation on top of current 1m/5m candle loading. No loader changes needed.
- Current engine still supports one position at a time. Add-to-winner behavior is modeled as `HOLD`, not pyramiding.
- Host emits benign D-Bus warnings during some `go vet`/`make` commands in this environment, but command exit codes remain `0`.
- Optional local parquet smoke data exists at `../ak-historian/.ak-historian/work`.
- Grid sweep over 432 combinations evaluates instantly (less than 1s) on small local fixtures, and runs in about 8s on full 44640 candles monthly dataset.

## Research Findings
- On the `LINKUSDT` `1m` monthly dataset (January 2023, 44640 candles):
  - Baseline Strategy: 428 trades, net PnL `-2108.50`, ending cash `7891.50`, status `PASS`.
  - Decision diagnostics confirm loss concentration in score buckets `55-69` and `70-84`, with `15M_CHOP` and `EXPECTED_MOVE_BELOW_COST` as dominant hard blocks.
  - Baseline action buckets are all negative:
    - `FULL_LONG`: `-562.88`
    - `FULL_SHORT`: `-534.67`
    - `PROBE_LONG`: `-475.25`
    - `PROBE_SHORT`: `-519.67`
    - `REVERSE`: `-16.03`
  - Grid sweep evaluated `432` combinations.
  - Verified top configurations on current code are profitable on this month:
    - Best verified family: `full_trade_min_score=90`, `normal_trade_min_score=80`, `probe_min_score=55/60/65`, `cost_multiple_required=3/4`, `max_hold_windows=4`, `time_stop_windows=2`, `allow_probe_trade=false`
    - Best result: `24` trades, `9` wins, `15` losses, `net_pnl=86.20685951268678`, `profit_factor=1.1582641812886265`, `max_drawdown=242.94210568323797`, `ending_cash=10086.206859512686`
  - Earlier note claiming best sweep stayed negative was stale versus current verified code/output.

## Technical Decisions
| Decision | Rationale |
|----------|-----------|
| Use repo-local planning files in `ak-engine` | Task target is `ak-engine`, root plan is unrelated |
| Keep `Strategy.OnCandle` interface and extend `Signal`/`State` | Preserve baseline behavior while adding decisions, risk fractions, exits, and reversals |
| Aggregate decisions in strategy and summarize in backtest report | Keep package boundaries clean and avoid global state |
| Use deterministic score ladder plus hard blocks | Required for research/backtest-only behavior and testability |
| Pre-load candles once for the parameter sweep | Reduces I/O cost from 432x disk reads to 1x, making the sweep complete in seconds |
| Reset global package variables inside test run helpers | Prevents state leakage when flags are parsed sequentially across tests in the same process |

## Issues Encountered
| Issue | Resolution |
|-------|------------|
| Strategy test compared structs with slice fields | Switched to `reflect.DeepEqual` |
| Global command variables leaking state between tests | Reset variables inside test run helpers using a dedicated `resetGlobals` utility |

## Resources
- `../directive.md`
