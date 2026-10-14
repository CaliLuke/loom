# Recursive selected-view ownership (#788)

`expr.Project` selects the semantic graph before service conversion or transport
body type collection. Every nested result selects its authored child view or its
own default; a parent's non-default view does not implicitly choose the same view
for its children. The complete physical service/view carriers remain separate.

`RecursiveProjection.tla` abstracts direct, adjacent, array, map, union, ordinary
object and supported recursive-reference occurrences, parent and child views,
and two excluded-secret values. The legacy boundary converts a non-direct child
canonically; its negative control fails `NoExcludedDisclosure` on an adjacent
child using default. The checked design uses child selection on every path and
passes 112 distinct states, exact selection, no excluded read and termination.
This model does not prove extraction of arbitrary expression graphs, physical
pointer layouts, union dispatch or Go recursion. Actual cyclic Go values remain
outside the contract. It abstracts a recursive *type reference*, not a cyclic
runtime object.

`ProjectionIdentity.tla` explores every ordering of three occurrences sharing one
physical type but selecting default/full/default. The nominal-only helper cache
reuses the first conversion incorrectly. The candidate keys helper allocation and
references by semantic selection, so physical carrier reuse cannot widen or
narrow selected data. The legacy control fails `ReferencesMatchSelection` after
default then full; the checked control passes 14 generated / 8 distinct states. Generated tests additionally check that identity survives
`DupAtt` and that every helper call has a matching declaration. This finite model
assumes stable child selections and deterministic source ordering; it does not
prove the expression-to-selection adapter or namescope implementation.

## Reproduce

Use TLC 2.19 revision `5a47802`, JAR SHA-256
`936a262061c914694dfd669a543be24573c45d5aa0ff20a8b96b23d01e050e88`.
Recorded runs use Corretto Java 25.0.4.1. Set `TLA_TOOLS_JAR` to that installed
checker. From this directory run each command independently:

```sh
java -XX:+UseParallelGC -cp "$TLA_TOOLS_JAR" tlc2.TLC -workers 1 -metadir /tmp/loom-recursive-projection-legacy -config legacy.cfg RecursiveProjection.tla
java -XX:+UseParallelGC -cp "$TLA_TOOLS_JAR" tlc2.TLC -workers 1 -metadir /tmp/loom-recursive-projection-checked -config checked.cfg RecursiveProjection.tla
java -XX:+UseParallelGC -cp "$TLA_TOOLS_JAR" tlc2.TLC -workers 1 -metadir /tmp/loom-projection-identity-legacy -config identity-legacy.cfg ProjectionIdentity.tla
java -XX:+UseParallelGC -cp "$TLA_TOOLS_JAR" tlc2.TLC -workers 1 -metadir /tmp/loom-projection-identity-checked -config identity-checked.cfg ProjectionIdentity.tla
```

Legacy runs must report the named violated invariant (exit 12), not a parser or
resource error. Checked runs must complete successfully. Keep these sources and
concise evidence; remove task-owned state and trace output only after review.

## Implementation correspondence and evidence

- Baseline `4535a403`: `TestRecursiveResultProjection` fails maps and unions
  because `valueViewPlan.projectType` used `plainType`. Arrays already select
  structurally but service conversion ignores that selection.
- At the same baseline, `TestRecursiveProjectionExcludesFields` finds `Secret`
  copied by the generic child helper for collection, array, map and union paths.
  `buildConstructorCode` additionally deleted adjacent fields while iterating.
- `projection_conversion.go` consumes the selected graph and binds detached
  subsets to physical carriers. It retains union value layout while treating
  other projected fields as optional before selected-view validation. Semantic
  conversion identity survives mapped-attribute copies and names both emitted
  calls and helpers through `GoTransform`'s single allocator seam.
- Existing recursive collection tests exposed a partially initialized selected
  result whose view was unavailable during a back-reference. Expression
  projection now publishes its view header before visiting children.
- Generated HTTP, gRPC and JSON-RPC tests perturb excluded secrets, compare exact
  bytes, retain explicitly selected full-view data, and compile conversions.
  Final command/results and reviewed byte differences are recorded here before
  the implementation's final independent review.

The neighboring `result_output` model assumes projection is correct and owns
validation ordering/classification. These models add selection and helper
identity obligations without claiming that those earlier proofs covered them.


## Physical identity and layout boundary

The detached conversion graph preserves each physical `TypeName`, field name,
nullable marker and struct metadata. Its private UID/Identifier separates
selected subsets for `DupAtt`; `NameScope` still uses `UserTypeExpr.Hash`, whose
name-only hash excludes those private IDs. Full structural hashes of the selected
and physical graphs key helper identity. Requiredness is intentionally excluded
from that identity: it changes validation, not the selected field set; physical
layout remains supplied by the physical carrier. Two views with identical field
sets and different requiredness may reuse the same conversion.

Named pointer carriers normalize non-union requiredness for guarded reads.
Anonymous objects keep physical requiredness because their tags are part of Go
type identity. The existing HTTP presence module caught an over-broad version of
that normalization (`json:"name,omitempty"` vs `json:"name"`); preserving inline
layout fixes it. Customized-result optional-view validation, the HTTP inline and
named presence matrix, and generated default/full child selections verify this
boundary. Existing recursive object/collection fixtures verify that private
conversion identities terminate and keep references to their declared physical
carriers. These tests establish bounded correspondence, not hash collision or
arbitrary memory-cycle proofs.

## Explicitly excluded siblings

- #784: `excluded_union.go.txt` is a standalone DSL repro. It still generates
  `undefined: KindCircleResponseBody` / `KindNameResponseBody` on the HTTP server
  after semantic projection is repaired. HTTP analysis shares one `UnionTypes`
  registry across client/server packages with different physical body graphs.
  Root split this physical declaration-owner issue; it is not an accepted test
  or part of the #788 implementation.
- #727: stream selection of a non-default runtime view is a separate transport
  decision. This fixture streams the default parent and an explicit nested view.
- JSON-RPC SSE envelopes contain map keys whose order already varies at the
  baseline (`jsonrpc`, `method`, `params`, `loom_view`). The SSE regression checks
  envelope semantics and compares **exact raw params bytes**, without decoding
  or re-encoding the selected value. It does not claim whole-envelope wire
  determinism. Other selected wire comparisons cover complete HTTP/SSE JSON and
  deterministic gRPC protobuf bytes. Independent-process generated-file
  determinism includes every generated byte, with no exclusions.


## Final bounded validation

Source: baseline `4535a403048510f055fc59ec3dfed41e3769ad58` versus the uncommitted
#788 candidate in the managed recursive-projection worktree. Toolchain: Go
1.27.2 darwin/arm64; Java/checker pin above. No dependency change.

Passed direct packages:

```sh
go test ./expr ./codegen ./codegen/service -count=1
```

Passed generated regression and neighboring gates (each command uses
`LOOM_DIR=/Users/luca/.codex/worktrees/recursive-projection/loom`):

```sh
go test ./http/codegen ./grpc/codegen ./jsonrpc/codegen -run '^TestRecursiveProjection' -count=1
go test ./http/codegen -run 'TestCustomizedResultViewsGeneratedModule|TestResultViewPresenceGeneratedIntegration' -count=1
go test ./jsonrpc/codegen -run 'TestJSONRPCViewedUnaryResultGeneratedModule|TestJSONRPCViewedStreamResultGeneratedModule' -count=1
go test ./grpc/codegen -run '^TestGRPCProjectionParity$' -count=1
go test ./grpc/codegen -run '^TestResponseEncoder$/^response-encoder-result-with-(views|explicit-view)$' -count=1
go test ./grpc/codegen -run '^TestStreaming$/^server-streaming-result-(with-views|collection-with-explicit-views)$' -count=1
```

The new transport modules run `go vet ./gen/...` and `go test -race -count=1`
on the rendered module (gRPC excludes generated empty application test
scaffolds). They exercise both union-first and array-first declarations,
adjacent direct/collection fields, array/map/map-of-array, ordinary wrapper,
union branch, explicit full child view, absent and empty collections, HTTP
reverse conversion, unary messages and one stream per transport. Canonical
`secret` is required, so omitted fields cannot be accidentally dereferenced.
Existing expr tests retain authored union-cycle rejection.

Ten retained CLI cases cover the same six transport/order variants and these
four existing service fixtures: `ResultWithResultTypeDSL`,
`ResultWithRecursiveResultTypeDSL`,
`ResultWithRecursiveCollectionOfResultTypeDSL`, and
`ResultWithOneOfInResultTypeDSL`. The fixture is copied unchanged into standalone
design modules for all three source selections; baseline source stays read-only.
Each module requires pushed version
`v1.10.0-alpha.8.0.20261011180955-4535a4030485`, with a local replacement to the
selected source. Run `go mod tidy`, then:

```sh
LOOM_DIR="$SOURCE" go run -mod=mod github.com/CaliLuke/loom/cmd/loom gen example.com/projectioncomparison/design
LOOM_DIR="$SOURCE" go run -mod=mod github.com/CaliLuke/loom/cmd/loom example example.com/projectioncomparison/design
go mod tidy
go build ./...
go vet ./gen/... ./design/... ./cmd/...
```

All 30 generation/build/vet receipts passed. Current and repeat invoke fresh CLI
processes in separate modules. Every one of their **124 generated files** is
byte-identical, including manifests. The six unchanged baseline-generated
runtime probes fail on copied excluded fields, gRPC unary/stream disclosure or
required-field nil dereferences; all six current probes pass `go test -race
-count=1 ./probe`. HTTP/JSON-RPC array-first body conversion can hide the carrier
bug, so the probes also directly assert the generated service projection.

Exact intended base/current differences:

- All ten cases change only their service `service.go` conversion bodies/helper
  declarations, except the two additional differences below. Selected subsets
  replace generic canonical copies; helpers use conversion-local semantic
  identity and nil-aware collection loops. Public helper names/signatures stay
  unchanged. Excluded fields remain zero/absent; explicitly full children keep
  their selected secret.
- HTTP union-first and JSON-RPC union-first also change the respective client's
  and server's `types.go` and `encode_decode.go`: `OwnerResponseBody` loses the
  excluded `Secret`, the client no longer requires it, and both conversion
  directions omit its assignment. Array-first transport bodies already used
  the default selection and stay byte-identical. gRPC schemas and all other
  transport/schema/manifest bytes stay unchanged.
- Exactly five checked-in service goldens change: `service-result-collection-multiple-views`
  and `service-result-with-result-collection` adopt generic collection loops and
  selected child helpers; `service-result-with-other-result` and
  `service-result-with-dashed-mime-type` use the same nested conversion renderer;
  `service-result-with-one-of-type` uses semantic helper names. No unrelated
  golden was updated.

Neighboring checked models pass unchanged: view-union names (13 distinct), union
registration (256 distinct), projected presence (192 distinct). These preserve
existing ordering/layout assumptions while the new models establish selection
and semantic helper obligations. `git diff --check` passes.

Review artifacts are retained under
`/tmp/loom-recursive-projection-comparison`: each case has exact source,
commands/results, generated bytes, and runtime probes; the root holds exact
base/current diffs, SHA-256 manifests and byte/runtime comparison summaries.
After independent review/integration, remove these task-owned artifacts and
checker state directories; keep this concise evidence and reproduction inputs.

## Carrier location snapshot ordering

`CarrierLocation.tla` captures a custom-package regression discovered by the
existing `TestStructPkgPathFrameworkImportNamesCompile` CI matrix. Projection
previously built conversion snapshots before removing canonical package metadata
from view carriers. Detached metadata copies preserve that stale location even
when the carrier is lowered later, emitting references such as `common.CardView`
to a declaration that belongs in the service views package.

The shared carrier preparation boundary must lower its graph's package location
before name allocation, conversion snapshots or reference generation. Canonical
service types keep their authored locations. The finite model has root, child
and recursive carrier symbols: legacy order violates
`ReferencesNameDeclaration` after four states; corrected order checks both that
invariant and `CanonicalLocationPreserved`, with termination (five generated,
four distinct states). These symbolic carriers do not prove graph traversal,
recursive Go compilation, alias allocation or view selection.

Using the checker pin above, run from this directory:

```sh
java -XX:+UseParallelGC -cp "$TLA_TOOLS_JAR" tlc2.TLC -workers 1 -metadir /tmp/loom-carrier-location-legacy -config carrier-location-legacy.cfg CarrierLocation.tla
java -XX:+UseParallelGC -cp "$TLA_TOOLS_JAR" tlc2.TLC -workers 1 -metadir /tmp/loom-carrier-location-checked -config carrier-location-checked.cfg CarrierLocation.tla
```

`TestProjectedCarrierPackageReferences` covers canonical and projected
references, nested arrays, recursive children, selected-secret exclusion and
file alias reanalysis. The generator's existing custom-package matrix supplies
rendered compilation coverage.
