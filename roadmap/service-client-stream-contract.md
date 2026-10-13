# Service client stream contract — #720

Root owns this investigation in the reused, previously clean and inactive
`metadata-defaults-parent` worktree, initially at pushed `ae3a281a2`.
Readiness was independently approved without findings before implementation. The existing accepted contract is that generated transport
streams work through the generated service client and its client interceptors.
Do not turn the type assertion into a runtime error or bypass interceptors.

1. Reproduce current HTTP SSE and JSON-RPC SSE failures by regenerating the
   existing ticktock fixtures, compiling their ordinary code, then asserting
   concrete transport stream implementation of the service interface. Preserve
   exact inputs and failing method signatures. Add the same check to the existing
   JSON-RPC WebSocket streaming-payload specimen; keep ordinary HTTP WebSocket
   and gRPC method-set controls.
2. Inventory stream declaration and invocation ownership in
   `codegen/service/service_data_methods.go`, `service_interface_sections.go`,
   `service_data_interceptors.go`, `interceptor_sections.go`, HTTP
   `stream_sections_sse_client.go` and JSON-RPC `client_stream_sections*.go`.
   Reconcile server/client roles and operation names, arguments and result types.
   Distinguish existing connection/context lifecycle ownership from method-set
   disagreement; do not fold unrelated #726 context behavior into this ticket.
3. Settle one generated client operation contract consumed by service interface,
   interceptors and transport emitters. Preserve existing supported service
   operation semantics, JSON-RPC server notification/response semantics and
   transport framing. Record any public migration required by corrected method
   signatures. Inspect actual caller designs and mixed-result methods before
   declaring the design ready. No compatibility-only aliases or runtime casts.
4. Before production edits, independently review the owner proposal, current
   counterexamples and bounded acceptance. Model role/operation selection where
   it exposes cross-consumer disagreement; reuse existing lifecycle models for
   unchanged lifetime guarantees and state their limits.
5. After approval, add direct contract tests first; implement the shared owner;
   compile the generated method-set assertions and run mounted typed service
   clients with client interceptors on both SSE transports and JSON-RPC WebSocket.
   Check values, EOF/error propagation and cancellation/close, preserving ordinary
   HTTP WebSocket and gRPC controls. Regenerate checked-in ticktock fixtures.
6. Compare only affected meaningful specimens against the parent and independent
   process repeat, enumerate intentional signature/caller differences, compile/vet
   emitted modules, and run affected gates. Obtain fresh exact final-diff review,
   integrate atomically, push and close #720 only with its own acceptance proof.

- [x] Current regenerated HTTP and JSON-RPC ticktock modules compile normally.
- [x] Both static service-interface assertions reproduce incompatible `Recv`
  signatures: transport `Recv(context.Context)`, service `Recv()` plus
  `RecvWithContext(context.Context)`.
- [x] Reproduce JSON-RPC WebSocket mismatch using the existing streaming-payload
  specimen: ordinary module builds; Upload/Count/Names concrete stream assertions
  fail because the service interface requires server-only SendError.
- [x] Establish ordinary HTTP WebSocket and gRPC method-set controls.
- [x] Freeze owner/model proposal and get independent readiness approval.
- [x] Implement and validate the accepted contract.
- [ ] Obtain independent review of the exact final diff.
- [ ] Commit, push and close the proven issue.

Evidence: `/tmp/loom-720-current-evidence.json`,
`/tmp/loom-720-{http,jsonrpc}-current-mismatch.log`, and regenerated module roots
`/tmp/loom-720-current-{http,jsonrpc}`. The root source is explicitly replaced by
current main for these fixture regeneration probes. Baseline test logs are
`/tmp/loom-720-{http,jsonrpc}-baseline-{0,1}.log`. These establish the current
method-set defect, not mounted interceptor behavior or final acceptance.


Current owner trace:

- `StreamData` already records canonical receive/send names and contextual
  variants, and the service client/interceptor paths consume those declarations.
- HTTP and JSON-RPC SSE emitters independently put a context argument on Recv
  rather than emitting the recorded RecvWithContext variant. HTTP additionally
  invents a second local client interface, so its existing static assertion
  proves the wrong contract. Both need to derive their method surface from the
  service contract and retain the endpoint context for non-contextual calls.
- JSON-RPC WebSocket transport clients already emit canonical Send/Recv and
  contextual variants. `service_interface_sections.go` applies server JSON-RPC
  notification/response methods to both roles and suppresses client receive.
  Resolve role and protocol once for method-surface emission, preserve existing
  server semantics, and make actual transport implementations statically satisfy
  the generated client interface. Do not add server-response methods to clients.
- Context capture is implementation work, not permission to redesign lifecycle:
  the existing SSE reader and WebSocket owner remain responsible for cancellation,
  EOF and Close. Typed service stream lifetimes must be exercised under their
  original endpoint context; do not substitute a Background context.
- The HTTP SSE transport interface may derive its operation surface by embedding
  the service interface while retaining its existing explicit Close operation.
  Avoid duplicating the service method set. Public direct SSE calls will need the
  corrected RecvWithContext(ctx) spelling, which needs migration guidance and
  fixture regeneration. This proposal still requires independent readiness review.

The WebSocket witness is `jsonrpc/codegen/service_client_stream_readiness_test.go`;
its generated build succeeds before the three static assertions fail. Log:
`/tmp/loom-720-websocket-readiness.log` (1.614s). No production changes.

## Readiness refinement at ae3a281a2

This worktree contains only readiness sources, controls and a model. Production
files remain byte-identical to the parent. Ordinary controls now pass concrete
transport-to-service interface assertions and generated vet for server-streaming,
client-streaming with a final result, client-streaming without a result, and
bidirectional HTTP WebSocket/gRPC methods. Explicit-local-source reruns passed in HTTP 7.998s and gRPC
11.921s, including the same-method SSE/gRPC control. A separate same-method HTTP SSE/gRPC control protects shared interfaces.

### Accepted owner and implementation sequence

1. Keep `StreamData` as the operation owner. Its canonical Send/Recv names,
   contextual variants and result references already drive service clients and
   interceptor dispatch. Resolve effective transport capability in
   `buildMethodTransportData`, retaining whether the actual endpoint uses SSE
   alongside existing JSON-RPC flags; do not rediscover it in emitters. Include
   HTTP mixed-result SSE methods, not only `m.IsStreaming()` cases.
2. After direction-specific initialization, retain the client Close capability
   for SSE and JSON-RPC WebSocket in `StreamData.MustClose`. Existing transport
   cleanup must remain reachable from service clients and interceptor wrappers.
   Root accepted this as correction of existing behavior, with no new policy
   question. Ordinary HTTP WebSocket and gRPC methods retain their operation set.
3. Resolve protocol role once when constructing service stream-interface data:
   JSON-RPC notification/response/error operations belong to the server role.
   Client interfaces use the shared canonical operations, including receive and
   the retained Close capability. Preserve server-only SetView and all server
   framing, service request/reply signatures and response-ID behavior.
4. Both SSE adapters emit the two receive methods from `ClientStream`: Recv()
   delegates to RecvWithContext(savedEndpointContext), and RecvWithContext(ctx)
   runs the existing decoder/reader path with its supplied context. Keep the
   canonical streaming-result type, including mixed-result methods. No unary
   result substitution or transport-local method-set policy.
5. HTTP's client-local interface embeds the service interface rather than
   re-declaring its methods. The generated SSE constructor takes an explicit
   opening context, and the endpoint passes its original context. JSON-RPC's
   endpoint retains that context in its stream adapter. Do not derive it from
   an optional Response.Request, supply Background, or retain only a transient
   per-call context. No variadic constructor compatibility fallback.
6. Generate static concrete-to-service client-interface assertions for affected
   adapters. Interceptor wrappers derive operation names/types/Close from the
   same StreamData, forward Close and keep contextual interception intact. Do
   not bypass the existing type assertion, add server operations to clients,
   or substitute a runtime error for an impossible assertion.
7. Document regeneration and direct caller migration: SSE Recv(ctx) becomes
   RecvWithContext(ctx); Recv() uses the opening context. Direct HTTP SSE stream
   constructor callers pass that context explicitly. Typed clients expose the
   existing Close operation; callers defer it when they stop before EOF.

A same-method SSE/gRPC design exposes a shared service interface. If SSE adds
Close to it, the gRPC adapter must implement that operation too, through its
existing Close emitter. Its current CloseSend half-close semantics remain
unchanged; canceling the original RPC context remains the complete lifetime
bound. This ticket must not claim to implement gRPC per-operation cancellation
or full-close semantics. The ordinary gRPC-only controls must remain byte-identical.

### Consumer and generator inventory

| Boundary | Current consumer | Required correspondence |
| --- | --- | --- |
| Method/transport analysis | `service_data_methods.go`, `MethodTransportData` | Classify actual SSE once, resolve direction and Close |
| Service declarations | `service_interface_sections.go`, `client_jennifer.go` | Correct role; canonical stream interface returned by typed Client |
| Client interceptors | `service_data_interceptors.go`, `interceptor_sections.go`, `interceptor_sections_stream.go` | Same operations and Close delegation; contextual dispatch retained |
| HTTP SSE adapter | `stream_sections.go`, `stream_sections_sse_client.go`, `client_jennifer.go` | Embedded service contract, explicit captured context, paired receive methods |
| JSON-RPC SSE adapter | `client_sections.go`, `client_stream_sections.go` | Retained opening context, paired receive methods and static assertion |
| JSON-RPC WebSocket adapter | `client_stream_sections_websocket.go`, HTTP `websocket.go` | Existing Send/Recv names retained; static assertion against corrected role |
| Ordinary adapters | HTTP `websocket.go`, gRPC `service_data_stream.go` and `jennifer_stream.go` | Existing names/types unchanged; shared SSE/gRPC Close addition explicitly compared |
| Fixtures/callers | Both ticktock fixtures and SSE generator harnesses | Regenerate; update direct receive/constructor calls mechanically |

Actual checked-in callers include ticktock EOF/close tests, HTTP field-presence,
projection/viewed-result and result-type harnesses, plus JSON-RPC mixed-result
and result-type harnesses. They currently call transport Recv(ctx), bypassing
the typed service client; all touched caller migrations must remain exercised.
`docs/interceptors.md` already documents SendWithContext/RecvWithContext as
per-message interception. The consumer skill and HTTP guide and interceptor guide need the
specific SSE migration and cleanup contract. Inspection of the local Loom-MCP
consumer found `progressive_discovery/design`'s `stream_chunks` contract, but
its model Streamer.Recv APIs are unrelated; do not mechanically rewrite those.
The real compiled ticktock applications are the primary adoption controls.

### Evidence, model and scope boundaries

The direct cleanup witness fails at all three affected transports because
`ClientStream.MustClose` is false. The model under
`codegen/service/tla/client_stream_contract` reproduces all three method-set
failures, signature-only cleanup loss and a Background-context workaround;
the owned proposal passes 57 states. This checks operation ownership, not Go
execution or network liveness. Existing SSE and WebSocket lifetime owners and
models remain unchanged; their assumptions and limitations are linked in the
new model README. #726 remains separate.

A separate mixed-result interceptor defect was found: validation and accessor
analysis select unary Result rather than StreamingResult. The standalone
readiness witness `/tmp/loom-720-mixed-interceptor_test.go` fails validation for
an actually authored streamed `chunk` attribute (0.511s). Report draft:
`/tmp/loom-720-mixed-accessor-report.md`. Root independently reproduced and reviewed it, then filed
[#859](https://github.com/CaliLuke/loom/issues/859). #720 includes mixed-result
stream method-set/type preservation, but must
not claim to repair that independently defective attribute-access contract.

### Bounded implementation acceptance

- Direct operation tests: three affected transports; ordinary four directions;
  role separation; canonical mixed-result type; Close preservation through the
  service interface and wrapper; constructor and contextual invocation shape.
- Mounted typed service clients, with and without client streaming interceptors:
  HTTP SSE, JSON-RPC SSE and JSON-RPC WebSocket. Assert observed values and
  interceptor mutation/count/context identity; original endpoint cancellation,
  explicit canceled/deadline operation context, clean EOF/protocol errors,
  early Close and Close through wrappers. Keep server JSON-RPC notification,
  response and error behavior and shared-connection peer-stream isolation.
- Bounded base/candidate/repeat output comparison: 16 complete modules. Reuse
  both ticktock fixtures, HTTP MixedResultsDSL, JSON-RPC mixed-result specimen,
  JSON-RPC streaming-payload specimen, existing bidirectional WebSocket specimen,
  HTTP primitive/collection SSE specimen, four ordinary HTTP WebSocket controls,
  four ordinary gRPC controls and one same-method SSE/gRPC control. Compile/vet
  every distinct candidate module once. Independent-process repeat checks bytes,
  not duplicate compilation. Preserve parent failures as failures, not missing
  output. Reuse baseline evidence where exact inputs/source/toolchain match.
- Intended differences: affected service client method sets (including Close),
  HTTP SSE constructor context and embedded local interface, paired SSE receive
  methods/context field, adapter assertions, client-wrapper Close delegation,
  migrated direct caller spellings, and gRPC Close only for a shared SSE method.
  Server interfaces/operations, HTTP/gRPC ordinary stream adapters, wire formats,
  proto files and unary behavior must remain unchanged. Any extra difference
  requires explanation and an affected regression before review.
- Run the affected service/HTTP/JSON-RPC/gRPC direct and selected generation
  tests, selected existing lifecycle tests, formatting/lint, documentation
  checks and all seven new model configurations. No full exported-design audit.
- Freeze exact final diff, source/evidence hashes and comparison results for a
  fresh agent. No production implementation until readiness approval; no commit
  until exact final review has no unresolved findings.


Additional current evidence: mixed-result HTTP and JSON-RPC modules build normally,
then fail concrete-to-service assertions with the canonical Event/Item type on
both sides and only the Recv argument mismatch. Logs:
`/tmp/loom-720-mixed-method-readiness-local.log`. The explicit-local ordinary
controls are `/tmp/loom-720-ordinary-controls-local.log`; they supersede initial
HTTP helper runs that used the repository's default pinned runtime source.
All final acceptance runs must set LOOM_DIR to the candidate worktree.
The production owner paths in the ticktock regeneration source remain identical
to the readiness parent (main diff against ae3a281a2 is empty for
`codegen/service`, `http/codegen`, and `jsonrpc/codegen` at capture).

## Implementation and validation

The worktree advanced safely to pushed `4710891ff` before implementation; all
14 readiness inputs retained their recorded hashes. Historical readiness
evidence above remains a parent failure record, not a claim about the repaired
generator. The implementation derives operation names and cleanup capability
from StreamData, restricts JSON-RPC server operations to the server role, embeds
the shared service interface in HTTP SSE, and captures the original endpoint
context in both SSE adapters. Static adapter assertions catch future drift.

Both ticktock fixtures now declare a client streaming interceptor with read/write
access to their event field. This makes the generated typed Client and wrapper
part of their mounted regression surface. The CLI example wiring supplies nil
interceptors; generated declarations and CLI parameter additions are expected
fixture enrichment. Test-module assembly also emits the existing service
ClientFile so compile/runtime checks exercise the actual public typed client.
JSON-RPC's existing bidirectional specimen has the same interceptor enrichment.

Final focused service/HTTP/JSON-RPC/gRPC checks passed in 0.393s/33.990s/19.815s/
17.114s. Both checked-in fixture suites pass with the race detector and vet
(1.964s HTTP, 1.901s JSON-RPC). Mounted WebSocket accessor and lifecycle checks
pass, including two logical streams sharing a connection. Scoped lint reports
zero issues. All seven model configurations were rerun: six expected invariant
counterexamples and the owned model's 57-state pass. This does not claim gRPC
per-call cancellation (#726), mixed-result interceptor accessors (#859), or
stronger network liveness than the existing runtime owners provide.

The 16-module comparison contains 231 generated files per source. Independent
process regeneration is byte-identical. Exactly 23 files change: service client
interfaces and affected SSE adapters/endpoint constructors; the two ticktock
client wrappers gain Close; RPC WebSocket adapters gain static assertions; and
the shared SSE/gRPC client gains its existing CloseSend implementation. All
server adapters, ordinary HTTP WebSocket and gRPC-only modules, protobuf and
wire representations remain byte-identical. The exact path list and per-file
hashes are in `/tmp/loom-720-artifact-hashes.json`; the generated diff is
`/tmp/loom-720-generated.patch`. Checked-in fixture gen trees exactly match the
comparison candidates. The interceptors are equal inputs on both sides.

Every candidate module builds and passes vet. Parent compilation passes for
15 modules; the enriched JSON-RPC bidirectional specimen fails in its existing
wrapper because the erroneous client interface omits SendWithContext and
RecvWithContext. That is another concrete manifestation of #720, not an omitted
comparison. The 14 direct package generators use an overlay containing the nine
production owner files from `4710891ff`. Fixture CLI regeneration spawns its own
generator and bypasses that overlay, so the two fixture baselines instead use an
archived exact parent source tree; the earlier ineffective CLI-overlay baseline
is superseded by `/tmp/loom-720-archived-base-results.json`.

Final evidence includes `loom-720-final-affected.log`, `loom-720-fixture-results.json`,
`loom-720-module-build-results.json`, `loom-720-final-model-results.json`,
`loom-720-scoped-lint.log` and `loom-720-static-gates.json` under `/tmp`. Formatting,
file-size, NameScope, JSON-v2 and documentation gates pass. Both changed fixture
root packages also pass lint. An exploratory lint of their entire example
command trees reports eight existing diagnostics in six unchanged files; these
fixture commands are excluded by the repository's hand-written source lint
gate. Their byte-identical parent receipt is
`/tmp/loom-720-fixture-command-lint-baseline.json`. This is not reported as a
fully green repository-wide lint run.

The mounted RPC fixture now also sends a request ID and observes a post-event
protocol error through the typed client and its wrapper, retaining safe error
text. Notification-only calls correctly cannot receive JSON-RPC error responses.
