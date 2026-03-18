# Optional JSON Bodies

## Goal

Allow endpoints to accept either no JSON body or a typed JSON body without handwritten EOF-tolerant decoder logic.

## Problem

Some endpoints legitimately accept an empty body or a small typed JSON body. Today `auto-k-server` still needs custom HTTP decoder handling to tolerate EOF on those endpoints instead of relying on generated transport behavior.

## Scope

Framework work only:

- HTTP JSON request decoding behavior
- DSL-to-transport semantics for optional request bodies
- generated tests around empty-body versus typed-body handling

Out of scope:

- application-specific defaulting after decode
- custom auth/session runtime logic

## Desired Outcome

- the DSL can express an optional JSON request body clearly
- generated decoders accept an empty body when the contract allows it
- generated decoders still reject malformed JSON and invalid typed payloads

## Work Plan

1. Define the contract rule for “optional JSON body” in Goa-light terms.
2. Audit current HTTP JSON decoder generation to identify where EOF is treated as a hard error.
3. Add generated behavior that treats EOF as empty-body success only when the contract allows it.
4. Preserve normal validation and required-field checks when a body is present.
5. Add regression tests for the OAuth-style endpoint shape that motivated this request.

## Design Constraints

- Do not blur the difference between “body omitted” and “body present but invalid”.
- The feature should be contract-driven, not based on app-local decoder swaps.
- Avoid broad decoder leniency that changes behavior for existing required-body endpoints.

## Risks

- Optional-body semantics can become ambiguous if the DSL does not express them cleanly.
- Overly broad EOF tolerance could accidentally weaken required-body endpoints.

## Finish Criteria

- A typed optional JSON body is expressible and generated without a custom decoder.
- EOF is accepted only for the intended optional-body contract shape.
- Regression tests cover at least:
  - empty body accepted
  - valid JSON body accepted
  - malformed JSON rejected
  - required-body endpoint still rejects empty input
