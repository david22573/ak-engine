# PR4B0-R1 Research Protocol

## Protocol status

`BLOCKED_DURING_PHASE_A_NON_EXECUTABLE`

Authoritative JSON SHA-256: `7d1305bb418d2463c966f1383be4e9b7e30b0aa08dd254cd60eb55e7825cb072`

This artifact records the pre-result readiness decision for `DowntrendMidVolReliefLong240m`. It is deliberately non-executable: Phase A proved that the baseline implementation is reproducible, but the immutable research identity, PIT evidence, retained event/cluster data, and accepted uncertainty method needed to create partitions or variants are unavailable.

No development, validation, walk-forward, or final-holdout data may be read through this protocol.

## Accepted authorities

| Authority | Commit |
|---|---|
| Engine accepted source | `25efa97ca89f8dcb724f9872e798bc789123caac` |
| Engine accepted report | `205cf59555006ce23fc58bc2c73262660a894850` |
| RIF | `29350344a57e46f064442eada26e9418515990be` |
| Historian | `3eeff1eb45da281e0003dc1577ec55aa6cda1b1b` |

## Baseline reproduction

The exact implementation at `c2c7988712699b26ba7ab28e1cebb1f5312812a6` was replayed over the original 8-symbol, 24-month inputs. The regenerated JSON and Markdown are byte-equivalent to the committed originals:

| Artifact | SHA-256 |
|---|---|
| Implementation | `3c2e20fd5bf615864aebc5be35ce86c15a6ed8f83de33b2f1d33b00dae6fbfa1` |
| Evaluation JSON | `fb27fe46ab1139ccafea3a7b3cbb7bfdfc7fb3bd2f7e545f1b7b566d2e6c9066` |
| Evaluation Markdown | `fba5947aad5dad971b24e634cb553dd8c4c3694f907bd3bdf0ef40060e95a9ed` |

The implementation is LONG-only on 1-minute futures candles with a 240-minute evaluation horizon. It requires BTC/ETH feature context and warm-up input. Its effective signal is `Close < EMA50 < EMA200`, negative `TrendSlope20`, and `RealizedVol60` in `[0.0015, 0.006]`; the legacy named regime/pullback/continuation gates then pass unconditionally.

Baseline aggregate PF is `1.168964 / 1.141682 / 1.115004` at `5 / 7.5 / 10` bps. Worst-quarter PF is `0.847931` at 5 bps and `0.813545` at 10 bps. The prior 13,178 “de-clustered” count is not accepted independence evidence because it is a sum of per-symbol-month one-hour spacing counts without cluster P&L or cross-symbol treatment.

## Immutable identity and retained schema

No dataset ID, dataset version, manifest ID/hash, evaluation cutoff, coverage/availability policy identity, expected partition set, or Historian PIT evidence exists for the replayed inputs. The diagnostic local-source manifest is not a substitute: it covers only 72 objects across three symbols, has no object hashes or availability timestamps, and is not an accepted PIT envelope.

The original evaluation explicitly retained no raw event details. Schema `11.2-retained` lacks event rows, decision-time context values, event-quality values, cluster membership/timestamps, per-event returns, and replay hashes. It therefore cannot reproduce baseline decisions, evaluate any allowed constraint, construct independent clusters, or replay accepted events.

## Partitions

| Partition | Boundary |
|---|---|
| DEVELOPMENT | `NOT_CREATED` |
| VALIDATION | `NOT_CREATED` |
| FINAL_HOLDOUT | `NOT_CREATED` |

No dates were invented because complete PIT-valid archive availability was not established.

## Gates and costs

The accepted gates remain unchanged: at least 300 independent decisions; at least 4 symbols and 12 months; net PF at 10 bps at least 1.10; net expectancy at least 0.01 bps; strictly positive expectancy lower bound; drawdown at most 2,500 bps; worst-period PF at least 0.95; symbol and temporal contribution at most 50%; regime contribution at most 60%; at least two stable parameter neighbors; OOS, walk-forward, cluster deduplication, missing-context sensitivity, reproducibility, PIT identity, and RIF holdout control required.

The cost stress is 10 bps: 5 fees + 1 spread + 1 slippage + 1 funding + 2 adverse selection. The reporting ladder is 5, 7.5, and 10 bps. The explicit PR4B0-R1 PF floor of 1.10 at 10 bps controls over the accepted report's inconsistent legacy 1.01 stress field.

The accepted report does not define the lower-confidence-bound estimator, confidence level, resampling unit, seed, or interval construction. No method was invented.

## Search and selection

- Maximum budget: 12 variants including the unchanged baseline.
- Declared variants: 0.
- Executed variants: 0.
- The Phase A baseline replay is an implementation-readiness check, not a candidate variant evaluation.
- Allowed families remain context agreement, event quality, and independence/cooldown.
- Calendar filters, Q1 exclusion, loss-derived symbol filters, hindsight thresholds, future fields, ML rankings, and report-derived exception lists remain prohibited.
- If execution had been legal, selection order would have been: all gates, validation lower bound, worst-period PF, concentration, then simplicity.

## Holdout

The accepted RIF one-time exposure capability was verified. No candidate was frozen or registered, no ledger was created, no exposure was authorized or recorded, and no holdout data were read.

## Blocking decision

The protocol cannot be made executable without inventing or substituting evidence. Blocking codes:

1. `IMMUTABLE_DATASET_IDENTITY_NOT_ESTABLISHED`
2. `COMPLETE_PIT_VALID_DATA_UNAVAILABLE`
3. `CANDIDATE_EVALUATION_BLOCKED_BY_RETAINED_SCHEMA`
4. `ACCEPTED_UNCERTAINTY_METHODOLOGY_UNSPECIFIED`

No gates were altered, no variants were created, and no results existed when this protocol was written.
