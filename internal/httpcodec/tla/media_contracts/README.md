# Media contract selection

Loom preserves a media map as typed union variants. Incoming Content-Type selects
one branch before decoding. Exact types outrank subtype ranges, which outrank
`*/*`; matching parameter constraints refine equal type ranges. Equally specific
overlapping contracts are rejected. Missing, malformed, wildcard and unmatched
incoming Content-Type values are rejected for media unions.

Outgoing selection belongs to the service/client caller through the typed union
variant. Exact variants provide their authored media type. A range variant needs
an explicit concrete content type. The selected concrete type must resolve back
to the selected variant, so a broad branch cannot bypass a more specific schema.
No Accept negotiation is added. Schema/example/encoding ownership stays with each
variant. Ordinary single-media endpoints preserve their current behavior.

Alternatives considered: application-owned raw codecs keep per-media contracts
outside generated validation and leave repeated dispatch glue; choosing a response
from Accept would add negotiation policy independent of the returned typed value.
Typed variants keep the wire choice explicit and reuse existing union validation.

The bounded model explores exact, subtype and wildcard overlap with four headers.
`checked.cfg` preserves specificity through decode and publication (9 states).
`collapsed.cfg` deliberately chooses the first contract and fails Specificity.
This control models a rejected design, not the current importer's explicit rejection.
It does not prove MIME parsing, parameter admission, codec correctness or generated
Go behavior; direct tests and generated exchanges must cover those seams.

Run explicitly with TLC (not a build, hook or CI gate):

```sh
java -cp /path/to/tla2tools.jar tlc2.TLC -config checked.cfg MediaContracts.tla
java -cp /path/to/tla2tools.jar tlc2.TLC -config collapsed.cfg MediaContracts.tla
```

## Physical branch layouts

`PhysicalLayouts.tla` models a JSON union containing a native form branch and a
nested JSON field. `inherited-layout.cfg` reproduces the old conversion rule:
a recorded false presence policy cannot override the enclosing true policy.
TLC finds a form declaration/conversion mismatch. `native-layout.cfg` honors the
recorded policy in both directions and checks all 7 reachable states. This is a
bounded correspondence check, not proof of generated Go layout.

The implementation keeps media-form roots and aliases physically separate in
`transportir.mediaFormCarrier`, records their native layout in
`representation.Layout.Record`, and honors explicit true and false layouts in
`codegen.transformPhysicalContext`. CLI JSON inputs remove the wire form-root
marker from their detached copy. Generated forms exercise optional strings,
collections and JSON map fields; generated JSON unions and ordinary form bodies
remain regression cases.

```sh
java -cp /path/to/tla2tools.jar tlc2.TLC -config native-layout.cfg PhysicalLayouts.tla
java -cp /path/to/tla2tools.jar tlc2.TLC -config inherited-layout.cfg PhysicalLayouts.tla
```

## Validation scope

The parent for this ticket is `62d12653b`; validation uses Go 1.27.3. The bounded generated-output comparison
uses `PayloadBodyUnionValidateDSL`, `PayloadFormBodyObjectOptionalDSL`,
`ExplicitViewDSL`, and `StreamingPartialExamplesDSL`. Generated Go and OpenAPI
files are byte-identical at parent and current source; `gen/loom.json` changes
only its design digest because the expression model gains media mappings. Each
app compiles. Two independent generations of `MultipleMediaContractsDSL` produce
identical bytes and its generated app compiles. The checked-in HTTP ticktock,
JSON-RPC ticktock and mixedtick fixtures are regenerated; only their design
fingerprints change. The HTTP and JSON-RPC ticktock exact-tree/build tests pass.

The new specimen covers equal JSON schemas with different examples, distinct
text/binary/form/multipart schemas, range publication, optional bodies, nullable
JSON branches, per-part types, declared error bodies, custom error formatter
replacement, and enclosing example projection. Import rejection tests cover
incompatible schemas (including nested nullable values through references), part
codecs, field names and array serialization. Required and optional empty forms
retain the selected branch. Mapped Content-Type checks use the semantic source
attribute for requests, results and errors; wildcard parameter values do not
turn an exact media key into a range. Direct tests
cover parameter specificity, rejected mappings and codec boundaries. Mounted
client/server tests cover all four directions, validation, malformed/null/empty
inputs, status/problem responses and absence of service calls on rejection.
OpenAPI JSON/YAML are parsed with libopenapi and linted with the pinned Vacuum
rules for 3.1 and 3.2. The full exported-design corpus is outside this ticket's
acceptance scope.
