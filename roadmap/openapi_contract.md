# OpenAPI Contract

## Goal

Treat the generated OpenAPI 3.1 document as a machine-consumable contract artifact, not just human documentation.

## Status

### Completed

- Remove OpenAPI v2 generation and keep the framework on OpenAPI 3.x only.
- Upgrade OpenAPI generation to 3.1 / JSON Schema 2020-12.
- Use `libopenapi` in the test harness for spec parsing and validation.
- Align cookie documentation with actual wire-format serialization.
- Use stable hash-based schema collision suffixes.
- Canonicalize generated `operationId` values into a deterministic normalized form.
- Emit operation-level security requirements for secured endpoints, and explicit
  empty `security: []` arrays for `NoSecurity()` operations.
- Prune unreferenced generated component schemas so the published contract only
  contains reachable request/response shapes.

### Next

- keep OpenAPI 3.1 / JSON Schema 2020-12 as the canonical output
- improve contract stability for downstream consumers
- prefer semantic accuracy over preserving historical document shape

## Working Rules

### Keep Outsourcing Commodity Validation

Continue using libraries for:

- OpenAPI parsing and validation
- spec sanity checks
- protocol-level correctness checks in tests

Avoid reintroducing bespoke parsing or validator logic where standard libraries already do the job well.

### Stability Rules

Keep these policies in `goa-light`, not in plugins:

- stable `operationId`
- stable schema naming
- explicit operation-level security semantics, including public-operation
  overrides
- truthful response/body/security modeling
- no dead generated component schemas in published output
- stable and accurate examples where possible

## Backlog

- continue improving OpenAPI output where it materially helps machine consumers
- keep contract-shape decisions centralized in `goa-light`
- do not let `goa-ai` develop its own contract-stability layer
