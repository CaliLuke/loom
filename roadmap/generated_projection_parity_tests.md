# Generated Projection Parity Tests

## Goal

Generate parity tests and guardrails that fail when transport projections drift from their canonical result shapes.

## Problem

Even with better modeling, drift can still sneak in when generator output changes or application code bypasses the intended projection path. The framework should make projection incompleteness visible in tests, not at runtime.

## Scope

Framework work only:

- generated tests or fixtures
- golden/parity harness support
- CI-oriented guardrails for projection stability

Out of scope:

- application-specific manual assertions
- broad integration test orchestration unrelated to projection parity

## Desired Outcome

- generated parity tests can verify projected output completeness
- generated goldens make projection-shape changes explicit in review
- applications get a framework-standard guardrail instead of inventing custom parity tests

## Work Plan

1. Define what parity means for projected types:
   - declared fields are present
   - optional fields preserve presence semantics
   - nested projections stay complete
2. Extend generator test infrastructure to emit parity fixtures for supported projections.
3. Generate representative source values for canonical result types.
4. Compare generated projected values or serialized JSON against the declared transport contract.
5. Add clear failure messages that point to the missing or mismatched field.

## Design Constraints

- Generated tests should stay deterministic and cheap.
- The harness should validate framework semantics, not random example generation.
- Guardrails belong in `goa-light` generators and shared framework test infrastructure, not in app-specific CI scripts alone.

## Risks

- Auto-generated sample values can become noisy if not canonicalized.
- Tests can become brittle if examples, ordering, or non-semantic formatting leaks into parity assertions.

## Finish Criteria

- Supported projections get generated parity coverage.
- Missing-field regressions are caught by generator tests.
- The framework has a documented pattern for CI review of projection-affecting generator diffs.
