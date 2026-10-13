# gRPC binary metadata — #705

Implementation in progress against pushed `e631f0a89`, after independent readiness
approval and delivery of the separate named-element prerequisite #877.

1. [x] Trace metadata extraction, encoder/decoder consumers and response required-key contracts.
2. [x] Reproduce arbitrary-byte failures separately for request metadata, headers and trailers; retain explicit `-bin` and printable-byte controls.
3. [x] User accepted automatic standard `-bin` wire-key derivation for `Bytes`; preserve already suffixed keys.
4. [x] Establish one evaluated binding owner before validation/codegen, including alias/array shape, mapped names, collision admission and required-key consistency. Inspect actual consumers and reject per-renderer spelling rules.
5. [x] Independently review readiness and bounded acceptance, then implement the accepted owner.
6. [x] Compare bounded parent/candidate/repeat generation, compile and exercise mounted binary/ordinary/absence/default/required and independently implemented peer cases.
7. [ ] Independently review the exact final diff, commit, push and close.

The protocol defines binary metadata through `-bin` names and base64 wire values;
standard runtimes encode/decode those values. Source:
https://github.com/grpc/grpc/blob/master/doc/PROTOCOL-HTTP2.md . Plain keys accept
printable ASCII only. This is a cross-language protocol rule, not a Go-only codec.

`extractMetadata` currently passes the mapped element name unchanged into
`MetadataData.Name`. Both encoding and decoding use it. Response contract analysis
separately reads the evaluated mapped attributes, so changing the codegen name
alone would leave required-header/trailer checks inconsistent. Derived-name
collisions must be validated before generation rather than silently combining
unrelated fields. Existing absence/default and required admission rules remain.

Accepted contract: derive `x-raw-bin` from `Bytes` mapped to `x-raw`, and preserve
explicitly suffixed keys. Generated clients and servers carry arbitrary bytes
through the standard cross-language metadata protocol. Handwritten peers must use
the resulting wire key. Rejecting every unsuffixed binary mapping was considered
and not selected; it would require extra authoring without improving the wire
contract. Do not add a private encoding under ordinary keys. Reject collisions
between distinct fields after derivation before generating code.


## Original witness and controls

`TestBinaryMetadataRoundTrip` generates six bounded modules: request, header and
trailer placements, each using a plain key or an explicitly suffixed key. On the
published owner, all three plain-key cases fail for `00ff01` and a newline-bearing
value, while printable `hello` succeeds. All three explicit `x-raw-bin` cases pass
under the race detector. Generation and module compilation succeed for every case;
failures occur at the actual gRPC transport boundary. Combined generation/runtime
run: 12.927s. `/tmp/loom-705-parent.log` retains exact diagnostics and
`/tmp/loom-705-parent` retains generated input/output for readiness review.

The maintained #690 metadata-default model explicitly excludes binary-key framing.
Its defaults and required-before-default semantics are useful controls, not proof
of this repair. Bounded consumer inspection found gRPC registry designs using
Bytes in message fields rather than metadata, and the quality fixture uses text
metadata. No broad consumer-compatibility claim follows from that sample.

## Binding design model

`expr/tla/grpc_metadata_bindings` models current raw names, an encoder-only patch,
unchecked suffix derivation and the two candidate public policies. Current fails
binary framing in 58 distinct states; encoder-only fails consumer agreement in
58; unchecked derivation merges two fields in 71. Both a shared derived binding
with collision admission and an explicit-suffix admission owner pass (168
states generated/112 distinct each). This establishes why binding selection must
precede all consumers; it does not select the public policy.

## Expanded readiness on pushed71a0758f6

The accepted policy is unchanged. Root's original inputs were copied with exact
hash verification into `grpc-message-context`; source receipt is
`/tmp/loom-705-copied-readiness-inputs.json`. No production code has changed.

Sequence for the remaining readiness work:

1. [x] Compile explicit-bin native Bytes, named Bytes, native byte array, named
   byte array and named-element array before changing key semantics. Four controls
   compile; ArrayOf(named Bytes) fails client/server references and element
   conversions. Preserve as separate #877 prerequisite, not a #705 key fix.
2. [x] Add direct source-shape, placeholder-type, suffix/case, canonical-collision,
   channel-separation, implicit-credential and inherited-error-source witnesses.
3. [x] Extend the model with case folding, channel separation and late implicit
   bindings. Current, encoder-only, unchecked and auth-late alternatives fail;
   selected complete binding owner passes1008 distinct states.
4. [x] Complete independently implemented peer evidence and freeze owner/acceptance
   for independent readiness review. No production until that verdict.
5. [x] Implement only the approved binding owner, add remaining direct structural
   and bounded mounted absence/default/repeated-value controls, and review the
   final atomic packet. #877 remains separate.

### Selected expression owner

A private cached binding plan is entered from `GRPCEndpointExpr.Validate` after
Prepare composes response/error mappings. Resolve each logical member from its
actual payload/result/error declaration, not the local metadata placeholder.
Unwrap named scalar/one-level array storage to classify Bytes. Lowercase the wire
key, append `-bin` only when binary storage lacks it, and reject collisions within
one physical request/header/trailer list. Distinct fields in different channels
may reuse a key. Preserve already-suffixed text mappings and existing unsupported
shape/nullable admission; this is not a new codec facility.

The owner must compose implicit credentials before admission. Method finalization
has not yet inherited requirements, so use `MethodExpr.EffectiveRequirements`.
Move/reuse the existing security-placement algorithm with that effective input;
do not duplicate it in validation. Retain explicit credential placements,
Scheme.Name/In and requiredness. Finalize consumes the admitted map and must not
add another unvalidated credential binding. Validate unavailable source fields
without panicking; publication cannot depend on successful generation.

Publish fresh detached mapped attributes to keep forward/reverse identity
consistent. Calling Map with a new key on an existing mapping leaves the old
reverse alias; do not fix that generic API incidentally. Logical source names,
message exclusion, Go-field metadata and default/validation inheritance retain
existing owners. Error metadata resolves through `GRPCErrorExpr.sourceError`,
including service/API mapping provenance even when a method overrides its error.

Codegen continues consuming ElemName in `extractMetadata` and response-contract
collection. No suffix fallback belongs in encoding, decoding or contract code.
Scheme metadata and required-key diagnostics must agree with those same names.

### Bounded acceptance

Reuse six request/header/trailer plain/explicit modules. Add at most one aggregate
supported-shape module covering named scalar/outer array, repeated binary values
including an empty element, optional absent/default/supplied-empty and ordinary
text metadata. Direct analysis verifies encoder/decoder/required-contract key
agreement including designed errors; source immutability, mapped source names,
repeat query and no stale reverse alias are direct obligations. A separate #877
named-element module remains an explicit prerequisite, not silently replaced by
an outer alias. Required absence must fail before defaults; use existing #690
controls rather than expand its entire matrix.

The six existing modules include a protobuf-only TCP client against the generated
server and a handwritten protobuf server against the generated client. Inspect
actual standard keys/raw bytes and preserve required-absence errors. Comparison
uses pushed-parent/candidate/process-isolated repeat exact bytes, complete module
build/vet and focused mounted race checks. Intended changes are canonical wire
keys, matched key lookup/diagnostics and collision rejection; ordinary and
explicit-lowercase contracts retain byte-identical output where applicable.

## Readiness evidence and review boundary

All six peer modules build and vet. Three explicit-bin modules pass generated
round trips, both independent TCP peer directions, empty binary values and
required-absence controls. The three plain-key modules fail at the transport or
lookup boundary (26.701s combined run), not compilation. The direct expression
matrix reproduces21 spelling failures,6 unadmitted collisions, the implicit
credential collision and the inherited-error spelling failure. Nine explicit/text
key controls and two independent-channel/security controls pass under race
(1.326s and2.349s). There are no expression panics.

The initial full five-shape compiler input is preserved separately as
`/tmp/loom-877-named-metadata-element_test.go` with its red log and generated
modules. The #705 compile test now contains only the four passing supported
shapes; expression classification still covers named elements, whose mounted
acceptance is explicitly gated on #877. No acceptance case was silently removed.

Readiness is frozen for independent owner/model/acceptance review. Defaults,
repeated-value wire controls, mapped source identity and final structural
consumer agreement remain implementation acceptance obligations, not claims of
completed checks. No production mutation is authorized by these probes alone.

## Implementation start after prerequisite delivery

#877 is delivered at `6d2c03510`; this checkout advanced to pushed `e631f0a89`
with all 15 approved readiness source hashes preserved. Readiness and the R1
formatting re-review are approved. Implement the cached endpoint binding owner,
reuse effective security placement before admission, and publish fresh canonical
mapped attributes before consumers run. Then add the bounded structural and
aggregate wire obligations listed above, compare parent/candidate/repeat and
freeze for independent final review. No production encoder suffix fallback and
no designed-error delivery change are part of this ticket.

## Implementation evidence and collision migration

The owner is `GRPCEndpointExpr.ensureMetadataBindings`, cached before validation
and reused by finalization. Security placement now runs there from effective
requirements; codecs and response contracts have no new spelling branch. The
three production files are the endpoint declaration/validation, finalization,
and the extracted binding owner. Full expression/DSL checks pass; direct race
controls cover inherited/mapped credentials, unchanged source occurrences,
repeated admission, channel separation, type shapes and rejected collisions.

The seven generated binary specimens have 84 artifacts per generation. Candidate
and separate-process repeat are byte-identical. Only 11 artifacts differ from the
pushed-parent owner: server/client codecs and success required-key contract lists.
Explicit-suffix specimens remain identical. Six round-trip specimens build, vet
and pass mounted race checks; the aggregate covers named Bytes, named-element
arrays, repeated empty entries and absent/default/supplied-empty values. Parent
aggregate build/vet succeeds but all nine mounted cases fail lookup, as expected.
Designed-error direct checks cover evaluated names and required-key contracts;
they do not claim repair of the separately observed error-metadata delivery gap.

Focused security validation exposed the old session fixtures' two implicit
credentials sharing `authorization`. This is the modeled complete-binding
collision, not a reason to bypass admission. Root approved retaining a rejected
implicit-session witness and mapping the cookie explicitly to `x-session` in the
existing scope/security controls. Preserve their original auth-scope assertions;
add native TCP bearer/cookie/invalid controls. No new cookie-default policy is
introduced. Compare the two bounded scope fixtures separately; their explicit
mapping is an authored-input migration and must not be attributed to binary-key
output changes. Historical current behavior and the migration remain in the
review packet.

### Final bounded receipt

The two scope specimens add 24 unchanged artifacts to the seven binary specimens:
108 files per parent/candidate/repeat group, with exactly the same 11 binary key
changes and independent-process repeat equality. Authored cookie mappings are
identical in the three compared scope inputs; the separate original implicit
collision witness fails admission only on the corrected owner. Existing session
scope assertions and four native TCP auth scenarios pass (20.859s combined
codegen run). Full expression and DSL race suites pass (7.580s/1.697s), direct
consumer/default/security controls pass (6.224s), and scoped pinned lint reports
zero issues. The aggregate direct/mounted run passes (10.482s). The earlier
six-case binary run passed every round-trip module; its overall failure was the
then-new aggregate harness indexing a named protobuf wrapper, corrected before
the final aggregate run. Test-only setup failures are retained in the evidence,
not represented as passing runs.

The inspected checked-in gRPC quality design uses ordinary text metadata;
no affected binary mapping was found there. The bounded generated specimens and
existing auth-scope fixtures cover the changed owner; this is not a full fixture
audit. The model source/configurations remain the reviewed
readiness inputs, so their six checker results remain valid; correspondence docs
now name the implemented owner and delivered #877 dependency. Exact source,
inputs, output hashes and logs are recorded in the final manifest. Independent
final review, integration, commit and push remain pending.

The affected checked-in quality design adds one temp-copy comparison (16 artifacts
per group). Its uppercase authored metadata keys become lowercase in exactly two
codec files; all other output is identical, including a fresh process repeat.
Parent and candidate complete modules both build and vet. The comparison uses an
archived `e631f0a89` source tree for the parent because the CLI intentionally clears
Go overlay flags when rebuilding generators. The initial overlay attempt was
rejected as invalid baseline evidence. No generated tree is checked in for this
fixture, so no generated source files need committing. Named credential and
constraint generated controls also pass (5.929s); existing early-admission direct
controls remain green. All together there are 124 artifacts per source group and
13 intended differences, with no output added or removed.
