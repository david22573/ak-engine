# PR4B0-R1P3B Independence Contract

Identity: `ak.engine.independence.downtrend-midvol-relief.v3`  
Status: **ACCEPTED**  
Hash: `sha256:84a6863b354b453dbe13698b9854ec4adcd116466a0831e7107efb892042cc1f`

V3 preserves the V2 240-minute half-open UTC interval, same-symbol transitive overlap, cross-symbol common-market episode rules, deterministic deduplication/order, and canonical cluster identity, then binds the accepted structural concentration authority.

- Symbol: each cluster contributes exact mass 1 split `1/K` across sorted unique primary symbols; maximum share `<= 1/2`.
- Temporal: each cluster belongs to the UTC `YYYY-MM` containing its earliest normalized event timestamp; maximum cluster-count share `<= 1/2`.
- Largest cluster: maximum unique member-event count divided by total unique represented member events; `<= 1/2`.
- Top five: order by member count descending then canonical cluster ID ascending, sum up to five over total member events; `<= 7/10`.

Equality passes. All comparisons use exact rational arithmetic. Reporting cannot round a failure into a pass. Missing/duplicate/malformed identities and zero denominators fail closed. V1 and pending V2 remain rejected.
