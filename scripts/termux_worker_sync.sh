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
SSH_CONFIG_FILE=${SSH_CONFIG_FILE:-}

load_env_file() {
  if [[ -f "${ENV_FILE}" ]]; then
    # shellcheck disable=SC1090
    . "${ENV_FILE}"
  fi
}

ssh_args_array=()

init_ssh_args() {
  if [[ -n "${SSH_CONFIG_FILE}" ]]; then
    ssh_args_array=(-F "${SSH_CONFIG_FILE}")
  else
    ssh_args_array=()
  fi
  local source_args=${SSH_ARGS:-}
  if [[ -z "${source_args}" && -n "${RSYNC_RSH:-}" ]]; then
    source_args=${RSYNC_RSH#ssh}
  fi
  if [[ -n "${source_args}" ]]; then
    # shellcheck disable=SC2206
    ssh_args_array+=(${source_args})
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
  SSH_CONFIG_FILE
               Optional ssh config file path, for example '/dev/null' to ignore
               broken system ssh config snippets on constrained hosts.
  SSH_ARGS     Optional SSH arguments, for example '-p 8022'
  RSYNC_RSH    Backward-compatible fallback; if set, SSH args are derived from it

Behavior:
  - push: update the phone-side git checkout from local HEAD, then sync source
          code and lightweight repo files to the phone worker.
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

require_local_git() {
  if ! git -C "${LOCAL_REPO}" rev-parse --show-toplevel >/dev/null 2>&1; then
    echo "LOCAL_REPO must be a git repository" >&2
    exit 1
  fi
}

local_branch_name() {
  git -C "${LOCAL_REPO}" symbolic-ref --quiet --short HEAD 2>/dev/null || echo "sync-head"
}

local_origin_url() {
  git -C "${LOCAL_REPO}" remote get-url origin 2>/dev/null || true
}

sync_remote_git_checkout() {
  require_host
  require_local_git

  local branch bundle_path origin_url remote_origin_cmd
  branch=$(local_branch_name)
  origin_url=$(local_origin_url)
  bundle_path=$(mktemp "${TMPDIR:-/tmp}/ak-engine-phone-sync.XXXXXX.bundle")

  remote_origin_cmd="true"
  if [[ -n "${origin_url}" ]]; then
    remote_origin_cmd="if git -C \"${PHONE_REPO}\" remote get-url origin >/dev/null 2>&1; then \
      git -C \"${PHONE_REPO}\" remote set-url origin \"${origin_url}\"; \
    else \
      git -C \"${PHONE_REPO}\" remote add origin \"${origin_url}\"; \
    fi"
  fi

  git -C "${LOCAL_REPO}" bundle create "${bundle_path}" HEAD >/dev/null
  cat "${bundle_path}" | run_ssh "tmp_bundle=\$(mktemp) && \
    cat > \"\${tmp_bundle}\" && \
    mkdir -p \"${PHONE_REPO}\" && \
    if [ ! -d \"${PHONE_REPO}/.git\" ]; then git -C \"${PHONE_REPO}\" init -b \"${branch}\" >/dev/null; fi && \
    ${remote_origin_cmd} && \
    git -C \"${PHONE_REPO}\" fetch \"\${tmp_bundle}\" \"HEAD:refs/remotes/origin/${branch}\" >/dev/null && \
    git -C \"${PHONE_REPO}\" checkout --force -B \"${branch}\" FETCH_HEAD >/dev/null && \
    git -C \"${PHONE_REPO}\" branch --set-upstream-to \"origin/${branch}\" \"${branch}\" >/dev/null && \
    git -C \"${PHONE_REPO}\" reset --hard FETCH_HEAD >/dev/null && \
    rm -f \"\${tmp_bundle}\""
  rm -f "${bundle_path}"
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
  sync_remote_git_checkout
  local rsync_ssh_cmd="${SSH_BIN}"
  if [[ -n "${SSH_CONFIG_FILE}" ]]; then
    rsync_ssh_cmd+=" -F ${SSH_CONFIG_FILE}"
  fi
  if [[ -n "${SSH_ARGS}" ]]; then
    rsync_ssh_cmd+=" ${SSH_ARGS}"
  fi
  if has_rsync; then
    rsync -av --delete -e "${rsync_ssh_cmd}" \
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
  local rsync_ssh_cmd="${SSH_BIN}"
  if [[ -n "${SSH_CONFIG_FILE}" ]]; then
    rsync_ssh_cmd+=" -F ${SSH_CONFIG_FILE}"
  fi
  if [[ -n "${SSH_ARGS}" ]]; then
    rsync_ssh_cmd+=" ${SSH_ARGS}"
  fi
  if has_rsync; then
    rsync -av -e "${rsync_ssh_cmd}" \
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
  local rsync_ssh_cmd="${SSH_BIN}"
  if [[ -n "${SSH_CONFIG_FILE}" ]]; then
    rsync_ssh_cmd+=" -F ${SSH_CONFIG_FILE}"
  fi
  if [[ -n "${SSH_ARGS}" ]]; then
    rsync_ssh_cmd+=" ${SSH_ARGS}"
  fi
  if has_rsync; then
    rsync -av -e "${rsync_ssh_cmd}" \
      --include 'phase10_*.json' \
      --include 'phase10_*.md' \
      --exclude 'chunks/' \
      --exclude '*' \
      "${PHONE_HOST}:${PHONE_REPO}/runs/reports/" \
      "${LOCAL_REPO}/runs/reports/"
    rsync -av -e "${rsync_ssh_cmd}" \
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
