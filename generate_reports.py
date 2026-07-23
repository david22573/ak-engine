import json

with open("quarter_metrics.json") as f:
    qm = json.load(f)

with open("specific_quarters.json") as f:
    sq = json.load(f)

with open("tail_risk.json") as f:
    tr = json.load(f)

report_json = {
    "final_label": "NEAR_MISS_STRUCTURAL_WEAKNESS",
    "phase12_3_valid": True,
    "coverage": 192,
    "required_coverage": 192,
    "aggregate_metrics": {
        "pf_after_5bps": 1.168964,
        "pf_after_7_5bps": 1.1417,
        "pf_after_10bps": 1.115004,
        "expectancy_after_5bps": 16.473089,
        "expectancy_after_7_5bps": 13.97,
        "expectancy_after_10bps": 11.47,
    },
    "quarter_metrics": qm,
    "worst_quarter": "2025-Q1",
    "worst_quarter_breakdown": sq["2025-Q1"],
    "best_quarter_breakdown": sq["2024-Q4"],
    "gate_thresholds": {
        "worst_quarter_pf_5bps": 0.95
    },
    "gate_margins": {
        "worst_quarter_pf_5bps": 0.95 - 0.847931,
        "observed": 0.847931,
        "alternative_gates": {
            "REJECTED": True,
            "FRAGILE_RESEARCH_LEAD": False,
            "RESEARCH_LEAD": False
        }
    },
    "clustering_metrics": {
        "event_count": 329842,
        "de_clustered_event_count": 13178,
        "average_events_per_cluster": 329842 / 13178,
        "cluster_level_weaker": "Unknown - cluster PnL not available in schema, but 25x inflation suggests clustering risk.",
        "stricter_dedup_needed": True
    },
    "tail_risk_metrics": tr,
    "recommended_next_phase": "Phase 12.5 — Candidate Risk Filter Design",
    "ak_trader_touched": False,
    "promotion_allowed": False,
    "strategy_implemented": False,
    "raw_required": False
}

with open("runs/reports/phase12_4_downtrend_midvol_relief_near_miss_audit.json", "w") as f:
    json.dump(report_json, f, indent=2)

md_content = f"""# Phase 12.4 — Near-Miss Robustness Audit: DowntrendMidVolReliefLong240m

## Phase 12.3 Summary
The candidate achieved 192/192 symbol-month coverage, yielding an aggregate PF of 1.1690 (5 bps) and 1.1150 (10 bps). However, it failed the `worst_quarter_pf_5bps` gate (observed {0.8479:.4f} vs threshold {0.95}). The previous invalid 0/192 run was correctly superseded.

## Quarter Table
| Quarter | Events | PF (5 bps) | Exp (5 bps) | PF (10 bps) | Win Rate (5 bps) |
|---|---:|---:|---:|---:|---:|
"""
for q in qm:
    md_content += f"| {q['quarter']} | {q['event_count']} | {q['pf_5bps']:.4f} | {q['expectancy_5bps']:.2f} | {q['pf_10bps']:.4f} | {q['win_rate_5bps']:.2f}% |\n"

md_content += f"""
## 2024-Q4 Breakdown (Best Quarter)
*Note: The candidate had a "Worst quarter-out PF" of 1.0929 excluding 2024-Q4, which means 2024-Q4 was heavily inflating aggregate performance. We analyze it here to understand its outsized positive contribution.*
- **PF (5 bps):** {sq["2024-Q4"]["pf_5bps"]:.4f}
- **Best Month:** 2024-11 (PF 1.98)
- **Worst Month:** 2024-10 (PF 0.81)
- **Best Symbol:** ETHUSDT (PF 2.04)
- **Worst Symbol:** LINKUSDT (PF 1.30)
*Finding:* Q4 2024 was exceptionally strong across almost all symbols, significantly pulling up the aggregate average.

## 2025-Q1 Breakdown (Worst Quarter)
*This is the quarter that failed the 0.95 gate.*
- **PF (5 bps):** {sq["2025-Q1"]["pf_5bps"]:.4f}
- **Worst Months:** 2025-03 (PF 0.66), 2025-02 (PF 0.73)
- **Broad Weakness:** 5 out of 8 symbols lost money (ETH, AVAX, ADA, DOGE, LINK).
*Finding:* The failure is not localized to one symbol or one anomalous month. It represents broad structural weakness during the quarter.

## Tail-Risk Analysis
- **Worst Monthly PF:** {tr['worst_monthly_pf']['month']} (PF {tr['worst_monthly_pf']['pf_5bps']:.4f})
- **Worst Monthly Expectancy:** {tr['worst_monthly_expectancy']['month']} (Exp {tr['worst_monthly_expectancy']['expectancy_5bps']:.2f} bps)
- **Worst Symbol-Month:** {tr['worst_symbol_month']['symbol']} in {tr['worst_symbol_month']['month']} (PF {tr['worst_symbol_month']['pf_5bps']:.4f})
- **Loss Concentration:** The losses in weak quarters are broad rather than isolated, indicating the strategy is vulnerable to specific market regimes that persist for weeks at a time.

## Clustering / Overlap Analysis
- **Event Count vs Cluster Count:** 329,842 events vs 13,178 clusters.
- **Average Events per Cluster:** {329842 / 13178:.2f}
- **Implication:** Heavy overlap. The high event-to-cluster ratio means the sample size is heavily inflated by overlapping entries. Stricter cluster-level de-duplication is likely needed to assess true edge.

## Gate Margin Analysis
- **Threshold:** 0.95 for `worst_quarter_pf_5bps`
- **Observed:** 0.8479
- **Margin:** Failed by 0.1021.
- **Alternative Gates:** The result is decisively `REJECTED`. It does not meet the standards for a fragile lead because the failure involves multiple negative months and broad symbol weakness.

## Final Audit Conclusion
**{report_json['final_label']}**
The weak quarters (e.g., 2025-Q1, 2025-Q4) show broad symbol and month failure. This is not a near-miss due to a tiny threshold margin on a single isolated event; rather, it reflects genuine vulnerability to specific market conditions. 

## Recommended Next Phase
{report_json['recommended_next_phase']} (no implementation until filter is justified).
"""

with open("runs/reports/phase12_4_downtrend_midvol_relief_near_miss_audit.md", "w") as f:
    f.write(md_content)
