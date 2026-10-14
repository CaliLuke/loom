# Reader registration identity

Readiness for #802, parent `af05fa8aa1b664e42d962ba7df617966a99bcbaf`.
Readiness was independently approved; production is implemented and awaits
independent final review.

## Contract and candidates

A batch's cursor advancement belongs to the registration admitted before XREAD,
not whichever registration has that stream key when Redis replies or fanout ends.
The model admits one batch of two ordered events, allows removal and up to two
replacement generations with a later start boundary, then interleaves the reply,
delivery, unsubscribe, Close and completion. A replacement may precede the reply
or occur during fanout. Unsubscribe releases fanout without changing ownership.

- `key-only.cfg` models current completion against the installed stream key.
- `reply-identity.cfg` captures identity at reply time, which is too late.
- `admission-identity.cfg` retains identity from admission through completion.
- `unchecked-close.cfg` retains identity but omits the closing check.

The first two must violate `NoStaleAdvance`; the third must pass all invariants;
the fourth must violate `NoPostCloseAdvance`. `ReplacementBoundary` protects the
later start cursor, and `DeliveryOrder` records unchanged order within the batch.

## Run

Use TLC 2.19 / Java 25, jar SHA256
`936a262061c914694dfd669a543be24573c45d5aa0ff20a8b96b23d01e050e88`:

```sh
java -XX:+UseParallelGC -cp "$HOME/.cache/loom-tools/tla2tools-1.7.4.jar" \
  tlc2.TLC -workers 1 -metadir /tmp/loom-reader-registration/checked \
  -config admission-identity.cfg ReaderRegistration.tla
```

Run the other configs by substituting their names and distinct temporary state
directories. Checker state and raw logs are temporary, not committed evidence.

## Implementation obligations and limitations

The implementation in `../../reader.go` replaces `Reader.streams`, `streamKeys`
and `streamCursors` with an ordered slice of `readerRegistration` objects,
each holding a Stream and cursor. Every successful add allocates a fresh
registration, including re-add of the same Stream pointer. `armRead` captures a
`readerRead` containing copied registration references and physical cursor
values under the Reader mutex. XREAD arguments derive from that snapshot;
`xread` associates each reply with the captured registration and returns
`readerBatch` values retaining those references rather than consulting current
membership. Unexpected, unrequested stream keys return an error.

`fanOut` completion advances only the same still-installed registration and
retains the existing Close guard. A request-local association is not a second
mutable registry. `slices.Clone` copies the slice before the mutex is released
so removal cannot shift or overwrite an in-flight admission snapshot.

`../../reader_registration_readiness_test.go` promotes the original committed
probe. `../../reader_registration_boundaries_test.go` tests held Redis replies
and backpressured fanout separately, with explicit/newest replacement, unchanged
registrations, unsubscribe, removal and Close. The native ProcessHook gate holds
a completed reply before the caller receives it. In every fanout case the test
receives and asserts the first event before mutating registration, unsubscribing
or closing, while the second event remains unbuffered and cannot finish. This
proves fanout has begun. The next actual command reply establishes completion
ordering.
No scheduling sleep establishes the race.

The model bounds one stream, one batch, three total generations and two events.
It does not prove network cancellation, Redis correctness, fair progress,
multistream key/result association, concurrent AddStream publication (#761),
or delivery cancellation on RemoveStream. Existing admitted delivery remains
permitted; removing a registration must not be advertised as draining pending
subscriber messages. Close and Unsubscribe retain their current channel behavior;
the model does not prove that a Go select cannot send an already-admitted event
while a close signal becomes ready. Tests cover concrete behavior outside the
model's abstraction. #800 owns stronger shutdown/operation guarantees.

`equivalent_handle_probe.go.txt` retains the original #773 reproduction without
any active read. The repaired key-based public membership lookup is covered by
`../../reader_membership_test.go`. The model already abstracts successful removal
of a selected registration and remains unchanged: public stream-key identity
must not replace registration-instance identity at completion.

Checked results: key-only fails after 92 distinct states; reply identity fails
after 141; admission identity passes 192 distinct / 384 generated; unchecked
Close fails after 35. Raw results are `/tmp/loom-reader-registration-802-*.log`.

The multistream regression removes the earlier registration, shifts the ordered
slice and twice re-adds that same Stream before releasing the admitted batch.
Its first removal now uses an equivalent Stream handle; the second uses the
original handle. It checks original event names/IDs/order, the replacement cursor and unaffected
stream advancement for both reply and fanout interleavings. All nine original
and extended failure cases now pass; six lifetime controls and #761's cursor
suite remain green. Full streaming/options race tests, selected pool callers and
focused Redis 8.10.2 pass. The four model configurations were rerun unchanged
(`/tmp/loom-reader-registration-802-final-*.log`); the roadmap records commands,
results and remaining scope. #773 now shares one public key lookup between add and remove; current focused
races and the unchanged admission-identity model verify this correction.
