# Multi-Transport Session Auth

## Goal

Model one logical authenticated session that may arrive through multiple
transports, while removing duplicated payload types and repeated HTTP glue from
application designs such as `auto-k-server`.

## Status

Implemented in `goa-light`.

The core session/auth DSL and its transport behavior are now part of the
framework surface.

## What Landed

- top-level session auth contracts with `SessionAuth(...)`
- bearer and cookie transport declarations with:
  - `BearerTransport(...)`
  - `CookieTransport(...)`
- method, service, and API usage through `SessionSecurity(...)`
- payload auto-injection for required session transport fields
- inferred HTTP transport bindings for session auth
- standard HTTP auth error responses via `AuthErrorResponses()`
- secure-default response cookies via `SessionCookie(...)`
- regression coverage across HTTP, gRPC, JSON-RPC, and cookie handling paths

## Problem This Solves

Before this work, designs often had to model one auth concept through repeated
transport-specific glue.

Typical duplication included:

- paired bearer-only and bearer-plus-cookie payload families
- repeated `Security(...)` declarations for each accepted transport
- repeated HTTP bindings such as `Header("auth:Authorization")`
- local helper DSL just to accept a browser session cookie

That shape was correct but noisy. The design expressed transport mechanics
instead of the auth contract.

## DSL Shape

The core contract is a named session auth definition:

```go
var AppSession = SessionAuth("app_session", func() {
	BearerTransport(AuthSessionToken, "auth")
	CookieTransport(BrowserSessionCookie, "browser_session")
})
```

The contract is then attached where authorization is required:

```go
Method("GetProfile", func() {
	SessionSecurity(AppSession)
	Payload(GetProfileRequest)
	HTTP(func() {
		GET("/auth/profile")
		Response(200)
	})
})
```

Semantics:

- one `SessionAuth(...)` models one logical auth contract
- transports inside the contract are alternatives, not additive requirements
- `SessionSecurity(...)` lowers to ordinary security requirements
- required transport fields are synthesized into payloads when absent
- HTTP transport binding is inferred from the declared session transports

## OpenAPI And Transport Behavior

For bearer-or-cookie auth, the generated OpenAPI security shape is the expected
OR form, with one requirement object per accepted transport.

Operation-level OpenAPI security is emitted even when the effective security is
inherited from API or service scope, and explicit public overrides now render
`security: []` instead of silently omitting the field.

The framework owns:

- DSL semantics
- generated payload and binding behavior
- OpenAPI/security emission

The application still owns actual credential verification and runtime auth
logic.

## Boundaries

This feature does not turn `goa-light` into a session-management framework.

It intentionally does not own:

- JWT, cookie, or OIDC verification libraries
- session issuance or revocation policy
- application principal/runtime identity modeling

The framework models the contract and transport semantics. Applications keep
their preferred auth runtime.

## Related Helpers

This work enabled or aligns closely with:

- `AuthErrorResponses()` for standard HTTP 401/403 responses
- `SessionCookie(...)` for secure-default session cookies
- the per-cookie response model used for accurate `Set-Cookie` emission

## Remaining Follow-Up

The core feature is done. Remaining work is consumer proof, not framework
foundation:

- prove the DSL against real `auto-k-server` design cleanup
- measure how much duplicated payload and transport glue disappears
- decide whether any additional ergonomics are justified by real consumer
  friction

## Related Roadmap Docs

- [Auth and Session](./auth_and_session.md)
- [Finish Checklist](./finish_checklist.md)
