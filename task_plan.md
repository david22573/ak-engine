# Task Plan: ak-engine Research Script Debt Refactor

## Goal
Reduce research-script debt, phase-specific sprawl, and fragile pipeline orchestration without changing research math, output semantics, existing CLI behavior, or report schemas.

## Constraints
- Work only in `ak-engine`.
- Do not change strategy formulas, research math, scoring logic, event definitions, or cost assumptions.
- Preserve existing CLI commands and report schemas unless adding backward-compatible aliases.
- Prefer move-only refactors, small helper extraction, and tests.
- Preserve low-resource behavior: monthly chunks, max rows, reports-only retention, resumable manifests, summary-only operation.
- Do not touch `ak-trader` or `ak-historian` except documenting expected path contracts.
- After every step, run the verification commands listed for that step.
- If a verification command fails, stop and report exact failure.

## Current Phase
Step 5 complete; verification closeout recorded for side-specific scoring, exit slippage, and state-retention fixes

## Phases

### Step 0: Capture Baseline
- [x] Run compile-only tests.
- [x] Run core package tests.
- [x] Run full go test and save `runs/refactor_baseline_go_test.txt`.
- [x] Build `./ak-engine` and run `./ak-engine version`.
- [x] Compile specified Python scripts.
- [x] Check shell scripts with `bash -n`.
- **Status:** complete

### Step 1: Fix Proof/Golden-Output Harness
- [x] Refactor app tests reading `../../runs/reports` to use `testdata` or `t.TempDir`.
- [x] Run required verification commands.
- **Status:** complete

### Step 2: Add Atomic Artifact Writing
- [x] Add temp-file plus rename helper for JSON/file writes.
- [x] Replace direct manifest/report writes in `phase10_low_resource_prep.go`.
- [x] Replace direct manifest/report writes in `phase10_funding_event_pipeline.go`.
- [x] Preserve exact JSON schemas.
- [x] Run required verification commands.
- **Status:** complete

### Step 3: Centralize Engine/Historian Path Contracts
- [x] Add helper resolving historian workdir from flag, env var, default.
- [x] Replace duplicated derivative path construction.
- [x] Run required verification commands.
- **Status:** complete

### Step 4: Formalize Leakage/Integrity Checks
- [x] Replace assumed PASS behavior with PASS/FAIL/UNKNOWN logic.
- [x] Add tests for PASS, FAIL, UNKNOWN.
- [x] Run required verification commands.
- **Status:** complete

### Step 5: Extract Phase10 Funding Pipeline Core
- [x] Extract config normalization.
- [x] Extract path building.
- [x] Extract manifest store.
- [x] Extract step runner.
- [x] Extract report rendering.
- [x] Run required verification commands after each extraction.
- **Status:** complete

### Verification Closeout: Side-Specific Scoring, Exit Slippage, and State Retention
- [x] Confirm side-specific scoring no longer cross-applies long and short overrides.
- [x] Add asymmetric tests for strict-long/lenient-short and strict-short/lenient-long cost and chop gating.
- [x] Confirm take-profit skips adverse exit slippage while stop-loss and non-TP exits retain conservative slippage.
- [x] Confirm completed-window trim-to-4 does not break hourly context, breakout-retest logic, decisions snapshot, or UTC day entry counting.
- [x] Run full package verification.
- [x] Regenerate one deterministic local backtest smoke report and record impact limits honestly.
- **Status:** complete

### Step 6: Add Manifest-Driven Research-Run Wrapper
- [ ] Add `research-run --plan <plan.json> --dry-run`.
- [ ] Add example funding research plan.
- [ ] Preserve existing scripts.
- [ ] Run required verification commands.
- **Status:** pending

### Step 7: Standardize Report Prefix/Alias Behavior
- [ ] Add or centralize `--report-prefix` where appropriate.
- [ ] Preserve old report names.
- [ ] Add report alias manifest before replacing script-level copy behavior.
- [ ] Run required verification commands.
- **Status:** pending

### Step 8: Split Large Analyzers Safely
- [ ] Extract pure functions from listed analyzers.
- [ ] Target metrics, gates, renderers, and report schemas.
- [ ] Preserve formulas.
- [ ] Run required verification commands.
- **Status:** pending

### Step 9: Add Raw-Event vs Summary-Only Equivalence Tests
- [ ] Create tiny synthetic fixtures.
- [ ] Assert raw JSONL and native summary metrics match.
- [ ] Run required verification commands.
- **Status:** pending

### Step 10: Clean Command Stubs and Script Status
- [ ] Mark unimplemented commands deprecated or return explicit non-success errors.
- [ ] Add command inventory check if practical.
- [ ] Run required verification commands.
- **Status:** pending

## Decisions Made
| Decision | Rationale |
|----------|-----------|
| Preserve existing dirty worktree changes | User scope is refactor within `ak-engine`; unrelated or pre-existing edits must not be reverted. |

## Errors Encountered
| Step | Error | Resolution |
|------|-------|------------|
| Step 0 | `go test ./...` failed in `internal/app` because tests read missing `../../runs/reports/...` files. | Recorded to `runs/refactor_baseline_go_test.txt`; proceed to Step 1 per user expectation. |
| Step 1 | Initial patch targeted parent workspace paths instead of `ak-engine/...`. | Retried with scoped paths; no source change from failed patch. |
| Step 1 | Planning update first wrote to workspace root planning files. | Restored root planning files to prior contents and wrote refactor plan to `ak-engine`. |
| Step 3 | `grep -R "../ak-historian/.ak-historian/work" -n . || true` still reports historical generated artifacts, caches, and old planning data. | Source Go/shell/Python script references were replaced; command exits 0 by design. |
| Step 4 | First V2 aggregate consistency check treated positive `gross_loss_bps` as failure. | Inspected generated rows; gross loss is stored as positive magnitude, so sign check was removed. |
| Verification Closeout | Direct `GOWORK=off GOTOOLCHAIN=local go test ./...` hit read-only host Go build cache under `~/.cache/go-build`. | Re-ran equivalent suite with `GOCACHE` and `GOMODCACHE` redirected into workspace-local `.cache/`; full suite passed. |
