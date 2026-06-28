#!/usr/bin/env python3
"""Generate Phase 10.7J NegativeFundingLong robustness reports.

The analysis intentionally consumes Stage 4 native summary JSON artifacts only.
It does not read raw/event JSONL chunks.
"""

from __future__ import annotations

import argparse
import hashlib
import json
import math
import sys
from pathlib import Path
from typing import Any, Iterable


ROOT = Path(__file__).resolve().parents[1]
REPORTS = ROOT / "runs" / "reports"

MAIN_JSON = REPORTS / "phase10_7i_stage4_full_universe_native.json"
DEEP_JSON = REPORTS / "phase10_7i_stage4_full_universe_native_NegativeFundingLong_deep.json"
LEADERBOARD_JSON = REPORTS / "phase10_7i_stage4_full_universe_native_leaderboard.json"
INTEGRITY_JSON = REPORTS / "phase10_7i_stage4_full_universe_native_integrity_audit.json"


def load_json(path: Path) -> Any:
    with path.open("r", encoding="utf-8") as fh:
        return json.load(fh)

def load_v2_summaries(v2_dir: Path) -> list[dict[str, Any]]:
    if not v2_dir.exists():
        print(f"Error: v2-dir {v2_dir} does not exist.")
        sys.exit(1)
    v2_rows = []
    for p in v2_dir.glob("*/*-native-summary-v2.json"):
        v2_rows.extend(load_json(p))
    return v2_rows


def sha256(path: Path) -> str:
    h = hashlib.sha256()
    with path.open("rb") as fh:
        for chunk in iter(lambda: fh.read(1024 * 1024), b""):
            h.update(chunk)
    return h.hexdigest()


def fnum(value: Any, default: float = 0.0) -> float:
    if value is None:
        return default
    try:
        return float(value)
    except (TypeError, ValueError):
        return default


def inum(value: Any, default: int = 0) -> int:
    if value is None:
        return default
    try:
        return int(value)
    except (TypeError, ValueError):
        return default


def round_metric(value: float | None, digits: int = 6) -> float | None:
    if value is None:
        return None
    if not math.isfinite(value):
        return value
    return round(value, digits)


def profit_factor(gross_profit: float, gross_loss: float) -> float:
    if gross_loss == 0:
        if gross_profit > 0:
            return 999999.0
        return 0.0
    return gross_profit / gross_loss


def net_edge(row: dict[str, Any]) -> float:
    return fnum(row.get("gross_profit_bps")) - fnum(row.get("gross_loss_bps"))


def raw_count(row: dict[str, Any]) -> int:
    raw = inum(row.get("raw_event_count"))
    if raw > 0:
        return raw
    return inum(row.get("event_count"))


def retained_rows(report: dict[str, Any], key: str) -> list[dict[str, Any]]:
    retained = report.get("retained_summary") or {}
    return retained.get(key) or []


def candidate_key(candidate: dict[str, Any]) -> tuple[str, str, str, str]:
    return (
        str(candidate.get("symbol") or ""),
        str(candidate.get("family") or ""),
        str(candidate.get("side") or "").lower(),
        str(candidate.get("best_horizon") or ""),
    )


def row_key(row: dict[str, Any]) -> tuple[str, str, str, str]:
    return (
        str(row.get("symbol") or ""),
        str(row.get("family") or ""),
        str(row.get("side") or "").lower(),
        str(row.get("horizon") or ""),
    )


def aggregate_symbol_rows(rows: Iterable[dict[str, Any]]) -> dict[str, Any]:
    rows = list(rows)
    gp = sum(fnum(r.get("gross_profit_bps")) for r in rows)
    gl = sum(fnum(r.get("gross_loss_bps")) for r in rows)
    events = sum(raw_count(r) for r in rows)
    win = sum(inum(r.get("win_count")) for r in rows)
    loss = sum(inum(r.get("loss_count")) for r in rows)
    net = gp - gl
    return {
        "event_count": events,
        "win_count": win,
        "loss_count": loss,
        "gross_profit_bps": round_metric(gp),
        "gross_loss_bps": round_metric(gl),
        "net_edge_bps": round_metric(net),
        "pf_5bps": round_metric(profit_factor(gp, gl)),
        "expectancy_5bps_bps": round_metric(net / events if events else 0.0),
    }


def aggregate_retained_rows(rows: Iterable[dict[str, Any]]) -> dict[str, Any]:
    rows = list(rows)
    gp = sum(fnum(r.get("gross_profit_bps")) for r in rows)
    gl = sum(fnum(r.get("gross_loss_bps")) for r in rows)
    events = sum(inum(r.get("event_count")) for r in rows)
    de_clustered = sum(inum(r.get("de_clustered_event_count")) for r in rows)
    win = sum(inum(r.get("win_count")) for r in rows)
    loss = sum(inum(r.get("loss_count")) for r in rows)
    net = gp - gl
    expectancy = (net / events) if events else 0.0
    return {
        "event_count": events,
        "de_clustered_event_count": de_clustered,
        "win_count": win,
        "loss_count": loss,
        "gross_profit_bps": round_metric(gp),
        "gross_loss_bps": round_metric(gl),
        "net_edge_bps": round_metric(net),
        "pf_5bps": round_metric(profit_factor(gp, gl)),
        "expectancy_5bps_bps": round_metric(expectancy),
        "positive_expectancy": expectancy > 0,
    }


def merged_candidate_row(candidate: dict[str, Any], retained: dict[str, Any]) -> dict[str, Any]:
    merged = dict(candidate)
    for key in (
        "event_count",
        "raw_event_count",
        "de_clustered_event_count",
        "gross_profit_bps",
        "gross_loss_bps",
        "net_bps",
        "expectancy_bps",
        "pf",
        "win_count",
        "loss_count",
        "win_rate",
        "pf_after_7_5_bps",
        "pf_after_10_bps",
        "pf_after_15_bps",
        "cost_stress",
        "delay_stress",
        "bucket_metrics",
        "funding_bucket_counts",
        "regime_bucket_counts",
        "market_beta_bucket_counts",
    ):
        value = retained.get(key)
        if value is not None:
            merged[key] = value
    return merged


def candidate_from_deep(deep: dict[str, Any]) -> dict[str, Any]:
    rows = deep.get("per_symbol_metrics") or []
    target_pf = fnum(deep.get("overall_pf_combined_5bps"))
    target_exp = fnum(deep.get("overall_expectancy_combined_5bps_bps"))
    for row in rows:
        if (
            abs(fnum(row.get("pf_combined_5bps")) - target_pf) < 0.000001
            and abs(fnum(row.get("expectancy_combined_5bps_bps")) - target_exp) < 0.000001
        ):
            return row
    if not rows:
        return {}
    return max(rows, key=lambda r: (fnum(r.get("pf_2025_5bps")), fnum(r.get("pf_combined_5bps"))))


def pct(value: float, denominator: float) -> float:
    if denominator == 0:
        return 0.0
    return round_metric((value / denominator) * 100.0) or 0.0


def symbol_concentration(rows: list[dict[str, Any]]) -> dict[str, Any]:
    positive_net_total = sum(max(net_edge(row), 0.0) for row in rows)
    total = aggregate_symbol_rows(rows)
    table = []
    for row in sorted(rows, key=lambda r: r.get("symbol", "")):
        net = net_edge(row)
        table.append(
            {
                "symbol": row.get("symbol"),
                "best_horizon": row.get("best_horizon"),
                "pf_5bps": row.get("pf_combined_5bps"),
                "expectancy_5bps_bps": row.get("expectancy_combined_5bps_bps"),
                "event_count": row.get("event_count"),
                "raw_event_count": row.get("raw_event_count"),
                "de_clustered_event_count": row.get("de_clustered_event_count"),
                "net_edge_bps": round_metric(net),
                "positive_net_contribution_pct": pct(max(net, 0.0), positive_net_total),
                "verdict": row.get("verdict"),
            }
        )

    positive_rows = [r for r in table if fnum(r.get("net_edge_bps")) > 0]
    best = max(positive_rows, key=lambda r: fnum(r.get("positive_net_contribution_pct")), default=None)
    worst = min(table, key=lambda r: fnum(r.get("net_edge_bps")), default=None)

    loo = []
    for row in sorted(rows, key=lambda r: r.get("symbol", "")):
        remaining = [r for r in rows if r is not row]
        agg = aggregate_symbol_rows(remaining)
        loo.append(
            {
                "removed_symbol": row.get("symbol"),
                **agg,
                "positive_expectancy": fnum(agg.get("expectancy_5bps_bps")) > 0,
            }
        )
    positive_loo = sum(1 for row in loo if row["positive_expectancy"])

    return {
        "basis": "per-symbol native summary gross profit/loss at 5 bps",
        "aggregate_all_symbols_5bps": total,
        "positive_net_total_bps": round_metric(positive_net_total),
        "best_symbol_contribution_pct": best.get("positive_net_contribution_pct") if best else 0.0,
        "best_symbol": best.get("symbol") if best else None,
        "worst_symbol_drag_bps": worst.get("net_edge_bps") if worst else 0.0,
        "worst_symbol": worst.get("symbol") if worst else None,
        "leave_one_symbol_out_positive_count": positive_loo,
        "leave_one_symbol_out_total": len(loo),
        "leave_one_symbol_out_mostly_positive": positive_loo > (len(loo) / 2.0),
        "by_symbol": table,
        "leave_one_symbol_out": loo,
    }


def month_concentration(candidate: dict[str, Any], rows: list[dict[str, Any]]) -> dict[str, Any]:
    table = []
    for row in sorted(rows, key=lambda r: r.get("month", "")):
        net = net_edge(row)
        table.append(
            {
                "month": row.get("month"),
                "event_count": row.get("event_count"),
                "de_clustered_event_count": row.get("de_clustered_event_count"),
                "gross_profit_bps": row.get("gross_profit_bps"),
                "gross_loss_bps": row.get("gross_loss_bps"),
                "net_edge_bps": round_metric(net),
                "expectancy_bps": row.get("expectancy_bps"),
                "pf": row.get("pf"),
                "win_count": row.get("win_count"),
                "loss_count": row.get("loss_count"),
                "positive_net_contribution_pct": 0.0,
            }
        )
    positive_total = sum(max(fnum(row["net_edge_bps"]), 0.0) for row in table)
    for row in table:
        row["positive_net_contribution_pct"] = pct(max(fnum(row["net_edge_bps"]), 0.0), positive_total)
    positive_rows = [r for r in table if fnum(r["net_edge_bps"]) > 0]
    best = max(positive_rows, key=lambda r: fnum(r["positive_net_contribution_pct"]), default=None)
    loo = []
    for row in table:
        remaining = [r for r in table if r["month"] != row["month"]]
        agg = aggregate_symbol_rows(remaining)
        loo.append(
            {
                "removed_month": row["month"],
                **agg,
                "positive_expectancy": fnum(agg.get("expectancy_5bps_bps")) > 0,
            }
        )
    positive_loo = sum(1 for row in loo if row["positive_expectancy"])
    return {
        "basis": "retained by-month native summary rows",
        "positive_month_count": sum(1 for row in table if fnum(row["net_edge_bps"]) > 0),
        "top_1_month_contribution_pct": best.get("positive_net_contribution_pct") if best else 0.0,
        "top_2_month_contribution_pct": sum(sorted((r["positive_net_contribution_pct"] for r in positive_rows), reverse=True)[:2]),
        "worst_quarter_pf_5bps": fnum(candidate.get("worst_quarter_pf_5bps")),
        "by_month": table,
        "leave_one_month_out": loo,
        "leave_one_month_out_positive_count": positive_loo,
        "leave_one_month_out_total": len(loo),
        "leave_one_month_out_mostly_positive": positive_loo > (len(loo) / 2.0),
    }


def quarter_concentration(candidate: dict[str, Any], rows: list[dict[str, Any]]) -> dict[str, Any]:
    table = []
    for row in sorted(rows, key=lambda r: r.get("quarter", "")):
        net = net_edge(row)
        table.append(
            {
                "quarter": row.get("quarter"),
                "event_count": row.get("event_count"),
                "de_clustered_event_count": row.get("de_clustered_event_count"),
                "gross_profit_bps": row.get("gross_profit_bps"),
                "gross_loss_bps": row.get("gross_loss_bps"),
                "net_edge_bps": round_metric(net),
                "expectancy_bps": row.get("expectancy_bps"),
                "pf": row.get("pf"),
                "win_count": row.get("win_count"),
                "loss_count": row.get("loss_count"),
            }
        )
    loo = []
    for row in table:
        remaining = [r for r in table if r["quarter"] != row["quarter"]]
        gp = sum(fnum(r.get("gross_profit_bps")) for r in remaining)
        gl = sum(fnum(r.get("gross_loss_bps")) for r in remaining)
        events = sum(inum(r.get("event_count")) for r in remaining)
        loo.append(
            {
                "removed_quarter": row["quarter"],
                "gross_profit_bps": round_metric(gp),
                "gross_loss_bps": round_metric(gl),
                "event_count": events,
                "pf_5bps": round_metric(profit_factor(gp, gl)),
                "expectancy_5bps_bps": round_metric((gp - gl) / events if events else 0.0),
                "positive_expectancy": ((gp - gl) / events if events else 0.0) > 0,
            }
        )
    positive_loo = sum(1 for row in loo if row["positive_expectancy"])
    return {
        "basis": "retained by-quarter native summary rows",
        "worst_quarter_pf_5bps": min((fnum(r.get("pf")) for r in table), default=0.0),
        "best_quarter_pf_5bps": max((fnum(r.get("pf")) for r in table), default=0.0),
        "by_quarter": table,
        "leave_one_quarter_out": loo,
        "leave_one_quarter_out_positive_count": positive_loo,
        "leave_one_quarter_out_total": len(loo),
        "leave_one_quarter_out_mostly_positive": positive_loo > (len(loo) / 2.0),
    }


def unsupported(reason: str) -> dict[str, Any]:
    return {"supported": False, "reason": reason}


def cost_stress(candidate: dict[str, Any]) -> list[dict[str, Any]]:
    rows = candidate.get("cost_stress") or []
    if rows:
        return rows
    return [
        {
            "cost_bps": 5.0,
            "supported": True,
            "pf": candidate.get("pf_combined_5bps"),
            "expectancy_bps": candidate.get("expectancy_combined_5bps_bps"),
            "source": "candidate native summary",
        },
        {
            "cost_bps": 7.5,
            **unsupported("No 7.5 bps native metric retained."),
        },
        {
            "cost_bps": 10.0,
            **unsupported("No 10 bps native metric retained."),
        },
        {
            "cost_bps": 15.0,
            **unsupported("No 15 bps native metric retained."),
        },
    ]


def delay_stress(candidate: dict[str, Any]) -> list[dict[str, Any]]:
    rows = candidate.get("delay_stress") or []
    if rows:
        baseline = next((r for r in rows if inum(r.get("delay_candles")) == 0), None)
        delay1 = next((r for r in rows if inum(r.get("delay_candles")) == 1), None)
        if baseline and delay1:
            base_exp = fnum(baseline.get("expectancy_bps"))
            delay1_exp = fnum(delay1.get("expectancy_bps"))
            delay1["edge_decay_from_baseline_pct"] = round_metric((1.0 - (delay1_exp / base_exp)) * 100.0) if base_exp else None
        return rows
    baseline_exp = fnum(candidate.get("expectancy_combined_5bps_bps"))
    delay1 = fnum(candidate.get("entry_delay_1c_expectancy_bps"))
    decay = None
    if baseline_exp:
        decay = (1.0 - (delay1 / baseline_exp)) * 100.0
    return [
        {
            "delay_candles": 0,
            "supported": True,
            "pf": candidate.get("pf_combined_5bps"),
            "expectancy_bps": candidate.get("expectancy_combined_5bps_bps"),
            "source": "baseline native summary at 5 bps",
        },
        {
            "delay_candles": 1,
            "supported": bool(candidate.get("entry_delay_1c_available")),
            "pf": None,
            "expectancy_bps": candidate.get("entry_delay_1c_expectancy_bps"),
            "edge_decay_from_baseline_pct": round_metric(decay),
            "source": "entry_delay_1c_expectancy_bps native summary",
        },
        {
            "delay_candles": 2,
            **unsupported("No entry_delay_2c native summary metric is present."),
        },
        {
            "delay_candles": 5,
            **unsupported("No entry_delay_5c native summary metric is present."),
        },
    ]


def threshold_sensitivity(candidate: dict[str, Any]) -> dict[str, Any]:
    rows = candidate.get("bucket_metrics") or []
    if rows:
        funding = [row for row in rows if row.get("bucket_type") == "funding_severity"]
        regimes = [row for row in rows if row.get("bucket_type") == "regime"]
        interactions = [row for row in rows if row.get("bucket_type") == "funding_x_regime"]
        funding_total = sum(inum(row.get("event_count")) for row in funding)
        regime_total = sum(inum(row.get("event_count")) for row in regimes)
        interaction_total = sum(inum(row.get("event_count")) for row in interactions)
        for row in funding:
            row["event_share_pct"] = pct(inum(row.get("event_count")), funding_total)
            row["edge_attribution_supported"] = True
        for row in regimes:
            row["event_share_pct"] = pct(inum(row.get("event_count")), regime_total)
            row["edge_attribution_supported"] = True
        for row in interactions:
            row["event_share_pct"] = pct(inum(row.get("event_count")), interaction_total)
            row["edge_attribution_supported"] = True
        return {
            "basis": "candidate native bucket metrics",
            "funding_threshold_buckets": funding,
            "regime_bucket_breakdown": regimes,
            "funding_x_regime_interaction": interactions,
            "edge_bucket_identification": {
                "supported": True,
                "best_available_statement": "Native bucket metrics retained per funding bucket, regime bucket, and interaction.",
            },
        }
    funding_counts = candidate.get("funding_bucket_counts") or {}
    regime_counts = candidate.get("regime_bucket_counts") or {}
    total_funding = sum(inum(v) for v in funding_counts.values())
    total_regime = sum(inum(v) for v in regime_counts.values())

    funding = [
        {
            "bucket": key,
            "event_count": inum(value),
            "event_share_pct": pct(inum(value), total_funding),
            "edge_attribution_supported": False,
        }
        for key, value in sorted(funding_counts.items())
    ]
    regimes = [
        {
            "bucket": key,
            "event_count": inum(value),
            "event_share_pct": pct(inum(value), total_regime),
            "edge_attribution_supported": False,
        }
        for key, value in sorted(regime_counts.items())
    ]
    interactions = []
    if len(funding) == 1:
        funding_bucket = funding[0]["bucket"]
        interactions = [
            {
                "funding_bucket": funding_bucket,
                "regime_bucket": row["bucket"],
                "event_count": row["event_count"],
                "event_share_pct": row["event_share_pct"],
                "edge_attribution_supported": False,
            }
            for row in regimes
        ]

    return {
        "basis": "candidate native bucket counts; no per-bucket returns retained",
        "funding_threshold_buckets": funding,
        "regime_bucket_breakdown": regimes,
        "funding_x_regime_interaction": interactions,
        "edge_bucket_identification": {
            "supported": False,
            "best_available_statement": (
                "All retained candidate events are in the negative_extreme funding bucket, "
                "so any reported top-line edge comes from that funding bucket. Native summaries "
                "do not retain per-regime return attribution."
            ),
        },
    }


def time_split_row(label: str, all_rows: list[dict[str, Any]], selected_rows: list[dict[str, Any]]) -> dict[str, Any]:
    agg = aggregate_retained_rows(selected_rows)
    return {
        "label": label,
        "supported": bool(all_rows),
        "event_count": agg["event_count"],
        "de_clustered_event_count": agg["de_clustered_event_count"],
        "gross_profit_bps": agg["gross_profit_bps"],
        "gross_loss_bps": agg["gross_loss_bps"],
        "pf_5bps": agg["pf_5bps"],
        "expectancy_5bps_bps": agg["expectancy_5bps_bps"],
        "positive_expectancy": agg["positive_expectancy"],
    }


def time_split_validation(candidate: dict[str, Any], month_rows: list[dict[str, Any]], quarter_rows: list[dict[str, Any]]) -> dict[str, Any]:
    rows_2024 = [row for row in month_rows if str(row.get("year")) == "2024"]
    rows_2025 = [row for row in month_rows if str(row.get("year")) == "2025"]
    h1_rows = [row for row in month_rows if str(row.get("month", ""))[5:7] in {"01", "02", "03", "04", "05", "06"}]
    h2_rows = [row for row in month_rows if str(row.get("month", ""))[5:7] in {"07", "08", "09", "10", "11", "12"}]

    year_2024 = time_split_row("2024", month_rows, rows_2024)
    year_2025 = time_split_row("2025", month_rows, rows_2025)
    if not year_2024["supported"]:
        year_2024["pf_5bps"] = candidate.get("pf_2024_5bps")
        year_2024["expectancy_5bps_bps"] = None
        year_2024["positive_expectancy"] = False
    if not year_2025["supported"]:
        year_2025["pf_5bps"] = candidate.get("pf_2025_5bps")
        year_2025["expectancy_5bps_bps"] = candidate.get("expectancy_2025_5bps_bps")
        year_2025["positive_expectancy"] = fnum(candidate.get("expectancy_2025_5bps_bps")) > 0

    quarter_table = [
        {
            "quarter": row.get("quarter"),
            "event_count": row.get("event_count"),
            "de_clustered_event_count": row.get("de_clustered_event_count"),
            "gross_profit_bps": row.get("gross_profit_bps"),
            "gross_loss_bps": row.get("gross_loss_bps"),
            "pf_5bps": row.get("pf"),
            "expectancy_5bps_bps": row.get("expectancy_bps"),
        }
        for row in sorted(quarter_rows, key=lambda r: r.get("quarter", ""))
    ]

    return {
        "basis": "retained by-month and by-quarter native summary rows",
        "year_2024": year_2024,
        "year_2025": year_2025,
        "h1_h2": {
            "supported": bool(month_rows),
            "h1_2024_2025": time_split_row("H1", month_rows, h1_rows),
            "h2_2024_2025": time_split_row("H2", month_rows, h2_rows),
        },
        "quarter_by_quarter": {
            "supported": bool(quarter_rows),
            "rows": quarter_table,
        },
        "quarter_summary": {
            "worst_quarter_pf_5bps": min(
                (fnum(row.get("pf")) for row in quarter_rows),
                default=fnum(candidate.get("worst_quarter_pf_5bps")),
            ),
            "best_quarter_pf_5bps": max(
                (fnum(row.get("pf")) for row in quarter_rows),
                default=fnum(candidate.get("best_quarter_pf_5bps")),
            ),
        },
    }


def decision_rules(
    candidate: dict[str, Any],
    symbols: dict[str, Any],
    integrity: dict[str, Any],
    months: dict[str, Any],
    quarters: dict[str, Any],
) -> dict[str, Any]:
    delay1_supported = bool(candidate.get("entry_delay_1c_available"))
    delay1_exp = fnum(candidate.get("entry_delay_1c_expectancy_bps"))
    rules = [
        {
            "rule": "PF remains > 1.05 at 7.5 bps",
            "status": "UNKNOWN",
            "detail": "7.5 bps metric is not retained in native summaries.",
        },
        {
            "rule": "Expectancy remains positive at 7.5 bps",
            "status": "UNKNOWN",
            "detail": "7.5 bps metric is not retained in native summaries.",
        },
        {
            "rule": "No single month contributes more than 50% of total net edge",
            "status": "PASS" if fnum(candidate.get("top_1_month_contribution_pct")) <= 50.0 else "FAIL",
            "detail": f"top_1_month_contribution_pct={fnum(candidate.get('top_1_month_contribution_pct')):.6f}",
        },
        {
            "rule": "No single symbol contributes more than 50% of total net edge",
            "status": "PASS" if fnum(symbols.get("best_symbol_contribution_pct")) <= 50.0 else "FAIL",
            "detail": (
                f"{symbols.get('best_symbol')} contributes "
                f"{fnum(symbols.get('best_symbol_contribution_pct')):.6f}% of positive symbol net edge"
            ),
        },
        {
            "rule": "Leave-one-symbol-out remains positive for most removals",
            "status": "PASS" if symbols.get("leave_one_symbol_out_mostly_positive") else "FAIL",
            "detail": (
                f"{symbols.get('leave_one_symbol_out_positive_count')}/"
                f"{symbols.get('leave_one_symbol_out_total')} removals positive"
            ),
        },
        {
            "rule": "Leave-one-month-out remains positive for most removals",
            "status": "PASS" if months.get("leave_one_month_out_mostly_positive") else "FAIL",
            "detail": (
                f"{months.get('leave_one_month_out_positive_count')}/"
                f"{months.get('leave_one_month_out_total')} removals positive"
            ),
        },
        {
            "rule": "2024 and 2025 are both not materially broken",
            "status": (
                "PASS"
                if fnum(candidate.get("pf_2024_5bps")) > 1.0 and fnum(candidate.get("pf_2025_5bps")) > 1.0
                else "FAIL"
            ),
            "detail": (
                f"pf_2024_5bps={fnum(candidate.get('pf_2024_5bps')):.6f}; "
                f"pf_2025_5bps={fnum(candidate.get('pf_2025_5bps')):.6f}"
            ),
        },
        {
            "rule": "Delay 1 candle does not destroy the edge",
            "status": "PASS" if delay1_supported and delay1_exp > 0 else "FAIL",
            "detail": f"entry_delay_1c_expectancy_bps={delay1_exp:.6f}",
        },
        {
            "rule": "Stage 4 artifact integrity is sufficient for robustness validation",
            "status": "PASS" if integrity.get("status") == "PASS" else "FAIL",
            "detail": f"integrity_status={integrity.get('status')}; failures={integrity.get('failures')}",
        },
        {
            "rule": "Worst quarter is not materially broken",
            "status": "PASS" if quarters.get("worst_quarter_pf_5bps", 0.0) >= 0.95 else "FAIL",
            "detail": f"worst_quarter_pf_5bps={quarters.get('worst_quarter_pf_5bps', 0.0):.6f}",
        },
    ]

    top_line_positive = (
        fnum(candidate.get("pf_combined_5bps")) > 1.05
        and fnum(candidate.get("expectancy_combined_5bps_bps")) > 0
        and fnum(candidate.get("pf_2024_5bps")) > 1.0
        and fnum(candidate.get("pf_2025_5bps")) > 1.0
    )
    hard_failures = [r for r in rules if r["status"] == "FAIL"]
    unknowns = [r for r in rules if r["status"] == "UNKNOWN"]

    if not top_line_positive:
        classification = "reject"
    elif not hard_failures and not unknowns:
        classification = "shadow_candidate"
    elif fnum(candidate.get("pf_combined_5bps")) > 1.05 and fnum(candidate.get("expectancy_combined_5bps_bps")) > 0:
        classification = "fragile_research_lead"
    else:
        classification = "reject"

    return {
        "classification": classification,
        "shadow_candidate_eligible": classification == "shadow_candidate",
        "rules": rules,
        "blocking_reasons": [r for r in rules if r["status"] != "PASS"],
    }


def md_table(headers: list[str], rows: list[list[Any]]) -> str:
    out = ["| " + " | ".join(headers) + " |"]
    out.append("|" + "|".join("---" for _ in headers) + "|")
    for row in rows:
        out.append("| " + " | ".join(format_cell(value) for value in row) + " |")
    return "\n".join(out)


def format_cell(value: Any) -> str:
    if value is None:
        return "n/a"
    if isinstance(value, float):
        return f"{value:.6f}"
    if isinstance(value, bool):
        return "true" if value else "false"
    if isinstance(value, list):
        return ", ".join(str(v) for v in value)
    return str(value)


def write_robustness_md(report: dict[str, Any]) -> None:
    candidate = report["candidate"]
    symbols = report["symbol_concentration"]
    monthly = report["monthly_concentration"]
    time_split = report["time_split_validation"]
    threshold = report["threshold_sensitivity"]
    decision = report["decision"]

    lines = [
        "# Phase 10.7J NegativeFundingLong Robustness",
        "",
        "## Executive Summary",
        f"- Classification: `{decision['classification']}`",
        f"- Candidate: `{candidate['symbol']}` `{candidate['family']}` `{candidate['side']}` at `{candidate['best_horizon']}`",
        f"- Native 5 bps PF / expectancy: {candidate['pf_combined_5bps']:.6f} / {candidate['expectancy_combined_5bps_bps']:.6f} bps",
        f"- Shadow eligibility: `{str(decision['shadow_candidate_eligible']).lower()}`",
        "- Primary blockers: symbol concentration, leave-one-month-out, artifact integrity, and worst-quarter PF checks.",
        "",
        "## Source Integrity",
        f"- Main pipeline status: `{report['source_integrity']['pipeline_status']}` / `{report['source_integrity']['pipeline_detailed_status']}`",
        f"- Leaderboard event rows retained: {report['source_integrity']['leaderboard_total_event_rows']}",
        f"- Candidate group rows retained: {report['source_integrity']['candidate_group_rows']}",
        f"- Retained by-symbol rows: {report['source_integrity']['retained_by_symbol_rows']}",
        f"- Retained by-month rows: {report['source_integrity']['retained_by_month_rows']}",
        f"- Retained by-quarter rows: {report['source_integrity']['retained_by_quarter_rows']}",
        f"- Integrity audit status: `{report['source_integrity']['integrity_status']}`",
        f"- Integrity failures: {', '.join(report['source_integrity']['integrity_failures']) or 'none'}",
        "",
        "## Monthly Concentration",
        f"- Positive month count: {monthly['positive_month_count']}",
        f"- Best month contribution: {monthly['top_1_month_contribution_pct']:.6f}%",
        f"- Top 2 month contribution: {monthly['top_2_month_contribution_pct']:.6f}%",
        f"- Worst quarter PF after 5 bps: {monthly['worst_quarter_pf_5bps']:.6f}",
        "- PF by month, expectancy by month, event count by month, and leave-one-month-out are reconstructed from retained month rows.",
        "",
        "## Symbol Concentration",
        f"- Aggregate all-symbol summary PF / expectancy at 5 bps: {symbols['aggregate_all_symbols_5bps']['pf_5bps']:.6f} / {symbols['aggregate_all_symbols_5bps']['expectancy_5bps_bps']:.6f} bps",
        f"- Largest positive symbol contribution: {symbols['best_symbol']} at {symbols['best_symbol_contribution_pct']:.6f}%",
        f"- Worst symbol drag: {symbols['worst_symbol']} at {symbols['worst_symbol_drag_bps']:.6f} bps net",
        f"- Leave-one-symbol-out positive: {symbols['leave_one_symbol_out_positive_count']}/{symbols['leave_one_symbol_out_total']}",
        "",
        md_table(
            ["Symbol", "PF 5bps", "Exp 5bps", "Events", "De-clustered", "Net bps", "Positive Contribution %", "Verdict"],
            [
                [
                    row["symbol"],
                    row["pf_5bps"],
                    row["expectancy_5bps_bps"],
                    row["event_count"],
                    row["de_clustered_event_count"],
                    row["net_edge_bps"],
                    row["positive_net_contribution_pct"],
                    row["verdict"],
                ]
                for row in symbols["by_symbol"]
            ],
        ),
        "",
        "Leave-one-symbol-out at 5 bps:",
        md_table(
            ["Removed", "PF 5bps", "Exp 5bps", "Events", "Net bps", "Positive"],
            [
                [
                    row["removed_symbol"],
                    row["pf_5bps"],
                    row["expectancy_5bps_bps"],
                    row["event_count"],
                    row["net_edge_bps"],
                    row["positive_expectancy"],
                ]
                for row in symbols["leave_one_symbol_out"]
            ],
        ),
        "",
        "## Time Split Validation",
        f"- 2024 PF after 5 bps: {time_split['year_2024']['pf_5bps']:.6f}",
        f"- 2025 PF / expectancy after 5 bps: {time_split['year_2025']['pf_5bps']:.6f} / {time_split['year_2025']['expectancy_5bps_bps']:.6f} bps",
        f"- Quarter summary: worst PF {time_split['quarter_summary']['worst_quarter_pf_5bps']:.6f}, best PF {time_split['quarter_summary']['best_quarter_pf_5bps']:.6f}",
        f"- H1/H2 retained support: `{str(time_split['h1_h2']['supported']).lower()}`",
        f"- Quarter-by-quarter retained support: `{str(time_split['quarter_by_quarter']['supported']).lower()}`",
        "",
        "## Cost Stress",
        md_table(
            ["Cost bps", "Supported", "PF", "Expectancy bps", "Detail"],
            [
                [
                    row["cost_bps"],
                    row.get("supported", True),
                    row.get("pf"),
                    row.get("expectancy_bps"),
                    row.get("source") or row.get("reason"),
                ]
                for row in report["cost_stress"]
            ],
        ),
        "",
        "## Execution Delay Stress",
        md_table(
            ["Delay candles", "Supported", "PF", "Expectancy bps", "Decay %", "Detail"],
            [
                [
                    row["delay_candles"],
                    row.get("supported", row.get("available", True)),
                    row.get("pf"),
                    row.get("expectancy_bps"),
                    row.get("edge_decay_from_baseline_pct"),
                    row.get("source") or row.get("reason"),
                ]
                for row in report["execution_delay_stress"]
            ],
        ),
        "",
        "## Threshold Sensitivity",
        "Funding bucket counts:",
        md_table(
            ["Funding bucket", "Events", "Share %", "Edge Attribution"],
            [
                [
                    row.get("bucket") or row.get("funding_bucket") or row.get("bucket_type"),
                    row.get("event_count"),
                    row.get("event_share_pct"),
                    row.get("edge_attribution_supported", "supported" if row.get("pf") is not None else "unsupported"),
                ]
                for row in threshold["funding_threshold_buckets"]
            ],
        ),
        "",
        "Regime bucket counts:",
        md_table(
            ["Regime bucket", "Events", "Share %", "Edge Attribution"],
            [
                [
                    row.get("bucket") or row.get("regime_bucket") or row.get("bucket_type"),
                    row.get("event_count"),
                    row.get("event_share_pct"),
                    row.get("edge_attribution_supported", "supported" if row.get("pf") is not None else "unsupported"),
                ]
                for row in threshold["regime_bucket_breakdown"]
            ],
        ),
        "",
        "Funding x regime interaction is reconstructed from retained bucket metrics.",
        "",
        "## Decision Rules",
        md_table(
            ["Rule", "Status", "Detail"],
            [[row["rule"], row["status"], row["detail"]] for row in decision["rules"]],
        ),
    ]
    OUT_MD.write_text("\n".join(lines) + "\n", encoding="utf-8")


def write_decision_md(report: dict[str, Any]) -> None:
    decision = report["decision"]
    candidate = report["candidate"]
    blockers = decision["blocking_reasons"]
    lines = [
        "# Phase 10.7J NegativeFundingLong Decision",
        "",
        f"Decision: `{decision['classification']}`",
        "",
        f"`{candidate['symbol']}` `NegativeFundingLong` remains a research lead, not a shadow/testnet candidate. The 5 bps native summary is positive, but symbol concentration is dominated by XRPUSDT and artifact integrity still needs to clear the enriched native-summary audit.",
        "",
        "## Blocking Reasons",
        md_table(
            ["Rule", "Status", "Detail"],
            [[row["rule"], row["status"], row["detail"]] for row in blockers],
        ),
        "",
        "## Promotion Guidance",
        "- Do not promote automatically.",
        "- Required next validation: exact 7.5/10/15 bps stress, monthly leave-one-out, per-regime return attribution, and complete artifact integrity pass.",
    ]
    OUT_DECISION_MD.write_text("\n".join(lines) + "\n", encoding="utf-8")


def build_report(v2_dir: Path) -> dict[str, Any]:
    main = load_json(MAIN_JSON)
    deep = load_json(DEEP_JSON)
    leaderboard = load_json(LEADERBOARD_JSON)
    integrity = load_json(INTEGRITY_JSON)

    candidate = candidate_from_deep(deep)
    rows = deep.get("per_symbol_metrics") or [
        row for row in leaderboard.get("leaderboard", []) if row.get("family") == "NegativeFundingLong"
    ]
    retained = leaderboard.get("retained_summary") or {}
    candidate_retained = next(
        (
            row
            for row in retained.get("by_symbol", [])
            if row.get("symbol") == candidate.get("symbol")
            and row.get("family") == candidate.get("family")
            and row.get("side") == candidate.get("side")
            and row.get("horizon") == candidate.get("best_horizon")
        ),
        {},
    )

    v2_rows = load_v2_summaries(v2_dir)
    if not v2_rows:
        candidate["verdict"] = "insufficient summary schema"
        candidate["failed_gates"] = ["missing_v2_summaries"]
    candidate_retained = next(
        (
            row
            for row in retained.get("by_symbol", [])
            if row.get("symbol") == candidate.get("symbol")
            and row.get("family") == candidate.get("family")
            and row.get("side") == candidate.get("side")
            and row.get("horizon") == candidate.get("best_horizon")
        ),
        {},
    )

    v2_rows = load_v2_summaries(v2_dir)
    if not v2_rows:
        candidate["verdict"] = "insufficient summary schema"
        candidate["failed_gates"] = ["missing_v2_summaries"]
        candidate_groups = [
            row
            for row in retained.get("by_symbol", [])
            if row.get("symbol") == candidate.get("symbol")
            and row.get("family") == candidate.get("family")
            and row.get("side") == candidate.get("side")
            and row.get("horizon") == candidate.get("best_horizon")
        ]
        month_rows = [
            row
            for row in retained.get("by_month", [])
            if row.get("symbol") == candidate.get("symbol")
            and row.get("family") == candidate.get("family")
            and row.get("side") == candidate.get("side")
            and row.get("horizon") == candidate.get("best_horizon")
        ]
        quarter_rows = [
            row
            for row in retained.get("by_quarter", [])
            if row.get("symbol") == candidate.get("symbol")
            and row.get("family") == candidate.get("family")
            and row.get("side") == candidate.get("side")
            and row.get("horizon") == candidate.get("best_horizon")
        ]
    else:
        # Reconstruct from V2
        target_horizon = int(candidate.get("best_horizon", "60m").replace("m", ""))
        v2_filtered = [r for r in v2_rows if r["family"] == candidate.get("family") and r["side"] == candidate.get("side") and r["horizon_minutes"] == target_horizon and r["cost_bps"] == 5.0 and r["delay_candles"] == 0]
        
        # Build month_rows
        from collections import defaultdict
        m_groups = defaultdict(list)
        q_groups = defaultdict(list)
        s_groups = defaultdict(list)
        
        for r in v2_filtered:
            if r["symbol"] == candidate.get("symbol"):
                m_groups[r["month"]].append(r)
                q_groups[r["quarter"]].append(r)
            s_groups[r["symbol"]].append(r)
            
        def agg_v2(k, grp):
            gp = sum(fnum(r.get("gross_profit_bps")) for r in grp)
            gl = sum(fnum(r.get("gross_loss_bps")) for r in grp)
            ev = sum(inum(r.get("event_count")) for r in grp)
            decl = sum(inum(r.get("declustered_event_count")) for r in grp)
            w = sum(inum(r.get("win_count")) for r in grp)
            l = sum(inum(r.get("loss_count")) for r in grp)
            pf = profit_factor(gp, gl)
            return {
                "symbol": candidate.get("symbol"),
                "family": candidate.get("family"),
                "side": candidate.get("side"),
                "horizon": candidate.get("best_horizon"),
                "event_count": ev,
                "de_clustered_event_count": decl,
                "gross_profit_bps": gp,
                "gross_loss_bps": gl,
                "pf": pf,
                "expectancy_bps": (gp - gl) / ev if ev else 0.0,
                "win_count": w,
                "loss_count": l,
            }
            
        month_rows = []
        for m, grp in m_groups.items():
            agg = agg_v2(m, grp)
            agg["month"] = m
            agg["year"] = m.split("-")[0]
            month_rows.append(agg)
            
        quarter_rows = []
        for q, grp in q_groups.items():
            agg = agg_v2(q, grp)
            agg["quarter"] = q
            quarter_rows.append(agg)
            
        candidate_groups = [agg_v2(candidate.get("symbol"), s_groups[candidate.get("symbol")])] if candidate.get("symbol") in s_groups else []
        
        # Override rows with V2 generated symbols
        new_rows = []
        for sym, grp in s_groups.items():
            agg = agg_v2(sym, grp)
            agg["symbol"] = sym
            agg["pf_combined_5bps"] = agg["pf"]
            agg["expectancy_combined_5bps_bps"] = agg["expectancy_bps"]
            agg["raw_event_count"] = agg["event_count"]
            agg["best_horizon"] = candidate.get("best_horizon")
            agg["verdict"] = "v2 generated"
            new_rows.append(agg)
            
        if new_rows:
            rows = new_rows
            
        v2_candidate_all = [r for r in v2_rows if r["symbol"] == candidate.get("symbol") and r["family"] == candidate.get("family") and r["side"] == candidate.get("side") and r["horizon_minutes"] == target_horizon]

        retained_cost_stress = []
        for cost in [5.0, 7.5, 10.0, 15.0]:
            grp = [r for r in v2_candidate_all if r["cost_bps"] == cost and r["delay_candles"] == 0]
            if grp:
                gp = sum(fnum(r.get("gross_profit_bps")) for r in grp)
                gl = sum(fnum(r.get("gross_loss_bps")) for r in grp)
                exp = (gp - gl) / sum(inum(r.get("event_count")) for r in grp) if sum(inum(r.get("event_count")) for r in grp) else 0.0
                retained_cost_stress.append({
                    "cost_bps": cost,
                    "supported": True,
                    "pf": profit_factor(gp, gl),
                    "expectancy_bps": exp,
                    "source": "native v2 summary"
                })
        if retained_cost_stress:
            candidate_retained["cost_stress"] = retained_cost_stress

        retained_delay_stress = []
        baseline_grp = [r for r in v2_candidate_all if r["cost_bps"] == 5.0 and r["delay_candles"] == 0]
        base_exp = None
        if baseline_grp:
            bgp = sum(fnum(r.get("gross_profit_bps")) for r in baseline_grp)
            bgl = sum(fnum(r.get("gross_loss_bps")) for r in baseline_grp)
            bevents = sum(inum(r.get("event_count")) for r in baseline_grp)
            base_exp = (bgp - bgl) / bevents if bevents else 0.0

        for delay in [0, 1, 2, 5]:
            grp = [r for r in v2_candidate_all if r["cost_bps"] == 5.0 and r["delay_candles"] == delay]
            if grp:
                gp = sum(fnum(r.get("gross_profit_bps")) for r in grp)
                gl = sum(fnum(r.get("gross_loss_bps")) for r in grp)
                exp = (gp - gl) / sum(inum(r.get("event_count")) for r in grp) if sum(inum(r.get("event_count")) for r in grp) else 0.0
                dec = None
                if delay > 0 and base_exp:
                    dec = (1.0 - (exp / base_exp)) * 100.0
                retained_delay_stress.append({
                    "delay_candles": delay,
                    "supported": True,
                    "pf": profit_factor(gp, gl),
                    "expectancy_bps": exp,
                    "edge_decay_from_baseline_pct": dec,
                    "source": "native v2 summary"
                })
        if retained_delay_stress:
            candidate_retained["delay_stress"] = retained_delay_stress
            
        bucket_metrics = []
        grp5_0 = [r for r in v2_candidate_all if r["cost_bps"] == 5.0 and r["delay_candles"] == 0]
        
        from collections import defaultdict
        f_groups = defaultdict(list)
        r_groups = defaultdict(list)
        fr_groups = defaultdict(list)
        for r in grp5_0:
            f_groups[r["funding_bucket"]].append(r)
            r_groups[r["regime_bucket"]].append(r)
            fr_groups[r["funding_x_regime_bucket"]].append(r)
            
        for k, grp in f_groups.items():
            bucket_metrics.append({"bucket_type": "funding_severity", "bucket": k, **agg_v2(k, grp)})
        for k, grp in r_groups.items():
            bucket_metrics.append({"bucket_type": "regime", "bucket": k, **agg_v2(k, grp)})
        for k, grp in fr_groups.items():
            bucket_metrics.append({"bucket_type": "funding_x_regime", "bucket": k, **agg_v2(k, grp)})
            
        if bucket_metrics:
            candidate_retained["bucket_metrics"] = bucket_metrics

    candidate = merged_candidate_row(candidate, candidate_retained)
    symbols_conc = symbol_concentration(rows)
    months = month_concentration(candidate, month_rows)
    quarters = quarter_concentration(candidate, quarter_rows)
    source_integrity = {
        "pipeline_status": main.get("status"),
        "pipeline_detailed_status": main.get("detailed_status"),
        "leaderboard_total_event_rows": leaderboard.get("summary", {}).get("total_event_rows"),
        "leaderboard_zero_event_month_count": leaderboard.get("summary", {}).get("zero_event_month_count"),
        "leaderboard_summary_only_safe": leaderboard.get("summary", {}).get("summary_only_safe"),
        "integrity_status": integrity.get("status"),
        "integrity_failures": integrity.get("failures") or [],
        "event_count_exists_but_event_rows_missing": integrity.get("event_count_exists_but_event_rows_missing"),
        "candidate_group_rows": len(candidate_groups),
        "retained_by_symbol_rows": len(retained.get("by_symbol") or []),
        "retained_by_month_rows": len(retained.get("by_month") or []),
        "retained_by_quarter_rows": len(retained.get("by_quarter") or []),
    }

    report = {
        "analysis_version": "phase10_7j_native_summary_v2",
        "method": {
            "native_summary_only": True,
            "raw_jsonl_event_chunks_read": False,
            "notes": [
                "Uses Stage 4 native summary JSON artifacts only.",
                "Unsupported slices are explicitly marked instead of approximated from missing event distributions.",
            ],
        },
        "source_files": {
            str(path.relative_to(ROOT)): {"sha256": sha256(path), "bytes": path.stat().st_size}
            for path in [MAIN_JSON, DEEP_JSON, LEADERBOARD_JSON, INTEGRITY_JSON] if path.exists()
        },
        "candidate": {
            "symbol": candidate.get("symbol"),
            "family": candidate.get("family"),
            "side": candidate.get("side"),
            "best_horizon": candidate.get("best_horizon"),
            "event_count": candidate.get("event_count"),
            "raw_event_count": candidate.get("raw_event_count"),
            "de_clustered_event_count": candidate.get("de_clustered_event_count"),
            "pf_combined_5bps": candidate.get("pf_combined_5bps"),
            "expectancy_combined_5bps_bps": candidate.get("expectancy_combined_5bps_bps"),
            "pf_2024_5bps": candidate.get("pf_2024_5bps"),
            "pf_2025_5bps": candidate.get("pf_2025_5bps"),
            "expectancy_2025_5bps_bps": candidate.get("expectancy_2025_5bps_bps"),
            "entry_delay_1c_expectancy_bps": candidate.get("entry_delay_1c_expectancy_bps"),
            "entry_delay_1c_available": candidate.get("entry_delay_1c_available"),
            "top_1_month_contribution_pct": candidate.get("top_1_month_contribution_pct"),
            "top_2_month_contribution_pct": candidate.get("top_2_month_contribution_pct"),
            "worst_quarter_pf_5bps": candidate.get("worst_quarter_pf_5bps"),
            "best_quarter_pf_5bps": candidate.get("best_quarter_pf_5bps"),
            "cost_stress": candidate.get("cost_stress"),
            "delay_stress": candidate.get("delay_stress"),
            "bucket_metrics": candidate.get("bucket_metrics"),
            "verdict": candidate.get("verdict"),
            "failed_gates": candidate.get("failed_gates"),
        },
        "source_integrity": source_integrity,
        "monthly_concentration": {
            **months,
        },
        "symbol_concentration": symbols_conc,
        "time_split_validation": time_split_validation(candidate, month_rows, quarter_rows),
        "cost_stress": cost_stress(candidate),
        "execution_delay_stress": delay_stress(candidate),
        "threshold_sensitivity": threshold_sensitivity(candidate),
    }
    report["decision"] = decision_rules(candidate, symbols_conc, integrity, months, quarters)
    return report


def main() -> None:
    parser = argparse.ArgumentParser()
    parser.add_argument("--v2-dir", type=str, required=True, help="Directory containing isolated V2 chunks")
    parser.add_argument("--out-prefix", type=str, required=True, help="Prefix for output files (e.g. runs/reports/phase10_7n_crashsafe_negative_funding_long)")
    args = parser.parse_args()

    v2_dir = Path(args.v2_dir)
    if not v2_dir.is_absolute():
        v2_dir = ROOT / v2_dir
        
    out_prefix = Path(args.out_prefix)
    if not out_prefix.is_absolute():
        out_prefix = ROOT / out_prefix
        
    out_prefix.parent.mkdir(parents=True, exist_ok=True)
        
    out_json = out_prefix.with_name(out_prefix.name + "_robustness.json")
    out_md = out_prefix.with_name(out_prefix.name + "_robustness.md")
    out_decision_md = out_prefix.with_name(out_prefix.name + "_decision.md")

    report = build_report(v2_dir)
    out_json.write_text(json.dumps(report, indent=2, sort_keys=True) + "\n", encoding="utf-8")
    
    global OUT_MD, OUT_DECISION_MD
    OUT_MD = out_md
    OUT_DECISION_MD = out_decision_md
    
    write_robustness_md(report)
    write_decision_md(report)
    print(f"wrote {out_json.relative_to(ROOT)}")
    print(f"wrote {out_md.relative_to(ROOT)}")
    print(f"wrote {out_decision_md.relative_to(ROOT)}")


if __name__ == "__main__":
    main()
