# Termux Phone Worker Runbook

This repo is prepared for a low-storage two-machine workflow:

- Chromebook: planner, code review, tests, final reports, compact summaries.
- Termux phone worker: monthly chunk generation, temporary raw files, long `tmux` jobs, repo-local caches.

## Before SSH setup

On the phone, install the baseline packages:

```bash
pkg update
pkg install git openssh tmux rsync golang make python rclone
termux-setup-storage
```

Use `$HOME/Github` as the phone-side repo root for all `ak-*` repos:

```bash
mkdir -p ~/Github
cd ~/Github
git clone <repo-url> ak-engine
cd ~/Github/ak-engine
```

Use repo-local Go caches:

```bash
export GOCACHE=$PWD/.cache/go-build
export GOMODCACHE=$PWD/.cache/go-mod
export GOWORK=off
export GOTOOLCHAIN=local
export AK_HISTORIAN_WORKDIR="$HOME/Github/ak-historian/.ak-historian/work"
```

Start long work inside `tmux`:

```bash
tmux new -s ak-engine-109c
```

Current standardized phone layout for agents:

```text
$HOME/Github/ak-engine
$HOME/Github/ak-historian
$HOME/Github/ak-scout
$HOME/Github/ak-trader
```

The repo also ships a low-storage helper:

```bash
scripts/phase10_9c_phone_worker.sh --help
```

## Raw archive backup to Google Drive

The phone is the raw-file holder. The Chromebook should keep code, compact summaries, and final reports, but not raw event JSONL/gzip files.

Configure an rclone Google Drive remote on the phone:

```bash
rclone config
```

Use a normal Drive remote named `gdrive`. For a personal Drive, OAuth in the phone browser is usually the simplest setup. After the remote is configured, test it and create the raw archive folder:

```bash
rclone lsd gdrive:
rclone mkdir gdrive:ak-engine-raw-archive
rclone mkdir gdrive:ak-engine-raw-archive/funding-events
```

Optional encrypted storage is supported by creating an rclone `crypt` remote over Drive:

```text
name: gcrypt
storage: crypt
remote: gdrive:ak-engine-raw-archive-encrypted
filename_encryption: standard
directory_name_encryption: true
```

Store the crypt passwords somewhere durable. Losing them makes the encrypted Drive files unrecoverable.

After a phone worker run has raw funding-event gzip files, copy and verify them with:

```bash
make phone-raw-count
make phone-backup-raw-to-drive
```

The default destination is:

```text
gdrive:ak-engine-raw-archive/funding-events
```

For encrypted Drive storage:

```bash
AK_RAW_REMOTE="gcrypt:funding-events" make phone-backup-raw-to-drive
```

The helper uses `rclone copy` followed by `rclone check --one-way`; it does not move or delete phone-side files. Logs are written to:

```text
runs/reports/rclone_raw_upload.log
runs/reports/rclone_raw_check.log
```

## Local sync helper

After the SSH address or alias is available, either export `PHONE_HOST` inline or create a local env file:

```bash
cp termux_worker.env.example termux_worker.env
```

Then edit `termux_worker.env` with the real SSH target and use:

```bash
make termux-worker-remote-mkdir
make termux-worker-push
make termux-worker-pull-summaries
make termux-worker-pull-reports
```

`make termux-worker-push` now bootstraps or refreshes a real git checkout on the
phone from the local `HEAD` commit before syncing the lightweight working-tree
files. The helper still excludes `.git` from the file mirror itself.

If the local machine has a broken system SSH config snippet and direct helper calls fail with a message like `Bad owner or permissions on /etc/ssh/ssh_config.d/...`, set:

```bash
SSH_CONFIG_FILE=/dev/null
```

in `termux_worker.env` so the helper ignores system SSH config includes.

Inline override still works:

```bash
PHONE_HOST=<user@host-or-ssh-alias> PHONE_REPO='$HOME/Github/ak-engine' SSH_CONFIG_FILE=/dev/null SSH_ARGS='-p 8022' scripts/termux_worker_sync.sh push
```

## Execution rules

- Do not run full multi-symbol raw regeneration on the Chromebook.
- Run one symbol and one month at a time on the phone worker.
- Retain only compact `*-alpha-summary.json` outputs plus final phase reports.
- Back up raw `*-funding-events.jsonl.gz` files from the phone to Drive before deleting raw files.
- Delete heavy raw `*.jsonl`, `*.jsonl.gz`, and `*events*.json` files after summaries are produced and the raw backup has verified.
- Keep final outputs under `runs/reports` or `runs/reports/chunks`, not only in `/tmp`.

## Suggested 10.9C flow

1. Scan coverage on the Chromebook:

```bash
make phase10-9c-coverage
```

This writes:

- `runs/reports/phase10_8c_retained_coverage.md`
- `runs/reports/phase10_8c_retained_coverage.json`
- `runs/reports/phase10_8_ranked_inventory.md`

2. Identify missing symbols from the retained coverage report.
3. Push the repo to the phone worker.
4. Generate one symbol at a time on the phone:

```bash
make phase10-9c-run-symbol SYMBOL=LINKUSDT FROM=2024-01 TO=2025-12
```

This uses the exact repo-supported flag combination for low storage:

- keep event detail long enough for the pipeline's aggregate verification
- delete raw event `jsonl/jsonl.gz` after aggregate via `--summary-only-after-aggregate`
- retain compact `*-alpha-summary.json`, summaries, diagnostics, and phase reports

5. Check for leftover heavy raw files after each symbol:

```bash
make phase10-9c-raw-files
```

6. Count and back up retained raw funding-event gzip files from the phone to Drive when raw retention is needed:

```bash
make phone-raw-count
make phone-backup-raw-to-drive
```

7. Pull only summaries and phase reports back to the Chromebook.
8. Rerun final coverage and ranked inventory on the Chromebook:

```bash
make phase10-9c-coverage
```
