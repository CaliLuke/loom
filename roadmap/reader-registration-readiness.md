# Reader registration identity readiness — #802

Status: implementation and bounded validation complete; frozen for independent final review. No commit before that review.
Parent: `af05fa8aa1b664e42d962ba7df617966a99bcbaf`.

## Sequence

1. Promote the committed stale-registration probe and reproduce #802 on the #761 parent.
2. Inspect stream identity, read admission, fanout completion, RemoveStream, Close and unsubscribe behavior. Reproduce #773 separately without bundling its semantic correction.
3. Model current key-only completion, a reply-time identity candidate, and admission-time identity through completion. Retain counterexamples and bounded checked evidence.
4. Specify the smallest shared registration/cursor owner and tests covering explicit/newest replacement, response/fanout races, unchanged registrations, delivery order, unsubscribe and Close.
5. Run bounded race tests on miniredis and isolated Redis 8.10.2. Freeze readiness for independent review; do not change production.

## Scope boundary

#761 is committed. #802 must prevent an admitted read from advancing a replacement registration. The same Stream pointer and explicit IDs reproduce it. #773 separately concerns matching an equivalent Stream by key in RemoveStream; preserve its current pointer-based removal during #802 unless the orchestrator explicitly accepts a different ticket scope. #800 shutdown and #687 constructor rollback remain downstream, not production work in this task.

## Findings and shared owner

`Reader.armRead` copies only key and cursor strings. `xread` returns unassociated
Redis batches. `fanOut` then searches the current keys and advances whichever
registration now uses the key. Thus removing and re-adding the same Stream
pointer can let an earlier admitted reply overwrite the replacement cursor.
The bug exists before and during fanout; checking identity only after the Redis
reply loses the read's original owner.

Use one private registration object holding `stream *Stream` and `cursor string`,
with object identity marking this specific installation. Replace the three
parallel Reader slices with one ordered slice of pointers to these objects;
do not add generation counters or a separately synchronized cursor cache.
Initial construction and every successful AddStream publication allocate one
object after #761's shared cursor preparation. Duplicate adds retain the existing
object and cursor. Failed lookups, Close rejection and duplicate-winner behavior
remain owned by the #761 admission checks.

Under the same Reader mutex, `armRead` captures a detached slice of registration
references and copies their cursor values into XREAD arguments. Associate each
returned Redis stream with the registration from this admission snapshot, never
with the live slice after the call. Carry the reference alongside its messages
through fanout, then advance only if that exact reference remains installed and
the Reader is not closing. The association is immutable request-local data;
there is still only one mutable cursor owner. An old batch may finish existing
subscriber delivery, but cannot change or resurrect a replacement registration.
The read loop remains serial, so no extra within-registration sequence counter
or cursor max-arithmetic is required. Preserve Redis result order and each
stream's event order. Keep all network I/O and subscriber sends outside the mutex.

RemoveStream keeps its current Stream-pointer selection for this atomic ticket.
The representation can retain `registration.stream == stream`, so #802 does not
require correcting #773. That report independently reproduces with no read at
all: addition deduplicates an equivalent handle, while removal leaves it
registered. Its separate probe and an overlay command are retained beside the
model. A future #773 change can select the same registration owner by key without
introducing another registry. No public API, options or client cloning is needed.

## Model results

`pulse/streaming/tla/reader_registration` holds the current/candidate models,
configurations and correspondence. With TLC 2.19 / Java 25 and the pinned jar:

- Key-only completion violates NoStaleAdvance (92 distinct states).
- Reply-time identity also violates NoStaleAdvance (141 distinct states).
- Admission identity passes 192 distinct states, 384 generated states.
- The unchecked-close negative control demonstrates why completion must retain
  the existing closing guard (35 distinct states before NoPostCloseAdvance fails).

Only bounded safety is claimed. The model covers one stream and one ordered
batch, not multistream association, fairness or I/O termination. Preserve #761's
independent cursor/publication models; they establish different prerequisites.

## Parent tests and scope

Go 1.27.2 / Darwin arm64. The promoted original probe fails under race detection
on the declared parent (`/tmp/loom-reader-registration-802-parent.log`, 0.639s).
The expanded matrix independently holds a native command reply or receives and
asserts the first event before mutation, leaving the second event unbuffered in
fanout. This receive proves fanout has started, including the Unsubscribe and
Close controls. In each stage explicit replacement, newest replacement and
replacement followed by Unsubscribe fail. Unchanged registrations preserve both
events in order; removal alone does not resurrect state; Close prevents cursor
advancement. These six controls pass. Together with the original test there are
**seven intentional failures and six passing controls** on miniredis and on
isolated Redis 8.10.2. All eleven #761 cursor leaf cases also pass on both tiers.

```sh
go test -race ./pulse/streaming \
  -run '^TestReader(StaleRegistrationReadiness|RegistrationBoundaries|CursorReadiness)' -count=1
LOOM_PULSE_REDIS_ADDR=127.0.0.1:32768 go test -race ./pulse/streaming \
  -run '^TestReader(StaleRegistrationReadiness|RegistrationBoundaries|CursorReadiness)' -count=1
```

The real tier uses disposable container `loom-read-shutdown-probe` and flushes
only its streaming test database. Do not target a shared server. JSON result
logs are `/tmp/loom-reader-registration-802-revised-miniredis.json` (0.838s) and
`/tmp/loom-reader-registration-802-revised-real.json` (2.416s); neither contains a
data-race report. These replace the original matrix evidence after independent
review found that the first ProcessHook signal alone did not establish fanout
had begun. Both revised runs have seven expected failures and seventeen passing
leaves (six lifetime controls and eleven #761 controls). The gate after the next actual reply establishes
that prior fanout completion ran. No timing sleep creates the interleaving.

#773's independent red input is `equivalent_handle_probe.go.txt`. An overlay maps
it to a package test without installing it as an active #802 regression:
`/tmp/loom-reader-registration-773-overlay.json`; output is
`/tmp/loom-reader-registration-773-parent.log`. This failure is not part of the
#802 green acceptance requirement and must not be counted as fixed by it.

## Bounded implementation acceptance

After independent readiness approval:

1. Implement the shared registration object and admission-to-completion identity,
   deleting superseded parallel slices. Update private seam tests to inspect the
   actual new owner, rather than adding compatibility fields/accessors for tests.
2. Turn all seven failures green and retain six controls. Keep exact original
   event IDs/order and explicit/newest replacement boundaries. Add a multistream
   regression where removing an earlier registration shifts slice indices while
   its batch remains pending; verify unrelated stream cursor advancement and
   request/result association. Cover repeated remove/re-add of the same pointer.
3. Preserve #761 newest snapshot, lookup count/error, duplicate-winner and
   Close-during-lookup controls. Run full streaming/options packages with race,
   five selected pool caller tests used for #761, and focused real Redis tests.
   Inspect helpers shared with Sink; do not change Sink fanout/ownership.
4. Update RemoveStream GoDoc/public Pulse guidance only to make admitted-batch
   behavior precise; do not promise immediate subscriber draining or stronger
   shutdown cancellation. Root updates shared framework guidance/model index.
5. Rerun affected model configurations, `git diff --check`, then freeze exact
   production diff plus evidence for fresh review. No generator artifacts change;
   no exhaustive framework corpus is appropriate. #773/#800/#687 remain separate.

## Status

- [x] Current parent repro and boundary/control matrix, miniredis and real Redis.
- [x] Current and candidate models, shared owner and explicit scope.
- [x] Independent readiness review: revised 10-file manifest `54cea2e692309478b28c42f016a699a41363994244d04f8a11069cd3e0ae5990` approved; medium fanout-gate finding fixed and re-reviewed. Root authorized implementation.
- [x] Shared registration owner implementation and bounded gates.
- [ ] Fresh final review, integration and ticket closure (root-owned).


## Implemented owner and final evidence

`Reader.registrations` is now the only mutable per-stream cursor owner.
`readerRegistration` holds Stream and cursor. `readerRead` clones registration
references and physical argument strings under the admission lock. `xread`
associates returned keys only with that captured snapshot and returns
`readerBatch` values; unrequested keys return an error. `fanOut` preserves delivery
then checks installed object identity and the closing guard before advancing.
Remove retains pointer selection (#773); Add retains key deduplication (#761).
No numeric cursor comparisons, independent registries or compatibility fields
were added. The obsolete unassociated fanOutStreams forwarding path was removed.
Existing seam tests now inspect registrations directly.

Two additional multistream/repeated-replacement cases failed first on the parent
(`/tmp/loom-reader-registration-802-multistream-red.log`, 0.761s). They exercise
both held replies and a first received event before mutation, remove the earlier
registration to shift indices, twice re-add the same pointer with later cursors,
and preserve the unaffected stream's advancement. Final assertions cover event
names as well as IDs and order. There are now nine original/extended parent
failure cases, all green, alongside six lifetime controls and the #761 suite.

Final checks on Go 1.27.2 / Darwin arm64, source at the declared parent plus this
diff:

```sh
go test -race ./pulse/streaming ./pulse/streaming/options -count=1
go test -race ./pulse/pool -run '^(TestDispatchJobLifecycle|TestClientOnlyNodeDispatches|TestNotifyWorker|TestNodeCloseRequeuesJobsToOtherNode|TestShutdownPropagatesAcrossNodes)$' -count=1
LOOM_PULSE_REDIS_ADDR=127.0.0.1:32768 go test -race ./pulse/streaming \
  -run '^TestReader(Registration|StaleRegistration|Cursor|Newest|AddStream)' -count=1
```

All pass without race reports: streaming 6.490s, options 1.318s, selected pool
4.807s, real Redis streaming 4.626s. Logs are
`/tmp/loom-reader-registration-802-packages.log`, `-pool.log` and `-real-green.log`.
`go vet ./pulse/streaming ./pulse/streaming/options` and `git diff --check`
also pass. The real tier uses the isolated Redis 8.10.2 probe. Full streaming tests include
Sink/shared subscriber controls; Sink production code is unchanged. Model source
is unchanged from readiness and all four configurations were rerun with the same
counts/expected failures. Earlier #761 model evidence remains valid, and its
concrete tests were rerun against the new owner. No generator paths change.

The #773 overlay still fails for the intended unchanged pointer mismatch
(`/tmp/loom-reader-registration-773-still-red.log`, 0.603s), not a compile error.
Its source was mechanically adapted to the new owner; no fix is claimed.
The original #802 standalone probe was promoted into the active regression test,
and the superseded copy removed with its README/roadmap references updated.
GoDoc and `docs/pulse.md` explain admitted delivery and replacement-cursor safety.
Root integrates shared skill/model-index guidance; this worktree does not edit
shared guides or master trackers. #800/#687 remain separate and unimplemented.
Temporary logs and checker states are retained until independent review finishes.

## Final review and integration

Independent final review approved the exact nineteen-file repair with no findings:
manifest `31627e1c67c8e229d4424cccf069754ab021d9046709c8456f0afb14d9d7d2f9`.
The reviewer independently passed streaming/options race tests (7.184s/1.182s),
focused Redis race tests (4.587s), vet, formatting and diff checks. Integration
onto `e0e40aaff` preserves every production/test/model byte from that approval,
adds shared consumer/framework/model-index guidance, and records this receipt.
Parent changes since `af05fa8aa` affect unrelated HTTP projection and docs; the
Pulse source/dependency graph is unchanged. This integration requires its own
scoped review before commit. #773 and #800 remain open.
