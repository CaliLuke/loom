# Explicit empty response bodies — #886, prerequisite to #776

Owner: expr explicit response-body preparation, shared by runtime Body and
OpenAPIBody. An explicit object shape remains an object even when it has zero
fields. Empty remains the no-body sentinel. Inferred bodies with no unmapped
fields retain existing no-body behavior; this does not redefine empty views.

Sequence:
1. Reproduce explicit selected/type/documentation-only zero-field loss at the
   expression boundary, with Empty/inferred/nonempty controls.
2. Extend existing ResponseSourceBinding finite model with explicit empty body
   presence; reproduce the old cardinality collapse and verify source binding.
3. Consolidate duplicate explicit runtime/documentation body preparation at its
   existing expr owner. Delete cardinality-based suppression there.
4. Verify generated HTTP success/error and JSON-RPC selected-body behavior,
   ordinary/bodyless controls, exact selected parent/candidate outputs, isolated
   repeats and compile/vet. Parse/lint affected OpenAPI specimens. Reuse existing
   fixture generators/models; no full corpus or transport workaround.
5. Document the contract, independently review exact final diff, atomic delivery,
   then resume #776 against the corrected owner.

Explicit Body selection and OpenAPIBody already promise the selected type's
shape. This repair follows that contract; importer runtime syntax must express
source content explicitly when ordinary inference would choose no body.

## Acceptance completed; independent review pending

- [x] Three direct expression failures reproduce on the parent: selected empty
  field, explicit named type and documentation-only shape. Empty, inferred
  named-empty, metadata-only and ordinary selected-object controls pass.
- [x] Existing source-binding model extended: cardinality collapse fails the
  authored-presence invariant (14 states), old binding still fails (13), and
  the shared owner passes all three invariants (21). No concurrency claim.
- [x] Consolidate explicit runtime/documentation preparation in
  `expr.explicitHTTPResponseBody`; remove duplicate field-count suppression.
- [x] Archived parent `c2d3d426b` with identical test inputs builds/vets both
  selected modules, then fails actual HTTP/JSON-RPC wire and binding assertions.
  The snapshot has Git metadata referencing the existing parent solely for
  source resolution; production is the exact Git archive, not a CLI overlay.
- [x] Candidate HTTP wire/client/spec controls and ordinary inline selections
  pass under race (12.540s); JSON-RPC's 27 success/error/notification cases pass
  (4.905s), including implicit null bodies and metadata/codec counts. Full expr
  and DSL race suites pass (8.194s/1.976s). Scoped lint passes after a mechanical
  test switch correction. An untyped empty Body callback remains rejected.
- [x] Compare 20 HTTP and 12 JSON-RPC artifacts byte-for-byte against parent and
  independent-process repeats. All 32 repeat exactly. No checked-in generated
  fixture uses the changed explicit named-empty response shape.
- [x] Parse the selected OpenAPI specimen with libopenapi; pinned Vacuum
  v0.33.0 plus the repository's specified dependency override reports zero
  errors (21 style/description warnings, six informational items).
- [ ] Independent exact final-diff approval, commit, push and issue closure.

Intended generated differences: HTTP adds selected/type response body types,
constructors, encoding/decoding and response-contract content types; OpenAPI
adds closed-object content for both successes and the existing typed error.
The typed-error wire body is an unchanged positive control. The same response
plan causes JSON-RPC's explicit/selected methods to encode/decode object results,
and binds the formerly ignored explicit endpoint result. Other JSON-RPC methods
remain unchanged. HTTP CLI help has three synthesized string-example changes,
and one synthesized header example changes in the spec: the newly retained body
participates in example traversal. Both repeat deterministically. The full exact
14-file output diff is `/tmp/loom-886-generated-diff.patch`; artifact hashes are
in `/tmp/loom-886-artifact-comparison.json`.

This does not change implicit named-empty result semantics or introduce an empty
callback DSL constructor. Importers with source-declared response content must
emit an explicit Body when ordinary inference would be bodyless; #776 owns that
mapping and its headerless/wrapped wire acceptance. No transport generator
production changes are needed here.
