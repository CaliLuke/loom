# Builtin uint numeric-bound admission — #696

Parent: `aa9ad3a3c`. All four numeric-bound DSL functions delegate source
eligibility to `expr.NewNumericBound`. That owner accidentally excludes builtin
`uint`, although exact scalar extraction already supports it. Named source
wrappers and pointers remain separate unsupported source types.

1. [x] Extend the existing source-admission and four-bound precision matrices
   with builtin uint, including the native maximum; record the parent failure.
2. [x] Admit uint in the shared numeric source adapter; update its GoDoc and the
   public numeric guide. Do not change arithmetic, schema grammar or carriers.
3. [x] Exercise the existing exact-numeric fixture with uint bounds and verify
   generated HTTP, JSON-RPC, gRPC and OpenAPI consumers. Compare bounded output
   with the equivalent previously accepted integer input and independent repeat.
4. [x] Obtain independent exact-diff review, then commit, push and reconcile #696.

Acceptance: uint bounds preserve the same exact number as equivalent uint64
values for inclusive/exclusive upper/lower bounds. Existing rejected source
controls remain rejected. Generated contracts retain the same numbers and
behavior; no transport-specific conversion is added. Source eligibility alone
changes; the existing exact arithmetic/projection models are unchanged.

## Verification and remaining review

The original source row and eight all-four-bound uint/maximum-uint cases fail
before the adapter change (`/tmp/loom-696-red.log`). Selected owner suites now
pass: DSL 0.677s, expr 0.721s, exact arithmetic 0.249s. Existing named-source,
pointer, nonfinite and string-syntax eligibility controls remain unchanged.

The existing exact-numeric fixture now authors its ordinary control bounds as
uint. HTTP and gRPC generated module/wire gates pass (4.947s and 3.887s). OpenAPI
JSON/YAML contract validation and pinned lint pass (0.647s). One bounded design
reuses that fixture across HTTP, gRPC and JSON-RPC: parent and candidate modules
build/vet, and all 40 generated artifacts are byte-identical across parent,
candidate and independent-process repeat. The comparison is recorded under
`/tmp/loom-696-comparison/{results,comparison}.json`.

The pre-existing combined JSON-RPC effective-constraints test stops at a stale
inline required-error assertion before reaching its numeric runtime checks.
The exact same assertion fails with a parent-source overlay (0.487s), so no
whole-test pass is claimed. Reusing its numeric harness in the independently
generated candidate module passes all 18 wire rows (0.446s), asserting status,
error code/message and service invocation. That older assertion is a separate
test-maintenance residual; this ticket does not weaken it.

Scoped lint reports one pre-existing unchecked `expr.RunInvalidDSL` result in
`dsl/numeric_value_admission_test.go:34`, unchanged by this repair. Changed-code
lint passes with zero issues. Logs are `/tmp/loom-696-{owner,transports,openapi,
jsonrpc-wire,parent-jsonrpc-control,lint,changed-lint}.log`. Independent final
review and delivery remain pending.

## Delivery

Independently reviewed, committed as `90aaa5f89`, pushed and #696 closed.
The final wording correction was re-reviewed before commit. The two identified
CI test residuals are separately delivered as #805 (`a9285e5ec`) and #806
(`6f1297e7a`). Inactive comparison modules were removed after review/delivery;
compact JSON receipts and command logs remain in `/tmp/loom-696-cleanup-evidence`.
Cleanup receipt `/tmp/loom-696-cleanup-receipt.json` SHA
`8ea6dd22462aea1abc62b7a7a5c5fc6632fca3740222d024c985a386af84be6f`.
