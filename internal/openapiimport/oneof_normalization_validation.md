# oneOf importer normalization evidence (#1014 M6)

This slice owns normalized import facts, expression-grammar projection, object
promotion, rendering and media admission. It also corrects the response raw-codec
direction check in `expr/http_response.go`. It does not claim completion of the
parallel form/multipart runtime slice.

The acceptance rule is the complete accepted design, not the old object/array
subset: all 44 behavior-table specimens must receive their expected admission or
owning diagnostic; admitted branches retain assertions, occurrence nullability,
closure and nesting; generated designs compile at representative constructor and
transport boundaries. Six existing import controls must remain byte-identical.

## Ownership and implementation

- `oneof_normalize.go` applies the bottom-up bare-null rule. Declared null
  membership (including Any references) cannot be removed; occurrence-local null
  can be stripped recursively without flattening exact-one nesting.
- `oneof_siblings.go` checks siblings after the entire normalized declaration
  graph exists, before constructor promotion. This makes forward references
  independent of declaration order. Redundant assertions disappear; rejected
  siblings report their source JSON pointer and composition owner.
- `expr.ValidateUntaggedUnion` is a public, detached-graph admission seam because
  import is a different package. It calls the existing private grammar and
  cycle owners with a fresh traversal, avoiding global DSL validation state and
  a second grammar in the importer.
- `oneof_promotion.go` establishes names for structured object fields throughout
  referenced branch graphs; maps and inline scalar/collection branches retain
  their own constructor representation.
- `Schema.MediaEncoding` retains authored Encoding Object slots for
  single and multiple media contracts. Source type facts remain on schemas. The former `RequestBody.FormEncoding`,
  duplicate form admission path and duplicate metadata renderer are removed.
  Source type and array-item facts survive normalization so content application
  and branch-local defaults reach the shared resolver. Typeless unions keep their
  octet-stream default and fail at the media codec owner.
- The response raw check now uses `SkipResponseBodyEncodeDecode`; a raw request
  can no longer disable validation of a typed text response.

## Recorded checks

Go 1.27.3; `GOCACHE=/tmp/loom-957-build-cache.r6al1q`; generated checks set
`LOOM_DIR=/Users/luca/.codex/worktrees/oneof-import/loom`.
Scratch evidence: `/tmp/loom-1014-importer`.

- The initial 44-row test failed against the prior importer (`red.log`). All
  44 Analyze/Render expectations now pass: 32 admitted and 12 rejected at their
  owning boundaries (`final-normalization.log`).
- Detached expression grammar, recursive consumption and fresh traversal tests
  pass (`expr-admission.log`). The response-direction regression fails in both
  directions before the correction and passes afterward (`raw-response-red.log`,
  `raw-response-green.log`).
- Fourteen representative generated designs compile, and generated OpenAPI
  parses through libopenapi. Common Draft 7/2020-12 vocabulary is checked for
  source/regenerated instance-membership agreement. Closure is checked
  structurally instead of pretending Draft 7 implements unevaluatedProperties.
  `generated-semantic.log` records the initial successful cases plus a corrected
  whitespace-sensitive closure assertion; `closure-spec.log` records its pass.
- Inline enums, nested unions and captured JSON form codecs regenerate in
  independent processes with exact JSON/YAML equality.
- A referenced union error also uses a service error envelope and selected HTTP
  body (`TestOneOfReferencedErrorEnvelope`, `referenced-error.log`).
- `supported.yaml`, `supported.json`, `symmetry.yaml`,
  `encoded_component_reference.yaml`, `shared_status_different_schemas.yaml`
  and `object_field_names.yaml` render byte-identically to the parent captures
  in `/tmp/loom-1014-prep/base-import/design` (`controls.json`).
- The full importer package reached its existing 10-minute timeout in
  `TestRenderReferenceContexts`; it is not recorded as passing. Every reported
  failure before that timeout was addressed and its focused rerun passes
  (`package-fixes.log`, 92.695s). The remaining test-name expression is saved in
  `remaining-tests.txt` for continuation with a suitable timeout.

The bounded generated selection covers inline assertions, bare/nested null,
map and nested union roots, named nested fields, errors, query/header codecs,
JSON form fields, source codec capture, closure, raw documentation and typed
additional members. It is not a full exported-design corpus audit.

## Parallel completion obligations

At this freeze, generated rows 35, 37, 38 and 39 still depend on the native
FormRoot/MultipartRoot admission, body-presence and runtime slice. The full
section 5 field-codec cases are maintained in `TestOneOfFormFieldCodecs`:
explicit text/plain, source scalar/array defaults, explicit form style and
multipart JSON (including null). The shared codec/runtime owner must complete
those cases; the importer must not reject their accepted scope or bypass the
codec owner. Multipart admission now passes the complete Serialization to
`ResolveMultipart`, preserving union media intent.

Rerun those cases, the generated selection affected by the final shared changes,
and the remaining package tests before final ticket review. The overall #1014
validation document owns the exporter Vacuum checks and final ticket verdict.

## Formal correspondence

The existing opt-in `OneOfNull.tla` checked configuration (126,144 states) and
Lean `normalize_sound` theorem remain the design evidence for the unchanged
normalization rule. Their READMEs now name the implemented owners and tests.
They do not prove reference traversal, codec selection, Go compilation, full
JSON Schema implication or transport behavior. No proof command was added to
ordinary test/build/CI targets.

## Root Encoding Object inheritance follow-up

The suspected gap was reproduced: inline, referenced and nested root unions
reported a typeless field's octet-stream default even with an explicit root
`contentType: application/json` (`root-encoding-red.log`). The media traversal
now carries a detached `mediaSchemaContext` through aliases and alternatives.
`mediaEncodingFor` and `mergeMediaEncoding` retain occurrence-first slots without
writing an override into a shared declaration.

Authored root Encoding Object slots override the same authored slots on branch
declarations. Unspecified authored slots inherit; explicit false boolean slots
remain explicit. A wholly empty Encoding Object preserves the source default.
The first implementation also stored source defaults in authored slots; the
review follow-up below corrects that lost provenance while retaining the same
occurrence-first rule for authored metadata.

Encoding names are checked against the union of possible properties at the
schema where they are declared. A property may exist in only one alternative.
Each branch resolves only its own fields, while a name absent from every
alternative remains an error at the declaring media path. An active traversal
set prevents unconsumed cycles from causing recursive media admission; the
expression grammar remains responsible for the cycle diagnostic.

`TestOneOfRootEncodingContext`, `TestOneOfRootEncodingContextIsolation`,
`TestOneOfRootEncodingOverridesCapturedDefault` and
`TestMediaEncodingContextRetainsExplicitFalse` cover the real regression,
missing/unknown/invalid overrides, disjoint alternatives, alias/nested descent,
same-document consumer isolation, default precedence and detached maps. The
final direct selection, including the 44-row matrix, passes
(`root-encoding-direct-final.log`). The affected media/map selection passes
(`root-encoding-focused.log`, 16.562s).

The opt-in Lean model in `lean/media_encoding_context` checks the old
counterexample and candidate precedence through arbitrary nested contexts
(Lean 4.35.0, `root-encoding-model.log`, success). Its assumptions and unproved
implementation obligations are in that directory's README. It adds no CI,
ordinary test or toolchain-install requirement and produces no retained compiled
proof artifacts.

`TestOneOfRootEncodingGenerates` preserves a generated regression for the shared
root runtime integration. At this follow-up it reaches the existing expr
rejection, `FormRequest does not support untagged OneOf request bodies`
(`root-encoding-final.log`); it is not counted as passing. The root Encoding
Object is preserved in emitted DSL for the shared owner to lower.

The interrupted package run is continued with twelve bounded later tests, not
the full suite: reference contexts; multiple request media; multipart/binary
media; shared media admission; reusable transport components; annotations;
nullable/cyclic semantic views; unconstrained wire round trips; value-domain
round trips; YAML component indexing. The selection uses a 15-minute timeout
and is expected to take a few minutes because several tests generate and compile
applications. The selection passes in 254.099s (`later-package-selection.log`). This is
not an exported-design corpus audit or a claim that every unselected test ran.


## Earlier default-label experiment (superseded)

The intermediate mediaDefaults/DefaultMediaType representation was tested for
occurrence isolation and generated code, but incorrectly treated source arrays
as whole JSON fields and retained an authored content/style conflict. Those
claims are withdrawn. The official-spec correction below replaces that
representation and its acceptance cases. Historical logs are retained under
/tmp/loom-1014-media-default-fix only until the combined independent review and
cleanup; they do not prove the corrected application contract. The valid
occurrence-first inheritance and declaration-isolation obligations are retained.

## Source application correction

The prior fallback-label implementation and its generated source-array result
are historical evidence, not acceptance for the corrected contract. Official
OpenAPI 3.2 encoding-by-name rules require item application for source arrays;
explicit style flags suppress even authored content labels. Design §5 now records
the concrete scalar, object, array, nested-array, typeless-union, per-branch item
and explicit-false cases. No accepted ticket scope is deferred by this correction.

The importer replaces mediaDefaults and DefaultMediaType handoff with preserved
source type and array item facts on the existing schema graph. Attribute-local
http:encoding:sourceType overrides distinguish a normalized typed array union
from a typeless union of arrays, and protect typeless singleton/nullable wrappers.
Required ref/allOf type inspection is separate from alternative branch inference.
The shared codec chooses application and media; the exporter must retain this
same application rather than impose a shared label across distinct item defaults.

The red source-fact regressions reproduced loss of absence, singleton provenance,
nullable source array type, nullable-composition provenance and allOf references.
Current bounded results and exact source hashes are recorded under
/tmp/loom-1014-source-application pending independent review. No proof tool is
added to normal checks or CI.


The final source/application selection passes in 1.307s: 26 source/media cases,
normalization and reference provenance, shape adapter agreement, four multipart
namespace controls, four binary application controls, occurrence isolation and
the existing 44-row behavior matrix. The prior binary-location suite passes in
0.690s with repeated binary parts corrected to accepted; nested JSON binary
locations retain their diagnostic. Multipart raw permission now follows the
resolved codec at `mediaFieldEncodings`/`multipartSchemaEncoding`. The obsolete
blanket multipart-body role is removed from generic schema traversal.

The explicit text, source-scalar-default and explicit style generated controls
passed. After the runtime field-application API landed, specimen 40 and the
inherited array numeric-source generation regression pass together in 20.976s.
Specimen 40 also checks exported whole-object JSON representation, libopenapi
validation and independent-process regeneration. This does not certify the
remaining shared array-item/root-form/multipart transport generation work.

A bounded before/current render comparison keeps supported.yaml byte-identical.
Specimen 40 replaces the old body-field defaultContentType metadata with
attribute-local sourceType=object; no other emitted bytes change. Both specimens
match across independent current processes. A separate alias regression first
reproduced unnecessary metadata/reference overlays, then confirmed that ordinary
aliases inherit equivalent source facts while a typeless singleton reference
keeps its necessary override. The corrected Lean context model passes, and
scoped importer golangci-lint reports zero issues. Temporary review sources,
logs and hashes remain in /tmp/loom-1014-source-application until review cleanup.
