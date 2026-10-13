# Transport server member identity — #700

The generated HTTP Server places endpoint handler fields in the same receiver
namespace as Mount, Service and Use, and its Mounts field. Endpoint fields
currently reuse canonical service method names without owning that namespace.
JSON-RPC ordinary server fields reuse the same service names and have additional
fixed members; its WebSocket fields also collide with private codec members.
gRPC appends H to method names for fields without reserving actual RPC methods:
methods mount and mountH generate a field/method collision on MountH.

1. [x] Read the report, service method/package identity owner and HTTP/JSON-RPC
   struct/method renderers. Preserve the existing separation of package types,
   canonical service methods and concrete receiver members.
2. [x] Reproduce direct rendered member collisions and generated compilation;
   retain noncolliding controls and unchanged service method names.
3. [ ] Inventory every server member consumer, including file handlers and
   ordinary/streaming JSON-RPC. Establish the common namespace owner and model
   registration-before-allocation without broadening unrelated namespaces.
4. [ ] Obtain independent readiness review of the proposed owner and bounded
   acceptance before production edits.
5. [ ] Implement canonical recorded transport member identities; derive all
   receiver accesses from them, with no renderer-specific suffix exceptions.
6. [ ] Compare representative parent/candidate/repeat output exactly, compile
   and exercise mounted routing, and update owner/consumer guidance.
7. [ ] Obtain independent final review, integrate, push and close the ticket.

No public policy change is proposed. Authored wire paths/method names and the
canonical service method interface should not change merely because a transport
Server has a fixed member with that spelling. The precise common owner and
JSON-RPC/gRPC scope remain under investigation; no production change is made.

## Reproduced boundaries

HTTP direct AST checks find Mount, Service and Mounts duplicates; a generated
module fails compilation for the same fields and methods. Its ordinary read/use
control passes the direct check. JSON-RPC unary output duplicates Methods and
embedded Handler and collides with Service, MethodNames and Mount methods.
WebSocket output additionally duplicates decoder, encoder and errhandler fields.
The final WebSocket reproduction uses a valid GET transport route; an earlier
POST probe was invalid and is not evidence. gRPC mount/mountH fails on MountH,
while mount/service/use without the H-suffix competitor compiles.

Evidence logs: `/tmp/loom-700-readiness-red.log`,
`/tmp/loom-700-rpc-readiness-red-final.log`, and
`/tmp/loom-700-grpc-readiness.log`. These are readiness regressions, not passing
acceptance or delivered work.

## Owner inventory underway

HTTP endpoint fields, New struct keys, Use wrapping, Mount routing and CORS
OPTIONS routing all reconstruct Method.VarName. File server VarName must share
the concrete Server member scope, not its unrelated package-name scope.
JSON-RPC top-level Server and shared WebSocket Stream have different fixed
members but copy the same endpoint decoder/endpoint identities between them;
allocation must account for both receiving namespaces. The declaration, server
initializer, stream constructor, Use/dispatch and stream Receive call sites must
consume recorded names. gRPC New and all unary/streaming handler accesses derive
Method.VarName+H; actual RPC methods and admission-check fields need reservation
before handler fields. Keep canonical service and protobuf RPC identities stable.

Reuse existing NameScope allocation, with one recorded concrete-member binding
per semantic role. Do not rename service methods globally or add suffix decisions
in renderers. The appropriate HTTP/JSON-RPC analysis boundary and gRPC counterpart
still require a complete consumer inventory, bounded collision model and
independent readiness review before production changes.
