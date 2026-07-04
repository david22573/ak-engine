# Phase 10.11D VolumeImbalanceFundingReversionProxy Evaluation

- Result: `REJECTED`
- Research leads: `0`
- Shadow candidates: `0`
- Coverage: `192/192` retained summaries, `full_universe_ready=true`
- Raw required: `false`
- ak-trader touched: `false`
- Limitation: uses `TakerBuyRatio` fallback only; full taker-buy-sell-volume join not implemented.

## Best Long

- Candidate: `VolumeImbalanceFundingReversionProxyLong|long|240m`
- Events: `1,110,149`
- Clusters: `77,502`
- Expectancy after 5 bps: `-2.511184`
- PF after 5 bps: `0.958095`
- Label: `REJECTED`

## Best Short

- Candidate: `VolumeImbalanceFundingReversionProxyShort|short|5m`
- Events: `1,645,518`
- Clusters: `118,173`
- Expectancy after 5 bps: `-4.963354`
- PF after 5 bps: `0.550962`
- Label: `REJECTED`

No promotion. No ak-trader changes.
