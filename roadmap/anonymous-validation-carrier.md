# Anonymous object validation carrier (#803)

Parent: `0dfab0ccd`. Owner: shared `codegen` validation, not an HTTP decoder workaround.

## Contract and sequence

An object unwrapped from Optional or Nullable is a concrete Go value. Its fields
retain their generated pointer/presence layout. Validation must distinguish the
current value from its field layout, for named and anonymous objects alike.
No DSL admission, default application, nullability or presence policy changes.

- [x] Reproduce original no-default anonymous nested-object compile failure on pushed parent.
- [x] Add direct normal/edge controls for unwrapped anonymous objects, object enums,
  nested native pointer fields, named calls, and Optional/Nullable modes.
- [x] Model current carrier confusion, reject clearing field layout wholesale,
  and check occurrence-local physical carrier with field reset.
- [x] Implement shared occurrence-carrier decision and reuse at all validation guard owners.
- [x] Run direct and affected package tests; compile and execute representative generated
  HTTP and JSON-RPC specimens, preserve named/bytes/collection controls.
- [x] Compare bounded parent/candidate outputs and independent candidate generations;
  list intended differences, retain rejection and absence/null tests.
- [x] Independent exact-diff review; resolve findings; integrate, commit, push,
  close #803.
- [x] Clean task-owned temporary output after preserving reviewed receipts.

## Evidence and limits

The original probe contains no Default or Enum and fails on `actual != nil`, where
`actual` is an anonymous struct with Optional children. The prior recursive named
validator repair correctly retained `AttributeContext.validationValue` but the
anonymous guard and predicate render-data owners still derive pointer status from
field layout. The recursive-validation model explicitly abstracts physical Go
carrier agreement; this extension checks that missing assumption, while generated
compile/runtime tests remain the implementation evidence.

## Frozen acceptance evidence

- Direct parent test: all four Optional/Nullable × enum/no-enum cases fail the
  concrete-value guard assertion. Generated parent: all eight HTTP/JSON-RPC cases
  fail build on anonymous struct nil comparisons (no default required).
- Candidate: eight modules build, vet and execute mounted transport tests under
  race detection; final formatted harness HTTP19.457s, JSON-RPC19.039s. Valid payload content reaches the
  endpoint; rejected input never invokes it. Tests assert HTTP status/problem code
  and applicable detail, JSON-RPC error code/message, absence/null and nested
  named/anonymous/array/map predicates.
- 88 generated Go files compared at parent/candidate/repeated independent process:
  exactly eight server `types.go` validators change; all other 80 files unchanged,
  all 88 repeat byte-for-byte. Intended differences remove invalid nil guards from
  unwrapped anonymous objects and their enums; native child/collection guards
  and named calls retain their carriers. Receipt `/tmp/loom-803-output-comparison.json`.
- Full shared-codegen package passes in 21.366s. Bounded named/nullable/recursive/bytes
  regression set passes shared1.753s, HTTP42.924s, JSON-RPC20.172s, gRPC6.891s.
  Shared/HTTP/JSON-RPC `go vet` passes. Existing HTTP server-type goldens pass in 0.425s unchanged; no checked-in generated
  fixture requires a byte change for the sampled existing contracts.
- Carrier model: current TypedGuard and clear-layout FieldLayout controls each
  fail at 13 distinct states; scoped candidate passes 14. Java25, pinned TLC2.19.
- No public semantics or authoring changes. Regenerate affected services to replace
  invalid validators. Separate gRPC Nullable conversion bug #804 reproduced without
  defaults; its transform ownership is outside this validator repair.

The affected shared/HTTP/JSON-RPC golangci-lint run reports zero issues.

Independent review approved the final 15-file diff against `0dfab0ccd`, patch
`01f13117241e621b887f525ee23abe742f4d045d4c1c29a9a9c1bd233de501a8`,
after confirming one low import-grouping correction. Delivered as `090427558`,
pushed to main and #803 confirmed closed. Reviewer independently reran the direct
and eight generated cases and scoped model. Temporary cleanup is the remaining
local housekeeping step; durable reproduction inputs and models are committed.
