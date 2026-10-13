# Passive request capture

`RequestCapture.tla` explores the passive Debug request observation implemented for #680. Downstream Read/Close operations are the only I/O authorities; handler return
only snapshots the observation.
The captured sequence includes byte counts and the exact error category;
replacement of a failed read with successful diagnostic text is forbidden.
The preview bound is two bytes, representing the selected 64 KiB runtime cap.
After overflow, the preview is discarded permanently while application reads
continue unchanged. Returning without EOF never drains or closes the body.

Run with Java and TLC 1.7.4, using a temporary metadata directory outside the
repository (each run needs its own directory):

```sh
java -cp /Users/luca/.cache/loom-tools/tla2tools-1.7.4.jar tlc2.TLC \
  -config Checked.cfg -metadir /tmp/loom-680-tlc-checked RequestCapture.tla
```

The pinned JAR SHA256 is
`936a262061c914694dfd669a543be24573c45d5aa0ff20a8b96b23d01e050e88`.
Run Eager, Replace, Close and Unbounded configurations to reproduce individual
current defects; each must exit 12 with its corresponding invariant violation.
Checked must pass all four invariants. Splitting negative configurations avoids
the first eager-read counterexample hiding later error, Close or memory faults.

Correspondence: `http/middleware/debug_request_test.go` checks unchanged bytes
and concrete error identity, requested read lengths, source Close delegation,
request-size policy ordering, partial/unread bodies and 64 KiB boundaries.
The original `writeDebugRequest` eagerly invoked `readDebugBody`, replaced
failures with successful text, and installed a NopCloser. The implemented
`transport.CaptureBodyReads` calls the original body once per operation, then
records only returned bytes under a short capture-state lock. Its promoted Close
method delegates directly; the lock never covers underlying I/O. Request and
response capture share private bounded retention, not their I/O authority.
Debug snapshots after the handler and marks a non-EOF observation as incomplete.
`observability/transport/body_reads_test.go` additionally checks detached and
concurrent snapshots, nested capture, exact read/error/Close identity and Close
unblocking a blocked Read. A real-network middleware test flushes an SSE response
while its upload pipe remains open. Downstream policy probe reads are observable
when the policy is inside Debug; they are not claimed as endpoint-visible bytes.

Limits: three handler operations and abstract bytes/error categories, no actual
network, byte contents, Go interface implementation, allocator or goroutine
scheduler. Concurrent Close must unblock a blocked Read without a capture lock;
that requires a runtime test and is not proved by this sequential-operation
model. HTTP server final closure remains outside the middleware. No liveness
claim is made about arbitrary underlying bodies, which can block independently.
The shared passive read owner also serves client responses; automatic client
completion timing, response writer semantics and generated code are not modeled
here. Client completion has a separate model under `http/tla/client_debug_response`.

Readiness run: Eager, Replace, Close and Unbounded exited 12 on NoExtraRead,
ReadIdentity, CloseOwnership and BoundedPreview respectively, after 2, 6, 12
and 27 states. Checked exited 0 with 2,866 generated / 1,845 distinct states.
