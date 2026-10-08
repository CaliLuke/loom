# Reachable protobuf field validation

`FieldTags.tla` models traversal of selected protobuf message roots through
object fields, collections and union branches. Each object owns its field-number
namespace. Sharing and cycles must terminate without omitting reachable objects;
custom protobuf mappings are opaque. The finite specimen includes a shared
cycle and an external mapping whose internals must remain unvisited.

Run with the repository's TLC 1.7.4 distribution (TLC 2.19, revision 5a47802):

```sh
java -cp "$TLA_JAR" tlc2.TLC -config Recursive.cfg FieldTags.tla
java -cp "$TLA_JAR" tlc2.TLC -config Shallow.cfg FieldTags.tla
```

`Shallow.cfg` reproduces the old defect: validation finishes after checking only
the root, violating `Coverage` (2 states). `Recursive.cfg` satisfies `Coverage`
and `Opaque` (7 distinct states). This supports a visited-attribute traversal,
with field-number claims reset at each object, instead of generator panic
recovery or special treatment of `Any`.

Implementation: `expr/grpc_field_tags.go`, especially `validateRPCMessageTags`.
`TestNestedRPCFieldTags` covers object, array, map and union edges, missing and
duplicate tags, and custom protobuf mappings. `TestRecursiveRPCFieldTags` checks
termination and a single diagnostic on a cyclic graph. The DSL message-role
test checks request, result, stream, collection, error and metadata boundaries.
The exported tagged and untagged Any-union fixtures connect the traversal to
real generation: the invalid specimen must fail DSL validation; the valid one
must retain byte-identical protobuf/Go output and compile.

The model assumes correctly selected message roots and edges. It does not prove
Go graph extraction, metadata filtering, numeric tag validity, protobuf naming,
DSL inheritance, or emitted code. Those remain expression and generator tests.
