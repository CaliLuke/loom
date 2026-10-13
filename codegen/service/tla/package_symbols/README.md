# Fixed service declarations and authored type names

This bounded model explores #694's allocation ordering. `Missing` models the
current service scope: it allocates user declarations without recording the fixed
framework declarations that renderers will emit. `Reserved` makes those framework
names occupy the same package scope before authored types are allocated.

`Missing` must fail `NoCollision`; `Reserved` must satisfy it and `DistinctTypes`
for all allocation orders of the four modeled authored names. `Client2` checks
that a suffix allocated for Client cannot collide with a subsequently encountered
authored suffix name. Allocation order is not required to give the same mapping:
the generator uses a deterministic design traversal, tested by separate-process
exact output comparison. Reference identity follows the cached NameScope owner;
this model does not separately execute GoTypeRef or Go source rendering.

Implementation seam: service analysis establishes package reservations before
`collectServiceAnalysisData` calls the cached nominal `NameScope` allocator.
Methods and client members must use separate lexical scopes (see refinement
below); sharing this package scope with them was rejected in readiness review.
`serviceSymbols` keeps fixed declarations and reservations under one owner. Conditional symbol admission must reflect the evaluated service's
actual emitted capabilities; the model's Fixed set assumes this inventory is
already correct. Dynamic error and independent interceptor names, import aliases,
external package placement, protobuf symbols and transport members are outside
this finite model and require their own dispositions and executable controls.

From this directory, use the pinned TLC 1.7.4 jar (SHA256
`936a262061c914694dfd669a543be24573c45d5aa0ff20a8b96b23d01e050e88`):

```sh
java -cp /Users/luca/.cache/loom-tools/tla2tools-1.7.4.jar tlc2.TLC \
  -metadir /tmp/loom-694-model-missing \
  -config Missing.cfg PackageSymbols.tla
java -cp /Users/luca/.cache/loom-tools/tla2tools-1.7.4.jar tlc2.TLC \
  -metadir /tmp/loom-694-model-reserved \
  -config Reserved.cfg PackageSymbols.tla
```

Use distinct external state directories; retain results here as a concise receipt
and remove checker output after review. Runtime correspondence starts with
`TestServicePackageSymbolsDoNotCollide` (nine original names) and
`TestServiceReservedSymbolsCompile` (actual generated Go package rejection on the
parent). Final tests cover suffix occupancy, normal names, conditional symbols, external
packages, and transport references. `TestServicePackageHelperReferences` checks
actual Go `VarName` and references (authored `Name` deliberately does not change).
`TestServiceSymbolArtifacts` records a bounded 14-design matrix with a separate
process repeat; `TestServiceSymbolArtifactBuilds` builds/vets those modules.
`TestServiceSymbolGRPCDescriptors` verifies service/client descriptor agreement
when the independent method identity corrects an old gRPC method name.

Initial checker receipt: Missing fails NoCollision in 2 states; Reserved passes
with 39 generated and 20 distinct states. Both ran with the pinned TLC jar and
Amazon Corretto 25.0.4.1. Logs are `/tmp/loom-694-model-{missing,reserved}.log`.


## Namespace ownership refinement

`NamespaceOwnership.tla` exposes the flaw in treating all emitted identifiers as
one namespace. A legal method named Client can coexist with package type Client.
A method Foo and domain type Foo can also coexist, but its package stream helper
FooServerStream cannot coexist with a domain type of that name.

- `Shared.cfg` places fixed package names and methods in one scope; it must fail
  `IndependentMethods` as Client becomes Client2.
- `Split.cfg` gives methods a separate scope but concatenates their stream helper
  without allocating/reserving it in the package scope. It must fail
  `PackageDeclarationsDistinct` when authored FooServerStream is allocated.
- `Owned.cfg` separates method identity and reserves the emitted helper before
  authored types. It must satisfy both invariants for every modeled allocation
  order, including authored FooServerStream2 suffix occupancy.

```sh
java -cp /Users/luca/.cache/loom-tools/tla2tools-1.7.4.jar tlc2.TLC \
  -metadir /tmp/loom-694-namespace-model-Shared \
  -config Shared.cfg NamespaceOwnership.tla
java -cp /Users/luca/.cache/loom-tools/tla2tools-1.7.4.jar tlc2.TLC \
  -metadir /tmp/loom-694-namespace-model-Split \
  -config Split.cfg NamespaceOwnership.tla
java -cp /Users/luca/.cache/loom-tools/tla2tools-1.7.4.jar tlc2.TLC \
  -metadir /tmp/loom-694-namespace-model-Owned \
  -config Owned.cfg NamespaceOwnership.tla
```

Refinement receipt on the same pinned TLC/Corretto toolchain: Shared fails
IndependentMethods in 2 generated/distinct states; Split fails
PackageDeclarationsDistinct in 8 generated/7 distinct states; Owned passes with
21 generated/14 distinct states. Logs are
`/tmp/loom-694-namespace-model-{Shared,Split,Owned}.log`.

Correspondence: `serviceSymbols` now obtains method/member and adapter identities in
separate scopes. Its `allocatePackageSymbols` records method-derived package
helpers before authored types. `buildMethodData` and `buildBaseStreamData` consume
those records; `assignEndpointFields` seeds a client member scope before naming
its fields. Interceptor and CLI renderers consume the same recorded wrappers.
Authentication already has a separate endpoint member scope and retains it. `TestFixedPackageSymbolsDoNotRenameMethods`
checks the reviewer's seven legal names. `TestServiceMemberNamespaceControls`
checks real method/field collisions. `TestMethodAndPackageTypeNamespacesAreIndependent`
reproduces artificial method/type coupling, while
`TestServiceMethodNamespaceControlsCompile` establishes that all three parent
controls compile and vet before correction. The roadmap lists exact observed and
intended identities plus the remaining final acceptance matrix.

Limits: this refinement uses one helper whose base cannot collide with Fixed;
production must use unique allocation for helper/helper collisions. It does not
prove capability classification, wrapper inventories, field/adapter namespaces,
nominal-reference caching or renderer correspondence. Its finite candidate list
and arbitrary allocation orders establish only bounded collision freedom, not
stable spelling across reordered designs. Direct and generated tests must cover
those boundaries, including ordinary byte stability and process-isolated repeat
determinism. Keep independent error, interceptor, protobuf and transport-member
families outside #694 except method-derived consistency explicitly scoped in the
roadmap. These models support the implemented design; they are not proofs of Go
refinement or arbitrary generated-program correctness.


Final checker rerun reproduces the same three expected counterexamples and both
passing state counts. All 14 candidate modules build/vet; seven parent modules
fail at the recorded duplicate declarations, and seven parent controls build/vet.
The 243 generated artifacts repeat byte for byte; all 18 OpenAPI documents are
unchanged. The sole protobuf source change in the coupling fixture is RPC
`FooEndpoint` to `Foo`, inherited from corrected service method identity; field
numbers and messages are unchanged. The readiness assumption of universally
unchanged wire names was corrected explicitly in the roadmap and migration guide.

## Error factory identity (#734)

`ErrorConstructorIdentity.tla` extends this directory's scope to emitted error
factories. Distinct wire names may prefer the same Go spelling. Factory and
nominal declarations share the service package, while repeated method references
must retrieve exactly the previously allocated factory identity.

Run each config with the pinned TLC jar above and an external state directory:

```sh
java -cp /Users/luca/.cache/loom-tools/tla2tools-1.7.4.jar tlc2.TLC \
  -metadir /tmp/loom-734-model-Owned \
  -config ErrorOwned.cfg ErrorConstructorIdentity.tla
```

Replace `Owned` in both paths with `Concatenated`, `Separate` or `Uncached` to
check the rejected alternatives. `Concatenated` fails `DistinctFactories`
(5 generated/distinct states). `Separate` allocates distinct factories in its
own scope but fails `PackageDeclarationsDistinct` (15 generated/13 distinct).
`Uncached` allocates in the package scope on every use and fails `ReferencesAgree`
(31 generated/25 distinct). `Owned` caches factory identities in the shared
package owner and passes all invariants (50 generated/40 distinct). Deadlock
checking remains enabled; completion self-loops. These readiness runs used the
pinned TLC jar and Corretto 25.0.4.1.

The model covers three factory identities, two nominal names, suffix occupancy,
all factory/type traversal orders within their phases and repeated references.
It assumes the emitted-factory inventory is correct. It does not establish
deterministic spelling across reordered designs, arbitrary namespace bounds,
Go rendering or wire error routing. The implementation seam is early
`serviceSymbols` reservation and the same cached identity in `buildErrorInitData`.
`TestErrorConstructorIdentity` and `TestErrorConstructorSharesPackageScope`
cover the direct defects; `TestErrorConstructorIdentityCompile` covers the
original generated duplicate declaration, suffix/type collisions, custom external
errors, ordinary output and mounted wire routing. Parent overlays reproduce
the duplicate declarations; the corrected generated modules compile and vet.
The bounded final matrix and remaining implementation work are tracked in
`roadmap/error-constructor-identity.md` at repository root. Preserve this source,
configs and receipt; remove temporary checker states and logs after final review.

## Public interceptor declarations (#770)

The same package-scope allocation invariant covers public interceptor Info and
accessor types. `serviceSymbols` now records their identities before nominal
user types are named; both side records, declaration placement, wrappers and
examples consume the recorded names. No allocator or state transition changed.
The existing model assumes a complete declaration inventory; it does not establish
which accessor roles a design emits. `TestInterceptorPublicRoles` checks that
inventory, helper/helper and nominal suffix collisions, side sharing and absent
roles. Generated public-role modules check constructor and reference agreement;
existing accessor runtime and located-import controls check the consumers.
