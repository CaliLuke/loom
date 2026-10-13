# Strict OpenAPI schema normalization model

This small Lean model records the value-acceptance contract for the strict
OpenAPI importer cases tracked by Loom issue [#659](https://github.com/CaliLuke/loom/issues/659).
It is design evidence for the Go importer, not a proof that the Go code refines
the model.

Run `lake build` in this directory with Lean 4.34.1. The model has no external
packages. `required-theorems.txt` is the named theorem manifest for review.

To inspect each theorem's type and axiom dependencies after building, run:

```sh
{
  printf 'import SchemaNormalization\n'
  while IFS= read -r theorem; do
    printf '#check %s\n#print axioms %s\n' "$theorem" "$theorem"
  done < required-theorems.txt
} | lake env lean --stdin
```

The current theorem set depends only on Lean's standard `propext` axiom where
Boolean acceptance equalities are simplified; no theorem uses `sorryAx` or an
external axiom.

## Findings and design decision

`candidate_preserves_acceptance` proves, over the finite representative value
domain, that each admitted candidate normalization has exactly the same
acceptance as the source schema. The `counterexample_*` theorems witness the
current lossy decision rules: `false` becomes unconstrained, properties infer
an object type that the source did not declare, a constrained null composition
introduces null, and the OAS 3.0 `nullable` rule is applied in modern dialects.

The candidate is fail-closed where the current normalized Go schema has no
equivalent representation: boolean `false`, typeless schemas with properties,
null-only schemas, constrained null compositions, and use of the legacy
`nullable` keyword outside the plain OAS 3.0 string control are rejected.
Boolean `true` and `{}` both
normalize to unconstrained `Any`. A bare null branch alongside a string branch
and a modern `type: [string, null]` union normalize to nullable string only in
OAS 3.1/3.2. The only admitted OAS 3.0 nullable control is a plain explicit
string type. The modeled constrained composition is
`anyOf: [{type: string}, {type: null, enum: ["forbidden"]}]`; the null branch
accepts no values, so the source accepts all strings and rejects null. The
source semantics in 3.1/3.2 ignore `nullable`
as an assertion, consistent with JSON Schema's unknown-keyword behavior, but
admission rejects that authored keyword until its intended use can be handled
without silent loss.

The modeled Go ownership seams are `schema()` and
`schemaTypeAndProperties()` in `internal/openapiimport/analyze.go`, plus
composition reduction in `internal/openapiimport/analyze_schema_composition.go`.
Recommended normalized Go shape: retain whether a type was explicitly
declared separately from any inferred implementation shape; represent false
as bottom or reject it explicitly; make nullable normalization dialect-aware;
and only collapse a composition when all sibling constraints survive. If the
IR cannot express the source acceptance set, report an import diagnostic.

## Assumptions and limits

The dialect split follows the [OpenAPI 3.0.3 Schema Object](https://spec.openapis.org/oas/v3.0.3.html#schema-object),
the [OpenAPI 3.1.1 Schema Object](https://spec.openapis.org/oas/v3.1.1.html#schema-object),
and the [OpenAPI 3.2.0 Schema Object](https://spec.openapis.org/oas/v3.2.0.html#schema-object).
OAS 3.1 and 3.2 use the JSON Schema 2020-12 vocabulary and dialect model;
boolean schemas are covered by [JSON Schema Core 2020-12 §4.3](https://json-schema.org/draft/2020-12/json-schema-core#section-4.3),
and `type` arrays by [JSON Schema Validation 2020-12 §6.1.1](https://json-schema.org/draft/2020-12/json-schema-validation#section-6.1.1).

The model abstracts JSON values to eight representatives and schema inputs to
twelve constructors. This is enough to expose the distinctions at issue,
but it does not model arbitrary JSON trees, references, evaluation annotations,
formats, numeric bounds, additional properties, or the generated Go validator.
The acceptance theorem therefore proves only the finite abstract model. Go
regression tests must establish correspondence at the importer and rendering
seams, including every dialect branch.

OAS 3.0 boolean schemas and modern type unions are marked invalid in this
model; the candidate rejects them. The model treats `true` and `{}` as the
unconstrained schema in 3.1/3.2, and treats legacy `nullable` as effective only
for OAS 3.0. These version assumptions should be revisited if the importer
adds a distinct compatibility dialect or changes its supported schema subset.


## Finite value domains (#777 / #778)

The `ValueDomain` namespace extends the same model with the missing intersection
between an admitted string/null type and enum/const membership. Its six theorems
cover all modeled membership constructors, both type-nullability choices and the
eight representative values. `current_nullable_enum_counterexample` reproduces
#778: copied nullability accepts null excluded by the source enum.
`strip_only_counterexample` rejects the tempting #777 repair of merely removing
null: a null-only enum would become unrestricted strings. The candidate preserves
acceptance for every admitted domain and explicitly rejects empty/null-only
intersections. Valid nullable and nonnullable enums, absent enum and scalar const
remain admitted. The existing theorem manifest and `lake build` cover this model.

The Go owner is `analyzer.schemaValueDomain` after type analysis, before defaults,
examples and unconstrained classification. Source enum presence must survive an
empty normalized slice, and a retained admission reason must reach
`documentPlanner.schemaAdmission` so direct Render cannot bypass rejection.
Renderers consume the normalized enum/nullability without recomputing policy.
An outer bare-null `anyOf` can legitimately widen a nonnullable inner enum;
therefore absent outer membership must not re-intersect the copied branch enum.
`TestImportedValueDomain*` checks real analysis, rendering and the existing
roundtrip instance validator across references, fields and collection elements.

This finite addition assumes dialect/type admission already chose nullability.
It does not prove arbitrary composition, reference graph resolution, numeric
identity, codec behavior or Go compilation. Existing DSL Enum requires at least
one type-compatible value; null-only and empty-domain rejection also follow the
documented strict subset. No new universal bottom type is proposed. The previous
#659 model remains unchanged and does not claim these additional obligations.
