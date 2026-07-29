#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR=$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")" && pwd)
REPO_ROOT=$(cd -- "${SCRIPT_DIR}/.." && pwd)

DEFAULT_SYMBOLS="ADAUSDT,AVAXUSDT,BNBUSDT,DOGEUSDT,ETHUSDT,LINKUSDT,SOLUSDT"
DEFAULT_FROM="2024-01"
DEFAULT_TO="2025-12"

export GOCACHE="${GOCACHE:-${REPO_ROOT}/.cache/go-build}"
export GOMODCACHE="${GOMODCACHE:-${REPO_ROOT}/.cache/go-mod}"
export GOWORK="${GOWORK:-off}"
export GOTOOLCHAIN="${GOTOOLCHAIN:-local}"

source "${SCRIPT_DIR}/historian_env.sh"

usage() {
  cat <<'EOF'
Usage:
  scripts/phase10_9c_phone_worker.sh coverage
  scripts/phase10_9c_phone_worker.sh run-symbol SYMBOL [FROM] [TO]
  scripts/phase10_9c_phone_worker.sh raw-files [SYMBOL]

Commands:
  coverage
    Run the retained compact-summary coverage and ranked inventory scan.

  run-symbol SYMBOL [FROM] [TO]
    Run the funding-event pipeline for one symbol across the requested month span,
    then delete raw event JSONL after aggregate reports are written.

  raw-files [SYMBOL]
    Print heavy raw files still present in the active report paths. When SYMBOL is
    provided, limit the check to that symbol plus top-level phase report JSONL files.

Environment:
  AK_HISTORIAN_WORKDIR  Historian parquet root. If unset, this script auto-uses
                        $HOME/Github/ak-historian/.ak-historian/work when present.
  GOCACHE, GOMODCACHE, GOWORK, GOTOOLCHAIN
                        Override repo-local Go execution defaults if needed.
EOF
}

run_go() {
  (
    cd "${REPO_ROOT}"
    go run ./cmd/ak-engine "$@"
  )
}

coverage() {
  run_go analyze-compact-robustness \
    --coverage-only \
    --family NegativeFundingLong \
    --side long \
    --symbols "${PHASE10_SYMBOLS:-${DEFAULT_SYMBOLS}}" \
    --from "${PHASE10_FROM:-${DEFAULT_FROM}}" \
    --to "${PHASE10_TO:-${DEFAULT_TO}}" \
    --reports-dir runs/reports \
    --chunks-dir runs/reports/chunks
}

run_symbol() {
  local symbol=${1:-}
  local from=${2:-${PHASE10_FROM:-${DEFAULT_FROM}}}
  local to=${3:-${PHASE10_TO:-${DEFAULT_TO}}}

  if [[ -z "${symbol}" ]]; then
    echo "SYMBOL is required" >&2
    usage >&2
    exit 1
  fi

  run_go phase10-funding-event-pipeline \
    --workdir "${HISTORIAN_WORKDIR}" \
    --symbols "${symbol}" \
    --from "${from}" \
    --to "${to}" \
    --max-symbols 1 \
    --retain-policy reports_only \
    --retain-event-detail \
    --summary-only-after-aggregate \
    --event-format jsonl.gz \
    --out "runs/reports/phase10_9c_${symbol}_pipeline.md"
}

raw_files() {
  local symbol=${1:-}
  (
    cd "${REPO_ROOT}"
    if [[ -n "${symbol}" ]]; then
      if [[ -d "runs/reports/chunks/${symbol}" ]]; then
        find "runs/reports/chunks/${symbol}" -type f \( -name "*.jsonl" -o -name "*.jsonl.gz" -o -name "*events*.json" \) -print
      fi
      if [[ -d runs/reports ]]; then
        find runs/reports -maxdepth 1 -type f \( -name "*${symbol}*.jsonl" -o -name "*${symbol}*.jsonl.gz" -o -name "*${symbol}*events*.json" \) -print
      fi
      exit 0
    fi
    if [[ -d runs/reports/chunks ]]; then
      find runs/reports/chunks -type f \( -name "*.jsonl" -o -name "*.jsonl.gz" -o -name "*events*.json" \) -print
    fi
    if [[ -d runs/reports ]]; then
      find runs/reports -maxdepth 1 -type f \( -name "*.jsonl" -o -name "*.jsonl.gz" -o -name "*events*.json" \) -print
    fi
  )
}

cmd=${1:-}
case "${cmd}" in
  coverage)
    coverage
    ;;
  run-symbol)
    shift
    run_symbol "$@"
    ;;
  raw-files)
    shift
    raw_files "$@"
    ;;
  ""|-h|--help|help)
    usage
    ;;
  *)
    echo "unknown command: ${cmd}" >&2
    usage >&2
    exit 1
    ;;
esac
