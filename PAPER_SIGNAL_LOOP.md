# Paper Signal Loop

The paper signal loop records forward-facing observations from existing `ak-engine` candidates without touching execution systems.

## What Paper Signals Are

- Local artifacts that describe an observed candidate condition.
- RIF-gated records with dataset, universe, lifecycle, and point-in-time evidence attached when available.
- JSON and Markdown files plus a JSONL journal row for repeatable review.

## What Paper Signals Are Not

- They are not orders.
- They are not `ak-trader` instructions.
- They do not use broker APIs, authenticated APIs, signing packages, secrets, credentials, or mainnet execution logic.
- They do not tune or modify candidate strategy logic.

## One-Off Signal Commands

Generate one RIF-gated signal:

```bash
ak-engine paper-signal \
  --candidate NegativeFundingLong \
  --symbol BTCUSDT \
  --market-type SPOT \
  --timeframe 1m \
  --dataset-manifest runs/manifests/dataset_manifest.json \
  --out-dir runs/paper/signals \
  --journal runs/paper/signals/paper_signal_journal.jsonl \
  --paper-only true
```

Grade a specific signal artifact:

```bash
ak-engine paper-signal-grade \
  --signal runs/paper/signals/paper_signal.json \
  --market-data runs/paper/market_data/BTCUSDT_1m.json \
  --out runs/paper/outcomes \
  --journal runs/paper/signals/paper_signal_journal.jsonl
```

Review the journal:

```bash
ak-engine paper-signal-review \
  --journal runs/paper/signals/paper_signal_journal.jsonl \
  --out runs/reports/paper_signal_review.md \
  --json-out runs/reports/paper_signal_review.json
```

## Forward Observation Commands

Use these commands for repeated paper observation:

- `paper-forward`: generates up to `--max-signals` RIF-gated paper signals for existing candidates and appends journal rows.
- `paper-forward-grade-pending`: scans the journal, grades due pending rows from local future candles, and leaves not-yet-due rows pending.
- `paper-shadow-readiness`: summarizes allowed vs blocked, graded vs pending, outcome distribution, RIF blockers, sample-size label, and shadow-readiness label.
- `paper-forward-safety-check`: verifies that the paper-forward implementation has no forbidden execution imports.

See [FORWARD_PAPER_OBSERVATION.md](./FORWARD_PAPER_OBSERVATION.md) for the full repeated-observation workflow.

## Sample Size

The paper journal must reach meaningful sample size before any shadow-mode review:

- `<30` graded outcomes remains `BLOCKED_BY_SAMPLE_SIZE`.
- `30-99` graded outcomes remains early paper observation.
- `>=100` graded outcomes can only become `SHADOW_CANDIDATE` if RIF gates pass and paper results are acceptable.
