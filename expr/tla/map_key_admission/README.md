# Map-key admission — #702

This design model isolates validation context: one declaration is used as a
normal value and as a map key, in both visitation orders. Framework-owned slice
storage is valid data but invalid as a key. Scalars and object pointers remain
valid keys. An opaque representation is assumed valid and externally owned;
the model does not establish comparability of arbitrary authored Go types.

The current owner has no key admission. A primitive-only repair erases documented
object-pointer and custom representations. A check hidden behind a global
already-validated-type cache misses a later key use. The selected owner checks
map occurrences independently of child declaration validation.

Run each configuration with Java25 and TLC1.7.4 (jar SHA256
`936a262061c914694dfd669a543be24573c45d5aa0ff20a8b96b23d01e050e88`):

```sh
java -cp /Users/luca/.local/share/tlaplus/tla2tools-1.7.4.jar tlc2.TLC \
  -workers 1 -metadir /tmp/loom-702-model-owned \
  -config owned.cfg MapKeys.tla
```

Use the matching configuration/metadir for `current`, `primitive-only`,
`global-cache` and `owned`. Negative variants must exit12 with an invariant
counterexample; the owned candidate must exit0. Deadlock checking stays enabled.
Results: current fails KeySafety at19 distinct states; global-cache fails
KeySafety at19; primitive-only fails PreserveContracts at21. Owned passes with
32 generated/24 distinct states. These are bounded model checks, not proofs of
all generated Go or custom codecs.

`RootExpr.Validate` owns service Go-representation admission after preparation
and before finalization/capture. Its traversal classifies each map's selected
key with `nonComparableMapKeyStorage` before visiting the key declaration. It
includes unused types and errors, and does not duplicate admission in ordinary
attribute validation. Graph memoization skips only already-visited occurrences;
it does not hide a new map use of an already-seen ordinary key declaration.
`TestMapKeyAdmissionIsOccurrenceScoped` checks both visitation orders.
Native/named slice rejection and retained
object/custom contracts are covered by the other direct and actual CLI tests.

Limits: this model omits alias resolution, Go type syntax, examples, JSON codecs,
protobuf's narrower key domain, and arbitrary graph recursion. It establishes
ordering properties only under the listed representation assumptions. Compiler
and generator tests check the actual representation boundary. The separate
custom-Bytes example synthesis panic is recorded in the roadmap, not hidden by
rejecting a comparable authored representation.
