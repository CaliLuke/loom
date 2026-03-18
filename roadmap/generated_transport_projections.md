# Generated Transport Projections

## Goal

Generate transport projection helpers from canonical result types so applications stop hand-copying fields into sibling REST, JSON-RPC, or MCP result shapes.

## Problem

Projection drift is still easy when one canonical service result is manually reshaped for another transport or tool surface. The issue is not just missing helpers; the framework currently leaves enough boundary work to application code that drift remains likely.

## Scope

Framework work only:

- codegen for projection helpers
- view/result modeling guidance
- reusable generated conversion patterns

Out of scope:

- application-specific adapter logic
- custom non-Goa transport runtimes

## Desired Outcome

- applications can define one canonical result type and one or more projected views
- `goa-light` generates conversion helpers for projected transport results
- nested structs, slices, maps, and optionals are projected recursively

## Work Plan

1. Audit current Goa projection/view support and identify where it stops short for sibling transport DTOs.
2. Define the canonical framework pattern:
   - canonical result type
   - projected transport view or sibling result type
   - generated converter
3. Prefer `ResultType` / `View` semantics where possible instead of inventing parallel DSL.
4. Add codegen for strongly typed projection helpers.
5. Support optional field propagation and nested collection projection.
6. Add generator tests that prove application code can call the helper instead of rewriting field copies.

## Design Constraints

- Generated helpers should use existing Goa type metadata and naming scopes.
- Do not force applications into transport-specific wrapper types when a view is sufficient.
- Keep helper visibility minimal and generator output predictable.

## Risks

- Projection generation can become too magical if the source/target relationship is ambiguous.
- View semantics may be sufficient for some cases and insufficient for others; the framework needs a clear rule for both.

## Finish Criteria

- `goa-light` can generate typed projection helpers for canonical-to-transport result shapes.
- Nested and optional fields are preserved correctly.
- Golden coverage includes at least:
  - flat struct projection
  - nested struct projection
  - slice-of-struct projection
  - optional field propagation
