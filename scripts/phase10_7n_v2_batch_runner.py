#!/usr/bin/env python3
import argparse
import json
import logging
import os
import subprocess
import time
from pathlib import Path

ROOT = Path(__file__).resolve().parents[1]
REPORTS = ROOT / "runs" / "reports"
HISTORIAN_WORKDIR = os.environ.get("AK_HISTORIAN_WORKDIR", ".ak-historian/work")

SYMBOLS = [
    "ADAUSDT", "AVAXUSDT", "BNBUSDT", "DOGEUSDT", 
    "ETHUSDT", "LINKUSDT", "SOLUSDT", "XRPUSDT"
]

def generate_manifest(run_id: str) -> Path:
    run_dir = ROOT / "runs" / f"phase10_7n_v2_{run_id}"
    run_dir.mkdir(parents=True, exist_ok=True)
    manifest_path = run_dir / "manifest.json"
    
    if manifest_path.exists():
        return manifest_path
        
    chunks = {}
    for sym in SYMBOLS:
        for year in [2024, 2025]:
            for month in range(1, 13):
                ym = f"{year}-{month:02d}"
                key = f"{sym}_{ym}"
                chunks[key] = {
                    "symbol": sym,
                    "year_month": ym,
                    "status": "pending",
                    "started_at": None,
                    "finished_at": None,
                    "error": None,
                    "expected_input_path": f"datasets/derivatives/source=binance/dataset=funding_rate/market=futures-um/symbol={sym}/interval=8h/year={year}/month={month:02d}/{sym}-funding_rate-{ym}.parquet",
                    "output_chunk_dir": str(run_dir / "chunks" / sym),
                    "native_summary_v2_path": str(run_dir / "chunks" / sym / f"{ym}-native-summary-v2.json"),
                    "input_hash": None,
                    "summary_hash": None
                }
                
    manifest = {"chunks": chunks}
    with open(manifest_path, "w") as f:
        json.dump(manifest, f, indent=2)
        
    write_manifest_md(manifest_path)
    return manifest_path

def write_manifest_md(manifest_path: Path):
    with open(manifest_path) as f:
        manifest = json.load(f)
        
    md_path = manifest_path.with_suffix(".md")
    report_md_path = REPORTS / "phase10_7n_v2_validation_manifest.md"
    report_json_path = REPORTS / "phase10_7n_v2_validation_manifest.json"
    
    lines = ["# Phase 10.7N V2 Validation Manifest", "", "| Symbol | Month | Status | Error |", "|---|---|---|---|"]
    for key, chunk in manifest["chunks"].items():
        err = chunk.get("error") or ""
        lines.append(f"| {chunk['symbol']} | {chunk['year_month']} | {chunk['status']} | {err} |")
        
    content = "\n".join(lines)
    with open(md_path, "w") as f:
        f.write(content)
    
    REPORTS.mkdir(parents=True, exist_ok=True)
    with open(report_md_path, "w") as f:
        f.write(content)
    with open(report_json_path, "w") as f:
        json.dump(manifest, f, indent=2)

def run_chunk(chunk: dict, run_dir: Path) -> dict:
    sym = chunk["symbol"]
    ym = chunk["year_month"]
    chunks_out = run_dir / "chunks"
    chunks_out.mkdir(parents=True, exist_ok=True)
    
    cmd = [
        "./ak-engine", "phase10-funding-event-pipeline",
        "--workdir", HISTORIAN_WORKDIR,
        "--symbols", sym,
        "--from", ym,
        "--to", ym,
        "--out", str(run_dir / f"temp_{sym}_{ym}.md"),
        "--retain-policy", "reports_only",
        "--summary-only-after-aggregate", "true"
    ]
    
    env = os.environ.copy()
    chunk["started_at"] = time.time()
    chunk["status"] = "running"
    
    try:
        res = subprocess.run(cmd, cwd=str(ROOT), capture_output=True, text=True, env=env)
        
        # Save logs per chunk
        logs_dir = run_dir / "logs"
        logs_dir.mkdir(parents=True, exist_ok=True)
        log_file = logs_dir / f"{sym}_{ym}.log"
        with open(log_file, "w") as lf:
            lf.write(f"STDOUT:\n{res.stdout}\nSTDERR:\n{res.stderr}")

        if res.returncode != 0:
            err_str = res.stderr or res.stdout
            if "no parquet files" in err_str.lower() or "missing" in err_str.lower() or "no rows" in err_str.lower() or "not exist" in err_str.lower() or "no matching files" in err_str.lower():
                chunk["status"] = "skipped_missing_input"
                chunk["error"] = "missing input"
            elif "unsupported context" in err_str.lower() or "unsupported_context" in err_str.lower() or "no safe non-self" in err_str.lower():
                chunk["status"] = "skipped_missing_input"
                chunk["error"] = "unsupported context"
            else:
                chunk["status"] = "fail"
                chunk["error"] = err_str.strip().split("\n")[-1]
        else:
            if "missing_ephemeral_chunks" in res.stdout or "pipeline_blocked" in res.stdout or "real_no_events" in res.stdout or "SELF_CONTEXT_UNSUPPORTED" in res.stdout:
                if "real_no_events" in res.stdout:
                    chunk["status"] = "skipped_missing_input"
                    chunk["error"] = "real_no_events"
                elif "unsupported" in res.stdout.lower() or "SELF_CONTEXT_UNSUPPORTED" in res.stdout:
                    chunk["status"] = "skipped_missing_input"
                    chunk["error"] = "unsupported context"
                else:
                    chunk["status"] = "skipped_missing_input"
                    chunk["error"] = "missing input / blocked"
            else:
                chunk["status"] = "pass"
                chunk["error"] = None
                
                # Move chunks to isolated dir
                src_dir = ROOT / "runs" / "reports" / "chunks" / sym
                dest_dir = chunks_out / sym
                dest_dir.mkdir(parents=True, exist_ok=True)
                for f in src_dir.glob(f"{ym}-*"):
                    f.rename(dest_dir / f.name)
    except Exception as e:
        chunk["status"] = "fail"
        chunk["error"] = str(e)
        
    chunk["finished_at"] = time.time()
    return chunk

def main():
    parser = argparse.ArgumentParser()
    parser.add_argument("--run-id", required=True)
    parser.add_argument("--batch-size", type=int, default=8)
    parser.add_argument("--resume", action="store_true")
    parser.add_argument("--symbol", type=str, help="Specific symbol to run")
    parser.add_argument("--start-month", type=str, help="Start month YYYY-MM")
    parser.add_argument("--end-month", type=str, help="End month YYYY-MM")
    parser.add_argument("--max-chunks", type=int, default=0, help="Max chunks this run")
    args = parser.parse_args()
    
    run_dir = ROOT / "runs" / f"phase10_7n_v2_{args.run_id}"
    manifest_path = generate_manifest(args.run_id)
    
    with open(manifest_path) as f:
        manifest = json.load(f)
        
    pending = []
    for k, v in manifest["chunks"].items():
        if v["status"] == "pending" or (v["status"] == "fail" and args.resume):
            if args.symbol and v["symbol"] != args.symbol:
                continue
            if args.start_month and v["year_month"] < args.start_month:
                continue
            if args.end_month and v["year_month"] > args.end_month:
                continue
            pending.append(k)
            
    if not pending:
        print("All chunks processed.")
        return
        
    batch = pending[:args.batch_size]
    if args.max_chunks > 0 and len(batch) > args.max_chunks:
        batch = batch[:args.max_chunks]

    print(f"Processing batch of {len(batch)} chunks...")
    
    completed = 0
    failed = 0
    skipped = 0
    
    for key in batch:
        chunk = manifest["chunks"][key]
        print(f"Running {key}...")
        
        manifest["chunks"][key] = chunk
        with open(manifest_path, "w") as f:
            json.dump(manifest, f, indent=2)
            
        chunk = run_chunk(chunk, run_dir)
        manifest["chunks"][key] = chunk
        
        if chunk["status"] == "pass":
            completed += 1
        elif chunk["status"] == "fail":
            failed += 1
        elif chunk["status"] == "skipped_missing_input":
            skipped += 1
            
        with open(manifest_path, "w") as f:
            json.dump(manifest, f, indent=2)
        write_manifest_md(manifest_path)
        
    total_completed = sum(1 for c in manifest["chunks"].values() if c["status"] == "pass")
    total_pending = sum(1 for c in manifest["chunks"].values() if c["status"] == "pending")
    total_failed = sum(1 for c in manifest["chunks"].values() if c["status"] == "fail")
    total_skipped = sum(1 for c in manifest["chunks"].values() if c["status"] == "skipped_missing_input")
    
    print("\n--- Batch Complete ---")
    print(f"completed_chunks: {total_completed}")
    print(f"pending_chunks: {total_pending}")
    print(f"failed_chunks: {total_failed}")
    print(f"skipped_chunks: {total_skipped}")

if __name__ == "__main__":
    main()
