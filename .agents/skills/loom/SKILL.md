---
name: loom
description: Build and maintain `loom` services in Go. Use this skill when a user mentions Loom, Loom migration, Loom DSL, `loom gen`, generated `gen/` transport code, OpenAPI/proto generation, service implementation after DSL changes, or refactoring a project with a `design` package.
---
# Loom

Use this skill when building or changing a service that uses `loom`. It is for framework users: service designers and implementers working from the Loom DSL and generated code.

Loom is a design-first framework that diverged to support AI-first development, stronger
machine-facing OpenAPI 3.1 contracts, and the framework capabilities needed to
build Auto-K without repeating large amounts of app-local glue.

## Non-Negotiables

- Treat `design/*.go` as the source of truth.
- Regenerate after every design change with `loom gen <module-import-path>/design`.
- Never hand-edit generated `gen/` files.
- Implement business logic in non-generated files.
- Use Go import paths for Loom commands, not filesystem paths.
- Commit generated code; do not rely on CI to regenerate it.

## Runtime Gotchas

- Do not "fix" SSE by hand-editing generated stream files. Keep the fix in `design/*.go` or non-generated transport/runtime code.
- Do not map multi-cookie responses through ad hoc `Header("set_cookies:Set-Cookie")` bags and then patch generated encoders. Prefer idiomatic framework cookies in the DSL when feasible. If the response shape still depends on raw cookie header values, emit them from non-generated transport code on the live `http.ResponseWriter` instead of editing generated files.

## Loom Contract Rules

- `loom` emits OpenAPI 3.1 / JSON Schema 2020-12 only. The canonical artifacts are `gen/http/openapi.json` and `gen/http/openapi.yaml`.
- Treat OpenAPI output shape as framework contract. Stable schema names, canonical `operationId`, and `libopenapi` validation are intentional behavior, not incidental formatting.
- When changing OpenAPI contract generation in `loom`, start in
  `http/codegen/openapi/internal/ir` first. That package now owns schema,
  parameter, operation-metadata, and reusable-component analysis; the
  `http/codegen/openapi/v3` package should mostly render IR-owned decisions.
- Go-source generation in framework code should be implemented in Go via the
  generic section model (`codegen.Section`, `codegen.JenniferSection`,
  `codegen.RawSection`) rather than file-backed template assets.
- Use `codegen.JenniferSection` when a section benefits from typed Go emission.
  Use `codegen.RawSection` when direct source assembly is simpler and keeps the
  logic local and explicit.
- Non-Go artifact generation may still use text templates, but those assets
  should use neutral `.tmpl` names rather than Go-specific suffixes.
- Structurally identical generated OpenAPI components are deduplicated and reused by `$ref`.
- For explicit HTTP `Body(...)` request/response types, `Meta("openapi:typename", "...")` is the public OpenAPI component name contract. When two non-equivalent explicit body schemas claim the same name, generation fails instead of leaking a hash-suffixed fallback into the spec.
- Treat that failure as a modeling conflict, not as a cue to add more aliases. It usually means one DSL type is being asked to represent both the semantic service/result shape and a transport-only projection (for example, “same object minus cookie/header fields”).
- If some fields are transport-only, keep them in HTTP headers/cookies and out of the canonical body/result type. Do not rely on OpenAPI naming to paper over service-shape vs transport-shape drift.
- Generated OpenAPI emits operation-level security for secured endpoints, including inherited service/API requirements; `NoSecurity()` emits explicit `security: []` on the operation instead of relying on omission.
- Generated OpenAPI prunes unreferenced component schemas; top-level types and result types that are not reachable from any published request/response path should not appear in `components.schemas`.
- Generated OpenAPI now also hoists repeated path, query, header, and cookie
  parameters into `components.parameters` with stable component names; repeated
  inline parameter shapes should appear as `$ref`s rather than duplicated
  objects.
- Generated OpenAPI also hoists repeated request bodies, headers, named
  examples, and structurally identical no-body responses into reusable
  components where the public contract shape is stable enough to share safely.
- Use `ProblemResult` / `ProblemResultIdentifier` when you want
  RFC 9457-style `application/problem+json` error contracts with a stable
  machine-readable `code` field instead of the legacy upstream error media type.
- Shared reusable request bodies and responses now prefer schema-derived or
  generic public component names when that contract identity can be inferred
  safely, instead of defaulting to operation-derived names; hash suffixes
  remain only as a collision fallback.
- Use `Meta("openapi:component:requestBody", "...")` when a hoisted reusable
  request body needs an explicit public component name, and
  `Meta("openapi:component:parameter", "...")` when a hoisted reusable
  path/query/header/cookie parameter needs one.
- When the same domain type is used on both request and response paths,
  `readOnly` and `writeOnly` metadata now trigger automatic request/response
  schema splitting so server-managed and secret fields do not share one public
  schema component across both directions.
- Service-level OpenAPI tags are inherited by operations and file servers when
  those operations do not declare method-level tags of their own.
- Use `Link(...)`, `LinkOperation(...)`, `LinkOperationRef(...)`,
  `LinkParam(...)`, and `LinkRequestBody(...)` on HTTP responses when a
  workflow or follow-up operation should be published as an OpenAPI response
  link instead of a handwritten patch.
- Attribute-level `Meta("openapi:readOnly", ...)`,
  `Meta("openapi:writeOnly", ...)`, `Meta("openapi:deprecated", ...)`,
  `Meta("openapi:contentEncoding", ...)`, and
  `Meta("openapi:contentMediaType", ...)` flow through to generated OpenAPI
  schemas.
- Generated OpenAPI suppresses closed-object union-wrapper examples that would be invalid against the emitted schema, and field-level `Meta("openapi:example", "false")` must suppress wrapper examples all the way through enclosing request bodies/media types.
- Generated OpenAPI also suppresses synthesized examples for closed-object union collections when the array/map element shape would otherwise emit invalid discriminator-wrapper examples.
- Generated OpenAPI does not emit transport-level media-type examples for streaming responses; SSE and WebSocket response shapes still appear via their schemas, but the generator should not synthesize single-message examples from sparse field examples.
- Wrapper-style unions now emit OpenAPI discriminators with:
  - `discriminator.propertyName`
  - `discriminator.mapping`
  - `oneOf` refs to generated `...Envelope` component schemas
- Use API metadata `Meta("openapi:closed-objects", "true")` when machine consumers need stricter object contracts in generated OpenAPI.
- In closed-object mode, normal object schemas emit `additionalProperties: false`, composed union wrappers emit `unevaluatedProperties: false`, and explicit dictionaries such as `MapOf(...)` remain open.
- Generated OpenAPI keeps SSE endpoints on ordinary HTTP success responses instead of rewriting them to WebSocket `101` semantics, and advertises those responses as `text/event-stream` rather than `application/json`.
- Generated OpenAPI also publishes framework-owned async streaming contracts
  under `x-loom-async` for SSE and WebSocket endpoints, with inline message
  schemas plus truthful handshake metadata.
- Generated OpenAPI normalizes binary (`Bytes`) examples to string form; do not expect byte-array literals in emitted OpenAPI examples.
- The OpenAPI regression gate in `http/codegen/openapi/v3` now includes
  Redocly lint plus downstream `openapi-typescript` and `oapi-codegen` smoke
  generation. Treat those tests as contract-shape enforcement, not optional
  extras.
- `OneOf(...)` works both as a named union declaration and as a type constructor.
- Explicit union discriminator tags control the wire value even when schema/type names are renamed for OpenAPI purposes.
- When modeling alternate transport/tool result shapes, prefer a canonical `ResultType` plus `View(...)` definitions over hand-maintained sibling DTO copies.
- The service generator now emits exported typed projection helpers for result views:
  - `Project<ResultType>[ViewSuffix](...)` to project a canonical result into the generated view type
  - `New<ResultType>From<ProjectedType>[ViewSuffix](...)` to rebuild the canonical result from a projected view
- Use `FormRequest()` on HTTP endpoints when the request body contract is `application/x-www-form-urlencoded`.
- `FormRequest()` is for typed object payloads and constructor unions only; incompatible body/param mixes are rejected during design validation instead of silently falling back to app-local parsing.
- Form-encoded unions keep scalar branches on the canonical wrapper shape (`type` + `value`) but flatten object branches onto normal form fields; direct top-level union form payloads do not add an extra synthetic wrapper key, and all-optional object branches may be selected by discriminator alone without synthetic `value` fields.
- `MultipartRequest()` now generates server-side decoding for supported object payloads, including common file-plus-fields uploads, instead of requiring a handwritten decoder hook.
- Generated multipart decoding is intentionally narrower than form decoding: unsupported multipart payload shapes still use the legacy custom encoder/decoder seam instead of partial magic.
- For supported multipart object payloads with a single top-level file field, sibling body attributes named `filename` and `content_type` are auto-populated from the uploaded part when those fields are present and not explicitly supplied.
- Use `OptionalRequestBody()` when an HTTP endpoint may omit a JSON request body entirely.
- `OptionalRequestBody()` is intentionally narrow: JSON only, object request bodies only, no raw body streaming, no multipart, no form bodies, and no required body-mapped payload attribute.
- OpenAPI request bodies generated from `OptionalRequestBody()` render with `required: false`.
- Session auth is first-class. Prefer the built-in DSL instead of hand-rolling bearer-or-cookie glue:
  - `SessionAuth(name, fn)`
  - `BearerTransport(scheme, fieldName, fn...)`
  - `CookieTransport(scheme, fieldName, fn...)`
  - `CookieName(name)`
  - `SessionSecurity(contract)`
- Use `AuthErrorResponses()` for standard HTTP auth failures instead of duplicating 401/403 mappings.
- `AuthErrorResponses()` now reuses compatible canonical 401/403 mappings from
  method, service, or API scope when those auth errors are already modeled,
  instead of forcing the helper-owned fallback descriptions into the contract.
- Prefer modeled response cookies over raw `Set-Cookie` header bags. `SessionCookie(...)` is the secure-default helper for common session issuance.
- Structured remediation metadata is part of the contract surface. Use:
  - `Remedy(fn)`
  - `RemedyCode(code)`
  - `SafeMessage(message)`
  - `RetryHint(hint)`
- JSON-RPC is a first-class transport in this repo. Do not assume HTTP or gRPC semantics automatically carry over.
- JSON-RPC SSE event names are part of the transport contract: streamed notifications use `message`, final success envelopes use `response`, and JSON-RPC error envelopes also ride the normal `message` channel.
- JSON-RPC SSE streams defer committing `text/event-stream` until the first frame is written. The narrow exception is the raw streamable-HTTP `GET /rpc` listener for the `events/stream` method, which must eagerly establish the SSE response so clients can observe readiness before the first domain event.
- For mixed JSON-RPC HTTP/SSE services, treat `Accept: text/event-stream` as necessary but not sufficient for SSE routing: normal methods like `initialize` must still go through the JSON response path, while only the actual SSE methods (for example `events/stream`) should route into the stream handler.
- HTTP SSE streams also defer committing `text/event-stream` until the first application event is written.
- OpenTelemetry transport instrumentation is first-class. Prefer:
  - `github.com/CaliLuke/loom/observability/otel` when you want framework-owned
    trace, metric, and OTLP log bootstrap plus transport policy.
  - `github.com/CaliLuke/loom/http/middleware/otel`
  - `github.com/CaliLuke/loom/grpc/middleware/otel`
- The root observability package is the preferred path for services that want to
  replace repeated app-local observability glue. The lower-level HTTP and gRPC
  packages remain the transport-only escape hatch.
- These packages intentionally wrap the official contrib libraries:
  - `otelhttp` for HTTP
  - `otelgrpc` for gRPC
- Keep environment parsing and domain-specific metrics in application bootstrap.
  The root package owns provider setup, transport policy, and request-scoped
  transport hooks; it does not own app-specific exporter configuration parsing.
- For HTTP servers, use `loomhttp.NewMuxer()` plus `otel.HTTPMiddleware(...)` so
  spans can use the matched `METHOD /pattern` route name from `r.Pattern`.
- For downstream HTTP middlewares that need to attach request-scoped transport
  attributes after the span starts, call `otel.AddHTTPAttributes(...)` instead
  of mutating spans directly.
- For generated HTTP clients, wrap an `*http.Client` with
  `otel.WrapHTTPClient(...)` before passing it anywhere a Loom HTTP Doer is
  expected.
- For gRPC, prefer `otel.GRPCServerOption(...)` and `otel.GRPCClientOption(...)`
  over the legacy trace/X-Ray middleware.

## Practical Checks

- If a design hand-models bearer-or-cookie auth, duplicated auth responses, or raw `Set-Cookie` headers, check whether the newer session and cookie DSL should replace that glue first.
- If a consumer compares OpenAPI outputs, verify it reads the OpenAPI 3.1 artifacts before changing framework code.
- If a union-related change looks wrong, inspect both `OneOf(...)` usage and explicit discriminator tags before changing codegen.
- If the task touches generated transport errors, confirm whether remediation metadata should flow through the contract before adding ad hoc fields.

## Default Workflow

1. Detect the Loom service surface: `go.mod`, `design/`, DSL imports, or `gen/` folders.
2. Edit the DSL in `design/`.
3. Run `loom gen <module>/design`.
4. Run `loom example <module>/design` only when scaffolding a new service or new starter files are explicitly wanted.
5. Implement logic outside `gen/`.
6. Verify with `go mod tidy` and project tests.

## Command Reminders

```bash
go install github.com/CaliLuke/loom/cmd/loom@latest
loom version
loom gen <module-import-path>/design
loom example <module-import-path>/design
```

- Correct: `loom gen example.com/myapi/design`
- Incorrect: `loom gen ./design`

## References

- Framework/source map: `references/repo-map.md`
- Use only the original full guide pages under `references/user-guides/*.md`.
- For framework/runtime internals, inspect the Loom source tree described in `references/repo-map.md`.

## Original Guide Pages

- `references/user-guides/quickstart.md`
- `references/user-guides/dsl-reference.md`
- `references/user-guides/code-generation.md`
- `references/user-guides/http-guide.md`
- `references/user-guides/grpc-guide.md`
- `references/user-guides/error-handling.md`
- `references/user-guides/interceptors.md`
- `references/user-guides/production.md`

## Selection Rules

- Start with the one full guide page that best matches the immediate task.
- For repo-specific behavior differences from upstream Loom releases, use the `Loom Contract Rules` section in this skill before inspecting the wider source tree.
- Load additional full guide pages only if the first one is insufficient.
- Prefer `references/repo-map.md` and the Loom source tree for framework internals or runtime behavior.
