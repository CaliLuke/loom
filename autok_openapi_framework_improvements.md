# Auto-K OpenAPI Framework Improvements

## Status

This file is now a historical audit note, not the canonical framework plan.

The canonical backlog for remaining framework work lives in
[roadmap/openapi_contract.md](./roadmap/openapi_contract.md).
The top-level sequencing lives in
[roadmap/ROADMAP.md](./roadmap/ROADMAP.md).

## Why This Was Reframed

The original version of this document mixed three kinds of work:

- framework capabilities that were already completed
- valid future framework work
- one stale item that the framework already supports

That made it a poor source of roadmap truth.

## Work Already Completed

These items were previously called out here as remaining smells, but the
framework already owns them:

- SDK-safe `operationId` generation
- public tag alignment
- stable explicit-body/schema naming
- broad schema-level `readOnly` / `writeOnly` support
- baseline named-example hoisting support
- documentation-only `OpenAPIBody(...)` support for
  `SkipResponseBodyEncodeDecode()` responses
- truthful SSE handshake modeling with normal HTTP success status and
  `text/event-stream`

## Remaining Framework Work

The remaining valid framework work from the original audit has been folded into
the canonical self-contained units in
[roadmap/openapi_contract.md](./roadmap/openapi_contract.md):

1. Payload-bearing response reuse.
2. Public component naming and adapter hygiene.
3. Auth error canonicalization.
4. Problem-document error contracts.
5. Projection controls for public request/response surfaces.
6. Async contract publication for SSE and WebSocket.
7. OpenAPI links DSL.
8. Contract linting and downstream smoke tests.

## Stale Item Removed

The old proposal treated skip-encode/manual-response schema publication as a
missing framework feature. That is no longer correct: the framework already
supports documentation-only `OpenAPIBody(...)` declarations for those responses.
