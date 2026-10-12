# Reader newest cursor preparation

Design and implementation correspondence for #761, committed in `af05fa8aa`.

`ReaderCursor.tla` explores two streams and up to two appended events per stream.
Preparation snapshots the newest ID once. Events after that snapshot are owed;
publication attaches the prepared cursor. Polling consumes concrete positions.
The legacy variant instead evaluates newest at every empty-history poll and can
advance beyond an owed event. The unchecked-publication variant demonstrates why
an AddStream lookup performed outside the mutex must recheck closing before
publication. A failed lookup is never published.

Run TLC 2.19 with Java 25 from this directory:

```sh
java -XX:+UseParallelGC -cp "$HOME/.cache/loom-tools/tla2tools-1.7.4.jar" \
  tlc2.TLC -workers 1 -metadir /tmp/loom-reader-cursor-model/checked \
  -config checked.cfg ReaderCursor.tla
```

Run all three configurations. `legacy` must fail NoLostEvents;
`unchecked-publication` must fail NoPostClosePublication; `checked` must pass.
The pinned jar SHA256 is
`936a262061c914694dfd669a543be24573c45d5aa0ff20a8b96b23d01e050e88`.

The cursor lookup and each poll are atomic abstraction steps. This model does
not prove Redis lookup correctness, network cancellation, driver behavior,
fanout ordering, history trimming/deletion/recreation, read replies from removed
registrations, or progress. Delivery obligations terminate when a reader is
explicitly closed; the invariant is about avoiding silent cursor advancement
while registered, not delivery after Close. Native explicit-ID and oldest modes
need executable controls; they do not evaluate newest and are not modeled here.

The Go readiness regressions in `../../reader_cursor_readiness_test.go` use
complete direct polls and a driver ProcessHook signal after the second actual
XREAD returns. They establish the event insertion occurs before a subsequent
poll without sleeps. The multistream case's buffer blocks fanout until the test
receives, preserving the issue's original interleaving. A deadline bounds the
negative observation and every Reader is synchronously closed afterward.

Checked results: legacy violates NoLostEvents after 95 distinct states;
unchecked publication violates NoPostClosePublication after 29; checked explores
1,682 distinct states (4,148 generated) without an invariant failure. Only the
corrected `/tmp/loom-reader-cursor-model-v2-*` runs are evidence. An initial
Boolean-assignment precedence error was corrected before these runs.

`CursorRegistration.tla` adds concurrent duplicate snapshots: run with
`unchecked-duplicate.cfg` (NoDuplicateRewind fails after 173 distinct states) and
`checked-duplicate.cfg` (358 distinct states, 782 generated, passes). A slower
lookup cannot reset the committed winner after that winner consumes events.
This model still does not include remove/re-add registration epochs.

The original stale-registration probe was promoted to
`../../reader_registration_readiness_test.go` for #802. Its parent failure uses
explicit IDs and real command replies. The separate
[registration identity model](../reader_registration/README.md) now covers
admission through fanout completion; this newest-cursor model still makes no
registration-epoch claim.

## Implementation correspondence

`prepareReaderCursor` in `../../reader_cursor.go` owns Prepare for both
`newReader` and `Reader.AddStream`: native XREVRANGE captures a retained newest
ID, or `0-0` for missing/empty streams. Explicit positions pass through without
lookup. The constructor performs this before allocating its runtime context.
AddStream performs I/O outside its mutex, then `canAddStream` repeats closing
and duplicate admission under the same mutex before cursor publication.
The existing read/fanout loop consumes and advances concrete cursors.

`../../reader_cursor_publication_test.go` fixes the corresponding concrete
interleavings by pausing after native lookup replies: Close remains available,
a slower duplicate cannot replace the winner, and failed preparation leaves
an active read untouched. It also checks pre-Subscribe event delivery,
missing/populated/trimmed streams, canceled lookup errors and lookup counts.
Together with the original readiness tests these pass under race detection
on miniredis and Redis 8.10.2. The full streaming package and selected pool
oldest-reader callers also pass. The roadmap records commands and result scope.

The final rerun retained the same state counts and expected negative controls
(`/tmp/loom-reader-cursor-final-*.log`). These models still do not establish
registration epochs (#802) or terminating I/O/Close (#800).
