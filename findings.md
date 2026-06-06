# Findings

## Observations
- Phase 10.2 directive: formally reject `RegimeAwareCompressionBreakout_LONG` after OOS failure and begin cleaner multi-symbol alpha rediscovery.
- Prior candidate rejection facts supplied by user: LINK H2 PF after 5 bps `0.7720`, LINK H2 expectancy after 5 bps negative, LINK H2 positive months after cost `0`, LINK FY2023 PF after 5 bps `0.9644`, and multi-symbol OOS did not rescue the candidate.
- Phase 8.6 directive read from `../prompt.md`.
- Work scope is limited to `ak-engine`; do not modify `ak-trader`, promotion artifacts, shadow integration, or any runtime/order-placement path.
- Phase 8.5 strict improved out-of-sample loss but remains non-promotable, with remaining drag concentrated in long trades from the 70-84 score bucket.

## Key Constraints
- Phase 10.2 is research-only.
- Do not move toward promotion, shadow mode, or testnet.
- Do not modify "ak-trader".
- Do not add runtime configs, order placement, live execution behavior, exchange integration, testnet flow, shadow mode, or promotion logic.
- Work only in `ak-engine/`, with `ak-historian/` only if strictly needed.
- Keep sweep grid bounded.

## Phase 8.5 Results
- Strict local-json fixture backtest still loses, but its hard-block mix now surfaces chop explicitly (`15M_CHOP`) and disables probe behavior by config.
- Strict local-json fixture walk-forward remains a valid PASS with `promotion_candidate=false`; compared with prior fallback it reduces loss magnitude from `-720.8210441712104` test net PnL to `-255.53590483845485` on the 4-split sample, but still fails profitability.
- Strict local-parquet Jan 2023 backtest is materially negative (`net_pnl=-1515.1292370567169`, `profit_factor=0.3890146164478057`) despite stricter filters.
- Strict local-parquet Jan-Jun 2023 walk-forward is over-constrained on the current dataset (`selected_candidate_count=0` across `81216` tested candidates), so diagnostics now show filter failure clearly instead of producing a superficially robust result.

## Phase 8.6 Focus
- Add side-specific thresholds for entry score, trend, chop, expected move, and cost multiple.
- Add long/short-specific 70-84 bucket disables plus entry frequency guards.
- Add research-only calibration presets and a bounded calibration sweep profile.
- Keep valid losing runs as `PASS`; a losing strategy is not a simulator failure.

## Phase 10.2 Research Surface
- Existing `internal/app/evaluate_alpha_baselines.go` generates the required baseline families from features/regimes but only as a single-symbol report with 15m proxy metrics.
- Existing `internal/app/evaluate_compression_breakout.go` computes 60m cost haircuts, entry delay, monthly buckets, MFE/MAE, leakage, and acceptance gates for CompressionBreakout LONG.
- Existing generated 2023 context artifacts are present under `runs/features/*-2023-FY-context.json` and `runs/regimes/*-2023-FY-context.json` for the requested nine symbols.
- Feature/regime JSON files are large; avoid broad `rg`/`jq` over `runs/features` and `runs/regimes`. Use bounded file lists or command-level parsing.
- Phase 10.1 multi-symbol CompressionBreakout report already shows ETHUSDT PF after 5 bps `1.1156` and DOGEUSDT `1.0591`, but both failed concentration/gate robustness.
- For period split metrics, Phase 10.2 aggregation should only include events whose forward horizon remains inside the reported split to avoid borrowing return data from the next period.
