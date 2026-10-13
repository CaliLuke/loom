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
`writeStreamResultBodyInit` and `writeServerWebSocketResponseBody`. Metadata
capture already uses the correct HTTP response encoder. `Expected` is the
complete shared response projection. Mixed SSE has a separate StreamingResult,
so its event remains independent of unary selection. The model does not assert
that an otherwise unsupported pure StreamingResult/Body DSL design is accepted.

`calls` counts calls to the configured HTTP encoder, not JSON marshaler methods
or WebSocket framing internals. Existing nonmetadata HTTP responses call it once
and metadata captures twice; the shared plan preserves that boundary. WebSocket
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
its result. Existing metadata capture still invokes the HTTP encoder. The SSE
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
with `NoUnusedResult` failing (11 states); `binding-owned.cfg` passes both
invariants (17 states). Run with the pinned toolchain above, replacing the module
and configuration names and choosing a separate temporary metadir.

`hasJSONRPCResponseBody` consumes the existing effective `EncodePlan.HasBody`.
`writeJSONRPCEndpointInvoke` binds a result only when that body, metadata capture,
or WebSocket endpoint forwarding consumes it. Existing success renderers keep
their notification, framing and error-observation paths. Named types are not a
separate body-presence policy. Do not silence unused bindings or send an
unprojected service result merely to make it appear used.

The model's four shapes are no-result, named-empty, body and metadata-only, with
and without forwarding. It checks source availability and absence of unused
bindings; it abstracts payload/error variable declarations, type relocation,
wire bytes and actual Go compilation. `TestJSONRPCBodylessResultBinding` and
`TestJSONRPCBodylessResultGenerated` establish those seams for local/relocated
empty types, payload/no payload, metadata-only and ordinary controls. Mounted
success, error and notification cases check null/value results, status, service
invocation, headers/cookies and codec counts. Existing selected-body and
WebSocket-no-result module tests preserve the affected consumers. An empty view
is rejected by the DSL; this does not create a new empty-view contract.
