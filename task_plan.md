# Task Plan - Fast Accumulation Phase 7

## Status
- **Goal**: Implement Phase 7 Fast Accumulation strategy research and Phase 7.5 diagnostics/sweep tools in `ak-engine` only.
- **Current Phase**: Phase 7.5 - Fast Accumulation Diagnostics and Tuning Foundation
- **Progress**: 100%
- **Verification**: Re-verified on 2026-06-03 against `directive.md`; required commands and local-parquet diagnostics/sweep all passed.

## Phases

### Phase 1: Recovery, Baseline Validation, and Existing Diff Review
- [x] Read directive and repo state
- [x] Read existing strategy/backtest code
- [x] Confirm Phase 6 commands are green
- [x] Capture existing dirty-worktree context before edits
- Status: `complete`

### Phase 2: Fast Accumulation Strategy Design
- [x] Define `WindowDecision` model and action set
- [x] Define Fast Accumulation config and deterministic scoring rules
- [x] Define 15m window aggregation and no-lookahead behavior
- Status: `complete`

### Phase 3: Strategy and Backtest Implementation
- [x] Add `internal/strategy/window_decision.go`
- [x] Add `internal/strategy/scoring.go`
- [x] Add `internal/strategy/mtf.go`
- [x] Add `internal/strategy/fast_accumulation.go`
- [x] Update strategy wiring, backtest engine, report, CLI, and Makefile
- Status: `complete`

### Phase 4: Tests
- [x] Add window decision and scoring tests
- [x] Add no-lookahead coverage
- [x] Add backtest integration coverage
- [x] Add Makefile target coverage
- Status: `complete`

### Phase 5: Verification
- [x] Run `go fmt ./...`
- [x] Run `go test ./...`
- [x] Run `go vet ./...`
- [x] Run `make ci`
- [x] Run `make proof-backtest-local`
- [x] Run `make proof-fast-accumulation-local`
- [x] Run optional local-parquet smoke if local historian data exists
- Status: `complete`

### Phase 7.5: Fast Accumulation Diagnostics and Tuning Foundation
- [x] Add optional `--include-decisions` CLI flag and Decisions export in Report JSON
- [x] Extend Trade struct with diagnostic fields (mae_bps, mfe_bps, hold_windows, r_multiple, etc.)
- [x] Add Fast Accumulation summary buckets to Report (trades_by_action, score buckets, hard blocks, etc.)
- [x] Add sweep subcommand to evaluate combinations of parameters on pre-loaded candles
- [x] Add Makefile targets `proof-fast-accumulation-diagnostics-local` and `proof-fast-accumulation-sweep-local`
- [x] Add unit tests for include-decisions, diagnostics, and sweep subcommand
- Status: `complete`

## Errors Encountered
| Error | Attempt | Resolution |
|-------|---------|------------|
| Strategy test compared `WindowDecision` values with slice fields | 1 | Switched to `reflect.DeepEqual` |
| "math" imported but not used in sweep.go | 1 | Cleaned unused imports |
| undefined: math in backtest_test.go | 1 | Added `"math"` import in test file |
| Global flags leakage across command test executions | 1 | Created `resetGlobals()` helper called before test command executes |

## Verification Notes
- Existing dirty worktree already contained full Phase 7.5 implementation when this session started.
- 2026-06-03 verification confirmed directive acceptance without `ak-trader` changes or walk-forward work.
- Local-parquet diagnostics reproduced baseline loss case: `2976` decisions, `428` trades, `net_pnl = -2108.498336430317`, `ending_cash = 7891.501663569686`.
- Local-parquet sweep evaluated `432` combinations and found profitable top configs on January 2023, led by:
  - `full_trade_min_score=90`
  - `normal_trade_min_score=80`
  - `probe_min_score=55/60/65`
  - `cost_multiple_required=3/4`
  - `max_hold_windows=4`
  - `time_stop_windows=2`
  - `allow_probe_trade=false`
  - Result: `24` trades, `net_pnl = 86.20685951268678`, `profit_factor = 1.1582641812886265`, `max_drawdown = 242.94210568323797`
