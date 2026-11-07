# Recursive canonical component acquisition

The preliminary expression fingerprint is not a proof of rendered graph
identity: evaluated aliases and copied recursive declarations can represent the
same schema with different finite unfoldings. The nested-promoted imported
fixture demonstrates this without requiring a hash collision. Previously
`reuseEquivalentCanonicalSchema` rendered a candidate before reserving its
reference, reentering the same comparison until stack overflow. Comparing a
completed candidate against an incomplete original would instead reject it.

`canonical_comparison.go` now reserves each candidate reference before traversal
and records its exact rendered-body equality obligation. The outermost
`Analyzer.analyzeSchema` return is the baseline-completion barrier. Each pending
candidate must equal its completed canonical body before a successful return.
The existing same-logical-declaration/same-baseline representation variant rule
runs before this comparison; representation allocation and its graph quotient
remain owned by `representation_registry.go`. Canonical equality still uses
`reflect.DeepEqual`, including annotations and exact reference slots. This does
not broaden equality to arbitrary semantically equivalent JSON Schema forms.

On a projection error or panic, provisional reference entries are removed without
removing other entries under the same fingerprint. The first error/panic is
preserved. A completed incompatible claim retains the existing explicit-name
conflict panic. A failed analyzer is not a reusable successful document builder.

## Model and correspondence

`CanonicalAcquisition.tla` abstracts one repeated differing-fingerprint claim
against an in-progress canonical body. `Equivalent` is an independent oracle for
completed body equality, not a proof of hashing, expression capture, rendering,
annotation ownership or `reflect.DeepEqual`. Multiple claims, graph extraction,
registry rollback, source-type wrappers and representation naming require Go
correspondence tests; the model does not prove those implementation properties.

Using TLA+ 1.7.4 (TLC 2.19), Java 25.0.5:

- Unreserved violates `BoundedAcquisition`, reproducing recursive reentry.
- Early violates `CompletedComparison`, checking before the original completes.
- Owned and Conflict pass bounded acquisition, completion and sound decisions:
  7 generated / 6 distinct states, depth 6 each.

Run explicitly from this directory (not in CI or default test targets):

```sh
java -Xmx256m -cp "$TLA_JAR" tlc2.TLC -workers 1 \
  -metadir "$TLC_STATE" -config Owned.cfg CanonicalAcquisition.tla
```

Repeat Conflict as a positive control and require the named invariant failures
for Unreserved and Early. Keep states/traces outside the repository.

`canonical_comparison_test.go` checks finite-stack accepted and conflicting
self-recursive claims, mutually recursive unfoldings in both registration orders,
first-error preservation, reference rollback and distinct recursive occurrence
closure/validation policy. Its subprocess bounds a regression to 2 MiB of stack.
The maintained importer mixed-inheritance and nested-promoted fixtures exercise
real DSL evaluation, generated compilation, parsed JSON/YAML, exact schema graphs
and byte-identical independent generation. The nullable source-absent recursive
occurrence retains a singleton oneOf around its Node reference beneath its null
wrapper; all constraints and canonical target components remain present.

A shallow root copy was considered and rejected: evaluated DSL graphs already
contain different finite unfoldings, so avoiding an additional copy did not
establish canonical equivalence. The correction leaves expression copying alone.
