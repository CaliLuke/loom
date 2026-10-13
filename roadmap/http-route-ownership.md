# Effective HTTP route ownership — #781

Status: implementation complete on `1b9833038`; bounded acceptance passes and
exact final review is pending. Original readiness and phase-gap evidence retained.

## Contract and scope

Each server process has one physical owner for a method and effective path.
Path parameter names do not distinguish route slots; static suffixes, literal
segments and catch-all behavior do. A JSON-RPC service intentionally multiplexes
its logical methods through one dispatcher. Different services are not one owner.
The documented Server/Services contract defines cohosting: disjoint servers may
reuse paths. Omitted server selection means all services, as it does today.

The report's three original cases must become design errors before codegen.
Do not add runtime last-writer detection alone or separate HTTP/JSON-RPC checks.
OpenAPI retains its stronger document-template and operation-ID constraints.

## Ordered work

- [x] Read #781, expression evaluation/path expansion, Server/Services docs,
  HTTP and JSON-RPC mount renderers, CORS mounts and the existing document-identity
  model. Confirm routes are prepared before root validation.
- [x] Add and run the three original failing DSL cases plus cross-transport and
  wildcard-equivalence witnesses. Keep different-method, shared JSON-RPC dispatch
  and disjoint-server controls.
- [x] Reconcile physical mounts in `http/codegen/server_sections.go`,
  `jsonrpc/codegen/top_level_sections.go`, their CORS paths and static-file
  mounting. Identify synthetic GET/OPTIONS claims and deliberate shared owners.
  Record a single source of route identity/ownership that consumers can use;
  do not duplicate generated mount rules in a validator.
- [x] Extend the existing document-identity model only for server scopes and
  deliberate shared dispatch. Reproduce missing admission and reject blanket
  API-global or per-endpoint uniqueness alternatives. Model assumes route
  extraction; direct/generated checks establish that correspondence.
- [x] Freeze owner proposal, exact test/model inputs and bounded acceptance for
  independent readiness review. Settle any genuinely unresolved public behavior
  before production changes; routine implementation details need no new policy.
- [x] Implement the approved shared owner and remove superseded route inference.
- [x] Run focused expression/routing tests, affected generator sections, one
  existing mixed HTTP/JSON-RPC specimen and disjoint-server control. Check actual
  mounted dispatch, deterministic diagnostics, invalid CLI generation, parent
  and candidate bytes, compilation and affected OpenAPI specimens. No corpus audit.
- [ ] Independently review the exact final diff, deliver one atomic #781 commit,
  push, close with evidence, update the recovery checkpoint and clean task output.

## Readiness boundaries

Authored exact/wildcard duplicates and automatic route claims must be distinguished
from overlapping routes with deterministic router precedence. Do not reject a
literal child route merely because a parameter or catch-all also matches it.
Invalid parent/path graphs must not make this admission recurse indefinitely.
Server grouping must follow the documented expression contract, not host URL
string guesses. Repeated logical JSON-RPC route declarations must not create
false duplicate errors. Static-file and CORS claims need correspondence evidence
before they can be included or explicitly deferred as a separate defect.

Existing OpenAPI document checks reject equivalent templates even across distinct
verbs; this is stricter than physical mux identity. Preserve that documented
export contract rather than merging the two predicates accidentally.

## Initial reproduction

All five conflict cases fail on `52aa2a64a`: same-service HTTP, cross-service
HTTP, cross-service JSON-RPC, HTTP versus JSON-RPC, and equivalent wildcard names.
Different methods, one shared JSON-RPC dispatcher and disjoint server processes
pass. `/tmp/loom-781-readiness.log` records this direct parent run (0.439s).
The shared document-identity model is extended with physical owner/server claims;
its existing stronger OpenAPI identity rules remain unchanged.

## Resolved owner proposal

Own physical registration in `expr`, after all expression `Prepare` methods and
before code generation. A single resolver returns ordered per-service mount
records: effective path, canonical slot key, verb, physical handler identity,
source endpoint/file-server binding, and CORS eligibility. Handler identity is
separate from service identity: HTTP endpoints and file servers are distinct;
all logical methods of one JSON-RPC transport service share its dispatcher.
HTTP and JSON-RPC mounts for the same service are still distinct handlers.

Root validation consumes that plan for admission. HTTP and JSON-RPC analysis
retain the same records for emitted mount functions, mount metadata and CORS
method enumeration. Endpoint route data still owns URLs/parameters and logical
OpenAPI operations; it must not independently infer synthetic physical mounts.
Use one pure resolver, not a validation-only approximation or a global cache.
Invalid roots return contextual design errors before dereferencing empty routes.

Preparation must make **both service defaults and file paths** phase-stable.
`HTTPServiceExpr.FullPaths` currently returns the API base immediately when
`Paths` is empty; its later `Finalize` inserts `["/"]`, which instead activates
Parent/canonical-route expansion. Move that default into `HTTPServiceExpr.Prepare`
before resolving paths. Preserve explicit service bases, the parent's canonical
endpoint base, absolute service paths and absolute endpoint bypass exactly.
The new five-case direct phase witness compares the same route before/after
finalization: the omitted child base currently changes `/api/child` into
`/api/parents/{id}/child`; the four explicit/absolute controls are stable.

Static files also currently replace authored `RequestPaths` with expanded paths
in `Finalize`. Move that normalization to their preparation owner, make
validation inspect those same prepared paths, and remove superseded finalization
expansion for services/files. The route resolver then reads the same effective
inputs at validation and after finalization. It must not prefix files twice;
retain existing file expansion, including absolute paths and service bases.

Graph safety belongs inside the shared path-resolution boundary, **before any
recursive expansion**, not only in root validation. Endpoint preparation calls
`inheritCanonicalEndpointTransport`, then `canonical.Prepare`, then
`canonical.PathParams`, which calls `RouteExpr.FullPaths`. The existing endpoint
`prepared` flag prevents recursive endpoint preparation but does not protect
recursive service/route path expansion. Thread a visited-service set through one
shared service/route resolver and produce a structured failure for missing
parents/canonical paths or cycles. Existing `FullPaths` queries must return no
paths for that invalid resolution rather than recurse or synthesize a route;
admission reports the same contextual resolution failure. Do not add a second
parent-graph predicate just for validation or defer the guard until codegen.
Absolute route bypass still resolves directly; invalid authored Parent graphs
remain design errors through their own graph admission. All early preparation
callers must use the guarded resolver, including canonical path-parameter
inheritance. Root validation must never receive a partially fabricated plan.

`Server.Services` remains an admission scope: absent servers or an empty service
selection mean all services. Two different services cohost if any declared
server selects both; host strings are not identity. A service always cohosts with
itself even when no server selects it, because its standalone Mount still emits
all its handlers. Sort diagnostics by service, verb and canonical path and name
both owners. Keep literal text, suffixes, trailing slash and catch-all kind in
slot keys; remove only wildcard names. Do not reject deterministic overlapping
routes such as a literal child below a catch-all.

## Complete physical consumer map

| Producer | Physical records and consuming paths |
| --- | --- |
| HTTP endpoint | Each effective route maps to its endpoint handler; `renderEndpointMounts`/per-endpoint mount functions consume the plan. Duplicate records for the same handler and identical authored path can coalesce, but differently named parameter bindings must not silently merge. |
| JSON-RPC ordinary or WebSocket service | One dispatcher owns the common prepared route. Logical methods repeat this identity, not independent claims. `jsonrpcServerMountSection` reads the plan. |
| JSON-RPC mixed ordinary/SSE | The dispatcher owns its authored POST and synthetic GET on each effective route. The same transport classification already used for handler selection supplies this rule; do not retain a second classification in CORS. |
| JSON-RPC SSE-only | Authored POST routes plus GET only for `events/stream`; an ordinary named SSE method does not imply GET. |
| Static files and redirects | GET for a file; a directory contributes its root-with-slash GET and catch-all GET. Move expansion from `renderFileServerBody` and the codegen path-prefix inference to the shared expression plan. Keep filesystem stripping metadata separate from registration paths. |
| CORS | Derive OPTIONS/method sets from CORS-eligible endpoint/dispatcher plan records. Static file mounts currently have no CORS wrapping/preflight; keep that contract unchanged. |

For HTTP CORS, merge methods within one transport service and canonical path.
An explicit local OPTIONS endpoint is the owner of its preflight wrapper, not a
second mount. Its actual path/bindings take precedence over an automatic OPTIONS
spelling. Without an explicit endpoint, one automatic preflight owner serves
that service/path. Two independent services' OPTIONS claims conflict even when
their ordinary verbs differ: they can carry distinct service or runtime CORS
policies. Reject the collision; do not merge their policies or choose a winner.
The existing `corsPreflightRoutes` and `jsonrpcCORSRoutes` may remain formatting
projections over resolved records, but their endpoint/transport inference loops
must disappear. No global CORS policy rewrite is needed.

OpenAPI keeps its stricter document-wide template and operation-ID predicate.
In particular, disjoint physical server ownership does not waive a conflict in
one exported document. Generator acceptance can suppress OpenAPI for a specimen
whose purpose is solely disjoint physical mounts, with separate existing IR
checks proving that export rejection remains intact.

## Expanded evidence and bounded implementation acceptance

`expr/http_route_ownership_test.go` now covers twenty cases: thirteen admitted
collisions fail as expected on current production, and seven controls pass
(`/tmp/loom-781-expanded-readiness.log`, 0.484s). Added witnesses cover unselected
same-service mounts, omitted server selection, both synthetic GET branches,
independent preflights, foreign explicit OPTIONS, static files and directory
root expansion. Controls retain ordinary SSE without GET, local explicit OPTIONS,
literal/parameter precedence and literal children of directory catch-alls.
An initial fixture used Files inside HTTP incorrectly; that scratch run was
corrected to documented service-level Files before recording this receipt.

The existing Lean model now also checks unselected same-service ownership,
synthetic-claim omission, local explicit OPTIONS sharing, synthetic GET conflicts
and catch-all/literal distinction. All required theorem axiom audits pass via
`http/codegen/openapi/internal/ir/tla/document_identity/check.sh`; the model
assumes correctly extracted claims and does not prove extraction or router
matching. It is not a general route-language equivalence checker.

Before final implementation review:

1. Turn all thirteen admission reds green with owner-specific diagnostics;
   retain all seven controls. Retain the five direct phase cases and add invalid
   parent-graph controls through actual DSL preparation (bounded subprocess if
   needed to expose the parent stack overflow safely) at the expression resolver boundary. Assert plan records
   directly for multiple base paths, CORS inheritance/runtime policy, static
   directory root/catch-all, and same-owner repeat elimination.
2. Reuse existing HTTP server/CORS/file-server sections and JSON-RPC CORS tests.
   Their rendered registrations and preflight methods must equal the resolved
   plan. Preserve HTTP explicit OPTIONS fallback, ordinary SSE without GET,
   mixed and events/stream GET, and WebSocket GET transport selection.
3. Use two bounded generated modules: one valid combined HTTP/file/JSON-RPC mixed
   specimen with distinct paths and local CORS, and one disjoint-server specimen.
   Compile and test actual dispatch/preflight ownership, including directory root
   and literal child. Compare physical parent/candidate and independent repeated
   generations. Existing accepted output should remain exact unless the plan
   intentionally removes duplicate same-owner registrations or stabilizes their
   ordering; list every such difference. No full fixture corpus.
4. Run an original duplicate design through the CLI and check rejection before
   output replacement. Reuse current OpenAPI identity cases to prove its stronger
   export predicate remains. Validate/lint only rendered OpenAPI specimens that
   actually change.

Exit: every generated physical mount comes from the same resolved plan used by
admission, cohosting scopes are correct, and no renderer or CORS enumerator
reconstructs its own dispatcher/synthetic-route identity.

At readiness, the seven valid controls also passed independently under race
(1.548s); no admission semantics were changed by those readiness files.

## Readiness correction R1: preparation correspondence

Root review identified the omitted-service-path phase gap and the earlier
canonical-inheritance recursion seam. The proposal above now moves service
path defaults to Prepare alongside file normalization, and puts cycle safety
inside shared path resolution used by preparation itself. A root-validation-only
graph guard is explicitly rejected. `TestHTTPRouteOwnershipPreparationPhase`
checks inherited child bases, explicit root/nested bases and both absolute
bypass forms. The omitted-base case fails on unchanged production; all four
controls pass. No production changes or original model reruns are required for
this proposal/test correction; existing claim-level model assumptions are
unchanged. Actual invalid-parent DSL preparation remains an implementation gate.

## Implementation and final acceptance

`expr.HTTPServiceExpr.MountPlan` now owns ordered endpoint, file, dispatcher and
preflight registrations. Root admission and HTTP/JSON-RPC analysis consume it.
The old mount/CORS inference loops and late service/file finalization are removed.
`http_paths.go` protects early canonical parameter inheritance and supplies the
same path errors to admission. Invalid cycles, missing parents and missing
canonical routes produce design diagnostics rather than partial paths.

Static-file correspondence exposed one additional manifestation of the same
mount owner defect: the previous service Mount invoked a helper that registered
**every** base path once for each prefix wrapper. Two bases therefore registered
each physical slot twice and the first base used the second prefix. The plan now
binds each physical registration to its own strip prefix. Exported file mount
helpers still accept one caller-provided handler and register all their routes.
Mount metadata lists actual directory root/catch-all and automatic preflights.

The original twenty admission cases, five phase controls, three actual cyclic
Parent graphs and direct binding/coalescing controls pass. Full expression/DSL
race checks passed (6.689s/1.316s); after mechanical lint extraction, focused
expression, HTTP, JSON-RPC and generated-module race checks passed
(1.534s/5.980s/3.509s/6.148s). Scoped pinned lint reports zero issues. The unchanged
claim model's 26 theorem axiom audits are reused from approved readiness; its
extraction limitations remain explicit. Existing stronger OpenAPI operation
identity tests pass (0.485s), including equivalent templates across methods.

Physical parent generation uses a `git archive` of `1b9833038`, with only the
exact test inputs added. A local `git init` supplies repository discovery for the
existing helper; it does not substitute production source. The parent combined
module builds/vets and its race test reproduces four double registrations plus
two first-base 404s. Both candidate modules build/vet and mounted race checks
pass. Independent repeated generation is exact: 54 artifacts per stage, with
only `combined/gen/http/web/server/server.go` differing from parent. Its changes
are the per-prefix mounts and complete mount metadata. The disjoint-server
module is unchanged. A final generation after lint extraction reproduces all
54 candidate bytes; identical compiled artifacts reuse their build/vet evidence.

All three checked-in transport fixture apps were copied and regenerated through
the actual CLI at parent, candidate and an independent repeat: all 55 artifacts
remain byte-identical to each other and to checked-in files; parent/candidate
apps build and vet. Their unchanged OpenAPI artifacts do not require a new
validator run. Six existing file mount goldens were regenerated with the
sanctioned test update flag. They adopt per-route registration and move the
fixture's directory to `/static` because its former synthetic root collided
with its explicit root file. The explicit collision remains a rejection test.
An unrelated body-validation fixture now gives its five methods distinct paths;
its original body assertions remain unchanged. Existing CORS, mount-branch and
multi-base static-file runtime controls pass.

An actual duplicate-design CLI invocation now fails during DSL evaluation and
preserves a sentinel in the existing gen directory. Its first scratch attempt
stopped earlier on missing module tidy; that failure is retained and superseded
by the successful rejection check. The initial parent capture lacked Git
metadata, failed before generation, and was similarly corrected and retained.
No unrelated JSON-RPC stream snapshot changes are included.

Evidence: `/tmp/loom-781-module-ledger.json`, `loom-781-fixture-ledger.json`,
`loom-781-final-refactor-byte-receipt.json`, `loom-781-invalid-cli.json`,
`loom-781-refactor-race.log`, `loom-781-expr-final-race.log` and
`loom-781-lint-final.log`; the final manifest records exact source, inputs,
commands, toolchain and artifact hashes for independent review.

## Final review correction R1: shared-plan test input

The existing forced-GET JSON-RPC dedup case changed old endpoint analysis data
after `MountPlan` was resolved, so it no longer exercised that boundary. It now
changes the prepared expression route before analysis, asserts one dispatcher
GET in the resolved plan, and checks the rendered mount has exactly one GET and
no POST. No production code or generated fixture output changes. Focused route
and CORS checks establish the corrected test correspondence.
