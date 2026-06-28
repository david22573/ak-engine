import json
import collections

filepath = '/home/davidmiguel22573/Github/ak-engine/runs/reports/chunks/LINKUSDT/2024-01-funding-events.jsonl'

by_family = collections.Counter()
by_side = collections.Counter()
by_horizon = collections.Counter()

dup_sfst = collections.Counter()
dup_sfsth = collections.Counter()

total_rows = 0

with open(filepath, 'r') as f:
    for line in f:
        line = line.strip()
        if not line: continue
        try:
            row = json.loads(line)
        except:
            continue
        
        total_rows += 1
        
        family = row.get('family', 'unknown')
        side = row.get('side', 'unknown')
        horizon = row.get('horizon', 'unknown')
        symbol = row.get('symbol', 'unknown')
        time_ms = row.get('event_time_ms', 0)
        
        by_family[family] += 1
        by_side[side] += 1
        by_horizon[horizon] += 1
        
        k1 = f"{symbol}_{time_ms}_{family}_{side}"
        dup_sfst[k1] += 1
        
        k2 = f"{symbol}_{time_ms}_{family}_{side}_{horizon}"
        dup_sfsth[k2] += 1

print(f"Total Rows: {total_rows}")
print("Rows by family:", dict(by_family))
print("Rows by side:", dict(by_side))
print("Rows by horizon:", dict(by_horizon))

dups1 = sum(1 for v in dup_sfst.values() if v > 1)
print(f"Duplicate count by symbol+time+family+side (keys with >1 row): {dups1}")
max_dup1 = max(dup_sfst.values()) if dup_sfst else 0
print(f"Max rows for a single symbol+time+family+side: {max_dup1}")

dups2 = sum(1 for v in dup_sfsth.values() if v > 1)
print(f"Duplicate count by symbol+time+family+side+horizon (keys with >1 row): {dups2}")

