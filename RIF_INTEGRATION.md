# RIF Integration — Superseded Current Guidance

Status: superseded by the reviewed Phase 1 Engine-to-RIF architecture. The historical paper-observation guidance is retained below for traceability; it is not proof of RIF acceptance, paper readiness, or authorization.

## Current boundary

Engine emits local research diagnostics only when the explicit, default-off `--research-diagnostics` flag is used. `validated_research_lead` is an Engine-local research classification. Change 2 diagnostics use `authority_status: NONE_RESEARCH_ONLY` and typed exact identity results. `eligible_for_rif_review` can be true only for complete clean evidence and means only locally complete enough to submit for future independent RIF review; no RIF acceptance endpoint exists.

RIF acceptance and durable lifecycle storage are not implemented. No path described in this file freezes a candidate, makes it paper-eligible, or authorizes a paper run. Paper, testnet, and mainnet execution remain blocked.

Legacy `research.lock`, `research_audit.json`, and promotion-packet files are historical untrusted diagnostics. Engine no longer produces them through `internal/rifbridge`, and no legacy file is upgraded in place. The existing paper-observation status strings are local scaffolding only; they do not establish RIF acceptance or execution permission and must not consume the new local research-diagnostics artifact as authorization.

Change 2 derives temporary exact candidate, configuration, source, Historian, feature/regime, consumed-input, and metric-series identities. Changes 3–6 remain required before shared canonical evidence or trustworthy RIF lifecycle authority exists.

## Historical guidance (superseded; preserved verbatim below)

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
