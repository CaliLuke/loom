# Projected constructor target context (#930)

1. [x] Reproduce mapped named-scalar view constructor failure at the shared HTTP
   response analysis seam and in generated code. Cover header/cookie mappings,
   required/optional values and ordinary service controls.
2. [x] Trace target context selection through type references, field casts and
   constructor finalization. Model the current split declaration/field package
   and the single selected-context alternative before implementation.
3. [x] Obtain independent readiness review. Prefer deriving return references,
   package and nominal scope from the existing selected AttributeContext; remove
   parallel service/view selection rather than adding per-carrier cast fixes.
4. [x] Implement the shared response constructor owner correction. Confirm actual
   allocated-name/import contexts as well as simple package qualification; do not
   assume a package-string repair proves the nominal scope correct.
5. [x] Compare representative parent/candidate generated output, compile and
   exercise header/cookie constructors, and verify independent-process equality.
   Preserve ordinary, selected-body and shared-package controls.
6. [ ] Independent exact-diff review, atomic commit/push, close and reconcile.

The published-parent reproduction is retained in `/tmp/loom-viewed-alias-repro`.
`buildResponsesFromIR` already selects an AttributeContext for the effective
service/view destination. `buildResponseResultInit` independently selects its
return references but carries the original service package into field assignment.
`fieldCode` then qualifies named field casts using that stale package. Trace the
existing NameScope parameter too: package and declaration identity must agree.

No codec, default, validation or public view-selection semantics change. #923 owns
custom mapped text publication; #926 owns custom view validation/defaults. This
case uses an ordinary named String with none of those operations.


## Readiness proposal

The already selected `svcctx` is authoritative. Response analysis will derive
return name/ref/package and constructor naming scope from it, deleting redundant
`viewed`, `pkg` and `scope` parameters where only that selection needs them.
Retain this context privately on the response InitData until `fieldCode`; use its
underlying nominal scope and package for mapped-field assignment. Other InitData
owners continue their existing context path, avoiding a broad unrelated rewrite.
Finalization's generic service-scope argument cannot override a retained response
context. Body transforms already consume the same context.

Acceptance: header/cookie × service/view × required/optional direct cases; an
actual service-name collision where package-only correction is insufficient;
actual generated header/cookie constructor calls with present/absent controls;
ordinary shared-package/import-alias constructor control; selected-body storage
regressions unchanged. Capture parent/candidate rendered output for the direct
matrix, compare exact bytes with only view field-cast identity differences
expected, and repeat generation in independent processes. Compile generated
modules once for each relevant shape, not the full exported-design corpus.


## Acceptance evidence

- Parent e72608c75e9bdcd03341f8a302745932059b3c0e: four projected direct
  cases fail package identity; generated optional view header fails compilation
  on `mappedview.LabelView`. The separate nominal test fails after a manual
  package-only repair: `LabelView` does not match declared `LabelView2`.
- Shared correction changes three production files, retaining selected context
  privately and removing redundant response selector arguments. Other InitData
  owners and public semantics are unchanged.
- Focused direct/generated constructor race matrix:14.965s. Eight header/cookie,
  service/view, required/optional modules exercise supplied empty/nonempty values
  and optional absence; collision module compiles. Existing constructor identity,
  imported alias, selected storage, shared-package compile/vet and server golden
  controls pass under race18.698s.
- Existing selected-body generator suite builds, vets and runs mounted HTTP and
  JSON-RPC plus constructor/absence controls under race:6.574s. Existing response
  body/view client goldens pass0.840s. HTTP codegen lint:0 issues.
- Ten representative designs render166 files. Exactly five client types files
  differ: four projected field casts select the view package, the collision also
  selects its allocated suffix, and their unused service imports disappear.
  Ordinary service and located shared-package outputs remain byte-identical.
  A separate-process repeat of all166 candidate files is byte-identical.
- ConstructorContext model:split6 and package-only7 state counterexamples;
  selected8 state success. Existing storage models are unchanged: this model
  discharges their previously assumed context-selection premise only.

Evidence receipts: `/tmp/loom-930-final-manifest.json`, render comparison and exact
output diff; raw checker states and temporary generation trees remain outside the
repository and are removed after review/delivery. No exhaustive corpus gate was
run or needed for this bounded constructor change.
