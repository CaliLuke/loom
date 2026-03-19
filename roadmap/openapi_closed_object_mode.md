# OpenAPI Closed Object Mode

## Goal

Add a contract-focused OpenAPI mode that defaults normal object schemas to closed shapes, making machine reconciliation stricter and less ambiguous while preserving explicit dictionary types where needed.

## Status

Completed.

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

## Shipped Outcome

- API-level `Meta("openapi:closed-objects", "true")` enables closed-object contract mode for generated OpenAPI.
- Ordinary object schemas emit `additionalProperties: false`.
- Wrapper unions emit `unevaluatedProperties: false` on the composed outer schema.
- Explicit dictionary shapes such as `MapOf(...)` remain open.
- The generated 3.1 spec is validated with `libopenapi`, and regression coverage includes standard, nested, map, and union cases.

## Implementation Notes

1. Closed-object mode is opt-in and controlled from API metadata rather than changing the global default.
2. Plain object schemas close with `additionalProperties: false`.
3. Composed union wrapper schemas close with `unevaluatedProperties: false` so branch refs and discriminators remain valid under JSON Schema 2020-12.
4. Explicit maps and dictionary-like schemas still emit schema-valued `additionalProperties`.
5. The behavior is covered both at the schema-builder level and in rendered-spec assertions.

## Design Constraints

- Closed mode should be opt-in unless there is a strong reason to change the default globally.
- Explicit dynamic maps must remain possible and truthful.
- The emitted shape should remain valid OpenAPI 3.1 / JSON Schema 2020-12.

## Risks

- Closed schemas interact subtly with composed schemas and unions.
- Some downstream tools still handle `unevaluatedProperties` inconsistently.

## Verification

- `go test ./http/codegen/openapi/v3`
- `go test ./http/codegen/openapi/...`
- `go test ./http/codegen/...`
