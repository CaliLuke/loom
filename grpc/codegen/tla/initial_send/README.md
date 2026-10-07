# Initial gRPC send and final status

The model informed #591. A client-streaming or bidirectional RPC with a regular
payload opens a stream and sends an initial frame before returning it. grpc-go
`ClientStream.SendMsg` documents that send-side EOF can signal a remote stream
termination whose status must be read with `RecvMsg`. Discarding the stream at
that point loses its response or final status.

`Legacy.cfg` models returning every send error immediately. It violates
`FinalStatusRecoverable`: after opening and sending, EOF discards the stream.
`Preserved.cfg` retains the stream after EOF or a successful send and returns
other send failures immediately. The caller then owns completion through
`CloseAndRecv` (client streaming) or `Recv` (bidirectional streaming).

The model covers both kinds, with and without an initial payload, three send
outcomes and four final outcomes. `LocalFailuresVisible` checks that local
send errors stay terminal; `ReceivedStatus` checks final outcome preservation.
Under weak fairness, `EventuallyTerminal` checks completion. Fairness assumes
the caller eventually receives from a returned stream and that receive
completes. It does not promise progress from an idle caller or unresponsive
network. The two receive APIs share one abstract transition; their actual Go
behavior, wrapped EOF classification, protobuf decoding, and status codes are
test obligations. Retries, cancellation timing, metadata and concurrent calls
are outside this bounded model.

## Implementation and tests

- [grpcRemoteMethodBuilderSection](../../jennifer.go) owns the initial-send
  branch for both streaming kinds. `errors.Is(err, io.EOF)` preserves the
  stream; other errors still return immediately.
- [TestClientInitialSend](../../client_initial_send_test.go) checks the common
  renderer and goldens, including branches without a regular payload.
- [TestGRPCInitialSendPreservesCompletion](../../../../codegen/generator/grpc_initial_send_test.go)
  compiles the existing streaming fixture in all four variants. A deterministic
  stream stub covers EOF and wrapped EOF, remote rejection, a valid result,
  empty completion, canceled status and local send failure. Counters verify
  opening does not consume final status and designs without a payload do not
  send an initial frame. The raw builder preserves non-EOF error identity.

Before the implementation change, the model failed at depth 3 and both compiled
payload variants lost the final status after EOF. The no-payload controls
passed. The repaired generated clients pass all cases. This establishes
correspondence at the tested seams, not a proof of arbitrary generated Go.

## Reproduce the checker runs

Use the official TLA+ tools v1.7.4 JAR (TLC 2.19, revision `5a47802`), SHA-256
`936a262061c914694dfd669a543be24573c45d5aa0ff20a8b96b23d01e050e88`.
It is available from
[the upstream release](https://github.com/tlaplus/tlaplus/releases/tag/v1.7.4).
Set `TLA2TOOLS_JAR` to its path. Runs below used Corretto 25.0.4.1.
From this directory:

```sh
java -XX:+UseParallelGC -Xmx512m -cp "$TLA2TOOLS_JAR" tlc2.TLC -workers 1 \
  -metadir /tmp/loom-initial-send-legacy -config Legacy.cfg InitialSend.tla
java -XX:+UseParallelGC -Xmx512m -cp "$TLA2TOOLS_JAR" tlc2.TLC -workers 1 \
  -metadir /tmp/loom-initial-send-preserved -config Preserved.cfg InitialSend.tla
```

| Configuration | Required result | Observed states | Depth |
| --- | --- | ---: | ---: |
| Legacy | Exit 12, `FinalStatusRecoverable` counterexample | 113 | 3 |
| Preserved | Exit 0, all invariants and liveness pass | 160 | 4 |

Run the Go regressions from the repository root:

```sh
go test ./grpc/codegen -run '^TestClientInitialSend$' -count=1
LOOM_DIR="$PWD" go test ./codegen/generator -run '^TestGRPCInitialSendPreservesCompletion$' -count=1
```

Keep checker state and traces outside the repository and remove them after
validation and independent review. Keep this model, both configurations and
these summarized results with the owning generator.
