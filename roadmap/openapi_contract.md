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
- Hoist repeated OpenAPI parameters into `components.parameters` and rewrite
  repeated inline operation/path parameter occurrences to `$ref`s with stable
  component names.
- Prune unreferenced generated component schemas so the published contract only
  contains reachable request/response shapes.
- Suppress invalid closed-object union-wrapper examples, honor
  `Meta("openapi:example", "false")` on wrapper fields, keep SSE response
  statuses on normal HTTP success codes, and normalize binary request examples
  to JSON/OpenAPI string form.
- Suppress invalid synthesized examples for closed-object direct-union
  collections, including response/media-type examples for arrays whose element
  shape contains discriminator-driven closed unions.
- Omit transport-level media-type examples for streaming responses instead of
  synthesizing SSE/WebSocket payload examples from partial field examples.
- Keep a non-trivial specimen API under `http/codegen/testdata` and validate its
  rendered OpenAPI with both `libopenapi` and Redocly as a closed-loop contract
  check.

### Next

- keep OpenAPI 3.1 / JSON Schema 2020-12 as the canonical output
- improve contract stability for downstream consumers
- prefer semantic accuracy over preserving historical document shape
- use the live Auto-K contract as a forcing function for framework-level OpenAPI
  improvements

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
- no transport-level streaming examples unless the generator can prove they
  match the published schema
- reusable shared components for repeated parameters, request bodies, responses,
  headers, and examples
- standards-first error contracts and auth field annotations that downstream SDK
  generators can consume directly

## Framework Improvement Tracks

These are framework changes, not app-local cleanup. They should be implemented
in `goa-light` so downstream services inherit better OpenAPI contracts by
default.

### 1. Shared Component Deduplication Beyond Schemas

The current generator already prunes and deduplicates schemas, but it still
inlines repeated parameters, request bodies, responses, and examples.

Add framework support to:

- hoist repeated path and query parameters into `components.parameters`
- hoist repeated request bodies into `components.requestBodies`
- hoist repeated success and error responses into `components.responses`
- hoist repeated headers into `components.headers`
- hoist named examples into `components.examples`

The generator should prefer component reuse whenever the public contract shape
is equivalent and the component name is stable.

### 2. SDK-Safe Operation And Tag Naming

The framework should expose a first-class public naming policy for:

- `operationId`
- tag names
- security scheme names

Requirements:

- generated `operationId` values must be stable and SDK-safe
- default operation tags must match the top-level tag objects exactly
- internal transport naming should not leak into published contract names unless
  the API author opts into it explicitly

### 3. Standards-First Error Contracts

The framework should stop treating Goa-specific error envelopes as the only
default machine contract for errors.

Add framework support to:

- generate `application/problem+json` or a close RFC 9457-compatible profile
- keep a stable machine-readable code on every typed error
- reuse shared error responses from `components.responses`
- let API authors opt into richer typed errors without handwritten OpenAPI
  patches

### 4. Read-Only, Write-Only, And Deprecation Metadata

The schema model already supports some OpenAPI field metadata, but the DSL and
generator contract should go further.

Add framework support to:

- declare request-only fields as `writeOnly`
- declare response-only or server-computed fields as `readOnly`
- declare deprecated fields, parameters, and operations centrally
- split request and response schemas automatically when the same domain type
  would otherwise leak secrets or server-managed fields both ways

This is especially important for auth, session, secret, and token-heavy APIs.

### 5. Better JSON Schema 2020-12 Coverage

The framework should lean harder on the 3.1 / 2020-12 feature set it already
targets.

Add framework support to:

- emit `$id` and other modern JSON Schema identifiers consistently
- expose `const`, `allOf`, and richer object-composition controls where they
  materially improve client generation
- keep discriminated unions first-class for typed event and command envelopes
- support more explicit binary / encoded payload metadata such as
  `contentEncoding` and `contentMediaType` where relevant

### 6. Truthful Async Contract Modeling

The current OpenAPI output can describe SSE response media types, but WebSocket
and stream message contracts still need a stronger story.

Add framework support to:

- keep HTTP handshake documentation in OpenAPI
- publish message envelopes for SSE and WebSocket endpoints in a framework-owned
  async contract artifact or documented extension surface
- avoid fake JSON response bodies on `101` websocket upgrades
- let downstream consumers generate async clients from a truthful contract

### 7. Response Links For Workflow APIs

Auto-K has many workflow-style operations where a successful mutation naturally
points to the next resource or follow-up operation.

Add framework support to:

- define links from create operations to fetch operations
- define links from queued or asynchronous operations to status operations
- define links from list results to item-level operations where identifiers are
  already present in the payload

### 8. Contract Linting And Consumer Smoke Tests

The framework should enforce high-signal OpenAPI rules directly in its own test
suite.

Add checks for:

- tag mismatches between operations and top-level tag objects
- unsafe `operationId` values
- repeated inline parameter / request / response shapes that should be
  componentized
- missing `readOnly` / `writeOnly` on obvious secret-bearing fields
- invalid async handshake modeling

Pair those checks with downstream SDK smoke generation against at least one
TypeScript target and one Go target.

## Backlog

- continue improving OpenAPI output where it materially helps machine consumers
- keep contract-shape decisions centralized in `goa-light`
- use the Auto-K contract checklist as a real-world acceptance surface:
  [Auto-K OpenAPI Contract Checklist](./autok_openapi_contract_checklist.md)
- implement shared-component reuse for parameters, request bodies, responses,
  headers, and examples
- extend the completed parameter-componentization pass to request bodies,
  responses, headers, and examples
- add SDK-safe naming policy for operation IDs and tags
- add standards-first typed error contracts
- add first-class `readOnly`, `writeOnly`, and deprecation semantics to the DSL
  and generator path
- add a truthful async contract story for SSE and WebSocket endpoints
- keep the specimen matrix broad enough to cover:
  - form and multipart request bodies
  - closed-object union wrappers and union collections
  - result views and collections
  - SSE and WebSocket streaming response shapes
  - streaming endpoints whose field-level examples are incomplete or
    intentionally sparse
