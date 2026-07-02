# Phase 10 Final Closeout

## Conclusion
Current retained funding-event candidate set is not promotable.

- final_label: `all_candidates_rejected`
- full_universe_ready: `true`
- raw_required: `false`
- candidate_count: `36`
- label counts: `SHADOW_CANDIDATE=0, RESEARCH_LEAD=0, FRAGILE_RESEARCH_LEAD=0, REJECTED=36`
- strongest rejected candidate: `NegativeFundingLong|long|240m`

## Closeout Findings
- Full coverage made the earlier XRP-only fragile leads disappear.
- `NegativeFundingLong|long|240m` remains the strongest but is still rejected.
- No candidate should touch ak-trader.
- Phase 10 infrastructure is successful because it prevented single-symbol artifacts from being promoted.
- Old June 5 baseline reports are stale for net/slippage-sensitive comparisons after 10.9A/10.9B.
- Earlier Phase 10.8C state was XRPUSDT-only, but final 10.9C state is full coverage. The final coverage reports list all 8 symbols because they reflect the recovered end state, not the original partial before-state.

## Why All Candidates Remain Rejected
- Every retained candidate has negative 5 bps baseline expectancy and PF below 1.
- Every retained candidate fails 7.5 bps cost stress.
- Every retained candidate fails delay_1 expectancy.
- Every retained candidate fails leave-one-symbol-out, leave-one-month-out, and leave-one-quarter-out robustness.
- Many candidates also show symbol, month, quarter, or bucket concentration; the strongest candidate depends heavily on XRPUSDT-positive contribution.

## Recommended Next Phase
Phase 10.11 - New Candidate Family Design Plan. Do not implement it in Phase 10.10.

### funding + regime + volatility compression interaction
- hypothesis: Funding extremes may only have edge when paired with regime state and pre-event volatility compression, not as standalone funding events.
- data needed: funding event summaries, regime labels, realized volatility windows, compression/expansion features
- leakage risks: using post-entry volatility labels, selecting regimes after seeing returns
- expected robustness gates: positive net expectancy at 5/7.5/10 bps, delay_1 remains positive, leave-one-symbol/month/quarter passes, bounded bucket concentration
- why it might be better: It targets conditional structure instead of averaging all funding events into a broad negative-expectancy set.

### funding reversal after extreme persistence
- hypothesis: Reversal edge may appear after funding stays extreme for multiple intervals and then decays, rather than immediately after one extreme print.
- data needed: funding history by symbol, persistence counters, decay/normalization markers, forward returns
- leakage risks: requiring future normalization before entry, misaligning funding timestamps with tradable bars
- expected robustness gates: walk-forward parameter lock, minimum cluster count, positive stressed PF, delay and leave-out survival
- why it might be better: It tests a more specific behavior than single-event mean reversion, which failed full-universe robustness.

### cross-symbol relative strength / weakness confirmation
- hypothesis: Funding signals may need confirmation from relative performance versus the retained universe before entry.
- data needed: cross-sectional returns, relative strength ranks, funding buckets, liquidity-normalized symbol panels
- leakage risks: ranking with bars that close after entry, survivorship or symbol availability bias
- expected robustness gates: symbol-balanced contribution, leave-one-symbol pass, quarter pass, net positive under 10 bps
- why it might be better: It directly addresses the XRP concentration artifact exposed by Phase 10.9C.

### BTC/ETH context-gated funding candidates
- hypothesis: Alt funding events may only be tradable when BTC/ETH trend and volatility context align with the candidate side.
- data needed: BTC/ETH context bars, market beta buckets, funding events, symbol forward returns
- leakage risks: using incomplete BTC/ETH candles, context labels computed with future returns
- expected robustness gates: context gate precomputed at entry time, cost stress pass, delay pass, leave-one-quarter pass
- why it might be better: It conditions entries on market state rather than assuming each symbol funding event is independently predictive.

### volume/liquidity-filtered funding candidates
- hypothesis: Net edge may survive only where funding dislocation occurs with sufficient tradable liquidity and clean volume participation.
- data needed: volume windows, spread/slippage proxies, funding events, symbol liquidity ranks
- leakage risks: estimating liquidity with future bars, excluding bad fills after observing outcomes
- expected robustness gates: pre-entry liquidity filters, 10 bps stress pass, minimum clusters per symbol, bounded month/quarter concentration
- why it might be better: It attacks the net/slippage sensitivity that kept every retained candidate below promotion quality.

## Boundary Confirmation
- no promotion to ak-trader
- no live trading/order/exchange key/mainnet code added
- no remote raw gzip cleanup performed
- no threshold tuning to force a pass
- no new candidate family implemented in this phase

## Heavy Files
Heavy local run artifacts remain and were not deleted: `24` files.
- `runs/reports/chunks/DOGEUSDT/2024-01-funding-events.jsonl.gz`
- `runs/reports/chunks/DOGEUSDT/2024-02-funding-events.jsonl.gz`
- `runs/reports/chunks/DOGEUSDT/2024-03-funding-events.jsonl.gz`
- `runs/reports/chunks/DOGEUSDT/2024-04-funding-events.jsonl.gz`
- `runs/reports/chunks/DOGEUSDT/2024-05-funding-events.jsonl.gz`
- `runs/reports/chunks/DOGEUSDT/2024-06-funding-events.jsonl.gz`
- `runs/reports/chunks/DOGEUSDT/2024-07-funding-events.jsonl.gz`
- `runs/reports/chunks/DOGEUSDT/2024-08-funding-events.jsonl.gz`
- `runs/reports/chunks/DOGEUSDT/2024-09-funding-events.jsonl.gz`
- `runs/reports/chunks/DOGEUSDT/2024-10-funding-events.jsonl.gz`
- `runs/reports/chunks/DOGEUSDT/2024-11-funding-events.jsonl.gz`
- `runs/reports/chunks/DOGEUSDT/2024-12-funding-events.jsonl.gz`
- `runs/reports/chunks/DOGEUSDT/2025-01-funding-events.jsonl.gz`
- `runs/reports/chunks/DOGEUSDT/2025-02-funding-events.jsonl.gz`
- `runs/reports/chunks/DOGEUSDT/2025-03-funding-events.jsonl.gz`
- `runs/reports/chunks/DOGEUSDT/2025-04-funding-events.jsonl.gz`
- `runs/reports/chunks/DOGEUSDT/2025-05-funding-events.jsonl.gz`
- `runs/reports/chunks/DOGEUSDT/2025-06-funding-events.jsonl.gz`
- `runs/reports/chunks/DOGEUSDT/2025-07-funding-events.jsonl.gz`
- `runs/reports/chunks/DOGEUSDT/2025-08-funding-events.jsonl.gz`
- `runs/reports/chunks/DOGEUSDT/2025-09-funding-events.jsonl.gz`
- `runs/reports/chunks/DOGEUSDT/2025-10-funding-events.jsonl.gz`
- `runs/reports/chunks/DOGEUSDT/2025-11-funding-events.jsonl.gz`
- `runs/reports/chunks/DOGEUSDT/2025-12-funding-events.jsonl.gz`
