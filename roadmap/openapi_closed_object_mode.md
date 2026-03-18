# OpenAPI Closed Object Mode

## Goal

Add a contract-focused OpenAPI mode that defaults normal object schemas to closed shapes, making machine reconciliation stricter and less ambiguous while preserving explicit dictionary types where needed.

## Problem

Many generated object schemas remain effectively open. That is convenient, but weak for machine consumers trying to detect drift or unexpected fields. A stricter contract mode would improve:

- reconciliation after feature changes
- downstream validator behavior
- confidence that generated objects match declared DSL fields

## Scope

Framework work only:

- OpenAPI / JSON Schema emission policy
- object schema defaults
- explicit escape hatches for map-like types

Out of scope:

- changing application DSL object definitions
- changing wire behavior at runtime

## Desired Outcome

- normal object schemas can be emitted as closed by default in contract mode
- explicit `MapOf(...)` and intentionally open types remain open
- the mode is deterministic and testable

## Work Plan

1. Classify current schema emission paths into:
   - regular object types
   - explicit maps / dictionaries
   - special framework-owned open shapes
2. Decide whether closed mode uses `additionalProperties: false`, `unevaluatedProperties: false`, or both depending on schema composition rules.
3. Add a framework-level switch or policy hook for contract-oriented output.
4. Preserve open behavior for explicit dictionaries and dynamic metadata maps.
5. Add validation tests against `libopenapi` and JSON Schema 2020-12 expectations.

## Design Constraints

- Closed mode should be opt-in unless there is a strong reason to change the default globally.
- Explicit dynamic maps must remain possible and truthful.
- The emitted shape should remain valid OpenAPI 3.1 / JSON Schema 2020-12.

## Risks

- Closed schemas interact subtly with composed schemas and unions.
- Some downstream tools still handle `unevaluatedProperties` inconsistently.

## Finish Criteria

- Contract mode can emit closed object schemas for ordinary object types.
- Explicit maps remain open.
- Golden tests cover:
  - standard object type
  - nested object type
  - map/dictionary type
  - union or composed object case
