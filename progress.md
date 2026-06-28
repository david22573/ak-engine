# Progress Log

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
