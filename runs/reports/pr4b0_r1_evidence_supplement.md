# PR4B0-R1 Evidence Supplement

## Closeout verdict and scope

Final label: `PR4B0_R1_RESEARCH_BLOCKED`.

This is a documentation-and-evidence correction. It does not rerun candidate research, evaluate a variant, inspect a holdout, change either protocol artifact, implement a retained schema, generate Historian data, select an uncertainty method, or begin another phase.

The precise baseline statement is:

> Legacy aggregate baseline metrics were reproduced exactly from the available accepted summary artifacts. Event-level and canonical-input reproduction could not be established.

Where older PR4B0-R1 artifacts say that the baseline or implementation reproduced byte-for-byte, this supplement narrows that statement to the artifact evidence below. The protocol remains an immutable historical artifact; the final-decision reports refer to this correction.

## Commit inspection

| Commit | Parent | Commit time | Exact files and changes |
|---|---|---|---|
| `ce1377682975c8bc3d5b947d35900500b03403bf` | `205cf59555006ce23fc58bc2c73262660a894850` | `2026-07-13T16:18:31-07:00` | Added `runs/reports/pr4b0_r1_research_protocol.json` (255 lines) and `.md` (83 lines), 338 insertions total. JSON records accepted authorities, the baseline implementation/report references, missing immutable identity and retained-event readiness, unchanged gates/costs, zero declared/executed variants, non-created partitions, holdout controls, and four blockers. Markdown renders the same blocked non-executable protocol. No source, test, build, trader, RIF, or Historian file changed. |
| `8f4df1e61455541262cc1c95e6a32e6b8948f980` | `ce1377682975c8bc3d5b947d35900500b03403bf` | `2026-07-13T16:25:51-07:00` | Added `pr4b0_r1_final_decision.json` (273 lines), `pr4b0_r1_final_decision.md` (134), `pr4b0_r1_variant_results.json` (203), and `pr4b0_r1_variant_results.md` (77), 687 insertions total. They record the blocked label, legacy aggregate values, zero variant/partition/holdout execution, blockers, gate summaries, integrity statements, and then-pending result-source/fresh-clone verification identities. No source or protocol file changed. |
| `945640fcd16537b8e1a82c49c4de28b5899982b9` | `8f4df1e61455541262cc1c95e6a32e6b8948f980` | `2026-07-13T16:30:14-07:00` | Modified exactly the four result/decision files (54 insertions, 11 deletions). It set `result_source_commit` to `8f4df1e...`; changed fresh-clone state from pending to pass; added the exact tidy, diff, vet, test, race, build, `make verify`, JSON/integrity, and clean-tree results for that clone; and updated the Markdown verification prose. No protocol or source file changed. |

At closeout, current protocol hashes still equal their values at `ce137768...`:

| Protocol artifact | Current SHA-256 | SHA-256 at `ce137768...` | Result |
|---|---|---|---|
| `runs/reports/pr4b0_r1_research_protocol.json` | `7d1305bb418d2463c966f1383be4e9b7e30b0aa08dd254cd60eb55e7825cb072` | same | unchanged |
| `runs/reports/pr4b0_r1_research_protocol.md` | `1d1a510de4a047faa4f4a6b2e73ef6225105d0fa8bb9cea73aad46dff09d6a37` | same | unchanged |

## What the baseline evidence establishes

| Reproduction question | Finding | Exact evidence |
|---|---|---|
| Rendered JSON report artifact | Established for the generated and committed legacy JSON | Original SHA-256 and reproduced SHA-256 are both `fb27fe46ab1139ccafea3a7b3cbb7bfdfc7fb3bd2f7e545f1b7b566d2e6c9066`; `cmp -s` returned 0. |
| Rendered Markdown report artifact | Matching hashes established; no separate `cmp` result was recorded | Original and reproduced SHA-256 are both `fba5947aad5dad971b24e634cb553dd8c4c3694f907bd3bdf0ef40060e95a9ed`. |
| Aggregate metrics from legacy summaries | Established exactly | Reaggregation of the 384 `retained_summaries` rows yields all counts and displayed metrics below. |
| Event-level decisions | Not established | The report says `raw_event_detail_retained=false`; no event IDs, timestamps, feature values, reference prices, horizon returns, or cluster assignments remain in the accepted report. |
| Actual implementation identity | Source identity established | Commit `c2c7988712699b26ba7ab28e1cebb1f5312812a6`, path `internal/app/phase12_downtrend_midvol_relief.go`, SHA-256 `3c2e20fd5bf615864aebc5be35ce86c15a6ed8f83de33b2f1d33b00dae6fbfa1`. |
| Actual implementation against immutable canonical input files | Not established | The source was executed against locally available files, but no complete manifest, per-input hashes, dataset identity, source `available_at`, or PIT envelope binds those files as canonical. |

### Aggregate derivation

The report lists 8 target symbols and 24 months, so expected coverage is `8 x 24 = 192` symbol-months. `retained_summaries` contains 384 rows because each of the 192 distinct `symbol|month` pairs has one long row and one zero-event short row. The command increments `completed_symbol_months` by 24 only after the input builder succeeds for a symbol; the accepted report records `192` completed and no missing pair.

Summing long-row `stats.raw_event_count` gives `329,842`. Summing long-row `stats.de_clustered_event_count` gives `13,178`. That second number is only the sum of one-hour timestamp-spacing counts computed independently inside each symbol-month; the report retains neither cluster membership nor cluster P&L and performs no cross-symbol common-market clustering.

For each cost, `event_count` is the sum of monthly event counts, `net_bps` is the sum of monthly net bps, profit factor is `sum(gross_profit_bps) / sum(gross_loss_bps)`, and expectancy is `sum(net_bps) / 329842`:

| Cost | Events | Gross profit bps | Gross loss bps | Net bps | PF | Expectancy bps |
|---:|---:|---:|---:|---:|---:|---:|
| 5 | 329,842 | 37,591,413.439301975 | 32,157,896.931923002 | 5,433,516.507368003 | 1.168963677 | 16.473088653 |
| 7.5 | 329,842 | 37,138,910.542528 | 32,529,999.035146035 | 4,608,911.507368002 | 1.141681883 | 13.973088653 |
| 10 | 329,842 | 36,690,202.051059 | 32,905,895.54368702 | 3,784,306.5073679998 | 1.115003906 | 11.473088653 |
| 15 | 329,842 | 35,804,423.884224 | 33,669,327.376830995 | 2,135,096.5073679993 | 1.063413696 | 6.473088653 |

Monthly and quarterly PFs use the same gross-profit/gross-loss aggregation grouped by `month` or `quarter`. The minima are March 2025 (`0.661711080` at 5 bps; `0.634441669` at 10 bps) and 2025-Q1 (`0.847931049` at 5 bps; `0.813544771` at 10 bps).

These are unpartitioned legacy aggregate-summary values. None is a legal DEVELOPMENT, VALIDATION, or FINAL_HOLDOUT qualification result.

## Execution ordering

The successful baseline command was invoked at `2026-07-13T22:51:50.238Z` (`2026-07-13T15:51:50.238-07:00`) and completed at `2026-07-13T23:15:24.865Z` (`2026-07-13T16:15:24.865-07:00`) with exit 0:

```text
GOWORK=off GOFLAGS=-buildvcs=false go build -o /tmp/pr4b0_r1_baseline_engine ./cmd/ak-engine && GOMEMLIMIT=1400MiB GOGC=25 /tmp/pr4b0_r1_baseline_engine phase12-downtrend-midvol-relief --workdir /home/davidmiguel22573/Github/ak-historian/.ak-historian/work --market futures-um --interval 1m --symbols LINKUSDT,SOLUSDT,AVAXUSDT,DOGEUSDT,ADAUSDT,BNBUSDT,XRPUSDT,ETHUSDT --from 2024-01 --to 2025-12 --out /tmp/pr4b0_r1_baseline_reproduction.md
```

The command ran from a detached worktree at `c2c7988...`. Protocol commit `ce137768...` was created later, at `2026-07-13T16:18:31-07:00`. Therefore:

- protocol commit observed by the command: none; it did not yet exist;
- protocol SHA-256 observed by the command: none; the protocol artifact did not yet exist;
- command relative to `ce137768...`: before;
- strategy metrics calculated before protocol freeze: yes, the legacy unpartitioned aggregate calculations above;
- modified candidate metrics calculated before protocol freeze: no; zero modified variants existed;
- proof of execution ordering: session-log invocation/completion timestamps plus the commit timestamp, not Git artifact ordering.

A1 treated this execution as an implementation/readiness check, not candidate variant V00. The reported `0/0` search count means zero declared modified variants and zero executed modified variants. The protocol allowed a maximum of 12 total variants, but stopped before declaring any. Retroactively calling the earlier readiness calculation V00 would invent a pre-registration that did not occur. The readiness calculation cannot contribute legal qualification metrics because it preceded freeze and lacks canonical/PIT identity.

## Immutable-identity recovery inventory

### Paths and artifacts searched

| Authority/location | Paths searched | Manifests found | Snapshots/files found | Hashes and coverage found | Identity conclusion |
|---|---|---|---|---|---|
| Historian commit `3eeff1eb45da281e0003dc1577ec55aa6cda1b1b` | `docs/pit_archive_evidence.md`; `internal/pitarchive/{types,manifest,snapshot,coverage,evaluator,evidence,canonical,atomic}.go` and tests; `runs/reports/pr3_historian_pit_archive_enforcement.{md,json}`; `testdata/pitarchive/v1/evidence.example.json` | No production snapshot manifest. One compatibility evidence fixture only. | No tracked production snapshots. The fixture describes three BTC 1h test snapshots for a three-hour 2025-01-01 window; it is not candidate data. | Fixture contains test-only hashes/coverage. The authority defines, but does not instantiate, production identity for this candidate. | No production PR4B0-R1 manifest/evidence exists at the accepted Historian commit. |
| Accepted Engine commit `205cf595...` | `runs/reports/pr4b0_candidate_{inventory,qualification}.{md,json}`; `internal/app/pr4b0_candidate_qualification{,_test}.go`; `internal/qualification/qualification{,_test}.go` | No candidate input manifest. | No snapshots. | Artifact hashes listed below; report declares identity requirements but no candidate values. | Requirements exist; candidate identity does not. |
| Phase 12 source commit `c2c7988...` | `internal/app/phase12_downtrend_midvol_relief{,_test}.go`; `runs/reports/phase12_3_downtrend_midvol_relief_eval.{md,json}` | No input manifest. | Report contains aggregate summaries, not snapshots/events. | Source/report hashes listed below; coverage is 8 x 24 legacy symbol-months. | Implementation/report identity exists; input identity does not. |
| Local Historian work directory used by A1 | `.ak-historian/work/manifests/**`; `.ak-historian/work/candles/futures-um/1m/**` | One candidate-candle-adjacent file: `manifests/local_source_manifest.json`. Funding-rate and sentiment manifests were also present but are different datasets and not candidate candle identity. | Local parquet filenames exist for target/BTC/ETH coverage, but none is bound as an `ak-historian.snapshot.v1` snapshot. | Local manifest SHA-256 `80ed307f...`; 72 entries for AVAX/LINK/SOL only; zero per-object checksums, known row counts, or coverage declarations. | Local availability is useful for diagnostics only; it is not immutable canonical/PIT evidence. |

Relevant artifact SHA-256 values:

| Artifact | SHA-256 |
|---|---|
| `runs/reports/pr4b0_candidate_inventory.json` | `3c94708c81788cb7b21219c08c9482ac621d90ae9b58970365cace29fbaf2d6a` |
| `runs/reports/pr4b0_candidate_inventory.md` | `71cb635b74f999cc9c0c2c93c2f68b47e233d5c61ea46821b376960090ab12e1` |
| `runs/reports/pr4b0_candidate_qualification.json` | `1caac0af112610905229f2c620919508ac9db3a4fe98d1a9640696c066c52765` |
| `runs/reports/pr4b0_candidate_qualification.md` | `4eb185283322fc3b3aac0cc29d31ba9b03dac98c1532e6f52e6d519b2123efc0` |
| `internal/app/pr4b0_candidate_qualification.go` | `e04b57f560aa4ac1c693dccbc95b30a1e9df06a2ae97d53cafcc5abea6d12f23` |
| `internal/app/pr4b0_candidate_qualification_test.go` | `4afcad1cceb51e2e0a285607b8d4edae97a6b422368617df33e28652c2a0bd6d` |
| `internal/qualification/qualification.go` | `efff909fff83d9821900cc3ac227450d6525b4d25961882af15960e94dabec29` |
| `internal/qualification/qualification_test.go` | `92f51c1960e33717b5c1f694949693a640c60734a5c676de2111c4492e76bb88` |
| `internal/app/phase12_downtrend_midvol_relief.go` | `3c2e20fd5bf615864aebc5be35ce86c15a6ed8f83de33b2f1d33b00dae6fbfa1` |
| `internal/app/phase12_downtrend_midvol_relief_test.go` | `6406f9a706ede8e06e83d85aa5c617cbe1fc7c66e2082b214819dd3faf3350a6` |
| `runs/reports/phase12_3_downtrend_midvol_relief_eval.json` | `fb27fe46ab1139ccafea3a7b3cbb7bfdfc7fb3bd2f7e545f1b7b566d2e6c9066` |
| `runs/reports/phase12_3_downtrend_midvol_relief_eval.md` | `fba5947aad5dad971b24e634cb553dd8c4c3694f907bd3bdf0ef40060e95a9ed` |
| local `manifests/local_source_manifest.json` | `80ed307f181a16026b0b20970dfe4cfaeb8b1acdf1c9f0298181ecb2bb9ff329` |

The local manifest has only an `objects` map. Its 72 entries cover AVAXUSDT, LINKUSDT, and SOLUSDT for 2024-01 through 2025-12. `last_verified_at` ranges from `2026-06-07T20:07:30.188649389-07:00` to `2026-06-07T20:13:18.462730363-07:00`; this is a local archive-verification timestamp, not source `available_at`. All 72 `checksum_if_available` fields are empty, `row_count_if_known` is zero, and `coverage_status` is empty.

Path-only inspection of `.ak-historian/work/candles/futures-um/1m/symbol=<SYMBOL>/**` found the following local files. These counts include files outside the requested research window and do not constitute a coverage declaration or Historian snapshot set:

| Symbol | Files found | Lexicographically first path month | Lexicographically last path/date |
|---|---:|---|---|
| LINKUSDT | 69 | 2023-01 | 2026-05-29 |
| SOLUSDT | 36 | 2023-01 | 2025-12 |
| AVAXUSDT | 36 | 2023-01 | 2025-12 |
| DOGEUSDT | 36 | 2023-01 | 2025-12 |
| ADAUSDT | 36 | 2023-01 | 2025-12 |
| BNBUSDT | 36 | 2023-01 | 2025-12 |
| XRPUSDT | 36 | 2023-01 | 2025-12 |
| ETHUSDT | 37 | 2022-12 | 2025-12 |
| BTCUSDT | 37 | 2022-12 | 2025-12 |

The same manifest directory also contains nine funding-rate manifests under `manifests/datasets/derivatives/binance/funding_rate/**` and one sentiment manifest. They describe different datasets; the implemented signal declares `funding_primary_trigger=false`, so they cannot identify the candidate candle inputs.

### Required identity-field recovery

| Required field | Recovered value | Exact evidence of absence or limitation |
|---|---|---|
| dataset ID | not recovered | The local manifest has only top-level key `objects`; accepted Engine reports set no candidate dataset ID; no production Historian manifest exists. |
| dataset version | not recovered | Same inventory; no `dataset_version` field binds the local candle set. |
| manifest ID | not recovered | Local manifest has no `manifest_id`; Historian tree has no production manifest. |
| manifest hash | not recovered | File SHA-256 `80ed...` is available, but it is not a declared canonical `manifest_hash` and its objects have no content hashes. |
| evaluation cutoff | not recovered | No candidate artifact contains an `evaluation_cutoff`; local `last_verified_at` cannot substitute. |
| coverage-policy version | not recovered | Accepted Historian code defines `ak-historian.coverage-policy.v1`, but no candidate manifest instantiates it or declares expected partitions. |
| availability-policy version | not recovered | Accepted Historian code defines `ak-historian.availability-policy.v1`, but no candidate manifest instantiates it. |
| source availability timestamps | not recovered | No candidate snapshot has `available_at`; local `last_verified_at` describes local verification, not source publication availability. |
| expected coverage declaration | not recovered | Filename inspection shows apparent files; the local manifest's `coverage_status` is empty and it covers only three of the eight target symbols. Files on disk do not define expected coverage under the accepted Historian contract. |
| per-input hashes | not recovered | All 72 local `checksum_if_available` values are empty; the remaining target/BTC/ETH files have no candidate manifest entries. |
| production snapshot identity | not recovered | No `ak-historian.snapshot-manifest.v1` or candidate `ak-historian.pit-evidence.v1` artifact was found. |

## Retained-schema sufficiency

`features.Row` held decision-time runtime fields, and `phase12DTMVREvent` temporarily held event results. The accepted report retained neither structure; it retained monthly `phase12DTMVRSummaryRow` aggregates under schema `11.2-retained`.

| Required capability/field | Exists in retained report? | Exact runtime or retained source | Decision-time available? | Deterministic replay? | Canonical clustering? | Consequence |
|---|---|---|---|---|---|---|
| Stable event identity | No | Ephemeral duplicate key `symbol|side|EventTimeMS`; no retained field | Runtime only | No | No | Cannot prove the same event set or deduplicate across runs. |
| Input hashes | No | No source/report field | No | No | No | Cannot bind replay to immutable inputs. |
| Event timestamp | No | Runtime `features.Row.EventTimeMS` / `phase12DTMVREvent.EventTimeMS` | Yes at runtime | No after retention | No | Cannot replay decisions or reconstruct clusters. |
| Cluster timestamps/membership | No | Runtime local `eventTimes`; retained only `de_clustered_event_count` and string `cluster_key_version` | Runtime only | No | No | Cannot audit the one-hour spacing count or form cross-symbol clusters. |
| Reference/entry price | No | Runtime `features.Row.Close` / `phase12DTMVREvent.EntryPrice` | Yes at runtime | No | No | Cannot recompute entry or verify price selection. |
| 240-minute horizon return | No | Runtime `ReturnsBps["240m"]`; retained aggregate gross/net/PF only | Outcome, not decision-time feature | No | No | Cannot recompute event P&L or uncertainty. |
| Primary-symbol regime | No value | Derived at runtime from close/EMA/trend; retained diagnostic counts only | Yes at runtime | No | No | Cannot refilter or verify event-level regime assignment. |
| Volatility bucket | No value | Runtime `RealizedVol60` with `[0.0015,0.006]`; aggregate pass counts only | Yes at runtime | No | No | Cannot verify MID classification or test an allowed constraint. |
| Relief magnitude | No; not implemented as a numeric gate | Named pullback gate is hard-coded true in the source | Not applicable | No | No | The family name cannot be translated into a replayable relief field. |
| BTC context | No value | Loader requires BTC candles and builds `BTCReturn60`; signal does not consume it | Built at runtime | No | No | Input dependency cannot be proven or used for context variants. |
| ETH context | No value | Loader requires ETH candles when target differs and builds `ETHReturn60`; signal does not consume it | Built at runtime | No | No | Input dependency cannot be proven or used for context variants. |
| Trend state | No value | Runtime `Close < EMA50 < EMA200` and `TrendSlope20 < 0`; aggregate counts only | Yes at runtime | No | No | Cannot verify the signal predicate per event. |
| Funding/basis context | Not required and not retained | Report says `funding_primary_trigger=false`; source uses no funding/basis field | Not applicable to implemented signal | No | No | Not a blocker for replaying this exact signal, but cannot support a new funding/basis constraint. |
| Warm-up evidence | No event-level evidence | Runtime `features.Row.Warmup`; loader starts two months early; report has feature-row counts only | Yes at runtime | No | No | Cannot prove each accepted event had complete warm-up or bind warm-up bytes. |
| Feature availability time | No | Runtime `AvailableAtMS`; retained only aggregate `leakage_status` | Yes at runtime | No | No | Cannot independently audit point-in-time availability. |
| Primary symbol/side/month | Yes, aggregate scope only | `retained_summaries[].symbol/side/month/quarter` | Yes | No event replay | No | Supports monthly aggregation only. |
| Aggregate cost totals/counts | Yes | `retained_summaries[].stats.cost_stress[]` | Outcomes | Aggregate reaggregation only | No | Supports the legacy aggregate calculations above, not legal event-level qualification. |

## Uncertainty-methodology recovery

The search covered every accepted PR4B0-specific report and implementation, plus the generic accepted qualification contract on which the gate depends:

- `runs/reports/pr4b0_candidate_inventory.json` and `.md` at `205cf595...`;
- `runs/reports/pr4b0_candidate_qualification.json` and `.md` at `205cf595...`;
- `internal/app/pr4b0_candidate_qualification.go` and `_test.go` at `25efa97...`/`205cf595...`;
- `internal/qualification/qualification.go` and `_test.go` at `25efa97...`/`205cf595...`;
- `internal/app/phase12_downtrend_midvol_relief.go` and `_test.go` at `c2c7988...`;
- `runs/reports/phase12_3_downtrend_midvol_relief_eval.json` and `.md` at `c2c7988...`.

Unrelated percentile helpers in `evaluate_candidate_deep.go` calculate descriptive percentiles; they do not define the PR4B0 expectancy lower-confidence-bound gate.

| Method element | Recovered? | Evidence |
|---|---|---|
| Estimator | No | Gate stores only `minimum_confidence_lower_bound_bps: 0`. |
| Confidence level | No | No accepted PR4B0 artifact states one. |
| Independent resampling unit | No | No event/cluster uncertainty contract; accepted cluster evidence itself is missing. |
| Bootstrap or analytical method | No | No accepted PR4B0 artifact chooses either. |
| Block construction | No | No block length, grouping, or dependence rule exists. |
| Seed | No | No seed exists. |
| Number of resamples | No | No resample count exists. |
| Interval rule | No | No percentile, basic, BCa, studentized, analytical, or other rule exists. |

The blocker remains `ACCEPTED_UNCERTAINTY_METHODOLOGY_UNSPECIFIED`. Choosing any of these items during PR4B0-R1 would add a substantive rule to an already accepted qualification gate and could change pass/fail behavior. That would be methodology design and gate alteration, both outside this closeout.

## Complete mandatory qualification-gate table

Statuses apply to legal PR4B0-R1 qualification, not to the legacy unpartitioned aggregates.

| Mandatory gate | Recovered threshold/method | Applicable scope | Status | Exact evidence | Can contribute to qualification? |
|---|---|---|---|---|---|
| Dataset ID | Exact immutable identity | All partitions | FAIL | No candidate dataset ID in the inventory. | No |
| Dataset version | Immutable version; mutable aliases prohibited | All partitions | FAIL | No candidate dataset version. | No |
| Manifest ID/hash | Exact ID and canonical SHA-256 matching PIT evidence | All partitions | FAIL | Local manifest has no ID/canonical hash and zero object checksums. | No |
| Explicit non-overlapping partitions | DEVELOPMENT, VALIDATION, FINAL_HOLDOUT inside research window | All qualification | NOT RUN | No legal partition boundaries were created. | No |
| Expected partitions/gap policy | Exact expected set; reject undeclared gaps | All inputs | FAIL | No candidate coverage policy or expected set. | No |
| Minimum events | >=300 | Each legal evaluated scope | NOT RUN | 329,842 is an unpartitioned legacy aggregate, not a legal scope result. | No |
| Minimum independent clusters | >=300 | Each legal evaluated scope | BLOCKED | 13,178 spacing sum has no membership/P&L and no cross-symbol clustering. | No |
| Minimum trades/decisions | >=300 | Each legal evaluated scope | BLOCKED | Event decisions were not retained. | No |
| Minimum symbols | >=4 | Qualification result | NOT RUN | Legacy report lists 8, but no legal partition evaluation ran. | No |
| Minimum months | >=12 | Qualification result | NOT RUN | Legacy report lists 24, but no legal partition evaluation ran. | No |
| Positive-regime coverage | >=1 positive regime | Qualification result | NOT RUN | No event-level regime outcomes or legal result. | No |
| Negative-regime coverage | >=1 negative regime | Qualification result | NOT RUN | No event-level regime outcomes or legal result. | No |
| Net expectancy | >=0.01 bps | DEVELOPMENT/VALIDATION and frozen result | NOT RUN | Legacy 10 bps aggregate is informational only. | No |
| Profit factor | >=1.10; explicit R1 10 bps floor controls over inconsistent legacy 1.01 field | DEVELOPMENT/VALIDATION and frozen result | NOT RUN | Legacy 10 bps PF 1.115004 is not a legal partition result. | No |
| Maximum drawdown | <=2,500 bps | Each legal evaluated scope | NOT RUN | Event/equity path unavailable. | No |
| Expectancy lower confidence bound | Strictly >0 | VALIDATION selection and qualification | BLOCKED | Estimator and interval methodology unspecified. | No |
| Downside tail | Worst decile/worst-symbol-month loss within declared 2,500 bps drawdown budget | Legal evaluated scopes | NOT RUN | Event returns and legal partitions unavailable. | No |
| Out-of-sample validation | Required | VALIDATION | NOT RUN | Validation partition absent; zero reads. | No |
| Walk-forward stability | Required | Pre-holdout development/validation | NOT RUN | Zero slices defined or run. | No |
| Worst-period PF | >=0.95 | Legal evaluation periods | NOT RUN | Legacy Q1 2025 PF is below the threshold, but is not a legal qualification result. | No |
| Symbol concentration | <=50% | Legal evaluated scope | NOT RUN | No event/cluster P&L in a legal scope. | No |
| Temporal concentration | <=50% | Legal evaluated scope | NOT RUN | Legacy month contribution is not a legal partition result. | No |
| Regime concentration | <=60% | Legal evaluated scope | NOT RUN | No retained event-level regime P&L. | No |
| Stable parameter neighbors | >=2 | DEVELOPMENT then VALIDATION | NOT RUN | Zero modified variants and neighbors. | No |
| Cluster deduplication | Required | All performance/sample results | BLOCKED | Only within-symbol-month one-hour count; no canonical clusters. | No |
| Missing-context sensitivity | Required | DEVELOPMENT/VALIDATION | BLOCKED | BTC/ETH context values and event rows absent. | No |
| Fee cost | 5 bps | All net results | NOT RUN | Cost assumption recovered; no legal result. | No |
| Spread cost | 1 bps | All net results | NOT RUN | Cost assumption recovered; no legal result. | No |
| Slippage cost | 1 bps | All net results | NOT RUN | Cost assumption recovered; no legal result. | No |
| Funding cost | 1 bps | All net results | NOT RUN | Cost assumption recovered; no legal result. | No |
| Adverse-selection cost | 2 bps | All net results | NOT RUN | Cost assumption recovered; no legal result. | No |
| Total cost stress | 10 bps and PF >=1.10 / expectancy >=0.01 bps | All legal results | NOT RUN | Legacy stress values cannot substitute for partitioned qualification. | No |
| Reject future candles | Required | Feature construction | BLOCKED | No retained event timing/input PIT evidence. | No |
| Reject revised data unavailable at cutoff | Required | Feature construction | BLOCKED | Source `available_at` and evaluation cutoff absent. | No |
| Reject final outcomes in features | Required | Feature construction | NOT RUN | Source review is consistent, but no canonical event replay was possible. | No |
| Reject holdout-derived feature selection | Required | Search/selection | PASS | Zero holdout reads and zero modified variants. | Yes, as an integrity control only |
| Require PIT-compatible source timing | Required | All inputs/features | FAIL | No candidate PIT evidence or availability timestamps. | No |
| Reject manifest/dataset mismatch | Exact match required | All inputs | FAIL | No candidate identities exist to match. | No |
| Simplicity | Simplest candidate only after every mandatory gate passes | Selection | NOT RUN | No declared variants or survivors. | No |
| Finite search budget | <=12 total variants; no expansion | Development search | PASS | Zero declared and zero executed modified variants; no expansion. | Yes, as an integrity control only |
| Implementation complete/identified | Exact source and required behavior | Pre-evaluation | PASS | Source commit/path/hash recovered and tests exist. Input replay remains unproven. | Yes, as a prerequisite only |
| Historian PIT evidence | Accepted exact evidence required | All qualification | FAIL | No production candidate manifest/evidence envelope. | No |
| RIF-controlled holdout authorization | One-time authorization after frozen survivor | FINAL_HOLDOUT | NOT RUN | No survivor/freeze/registration; zero reservations and consumptions. | No |

No combination of PASS integrity controls qualifies the candidate while mandatory FAIL, BLOCKED, or NOT RUN rows remain.

## Research-integrity boundaries

| Boundary | Count/status | Evidence |
|---|---|---|
| Declared modified variants | 0 | `declared_variants` empty; no variant descriptors. |
| Executed modified variants | 0 | Development/validation/holdout result arrays empty. |
| Development reads | 0 | DEVELOPMENT partition was never created. |
| Validation reads | 0 | VALIDATION partition was never created. |
| Holdout enumerations | 0 | FINAL_HOLDOUT partition was never created or enumerated. |
| Holdout event reads | 0 | `holdout_read=false`; no holdout result. |
| RIF exposure reservations | 0 | No ledger/registration/reservation artifact created. |
| RIF exposure consumptions | 0 | `exposure_count=0`; accepted RIF worktree remains clean. |
| Paper-authority artifacts | 0 | `candidate_registration_artifact=null`; no paper authority or qualified-only artifact produced. |
| Trader changes | 0 | Target diff contains report artifacts only; `ak-trader` remains clean at `1727af8327f2fb6a51fa907e8fab83944a661696`. |
| Paper-evaluator changes | 0 | No Go/source file changed in the PR4B0-R1 chain or this closeout. |

Archive coverage inspection enumerated manifest metadata and candle filenames to establish what files appeared present. It did not inspect candidate outcomes. The separate A1 readiness command did calculate the legacy unpartitioned candidate aggregates before protocol freeze; that fact is disclosed above and those values are excluded from legal qualification.

## Verification

The follow-up local suite passed: module tidy/no diff, vet, tests, race tests, build, `make verify`, JSON parsing, both protocol-hash assertions, secret/private-key scans, sibling-import scan, no-Go-change scan, qualified-artifact scan, and `git diff --check`.

The original absolute-host-path scan was also run. It found exactly two required evidence occurrences: the exact baseline command in this Markdown and its JSON counterpart. A narrow allowlist assertion permits those two fields only; no other absolute host path is accepted. This is recorded as `PASS_WITH_MANDATED_EXACT_COMMAND_EXCEPTION`, not as an unqualified scan pass.

The same suite, without the linked-worktree VCS workaround, passed in an isolated no-sibling fresh clone of the follow-up commit. Resolve that commit with `git log -1 -- runs/reports/pr4b0_r1_evidence_supplement.json`.

Final recommendation remains: do not begin PR4B1 or PR4B0-R2. This closeout does not authorize retained-schema implementation, Historian generation, or uncertainty-method design.
