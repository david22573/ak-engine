# Task Plan

## Goal
Implement Phase 10.2 research-only OOS failure postmortem and multi-symbol alpha rediscovery. Generate rejection, decay, multisymbol baseline, and leaderboard reports without touching `ak-trader` or any runtime trading behavior.

## Phases

- [x] **Phase 1: Restore And Rescope**
  - Read current planning files.
  - Confirm hard boundaries: `ak-engine` only unless historian is strictly needed; no `ak-trader`; no runtime, testnet, shadow, promotion, exchange, order, or config flow.
- [x] **Phase 2: Map Existing Research Surface**
  - Locate alpha baseline evaluator, compression breakout reports, feature/regime readers, and CLI registration.
  - Identify available 2023 datasets and prior report artifacts.
- [x] **Phase 3: Implement Research Commands**
  - Add `analyze-candidate-decay`.
  - Add or extend `evaluate-alpha-baselines-multisymbol`.
  - Ensure outputs include Markdown and JSON.
- [x] **Phase 4: Generate Phase 10.2 Reports**
  - Generate formal rejection record for `RegimeAwareCompressionBreakout_LONG`.
  - Generate H1 vs H2 decay analysis.
  - Generate multi-symbol baseline report and candidate leaderboard.
- [x] **Phase 5: Tests And Verification**
  - Add tests for rejection/verdict logic, H2 overrides, missing data reporting, leakage, and no `ak-trader` imports.
  - Run `go test ./...` in `ak-engine`.

## Current Status
- `in_progress`: Phase 3 implementation

## Errors Encountered
| Error | Attempt | Resolution |
|-------|---------|------------|
