# Service error constructor identity — #734

Implemented against pushed parent `cdc581066`, including #867. Distinct wire error names retain distinct generated constructors without colliding with other package declarations. The existing service package symbol owner now records the factory identity once. Independently reviewed, pushed as `555d13306` and #734 confirmed closed.

## Sequence

1. [x] Read the issue and trace error collection, method references and constructor emission.
2. [x] Reproduce the original name collision on current source: both default and custom-field `foo_bar`/`FooBar` errors produce duplicate `MakeFooBar` entries.
3. [x] Establish one emitted error-constructor identity shared by service declarations and every method reference. Account for service/API/method error sources, non-emitted custom errors and repeated references.
4. [x] Extend the package-symbol model to expose concatenation, uncached per-use allocation and type/factory collisions; compare a cached shared allocation owner.
5. [x] Freeze bounded acceptance: original error pairs, authored suffix/type collisions, repeated method use, ordinary output stability, generated compile and wire error identity across relevant transports.
6. [x] Independently review readiness, implement the accepted owner, compare bounded parent/candidate/repeat artifacts and run focused checks.
7. [x] Independently review the exact final diff, commit, push and close #734.

## Evidence and boundaries

The original `buildErrorInitData` concatenated `Make` with Goify of the wire name. It is called during service error collection and again for each method. A simple uncached Unique call would allocate different names for declaration and reference; independent type allocation must also be accounted for. `serviceSymbols` already owns fixed and method-derived package declarations before nominal type collection. Preserve its independent method/member and transport namespaces.

The current direct probe `/tmp/loom-734-probe-overlay.json` leaves the repository unchanged and fails both constructor cases; log `/tmp/loom-734-probe.log`. The issue report is preserved at `/tmp/loom-734-issue.json`. Existing `codegen/service/tla/package_symbols` explicitly excludes error constructors, so its passing models do not prove this extension.

#699 duplicate authored error names, #770/#786 interceptor declarations, #736 protobuf declarations and #856 import aliases remain separate. Do not reject distinct wire names merely because their preferred Go identifiers collide. No public policy choice is needed for unique generated identities.

## Implemented ownership

Use the service package NameScope's cached allocation for an error-factory key
tagged separately from nominal type and service keys. The key is the exact wire
error name within this service, matching existing declaration deduplication.
Reserve all emitted factories after existing fixed/method package symbols and
before nominal type analysis. Traverse service errors then each method's errors
in their existing design order. API errors inherited into those evaluated lists
are covered there; `mappedServiceErrors` explicitly includes only non-default
custom errors, which do not emit factories. Preserve current precedence for
repeated wire names; redefining identical names remains #699.

One private helper owns emission eligibility (`expr.IsDefaultErrorResult`), the
preferred `Make` spelling and the cached key. Both early symbol allocation and
`buildErrorInitData` use it. Repeated method analysis retrieves the same recorded
identity without reserving another name. Do not allocate constructor names for
custom user error types that emit no factory; represent their absent factory as
an empty `ErrorInitData.Name` and document that field accurately. Search found
constructor emission and tests as the current consumers of that field; method
error documentation uses `ErrName` and `TypeRef`. Check this boundary explicitly.

Keep this helper in a small service-private owner file. Do not add error naming
policy to transports or change wire error names, error fields, status mappings,
remedies or nominal type placement. Existing noncolliding factory names remain.
Factory/type collisions may rename a nominal type consistently through NameScope;
regenerate and update application references to the emitted Go names. Types in
external packages must not be renamed for a local factory merely by sharing a
preferred name. The active #867 work also extends `package_symbols.go`; integrate
its reviewed change before finalizing this ticket's parent and naming evidence.

Rejected alternatives: raw concatenation duplicates declarations; a separate
factory scope permits collisions with service types; per-use Unique breaks
declaration/reference agreement; allocating from nominal type names couples
public factory identity to unrelated type allocation order. Rejecting valid wire
names conceals the deficient package declaration owner.

## Bounded acceptance

| Specimen | Obligation |
| --- | --- |
| Original six pairs, repeated references | Distinct constructors, exact shared method/declaration identity; default and custom-field ErrorResult forms |
| Authored `MakeFooBar`/suffix types plus factory suffix | Shared package occupancy; every emitted type/factory declaration distinct; generated compilation |
| API/service/method sources and custom error control | Existing deduplication/precedence and inherited mappings; no fictitious factory reservation for custom user errors; external package names preserved |
| Existing ordinary error/remedy specimens | Byte stability and unchanged wire names, error properties and status mapping |

Use at most four generated groups, reusing existing ordinary/remedy and inherited
API fixtures. All six spelling pairs need direct seam tests; one representative
pair and suffix collisions need generated compile tests. Run parent/candidate/
independent-process repeat and record all actual output differences. Compile/vet
the candidate modules; mounted HTTP error routing proves both colliding wire
names retain their distinct statuses. Unchanged transport artifacts and scoped
existing JSON-RPC/gRPC error controls establish their boundary without a full
transport matrix. Expected output changes are collided factory declarations and
Go type references displaced by their new reservations, plus accurate absent
factory metadata; ordinary errors and wire specifications must remain unchanged.

## Readiness evidence

`TestErrorConstructorIdentity` fails all six pairs on the parent. Its repeated
method-reference assertions are controls for the future uncached-allocation
failure. `TestErrorConstructorSharesPackageScope` fails on the actual method
payload type `MakeFailure`; an initial assertion over nested `userTypes` was
vacuous because root payload types are recorded separately, and was replaced
before recording this result. Direct suite: 0.472s. The generated original
custom-field specimen fails Go build with duplicate `MakeFooBar`; the ordinary
control builds and vets (combined 1.677s). Local generation explicitly sets
`LOOM_DIR` to this source checkout.

`ErrorConstructorIdentity.tla` reproduces raw concatenation failure in 5 states,
separate-scope package collision in 13 distinct states, and uncached reference
drift in 25 distinct states. The shared cached owner passes with 50 generated/40
distinct states, deadlock checking enabled. It models three error identities,
two nominal names, suffix collisions and repeated references. It does not prove
the source inventory, Go rendering, arbitrary namespace size or transport error
dispatch. Direct and compiled checks supply those implementation obligations.


## Final acceptance evidence

Readiness was independently approved without findings in
`/tmp/loom-734-independent-readiness-verdict.json`. The implementation uses
`errorConstructorKey` and `errorConstructorName` for both early reservation and
subsequent error records. No transport renderer independently selects a name.
The #867 endpoint-role allocation remains intact and precedes factory allocation.

The four generated groups produce 61 artifacts each. Base, candidate and a
separate-process candidate repeat are recorded in
`/tmp/loom-734-integrated-artifacts.json`; candidate and repeat match exactly.
They also match the earlier pre-#867 artifact comparison byte for byte. All eight
OpenAPI JSON/YAML artifacts are unchanged, so no schema validation obligation was
introduced. All four candidate modules build and vet; the collision module also
passes its mounted HTTP race test, preserving `foo_bar`/`FooBar` headers, distinct
409/400 statuses and one service invocation. Integrated run: 5.532s. The original
base collision and namespace modules fail compilation; base ordinary and external
controls build and vet.

Exactly ten generated Go files differ:

- The collision service changes only its second factory to `MakeFooBar2`.
- The namespace service records `MakeFooBar`, `MakeFooBar2`, `MakeFooBar22`
  factories. Its authored types become `MakeFooBar3` and `MakeFooBar23`. Nine
  service/client/endpoint and HTTP client/server files change their declarations,
  references, derived conversion names, comments and declaration ordering to match.
- Ordinary and external groups remain byte-identical; routes, specifications and
  wire error names retain their existing values.

Direct identity, absent-factory, external-name and API/service/method-source tests
pass (0.595s after integration). The full affected service package passed before
integration (31.707s), covering existing error/remedy goldens; unchanged integrated
artifact bytes and focused checks reuse that evidence. Existing gRPC shared-error
message controls pass (0.500s) and JSON-RPC public/mapped/SSE error projection
controls pass (0.505s). Scoped service, generator and explicit testdata lint passes.
No exhaustive exported-design sweep was used.

Bounded consumer inspection found ordinary default error names in Loom-MCP registry
contracts, custom errors in the HTTP quality fixture and no errors in the quickstart
control. No collided factory/type references appeared in that inspected sample;
this is not a complete consumer audit. Applications with collisions must regenerate
and use emitted Go names, as documented in the public error/code-generation guides
and consumer skill. Duplicate identical wire names remain #699.

Models and original negative runs are unchanged from the reviewed readiness packet.
The model README maps the supported design to implementation and regression tests.
Temporary generated modules and model outputs are retained for independent final
review and will be removed after preserving inputs and concise evidence.
