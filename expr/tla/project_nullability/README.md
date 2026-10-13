# Projected declaration nullability (#837)

The shared `valueViewPlan.project` object constructor must preserve intrinsic
Nullable while field/collection occurrence copying retains local Nullable.
`project_nullability_test.go` checks root/nested/collection semantic values,
source immutability, both occurrence visitation orders, declaration reuse and
selected-view separation. `project_nullability_dsl_test.go` checks the valid
public `Attributes(Nullable...)` authoring path.

The bounded model explores three projection requests, two named declarations,
two views and ordinary/locally-nullable occurrences. The declaration cache owns
intrinsic nullability and selected-view identity; each occurrence owns its local
bit. Null admission is their disjunction. Source declarations remain immutable.
Repeated requests for one declaration/view reuse the same identity; another view
must remain distinct. Three requests cover both occurrence orders plus reuse.

| Configuration | Result | Generated/distinct states |
| --- | --- | --- |
| current | DeclarationContract counterexample: intrinsic bit discarded | 513/513 |
| promote-occurrence | DeclarationContract counterexample: local bit infects shared declaration | 769/769 |
| view-conflation | SelectedView counterexample: first selected view reused for another | 1041/1041 |
| mutate-source | SourceUnchanged counterexample: local bit modifies source | 769/769 |
| checked | All six invariants pass | 2048/2048 |

Assumptions: source declarations and selected views are valid; projection runs
sequentially. The model abstracts field contents and collection wrapping because
those delegate to the same declaration owner. Direct tests check actual origin
links, selected field shape, semantic JSON null/concrete values and collections.
It does not prove recursion termination, arbitrary expression graphs, physical
Go carriers or transport behavior. Existing recursive projection tests remain
controls. Generated intrinsic nullable physical handling is the separate #834
prerequisite boundary; this semantic repair cannot claim that output compiles.

Run with Corretto 25.0.4.1 and TLC 1.7.4, JAR SHA256
`936a262061c914694dfd669a543be24573c45d5aa0ff20a8b96b23d01e050e88`:

```sh
java -XX:+UseParallelGC -cp "$HOME/.cache/loom-tools/tla2tools-1.7.4.jar" \
  tlc2.TLC -config checked.cfg -metadir /tmp/loom-837-checked ProjectNullability.tla
```

Repeat with each negative configuration and an independent metadata directory.
Negative runs exit 12 with the invariant named above; syntax/type errors do not
count as counterexamples. Preserve sources and concise receipts; delete temporary
checker output after independent review and dependent acceptance complete.

Implementation correspondence: the object constructor copies `rt.Nullable` into
the new AttributeExpr. It does not copy an occurrence bit into that declaration
or alter cache keys. All direct matrix controls pass after this one-line change.
The valid public DSL transport witness advances from semantic conversion panic
to the explicitly retained #834 generated nil-comparison compile failure.
