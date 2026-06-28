#!/bin/bash
set -e

HISTORIAN_WORKDIR="${AK_HISTORIAN_WORKDIR:-.ak-historian/work}"
AK_HISTORIAN_BIN="${AK_HISTORIAN_BIN:-../ak-historian/ak-historian}"
SYMBOLS=("DOGEUSDT" "ADAUSDT" "BNBUSDT" "XRPUSDT" "ETHUSDT")
YEARS=("2024" "2025")

for SYM in "${SYMBOLS[@]}"; do
  for Y in "${YEARS[@]}"; do
    CTX="BTCUSDT,ETHUSDT"
    if [ "$SYM" == "ETHUSDT" ]; then
      CTX="BTCUSDT"
    fi

    echo "Running prep for $SYM $Y"
    ./ak-engine phase10-low-resource-prep \
      --workdir "$HISTORIAN_WORKDIR" \
      --symbols $SYM \
      --context-symbols $CTX \
      --from $Y-01 \
      --to $Y-12 \
      --chunk monthly \
      --max-symbols 1 \
      --max-months 1 \
      --max-rows 50000 \
      --min-free-gb 5 \
      --disk-budget-gb 8 \
      --cleanup-after-chunk \
      --retain-policy reports_only \
      --gc-between-chunks \
      --verbose \
      --out runs/reports/phase10_7b_${SYM}_${Y}_low_resource.md

    echo "Aggregating chunk reports for $SYM $Y"
    ./ak-engine aggregate-chunk-reports \
      --chunks runs/reports/chunks \
      --symbols $SYM \
      --from $Y-01 \
      --to $Y-12 \
      --out runs/reports/phase10_7b_${SYM}_${Y}_aggregate.md
      
    echo "Cleaning up verified target source parquet for $SYM $Y"
    "$AK_HISTORIAN_BIN" cleanup-workdir \
      --workdir "$HISTORIAN_WORKDIR" \
      --market futures-um \
      --interval 1m \
      --symbols $SYM \
      --from $Y-01 \
      --to $Y-12 \
      --force \
      --only-verified-archive \
      --retain-symbols BTCUSDT,ETHUSDT
  done
done
