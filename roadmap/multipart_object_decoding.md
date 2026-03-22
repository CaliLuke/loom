# Multipart Object Decoding

## Goal

Generate multipart object request decoding end-to-end so applications do not need handwritten decoder hooks for typed multipart payloads.

Status: completed.

## Problem

Downstream consumers still carry custom multipart request decoding when generated server code expects an application-provided decoder seam for object-shaped multipart payloads. That keeps high-value ingestion paths outside the framework contract.

## Scope

Framework work only:

- HTTP server request decoding for multipart object payloads
- generated decoder behavior for typed multipart fields
- regression coverage for generated multipart decoder paths

Out of scope:

- application-specific file validation or storage logic
- custom upload policies

## Desired Outcome

- typed multipart object payloads decode without handwritten transport hooks
- generated server code handles file parts and typed non-file parts consistently
- applications can model multipart payloads in the DSL and rely on generated decoding

## Work Plan

1. Audit the current multipart request decode path and identify exactly where generation stops short for object payloads.
2. Define the supported multipart object shape:
   - file fields
   - scalar fields
   - optional fields
   - repeated fields, if already representable in DSL
3. Generate multipart decoding for those shapes directly in HTTP transport code.
4. Preserve normal Loom validation flow after decoding instead of pushing validation into custom hooks.
5. Add regression tests proving the generated decoder replaces the handwritten multipart hook used by representative consumers.

## Design Constraints

- Keep the feature in generated HTTP transport code, not in app-local helper seams.
- Reuse existing payload typing and validation instead of inventing a multipart-only modeling path.
- Be explicit about unsupported multipart shapes rather than silently falling back to custom hooks.

## Risks

- Multipart decoding gets messy when field presence, repeated parts, and files interact.
- The framework should avoid introducing a partial feature that only works for one app-specific shape.

## Finish Criteria

- A typed multipart object payload can be modeled in the DSL and decoded without a handwritten decoder hook.
- Generated server code covers the high-value object multipart shapes that previously forced handwritten hooks.
- Regression tests cover at least:
  - multipart object with file plus scalar fields
  - optional multipart fields
  - invalid multipart field values

## Completed Notes

- Supported object multipart payloads now use framework-owned HTTP server decoding.
- Generated multipart decoding reuses the request-body type, constructor, and validator flow instead of a custom transport hook.
- Generated multipart decoders now share the same validation accumulator across body validation and generated request-element validation, so multipart object endpoints continue to compile when they also include validated path/query/header/cookie-derived fields.
- Top-level `Bytes` fields are populated from multipart file parts, and a single file part can auto-populate sibling `filename` and `content_type` fields when those attributes exist on the body.
- Unsupported multipart payload shapes still use the legacy custom decoder seam instead of silently generating partial behavior.
- Regression coverage now includes multipart object decoding combined with a validated generated request element, closing the compile-time `err` redeclaration gap that showed up in real generated server code.
