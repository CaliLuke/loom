# Example membership over the final normalized graph

`ExampleMembership.tla` decides when the importer may judge an example, which
registry resolves its references, and which domain judges an example that
normalization copied into another schema. Components are analyzed in
declaration order; promoted object constructors join `Document.Components`
after analysis. Each component has local assertions, an optional reference
(`None`, another component or `Missing`) and an optional example over two
values and `Null`. A component may be flattened into a nullable wrapper
(`wrap`, as `anyOf: [T, {type: null}]` or a singleton `oneOf` with a bare null
does), and its example is authored either on the inner non-null branch or on
the wrapper occurrence (`author`).

- `Sound`: every retained example is a member of its authored owner and of the
  schema that renders it. The inner branch domain is the conjunction of local
  assertions and the reference target; a missing target or a reference cycle
  at the same value position is never membership.
- `Complete`: every example that is a member of both is retained.

Implementation correspondence: `exampleSource.owner` binds each example at
read time (`analyze_examples.go`) to the normalized schema it was authored on;
constructors complete that owner in place with `Ref`, `Bases` and properties.
`analyzer.retainMemberExamples` and `exampleGraph.retain`/`member` in
`example_membership.go` judge owner and rendering schema over
`Document.Components.Schemas`. `member` returns membership and, for an object
value, the members its contributors evaluated; that set is outside this
model. Regression tests: `example_membership_test.go`
(`TestExampleMembershipFinalGraph`, `TestExampleMembershipAuthoredOwner`,
`TestExampleMembershipMatcher`, `TestExampleMembershipClosureProfiles`).

Run TLC 1.7.4 explicitly with a distinct temporary directory per run:

```sh
model_tmp=$(mktemp -d)
java -XX:+UseParallelGC -Djava.io.tmpdir="$model_tmp" -cp "$TLA2TOOLS_JAR" tlc2.TLC \
  -workers 1 -config final.cfg -metadir "$model_tmp/meta" ExampleMembership.tla
```

Observed with Java 25.0.5 and TLC 1.7.4 (`tla2tools` sha256
`936a262061c914694dfd669a543be24573c45d5aa0ff20a8b96b23d01e050e88`):

| Configuration | Expected and observed |
|---|---|
| `final` | pass, 1 572 864 distinct states: membership after the complete graph, authored owner and rendering schema, conjunctive matcher, final component registry |
| `container` | exit 12, `Sound`: `b` is flattened into a nullable wrapper and its inner non-null branch carries `example: null`; judging only the schema that now carries the copy retains it (path-only provenance) |
| `legacy` | exit 12, `Sound`: `b` is declared before its target `a`; the unregistered target is accepted and an example outside both local domains is retained (pre-fix read-time phase) |
| `reffirst` | exit 12, `Sound`: final timing and authored binding, but the pre-fix reference-first matcher accepts a `Missing` target |
| `readtime` | exit 12, `Complete`: a conjunctive matcher at read time rejects a valid example whose target is declared later |
| `stale` | exit 12, `Complete`: a final phase that resolves through the analysis registry rejects a valid example whose target is a promoted constructor |

Design decisions informed: membership cannot run while reading, because no
read-time choice of missing-target result is both sound and complete; the
phase must resolve through the final component list, not `a.schemas`; the
matcher conjoins the reference instead of returning at it; and a copied
example keeps its authored owner, because the copy target's domain can be
larger than the domain it was authored against. A diagnostic path alone does
not identify that domain. Scalar values only: recursion through child values,
`Bases`, union exact-one, outer constraints narrower than the authored owner,
and closed-union evaluated-member coverage are checked by Go tests. Closure is
held to the existing source-oracle closure profiles
(`singletonClosureProfiles`, `oneOfClosureProfiles`), not proved here.

## External references and decoded values

Membership classifies decoded values, including authored JSON null. A blocked
external example is explicitly `Example.unresolved`, not a nil-valued sample.
The completed-graph filter retains that component edge without classifying it;
the existing bundle diagnostic and selected closure remain authoritative. Copies
retain the unresolved state while rebinding the authored occurrence owner/path.
`TestExampleUnresolvedValueState`, `TestExampleUnresolvedSelectionProvenance`, and
`TestBlockedOccurrenceSelection` cover this boundary. This is outside the model's
decoded-value domain and does not change its membership algorithm.
