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

## Completed

- Remove OpenAPI v2 generation and keep the framework on OpenAPI 3.x only.
- Upgrade OpenAPI generation to 3.1 / JSON Schema 2020-12.
- Use `libopenapi` in the test harness for spec parsing and validation.
- Restore `OneOf(...)` constructor support needed by `auto-k-server`.
- Preserve explicit union discriminator tags through codegen.
- Restore request-body validator generation for tagged union bodies.
- Restore pinned transform helper nil guards.
- Add multi-transport session auth DSL:
  - `SessionAuth(...)`
  - `BearerTransport(...)`
  - `CookieTransport(...)`
  - `SessionSecurity(...)`
- Auto-inject session auth payload fields.
- Infer HTTP cookie bindings for session auth.
- Add standard HTTP auth error response helper:
  - `AuthErrorResponses()`
- Add response-side session cookie helper with secure defaults:
  - `SessionCookie(...)`
- Harden auth and session-cookie tests, including real parser / cookie-jar round trips.

## Current Direction

The next work should continue to optimize for two things:

1. Reducing application design-file glue.
2. Making security-related defaults safer without turning `goa-light` into an auth runtime.

Framework-level modeling and code generation belong in `goa-light`.
Token verification, OIDC, session storage, cookie signing/encryption, and similar runtime security concerns should stay delegated to specialized libraries.

## Near-Term Priorities

### 1. Prove the New DSL Against `auto-k-server`

Use the new session/auth helpers in real `auto-k-server` designs and measure:

- which duplicated payload types disappear
- which explicit `Header(...)`, `Cookie(...)`, and `Response(...)` blocks disappear
- whether any awkward edge cases remain

This is the highest-value next step because it validates whether the new surface actually reduces glue where it matters.

### 2. Clean Up Auth DSL Ergonomics Further

If real app usage still shows friction, prioritize:

- better composition of auth-bound payload fragments
- cleaner modeling of bearer-or-cookie session auth at API/service scope
- sharper defaulting around standard auth responses

This should be driven by actual design churn in `auto-k-server`, not by abstract feature expansion.

### 3. Keep Outsourcing Commodity Validation

Continue using libraries for:

- OpenAPI parsing and validation
- spec sanity checks
- protocol-level correctness checks in tests

Avoid reintroducing bespoke parsing or validator logic where standard libraries already do the job well.

## Important Architectural Backlog

### Per-Cookie Attribute Modeling

Current limitation:

- cookie attributes such as `Path`, `Domain`, `MaxAge`, `Secure`, `HttpOnly`, and `SameSite` are modeled at the HTTP response cookie container level
- not per individual cookie

Why this matters:

- `SessionCookie(...)` is currently a strong convenience helper for the dominant single-session-cookie case
- but it is not yet a fully general primitive when a response sets multiple cookies with distinct policies

Why this is worth doing eventually:

- it is the cleaner long-term architecture
- it makes `SessionCookie(...)` a real primitive rather than a convenience wrapper
- it enables precise multi-cookie composition without ambiguous shared metadata

Why it does not need to happen immediately:

- `auto-k-server` does not currently appear to need multiple response cookies with distinct policies
- the existing model is sufficient for the common hardened session-cookie case

Trigger to prioritize:

- as soon as a real app flow needs multiple response cookies with different attributes in one response

### Cookie Issuance / Clearing Helpers

Potential follow-up:

- helpers for common session-cookie issuance and clearing patterns
- explicit “expire this session cookie” and “issue this session cookie” DSL conveniences

Only do this after confirming the current `SessionCookie(...)` helper materially reduces glue and after deciding whether per-cookie metadata needs to come first.

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
