# Auth and Session DSL

## Goal

Reduce application design-file glue while making the default security posture cleaner and safer.

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
- `goa-ai` should consume that model for tool and MCP behavior
- concrete runtime libraries such as `remedy` may remain the preferred implementation target, but should not be the root abstraction in framework code

## Architectural Backlog

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
