# Constructor type identity (#894)

## Contract and reproduced owner defect

Field constructors must use the nominal Go type identity already assigned by
service analysis and the import aliases assigned by FileData. Local result or
temporary variable names do not belong in that package type namespace.

Archived pushed parent e67f359e25d7df4d0af1166a66bc572495e86dcc reproduces this
without #770 or interceptors: `Type("Client", String)` used by an optional HTTP
query field is declared `Client2`, while both HTTP server and client CLI cast to
`labels.Client`. Actual CLI generation succeeds, then the module build fails.
Changing only the authored name to Ordinary builds and vets. A direct shared
InitStructFields probe also fails. Inputs and command results live under
/tmp/loom-constructor-scope-parent and /tmp/loom-constructor-scope-*. The final candidate is based on e6673300f; its physical archived-parent
comparison independently reproduces the same defect.

`InitStructFields` constructs NewNameScopeLike(pkgs), retaining only import names
and discarding nominal cached allocations. It also reserved a local variable in
that fresh type scope. Both primitive casts and collection/object GoTransform
contexts consume the discarded identity. This is separate from #856's invalid
import identifier and from #770's missing public interceptor declarations.

## Correction and consumer inventory

1. Make the shared initializer consume the caller's canonical type scope for
   casts and transformation type references, preserving located import aliases.
   Remove the local-target reservation from the type namespace; any genuinely
   necessary local allocator remains private and separate. Retain nil-scope
   behavior for direct callers. Do not change NewNameScopeLike globally.
2. Check callers supply the scope of the physical target: HTTP constructor
   finalization and server/client type sections already pass Service.Scope;
   shared HTTP/JSON-RPC and gRPC CLI builders pass Service.Scope; gRPC metadata
   construction passes Service.Scope. HTTP problem-response construction has
   its own targetContext.Scope and must remain tied to that carrier, not acquire
   arbitrary service type allocations. Adjust only proven propagation gaps.
3. Keep #770's public symbol owner patch separate; its scalar alias competitor
   remains a required generated control after #894 delivery. No suffix/cast
   exceptions in consumers. No new allocator or formal model is required for
   preserving an existing identity map; reuse package-symbol invariants.

## Bounded acceptance

- Direct shared helper: allocated primitive name, pointer/value destinations,
  array/map of allocated aliases, ordinary unallocated names, explicit located
  import alias; demonstrate target-variable naming does not change type identity
  or perturb caller reservations. Preserve existing package-name tests.
- Existing generated-module seams: one HTTP design maps allocated scalar and
  collection aliases through request query/header and response header, exercising
  server/client constructors and CLI. Pair with ordinary control, and retain
  default/error/body construction controls from existing affected tests.
- One gRPC metadata specimen exercises allocated scalar/collection aliases in
  both directions and CLI; use existing metadata harness and shared model.
- One JSON-RPC existing constructor/CLI control establishes inherited path
  composition. FileData alias reanalysis uses existing direct tests.
- Compare parent/candidate/repeat outputs for that bounded selection, list every
  intended corrected type reference, compile generated code and use mounted
  roundtrip checks where an existing harness is available. Current three SSE
  fixture generation captures can be reused from #770 only if owner hashes prove
  the same source; otherwise regenerate affected fixtures once plus repeat.
- Run focused package/race/lint gates, not full design corpus.

## Sequence

- [x] Parent direct + generated failure and ordinary build/vet control.
- [x] Reconcile #856; file self-contained distinct #894.
- [x] Independent readiness review, then isolated implementation and bounded gates.
- [ ] Exact final independent review and separate atomic delivery by root.
- [ ] Resume #770 unchanged and rerun its invalidated scalar-alias checks.


## Final candidate and evidence

The shared initializer now retains its supplied scope for scalar casts and
nested transforms, with the existing nil-scope behavior. It no longer reserves
constructor locals as package declarations. The problem-response caller supplies
its own physical target scope. No public DSL or application policy changes.

The final comparison uses an actual Git archive of e6673300f, with the two new
test files added for the parent runs. Parent ordinary generation, build, vet and
mounted roundtrip pass; the reserved name fails compilation in six constructor
files. Both candidate modules pass build, vet and mounted HTTP race checks
(8.165s), including nil, empty and nonempty collections and located imports.
All 76 generated files match an independent repeat exactly. The ordinary module
is unchanged; only six reserved-name files change `Client` casts or collection
allocations to the already declared `Client2`: HTTP and gRPC client CLI,
client types and server types. JSON-RPC generated output remains unchanged.

Three checked-in SSE fixtures (HTTP ticktock, JSON-RPC ticktock and mixedtick)
were regenerated from the physical parent, candidate and independent repeat.
All 55 generated files per stage are identical; parent and candidate fixture
modules build and vet. No checked-in generated fixture edits are needed.

Focused direct races pass (codegen 1.584s, HTTP 2.503s). Existing metadata,
constructor ownership/reanalysis and selected problem-property controls pass
under race (gRPC 17.338s, HTTP 17.148s, JSON-RPC 16.302s). The full shared codegen
package race suite passes (38.356s); scoped configured lint reports zero issues.
The exact source, inputs, commands, logs and output hashes are recorded in
`/tmp/loom-894-final-manifest.json`; output deltas and fixture hashes are in
`/tmp/loom-894-generated-ledger.json` and `/tmp/loom-894-fixture-ledger.json`.

The initial direct test setup incorrectly assumed `GoFullTypeRef` allocates a
name and that separately located types share service-name reservations. Those
expectations were corrected before the valid parent run
(`/tmp/loom-894-direct-parent-final.log`); earlier setup failures are not evidence
for the repair. Archived generation also required empty Git metadata for the
existing repository-root discovery helper. It uses physical parent source, not
an overlay. Later harness formatting changed only whitespace and does not alter
generated artifacts or invalidate the preceding behavior checks.

## Independent response-header boundary (#895)

The expanded specimen also exposed a separate defect: response headers whose
array elements are named strings pass `[]labels.Label` to the native string-list
codec. The same failure reproduces on e6673300f without a naming collision;
`ArrayOf(String)` builds and vets. The original combined witness is preserved,
and the self-contained report is [#895](https://github.com/CaliLuke/loom/issues/895).
This candidate retains request-header collection and gRPC metadata conversion,
scalar response-header conversion and body collection roundtrips. It does not
claim that the combined named-element response-header case passes. That physical
header conversion owner remains separate; the accepted standard-list contract
and #770's required scalar competitor are unchanged.
