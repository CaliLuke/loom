# Collection declaration identity

`CollectionIdentity.tla` separates declaration ownership from reference
qualification. Two service types and one relocated external type share the Go
base name `Item`. Declaration allocation is explored in either reservation state
and all orders. Each allocated declaration may subsequently be referenced.
Local allocation reserves names even for external types, but the external
physical declaration uses its own package's canonical name, as the current
`NameScope` contract requires.

The `raw` negative control reconstructs qualified references from the authored
base name, reproducing #733. `local-all` instead applies every service allocation
to external references and violates the external declaration contract. `checked`
retains each declaration's physical identifier. Qualification changes only the
package prefix; it cannot choose a different type identifier.

The model assumes a finite collision-free allocator, prior allocation of each
referenced declaration, correct package selection and distinct type identities.
It does not prove the allocator, arbitrary metadata aliases, package declaration
collisions, Go syntax, pointer/presence semantics or transform runtime behavior.
Those remain regression and generated compilation obligations. References before
declaration analysis are outside this contract and must retain existing behavior.

## Run

Use the installed Corretto Java 25 and TLA+ tools 1.7.4 (TLC 2.19), JAR SHA256
`936a262061c914694dfd669a543be24573c45d5aa0ff20a8b96b23d01e050e88`.
From this directory:

```sh
java -cp "$TLA_JAR" tlc2.TLC -workers 1 -config raw.cfg -metadir /tmp/loom-733-raw CollectionIdentity.tla
java -cp "$TLA_JAR" tlc2.TLC -workers 1 -config local-all.cfg -metadir /tmp/loom-733-local-all CollectionIdentity.tla
java -cp "$TLA_JAR" tlc2.TLC -workers 1 -config checked.cfg -metadir /tmp/loom-733-checked CollectionIdentity.tla
```

## Implementation correspondence

- `NameScope.GoTypeName` supplies declaration identity and its external-package
  exception; existing external reservation tests remain controls.
- `NameScope.userTypeDefWithPkgOverride` consumes `goFullValueTypeName` after
  resolving qualification, retaining the same physical declaration identity as
  singular references without introducing a second presence wrapper.
- `collectionTypeRef` is a consumer through the destination context. Arrays,
  maps and nested collections must preserve that identity.
- `TestCollectionDeclarationIdentity` covers ordinary/reserved/colliding/external
  names, qualification and nested collection shapes.
- `TestCollectionDeclarationIdentityGeneratedHTTP` compiles the exact issue
  design; `TestUserTypePackageHTTPModuleCompiles` protects relocated names.
- `TestCollectionAllocatedIdentityOrder`, `TestCollectionNullableValueIdentity`
  and `TestCollectionExternalAliasIdentity` cover allocation order and name
  preservation at presence/alias boundaries.
- `TestCollectionDeclarationIdentityRoundTripHTTP` checks collision/reservation
  payload and result arrays, maps and nested collections through mounted HTTP.

Keep configurations, model and summarized outcomes here. Remove checker states,
raw logs and temporary generation modules after independent review and delivery,
following process checks; retain reusable toolchains.

Readiness results before implementation: `raw` exits 12 after 18 distinct states;
`local-all` exits 12 after 26 distinct states; `checked` exits 0 with 278 generated
and 158 distinct states. Both negatives violate ReferenceIdentity. Direct Go
regressions reproduce the raw-name mismatch and preserve external controls.
