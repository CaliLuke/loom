# Query and form serialization readiness

Status: **user adopted uniform JSON for named objects/maps**. The earlier bracket
preservation/export proposal is superseded. Revised readiness independently approved with no unresolved findings; implementation and bounded acceptance complete; independent exact-diff review pending.
Readiness parent: `d0fefa59ed98f20ee4cfe52f4f2c16871135eb1d` (verified pushed main).
Implementation parent: `22e452f21fb380f13aaee93cf86ea058aaaac83b`, fast-forwarded with readiness inputs preserved.
Worktree: `/Users/luca/.codex/worktrees/deep-object-contract/loom`.
Reports: #706, #719, #793. Root owns atomic disposition and delivery.

## Recovery sequence

- [x] Reproduce parent query/form/OpenAPI/import disagreements; inspect actual consumers.
- [x] Obtain user choice: one JSON value for each named object/map at every depth.
- [x] Revise direct import/export/runtime and generated-wire expectations to JSON.
- [x] Refine shared-owner model; demonstrate parent failures and candidate consistency.
- [x] Validate native JSON parameter/form contracts in supported OpenAPI outputs.
- [x] Freeze revised readiness evidence.
- [x] Obtain fresh independent readiness assessment (manifest e02f889e8f2b7d186744303ae4cfb631cfd7996fabd364cbc8f2cdd16df5958d).
- [x] Implement after approval, removing superseded named-composite bracket inference paths.
- [x] Compare bounded parent/candidate output, isolated determinism, generated
  build/vet, actual wire and rejected-input cases; inspect every intended change.
- [ ] Independent exact-diff review, one owning atomic commit, push and reconcile.

## Accepted contract and scope

Named query maps and named form object/map properties carry one JSON document in
one URL-encoded field. Shallow and nested composites use the same mode. Scalar
fields and repeated scalar arrays keep their current spelling. Example decoded
URL values (normal percent encoding happens after JSON serialization):

| Value | Query or form field value |
| --- | --- |
| filter map | `filter={"status":"active"}` |
| nested tree | `tree={"branch":{"leaf":"value"}}` |
| map of arrays | `many={"k":["first","second"]}` |
| mixed ordinary fields | `page=2&filter={"status":"active"}&tag=a&tag=b` |
| fixed form object | `address={"street":"Main Street"}` |

There is no shallow-deepObject/nested-JSON split, dual decoder or legacy switch.
Bracket forms such as `filter[status]=active` are not a second accepted object
codec. Importing deepObject or flattened form maps as a named JSON map is a strict
wire mismatch, not an `--allow-lossy` metadata warning. Malformed JSON must report
normal decoding errors; a JSON parse failure must not fall back to brackets.

Whole-query MapParams keeps its unprefixed root-key contract. Root form maps keep
their root keys (including empty keys), without a synthetic JSON wrapper. OAuth
constructor-union forms keep their existing top-level fields/discriminator.
These root layouts are distinct from a named object/map property; they do not
justify preserving bracket encoding for ordinary named composites. Preserve the
existing generated form union/custom codec boundary and ValueTarget's physical
value conversion; inspect nested union cases before claiming coverage.

Fixed-object query admission is not expanded by this repair: current query DSL
supports maps, not arbitrary fixed object parameters. Reject imported fixed
object JSON query parameters until a separately supported DSL path exists.
Arrays of objects/maps directly used as form properties remain a separate
admission defect; arrays *inside a named JSON object/map* are part of that JSON.
No defaults/absence/explicit-empty M7 policy is added here. Existing ownership of
presence, nil and requiredness must feed the new codec rather than be reinvented.

## Standards representation

Named query maps use a Parameter `content` entry with media type
`application/json` and the real typed schema. They have no sibling `schema`,
`style` or `explode`. Other ordinary query parameters can coexist, so no whole
querystring promotion or custom media type is needed. Named form composites use
Encoding Object `contentType: application/json`; omit style, explode and
allowReserved so RFC6570 serialization cannot override media encoding.

The [OpenAPI 3.2 Parameter Object](https://spec.openapis.org/oas/v3.2.0.html#parameter-object)
and [Encoding Object](https://spec.openapis.org/oas/v3.2.0.html#encoding-object)
provide these representations. Check the same rendered form in the supported
[3.1.1 contract](https://spec.openapis.org/oas/v3.1.1.html#encoding-object).
Schema aliases/references, validation and actual media examples stay typed.
Authored style metadata that conflicts with the selected built-in codec must
produce a source-located diagnostic, not silently override runtime behavior.

## Shared cause and owner

The parent has independent rules for query emitters, reflected form recursion,
OpenAPI parameter/media export and importer admission. Current query/form codecs
use brackets while export omits their serialization; importer rejects explicit
modes and accepts incompatible defaults. Repair that contract once.

Use dependency-neutral `internal/httpencoding` for location/shape classification,
selected wire mode and normalization of external serialization declarations.
Inputs distinguish named query, whole-query map, named form field and root form;
scalar, scalar sequence, map, object and unsupported/opaque shapes; source
dialect and explicit serialization fields. Outputs are scalar, repeated field,
JSON field, root layout or unsupported. Alias/reference adapters retain source
identity/diagnostic paths and do not duplicate the mode decision.

- Transport IR carries the selected mode on parameter/body-field occurrences.
  Query client/server generation derives from it; remove old map bracket
  inference. Reuse ValueTarget's physical/presence/type conversion.
- Form runtime applies the same named-composite decision to reflected/generated
  fields while root layout/custom form marshalers remain explicit boundaries.
  JSON bytes use the framework's JSON behavior, including deterministic maps,
  type aliases, escaping and custom JSON marshalers. The outer form tag names the
  URL field; inside its JSON document, JSON names/tags own field serialization. Do not keep independent
  form-only depth rules.
- OpenAPI IR consumes the selected mode for parameter content and form encoding.
  Every exported built-in declaration must match its actual runtime codec.
- Import normalizes content/media/style/explode with resolved schema shape and
  compares to the same mode. Admit one application/json parameter content entry
  only for supported shapes; retain strict diagnostics for unsupported media,
  ambiguous schema+content, bracket modes and mismatched array serialization.
  Remove blanket Explode rejection; explicit equivalent scalar/repeated defaults
  remain valid. Source default JSON form composites are equivalent where the
  dialect defines them, but explicit style settings must be normalized first.

The implementation uses this small mode owner; physical value and custom-codec
semantics stay in the existing value graph and framework JSON codecs.

## Readiness revision and migration

The earlier reviewed readiness established the owner and parent defects, but its
bracket-preserving tests/model are superseded by this accepted contract. Preserve
historical log provenance; do not cite old passing bracket-wire tests as proof of
JSON behavior. The previous native querystring candidate is no longer selected.

Migration is a wire change: regenerate Loom clients/servers and update external
clients to serialize the complete named object once, then let their URL/form
library percent-encode the field. JavaScript: `new URLSearchParams({filter:
JSON.stringify({status: "active"})})`; Python: `urlencode({"filter":
json.dumps({"status": "active"}, separators=(",", ":"))})`. These examples express
the contract independent of Go. Scalar/repeated fields, OAuth root forms and
MapParams retain their layout. No compatibility codec is planned.

## Bounded acceptance

Retain the existing scalar map fixture and add aliases/remapped names, map of
arrays, nested maps, a mixed scalar/array/map request, named form objects/maps,
and referenced imported schemas. Direct runtime tests cover JSON encode/decode,
wrong type/malformed/trailing content, reserved characters, empty map keys and
bracket non-decoding. Presence/default policy remains a separate owner; do not
assert a new nil/empty distinction to make JSON tests pass.

Generated modules must compile and vet before wire execution. Check client
encoding and server decoding independently so one incorrect side cannot mask the
other; use external JSON field values and explicit malformed requests. Test
HTTP status/problem detail and non-invocation for malformed requests at the
mounted generated handler before final review. Reuse OAuth/root-map integration
controls. Import accepted JSON -> design -> full generated compilation and wire;
strictly reject bracket/flattened incompatible sources including --allow-lossy.
Validate rendered 3.1/3.2 JSON/YAML with libopenapi and pinned Vacuum. Compare exact
parent/candidate outputs for selected cases, listing intended runtime, schema,
imports and documentation changes; repeat generation in independent processes.
Do not run the full exported-design corpus.

The historical parent module pin is verified
`v1.10.0-alpha.9.0.20261012211839-d0fefa59ed98`; generated comparisons explicitly
replace the framework source with this worktree. A baseline update may reuse a
new verified pushed pin, but must identify which source each log actually tests.


## Revised parent evidence

- `/tmp/loom-uniform-json-direct-red.log`: runtime still emits/reads brackets,
  ignores invalid JSON, and exporter lacks parameter content/form JSON encoding.
  Import rejects matching JSON map media and explicit equivalent scalar defaults;
  it still admits incompatible implicit-form query maps. Accepted/rejected
  controls in the same table remain passing.
- `/tmp/loom-uniform-json-generated-local-red.log`: **explicit local source** and
  verified pushed parent module pin; full generated HTTP service/client/server/CLI
  build and vet pass, then actual query/form client encoding differs from JSON
  and independently supplied JSON decodes to empty composite fields. Mixed page
  and repeated tags retain their wire values. Malformed JSON also wrongly passes
  both generated decoders. This is a red compile-before-wire witness.
- `/tmp/loom-uniform-json-generated-red.log` was an initial exploratory run using
  the harness's default remote source. It is superseded by the explicit-local log
  above; it must not be cited as exact-parent runtime evidence.
- The imported JSON/ref compile test currently stops at strict admission, as
  expected. Full import-to-generated-module evidence is an implementation gate;
  do not claim it already compiles merely because the parent wrong-map probe did.
- The revised finite model passes 160 states for shared selection. Parent,
  metadata-only and client-only choices each give targeted counterexamples;
  model assumptions and commands are beside its sources.

The separate composite-form-array admission red remains classified outside this
repair; omit it from the eventual atomic production diff or preserve it as a
separate durable repro. Likewise whole-query MapParams keeps its wire format;
its known 3.1 deepObject export mismatch is an owner-coherence regression, not a
reason to JSON-wrap the whole query. Historical evidence is retained only for
parent provenance and superseded-contract comparison.


Final readiness controls: `/tmp/loom-uniform-json-native-contract.log` passes
libopenapi and pinned Vacuum for candidate typed JSON parameter/form documents
in 3.1.1 and 3.2.0. `/tmp/loom-uniform-json-root-controls.log` passes existing
OAuth and root-form-map generated integrations with explicit local source.
`/tmp/loom-uniform-json-readiness-lint.log` reports zero scoped lint issues.
At that readiness freeze no production source was modified. The revised frozen manifest is
`/tmp/loom-uniform-json-readiness-manifest.json`; old deep-object manifest refers
to superseded bracket-preserving inputs and must not be used to authorize this
new contract.


## Initial implementation acceptance (superseded by correction below)

Implementation parent is `22e452f21fb380f13aaee93cf86ea058aaaac83b`.
The verified pushed temp-module pin is
`v1.10.0-alpha.9.0.20261012215933-22e452f21fb3`; all candidate subprocesses use
`LOOM_DIR=/Users/luca/.codex/worktrees/deep-object-contract/loom` or an explicit
local replacement. The comparison parent is the managed `json-field-parent`
worktree at that exact commit. Toolchain: Go 1.27.2, golangci-lint 2.14.0;
OpenAPI contracts use the repository's pinned libopenapi/Vacuum harness.

The shared owner now drives transport parameter/form-field modes, JSON value
projections, reflected form boundaries, OpenAPI export and strict import.
`http.EncodeJSONField` uses Loom's JSON options plus deterministic map order;
`DecodeJSONField` reuses the existing non-nullable JSON decoder. Expr forbids
Nullable URL fields, so top-level null is an error rather than absence. Custom
JSON codecs and inner JSON tags remain authoritative. Scalar `allowReserved:
true` imports remain accepted: percent-escaped reserved characters preserve the
same decoded scalar, and permitting unescaped characters does not require them.

Intended output differences:

1. Named query map client/server loops become shared JSON-field calls, with safe
   decode errors, existing absence/default handling and post-decode validation.
2. Whole-query MapParams selects its existing unprefixed decoder instead of
   accidentally entering the named-map branch. OpenAPI 3.1 reports form style;
   3.2 retains the existing querystring promotion.
3. Named form maps/objects use JSON; root maps and constructor/OAuth forms keep
   their layouts. Named fields inside a root map still use their own shape.
4. Named JSON query schemas/examples move into Parameter.content, retaining
   referenced/reusable schema declarations and removing incompatible sibling
   fields. Form composites gain Encoding.contentType application/json.
5. Strict import admits matching JSON and equivalent scalar/repeated defaults;
   incompatible bracket/flattened modes remain fatal even with allow-lossy.

The 50 changed checked-in goldens comprise 48 HTTP request snippets and two
scalar-map OpenAPI documents; each follows the changes above. A bounded complete
mixed query/form/root design produces 12 Go files: only the client and server
`encode_decode.go` differ from parent. Independent candidate processes produce
identical bytes for all 12. Eight rendered specimens (scalar-map-keys and form,
3.1/3.2, JSON/YAML) differ only in query media/schema placement and form encoding;
all eight repeat byte-identically. Full per-file hashes and semantic paths are
in `/tmp/loom-json-comparison/final-output-comparison.json` and
`spec-semantic-differences.json` beside the parent/candidate/repeat artifacts.
The parent generated module built and vetted before its expected wire failures;
all candidate generated modules build, vet and pass actual wire tests.

Acceptance logs (preserve through independent review, then remove temporary
artifacts while retaining this summary and durable sources):

- `/tmp/loom-uniform-json-final-packages.log`: complete http runtime, shared
  owner, representation, transport IR, OpenAPI IR and v3 packages pass.
- `/tmp/loom-uniform-json-final-generated-gates.log`: request goldens,
  mixed generated wire/mounted handlers, numeric named-map aliases with
  validation, and custom multipart bindings pass. Malformed, whitespace, null,
  wrong-type and trailing JSON return HTTP400/decode_payload with safe detail,
  zero service calls and no delivered payload; external valid JSON returns204.
- `/tmp/loom-uniform-json-final-harness-format.log`: mixed generated harness
  passes after final source formatting.
- `/tmp/loom-uniform-json-final-import-wire.log`: strict accepted/rejected
  import matrix and imported referenced-map design generation/build/vet/wire pass.
- `/tmp/loom-uniform-json-final-root-controls.log`: existing OAuth constructor
  form and empty-key root-map generated integrations pass.
- `/tmp/loom-uniform-json-final-spec-contract.log`: actual rendered 3.1/3.2
  JSON/YAML specimens pass libopenapi and pinned Vacuum contract rules.
- `/tmp/loom-uniform-json-final-lint.log`: scoped golangci-lint, zero issues.
- `/tmp/loom-uniform-json-final-model-*.log`: checked160 states; each of five
  parent/partial-owner configurations reproduces its documented113-state
  counterexample. Model correspondence and limitations are beside its source.

Representative reruns:

```sh
LOOM_DIR="$PWD" go test ./http ./internal/httpencoding ./http/codegen/internal/representation ./http/codegen/internal/transportir ./http/codegen/openapi/internal/ir ./http/codegen/openapi/v3 -count=1
LOOM_DIR="$PWD" go test ./http/codegen -run '^(TestDeepObjectReadinessGeneratedWire|TestJSONFieldNumericAlias|TestClientEncode|TestDecode|TestCustomMultipartBindingsGenerated)$' -count=1
LOOM_DIR="$PWD" go test ./internal/openapiimport -run '^TestUniformJSONReadiness' -count=1
LOOM_DIR="$PWD" LOOM_OPENAPI_CONTRACT=1 go test ./http/codegen/openapi/v3 -run '^TestJSONFieldRenderedContracts$' -count=1
```

No full exported-design corpus was run. The full importer package exposed the
parent-confirmed valid Bearer/OAuth source-coalescing defect, now tracked by root
as #801; `TestImportPreservesRepresentableSecuritySchemes` is unchanged. Earlier
scalar explode and media expectations were corrected only where the adopted
contract requires it, with focused importer controls passing. The separate
composite form-array admission repro is preserved as
`roadmap/deep_object_composite_form_array_repro.go.txt`; it is not a passing gate
or a claimed repair. M7 presence/default semantics, fixed-object query admission
and composite arrays directly used as form properties remain outside scope.

Against delivered main `25179e87b8b5d4e9cf5bb233d1818600469d060d`, the only
shared edited file is `http/codegen/service_data_element_types.go`: this repair
adds ParamData.JSONField, while #709 changes named-validation generation. Root
must preserve both owners during integration and review any changed diff.
Shared consumer/framework skills and master trackers remain root-owned.

## Independent final review correction

Final review of manifest fd007511a99fec42669d8464e0f9949eb4cf2e06d48207e824c6f04effd5a620
found two medium issues. Production delivery remains blocked pending confirmed
corrections: named JSON query/form maps decode null elements into native zero
values because their physical carrier misses the existing JSON Optional element
wrappers; the external import harness gained an invalid internal-package import
during final formatting. Earlier green import evidence predates that accidental
edit and does not authorize the frozen harness.

Correction sequence: preserve reviewer reds, remove only the invalid embedded
import and rerun the exact external harness; select existing JSON physical
carriers at each named composite boundary, retaining native outer form layouts;
add nested map/array/object adversarial checks; rerun affected generated output,
compile/wire, schema and determinism gates; refreeze for focused re-review.
No manual recursive null scanner or new absence/default policy is authorized.


### Correction acceptance

Both review reproductions are corrected; fresh re-review remains pending. The
import harness's internal testprocess dependency exists only in the framework
host test. The emitted external wire_test.go imports public APIs only. Its full
build/vet/wire rerun is `/tmp/loom-uniform-json-review-import-fixed.log`, and the
complete focused import matrix rerun is
`/tmp/loom-uniform-json-carrier-import.log`.

The JSON carrier correction reuses existing Optional collection/object carriers
and GoTransform. Form JSON subtree declarations get separate names and JSON
layouts, preserving effective inherited constraints/defaults and opaque custom
codec boundaries; the form root remains native. Query fields decode into the
existing JSON carrier and transform into their native parameter only when
present. Shared transformation dispatch now applies registered JSON layouts at
named collection boundaries, removing the previous helper-only inference. A
root form map containing composite entries needs the corresponding payload
constructor even when the service type itself has no named types.

Direct correspondence: `codegen/json_boundary_transform_test.go` fails for both
map and array fields on parent22 and passes on the corrected source. The
initial seven generated nested-null cases all fail before correction; the final
matrix adds nested map/array form cases and root-form composite entries. It
checks map-value null, array-element null, nested map-leaf null, and object-field
null through mounted handlers with safe errors and no invocation, plus valid
values and conversion. The root-form map uses an empty outer key. Existing
OAuth, root scalar-map, numeric map alias validation, and bytes-default form
controls pass.

The updated model explicitly models physical carrier selection: native-json
still agrees on serialization mode but violates JSONCarrierRejectsNull at113
states; checked passes160 states. All five original failure configurations were
rerun against the updated source and retain their counterexamples. The model
cannot establish decoder correctness; generated tests verify correspondence.

Latest evidence:

- `/tmp/loom-uniform-json-carrier-final-generated.log`: combined generated
  build/vet/wire, mounted null matrix, form root/bytes/OAuth and alias controls.
- `/tmp/loom-uniform-json-carrier-codegen-package.log`: full shared codegen
  package passes after transformation ownership change.
- `/tmp/loom-uniform-json-carrier-owner-packages.log`: representation,
  transport IR and both OpenAPI packages pass.
- `/tmp/loom-uniform-json-carrier-goldens-check.log`: selected request,
  body declaration, constructor, client/server type and payload goldens pass.
- `/tmp/loom-uniform-json-carrier-final-lint.log`: scoped lint, zero issues.
- `/tmp/loom-json-boundary-parent.log` and `loom-json-boundary-candidate.log`:
  direct transform red/green; `/tmp/loom-uniform-json-nested-null-red.log`:
  original generated null reds.
- `/tmp/loom-json-carrier-model-*.log`: updated model results.

The expanded bounded comparison uses the same current input design on parent
and candidate, now including root form composite entries and nested form maps
and arrays. All12 generated files repeat byte-identically. Six files differ
from parent: client/server encoding, client/server types, and two CLI files.
Besides previously listed JSON encoding changes, intended differences are
named JSON physical carrier declarations, collection unwrap conversions,
root-form entry constructors, renamed form conversion helpers, and synthesized
CLI example text due to new carrier identities. CLI flags and service APIs do
not change. `/tmp/loom-json-comparison/carrier-output-comparison.json` records
all hashes; `carrier-output.diff` records each exact difference. Both candidate
processes pass full module build/vet/wire. The parent builds/vets before expected
wire failures. Spec bytes regenerated from this source match all eight already
validated candidate specimens exactly, so the prior libopenapi/Vacuum evidence
remains valid without repeating identical linting.

The revised review manifest is
`/tmp/loom-uniform-json-revised-final-manifest.json`. The prior manifest remains
historical and does not authorize this correction. Compared with25179e87b, two
files overlap root's delivered work: `service_data_element_types.go` and
`service_data_type_registry.go`; preserve #709's validation-function ownership
alongside JSON fields/layout registration. No main/shared-guide/tracker changes
or commits were made.

## Final correction review and integration

Independent re-review approved all 105 files with no unresolved findings, including
both medium corrections. Manifest `b38b484912fd2c982368e8d6640f27e7d2460ffe5a53d8d2c261e76eec46e136`;
patch `dc7fbcd48f642fbd4d2f4553ffa35290c72bb1cce7a2455964ebaf63864aae2b`.
The reviewer independently reran shared collection conversion/carrier checks,
mounted nested-null/root controls and the imported module's generation/build/vet/
wire checks. Integration onto `81192e7a9` preserves the candidate files except
for two HTTP registry merges retaining #709's named-validation owner. The three
shared guides are updated with the accepted contract. Bounded overlap checks
cover recursive/named-nullable generated consumers and the delivered problem
projection path before scoped integration review. #801 remains separate.

Integration acceptance passes with explicit current source:
- shared recursive/physical-carrier checks 0.669s;
- HTTP recursive, named-nullable, JSON carrier and problem projection generated
  matrix 65.862s; JSON-RPC recursive/named-nullable 16.359s; gRPC 6.392s;
- mounted query/form JSON matrix and alias/carrier controls 10.645s;
- importer admission and external module generation/build/vet/wire 13.084s.
Logs: `/tmp/loom-json-integrated-overlap.log` and `loom-json-integrated-wire.log`.
A fresh exact-parent (`81192e7a9`) archive and the integrated candidate each render
all twelve comparison artifacts; two independent candidate processes match every
byte and pass build/vet/wire (7.149s/7.328s). Parent build/vet succeeds and the
old bracket serialization fails the expected wire matrix (2.765s). The six intended
changed files remain the same as the earlier comparison. Against the approved
isolated candidate, only server/types.go differs: #709's validator context
arguments and diagnostic paths. See `/tmp/loom-json-integrated-output-comparison.json`
and `loom-json-integrated-output.diff`. Initial parent setup attempts had no usable
results (archive extraction/tooling and test discovery); the final run copies the
same harness into a source archive with local Git metadata and explicit LOOM_DIR.
No fixture or schema-policy differences beyond the already-reviewed scope are
claimed. Independent integration review is required before delivery.
