# Effective YAML indexing — #922

Status: dependency implementation reviewed and published as 7886f0170cc4. Local
importer/generator acceptance passes with the separately delivered #925 extension
correction. Framework adoption and downstream published-source checks remain.

1. [x] Preserve six-shape red/control matrix on released v0.41.6 and published
   immutable-view fork 856f6f2d; record source pointers and lookup ordering.
2. [x] Model raw traversal, global visited-node suppression and path-local ancestry;
   establish effective occurrence coverage, source preservation and termination.
3. [x] Independently review readiness and bounded acceptance before implementation.
4. [x] Add failing index/public regression tests, then route fragment lookup,
   component enumeration and reference/schema-ID traversal through the existing
   immutable effective mapping owner. Preserve authored source and metadata.
5. [x] Run focused index/alias/reference/model race checks and representative
   public merge/normal/invalid controls. Compare repeated importer/rendered output
   with equivalent explicit YAML; compile selected generated code.
6. [x] Independently review and publish exact final fork diff; PR #658 and upstream
   discussions remain unchanged.
7. [ ] Independently review atomic framework dependency integration, publish, and
   verify an ordinary downstream consumer before closing #922.

## Confirmed boundary

`createNewIndex` resolves extracted references before `BuildIndex` fills direct
component maps. `navigateToFragment` reads raw mapping contents. Some inherited
components appear in those maps only after the failing lookup. Others never do:
`extractDefinitionsAndSchemas` registers an authored merge key `<<` as a component,
while an aliased whole components map has no direct Content. `walkExtractRefs`
also ignores alias occurrence traversal. Reordering stages or patching only
fragment lookup would leave these consumers inconsistent.

Candidate: reuse `utils.ResolveMergeNodes` for effective node reads at index
traversal and collection boundaries. Keep source node pointers for provenance;
never flatten or mutate the source document. Resolve aliases before deriving
schema-ID scope or inspecting keys. Recursion tracks original source ancestry
per active path, so sibling reuse of an anchor is retained while cycles stop.
A global visited set would incorrectly suppress the second semantic occurrence.

The model will abstract existing merge expansion as correct, pure and terminating.
Concrete tests own precedence, escaped pointers, source metadata, race safety and
actual schema-ID/reference semantics. Valid recursive `$ref` graphs remain a
separate existing mechanism; cycle handling here is YAML node traversal only.

## Bounded acceptance

- Six shapes: explicit map; merged components; aliased components; merged schemas;
  aliased schemas; merge inside an explicitly named schema. Check model build,
  index errors, component names and direct lookup both before/after full build.
- Direct alias, merge sequence/aliased sequence, local override and earlier-source
  precedence; repeated anchor at distinct paths and malformed cyclic aliases.
- Effective occurrence paths for inherited `$ref` and `$id`; preserve missing-ref
  rejection, existing encoded-pointer/nested-fragment/array and recursive controls.
- Original YAML graph unchanged; original key/value line/comment pointers retained.
  Race-enabled public model construction and low/high metadata/render controls.
- Loom merged versus explicit import yields equivalent DSL/OpenAPI exact bytes;
  independent-process repeats deterministic; generated design/service compilation.
  Reuse existing importer fixtures/tests, not an exported-design corpus sweep.

Existing evidence is `/tmp/loom-libopenapi-index-control` and
`/tmp/loom-libopenapi-index-shapes.log`; issue #922 preserves the minimal input.
No upstream communication is authorized by this task. Root reviews/publishes any
final fork commit and owns the atomic framework adoption commit.

## Readiness evidence and implementation boundary

Published fork 856f6f2d is clean; upstream PR #658 remains untouched. Six-shape
public controls run against both released v0.41.6 and that exact fork. Four
container alias/merge shapes fail identically; explicit components and a merge
inside an explicitly named schema pass. After construction, merged-components
and aliased-schema-map lookup succeeds too late; merged-schema maps instead
contain a false `<<` component, and aliased components have no entries. Receipt:
`/tmp/loom-922-shape-matrix.log` and public source
`/tmp/loom-libopenapi-index-control/probe/main.go`.

The model records four rejected approaches/configurations and two passing
mapping/sequence controls. The independently identified shared helper hazard is
confirmed: `x-sequence: &sequence [*sequence]` parses, then entering
`ResolveMergeNodes` on `components: {<<: *sequence}` does not return within the
isolated 0.2-second probe. Source inspection confirms recursion follows the same
sequence forever because only mappings enter the ancestor set. The subprocess
was killed; no production process or global cache changed. Receipt:
`/tmp/loom-922-sequence-probe-receipt.json`. This expands the shared effective-view
prerequisite to sequence ancestry, not an index-specific workaround.

Proposed production seams:

1. `utils` effective merge expansion: guard canonical sequence ancestors before
   following elements; preserve local/first-source precedence and source identity.
2. `index/schema_id_resolve.go`: descend effective mappings/aliases at each finite
   fragment step; retain existing JSON Pointer/array grammar and source metadata.
3. Component collection boundary in `spec_index_counts.go`/`utility_methods.go`:
   enumerate effective containers uniformly instead of registering merge keys.
4. `ExtractRefs`/`walkExtractRefs`: canonical active-path ancestry and effective
   occurrence traversal before scope/key extraction; preserve raw example-payload
   exclusions and valid sibling reuse. Do not use a global visited-node set.

The public contract is already settled: preserve supported effective YAML values
and local references while keeping missing-reference errors. No new DSL or
transport semantics are proposed. Tests must establish termination on malformed
cycles without promising that cyclic YAML denotes a valid OpenAPI document.

## Implementation and current evidence

Independent readiness passed against published fork 856f6f2d. The final candidate
is based on 819f0942, which separately corrects a Windows-path test fixture left
inconsistent by the module rename. The index correction changes no module path
or dependency versions itself.

The shared low/v3 component collection extractor also needed effective collection
reads: after index repair, a schemas-level merge built without errors but silently
omitted `Bound` from the model. Inline schema collections now resolve their child
views too. Both are covered by concrete model/index regressions.

Full affected dependency package race tests pass: `index` (164 s), `utils` (3 s),
`datamodel/low/v3` (10 s), and the root public model package (20 s). Six dependency
model shapes pass, with all ten indexed component collections, source identity,
render/reload, repeated alias paths, schema-ID scope, inline collections, pointer
controls, existing invalid references and recursive references covered.

The Loom matrix is explicitly five container shapes. A sixth schema-value merge
builds at the dependency seam but Loom's raw-proxy grammar reader rejects `<<`;
that independent semantic-inspection owner is tracked in #924. The raw proxy API
was preserved. This ticket does not claim that schema-admission defect is fixed.

Three process-isolated explicit/merged/repeated imports generated compiling apps.
Their exact-byte comparison exposed a separate existing nested-extension JSON
serialization defect: `renderedExtensions` omitted deterministic encoding. The
failed comparison was preserved. After the separate #925 owner correction, the
exact-byte gate passed across all three processes. No extension fields were
dropped or output normalized to hide differences.

The unchanged dependency candidate was independently approved against parent
819f09426ceaa19ea8b04c6901e70e3c9b29344b (11 files, exact full-index patch SHA-256
`d3b0f126051869ac5978908296c732521e9bd6c6cabb0bc5a13aeadb197a068a`).
The separate #925 shared renderer correction then made all 17 design/generated
files byte-identical across explicit, merged and process-isolated merged repeat.
Only deterministic extension strings in `design/design.go` and the corresponding
`gen/loom.json` digest changed from the preserved prior output. OpenAPI and all
service/transport bytes stayed identical. Generated compilation passes, the five
importer shapes pass under race detection, and generated JSON/YAML pass both
libopenapi model construction and pinned Vacuum contract rules (zero errors).

Published module: `github.com/CaliLuke/libopenapi`
`v0.41.7-0.20261014145029-7886f0170cc4`, source commit
`7886f0170cc4ff64f4be95a18a2576767989aaa2`. An ordinary external module fetches
that version without replacements and passes all six public component-shape
model tests under race detection (1.807 s).
