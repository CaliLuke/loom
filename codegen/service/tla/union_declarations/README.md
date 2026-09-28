# Union declaration identity

References use `NameScope` names keyed by the referenced type's hash. The old
service and view collectors instead keyed named unions by their underlying
union's hash. A raw constructor union and a promoted block branch can share
that underlying union while referring to different declarations. Collecting
either first suppressed the other and left an undefined Go type.

`UnionDeclarations.tla` models a raw union, two named wrappers of the same
underlying union, and one independent union. Each occurs twice. It explores
every collection order, checking that every visited identity has exactly one
declaration and that collection terminates.

```sh
java -cp /path/to/tla2tools.jar tlc2.TLC -config legacy.cfg -metadir /tmp/loom-unions-legacy UnionDeclarations.tla
java -cp /path/to/tla2tools.jar tlc2.TLC -config checked.cfg -metadir /tmp/loom-unions-checked UnionDeclarations.tla
```

Run from this directory. The legacy configuration must fail
`EveryReferenceDeclared`: collecting the raw union then the first wrapper
leaves the wrapper undeclared. The checked configuration passes 1,026 generated
states and 256 distinct states.

The unbounded argument is by induction on visits: collecting a new reference
identity inserts its declaration, and collecting an existing identity reuses
it. Distinct reference identities cannot suppress one another. This assumes
the hash and name allocation contracts of `NameScope`; the model does not
prove those contracts or the generated Go code. Direct tests compare real
scope references with service and view declarations in all six orders of the
three shared-union identities, including repeated visits. Generated fixtures
provide the compile and transport checks.
