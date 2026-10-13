# Client endpoint interceptor dispatch (#867)

The client constructor knows which endpoint returns an ordinary result and which
returns a stream. A shared endpoint plan must carry that role into interception.
Two ordered interceptor profiles cover stream-only, combined payload/stream,
payload-only and generic/no-access callbacks, with nil interceptor and failed
opening controls. Chain order is modeled in execution order, including repeated
profiles; callbacks must retain their sequence and multiplicity.

Run each configuration with the pinned TLC jar and a distinct temporary metadir:

```sh
java -cp /Users/luca/.local/share/tlaplus/tla2tools-1.7.4.jar tlc2.TLC -workers 1 \
  -metadir /tmp/loom-867-model-owned -config owned.cfg ClientInterceptorDispatch.tla
```

| Configuration | Expected result |
| --- | --- |
| current | OrdinaryResultRetained fails: mixed ordinary result asserted to stream |
| skip-ordinary | OpeningCallbacks fails: combined/payload/generic callback lost |
| ordinary-only | MessageCallbacks fails: stream receive/send callback lost |
| owned | All invariants hold |

The constructor seam is `client_jennifer.go:wrappedClientEndpointExpr`; current
wrapper policy is `interceptor_sections.go:renderClientInterceptorWrapperBody`.
`client_interception.go` now owns the plan connecting constructor selection
and public/private wrapper emission. `MethodData.ClientInterceptorWrapper`
exposes each allocated endpoint binding to shared CLI command data. HTTP CLI
selects the ordinary mixed endpoint; JSON-RPC SSE CLI selects the stream-opening
endpoint; gRPC selects its actual stream role. These are references to the same
plan, not transport-specific applicability decisions. `serviceSymbols` remains the separate declaration-name owner. The
`TestClientInterceptorDispatch` seam and `TestMixedClientInterception` generated
harness connect the counterexample to code. The latter covers six profiles, nil
interception, ordered callbacks, payload/message mutation, failed opens, short
circuits, receive errors/EOF, context and Close delegation, mounted transports
and both CLI endpoint selections. Ordinary/final controls and a new wrapper-name
collision are retained in the bounded artifact comparison.

Assumptions: ordinary and stream endpoints obey their successful return type
contract; failed opens have no stream operation. Application interception can
short circuit, and successful short circuits must obey the same endpoint type
contract. This model does not simulate arbitrary callback code or nil-success
values. It abstracts message send/receive as the same dispatch event, and does
not establish transport support for all abstract combinations. Context and Close
are unchanged identity records, not a cancellation/liveness proof; keep #720's
actual operation tests. The #859 accessor owner and #785 server carrier issues
are outside this model. Go compilation, name allocation and protocol negotiation
require implementation evidence and are not proven here.

Readiness check: the three counterexample configurations exit 12 with the named
invariant; `owned` checks 817 generated / 545 distinct states with no error.
All 272 selected input states advance; terminal states self-loop and deadlock
checking is enabled. During model development a missing parenthesis in a boolean
assignment constrained transitions; the recorded results use the corrected model.
