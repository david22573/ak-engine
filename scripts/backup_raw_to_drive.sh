#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR=$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")" && pwd)
REPO_ROOT=$(cd -- "${SCRIPT_DIR}/.." && pwd)

SOURCE_DIR=${AK_RAW_SOURCE_DIR:-runs/reports/chunks}
REMOTE=${AK_RAW_REMOTE:-gdrive:ak-engine-raw-archive/funding-events}
LOG_DIR=${AK_RAW_LOG_DIR:-runs/reports}
INCLUDE_PATTERN=${AK_RAW_INCLUDE_PATTERN:-*-funding-events.jsonl.gz}

usage() {
  cat <<'EOF'
Usage:
  scripts/backup_raw_to_drive.sh [copy|check|backup]

Commands:
  count    Count raw funding-event gzip files in the source directory.
  copy     Copy raw funding-event gzip files to the configured rclone remote.
  check    Verify source files exist on the configured rclone remote.
  backup   Copy, then verify. This is the default.

Environment:
  AK_RAW_REMOTE
             Rclone destination. Defaults to:
             gdrive:ak-engine-raw-archive/funding-events
             Use gcrypt:funding-events for an encrypted rclone crypt remote.
  AK_RAW_SOURCE_DIR
             Source directory. Defaults to runs/reports/chunks.
  AK_RAW_LOG_DIR
             Log directory. Defaults to runs/reports.
  AK_RAW_INCLUDE_PATTERN
             Rclone include pattern. Defaults to *-funding-events.jsonl.gz.
  RCLONE_TRANSFERS
             Upload transfer count. Defaults to 2.
  RCLONE_CHECKERS
             Rclone checker count. Defaults to 4.
  RCLONE_DRIVE_CHUNK_SIZE
             Google Drive chunk size. Defaults to 64M.
  RCLONE_STATS_INTERVAL
             Stats interval for non-interactive runs. Defaults to 30s.
  RCLONE_TIMEOUT
             I/O idle timeout. Defaults to 30s.
  RCLONE_CONTIMEOUT
             Connection timeout. Defaults to 15s.
  RCLONE_RETRIES
             High-level retry count. Defaults to 5.
  RCLONE_LOW_LEVEL_RETRIES
             Low-level retry count. Defaults to 5.
  RCLONE_TPSLIMIT
             API transactions per second limit. Defaults to 1.
  RCLONE_DRIVE_PACER_MIN_SLEEP
             Minimum Google Drive pacer sleep. Defaults to 1s.

This script copies raw files. It does not move or delete phone-side files.
EOF
}

require_rclone() {
  if ! command -v rclone >/dev/null 2>&1; then
    echo "rclone is required. Install it in Termux with: pkg install rclone" >&2
    exit 127
  fi
}

count_raw() {
  (
    cd "${REPO_ROOT}"
    if [[ ! -d "${SOURCE_DIR}" ]]; then
      echo 0
      exit 0
    fi
    find "${SOURCE_DIR}" -type f -name "${INCLUDE_PATTERN}" | wc -l
  )
}

rclone_status_args() {
  if [[ -t 1 ]]; then
    printf '%s\n' --progress
    return
  fi
  printf '%s\n' --stats "${RCLONE_STATS_INTERVAL:-30s}"
}

copy_raw() {
  (
    cd "${REPO_ROOT}"
    mkdir -p "${LOG_DIR}"
    echo "Uploading raw funding event gzip files to: ${REMOTE}"
    mapfile -t status_args < <(rclone_status_args)
    rclone copy "${SOURCE_DIR}" "${REMOTE}" \
      --include "${INCLUDE_PATTERN}" \
      --transfers "${RCLONE_TRANSFERS:-2}" \
      --checkers "${RCLONE_CHECKERS:-4}" \
      --drive-chunk-size "${RCLONE_DRIVE_CHUNK_SIZE:-64M}" \
      --timeout "${RCLONE_TIMEOUT:-30s}" \
      --contimeout "${RCLONE_CONTIMEOUT:-15s}" \
      --retries "${RCLONE_RETRIES:-5}" \
      --low-level-retries "${RCLONE_LOW_LEVEL_RETRIES:-5}" \
      --retries-sleep "${RCLONE_RETRIES_SLEEP:-5s}" \
      --tpslimit "${RCLONE_TPSLIMIT:-1}" \
      --drive-pacer-min-sleep "${RCLONE_DRIVE_PACER_MIN_SLEEP:-1s}" \
      "${status_args[@]}" \
      --log-file "${LOG_DIR}/rclone_raw_upload.log" \
      --log-level INFO
  )
}

check_raw() {
  (
    cd "${REPO_ROOT}"
    mkdir -p "${LOG_DIR}"
    echo "Verifying raw funding event gzip files at: ${REMOTE}"
    rclone check "${SOURCE_DIR}" "${REMOTE}" \
      --include "${INCLUDE_PATTERN}" \
      --one-way \
      --timeout "${RCLONE_TIMEOUT:-30s}" \
      --contimeout "${RCLONE_CONTIMEOUT:-15s}" \
      --retries "${RCLONE_RETRIES:-5}" \
      --low-level-retries "${RCLONE_LOW_LEVEL_RETRIES:-5}" \
      --retries-sleep "${RCLONE_RETRIES_SLEEP:-5s}" \
      --tpslimit "${RCLONE_TPSLIMIT:-1}" \
      --drive-pacer-min-sleep "${RCLONE_DRIVE_PACER_MIN_SLEEP:-1s}" \
      --log-file "${LOG_DIR}/rclone_raw_check.log" \
      --log-level INFO
    echo "Backup verified."
  )
}

main() {
  local cmd=${1:-backup}
  case "${cmd}" in
    count)
      count_raw
      ;;
    copy)
      require_rclone
      copy_raw
      ;;
    check)
      require_rclone
      check_raw
      ;;
    backup)
      require_rclone
      copy_raw
      check_raw
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
