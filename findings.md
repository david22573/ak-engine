# Findings

## Requirements
- Work only in `ak-engine`.
- Reduce research-script debt, phase-specific sprawl, and fragile pipeline orchestration.
- Do not change research math, scoring, event definitions, or cost assumptions.
- Preserve CLI commands and report schemas unless adding backward-compatible aliases.
- Keep low-resource constraints: monthly chunks, max rows, reports-only retention, resumable manifests, summary-only operation.
- Respect low-storage execution constraints from pasted setup guidance:
  - Treat the Chromebook as planner/reviewer plus final artifact holder, not the raw-regeneration worker.
  - Treat the Termux phone as the heavy batch worker for monthly chunk generation, temporary raw event files, caches, and long tmux jobs.
  - Run symbol-by-symbol and month-by-month; do not run all symbols and months in one raw job.
  - Retain only compact `*-alpha-summary.json` files and final phase reports; delete heavy raw `*.jsonl`, `*.jsonl.gz`, and `*events*.json` files after summaries are produced.
  - Keep final outputs under `runs/reports` or `runs/reports/chunks`, never only in `/tmp`.
- Use repo-local Go caches when running on storage-constrained or sandboxed environments:
  - `GOCACHE=$PWD/.cache/go-build`
  - `GOMODCACHE=$PWD/.cache/go-mod`
  - `GOWORK=off`
  - `GOTOOLCHAIN=local`
- Shared agent context for the Termux phone worker:
  - SSH target: `davidmiguel22573@192.168.1.79` on port `8022`
  - Key-based SSH is installed; password should not be needed for routine access now
  - Standard remote repo base: `$HOME/Github`
  - Standard remote repos present: `ak-engine`, `ak-historian`, `ak-scout`, `ak-trader`
  - Standard remote `ak-engine` path: `$HOME/Github/ak-engine`
  - Standard remote tmux session for long runs: `ak-engine-109c`
- Stop and report exact verification failure before patching around it.

## Baseline Context
- Worktree was already dirty with many modified and untracked `ak-engine` files.
- Pre-existing `ak-engine` plan files described Phase 10.7N-R and are superseded by this task.

## Discoveries
- Step 0 compile-only tests and core package tests pass.
- Step 0 full `go test ./...` fails only in `internal/app` tests reading generated reports:
  - `evaluate_alpha_baselines_multisymbol_test.go` reads `../../runs/reports/phase10_4_price_regime_branch_closure.json`.
  - `evaluate_alpha_baselines_multisymbol_test.go` reads `../../runs/reports/phase10_4_research_guardrails.md`.
  - `plan_compression_breakout_oos_test.go` reads `../../runs/reports/phase10_0_compression_breakout_long_spec.json`.
- Baseline output stored at `runs/refactor_baseline_go_test.txt`.
- Step 1 app tests now use repository-local test fixtures instead of generated `runs/reports` files.
- Step 1 verification fully passes, including `go test ./...`.
- Step 2 added atomic file writes without changing report JSON fields or research calculations.
- Existing `writeFundingJSONReport` now writes atomically, preserving indented JSON encoding.
- Step 3 centralized historian workdir resolution in `internal/app/historian_paths.go`.
- Step 3 source path contract is now explicit: flag value wins when `--workdir` is provided, then `AK_HISTORIAN_WORKDIR`, then `.ak-historian/work`.
- Step 3 grep still finds old generated artifacts, old planning files, and cache/binary matches containing historical sibling-historian paths.
- Step 4 `audit-v2-integrity` now reports `UNKNOWN` for `v2_no_stale_10_5_artifacts` because current native summary V2 rows do not contain source-phase provenance.
- Step 4 generated V2 rows pass aggregate row consistency checks once gross-loss sign is not assumed.
- Side-specific scoring bug source was shared pre-gating in Fast Accumulation: shared scoring helpers collapsed `LongCostMultipleRequired` and `ShortCostMultipleRequired` into one effective threshold and collapsed `LongMaxChopScore` and `ShortMaxChopScore` into one effective chop bound before side selection.
- Fix approach was to keep `ScoreWindow` reason annotation intact but move cost/chop hard-block enforcement to side-aware `selectDecision` logic so long overrides affect only longs and short overrides affect only shorts.
- Exit slippage is now explicitly conditional on exit reason: take-profit skips adverse exit slippage, while stop-loss, strategy close, time stop, and end-of-data still apply conservative exit slippage.
- Partial take-profit accounting now separates structural, fill, and net R-multiples so no-slip TP handling does not blur report semantics.
- `completedWindows` can be trimmed to 4 without losing required context for `BuildHourlyContext`, breakout-retest entry checks, or `DecisionsSnapshot`.
- Rolling day entry counting now uses UTC day keys and matches prior `entryCountOnDay` behavior across day boundaries while avoiding rescans on every check.
- Deterministic local smoke report using `testdata/candles/btc_5m_fast_accumulation_sample.json` under `fast_accumulation_strict` produced one stop-loss trade, so it does not exercise the TP-no-slip path; that smoke is valid for regeneration proof but not for before/after TP metric deltas.
- `analyze-compact-robustness` needed an explicit `--coverage-only` mode for Phase 10.9C because the retained-coverage recovery workflow must still emit reports even when no target `NegativeFundingLong/long` candidate exists.
- The first live phone-worker helper run exposed a wrapper bug: `run-symbol` incorrectly passed `--max-months 1`, which caused a full-range symbol run to stop after rebuilding a single month and then fail aggregate with `pipeline_blocked_missing_ephemeral_chunks`.
- The next live phone-worker blocker was external: Termux had the monthly funding-rate parquet files for all seven missing symbols, but did not have `duckdb` installed, so `join-research-features` could not read parquet derivatives.
- After installing Termux `duckdb`, the live `LINKUSDT` worker run began progressing correctly month-by-month while retaining `*-alpha-summary.json`, `*-funding-summary.json`, `*-funding-diagnostics.json`, and `*-context-audit.json`, and deleting the three heavy per-month JSON intermediates.
- Empirical live worker cleanup evidence for `LINKUSDT`:
  - `2024-01`: `summary_status=PASS`, `event_rows=78278`, `bytes_freed=141255143`
  - `2024-02`: `summary_status=PASS`, `event_rows=56388`, `bytes_freed=132447708`
  - `2024-03`: `summary_status=PASS`, `event_rows=30158`, `bytes_freed=142844460`

## Remaining Risks
- Many research and app files were already modified or untracked before this task; preserve them unless directly needed.
- The local smoke target does not include a take-profit exit, so historical metric impact for the TP-no-slip fix is bounded by code-path analysis and unit tests rather than by a paired TP before/after artifact in this workspace.
- Heavy multi-symbol raw regeneration is a poor fit for the Chromebook storage budget; future phase 10.8C/10.9C style work should prefer SSH/tmux execution on the Termux phone worker and copy back only summaries/reports.
- Local sandboxed SSH behavior can differ from unsandboxed execution because of host SSH config and missing local `rsync`; the sync helper now supports `tar` over SSH fallback, and out-of-sandbox execution may still be needed for some live remote operations.
- The exact low-storage pipeline contract is subtle: per-chunk event JSONL cannot be disabled up front because pipeline verification reads it before cleanup; the correct storage-safe path is to retain event detail during the run and use `--summary-only-after-aggregate` to delete raw event files after aggregate reports are written.
- The active `LINKUSDT` worker run is still in flight; full-universe readiness cannot be decided until the phone-side run completes, raw event files are removed after aggregate, compact outputs are pulled back, and Chromebook retained coverage is rerun.
