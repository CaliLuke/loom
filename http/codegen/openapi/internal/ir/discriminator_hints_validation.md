# Discriminator hints and captured form representations

This implementation preserves the discriminator contract in
`docs/openapi-import-coverage.md` and `docs/dsl-reference.md`. Hints describe
schemas; they never route runtime union selection, change accepted values or
turn alias keys into wire tags. Importer admission, source identity, partial
selection and roundtrip evidence are maintained by the importer owner.

## Shared ownership corrections

- Named union metadata is applied before type publication. Pending constructor
  references complete through one idempotent occurrence lifecycle, including
  copied refinements and functionless Types. Early completion visits only the
  current declaration's owned inline attributes; it does not change ordinary
  named-declaration scheduling or kept structural back-reference identity.
- HTTP body edge removal captures only effective discriminator metadata and
  sourceType. Occurrence values override inherited values, copied slices are
  detached, and ParentHint declaration identity remains independent of metadata.
- ParentHint explicit mappings require direct CompositionBases children. Literal
  keys, including empty keys and `=`, survive. Required additional-member names
  count toward discriminator requiredness. Implicit children need no synthetic
  mappings: the named root that admits the hint records its direct allOf
  children as publication dependencies, which component pruning and
  registration arbitration follow. A projected parent publishes its children,
  composition contributors and mapping/default targets in its own
  representation; a JSON occurrence projects a parent whose publication
  context has byte fields even when its own body has none, and the allocation
  quotient compares those dependencies (`tla/discriminator_reachability`). The implicit-only
  regression checks child allOf and an empty mapping without force-generation
  metadata.
- Named occurrence overlays keep discriminator policy on the complete named
  component, including local mapping/default/extension overrides. The reference
  keeps local annotations and constraints. Metadata maps and values are detached;
  another occurrence keeps its original hint and component identity.
- Native tagged form capture has separate TaggedBranches; it does not become
  untagged exact-one dispatch. Carrier planning keeps native scalar arrays native
  and preserves JSON layout inside complex fields. Schema/media examples share
  the same detached flat-root projection; shared JSON envelopes are untouched.
- `internal/formgen` owns captured JSON/native leaf calls, item/value application,
  nullability and physical presence. Ordinary JSON fields, ordinary Content
  alternatives and generated tagged members consume it. Whole-object generated
  reflection delegation is removed. HTTP names remain allocated once while
  immutable form descriptors bind separately to client/server physical layouts.
- Selected repeated arrays may emit zero value keys. Their tag proves branch
  presence; decoding materializes nonnil empty arrays and then applies minItems.
  Nil versus nonnil empty is not distinguishable on this wire. Complex arrays
  encode one JSON item per key, including an inner array as one JSON item.
  Nullable complex members and maps retain explicit JSON null. Ordinary object
  fields still require their own key evidence.

The form restriction on nested untagged unions under a tagged root remains an
admission negative, as do unsupported styles, nullable flattened objects,
nullable repeated arrays, typed additional fields and discriminator-key
collisions. JSON-only declarations acquire no form admission requirement.
Explicit custom field codecs still receive their opaque values; generic runtime
reflection policy is unchanged.

## Design evidence

Models remain opt-in and do not enter build/test/release gates. The model READMEs
record assumptions, counterexamples, commands and implementation correspondence:

- `expr/tla/type_copy/README.md`: effective metadata capture and isolation.
- `dsl/tla/constructor_reference/README.md`: pending publication/completion and
  declaration-owned traversal; ordinary kept-reference scheduling is separately
  covered by maintained implementation regressions.
- `internal/httpcodec/tla/media_contracts/README.md`: native versus JSON layout.
- `codegen/uniongen/tla/form_selection/README.md`: tag presence, missing/null/value
  keys, empty normalization and constraints.
- `internal/formgen/tla/captured_fields/README.md`: lost item application and
  nullable-member policy, plus the extraction-introduced shared-descriptor
  regression. Detached package binding checks both client/server orders.
- `tla/discriminator_overlay/README.md`: complete hint ownership, reference-local
  annotations and detached override/source isolation.
- `tla/tagged_form/README.md`: closure/cardinality pullback and collision/null
  admission preconditions.

An attempted Content-union tagged-root positive failed existing admission before
runtime and was corrected to a negative control. An early custom branch-metadata
probe did not produce an opaque Go type in either baseline or candidate, so it
was not counted as opaque-codec coverage. The actual documented custom field
probe produces *time.Time, compiles at both sources and retains the native field
codec call. These fixture corrections are distinct from reproduced defects.

## Implementation checks

Toolchain: Go 1.27.3 darwin/arm64, libopenapi v0.42.3 and pinned Vacuum 0.39.3 with
`http/codegen/openapi/v3/testdata/vacuum.yaml`. All external generated modules use
an explicit local source override while the ticket is unpublished. No dependency
source was changed and no new dependency was introduced.

The focused checks include:

- Full DSL tests after the owned-completion correction (0.468 s); full expr,
  uniongen and formgen checks after captured lowering (1.366/0.880/0.094 s).
- Full service package run identified only two expected form-code golden changes.
  The existing golden owner regenerated exactly those two fixtures; all TestService
  cases then passed (1.378 s). Other service tests in that full run passed.
- Generated hinted/unhinted wire, ordinary Content selection/ambiguity and native
  OAuth controls passed. After per-package binding, direct order/isolation checks
  passed (0.723 s), generated/IR controls passed (44.966/0.306 s), and final
  discriminator/schema direct checks passed (IR 4.187, v3 0.661, HTTP 5.313 s).
  Checks cover actual client publication, form bytes, server decoded state, JSON
  response/client result, missing/null/malformed values and minimum constraints.
- Scope lint passed with no findings; file-size and documentation gates passed.
  Formatting-only edits are included in the final review manifest.

## Generated artifacts

The same existing eight-specimen driver generates and compiles baseline/candidate
modules and regenerates each candidate in an independent process. It never edits
or normalizes generated artifacts; module names and CLI basenames are identical.
Ordinary immutable baseline outputs are reused when their source/input identities
are unchanged. The two expanded new fixtures receive fresh baseline output.

The six ordinary specimens contain 109 files. Ninety-one are byte-identical;
18 Go files intentionally change only generated form methods from whole-object
reflection delegation to captured field calls and presence. They occur in:

- ResultBodyUnionCustomKeysMultiDSL: service, HTTP client types, HTTP server types.
- OpenAPI32FeaturesDSL: service, HTTP client types, HTTP server types.
- OpenAPIClosedObjectsDSL: service, client CLI types, HTTP client/server types.
- MealPlannerDSL: service, client CLI types, HTTP client/server union types.
- PayloadFormBodyUnionDSL: service, client CLI types, HTTP client/server types.

PayloadBodyUnionCustomKeysDSL remains entirely byte-identical. Ordinary OpenAPI
JSON/YAML artifacts retain exact bytes; the prior 24-file 3.1/3.2 baseline check
remains valid because later lowering changes do not change those schemas.

DiscriminatorHintDSL and DiscriminatorNoHintDSL each contain 24 files, with 14
intended differences: service and generated form methods, separately allocated
native/JSON body carriers and their conversions/validation, CLI examples using
the captured type/value keys, flat occurrence OpenAPI schemas/media examples,
hint metadata where authored, and the resulting loom.json design digest. The
per-file comparison manifest lists every changed path. All eight candidate apps
compile and all 157 generated candidate files are process-deterministic.

Both new specimens are also rendered independently for OpenAPI 3.1.1 and 3.2.0,
JSON and YAML. All eight files pass libopenapi, the pinned Vacuum rules and an
exact-byte comparison across independent rendering processes. The maintained
schema-membership oracle validates rendered flat examples against their schemas.

## Delivery status

These are local implementation results, not prerequisite delivery or review
approval. The exact prospective reference-resolver revision, source manifests,
raw failures, terminal commands and comparison artifacts accompany the review
packet. Fresh independent review and the prerequisite's actual atomic delivery
remain required before this ticket's one atomic integration commit.

## Native review overlay correction

The shared analyzer accepted an annotated named occurrence but applied its hint
on the reference wrapper. The independent review reproduced a branch-index panic
for a hinted union and a false named-object error for ParentHint. The corrected
owner partitions component identity/hint metadata from reference annotations;
it does not suppress the hint or mutate the named declaration.

The maintained direct regression covers both `$ref` and explicit `allOf` wrappers,
local overrides and inherited hints, default/mapping/extension fidelity, local
required constraints, source isolation and component reuse. The old behavior is
preserved as failing scratch evidence. Two invalid test assumptions (object
length validation and extension storage type) were corrected separately; they
are not framework defects. Full IR tests passed (9.142 s), focused IR/v3 consumers
passed (0.720/0.716 s), and explicit JSON test events confirm both new top-level
tests and all six cases ran (0.217 s). Scope lint passed with no findings.

A bounded rerun of OpenAPIClosedObjectsDSL, DiscriminatorHintDSL and
DiscriminatorNoHintDSL compiled all three generated applications and regenerated
in separate processes. All 64 files match the frozen native r1 output byte for
byte, with no intended generated delta for these controls. Prior parser/Vacuum
validation remains applicable to those exact specification bytes; focused v3
consumer checks also revalidate current rendered output with libopenapi. This
slice does not claim to fix or validate the separate JSON-query publication or
implicit-child pruning findings.
