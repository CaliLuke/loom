# OpenAPI Contract

## Goal

Treat the generated OpenAPI 3.1 document as a machine-consumable contract artifact, not just human documentation.

## Current Focus

- keep OpenAPI 3.1 / JSON Schema 2020-12 as the canonical output
- improve contract stability for downstream consumers
- prefer semantic accuracy over preserving historical document shape

## Priorities

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
- truthful response/body/security modeling
- stable and accurate examples where possible

## Follow-Up Work

- continue improving OpenAPI output where it materially helps machine consumers
- keep contract-shape decisions centralized in `goa-light`
- do not let `goa-ai` develop its own contract-stability layer
