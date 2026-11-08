# SSE response import

The strict OpenAPI 3.2 inverse recognizes a single `text/event-stream`
representation whose inline `itemSchema` is a closed object containing only
required `data`. That member must be a string with `contentMediaType:
application/json` and a `contentSchema` describing the event. The envelope is a
wire description, not a service value. Only its decoded payload enters normal
schema admission. Ordinary content keywords remain unsupported in every other
schema position.

The operation response plan classifies selectors before allocating names.
Every successful response must describe the same event declaration and equal
normalized local contracts. It preserves every response's source path and
headers, then selects the first canonical selector in lexical order for the
common `StreamingResult`. When metadata or status selection needs a carrier, a shared named handshake
type carries those fields. A singleton exact status without headers does not
invent an empty metadata type. No event `Body`, raw-body flag, or independent documentation
example owner is introduced. Ordinary encoded, raw, file, error and
informational response rules remain separate.

The reserved `x-loom-async` extension is checked before opaque extension
rendering. It must describe the actual route, method, canonical successful
selector list, content types and complete outbound event contract with all five
SSE mappings null. Unknown members, inbound messages, projections, legacy scalar
status and conflicting payloads reject. The importer consumes an admitted
contract; it never emits the reserved key as authored metadata. Reserved schema
roots remain comparison-only analyzed graphs: they do not receive native object
promotion, allocate public declarations, or acquire a second rendering role.
Source resolution and normalizer diagnostics still apply. Complete equality to
the ordinarily admitted event is mandatory; that event owns representability.

Resolution uses the existing loader and fixed point. The grammar records
operation positions and the exact payload position in the fixed envelope. Once
an operation's Response reference chain reaches an item-schema stream, only
`messages.inbound.schema` and `messages.outbound.schema` become typed reserved
roots. Examples, defaults and unrelated extensions remain literal data.
Referenced declarations participate in ordinary selection and naming. Recursive
comparison uses typed edges and a pair guard rather than searching arbitrary
JSON objects for `$ref` strings.

Local assertions, exact numeric bounds, default presence, ordered samples and
sample metadata participate in comparison. Equal nonempty local examples are
kept on the event occurrence. Child-owned and explicit-null examples retain
normal schema ownership. No custom example hook is called during import.
Authored `examples: []` has no faithful current DSL representation: neither
absence nor suppression of generated examples removes inherited authored
samples or preserves an empty authored group. Such occurrences receive a
`schema-examples-presence` diagnostic, including when no inherited example is
present. This is an intentional strict-admission change for ordinary non-SSE
schemas too, including inline schemas and reference overlays that older imports
accepted while losing the empty-group presence. Nonempty examples and ordinary
admitted output remain unchanged. The importer does not substitute a metadata
workaround.

Reference and expanded graphs are compared conservatively. Bare references and
annotation-only overlays can be paired with inline expansions only when exactly
one side is a reference. Two references must name the same declaration at root
and child positions; equal shapes or alias targets do not erase a distinct
reference claim. Recursive cuts retain the same declaration identity. Local assertion
intersections retain their normalized reference form rather than invoking a
general schema-equivalence engine. Exporter-produced contracts are additionally
subject to first- and second-roundtrip acceptance before the capability can be
released; authored-fixture compilation alone does not establish that boundary.

Focused checks use `TestImportSSEInverseRequired`, `TestImportSSEReservedContract`,
`TestImportSSEOccurrenceContract`, the strict envelope and annotation controls,
and `TestResolverSSETypedSourceClosure`. `TestImportSSEArtifacts` can retain the
bounded rendered DSL with `LOOM_IMPORT_SSE_OUTPUT` for process-isolated byte
comparison. Models remain opt-in design evidence, not test or CI dependencies.
