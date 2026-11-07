# Copy source nodes without merging representations

`Dup` and `DupAtt` clone a source graph. Distinct named nodes can share a nominal
ID after transport normalization while carrying different fields or metadata.
The copy cache must use source-node identity, retain sharing for repeated edges
to the same node, and register a clone before traversing its recursive edges.
Nominal identity remains appropriate for `DupKeeping`'s declaration selection;
copying does not perform declaration deduplication.

`TypeCopy.tla` explores both traversal orders of a root referencing text and JSON
representations with the same nominal ID. Each representation has a self edge.
`nominal.cfg` reproduces the old cache merging `jsonX` into `x` (or the reverse).
`reference.cfg` keys by source node. Shapes, distinct nodes and complete reachable
edges must survive; weak fairness requires the finite traversal to finish. The
abstract clone is named by its source node, not a real memory address. Allocation
and cache publication are atomic, source graphs are immutable during copying,
and user-type references are assumed comparable. This bounded model does not
prove arbitrary graphs, builtin attribute copying, `DupKeeping`, composition
publication or generated Go/OpenAPI naming.

Run explicitly with TLA+ tools 1.7.4 (TLC 2.19) and Java 25.0.5:

```sh
java -XX:+UseParallelGC -cp "$TLA_TOOLS_JAR" tlc2.TLC -workers 1 -config nominal.cfg -metadir /tmp/loom-copy-nominal TypeCopy.tla
java -XX:+UseParallelGC -cp "$TLA_TOOLS_JAR" tlc2.TLC -workers 1 -config reference.cfg -metadir /tmp/loom-copy-reference TypeCopy.tla
```

The nominal control fails `ShapesRetained` in five generated/distinct states;
the reference model passes with six generated/five distinct states, depth four,
including termination. Proof execution stays opt-in, outside all normal gates.

Implementation: `dupper.uts` in `expr/dup.go` maps source `UserType` references
to clones. `designComposer.detachNamedCopies` seeds that map with the actual
enclosing owner so structural recursion stays attached to it. `typeCopyKey`
still owns semantic declaration identity for kept references and closure.
`TestDupPreservesDistinctRepresentations` checks both visit orders, self cycles,
shared repeated references, different fields and detached mutable attributes.
Existing composition and body-identity tests cover canonical recursive owner
binding and explicit kept-reference selection. `TestFormRootEncodingSchemaIsolation`
checks rendered OpenAPI for the maintained form fixture: text fields use `x`,
JSON values and JSON array items use `jsonX`, with libopenapi model validation.

## Mutating a detached graph

`SourceTraversal.tla` follows a root's two distinct same-ID nodes, each with a
self edge. The nominal visited set skips one node: `traversal-nominal.cfg`
fails `Complete` after nine generated/distinct states. Source identity in
`traversal-reference.cfg` passes complete and exactly-once mutation plus
termination with 14 generated/10 distinct states. Visitation is registered
before invoking a mutating callback. The model treats node references as stable
and comparable and abstracts naming, package strings and effective constraints;
compiled tests check those implementation details.

```sh
java -XX:+UseParallelGC -cp "$TLA_TOOLS_JAR" tlc2.TLC -workers 1 -config traversal-nominal.cfg -metadir /tmp/loom-traversal-nominal SourceTraversal.tla
java -XX:+UseParallelGC -cp "$TLA_TOOLS_JAR" tlc2.TLC -workers 1 -config traversal-reference.cfg -metadir /tmp/loom-traversal-reference SourceTraversal.tla
```

`expr.walkUserTypes` is the shared owner for body suffixes and `RemovePkgPath`.
`makeHTTPTypeRecursive` and transport IR's `normalizeHTTPAttributeRecursive`
likewise mutate detached graphs using source-node visited sets.
`TestHTTPBodyTraversalUsesSourceIdentity`, `TestHTTPTypeNormalizationUsesSourceIdentity`
and `TestNormalizationUsesSourceIdentity` cover both orders, independent same-ID
nodes, shared references, cycles and unchanged sources. Existing HTTP and JSON-RPC
`TestObjectDefaultsTransport` compile and exercise nested service-package types
through their local body/CLI declarations. `TestUserTypePackageHTTPModuleCompiles`
checks request, response, error and streaming package placement.
