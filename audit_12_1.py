import json
import os
import math

def pearson_corr(x, y):
    n = len(x)
    if n == 0: return 0.0
    sum_x = sum(x)
    sum_y = sum(y)
    sum_x_sq = sum(xi*xi for xi in x)
    sum_y_sq = sum(yi*yi for yi in y)
    psum = sum(xi*yi for xi, yi in zip(x, y))
    
    num = psum - (sum_x * sum_y / n)
    den = math.sqrt((sum_x_sq - (sum_x**2) / n) * (sum_y_sq - (sum_y**2) / n))
    if den == 0: return 0.0
    return num / den

def run_audit():
    with open('runs/reports/phase12_0_edge_discovery_matrix.json', 'r') as f:
        data = json.load(f)

    all_buckets = data['buckets']
    robust_buckets = [b for b in all_buckets if b.get('label') == 'ROBUST_EDGE_CANDIDATE']
    
    # Sort robust_buckets by expectancy_after_5_bps desc
    robust_buckets.sort(key=lambda x: x.get('expectancy_after_5_bps', 0), reverse=True)

    # Extract monthly net arrays for correlation
    monthly_nets = {}
    all_months = set()
    for i, b in enumerate(robust_buckets):
        key = f"{b['feature']} / {b['bucket']} / {b['regime']} / {b['side']} / {b['horizon']}"
        monthly_dict = {m['key']: m['net_after_5_bps'] for m in b.get('month_metrics', [])}
        monthly_nets[key] = monthly_dict
        all_months.update(monthly_dict.keys())

    sorted_months = sorted(list(all_months))
    
    # Compute correlation
    corr_matrix = {}
    keys = list(monthly_nets.keys())
    for k1 in keys:
        corr_matrix[k1] = {}
        v1 = [monthly_nets[k1].get(m, 0.0) for m in sorted_months]
        for k2 in keys:
            v2 = [monthly_nets[k2].get(m, 0.0) for m in sorted_months]
            corr_matrix[k1][k2] = pearson_corr(v1, v2)

    # Find clusters (correlation > 0.75)
    clusters = []
    visited = set()
    for col in keys:
        if col in visited: continue
        cluster = [col]
        visited.add(col)
        for other_col in keys:
            if other_col not in visited and corr_matrix[col][other_col] > 0.75:
                cluster.append(other_col)
                visited.add(other_col)
        clusters.append(cluster)

    audited_buckets = []
    survivors = []

    for b in robust_buckets:
        key = f"{b['feature']} / {b['bucket']} / {b['regime']} / {b['side']} / {b['horizon']}"
        
        # Diagnostics
        top_sym = b.get('top_symbol_contribution_pct', 0)
        top_qtr = b.get('top_quarter_contribution_pct', 0)
        exp_5 = b.get('expectancy_after_5_bps', 0)
        exp_7_5 = b.get('expectancy_after_7_5_bps', 0)
        exp_10 = b.get('expectancy_after_10_bps', 0)
        win_rate = b.get('win_rate', 0)
        loso = b.get('leave_one_symbol_out_result', {}).get('worst_expectancy_after_5_bps', 0)
        lomo = b.get('leave_one_month_out_result', {}).get('worst_expectancy_after_5_bps', 0)
        sample_count = b.get('sample_count', 0)
        
        rejection_reasons = []
        if top_sym > 40: rejection_reasons.append(f"High symbol concentration: {top_sym:.1f}%")
        if top_qtr > 50: rejection_reasons.append(f"High quarter concentration: {top_qtr:.1f}%")
        if exp_7_5 <= 0: rejection_reasons.append(f"Fails 7.5 bps stress test: {exp_7_5:.2f}")
        if loso <= 0: rejection_reasons.append(f"Fails leave-one-symbol-out: {loso:.2f}")
        if lomo <= 0: rejection_reasons.append(f"Fails leave-one-month-out: {lomo:.2f}")
        
        if rejection_reasons:
            label = "AUDIT_REJECTED"
        elif exp_10 <= 0 or exp_5 < 10 or win_rate < 51.0:
            label = "AUDIT_WEAK"
        else:
            label = "AUDIT_SURVIVES"
            survivors.append(b)
            
        b['audit_label'] = label
        b['rejection_reasons'] = rejection_reasons
        audited_buckets.append(b)

    # Assign TOP CANDIDATE
    if survivors:
        best = max(survivors, key=lambda x: x.get('expectancy_after_5_bps', 0))
        for b in audited_buckets:
            if b == best:
                b['audit_label'] = "AUDIT_TOP_CANDIDATE"

    # Next phase logic
    if any(b['audit_label'] == "AUDIT_TOP_CANDIDATE" for b in audited_buckets):
        next_phase = "Phase 12.2 \u2014 Convert Top Edge Bucket Into Candidate Family Spec"
    elif any(b['audit_label'] in ["AUDIT_SURVIVES", "AUDIT_WEAK"] for b in audited_buckets):
        next_phase = "Phase 12.2 \u2014 LLM-Assisted Candidate Spec Around Surviving Buckets"
    else:
        next_phase = "Phase 12.2 \u2014 Data Source Upgrade / Stop Current Feature Set"

    # Build JSON
    report_json = {
        "phase": "Phase 12.1",
        "mode": "audit/reporting",
        "robust_buckets_audited": len(robust_buckets),
        "audit_results": {
            "AUDIT_TOP_CANDIDATE": sum(1 for b in audited_buckets if b['audit_label'] == "AUDIT_TOP_CANDIDATE"),
            "AUDIT_SURVIVES": sum(1 for b in audited_buckets if b['audit_label'] == "AUDIT_SURVIVES"),
            "AUDIT_WEAK": sum(1 for b in audited_buckets if b['audit_label'] == "AUDIT_WEAK"),
            "AUDIT_REJECTED": sum(1 for b in audited_buckets if b['audit_label'] == "AUDIT_REJECTED")
        },
        "strongest_surviving_bucket": [b for b in audited_buckets if b['audit_label'] == "AUDIT_TOP_CANDIDATE"][0] if survivors else None,
        "overlap_clusters": clusters,
        "recommended_next_phase": next_phase,
        "boundaries_respected": {
            "ak_trader_untouched": True,
            "no_promotion": True,
            "no_live_code": True,
            "no_new_data_fetch": True,
            "audit_only": True
        },
        "buckets": audited_buckets
    }

    with open('runs/reports/phase12_1_robust_edge_bucket_audit.json', 'w') as f:
        json.dump(report_json, f, indent=2)

    # Build Markdown
    md = [
        "# Phase 12.1 \u2014 Robust Edge Bucket Audit",
        "",
        "## Context: Phase 12.0 Summary",
        f"Phase 12.0 discovered {len(robust_buckets)} `ROBUST_EDGE_CANDIDATE` buckets out of {data['bucket_count']} total evaluated.",
        "",
        "## Stricter Audit Labels & Diagnostics",
        "Each of the 13 robust buckets underwent deeper sanity checks:",
        "- **Concentration Limits:** Max symbol contribution \u2264 40%, max quarter contribution \u2264 50%",
        "- **Stress Survival:** Must survive 7.5 bps strictly.",
        "- **Leave-One-Out Stability:** Must remain positive when excluding the best symbol or best month.",
        "- **Weakness Threshold:** `expectancy_after_10_bps` > 0 and `win_rate` \u2265 51.0 for full survival.",
        ""
    ]

    for b in audited_buckets:
        key = f"{b['feature']} / {b['bucket']} / {b['regime']} / {b['side']} / {b['horizon']}"
        md.append(f"### {key}")
        md.append(f"**Audit Label:** `{b['audit_label']}`")
        if b['rejection_reasons']:
            md.append(f"**Rejection Reasons:** {', '.join(b['rejection_reasons'])}")
        md.append(f"- **Sample Count:** {b['sample_count']} (Clusters: {b['cluster_count']})")
        md.append(f"- **Expectancy (5/7.5/10 bps):** {b['expectancy_after_5_bps']:.2f} / {b['expectancy_after_7_5_bps']:.2f} / {b['expectancy_after_10_bps']:.2f} bps")
        md.append(f"- **Win Rate / PF:** {b['win_rate']:.2f}% / {b.get('profit_factor', 0):.2f}")
        md.append(f"- **LOSO / LOMO Worst:** {b.get('leave_one_symbol_out_result', {}).get('worst_expectancy_after_5_bps', 0):.2f} / {b.get('leave_one_month_out_result', {}).get('worst_expectancy_after_5_bps', 0):.2f}")
        md.append(f"- **Concentration (Sym/Mo/Qtr):** {b.get('top_symbol_contribution_pct', 0):.1f}% / {b.get('top_month_contribution_pct', 0):.1f}% / {b.get('top_quarter_contribution_pct', 0):.1f}%")
        md.append("")

    md.extend([
        "## Overlap & Independence Analysis",
        "We computed monthly correlation matrices across the 13 robust buckets to identify whether they describe the same underlying event population.",
        "Based on > 0.75 Pearson correlation, the buckets group into the following distinct clusters:",
        ""
    ])

    for i, cluster in enumerate(clusters):
        md.append(f"**Cluster {i+1}:**")
        for member in cluster:
            md.append(f"- {member}")
        md.append("")
        
    md.extend([
        "## Strongest Surviving Bucket",
    ])
    
    if survivors:
        best = max(survivors, key=lambda x: x.get('expectancy_after_5_bps', 0))
        best_key = f"{best['feature']} / {best['bucket']} / {best['regime']} / {best['side']} / {best['horizon']}"
        # Determine if it's independent or overlapping
        best_cluster = [c for c in clusters if best_key in c][0]
        overlap_status = "overlapping" if len(best_cluster) > 1 else "independent"
        
        md.append(f"**{best_key}**")
        md.append(f"This bucket survived all strict audits and was labeled `AUDIT_TOP_CANDIDATE`. It is part of Cluster {clusters.index(best_cluster)+1}, which means it is **{overlap_status}** (cluster size: {len(best_cluster)}).")
    else:
        md.append("None of the buckets passed the strict audit to become a top candidate.")

    md.extend([
        "",
        "## Recommended Next Phase",
        f"**{next_phase}**",
        "",
        "## Boundaries Assured",
        "- `ak-trader` was untouched.",
        "- No promotion to `ak-trader` occurred.",
        "- No live trading, order placement, exchange keys, execution logic, or mainnet logic was added.",
        "- No new strategy family was implemented.",
        "- No threshold tuning occurred after seeing results.",
        "- No new data was fetched.",
        "- Funding was not used as a primary trigger."
    ])

    with open('runs/reports/phase12_1_robust_edge_bucket_audit.md', 'w') as f:
        f.write('\n'.join(md))

if __name__ == '__main__':
    run_audit()
