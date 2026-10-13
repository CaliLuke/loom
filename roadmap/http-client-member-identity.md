# Concrete HTTP and JSON-RPC client member identity (#879)

Readiness starts at pushed `dc558fcbf5dd61d59d85f0c72d3e4677a8bbf4bb`.
Independent readiness and R1 review approved the plan. Implementation parent is
`5e7ddce00f9442df71543ef37f27191b2b0455ee`, including #882 typed SSE fixture
coverage. All 21 readiness owner hashes remain exact; all 55 artifacts across
HTTP ticktock, JSON-RPC ticktock and mixedtick regenerate exactly and build/vet.

## Sequence

- [x] Identify the shared owner and separate the gRPC residual (#880).
- [x] Reproduce fixed and derived client receiver collisions with direct emitted
  declaration checks and generated compiler witnesses; retain ordinary controls.
- [x] Model current allocation, reject incomplete fixed-only allocation, and
  check complete receiver registration/allocation before rendering.
- [x] Freeze evidence, concrete owner inventory and bounded acceptance for review.
- [x] After approval, implement shared client member ownership and update consumers.
- [ ] Check final generated behavior, exact bytes, compilation and determinism;
  obtain independent final review before the single atomic commit.

## Contract and proposed owner

Canonical service methods and wire method names remain unchanged. Concrete client
receiver members need their own shared `NameScope`, initialized during
`http/codegen.ServicesData.analyze` after all endpoint roles are known and before
constructor reservations consume operation-group declarations. The owner must
record endpoint factories, request builders, per-endpoint Doer fields and mixed
HTTP result stream factories. Fixed transport members are reserved only when
actually emitted. Distinct semantic roles share the receiver namespace, including
roles derived from another authored method. Renderers consume records; they do
not append suffixes or recompute role spelling.

Proposed deterministic priority: reserve fixed members; allocate endpoint
factories in canonical service order; then allocate per-endpoint request-builder,
Doer and mixed-stream roles in that order. Use existing NameScope with meaningful
role suffixes. Ordinary names remain byte-identical. HTTP operation-group fields
are allocated after all concrete Client members and their wrapper methods reuse
the recorded member identities. Group semantic membership, path grouping and
package type identity remain their existing owners; package reservations must
consume the finalized group plan. FileData reanalysis must produce identical
member identities regardless of imported type aliases.

`EndpointInit` already represents a concrete endpoint factory and should retain
that role; add explicit records for the other derived roles rather than mutating
`Method.VarName`. `RequestInit.Name` is the request builder record. Shared CLI
command data has a canonical service method identity used by interceptor
selection; add/project a distinct transport invocation identity in the HTTP CLI
adapter, without changing service/interceptor data. JSON-RPC uses that adapter.

## Owner and consumer inventory

- `http/codegen/service_data_analysis.go`, `service_data.go`: role records and
  finalization after endpoint analysis, before `finalizeConstructors`.
- `service_data_routes.go`: request-builder initial preference and description.
- `client_jennifer.go`: Client fields/New, endpoint factories, HTTP/SSE Doer
  access and mixed stream factory selection.
- `client_groups.go`: group field reservation/initialization and wrapper methods,
  including mixed result methods; remove duplicated member inventory.
- `misc_sections.go`, `client.go`: concrete request-builder declaration/calls.
- `client_cli.go`, `cli_sections.go`: CLI transport factory selection while
  preserving canonical command names, payload builders and interceptor identity.
- `constructor_reservations.go`: use finalized HTTP group package declarations.
- `jsonrpc/codegen/top_level_sections.go`, `client_sections.go`: JSON-RPC Client
  fields/New, endpoint factory invocation, fixed shared Doer, WebSocket lifecycle
  methods (Close, IsClosed, getConn). SSE inherits shared HTTP client emission.
- Shared `FileData` reanalysis and all transport CLI renderers must consume the
  same recorded names. Stream wrapper types, codecs, paths and canonical service
  client names are separate package/receiver namespaces.

## Bounded acceptance

Use a small shared design matrix: ordinary HTTP, HTTP fixed/derived collisions,
HTTP mixed results and grouped endpoints; JSON-RPC ordinary/unary/SSE/WebSocket
fixed and derived collisions. Direct tests inspect actual emitted Client members
across files, preserve service identity and distinguish fixed members that are
not emitted. Generated modules include transport and CLI code. Current compiler
failures are evidence, not expected final behavior. Final mounted client checks
must exercise selected renamed factories and per-endpoint Doer bindings, request
builders, mixed stream access and group delegation, JSON-RPC wire method names,
WebSocket lifecycle operations and CLI selection. Cover FileData alias
reanalysis explicitly. Compare a bounded representative selection plus all three
affected checked-in fixtures (HTTP ticktock, JSON-RPC ticktock and JSON-RPC mixedtick) at parent/candidate/repeat, listing every intended
difference; compile candidate modules, race-test mounted cases, and reuse exact
repeat bytes rather than rebuilding identical output. No full corpus is needed.

This is a generated Go naming repair under the existing legal-method-name
contract, not a new DSL restriction. Users regenerate and adopt any changed
transport factory/Doer names; canonical service implementations and wire names
remain stable. #700 owns servers; #880 owns native gRPC server members; #767 is
interceptor accessor compatibility. None is claimed closed by #879.

## Frozen readiness evidence

`TestHTTPClientMemberIdentity` and `TestJSONRPCClientMemberIdentity` snapshot
canonical service names before transport analysis and inspect actual emitted
fields/methods across client files. Six adversarial designs fail with the intended
member collisions; three ordinary/control designs pass. The same nine generated
modules include service, client, server and transport CLI sources. Six fail in
Go's compiler at the reproduced members; HTTP ordinary, HTTP WebSocket and
JSON-RPC ordinary pass build/vet. The latter controls prove Close/IsClosed stay
available without JSON-RPC WebSocket lifecycle methods, HTTP Doer stays available
without a fixed Doer, and JSON-RPC ReadDoer stays available without an SSE Doer.

| Specimen | Obligation | Current result |
| --- | --- | --- |
| http-ordinary | No JSON-RPC fixed members; ordinary CLI | Build/vet pass |
| http-fixed | RestoreResponseBody; Doer roles; BuildReadRequest; suffix competition | Direct/compiler collision |
| http-mixed | WatchStream derived stream role vs authored method | Direct/compiler collision |
| http-grouped | Same roles copied into operation-group wrapper | Direct/compiler collision |
| http-websocket | HTTP lacks JSON-RPC Close/IsClosed lifecycle methods | Build/vet pass |
| rpc-ordinary | No WS lifecycle or per-endpoint SSE Doer | Build/vet pass |
| rpc-unary | Doer/RestoreResponseBody and request-builder roles | Direct/compiler collision |
| rpc-sse | WatchDoer and BuildWatchRequest role collisions | Direct/compiler collision |
| rpc-websocket | Close/IsClosed lifecycle methods and fixed fields | Direct/compiler collision |

The final readiness run is `/tmp/loom-879-readiness-final.log`; an earlier draft
SSE design omitted ServerSentEvents and was rejected during DSL validation. That
draft is not compiler evidence. The corrected SSE specimen explicitly selects
SSE and fails in the generated Client. Scoped lint reports zero issues. The
model current strategy fails at 7 states, fixed-only at 13, and owned passes
30 generated / 24 distinct. The fixed-only counterexample also catches the
allocated fixed-name fallback colliding with an authored endpoint preference.

Implementation acceptance reuses these nine modules. Add focused mounted
harnesses inside them rather than expanding the matrix: HTTP fixed/grouped must
prove each renamed factory and Doer selects its own request path/payload, explicit
request-builder identity, and group delegation; HTTP mixed must distinguish
ordinary and streaming calls. JSON-RPC unary/SSE/WebSocket must prove selected
factory-to-wire-method identity; WebSocket Close and IsClosed retain lifecycle
behavior independently of authored close/isClosed endpoint factories. Exercise
CLI selection and a client interceptor in a collision specimen, keeping the
interceptor's canonical role separate. Direct alias reanalysis and emitted
namespace inventory checks complete the owner contract. No unrelated server,
protobuf, canonical service interface, codec or wire-name change is intended.

Record final parent/candidate/repeat bytes and all intentional member-reference
changes, pair with the existing ordinary grouping goldens, and regenerate HTTP ticktock, JSON-RPC ticktock and
JSON-RPC mixedtick through their existing local-only smoke checks. Update public
regeneration guidance and the framework owner/model index only after readiness
approval. The finite model's suffix abstraction is not a promise of exact public
spellings; NameScope and rendered compile tests own those spellings.

Readiness review R1: the empty JSON-RPC callback now follows repository block
formatting. Parent requested JSON-RPC mixedtick as the third affected fixture
because its unary/SSE client roles share this owner. Incorporate the delivered
#881 fixture correction before final fixture comparison; preserve the source
parent and existing direct/model/compile evidence for unchanged readiness inputs.
These are a mechanical source correction and acceptance addition only.

## Implementation evidence and scope refinement

The concrete owner now records all roles before constructor/group consumption;
renderers and CLI use those identities. The CLI retains its existing endpoint
source reference, so it needs no duplicate transport-name field.

The first candidate compiled eight modules and exposed an independent service
Client `WatchStream` duplication in the original mixed specimen. Report #883
preserves the full minimal design and compiler witness. That original specimen
remains a direct transport-emission check; the compiled mixed specimen now uses
`watch`, `restoreResponseBody` and `buildWatchRequest`, preserving mixed dispatch
and transport fixed/builder competition without the independent service defect.
The HTTP fixed specimen adds a generic client interceptor to verify canonical
CLI wrapping. Only these two design inputs invalidate their baseline evidence;
parent/candidate comparisons are refreshed for both. No claim covers #883.

A concurrent unrelated project deleted the default Go build cache during the
first mounted test launch. Subsequent validation uses the shared reusable
`GOCACHE=/Users/luca/Library/Caches/loom-correctness-go-build`; no cache is removed
by this task. The earlier nine-module build/vet results remain valid.

## Final bounded validation

- Ten direct rendered specimens (nine compiled plus the isolated mixed-stream
  conflict) have no duplicate Client or group members. Exact emitted Client
  inventories equal their planned fixed and endpoint roles. Canonical service
  names and FileData import-alias reanalysis pass.
- All nine generated modules build and vet. Six mounted harnesses pass race:
  HTTP fixed/grouped verify renamed factories, builders and per-endpoint Doers;
  HTTP mixed distinguishes ordinary and stream dispatch; JSON-RPC unary, SSE
  and WebSocket retain exact wire method selection. WebSocket Close/IsClosed
  remain lifecycle methods. HTTP CLI uses the renamed transport factory while
  its interceptor receives canonical `RestoreResponseBody` metadata.
- Parent/candidate/independent repeat contain 127 artifacts each. Exactly 16
  client/CLI files change: receiver declarations, role uses, group delegation,
  initialization and matching comments. Service, server, path and body-type
  files remain exact; wire encoding behavior is unchanged. Candidate and repeat are byte-identical. Ordinary HTTP,
  HTTP WebSocket and JSON-RPC ordinary outputs are wholly unchanged.
- HTTP ticktock, JSON-RPC ticktock and mixedtick preserve all 55 checked-in
  artifacts at parent/candidate/repeat. Parent and candidate copies build/vet.
- Existing affected client, grouping, mixed, SSE and CLI controls pass (HTTP
  12.085s, JSON-RPC 0.388s); direct owner checks pass (0.927s/0.315s). Scoped lint
  reports zero issues. No goldens or checked-in generated files change.
- Model/config bytes are identical to the approved readiness packet, so the
  existing current/fixed-only counterexamples and owned 30/24 result are reused.

The first mounted HTTP fixed assertion expected a wire spelling in interceptor
metadata; the generated metadata correctly carries the canonical Go method name.
The corrected assertion passes in 3.079s, reusing unaffected mounted results.
The exact final manifest records all source/input/artifact hashes and commands.
Independent final review, integration, commit, push and ticket closure remain.
