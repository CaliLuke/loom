# Completed Foundation

These changes are already in `goa-light` and define the current baseline.

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
