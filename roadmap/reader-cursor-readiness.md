# Reader cursor readiness — #761

Status: isolated final review approved; main integration review pending. Not delivered.
Parent: `d57f96c6c393ab51c27a2d06c25298ca32e88af4`.

## Sequence

1. Preserve the #800 native finite-block source/socket evidence and clarify the shutdown prerequisites.
2. Read the Reader newest/start/add-stream contracts and reproduce the repeated `$` cursor loss before implementation, with synchronized single-stream polling, multistream fanout and added-stream cases.
3. Model the current repeated newest selection and compare one-time cursor snapshot/publication ordering, including failure and concurrent close.
4. Specify one shared cursor initialization owner for initial and added streams; retain explicit IDs and existing subscription/admission contracts.
5. Record bounded normal, edge, failure and real-Redis acceptance obligations; freeze the readiness diff for independent assessment. Production remains gated.

## Dependency

Resolve Reader newest cursors once (#761), then repair stale registration updates (#802), before introducing finite internal read polling (#800). Then prove the shared streaming operation/Close owner terminates its owned I/O and cleanup before implementing failed node acquisition rollback (#687). Worker publication (#704) remains separately owned.

## Contract and source evidence

`Stream.NewReader` selects the newest position by default; Reader options encode
that selection as `$`. `Reader.Subscribe` starts the goroutine lazily, but
`NewReader(ctx)` is the error-returning construction boundary. `Reader.AddStream`
is an error-returning registration boundary and copies the authored `startID`
unless an add-stream option overrides it. Its GoDoc mistakenly calls the reader
a sink; fix that documentation alongside the owning correction.

The coherent interpretation is to resolve newest once at successful construction
or AddStream, before publishing the cursor. Events appended after the successful
snapshot remain eligible, including events before first Subscribe or before the
next XREAD. Existing history at the snapshot is excluded. This makes the operation
with an error return own failed lookup handling. Lazy resolution in Subscribe
would add a failure with no error return and make start time dependent on the
first subscriber. No new option or compatibility mode is proposed.

The concrete risk is #761's lost event during fanout, also reproduced between
ordinary complete polls. Finite polling for #800 would expose this existing gap
on previously continuous BLOCK0 reads, so that repair depends on this one.
Explicit oldest, timestamp and event-ID positions must keep their existing
spelling and behavior. The default AddStream policy must remain the authored
reader start mode: a newest reader snapshots each newly added stream, while a
reader with an explicit ID reuses that ID. Do not substitute the main stream's
most recently consumed position or force all streams to share one Redis ID.

## Parent reproduction

Go 1.27.2, Darwin/arm64; root remains at the declared parent with no production
edits. `reader_cursor_readiness_test.go` gives **eight failing leaf cases and
three passing controls**, with no race report under both miniredis and real
Redis 8.10.2:

- Empty and populated newest constructors retain `$` instead of a concrete ID.
- An event inserted strictly between complete XREAD calls is lost for both
  initial and added newest streams; explicit oldest delivers it.
- With A1 filling the subscriber buffer and A2 held in fanout, B's new event is
  lost. The witness covers B registered before fanout and B added while A is
  backpressured. A ProcessHook signal after the second real command reply fixes
  the interleaving; no scheduling sleep is used.
- A newest lookup against a wrong-type Redis key cannot currently fail startup
  or prevent AddStream publication, because neither path resolves newest.
- Explicit `0` and `123-4` controls retain their cursors; duplicate registration
  is a no-op even if a fresh lookup would fail, and AddStream after Close rejects.

Commands:

```sh
go test -race ./pulse/streaming -run '^TestReaderCursorReadiness' -count=1
LOOM_PULSE_REDIS_ADDR=127.0.0.1:32768 go test -race ./pulse/streaming \
  -run '^TestReaderCursorReadiness' -count=1
```

The second command uses the existing disposable `loom-read-shutdown-probe`
Redis 8.10.2 container and **flushes its streaming test database**; do not use a
shared server. Recorded package times are 1.223s and 1.511s. Disposable JSON logs:
`/tmp/loom-reader-cursor-761-parent-race.json` and
`/tmp/loom-reader-cursor-761-real-race.json`. The tests are intentional red
readiness inputs, not a claim that the package suite passes.

## Shared owner proposal

Add one private cursor preparation helper used by newReader and AddStream. It
returns explicit positions unchanged. For `$`, retrieve the last retained event
ID using native `XREVRANGE key + - COUNT 1`; absent/empty streams become `0-0`.
A successful read captures the boundary once. Errors retain Redis causes and
stream identity; do not replace wrong-type, authentication or network errors
with `0-0`. No stream or consumer group is created by lookup.

NewReader prepares the cursor before allocating its cancelable runtime or
publishing a Reader. AddStream checks closing/duplicate registration under its
mutex, snapshots the selected authored mode, releases the mutex for lookup,
then reacquires it and rechecks closing/duplicate before committing the stream
and concrete cursor together. This preserves Close access while I/O is pending.
A duplicate winner cannot have its cursor reset by a slower concurrent AddStream.
Lookup failure does not publish, cancel an existing read or change other cursors.
A successful addition cancels the old active read only after it is attached.

Keep the current authored `startID` separate from per-stream resolved cursors.
Only a real delivered/read batch advances its stream's concrete position; no
retry, nil reply, unrelated-stream batch or option normalization re-resolves
newest. Existing explicit-ID arithmetic, timestamp behavior and stream retention
remain unchanged.

The helper may add one Redis command at newest construction/addition, so these
calls now expose connectivity/ACL/type errors at their actual initialization
boundary. There is no unbounded background startup retry. This change does not
establish an I/O cancellation guarantee: #800 remains the owner of shutdown and
bounded operations. Callers with explicit cursors do not acquire new lookup I/O.

## Model evidence and limitations

`pulse/streaming/tla/reader_cursor/ReaderCursor.tla` models independent stream
snapshot, publication, append and polling states, plus failed preparation and
Close. Checked using TLC2.19 / Java25:

| Mode | Result |
| --- | --- |
| legacy repeated newest | NoLostEvents fails; 95 distinct states before counterexample |
| snapshot without close recheck | NoPostClosePublication fails; 29 distinct states |
| checked snapshot/publication | 1,682 distinct states, 4,148 generated, no invariant failures |

The first experimental model had a precedence error in a Boolean assignment;
that run was discarded. Only `/tmp/loom-reader-cursor-model-v2-*.log` records
acceptance evidence. The model checks bounded safety, not network, fair delivery
or Go implementation correctness. Tests must cover the concrete snapshot command,
callbacks, registration races and emitted stream events.

`CursorRegistration.tla` separately models two concurrent AddStream preparations.
Without a duplicate recheck, a slower snapshot rewinds a cursor advanced by the
winner (173 distinct states before failure). Rechecking leaves the winner intact
and passes 358 distinct states (782 generated). Close and failed preparation are
also explored. These obligations are implemented by the shared lookup and repeated admission
check; the old serialized AddStream did not perform lookup I/O.

A distinct parent defect was reproduced by `stale_registration_probe.go.txt`,
now promoted to `pulse/streaming/reader_registration_readiness_test.go` for #802:
a real XREAD reply is held by an unbuffered subscriber while the stream is removed
and re-added at an explicit future ID. Receiving that old event lets fanOut rewind
the replacement to the old event ID. ProcessHook signals after the first and
second actual replies establish ordering; no sleep or direct fanOut invocation
is needed. This requires neither `$` nor an equivalent-but-different Stream
pointer (#773). It reproduces with race detection and remains separately scoped;
root independently reproduced and filed [#802](https://github.com/CaliLuke/loom/issues/802).
The original standalone input at parent `af05fa8aa` could be copied into
pulse/streaming or run with a Go overlay;
`/tmp/loom-reader-stale-registration-overlay.json` maps it
to the package, and `/tmp/loom-reader-stale-registration-parent.log` is the red
result.

Both problems belong to the Reader's per-stream registration/cursor owner. A
future correction should carry registration identity alongside a concrete cursor
from read admission through fanout completion, and apply advancement only if the
same registration remains installed. Do not add an independent cursor registry
or competing last-ID cache to solve the stale-reply defect. The #761 snapshot
helper and AddStream publication must leave a clear path to that same owner.
Root chose a separate atomic registration-identity repair after #761, filed as
[#802](https://github.com/CaliLuke/loom/issues/802). The sequence is #761 → #802 →
#800 → #687, with #704 separate. Only #761 production is authorized. The present bounded
models do not prove generation-safe delivery and the readiness approval must
not be presented as resolving this separate reproduced defect.

## Adoption and bounded production acceptance

The public Pulse guide constructs a default reader, so normal newest behavior is
a real supported path. In the current bounded consuming-code inventory,
`pulse/pool/node.go` and `worker.go`, plus loom-mcp registry recovery/catalog tests
at `f6aa6cfd24eb7c48ba22961125a67fe6d95f2c28`, explicitly use oldest. They must
retain no-new-lookup construction and identical delivery. This is not a full
consumer audit and authorizes no application-specific exception.

Before production review:

1. Turn the eight regressions green; retain all three controls and existing
   stream tests. Add snapshot-to-Subscribe and snapshot-to-first-read event
   witnesses, populated exclusion, multiple streams, repeated empty polls and
   explicitly selected positions. Assert event IDs and no duplicate delivery.
2. Add deterministic lookup error/cancellation, duplicate-winner, Close-during-
   lookup, and failed-add atomicity tests. Use the checked duplicate-publication model and verify its concrete race
   correspondence. Preserve the separately reproduced stale-reply follow-up.
3. Verify empty/missing/trimmed streams and wrong-type keys against miniredis and
   pinned real Redis. Preserve caller clients/options/hooks. Check the native
   lookup emits only the intended single read command for newest; explicit IDs
   and duplicate/closed paths need no Redis lookup.
4. Update internal nil-client test fixtures that previously constructed default
   readers without Redis to select explicit oldest where snapshot behavior is not
   under test. Never add a nil-client bypass to production cursor initialization.
5. Run the affected streaming package with race detection, its relevant pool
   caller controls, and selected real-Redis tests. No generator outputs or full
   framework corpus are affected. Public docs describe the one-time boundary and
   possible initialization error; framework guidance records the shared owner.
6. Freeze the final diff, exact source and evidence for a fresh independent
   reviewer. Readiness approval does not authorize a commit.

## Status

- [x] Parent failures and preserved controls, including real Redis.
- [x] Shared owner, consuming contracts and bounded candidate model.
- [x] Independent readiness assessment: `/root/review_reader_cursor_readiness`, exact 11-file manifest `7cfc9b349393046deb17f2d884d83dfe62faced8c88c21e3be801f3fa61578f1`, no findings.
- [x] Shared cursor implementation and bounded validation.
- [ ] Independent final review, integration and ticket closure (root-owned).


## Authorized implementation sequence

1. Implement the shared one-time newest lookup and atomic AddStream publication,
   keeping lookup outside the Reader mutex and rechecking duplicate/Close state.
2. Add deterministic snapshot/delivery, duplicate-winner, Close-during-lookup,
   cancellation and failed-add controls; retain parent red evidence.
3. Run direct and affected streaming/pool gates, race and pinned real Redis;
   align owning models/public documentation, then freeze for fresh final review.

#802's registration-identity probe remains separate evidence and is not installed
as a passing #761 regression. No #800/#687 production changes are authorized.

## Final implementation evidence

The production owner is `pulse/streaming/reader_cursor.go`; both initial and
added streams use it. AddStream snapshots through the reader's own Redis client,
matching its subsequent XREAD owner, and preserves caller hooks/options.
The authored start mode remains separate from each installed cursor. Redis
causes are wrapped with stream identity. Native lookup errors, canceled contexts
and a nil Redis client do not create a partial newest Reader. Two existing local
registration tests now explicitly select oldest because they intentionally use
no Redis client. No generated artifacts change.

The original eight parent failures and three controls all pass. New publication
and pre-Subscribe tests were added before implementation; their parent red log
is `/tmp/loom-reader-cursor-761-publication-red.log` (6.541s). Final checks use
Go 1.27.2 / Darwin arm64 and the worktree at the declared parent plus this diff:

```sh
go test -race ./pulse/streaming ./pulse/streaming/options -count=1
go test -race ./pulse/pool -run '^(TestDispatchJobLifecycle|TestClientOnlyNodeDispatches|TestNotifyWorker|TestNodeCloseRequeuesJobsToOtherNode|TestShutdownPropagatesAcrossNodes)$' -count=1
LOOM_PULSE_REDIS_ADDR=127.0.0.1:32768 go test -race ./pulse/streaming \
  -run 'TestReader(Cursor|Newest|AddStream)' -count=1
```

All pass without race reports: streaming 7.301s, options 1.357s, selected pool
5.251s, pinned real Redis streaming 2.383s. Logs are
`/tmp/loom-reader-cursor-761-packages.log`, `-pool.log`, and `-real-green.log`.
Pool selection covers node/worker readers, dispatch, notification, close and
cross-node shutdown while excluding separately scoped constructor-readiness
failures. The real tier uses the isolated Redis 8.10.2 probe and flushes only
its streaming database. No generated-code or exhaustive framework gates are
applicable to this runtime-only change.

All five model configurations were rerun with the pinned toolchain: three
expected counterexamples and two checked passes, retaining the recorded state
counts. Commands and assumptions remain beside the model. The separate #802
probe is retained as a known failure, not misrepresented as a passing test or
a defect fixed here. #800 I/O termination and #687 cleanup remain unimplemented.

Public GoDoc and `docs/pulse.md` document the snapshot boundary and initialization
errors. Root must integrate the matching consumer/framework skill and model-index
updates with this atomic ticket; this worktree intentionally does not edit shared
guides or master trackers. Temporary logs/checker state may be deleted after
independent review; durable model inputs and this concise evidence remain.

## Main integration evidence

Integrated on `25179e87b8b5d4e9cf5bb233d1818600469d060d` after independent
approval of the exact eighteen-file scope, manifest
`8a98484c5df4a1c1561564fe6002fe9a9c6d5c2de9c9b470d16b05f4a4c561e8`.
All production, test, model and public guide bytes match that candidate. No Pulse
source or module dependency changed between the isolated parent and this main
parent. Root added consumer/framework guidance and the model index, and reran
streaming/options under race detection: 6.860s / 1.273s, no race report; see
`/tmp/loom-reader-cursor-761-integrated.log`. The independent reviewer separately
passed the selected pool and Redis 8.10.2 checks, so their unchanged evidence is
reused. Final integrated exact-diff review is required before committing.
