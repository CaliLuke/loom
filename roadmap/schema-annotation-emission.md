# Imported schema annotation emission (#783)

## Contract and owner

Schema descriptions and titles are already documented as preserved by the OpenAPI
importer. Rendering must preserve the normalized schema's annotations independently
of its constructor shape: scalar, enum, collection, object, named type or supported
reference overlay. Collection annotations belong to the collection; item/value
annotations belong to their nested schema. Reference occurrence annotations must
not change the referenced component or sibling occurrences.

The current renderer emits schema Description only in schemaBlock, while scalar
attributes, array items, map values and scalar request/result bodies consume
validationBlock. Named aliases already use schemaBlock and are controls. The
shared schema annotation emission boundary must own Description, Title and schema
annotation metadata, while structural constructors and value validations retain
their separate responsibilities. Remove the object-only description policy rather
than adding description conditionals to each caller.

Parameter/Header descriptions are a separate role. The parent fixture proves
that query schema descriptions disappear and response-header schemas acquire the
outer header description instead. No existing schema-description metadata carrier
exists. Preserve the current outer Attribute.Description and introduce an explicit
scoped schema-description carrier for these location attributes, emitted by the
same annotation owner with a location role. Shared OpenAPI location-schema
preparation consumes that carrier on its detached prepared schema, independently
of the parameter/header Description. The parameter renderer must stop clearing an
explicit schema description; existing unscoped attributes retain their documented
outer-description behavior. Keep location role at the importer attribute boundary,
not inferred from source paths, and do not mutate normalized or shared named schemas.

This is an internal representation correction required to preserve both authored
texts, not arbitrary precedence or a new public policy. Document the scoped
metadata contract with existing OpenAPI metadata guidance. Preserve parameter
reference overrides and explicit-empty outer descriptions. A schema reference's
base annotation stays on its component; a supported allOf occurrence annotation
belongs to its overlay and must not change siblings. Unsupported direct reference
siblings retain their current diagnostic; URI resolution remains #891.

## Sequence

- [x] Read current report, normalized schema model, renderer callers and exporter.
- [x] Reproduce description loss across shape and constructor roles, retaining
  named/object/Title controls and supported allOf reference precedence.
- [x] Round-trip the existing supported fixture with annotated fields, collection
  containers/elements and named/reference schemas; compile actual generated code.
- [x] Freeze additive readiness evidence and obtain independent review.
- [x] Implement the shared annotation owner after readiness approval; preserve
  source schema and role distinctions, remove superseded object-only emission.
- [x] Run affected direct tests, existing Title/parameter-reference controls and
  bounded fixture generation. Compare exact parent/candidate output, repeat in
  independent processes, compile changed generated code, validate OpenAPI through
  libopenapi and pinned Vacuum rules. List intended annotation/doc deltas.
- [ ] Freeze final diff/evidence for fresh review, then root integrates and delivers.

## Bounded acceptance

Use one direct renderer matrix and the existing supported fixture; no new broad
module harness. Include primitive, constrained primitive, enum, array/map container,
array item/map value, object and named alias, plus reference occurrence versus base
annotations. Reuse existing title coverage and parameter-reference description
controls. Test both Analyze→Render and direct rendering where appropriate.

This is deterministic annotation projection, with no new lifecycle, concurrency or
ordering protocol. Direct assertions and the compiled fixture round-trip check
source-to-output correspondence; a new formal model would duplicate those rules.
Existing reference planning and object-representation models remain unchanged.

## Historical readiness checkpoint

Before production changes, parent was 43edbf9fe. Direct renderer matrix
has 20 failing description cases and eight passing named/object controls; all 28
Title assertions pass. The actual supported-fixture generated module compiles,
then eight of ten schema-description assertions fail and three additional location
schema-description assertions fail. Outer parameter/header descriptions remain
correct, including a parameter with no outer description. This establishes that
preserving schema descriptions must not change those independent values.

Evidence: /tmp/loom-783-direct-parent.log and /tmp/loom-783-fixture-parent.log.
Existing Title and three-version parameter-reference controls pass on this parent
(17.307s) and remain final acceptance gates;
no state model changes are proposed. #891 reference parsing work is independent;
reconcile its delivered helper before final validation if it overlaps this file.

## Final candidate checkpoint

Implementation follows the approved schema annotation and detached location-schema
owners. The ordinary schema Description path no longer depends on object shape.
Parameter/header roles carry an explicit schema description, including empty,
without overriding their outer description or mutating cached prepared schemas.
The same reference occurrence checks retain named component annotations.

Safely advanced to delivered #891 parent e67f359e2 after archiving/inverting the
exact own patch; reference imports/calls compose unchanged. No CI nullable/error
binding repair is included here.

- All 28 constructor cases pass. Sixteen direct IR location/cache combinations pass
  under race (1.560s); full affected IR package race passes (14.039s).
- The compiled annotated fixture and existing Title/parameter-reference controls
  pass under race (24.185s); after #891 composition the direct/reference controls
  pass (15.467s). Scoped lint is clean.
- Actual archived-parent versus candidate generation uses three existing fixture
  scenarios, 48 artifacts per side. Both sides repeat exactly in independent CLI
  processes; all modules compile and candidate modules pass vet.
- Title fixture: all 16 artifacts unchanged.
- Original supported fixture: only JSON/YAML OpenAPI and the design digest change.
  The outer response-header description no longer appears incorrectly on its
  otherwise unannotated schema.
- Annotated supported fixture: seven artifacts change (JSON/YAML OpenAPI, design
  digest, service comments, client/server type comments and CLI type comments).
  New schema descriptions and preserved outer/schema location texts account for
  these differences; code representation/validation stays unchanged.
- All six emitted JSON/YAML specs pass libopenapi parsing and pinned Vacuum rules.

Exact outputs, hashes and command evidence are recorded in
`/tmp/loom-783-final-artifact-comparison.json`,
`/tmp/loom-783-final-artifact-checks.json` and the final review manifest. Final review
and root delivery remain pending. No new state model or exhaustive corpus run.
