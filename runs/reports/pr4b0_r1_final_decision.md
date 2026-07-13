# PR4B0-R1 Final Decision

## Executive verdict

The exact `DowntrendMidVolReliefLong240m` implementation and aggregate report reproduce byte-for-byte, but this research epoch cannot legally create partitions or variants. Immutable dataset/PIT identity is absent, the retained schema cannot replay events or independent clusters, and the accepted uncertainty method is unspecified.

Final label:

```text
PR4B0_R1_RESEARCH_BLOCKED
```

This is not a no-candidate result: the registered hypothesis was not evaluated.

## Commits and protocol

| Item | Identity |
|---|---|
| Accepted Engine source | `25efa97ca89f8dcb724f9872e798bc789123caac` |
| Accepted Engine report | `205cf59555006ce23fc58bc2c73262660a894850` |
| RIF authority | `29350344a57e46f064442eada26e9418515990be` |
| Historian authority | `3eeff1eb45da281e0003dc1577ec55aa6cda1b1b` |
| Protocol commit | `ce1377682975c8bc3d5b947d35900500b03403bf` |
| Protocol JSON SHA-256 | `7d1305bb418d2463c966f1383be4e9b7e30b0aa08dd254cd60eb55e7825cb072` |
| Result source commit | `PENDING_RESULT_SOURCE_COMMIT` |
| Final report commit | Resolve with `git log -1 -- runs/reports/pr4b0_r1_final_decision.md` |

The protocol commit directly follows the accepted report commit and contains exactly the two protocol artifacts. Results files were created only afterward.

## Dataset, PIT, and partitions

Dataset ID/version, manifest ID/hash, Historian PIT evidence ID/hash, evaluation cutoff, coverage-policy version, and availability-policy version are all `UNAVAILABLE`.

The only diagnostic local-source manifest covers 72 objects for three symbols, has no object checksums, row counts, or source availability timestamps, and is not accepted PIT evidence. It cannot identify the full replayed dataset.

| Partition | Exact boundary |
|---|---|
| DEVELOPMENT | `NOT_CREATED_PHASE_A_BLOCKED` |
| VALIDATION | `NOT_CREATED_PHASE_A_BLOCKED` |
| FINAL_HOLDOUT | `NOT_CREATED_PHASE_A_BLOCKED` |

No partition dates or gaps were invented.

## Baseline reproduction

The implementation at `c2c7988712699b26ba7ab28e1cebb1f5312812a6` has SHA-256 `3c2e20fd5bf615864aebc5be35ce86c15a6ed8f83de33b2f1d33b00dae6fbfa1`. Replay completed all 192 target symbol-months.

| Artifact | Result |
|---|---|
| Evaluation JSON | byte-identical, SHA-256 `fb27fe46ab1139ccafea3a7b3cbb7bfdfc7fb3bd2f7e545f1b7b566d2e6c9066` |
| Evaluation Markdown | SHA-256 `fba5947aad5dad971b24e634cb553dd8c4c3694f907bd3bdf0ef40060e95a9ed` |

| Cost | PF | Expectancy (bps) |
|---:|---:|---:|
| 5 bps | 1.168964 | 16.473089 |
| 7.5 bps | 1.141682 | 13.973089 |
| 10 bps | 1.115004 | 11.473089 |
| 15 bps | 1.063414 | 6.473089 |

Worst month: March 2025, PF `0.634442` at 10 bps. Worst quarter: Q1 2025, PF `0.813545` at 10 bps.

The 13,178 legacy spacing count is not accepted independent-cluster evidence.

## Variants, development, and validation

| Item | Count/status |
|---|---|
| Search budget | 12 maximum |
| Complete declared variants | 0 |
| Development results | 0, not run |
| Validation results | 0, not run |
| Walk-forward slices | 0, not run |
| Parameter-neighborhood results | 0, not run |

The baseline replay was a Phase A reproducibility check, not a candidate variant. No context, quality, cooldown, calendar, quarter, or symbol filter was evaluated.

## Holdout and frozen identity

RIF's accepted one-time exposure mechanism was verified, but no survivor existed to freeze. No candidate registration was requested, no ledger was created, no exposure was authorized or recorded, and no holdout data were read.

Qualified candidate identity: none. Frozen descriptor hash: none. Registration-request hash: none. Qualified-only parity fixtures: not produced.

## Qualification gates

| Gate | Threshold | Status | Evidence |
|---|---|---|---|
| Implementation reproducible | exact tolerance | PASS | JSON byte-identical; both report hashes match |
| Dataset/manifest/PIT identity | exact and verified | FAIL | identities unavailable |
| Independent decisions | >=300 | UNPROVEN | legacy spacing is not accepted clustering |
| Net PF at 10 bps | >=1.10 | BASELINE-ONLY PASS | 1.115004 unpartitioned; no variants |
| Net expectancy | >0; accepted minimum 0.01 bps | BASELINE-ONLY PASS | 11.473089 bps unpartitioned |
| Expectancy lower bound | >0 | FAIL/UNAVAILABLE | accepted method unspecified |
| Worst-period PF | >=0.95 | BASELINE FAIL | Q1 2025 is 0.813545 at 10 bps |
| OOS/validation | required | NOT RUN | no legal partitions |
| Walk-forward | required | NOT RUN | Phase A blocked |
| Concentration | 50% symbol, 50% temporal, 60% regime, accepted cluster limit | NOT EVALUABLE | event/cluster rows absent |
| Stable neighbors | >=2 | NOT RUN | zero variants |
| RIF final holdout | required after freeze | NOT REQUESTED | no freeze; holdout unread |

Any one mandatory failure prevents qualification. Aggregate PF does not bypass missing identity, cluster evidence, uncertainty, worst-period, OOS, walk-forward, concentration, parameter, or holdout gates.

## Security and integrity review

- Future-data and publication-delay leakage: blocking fail-closed result because accepted PIT/availability evidence is absent.
- Holdout inspection/reuse: pass; zero reads and zero exposures.
- Threshold, calendar, symbol, and variant overfitting: pass; zero variants and no Q1 exclusion or blacklist.
- Dataset/report/implementation substitution: exact source/report hashes verified; missing data identity caused a block rather than substitution.
- Cluster-count inflation/concentration masking: legacy count rejected and never used to qualify.
- Cost omission/fail-open logic: full 10 bps decomposition preserved and all mandatory failures remain disqualifying.

No qualified result is claimed while a high-severity readiness gap remains.

## Verification

Local verification passed: module tidy produced no diff; vet, tests, race tests, build, `make verify`, JSON validation, integrity scans, and `git diff --check` all passed. No Go file changed, so `gofmt` was not applicable.

The workspace parent contains an invalid empty `.git` directory, so linked-worktree Go commands used the documented `GOFLAGS=-buildvcs=false` workaround. Exact commands without that workaround remain pending the result-source commit and isolated clone. The final JSON records every command and exit status.

## Artifacts and scope statements

Generated paths:

- `runs/reports/pr4b0_r1_research_protocol.md`
- `runs/reports/pr4b0_r1_research_protocol.json`
- `runs/reports/pr4b0_r1_variant_results.md`
- `runs/reports/pr4b0_r1_variant_results.json`
- `runs/reports/pr4b0_r1_final_decision.md`
- `runs/reports/pr4b0_r1_final_decision.json`

Gates were not altered after results. No paper evaluator was implemented. No RIF paper authorization was issued. No trader behavior changed. No next phase was started.

## Recommendation

Do not start PR4B1 or PR4B0-R2. First create accepted immutable Historian PIT evidence for the full input set, retain a pre-result event/cluster schema capable of deterministic replay and allowed constraints, and recover or separately authorize an exact uncertainty method. Then rerun PR4B0-R1 as a new controlled epoch.
