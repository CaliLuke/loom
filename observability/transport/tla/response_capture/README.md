# Response capture (#608)

One response capture state owns final status and byte accounting for both
request observation and clue HTTP logging. httpsnoop v1.1.0 owns conditional
interface forwarding. HTTP failure routing reads state through the observer,
not through a concrete writer type. Unary handlers share HandlerLifecycle.

The model explores three sequential header/write/flush operations with 101,
103, 200 and 500. 103 remains informational, 101 commits an upgrade, and the
first final status wins. `Legacy` fails `Status` after 103; `Checked` passes
with 13 distinct states. Model assumptions are valid HTTP codes and serialized
writer calls. It does not model bytes, I/O failures, optional interfaces,
hijacked connection I/O or arbitrary middleware. Tests establish those seams;
atomic capture accessors do not make the underlying writer concurrent-safe.

Run with official TLA+ tools v1.7.4 (TLC 2.19 rev `5a47802`), JAR SHA-256
`936a262061c914694dfd669a543be24573c45d5aa0ff20a8b96b23d01e050e88`.
Validated using Corretto 25.0.4.1. From this directory:

```sh
for cfg in Legacy Checked; do
  java -XX:+UseParallelGC -cp "$TLA_TOOLS_JAR" tlc2.TLC -workers 1 \
    -metadir "/tmp/loom-capture-$cfg" -config "$cfg.cfg" ResponseCapture.tla
done
```

Legacy must fail the invariant (exit 12), not fail parsing. Checked must exit 0.
Delete checker artifacts after review.

`ResponseCapture.commit` implements the first-final transition; WriteHeader
filters informational responses. Write/WriteString and Flush/FlushError commit
200, while ReadFrom commits only after transferring bytes. An empty ReadFrom
need not commit, matching net/http. No-write state remains zero. Upgrade writes
directly to a hijacked connection are outside capture. The library metrics API
is callback-scoped; it cannot own independently started/ended observation and
its status policy does not cover flush commits and terminal 101.

`TestCaptureResponseWriterInformationalStatus` and the capability regression
failed before implementation. `TestCaptureResponseStatus`, `Capabilities` and
`FastPaths` cover transitions, nested capture, optional interface absence,
forwarded errors, controller operations and optimized I/O. HTTP lifecycle tests
cover committed write failures and failed WebSocket upgrades; compiled generated
HTTP/WebSocket and SSE fixtures cover transport consumers. No generator changes
or generated-output differences are intended.

## Debug response adapter (#679)

`DebugTransparency.tla` isolates the responseDupper defects from the shared
owner. It explores three sequential operations: informational/final/upgrade
headers, attempted two-byte writes accepting zero/one/two bytes, and Flush.
`DebugStatus`, `DebugBytes`, and `DebugFlush` independently reproduce last-header
or missing-implicit status, capture-before-write, and a hidden Flush operation.
`DebugChecked` derives status from the shared first-final owner, observes only
accepted bytes, and forwards flush. The accepted server policy adds bounded
retention: model cap 2 represents the concrete 64 KiB cap. DebugCap models the
old unbounded buffer and fails BoundedCapture. The checked owner discards the
preview after overflow and marks omission while accepted wire bytes continue.
Run each configuration with the same pinned
TLC above, using `DebugTransparency.tla` and an external metadir.

The counters abstract successful flush forwarding; concrete errors, optional
interface sets, deadline/duplex controls and SSE terminal state remain Go test
obligations. The user delegated the capture policy choice; the model checks its consequences
and does not decide that policy. It does not model concurrent ResponseWriter use,
transport framing, custom handler ownership, request reads, ReadFrom lookahead,
or hijacked connection data.

Implementation correspondence: `TestDebugResponseStatus` checks Header/Write;
`TestDebugResponseAcceptedBytes` checks actual partial writes through Write,
WriteString and ReadFrom; `TestDebugResponseController` and Capabilities check
forwarding; `TestDebugResponseSSEWithBodyPolicy` checks Open and SendComment
flushing before handler return with an unchanged valid limited request.

Results: DebugStatus/Bytes/Flush exit 12 on their named invariant
(3/3, 6/6, 9/9 generated/distinct states). DebugCap exits 12 on BoundedCapture
(56/27 states); DebugChecked exits 0 (293/75).
These negative controls must fail invariants, not parser/action completeness.

`WithResponseBodyCapture` configures the shared CaptureResponse hooks; default
metrics consumers retain no body. Body snapshots are detached and synchronized,
without promising concurrent ResponseWriter use. Write/String retain only n
accepted bytes. ReadFrom retains bounded lookahead, then trims to the returned n;
reader consumption alone cannot mark a response oversized. Tests cover full,
partial, error, exact-limit, overflow, nested capture and concurrent observation.
`TestDebugResponseSSEFlushVisibleBeforeReturn` checks a real HTTP connection;
`TestDebugResponseSSETerminalFailure` keeps failed streams terminal.

## Diagnostic metadata (#718)

`ResponseMetadata.tla` explores informational preparation, an observable commit,
an opaque ReadFrom, a forwarding writer changing metadata, later handler mutation
and final observation. `LateHeader` reproduces the original JSON-to-plain parser
switch. `EarlyRead` rejects using a source read as proof of ReadFrom's commit;
`AfterCall` rejects trusting metadata after a forwarding writer changes it.
`MetadataChecked` preserves only snapshots stable across the forwarding call,
and omits previews whose first commit has ambiguous metadata. Run these four
configurations with `ResponseMetadata.tla` and separate external Java temporary
and checker directories using the pinned toolchain above.

The model assumes serialized writer calls and no hidden mutate-and-restore
sequence inside a forwarding writer. It cannot observe such a writer's internal
commit; custom byte/header transformations remain outside capture's boundary.
The shared `headerCandidate`/`commit` pair owns the stable before/after snapshot;
`Headers` returns detached immutable data. ReadFrom preserves an earlier observed
commit but never establishes metadata from its opaque internal I/O. A response
whose first Write adds an implicit Content-Type is conservatively ambiguous too.
Debug omits its body preview rather than select a parser from mutable headers.
No early WriteHeader, extra Read, interface change or body rewriting is introduced.

`TestDebugResponseParserUsesCommittedMetadata`,
`TestDebugOpaqueReadFromOmitsAmbiguousMetadata` and
`TestDebugForwardedMetadataMutation` verify disclosure prevention and original
wire bytes. `TestCaptureResponseHeaders` covers commit paths, informational and
upgrade statuses, empty/committed ReadFrom and detached access. Concurrent
observation and implicit sniff controls exercise implementation-only boundaries.
Existing writer capability, accepted-byte, streaming and error tests remain gates.

Final results: LateHeader, EarlyRead and AfterCall exit 12 on NoDisclosure;
MetadataChecked exits 0 (26 generated, 20 distinct states). The forwarding action
requires an observable header change, matching the no-hidden-restore assumption.

```sh
for cfg in LateHeader EarlyRead AfterCall MetadataChecked; do
  model_tmp=$(mktemp -d /tmp/loom-response-metadata-XXXXXX)
  java -XX:+UseParallelGC -Djava.io.tmpdir="$model_tmp" \
    -cp "$TLA_TOOLS_JAR" tlc2.TLC -workers 1 \
    -metadir "$model_tmp/states" -config "$cfg.cfg" ResponseMetadata.tla
done
```
