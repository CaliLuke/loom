# OpenAPI Contract

## Goal

Treat the generated OpenAPI 3.1 document as a machine-consumable contract
artifact, not just human documentation.

## Canonical Plan

This document is the canonical framework plan for remaining OpenAPI work in
`goa-light`.

Do not create parallel plan docs for the same work. Downstream audit notes may
capture evidence, but backlog state and sequencing belong here.

## Completed Foundation

The following framework work is already done and should not be re-opened as new
roadmap items unless a regression appears:

- OpenAPI 3.x only, with OpenAPI 3.1 / JSON Schema 2020-12 as the canonical
  output.
- `libopenapi`-backed validation in the test harness.
- Stable schema naming for explicit HTTP body types, with generation failure on
  conflicting public names instead of hash-suffixed leakage.
- Canonicalized `operationId` generation.
- Service-level OpenAPI tag inheritance onto operations and file servers.
- Operation-level security emission for secured endpoints and explicit
  `security: []` for `NoSecurity()`.
- Reusable OpenAPI components for repeated path/query/header/cookie parameters.
- Reusable OpenAPI components for repeated request bodies, headers, named
  examples, and structurally identical no-body responses.
- Attribute-level schema metadata for `readOnly`, `writeOnly`, `deprecated`,
  `contentEncoding`, and `contentMediaType`.
- Documentation-only `OpenAPIBody(...)` support for
  `SkipResponseBodyEncodeDecode()` responses.
- Pruning of unreferenced generated component schemas.
- Closed-object union example suppression and related invalid-example cleanup.
- Streaming example suppression for SSE and WebSocket response media types.
- Truthful SSE handshake modeling in ordinary HTTP success responses with
  `text/event-stream`.
- Typed IR ownership for schema, parameter, operation-metadata, and reusable
  component analysis, leaving `http/codegen/openapi/v3` mostly as a renderer.

## Not Future Work

The following items sometimes still appear in audit notes, but they are already
framework capabilities rather than future roadmap units:

- SDK-safe `operationId` cleanup.
- Public tag alignment.
- Stable schema naming for explicit body/result projections.
- Broad `readOnly` / `writeOnly` support in the schema generator.
- Baseline named-example hoisting support.
- `OpenAPIBody(...)` for skip-encode/manual-response endpoints.
- SSE response status and `text/event-stream` contract correctness.

## Working Rules

- Keep contract-shape decisions in `goa-light`, not in per-app patches.
- Prefer semantic accuracy over preserving historical document shape.
- Do not add DSL surface unless it removes repeated app-local glue or repeated
  downstream contract drift.
- For each unit below, add direct seam tests first, then rendered-spec coverage,
  then `libopenapi` validation where output shape matters.
- Treat the Auto-K contract as an acceptance surface, not as the source of
  framework truth.

## Open Units

Each unit below is intentionally self-contained: it names the framework gap, the
work it owns, what it does not own, and the acceptance bar.

### Unit 1: Payload-Bearing Response Reuse

Status: planned

Problem:
Repeated success and error responses that carry equivalent payload schemas still
remain inline or reuse poorly unless they collapse to the small subset already
handled today.

Scope:

- Extend response componentization beyond structurally identical no-body
  responses.
- Reuse repeated payload-bearing responses when description, headers, content
  type, and schema shape are equivalent enough for a shared public contract.
- Cover common array, primitive/text, and equivalent error payload responses.

Out of scope:

- Problem-document migration.
- New DSL naming surface.

Acceptance:

- Repeated payload-bearing responses hoist into `components.responses` when the
  public contract shape is equivalent.
- The emitted response components validate through `libopenapi`.
- Regression tests cover both direct IR/componentization seams and rendered
  OpenAPI.

### Unit 2: Public Component Naming And Adapter Hygiene

Status: planned

Problem:
Some reusable public components still get operation-derived names or hash
suffixes where a stable public name should exist, and alias-collapse cleanup
still needs a final hygiene pass in emitted component maps.

Scope:

- Tighten naming for reusable responses, request bodies, parameters, and
  examples where a stable public name can be inferred safely.
- Add explicit naming controls where inference is not safe enough.
- Prune nil or dead component placeholders that survive alias collapse or IR to
  v3 adaptation.

Out of scope:

- Banning hash suffixes entirely. Hash fallback remains the collision escape
  hatch when no stable public name can be claimed safely.
- Re-opening already solved explicit schema naming for `Body(...)`.

Acceptance:

- No nil-valued keys appear in emitted `components.*` maps.
- Operation-derived component names stop leaking into shared public responses
  when a generic or explicit public name is available.
- Hash suffixes remain only as a collision fallback, not as the default naming
  strategy for reusable public components.

### Unit 3: Auth Error Canonicalization

Status: planned

Problem:
`AuthErrorResponses()` still injects framework-owned 401/403 descriptions and
response variants instead of consistently reusing canonical named auth errors
already modeled in the design.

Scope:

- Make `AuthErrorResponses()` reuse existing compatible named auth errors when
  status, media type, and payload shape line up.
- Provide an explicit mapping surface if the framework cannot infer the intended
  canonical auth errors safely.

Out of scope:

- Full problem-document migration.
- App-specific auth semantics beyond standard 401/403 wiring.

Acceptance:

- Standard 401 and 403 auth responses reuse canonical named components when the
  design already has them.
- Generated auth responses stop diverging only because of framework-owned
  description text.
- Seam tests cover method-, service-, and API-scoped auth error reuse.

### Unit 4: Problem-Document Error Contracts

Status: planned

Problem:
The framework still defaults to `application/vnd.goa.error`, which is not the
best machine-facing contract for SDKs and automation.

Scope:

- Add first-class support for `application/problem+json` or a close
  RFC 9457-compatible profile.
- Preserve stable machine-readable error codes in the generated contract.
- Reuse shared error components and responses where possible.

Out of scope:

- A forced breaking migration for all consumers in one step.
- Per-application custom error vocabularies outside the framework-owned model.

Acceptance:

- The framework can publish standards-first typed errors without handwritten
  OpenAPI patches.
- Reusable problem schemas and responses are generated and validated.
- Transport and runtime expectations remain coherent with generated contracts.

### Unit 5: Projection Controls For Public Request/Response Surfaces

Status: planned

Problem:
The framework still lacks a complete public contract story for when a stable
request body, parameter component, or split request/response schema should be
published explicitly rather than inferred from repeated inline shapes.

Scope:

- Add explicit component naming controls for request bodies and parameters where
  a public identity exists but automatic hoisting is not sufficient.
- Add automatic request/response schema splitting on top of the existing
  metadata pass when the same domain type would otherwise leak secrets or
  server-managed fields both ways.

Out of scope:

- Replacing normal DSL modeling with contract-only DTO sprawl.
- Re-opening already completed `readOnly` / `writeOnly` metadata support.

Acceptance:

- The framework can publish stable request-body and parameter components without
  app-specific manual OpenAPI patching.
- Request/response schema splitting removes real secret/computed-field glue in
  consuming apps.
- Tests cover both explicit metadata-driven naming and automatic split-schema
  behavior.

### Unit 6: Async Contract Publication

Status: planned

Problem:
OpenAPI can describe the HTTP handshake for SSE and WebSocket endpoints, but it
still lacks a truthful first-class contract for stream message envelopes.

Scope:

- Keep handshake-level HTTP/OpenAPI behavior accurate.
- Publish a framework-owned async contract artifact or documented extension for
  SSE and WebSocket message shapes.
- Give downstream consumers a stable contract surface for real-time APIs.

Out of scope:

- Pretending upgraded connections are normal JSON request/response exchanges.
- Handwritten per-app async artifacts as the primary framework story.

Acceptance:

- SSE and WebSocket endpoints can publish message-envelope contracts in a
  framework-owned way.
- The framework no longer stops at handshake-only documentation for streaming
  APIs.
- Tests cover both handshake correctness and async artifact generation.

### Unit 7: OpenAPI Links DSL

Status: planned

Problem:
The OpenAPI model and renderer can represent links, but the DSL still has no
ergonomic first-class way to publish them.

Scope:

- Add a DSL surface for response links.
- Emit those links in generated OpenAPI responses and reusable components.
- Cover the common workflow cases that remove repeated app-local OpenAPI glue.

Out of scope:

- General hypermedia runtime behavior.
- Async-contract publication for stream messages.

Acceptance:

- API authors can declare response links without handwritten OpenAPI patches.
- Rendered OpenAPI publishes those links in a stable and validated shape.
- The DSL is narrow and workflow-focused rather than a generic metadata dump.

### Unit 8: Contract Linting And Consumer Smoke Tests

Status: planned

Problem:
The framework still relies too heavily on human review to catch high-signal
OpenAPI regressions that should be enforced automatically.

Scope:

- Add contract linting for high-value generator regressions.
- Add downstream smoke generation against at least one TypeScript target and one
  Go target.
- Keep the specimen matrix broad enough to exercise completed and planned
  contract behavior.

Out of scope:

- Replacing focused seam tests with only end-to-end generation tests.
- Building a full external SDK toolchain inside the framework repo.

Acceptance:

- The test suite fails on unsafe `operationId`, tag drift, reusable-component
  regressions, obvious secret-field annotation misses, and invalid async
  handshake modeling.
- At least one TypeScript and one Go downstream consumer smoke generation path
  stay green.

## Execution Order

Land the remaining OpenAPI work in this order unless a concrete downstream bug
forces reprioritization:

1. Unit 1: Payload-Bearing Response Reuse
2. Unit 2: Public Component Naming And Adapter Hygiene
3. Unit 3: Auth Error Canonicalization
4. Unit 5: Projection Controls For Public Request/Response Surfaces
5. Unit 4: Problem-Document Error Contracts
6. Unit 7: OpenAPI Links DSL
7. Unit 6: Async Contract Publication
8. Unit 8: Contract Linting And Consumer Smoke Tests

## Downstream Acceptance Surface

Use [Auto-K OpenAPI Contract Checklist](./autok_openapi_contract_checklist.md)
as the main downstream acceptance surface for these units.
