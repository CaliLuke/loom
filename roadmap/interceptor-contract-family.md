# Interceptor contract family — #767 and #785

Readiness investigation at `d928a9fef5b96dde85bb2cae3b535f9bab7b52e7`.
Independent readiness approved in
`/tmp/loom-interceptor-family-root-readiness-verdict.json`. #767 implementation
uses parent `b7514c8657d5cc51b2316ae7ef5e86c78aded921`; safe fast-forward preserved
readiness inputs. #785 implementation remains a separate following delivery. #770 and #786 are package-identity boundary controls; their
original reports must be checked against the delivered service symbol owner.

1. [x] Inventory the shared accessor contract, method-local storage bindings,
   endpoint roles and stream operation adapters across server and client.
2. [x] Reproduce optional/required and incompatible shared field signatures;
   retain mapped/renamed and nominal/import identity obligations. Reproduce
   JSON-RPC SSE, bidirectional and request/reply wrapper failures using actual
   generated modules, with ordinary HTTP/gRPC and existing client controls.
3. [x] Reconcile #770/#786 against existing package symbol ownership. Keep
   independently owned symbol defects outside the accessor/stream repair unless
   evidence establishes a necessary shared decision.
4. [x] Extend the existing interceptor models to distinguish a shared signature
   from each method's storage, and supported stream operations from endpoint
   carriers. Reproduce current failures and reject partial fixes.
5. [x] Propose one service-level typed contract analysis with method-specific
   bindings and adapters. Validation rejects incompatible shared signatures;
   optionality alone remains compatible through each method's field storage.
   Preserve the accepted JSON-RPC service contracts and side-local invocation.
6. [x] Freeze exact readiness sources, owners, failing/ordinary evidence and a
   bounded final acceptance matrix for independent review before implementation.

Initial source findings: `buildInterceptorData` copies accessor TypeRef, Pointer
and field spelling from the first applicable method. The shared declaration union
then prefers server metadata. `addAccessorMethods` applies this shared physical
representation to every method, although the shared interface only requires one
logical getter/setter signature. `renderServerInterceptorWrapperBody` assumes a
stream carrier, and `addStreamWrappersSection` assumes the ordinary HTTP/gRPC
stream signature. JSON-RPC event sends and request/reply endpoints do not have
those shapes. Existing client endpoint-role planning and stream naming must be
reused rather than replaced with per-transport policy branches.

## Readiness evidence

Actual CLI generation uses the local readiness checkout and a pushed parent pin.
`/tmp/loom-interceptor-family-readiness/results.json` records eight modules:
normal shared optional storage and native gRPC streaming build/vet; required
second-method storage, renamed second-method physical fields, incompatible
string/int signatures, JSON-RPC SSE and bidirectional wrappers fail compilation;
JSON-RPC request/reply fails generation on its missing endpoint carrier. The
paired direct accessor test reproduces required and renamed selectors while its
normal control passes. These are recorded failures, not implementation changes.

`symbol-results.json` separately confirms #770 still redeclares `AuditPayload`
and `AuditResult`, whereas #786's original concatenated wrapper-pair example now
builds/vets under the delivered `serviceSymbols` allocator. Package reservations
for public shared interceptor declarations remain a distinct obligation; the
field-binding repair must not create its own naming scope or absorb unrelated
namespace changes. #786 needs ticket reconciliation with its existing owner and
retained regression evidence, not a new allocation algorithm.

## Consumer inventory and proposed division of responsibility

| Decision | Owner / consumers at readiness | Corrected responsibility |
| --- | --- | --- |
| Access applicability | `expr.InterceptorExpr.validate`; API/service/method lists | Keep field existence, direction and side semantics; admit one compatible logical signature per service/interceptor/access role/field over both sides. |
| Shared declarations | `buildInterceptorDeclarations`, `interceptorTypesSection`, Info accessors | Keep the server/client union and one shared typed getter/setter contract; never choose physical field layout from the first/server method. |
| Method binding | `populateInterceptorAttributes`, `collectAttributes`, `addAccessorMethods` | Record each method's physical selector, field carrier and read/write conversion independently; reuse `ServiceFieldLayout`, field-name metadata and qualified type owners. |
| Client endpoint role | `newClientInterception`, `ClientInterceptorWrapper`, constructor | Retain delivered ordinary/mixed/stream selection, callback order, canonical method metadata and error behavior. |
| Server stream role | `initStreamData`, `streamInterfaceData`, `buildStreamInterface`, server wrapper | One operation/endpoint shape consumed by both interfaces and adapters. Generic send/recv, JSON-RPC SSE event sends, WebSocket response/notification/error sends and per-message endpoint input are distinct recorded roles. |
| Package identities | `serviceSymbols`, interceptor declaration names | Preserve the existing package NameScope. #770's missing public declarations belong here; #786 wrapper pairs are already recorded and scoped. |

Optional versus required primitive fields can share the existing value-returning
getter and value-accepting setter: nil reads as zero and writes initialize an
optional pointer, while required values use their own field directly. Renamed Go
fields change the local selector, not the public logical accessor name. Different
primitive signatures cannot implement one shared typed interface; reject them
with both conflicting methods and the access role/field named. Constraint
variation alone does not change a getter/setter type. Nominal types, explicit
presence and custom Go replacements require canonical signature identity rather
than structural `expr.Equal`, which intentionally ignores named-type identity.
Admit the shared signature in finalized service analysis before publishing
`Data` or emitting files. Reuse `NameScope.GoFullTypeRef` with the service's
qualified import identities and `ServiceFieldLayout`; do not recreate Go type
policy in `expr` or compare structural `expr.Equal`. Compare signatures within
one resolved service scope and rebuild on `fileData` import reanalysis. Custom
external aliases are compared by their declared qualified reference; Loom does
not infer assignability from arbitrary external package implementations.

Use the existing `codegen.NewError(d.Ctx.WithMethod(method), method, err)` path.
`ServicesData.analyze` already preserves attributed errors, and `codegen.Error`
uses `eval.ExpressionLocation` for source attribution. The error detail names the
interceptor, read/write role, field, both methods and both signatures. Keep this
private to the service owner; no new framework-wide validation API, coercions,
`any` accessors or rejection of optionality.

For JSON-RPC WebSocket request/reply and bidirectional service methods, incoming
messages arrive as endpoint payloads, not via a server stream `Recv`. Preserve
that accepted service shape and apply streaming-payload interception at that
message boundary. SSE `Send`/`SendAndClose` and WebSocket
`SendNotification`/`SendResponse` intercept their canonical result value, preserve
the operation/context and delegate error sends and cleanup without treating them
as result data. Existing HTTP/gRPC and client stream behavior remains the control.
#767 now separates shared signatures from method-local bindings and rejects
incompatible signatures through the existing attributed service analysis error.
The #785 failing test and extra DSL witnesses are preserved in the exact readiness
source archive outside #767's atomic diff, and will be restored for that repair.


## Implementation sequence and bounded acceptance

This family has two different semantic owners and two atomic deliveries.

1. **#767 — typed accessor contract and method binding.** Replace first-method
   physical metadata with a shared logical field signature and method-local
   bindings for the eight read/write payload/result/message roles. The existing
   declaration union still determines placement and coverage. Keep server/client
   applicability and callback ordering unchanged; use each method's selector,
   carrier and conversion, including viewed-result access. Validate incompatible
   signatures across the whole union, including disjoint server/client methods.
   A named object pointer already present in its logical TypeRef must not gain
   an extra dereference merely because field storage is a pointer.
2. **#785 — server operation and endpoint adapters.** Record canonical operation
   and carrier roles in the existing stream analysis, shared by interface and
   interceptor renderers. Delete the generic-server reconstruction that invents
   unsupported Send/Recv/Close methods. Keep client operations on their existing
   owner. Preserve result values, contexts, errors and cleanup delegation; incoming
   JSON-RPC messages use their actual endpoint boundary. This can follow #767 but
   does not justify combining the tickets into one commit.
3. **Symbol reports.** #770 remains an independently reproduced package identity
   defect; fix its reservations through existing `serviceSymbols`, separately.
   #786's original generated design builds/vets; retain this and wrapper reference
   controls as reconciliation evidence, without claiming every symbol report fixed.

For #767, extend the existing storage specimen and accessor-coverage harness,
not a new corpus: optional/required in both method orders, renamed selectors,
compatible named/custom/nullable fields, incompatible primitive/nominal/nullable
signatures, and disjoint server/client coverage. The six signature probes record
actual emitted getter and field AST types: optional `string`/`*string`, required
`string`/`string`, nominal `*FirstValue` versus `*SecondValue`, custom
`time.Duration`/`*time.Duration`, and `loom.Nullable[string]` on both sides.
Generate/compile valid representatives and reject invalid pairs before output.
One mounted unary read/write case exercises server and client values, zero/nil
reads and setter writes; reuse existing native-stream coverage for all message
accessor directions and viewed-result tests for projection conversion. Assert
qualified import/file reanalysis retains logical signatures and local selectors.

For #785, the three failing RPC specimens plus native gRPC control are the core
matrix. Mounted generated SSE and WebSocket cases must assert callback call type,
canonical method metadata, field mutation, service invocation, response/error
propagation, contextual calls and cleanup. Cover request/reply ordinary result
access and streamed-payload access together, and bidi incoming messages plus
outgoing responses/notifications. Keep generic no-access interceptors and the
existing client ordinary/mixed/stream endpoint selection controls. Do not change
HTTP/gRPC final-response semantics or reject supported JSON-RPC stream designs.

For each ticket, capture immutable parent and candidate output plus a second
process-isolated candidate generation for its affected specimens. List intended
file changes; compare unaffected bytes exactly, build/vet successful modules and
run selected mounted race tests. The affected service generator also warrants
all three checked-in SSE fixture captures (HTTP ticktock, JSON-RPC ticktock and
mixedtick); use current designs after integration and compile their generated
apps. Reuse existing fixtures and model evidence. No full exported-design corpus,
new allocator or universal interceptor audit is an acceptance condition.

The readiness direct run has six expected failing subcases: required storage,
renamed storage, missing incompatible-signature admission, RPC SSE, RPC bidi and
RPC request/reply. Normal storage, all six signature probes, native gRPC, existing
shared declaration coverage/import aliases, client endpoint selection and wrapper
reference controls pass. The new model rejects current storage and operations,
layout-only, signature-only, methods-only and rejecting-valid-variants repairs;
the owned plan passes 24 generated / 16 distinct states. See the owning model
README for correspondence and limits. Those results describe the independently approved readiness packet. #767 final
validation is recorded below; none establishes #785 delivery.


## #767 implementation and final validation

- [x] Separate shared logical AttributeData from per-method selector and storage
  conversion. Admit signatures over both invocation sides before Data publication.
- [x] Preserve existing declaration placement, stream shape and side applicability;
  remove first-method physical metadata and renderer reuse of that storage.
- [ ] Complete bounded parent/candidate/repeat comparisons, mounted checks and
  exact independent final review before the atomic commit.

`interceptor_contract.go` owns all eight read/write field binding groups. It uses
`ServiceFieldLayout` and NameScope's actual type references, including explicit
presence and nullable collection carriers; object references already containing a
pointer are not dereferenced twice. Value storage behind a pointer logical
reference uses an address getter and a nil-safe setter. No user type is coerced.
The existing declaration union only controls output placement; server preference
cannot alter validated signatures or method-local storage. FileData reanalysis
rebuilds the same decisions under its import aliases.


The final bounded CLI comparison uses an immutable `git archive` of the final
parent, with only the identical authored test design factory added. Eight valid
specimens cover ordinary storage, both requiredness orders, renamed selectors,
shared named objects, custom scalar representations, nullable fields and optional
versus required union storage. All build/vet. Three incompatible primitive,
nominal and nullable specimens now reject before output; their parent versions
emit invalid Go. Exactly six `service_interceptors.go` files change in valid
outputs; ordinary and nominal cases are unchanged. All 152 candidate artifacts
match two independent final-source generations. The 55 generated files of HTTP
ticktock, JSON-RPC ticktock and mixedtick match parent/candidate/repeat and checked-in
bytes; candidate apps build/vet.

The full service package passes (40.796s). Focused direct race checks pass
(1.958s), covering compatible and disjoint-side rejection, local streaming
bindings, import reanalysis and retained client roles. Generated unary/native
stream/import coverage and mounted optional/required mutation pass (15.971s);
the actual HTTP fixture harness also passes race (1.500s). A separate generated
union harness checks address getters and nil-safe writes to required value
storage (3.349s). Scoped lint reports zero issues. The explored model sources and
configurations are unchanged from approved readiness and their evidence is reused.

Evidence: `/tmp/loom-767-artifact-comparison.json`,
`/tmp/loom-767-generated-delta.patch`, `/tmp/loom-767-artifacts.json` and the final
manifest. #785 operation/carrier fixes, #770 package declarations and #786 ticket
reconciliation are excluded. No full-corpus or aggregate-CI success is claimed.
