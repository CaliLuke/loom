# WebSocket close observation — #825

The generated message-limit harness must distinguish peer receipt of a close
frame from completion of the sending endpoint's native socket closure.
`http.WebSocketStream.releaseConnection` sends the control frame before Close.
The remote reader can run between those operations. Waiting on native-close
completion establishes the assertion's observation point without changing the
runtime's protocol ordering or polling deadlines that could cause closure.

## Model and correspondence

`CloseObservation.tla` allows peer receipt and native closure in either order
after frame send. Peer.cfg reproduces the unsound observation in 6 generated /
6 distinct states (SoundObservation counterexample). Native.cfg requires the
native completion event before checking, and passes 7 generated / 6 distinct
states. These are ordering/safety results only; fairness, termination, transport
errors and message-size policy are not modeled. The actual harness requires
close entry/completion within its five-second context and verifies the native
socket subsequently returns net.ErrClosed.

The regression wraps the listener's accepted connection and the client dialed
connection. Close forwards the native error unchanged and publishes completion
after the call returns. In the server-overflow case a gate holds native Close
after the peer has received code1009. The gate is then released and the harness
waits for the socket's own completion signal. All client/server small/large,
close-code, error-identity and service-invocation checks remain. Cleanup releases
the gate on failures as well. No runtime or generator output is changed.

## Reproduce

Use Java25 and TLA+tools1.7.4, JAR SHA256
936a262061c914694dfd669a543be24573c45d5aa0ff20a8b96b23d01e050e88.
From this directory with TLA_JAR pointing at the installed JAR:

```sh
java -XX:+UseParallelGC -cp "$TLA_JAR" tlc2.TLC -workers 1 -config Peer.cfg -metadir /tmp/loom-close-peer CloseObservation.tla
java -XX:+UseParallelGC -cp "$TLA_JAR" tlc2.TLC -workers 1 -config Native.cfg -metadir /tmp/loom-close-native CloseObservation.tla
```

From the repository root with Go1.27.2:

```sh
LOOM_DIR="$PWD" go test ./jsonrpc/codegen -run '^TestJSONRPCWebSocketMessageLimitGeneratedModule$' -count=1
```

At parent5bb31fd1910eb17c7fa1ba8256b201f4d1b0fc9c, 100 generated
runtime iterations reproduce the CI failure. Holding native Close with the same
test wrapper makes the old immediate socket assertion fail on the first run.
The repaired harness passes two process-isolated generations, each with100
runtime iterations under race detection. All generated files match parent and
repeat byte for byte. Exact hashes are preserved with the review receipt; the
owning plan records validation results. Module pins reference pushed source and
replace it explicitly with the candidate during local verification.

Keep this model and concise results; delete task-owned checker/generated output
when independent review and all active consumers finish. Do not commit traces.
