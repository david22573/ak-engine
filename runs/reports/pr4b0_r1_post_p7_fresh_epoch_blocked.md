# PR4B0-R1 Post-P7 Fresh Epoch Closeout

## Executive verdict

The epoch is blocked before protocol registration. Accepted P7 source verifies and the immutable Historian checkpoint is present and exact, but no accepted production path constructs the Engine's required Historian-backed `PartitionArtifact`, and no accepted production entry point orchestrates P7 RIF authorization/access/result/seal calls with Engine execution. All such construction and consumption is synthetic test code.

The runner requires precomputed decision features, BTC/ETH context, and `future_close_240m`. Creating that data-preparation/orchestration path now would be a new infrastructure or runner-authority phase and an ad hoc dataset path, all explicitly prohibited by this task. Independently, frozen V00 configures eight symbols and omits BTCUSDT while this epoch mandates all nine and bans narrower symbol whitelists.

```text
PR4B0_R1_RESEARCH_BLOCKED
```

## Safety boundary

- Protocol identity/commit: none.
- Registered variants: 0; IDs/hashes: none.
- RIF research identity and holdout reservation: not created.
- DEVELOPMENT: not authorized, 0 executions, empty gate matrix, nominee none.
- VALIDATION: not opened, 0 executions.
- Candidate frozen: no.
- FINAL_HOLDOUT: 0 accesses, 0 executions, no gate result.
- Concentration, Independence V3, Uncertainty V2: exact source identities verified; not executed.
- Candidate partition rows, events, outcomes, result artifacts, paper evaluations, trader changes, and order actions: 0.

## Accepted identities

- Engine start `76b31415791cc98308bc3f0d4d23bec326cedd0b`; P7 semantic source `83ecf47d765fc2941a8b36c0a465d10612653ad7`.
- RIF start `ed2cbc903e5b34052f4f9d63ad99d02101effb37`; P7 semantic source `6e8dca3aad579ae8e8c0b54a62a15156ec52ece0`.
- Historian `14e0b569cc41e3517d2393d87d39c00559ac408e`; tree `74182d50c8f6dc6639c68cd6af9a82656640701c`.
- Both P7 finals are direct reports-only descendants of their semantic source commits.
- P7 runner binary `sha256:371153d611cebce5838caebf1a57346f15f566c7494042538758b133cd98faa7`; stage-set `ak.rif.stage_execution_set.v1`; store/envelope V2.
- Checkpoint `r1p5r-checkpoint-20260717T073628Z`, `sha256:bef53d11aa9ce9a6dad61b89ef7ace063b6da812ff92208d463c6ecfbfe8f29c`.
- Availability: latest durable evidence `2026-07-17T07:36:28.276810652Z`; strict cutoff `2026-07-17T07:36:29Z`.
- Gates `ak.engine.qualification-gates.pr4b0.v1`, `sha256:647e5d0885dbe8b73df736762e2f998bfa88f54d9331ed6aa4604888ab982635`.
- Independence V3 `sha256:84a6863b354b453dbe13698b9854ec4adcd116466a0831e7107efb892042cc1f`; Uncertainty V2 `sha256:1a91541c94378cc6f34e62a39ae504d3d013b5dab63a2b622641cdd1088148fb`; concentration `sha256:a126849e4cc0bd6457cf3f11079c4e3e2865ffce6c53a95ce92fa250130d39d5`.
- V00 config `sha256:9a3b4d2797daedac643491b8b420b033d05ca46bf051f9fa42b656eb29ede4de`; source `sha256:3c2e20fd5bf615864aebc5be35ce86c15a6ed8f83de33b2f1d33b00dae6fbfa1`.

## Partitions

- DEVELOPMENT `[2026-01-01T00:00:00Z, 2026-04-10T00:00:00Z)` — 99 days.
- VALIDATION `[2026-04-10T00:00:00Z, 2026-05-29T00:00:00Z)` — 49 days.
- FINAL_HOLDOUT `[2026-05-29T00:00:00Z, 2026-07-17T00:00:00Z)` — 49 days.

All nine-symbol structural coverage, 1,773 checkpoint manifest bindings, source-chain hashes, and zero gap/duplicate/conflict/schema/evidence/clock counts passed accepted-code verification. No prohibited prior evidence entered an epoch dependency; old identities occur only in tracked quarantine/rejection enforcement and prior reports.

## Verification

Engine and RIF full tests/vet/race checks pass. Historian R1P5R/prospective tests/vet and the exact accepted checkpoint overlay audit pass. Two independent deterministic builds reproduce the accepted Engine and RIF hashes byte-for-byte. Formatting/diff, no-trader, secret, symlink, cache, checkpoint-mutation, prohibited-data, ancestry, and clean-worktree audits pass. `govulncheck`, `gosec`, and `staticcheck` are unavailable.

The canonical JSON contains the exact source blobs, commands, artifact disposition, blocker evidence, and zero-access lifecycle matrix.

Artifact hash: `sha256:b8b1305372bc6cb07b3d4d7781a7e443cd01f794620e230f82de7b409e43e65c`
