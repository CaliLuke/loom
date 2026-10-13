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

## Fixed client receiver reservation

The #486 combined CLI compile probe exposed a missing reservation for the
generated client receiver `c`. An import also named `c` survives Go import
cleanup because receiver selectors look like package selectors, leaving an
unused import in a unary client. It can also shadow a service type reference.
This was outside the original model's fixed-import reservation set.

`Reserved` now includes this fixed receiver. `LegacyReceiver.cfg` isolates
service `c` and reproduces the raw alias collision: `NamespaceSafe` fails after
4 states at depth 4. `AllocatedReceiver.cfg` checks the corrected service alias
`csvc`: both invariants and termination pass over 10 states at depth 10. Run
either with the command above, substituting its configuration name. The four
original configurations were rerun with unchanged outcomes and state counts.
These runs used the [CLI model's pinned runtime](../../../../codegen/cli/tla/identifiers/README.md).

Production `newTransportNameScope` reserves receiver `c` before protobuf and
service alias allocation. `TestGRPCServiceImportAliasAvoidsClientReceiver` checks the
alias and retained payload reference. The combined
`TestCLIIdentifierCollisionGeneratedExamplesCompile` regression compiles the
service `c` with its allocated alias and CLI client import. The model checks
name separation; the generated compile test covers import cleanup behavior.


## Fixed local scopes and example bindings (#691)

The preceding #435/#486 model only reserved fixed imports and receiver `c`
before import allocation. `metadataVarScope` had its own fixed local list.
`FixedLocals.tla` models the missing reverse obligation: service **and protobuf**
imports must avoid fixed function names before dynamic metadata variables are
allocated. The corrected implementation uses `newTransportNameScope` for both import
and metadata allocation.

The new model explores all 120 declaration orders of md, mdsvc, mdsvc2, ctx and
req. Imports allocate protobuf first, service second, with bounded suffix
choices representing NameScope.Unique. Metadata uses the actual service and
protobuf aliases plus a pre-resolved custom import. It checks both directions:
ImportsVisible separates import identities from fixed locals and each other;
MetadataSafe separates dynamic locals from fixed locals, actual imports and
other metadata locals. EventuallyDone checks completion under weak fairness.
Custom import collision resolution is assumed to have completed; production
FileImports/WithPackageNames and the relocated mdsvc fixture check that seam.
This model does not prove arbitrary suffix depth, complete local inventories,
name normalization, Go AST semantics, neutral package names or wire identities.

The `req` witness is a protobuf **import alias** reqpb shadowed by a generated
local. It does not change protobuf declarations and is not #736's message-name
problem. Special Go import identifier `init` is tracked separately in #856.

`ExampleBindings.tla` refines the consumer boundary. The generated example
header already allocates collision-free aliases for server/service/protobuf
roles, but its renderer recomputes qualifiers from the earlier transport names.
A service alias psvc and another service psvcsvr demonstrate the mismatch.
The corrected design retains each final binding in private file data and uses
it for both header and references. This preserves the currently compiling
p/psvcsvr unary example when the shared local owner starts escaping p to psvc.
The existing ordinary/ordinarysvr example independently reproduces the defect.

Use Corretto 25.0.4.1 and the existing pinned TLC 1.7.4 jar, SHA256
`936a262061c914694dfd669a543be24573c45d5aa0ff20a8b96b23d01e050e88`:

```sh
java -XX:+UseParallelGC -Xmx512m -cp "$TLA2TOOLS_JAR" tlc2.TLC -workers 1 \
  -metadir /tmp/loom-691-current -config CurrentFixedLocals.cfg FixedLocals.tla
java -XX:+UseParallelGC -Xmx512m -cp "$TLA2TOOLS_JAR" tlc2.TLC -workers 1 \
  -metadir /tmp/loom-691-owned -config OwnedFixedLocals.cfg FixedLocals.tla
java -XX:+UseParallelGC -Xmx512m -cp "$TLA2TOOLS_JAR" tlc2.TLC -workers 1 \
  -metadir /tmp/loom-691-unreserved -config UnreservedMetadataImports.cfg FixedLocals.tla
java -XX:+UseParallelGC -Xmx512m -cp "$TLA2TOOLS_JAR" tlc2.TLC -workers 1 \
  -metadir /tmp/loom-691-recomputed-example -config RecomputedExampleBindings.cfg ExampleBindings.tla
java -XX:+UseParallelGC -Xmx512m -cp "$TLA2TOOLS_JAR" tlc2.TLC -workers 1 \
  -metadir /tmp/loom-691-owned-example -config OwnedExampleBindings.cfg ExampleBindings.tla
```

| Configuration | Observed result | Distinct states | Depth |
| --- | --- | ---: | ---: |
| CurrentFixedLocals | Exit 12, ImportsVisible fails | 217 | 2 |
| OwnedFixedLocals | Exit 0, invariants and completion pass | 2,400 | 20 |
| UnreservedMetadataImports | Exit 12, MetadataSafe fails | 1,561 | 14 |
| RecomputedExampleBindings | Exit 12, ReferencesAgree fails | 6 | 6 |
| OwnedExampleBindings | Exit 0, invariants and completion pass | 7 | 7 |

Production correspondence: newImportAliases owns transport service/protobuf
bindings; metadataVarScope owns dynamic locals; fileData must carry actual
relocated package names; exampleServer and grpcExampleServerSection must share
the recorded file roles. TestGRPCServiceLocalSymbolsScopeInventory pairs real
rendered scope analysis with the allocator. AliasContracts/DeclarationOrders
check identity boundaries; GeneratedCompile/CustomCompile and the generator's
TestGRPCLocalSymbolsExamplesCompile check real Go compilation. The readiness
plan records the bounded parent/candidate/repeat matrix and actual parent
failure/pass receipts. Keep checker state and raw logs outside the repository.

### Implementation and validation receipt

`transportGeneratedLocalNames` now owns the receiver/parameter/local inventory
alongside `transportGeneratedImportNames`; both `newImportAliases` and
`metadataVarScope` use `newTransportNameScope`. The example header allocator
retains all three role names in private `exampleServerServiceData` and its
renderer uses those names. Host-order tests confirm no cached data mutation.
The HTTP/JSON-RPC owners are unchanged.

Against parent `8ce69db06302f98bc9f131071f9c42bd4b55f6f4`, 479 artifacts from
21 bounded designs were compared. All candidate artifacts match an independent
process repeat exactly; 169 files differ from the parent. Differences are
escaped service/protobuf import qualifiers, related transform/CLI/scaffold
identifier prefixes and comments, custom-import/metadata suffixes, and retained
example bindings. All `.proto`, generated protobuf Go, neutral service files
and ordinary control outputs remain byte-identical. Existing checked-in
encoder/decoder/type/CLI/example goldens remain unchanged.

All 21 candidate modules build and vet. Eighteen parent inputs match the reviewed
readiness artifacts byte for byte, so their recorded builds are reused; three
new suffix/host parents fail at the expected import collisions. Ordinary module,
ordinary full example and the aggregate p/psvcsvr parent control build and vet.
The original six ImportAliases configurations and the five new configurations
were rerun with all expected outcomes unchanged. Go 1.27.2 darwin/arm64 and the
Java/TLC pin above were used. Direct, rendered scope, fixed import/custom type,
metadata round-trip, host-isolation, HTTP/JSON-RPC owner controls and scoped lint
pass. Tests establish emitted Go correspondence, not only model invariants.

Reproduce the bounded artifact comparison with the retained test fixtures.
`LOOM_691_ARTIFACTS=/absolute/directory` preserves each generated module;
`LOOM_691_GENERATE_ONLY=1` renders it without repeating build/vet. Run separate
`go test -count=1` processes for candidate and repeat. Set `LOOM_DIR` explicitly
to the source under test. The generation entry tests are
`TestGRPCServiceLocalSymbolsGeneratedCompile`,
`TestGRPCServiceLocalSymbolsCustomCompile`, and
`TestGRPCLocalSymbolsExamplesCompile`. Leave the generate-only variable unset
for candidate compile/vet acceptance. Compare `.go` and `.proto` bytes, excluding
only module dependency bookkeeping and the separately rendered scope-inventory
fixture. Existing tests exercise fixed-framework alias and metadata wire controls.
