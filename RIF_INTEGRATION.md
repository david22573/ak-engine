# RIF Integration

The Research Integrity Framework (RIF) is the gate between research artifacts and any paper observation record that could later be reviewed for shadow mode.

## Dataset Evidence

Paper observation reads the supplied `dataset_manifest.json` and attaches the available evidence to every signal:

- `dataset_hash`
- `manifest_hash`
- `universe_hash`
- `lifecycle_hash`
- `point_in_time_coverage_hash`
- RIF status and warnings

Missing manifest evidence blocks an actionable paper signal. A blocked signal is still written as an artifact and journal row so the blocked state is auditable.

## PIT Eligibility

The paper-forward RIF gate blocks signals when the manifest indicates unsafe point-in-time evidence, including:

- missing dataset manifest
- malformed or unreadable dataset manifest
- `point_in_time_coverage_status` of `PIT_NOT_ELIGIBLE`, `NOT_ELIGIBLE`, `CURRENT_ONLY`, or `UNKNOWN`
- `point_in_time_promotion_recommendation` of `BLOCK_STRICT_PROMOTION`
- current-only exchange metadata snapshot evidence
- validation status of `FAIL` or `ERROR`

When only non-blocking RIF warnings are present, `--allow-rif-warnings=false` keeps the signal blocked. Setting `--allow-rif-warnings=true` permits non-blocking warnings only; it does not bypass hard RIF blocks.

## Forward Paper Observation

`ak-engine paper-forward` emits either:

- `PAPER_SIGNAL_ALLOWED`: actionable only for paper observation and appended with a `PENDING` outcome.
- `PAPER_SIGNAL_BLOCKED_BY_RIF`: non-actionable, not pending, and counted in RIF block summaries.

`paper-shadow-readiness` cannot advance a candidate with unresolved RIF blockers to `SHADOW_CANDIDATE`.

## Execution Isolation

RIF-gated paper observation remains separate from execution. The `paper-forward-safety-check` command parses paper-forward imports and fails if broker, execution, order, signing, secrets, credentials, or `ak-trader` imports are introduced.
