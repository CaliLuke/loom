# Responses sharing an HTTP status

OpenAPI has one response per status, while a Loom endpoint can return several
errors or tagged successes with that status. The old assignment replaced each
previous response, losing reachable media types and schemas (#642).

`ResponseMedia.tla` explores all six orders of three alternatives: two JSON
schemas and one HTML schema, with and without a conflicting shared metadata
entry. `legacy.cfg` reproduces loss of alternatives. `checked.cfg` accumulates
schema sets and rejects conflicting metadata; it passes 60 generated / 48
distinct states. Equality with the expected schema sets also checks order
independence within this bound.

The chosen projection uses `anyOf`, because alternatives can overlap. `oneOf`
would incorrectly reject values accepted by multiple branches. Media types
remain separate. Non-schema metadata conflicts produce a diagnostic instead
of silently dropping authored intent. Header schemas use the same union;
requiredness is the intersection of all alternatives, including absent headers.

## Correspondence and limits

- `document.go` collects success and error alternatives before status projection.
- `response_media.go` merges schemas, descriptions, examples and headers without
  mutating the transport responses, and rejects incompatible metadata.
- `TestResponseMediaDeclarationOrder` checks whole-document order independence.
- `TestMergeResponseAlternatives` checks schema unions, header case folding,
  requiredness, example retention, non-mutation and metadata diagnostics.
- `TestRenderedErrorMedia` parses both formats with libopenapi and compares bytes
  across independent processes. The fixture also passes Redocly.
- `TestErrorMediaGenerated` checks server/client round trips and each generated
  response contract for both declaration orders.

The model abstracts schema equivalence and metadata compatibility. It does not
prove schema extraction, `anyOf` serialization, example naming, header semantics,
or runtime dispatch. Those are Go test obligations. OpenAPI cannot express
correlations between a particular body alternative and a particular header;
individual generated response contract cases retain that association.

## Run

Use TLA+ 1.7.4's `tla2tools.jar`, SHA-256
`936a262061c914694dfd669a543be24573c45d5aa0ff20a8b96b23d01e050e88`, and Java 25.
From this directory, point `TLA_JAR` to the installed jar:

```sh
java -cp "$TLA_JAR" tlc2.TLC -config legacy.cfg -metadir /tmp/loom-response-media-legacy ResponseMedia.tla
java -cp "$TLA_JAR" tlc2.TLC -config checked.cfg -metadir /tmp/loom-response-media-checked ResponseMedia.tla
```

The legacy run must report an invariant violation; a parser or tool error does
not reproduce the defect. The checked run must pass. Remove temporary checker
state and traces after validation and review.

## Shared response descriptions (#644)

`ResponseDescriptions.tla` models four operation responses in all 24 orders:
two equal status/shape pairs with different descriptions, another shape at the
same status, and the first shape at another status. Status and shape own shared
identity; references own occurrence descriptions. The shape abstraction includes
schemas, headers, media, examples, links, summaries, extensions, explicit naming
and description-omission policy. Omitted descriptions retain their fallback text
in that shape because 3.1 still needs it.

Using the same pinned toolchain above, run `ResponseDescriptions.tla` with
`descriptions-legacy.cfg`, `descriptions-overmerge.cfg`, and
`descriptions-checked.cfg`, each with its own temporary `-metadir`.
The legacy key includes descriptions and fails `SameShapesShare` (exit 12).
The status-only negative control fails `PreservesShape` (exit 12).
The checked key passed: 144 generated / 120 distinct states, depth 5.

`responseKeys` extracts the shape in `reusable_components_helpers.go`;
`componentizeResponses` carries overrides without mutating original responses.
`response_description_test.go` covers description changes, empty strings,
single uses, explicit names, status separation and non-description differences.
The renderer's `TestResponseRefDescriptionRoundTrip` checks absent/empty override
semantics and reused decode destinations in JSON and YAML.
`TestRenderedResponseDescriptions` uses the shared-error-header specimen, parses
both versions and formats with libopenapi, and compares isolated-process bytes.
The existing incompatible-metadata tests remain rejection controls.

The model assumes exact shape equality and collision-free identity. It does not
prove schema normalization, allocated names, codecs, or generated Go behavior.
Response allocation's existing model still applies to the keys supplied to it;
#644 deliberately changes key extraction and automatic naming, not its slot
reservation algorithm. Existing hashed automatic names can change once.

### #644 acceptance evidence

Compared base `5360d1585f16d1cbc6b752e9547cfd1f83966b40` with the change using
Go 1.27.2. Both revisions used the extended `OpenAPISharedErrorHeaderDSL` input.
`ErrorMediaDSL` supplied unchanged multi-media/anyOf controls; `MealPlannerDSL`
and `OpenAPIProblemLinksAsyncDSL` supplied real problem, link and async contracts.
For all four designs in both targets, exact JSON/YAML comparisons found only
response component allocation and description placement changes; resolving local
response references with their description overrides gave equal documents.
The error-media outputs were byte-identical. Six affected checked-in goldens
were regenerated (`meal-planner`, `activity-feed`, `problem-links-async`).

The shared-error and error-media designs generated 24 identical Go files at
both revisions; both generated modules passed build and vet. The checked-in
HTTP ticktock fixture regenerated to 18 identical artifacts at both revisions,
including OpenAPI, and passed its test suite. Generated error-media transport
round trips passed. Both OpenAPI packages passed their tests, including the
conflicting-metadata rejection cases. Redocly 2.60.0 passed the four selected
specimens. Existing TypeScript/Go consumer smoke checks passed; oapi-codegen
2.8.0 additionally compiled the shared-error specimen before and after. Its
expected difference is a new `BadRequestError` type alias and use of that alias
by both operations' 400 response fields/accessors.

Intended differences: description overrides beside response refs, neutral shared
component descriptions, status-based automatic error names (including 5xx),
description-independent suffixes, status separation, and distinct explicitly
named response ownership. Schemas, media, headers, links, examples and runtime
behavior are unchanged. Regeneration may change automatic reference names;
SDKs may add response aliases. Single-use unnamed shapes remain inline.

The optional importer shared-status round trip passed. The exporter-symmetry
round trip failed at **both** revisions on the existing unsupported schema
`$ref` sibling at `/items` GET 200 (`schema-reference-siblings`), before any
response-description comparison. This pre-existing importer limitation is not
part of the acceptance claim.
