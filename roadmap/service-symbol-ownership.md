# Service package declaration ownership — #694

Parent: pushed `951475f2d`. Valid user types must not collide with fixed framework
declarations in the generated service package. Allocate names through the package
owner and derive declarations/references from the same symbols. Do not add
transport-specific type renames or reject otherwise valid domain names.

## Milestones

### Milestone 1: Reproduce and identify the declaration owner

Goal

Establish current failures and the exact package-level declaration inventory before
implementation, distinguishing fixed names from dynamic constructors and members.

Acceptance Criteria

- Direct analysis and actual generated compilation reproduce the nine #694 names.
- The inventory reconciles fixed constants/types/functions, conditional security
  and streaming declarations, and separately owned dynamic symbols.
- A bounded namespace model reproduces missing reservations and checks the proposed
  allocation ordering, with explicit limitations and implementation correspondence.
- Independent readiness review approves ownership, scope and acceptance.

Checklist

- [x] Read AGENTS.md and loom-framework; follow test-first reproduction, inventory,
  model, readiness review, implementation, bounded output/runtime gates, independent
  exact review and one atomic commit/push. Never edit gen or use forbidden Git
  operations. Root orchestrates delivery and other tickets remain separate.
- [x] Inventory package declarations under `codegen/service/*sections*.go`,
  `endpoint.go`, `client.go`, `authentication.go` and `authorization*.go`; trace
  allocation in `package_names.go`, `service_data_analysis.go` and `scope.go`.
- [x] Add direct scope/type-reference and generated compilation regressions for
  Client, Endpoints, Service, APIName, APIVersion, ServiceName, MethodNames,
  NewClient and NewEndpoints; retain ordinary and suffixed domain-name controls.
- [x] Model fixed declarations and user allocation before/after reservation; keep
  source/configurations and README beside the owning service generator.
- [x] Freeze revised readiness with exact parent failures, inventory and proposed correction.
- [x] Original reviewer approved the revised readiness packet without findings.

### Milestone 2: Centralize fixed package symbols

Goal

Make service analysis reserve emitted framework symbols before naming authored
service-package types, with renderer constants owned by the same definition.

Acceptance Criteria

- All original reproductions generate compiling code, with stable type references
  across generated service, HTTP, gRPC and JSON-RPC consumers.
- Ordinary output is unchanged; same-spelling names in unrelated packages remain
  usable. Conditional declarations and suffix occupancy have explicit controls.
- Representative parent/candidate/repeat generation is compared byte for byte;
  every changed module builds/vets. The full exported corpus is not a local gate.
- Documentation explains deterministic naming and regeneration; all medium/higher
  review findings are fixed and confirmed before one atomic #694 commit.

Checklist

- [x] Implement only after independent readiness review; delete duplicated fixed
  renderer-name literals where the shared owner replaces them.
- [x] Run direct service analysis and actual generated compilation tests, then the
  bounded ordinary/collision/conditional/external-package artifact matrix.
- [x] Check generated type references, explicit gRPC method migration, schema contracts and isolated
  repeat determinism; run relevant package and scoped lint checks.
- [ ] Update owning documentation/model correspondence; freeze exact final diff and
  evidence for a nonimplementer review, fix/re-review findings, commit and push.
- [ ] Close #694 only after pushed delivery; clean inactive generated/checker output
  after preserving concise evidence and update the central recovery tracker.

## Scope boundaries under investigation

#700 collides in transport struct members, #734 in dynamic error constructors and
#736 in protoc-generated symbols. They are evidence for the wider naming family,
not justification to combine unrelated ownership fixes in #694. Dynamic endpoint,
stream and interceptor declarations must be inventoried and their disposition
made explicit rather than silently assuming fixed reservations solve them.

## Readiness evidence and declaration inventory

On `951475f2d`, all nine direct nominal-type assertions fail. Actual generated
service/HTTP code also fails compilation with duplicate fixed declarations;
`/tmp/loom-694-parent-red.log` contains the complete diagnostics. The fixture uses
one field per original name, so no Go-name-equivalence policy is involved.

| Declaration family | Existing owner | Disposition |
| --- | --- | --- |
| Service, Client, Endpoints | service interface, client and endpoint renderers | #694 fixed-symbol owner; existing constants already cover the three names but the interface renderer duplicates its literal |
| NewClient, NewEndpoints | client/endpoint initialization renderers concatenate fixed struct names | #694 derived fixed symbols must share that definition with reservations |
| APIName, APIVersion, ServiceName, MethodNames | service interface constants renderer | #694 fixed-symbol owner |
| Authorizer | service interface renderer when effective Schemes is nonempty | Same fixed-symbol inventory; secure/plain controls required |
| Stream, Event | JSON-RPC stream support, conditional on WebSocket/SSE and results | Same fixed-symbol inventory; WS/SSE/ordinary controls required |
| ServerInterceptors, ClientInterceptors | interceptor interfaces when corresponding effective interceptors exist | Same fixed-symbol inventory; intercepted/ordinary controls required |
| AccessAuthorizer and access-check factories | authorization analysis uses Scope.Unique | Already allocated through shared scope; retain suffixed-name controls |
| User declarations and nominal references | NameScope.GoTypeName / GoFullTypeRef | Must remain one cached identity; relocated struct:pkg:path declarations deliberately retain canonical base spelling in scope.go |
| Make + error name | buildErrorInitData uses Goify directly | #734 separate dynamic-constructor owner |
| Method endpoint/request/response/stream helpers | method analysis and renderers concatenate names | #694 must allocate these package declarations and record every reference when separating method identity |
| Interceptor Info/Payload/Result and wrappers | interceptor analysis/renderers | Independent interceptor identity stays #770/#786; method-derived wrappers/accessor structs are included only to preserve consistency after the method split |
| HTTP server members / transport constructor names / protobuf symbols | transport generator owners | #700/#735/#736 separate lexical namespaces |

The expanded parent matrix confirms conditional collisions: Authorizer in secure
HTTP; Stream/Event in JSON-RPC SSE; Stream in JSON-RPC WebSocket; both interceptor
interfaces when interceptors are present. The same five domain names compile in
plain HTTP. Relocating all nine original names to `domain` also builds/vets;
external declarations must keep their existing canonical spelling.

## Readiness revision: distinguish lexical namespaces

The first readiness review found a medium-severity flaw in the proposal: the
current `scope` also allocates `MethodData.VarName`, so fixed package reservations
would rename legal receiver/interface methods. Before implementation, revise the
proposal and add these controls in sequence:

1. Preserve the reviewer's compiling seven-method fixed-spelling control as a
   durable fixture/test. Package declarations do not occupy receiver namespaces.
2. Characterize the current accidental type/method coupling with a compiling
   streaming fixture containing `Foo`, `FooServerStream`, `FooClientStream`,
   `FooEndpointInput`, and `NewFooEndpoint`, plus method `foo`. Record the parent
   method name and helper identities; add a direct expectation for independent
   method naming that fails before repair.
3. Model the failed shared namespace proposal, the insufficient split that leaves
   concatenated helpers unowned, and a split with package-owned helper declarations.
4. Refine the exact early feature/name-allocation order and bounded matrix. Freeze
   readiness again for the original reviewer's fix review. Production stays frozen.

## Revised allocation sequence

The accepted scope is a package declaration owner plus distinct method/member
owners. Keep the existing cached nominal type identity; do not make transports
choose replacement service type names. The earlier proposal to reserve fixed
names in the scope that also names methods is rejected.

1. Compute pre-name capability facts through the existing classification owners:
   effective security requirements, `classifyJSONRPCStreamTransport`,
   `httpTransportFlags`, and effective interceptor selection. Extract/reuse these
   facts before naming; later analysis and renderers consume the same facts.
   Do not separately infer security, SSE, raw-body or interceptor applicability.
2. Allocate method identities in design traversal order in a member scope.
   Reserve actual fixed members (`Use` on Endpoints and `HandleStream` for a
   WebSocket service), not package declarations or domain types. Preserve the
   existing method collision suffix rule. Seed the client member owner with all
   emitted client methods before allocating endpoint fields, so `foo` alongside
   `fooEndpoint` still gets `FooEndpoint2` as its client field. Authentication
   fields must also use this actual member owner rather than nominal type names.
3. Establish the package NameScope, reserve the nine unconditional and five
   conditional fixed symbols from the inventory, then allocate emitted
   method-derived package declarations in deterministic method/role order before
   collecting authored types. Framework declarations take precedence over domain
   types with the same package spelling. Use the existing unique allocator for
   helper/helper collisions and record each resulting identity once. The plan
   must supply declaration names to MethodData, not retain renderer concatenation
   as a second identity source.
4. Include endpoint factories (`New` + method + `Endpoint`); server/client stream
   interfaces and endpoint input carriers; request, response and file-response
   carriers; and per-method SSE event declarations. Allocate only emitted roles.
   Include the method-derived interceptor endpoint wrappers, private wrapper
   functions/accessor structs and wrapped-stream structs where present. Independent
   interceptor Info/accessor-interface identity remains #770/#786. This boundary
   is essential: splitting methods without owning these package helpers can turn
   a previously compiling design into duplicate declarations.
5. Collect authored types, raw-object wrappers, forced types and projected types
   through the common cached NameScope, preserving external canonical references.
   Populate MethodData using its recorded method/helper identities and those
   nominal references. Existing scope-allocated authentication/access factories
   stay under the package owner. All endpoint, authorization, stream, transport
   and interceptor references consume recorded identities; no downstream renames.
6. Stream `VarName` values describe transport adapter structs, not service-package
   interfaces. Allocate their existing authored-method role bases
   (`FooServerStream`, `FooClientStream`) in a distinct adapter namespace, retaining ordinary output
   and consistency across HTTP/gRPC consumers. Do not let service types or method
   names accidentally occupy that namespace. This covers adapter identity changes
   caused by the split, not #700's independent transport struct-member collisions.

Allocation metadata may be a private per-method plan populated into MethodData;
that is an implementation choice. The owner and ordering above are requirements.
Any newly discovered independent declaration family must be reported to the root
before broadening scope. No public-contract decision remains pending.

## Observed parent behavior and intended differences

The seven fixed-spelling methods (`Client`, `Endpoints`, `APIName`, `APIVersion`,
`ServiceName`, `NewClient`, `NewEndpoints`) retain exactly those names. They compile
on the parent and must retain byte-identical ordinary output after the repair.
The actual member control also compiles: `foo` -> `Foo`, `fooEndpoint` ->
`FooEndpoint`, `Use` -> `UseEndpoint`, and plain-HTTP `HandleStream` -> `HandleStream`;
`foo`'s client field is `FooEndpoint2`. These names must remain unchanged.

The streaming coupling fixture compiles on the parent because a previously
allocated type `Foo` artificially moves method `foo` to `FooEndpoint`. Its exact
intended identity changes are:

| Role | Parent | Candidate |
| --- | --- | --- |
| Method | FooEndpoint | Foo |
| Endpoint factory | NewFooEndpointEndpoint | NewFooEndpoint |
| Server interface | FooEndpointServerStream | FooServerStream |
| Client interface | FooEndpointClientStream | FooClientStream |
| Endpoint input | FooEndpointEndpointInput | FooEndpointInput |
| Client endpoint field | FooEndpointEndpoint | FooEndpoint |
| Server adapter | FooServerStream2 | FooServerStream |
| Client adapter | FooClientStream2 | FooClientStream |
| Authored Foo | Foo | Foo |
| Authored FooServerStream | FooServerStream | FooServerStream2 |
| Authored FooClientStream | FooClientStream | FooClientStream2 |
| Authored FooEndpointInput | FooEndpointInput | FooEndpointInput2 |
| Authored NewFooEndpoint | NewFooEndpoint | NewFooEndpoint2 |

The original nine fixed-name types gain the existing allocator's `2` suffix in
that fixture; each conditional fixed-name type gains it only when its capability
emits the declaration. Plain conditional and external canonical-name controls
remain unchanged. An occupied suffix uses the existing next-free rule; do not
invent a new numbering policy. Method/wrapper identities affected by the old
artificial coupling change together, with every declaration/reference difference
listed in the final comparison. HTTP/JSON-RPC wire names and schema names stay authored. gRPC derives RPC names
from service method identity; the correction below is an explicit migration.

## Bounded acceptance matrix and readiness receipt

Before final implementation review, compare parent/candidate/separate-process
repeat outputs for these obligations (reuse the supplied fixtures):

- Original nine names, plus an occupied suffix, across HTTP and gRPC: package
  declarations and qualified nominal references; parent failures are expected.
- Plain, secure, intercepted, JSON-RPC SSE and WebSocket conditional variants:
  actual feature admission and no unnecessary reservation. Include a per-method
  SSE event-name collision and method-derived interceptor wrapper collision.
- External canonical types and a normal collision-free design: unchanged output.
- Seven fixed-spelling methods and actual method/field controls: unchanged output.
- Streaming type/method/helper coupling above: all listed changes and compiling
  service plus transport adapter references. Exercise HTTP and gRPC stream forms.
- Raw-request, raw-response and file-response method carriers: a domain type and
  emitted method carrier sharing a base; direct assertions plus rendered compile
  coverage establish each newly owned declaration role.

Build/vet every changed candidate module and both revisions of compiling controls;
compare every artifact byte for byte and record all intended differences. Repeated
identical output needs no repeated build. This is a bounded matrix, not the full
exported-design corpus. Interceptor variants must cover server/client applicability
and streaming wrappers only to the extent these method-derived names change.

Parent readiness checks used Go from root go.mod, local source `951475f2d`:
`TestFixedPackageSymbolsDoNotRenameMethods` and `TestServiceMemberNamespaceControls`
pass; `TestMethodAndPackageTypeNamespacesAreIndependent` fails exactly with
`FooEndpoint` instead of `Foo` (0.972s total). All three corresponding generated
modules build and vet (4.009s total). Logs are
`/tmp/loom-694-namespace-parent.log` and
`/tmp/loom-694-namespace-parent-compile.log`.

The original fixed-symbol model remains useful (Missing fails in 2 states;
Reserved passes 39 generated/20 distinct), but alone cannot justify sharing its
scope with methods. The refinement rejects that proposal in 2 states, rejects a
split without helper ownership in 8 generated/7 distinct states, and checks the
owned-helper candidate in 21 generated/14 distinct states. See the adjacent model
README for exact assumptions and implementation correspondence. These readiness results preceded production implementation. Final implementation
validation is recorded below.


## Implemented correction and validation

The original reviewer approved revised readiness manifest
`4d7c8261a81e0c9cc05b29b49641f720ab22ec52800954c162d85ce8859bc801`.
`package_symbols.go` now records package helpers and separates method/member and
adapter scopes. Actual transport/security classification is reused early;
interceptor declaration collection copies its input before sorting/deduplicating,
so repeated pre-analysis cannot mutate the evaluated API's lists. Unemitted raw
body/file carrier metadata is empty. File responses emit both file and response
carriers, and both are owned. Authentication already had an independent endpoint
member owner; it remains separate from client fields and package declarations.

The direct readiness assertions originally checked authored `UserTypeData.Name`;
implementation corrected them to the emitted `VarName`. The parent generated
compilation failures independently establish the original bug. The expanded
fixture includes occupied `Client2`: Client becomes Client2 and authored Client2
becomes Client22 under the existing allocator. Carrier types gain `2`; the
`WrapFooClientEndpoint` domain type gains `3` because two emitted helper roles
occupy the base and `2`. SSE domain WatchEvent becomes WatchEvent2. External
canonical domain declarations retain their names. A checked-in external-package
fixture formerly renamed method Foo to FooEndpoint merely because its payload
used external foo.Foo; its method is now Foo, consistently with the same owner.

The parent/candidate comparison found the existing gRPC dependency on service
method identity (`RPCName: protoServiceName(md.VarName)`). Thus the coupling
fixture's proto RPC changes from FooEndpoint to Foo, as do generated client/server
descriptors and the full method path. Root explicitly accepted this direct
consequence within #694; there is no compatibility alias or unrelated protobuf
repair. `TestServiceSymbolGRPCDescriptors` checks generated descriptor and client
call agreement. Regeneration and handwritten-client migration are documented.
All protobuf messages/field numbers and all 18 OpenAPI documents remain identical.

Bounded matrix: 14 cases, 243 artifacts, 94 changed artifacts, exact isolated
candidate/repeat agreement. Unchanged cases: external, plain, fixed-methods,
members and ordinary. Changed cases: fixed (12 artifacts), secure (6), SSE (9),
WebSocket (6), interceptors (6), coupling (26), carriers (18), SSE-event (8) and
external-method (3). All differences are declaration/reference/helper identities
and their generated comments, with the gRPC RPC migration explicitly noted above.
Exact file list/diff/hash receipts are `/tmp/loom-694-output-comparison.json`,
`/tmp/loom-694-output-diff.patch` and `/tmp/loom-694-output-hashes.json`.

All 14 candidate modules build/vet (12.94s); seven parent controls build/vet and
seven parent collision cases fail as expected (9.40s total). OpenAPI-only harness
extension preserved every previously built artifact hash. Service and shared CLI
package tests pass (38.243s/0.350s); focused HTTP/gRPC CLI tests pass
(54.838s/19.676s). All five pinned TLC configurations rerun with the expected
results. Final direct, descriptor, lint and independent review receipts belong in
the final manifest. No commit, push or ticket closure has been performed here.

Final gates: direct symbol/type/reference controls pass (0.470s); the generated
gRPC descriptor/client-call regression passes (4.276s); scoped golangci reports
zero issues. File-size, gofmt, NameScope and documentation checks pass (existing
file-size warnings remain informational). Regenerating from the final production
source preserves all 243 comparison hashes. Independent final review, delivery
and closure remain pending; the root orchestrator owns those steps.
