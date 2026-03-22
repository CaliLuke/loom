# Form URL Encoded Decoding

Status: completed on 2026-03-18

## Goal

Make `application/x-www-form-urlencoded` request decoding first-class for typed and union payloads so applications do not need custom form parsers.

## Problem

Downstream consumers still need handwritten form parsing for OAuth token flows when generated request decoding does not fully support typed or union-shaped `application/x-www-form-urlencoded` payloads.

## Scope

Framework work only:

- HTTP request decoding for `application/x-www-form-urlencoded`
- typed payload and union payload support
- generated regression coverage for form decoding

Out of scope:

- application-specific OAuth semantics
- non-HTTP transport behavior

## Delivered Outcome

- typed form payloads decode through generated transport code
- union-shaped form payloads use framework-owned discriminator behavior, including flat top-level fields for standard object-branch unions
- applications do not need custom form parsing for standard typed form endpoints

## What Changed

1. Added `FormRequest()` to the HTTP DSL so request bodies can explicitly opt into `application/x-www-form-urlencoded`.
2. Added endpoint validation so form requests are restricted to supported payload shapes and incompatible body/param combinations fail at design time.
3. Generated HTTP request decoders now parse form bodies through framework-owned decoding instead of forcing app-local parser hooks.
4. Generated HTTP request encoders now emit `application/x-www-form-urlencoded` bodies for matching endpoints.
5. Added runtime form helpers for typed object decoding plus union encoding and decoding that keeps scalar branches on the canonical `{type,value}` shape while flattening object branches onto standard form fields.
6. Flat object union branches with no required fields now accept discriminator-only form submissions, so cookie-backed OAuth refresh flows can select the branch without synthetic `value` fields or custom decoder shims.
7. Added wrapper delegation for direct union form payloads so generated request bodies do not introduce extra top-level nesting around standard OAuth-style fields.
8. Added regression coverage for typed, optional, invalid, and union-shaped form payloads, including an end-to-end temp-module test that exercises generated decoding with `golang.org/x/oauth2`, discriminator-only zero-field refresh branches, and the generated client path.

## Design Constraints

- Keep form decoding semantics aligned with existing Loom payload typing.
- Do not special-case OAuth in the framework; solve the generic transport capability.
- Be explicit about unsupported shapes rather than silently requiring custom parsing again.

## Remaining Follow-Up

- Decide whether `Optional JSON Bodies` should reuse pieces of the new form runtime helpers for presence handling or remain a separate HTTP body path.
- Revisit whether additional form payload shapes should be promoted from “unsupported by design-time validation” to first-class support once a real consumer needs them.

## Finish Criteria

- Typed `application/x-www-form-urlencoded` payloads decode without app-local parsers.
- Supported union-shaped form payloads decode through generated transport code, including flat OAuth-style object unions and discriminator-only selection of all-optional object branches.
- Regression tests cover at least:
  - simple typed form payload
  - optional form fields
  - union/discriminator form payload
  - invalid form value handling
