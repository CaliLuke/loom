# Untagged JSON matching

`UntaggedJSON.tla` models the orchestration in `pkg/json_union.go` and the
transactional adapters emitted by `codegen/unionjson`. It supplements the
[value-contract model](../../../expr/lean/value_projection/README.md): schema
membership and concrete Go decoding are independent predicates over one wire
value. Both must identify the same single branch; emission must retain the
selected branch, and failure must leave the destination unchanged.

Use TLC 1.7.4 (the repository's pinned TLA+ tools version), from this directory:

```sh
java -cp "$TLA2TOOLS_JAR" tlc2.TLC -config Legacy.cfg -metadir /tmp/loom-350-legacy UntaggedJSON.tla
java -cp "$TLA2TOOLS_JAR" tlc2.TLC -config Strict.cfg -metadir /tmp/loom-350-strict UntaggedJSON.tla
```

The legacy control must exit 12 with an `EmissionRetainsIdentity` counterexample:
a selected decodable branch can emit without a unique matching schema. The strict
configuration must pass all three invariants. Checked with TLC 1.7.4 during #350:
legacy failed as intended (69 states generated); strict passed (128 distinct states).

The bounded model abstracts two branches as schema and decoder match sets and an
unexpected-failure flag. It includes zero/multiple matches, different unique
identities, and failures despite an otherwise valid match. Set membership makes
order irrelevant in the model; reversed candidate tests check implementation
ordering. The initial destination sentinel represents an arbitrary preexisting
value. Predicate extraction, JSON syntax, recursive constraints, numeric
representability and codec classification are assumptions, not proved here.

Correspondence checks:

- `pkg/json_union_test.go`: order reversal, independent signed/unsigned schema
  matches, unique-identity disagreement, and unexpected errors after success.
- `expr/value_plan_json_shape_test.go`: lowered matching agrees with the captured
  schema plan for inherited constraints, bytes, closed members and nulls.
- `http/codegen/untagged_arrays_test.go`: compiled encoding/decoding, nil/empty
  arrays, invalid nested elements, ambiguous arrays, unchanged destinations,
  both HTTP directions and required/optional request error contracts.
- `jsonrpc/codegen/untagged_arrays_test.go`: typed arrays and objects through the
  existing JSON-RPC envelope.

The model does not prove Go refinement, unlimited recursion, arbitrary codecs,
or full OpenAPI equivalence. Opaque codecs are rejected by DSL validation. Keep
checker state, traces and logs outside the repository and remove task-owned
outputs after validation and independent review.

## Nullable enum membership (#1037)

`NullableEnum.tla` models a record field's missing, null, enum-member and
non-member states. Requiredness and occurrence/inherited nullability vary
independently. `Nullable()` adds null to the value domain; enum predicates
apply to concrete values. This is distinct from Any's intrinsic admission of
null, which does not bypass authored enum membership.

Use the pinned TLC command above with `NullableEnum.tla` and the
`NullableLegacy.cfg`, `NullableLocalOnly.cfg` and `NullableChecked.cfg`
configurations, each with its own temporary state directory. Legacy reproduces
enum rejection of an admitted null. Local-only preserves direct nullability
but loses named-type nullability. Checked retains both and leaves required
absence and concrete non-member rejection intact.
With Java 25.0.5 and TLC 1.7.4, both negative controls fail `Agreement` after
42 distinct states; checked passes all 64 states. The implementation comparison
uses base `a2df44b82` and Go 1.27.3. The initial local-only repair failed the
inherited-null direct test before effective nullability was added to lowering.

`schemaJSON` uses the captured attribute's `IsNullable` fact after syntax/body
admission; `lowerJSONShape` puts that effective fact in the runtime descriptor.
`JSONShape.match` accepts admitted null before value predicates, while `Match`
still validates the full description first. `TestNullableEnumMatchingAgreement`
compares prepared and lowered matching with rendered schemas for primitive,
inherited, inherited-with-occurrence-refinement, conjoined and array enums.
Review exposed a separate rendering gap: an occurrence enum outside a nullable
named schema removed null. Inline JSON Schema and OpenAPI now wrap the complete
refinement in the nullable domain. Direct IR checks and the rendered 3.1/3.2
specimen verify null/open acceptance and closed/other/missing rejection.
The extended untagged-array fixtures check
ordinary required record validation, union marshal/unmarshal, HTTP and JSON-RPC
round trips, and invalid concrete values. Runtime tests retain ambiguous-null
rejection and fatal malformed-description errors.

The bounded before/after generation comparison uses the six designs in
`generationDeterminismDSLs` with identical fixture inputs on both revisions.
Intended differences are inherited `Nullable: true` descriptors in the service
and both HTTP union codecs, and null-preserving refinement wrappers in the
OpenAPI JSON/YAML specimen. The five control designs remain byte-identical.
The generated HTTP fixture compiles and runs; independent-process generation
checks exact bytes across five runs.

The model assumes the nullability query, JSON parser and enum evaluator; it does
not prove arbitrary alias graphs, Go codecs or JSON Schema rendering. Those
boundaries remain direct and generated-test obligations, and the original
union uniqueness/transaction model remains separate.

## Map field membership (#1041)

`MapMembership.tla` checks universal value membership together with field
presence, map nullability and entry-count minimums. It compares universal
membership against an existential candidate: accepting any matching member
incorrectly admits a mixed valid/invalid map and rejects the unconstrained empty
map. Run TLC 1.7.4 with `MapChecked.cfg` and `MapExistential.cfg`, each with its
own temporary `-metadir`, using the Java invocation above and `MapMembership.tla`.
Checked passes 384 distinct states; existential fails `Agreement` after 297
states. This is a capability extension; the negative candidate is a design
alternative, not a previously shipped implementation.

The bounded abstraction assumes correct per-value matching and counts at most
one representative valid and invalid entry. It does not prove arbitrary
recursion, JSON parsing, aliases, Go validation or schema rendering. The existing
strict orchestration model still owns outer uniqueness, identity and transactional
assignment. Map traversal must not weaken those obligations.

Correspondence tests include `pkg/json_shape_map_test.go` (empty, invalid member,
nullability, aliases, duplicates and recursive values),
`expr/value_plan_json_shape_test.go` (prepared/lowered predicate agreement),
`codegen/unionjson/union_test.go` (member policies beneath maps), and the existing
HTTP/JSON-RPC generated fixtures (typed round trips and malformed values).
The HTTP fixture additionally exercises named map validation, nested maps and
arrays, recursive values, retained destinations and ambiguous map-bearing objects.
OpenAPI 3.1/3.2 specimen checks retain value schemas and entry-count bounds.

The before/after comparison uses base `c74af637189d1af43a5c352fb8ce0aee6af49225`
and Go 1.27.3. All six `generationDeterminismDSLs` with the original fixture
inputs remain byte-identical. With the extended map fixture, only
`UntaggedArraysDSL` changes: service types/validators/descriptors; HTTP client
and server body conversion helpers, validators and union descriptors; CLI
examples; and OpenAPI JSON/YAML map components/properties/examples. Those are
the intended differences. The five other designs remain exact-byte controls.
Five process-isolated generations of the extended inputs agree byte for byte.
No checked-in integration fixture declares `Untagged`; none needs regeneration.

## Nested untagged unions (#1036)

`NestedMembership.tla` models up to two rows, each with zero, one or two matching
inner branches. `NestedChecked.cfg` requires exactly one match per row and passes
42 states. `NestedAny.cfg` fails `Agreement` after 26 states because an ambiguous
row would be admitted. Empty rows are accepted by universal membership; collection
length constraints remain independent predicates.

`NestedFailure.tla` models an inner failure while another outer branch is the
unique schema and Go match. `NestedFailureChecked.cfg` distinguishes an ordinary
contract mismatch from an unexpected failure and passes four states.
`NestedFailureUnclassified.cfg` fails after three states: treating all inner errors
as fatal blocks the valid alternative. These are design candidates for a new
capability; the previous DSL rejected nested unions before code generation.
The direct decoder regression nevertheless reproduces the unclassified error at
the runtime seam before the implementation change.

Run TLC 1.7.4 with the invocation above, each corresponding `.tla`/`.cfg` pair
and a separate temporary `-metadir`. Both models assume accurate branch
predicates and error classification. They do not prove Go codecs, arbitrary
recursion, JSON syntax or policy extraction. Keep proof execution opt-in.

`pkg/json_shape_union_test.go` covers exact-one nesting, empty and mixed arrays,
map values, malformed descriptors, recursive objects and unconsumed cycles.
`pkg/json_union_nested_test.go` covers wrapped mismatch classification, candidate
order and unexpected error preservation. Prepared/lowered agreement and member
policy projection are checked in `expr` and `codegen/unionjson`. The existing
HTTP specimen adds nested arrays/maps, nullable unions, recursive members,
ambiguous nested objects and the distinct valid outer alternative. JSON-RPC
checks typed round trips and invalid-row error envelopes without service calls.

Verification uses Go 1.27.3 and base
`78236cdcad977f296c80d9eef9907f5f0d3b75ea`. The six existing determinism designs
with unchanged inputs remain byte-identical. The extended `UntaggedArraysDSL`
changes only its generated service/client/endpoints, HTTP routing and CLI files,
body conversion helpers/validators/union descriptors, and OpenAPI JSON/YAML.
Those changes add nested Entry alternatives, recursive Group members and the
Fallback endpoint used to check candidate rejection under HTTP closed-object
policy. The other five designs remain byte-identical. Five independent processes
produce identical raw sections and rendered files. Generated HTTP and JSON-RPC
modules compile and run; the 3.1/3.2 OpenAPI specimens pass parsing and pinned
Vacuum lint. Direct projection/matching checks pass with the race detector.
No checked-in integration fixture declares `Untagged`; none needs regeneration.

## Text-location unions (#1014 design)

`UntaggedText.tla` checks the proposed branch-indexed text matcher for
parameters, headers, cookies and form fields. Each branch owns a text codec
resolved by the shared `internal/httpencoding` owner (#968).

Matching keeps **two accounting domains**, like `MatchUntaggedJSON`:

- *schema*: the branch codec parses the text into the JSON data model, and the
  projected schema validates it, with no Go widths;
- *carrier*: the typed decode, followed by validation.

The server accepts only one schema match and one carrier match, on the same
branch. It writes the destination only then. The client encodes its selected
branch and runs the same matcher over its own text before publishing. OAS 3.2
§4.24.4.2 forbids resolving ambiguities, so the matcher counts instead of
choosing.

Model data:

- **Branch kinds.** An unconstrained string, a `^[a-z]+$` string, `Int`,
  `Int32` and an unexploded integer list.
- **Texts.** `a`, `5`, `6`, `5,6`, the empty string, `big` (2147483648, which
  `Int` holds and `Int32` does not) and `?`, which no constrained branch
  decodes.
- **Values.** Canonical string tokens, because TLC compares one value kind.
- **Peers.** An `Inject` action lets an arbitrary peer send any text.

Run each configuration with its own state directory:

```sh
java -cp "$TLA2TOOLS_JAR" tlc2.TLC -workers 1 -config TextChecked.cfg -metadir "$TMPDIR/untagged-text-checked" UntaggedText.tla
```

Observed with Java 25.0.5 and TLC 1.7.4 (`tla2tools` sha256
`936a262061c914694dfd669a543be24573c45d5aa0ff20a8b96b23d01e050e88`). Checked
configurations verify `ServerSound`, `ServerComplete`, `Agreement`,
`NoAmbiguousSend`, `Transactional` (destination equals the sentinel in every
state until acceptance) and `ZeroMatchRejected`.

| Configuration | Expected and observed |
|---|---|
| `TextChecked` (string, int, list) | pass, 114 distinct states |
| `TextCheckedPattern` (pattern string, int, list) | pass, 92 distinct states |
| `TextCheckedWidths` (`Int`, `Int32`, pattern) | pass, 70 distinct states |
| `TextCheckedReversed` (reversed branch order) | pass, 114 distinct states |
| `TextFirstMatch` | exit 12, `ServerSound`: injected `5` accepted as the first of three matching branches |
| `TextNoRoundtrip` | exit 12, `NoAmbiguousSend`: client published string `"5"` |
| `TextCoerce` | exit 12, `ServerSound`: JSON-coerced `5` accepted as integer although three text decoders accept it |
| `TextEager` | exit 12, `Transactional`: a candidate wrote the destination mid-loop for input `a` |
| `TextCarrierOnly` | exit 12, `ServerSound`: `big` accepted as `Int` although `Int` and `Int32` both match in the schema domain |
| `TextZeroWitness` | exit 12 by design: `NoZeroMatchWitness` shows `?` reaches a zero-match decode (then `ZeroMatchRejected` holds) |
| `TextWidthWitness` | exit 12 by design: `NoWidthSplitWitness` shows `big` splits the two domains |

What can be sent in each profile:

- Overlapping profile: only non-numeric strings and non-singleton lists.
- Pattern profile: `a`, `[]` and `[5,6]`. `5` stays ambiguous between the
  integer and the one-element list.
- Widths profile: only `a`.

Generated errors must name the matched counts.

`UntaggedPresence.tla` checks union presence in two contexts with different
owners.

**Parameter context** (query namespace, #968):

- Branch A is a scalar `id`.
- Branch B is a form-exploded object whose optional properties are top-level
  keys. The sibling key `z` is ignored.
- A branch with none of its keys present was not sent.
- Rule A sends an optional empty object as absence and rejects a required one
  before publication.

**Body context** (form or multipart root, media owner):

- Presence is body presence, not key presence.
- Branch A is an open object requiring `x`; branch B is a closed empty object.
- A present empty body is a value.
- For optional bodies, `OptionalEmptyBody` gives the media owner's rule for a
  zero-length body: `absent` for form, `value` for an empty multipart envelope.

Requests and results are records, for well-typed comparison. The invariants are
`Agreement`, `NoFalseRejection`, `UnspellableRejected` and `EmptyBodyIsValue`.

| Configuration | Expected and observed |
|---|---|
| `PresenceChecked` | pass, 162 distinct states |
| `BodyCheckedEmptyValue` | pass, 55 distinct states |
| `BodyCheckedEmptyAbsent` | pass, 55 distinct states |
| `PresenceEmptyIsValue` | exit 12, `NoFalseRejection`: `A` unsendable because the empty object also matched `{id}` |
| `PresenceNoRuleA` | exit 12, `NoFalseRejection`: optional empty object refused instead of sent as absence |
| `BodyParamRule` | exit 12, `NoFalseRejection`: the parameter not-sent rule makes the closed-empty body branch unsendable |

`BodyKeyCount` reproduces the ordinary form decoder's rejected key-count policy:
nonempty `&&` is present raw input with no keys, but the decoder reports absence.
It fails `EmptyBodyIsValue` (31 distinct states); `BodyCheckedEmptyAbsent` passes
all four invariants (55 states). The same key-count policy also refuses a required
encoded empty object. Presence is resolved by the media owner before this model's
body decoder; parsing, Go pointer allocation and error formatting are outside its
scope. `renderFormRootRequestDecode` now handles ordinary, map and union forms
through `ParseRawFormWithLimit`: required forms decode the complete value set;
optional forms consult `RawPairs.HasInput`. `TestOrdinaryFormBodyPresence` and the
maintained `UntaggedFormDSL` harness (`TestOrdinaryFormPresence` and
`TestOrdinaryFormClientRoundTrip`) check required/optional roots, selected native
pointers, byte-empty versus separators-only input, field validation, errors,
limits, service invocation and client replay. Direct optional roots retain their
native nonnil empty-object representation; selected optional object members retain
nil absence. Nullable form roots remain inadmissible.

Run the existing documented TLA command with `-config BodyKeyCount.cfg` to reproduce
the negative control, or `BodyCheckedEmptyAbsent.cfg` for the supported policy.

Expression/transport correspondence also covers standalone empty object bodies:
`TestHTTPEmptyMediaBody` and `TestEmptyMediaBodyPresence` ensure explicit form or
multipart ownership retains the empty object and required/optional body facts.
`httpRequestBody` must not collapse that declared body into the no-body sentinel;
`validateMissingPayload` distinguishes it from an absent payload. Ordinary
bodyless endpoints keep their existing contract. These direct checks establish
capture and presence facts, while generated transport tests remain responsible
for the media-specific empty wire spelling described above.

`UntaggedClosure.tla` checks `unevaluatedProperties: false` beside a `oneOf`,
following JSON Schema 2020-12 Core §11.3. Count the original branches, then
require the instance members to be within the selected branch's evaluated
annotations:

- an object with absent `additionalProperties` evaluates its declared members;
  with `additionalProperties` present, every member;
- a map evaluates every member;
- `Any` evaluates none;
- a nested union defers to its selected alternative.

Members are `kind`, `x`, `y` and `z`; scalar instances bypass closure.

| Configuration | Expected and observed |
|---|---|
| `ClosureChecked{OpenPair,Discriminated,Map,Any,Nested,Additional}` | pass, 68 distinct states each |
| `ClosureCloseFirstOpenPair` | exit 12, `Agreement`: `{x}` matches open `A` and `B`, but closing `B` first accepts `A` |
| `ClosureCloseFirstDiscriminated` | exit 12, `Agreement`: `{kind, x}` accepted after closing first (#1005 shape) |
| `ClosureCloseFirstAny` | exit 12, `Agreement`: closing `Any` first rejects a scalar the source accepts |
| `ClosureIgnoredNested` | exit 12, `Agreement`: the generated text matcher previously ignored outer closure and accepted an unevaluated member (58 generated/distinct states) |
| `ClosureNamesListMap` | exit 12, `Agreement`: a map member treated as unevaluated |
| `ClosureNamesListNested` | exit 12, `Agreement`: the union of inner names admits `y`, which the selected alternative does not evaluate |
| `ClosureNamesListAdditional` | exit 12, `Agreement`: an `additionalProperties` member treated as unevaluated |

Assumptions and bounds:

- **Codecs.** Each branch codec is injective on its representable domain, and
  `Decode` inverts `Encode` there. This is a #968 obligation; the `ASSUME`
  checks it only over the modeled tables.
- **Validation.** It is part of the carrier predicate. The schema predicate is
  the projected-shape result for the codec's data-model value.
- **Scope.** One text value per binding. Repeated keys, percent-encoding,
  delimiter escaping, path raw capture and header line joining are not
  modeled. They are #968 codec obligations.
- **Order.** Branch order is fixed per configuration, and two orders are
  checked. Counting has no order dependence by construction.
- **Fatal errors.** Fatal callback errors (evaluator or configuration) are
  outside the model and keep the existing `pkg` classification.
- **Closure.** It is modeled for one instance location, with valid member
  values and bounded nesting (one nested union level). Nested object fields
  are outside closure, by definition.
- **What is not proved.** The models do not prove Go codecs, JSON parsing,
  `JSONShape` annotation output or the generated client and server code.

Implementation correspondence, when implemented:

- `internal/unionmatch.Select`, shared by `loom.MatchUntaggedJSON` and `loomhttp.MatchUntaggedText`, with
  order-reversal, ambiguity, width split, zero-match, fatal-error and
  unchanged-destination tests;
- `JSONShape` evaluated-member output and union closure checks in both
  decoders;
- generated client encoders, with the re-decode check before request
  publication;
- response-header encoders, with the check before commit;
- CLI flag decoding with the same matcher;
- mounted HTTP and JSON-RPC-over-HTTP tests for the #1014 wire and body
  presence tables.


The shared text matcher now captures `TextUnionAnnotations` from the same
codec-owned view used for schema matching. It carries both present members and
schema-evaluated members, applies closure after exact-one selection, and retains
nested annotations once per input. `ComposeTextUnion` counts original inner
branches before its own closure; an ordinary inner closure mismatch can leave
an enclosing alternative eligible. It never filters individual leaves by the
enclosing closure before counting.

`TextBranch.Annotations` derives logical members from its decoded `ParamValue`,
not the whole query/header/cookie namespace. Fixed exploded objects keep their
existing allowlist; dynamic maps read other alternatives' keys but exclude real
sibling bindings. Root form/multipart adapters supply their whole-body instance
through the same callback. The callback interface does not apply parameter
presence rules to root bodies.

`TestTextBranchAnnotationsUseCodecOwnedView` checks fixed, dynamic and deep
namespaces. `TestNestedTextUnionClosureOrdering` checks ambiguity before closure,
inner rejection with an outer fallback, and outer annotation consumption.
`TestComposeTextUnionRetainsPredicateResults` checks schema, decoder and
annotation callbacks each run once through 32 nested unions. Generated
`UntaggedLocationsDSL` covers closed objects/maps/nested unions in query, header,
cookie and path bindings plus client rechecks, CLI parsing and response headers.
The ignored and close-first model controls fail; checked nested closure passes
68 generated/distinct states. This does not change the #968 codec domain or
turn unknown parameter keys into authored object members.

These models are design evidence. They are not CI or default gates.

## Recursive annotation work

Independent review reproduced exponential closure work: the matcher selected a
branch and then matched it again to obtain evaluated members. A depth-20,
305-byte valid recursive object took 2.262s with closure versus 22.584 microseconds
without closure in the local probe. `ClosureWork.tla` abstracts one recursive
branch, two units of local matching/annotation work and one terminal visit.
`rematch` visits the child twice per level; `retained` visits it once and carries
the annotation result. The `LinearVisits` bound counts shape visits, not parsing
bytes, allocations, schema validation or arbitrary alternative graphs.

Run explicitly with Corretto 25.0.5 and TLC 1.7.4:

```sh
java -cp "$TLA2TOOLS_JAR" tlc2.TLC -workers 1 -config ClosureWorkRematch.cfg -metadir "$TMPDIR/closure-work-rematch" ClosureWork.tla
java -cp "$TLA2TOOLS_JAR" tlc2.TLC -workers 1 -config ClosureWorkRetained.cfg -metadir "$TMPDIR/closure-work-retained" ClosureWork.tla
```

On 2026-11-06, rematching violated the invariant after 2 generated/distinct
states; retaining annotations passed all 7 generated/distinct states through
depth 6. The corrected `JSONShape.matchAnnotated` returns evaluated members with
the match result; aliases and unions retain that result, and closure never
reselects. The same depth-20 probe took 36.458 microseconds after the fix. These
timings are observations, not test thresholds. `TestJSONShapeRecursiveClosure`
checks valid and rejected nested inputs plus annotation retention through depth
64. The model does not claim linear total JSON parsing cost: nested raw values
and distinct union alternatives still have their ordinary traversal cost.

## Nested text accounting (#1014)

`NestedText.tla` enumerates schema and decoder membership independently for
three leaves: two inside an inner union and one outer alternative. A nested
schema predicate requires one inner schema match; its decoder predicate requires
agreement of the inner schema and decoder identities. The outer selection uses
those two independently derived predicates. Flattening the leaves is a negative
control: an inner contract mismatch must not prevent a valid outer alternative.

Run TLC 1.7.4 explicitly using the invocation above and a separate temporary
`-metadir` for `NestedTextChecked.cfg` and `NestedTextFlatten.cfg`.
The composed strategy passes all 128 states (depth 2). Flattening violates
`Agreement` (94 states generated in the checked run). The model assumes accurate
leaf predicates; it does not model raw codec parsing, unbounded recursion,
closure annotations, allocation or custom application code.

`http.ComposeTextUnion` retains each predicate result for one immutable raw
input, preserves nested mismatch classification and assigns only a private
candidate after inner selection. `http/union_text_nested_test.go` checks nested
ambiguity, schema/carrier width disagreement, identity disagreement, unexpected
errors and one leaf evaluation through 32 nested levels. The generated text
location fixture checks the same composition at HTTP query boundaries. Shared
selection remains owned by `internal/unionmatch`; the generator retains a full
branch-index path to check nested identities before encoding publication.

### Singleton union cardinality (#1014)

A retained one-branch union still applies exact-one selection and occurrence
closure. `internal/unionmatch.Select`, `SelectSchema` and `JSONShape.validate`
accept a nonempty branch set; zero branches remain malformed. A singleton must
pass its schema predicate and typed decoder, with failures and selected identity
handled by the ordinary shared algorithm. Closure then consumes only that
passing branch's evaluated members: singleton Any accepts scalars, null and an
empty object but rejects a nonempty object; a matching map evaluates its members.
No branch is duplicated, discarded, or closed before matching.

`UntaggedClosure.tla` adds singleton scalar, object, Any, map and nested profiles.
`ClosureMinimumTwoSingleton.cfg` reproduces the previous blanket rejection and
fails Agreement after 35 states. `ClosureCheckedSingleton{Scalar,Object,Any,Map,Nested}`
pass 51 states each; the existing open-pair control still passes 68 states. Use the
existing explicit TLC command with each config and a separate temporary state
directory. Scalar abstracts non-object values (including arrays after their own
predicate); JSON syntax, null normalization, constraints, callback error classes
and Go refinement remain implementation-test obligations.

`internal/openapiimport/render_oneof.go` emits the function constructor for a
retained singleton. The public compact `OneOf(T1, T2, ...)` form still requires
two arguments. Constructor tagging comes from representation ownership: schema
oneOf uses Untagged; HTTP media alternatives retain their tagged representation.
Current import collapses a single media entry directly, so its tagged singleton
fallback is a normalized-model control rather than a new import behavior.

Direct selector/JSONShape tests cover empty configurations, singleton membership,
missing callbacks, fatal errors, mismatches, nesting and unconsumed cycles.
`oneof_singleton_closure_test.go` extends the maintained source/export closure
oracle with primitive, constrained primitive, promoted object, Any, array, map,
null-normalized and nested cases. Generated service codecs round-trip their
accepted values and reject invalid/unevaluated values; ordinary two-branch
closure profiles remain controls. Rendered JSON/YAML parse, generated modules
compile, and independent generation compares exact bytes.
