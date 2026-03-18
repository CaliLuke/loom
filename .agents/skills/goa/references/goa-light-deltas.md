# Goa-Light Deltas

This reference captures the current `goa-light` changes that differ from older Goa expectations and the newer DSL helpers that matter when editing `design/*.go` or reviewing generated output.

Use this file before changing designs that depend on OpenAPI contracts, session auth, cookies, unions, JSON-RPC, or structured error metadata.

## Contract And Feature Changes

### OpenAPI 3.1 is the only OpenAPI output

- `goa-light` emits OpenAPI 3.1 / JSON Schema 2020-12 as the canonical contract.
- Generated files are `gen/http/openapi.json` and `gen/http/openapi.yaml`.
- Do not expect Swagger 2.0 output or separate `openapi3.{json,yaml}` artifacts.
- If an application or toolchain still expects the old files or Swagger 2.0 shape, treat that as an application migration task, not a generator bug.

### OpenAPI stability work is part of the framework contract

- Schema naming is intentionally more stable.
- `operationId` generation is canonical by default and can be customized with `Meta("openapi:operationId", ...)`.
- The default template is `{service}.{method}(.{routeIndex})`.
- `libopenapi`-backed validation is part of the repo’s OpenAPI test harness.

### Unions now support constructor-style `OneOf(...)`

- `OneOf(...)` works both as a named union declaration and as a type constructor.
- Examples:

```go
var Filter = OneOf(TextFilter, JSONFilter)

var _ = Service("search", func() {
	Method("query", func() {
		Payload(func() {
			Attribute("filter", OneOf(TextFilter, JSONFilter))
		})
	})
})
```

- This constructor form was added specifically to remove real compatibility friction in consumer designs.
- When checking generated contracts, pay attention to explicit discriminator tag behavior as well as type shape.

### Session auth is now a first-class multi-transport DSL concept

- The main helpers are:
  - `SessionAuth(name, fn)`
  - `BearerTransport(scheme, fieldName, fn...)`
  - `CookieTransport(scheme, fieldName, fn...)`
  - `CookieName(name)`
  - `SessionSecurity(contract)`
- These helpers are meant to replace repeated bearer-or-cookie glue in application designs.
- `SessionSecurity(...)` attaches the named contract at API, service, or method scope.
- Payload/session transport fields may be inferred and injected from the contract, so avoid recreating the same transport fields manually unless the design genuinely needs something custom.
- Cookie transport definitions can infer HTTP cookie binding, with `CookieName(...)` available when the wire cookie name must differ from the field name.

### HTTP auth errors have a standard helper

- `AuthErrorResponses()` adds the standard unauthorized and forbidden HTTP responses for secured endpoints.
- Use it in API, service, or method `HTTP(...)` blocks instead of duplicating the same 401/403 mappings repeatedly.

### Response cookies are modeled directly in the DSL

- Prefer Goa cookie modeling over raw `Set-Cookie` header bags.
- `SessionCookie(name, args...)` is the secure-default helper for common session issuance.
- `SessionCookie(...)` behaves like `Cookie(...)` and then applies:
  - `CookiePath("/")`
  - `CookieSecure()`
  - `CookieHTTPOnly()`
  - `CookieSameSite(CookieSameSiteLax)`
- Explicit cookie setters after `SessionCookie(...)` override those defaults.
- The response-side cookie model is per-cookie, which is cleaner than the older response-wide metadata approach.

### Errors can carry structured remediation metadata

- Errors now support structured remedy metadata in the DSL.
- The relevant helpers are:
  - `Remedy(fn)`
  - `RemedyCode(code)`
  - `SafeMessage(message)`
  - `RetryHint(hint)`
- This metadata projects into generated service errors and transport errors.
- Use it when the contract needs machine-consumable guidance, not just a free-form message.

Example:

```go
Error("bad_request", func() {
	Remedy(func() {
		RemedyCode("bad_request.fix")
		SafeMessage("Retry with a valid request.")
		RetryHint("Correct the payload and retry.")
	})
})
```

### JSON-RPC is an active first-class transport in this repo

- The repo owns JSON-RPC 2.0 support, including HTTP, SSE, and WebSocket transport behavior.
- JSON-RPC behavior and generated types are part of the framework surface, not incidental examples.
- When debugging JSON-RPC generation, compare the design, generated JSON-RPC code, and the JSON-RPC package docs/tests instead of assuming parity with HTTP or gRPC behavior.

## Practical Guidance

- If a design currently hand-models bearer-or-cookie auth, duplicated auth responses, or raw `Set-Cookie` headers, check whether the newer session and cookie DSL can replace that glue first.
- If a consumer compares OpenAPI outputs, verify it reads the canonical OpenAPI 3.1 artifacts before changing framework code.
- If a union-related change looks broken, inspect both `OneOf(...)` usage and explicit discriminator metadata before changing codegen.
- If the task touches generated transport errors, check whether remediation metadata is supposed to flow through the contract before adding ad hoc fields.

## Pointers Back Into The Repo

- High-level status: `roadmap/ROADMAP.md`
- Session and cookie direction: `roadmap/auth_and_session.md`
- OpenAPI contract direction: `roadmap/openapi_contract.md`
- Current comparison notes against the previous pinned stack: `AUTOK_BACKPORT_REPORT.md`
