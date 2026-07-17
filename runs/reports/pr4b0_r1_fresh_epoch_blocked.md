# PR4B0-R1 fresh epoch blocked closeout

## Executive verdict

Stage A failed closed before protocol registration. The accepted RIF baseline cannot create the mandatory closed FINAL_HOLDOUT reservation before DEVELOPMENT: its sole reservation API requires an already `VALIDATED` candidate with a matching positive lifecycle epoch, while fresh registration begins at `DISCOVERY`. Its accepted V3 research identity also cannot bind the complete protocol commit, variant ledger, three partition boundaries, repository baselines, source identity, and prohibited interval required by this epoch.

No candidate outcome was produced or read. DEVELOPMENT, VALIDATION, and FINAL_HOLDOUT remained closed. No real RIF state, freeze, paper path, trading path, or `ak-trader` change occurred.

Final label: `PR4B0_R1_RESEARCH_BLOCKED`

## Repository basis

| Repository | Clean starting commit | Result |
|---|---|---|
| Engine | `1d575f4b5331a640b1dc4fa46a2c281aa6a2487a` | the self commit containing this report |
| Historian | `14e0b569cc41e3517d2393d87d39c00559ac408e` | unchanged |
| RIF | `c5823e88d0f1fd8f25a93ee70c5a3490fc12dbcb` | unchanged |

The required Historian chain `3864a0... -> fba40d... -> a81d34... -> 14e0b5...` was verified with local commit-object and ordered ancestor checks.

## Verified authorities and checkpoint

- Independence V3: `sha256:84a6863b354b453dbe13698b9854ec4adcd116466a0831e7107efb892042cc1f`.
- Uncertainty V2: `sha256:1a91541c94378cc6f34e62a39ae504d3d013b5dab63a2b622641cdd1088148fb`.
- Concentration governance: `sha256:a126849e4cc0bd6457cf3f11079c4e3e2865ffce6c53a95ce92fa250130d39d5`.
- Checkpoint: `r1p5r-checkpoint-20260717T073628Z`, canonical `sha256:bef53d11aa9ce9a6dad61b89ef7ace063b6da812ff92208d463c6ecfbfe8f29c`.
- Reacquisition protocol: `sha256:7fd6c667d97a0ff3387e352d8c8e9b25ef5a744d641147ed66b98df75dcc0e1a`.
- Source identity: `sha256:d99a88a72b4bfe84c2ae43a4a477724b95fde2b45b381f7b75fc4c107d2a161a`.
- Pre-acquisition seal: `sha256:8046a306c80c127bd631df6eb4ae07ef587c350d4ac3a5f3be33f84a0f4681c9`.
- Sealed binary: `sha256:c10fdf10255a8c88c817d5189b20ca7411be1fcd2ae64df8c07e2d1934054ae6` from two original retained builds and one independent deterministic rebuild.
- Abandoned-evidence registry: `sha256:f8a47626a234544f34ae59846c330682e78943add69edb171c558287b35417ca`.

The exact sealed Historian binary revalidated all 197 consecutive complete all-symbol UTC days, 1,773 daily manifests, healthy backfill/live chains, and zero gaps, duplicates, conflicts, schema failures, evidence gaps, and clock errors. Its authoritative label remained `PR4B0_R1P5R_SOURCE_REPAIRED_AND_COVERAGE_REACQUIRED`.

Because evaluation never opened, the immutable checkpoint, four source-chain JSON files, and sealed binary were rehashed after the fail-closed decision and before this evidence commit. Every raw byte hash was unchanged.

## Availability cutoff and immutable partitions

The latest authoritative durable timestamp was checkpoint creation at `2026-07-17T07:36:28.276810652Z`; the derived first whole UTC second strictly later is `2026-07-17T07:36:29Z`. No protocol was registered because the RIF blocker was proven first.

| Partition | Exact half-open interval | Days | Opened |
|---|---|---:|---|
| DEVELOPMENT | `[2026-01-01T00:00:00Z, 2026-04-10T00:00:00Z)` | 99 | no |
| VALIDATION | `[2026-04-10T00:00:00Z, 2026-05-29T00:00:00Z)` | 49 | no |
| FINAL_HOLDOUT | `[2026-05-29T00:00:00Z, 2026-07-17T00:00:00Z)` | 49 | no |

## Protocol, V00, variants, and gates

Protocol identity, protocol hash, and protocol-only commit are `none`; producing them after a proven Stage A incompatibility would misrepresent an executable epoch.

No variant was registered or executed; variant count is 0 and nominee is `none`. The required Engine tree does not contain the referenced candidate source. The inventory points to clean commit `c2c7988712699b26ba7ab28e1cebb1f5312812a6`, where the source byte-hashes to `sha256:3c2e20fd5bf615864aebc5be35ce86c15a6ed8f83de33b2f1d33b00dae6fbfa1`, but that runner retains summaries only, has no registered context-agreement/event-quality/cooldown variant controls, requests barred pre-2026 warmup for a January 2026 run, and does not invoke independence V3 or uncertainty V2. V00 therefore was not silently reconstructed.

No partition gate was run. Concentration, independence V3, and uncertainty V2 identities were verified; their candidate results are `NOT_RUN`.

## Blocking evidence

1. `registry.HoldoutAccessAuthority.Reserve` is the only accepted reservation API. `normalizeFrozenHoldoutAccessIdentity` requires `VALIDATED` plus a positive lifecycle epoch, and request validation requires the persistent candidate to match. RIF registration begins at `DISCOVERY`. This cannot satisfy pre-DEVELOPMENT reservation ordering.
2. `core.ResearchIdentity` V3 lacks fields for the protocol identity/hash/commit, repository baselines, source-identity hash, complete variant ledger, separate three-partition boundaries, and prohibited interval. Overloading unrelated fields would be an unaccepted alias, which the protocol forbids.
3. The baseline candidate execution path is not authority-complete and cannot be repaired after registration without violating the semantic/code-ordering constraints.

These are integrity failures, not poor performance, so `PR4B0_R1_NO_CANDIDATE_QUALIFIED` would be false.

## Prohibited evidence and access

Legacy pre-2026 reports and references exist in the accepted Engine tree but remained quarantined and did not influence a protocol or variant choice. No quarantined R1P5 source, cache, checkpoint, or outcome was used. No new data was fetched. No symlink dependency was found. FINAL_HOLDOUT authorization/evaluation count is 0.

## Verification

- Engine canonical authority overlay test: PASS.
- Historian committed-authority byte test: PASS.
- Exact sealed Historian readiness scan: PASS.
- `GOWORK=off go test ./governance ./registry ./lifecycle -count=1`: PASS.
- `GOWORK=off go test ./internal/preconditions ./internal/qualification -count=1`: PASS.
- `GOWORK=off go vet ./internal/preconditions ./internal/qualification`: PASS.
- JSON parse, canonical artifact hash, Git diff audit, and clean post-commit checks are recorded at commit time.

The canonical machine-readable evidence, complete test commands, raw/canonical artifact hashes, V00 audit, gate contract, files changed, and blockers are in `runs/reports/pr4b0_r1_fresh_epoch_blocked.json`.
