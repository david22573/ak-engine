# PR4B0 Candidate Qualification

## Executive verdict

Existing Engine evidence was completely inventoried. No candidate satisfies every mandatory qualification gate; no search, holdout exposure, freeze, registration request, promotion, or paper evaluator was performed.

Final label: `PR4B0_NO_CANDIDATE_QUALIFIED`.

## Baselines and result

- Engine: `a04f6d6c8631e06049c7e108581dd638e9962a7b`
- RIF contracts/fixtures: `29350344a57e46f064442eada26e9418515990be`
- Historian PIT authority: `3eeff1eb45da281e0003dc1577ec55aa6cda1b1b`
- Resulting source commit: `25efa97ca89f8dcb724f9872e798bc789123caac`
- Qualification report hash: `sha256:1ada24828f159a386f5337441b347f204f1e29ca5165c70c69ebe4259f4e1252`

## Candidate inventory and decisions

99 candidates were inventoried; none was omitted and none qualified.

| Candidate | Classification | Final status | Decision |
|---|---|---|---|
| `baseline` | `REJECTED` | `REJECTED` | H2 out-of-sample failure |
| `fast_accumulation` | `REJECTED` | `REJECTED` | H2 out-of-sample failure |
| `fast_accumulation_breakeven_guard` | `MISSING_EVIDENCE` | `PIT_EVIDENCE_MISSING` | no complete immutable OOS, walk-forward, cost-tail-concentration, PIT, and controlled-holdout evidence for this exact preset |
| `fast_accumulation_breakout_retest` | `MISSING_EVIDENCE` | `PIT_EVIDENCE_MISSING` | no complete immutable OOS, walk-forward, cost-tail-concentration, PIT, and controlled-holdout evidence for this exact preset |
| `fast_accumulation_cut_no_progress` | `MISSING_EVIDENCE` | `PIT_EVIDENCE_MISSING` | no complete immutable OOS, walk-forward, cost-tail-concentration, PIT, and controlled-holdout evidence for this exact preset |
| `fast_accumulation_economics_guard` | `MISSING_EVIDENCE` | `PIT_EVIDENCE_MISSING` | no complete immutable OOS, walk-forward, cost-tail-concentration, PIT, and controlled-holdout evidence for this exact preset |
| `fast_accumulation_momentum_continuation` | `MISSING_EVIDENCE` | `PIT_EVIDENCE_MISSING` | no complete immutable OOS, walk-forward, cost-tail-concentration, PIT, and controlled-holdout evidence for this exact preset |
| `fast_accumulation_partial_trail` | `MISSING_EVIDENCE` | `PIT_EVIDENCE_MISSING` | no complete immutable OOS, walk-forward, cost-tail-concentration, PIT, and controlled-holdout evidence for this exact preset |
| `fast_accumulation_pullback_reclaim` | `MISSING_EVIDENCE` | `PIT_EVIDENCE_MISSING` | no complete immutable OOS, walk-forward, cost-tail-concentration, PIT, and controlled-holdout evidence for this exact preset |
| `fast_accumulation_strict` | `MISSING_EVIDENCE` | `PIT_EVIDENCE_MISSING` | no complete immutable OOS, walk-forward, cost-tail-concentration, PIT, and controlled-holdout evidence for this exact preset |
| `fast_accumulation_strict_1h` | `MISSING_EVIDENCE` | `PIT_EVIDENCE_MISSING` | no complete immutable OOS, walk-forward, cost-tail-concentration, PIT, and controlled-holdout evidence for this exact preset |
| `fast_accumulation_strict_30m` | `MISSING_EVIDENCE` | `PIT_EVIDENCE_MISSING` | no complete immutable OOS, walk-forward, cost-tail-concentration, PIT, and controlled-holdout evidence for this exact preset |
| `fast_accumulation_strict_cost_guard` | `MISSING_EVIDENCE` | `PIT_EVIDENCE_MISSING` | no complete immutable OOS, walk-forward, cost-tail-concentration, PIT, and controlled-holdout evidence for this exact preset |
| `fast_accumulation_strict_high_confidence` | `MISSING_EVIDENCE` | `PIT_EVIDENCE_MISSING` | no complete immutable OOS, walk-forward, cost-tail-concentration, PIT, and controlled-holdout evidence for this exact preset |
| `fast_accumulation_strict_low_frequency` | `MISSING_EVIDENCE` | `PIT_EVIDENCE_MISSING` | no complete immutable OOS, walk-forward, cost-tail-concentration, PIT, and controlled-holdout evidence for this exact preset |
| `fast_accumulation_strict_no_70_84_longs` | `MISSING_EVIDENCE` | `PIT_EVIDENCE_MISSING` | no complete immutable OOS, walk-forward, cost-tail-concentration, PIT, and controlled-holdout evidence for this exact preset |
| `fast_accumulation_strict_short_bias` | `MISSING_EVIDENCE` | `PIT_EVIDENCE_MISSING` | no complete immutable OOS, walk-forward, cost-tail-concentration, PIT, and controlled-holdout evidence for this exact preset |
| `funding-alpha/BreakoutFundingLong|long|240m` | `REJECTED` | `REJECTED` | failed net performance, robustness, and/or concentration gates |
| `funding-alpha/BreakoutFundingShort|short|5m` | `REJECTED` | `REJECTED` | failed net performance, robustness, and/or concentration gates |
| `funding-alpha/ConfirmedNegativeFundingLong|long|240m` | `REJECTED` | `REJECTED` | failed net performance, robustness, and/or concentration gates |
| `funding-alpha/ConfirmedPositiveFundingShort|short|5m` | `REJECTED` | `REJECTED` | failed net performance, robustness, and/or concentration gates |
| `funding-alpha/FundingFlipLong|long|120m` | `REJECTED` | `REJECTED` | candidate side/horizon row rejected after full retained coverage |
| `funding-alpha/FundingFlipLong|long|15m` | `REJECTED` | `REJECTED` | candidate side/horizon row rejected after full retained coverage |
| `funding-alpha/FundingFlipLong|long|240m` | `REJECTED` | `REJECTED` | candidate side/horizon row rejected after full retained coverage |
| `funding-alpha/FundingFlipLong|long|30m` | `REJECTED` | `REJECTED` | candidate side/horizon row rejected after full retained coverage |
| `funding-alpha/FundingFlipLong|long|5m` | `REJECTED` | `REJECTED` | candidate side/horizon row rejected after full retained coverage |
| `funding-alpha/FundingFlipLong|long|60m` | `REJECTED` | `REJECTED` | candidate side/horizon row rejected after full retained coverage |
| `funding-alpha/FundingFlipShort|short|120m` | `REJECTED` | `REJECTED` | candidate side/horizon row rejected after full retained coverage |
| `funding-alpha/FundingFlipShort|short|15m` | `REJECTED` | `REJECTED` | candidate side/horizon row rejected after full retained coverage |
| `funding-alpha/FundingFlipShort|short|240m` | `REJECTED` | `REJECTED` | candidate side/horizon row rejected after full retained coverage |
| `funding-alpha/FundingFlipShort|short|30m` | `REJECTED` | `REJECTED` | candidate side/horizon row rejected after full retained coverage |
| `funding-alpha/FundingFlipShort|short|5m` | `REJECTED` | `REJECTED` | candidate side/horizon row rejected after full retained coverage |
| `funding-alpha/FundingFlipShort|short|60m` | `REJECTED` | `REJECTED` | candidate side/horizon row rejected after full retained coverage |
| `funding-alpha/NegativeFundingLong|long|120m` | `REJECTED` | `REJECTED` | candidate side/horizon row rejected after full retained coverage |
| `funding-alpha/NegativeFundingLong|long|15m` | `REJECTED` | `REJECTED` | candidate side/horizon row rejected after full retained coverage |
| `funding-alpha/NegativeFundingLong|long|240m` | `REJECTED` | `REJECTED` | candidate side/horizon row rejected after full retained coverage |
| `funding-alpha/NegativeFundingLong|long|30m` | `REJECTED` | `REJECTED` | candidate side/horizon row rejected after full retained coverage |
| `funding-alpha/NegativeFundingLong|long|5m` | `REJECTED` | `REJECTED` | candidate side/horizon row rejected after full retained coverage |
| `funding-alpha/NegativeFundingLong|long|60m` | `REJECTED` | `REJECTED` | candidate side/horizon row rejected after full retained coverage |
| `funding-alpha/PositiveFundingShort|short|120m` | `REJECTED` | `REJECTED` | candidate side/horizon row rejected after full retained coverage |
| `funding-alpha/PositiveFundingShort|short|15m` | `REJECTED` | `REJECTED` | candidate side/horizon row rejected after full retained coverage |
| `funding-alpha/PositiveFundingShort|short|240m` | `REJECTED` | `REJECTED` | candidate side/horizon row rejected after full retained coverage |
| `funding-alpha/PositiveFundingShort|short|30m` | `REJECTED` | `REJECTED` | candidate side/horizon row rejected after full retained coverage |
| `funding-alpha/PositiveFundingShort|short|5m` | `REJECTED` | `REJECTED` | candidate side/horizon row rejected after full retained coverage |
| `funding-alpha/PositiveFundingShort|short|60m` | `REJECTED` | `REJECTED` | candidate side/horizon row rejected after full retained coverage |
| `funding-alpha/RegimeFundingLong|long|120m` | `REJECTED` | `REJECTED` | candidate side/horizon row rejected after full retained coverage |
| `funding-alpha/RegimeFundingLong|long|15m` | `REJECTED` | `REJECTED` | candidate side/horizon row rejected after full retained coverage |
| `funding-alpha/RegimeFundingLong|long|240m` | `REJECTED` | `REJECTED` | candidate side/horizon row rejected after full retained coverage |
| `funding-alpha/RegimeFundingLong|long|30m` | `REJECTED` | `REJECTED` | candidate side/horizon row rejected after full retained coverage |
| `funding-alpha/RegimeFundingLong|long|5m` | `REJECTED` | `REJECTED` | candidate side/horizon row rejected after full retained coverage |
| `funding-alpha/RegimeFundingLong|long|60m` | `REJECTED` | `REJECTED` | candidate side/horizon row rejected after full retained coverage |
| `funding-alpha/RegimeFundingShort|short|120m` | `REJECTED` | `REJECTED` | candidate side/horizon row rejected after full retained coverage |
| `funding-alpha/RegimeFundingShort|short|15m` | `REJECTED` | `REJECTED` | candidate side/horizon row rejected after full retained coverage |
| `funding-alpha/RegimeFundingShort|short|240m` | `REJECTED` | `REJECTED` | candidate side/horizon row rejected after full retained coverage |
| `funding-alpha/RegimeFundingShort|short|30m` | `REJECTED` | `REJECTED` | candidate side/horizon row rejected after full retained coverage |
| `funding-alpha/RegimeFundingShort|short|5m` | `REJECTED` | `REJECTED` | candidate side/horizon row rejected after full retained coverage |
| `funding-alpha/RegimeFundingShort|short|60m` | `REJECTED` | `REJECTED` | candidate side/horizon row rejected after full retained coverage |
| `funding-alpha/VolumeImbalanceFundingReversionProxyLong|long|120m` | `REJECTED` | `REJECTED` | failed net performance, robustness, and/or concentration gates |
| `funding-alpha/VolumeImbalanceFundingReversionProxyLong|long|15m` | `REJECTED` | `REJECTED` | failed net performance, robustness, and/or concentration gates |
| `funding-alpha/VolumeImbalanceFundingReversionProxyLong|long|240m` | `REJECTED` | `REJECTED` | failed net performance, robustness, and/or concentration gates |
| `funding-alpha/VolumeImbalanceFundingReversionProxyLong|long|30m` | `REJECTED` | `REJECTED` | failed net performance, robustness, and/or concentration gates |
| `funding-alpha/VolumeImbalanceFundingReversionProxyLong|long|5m` | `REJECTED` | `REJECTED` | failed net performance, robustness, and/or concentration gates |
| `funding-alpha/VolumeImbalanceFundingReversionProxyLong|long|60m` | `REJECTED` | `REJECTED` | failed net performance, robustness, and/or concentration gates |
| `funding-alpha/VolumeImbalanceFundingReversionProxyShort|short|120m` | `REJECTED` | `REJECTED` | failed net performance, robustness, and/or concentration gates |
| `funding-alpha/VolumeImbalanceFundingReversionProxyShort|short|15m` | `REJECTED` | `REJECTED` | failed net performance, robustness, and/or concentration gates |
| `funding-alpha/VolumeImbalanceFundingReversionProxyShort|short|240m` | `REJECTED` | `REJECTED` | failed net performance, robustness, and/or concentration gates |
| `funding-alpha/VolumeImbalanceFundingReversionProxyShort|short|30m` | `REJECTED` | `REJECTED` | failed net performance, robustness, and/or concentration gates |
| `funding-alpha/VolumeImbalanceFundingReversionProxyShort|short|5m` | `REJECTED` | `REJECTED` | failed net performance, robustness, and/or concentration gates |
| `funding-alpha/VolumeImbalanceFundingReversionProxyShort|short|60m` | `REJECTED` | `REJECTED` | failed net performance, robustness, and/or concentration gates |
| `phase11/CompressionVolumeBreakout|long|15m` | `REJECTED` | `REJECTED` | side/horizon row failed net and worst-period gates |
| `phase11/CompressionVolumeBreakout|long|240m` | `REJECTED` | `REJECTED` | side/horizon row failed net and worst-period gates |
| `phase11/CompressionVolumeBreakout|long|60m` | `REJECTED` | `REJECTED` | side/horizon row failed net and worst-period gates |
| `phase11/CompressionVolumeBreakout|short|15m` | `REJECTED` | `REJECTED` | side/horizon row failed net and worst-period gates |
| `phase11/CompressionVolumeBreakout|short|240m` | `REJECTED` | `REJECTED` | side/horizon row failed net and worst-period gates |
| `phase11/CompressionVolumeBreakout|short|60m` | `REJECTED` | `REJECTED` | side/horizon row failed net and worst-period gates |
| `phase11/RegimeTrendPullbackContinuation|long|120m` | `REJECTED` | `REJECTED` | side/horizon row failed; exact implementation is not committed/reproducible |
| `phase11/RegimeTrendPullbackContinuation|long|15m` | `REJECTED` | `REJECTED` | side/horizon row failed; exact implementation is not committed/reproducible |
| `phase11/RegimeTrendPullbackContinuation|long|240m` | `REJECTED` | `REJECTED` | side/horizon row failed; exact implementation is not committed/reproducible |
| `phase11/RegimeTrendPullbackContinuation|long|30m` | `REJECTED` | `REJECTED` | side/horizon row failed; exact implementation is not committed/reproducible |
| `phase11/RegimeTrendPullbackContinuation|long|60m` | `REJECTED` | `REJECTED` | side/horizon row failed; exact implementation is not committed/reproducible |
| `phase11/RegimeTrendPullbackContinuation|short|120m` | `REJECTED` | `REJECTED` | side/horizon row failed; exact implementation is not committed/reproducible |
| `phase11/RegimeTrendPullbackContinuation|short|15m` | `REJECTED` | `REJECTED` | side/horizon row failed; exact implementation is not committed/reproducible |
| `phase11/RegimeTrendPullbackContinuation|short|240m` | `REJECTED` | `REJECTED` | side/horizon row failed; exact implementation is not committed/reproducible |
| `phase11/RegimeTrendPullbackContinuation|short|30m` | `REJECTED` | `REJECTED` | side/horizon row failed; exact implementation is not committed/reproducible |
| `phase11/RegimeTrendPullbackContinuation|short|60m` | `REJECTED` | `REJECTED` | side/horizon row failed; exact implementation is not committed/reproducible |
| `phase12/DowntrendMidVolReliefLong240m` | `NEAR_MISS` | `NEAR_MISS` | failed fixed worst-quarter gate and cluster-independence evidence |
| `phase13/ContextFreeMomentumBreakoutProbe` | `INFRASTRUCTURE_PROBE` | `INSUFFICIENT_SAMPLE` | infrastructure proof only; one symbol-month; negative net result; leave-one-out segments unavailable |
| `price-alpha/BetaAgrees/long` | `MISSING_EVIDENCE` | `PIT_EVIDENCE_MISSING` | no complete exact-candidate qualification evidence |
| `price-alpha/BetaAgrees/short` | `MISSING_EVIDENCE` | `PIT_EVIDENCE_MISSING` | no complete exact-candidate qualification evidence |
| `price-alpha/BetaDiverges/long` | `MISSING_EVIDENCE` | `PIT_EVIDENCE_MISSING` | no complete exact-candidate qualification evidence |
| `price-alpha/BetaDiverges/short` | `MISSING_EVIDENCE` | `PIT_EVIDENCE_MISSING` | no complete exact-candidate qualification evidence |
| `price-alpha/CompressionBreakout/long` | `NEAR_MISS` | `NEAR_MISS` | bracket/concentration failure; fragile is not paper eligible |
| `price-alpha/CompressionBreakout/short` | `NEAR_MISS` | `NEAR_MISS` | bracket/concentration failure; fragile is not paper eligible |
| `price-alpha/ShockFade/long` | `REJECTED` | `REJECTED` | out-of-sample and/or concentration/sample gates failed |
| `price-alpha/ShockFade/short` | `MISSING_EVIDENCE` | `PIT_EVIDENCE_MISSING` | no complete exact-candidate qualification evidence |
| `price-alpha/TrendContinuation/long` | `MISSING_EVIDENCE` | `PIT_EVIDENCE_MISSING` | no complete exact-candidate qualification evidence |
| `price-alpha/TrendContinuation/short` | `MISSING_EVIDENCE` | `PIT_EVIDENCE_MISSING` | no complete exact-candidate qualification evidence |
| `price-alpha/VolumeMomentum/long` | `MISSING_EVIDENCE` | `PIT_EVIDENCE_MISSING` | no complete exact-candidate qualification evidence |
| `price-alpha/VolumeMomentum/short` | `MISSING_EVIDENCE` | `PIT_EVIDENCE_MISSING` | no complete exact-candidate qualification evidence |

## Qualification protocol and gates

No new research was performed, so no new-search protocol artifact was created. Existing evidence was assessed against the declared fail-closed gates embedded in the JSON report: exact immutable data/PIT identity, at least 300 independent clusters/decisions, PF >= 1.10 and positive expectancy/confidence bound, worst-period PF >= 0.95, bounded concentration, OOS and walk-forward stability, parameter-neighborhood stability, realistic 10 bps total stress, leakage safety, simplicity, exact implementation, PIT evidence, and RIF-controlled holdout exposure.

## Data and holdout controls

No Engine candidate has a complete accepted `ak.rif.research_identity.v1`, candidate-bound Historian PIT evidence, or accepted RIF holdout exposure. The final holdout was not inspected or used. The accepted Historian/RIF fixture belongs to a synthetic Historian candidate and was not substituted.

## Selected candidate, frozen identity, parity, and registration

No candidate was selected. No descriptor was frozen, no parity fixtures were created, and no Engine candidate-registration request was emitted. RIF did not accept or authorize any candidate.

## Verification

| Command | Status | Exit |
|---|---|---:|
| `gofmt -w internal/app/pr4b0_candidate_qualification.go internal/app/pr4b0_candidate_qualification_test.go internal/qualification/qualification.go internal/qualification/qualification_test.go` | PASS | 0 |
| `GOWORK=off go mod tidy` | PASS | 0 |
| `git diff --exit-code -- go.mod go.sum` | PASS | 0 |
| `GOWORK=off go vet ./...` | PASS | 0 |
| `GOWORK=off go test ./...` | PASS | 0 |
| `GOWORK=off go test -race ./...` | PASS | 0 |
| `GOWORK=off go build ./...` | PASS | 0 |
| `GOWORK=off make verify` | PASS | 0 |
| `git diff --check` | PASS | 0 |
| `GOWORK=off go run ./cmd/ak-engine pr4b0-candidate-qualification --out-dir runs/reports --resulting-commit 25efa97ca89f8dcb724f9872e798bc789123caac --verification-complete --fresh-clone-commit 25efa97ca89f8dcb724f9872e798bc789123caac` | PASS | 0 |
| `find runs/reports -maxdepth 1 -type f -name 'pr4b0_*.json' -print0 | xargs -0 -n1 jq -e .` | PASS | 0 |
| `test -z "$(rg -l '/h[o]me/|/U[s]ers/|[A-Za-z]:\\\\' internal/app/pr4b0_candidate_qualification.go internal/app/pr4b0_candidate_qualification_test.go internal/qualification runs/reports/pr4b0_*)"` | PASS | 0 |
| `test -z "$(rg -l 'github\.com/.+/(ak-rif|ak-historian|ak-trader)' --glob '*.go' internal/app/pr4b0_candidate_qualification.go internal/app/pr4b0_candidate_qualification_test.go internal/qualification)"` | PASS | 0 |
| `test -z "$(rg -l -i '(api[_-]?key|api[_-]?secret|private[_-]?key|access[_-]?token|password)[[:space:]]*[:=][[:space:]]*\"' internal/app/pr4b0_candidate_qualification.go internal/app/pr4b0_candidate_qualification_test.go internal/qualification runs/reports/pr4b0_*)"` | PASS | 0 |
| `test -z "$(rg -l -- '-----BEGIN [A-Z ]*PRIVATE KEY-----' internal/app/pr4b0_candidate_qualification.go internal/app/pr4b0_candidate_qualification_test.go internal/qualification runs/reports/pr4b0_*)"` | PASS | 0 |
| `test -z "$(rg -l 'n[e]t/http|os[.]Getenv|exec[.]Command|w[e]bsocket|N[e]wClient' internal/app/pr4b0_candidate_qualification.go internal/qualification)"` | PASS | 0 |
| `test -z "$(find runs/reports -maxdepth 1 -type f \( -name 'pr4b0_candidate_qualification_protocol.*' -o -name 'pr4b0_frozen_candidate.*' -o -name 'pr4b0_candidate_registration_request.*' \) -print)"` | PASS | 0 |

Fresh clone: `PASS`; no sibling AK repositories: `true`.

## Security and boundaries

The JSON report records all required security-review areas. No unresolved in-scope high-severity finding remains. No paper evaluator was implemented. RIF did not authorize a candidate. No candidate was promoted. Trader behavior was unchanged. Historian was not modified. No mainnet or authenticated exchange behavior was run.

## Deferred work and recommendation

A separately pre-registered bounded research phase must establish immutable dataset/manifest/PIT identity, complete candidate context, finite search budget, and RIF-controlled one-time holdout treatment. Do not begin PR4B1.

Recommended next phase: `SEPARATELY_PRE_REGISTERED_BOUNDED_RESEARCH_PHASE`.
