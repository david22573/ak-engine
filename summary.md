# Fast Accumulation Verification Closeout Summary

## Scope
- Repo: `ak-engine`
- Directive source: `../directive.md`
- Stop condition: close walk-forward metrics patch with full verification, then stop before strategy refinement
- Explicitly excluded: `ak-trader` changes, shadow integration, promotion artifacts

## Verification Commands Run
- `go fmt ./...`
- `go test ./...`
- `go vet ./...`
- `make ci`
- `make proof-backtest-local`
- `make proof-fast-accumulation-local`
- `make proof-fast-accumulation-diagnostics-local`
- `make proof-fast-accumulation-sweep-local`
- `make proof-walk-forward-local`
- `go run ./cmd/ak-engine walk-forward --source local-parquet --path ../ak-historian/.ak-historian/work --market futures-um --symbol LINKUSDT --interval 1m --from 2023-01-01 --to 2023-06-30 --strategy fast_accumulation --train-window 60d --test-window 30d --format json`

## Full Gate Results
- `go fmt ./...`: PASS
- `go test ./...`: PASS
- `go vet ./...`: PASS
- `make ci`: PASS
- `make proof-backtest-local`: PASS
- `make proof-fast-accumulation-local`: PASS
- `make proof-fast-accumulation-diagnostics-local`: PASS
- `make proof-fast-accumulation-sweep-local`: PASS
- `make proof-walk-forward-local`: PASS

## Files In Scope
- `Makefile`
- `internal/app/walk_forward.go`
- `internal/walkforward/config.go`
- `internal/walkforward/split.go`
- `internal/walkforward/runner.go`
- `internal/walkforward/result.go`
- `internal/walkforward/selection.go`
- `internal/walkforward/validation.go`
- `internal/walkforward/walkforward_test.go`
- `testdata/candles/btc_5m_walk_forward_sample.json`

## Findings

### proof-walk-forward-local
- Command:
  - `go run ./cmd/ak-engine walk-forward --source local-json --path testdata/candles/btc_5m_walk_forward_sample.json --market futures-um --symbol BTCUSDT --interval 5m --from 2024-01-01 --to 2024-01-02 --strategy fast_accumulation --train-window 45m --test-window 15m --format json`
  - result: PASS
  - `split_count = 188`
  - `candidate_count = 81216`
  - `selected_candidate_count = 0`
  - `promotion_candidate = false`
  - rejection reasons:
    - `aggregate_test.net_pnl <= 0`
    - `profitable_split_count <= losing_split_count`
    - `aggregate_test.total_trades < min_trades`

### Accepted local-parquet Walk-Forward Rerun
- Dataset: `LINKUSDT`, `1m`, `2023-01-01` to `2023-06-30`
- exit code: `0`
- elapsed: `2m8.936s`
- stderr: only benign D-Bus sandbox noise
- `split_count = 4`
- `candidate_count = 1728`
- `selected_candidate_count = 20`
- Aggregate train:
  - `total_trades = 335`
  - `wins = 133`
  - `losses = 202`
  - `win_rate = 0.3970149253731343`
  - `net_pnl = -246.58764786099897`
  - `fees_paid = 1778.5752766018747`
  - `slippage_paid = 355.7150741973511`
  - `profit_factor = 0.9517926384494908`
  - `max_drawdown = 516.1258098672606`
  - `max_consecutive_losses = 10`
  - `expectancy = -0.7360825309283552`
  - `average_hold_minutes = 15.113432835820895`
- Aggregate test:
  - `total_trades = 161`
  - `wins = 62`
  - `losses = 99`
  - `win_rate = 0.38509316770186336`
  - `net_pnl = -720.8210441712104`
  - `fees_paid = 857.2835444334341`
  - `slippage_paid = 171.4567119658609`
  - `profit_factor = 0.700862517381869`
  - `max_drawdown = 589.338229720146`
  - `max_consecutive_losses = 8`
  - `expectancy = -4.4771493426783255`
  - `average_hold_minutes = 14.937888198757763`
- `promotion_candidate = false`
- rejection reasons:
  - `aggregate_test.net_pnl <= 0`
  - `profitable_split_count <= losing_split_count`

### Latest directive rerun on 2026-06-04
- Same accepted fallback command rerun after interruption recovery.
- exit code: `0`
- elapsed: `123.903s`
- stderr: timing-only capture; no Go/app error text recorded
- aggregate train `average_hold_minutes = 15.113432835820895`
- aggregate test `average_hold_minutes = 14.937888198757763`
- `promotion_candidate = false`
- rejection reasons unchanged:
  - `aggregate_test.net_pnl <= 0`
  - `profitable_split_count <= losing_split_count`

### Aggregate Hold-Minutes Bug Result
- Before fix: aggregate `average_hold_minutes` reported `0` in walk-forward aggregate output despite non-zero per-candidate hold times.
- After fix and full verification rerun:
  - aggregate train `average_hold_minutes = 15.113432835820895`
  - aggregate test `average_hold_minutes = 14.937888198757763`
- Scope of fix: reporting correctness only. Promotion verdict and OOS economics did not improve.

### Top Selected Candidate Family
- No single family dominated all splits; winning family changed by split.
- Split 1 best train/test candidate:
  - `full_trade_min_score=90`
  - `normal_trade_min_score=80`
  - `probe_min_score=55`
  - `cost_multiple_required=4`
  - `max_hold_windows=4`
  - `time_stop_windows=2`
  - `allow_probe_trade=false`
- Split 4 best train/test candidate:
  - `full_trade_min_score=85`
  - `normal_trade_min_score=70`
  - `probe_min_score=65`
  - `cost_multiple_required=5`
  - `max_hold_windows=4`
  - `time_stop_windows=1`
  - `allow_probe_trade=false`
- Pattern across selected families:
  - `allow_probe_trade=false`
  - `max_hold_windows=4` favored often
  - `time_stop_windows=1/2`
  - high thresholds (`90/80`) still common, but not robust out-of-sample

## Conclusion
- Full verification closeout is complete.
- Walk-forward metrics patch is verified end-to-end in `ak-engine`.
- Accepted fallback rerun confirms aggregate `average_hold_minutes` no longer sticks at `0`.
- `promotion_candidate` remains `false` because aggregate OOS economics are still bad.
- Rejection reasons remain:
  - `aggregate_test.net_pnl <= 0`
  - `profitable_split_count <= losing_split_count`
- Phase 8.5 is ready to begin next pass, but was not started here.
