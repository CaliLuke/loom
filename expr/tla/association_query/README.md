# Indexed semantic-owner queries

`AssociationQuery` checks the relation used by
`ValuePlan.OccurrenceForTarget`: reverse structural reachability must select
exactly the owners selected by exhaustive forward search. It informed the
per-capture parent index and per-plan inverse association index. It includes
cycles, disconnected roots, several owners, duplicate associations and a
pair-local preferred association. Constraint provenance and stored result views
are not structural edges.

The finite domain contains three nodes, every directed graph on them, every
owner subset, each root, and every duplicate/preferred assignment. Queries are
atomic; this model does not establish Go synchronization. The implementation
publishes immutable derived indexes through sync.Once, covered by race tests.

## Results and reproduction

With official TLA+ v1.7.4 (TLC 2.19), jar SHA-256
`936a262061c914694dfd669a543be24573c45d5aa0ff20a8b96b23d01e050e88`, and
Corretto Java 25.0.4.1:

- `Indexed.cfg` passes owner-set and outcome equivalence: 196,608 generated and
  distinct states, depth 2.
- `Unscoped.cfg` violates `SameOwners` in an initial state: root 2, owner 1 and
  no connecting edges. This is the deliberately unsafe omitted-scope control.

From this directory:

```sh
java -Xmx512m -cp "$TLA_JAR" tlc2.TLC -workers 1 \
  -metadir "$TLC_STATE" -config Indexed.cfg AssociationQuery.tla
```

Repeat with `Unscoped.cfg`, requiring the named invariant failure rather than a
parser/type/resource error. Keep jar, state and logs outside the repository.

## Correspondence and limits

`expr/value_plan_query_test.go` compares the implementation with an independent
exhaustive traversal over recursive aliases, objects, arrays, maps and unions.
It checks context/capture and target-subplan admission, selected-body ambiguity,
pair-local precedence, exclusion of constraint-parent edges, concurrent index
publication and allocation scaling. Metadata tests check immutable captured
strings and inline-body classification without exporting mutable graph pointers.
HTTP member/branch lookup uses the shared query; ordered alias-chain lookup keeps
its distinct first-match contract. Existing representation/OpenAPI tests and
independent-process exact bytes cover default, hidden-field, recursion and codec
behavior beyond this model.

The model assumes the captured relation is correct. It does not prove plan
construction, native memory bounds, Go race freedom or arbitrary callback purity.
Reverse reachability can still visit a whole dense graph; the index avoids
repeating exhaustive sibling searches in the common sparse case and retains
linear edge/association storage, not a transitive closure.
