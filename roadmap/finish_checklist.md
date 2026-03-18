# Finish Checklist

This is the complete checklist to call the current effort finished.

The finish line is not "more ideas remain". The finish line is:

- the generic Goa-core work is fully moved out of `goa-ai`
- `goa-ai` consumes the cleaned core instead of duplicating it
- `auto-k-server` proves the stack in real generation and real design cleanup
- the final swap can happen with understood blast radius

## A. Finish The Goa-AI To Goa-Core Boundary

- [x] Move generic union discriminator semantics into `goa-light`.
- [x] Move generic union example canonicalization into `goa-light`.
- [x] Move temporary root prepare/validate/finalize into `goa-light`.
- [x] Move synthesized JSON-RPC HTTP service construction into `goa-light`.
- [x] Move deterministic named user-type registration into `goa-light`.
- [x] Move remediation-aware error metadata into `goa-light`.
- [x] Move inline JSON Schema generation into `goa-light`.
- [x] Move generic attribute import gathering into `goa-light`.
- [ ] Rewire `goa-ai` to use those core helpers instead of local duplicates.
- [ ] Delete or collapse the now-redundant helpers in `goa-ai/codegen/shared`.
- [ ] Verify the remaining `goa-ai` shared code is only MCP or agent specific.

## B. Finish The Remedy Boundary Cleanly

- [x] Add first-class remediation metadata to Goa error contracts.
- [ ] Decide whether remediation should project beyond OpenAPI and service data into generated runtime helpers.
- [ ] If yes, add transport projections for HTTP, JSON-RPC, and other Goa-owned transports without introducing app-specific coupling.
- [ ] Remove any direct `remedy` package dependence from generic `goa-ai` code generation.
- [ ] Keep the generic abstraction in Goa-core and the concrete `remedy` package as an adapter target, not the framework root abstraction.

## C. Prove Goa-AI Cleanly On Top Of Goa-Light

- [ ] Point the `goa-ai` fork at local `goa-light` and make it compile cleanly.
- [ ] Run the full `goa-ai` test suite against `goa-light`.
- [ ] Fix any breakage caused by the moved core helpers.
- [ ] Commit the `goa-ai` cleanup in isolated feature commits.

## D. Prove The Stack In Auto-K With Temp Generation

- [ ] Point a temporary `auto-k-server` generation run at the cleaned `goa-light` and cleaned `goa-ai`.
- [ ] Regenerate outside the repo tree, not into checked-in `gen/`.
- [ ] Compare the result against the current pinned output with `/Users/luca/code/goa-light/scripts/compare_regen.sh`.
- [ ] Confirm the remaining drift is intentional contract drift, not transport or codegen regression.

## E. Replace Real Design Glue In Auto-K

- [ ] Pick one representative auth area in `auto-k-server` and replace duplicated bearer/browser auth DSL with:
  - `SessionAuth(...)`
  - `SessionSecurity(...)`
  - `SessionCookie(...)`
  - `AuthErrorResponses()`
- [ ] Regenerate into a temp tree and measure the actual reduction in duplicated payloads, cookie/header wiring, and auth-response boilerplate.
- [ ] If the result is clean, apply the same cleanup across the rest of the design.
- [ ] Confirm no new awkwardness appears in JSON-RPC, gRPC, or OpenAPI.

## F. Final OpenAPI Contract Review

- [ ] Review the final OpenAPI 3.1 artifact as a machine-consumable contract, not just documentation.
- [ ] Confirm `operationId`, schema naming, request bodies, union discriminators, security requirements, cookie docs, and examples are stable enough for the consumer app.
- [ ] Decide whether any further request-body naming or example-shape stabilization is needed.
- [ ] Lock final policy decisions with tests before the swap.

## G. Swap Readiness Gate

- [x] `goa-light`: `go test ./...`
- [ ] `goa-ai`: full suite green against `goa-light`
- [ ] `auto-k-server`: temp regeneration succeeds against the cleaned stack
- [ ] drift review is complete and understood
- [ ] at least one real `auto-k-server` design area is simplified with the new DSL
- [ ] no remaining generic helper duplication exists in `goa-ai`
- [ ] roadmap and boundary docs reflect the final split accurately

## H. Actual Swap

- [ ] Repin `auto-k-server` from `goa` to `goa-light`.
- [ ] Repin `goa-ai` to the cleaned version that consumes `goa-light`.
- [ ] Regenerate the checked-in `gen/` tree.
- [ ] Run app tests and integration checks.
- [ ] Commit the swap as its own isolated change.

## Not Left

These are not open items anymore:

- transport/codegen parity work in `goa-light` for `auto-k-server`
- the OpenAPI 3.1 baseline migration
- the session auth/session cookie/auth error DSL foundation
- the major generic helper moves out of `goa-ai/shared`

## Current Next Step

The next concrete step is:

- rewire `goa-ai` to consume the core helpers already moved into `goa-light`

Until that is done, the boundary cleanup pass is not complete.
