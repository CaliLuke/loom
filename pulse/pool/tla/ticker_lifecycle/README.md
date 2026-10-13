# Ticker local lifecycle

`TickerLifecycle.tla` models the ticker-owned subscription, child context and
local timer, including work selected before local termination. It extends the
pool lifecycle model set with states absent from `PoolOwnership.tla` and
`ShutdownClose.tla`; those models describe job ownership and Node close, not
Ticker construction or timer rearming. No broad ownership-model run is needed
for these local resources.

## Decisions and results

Run with TLC 2.19 (`tla2tools-1.7.4.jar`, SHA256
`936a262061c914694dfd669a543be24573c45d5aa0ff20a8b96b23d01e050e88`)
and Java 25. Use separate temporary Java and checker directories per run:

```sh
java -Djava.io.tmpdir=/tmp/ticker-owned-java -cp /path/to/tla2tools-1.7.4.jar \
  tlc2.TLC -workers 1 -deadlock -metadir /tmp/ticker-owned-states \
  -config owned.cfg TickerLifecycle.tla
```

Create the Java directory first. `-deadlock` permits normal quiescent terminal
states. Expected invariant failures exit 12; successful bounded checks exit 0.
Raw traces and checker states remain outside the repository.

| Configuration | Result | Generated / distinct states |
| --- | --- | --- |
| current-abort | TerminalReleasesLocal fails: failed constructor retains resources | 2 / 2 |
| current-remote | NoParseOfAbsence fails: absent CAS result is parsed | 19 / 11 |
| current-close | TerminalReleasesLocal fails: selected work rearms after Close | 14 / 9 |
| constructor-only | Constructor cleanup alone leaves remote/selected-work failures | 19 / 12 |
| missing-only | Absence handling plus constructor cleanup still permits late rearm | 25 / 15 |
| tick-only | Guarding tick admission still allows a selected map update to rearm | 37 / 19 |
| abort-deletes-shared | LocalCleanupPreservesPeer fails: uncertain abort deletes peer state | 2 / 2 |
| local-owned | Constructor and explicit local/global close owner checks pass | 39 / 15 |
| owned | Full owner plus remote-absence handling checks pass | 76 / 21 |

The selected design centralizes local release and timer-rearm admission. It
never deletes shared state as constructor rollback. #738 can deliver the local
owner checked by `local-owned`; #707 then adds remote-absence/event-loop exit
consumers checked by `owned`. `local-owned` corresponds to the #738
implementation. The full `owned`
configuration remains design readiness for #707, not a claim that remote
observations have already been implemented.

## Implementation correspondence

- `Construct` / `Abort`: `Node.NewTicker` subscribes and creates a runtime child
  before SetAndWait. The constructor transfers those resources on success and
  must release them on any error, including a committed mutation with lost reply.
- `SelectTick`: the event loop selects a timer event before `handleTick` acquires
  its mutex. `Close` may acquire/release that mutex first.
- `SelectNotice` / `ApplyNotice`: the map branch reads Get before it acquires the
  ticker mutex and calls initTimer. All timer arms must honor terminal ownership.
- `Retry` / `Advance`: live Redis errors retry; successful compare-and-set keeps
  normal scheduling. A canceled call after Close must not rearm the timer.
- `RemoteStop` / `Missing` / `ObserveDeletion`: map replication may lag Redis
  deletion. TestAndSetEx's existence result and the later map absence are two
  observations of the same terminal contract. Absence is never a timestamp.
- `Close` / `GlobalStop`: local release preserves the entry; successful global
  deletion stops the shared ticker. Neither branch owns the borrowed rmap or
  caller's Redis client.

`pulse/pool/ticker_lifecycle_test.go` checks constructor cleanup, uncertain
write and late-rearm behavior; it retains direct healthy
local/global close and transient retry controls. The parent-cancellation counter
uses context's AfterFunc contract. The notification case tests the common rearm
seam after terminal release; code review checks its event-loop correspondence.
The pending missing-entry repro is preserved in the readiness packet and
`/tmp/loom-707-pending-repro_test.go` for its next atomic delivery. Existing
Tick/Schedule runtime tests remain consumer controls. Event-loop cleanup
must finish before its WaitGroup Done, and must never wait on itself.

## Assumptions and limits

One participant and one peer are sufficient for local ownership and shared-state
preservation. The initial shared entry represents either an existing peer entry
or an uncertain committed constructor write. Valid serialized values and positive
intervals are assumed. Redis outcomes are atomic and return under existing client
limits; this model does not prove network cancellation latency, deadlock freedom
inside the Redis client, clock scheduling, fairness or eventual tick delivery.
Normal map/context closure is another caller of local teardown; the detailed
rmap/runtime internals are not modeled. Retries can repeat indefinitely while
live. Global deletion failure remains an explicit operation outcome governed by
the existing Stop contract; it is not modeled as a successful global stop.
