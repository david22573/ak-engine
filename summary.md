# Fast Accumulation Phase 7.5 Summary

## Scope
- Repo: `ak-engine`
- Directive source: `../directive.md`
- Stop condition: complete Phase 7.5 only
- Explicitly excluded: `ak-trader` changes, shadow integration, walk-forward, promotion artifacts

## What Was Verified
- Optional `--include-decisions` flag exists on `backtest`
- Full decision export is omitted by default and included when flag is set
- Trade diagnostics are populated in backtest JSON
- Fast Accumulation summary buckets are populated in report JSON
- `sweep` command exists and evaluates parameter combinations
- Makefile targets exist for diagnostics and sweep proof flows
- Required tests and local proof commands pass

## Files Implementing Phase 7.5
- `Makefile`
- `internal/app/backtest.go`
- `internal/app/sweep.go`
- `internal/app/backtest_test.go`
- `internal/app/sweep_test.go`
- `internal/app/makefile_test.go`
- `internal/backtest/*`
- `internal/strategy/*`
- `testdata/candles/btc_5m_fast_accumulation_sample.json`

## Findings

### Decision Export
- `--include-decisions=false`: `decisions` array omitted
- `--include-decisions=true`: report includes per-window Fast Accumulation decisions with:
  - `window_start_ms`
  - `window_end_ms`
  - `action`
  - `confidence`
  - `long_score`
  - `short_score`
  - `chop_score`
  - `volatility_score`
  - `trend_score`
  - `pullback_score`
  - `breakout_score`
  - `expected_move_bps`
  - `estimated_cost_bps`
  - `reason_codes`
  - `risk_fraction`

### Trade Diagnostics
- Backtest trades include:
  - `entry_window_ms`
  - `exit_window_ms`
  - `entry_reason_codes`
  - `exit_reason`
  - `score_at_entry`
  - `risk_fraction`
  - `estimated_cost_bps`
  - `expected_move_bps`
  - `r_multiple`
  - `mae_bps`
  - `mfe_bps`
  - `hold_windows`
  - `entry_action`

### Summary Buckets
- Report includes:
  - `trades_by_action`
  - `pnl_by_action`
  - `trades_by_score_bucket`
  - `pnl_by_score_bucket`
  - `win_rate_by_score_bucket`
  - `avg_pnl_by_score_bucket`
  - `hard_blocks_by_reason`
  - `losses_by_reason_code`
  - `fees_by_action`
  - `slippage_by_action`

### Sweep Command
- Command: `go run ./cmd/ak-engine sweep ...`
- Grid size: `432` parameter combinations
- Sorting verified:
  - `net_pnl` descending
  - then `max_drawdown` ascending
  - then `max_consecutive_losses` ascending

## Verification Commands Run
- `go fmt ./...`
- `go test ./...`
- `go vet ./...`
- `make ci`
- `make proof-backtest-local`
- `make proof-fast-accumulation-local`
- `make proof-fast-accumulation-diagnostics-local`
- `make proof-fast-accumulation-sweep-local`
- `go run ./cmd/ak-engine backtest --source local-parquet --path ../ak-historian/.ak-historian/work --market futures-um --symbol LINKUSDT --interval 1m --from 2023-01-01 --to 2023-01-31 --strategy fast_accumulation --format json --include-decisions`
- `go run ./cmd/ak-engine sweep --source local-parquet --path ../ak-historian/.ak-historian/work --market futures-um --symbol LINKUSDT --interval 1m --from 2023-01-01 --to 2023-01-31 --strategy fast_accumulation --format json`

## Key Results

### Diagnostics Smoke
- Dataset: `LINKUSDT`, `1m`, `2023-01-01` to `2023-01-31`
- `window_decision_count = 2976`
- `total_trades = 428`
- `net_pnl = -2108.498336430317`
- `ending_cash = 7891.501663569686`
- `profit_factor = 0.4905036099736383`

### Baseline Loss Drivers
- All action buckets were negative in baseline month:
  - `FULL_LONG = -562.8824921126006`
  - `FULL_SHORT = -534.6683843042117`
  - `PROBE_LONG = -475.2533595747022`
  - `PROBE_SHORT = -519.6654654616528`
  - `REVERSE = -16.02863497715034`
- Hard blocks concentrated in:
  - `15M_CHOP = 432`
  - `EXPECTED_MOVE_BELOW_COST = 381`
- Worst trade concentration by score bucket:
  - `55-69`: `301` trades, `-900.8556469905849`
  - `70-84`: `104` trades, `-990.8446013023208`

### Sweep Winners
- Best verified config family:
  - `full_trade_min_score=90`
  - `normal_trade_min_score=80`
  - `probe_min_score=55/60/65`
  - `cost_multiple_required=3/4`
  - `max_hold_windows=4`
  - `time_stop_windows=2`
  - `allow_probe_trade=false`
- Best verified result:
  - `total_trades = 24`
  - `wins = 9`
  - `losses = 15`
  - `win_rate = 0.375`
  - `net_pnl = 86.20685951268678`
  - `fees_paid = 124.22251696612119`
  - `slippage_paid = 24.84450574595988`
  - `profit_factor = 1.1582641812886265`
  - `max_drawdown = 242.94210568323797`
  - `max_consecutive_losses = 6`
  - `expectancy = 3.591952479695282`
  - `ending_cash = 10086.206859512686`

## Conclusion
- Phase 7.5 is complete.
- Diagnostics and sweep foundation now support evidence-driven tuning.
- Next recommended phase: Phase 8 walk-forward using top sweep candidates, not baseline parameters.
