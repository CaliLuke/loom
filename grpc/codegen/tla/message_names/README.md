# Generated protobuf message reservations

Endpoint messages and anonymous messages share the protobuf name namespace.
Previously, endpoint allocation checked anonymous reservations without recording
its own choice. A later anonymous union could reuse a numbered endpoint name.
Type collection then merged two distinct messages by their name-based identity.

The model explores three competing positions, each allocated twice, in every
interleaving. A design name forces the endpoint to choose a numbered fallback.
An anonymous message requests that fallback, and another requests its fallback.
`legacy_unique.cfg` reproduces the collision; `legacy_stable.cfg` reproduces a
repeated endpoint lookup changing its answer after an anonymous allocation.
`shared.cfg` checks uniqueness, stable repeated lookup, and termination with
shared reservations: 47 distinct states and 90 generated states.

Run from this directory:

```sh
java -cp /path/to/tla2tools.jar tlc2.TLC -config legacy_unique.cfg MessageNames.tla
java -cp /path/to/tla2tools.jar tlc2.TLC -config legacy_stable.cfg MessageNames.tla
java -cp /path/to/tla2tools.jar tlc2.TLC -config shared.cfg MessageNames.tla
```

For arbitrary finite allocations with stable owner identities, reservations are
monotonic. A name can be assigned only to its recorded owner or to an owner that
first claims the free name. Distinct owners therefore cannot receive the same
name. Repeated allocation returns its original candidate because all earlier
candidates remain reserved and its own reservation remains available to it.
The implementation's unbounded numeric suffix search terminates when the set of
existing reservations and design names is finite.

The bounded model does not prove string normalization, hash identity, design
shape equivalence, or protobuf compilation. Compatible design-message reuse is
an existing separate rule. Endpoint requests for the same candidate share an
allocation owner; `registerProtoMessage` separately rejects differing shapes. Go tests exercise actual allocation orders and
repeated lookups; generated-module tests compile and round-trip both endpoint
orders. Nested collection wrapper names and explicit protobuf metadata names
have their own allocation contracts and are outside this model.

Run implementation checks from the repository root:

```sh
LOOM_DIR="$PWD" go test ./grpc/codegen -run 'MessageReservations' -count=1
```
