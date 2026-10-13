# Client stream operation ownership

This model first reproduced #720 at source
`ae3a281a219da31497c22aa9f08531a016662b88` before implementation.
The implementation retains its owned operation, cleanup and context contract.
The service interface, transport adapter and client interceptor wrapper must
agree on operation names, arguments and result types. A method signature is a
triple of those facts. The model checks five transports, four stream directions
where supported, and equal versus different unary/streaming result identities.
SSE admits only the download direction. It models client operations only;
JSON-RPC server notification/response operations are distinguished explicitly
and must not enter the client interface.

`current-*` reproduces the independent SSE `Recv(ctx)` signature and the
JSON-RPC WebSocket server-role interface. Correcting only these signatures
still hides SSE cleanup and leaves some WebSocket client interceptor wrappers
without the Close required by their service interface. The owned candidate
records Close alongside the method names/types, and each consumer uses that
contract. A separate background-context candidate demonstrates why adding a
no-argument Recv must retain the endpoint context.

| Configuration | Expected result | Observed distinct states |
| --- | --- | ---: |
| current-http-sse | ServiceImplemented violation | 4 |
| current-rpc-sse | ServiceImplemented violation | 4 |
| current-rpc-ws | ServiceImplemented violation | 10 |
| signatures-close-sse | CleanupReachable violation | 6 |
| signatures-close-ws | WrapperImplemented violation | 10 |
| background | OriginalContext violation | 6 |
| owned | All five invariants pass | 57 |

Run from this directory, using Java and TLA+ tools 1.7.4 (TLC 2.19), jar SHA-256
`936a262061c914694dfd669a543be24573c45d5aa0ff20a8b96b23d01e050e88`:

```sh
java -XX:+UseParallelGC -cp /path/to/tla2tools-1.7.4.jar tlc2.TLC \
  -workers 1 -metadir /tmp/loom-client-stream-owned \
  -config owned.cfg ClientStreamContract.tla
```

Run each other configuration with a separate temporary metadir. Expected
counterexamples exit 12 with the listed invariant; parser/evaluation failures
are not evidence. Initial model development exposed an undefined string
concatenation operator; the model now records the two concrete receive names
explicitly. All table results come from the corrected model.

## Correspondence and limits

- Generic signatures correspond to `service_data_methods.go` StreamData fields:
  Send/Recv names, contextual names, type references and MustClose.
- `OldAPI` corresponds to `streamInterfaceData` applying JSON-RPC WebSocket
  server operations to both roles, then suppressing client receive.
- `OldConcrete` corresponds to the two SSE client emitters' `Recv(ctx)` and the
  WebSocket emitter's existing canonical operations.
- `wrapper` corresponds to `buildInterceptorMethodData` and
  `interceptor_sections_stream.go`, including Close delegation.
- Context identity corresponds to the SSE constructor/endpoint retaining the
  opening context, no-argument Recv forwarding it, and contextual Recv accepting
  the caller's context. The model does not execute Go or prove I/O cancellation.
- The two regenerated ticktock static assertions, JSON-RPC streaming-payload
  assertions, and direct cleanup-capability test reproduce current defects.
  Ordinary HTTP WebSocket and gRPC generated compilation controls cover all four
  directions. The ticktock `service_client_stream_test.go` tests exercise typed
  clients with and without interceptors: accessor mutation, operation context,
  EOF, opening/operation cancellation, deadline and early Close. JSON-RPC
  `TestServiceClientWebSocketStreamRuntime` checks paired send/receive, accessor
  mutation and cleanup while another stream shares the connection. The shared
  SSE/gRPC control explicitly checks that Close still calls CloseSend.

The model abstracts concrete decoding, views, nils, framing, errors, concurrency
and network I/O. Different result identities check method-set consistency, not
mixed-result interceptor accessor correctness; that independent defect has a
separate reproduction. It assumes one transport contract at a time; the shared
HTTP SSE/gRPC generated control checks multi-transport Go consistency outside
this model. Close reachability means the existing transport operation remains
callable, not that every transport gives Close identical lifetime semantics.
In particular, existing gRPC Close delegates CloseSend; gRPC per-operation
cancellation (#726) is outside this change.

Lifetime remains owned by [SSEStreamReader](../../../../http/tla/sse_framing/README.md),
the [HTTP WebSocket owner](../../../../http/tla/websocket_lifetime/README.md), and
[JSON-RPC connection/stream owners](../../../../jsonrpc/tla/README.md).
Their models state their own safety/liveness assumptions; this method-set model
does not strengthen those claims. New typed-client tests must retain their
EOF, error, close and context behavior, including original-context cancellation
and early cleanup through an interceptor wrapper.
