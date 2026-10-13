# Projected field carriers (#887)

Status: implemented against delivered `7ed1bfc63`; final review pending.

## Sequence

- [x] Preserve #689 source, inputs and compact evidence; safely reverse its exact
  reviewed patch and fast-forward the reused worktree.
- [x] Retain the original minimal named-slice/view compiler witness and reconcile
  the earlier union validation (#796), selected-body (#854) and selected-view
  presence (#834) owners.
- [x] Reproduce direct context/binding disagreement and the original generated
  failure; retain ordinary scalar, array and non-view alias controls.
- [x] Independent readiness review of the shared correction and bounded consumers.
- [x] Implement, compare selected generated outputs and independently repeat;
  compile and exercise the corrected named-view header wire path.
- [ ] Independent exact final review, atomic delivery and temporary-output cleanup.

## Owner and correction

`AttributeContext.IsPrimitivePointer` currently returns true for any field when
`Pointer` is true (except Any/direct Bytes). It therefore labels missing fields,
arrays, maps, objects and unions as primitive pointers. HTTP's
`transportFieldBinding` consumes that flag for projected header conversion and
emits a dereference of the named slice. Actual declarations use
`expr.ServiceFieldLayout` through `NameScope.objectFieldTypeDef`.

The proposed correction makes the shared context query describe the physical
primitive occurrence, using the existing declaration owner. It must first locate
the actual field and classify its type; nonprimitive pointer storage is not a
primitive pointer. Nullable/explicit presence carriers must retain their distinct
representation. JSON transport presence remains owned by `FieldPresence`; do not
infer its wrapper from optionality alone or alter declared types to fit a flag.
Preserve named Bytes' actual declaration if its context currently declares a
pointer; this ticket corrects query/declaration disagreement, not unrelated
layout policy. No header template exception or new storage classifier is needed.

The selected body helper `serviceFieldPrimitivePointer` already uses canonical
layout after #854. Once the shared context query has the same native contract,
its duplicated classification can be removed if equivalent across its existing
controls. #834 owns nullable selected-view validation/calls/envelopes, which
remain separate consumers of physical storage. Its representation semantics and
public absence/null contract do not change. #796 supplies the correct physical
context to union-branch validators; it does not classify transport-bound fields.
The current failure is therefore not a reopened instance of either validator bug.

## Bounded consumer inventory

- `AttributeContext.IsPrimitivePointer`: direct primitive, named scalar, missing
  field, array/map/object/union, Any/Bytes aliases, required/default flags and
  explicit nullable/JSON-presence occurrences.
- `transportFieldBinding`: request path/query/header analysis, response/error
  header binding and selected error origin use the same field identity/type.
- Transformation/default owners: primitive object initialization, native source
  presence guards and default assignment query this method; most already guard
  primitive kinds and `FieldPresence`. Preserve their existing scalar behavior.
- `serviceFieldPrimitivePointer`: selected body source/target construction derives
  from canonical declarations; retain its existing #854 focused seam controls.
- Header encoders/decoders consume recorded pointer flags. HTTP and JSON-RPC share
  analysis; JSON-RPC's writer consumes the same metadata. gRPC response metadata
  constructors and protobuf primitive conversion also query the shared context: retain their existing binary metadata and primitive/presence
  seam controls. This correction changes no protobuf declaration policy.
- `problemRepresentation` uses the context for selected runtime error fields;
  mapped SSE schema codec selection queries it only after native `FieldPresence`
  admission. Preserve both existing representation controls.

## Existing evidence and acceptance

The exact original CLI input and failing generated encoder are retained at
`/tmp/loom-887-original-witness`; the original source pin is `628db82b4` and its
source archive remains `/tmp/loom-689-parent-source.tar`. Generation succeeds,
then `[]string(*res.Projected.Tags)` fails because `views.TagsView` is a slice.
This failure predates header framing; #689 separately proved valid non-view
aliases and plain-array views. Reuse that minimal design rather than a new
framework harness.

Add direct context and binding tests before production. Compare expected flags
against the actual declaration layout, retaining pointer scalars and the distinct
wrapper/collection/object rules. Reproduce the original actual CLI build on the
current pushed parent; compile plain-array-view and non-view-alias controls.
Final acceptance uses the same three designs plus a meaningful generated wire
check for the combined case, the existing selected-body/selected-view seams and
one shared JSON-RPC analysis control. Compare bytes and independently regenerate;
list every intended artifact change. Reuse #689's three-fixture capture for the 55 checked-in HTTP/RPC generated files;
expect unchanged bytes and compile them. No full design corpus.

The existing `http/codegen/tla/response_field_storage` model explicitly rejects
blanket context Pointer as a declaration classifier and accepts layout-derived
storage. Its assumptions already cover this decision. Reuse its context/layout
runs and direct declaration correspondence; no new model or global carrier API
is needed for this bounded query correction.

## Readiness results

On the recorded parent, native declaration agreement fails for arrays, named
arrays, maps, objects, unions and nullable scalars; a missing field also falsely
reports pointer storage. Scalar/named-scalar, direct/named Bytes and Any controls
pass. Binding fails for named and plain viewed arrays, while non-view aliases and
actual pointer scalars pass. The plain-array flag is also wrong even though its
current template avoids the invalid dereference; fixing only named-header output
would leave the shared disagreement intact.

The original named-view CLI module reproduces the slice dereference compiler
error at the current pushed parent. Plain-view and ordinary-alias modules build
and vet. Their 50 generated files are retained with exact hashes. Existing
Bytes/default Optional/Nullable conversion, selected body storage and selected
view carrier seams pass. These check context-presence boundaries without
reinterpreting `FieldPresence` as an ordinary primitive pointer.

The unchanged context model fails its declaration-assignment invariant in 18
states; the unchanged layout model passes 22. Both results are expected and
reused as evidence of the owning choice, not a new proof of every consumer.

## Implementation and bounded results

`AttributeContext.IsPrimitivePointer` now uses the same native declaration query
as selected-body conversion. The redundant selected-body helper was deleted and
all three selected-body call sites consume the shared method. Mapped field
requiredness uses the resolved authored key. Existing explicit presence wrappers,
named Bytes layout and nonprimitive pointer carriers remain separate contracts.

The three actual CLI designs contain 50 generated artifacts per generation.
Candidate and independent repeat match exactly. The sole parent/candidate delta
is the original named-view server encoder: `[]string(*res.Projected.Tags)` becomes
`[]string(res.Projected.Tags)`. The two ordinary controls are byte-identical.
All candidate modules build and vet. All 55 checked-in fixture outputs match
parent, candidate and repeat bytes; both parent and candidate fixture apps build
and vet. No checked-in generated file changes are needed.

Focused tests use explicit `LOOM_DIR` for the candidate source and exercise native
query/declaration agreement, missing fields, mapped requiredness, viewed bindings,
selected-body conversions, nullable views, Bytes/default presence, gRPC primitive
and metadata conversions, mapped SSE codec admission and shared JSON-RPC analysis.
The mounted named-view HTTP check covers both views: absent, empty and nonempty
server headers, plus empty and nonempty client round trips. Existing context/layout
model evidence remains valid because the model source and contract are unchanged.

## Independent collection conversion limitation (#889)

The full mounted nil round trip exposed a separate shared conversion bug:
`NewShowCardOK(nil)` allocates an empty named slice. The server wire correctly omits
the header, but the generated client constructor turns absence into supplied empty.
The constructor is byte-identical at parent and candidate. An ordinary non-view
named-array module reproduces the failure on archived `7ed1bfc63`; a plain-array
viewed constructor preserves nil. Reports #848 (JSON omission), #550 (nullable
nominal type mismatch) and #104 (gRPC copy performance) do not cover this defect.

[Issue #889](https://github.com/CaliLuke/loom/issues/889) retains the complete
minimal design, actual generated-constructor test and source pin. Its likely
owner is root collection conversion reached through `InitStructFields`; no
header-template workaround belongs here. The exact initially failing mounted
harness and logs are retained in the final packet. This ticket does not claim
client nil round trips pass or weaken the absence/empty contract. The committed
mounted check makes that boundary explicit while retaining server absence and
client supplied-value coverage.
