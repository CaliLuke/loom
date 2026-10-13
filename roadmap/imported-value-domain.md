# Imported scalar value domains (#777 / #778)

Readiness began at `6a549820f795acbe550c25bb45034585d99a5d6f` and was
approved by root. The shared owner is delivered in `6edea4932`; independent final and
integration reviews passed, and both reports are confirmed closed. The two reports
described one missing intersection: the parent importer handled type/nullability
separately from enum/const and published both decisions to every renderer. The
shared normalization owner replaces that split; do not repair rendered occurrences.

1. [x] Read the reports, analyzer/rendering owners and documented import subset.
2. [x] Add direct failing analysis/rendering regressions and existing roundtrip
   controls for OAS 3.0 and 3.1; include scalar component references, inline fields,
   array items and map values. Preserve unsupported composition boundaries.
3. [x] Extend the existing schema-normalization Lean model with finite membership
   intersection, current counterexamples and candidate acceptance equivalence.
4. [x] Freeze one normalization-owner proposal and bounded evidence for review.
5. [x] Implement the shared admission/normalization owner and run affected
   regressions plus bounded generated-output/contract checks.
6. [x] Obtain independent exact final review before atomic #777 delivery.
7. [x] Reconcile #778 independently against that delivered shared owner.

## Contract and proposed owner

The accepted strict importer preserves source acceptance or produces a blocking,
source-qualified diagnostic. OAS 3.0 nullable widens the explicitly declared type;
enum still intersects it. OAS 3.1/3.2 type arrays and enum/const likewise intersect.
See [OAS 3.0 Schema Object](https://spec.openapis.org/oas/v3.0.3.html#schema-object)
and [JSON Schema validation](https://json-schema.org/draft/2020-12/json-schema-validation).

Normalize at `analyzer.schema` after its type/composition analysis and before
defaults, examples, unconstrained classification or DSL rendering. The normalized
IR must carry only admitted non-null enum values and effective nullability:

- Nullable string plus enum `[a, null]` becomes `Nullable()` plus `Enum("a")`.
- Nullable string plus enum `[a]` becomes nonnullable `Enum("a")`; the source
  excludes null, so removing ineffective nullability preserves its contract.
- Integer plus enum `[1, null]` becomes nonnullable `Enum(1)`; membership does
  not widen a type. Other excluded member types require the same intersection.
- Scalar const becomes a singleton finite domain through the same owner.
- Null-only or empty effective domains have no current DSL form; reject them
  before rendering. Do not turn an empty finite domain into absent validation.
- Existing rejection of structured const, const combined with enum, typeless
  constrained schemas and unsupported reference siblings/compositions remains.

No public decision is needed for these examples: all follow source semantics and
the documented strict subset. Numeric membership identity must reuse existing
numeric representation rather than add a float-based equality shortcut.

At the readiness parent, `schemaEnum` copied null directly, `enumValidation` emits `Enum(nil)`,
and `effectiveNullable` propagates the independent nullable bit through references.
Repair the analyzed domain once; renderers and reference consumers use that result.
Keep authored enum absence distinct from an authored finite domain with no usable
members. The complete reference/composition inventory and exact rejection seams
will be recorded after direct probes.

## Scope and delivery boundary

#777 and #778 cannot be repaired correctly by independent null-removal and
nullable toggles. Propose #777 as the atomic shared-owner repair, then reconcile
#778 against that delivered contract with independent evidence; do not combine
unrelated changes to meet an issue count. Root approved this delivery boundary: one atomic #777 shared-owner repair, followed by independent #778 reconciliation.
#776 empty objects, #783 annotations and #686 byte lengths get consumer inventory
only unless evidence proves a dependency. #791 remains a later reconciliation
against the delivered named query JSON contract. No broad importer corpus or new
generation harness is planned; reuse `requireRenderedDesignGenerates` and the
existing acceptance comparison helpers.


## Readiness results and implementation boundary

The new direct suite has 33 expected failing subcases plus the selection/admission
failure, and 11 passing controls. Integral decimal and ordinary number members
remain controls; fractional integer and wrong-type members require intersection.
It reproduces null members reaching Enum, nullability independent of membership,
wrong-type members retained, missing empty/null-only admission, and all six
request/response/component/field/item/map paths. Two existing generated roundtrips
for OAS 3.0/3.1 each demonstrate both failures: null-containing designs fail DSL
evaluation; null-excluding designs generate but their contracts accept null.
The instance comparison uses an equivalent modern type union for OAS 3.0 because
the existing Draft 7 validator does not interpret nullable. Existing strict
admission, selection and const-boundary controls pass. Scoped lint passes.

The extended Lean model builds with the existing pinned toolchain. Six new
manifested theorems establish bounded intersection preservation and two current/
partial-repair counterexamples. No Go behavior is claimed proved. Actual DSL
`Enum` requires at least one type-compatible value (`dsl/validation.go`); the
coverage guide already rejects null-only schemas. Thus rejecting these effective
domains follows expressibility, rather than adding a new policy.

The implementation replaces `schemaEnum` with one normalization operation,
not add renderer checks. Retain authored enum presence (including `[]`) and any
blocking domain reason on Schema. The existing planner traverses every schema
occurrence for both Analyze and Render; consume that retained reason there so
there is one source-qualified diagnostic and a direct Render cannot publish a
weakened schema. Selection/partial import must retain the established component
closure filtering. Use the original source presence, not len(normalized Enum),
to distinguish no assertion from an empty domain.

Normalize only the occurrence's authored membership. An accepted bare-null anyOf
with an inner nonnullable enum must remain nullable: copied inner Enum is not an
outer intersection. Outer empty enum is still a constraint and must reject.
Existing unsupported ref-sibling/allOf/structured-const boundaries remain; do not
expand general composition support. Use the source scalar's JSON type to remove
members excluded by the declared type, then verify the remaining values can be
rendered in the existing DSL representation. Do not silently discard values merely
because a Go carrier cannot represent them, and do not introduce float-based
numeric equality or coerce strings into numeric members.

Consumer inventory keeps three adjacent reports separate: #776 is the empty
inline-object DSL constructor in objectSchemaExpression; #783 is annotation
placement in attribute/schemaBlock/validationBlock; #686 is encoded string versus
decoded Bytes length in collectionLength and format lowering. None is repaired
by scalar enum/nullability intersection. Their reports are saved separately.

Final acceptance after approval: green direct/roundtrip tests and existing
nullable/schema-admission/const controls; actual generated Go compilation, parsed
and pinned-rule-linted emitted OpenAPI on the two bounded roundtrip specimens;
exact process-repeat generated bytes; unaffected ordinary enum/nullable control
output compared to parent. Reuse existing helpers and source-mode contract.
Readiness approval: `/tmp/loom-777-root-readiness-verdict.json`. The bounded
composition matrix additionally found an outer const overwriting a copied inner
enum. Its authored sibling now prevents unsupported composition normalization.
No general composition support is added. Numeric classification uses original
exact tokens before carrier conversion; included unrepresentable members block.


## Final candidate evidence

The direct owner/selection/composition and existing admission controls pass with
race detection (1.684s); scoped lint reports zero issues. The four OAS 3.0/3.1
roundtrips compile, parse with libopenapi and preserve the eight checked instances
across references, inline fields, array items and map values. Numeric generation
retains Int64/Int32 boundaries and the Float32 decimal spelling; a source fraction
that rounds to one is excluded from integer membership before decoding. YAML
hexadecimal, octal, binary and exact fractional token controls pass as well.

Seven bounded modules were generated with the actual archived parent source and
candidate source. The test-only capture overlay changes output retention and
first-generation byte capture; it does not replace production source. The
initial parent capture used a non-canonical macOS `/tmp` overlay key and did not
retain output; the final capture used its resolved `/private/tmp` path. Only the
latter supplies artifact comparison evidence. Parent/candidate inputs and the
exact archive are retained in `/tmp/loom-777-*` records until review and cleanup.

Every intended output difference is recorded in
`/tmp/loom-777-artifact-comparison.json`:

- Null-containing membership previously failed DSL evaluation with Enum(nil).
  It now emits all 16 artifacts per version, preserving nullable carriers.
- Null-excluding membership changes only three carrier/validator Go files,
  OpenAPI JSON/YAML and the generated design digest per version. The generated
  schema and carriers now exclude null consistently.
- Numeric normalization removes the rounded duplicate integer member from the
  generated client validator and JSON/YAML enum. The parent module's go test
  gate fails vet on the redundant predicate; the candidate passes. The design
  digest also changes.
- The existing supported enum fixture and nullable-without-enum fixture have
  identical rendered designs and all 16 generated artifacts against the parent.

All 112 candidate generated artifacts are byte-identical across independent
process generations. Five unique generated contracts cover the seven specimens;
vet and pinned Vacuum JSON/YAML lint pass for those five, with zero lint errors
(editorial warnings remain). Equivalent OAS 3.0/3.1 outputs reuse those checks.
The checked-in input fixtures need no regeneration changes; this importer repair
does not alter the general generator's output for unchanged designs.

The unchanged bounded Lean model and its 17 manifested theorem checks are reused
from approved readiness. It checks finite string/null membership, not numeric
carriers or Go implementation equivalence. A broad TestAnalyze prefix accidentally
included the known #820 libopenapi YAML-merge race; that log is preserved separately
and the explicit domain-only race gate above is green. No dependency change or
claim of a green full repository suite is included.

Delivery: independent direct race checks pass (1.575s); the integrated OAS 3.1
null-exclusion generation/acceptance check passes (11.926s). Separate #778 review
confirmed source/generated acceptance for OAS 3.0/3.1 and all covered occurrence
paths. Cleanup preserves exact inputs and condensed evidence in
`/tmp/loom-777-cleanup-reuse-receipt.json`.
