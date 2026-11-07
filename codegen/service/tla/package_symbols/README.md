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
Preserve this source, configs and summarized results; remove temporary checker
states and logs after final review.

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


## Nominal declaration families

`TypeFamilies.tla` models nominal types whose declarations also emit union kind
names, branch constants and constructors, or validators. The old single-name
allocation reserves only the nominal identifier. A later authored type can then
claim an already implied companion. The model permits every allocation order of
five identities and six candidate spellings. `families-current.cfg` reproduces
`DeclarationsDistinct` failure after allocating `Choice` and `ChoiceKindCat`
(7 distinct states; a three-state trace). `families-owned.cfg` claims the complete
family atomically and passes (119 generated states, 55 distinct, depth 6).

Run explicitly from this directory with the existing pinned TLC installation:

```sh
java -cp /Users/luca/.local/share/tlaplus/tla2tools-1.7.4.jar tlc2.TLC \
  -metadir /tmp/loom-service-type-families-owned \
  -config families-owned.cfg TypeFamilies.tla
```

Use `families-current.cfg` to reproduce the original failure. These are opt-in
design checks, never CI or build prerequisites. The recorded runs used TLC 2.19
and Java 25.0.5. Keep checker state outside the source tree and remove idle state
after review.

`NameScope.SetTypeFamilies` installs the physical package owner's inventory;
`HashedUniqueFamily` claims it at the first nominal reference, before that name
is cached. This handles late synthesized and recursively projected types without
a separate projection/name prepass. `serviceTypeFamilies` inventories only the
validators emitted for named object or collection descendants of used untagged
unions; plain unused validator roles do not reserve identifiers. `viewTypeFamilies`
includes each projected named type's actual view validators. Both use
`unionnames.PackageDeclarations`, with native Any aliases added by their owning
service layer. Union builders take the actual nominal identity and consume the
same kind/branch declarations instead of allocating kind names a second time.
Receiver members retain their independent namespace.

The regression tests `TestServicePackageTypeFamilies` and
`TestViewPackageTypeFamilies` render service/view declarations and reject duplicate
top-level identifiers. Before correction they reproduce `NewChoiceCat`,
`ValidateCat`, and `NewChoiceViewCatView` collisions; the ordinary control remains
green. Existing identity, member-sanitization, null, Any representation and fixed
package-symbol tests cover the surrounding boundaries. The combined HTTP naming
specimen owns generated module compilation and artifact comparison.

The model assumes a complete emitted-family inventory and does not prove the Go
walker, arbitrary string/suffix bounds, imported package names, deterministic
spelling across different authored orders, or generated program compilation.
Located types retain canonical external names and do not compete in the service's
local package scope; ordinary located import controls remain part of validation.
The model's bounded disjointness result is not a claim about unrelated generator
namespaces or arbitrary user custom codecs.

### Intrinsic native alias collisions

`NativeAliases.tla` checks the declaration family's internal disjointness, which
`TypeFamilies.tla` assumes when representing each family as a set. Native Any
branches named `Kind` or `KindCat` can collide with their union's kind type or
Cat constant. For primary `New`, `NewCat` can collide with the Cat constructor.
Changing the primary suffix cannot repair these internal overlaps.

`aliases-current.cfg` reproduces `IntrinsicallyDistinct` failure (3 generated and
distinct states, depth 2). `aliases-owned.cfg` reserves common declarations,
retains all available natural aliases, then allocates suffixes for deferred
aliases. It passes disjointness and natural-name preservation (406 generated,
140 distinct states, depth 11) with the same pinned TLC/Java installation.
Run either config explicitly with `NativeAliases.tla` and an external `-metadir`;
these are design checks only. The finite model does not establish deterministic
spelling or the Go emitter; its arbitrary allocation order is stronger only for
collision freedom. Production sorts authored branches for stable allocation.

`nativeTypeDeclarations` computes this complete native inventory once per query;
the scope callback and service/view emitters consume that same policy. A local
NameScope supplies suffix semantics after all common roles and validators have
been reserved. `TestNativeUnionAliasFamilies` reproduces both primary cases in
both emitters, checks suffix occupancy and reverse branch traversal. Ordinary
noncolliding alias spellings remain unchanged; formerly conflicting aliases use
an available suffix. Logical branch names and wire tags are unchanged.

### Logical view names and helper identity

`ViewSuffixes.tla` applies the same intrinsic-disjointness obligation to distinct
logical views whose Go spelling coincides. Views `tiny-view` and `tiny_view` are
both admitted. Plain sanitization emits duplicate validators and projection
helpers; `tiny_view2` checks natural suffix occupancy. Run the existing
`aliases-current.cfg` or `aliases-owned.cfg` against `ViewSuffixes.tla`. Current
fails in 5 generated/distinct states (depth 3); owned passes in 20 generated,
14 distinct states (depth 6). These checks ran before the implementation change.

`viewHelperSuffixes` owns a deterministic mapping from authored view names to
helper suffixes, using NameScope after claiming natural names. The default view
owns the empty suffix. View validator families, declarations, nested validator
calls, and both directions of projection consume this mapping. `ViewData.Name`
keeps its authored logical value for wire dispatch; the private helper suffix
only controls Go symbols. The natural `TinyView2` remains unchanged while the
second colliding `TinyView` becomes `TinyView3`.

`TestViewSanitizedTypeFamilies` renders both packages, checks declaration
uniqueness and verifies that each logical switch case invokes its allocated
validator, projector and result constructor. `TestViewHelperSuffixIdentity`
checks reversed view order and nested validation calls. Existing views golden,
projection parity, selected-view carrier and customized-result tests cover
ordinary output stability. The model is bounded to these three nondefault names;
it does not establish arbitrary sanitization or cross-family/package collisions.

### Viewed service helper roles (G1)

`ViewedHelpers.tla` removes the previous inventory assumption for service-package
projection helpers, reconstruction helpers and both viewed wrappers. Five small
scenarios cover each authored/helper name collision (`ProjectFoo`, `NewFoo`,
`NewViewedFoo`, `NewFooFromFooView`) and `Foo/bar` versus `FooBar/default`, with
an occupied natural suffix. Nominal and helper allocation can interleave; cached
references must remain stable after other declarations allocate names.

```sh
java -Xmx256m -cp /Users/luca/.local/share/tlaplus/tla2tools-1.7.4.jar tlc2.TLC \
  -workers 1 -metadir /tmp/loom-viewed-helpers-current \
  -config viewed-helpers-current.cfg ViewedHelpers.tla
java -Xmx256m -cp /Users/luca/.local/share/tlaplus/tla2tools-1.7.4.jar tlc2.TLC \
  -workers 1 -metadir /tmp/loom-viewed-helpers-owned \
  -config viewed-helpers-owned.cfg ViewedHelpers.tla
```

Current fails `DeclarationsDistinct` with authored and projected `ProjectFoo`
(17 generated/distinct states, depth 3). Owned passes declaration disjointness
and cached-reference stability (130 generated, 66 distinct, depth 6), using the
same pinned TLC installation and Java 25.0.5. Source/model drafts were prepared
under a disk-related execution hold; these runs became possible after cleanup,
before the candidate was frozen. This is not a claim that Go regression tests
ran under that hold. Proof tools remain opt-in design aids.

`viewedHelperKey` distinguishes role, source declaration, target declaration and
selection in the service NameScope. `allocateViewConversions` waits until actual
projected names exist, then allocates projector and reconstructor names; nominal
references never need to be renamed to accommodate late helpers.
`allocateViewedWrappers` owns wrappers separately by physical occurrence layout.
All constructors, view dispatch and nested conversion helper declarations consume
those cached identities. The conversion transform's former private name scope
has been removed; its actual emitted helpers also occupy the service package.

`TestViewedHelpersShareServicePackage` renders the four authored-helper cases,
the cross-result case and an ordinary control, and checks each wrapper's calls
against its conversion declarations. `TestViewedHelperIdentityAfterProjectionNaming`
checks late projected suffixes, existing helper occupancy, repeated use and
occurrence-independent conversion identity. `TestViewedWrapperOccurrenceIdentity`
checks shared explicit-view wrappers and distinct nullable layouts. The existing
nullable helper control now queries the same allocated record. Execution of
these Go tests and generated compilation was pending when this source revision
was prepared; the integration packet records the final attributed results.

The model bounds names and independent roles; it does not prove semantic hashing,
projection transformation, arbitrary string collisions or emitted program
correctness. Ordinary names are preserved when available; colliding helpers gain
suffixes without changing authored view strings, wire names or physical types.

### Viewed envelope declaration ownership

The compiled cross-result `Foo/bar` and `FooBar/default` witness exposed a second
physical identity defect: metadata map construction allocated a source result
in the views scope, reserving validators for views that its envelope never emits.
Its reference became `FooBar2`, while the envelope declaration and validator used
the service scope's `FooBar`. `ViewedEnvelope.tla` reproduces this mismatch:
`viewed-envelope-current.cfg` fails `ReferenceDeclared` (10 generated/distinct,
depth 4); `viewed-envelope-owned.cfg` passes reference/declaration agreement,
metadata map ownership and bounded role-key disjointness (15 generated,
12 distinct, depth 4). Run either config explicitly against `ViewedEnvelope.tla`
with the same pinned TLC jar, `-Xmx256m -workers 1` and an external `-metadir`.
The core model ran before production changes; the final rerun adds the explicit
physical-key disjointness obligation. These commands remain opt-in design aids.

`allocateViewedEnvelope` owns a plain named physical wrapper with one validator.
It does not reuse or mutate the source ResultType. `UserTypeExpr.Hash` uses
TypeName alone, so changing only UID cannot distinguish source and physical
roles. The envelope therefore uses a quoted role/source-ID/source-name/layout
identity ending in `)`, registered with `HashedUniqueFamily` using the preferred
public Go name before any reference. Projected nominal types are renamed by
`collectProjectedTypes` and end in `View`; raw unions have a distinct hash prefix.
Thus role disjointness follows the physical naming grammar, not an unreserved
preferred-name prefix that authored names could also use. The bounded model's
adversarial names illustrate this separation; it does not prove every Go
registration path satisfies that grammar. Review and direct tests cover those
implementation seams.

Metadata maps now have independent cached role identities in the views scope;
constructing map data never registers a phantom source type there. Envelope
VarName, FullName, FullRef, validator name/parameter and constructor target all
come from the views owner. Projected type names in helper documentation also use
the views scope, avoiding an accidental service-package registration. Map records
carry their allocated name rather than asking a renderer to concatenate it.
Authored source view strings and ordinary public Go spellings remain unchanged.

`TestViewedEnvelopeDeclarationIdentity` renders cross-result, `Foo`/`FooView`,
metadata-map/type, and ordinary controls, checking declarations against qualified
references, local validator parameters and constructor targets. It rejects both
raw and Go-sanitized internal identity leakage. `TestViewedEnvelopePhysicalIdentity`
checks distinct physical hashes, repeated/copy references and the actual single
validator family. The existing G1 wrapper-to-projection assertion now follows the
recorded Projected field's identity, rather than assuming a name concatenation.
Go execution was coordinated separately; the frozen increment records which
checks have actually run. Preserve the compiled red witness until final review.
