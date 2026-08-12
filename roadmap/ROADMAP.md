# Loom Roadmap

This file contains only unresolved, framework-owned work. Completed work belongs
in the implementation, tests, user documentation, release notes, and Git
history—not in the roadmap.

## Priority 1: Own OpenAPI Collision Identities

Loom currently uses `github.com/gohugoio/hashstructure` to derive validation
hashes that participate in hash-suffixed OpenAPI component names. That makes an
external reflection-hashing algorithm part of generated contract identity: the
1.0 dependency update corrected set hashing and consequently renamed existing
fallback components.

Replace that dependency with a Loom-owned canonical fingerprint over the schema
and validation IR that actually affects generated OpenAPI shape.

### Required design properties

- Define canonical encodings for every schema and validation field that affects
  structural identity, including explicit ordering and duplicate semantics for
  sets, slices, maps, and nested attributes.
- Keep fingerprints deterministic across processes, platforms, Go releases,
  map iteration order, and unrelated dependency updates.
- Keep explicit `openapi:typename` and `openapi:component:*` metadata as the
  public naming contract. Do not add a new DSL naming surface solely to replace
  the internal hash.
- Decide explicitly whether the Loom-owned algorithm preserves the 1.0-era
  fallback names or introduces one documented regeneration transition. Do not
  allow accidental name churn.
- Remove `hashstructure` from framework and fixture module graphs, including the
  temporary code-generation dependency import in `dsl/doc.go`.

### Acceptance criteria

- Add table-driven characterization tests for each validation field and for
  order-independent collections, duplicates, recursive attributes, and
  structurally distinct collision candidates.
- Pair exact fingerprint assertions with rendered JSON and YAML OpenAPI golden
  coverage for hash-suffixed schema, request-body, and response components.
- Prove that repeated runs and reordered equivalent inputs produce identical
  component names, while contract-relevant changes produce different names.
- Run the OpenAPI consumer smoke tests and document any intentional downstream
  SDK type rename in the corresponding release notes.
- Remove this roadmap item when the implementation, tests, dependency cleanup,
  and durable user guidance are complete.

## Decision Rules

Add roadmap work only when it is unresolved, framework-owned, and backed by a
concrete defect, maintenance cost, or downstream consumer need. Remove an item
as soon as the implementation, tests, and durable documentation are complete.

Do not add compatibility work solely to preserve historical upstream behavior,
runtime security policy better owned by applications, or speculative DSL
surface without a current consumer.
