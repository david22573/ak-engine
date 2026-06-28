#!/usr/bin/env python3
import json
import math
import os
import sys
from collections import deque

def percentile(data, p):
    if not data:
        return 0.0
    sorted_data = sorted(data)
    n = len(sorted_data)
    idx = (n - 1) * p
    lower = int(math.floor(idx))
    upper = int(math.ceil(idx))
    weight = idx - lower
    if lower == upper:
        return sorted_data[lower]
    return sorted_data[lower] * (1 - weight) + sorted_data[upper] * weight

def get_funding_bucket(rate, z, p20, p80):
    if z <= -1 or rate <= p20:
        return "negative_extreme"
    if z >= 1 or rate >= p80:
        return "positive_extreme"
    if rate < 0:
        return "negative_moderate"
    if rate > 0:
        return "positive_moderate"
    return "flat"

class ReplayAdapter:
    def __init__(self):
        self.history = deque()
        
    def update(self, rate):
        self.history.append(rate)

    def stats(self, rate):
        if len(self.history) < 20:
            return 0.0, 0.0, 0.0, 0.0, 0.0, False
        mean = sum(self.history) / len(self.history)
        ss = sum((x - mean) ** 2 for x in self.history)
        sd = math.sqrt(ss / len(self.history))
        if sd == 0:
            return mean, sd, 0.0, 0.0, 0.0, False
            
        z = (rate - mean) / sd
        p20 = percentile(self.history, 0.20)
        p80 = percentile(self.history, 0.80)
        return mean, sd, z, p20, p80, True

def run_parity_check(ledger_path, out_prefix):
    metrics = {
        "research_event_count": 0,
        "reconstructed_shadow_signal_count": 0,
        "exact_timestamp_matches": 0,
        "false_positives": 0,
        "false_negatives": 0,
        "match_rate": 0.0,
        "missing_input_count": 0,
        "invalid_data_count": 0,
        "no_trade_count": 0,
        "mismatch_examples": []
    }
    
    if not os.path.exists(ledger_path):
        print(f"Error: {ledger_path} not found.")
        sys.exit(1)
        
    adapter = ReplayAdapter()
    
    # We will replay over the ledger events themselves to prove we can
    # reconstruct the stats causally! Since the ledger contains ALL evaluation ticks for XRPUSDT
    
    with open(ledger_path, 'r') as f:
        for line in f:
            if not line.strip():
                continue
            row = json.loads(line)
            metrics["research_event_count"] += 1
            
            # Use Replay Adapter
            rate = row["funding_rate"]
            mean, sd, z, p20, p80, ok = adapter.stats(rate)
            
            # Update causal history AFTER stats evaluation
            adapter.update(rate)
            
            is_trade = row.get("no_trade_reason") == ""
            if not is_trade:
                metrics["no_trade_count"] += 1
                if row.get("no_trade_reason") == "warmup":
                    metrics["missing_input_count"] += 1
                elif row.get("no_trade_reason") == "unsupported_context":
                    metrics["invalid_data_count"] += 1
                
            # Simulate shadow decision
            shadow_trade = False
            if ok and row.get("valid_regime_state"):
                if z <= -1 or rate <= p20:
                    shadow_trade = True
                    
            if shadow_trade:
                metrics["reconstructed_shadow_signal_count"] += 1
                
            # Parity check
            match = (is_trade == shadow_trade)
            if match:
                metrics["exact_timestamp_matches"] += 1
            else:
                if shadow_trade and not is_trade:
                    metrics["false_positives"] += 1
                    if len(metrics["mismatch_examples"]) < 20:
                        metrics["mismatch_examples"].append({"time": row["event_time_ms"], "type": "fp", "reason": "shadow traded, research did not", "row": row})
                elif not shadow_trade and is_trade:
                    metrics["false_negatives"] += 1
                    if len(metrics["mismatch_examples"]) < 20:
                        metrics["mismatch_examples"].append({"time": row["event_time_ms"], "type": "fn", "reason": "research traded, shadow did not", "row": row})

    if metrics["research_event_count"] > 0:
        metrics["match_rate"] = metrics["exact_timestamp_matches"] / metrics["research_event_count"]
        
    out_json = out_prefix + ".json"
    with open(out_json, "w") as f:
        json.dump(metrics, f, indent=2)
        
    out_md = out_prefix + ".md"
    with open(out_md, "w") as f:
        f.write("# XRPUSDT Replay Parity Report\n\n")
        f.write(f"- match_rate: {metrics['match_rate']*100:.2f}%\n")
        f.write(f"- research_event_count: {metrics['research_event_count']}\n")
        f.write(f"- exact_timestamp_matches: {metrics['exact_timestamp_matches']}\n")
        f.write(f"- false_positives: {metrics['false_positives']}\n")
        f.write(f"- false_negatives: {metrics['false_negatives']}\n")
        
    print(f"Parity evaluation complete. Output saved to {out_json}")

if __name__ == "__main__":
    import argparse
    parser = argparse.ArgumentParser()
    parser.add_argument("--ledger", required=True)
    parser.add_argument("--out-prefix", required=True)
    args = parser.parse_args()
    
    run_parity_check(args.ledger, args.out_prefix)
