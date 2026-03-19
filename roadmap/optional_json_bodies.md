# Optional JSON Bodies

Status: completed on 2026-03-18

## Goal

Allow endpoints to accept either no JSON body or a typed JSON body without handwritten EOF-tolerant decoder logic.

## Problem

Some endpoints legitimately accept an empty body or a small typed JSON body. Today downstream consumers still need custom HTTP decoder handling to tolerate EOF on those endpoints instead of relying on generated transport behavior.

## Scope

Framework work only:

- HTTP JSON request decoding behavior
- DSL-to-transport semantics for optional request bodies
- generated tests around empty-body versus typed-body handling

Out of scope:

- application-specific defaulting after decode
- custom auth/session runtime logic

## Delivered Outcome

- the DSL can express an optional JSON request body clearly
- generated decoders accept an empty body when the contract allows it
- generated decoders still reject malformed JSON and invalid typed payloads

## What Changed

1. Added `OptionalRequestBody()` to the HTTP DSL so an endpoint can explicitly allow an omitted JSON request body.
2. Added endpoint validation to keep the feature narrow and predictable:
   - request body must be defined
   - request body must be JSON/object-shaped
   - body-mapped payload attributes must remain optional
   - form, multipart, and raw body streaming paths are rejected
3. Updated HTTP request decoding so EOF is accepted only for endpoints that opt into `OptionalRequestBody()`.
4. Updated OpenAPI request-body generation so optional JSON bodies render with `required: false`.
5. Added regression coverage for explicit body types, `Body("body")` origin mapping, and invalid DSL combinations.

## Design Constraints

- Do not blur the difference between “body omitted” and “body present but invalid”.
- The feature should be contract-driven, not based on app-local decoder swaps.
- Avoid broad decoder leniency that changes behavior for existing required-body endpoints.

## Remaining Follow-Up

- If a real consumer needs optional non-object JSON bodies or optional multipart/form behavior, handle that as a separate feature instead of broadening `OptionalRequestBody()` silently.
- If a future consumer needs omitted-body semantics combined with required nested body fields, the framework will need explicit presence-tracking rather than the current “optional object with optional fields” contract.

## Finish Criteria

- A typed optional JSON body is expressible and generated without a custom decoder.
- EOF is accepted only for the intended optional-body contract shape.
- Regression tests cover at least:
  - empty body accepted
  - valid JSON body accepted
  - malformed JSON rejected
  - required-body endpoint still rejects empty input
