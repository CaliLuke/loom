# Projected union declaration ordering

Two different raw unions can both prefer `Block` in one views package. A
qualified reference uses the allocated local name if available, or the base
name otherwise. Rendering a conversion before allocating its declaration can
therefore refer to `Block` when the declaration later receives `Block2`.

`ViewUnionNames.tla` explores both declaration orders and every interleaving
with conversion references. The legacy configuration reproduces a mismatched
reference. The checked configuration requires allocation before a reference
and passes 19 generated states and 13 distinct states, including termination.

```sh
java -cp /path/to/tla2tools.jar tlc2.TLC -config legacy.cfg -metadir /tmp/loom-view-names-legacy ViewUnionNames.tla
java -cp /path/to/tla2tools.jar tlc2.TLC -config checked.cfg -metadir /tmp/loom-view-names-checked ViewUnionNames.tla
```

The legacy run must fail `ReferencesMatchDeclarations`; the checked run must
pass. This finite model assumes stable type identities and abstracts traversal,
metadata and Go rendering. The implementation reserves raw union names after
their branches are projected, before generating conversions for the enclosing
result. `TestViewUnionReferencesUseDeclaredNames` checks that order directly;
the generated views and streaming harness compiles and converts values through
two differently shaped unions with the same block and branch names.
