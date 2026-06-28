#!/bin/bash
while true; do
  OUTPUT=$(python3 scripts/phase10_7n_v2_batch_runner.py --run-id full_v2_2024_2025 --batch-size 12 --resume)
  echo "$OUTPUT"
  if echo "$OUTPUT" | grep -q "pending_chunks: 0"; then
    echo "All chunks processed."
    break
  fi
  if echo "$OUTPUT" | grep -q "All chunks processed."; then
    echo "All chunks processed."
    break
  fi
done
