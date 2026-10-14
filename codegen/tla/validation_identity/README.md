# Named validator physical identity

The shared named-occurrence renderer used a bare `AttributeExpr{Type: ut}` to
resolve its callable while declaration collectors retained normalized physical
metadata. For an explicit protobuf or shared Go type name, the same scope then
returned different helper names. The finite model isolates this loss of identity
from the recursive predicate worklist modeled in `../recursive_validation`.

`ValidationIdentity.tla` explores two named types, explicit/default physical
names, intrinsic Nullable declarations and occurrence-only Nullable wrappers.
Both declaration emission orders are explored. Definitions and calls must agree
on the pair of physical name and intrinsic parameter carrier. Upstream name
allocation is assumed collision-free for accepted declarations; conflicting
protobuf names are independently rejected by the existing message declaration
owner. The model does not prove that allocator, arbitrary metadata traversal,
Go types, predicate bodies, runtime recursion or diagnostics.

The `drop-metadata` negative control reproduces the current defect by resolving
call names from authored type names alone. The `copy-presence` negative control
shows why blindly copying an occurrence's nullability into its named parameter
is also wrong. The checked design retains physical names while deriving the
parameter carrier from the intrinsic declaration. It makes no transport-specific
renaming decision and introduces no new naming policy.

## Run

Use Corretto Java 25.0.4.1 and TLA+ tools release 1.7.4 (TLC 2.19), JAR SHA-256
`936a262061c914694dfd669a543be24573c45d5aa0ff20a8b96b23d01e050e88`.
From this directory, set `TLA_JAR` to the installed JAR and run sequentially:

```sh
java -XX:+UseParallelGC -cp "$TLA_JAR" tlc2.TLC -workers 1 -config drop-metadata.cfg -metadir /tmp/loom-validator-identity-drop ValidationIdentity.tla
java -XX:+UseParallelGC -cp "$TLA_JAR" tlc2.TLC -workers 1 -config copy-presence.cfg -metadir /tmp/loom-validator-identity-presence ValidationIdentity.tla
java -XX:+UseParallelGC -cp "$TLA_JAR" tlc2.TLC -workers 1 -config checked.cfg -metadir /tmp/loom-validator-identity-checked ValidationIdentity.tla
```

Parent readiness results, before production edits:

| Configuration | Result | Generated / distinct states |
| --- | --- | --- |
| drop-metadata | `CallIdentity` violated, exit 12 | 225 / 209 |
| copy-presence | `CallIdentity` violated, exit 12 | 195 / 194 |
| checked | all three invariants pass | 320 / 256 |

## Implementation correspondence

`NamedValidationFunction` remains the common definition/name/carrier resolver.
`validateNamedOccurrence` supplies physical metadata together with the named
type without substituting occurrence-only nullability or local predicates for
its intrinsic declaration. Existing scopes own interpretation of that metadata.
The correction must not add a protobuf naming rule to shared validation.

`codegen.TestNamedValidationOccurrenceIdentity` checks the existing shared type
name scope, a custom validation prefix, all intrinsic/occurrence nullability
combinations, preserved occurrence requiredness and no source mutation.
`grpc/codegen.TestProtoNamedValidationOccurrenceIdentity` uses the actual
protobuf scope and covers explicit/default names, reserved `Message` naming and
Go conversion of `wire_node`. Before production repair, seven explicit-name
cases fail and seven default-name controls pass across these tests.

Existing generated `TestNestedStructMetaNames` and
`TestGeneratedStructMetaRoundTrip` reproduce the resulting undefined helpers.
They also cover rejected name conflicts, derived types, named union wrappers,
arrays, maps, recursion, client/server calls and streaming. HTTP and JSON-RPC
recursive/nullable generated controls must remain green because the fix lives
in their shared named-occurrence owner. Root-specific predicate semantics and
runtime paths retain the #709 model/tests as separate obligations.

Checker state, logs and temporary modules are task-owned validation artifacts.
Remove them after independent review/integration while retaining this model,
configurations, regression inputs and summarized evidence.

## Final correspondence checks

At parent `baf64781d`, the seven explicit-name seam cases fail; after preserving
physical metadata at the common occurrence resolver, all fourteen cases pass.
Full shared codegen tests and selected generated HTTP/JSON-RPC/gRPC recursive,
nullable and explicit-name tests pass. A bounded parent/candidate/independent
repeat compares 91 artifacts across four existing designs. Only three calls in
the protobuf metadata specimen's client/server `types.go` change, to the existing
`ValidateChoiceProto` definition. All candidate/repeat bytes agree; eleven
modules compile/vet, with the old metadata specimen's expected undefined-helper
failure as the twelfth negative control. No model transition changed after the
readiness check; its recorded results remain valid.

Run the focused implementation checks from the repository root with local source:

```sh
LOOM_DIR="$PWD" go test ./codegen ./grpc/codegen -run '^Test(NamedValidationOccurrenceIdentity|ProtoNamedValidationOccurrenceIdentity|NestedStructMetaNames|GeneratedStructMetaRoundTrip|RecursiveValidationGenerated)$' -count=1
LOOM_DIR="$PWD" go test ./http/codegen ./jsonrpc/codegen -run '^Test(NamedNullableValidationGenerated|RecursiveValidationGeneratedReadiness|RecursiveValidationGenerated)$' -count=1
```
