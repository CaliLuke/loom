# OpenAPI Union Discriminators

Status: completed on 2026-03-18

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

## Delivered Outcome

- wrapper unions emit `oneOf` refs to generated branch-envelope component schemas
- generated schemas include `discriminator.propertyName`
- generated schemas include stable discriminator mappings to component refs
- examples align with discriminator-based branch selection
- rendered OpenAPI output is covered by `libopenapi`-validated regression tests

## What Changed

1. The OpenAPI v3 schema generator now emits `oneOf` refs plus discriminator mappings for wrapper unions.
2. The generator creates stable branch-envelope component schemas instead of leaving branch selection implicit under `value.anyOf`.
3. Unit coverage now checks request and response unions, custom discriminator field names, and renamed-type discriminator stability.
4. Rendered-spec tests now assert the generated OpenAPI shape and validate it through `libopenapi`.

## Design Constraints

- Keep the contract policy in `goa-light`, not in downstream plugins or consumers.
- Do not require application-specific meta tags for the common `type`/`value` pattern.
- Preserve stable schema names and refs while adding discriminator metadata.

## Remaining Follow-Up

- Deduplicate structurally identical schemas that still produce multiple envelope or body component names.
- Decide whether additional wrapper-union golden fixtures should be added to the checked-in OpenAPI golden set instead of remaining in assertion-based regression tests.

## Finish Criteria

- Generated OpenAPI 3.1 includes discriminator metadata for supported wrapper unions.
- Existing union examples still validate.
- Unit and rendered-spec tests cover:
  - simple wrapper union
  - nested wrapper union
  - custom discriminator field names
  - discriminator stability for renamed variant types
