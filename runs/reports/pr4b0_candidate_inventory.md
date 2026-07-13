# PR4B0 Candidate Inventory

Accepted Engine baseline: `a04f6d6c8631e06049c7e108581dd638e9962a7b`.

Candidate records: **99**. Unknown implementations: **0**. Omitted candidates: **0**.

| Candidate | Family | Phase | Classification | Final status | Reproducible | Exclusion |
|---|---|---|---|---|---:|---|
| `baseline` | BaselineMomentum | Phase 10.4 | `REJECTED` | `REJECTED` | true | H2 out-of-sample failure |
| `fast_accumulation` | FastAccumulation | Phase 10.4 | `REJECTED` | `REJECTED` | true | H2 out-of-sample failure |
| `fast_accumulation_breakeven_guard` | FastAccumulation | Phase 5-10 preset | `MISSING_EVIDENCE` | `PIT_EVIDENCE_MISSING` | true | no complete immutable OOS, walk-forward, cost-tail-concentration, PIT, and controlled-holdout evidence for this exact preset |
| `fast_accumulation_breakout_retest` | FastAccumulation | Phase 5-10 preset | `MISSING_EVIDENCE` | `PIT_EVIDENCE_MISSING` | true | no complete immutable OOS, walk-forward, cost-tail-concentration, PIT, and controlled-holdout evidence for this exact preset |
| `fast_accumulation_cut_no_progress` | FastAccumulation | Phase 5-10 preset | `MISSING_EVIDENCE` | `PIT_EVIDENCE_MISSING` | true | no complete immutable OOS, walk-forward, cost-tail-concentration, PIT, and controlled-holdout evidence for this exact preset |
| `fast_accumulation_economics_guard` | FastAccumulation | Phase 5-10 preset | `MISSING_EVIDENCE` | `PIT_EVIDENCE_MISSING` | true | no complete immutable OOS, walk-forward, cost-tail-concentration, PIT, and controlled-holdout evidence for this exact preset |
| `fast_accumulation_momentum_continuation` | FastAccumulation | Phase 5-10 preset | `MISSING_EVIDENCE` | `PIT_EVIDENCE_MISSING` | true | no complete immutable OOS, walk-forward, cost-tail-concentration, PIT, and controlled-holdout evidence for this exact preset |
| `fast_accumulation_partial_trail` | FastAccumulation | Phase 5-10 preset | `MISSING_EVIDENCE` | `PIT_EVIDENCE_MISSING` | true | no complete immutable OOS, walk-forward, cost-tail-concentration, PIT, and controlled-holdout evidence for this exact preset |
| `fast_accumulation_pullback_reclaim` | FastAccumulation | Phase 5-10 preset | `MISSING_EVIDENCE` | `PIT_EVIDENCE_MISSING` | true | no complete immutable OOS, walk-forward, cost-tail-concentration, PIT, and controlled-holdout evidence for this exact preset |
| `fast_accumulation_strict` | FastAccumulation | Phase 5-10 preset | `MISSING_EVIDENCE` | `PIT_EVIDENCE_MISSING` | true | no complete immutable OOS, walk-forward, cost-tail-concentration, PIT, and controlled-holdout evidence for this exact preset |
| `fast_accumulation_strict_1h` | FastAccumulation | Phase 5-10 preset | `MISSING_EVIDENCE` | `PIT_EVIDENCE_MISSING` | true | no complete immutable OOS, walk-forward, cost-tail-concentration, PIT, and controlled-holdout evidence for this exact preset |
| `fast_accumulation_strict_30m` | FastAccumulation | Phase 5-10 preset | `MISSING_EVIDENCE` | `PIT_EVIDENCE_MISSING` | true | no complete immutable OOS, walk-forward, cost-tail-concentration, PIT, and controlled-holdout evidence for this exact preset |
| `fast_accumulation_strict_cost_guard` | FastAccumulation | Phase 5-10 preset | `MISSING_EVIDENCE` | `PIT_EVIDENCE_MISSING` | true | no complete immutable OOS, walk-forward, cost-tail-concentration, PIT, and controlled-holdout evidence for this exact preset |
| `fast_accumulation_strict_high_confidence` | FastAccumulation | Phase 5-10 preset | `MISSING_EVIDENCE` | `PIT_EVIDENCE_MISSING` | true | no complete immutable OOS, walk-forward, cost-tail-concentration, PIT, and controlled-holdout evidence for this exact preset |
| `fast_accumulation_strict_low_frequency` | FastAccumulation | Phase 5-10 preset | `MISSING_EVIDENCE` | `PIT_EVIDENCE_MISSING` | true | no complete immutable OOS, walk-forward, cost-tail-concentration, PIT, and controlled-holdout evidence for this exact preset |
| `fast_accumulation_strict_no_70_84_longs` | FastAccumulation | Phase 5-10 preset | `MISSING_EVIDENCE` | `PIT_EVIDENCE_MISSING` | true | no complete immutable OOS, walk-forward, cost-tail-concentration, PIT, and controlled-holdout evidence for this exact preset |
| `fast_accumulation_strict_short_bias` | FastAccumulation | Phase 5-10 preset | `MISSING_EVIDENCE` | `PIT_EVIDENCE_MISSING` | true | no complete immutable OOS, walk-forward, cost-tail-concentration, PIT, and controlled-holdout evidence for this exact preset |
| `funding-alpha/BreakoutFundingLong|long|240m` | BreakoutFundingLong | Phase 10.11C | `REJECTED` | `REJECTED` | true | failed net performance, robustness, and/or concentration gates |
| `funding-alpha/BreakoutFundingShort|short|5m` | BreakoutFundingShort | Phase 10.11C | `REJECTED` | `REJECTED` | true | failed net performance, robustness, and/or concentration gates |
| `funding-alpha/ConfirmedNegativeFundingLong|long|240m` | ConfirmedNegativeFundingLong | Phase 10.11B | `REJECTED` | `REJECTED` | true | failed net performance, robustness, and/or concentration gates |
| `funding-alpha/ConfirmedPositiveFundingShort|short|5m` | ConfirmedPositiveFundingShort | Phase 10.11B | `REJECTED` | `REJECTED` | true | failed net performance, robustness, and/or concentration gates |
| `funding-alpha/FundingFlipLong|long|120m` | FundingFlipLong | Phase 10.8 | `REJECTED` | `REJECTED` | true | candidate side/horizon row rejected after full retained coverage |
| `funding-alpha/FundingFlipLong|long|15m` | FundingFlipLong | Phase 10.8 | `REJECTED` | `REJECTED` | true | candidate side/horizon row rejected after full retained coverage |
| `funding-alpha/FundingFlipLong|long|240m` | FundingFlipLong | Phase 10.8 | `REJECTED` | `REJECTED` | true | candidate side/horizon row rejected after full retained coverage |
| `funding-alpha/FundingFlipLong|long|30m` | FundingFlipLong | Phase 10.8 | `REJECTED` | `REJECTED` | true | candidate side/horizon row rejected after full retained coverage |
| `funding-alpha/FundingFlipLong|long|5m` | FundingFlipLong | Phase 10.8 | `REJECTED` | `REJECTED` | true | candidate side/horizon row rejected after full retained coverage |
| `funding-alpha/FundingFlipLong|long|60m` | FundingFlipLong | Phase 10.8 | `REJECTED` | `REJECTED` | true | candidate side/horizon row rejected after full retained coverage |
| `funding-alpha/FundingFlipShort|short|120m` | FundingFlipShort | Phase 10.8 | `REJECTED` | `REJECTED` | true | candidate side/horizon row rejected after full retained coverage |
| `funding-alpha/FundingFlipShort|short|15m` | FundingFlipShort | Phase 10.8 | `REJECTED` | `REJECTED` | true | candidate side/horizon row rejected after full retained coverage |
| `funding-alpha/FundingFlipShort|short|240m` | FundingFlipShort | Phase 10.8 | `REJECTED` | `REJECTED` | true | candidate side/horizon row rejected after full retained coverage |
| `funding-alpha/FundingFlipShort|short|30m` | FundingFlipShort | Phase 10.8 | `REJECTED` | `REJECTED` | true | candidate side/horizon row rejected after full retained coverage |
| `funding-alpha/FundingFlipShort|short|5m` | FundingFlipShort | Phase 10.8 | `REJECTED` | `REJECTED` | true | candidate side/horizon row rejected after full retained coverage |
| `funding-alpha/FundingFlipShort|short|60m` | FundingFlipShort | Phase 10.8 | `REJECTED` | `REJECTED` | true | candidate side/horizon row rejected after full retained coverage |
| `funding-alpha/NegativeFundingLong|long|120m` | NegativeFundingLong | Phase 10.8 | `REJECTED` | `REJECTED` | true | candidate side/horizon row rejected after full retained coverage |
| `funding-alpha/NegativeFundingLong|long|15m` | NegativeFundingLong | Phase 10.8 | `REJECTED` | `REJECTED` | true | candidate side/horizon row rejected after full retained coverage |
| `funding-alpha/NegativeFundingLong|long|240m` | NegativeFundingLong | Phase 10.8 | `REJECTED` | `REJECTED` | true | candidate side/horizon row rejected after full retained coverage |
| `funding-alpha/NegativeFundingLong|long|30m` | NegativeFundingLong | Phase 10.8 | `REJECTED` | `REJECTED` | true | candidate side/horizon row rejected after full retained coverage |
| `funding-alpha/NegativeFundingLong|long|5m` | NegativeFundingLong | Phase 10.8 | `REJECTED` | `REJECTED` | true | candidate side/horizon row rejected after full retained coverage |
| `funding-alpha/NegativeFundingLong|long|60m` | NegativeFundingLong | Phase 10.8 | `REJECTED` | `REJECTED` | true | candidate side/horizon row rejected after full retained coverage |
| `funding-alpha/PositiveFundingShort|short|120m` | PositiveFundingShort | Phase 10.8 | `REJECTED` | `REJECTED` | true | candidate side/horizon row rejected after full retained coverage |
| `funding-alpha/PositiveFundingShort|short|15m` | PositiveFundingShort | Phase 10.8 | `REJECTED` | `REJECTED` | true | candidate side/horizon row rejected after full retained coverage |
| `funding-alpha/PositiveFundingShort|short|240m` | PositiveFundingShort | Phase 10.8 | `REJECTED` | `REJECTED` | true | candidate side/horizon row rejected after full retained coverage |
| `funding-alpha/PositiveFundingShort|short|30m` | PositiveFundingShort | Phase 10.8 | `REJECTED` | `REJECTED` | true | candidate side/horizon row rejected after full retained coverage |
| `funding-alpha/PositiveFundingShort|short|5m` | PositiveFundingShort | Phase 10.8 | `REJECTED` | `REJECTED` | true | candidate side/horizon row rejected after full retained coverage |
| `funding-alpha/PositiveFundingShort|short|60m` | PositiveFundingShort | Phase 10.8 | `REJECTED` | `REJECTED` | true | candidate side/horizon row rejected after full retained coverage |
| `funding-alpha/RegimeFundingLong|long|120m` | RegimeFundingLong | Phase 10.8 | `REJECTED` | `REJECTED` | true | candidate side/horizon row rejected after full retained coverage |
| `funding-alpha/RegimeFundingLong|long|15m` | RegimeFundingLong | Phase 10.8 | `REJECTED` | `REJECTED` | true | candidate side/horizon row rejected after full retained coverage |
| `funding-alpha/RegimeFundingLong|long|240m` | RegimeFundingLong | Phase 10.8 | `REJECTED` | `REJECTED` | true | candidate side/horizon row rejected after full retained coverage |
| `funding-alpha/RegimeFundingLong|long|30m` | RegimeFundingLong | Phase 10.8 | `REJECTED` | `REJECTED` | true | candidate side/horizon row rejected after full retained coverage |
| `funding-alpha/RegimeFundingLong|long|5m` | RegimeFundingLong | Phase 10.8 | `REJECTED` | `REJECTED` | true | candidate side/horizon row rejected after full retained coverage |
| `funding-alpha/RegimeFundingLong|long|60m` | RegimeFundingLong | Phase 10.8 | `REJECTED` | `REJECTED` | true | candidate side/horizon row rejected after full retained coverage |
| `funding-alpha/RegimeFundingShort|short|120m` | RegimeFundingShort | Phase 10.8 | `REJECTED` | `REJECTED` | true | candidate side/horizon row rejected after full retained coverage |
| `funding-alpha/RegimeFundingShort|short|15m` | RegimeFundingShort | Phase 10.8 | `REJECTED` | `REJECTED` | true | candidate side/horizon row rejected after full retained coverage |
| `funding-alpha/RegimeFundingShort|short|240m` | RegimeFundingShort | Phase 10.8 | `REJECTED` | `REJECTED` | true | candidate side/horizon row rejected after full retained coverage |
| `funding-alpha/RegimeFundingShort|short|30m` | RegimeFundingShort | Phase 10.8 | `REJECTED` | `REJECTED` | true | candidate side/horizon row rejected after full retained coverage |
| `funding-alpha/RegimeFundingShort|short|5m` | RegimeFundingShort | Phase 10.8 | `REJECTED` | `REJECTED` | true | candidate side/horizon row rejected after full retained coverage |
| `funding-alpha/RegimeFundingShort|short|60m` | RegimeFundingShort | Phase 10.8 | `REJECTED` | `REJECTED` | true | candidate side/horizon row rejected after full retained coverage |
| `funding-alpha/VolumeImbalanceFundingReversionProxyLong|long|120m` | VolumeImbalanceFundingReversionProxyLong | Phase 10.11D | `REJECTED` | `REJECTED` | true | failed net performance, robustness, and/or concentration gates |
| `funding-alpha/VolumeImbalanceFundingReversionProxyLong|long|15m` | VolumeImbalanceFundingReversionProxyLong | Phase 10.11D | `REJECTED` | `REJECTED` | true | failed net performance, robustness, and/or concentration gates |
| `funding-alpha/VolumeImbalanceFundingReversionProxyLong|long|240m` | VolumeImbalanceFundingReversionProxyLong | Phase 10.11D | `REJECTED` | `REJECTED` | true | failed net performance, robustness, and/or concentration gates |
| `funding-alpha/VolumeImbalanceFundingReversionProxyLong|long|30m` | VolumeImbalanceFundingReversionProxyLong | Phase 10.11D | `REJECTED` | `REJECTED` | true | failed net performance, robustness, and/or concentration gates |
| `funding-alpha/VolumeImbalanceFundingReversionProxyLong|long|5m` | VolumeImbalanceFundingReversionProxyLong | Phase 10.11D | `REJECTED` | `REJECTED` | true | failed net performance, robustness, and/or concentration gates |
| `funding-alpha/VolumeImbalanceFundingReversionProxyLong|long|60m` | VolumeImbalanceFundingReversionProxyLong | Phase 10.11D | `REJECTED` | `REJECTED` | true | failed net performance, robustness, and/or concentration gates |
| `funding-alpha/VolumeImbalanceFundingReversionProxyShort|short|120m` | VolumeImbalanceFundingReversionProxyShort | Phase 10.11D | `REJECTED` | `REJECTED` | true | failed net performance, robustness, and/or concentration gates |
| `funding-alpha/VolumeImbalanceFundingReversionProxyShort|short|15m` | VolumeImbalanceFundingReversionProxyShort | Phase 10.11D | `REJECTED` | `REJECTED` | true | failed net performance, robustness, and/or concentration gates |
| `funding-alpha/VolumeImbalanceFundingReversionProxyShort|short|240m` | VolumeImbalanceFundingReversionProxyShort | Phase 10.11D | `REJECTED` | `REJECTED` | true | failed net performance, robustness, and/or concentration gates |
| `funding-alpha/VolumeImbalanceFundingReversionProxyShort|short|30m` | VolumeImbalanceFundingReversionProxyShort | Phase 10.11D | `REJECTED` | `REJECTED` | true | failed net performance, robustness, and/or concentration gates |
| `funding-alpha/VolumeImbalanceFundingReversionProxyShort|short|5m` | VolumeImbalanceFundingReversionProxyShort | Phase 10.11D | `REJECTED` | `REJECTED` | true | failed net performance, robustness, and/or concentration gates |
| `funding-alpha/VolumeImbalanceFundingReversionProxyShort|short|60m` | VolumeImbalanceFundingReversionProxyShort | Phase 10.11D | `REJECTED` | `REJECTED` | true | failed net performance, robustness, and/or concentration gates |
| `phase11/CompressionVolumeBreakout|long|15m` | CompressionVolumeBreakout | Phase 11.1 | `REJECTED` | `REJECTED` | true | side/horizon row failed net and worst-period gates |
| `phase11/CompressionVolumeBreakout|long|240m` | CompressionVolumeBreakout | Phase 11.1 | `REJECTED` | `REJECTED` | true | side/horizon row failed net and worst-period gates |
| `phase11/CompressionVolumeBreakout|long|60m` | CompressionVolumeBreakout | Phase 11.1 | `REJECTED` | `REJECTED` | true | side/horizon row failed net and worst-period gates |
| `phase11/CompressionVolumeBreakout|short|15m` | CompressionVolumeBreakout | Phase 11.1 | `REJECTED` | `REJECTED` | true | side/horizon row failed net and worst-period gates |
| `phase11/CompressionVolumeBreakout|short|240m` | CompressionVolumeBreakout | Phase 11.1 | `REJECTED` | `REJECTED` | true | side/horizon row failed net and worst-period gates |
| `phase11/CompressionVolumeBreakout|short|60m` | CompressionVolumeBreakout | Phase 11.1 | `REJECTED` | `REJECTED` | true | side/horizon row failed net and worst-period gates |
| `phase11/RegimeTrendPullbackContinuation|long|120m` | RegimeTrendPullbackContinuation | Phase 11.2 | `REJECTED` | `REJECTED` | false | side/horizon row failed; exact implementation is not committed/reproducible |
| `phase11/RegimeTrendPullbackContinuation|long|15m` | RegimeTrendPullbackContinuation | Phase 11.2 | `REJECTED` | `REJECTED` | false | side/horizon row failed; exact implementation is not committed/reproducible |
| `phase11/RegimeTrendPullbackContinuation|long|240m` | RegimeTrendPullbackContinuation | Phase 11.2 | `REJECTED` | `REJECTED` | false | side/horizon row failed; exact implementation is not committed/reproducible |
| `phase11/RegimeTrendPullbackContinuation|long|30m` | RegimeTrendPullbackContinuation | Phase 11.2 | `REJECTED` | `REJECTED` | false | side/horizon row failed; exact implementation is not committed/reproducible |
| `phase11/RegimeTrendPullbackContinuation|long|60m` | RegimeTrendPullbackContinuation | Phase 11.2 | `REJECTED` | `REJECTED` | false | side/horizon row failed; exact implementation is not committed/reproducible |
| `phase11/RegimeTrendPullbackContinuation|short|120m` | RegimeTrendPullbackContinuation | Phase 11.2 | `REJECTED` | `REJECTED` | false | side/horizon row failed; exact implementation is not committed/reproducible |
| `phase11/RegimeTrendPullbackContinuation|short|15m` | RegimeTrendPullbackContinuation | Phase 11.2 | `REJECTED` | `REJECTED` | false | side/horizon row failed; exact implementation is not committed/reproducible |
| `phase11/RegimeTrendPullbackContinuation|short|240m` | RegimeTrendPullbackContinuation | Phase 11.2 | `REJECTED` | `REJECTED` | false | side/horizon row failed; exact implementation is not committed/reproducible |
| `phase11/RegimeTrendPullbackContinuation|short|30m` | RegimeTrendPullbackContinuation | Phase 11.2 | `REJECTED` | `REJECTED` | false | side/horizon row failed; exact implementation is not committed/reproducible |
| `phase11/RegimeTrendPullbackContinuation|short|60m` | RegimeTrendPullbackContinuation | Phase 11.2 | `REJECTED` | `REJECTED` | false | side/horizon row failed; exact implementation is not committed/reproducible |
| `phase12/DowntrendMidVolReliefLong240m` | DowntrendMidVolRelief | Phase 12.4 | `NEAR_MISS` | `NEAR_MISS` | true | failed fixed worst-quarter gate and cluster-independence evidence |
| `phase13/ContextFreeMomentumBreakoutProbe` | ContextFreeMomentumBreakoutProbe | Phase 13.0 | `INFRASTRUCTURE_PROBE` | `INSUFFICIENT_SAMPLE` | true | infrastructure proof only; one symbol-month; negative net result; leave-one-out segments unavailable |
| `price-alpha/BetaAgrees/long` | BetaAgrees | Phase 10 price-alpha baseline | `MISSING_EVIDENCE` | `PIT_EVIDENCE_MISSING` | true | no complete exact-candidate qualification evidence |
| `price-alpha/BetaAgrees/short` | BetaAgrees | Phase 10 price-alpha baseline | `MISSING_EVIDENCE` | `PIT_EVIDENCE_MISSING` | true | no complete exact-candidate qualification evidence |
| `price-alpha/BetaDiverges/long` | BetaDiverges | Phase 10 price-alpha baseline | `MISSING_EVIDENCE` | `PIT_EVIDENCE_MISSING` | true | no complete exact-candidate qualification evidence |
| `price-alpha/BetaDiverges/short` | BetaDiverges | Phase 10 price-alpha baseline | `MISSING_EVIDENCE` | `PIT_EVIDENCE_MISSING` | true | no complete exact-candidate qualification evidence |
| `price-alpha/CompressionBreakout/long` | CompressionBreakout | Phase 10.4 | `NEAR_MISS` | `NEAR_MISS` | true | bracket/concentration failure; fragile is not paper eligible |
| `price-alpha/CompressionBreakout/short` | CompressionBreakout | Phase 10.4 | `NEAR_MISS` | `NEAR_MISS` | true | bracket/concentration failure; fragile is not paper eligible |
| `price-alpha/ShockFade/long` | ShockFade | Phase 10.4 | `REJECTED` | `REJECTED` | true | out-of-sample and/or concentration/sample gates failed |
| `price-alpha/ShockFade/short` | ShockFade | Phase 10 price-alpha baseline | `MISSING_EVIDENCE` | `PIT_EVIDENCE_MISSING` | true | no complete exact-candidate qualification evidence |
| `price-alpha/TrendContinuation/long` | TrendContinuation | Phase 10 price-alpha baseline | `MISSING_EVIDENCE` | `PIT_EVIDENCE_MISSING` | true | no complete exact-candidate qualification evidence |
| `price-alpha/TrendContinuation/short` | TrendContinuation | Phase 10 price-alpha baseline | `MISSING_EVIDENCE` | `PIT_EVIDENCE_MISSING` | true | no complete exact-candidate qualification evidence |
| `price-alpha/VolumeMomentum/long` | VolumeMomentum | Phase 10 price-alpha baseline | `MISSING_EVIDENCE` | `PIT_EVIDENCE_MISSING` | true | no complete exact-candidate qualification evidence |
| `price-alpha/VolumeMomentum/short` | VolumeMomentum | Phase 10 price-alpha baseline | `MISSING_EVIDENCE` | `PIT_EVIDENCE_MISSING` | true | no complete exact-candidate qualification evidence |

Every registered Engine strategy name is present. Failed, fragile, near-miss, infrastructure-only, and missing-evidence hypotheses are retained rather than omitted.
