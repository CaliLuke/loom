# Per-scheme required scope projection (#752)

Parent: `05567e2b08f99f3bb5aaa5ce5b8303bc94795b9a`. Delivered as `24a277291`, pushed and confirmed closed after independent final and integration reviews.

The shared contract already exists: `security/*Scheme.RequiredScopes` must be a
subset of that scheme's declared `Scopes`. Expression validation admits a scope
when any scheme in the AND requirement declares it. Before this repair, generated endpoint callbacks
and OpenAPI copied the whole requirement into each scheme, contradicting
that contract. This makes an unscoped key fail validation alongside a scoped JWT.
Application callbacks continue to decide credential validity and granted scopes.

## Sequence

- [x] Reproduce endpoint and OpenAPI projection failures, preserving union admission
  and repeated-scheme OR controls.
- [x] Independently review this owner proposal and bounded acceptance.
- [x] Add one expression-owned requirement-to-scheme intersection. Preserve authored
  required-scope order and return independently owned slices. Do not change AND/OR
  alternatives, auth order, session expansion or unknown-scope rejection.
- [x] Record projected RequiredScopes on each requirement's fresh SchemeData; make
  endpoint callbacks and OpenAPI IR consume the expression decision. Preserve the
  projection in SchemeData.Dup. Global scheme deduplication supplies transport and
  constructor declarations only; it must not own an OR branch's required scopes.
- [x] Delete unused internal/securityreq: repository-wide reference search found
  only its own package declaration, including non-Go files. Its obsolete session
  expansion and requirement-wide projection are superseded by current owners.
- [x] Run bounded generated/runtime/output checks and document behavior.
- [x] Independently review the exact final diff and deliver one atomic correction.

## Consumers and acceptance

Expression admission: `MethodExpr.validateRequirementScopes` admits the union
across Basic/APIKey/JWT/OAuth2 declarations. Keep undeclared-scope rejection.
Shared generation: `BuildRequirementsData` creates per-requirement SchemeData;
`buildSchemeAuth` and its three renderers emit Basic/APIKey/JWT/OAuth2 callbacks.
`SchemeData.Dup` and `ExpandRequirementSchemes` preserve transport data; the latter
is not an authentication decision owner. OpenAPI IR `securityBindings.requirements`
projects API/method requirements; only OAuth2 has OpenAPI scope arrays. Both
versioned rendering and JSON/YAML output consume this IR.

Direct cases: disjoint and overlapping scopes for all four scope-capable kinds;
unscoped APIKey plus scoped JWT; authored ordering; same scheme in separate OR
requirements with different scopes and independent projection storage; scheme
copy preservation; distinct OAuth2 schemes in one requirement; OAuth2 with no
matching scopes renders an empty array, never null; ordinary single-scheme scope
output unchanged; undeclared scopes still rejected.

Reuse existing `runDesignHarness` for one generated compound/OR service. Its
callbacks invoke actual security Scheme.Validate, check each scheme's declared
subset, exercise valid credentials, missing scopes, and fallback to the second
OR requirement. Compare parent/candidate/process-repeat generation for that
specimen and existing ordinary security fixtures, compile generated code. Reuse
existing OpenAPI security tests and validate the rendered multi-OAuth2 contract
with the repository's parser/lint helpers. Existing checked-in SSE fixture scope
is unaffected: only required-scope values change, not transport rendering; reuse
owner hashes and direct ordinary security goldens rather than unrelated streams.

No state/ordering model is needed for a pure set intersection. No new harness,
exhaustive corpus or auth lifecycle policy is part of acceptance.

## Readiness evidence

The compound all-kind endpoint and OpenAPI IR readiness failures were valid.
The original alternative AST row used a title-cased selector with a first-method
fallback, so it repeated the compound witness and is invalid as OR evidence.
The corrected selector uses authored names and requires an exact match. Both
endpoint cases and the IR case fail on an archived parent with the corrected
additive tests; `/tmp/loom-752-corrected-parent-red.log` supersedes that old row.
Root approved implementation based on the valid compound/IR failures and existing
contract: `/tmp/loom-752-root-readiness-verdict.json`.

Existing IR `TestSecurityBindingRequirementComposition` constructs an OAuth2
scheme without its required scope declarations. Its fixture must declare read/write
when adopting the subset contract; this corrects its invalid input, preserving
its current composition and output-isolation assertion. The importer retains its
separate conservative admission for AND OAuth2 schemes requiring different scope
sets; changing that accepted subset is not necessary to repair generation.

## Final bounded evidence

`expr.SecurityExpr.RequiredScopes` intersects in authored order. Each requirement
gets fresh SchemeData; Dup copies its RequiredScopes slice. The endpoint helpers
no longer receive requirement-wide scope policy. OpenAPI IR consumes the same
projection but emits scope arrays only for OAuth2. The unused internal package is
removed. Public documentation describes regeneration and direct callback Validate.

Three actual CLI specimens use an exact archived parent and candidate source:
compound four-kind/OR, two OAuth2 schemes, and the existing single-JWT fixture.
The compound endpoint explicitly excludes its custom-header OAuth2 contract from
OpenAPI, as already required by credential-location admission. A separate OAuth2
specimen uses Authorization and exercises real JSON/YAML output.

All 51 candidate artifacts repeat byte for byte in independent processes. The
compound specimen changes only generated endpoints.go RequiredScopes literals.
The OAuth2 specimen changes only those literals and JSON/YAML scope arrays. All
17 ordinary fixture artifacts are identical to the parent. Parent and candidate
modules build/vet; the generated callback regression fails against parent output
and passes with race detection against candidate output (1.414s). It calls real
Scheme.Validate for valid compound scopes, denied missing scopes, first/second OR
selection and total denial, asserting service calls and auth order.

Focused race checks pass for expr (1.709s), service (1.724s), generator (3.829s),
OpenAPI IR (1.375s), OpenAPI rendering (1.781s), and DSL (1.311s). They include
unknown-scope rejection, session/AND/OR controls, copy isolation, all four scheme
kinds, zero-intersection arrays, both supported OpenAPI versions and existing
ordinary endpoint goldens. OAuth2 JSON/YAML pass the pinned Vacuum contract rules;
libopenapi parses both rendered versions. Scoped lint reports zero issues.

No full-corpus/aggregate CI claim is made. An accidentally broad package command
was interrupted before producing evidence; acceptance uses the explicit focused
selection above. Unchanged SSE fixtures are outside this scope: only auth scope
literals and OAuth2 requirement arrays changed, with actual ordinary generated
and existing security-golden parity established.
