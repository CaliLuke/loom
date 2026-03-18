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

## Status

### Completed

- OpenAPI v2 removal and OpenAPI 3.1 / JSON Schema 2020-12 baseline.
- `libopenapi`-backed spec validation in the test harness.
- `OneOf(...)` constructor support and explicit union discriminator tag preservation.
- Request-body validator parity and transform helper parity needed by `auto-k-server`.
- Session auth DSL and derived auth/session transport behavior.
- Per-cookie response model and improved `Set-Cookie` OpenAPI output.
- Stable OpenAPI schema naming and canonical `operationId` generation.
- Major generic helper moves out of `goa-ai/shared` into `goa-light`.

### In Progress

- none

### Next

- rewire `goa-ai` to consume the core helpers already moved into `goa-light`
- prove the cleaned stack against `auto-k-server` in temp generation
- replace real auth/session glue in `auto-k-server` and then perform the swap

## Roadmap Index

- [Finish Checklist](./finish_checklist.md)
- [OpenAPI Contract](./openapi_contract.md)
- [Auth and Session](./auth_and_session.md)
- [Goa-AI Boundary](./goa_ai_boundary.md)

## Definition Of Finished

This effort is finished only when all items in [Finish Checklist](./finish_checklist.md) are complete.

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
