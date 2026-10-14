# Schema semantic views (#924)

Base: published #922 effective-index dependency plus #925 deterministic extension
rendering. The dependency exposes authored proxy nodes and an immutable effective
YAML view as separate concerns. Three importer decisions currently inspect raw
proxy nodes: unknown schema keywords, null-only alternatives and boolean schemas.
A merge marker can consequently be rejected as a schema keyword; an alias can
hide the shape that those decisions require.

1. [x] Preserve failing mapping/alias semantic-consumer cases before production edits.
   Cover exact explicit/aliased/merged DSL, inherited unknown-keyword rejection,
   local/first-source precedence, nullable alternatives and boolean scalar aliases.
2. [x] Independently review this bounded direction and acceptance. Reuse #922's model
   and shared immutable owner; this adds no traversal state or public policy.
3. [x] Add one private semantic-node accessor that resolves the proxy's authored node
   through `utils.ResolveMergeNodes`, including scalar aliases. Route all three
   consumers through it. Keep raw proxy APIs, source graph and location nodes.
4. [x] Run direct races and relevant grammar/admission/composition controls. Verify
   source identity and unchanged YAML, malformed cycle rejection, independent
   process exact import→generation bytes and generated compilation. Preserve
   explicit controls and list intended differences; avoid a full corpus sweep.
5. [ ] Independent exact-diff review, one atomic commit, publication and issue closure.

Contract: explicit and effectively equivalent YAML have identical schema
semantics. Unsupported inherited keywords and constrained/annotated null branches
remain rejected. True/false boolean schema policy stays as currently documented;
this repair makes aliases obey it too. Mappings may merge; scalars may alias.
The accessor must not recursively flatten or rewrite authored source, implement
merge precedence again, or redefine the dependency's source-node accessor.

## Acceptance record

The failing tests were preserved before the implementation. Independent readiness
review approved the single shared accessor; the final source review remains the
commit gate. Production changes are limited to the accessor and its three callers.

Go 1.27.2 with `GOEXPERIMENT=jsonv2` and the published libopenapi pin
`v0.41.7-0.20261014145029-7886f0170cc4` passed:

- Direct semantic-view races (1.867 seconds), then relevant schema admission,
  grammar, reference-sibling, numeric-source and component-index races
  (3.203 seconds). Source identity, unchanged YAML, local/first-source precedence,
  boolean aliases, null alternatives, cycles and fatal inherited diagnostics are
  explicit controls in `schema_semantic_view_test.go`.
- Focused importer lint: zero issues.
- Four separate import/generation processes: the published
  `f4c12945d0faf19ee4d8cd377c671a7e57003196` explicit baseline, candidate explicit,
  composed form and composed repeat. All 17 DSL/generated files match byte for
  byte. The specimen is a response object with integer bounds 5–10, a nullable
  string and a true-schema property; composed form inherits the integer mapping,
  merges the null-only branch and aliases the boolean scalar. Identical extension
  anchors appear in both source documents. Baseline and composed apps compile.
  Generation uses the published Loom pin with no module replacements.
- Rendered JSON and YAML both build with libopenapi and pass pinned Vacuum
  v0.33.0 contract rules: zero errors, six ordinary missing-description/example
  warnings and two missing-type informational results for permissive schemas.
  The rendered bytes are identical to the baseline, so these are unchanged.

Intended differences: accepted equivalent YAML now reaches the same DSL and wire
artifacts as explicit schemas; aliased false schemas receive the same fatal
rejection as explicit false schemas. No generated output changes for explicit
controls. Existing diagnostic classification remains unchanged.

This adds no traversal, cycle or precedence implementation. The immutable view
owner and its model remain in the pinned dependency and
`internal/openapiimport/tla/yaml_index_owner`; the new accessor only selects that
view for semantic inspection. Raw proxy nodes still own source locations.
