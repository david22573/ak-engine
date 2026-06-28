#!/bin/bash
set -e

SYMBOLS="LINKUSDT,SOLUSDT,AVAXUSDT"
START_MONTH="2024-01"
END_MONTH="2025-12"

MONTHS=()
current=$START_MONTH
while [[ "$current" < "$END_MONTH" || "$current" == "$END_MONTH" ]]; do
  MONTHS+=("$current")
  Y=$(echo $current | cut -d'-' -f1)
  M=$(echo $current | cut -d'-' -f2)
  M=$((10#$M + 1))
  if [ $M -eq 13 ]; then
    M=1
    Y=$((Y + 1))
  fi
  current=$(printf "%04d-%02d" $Y $M)
done

for SYM in LINKUSDT SOLUSDT AVAXUSDT; do
  for M in "${MONTHS[@]}"; do
    echo "Processing $SYM for $M"
    ./ak-engine audit-funding-chunk --symbol $SYM --month $M
    ./ak-engine evaluate-funding-chunk --symbol $SYM --month $M
  done
done

echo "Aggregating reports..."
./ak-engine aggregate-funding-audit
./ak-engine aggregate-funding-alpha-baselines
echo "Done."
