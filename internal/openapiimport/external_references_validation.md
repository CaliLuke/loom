# External references (#957) acceptance record

Historical implementation parent: `f7a93576833c70eaa00d7a5bb9884f83b55ec15d` on main.
The comparison baseline is `c7d842ecd20647a0ba51e51828c58f3ccf7594a1`; its
production source is identical to that parent (the intervening commit only
corrects the rmap test recorder). Full-scope design and plan were reviewed
before implementation.

## Atomic delivery

This atomic change completes #957 against parent `b9e92c7c98b2bbc1f082a95bb76de363b55edf27`.
Implementation and bounded qualification are complete. Publication is pending the
normal batch push; this record does not assert a release. The accepted contract
remains available for #1005 and #962.

The final prerequisite rebind preserves all 203 ticket paths. Its only changed
ticket postimage is the historical Vacuum checkpoint clarification below. One
ordinary symmetry HTTP module was renewed after the shared selected-body guard
correction: import, generation, compilation and concrete/absent-parent/absent-body
wire controls pass. Rejected selections publish no response metadata or body.
Independent generation is byte-identical; the only parent/output difference is
the server encoder separating parent absence from selected-member absence.
The two semantic-dialect specimens use the same guard owner and retain their
prior qualification; they were not freshly regenerated for this rebind.

## Historical composition checkpoint

The earlier qualified source was composed against main
`c9da9d9d8eb7f45b09876530d705bc8df9739926` (tree
`09b9085178436269d181af208e45ba6e72e667e5`). It contains the complete
203-path implementation delta; its 7,857-provider implementation inventory SHA-256 is
`bb36c68b9d1a0506ce3c63e3ccc8eb9072ea0b0c142f35518cedb35af03f979c`.
This inventory precedes these documentation-only finalization edits.
This is an integration checkpoint, not the final atomic commit or a release.
The earlier parents and runs below retain their historical attribution.

Independent source review accepts the shared completed-graph example lifecycle
and external-resolution joins. Unresolved examples retain explicit no-value
provenance and their original diagnostics; they are not interpreted as JSON null.
The focused correction passed 149 named checks, including native DSL evaluation.

Prior bounded qualification, reused where unaffected, passed the maintained external CLI roundtrip,
exporter symmetry and identity cases, semantic 3.1.1/3.2.0 checks and three-process
relocation determinism. Five generated modules compile and pass their tests;
77 parent/candidate or independent-generation file pairs are byte-identical.
Eight selected JSON/YAML specimens pass Vacuum v0.39.3 with libopenapi v0.42.4.
These results do not include a passing external-security identity lint gate.

The earlier final-source renewal executed the five imported/generated modules
against that implementation. 87 captured Go/JSON/YAML files include 80 exact prior
outputs and seven intended inherited changes: four nonnullable response decoder
wrappers and three checked selected-body preparations. All imported designs and
all ten OpenAPI specimens are exact prior bytes. An output-only capture directory
was missing in two named cases after their module tests succeeded, so the aggregate
command exits 1. The two subsequent assertions are explicitly reused from the prior
successful run against identical specs/designs and 462 unchanged input/test/parser
files; those interrupted cases are not relabeled as passing. No module is rebuilt
solely to repair the capture setup. The other three contract comparisons execute
successfully on the final source.

The generated identity specimen (SHA-256
`79af94c9f43a188cffcfdfdf8ee03fd09c2965c031e0a0a128c3cadbc467542a`)
now passes the unchanged positive security gate using official Vacuum commit
`b6c4e203d2eae41934087140a4d809dd1cce6d82`, module version
`v0.39.4-0.20261107034229-b6c4e203d2ea`, with libopenapi v0.42.4. The supplied
security document, authored URI, admitted-resource policy and enabled security
rule are unchanged. The result contains no preparation errors or error findings.
The resource parity, invalid/unadmitted input and security negative controls also
pass; a missing scheme still fails the security rule. The v0.39.3 failing receipt
remains historical evidence.

This is the documented required-unreleased-correctness exception for
[Vacuum #1056](https://github.com/daveshanley/vacuum/issues/1056), using the
existing official fix in [PR #1029](https://github.com/daveshanley/vacuum/pull/1029).
It is not a released Vacuum version. The owning
[validation guide](../../.agents/skills/loom-framework/references/openapi-validation.md)
records the exact pin and return-to-stable condition. Final parent binding,
exact-diff authorization and normal delivery gates remain open. Additional
prerequisites require renewing only the evidence they actually invalidate.

## Workflow

Baseline import/gen/compile comparisons precede compile-safe API seams. Each resolver,
bundling, selection and projection milestone records red behavioral tests before its
implementation. Generated contracts, exact-byte process determinism, affected package
checks, lint and required gates precede exact-final-diff independent review and one
atomic commit. No generated edits, proof gates, dependency development or unrelated
prep changes are authorized by this ticket.

## Temporary artifacts

- `PARENT_SRC`: `/var/folders/v7/x61rxqvx60q5w_p9jwgc9t_m0000gn/T/loom-957-parent_src-gs9hvh8p`
- `BASE`: `/var/folders/v7/x61rxqvx60q5w_p9jwgc9t_m0000gn/T/loom-957-base-oqci4hn1`
- `CHANGE`: `/var/folders/v7/x61rxqvx60q5w_p9jwgc9t_m0000gn/T/loom-957-change-tvneeeoc`
- `GEN`: `/var/folders/v7/x61rxqvx60q5w_p9jwgc9t_m0000gn/T/loom-957-gen-ag8pslb6`
- `SPEC_OUT`: `/var/folders/v7/x61rxqvx60q5w_p9jwgc9t_m0000gn/T/loom-957-spec_out-3rila3m7`
- `REF_VACUUM_BIN`: `/var/folders/v7/x61rxqvx60q5w_p9jwgc9t_m0000gn/T/loom-957-ref_vacuum_bin-200la6w5`

Temporary outputs are removed after validation and review; toolchains/shared caches remain.

## Pre-existing work

The implementation worktree started with unrelated modified and untracked
files, including formal-model drafts and untracked roadmap designs and plans.
They were preserved and excluded from this ticket.

## Results

Implementation and the bounded framework checks are complete at the current
checkpoint above. The external-security positive gate now passes with the qualified correctness
pin. Final acceptance remains open for exact-final-diff review and normal
delivery gates.
The original parent above is comparison history;
integration must record the actual current-main parent before authorization.

## Inventory dispositions

The exact plan search returned 146 matches. Line numbers below
refer to the recorded parent/worktree before #957 code edits; every match is assigned.

| File | Matching lines | Disposition |
|---|---|---|
| `docs/code-generation.md` | 1225 | M6 documentation/correspondence update |
| `cmd/loom/import.go` | 180, 182, 211, 213, 259 | M1 Source compile seam, M4 CLI/report threading |
| `internal/openapiimport/selection.go` | 44, 54, 154, 155, 208, 211, 222, 345, 404 | M1 Source seam; M3/M4 complete closure and fail-closed selection |
| `cmd/loom/import_test.go` | 633 | Named regression/call-site update; retain existing behavior controls |
| `internal/openapiimport/analyze_test.go` | 33, 34, 36, 139, 140, 143, 144, 186, 688 | Named regression/call-site update; retain existing behavior controls |
| `internal/openapiimport/grammar_coverage_test.go` | 477 | Named regression/call-site update; retain existing behavior controls |
| `internal/openapiimport/render.go` | 112, 172, 210, 217, 255, 256 | Fragment-only consumer unchanged; M3 bundled pointers/names feed existing owner |
| `internal/openapiimport/names.go` | 20, 26, 30, 36, 40, 61 | M3 shared design-name/error-clone inventory |
| `internal/openapiimport/render_index.go` | 10, 12, 53, 57, 61, 62, 65 | M3 shared design-name/error-clone inventory |
| `internal/openapiimport/reference_identity_test.go` | 50 | Named regression/call-site update; retain existing behavior controls |
| `internal/openapiimport/security_credentials_partial_test.go` | 27, 59 | Named regression/call-site update; retain existing behavior controls |
| `internal/openapiimport/analyze_schema_promotion.go` | 79, 83, 88 | Fragment-only consumer unchanged; M3 bundled pointers/names feed existing owner |
| `internal/openapiimport/byte_format_test.go` | 37 | Named regression/call-site update; retain existing behavior controls |
| `internal/openapiimport/security.go` | 246, 296, 297, 312, 348, 363, 381, 417 | M5 requirement identity and alias-scope union |
| `internal/openapiimport/plan.go` | 46, 57, 90, 95, 104, 115, 131, 135, 137, 149, 150 | M3 shared design-name/error-clone inventory |
| `internal/openapiimport/parameter_admission_test.go` | 119, 132 | Named regression/call-site update; retain existing behavior controls |
| `internal/openapiimport/partial.go` | 42, 43, 126, 162, 163 | M1 Source seam; M3/M4 complete closure and fail-closed selection |
| `internal/openapiimport/partial_test.go` | 41, 84, 121, 161, 194 | Named regression/call-site update; retain existing behavior controls |
| `internal/openapiimport/selection_test.go` | 99, 146, 179, 225, 230 | Named regression/call-site update; retain existing behavior controls |
| `internal/openapiimport/security_credentials_readiness_test.go` | 199 | Named regression/call-site update; retain existing behavior controls |
| `internal/openapiimport/tla/bundle_naming/BundleNaming.tla` | 4, 8 | Fragment-only consumer unchanged; M3 bundled pointers/names feed existing owner |
| `internal/openapiimport/tla/resolution_identity/README.md` | 145 | M6 documentation/correspondence update |
| `internal/openapiimport/tla/security_projection/README.md` | 6 | M6 documentation/correspondence update |
| `internal/openapiimport/tla/bundle_naming/README.md` | 12, 17 | M6 documentation/correspondence update |
| `internal/openapiimport/tla/bundle_naming/single-pass-types.cfg` | 1 | Fragment-only consumer unchanged; M3 bundled pointers/names feed existing owner |
| `internal/openapiimport/name_occupancy_test.go` | 50, 51, 52, 65, 66, 67, 71, 72, 73, 123 | Named regression/call-site update; retain existing behavior controls |
| `internal/openapiimport/unconstrained_test.go` | 278 | Named regression/call-site update; retain existing behavior controls |
| `internal/openapiimport/render_schema.go` | 147 | Fragment-only consumer unchanged; M3 bundled pointers/names feed existing owner |
| `internal/openapiimport/tla/blocked_selection/README.md` | 78, 86 | M6 documentation/correspondence update |
| `internal/openapiimport/tla/security_projection/SecurityProjection.tla` | 3 | Fragment-only consumer unchanged; M3 bundled pointers/names feed existing owner |
| `internal/openapiimport/security_credentials_import_test.go` | 100 | Named regression/call-site update; retain existing behavior controls |
| `internal/openapiimport/schema_admission_test.go` | 94, 100, 103 | Named regression/call-site update; retain existing behavior controls |
| `internal/openapiimport/value_domain_test.go` | 156, 161, 165 | Named regression/call-site update; retain existing behavior controls |
| `internal/openapiimport/analyze.go` | 26, 32, 33, 41, 43, 47, 51, 54, 56, 59, 64, 69, 83, 109, 115, 119, 124, 132, 134, 139, 164, 165, 175, 180, 181, 196, 204, 273 | M1 Source seam; M2/M3 resolver integration; M5 identity projection |

## Parent rendered designs

Parent CLI built with `go build -o <BASE>/loom ./cmd/loom` from the archived parent.
Each fixture used `<BASE>/loom import openapi <repo>/internal/openapiimport/testdata/<fixture> -o <BASE>/out/<fixture>.go`. All exited 0.

| Fixture | SHA-256 |
|---|---|
| `supported.yaml` | `87c102eaf69f4f169113aee5a866a3d1aacf409856e4397c7f2081684975bae8` |
| `supported.json` | `87c102eaf69f4f169113aee5a866a3d1aacf409856e4397c7f2081684975bae8` |
| `symmetry.yaml` | `b5476129292027eca82bdc91f8a9f0e01a7a00aaa42bc6ba2e35fb4de6ab64b5` |
| `encoded_component_reference.yaml` | `49b862b18e3bf548dfa34aa1f8c14520d0f501272329a1a1a9c395541fce2679` |
| `shared_status_different_schemas.yaml` | `cf3951b9b615520b2da6a956803a5218b6ce44d9f803e9f4f8ef799295d02a70` |
| `object_field_names.yaml` | `cce77d70beb67136ed957d56cb3ad3189ea35ae25ab648f62e9be7a8199e99b2` |

- Parent `supported.yaml`: import, tidy, `loom gen example.com/gen957/design -o .`, tidy, `go build ./...` passed in `/var/folders/v7/x61rxqvx60q5w_p9jwgc9t_m0000gn/T/loom-957-gen-ag8pslb6/parent/supported.yaml`. Design and gen trees retained for comparison.

- Parent `symmetry.yaml`: import, tidy, `loom gen example.com/gen957/design -o .`, tidy, `go build ./...` passed in `/var/folders/v7/x61rxqvx60q5w_p9jwgc9t_m0000gn/T/loom-957-gen-ag8pslb6/parent/symmetry.yaml`. Design and gen trees retained for comparison.

Implementation uses managed checkout `/Users/luca/.codex/worktrees/external-references/loom`
at the same parent while #1024 main-push checks run. No commit is made there;
final diff is transferred to main for independent review and one atomic commit.

## M1 and M2 implementation evidence

- M1 source API build passed; the existing full importer/CLI suite started before
  resolver behavior was added. Generated sub-builds failed because the independent
  cache cleanup removed required archives. The complete existing suite is rerun
  with the isolated cache and `-skip ^TestResolver` while new resolver tests are
  intentionally red; analyzer behavior is still unchanged.
- M2 compile-safe stubs: `go test ./internal/openapiimport -run '^$'` passed.
- M2 direct regression command from the plan exited 1 against explicit resolver
  stubs. Each behavior failed with `description resolver not implemented`.
  The red log is `<CHANGE>/resolver-red.log`, not a compilation failure.
- URI and fragment tests passed after implementation (0.626s).
- Two confinement test builds failed at link time because shared Go build-cache
  archives disappeared. Process inspection found an independent Claude session
  executing `go clean -cache` (PID 93482), outside this task. No dependency defect
  is inferred. This task now uses `GOCACHE=/tmp/loom-957-build-cache.r6al1q` for
  build/test commands. The other session was not interrupted; no shared cache
  was cleared by this task. Affected checks are rerun in the isolated cache.

- M1 existing importer/CLI rerun did not complete cleanly: generated sub-builds
  observed compile-incomplete resolver edits while the suite was running, then
  the importer reached its default ten-minute timeout. The CLI package passed.
  This is not accepted evidence; rerun on stable source with an explicit bounded
  timeout after the integration milestone. Do not edit source during sub-builds.
- M2 grammar controls first failed because callback/encoding headers were not
  classified and a malformed `$ref` lost its type error. Those failures were
  corrected at the grammar/resolver owners. The document-class and mixed-context
  cases passed. `go test -race ./internal/openapiimport -run '^TestResolver'
  -count=1` passed (119.166s), including all identity/budget controls.
- M3 compile-safe bundle/Path Item stubs passed the empty-test command. Direct
  bundle and Path Item controls then failed with `description bundle not
  implemented`. After implementation they passed (0.832s): semantic transplant
  equivalence, original JSON/YAML numeric tokens and tags (15 cases), portable
  key order in both mapping orders, per-node provenance, per-occurrence blocked
  identities, and Path Item local/external/chained/overlap/cycle cases.
- Parent CLI on `testdata/external/openapi.yaml --report`: exit 2, nil document,
  external-reference only. Hand-bundled fixture imports with exit 0; design SHA256
  `bf0dbefbd655570dc0d08e80618f13146052b6bc55e7ee88f164488d8d688f60`.

- M3 integration first failed at the old external-reference preflight. With
  analysis routed through the bundle, the multi-file fixture renders byte-for-byte
  equal to its hand-bundled equivalent. The byte-only missing-reference test now
  returns a model with two blockers; `Render` refuses it. Direct integration
  controls passed (1.037s).
- All six unchanged-class fixture designs remain byte-identical to the parent
  after initial bundler integration. The built change CLI and outputs are under
  `<CHANGE>`. Repeat after final naming/projection changes.
- Whole normalized-model YAML/JSON parity now compares contracts separately
  from retained source provenance (which intentionally includes distinct token
  formats). Public contract fields are still compared in full.
- Naming inventory stubs compiled, then entry-stability, error-clone collision,
  and duplicate-name tests failed with the explicit unimplemented error. The
  allocator passed them after implementation (0.715s). Integrated name consumers,
  contract parity, selection renaming, missing refs and transplant controls passed
  (0.812s). The shared clone predicate consumes existing normalized failure plans,
  preserving the synthetic Content-Type header exclusion.

- Error-clone origin control failed for an entry operation referencing an external
  reusable response, then passed after checking the retained reference chain.
  All related allocator controls passed (0.765s).
- Shared Path Item duplicate-ID control failed with zero diagnostics before the
  analyzer check. It now diagnoses both selected occurrences, allows a selection
  containing only one, and partial import skips both duplicates. The synthetic
  Content-Type header exclusion also passed (0.666s). Combined M3 direct controls
  passed (0.714s).
- A frozen M3 source copy lives at `<CHANGE>/m3-source`, based on the parent plus
  the exact worktree source inventory SHA256
  `d08a544ec8a3124b61f84b6c5073ba221d8562c75b8823bd9bba4afb8471e05f`.
  The plan's focused generated-name/error checks run there with `-timeout=25m`
  and the isolated cache. Source freezing prevents later edits from invalidating
  generated sub-build evidence; no new test harness or exhaustive corpus is added.

- The frozen M3 generated-name/error regression selection passed (145.702s).
  This covers the plan's named shared-error, collision, generated compilation
  and selection-renaming cases under the recorded frozen source digest.
- M4 closure and CLI seams compiled. Blocked-selection controls failed on unknown
  Path Item filtering/skipped reporting and lost example dependency closure; the
  implementation now retains bundled grammar edges alongside normalized values,
  so a source-only admission failure cannot hide a blocker. Those controls passed
  (0.591s), including explicit lossy mode for the blocked example.
- CLI controls first failed on unknown mapping flags and missing unresolved-tag
  notes. After wiring Source through every mode, strict/selected/partial imports,
  report provenance, flag validation, routing, and existing import wrappers passed
  (0.911s). One new test's output file initially selected the temporary directory
  name as a Go package; its fixture now uses a `design/` directory.
- Source graph closure supplements normalized closure for rejected source-only
  constructs. It traverses only the resolver's grammar-classified bundled edges,
  excludes discriminator-mapping edges until #1005, and includes examples and
  security schemes. It does not scan arbitrary example/default/extension data.

## M5 identity and M6 generated contracts

- Frozen M5 suite input: `m5-source-manifest.json`, inventory SHA-256
  `4ec635b3573d6104d30539ac7ea6e975fb6cff5b76f9a6a3efbb5c6ffd863bcc`.
  Full importer/CLI tests exposed reusable-reference inventory being resolved
  twice, loss of manually assigned declaration names, and redundant wrapped
  planner diagnostics. Existing tests supplied the failing regressions.
- The allocator now uses the original reference graph on a copied Document;
  planning consumes resolved occurrences. Manual declarations retain supplied
  names. Typed admission diagnostics retain their individual paths and codes.
  Focused reusable, naming, comparison and blocker regressions passed:
  importer 11.339s, CLI 0.357s.
- Self projection tests passed (0.695s); URI security and existing per-scheme
  requirements passed (9.408s); discriminator edge recording passed (0.592s).
  Expanded URI containing-document/name/version matrix passed (0.564s).
- JSON entry bundling initially emitted flow-style YAML beginning with `{`,
  which libopenapi detected as JSON. `TestBundleJSONEntry` reproduced the parser
  error before the bundle root was changed to a block mapping. JSON, YAML
  transplant equivalence and grammar coverage passed (0.532s).
- `TestBundleNumericAnalysisParity` compares normalized exact bounds and
  diagnostic codes for inline and external values, including integer precision,
  huge/tiny exponents and YAML scalar tags. Passed (0.588s).
- Dynamic-reference rejection now includes `definitions` and `unevaluatedItems`,
  matching the grammar traversal. Both new cases failed before correction;
  `TestResolverDocumentClasses` passed afterward (0.661s).
- `LOOM_OPENAPI_CONTRACT=1 LOOM_EXTERNAL_SPEC_OUT=<SPEC_OUT> go test ./cmd/loom
  -run '^Test(OpenAPIImport(ExternalReferencesRoundTrip|ExporterSymmetryRoundTrip|SemanticRoundTrip)|ExternalImportProcessDeterminism)$'
  passed (19.988s). This includes local mapped imports, compilation, semantic
  comparison, three process-isolated source copies and permuted mapping order.
  The identity subtest also passed after adding an authored operation ID (9.857s).
- The existing symmetry corruption control referenced a schema-level example
  that does not exist in parent-generated output. The retained parent artifact
  confirms the actual example lives at the media object's
  `examples.inline.dataValue.id`; the control now corrupts that value and still
  requires semantic comparison to reject it.
- All six unchanged-class designs match the parent SHA-256 table byte for byte.
  `supported.yaml` and `symmetry.yaml` regenerated service trees compile and
  have empty recursive diffs for both `design/` and `gen/` against the parent.
  No differences are intended for these fixtures.

### Final source validation

- `make lint` passed, including root and integration/testdata source checks.
  The final refactor separates grammar dispatch, naming allocation and bundle
  operations into their owning methods; no lint suppression was introduced.
- `LOOM_DIR=<repo> make openapi-contract` passed on the managed worktree. Earlier
  archive-copy attempts lacked Git metadata needed by source discovery and are
  not accepted gate evidence. The worktree source stayed frozen during the run.
- A 30-layer shared-reference DAG exposed exponential revisits in the hop-budget
  traversal (the red run exceeded its five-second timeout). `checkHops` now
  memoizes only context-independent acyclic heights and checks cached heights
  against the current depth. All resolver controls passed after correction
  (7.986s); the model README records the unmodeled performance boundary.
- Rebuilding the final CLI after those refactors and importing all six unchanged
  fixtures produced the exact parent hashes in the table above. Existing generated
  compilation evidence remains applicable: their design inputs are byte-identical
  and this ticket does not change the generators.
- The hand-bundled control gained an authored operation tag during round-trip
  fixture preparation. Both parent and final CLIs were rerun against that same
  updated input; their designs are byte-identical. This is distinct from the
  earlier pre-tag fixture hash.
- Final-source semantic/external/identity round trips and process-isolated import
  determinism passed (73.582s), including generated service compilation.
- The frozen M6 full importer/CLI race suite passed (1730.976s / 237.516s).
  Final-refactor targeted race checks also passed (223.946s / 8.326s), covering
  grammar, naming, bundle, hop-budget and CLI changes made after that snapshot.
  Full `make test` is still pending. The M6 archive's source digest is
  `1540734a0f530f02bab98fea6bc8f6633abbe1b79438a9828a8fbc5e3a1d7f86`;
  the final-source controls above cover the later refactors. `make lint-docs`
  passed after the correspondence and validation-record updates.

### External lint prerequisite

Vacuum v0.36.2 passes `external_references.json` with zero errors. Its security
rule rejects the valid OpenAPI 3.2 URI key in `symmetry_identity.json`. Latest
upstream release v0.39.0 (release metadata checked 2026-11-06) has the same defect.
A standalone one-file document using `#/components/securitySchemes/ApiKey`
reproduces it without remote resolution; the name-keyed control passes.
Reported upstream as [Vacuum #1044](https://github.com/daveshanley/vacuum/issues/1044).
A later check of released v0.39.1 (2026-11-06, official darwin/arm64 archive
verified against the published SHA-256 manifest) reproduces the same error with
the unchanged repository contract rules: identity exits 1 with one error/one
warning, external references exits 0 with zero errors/five warnings. Its release
notes cover browser optimization and cross-file camel-case checking, not #1044.
The issue remains open. Probe logs are in
`/tmp/loom-968.GRZPUB/vacuum-v0391/*-contract.log`.

No dependency edits, rule suppression, schema rewriting or local workaround was
added. The plan's zero-error lint obligation for the identity specimen remains
open until a released fix or an explicitly revised acceptance rule addresses it.

## Independent review follow-up

Candidate diff `024f2829506ef338dbd7e92a97414b2fae400276db915bb6d7f605d2bbc212bb`
against f7a935768 received three medium findings: cyclic graph enumeration can
be factorial; ancestor resource detection confuses property dictionaries with
Schema Objects; FIFO opens can block before regular-file admission.

The full live `make test` completed successfully before applying review fixes.
The resource/FIFO red controls failed in 4.734s as expected. Extra keyword-named
property controls caught an ambiguous backwards grammar approach; forward grammar
classification replaced it. The final resource/FIFO/confinement race selection
passed (3.892s). Re-review caught malformed resource keywords on known schema
ancestors; the new red controls reproduced that case and the corrected grammar
classification passed the same race selection (review-resource-green3.log).
Focused independent re-review cleared the resource guard and FIFO findings.
These later fixes still need the affected final checks before commit.

The exact longest-simple-path hop rule has a policy question pending with the
user: bound dependency depth between recursive components, or retain the exact
rule with a separately bounded analysis-work budget. No alternative has been
selected or implemented. SCC upper-bound pruning with exact DFS fallback is
insufficient: a complete 12-node core plus 53 bidirectional leaves on one core
node is a 65-node SCC with longest simple path only 13, yet DFS still enumerates
factorially many paths. The reviewer supplied this counterexample before code
was changed. No claim of general polynomial work is made for the current code.


### Final affected checks after resource-loader review fixes

The frozen snapshot `/tmp/loom-957-review-final-shm_54bm` includes the reviewed
forward resource grammar and nonblocking FIFO admission fixes. Its manifest is
`/tmp/loom-957-review-final-manifest.json`. With Go 1.27.3, the task cache and
explicit `LOOM_DIR` selecting that snapshot, full affected package tests pass:
`internal/openapiimport` 516.617s and `cmd/loom` 86.480s. Scoped golangci-lint for
both packages reports zero issues. These checks complete the later-fix test
obligation recorded above; they do not resolve the hop-policy question or the
upstream Vacuum #1044 gate. The earlier full-suite and race evidence remains
applicable to unchanged paths.


### Delegated depth policy and bounded graph correction

The user delegated the remaining choice. The selected limit is 64 strongly
connected reference groups on a dependency path; Path Items retain a separate
64-object expansion limit and cycle rejection. ReferenceDepth.tla first exposes
the old simple-path work problem; its checked group-depth model passes all 512
three-vertex graphs. Both semantic negative controls fail as intended; the model
README records commands, bounds and counts. The 65-node cycle test fails against
the old Go code, then passes with iterative SCC and topological traversal. The
initial depth/budget/shared-DAG regression selection passes (1.573s). No exact
path enumeration or recursive graph walk remains in the budget checker.

Vacuum v0.39.2 was checked after its release (2026-11-06T14:31:21Z). The official
darwin/arm64 archive SHA-256 matches its published checksums:
`96da53d52e870b4396c1c7445ca02705ddd63c08aae9f493de60457dc7a59242`.
The same identity specimen still fails the unchanged contract rules. Upstream
#1044 remains open with the maintainer investigating; the release addresses YAML
overlay indentation. No rule waiver or dependency change was introduced.

Vacuum v0.39.3 was checked after its 2026-11-06T16:56:48Z release. Its official
darwin/arm64 archive matches GitHub's SHA-256 digest
`ff0b4e58b86767f311266ad2a41f311cd0c61ffb87461362c0b2e6f0d70febbf`.
The release closes #1044 and accepts fragment-only Security Requirement URIs,
but `symmetry_identity.json` still reports one security error; external references
still passes with zero errors. The implementation explicitly excludes all
document-prefixed URIs. A standalone 3.2 document with
`$self: https://example.test/security.yaml` and a declared ApiKey scheme proves
the remaining defect without network retrieval: `#/components/securitySchemes/ApiKey`
passes, while both `https://example.test/security.yaml#/components/securitySchemes/ApiKey`
and `./security.yaml#/components/securitySchemes/ApiKey` fail. A missing fragment
target correctly fails. This is reported separately as
[Vacuum #1053](https://github.com/daveshanley/vacuum/issues/1053), with the complete
reproduction and version evidence. The identity lint gate remains pending that
released correction. No rule waiver, dependency edit or schema rewrite is retained.
Probe inputs and results are under `/tmp/loom-957-depth/vacuum-v0393`.

The final affected race selection passes (63.200s): component boundaries,
dense cycles, the 512-graph independent oracle with renumbering/duplicate edges,
containment, existing document budgets/shared DAG and Path Item tests. An initial
root-containment test incorrectly expected 65 groups without a root-to-first-node
edge; correcting that fixture confirms both the 64-group path and the actual
65-group path. No production change was required. Scoped golangci-lint reports
zero issues; documentation lint and whitespace checks pass. Logs and TLC output
are in `/tmp/loom-957-depth` pending independent review.

Independent focused re-review cleared the depth finding with no new findings
for patch `77e46d683cf32370300777a8c013873fc831bca55251dbad900efb1d3987acf4`
and its 16-file manifest. It confirmed linear graph passes, containment and
separate Path Item semantics. This is not final commit authorization: new-main
integration and the external Vacuum gate remain. Task-owned TLC state directories
were removed after review; durable models, configurations and results remain.


### Source dialect and duplicate-key audit corrections

The fresh DSL-extension audit reproduced two source-contract losses: external
`jsonSchemaDialect` disappeared when targets were transplanted, and duplicate
explicit YAML mapping keys collapsed during canonicalization. The resolver now
retains the external document dialect and its grammar classifies schema-bearing
targets before admission. Such targets keep an occurrence-owned dialect blocker;
schema-free examples/security schemes and unrelated selected operations remain
usable. Duplicate explicit keys fail before canonicalization. Legal merge
precedence, local overrides, aliases and exact numeric tokens remain unchanged.

`/tmp/loom-fresh-audit/reference-source-fix/red.log` records both failures before
the correction. The focused regression passed in 0.586s, existing numeric/merge/
reference controls in 2.907s, and final regression in 0.611s. The source-dialect
model rejects both discarding context and blanket document rejection; the
correct target policy passes 60 generated / 40 distinct states, depth 2.
Independent review approved the exact 10-file patch SHA-256
`5f97fc461352e8c9009fc877c8f66c812a1c928fcdd6c7b8e49a5ad66bd6b99e`.
All ten reviewed file hashes matched before this documentation reconciliation.
No production code changed during that reconciliation; prior test evidence is
reused. Checker-state directories were removed after review. Durable sources and
instructions live in `internal/openapiimport/tla/source_contracts`.

### Final handoff boundaries

- The full #957 candidate includes the resolver, CLI, naming/selection changes,
  source-contract/depth/FIFO corrections, tests, fixtures, models and documentation.
  The staged `pulse/rmap/rmap_ttl_miniredis_test.go` is already-main prerequisite
  history and is excluded from the ticket patch.
- Do not replay the entire suite solely for documentation edits. After integration,
  rerun checks invalidated by merged source changes and obtain final independent
  review against the actual parent.
- Vacuum v0.39.3 is the current composed parent's pinned contract tool. Historical
  v0.36.2/v0.39.2 and same-document URI observations remain recorded above.
  The actual external Security Requirement URI specimen still fails under
  v0.39.3 ([upstream #1056](https://github.com/daveshanley/vacuum/issues/1056));
  #1053's same-document correction does not satisfy that case. No suppression,
  schema rewrite or dependency edit is retained. The zero-error obligation stays open.
- Keep the plan and its inbound link until delivery finalization. Update the
  readiness index, remove the plan, and finalize the design status before final
  commit review. No commit or push has been performed by this readiness pass.


### Readiness reconciliation on final source

A fresh Go 1.27.3 CLI build with
`GOCACHE=/tmp/loom-957-build-cache.r6al1q go build -o /tmp/loom-957-readiness/loom ./cmd/loom`
passed. The bounded import comparison was repeated on this source, including all
reviewed loader/depth/source-contract corrections. Every row below has identical
parent and current bytes (one shared SHA-256 is shown). Output directories both
use the package name `out`; an initial differently named directory changed only
the authored package line and was discarded before this comparison.

| Input | Parent and final SHA-256 | Intended differences |
|---|---|---|
| `supported.yaml` | `87c102eaf69f4f169113aee5a866a3d1aacf409856e4397c7f2081684975bae8` | 0 |
| `supported.json` | `87c102eaf69f4f169113aee5a866a3d1aacf409856e4397c7f2081684975bae8` | 0 |
| `symmetry.yaml` | `b5476129292027eca82bdc91f8a9f0e01a7a00aaa42bc6ba2e35fb4de6ab64b5` | 0 |
| `encoded_component_reference.yaml` | `49b862b18e3bf548dfa34aa1f8c14520d0f501272329a1a1a9c395541fce2679` | 0 |
| `shared_status_different_schemas.yaml` | `cf3951b9b615520b2da6a956803a5218b6ce44d9f803e9f4f8ef799295d02a70` | 0 |
| `object_field_names.yaml` | `cce77d70beb67136ed957d56cb3ad3189ea35ae25ab648f62e9be7a8199e99b2` | 0 |
| `external/bundled.yaml` | `09d8c6a6745f6fa31100e1de83edbdfbe0bdd5f1569acb30e928af59e76f5497` | 0 |

The new descriptions' `--report` outcomes are:

| Input | Parent | Final source |
|---|---|---|
| `external/openapi.yaml` | exit 2; 0/0 operations, external-reference blocker for `paths/pets.yaml` | exit 0; 1/1 operations and 1/1 schemas, with component and Path Item provenance |
| `external/symmetry_identity.yaml` | exit 2; 0/1 operations, document-identity omission and security-requirement blocker | exit 0; 1/1 operations, retained source identity and security scheme provenance |

Final commands were `/tmp/loom-957-readiness/loom import openapi
internal/openapiimport/testdata/external/openapi.yaml --report` and the same
command for `symmetry_identity.yaml` with
`--document https://auth.test/security.yaml=internal/openapiimport/testdata/external/remote/security.yaml`.
Parent reports remain in `<BASE>/{external-openapi,identity-parent}.log`; final
reports and all seven output hashes are under `/tmp/loom-957-readiness`.
The supported/symmetry design and generated-tree compile comparisons recorded
above remain valid: the final imported designs are byte-identical and no
framework generator changed in #957. `make lint-docs` and `git diff --check`
passed after the documentation reconciliation. No broad suite was repeated.

### Composed integration against #1014

The scratch candidate at `/tmp/loom-957-integration-prep/revision-r3/source`
uses actual parent `4d5daba25d622d881a9ff1cdce35602880b5d0d4`, tree
`13cf2edc1de14e3ad18ac46fc1d4de4c1d4d5b52`. The 138-path prior composition
replayed 136 files and merged only two additive framework skill/model-index paths.
All newer singleton/nullable-pair/form presence corrections, current dependencies
and unrelated parent files were retained. Source inventories, exact composition,
output comparisons and terminal logs are in that scratch packet.

The composed resolver/provenance/naming/source/depth controls passed 2.340s. The
existing CLI external, identity/exporter, supported semantic, parameter-style and
composed oneOf round trips passed 44.418s, including generated module compilation,
libopenapi analysis and contract comparison. All seven unchanged-class imported
DSL controls are byte-identical to this actual parent. The two external reports
retain parent exit 2 versus candidate exit 0, with source identity and provenance.
The supported/symmetry imported designs and complete generated trees are identical;
all four parent/candidate modules build and separate candidate generation processes
produce identical bytes. An initial comparison used different CLI basenames and
therefore differed only in generated command comments; rerunning both as `loom`
established exact equality without normalizing output.

The current lint policy identified two maintained resolver functions below its
maintainability threshold. Their private responsibilities were decomposed without
changing policy or operation order: entry-name precedence, load-before-alias
resolution, typed OpenAPI target locations, source dialect and schema-resource
admission. No suppression or threshold change was used. Direct checks passed 7.127s
and scoped lint reports zero issues. The final affected race selection passed
77.509s for the importer and 1.565s for the CLI. Nine imported designs (the seven controls
plus external and identity entries) and both complete generated control trees are
byte-identical before/after that decomposition. External import determinism passed
6.537s across three relocated source trees and permuted document mappings. Existing
formal models and assumptions remain unchanged; no proof execution entered a gate.

The existing external inline/reference/nested oneOf origin fixture was also run
through the maintained `requireRenderedDesignGenerates` helper in a temporary test
overlay. All three generated modules passed 20.572s while retaining the original
source-origin assertions. The overlay uses canonical source paths; an initial
`/tmp` spelling did not match Go's `/private/tmp` package path and is not counted as
generated acceptance. Its corrected input and terminal log remain in the packet.

The exact current contract rules with released Vacuum v0.39.3 pass
`external_references.json` with zero errors. `symmetry_identity.json` still fails
with one `oas3-operation-security-defined` error for the authored external Security
Requirement URI. This is the remaining upstream #1056 gate, not a reason to alter
the URI, omit its source identity, relax the rules or modify the dependency.
Historical #1044/#1053 evidence remains above; the accepted zero-error gate remains
open. Final whole-ticket review, outstanding delivery gates and the atomic #957
commit/push are not claimed by these scratch checks. Keep this plan active.

### Focused whole-review corrections after composition

Scratch revisions 4 and 5 preserve the full contract and supersede only their
identified implementation gaps. Revision 5 targets amended actual #1014 parent
`c86613180fbba9eb232f9969f95d18a9427ebe72` (tree
`dd404ccecb18b8dc1217b3f9a960ce6ebd59a23b`). The amendment only refreshes two
JSON-RPC generated CLI fixtures and their validation note; its exact delta is
preserved in this scratch composition.

Operations retain authored source addresses, including Additional Operations,
through security/selection and planning. Explicit public overrides remove only
irrelevant inherited security failures; unresolved required protection still
blocks strict and partial import. The shared grammar index admits actual local
and external Schema positions, never dictionaries, operations or literal data.
Local non-component Schema targets now use the same canonical promotion as
external ones. Ordinary references and mapping outcomes share the allocated
component while inline/promoted occurrences retain authored identity. Introduced
local declarations follow the complete authored naming inventory; a source in the
entry document does not make its newly introduced component authored. URI mappings
do not gain implicit component-name keys. Component-root controls stay unchanged.

Public diagnostics attach full occurrence origins after analysis, selection and
planning, then project detached safe fields. Pointer-token, typed-URI and prose
boundaries prevent redaction from consuming unrelated path segments or emails.
Valid and malformed credentials remain suppressed across structured fields,
strict/report output, duplicate-key errors and CLI argument wrappers. The bounded
TargetPromotion and PublicProjection models reproduce the old local-pointer and
early-origin policies before checking corrected invariants; normal Go tests cover
actual parsers, tokenization, recursion, declaration naming and output contracts.
Models stay explicit opt-in design evidence, outside all delivery gates.

Revision 5's retained packet records direct red/green controls, affected race
(importer 68.884s, CLI 1.496s), nine exact r4/candidate/independent imported designs,
compiled local property/alternative/response/recursive and naming-collision
contracts, separate-process generated bytes, and external/identity/exporter
round trips (26.414s). The two rendered existing external specs remain byte-identical
to revision 4. Scoped lint reports zero issues. Initial fixture indentation and
missing scratch artifact-directory failures remain separate historical evidence;
final retries supersede them. Exact sources, commands, terminal logs and intended
differences are in `/tmp/loom-957-integration-prep/revision-r5/acceptance.md`.

These corrections still require focused independent re-review and final atomic
integration gates. Vacuum v0.39.3 still fails the unchanged external security URI
identity specimen at upstream #1056; no rule waiver, URI rewrite or dependency
change is accepted. This record does not claim delivery or close that gate.

### Revision 6: typed public display and source admission

The revision-5 focused review confirmed local target promotion and final diagnostic
origins. Its remaining low display finding was reproduced with a quoted format
value beginning `https://`, plus separate real CLI duplicate-property and pointer
cases supplied by the parent. All three nonsecret witnesses were erased by prefix
inference. Revision 6 keeps generic quoted text and pointer tokens prose-bounded;
known authored URI metadata supplies whole-value context without changing raw
identity. Source mapping errors project the URI before formatting and preserve
flag/refusal context without the standard flag wrapper repeating raw input.

Duplicate keys require structural roles before error formatting. The source parser
first validates aliases, then uses the existing grammar walker to identify exact
security key nodes, including alias and merge sources. The merge case had its own
retained red before the shared YAML/grammar correction. Generic property keys keep
their text. Entry syntax and supported-version admission now run before external
roots are opened and before libopenapi formats errors; the parsed entry is reused
when binding its retrieval identity. Existing source identity, numeric precision,
alias/merge, library/model error, version and selected/partial semantics remain
covered by bounded affected tests.

Terminal evidence in `/tmp/loom-957-integration-prep/revision-r6/acceptance.md`:
owner checks pass importer0.745s/CLI0.665s; affected direct checks3.096s/0.479s;
expanded final race33.573s/1.904s; scoped lint zero issues. The ordinary supported
import is byte-identical across revision5/revision6/independent processes, so
unchanged generated compilation/spec evidence is reused. Four real CLI flag
controls retain unknown/missing/duplicate refusal and malformed-space privacy.
The projection model's accurate final counterexample captures ordinary-text loss
when structural type is discarded; typed publication passes72 distinct states.
String parsing and external-load rejection remain concrete Go-test obligations,
not claims of the bounded proof. An earlier raw-bypass model attempt is retained
as rejected correspondence, not evidence of the actual old heuristic.

Revision 6 is scratch correction evidence against the same amended #1014 parent
c86613180fbba9eb232f9969f95d18a9427ebe72. It awaits the original reviewer's focused
re-review, final integration and the unchanged external Vacuum #1056 gate. No
production generator/runtime/dependency change, waiver, commit or delivery is
claimed by this record.

### Revision 7: complete merge roles and safe mapping refusal

The revision-6 re-review confirmed the ordinary quoted prose/property/pointer
corrections, then reproduced two remaining privacy leaks. A malformed userinfo
value containing both whitespace and `=` defeated sanitization of the first
composite segment. `ParseDocumentMapping` now returns the existing safe reason
without echoing any rejected `URI=PATH` argument. CLI flag context and error
wrapping remain; successful literal paths retain spaces, `@` and `=`.

Grammar node fields already consumed effective YAML values, but map/component
collection traversal skipped inherited members. `orderedYAMLMapping` now supplies
both those collections and canonicalization from the existing merge owner.
Explicit members retain source order, inherited members are sorted, first merge
source wins, explicit whole values override inherited subtrees, and original raw
value nodes remain available for URI-key classification. No new merge policy or
string heuristic is introduced. The red matrix fails six inherited collection
cases before the fix; all 20 node/map/component/array and merge combinations now
pass. Separate override and exact canonical-byte tests preserve selection/order.

Terminal focused evidence lives in
`/tmp/loom-957-integration-prep/revision-r7/acceptance.md`: corrected owner tests
pass importer0.660s/CLI0.572s; affected race passes41.742s/1.877s; scoped lint
reports zero issues after one mechanical blank-line correction. Actual CLI
mapping/merge failures retain refusal but no credentials; all three ordinary-text
controls remain intact. Supported import output is exact revision6/revision7/
independent-process bytes. Prior unchanged generated compilation and OpenAPI
specimen evidence remains applicable; no broad suite or generated harness was
added. Model old collection/mapping policies violate privacy, and the corrected
finite policy passes288 distinct states with an explicit safe-reason invariant.
Concrete Go tests, not the model, establish parser and raw-node behavior.

This is scratch acceptance against c86613180fbba9eb232f9969f95d18a9427ebe72,
awaiting original-reviewer re-review and final atomic integration. The external
Vacuum v0.39.3 security URI gate #1056 remains unsatisfied, with no workaround,
waiver or dependency change. No delivery or commit authorization is claimed.

## Explicit external resources for contract validation

The generated `symmetry_identity.json` preserves an external Security Requirement
URI. Its import and re-import checks already supply `remote/security.yaml` under
`https://auth.test/security.yaml`; standalone lint of the generated entry alone
omits that validation input. Historical standalone failures above remain recorded,
but they do not establish behavior with a complete supplied description.

The existing OpenAPI `contractVacuum` test owner now accepts explicit retrieval
URI/file pairs through a private public-motor adapter. It uses the same
`vacuumVersion` and `testdata/vacuum.yaml` as ordinary CLI checks. No Vacuum
production dependency, synthetic component alias, modified specimen, suppressed
rule or separate contributor linter command is introduced. The adapter validates
supplied JSON/YAML syntax, rejects duplicate/relative/fragment identities and
unreadable resources, denies unknown retrievals, and verifies exact admitted
identity and parsed contents at admission into each actual index-build owner.
After rule execution it verifies membership in both owners and full unresolved
contents; the resolved graph may contain normal reference expansions. Admission
records are associated with the exact returned index and its owning rolodex,
not inferred from equal content. Every motor error and error-severity finding
fails the check.

The green backend controls run in the existing `make openapi-contract` target.
Run only those focused backend/CLI parity and input controls with:

```bash
LOOM_OPENAPI_CONTRACT=1 go test ./http/codegen/openapi/v3 \
  -run '^TestContractVacuumResourceBackend$' -count=1
```

The cases reuse the existing non-trivial OpenAPI3.2 specimen and five invalid
mutations. They compare CLI/motor success and error rule IDs/source locations;
exercise JSON and YAML admission, an actually used resource with a resolvable
internal reference, malformed/duplicate/missing resources, unknown
reference denial and motor errors; and retain absent-target/missing-scheme and
URI-like local-name controls. The stronger local-name case supplies a conflicting
non-Security-Scheme document at the relative URI, so a future URI-first resolver
would fail instead of selecting the valid local scheme. On v0.39.3, which rejects
external URI interpretation generally, negative URI controls alone do not prove
correct resolution. Actual index-content assertions prove resource admission.

After the existing CLI roundtrip copies the unchanged generated specimen to
`SPEC_OUT`, evaluate the complete-input publication prerequisite with:

```bash
LOOM_OPENAPI_CONTRACT=1 \
LOOM_EXTERNAL_SECURITY_SPEC="$SPEC_OUT/symmetry_identity.json" \
LOOM_EXTERNAL_SECURITY_DOCUMENT="$PWD/internal/openapiimport/testdata/external/remote/security.yaml" \
go test ./http/codegen/openapi/v3 \
  -run '^TestExternalSecurityDocumentsPassVacuum$' -count=1
```

This selector requires zero errors; it does not turn the known upstream failure
into an expected-success test. Released v0.39.3 still rejects the loaded target at
`oas3-operation-security-defined`, as tracked by
[Vacuum #1056](https://github.com/daveshanley/vacuum/issues/1056).

The additional unresolved-reference observation from the alias investigation is
now independently reproduced as a released-library defect:
[libopenapi #699](https://github.com/pb33f/libopenapi/issues/699) includes the
self-contained two-case reproduction against v0.42.3. It uses a direct external
response-schema reference, without a component alias or the validation adapter.
Without `$self`, the correct external host is requested and model construction
succeeds. Adding only `$self` rewrites that absolute reference to the entry host
and causes a missing-component error. This confirms the library trigger; it does
not claim the reconstructed broad alias specimen was re-executed.

The focused adapter review findings are resolved. The selected released
[libopenapi v0.42.4](https://github.com/pb33f/libopenapi/releases/tag/v0.42.4)
includes the correction for #699. The unchanged standalone two-case reproduction
now passes: both inputs request the original external host and build the model.
The validation CLI and resource adapter use this same selected library version,
verified from each binary's build information.

At this historical v0.39.3 checkpoint, publication still depended on
[Vacuum #1056](https://github.com/daveshanley/vacuum/issues/1056).
With Vacuum v0.39.3 and libopenapi v0.42.4, the unchanged complete-input check above
still reports an error from `oas3-operation-security-defined`, despite both
resolved and unresolved resource indices retaining the exact supplied security
document. The 27 resource-admission, invalid-input and name controls pass in both
pending delivery sources; they do not waive the failing positive check. No
adapter alias, BaseURL override, URI rewrite or other workaround is introduced.
The recorded next step was to rerun the positive check and retained controls
after a Vacuum correction. The delivery hold remained at that checkpoint; no draft
PR or unreleased source had been consumed. The qualified correctness pin and
positive result in the current composition checkpoint above supersede this hold.
