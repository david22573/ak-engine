# PR4B0-R1 Variant Results

## Verdict

`PR4B0_R1_RESEARCH_BLOCKED`

The exact baseline is byte-reproducible, but Phase A cannot establish immutable dataset/PIT identity, replayable retained events and clusters, or the accepted uncertainty method. The epoch stopped before partitions or variants.

## Protocol proof

- Protocol JSON SHA-256: `7d1305bb418d2463c966f1383be4e9b7e30b0aa08dd254cd60eb55e7825cb072`
- Protocol commit: `ce1377682975c8bc3d5b947d35900500b03403bf`
- The commit contains only the two protocol artifacts and directly follows the accepted PR4B0 report commit.
- No variant-results or final-decision artifact existed at that commit.

## Search budget and variants

| Item | Value |
|---|---:|
| Maximum variants | 12 |
| Declared variants | 0 |
| Development variants run | 0 |
| Validation variants run | 0 |
| Holdout variants run | 0 |

Complete declared variant list: empty. The Phase A baseline reproduction is an implementation-readiness replay, not a variant evaluation.

## Baseline reproduction

| Evidence | Result |
|---|---|
| Implementation commit | `c2c7988712699b26ba7ab28e1cebb1f5312812a6` |
| Implementation SHA-256 | `3c2e20fd5bf615864aebc5be35ce86c15a6ed8f83de33b2f1d33b00dae6fbfa1` |
| Replayed coverage | 192/192 target symbol-months |
| JSON SHA-256 | `fb27fe46ab1139ccafea3a7b3cbb7bfdfc7fb3bd2f7e545f1b7b566d2e6c9066` |
| Markdown SHA-256 | `fba5947aad5dad971b24e634cb553dd8c4c3694f907bd3bdf0ef40060e95a9ed` |
| JSON byte equality | PASS |

| Cost | PF | Expectancy (bps) |
|---:|---:|---:|
| 5 bps | 1.168964 | 16.473089 |
| 7.5 bps | 1.141682 | 13.973089 |
| 10 bps | 1.115004 | 11.473089 |
| 15 bps | 1.063414 | 6.473089 |

Worst month is March 2025: PF `0.661711` at 5 bps and `0.634442` at 10 bps. Worst quarter is Q1 2025: PF `0.847931` at 5 bps and `0.813545` at 10 bps.

The 329,842 raw events do not establish 300 independent decisions. The legacy 13,178 count is only per-symbol-month one-hour spacing without cluster P&L or cross-symbol common-market treatment.

## Development, validation, and walk-forward

| Stage | Status | Results |
|---|---|---:|
| Development | `NOT_RUN_PHASE_A_BLOCKED` | 0 |
| Validation | `NOT_RUN_NO_DEVELOPMENT_SURVIVORS` | 0 |
| Walk-forward | `NOT_RUN_PHASE_A_BLOCKED` | 0 |
| Parameter neighborhood | `NOT_RUN_PHASE_A_BLOCKED` | 0 |
| Combined | `NOT_COMPUTED` | 0 |

No period, symbol, regime, cluster, exclusion, drawdown, uncertainty, or concentration result was manufactured from aggregate summaries.

## Blocking evidence

1. `IMMUTABLE_DATASET_IDENTITY_NOT_ESTABLISHED`
2. `COMPLETE_PIT_VALID_DATA_UNAVAILABLE`
3. `CANDIDATE_EVALUATION_BLOCKED_BY_RETAINED_SCHEMA`
4. `ACCEPTED_UNCERTAINTY_METHODOLOGY_UNSPECIFIED`

## Holdout

The accepted RIF exposure mechanism is available, but no candidate was frozen or registered. No ledger was created, no authorization or exposure was recorded, and no holdout data were read.

## Integrity statements

No gates were altered after results; in fact no variant result was produced. No paper evaluator was implemented, no RIF paper authorization was issued, no trader behavior changed, no holdout was inspected, and no failing variant was promoted.

Local module tidy, vet, tests, race tests, build, `make verify`, JSON validation, integrity scans, and diff checks passed. The exact no-workaround suite also passed in a clean no-sibling clone at `8f4df1e61455541262cc1c95e6a32e6b8948f980`.
