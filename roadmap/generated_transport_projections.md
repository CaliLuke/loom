# Generated Transport Projections

## Goal

Generate transport projection helpers from canonical result types so applications stop hand-copying fields into sibling REST, JSON-RPC, or MCP result shapes.

## Status

Completed for the canonical `ResultType`/`View` pattern.

## Problem

Projection drift is still easy when one canonical service result is manually reshaped for another transport or tool surface. The issue is not just missing helpers; the framework currently leaves enough boundary work to application code that drift remains likely.

## Scope

Framework work only:

- codegen for projection helpers
- view/result modeling guidance
- reusable generated conversion patterns

Out of scope:

- application-specific adapter logic
- custom non-Loom transport runtimes

## Shipped Outcome

- applications can define one canonical result type and one or more projected views
- `loom` now generates exported projection helpers in the service package for canonical result-to-view and view-to-result conversion
- nested structs, slices, maps, unions, collections, and optionals are projected recursively through the existing transform-helper machinery
- wrappers such as `NewViewed...` and `New...` now build on those exported helpers instead of private-only projection functions

## Implementation Notes

1. The supported framework pattern is now:
   - canonical service result type
   - projected Loom view type in the generated `views` package
   - exported service-package helpers for projection in both directions
2. The generated helper names are stable and typed:
   - `Project<ResultType>[ViewSuffix](...)`
   - `New<ResultType>From<ProjectedType>[ViewSuffix](...)`
3. Existing recursive transform generation is reused rather than duplicated in transport generators.
4. The capability is intentionally based on `ResultType` / `View`; arbitrary sibling non-view DTO mapping remains out of scope.

## Design Constraints

- Generated helpers should use existing Loom type metadata and naming scopes.
- Do not force applications into transport-specific wrapper types when a view is sufficient.
- Keep helper visibility minimal and generator output predictable.

## Risks

- Projection generation can become too magical if the source/target relationship is ambiguous.
- View semantics may be sufficient for some cases and insufficient for others; the framework needs a clear rule for both.

## Verification

- `go test -update ./codegen/service`
- `go test ./codegen/service/...`
- `go test ./http/codegen/...`
- `go test ./grpc/...`
- `go test ./jsonrpc/...`
