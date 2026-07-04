# Phase 10.12 - Second-Generation Funding Candidate Rejection Audit

## Verdict

Funding-event research should close with the current data set. Phase 11 should pivot away from funding events unless the next phase first adds genuinely new positioning or flow data: true open interest, liquidations, positioning, or a real taker buy/sell volume join.

No candidate is promotable. No `ak-trader` change is justified. No shadow plan is justified.

## Evidence Board

| Phase | Family | Coverage | Best candidate | Events | Clusters | Expectancy 5 bps | PF 5 bps | Result |
| --- | --- | ---: | --- | ---: | ---: | ---: | ---: | --- |
| 10.11B | ConfirmedFundingExtreme | 192/192 | `ConfirmedNegativeFundingLong|long|240m` | 1,353,932 | 51,997 | -2.014372 | 0.966931 | REJECTED |
| 10.11C | BreakoutFundingMomentum | 192/192 | `BreakoutFundingLong|long|240m` | 197,685 | 45,740 | -2.943941 | 0.953242 | REJECTED |
| 10.11D | VolumeImbalanceFundingReversionProxy | 192/192 | `VolumeImbalanceFundingReversionProxyLong|long|240m` | 1,110,149 | 77,502 | -2.511184 | 0.958095 | REJECTED |

The short-side variants were weaker:

| Phase | Candidate | Events | Clusters | Expectancy 5 bps | PF 5 bps | Result |
| --- | --- | ---: | ---: | ---: | ---: | --- |
| 10.11B | `ConfirmedPositiveFundingShort|short|5m` | 1,589,488 | 60,407 | -5.164622 | 0.547049 | REJECTED |
| 10.11D | `VolumeImbalanceFundingReversionProxyShort|short|5m` | 1,645,518 | 118,173 | -4.963354 | 0.550962 | REJECTED |

## Why They Failed

All three second-generation variants failed the same core tests:

- Baseline expectancy stayed below zero.
- Baseline profit factor stayed below 1.0.
- 7.5 bps cost stress failed.
- One-candle entry delay remained negative.
- Leave-one-month, leave-one-quarter, and leave-one-symbol robustness failed for the strongest 10.11B and 10.11D variants.

The repeated shape matters more than any single number. Adding confirmation, breakout/momentum context, and a taker-ratio volume proxy did not flip the family into positive expectancy.

## Data Limitation

10.11D tested a proxy only:

```text
uses TakerBuyRatio fallback only; full taker-buy-sell-volume join not implemented
```

That means 10.11D does not reject a true taker buy/sell imbalance hypothesis. It rejects the currently available fallback proxy.

`SqueezeFundingUnwind` should remain blocked until true OI/liquidation/positioning data exists. Implementing another funding-event variant now would mostly retest the same weak signal surface.

## Audit Answers

**Is funding-event alpha dead with current data?**  
Yes for the tested feature set. Plain, confirmed, breakout/momentum, and taker-ratio proxy funding variants have all failed full-coverage gates.

**Would OI/liquidations/taker buy-sell volume plausibly change the answer?**  
Possibly, but only because they are real new data. The current candle/funding/regime/TakerBuyRatio feature set is not enough.

**Should Phase 11 pivot away from funding events?**  
Yes, unless Phase 11 begins by adding new positioning or flow data sources.

## Closeout State

- Research leads: `0`
- Shadow candidates: `0`
- `ak-trader` changes: `0`
- Chromebook raw funding-event files: `0`
- Promotion: `false`
- Shadow planning: `false`
