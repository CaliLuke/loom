# OpenAPI Schema Deduplication

## Goal

Reduce duplicate hashed component schemas in generated OpenAPI when the shapes are structurally identical, so the contract is easier to diff, reference, and reconcile over time.

## Problem

`goa-light` already uses stable hash-based suffixes to avoid collisions, but identical or near-identical schemas can still be emitted under separate names. That creates noise for:

- contract diffs
- downstream schema references
- review of generator changes

## Scope

Framework work only:

- component naming and reuse in OpenAPI generation
- schema identity comparison
- test and validation coverage

Out of scope:

- changing service DSL names in applications
- runtime serialization behavior

## Desired Outcome

- structurally identical schemas are emitted once and reused by `$ref`
- hash suffixes remain available only for real collisions
- generated component naming stays deterministic across runs

## Work Plan

1. Audit where component schemas are registered and deduplicated today.
2. Define a canonical structural identity for OpenAPI component schemas.
3. Distinguish true collisions from equivalent schemas with different construction paths.
4. Reuse the first stable component name for equivalent schemas.
5. Keep hash suffixes only for non-equivalent same-name conflicts.
6. Add golden tests for duplicated wrapper and request-body shapes.

## Design Constraints

- Deduplication must be deterministic.
- Deduplication must not merge semantically distinct schemas that happen to look similar because of generator lossiness.
- Keep the current contract-stability rules centered in `goa-light`.

## Risks

- Over-aggressive deduplication could collapse intentionally separate schemas.
- Structural comparison must normalize examples and non-semantic ordering correctly.

## Finish Criteria

- Known duplicate schema pairs collapse to a single component.
- Stable names remain stable across repeated generation.
- Golden tests cover:
  - repeated request body wrappers
  - repeated union wrappers
  - true collision cases that still need suffixes
