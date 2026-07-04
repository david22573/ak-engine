# Phase 11.0 - Non-Funding Candidate Family Design Plan

## Boundary

This is design and research planning only. No candidate code is implemented. No `ak-trader` files are modified, no promotion is allowed, and no live trading, order placement, exchange key, execution, or mainnet code is added.

Funding may remain a later context feature, but none of the Phase 11 candidates below uses funding as the primary trigger.

## Phase 10 Conclusion Summary

Phase 10 funding-event research is closed for the current data/features. Retained compact-summary coverage is complete for 8 symbols (`ADAUSDT`, `AVAXUSDT`, `BNBUSDT`, `DOGEUSDT`, `ETHUSDT`, `LINKUSDT`, `SOLUSDT`, `XRPUSDT`) across `2024-01` through `2025-12`, for 192 symbol-month summaries. `raw_required=false`.

Rejected funding-event families: NegativeFundingLong, PositiveFundingShort, FundingFlipLong/Short, RegimeFundingLong/Short, ConfirmedFundingExtreme, BreakoutFundingMomentum, and VolumeImbalanceFundingReversionProxy. Phase 10.12 concluded that current funding-event triggers are not promotable and Phase 11 should pivot away from funding-primary triggers.

## Available Feature Inventory

| Field name | Source | Availability | Leakage risk | Candidate uses |
| --- | --- | --- | --- | --- |
| `Close` | `features.Row` from candle close | READY_NOW | Low after `AvailableAtMS` | EMA relation, breakout direction, reversion distance |
| `Return1`, `Return5`, `Return15` | trailing close returns | READY_NOW | Low after candle close | momentum, shock direction, lead/lag proxy |
| `RealizedVol20`, `RealizedVol60` | trailing realized volatility | READY_NOW | Low after warmup | volatility regime, risk filter |
| `ATR14`, `ATRPct14` | trailing ATR | READY_NOW | Low after candle close | shock detection, stop sizing |
| `BBWidth20`, `BBWidthPctRank60` | Bollinger width and trailing percentile | READY_NOW | Low if trailing only | compression, range filter, breakout setup |
| `EMA20`, `EMA50`, `EMA200` | trailing EMAs | READY_NOW | Low after warmup | trend continuation, pullback, side filter |
| `TrendSlope20` | trailing linear slope | READY_NOW | Low after warmup | trend strength, chop filter |
| `VolumeRatio20`, `QuoteVolumeRatio20` | current volume over trailing 20-candle average | READY_NOW | Low after candle close | volume confirmation, liquidity filter |
| `TakerBuyRatio` | taker buy base volume / volume | READY_NOW as candle proxy | Medium if intrabar; low post-close | optional side confirmation; not true taker buy/sell join |
| `BTCReturn60`, `ETHReturn60` | context BTC/ETH candles | READY_NOW when context exists | Medium; timestamp alignment required | market beta, cross-symbol context |
| `regime.volatility` | `regime.Label` | READY_NOW | Low with as-of join | compression, shock, expansion filters |
| `regime.trend` | `regime.Label` | READY_NOW | Low with as-of join | trend/range/chop gates |
| `regime.liquidity` | `regime.Label` | READY_NOW | Low with as-of join | thin/heavy/abnormal liquidity filters |
| `regime.market_beta` | `regime.Label` from BTC context | READY_NOW with BTC context | Medium; context availability matters | risk-on/risk-off, beta agreement/divergence |
| `regime.composite` | `regime.Label` | READY_NOW | Low with as-of join | compact regime gates |
| OHLCV candle fields | `pkg/protocol.Candle` | READY_NOW | Low after close | failed breakout shape, excursion, stop/target tests |
| `EventTimeMS`, `AvailableAtMS` | feature and regime rows | READY_NOW | Critical guardrail | as-of joins, session windows, leakage validation |

Missing or limited fields:

| Field | Status | Note |
| --- | --- | --- |
| true taker buy/sell imbalance | BLOCKED_MISSING_DATA | Join support exists, but retained 10.11D used only `TakerBuyRatio` fallback. |
| open interest, liquidations, positioning | BLOCKED_MISSING_DATA | Future flow/positioning research data, not READY_NOW. |
| cross-symbol relative strength, basket dispersion | READY_WITH_SMALL_ENGINE_CHANGE or BLOCKED_MISSING_DATA | Derivable from synchronized features but not exposed as current retained-summary fields. |

## Candidate Family Table

| Rank | Candidate | Primary trigger | Side | Complexity | Feasibility |
| ---: | --- | --- | --- | --- | --- |
| 1 | CompressionVolumeBreakout | compressed regime/range plus EMA20 directional break and volume confirmation | both | EASY | READY_NOW |
| 2 | RegimeTrendPullbackContinuation | trend regime pullback toward EMA20 followed by return resumption | both | EASY | READY_NOW |
| 3 | ShockFadeRegimeFiltered | shock event after extreme `Return5`, faded only outside trend-confirmed context | both | EASY | READY_NOW |
| 4 | BetaDivergenceReversion | target short-term return diverges from BTC beta in weak local trend | both | MEDIUM | READY_NOW |
| 5 | VolumeMomentumContinuation | heavy liquidity with `Return5` and `Return15` aligned | both | EASY | READY_NOW |
| 6 | ThinChopMeanReversion | thin chop/range extension away from EMA20 with small target/time stop | both | MEDIUM | READY_NOW |
| 7 | UTCSessionWindowFilter | predeclared UTC liquidity windows as context for another trigger | both | MEDIUM | READY_WITH_SMALL_ENGINE_CHANGE |
| 8 | BasketDispersionLeadLag | target lag/lead versus multi-symbol basket dispersion after BTC impulse | both | HARD | BLOCKED_MISSING_DATA |

## Candidate Scopes

`CompressionVolumeBreakout` uses `regime.volatility=compressed` or `regime.composite=compressed_range`, a completed-candle directional break around `EMA20`, volume confirmation through `VolumeRatio20`, and market-beta agreement. Expected horizons are `15m`, `60m`, and `240m`. Main risks are false breakouts in thin chop and beta overfit. Robustness gates should include cost stress, one-candle delay, leave-one-symbol/month/quarter, and retained-summary compatibility.

`RegimeTrendPullbackContinuation` uses bull/bear trend context, pullback toward `EMA20`, side-consistent `Return5`/`Return15`, `EMA20`/`EMA50` alignment, and non-thin volume. Expected horizons are `15m`, `60m`, and `240m`. Main risks are late-trend exhaustion and lagged trend labels.

`ShockFadeRegimeFiltered` fades extreme `Return5` shocks only when trend and beta context do not confirm continuation. Expected horizons are `5m`, `15m`, and `60m`. Main risks are fading true acceleration and short-horizon cost drag.

`BetaDivergenceReversion` trades target divergence from BTC beta when local trend is range/chop. Expected horizons are `15m` and `60m`. Main risks are idiosyncratic news continuation and context-alignment leakage.

`VolumeMomentumContinuation` follows heavy-liquidity moves when `Return5` and `Return15` agree, with optional post-close `TakerBuyRatio` confirmation. Expected horizons are `5m`, `15m`, and `60m`. Main risk is overly common weak events.

`ThinChopMeanReversion` targets small, risk-controlled reversions in thin chop using low Bollinger rank, near-zero slope, small targets, and strict time stops. Expected horizons are `5m` and `15m`. Main risk is costs dominating small targets.

`UTCSessionWindowFilter` should be a predeclared context filter paired with another trigger, not a standalone strategy. Funding-time-adjacent behavior may be context only, never a trigger.

`BasketDispersionLeadLag` is deferred because relative strength and basket dispersion are not current retained-summary fields and require synchronized cross-symbol feature support.

## Ranked Recommendation

Recommend `Phase 11.1 - Implement Top Non-Funding Candidate Family`.

Implement exactly one candidate first: `CompressionVolumeBreakout`.

Reasons: it is READY_NOW, low-leakage with `AvailableAtMS` joins, independent from failed funding triggers, uses existing price/volatility/liquidity/regime fields, should have reasonable sample size, and is compatible with retained-summary promotion gates.

Second candidate: `RegimeTrendPullbackContinuation`.

Blocked/deferred candidate: `BasketDispersionLeadLag`.

## Rejected Or Deferred Ideas

| Idea | Decision | Reason |
| --- | --- | --- |
| Any funding-primary trigger | rejected for Phase 11 | Phase 10 families repeatedly failed. |
| SqueezeFundingUnwind | deferred | Needs real OI/liquidation/positioning data. |
| True taker buy/sell imbalance family | deferred | Current retained work has only `TakerBuyRatio` fallback. |
| Basket dispersion lead/lag | deferred | Needs cross-symbol relative-strength/basket-dispersion support. |
| Session-only strategy | rejected as standalone | Clock should be context/filter, not sole trigger. |

## Next Phase Prompt Outline

```text
Task: Phase 11.1 - Implement CompressionVolumeBreakout.

Boundaries:
- Do not modify ak-trader.
- Do not promote anything.
- Do not add live trading, order placement, exchange keys, execution, or mainnet logic.
- Do not fetch new data.
- Do not use funding as a primary trigger.
- Use current ak-engine retained features/regimes only.

Implementation scope:
- Add one research candidate family: CompressionVolumeBreakout.
- Trigger from compressed volatility/compressed_range regime plus completed-candle EMA20 directional break.
- Confirm with VolumeRatio20 and BBWidthPctRank60 expansion from prior completed rows.
- Use market_beta agreement as context.
- Evaluate long and short sides over 15m, 60m, and 240m.
- Validate leakage with AvailableAtMS.
- Produce JSON and Markdown evaluation reports.
- Run app tests and JSON validation.
```

## Final Label

`design_plan_complete`
