# CLI identifier allocation

This bounded model informed the #486 repair. The aggregate CLI has one shared
namespace for generated imports, usage functions and parser variables. A service
`en` requests the client alias `enc`, which shadows the HTTP parser parameter.
Service `ckRaw` and method `raw` of service `ck` request the same usage-function
and flag-set names. Allocation must reserve fixed names, make derived names
unique, and carry each allocated identifier to all its references.

`Identifiers.tla` explores all 720 orders of six requests, including the literal
suffix competitor `enc2`. Seven finite candidates per request stand for
`NameScope.Unique`; the model does not implement Go name normalization or prove
arbitrary suffix depth. `NamespaceSafe` checks uniqueness and avoidance of fixed
names. `ReferencesAgree` rejects allocating a safe declaration but reconstructing
the old reference. `EventuallyDone` requires completion under weak fairness.
The two expected-failure configurations establish distinct counterexamples;
they are not successful generated-program tests.

Run from this directory with the existing TLA+ tools runtime:

```sh
java -XX:+UseParallelGC -Xmx512m -cp "$TLA2TOOLS_JAR" tlc2.TLC -workers 1 \
  -noGenerateSpecTE -metadir /tmp/loom-486-tlc/legacy \
  -config Legacy.cfg Identifiers.tla
java -XX:+UseParallelGC -Xmx512m -cp "$TLA2TOOLS_JAR" tlc2.TLC -workers 1 \
  -noGenerateSpecTE -metadir /tmp/loom-486-tlc/recomputed \
  -config Recomputed.cfg Identifiers.tla
java -XX:+UseParallelGC -Xmx512m -cp "$TLA2TOOLS_JAR" tlc2.TLC -workers 1 \
  -noGenerateSpecTE -metadir /tmp/loom-486-tlc/allocated \
  -config Allocated.cfg Identifiers.tla
```

Recorded with Corretto 25.0.4.1 and TLC `2026.10.01.024053` (`0dab95e`). The
tools JAR was obtained from the official `tlaplus/tlaplus` v1.8.0 release;
SHA-256: `1d99ab9ad6cf6fb9839dcc7d4a04fd262e136d452edf2a4928c5c18dd4b3468f`.

| Configuration | Expected and observed result | Distinct states | Depth |
| --- | --- | ---: | ---: |
| `Legacy.cfg` | Exit 12: `NamespaceSafe`; `enc` collides with the reserved parser name | 721 | 2 |
| `Recomputed.cfg` | Exit 12: `ReferencesAgree`; declaration `enc2` is referenced as `enc` | 721 | 2 |
| `Allocated.cfg` | Exit 0: both invariants and termination pass | 5,040 | 7 |

The model assumes a complete reservation set and file-local data ownership.
Actual renderer reservations, import/reference correspondence, parser-variable
scoping and isolation between servers require direct and generated compile
tests. It establishes safety for every modeled order, not identical spellings
across reordered DSL declarations. The independent service-package rule that
generated libraries cannot declare `package main` is checked in Go tests.

## Implementation correspondence

| Model obligation | Implementation and regression |
| --- | --- |
| Seed `used` with fixed names | HTTP/JSON-RPC `allocateHTTPCommandIdentifiers` and gRPC `endpointParser` reserve their imports, parser names and dynamic parameters before allocation. |
| Allocate a free final name | `codegen/cli/AllocateCommandIdentifiers` uses `NameScope.Unique` for client aliases, usage functions, flag sets and flag variables. `TestFlagsCodeIdentifiersDoNotCollide` covers competing derived names. |
| Store and reuse references | `CommandData`, `SubcommandData` and `FlagData` retain allocated names; HTTP and gRPC renderers use those fields for declarations and references. Direct payload conversion is rebuilt from the retained payload type and allocated flags. `TestAggregateClientCLIIdentifiersDoNotCollide` covers HTTP and JSON-RPC output. |
| Keep allocation local to a file | The allocator clones command, subcommand and flag data. `TestAggregateClientCLIIdentifiersAreIsolatedPerServer` renders two servers whose import reservations require different aliases. This ownership assumption is tested, not modeled. |
| Keep service libraries importable | `codegen/service/newServiceNameScope` reserves `main`; `TestServicePackageNameAvoidsMainPackage` checks the canonical name. This package rule is outside the model. |

Flag variables use the same allocation operation as modeled flag sets; the
model does not enumerate every payload shape. Generated compilation checks the
actual Go scopes and type references that this abstract name model omits.
`TestCLIIdentifierCollisionGeneratedExamplesCompile` in `codegen/generator`
retains the combined HTTP, JSON-RPC and gRPC regression, including colliding
payload flag names and an authored `main` service. Its design is
`codegen/testdata.CLIIdentifierCollisionDSL`.

## Bounded generated comparison

Compared with parent `aca6696beb3b9af3ec8eb34c7ce6702ef86a0719` using Go 1.27.1
on darwin/arm64. The existing mixed HTTP/gRPC/JSON-RPC design produces 53
identical files, including root example implementations. Its parent module
builds and vets successfully. The adversarial design produces 192 files with
18 intended differences and no added or removed files:

- The three aggregate CLI files change colliding client aliases, usage
  functions, flag sets and payload flag variables, including their references.
- Seven gRPC `c` client/server files and `cmd/cli_identifiers/grpc.go` use the
  receiver-safe service alias; the unused client import is removed.
- Three `gen/main` service files declare `mainsvc`. Their HTTP server, root
  example implementation and two remaining command server files use that name.

The parent adversarial module fails compilation because `gen/main` declares a
program package. The corrected module builds and vets successfully. Direct
regressions separately reproduced the alias and derived-name failures before
implementation. Authored service names, command spellings, paths and protocol
contracts remain unchanged.
