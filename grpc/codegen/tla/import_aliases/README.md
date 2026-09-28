# Generated import alias allocation

`ImportAliases.tla` models the shared namespace of fixed framework imports,
protocol buffer imports, service imports, and function-local metadata variables.
It covers the name and suffix collisions that motivated #435. The model does
not change package declarations or protobuf wire names.

The old allocator uses each requested name unchanged. `LegacyService.cfg`
reproduces a service named `protojson` colliding with the framework decoder.
`LegacyProtobuf.cfg` reproduces a service named `loom` producing the protobuf
import `loompb`, which collides with the framework's error-message package.
Both configurations must violate `NamespaceSafe`.

The corrected allocator reserves fixed imports, allocates every protobuf
import, then allocates every service import. Each allocation takes the first
unused entry in an ordered candidate list. These finite lists represent
`NameScope.Unique`: protobuf names use numeric suffixes; service names first
use `svc`, then numeric suffixes. A literal service name can equal an earlier
candidate, such as `protojsonsvc`, `protojsonsvc2`, or `loompb2`.

`Allocated.cfg` checks all 720 orders of six adversarial service names.
`NamespaceSafe` requires that allocated names avoid fixed imports, remain
unique across both dynamic import sets, and exactly account for the allocator's
reserved-name set. `EventuallyDone` checks termination under weak fairness.
The model also allocates four metadata variables, including alias and numeric
suffix collisions, for every service. `LocalsSafe` requires that locals avoid
that service's imports and each other. `LegacyLocals.cfg` keeps the corrected
import allocator but leaves imports unreserved in local scopes; it reproduces
the review finding after 15 states. The corrected model checks both invariants
and termination over 14,400 states, with no error. The two original legacy runs
still produce the expected import counterexamples after 31 and 7 states.

Run from this directory with a TLA+ tools jar available:

```sh
java -Xmx512m -cp "$TLA2TOOLS_JAR" tlc2.TLC -workers 1 \
  -metadir "$(mktemp -d)" -config LegacyService.cfg ImportAliases.tla
java -Xmx512m -cp "$TLA2TOOLS_JAR" tlc2.TLC -workers 1 \
  -metadir "$(mktemp -d)" -config LegacyProtobuf.cfg ImportAliases.tla
java -Xmx512m -cp "$TLA2TOOLS_JAR" tlc2.TLC -workers 1 \
  -metadir "$(mktemp -d)" -config LegacyLocals.cfg ImportAliases.tla
java -Xmx512m -cp "$TLA2TOOLS_JAR" tlc2.TLC -workers 1 \
  -metadir "$(mktemp -d)" -config Allocated.cfg ImportAliases.tla
```

The legacy commands exit unsuccessfully by design; the corrected command must
succeed. This is a bounded model of allocation, not a proof of Go rendering,
name normalization, arbitrary suffix depth, or every possible design. The Go
regressions check 48 declaration orders against the actual allocator, emitted
import lists, example stubs and server imports, custom type packages,
compiled CLI union decoding, and CLI/request/header/trailer round trips with
custom body type imports. The local-variable model uses a fixed metadata order
and does not model file-specific custom type aliases; generated Go regressions
cover those bindings. Generation probes compile and vet services named
after every fixed gRPC import. Exact-byte comparisons separately check that
existing designs and protobuf/service declarations keep their output.
