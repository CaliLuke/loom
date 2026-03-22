# Finish Checklist

This is the complete checklist to call the current effort finished.

The finish line is not "more ideas remain". The finish line is:

- the framework behavior is fully owned by `loom`
- representative downstream generation succeeds against the cleaned stack
- at least one real consumer design is simplified by the new DSL/capability
- the final release or repin can happen with understood blast radius

## A. Keep Framework Ownership Clean

- [x] Move generic union discriminator semantics into `loom`.
- [x] Move generic union example canonicalization into `loom`.
- [x] Move temporary root prepare/validate/finalize into `loom`.
- [x] Move synthesized JSON-RPC HTTP service construction into `loom`.
- [x] Move deterministic named user-type registration into `loom`.
- [x] Move remediation-aware error metadata into `loom`.
- [x] Move inline JSON Schema generation into `loom`.
- [x] Move generic attribute import gathering into `loom`.
- [ ] Audit for any remaining generic helper duplication or leaked consumer-specific behavior.
- [ ] Keep only framework-owned abstractions in roadmap and implementation docs.

## B. Finish The Remedy Boundary Cleanly

- [x] Add first-class remediation metadata to Loom error contracts.
- [ ] Decide whether remediation should project beyond OpenAPI and service data into generated runtime helpers.
- [ ] If yes, add transport projections for HTTP, JSON-RPC, and other Loom-owned transports without introducing app-specific coupling.
- [ ] Keep the generic abstraction in Loom-core and the concrete `remedy` package as an adapter target, not the framework root abstraction.

## C. Prove Downstream Generation Cleanly On Top Of Loom

- [ ] Point at least one representative downstream generator or consumer at local `loom` and make it compile cleanly.
- [ ] Run the relevant downstream suite against `loom`.
- [ ] Fix any breakage caused by moved or tightened core helpers.
- [ ] Commit downstream cleanup in isolated feature commits when needed.

## D. Prove The Stack With Temp Generation

- [ ] Point a temporary downstream generation run at the cleaned `loom`.
- [ ] Regenerate outside the repo tree, not into checked-in `gen/`.
- [ ] Compare the result against the current pinned output with `/Users/luca/code/loom/scripts/compare_regen.sh`.
- [ ] Confirm the remaining drift is intentional contract drift, not transport or codegen regression.

## E. Replace Real Design Glue In A Consumer

- [ ] Pick one representative auth area in a real consumer and replace duplicated bearer/browser auth DSL with:
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

- [x] `loom`: `go test ./...`
- [ ] representative downstream suite green against `loom`
- [ ] temp regeneration succeeds against the cleaned stack
- [ ] drift review is complete and understood
- [ ] at least one real consumer design area is simplified with the new DSL
- [ ] no remaining generic helper duplication or consumer-specific roadmap residue exists in `loom`
- [ ] roadmap and boundary docs reflect the final split accurately

## H. Actual Swap

- [ ] Repin the target downstream repo(s) from `goa` to `loom`.
- [ ] Regenerate the checked-in `gen/` tree.
- [ ] Run app tests and integration checks.
- [ ] Commit the swap as its own isolated change.

## Not Left

These are not open items anymore:

- transport/codegen parity work in `loom` for downstream consumers
- the OpenAPI 3.1 baseline migration
- the session auth/session cookie/auth error DSL foundation
- the major generic helper consolidation into `loom`

## Current Next Step

The next concrete step is:

- prove the cleaned stack against representative downstream generation and design cleanup

Until that is done, the boundary cleanup pass is not complete.
