# Imported byte representation (#686)

Status: implemented and validated on `53849ce33`; exact final review pending.
The user adopted encoded String with shared standard padded Base64 validation,
rejecting whitespace/bad padding and accepting decodable alternate pad bits.
Independent readiness re-review approved the exact packet on 2026-10-13
(`/tmp/loom-686-independent-readiness-rereview-verdict.json`); implementation is authorized.

## Contract and owning boundary

OpenAPI string constraints describe the authored string representation. The
importer currently selects decoded `Bytes` for both `byte` and `binary`, then
copies encoded length bounds, patterns and string literals into that semantic
carrier. Keeping the exported format annotation does not correct this mismatch.
The correction must choose one representation before emitting type, validation,
defaults, enums and examples; Analyze and direct Render must share admission.

The official format registry describes `byte` as an RFC4648 base64 string and
`binary` as raw octets, suitable only for a whole body or multipart part. Source:
https://spec.openapis.org/registry/format/byte and
https://spec.openapis.org/registry/format/binary (checked 2026-10-13).
Raw-body schema metadata must remain distinct from JSON value conversion.

## Sequence

1. [x] Inspect current importer type/constraint/literal emission, planner and raw
   body roles; reuse existing byte-format and form/multipart fixtures.
2. [x] Reproduce parent evaluation/runtime disagreements for bounds and literals,
   with normal string and raw binary controls. Record actual unsupported contexts.
3. [x] Compare a decoded-byte translation against retained wire-string lowering.
   Preserve source spelling/meaning and distinguish mathematically invertible
   bounds from non-injective byte decoding, pattern and finite-domain contracts.
   Freeze the shared owner proposal and concrete accepted/rejected cases.
4. [x] Reuse the existing byte-length arithmetic proof for any inverse-bound
   operation; model only new representation obligations the selected plan needs.
5. [x] Obtain independent readiness review before production.
6. [x] Implement the agreed owner, remove competing lowering, and run bounded
   direct/generated/wire/spec and determinism checks.
7. [ ] Obtain independent exact final review and deliver the atomic repair.

## Existing owners and constraints

`internal/byteschema` projects decoded byte bounds to padded-base64 constraints;
`internal/valuecontract/BYTE_SCHEMA.md` records codec and enum limits, and
`expr/lean/value_projection/ValueContract/ByteLengthProjection.lean` establishes
its arithmetic. Reuse them rather than introducing a second arithmetic policy.
The decoded interval for encoded minimum n>0 is at least
3*floor((n-1)/4)+1; encoded maximum n admits at most3*floor(n/4) decoded bytes.
Empty intervals and large source integers require exact admission.

`render_schema.go` currently emits both representation choice and generic scalar
validation/literals. `planDocument` owns admission and shared rendering plans.
`render_request_body.go` and response plans distinguish built-in JSON values,
application-owned raw/form bodies and schema-only file/stream bodies. Component
references may reach more than one context, so a format annotation alone cannot
select a codec or make shared source storage mutable.

The ordinary byte-format test currently verifies exported formats, including a
nested JSON binary field, but does not establish runtime contract equivalence.
Do not treat that accidental admission as a settled raw-octet JSON contract.
Conversely, retain supported raw response and multipart shapes instead of
rejecting every binary annotation.

Readiness remains bounded: reuse the importer evaluation/generation helpers and
byte-format fixture, one meaningful mounted byte matrix, the existing schema
validator and arithmetic model. No new framework-wide corpus or transport fix.

## Accepted public choice and considered alternative

Accepted: import `type: string, format: byte` as `String` with an ordinary
`FormatBase64` validation in the existing shared format owner. Preserve source
lengths, patterns, defaults, enum membership and examples as encoded strings.
For example, `minLength: 4` admits `"aGk="` (two decoded bytes), `maxLength: 8`
rejects `"aGhpamtsbw=="` (twelve encoded characters), an enum containing only
`"aGk="` admits that exact spelling, and the default `"aGk="` stays unchanged.
Applications receive a Go string and decode it explicitly only when they need
bytes. This makes the source contract portable across languages, at the cost of
changing imported service/client field types from `[]byte` to `string` after
regeneration. Native authored Loom `Bytes` keeps its existing decoded-byte API.

Alternative: retain decoded `Bytes`, translate encoded length bounds, decode
literal defaults/examples, and reject source patterns or finite enum contracts
that the many-to-one decoder cannot preserve exactly. This retains convenient
native byte access but rejects representable wire-string contracts and requires
more admission restrictions. Merely translating lengths is insufficient: an enum
containing `"aGk="` must not accidentally admit `"aGl="` just because both decode
to the same bytes. Silently dropping patterns or weakening enum equality is not
an option. Retaining plain String without base64 validation is also not an option.

### Accepted Base64 grammar

The accepted format validates the standard RFC4648 alphabet and complete padded
quanta: `""`, `"Zg=="`, `"aGk="`, `"Zm9v"`, and `"Zm9vYmFy"` are valid;
`"aGk"`, `"aGk=="`, `"a=k="`, whitespace/newlines, and the URL alphabet `"__8="`
are invalid. RFC4648 §§3.2–3.3 require padding and rejection of non-alphabet
characters unless a referring specification explicitly changes those rules.

The user selected tolerant unused pad bits after considering these alternatives. RFC4648 §3.5 requires
encoders to emit zero pad bits but permits decoders to reject nonzero pad bits;
it does not mandate that rejection. A tolerant validator admits `"aGl="` as a
base64 string while preserving that exact spelling; an authored enum containing
only `"aGk="` still rejects it. A canonical-only validator rejects `"aGl="`
regardless of other constraints. Tolerance preserves source spelling and avoids
rejecting decodable peer input; canonical-only acceptance narrows interoperability
but guarantees one spelling per decoded byte sequence. The current JSON codec's
choice is supporting implementation evidence, not authority for this public rule.
The wire-string representation supports either rule coherently; the user adopted
the tolerant rule checked by the readiness format table. Production starts only
after independent readiness approval.

Primary evidence: [OpenAPI byte registry](https://spec.openapis.org/registry/format/byte)
identifies a JSON string using RFC4648. [RFC4648 §§3.2–3.5](https://www.rfc-editor.org/rfc/rfc4648.html#section-3.2)
defines padding, alphabet and canonicalization. The registry recommends
`contentEncoding: base64` for OAS3.1; that keyword is currently explicitly rejected
by this importer and adding it is outside this repair.

## Proposed shared ownership and bounded acceptance

- Add Base64 to the existing `pkg.ValidateFormat` contract, then its DSL/expr
  supported-format inventory, generated constant selection, JSONShape rules and
  example synthesis. Do not create a parallel importer-only validator.
- One importer string representation decision selects String+FormatBase64 for
  encoded `byte`; every field, named alias, array/map element, nullable occurrence,
  parameter and JSON body consumes it. Generic literal/constraint emission can
  then preserve the wire-domain values without decoder-specific rewriting.
- Raw `binary` remains Bytes only where the actual protocol represents raw octets
  (supported whole raw response body or whole multipart part). Whole raw request
  bodies remain outside the importer’s existing media subset; this repair does
  not add raw request support. Shared occurrence admission must
  traverse references with the actual location; a component used by both raw and
  JSON paths cannot be globally cached as safe. Reject unsupported JSON/text
  binary occurrences before rendering, preserving actual raw/multipart controls.
- Compare parent/candidate/repeat outputs for the existing byte-format fixture,
  one mounted bounds/literals/pattern/nullable/nested matrix, and the existing
  multipart/form/raw-media control. Compile and validate generated specs through
  the existing importer helpers. Record every intended type/validator/example
  and spec difference; ordinary string and raw outputs must stay unchanged.
- Extend the existing schema normalization model only with the representation
  choice, concrete length/literal counterexamples and non-injective enum negative
  alternative. Existing native Bytes projection proofs remain unchanged because
  no inverse length arithmetic is selected for encoded String.

## Parent evidence so far

Actual CLI generation and mounted HTTP execution in
`/tmp/loom-686-parent-wire-module.log` reproduce four disagreements: encoded
minimum rejects a valid value, encoded maximum accepts an oversized value, exact
source enum rejects its own member, and the default is double-encoded to
`"YUdrPQ=="`. The carrier-neutral mounted test also checks service invocation.
`/tmp/loom-686-parent-probe.log` records the direct `Pattern` DSL failure on Bytes.
The mounted generated module was reused after correcting only its authored test
assertion from case-sensitive struct JSON names to map lookup; generated code
was unchanged. This is not repeated generation or determinism evidence.


Readiness additions are limited to three Go test files, the existing normalization
Lean source/README/theorem manifest, and this plan. The direct binary matrix has
seven current failures (JSON root/field/array/map, query, response header, and a
shared raw/JSON reference) and two passing actual-context controls (multipart part,
raw response). Analyze and direct Render assertions are independent. The format
table has six proposed accepted strings and eight rejected strings, all through
the existing public format API; unsupported-format errors do not count as correct
invalid-format rejection. `/tmp/loom-686-admission-parent.log` records these runs.
The format grammar now matches the user’s accepted policy.

The extended existing Lean model builds successfully (`/tmp/loom-686-model.log`).
It checks the current length/literal failures before the candidate, exhibits enum
widening after inverse-length translation, and establishes a general non-injective
decoder limitation. It assumes correct extraction/grammar and does not establish
arbitrary parser correctness or transport generation.

Final acceptance after readiness approval will extend the existing mounted
specimen with empty/padded/invalid/alias strings, exact enum exclusion, encoded
pattern, nullable/reference/array/map locations, plus ordinary String controls.
Rejections assert status, problem code/detail and no service invocation; successes
assert the service value and re-encoded wire spelling. Roundtrip spec assertions
cover source bounds, default, enum and explicit example. Existing format and raw
media tests cover shared JSONShape admission, DSL validation/constant generation,
format example synthesis, and retained raw/multipart semantics. The format’s
example generator must honor ordinary finite bounds and conjoined constraints;
it may decline synthesis when the existing generator cannot find a witness, but
must never publish an invalid example. Source schemas with impossible constraints
retain existing admission semantics; this repair will not invent a new solver.


### Implementation boundary proposed for review

Replace the special cases in `stringSchemaExpression`, `schemaAttributeBlock`
and `hasSchemaBlock` with one small string-format representation record carrying
the DSL carrier, optional standard format validator and raw-binary role. Unknown
formats keep the existing lossy diagnostic/plain String behavior. Known ordinary
formats keep their existing Format expressions. The planner uses the same record
to distinguish raw-only binary from encoded byte strings; renderers consume it
without independently deciding that both formats mean Bytes.

Location admission starts at the existing parameter/header, request content and
response content entry points. Recursion follows properties, items, map values,
composition and local references, carrying an occurrence role. Whole multipart
properties may be raw parts; their nested object/array children return to encoded
value roles. Ordinary raw response schema roots permit raw binary. Declaration
scanning cannot assume a wire codec, and repeated references must be keyed by
role or use a recursion-path guard, never a type-global visited cache. This is an
additional obligation of the existing shared planner, not renderer-specific
binary checks or a generic new encoding framework.

Latest exact parent run `/tmp/loom-686-parent-ready.log` regenerates the mounted
specimen from the current readiness source: all four runtime regressions and the
separate encoded-pattern DSL regression fail (8.251s total). Its 16 actual generated
artifact hashes are in `/tmp/loom-686-parent-artifacts.json`. The exported schema
also double-encodes the enum, default and explicit example to `"YUdrPQ=="` while
retaining encoded min/max bounds, independently demonstrating the incomplete
translation. Existing multipart/form/raw and ordinary URI/runtime format controls
pass (importer 3.715s, runtime 0.152s). Seven added Lean theorems build; six use no
axioms and the general non-injective theorem uses only standard `propext`.

## Candidate validation and intended output changes

The shared `FormatBase64` follows the accepted standard alphabet/padding grammar,
without whitespace and with tolerant pad bits. Runtime and JSONShape admission,
DSL/expr validation, generated validation constants, and bounded example synthesis
consume this format. `stringRepresentation` selects the importer carrier and
validator once; rendering, authored-example checks and raw-location admission
consume it. The old independent string-carrier and format lookup paths are gone.
`schemaEncoding` follows actual parameter/header/content occurrences with a
path-local `(schema, role)` guard, so permission from a raw use cannot admit a
second JSON use of the same declaration. Multipart descendants return to encoded
roles. Existing media admission still rejects whole raw request bodies and
unsupported custom multipart encodings.

The expanded existing mounted specimen passes 26 cases: encoded bounds, exact
enum spelling, missing/default/supplied-empty values, padded/malformed text,
nullable values, referenced arrays/maps, conjoined patterns and ordinary strings.
Rejected input checks HTTP status, problem code, the existing safe detail and no
service invocation. Accepted input checks the actual supplied service values and
re-encoded spelling. Ten binary-location rejections and two raw/multipart controls
pass, including aliases and nested multipart children. Runtime/JSONShape's14
format cases and six example-length/allocation controls pass. The unknown-format
and unknown-example-format parent failures were recorded before their repairs.

Bounded actual CLI comparison uses the original byte bounds/literals specimen,
the existing URI-reference specimen, and the existing multipart/form/raw-media
specimen. Parent generation uses an exact Git archive of `53849ce33` and a CLI
built from that archive; candidate and repeat use the actual changed worktree.
All six distinct parent/candidate outputs compile and pass vet. Repeat output
matches all48 candidate artifacts exactly. All32 ordinary/raw artifacts also
match their parent byte for byte. The common byte specimen changes exactly:

- `gen/encoded/service.go`: decoded byte carriers become native/pointer strings.
- `gen/http/cli/encoded/cli.go`: valid encoded-string examples replace byte-based
  and double-encoded examples.
- `gen/http/encoded/client/cli.go`: CLI carriers, exact enum/format/length checks,
  native defaults and examples follow encoded String.
- `gen/http/encoded/client/types.go` and `server/types.go`: body carriers,
  constructors, defaults and validators follow that same String representation.
- `gen/http/openapi.json` and `openapi.yaml`: default/enum/example literals retain
  their source spelling; type, format and authored bounds remain string-domain.
- `gen/loom.json`: recorded type/validation metadata follows the corrected carrier.

The other eight common artifacts stay exact. The expanded mounted specimen's16
artifacts also match a second independent CLI generation exactly; identical repeat
outputs were not rebuilt. Its26-case race run passes1.562s. Direct shared-format
race checks pass pkg1.491s, expr1.373s, DSL1.274s, codegen1.411s. Existing encoded
format/raw controls pass7.285s. Exported JSON/YAML specs pass libopenapi, with exact
bounds/default/enum/example assertions, and all four changed-spec Vacuum runs
report zero errors under the pinned rules. Scoped lint reports zero issues;
gofmt and JSON-v2 gates pass. The existing Lean proof evidence remains valid:
no proof source changed after readiness, and implementation tests check the
assumed extraction/grammar/occurrence boundaries.

Evidence: `/tmp/loom-686-cli/{commands,comparison,artifacts}.json`,
`/tmp/loom-686-common-output.patch`, `/tmp/loom-686-expanded-repeat/receipt.json`,
`/tmp/loom-686-expanded-wire-final.log`, `/tmp/loom-686-shared-final.log`,
`/tmp/loom-686-representation-controls.log`, `/tmp/loom-686-spec.log`,
`/tmp/loom-686-vacuum.json`, and `/tmp/loom-686-lint-final-tests.log`.
The earlier expanded test incorrectly expected a field name in the public safe
problem detail; its failed run is not acceptance evidence. The corrected assertion
preserves the established generic validation detail and exact public error code.
No full importer/corpus sweep or dependency-race claim is made; known #820 remains
outside this repair. No checked-in transport fixture declares the new format, so
unchanged transport fixtures do not require regeneration.
