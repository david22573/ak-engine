# PR4B0-R1 fresh epoch restart — blocked

Canonical machine-readable authority: `pr4b0_r1_fresh_epoch_restart_blocked.json`.

Canonical self-hash: `sha256:59d9687c155e4ffb6c9b7290c7411edeb141624c431d659db8358b3f1072489f`.

Verdict: `PR4B0_R1_RESEARCH_BLOCKED`.

All three accepted starting commits, the P6 RIF/runner/gate identities, the accepted R1P5R checkpoint, and every required authority hash were independently verified. The accepted Engine binary also reproduced twice at the P6 deterministic hash with `-buildvcs=false`.

Execution stopped before a protocol commit, RIF registration, reservation, partition access, or candidate outcome because the accepted P6 contracts cannot represent the requested epoch without bypass:

1. RIF requires each immutable partition's coverage hash before registration, while Engine requires that value to equal the exact sealed feature/outcome-row artifact hash. The artifact cannot be derived without opening protected partition data, and the accepted checkpoint does not contain it.
2. RIF authorizes exactly one bound configuration for DEVELOPMENT and one for VALIDATION. Engine rejects other configurations against that authorization, so it cannot execute a complete DEVELOPMENT ledger or a nominee plus all mandatory VALIDATION neighbors.
3. The Engine runner counts registered neighbor IDs as `stable_neighbors` but consumes no neighbor results, so registration alone cannot prove that every mandatory neighbor passed.

No old R1P5 source, receipt chain, checkpoint, or derived evidence was used. No DEVELOPMENT, VALIDATION, or FINAL_HOLDOUT partition was opened. No RIF state, candidate event, or candidate outcome was produced. Historian and RIF remain unchanged, and no downstream work began.
