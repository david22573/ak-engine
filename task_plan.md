# Task Plan: Phase 10.12 Second-Generation Funding Candidate Rejection Audit

## Current Phase
Phase 10.12 is complete as a report-only rejection audit. It compares 10.11B, 10.11C, and 10.11D, confirms that all second-generation funding candidates remain rejected after full `192/192` coverage, and recommends closing funding-event research with current data.

## Phase 10.12 Artifacts
- `runs/reports/phase10_12_second_generation_funding_rejection_audit.json`
- `runs/reports/phase10_12_second_generation_funding_rejection_audit.md`

## Phase 10.12 Decision
- Funding-event alpha is dead for the current feature set.
- Do not implement `SqueezeFundingUnwind` yet.
- Phase 11 should pivot away from funding events unless it first adds real new data sources: true OI, liquidations, positioning, or true taker buy/sell volume.
- No candidate promotion, no shadow planning, and no `ak-trader` changes are justified.

## Phase 10.12 Verification
- 10.11B `ConfirmedFundingExtreme`: `192/192`, strongest `ConfirmedNegativeFundingLong|long|240m`, expectancy `-2.014372`, PF `0.966931`, `REJECTED`.
- 10.11C `BreakoutFundingMomentum`: `192/192`, strongest `BreakoutFundingLong|long|240m`, expectancy `-2.943941`, PF `0.953242`, `REJECTED`.
- 10.11D `VolumeImbalanceFundingReversionProxy`: `192/192`, strongest `VolumeImbalanceFundingReversionProxyLong|long|240m`, expectancy `-2.511184`, PF `0.958095`, `REJECTED`.
- 10.11D limitation preserved: `uses TakerBuyRatio fallback only; full taker-buy-sell-volume join not implemented`.
- `ak-trader` status inspected clean.
- Local Chromebook raw funding-event file count under `runs/reports/chunks` is `0`.

# Historical Task Plan: Phase 10.8C / 10.9C Retained-Coverage Recovery

## Goal
Restore full retained compact-summary coverage for the expected symbol universe on the local Chromebook by using the Termux phone as the low-storage batch worker, then verify whether `full_universe_ready=true`.

## Phase 10.11B Active Addendum

### Goal
Complete the ConfirmedFundingExtreme retained-summary evaluation from `132/192` to full `192/192` coverage using the Termux phone worker for only the missing confirmed-family symbol-months, then pull compact reports/summaries back locally.

### Scope and Boundaries
- Work only in `ak-engine`; do not modify `ak-trader`.
- Regenerate only confirmed-family missing months:
  - `AVAXUSDT` `2024-01 .. 2025-12`
  - `SOLUSDT` `2024-01 .. 2025-12`
  - `LINKUSDT` `2025-01 .. 2025-12`
- Use the phone worker for heavy regeneration; do not fetch new data if phone primary candles already exist.
- Do not use R2 restore/download in this phase.
- Pull compact `*-alpha-summary.json`, phase10_11b reports, and closeout logs only; do not pull raw JSONL/GZIP funding-event files.
- Do not implement additional candidate families or tune thresholds.

### Phase 10.11B Steps
- [x] Verify phone repo/process/data/disk/test state.
- [x] Regenerate missing confirmed summaries on phone, one symbol at a time: `AVAXUSDT`, `SOLUSDT`, `LINKUSDT`.
- [x] Pull compact summaries/reports/logs only back to Chromebook.
- [x] Rerun local 10.11B ConfirmedFundingExtreme evaluation and coverage gap reports from retained summaries.
- [x] Run required local Go tests.
- [x] Close out with coverage, labels, artifacts, commands, and boundary confirmations.

### Phase 10.11B Starting State
- `coverage_before=132/192`
- `missing_before`: `AVAXUSDT 2024-01..2025-12`, `SOLUSDT 2024-01..2025-12`, `LINKUSDT 2025-01..2025-12`
- Available-coverage labels:
  - `ConfirmedNegativeFundingLong|long|240m`: `REJECTED`, expectancy `-1.477516`, PF `0.974044`
  - `ConfirmedPositiveFundingShort|short|5m`: `REJECTED`, expectancy `-5.162780`, PF `0.527383`

### Phase 10.11B Current State
- `coverage_after=192/192`
- `missing_after=[]`
- `full_evaluation_complete=true`
- Exact blocker: none.
- Phone-side regeneration completed for the original missing 60 symbol-months:
  - `AVAXUSDT 2024-01..2025-12`
  - `SOLUSDT 2024-01..2025-12`
  - `LINKUSDT 2025-01..2025-12`
- Repair note: an initial broad summary pull overwrote some local confirmed summaries with stale phone copies. To restore full confirmed coverage, the phone regenerated `ADAUSDT`, `BNBUSDT`, `DOGEUSDT`, and `ETHUSDT` for `2024-01..2025-12`, plus `LINKUSDT 2024-01..2024-12`, then only those repaired alpha summaries were pulled back.
- No R2 restore was used.
- No new data fetch was performed.
- Local and touched phone-side raw funding-event file counts are `0`.
- Required local tests `go test ./internal/app` and `go test ./...` passed.
- Final labels:
  - `ConfirmedNegativeFundingLong|long|240m`: `REJECTED`, expectancy `-2.014372`, PF `0.966931`
  - `ConfirmedPositiveFundingShort|short|5m`: `REJECTED`, expectancy `-5.164622`, PF `0.547049`

## Constraints
- Work only in `ak-engine`.
- Treat the Chromebook as planner/reviewer plus final artifact holder.
- Treat the Termux phone as the heavy worker for monthly chunk generation, temporary raw event files, caches, and long tmux jobs.
- Run one symbol at a time across monthly chunks; do not run full-universe raw regeneration in one job.
- Preserve compact `*-alpha-summary.json` outputs and final phase reports.
- Delete heavy raw `*.jsonl`, `*.jsonl.gz`, and `*events*.json` files only after aggregate reports are written.
- Keep final artifacts under `runs/reports` or `runs/reports/chunks`.
- Use repo-local Go caches when running locally:
  - `GOCACHE=$PWD/.cache/go-build`
  - `GOMODCACHE=$PWD/.cache/go-mod`
  - `GOWORK=off`
  - `GOTOOLCHAIN=local`
- Use serial pull/validate/coverage cycles; never overlap summary pulls with the coverage scan.
- Verify state from artifacts and reports, not intent or stale manifest timestamps.

## Current Phase
Step 4 closeout complete for local retained coverage and inventory: the all-8-symbol retained scan refreshed Phase 10.8C coverage, Phase 10.9C recovery coverage copies, and the unfiltered retained ranked inventory with `full_universe_ready=true`, `raw_required=false`, `summary files found=192`, and `malformed summaries=0`. The phone worker is idle, the remote phone `ak-engine` tree is now git-backed, and remote raw gzip cleanup was intentionally not performed; aggregate-report reconciliation remains future work outside this closeout.

## Current Verified State
- Local retained coverage report date: `2026-06-30`
- Coverage status: `full_universe_ready`
- `full_universe_ready=true`
- Summary files found: `192` (includes XRPUSDT, which is not in the coverage target set of the script)
- Malformed summaries: `0`
- Fully retained locally: `ADAUSDT`, `AVAXUSDT`, `BNBUSDT`, `DOGEUSDT`, `ETHUSDT`, `LINKUSDT`, `SOLUSDT`, `XRPUSDT`
- Remaining missing symbols: none
- Remote phone worker state: idle
- Remote phone repo state: git-backed checkout, not merely a non-git synced copy
- `AVAXUSDT` is no longer a blocker; it is fully retained locally through `2025-12`
- `ETHUSDT`, `BNBUSDT`, `ADAUSDT`, and `DOGEUSDT` compact summaries are complete locally, but their phone-side aggregate reports were not visible during the latest remote report check, so do not delete their raw gzip files until the aggregate/report state is reconciled.

## Phases

### Step 0: Restore and Verify Planning Context
- [x] Read the active pasted runbook/context attachment.
- [x] Read `task_plan.md`, `findings.md`, and `progress.md`.
- [x] Re-run or re-read the authoritative local retained-coverage report.
- **Status:** complete

### Step 1: Harden and Verify the Two-Machine Workflow
- [x] Add the phone sync helper, runbook, and env template.
- [x] Verify helper-backed push/pull paths with the real phone target.
- [x] Add the `coverage-only` retained scan path.
- [x] Record the SSH config workaround with `SSH_CONFIG_FILE=/dev/null`.
- **Status:** complete

### Step 2: Recover Known Missing Symbols Already Completed
- [x] Complete and pull `LINKUSDT`.
- [x] Complete and pull `SOLUSDT`.
- [x] Re-run local retained coverage after each clean serial pull.
- **Status:** complete

### Step 3: Complete the Remaining Missing Symbols
- [x] Verify current remote worker/session state before starting new work.
- [x] Run and pull `ETHUSDT`. Current state: local retained summaries exist for all `2024-01 .. 2025-12`; `jq` validation passed for all 24 local `ETHUSDT` alpha summaries; retained coverage now shows `summary files found=96`, `malformed summaries=0`.
- [x] Run and pull `BNBUSDT`. Current state: all `24` local BNB alpha summaries validate cleanly for `2024-01 .. 2025-12`; retained coverage now shows `summary files found=120`, `malformed summaries=0`.
- [x] Run and pull `AVAXUSDT`. Current state: fully completed and pulled locally for `2024-01 .. 2025-12` (24 summaries, no malformed).
- [x] Run and pull `ADAUSDT`. Current state: all 24 local alpha summaries validate and retained coverage includes all expected months.
- [x] Run and pull `DOGEUSDT`. Current state: all 24 local alpha summaries validate and retained coverage includes all expected months.
- [x] Re-run local retained coverage after each completed symbol.
- [ ] Verify raw event cleanup after each aggregate completes.
- **Status:** complete for retained compact coverage; cleanup verification remains tracked in Step 4

### Step 4: Final Full-Universe Verification
- [x] Re-run retained coverage with all expected symbols present.
- [x] Re-run ranked inventory and inspect `full_universe_ready`.
- [x] Confirm zero malformed summaries.
- [ ] Reconcile remote aggregate reports before raw-file cleanup. Deferred by request; do not clean remote raw gzip files yet.
- [ ] Confirm no accidental raw-file retention after cleanup is safe. Deferred by request.
- [x] Record the final local authoritative state in planning files.
- **Status:** complete for final local closeout; remote cleanup deferred

## Decisions Made
| Decision | Rationale |
|----------|-----------|
| Use the phone as the heavy worker and the Chromebook as the final artifact holder | The local machine is storage-constrained and unsuitable for full raw regeneration. |
| Pull summaries serially, then validate, then scan coverage | Concurrent pull+scan caused transient malformed-summary warnings. |
| Trust direct artifact inspection over stale pipeline manifests | The live phone pipeline can outpace manifest/report timestamps. |

## Errors Encountered
| Step | Error | Resolution |
|------|-------|------------|
| Step 1 | Local helper-backed pulls failed on a broken system SSH config include. | Added `SSH_CONFIG_FILE` support and verified `SSH_CONFIG_FILE=/dev/null` works. |
| Step 2 | `LINKUSDT` helper run originally stopped after one month because `--max-months 1` leaked into the wrapper. | Removed the unintended limit and re-synced the helper. |
| Step 2 | First `SOLUSDT` worker attempt failed because direct candle parquet files were missing from the path consumed by `build-features`. | Backfilled `SOLUSDT` monthly candles with phone-side `ak-historian fetch`, verified `build-features`, and relaunched the worker. |
| Step 3 | Phone SSH on `192.168.1.79:8022` returned `Connection refused` during `ETHUSDT` continuation checks and `termux-worker-pull-summaries`. | Local validation and coverage were rerun successfully; wait for phone SSH to come back before further `ETHUSDT` pull/resume work. |
| Step 3 | `ETHUSDT` has all compact summaries but no phone-side `phase10_9c_ETHUSDT_pipeline.md`; 24 raw gzip event files remain. | Kept raw files instead of deleting them; aggregate/report state must be reconciled before cleanup. |
| Step 3 | Phone IP changed from `192.168.1.79` to `192.168.1.81`; first helper pull failed on host-key verification. | Updated `termux_worker.env` to `PHONE_HOST=davidmiguel22573@192.168.1.81` and repo-local known-hosts args; pull succeeded and accepted `[192.168.1.81]:8022`. |
| Step 3 | Phone IP reverted from `192.168.1.81` to `192.168.1.79`; `.81` became unreachable with `No route to host`. | Updated `termux_worker.env` back to `.79`, recreated `ak-phone-ssh`, verified remote BNB still had 15 summaries and no active worker, then launched `ak-engine-109c-bnb-resume` for `2025-04 .. 2025-12`. |
| Step 3 | Latest `.81` fallback probe timed out, while `.79` succeeded and showed `remote_bnb_count=24`. | Treat `.79:8022` as the current phone SSH target; pulled BNB summaries/reports, validated local JSON, reran coverage, and left BNB/ETH raw cleanup blocked pending aggregate-report reconciliation. |
| Step 3 | Phone SSH target `192.168.1.79:8022` returned `Connection refused` when starting the AVAX pull/verify turn. | Had user restart `sshd` inside Termux; reconnected successfully, pulled all 24 completed summaries, and started `ADAUSDT` worker after fetching missing candles. |
