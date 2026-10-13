# gRPC receiver member ownership (#880)

This finite model covers five receiver shapes: ordinary methods, a handler/RPC
collision, authored names that occupy proposed suffixes, a protobuf embedding/RPC
collision, and a streaming handler collision. Allocation and reference binding
are separate so repairing declarations alone is insufficient.

The current strategy reproduces duplicate members. Reserving only handlers still
leaves an embedding collision. Changing declarations while reconstructing old
reference names violates binding consistency. The owned strategy reserves all
actual RPC methods, allocates embedding and handler roles in that receiver scope,
and publishes the same identities to all consumers. Ordinary names stay exact.
Candidate suffixes abstract NameScope choices; this model does not prescribe
literal Go identifiers or prove an unbounded allocator.

Run with Corretto 25 and TLA+ tools 1.7.4 (TLC 2.19), jar SHA-256
`936a262061c914694dfd669a543be24573c45d5aa0ff20a8b96b23d01e050e88`:

```sh
java -XX:+UseParallelGC -cp /path/to/tla2tools-1.7.4.jar tlc2.TLC \
  -workers 1 -metadir /tmp/loom-grpc-server-members-owned \
  -config owned.cfg ServerMembers.tla
```

Repeat with each other configuration and a distinct temporary state directory.
Current and handlers-only fail UniqueMembers at 9 generated/distinct states;
declarations-only fails BindingConsistency at 9 states. Owned passes all three
invariants at 26 generated / 21 distinct states.

## Implementation correspondence and limits

`ServicesData.analyze` finishes collecting endpoint RPC names and conditional
admission data. `finalizeServerMembers` then assigns concrete
handler and embedding roles. `grpcServerStructSection`, `grpcServerInitSection`,
`grpcServerSpecFields`, `writeGRPCStreamServerMethod` and
`writeGRPCStreamInputDecode` must consume those records. Admission's CheckField
belongs to this receiver owner; its decoder and metadata helpers remain in the
package scope. `fileData` reanalysis must preserve semantic binding when import
aliases differ.

`TestGRPCServerMemberDeclarations` checks actual rendered receiver ASTs and keeps
canonical service/protobuf identities. `TestGRPCServerMemberIdentityGenerated`
reproduces actual compiler failures and ordinary/authenticated controls. Mounted unary/streaming/auth dispatch verifies protobuf registration, endpoint
identity and authentication before any stream frame. The generated test also
checks package declarations against the alias reservation inventory.
`TestGRPCServerEmbeddingImportIdentity` covers both occupied alias spellings and
FileData reanalysis of located type imports; actual CLI comparisons use an
archived parent and separate candidate/repeat processes.

A colliding protobuf embedding uses a local type alias so the mandatory
unimplemented-server methods remain promoted. Go/protobuf method-set behavior,
package/import collisions, stream admission and runtime dispatch are not modeled;
the corresponding generated compiler and mounted tests are required. This model
neither authorizes wire renaming nor treats rejected legal designs as a solution.
