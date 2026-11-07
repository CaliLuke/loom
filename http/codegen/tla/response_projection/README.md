# Current JSON-RPC response admission (#905)

The user accepted uniform application response bodies. `uniform-bodies.cfg` and
`binding-uniform-bodies.cfg` retain the projection/source invariants while admitting
no authored metadata mappings. The prepared endpoint validator rejects these
mappings for every JSON-RPC carrier; generators no longer capture HTTP metadata or
decode application fields from it. HTTP envelopes call the configured encoder
once; notifications do not. Framework-owned view identity is unaffected.

The configurations and descriptions below retain historical #853/#868/#886 evidence,
including the old metadata-capture alternatives. They do not describe currently
admitted metadata mappings. Run the current configurations with their respective
ResponseProjection and ResponseSourceBinding modules using the toolchain below.
They pass with 97 and 17 distinct states, respectively.
Static DSL admission is checked by TestJSONRPCResponseCarrierAdmission; mounted
body/typed-error tests cover actual codecs and physical carriers outside this model.

# Response projection before protocol framing

This #853 model describes the constructor-only fallback in JSON-RPC
HTTP, JSON-RPC WebSocket and HTTP WebSocket at `d4b2905d8`. An explicit selected
body can be a native scalar needing no conversion constructor; treating absence
of a constructor as permission to send the service result leaks sibling fields.
Mounted generated-code witnesses reproduce that leak in all three channels.

The implemented owner completes the existing ResponseEncodePlan with the selected body
value for every view. It does not change source-field identity or physical
storage, already owned upstream by response analysis and service layout.

Run with TLA+ tools 1.7.4 (TLC2.19), jar SHA256
`936a262061c914694dfd669a543be24573c45d5aa0ff20a8b96b23d01e050e88`:

```sh
java -cp /path/to/tla2tools-1.7.4.jar tlc2.TLC -workers 1 \
  -metadir /tmp/loom-response-projection-owned -config owned.cfg ResponseProjection.tla
```

Run each other cfg using a separate temporary metadir. Expected negative
configurations must exit12 due to the specified invariant, not parse errors.

| Configuration | Intended result | Distinct states |
| --- | --- | ---: |
| current | NoExcludedFields fails: selected scalar uses whole result | 63 |
| scalar-patch | NoExcludedFields fails: unary fix leaves WebSocket leaks | 63 |
| first-view | SelectedWire fails: all results projected with default view | 97 |
| unary-everywhere | SelectedWire fails: mixed SSE event uses unary body | 65 |
| capture-everywhere | CodecBoundaryPreserved fails: formerly single encode becomes two | 77 |
| owned | All invariants pass | 121 |

## Correspondence and limitations

`shape` distinguishes a constructor-free scalar selection, constructor-backed
projection and whole-body identity. The selected/whole member sets abstract JSON
values. `view` distinguishes a constructor-backed default and narrow view.
The model treats bytes, names and nullable representations as correctly resolved
by the pre-existing analysis; direct Go tests must establish that correspondence.

`CurrentProjection` follows `writeJSONRPCResultSuccess`,
`writeStreamResultBodyInit` and `writeServerWebSocketResponseBody`. The historical metadata
capture used the correct HTTP response encoder. `Expected` is the
complete shared response projection. Mixed SSE has a separate StreamingResult,
so its event remains independent of unary selection. The model does not assert
that an otherwise unsupported pure StreamingResult/Body DSL design is accepted.

`calls` counts calls to the configured HTTP encoder, not JSON marshaler methods
or WebSocket framing internals. In the historical design, nonmetadata HTTP responses called it once
and metadata captures twice; the then-shared plan preserved that boundary. WebSocket
and SSE do not use that configurable HTTP codec here. HTTP requests without IDs
invoke no response encoder; outgoing stream notifications still carry their
projected params. Request parsing, endpoint invocation and batch envelope joining
are outside this model and remain covered by mounted tests and existing runtime
batch tests.

View metadata is tied to the same selected view. The model assumes valid views
and does not prove rejected-view handling, metadata serialization, custom codec
behavior, allocation bounds, concurrency, cancellation or network liveness.
Tests must cover those affected implementation boundaries and preserve existing
stream owner guarantees. No new lifecycle protocol is proposed.

## Implementation and regression seams

`newResponseEncodePlan` records `BodyValues` for each actual ServerBody variant:
all constructor ServerArgs, or the canonical service/projected/selected value.
HTTP response templates and WebSocket final sends consume that map. JSON-RPC
`responseBodyInit` binds the canonical source before using the same expressions;
`writeJSONRPCResultSuccess`, stream responses and outgoing notifications frame
its result. The historical metadata capture invoked the HTTP encoder. The SSE
body family uses the owning response plan; a separate mixed StreamingResult
retains its event source. View construction and validation remain upstream.

`TestResponseProjectionValues` checks direct expression ownership, including
multiple constructor arguments, viewed variants and bodyless responses.
`TestSelectedResponseProjectionReadiness` exercises mounted unary wire/client,
batch, incoming notifications, selected arrays/objects and codec failure/count
contracts. `TestSelectedStreamProjectionInventory` and the HTTP
`TestSelectedStreamProjectionReadiness` cover selected WebSocket values, empty
values and JSON-RPC outgoing notification params. Existing viewed unary/stream
and Any stream tests retain conversion and view metadata controls.
`TestSelectedResponseStorageGeneratedMixed` extends the physical-storage fixture
with mounted JSON-RPC constructors, clients, absence/null/empty values and invalid
views. Its HTTP assertions remain in place.

The final bounded generation comparison covers 11 modules (182 files each):
13 server files change, with no additions/removals and exact independent-process
repeat. Every candidate module builds and vets; the scoped mounted suites run
under the race detector. Model states/checker outcomes above were reproduced
before implementation; these tests establish the implementation correspondence.


## Endpoint source binding (#868)

`ResponseSourceBinding.tla` complements the framing model with the local result
binding that precedes it. A named empty result has a Go reference but no response
body. The old `Result.Ref` rule binds an unused local; framing correctly emits a
null result without consuming that value. `binding-current.cfg` reproduces this
with `NoUnusedResult` failing (13 states after the explicit-empty extension);
`binding-owned.cfg` passes all three invariants (21 states). Run with the pinned toolchain above, replacing the module
and configuration names and choosing a separate temporary metadir.

`hasJSONRPCResponseBody` consumes the existing effective `EncodePlan.HasBody`.
`writeJSONRPCEndpointInvoke` binds a result only when that body, metadata capture,
or WebSocket endpoint forwarding consumes it. Existing success renderers keep
their notification, framing and error-observation paths. Named types are not a
separate body-presence policy. Do not silence unused bindings or send an
unprojected service result merely to make it appear used.

The model's five shapes are no-result, implicit named-empty, explicit-empty,
body and metadata-only, with
and without forwarding. It checks source availability and absence of unused
bindings; it abstracts payload/error variable declarations, type relocation,
wire bytes and actual Go compilation. `TestJSONRPCBodylessResultBinding` and
`TestJSONRPCBodylessResultGenerated` establish those seams for local/relocated
empty types, payload/no payload, metadata-only and ordinary controls. Mounted
success, error and notification cases check null/value results, status, service
invocation, headers/cookies and codec counts. Existing selected-body and
WebSocket-no-result module tests preserve the affected consumers. An empty view
is rejected by the DSL; this does not create a new empty-view contract.

### Explicit empty response presence (#886)

`IntendedBody` distinguishes an authored zero-field object from the no-body
sentinel and an inferred empty projection. `binding-cardinality.cfg` reproduces
collapsing the explicit object by field count: `PreservesAuthoredBody` fails in
14 states. The shared explicit preparation owner preserves it; owned passes
21 states, including source availability for consumers of that body. This is a
finite representation model, not a proof of JSON encoding or object validation.
`TestExplicitEmptyResponseBody` checks actual DSL preparation, and the extended
InlineBodySelection and JSON-RPC bodyless fixtures check rendered compilation,
wire objects, client reconstruction, metadata, and retained implicit null results.

## Nullable parent storage and metadata

`NullableResponseProjection.tla` extends the earlier assumption that nullable
representations were already resolved. The concrete witness accessed selected
body/header fields on a Nullable wrapper and emitted uncompilable clients and
servers for ordinary results and errors. The corrected projection extracts the
concrete value before field access. An explicit selected body requires a concrete
parent; its child's authored nullability remains independent. Whole-body null is
accepted only without supplied mapped metadata and without required metadata.
Defaults do not count as wire-supplied metadata or create a parent.

Run the same pinned TLC tool explicitly:

```sh
java -cp /path/to/tla2tools-1.7.4.jar tlc2.TLC -workers 1 \
  -metadir /tmp/loom-nullable-response-model \
  -config nullable-owned.cfg NullableResponseProjection.tla
```

| Configuration | Expected exit / property | Generated / distinct states |
| --- | --- | --- |
| nullable-owned | 0, all listed invariants | 3456 / 3456 |
| nullable-wrapper-fields | 12, FieldStorage | 1729 / 1729 |
| nullable-zero-parent | 12, CheckedPublication | 1729 / 1729 |
| nullable-drop-header | 12, CheckedPublication | 2265 / 2265 |
| nullable-parent-only | 12, CheckedPublication | 1777 / 1777 |
| nullable-whole-body-reuse | 12, ValidReconstruction | 3217 / 3217 |
| nullable-missing-object | 12, FieldStorage | 2065 / 2065 |
| nullable-view-wrapper | 12, NativeBinding | 1796 / 1796 |

Each run needs its own temporary metadir. This remains an opt-in design tool.
`CheckedPublication` rejects invalid combinations before publication;
`SelectedParent` and `PresencePreserved` additionally constrain successful
reconstruction. The model abstracts headers/cookies to one supplied bit and
constraints to requiredness/child nullability. It assumes the existing native
field layout, checked scalar conversion and actual codec null-admission owners;
those are tested separately, not proved by this state machine.

`internal/representation.ConcreteAttribute` retains root/child identity while
extracting storage. `nullable_response_projection.go` captures checked binding,
metadata presence and reconstruction; the response/body/constructor owners and
shared EncodePlan consume it. `TestNullableResponseProjectionGenerated` compiles
and runs both ordinary and nominal-error selected scalar, nullable child,
constructor-backed object and whole-body metadata cases, including native int8,
defaulted cookies, required headers, absence and pre-publication failure. Existing
concrete presence copy/alias/recursive tests qualify the mechanical shared-owner
move. No policy is inferred from temporary generated output.

The nullable source binding is also a required input to WebSocket frame
projection and the shared JSON-RPC `responseBodyInit` consumer. The former
WebSocket omission is reproduced by `TestNullableSelectedStreamProjection`;
`TestResponseBodyInitConsumesSourcePreparation` checks the neighboring consumer
with an actual captured HTTP plan. JSON-RPC permits Body selection; this direct
check is complemented by `TestNullableSelectedJSONRPCStream`, which compiles an
actual admitted nullable selected-result WebSocket module. Invalid null/absent
notifications return errors, then a valid empty notification is the first frame;
ordinary concrete request/response controls preserve selected values.
Both consumers run the same EncodePlan preparation before reading its
BodyValues. No separate nullable policy or conversion is introduced.

For WebSocket calls, `CheckedPublication` describes the application data frame.
The connection may already have upgraded, for example in Recv before the
application computes its result. Invalid selected parents return through Send's
existing error channel and produce no application frame; this model does not
promise rollback or prevention of that earlier handshake. The mounted regression
covers concrete/empty selected values and null/absent rejection after upgrade.
It preserves SendAndClose cancellation precedence: closing the stream cancels
the request context, so that caller receives context.Canceled; a detached
operation context retains the original checked-projection error. Both paths
reject before any application frame.
The WebSocket consumer correction did not change model transitions. The subsequent
shared projection extension above adds ordinary/nullable parent storage, selected
child absence, metadata-only reconstruction and actual native/view source binding;
the table records the renewed combined checks.

Selected-body publication requires both a concrete parent and a supplied child,
independently of whether the parent is nullable. Optionality is not permission to
publish null. The existing physical presence classifier observes pointers,
Nullable presence and union selection without treating empty values as absent or
invoking opaque codecs. A metadata-only mapping also requires a concrete parent;
its client reconstructs one even when all optional metadata was omitted. Truly
bodyless mappings without metadata retain their prior absence policy.

`source_presence.go` is shared with request presence capture.
`responseBodyAbsence` captures each body's actual native or view layout;
`ResponseEncodePlan.SourcePreparation` checks it before body conversion and
publication. `responseNativeSource` selects `res.Projected` for viewed results.
Tag matching is a separate safe predicate: null/absent parents do not match and
may reach a whole-body fallback that permits null. Selected-body validity is
checked only after that selection. `TestResponseProjectionCompletionGenerated`
checks these boundaries, metadata-only result/error constructors, local-name
collisions and nullable tag fallback. `TestSelectedResponseStorageGeneratedMixed`
retains named, viewed, custom-storage, default, empty and explicit-null controls;
its optional named absence regression now requires checked rejection.

The model does not implement selector arbitration, native Go storage rules or
custom codec behavior. Direct/generated controls supply that correspondence;
no blanket nil or collection-length policy is inferred from its presence symbols.

The direct generated `TestViewedEncoderRejectsBeforeMetadata` asserts an empty
writer header map and no status/body writes when source preparation fails.
The response partial publishes dynamic `loom-view` only after its shared
preparation, rather than relying on an outer handler to clear it. The model's
atomic publication abstraction does not prove individual writer mutations; this
direct check establishes that concrete ordering.

## Nonnullable authored object projections

The `object` mode represents a nonnullable emitted object body from a nullable
native parent; its initialization explicitly requires nullableParent. The
missing-object mutant retains that restriction alongside the expanded ordinary
parent modes. It requires a concrete source before encoding and reconstructs a
concrete parent on decoding, independently of whether an origin member was
selected. The combined owned configuration includes the ordinary/view/metadata dimensions
above; renewed counts are recorded in the table.
`nullable-missing-object.cfg` fails FieldStorage by omitting concrete-parent
lowering for this mode. This abstracts the observed generator failure, not a
claim that the uncompilable output published a response. Header requiredness and
codec behavior remain the stated assumptions above.

`nullableResponseRequiresConcrete` derives the mode from the captured emitted
body and native context. `responseBodyArgument`, shared result reconstruction
and SourcePreparation consume it. `TestNullableObjectBodyProjection` reproduces
the ordinary-result seam; `TestNullableDefaultErrorGenerated` compiles and runs
both ordinary and problem-object cases, including null/absent rejection before
publication and concrete reconstruction. No nullable annotation is added to the
authored body to avoid the checked projection.


## Replacement body ownership and native tag presence

`ResponseBodyOwnership.tla` refines the selected-body boundary above. A custom
error formatter replaces the declared body, as documented in the error guide;
it does not replace native metadata or make an absent payload supplied. The
model separates the selected member from its native parent and metadata access.
A null parent with no mapped metadata may supply a replacement body. A selected
native body still needs its concrete parent and supplied member. An absent tag
parent is unmatched; the selected default's own projection decides admission,
so a truly bodyless default can succeed without field access.

Use the pinned TLC tool above, with `ResponseBodyOwnership.tla` and one of these
configurations. These commands are opt-in design checks, never CI prerequisites.

| Configuration | Expected exit / invariant | Generated / distinct states |
| --- | --- | --- |
| body-ownership-current | 12, ValidReplacement | 69 / 69 |
| body-ownership-nil-tag | 12, NativeTagSafety | 50 / 50 |
| body-ownership-bypass | 12, CheckedPublication | 53 / 53 |
| body-ownership-owned | 0, all listed invariants | 96 / 96 |

The current alternative checks the unused native body even when a formatter
owns publication. The bypass alternative incorrectly skips native prerequisites
as well. The nil-tag alternative reads a field before checking parent presence.
The accepted owner checks native prerequisites independently and applies member
checks only when the declared body is selected.

`responseBodyAbsence` captures separate parent/member facts using the existing
physical absence owner. `SourcePreparation` takes an explicit generated body
condition; HTTP error encoding supplies `formatter == nil`, while ordinary HTTP,
WebSocket and JSON-RPC consumers select the declared body unconditionally.
`responseTagData` uses that same native presence classifier before tag access.
The ordinary response-selection module covers nil rejection for selected bodies
and successful nil fallback for a bodyless default. Problem-field and nullable
projection modules cover replacement callbacks, absent selected members, nullable
parents and preserved metadata rejection before publication.

The model abstracts actual Go layout, codecs, constraints and callback values.
Its selected-body admission does not cover the bodyless fallback; that case is
an implementation regression for the separate tag-safety predicate. Existing
projection models retain full nullable child, view and reconstruction coverage.

## Preparation failure completion

`ResponsePreparationFailure.tla` covers the failure channel omitted by the earlier
projection models: source presence and checked conversion may reject before an
HTTP JSON-RPC envelope exists. Returning that error to the dispatch runtime is
not protocol completion: the runtime's error path reports transport failures and
does not manufacture an envelope. `preparation-current.cfg` reproduces that
missing response with `FailureFramed` failing (36 states). The corrected caller
supplies the error renderer to shared preparation and conversion, so unary
requests with IDs attempt one safe internal-error envelope; stream APIs return
the failure to their existing caller. Notifications still invoke no response
encoder. Physical encoder failure stays diagnostic and never triggers a retry.

Run the pinned tool above with `-deadlock` (terminal states are intentional),
`-config preparation-owned.cfg ResponsePreparationFailure.tla` and a fresh
external metadir. Owned passes all five invariants (77 states).
`preparation-retry-write.cfg` fails `OnePublicationAttempt` (66 states).
`preparation-reach-error.cfg` and `preparation-reach-stream.cfg` deliberately
fail their reachability invariants (66 and 35 states), proving an attempted error
envelope with encoder failure and a returned stream preparation failure are
reachable. These checks are opt-in, never CI gates.

The booleans represent checks selected by the actual plan; a bodyless plan
with no preparation checks is represented by true, not a new nullable guard.
Source/constraint evaluation and conversion are abstract booleans; error values,
safe-message resolution, actual encoding and callback behavior are not proved.
The model records encoding attempts, not successful network delivery. It assumes
callbacks return and treats each attempt atomically; concurrent stream ownership
remains with its existing runtime model. Actual mounted shared/shared-reverse,
nullable result/error and optional-failure-reporting tests establish the generated
code and error-framing correspondence. The source owner still supplies presence
and projection; the caller owns failure completion. No nullable/status rule or
runtime postwrite policy is duplicated here.
