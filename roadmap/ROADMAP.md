# Loom Roadmap

## Purpose

`loom` is not trying to preserve every historical Loom feature.
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
- OpenAPI schema deduplication now reuses structurally identical generated components while treating explicit HTTP `Body(...)` `openapi:typename` declarations as authoritative public names; conflicting non-equivalent claims now fail generation instead of leaking hash-suffixed public schemas.
- OpenAPI now emits operation-level security requirements for secured endpoints and explicit `security: []` for `NoSecurity()` operations.
- OpenAPI now hoists repeated path/query/header/cookie parameters into
  `components.parameters` with stable component names and rewrites repeated
  inline occurrences to parameter refs.
- OpenAPI now hoists repeated request bodies, headers, named examples, and
  structurally identical no-body responses into reusable components where the
  contract shape is stable enough for downstream client generators.
- OpenAPI now hoists repeated payload-bearing responses into
  `components.responses` when the response description, headers, content type,
  and referenced schema shape are equivalent even if duplicate generated schema
  refs only differ by internal alias names.
- OpenAPI now gives shared reusable request bodies and responses stable
  schema-driven or generic public component names where safe, while pruning
  nil-valued component placeholders before rendering `components.*` maps.
- `AuthErrorResponses()` now reuses compatible canonical 401/403 auth mappings
  across method, service, and API scopes instead of forcing helper-owned auth
  response descriptions when the design already defines those contract shapes.
- OpenAPI now supports explicit public names for hoisted reusable request-body,
  parameter, response, and named-example components, explicit request-body
  descriptions, and automatic request vs response schema splitting when
  `readOnly`/`writeOnly` metadata would otherwise leak server-managed or
  secret fields across both directions.
- OpenAPI and HTTP transport generation now default Loom errors to
  RFC 9457-style `application/problem+json` contracts with stable `code`,
  `instance`, and optional `retry_hint` fields, support error-local
  `ProblemType(...)` / `ProblemTitle(...)` overrides, and keep first-class
  response links, framework-owned async streaming contracts under
  `x-loom-async`, plus representative Redocly and downstream TypeScript/Go
  smoke-generation gates.
- OpenAPI operations now inherit service-level tag declarations by default, so
  operation tags line up with published top-level tag objects without
  duplicating method-level metadata.
- OpenAPI schema generation now honors attribute-level `readOnly`,
  `writeOnly`, `deprecated`, `contentEncoding`, and `contentMediaType`
  metadata in the IR-backed schema paths.
- OpenAPI now prunes unreferenced generated component schemas instead of publishing every top-level type and result type.
- OpenAPI closed-object contract mode now supports opt-in `additionalProperties: false` / `unevaluatedProperties: false` output while preserving explicit dictionary schemas.
- OpenAPI now suppresses invalid closed-object union-wrapper examples, honors field-level `Meta("openapi:example", "false")` on those wrappers, keeps SSE stream responses on normal HTTP success codes, advertises SSE responses as `text/event-stream`, and normalizes binary request examples to string form.
- OpenAPI now suppresses invalid synthesized examples for closed-object direct-union collections in response/media-type arrays instead of emitting examples that fail schema validation.
- OpenAPI now omits transport-level media-type examples for streaming responses instead of synthesizing partial SSE/WebSocket payload examples that can drift from the referenced schema.
- Generated service-package projection helpers now expose canonical result-to-view and view-to-result transforms for `ResultType` / `View` modeling.
- First-class `application/x-www-form-urlencoded` request encoding and decoding for typed and union payloads, including flat OAuth-style object-union fields.
- Explicit optional JSON request bodies via `OptionalRequestBody()`.
- Multipart object request decoding without handwritten decoder hooks, including shared validation flow when multipart bodies are combined with generated request-element decoding.
- Request-body validator parity and transform helper parity for downstream consumers.
- JSON-RPC SSE server generation now emits `message` events for streamed payloads and JSON-RPC error envelopes, while final success envelopes still use `response`; generated SSE clients and the integration harness preserve and validate those event types while remaining backward compatible with legacy/default frames.
- JSON-RPC SSE server streams now defer committing `200 OK` plus `Content-Type: text/event-stream` until the first SSE frame is actually written, so endpoint setup failures can still surface as the correct HTTP error response. The raw streamable-HTTP `GET /rpc` listener for `events/stream` remains an explicit eager-open exception so clients can observe stream establishment before the first published notification.
- Mixed JSON-RPC HTTP/SSE servers now inspect the decoded JSON-RPC method before routing `Accept: text/event-stream` requests into SSE handling, so MCP-style `initialize` calls can still return normal JSON while `events/stream` keeps the streamable HTTP behavior.
- JSON-RPC integration tests now include a persistent generated `ticktock` SSE fixture plus an external-client interoperability check using `github.com/tmaxmax/go-sse`, so generated streams are verified against a real third-party client as well as the in-repo harness.
- HTTP SSE server streams now defer committing `200 OK` plus `Content-Type: text/event-stream` until the first application event is written, and `http/integration_tests` carries a persistent generated `ticktock` fixture verified with `github.com/tmaxmax/go-sse`.
- SSE wire parsing and formatting now flow through shared helpers in `github.com/CaliLuke/loom/http` backed by `github.com/tmaxmax/go-sse`, replacing duplicated hand-rolled frame logic across generated HTTP clients, generated JSON-RPC streams, and the local JSON-RPC SSE harness.
- First-class OpenTelemetry transport wrappers now live in `github.com/CaliLuke/loom/http/middleware/otel` and `github.com/CaliLuke/loom/grpc/middleware/otel`, using the official `otelhttp` and `otelgrpc` libraries while keeping provider/exporter bootstrap app-owned.
- OpenTelemetry V2 now adds a framework-owned observability package in `github.com/CaliLuke/loom/observability/otel` plus an optional `logrusbridge` adapter, covering provider bootstrap, HTTP metric-mode selection, request-scoped transport enrichment hooks, and OTLP log bootstrap while retaining the lower-level transport wrappers as escape hatches.
- CLI example rendering now tolerates empty-map examples instead of panicking when OpenAPI example suppression removes wrapper examples.
- Session auth DSL and derived auth/session transport behavior.
- Per-cookie response model and improved `Set-Cookie` OpenAPI output.
- Stable OpenAPI schema naming and canonical `operationId` generation.
- `http/codegen/openapi/v3` now carries a non-trivial `meal-planner` specimen that closes the loop with rendered-spec assertions plus Redocly lint for auth, forms, multipart, union wrappers, views, and SSE.
- `http/codegen/openapi/v3` now carries a small specimen matrix (`meal-planner`, `collab-streams`, `activity-feed`, `ops-socket`, `streaming-partial-examples`) to exercise form/multipart, SSE, closed-object union collections, and WebSocket-style streaming OpenAPI output.
- OpenAPI v3 request/response body and content generation now flows through the
  typed IR analyzer/renderer end to end, including the direct operation helper
  path used by local seam tests, with the duplicate legacy v3 response/body
  builder removed.
- OpenAPI parameter analysis, route-level operation metadata, and reusable
  component hoisting now also live under the typed IR layer, so the remaining
  `http/codegen/openapi/v3` package mostly renders IR-owned decisions instead
  of duplicating contract logic.
- Go-source generation now uses the shared section model
  (`codegen.Section`, `codegen.JenniferSection`, `codegen.RawSection`) instead
  of file-backed Go template assets, and non-Go template assets use neutral
  `.tmpl` names.
- Temp-module integration tests that generate code outside the repo now pin the pushed GitHub commit instead of the local working tree when materializing `github.com/CaliLuke/loom`.
- Generic helper consolidation into `loom`.

### In Progress

- none

### Next

- prove the cleaned stack against representative downstream generation in temp modules
- centralize temp-module local-source toggles across harnesses and release
  tooling so worktree switches use shared git-common-dir state instead of any
  tracked repo files
- finish the direct follow-up test backlog for refactored transport/service-data seams
- keep new generator work on the shared Go-section architecture and use typed
  Go emission for logic-heavy sections by default

## Prioritized Backlog

These items are prioritized based on two goals:

- produce a stronger OpenAPI 3.1 contract for machine reconciliation
- remove transport projection glue that is currently hand-maintained in application code

1. Generated projection parity tests and guardrails
   See [Generated Projection Parity Tests](./generated_projection_parity_tests.md).

2. Refactor follow-up test backlog
   The recent Fowler-style refactor of HTTP endpoint validation and transport/service-data assembly exposed several helper seams that need direct tests instead of relying only on broad package and golden coverage.
   Started 2026-03-18:
   Direct `http/codegen` coverage now exists for decoder return-value fallback, map-query shaping, request encoder gating, request validation gating, tagless response ordering, file-server path normalization/wildcard extraction, aliased path-param request init, result-init and error-init arg assembly, response-body generation for explicit origin/view fanout, error content-type suppression, and multipart decoder/encoder gating.

   `expr/http_endpoint.go`
   - Add validation coverage for map/array payload skip-request-body edge cases.

   `http/codegen/service_data.go`
   - Add request validation-flag coverage for optional-body origin handling.
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
- [Auth and Session](./auth_and_session.md)
- [Generated Projection Parity Tests](./generated_projection_parity_tests.md)
- [Refactor Checklist](./refactor_checklist.md)

## Definition Of Finished

This effort is finished only when all items in [Finish Checklist](./finish_checklist.md) are complete.

## Things to Avoid

- Building auth runtime behavior into `loom`.
- Adding features solely to preserve historical Loom behavior.
- Expanding the DSL without validating that it removes real application complexity.
- Replacing core DSL-to-codegen semantics with libraries.

## Decision Rule

Before starting a new framework feature, ask:

1. Does this remove real glue or real risk in application design files?
2. Is this framework semantics, rather than runtime security logic better handled by libraries?
3. Is there a concrete downstream consumer that benefits now?

If the answer to any of these is “no”, the feature should usually wait.
