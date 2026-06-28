import json
import glob

manifest_path = 'runs/manifests/phase10_5_low_resource_manifest.json'
with open(manifest_path) as f:
    manifest = json.load(f)

symbols = ['LINKUSDT', 'SOLUSDT', 'AVAXUSDT']
years = ['2024', '2025']

results = []

for sym in symbols:
    for year in years:
        months_done = 0
        months_failed = 0
        months_skipped = 0
        total_rows = 0
        failed_chunks = []
        error_if_failed = ""
        
        for k, v in manifest.get('chunks', {}).items():
            if v.get('symbol') == sym and v.get('month', '').startswith(year):
                if v.get('feature_status') == 'DONE' and v.get('regime_status') == 'DONE':
                    months_done += 1
                elif 'failed' in v.get('error_if_failed', '').lower():
                    months_failed += 1
                    failed_chunks.append(v.get('month'))
                    error_if_failed = v.get('error_if_failed')
                total_rows += v.get('row_count', 0)
        
        feat_chunks = len(glob.glob(f"runs/features/chunks/{sym}/{year}-*-context.json"))
        reg_chunks = len(glob.glob(f"runs/features/chunks/{sym}/{year}-*-regime.json"))
        sum_chunks = len(glob.glob(f"runs/reports/chunks/{sym}/{year}-*-summary.json"))
        
        results.append({
            "symbol": sym,
            "year": year,
            "months_requested": 12,
            "months_done": months_done,
            "months_failed": months_failed,
            "months_skipped": months_skipped,
            "feature_chunks_done": feat_chunks,
            "regime_chunks_done": reg_chunks,
            "summary_chunks_done": sum_chunks,
            "total_rows": total_rows,
            "failed_chunks": failed_chunks,
            "error_if_failed": error_if_failed
        })

with open('runs/reports/phase10_5c_manifest_audit.json', 'w') as f:
    json.dump(results, f, indent=2)

md = "# Manifest Audit\n\n"
md += "| Symbol | Year | Requested | Done | Failed | Skipped | Feat | Reg | Sum | Total Rows | Failed Chunks |\n"
md += "|---|---|---|---|---|---|---|---|---|---|---|\n"
for r in results:
    fail_str = ",".join(r['failed_chunks']) if r['failed_chunks'] else "None"
    md += f"| {r['symbol']} | {r['year']} | {r['months_requested']} | {r['months_done']} | {r['months_failed']} | {r['months_skipped']} | {r['feature_chunks_done']} | {r['regime_chunks_done']} | {r['summary_chunks_done']} | {r['total_rows']} | {fail_str} |\n"

with open('runs/reports/phase10_5c_manifest_audit.md', 'w') as f:
    f.write(md)
    
print("Audit generated.")
