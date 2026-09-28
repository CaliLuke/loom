# Type names in external packages

A service method and an authored type can both be named `Moved`. If the type
has `struct:pkg:path`, its declaration belongs to that package, not the service
package. The old declaration path consumed the service's name scope and could
produce `Moved2`, while qualified references still used `types.Moved`.

`ExternalTypeNames.tla` models two services, every subset of reservations for
`Moved` and `Moved2`, and both declaration orders. `legacy.cfg` reproduces the
unresolved reference. `checked.cfg` uses the canonical type name independently
of service reservations while retaining those reservations for helper names. TLC checks
96 generated states and 64 distinct states, including agreement between shared
package declarations, preserved reservations and termination.

`SharedHelpers.tla` reproduces a rejected candidate that stopped reserving
relocated names. Its generated union discriminator then steals an authored
`ChoiceKind` name. The checked candidate retains the prior reservations before
returning the canonical declaration name. For two services, every subset of
authored names and existing reservations, and both declaration orders, TLC
checks 384 generated states and 256 distinct states. Integer names represent
`ChoiceKind`, `ChoiceKind2`, and further available suffixes.

Run with a TLA+ tools jar:

```sh
java -cp /path/to/tla2tools.jar tlc2.TLC -config legacy.cfg -metadir /tmp/loom-external-types-legacy ExternalTypeNames.tla
java -cp /path/to/tla2tools.jar tlc2.TLC -config checked.cfg -metadir /tmp/loom-external-types-checked ExternalTypeNames.tla
java -cp /path/to/tla2tools.jar tlc2.TLC -config helpers-local.cfg -metadir /tmp/loom-external-helpers-local SharedHelpers.tla
java -cp /path/to/tla2tools.jar tlc2.TLC -config helpers-checked.cfg -metadir /tmp/loom-external-helpers-checked SharedHelpers.tla
```

The legacy configuration must fail `ReferenceResolves`; the checked
configuration must pass. The local-helper candidate must fail
`AuthoredNamesPreserved`; the checked helper model must pass. These are bounded
models, not proofs of the generator. Names represent normalized Go names.
Import aliases, exact helper-name agreement between services, wire encoding and
Go syntax are outside their scope. NameScope tests cover all six supported type
shapes; generated fixture compilation checks service and transport references,
including an authored type that shares a union discriminator's preferred name.
