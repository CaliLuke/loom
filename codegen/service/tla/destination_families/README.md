# Destination package families

This extends the existing service package family model to references from two independent service analyses into one generated destination. The current policy records nominal names but skips their destination companion families: `current.cfg` fails DeclarationsDistinct (Choice's kind collides with ChoiceKind). The owned policy allocates each family once in the destination and every service reference reuses it: `owned.cfg` passes declaration disjointness and reference agreement, 47 generated/25 distinct states, depth5. Current/owned ran before production implementation; the current configuration fails DeclarationsDistinct and the owned configuration passes.

Run explicitly (never CI), using the existing pinned TLC jar:

```
java -Xmx128m -cp /Users/luca/.local/share/tlaplus/tla2tools-1.7.4.jar tlc2.TLC -workers 1 -metadir /tmp/loom-destination-model-state -config current.cfg DestinationFamilies.tla
java -Xmx128m -cp /Users/luca/.local/share/tlaplus/tla2tools-1.7.4.jar tlc2.TLC -workers 1 -metadir /tmp/loom-destination-model-state -config owned.cfg DestinationFamilies.tla
```

The model abstracts declarations by physical type identity and bounds names/candidates. `destination_type_names.go` and `destination_type_walk.go` supply those identities, `NameScope.HashedUniqueFamily` supplies atomic family allocation, and `NameScope.SetTypeNames` with `WithTypeNameLocation` supplies cached qualified/unqualified references. It does not prove completeness of graph traversal, Go import qualification, opaque codec handling, template output or every allocation order. Direct and generated tests must establish those correspondences. The admitted raw-inline-union red demonstrates why the inventory must include every emitted declaration, not only expr.UserType nodes. Local declaration families remain under their existing model; no proof tool is added to build/test gates. Temporary state directories were removed after retaining results.

`TestRelocatedTypeFamiliesOwnedByDestination`, `TestRelocatedInlineUnionFamily`,
`TestInheritedDestinationCollectionContext` and `TestNameScopeDestinationIdentity`
cover named and raw families, shared service/alias allocation, inherited wrapper
context, independent physical destinations and opaque type spelling. The maintained
UserTypePackageTransportsDSL fixture compiles the named/inline collisions across
HTTP, gRPC and JSON-RPC. The unchanged local gRPC member-name output remains a
normal byte-comparison control. Used same-service relocation remains rejected by the existing generation policy;
forced declarations in the owning service package remain supported and share its
package-role reservations. Inventory does not authorize an import cycle.

## Physical conversion contexts

`PhysicalLocations.tla` adds independent source and target contexts and a bare
child whose declaration lives in an enclosing destination. The prior alias-only
policy (`location-current.cfg`) fails `ReferenceDeclared`. Carrying physical
location (`location-owned.cfg`) passes: 14 generated / 9 distinct states, depth 5.
Both runs preceded the context propagation implementation. Run explicitly with
the same TLC command above, replacing the config and module names. No proof runs
in contributor gates.

`AttributeContext.ForAttribute` derives the physical scope before generic and
protobuf helper identity, seen keys and body traversal. `Attributor.WithScope`
preserves native, HTTP and protobuf strategies. The protobuf helper clone copies
its embedded TransformAttrs before rebinding either context. Import aliases are
output spellings, while helper identity uses the canonical destination. The model
abstracts these representation details; `TestAttributeContextPhysicalLocations`
and `TestProtoTransformPhysicalContexts` verify independent contexts, retained
concrete strategy, local qualification and unchanged callers. The extended
maintained compile fixture exposed the actual declaration/reference disagreement
for a relocated inline union, and compiles all three transport consumers after
this correction. This does not establish arbitrary projection or admission
semantics, and does not change default materialization policy.

The same physical-location obligation applies at recursive default-renderer
entry: an object-field conversion may supply its enclosing context before a
child transform occurs. `TestRelocatedObjectDefaultContext` uses a valid default
on a relocated object containing an explicitly nullable union with supplied
null, plus an authored discriminator-name collision. It reproduces the wrong
package and unallocated type before the one-line entry binding; the generated
compilation/runtime companion retains null. No default is placed on a whole
union. `TestRelocatedAuthorizationValidation` renders the real Files path with
a wrapped relocated input, named validator and active length predicate. Its
passing result justifies leaving validation production files unchanged; the
native validator uses branch accessors and already-bound union builders supply
physical type contexts. These are bounded correspondence controls, not a proof
of every authorization or default contract.

## Physical emission completeness (DN-1)

Independent review found that one admitted raw union with explicit primitive
representations can appear in two relocated parents. Allocation used physical
path plus type hash, but declaration collection deduplicated by hash alone and
omitted the second package. The updated DestinationFamilies model adds emission
identity independently of allocation: `emission-current.cfg` retains correct
allocation and fails EmittedDestinationsComplete with only the left declaration;
`current.cfg` still reproduces the earlier family collision. `owned.cfg` checks
both obligations and passes 279 generated / 100 distinct states, depth 7. These
runs preceded the DN-1 implementation. Prior model results remain historical
findings above; this expanded model supersedes their state counts.

`destinationTypeIdentity` now keys both the read-only destination name registry
and native union collection/visited state. An implicit service location uses its
actual service path, so spelling location implicitly or explicitly cannot create
two identities. Equal Go names sort by physical path then type hash. The model
abstracts ordering; `TestRelocatedSharedRawUnionDestinations` checks reversed
reachability order and alias reanalysis, while
`TestUnionDestinationIdentityNormalizesLocalPath` checks physical normalization.
The maintained transport fixture contains the same represented-primitive raw
union under left and right parents and compiles both declarations. Named-type
collection remains ID-based: promoteSharedTypeLocations rejects conflicting
inherited named destinations and stamps accepted locations; the confirmed case
has no promoted named branch and does not justify changing that admission rule.

## Generation-wide declaration emission (DN-2)

Allocation and service-local collection do not guarantee that a declaration is
emitted once across services. `DeclarationEmission.tla` counts emitted occurrences,
including service and custom files, rather than hiding duplicates in an output
set. `declarations-current.cfg` models the prior file-keyed ledger, package-blind
local name set and untracked service file. The independent negative controls
`declarations-file-current.cfg`, `declarations-local-current.cfg` and
`declarations-service-current.cfg` isolate duplicate emission, missing declarations
and the service-file bypass. All four fail uniqueness or completeness.
`declarations-owned.cfg` passes uniqueness, completeness and valid placement:
95 generated / 40 distinct states, depth 6. All ran before implementation.
Run explicitly with the command above, substituting the config and
`DeclarationEmission.tla`; proof execution remains outside delivery gates.

`service.go` owns one selection path for type sections in both service.go and
custom files. The existing generation-wide `userTypePkgs` ledger keys physical
package directory plus allocated declaration role. Local seen state uses the same
pair; complete-family allocation makes these roles injective within a package.
Internal file paths are consistently relative to gen; only output construction
adds Gendir. Thus an authored gen/foo package retains that entire path.
Service scaffolding and initialization are not shared declarations.

The first existing file candidate in evaluated service traversal order owns a
shared declaration. Root finalization sorts services; direct callers can also
supply another traversal order. Custom paths and section names are sorted.
Changing traversal order may change placement, but must preserve one complete
physical declaration. The model explores contribution orders and proves only
valid placement, not byte equality across different input orders. Fixed-input
process-isolated artifact comparisons establish determinism.

`TestDestinationRawUnionEmission` exercises custom/custom and service/custom
sharing, both caller traversal orders, and fresh ledgers over cached semantic
data. `TestDestinationNominalEmission` covers equal allocated names in distinct
packages. `TestDestinationAuthoredGenPath` covers the admitted path-prefix trap.
`TestDestinationEmissionGenerated` compiles all six corresponding designs;
the maintained transport fixture uses distinct parents/services in one common
package, compiling HTTP, gRPC and JSON-RPC references. DSL service declarations
alone cannot reverse finalized order; tests reverse a detached traversal slice.
The model abstracts file rendering, imports, actual allocation and DSL admission;
these tests and the existing family model supply the corresponding evidence.

## Allocated declaration roles (DN-3)

The generation ledger assumes that every producer supplies an allocated Go
identity. Independent review found that error collection checked VarName but
inserted authored Name. Foo-Bar payload allocates FooBar; FooBar error allocates
FooBar2. An authored FooBar ledger claim suppressed the payload, and the merged
generated module failed compilation. A shared foo_bar error/payload alone exposed
duplicate sections but was coalesced by the artifact merger; it is a seam control,
not the consumer-failure proof. The ordinary Foo/foo control already compiles.

`DeclarationRoles.tla` distinguishes authored spelling from allocated type and
receiver identities. `roles-current.cfg` fails Unique/Complete;
`roles-owned.cfg` passes 82 generated / 32 distinct states, depth 6. Both ran
before the correction. Run explicitly using the earlier TLC command with these
config/module names. The model retains separate type and method roles and assumes
physical allocation injectivity; it does not prove the section producers use it.

`collectErrorTypeSections` now uses `maybeAddNamedTypeSection` with VarName, and
its local error traversal key includes physical package plus that name.
`buildErrorSections` uses the same allocated receiver for its method role.
Method payload/result/stream declarations already supply allocated MethodData
names; nominal types, union families and named validators supply VarName,
UnionTypeData.Name and ValidateData.Name respectively. Authored descriptions and
wire error-name selection stay under the unchanged error method renderer.

`TestDestinationErrorRoles` checks one declaration and one error-method family
in both traversal orders for collided, shared and ordinary roles.
`TestDestinationErrorRolesGenerated` uses actual Service/Transport generation,
mergeFilesByPath, rendering and Go compilation/runtime assertions. It verifies
Error() returns the authored message and LoomErrorName() returns authored_failure
in every case. This establishes producer/ledger correspondence without changing
admission, wire semantics or the artifact merger.
