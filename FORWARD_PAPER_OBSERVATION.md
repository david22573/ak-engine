# Forward Paper Observation Loop V1

Forward paper observation repeatedly records RIF-gated paper signals from existing `ak-engine` candidates, grades them after their observation window, and reports whether the sample is still too small for shadow-mode review.

This workflow is paper-only. It does not connect to brokers, does not place orders, does not call private APIs, and does not change strategy logic.

## Generate Signals

Use `paper-forward` with an existing candidate id and a read-only dataset manifest:

```bash
ak-engine paper-forward \
  --candidate NegativeFundingLong \
  --symbols BTCUSDT,ETHUSDT \
  --timeframe 1m \
  --dataset-manifest runs/manifests/dataset_manifest.json \
  --snapshot-dir runs/scout/snapshots \
  --out-dir runs/paper/forward \
  --journal runs/paper/signals/paper_signal_journal.jsonl \
  --mode PAPER_FORWARD \
  --max-signals 1 \
  --paper-only true
```

The command writes per-signal `*_paper_signal.json` and `*_paper_signal.md` files, updates the compatibility `paper_signal.json` and `paper_signal.md` files, appends JSONL rows to the paper journal, and emits `forward_paper_observation_run.json`.

If RIF blocks the candidate or dataset manifest, the signal artifact is still written with `PAPER_SIGNAL_BLOCKED_BY_RIF`, but it is not counted as actionable and has no pending outcome.

## Grade Pending Outcomes

After `outcome_due_at_utc` has passed, grade due rows with local read-only market data:

```bash
ak-engine paper-forward-grade-pending \
  --journal runs/paper/signals/paper_signal_journal.jsonl \
  --market-data-root runs/paper/market_data \
  --out-dir runs/paper/outcomes \
  --max-grade 50
```

The grader scans `PENDING` rows, ignores rows that are not due, skips blocked rows, and grades only due rows with available future candles. If future data is missing, the due row is marked `INSUFFICIENT_DATA`; no outcome is invented.

Supported local market data is JSON candle data accepted by `internal/data.ParseJSONCandlesNoValidate`, for example `BTCUSDT_1m.json` under the market-data root.

## Review And Readiness

Journal review:

```bash
ak-engine paper-signal-review \
  --journal runs/paper/signals/paper_signal_journal.jsonl \
  --out runs/reports/paper_signal_review.md \
  --json-out runs/reports/paper_signal_review.json
```

Shadow-readiness report:

```bash
ak-engine paper-shadow-readiness \
  --journal runs/paper/signals/paper_signal_journal.jsonl \
  --candidate NegativeFundingLong \
  --out runs/reports/paper_shadow_readiness.md \
  --json-out runs/reports/paper_shadow_readiness.json
```

Readiness labels are capped at shadow mode:

- `<30` graded outcomes: `PAPER_INSUFFICIENT_SAMPLE` and `BLOCKED_BY_SAMPLE_SIZE`.
- `30-99` graded outcomes: `PAPER_EARLY_SAMPLE` and usually `CONTINUE_PAPER`, unless paper results are clearly bad.
- `>=100` graded outcomes: `PAPER_CALIBRATION_READY`; a candidate may become `SHADOW_CANDIDATE` only if RIF gates pass and paper outcomes remain positive enough.

Fewer than 30 graded outcomes is insufficient because a handful of forward observations cannot separate real behavior from noise. The loop must continue until the journal contains enough due, graded, file-backed outcomes to support a review.

## Safety Check

Run the import-level isolation check:

```bash
ak-engine paper-forward-safety-check \
  --out runs/reports/forward_paper_safety_check.md \
  --json-out runs/reports/forward_paper_safety_check.json
```

The check parses imports for the paper-forward runner, pending grader, readiness report, and paper signal schema. It fails if broker, execution, order, signing, secrets, credentials, or `ak-trader` imports are introduced.

## Smoke Flow

The deterministic smoke script exercises the full paper-only loop:

```bash
./runs/test_forward_paper_observation_flow.sh
```

It creates fixture manifests and local candle data, generates one allowed signal, generates one RIF-blocked signal, grades the due allowed signal, reviews the journal, emits readiness, runs the safety check, and validates generated JSON with `jq`.
