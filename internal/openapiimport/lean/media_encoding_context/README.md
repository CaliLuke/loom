# Media encoding context (#1014)

Run explicitly, outside ordinary checks or CI:

```sh
cd internal/openapiimport/lean/media_encoding_context
lean MediaEncodingContext.lean
```

Lean 4.35.0 checks the occurrence-first inheritance rule used by
`mergeMediaEncoding` during `mediaBranchSchema` traversal. The old descent drops
its outer context; `old_loses_root_json` reproduces explicit root JSON becoming
the typeless octet-stream default. The candidate preserves explicit slots through
arbitrary nested descent and leaves branch defaults intact when no outer slot is
authored. Each Serialization field follows this rule independently.

The representation is immutable: resolving one occurrence never writes its
encoding into the referenced declaration. Go regressions
`TestOneOfRootEncodingContext` and `TestOneOfRootEncodingContextIsolation` check
real inline/reference/nested traversal, alternatives with disjoint fields,
unknown names and distinct consumers. The proof does not establish Go map
copying, reference traversal, field-name membership, codec validity, generated
code, or runtime form parsing. Those remain implementation test obligations.

The expression-side `HTTPFormBodyEncoding` capture uses the same independent-slot
rule when descending untagged form-body alternatives. It validates encoding names
where they are authored, then overlays only Encoding Object slots into detached
branch contexts. `TestHTTPFormBodyEncodingContext` checks nested aliases, disjoint
fields, root content-type precedence and map/slice isolation;
`TestHTTPFormBodyEncodingContextSlots` checks inherited style, explicit false,
non-inheritance of unrelated metadata and invalid branch-owned names. Captured
object openness is resolved separately through the existing expression owner:
it is not an Encoding Object slot and does not propagate to union alternatives.
The Go grammar tests additionally check nullable roots/branches, cycles, map keys,
opaque tagged roots and rejection of tagged root alternatives. These shape and
openness properties are outside the slot model. Lean 4.35.0 was rerun successfully
for this capture integration; model execution remains opt-in.
## Source type, application and authored flags

The source-default review exposed two distinct representation losses. First,
materializing a default into an authored slot lost provenance. Further checking
against [OAS 3.2 §4.14.5.1 and §4.15.1](https://spec.openapis.org/oas/v3.2.0.html#encoding-object)
established that a media label alone also loses item application, and that any
explicit style flag suppresses authored content labels as well as defaults.
The intermediate DefaultMediaType design is superseded, not a retained fallback.

`Schema.sourceType` records the source fact before normalization, inspecting
references and allOf but not inferring types from alternative branches.
`schemaEncodingType` renders necessary overrides as attribute-local
`http:encoding:sourceType` metadata. `urlBranchValue` and `HTTPValueShape` retain
that override plus array Items edges. The shared resolver owns content
application and derives defaults from the actual encoded schema, including
branch-specific item types. Source type facts stay with their attribute;
only authored root Encoding Object slots propagate toward alternatives.

`old_authored_label_hides_style` reproduces the obsolete precedence with an
explicit false flag. The corrected theorems cover flag presence, authored media
when no flag exists, and independent branch defaults. They abstract the selected
style path as text; they do not prove style grammar, array application, null
representation, binary labeling or multipart framing. The shared codec model and
implementation tests own those obligations. A common materialized root label
and specialized schema copies remain rejected because they change branch labels
or add unnecessary declaration identity and naming churn.

`TestEncodingSourceTypeSurvivesNormalization` covers typed and typeless unions,
singleton and nullable normalization. `TestEncodingSourceTypeThroughAllOf`
checks forward reference chains. `TestEncodingSourceShapeAndMetadata` checks
array edges and emitted provenance. The multipart context tests retain inherited
styles, explicit false, source object defaults, two consumers and same-name
binary/array alternatives while replacing obsolete whole-array JSON/conflict
assertions. Full generated root transport validation remains a separate
integration obligation; these checks do not claim it complete.

The corrected context model was rerun successfully with Lean 4.35.0. The direct
importer selection passes, including the prior 44-row matrix and the new source
application cases. Raw binary occurrence permission is now assigned from the
resolved multipart codec at its actual Value/Items target, and multipart field
namespace checks consume the same captured physical claims as expression
capture. Those implementation checks extend beyond this context-only model.
