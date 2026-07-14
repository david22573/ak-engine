# PR4B0-R1P3 Independence Decision

Decision: **REVISE**. The V1 proposal remains unaccepted.

Revised V2: `ak.engine.independence.downtrend-midvol-relief.v2`

Revised contract hash: `sha256:006f19c3f89650f6905931164d6c98ead20800a2346369dadda708cfadf36528`

V2 implements half-open 240-minute UTC exposure, transitive same-symbol overlap, exact episode-qualified cross-symbol overlap, deterministic canonical cluster IDs, deduplication, and one-cluster/one-decision semantics. It is **not accepted** because no accepted source report supplies exact largest-cluster or aggregate cluster-concentration thresholds and denominators. Symbol and temporal `<=50%` thresholds were recovered from `runs/reports/pr4b0_candidate_qualification.json` at commit `205cf59555006ce23fc58bc2c73262660a894850`, but their denominator definitions are incomplete. No threshold was invented.

Common-market episode identity is the canonical SHA-256 of exact BTCUSDT/ETHUSDT context symbols, snapshot IDs, source hashes, and UTC availability instants. This conservative rule can under-cluster when distinct provenance records describe one move; same-symbol bridges can join multiple episode identities; and it intentionally does not infer latent dependence from future or retrospective values.

Acceptance was decided without inspecting candidate-performance results.

Decision-record hash: `sha256:a8accf33d27193a2c3b29bedfd10da3afa1e34656385cfda7af53b3067b388bf`
Artifact hash: `sha256:121b09cdf9e8567147295c61955a0293821041cdc216ecce44d8fc56558bcf43`
