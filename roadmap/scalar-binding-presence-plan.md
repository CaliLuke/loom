# Scalar binding presence (#674)

Parent: `182a5cc8d1958bb78f399283450153f9696f6bd4`.

1. [x] Reproduce header/cookie request and response defects plus related required,
   numeric and custom-codec query cases with generated transport tests.
2. [x] Check the bounded scalar presence model: old content-based absence fails;
   explicit presence preserves supplied empty values, requiredness and defaults.
3. [x] Obtain independent readiness approval for the shared scalar decoder owner.
4. [x] Replace repeated scalar branches with one conversion/default/validation
   partial that consumes raw value and separately extracted presence. Keep list
   grammar and framed Authorization separate.
5. [x] Run direct rendered tests, generated HTTP and JSON-RPC request checks,
   HTTP response checks, selected existing specimens and process-isolated byte
   comparisons. Document intended generated differences and unchanged controls.
6. [ ] Update public guidance and owner/model correspondence. Freeze exact diff
   and acceptance evidence for independent review before one atomic commit.

The accepted contract distinguishes absence from supplied empty values. Required
bindings still reject absence even when they have a default: transport IR uses
`IsRequired`, not `IsRequiredNoDefault`. Defaults fill only optional absence.
An empty supplied value reaches its normal conversion and DSL validation; an
empty number or UUID therefore fails conversion rather than being dropped.
Header decoding retains first-value semantics and cookie parsing remains owned
by net/http. No authored tag or body codec semantics change.

## Acceptance evidence

- Readiness independently approved before production edits by the root agent.
  Parent generated modules compile; mounted request/response cases reproduce
  optional-empty loss, falsely missing required empties, and bypassed numeric or
  length validation. The same matrix demonstrates the related query gap.
- `TestScalarBindingPresenceGenerated` covers request headers/cookies/query,
  response headers/cookies, absence/empty/value, required/default combinations,
  named strings, bytes, zero/false defaults, conversion/length rejection, duplicate
  header first-value semantics and request-only custom UUID codecs. Mounted
  failures assert status, problem code/detail and no endpoint invocation.
- `TestJSONRPCScalarBindingPresence` exercises the shared request path through
  actual mounted JSON-RPC handlers; invalid bindings return -32602 without service
  invocation. Existing JSON-RPC wire/auth controls pass.
- The bounded TLA+ legacy model fails Presence for supplied empty input. The
  explicit-presence model passes 64 generated/32 distinct states. Required
  mappings still reject absence with a default, consistent with mapped.IsRequired.
- Affected Decode/ClientDecode goldens, generated transport name-collision
  compilation, cookie/security controls and direct scalar-plan checks pass under
  race detection. The initial combined run exposed two stale name assertions;
  those assertions were updated to the new extracted-local spelling, and the
  complete name assertion test passed separately (2.789s). Collection wire/shapes,
  optional-query and Basic-presence controls passed in the earlier focused run.
- Parent/candidate generation compares three specimens: the new scalar matrix
  and existing HeaderCollectionWire/HeaderCollectionShapes. Forty generated files
  have identical inventories; 32 are unchanged. Four decoder files change scalar
  presence handling. Four other files change only the deliberate collision's
  optionalPresent variable/CLI argument/flag to optionalPresent2 plus formatting.
  All six generated parent/candidate modules compile. Two independent candidate
  processes reproduce all 40 generated files byte-for-byte.
- Pinned golangci-lint on both affected generator packages reports zero issues.
  No full exported-design audit was run. The existing checked-in SSE fixtures and
  JSON-RPC goldens have no affected mapped scalar decoder branches.
- Receipts: /tmp/loom-674-{red,expanded,rpc,final-focused,name-assertion,lint}.log,
  /tmp/loom-674-comparison.json and model logs. Temporary archive source uses
  GIT_DIR pointing to the shared repository and GIT_WORK_TREE/LOOM_DIR pointing
  to the archive for generator-helper source resolution, with Go1.27.2/jsonv2.
- The separate pre-existing custom UUID response representation compilation
  failure is tracked by #923 and excluded from this change; its self-contained
  test input is retained as /tmp/loom-923-reproduction_test.go.

Independent final review, integration commit/push and issue closure remain with
root. Do not mark delivery complete before those steps.

Final self-review added `TestScalarBindingPreservesMapParamsValidation`: the first
candidate's duplicated classifier omitted whole-query map validation when its
Map flag was false. The regression failed that candidate; the corrected template
computes scalar classification once and shares it between dispatch and remaining
validation. This restores the parent's map validator and prevents classifier
lists from drifting. The bounded generated comparison and scalar/model evidence
remain applicable; affected rendered/mounted gates were repeated after this fix.
