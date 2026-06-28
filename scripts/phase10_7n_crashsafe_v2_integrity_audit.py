#!/usr/bin/env python3
import argparse
import json
import subprocess
from pathlib import Path

ROOT = Path(__file__).resolve().parents[1]
REPORTS = ROOT / "runs" / "reports"

def main():
    parser = argparse.ArgumentParser()
    parser.add_argument("--run-id", required=True)
    args = parser.parse_args()

    run_dir = ROOT / "runs" / f"phase10_7n_v2_{args.run_id}"
    manifest_path = run_dir / "manifest.json"
    chunks_dir = run_dir / "chunks"

    if not manifest_path.exists():
        print(f"Manifest not found: {manifest_path}")
        return

    with open(manifest_path) as f:
        manifest = json.load(f)

    # 1. Run ak-engine audit-v2-integrity
    print("Running ak-engine audit-v2-integrity...")
    res = subprocess.run(
        ["./ak-engine", "audit-v2-integrity", "--path", str(chunks_dir)],
        cwd=str(ROOT), capture_output=True, text=True
    )
    
    try:
        engine_audit = json.loads(res.stdout)
    except Exception as e:
        engine_audit = {"Status": "FAIL", "Failures": [f"Could not parse ak-engine audit output: {e}\n{res.stdout}"]}

    # 2. Audit against manifest
    expected_chunks = 0
    actual_chunks = 0
    passed_chunks = 0
    failed_chunks = 0
    skipped_chunks = 0
    symbols = set()
    months = set()

    for k, v in manifest.get("chunks", {}).items():
        expected_chunks += 1
        if v.get("status") == "pass":
            passed_chunks += 1
            symbols.add(v.get("symbol"))
            months.add(v.get("year_month"))
        elif v.get("status") == "fail":
            failed_chunks += 1
        elif v.get("status") == "skipped_missing_input":
            skipped_chunks += 1
            
    v2_files_present = list(chunks_dir.glob("*/*-native-summary-v2.json"))
    actual_chunks = len(v2_files_present)

    failures = []
    if engine_audit.get("Status") != "PASS":
        failures.extend(engine_audit.get("Failures", []))

    if actual_chunks != passed_chunks:
        failures.append(f"Mismatch between passed chunks in manifest ({passed_chunks}) and V2 files on disk ({actual_chunks})")

    # Check for no stale/partial 10.7M chunks
    stale_found = False
    for f in chunks_dir.glob("**/*"):
        if f.is_file() and "phase10_7m" in str(f):
            stale_found = True
            failures.append(f"Stale partial 10.7M chunk found: {f}")

    # Check for no 10.5 CSV inputs
    csv_found = False
    for f in chunks_dir.glob("**/*.csv"):
        csv_found = True
        failures.append(f"Phase 10.5 CSV inputs found in chunks dir: {f}")

    audit_result = {
        "status": "PASS" if not failures else "FAIL",
        "failures": failures,
        "engine_audit": engine_audit,
        "metrics": {
            "expected_chunks_total": expected_chunks,
            "manifest_passed": passed_chunks,
            "manifest_failed": failed_chunks,
            "manifest_skipped": skipped_chunks,
            "actual_v2_files_on_disk": actual_chunks,
            "symbols_with_passed_chunks": sorted(list(symbols)),
            "months_with_passed_chunks": sorted(list(months)),
            "no_phase10_5_csv_inputs": not csv_found,
            "no_partial_10_7m_chunks": not stale_found,
        }
    }

    out_json = REPORTS / "phase10_7n_crashsafe_v2_integrity_audit.json"
    out_md = REPORTS / "phase10_7n_crashsafe_v2_integrity_audit.md"

    with open(out_json, "w") as f:
        json.dump(audit_result, f, indent=2)

    lines = [
        "# Phase 10.7N-R Crash-Safe V2 Integrity Audit",
        "",
        f"**Status**: `{audit_result['status']}`",
        "",
        "## Metrics",
        f"- Expected chunks (total): {expected_chunks}",
        f"- Manifest Passed: {passed_chunks}",
        f"- Manifest Skipped: {skipped_chunks}",
        f"- Manifest Failed: {failed_chunks}",
        f"- Actual V2 Files on Disk: {actual_chunks}",
        f"- Symbols Covered: {len(symbols)}",
        f"- Months Covered: {len(months)}",
        f"- No 10.5 CSV inputs: {not csv_found}",
        f"- No 10.7M stale partials: {not stale_found}",
        "",
        "## Failures"
    ]
    if failures:
        for fail in failures:
            lines.append(f"- {fail}")
    else:
        lines.append("- None")

    with open(out_md, "w") as f:
        f.write("\n".join(lines) + "\n")

    print(f"Audit Status: {audit_result['status']}")
    print(f"Passed chunks: {passed_chunks}")
    if failures:
        print("Failures:")
        for fail in failures:
            print(f" - {fail}")

if __name__ == "__main__":
    main()
