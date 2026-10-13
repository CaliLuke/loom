# HTTP and JSON-RPC server member identity — #700

Readiness evidence was collected on pushed `af21e5098fcaba0281c2b6bc0c8d588c105a5bbc`.
The readiness packet targeted `a7b09d5a8d2f1900d0a4b50f8af3bb78e245da24`,
which added #723 optional diagnostic reporting; direct member checks confirmed
the unchanged naming defect on that integration. The final implementation packet
targets `055c2eb0ad5e482f7b0a5cc356ee6bd33f7c36a4`, after the documentation
checkpoint. Final artifact comparisons use that exact parent as recorded below.
This ticket owns the shared HTTP/JSON-RPC server member analysis and its consumers.
Independent readiness review approved the shared owner with no findings.
Implementation, bounded verification and independent exact-final review are
complete; delivery is recorded below.

## Sequence

- [x] Read the report, canonical service method owner and all receiver consumers.
- [x] Reproduce HTTP endpoint/file-handler collisions and JSON-RPC unary,
  WebSocket, SSE, mixed and keyword failures, retaining ordinary controls.
- [x] Separate gRPC and client residuals using current compiler evidence and
  reconcile #767/#736 against their distinct owners.
- [x] Model full receiver registration before role allocation, including copied
  fields, derived-role competitors, keywords and unchanged ordinary preferences.
- [x] Record the owner/consumer inventory and bounded acceptance below.
- [x] Obtain independent readiness approval before production edits.
- [x] Implement the shared private member owner and replace reconstructed uses.
- [x] Verify exact parent/candidate/repeat artifacts, mounted behavior and docs.
- [x] Obtain independent exact-final review, integrate, push and close #700 only.

## Contract and scope

Canonical service methods, endpoint aggregate selectors, authored wire method
strings/routes and protobuf RPC names remain unchanged. A transport Server has
its own Go member namespace. Endpoint handlers, file handlers and WebSocket
endpoint roles must receive distinct legal Go identifiers within every concrete
receiver that stores them. Keep noncolliding preferred names where legal; use
existing NameScope allocation for collisions. Do not rename service methods to
avoid one transport's fixed members, reject otherwise valid designs, add a public
compatibility mode or make renderer-local suffix decisions.

HTTP and JSON-RPC already share `http/codegen.ServicesData`, `ServiceData` and
`EndpointData`; JSON-RPC receives those records through `CreateJSONRPCServices`
and `FileData`. Their server handler identity is missing from that shared model.
gRPC instead owns `grpc/codegen.ServiceData` and emits canonical RPC methods plus
`Method.VarName + "H"` fields in `grpc/codegen/jennifer.go`. Sharing a concept does
not make that renderer or data model the same owner. Its independently reproduced
residual #880 stays outside this atomic repair, as do client receiver collisions
tracked by #879.

## Analysis owner and representation

Add one package-private server-member finalization owner in `http/codegen`, called
from `ServicesData.analyze` after all endpoint and file-server records exist and
before returning analyzed data. Transport identity comes from the already prepared
endpoint IR's `IsJSONRPC`; use the existing WebSocket/SSE classifications and
CORS data for conditional declarations. Do not re-infer design semantics in a
renderer. Empty/file-only HTTP services must be safe to analyze.

Record separate semantic roles on the existing data: endpoint handler field,
optional JSON-RPC streaming endpoint field, and file-handler field. For example,
`EndpointData.ServerHandler` and `EndpointData.ServerEndpoint` may hold recorded
names, while `FileServerData.VarName` remains its recorded member name. The exact
field spelling is routine implementation detail, but role identity must be
explicit and must never alias an unrelated endpoint's derived role.

Build a fresh member NameScope distinct from package, local/import, canonical
method and wrapper-type scopes. Reserve the concrete fixed declarations first.
Allocate endpoint roles in stable endpoint order, then file-handler roles in
stable file order. HTTP and non-WebSocket JSON-RPC prefer canonical method names.
For WebSocket private fields, retain the existing valid lower-initial preference;
normalize illegal Go identifiers through the existing Go naming helper before
allocation. A keyword such as `type` cannot be passed directly to NameScope,
which only ensures uniqueness. Derived streaming endpoint preferences append
`Endpoint` to the canonical private preference, then allocate independently; do
not append the suffix to an already allocated name and assume uniqueness.

WebSocket handler and endpoint fields are copied unchanged from Server into the
service stream receiver. Allocate these shared roles against the union of both
receivers' fixed namespaces. This is one semantic role with two physical homes;
independent per-receiver allocation would require an unnecessary translation.

Keep the fixed-member inventory in the analysis owner, with conditional membership
matching actual emission. Direct AST completeness checks must compare the emitted
concrete fields and methods so new renderer declarations cannot silently escape
the scope. This change need not cosmetically rewrite unrelated fixed-member
rendering or generated stream-wrapper/package identities.

`ServicesData.FileData` rebuilds analysis when imports are aliased. Member identity
must depend on semantic receiver roles, never import aliases; reanalysis must
produce the same recorded member names. No exported helper API is needed merely
for JSON-RPC to share analysis: it consumes the recorded data it already receives.

## Concrete receiver inventory

| Receiver | Fixed declarations before field allocation | Variable roles |
| --- | --- | --- |
| HTTP Server | `Mounts`, `Service`, `Use`, `MethodNames`, `Mount`; `corsPolicy` only for runtime CORS | Endpoint handlers and file handlers |
| JSON-RPC Server, all modes | Embedded `Handler`, `Methods`, `Service`, `Use`, `MethodNames`, `ServeHTTP`, `Mount`, `decoder`, `encoder`, `errhandler`; runtime `corsPolicy` when emitted | Unary/SSE exported handlers or WebSocket private decoder/handler roles |
| JSON-RPC HTTP/mixed Server | `serveHTTP`, `handleHTTP`, `httpHandlerSpec`, `dispatchHTTP`, `encodeJSONRPCError` when their sections emit | Same recorded endpoint handlers |
| JSON-RPC SSE/mixed Server | `handleSSE` for SSE-only; `sseHandlerSpec`, `dispatchSSE`, `sendSSEError`; `streamWritePolicy` | Same recorded SSE handlers |
| JSON-RPC WebSocket Server | `StreamHandler`, `upgrader`, `webSocketConfig`, `streamWritePolicy`, `serveHTTP` | Private handler and server/bidirectional endpoint roles |
| JSON-RPC service WebSocket stream | `cancel`, `r`, `w`, `conn`; `Recv`, `Close`, `SendError`, `send`, `sendError`, `matchesMethod`, `processRequest`; emitted `Send<Method>Response` and `Send<Method>Notification` methods | The same copied handler and endpoint roles |

Conditional inventories follow the renderer branches, not a union of every
possible transport. For example, HTTP has no fixed `Handler` member and must not
rename a noncolliding authored handler merely because JSON-RPC embeds one.

## Consumer inventory

| Boundary | Source and uses | Required distinction |
| --- | --- | --- |
| Shared analysis | `http/codegen/service_data_analysis.go`, `service_data.go`, `service_data_lookup.go` | Build once after endpoint/file metadata; stable through FileData |
| HTTP declarations and construction | `server_sections.go`: `serverStructSection`, `renderServerInitBody` | Struct keys use member identity; `e.<Method>` and `<Method>Config` stay canonical |
| HTTP middleware and mounting | `serverUseSection`, `renderCORSOptionsMount`, `renderEndpointMounts`, `renderNonOptionsEndpointMounts`, `renderFileServerMounts` | Every `s/h.<handler>` uses its recorded role; route/mount metadata keeps method identity |
| JSON-RPC declarations and construction | `top_level_sections.go`, `top_level_server_init_sections.go` | Handler/endpoint keys consume records; service endpoint selectors stay canonical |
| JSON-RPC ordinary/SSE dispatch | `handler_dispatch_sections.go`, `handler_sections_sse.go` | Dispatch selects by authored wire name, calls recorded handler |
| WebSocket Server-to-stream transfer | `handler_sections.go` | Both struct keys and Server selectors consume the shared role identity |
| WebSocket stream declaration and receive | `stream_sections_websocket.go`, `stream_sections_websocket_request.go` | Private decoder calls and endpoint calls use their separate recorded roles |

Other `Method.VarName` uses are not automatically wrong. Keep canonical names for
service interfaces/endpoint aggregates, stream configurers, package helper names,
method-specific wrapper types and Send response/notification methods. Do not
perform a blind replacement. `lowerInitial` remains useful for unrelated type
names; only member reconstruction is superseded here.

## Readiness evidence

`/tmp/loom-700-readiness-complete.log` contains actual evaluated designs, emitted
AST and generated compile/vet checks. HTTP ordinary compiles; collisions include
Mount, Service, Mounts, MethodNames and a file handler competing with endpoint
Read. Runtime CORS and an authored OPTIONS route are included so their member
consumers belong to the acceptance surface. JSON-RPC unary, server-streaming,
bidirectional, SSE-only and mixed ordinary modules all compile/vet; each matching
collision module fails in the concrete Server or copied WebSocket receiver.
Direct keyword evidence fails while formatting the actual emitted private fields.
`/tmp/loom-700-direct-final.log` additionally snapshots canonical method identities
before transport analysis and checks they remain unchanged.

The final bounded model has six specimens. Current fails at 7 states, Server-only
reservation fails at 15, and owned allocation passes 30 generated / 24 distinct
states. It covers file/endpoint preference collisions, copied receivers, derived
endpoint roles and keywords; it does not prove complete renderer registration or
Go identifier normalization. Scoped lint over all three relevant test packages
passes with zero issues.

## Bounded implementation acceptance

Reuse the twelve existing HTTP/JSON-RPC generated modules: HTTP ordinary/fixed,
plus JSON-RPC ordinary/fixed for unary, server-WebSocket, bidi-WebSocket, SSE and
mixed. Add one keyword WebSocket compile specimen after the formatter failure is
fixed. Do not run the exported design corpus. Preserve parent output now and
compare candidate/fresh-process output exactly. Every intended difference must
be a recorded member declaration/access/related comment or necessary identifier
normalization. Canonical service files, clients, wire routes/method names, specs
and protobuf identities must remain exact.

Direct checks must cover complete receiver fields/methods, keyword normalization,
role-vs-role competitors (`read` endpoint field versus `readEndpoint` handler),
file-handler competition, stable ordinary names, conditional fixed names and
FileData reanalysis under a real import alias conflict. Pair the semantic record
checks with actual rendered AST assertions; do not assert only strings produced
by the new allocator itself.

Mounted tests are an implementation obligation before final review: HTTP must
route colliding handlers and the file handler to their distinct expected outputs,
exercise Use wrapping, and verify explicit CORS OPTIONS selection. JSON-RPC unary,
SSE/mixed and WebSocket must route at least one renamed handler to its intended
service method; WebSocket server/bidi checks must cover both the copied decoder
and endpoint role. Reuse existing runtime/module harnesses, keeping coverage
bounded to changed roles. Verify the two ticktock fixtures and affected goldens;
update only intentional member changes. Public docs must explain any renamed
exported Server fields and required regeneration, while preserving fixed methods.

## Separate residuals and closure boundary

- gRPC `mount` plus `mountH` still generates a MountH field/method conflict on
  current main; the mount/service/use control compiles. Evidence is
  `/tmp/loom-700-grpc-residual-final.log`. #736 concerns protobuf package message
  symbols, not this receiver, and #767 concerns interceptor field compatibility.
- HTTP `restoreResponseBody`, JSON-RPC unary `doer`, and JSON-RPC WebSocket `close`
  still fail in their Client receivers on current main. Self-contained modules
  and final command results are under `/tmp/loom-700-client-residuals`; an initial
  probe missed the post-generation tidy step and is excluded from evidence.
- Standalone report drafts and reproduction inputs are retained under
  `/tmp/loom-700-residual-reports`. The independently reviewed reports were filed
  as #879 (clients) and #880
  (gRPC), each with self-contained design snippets and pinned reproduction
  commands. They are not production work or passing acceptance for this ticket.

The existing `grpc/codegen/server_member_identity_test.go` is a durable residual
reproducer currently handed off by root. Exclude it from the eventual HTTP/RPC
#700 commit; move it with its separately tracked owner. Do not claim family closure
when only the shared server analysis repair is delivered.

## Implemented owner and verification

`finalizeServerMembers` now runs after shared transport analysis, with separate
`ServerHandler` and `ServerEndpoint` records. It reserves concrete fixed members
before allocating endpoint roles. WebSocket allocation uses the Server/stream
union; illegal lower-initial keywords pass through the existing Go naming helper.
File-handler names are no longer allocated in the package scope. They retain
NameScope's existing numeric duplicate suffix convention in their real receiver,
so ordinary repeated-file goldens remain byte-identical. Endpoint handlers use
`Handler` as their collision suffix. Canonical methods and package helper names
are unchanged.

HTTP AST checks compare every emitted fixed member to the owner's reservations.
JSON-RPC checks inspect both receiving types and derive additional adversarial
method names from their actual fixed declarations. Receiver inventory inspection
also includes `httpHandlerSpec` and both response/notification send methods in
the registration inventory. Direct controls cover ordinary spelling, keywords,
derived endpoint roles, file-only services, package-helper independence and
actual FileData reanalysis with colliding imported package aliases.

Final bounded evidence:

- Thirteen generated modules compile and vet: the original twelve plus a keyword
  WebSocket module. Mounted race tests call all authored collision methods across
  HTTP, JSON-RPC unary, SSE, mixed, server-WebSocket, bidi-WebSocket and keywords.
  Results and decoded payload markers identify the exact intended method; RPC
  responses preserve request ID 7. HTTP also checks Use, the static file, and
  authored CORS OPTIONS dispatch. WebSocket middleware is checked by invocation
  count because its upgrader does not propagate ordinary response headers.
- Parent generation restores every changed executable owner to pushed
  `055c2eb0ad5e482f7b0a5cc356ee6bd33f7c36a4` through an external Go overlay. Two
  additive data fields and an uncalled helper remain solely so candidate tests
  compile; parent analysis never calls the helper and no parent renderer reads
  those fields. The parent keyword server fails formatting after ten other
  artifacts are written, matching its direct readiness failure.
- The exact artifact ledger records 174 parent artifacts and 177 candidate
  artifacts; all 177 match independent-process repetition. Eight existing files
  change, solely concrete server member declarations, references, comments and
  their formatting. Three keyword server files become generatable. All ordinary
  outputs, canonical service files, clients and wire identities remain exact.
- Existing affected HTTP/RPC server, mounting, CORS, handler and SSE checks pass.
  No checked-in golden requires updating. Both ticktock fixture regeneration and
  build checks pass byte-for-byte. Scoped codegen lint reports zero issues.
- The readiness model is unchanged: current fails at 7/7 states, Server-only
  fails at 15/15, owned passes at 30/24. Its finite candidate spellings abstract
  NameScope suffix choices; tests prove actual spelling and renderer coverage.

The frozen final manifest records exact source, input, toolchain and evidence
hashes. Temporary modules, state and raw logs were removed after independent
review. The cleanup receipt preserves concise results, source archives and exact
reproduction inputs.

## Delivery receipt

Delivered/pushed `f0dbd2e20`; root independent final review and documentation re-review passed, ticket #700 confirmed closed. Cleanup preserves 26 source files and exact reproduction inputs.
