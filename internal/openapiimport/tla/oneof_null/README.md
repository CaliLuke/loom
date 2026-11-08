# Null membership in untagged `oneOf` normalization (#1014 design)

`OneOfNull.tla` checks how import classifies null admission before applying
the bare-null rule. The flat rule is proved for arbitrary branch lists by
`normalize_sound` in
[`OneOfNormalization.lean`](../../lean/schema_normalization/README.md). This
model checks the inputs to that rule: the null kinds of `Any` and of nested
unions.

Source trees are OpenAPI `oneOf` schemas:

- **Leaves** admit null not at all (`e`), at this occurrence (`o`,
  `type: [T, null]`), or through a declared nullable component (`d`).
- **`Any`** (`{}`) admits null.
- **A nested union** admits null when exactly one of its alternatives does,
  counting its own bare `{type: null}` branches.

Normalization runs bottom-up. It normalizes each outer branch's subtree, one
branch per step, then applies the bare-null rule to the outer union. The
representation semantics is Loom's:

- an occurrence `nullable` flag accepts null;
- otherwise exact-one counts each branch's own null admission.

The bounds are one outer tree of depth ≤ 2 and one of depth ≤ 1, each union
with two alternatives, and 0..2 outer bare nulls (inner unions 0..1). That
gives 31 536 initial documents.

```sh
java -cp "$TLA2TOOLS_JAR" tlc2.TLC -workers 1 -config checked.cfg -metadir "$TMPDIR/oneof-null-checked" OneOfNull.tla
```

Observed with Java 25.0.5 and TLC 1.7.4 (`tla2tools` sha256
`936a262061c914694dfd669a543be24573c45d5aa0ff20a8b96b23d01e050e88`):

| Configuration | Expected and observed |
|---|---|
| `checked` | pass, 126 144 distinct states; `NullAgreement` (representation accepts null iff the source does) and `RejectionIsNecessary` (every rejection is a declared null beside a bare null at some level) |
| `flag-only` | exit 12, `NullAgreement`: inner `oneOf: [string, {}, null]` classifies `Any` as non-null, makes the occurrence nullable and accepts null that matches two source branches |
| `shallow` | exit 12, `NullAgreement`: inner `oneOf: [oneOf: [string, {type: [string, null]}], string, null]` misses the nested local null |

Assumptions and limits:

- Non-null acceptance is not modeled. Stripping changes only leaf null kinds
  and union occurrence flags, so non-null domains are unchanged by
  construction; `count_some_strip` proves the flat case.
- The trees are bounded. The model does not prove arbitrary recursion,
  references or alias graphs, Go carriers, or the generated decoder's
  null-to-zero guard.
- Implementation seams: the importer's composition normalization in
  `schemaUntaggedOneOf`, `normalizedNullKind` and `removeOccurrenceNull` in
  `internal/openapiimport/oneof_normalize.go`, the bottom-up normalization order,
  and the expr nullable-branch rule in
  `expr/untagged_validation.go`.
- Implementation regressions: `TestOneOfNormalizedFacts`,
  `TestOneOfImportBehaviorTable` and `TestOneOfImportedDesignGenerates` check
  import, evaluation and runtime null matching for the null normalization
  cases, including `[{}, string]`, `[{}, string, null]`
  (rejected) and the nested local-null case.

This is opt-in design evidence. It is not a build, CI or release gate.

## Nullable wrapper recognition

A retained closed singleton exports as `anyOf: [closed oneOf, null]`.
`schemaNullableAnyOf` must admit the already-normalized OneOf value even when
that value has no direct Type or Ref. It retains the complete branch contract
and closure, then enables occurrence nullability; it does not broaden the
recognized two-alternative nullable wrapper to general anyOf or unsupported
compositions.

`OneOfNormalization.Rep.accepts` already represents arbitrary branch lists and
an occurrence nullable flag: null is admitted by that flag or unique branch
membership; non-null values still require exactly one branch. This correction
restores an omitted carrier shape to that representation. The existing
`normalize_sound` theorem proves bare-null **oneOf** normalization, not
**anyOf** source parsing or closure. No new proof claim follows from reusing
the representation. Nullable-pair grammar admission and retained closure remain
implementation correspondence obligations, checked by
`TestNullableClosedOneOfAdmission` and the generated export/reimport/evaluation
and runtime checks in `TestOneOfSingletonClosureGeneratedContract`.
