# Canonical body component binding

This companion explores the explicit claim selection missing from the recursive
reservation model. It is design-only, not proof of a production implementation.
The maintained two-explicit/one-implicit JSON witness has two wrong suffixed
canonical refs while all three method media-root examples remain correct.

A recorded canonical owner and its actual annotation binding are separate from
each request's prepared component samples. Explicit ordinary claims conditionally
select the recorded owner before component materialization; exact complete
candidate comparison validates that choice. Method media examples and authored
reference-local overlays remain independently attached. An actual representation
projection can differ while inheriting baseline binding; an incompatible authored
canonical shape rejects. A fresh noncanonical allocation selects its actual
allocated context, not a prior logical declaration's sample context.

The request captures wire/shape/plan/binding inputs before deferred construction.
An unrelated restored global scope cannot replace them. A selected reference is
committed only after validation. These refs are internal claim decisions, not
successful external document publication; the companion's final conflict aborts
that document. The original reservation model owns complete graph barriers and
failure rollback. It must be composed with this selection protocol in Go.

Run explicitly with installed TLA+1.7.4/TLC2.19, Amazon Corretto25.0.5, outside
ordinary build/test/CI gates:

```sh
java -Xmx256m -XX:+UseParallelGC -cp "$TLA_JAR" tlc2.TLC -workers 1 \
  -config Owned.cfg -metadir "$TLC_STATE" CanonicalBodyBinding.tla
```

Owned checks ten invariants and weak-fair completion:50 distinct/52 generated
states, queue empty. Current's premature independent prepared-component binding
fails CanonicalOrdinary at9 states. SortFirst still fails the second canonical
claim at17 states. Negative controls reject forcing one name for different wire
representations, overwriting media roots, accepting authored conflict, replacing
fresh allocated binding by a logical prior, reading restored global wire/plan,
committing before validation, and deleting an authored local overlay. Exact
outcomes and tool/model hashes are kept in the external qualification packet.

Limits: one already recorded canonical owner, two explicit claims in both orders,
one actual implicit reuse, one distinct representation, one fresh allocation,
then one incompatible claim. Shape compatibility, selected binding and actual
representation effects are finite fixture inputs, not algorithms inferred from
names or codec labels. Graphs here abstract complete component contracts by
shape/wire/annotation tokens; recursive traversal, actual schema equality, nested
ExampleSet positions, custom codecs, hash extraction and Go call sequencing are
not proved. Ordinary JSON is a fixture result, not a blanket reuse shortcut.
The media examples are distinct independent tokens; count/equality does not pick
component authority. The model permits no final-name ordering fix to substitute
for canonical acquisition selection.

Implementation correspondence: `componentName` selects the recorded canonical component acquisition before synthetic wrapper annotations when the captured logical/baseline and semantic contract select that owner. Other canonical claims use `acquireCanonicalComparison`, whose live candidate is compared only after held completion. Both paths bind a contributor root before candidate traversal. Complete wire projections remain separate from that baseline selection. `TestDeclaredBodyCanonicalAcrossMethods`, `TestBuildBodyTypesReusesTypeOfDeclaredBody`, canonical recursive/conflict tests and the actual annotation-owner controls cover these seams. Suffixed representation claims additionally require the exact prepared binding described in [CLAIM-BINDING.md](../projected_acquisition/CLAIM-BINDING.md); authored canonical metadata alone is insufficient. These implementation tests do not enlarge the finite model's proof scope.
