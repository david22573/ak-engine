#!/bin/bash
set -e

for sym in BTCUSDT ETHUSDT SOLUSDT ADAUSDT DOGEUSDT BNBUSDT XRPUSDT AVAXUSDT; do
  echo "Running $sym"
  CTX="BTCUSDT,ETHUSDT"
  if [ "$sym" == "BTCUSDT" ]; then
    CTX="ETHUSDT"
  elif [ "$sym" == "ETHUSDT" ]; then
    CTX="BTCUSDT"
  fi

  go run ./cmd/ak-engine build-features \
    --source local-parquet \
    --path ../ak-historian/.ak-historian/work \
    --market futures-um \
    --symbol $sym \
    --interval 1m \
    --from 2023-01-01 \
    --to 2023-12-31 \
    --context-symbols $CTX \
    --out runs/features/${sym}-2023-FY-context.json \
    --format json

  go run ./cmd/ak-engine classify-regimes \
    --features runs/features/${sym}-2023-FY-context.json \
    --out runs/regimes/${sym}-2023-FY-context.json \
    --format json \
    --threshold-lookback 43200 \
    --threshold-min-rows 1000

  go run ./cmd/ak-engine evaluate-compression-breakout \
    --features runs/features/${sym}-2023-FY-context.json \
    --regimes runs/regimes/${sym}-2023-FY-context.json \
    --side long \
    --out runs/reports/phase10_1_compression_breakout_${sym}_FY.md
done
