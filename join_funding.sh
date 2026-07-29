#!/bin/bash
source "$(dirname "$0")/scripts/historian_env.sh"

for SYMBOL in LINKUSDT SOLUSDT AVAXUSDT; do
  for YEAR in 2024 2025; do
    for MONTH in {01..12}; do
      FEATURE="runs/features/chunks/${SYMBOL}/${YEAR}-${MONTH}-context.json"
      OUT="runs/features/chunks/${SYMBOL}/${YEAR}-${MONTH}-funding.json"
      DERIV="${HISTORIAN_WORKDIR}/datasets/derivatives/source=binance/dataset=funding_rate/market=futures-um/symbol=${SYMBOL}"
      
      if [ -f "$FEATURE" ]; then
        echo "Joining $SYMBOL ${YEAR}-${MONTH}"
        ./ak-engine join-research-features \
          --features "$FEATURE" \
          --derivatives "$DERIV" \
          --out "$OUT"
      fi
    done
  done
done
