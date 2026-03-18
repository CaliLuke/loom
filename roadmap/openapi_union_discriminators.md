# OpenAPI Union Discriminators

## Goal

Emit machine-usable OpenAPI 3.1 discriminators for wrapper-style Goa unions so downstream consumers can reconcile contract branches without guessing from examples or `anyOf` ordering.

## Problem

`goa-light` already preserves union tag semantics in code generation, but the published OpenAPI for wrapper unions still degrades into object shells that contain:

- a `type` field with enum values
- a `value` field with `anyOf` branches
- no OpenAPI `discriminator`

That shape is readable, but it is weak for:

- automated contract reconciliation
- stable code generation in downstream tools
- branch-specific validation and diffing

## Scope

Framework work only:

- Goa DSL metadata to OpenAPI projection
- OpenAPI schema generation for wrapper unions
- tests and golden files in `goa-light`

Out of scope:

- application-specific contract redesign
- MCP runtime behavior
- manual schema patching in generated output

## Desired Outcome

- wrapper unions emit `oneOf` instead of ambiguous `anyOf` where the branch is exclusive
- generated schemas include `discriminator.propertyName`
- generated schemas include stable discriminator mappings to component refs
- examples align with discriminator-based branch selection

## Work Plan

1. Audit current union-shape emission paths in OpenAPI generation.
2. Define the exact criteria for when a Goa union can safely emit `oneOf` plus discriminator.
3. Reuse existing Goa union metadata instead of inventing new app-facing DSL.
4. Add generator support for discriminator mappings on wrapper unions.
5. Add golden coverage for nested and reused wrapper unions.
6. Validate generated specs with `libopenapi`.

## Design Constraints

- Keep the contract policy in `goa-light`, not in `goa-ai`.
- Do not require application-specific meta tags for the common `type`/`value` pattern.
- Preserve stable schema names and refs while adding discriminator metadata.

## Risks

- Some unions may not be exclusive enough for `oneOf`; those cases need a clear fallback rule.
- Discriminator mappings can become unstable if component naming is unstable.

## Finish Criteria

- Generated OpenAPI 3.1 includes discriminator metadata for supported wrapper unions.
- Existing union examples still validate.
- Golden tests cover at least:
  - simple wrapper union
  - nested wrapper union
  - reused wrapper union component
