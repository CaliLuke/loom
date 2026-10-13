# Generated receiver naming closure — #883

## Contract and bounded inventory

Every concrete Go receiver has one complete declaration plan. Preserve authored
service methods and wire identities; allocate synthetic roles in that receiver's
scope and make declarations and references consume their recorded identities.
`NameScope` already provides allocation. This work adds no allocator or global
namespace and does not merge receiver, package and import scopes.

| Scope | Owner and emitted consumers | Evidence / remaining obligation |
| --- | --- | --- |
| Service Client | `service.assignEndpointFields`; `clientStructSection`, `clientInitSection`, `buildClientMethod`, `endpointData` | Defect: mixed method rendering reconstructs `VarName + "Stream"`; allocation reserves but discards that identity. Authored `watchStream` therefore duplicates the synthetic method. Constructor arguments already have their own scope; interceptor wrapper declarations are package-owned. |
| HTTP/JSON-RPC Client | `http/codegen.finalizeClientMembers`; concrete methods/builders/Doers, group accessors, CLI adapters | Delivered #879 records endpoint, stream, request and Doer identities. Existing rendered complete-member inventories and mounted dispatch checks cover this scope. The same colliding design has unique transport members while the service Client fails. |
| HTTP/JSON-RPC Server | `http/codegen.finalizeServerMembers`; constructor, mount, middleware, dispatch, copied WebSocket fields | Delivered #700 records handler/endpoint identities and includes copied WebSocket receiver members. Existing emitted-member inventories and mounted checks cover this scope. |
| gRPC Server | `grpc/codegen.finalizeServerMembers` in active #880 | Root-owned implementation separately reserves RPCs and embedding before handler/admission fields; an embedding alias crosses package/import and receiver scopes. Its pending compiler/registration checks own that boundary. This task neither edits nor claims delivery of #880. |

The inventory stops here: these are the concrete receiver owners implicated by
the known naming family. It is not a claim that all generated package namespaces
or every possible authored name have been exhaustively audited.

## Repair and acceptance

1. [x] Preserve delivered #705 inputs/evidence, reverse only its exact patch and
   fast-forward the clean managed checkout to `64fcddc2550dec9fc358d037b9dacb116b55fbca`.
2. [x] Add direct rendered service-Client tests before production; reuse the
   existing HTTP client-member specimen and module renderer for compiler proof.
   Include ordinary, mixed without collision, authored collision in both orders,
   occupied fallback and endpoint-field competitors. Preserve authored names.
3. [x] Independently review this owner proposal and the red/control evidence.
4. [x] Replace `assignEndpointFields` with complete service-Client role allocation:
   reserve all canonical `MethodData.VarName` values first; allocate and record
   each mixed streaming method identity, then endpoint storage fields. Render
   streaming method declaration and GoDoc from that record. Delete the discarded
   stream reservation and renderer suffix reconstruction. Constructor arguments
   and package interceptor wrappers retain their existing owners.
5. [x] Extend existing mounted mixed-client coverage to invoke both authored and
   synthetic methods, proving ordinary/stream endpoint routing, interception and
   propagation of endpoint errors. Verify file/import reanalysis retains the
   recorded receiver identities. No new transport protocol behavior is proposed.
6. [ ] Compare bounded parent/candidate/repeat output through existing renderer:
   ordinary, normal mixed and original colliding HTTP modules. Only collision
   service-client declaration/GoDoc and necessary field identities may change;
   service interfaces, transport clients, wire mappings and wrappers stay exact.
   Compile/vet all three modules; run the affected mounted test. Reuse checked-in
   HTTP/JSON-RPC ticktock fixture inputs for affected service-client byte parity,
   not another unrelated fixture matrix. Obtain fresh exact-final review.

The existing `codegen/tla/client_members/ClientMembers.tla` `streams` specimen is
already the missing case: `Watch`, authored `WatchStream`, synthetic `WatchStream`.
Its current allocation fails uniqueness and the owned strategy passes by
publishing the allocated role identity. Reuse its unchanged checked evidence;
Go AST uniqueness, constructor/method binding and mounted behavior check actual
registration and correspondence. A second naming model adds no obligation.

Public effect: when an authored method occupies the conventional mixed-stream
client name, the synthetic client entry point receives a deterministic suffix.
Applications must regenerate and use its emitted declaration. Authored service
interface methods, CLI commands, wire names and interceptor method identity stay
unchanged. No user policy decision is required by this existing contract.

Recovery: `/tmp/loom-883-parent-reuse.json`; #705 compact cleanup receipt at
`/tmp/loom-705-cleanup-receipt.json`. The root agent owns the shared backlog tracker.

## Readiness result

`TestServiceClientMemberIdentity` reports exactly four duplicate-declaration
failures: authored collision, reverse order, occupied fallback and endpoint-field
competitor. Ordinary and noncolliding mixed controls pass. The reused complete
HTTP generator fails `go build ./...` with duplicate `Client.WatchStream` on
lines 41/51 of its generated service client; generation and module tidy pass.
Evidence: `/tmp/loom-883-direct-parent.log` and
`/tmp/loom-883-compiler-parent.log`.

The existing HTTP/JSON-RPC client and server rendered-member inventories pass
under race (2.097s / 2.643s), including the transport side of the original
colliding design. Evidence: `/tmp/loom-883-existing-owner-inventory.log`.
The original #879 approved model/source hashes remain unchanged; its `streams`
specimen and allocation counterexample are reused rather than rerun or cloned.

Proposed record: `MethodTransportData.ClientStreamMethod` (set only for mixed
results). The service analysis finalizer owns it with `EndpointField` and
`StreamEndpointField`. Canonical `VarName` retains ordinary/authored invocation;
`ClientInterceptorWrapper(stream)` retains package wrapper identity and receives
the already-scoped constructor argument. The only production consumers requiring
change are streaming method GoDoc and declaration in `client_jennifer.go`.
Remove or internalize the now-pass-through `AssignEndpointFields` method rather
than creating an additional allocation layer. File/import reanalysis must use
the same canonical method order and therefore reproduce the role plan.

## Final acceptance (awaiting independent review)

The owner now records `ClientStreamMethod` and renders both its declaration and
GoDoc; the discarded reservation and pass-through field allocator are removed.
The six direct scenarios pass with complete emitted-member/record agreement and
file-scope reanalysis. Existing client goldens and interceptor dispatch controls
pass under race (2.040s). Scoped service/HTTP codegen lint reports zero issues.

Three modules build/vet; the original collision is extended with a generic audit
interceptor before parent and candidate generation. Its mounted peer proves
ordinary `Watch`, synthetic `WatchStreamStream`, authored `WatchStream` and the
endpoint-field competitor route correctly. Interceptor method names remain
canonical; nil interception preserves original endpoint errors for both forms.
The generation/build/vet/mounted gate passes (10.464s).

Parent/candidate/independent-repeat each contain 45 generated files. Exactly one
file changes: the collision service client's synthetic method declaration and
GoDoc. All transport, canonical service, endpoint, constructor and interceptor
wrapper files stay exact. Both ordinary modules stay exact. HTTP and JSON-RPC
ticktock regenerate all 39 checked-in files unchanged and build/vet. These
fixture parent bytes and inputs are unchanged in this checkout; no new fixture
matrix or model was introduced. Evidence: `/tmp/loom-883-artifact-comparison.json`.

The exit condition is the remaining service Client receiver boundary, not every
possible codegen namespace. #879/#700 already cover the inspected transport
receiver boundaries; #880's distinct native-server delivery remains root-owned.
