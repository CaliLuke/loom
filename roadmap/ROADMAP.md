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

## Current Direction

The next work should continue to optimize for two things:

1. Reducing application design-file glue.
2. Making security-related defaults safer without turning `goa-light` into an auth runtime.

Framework-level modeling and code generation belong in `goa-light`.
Token verification, OIDC, session storage, cookie signing/encryption, and similar runtime security concerns should stay delegated to specialized libraries.

## Roadmap Index

- [Completed Foundation](./completed_foundation.md)
- [OpenAPI Contract](./openapi_contract.md)
- [Auth and Session DSL](./auth_and_session.md)
- [Goa-AI Boundary Cleanup](./goa_ai_boundary.md)

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
