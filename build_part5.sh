#!/bin/bash
set -e

mkdir -p runs/features runs/regimes

for SYMBOL in LINKUSDT SOLUSDT AVAXUSDT; do
  for RANGE in "2024-01-01 2025-12-31 2024_2025" "2024-01-01 2024-12-31 2024" "2025-01-01 2025-12-31 2025"; do
    FROM=$(echo $RANGE | awk '{print $1}')
    TO=$(echo $RANGE | awk '{print $2}')
    NAME=$(echo $RANGE | awk '{print $3}')

    echo "Building features for $SYMBOL $NAME..."
    go run ./cmd/ak-engine build-features \
      --source local-parquet \
      --path ../ak-historian/.ak-historian/work \
      --market futures-um \
      --symbol $SYMBOL \
      --interval 1m \
      --from $FROM \
      --to $TO \
      --context-symbols BTCUSDT,ETHUSDT \
      --out runs/features/$SYMBOL-$NAME-context.json \
      --format json

    echo "Classifying regimes for $SYMBOL $NAME..."
    go run ./cmd/ak-engine classify-regimes \
      --features runs/features/$SYMBOL-$NAME-context.json \
      --out runs/regimes/$SYMBOL-$NAME-context.json \
      --format json \
      --threshold-lookback 43200 \
      --threshold-min-rows 1000
  done
done
echo "DONE Part 5"
