# Span completion observation

The gRPC health response can reach the client before server OnEnd publishes to
the test recorder. Client completion therefore cannot authorize a two-span
snapshot. `SpanCompletion.tla` models ordinary and gated server completion,
client return, gate release, publication notification and the assertion snapshot.
Old exposes the missing happens-before edge. NotifyFirst shows why signaling
before recording is insufficient. Checked requires publication before notification
and notification before observation. The model is a bounded ordering argument,
not a proof of the OpenTelemetry SDK or of eventual server completion.

Run from this directory with Java 25 and TLC 1.7.4:

```sh
java -Djava.io.tmpdir=/tmp/loom-828-java -XX:+UseParallelGC \
  -cp "$HOME/.cache/loom-tools/tla2tools-1.7.4.jar" tlc2.TLC \
  -config Checked.cfg -metadir /tmp/loom-828-tlc-checked SpanCompletion.tla
```

Create the external Java temporary directory first. JAR SHA256:
`936a262061c914694dfd669a543be24573c45d5aa0ff20a8b96b23d01e050e88`.
Run Old and NotifyFirst separately with distinct metadata paths; both must exit
12 on CompleteSnapshot. Checked must exit 0. Sources/configurations are durable;
checker state, traces and logs remain temporary.

The retained parent witness `testdata/parent_test.go.txt` is a complete replacement
for `grpc/middleware/otel/middleware_test.go` using `go test -overlay`. It gates
only server OnEnd and intentionally retains the original immediate assertion.
The gate is released before cleanup and the Serve result is collected. Run the
original named test with race detection; the expected failure is exactly one
recorded client span rather than the two asserted spans. The readiness plan
records the command and source parent.

Implementation correspondence: a test-local processor first invokes
SpanRecorder.OnEnd, then nonblockingly notifies the bounded wait. The normal and
gated RPC cases use the same count/kind/name assertions. Context deadline tests
cover missing completion; no fixed sleep or production delay is introduced.
Counts, names, channels, shutdown behavior and Go races require runtime tests;
this model abstracts them into two publication events. Additional unexpected
spans must still fail the exact-count assertion, not block a processor channel.

Readiness results: Old exits 12 (7 generated / 6 distinct states), NotifyFirst
exits 12 (15 / 10), Checked exits 0 (9 / 7). Both failures are CompleteSnapshot.
The gated original test fails with one recorded span; the ordinary parent
package passes. The model does not claim that missing span completion can be
made successful; a bounded runtime wait must report timeout in that case.

To create the parent overlay from this directory:

```sh
python3 - <<'PYCODE'
import json
from pathlib import Path
root = Path.cwd().parents[4]
json.dump({"Replace": {
    str(root / "grpc/middleware/otel/middleware_test.go"):
    str(Path.cwd() / "testdata/parent_test.go.txt")
}}, open("/tmp/loom-828-parent-overlay.json", "w"))
PYCODE
go -C "$(git rev-parse --show-toplevel)" test -race \
  -overlay /tmp/loom-828-parent-overlay.json ./grpc/middleware/otel \
  -run '^TestServerAndClientOptionsCreateTracingSpans$' -count=1
```
