# Header collection physical representation (#895)

Status: implementation and bounded validation complete; exact final review pending. This task follows delivered
#894, which preserves constructor nominal identity. The collection framing policy
in `docs/http-guide.md` is already accepted and is unchanged.

## Contract and owner

A header element has both a semantic primitive kind and a physical Go carrier.
`ArrayOf(Type("Label", String))` is a string list on the wire but its service value
is `[]Label`, not `[]string`. The shared encoder must convert each element using
its actual carrier before passing native strings to `http.SetHeaderValues`.
Nil remains absent; nonnil empty remains a present empty header. Existing grammar,
required/default admission, error redaction and before-write rejection remain.

Response `AttributeData` already retains normalized `Type` and physical
`FieldType`. The parent `header_conversion` discarded the latter and returned a semantic
string array unchanged. The parent request encoder separately considered element aliases.
JSON-RPC response encoding reuses HTTP response sections, so the repair belongs to
HTTP codegen's shared typed-to-wire conversion, not a JSON-RPC template patch or
whole-slice cast. Decode constructors already use the canonical scope fixed by
#894; their typed reconstruction remains a bounded control.

## Sequence

1. [x] Preserve the original ordinary `Label` failure and native control from
   `/tmp/loom-894-header-parent`, including the exact CLI input and build output.
2. [x] Reproduce on the delivered parent and add focused representation regressions
   for string and non-string aliases, named collections, and located elements.
   Use existing HTTP and JSON-RPC module renderers and header fixtures.
3. [x] Freeze readiness: identify the shared scalar/list conversion input contract,
   actual consumers and bounded acceptance; obtain independent review.
4. [x] Implement one typed conversion owner consumed by request and response
   encoders. Remove superseded array/string shortcuts and formatting decisions.
   Keep framing, defaults, body conversion and type-name allocation with their
   existing owners. No new exported API or compatibility path is needed.
5. [x] Verify direct regressions, generated compilation and mounted list behavior;
   compare actual parent/candidate/repeat outputs and affected existing goldens.
6. [ ] Freeze exact diff/evidence, obtain independent final review, integrate and
   deliver as one atomic ticket, then preserve inputs and clean temporary outputs.

## Bounded acceptance

- Native string/numeric lists remain valid controls; named string and non-string
  elements and a named outer list generate legal casts and compile. A located
  element retains its package identity without allocating a new type name.
- HTTP request and response and JSON-RPC response use the same conversion;
  independent decoder constructors reconstruct typed service collections.
- Reuse header collection wire/shapes/RPC fixtures for nil, present empty, ordinary
  values, defaults only on absence, required empties, repeated/comma-list decoding,
  views/errors and rejected comma/empty/edge-whitespace items. Add only cases whose
  named carriers are not already represented there.
- Preserve scalar header/cookie and query formatting where shared scalar logic is
  reused; list framing and URL escaping remain separate owners.
- Actual parent/candidate generations establish intended output changes;
  independent repeated generation establishes byte determinism. Compile/vet the
  distinct generated modules once. Regenerate only affected snapshots.

This is a pure representation conversion, not a new concurrent or protocol-state
machine. Existing header-list runtime tests establish framing and error ordering;
compiled carrier regressions establish Go assignability. Add no separate state
model unless implementation reveals a new ordering or ownership obligation.

## Readiness evidence and proposed conversion contract

Parent `2a98e412322b411b4ba01c9edfe0a07f91ec53b2` includes #894. The new
HTTP module has one native passing case and one named failing case: named string,
bool, int64, uint64, float64, named outer list, and located string all fail in
response encoding. Named bytes already convert correctly and remain a control.
The HTTP named request client compiles independently. The JSON-RPC named string
and int64 response module fails at the same two conversion operations. These
failures are retained in `/tmp/loom-895-parent-regressions.log`; the original
minimal actual-CLI report remains in `/tmp/loom-header-alias-array-report.md`.
The initial test-authoring `Uint64` spelling error was corrected to documented
`UInt64` before this final evidence; it is not a product failure.

`TestHeaderCollectionPhysicalMetadata` checks the existing analysis seam in both
directions: every wire element is primitive and every actual declared element is
named. No new metadata or allocator is needed. The owner will accept normalized
semantic type, actual field type, dereferenced source expression, destination
local names and declaration/assignment needs. One private scalar formatter
converts the physical value into the native argument expected by `strconv` or
string conversion. One list renderer applies that formatter per element and
preserves nil versus nonnil empty; native-string direct assignment is legal only
when physical element assignability permits it. Never cast a whole `[]Named`
collection to `[]string`, and never infer a physical element from semantic kind.

Both request and response encoding derive from this owner. Remove their competing
list conversion branches. Response defaults pass their actual native literal
carrier rather than the named service carrier; scalar optional dereferencing
remains explicit at the caller. Reuse the same scalar formatter for the existing
client scalar conversion adapter (including query/cookies), replacing duplicated
format switches instead of adding another primitive policy. URL escaping, bearer
framing, custom text codecs, list validation and header writing remain outside
this formatter. JSON-RPC inherits the response encoder unchanged.

Final acceptance extends existing wire fixtures only with named carriers and
nil/empty/ordinary/rejected values. Existing native shapes/default/view/error,
scalar header/cookie and query controls guard adapter composition. Compare the
three focused modules and affected owner snapshots at actual parent and candidate;
repeat generation independently. Do not regenerate unrelated SSE applications or
expand the full exported-design corpus for this bounded representation repair.

## Implementation and focused checks

Root approved readiness in `/tmp/loom-895-root-readiness-verdict.json`. The private
`transportScalarValue` formatter preserves physical element information;
`renderHeaderConversion` supplies nil-preserving scalar/list lowering. Both header
encoders use it, and the existing client scalar adapter uses the same formatter.
The old response conversion partial, request list loop and unused `EncodedSlice`
allocation were removed. No runtime header codec changed.

All original named HTTP/JSON-RPC compilation failures pass. Existing header wire,
shape/default/view/error, projected carrier, JSON-RPC metadata and scalar
header/cookie tests pass (HTTP 27.593s, RPC 5.621s). Added mounted named-carrier
checks reuse the same generated modules for absence, empty and populated values;
HTTP also rejects unrepresentable named strings before changing their header.
The original prototype whitespace drift in scalar query/cookie templates was
removed; their existing owner snapshots remain exact. Header snapshots alone
change: native request lists use the shared renderer, numeric response lists use
one nil-preserving destination and optional scalar dereference occurs once before
formatting. Named element casts and removal of illegal whole-slice casts are the
intended generated behavior changes.

## Final acceptance evidence

The final focused race run passes HTTP codegen (35.025s) and JSON-RPC codegen
(8.694s). It covers physical analysis, all three generated modules with mounted
named-carrier round trips and before-write rejection, existing header wire/shapes,
projected headers, RPC headers, scalar header/cookie values, both encoder snapshot
owners and shared lexical reservations. Each generated module runs `go test -race`
and `go vet`; repeated identical output is not rebuilt. Scoped golangci-lint
reports zero issues. Restoring the existing conditional cookie field selector
preserves its previous caller contract; the affected scalar wire and encoder
snapshot checks pass again (11.960s), without a header output change.

Actual parent, candidate and independent repeated generation cover 38 files in
three modules. Every candidate file matches the repeat exactly; 33 match the
parent exactly. The five intended changed files are:

- HTTP native and named `gen/http/headers/client/encode_decode.go`: the shared
  list renderer replaces the request-only loop; native strings avoid allocation.
- HTTP native and named `gen/http/headers/server/encode_decode.go`: shared
  nil-preserving conversion replaces the response-only loop, named elements gain
  primitive casts and named outer collections no longer use whole-slice casts.
- JSON-RPC `gen/jsonrpc/headers/server/encode_decode.go`: the same shared response
  conversion casts named string and int64 elements.

The exact path/hash ledger and generated diff are
`/tmp/loom-895-final-artifact-comparison.json` and
`/tmp/loom-895-final-artifact-delta.patch`. Twenty-eight owning header snapshots
change (six client, twenty-two server); query and cookie snapshots stay unchanged.
The existing non-header application fixtures are outside this bounded encoder
scope. No schema output or OpenAPI contract changes, so no new spec validation
is needed. Source and command hashes are frozen in the final manifest for review;
delivery and cleanup remain pending.
