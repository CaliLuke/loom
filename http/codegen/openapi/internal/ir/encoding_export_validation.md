# Source-aware field encoding export

The exporter consumes the occurrence-owned form and multipart contracts,
including nested root alternatives. It groups captured fields by physical name,
then resolves each proposed Encoding Object against every occurrence of that
name. Explicit labels are preserved when equivalent; heterogeneous implicit item
defaults stay implicit. A genuinely incompatible shared physical name produces
an export error with the field name. Style presence suppresses contentType for
both media; `style: form` retains a lone explicit false flag when the renderer
omits `allowReserved: false`.

Source-type metadata belongs to schema structure and fingerprints. A normalized
source-array union retains `type: array`; a source-absent singleton retains its
necessary type assertion inside one `oneOf` branch. Nullable wrapping preserves
original exact-one null membership. The expression source-type validator checks
that concrete provenance contains the normalized non-null domain, using effective
finite enum projection and exact numeric point refinements when structure alone
is insufficient. It never changes the DSL's value domain to fit metadata.

## Bounded acceptance evidence

- Failing direct tests preceded implementation: source array versus typeless
  singleton, heterogeneous item defaults, and nested root authored media were
  lost by the previous exporter. Separate expression regressions exposed
  incompatible source markers admitted before validation.
- `TestEncoding*`, `TestFormEncodingMetadataUsesCapturedNames`, and
  `TestFormUnionEncodingRemainsExplicit` cover source facts, alias isolation,
  null multiplicity, inherited override precedence, metadata name swaps,
  omission, explicit MIME labels, false flags, branch reconciliation and
  unrepresentable conflicts. `TestHTTPSourceType*` covers concrete kinds,
  scalar/array alternatives, number/integer refinements, conjoined enums,
  tagged envelopes, null and source absence.
- Full `expr`, `dsl`, `http/codegen/openapi/internal/ir`, and
  `http/codegen/openapi/v3` suites passed (1.211s, 0.510s, 9.106s, 10.193s).
  Subsequent test-only additions passed their affected tests. Scoped
  golangci-lint passed without findings.
- `EncodingSourceDSL` is the durable four-method form/multipart specimen.
  `TestEncodingSourceRenderedContract` validates JSON and YAML with libopenapi,
  and, with `LOOM_OPENAPI_CONTRACT=1`, the pinned Vacuum rules. Its body-contract
  import excludes unrelated generated document identity and server metadata
  that the strict importer does not admit. Import and Render preserve source
  markers, explicit MIME labels and field contracts. The rendered imported DSL
  was compiled, evaluated, and exported again in an isolated module.
- Two independent processes produced identical JSON, YAML and imported DSL
  bytes for the specimen. Against committed main `5b9a27d`, the existing
  `PayloadFormBodyObjectDSL` and `ParameterStylesDSL` specimens produced exactly
  identical JSON and YAML. The new #1014 source/application specimen exercises
  behavior absent at that parent; its intended differences are restored source
  assertions, preserved branch encodings, and explicit incompatible-name errors.
- The opt-in field-encoding model rejects normalized-shape inference and global
  JSON defaults, then passes 8 generated/4 distinct states. The expression
  source-type model reproduces narrowed domains and lost null separately; its
  checked policy passes 72 generated/36 distinct states. These are finite design
  models, not generator or proof-tool CI requirements.

## Integration boundary

This evidence establishes the exporter and source-provenance owners. The full
four-method imported specimen also generates service/server code, but its client
compile exposed pending generated carrier conversions: `Optional[string]`
elements appended to `[]string`, and named JSON arrays/nested arrays assigned
from native collection carriers. Those belong to the shared generator owners;
no exporter exception or source-contract weakening was introduced. The durable
specimen retains these shapes for the final #1014 integration check. This slice
is not a whole-ticket completion or commit authorization.
