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

### 4. Make Remediation a First-Class Contract Concept

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

### Goa-AI to Goa-Core Boundary Cleanup

The current `goa-ai` fork still contains a few things that are better understood as missing Goa-core capabilities rather than AI-specific features.

The direction here is:

- move generic API-contract and transport semantics into `goa-light`
- keep MCP, agent, planner, registry, and tool-runtime concerns in `goa-ai`
- remove application-specific leaks from generic framework generators

Candidates to move into `goa-light`:

- generic union discriminator and example/schema semantics
- generic OpenAPI contract stability rules
- generic remediation-aware error modeling
- generic auth/session transport modeling where it benefits ordinary APIs too

Detailed migration plan:

1. Union discriminator and schema semantics

- Move any remaining generic union wire-tag behavior out of `goa-ai/codegen/shared`.
- The canonical source of truth for variant wire tags, discriminator values, and
  union example normalization should live in `goa-light`.
- `goa-ai` should only add MCP-specific wrapping on top of Goa’s generic union model.
- Goal: no MCP generator should need its own fallback copy of generic union-tag logic.

2. JSON Schema example canonicalization

- Move generic example canonicalization for unions and nested Goa types into
  `goa-light` where it can benefit HTTP, JSON-RPC, OpenAPI, and any future generators.
- Keep only MCP-specific final shaping in `goa-ai` when the protocol requires a
  different outer envelope.
- Goal: example correctness is a Goa property, not an MCP plugin property.

3. OpenAPI contract stability policy

- Keep stable `operationId`, schema naming, response modeling, and other
  machine-consumable contract choices in `goa-light`.
- Do not let `goa-ai` carry its own naming or contract-stability conventions.
- Goal: there is one contract policy for generated APIs, not separate policies for AI and non-AI generators.

4. Generic auth/session transport modeling

- Keep bearer-or-cookie session auth, cookie transport binding, standard auth
  responses, and related OpenAPI/security emission in `goa-light`.
- `goa-ai` should consume those contracts instead of inventing its own auth
  transport shortcuts.
- Goal: MCP and ordinary APIs share one security model wherever the transport semantics are the same.

5. Generic remediation-aware error modeling

- Add first-class remediation/error contract primitives in `goa-light`.
- Support stable error code, safe message, retryability, hint, and optional
  structured fields as design-level concepts.
- Generate these consistently across HTTP, JSON-RPC, OpenAPI, and any transport
  that Goa owns directly.
- Goal: tool-style actionable failures come from the Goa design model, not from
  app-specific template hooks.

6. JSON-RPC contract semantics that are not MCP-specific

- Audit `goa-ai` for any JSON-RPC generator behavior that is generic transport
  semantics rather than MCP behavior.
- Move anything that improves plain JSON-RPC correctness or stability into
  `goa-light/jsonrpc`.
- Goal: `goa-ai` should rely on Goa’s JSON-RPC transport, not patch around it.

7. Generic transport/runtime metadata hooks

- If `goa-ai` needs structured method metadata beyond raw `Meta(...)` for things
  that are not MCP-specific, add proper extension points to `goa-light` instead
  of duplicating metadata interpretation in plugin code.
- Goal: generic framework hooks live in Goa; protocol-specific metadata stays in plugins.

8. Error projection into OpenAPI and JSON Schema

- Once remediation-aware errors exist in `goa-light`, ensure OpenAPI 3.1 output
  exposes them in a machine-usable way with stable schemas and examples.
- `goa-ai` should then map tool/MCP failure behavior onto the same underlying
  contract instead of bypassing it.
- Goal: one error contract across normal APIs and tool-style APIs.

Candidates to keep in `goa-ai`:

- MCP DSL and code generation
- agent and toolset DSL/runtime behavior
- planner/runtime structured tool execution features
- MCP-specific annotations and protocol-specific metadata

Candidates to push out of the frameworks:

- application-specific special-casing such as direct framework dependence on a single app-owned error package

Sequencing:

1. Audit `goa-ai` for remaining generic union/schema/example workarounds.
2. Move generic remediation-aware errors into `goa-light`.
3. Audit `goa-ai` for generic JSON-RPC transport patches and move them down.
4. Remove direct app-specific framework coupling once the generic Goa-core hooks exist.
5. Leave MCP annotations, tool runtime, planners, and registries in `goa-ai`.

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
