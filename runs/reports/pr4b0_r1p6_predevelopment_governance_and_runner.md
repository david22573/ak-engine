# PR4B0-R1P6 pre-development governance and runner

Verdict: `PR4B0_R1P6_PREDEVELOPMENT_GOVERNANCE_AND_RUNNER_COMPLETE`

> No real candidate configuration was executed.
>
> No DEVELOPMENT outcomes were produced.
>
> No VALIDATION outcomes were produced.
>
> No FINAL_HOLDOUT rows were accessed.
>
> No paper evaluation was started.

## Preflight and evidence

Engine started at `1d575f4b5331a640b1dc4fa46a2c281aa6a2487a`, RIF at `c5823e88d0f1fd8f25a93ee70c5a3490fc12dbcb`, and read-only Historian at `14e0b569cc41e3517d2393d87d39c00559ac408e`. Historian finished at the same commit and tree. Prior blocked evidence `5416be01d35f5e86cf3c14044a8d84674c4d6544` is exactly one report-only commit after the Engine baseline; its canonical hash `sha256:5b3e0c857f8286762e44ff503e390d5dc2dff96a141c52b626dfa210a0b3de36` was reproduced, and the commit was not cherry-picked.

## RIF V4 governance

The additive contract is `ak.rif.research_identity.v4`, persisted as `ak.rif.research_governance_store.v1` and exported as `ak.rif.research_governance_envelope.v1`. It binds repositories/runner, content-addressed protocol, candidate scope, complete dataset identity, Historian commit, intervals, symbols, three partitions, variant ledger/neighborhoods, authorities/gates/cost/seed, and one-shot access policy while leaving V1–V3 compatible.

The lifecycle is `RESEARCH_IDENTITY_REGISTERED → HOLDOUT_RESERVED → DEVELOPMENT_AUTHORIZED → DEVELOPMENT_SEALED → VALIDATION_AUTHORIZED → VALIDATION_SEALED → CANDIDATE_FROZEN → FINAL_HOLDOUT_AUTHORIZED → FINAL_HOLDOUT_SEALED → QUALIFIED|REJECTED`, with separate `BLOCKED` integrity disposition. Reservation precedes DEVELOPMENT, requires no outcomes/validated/frozen candidate, is idempotent only for an identical request, and leaves every partition closed. Atomic locked state, expected sequence/state hash, and hash-chained lifecycle/authorization/receipt records prevent forks and replay. FINAL_HOLDOUT requires the exact frozen candidate and consumes access once before execution.

## Engine runner and enforcement

The runner identity is `pr4b0-r1-qualification-runner`; schemas are `ak.engine.qualification_execution_request.v1`, `ak.engine.qualification_structural_readiness.v1`, and `ak.engine.qualification_execution_result.v1`. Source tree is `5205627d259d3b8039c191b81b7ea5a43f17b141`; two deterministic `-trimpath` builds matched at `sha256:eefec664f453db2a19bc643263fdce56abc4dde63b1226bd746e77bdbf3e48fd`. Dry verification produces `NO_CANDIDATE_OUTCOMES_PRODUCED` with zero row/event/outcome loads. There is no caller executor or dataset-path bypass.

V00 binds source `sha256:3c2e20fd5bf615864aebc5be35ce86c15a6ed8f83de33b2f1d33b00dae6fbfa1` and configuration `sha256:9a3b4d2797daedac643491b8b420b033d05ca46bf051f9fa42b656eb29ede4de`. The ledger allows at most 12 variants and only context agreement, event quality, and cooldown/independence. Symbols, dates, quarters, costs, side, horizon, sizing, outcome filters, authorities, indicators, features, and semantic changes are rejected.

The accepted gate contract is `ak.engine.qualification-gates.pr4b0.v1`, `sha256:647e5d0885dbe8b73df736762e2f998bfa88f54d9331ed6aa4604888ab982635`; thresholds and strict/inclusive comparisons are unchanged. Synthetic execution called and bound the actual accepted independence V3 (`sha256:84a6863b354b453dbe13698b9854ec4adcd116466a0831e7107efb892042cc1f`), uncertainty V2 (`sha256:1a91541c94378cc6f34e62a39ae504d3d013b5dab63a2b622641cdd1088148fb`), and concentration governance (`sha256:a126849e4cc0bd6457cf3f11079c4e3e2865ffce6c53a95ce92fa250130d39d5`) implementations.

Checkpoint, source/version/binary, exact interval, partition, symbols, cutoff, and barred exposure are enforced. Newer/regenerated/extended/shortened datasets, pre-2026 or cross-partition rows, symbol changes, unregistered cache, symlink, and caller-path substitution fail closed.

## Synthetic integration and proofs

Fixture-only tests traversed the complete RIF chain, kept partitions closed after reservation, executed synthetic registered configurations in all Engine modes, rejected a mismatched frozen candidate, consumed FINAL_HOLDOUT exactly once, sealed it, and verified every chain. Fixtures are labeled `SYNTHETIC_NON_RESEARCH_EVIDENCE` and do not use accepted real checkpoint rows.

Real partition accesses: **0**. Real candidate configurations/outcomes: **0**. Historian changes: **0**. Trader dependencies, modifications, and order-placement paths: **0**.

## Commands, commits, and files

Full `GOWORK=off go test ./... -count=1` and `go vet ./...` passed in both repositories. RIF normal build, verifier, and `go test -race ./research` passed. Engine runner/bridge/gate race tests passed, as did the full verifier with `GOFLAGS=-buildvcs=false`; normal Engine `go build ./...` alone hits Go VCS stamping `exit status 128`, while direct Git probes and the identical build with stamping disabled pass. Formatting, module-diff, dependency, safety, determinism, exact-diff, worktree, and Historian audits passed.

RIF commits: `de3883207c36baeca3a267170937892d355acf43`, `6dd7442d215d10d73fa3702a69c5811cb291e974`, reports `50e07b55b88d2c9b98b34ee8d62fe32b4a39eb0a`. Engine source commits: `9cf545ab75e09954b08c1e1aa85c6db36aa22a64`, `4852d5e8fe4be30f26dd0156015af2b11a8a50dd`. Exact per-commit files and command results are in the canonical JSON. The Engine report-only commit is recorded externally because the report cannot contain its own commit hash.

There are no infrastructure blockers. PR4B0-R1 may now be restarted as a separate fresh task, but it was not restarted here.
