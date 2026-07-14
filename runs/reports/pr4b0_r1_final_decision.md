# PR4B0-R1 Final Decision

## Executive verdict

Legacy aggregate baseline metrics were reproduced exactly from the available accepted summary artifacts. Event-level and canonical-input reproduction could not be established. This research epoch cannot legally create partitions or variants because immutable dataset/PIT identity is absent, the retained schema cannot replay events or independent clusters, and the accepted uncertainty method is unspecified.

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
| Result source commit | `8f4df1e61455541262cc1c95e6a32e6b8948f980` |
| Final report commit | Resolve with `git log -1 -- runs/reports/pr4b0_r1_final_decision.md` |

The protocol commit directly follows the accepted report commit and contains exactly the two protocol artifacts. Results files were created only afterward. That Git artifact ordering does not prove execution ordering: the successful A1 readiness calculation completed before the protocol commit. Exact command/timestamp evidence and the resulting qualification limitation are recorded in `pr4b0_r1_evidence_supplement.{md,json}`.

## Dataset, PIT, and partitions

Dataset ID/version, manifest ID/hash, Historian PIT evidence ID/hash, evaluation cutoff, coverage-policy version, and availability-policy version were not recovered. The supplement inventories every searched path and the concrete evidence for each missing field.

The only diagnostic local-source manifest covers 72 objects for three symbols, has no object checksums, row counts, or source availability timestamps, and is not accepted PIT evidence. It cannot identify the full replayed dataset.

| Partition | Exact boundary |
|---|---|
| DEVELOPMENT | `NOT_CREATED_PHASE_A_BLOCKED` |
| VALIDATION | `NOT_CREATED_PHASE_A_BLOCKED` |
| FINAL_HOLDOUT | `NOT_CREATED_PHASE_A_BLOCKED` |

No partition dates or gaps were invented.

## Baseline reproduction

The implementation identity at `c2c7988712699b26ba7ab28e1cebb1f5312812a6` has SHA-256 `3c2e20fd5bf615864aebc5be35ce86c15a6ed8f83de33b2f1d33b00dae6fbfa1`. A1 ran that source against available local files and generated a report with all 192 target symbol-months, but missing input hashes and PIT identity prevent calling those files canonical or the run an event-level reproduction.

| Artifact | Result |
|---|---|
| Evaluation JSON | Original/reproduced SHA-256 `fb27fe46ab1139ccafea3a7b3cbb7bfdfc7fb3bd2f7e545f1b7b566d2e6c9066`; explicit `cmp` PASS |
| Evaluation Markdown | Original/reproduced SHA-256 `fba5947aad5dad971b24e634cb553dd8c4c3694f907bd3bdf0ef40060e95a9ed`; no separate `cmp` recorded |

| Cost | PF | Expectancy (bps) |
|---:|---:|---:|
| 5 bps | 1.168964 | 16.473089 |
| 7.5 bps | 1.141682 | 13.973089 |
| 10 bps | 1.115004 | 11.473089 |
| 15 bps | 1.063414 | 6.473089 |

Worst month: March 2025, PF `0.634442` at 10 bps. Worst quarter: Q1 2025, PF `0.813545` at 10 bps.

The values above are reaggregated from the legacy monthly summaries; formulas and exact gross/net totals are in the supplement. They are not legal DEVELOPMENT, VALIDATION, or FINAL_HOLDOUT results. The 13,178 legacy spacing count is not accepted independent-cluster evidence.

## Variants, development, and validation

| Item | Count/status |
|---|---|
| Search budget | 12 maximum total |
| Declared modified variants | 0 |
| Executed modified variants | 0 |
| Development results | 0, not run |
| Validation results | 0, not run |
| Walk-forward slices | 0, not run |
| Parameter-neighborhood results | 0, not run |

The A1 calculation was treated as a readiness check, not V00. It completed before protocol freeze and calculated legacy unpartitioned strategy aggregates, so those aggregates cannot contribute to qualification. The `0/0` count means zero declared and zero executed modified variants. No context, quality, cooldown, calendar, quarter, or symbol filter was evaluated.

## Holdout and frozen identity

RIF's accepted one-time exposure mechanism was verified, but no survivor existed to freeze. No candidate registration was requested, no ledger was created, no exposure was authorized or recorded, and no holdout data were read.

Qualified candidate identity: none. Frozen descriptor hash: none. Registration-request hash: none. Qualified-only parity fixtures: not produced.

## Qualification gates

| Gate | Threshold | Status | Evidence |
|---|---|---|---|
| Implementation identity | exact source identity | PASS | commit/path/source hash recovered |
| Legacy report artifact equality | exact artifact evidence | PASS | JSON hash + `cmp`; Markdown matching hashes |
| Dataset/manifest/PIT identity | exact and verified | FAIL | concrete recovery inventory found no candidate identity |
| Event replay and canonical clustering | required | BLOCKED | event/cluster rows were not retained |
| Independent decisions | >=300 | BLOCKED | legacy spacing sum is not canonical clustering |
| Net PF/expectancy at 10 bps | PF >=1.10; expectancy >=0.01 bps | NOT RUN | legacy aggregate is not a legal partition result |
| Expectancy lower bound | >0 | BLOCKED | accepted method unspecified |
| Worst-period PF | >=0.95 | NOT RUN | legacy Q1 value is not a legal partition result |
| OOS/validation | required | NOT RUN | no legal partitions |
| Walk-forward | required | NOT RUN | no legal partitions |
| Concentration and neighbors | accepted limits; >=2 neighbors | NOT RUN | zero variants; event/cluster evidence absent |
| RIF final holdout | required after freeze | NOT RUN | no freeze; holdout unread |

The supplement contains every accepted mandatory gate separately, using only `PASS`, `FAIL`, `BLOCKED`, or `NOT RUN`, with methodology, scope, exact evidence, and qualification contribution.

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

The workspace parent contains an invalid empty `.git` directory, so linked-worktree Go commands used the documented `GOFLAGS=-buildvcs=false` workaround. Exact commands without that workaround passed in an isolated no-sibling clone at `8f4df1e61455541262cc1c95e6a32e6b8948f980`: tidy/no module diff, vet, tests, race tests, build, `make verify`, JSON/hash checks, integrity scans, `git diff --check`, and clean-tree verification all returned zero. The final JSON records every command and exit status.

The documentation-closeout suite was repeated locally and in an isolated no-sibling clone of the follow-up commit. All checks passed except that the original absolute-host-path scan intentionally found the two required exact-command fields in the supplement; a narrow assertion proved there were exactly two and no others. Full evidence is in the supplement.

## Artifacts and scope statements

Generated paths:

- `runs/reports/pr4b0_r1_research_protocol.md`
- `runs/reports/pr4b0_r1_research_protocol.json`
- `runs/reports/pr4b0_r1_variant_results.md`
- `runs/reports/pr4b0_r1_variant_results.json`
- `runs/reports/pr4b0_r1_final_decision.md`
- `runs/reports/pr4b0_r1_final_decision.json`
- `runs/reports/pr4b0_r1_evidence_supplement.md`
- `runs/reports/pr4b0_r1_evidence_supplement.json`

Gates were not altered after results. No paper evaluator was implemented. No RIF paper authorization was issued. No trader behavior changed. No next phase was started.

## Recommendation

Do not start PR4B1 or PR4B0-R2. First create accepted immutable Historian PIT evidence for the full input set, retain a pre-result event/cluster schema capable of deterministic replay and allowed constraints, and recover or separately authorize an exact uncertainty method. Then rerun PR4B0-R1 as a new controlled epoch.
