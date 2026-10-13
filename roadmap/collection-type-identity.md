# Collection type declaration identity (#733)

Status: implementation and bounded validation complete; final independent review
pending after independent
readiness approval of manifest `c44e2358d18306fb4815cf852e6e719843ca4f68a0914369fed7b47448633e9d` and root release.
Parent: `abeb200e9`; owner: root; validation: `review_uniform_json_corrections`;
worktree: `collection-type-identity`.

The accepted contract is that collection element references use the identifier
allocated for their declaration. External `struct:pkg:path` types retain their
own package identity; service reservations must not rename them. This task does
not reject valid collisions or add per-transport suffix rules.

## Sequence

- [x] Recite the test-first workflow, bounded comparison, model and independent
  readiness/final review gates before source edits. Preserve unrelated changes,
  use managed worktrees, never edit generated output or use forbidden Git cleanup.
- [x] Read issue #733, shared collection and scope rendering, HTTP transform
  contexts and existing external-type reservation controls.
- [x] Add direct declaration/reference regressions before implementation; run
  `go test ./codegen -run 'TestCollectionDeclarationIdentity|TestExternalTypeNamesIgnoreServiceReservations' -count=1`.
- [x] Generate the original accepted HTTP design and establish the current
  compile failure; retain ordinary and external-package controls.
- [x] Model declaration allocation and qualified reference consumption, including
  a negative control for blindly applying service reservations to external names.
- [x] Obtain independent readiness review of scope, regressions, model and bounded
  acceptance before changing the owning implementation.
- [x] Repair the shared type-reference owner; rerun direct and affected tests,
  bounded parent/candidate/repeat output and generated build/vet/wire checks.
- [ ] Update the owning guidance and receipt, obtain exact final diff review,
  then one atomic commit, push and proven issue closure. Clean temporary artifacts.

## Bounded acceptance

- Arrays, maps and nested collections reference the declared service name after
  collision allocation or fixed-name reservation, including both allocation orders.
- Singular references and collection references agree; same-package conversion
  removes only qualification, preserving declaration identity and presence.
- External named objects, scalars, arrays, maps and unions retain their own names
  regardless of service reservations; explicit import aliases remain correct.
- Preserve intrinsic/occurrence presence and custom type metadata behavior.
- Compare a small set of existing meaningful generated fixtures and the original
  collision witness at parent and candidate; compile/vet each accepted candidate.
  Repeat candidate generation in isolated processes and compare exact bytes.
- Enumerate intended differences. No full exported-design audit is required.

## Ownership hypothesis

`codegen.collectionTypeRef` delegates to the context's `NameScope`, but
`NameScope.userTypeDefWithPkgOverride` reconstructs every qualified user type
with `Goify(actual.Name())`. That discards allocated service identifiers while
trying to protect external package names from unrelated local reservations.
Existing `GoFullTypeName` distinguishes external declaration identity. Readiness
must establish the correct owner across both cases before choosing an API change.

## Readiness evidence

Before implementation, production source was unchanged. At the pinned parent the shared regression
has eight red qualified reserved/colliding element/array/map/nested cases. The
other 24 direct cases, external name reservations and the generated relocated
package control pass. The exact issue design fails generated HTTP server and
client compilation with `Item` versus `Item2` at the original six assignments.
Receipt: `/tmp/loom-733-readiness-red.log`. Affected shared/HTTP pinned lint
passes: `/tmp/loom-733-readiness-lint.log`.

TLC negative controls fail ReferenceIdentity: raw reconstruction 18 states;
blind service allocation 26 states. The declaration-owned candidate passes
278 generated / 158 distinct states. Logs `/tmp/loom-733-model-*.log`.

Implementation direction: make the existing qualified user-type-definition
renderer consume the common physical-name decision already used by singular
references. Avoid changing `collectionTypeRef` consumers independently or
introducing a new name registry. Preserve external aliases and existing
presence/type metadata branches. The model's prior-allocation assumption must
be checked in actual generated consumers; unallocated references retain their
existing spelling.

Bounded output inputs: the exact issue collision design in the generated test,
a variant using reserved `Use` as payload and result collection elements, and
existing `codegen/testdata.UserTypePackageTransportsDSL`. Compare parent and
candidate with identical commands; build/vet all candidate modules, then repeat
each candidate generation in a separate process and compare exact bytes.
Extend generated runtime checks to preserve distinct fields through payload and
result round trips rather than accepting successful compilation alone.
Existing scope collection, external package and presence transform tests cover
unchanged named scalar/union/nullable/custom representation seams.

## Implementation and acceptance receipt

The only production change is `NameScope.userTypeDefWithPkgOverride` in
`codegen/scope.go`: after existing package selection, it delegates to the existing
`goFullValueTypeName` owner. That owner already distinguishes allocated service
names from external physical names. Collection consumers, allocation order,
presence wrappers, and custom metadata branches are unchanged.

Root implemented this production change after readiness approval. The independent
validation agent completed the bounded acceptance and made only test formatting
and documentation changes. Final review must be performed by a fresh agent that
did not implement or extend these files.

Additional test-first evidence: `/tmp/loom-733-additional-red.log` records reversed
allocation order and nullable name failures before the change. Direct controls
also cover named scalar/map-key references and external aliases. The original
issue design is preserved in `TestCollectionDeclarationIdentityGeneratedHTTP`;
its DSL semantics have not changed. `CollectionDeclarationIdentityDSL` adds
payload/result arrays, maps and nested collections, and its HTTP round-trip
harness preserves distinct scalar values through each conversion.

Validation with Go 1.27.2:

- `/tmp/loom-733-owner-tests.log`: complete `codegen` and `codegen/service` tests
  pass (30.513s and 56.494s), including existing scope, transform, presence,
  external-package and checked-in golden controls.
- `/tmp/loom-733-http-final.log`: original issue, both collision/reserved runtime
  round trips, collection enum, named/nullable collection, union collection and
  relocated-package generated controls pass (42.068s).
- `/tmp/loom-733-transport-controls.log`: selected gRPC and JSON-RPC relocated
  package, collection and protobuf transform controls pass (5.628s and 9.660s).
- `/tmp/loom-733-changed-lint.log`: pinned golangci-lint 2.14.0 over shared/HTTP
  codegen and HTTP testdata reports zero issues in the final changed Go lines.
  The unfiltered scoped run (`/tmp/loom-733-final-lint.log`) reports two unchanged
  parent fixture findings: `custom_multipart_bindings_dsl.go` gocritic and
  `viewed_omitted_union_dsl.go` maintidx. Both source files are byte-identical to
  the parent and are handled separately by root; the overall lint gate is not
  represented as green.

### Exact output comparison

`/tmp/loom-733-comparison/report.json` records parent, candidate and repeat hashes
for the three approved inputs. The parent run overlays only `codegen/scope.go`
with its exact `abeb200e9` contents; the rest of the worktree has no other
production changes. All runs use the same renderer, inputs and command in
separate Go test processes. The local replacement used for generated module
compilation is identical, since this change affects only generator naming.

- Original issue: 12 generated artifacts, with only client `cli.go` and server
  `types.go` changed. Array/map allocations reference declared `Item2`, replacing
  the erroneous `Item` type. The parent retains the exact six reported compiler
  errors; the candidate builds and vets.
- Reserved `Use` payload/result: 12 artifacts, with only client `cli.go`, client
  `types.go` and server `types.go` changed. Array/map/nested allocations use
  declared `Use2`. Parent compilation fails; candidate build/vet and mounted
  generated-client HTTP round trip pass.
- Existing `UserTypePackageTransportsDSL`: 40 artifacts, no byte differences.
  The generated HTTP module builds/vets; the separate existing gRPC/JSON-RPC
  generated controls above cover those transport consumers.
- All 64 candidate artifacts match the independent repeat byte for byte.
  Every changed line is listed in `original.diff` and `reserved.diff`; the
  external comparison is empty. No public schema or wire semantics change.

`checks.json` and per-module logs preserve build/vet results. `source-pins.json`
records verified pushed base pin
`v1.10.0-alpha.9.0.20261013010152-abeb200e9f20` and the explicit local replacement.
The shared renderer initially writes its existing `v1.0.0` placeholder; comparison
modules were updated to the verified pin and tidied. This does not change the
local replacement or resolved dependency source, so identical generated code was
not rebuilt redundantly. Repeated output likewise reuses compile evidence.

The model source/configurations are unchanged; reuse the recorded two negative
controls and the 158-state candidate check. The README now identifies the actual
shared implementation seam. Runtime/compilation and exact output checks establish
implementation correspondence outside the model's naming abstraction.

No additional production bug was found. Regenerate affected designs to obtain
correct collection type references. Atomic commit/push/closure and shared
framework guidance are root-owned. Keep temporary comparison inputs and receipts
through final review; then clean inactive artifacts after preserving this summary.

## Integration review

The independent reviewer approved the exact ten-file candidate with no findings,
verified all 192 stored artifacts, and reran owner and generated round-trip
controls. Integration against `0b5657422` preserves every reviewed source byte;
only this receipt and the framework guide/model index are added. The two parent
fixture lint failures were separately repaired in #815. Final integrated review
is required before atomic commit and push.
