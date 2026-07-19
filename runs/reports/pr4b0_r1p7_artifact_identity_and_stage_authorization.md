# PR4B0-R1P7 artifact identity and stage authorization

Verdict: complete. The pre-execution identity cycle is removed, DEVELOPMENT and VALIDATION now support their required execution sets, and FINAL_HOLDOUT remains exact frozen-candidate-only.

Canonical JSON self-hash: `sha256:b2647185c3f9ef64cf75b7c787e249b9d7ad48557d9c233af1f7de789c52546c`.

> No real candidate configuration was executed.
>
> No real DEVELOPMENT outcomes were produced.
>
> No real VALIDATION outcomes were produced.
>
> No real FINAL_HOLDOUT rows were accessed.
>
> No paper evaluation was started.

## Repository preflight and blocked evidence

Engine started in an isolated clean worktree at `2c210eb739596024aee4d411c1c2c3a85dfb0702` and reached source commit `83ecf47d765fc2941a8b36c0a465d10612653ad7`. RIF started at `50e07b55b88d2c9b98b34ee8d62fe32b4a39eb0a` and reached source commit `6e8dca3aad579ae8e8c0b54a62a15156ec52ece0`. Dirty original checkouts were not modified. Historian remained clean and unchanged at commit `14e0b569cc41e3517d2393d87d39c00559ac408e`, tree `74182d50c8f6dc6639c68cd6af9a82656640701c`.

The latest blocked evidence is Engine commit `3ac81aa1a44bbcac14317193215c016731a6d1c3`, directly after the accepted Engine start. It changes only `pr4b0_r1_fresh_epoch_restart_blocked.{json,md}`. Its canonical JSON self-hash was independently reproduced as `sha256:59d9687c155e4ffb6c9b7290c7411edeb141624c431d659db8358b3f1072489f`. It was neither used as the semantic baseline nor cherry-picked.

## Cycle reproduction and identity repair

P6 coupled an opened, post-access partition artifact/result identity to a preregistered structural identity and exposed only one lifecycle authorization per stage. That requires information unavailable before access and cannot authorize the full DEVELOPMENT ledger or the VALIDATION neighborhood without bypass.

P7 separates runner implementation, registered configuration, and ordered stage-plan identities from the post-execution result identity. The RIF authorization binds source commit, canonical package, sorted build inputs, compiler/build mode, binary, full canonical configurations, protocol, dataset, partition, checkpoint, seed/cost policy, authorities, and gate set. It contains no future result hash. Deterministic prebuild receipts prove zero data loads, events, and outcomes.

After execution, RIF canonicalizes and re-hashes the actual result artifact. Missing, all-zero, wrong, or caller-substituted hashes fail. Runner/build, configuration, protocol, partition, checkpoint, authority, and gate substitutions fail. Sealing requires the exact access receipt, deterministic run ID, output manifest, and actual invocation evidence from the accepted authority implementations.

## Stage execution sets

RIF adds explicit V2 store/envelope support and `ak.rif.stage_execution_set.v1` without rewriting V4 or V1 records. DEVELOPMENT authorizes the complete 1–12 configuration ledger in numeric order with unique derived authorizations. Synthetic coverage uses V00–V03; missing, extra, duplicate, reordered, mutated, cross-set, and duplicate-success inputs fail, and incomplete sets cannot seal. The complete manifest seals before the lowest numeric passing nominee is chosen; synthetic scores prove no performance ranking is used.

VALIDATION is derived only from sealed DEVELOPMENT. The synthetic set is exactly V00, V01, and V02 for nominee V01 plus its two registered neighbors. Missing neighbors and unrelated variants fail. Better-performing neighbors cannot replace the nominee. A failed required member is a legitimate `REJECTED/PERFORMANCE` result and never opens holdout.

FINAL_HOLDOUT does not accept an execution set. The unchanged V1 path authorizes one exact frozen nominee/configuration/runner/checkpoint/partition/protocol once. Alternatives, neighbors, mutation, and replay fail.

## Engine, restart, and synthetic integration

The Engine application contract verifies complete RIF sets and per-variant authorizations before invoking the existing governed qualification implementation. It accepts no caller executor, result hash, gate outcome, or authority-evidence seam. Actual synthetic batch execution invokes the accepted independence V3, uncertainty V2, concentration, and gate implementations and emits canonical per-variant envelopes, receipts, and a manifest.

Progress is locked, atomically replaced, self-hashed, and receipt-chained. Completed variants resume at the next ordinal and never rerun. An indeterminate access attempt blocks; retry requires exact durable RIF proof of zero rows and zero outcome artifacts. Tamper, duplicate success, incomplete sealing, and post-seal execution fail.

The explicitly synthetic integration covers four DEVELOPMENT variants, deterministic nominee selection, two-neighbor VALIDATION, one frozen nominee, and one synthetic FINAL_HOLDOUT evaluation. All individual receipts, manifests, lifecycle hashes, and synthetic access counts verify. No accepted checkpoint identity or real row appears in changed Go source.

## Compatibility, verification, and safety

Existing V1 store/envelope records remain readable; V2 activation is explicit. Existing single-execution, lifecycle, canonicalization, one-shot, gate-comparison, and frozen-candidate tests pass. Accepted authority and gate implementations, thresholds, comparisons, IDs, and hashes are unchanged.

Both repositories pass full `GOWORK=off go test ./... -count=1`, `go vet`, their `GOFLAGS=-buildvcs=false ./scripts/verify.sh`, formatting, exact-diff, module, dependency, no-trader, safety, no-real-data, no-outcome, and clean-worktree audits. RIF race passes for `./research`; Engine race passes for `./internal/qualificationrunner ./internal/rifbridge`. Two deterministic builds are byte-identical at RIF `sha256:ac761e3d05abe8453427236c217202a4ff633acdf822d11ce30ed00c3c9f839d` and Engine `sha256:371153d611cebce5838caebf1a57346f15f566c7494042538758b133cd98faa7`. `govulncheck`, `gosec`, and `staticcheck` were unavailable; no failing test was concealed. The canonical JSON records the resolved VCS-stamping and wrong-worktree audit-command incidents.

Real partition access count: **0**. Real candidate outcome count: **0**. Paper evaluations: **0**. Remaining blockers: **none**. PR4B0-R1 may restart in a separate task; it was not restarted here.
