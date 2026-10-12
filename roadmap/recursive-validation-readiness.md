# Recursive validator ownership (#709)

Status: isolated final review approved; integrated verification and final review pending, not delivered.
Implementation base: `66a9dc9090a06cd4136be1e5ebdaf26883ff90fa` (numeric owner delivered).
All ten reviewed readiness files were hash-verified before and after fast-forward.
Parent: `d57f96c6c393ab51c27a2d06c25298ca32e88af4` (pushed).
Worktree: `/Users/luca/.codex/worktrees/recursive-validation/loom`.

## Sequence

1. Promote the committed Optional recursion witness into direct red tests; compare
   native pointers, Optional and Nullable, both field orders, named indirection,
   array/map edges and mutually recursive shapes.
2. Trace declaration and call ownership across shared validation, service/view
   and transport generators. Establish whether named calls preserve occurrence
   constraints, physical carriers and error context; do not assume a local
   Optional substitution is sufficient.
3. Reproduce finite JSON trees at the generated request boundary, including deep
   invalid leaves and valid/absent/null controls. Separate cyclic in-memory values
   from finite decoded trees and generation termination.
4. Model the current in-progress buffer reuse and candidate finite named-validator
   graph. Record coverage, field-order independence, termination assumptions and
   correspondence; reject local depth limits or partial-buffer reuse.
5. Propose one shared occurrence-validation decision after presence unwrapping,
   with explicit declaration/call and error-path obligations. Preserve #796's
   delivered physical-context and selected-view behavior.
6. Freeze readiness source, red evidence and bounded production acceptance for an
   independent assessment before changing production. No main/shared-guide edits,
   commits or pushes in this task.

## Initial source evidence

`recurseValidationCode` caches mutable `bytes.Buffer` values by named type ID
before generating field checks. Returning an existing buffer observes only its
current prefix, with target and context names from a different occurrence.
`validateOptionalAttribute` and `validateNullableAttribute` enter that traversal
on their unwrapped named value; `validateNamedOccurrence` instead calls the named
validator. Arrays and maps already route named elements through that decision.
The #796 physical-context owner is unrelated and remains unchanged.

## Reproduction and boundaries

`TestRecursiveValidationReadiness` has 30 direct cases. Twelve fail: Optional and
Nullable object edges in self recursion, mutual recursion and named-object
indirection, with both field orders. Eighteen native-pointer and array/map
controls pass. The direct assertions require a named call but deliberately do
not prescribe its argument count; diagnostic context is part of the design below.

`TestRecursiveValidationGeneratedReadiness` builds six full HTTP modules at the
recorded parent with explicit local source and pushed dependency pin
`v1.10.0-alpha.9.0.20261012201303-8c7d8d711b9e`. The fixture contains a constrained
scalar alias and finite recursive trees. All modules compile. Coverage-only
checks fail four object-wrapper modules: invalid values at depth four are
accepted with the value field first; recursive-first order starts accepting at
depth three. The service runs and returns 204 instead of rejection with 400.
Native named calls through array/map edges reject invalid leaves at every tested
depth. Valid depths one through six, absence, nullable null and nonnullable-null
rejection are controls.

The final test also inspects `ServiceError.Field` from the real generated request
decoder. Inline recursion reports `body.next.value` for a `body.next.next.value`
error. Named array/map calls correctly reject values but restart their field at
`body.value`, losing the caller's occurrence path. These are two distinct failure
mechanisms of the same validator-call boundary. Routing only Optional to the
existing static named function fixes traversal but does not fulfill the ticket's
correct-path requirement. Six final modules therefore have expected diagnostic
or coverage failures; their successful valid-input cases remain intact.

The existing termination tests only prove finite generated text and syntax.
They do not prove all runtime occurrences are checked. Existing nonrecursive
Optional/Nullable, alias and occurrence-predicate controls remain unchanged and
pass. Some assert that a named body was expanded inline; production may replace
those text assumptions only after preserving the actual physical-layout checks
at the named definition and proving generated behavior.

No generator or runtime production source has changed. Cyclic in-memory Go values
are distinct from finite decoded JSON trees. Existing native validators already
recurse indefinitely through those values; this ticket does not promise arbitrary
cyclic-value termination, introduce a depth limit, or change supported DSL cycles.

## Proposed shared owner

Use a shared validation graph/occurrence owner in `codegen`, rather than replacing
one Optional call locally:

1. Distinguish generating a named validator definition from validating a named
   occurrence. A definition owns its complete local field traversal. A named
   occurrence emits a reference to the complete definition and never receives or
   copies the definition's partially written buffer. An active-definition set
   may ensure generation terminates, but cannot supply a validation body.
2. Presence wrappers own only admission/unwrapping. Once a value is available,
   both Optional and Nullable use the same occurrence decision as native fields,
   arrays/maps and other named edges. Preserve `presenceValueContext`, semantic
   nullability and selected-view mode. Named nullable declarations and their
   concrete carriers must remain distinct when their physical Go types differ;
   semantic type ID alone cannot identify the callable validator.
3. Reuse `validateNamedOccurrence`'s effective-constraint split: occurrence-local
   predicates remain at the edge; intrinsic predicates belong to the named
   definition. Preserve exact-predicate deduplication and declaring provenance.
   Do not lose a local required/enum/bound constraint when replacing inline text.
4. Represent diagnostic context independently from the Go target expression.
   A named invocation carries its caller's authored path into the callee. Nested
   field, collection and union traversal append the same suffixes used today.
   Static roots remain `body`, `result`, `message` or the existing entry root;
   recursive edges must not restart them or inherit another frame's literal.
   Do not fix paths by rewriting formatted error messages or parsing Go source.
5. One shared function descriptor must bind declaration identity, physical
   parameter type, allocated function/prefix, diagnostic parameter and root
   context. All declaration emitters and call sites derive from that descriptor.
   Its identity includes actual carrier and static validation mode, not merely a
   semantic UID. Runtime diagnostic paths are call arguments only: exclude them
   from helper identity, declaration keys and caches. Distinct recursive depths
   and sibling paths must reuse the same complete physical helper; otherwise
   path-dependent allocation can unroll recursion without bound.
   Decide the concrete generated helper signature during implementation review;
   it must expose a context-carrying call without introducing a second predicate
   renderer, per-transport recursion policy or pass-through helper layers. Root
   call sites supply their established root context. Regeneration is required.
6. Preserve declaration reachability and one definition per physical validator.
   Self/mutual recursion, repeated sibling occurrences and named indirection must
   terminate during generation without incomplete memoized text. Anonymous
   objects/containers may still render inline; scalar aliases retain their
   existing representation-sensitive inline predicates.

Known consumers that the shared descriptor must cover: HTTP/JSON-RPC body
`validateSection`, service/view `validateSection`, gRPC `addValidation` and its
renderer, untagged inline candidate validators, and authorization's
`validateAccess` prefix. The latter is a real alternate helper owner; an ordinary
`Validate` fallback would introduce unresolved or incorrect calls. Keep helper
visibility within the shared codegen owner and existing generator consumers.
The gRPC boundary also includes explicit calls in `codec_sections.go` (unary
request/response validation), `jennifer.go` (client designed-error validation),
and `jennifer_stream.go` (stream receive and designed-error validation, plus
viewed-result validation). Updating shared nested calls alone cannot establish
that these separately emitted entry calls still match their declarations.

The additional path propagation corrects a concrete defect observed through
array/map named calls, not a speculative new diagnostics API. Preserve error
classification, safe public messages and merged-error history; only the field
occurrence and corresponding internal diagnostic path should become accurate.
Whether missing-field errors report a leaf field or full path remains their
current owning convention; the repair must not independently redesign it.

## Model findings

`codegen/tla/recursive_validation/ValidationGraph.tla` reproduces mutable-prefix
coverage failure and stale paths, rejects complete named calls that retain a
static callee context, and checks complete declarations plus context-carrying
runtime calls. The model explores self/mutual graphs, both field orders, and
six-level finite input chains. The candidate separates a finite declaration
worklist from the decreasing finite input traversal. See the model README for
assumptions and correspondence; this is not a proof of physical Go typing,
all collection shapes, arbitrary runtime cycles or the future implementation.

## Bounded production acceptance after independent release

- Direct owner tests cover all 30 initial cases, occurrence-only and inherited
  constraints, absent/required/nullable wrappers, scalar aliases, named nullable
  declarations, mutual references, repeated sibling uses and custom validation
  prefixes. Generation must terminate and allocate complete declarations once.
- All six HTTP fixtures must reject every invalid depth and prevent service
  invocation, preserve valid/absent/null controls, and return occurrence-correct
  validation fields. Extend with one mutual/named-indirection fixture, one
  nonrecursive control and one occurrence-refinement control. Assert HTTP status,
  public problem code/detail, decoded payload and service calls; use race tests.
- Compile and exercise one JSON-RPC recursive request, one authorization-prefix
  recursion and the existing projected/selected-view controls. Reuse existing
  fixtures; do not audit the complete design corpus.
- Use one bounded representative gRPC specimen covering all separately emitted
  validator call forms: unary request and response (`codec_sections.go`), client
  designed errors (`jennifer.go`), stream receive and stream designed errors
  (`jennifer_stream.go`), including its viewed-result call when selected. Generate
  and compile/vet the complete client/server module; explicitly verify each call
  form has its compatible declaration. Exercise valid and deep-invalid recursive
  unary requests/responses and streaming receives under race detection, preserving
  service/receive barriers, expected error classification and established root
  contexts. Exercise designed-error decoding for unary and streaming paths so a
  signature-correct but wrong context argument cannot hide behind compilation.
  Reuse or extend one existing fixture rather than starting a broad gRPC corpus.
- Compare exact parent/candidate and independent-process artifacts for these
  bounded cases. List all changes to named validation calls/definitions and
  paths; ordinary predicate semantics and transport encodings stay stable.
  Every emitted named call must have one compatible declaration. Compile/vet
  each distinct candidate module; do not repeat builds of byte-identical output.
- Run affected direct/shared generator packages, relevant existing alias,
  occurrence, Optional/Nullable and #796 regressions; update only text/golden
  assumptions invalidated by the new complete helper owner, with runtime proof.
  Run pinned scoped lint and the model checks. Coordinate common numeric
  effective-constraint changes with #681 rather than overwriting them.
- Freeze exact source and evidence for a fresh final reviewer. This readiness
  neither authorizes production edits nor closes #709.

## Readiness evidence

- `/tmp/loom-recursive-709-direct-final.log`: twelve direct reds, eighteen controls.
- `/tmp/loom-recursive-709-generated.log`: initial coverage-only run, four red
  modules and two passing array/map modules (8.88s test / 9.611s package).
- `/tmp/loom-recursive-709-generated-paths.log`: final six-module diagnostic and
  coverage reproduction (11.38s test / 12.024s package).
- `/tmp/loom-recursive-709-controls.log`: existing physical-context, alias,
  occurrence-constraint and generation-termination tests.
- `/tmp/loom-recursive-709-model-final-*.log`: final checker results. Preserve
  these inputs/logs through independent readiness review; remove scratch after
  final task validation and review according to the repository lifecycle policy.

Existing controls pass in 0.802s. Pinned golangci-lint v2.14.0 reports zero
issues for `./codegen ./http/codegen` (`/tmp/loom-recursive-709-lint.log`).
Gofmt and diff checks pass. All tracked production files remain parent-identical.

## Independent readiness review amendment

`/root/review_recursive_validation_readiness` found a medium acceptance gap:
a single recursive gRPC message did not cover all independent validator-call
emitters. The bounded gRPC obligation now names unary request/response, client
designed-error, stream receive/error and viewed-result calls, with complete-module
compilation and recursive runtime/error-decoding checks. The helper identity rule
also explicitly excludes runtime paths. This is a plan-only amendment; existing
reproduction, model and control results remain valid. Targeted re-review is the
release gate; production is still unchanged.


## Implementation sequence after release

1. Route unwrapped named occurrences through the common occurrence decision;
   distinguish root nullable definitions and remove the partial-buffer cache.
2. Add one validation-function descriptor for declarations and calls. Generated
   validators accept optional diagnostic path components; absence uses their
   established root. Nested calls carry authored paths, and context expressions
   are rendered separately from target expressions and helper identity.
3. Adopt the descriptor at HTTP/JSON-RPC, service/views, gRPC and authorization
   declaration/call seams; preserve exact numeric predicates and physical types.
4. Turn direct/runtime reds green, extend the reviewed consumer matrix, then
   compare bounded parent/candidate/repeat output and compile/vet/race checks.
5. Update public regeneration guidance and model correspondence; freeze the
   final diff for a reviewer who did not implement it. No commits in this lane.


## Implementation completion and final review handoff

Independent readiness re-review resolved its sole medium finding before root
released implementation. The implementation starts at the exact numeric-owner
commit `66a9dc9090a06cd4136be1e5ebdaf26883ff90fa`; historical readiness statements
above describe the pre-implementation checkpoint, not the final source.

- [x] Remove the in-progress named-buffer cache and route unwrapped named
  occurrences through the common occurrence-validation decision.
- [x] Bind complete declarations and calls through `ValidationFunction`, keeping
  runtime diagnostic paths outside definition identity and allocation.
- [x] Adopt the descriptor across HTTP/JSON-RPC, service/views, authorization and
  all reviewed gRPC call forms, preserving exact numeric predicate rendering.
- [x] Complete direct, generated runtime, model, bounded output and quality gates.
- [x] Update public regeneration guidance and model correspondence; remove
  temporary capture drivers and source toggles from the implementation tree.
- [ ] Fresh independent final-diff review, main integration and delivery (root).

The unwrapped object value has a separate physical context fact from pointer
fields inside that object. Nested calls pass the object's address without
changing field pointer policy. gRPC retains the authored diagnostic root
separately from its Go parameter name. Validators accept optional diagnostic path
components; root calls preserve their established context, while recursive calls
carry the actual occurrence path. There is no depth limit, mutable prefix cache,
per-transport predicate policy or change to numeric thresholds. Cyclic in-memory
Go values remain outside the finite decoded-tree contract.

### Bounded artifact and runtime evidence

Parent, candidate and independent-process repeat outputs are retained under
`/tmp/loom-recursive-709-compare`. `comparison.json` records all 201 artifacts,
35 intended changed files and zero repeat differences; `difference.patch` records
every exact difference. `build-results.json` records successful tidy/build/vet
for all 13 distinct candidate modules. They cover eight HTTP recursive shapes,
one combined gRPC specimen, JSON-RPC, recursive authorization, an ordinary and
effective-constraint control, and nullable body controls. Each temporary module
uses pushed pin `v1.10.0-alpha.9.0.20261012201303-8c7d8d711b9e` plus an explicit
local-source replacement. The baseline archive is the exact implementation base.

The 35 changed files contain complete wrapper calls, object address arguments,
context-aware helper declarations and nested calls, and the retained explicit
named-alias value conversion. No wire type, serialization, transport mapping or
numeric threshold changes are intended. Eleven view declaration fragments plus
their regenerated checked-in aggregate are independently repeated byte-for-byte
under `/tmp/loom-recursive-709-views-{parent,candidate,repeat}`. These are fragment
goldens, not eleven separately compiled modules; their compilation controls are
the gRPC viewed result and existing selected/projected view fixtures.

- Eight generated HTTP modules pass race checks (22.230s), including invalid
  leaves at depths one through six, valid/null/absent inputs, status, safe public
  problem code/detail, internal occurrence path, decoded payload and service
  barriers: `/tmp/loom-recursive-709-http-expanded.log`.
- The generated gRPC matrix passes race checks (8.192s): recursive unary
  request/response, inbound/outbound streaming, unary/stream designed-error
  decoding and viewed-result client validation. All helper call forms compile:
  `/tmp/loom-recursive-709-grpc-runtime.log`.
- JSON-RPC recursive decoding and authorization-prefix recursion pass generated
  race checks (4.251s and 3.190s), including error paths and invocation barriers:
  `/tmp/loom-recursive-709-jsonrpc-runtime.log` and
  `/tmp/loom-recursive-709-auth-runtime.log`.
- Existing projected/selected/untagged #796 controls pass (38.621s); named
  scalar/collection Nullable, defaults and view-requiredness controls pass
  (92.638s); effective numeric/occurrence predicates and nullable body controls
  pass (11.908s). Logs are `/tmp/loom-recursive-709-service-stage.log`,
  `/tmp/loom-recursive-709-presence-controls.log` and
  `/tmp/loom-recursive-709-effective-controls.log`.
- The full shared codegen package passes (12.704s). Relevant HTTP/gRPC declaration,
  decode, CLI and call goldens pass. Final direct and service assertion checks
  pass (0.312s and 0.627s). Pinned golangci-lint v2.14.0 reports zero issues across
  the five affected owner packages. Logs are
  `/tmp/loom-recursive-709-codegen-final.log`,
  `/tmp/loom-recursive-709-goldens-initial.log`,
  `/tmp/loom-recursive-709-call-goldens.log`,
  `/tmp/loom-recursive-709-final-direct.log`,
  `/tmp/loom-recursive-709-service-assertions.log` and
  `/tmp/loom-recursive-709-lint-final.log`.
- TLC reproduces legacy coverage failure (17 states), legacy path failure
  (19 states) and complete-but-static-path failure (17 states). The candidate
  passes all invariants and declaration completion (28 states). Logs are
  `/tmp/loom-recursive-709-model-implementation-*.log`; durable model sources,
  configuration and implementation correspondence remain beside codegen.

The full service package run exposed one independent baseline fixture failure:
`TestSecureEndpointInit/endpoints-with-requirements` combines Basic and JWT on the
same default Authorization header, which the delivered wire-binding owner rejects.
The exact parent reproduces it in `/tmp/loom-recursive-709-security-parent.log`.
Root owns the separate fixture correction (explicit `token:X-Token` mapping);
this implementation does not change that fixture or its endpoint assertions.
The full run also exposed a mapped-name text assertion invalidated by dynamic
paths; that assertion was corrected and its focused rerun passes. No other
failure was reported in `/tmp/loom-recursive-709-service-full.log`.

Temporary capture inputs, generated modules and raw logs remain outside the
repository for independent review. Cleanup follows final review and is root-owned.
The final manifest covers the exact source and durable evidence, including all
new tests and model files; it excludes external checker state and build products.

## Final-review correction: named nullable callable identity

The first final reviewer found a medium regression: a nonrecursive named object
whose own declaration is `Nullable()` compiled at the parent but the candidate
passed `&actual` to a helper accepting `loom.Nullable[Node]`. Required field and
CLI paths both reproduced it. Evidence remains under
`/tmp/loom-recursive-709-review-namednullable-{parent,candidate}`.

Correction sequence (root released):

1. Add direct and generated red controls distinguishing named-owned Nullable
   from occurrence-only Nullable, including object and collection carriers,
   optional/required fields and root bodies.
2. Resolve named callable identity, physical parameter and static mode before
   occurrence unwrapping, through the common validation-function owner. Preserve
   local predicates on the unwrapped value and pass the actual wrapper only when
   the named declaration owns it. Do not restore partial inline named bodies.
3. Re-run invalidated direct/runtime checks, extend bounded parent/candidate/
   repeat comparison and compilation, then freeze for the same independent
   reviewer to confirm the repair and surrounding effects.

The correction sequence is complete; final re-review approved the isolated scope. The shared
`NamedValidationFunction` resolves the allocated identifier/prefix, physical
parameter reference and static field mode. HTTP body/attribute declarations,
gRPC named declarations, ordinary projected/untagged declarations and
authorization declarations derive these facts from that owner. Named nullable
occurrences retain their original intrinsic type while local predicates inspect
its concrete value. The complete helper receives the original carrier only when
its declaration owns presence; optional absence skips that required-root helper.
No transport-specific cast or inline recursive named-body expansion was added.

Correction evidence supersedes only the invalidated gates above:

- `TestNamedNullableValidationCallCarrier` first failed for the intrinsic Nullable
  case (`/tmp/loom-recursive-709-carrier-red.log`); it now checks the resolved
  carrier, allocated prefix, static mode and retained local Required predicate.
  The generated HTTP object matrix reproduced both server and CLI compilation
  failures before the repair (`/tmp/loom-recursive-709-namednullable-red.log`).
- The new fixture supplies six modules: HTTP and JSON-RPC, each with named
  nullable object, array and map declarations. Every module has required/optional
  field and selected-body-root methods, tested with valid, invalid, absent and
  null values. Generated tests assert response status/error code and service
  barriers, and inspect decoded presence/null state. JSON-RPC also asserts its
  established safe message, including the existing missing-field message.
- The combined original recursive and new named-nullable HTTP/JSON-RPC race
  matrix passes (32.241s / 14.263s):
  `/tmp/loom-recursive-709-correction-wire-final.log`. Existing recursive gRPC,
  authorization and projected union checks pass (4.653s / 30.926s):
  `/tmp/loom-recursive-709-correction-owners-final.log`. The full shared codegen
  package passes (15.714s), existing transport goldens pass unchanged (1.126s /
  0.437s), and final direct/view assertions pass (0.479s / 0.549s). See
  `/tmp/loom-recursive-709-correction-codegen-full.log`,
  `/tmp/loom-recursive-709-correction-goldens.log` and
  `/tmp/loom-recursive-709-correction-final-direct.log`.
- `/tmp/loom-recursive-709-repair-compare/comparison.json` records 267 generated
  artifacts, zero independent-repeat differences, and 50 intended parent
  differences. The original 201 artifacts are byte-identical to the first
  reviewed candidate, so their original 13-module compile/vet evidence remains
  valid. The six additional modules build/vet successfully; they also build/vet
  and pass the same race runtime controls at the exact parent, proving the
  correction restores existing named-nullable behavior. Reports are
  `named-build-results.json` and `named-parent-results.json` in that directory.
  `difference.patch` lists the original 35 intended changes plus 15 additional
  client/server/CLI files: complete declarations and diagnostic paths, with
  intrinsic-nullable calls preserving the original carrier. No other output
  changes occur. All temporary modules retain the verified pushed pin and an
  explicit local source replacement.
- Pinned scoped lint passes with zero issues in
  `/tmp/loom-recursive-709-correction-lint-final.log`. Its sole intermediate
  style finding was a mechanical if-chain-to-switch change; final direct checks
  pass after that change. The model sources are unchanged; their physical-call
  assumption and concrete test correspondence are clarified in the model README.
  The existing TLC evidence remains valid and is not claimed to prove Go typing.

Temporary capture branches and drivers were removed and their original source
bytes restored before final checks. The corrected manifest and delta from the
first reviewed manifest are retained for targeted independent re-review. No
commit, main checkout change or unrelated baseline fixture change was made here.

## Main integration

Integrated onto `1168ec6c1af34a9f6436aa3c84056e29ca8ee7fa`. The isolated 85-file
scope was independently approved after the medium nullable-carrier correction
and mechanical import-group correction, manifest
`0fed1c2d400fc1346c4ad05c1478aef4622c81531bb0a8fcadc2f7040db985fb`.
All production, model and test bytes are unchanged during integration. The DSL
reference retains the delivered typed-value admission documentation; shared
framework guidance records the owner. Main's separate authentication fixture
correction is retained. Integrated shared codegen/service suites pass (25.478s / 51.164s); the separate
combined-auth fixture now passes in the complete service package. Selected HTTP,
JSON-RPC and gRPC recursive/nullable generation and runtime checks pass (40.290s /
25.244s / 11.279s), all with explicit main-checkout LOOM_DIR. These gates exercise
the delivered typed-value owner and authentication fixture together with this
repair. Logs are `/tmp/loom-recursive-709-integrated-{shared,wire}.log`. The
subsequent parent `9f156f166` changes only recovery trackers. Final integrated
review is required before commit.
