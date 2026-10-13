# Replicated-map notification precedence (#771)

Status: implemented on delivered `ebf2c4ea9`; final independent review pending.

## Contract and owner

`Map.Subscribe` exposes a nonblocking, capacity-one invalidation notification,
not an event journal. Existing reconnect replay already folds kinds with
`strongerEventKind`: Reset dominates Delete, which dominates Change. The live
publisher drops any new kind when the channel is full, so an unread Change can
hide Reset. The scheduler's actual `handleStop` consumer relies on Reset to stop
globally. Correct the shared pending-notification owner rather than polling map
contents or patching the scheduler.

The map lock serializes publishers, Subscribe and Unsubscribe; the run goroutine
owns final channel closure after Close prevents subscription-list mutation.
Receivers race independently with publication. Coalescing must preserve the
strongest undelivered kind while allowing already-consumed events to leave the
pending set. Unsubscribe/Close must not race a send, and a slow subscriber must
not block map updates. A channel may still yield already-buffered data after
closure, as before; no publication occurs after its terminal ownership ends.

## Sequence and bounded acceptance

1. [x] Preserve direct Change/Delete/Reset permutations and ordinary controls;
   reproduce actual Redis-backed live delivery without sleeps as ordering claims.
2. [x] Gate the existing scheduler's Subscribe return after registration, queue
   an actual map Change then Reset, and show that handleStop misses the global
   stop. Reset-only passes; direct ordinary Change remains unchanged.
3. [x] Model the current drop path, reject last-value replacement, and check
   shared strongest-kind replacement across receiver interleavings and serialized
   unsubscribe/close. Reuse the existing priority helper and map/scheduler tests;
   this does not change #800 reader/sink shutdown or scheduler command issuance.
4. [x] Obtain independent readiness review before production.
5. [x] Implement only the shared notifier plus contract/tests/model correspondence;
   run focused race, live/reconnect, slow-consumer and scheduler checks. No new
   framework harness, generated design corpus or allocator.
6. [ ] Freeze final exact diff for independent review and root atomic delivery.

## Proposed implementation boundary

While holding the map lock, the sole publisher nonblockingly takes the buffered
kind if present, combines it using `strongerEventKind`, and republishes. A receiver
may win the take; then only the new kind remains pending. No other writer can fill
the emptied slot, so publishing stays bounded. The merged kind is local to each
subscriber; a stronger pending kind on one subscription must not contaminate
another subscription. Both live and reconnect paths
already call this owner. Do not duplicate priority logic in either path.

Consumer inventory: scheduler `handleStop` interprets Reset; pool node membership,
worker ownership/recovery, shutdown watch and ticker subscribers use notifications
to reread current state. Preserve those invalidation semantics and ordinary
coalescing. The existing best-effort test currently asserts the defective lost
Reset and must be replaced after the parent failure is recorded. Existing
reconnect and channel-close tests remain controls.

## Readiness evidence and final acceptance

The test-only packet is based on `ebf2c4ea91e239e0c1a59a8778e18ad7ae6147cc`.
With `GOEXPERIMENT=jsonv2` and the shared task Go cache, run:

```sh
go test -race ./pulse/rmap ./pulse/pool -run 'TestNotification(Pending|Consumed|Live)|TestSchedulerConsumesCoalescedReset' -count=1
```

The current publisher fails three of nine direct ordered pairs: Change→Delete,
Change→Reset and Delete→Reset. The other six pairs and consumed-kind reset control
pass. The actual live map test returns Change after applying Reset. The gated
scheduler misses global stop with pending Change, while its Reset-only control
passes. The subscriber registers before writes, and `SetAndWait` provides an
applied-revision barrier; no sleep establishes event ordering. Current race runs
finish in 0.702 s (map) and 1.732 s (pool), both expected failures.

The current model fails pending precedence after 26 generated/24 distinct states;
blind replacement fails after 59/42; the owned merge checks safety and eventual
delivery over 170/98. These are finite state checks, assuming serialized writers,
fair consumer progress and at most three publications. Redis delivery and actual
scheduler lifetime remain implementation-test obligations.

Final acceptance is bounded to this runtime owner:

- All direct pairs, consumed-event independence and per-subscriber independence.
- Slow subscriber stays nonblocking; concurrent receiving preserves the strongest
  unconsumed kind; subscription removal and existing Close semantics remain safe.
- Actual live Reset and the gated scheduler regression pass; ordinary change,
  Reset-only, existing reconnect resync and shutdown-watch controls remain valid.
- The two affected runtime packages pass their direct suites under the race
  detector, and changed Go files pass repository-configured lint.
- The model remains aligned with the sole publication owner; public Subscribe
  documentation and relevant maintainer/consumer guidance explain coalescing.

Readiness controls passed under the race detector: map 1.453 s and pool 1.456 s
for existing Close, reconnect resync, consumed-kind, Reset unsubscribe and durable
shutdown-watch checks. Repository-configured lint on both packages reports zero
issues. Exact logs and source hashes are retained in the readiness manifest.

There is no generated code or output contract change, so no generator corpus,
fixture regeneration, new scheduler protocol or shutdown model is needed.

## Implemented owner and verification

Independent readiness review approved the contract and bounded evidence without
findings. The sole implementation change drains each subscriber's unread slot,
merges through the existing priority helper and republishes. The final send is
always ready: a receiver cannot fill a slot and no other publisher can enter the
map lock. No scheduler production code or reconnect policy was changed. The
obsolete test expecting a lost Reset was replaced by the nine-pair regression.

Additional direct checks cover independent subscribers, repeated publication to
a slow subscriber, concurrent receiving of a pending Reset, and unsubscribe
serialized with publication. The original live map and gated scheduler failures
now pass. Focused race checks pass in 1.799 s (map) and 1.796 s (pool); configured
lint reports zero issues. Existing model evidence remains valid because the
implementation matches the previously checked owned transition exactly.

Full affected package race suites pass: `pulse/rmap` 2.164 s and `pulse/pool`
116.969 s. These are the complete hermetic package tests, including existing
map, ownership and scheduler controls. No Redis wire operation changed, so the
real-Redis protocol tier was not rerun for this in-process notification repair.
