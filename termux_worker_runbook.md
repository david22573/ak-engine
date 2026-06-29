# Termux Phone Worker Runbook

This repo is prepared for a low-storage two-machine workflow:

- Chromebook: planner, code review, tests, final reports, compact summaries.
- Termux phone worker: monthly chunk generation, temporary raw files, long `tmux` jobs, repo-local caches.

## Before SSH setup

On the phone, install the baseline packages:

```bash
pkg update
pkg install git openssh tmux rsync golang make python
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

Inline override still works:

```bash
PHONE_HOST=<user@host-or-ssh-alias> PHONE_REPO='$HOME/Github/ak-engine' SSH_ARGS='-p 8022' scripts/termux_worker_sync.sh push
```

## Execution rules

- Do not run full multi-symbol raw regeneration on the Chromebook.
- Run one symbol and one month at a time on the phone worker.
- Retain only compact `*-alpha-summary.json` outputs plus final phase reports.
- Delete heavy raw `*.jsonl`, `*.jsonl.gz`, and `*events*.json` files after summaries are produced.
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

6. Pull only summaries and phase reports back to the Chromebook.
7. Rerun final coverage and ranked inventory on the Chromebook:

```bash
make phase10-9c-coverage
```
