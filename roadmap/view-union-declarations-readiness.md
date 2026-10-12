# Viewed response union declarations (#784)

Status: implementation frozen for independent final review; no commit or push.
Parent: `8b5d265cb21b885a85c1e77385ef5638f14b9f91`.
Worktree: `/Users/luca/.codex/worktrees/view-union-declarations/loom`.

## Sequence

1. Reproduce the reported omitted nested union in a reusable HTTP testdata design
   and a generated module build. Add a direct declaration reachability assertion.
2. Inspect server/client body projection and declaration collection together;
   test JSON-RPC separately rather than infer that shared analysis proves it.
3. Establish one body-declaration graph per emitted package, with declarations,
   branch dependencies and validators derived from its actual prepared bodies.
   Model the current shared-registry failure before selecting a replacement.
4. Cover excluded/included nested views, arrays/maps, request/response reuse,
   errors, recursion and deterministic declaration order with bounded cases.
5. Obtain independent readiness review before implementation; then compare
   parent/candidate artifacts and compile selected modules, preserving view wire
   selection and already-delivered secret exclusion. Final exact-diff review,
   atomic commit, push and issue reconciliation follow.

## Initial source evidence

`ServicesData.analyze` stores one `ServiceData.UnionTypes` list collected from
request bodies, effective client response bodies and errors. Both `serverType`
and `clientType` emit that list. Server response declarations instead pass through
`projectResponseBodyView` and `collectServerResponseBodyTypes`, so a client union
can be emitted in the server package without the branches it references.

The shared registry is the suspected ownership error. Adding missing unused
branch types or filtering only the reported field would retain that disagreement.
The fix must preserve normal request/response reuse, named union identity and
package-local validation requirements. #733 name allocation is separate unless
new evidence establishes the same owner; no other report is claimed resolved.

## Reproduction checkpoint

Current parent `8b5d265cb` reproduces both a direct server reachability failure and
full generated HTTP compilation failure. Equivalent JSON-RPC compilation fails
on the same missing branch types. JSON-RPC delegates its client/server type file
construction to HTTP codegen, so both consumers are now actual witnesses.
Fixture: `http/codegen/testdata/viewed_omitted_union_dsl.go`.
Tests: `http/codegen/view_union_declarations_test.go` and the JSON-RPC counterpart.
The generated module helpers use explicit local source and the already-pushed
framework pin. Logs remain under `/tmp/loom-view-union-784-*` until review.

The legacy declaration-owner model reproduces the actual mismatch in nine
states; the candidate package-specific owner passes all three invariants in
twelve states. This only models prepared graph membership and a union's branch
dependency. It does not establish name allocation, recursive traversal or
runtime projection. See `http/codegen/tla/union_declarations/README.md`.

## Candidate owner and unresolved implementation details

Server union reachability must consume the projected `body` at
`buildResponseBodyType`, the same graph passed to `collectServerResponseBodyTypes`.
Client reachability must consume its effective response carrier. Request,
streaming request and error body roots must be included independently for both
packages. Keep this in the existing shared type analysis; JSON-RPC consumes it.

The current union builder allocates `Kind` names with `NameScope.Unique`, so
simply invoking it twice can introduce naming drift. The proposal needs a single
allocated declaration record per physical identity plus package membership, or
independent package scopes with consistently retained references. Choose after
examining actual body identities and existing order tests; do not reconstruct
Go names while filtering. Any union validator references must have the same
package's branch declaration/validator. Root authorized this owner after independent readiness review.

## Independent assessment and selected owner

The proposal fits the actual source boundary. `buildRequestBodyType` receives
prepared request and streaming roots for each package. `buildResponseBodyType`
projects the response view before computing type data and collecting server
branches; it also covers errors through `buildResponseBodyPair`. Register those
actual prepared roots at these owners, including their request/response and
package context. Never reconstruct membership from the canonical response later.

The registry must distinguish two decisions:

1. **Physical declaration allocation:** one immutable `uniongen.Type` per existing
   allocated Go declaration identity. Keep `NameScope.GoTypeName` and the current
   enclosing-user-type identity rule. Inner union shape is not a declaration key.
   `buildHTTPUnionTypeData` calls `scope.Unique` for the discriminator name; call
   it once for a declaration, then reuse the record.
2. **Package membership:** independent traversal/visited sets and emitted-name
   sets for client and server prepared roots. A globally visited type cannot
   suppress its later membership in another package. Requests shared with an
   omitted response still require their request union; they do not justify the
   unreachable response union. Sort each package's final declarations.

Preserve established allocation order by completing client-root discovery in
the existing request/stream/response/error endpoint order before allocating any
new server-only declarations. Existing names and kind constants must remain
byte-identical in preservation controls. Keep the existing declaration-identity
model and its 576 allocation/collection order pairs; package reachability is an
additional invariant, not a replacement for identity.

The same registered roots must supply branch declaration and validator discovery.
Keep existing physical layout and ordinary declaration owners, but replace their
parallel canonical-root rediscovery with these inputs where applicable. This
does not authorize changing field presence or generating unused branch types.
Projected server roots must be registered before response validation/type data
is built. The existing `recordServiceTypeLayouts` identifies untagged response
validators from canonical hashes; a projected body can have different physical
identity. Validator requirements must follow the actual registered body. The
included untagged control currently passes, so this is a required correspondence
check, not an independently established second defect.

Replace the shared emitted `ServiceData.UnionTypes` list with package-specific
records; do not retain a compatibility alias that lets renderers choose the
wrong list. Remove `collectEndpointUnionTypes` canonical response traversal.
HTTP type files consume their package's list; JSON-RPC continues to derive these
files through the shared HTTP owner. Do not add JSON-RPC-specific filtering.

## Bounded controls and reproduced outcomes

All checks use Go 1.27.2 with explicit `LOOM_DIR` selecting this worktree. Existing
module helpers resolve the source through `internal/loomsource`; default remote
mode fetches the pushed parent commit. Comparison modules before final review
must also record an exact pushed pseudo-version plus their explicit local replace.

`ViewedUnionDSL` extends the original reusable fixture with named options, without
changing its two default exported designs. Six generated HTTP cases establish:

- Included tagged union, included map and included untagged object union all
  compile and vet. The untagged variants cover a real numeric validation and a
  branch with no authored validation (which still needs a callable validator).
- Excluded tagged response union with request reuse, excluded untagged union and
  excluded map all fail compilation on missing response branch types. Request
  reuse is therefore an independent membership requirement, not permission to
  emit every shape in both packages. Combined run: 6.143s, three expected reds
  and three passing controls; `/tmp/loom-view-union-784-controls.log`.
- The original JSON-RPC generated fixture still fails on the same undefined
  branch declarations. Included/recursive JSON-RPC projection controls in the
  same run do not fail; `/tmp/loom-view-union-784-jsonrpc-controls.log` (8.113s).
- Existing allocated-name permutations, object-order determinism, nested
  request unions through objects/arrays/maps, union success/error bodies and
  recursive HTTP view projection pass (14.879s). The generated error and view
  cases include wire/race checks and retain prior secret omission behavior.
  `/tmp/loom-view-union-784-preservation.log`.
- Independent TLC rerun reproduces the legacy missing-branch counterexample
  at nine states and passes the candidate at twelve states. The preexisting
  `UnionDeclarations.tla` allocation model remains unchanged.

## Production acceptance after root release

1. Add direct red assertions for per-package membership, shared request reuse,
   repeated root registration and one allocation per physical declaration.
   Keep all existing named-union identity and order assertions meaningful.
2. Implement registered prepared roots, one declaration allocator and separate
   client/server membership. Feed branch/validator discovery from those roots;
   remove the superseded shared collection path. No view projection rewrite.
3. Promote the current readiness witnesses to passing regressions. Extend direct
   assertions to included/excluded fields and untagged validator availability;
   do not rely solely on compilation. Check recursive graph termination and
   preserve both field orders in the existing projection fixture.
4. Compare a bounded parent/candidate matrix: original HTTP and JSON-RPC witness,
   included union, omitted-with-request-reuse, included/omitted untagged and map
   controls, existing error-body and recursive projection designs. Reuse existing
   harnesses. Record every changed file and verify selected/omitted wire data.
   Compile/vet modules and compare independent-process output bytes; reuse builds
   for byte-identical repeats.
5. Run affected direct/package checks and pinned scoped lint. Document the
   registry invariant and model correspondence, then freeze the exact diff for
   a different agent's final review. Root integrates shared skills, delivery and
   cleanup. The following implementation record supersedes the readiness status.


## Implementation and acceptance record

`ServiceData.bodyRoots` registers each nonempty prepared request/response body at
its declaration owner. Response registration follows `projectResponseBodyView`
and precedes validation/type construction. Request and streaming roots retain
their package and physical presence context. The same roots feed ordinary branch
declarations, validator requirements and union membership. Server response
branches remain collected at the original declaration point, preserving allocation
order; the canonical post-analysis response walk has been removed.

`collectBodyUnionTypes` allocates each union once by the existing NameScope name,
then maintains independent server/client membership and traversal sets. Client
roots retain endpoint/request/stream/response/error order; server-only declarations
follow. `ServerUnionTypes` and `ClientUnionTypes` replace the shared emitted list.
HTTP renderers choose the correct list; JSON-RPC inherits this shared behavior.
No request-body presence, view projection or allocation identity policy changed.

Direct evidence:

- New `TestViewedUnionPackageMembership` reproduced three parent failures:
  omitted response, request reuse, and omitted object-branch response. All pass.
- `TestPreparedBodyUnionAllocationAndMembership` registers repeated roots in both
  package orders and verifies shared declarations are the same allocated record,
  discriminators do not gain a suffix, and package-only membership stays separate.
- `TestViewedUntaggedUnionValidatorDependencies` explicitly verifies `Untagged`
  and callable validators for both constrained and unconstrained object branches.
- Existing server type goldens, direct builders, union naming/identity/order,
  nested body, error wire/race, and recursive HTTP projection cases pass in
  13.553s (`/tmp/loom-view-union-784-owners.log`).
- Corrected generated controls plus direct tests pass in 9.363s. New selected/
  omitted union encode/decode tests pass with race detection in 5.598s: HTTP 200,
  default/tiny view JSON, retained variant value and excluded field absence are
  asserted. Existing recursive controls retain their secret exclusion checks.

### Correction to the readiness evidence

The initial fixture option named `Untagged` omitted the DSL `Untagged()` call.
Its initial successful build therefore proved tagged object-branch coverage,
not untagged matching. That subclaim is invalidated. The fixture now calls
`Untagged()` explicitly, and the direct validator test asserts that fact.

The corrected fixture first exposed an independent existing service-view bug:
a required numeric branch field is projected to `*int`, but its generated view
validator compares `v.Radius < 1`. Both parent and candidate fail identically in
`gen/svc/views/view.go:312`. The exact required-field input is retained as
`http/codegen/tla/union_declarations/required_untagged_probe.go.txt`. In a temporary
copy, replace `http/codegen/testdata/viewed_omitted_union_dsl.go` with that file and
run `TestViewedUnionPackageControls/included_untagged` with explicit local source.
This belongs to service-view validation, not HTTP declaration membership, and is
reported separately. Its parent log is
`/tmp/loom-view-union-784-required-parent-build.log`.

The bounded #784 fixture uses an optional numeric branch with `Minimum(1)`, which
still requires a real validator, plus an unconstrained branch requiring a stub.
The corrected parent included case builds; its omitted case fails on missing
`CircleResponseBody`/`NameResponseBody`. Both candidate cases compile and vet.

### Exact generated comparison

Parent archive: `8b5d265cb21b885a85c1e77385ef5638f14b9f91`, under
`/tmp/loom-view-union-784-parent`. Only testdata/capture drivers were added; a local
empty Git directory lets existing source-resolution helpers identify the archive.
Comparison evidence: `/tmp/loom-view-union-784-comparison/{comparison,builds}.json`.
The temporary capture drivers were removed from the worktree after generation.
All external modules use the verified pushed requirement
`v1.10.0-alpha.9.0.20261012201303-8c7d8d711b9e` and explicit local replacement for
the corresponding parent/candidate source. Go 1.27.2 was used throughout.

Eleven designs produced 267 artifacts, each byte-identical in a second independent
candidate process. The matrix covers omitted/included array union, request reuse,
omitted/included maps, real untagged omitted/included branches, the existing
multi-service success/error union fixture, both recursive field orders, and the
original JSON-RPC witness. All eleven candidate modules build and vet; the five
parent omitted variants fail compilation. Combined build comparison: 17.747s.

Exactly five files change, all by removing unreachable server union declarations:

- omitted, map-omitted, reuse and untagged-omitted:
  `gen/http/svc/server/types.go` in their respective generated modules;
- jsonrpc-omitted: `gen/jsonrpc/svc/server/types.go`.

Included/map-included/untagged-included, error and both recursive designs remain
entirely byte-identical to parent. Every client file is unchanged. No broad golden
updates, dependency changes or generated-file hand edits are needed.


Final focused checks after removing capture drivers pass:
`TestViewed.*|TestPreparedBodyUnionAllocationAndMembership|TestRecursiveProjectionJSONRPC`
passes HTTP in 11.927s and JSON-RPC in 9.857s
(`/tmp/loom-view-union-784-final-checks.log`). Pinned golangci-lint v2.14.0 built
with Go 1.27.2 reports zero issues for `./http/codegen ./jsonrpc/codegen`
(`/tmp/loom-view-union-784-lint-final.log`). Lint initially identified the now-dead
`hasHTTPStreamingBody` helper; deleting that unused function does not invalidate
generated-output or behavioral evidence. Gofmt and `git diff --check` are clean.
TLC repeats the package candidate's twelve-state pass; the prior negative model
and allocation-model evidence remain valid because those model sources did not
change. Generated output directories, raw logs and checker states are outside
the worktree and retained only until independent review completes.
