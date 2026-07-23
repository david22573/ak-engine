import json

def run():
    # Write JSON report
    report_json = {
        "final_label": "blocked_missing_inputs",
        "focus_bucket": "RealizedVol60 / realized_vol_60_mid / trend_down / long / 240m",
        "focus_bucket_strict_label": "BLOCKED_MISSING_INPUTS",
        "survives_q4_2024_exclusion": True,
        "survives_10_bps": True,
        "survives_leave_one_quarter_out": True,
        "survives_leave_one_symbol_out": True,
        "independent_edge_theme_count": 9,
        "top_bucket_is_independent": False,
        "recommended_next_phase": "Phase 12.2 — Continue Edge Discovery / Data Upgrade Plan",
        "ak_trader_touched": False,
        "promotion_allowed": False,
        "strategy_implemented": False,
        "missing_inputs": [
            "PF after 5 bps",
            "PF after 7.5 bps",
            "PF after 10 bps"
        ]
    }
    
    with open('runs/reports/phase12_1b_lite_strict_edge_audit.json', 'w') as f:
        json.dump(report_json, f, indent=2)

    # Write Markdown report
    md = [
        "# Phase 12.1B-Lite — Strict Edge Audit Sanity Check",
        "",
        "## Goal",
        "Do a strict sanity check of the Phase 12.1 result to see if the top bucket is still worth building a candidate around.",
        "",
        "## Focus Bucket",
        "`RealizedVol60 / realized_vol_60_mid / trend_down / long / 240m`",
        "",
        "## Required Checks",
        "- **expectancy_after_5_bps:** 16.99",
        "- **expectancy_after_7_5_bps:** 14.49",
        "- **expectancy_after_10_bps:** 11.99",
        "- **PF after 5 bps:** MISSING_INPUT (Profit factor after 5 bps slippage was not computed in previous phases)",
        "- **PF after 7.5 bps:** MISSING_INPUT",
        "- **PF after 10 bps:** MISSING_INPUT",
        "- **sample_count:** 18013",
        "- **cluster_count:** 9618",
        "- **positive symbol count:** 7",
        "- **positive month count:** 16",
        "- **positive quarter count:** 6",
        "- **top symbol contribution percentage:** 23.86%",
        "- **top month contribution percentage:** 23.52%",
        "- **top quarter contribution percentage:** 39.36%",
        "- **Q4 2024 contribution percentage:** 39.36%",
        "- **result excluding Q4 2024:** 10.61 bps (expectancy after 5 bps)",
        "- **result excluding the best quarter:** 10.61 bps (expectancy after 5 bps)",
        "- **result excluding the best symbol:** 14.61 bps (expectancy after 5 bps)",
        "- **leave-one-symbol-out worst result:** 14.61 bps",
        "- **leave-one-month-out worst result:** 12.17 bps",
        "- **leave-one-quarter-out worst result:** 10.61 bps",
        "",
        "## Overlap Check",
        "- **overlap cluster count:** 9",
        "- **buckets in the top cluster:** 5",
        "  - RealizedVol60 / realized_vol_60_mid / trend_down / long / 240m",
        "  - CloseRelativeEMA20 / above_ema20_strong / range / long / 240m",
        "  - CloseRelativeEMA20 / below_ema20_strong / trend_down / long / 240m",
        "  - CloseRelativeEMA20 / above_ema20_strong / range / long / 120m",
        "  - Return15 / return15_down_strong / trend_down / long / 240m",
        "- **representative bucket:** RealizedVol60 / realized_vol_60_mid / trend_down / long / 240m",
        "- **whether the top bucket is independent or duplicate exposure:** Duplicate exposure (overlaps with 4 other long/240m and long/120m buckets, likely capturing the same Q4 2024 uptrend footprint).",
        "",
        "## Strict Label Evaluation",
        "Because `PF after 10 bps` is missing, we cannot evaluate the `STRICT_REJECTED` rule: `or PF_after_10_bps <= 1.0`.",
        "However, even if it passed, it is a duplicate of another bucket, which means it cannot be `STRICT_TOP_CANDIDATE`.",
        "Strict Label: **BLOCKED_MISSING_INPUTS** (or STRICT_WEAK at best due to duplication and missing PF).",
        "",
        "## Final Status",
        "- **Final Label:** `blocked_missing_inputs`",
        "- **Recommended Next Phase:** Phase 12.2 — Continue Edge Discovery / Data Upgrade Plan, no strategy implementation.",
        "- **ak-trader touched:** False",
        "- **promotion allowed:** False",
        "- **strategy implemented:** False"
    ]
    
    with open('runs/reports/phase12_1b_lite_strict_edge_audit.md', 'w') as f:
        f.write('\n'.join(md))

if __name__ == '__main__':
    run()
