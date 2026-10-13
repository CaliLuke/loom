# HTTP contract snapshot alignment — #830

Parent: `942e58c6f5b5e50845f99a23af40d22799a5da3f`.
Managed worktree: `http-contract-golden-alignment/loom`.
Owner: golden expectations under `http/codegen/testdata/golden`; generator
production source and authored DSL remain unchanged.

1. [x] Reproduce the three exact CI failures on the current parent.
2. [x] Inspect each generated contract and compile the complete three specimens.
3. [x] Search for duplicates and file the bounded snapshot report (#830).
4. [x] Independent readiness approval before changing the expectations.
5. [x] Regenerate only the union-user decoder and two viewed WebSocket goldens.
6. [x] Run the affected golden families, contract controls and independent-process
   exact artifact comparison.
7. [ ] Obtain fresh final review before the atomic commit.

## Readiness evidence

`/tmp/loom-http-golden-parent-red.log` reproduces the three CI mismatches in
0.716s with Go 1.27.2, selecting only `TestDecode/decode-body-union-user` and
`TestWebSocketGoldenFiles/{websocket-server-streaming-with-views,websocket-bidirectional-streaming-with-views}`.

The union decoder expectation is missing its validator call and error return.
Its actual validator rejects nil pointers in the selected SomeType/SomeOtherType
branches while leaving absent optional unions alone. That presence-only body is
required by the delivered #823/#819 shared contract; no authored scalar predicate
is being invented.

Both WebSocket expectations use `s.view`; the generated body selector uses the
validated viewed result's `res.View`. This follows the centralized selection
owner delivered in #727 (`00fc9ba8b`) and agrees with its direct regression.

A temporary capture overlay reuses all three existing DSL functions and
`renderHTTPModule`. The complete parent artifacts compile/vet in 2.476s.
`TestWriteServerWebSocketResponseBodySwitchesViews` and the existing generated
viewed-result integration test pass in 5.538s, including runtime checks and its
race-enabled harness. These are expectation defects, not production defects.

Evidence lives under `/tmp/loom-http-golden-comparison` and
`/tmp/loom-http-golden-contract.log`. No production source or golden was changed
at readiness. A new formal model is unnecessary: this ticket changes only stored
expectations and relies on the owning validation/view contracts and regressions.

## Bounded acceptance

Regenerate the three named goldens through the existing `-update` mechanism.
The only intended expectation differences are one validator call/error block and
two selections changing from `s.view` to `res.View`. All complete generated
artifacts for the union-user, server-view and bidirectional-view designs must be
byte-identical to the parent and to an independent-process repeat. Preserve
compilation/vet and run both affected golden families plus existing contract
controls. Do not add a corpus sweep or weaken an assertion to avoid the mismatch.


## Final candidate evidence

Root approved readiness before regeneration. The exact three goldens were
regenerated with the existing `-update` flag. The final diff has only the intended
four-line validation/error block and two one-line view selector changes, plus this
plan. No generator, test logic or DSL changed.

Both complete affected golden families pass (0.803s). The parent contract controls
remain valid because production source is byte-identical: viewed WebSocket direct
and generated race/runtime checks pass (5.538s), physical union carrier direct and
generated runtime controls pass (0.672s and 1.959s).

The bounded comparison covers all 40 generated artifacts: union-user (12),
server-streaming with views (14) and bidirectional with views (14). Parent,
candidate and separate-process repeat hashes all match exactly. All three module
sets compile/vet; candidate generation/verification passes in 2.243s. Hashes and
logs are under `/tmp/loom-http-golden-comparison`.

Every captured temporary module requires the verified pushed parent pseudo-version
`v1.10.0-alpha.9.0.20261013030925-942e58c6f5b5`, with an explicit replacement to this
exact local worktree. The capture helper sets this requirement before tidy. Initial
parent/current module metadata was normalized to the same requirement and tidied
without regenerating or changing their artifacts. `go list -m` resolved the pin to
full parent `942e58c6f5b5e50845f99a23af40d22799a5da3f`. No source changes remain outside
the three goldens and this plan. Final review and integration are pending; retain
comparison outputs until they finish.
