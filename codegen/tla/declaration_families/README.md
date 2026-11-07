# Atomic package declaration families

The naming regressions in HTTP query carriers and native nested unions have one
cause: a primary type was assigned before its emitted helper family was claimed
in the same Go package. A later authored type could become an existing validator,
kind constant or constructor; fixed transport names were absent from the scope.

`PackageSymbols.tla` models six nominal identities, two fixed declarations and
complete derived helper families. Allocation order is unconstrained. The
`missing` mode reproduces absent framework reservations; `fixed` reserves only
fixed declarations and still collides with companions; `late` checks the result
of reserving after nominal identities have already been cached. `owned` atomically
claims every family. `Unique` requires disjoint package declarations;
`ReferencesAgree` checks that each cached primary is actually published.

Recorded results with TLC 1.7.4 (jar SHA-256
`936a262061c914694dfd669a543be24573c45d5aa0ff20a8b96b23d01e050e88`):

| Mode | Expected result | Generated / distinct states |
| --- | --- | --- |
| missing | Unique violated | 2 / 2 |
| fixed | Unique violated | 20 / 18 |
| late | Unique violated | 188 / 64 |
| owned | all invariants pass | 309 / 125 |

Run explicitly from this directory with the existing pinned TLC jar:

```sh
java -cp /Users/luca/.local/share/tlaplus/tla2tools-1.7.4.jar tlc2.TLC \
  -workers 1 -metadir /tmp/loom-declaration-families-owned \
  -config owned.cfg PackageSymbols.tla
```

Replace `owned` in both arguments with each counterexample mode. These are
opt-in design checks, never CI, build, hook or release requirements. Keep checker
states and raw logs outside the source tree.

## Implementation correspondence

`codegen.NameScope.HashedUniqueFamily` owns the atomic allocation and cached
identity. `SetTypeFamilies` installs the owning package's inventory policy in the
first local user-type/union allocation path. This also covers types synthesized
late during view or recursive analysis; a prewalk alone was rejected because it
cannot inventory every future nominal type. The callback must be deterministic
and must not resolve types or allocate in that same scope.

`unionnames.PackageDeclarations` owns common union kind/constants/constructor
names; its branch inventory is also consumed by the constructor emitter and
value construction. Service, view and HTTP owners add their domain-specific
roles. HTTP reserves fixed and endpoint package roles before resolving nominal
references, while receiver members remain independent. Service projection and
occurrence-wrapper helpers whose identities include late projected types use
separate cached role keys in that same package NameScope, before their calls
capture a spelling. Their allocation cannot rename an already captured nominal
identity; the service-owned ViewedHelpers model covers these dynamic roles. Ordinary HashedUnique
callers retain their prior behavior unless their scope installs a family policy.

`TestNameScopeHashedUniqueFamily` checks fixed and companion conflicts, occupied
suffixes, cached identity and qualifier-scope sharing. The late/changed inventory
regression checks atomic failure rather than silently changing already captured
references. HTTP body tests cover the observed Server, endpoint, validator and union
companion duplicates and method independence. TestHTTPPackageRoleSymbols adds
ordinary and conflicting operation-group, custom multipart and path roles.
The separate JSON-query ticket carries its corresponding consumer regressions. Native service/view
controls belong beside their integration owner.

The model assumes a correct emitted-family inventory and finite candidate names;
it does not prove inventory extraction, sanitization, arbitrary suffix exhaustion,
Go rendering, qualified/external package ownership or physical branch layouts.
Located types retain their existing external namespace policy. Generated-module
compilation and exact process-isolated output comparisons establish concrete
correspondence; a green model does not substitute for those gates.
