# ak-engine
An engine for ak related extensions

## Development

Required Go version: `1.25.6`.

Standalone setup from a fresh clone:

```bash
git clone git@github.com:david22573/ak-engine.git
cd ak-engine
GOWORK=off go mod download
make verify
```

The first dependency download needs network access unless the module cache is already populated. After dependencies are available, `make verify` runs local formatting, vet, test, and build checks with `GOWORK=off`.

`go.work` is optional for local multi-repository development and is not required to build or test this repository. `ak-engine` does not import `ak-rif`.

## Engine research boundary

Engine performs deterministic local candidate evaluation. `validated_research_lead` means only that the candidate passed Engine's local research gates. It is not RIF acceptance, candidate freeze, paper eligibility, readiness, or authorization.

With the explicit, default-off `--research-diagnostics` flag, the optional bridge output is `<stem>.research_diagnostics.json`, using temporary schema `ak.engine.local_research_diagnostics` version 2. Every current diagnostic states:

- `authority_status: NONE_RESEARCH_ONLY`
- one exact typed identity status, such as `COMPLETE_RESEARCH_IDENTITY`, `DIRTY_ENGINE_SOURCE`, or a domain-specific incomplete/conflict status
- `eligible_for_rif_review: true` only for a complete, clean, cross-validated identity with passing local integrity and the unchanged `validated_research_lead` classification

Complete diagnostics bind the registered candidate and implementation inventory, resolved configuration, clean Engine commit/tree/build environment, strict Historian dataset/archive/PIT/coverage evidence, feature/regime artifacts and implementation inventories, exact consumed parquet rows, and the metric return/timestamp series. Incomplete, dirty, or conflicting derivations remain local and non-reviewable and contain no invented complete identity. Engine no longer produces the legacy flat `research.lock`, `research_audit.json`, or promotion-packet artifacts, and it does not parse promotion evidence.

RIF acceptance is not implemented. No candidate is durably frozen or paper-eligible through this path. Paper, testnet, and mainnet execution remain blocked.

Change 2 encodings are explicitly temporary and are not the draft cross-repository canonical contract. Changes 3–6 are still required for shared canonical evidence, atomic publication, strict RIF validation, and durable RIF lifecycle authority.
