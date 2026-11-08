# Discriminator property guarantees

The importer must distinguish a conditional JSON Schema `required` implication
from an unconditional property-presence guarantee. `required` does not constrain
null values. A discriminator alternative, however, defines its property only
when it is an object with that required member. The source-version rules then
reject optional hints in 3.0, admit them in 3.1, and require `defaultMapping` in
3.2. Hints do not change the source validation domain.

`PropertyGuarantees.tla` models one varying alternative and one fixed required,
nonnullable alternative. Its finite value domain is null, an object with the
member, and an object without it. A declared nullable domain can be narrowed by
an object intersection; a reference occurrence can independently add null. A
nullable wrapper around the whole union does not own the inner hint and must
not change its alternative classification.

## Run explicitly

Use the installed official TLA+ tools; this is an opt-in design aid, not a build,
test, hook or CI prerequisite. Keep checker output outside the source tree.

```sh
java -cp "$TLA2TOOLS_JAR" tlc2.TLC -workers 1 \
  -config checked.cfg -metadir "$TMPDIR/discriminator-admission" PropertyGuarantees.tla
```

With TLC 2.19 / TLA+ tools 1.7.4 and Java 25.0.5, the checked configuration
exhausts 384 distinct states (192 initial inputs). Countercontrols:

| Configuration | Expected result |
|---|---|
| `old` | `VersionAdmission` fails: required-only classification admits a nullable alternative without the version's required fallback |
| `conflate` | `KeywordPreserved` fails: treating null as a required-keyword violation rejects a redundant sibling |
| `wrapper` | `VersionAdmission` fails: an outer nullable wrapper is incorrectly treated as a property of an inner hinted alternative |
| `checked` | both invariants pass |

The rejected configurations were checked before the production correction.

## Correspondence and limits

`schemaDefinesProperties` combines the existing `oneOfTypeSubset` and
`oneOfRequires` facts. The former excludes null and non-object alternatives; the
latter remains the conditional required-keyword proof used by oneOf sibling
admission. `normalizeDiscriminator` applies the stronger fact to each authored
alternative, or the parent declaration for ParentHint. It retains the existing
missing-normalized-alternative check for a bare null branch.

`TestDiscriminatorNullableAlternativeAdmission` covers source versions, declared
nullability, nullable reference occurrences, aliases and fallback presence.
`TestDiscriminatorNullableParentAdmission` covers the parent owner.
`TestSchemaPropertyGuarantees` separates the two facts, including an object
intersection with a nullable base. That direct normalized-model control does
not widen the importer's allOf grammar: unsupported source intersections still
receive their existing diagnostic. `TestNullableRequiredSiblingRemainsRedundant`
and `TestDiscriminatorSurvivesNullableNormalization` protect the two consumers
and inner-hint ownership. The generated admission test compiles retained nullable
contracts, parses JSON/YAML output, compares independent-process bytes and checks
representative source/generated values.

The model assumes normalized null membership and required facts are correct.
It does not prove source parsing, arbitrary reference cycles, enum normalization,
exact-one ambiguity, codecs, naming, extension admission or actual Go output.
Those remain implementation-test obligations. The existing discriminator import
representation model and oneOf null/domain models retain their separate scopes.
