# Progress Log

## Session: 2026-06-28

### Current Status
- Active task: prepare low-storage/two-machine execution context before SSH handoff.
- Current phase: standby for phone SSH address after recording workspace constraints.

### Actions Taken
- Read pasted setup guidance from `/home/davidmiguel22573/.codex/attachments/fda79e97-8fe0-4c41-8ca0-cb23341f9411/pasted-text-1.txt`.
- Captured and persisted the important execution constraints for future work:
  - Chromebook is the control/review machine and final report holder.
  - Termux phone is the heavy worker for monthly chunk generation, raw temporary files, caches, and long tmux jobs.
  - Run one symbol and one month at a time; do not run full 8-symbol x 24-month raw regeneration in one job.
  - Keep compact summaries and final reports; delete heavy raw `*.jsonl`, `*.jsonl.gz`, and `*events*.json` outputs after summary generation.
  - Keep final artifacts in `runs/reports` or `runs/reports/chunks`, not only in `/tmp`.
  - Prefer repo-local Go caches: `GOCACHE=$PWD/.cache/go-build`, `GOMODCACHE=$PWD/.cache/go-mod`, `GOWORK=off`, `GOTOOLCHAIN=local`.
- Checked repo context files and used the existing `task_plan.md` / `findings.md` / `progress.md` convention for persistence.
- Added `scripts/termux_worker_sync.sh` as a local push/pull helper for the future SSH target.
- Added `termux_worker_runbook.md` with the prepared two-machine workflow, Termux bootstrap commands, repo-local cache env, and low-storage execution rules.
- Added `termux_worker.env.example` so the future SSH target can be filled in once and reused by the sync helper.
- Tightened `pull-reports` to use `rsync` include/exclude filters instead of brittle shell glob patterns.
- Added `termux_worker.env` to `.gitignore` so future SSH host details stay local.
- Added `make` targets for remote mkdir, push, pull-summaries, and pull-reports as thin wrappers around the sync helper.
- Installed the local public key on the phone with `ssh-copy-id`, enabling passwordless SSH to `192.168.1.79:8022`.
- Installed missing `rsync` on the Termux phone worker with `pkg install rsync`.
- Created the local ignored `termux_worker.env` for the phone target and corrected it to use remote-safe path expansion via `$HOME/ak-engine`.
- Refactored `scripts/termux_worker_sync.sh` to use `SSH_ARGS`, support backward-compatible `RSYNC_RSH`, and fall back to `tar` over SSH when local `rsync` is unavailable.
- Verified the helper-backed `make termux-worker-remote-mkdir` and `make termux-worker-push` path against the real phone target.
- Standardized the phone repo root to `$HOME/Github`, moved `ak-engine` there, and recreated remote tmux session `ak-engine-109c` with working directory `$HOME/Github/ak-engine`.
- Synced all local sibling `ak-*` repos to the phone: `ak-engine`, `ak-historian`, `ak-scout`, and `ak-trader`.
- Updated the example env file and runbook to use `$HOME/Github/ak-engine` as the default phone-side target so other agents inherit the same layout.
- Added `scripts/phase10_9c_phone_worker.sh` to codify the exact retained-coverage scan, one-symbol funding regeneration, and raw-file audit commands for low-storage runs.
- Updated the `Makefile` with `phase10-9c-*` targets so future agents can run the supported coverage and one-symbol workflows without reconstructing flags.
- Corrected the sync helper default remote repo path to `$HOME/Github/ak-engine` so the script matches the standardized phone layout.
- Clarified in the runbook and findings that raw event files must survive until aggregate verification completes, then be deleted via `--summary-only-after-aggregate`.
- Added `--coverage-only` support to `analyze-compact-robustness` so the 10.9C retained-coverage scan works even when no target `NegativeFundingLong/long` candidate exists.
- Ran the local retained coverage scan and confirmed the current Chromebook state is `single_symbol_only`: only `XRPUSDT` is retained, and `ADAUSDT, AVAXUSDT, BNBUSDT, DOGEUSDT, ETHUSDT, LINKUSDT, SOLUSDT` are missing.
- Performed the Chromebook storage cleanup from the runbook by deleting local raw/event files under `runs/` and removing `.cache/go-build`, improving free space from `2.1G` to `4.2G`.
- Verified on the phone worker that each missing symbol (`ADAUSDT`, `AVAXUSDT`, `BNBUSDT`, `DOGEUSDT`, `ETHUSDT`, `LINKUSDT`, `SOLUSDT`) has `36` monthly funding-rate parquet files under `$HOME/Github/ak-historian/.ak-historian/work`.
- Started a live phone-worker `LINKUSDT` run, found that the helper incorrectly passed `--max-months 1`, fixed the helper, re-synced the repo, then diagnosed the next blocker from the manifest: missing Termux `duckdb`.
- Installed Termux `duckdb` (`v1.5.4`) on the phone worker, relaunched the `LINKUSDT` run, and confirmed live progress through the first three months with cleanup functioning as intended:
  - `2024-01` PASS, `event_rows=78278`, `bytes_freed=141255143`
  - `2024-02` PASS, `event_rows=56388`, `bytes_freed=132447708`
  - `2024-03` PASS, `event_rows=30158`, `bytes_freed=142844460`
  - `2024-04` classification had started at last poll

### Verification Results
| Step | Command | Status | Notes |
|------|---------|--------|-------|
| Context Capture | `sed -n '1,220p' /home/davidmiguel22573/.codex/attachments/fda79e97-8fe0-4c41-8ca0-cb23341f9411/pasted-text-1.txt` | pass | Parsed low-storage Chromebook/Termux execution guidance. |
| Context Location | `rg --files -g 'CLAUDE.md' -g 'README*' -g '*memory*' -g '*context*' -g '.agents/**' -g '.codex/**'` | pass | Repo already uses `task_plan.md`, `findings.md`, and `progress.md` for persistent context. |
| Sync Helper | `scripts/termux_worker_sync.sh --help` | pass | Printed prepared push/pull helper usage for the future Termux SSH target. |
| Sync Helper | `scripts/termux_worker_sync.sh push` | pass | Failed fast with `PHONE_HOST is required`, confirming no ambiguous default remote target. |
| Sync Helper | `bash -n scripts/termux_worker_sync.sh` | pass | Script syntax verified after env-file support and `rsync` filter changes. |
| Sync Helper | `ENV_FILE=/tmp/does-not-exist scripts/termux_worker_sync.sh pull-reports` | pass | Missing env file is tolerated; command still fails fast on missing `PHONE_HOST` instead of misrouting. |
| Make Wrapper | `make termux-worker-push` | pass | Wrapper reaches the sync helper and fails fast on missing `PHONE_HOST`, as intended before SSH details exist. |
| Git Ignore | `git check-ignore -v termux_worker.env` | pass | `termux_worker.env` is ignored by `.gitignore`, keeping future SSH host details local. |
| SSH Bootstrap | `ssh-copy-id -p 8022 192.168.1.79` | pass | Local ED25519 key installed on the phone; future SSH no longer needs the password. |
| Remote Packages | `ssh -p 8022 192.168.1.79 'yes | pkg install rsync'` | pass | Installed missing `rsync` on the Termux phone worker. |
| Helper Workflow | `make termux-worker-remote-mkdir` | pass | Helper-backed remote mkdir succeeded once `PHONE_REPO` used remote-safe `$HOME/ak-engine`. |
| Helper Workflow | `make termux-worker-push` | pass | Helper-backed push succeeded using the `tar` fallback because this Chromebook lacks local `rsync`. |
| Remote Verify | `ssh -p 8022 192.168.1.79 'cd ~/ak-engine && pwd && ls -1 | sed -n "1,20p"'` | pass | Repo snapshot is present under `/data/data/com.termux/files/home/ak-engine`. |
| Remote Verify | `ssh -p 8022 192.168.1.79 'for x in git rsync tmux go make python; do ...; done'` | pass | Phone worker has the expected toolchain installed. |
| Remote Verify | `ssh -p 8022 192.168.1.79 'tmux list-sessions'` | pass | Session `ak-engine-109c` exists for long-running worker jobs. |
| Remote Layout | `ssh -p 8022 192.168.1.79 'find "$HOME/Github" -maxdepth 1 -mindepth 1 -type d -name "ak-*" | sort'` | pass | Phone-side repo root is standardized under `$HOME/Github` with `ak-engine`, `ak-historian`, `ak-scout`, and `ak-trader`. |
| Remote Layout | `ssh -p 8022 192.168.1.79 'tmux kill-session -t ak-engine-109c ...; tmux new-session -d -s ak-engine-109c -c "$HOME/Github/ak-engine"'` | pass | Standard tmux session now starts in `$HOME/Github/ak-engine`. |
| 10.9C Coverage | `env GOCACHE=$PWD/.cache/go-build GOMODCACHE=$PWD/.cache/go-mod GOWORK=off GOTOOLCHAIN=local ./scripts/phase10_9c_phone_worker.sh coverage` | pass | Coverage-only path now emits retained coverage/inventory even without a target candidate. |
| 10.9C Coverage | `sed -n '1,220p' runs/reports/phase10_8c_retained_coverage.md` | pass | Current retained state is `single_symbol_only`; only `XRPUSDT` is retained locally. |
| Storage Cleanup | `find runs -type f \( -name '*.jsonl' -o -name '*.jsonl.gz' -o -name '*events*.json' \) -delete` | pass | Chromebook raw/event files removed. |
| Storage Cleanup | `rm -rf .cache/go-build` | pass | Chromebook Go build cache removed. |
| Storage Cleanup | `df -h` / `du -sh runs .cache testdata` | pass | Free space improved from `2.1G` to `4.2G`; `runs` reduced to `6.1G`, `.cache` to `686M`. |
| Worker Data | `ssh -p 8022 ... '... for s in ADAUSDT AVAXUSDT BNBUSDT DOGEUSDT ETHUSDT LINKUSDT SOLUSDT; do ...'` | pass | Each missing symbol currently has `36` monthly funding-rate parquet files on the phone worker. |
| Worker Sync | `make termux-worker-push` | pass | Updated repo synced to the phone worker. |
| Worker Bug | `ssh -p 8022 ... 'sed -n "1,220p" runs/reports/phase10_9c_LINKUSDT_pipeline.md'` | fail recorded | First live LINK run blocked because helper incorrectly limited the run to one month with `--max-months 1`. |
| Worker Bugfix | `bash -n scripts/phase10_9c_phone_worker.sh` | pass | Helper syntax verified after removing `--max-months 1`. |
| Worker Dependency | `ssh -p 8022 ... 'yes | pkg install duckdb'` | pass | Installed Termux `duckdb` after the manifest showed parquet derivative reads were blocked without it. |
| Worker Live Run | `ssh -p 8022 ... 'sed -n "1,260p" runs/manifests/phase10_7e_funding_event_manifest.json'` | pass in progress | Relaunched LINK run now shows `2024-01`, `2024-02`, and `2024-03` PASS with cleanup, and `2024-04` has started. |

## Session: 2026-06-20

### Current Status
- Active task: ak-engine research-script debt refactor.
- Current phase: Step 6.
- Existing dirty worktree detected and preserved.

### Actions Taken
- Read `caveman` and `planning-with-files` skill instructions.
- Restored local `ak-engine` plan context.
- Ran planning session catchup; no catchup output.
- Captured `git status --short` before edits.
- Replaced generated-report reads in app tests with `internal/app/testdata` fixtures.
- Added fixture reports for Phase 10.0 candidate spec and Phase 10.4 guardrails/branch closure.
- Restored accidentally edited workspace-root planning files to prior contents.
- Added `atomicWriteFile` helper using same-directory temp file plus rename.
- Swapped manifest/report writes in phase10 low-resource prep and funding event pipeline to atomic writes.
- Added historian workdir helper with `--workdir` > `AK_HISTORIAN_WORKDIR` > `.ak-historian/work` precedence.
- Reused shared funding-rate derivative path helper in phase10 prep/pipeline and audit code.
- Updated source shell/Python scripts to use `AK_HISTORIAN_WORKDIR` with `.ak-historian/work` default.
- Replaced placeholder V2 audit PASS checks with PASS/FAIL/UNKNOWN logic.
- Added V2 integrity tests covering PASS, FAIL, and UNKNOWN check states.
- Adjusted `audit-v2-integrity --fail-on-error` to exit nonzero only for FAIL, not UNKNOWN.
- Removed unused `"runtime"` import in `phase10_funding_event_pipeline.go`.
- Extracted and moved reporting/audit logic (`Phase10FundingContextAudit`, `writePhase10FundingContextAudit`, and helper functions) from `phase10_funding_event_pipeline.go` to `phase10_funding_event_reports.go`, cleaning up unused imports.
- Formatted Go files and successfully completed all 4 steps of Step 5 verification.
- Verified and tightened Fast Accumulation side-specific scoring so long-only cost/chop overrides no longer loosen or tighten short decisions and short-only overrides no longer affect long decisions.
- Added explicit asymmetric tests for strict-short/lenient-long and strict-long/lenient-short cost and chop gating.
- Verified exit slippage behavior: TP skips adverse exit slippage; SL, strategy close, time stop, and end-of-data retain conservative exit slippage.
- Added explicit trim-survival tests for completed-window retention, hourly context, breakout-retest logic, decisions snapshot, and UTC day entry counting.
- Regenerated a deterministic local strict backtest smoke report from `testdata/candles/btc_5m_fast_accumulation_sample.json`.
- Prepared final closeout notes covering files changed, exact bug fixed, test coverage, and historical-impact limits.

### Verification Results
| Step | Command | Status | Notes |
|------|---------|--------|-------|
| Step 0 | `go test ./... -run '^$'` | pass | All packages compiled; no tests run. |
| Step 0 | `go test ./internal/backtest ./internal/data ./internal/features ./internal/regime ./internal/research ./internal/strategy ./internal/walkforward` | pass | Core package tests passed. |
| Step 0 | `go test ./... 2>&1 | tee runs/refactor_baseline_go_test.txt || true` | recorded fail | `internal/app` missing generated reports. |
| Step 0 | `go build -o ./ak-engine ./cmd/ak-engine` | pass | Build succeeded. |
| Step 0 | `./ak-engine version` | pass | Output: `ak-engine v0.1.0`. |
| Step 0 | `python3 -m py_compile ...` | pass | Listed research scripts compiled. |
| Step 0 | `bash -n ...` | pass | Listed shell scripts parsed. |
| Step 1 | `go test ./... -run '^$'` | pass | Compile-only verification passed. |
| Step 1 | `go test ./internal/app` | pass | App tests passed. |
| Step 1 | `go test ./...` | pass | Full test suite passed after fixture refactor. |
| Step 1 | `go build -o ./ak-engine ./cmd/ak-engine` | pass | Build succeeded. |
| Step 2 | `go test ./... -run '^$'` | pass | Compile-only verification passed. |
| Step 2 | `go test ./internal/app` | pass | App tests passed. |
| Step 2 | `go build -o ./ak-engine ./cmd/ak-engine` | pass | Build succeeded. |
| Step 2 | `python3 - <<'PY' ... manifest json parse ... PY` | pass | Output: `manifest json parse OK`. |
| Step 3 | `go test ./... -run '^$'` | pass | Compile-only verification passed. |
| Step 3 | `go test ./internal/app` | pass | App tests passed. |
| Step 3 | `go build -o ./ak-engine ./cmd/ak-engine` | pass | Build succeeded. |
| Step 3 | `./ak-engine phase10-funding-event-pipeline --help` | pass | Help shows `--workdir` default `.ak-historian/work`. |
| Step 3 | `grep -R "../ak-historian/.ak-historian/work" -n . || true` | diagnostic pass | Remaining matches are historical generated artifacts/cache/planning data, not source Go/shell/Python script path contracts. |
| Step 4 | `go test ./... -run '^$'` | pass | Compile-only verification passed. |
| Step 4 | `go test ./internal/research` | pass | Research package tests passed. |
| Step 4 | `go test ./internal/app -run 'Test.*Leakage|Test.*Funding|TestAnalyzeCandidateDecay'` | pass | Targeted app tests passed. |
| Step 4 | `go build -o ./ak-engine ./cmd/ak-engine` | pass | Build succeeded. |
| Step 4 | `./ak-engine audit-v2-integrity --path runs/reports/chunks --fail-on-error || true` | diagnostic pass | Output status `UNKNOWN`; `v2_no_stale_10_5_artifacts` is UNKNOWN because schema has no source phase field. |
| Step 5 | `go test ./... -run '^$'` | pass | Compile-only verification passed after removing unused imports. |
| Step 5 | `go test ./internal/app -run 'Test.*FundingEvent|TestAnalyzeCandidateDecay'` | pass | Targeted app tests passed. |
| Step 5 | `go build -o ./ak-engine ./cmd/ak-engine` | pass | Build succeeded. |
| Step 5 | `./ak-engine phase10-funding-event-pipeline --help` | pass | Help output printed successfully. |
| Verification Closeout | `GOWORK=off GOTOOLCHAIN=local go test ./...` | blocked by sandbox cache | Host Go build cache under `~/.cache/go-build` was read-only in this sandbox. |
| Verification Closeout | `env GOCACHE=/home/davidmiguel22573/Github/ak-engine/.cache/go-build GOMODCACHE=/home/davidmiguel22573/Github/ak-engine/.cache/go-mod GOWORK=off GOTOOLCHAIN=local go test ./...` | pass | Full suite passed with workspace-local Go caches. |
| Verification Closeout | `env GOCACHE=/home/davidmiguel22573/Github/ak-engine/.cache/go-build GOMODCACHE=/home/davidmiguel22573/Github/ak-engine/.cache/go-mod GOWORK=off GOTOOLCHAIN=local go test ./internal/strategy` | pass | Post-edit targeted rerun passed after adding trim-specific strategy tests. |
| Verification Closeout | `env GOCACHE=/home/davidmiguel22573/Github/ak-engine/.cache/go-build GOMODCACHE=/home/davidmiguel22573/Github/ak-engine/.cache/go-mod GOWORK=off GOTOOLCHAIN=local go run ./cmd/ak-engine backtest --source local-json --path testdata/candles/btc_5m_fast_accumulation_sample.json --market futures-um --symbol BTCUSDT --interval 5m --from 2024-01-01 --to 2024-01-02 --strategy fast_accumulation_strict --format json` | pass | Deterministic local smoke regenerated; sample produced one stop-loss trade, not a TP exit. |

### Errors
| Step | Error | Resolution |
|------|-------|------------|
| Step 0 | `TestPhase104BranchClosureReportIncludesRejectedFragileCandidates`, `TestPhase104GuardrailChecklistIncludesConcentrationAndOOSGates`, and `TestPhase10CandidateSpec` failed due missing `../../runs/reports` files. | Baseline captured; target for Step 1. |
| Step 1 | `apply_patch` failed because paths were relative to `/home/davidmiguel22573/Github`, not `ak-engine`. | Re-applied patch using `ak-engine/...` paths. |
| Step 1 | Planning update first wrote to workspace root planning files. | Restored root planning files to prior contents and rewrote plan in `ak-engine`. |
| Step 3 | Grep verification crawled large generated `runs` logs/cache and printed many historical path strings. | Source references were removed; generated historical artifacts retained. |
| Step 4 | Initial V2 aggregate check failed because it assumed `gross_loss_bps` must be negative. | Verified generated schema stores gross loss as positive magnitude; removed sign assumption. |
| Step 5 | `go test ./... -run '^$'` failed because `internal/app/phase10_funding_event_pipeline.go` still imported unused `"runtime"` (and other reporting packages). | Removed unused `"runtime"` import. Moved all report rendering and context audit functions from `phase10_funding_event_pipeline.go` to `phase10_funding_event_reports.go`, removing unused imports. |
| Verification Closeout | Requested exact `go test` command could not write to host Go cache in sandbox. | Re-ran with workspace-local `GOCACHE` and `GOMODCACHE`; same package set passed. |
