# Findings

## Requirements
- Work only in `ak-engine`.
- Reduce research-script debt, phase-specific sprawl, and fragile pipeline orchestration.
- Do not change research math, scoring, event definitions, or cost assumptions.
- Preserve CLI commands and report schemas unless adding backward-compatible aliases.
- Keep low-resource constraints: monthly chunks, max rows, reports-only retention, resumable manifests, summary-only operation.
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

## Remaining Risks
- Many research and app files were already modified or untracked before this task; preserve them unless directly needed.
- The local smoke target does not include a take-profit exit, so historical metric impact for the TP-no-slip fix is bounded by code-path analysis and unit tests rather than by a paired TP before/after artifact in this workspace.
