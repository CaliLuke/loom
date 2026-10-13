# OpenAPI reference contract comparison — #812

Status: implemented; acceptance complete, pending fresh final review and root
integration. Independent readiness approved the seven-file manifest
`eb37ac4b57e86d04d983184acc41d19ba1d51063f85cd958aa8a3e34dedf3e85`;
root released implementation. Parent `55cd4ca779f2535c2f4a644162289a265de4a830`.
Worktree: `/Users/luca/.codex/worktrees/import-export-reference-contract/loom`.
Issue: https://github.com/CaliLuke/loom/issues/812.

## Sequential tasks

1. [x] Reproduce the generated symmetry failure and inspect actual source/output.
2. [x] Establish importer/exporter and comparison contracts without weakening
   authored-source admission.
3. [x] Reproduce false acceptance and shared-reference loss directly; compare
   current and proposed ownership in the bounded model.
4. [x] Independent readiness review, then root implementation release.
5. [x] Implement one lossless comparison owner and remove destructive inlining.
6. [x] Run the bounded acceptance below; freeze exact diff and evidence.
7. [ ] Fresh final review, then root atomic integration/commit/push/closure.

## Evidence and scope

`TestOpenAPIImportExporterSymmetryRoundTrip` imports the existing substantive
`internal/openapiimport/testdata/symmetry.yaml`, generates and compiles it, then
fails when strict analysis sees a response schema with `$ref` plus `example`.
The generated component has the same `id: item-2` example. Named response examples
and the reusable Item example remain present. This is legal exporter output:
`analyzePreparedSchemaPositions` and `analyzeUserTypeOverlay` retain occurrence
annotations beside references. Existing representation-equivalence models and
annotation tests intentionally cover that behavior.

The strict importer deliberately rejects schema-reference siblings, documented
in `docs/openapi-import-coverage.md` and tested by
`TestSchemaReferenceSiblingsAreExplicitlyRejected`. Single-reference `allOf`
accepts a bounded set of overlays. This task does not expand either public rule.
The CLI comparison already removes verified generated server/dialect defaults,
normalizes generated components, and recognizes duplicated response annotations.
The coverage guide promises comparison of regenerated contracts, not closure of
strict import grammar over every generated spelling. Original symmetry commit
`690a5a5bd` supports this interpretation.

A readiness-only probe re-expresses the generated reference as single-reference
`allOf` while preserving its local example. Strict analysis then succeeds and
comparison passes. Changing only that example to `unexpected-local-value` still
passes: `inlineGeneratedResponseSchema` replaces the entire occurrence with its
component. Four direct reds prove examples, descriptions, enums and maximum
lengths are erased. Their pure-reference controls pass. A fifth direct red shows
that two references to one generated component fail because the first resolution
removes the target. A sixth red shows generated-input preparation rounds
`18446744073709551615` to `18446744073709552000` before analysis; the existing
exact typed-clone controls still pass, so the lossless boundary must include raw
preparation. These are shared comparison-owner failures, not production
importer or exporter defects.

The four false-acceptance cases also fail against actual main
`8b8dccf7274e0c4ce4fd76cf3c9c023a0f07f411` using a Go overlay; main was not edited.
The model under `cmd/loom/tla/import_contract_comparison` reproduces both defects
and supports retained registry ownership. Its README records precise assumptions;
it is not a proof of arbitrary JSON Schema equivalence.

## Proposed owner and limits

Keep all changes in the CLI contract-comparison test support and its tests/model.
Do not modify production `internal/openapiimport` admission, IR representation
selection, generated output, or source fixtures to suppress differences.

The comparison adapter has two phases:

1. Prepare generated input using schema-aware, lossless representation changes.
   A schema `$ref` with supported siblings becomes one `allOf` reference, retaining
   every sibling. Visit actual schema positions and structural children only;
   example/default/extension literal objects containing `$ref` are data. Preserve
   exact numeric tokens through the entire preparation path, including the
   existing generated-document-default removal helper. Existing `allOf` or an
   unsupported ambiguous combination must produce a comparison failure rather
   than be flattened or have keys discarded. Strict analysis remains the gate
   after preparation. The authored source still enters unchanged strict analysis.
2. Resolve generated response occurrences from an immutable component registry
   into detached comparison schemas. Retain every occurrence constraint and
   annotation. Disjoint fields or exactly equal redundant fields may combine;
   incompatible overlaps must fail normalization explicitly instead of selecting
   one side. Do not implement a general JSON Schema solver. Retain targets until
   all relevant occurrences resolve, and remove only generated components proven
   unreferenced and absent from authored components. Remaining authored references
   and nested references must prevent pruning. Missing references and cycles fail
   closed when expansion is required. Resolution cannot depend on response order.

Duplicate schema/media examples may be removed only after proving their complete
meaning is already represented; unique annotations or changed values must remain
visible to comparison. Preserve existing verified generated server and dialect
handling. Consolidate the two phases as comparison-owned support rather than add
per-fixture branches. Source documents and component records remain detached.

Accepted examples: a pure generated component replacing an inline schema; two
responses sharing that component; the actual symmetry response's redundant
`item-2` annotation; an exactly equal occurrence constraint retained through
reference representation. Rejected examples: changed local example or description,
added enum or length restriction, unresolved target, ambiguous conflicting overlay,
and unsupported schema keywords still rejected by strict analysis. Literal data
with a `$ref` key must remain unchanged, not be interpreted as a schema reference.

No user policy choice is needed: a comparison must not turn a changed contract
into equality. No claim is made that all valid OpenAPI can be imported.

## Bounded implementation acceptance

- [x] All six direct reds become green, including pure-reference healthy controls.
- [x] Cover matched overlays, changed overlays, defaults and vendor annotations,
  shared references in both orders, nested retained references, authored component
  retention, missing/cyclic references, ambiguous overlaps, idempotence and source
  detachment. Every Schema field must be retained or explicitly rejected; adding
  a future field must not silently weaken comparison.
- [x] Verify raw preparation at schema positions and literal `$ref`/`allOf` data;
  retain large integers and extreme decimal/exponent values through preparation
  and the existing exact contract clone. Unsupported keywords and conflicting
  composition still fail clearly. The environment-dependent exploratory
  probe was removed; actual symmetry generation now tests changed-example rejection.
- [x] Run the three existing CLI round-trip tests: semantic 3.1.1/3.2.0,
  shared-status strict/lossy, and exporter symmetry. Keep every existing comparison
  assertion. Assert a changed example in actual generated symmetry output fails.
- [x] Run affected fast comparison and strict-import reference/composition tests.
  Use existing importer/exporter fixture tests as controls, not a full corpus.
- [x] Compare the existing symmetry fixture's imported design and generated files
  byte-for-byte at base/candidate and in a separate candidate process. Since only
  comparison support changes, intended production output differences are none.
  Build/vet generated module once per distinct source/output, using verified
  pushed pin plus explicit local source. Existing CLI tests also compile modules.
- [x] Validate rendered symmetry JSON/YAML with existing libopenapi/Vacuum checks;
  preserve their established contract expectations. Reuse valid evidence for
  unchanged bytes. Repeat the bounded model and record correspondence.
- [x] Freeze exact files/hashes, scope, intended differences and evidence for fresh
  review. No production policy changes, unrelated fixes or broad corpus run.

## Reproduction and evidence inventory

Current parent failure:

```sh
LOOM_OPENAPI_CONTRACT=1 LOOM_DIR="$PWD" go test ./cmd/loom \
  -run '^TestOpenAPIImportExporterSymmetryRoundTrip$' -count=1

go test ./cmd/loom -run '^TestGeneratedResponseInlining' -count=1 -v
```

The first command regenerates from the durable fixture and now also checks that
changing the generated response's local example fails semantic comparison. The
second command runs the direct reference-resolution regressions. Both pass with
the repair; the recorded parent failures remain historical evidence.

The removed `TestReferenceSiblingComparisonReadiness` and its
`LOOM_REFERENCE_READINESS_OUTPUT` variable were exploratory readiness probes only.
Their archived log below records the original false acceptance; they are no longer
runnable. Use `TestOpenAPIImportExporterSymmetryRoundTrip` above for the maintained
actual-generated changed-example regression.

Ephemeral evidence (delete after final review once summarized):

- `/tmp/loom-import-export-reference-red.log`: exact parent generated failure.
- `/tmp/loom-import-export-reference-comparison-probe.log`: actual output changed
  example false acceptance after lossless-for-this-fixture reference rewrite.
- `/tmp/loom-import-export-reference-direct-red.log`: four overlay reds and one
  shared-reference red, with pure-reference controls passing.
- `/tmp/loom-import-export-reference-numeric-red.log`: raw preparation numeric
  rounding; `/tmp/loom-import-export-reference-controls.log`: existing loss
  detection and exact typed-clone controls pass.
- `/tmp/loom-import-export-reference-main-red.log` and matching overlay JSON:
  four false-acceptance reds on main `8b8dccf7274e0c4ce4fd76cf3c9c023a0f07f411`.
- `/tmp/loom-import-export-reference/`: generated fixture module/output, logs and
  verified pin metadata. Original generation used pushed4b459a121 with an explicit
  local replacement; final acceptance will use verified55cd4ca779f2 pin
  `v1.10.0-alpha.9.0.20261013003626-55cd4ca779f2` plus explicit source selection.
- `/tmp/loom-reference-model-{drop-overlay,eager-prune,retained}.log`: checker
  counterexamples and candidate result; durable source/configurations are tracked
  in this task. Java25.0.4.1, TLC2.19, Go1.27.2.

## Implemented owner and final evidence

All implementation is test support under `cmd/loom`; production import/export and
all checked-in input specimens remain unchanged. `prepareGeneratedOpenAPIContract`
verifies generated server/dialect defaults and preserves raw JSON values while
following schema positions. It rewrites reference siblings without dropping
keys; ambiguous existing `allOf` fails. `comparisonSchemaResolver` reads a retained
registry, clones occurrences, merges only disjoint/equal fields, then prunes
unreferenced generated targets after resolving all responses. Typed model traversal
ignores literal interfaces when finding references and keeps authored/nested uses.
Unknown exported schema fields join the generic merge; nonzero inaccessible fields
are rejected. The owner assumes strict analysis already rejected unsupported
composition before constructing the typed comparison model.

Implementation exposed a second precision loss in interface-valued literals:
`18446744073709551615` and `18446744073709551614` defaults compared equal after the
old clone. The direct red is preserved in `/tmp/loom-reference-812-literal-red.log`.
The clone now uses the same exact-number owner for nested defaults, enums, examples
and extensions, preserving nil collection layout and detachment. Comparison JSON
uses deterministic map ordering. Numeric bounds keep their existing exact decoder.

Acceptance evidence:

- `/tmp/loom-reference-812-final-tests.log`: direct regressions/controls and all
  three generated round-trip families pass, including generated changed-example
  rejection. Scalar literals, nested arrays/maps and exact numeric bounds remain
  distinct. The generated fixture is compiled by the existing CLI tests.
- `/tmp/loom-reference-812-import-controls.log`: strict reference-sibling rejection,
  supported/rejected single-reference compositions and exporter metadata symmetry
  controls pass. No public admission test was weakened.
- `/tmp/loom-reference-812-vet.log`: affected CLI package vets successfully.
- `/tmp/loom-reference-812-acceptance/comparison.json`: all 17 design/generated
  files match the preserved parent output and a second independent candidate
  process byte for byte. An initial comparison used `-o .` only on candidate,
  causing command-comment differences; the corrected comparison uses the exact
  original invocation for both. No intended output differences remain.
- `/tmp/loom-reference-812-acceptance/candidate.log`: generated module build/vet
  pass with verified pushed55cd pin and explicit local source. The repeat reuses
  this validation for identical output rather than rebuilding it redundantly.
- `/tmp/loom-reference-812-spec.log`: generated JSON/YAML pass libopenapi model
  construction as OpenAPI3.2.0. `/tmp/loom-reference-812-vacuum-{json,yaml}.log`:
  both pass pinned Vacuum0.33.0 with the repository's rules and exact libopenapi
  override, zero errors (seven warnings and one informational notice each).
- Model sources/configurations are unchanged from the successful readiness run:
  two current-behavior counterexamples and retained ownership20states. Reuse those
  checks; the README now links the implemented seams and unmodeled tests.

Intended differences are exclusively comparison behavior: supported reference
spelling reaches semantic comparison; local overlays cannot disappear; shared
components resolve independently of traversal order; numbers retain exact meaning;
ambiguous expansion fails rather than reporting equality. Application generation,
wire behavior and production strict import/export policies do not change.

Fresh review and integration remain root-owned. Temporary evidence and tool build
paths remain available until review finishes; then remove task-owned scratch while
retaining these regression/model sources and concise findings.

## Main integration

Root independently reviewed the thirteen-file candidate and corrected one low
finding in recovery instructions; code and evidence hashes stayed unchanged.
The three generated round-trip families and selected direct controls pass in
14.896s; scoped CLI lint reports zero issues. Candidate manifest after re-review:
`44864b005080528bfe974c70142f5e7abdd451efbfcc808860caf8b8c07890df`.
Root integrates against `abeb200e9`, preserving candidate code and adding only
shared framework/model-index guidance and this receipt. Final exact integration
review, atomic commit, push and issue closure remain pending.
