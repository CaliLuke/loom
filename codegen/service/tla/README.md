# Final result interception

`FinalResultInterceptor.tla` models one HTTP or gRPC client-streaming method's final
response. The service sends a canonical value with `SendAndClose`; the endpoint
then returns nil. An interceptor at the endpoint return cannot inspect or
change the already emitted value. This is the failure reproduced by #513.

The checked design dispatches result interception at the message operation.
A server interceptor reads or changes the canonical value before forwarding;
a client interceptor accesses the canonical value after `CloseAndRecv`.
Ordinary payload interception remains a separate callback. Dynamic view
selection must pass through the stream wrapper.

Run TLC with Java and a TLA+ tools jar, keeping state files outside the repo:

```sh
java -cp "$TLA2TOOLS_JAR" tlc2.TLC -config before.cfg -metadir /tmp/loom-final-result-before FinalResultInterceptor.tla
java -cp "$TLA2TOOLS_JAR" tlc2.TLC -config after.cfg -metadir /tmp/loom-final-result-after FinalResultInterceptor.tla
java -cp "$TLA2TOOLS_JAR" tlc2.TLC -config before-no-payload.cfg -metadir /tmp/loom-final-result-before-no-payload FinalResultInterceptor.tla
java -cp "$TLA2TOOLS_JAR" tlc2.TLC -config after-no-payload.cfg -metadir /tmp/loom-final-result-after-no-payload FinalResultInterceptor.tla
java -cp "$TLA2TOOLS_JAR" tlc2.TLC -config lost-view.cfg -metadir /tmp/loom-final-result-lost-view FinalResultInterceptor.tla
```

Run these commands from this directory. The two `before` configurations and
`lost-view` are counterexample checks and must fail. Both `after` configurations
must pass. TLC 2.19 reproduces emission before interception in the original
configurations and lost view selection in the candidate without forwarding.
Both corrected configurations pass all listed invariants.

The model checks canonical access, interception before emission, propagation of
mutations and selected views, suppression on service or interceptor errors,
the nil endpoint result, and one result callback for the final operation. It
covers two views and both an ordinary unary interceptor and a stream wrapper
with no ordinary payload access.

This is finite-state model checking of the lifecycle, not a proof of the Go
renderer. It assumes one final send and an interceptor that forwards once or
returns an error. It does not model arbitrary interceptor code, network
failures or concurrent stream operations. Generated module tests connect the
model to real code, covering canonical and viewed results, fixed views,
payload callbacks, context propagation, short circuits and transport errors.
DSL validation tests reject unary result access on HTTP/gRPC client streams.
JSON-RPC streaming-payload methods use a request/reply service boundary and
are outside this model; a generated regression test preserves unary result
access there. Golden and
compile-corpus checks verify the emitted interfaces and wrappers.
