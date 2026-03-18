# Form URL Encoded Decoding

## Goal

Make `application/x-www-form-urlencoded` request decoding first-class for typed and union payloads so applications do not need custom form parsers.

## Problem

`auto-k-server` still needs handwritten form parsing for OAuth token flows because generated request decoding does not fully support typed or union-shaped `application/x-www-form-urlencoded` payloads.

## Scope

Framework work only:

- HTTP request decoding for `application/x-www-form-urlencoded`
- typed payload and union payload support
- generated regression coverage for form decoding

Out of scope:

- application-specific OAuth semantics
- non-HTTP transport behavior

## Desired Outcome

- typed form payloads decode through generated transport code
- union-shaped form payloads use framework-owned discriminator behavior where applicable
- applications do not need custom form parsing for standard typed form endpoints

## Work Plan

1. Audit current form decoding support and identify which typed/union shapes fall through to app-local parsing.
2. Define the supported payload matrix:
   - scalar fields
   - optional fields
   - repeated fields
   - union/discriminator-driven shapes where contractually valid
3. Generate typed form decoding for the supported matrix.
4. Reuse existing validation and union-tag semantics after decode.
5. Add regression tests covering the OAuth token-style request shape from `auto-k-server`.

## Design Constraints

- Keep form decoding semantics aligned with existing Goa payload typing.
- Do not special-case OAuth in the framework; solve the generic transport capability.
- Be explicit about unsupported shapes rather than silently requiring custom parsing again.

## Risks

- Union handling for form payloads may need a stricter rule than JSON payloads.
- Repeated and optional form fields can create ambiguous presence semantics if not specified carefully.

## Finish Criteria

- Typed `application/x-www-form-urlencoded` payloads decode without app-local parsers.
- Supported union-shaped form payloads decode through generated transport code.
- Regression tests cover at least:
  - simple typed form payload
  - optional form fields
  - union/discriminator form payload
  - invalid form value handling
