# Retained OpenAPI examples

This record documents the implementation and validation delivered by the atomic
#572 commit containing it. The full-catalog audit is not a ticket acceptance gate.
The accepted contract is in the
[value-contract design](../../roadmap/value-contract-design.md).

## Ownership and compatibility

`expr.ValueContext` selects and resolves each occurrence's examples once.
`http/codegen/internal/representation` binds retained results to captured schema
and example plans. The OpenAPI IR consumes those bindings without independently
sampling values or selecting a union branch. Structural schema authority and
example positions remain separate, including at recursive reference cuts.

Automatic request-body, response, parameter and header reuse compares complete
definitions, including examples. Distinct retained examples can leave equivalent
definitions inline. This is the accepted #572 compatibility exception; the reuse
algorithm itself is unchanged. Explicitly authored component names, schema and
Go names, operation IDs, constraints and runtime behavior remain protected.
Schema equivalence alone does not establish generated-client compatibility.

Security parameters omitted from ordinary OpenAPI parameters are also omitted
from example preparation. Both paths use `transportir.Security.IsParameter`.
Ordinary parameter binding failures remain errors; no independent source is
manufactured for a synthetic security credential.

## Comparison inputs

- Parent: `95cfbbff3bcaeebec3037b5ee996d9e89f084789`.
- Parent source content ID:
  `b613254468a747bb948893206b5eb0118c6a33cde963e590e56d4ff66e041851`.
- Final candidate source content ID (4,448 paths):
  `5cdcb60abad0de9973871266017e62b94f2fff2b00c50d799a318e749af1c0f0`.
- Earlier diagnostic candidate source content ID:
  `f3101ca471f7fb4b804f1bfc6c19b96da80a23347213bd77df705dae1d4b845b`.
- This inventory contains 4,445 paths. Its initial code delta from the
  subsequently linted candidate is moving the unchanged
  `TestRenderedSpecDeduplicatesGeneratedRequestBodiesAndUnionEnvelopes` function
  from `files_test.go` to `schema_dedup_render_test.go`. The function is byte-equal;
  the move adds no generator or design input change. Later evidence-document
  updates are likewise outside generation inputs. The consumer comparison then
  found a missing guard for singleton explicitly named examples. This snapshot
  is diagnostic evidence, not the final source for the full-catalog comparison.

The final snapshot adds the singleton guard, importer producer correction and
their regressions, alongside test layout and documentation changes. The earlier
three-transport and SSE checks remain applicable: their inputs contain no named
example-component metadata and do not invoke OpenAPI import. The final catalog
comparison uses the final snapshot's specimen inputs for both parent and
candidate, including the expanded singleton-example fixture. Later changes are
documentation/policy updates and removal of one duplicate inherited-example
test. The retained source-group and alias-owner tests pass. None of these edits
changes generator inputs or invalidates the recorded output evidence.

## Classified fixture changes

All 88 changed golden files were checked against the parent, independently for
JSON and YAML. They represent 44 fixtures. Changed paths were checked as OpenAPI
annotations, rather than assuming every object key named `example` or `examples`
is an annotation. None of these fixtures has a literal schema property with
either name.

These 39 fixtures change only example annotations:

`activity-feed`, `array`, `async-session-security`, `body-inline-object`,
`body-object-required`, `body-object`, `collab-streams`, `endpoint`,
`error-examples`, `explicit-body-result-object-views`,
`explicit-body-result-object`, `explicit-body-result-type`, `explicit-view`,
`fingerprint-collisions`, `headers`, `mapped-names`, `meal-planner`,
`multiple-services`, `multiple-views`, `not-generate-attribute`,
`not-generate-host`, `not-generate-server`, `path-with-multiple-explicit-wildcards`,
`path-with-multiple-wildcards`, `path-with-wildcards`, `problem-links-async`,
`raw-request-bodies`, `recursive-named-array`, `request-response-split`,
`reusable-components`, `scalar-map-keys`, `skip-response-body-encode-decode`,
`streaming-partial-examples`, `typename`, `valid`, `with-any`, `with-map`,
`with-spaces`, `with-tags`.

Five fixtures also change automatic reuse:

| Fixture | Automatic definition replaced by inline definitions |
| --- | --- |
| `explicit-reusable-component-names` | Response `ComponentNamesSearchGadgetsStatus200Response`; explicitly authored names remain intact |
| `parameter-components` | Response `ComponentServiceListGadgetsStatus200Response` |
| `inline-body-selection` | Parameter `QueryQ` and header `HHeader` |
| `inline-body-shared-method-name` | Parameter `QueryQ` and request body `FirstDoRequestBody` |
| `schema-dedup` | Request body `DedupServiceUnionFirstRequestBody`; the two operations retain different union-branch examples |

For these five fixtures, resolving the former automatic references leaves equal
non-example content. Schema names, union envelope references and discriminators,
constraints and operation structure remain intact. This statement covers these
fixtures; the full-catalog and consumer results must be assessed separately.

## Verification evidence

| Obligation | Current evidence |
| --- | --- |
| Source selection, ownership and planned declared observations | Complete expr, representation and OpenAPI IR suites and affected vet checks pass |
| Example validity | Rendered JSON/YAML matrix passes for OpenAPI 3.1 and 3.2, including bytes, explicit null, exact numeric values and a rejected mutated union discriminator |
| Output acceptance and determinism | Full OpenAPI v3 suite passes with `LOOM_OPENAPI_CONTRACT=1`; independent-process byte equality remains enforced, with updated expected hashes |
| Integration findings | Focused async, selected-body, body-identity, mapped-reference and session-security checks pass; the interrupted HTTP diagnostic remainder passes |
| Lint | Final full `make lint` passes after the importer and test-helper mechanical fixes; root and HTTP/JSON-RPC integration modules report zero issues |
| Three transport probes | Parent and candidate pass two-process byte equality, runtime, race, test, build, vet, schema assertions, libopenapi and Redocly; only OpenAPI example annotations differ |
| Checked-in HTTP and JSON-RPC SSE fixtures | Both revisions regenerate, test and vet successfully; JSON-RPC generated output is identical, and HTTP differs only in OpenAPI example annotations |
| Repository test batch | All packages except `internal/openapiimport` passed the complete batch; after repair, the full importer suite and vet pass, as do the affected IR/v3 suites and post-lint focused regressions |
| Importer output comparison | The two original shared-error designs generate and compile on parent and final candidate; exact comparisons retain the canonical OpenAPI output and classify the intended service/HTTP error-type changes |
| Optional full-catalog audit | Stopped by the revised validation policy after 656 of 934 cases (shards 00–40) passed both revisions and repeated generations; completed archives and rehydration checks are preserved. Shard 41 was interrupted and is excluded. The remaining 278 cases are unverified by this audit, not waived failures or a ticket blocker. |
| AutoK and drum-meoh generated clients | Final snapshot recheck passes: all eight AutoK authored examples and references match the parent; Drum retains only the assessed automatic export change; both applications typecheck |
| Independent exact-diff review and atomic delivery | Exact-diff review found no production, proof, test or generated-output defects. The carrier comments and delivery-status wording were corrected and re-reviewed before the atomic #572 commit containing this record. |

The stopped audit inspected 10,859 generated files per revision/run across its
656 completed cases. All 524 changed files were OpenAPI JSON/YAML: 243 designs
changed only example annotations, and 19 also changed automatic component reuse.
Resolving those references preserved all non-example definitions. Both isolated
runs matched byte for byte on each revision; no non-OpenAPI difference or
unclassified difference was found. The uncovered remainder is 271 HTTP designs
and seven JSON-RPC designs. These are limits of the optional audit, not a claim
of full coverage. Ticket acceptance relies on the focused obligations above,
including separate HTTP, gRPC and JSON-RPC probes and both SSE fixtures.

## Consumer compatibility assessment

The diagnostic comparison used AutoK commit
`a208490d6a60522fd3752bc808427e23a2d13ee8` and drum-meoh commit
`d286f49f23c7f3b7a4656807798cade2cabe9704`, with their actual pinned HeyAPI
generation paths and isolated copies. Both source repositories remained clean.
All non-OpenAPI generated Go files were byte-identical between revisions, and
`oasdiff breaking` reported no breaking changes in either contract.

AutoK's generated TypeScript was byte-identical except for the recorded OpenAPI
hash. Its earlier diagnostic contract nevertheless lost four explicitly authored
example components: `CreateGraphNodesLinkExample`, `GraphBulkPatchNodesExample`,
`ProductGitHubLinkRepositoryExample`, and
`ProductGitHubSelectedTasksExportExample`. This violates the accepted contract
and is repaired by retaining the structured example path for explicitly named
singletons. Direct IR and OpenAPI 3.1/3.2 JSON/YAML regressions pass. The final
consumer recheck restores all eight authored example components byte for byte,
and the full example-reference path/value map matches the parent. Removing
example annotations leaves identical full contracts; operation IDs, constraints
and requiredness remain intact. Passing client typechecks alone would not have
established preservation of the named examples.

Drum-meoh's raw contract retains distinct occurrence examples. Its existing
normalizer removes examples before client generation. Four automatic parameter
components disappear and one automatic ETag header component changes name.
The client consequently loses aliases `PathAudioTrackId`, `PathSongId`,
`PathStem`, and `QueryArtifactId`, plus their corresponding `z`-prefixed Zod
validators and barrel exports. No non-generated frontend source uses any of
these eight names. SDK functions, operation-specific types and validators,
TanStack Query output, and client/runtime files are byte-identical; both full
application typechecks pass.

The final snapshot's Drum raw and normalized OpenAPI files are byte-identical
to its earlier assessed candidate outputs, as are all generated TypeScript
files. The final AutoK and Drum checks reused the same isolated consumer inputs
and pinned generation tools; both source repositories remained unchanged.

These results support compatibility for the two inspected applications. The
automatic export removal can break other consumers that import those names;
it is a generated-client export change, not merely an example annotation
change. It follows the accepted complete-definition reuse rule and does not
justify discarding occurrence examples or introducing schema-only reuse.

## Importer producer correction

The repository batch's two shared-error failures had one cause: the importer
emitted an `Error` declaration containing only `Extend`, which selected the
built-in problem type as its service source. HTTP body derivation separately
merged the imported object's members, causing the strict source/target plan to
reject `message` as unrelated. The correction emits explicit object `Type`
clones for the distinct error identities, extends the imported type there, and
references each clone from `Error`. The canonical `OpenAPIBody` reference stays
intact. Clone allocation reserves imported type and top-level identifier names.

Focused tests generate and compile the resulting programs and assert exact
evaluated error fields. Cases include shared errors, identical status codes in
different operations, an inherited object with no local fields, and collisions
in both naming namespaces. No generic DSL defaults, occurrence capture rules or
ancestry guards change.

The paired parent/candidate output comparison covers both original failing
importer cases. Each changes its rendered design and seven generated files:
service error types and client signatures, HTTP client/server types and codecs,
and the design fingerprint. The generated types now contain the imported
`message` field instead of the accidental problem-document fields, and clients
no longer require those absent problem fields. OpenAPI output and canonical
`OpenAPIBody` declarations are byte-identical. Both revisions generate and
compile successfully. This is an intended correction to newly imported designs,
not a claim that their generated Go API is unchanged.

The proof correspondence and its limits are recorded in
[the ledger](../../expr/lean/value_projection/correspondence.md). The 644 main
theorem audits, 29 collision-diagnostic claims and fresh kernel replay passed.
The original proof wrapper timed out in its final negative-control group; that
group passed separately with a thirty-minute timeout, which is now reflected in
the Makefile. These checks do not prove Go extraction, component allocation or
downstream-client compatibility.
