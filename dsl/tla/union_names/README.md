# Promoted union branch identity and names

Two different objects can each declare a block `r` with a nested constructor
union in branch `s`. Deriving both promoted type names from `r` and `s` gives
both definitions the same identity. A DSL copy can then substitute the first
definition for the second; generation may compile while rejecting valid values.

`UnionNames.tla` models three definitions, two copies of each, all declaration
orders and all interleavings of declaration and copying. Two definitions prefer
`RS`, the third prefers `RS3`, and an authored type reserves `RS2`.
An existing copied authored type has identity `1`, modeling a user spelling
that matches the first temporary generated ID.

- `legacy.cfg` reproduces loss of definition identity during a copy.
- `untyped.cfg` uses definition counters but fails because an authored identity
  can equal a temporary generated identity in the same memo-key namespace.
- `suffix.cfg` separates copies but fails because suffix allocation takes the
  third definition's preferred name.
- `order.cfg` reserves names but fails because allocation follows declaration
  order rather than stable source paths.
- `checked.cfg` preserves copy identity, reserves names and allocates by stable
  source order. It passes 502 generated states and 244 distinct states.

Run from this directory with a TLA+ tools jar:

```sh
java -cp /path/to/tla2tools.jar tlc2.TLC -config legacy.cfg -metadir /tmp/loom-union-names-legacy UnionNames.tla
java -cp /path/to/tla2tools.jar tlc2.TLC -config untyped.cfg -metadir /tmp/loom-union-names-untyped UnionNames.tla
java -cp /path/to/tla2tools.jar tlc2.TLC -config suffix.cfg -metadir /tmp/loom-union-names-suffix UnionNames.tla
java -cp /path/to/tla2tools.jar tlc2.TLC -config order.cfg -metadir /tmp/loom-union-names-order UnionNames.tla
java -cp /path/to/tla2tools.jar tlc2.TLC -config checked.cfg -metadir /tmp/loom-union-names-checked UnionNames.tla
```

The first four configurations must fail their listed invariants; the checked
configuration must pass. The model abstracts expression graph traversal,
metadata, Go identifier conversion and serialization. Its stable source order
is an input assumption, not a proof of the Go traversal. Direct tests exercise
36 combinations of definition and root order, copied references, reservations
and repeated preparation. DSL and generated transport tests check actual
branch types and runtime values.

The implementation gives each promoted definition a private identity before
any copy, and copies retain it. Copy keys contain the string ID and a separate
definition counter, so an authored name can never impersonate a promoted
definition. DSL evaluation visits types by pointer before stable names exist.
Preparation groups reachable copies by that
identity, sorts their stable source paths, and assigns distinct names while
reserving every preferred and authored name. A finite graph walk terminates
because it visits attributes and types by pointer once. For a finite reserved
set, suffix search terminates because it offers unbounded distinct candidates.
The private counter is not serialized; generated identity uses the stable name.
DSL tests cover an authored name matching either the branch base name or the
old temporary-ID spelling, including forward evaluation before localization.
Direct copy tests check that retaining one definition cannot retain the other.
