# PR4B0-R1P6 qualification runner

`pr4b0-r1-qualification-runner` is the dedicated, registration-enforcing
execution path for a future PR4B0-R1 epoch. It is not a wrapper around the
legacy candidate inventory command.

The runner accepts one strict `ak.engine.qualification_execution_request.v1`
that duplicates every security-critical identity from a fully hash-chain-
verified RIF V4 governance envelope. Protocol, checkpoint, source, partition,
symbol, candidate, configuration, authorities, gate set, cost policy, seed
policy, and runner identities must all match exactly.

Modes are explicit:

- `verify` validates structure and emits
  `NO_CANDIDATE_OUTCOMES_PRODUCED`; partition artifacts are prohibited.
- `development` requires a consumed one-shot DEVELOPMENT authorization.
- `validation` requires a consumed one-shot VALIDATION authorization for a
  registered nominee or stability configuration.
- `final-holdout` requires a consumed one-shot FINAL_HOLDOUT authorization and
  the exact frozen variant, configuration, executable, protocol, checkpoint,
  authorities, and gate set. Neighboring variants are rejected.

V00 resolves internally to the accepted historical source identity
`sha256:3c2e20fd5bf615864aebc5be35ce86c15a6ed8f83de33b2f1d33b00dae6fbfa1`
and fixed long/240m/downtrend/mid-volatility behavior. All effective defaults
are canonicalized. Only `context-agreement`, `event-quality`, and
`cooldown/independence` may vary, and the ledger is capped at twelve entries.
There is no caller-provided executor interface.

The accepted PR4B0 gate declaration is centralized as
`ak.engine.qualification-gates.pr4b0.v1`. Existing report behavior delegates
to this declaration, with regression coverage for every threshold and strict
or inclusive comparison.

Execution invokes the accepted implementations directly:

- independence V3 clustering;
- structural concentration evaluation under its accepted governance hash;
- uncertainty V2 deterministic cluster bootstrap.

Result artifacts bind evidence hashes from the implementations actually run.
Authority names or caller booleans cannot substitute for execution.

Partition data is accepted only as canonical, hash-bound registered artifact
bytes after RIF has durably consumed the authorization. Paths are not part of
the execution request; the CLI rejects symlinked authority/artifact files.
Rows outside the exact partition, before 2026, outside the symbol universe, or
unavailable at decision time fail closed.

All PR4B0-R1P6 execution tests use `SYNTHETIC_NON_RESEARCH_EVIDENCE`. They do
not open the accepted Historian checkpoint or access any real research row.
