# Progress Log

## Session: 2026-07-04 Phase 11.1 CompressionVolumeBreakout Evaluation

## Actions Taken
- Committed the Phase 11.0 planning/report closeout as `dd998ec`.
- Implemented one research-only candidate family: `CompressionVolumeBreakout`.
- Ran the full 8-symbol x 24-month local evaluation using existing local historian data only.
- Created the Phase 11.1 reports:
  - `runs/reports/phase11_1_compression_volume_breakout.json`
  - `runs/reports/phase11_1_compression_volume_breakout.md`
- Updated planning files with the Phase 11.1 rejection result.

## Evidence
- Coverage: `192/192` symbol-months.
- Raw event detail retained: `false`.
- Verdict counts: `rejected=6`, `fragile=0`, `research_lead=0`.
- Best PF row: `CompressionVolumeBreakout|long|240m`, PF `0.877362`, expectancy `-8.200320` bps after 5 bps.
- `go test ./internal/app` passed.
- JSON validated with `jq`.

## Boundaries Preserved
- No `RegimeTrendPullbackContinuation` implementation.
- No funding-primary trigger.
- No data fetch.
- No `ak-trader` modification.
- No promotion, shadow plan, live trading, order placement, exchange key, execution, or mainnet logic.

## Session: 2026-07-04 Phase 11.0 Non-Funding Candidate Design

## Actions Taken
- Read the Phase 11.0 objective attachment.
- Inspected current feature, regime, candle, research-join, and candidate-evaluator code surfaces.
- Created the Phase 11.0 design reports:
  - `runs/reports/phase11_0_non_funding_candidate_design.json`
  - `runs/reports/phase11_0_non_funding_candidate_design.md`
- Updated planning files with the Phase 11.0 design findings and recommendation.

## Current Recommendation
- Next phase: `Phase 11.1 - Implement Top Non-Funding Candidate Family`.
- Implement first: `CompressionVolumeBreakout`.
- Second candidate: `RegimeTrendPullbackContinuation`.
- Blocked/deferred: `BasketDispersionLeadLag`.

## Boundaries Preserved
- No candidate code implemented.
- No `ak-trader` modification or promotion.
- No live trading, order placement, exchange key, execution, or mainnet logic.
- No new data fetch.
- Funding is not used as the primary trigger in proposed Phase 11 candidates.

## Session: 2026-07-04 Phase 10.12 Second-Generation Funding Rejection Audit

## Actions Taken
- Read the pasted 10.11D closeout objective.
- Verified current `ak-engine` artifacts for 10.11B, 10.11C, and 10.11D.
- Added the Phase 10.12 rejection audit report:
  - `runs/reports/phase10_12_second_generation_funding_rejection_audit.json`
  - `runs/reports/phase10_12_second_generation_funding_rejection_audit.md`
- Updated `task_plan.md` with the Phase 10.12 closeout decision.

## Evidence
- 10.11B `ConfirmedFundingExtreme`: full `192/192`, strongest candidate `ConfirmedNegativeFundingLong|long|240m`, expectancy `-2.014372`, PF `0.966931`, rejected.
- 10.11C `BreakoutFundingMomentum`: full `192/192`, strongest candidate `BreakoutFundingLong|long|240m`, expectancy `-2.943941`, PF `0.953242`, rejected.
- 10.11D `VolumeImbalanceFundingReversionProxy`: full `192/192`, strongest candidate `VolumeImbalanceFundingReversionProxyLong|long|240m`, expectancy `-2.511184`, PF `0.958095`, rejected.
- 10.11D limitation explicitly retained: it uses only `TakerBuyRatio` fallback, not a true taker buy/sell volume join.
- `git -C /home/davidmiguel22573/Github/ak-trader status --short` produced no output.
- Chromebook raw funding-event file count under `runs/reports/chunks` is `0`.

## Decision
- Close funding-event research with the current data set.
- Do not implement `SqueezeFundingUnwind` until true OI/liquidation/positioning or true taker buy/sell data exists.
- Phase 11 should pivot away from funding events unless it starts with new data-source work.
- No promotion, no shadow plan, and no `ak-trader` change.

## Session: 2026-06-30

### Final Local + SSH Status Check
- Check time: `2026-06-30 23:01 PDT`.
- Restored planning context from `task_plan.md`, `findings.md`, and `progress.md`.
- Local repo state before planning-file update:
  - branch: `main`
  - changed files: `findings.md`, `progress.md`, `task_plan.md`, and `internal/app/funding_aggregation.go`
- Local retained alpha-summary artifacts now contain `24` months each for:
  - `ADAUSDT`, `AVAXUSDT`, `BNBUSDT`, `DOGEUSDT`, `ETHUSDT`, `LINKUSDT`, `SOLUSDT`, and `XRPUSDT`
- Validated all local retained alpha-summary JSON files with `jq`; all passed.
- Reran local retained coverage with `make phase10-9c-coverage`.
- Refreshed authoritative local coverage:
  - coverage status: `full_universe_ready`
  - `full_universe_ready=true`
  - `summary files found=192`
  - `malformed summaries=0`
  - missing expected symbols: none
- Local tmux check found no tmux server/session.
- Direct SSH to `davidmiguel22573@192.168.1.79:8022` succeeded.
- Remote phone status:
  - remote date: `Tue Jun 30 23:01:34 PDT 2026`
  - no remote tmux sessions listed
  - no active `phase10_9c_phone_worker`, `phase10-funding-event-pipeline`, or `go run ./cmd/ak-engine` process found except the probe command itself
  - remote phone `ak-engine` repo is git-backed, not merely a non-git synced copy
  - remote compact summaries: `24` months each for `ADAUSDT`, `AVAXUSDT`, `BNBUSDT`, `DOGEUSDT`, `ETHUSDT`, `LINKUSDT`, and `SOLUSDT`
  - remote raw gzip leftovers: `24` each for `ADAUSDT`, `AVAXUSDT`, `BNBUSDT`, `DOGEUSDT`, and `ETHUSDT`
  - visible remote aggregate reports: `phase10_9c_AVAXUSDT_pipeline.md`, `phase10_9c_LINKUSDT_pipeline.md`, and `phase10_9c_SOLUSDT_pipeline.md`
- Updated planning files so the current handoff is cleanup/report reconciliation, not more retained-summary generation.

### Phase 10.9C Final Closeout
- Check time: `2026-06-30 23:26 PDT`.
- Reran the unfiltered retained ranked inventory across all local retained summaries with all 8 symbols:
  - command: `env PHASE10_SYMBOLS=ADAUSDT,AVAXUSDT,BNBUSDT,DOGEUSDT,ETHUSDT,LINKUSDT,SOLUSDT,XRPUSDT GOCACHE=/home/davidmiguel22573/Github/ak-engine/.cache/go-build GOMODCACHE=/home/davidmiguel22573/Github/ak-engine/.cache/go-mod GOWORK=off GOTOOLCHAIN=local ./scripts/phase10_9c_phone_worker.sh coverage`
  - scope: all families, all sides, all horizons, all retained candidates
  - refreshed `runs/reports/phase10_8c_retained_coverage.{json,md}`
  - refreshed `runs/reports/phase10_8_ranked_inventory.{json,md}`
  - copied refreshed coverage to `runs/reports/phase10_9c_retained_coverage_recovery.{json,md}`
- Coverage closeout:
  - found_symbols_before: `ADAUSDT`, `AVAXUSDT`, `BNBUSDT`, `DOGEUSDT`, `ETHUSDT`, `LINKUSDT`, `SOLUSDT`, `XRPUSDT`
  - found_symbols_after: `ADAUSDT`, `AVAXUSDT`, `BNBUSDT`, `DOGEUSDT`, `ETHUSDT`, `LINKUSDT`, `SOLUSDT`, `XRPUSDT`
  - missing_symbols_after: `[]`
  - summary files found: `192`
  - malformed summaries: `0`
  - full_universe_ready: `true`
  - raw_required: `false`
- Inventory closeout:
  - candidate_scope: `all retained candidates`
  - candidate_count: `36`
  - SHADOW_CANDIDATE count: `0`
  - RESEARCH_LEAD count: `0`
  - FRAGILE_RESEARCH_LEAD count: `0`
  - REJECTED count: `36`
  - strongest ranked candidate: `NegativeFundingLong|long|240m`, still `REJECTED`
  - no candidate became strong enough for a research lead or shadow candidate after full coverage
- Verification:
  - `env GOCACHE=/home/davidmiguel22573/Github/ak-engine/.cache/go-build GOMODCACHE=/home/davidmiguel22573/Github/ak-engine/.cache/go-mod GOWORK=off GOTOOLCHAIN=local go test ./...` passed
- Boundaries:
  - ak-trader untouched
  - no promotion to ak-trader
  - remote raw gzip files were not deleted in this closeout

### SSH Run Status Check
- Check time: `2026-06-30 15:02:06 PDT` on the Termux phone.
- Restored planning context from `task_plan.md`, `findings.md`, and `progress.md`.
- Local tmux state:
  - `ak-phone-ssh` is not present and must be recreated if persistent remote monitoring is needed.
  - `agy-bnb-watch` is still present but stale; it stopped after detecting the first BNB summary and does not reflect final BNB completion.
- Direct SSH to `davidmiguel22573@192.168.1.79:8022` succeeded with `-F /dev/null`.
- Remote active-run state:
  - no remote tmux sessions listed
  - no active `phase10_9c_phone_worker`, `phase10-funding-event-pipeline`, or `go run ./cmd/ak-engine` pipeline process found, aside from the current probe command itself
- Remote summary state:
  - `BNBUSDT`: `24` alpha summaries, `2024-01 .. 2025-12`
  - `ETHUSDT`: `24` alpha summaries, `2024-01 .. 2025-12`
  - `LINKUSDT`: `24` alpha summaries, `2024-01 .. 2025-12`
  - `SOLUSDT`: `24` alpha summaries, `2024-01 .. 2025-12`
  - `AVAXUSDT`: `0`
  - `ADAUSDT`: `0`
  - `DOGEUSDT`: `0`
- Remote aggregate reports currently visible only for `LINKUSDT` and `SOLUSDT`.
- Remote raw gzip leftovers still visible:
  - `BNBUSDT`: `24`
  - `ETHUSDT`: `24`
  - `AVAXUSDT`, `ADAUSDT`, `DOGEUSDT`: `0`
- Local retained coverage remains the current authoritative artifact:
  - `summary files found=120`
  - `malformed summaries=0`
  - `full_universe_ready=false`
  - missing expected symbols: `ADAUSDT`, `AVAXUSDT`, `DOGEUSDT`
- Next action from the planning files remains: launch `AVAXUSDT` next on the phone, then pull summaries serially, validate with `jq`, rerun retained coverage, and only then proceed to `ADAUSDT` and `DOGEUSDT`.

### AVAX Launch Checkpoint
- Check time: `2026-06-30 15:16:25 PDT`.
- Started the next pending symbol, `AVAXUSDT`, on the Termux phone.
- First launch in remote tmux session `ak-engine-109c-avax` failed immediately during `build-features` for `2024-01`:
  - error: `failed to load primary candles: no matching files in range`
  - direct monthly AVAX candle path count was `0`
  - partitioned AVAX candle/funding data existed, but the pipeline still needed the historian fetch repair path already used for `SOLUSDT`
- Ran phone-side candle backfill in remote tmux session `ak-historian-avax-candles`:
  - command: `./bin/ak-historian fetch --market futures-um --symbols AVAXUSDT --interval 1m --period monthly --start 2024-01 --end 2025-12 --force --workdir "$HOME/Github/ak-historian/.ak-historian/work" --keep`
  - result: `planned=24`, `uploaded=24`, `skipped_existing=0`, `skipped_missing=0`, `failed=0`
- Verified the repair with a direct phone-side `build-features` smoke for `AVAXUSDT 2024-01`:
  - status: `PASS`
  - rows: `44640`
  - output: `runs/features/chunks/AVAXUSDT/2024-01-context-smoke.json`
- Relaunched remote AVAX worker:
  - tmux session: `ak-engine-109c-avax`
  - command: `./scripts/phase10_9c_phone_worker.sh run-symbol AVAXUSDT 2024-01 2025-12`
  - verified live process chain: bash wrapper, `go run ./cmd/ak-engine phase10-funding-event-pipeline`, and compiled `ak-engine phase10-funding-event-pipeline`
- Remote AVAX progress:
  - first `2024-01-alpha-summary.json` appeared and the worker advanced into `2024-02`
  - later remote check showed AVAX summaries through `2024-05`
  - worker remained live and CPU-active
- Pull/validate/coverage:
  - first helper pull succeeded and local `AVAXUSDT 2024-01` validated with `jq`
  - a later sandboxed helper pull failed with `socket: Operation not permitted`; reran the same pull with escalated network permission and it succeeded
  - local `AVAXUSDT` summaries now exist and validate through `2024-04`
  - local retained coverage rerun succeeded with `summary files found=124`, `malformed summaries=0`, `full_universe_ready=false`
  - local coverage now lists found expected symbols including `AVAXUSDT`, and missing expected symbols are `ADAUSDT`, `DOGEUSDT`
- Next action: keep polling `ak-engine-109c-avax`; when more remote AVAX summaries accumulate, pull summaries, validate with `jq`, rerun retained coverage, and do not start `ADAUSDT` until AVAX is locally complete or a concrete AVAX failure requires intervention.

### AVAX Follow-up Pull Checkpoint
- Check time: `2026-06-30 15:18:36 PDT`.
- Remote `AVAXUSDT` worker is still live in tmux session `ak-engine-109c-avax`.
- Remote AVAX alpha summaries now exist through `2024-08`.
- Pulled summaries again with `SSH_CONFIG_FILE=/dev/null make termux-worker-pull-summaries` using escalated network permission after the sandboxed helper path previously hit a socket permission error.
- Local `AVAXUSDT` alpha summaries now exist for `2024-01 .. 2024-07`.
- `jq` validation passed for all 7 local AVAX alpha summaries.
- Reran local retained coverage successfully:
  - `summary files found=127`
  - `malformed summaries=0`
  - `full_universe_ready=false`
  - found expected symbols include `AVAXUSDT`
  - missing expected symbols remain `ADAUSDT`, `DOGEUSDT`
  - local AVAX missing months are `2024-08 .. 2025-12`
- Next action: keep polling remote AVAX, pull once more when a useful batch accumulates or when all 24 AVAX summaries appear, validate locally, and rerun coverage before moving to `ADAUSDT`.

### AVAX Completion & ADAUSDT Launch Checkpoint
- Check time: `2026-06-30 15:40:00 PDT`.
- Polled remote phone worker at `192.168.1.79:8022`. Found SSH connection returned `Connection refused`. Requested user to start `sshd` inside Termux on the phone.
- After user confirmed `sshd` was running, re-connected to the phone. Tmux server was not running (likely due to Termux or system restart), but the remote AVAX worker had already completed all 24 months through `2025-12` (final summary generated at `15:28`).
- Pulled AVAX summaries locally using `SSH_CONFIG_FILE=/dev/null make termux-worker-pull-summaries`.
- Validated all 24 local `AVAXUSDT` alpha summaries with `jq`; all passed.
- Reran local retained coverage successfully:
  - `summary files found=120` (including 24 AVAX, 24 BNB, 24 ETH, 24 LINK, and 24 SOL summaries)
  - `malformed summaries=0`
  - `full_universe_ready=false`
  - missing expected symbols: `ADAUSDT`, `DOGEUSDT`
- Checked remote phone historian directory and found that `ADAUSDT` candles did not exist.
- Backfilled `ADAUSDT` monthly candles remotely:
  - command: `./bin/ak-historian fetch --market futures-um --symbols ADAUSDT --interval 1m --period monthly --start 2024-01 --end 2025-12 --force --workdir "$HOME/Github/ak-historian/.ak-historian/work" --keep`
  - result: `planned=24`, `uploaded=24`, `failed=0`
- Launched the next missing symbol (`ADAUSDT`) in a new remote tmux session `ak-engine-109c-ada`:
  - command: `./scripts/phase10_9c_phone_worker.sh run-symbol ADAUSDT 2024-01 2025-12`
- Verified that the `go run ./cmd/ak-engine phase10-funding-event-pipeline` process is running and compiling on the phone worker.
- Next action: poll `ak-engine-109c-ada` remote worker, pull `ADAUSDT` summaries serially as they appear, validate with `jq`, rerun coverage, and only then proceed to `DOGEUSDT`.

## Session: 2026-06-28

### Refresh Checkpoint
- Refresh time: `2026-06-28 23:14:29 PDT`
- User requested a context refresh summary, a freeze of all local changes, a git push, a phone-sync check, and a concise statement of the remaining goal work.
- Restored the current planning context from `task_plan.md`, `findings.md`, `progress.md`, and the active runbook attachment before taking any new action.
- Re-read the authoritative local retained-coverage report:
  - coverage status remains `partial_universe_only`
  - found symbols remain `LINKUSDT`, `SOLUSDT`, `XRPUSDT`
  - local `SOLUSDT` retained months remain `2024-01 .. 2025-07`
  - local missing symbols remain `ADAUSDT`, `AVAXUSDT`, `BNBUSDT`, `DOGEUSDT`, `ETHUSDT`
- Re-checked the current local `SOLUSDT` month frontier directly from `runs/reports/chunks/SOLUSDT`; it still tops out at `2025-07`.
- Checked the local repo publish context:
  - current branch is `main`
  - remote `origin` is `git@github.com:david22573/ak-engine.git`
  - local staged/unstaged change set for this refresh consists of `findings.md`, `progress.md`, `scripts/termux_worker_sync.sh`, `termux_worker.env.example`, and `termux_worker_runbook.md`
- Attempted a fresh phone-side status check with `ssh -F /dev/null -p 8022 davidmiguel22573@192.168.1.79 ...` before syncing.
- Phone-side verification failed immediately with `connection refused`, so this refresh turn cannot confirm whether the remote `SOLUSDT` worker has progressed beyond the last known `2025-09` phone frontier or accept a fresh repo sync yet.

## Session: 2026-06-29

### Reachability Checkpoint
- Check time: `2026-06-29 07:06:57 PDT`
- Restored planning context from `findings.md`, `progress.md`, and the retained-coverage report before retrying remote work.
- Retried a direct phone probe with `ssh -F /dev/null -p 8022 davidmiguel22573@192.168.1.79 ...`.
- The phone again returned `ssh: connect to host 192.168.1.79 port 8022: Connection refused`.
- No fresh remote sync, pull, or worker-state verification was possible during this turn.
- Reconfirmed the authoritative local retained frontier is unchanged:
  - coverage status: `partial_universe_only`
  - found symbols: `LINKUSDT`, `SOLUSDT`, `XRPUSDT`
  - local `SOLUSDT` months: `2024-01 .. 2025-07`
  - local missing `SOLUSDT` months: `2025-08 .. 2025-12`
  - remaining missing symbols: `ADAUSDT`, `AVAXUSDT`, `BNBUSDT`, `DOGEUSDT`, `ETHUSDT`

### Recovery Checkpoint
- Check time: `2026-06-29 11:11:06 PDT`
- Retried the direct phone probe after the user reported the phone should be back up.
- SSH connectivity to `192.168.1.79:8022` was restored.
- Phone-side `SOLUSDT` alpha-summary files now show completion through `2025-12`.
- The remote `~/Github/ak-engine` path is a synced working tree without `.git`, so `git rev-parse HEAD` there is not a valid verification check; artifact checks are the correct source of truth on the phone.
- `SSH_CONFIG_FILE=/dev/null make termux-worker-pull-summaries` completed successfully and advanced the local `SOLUSDT` summary set through `2025-12`.
- `SSH_CONFIG_FILE=/dev/null make termux-worker-pull-reports` also completed.
- Verified all local `SOLUSDT` alpha summaries parse with `jq`.
- Reran local retained coverage successfully:
  - `summary files found=72`
  - `malformed summaries=0`
  - `SOLUSDT` now has full retained coverage for `2024-01 .. 2025-12`
  - overall coverage status remains `partial_universe_only`
  - remaining missing symbols are now only `ADAUSDT`, `AVAXUSDT`, `BNBUSDT`, `DOGEUSDT`, `ETHUSDT`

### ETH Launch Checkpoint
- Check time: `2026-06-29 11:18 PDT`
- Restored planning context from the pasted attachment plus current `findings.md` / `progress.md`, then corrected `task_plan.md` so the persistent plan now tracks the retained-coverage recovery objective rather than the older refactor task.
- Re-ran the local retained-coverage workflow and reconfirmed the current authoritative Chromebook state:
  - coverage status: `partial_universe_only`
  - `full_universe_ready=false`
  - `summary files found=72`
  - `malformed summaries=0`
  - remaining missing symbols: `ADAUSDT`, `AVAXUSDT`, `BNBUSDT`, `DOGEUSDT`, `ETHUSDT`
- Probed the phone worker before launching more work:
  - no active `phase10_9c_phone_worker.sh` or `phase10-funding-event-pipeline` process was running
  - remote compact summaries still existed only for `LINKUSDT` and `SOLUSDT`
- Launched the next missing symbol on the phone worker:
  - new tmux session: `ak-engine-109c-eth`
  - worker command: `./scripts/phase10_9c_phone_worker.sh run-symbol ETHUSDT 2024-01 2025-12`
  - live pipeline child verified: `go run ./cmd/ak-engine phase10-funding-event-pipeline ... --symbols ETHUSDT ...`
- First post-launch verification showed the worker was live but still early:
  - `runs/reports/phase10_9c_ETHUSDT_worker.log` existed but had not yet emitted useful lines
  - no `runs/reports/chunks/ETHUSDT/*-alpha-summary.json` files existed yet
  - this is consistent with an in-flight start, not an immediate failure
- Follow-up poll a short time later still showed:
  - tmux session `ak-engine-109c-eth` alive
  - the bash wrapper and `go run ... phase10-funding-event-pipeline` child both still running
  - no `runs/features/chunks/ETHUSDT/*`, `runs/regimes/chunks/ETHUSDT/*`, or `runs/reports/chunks/ETHUSDT/*-alpha-summary.json` files yet
  - no early failure signal has appeared; the run is simply still at a very early stage
- Stronger artifact-level follow-up showed:
  - active child command: `classify-regimes --features runs/features/chunks/ETHUSDT/2024-01-context.json --out runs/regimes/chunks/ETHUSDT/2024-01-context.json`
  - `runs/features/chunks/ETHUSDT/2024-01-context.json` now exists on the phone at about `39.9 MB`
  - the matching regime chunk, alpha summaries, and `runs/reports/phase10_9c_ETHUSDT_pipeline.md` still do not exist yet
  - this confirms `ETHUSDT` has advanced past feature building for the first month and is not failing on the same missing-input path that originally blocked `SOLUSDT`
- A later bounded SSH probe succeeded and improved the `ETHUSDT` state checkpoint:
  - `runs/regimes/chunks/ETHUSDT/2024-01-context.json` now exists remotely
  - `runs/reports/chunks/ETHUSDT/2024-01-alpha-summary.json` still does not exist remotely
  - `runs/features/chunks/ETHUSDT/2024-01-context.json` still exists remotely
- Ran `SSH_CONFIG_FILE=/dev/null make termux-worker-pull-summaries` after that probe:
  - helper pull completed successfully
  - local `runs/reports/chunks/ETHUSDT/` still remained empty, confirming no completed remote alpha summaries were available to pull yet
- Reran local retained coverage after the pull attempt and reconfirmed the authoritative local state is unchanged:
  - coverage status: `partial_universe_only`
  - `full_universe_ready=false`
  - `summary files found=72`
  - `malformed summaries=0`
  - missing symbols remain `ADAUSDT`, `AVAXUSDT`, `BNBUSDT`, `DOGEUSDT`, `ETHUSDT`
- A subsequent bounded SSH probe for `2024-02` progress timed out (`exit 124`), so remote transport remains intermittently flaky even when using `timeout 8s ssh ...`
- Another bounded SSH probe later also timed out while checking whether `ETHUSDT` had produced `2024-01-alpha-summary.json` or advanced into `2024-02`.
- Ran a second `SSH_CONFIG_FILE=/dev/null make termux-worker-pull-summaries` after that timeout:
  - helper pull completed successfully again
  - local `runs/reports/chunks/ETHUSDT/` still remained empty, so there were still no completed remote `ETHUSDT` summaries available to copy back
- A minimal bounded liveness check for the `ETHUSDT` pipeline process also timed out (`exit 124`), so the current blocker is transport observability rather than proven pipeline failure.
- Reran local retained coverage after the second empty pull and reconfirmed the authoritative local state is still unchanged:
  - coverage status: `partial_universe_only`
  - `full_universe_ready=false`
  - `summary files found=72`
  - `malformed summaries=0`
  - missing symbols remain `ADAUSDT`, `AVAXUSDT`, `BNBUSDT`, `DOGEUSDT`, `ETHUSDT`
- Later ops-style bounded inspection resolved the actual `ETHUSDT` state:
  - tmux session `ak-engine-109c-eth` still exists
  - the `ETHUSDT` worker process is still live; no restart is needed
  - remote `ETHUSDT` file count is `27`
  - remote retained `ETHUSDT` alpha-summary count is `4`
  - remote retained months currently present are `2024-01`, `2024-02`, `2024-03`, `2024-04`
  - heavy raw leftovers still include `ETHUSDT/2024-01..2024-04-funding-events.jsonl.gz` and many `SOLUSDT` funding-event gzip files
- Pulled summaries immediately after that inspection:
  - `SSH_CONFIG_FILE=/dev/null make termux-worker-pull-summaries` completed successfully
  - local `ETHUSDT` summaries now exist for `2024-01`, `2024-02`, `2024-03`, `2024-04`
  - `jq` validation passed for all four local `ETHUSDT` alpha summaries
- Reran local retained coverage after the pull:
  - coverage status remains `partial_universe_only`
  - `full_universe_ready=false`
  - found symbols are now `ETHUSDT`, `LINKUSDT`, `SOLUSDT`, `XRPUSDT`
  - missing symbols are now only `ADAUSDT`, `AVAXUSDT`, `BNBUSDT`, `DOGEUSDT`
  - `ETHUSDT` retained months locally: `2024-01`, `2024-02`, `2024-03`, `2024-04`
  - `ETHUSDT` missing expected months locally: `2024-05 .. 2025-12`
  - `summary files found=76`
  - `malformed summaries=0`
- Next-context handoff:
  - continue `ETHUSDT` only for missing months `2024-05 .. 2025-12`
  - do not restart `ETHUSDT` from scratch because the worker is live and partial retained summaries already exist
  - do not start `BNBUSDT` until `ETHUSDT` is locally complete or a concrete failure requires intervention
  - keep the serial loop: pull summaries, validate with `jq`, rerun retained coverage, then reassess the next missing `ETHUSDT` month frontier

### ETH Continuation Checkpoint
- Check time: `2026-06-29 23:22:25 PDT`
- Restored planning context with `planning-with-files` and ran `session-catchup.py`; no extra catch-up context was emitted.
- Confirmed local `ETHUSDT` retained summaries are still only:
  - `2024-01-alpha-summary.json`
  - `2024-02-alpha-summary.json`
  - `2024-03-alpha-summary.json`
  - `2024-04-alpha-summary.json`
- Validated all four local `ETHUSDT` alpha summaries with `jq`.
- Attempted bounded direct phone inspection with `ssh -F /dev/null -p 8022 davidmiguel22573@192.168.1.79 ...`; the phone returned `Connection refused`.
- Reran local retained coverage with `make phase10-9c-coverage`; it completed successfully and rewrote `runs/reports/phase10_8c_retained_coverage.{json,md}` plus ranked inventory artifacts.
- Current authoritative local coverage state after the rerun:
  - coverage status: `partial_universe_only`
  - `full_universe_ready=false`
  - `summary files found=76`
  - `malformed summaries=0`
  - found symbols: `ETHUSDT`, `LINKUSDT`, `SOLUSDT`, `XRPUSDT`
  - `ETHUSDT` retained months locally: `2024-01`, `2024-02`, `2024-03`, `2024-04`
  - `ETHUSDT` missing expected months locally: `2024-05 .. 2025-12`
  - missing expected symbols in the report: `ADAUSDT`, `AVAXUSDT`, `BNBUSDT`, `DOGEUSDT`
- Attempted the standard serial pull step with `SSH_CONFIG_FILE=/dev/null make termux-worker-pull-summaries`; it failed because SSH to `192.168.1.79:8022` returned `Connection refused`, and the tar fallback then failed because no archive was received.
- No new `ETHUSDT` summaries were pulled in this checkpoint.
- Next action remains blocked on phone SSH reachability: restore access to `192.168.1.79:8022`, inspect `ak-engine-109c-eth`, then pull/validate/coverage before starting any other symbol.

### Persistent SSH + BNB Launch Checkpoint
- Check time: `2026-06-29 23:33:22 PDT`
- Created local tmux session `ak-phone-ssh` running a persistent Termux SSH login.
- Active phone SSH target was later updated to:
  - `ssh -F /dev/null -p 8022 davidmiguel22573@192.168.1.81`
- Verified remote commands can be sent and captured through `ak-phone-ssh:0.0`; future phone probes should reuse this persistent pane instead of opening a new SSH connection each time.
- Remote `ETHUSDT` state after phone SSH returned:
  - phone-side alpha summaries exist for all `2024-01 .. 2025-12`
  - phone-side raw event gzip files still exist for all 24 ETH months
  - `runs/reports/phase10_9c_ETHUSDT_pipeline.md` is missing
  - `runs/reports/phase10_9c_ETHUSDT_worker.log` exists but is empty
- Pulled summaries with `SSH_CONFIG_FILE=/dev/null make termux-worker-pull-summaries`; the pull completed successfully.
- Validated all 24 local `ETHUSDT` alpha summaries with `jq`.
- Reran local retained coverage with `make phase10-9c-coverage`; it completed successfully.
- Current authoritative local coverage state after ETH completion:
  - coverage status: `partial_universe_only`
  - `full_universe_ready=false`
  - `summary files found=96`
  - `malformed summaries=0`
  - found symbols: `ETHUSDT`, `LINKUSDT`, `SOLUSDT`, `XRPUSDT`
  - `ETHUSDT` retained months locally: `2024-01 .. 2025-12`
  - missing expected symbols: `ADAUSDT`, `AVAXUSDT`, `BNBUSDT`, `DOGEUSDT`
- Skipped ETH raw cleanup because the phone-side ETH aggregate report is missing; this preserves the rule that raw event files are deleted only after aggregate reports are written.
- Checked phone storage before launching the next symbol:
  - available space: about `38G`
  - `runs/reports/chunks/ETHUSDT`: about `188M`
- Launched `BNBUSDT` on the phone in remote tmux session `ak-engine-109c-bnb` with:
  - `./scripts/phase10_9c_phone_worker.sh run-symbol BNBUSDT 2024-01 2025-12`
- Verified live BNB process chain:
  - `bash ./scripts/phase10_9c_phone_worker.sh run-symbol BNBUSDT 2024-01 2025-12`
  - `go run ./cmd/ak-engine phase10-funding-event-pipeline ... --symbols BNBUSDT ... --out runs/reports/phase10_9c_BNBUSDT_pipeline.md`
- BNB poll at `2026-06-29 23:33:10 PDT`:
  - remote tmux session `ak-engine-109c-bnb` exists
  - BNB process chain is live
  - `bnb_summaries=0`
  - no first feature/regime/report artifacts observed yet
- BNB poll at `2026-06-29 23:35:21 PDT`:
  - remote tmux session `ak-engine-109c-bnb` still exists
  - compiled `ak-engine phase10-funding-event-pipeline` child is running
  - current child process is `classify-regimes --features runs/features/chunks/BNBUSDT/2024-01-context.json --out runs/regimes/chunks/BNBUSDT/2024-01-context.json`
  - `runs/features/chunks/BNBUSDT/2024-01-context.json` exists
  - `bnb_summaries=0`
- Next action: continue polling BNB via `ak-phone-ssh`, pull summaries serially once they appear, validate local JSON, rerun retained coverage, then reassess before starting `AVAXUSDT`.

### Agy Watcher Checkpoint
- Check time: `2026-06-29 23:40 PDT`
- Launched local tmux session `agy-bnb-watch` with:
  - `agy --model "Gemini 3.5 Flash (Medium)" --prompt-interactive ...`
- Watcher objective:
  - reuse existing persistent SSH pane `ak-phone-ssh:0.0`
  - poll remote `BNBUSDT` alpha-summary count, process state, and latest artifacts every 2 minutes
  - stop and report when BNB summaries appear, saying `ready_to_pull_bnb_summaries` if all 24 appear
  - do not start new symbols
  - do not delete `ETHUSDT` raw gzip files
- Verified the watcher session started and began tool activity by listing tmux sessions and capturing `ak-phone-ssh:0.0`.

### Termux IP Update + BNB Pull Checkpoint
- Check time: `2026-06-30 12:13:00 PDT`
- User reported the Termux phone SSH IP changed to `192.168.1.81` with the same port `8022`.
- Updated ignored local env file `termux_worker.env`:
  - `PHONE_HOST=davidmiguel22573@192.168.1.81`
  - `SSH_ARGS='-p 8022 -o UserKnownHostsFile=.cache/termux_known_hosts -o StrictHostKeyChecking=accept-new'`
- Existing `ak-phone-ssh` pane was still alive and showed the BNB worker had already produced at least `2024-01-alpha-summary.json`.
- First pull attempt through the new IP failed on SSH host-key verification because no askpass was available.
- Retried after adding repo-local known-hosts args; `SSH_CONFIG_FILE=/dev/null make termux-worker-pull-summaries` succeeded and accepted `[192.168.1.81]:8022`.
- Local BNB pull result:
  - `15` BNB alpha summaries present
  - BNB local months: `2024-01 .. 2024-12`, `2025-01 .. 2025-03`
  - `jq` validation passed for all 15 BNB alpha summaries
- Reran `make phase10-9c-coverage`; it passed.
- Current authoritative local coverage after BNB partial pull:
  - coverage status: `partial_universe_only`
  - `full_universe_ready=false`
  - `summary files found=111`
  - `malformed summaries=0`
  - found symbols: `BNBUSDT`, `ETHUSDT`, `LINKUSDT`, `SOLUSDT`, `XRPUSDT`
  - missing expected symbols: `ADAUSDT`, `AVAXUSDT`, `DOGEUSDT`
  - BNB retained months locally: `2024-01 .. 2025-03`
  - BNB missing expected months locally: `2025-04 .. 2025-12`
- Next action: continue BNB only, using `192.168.1.81:8022` for new SSH connections; pull/validate/coverage again when more BNB summaries are available.

### Final Pre-Reset BNB Resume Checkpoint
- Check time: `2026-06-30 12:19:15 PDT`
- User reported the phone is back on `192.168.1.79:8022`; `.81` was a temporary issue.
- Updated ignored local env file `termux_worker.env` back to:
  - `PHONE_HOST=davidmiguel22573@192.168.1.79`
- Recreated persistent local tmux SSH session `ak-phone-ssh` against `.79`.
- Verified the phone-side BNB state before resuming:
  - `remote_bnb_count=15`
  - remote BNB summaries still ended at `2025-03`
  - no active BNB worker process was running
- Launched remote resume worker:
  - remote tmux session: `ak-engine-109c-bnb-resume`
  - command: `./scripts/phase10_9c_phone_worker.sh run-symbol BNBUSDT 2025-04 2025-12`
  - log path: `runs/reports/phase10_9c_BNBUSDT_resume_worker.log`
- Verified the resume process chain is live:
  - `bash ./scripts/phase10_9c_phone_worker.sh run-symbol BNBUSDT 2025-04 2025-12`
  - `go run ./cmd/ak-engine phase10-funding-event-pipeline ... --symbols BNBUSDT --from 2025-04 --to 2025-12 ...`
  - compiled `ak-engine phase10-funding-event-pipeline` child running
- Immediate post-launch remote count remains `15`, expected before the first resumed month completes.
- Fresh-context next action:
  - read `task_plan.md`, `findings.md`, and `progress.md`
  - use `ak-phone-ssh` or SSH to `192.168.1.79:8022`

### BNB Completion Pull Checkpoint
- Check time: `2026-06-30 12:52:49 PDT`
- Retried Termux SSH endpoints:
  - `192.168.1.79:8022` succeeded
  - `192.168.1.81:8022` timed out and is stale for now
- Remote BNB state on `.79`:
  - `remote_bnb_count=24`
  - latest BNB alpha summaries include `2025-04 .. 2025-12`
- Ran `SSH_CONFIG_FILE=/dev/null make termux-worker-pull-summaries`; pull completed successfully.
- Local BNB state after pull:
  - `24` local `BNBUSDT` alpha summaries exist for `2024-01 .. 2025-12`
  - `jq empty runs/reports/chunks/BNBUSDT/*-alpha-summary.json` passed
- Ran `SSH_CONFIG_FILE=/dev/null make termux-worker-pull-reports`; pull completed successfully.
- Reran retained coverage with repo-local Go caches via `make phase10-9c-coverage`; it completed successfully.
- Current authoritative local coverage state:
  - coverage status: `partial_universe_only`
  - `full_universe_ready=false`
  - `summary files found=120`
  - `malformed summaries=0`
  - `BNBUSDT`, `ETHUSDT`, `LINKUSDT`, and `SOLUSDT` are fully retained locally through `2025-12`
  - remaining missing expected symbols: `ADAUSDT`, `AVAXUSDT`, `DOGEUSDT`
- Remote cleanup state:
  - BNB raw `*-funding-events.jsonl.gz` files were still visible on the phone
  - no BNB aggregate markdown was visible in the latest remote report check
  - keep BNB raw cleanup blocked until aggregate/report state is reconciled
- Next action: start or resume `AVAXUSDT` on the phone worker; do not start multiple symbols concurrently.
  - poll `ak-engine-109c-bnb-resume` until `2025-04-alpha-summary.json` and later months appear
  - run `SSH_CONFIG_FILE=/dev/null make termux-worker-pull-summaries`
  - validate BNB summaries with `jq`
  - rerun `make phase10-9c-coverage`
  - do not start `AVAXUSDT` until BNB is complete locally through `2025-12`

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
- Committed the local phone-worker and retained-coverage support as `df1a833` (`Add termux phone worker flow`) while leaving the unrelated untracked fixture `testdata/candles/btc_5m_take_profit_fixture.json` untouched.
- Pulled `LINKUSDT` compact outputs and reports back to the Chromebook with direct `ssh -F /dev/null -p 8022 ... | tar ...` because the helper-backed pull path hit a local SSH config permission error on `/etc/ssh/ssh_config.d/20-systemd-ssh-proxy.conf`.
- Reran local retained coverage after the `LINKUSDT` pull and confirmed the Chromebook state advanced from `single_symbol_only` to `partial_universe_only`:
  - found symbols: `LINKUSDT`, `XRPUSDT`
  - missing symbols: `ADAUSDT`, `AVAXUSDT`, `BNBUSDT`, `DOGEUSDT`, `ETHUSDT`, `SOLUSDT`
  - `full_universe_ready=false`
- Removed the stale phone-side `LINKUSDT` raw event gzip files after the pull:
  - before cleanup: `24` files, about `92.5 MB`
  - after cleanup: `0` files
  - remote `runs/reports/chunks/LINKUSDT` size dropped from `196M` to `108M`
- Started the next symbol worker for `SOLUSDT` in remote tmux session `ak-engine-109c-sol`.
- Diagnosed the first `SOLUSDT` worker failure as missing direct candle parquet for `2024-01` in the path consumed by `build-features`, even though phone-side funding-rate parquet existed and partitioned candle parquet already existed under `work/candles/...`.
- Backfilled `SOLUSDT` 1m candles on the phone with:
  - `~/Github/ak-historian/bin/ak-historian fetch --market futures-um --symbols SOLUSDT --interval 1m --period monthly --start 2024-01 --end 2025-12 --force --workdir "$HOME/Github/ak-historian/.ak-historian/work" --keep`
  - Result: `planned=24`, `uploaded=24`, `failed=0`
- Re-ran the exact failing probe after backfill and confirmed it now passes:
  - `go run ./cmd/ak-engine build-features ... --symbol SOLUSDT --from 2024-01-01 --to 2024-01-31 ...`
  - status `PASS`, rows `44640`
- Relaunched the `SOLUSDT` phone worker after backfill and verified the new run is active:
  - tmux session `ak-engine-109c-sol` exists
  - process chain includes `phase10_9c_phone_worker.sh` and `phase10-funding-event-pipeline`
  - fresh files now exist under `runs/features/chunks/SOLUSDT/2024-01-*` and `runs/regimes/chunks/SOLUSDT/2024-01-context.json`
  - the manifest and pipeline markdown are still stale from the pre-backfill failed attempt, so they cannot yet be trusted as live progress indicators for the restarted run
- Patched `scripts/termux_worker_sync.sh` to support optional `SSH_CONFIG_FILE`, allowing helper-based push/pull commands to ignore broken system SSH config includes by prepending `-F <path>` to SSH and rsync transport commands.
- Updated `termux_worker.env.example` and `termux_worker_runbook.md` to document `SSH_CONFIG_FILE=/dev/null` as the Chromebook workaround for `/etc/ssh/ssh_config.d/20-systemd-ssh-proxy.conf` permission failures.
- Verified the patched helper path with:
  - `bash -n scripts/termux_worker_sync.sh`
  - `SSH_CONFIG_FILE=/dev/null make termux-worker-pull-summaries`
- Confirmed the patched helper successfully pulled the first live `SOLUSDT` alpha summary back to the Chromebook:
  - local file present: `runs/reports/chunks/SOLUSDT/2024-01-alpha-summary.json`
- Current live `SOLUSDT` worker state after helper verification:
  - remote alpha summaries initially observed: `1`, later advanced to `2`
  - remote funding summaries initially observed: `1`, later advanced to at least `2`
  - remote artifacts confirmed for completed months: `2024-01`, `2024-02`
  - local mirror now contains `runs/reports/chunks/SOLUSDT/2024-01-alpha-summary.json` and `2024-02-alpha-summary.json`
  - remote process is still active and consuming CPU (`ak-engine` child about `12%` CPU at the latest poll)
  - remote manifest `updated_at` remains stale at `2026-06-29T00:21:32.130175957Z`, so manifest/report still lag the live run
  - direct phone-side alpha-summary listing is the current source of truth for completed `SOLUSDT` months because manifest/report state is lagging
- Reran local retained coverage after pulling the first two `SOLUSDT` alpha summaries:
  - found symbols now: `LINKUSDT`, `SOLUSDT`, `XRPUSDT`
  - missing symbols now: `ADAUSDT`, `AVAXUSDT`, `BNBUSDT`, `DOGEUSDT`, `ETHUSDT`
  - `SOLUSDT` retained months currently present locally: `2024-01`, `2024-02`
  - `full_universe_ready` remains `false`
- Confirmed the live phone worker has already advanced its scratch files into `SOLUSDT 2024-03`, so the run is still moving even though the report/manifest lag behind.
- Continued clean serial pull/catch-up cycles for `SOLUSDT` while the phone worker advanced:
  - remote completed months observed over time: `2024-03`, `2024-04`, `2024-05`, then `2024-06` through `2024-11`
  - local mirror is now caught up through `SOLUSDT 2024-11`
  - phone-side completed months match the local mirror at the last poll: `2024-01` .. `2024-11`
- Identified and corrected an operator error in the local workflow: running `pull-summaries` and the retained-coverage scan in parallel caused transient `malformed summary` warnings because coverage read files while the pull was overwriting them.
  - corrected procedure is now strictly serial:
    1. `SSH_CONFIG_FILE=/dev/null make termux-worker-pull-summaries`
    2. validate local summary JSONs if needed
    3. rerun `./scripts/phase10_9c_phone_worker.sh coverage`
- Latest clean retained-coverage result after serial pull + scan:
  - found symbols: `LINKUSDT`, `SOLUSDT`, `XRPUSDT`
  - missing symbols: `ADAUSDT`, `AVAXUSDT`, `BNBUSDT`, `DOGEUSDT`, `ETHUSDT`
  - `SOLUSDT` retained months locally: `2024-01` .. `2024-11`
  - `SOLUSDT` still missing locally: `2024-12`, `2025-01` .. `2025-12`
  - `summary files found=59`
  - `malformed summaries=0`
- Continued one more clean serial pull/check cycle:
  - phone-side `SOLUSDT` advanced to `2024-12`
  - local mirror now matches phone-side exactly for all `SOLUSDT` months `2024-01` .. `2024-12`
  - all local `SOLUSDT` alpha summaries validate with `jq`
- Latest retained-coverage result after the `2024-12` sync:
  - found symbols: `LINKUSDT`, `SOLUSDT`, `XRPUSDT`
  - missing symbols: `ADAUSDT`, `AVAXUSDT`, `BNBUSDT`, `DOGEUSDT`, `ETHUSDT`
  - `SOLUSDT` retained months locally: all of `2024-01` .. `2024-12`
  - `SOLUSDT` still missing locally: `2025-01` .. `2025-12`
  - `summary files found=60`
  - `malformed summaries=0`
- Verified the next live frontier for `SOLUSDT`:
  - phone-side completed report artifacts still top out at `2024-12`
  - phone-side scratch files have advanced into `2025-01` for both feature and regime chunks
  - active pipeline process remains healthy and CPU-active (latest poll about `26%` CPU on the `ak-engine` child)
  - no `2025-01-alpha-summary.json` exists yet, so there was nothing new to pull during this cycle
- Next clean serial pull/check cycle:
  - phone-side completed `SOLUSDT` months advanced to include `2025-01`
  - local mirror now matches through `SOLUSDT 2025-01`
  - local `SOLUSDT` alpha summaries still validate cleanly with `jq`
- Latest retained-coverage result after the `2025-01` sync:
  - found symbols: `LINKUSDT`, `SOLUSDT`, `XRPUSDT`
  - missing symbols: `ADAUSDT`, `AVAXUSDT`, `BNBUSDT`, `DOGEUSDT`, `ETHUSDT`
  - `SOLUSDT` retained months locally: `2024-01` .. `2024-12`, `2025-01`
  - `SOLUSDT` still missing locally: `2025-02` .. `2025-12`
  - `summary files found=61`
  - `malformed summaries=0`
- Continued the serial catch-up loop:
  - phone-side report artifacts now include `SOLUSDT 2025-02`
  - local mirror now matches through `SOLUSDT 2025-02`
  - local `SOLUSDT` alpha summaries continue to validate cleanly with `jq`
- Latest retained-coverage result after the `2025-02` sync:
  - found symbols: `LINKUSDT`, `SOLUSDT`, `XRPUSDT`
  - missing symbols: `ADAUSDT`, `AVAXUSDT`, `BNBUSDT`, `DOGEUSDT`, `ETHUSDT`
  - `SOLUSDT` retained months locally: `2024-01` .. `2024-12`, `2025-01`, `2025-02`
  - `SOLUSDT` still missing locally: `2025-03` .. `2025-12`
  - `summary files found=62`
  - `malformed summaries=0`
- Continued the serial catch-up loop again:
  - phone-side completed `SOLUSDT` months advanced to include `2025-03`
  - local mirror now matches through `SOLUSDT 2025-03`
  - local `SOLUSDT` alpha summaries continue to validate cleanly with `jq`
- Latest retained-coverage result after the `2025-03` sync:
  - found symbols: `LINKUSDT`, `SOLUSDT`, `XRPUSDT`
  - missing symbols: `ADAUSDT`, `AVAXUSDT`, `BNBUSDT`, `DOGEUSDT`, `ETHUSDT`
  - `SOLUSDT` retained months locally: `2024-01` .. `2024-12`, `2025-01`, `2025-02`, `2025-03`
  - `SOLUSDT` still missing locally: `2025-04` .. `2025-12`
  - `summary files found=63`
  - `malformed summaries=0`
- Continued the serial catch-up loop with a larger jump:
  - phone-side completed `SOLUSDT` months advanced to include `2025-04`, `2025-05`, and `2025-06`
  - local mirror now matches through `SOLUSDT 2025-06`
  - local `SOLUSDT` alpha summaries continue to validate cleanly with `jq`
- Latest retained-coverage result after the `2025-06` sync:
  - found symbols: `LINKUSDT`, `SOLUSDT`, `XRPUSDT`
  - missing symbols: `ADAUSDT`, `AVAXUSDT`, `BNBUSDT`, `DOGEUSDT`, `ETHUSDT`
  - `SOLUSDT` retained months locally: `2024-01` .. `2024-12`, `2025-01` .. `2025-06`
  - `SOLUSDT` still missing locally: `2025-07` .. `2025-12`
  - `summary files found=66`
  - `malformed summaries=0`
- Continued the serial catch-up loop again:
  - phone-side completed `SOLUSDT` months advanced to include `2025-07`
  - local mirror now matches through `SOLUSDT 2025-07`
  - local `SOLUSDT` alpha summaries continue to validate cleanly with `jq`
- Latest retained-coverage result after the `2025-07` sync:
  - found symbols: `LINKUSDT`, `SOLUSDT`, `XRPUSDT`
  - missing symbols: `ADAUSDT`, `AVAXUSDT`, `BNBUSDT`, `DOGEUSDT`, `ETHUSDT`
  - `SOLUSDT` retained months locally: `2024-01` .. `2024-12`, `2025-01` .. `2025-07`
  - `SOLUSDT` still missing locally: `2025-08` .. `2025-12`
  - `summary files found=67`
  - `malformed summaries=0`

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
| Freeze Commit | `git commit -m "Add termux phone worker flow"` | pass | Local phone-worker workflow, coverage-only support, and runbook changes are frozen as commit `df1a833`. |
| Local Coverage | `env GOCACHE=$PWD/.cache/go-build GOMODCACHE=$PWD/.cache/go-mod GOWORK=off GOTOOLCHAIN=local ./scripts/phase10_9c_phone_worker.sh coverage` | pass | After pulling `LINKUSDT`, retained coverage advanced to `partial_universe_only` with `LINKUSDT` + `XRPUSDT` present. |
| Remote Cleanup | `ssh -F /dev/null -p 8022 ... 'find ~/Github/ak-engine/runs/reports/chunks/LINKUSDT -name "*-funding-events.jsonl.gz" -delete ...'` | pass | Deleted stale raw `LINKUSDT` event gzip files after summaries/reports were safely pulled back. |
| Worker Pull | `make termux-worker-pull-summaries` / `make termux-worker-pull-reports` | fail recorded | Helper-backed pull path hit local SSH config permission error; direct `ssh -F /dev/null -p 8022 ... | tar ...` succeeded as a workaround. |
| SOL Worker | `ssh -F /dev/null -p 8022 ... 'sed -n "1,260p" runs/reports/phase10_9c_SOLUSDT_worker.log'` | fail recorded | First `SOLUSDT` worker attempt blocked immediately on `failed to load primary candles: no matching files in range` for `2024-01`. |
| SOL Backfill | `ssh -F /dev/null -p 8022 ... 'cd ~/Github/ak-historian && ./bin/ak-historian fetch --market futures-um --symbols SOLUSDT --interval 1m --period monthly --start 2024-01 --end 2025-12 --force --workdir "$HOME/Github/ak-historian/.ak-historian/work" --keep'` | pass | Backfilled all `24` planned `SOLUSDT` monthly candle files needed by `build-features`. |
| SOL Probe | `ssh -F /dev/null -p 8022 ... 'cd ~/Github/ak-engine && ... go run ./cmd/ak-engine build-features --symbol SOLUSDT --from 2024-01-01 --to 2024-01-31 ...'` | pass | Previously failing `2024-01` context build now succeeds with `44640` rows. |
| SOL Relaunch | `ssh -F /dev/null -p 8022 ... 'tmux new-session -d -s ak-engine-109c-sol ... ./scripts/phase10_9c_phone_worker.sh run-symbol SOLUSDT 2024-01 2025-12 ...'` | pass in progress | Relaunched `SOLUSDT` worker is active; live progress is currently visible via process state and fresh chunk files rather than the stale report/manifest. |
| Helper Hardening | `bash -n scripts/termux_worker_sync.sh` | pass | `SSH_CONFIG_FILE` support added without breaking script syntax. |
| Helper Workflow | `SSH_CONFIG_FILE=/dev/null make termux-worker-pull-summaries` | pass | Patched helper now works around the Chromebook SSH-config permission issue and successfully pulled the first `SOLUSDT` alpha summary. |
| SOL Progress | `ssh -F /dev/null -p 8022 ... 'find ~/Github/ak-engine/runs/reports/chunks/SOLUSDT -name "*-alpha-summary.json" ...'` | pass in progress | Direct phone-side listing now shows completed `SOLUSDT` months `2024-01` and `2024-02`, and both summaries are mirrored locally. |
| Local Coverage | `env GOCACHE=$PWD/.cache/go-build GOMODCACHE=$PWD/.cache/go-mod GOWORK=off GOTOOLCHAIN=local ./scripts/phase10_9c_phone_worker.sh coverage` | pass | Retained coverage now recognizes `SOLUSDT` as present with partial month coverage (`2024-01`, `2024-02`) alongside full `LINKUSDT` and `XRPUSDT`. |
| Serial Pull | `SSH_CONFIG_FILE=/dev/null make termux-worker-pull-summaries` | pass | Repeated clean serial pulls advanced the local `SOLUSDT` mirror to `2024-11` while avoiding concurrent-write scan races. |
| Local Coverage | `env GOCACHE=$PWD/.cache/go-build GOMODCACHE=$PWD/.cache/go-mod GOWORK=off GOTOOLCHAIN=local ./scripts/phase10_9c_phone_worker.sh coverage` | pass | Latest clean scan shows `SOLUSDT` retained locally through `2024-11`, `summary files found=59`, and `malformed summaries=0`. |
| Serial Pull | `SSH_CONFIG_FILE=/dev/null make termux-worker-pull-summaries` | pass | Latest clean pull advanced the local `SOLUSDT` mirror to all of `2024` and matched the phone-side month set through `2024-12`. |
| Local Coverage | `env GOCACHE=$PWD/.cache/go-build GOMODCACHE=$PWD/.cache/go-mod GOWORK=off GOTOOLCHAIN=local ./scripts/phase10_9c_phone_worker.sh coverage` | pass | Latest clean scan shows `SOLUSDT` retained locally through `2024-12`, `summary files found=60`, and `malformed summaries=0`. |
| Live Frontier | `ssh -F /dev/null -p 8022 ... 'find ~/Github/ak-engine/runs/features/chunks/SOLUSDT ...'` / `ps ...` | pass in progress | `SOLUSDT` has entered `2025-01` scratch generation on the phone, but no completed `2025` alpha summary exists yet. |
| Serial Pull | `SSH_CONFIG_FILE=/dev/null make termux-worker-pull-summaries` | pass | Latest clean pull advanced the local `SOLUSDT` mirror to include `2025-01`. |
| Local Coverage | `env GOCACHE=$PWD/.cache/go-build GOMODCACHE=$PWD/.cache/go-mod GOWORK=off GOTOOLCHAIN=local ./scripts/phase10_9c_phone_worker.sh coverage` | pass | Latest clean scan shows `SOLUSDT` retained locally through `2025-01`, `summary files found=61`, and `malformed summaries=0`. |
| Serial Pull | `SSH_CONFIG_FILE=/dev/null make termux-worker-pull-summaries` | pass | Latest clean pull advanced the local `SOLUSDT` mirror to include `2025-02`. |
| Local Coverage | `env GOCACHE=$PWD/.cache/go-build GOMODCACHE=$PWD/.cache/go-mod GOWORK=off GOTOOLCHAIN=local ./scripts/phase10_9c_phone_worker.sh coverage` | pass | Latest clean scan shows `SOLUSDT` retained locally through `2025-02`, `summary files found=62`, and `malformed summaries=0`. |
| Serial Pull | `SSH_CONFIG_FILE=/dev/null make termux-worker-pull-summaries` | pass | Latest clean pull advanced the local `SOLUSDT` mirror to include `2025-03`. |
| Local Coverage | `env GOCACHE=$PWD/.cache/go-build GOMODCACHE=$PWD/.cache/go-mod GOWORK=off GOTOOLCHAIN=local ./scripts/phase10_9c_phone_worker.sh coverage` | pass | Latest clean scan shows `SOLUSDT` retained locally through `2025-03`, `summary files found=63`, and `malformed summaries=0`. |
| Serial Pull | `SSH_CONFIG_FILE=/dev/null make termux-worker-pull-summaries` | pass | Latest clean pull advanced the local `SOLUSDT` mirror through `2025-06`. |
| Local Coverage | `env GOCACHE=$PWD/.cache/go-build GOMODCACHE=$PWD/.cache/go-mod GOWORK=off GOTOOLCHAIN=local ./scripts/phase10_9c_phone_worker.sh coverage` | pass | Latest clean scan shows `SOLUSDT` retained locally through `2025-06`, `summary files found=66`, and `malformed summaries=0`. |
| Serial Pull | `SSH_CONFIG_FILE=/dev/null make termux-worker-pull-summaries` | pass | Latest clean pull advanced the local `SOLUSDT` mirror through `2025-07`. |
| Local Coverage | `env GOCACHE=$PWD/.cache/go-build GOMODCACHE=$PWD/.cache/go-mod GOWORK=off GOTOOLCHAIN=local ./scripts/phase10_9c_phone_worker.sh coverage` | pass | Latest clean scan shows `SOLUSDT` retained locally through `2025-07`, `summary files found=67`, and `malformed summaries=0`. |
| Planning Repair | `task_plan.md` rewrite | pass | Replaced the stale refactor plan with the actual retained-coverage recovery phases and current verified state. |
| Local Coverage | `env GOCACHE=$PWD/.cache/go-build GOMODCACHE=$PWD/.cache/go-mod GOWORK=off GOTOOLCHAIN=local ./scripts/phase10_9c_phone_worker.sh coverage` | pass | Current local authoritative state remains `partial_universe_only` with `72` summary files, `0` malformed summaries, and missing symbols `ADAUSDT`, `AVAXUSDT`, `BNBUSDT`, `DOGEUSDT`, `ETHUSDT`. |
| Remote Status | `ssh -F /dev/null -p 8022 ... 'cd $HOME/Github/ak-engine; ...'` | pass | Direct remote inspection showed no active phone worker and no hidden compact-summary progress beyond `LINKUSDT` and `SOLUSDT`. |
| ETH Launch | `ssh -F /dev/null -p 8022 ... 'tmux new-session -d -s ak-engine-109c-eth ... ./scripts/phase10_9c_phone_worker.sh run-symbol ETHUSDT 2024-01 2025-12 ...'` | pass in progress | `ETHUSDT` is now the active phone-side worker in tmux; first verification confirmed the bash wrapper and `go run ... phase10-funding-event-pipeline` child are live, with summaries not yet written. |
| ETH Follow-up | `ssh -F /dev/null -p 8022 ... 'cd $HOME/Github/ak-engine; ...'` | pass in progress | A short follow-up poll still showed the `ETHUSDT` wrapper and `go run` child alive with no early artifact output yet, which is consistent with a very early in-flight launch. |
| ETH Artifact Probe | `ssh -F /dev/null -p 8022 ... 'cd $HOME/Github/ak-engine; ls -l runs/features/chunks/ETHUSDT/2024-01-context.json ...'` | pass in progress | `ETHUSDT` now has a `2024-01` feature context chunk on the phone (~39.9 MB) while regime and summary outputs are still pending, confirming real first-month progress. |
| ETH Bounded Probe | `timeout 8s ssh -F /dev/null -p 8022 ...` | pass in progress | Short bounded probe confirmed `ETHUSDT` now has both `2024-01` feature and regime chunks remotely, but still no `2024-01` or `2024-02` alpha summaries. |
| ETH Pull Attempt | `env SSH_CONFIG_FILE=/dev/null make termux-worker-pull-summaries` | pass | Helper pull completed cleanly, but no `ETHUSDT` summaries arrived locally because none were complete remotely yet. |
| Local Coverage | `env GOCACHE=$PWD/.cache/go-build GOMODCACHE=$PWD/.cache/go-mod GOWORK=off GOTOOLCHAIN=local ./scripts/phase10_9c_phone_worker.sh coverage` | pass | After the successful pull attempt, local retained coverage remained unchanged at `partial_universe_only` with `72` summary files and `0` malformed summaries. |
| ETH Bounded Probe | `timeout 8s ssh -F /dev/null -p 8022 ...` | fail recorded | A follow-up bounded probe for `2024-02` ETH progress timed out with `exit 124`, indicating intermittent SSH transport instability rather than proven worker failure. |
| ETH Bounded Probe | `timeout 8s ssh -F /dev/null -p 8022 ...` | fail recorded | Another bounded probe for `ETHUSDT` summary and `2024-02` progress timed out with `exit 124`. |
| ETH Pull Attempt | `env SSH_CONFIG_FILE=/dev/null make termux-worker-pull-summaries` | pass | Second helper pull also completed cleanly, but `runs/reports/chunks/ETHUSDT` remained empty. |
| ETH Liveness Probe | `timeout 8s ssh -F /dev/null -p 8022 ... 'if pgrep -af ...'` | fail recorded | Even the minimal bounded process-liveness check timed out, so SSH transport is currently too flaky to prove live-vs-gone from this side. |
| Local Coverage | `env GOCACHE=$PWD/.cache/go-build GOMODCACHE=$PWD/.cache/go-mod GOWORK=off GOTOOLCHAIN=local ./scripts/phase10_9c_phone_worker.sh coverage` | pass | After the second empty pull, local retained coverage remained unchanged at `partial_universe_only` with `72` summary files and `0` malformed summaries. |
| ETH Ops Probe | `timeout 8s ssh -F /dev/null -p 8022 ...` | pass | Remote ops inspection confirmed Case C: `ETHUSDT` worker is still live and `2024-01..2024-04` retained alpha summaries exist on the phone. |
| ETH Pull Attempt | `env SSH_CONFIG_FILE=/dev/null make termux-worker-pull-summaries` | pass | Summary pull brought back the four remote `ETHUSDT` retained alpha summaries. |
| ETH Validate | `jq empty runs/reports/chunks/ETHUSDT/*-alpha-summary.json` | pass | All four local `ETHUSDT` alpha summaries parse cleanly. |
| Local Coverage | `env GOCACHE=$PWD/.cache/go-build GOMODCACHE=$PWD/.cache/go-mod GOWORK=off GOTOOLCHAIN=local ./scripts/phase10_9c_phone_worker.sh coverage` | pass | Local retained coverage now includes `ETHUSDT` with months `2024-01..2024-04`; `summary files found` advanced to `76`, and remaining missing symbols are `ADAUSDT`, `AVAXUSDT`, `BNBUSDT`, `DOGEUSDT`. |

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
# Session: 2026-07-03 Phase 10.11B Completion

## Actions Taken
- Restored existing planning context and added the Phase 10.11B active addendum.
- Confirmed the local starting reports show `coverage_before=132/192` with missing confirmed-family coverage for `AVAXUSDT 2024-01..2025-12`, `SOLUSDT 2024-01..2025-12`, and `LINKUSDT 2025-01..2025-12`.
- Preserving pre-existing dirty local worktree changes; current task will only update 10.11B summaries/reports/planning unless a code change becomes necessary.

## Boundaries
- Do not modify `ak-trader`.
- Do not promote candidates.
- Do not add live trading/order/exchange-key/mainnet logic.
- Do not fetch new data if phone primary candles already exist.
- Do not use R2 restore in this phase.
- Pull compact summaries/reports/logs only back to the Chromebook.

## Verification Results
| Step | Command | Status | Notes |
|------|---------|--------|-------|
| Phone SSH | `ssh -F /dev/null -p 8022 ... 192.168.1.79` | fail/blocker | `ssh: connect to host 192.168.1.79 port 8022: No route to host`. |
| Phone SSH fallback | `ssh -F /dev/null -p 8022 ... 192.168.1.81` | fail/blocker | Timed out after 12 seconds. |
| Tailscale status | `tailscale status` | diagnostic | Phone nodes listed offline (`moto-g-5g---2023`, `moto-g-power-5g---2024`). |
| Local app tests | `env GOCACHE=$(pwd)/.cache/go-build GOMODCACHE=$(pwd)/.cache/go-mod GOWORK=off GOTOOLCHAIN=local go test ./internal/app` | pass | `ok github.com/davidmiguel22573/ak-engine/internal/app 12.418s`. |
| Chromebook raw audit | `find runs/reports/chunks -type f \\( -name '*.jsonl' -o -name '*.jsonl.gz' -o -name '*funding-events*' \\) | wc -l` | pass after cleanup | Count is `0`. |
| ak-trader status | `git -C /home/davidmiguel22573/Github/ak-trader status --short` | pass | No output; no ak-trader changes detected. |
| Report JSON validation | `jq empty runs/reports/phase10_11b_confirmed_funding_extreme_evaluation.json runs/reports/phase10_11b_confirmed_coverage_gap.json` | pass | Both refreshed report JSON files parse. |

## Phase 10.11B Outcome
- Phone worker became reachable again at `192.168.1.79:8022`; `.81` remained stale.
- Phone verification completed:
  - repo is git-backed
  - no stale phase10 worker was running before launch
  - confirmed-family source changes were present on phone
  - direct primary candle parquet files existed for `AVAXUSDT 2024-01..2025-12`, `SOLUSDT 2024-01..2025-12`, and `LINKUSDT 2025-01..2025-12`
  - disk had sufficient free space (`36G` observed before launch, `29G` after repair)
  - phone-side `go test ./internal/app` passed
- Regenerated the original missing 60 symbol-months on phone in order:
  - `AVAXUSDT 2024-01..2025-12`
  - `SOLUSDT 2024-01..2025-12`
  - `LINKUSDT 2025-01..2025-12`
- A broad `pull-summaries` pulled stale phone alpha summaries for some already-complete local symbols. To restore full confirmed coverage, repaired on phone and selectively pulled:
  - `ADAUSDT 2024-01..2025-12`
  - `BNBUSDT 2024-01..2025-12`
  - `DOGEUSDT 2024-01..2025-12`
  - `ETHUSDT 2024-01..2025-12`
  - `LINKUSDT 2024-01..2024-12`
- Updated final reports:
  - `runs/reports/phase10_11b_confirmed_funding_extreme_evaluation.json`
  - `runs/reports/phase10_11b_confirmed_funding_extreme_evaluation.md`
  - `runs/reports/phase10_11b_confirmed_coverage_gap.json`
  - `runs/reports/phase10_11b_confirmed_coverage_gap.md`
- Pulled logs:
  - `runs/reports/phase10_11b_phone_worker.log`
  - `runs/reports/phase10_11b_phone_repair_worker.log`
- `coverage_after=192/192`.
- `missing_after=[]`.
- `full_evaluation_complete=true`.
- Final labels:
  - `ConfirmedNegativeFundingLong|long|240m`: `REJECTED`, expectancy `-2.014372`, PF `0.966931`.
  - `ConfirmedPositiveFundingShort|short|5m`: `REJECTED`, expectancy `-5.164622`, PF `0.547049`.
- Strongest confirmed-family candidate: `ConfirmedNegativeFundingLong|long|240m`.
- No candidate became stronger in label or expectancy after full coverage; neither became `RESEARCH_LEAD` nor `SHADOW_CANDIDATE`.
- Local raw funding-event gzip files present at the start of this closeout were removed to restore Chromebook raw-free state; after final pullback, Chromebook raw count under `runs/reports/chunks` is `0`.
- Phone-side raw counts for touched symbols `AVAXUSDT`, `SOLUSDT`, `LINKUSDT`, `ADAUSDT`, `BNBUSDT`, `DOGEUSDT`, and `ETHUSDT` are all `0`.
- No R2 restore, no new data fetch, no threshold tuning, no candidate promotion, no ak-trader changes, and no live trading/order/exchange-key/mainnet code were added.

## Final Test Results
| Step | Command | Status | Notes |
|------|---------|--------|-------|
| Local app tests | `env GOCACHE=$(pwd)/.cache/go-build GOMODCACHE=$(pwd)/.cache/go-mod GOWORK=off GOTOOLCHAIN=local go test ./internal/app` | pass | `ok github.com/davidmiguel22573/ak-engine/internal/app`. |
| Full local tests | `env GOCACHE=$(pwd)/.cache/go-build GOMODCACHE=$(pwd)/.cache/go-mod GOWORK=off GOTOOLCHAIN=local go test ./...` | pass | All packages passed. |
| Final report JSON | `jq empty runs/reports/phase10_11b_confirmed_funding_extreme_evaluation.json runs/reports/phase10_11b_confirmed_coverage_gap.json` | pass | Both parse. |
| Chromebook raw audit | `find runs/reports/chunks -type f \\( -name '*.jsonl' -o -name '*.jsonl.gz' -o -name '*funding-events*' \\) | wc -l` | pass | `0`. |
| ak-trader status | `git -C /home/davidmiguel22573/Github/ak-trader status --short` | pass | No output. |
