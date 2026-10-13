# gRPC server receiver identity — #880

The generated server currently derives handler fields by appending `H` to the
canonical service method. A legal `mount` / `mountH` pair therefore declares a
field and RPC method named `MountH` on one receiver. Handler declaration,
construction, unary dispatch and streaming decode/dispatch independently repeat
that derivation. Stream admission uses the broader protobuf scope; embedding
adds another concrete receiver member.

1. [x] Reproduce the existing compiler failure and inventory every concrete
   receiver declaration and use. Add direct rendered-member checks and bounded
   unary, streaming, fallback-suffix, embedded-name and ordinary controls.
2. [x] Model current allocation, a partial reservation alternative and one
   complete receiver owner. Check uniqueness, binding consistency and unchanged
   canonical service/protobuf identity before selecting implementation.
3. [x] Independently review readiness, including any embedding collision
   exposed by the actual generated module; freeze the acceptance matrix.
4. [x] Assign receiver roles after endpoint analysis in one private gRPC owner.
   Reserve actual RPC and embedding members before handler/admission allocation;
   make all declarations and references consume recorded identities. Keep
   package helpers, import aliases and service/wire method identities separate.
5. [x] Compare parent/candidate output for a bounded representative matrix;
   build/vet complete generated modules, exercise mounted unary/stream dispatch
   and authentication, and compare independent-process generation byte for byte.
   Preserve unchanged ordinary outputs and identify each intended difference.
6. [ ] Update framework ownership documentation and any necessary regeneration
   guidance, obtain independent exact-final review, commit, push and close.

The embedding name is externally determined by protobuf. Investigation must
establish whether an authored RPC can collide with it before selecting a local
embedding representation. Do not rename RPCs or silently reject a legal design
merely to accommodate transport fields. Generated stream-wrapper receiver fields
and protobuf package declarations have separate owners. Reuse existing generated
module, native transport and naming tests; no broad exported-design audit.

## Readiness evidence and proposed owner

Seven generated designs cover ordinary methods, a handler suffix, occupied
fallback suffixes, an authored `UnimplementedMembersServer` RPC, server streaming,
ordinary authenticated client streaming and authenticated handler collision.
The five collision designs fail compilation; both ordinary designs compile.
The direct rendered AST finds the same five collisions and preserves the
canonical neutral method and protobuf RPC identity. Initial direct-test setup
incorrectly expected raw Goify for the reserved `use` method; the control now
compares against the actual neutral service owner (`UseEndpoint`). That setup
failure is not bug evidence. Current matrix evidence is
`/tmp/loom-880-readiness-matrix.log` (10.564s).

The embedding collision is real. An RPC named `UnimplementedMembersServer`
conflicts with the embedded `pb.UnimplementedMembersServer` field even though
protobuf generation itself succeeds. Use a local alias only for a colliding
embedding so its method set, including mandatory unexported protobuf methods,
remains promoted. Allocate that alias against actual package/import declarations
as well as receiver roles. Record the resulting embedding reference; preserve
ordinary embedding output. Do not replace it with UnsafeServer, change the wire
RPC, or remove required protobuf behavior. Actual generated registration must
verify the selected representation.

The private finalizer belongs at the end of gRPC endpoint analysis. Reserve all
actual RPCGoName values first; resolve the embedding role, then allocate handler
and conditional authentication fields in canonical endpoint order. Record each
handler identity on EndpointData. All five declaration/use seams listed in the
model README must read it. Decoder/helper names retain the package owner; delete
admission field allocation from that unrelated scope. Ordinary spellings remain
unchanged when no collision exists. FileData reanalysis must preserve references
and avoid import/package alias collisions.

The bounded model reproduces current and partial fixes; owned allocation passes
26/21 states. It does not prove Go embedding or import correctness. Final output
comparison covers these seven modules and representative existing stream-admission
and import-alias designs, not the exported corpus. Compare previous successful
artifacts even for parent modules that fail compilation; canonical service, proto
and client bytes must remain exact. Intended changes are receiver field names,
initializers/references and, for the embedding case, its local alias declaration.
Mounted calls must identify each target endpoint and preserve stream admission
before input frames; include an independent ordinary protobuf client. Unrelated
HTTP/JSON-RPC fixtures are outside this native gRPC receiver-only acceptance.

Scoped codegen lint passes using the task cache. An initial lint attempt failed
before analysis because another project was deleting the default shared cache;
that attempt is preserved separately and is not acceptance evidence.

Implementation starts at `22dddd67fc3d700ca242c96c39b06f51a11cb1cc` after independent readiness
approval. The archived parent tree in the final comparison exercises actual CLI
generation without relying on GOFLAGS overlays. All eight recorded receiver owner hashes remain exact in this parent.

## Final implementation and acceptance

`finalizeServerMembers` now owns the concrete receiver after endpoint analysis.
It reserves canonical RPCs and the embedding before assigning handler and
admission fields. All declaration, initialization, unary and stream consumers
read those records. A colliding embedding uses `type alias = pb.Type`; the
package/import inventory includes emitted helpers, constructors, validation,
response-contract accessors and located type bindings in the current file scope.
The generated test independently checks actual package declarations against that
inventory. No RPC/service/protobuf renaming or unsafe embedding is introduced.

The accepted seven-design matrix adds one required alias/import obligation:
`embedding-import` occupies both the preferred private alias and its first
fallback using located payload/result packages. Direct FileData reanalysis checks
a renamed import, and mounted native protobuf registration exercises the alias's
promoted method set. The final eight modules build and pass vet. Five mounted
race harnesses cover unary handler routing, a unary embedding collision, both
server-stream handlers, located streaming embedding, and authenticated client
streaming. Invalid credentials are rejected while the client has sent no frames;
valid credentials reach the correct method with the expected payload and stream
item. Direct declarations, reanalysis and existing admission/alias controls pass.

Actual CLI generation uses immutable archived parent
`22dddd67fc3d700ca242c96c39b06f51a11cb1cc`, not a GOFLAGS overlay. Parent ordinary and
admission-control modules build/vet; the other six fail with their recorded
receiver collisions. Candidate and independent repeat each contain 122 generated
artifacts. Exactly six `gen/grpc/members/server/server.go` files change: handler
field declarations/initializers/references or the private embedding alias.
All canonical service, client and protobuf artifacts and both ordinary outputs
remain byte-identical. A final maintainability-only extraction of located-import
inventory is checked by direct race tests and two further isolated generations;
these must remain exact with the already compiled candidate artifacts.

The current and partial model counterexamples and the owned 26/21 result remain
valid: model/configuration bytes are unchanged from approved readiness. Models
do not establish package/import legality or protobuf method promotion; the
emitted-declaration checks, compiler and mounted tests discharge those limits.
Scoped gRPC codegen lint reports zero issues. HTTP/JSON-RPC fixture generation is
outside this native-server-only change. Earlier temporary CLI setup and mounted
harness arity errors were corrected and are not counted as bug or acceptance
results. The final manifest records exact source, input and evidence hashes.
