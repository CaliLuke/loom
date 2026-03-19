# Goa Light Roadmap

## Purpose

`goa-light` is not trying to preserve every historical Goa feature.
The value proposition is:

- a smaller framework surface
- cleaner OpenAPI 3.x output
- less application-side glue in design files
- safer defaults for common auth and session patterns
- reduced maintenance by outsourcing commodity protocol correctness to libraries where appropriate

This roadmap is meant to keep work focused on those outcomes instead of accumulating disconnected compatibility patches.

## Status

### Completed

- OpenAPI v2 removal and OpenAPI 3.1 / JSON Schema 2020-12 baseline.
- `libopenapi`-backed spec validation in the test harness.
- `OneOf(...)` constructor support and explicit union discriminator tag preservation.
- OpenAPI wrapper unions now emit `oneOf` branch-envelope refs with discriminator mappings.
- OpenAPI schema deduplication now reuses structurally identical generated components while preserving explicit `openapi:typename` names and real collisions.
- OpenAPI now emits operation-level security requirements for secured endpoints and explicit `security: []` for `NoSecurity()` operations.
- OpenAPI now prunes unreferenced generated component schemas instead of publishing every top-level type and result type.
- OpenAPI closed-object contract mode now supports opt-in `additionalProperties: false` / `unevaluatedProperties: false` output while preserving explicit dictionary schemas.
- OpenAPI now suppresses invalid closed-object union-wrapper examples, honors field-level `Meta("openapi:example", "false")` on those wrappers, keeps SSE stream responses on normal HTTP success codes, and normalizes binary request examples to string form.
- OpenAPI now suppresses invalid synthesized examples for closed-object direct-union collections in response/media-type arrays instead of emitting examples that fail schema validation.
- OpenAPI now omits transport-level media-type examples for streaming responses instead of synthesizing partial SSE/WebSocket payload examples that can drift from the referenced schema.
- Generated service-package projection helpers now expose canonical result-to-view and view-to-result transforms for `ResultType` / `View` modeling.
- First-class `application/x-www-form-urlencoded` request encoding and decoding for typed and union payloads, including flat OAuth-style object-union fields.
- Explicit optional JSON request bodies via `OptionalRequestBody()`.
- Multipart object request decoding without handwritten decoder hooks, including shared validation flow when multipart bodies are combined with generated request-element decoding.
- Request-body validator parity and transform helper parity needed by `auto-k-server`.
- JSON-RPC SSE server generation now emits MCP-compatible `message` events for streamed payload delivery, and generated SSE clients accept both `message`/default frames and the legacy custom event names.
- CLI example rendering now tolerates empty-map examples instead of panicking when OpenAPI example suppression removes wrapper examples.
- Session auth DSL and derived auth/session transport behavior.
  See [Multi-Transport Session Auth](./multi_transport_session_auth.md).
- Per-cookie response model and improved `Set-Cookie` OpenAPI output.
- Stable OpenAPI schema naming and canonical `operationId` generation.
- `http/codegen/openapi/v3` now carries a non-trivial `meal-planner` specimen that closes the loop with rendered-spec assertions plus Redocly lint for auth, forms, multipart, union wrappers, views, and SSE.
- `http/codegen/openapi/v3` now carries a small specimen matrix (`meal-planner`, `collab-streams`, `activity-feed`, `ops-socket`, `streaming-partial-examples`) to exercise form/multipart, SSE, closed-object union collections, and WebSocket-style streaming OpenAPI output.
- Temp-module integration tests that generate code outside the repo now pin the pushed GitHub commit instead of the local working tree when materializing `goa.design/goa/v3`.
- Major generic helper moves out of `goa-ai/shared` into `goa-light`.

### In Progress

- none

### Next

- rewire `goa-ai` to consume the core helpers already moved into `goa-light`
- prove the cleaned stack against `auto-k-server` in temp generation
- replace real auth/session glue in `auto-k-server` and then perform the swap

## Prioritized Backlog

These items are prioritized based on two goals:

- produce a stronger OpenAPI 3.1 contract for machine reconciliation
- remove transport projection glue that is currently hand-maintained in application code

1. Generated projection parity tests and guardrails
   See [Generated Projection Parity Tests](./generated_projection_parity_tests.md).

2. Refactor follow-up test backlog
   The recent Fowler-style refactor of HTTP endpoint validation and transport/service-data assembly exposed several helper seams that need direct tests instead of relying only on broad package and golden coverage.
   Started 2026-03-18:
   Direct `http/codegen` coverage now exists for decoder return-value fallback, map-query shaping, request encoder gating, request validation gating, tagless response ordering, file-server path normalization/wildcard extraction, result-init and error-init arg assembly, error content-type suppression, and multipart decoder/encoder gating.

   `expr/http_endpoint.go`
   - Add prepare tests for implicit default success responses: redirect status and no-content status.
   - Add prepare tests for SSE inheritance from service and API scopes.
   - Add prepare tests for inherited HTTP errors from service and API scopes.
   - Add prepare test for WebSocket route method coercion to `GET`.
   - Add validation tests for all-tagged responses rejection and tagged responses requiring object results.
   - Add validation tests for SSE on non-streaming endpoints and JSON-RPC result/request `id` consistency.
   - Add validation tests for redirect plus mismatched response status, map/array payload skip-request-body cases, and JSON-RPC payload migration in `Finalize`.
   - Add finalize test for implicit session cookie mapping.

   `http/codegen/service_data.go`
   - Add direct tests for file-server path normalization and directory wildcard extraction.
   - Add direct tests for request encoder emission and omission.
   - Add request-init tests for aliased path params using service type refs.
   - Add request-shape tests for whole-payload and named-field `MapParams(...)`.
   - Add request validation-flag tests covering cookies, headers, query params, path params, and optional-body origin handling.
   - Add payload decoder return-value precedence tests for params, headers, cookies, and whole-payload map query.
   - Add JSON-RPC payload/result `id` projection tests.
   - Add response-body generation tests for explicit origin attributes, explicit views, and per-view fanout.
   - Add response ordering test that keeps the tagless response last.
   - Add error content-type suppression test for `expr.ErrorResultIdentifier`.
   - Add error body description rewrite tests.
   - Add multipart gating tests for decoder generation, encoder generation, and `BuildStreamPayload`.
   - Add direct union collection tests across request body, streaming body, responses, and errors.

   `codegen/service/service_data.go`
   - Add direct tests for remediation metadata on collected service errors.
   - Add tests for raw-object payload/result wrapping into synthetic user types.
   - Add tests for viewed-result deduplication by view and separation across different views.
   - Add direct union collection coverage across payload, streaming payload, result, and errors.
   - Add forced-type generation tests with and without service filters.

   Lower-priority integration/golden follow-up
   - Add representative golden coverage for viewed results, explicit response tags, explicit body origin attributes, multipart, skip request body encode/decode, skip response body encode/decode, and JSON-RPC mixed results.
   - Add one complex end-to-end generator test that combines params, headers, cookies, tagged responses, and typed error responses.

## Roadmap Index

- [Finish Checklist](./finish_checklist.md)
- [OpenAPI Contract](./openapi_contract.md)
- [OpenAPI Union Discriminators](./openapi_union_discriminators.md)
- [OpenAPI Schema Deduplication](./openapi_schema_dedup.md)
- [OpenAPI Closed Object Mode](./openapi_closed_object_mode.md)
- [Auth and Session](./auth_and_session.md)
- [Multi-Transport Session Auth](./multi_transport_session_auth.md)
- [Multipart Object Decoding](./multipart_object_decoding.md)
- [Optional JSON Bodies](./optional_json_bodies.md)
- [Form URL Encoded Decoding](./form_urlencoded_decoding.md)
- [Goa-AI Boundary](./goa_ai_boundary.md)
- [Generated Transport Projections](./generated_transport_projections.md)
- [Generated Projection Parity Tests](./generated_projection_parity_tests.md)

## Definition Of Finished

This effort is finished only when all items in [Finish Checklist](./finish_checklist.md) are complete.

## Things to Avoid

- Building auth runtime behavior into `goa-light`.
- Adding features solely to preserve historical Goa behavior.
- Expanding the DSL without validating that it removes real application complexity.
- Replacing core DSL-to-codegen semantics with libraries.

## Decision Rule

Before starting a new framework feature, ask:

1. Does this remove real glue or real risk in application design files?
2. Is this framework semantics, rather than runtime security logic better handled by libraries?
3. Is there a concrete consumer, ideally `auto-k-server`, that benefits now?

If the answer to any of these is “no”, the feature should usually wait.
