# OpenAPI contract comparison

This model describes the test comparison owner in
`cmd/loom/import_roundtrip_test.go`, not production OpenAPI admission or export.
Issue [#812](https://github.com/CaliLuke/loom/issues/812) has two defects at
the former `inlineGeneratedResponseSchema`: replacing a reference occurrence with its target
discards the occurrence contract, and deleting the target immediately makes a
second reference depend on traversal order.

`ReferenceComparison.tla` has two consumers of one generated component. The
component accepts values 1 and 2 and has a base annotation; an occurrence overlay
restricts it to 2 and supplies a local annotation. Expected and actual overlays
vary independently. Resolution order is nondeterministic.

| Configuration | Correspondence | Result |
| --- | --- | --- |
| `drop-overlay.cfg` | Current target substitution loses occurrence fields | `ComparisonPreservesContract` fails after the first resolution (7 distinct states) |
| `eager-prune.cfg` | Preserving overlays alone still deletes a shared target early | `ReferenceTargetsRemainAvailable` fails on the second resolution (13 distinct states) |
| `retained.cfg` | Resolve detached occurrences from an immutable registry; prune only afterward | Both invariants hold (20 distinct states, depth 4) |

`ComparisonPreservesContract` requires comparison to accept exactly matching
contracts; an extra constraint or changed annotation cannot become a false
success. `ReferenceTargetsRemainAvailable` requires all admitted references to
resolve. `TypeOK` is a type invariant, not a source-detachment proof.

Run from this directory with the existing pinned TLC jar (SHA-256
`936a262061c914694dfd669a543be24573c45d5aa0ff20a8b96b23d01e050e88`):

```sh
java -XX:+UseParallelGC -cp "$HOME/.cache/loom-tools/tla2tools-1.7.4.jar" tlc2.TLC \
  -workers 1 -config retained.cfg -metadir /tmp/loom-reference-retained ReferenceComparison.tla
```

Substitute each negative configuration and a different temporary metadir to
reproduce its counterexample. Do not commit checker states or logs. Recorded
checks used TLC 2.19 and Java 25.0.4.1.

Implementation correspondence is checked by
`TestGeneratedResponseInliningPreservesOccurrenceContract` (separate example,
description, enum, and length cases) and
`TestGeneratedResponseInliningRetainsSharedReferenceTarget`. The generated
symmetry fixture exposes the real exporter reference plus occurrence example.

The model assumes references exist, have no cycles, and overlays have a known
composition. It does not justify merging arbitrary JSON Schema keywords or
discarding annotations. The implementation must reject unresolved, cyclic, or
ambiguous normalization explicitly. Schema-aware traversal, exact numeric token
retention, source detachment, and literal data containing `$ref` require direct
tests. The model does not prove strict importer closure over all exporter output;
that is deliberately not this comparison's contract.

The implemented resolver is `comparisonSchemaResolver` in
`import_comparison_schema_test.go`: `resolve` clones and combines disjoint/equal
occurrence fields, while `prune` runs after all response consumers and computes
remaining reference reachability against the retained registry. Conflicting fields
produce an explicit comparison error; the finite model abstracts only admitted
composable overlays. Direct controls cover source detachment, authored/nested
reference retention, missing/cyclic targets and reversed consumer order.
`prepareGeneratedOpenAPIContract` and `unmarshalComparisonLiteral` own raw token
and typed literal preservation respectively; these are checked by direct tests,
not this state model. No production importer/exporter implementation changed.
