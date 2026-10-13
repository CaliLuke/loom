# Imported object representation — #776

Original readiness parent: 6edea49323097517abe52e194b79905e27ba03f0.
Current implementation parent: 720d8a5f70122bb47a4e69711618ea75ff462e4d
(delivered #886 explicit response owner). Implementation follows the
independently approved readiness packet; final
review and delivery are pending.

1. [x] Read the report, expression consumers, DSL constructor defaults and
   existing promotion/admission owners. Keep #783 annotations and #686 byte
   length semantics separate.
2. [x] Reproduce implicit-empty-object corruption and missing inline-map type
   arguments using existing generated roundtrip helpers. Retain named empty
   field/map/array controls and existing rejection/array-promotion policy.
3. [x] Propose one object representation classification before consumers,
   using the existing promotion traversal and reference/name owners.
4. [x] Independent review of this proposal, bounded regressions and the existing
   normalization model's object-constructor extension.
5. [x] Implemented the approved owner; completed affected direct tests, bounded actual
   parent/candidate artifact comparison, compilation and spec acceptance checks.
6. [ ] Independent exact final review before atomic delivery.

## Established cause and contract

`schemaExpression` returns a type expression or an implicit object-body flag.
An Attribute callback establishes an object only by adding child fields; an
empty callback defaults to String. Type and method payload/result callbacks
start as objects. At the recorded parent, the map-value expression caller discarded the implicit-object
flag and generated a missing type argument for both empty and nonempty objects.
This is one object-constructor representation defect, not a collection of
transport-specific failures.

An open memberless object accepts arbitrary object keys and values, regardless
of whether additionalProperties is absent, true or an empty schema. It excludes
scalars and null unless separately nullable. The existing MapOf(String, Any)
carrier preserves this domain. A closed empty object accepts only {}, with
nullable handled independently; a named DSL object already represents it.
Required names, properties, inheritance, typed maps and reference identity must
remain explicit. Do not replace these with an unconstrained map or use Empty,
which is the distinguished no-value DSL type.

## Implemented owner and rejected alternatives

`classifyObject` is shared by the existing promotion traversal and
`objectSchemaExpression`. It distinguishes memberless
free-form maps, typed maps and structured objects while retaining authored IR
and admission failures. Missing additionalProperties on a memberless object
selects the same free-form carrier as explicit true; do not rewrite away source
metadata or turn a record with fields/bases/required names into a free-form map.

`prepareObjectConstructors` generalizes the existing recursive traversal to
prepare concrete object constructors before schema-name allocation and
reference planning. Preserve existing named declarations and ordinary nonempty
field callbacks. Promote an inline structured object when its containing role
requires a concrete type: an empty object field, or any inline object map value.
Reuse `uniqueComponentName`, retain the complete schema on its synthetic named
declaration, and replace only the occurrence with a local reference. Existing
array/union promotion stays in this traversal and retains its documented
consent/admission behavior. No second traversal, namespace allocator or per-body
promotion is needed. Recursion/reference planning remains the existing owner.

All schema expression consumers then receive a usable type or a legitimate
structured callback. The same classifier and constructor-role predicate are
used in the existing planner schema-admission traversal. An unprepared direct Document with a closed
empty field or inline structured map value must receive a source-qualified
admission error before rendering. Ordinary named forms, nonempty field
callbacks and concrete free-form maps remain supported. This explicit admission
boundary preserves Render's nonmutating contract without a second normalization
or copying graph. Map rendering rejects unresolved object flags. The shared
planner still rejects unsupported compositions and mixed additionalProperties before source emission.

Rejected alternatives: forcing empty Attribute callbacks (still String), using
Empty (changes body meaning), representing closed objects as maps (loses closed
membership), adding field/map/body-specific syntax exceptions, or rejecting all
inline object values despite an existing exact named representation.

## Consumer inventory and bounded acceptance

The traversal already visits component schemas, parameters, request bodies,
responses, headers, operation parameters/bodies/responses, properties, map
values, array items and union branches. Extend that owner, not its callers.
Rendering consumers include named declarations, attributes, method payloads and
results, response headers, documentation-only OpenAPIBody/OpenAPIRequestBody,
array/map/union expressions, and requestTransportBodyIsMap classification.
Shared classification must keep typed-form versus raw-body selection coherent.

Readiness tests use schemaAdmissionSource and
requireRenderedDesignGenerates/requireTypeNullAcceptance:

- OAS 3.0/3.1 omitted/true/empty-schema open-object spellings: four expected
  failures across the two dialects and four passing explicit controls.
- Five generated roundtrips: open/closed fields, open/closed map values, and a
  nonempty required-name map value. All five failed at the recorded parent (wrong String,
  missing MapOf argument or invalid Go syntax). These tests assert absent versus
  supplied objects, invalid scalar/null values and closed/required constraints.
- One passing generated named-object fixture exercises closed/open fields,
  closed map/array elements and an ordinary inline record; it excludes extra
  properties from closed shapes.
- Both dialects retain three existing mixed-map admission rejections and one
  inline closed-array promotion with the existing allow-lossy classification.

Direct Render checks three unprepared field/map occurrences fail at shared
admission and an ordinary nonempty field callback remains supported. Final
acceptance adds a nullable closed/open occurrence, a deterministic
synthetic-name collision, shared traversal locations and the existing Render
nonmutation test. Reuse the existing constructor-promotion location test and existing actual generation helper. Capture a bounded selection
of these same inputs at the exact parent and candidate, recording parent failures
as failures. Preserve ordinary named wire/schema contracts; list all intended
new map carriers, synthetic types/references and resulting schema changes.
Compile each unique generated module once, compare a second independent
regeneration byte for byte, validate rendered specs with libopenapi and the
pinned Vacuum rules. No new server harness or full import corpus is required.

The existing schema-normalization Lean model now includes a small
ObjectRepresentation namespace: three parent counterexamples, finite acceptance
preservation, complete constructor arguments, and retained named controls.
It models constructor roles and nullable acceptance, not graph traversal,
allocator uniqueness, arbitrary JSON or generated Go refinement. Real generated
regressions establish correspondence at those boundaries. Existing reference
planning models remain unchanged because their algorithm is unchanged.

#783's annotation emission and #686's encoded byte-length semantics have
separate owners and remain out of scope. There is no unresolved public policy:
this correction preserves object domains already expressible by the DSL.


## Implementation evidence and retained boundaries

The initial corrected five roundtrips and named control pass. Nullable object
fields/map values and a colliding synthetic component name also pass. The reused
location inventory covers three constructor shapes at nine locations, including
response headers. Root JSON requests and header-wrapped responses use the same
role owner as fields; direct Render rejects unprepared parameter/request/response
occurrences rather than generating String callbacks.

The closed-object JSON query specimen reaches correct named representation but
then hits existing #791 DSL location admission. Its exact source and failure are
preserved separately; this repair does not claim query-object runtime support or
change URL shape/location policy. The compiling root specimen therefore covers
JSON body and wrapped response roles; parameter preparation and direct admission
remain explicit structural controls. No unsupported query assertion was changed
into a passing runtime claim.


The same classification now feeds two existing representation consumers: the
untagged-union subset rejects every free-form map spelling, while form bodies
with unconstrained map values retain their documented raw-body mode. Direct
parent/candidate regressions cover those decisions; ordinary named-object/array
unions, inherited array-item promotion, typed form maps and explicit free-form
raw forms retain passing generated controls. #791's URL location check is
intentionally not changed.

## Explicit response-content completion after #886

The strong root response check exposed an independent expr defect: explicit
zero-field objects became Empty before encoding and schema export. #886 now
preserves explicit selected/named object responses through the shared owner,
while retaining inferred bodyless behavior. Its separately reviewed fix is a
prerequisite, not an importer workaround.

Remaining sequence:
1. Add a headerless named and inline closed-object response to the existing root
   fixture; keep the wrapped request/response and assert actual wire and schema.
2. Make the existing response constructor role require a concrete type for an
   empty structured JSON body, including headerless content. At response mapping,
   consume that same representation intent to emit explicit Body for all structured headerless records; raw-body mappings stay
   unchanged. This source-content rule also preserves inherited empty records.
3. Rerun the affected root module and direct role/admission checks; repeat actual
   generation and compare updated artifacts against the delivered prerequisite.
   Reuse unchanged classification/model/normal-fixture evidence with explicit
   source boundaries. Independently review the final diff.

## Final bounded evidence at 720d8a5f7

- Actual parent Git archive plus identical regression inputs: five original
  shape failures, nullable/concrete-constructor failure and root-role admission
  failure remain visible; the named-object and supported-fixture controls pass.
  Candidate runs all nine selected modules successfully (final root14.012s; remaining modules54.497s).
- All 144 generated files repeat exactly in independent generator processes.
  Both ordinary controls retain byte-identical OpenAPI contracts. Their DSL
  now explicitly maps structured response content: the named control changes
  three generated files (named response alias/validator plus loom.json), and
  the supported fixture changes six (response aliases, error constructor names,
  CLI synthesized examples and loom.json). Parent generation failures are recorded as
  missing outputs, never classified as an unchanged passing module.
- Intended differences are concrete free-form map carriers instead of String;
  synthetic named closed/record constructors and references for field/map roles;
  preserved nullable schema/type identity; and explicit selected or whole JSON
  response bodies for source-declared empty objects. Parent inputs that could
  not generate now produce compilable modules. Every artifact delta is recorded
  in `/tmp/loom-776-final-artifact-comparison.json`.
- Shared classification, direct admission, all nine root locations, union/form
  spelling controls pass under race (1.772s). Scoped importer lint reports zero
  issues. The root module verifies four actual `{}` wire responses, generated
  client reconstruction, a required closed-object request, preserved header and
  eight endpoint invocations. The post-#886/pre-mapping missing-inline-content
  failure remains in `/tmp/loom-776-post886-root-before.log`.
- Existing Lean sources/configuration and theorem manifest are unchanged since
  the accepted implementation proof run: 23 theorem checks, no sorryAx. Its
  object-role obligations already cover the explicit response constructor;
  #886's separate model and mounted tests establish body presence downstream.

Final independent review and delivery remain pending. #791 URL admission,
#783 annotations and #686 byte-length semantics remain outside this change.

The first targeted headerless mapping reused the field-constructor predicate.
An inherited empty record exposed why that predicate cannot own body presence:
Extend already establishes an object, yet its inferred body can still be absent.
The preserved counterexample is `/tmp/loom-776-inherited-empty.log`. The final
mapping follows declared structured response content uniformly and shares the
existing Body/OpenAPIBody schema renderer; it does not calculate inherited field
counts. The inherited specimen retains the existing schema-allof-flattened lossy
diagnostic and checks wire presence, without claiming to remove that established
composition boundary. Alias chains use the existing reference index and canonical
JSON-pointer name decoder; an escaped-name control reproduces the old lookup
failure and now passes. Final Render output equals all previously compiled and
independently repeated module designs (2.881s), including mounted root controls.

All nine unique generated modules vet successfully. All 18 JSON/YAML specimens
parse with libopenapi and lint with pinned Vacuum rules at zero errors; style
warnings are retained in evidence. No checked-in integration fixture depends on
this importer lowering: the actual imported supported fixture and named controls
are the bounded normal-output regression surface. The final difference ledger
lists every intentional output; no full exported-design audit was run.
