#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR=$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")" && pwd)
LOCAL_REPO=$(cd -- "${SCRIPT_DIR}/.." && pwd)
DEFAULT_ENV_FILE="${LOCAL_REPO}/termux_worker.env"

PHONE_HOST=${PHONE_HOST:-}
PHONE_REPO=${PHONE_REPO:-\$HOME/Github/ak-engine}
ENV_FILE=${ENV_FILE:-${DEFAULT_ENV_FILE}}
SSH_BIN=${SSH_BIN:-ssh}
SSH_ARGS=${SSH_ARGS:-}

load_env_file() {
  if [[ -f "${ENV_FILE}" ]]; then
    # shellcheck disable=SC1090
    . "${ENV_FILE}"
  fi
}

ssh_args_array=()

init_ssh_args() {
  local source_args=${SSH_ARGS:-}
  if [[ -z "${source_args}" && -n "${RSYNC_RSH:-}" ]]; then
    source_args=${RSYNC_RSH#ssh}
  fi
  if [[ -n "${source_args}" ]]; then
    # shellcheck disable=SC2206
    ssh_args_array=(${source_args})
  else
    ssh_args_array=()
  fi
}

usage() {
  cat <<'EOF'
Usage:
  PHONE_HOST=user@host scripts/termux_worker_sync.sh push
  PHONE_HOST=user@host scripts/termux_worker_sync.sh pull-summaries
  PHONE_HOST=user@host scripts/termux_worker_sync.sh pull-reports
  PHONE_HOST=user@host scripts/termux_worker_sync.sh remote-mkdir

Environment:
  PHONE_HOST   Required. SSH target or alias for the Termux phone worker.
  PHONE_REPO   Remote repo path. Defaults to $HOME/Github/ak-engine
  ENV_FILE     Optional env file. Defaults to ./termux_worker.env
  SSH_BIN      SSH binary. Defaults to ssh
  SSH_ARGS     Optional SSH arguments, for example '-p 8022'
  RSYNC_RSH    Backward-compatible fallback; if set, SSH args are derived from it

Behavior:
  - push: sync source code and lightweight repo files to the phone worker.
  - pull-summaries: pull only compact *-alpha-summary.json artifacts.
  - pull-reports: pull phase reports and the chunks directory.
  - remote-mkdir: create the remote repo directory over SSH.
EOF
}

require_host() {
  if [[ -z "${PHONE_HOST}" ]]; then
    echo "PHONE_HOST is required" >&2
    usage >&2
    exit 1
  fi
}

run_ssh() {
  "${SSH_BIN}" "${ssh_args_array[@]}" "${PHONE_HOST}" "$@"
}

has_rsync() {
  command -v rsync >/dev/null 2>&1
}

remote_mkdir() {
  require_host
  run_ssh "mkdir -p ${PHONE_REPO}"
}

push_repo_tar() {
  tar -C "${LOCAL_REPO}" -cf - \
    --exclude='.git' \
    --exclude='.cache' \
    --exclude='runs' \
    --exclude='.ak-engine' \
    --exclude='*.jsonl' \
    --exclude='*.jsonl.gz' \
    --exclude='*events*.json' \
    --exclude='ak-engine-code.zip' \
    --exclude='ak-engine' \
    . | run_ssh "mkdir -p ${PHONE_REPO} && tar -xf - -C ${PHONE_REPO}"
}

push_repo() {
  require_host
  if has_rsync; then
    rsync -av --delete -e "${SSH_BIN} ${SSH_ARGS}" \
      --exclude '.git' \
      --exclude '.cache' \
      --exclude 'runs/' \
      --exclude '.ak-engine/' \
      --exclude '*.jsonl' \
      --exclude '*.jsonl.gz' \
      --exclude '*events*.json' \
      --exclude 'ak-engine-code.zip' \
      --exclude 'ak-engine' \
      "${LOCAL_REPO}/" "${PHONE_HOST}:${PHONE_REPO}/"
    return
  fi
  push_repo_tar
}

pull_summaries_tar() {
  mkdir -p "${LOCAL_REPO}/runs/reports/chunks"
  run_ssh "cd ${PHONE_REPO}/runs/reports/chunks 2>/dev/null && find . -type f -name '*-alpha-summary.json' -print | tar -T - -cf -" \
    | tar -xf - -C "${LOCAL_REPO}/runs/reports/chunks"
}

pull_summaries() {
  require_host
  mkdir -p "${LOCAL_REPO}/runs/reports/chunks"
  if has_rsync; then
    rsync -av -e "${SSH_BIN} ${SSH_ARGS}" \
      --include '*/' \
      --include '*-alpha-summary.json' \
      --exclude '*' \
      "${PHONE_HOST}:${PHONE_REPO}/runs/reports/chunks/" \
      "${LOCAL_REPO}/runs/reports/chunks/"
    return
  fi
  pull_summaries_tar
}

pull_reports_tar() {
  mkdir -p "${LOCAL_REPO}/runs/reports" "${LOCAL_REPO}/runs/reports/chunks"
  run_ssh "cd ${PHONE_REPO}/runs/reports 2>/dev/null && find . -maxdepth 1 -type f \\( -name 'phase10_*.json' -o -name 'phase10_*.md' \\) -print | tar -T - -cf -" \
    | tar -xf - -C "${LOCAL_REPO}/runs/reports"
  run_ssh "cd ${PHONE_REPO}/runs/reports/chunks 2>/dev/null && find . -type f ! -name '*.jsonl' ! -name '*.jsonl.gz' ! -name '*events*.json' -print | tar -T - -cf -" \
    | tar -xf - -C "${LOCAL_REPO}/runs/reports/chunks"
}

pull_reports() {
  require_host
  mkdir -p "${LOCAL_REPO}/runs/reports" "${LOCAL_REPO}/runs/reports/chunks"
  if has_rsync; then
    rsync -av -e "${SSH_BIN} ${SSH_ARGS}" \
      --include 'phase10_*.json' \
      --include 'phase10_*.md' \
      --exclude 'chunks/' \
      --exclude '*' \
      "${PHONE_HOST}:${PHONE_REPO}/runs/reports/" \
      "${LOCAL_REPO}/runs/reports/"
    rsync -av -e "${SSH_BIN} ${SSH_ARGS}" \
      --exclude '*.jsonl' \
      --exclude '*.jsonl.gz' \
      --exclude '*events*.json' \
      "${PHONE_HOST}:${PHONE_REPO}/runs/reports/chunks/" \
      "${LOCAL_REPO}/runs/reports/chunks/"
    return
  fi
  pull_reports_tar
}

main() {
  load_env_file
  init_ssh_args
  local cmd=${1:-}
  case "${cmd}" in
    push)
      push_repo
      ;;
    pull-summaries)
      pull_summaries
      ;;
    pull-reports)
      pull_reports
      ;;
    remote-mkdir)
      remote_mkdir
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
}

main "$@"
