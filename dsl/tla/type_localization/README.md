# Deferred DSL localization ownership

Generic copying preserves source-node identity; it must not merge distinct
representations because they share a nominal name. The adjacent
[copy model](../../../expr/tla/type_copy/README.md) checks that owner. DSL
localization separately decides which references request local snapshots and
which structural back-references belong to the enclosing authored declaration.

The failing implementation copied every deferred reference once its target DSL
completed. In `X -> localized Y -> X`, that produced a separate X snapshot before
Payload ran. The former nominal copy cache accidentally merged it back into the
customized payload; identity-based copying correctly exposed the earlier loss.
Binding every deferred reference to X is also incorrect: an explicit refined
Node occurrence, including an array/map element, must retain its local copy.

`KeptOwnership.tla` models two independent occurrences, both possible roles,
copying their intent, completion in either order and isolated requirements.
The source/copy correspondence is assumed accurate. It does not model arbitrary
Go graphs, lexical-stack discovery, evaluating user DSL callbacks, pointer
allocation or declaration completion scheduling. Go regressions check those
implementation boundaries; the model establishes only its bounded ownership
invariants. It is not a CI or ordinary-test requirement.

| Policy | Result | Generated / distinct states |
| --- | --- | --- |
| snapshot | StructuralClosure fails | 5 / 5 |
| bind-all | LocalizedIsolation fails | 8 / 8 |
| checked | All three invariants pass | 24 / 16 |

Implementation correspondence:

- `expr.DupKeeping` reports `KeptReference.Source` and `.Copy`; it does not
  choose declaration binding policy. `TestDupKeepingReportsOccurrenceCorrespondence`
  covers repeated source attributes, separate copies and isolated metadata.
- `localizedTypeRoots` identifies local occurrences owned by a builtin
  container without traversing into referenced named declarations.
  `typeEvaluations.keep` preserves explicit localization intent through copies.
- `enclosingType` resolves lexical declaration ownership through nested DSL
  frames. `completeKeptReferences` retains a structural self-reference at that
  owner and snapshots explicit local occurrences only after completion.
- `TestLocalizedStructuralBackReference` proves binding before Payload copies,
  direct/nested paths, two independent local Y occurrences, payload closure and
  required-occurrence isolation. `TestPayloadOwnsDirectSelfReference` covers the
  direct self case. The existing in-progress, declaration-time, nested collection,
  copied-kept-reference and required-occurrence DSL suites remain green.
- Composition consumers only read `.Copy`; their existing explicit node-identity
  owner seed remains unchanged. No nominal coalescing was added to generic Dup.

Run explicitly with TLA+ tools 1.7.4 and Java 25, outside normal gates:

```sh
for mode in snapshot bind-all checked; do
  java -cp "$HOME/.cache/loom-tools/tla2tools-1.7.4.jar" tlc2.TLC \
    -workers 1 -metadir "/tmp/loom-localization-$mode" \
    -config "dsl/tla/type_localization/$mode.cfg" \
    dsl/tla/type_localization/KeptOwnership.tla
done
```

The negative controls exit 12; checked exits 0. Remove temporary checker states
after independent review, preserving these sources and summarized findings.

## Result declaration completion (#1064)

`Completion.tla` adds the preceding scheduling boundary: both `Type` and
`ResultType` must finish their authored DSL before localization snapshots their
fields. The former registry tracked only `Type`, so a result copied during user
type evaluation stayed empty even after the normal result pass completed.
Registering both declarations and executing each in its native DSL context
preserves fields/views and executes the callback once. Existing kept-reference
ownership handles a declaration already running; it must not be replayed.

Run the command above with `Completion.tla` and `completion-current.cfg` or
`completion-checked.cfg`, using separate temporary metadata directories. On
Java 25.0.5 and the same pinned TLC JAR, current fails `CompleteCopies` after
3 generated/distinct states; checked passes both invariants with 13 generated,
9 distinct states. The model assumes atomic callback completion and abstracts
views as declaration content. It does not prove callback validity, recursion,
Go naming or wire conversion. `TestResultTypeLocalizationCompletesDeclaration`
covers direct/array/map copies in both declaration orders, views, requiredness,
metadata isolation and callback count. The recursive-owner test checks the
kept-reference boundary; the localized HTTP specimen checks emitted Go and
OpenAPI field retention. These implementation checks complement the model.
