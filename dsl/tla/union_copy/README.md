# Copying union branch attributes during DSL evaluation

A branch attribute owns metadata, validations, and documentation, but refers to
a canonical type that may still be unresolved. Copying the entire type graph
before restoring that reference traverses incomplete definitions and can panic.
Sharing the whole attribute instead lets customization mutate its source.

This model explores all 16 directed graphs on two type nodes, both branch roots,
and every ordering of type completion, two branch copies, and their metadata
mutations. The graphs include self edges and mutual cycles. A deep copy fails
when any reachable node is incomplete. The repaired copy keeps the type
reference without traversing it and gives each branch its own metadata.

- `deep.cfg` reproduces the incomplete-type traversal failure.
- `shared.cfg` shows that sharing mutable metadata changes the source.
- `checked.cfg` checks safe copying, unchanged source metadata, independent
  copies, and termination under weak fairness. It passes 2,752 generated
  states and 1,152 distinct states.

Run from this directory with the TLA+ tools jar:

```sh
java -cp /path/to/tla2tools.jar tlc2.TLC -config deep.cfg UnionCopy.tla
java -cp /path/to/tla2tools.jar tlc2.TLC -config shared.cfg UnionCopy.tla
java -cp /path/to/tla2tools.jar tlc2.TLC -config checked.cfg UnionCopy.tla
```

The first two checks intentionally fail their invariants; the last must pass.
The reachability expansion is exact only for this two-node graph bound. The
model abstracts Go pointers, type-name resolution, nested attribute shapes,
and the individual metadata fields. It does not prove that every recursive
union can be rendered or serialized. DSL tests assert canonical references and
metadata isolation; generated probes must compile and exercise recursive values.

# Rejecting inline union branch cycles

The copy fix exposed a second failure: generators expand union branches and collections
inline and overflow their stack on a cycle without an object boundary. `UnionExpansion.tla` models
that expansion on all 512 directed graphs of three inline expansion nodes and each root.
Object fields are boundaries and therefore do not add edges to this graph.

`expansion_legacy.cfg` reproduces unbounded expansion. The candidate detects a
node already on the active expansion path and rejects that design before code
generation. `expansion_checked.cfg` checks that acceptance excludes reachable
cycles, rejection never rejects an acyclic graph, and checking terminates. It
passes 17,136 generated states and 7,521 distinct states.

```sh
java -cp /path/to/tla2tools.jar tlc2.TLC -config expansion_legacy.cfg UnionExpansion.tla
java -cp /path/to/tla2tools.jar tlc2.TLC -config expansion_checked.cfg UnionExpansion.tla
```

The first run must fail `BoundedExpansion`; the second must pass. This bound
covers three nodes, not every design size. In the Go checker each type changes
from unseen to active to done once; a second active visit rejects the graph and
a done visit stops. Thus a finite type graph terminates. An independent Go test
compares the checker with transitive closure on all three-node graphs. DSL tests
cover named aliases, self and mutual cycles, cycles through arrays and maps, and
recursion through an object. An actual collection-cycle generation reproduced
the stack overflow too.
