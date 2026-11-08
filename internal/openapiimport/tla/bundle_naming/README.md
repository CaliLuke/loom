# Bundled design-name allocation (#957)

`BundleNaming.tla` is design evidence for the design-name inventory in §3 of
[`roadmap/issue-957-external-references-design.md`](../../../../roadmap/issue-957-external-references-design.md).
It checks one property: adding external components never changes the name of an
entry declaration, and no two declarations share a name.

## The two namespaces

The imported design has two shared namespaces. The model covers both:

- **T: schema Go type names.** The pre-change `assignSchemaNames` assigned
  these with `uniqueName`, which compares names case-insensitively.
- **V: design variables.**
  - A schema's variable is always `"Imported"+GoName` in the renderer and planner. It cannot be suffixed: the
    pre-change reservation in `assignSecuritySchemeNames` discarded the value returned by
    `uniqueName`.
  - A security scheme's variable is
    `uniqueName("Imported"+Goify(name)+"Security")` in the pre-change allocator. That
    function reserved every schema variable before any scheme variable.

Error-clone types and variables share T and V as well in the render inventory.
They are not modeled; see the limits below.

## Inputs

Schema raw keys, in byte order, with their lower-cased Goify results:

| # | Raw key | Goify result |
|---|---|---|
| 1 | `Foo` | `foo` |
| 2 | `FooSecurity` | `foosecurity` |
| 3 | `Foo_Security` | `foosecurity` |

The Goify results come from a read-only probe of `codegen.Goify(name, true)`.

Security scheme raw keys, with their variable bases:

| # | Raw key | Variable base |
|---|---|---|
| 1 | `Foo` | `importedfoosecurity` |
| 2 | `FooSecurity` | `importedfoosecuritysecurity` |

Entry and external raw keys are disjoint within each namespace. Raw-key
allocation guarantees this.

## Modes

- **`checked`** reserves the complete entry inventory first:
  1. entry type names, in the pre-change raw order
  2. entry schemas' fixed variables
  3. entry scheme variables, in the pre-change raw order

  Then each external schema gets the first GoName candidate whose type name *and*
  fixed variable are both free. External scheme variables are allocated last.
  External orders range over all permutations.
- **`singlePass`** is the pre-change code, run over the union of entry and external
  names.
- **`perNamespace`** is the rejected revision 4 rule. It runs two passes in each
  namespace, but reserves every schema variable before any scheme variable.

The oracle is the names that the pre-change code allocates for an entry-only import.

## Commands and results (2026-11-05, cross-namespace revision)

The toolchain is the same as in `../resolution_identity/README.md`. Run each
configuration from this directory:

```sh
java -XX:+UseParallelGC -cp /path/to/tla2tools-1.7.4.jar tlc2.TLC -workers 4 -deadlock \
  -metadir "$(mktemp -d)" -config checked.cfg BundleNaming.tla
```

| Configuration | Expected | Observed |
|---|---|---|
| `checked.cfg` | `EntryStable` and `NoDuplicate` hold | Pass: 2,238 generated, 1,858 distinct, depth 9, exit 0 |
| `single-pass.cfg` | `EntryStable` fails | Fails, exit 12. Entry security `Foo` with external schema `FooSecurity`: the entry variable becomes `ImportedFooSecurity2`. |
| `per-namespace.cfg` | `EntryStable` fails | Fails, exit 12, on the same pair with the same renaming. This reproduces the review finding. |
| `single-pass-types.cfg` | `EntryTypesStable` fails | Fails, exit 12. Entry schema `Foo_Security` with external `FooSecurity`: the entry type becomes `FooSecurity2`. |
| `witness-pair.cfg` | Expected witness: `PairWitness` is "violated" | Exit 12. Under `checked`, the entry keeps `ImportedFooSecurity`, and the external schema becomes type `FooSecurity2` with variable `ImportedFooSecurity2`. |
| `witness-pair-per-namespace.cfg` | Expected witness | Exit 12. Under revision 4, the external schema keeps type `FooSecurity`, and the entry is renamed to `ImportedFooSecurity2`. |

The schema-only model's earlier pair also follows from `EntryTypesStable`:
external `PetStore` against entry `pet_store`. That earlier model has been
replaced, and its runs are superseded.

## Assumptions and limits

**Bounds:**

- three schema raw keys and two scheme raw keys
- numeric suffixes up to 8

**Abstractions:**

- The allocator is `uniqueName` under monotone claims. Its per-base counter
  bookkeeping is not reproduced.
- T membership is case-insensitive. This is conservative for the `NameScope`
  that error clones use.

**Not modeled:**

- Error-clone types and variables. They follow the same inventory rule in the
  design, and Go tests must cover them.
- Goify on arbitrary input.
- Raw-key sanitization.

**Required Go tests:**

- each pair above
- entry-only byte-identity
- an entry error clone whose variable an external schema could take

This model is an explicit design tool. It is never a CI gate.

## Implementation correspondence

`names_inventory.go` owns the shared entry-first inventory, including error
clones omitted from this bounded model. `resolve_bundle.go` owns raw component
keys. `TestAllocateDesignNamesEntryStable`, `TestAllocateDesignNamesErrorClones`
and `TestAllocateDesignNamesNoDuplicates` cover namespace interactions.
`TestErrorCloneInventoryResponseOrigins`, `TestErrorCloneInventoryMatchesRenderPlan`
and `TestErrorCloneInventorySyntheticHeader` link inventory decisions to the
actual response render plan. Process-isolated CLI tests cover deterministic output.

The bounded assumptions and recorded checker results above remain unchanged.

## Synthetic declaration ownership after oneOf normalization

The original allocation model assumes its entry/external classification is
correct. Inline declarations promoted while preparing a oneOf branch were
outside that assumption: constructing a diagnostic path from a generated type
name can miss the resolver's authored provenance and classify an external
schema as entry-owned before allocation.

The same model now includes an `OriginPreserved` prerequisite, with five bounded
cases: entry/external inline declarations, nested external inline declarations,
and references crossing the entry/external boundary in either direction.
Paths abstract decoded pointer segments and document ownership; they do not
model arbitrary YAML, URI lookup, or Goify. `syntheticPath` reproduces lookup of a
fabricated generated-component path; `callerPath` rejects preserving the caller
path across a reference instead of resetting to the target component.
`checkedOrigin` appends authored child positions and resets at reference targets.
The allocator's existing namespace invariants still run independently: this
extension checks their provenance prerequisite, not a proof of arbitrary
normalization graphs or the concrete path-construction implementation.

Run with the same pinned toolchain, from this directory:

```sh
java -cp /path/to/tla2tools-1.7.4.jar tlc2.TLC -workers 1 \
  -config origin-checked.cfg -metadir /tmp/loom-bundle-origin-checked BundleNaming.tla
```

`origin-synthetic.cfg` fails on `externalInline`; `origin-caller.cfg` fails on
`entryRefExternal`. `origin-checked.cfg` passes `OriginPreserved`, `EntryStable`
and `NoDuplicate`: 11,190 generated / 9,290 distinct states, depth 9.
These commands remain opt-in and do not become build, test or release gates.

Implementation correspondence: `prepareUnionFields` carries the authored path
through properties, oneOf/allOf, items and additionalProperties, and resets it
when following a component reference. `prepareSchemaObjects` passes that path
to `bundledDescription.declarationOrigin`. The direct integration regression
`TestIntegrationExternalOneOfPromotionOrigin` loads a standalone mapped external
OpenAPI document with inline, nested-inline and referenced branch controls. The
uncorrected merge loses ownership for `ChoiceVariantNode` and
`ChoiceVariantNestedNode`; the corrected traversal preserves ownership in all
three cases. This does not replace whole-ticket import, generation, determinism,
security-URI lint and independent integration review.

## Local Schema target promotion and declaration priority

`TargetPromotion.tla` distinguishes authored component roots from local nested
and external targets. The old entry-document shortcut leaves admitted local
properties, alternatives and response schemas at pointers the normalized
component-reference model cannot consume. `promotion-legacy.cfg` reproduces
`Consumable` failure (39 generated, 22 distinct, depth 11, exit 12).
`promotion-checked.cfg` passes `Consumable`, `EntryStable` and
`DeterministicClaims` (40 generated, 22 distinct, depth 11, exit 0). Run with the
same opt-in TLC command, substituting this model and configuration.

Bounds are four canonical targets, two reserved component keys and all discovery
orders. Identity and grammar admission are assumed inputs; arbitrary URI parsing,
YAML, recursion and Go name normalization are outside this model. The implementation
uses `descriptionBundler.allocate` for both local nested and external promotion,
and `bundledDescription.changed` ensures normalization consumes the bundled tree.
`TestLocalSchemaTargetsImport`, `TestLocalSchemaTargetSharedIdentity`,
`TestLocalSchemaTargetAliasOrder` and `TestLocalSchemaTargetsGenerated` check
property/alternative/response/recursive targets, aliases, raw collisions, origin
identity, mapping outcomes and generated compilation/libopenapi contracts.

Declaration priority is independent of source-document identity. The existing
BundleNaming entry/external sets now mean authored/introduced declarations:
introduced local components join external declarations after the complete authored
inventory. Re-running `single-pass-types.cfg` still exposes `EntryTypesStable`
failure; `checked.cfg` passes 2,238 generated / 1,858 distinct states, depth 9.
`TestLocalPromotionPreservesAuthoredNames` reproduces the concrete `foo_bar` versus
promoted `foo-bar` collision. `declarationOrigin.introduced` and
`allocateIntroducedSchemas` retain this priority without altering SourceLocation;
moved local Path Item operations remain local operations. Existing namespace and
error-clone tests cover all consumers. URI mappings do not acquire an implicit
component-name key merely because the target was promoted.

## Canonical materialization and authored occurrences

`CanonicalOccurrences.tla` models the copy boundary after canonical target
allocation. The old copier retargeted reference leaves but copied the addressed
Schema body again at its original property, alternative or response position.
This split a single source identity into multiple declarations. The corrected
copy rule materializes a promoted Schema only at its allocated component path;
every other occurrence references that component, with the original source
location. Authored component aliases retain their own declaration identities.

Run explicitly with the pinned toolchain above:

```sh
java -XX:+UseParallelGC -cp /path/to/tla2tools-1.7.4.jar tlc2.TLC -workers 1 \
  -config occurrence-checked.cfg -metadir /tmp/loom-canonical-occurrences CanonicalOccurrences.tla
```

The finite model explores all copy orders for ten sites and four source
identities. `occurrence-old.cfg` reproduces `CanonicalUse` failure (exit 12).
`occurrence-checked.cfg` passes canonical use, exactly one materialization,
distinct identities, aliases and origins: 5,122 generated / 1,024 distinct states,
queue exhausted, exit 0. Two rejected alternatives also produce counterexamples:
`occurrence-shape.cfg` merges equal-shaped distinct sources and violates
`DistinctIdentity`; `occurrence-self.cfg` replaces the canonical destination
itself and violates `Materialized`. Both exit 12 with 1,024 distinct states.
These results informed the copy rule before its implementation.

`descriptionBundler.copyNode` implements this boundary, using the existing
source-keyed `targets` and `kinds` maps. `TestCanonicalSchemaTargetOwnsOriginalOccurrence`
covers properties, alternatives, response schemas and recursive nodes.
`TestCanonicalSchemaOccurrenceKeepsAliasAssertions` checks authored aliases,
reference siblings, exact numeric assertions, provenance and an equal-shaped
unaddressed control. Existing local-target generation tests check compilation
and parsed exported contracts; selection and alias-order tests cover admission
and traversal-order interactions.

The model assumes allocation is already stable. It abstracts aliases as distinct
logical declarations and does not prove URI/YAML parsing, numeric values,
arbitrary recursion or the allocator itself. The Go tests cover correspondence
at these seams. Model execution remains opt-in and is never a delivery gate.
