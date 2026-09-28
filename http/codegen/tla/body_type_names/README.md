# HTTP body type identities and names

An endpoint named `other` can contain an authored `Other` type. Both can
produce `OtherRequestBody`, despite representing different shapes. Three
registries and caches must keep them distinct:

1. Expression copies use structural keys. A named union request wrapper needs
   an endpoint-owned ID and a separate transport-wrapper category, or an
   authored name equal to its ID can become a recursive reference to the union
   wrapper. Copies preserve this category.
2. Go transport registries use name hashes. Endpoint body names must be
   allocated separately from nested type names before registering layouts,
   declarations, validators and conversions.
3. Example memoization separates wrapper and authored identities too. Sharing
   an ID must not reuse a value of another shape or mistake a branch for a
   recursive reference.

`CopyIdentity.tla` reproduces the first failure in `copy-legacy.cfg` and
also rejects the initial string-UID candidate in `copy-string-uid.cfg`: an
authored `svc#Other` branch acquires that same ID after suffixing. The checked
configuration separates the wrapper category in the copy key and passes six
generated states and four distinct states. This small model abstracts the
copy memo table to the identity comparison that selects its entry; expression
and generated runtime tests check the actual copied branch shape and value.

`ExampleIdentity.tla` checks both generation orders for an authored type and
wrapper sharing one string ID. The legacy key returns the other type's value.
The checked category key passes six generated and four distinct states. This
model abstracts values to their owning type; expression tests additionally
cover recursive reservation, copying and the public authored-type cache API.
A rendered OpenAPI test preserves the canonical request body's union example.

`BodyTypeNames.tla` explores three distinct body identities, three preferred
names, every subset of nested names, and every allocation order. The legacy
configuration loses a nested type. The checked rule reserves all existing
names before allocating fresh names to colliding bodies; it passes 3,876
generated states and 2,580 distinct states, with distinct bodies, preserved
nested names and termination.

Run with a TLA+ tools jar:

```sh
java -cp /path/to/tla2tools.jar tlc2.TLC -config legacy.cfg -metadir /tmp/loom-body-legacy BodyTypeNames.tla
java -cp /path/to/tla2tools.jar tlc2.TLC -config checked.cfg -metadir /tmp/loom-body-checked BodyTypeNames.tla
java -cp /path/to/tla2tools.jar tlc2.TLC -config copy-legacy.cfg -metadir /tmp/loom-body-copy-legacy CopyIdentity.tla
java -cp /path/to/tla2tools.jar tlc2.TLC -config copy-string-uid.cfg -metadir /tmp/loom-body-copy-string CopyIdentity.tla
java -cp /path/to/tla2tools.jar tlc2.TLC -config copy-checked.cfg -metadir /tmp/loom-body-copy-checked CopyIdentity.tla
java -cp /path/to/tla2tools.jar tlc2.TLC -config example-legacy.cfg -metadir /tmp/loom-body-example-legacy ExampleIdentity.tla
java -cp /path/to/tla2tools.jar tlc2.TLC -config example-checked.cfg -metadir /tmp/loom-body-example-checked ExampleIdentity.tla
```

All legacy configurations and the string-UID candidate must fail their
preservation invariant; all checked configurations must pass. The models are
bounded checks, not proofs of the full generator. Names abstract Go normalization and numeric suffixes;
distinct model roots represent distinct identities after coalescing copies.
They do not model recursion, metadata, views, wire decoding or Go syntax.
Direct allocation tests cover copied identities, anonymous collections and
occupied suffixes. `TypeIdentityDSL` and its runtime harness cover object,
named union and WebSocket bodies through generated code.
