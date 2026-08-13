---
name: loom
description: Use Loom from a consuming Go service. Covers authoring the design DSL, running loom gen, implementing services outside gen, and adopting generated HTTP, gRPC, JSON-RPC, OpenAPI, auth, streaming, and observability contracts. Do not use this skill to modify the Loom framework repository itself.
---

# Use Loom

Use this skill when an application consumes `github.com/CaliLuke/loom`: writing
or changing `design/*.go`, regenerating `gen/`, implementing service methods,
or wiring generated transports and runtime packages.

This is deliberately not a framework-maintenance guide. If the task changes
Loom's DSL implementation, expression model, generators, transports, OpenAPI
renderer, or framework tests, use the `loom-framework` skill.

## Core Workflow

1. Treat `design/*.go` as the source of truth.
2. Put validation, transport mappings, security, errors, and public contract
   metadata in the design.
3. Run `loom gen <module-import-path>/design` after every design change.
4. Implement business logic outside `gen/`.
5. Run `loom example <module-import-path>/design` only when scaffolding missing
   starter files; it does not overwrite existing `cmd/` files.
6. Run the consuming repository's formatting, tests, and integration checks.

Never edit `gen/` directly. `loom gen` deletes and recreates it transactionally,
so manual changes are both temporary and misleading.

Use Go import paths, not filesystem paths:

```bash
loom gen example.com/myapi/design
```

## Design Rules

- Prefer concrete types over `Any`, especially when gRPC generation matters.
- Put lengths, enums, formats, requiredness, and other validation in the DSL;
  do not duplicate it in service implementations.
- Do not rely on nil versus empty slices or maps to encode presence. Generated
  JSON uses `omitempty`, so both serialize as missing.
- For each non-`Extend` type, payload, or result, start literal field tags at
  `1` and increment within that definition. For definitions using `Extend`,
  start newly introduced fields at `100`.
- Prefer a canonical `ResultType` with `View(...)` definitions over parallel
  hand-maintained DTOs for alternate public representations.
- Repeated `HTTP`, `GRPC`, or `JSONRPC` blocks in the same API, service, or
  method scope compose in declaration order. Use this to keep transport
  mappings near the errors or methods they describe; ordinary duplicate and
  conflict rules still apply to the combined contents.

## OpenAPI Contracts

Loom emits OpenAPI 3.2.0 by default at:

- `gen/http/openapi.json`
- `gen/http/openapi.yaml`

Set API metadata `Meta("openapi:version", "3.1")` only when a downstream
consumer still requires OpenAPI 3.1.1. The output paths remain the same and the
compatible surrounding contract is preserved.

OpenAPI 3.2 capabilities available from the DSL include:

- tag summaries, hierarchy, and kind
- QUERY and extension HTTP methods
- whole-query-string parameters
- `itemSchema` for sequential and streaming media
- reusable media types and nested encodings
- structured examples
- device authorization OAuth flows and OAuth metadata
- URI security schemes
- response summaries with optional descriptions
- XML node types
- optional discriminators and default mappings
- server names and `$self` document identity
- `allowReserved` for parameters and headers, plus cookie style

Use the metadata table and examples in `docs/dsl-reference.md` instead of
patching generated OpenAPI.

Other important OpenAPI usage rules:

- Use `Meta("openapi:typename", "...")` when a public schema component needs a
  stable explicit name.
- Use `Meta("openapi:component:requestBody", "...")`,
  `Meta("openapi:component:parameter", "...")`,
  `Meta("openapi:component:response", "...")`, and
  `Meta("openapi:component:example", "...")` for stable reusable component
  names.
- Model workflow links with `Link(...)`, `LinkOperation(...)`,
  `LinkOperationRef(...)`, `LinkParam(...)`, and `LinkRequestBody(...)`.
- Use `Meta("openapi:readOnly", ...)` and
  `Meta("openapi:writeOnly", ...)` so request and response schemas split
  correctly when one domain type serves both directions.
- Use API metadata `Meta("openapi:closed-objects", "true")` when consumers need
  strict object contracts.
- Unreferenced component schemas are intentionally omitted.

## Errors and Remediation

Loom's default HTTP errors are RFC 9457-style
`application/problem+json` documents with a stable `code` field.

- Use `ProblemResult` when explicitly modeling the same public document shape.
- Use `ProblemType(...)` and `ProblemTitle(...)` for public error overrides.
- Use `AuthErrorResponses()` for standard 401/403 mappings.
- Use `Remedy(...)`, `RemedyCode(...)`, `SafeMessage(...)`, and
  `RetryHint(...)` for structured remediation metadata.

Do not duplicate these contracts in handwritten transport code.

## Unions, Views, and Projections

- `OneOf(...)` works as both a named union declaration and a type constructor.
- Explicit discriminator tags control wire values independently of schema and
  Go type names.
- Optional object unions generate as pointers; required unions remain values.
- Missing and explicit JSON `null` are both rejected for required unions.
- Result views inherit canonical requiredness. Use `ViewRequired(...)` and
  `ViewOptional(...)` for deliberate overrides.
- Generated projection helpers convert between canonical results and view
  types. Use them instead of maintaining app-local conversion copies.
- For typed SSE projections, use `SSEProjection(eventType, view)` with
  `SSEEventType(...)`.

## HTTP Bodies and Parameters

- Use `FormRequest()` for typed `application/x-www-form-urlencoded` payloads.
- Use `MultipartRequest()` for supported multipart object payloads.
- Use `OptionalRequestBody()` for optional JSON object bodies.
- Use `OpenAPIRequestBody(...)` with `SkipRequestBodyEncodeDecode()` when a raw
  request stream needs a documentation-only OpenAPI contract.
- String-backed path, query, header, and cookie fields with
  `Meta("struct:field:type", ...)` decode through `encoding.TextUnmarshaler`.
- Let generated clients and routers handle path escaping exactly once; do not
  add app-local `url.PathEscape` or `url.PathUnescape` layers.
- Prefer modeled response cookies over raw `Set-Cookie` header bags.

If a body shape is unsupported, use the documented custom encoder/decoder seam
rather than modifying generated files.

## Authentication and Sessions

Prefer Loom's first-class session DSL:

- `SessionAuth(name, fn)`
- `BearerTransport(scheme, fieldName, fn...)`
- `CookieTransport(scheme, fieldName, fn...)`
- `CookieName(name)`
- `SessionSecurity(contract)`
- `SessionCookie(...)`
- `CookieInsecure()` for plain-HTTP local development only

`CookieTransport(scheme, "", fn...)` is the transport-owned browser-cookie
mode. It emits the security contract without synthesizing a payload field or
CLI flag, allowing the application to resolve the cookie from request metadata.

`SessionCookie(...)` remains secure by default. A response may call
`CookieInsecure()` immediately afterward for plain-HTTP local development, but
must not do so in production or for `__Host-`/`__Secure-` names or
`SameSite=None` cookies.

Alternative security requirements are isolated. Context returned by a failed
alternative does not leak into the next one; schemes within one successful
requirement retain AND semantics.

## Streaming

- SSE endpoints use normal HTTP success responses with
  `text/event-stream`.
- Generated HTTP and JSON-RPC SSE streams expose `loomhttp.SSEControl`.
  Use `Open(ctx)` for explicit readiness and `SendComment(ctx, text)` for
  heartbeat frames.
- Configure bounded stream writes with `loomhttp.NewStreamWritePolicy`.
- Read `Last-Event-ID` through `loomhttp.LastEventIDKey`.
- Do not recover or write the raw response writer to work around streaming
  behavior.
- Generated WebSocket streams use `loomhttp.WebSocketStream`; use the generated
  interface rather than adding parallel socket lifecycle code.

Loom also emits the framework-owned `x-loom-async` OpenAPI extension for richer
SSE and WebSocket handshake/message contracts.

## JSON-RPC

JSON-RPC is a first-class transport, not an HTTP behavior alias.

- Omitted `params` decode as `{}`; ordinary required-field validation still
  applies.
- SSE notifications, final responses, and protocol errors use the generated
  stream contract.
- Set intermediate notification names with
  `SSENotificationMethod(...)` when the default namespaced method is unsuitable.
- A raw `GET /rpc` events listener is ID-less and suppresses final responses;
  send every value that must reach that listener with `Send`.
- For mixed HTTP/SSE services, only designed SSE methods route to streams even
  if the client advertises `Accept: text/event-stream`.
- Mount and wrap the generated public `ServeHTTP` handler so middleware and
  transport policy are preserved.

## CORS and Request Metadata

- Model browser access with `CORS` in the HTTP or JSON-RPC design.
- Use `RuntimeCORS()` when origins come from deployment configuration, then
  pass a validated `loomhttp.RuntimeCORSPolicy` snapshot to the generated
  constructor.
- Apply `loomhttp.RequestMetadataMiddleware` through the generated server's
  `Use` method and read `loomhttp.RequestMetadataFromContext`.
- Configure retained headers and trusted proxies with
  `NewRequestMetadataPolicy`. Sensitive headers require explicit opt-in.
- Use `loomhttp.EffectiveClientAddress` instead of interpreting forwarding
  headers in application code.

## Observability and Debugging

Prefer framework packages over repeated bootstrap glue:

- `github.com/CaliLuke/loom/observability/otel`
- `github.com/CaliLuke/loom/http/middleware/otel`
- `github.com/CaliLuke/loom/grpc/middleware/otel`
- `github.com/CaliLuke/loom/observability/transport`

For HTTP clients, wrap `*http.Client` with `otel.WrapHTTPClient(...)`. For HTTP
servers, use `loomhttp.NewMuxer()` with `otel.HTTPMiddleware(...)`. For gRPC,
use `otel.GRPCServerOption(...)` and `otel.GRPCClientOption(...)`.

`observability/transport.Event.Reason` is a stable, low-cardinality value for
metrics and routing. Handle these values rather than parsing error messages:

- `ok`
- `request_decode_failed`
- `invalid_jsonrpc_envelope`
- `invalid_jsonrpc_batch`
- `invalid_jsonrpc_method`
- `invalid_jsonrpc_params`
- `unsupported_method`
- `missing_credentials`
- `invalid_credentials`
- `permission_rejected`
- `principal_mismatch`
- `handler_error`
- `panic`
- `response_write_failed`
- `stream_write_failed`
- `stream_flush_failed`
- `stream_write_timeout`
- `stream_flush_timeout`
- `stream_final_response_suppressed`
- `mcp_session_missing`
- `mcp_session_not_found`
- `mcp_session_principal_mismatch`
- `mcp_events_stream_write_failed`

Use `loomhttp.NewDebugDoer` only for bounded, redacted development diagnostics.
Set `DEBUG_LOOM=1` while generating when you need DSL/codegen decision traces.

## Installation and Commands

```bash
go install github.com/CaliLuke/loom/cmd/loom@latest
loom version
loom gen <module-import-path>/design
loom example <module-import-path>/design
```

## Canonical Guides

- `docs/quickstart.md`
- `docs/dsl-reference.md`
- `docs/code-generation.md`
- `docs/http-guide.md`
- `docs/grpc-guide.md`
- `docs/error-handling.md`
- `docs/interceptors.md`
- `docs/production.md`
- `jsonrpc/README.md`

Open the guide closest to the task before searching framework source. If using
Loom correctly still leaves repeated application glue, report the boundary and
route a separate framework task through `loom-framework`.
