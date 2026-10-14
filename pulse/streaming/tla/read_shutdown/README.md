# Bounded operation and synchronous shutdown — #800

The operation/lifecycle owner was delivered in commit `d899a0112e02b8dc258b81851573c860fb375549`
with the exact published hook-preserving module fork.
Historical implementation parent `ebc06ec94` included #761/#802/#683.
Current refresh parent `7da1b00ee` also retains #765/#901/#768 and #771/#772/#773. Go 1.27.2,
go-redis stable v9.23.0 plus the exact reviewed b77660a fix, Redis 8.10.2.
The [dependency prerequisite](dependency_fetchability.md) records reproducibility.

## Owner and implementation correspondence

`pulse/internal/lifecycle.Lifetime` separates cancellation from cleanup. Atomic
stopping seals admission before once-only cancellation; every external Close joins
once-only cleanup. Stop itself never joins, so Reader's fatal read path cannot
wait for itself. Reader synchronizes with Subscribe's startup lock before Wait.
Sink joins its admitted AddStream/RemoveStream operation through the existing
state lock before workers and maps. Map joins receive/reconnect, cancels waiters
and closes subscribers under their admission lock. Reader's unpublished newest
lookup retains its earlier contract: Close does not wait for a resource-free
lookup, and publication rechecks stopping. There is no new registration cache.

`pulse/internal/redisop` is shared by streaming and rmap. Each operation borrows
a fresh WithTimeout facade, snapshots installed hooks and adds an operation
context deadline. Caller pools are never closed; Options are never mutated.
The ordinary ceiling is the existing five-second poll/cleanup budget, reduced
by positive caller ordinary read/write limits. A shared native Read plan pairs
the capped BLOCK with an operation context allowing poll plus ordinary response
budget, again reduced by stricter positive caller limits. Without that headroom,
a healthy five-second empty reply races a five-second context deadline. Caller
context deadlines are never extended; stricter caller settings may intentionally
expire before a normal empty reply. Context covers pool waits, cooperative dial,
handshake and retries. Ordinary socket reads/writes have finite options. Native
XREAD/XREADGROUP remain native, preserving driver dispatch and instrumentation;
zero/long polls are capped at five seconds and repeated. Negative nonblocking
options retain their meaning. Malformed-entry and filter-rejection automatic ACK are owned; external
Event.Ack and producer operations retain caller ownership.

With ContextTimeoutEnabled=false the driver's native blocking-read deadline is
BLOCK + 10 seconds, overriding even shorter caller/facade ReadTimeout. The context
deadline does not interrupt active socket I/O. With true, the deadline may shorten
that socket deadline, but cancellation still does not actively interrupt it.
No fixed five-second or subsecond Close SLA is promised. Finite admitted cleanup
can contain several separately bounded operations. A fresh budget applies to
WithoutCancel rollback. Arbitrary never-returning caller hooks/loggers/dialers
are outside the contract; cancellation cannot forcibly interrupt them.

rmap bounds initial/reconnect subscription setup with explicit ReceiveTimeout.
Shutdown closes the owned PubSub directly without an unsubscribe network round
trip. This joins the map's workers and releases local handles; it does not claim
to join every private driver goroutine. External Map operations are bounded
commands, not added background workers. Reset/Destroy after Close remain valid.
Map Subscribe retains its documented nil result after stopping.

## Parent counterexamples

Historical parent `81192e7a93cfafee03e081a4be45184b3458b831`:

- `real_redis_close_repro.go.txt`: CLIENT LIST proves native read admission;
  Reader/Sink × context false/true × BLOCK0/1h reproduces eight active-read hangs
  and eight second-Close early returns. Releasing one event joins both closers and
  caller PING succeeds. Race run intentionally fails2.479s.
- `constructor_cleanup_probe.go.txt`: after two maps and membership are acquired,
  group creation fails while actual Redis is frozen. Canceled setup cannot finish
  WithoutCancel rollback. Unfreeze joins and preserves the original error;
  intentional failure0.949s. The 200ms observation is not a promised Close SLA.
- `close_join_test.go` regressions additionally reproduce Reader/Sink and Map
  early return at implementation parent ebc06ec94. The Map parent overlay preserves
  candidate tests but substitutes original map.go/map_runtime.go; intentional red
  0.640s. Candidate all-owner race joins pass.

These are separate from pending #687 node rollback and already delivered #704
worker publication. Delivered #773 equivalent-handle lookup remains intact.

## Dependency

The required WithTimeout hook-copy defect is upstream issue4094 / PR4095.
`withtimeout_hooks_upstream.patch` is the exact revised two-file stable-parent
patch, SHA256 `e4f5dcf042935f2fa407210a62ac874a6c4f192bd77757ef3399224d85ab8d57`.
The initialized-hook guard preserves existing minimal baseClient clone tests.
`withtimeout_hooks_repro_test.go.txt` retains the original standalone negative
control, and `withtimeout_hooks_outgoing.md` the receiving-project-only draft.
These drafts are historical; the issue and PR have been published by root.

The adopted module is the stable v9.23.0 + b77660a correction with only
self-import/module-path changes, published under `github.com/CaliLuke/go-redis/v9`.
See [the dependency record](dependency_fetchability.md) for the exact pin, upstream
reference, migration and return-to-upstream condition. Final acceptance uses that
published module; historical receipts below used a disposable local modfile.

## Historical acceptance receipts

All candidate receipts use the reviewed stable-plus-fix local modfile. Raw logs
are temporary; the durable test/probe source and summarized results live here.

| Obligation | Evidence |
| --- | --- |
| All three owners join every Close | streaming and rmap `TestCloseJoinsEveryCaller`; parent red, candidate race green |
| Cancel before I/O mutex | `TestSinkCloseCancelsBeforeMaintenanceLock`; real keepalive command holds mutex while observing cancellation |
| Admitted acquisition owned | `TestSinkCloseJoinsAdmittedStreamAcquisition`; success and rollback, map subscription count zero, post-stop rejected |
| Subscriber/startup/reconnect | #683 tests; Map `TestCloseJoinsSubscriberAdmission`, `TestCloseDuringReconnectSnapshot`; replacement map cannot be abandoned |
| Actual runtime reads | `TestFiniteNativePollClose`, eight combinations with disabled socket limits; real race gate42.846s including lock/acquisition/join controls |
| Actual owner operation phases | `implemented_operation_probe.go.txt`; six race cases23.446s; native read timeout10.026s, retry/pool/dial100ms, real write117ms, early/late hooks |
| Constructor cleanup | `implemented_constructor_probe.go.txt`; frozen actual Redis, rollback finishes5.22s and preserves setup error |
| PubSub sockets | `rmap_socket_probe.go.txt`; established Close0.22ms; reconnect subscription reply joins4.95s under frozen Redis; authored200ms read limit joins91ms; final race gate7.185s |
| Broad consumers | streaming/rmap package races7.876s/2.334s; selected pool shutdown/ownership/event-loop races4.317s |
| Ordinary settings | `redisop.TestOwnedBudgetsPreserveCallerOptions`; disabled, short read/write, long and earlier context; `TestNativeBlockContract` |
| Delivery semantics | real native ack/noack, closed-owner pending recovery, stale registration and newest cursor controls; lost server reply retained PEL or noack absence; streaming5.095s/rmap1.987s |

The socket operation probe uses authored 100ms positive ordinary limits for fast
phase checks. Native runtime tests separately prove disabled limits get finite
polls. Native frozen read intentionally exceeds that 100ms ordinary limit.
PubSub ReceiveTimeout uses the same reduced ordinary read budget from the
borrowed facade; it does not reintroduce a five-second override of shorter limits.
The PubSub reconnect probe pauses Redis only after the new connection's handshake
has completed and immediately before writing SUBSCRIBE; its read must terminate
without receiving a reply. This is stronger than the earlier handshake-only case.

## Repeat bounded checks

From the repository root, use the exact published fork pinned in `go.mod`.
No local modfile or replacement is required. With Go 1.27.2:

```sh
GOEXPERIMENT=jsonv2 go test -race \
  ./pulse/internal/redisop ./pulse/rmap ./pulse/streaming -count=1
GOEXPERIMENT=jsonv2 LOOM_PULSE_REDIS_ADDR=127.0.0.1:32768 go test -p 1 -race \
  ./pulse/streaming ./pulse/rmap \
  -run '^(TestFiniteNativePollClose|TestHealthyOwnedEmptyPoll|TestOwnedEmptyPollRespectsCallerDeadline|TestClose|TestSinkClose|TestReaderCursor|TestReaderRegistration)' -count=1
```

For destructive socket probes, use a temporary Go overlay mapping each `.go.txt`
to a `_test.go` in its package: implemented_operation/constructor go into
pulse/streaming; rmap_socket goes into pulse/rmap. Run sequentially with `-p 1`,
`-run '^(TestOperationOwner|TestImplemented)'`, `GOEXPERIMENT=jsonv2` and the same
Redis address. These three current-source probes use the fork module. Earlier
prototype and original-bug inputs retain their historical upstream imports and
must run against the source revision described by their evidence.
These probes pause/unpause the named `loom-read-shutdown-probe` container. Use only
that isolated disposable server. Redistest flushes package DBs1–3; PubSub channels
span databases. Do not run another real-Redis suite concurrently.

## Model and limits

`ReadShutdown.tla` isolates active-read liveness: context-only and server-only
fail; socket-owner/bounded-I/O pass under guaranteed-return assumptions. Native
positive BLOCK actually installs a finite socket deadline, so corresponds to
bounded-I/O, not hypothetical server-only behavior.

`OperationShutdown.tla` includes four finite actors: reader, lock-held maintenance,
resource acquisition and replica. The replica represents rmap subscription setup,
receive/reconnect and handle cleanup; acquisition represents Sink AddStream map
ownership. Either acquisition or maintenance can hold the state lock. Every
actor traverses pool/dial/write/read/retry phases, with acquired handles retained
until cleanup. Stop seals admission outside that lock. Every requested closer
must wait for all admitted work and owned handles. Tests above establish these
implementation seams; existing subscriber/registration models retain their own
ordering semantics and are not replaced by this shutdown abstraction.

| Config | Result |
| --- | --- |
| operations-current | Expected liveness failure13: unbounded operations prevent stop/join |
| operations-read-only | Expected liveness failure13: only reader socket bound is insufficient |
| operations-early-close | Expected invariant failure12: second Close returns before completion |
| operations-command-only | Expected liveness failure13: successful conflicting CAS attempts can outlive every command budget |
| operations-owned | Pass0:270,786 generated /44,160 distinct states; all admission/ownership/join invariants |

Current/read-only set JoinEvery=true to isolate I/O liveness. Each actor admits
one finite operation; a phase advance means eventual return including failure.
Retry is finite exhaustion through a whole-operation deadline (`BoundRetry`),
independent of individual command bounds (`BoundOther`). With adverse conditions,
stuttering in the retry phase represents any number of successful commands whose
CAS loses; it does not imply a blocked native command. This abstraction checks
termination under the declared budget, not the Redis CAS algorithm itself. Weak fairness proves
eventual completion, not wall-clock time. Redis protocol, native timeout details,
caller callback cooperation and delivery guarantees are tested outside the model.
User-triggered map operations that acquire no local background resource are not
modeled as owned workers. Internal fatal Reader Stop is exercised by its direct
non-self-join test rather than an additional abstract closer.

TLC2.19 / Java25, cached tla2tools-1.7.4.jar SHA256
`936a262061c914694dfd669a543be24573c45d5aa0ff20a8b96b23d01e050e88`.
From this directory, run each configuration (temporary output only):

```sh
java -cp "$HOME/.cache/loom-tools/tla2tools-1.7.4.jar" tlc2.TLC -workers 2 \
  -metadir /tmp/loom-800-model/operations-owned \
  -config operations-owned.cfg OperationShutdown.tla
```

## Historical alternatives

`operation_owner_probe.go.txt` is the preimplementation prototype (23.160s
historical passing receipt), not the implemented owner. `bounded_read_probe.go.txt`
is an unselected pipeline experiment that did not preserve ProcessHook semantics.
`native_finite_block_probe.go.txt` independently proved BLOCK25ms installs a
10.027s read deadline. `block_zero_close_repro.go.txt` is the initial miniredis
witness. None is an alternative production path. No mutable caller options,
shared-pool Close, CLIENT UNBLOCK requirement or abandoned cleanup is introduced.

## Review correction: empty native poll response headroom

`TestHealthyOwnedEmptyPoll` reproduced the reviewed defect on both Reader/Sink
zero-block paths against the first implementation: real Redis race10.934s, both
returned an I/O timeout instead of redis.Nil. `redisop.Read` now returns both the
capped block and matching client context budget. The facade's ordinary socket
settings remain unchanged; only the native operation context receives response
headroom. Internal `blockDuration` is private so consumers use the paired plan.
The original socket/rollback/PubSub/caller-hook evidence remains valid outside
that native context-budget change.

Four context-enabled healthy empty Reader/Sink polls (zero/hour configured) and
all eight native shutdown configurations pass together on real Redis in72.612s.
`TestOwnedEmptyPollRespectsCallerDeadline` proves both positive caller ReadTimeout
and a parent context deadline still expire early (race1.800s).
`TestReadReservesResponseBudget` observes the actual inner command context for
zero, long, short, nonblocking and stricter-caller plans. It does not infer the
operation context from a Dialer, whose independently shorter DialTimeout applies.

`EmptyPoll.tla` makes the omitted protocol obligation explicit: a server finishes
its poll, then needs a response step. The old context budget expires before that
step (`empty-current.cfg`, expected invariant exit12). Reserved response headroom
passes (`empty-owned.cfg`,8states), and a stricter caller deadline still wins
(`empty-stricter.cfg`,4states), both exit0. This small model assumes a healthy
response arrives within the ordinary I/O allowance; it does not promise successful
replies over arbitrarily slow networks. Repeat using the same TLC invocation with
`-config empty-owned.cfg EmptyPoll.tla` and the other two configurations. The
four-owner OperationShutdown model retains the finite read obligation. Its current
refresh additionally separates whole-operation retry bounds from command bounds.

The implemented socket probe now obtains native clients and BLOCK arguments
through the same `redisop.Read` plan as both runtime consumers. Its two frozen
native socket cases pass again; ordinary-operation, PubSub, constructor rollback
and pool-consumer receipts are reused because those paths did not change.

## Current-owner refresh

`close_current_owners_test.go` links cancellation to current recovery PTTL and
CREATECONSUMER calls, including lock ordering and joining. The membership budget
control observes the context before per-command hooks; removing the outer context
reproduces a zero-deadline failure. Every CAS attempt shares that same context,
including detached cleanup. This complements the model's command-only counterexample.

Candidate active-read probes explicitly Subscribe after #901. The current Sink
filter record, registration selectors, rejected-event ACK and cleanup comparisons
remain owned by their delivered layers. These new command consumers borrow
`redisop.Client`; they do not acquire another lifecycle or alter filter admission.
The shared Context owner bounds whole retry loops as well as PubSub setup.
Current native race read/empty-poll/Close controls pass63.444s; focused current
consumer races pass streaming5.038s, rmap2.187s, redisop1.242s. TLC command-only
fails liveness (196,723 generated/31,160 distinct); fully owned passes
270,786 generated/44,160 distinct. All tests remain conditional on the exact local
stable-plus-fix dependency, never evidence of original-module fetchability.

The final CAS control forces two successful conflicting replies, then verifies all
attempts retain the first deadline. Candidate membership controls pass5.947s;
the command-only overlay fails both initial and retry deadline checks0.566s.
Native current membership/pool integration passes1.860s/9.952s; native operation,
constructor rollback and PubSub phases pass28.008s/7.279s. The unknown existing
group recheck was extracted intact into `checkSinkGroupIdentity` to keep the CAS
owner within the existing maintainability gate; admission semantics are unchanged.
