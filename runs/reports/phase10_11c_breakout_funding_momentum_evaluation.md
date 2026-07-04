# Phase 10.11C BreakoutFundingMomentum Evaluation

- Coverage: 192/192 symbol-months
- Missing symbol-months: none
- Raw required: false
- Chromebook raw-free status: retained compact summaries only; 10.11C chunk directory has zero JSONL event files
- Strongest candidate: `BreakoutFundingLong|long|240m`
- RESEARCH_LEAD: false
- SHADOW_CANDIDATE: false
- ak-trader integration: untouched; no promotion

| Candidate | Label | Events | Clusters | Expectancy after 5 bps | PF after 5 bps |
|---|---:|---:|---:|---:|---:|
| `BreakoutFundingLong|long|240m` | REJECTED | 197685 | 45740 | -2.943941 | 0.953242 |
| `BreakoutFundingShort|short|5m` | REJECTED | 307793 | 68905 | -5.932819 | 0.580236 |

Both BreakoutFundingMomentum candidates remain rejected by the compact robustness gate. The long candidate is strongest but still has negative expectancy and PF below 1 after costs. The short candidate is weaker.

Event counts by symbol/month/family/side/horizon are included in `phase10_11c_breakout_funding_momentum_event_counts.json` and embedded in the evaluation JSON. Rows: 2298.

Stress, delay, leave-one-symbol, leave-one-month, and leave-one-quarter details are embedded in the JSON under each candidate report.
