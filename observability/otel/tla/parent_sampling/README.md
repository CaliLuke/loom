# Parent sampling (#748)

The shared provider must delegate valid local and remote parent decisions to
ParentBased, applying the existing ratio sampler only to roots. Explicit new-root
spans ignore the existing parent. The model enumerates valid/invalid parent,
local/remote, sampled/dropped, new-root and root-ratio decision booleans.
Current mode reapplies root policy and fails ParentDecision; checked mode follows
the parent only when valid and not explicitly discarded.

The model assumes extraction already supplied the correct context and abstracts
the SDK trace-ID algorithm as a Boolean. It does not prove remote propagation,
random ID generation, exporter delivery, or trust policy. Direct provider tests
use fixed parent IDs on both sides of the SDK threshold. Root tests compare the
actual generated ID with the SDK root policy, without statistical assertions.
An HTTP consumer test injects/extracts traceparent and tracestate through real
instrumentation. Test provider shutdown and application provider overrides remain
separate obligations.

Pinned TLC1.7.4 JAR SHA256:
`936a262061c914694dfd669a543be24573c45d5aa0ff20a8b96b23d01e050e88`.
Use Corretto25.0.4.1 and run from this directory with distinct output directories:

```sh
java -cp "$HOME/.cache/loom-tools/tla2tools-1.7.4.jar" tlc2.TLC -workers 1 -metadir /tmp/loom-748-state-checked -config checked.cfg ParentSampling.tla
```

Repeat with current.cfg; a semantic ParentDecision counterexample is expected.
Keep this model aligned with newTraceProvider and sampling_test.go. Production
must compose SDK samplers; do not replicate the root ratio algorithm or decide
sampling separately in HTTP/gRPC wrappers.

Results: current violates ParentDecision at50 generated/distinct states; checked
passes64 generated/distinct states. Independent readiness review repeated both
results. Implementation composes SDK ParentBased at newTraceProvider; it changes
no normalization, propagator, exporter or application provider override.
