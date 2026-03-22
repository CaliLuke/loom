# Auth and Session DSL

## Goal

Reduce application design-file glue while making the default security posture cleaner and safer.

## Status

### Completed

- Add multi-transport session auth DSL:
  - `SessionAuth(...)`
  - `BearerTransport(...)`
  - `CookieTransport(...)`
  - `SessionSecurity(...)`
  - design note: [Multi-Transport Session Auth](./multi_transport_session_auth.md)
- Auto-inject session auth payload fields.
- Infer HTTP cookie bindings for session auth.
- Add standard HTTP auth error response helper:
  - `AuthErrorResponses()`
- Add response-side session cookie helper with secure defaults:
  - `SessionCookie(...)`
- Harden auth and session-cookie tests, including non-happy-path coverage, gRPC, JSON-RPC, and real parser / cookie-jar round trips.
- Replace response-wide cookie metadata with a per-cookie response model.
- Simplify the representative `meal-planner` consumer specimen by lifting
  `SessionSecurity(...)` to service scope while preserving explicit
  `NoSecurity()` overrides on public endpoints.

### Next

### 1. Prove the New DSL Against Downstream Generation

The in-repo representative consumer specimen now uses inherited
`SessionSecurity(...)`, but the remaining proof is external:

- point a representative downstream repo or temp-module generation run at
  `goa-light`
- measure which duplicated payload types and explicit auth transport bindings
  disappear
- confirm the cleanup survives real regeneration across HTTP, JSON-RPC, gRPC,
  and OpenAPI

### 2. Clean Up Auth DSL Ergonomics Further

If real app usage still shows friction, prioritize:

- better composition of auth-bound payload fragments
- cleaner modeling of bearer-or-cookie session auth at API/service scope
- sharper defaulting around standard auth responses

This should be driven by actual design churn in downstream consumers, not by abstract feature expansion.

### 3. Make Remediation a First-Class Contract Concept

Current direction:

- keep design definitions as the driver of failure behavior
- avoid hard-wiring application-specific error packages directly into core templates

What this should become:

- a first-class DSL concept for remediation-aware errors
- generated consistently across HTTP, JSON-RPC, OpenAPI, and MCP/tool surfaces
- reusable outside AI/MCP-specific transports

Desired contract fields:

- stable error code
- user-safe message
- retryability
- hint / recommended next action
- optional structured fields for downstream consumers

Architectural rule:

- `goa-light` should own the generic remediation/error contract
- concrete runtime libraries such as `remedy` may remain the preferred implementation target, but should not be the root abstraction in framework code

## Backlog

### Per-Cookie Attribute Modeling

Status:

- completed for response cookies

Remaining follow-up:

- confirm whether request-side cookie modeling needs the same cleanup
- decide whether richer cookie issuance helpers should build directly on the new model

Why this matters:

- `SessionCookie(...)` is currently a strong convenience helper for the dominant single-session-cookie case
- but it is not yet a fully general primitive when a response sets multiple cookies with distinct policies

Why this is worth doing eventually:

- it is the cleaner long-term architecture
- it makes `SessionCookie(...)` a real primitive rather than a convenience wrapper
- it enables precise multi-cookie composition without ambiguous shared metadata

Why it does not need to happen immediately:

- no current downstream consumer appears to need multiple response cookies with distinct policies
- the existing model is sufficient for the common hardened session-cookie case

Trigger to prioritize:

- as soon as a real app flow needs multiple response cookies with different attributes in one response

### Cookie Issuance / Clearing Helpers

Potential follow-up:

- helpers for common session-cookie issuance and clearing patterns
- explicit “expire this session cookie” and “issue this session cookie” DSL conveniences

Only do this after confirming the current `SessionCookie(...)` helper materially reduces glue and after deciding whether per-cookie metadata needs to come first.
