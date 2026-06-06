#!/bin/bash
set -e

mkdir -p runs/features runs/regimes runs/reports

for SYM in BTCUSDT ETHUSDT SOLUSDT ADAUSDT DOGEUSDT BNBUSDT XRPUSDT AVAXUSDT; do
    echo "Processing $SYM"
    
    if [ "$SYM" = "BTCUSDT" ]; then
        CTX="ETHUSDT"
    elif [ "$SYM" = "ETHUSDT" ]; then
        CTX="BTCUSDT"
    else
        CTX="BTCUSDT,ETHUSDT"
    fi

    echo "Building features for $SYM with context $CTX"
    go run ./cmd/ak-engine build-features \
      --source local-parquet \
      --path ../ak-historian/.ak-historian/work \
      --market futures-um \
      --symbol $SYM \
      --interval 1m \
      --from 2023-01-01 \
      --to 2023-12-31 \
      --context-symbols $CTX \
      --out runs/features/${SYM}-2023-FY-context.json \
      --format json

    echo "Classifying regimes for $SYM"
    go run ./cmd/ak-engine classify-regimes \
      --features runs/features/${SYM}-2023-FY-context.json \
      --out runs/regimes/${SYM}-2023-FY-context.json \
      --format json \
      --threshold-lookback 43200 \
      --threshold-min-rows 1000

    echo "Evaluating compression breakout for $SYM"
    go run ./cmd/ak-engine evaluate-compression-breakout \
      --features runs/features/${SYM}-2023-FY-context.json \
      --regimes runs/regimes/${SYM}-2023-FY-context.json \
      --side long \
      --out runs/reports/phase10_1_compression_breakout_${SYM}_FY2023.md
done
